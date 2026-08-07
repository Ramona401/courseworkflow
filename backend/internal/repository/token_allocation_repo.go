package repository

// token_allocation_repo.go — Token积分系统数据访问层（分配记录+消费流水）
//
// v128 新增（阶段C · Token/积分系统）：
//   - 分配记录 CRUD
//   - 消费流水记录 + 查询
//
// v172 新增（积分管理三级数据权限隔离）：
//   - ListTokenConsumptionLogs 增加 userIDs 白名单参数（消费流水按 user_id 过滤）
//   - ListTokenAllocations 增加 ownerIDs 白名单参数（分配记录按 from/to 账户 owner_id 过滤）
//   - 白名单语义（安全关键，与 token_account_repo 统一约定）：
//       * 白名单切片 == nil  → 不过滤（admin 看全部）
//       * 白名单切片 非nil但为空 → 匹配空集（fail-closed，返回0行，绝不退化为看全部）
//       * 白名单切片 非空 → 加 = ANY($n) / owner 子查询过滤
//
// 究极彻底版·A（分配记录 total 精确）新增：
//   - ListTokenAllocations 增加 excludeMonthly 参数。为 true 时加固定条件
//     AND a.allocation_type <> 'monthly'，把"月度自充值"（from=to，语义混乱）
//     从分配记录中排除，使 items 与 total 一致（根治 P3 前端过滤导致的
//     "分页器条数 ≠ 实际显示条数"不一致）。该条件不含参数、不消耗 argIdx，
//     插在 owner 白名单之前，绝不影响白名单与分页参数编号。
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

// ListTokenAllocations 查询分配记录列表（按来源/目标账户筛选 + v172 owner 白名单 + A excludeMonthly）
//
// v172：ownerIDs 白名单——仅返回 来源账户 或 目标账户 的 owner_id ∈ 白名单 的分配记录。
//   - ownerIDs == nil  → 不过滤（admin）
//   - ownerIDs 为空切片 → AND 1=0（fail-closed，返回空集）
//   - ownerIDs 非空     → AND (from/to 账户任一 owner_id = ANY($n))
//
// A：excludeMonthly==true 时加 AND a.allocation_type <> 'monthly'（固定条件，无参数）。
func ListTokenAllocations(ctx context.Context, fromAccountID string, toAccountID string, excludeMonthly bool, ownerIDs []string, limit int, offset int) ([]*models.AllocationListItem, int, error) {
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

	// A：排除月度自充值（固定条件，不含参数、不消耗 argIdx；插在 owner 白名单之前）。
	if excludeMonthly {
		where += " AND a.allocation_type <> 'monthly'"
	}

	// v172 owner_id 白名单：来源或目标账户的 owner 命中即保留
	if ownerIDs != nil {
		if len(ownerIDs) == 0 {
			where += " AND 1=0"
		} else {
			where += fmt.Sprintf(`
                                AND (
                                        a.from_account_id IN (SELECT id FROM token_accounts WHERE owner_id = ANY($%d))
                                        OR a.to_account_id IN (SELECT id FROM token_accounts WHERE owner_id = ANY($%d))
                                )`, argIdx, argIdx)
			args = append(args, ownerIDs)
			argIdx++
		}
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

// ListTokenConsumptionLogs 查询消费流水列表（支持按账户、用户、场景筛选 + v172 user_id 白名单）
//
// v172：userIDs 白名单——仅返回 cl.user_id ∈ 白名单 的流水。
//   - userIDs == nil  → 不过滤（admin）
//   - userIDs 为空切片 → AND 1=0（fail-closed，返回空集）
//   - userIDs 非空     → AND cl.user_id = ANY($n)
//
// 说明：senior_operator 传本校成员 user_id 列表；operator/viewer 传 [自己的user_id]。
func ListTokenConsumptionLogs(
        ctx context.Context,
        accountID string,
        userID string,
        sceneCode string,
        userIDs []string,
        limit int,
        offset int,
) (
        []*models.ConsumptionListItem,
        int,
        error,
) {
        where := "1=1"
        args := []interface{}{}
        argIdx := 1

        if accountID != "" {
                where += fmt.Sprintf(
                        " AND cl.account_id = $%d",
                        argIdx,
                )
                args = append(
                        args,
                        accountID,
                )
                argIdx++
        }

        if userID != "" {
                where += fmt.Sprintf(
                        " AND cl.user_id = $%d",
                        argIdx,
                )
                args = append(
                        args,
                        userID,
                )
                argIdx++
        }

        if sceneCode != "" {
                where += fmt.Sprintf(
                        " AND cl.scene_code = $%d",
                        argIdx,
                )
                args = append(
                        args,
                        sceneCode,
                )
                argIdx++
        }

        // 三态白名单：
        // nil=不限制；空切片=空集；非空=只看名单用户。
        if userIDs != nil {
                if len(userIDs) == 0 {
                        where += " AND 1=0"
                } else {
                        where += fmt.Sprintf(
                                " AND cl.user_id = ANY($%d)",
                                argIdx,
                        )
                        args = append(
                                args,
                                userIDs,
                        )
                        argIdx++
                }
        }

        var total int

        countQuery :=
                fmt.Sprintf(
                        `SELECT COUNT(*)
                         FROM token_consumption_logs cl
                         WHERE %s`,
                        where,
                )

        if err :=
                database.DB.QueryRow(
                        ctx,
                        countQuery,
                        args...,
                ).Scan(
                        &total,
                ); err != nil {
                return nil,
                        0,
                        fmt.Errorf(
                                "统计消费流水数失败: %w",
                                err,
                        )
        }

        if limit <= 0 {
                limit = 50
        }

        listQuery :=
                fmt.Sprintf(
                        `
                        SELECT
                                cl.id,
                                COALESCE(ta.display_name, '') AS account_name,
                                COALESCE(u.display_name, '') AS user_name,
                                cl.amount,
                                cl.balance_before,
                                cl.balance_after,
                                cl.scene_code,
                                cl.model_used,
                                cl.tokens_used,
                                cl.memo,
                                cl.created_at,

                                COALESCE(cl.billing_category, 'text_ai'),
                                COALESCE(cl.billing_node_code, ''),
                                COALESCE(bn.display_name, ''),
                                COALESCE(cl.media_type, ''),
                                COALESCE(cl.media_unit, ''),
                                COALESCE(cl.media_quantity, 0),

                                cl.input_tokens,
                                cl.output_tokens,
                                cl.model_name,
                                cl.provider,
                                cl.cost_usd,
                                cl.exchange_rate,
                                cl.multiplier,
                                cl.credits_consumed,
                                cl.latency_ms
                        FROM token_consumption_logs cl
                        LEFT JOIN token_accounts ta
                          ON ta.id = cl.account_id
                        LEFT JOIN users u
                          ON u.id = cl.user_id
                        LEFT JOIN token_billing_nodes bn
                          ON bn.node_code = cl.billing_node_code
                        WHERE %s
                        ORDER BY cl.created_at DESC
                        LIMIT $%d OFFSET $%d
                        `,
                        where,
                        argIdx,
                        argIdx+1,
                )

        args = append(
                args,
                limit,
                offset,
        )

        rows, err :=
                database.DB.Query(
                        ctx,
                        listQuery,
                        args...,
                )
        if err != nil {
                return nil,
                        0,
                        fmt.Errorf(
                                "查询消费流水列表失败: %w",
                                err,
                        )
        }
        defer rows.Close()

        items :=
                make(
                        []*models.ConsumptionListItem,
                        0,
                )

        for rows.Next() {
                item :=
                        &models.ConsumptionListItem{}

                if err :=
                        rows.Scan(
                                &item.ID,
                                &item.AccountName,
                                &item.UserName,
                                &item.Amount,
                                &item.BalanceBefore,
                                &item.BalanceAfter,
                                &item.SceneCode,
                                &item.ModelUsed,
                                &item.TokensUsed,
                                &item.Memo,
                                &item.CreatedAt,

                                &item.BillingCategory,
                                &item.BillingNodeCode,
                                &item.BillingNodeName,
                                &item.MediaType,
                                &item.MediaUnit,
                                &item.MediaQuantity,

                                &item.InputTokens,
                                &item.OutputTokens,
                                &item.ModelName,
                                &item.Provider,
                                &item.CostUSD,
                                &item.ExchangeRate,
                                &item.Multiplier,
                                &item.CreditsConsumed,
                                &item.LatencyMs,
                        ); err != nil {
                        return nil,
                                0,
                                fmt.Errorf(
                                        "扫描积分消费流水失败: %w",
                                        err,
                                )
                }

                items = append(
                        items,
                        item,
                )
        }

        if err := rows.Err(); err != nil {
                return nil,
                        0,
                        fmt.Errorf(
                                "遍历积分消费流水失败: %w",
                                err,
                        )
        }

        return items,
                total,
                nil
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
