package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"

	"grain/config"
	"grain/internal/model"
	"grain/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

// NewUserHandler 创建用户处理句柄
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

// ChangePassword 修改密码
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取当前用户ID（从中间件设置的上下文）
	userID, exists := c.Get("user_id")
	if !exists {
		Error(c, http.StatusUnauthorized, "未找到用户信息")
		return
	}

	// 验证新密码格式
	if len(req.NewPassword) < 6 {
		Error(c, http.StatusBadRequest, "新密码长度至少为6位")
		return
	}
	if len(req.NewPassword) > 20 {
		Error(c, http.StatusBadRequest, "新密码长度不能超过20位")
		return
	}

	err := h.userService.ChangePassword(userID.(uint), req.OldPassword, req.NewPassword)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	SuccessWithMessage(c, "密码修改成功，请重新登录", nil)
}
