package service

import (
	"context"
	"fmt"
	"github.com/CharellKing/ela/config"
	"github.com/CharellKing/ela/pkg/es"
	"github.com/CharellKing/ela/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"strings"
)

type Task struct {
	bulkMigrator *BulkMigrator
	force        bool
}

func NewTaskWithES(ctx context.Context, taskCfg *config.TaskCfg, sourceES, targetES es.ES) *Task {
	taskId := uuid.New().String()
	ctx = utils.SetCtxKeySourceESVersion(ctx, sourceES.GetClusterVersion())
	ctx = utils.SetCtxKeyTargetESVersion(ctx, targetES.GetClusterVersion())
	ctx = utils.SetCtxKeyTaskName(ctx, taskCfg.Name)
	ctx = utils.SetCtxKeyTaskID(ctx, taskId)
	ctx = utils.SetCtxKeyTaskAction(ctx, string(taskCfg.TaskAction))

	bulkMigrator := NewBulkMigratorWithES(ctx, sourceES, targetES)
	bulkMigrator = bulkMigrator.WithScrollSize(taskCfg.ScrollSize).WithIndexPairs(taskCfg.IndexPairs...).
		WithParallelism(taskCfg.Parallelism).WithScrollTime(taskCfg.ScrollTime).WithSliceSize(taskCfg.SliceSize).
		WithBufferCount(taskCfg.BufferCount).WithWriteParallel(taskCfg.WriteParallelism)
	if taskCfg.IndexPattern != nil {
		bulkMigrator = bulkMigrator.WithPatternIndexes(*taskCfg.IndexPattern)
	}

	return &Task{
		bulkMigrator: bulkMigrator,
		force:        taskCfg.Force,
	}

}

func NewTask(ctx context.Context, taskCfg *config.TaskCfg, cfg *config.Config) (*Task, error) {
	if cfg == nil {
		return nil, nil

	}

	sourceESV0 := es.NewESV0(cfg.ESConfigs[taskCfg.SourceES])
	sourceES, err := sourceESV0.GetES()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	targetESV0 := es.NewESV0(cfg.ESConfigs[taskCfg.TargetES])
	targetES, err := targetESV0.GetES()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return NewTaskWithES(ctx, taskCfg, sourceES, targetES), nil
}

func (t *Task) GetCtx() context.Context {
	return t.bulkMigrator.GetCtx()
}

func (t *Task) Compare() (map[string]*DiffResult, error) {
	return t.bulkMigrator.Compare()
}

func (t *Task) SyncDiff() (map[string]*DiffResult, error) {
	return t.bulkMigrator.SyncDiff()
}

func (t *Task) Sync() error {
	return t.bulkMigrator.Sync(t.force)
}

func (t *Task) CopyIndexSettings() error {
	return t.bulkMigrator.CopyIndexSettings(t.force)
}

func (t *Task) Run() error {
	ctx := t.GetCtx()
	taskAction := config.TaskAction(utils.GetCtxKeyTaskAction(ctx))
	switch taskAction {
	case config.TaskActionCopyIndex:
		return t.CopyIndexSettings()
	case config.TaskActionSync:
		return t.Sync()
	case config.TaskActionSyncDiff:
		diffResultMap, err := t.SyncDiff()
		if err != nil {
			return errors.WithStack(err)
		}

		for indexes, diffResult := range diffResultMap {
			indexArray := strings.Split(indexes, ":")
			utils.GetLogger(t.GetCtx()).
				WithField("sourceIndex", indexArray[0]).
				WithField("targetIndex", indexArray[1]).
				WithField("percent", diffResult.Percent()).
				WithField("create", diffResult.CreateCount).
				WithField("update", diffResult.UpdateCount).
				WithField("delete", diffResult.DeleteCount).
				WithField("createDocs", diffResult.CreateDocs).
				WithField("updateDocs", diffResult.UpdateDocs).
				WithField("deleteDocs", diffResult.DeleteDocs).
				Info("difference")
		}
	case config.TaskActionCompare:
		diffResultMap, err := t.bulkMigrator.Compare()
		if err != nil {
			return errors.WithStack(err)
		}

		for indexes, diffResult := range diffResultMap {
			indexArray := strings.Split(indexes, ":")
			utils.GetLogger(t.GetCtx()).
				WithField("sourceIndex", indexArray[0]).
				WithField("targetIndex", indexArray[1]).
				WithField("percent", diffResult.Percent()).
				WithField("create", diffResult.CreateCount).
				WithField("update", diffResult.UpdateCount).
				WithField("delete", diffResult.DeleteCount).
				WithField("createDocs", diffResult.CreateDocs).
				WithField("updateDocs", diffResult.UpdateDocs).
				WithField("deleteDocs", diffResult.DeleteDocs).
				Info("difference")
		}
	default:
		taskName := utils.GetCtxKeyTaskName(ctx)
		return fmt.Errorf("%s invalid task action %s", taskName, taskAction)
	}
	return nil
}
