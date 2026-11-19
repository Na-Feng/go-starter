package sample

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-starter/internal/kernel/db"
	"go-starter/internal/kernel/errs"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) Repository {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, title string) (Item, error) {
	const q = `
		INSERT INTO samples (id, title)
		VALUES ($1, $2)
		RETURNING id, title, created_at`

	id := newSampleID()
	var item Item
	err := db.Conn(ctx, r.pool).QueryRow(ctx, q, id, title).Scan(
		&item.ID,
		&item.Title,
		&item.CreatedAt,
	)
	if err != nil {
		return Item{}, errs.Internal(err)
	}
	return item, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id string) (Item, error) {
	const q = `
		SELECT id, title, created_at
		FROM samples
		WHERE id = $1`

	var item Item
	err := db.Conn(ctx, r.pool).QueryRow(ctx, q, id).Scan(
		&item.ID,
		&item.Title,
		&item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, errs.NotFound("sample item not found")
		}
		return Item{}, errs.Internal(err)
	}
	return item, nil
}

func (r *PostgresRepo) List(ctx context.Context, limit, offset int) ([]Item, int64, error) {
	conn := db.Conn(ctx, r.pool)
	var total int64
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM samples`).Scan(&total); err != nil {
		return nil, 0, errs.Internal(err)
	}
	if total == 0 {
		return []Item{}, 0, nil
	}

	const q = `
		SELECT id, title, created_at
		FROM samples
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := conn.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, errs.Internal(err)
	}
	defer rows.Close()

	items := make([]Item, 0, limit)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Title, &it.CreatedAt); err != nil {
			return nil, 0, errs.Internal(err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errs.Internal(err)
	}
	return items, total, nil
}

func newSampleID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "smp_" + hex.EncodeToString(b[:])
}
