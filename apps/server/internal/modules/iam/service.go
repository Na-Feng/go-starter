package iam

import (
	"context"
	"errors"
	"slices"
	"time"

	"golang.org/x/crypto/bcrypt"

	"go-starter/internal/kernel/audit"
	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
)

type Service struct {
	repo       Repository
	tx         TxRunner
	jwtMgr     *auth.Manager
	blacklist  auth.TokenBlacklist
	permCache  PermCache
	audit      audit.Recorder
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(
	repo Repository,
	tx TxRunner,
	jwtMgr *auth.Manager,
	blacklist auth.TokenBlacklist,
	permCache PermCache,
	auditRec audit.Recorder,
	accessTTL, refreshTTL time.Duration,
) *Service {
	return &Service{
		repo:       repo,
		tx:         tx,
		jwtMgr:     jwtMgr,
		blacklist:  blacklist,
		permCache:  permCache,
		audit:      auditRec,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *Service) EnsureSeedAdmin(ctx context.Context) error {
	count, err := s.repo.CountUsers(ctx)
	if err != nil || count > 0 {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.runInTx(ctx, func(txCtx context.Context) error {
		u, err := s.repo.CreateUser(txCtx, "admin", string(hash), "系统管理员")
		if err != nil {
			return err
		}
		return s.repo.ReplaceUserRoles(txCtx, u.ID, []string{"role_super_admin"})
	})
}

func (s *Service) Login(ctx context.Context, req LoginReq) (TokenPair, error) {
	u, err := s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		var appErr *errs.AppError
		if errors.As(err, &appErr) && appErr.Code == 40401 {
			return TokenPair{}, errs.Unauthorized("invalid username or password")
		}
		return TokenPair{}, err
	}
	if u.Status != "active" {
		return TokenPair{}, errs.Forbidden("account is disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return TokenPair{}, errs.Unauthorized("invalid username or password")
	}
	return s.issueTokens(ctx, u.ID)
}

func (s *Service) Refresh(ctx context.Context, req RefreshReq) (TokenPair, error) {
	claims, err := s.jwtMgr.Verify(req.RefreshToken)
	if err != nil || claims.TokenType != auth.TokenTypeRefresh {
		return TokenPair{}, errs.Unauthorized("invalid refresh token")
	}
	if s.blacklist != nil {
		revoked, err := s.blacklist.IsRevoked(ctx, claims.ID)
		if err != nil {
			return TokenPair{}, errs.Internal(err)
		}
		if revoked {
			return TokenPair{}, errs.Unauthorized("refresh token revoked")
		}
		_ = s.blacklist.Revoke(ctx, claims.ID, time.Until(claims.ExpiresAt.Time))
	}
	u, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return TokenPair{}, err
	}
	if u.Status != "active" {
		return TokenPair{}, errs.Forbidden("account is disabled")
	}
	return s.issueTokens(ctx, u.ID)
}

func (s *Service) Logout(ctx context.Context, claims auth.UserClaims) error {
	if s.blacklist == nil || claims.ExpiresAt == nil {
		return nil
	}
	return s.blacklist.Revoke(ctx, claims.ID, time.Until(claims.ExpiresAt.Time))
}

func (s *Service) Me(ctx context.Context, userID string) (MeResp, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return MeResp{}, err
	}
	roles, err := s.repo.GetUserRoleCodes(ctx, userID)
	if err != nil {
		return MeResp{}, err
	}
	perms, err := s.loadPermissions(ctx, userID)
	if err != nil {
		return MeResp{}, err
	}
	return MeResp{User: u, Roles: roles, Permissions: perms}, nil
}

func (s *Service) ChangePassword(ctx context.Context, claims auth.UserClaims, req ChangePasswordReq) error {
	u, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)); err != nil {
		return errs.BadRequest("old password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errs.Internal(err)
	}
	if err := s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateUserPassword(txCtx, u.ID, string(hash)); err != nil {
			return err
		}
		return s.recordAudit(txCtx, claims.UserID, "user.change_password", "user", u.ID, nil)
	}); err != nil {
		return err
	}
	return s.Logout(ctx, claims)
}

func (s *Service) HasPermission(ctx context.Context, userID, permCode string) (bool, error) {
	perms, err := s.loadPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	return slices.Contains(perms, permCode), nil
}

func (s *Service) ListUsers(ctx context.Context, pq httpx.PageQuery) (httpx.PageResult[User], error) {
	users, total, err := s.repo.ListUsers(ctx, pq.Limit(), pq.Offset())
	if err != nil {
		return httpx.PageResult[User]{}, err
	}
	return httpx.PageResult[User]{Items: users, Total: total, Page: pq.Page, PageSize: pq.PageSize}, nil
}

func (s *Service) CreateUser(ctx context.Context, actorID string, req CreateUserReq) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, errs.Internal(err)
	}
	var created User
	err = s.runInTx(ctx, func(txCtx context.Context) error {
		u, err := s.repo.CreateUser(txCtx, req.Username, string(hash), req.Nickname)
		if err != nil {
			return err
		}
		if len(req.RoleIDs) > 0 {
			if err := s.repo.ReplaceUserRoles(txCtx, u.ID, req.RoleIDs); err != nil {
				return err
			}
		}
		created = u
		return s.recordAudit(txCtx, actorID, "user.create", "user", u.ID, map[string]any{"username": u.Username, "role_ids": req.RoleIDs})
	})
	return created, err
}

func (s *Service) UpdateUserStatus(ctx context.Context, actorID, userID string, req UpdateUserStatusReq) error {
	return s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateUserStatus(txCtx, userID, req.Status); err != nil {
			return err
		}
		return s.recordAudit(txCtx, actorID, "user.update_status", "user", userID, map[string]string{"status": req.Status})
	})
}

func (s *Service) AssignUserRoles(ctx context.Context, actorID, userID string, req AssignUserRolesReq) error {
	if err := s.runInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.ReplaceUserRoles(txCtx, userID, req.RoleIDs); err != nil {
			return err
		}
		return s.recordAudit(txCtx, actorID, "user.assign_roles", "user", userID, map[string]any{"role_ids": req.RoleIDs})
	}); err != nil {
		return err
	}
	if s.permCache != nil {
		_ = s.permCache.Delete(ctx, userID)
	}
	return nil
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) CreateRole(ctx context.Context, actorID string, req CreateRoleReq) (Role, error) {
	var role Role
	err := s.runInTx(ctx, func(txCtx context.Context) error {
		var err error
		role, err = s.repo.CreateRole(txCtx, req.Code, req.Name)
		if err != nil {
			return err
		}
		return s.recordAudit(txCtx, actorID, "role.create", "role", role.ID, map[string]string{"code": role.Code})
	})
	return role, err
}

func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *Service) AssignRolePermissions(ctx context.Context, actorID, roleID string, req AssignRolePermsReq) error {
	var affected []string
	if err := s.runInTx(ctx, func(txCtx context.Context) error {
		var err error
		affected, err = s.repo.ReplaceRolePermissions(txCtx, roleID, req.PermissionIDs)
		if err != nil {
			return err
		}
		return s.recordAudit(txCtx, actorID, "role.assign_permissions", "role", roleID, map[string]any{"permission_ids": req.PermissionIDs})
	}); err != nil {
		return err
	}
	if s.permCache != nil && len(affected) > 0 {
		_ = s.permCache.Delete(ctx, affected...)
	}
	return nil
}

func (s *Service) ListAuditLogs(ctx context.Context, pq httpx.PageQuery) (httpx.PageResult[audit.Entry], error) {
	if s.audit == nil {
		return httpx.PageResult[audit.Entry]{Items: []audit.Entry{}, Page: pq.Page, PageSize: pq.PageSize}, nil
	}
	return s.audit.List(ctx, pq)
}

func (s *Service) recordAudit(ctx context.Context, actorID, action, targetType, targetID string, detail any) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Record(ctx, actorID, action, targetType, targetID, detail)
}

func (s *Service) issueTokens(ctx context.Context, userID string) (TokenPair, error) {
	roles, err := s.repo.GetUserRoleCodes(ctx, userID)
	if err != nil {
		return TokenPair{}, err
	}
	access, err := s.jwtMgr.Sign(userID, roles, auth.TokenTypeAccess, s.accessTTL)
	if err != nil {
		return TokenPair{}, errs.Internal(err)
	}
	refresh, err := s.jwtMgr.Sign(userID, roles, auth.TokenTypeRefresh, s.refreshTTL)
	if err != nil {
		return TokenPair{}, errs.Internal(err)
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) loadPermissions(ctx context.Context, userID string) ([]string, error) {
	if s.permCache != nil {
		if perms, ok, err := s.permCache.Get(ctx, userID); err == nil && ok {
			return perms, nil
		}
	}
	perms, err := s.repo.GetUserPermissionCodes(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s.permCache != nil {
		_ = s.permCache.Set(ctx, userID, perms)
	}
	return perms, nil
}

func (s *Service) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}
	return s.tx.WithinTx(ctx, fn)
}
