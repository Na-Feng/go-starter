package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

const TypeSampleExport = "sample:export"

type SampleExportPayload struct {
	RequestedBy string `json:"requested_by"`
}

type Client struct {
	inner *asynq.Client
}

func NewClient(redisAddr, redisPassword string, redisDB int) *Client {
	return &Client{
		inner: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		}),
	}
}

func (c *Client) EnqueueSampleExport(ctx context.Context, requestedBy string) (string, error) {
	raw, err := json.Marshal(SampleExportPayload{RequestedBy: requestedBy})
	if err != nil {
		return "", err
	}
	info, err := c.inner.EnqueueContext(ctx, asynq.NewTask(TypeSampleExport, raw), asynq.MaxRetry(3))
	if err != nil {
		return "", fmt.Errorf("enqueue task: %w", err)
	}
	return info.ID, nil
}

func (c *Client) Close() error {
	return c.inner.Close()
}

type Server struct {
	inner *asynq.Server
	log   *slog.Logger
}

func NewServer(redisAddr, redisPassword string, redisDB, concurrency int, log *slog.Logger) *Server {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		},
		asynq.Config{Concurrency: concurrency},
	)
	return &Server{inner: srv, log: log}
}

func (s *Server) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeSampleExport, func(ctx context.Context, t *asynq.Task) error {
		var p SampleExportPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		s.log.Info("async task executed", "type", t.Type(), "requested_by", p.RequestedBy)
		return nil
	})
	return s.inner.Start(mux)
}

func (s *Server) Shutdown() {
	s.inner.Shutdown()
}
