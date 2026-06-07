package services

// 用户管理业务逻辑层
//
// 迭代一 Phase 3.2 改动（本次）：
//   - 新增 CreateUserWithSchool：把"建用户(users) + 入校(school_members)"包进同一事务，
//     任一步失败整体回滚，根治"建了用户却不在本校名单"的孤儿账号。
//   - 原 CreateUser 改为转调 CreateUserWithSchool(schoolID="")——只建用户不入校，
//     行为与历史完全一致(admin 不指定学校的场景)，所有现有调用方无需改动。
//   - 方案A 并发兜底：事务内 INSERT 撞 users_username_key 唯一约束(23505)时，
//     翻译为 ErrUsernameExists；事务外预查重仍保留，提供快速友好的错误提示。
//   - BatchCreateUsers(批量建用户) 在独立文件 user_batch_service.go，避免本文件膨胀。

import (
        "context"
        "errors"
        "strings"

        "github.com/google/uuid"
        "tedna/internal/database"
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

// ==================== 创建用户（基础校验，供单建与事务编排共用）====================

// validateCreateUserReq 创建用户请求的公共校验（去空格 + 必填 + 密码长度 + 角色合法）
// 原地修剪 req.Username/req.DisplayName 的首尾空格。校验顺序与历史 CreateUser 一致。
func validateCreateUserReq(req *models.CreateUserRequest) error {
        req.Username = strings.TrimSpace(req.Username)
        req.DisplayName = strings.TrimSpace(req.DisplayName)

        if req.Username == "" {
                return ErrUsernameRequired
        }
        if req.DisplayName == "" {
                return ErrDisplayNameRequired
        }
        if len(req.Password) < 6 {
                return ErrPasswordTooShort
        }
        if !models.IsValidRole(req.Role) {
                return ErrInvalidRole
        }
        return nil
}

// CreateUser 创建用户（不入校）——保持历史签名与行为不变
//
// 内部转调 CreateUserWithSchool(schoolID="")：只建 users 记录、不写 school_members。
// 适用于 admin 创建用户但不指定学校归属的场景(意图不明确，交后续手动或教研组绑定)。
// 所有历史调用方(包括 admin_handler 的非 senior 分支、其它处)无需改动。
func (s *UserService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserInfo, error) {
        return s.CreateUserWithSchool(ctx, req, "", "")
}

// CreateUserWithSchool 事务化创建用户 + 可选入校（Phase 3.2 核心）
//
// 语义：
//   - schoolID == ""  ：只建 users 记录，不写 school_members(等价于旧 CreateUser)。
//   - schoolID != ""  ：在同一事务内先建 users、再写 school_members(source 标记来源)，
//                       任一步失败整体回滚——杜绝"建了用户却不在本校名单"的孤儿。
//
// 校验与并发兜底(方案A)：
//   1. 事务外公共校验(validateCreateUserReq) + 用户名预查重(CheckUsernameExists)——
//      提供快速、友好的错误提示，挡掉绝大多数重名。
//   2. 事务内 INSERT 仍可能因并发撞 users_username_key 唯一约束(23505)，
//      此时 repository.IsUniqueViolation 判定后翻译为 ErrUsernameExists，事务回滚。
//
// 参数 source：写入 school_members.source 的来源标记
//   ('school_admin_create'/'admin_create'/'group_member'/'migration'/'manual')；
//   仅在 schoolID != "" 时有意义。
func (s *UserService) CreateUserWithSchool(ctx context.Context, req *models.CreateUserRequest, schoolID string, source string) (*models.UserInfo, error) {
        // 1. 公共校验(原地修剪空格)
        if err := validateCreateUserReq(req); err != nil {
                return nil, err
        }

        // 2. 事务外预查重(方案A第一道：友好错误)
        exists, err := repository.CheckUsernameExists(ctx, req.Username)
        if err != nil {
                userLog.Error("检查用户名唯一性失败", "username", req.Username, "error", err)
                return nil, err
        }
        if exists {
                return nil, ErrUsernameExists
        }

        // 3. 生成密码哈希
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

        // 4. 开启事务：建用户 + (可选)入校，原子提交
        tx, err := database.DB.Begin(ctx)
        if err != nil {
                userLog.Error("开启建用户事务失败", "username", req.Username, "error", err)
                return nil, err
        }
        // 任何提前 return 都会触发回滚；Commit 成功后再次 Rollback 是无操作(pgx 安全)
        defer func() { _ = tx.Rollback(ctx) }()

        // 4a. 事务内建用户
        if err := repository.CreateUserTx(ctx, tx, user); err != nil {
                // 方案A第二道：并发撞唯一约束 → 翻译为友好错误
                if repository.IsUniqueViolation(err) {
                        return nil, ErrUsernameExists
                }
                userLog.Error("创建用户失败(事务)", "username", req.Username, "role", req.Role, "error", err)
                return nil, err
        }

        // 4b. 事务内入校(仅当指定了学校)
        if schoolID != "" {
                if err := repository.AddSchoolMemberTx(ctx, tx, schoolID, user.ID, source); err != nil {
                        userLog.Error("写入学校成员失败(事务，将回滚)",
                                "username", req.Username, "user_id", user.ID, "school_id", schoolID, "error", err)
                        return nil, err
                }
        }

        // 5. 提交事务
        if err := tx.Commit(ctx); err != nil {
                userLog.Error("提交建用户事务失败", "username", req.Username, "user_id", user.ID, "error", err)
                return nil, err
        }

        userLog.Info("创建用户成功",
                "username", user.Username, "user_id", user.ID, "role", user.Role,
                "school_id", schoolID, "source", source)

        // 6. 回读返回(事务已提交，普通连接可见)
        created, err := repository.FindUserByID(ctx, user.ID)
        if err != nil {
                return nil, err
        }
        return created.ToUserInfo(), nil
}

// ==================== 编辑用户 ====================

func (s *UserService) UpdateUser(ctx context.Context, userID string, currentUserID string, req *models.UpdateUserRequest) (*models.UserInfo, error) {
        req.DisplayName = strings.TrimSpace(req.DisplayName)

        if req.DisplayName == "" {
                return nil, ErrDisplayNameRequired
        }
        if !models.IsValidRole(req.Role) {
                return nil, ErrInvalidRole
        }

        if userID == currentUserID {
                existing, err := repository.FindUserByID(ctx, userID)
                if err != nil {
                        return nil, err
                }
                if existing.Role != req.Role {
                        return nil, ErrCannotChangeOwnRole
                }
        }

        if err := repository.UpdateUser(ctx, userID, req.DisplayName, req.Role); err != nil {
                if errors.Is(err, repository.ErrUserNotFound) {
                        return nil, ErrUserNotFound
                }
                userLog.Error("更新用户失败", "user_id", userID, "new_role", req.Role, "error", err)
                return nil, err
        }

        updated, err := repository.FindUserByID(ctx, userID)
        if err != nil {
                return nil, err
        }

        userLog.Info("更新用户成功", "username", updated.Username, "user_id", userID, "role", updated.Role, "operator_id", currentUserID)
        return updated.ToUserInfo(), nil
}

// ==================== 重置密码 ====================

func (s *UserService) ResetPassword(ctx context.Context, userID string, req *models.ResetPasswordRequest) error {
        if len(req.NewPassword) < 6 {
                return ErrPasswordTooShort
        }

        user, err := repository.FindUserByID(ctx, userID)
        if err != nil {
                if errors.Is(err, repository.ErrUserNotFound) {
                        return ErrUserNotFound
                }
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
        if !models.IsValidStatus(req.Status) {
                return ErrInvalidStatus
        }
        if userID == currentUserID && req.Status == models.StatusDisabled {
                return ErrCannotDisableSelf
        }

        user, err := repository.FindUserByID(ctx, userID)
        if err != nil {
                if errors.Is(err, repository.ErrUserNotFound) {
                        return ErrUserNotFound
                }
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
                if errors.Is(err, repository.ErrUserNotFound) {
                        return nil, ErrUserNotFound
                }
                return nil, err
        }
        return repository.GetUserAssignments(ctx, userID)
}

func (s *UserService) UpdateAssignments(ctx context.Context, userID string, adminID string, req *models.UpdateAssignmentsRequest) ([]*models.CourseAssignment, error) {
        _, err := repository.FindUserByID(ctx, userID)
        if err != nil {
                if errors.Is(err, repository.ErrUserNotFound) {
                        return nil, ErrUserNotFound
                }
                return nil, err
        }

        if err := repository.ReplaceUserAssignments(ctx, userID, req.CourseCodes, adminID); err != nil {
                userLog.Error("更新课程分配失败", "user_id", userID, "course_count", len(req.CourseCodes), "error", err)
                return nil, err
        }

        userLog.Info("更新课程分配成功", "user_id", userID, "course_count", len(req.CourseCodes), "operator_id", adminID)
        return repository.GetUserAssignments(ctx, userID)
}
