package iam

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"go-starter/internal/kernel/db"
	"go-starter/internal/kernel/errs"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) Repository {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	if err := db.Conn(ctx, r.pool).QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, errs.Internal(err)
	}
	return n, nil
}

func (r *PostgresRepo) CreateUser(ctx context.Context, username, passwordHash, nickname string) (User, error) {
	const q = `
		INSERT INTO users (id, username, password_hash, nickname)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, password_hash, nickname, status, created_at`
	id := newID("usr_")
	var u User
	err := db.Conn(ctx, r.pool).QueryRow(ctx, q, id, username, passwordHash, nickname).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &u.Status, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, errs.Conflict("username already exists")
		}
		return User{}, errs.Internal(err)
	}
	return u, nil
}

func (r *PostgresRepo) GetUserByUsername(ctx context.Context, username string) (User, error) {
	return r.scanUser(ctx, `SELECT id, username, password_hash, nickname, status, created_at FROM users WHERE username = $1`, username)
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, id string) (User, error) {
	return r.scanUser(ctx, `SELECT id, username, password_hash, nickname, status, created_at FROM users WHERE id = $1`, id)
}

func (r *PostgresRepo) scanUser(ctx context.Context, q, arg string) (User, error) {
	var u User
	err := db.Conn(ctx, r.pool).QueryRow(ctx, q, arg).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &u.Status, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, errs.NotFound("user not found")
		}
		return User{}, errs.Internal(err)
	}
	return u, nil
}

func (r *PostgresRepo) ListUsers(ctx context.Context, limit, offset int) ([]User, int64, error) {
	conn := db.Conn(ctx, r.pool)
	var total int64
	if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, errs.Internal(err)
	}
	rows, err := conn.Query(ctx, `
		SELECT id, username, nickname, status, created_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, errs.Internal(err)
	}
	defer rows.Close()
	users := make([]User, 0, limit)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Nickname, &u.Status, &u.CreatedAt); err != nil {
			return nil, 0, errs.Internal(err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *PostgresRepo) UpdateUserStatus(ctx context.Context, id, status string) error {
	tag, err := db.Conn(ctx, r.pool).Exec(ctx, `UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return errs.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return errs.NotFound("user not found")
	}
	return nil
}

func (r *PostgresRepo) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	tag, err := db.Conn(ctx, r.pool).Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, passwordHash, id)
	if err != nil {
		return errs.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return errs.NotFound("user not found")
	}
	return nil
}

func (r *PostgresRepo) GetUserRoleCodes(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.Conn(ctx, r.pool).Query(ctx, `
		SELECT r.code FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, errs.Internal(err)
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, errs.Internal(err)
		}
		codes = append(codes, c)
	}
	return codes, rows.Err()
}

func (r *PostgresRepo) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.Conn(ctx, r.pool).Query(ctx, `
		SELECT DISTINCT p.code FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, errs.Internal(err)
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, errs.Internal(err)
		}
		codes = append(codes, c)
	}
	return codes, rows.Err()
}

func (r *PostgresRepo) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string) error {
	conn := db.Conn(ctx, r.pool)
	if _, err := conn.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return errs.Internal(err)
	}
	for _, rid := range roleIDs {
		if _, err := conn.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, rid); err != nil {
			return errs.BadRequest("invalid role_id: " + rid)
		}
	}
	return nil
}

func (r *PostgresRepo) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := db.Conn(ctx, r.pool).Query(ctx, `SELECT id, code, name, created_at FROM roles ORDER BY created_at ASC`)
	if err != nil {
		return nil, errs.Internal(err)
	}
	defer rows.Close()
	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.CreatedAt); err != nil {
			return nil, errs.Internal(err)
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *PostgresRepo) CreateRole(ctx context.Context, code, name string) (Role, error) {
	id := newID("role_")
	var role Role
	err := db.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO roles (id, code, name) VALUES ($1, $2, $3)
		RETURNING id, code, name, created_at`, id, code, name).
		Scan(&role.ID, &role.Code, &role.Name, &role.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return Role{}, errs.Conflict("role code already exists")
		}
		return Role{}, errs.Internal(err)
	}
	return role, nil
}

func (r *PostgresRepo) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := db.Conn(ctx, r.pool).Query(ctx, `SELECT id, code, name FROM permissions ORDER BY code ASC`)
	if err != nil {
		return nil, errs.Internal(err)
	}
	defer rows.Close()
	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Name); err != nil {
			return nil, errs.Internal(err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *PostgresRepo) ReplaceRolePermissions(ctx context.Context, roleID string, permIDs []string) ([]string, error) {
	conn := db.Conn(ctx, r.pool)
	if _, err := conn.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return nil, errs.Internal(err)
	}
	for _, pid := range permIDs {
		if _, err := conn.Exec(ctx, `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)`, roleID, pid); err != nil {
			return nil, errs.BadRequest("invalid permission_id: " + pid)
		}
	}
	rows, err := conn.Query(ctx, `SELECT user_id FROM user_roles WHERE role_id = $1`, roleID)
	if err != nil {
		return nil, errs.Internal(err)
	}
	defer rows.Close()
	var affected []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			affected = append(affected, uid)
		}
	}
	return affected, nil
}

type RedisPermCache struct {
	rdb *redis.Client
}

func NewRedisPermCache(rdb *redis.Client) PermCache {
	return &RedisPermCache{rdb: rdb}
}

func (c *RedisPermCache) Get(ctx context.Context, userID string) ([]string, bool, error) {
	if c.rdb == nil {
		return nil, false, nil
	}
	raw, err := c.rdb.Get(ctx, "iam:perm:"+userID).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var perms []string
	if err := json.Unmarshal(raw, &perms); err != nil {
		return nil, false, err
	}
	return perms, true, nil
}

func (c *RedisPermCache) Set(ctx context.Context, userID string, perms []string) error {
	if c.rdb == nil {
		return nil
	}
	raw, _ := json.Marshal(perms)
	return c.rdb.Set(ctx, "iam:perm:"+userID, raw, 5*time.Minute).Err()
}

func (c *RedisPermCache) Delete(ctx context.Context, userIDs ...string) error {
	if c.rdb == nil || len(userIDs) == 0 {
		return nil
	}
	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = "iam:perm:" + id
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func newID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + hex.EncodeToString(b[:])
}
