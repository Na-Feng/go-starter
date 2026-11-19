package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
)

type ctxKey struct{}

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
	SuperAdminRole   = "super_admin"
)

type UserClaims struct {
	UserID    string   `json:"user_id"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenBlacklist interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, permCode string) (bool, error)
}

type RedisBlacklist struct {
	rdb *redis.Client
}

func NewRedisBlacklist(rdb *redis.Client) *RedisBlacklist {
	return &RedisBlacklist{rdb: rdb}
}

func (b *RedisBlacklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if b.rdb == nil || jti == "" || ttl <= 0 {
		return nil
	}
	return b.rdb.Set(ctx, "auth:bl:"+jti, "1", ttl).Err()
}

func (b *RedisBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if b.rdb == nil || jti == "" {
		return false, nil
	}
	n, err := b.rdb.Exists(ctx, "auth:bl:"+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

type Manager struct {
	secret []byte
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

func (m *Manager) Sign(userID string, roles []string, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := UserClaims{
		UserID:    userID,
		Roles:     roles,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        newJTI(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) Verify(tokenStr string) (UserClaims, error) {
	var claims UserClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return UserClaims{}, errs.Unauthorized("invalid or expired token")
	}
	return claims, nil
}

func WithUser(ctx context.Context, claims UserClaims) context.Context {
	return context.WithValue(ctx, ctxKey{}, claims)
}

func UserFrom(ctx context.Context) (UserClaims, bool) {
	c, ok := ctx.Value(ctxKey{}).(UserClaims)
	return c, ok
}

func RequireAuth(mgr *Manager, bl TokenBlacklist, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				httpx.Fail(w, r, log, errs.Unauthorized("missing bearer token"))
				return
			}
			claims, err := mgr.Verify(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				httpx.Fail(w, r, log, err)
				return
			}
			if claims.TokenType != TokenTypeAccess {
				httpx.Fail(w, r, log, errs.Unauthorized("invalid token type"))
				return
			}
			if bl != nil {
				revoked, err := bl.IsRevoked(r.Context(), claims.ID)
				if err != nil {
					httpx.Fail(w, r, log, errs.Internal(err))
					return
				}
				if revoked {
					httpx.Fail(w, r, log, errs.Unauthorized("token has been revoked"))
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), claims)))
		})
	}
}

func RequirePermission(permCode string, checker PermissionChecker, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := UserFrom(r.Context())
			if !ok {
				httpx.Fail(w, r, log, errs.Unauthorized("unauthenticated"))
				return
			}
			if slices.Contains(claims.Roles, SuperAdminRole) {
				next.ServeHTTP(w, r)
				return
			}
			allowed, err := checker.HasPermission(r.Context(), claims.UserID, permCode)
			if err != nil {
				httpx.Fail(w, r, log, errs.Internal(err))
				return
			}
			if !allowed {
				httpx.Fail(w, r, log, errs.Forbidden("missing permission: "+permCode))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func newJTI() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
