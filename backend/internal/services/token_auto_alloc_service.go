package services

// token_auto_alloc_service.go — Token 积分自动分配·核心业务逻辑（2026-07-04 新增）
//
// ============================ 功能总览 ============================
// 本文件实现"用完自动补 / 月初补足"两大自动分配机制，让老师积分用尽时
// 只要学校池有余额即自动补齐，减少管理员手动分配负担。全部逻辑受一个全局
// 开关（ai_configs.token_auto_allocation_enabled）统一控制，可一键停用。
//
// 五条业务规则（与产品确认）：
//   规则B 每月上限：每位老师每月从学校池领取的额度上限 = monthlyAllocationCap(500)，
//                   跨自然月清零。计量 = 当月 auto+monthly 类型分配之和。
//   规则C 自动补  ：老师某次消费后余额 < autoRefillThreshold(5) 时，从其所在学校池
//                   补足到 personalTargetCredits(100)，受"每月剩余额度"与"学校池余额"双重截断。
//   规则D 月初补足：每月1号00:00，把所有 active 个人账户补足到100（余额≥100不动），
//                   同样受每月剩余额度与学校池余额约束。
//   规则E 全局开关：关闭后 规则C/D 全部停用。
//   规则F 池不足提醒：补款后若学校池余额 < 本校当月消费 × poolLowRatio(10%)，
//                   通知该校所在区域的区域管理员前来充值；同校每天最多1条。
//
// ============================ 安全设计 ============================
//  1. best-effort：所有自动补逻辑任何一步出错只记 Warn 日志，绝不阻断 AI 主流程
//     （与 ConsumeTokens 的 fail-open 风格一致——积分是辅助，AI 可用性优先）。
//  2. 原子扣加：扣学校池 + 加个人账户复用经验证的 DeductBalanceForAllocation
//     （带 FOR UPDATE 行锁 + 余额校验）+ AddBalance，加失败自动回滚扣款，钱不丢不多。
//  3. fail-closed 防超发：当月已领查询失败时不补（宁可少补也不超发500上限）。
//  4. 多级父链：老师→(组)→学校，自动补沿父链向上找到第一个 school 账户作为扣款池。

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 业务常量（改这里即改规则参数）====================
const (
	autoRefillThreshold   = 5.0   // 规则C 触发阈值：余额低于此值触发自动补
	personalTargetCredits = 100.0 // 补足目标：把老师余额补到这个值
	monthlyAllocationCap  = 500.0 // 规则B 每月每人从学校池领取上限
	poolLowRatio          = 0.10  // 规则F 学校池余额低于"本校当月消费×此比例"时提醒区域管理员
	maxParentClimb        = 5     // 沿父链向上找学校池的最大层数（防环）

	autoAllocSwitchKey = "token_auto_allocation_enabled" // 全局开关配置键
	systemOperatorID   = "00000000-0000-0000-0000-000000000000"
)

// ==================== 规则F 通知去重（每校每天最多1条）====================
// 内存 map 记录"某学校账户今天已通知过区域管理员"。key=学校账户ID，value=日期(2006-01-02)。
// 进程重启会重置（可接受：最坏情况重启当天多发1条，不影响正确性）。
var (
	poolLowNotifyMu   sync.Mutex
	poolLowNotifyDate = map[string]string{}
)

// shouldNotifyPoolLow 判断某学校今天是否还可以发"池不足"通知（true=可发并登记，false=今天已发过）。
func shouldNotifyPoolLow(schoolAccID string) bool {
	poolLowNotifyMu.Lock()
	defer poolLowNotifyMu.Unlock()
	today := time.Now().Format("2006-01-02")
	if poolLowNotifyDate[schoolAccID] == today {
		return false
	}
	poolLowNotifyDate[schoolAccID] = today
	return true
}

// ==================== 全局开关 ====================
// autoAllocEnabled 读 ai_configs.token_auto_allocation_enabled。
// 读失败或未配置 → 默认 true（功能默认开启；仅显式设为 "false" 才关闭）。
func autoAllocEnabled() bool {
	cfg, err := repository.GetConfigByKey(autoAllocSwitchKey)
	if err != nil || cfg == nil {
		return true
	}
	return cfg.ConfigValue != "false"
}

// ==================== 父链向上找学校池 ====================
// findSchoolPoolAccount 从个人账户沿 ParentAccountID 向上，找到第一个 account_type='school' 的账户。
// 返回 (学校账户, true) 或 (nil, false)（无学校池：游离账户 / 父链断裂 / 超过爬升上限）。
func findSchoolPoolAccount(ctx context.Context, personalAcc *models.TokenAccount) (*models.TokenAccount, bool) {
	if personalAcc == nil || personalAcc.ParentAccountID == nil || *personalAcc.ParentAccountID == "" {
		return nil, false
	}
	currentID := *personalAcc.ParentAccountID
	for i := 0; i < maxParentClimb; i++ {
		acc, err := repository.GetTokenAccountByID(ctx, currentID)
		if err != nil || acc == nil {
			return nil, false
		}
		if acc.AccountType == models.AccountTypeSchool {
			return acc, true
		}
		if acc.ParentAccountID == nil || *acc.ParentAccountID == "" {
			return nil, false
		}
		currentID = *acc.ParentAccountID
	}
	return nil, false
}

// ==================== 孪生原子分配（打 auto 标记）====================
// allocateAuto 从学校池扣 amount、加到个人账户、写一条 auto 类型流水，原子且可回滚。
// 逻辑镜像 AllocateTokens，仅两点不同：流水类型固定 auto；不做父子强校验
// （调用方已通过 findSchoolPoolAccount 保证 school 在个人账户父链上）。
// amount 必须 > 0 且已由调用方 clamp 到不超过学校池余额与每月剩余额度。
// 返回 error 表示分配失败（如学校池并发被扣光触发 ErrInsufficientBalance），调用方据此处理。
func allocateAuto(ctx context.Context, schoolAccID, personalAccID string, amount float64) error {
	if amount <= 0 {
		return nil
	}
	// 1. 扣学校池（带 FOR UPDATE 锁 + available<amount 校验，池不足返回 ErrInsufficientBalance）
	if err := repository.DeductBalanceForAllocation(ctx, schoolAccID, amount); err != nil {
		return err
	}
	// 2. 加个人账户；失败则退回学校池扣款（与 AllocateTokens 同款回滚）
	if err := repository.AddBalance(ctx, personalAccID, amount); err != nil {
		_ = repository.AddBalance(ctx, schoolAccID, amount)
		return fmt.Errorf("自动补：增加老师余额失败(已回滚扣款): %w", err)
	}
	// 3. 写 auto 类型流水（best-effort：失败仅 Warn，钱已到账不回滚）
	alloc := &models.TokenAllocation{
		FromAccountID:  schoolAccID,
		ToAccountID:    personalAccID,
		Amount:         amount,
		AllocationType: models.AllocationTypeAuto,
		Memo:           "系统自动补足积分",
		OperatorID:     systemOperatorID,
	}
	if err := repository.CreateTokenAllocation(ctx, alloc); err != nil {
		tokenLog.Warn("自动补成功但记录流水失败",
			"from", schoolAccID, "to", personalAccID, "amount", amount, "error", err)
	}
	return nil
}

// ==================== 补足额计算（三重截断）====================
// computeRefillAmount 计算本次应补多少：min(补到100所需, 每月剩余额度, 学校池可用余额)。
// 返回值 ≤ 0 表示无需补 / 无额度可补 / 池空。第二返回值 shortfall=true 表示
// "本可补更多但被学校池截断了"（用于触发规则F池不足提醒）。
func computeRefillAmount(currentBalance, monthlyUsed, poolBalance float64) (amount float64, shortfall bool) {
	need := personalTargetCredits - currentBalance // 补到100还差多少
	if need <= 0 {
		return 0, false
	}
	remainingQuota := monthlyAllocationCap - monthlyUsed // 本月还能领多少
	if remainingQuota <= 0 {
		return 0, false // 已达每月上限，不补（前端守卫会提示联系管理员）
	}
	want := need
	if want > remainingQuota {
		want = remainingQuota
	}
	// 学校池截断
	if want > poolBalance {
		if poolBalance <= 0 {
			return 0, true // 池空，一分补不了，且确实还需要 → 触发提醒
		}
		return poolBalance, true // 只能补池里剩的，不足期望 → 触发提醒
	}
	return want, false
}

// ==================== 规则C：消费后自动补 ====================
// TryAutoRefill 在 ConsumeTokens 扣费成功后调用。personalAcc 为扣费的个人账户，
// balanceAfter 为扣费后余额。全 best-effort，任何异常只 Warn，不返回错误、不阻断调用方。
func TryAutoRefill(ctx context.Context, personalAcc *models.TokenAccount, balanceAfter float64) {
	defer func() {
		if r := recover(); r != nil {
			tokenLog.Warn("自动补 panic 已恢复(不影响主流程)", "recover", r)
		}
	}()

	if !autoAllocEnabled() {
		return
	}
	if balanceAfter >= autoRefillThreshold {
		return // 余额还够，未触发
	}
	if personalAcc == nil || personalAcc.AccountType != models.AccountTypePersonal {
		return
	}

	// 找学校池（沿父链向上）
	schoolAcc, ok := findSchoolPoolAccount(ctx, personalAcc)
	if !ok {
		return // 游离账户，无学校池可补
	}

	// 当月已领（fail-closed：查询失败不补）
	monthlyUsed, err := repository.SumMonthlyAllocatedToAccount(ctx, personalAcc.ID)
	if err != nil {
		tokenLog.Warn("自动补：查询当月已领失败，跳过本次补款", "account", personalAcc.ID, "error", err)
		return
	}

	amount, shortfall := computeRefillAmount(balanceAfter, monthlyUsed, schoolAcc.Balance)
	if amount > 0 {
		if err := allocateAuto(ctx, schoolAcc.ID, personalAcc.ID, amount); err != nil {
			tokenLog.Warn("自动补：分配失败", "school", schoolAcc.ID, "teacher", personalAcc.ID, "amount", amount, "error", err)
			shortfall = true // 分配失败(多为池被并发扣空)也视作池不足
		} else {
			tokenLog.Info("自动补成功", "teacher", personalAcc.ID, "amount", amount, "balance_after", balanceAfter+amount)
		}
	}

	// 规则F：补后检查学校池是否见底，见底则提醒区域管理员
	maybeNotifyRegionAdminPoolLow(ctx, schoolAcc.ID, shortfall)
}

// ==================== 规则F：学校池不足提醒区域管理员 ====================
// maybeNotifyRegionAdminPoolLow 判断学校池是否"快见底"，是则通知该校区域的区域管理员。
// 触发条件：forceShortfall(补款被池截断/失败) 或 池余额 < 本校当月消费×poolLowRatio。
// 去重：同校每天最多1条。全 best-effort。
func maybeNotifyRegionAdminPoolLow(ctx context.Context, schoolAccID string, forceShortfall bool) {
	schoolAcc, err := repository.GetTokenAccountByID(ctx, schoolAccID)
	if err != nil || schoolAcc == nil {
		return
	}
	// 本校当月消费（用学校账户维度的消费汇总）
	_, monthConsumed, _, _ := repository.GetUserConsumptionSummary(ctx, schoolAccID)
	threshold := monthConsumed * poolLowRatio

	poolLow := forceShortfall || schoolAcc.Balance < threshold
	if !poolLow {
		return
	}
	if !shouldNotifyPoolLow(schoolAccID) {
		return // 今天已通知过该校
	}

	// schoolAcc.OwnerID 是学校组织ID → 找该校区域的区域管理员
	adminIDs, err := repository.ListRegionAdminUserIDsBySchool(ctx, schoolAcc.OwnerID)
	if err != nil || len(adminIDs) == 0 {
		return // 无区域管理员可通知（游离学校 / 区域未任命管理员）
	}

	title := fmt.Sprintf("学校「%s」积分池不足", schoolAcc.DisplayName)
	body := fmt.Sprintf("学校「%s」的积分池余额已不足，老师用完积分将无法自动补齐。请及时为该校充值积分。当前池余额约 %.0f。",
		schoolAcc.DisplayName, schoolAcc.Balance)

	if GlobalNotificationService != nil {
		GlobalNotificationService.EmitNotificationBatch(adminIDs, models.EmitNotificationInput{
			Type:       models.NotifTokenSchoolPoolLow,
			Title:      title,
			Body:       body,
			EntityType: models.NotifEntityTokenAccount,
			EntityID:   schoolAccID,
			Link:       "/token",
		})
	}
	tokenLog.Info("已提醒区域管理员学校池不足", "school", schoolAcc.DisplayName, "admins", len(adminIDs))
}

// ==================== 规则D：月初补足（定时任务入口）====================
// RunMonthEndTopUp 每月1号00:00由定时器调用：把所有 active 个人账户补足到100。
// 遍历全部 personal 账户，逐个应用与规则C相同的补足逻辑（余额≥100跳过）。
// 全 best-effort：单个账户失败不影响其余；返回处理统计供日志。
func RunMonthEndTopUp(ctx context.Context) {
	if !autoAllocEnabled() {
		tokenLog.Info("月初补足：全局开关关闭，跳过")
		return
	}
	tokenLog.Info("月初补足任务开始")

	// 分页遍历所有 active personal 账户
	const pageSize = 200
	offset := 0
	processed, refilled := 0, 0
	for {
		accounts, _, err := repository.ListTokenAccounts(ctx, models.AccountTypePersonal, "", models.AccountStatusActive, "", nil, pageSize, offset)
		if err != nil {
			tokenLog.Warn("月初补足：分页查询账户失败", "offset", offset, "error", err)
			break
		}
		if len(accounts) == 0 {
			break
		}
		for _, item := range accounts {
			processed++
			if topUpOneAccount(ctx, item.ID) {
				refilled++
			}
		}
		if len(accounts) < pageSize {
			break
		}
		offset += pageSize
	}
	tokenLog.Info("月初补足任务完成", "processed", processed, "refilled", refilled)
}

// topUpOneAccount 对单个个人账户执行月初补足，返回是否实际补了款。
func topUpOneAccount(ctx context.Context, personalAccID string) bool {
	defer func() {
		if r := recover(); r != nil {
			tokenLog.Warn("月初补足单账户 panic 已恢复", "account", personalAccID, "recover", r)
		}
	}()

	acc, err := repository.GetTokenAccountByID(ctx, personalAccID)
	if err != nil || acc == nil || acc.Status != models.AccountStatusActive {
		return false
	}
	if acc.Balance >= personalTargetCredits {
		return false // 没怎么用，配额还在，不动
	}
	schoolAcc, ok := findSchoolPoolAccount(ctx, acc)
	if !ok {
		return false
	}
	monthlyUsed, err := repository.SumMonthlyAllocatedToAccount(ctx, acc.ID)
	if err != nil {
		return false
	}
	amount, shortfall := computeRefillAmount(acc.Balance, monthlyUsed, schoolAcc.Balance)
	did := false
	if amount > 0 {
		if err := allocateAuto(ctx, schoolAcc.ID, acc.ID, amount); err != nil {
			shortfall = true
		} else {
			did = true
		}
	}
	maybeNotifyRegionAdminPoolLow(ctx, schoolAcc.ID, shortfall)
	return did
}
