package utils

import (
	"context"
	"runtime"
)

func GoRecovery(ctx context.Context, f func()) {
	defer Recovery(ctx)
	go f()
}

func Recovery(ctx context.Context) {
	if err := recover(); err != nil {
		buf := make([]byte, 64<<10) //nolint:gomnd
		n := runtime.Stack(buf, false)
		buf = buf[:n]
		GetLogger(ctx).WithField("error", err).WithField("stack", string(buf)).Error("panic recovered")
	}
}
