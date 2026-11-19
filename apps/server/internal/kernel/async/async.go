package async

import (
	"context"
	"log/slog"
	"runtime/debug"
)

// Go 安全启动后台协程：自动捕获 panic 并记录堆栈，防止单次异步任务崩溃导致整个进程退出
func Go(ctx context.Context, log *slog.Logger, taskName string, fn func(ctx context.Context)) {
	go func() {
		defer func() {
			if rec := recover(); rec != nil && log != nil {
				log.Error("async goroutine panic recovered",
					"task", taskName,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn(ctx)
	}()
}
