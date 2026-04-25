package repository

import (
	"gorm.io/gorm"

	"grain/internal/model"
)

type MenuRepository interface {
	GetUserMenus(userID uint) ([]model.Menu, error)
	GetRoleMenus(roleID uint) ([]model.Menu, error)
	GetAllMenus() ([]model.Menu, error)
	GetUserRoles(userID uint) ([]model.Role, error)
}

type menuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db *gorm.DB) MenuRepository {
	return &menuRepository{db: db}
}

// GetUserMenus 获取用户菜单
func (r *menuRepository) GetUserMenus(userID uint) ([]model.Menu, error) {
	var menus []model.Menu

	// 查询用户角色
	var userRoles []model.UserRole
	if err := r.db.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	if len(userRoles) == 0 {
		return menus, nil
	}

	// 获取角色ID列表
	roleIDs := make([]uint, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// 查询角色关联的菜单ID
	var roleMenus []model.RoleMenu
	if err := r.db.Where("role_id IN ?", roleIDs).Find(&roleMenus).Error; err != nil {
		return nil, err
	}

	if len(roleMenus) == 0 {
		return menus, nil
	}

	// 获取菜单ID列表
	menuIDs := make([]uint, len(roleMenus))
	for i, rm := range roleMenus {
		menuIDs[i] = rm.MenuID
	}

	// 查询菜单
	if err := r.db.Where("id IN ? AND status = 1 AND visible = 1 AND type IN (0, 1)", menuIDs).
		Order("sort ASC, id ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return menus, nil
}

// GetRoleMenus 获取角色菜单
func (r *menuRepository) GetRoleMenus(roleID uint) ([]model.Menu, error) {
	var menus []model.Menu

	var roleMenus []model.RoleMenu
	if err := r.db.Where("role_id = ?", roleID).Find(&roleMenus).Error; err != nil {
		return nil, err
	}

	if len(roleMenus) == 0 {
		return menus, nil
	}

	menuIDs := make([]uint, len(roleMenus))
	for i, rm := range roleMenus {
		menuIDs[i] = rm.MenuID
	}

	if err := r.db.Where("id IN ? AND status = 1", menuIDs).
		Order("sort ASC, id ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return menus, nil
}

// GetAllMenus 获取所有菜单
func (r *menuRepository) GetAllMenus() ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Where("type IN (0, 1) AND status = 1").
		Order("sort ASC, id ASC").
		Find(&menus).Error
	return menus, err
}

// GetUserRoles 获取用户角色
func (r *menuRepository) GetUserRoles(userID uint) ([]model.Role, error) {
	var roles []model.Role

	var userRoles []model.UserRole
	if err := r.db.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	if len(userRoles) == 0 {
		return roles, nil
	}

	roleIDs := make([]uint, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	err := r.db.Where("id IN ? AND status = 1", roleIDs).Find(&roles).Error
	return roles, err
}
