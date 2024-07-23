package utils

import (
	"context"
	"github.com/spf13/cast"
)

type CtxKey string

const (
	CtxKeySourceESVersion CtxKey = "sourceEsSource"
	CtxKeyTargetESVersion CtxKey = "targetEsTarget"
	CtxKeySourceIndex     CtxKey = "sourceIndex"
	CtxKeyTargetIndex     CtxKey = "targetIndex"
	CtxKeyTaskName        CtxKey = "taskName"
	CtxKeyTaskID          CtxKey = "taskId"
	CtxKeyTaskAction      CtxKey = "taskAction"
)

func GetCtxKeySourceESVersion(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxKeySourceESVersion))
}

func GetCtxKeyTargetESVersion(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxKeyTargetESVersion))
}

func SetCtxKeySourceESVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, CtxKeySourceESVersion, version)
}

func SetCtxKeyTargetESVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, CtxKeyTargetESVersion, version)
}

func GetCtxKeySourceIndex(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxKeySourceIndex))
}

func GetCtxKeyTargetIndex(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxKeyTargetIndex))
}

func SetCtxKeySourceIndex(ctx context.Context, index string) context.Context {
	return context.WithValue(ctx, CtxKeySourceIndex, index)
}

func SetCtxKeyTargetIndex(ctx context.Context, index string) context.Context {
	return context.WithValue(ctx, CtxKeyTargetIndex, index)
}

func GetCtxKeyTaskName(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxKeyTaskName))
}

func GetCtxKeyTaskID(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxKeyTaskID))
}

func SetCtxKeyTaskName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, CtxKeyTaskName, name)
}

func SetCtxKeyTaskID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyTaskID, id)
}

func GetCtxKeyTaskAction(ctx context.Context) string {
	return cast.ToString(ctx.Value(CtxKeyTaskAction))
}

func SetCtxKeyTaskAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, CtxKeyTaskAction, action)
}
