package iam

import "time"

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Nickname     string    `json:"nickname"`
	Status       string    `json:"status"`
	Roles        []string  `json:"roles,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Role struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Permission struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type LoginReq struct {
	Username string `json:"username" validate:"required,min=2,max=64"`
	Password string `json:"password" validate:"required,min=6,max=128"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=128"`
}

type CreateUserReq struct {
	Username string   `json:"username" validate:"required,min=2,max=64"`
	Password string   `json:"password" validate:"required,min=6,max=128"`
	Nickname string   `json:"nickname" validate:"max=64"`
	RoleIDs  []string `json:"role_ids"`
}

type UpdateUserStatusReq struct {
	Status string `json:"status" validate:"required,oneof=active disabled"`
}

type AssignUserRolesReq struct {
	RoleIDs []string `json:"role_ids" validate:"required"`
}

type CreateRoleReq struct {
	Code string `json:"code" validate:"required,min=2,max=64"`
	Name string `json:"name" validate:"required,min=2,max=64"`
}

type AssignRolePermsReq struct {
	PermissionIDs []string `json:"permission_ids" validate:"required"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type MeResp struct {
	User        User     `json:"user"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}
