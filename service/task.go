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
		WithParallelism(taskCfg.Parallelism).WithScrollTime(taskCfg.ScrollTime)
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

func (t *Task) Run() error {
	ctx := t.GetCtx()
	taskAction := config.TaskAction(utils.GetCtxKeyTaskAction(ctx))
	switch taskAction {
	case config.TaskActionCopyIndex:
		return t.bulkMigrator.CopyIndexSettings(t.force)
	case config.TaskActionSync:
		return t.bulkMigrator.Sync(t.force)
	case config.TaskActionSyncDiff:
		var diffs map[string][]utils.HashDiff
		err := t.bulkMigrator.SyncDiff(&diffs)
		if err != nil {
			return errors.WithStack(err)
		}

		for indexes, diff := range diffs {
			indexArray := strings.Split(indexes, ":")
			for _, d := range diff {
				utils.GetLogger(t.GetCtx()).WithField("docId", d.Id).
					WithField("sourceIndex", indexArray[0]).
					WithField("targetIndex", indexArray[1]).Info("difference")
			}
		}
	case config.TaskActionCompare:
		diffs, err := t.bulkMigrator.Compare()
		if err != nil {
			return errors.WithStack(err)
		}

		for indexes, diff := range diffs {
			indexArray := strings.Split(indexes, ":")
			for _, d := range diff {
				utils.GetLogger(t.GetCtx()).WithField("docId", d.Id).
					WithField("sourceIndex", indexArray[0]).
					WithField("targetIndex", indexArray[1]).Info("difference")
			}

		}
	default:
		taskName := utils.GetCtxKeyTaskName(ctx)
		return fmt.Errorf("%s invalid task action %s", taskName, taskAction)
	}
	return nil
}
