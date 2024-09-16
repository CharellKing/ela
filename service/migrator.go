package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/CharellKing/ela/config"
	es2 "github.com/CharellKing/ela/pkg/es"
	"github.com/CharellKing/ela/utils"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	lop "github.com/samber/lo/parallel"
	"github.com/spf13/cast"
	"strings"
	"sync"
	"sync/atomic"
)

type Migrator struct {
	ctx context.Context

	SourceES es2.ES
	TargetES es2.ES

	IndexPair config.IndexPair

	ScrollTime uint

	SliceSize uint

	BufferCount uint

	WriteParallel uint

	WriteSize uint
}

func NewMigrator(ctx context.Context, srcConfig *config.ESConfig, dstConfig *config.ESConfig) (*Migrator, error) {
	srcES, err := es2.NewESV0(srcConfig).GetES()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dstES, err := es2.NewESV0(dstConfig).GetES()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ctx = utils.SetCtxKeySourceESVersion(ctx, srcES.GetClusterVersion())
	ctx = utils.SetCtxKeyTargetESVersion(ctx, dstES.GetClusterVersion())

	return &Migrator{
		ctx:           ctx,
		SourceES:      srcES,
		TargetES:      dstES,
		ScrollTime:    defaultScrollTime,
		SliceSize:     defaultSliceSize,
		BufferCount:   defaultBufferCount,
		WriteParallel: defaultWriteParallel,
		WriteSize:     defaultWriteSize,
	}, nil
}

func (m *Migrator) GetCtx() context.Context {
	return m.ctx
}

func (m *Migrator) WithIndexPair(indexPair config.IndexPair) *Migrator {
	ctx := utils.SetCtxKeySourceIndex(m.ctx, indexPair.SourceIndex)
	ctx = utils.SetCtxKeyTargetIndex(m.ctx, indexPair.TargetIndex)

	return &Migrator{
		ctx:           ctx,
		SourceES:      m.SourceES,
		TargetES:      m.TargetES,
		IndexPair:     indexPair,
		ScrollTime:    m.ScrollTime,
		SliceSize:     m.SliceSize,
		BufferCount:   m.BufferCount,
		WriteParallel: m.WriteParallel,
		WriteSize:     m.WriteSize,
	}
}

func (m *Migrator) WithScrollTime(scrollTime uint) *Migrator {
	return &Migrator{
		ctx:           m.ctx,
		SourceES:      m.SourceES,
		TargetES:      m.TargetES,
		IndexPair:     m.IndexPair,
		ScrollTime:    scrollTime,
		SliceSize:     m.SliceSize,
		BufferCount:   m.BufferCount,
		WriteParallel: m.WriteParallel,
		WriteSize:     m.WriteSize,
	}
}

func (m *Migrator) WithSliceSize(sliceSize uint) *Migrator {
	return &Migrator{
		ctx:           m.ctx,
		SourceES:      m.SourceES,
		TargetES:      m.TargetES,
		IndexPair:     m.IndexPair,
		ScrollTime:    m.ScrollTime,
		SliceSize:     sliceSize,
		BufferCount:   m.BufferCount,
		WriteParallel: m.WriteParallel,
		WriteSize:     m.WriteSize,
	}
}

func (m *Migrator) WithBufferCount(sliceSize uint) *Migrator {
	return &Migrator{
		ctx:           m.ctx,
		SourceES:      m.SourceES,
		TargetES:      m.TargetES,
		IndexPair:     m.IndexPair,
		ScrollTime:    m.ScrollTime,
		SliceSize:     m.SliceSize,
		BufferCount:   sliceSize,
		WriteParallel: m.WriteParallel,
		WriteSize:     m.WriteSize,
	}
}

func (m *Migrator) WithWriteSize(writeSize uint) *Migrator {
	return &Migrator{
		ctx:           m.ctx,
		SourceES:      m.SourceES,
		TargetES:      m.TargetES,
		IndexPair:     m.IndexPair,
		ScrollTime:    m.ScrollTime,
		SliceSize:     m.SliceSize,
		BufferCount:   m.BufferCount,
		WriteParallel: m.WriteParallel,
		WriteSize:     writeSize,
	}
}

func (m *Migrator) CopyIndexSettings(force bool) error {
	existed, err := m.TargetES.IndexExisted(m.IndexPair.TargetIndex)
	if err != nil {
		return errors.WithStack(err)
	}

	if existed && !force {
		return nil
	}

	if existed {
		if err := m.TargetES.DeleteIndex(m.IndexPair.TargetIndex); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := m.copyIndexSettings(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func getQueryMap(docIds []string) map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"terms": map[string]interface{}{
				"_id": docIds,
			},
		},
	}
}

func (m *Migrator) SyncDiff() (*DiffResult, error) {
	var diffResult DiffResult

	docCh, errsCh := m.compare()
	for {
		doc, ok := <-docCh
		if !ok {
			break
		}
		switch doc.Op {
		case es2.OperationSame:
			diffResult.SameCount += 1
		case es2.OperationCreate:
			diffResult.CreateCount += 1
			diffResult.CreateDocs = append(diffResult.CreateDocs, doc.ID)
		case es2.OperationUpdate:
			diffResult.UpdateCount += 1
			diffResult.UpdateDocs = append(diffResult.UpdateDocs, doc.ID)
		case es2.OperationDelete:
			diffResult.DeleteCount += 1
			diffResult.DeleteDocs = append(diffResult.DeleteDocs, doc.ID)
		}
	}

	errs := <-errsCh

	if len(diffResult.CreateDocs) > 0 {
		utils.GetLogger(m.ctx).Debugf("sync with create docs: %+v", len(diffResult.CreateDocs))
		if err := m.syncUpsert(getQueryMap(diffResult.CreateDocs), es2.OperationCreate); err != nil {
			errs.Add(errors.WithStack(err))
		}
	}

	if len(diffResult.UpdateDocs) > 0 {
		utils.GetLogger(m.ctx).Debugf("sync with update docs: %+v", len(diffResult.UpdateDocs))
		if err := m.syncUpsert(getQueryMap(diffResult.UpdateDocs), es2.OperationUpdate); err != nil {
			errs.Add(errors.WithStack(err))
		}
	}

	if len(diffResult.DeleteDocs) > 0 {
		utils.GetLogger(m.ctx).Debugf("sync with delete docs: %+v", len(diffResult.DeleteDocs))
		if err := m.syncUpsert(getQueryMap(diffResult.DeleteDocs), es2.OperationDelete); err != nil {
			errs.Add(errors.WithStack(err))
		}
	}
	return &diffResult, errs.Ret()
}

func (m *Migrator) getESIndexFields(es es2.ES) (map[string]interface{}, error) {
	esSettings, err := es.GetIndexMappingAndSetting(m.IndexPair.SourceIndex)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	propertiesMap := esSettings.GetProperties()

	return cast.ToStringMap(propertiesMap["properties"]), nil
}

func (m *Migrator) getKeywordFields() ([]string, error) {
	sourceEsFieldMap, err := m.getESIndexFields(m.SourceES)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	targetEsFieldMap, err := m.getESIndexFields(m.TargetES)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var keywordFields []string
	for fieldName, fieldAttrs := range sourceEsFieldMap {
		if _, ok := targetEsFieldMap[fieldName]; !ok {
			continue
		}

		fieldAttrMap := cast.ToStringMap(fieldAttrs)
		fieldType := cast.ToString(fieldAttrMap["type"])
		if fieldType == "keyword" {
			keywordFields = append(keywordFields, fieldName)
			continue
		}
	}

	return keywordFields, nil
}

func (m *Migrator) getDocHash(doc *es2.Doc) string {
	doc.Source = utils.SanitizeData(doc.Source).(map[string]interface{})

	jsonData, _ := json.Marshal(doc.Source)
	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:])
}

func (m *Migrator) handleMultipleErrors(errCh chan error) chan utils.Errs {
	errsCh := make(chan utils.Errs, 1)

	utils.GoRecovery(m.GetCtx(), func() {
		errs := utils.Errs{}
		for {
			err, ok := <-errCh
			if !ok {
				break
			}

			errs.Add(err)
		}
		errsCh <- errs
		close(errsCh)
	})
	return errsCh
}

func (m *Migrator) compare() (chan *es2.Doc, chan utils.Errs) {
	errCh := make(chan error)
	errsCh := m.handleMultipleErrors(errCh)

	keywordFields, err := m.getKeywordFields()
	if err != nil {
		errCh <- errors.WithStack(err)
		close(errCh)
		return nil, errsCh
	}

	sourceDocCh, sourceTotal := m.search(m.SourceES, m.IndexPair.SourceIndex, nil, keywordFields, errCh, true)

	targetDocCh, targetTotal := m.search(m.TargetES, m.IndexPair.TargetIndex, nil, keywordFields, errCh, true)

	var (
		sourceOk bool
		targetOk bool

		sourceCount      uint64
		targetCount      uint64
		sourceDocHashMap = make(map[string]string)
		targetDocHashMap = make(map[string]string)
	)

	compareDocCh := make(chan *es2.Doc, m.BufferCount)

	utils.GoRecovery(m.GetCtx(), func() {
		for {
			var (
				sourceResult *es2.Doc
				targetResult *es2.Doc
				op           es2.Operation
			)

			sourceResult, sourceOk = <-sourceDocCh
			targetResult, targetOk = <-targetDocCh

			if !sourceOk && !targetOk {
				close(errCh)
				break
			}

			if sourceResult != nil {
				sourceCount++
				sourceDocHashMap[sourceResult.ID] = sourceResult.Hash
			}

			if targetResult != nil {
				targetCount++
				targetDocHashMap[targetResult.ID] = targetResult.Hash
			}

			sourceProgress := cast.ToFloat32(sourceCount) / cast.ToFloat32(sourceTotal)
			targetProgress := cast.ToFloat32(targetCount) / cast.ToFloat32(targetTotal)
			utils.GetLogger(m.GetCtx()).Debugf("compare source progress %.4f, target progress %.4f",
				sourceProgress, targetProgress)

			if sourceResult != nil {
				targetHashValue, ok := targetDocHashMap[sourceResult.ID]
				if ok {
					if sourceDocHashMap[sourceResult.ID] != targetHashValue {
						op = es2.OperationUpdate
					} else {
						op = es2.OperationSame
					}

					compareDocCh <- &es2.Doc{
						ID: sourceResult.ID,
						Op: op,
					}

					delete(targetDocHashMap, sourceResult.ID)
					delete(sourceDocHashMap, sourceResult.ID)
				}
			}

			if targetResult != nil {
				sourceHashValue, ok := sourceDocHashMap[targetResult.ID]
				if ok {
					if targetDocHashMap[targetResult.ID] != sourceHashValue {
						op = es2.OperationUpdate
					} else {
						op = es2.OperationSame
					}

					compareDocCh <- &es2.Doc{
						ID: targetResult.ID,
						Op: op,
					}

					delete(targetDocHashMap, targetResult.ID)
					delete(sourceDocHashMap, targetResult.ID)
				}
			}
		}

		for id := range sourceDocHashMap {
			compareDocCh <- &es2.Doc{
				ID: id,
				Op: es2.OperationCreate,
			}
		}

		for id := range targetDocHashMap {
			compareDocCh <- &es2.Doc{
				ID: id,
				Op: es2.OperationDelete,
			}
		}

		close(compareDocCh)
	})

	return compareDocCh, errsCh
}

type DiffResult struct {
	SameCount   uint64
	CreateCount uint64
	UpdateCount uint64
	DeleteCount uint64

	CreateDocs []string
	UpdateDocs []string
	DeleteDocs []string
}

func (diffResult *DiffResult) HasDiff() bool {
	return diffResult.CreateCount > 0 || diffResult.UpdateCount > 0 || diffResult.DeleteCount > 0
}

func (diffResult *DiffResult) Total() uint64 {
	return diffResult.SameCount + diffResult.CreateCount + diffResult.UpdateCount + diffResult.DeleteCount
}

func (diffResult *DiffResult) Percent() float64 {
	return float64(diffResult.Total()-diffResult.SameCount) / float64(diffResult.Total())
}

func (m *Migrator) Compare() (*DiffResult, error) {
	docCh, errsCh := m.compare()
	var diffResult DiffResult
	for {
		doc, ok := <-docCh
		if !ok {
			break
		}
		switch doc.Op {
		case es2.OperationSame:
			diffResult.SameCount += 1
		case es2.OperationCreate:
			diffResult.CreateCount += 1
			diffResult.CreateDocs = append(diffResult.CreateDocs, doc.ID)
		case es2.OperationUpdate:
			diffResult.UpdateCount += 1
			diffResult.UpdateDocs = append(diffResult.UpdateDocs, doc.ID)
		case es2.OperationDelete:
			diffResult.DeleteCount += 1
			diffResult.DeleteDocs = append(diffResult.DeleteDocs, doc.ID)
		}
	}

	errs := <-errsCh
	return &diffResult, errs.Ret()
}

func (m *Migrator) Sync(force bool) error {
	utils.GetLogger(m.ctx).Debugf("sync with force: %+v", force)
	if err := m.CopyIndexSettings(force); err != nil {
		return errors.WithStack(err)
	}
	if err := m.syncUpsert(nil, es2.OperationCreate); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *Migrator) searchSingleSlice(wg *sync.WaitGroup, totalWg *sync.WaitGroup, total *atomic.Uint64, es es2.ES,
	index string, query map[string]interface{}, sortFields []string,
	sliceId *uint, sliceSize *uint, docCh chan *es2.Doc, errCh chan error, needHash bool) {

	utils.GoRecovery(m.GetCtx(), func() {
		var (
			scrollResult *es2.ScrollResult
			err          error
		)
		defer func() {
			wg.Done()
			if scrollResult != nil {
				if err := es.ClearScroll(scrollResult.ScrollId); err != nil {
					utils.GetLogger(m.GetCtx()).WithError(err).Error("clear scroll")
				}
			}
		}()

		func() {
			defer totalWg.Done()
			scrollResult, err = es.NewScroll(index, &es2.ScrollOption{
				Query:      query,
				SortFields: sortFields,
				ScrollSize: m.BufferCount,
				ScrollTime: m.ScrollTime,
				SliceId:    sliceId,
				SliceSize:  sliceSize,
			})

			if err != nil {
				utils.GetLogger(m.GetCtx()).Errorf("searchSingleSlice error: %+v", err)
				errCh <- errors.WithStack(err)
			}

			if scrollResult == nil {
				return
			}

			total.Add(cast.ToUint64(scrollResult.Total))
		}()

		for {
			if needHash {
				lop.Map(scrollResult.Docs, func(doc *es2.Doc, _ int) *es2.Doc {
					doc.Hash = m.getDocHash(doc)
					return doc
				})
			}
			for _, doc := range scrollResult.Docs {
				docCh <- doc
			}

			if len(scrollResult.Docs) < cast.ToInt(m.BufferCount) {
				break
			}
			if scrollResult, err = es.NextScroll(m.GetCtx(), scrollResult.ScrollId, m.ScrollTime); err != nil {
				utils.GetLogger(m.GetCtx()).Errorf("searchSingleSlice error: %+v", err)
				errCh <- errors.WithStack(err)
			}
		}
	})
}

func (m *Migrator) search(es es2.ES, index string, query map[string]interface{},
	sortFields []string, errCh chan error, needHash bool) (chan *es2.Doc, uint64) {
	docCh := make(chan *es2.Doc, m.BufferCount)
	var wg sync.WaitGroup
	var total atomic.Uint64
	var waitTotal sync.WaitGroup

	if m.SliceSize <= 1 {
		wg.Add(1)
		waitTotal.Add(1)
		m.searchSingleSlice(&wg, &waitTotal, &total, es, index, query, sortFields, nil, nil, docCh, errCh, needHash)
	} else {
		for i := uint(0); i < m.SliceSize; i++ {
			idx := i
			wg.Add(1)
			waitTotal.Add(1)
			m.searchSingleSlice(&wg, &waitTotal, &total, es, index, query, sortFields, &idx, &m.SliceSize, docCh, errCh, needHash)
		}
	}
	utils.GoRecovery(m.GetCtx(), func() {
		wg.Wait()
		close(docCh)
	})

	waitTotal.Wait()
	return docCh, total.Load()
}

func (m *Migrator) singleBulkWorker(doc <-chan *es2.Doc, total uint64, count *atomic.Uint64,
	operation es2.Operation, errCh chan error) {
	var buf bytes.Buffer
	for {
		v, ok := <-doc
		if !ok {
			break
		}

		count.Add(1)
		utils.GetLogger(m.GetCtx()).Debugf("bulk progress %.4f",
			cast.ToFloat32(count.Load())/cast.ToFloat32(total))

		switch operation {
		case es2.OperationCreate:
			if err := m.TargetES.BulkBody(m.IndexPair.TargetIndex, &buf, v); err != nil {
				errCh <- errors.WithStack(err)
			}
		case es2.OperationUpdate:
			if err := m.TargetES.BulkBody(m.IndexPair.TargetIndex, &buf, v); err != nil {
				errCh <- errors.WithStack(err)
			}
		case es2.OperationDelete:
			if err := m.TargetES.BulkBody(m.IndexPair.TargetIndex, &buf, v); err != nil {
				errCh <- errors.WithStack(err)
			}
		default:
			utils.GetLogger(m.ctx).Error("unknown operation")
		}

		if buf.Len() >= cast.ToInt(m.WriteSize)*1024*1024 {
			if err := m.TargetES.Bulk(&buf); err != nil {
				errCh <- errors.WithStack(err)
			}
		}
	}

	if buf.Len() > 0 {
		if err := m.TargetES.Bulk(&buf); err != nil {
			errCh <- errors.WithStack(err)
		}
	}
}

func (m *Migrator) bulkWorker(doc <-chan *es2.Doc, total uint64, operation es2.Operation, errCh chan error) {
	var wg sync.WaitGroup
	var count atomic.Uint64
	if m.WriteParallel <= 1 {
		m.singleBulkWorker(doc, total, &count, operation, errCh)
	}

	wg.Add(cast.ToInt(m.WriteParallel))
	for i := 0; i < cast.ToInt(m.WriteParallel); i++ {
		utils.GoRecovery(m.ctx, func() {
			defer wg.Done()
			m.singleBulkWorker(doc, total, &count, operation, errCh)
		})
	}
	wg.Wait()
}

func (m *Migrator) syncUpsert(query map[string]interface{}, operation es2.Operation) error {
	errCh := make(chan error)
	errsCh := m.handleMultipleErrors(errCh)

	var (
		docCh chan *es2.Doc
		total uint64
	)
	if operation == es2.OperationDelete {
		docCh, total = m.search(m.TargetES, m.IndexPair.SourceIndex, query, nil, errCh, false)
	} else {
		docCh, total = m.search(m.SourceES, m.IndexPair.SourceIndex, query, nil, errCh, false)
	}
	m.bulkWorker(docCh, total, operation, errCh)
	close(errCh)
	errs := <-errsCh
	return errs.Ret()
}

func (m *Migrator) getTargetSetting(sourceSetting map[string]interface{}) map[string]interface{} {
	var copySourceSetting map[string]interface{}
	_ = copier.Copy(&copySourceSetting, sourceSetting)

	return map[string]interface{}{
		m.IndexPair.TargetIndex: copySourceSetting[m.IndexPair.SourceIndex],
	}
}

func (m *Migrator) getTargetMapping(sourceMapping map[string]interface{}) map[string]interface{} {
	var copySourceMapping map[string]interface{}
	_ = copier.Copy(&copySourceMapping, sourceMapping)

	return map[string]interface{}{
		m.IndexPair.TargetIndex: copySourceMapping[m.IndexPair.SourceIndex],
	}
}

func (m *Migrator) copyIndexSettings() error {
	sourceESSetting, err := m.SourceES.GetIndexMappingAndSetting(m.IndexPair.SourceIndex)
	if err != nil {
		return errors.WithStack(err)
	}

	targetESSetting := m.GetTargetESSetting(sourceESSetting)

	if err := m.TargetES.CreateIndex(targetESSetting); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (m *Migrator) GetTargetESSetting(sourceESSetting es2.IESSettings) es2.IESSettings {
	if strings.HasPrefix(m.TargetES.GetClusterVersion(), "8.") {
		return sourceESSetting.ToTargetV8Settings(m.IndexPair.TargetIndex)
	} else if strings.HasPrefix(m.TargetES.GetClusterVersion(), "7.") {
		return sourceESSetting.ToTargetV7Settings(m.IndexPair.TargetIndex)
	} else if strings.HasPrefix(m.TargetES.GetClusterVersion(), "6.") {
		return sourceESSetting.ToTargetV6Settings(m.IndexPair.TargetIndex)
	} else if strings.HasPrefix(m.TargetES.GetClusterVersion(), "5.") {
		return sourceESSetting.ToTargetV5Settings(m.IndexPair.TargetIndex)
	}

	return nil
}
