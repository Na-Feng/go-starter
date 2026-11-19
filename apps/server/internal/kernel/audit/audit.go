package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-starter/internal/kernel/db"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
	"go-starter/internal/kernel/logger"
)

type Entry struct {
	ID         string          `json:"id"`
	ActorID    string          `json:"actor_id"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Detail     json.RawMessage `json:"detail"`
	IP         string          `json:"ip"`
	CreatedAt  time.Time       `json:"created_at"`
}

type Recorder interface {
	Record(ctx context.Context, actorID, action, targetType, targetID string, detail any) error
	List(ctx context.Context, pq httpx.PageQuery) (httpx.PageResult[Entry], error)
}

type PostgresRecorder struct {
	pool *pgxpool.Pool
}

func NewPostgresRecorder(pool *pgxpool.Pool) Recorder {
	return &PostgresRecorder{pool: pool}
}

func (r *PostgresRecorder) Record(ctx context.Context, actorID, action, targetType, targetID string, detail any) error {
	payload := []byte("{}")
	if detail != nil {
		if raw, err := json.Marshal(detail); err == nil {
			payload = raw
		}
	}
	const q = `
		INSERT INTO audit_logs (id, actor_id, action, target_type, target_id, detail, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.Conn(ctx, r.pool).Exec(ctx, q,
		newAuditID(), actorID, action, targetType, targetID, payload, logger.ClientIPFrom(ctx),
	)
	if err != nil {
		return errs.Internal(err)
	}
	return nil
}

func (r *PostgresRecorder) List(ctx context.Context, pq httpx.PageQuery) (httpx.PageResult[Entry], error) {
	conn := db.Conn(ctx, r.pool)
	var total int64
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		return httpx.PageResult[Entry]{}, errs.Internal(err)
	}
	rows, err := conn.Query(ctx, `
		SELECT id, actor_id, action, target_type, target_id, detail, ip, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, pq.Limit(), pq.Offset())
	if err != nil {
		return httpx.PageResult[Entry]{}, errs.Internal(err)
	}
	defer rows.Close()

	items := make([]Entry, 0, pq.Limit())
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.Action, &e.TargetType, &e.TargetID, &e.Detail, &e.IP, &e.CreatedAt); err != nil {
			return httpx.PageResult[Entry]{}, errs.Internal(err)
		}
		items = append(items, e)
	}
	return httpx.PageResult[Entry]{
		Items: items, Total: total, Page: pq.Page, PageSize: pq.PageSize,
	}, rows.Err()
}

func newAuditID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "aud_" + hex.EncodeToString(b[:])
}
