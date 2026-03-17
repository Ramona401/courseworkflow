package services

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/google/uuid"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 用户管理服务错误常量 ====================

var (
	ErrUsernameRequired    = errors.New("用户名不能为空")
	ErrDisplayNameRequired = errors.New("显示名称不能为空")
	ErrPasswordTooShort    = errors.New("密码长度不能少于6位")
	ErrInvalidRole         = errors.New("无效的角色，可选值：admin/operator/viewer")
	ErrInvalidStatus       = errors.New("无效的状态，可选值：active/disabled")
	ErrUsernameExists      = errors.New("用户名已存在")
	ErrCannotDisableSelf   = errors.New("不能禁用自己的账户")
	ErrCannotChangeOwnRole = errors.New("不能修改自己的角色")
	ErrUserNotFound        = errors.New("用户不存在")
)

// UserService 用户管理服务
type UserService struct{}

// NewUserService 创建用户管理服务实例
func NewUserService() *UserService {
	return &UserService{}
}

// ==================== 用户列表 ====================

// ListUsers 获取所有用户列表（转换为 UserInfo）
func (s *UserService) ListUsers(ctx context.Context) (*models.UserListResponse, error) {
	users, err := repository.ListUsers(ctx)
	if err != nil {
		log.Printf("查询用户列表失败: %v", err)
		return nil, err
	}

	// 转换为 UserInfo（过滤敏感信息）
	userInfos := make([]*models.UserInfo, 0, len(users))
	for _, u := range users {
		userInfos = append(userInfos, u.ToUserInfo())
	}

	return &models.UserListResponse{
		Users: userInfos,
		Total: len(userInfos),
	}, nil
}

// ==================== 创建用户 ====================

// CreateUser 创建新用户（含参数校验）
func (s *UserService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserInfo, error) {
	// 1. 参数校验
	req.Username = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Username == "" {
		return nil, ErrUsernameRequired
	}
	if req.DisplayName == "" {
		return nil, ErrDisplayNameRequired
	}
	if len(req.Password) < 6 {
		return nil, ErrPasswordTooShort
	}
	if !models.IsValidRole(req.Role) {
		return nil, ErrInvalidRole
	}

	// 2. 检查用户名唯一性
	exists, err := repository.CheckUsernameExists(ctx, req.Username)
	if err != nil {
		log.Printf("检查用户名失败: %v", err)
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// 3. 生成密码哈希
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("生成密码哈希失败: %v", err)
		return nil, err
	}

	// 4. 构造用户对象
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Status:       models.StatusActive, // 新建用户默认启用
	}

	// 5. 写入数据库
	if err := repository.CreateUser(ctx, user); err != nil {
		log.Printf("创建用户失败: %v", err)
		return nil, err
	}

	log.Printf("创建用户成功: %s (%s, 角色: %s)", user.Username, user.ID, user.Role)

	// 6. 重新查询完整用户信息（含created_at等数据库生成字段）
	created, err := repository.FindUserByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return created.ToUserInfo(), nil
}

// ==================== 编辑用户 ====================

// UpdateUser 更新用户基本信息（显示名+角色）
func (s *UserService) UpdateUser(ctx context.Context, userID string, currentUserID string, req *models.UpdateUserRequest) (*models.UserInfo, error) {
	// 1. 参数校验
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.DisplayName == "" {
		return nil, ErrDisplayNameRequired
	}
	if !models.IsValidRole(req.Role) {
		return nil, ErrInvalidRole
	}

	// 2. 不允许修改自己的角色（防止admin把自己降级）
	if userID == currentUserID {
		// 先查当前角色
		existing, err := repository.FindUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if existing.Role != req.Role {
			return nil, ErrCannotChangeOwnRole
		}
	}

	// 3. 更新数据库
	if err := repository.UpdateUser(ctx, userID, req.DisplayName, req.Role); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		log.Printf("更新用户失败: %v", err)
		return nil, err
	}

	// 4. 返回更新后的用户信息
	updated, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	log.Printf("更新用户成功: %s (角色: %s)", updated.Username, updated.Role)
	return updated.ToUserInfo(), nil
}

// ==================== 重置密码 ====================

// ResetPassword 重置用户密码（仅admin可调用）
func (s *UserService) ResetPassword(ctx context.Context, userID string, req *models.ResetPasswordRequest) error {
	// 1. 参数校验
	if len(req.NewPassword) < 6 {
		return ErrPasswordTooShort
	}

	// 2. 确认用户存在
	user, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 3. 生成新密码哈希
	passwordHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("生成密码哈希失败: %v", err)
		return err
	}

	// 4. 更新密码
	if err := repository.UpdatePassword(ctx, userID, passwordHash); err != nil {
		log.Printf("重置密码失败: %v", err)
		return err
	}

	log.Printf("重置密码成功: %s (%s)", user.Username, userID)
	return nil
}

// ==================== 启用/禁用用户 ====================

// UpdateStatus 更新用户状态
func (s *UserService) UpdateStatus(ctx context.Context, userID string, currentUserID string, req *models.UpdateStatusRequest) error {
	// 1. 参数校验
	if !models.IsValidStatus(req.Status) {
		return ErrInvalidStatus
	}

	// 2. 不允许禁用自己
	if userID == currentUserID && req.Status == models.StatusDisabled {
		return ErrCannotDisableSelf
	}

	// 3. 确认用户存在
	user, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 4. 更新状态
	if err := repository.UpdateStatus(ctx, userID, req.Status); err != nil {
		log.Printf("更新用户状态失败: %v", err)
		return err
	}

	log.Printf("更新用户状态成功: %s → %s", user.Username, req.Status)
	return nil
}

// ==================== 课程分配 ====================

// GetAssignments 获取用户课程分配列表
func (s *UserService) GetAssignments(ctx context.Context, userID string) ([]*models.CourseAssignment, error) {
	// 确认用户存在
	_, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return repository.GetUserAssignments(ctx, userID)
}

// UpdateAssignments 全量替换用户课程分配
func (s *UserService) UpdateAssignments(ctx context.Context, userID string, adminID string, req *models.UpdateAssignmentsRequest) ([]*models.CourseAssignment, error) {
	// 1. 确认用户存在
	_, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 2. 执行全量替换（事务）
	if err := repository.ReplaceUserAssignments(ctx, userID, req.CourseCodes, adminID); err != nil {
		log.Printf("更新课程分配失败: %v", err)
		return nil, err
	}

	log.Printf("更新课程分配成功: 用户 %s, 课程数 %d", userID, len(req.CourseCodes))

	// 3. 返回最新的分配列表
	return repository.GetUserAssignments(ctx, userID)
}
