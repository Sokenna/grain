package handler

import (
	"github.com/gin-gonic/gin"
	"grain/config"
	"grain/internal/model"
	"grain/internal/service"
	"net/http"
	"strconv"
)

type UserHandler struct {
	userService service.UserService
}

/*type UserHandler struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
}
*/
/*
	func NewUserHandler(db *gorm.DB, jwtManager *auth.JWTManager) *UserHandler {
		return &UserHandler{
			db:         db,
			jwtManager: jwtManager,
		}
	}
*/
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	user, accessToken, refreshToken, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	Success(c, model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(config.Config.Auth.JWT.ExpireTime.Seconds()),
		User:         *user,
	})
}

// GetUserInfo 获取用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		Error(c, http.StatusUnauthorized, "未找到用户信息")
		return
	}

	user, err := h.userService.GetUserInfo(userID.(uint))
	if err != nil {
		Error(c, http.StatusNotFound, err.Error())
		return
	}
	Success(c, user)
}

// GetUserList 获取用户列表（需要管理员权限）
func (h *UserHandler) GetUserList(c *gin.Context) {
	var req model.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	users, total, err := h.userService.GetUserList(&req)
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Success(c, model.PageResponse{
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
		List:  users,
	})
}

// CreateUser 创建用户（管理员）
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	user, err := h.userService.CreateUser(&req)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	Success(c, user)
}

// UpdateUser 更新用户（管理员）
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.userService.UpdateUser(uint(id), &req); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	SuccessWithMessage(c, "更新成功", nil)
}

// DeleteUser 删除用户（管理员）
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	SuccessWithMessage(c, "删除成功", nil)
}

// UpdateUserStatus 更新用户状态（管理员）
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req struct {
		Status int `json:"status" binding:"required,oneof=0 1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.userService.UpdateUserStatus(uint(id), req.Status); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	SuccessWithMessage(c, "状态更新成功", nil)
}
