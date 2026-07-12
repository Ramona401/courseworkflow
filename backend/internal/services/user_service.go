package services

// 用户管理业务逻辑层
//
// 迭代一 Phase 3.2 改动：
//   - 新增 CreateUserWithSchool：把"建用户(users) + 入校(school_members)"包进同一事务，
//     任一步失败整体回滚，根治"建了用户却不在本校名单"的孤儿账号。
//   - 原 CreateUser 改为转调 CreateUserWithSchool(schoolID="")——只建用户不入校，
//     行为与历史完全一致(admin 不指定学校的场景)，所有现有调用方无需改动。
//   - 方案A 并发兜底：事务内 INSERT 撞 users_username_key 唯一约束(23505)时，
//     翻译为 ErrUsernameExists；事务外预查重仍保留，提供快速友好的错误提示。
//   - BatchCreateUsers(批量建用户) 在独立文件 user_batch_service.go，避免本文件膨胀。
//
// B4 修复（新用户无个人积分账户 → admin 看不到其消费）：
//   新建用户成功后，自动为其开一个余额 0 的个人积分账户（ensurePersonalTokenAccount）。
//   根因——ConsumeTokens 在 GetTokenAccountByOwner 查不到个人账户时静默 return，
//   既不扣费也不写 token_consumption_logs，导致外部学校用户的 AI 消费完全无痕，
//   admin 概览/消费流水里只看得到早期手动建过账户的北大实验室成员。
//   修法（best-effort，绝不阻断建用户）：
//     - 位置：事务 Commit 成功之后调用（建用户是刚需，开账户是锦上添花，二者解耦）。
//     - 幂等：CreateTokenAccount 撞 (account_type,owner_id) 唯一约束返 ErrDuplicateAccount，
//             本函数视其为"已存在=成功"，重复调用无害（对齐"通知旁路范式"的容错思路）。
//     - 失败仅记 Warn，不影响 CreateUserWithSchool 的返回（账户以后可补建）。
//     - 余额 0 的账户在"允许透支"体系下不影响使用，只是从此每次 AI 调用都会留痕。
//   注意：只给"会实际用 AI 备课"的普通用户开账户即可，但为实现简单且无副作用，
//   这里对所有新建用户统一开个人账户（admin/senior 自己也开一个 0 账户无害，
//   其消费本就该留痕；token scope 的越权隔离另有 ListPrivilegedUserIDs 差集兜底，不受影响）。
//
// 账户树父链修复（2026-07-04，"admin积分页只见一校"根因治理·代码侧）：
//   问题——ensurePersonalTokenAccount 建的个人账户不挂 parent_account_id 父链，
//   而消费汇总报告的 school/region 维度靠父链（personal→school→region 三层）归属学校/区域，
//   不挂链的消费全部落进"(无归属)"，admin 汇总报告只剩早期手动挂过链的学校。
//   数据侧已用一次性 SQL 补齐存量（6区域/63学校账户 + 362个个人账户挂链），
//   本次代码侧根治"新用户再断链"：
//     - resolvePersonalParentAccountID：best-effort 解析用户所属学校
//       （repository.GetSchoolIDByUserID：school_members 权威查 + 教研组兜底，
//        与门户板块/AI模型分流同一条学校解析链）→ 取该校 school 账户ID作父链。
//       任一步失败/用户无学校/学校无账户 → 返回 nil，行为退回旧版（账户照建暂无归属）。
//     - ensurePersonalTokenAccount 建账户时带上父链；撞 ErrDuplicateAccount（重复建号等）
//       时若旧账户缺父链且本次解析到父链，则顺手补挂（backfillPersonalParentIfMissing）。
//   全部失败路径只记 Warn，绝不阻断建用户主流程。
//   调用方零改动：user_batch_service.go / user_batch_multi_school_service.go 复用本函数天然继承。

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

// initialPersonalCredits 新用户个人积分账户的初始积分（2026-07-04 新增）。
// 所有角色新建账户时直接发放此额度作为起步福利（不从学校池扣，凭空发放），
// 并写一条 initial 类型分配流水使账目可追溯。改这一个常量即改初始额度。
const initialPersonalCredits = 100

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
	// 批C(任命唯一事实源)：任命制身份不可经建号/编辑直接授予或改动
	ErrRoleAppointmentOnly = errors.New("学校管理员/区域管理员为任命制身份：请在「组织架构」对应卡片的「🛡️ 管理员」面板任命（自动升级身份）或移除任命（末个任命移除后自动降级），不能在此直接设置")
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
	// 批C：任命制身份仅可经组织任命获得，建号一律不得直接授予
	if models.IsAppointmentOnlyRole(req.Role) {
		return ErrRoleAppointmentOnly
	}
	return nil
}

// resolvePersonalParentAccountID 解析用户个人积分账户应挂载的父账户（本校学校账户）ID
//
// 账户树父链修复（2026-07-04）新增，best-effort：
//  1. GetSchoolIDByUserID 解析用户所属学校（school_members 权威 + 教研组兜底，
//     与门户板块/AI模型分流同一条学校解析链，多校成员天然取权威链首选）。
//  2. GetTokenAccountByOwner 取该校 school 类型积分账户。
//
// 任一步失败/无学校/学校无账户 → 返回 nil（调用方退回旧行为：建无父链账户）。
// 绝不向上抛错，绝不阻断建用户/建账户主流程。
func resolvePersonalParentAccountID(ctx context.Context, userID string) *string {
	schoolID, err := repository.GetSchoolIDByUserID(ctx, userID)
	if err != nil || schoolID == "" {
		return nil // 无学校归属或解析失败 → 不挂父链
	}
	schoolAcc, err := repository.GetTokenAccountByOwner(ctx, models.AccountTypeSchool, schoolID)
	if err != nil || schoolAcc == nil || schoolAcc.ID == "" {
		return nil // 学校账户不存在（理论上已全量补建，防御性兜底）→ 不挂父链
	}
	parentID := schoolAcc.ID
	return &parentID
}

// backfillPersonalParentIfMissing 为已存在但缺父链的个人账户补挂父链（best-effort 幂等）
//
// 场景：ensurePersonalTokenAccount 撞 ErrDuplicateAccount（该用户已有个人账户）时调用。
// 若旧账户 parent_account_id 为空且本次解析到了父链，则补挂；已有父链一律不动
// （不做"换校迁移"——历史消费归属保持稳定，换校场景由管理员按需用补建 SQL 处理）。
func backfillPersonalParentIfMissing(ctx context.Context, userID string, parentID *string) {
	if parentID == nil || *parentID == "" {
		return // 本次也没解析到父链，无从补挂
	}
	existing, err := repository.GetTokenAccountByOwner(ctx, models.AccountTypePersonal, userID)
	if err != nil || existing == nil {
		return // 查不到旧账户（理论不可能，防御性兜底）
	}
	if existing.ParentAccountID != nil && *existing.ParentAccountID != "" {
		return // 已有父链，不动（保持历史归属稳定）
	}
	if err := repository.UpdateTokenAccountParent(ctx, existing.ID, *parentID); err != nil {
		userLog.Warn("补挂个人积分账户父链失败(不阻断)",
			"user_id", userID, "account_id", existing.ID, "error", err)
		return
	}
	userLog.Info("补挂个人积分账户父链成功",
		"user_id", userID, "account_id", existing.ID, "parent_account_id", *parentID)
}

// ensurePersonalTokenAccount 为用户确保存在一个个人积分账户（B4 修复，best-effort 幂等）
//
// 设计（与 CreateUserWithSchool 解耦，事务外调用）：
//   - 幂等：已存在同类型账户时 CreateTokenAccount 返回 ErrDuplicateAccount，本函数视为成功，
//     并顺手检查旧账户是否缺父链（缺则补挂，见 backfillPersonalParentIfMissing）。
//   - 容错：任何失败只记 Warn，绝不向上传播错误（不影响建用户主流程）。
//   - 余额 0：新账户 balance=0，在"允许透支"体系下不影响使用，仅使 AI 消费从此留痕。
//   - 父链（2026-07-04 新增）：建账户时 best-effort 挂 parent_account_id = 本校学校账户，
//     使消费汇总报告 school/region 维度能正确归属；解析失败退回旧行为（无父链照建）。
//
// displayName 用作账户展示名（取用户显示名，便于 admin 在积分列表里辨认）。
func ensurePersonalTokenAccount(ctx context.Context, userID string, displayName string) {
	if userID == "" {
		return
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = "个人积分账户"
	}

	// 账户树父链修复：best-effort 解析本校学校账户作父链（失败返回 nil，退回旧行为）
	parentID := resolvePersonalParentAccountID(ctx, userID)

	acc := &models.TokenAccount{
		AccountType:     models.AccountTypePersonal,
		OwnerID:         userID,
		ParentAccountID: parentID,
		DisplayName:     name,
		Balance:         initialPersonalCredits,
		FrozenAmount:    0,
		TotalConsumed:   0,
		TotalQuota:      0,
		MonthlyQuota:    0,
		Status:          models.AccountStatusActive,
	}
	err := repository.CreateTokenAccount(ctx, acc)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateAccount) {
			// 已有账户 → 幂等成功；旧账户若缺父链且本次解析到了父链，顺手补挂
			backfillPersonalParentIfMissing(ctx, userID, parentID)
			return
		}
		// 其它失败只记 Warn，不阻断建用户（账户以后可用补建 SQL 或再次建用户触发）
		userLog.Warn("自动创建个人积分账户失败(不阻断建用户)",
			"user_id", userID, "error", err)
		return
	}
	userLog.Info("自动创建个人积分账户成功",
		"user_id", userID, "account_id", acc.ID, "parent_linked", parentID != nil,
		"initial_credits", initialPersonalCredits)

	// 初始积分账本记录：写一条 initial 类型分配流水（from=to=账户自身），
	// 使新用户这 100 积分在账本上有明确来源，admin 在分配记录里可追溯每笔初始发放。
	// best-effort：流水写失败仅 Warn，不影响账户已带初始余额这一事实。
	initAlloc := &models.TokenAllocation{
		FromAccountID:  acc.ID,
		ToAccountID:    acc.ID,
		Amount:         initialPersonalCredits,
		AllocationType: models.AllocationTypeInitial,
		Memo:           "新用户初始积分",
		OperatorID:     "00000000-0000-0000-0000-000000000000", // 系统操作
	}
	if err := repository.CreateTokenAllocation(ctx, initAlloc); err != nil {
		userLog.Warn("初始积分分配流水写入失败(不影响账户初始余额)",
			"user_id", userID, "account_id", acc.ID, "error", err)
	}
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
//     任一步失败整体回滚——杜绝"建了用户却不在本校名单"的孤儿。
//
// 校验与并发兜底(方案A)：
//  1. 事务外公共校验(validateCreateUserReq) + 用户名预查重(CheckUsernameExists)——
//     提供快速、友好的错误提示，挡掉绝大多数重名。
//  2. 事务内 INSERT 仍可能因并发撞 users_username_key 唯一约束(23505)，
//     此时 repository.IsUniqueViolation 判定后翻译为 ErrUsernameExists，事务回滚。
//
// 参数 source：写入 school_members.source 的来源标记
//
//	('school_admin_create'/'admin_create'/'group_member'/'migration'/'manual')；
//	仅在 schoolID != "" 时有意义。
//
// B4：事务提交成功后，best-effort 为新用户开个人积分账户（见 ensurePersonalTokenAccount）。
//
//	账户树父链修复后，该账户会自动挂到本校学校账户下（入校已在事务内提交，解析必命中）。
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

	// 5b. B4：事务外 best-effort 开个人积分账户（失败只记 Warn，不影响建用户返回）
	ensurePersonalTokenAccount(ctx, user.ID, user.DisplayName)

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

	// 批C改造：无条件先取目标现状——自改角色守卫与任命制身份守卫都需要现值比对
	existing, err := repository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if req.Role != existing.Role {
		// 不能修改自己的角色（既有规则）
		if userID == currentUserID {
			return nil, ErrCannotChangeOwnRole
		}
		// 批C(任命唯一事实源)：任命制身份的升与降都不走编辑——
		//   升级：请到组织架构任命（B13 自动升身份）；
		//   降级：移除其全部任命后自动发生（organization_admin_service 批C）。
		// 目标现身份或目标新身份任一为任命制 → 拒绝并给出任命引导。
		// （现身份=任命制而新身份=普通：即"手动降级"，会制造"任命在、身份无"的反向悬空，同样禁止。）
		if models.IsAppointmentOnlyRole(req.Role) || models.IsAppointmentOnlyRole(existing.Role) {
			return nil, ErrRoleAppointmentOnly
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
