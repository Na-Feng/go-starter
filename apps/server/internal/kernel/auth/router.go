package auth

import (
	"log/slog"
	"net/http"
)

// Router 封装受控路由注册入口，消除业务层漏包鉴权中间件导致接口裸奔的风险。
type Router struct {
	mux         *http.ServeMux
	jwtMgr      *Manager
	blacklist   TokenBlacklist
	permChecker PermissionChecker
	log         *slog.Logger
}

func NewRouter(
	mux *http.ServeMux,
	jwtMgr *Manager,
	blacklist TokenBlacklist,
	permChecker PermissionChecker,
	log *slog.Logger,
) *Router {
	return &Router{
		mux:         mux,
		jwtMgr:      jwtMgr,
		blacklist:   blacklist,
		permChecker: permChecker,
		log:         log,
	}
}

// Public 显式注册无需登录即可访问的公开路由（如登录、刷新令牌、公开查询）。
func (r *Router) Public(pattern string, h http.HandlerFunc) {
	r.mux.HandleFunc(pattern, h)
}

// Authenticated 注册需有效 Access Token 登录态的路由。
func (r *Router) Authenticated(pattern string, h http.HandlerFunc) {
	mw := RequireAuth(r.jwtMgr, r.blacklist, r.log)
	r.mux.Handle(pattern, mw(h))
}

// Protected 注册需有效登录态且具备指定 RBAC 权限码的受保护路由。
func (r *Router) Protected(pattern string, permCode string, h http.HandlerFunc) {
	authMW := RequireAuth(r.jwtMgr, r.blacklist, r.log)
	permMW := RequirePermission(permCode, r.permChecker, r.log)
	r.mux.Handle(pattern, authMW(permMW(h)))
}
