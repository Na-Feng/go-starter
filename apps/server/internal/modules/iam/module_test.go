package iam_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/modules/iam"
)

type memBlacklist struct {
	mu  sync.RWMutex
	ids map[string]bool
}

func (b *memBlacklist) Revoke(_ context.Context, jti string, _ time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ids[jti] = true
	return nil
}

func (b *memBlacklist) IsRevoked(_ context.Context, jti string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ids[jti], nil
}

type memIAMRepo struct {
	mu        sync.RWMutex
	users     map[string]iam.User
	userRoles map[string][]string
	roles     map[string]iam.Role
	rolePerms map[string][]string
}

func newMemIAMRepo() *memIAMRepo {
	return &memIAMRepo{
		users:     make(map[string]iam.User),
		userRoles: make(map[string][]string),
		roles: map[string]iam.Role{
			"role_super_admin": {ID: "role_super_admin", Code: "super_admin", Name: "超级管理员"},
			"role_operator":    {ID: "role_operator", Code: "operator", Name: "普通运营"},
		},
		rolePerms: map[string][]string{
			"role_operator": {"iam:user:read"},
		},
	}
}

func (m *memIAMRepo) CountUsers(_ context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.users)), nil
}

func (m *memIAMRepo) CreateUser(_ context.Context, username, hash, nickname string) (iam.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := iam.User{ID: "usr_" + username, Username: username, PasswordHash: hash, Nickname: nickname, Status: "active"}
	m.users[u.ID] = u
	return u, nil
}

func (m *memIAMRepo) GetUserByUsername(_ context.Context, username string) (iam.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return iam.User{}, errs.NotFound("user not found")
}

func (m *memIAMRepo) GetUserByID(_ context.Context, id string) (iam.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return iam.User{}, errs.NotFound("user not found")
	}
	return u, nil
}

func (m *memIAMRepo) ListUsers(_ context.Context, _, _ int) ([]iam.User, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]iam.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, int64(len(out)), nil
}

func (m *memIAMRepo) UpdateUserStatus(_ context.Context, id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.users[id]
	u.Status = status
	m.users[id] = u
	return nil
}

func (m *memIAMRepo) UpdateUserPassword(_ context.Context, id, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.users[id]
	u.PasswordHash = hash
	m.users[id] = u
	return nil
}

func (m *memIAMRepo) GetUserRoleCodes(_ context.Context, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var codes []string
	for _, rid := range m.userRoles[userID] {
		if r, ok := m.roles[rid]; ok {
			codes = append(codes, r.Code)
		}
	}
	return codes, nil
}

func (m *memIAMRepo) GetUserPermissionCodes(_ context.Context, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var perms []string
	for _, rid := range m.userRoles[userID] {
		perms = append(perms, m.rolePerms[rid]...)
	}
	return perms, nil
}

func (m *memIAMRepo) ReplaceUserRoles(_ context.Context, userID string, roleIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userRoles[userID] = roleIDs
	return nil
}

func (m *memIAMRepo) ListRoles(_ context.Context) ([]iam.Role, error) { return nil, nil }
func (m *memIAMRepo) CreateRole(_ context.Context, c, n string) (iam.Role, error) {
	return iam.Role{Code: c, Name: n}, nil
}
func (m *memIAMRepo) ListPermissions(_ context.Context) ([]iam.Permission, error) { return nil, nil }
func (m *memIAMRepo) ReplaceRolePermissions(_ context.Context, _ string, _ []string) ([]string, error) {
	return nil, nil
}

func TestIAMAuthAndRBACFlow(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwtMgr := auth.NewManager("test-secret")
	bl := &memBlacklist{ids: make(map[string]bool)}
	repo := newMemIAMRepo()
	svc := iam.NewService(repo, nil, jwtMgr, bl, nil, nil, time.Hour, 24*time.Hour)
	if err := svc.EnsureSeedAdmin(context.Background()); err != nil {
		t.Fatalf("seed admin failed: %v", err)
	}

	mux := http.NewServeMux()
	iam.NewHandler(svc, log).Register(mux, jwtMgr, bl)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewBufferString(`{"username":"admin","password":"admin123"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 login, got %d", rec.Code)
	}
	var loginEnv struct {
		Data iam.TokenPair `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &loginEnv)
	token := loginEnv.Data.AccessToken

	reqUsers := httptest.NewRequest(http.MethodGet, "/api/v1/iam/users", nil)
	reqUsers.Header.Set("Authorization", "Bearer "+token)
	recUsers := httptest.NewRecorder()
	mux.ServeHTTP(recUsers, reqUsers)
	if recUsers.Code != http.StatusOK {
		t.Fatalf("expected 200 for super_admin, got %d", recUsers.Code)
	}

	reqLogout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqLogout.Header.Set("Authorization", "Bearer "+token)
	recLogout := httptest.NewRecorder()
	mux.ServeHTTP(recLogout, reqLogout)
	if recLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 logout, got %d", recLogout.Code)
	}

	reqAfterLogout := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqAfterLogout.Header.Set("Authorization", "Bearer "+token)
	recAfterLogout := httptest.NewRecorder()
	mux.ServeHTTP(recAfterLogout, reqAfterLogout)
	if recAfterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", recAfterLogout.Code)
	}
}
