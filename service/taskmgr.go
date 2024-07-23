package service

import (
	"context"
	"fmt"
	"github.com/CharellKing/ela/config"
	"github.com/CharellKing/ela/pkg/es"
	"github.com/CharellKing/ela/utils"
	"github.com/pkg/errors"
)

type TaskMgr struct {
	usedESMap map[string]es.ES
	taskCfgs  []*config.TaskCfg
}

func NewTaskMgr(cfg *config.Config) (*TaskMgr, error) {
	usedESMap := make(map[string]es.ES)
	for _, task := range cfg.Tasks {
		if cfg.ESConfigs[task.SourceES] == nil {
			return nil, fmt.Errorf("source es config not found: %s", task.SourceES)
		}

		if cfg.ESConfigs[task.TargetES] == nil {
			return nil, fmt.Errorf("target es config not found: %s", task.TargetES)
		}

		var err error
		sourceES := es.NewESV0(cfg.ESConfigs[task.SourceES])
		usedESMap[task.SourceES], err = sourceES.GetES()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		targetES := es.NewESV0(cfg.ESConfigs[task.TargetES])
		usedESMap[task.TargetES], err = targetES.GetES()
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return &TaskMgr{
		usedESMap: usedESMap,
		taskCfgs:  cfg.Tasks,
	}, nil
}

func (t *TaskMgr) Run(ctx context.Context) error {
	for _, taskCfg := range t.taskCfgs {
		task := NewTaskWithES(ctx, taskCfg, t.usedESMap[taskCfg.SourceES], t.usedESMap[taskCfg.TargetES])
		if err := task.Run(); err != nil {
			return errors.WithStack(err)
		}

		utils.GetLogger(task.GetCtx()).Info("task done")
	}
	return nil
}
