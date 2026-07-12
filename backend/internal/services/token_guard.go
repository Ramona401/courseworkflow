package services

// token_guard.go — Token积分前置检查守卫
//
// v128 新增（阶段C · Token/积分系统）
// v129 改造（积分机制融合 · 对齐AOCI精确积分计算）：
//   - 删除 sceneEstimatedCredits 固定预估映射表
//   - 改为动态检查：available > 0 即放行（允许最后一次透支，对齐AOCI）
//   - 无账户视为无限额度（未开通积分系统的用户不受限）
//   - 查询失败降级放行
//   - 账户已冻结时拒绝
// v141 改进：log.Printf → logger.WithModule 结构化日志
//
// 积分硬闸 batch 改造（本次，2026-07-04）：
//   背景：存量初始积分已全量发放（374个活跃非admin用户人人有个人账户，
//   郯城200/其他100），"允许透支/无账户不受限"的宽松期结束，正式收紧为硬闸。
//   三点变化：
//   1. 无账户 → 由"无限放行"改为【拒绝】。存量已全部补齐、新用户建号时自动
//      建账户（best-effort），从此"无账户"属于异常状态，继续放行即计费漏洞。
//   2. 可用余额 <= 0 → 拒绝（此判断 v129 就有），文案升级为带指引的
//      "积分余额不足，请联系学校管理员分配积分"——学校管理员/区域管理员
//      已有一键批量分配入口，指引可闭环。
//      注：available > 0 即放行的"允许最后一次透支"语义保留——余额 0.5 的
//      用户仍能发起一次消费 3 积分的调用打到 -2.5，下次才被拦。这是刻意的：
//      前置预估任何场景的精确消费不可靠，透支一次的成本远小于误拦的体验伤害。
//   3. admin 豁免：系统管理员不受积分限制（其个人账户仅作消费留痕）。
//      角色查询失败时【不豁免】、继续走正常余额检查——豁免是特权，
//      特权判定必须 fail-closed，不给数据库抖动留提权口子。
//   不变：账户/角色查询瞬时失败 → 降级放行（可用性优先，不因数据库抖动
//   让全站AI瘫痪；豁免判定除外，见上）。账户冻结 → 拒绝（文案带指引）。
//
// 调用链：routes.go 闭包注入 → ai.SetCreditHook(checkHook) →
//   CallAI / CallAIStream 发请求前 invokeCreditCheck → 不放行则直接返回错误
//  （流式场景连SSE上游连接都不建立），Message 原样透传到前端。
//
// 对齐AOCI: ai_proxy.go 的 checkCreditsGate + credits.go 的 HasAvailableCredits

import (
	"context"
	"errors"

	"tedna/internal/database"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// 模块日志
var tgLog = logger.WithModule("token_guard")

// ==================== TokenGuard 结构体 ====================

// TokenGuard Token积分前置检查守卫
type TokenGuard struct {
	// enabled 是否启用积分检查（false时所有检查直接放行）
	enabled bool
}

// NewTokenGuard 创建TokenGuard实例
// enabled: 是否启用积分检查
func NewTokenGuard(enabled bool) *TokenGuard {
	return &TokenGuard{
		enabled: enabled,
	}
}

// ==================== 核心方法 ====================

// CheckBalance 检查用户积分余额（积分硬闸版）
//
// 行为规则：
//   - enabled=false     → 直接放行
//   - admin 角色        → 豁免放行（角色查询失败时不豁免，继续正常检查）
//   - 无账户            → 【拒绝】（硬闸：存量已全量发放，无账户属异常状态）
//   - 账户查询瞬时失败  → 降级放行（可用性优先）
//   - 账户已冻结        → 拒绝
//   - available > 0     → 放行（允许最后一次透支）
//   - available <= 0    → 拒绝，文案指引联系学校管理员分配
func (g *TokenGuard) CheckBalance(ctx context.Context, userID string) *models.TokenBalanceCheckResult {
	result := &models.TokenBalanceCheckResult{
		HasBalance: true,
		Message:    "余额充足",
	}

	// 未启用积分检查，直接放行
	if !g.enabled {
		return result
	}

	// ---- admin 豁免（fail-closed：查询失败不豁免，落入正常余额检查）----
	// 用 id::text 匹配规避 uuid/text 参数类型歧义；users 表仅数百行，单行查询开销可忽略。
	var role string
	if err := database.DB.QueryRow(ctx,
		`SELECT role FROM users WHERE id::text = $1`, userID).Scan(&role); err != nil {
		tgLog.Warn("查询用户角色失败（不豁免，继续正常余额检查）",
			"user_id", userID, "error", err)
	} else if role == models.RoleAdmin {
		result.Message = "系统管理员豁免"
		return result
	}

	// ---- 查找用户个人账户 ----
	acc, err := repository.GetTokenAccountByOwner(ctx, models.AccountTypePersonal, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTokenAccountNotFound) {
			// 硬闸：无账户 → 拒绝（v129 的"无账户不受限"宽松期已结束）
			// 正常情况下不应出现：存量已全量补齐，新用户建号时自动建账户。
			// 出现即说明自动建账户失败过，需管理员介入，放行即计费漏洞。
			tgLog.Warn("用户无个人积分账户，拦截AI调用", "user_id", userID)
			result.HasBalance = false
			result.Message = "积分账户未开通，请联系系统管理员"
			return result
		}
		// 账户查询瞬时失败 → 降级放行（可用性优先，不因数据库抖动瘫痪全站AI）
		tgLog.Warn("查询用户账户失败（降级放行）", "user_id", userID, "error", err)
		result.Message = "查询失败，默认放行"
		return result
	}

	// ---- 账户已冻结 → 拒绝 ----
	if acc.Status != models.AccountStatusActive {
		result.HasAccount = true
		result.HasBalance = false
		result.AccountID = acc.ID
		result.Available = 0
		result.Message = "积分账户已冻结，请联系系统管理员"
		return result
	}

	// ---- 计算可用余额 ----
	available := acc.Balance - acc.FrozenAmount
	result.HasAccount = true
	result.AccountID = acc.ID
	result.Available = available

	// available > 0 即放行（允许最后一次透支）；<= 0 拒绝并指引
	if available <= 0 {
		result.HasBalance = false
		result.Message = "积分余额不足，请联系学校管理员分配积分"
	}

	return result
}

// IsEnabled 返回守卫是否启用
func (g *TokenGuard) IsEnabled() bool {
	return g.enabled
}
