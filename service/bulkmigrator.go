package service

import (
	"context"
	"fmt"
	"github.com/CharellKing/ela/config"
	es2 "github.com/CharellKing/ela/pkg/es"
	"github.com/CharellKing/ela/utils"
	"github.com/alitto/pond"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"regexp"
	"sync"
)

const defaultScrollSize = 1000
const defaultParallelism = 12
const defaultScrollTime = 10

type BulkMigrator struct {
	ctx context.Context

	SourceES es2.ES
	TargetES es2.ES

	Parallelism uint

	IndexPairMap map[string]*config.IndexPair

	Error error

	ScrollSize uint
}

func NewBulkMigratorWithES(ctx context.Context, sourceES, targetES es2.ES) *BulkMigrator {
	ctx = utils.SetCtxKeySourceESVersion(ctx, sourceES.GetClusterVersion())
	ctx = utils.SetCtxKeyTargetESVersion(ctx, targetES.GetClusterVersion())

	return &BulkMigrator{
		ctx:          ctx,
		SourceES:     sourceES,
		TargetES:     targetES,
		Parallelism:  defaultParallelism,
		IndexPairMap: make(map[string]*config.IndexPair),
		Error:        nil,
		ScrollSize:   defaultScrollSize,
	}
}

func NewBulkMigrator(ctx context.Context, srcConfig *config.ESConfig, dstConfig *config.ESConfig) (*BulkMigrator, error) {
	srcES, err := es2.NewESV0(srcConfig).GetES()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dstES, err := es2.NewESV0(dstConfig).GetES()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return NewBulkMigratorWithES(ctx, srcES, dstES), nil
}

func (m *BulkMigrator) GetCtx() context.Context {
	return m.ctx
}

func (m *BulkMigrator) getIndexPairKey(indexPair *config.IndexPair) string {
	return fmt.Sprintf("%s:%s", indexPair.SourceIndex, indexPair.TargetIndex)
}

func (m *BulkMigrator) WithIndexPairs(indexPairs ...*config.IndexPair) *BulkMigrator {
	if m.Error != nil {
		return m
	}

	newBulkMigrator := &BulkMigrator{
		ctx:          m.ctx,
		SourceES:     m.SourceES,
		TargetES:     m.TargetES,
		Parallelism:  m.Parallelism,
		IndexPairMap: m.IndexPairMap,
		Error:        m.Error,
		ScrollSize:   m.ScrollSize,
	}

	newIndexPairsMap := make(map[string]*config.IndexPair)
	for _, indexPair := range indexPairs {
		indexPairKey := m.getIndexPairKey(indexPair)
		if _, ok := newIndexPairsMap[indexPairKey]; !ok {
			newIndexPairsMap[indexPairKey] = indexPair
		}
	}

	if len(newIndexPairsMap) > 0 {
		newBulkMigrator.IndexPairMap = lo.Assign(newBulkMigrator.IndexPairMap, newIndexPairsMap)
	}
	return newBulkMigrator
}

func (m *BulkMigrator) WithScrollSize(scrollSize uint) *BulkMigrator {
	if m.Error != nil {
		return m
	}

	if scrollSize == 0 {
		scrollSize = defaultScrollSize
	}
	return &BulkMigrator{
		ctx:          m.ctx,
		SourceES:     m.SourceES,
		TargetES:     m.TargetES,
		Parallelism:  m.Parallelism,
		IndexPairMap: m.IndexPairMap,
		Error:        m.Error,
		ScrollSize:   scrollSize,
	}
}

func (m *BulkMigrator) filterIndexes(pattern string) ([]string, error) {
	indexes, err := m.SourceES.GetIndexes()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var filteredIndexes []string
	for _, index := range indexes {
		ok, err := regexp.Match(pattern, []byte(index))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if ok {
			filteredIndexes = append(filteredIndexes, index)
		}
	}
	return filteredIndexes, nil
}

func (m *BulkMigrator) WithPatternIndexes(pattern string) *BulkMigrator {
	if m.Error != nil {
		return m
	}

	newBulkMigrator := &BulkMigrator{
		SourceES:     m.SourceES,
		TargetES:     m.TargetES,
		Parallelism:  m.Parallelism,
		IndexPairMap: m.IndexPairMap,
		Error:        m.Error,
		ScrollSize:   m.ScrollSize,
	}

	var filterIndexes []string
	filterIndexes, newBulkMigrator.Error = m.filterIndexes(pattern)
	if newBulkMigrator.Error != nil {
		return newBulkMigrator
	}

	var newIndexPairsMap map[string]*config.IndexPair
	for _, index := range filterIndexes {
		indexPair := &config.IndexPair{
			SourceIndex: index,
			TargetIndex: index,
		}

		newIndexPairKey := m.getIndexPairKey(indexPair)
		if _, ok := newBulkMigrator.IndexPairMap[newIndexPairKey]; !ok {
			newIndexPairsMap[newIndexPairKey] = indexPair
		}
	}

	if len(newIndexPairsMap) > 0 {
		newBulkMigrator.IndexPairMap = lo.Assign(newBulkMigrator.IndexPairMap, newIndexPairsMap)
	}

	return newBulkMigrator
}

func (m *BulkMigrator) WithParallelism(parallelism uint) *BulkMigrator {
	if m.Error != nil {
		return m
	}

	if parallelism == 0 {
		parallelism = defaultParallelism
	}
	return &BulkMigrator{
		ctx:          m.ctx,
		SourceES:     m.SourceES,
		TargetES:     m.TargetES,
		Parallelism:  parallelism,
		IndexPairMap: m.IndexPairMap,
		Error:        m.Error,
		ScrollSize:   m.ScrollSize,
	}
}

func (m *BulkMigrator) Sync(force bool) error {
	if m.Error != nil {
		return errors.WithStack(m.Error)
	}

	m.parallelRun(func(migrator *Migrator) {
		if err := migrator.Sync(force); err != nil {
			utils.GetLogger(migrator.GetCtx()).WithError(err).Error("sync")
		}
	})
	return nil
}

func (m *BulkMigrator) SyncDiff(diffs *map[string][]utils.HashDiff) error {
	if m.Error != nil {
		return errors.WithStack(m.Error)
	}

	var diffMap sync.Map
	m.parallelRun(func(migrator *Migrator) {
		diffsItem, err := migrator.SyncDiff()
		if err != nil {
			utils.GetLogger(migrator.GetCtx()).WithError(err).Error("syncDiff")
		}
		diffMap.Store(m.getIndexPairKey(&migrator.IndexPair), diffsItem)
	})

	result := make(map[string][]utils.HashDiff)
	diffMap.Range(func(key, value interface{}) bool {
		keyStr := cast.ToString(key)
		diffsArrayList := value.([3][]utils.HashDiff)
		var diffArray []utils.HashDiff
		diffArray = append(diffArray, diffsArrayList[0]...)
		diffArray = append(diffArray, diffsArrayList[1]...)
		diffArray = append(diffArray, diffsArrayList[2]...)
		result[keyStr] = diffArray
		return true
	})

	diffs = &result
	return nil
}

func (m *BulkMigrator) Compare() (map[string][]utils.HashDiff, error) {
	if m.Error != nil {
		return nil, errors.WithStack(m.Error)
	}

	var diffMap sync.Map

	m.parallelRun(func(migrator *Migrator) {
		diffsItem, err := migrator.Compare()
		if err != nil {
			utils.GetLogger(migrator.GetCtx()).WithError(err).Error("compare")
		}

		if len(diffsItem[0]) > 0 || len(diffsItem[1]) > 0 || len(diffsItem[2]) > 0 {
			diffMap.Store(m.getIndexPairKey(&migrator.IndexPair), diffsItem)
		} else {
			utils.GetLogger(migrator.GetCtx()).Info("no difference")
		}
	})

	result := make(map[string][]utils.HashDiff)

	diffMap.Range(func(key, value interface{}) bool {
		keyStr := cast.ToString(key)
		diffsArrayList := value.([3][]utils.HashDiff)
		var diffArray []utils.HashDiff
		diffArray = append(diffArray, diffsArrayList[0]...)
		diffArray = append(diffArray, diffsArrayList[1]...)
		diffArray = append(diffArray, diffsArrayList[2]...)
		result[keyStr] = diffArray
		return true
	})

	return result, nil
}

func (m *BulkMigrator) CopyIndexSettings(force bool) error {
	if m.Error != nil {
		return errors.WithStack(m.Error)
	}

	m.parallelRun(func(migrator *Migrator) {
		if err := migrator.CopyIndexSettings(force); err != nil {
			utils.GetLogger(migrator.GetCtx()).WithError(err).Error("copyIndexSettings")
		}
	})
	return nil
}

func (m *BulkMigrator) parallelRun(callback func(migrator *Migrator)) {
	pool := pond.New(cast.ToInt(m.Parallelism), len(m.IndexPairMap))
	for _, indexPair := range m.IndexPairMap {
		newMigrator := &Migrator{
			ctx:        m.ctx,
			SourceES:   m.SourceES,
			TargetES:   m.TargetES,
			IndexPair:  *indexPair,
			ScrollSize: m.ScrollSize,
		}
		pool.Submit(func() {
			callback(newMigrator)
		})
	}
	pool.StopAndWait()
}
