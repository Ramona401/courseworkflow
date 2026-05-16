package repository

// token_allocation_repo.go — Token积分系统数据访问层（分配记录+消费流水）
//
// v128 新增（阶段C · Token/积分系统）：
//   - 分配记录 CRUD
//   - 消费流水记录 + 查询
//
// 对应数据库表：token_allocations / token_consumption_logs

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 分配记录 ====================

// CreateTokenAllocation 创建分配记录
func CreateTokenAllocation(ctx context.Context, alloc *models.TokenAllocation) error {
	err := database.DB.QueryRow(ctx, `
		INSERT INTO token_allocations
			(from_account_id, to_account_id, amount, allocation_type, memo, operator_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`,
		alloc.FromAccountID, alloc.ToAccountID, alloc.Amount,
		alloc.AllocationType, alloc.Memo, alloc.OperatorID,
	).Scan(&alloc.ID, &alloc.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建分配记录失败: %w", err)
	}
	return nil
}

// ListTokenAllocations 查询分配记录列表（支持按来源/目标账户筛选）
func ListTokenAllocations(ctx context.Context, fromAccountID string, toAccountID string, limit int, offset int) ([]*models.AllocationListItem, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if fromAccountID != "" {
		where += fmt.Sprintf(" AND a.from_account_id = $%d", argIdx)
		args = append(args, fromAccountID)
		argIdx++
	}
	if toAccountID != "" {
		where += fmt.Sprintf(" AND a.to_account_id = $%d", argIdx)
		args = append(args, toAccountID)
		argIdx++
	}

	// 统计总数
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM token_allocations a WHERE %s`, where)
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计分配记录数失败: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	// 分页查询（关联账户和用户获取名称）
	listQuery := fmt.Sprintf(`
		SELECT a.id,
		       COALESCE(fa.display_name, '') AS from_account_name,
		       COALESCE(ta.display_name, '') AS to_account_name,
		       a.amount, a.allocation_type, a.memo,
		       COALESCE(u.display_name, '') AS operator_name,
		       a.created_at
		FROM token_allocations a
		LEFT JOIN token_accounts fa ON fa.id = a.from_account_id
		LEFT JOIN token_accounts ta ON ta.id = a.to_account_id
		LEFT JOIN users u ON u.id = a.operator_id
		WHERE %s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询分配记录列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.AllocationListItem
	for rows.Next() {
		item := &models.AllocationListItem{}
		err := rows.Scan(
			&item.ID, &item.FromAccountName, &item.ToAccountName,
			&item.Amount, &item.AllocationType, &item.Memo,
			&item.OperatorName, &item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描分配记录行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, total, nil
}

// ==================== 消费流水 ====================

// CreateTokenConsumptionLog 创建消费流水记录
func CreateTokenConsumptionLog(ctx context.Context, log *models.TokenConsumptionLog) error {
	// v129变更：新增9个精确积分计算字段
	err := database.DB.QueryRow(ctx, `
		INSERT INTO token_consumption_logs
			(account_id, user_id, amount, balance_before, balance_after,
			 scene_code, model_used, tokens_used, lesson_plan_id, pipeline_id, memo,
			 input_tokens, output_tokens, model_name, provider,
			 cost_usd, exchange_rate, multiplier, credits_consumed, latency_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id, created_at
	`,
		log.AccountID, log.UserID, log.Amount, log.BalanceBefore, log.BalanceAfter,
		log.SceneCode, log.ModelUsed, log.TokensUsed, log.LessonPlanID, log.PipelineID, log.Memo,
		log.InputTokens, log.OutputTokens, log.ModelName, log.Provider,
		log.CostUSD, log.ExchangeRate, log.Multiplier, log.CreditsConsumed, log.LatencyMs,
	).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建消费流水失败: %w", err)
	}
	return nil
}

// ListTokenConsumptionLogs 查询消费流水列表（支持按账户、用户、场景筛选）
func ListTokenConsumptionLogs(ctx context.Context, accountID string, userID string, sceneCode string, limit int, offset int) ([]*models.ConsumptionListItem, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if accountID != "" {
		where += fmt.Sprintf(" AND cl.account_id = $%d", argIdx)
		args = append(args, accountID)
		argIdx++
	}
	if userID != "" {
		where += fmt.Sprintf(" AND cl.user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if sceneCode != "" {
		where += fmt.Sprintf(" AND cl.scene_code = $%d", argIdx)
		args = append(args, sceneCode)
		argIdx++
	}

	// 统计总数
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM token_consumption_logs cl WHERE %s`, where)
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计消费流水数失败: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	// 分页查询（关联账户和用户获取名称）
	listQuery := fmt.Sprintf(`
		SELECT cl.id,
		       COALESCE(ta.display_name, '') AS account_name,
		       COALESCE(u.display_name, '') AS user_name,
		       cl.amount, cl.balance_before, cl.balance_after,
		       cl.scene_code, cl.model_used, cl.tokens_used,
		       cl.memo, cl.created_at,
		       cl.input_tokens, cl.output_tokens, cl.model_name, cl.provider,
		       cl.cost_usd, cl.exchange_rate, cl.multiplier, cl.credits_consumed, cl.latency_ms
		FROM token_consumption_logs cl
		LEFT JOIN token_accounts ta ON ta.id = cl.account_id
		LEFT JOIN users u ON u.id = cl.user_id
		WHERE %s
		ORDER BY cl.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询消费流水列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ConsumptionListItem
	for rows.Next() {
		item := &models.ConsumptionListItem{}
		err := rows.Scan(
			&item.ID, &item.AccountName, &item.UserName,
			&item.Amount, &item.BalanceBefore, &item.BalanceAfter,
			&item.SceneCode, &item.ModelUsed, &item.TokensUsed,
			&item.Memo, &item.CreatedAt,
			&item.InputTokens, &item.OutputTokens, &item.ModelName, &item.Provider,
			&item.CostUSD, &item.ExchangeRate, &item.Multiplier, &item.CreditsConsumed, &item.LatencyMs,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描消费流水行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, total, nil
}

// GetUserConsumptionSummary 获取用户消费汇总（今日+本月+总计）
func GetUserConsumptionSummary(ctx context.Context, accountID string) (todayAmount float64, monthAmount float64, totalAmount float64, err error) {
	// 今日消费
	_ = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM token_consumption_logs
		 WHERE account_id = $1 AND created_at >= CURRENT_DATE`,
		accountID,
	).Scan(&todayAmount)

	// 本月消费
	_ = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM token_consumption_logs
		 WHERE account_id = $1 AND created_at >= date_trunc('month', CURRENT_DATE)`,
		accountID,
	).Scan(&monthAmount)

	// 总消费
	_ = database.DB.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM token_consumption_logs
		 WHERE account_id = $1`,
		accountID,
	).Scan(&totalAmount)

	return todayAmount, monthAmount, totalAmount, nil
}
