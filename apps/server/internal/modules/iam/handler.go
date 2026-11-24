package iam

import (
	"log/slog"
	"net/http"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
)

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Register(r *auth.Router) {
	r.Public("POST /api/v1/auth/login", h.login)
	r.Public("POST /api/v1/auth/refresh", h.refresh)
	r.Authenticated("POST /api/v1/auth/logout", h.logout)
	r.Authenticated("GET /api/v1/auth/me", h.me)
	r.Authenticated("POST /api/v1/auth/change-password", h.changePassword)

	r.Protected("GET /api/v1/iam/users", "iam:user:read", h.listUsers)
	r.Protected("POST /api/v1/iam/users", "iam:user:write", h.createUser)
	r.Protected("PATCH /api/v1/iam/users/{id}/status", "iam:user:write", h.updateUserStatus)
	r.Protected("PUT /api/v1/iam/users/{id}/roles", "iam:user:write", h.assignUserRoles)

	r.Protected("GET /api/v1/iam/roles", "iam:role:read", h.listRoles)
	r.Protected("POST /api/v1/iam/roles", "iam:role:write", h.createRole)
	r.Protected("PUT /api/v1/iam/roles/{id}/permissions", "iam:role:write", h.assignRolePerms)
	r.Protected("GET /api/v1/iam/permissions", "iam:role:read", h.listPermissions)
	r.Protected("GET /api/v1/iam/audit-logs", "iam:audit:read", h.listAuditLogs)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	pair, err := h.svc.Login(r.Context(), req)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, pair)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	pair, err := h.svc.Refresh(r.Context(), req)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, pair)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.Fail(w, r, h.log, errs.Unauthorized("unauthenticated"))
		return
	}
	if err := h.svc.Logout(r.Context(), claims); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, map[string]string{"status": "logged_out"})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.UserFrom(r.Context())
	resp, err := h.svc.Me(r.Context(), claims.UserID)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, resp)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	claims, _ := auth.UserFrom(r.Context())
	if err := h.svc.ChangePassword(r.Context(), claims, req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, map[string]string{"status": "password_updated"})
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ListUsers(r.Context(), httpx.ParsePageQuery(r))
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, res)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	claims, _ := auth.UserFrom(r.Context())
	u, err := h.svc.CreateUser(r.Context(), claims.UserID, req)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.Created(w, r, u)
}

func (h *Handler) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserStatusReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	claims, _ := auth.UserFrom(r.Context())
	if err := h.svc.UpdateUserStatus(r.Context(), claims.UserID, r.PathValue("id"), req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, map[string]string{"status": req.Status})
}

func (h *Handler) assignUserRoles(w http.ResponseWriter, r *http.Request) {
	var req AssignUserRolesReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	claims, _ := auth.UserFrom(r.Context())
	if err := h.svc.AssignUserRoles(r.Context(), claims.UserID, r.PathValue("id"), req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, map[string]string{"status": "roles_assigned"})
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, roles)
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	claims, _ := auth.UserFrom(r.Context())
	role, err := h.svc.CreateRole(r.Context(), claims.UserID, req)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.Created(w, r, role)
}

func (h *Handler) assignRolePerms(w http.ResponseWriter, r *http.Request) {
	var req AssignRolePermsReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	claims, _ := auth.UserFrom(r.Context())
	if err := h.svc.AssignRolePermissions(r.Context(), claims.UserID, r.PathValue("id"), req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, map[string]string{"status": "permissions_assigned"})
}

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, perms)
}

func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ListAuditLogs(r.Context(), httpx.ParsePageQuery(r))
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, res)
}
