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
//
// 对齐AOCI: ai_proxy.go 的 checkCreditsGate + credits.go 的 HasAvailableCredits

import (
	"context"
	"errors"
	"log"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== TokenGuard 结构体 ====================

// TokenGuard Token积分前置检查守卫
// 对齐AOCI: token_guard.go（简化为available > 0即放行）
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

// CheckBalance 检查用户积分余额
// 对齐AOCI: credits.go 的 HasAvailableCredits 三元组设计
//
// 行为规则（对齐AOCI）：
//   - enabled=false → 直接放行
//   - 无账户 → 无限额度（未开通不受限）
//   - 查询失败 → 降级放行
//   - 账户已冻结 → 拒绝
//   - available > 0 → 放行（允许最后一次透支）
//   - available <= 0 → 拒绝
func (g *TokenGuard) CheckBalance(ctx context.Context, userID string) *models.TokenBalanceCheckResult {
	result := &models.TokenBalanceCheckResult{
		HasBalance: true,
		Message:    "余额充足",
	}

	// 未启用积分检查，直接放行
	if !g.enabled {
		return result
	}

	// 查找用户个人账户
	acc, err := repository.GetTokenAccountByOwner(ctx, models.AccountTypePersonal, userID)
	if err != nil {
		if errors.Is(err, repository.ErrTokenAccountNotFound) {
			// 无账户视为无限额度（对齐AOCI：未开通积分系统的用户不受限）
			result.Message = "未开通积分账户，不受限"
			return result
		}
		// 查询失败降级放行（对齐AOCI：查询失败不阻断）
		log.Printf("[TokenGuard] 查询用户账户失败(放行): user=%s err=%v", userID, err)
		result.Message = "查询失败，默认放行"
		return result
	}

	// 账户已冻结 → 拒绝
	if acc.Status != models.AccountStatusActive {
		result.HasAccount = true
		result.HasBalance = false
		result.AccountID = acc.ID
		result.Available = 0
		result.Message = "积分账户已冻结"
		return result
	}

	// 计算可用余额
	available := acc.Balance - acc.FrozenAmount
	result.HasAccount = true
	result.AccountID = acc.ID
	result.Available = available

	// 对齐AOCI: available > 0 即放行（允许最后一次透支）
	if available <= 0 {
		result.HasBalance = false
		result.Message = "积分余额不足"
	}

	return result
}

// IsEnabled 返回守卫是否启用
func (g *TokenGuard) IsEnabled() bool {
	return g.enabled
}
