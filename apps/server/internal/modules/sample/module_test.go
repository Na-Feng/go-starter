package sample_test

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

	"github.com/stretchr/testify/require"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/httpx"
	"go-starter/internal/modules/sample"
)

type allowAllPerms struct{}

func (allowAllPerms) HasPermission(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func TestSampleModuleTableDriven(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwtMgr := auth.NewManager("test-secret")
	token, err := jwtMgr.Sign("usr_1", []string{"operator"}, auth.TokenTypeAccess, time.Hour)
	require.NoError(t, err)

	repo := sample.NewMemoryRepo()
	svc := sample.NewService(repo, nil)
	h := sample.NewHandler(svc, nil, log)

	mux := http.NewServeMux()
	h.Register(auth.NewRouter(mux, jwtMgr, nil, allowAllPerms{}, log))

	cases := []struct {
		name       string
		withAuth   bool
		body       string
		wantStatus int
	}{
		{name: "unauthenticated", withAuth: false, body: `{"title":"hello go"}`, wantStatus: http.StatusUnauthorized},
		{name: "title_too_short", withAuth: true, body: `{"title":"a"}`, wantStatus: http.StatusBadRequest},
		{name: "unknown_field_rejected", withAuth: true, body: `{"title":"hello","extra":1}`, wantStatus: http.StatusBadRequest},
		{name: "create_success", withAuth: true, body: `{"title":"hello go"}`, wantStatus: http.StatusCreated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/samples", bytes.NewBufferString(tc.body))
			if tc.withAuth {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			require.Equal(t, tc.wantStatus, rec.Code, rec.Body.String())
		})
	}

	recList := httptest.NewRecorder()
	mux.ServeHTTP(recList, httptest.NewRequest(http.MethodGet, "/api/v1/samples?page=1&page_size=10", nil))
	require.Equal(t, http.StatusOK, recList.Code)

	var listEnv struct {
		Data httpx.PageResult[sample.Item] `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recList.Body.Bytes(), &listEnv))
	require.Equal(t, int64(1), listEnv.Data.Total)
	require.Len(t, listEnv.Data.Items, 1)
}
