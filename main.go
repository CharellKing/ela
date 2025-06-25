package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/CharellKing/ela-lib/config"
	"github.com/CharellKing/ela-lib/service/gateway"
	"github.com/CharellKing/ela-lib/service/task"
	"github.com/CharellKing/ela-lib/utils"
	goflags "github.com/jessevdk/go-flags"
	"github.com/samber/lo"

	"github.com/spf13/viper"
)

func main() {
	var err error
	cmd := &Cmd{}

	// parse args
	_, err = goflags.Parse(cmd)
	if err != nil {
		log.Fatal(err)
		return
	}

	if cmd.ConfigFile == "" {
		fmt.Println("Usage: go run main.go --config <config_path> [--gateway] [--tasks] [--task <task name>]")
		return
	}

	viper.SetConfigFile(cmd.ConfigFile)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Unable reading config file, %v\n", err)
		return
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Printf("Unable to decode into struct, %v\n", err)
		return
	}

	utils.InitLogger(&cfg)

	utils.RegisterProgressCallBack(func(ctx context.Context, progress *utils.Progress) {
		progressGetter := []func(ctx context.Context) *utils.Progress{
			utils.GetCtxKeyTaskProgress,
			utils.GetCtxKeySourceIndexPairProgress,
			utils.GetCtxKeyTargetIndexPairProgress,
			utils.GetCtxKeySourceQueueExtrusion,
			utils.GetCtxKeyTargetQueueExtrusion,
			utils.GetCtxKeyPairProgress,
		}

		logger := utils.GetLogger(ctx)
		for _, progressGetter := range progressGetter {
			progress := progressGetter(ctx)
			if progress != nil && progress.Name != "" {
				logger = logger.WithField(progress.Name, fmt.Sprintf("%d/%d", progress.Current.Load(), progress.Total))
			}
		}

		ctxKeyMap := map[utils.CtxKey]func(ctx context.Context) string{
			utils.CtxKeySourceESVersion: utils.GetCtxKeySourceESVersion,
			utils.CtxKeyTargetESVersion: utils.GetCtxKeyTargetESVersion,
			utils.CtxKeySourceObject:    utils.GetCtxKeySourceObject,
			utils.CtxKeyTargetObject:    utils.GetCtxKeyTargetObject,
			utils.CtxKeyTaskName:        utils.GetCtxKeyTaskName,
			utils.CtxKeyTaskID:          utils.GetCtxKeyTaskID,
			utils.CtxKeyTaskAction:      utils.GetCtxKeyTaskAction,
		}
		for key, ctxFunc := range ctxKeyMap {
			value := ctx.Value(key)
			if lo.IsNotEmpty(value) {
				logger = logger.WithField(string(key), ctxFunc(ctx))
			}
		}

		logger.Info("progress")
	})

	ctx := context.Background()

	if cmd.Tasks {
		taskMgr, err := task.NewTaskMgr(&cfg)
		if err != nil {
			utils.GetLogger(ctx).WithError(err).Error("create task manager")
			return
		}

		if err := taskMgr.Run(ctx); err != nil {
			utils.GetLogger(ctx).WithError(err).Error("run task manager")
			return
		}
		return
	}

	if cmd.Gateway {
		esProxy, err := gateway.NewESGateway(&cfg)
		if err != nil {
			utils.GetLogger(ctx).Errorf("create task manager %+v", err)
			return
		}
		esProxy.Run()
		return
	}

	if cmd.Task != "" {
		taskMgr, err := task.NewTaskMgr(&cfg)
		if err != nil {
			utils.GetLogger(ctx).WithError(err).Error("create task manager")
			return
		}

		if err := taskMgr.Run(ctx, strings.Fields(cmd.Task)...); err != nil {
			utils.GetLogger(ctx).WithError(err).Error("run task manager")
			return
		}
		return
	}

	return
}
