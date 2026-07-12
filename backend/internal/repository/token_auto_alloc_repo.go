package repository

// token_auto_alloc_repo.go — Token 积分自动分配·数据访问层（2026-07-04 新增）
//
// 定位：承载"用完自动补 / 月底补足"机制专用的数据查询，与通用的
//   token_allocation_repo.go / token_account_repo.go 解耦，便于该功能独立演进。
//
// 目前仅一个查询：SumMonthlyAllocatedToAccount（当月已领额度），
//   是规则B"每月500上限"约束的计量基础。后续该机制若需更多专用查询，统一加在本文件。

import (
	"context"
	"fmt"

	"tedna/internal/database"
)

// SumMonthlyAllocatedToAccount 统计某账户"当月已领取"的自动分配额度（auto + monthly 两类之和）
//
// 用途（规则B 每月500上限约束的计量基础）：
//
//	每位老师每月从学校池领取的额度上限为 monthlyAllocationCap（500）。
//	"当月已领" = 本自然月内该账户作为收款方(to_account_id)、
//	分配类型为 auto（用完自动补）或 monthly（月度充值/月底补足）的分配金额之和。
//
// 为什么只算 auto+monthly、不算 initial/manual：
//   - initial（新用户初始100）是一次性入职福利，不占用每月500额度；
//   - manual（管理员手动分配）是管理员主动兜底，不应受自动机制的500上限约束，故不计入；
//   - auto/monthly 才是"系统自动发放"的部分，正是500上限要管住的对象。
//
// 自然月口径：created_at >= date_trunc('month', CURRENT_DATE)，与 GetUserConsumptionSummary
//
//	的"本月消费"口径完全一致，保证每逢自然月切换自动清零重置。
//
// 返回 0 表示本月尚未通过自动机制领取过。DB 异常上抛
//
//	（调用方据此 fail-closed：查询失败时不补，宁可少补也不超发）。
func SumMonthlyAllocatedToAccount(ctx context.Context, accountID string) (float64, error) {
	if accountID == "" {
		return 0, nil
	}
	var sum float64
	err := database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM token_allocations
		 WHERE to_account_id = $1
		   AND allocation_type IN ('auto','monthly')
		   AND created_at >= date_trunc('month', CURRENT_DATE)`,
		accountID,
	).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("统计账户当月已领额度失败: %w", err)
	}
	return sum, nil
}
