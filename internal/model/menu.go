package model

import "time"

// Menu 菜单模型
type Menu struct {
	ID         uint       `json:"id" gorm:"primarykey"`
	ParentID   uint       `json:"parent_id" gorm:"default:0;comment:父菜单ID"`
	Name       string     `json:"name" gorm:"size:50;not null;comment:菜单名称"`
	Path       string     `json:"path" gorm:"size:100;comment:路由路径"`
	Component  string     `json:"component" gorm:"size:100;comment:组件路径"`
	Icon       string     `json:"icon" gorm:"size:50;comment:菜单图标"`
	Sort       int        `json:"sort" gorm:"default:0;comment:排序"`
	Type       int        `json:"type" gorm:"default:0;comment:类型 0目录 1菜单 2按钮"`
	Permission string     `json:"permission" gorm:"size:100;comment:权限标识"`
	Status     int        `json:"status" gorm:"default:1;comment:状态 0禁用 1启用"`
	Visible    int        `json:"visible" gorm:"default:1;comment:可见性 0隐藏 1显示"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// 非数据库字段
	Children []Menu `json:"children,omitempty" gorm:"-"`
}

func (Menu) TableName() string {
	return "menus"
}

// Role 角色模型
type Role struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	Name        string     `json:"name" gorm:"size:50;not null;comment:角色名称"`
	Code        string     `json:"code" gorm:"size:50;not null;uniqueIndex;comment:角色编码"`
	Description string     `json:"description" gorm:"size:255;comment:角色描述"`
	Sort        int        `json:"sort" gorm:"default:0;comment:排序"`
	Status      int        `json:"status" gorm:"default:1;comment:状态 0禁用 1启用"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (Role) TableName() string {
	return "roles"
}

// UserRole 用户角色关联
type UserRole struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	UserID    uint      `json:"user_id" gorm:"not null;comment:用户ID"`
	RoleID    uint      `json:"role_id" gorm:"not null;comment:角色ID"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// RoleMenu 角色菜单关联
type RoleMenu struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	RoleID    uint      `json:"role_id" gorm:"not null;comment:角色ID"`
	MenuID    uint      `json:"menu_id" gorm:"not null;comment:菜单ID"`
	CreatedAt time.Time `json:"created_at"`
}

func (RoleMenu) TableName() string {
	return "role_menus"
}

// RouteVO 路由响应对象
type RouteVO struct {
	ID         uint      `json:"id"`
	ParentID   uint      `json:"parent_id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Component  string    `json:"component,omitempty"`
	Icon       string    `json:"icon,omitempty"`
	Sort       int       `json:"sort"`
	Type       int       `json:"type"`
	Permission string    `json:"permission,omitempty"`
	Children   []RouteVO `json:"children,omitempty"`
}
