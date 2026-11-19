//go:build integration

package iam_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-starter/internal/kernel/audit"
	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/db"
	"go-starter/internal/kernel/httpx"
	"go-starter/internal/kernel/testkit"
	"go-starter/internal/modules/iam"
)

func TestIAMPostgresRedisIntegration(t *testing.T) {
	pool := testkit.SetupPostgres(t)
	rdb := testkit.SetupRedis(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	txMgr := db.NewTxManager(pool)
	jwtMgr := auth.NewManager("integration-secret")
	bl := auth.NewRedisBlacklist(rdb)
	permCache := iam.NewRedisPermCache(rdb)
	auditRec := audit.NewPostgresRecorder(pool)
	repo := iam.NewPostgresRepo(pool)

	svc := iam.NewService(repo, txMgr, jwtMgr, bl, permCache, auditRec, time.Hour, 24*time.Hour)
	if err := svc.EnsureSeedAdmin(context.Background()); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	mux := http.NewServeMux()
	iam.NewHandler(svc, log).Register(mux, jwtMgr, bl)

	// 1. 真实库登录
	recLogin := httptest.NewRecorder()
	mux.ServeHTTP(recLogin, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewBufferString(`{"username":"admin","password":"admin123"}`)))
	if recLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 login, got %d: %s", recLogin.Code, recLogin.Body.String())
	}
	var loginEnv struct {
		Data iam.TokenPair `json:"data"`
	}
	_ = json.Unmarshal(recLogin.Body.Bytes(), &loginEnv)
	token := loginEnv.Data.AccessToken

	// 2. 创建用户（触发真实 Postgres 事务 + 审计日志落库）
	reqUser := httptest.NewRequest(http.MethodPost, "/api/v1/iam/users",
		bytes.NewBufferString(`{"username":"alice","password":"secret123","nickname":"Alice","role_ids":["role_operator"]}`))
	reqUser.Header.Set("Authorization", "Bearer "+token)
	recUser := httptest.NewRecorder()
	mux.ServeHTTP(recUser, reqUser)
	if recUser.Code != http.StatusCreated {
		t.Fatalf("expected 201 create user, got %d: %s", recUser.Code, recUser.Body.String())
	}

	// 3. 重复用户名触发真实唯一索引冲突 -> 409 Conflict
	reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/iam/users",
		bytes.NewBufferString(`{"username":"alice","password":"secret123"}`))
	reqDup.Header.Set("Authorization", "Bearer "+token)
	recDup := httptest.NewRecorder()
	mux.ServeHTTP(recDup, reqDup)
	if recDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d", recDup.Code)
	}

	// 4. 查询真实 audit_logs 表确认审计记录已写入
	reqAudit := httptest.NewRequest(http.MethodGet, "/api/v1/iam/audit-logs?page=1&page_size=10", nil)
	reqAudit.Header.Set("Authorization", "Bearer "+token)
	recAudit := httptest.NewRecorder()
	mux.ServeHTTP(recAudit, reqAudit)
	if recAudit.Code != http.StatusOK {
		t.Fatalf("expected 200 audit logs, got %d", recAudit.Code)
	}
	var auditEnv struct {
		Data httpx.PageResult[audit.Entry] `json:"data"`
	}
	_ = json.Unmarshal(recAudit.Body.Bytes(), &auditEnv)
	if auditEnv.Data.Total < 1 || auditEnv.Data.Items[0].Action != "user.create" {
		t.Fatalf("expected user.create audit log, got %+v", auditEnv.Data)
	}

	// 5. 真实 Redis 黑名单登出校验
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqLogout.Header.Set("Authorization", "Bearer "+token)
	recLogout := httptest.NewRecorder()
	mux.ServeHTTP(recLogout, reqLogout)
	if recLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 logout, got %d", recLogout.Code)
	}

	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+token)
	recMe := httptest.NewRecorder()
	mux.ServeHTTP(recMe, reqMe)
	if recMe.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after real redis logout, got %d", recMe.Code)
	}
}
