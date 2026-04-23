package service

import (
	"errors"
	"grain/internal/auth"
	"grain/internal/model"
	"grain/internal/repository"
	"time"
)

// UserService 用户业务逻辑接口
type UserService interface {
	Login(username, password string) (*model.User, string, string, error)
	GetUserInfo(userID uint) (*model.User, error)
	GetUserList(req *model.UserListRequest) ([]model.User, int64, error)
	CreateUser(req *model.UserCreateRequest) (*model.User, error)
	UpdateUser(userID uint, req *model.UserUpdateRequest) error
	DeleteUser(userID uint) error
	UpdateUserStatus(userID uint, status int) error
}

// userService 用户业务逻辑实现
type userService struct {
	userRepo   repository.UserRepository
	jwtManager *auth.JWTManager
}

// NewUserService 创建用户服务
func NewUserService(userRepo repository.UserRepository, jwtManager *auth.JWTManager) UserService {
	return &userService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// Login 用户登录
func (s *userService) Login(username, password string) (*model.User, string, string, error) {
	// 1. 查询用户
	user, err := s.userRepo.GetByUsernameOrEmail(username, username)
	if err != nil {
		return nil, "", "", errors.New("数据库查询失败")
	}
	if user == nil {
		return nil, "", "", errors.New("用户名或密码错误")
	}
	// 2. 验证密码
	if !auth.CheckPassword(password, user.Password) {
		return nil, "", "", errors.New("用户名或密码错误")
	}

	// 3. 检查用户状态
	if !user.IsActive() {
		return nil, "", "", errors.New("账号已被禁用")
	}

	// 4. 生成 token
	accessToken, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		return nil, "", "", errors.New("生成token失败")
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user)
	if err != nil {
		return nil, "", "", errors.New("生成refresh token失败")
	}

	// 5. 更新最后登录时间（异步处理，不影响主流程）
	go func() {
		now := time.Now()
		_ = s.userRepo.UpdateLastLogin(user.ID, &now)
	}()

	// 6. 清除敏感信息
	user.Sanitize()

	return user, accessToken, refreshToken, nil
}

// GetUserInfo 获取用户信息
func (s *userService) GetUserInfo(userID uint) (*model.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("数据库查询失败")
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	user.Sanitize()
	return user, nil
}

// GetUserList 获取用户列表
func (s *userService) GetUserList(req *model.UserListRequest) ([]model.User, int64, error) {
	// 参数校验和默认值设置
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.Size > 100 {
		req.Size = 100
	}

	users, total, err := s.userRepo.GetList(req)
	if err != nil {
		return nil, 0, errors.New("查询用户列表失败")
	}

	// 清除敏感信息
	for i := range users {
		users[i].Sanitize()
	}

	return users, total, nil
}

// CreateUser 创建用户
func (s *userService) CreateUser(req *model.UserCreateRequest) (*model.User, error) {
	// 1. 检查用户名是否已存在
	exists, err := s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, errors.New("检查用户名失败")
	}
	if exists {
		return nil, errors.New("用户名已存在")
	}

	// 2. 检查邮箱是否已存在
	exists, err = s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, errors.New("检查邮箱失败")
	}
	if exists {
		return nil, errors.New("邮箱已存在")
	}

	// 3. 加密密码
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	// 4. 创建用户
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Role:     req.Role,
		Status:   1, // 默认启用
	}

	if user.Role == "" {
		user.Role = "user"
	}
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("创建用户失败")
	}

	user.Sanitize()
	return user, nil
}

// UpdateUser 更新用户信息
func (s *userService) UpdateUser(userID uint, req *model.UserUpdateRequest) error {
	// 1. 检查用户是否存在
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("数据库查询失败")
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	// 2. 更新字段
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		// 检查新邮箱是否已被其他用户使用
		exists, err := s.userRepo.ExistsByEmail(req.Email, userID)
		if err != nil {
			return errors.New("检查邮箱失败")
		}
		if exists {
			return errors.New("邮箱已被其他用户使用")
		}
		user.Email = req.Email
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	// 3. 保存更新
	return s.userRepo.Update(user)
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(userID uint) error {
	// 检查用户是否存在
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("数据库查询失败")
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	return s.userRepo.Delete(userID)
}

// UpdateUserStatus 更新用户状态
func (s *userService) UpdateUserStatus(userID uint, status int) error {
	// 检查用户是否存在
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("数据库查询失败")
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	return s.userRepo.UpdateStatus(userID, status)
}
