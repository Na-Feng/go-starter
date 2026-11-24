package async

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

const DefaultTimeout = 30 * time.Second

// Go 安全启动后台协程：剥离上游 HTTP 请求取消信号（保留 RequestID 等上下文值）、注入超时保护并捕获 panic。
func Go(parentCtx context.Context, log *slog.Logger, taskName string, fn func(ctx context.Context)) {
	GoWithTimeout(parentCtx, DefaultTimeout, log, taskName, fn)
}

// GoWithTimeout 使用指定超时在独立生命周期上下文中执行后台任务。
func GoWithTimeout(
	parentCtx context.Context,
	timeout time.Duration,
	log *slog.Logger,
	taskName string,
	fn func(ctx context.Context),
) {
	detached := context.WithoutCancel(parentCtx)
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
		if timeout > 0 {
			ctx, cancel := context.WithTimeout(detached, timeout)
			defer cancel()
			fn(ctx)
			return
		}
		fn(detached)
	}()
}
