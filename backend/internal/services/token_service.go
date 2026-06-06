package services

// token_service.go — Token积分系统核心业务逻辑
//
// v128 新增（阶段C · Token/积分系统）
// v129 改造（积分机制融合 · 对齐AOCI精确积分计算）
// v172 改造（积分管理三级数据权限隔离）：TokenScope + ResolveTokenScope + 各 Scoped 方法
// v172.2 新增（采购记录隔离）：ListPurchases 旧签名委托 nil；新增 ListPurchasesScoped
//
// 权限范围设计（fail-closed，任何不确定一律收窄为空集，绝不放大为全量）：
//   - admin           → 看全部（白名单 nil）
//   - senior_operator → 仅本校：账户/概览/采购 owner_id ∈ {本校成员user_id ∪ 学校组织ID}；
//                       消费流水 user_id ∈ 本校成员user_id；
//                       未绑定学校 / 查询失败 → 空集（看不到任何数据 + 上层给提示）
//   - operator/viewer → 仅本人：账户/概览/采购 owner_id = 自己；消费流水 user_id = 自己
//
// 核心流程：采购 → 区域账户充值 → 分配到学校账户 → 分配到个人账户 → AI调用消费

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// tokenLog 模块级结构化日志器
var tokenLog = logger.WithModule("services.token")

// ==================== 错误常量 ====================

var (
	ErrTokenInvalidAccountType = errors.New("无效的账户类型")
	ErrTokenInvalidAmount      = errors.New("积分数量必须大于0")
	ErrTokenSelfAllocate       = errors.New("不能分配给自己")
	ErrTokenNotParentChild     = errors.New("只能向下级账户分配积分")
	ErrTokenAccountNotActive   = errors.New("账户不在活跃状态")
)

// ==================== TokenService 结构体 ====================

// TokenService Token积分系统核心服务
type TokenService struct{}

// NewTokenService 创建TokenService实例
func NewTokenService() *TokenService {
	return &TokenService{}
}

// ==================== v172 数据权限范围（TokenScope）====================

// TokenScope 描述当前请求者对积分数据的可见范围
//
// 白名单字段语义（与 repository 层完全一致）：
//   - nil        → 不过滤（看全部，仅 admin）
//   - 非nil空切片 → 匹配空集（fail-closed，看不到任何数据）
//   - 非空        → 仅匹配名单内
//
// 查询维度复用两个白名单：
//   - OwnerIDs：账户列表 / 概览统计 / 分配记录 / 采购记录（按 token_accounts.owner_id）
//   - UserIDs ：消费流水（按 token_consumption_logs.user_id）
type TokenScope struct {
	Role          string   // 请求者角色
	IsAdmin       bool     // 是否系统管理员（看全部）
	OwnerIDs      []string // 账户/概览/分配/采购 owner_id 白名单
	UserIDs       []string // 消费流水 user_id 白名单
	Blocked       bool     // 是否被收窄为空集（如 senior 未绑定学校）
	BlockedReason string   // 收窄原因（供上层提示）
}

// ResolveTokenScope 根据角色与用户ID解析数据可见范围（积分系统唯一权限决策点，fail-closed）
func (s *TokenService) ResolveTokenScope(ctx context.Context, role string, userID string) *TokenScope {
	if role == "" || userID == "" {
		return &TokenScope{
			Role:          role,
			IsAdmin:       false,
			OwnerIDs:      []string{},
			UserIDs:       []string{},
			Blocked:       true,
			BlockedReason: "未认证",
		}
	}

	switch role {
	case models.RoleAdmin:
		return &TokenScope{
			Role:     role,
			IsAdmin:  true,
			OwnerIDs: nil,
			UserIDs:  nil,
		}

	case models.RoleSeniorOperator:
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err != nil || school == nil || school.ID == "" {
			return &TokenScope{
				Role:          role,
				IsAdmin:       false,
				OwnerIDs:      []string{},
				UserIDs:       []string{},
				Blocked:       true,
				BlockedReason: "您尚未绑定学校，请联系系统管理员",
			}
		}
		memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, school.ID)
		if mErr != nil {
			tokenLog.Warn("查询本校成员失败，收窄为空集", "school", school.ID, "error", mErr)
			return &TokenScope{
				Role:          role,
				IsAdmin:       false,
				OwnerIDs:      []string{},
				UserIDs:       []string{},
				Blocked:       true,
				BlockedReason: "查询本校成员失败",
			}
		}
		ownerIDs := make([]string, 0, len(memberIDs)+1)
		ownerIDs = append(ownerIDs, memberIDs...)
		ownerIDs = append(ownerIDs, school.ID)
		userIDs := make([]string, 0, len(memberIDs))
		userIDs = append(userIDs, memberIDs...)

		return &TokenScope{
			Role:     role,
			IsAdmin:  false,
			OwnerIDs: ownerIDs,
			UserIDs:  userIDs,
		}

	default:
		return &TokenScope{
			Role:     role,
			IsAdmin:  false,
			OwnerIDs: []string{userID},
			UserIDs:  []string{userID},
		}
	}
}

// ==================== 账户管理 ====================

// CreateAccount 创建积分账户
func (s *TokenService) CreateAccount(ctx context.Context, req *models.CreateTokenAccountRequest) (*models.TokenAccount, error) {
	if req.AccountType != models.AccountTypeRegion &&
		req.AccountType != models.AccountTypeSchool &&
		req.AccountType != models.AccountTypePersonal {
		return nil, ErrTokenInvalidAccountType
	}

	if req.DisplayName == "" {
		return nil, fmt.Errorf("账户名称不能为空")
	}

	acc := &models.TokenAccount{
		AccountType:     req.AccountType,
		OwnerID:         req.OwnerID,
		ParentAccountID: req.ParentAccountID,
		DisplayName:     req.DisplayName,
		Balance:         0,
		FrozenAmount:    0,
		TotalConsumed:   0,
		TotalQuota:      0,
		MonthlyQuota:    req.MonthlyQuota,
		Status:          models.AccountStatusActive,
	}

	if err := repository.CreateTokenAccount(ctx, acc); err != nil {
		if errors.Is(err, repository.ErrDuplicateAccount) {
			return nil, err
		}
		return nil, fmt.Errorf("创建账户失败: %w", err)
	}

	tokenLog.Info("创建账户成功",
		"type", acc.AccountType,
		"owner", acc.OwnerID,
		"name", acc.DisplayName)
	return acc, nil
}

// GetAccount 获取账户详情（含预警配置和子账户）
func (s *TokenService) GetAccount(ctx context.Context, accountID string) (*models.TokenAccountDetail, error) {
	acc, err := repository.GetTokenAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	detail := &models.TokenAccountDetail{
		TokenAccount:     *acc,
		AccountTypeName:  models.AccountTypeNameMap[acc.AccountType],
		StatusName:       models.AccountStatusNameMap[acc.Status],
		AvailableBalance: acc.Balance - acc.FrozenAmount,
	}
	if acc.TotalQuota > 0 {
		detail.UsagePercent = float64(acc.TotalConsumed) * 100.0 / float64(acc.TotalQuota)
	}

	alertCfg, _ := repository.GetTokenAlertConfig(ctx, accountID)
	detail.AlertConfig = alertCfg

	children, _ := repository.ListChildAccounts(ctx, accountID)
	detail.ChildAccounts = children

	return detail, nil
}

// GetAccountByOwner 根据实体类型+实体ID获取账户
func (s *TokenService) GetAccountByOwner(ctx context.Context, accountType string, ownerID string) (*models.TokenAccount, error) {
	return repository.GetTokenAccountByOwner(ctx, accountType, ownerID)
}

// ListAccounts 查询账户列表（旧签名，无范围限制，委托 nil 白名单；保留兼容存量调用）
func (s *TokenService) ListAccounts(ctx context.Context, accountType string, parentAccountID string, status string, limit int, offset int) ([]*models.TokenAccountListItem, int, error) {
	return repository.ListTokenAccounts(ctx, accountType, parentAccountID, status, nil, limit, offset)
}

// ListAccountsScoped 查询账户列表（v172 范围感知）
func (s *TokenService) ListAccountsScoped(ctx context.Context, accountType string, parentAccountID string, status string, scope *TokenScope, limit int, offset int) ([]*models.TokenAccountListItem, int, error) {
	var ownerIDs []string
	if scope != nil {
		ownerIDs = scope.OwnerIDs
	}
	return repository.ListTokenAccounts(ctx, accountType, parentAccountID, status, ownerIDs, limit, offset)
}

// UpdateAccountStatus 更新账户状态
func (s *TokenService) UpdateAccountStatus(ctx context.Context, accountID string, status string) error {
	if status != models.AccountStatusActive &&
		status != models.AccountStatusSuspended &&
		status != models.AccountStatusExpired {
		return fmt.Errorf("无效的账户状态: %s", status)
	}
	return repository.UpdateTokenAccountStatus(ctx, accountID, status)
}

// GetSchoolAccountByAdmin 根据学校管理员用户ID查找本校积分账户（保留兼容）
func (s *TokenService) GetSchoolAccountByAdmin(ctx context.Context, adminUserID string) (*models.TokenAccount, error) {
	school, err := repository.GetSchoolByAdminUserID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	return repository.GetTokenAccountByOwner(ctx, models.AccountTypeSchool, school.ID)
}

// ==================== 积分分配 ====================

// AllocateTokens 从上级账户分配积分到下级账户
func (s *TokenService) AllocateTokens(ctx context.Context, fromAccountID string, req *models.AllocateTokensRequest, operatorID string) error {
	if req.Amount <= 0 {
		return ErrTokenInvalidAmount
	}
	if fromAccountID == req.ToAccountID {
		return ErrTokenSelfAllocate
	}

	fromAcc, err := repository.GetTokenAccountByID(ctx, fromAccountID)
	if err != nil {
		return fmt.Errorf("来源账户不存在: %w", err)
	}
	if fromAcc.Status != models.AccountStatusActive {
		return ErrTokenAccountNotActive
	}

	toAcc, err := repository.GetTokenAccountByID(ctx, req.ToAccountID)
	if err != nil {
		return fmt.Errorf("目标账户不存在: %w", err)
	}
	if toAcc.Status != models.AccountStatusActive {
		return ErrTokenAccountNotActive
	}

	validRelation := false
	if toAcc.ParentAccountID != nil && *toAcc.ParentAccountID == fromAccountID {
		validRelation = true
	}
	if !validRelation && toAcc.ParentAccountID != nil {
		parentAcc, parentErr := repository.GetTokenAccountByID(ctx, *toAcc.ParentAccountID)
		if parentErr == nil && parentAcc.ParentAccountID != nil && *parentAcc.ParentAccountID == fromAccountID {
			validRelation = true
		}
	}
	if !validRelation {
		return ErrTokenNotParentChild
	}

	if err := repository.DeductBalanceForAllocation(ctx, fromAccountID, req.Amount); err != nil {
		return err
	}

	if err := repository.AddBalance(ctx, req.ToAccountID, req.Amount); err != nil {
		_ = repository.AddBalance(ctx, fromAccountID, req.Amount)
		return fmt.Errorf("增加下级余额失败: %w", err)
	}

	alloc := &models.TokenAllocation{
		FromAccountID:  fromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		AllocationType: models.AllocationTypeManual,
		Memo:           req.Memo,
		OperatorID:     operatorID,
	}
	if err := repository.CreateTokenAllocation(ctx, alloc); err != nil {
		tokenLog.Warn("分配成功但记录流水失败",
			"from", fromAccountID,
			"to", req.ToAccountID,
			"amount", req.Amount,
			"error", err)
	}

	tokenLog.Info("分配成功",
		"from", fromAccountID,
		"to", req.ToAccountID,
		"amount", req.Amount,
		"operator", operatorID)
	return nil
}

// ListAllocations 查询分配记录（旧签名，无范围限制；保留兼容）
func (s *TokenService) ListAllocations(ctx context.Context, fromAccountID string, toAccountID string, limit int, offset int) ([]*models.AllocationListItem, int, error) {
	return repository.ListTokenAllocations(ctx, fromAccountID, toAccountID, nil, limit, offset)
}

// ListAllocationsScoped 查询分配记录（v172 范围感知，按 from/to 账户 owner 白名单）
func (s *TokenService) ListAllocationsScoped(ctx context.Context, fromAccountID string, toAccountID string, scope *TokenScope, limit int, offset int) ([]*models.AllocationListItem, int, error) {
	var ownerIDs []string
	if scope != nil {
		ownerIDs = scope.OwnerIDs
	}
	return repository.ListTokenAllocations(ctx, fromAccountID, toAccountID, ownerIDs, limit, offset)
}

// ==================== 采购/充值 ====================

// PurchaseTokens 采购/充值积分
func (s *TokenService) PurchaseTokens(ctx context.Context, req *models.PurchaseTokensRequest, operatorID string) error {
	if req.Amount <= 0 {
		return ErrTokenInvalidAmount
	}

	acc, err := repository.GetTokenAccountByID(ctx, req.AccountID)
	if err != nil {
		return fmt.Errorf("目标账户不存在: %w", err)
	}
	if acc.Status != models.AccountStatusActive {
		return ErrTokenAccountNotActive
	}

	var validUntil *time.Time
	if req.ValidUntil != nil && *req.ValidUntil != "" {
		t, parseErr := time.Parse(time.RFC3339, *req.ValidUntil)
		if parseErr == nil {
			validUntil = &t
		}
	}

	if err := repository.AddBalance(ctx, req.AccountID, req.Amount); err != nil {
		return err
	}

	purchase := &models.TokenPurchase{
		AccountID:    req.AccountID,
		Amount:       req.Amount,
		PurchaseType: req.PurchaseType,
		OrderNo:      req.OrderNo,
		Memo:         req.Memo,
		OperatorID:   operatorID,
		ValidUntil:   validUntil,
	}
	if err := repository.CreateTokenPurchase(ctx, purchase); err != nil {
		tokenLog.Warn("充值成功但记录采购流水失败",
			"account", req.AccountID,
			"amount", req.Amount,
			"error", err)
	}

	tokenLog.Info("充值成功",
		"account", req.AccountID,
		"amount", req.Amount,
		"type", req.PurchaseType,
		"operator", operatorID)
	return nil
}

// ListPurchases 查询采购记录（旧签名，无范围限制，委托 nil 白名单；保留兼容）
func (s *TokenService) ListPurchases(ctx context.Context, accountID string, limit int, offset int) ([]*models.PurchaseListItem, int, error) {
	return repository.ListTokenPurchases(ctx, accountID, nil, limit, offset)
}

// ListPurchasesScoped 查询采购记录（v172.2 范围感知，按目标账户 owner 白名单）
func (s *TokenService) ListPurchasesScoped(ctx context.Context, accountID string, scope *TokenScope, limit int, offset int) ([]*models.PurchaseListItem, int, error) {
	var ownerIDs []string
	if scope != nil {
		ownerIDs = scope.OwnerIDs
	}
	return repository.ListTokenPurchases(ctx, accountID, ownerIDs, limit, offset)
}

// ==================== 消费流程（AI调用时使用）====================

// ConsumeTokens 消费积分（AI调用完成后调用）
func (s *TokenService) ConsumeTokens(ctx context.Context, req *models.TokenConsumeRequest) error {
	var creditCost float64
	var calc *models.CreditCalculation

	if req.Calculation != nil && req.Calculation.CreditsConsumed > 0 {
		creditCost = req.Calculation.CreditsConsumed
		calc = req.Calculation
	} else if req.TokensUsed > 0 {
		creditCost = float64(req.TokensUsed) / 1000.0
		if creditCost <= 0 {
			return nil
		}
	} else {
		return nil
	}

	acc, err := repository.GetTokenAccountByOwner(ctx, models.AccountTypePersonal, req.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrTokenAccountNotFound) {
			return nil
		}
		return fmt.Errorf("查询用户积分账户失败: %w", err)
	}

	if acc.Status != models.AccountStatusActive {
		return nil
	}

	balanceBefore := acc.Balance

	if err := repository.DirectDeductBalance(ctx, acc.ID, creditCost); err != nil {
		tokenLog.Warn("消费扣减失败(不阻断AI)",
			"account", acc.ID,
			"amount", creditCost,
			"error", err)
		return nil
	}

	consumeLog := &models.TokenConsumptionLog{
		AccountID:     acc.ID,
		UserID:        req.UserID,
		Amount:        creditCost,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceBefore - creditCost,
		SceneCode:     req.SceneCode,
		ModelUsed:     req.ModelUsed,
		TokensUsed:    req.TokensUsed,
		LessonPlanID:  req.LessonPlanID,
		PipelineID:    req.PipelineID,
	}
	if calc != nil {
		consumeLog.InputTokens = calc.InputTokens
		consumeLog.OutputTokens = calc.OutputTokens
		consumeLog.ModelName = calc.ModelName
		consumeLog.Provider = calc.Provider
		consumeLog.CostUSD = calc.CostUSD
		consumeLog.ExchangeRate = calc.ExchangeRate
		consumeLog.Multiplier = calc.Multiplier
		consumeLog.CreditsConsumed = calc.CreditsConsumed
		consumeLog.LatencyMs = int(calc.LatencyMs)
	}
	if err := repository.CreateTokenConsumptionLog(ctx, consumeLog); err != nil {
		tokenLog.Warn("记录消费流水失败",
			"account", acc.ID,
			"amount", creditCost,
			"error", err)
	}

	return nil
}

// ListConsumptionLogs 查询消费流水（旧签名，无范围限制；保留兼容）
func (s *TokenService) ListConsumptionLogs(ctx context.Context, accountID string, userID string, sceneCode string, limit int, offset int) ([]*models.ConsumptionListItem, int, error) {
	return repository.ListTokenConsumptionLogs(ctx, accountID, userID, sceneCode, nil, limit, offset)
}

// ListConsumptionLogsScoped 查询消费流水（v172 范围感知，按 user_id 白名单）
func (s *TokenService) ListConsumptionLogsScoped(ctx context.Context, accountID string, userID string, sceneCode string, scope *TokenScope, limit int, offset int) ([]*models.ConsumptionListItem, int, error) {
	var userIDs []string
	if scope != nil {
		userIDs = scope.UserIDs
	}
	return repository.ListTokenConsumptionLogs(ctx, accountID, userID, sceneCode, userIDs, limit, offset)
}

// ==================== 统计 ====================

// GetOverviewStats 获取概览统计（旧签名，全系统；保留兼容）
func (s *TokenService) GetOverviewStats(ctx context.Context) (*models.TokenOverviewStats, error) {
	return repository.GetTokenOverviewStatsScoped(ctx, nil)
}

// GetOverviewStatsScoped 获取概览统计（v172 范围感知，按 owner_id 白名单）
func (s *TokenService) GetOverviewStatsScoped(ctx context.Context, scope *TokenScope) (*models.TokenOverviewStats, error) {
	var ownerIDs []string
	if scope != nil {
		ownerIDs = scope.OwnerIDs
	}
	return repository.GetTokenOverviewStatsScoped(ctx, ownerIDs)
}

// ==================== 预警配置 ====================

// GetAlertConfig 获取预警配置
func (s *TokenService) GetAlertConfig(ctx context.Context, accountID string) (*models.TokenAlertConfig, error) {
	return repository.GetTokenAlertConfig(ctx, accountID)
}

// UpdateAlertConfig 更新预警配置
func (s *TokenService) UpdateAlertConfig(ctx context.Context, accountID string, req *models.UpdateAlertConfigRequest) error {
	if req.WarnThreshold <= 0 || req.WarnThreshold > 100 {
		return fmt.Errorf("预警阈值必须在1-100之间")
	}
	if req.UrgentThreshold <= 0 || req.UrgentThreshold > 100 {
		return fmt.Errorf("紧急阈值必须在1-100之间")
	}
	if req.UrgentThreshold <= req.WarnThreshold {
		return fmt.Errorf("紧急阈值必须大于预警阈值")
	}
	return repository.UpsertTokenAlertConfig(ctx, accountID, req)
}
