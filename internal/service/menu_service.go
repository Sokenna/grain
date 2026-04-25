package service

import (
	"sort"

	"grain/internal/model"
	"grain/internal/repository"
)

type MenuService interface {
	GetUserRoutes(userID uint) ([]model.RouteVO, error)
}

type menuService struct {
	menuRepo repository.MenuRepository
}

func NewMenuService(menuRepo repository.MenuRepository) MenuService {
	return &menuService{
		menuRepo: menuRepo,
	}
}

// GetUserRoutes 获取用户路由
func (s *menuService) GetUserRoutes(userID uint) ([]model.RouteVO, error) {
	menus, err := s.menuRepo.GetUserMenus(userID)
	if err != nil {
		return nil, err
	}

	// 构建路由树
	return s.buildRouteTree(menus, 0), nil
}

// buildRouteTree 构建路由树
func (s *menuService) buildRouteTree(menus []model.Menu, parentID uint) []model.RouteVO {
	var routes []model.RouteVO

	for _, menu := range menus {
		if menu.ParentID == parentID {
			route := model.RouteVO{
				ID:         menu.ID,
				ParentID:   menu.ParentID,
				Name:       menu.Name,
				Path:       menu.Path,
				Icon:       menu.Icon,
				Sort:       menu.Sort,
				Type:       menu.Type,
				Permission: menu.Permission,
			}

			// 设置组件路径
			if menu.Type == 1 && menu.Component != "" {
				route.Component = menu.Component
			}

			// 递归构建子路由
			children := s.buildRouteTree(menus, menu.ID)
			if len(children) > 0 {
				route.Children = children
			}

			routes = append(routes, route)
		}
	}

	// 按排序值排序
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Sort < routes[j].Sort
	})

	return routes
}
