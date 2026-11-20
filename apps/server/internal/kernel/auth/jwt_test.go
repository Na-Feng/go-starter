package auth_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"testing/quick"
	"time"

	"go-starter/internal/kernel/auth"
)

type fixedPermChecker struct {
	granted []string
}

func (c fixedPermChecker) HasPermission(_ context.Context, _, permCode string) (bool, error) {
	return slices.Contains(c.granted, permCode), nil
}

func TestJWTRoundTripAndTokenTypeInvariants(t *testing.T) {
	mgr := auth.NewManager("prop-secret-key")
	otherMgr := auth.NewManager("different-secret-key")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	prop := func(userID string, roles []string) bool {
		if userID == "" {
			userID = "usr_default"
		}
		accessTok, err := mgr.Sign(userID, roles, auth.TokenTypeAccess, 10*time.Minute)
		if err != nil {
			return false
		}
		refreshTok, err := mgr.Sign(userID, roles, auth.TokenTypeRefresh, time.Hour)
		if err != nil {
			return false
		}

		claims, err := mgr.Verify(accessTok)
		if err != nil || claims.UserID != userID || claims.TokenType != auth.TokenTypeAccess || claims.ID == "" {
			return false
		}

		if _, err := otherMgr.Verify(accessTok); err == nil {
			return false
		}

		mw := auth.RequireAuth(mgr, nil, log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		recRefreshAsAccess := httptest.NewRecorder()
		reqRefresh := httptest.NewRequest(http.MethodGet, "/protected", nil)
		reqRefresh.Header.Set("Authorization", "Bearer "+refreshTok)
		mw.ServeHTTP(recRefreshAsAccess, reqRefresh)
		if recRefreshAsAccess.Code != http.StatusUnauthorized {
			return false
		}

		recValidAccess := httptest.NewRecorder()
		reqAccess := httptest.NewRequest(http.MethodGet, "/protected", nil)
		reqAccess.Header.Set("Authorization", "Bearer "+accessTok)
		mw.ServeHTTP(recValidAccess, reqAccess)
		return recValidAccess.Code == http.StatusOK
	}

	if err := quick.Check(prop, &quick.Config{MaxCount: 120}); err != nil {
		t.Fatalf("JWT token invariant violated: %v", err)
	}
}

func TestRequirePermissionAlgebraicInvariant(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	prop := func(isSuperAdmin bool, hasExplicitPerm bool, targetPerm string) bool {
		if targetPerm == "" {
			targetPerm = "sample:write"
		}
		var roles []string
		if isSuperAdmin {
			roles = append(roles, auth.SuperAdminRole)
		} else {
			roles = append(roles, "operator")
		}
		var granted []string
		if hasExplicitPerm {
			granted = append(granted, targetPerm)
		}

		handler := auth.RequirePermission(targetPerm, fixedPermChecker{granted: granted}, log)(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodPost, "/op", nil)
		req = req.WithContext(auth.WithUser(req.Context(), auth.UserClaims{
			UserID: "usr_prop",
			Roles:  roles,
		}))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		wantAllowed := isSuperAdmin || hasExplicitPerm
		if wantAllowed {
			return rec.Code == http.StatusOK
		}
		return rec.Code == http.StatusForbidden
	}

	if err := quick.Check(prop, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatalf("RBAC permission invariant violated: %v", err)
	}
}
