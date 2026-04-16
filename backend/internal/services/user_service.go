package services

// 用户管理业务逻辑层

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

type UserService struct{}

var userLog = logger.WithModule("user")

func NewUserService() *UserService {
	return &UserService{}
}

// ==================== 用户列表 ====================

func (s *UserService) ListUsers(ctx context.Context) (*models.UserListResponse, error) {
	users, err := repository.ListUsers(ctx)
	if err != nil {
		userLog.Error("查询用户列表失败", "error", err)
		return nil, err
	}
	userInfos := make([]*models.UserInfo, 0, len(users))
	for _, u := range users {
		userInfos = append(userInfos, u.ToUserInfo())
	}
	return &models.UserListResponse{Users: userInfos, Total: len(userInfos)}, nil
}

// ==================== 创建用户 ====================

func (s *UserService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserInfo, error) {
	req.Username    = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Username == ""    { return nil, ErrUsernameRequired }
	if req.DisplayName == "" { return nil, ErrDisplayNameRequired }
	if len(req.Password) < 6 { return nil, ErrPasswordTooShort }
	if !models.IsValidRole(req.Role) { return nil, ErrInvalidRole }

	exists, err := repository.CheckUsernameExists(ctx, req.Username)
	if err != nil {
		userLog.Error("检查用户名唯一性失败", "username", req.Username, "error", err)
		return nil, err
	}
	if exists { return nil, ErrUsernameExists }

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		userLog.Error("生成密码哈希失败", "username", req.Username, "error", err)
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Status:       models.StatusActive,
	}

	if err := repository.CreateUser(ctx, user); err != nil {
		userLog.Error("创建用户失败", "username", req.Username, "role", req.Role, "error", err)
		return nil, err
	}

	userLog.Info("创建用户成功", "username", user.Username, "user_id", user.ID, "role", user.Role)

	created, err := repository.FindUserByID(ctx, user.ID)
	if err != nil { return nil, err }
	return created.ToUserInfo(), nil
}

// ==================== 编辑用户 ====================

func (s *UserService) UpdateUser(ctx context.Context, userID string, currentUserID string, req *models.UpdateUserRequest) (*models.UserInfo, error) {
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.DisplayName == ""     { return nil, ErrDisplayNameRequired }
	if !models.IsValidRole(req.Role) { return nil, ErrInvalidRole }

	if userID == currentUserID {
		existing, err := repository.FindUserByID(ctx, userID)
		if err != nil { return nil, err }
		if existing.Role != req.Role { return nil, ErrCannotChangeOwnRole }
	}

	if err := repository.UpdateUser(ctx, userID, req.DisplayName, req.Role); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) { return nil, ErrUserNotFound }
		userLog.Error("更新用户失败", "user_id", userID, "new_role", req.Role, "error", err)
		return nil, err
	}

	updated, err := repository.FindUserByID(ctx, userID)
	if err != nil { return nil, err }

	userLog.Info("更新用户成功", "username", updated.Username, "user_id", userID, "role", updated.Role, "operator_id", currentUserID)
	return updated.ToUserInfo(), nil
}

// ==================== 重置密码 ====================

func (s *UserService) ResetPassword(ctx context.Context, userID string, req *models.ResetPasswordRequest) error {
	if len(req.NewPassword) < 6 { return ErrPasswordTooShort }

	user, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) { return ErrUserNotFound }
		return err
	}

	passwordHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		userLog.Error("生成密码哈希失败", "user_id", userID, "error", err)
		return err
	}

	if err := repository.UpdatePassword(ctx, userID, passwordHash); err != nil {
		userLog.Error("重置密码失败", "username", user.Username, "user_id", userID, "error", err)
		return err
	}

	userLog.Info("重置密码成功", "username", user.Username, "user_id", userID)
	return nil
}

// ==================== 启用/禁用 ====================

func (s *UserService) UpdateStatus(ctx context.Context, userID string, currentUserID string, req *models.UpdateStatusRequest) error {
	if !models.IsValidStatus(req.Status) { return ErrInvalidStatus }
	if userID == currentUserID && req.Status == models.StatusDisabled { return ErrCannotDisableSelf }

	user, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) { return ErrUserNotFound }
		return err
	}

	if err := repository.UpdateStatus(ctx, userID, req.Status); err != nil {
		userLog.Error("更新用户状态失败", "username", user.Username, "user_id", userID, "new_status", req.Status, "error", err)
		return err
	}

	userLog.Info("更新用户状态成功", "username", user.Username, "user_id", userID, "old_status", user.Status, "new_status", req.Status, "operator_id", currentUserID)
	return nil
}

// ==================== 课程分配 ====================

func (s *UserService) GetAssignments(ctx context.Context, userID string) ([]*models.CourseAssignment, error) {
	_, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) { return nil, ErrUserNotFound }
		return nil, err
	}
	return repository.GetUserAssignments(ctx, userID)
}

func (s *UserService) UpdateAssignments(ctx context.Context, userID string, adminID string, req *models.UpdateAssignmentsRequest) ([]*models.CourseAssignment, error) {
	_, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) { return nil, ErrUserNotFound }
		return nil, err
	}

	if err := repository.ReplaceUserAssignments(ctx, userID, req.CourseCodes, adminID); err != nil {
		userLog.Error("更新课程分配失败", "user_id", userID, "course_count", len(req.CourseCodes), "error", err)
		return nil, err
	}

	userLog.Info("更新课程分配成功", "user_id", userID, "course_count", len(req.CourseCodes), "operator_id", adminID)
	return repository.GetUserAssignments(ctx, userID)
}
