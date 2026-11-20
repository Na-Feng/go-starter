package iam_test

import (
	"context"
	"errors"
	"math/rand"
	"slices"
	"sync"
	"testing"
	"time"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/modules/iam"
)

type memPermCache struct {
	mu    sync.RWMutex
	items map[string][]string
}

func (c *memPermCache) Get(_ context.Context, userID string) ([]string, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[userID]
	if !ok {
		return nil, false, nil
	}
	return slices.Clone(v), true, nil
}

func (c *memPermCache) Set(_ context.Context, userID string, perms []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[userID] = slices.Clone(perms)
	return nil
}

func (c *memPermCache) Delete(_ context.Context, userIDs ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range userIDs {
		delete(c.items, id)
	}
	return nil
}

func TestIAMStatefulModelInvariants(t *testing.T) {
	ctx := context.Background()
	jwtMgr := auth.NewManager("stateful-invariant-secret")
	bl := &memBlacklist{ids: make(map[string]bool)}
	cache := &memPermCache{items: make(map[string][]string)}
	repo := newMemIAMRepo()
	repo.roles["role_editor"] = iam.Role{ID: "role_editor", Code: "editor", Name: "内容编辑"}
	repo.rolePerms["role_editor"] = []string{"sample:write"}

	svc := iam.NewService(repo, nil, jwtMgr, bl, cache, nil, 15*time.Minute, 24*time.Hour)

	password := "pass123456"
	user, err := svc.CreateUser(ctx, "actor_admin", iam.CreateUserReq{
		Username: "alice",
		Password: password,
		Nickname: "Alice",
		RoleIDs:  []string{"role_operator"},
	})
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	expectAppCode := func(err error, wantHTTP int) {
		t.Helper()
		if err == nil {
			t.Fatalf("expected HTTP %d error, got nil", wantHTTP)
		}
		var appErr *errs.AppError
		if !errors.As(err, &appErr) || appErr.HTTPStatus != wantHTTP {
			t.Fatalf("expected HTTP %d, got %v", wantHTTP, err)
		}
	}

	status := "active"
	currentRoles := []string{"role_operator"}
	var lastPair iam.TokenPair

	rng := rand.New(rand.NewSource(20261119))
	for step := 0; step < 60; step++ {
		switch rng.Intn(5) {
		case 0:
			pair, err := svc.Login(ctx, iam.LoginReq{Username: "alice", Password: password})
			if status == "disabled" {
				expectAppCode(err, 403)
				continue
			}
			if err != nil {
				t.Fatalf("step %d: active user login failed: %v", step, err)
			}
			lastPair = pair

		case 1:
			if lastPair.RefreshToken == "" {
				continue
			}
			usedRefresh := lastPair.RefreshToken
			newPair, err := svc.Refresh(ctx, iam.RefreshReq{RefreshToken: usedRefresh})
			if status == "disabled" {
				expectAppCode(err, 403)
				lastPair = iam.TokenPair{}
				continue
			}
			if err != nil {
				t.Fatalf("step %d: refresh failed: %v", step, err)
			}
			_, replayErr := svc.Refresh(ctx, iam.RefreshReq{RefreshToken: usedRefresh})
			expectAppCode(replayErr, 401)
			lastPair = newPair

		case 2:
			if status == "active" {
				status = "disabled"
			} else {
				status = "active"
			}
			if err := svc.UpdateUserStatus(ctx, "actor_admin", user.ID, iam.UpdateUserStatusReq{Status: status}); err != nil {
				t.Fatalf("step %d: update status failed: %v", step, err)
			}

		case 3:
			if slices.Equal(currentRoles, []string{"role_operator"}) {
				currentRoles = []string{"role_editor"}
			} else if slices.Equal(currentRoles, []string{"role_editor"}) {
				currentRoles = []string{"role_operator", "role_editor"}
			} else {
				currentRoles = []string{"role_operator"}
			}
			if err := svc.AssignUserRoles(ctx, "actor_admin", user.ID, iam.AssignUserRolesReq{RoleIDs: currentRoles}); err != nil {
				t.Fatalf("step %d: assign roles failed: %v", step, err)
			}

		case 4:
			if status != "active" || lastPair.AccessToken == "" {
				continue
			}
			claims, err := jwtMgr.Verify(lastPair.AccessToken)
			if err != nil {
				t.Fatalf("step %d: verify token failed: %v", step, err)
			}
			nextPwd := password + "x"
			if len(nextPwd) > 20 {
				nextPwd = "pass123456"
			}
			if err := svc.ChangePassword(ctx, claims, iam.ChangePasswordReq{
				OldPassword: password,
				NewPassword: nextPwd,
			}); err != nil {
				t.Fatalf("step %d: change password failed: %v", step, err)
			}
			password = nextPwd
			revoked, _ := bl.IsRevoked(ctx, claims.ID)
			if !revoked {
				t.Fatalf("step %d: invariant broken: access token not revoked after password change", step)
			}
			lastPair = iam.TokenPair{}
		}

		wantReadUser := slices.Contains(currentRoles, "role_operator")
		wantWriteSample := slices.Contains(currentRoles, "role_editor")

		gotReadUser, err := svc.HasPermission(ctx, user.ID, "iam:user:read")
		if err != nil || gotReadUser != wantReadUser {
			t.Fatalf("step %d: cache/perm invariant broken for iam:user:read: got=%v want=%v err=%v",
				step, gotReadUser, wantReadUser, err)
		}
		gotWriteSample, err := svc.HasPermission(ctx, user.ID, "sample:write")
		if err != nil || gotWriteSample != wantWriteSample {
			t.Fatalf("step %d: cache/perm invariant broken for sample:write: got=%v want=%v err=%v",
				step, gotWriteSample, wantWriteSample, err)
		}
	}
}
