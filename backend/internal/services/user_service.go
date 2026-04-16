package services

// 用户管理业务逻辑层
// Phase8日志升级：
//   - 数据库查询/写入失败 → ERROR（系统级错误，需要处理）
//   - 业务成功操作（创建/更新/重置密码） → INFO（含关键字段便于审计）
//   - 密码哈希失败 → ERROR（加密库错误，极少发生）
//
// v110修改：
//   - CreateUser 支持写入 SchoolID
//   - UpdateUser 支持更新/保留 SchoolID（兼容旧调用方）
//   - 保持原有接口行为不变，避免影响既有模块

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 用户管理服务错误常量 ====================

var (
	ErrUsernameRequired    = errors.New("用户名不能为空")
	ErrDisplayNameRequired = errors.New("显示名称不能为空")
	ErrPasswordTooShort    = errors.New("密码长度不能少于6位")
	ErrInvalidRole         = errors.New("无效的角色，可选值：admin/senior_operator/operator/viewer")
	ErrInvalidStatus       = errors.New("无效的状态，可选值：active/disabled")
	ErrUsernameExists      = errors.New("用户名已存在")
	ErrCannotDisableSelf   = errors.New("不能禁用自己的账户")
	ErrCannotChangeOwnRole = errors.New("不能修改自己的角色")
	ErrUserNotFound        = errors.New("用户不存在")
)

// UserService 用户管理服务
type UserService struct{}

// 模块日志：所有用户管理日志自动携带 module=user 字段
var userLog = logger.WithModule("user")

// NewUserService 创建用户管理服务实例
func NewUserService() *UserService {
	return &UserService{}
}

// ==================== 用户列表 ====================

// ListUsers 获取所有用户列表（转换为 UserInfo）
func (s *UserService) ListUsers(ctx context.Context) (*models.UserListResponse, error) {
	users, err := repository.ListUsers(ctx)
	if err != nil {
		userLog.Error("查询用户列表失败", "error", err)
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

	// 2. 检查用户名全局唯一性（数据库本身也有唯一约束）
	exists, err := repository.CheckUsernameExists(ctx, req.Username)
	if err != nil {
		userLog.Error("检查用户名唯一性失败",
			"username", req.Username,
			"error", err,
		)
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// 3. 生成密码哈希
	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		userLog.Error("生成密码哈希失败",
			"username", req.Username,
			"error", err,
		)
		return nil, err
	}

	// 4. 构造用户对象（v110：支持SchoolID）
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Status:       models.StatusActive, // 新建用户默认启用
		SchoolID:     req.SchoolID,        // v110
	}

	// 5. 写入数据库
	if err := repository.CreateUser(ctx, user); err != nil {
		userLog.Error("创建用户失败",
			"username", req.Username,
			"role", req.Role,
			"school_id", req.SchoolID,
			"error", err,
		)
		return nil, err
	}

	// INFO：用户创建成功，记录关键字段便于审计
	userLog.Info("创建用户成功",
		"username", user.Username,
		"user_id", user.ID,
		"role", user.Role,
		"school_id", user.SchoolID,
	)

	// 6. 重新查询完整用户信息（含created_at等数据库生成字段）
	created, err := repository.FindUserByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return created.ToUserInfo(), nil
}

// ==================== 编辑用户 ====================

// UpdateUser 更新用户基本信息（显示名+角色）
// v110：支持 req.SchoolID（若为空则保留原值，避免误清空）
func (s *UserService) UpdateUser(ctx context.Context, userID string, currentUserID string, req *models.UpdateUserRequest) (*models.UserInfo, error) {
	// 1. 参数校验
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.DisplayName == "" {
		return nil, ErrDisplayNameRequired
	}
	if !models.IsValidRole(req.Role) {
		return nil, ErrInvalidRole
	}

	// 2. 查询目标用户（用于：自改角色保护 + 保留school_id）
	existing, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 3. 不允许修改自己的角色（防止admin把自己降级）
	if userID == currentUserID && existing.Role != req.Role {
		return nil, ErrCannotChangeOwnRole
	}

	// 4. 计算目标 schoolID（req没传则保持原值）
	targetSchoolID := existing.SchoolID
	if req.SchoolID != nil {
		targetSchoolID = req.SchoolID
	}

	// 5. 更新数据库
	if err := repository.UpdateUser(ctx, userID, req.DisplayName, req.Role, targetSchoolID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		userLog.Error("更新用户失败",
			"user_id", userID,
			"new_role", req.Role,
			"target_school_id", targetSchoolID,
			"error", err,
		)
		return nil, err
	}

	// 6. 返回更新后的用户信息
	updated, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// INFO：用户更新成功
	userLog.Info("更新用户成功",
		"username", updated.Username,
		"user_id", userID,
		"role", updated.Role,
		"school_id", updated.SchoolID,
		"operator_id", currentUserID,
	)
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
		userLog.Error("生成密码哈希失败",
			"user_id", userID,
			"error", err,
		)
		return err
	}

	// 4. 更新密码
	if err := repository.UpdatePassword(ctx, userID, passwordHash); err != nil {
		userLog.Error("重置密码失败",
			"username", user.Username,
			"user_id", userID,
			"error", err,
		)
		return err
	}

	// INFO：密码重置成功
	userLog.Info("重置密码成功",
		"username", user.Username,
		"user_id", userID,
	)
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
		userLog.Error("更新用户状态失败",
			"username", user.Username,
			"user_id", userID,
			"new_status", req.Status,
			"error", err,
		)
		return err
	}

	// INFO：状态变更成功
	userLog.Info("更新用户状态成功",
		"username", user.Username,
		"user_id", userID,
		"old_status", user.Status,
		"new_status", req.Status,
		"operator_id", currentUserID,
	)
	return nil
}

// ==================== 课程分配 ====================

// GetAssignments 获取用户课程分配列表
func (s *UserService) GetAssignments(ctx context.Context, userID string) ([]*models.CourseAssignment, error) {
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
		userLog.Error("更新课程分配失败",
			"user_id", userID,
			"course_count", len(req.CourseCodes),
			"error", err,
		)
		return nil, err
	}

	// INFO：课程分配更新成功
	userLog.Info("更新课程分配成功",
		"user_id", userID,
		"course_count", len(req.CourseCodes),
		"operator_id", adminID,
	)

	// 3. 返回最新的分配列表
	return repository.GetUserAssignments(ctx, userID)
}
