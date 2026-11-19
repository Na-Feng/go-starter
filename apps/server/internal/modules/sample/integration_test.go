//go:build integration

package sample_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/db"
	"go-starter/internal/kernel/httpx"
	"go-starter/internal/kernel/testkit"
	"go-starter/internal/modules/sample"
)

func TestSamplePostgresIntegration(t *testing.T) {
	pool := testkit.SetupPostgres(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwtMgr := auth.NewManager("integration-secret")
	token, err := jwtMgr.Sign("usr_int", []string{auth.SuperAdminRole}, auth.TokenTypeAccess, time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	txMgr := db.NewTxManager(pool)
	repo := sample.NewPostgresRepo(pool)
	svc := sample.NewService(repo, txMgr)
	h := sample.NewHandler(svc, nil, log)

	mux := http.NewServeMux()
	h.Register(mux, jwtMgr, nil, allowAllPerms{})

	// 1. 真实写入 Postgres
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/samples", bytes.NewBufferString(`{"title":"real-pg-item"}`))
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	recCreate := httptest.NewRecorder()
	mux.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recCreate.Code, recCreate.Body.String())
	}

	var createEnv struct {
		Data sample.Item `json:"data"`
	}
	_ = json.Unmarshal(recCreate.Body.Bytes(), &createEnv)
	if createEnv.Data.ID == "" {
		t.Fatalf("expected generated ID from postgres")
	}

	// 2. 真实分页查询 Postgres
	recList := httptest.NewRecorder()
	mux.ServeHTTP(recList, httptest.NewRequest(http.MethodGet, "/api/v1/samples?page=1&page_size=10", nil))
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", recList.Code)
	}
	var listEnv struct {
		Data httpx.PageResult[sample.Item] `json:"data"`
	}
	_ = json.Unmarshal(recList.Body.Bytes(), &listEnv)
	if listEnv.Data.Total != 1 || len(listEnv.Data.Items) != 1 || listEnv.Data.Items[0].Title != "real-pg-item" {
		t.Fatalf("unexpected postgres list result: %+v", listEnv.Data)
	}
}
