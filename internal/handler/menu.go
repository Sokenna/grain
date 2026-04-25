package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"grain/internal/service"
)

type MenuHandler struct {
	menuService service.MenuService
}

func NewMenuHandler(menuService service.MenuService) *MenuHandler {
	return &MenuHandler{
		menuService: menuService,
	}
}

// GetRoutes 获取用户路由
func (h *MenuHandler) GetRoutes(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		Error(c, http.StatusUnauthorized, "未找到用户信息")
		return
	}

	routes, err := h.menuService.GetUserRoutes(userID.(uint))
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取菜单失败")
		return
	}

	Success(c, routes)
}
