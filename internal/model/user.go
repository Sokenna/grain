package model

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	*gorm.Model
	Username  string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;size:100;not null"`
	Password  string    `json:"-" gorm:"size:255;not null"`
	Nickname  string    `json:"nickname" gorm:"size:50"`
	Avatar    string    `json:"avatar" gorm:"size:255"`
	Role      string    `json:"role" gorm:"size:50;default:user"` //admin, user, guest
	Phone     string    `json:"phone" gorm:"size:20"`
	Gender    string    `json:"gender" gorm:"size:10"` //male, female
	Birthday  string    `json:"birthday" gorm:"size:20"`
	Address   string    `json:"address" gorm:"size:255"`
	CreatedBy int       `json:"created_by"`
	UpdatedBy int       `json:"updated_by"`
	Status    int       `json:"status" gorm:"default:1"` //1:启用 0:禁用
	LastLogin time.Time `json:"last_login"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// IsActive 是否启用
func (u *User) IsActive() bool {
	return u.Status == 1
}

// IsAdmin 是否是管理员
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsSuperAdmin 是否是超级管理员
func (u *User) IsSuperAdmin() bool {
	return u.Role == "super_admin"
}

// HasRole 检查是否有指定角色
func (u *User) HasRole(roles ...string) bool {
	for _, role := range roles {
		if u.Role == role {
			return true
		}
	}
	return false
}

// Sanitize 清除敏感信息
func (u *User) Sanitize() {
	u.Password = ""
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email" binding:"omitempty,email"`
	Role     string `json:"role"`
	Status   *int   `json:"status"` // 指针类型，区分零值和未传递
}

// UserListRequest 用户列表请求
type UserListRequest struct {
	PageRequest
	Keyword string `form:"keyword" json:"keyword"` // 搜索关键词
	Status  *int   `form:"status" json:"status"`   // 状态筛选
	Role    string `form:"role" json:"role"`       // 角色筛选
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}

// RefreshRequest 刷新请求
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" bind:"required"`
}
