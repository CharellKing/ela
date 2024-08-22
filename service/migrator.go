package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/CharellKing/ela/config"
	es2 "github.com/CharellKing/ela/pkg/es"
	"github.com/CharellKing/ela/utils"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"strings"
)

type Migrator struct {
	ctx context.Context

	SourceES es2.ES
	TargetES es2.ES

	IndexPair config.IndexPair

	ScrollSize uint
	ScrollTime uint
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
		ctx:        ctx,
		SourceES:   srcES,
		TargetES:   dstES,
		ScrollSize: defaultScrollSize,
		ScrollTime: defaultScrollTime,
	}, nil
}

func (m *Migrator) GetCtx() context.Context {
	return m.ctx
}

func (m *Migrator) WithIndexPair(indexPair config.IndexPair) *Migrator {
	ctx := utils.SetCtxKeySourceIndex(m.ctx, indexPair.SourceIndex)
	ctx = utils.SetCtxKeyTargetIndex(m.ctx, indexPair.TargetIndex)

	return &Migrator{
		ctx:        ctx,
		SourceES:   m.SourceES,
		TargetES:   m.TargetES,
		IndexPair:  indexPair,
		ScrollSize: m.ScrollSize,
		ScrollTime: m.ScrollTime,
	}
}

func (m *Migrator) WithScrollSize(scrollSize uint) *Migrator {
	return &Migrator{
		ctx:        m.ctx,
		SourceES:   m.SourceES,
		TargetES:   m.TargetES,
		IndexPair:  m.IndexPair,
		ScrollSize: scrollSize,
		ScrollTime: m.ScrollTime,
	}
}

func (m *Migrator) WithScrollTime(scrollTime uint) *Migrator {
	return &Migrator{
		ctx:        m.ctx,
		SourceES:   m.SourceES,
		TargetES:   m.TargetES,
		IndexPair:  m.IndexPair,
		ScrollSize: m.ScrollSize,
		ScrollTime: scrollTime,
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

	if existed && force {
		if err := m.TargetES.DeleteIndex(m.IndexPair.TargetIndex); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := m.copyIndexSettings(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *Migrator) ConvertHashDiffToDocs(diffs []utils.HashDiff) []es2.Doc {
	var docs []es2.Doc
	for _, diff := range diffs {
		docs = append(docs, es2.Doc{
			ID:   diff.Id,
			Type: diff.Type,
		})
	}
	return docs
}

func (m *Migrator) SyncDiff() ([3][]utils.HashDiff, error) {
	diffs, err := m.Compare()
	if err != nil {
		return diffs, errors.WithStack(err)
	}

	if len(diffs[0]) > 0 {
		ids := lo.Map(diffs[0], func(v utils.HashDiff, _ int) string {
			return cast.ToString(v.Id)
		})
		queryMap := map[string]interface{}{
			"query": map[string]interface{}{
				"terms": map[string]interface{}{
					"_id": ids,
				},
			},
		}

		if err := m.syncInsert(queryMap); err != nil {
			return diffs, errors.WithStack(err)
		}
	}

	if len(diffs[1]) > 0 {
		hitDocs := m.ConvertHashDiffToDocs(diffs[1])
		if err := m.syncDelete(hitDocs); err != nil {
			return diffs, errors.WithStack(err)
		}
	}

	if len(diffs[2]) > 0 {
		ids := lo.Map(diffs[2], func(v utils.HashDiff, _ int) string {
			return cast.ToString(v.Id)
		})
		queryMap := map[string]interface{}{
			"query": map[string]interface{}{
				"terms": map[string]interface{}{
					"_id": ids,
				},
			},
		}

		if err := m.syncUpdate(queryMap); err != nil {
			return diffs, errors.WithStack(err)
		}
	}
	return diffs, nil
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

func (m *Migrator) getDocHashMap(result *es2.ScrollResultYield, keywordFields []string) (map[string]*utils.DocHash, []string) {
	var lastKeywordFieldValues []string
	docHashMap := make(map[string]*utils.DocHash)
	for idx, doc := range result.Docs {
		jsonData, _ := json.Marshal(doc.Source)
		hash := md5.Sum(jsonData)
		docHashMap[doc.ID] = &utils.DocHash{
			ID:   doc.ID,
			Type: doc.Type,
			Hash: hex.EncodeToString(hash[:]),
		}

		if idx == len(result.Docs)-1 {
			for _, field := range keywordFields {
				lastKeywordFieldValues = append(lastKeywordFieldValues, cast.ToString(doc.Source[field]))
			}
		}
	}
	return docHashMap, lastKeywordFieldValues
}

func (m *Migrator) compareSortableFieldValues(lastSourceSortFieldValues []string, lastTargetSortFieldValues []string) int {
	for idx, sourceValue := range lastSourceSortFieldValues {
		ret := strings.Compare(sourceValue, lastTargetSortFieldValues[idx])
		if ret != 0 {
			return ret
		}
	}
	return 0
}

func (m *Migrator) compare(keywordFields []string) ([3][]utils.HashDiff, uint64) {
	sourceCh := lo.Generator(1, func(yield func(*es2.ScrollResultYield)) {
		if err := m.SourceES.SearchByScroll(m.GetCtx(), m.IndexPair.SourceIndex, nil, keywordFields, m.ScrollSize, m.ScrollTime, yield); err != nil {
			utils.GetLogger(m.ctx).WithError(err).Error("search scroll")
		}
	})

	targetCh := lo.Generator(1, func(yield func(*es2.ScrollResultYield)) {
		if err := m.TargetES.SearchByScroll(m.GetCtx(), m.IndexPair.TargetIndex, nil, keywordFields, m.ScrollSize, m.ScrollTime, yield); err != nil {
			utils.GetLogger(m.ctx).WithError(err).Error("search scroll")
		}
	})

	var (
		sourceOk bool
		targetOk bool

		lastSourceSortFieldValues []string
		lastTargetSortFieldValues []string

		sourceDocHashMap = make(map[string]*utils.DocHash)
		targetDocHashMap = make(map[string]*utils.DocHash)

		diffs [3][]utils.HashDiff

		sameCount   uint64
		sourceCount uint64
		targetCount uint64
		sourceTotal uint64
		targetTotal uint64
	)

	for {
		var (
			sourceResult *es2.ScrollResultYield
			targetResult *es2.ScrollResultYield
		)

		compareRet := m.compareSortableFieldValues(lastSourceSortFieldValues, lastTargetSortFieldValues)
		if compareRet < 0 {
			sourceResult, sourceOk = <-sourceCh
		} else if compareRet > 0 {
			targetResult, targetOk = <-targetCh
		} else {
			sourceResult, sourceOk = <-sourceCh
			targetResult, targetOk = <-targetCh
		}

		if !sourceOk && !targetOk {
			break
		}

		var subSourceDocHashMap map[string]*utils.DocHash
		if sourceResult != nil && len(sourceResult.Docs) > 0 {
			subSourceDocHashMap, lastSourceSortFieldValues = m.getDocHashMap(sourceResult, keywordFields)
			sourceCount += cast.ToUint64(len(sourceResult.Docs))
			sourceTotal = sourceResult.Total
		}

		var subTargetDocHashMap map[string]*utils.DocHash
		if targetResult != nil && len(targetResult.Docs) > 0 {
			subTargetDocHashMap, lastTargetSortFieldValues = m.getDocHashMap(targetResult, keywordFields)
			targetCount += cast.ToUint64(len(targetResult.Docs))
			targetTotal = targetResult.Total
		}

		for id, docHash := range subSourceDocHashMap {
			if _, ok := targetDocHashMap[id]; !ok {
				if _, ok = subTargetDocHashMap[id]; !ok {
					sourceDocHashMap[id] = docHash
				} else {
					if docHash.Hash != subTargetDocHashMap[id].Hash {
						diffs[2] = append(diffs[2], utils.HashDiff{
							Action:          utils.ActionTypeModify,
							Id:              id,
							Type:            docHash.Type,
							SourceHashValue: docHash.Hash,
							TargetHashValue: subTargetDocHashMap[id].Hash,
						})
					} else {
						sameCount++
					}
					delete(subTargetDocHashMap, id)
				}
			} else {
				if docHash.Hash != targetDocHashMap[id].Hash {
					diffs[2] = append(diffs[2], utils.HashDiff{
						Action:          utils.ActionTypeModify,
						Id:              id,
						Type:            docHash.Type,
						SourceHashValue: docHash.Hash,
						TargetHashValue: targetDocHashMap[id].Hash,
					})
				} else {
					sameCount++
				}
				delete(targetDocHashMap, id)
			}
		}

		for id, docHash := range subTargetDocHashMap {
			if _, ok := sourceDocHashMap[id]; !ok {
				targetDocHashMap[id] = docHash
			} else {
				if docHash.Hash != sourceDocHashMap[id].Hash {
					diffs[2] = append(diffs[2], utils.HashDiff{
						Action:          utils.ActionTypeModify,
						Id:              id,
						Type:            docHash.Type,
						SourceHashValue: sourceDocHashMap[id].Hash,
						TargetHashValue: docHash.Hash,
					})
				} else {
					sameCount++
				}
				delete(sourceDocHashMap, id)
			}
		}

		utils.GetLogger(m.ctx).Infof("source count(%d), target count(%d), source total(%d), target total(%d), "+
			"source progress(%d%%), target progress(%d%%), same count(%d)",
			sourceCount, targetCount, sourceTotal, targetTotal,
			sourceCount*100.0/sourceTotal, targetCount*100.0/targetTotal, sameCount)
	}
	return diffs, sameCount
}

func (m *Migrator) Compare() ([3][]utils.HashDiff, error) {
	keywordFields, err := m.getKeywordFields()
	if err != nil {
		return [3][]utils.HashDiff{}, errors.WithStack(err)
	}

	diffs, sameCount := m.compare(keywordFields)

	total := cast.ToUint64(len(diffs[0])+len(diffs[1])+len(diffs[2])) + sameCount
	utils.GetLogger(m.ctx).Infof("compare total (%d), add(%d), delete(%d), modified(%d), same(%d)",
		total, len(diffs[0]), len(diffs[1]), len(diffs[2]), sameCount)
	return diffs, nil
}

func (m *Migrator) Sync(force bool) error {
	if err := m.CopyIndexSettings(force); err != nil {
		return errors.WithStack(err)
	}
	return m.syncInsert(nil)
}

func (m *Migrator) syncInsert(query map[string]interface{}) error {
	for v := range lo.Generator(1, func(yield func(*es2.ScrollResultYield)) {
		if err := m.SourceES.SearchByScroll(m.GetCtx(), m.IndexPair.SourceIndex, query, nil, m.ScrollSize, m.ScrollTime, yield); err != nil {
			utils.GetLogger(m.ctx).WithError(err).Error("search scroll")
		}
	}) {
		if len(v.Docs) > 0 {
			if err := m.TargetES.BulkInsert(m.IndexPair.TargetIndex, v.Docs); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

func (m *Migrator) syncUpdate(query map[string]interface{}) error {
	for v := range lo.Generator(1, func(yield func(*es2.ScrollResultYield)) {
		if err := m.SourceES.SearchByScroll(m.GetCtx(), m.IndexPair.SourceIndex, query, nil, m.ScrollSize, m.ScrollTime, yield); err != nil {
			utils.GetLogger(m.GetCtx()).WithError(err).Error("search by scroll")
		}
	}) {
		if len(v.Docs) > 0 {
			if err := m.TargetES.BulkUpdate(m.IndexPair.TargetIndex, v.Docs); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

func (m *Migrator) syncDelete(hitDocs []es2.Doc) error {
	if err := m.TargetES.BulkDelete(m.IndexPair.TargetIndex, hitDocs); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *Migrator) getDocsHashValues(esInstance es2.ES, index string) (map[string]*utils.DocHash, error) {
	docHashMap := make(map[string]*utils.DocHash)
	for v := range lo.Generator(1, func(yield func(*es2.ScrollResultYield)) {
		if err := esInstance.SearchByScroll(m.GetCtx(), index, nil, nil, m.ScrollSize, m.ScrollTime, yield); err != nil {
			utils.GetLogger(m.ctx).WithError(err).Error("search by scroll")
		}
	}) {
		if len(v.Docs) > 0 {
			for _, doc := range v.Docs {
				jsonData, _ := json.Marshal(doc.Source)
				hash := md5.Sum(jsonData)
				docHashMap[doc.ID] = &utils.DocHash{
					ID:   doc.ID,
					Type: doc.Type,
					Hash: hex.EncodeToString(hash[:]),
				}
			}
		}
	}
	return docHashMap, nil
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
