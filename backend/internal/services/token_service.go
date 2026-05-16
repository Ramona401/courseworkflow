package services

// token_service.go — Token积分系统核心业务逻辑
//
// v128 新增（阶段C · Token/积分系统）
// v129 改造（积分机制融合 · 对齐AOCI精确积分计算）：
//   - ConsumeTokens 从固定汇率改为精确模式（接收CreditCalculation）
//   - 消费扣减允许透支（对齐AOCI，不再GREATEST防负数）
//   - 消费流水记录完整计算过程（9个新字段）
//
// 原v128注释保留：
//   - 账户管理（创建/查询/列表/状态更新）
//   - 积分分配（上级→下级，含余额校验）
//   - 采购/充值（增加余额+记录）
//   - 消费扣减（冻结→扣减→记录流水）
//   - 概览统计
//   - 预警配置
//
// 核心流程：
//   采购 → 区域账户充值 → 分配到学校账户 → 分配到个人账户 → AI调用消费
//   AI调用消费流程：预估→冻结→调用→扣减(或释放)→记录流水

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

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

// ==================== 账户管理 ====================

// CreateAccount 创建积分账户
// 自动校验账户类型有效性、防止重复创建
func (s *TokenService) CreateAccount(ctx context.Context, req *models.CreateTokenAccountRequest) (*models.TokenAccount, error) {
	// 校验账户类型
	if req.AccountType != models.AccountTypeRegion &&
		req.AccountType != models.AccountTypeSchool &&
		req.AccountType != models.AccountTypePersonal {
		return nil, ErrTokenInvalidAccountType
	}

	// 校验显示名不为空
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

	log.Printf("[Token] 创建账户成功: type=%s owner=%s name=%s", acc.AccountType, acc.OwnerID, acc.DisplayName)
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

	// 查预警配置
	alertCfg, _ := repository.GetTokenAlertConfig(ctx, accountID)
	detail.AlertConfig = alertCfg

	// 查子账户列表
	children, _ := repository.ListChildAccounts(ctx, accountID)
	detail.ChildAccounts = children

	return detail, nil
}

// GetAccountByOwner 根据实体类型+实体ID获取账户
func (s *TokenService) GetAccountByOwner(ctx context.Context, accountType string, ownerID string) (*models.TokenAccount, error) {
	return repository.GetTokenAccountByOwner(ctx, accountType, ownerID)
}

// ListAccounts 查询账户列表
func (s *TokenService) ListAccounts(ctx context.Context, accountType string, parentAccountID string, status string, limit int, offset int) ([]*models.TokenAccountListItem, int, error) {
	return repository.ListTokenAccounts(ctx, accountType, parentAccountID, status, limit, offset)
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

// GetSchoolAccountByAdmin 根据学校管理员用户ID查找本校积分账户
// 用于 senior_operator 数据过滤场景
func (s *TokenService) GetSchoolAccountByAdmin(ctx context.Context, adminUserID string) (*models.TokenAccount, error) {
	// 先查管理员管辖的学校
	school, err := repository.GetSchoolByAdminUserID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	// 再查该学校的积分账户
	return repository.GetTokenAccountByOwner(ctx, models.AccountTypeSchool, school.ID)
}

// ==================== 积分分配 ====================

// AllocateTokens 从上级账户分配积分到下级账户
// 流程：校验层级关系 → 扣减上级余额 → 增加下级余额 → 记录分配流水
func (s *TokenService) AllocateTokens(ctx context.Context, fromAccountID string, req *models.AllocateTokensRequest, operatorID string) error {
	if req.Amount <= 0 {
		return ErrTokenInvalidAmount
	}
	if fromAccountID == req.ToAccountID {
		return ErrTokenSelfAllocate
	}

	// 验证来源账户
	fromAcc, err := repository.GetTokenAccountByID(ctx, fromAccountID)
	if err != nil {
		return fmt.Errorf("来源账户不存在: %w", err)
	}
	if fromAcc.Status != models.AccountStatusActive {
		return ErrTokenAccountNotActive
	}

	// 验证目标账户
	toAcc, err := repository.GetTokenAccountByID(ctx, req.ToAccountID)
	if err != nil {
		return fmt.Errorf("目标账户不存在: %w", err)
	}
	if toAcc.Status != models.AccountStatusActive {
		return ErrTokenAccountNotActive
	}

	// 验证层级关系：目标的parent_account_id应等于来源ID
	// 允许的分配方向：region→school, school→personal, region→personal(跨级)
	validRelation := false
	if toAcc.ParentAccountID != nil && *toAcc.ParentAccountID == fromAccountID {
		validRelation = true // 直接父子关系
	}
	// 也允许从上级的上级分配（region→personal，只要personal的parent的parent是from）
	if !validRelation && toAcc.ParentAccountID != nil {
		parentAcc, parentErr := repository.GetTokenAccountByID(ctx, *toAcc.ParentAccountID)
		if parentErr == nil && parentAcc.ParentAccountID != nil && *parentAcc.ParentAccountID == fromAccountID {
			validRelation = true
		}
	}
	if !validRelation {
		return ErrTokenNotParentChild
	}

	// 扣减上级余额（事务内含余额检查）
	if err := repository.DeductBalanceForAllocation(ctx, fromAccountID, req.Amount); err != nil {
		return err
	}

	// 增加下级余额
	if err := repository.AddBalance(ctx, req.ToAccountID, req.Amount); err != nil {
		// 尝试回滚上级扣减（尽力而为）
		_ = repository.AddBalance(ctx, fromAccountID, req.Amount)
		return fmt.Errorf("增加下级余额失败: %w", err)
	}

	// 记录分配流水
	alloc := &models.TokenAllocation{
		FromAccountID:  fromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		AllocationType: models.AllocationTypeManual,
		Memo:           req.Memo,
		OperatorID:     operatorID,
	}
	if err := repository.CreateTokenAllocation(ctx, alloc); err != nil {
		log.Printf("[Token] 分配成功但记录流水失败: from=%s to=%s amount=%d err=%v",
			fromAccountID, req.ToAccountID, req.Amount, err)
	}

	log.Printf("[Token] 分配成功: from=%s to=%s amount=%d operator=%s",
		fromAccountID, req.ToAccountID, req.Amount, operatorID)
	return nil
}

// ListAllocations 查询分配记录
func (s *TokenService) ListAllocations(ctx context.Context, fromAccountID string, toAccountID string, limit int, offset int) ([]*models.AllocationListItem, int, error) {
	return repository.ListTokenAllocations(ctx, fromAccountID, toAccountID, limit, offset)
}

// ==================== 采购/充值 ====================

// PurchaseTokens 采购/充值积分
// 流程：校验 → 增加账户余额 → 记录采购流水
func (s *TokenService) PurchaseTokens(ctx context.Context, req *models.PurchaseTokensRequest, operatorID string) error {
	if req.Amount <= 0 {
		return ErrTokenInvalidAmount
	}

	// 验证目标账户
	acc, err := repository.GetTokenAccountByID(ctx, req.AccountID)
	if err != nil {
		return fmt.Errorf("目标账户不存在: %w", err)
	}
	if acc.Status != models.AccountStatusActive {
		return ErrTokenAccountNotActive
	}

	// 解析有效期
	var validUntil *time.Time
	if req.ValidUntil != nil && *req.ValidUntil != "" {
		t, parseErr := time.Parse(time.RFC3339, *req.ValidUntil)
		if parseErr == nil {
			validUntil = &t
		}
	}

	// 增加账户余额
	if err := repository.AddBalance(ctx, req.AccountID, req.Amount); err != nil {
		return err
	}

	// 记录采购流水
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
		log.Printf("[Token] 充值成功但记录采购流水失败: account=%s amount=%d err=%v",
			req.AccountID, req.Amount, err)
	}

	log.Printf("[Token] 充值成功: account=%s amount=%d type=%s operator=%s",
		req.AccountID, req.Amount, req.PurchaseType, operatorID)
	return nil
}

// ListPurchases 查询采购记录
func (s *TokenService) ListPurchases(ctx context.Context, accountID string, limit int, offset int) ([]*models.PurchaseListItem, int, error) {
	return repository.ListTokenPurchases(ctx, accountID, limit, offset)
}

// ==================== 消费流程（AI调用时使用）====================

// ConsumeTokens 消费积分（AI调用完成后调用）
// v129改造：从固定汇率改为精确模式
//
// 对齐AOCI: credits.go 的 ConsumeCredits
// 行为：
//   - Calculation为nil且TokensUsed<=0 → 跳过
//   - 无账户 → 静默跳过（不阻断AI调用，对齐AOCI"无账户视为无限额度"）
//   - 账户不活跃 → 跳过
//   - 直接扣减余额（允许透支，对齐AOCI不做余额预检查）
//   - 记录完整消费流水（含9个精确计算字段）
func (s *TokenService) ConsumeTokens(ctx context.Context, req *models.TokenConsumeRequest) error {
	// 确定消费积分数
	var creditCost float64
	var calc *models.CreditCalculation

	if req.Calculation != nil && req.Calculation.CreditsConsumed > 0 {
		// v129新路径：使用精确计算结果
		creditCost = req.Calculation.CreditsConsumed
		calc = req.Calculation
	} else if req.TokensUsed > 0 {
		// 兜底：没有精确计算时（不应发生，但防御性编程）
		creditCost = float64(req.TokensUsed) / 1000.0
		if creditCost <= 0 {
			return nil
		}
	} else {
		return nil // 无消费
	}

	// 查找用户的个人账户
	acc, err := repository.GetTokenAccountByOwner(ctx, models.AccountTypePersonal, req.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrTokenAccountNotFound) {
			// 无账户静默跳过（对齐AOCI：无账户视为无限额度）
			return nil
		}
		return fmt.Errorf("查询用户积分账户失败: %w", err)
	}

	// 账户不活跃，跳过
	if acc.Status != models.AccountStatusActive {
		return nil
	}

	// 记录消费前余额
	balanceBefore := acc.Balance

	// 扣减余额（允许透支，对齐AOCI：不做余额预检查，闸门已前置检查过）
	if err := repository.DirectDeductBalance(ctx, acc.ID, creditCost); err != nil {
		log.Printf("[Token] 消费扣减失败(不阻断AI): account=%s amount=%.4f err=%v", acc.ID, creditCost, err)
		return nil // 不阻断AI调用
	}

	// 记录消费流水（含完整计算过程）
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
	// 填充v129精确计算字段
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
		log.Printf("[Token] 记录消费流水失败: account=%s amount=%.4f err=%v", acc.ID, creditCost, err)
	}

	return nil
}

// ListConsumptionLogs 查询消费流水
func (s *TokenService) ListConsumptionLogs(ctx context.Context, accountID string, userID string, sceneCode string, limit int, offset int) ([]*models.ConsumptionListItem, int, error) {
	return repository.ListTokenConsumptionLogs(ctx, accountID, userID, sceneCode, limit, offset)
}

// ==================== 统计 ====================

// GetOverviewStats 获取Token系统概览统计
func (s *TokenService) GetOverviewStats(ctx context.Context) (*models.TokenOverviewStats, error) {
	return repository.GetTokenOverviewStats(ctx)
}

// ==================== 预警配置 ====================

// GetAlertConfig 获取预警配置
func (s *TokenService) GetAlertConfig(ctx context.Context, accountID string) (*models.TokenAlertConfig, error) {
	return repository.GetTokenAlertConfig(ctx, accountID)
}

// UpdateAlertConfig 更新预警配置
func (s *TokenService) UpdateAlertConfig(ctx context.Context, accountID string, req *models.UpdateAlertConfigRequest) error {
	// 校验阈值合理性
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
