package utils

import (
	"context"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"os"
)

var logger *log.Logger

func init() {
	logger = &log.Logger{
		Out:       os.Stdout,
		Formatter: &log.JSONFormatter{},
		Hooks:     make(log.LevelHooks),
		Level:     log.InfoLevel,
	}
	logger.SetReportCaller(true)
}

func GetLogger(ctx context.Context) *log.Entry {
	entry := log.NewEntry(logger)

	ctxKeyMap := map[CtxKey]func(ctx context.Context) string{
		CtxKeySourceESVersion: GetCtxKeySourceESVersion,
		CtxKeyTargetESVersion: GetCtxKeyTargetESVersion,
		CtxKeySourceIndex:     GetCtxKeySourceIndex,
		CtxKeyTargetIndex:     GetCtxKeyTargetIndex,
		CtxKeyTaskName:        GetCtxKeyTaskName,
		CtxKeyTaskID:          GetCtxKeyTaskID,
		CtxKeyTaskAction:      GetCtxKeyTaskAction,
	}
	for key, ctxFunc := range ctxKeyMap {
		value := ctx.Value(key)
		if lo.IsNotEmpty(value) {
			entry = entry.WithField(string(key), ctxFunc(ctx))
		}
	}
	return entry
}
