package async_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"go-starter/internal/kernel/async"
	"go-starter/internal/kernel/logger"
)

type syncLogBuffer struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	notify chan struct{}
	once   sync.Once
}

func (s *syncLogBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.buf.Write(p)
	s.once.Do(func() { close(s.notify) })
	return n, err
}

func (s *syncLogBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestAsyncGoDetachesCancelPreservesValuesAndRecoversPanic(t *testing.T) {
	reqCtx, cancelReq := context.WithCancel(logger.WithRequestID(context.Background(), "req_trace_1"))

	done := make(chan string, 1)
	cancelReq()

	async.Go(reqCtx, slog.Default(), "test-detach", func(bgCtx context.Context) {
		if bgCtx.Err() != nil {
			done <- "unexpected_canceled"
			return
		}
		done <- logger.RequestIDFrom(bgCtx)
	})

	select {
	case got := <-done:
		if got != "req_trace_1" {
			t.Fatalf("expected request_id preserved without cancel, got %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("async.Go task timed out")
	}

	logBuf := &syncLogBuffer{notify: make(chan struct{})}
	log := slog.New(slog.NewTextHandler(logBuf, nil))
	async.Go(context.Background(), log, "test-panic", func(_ context.Context) {
		panic("simulated background failure")
	})

	select {
	case <-logBuf.notify:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for panic recovery log")
	}

	if !strings.Contains(logBuf.String(), "async goroutine panic recovered") {
		t.Fatalf("expected recovered panic log, got %q", logBuf.String())
	}
}
