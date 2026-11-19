package iam

import "context"

type Repository interface {
	CountUsers(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, username, passwordHash, nickname string) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	ListUsers(ctx context.Context, limit, offset int) ([]User, int64, error)
	UpdateUserStatus(ctx context.Context, id, status string) error
	UpdateUserPassword(ctx context.Context, id, passwordHash string) error

	GetUserRoleCodes(ctx context.Context, userID string) ([]string, error)
	GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error)
	ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string) error

	ListRoles(ctx context.Context) ([]Role, error)
	CreateRole(ctx context.Context, code, name string) (Role, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	ReplaceRolePermissions(ctx context.Context, roleID string, permIDs []string) ([]string, error)
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type PermCache interface {
	Get(ctx context.Context, userID string) ([]string, bool, error)
	Set(ctx context.Context, userID string, perms []string) error
	Delete(ctx context.Context, userIDs ...string) error
}
