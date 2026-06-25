package repository

// token_summary_repo.go — Token积分消费汇总报告数据访问层
//
// 积分消费汇总报告 batch 新增：
//   为"积分消费汇总报告"提供 6 个维度的聚合查询。所有查询均带 scope 白名单，
//   白名单三态语义与 token_account_repo / token_allocation_repo 完全一致：
//     * 白名单切片 == nil   → 不过滤（admin 看全部）
//     * 白名单切片 非nil为空 → 匹配空集（fail-closed，返回空，绝不退化为看全部）
//     * 白名单切片 非空      → 加 = ANY($n) 过滤
//
// 维度与 scope 收窄方式（安全关键，与已验证的 TokenScope 对齐）：
//   - region 维度（admin 专用）：消费→personal→school→region→区域组织，四级 JOIN。
//                  按 owner_id 收窄（grandparent region 的 owner），排除非 region。
//   - school 维度：消费→personal→school→学校组织，三级 JOIN。按 owner_id 收窄（parent school
//                  的 owner），排除 region。
//   - user/model/scene/time 维度：按 user_id 收窄（直接 cl.user_id），用 userIDs 白名单。
//   这样 admin（两白名单皆 nil）看全部；region/senior（userIDs=辖区/本校成员）看辖区/本校；
//   operator/viewer（userIDs=[自己]）只看自己。与 ① region_admin 分配同一套防线。
//
// 账户树（已确认严格三层，无断层）：
//   personal(14) → school(3) → region(2) → 顶级
//   故 school 维度取 parent(school).owner_id；region 维度取 grandparent(region).owner_id。
//
// 金额字段：统一用 credits_consumed（已盘点确认全表 credits 与 amount 一致且非零）。
//
// uuid 兜底：owner_id 是 uuid，LEFT JOIN 未匹配时为 NULL，COALESCE 兜底前须 ::text
//   （COALESCE(parent.owner_id::text,'')），否则 PostgreSQL 报 "invalid input syntax for
//   type uuid"（空字符串非合法 uuid）。region/school 维度均如此处理。
//
// 下钻过滤（可选，handler 已做 scope 内二次校验后才传入）：
//   - schoolFilter：限定到某学校的成员（handler 传入该校成员 user_id 列表，已在 scope 内）
//   - userFilter  ：限定到某个 user_id（handler 已确认该 user_id ∈ scope.UserIDs）
//
// 对应数据库表：token_consumption_logs（JOIN token_accounts / organizations / users）

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ConsumptionSummaryParams 汇总查询参数（repo 层内部用，handler 组装后传入）
type ConsumptionSummaryParams struct {
	Dimension    string    // region/school/user/model/scene/time
	From         time.Time // 时间范围起（零值表示不限）
	To           time.Time // 时间范围止（零值表示不限）
	OwnerIDs     []string  // 账户 owner 白名单（region/school 维度用；nil=不过滤/空=空集）
	UserIDs      []string  // 用户 user_id 白名单（其余维度用；nil=不过滤/空=空集）
	UserFilter   string    // 下钻：限定单个 user_id（已经过 scope 校验）
	SchoolMember []string  // 下钻：限定某学校的成员 user_id 列表（已在 scope 内）
}

// GetConsumptionSummary 按维度聚合消费汇总（含总计）
//
// 返回 rows（排行/趋势）+ 三个总计值。占比 Percent 在本函数内基于 totalCredits 算好。
func GetConsumptionSummary(ctx context.Context, p *ConsumptionSummaryParams) ([]*models.ConsumptionSummaryRow, float64, float64, int, error) {
	// ---------- 1. 构造公共 WHERE（时间范围 + scope 白名单 + 下钻过滤）----------
	// cl = token_consumption_logs 别名。region/school 维度需 JOIN 账户树，其余维度仅查 cl。
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	// 时间范围（from/to 为零值则不加该条件）
	if !p.From.IsZero() {
		where += fmt.Sprintf(" AND cl.created_at >= $%d", argIdx)
		args = append(args, p.From)
		argIdx++
	}
	if !p.To.IsZero() {
		where += fmt.Sprintf(" AND cl.created_at < $%d", argIdx)
		args = append(args, p.To)
		argIdx++
	}

	// 下钻：限定单个 user_id（优先级最高，handler 已校验在 scope 内）
	if p.UserFilter != "" {
		where += fmt.Sprintf(" AND cl.user_id = $%d", argIdx)
		args = append(args, p.UserFilter)
		argIdx++
	}

	// 下钻：限定某学校成员（handler 已确认这些 user_id 在 scope 内）
	if p.SchoolMember != nil {
		if len(p.SchoolMember) == 0 {
			where += " AND 1=0"
		} else {
			where += fmt.Sprintf(" AND cl.user_id = ANY($%d)", argIdx)
			args = append(args, p.SchoolMember)
			argIdx++
		}
	}

	// scope 白名单：region 维度按 grandparent(region) owner 收窄，school 维度按 parent(school)
	// owner 收窄，其余维度按 user_id 收窄。
	switch p.Dimension {
	case models.SummaryDimRegion:
		if p.OwnerIDs != nil {
			if len(p.OwnerIDs) == 0 {
				where += " AND 1=0"
			} else {
				// 消费记在 personal，往上两级 grandparent(region) 取 owner
				where += fmt.Sprintf(` AND region.owner_id = ANY($%d) AND region.account_type = 'region'`, argIdx)
				args = append(args, p.OwnerIDs)
				argIdx++
			}
		}
	case models.SummaryDimSchool:
		if p.OwnerIDs != nil {
			if len(p.OwnerIDs) == 0 {
				where += " AND 1=0"
			} else {
				// 消费记在 personal，往上一级 parent(school) 取 owner
				where += fmt.Sprintf(` AND parent.owner_id = ANY($%d) AND parent.account_type <> 'region'`, argIdx)
				args = append(args, p.OwnerIDs)
				argIdx++
			}
		}
	default:
		// user/model/scene/time 维度：按 user_id 白名单收窄
		if p.UserIDs != nil {
			if len(p.UserIDs) == 0 {
				where += " AND 1=0"
			} else {
				where += fmt.Sprintf(" AND cl.user_id = ANY($%d)", argIdx)
				args = append(args, p.UserIDs)
				argIdx++
			}
		}
	}

	// ---------- 2. 按维度构造 SELECT / GROUP BY / FROM ----------
	var selectExpr, groupBy, orderBy, fromClause string

	switch p.Dimension {
	case models.SummaryDimRegion:
		// 消费(personal) → parent(school) → region → owner组织 = 区域
		// 四级 JOIN：cl.account_id → ta(personal) → school → region → o(区域组织)
		// owner_id 是 uuid，COALESCE 兜底前须 ::text
		fromClause = `
			FROM token_consumption_logs cl
			LEFT JOIN token_accounts ta ON ta.id = cl.account_id
			LEFT JOIN token_accounts school ON school.id = ta.parent_account_id
			LEFT JOIN token_accounts region ON region.id = school.parent_account_id
			LEFT JOIN organizations o ON o.id = region.owner_id`
		selectExpr = `COALESCE(region.owner_id::text,'') AS key, COALESCE(o.name,'(无区域归属)') AS label`
		groupBy = "region.owner_id, o.name"
		orderBy = "credits DESC"

	case models.SummaryDimSchool:
		// 消费(personal账户) → parent(学校账户) → owner组织 = 学校
		// 三级 JOIN：cl.account_id → ta(消费账户) → parent(学校账户) → o(学校组织)
		fromClause = `
			FROM token_consumption_logs cl
			LEFT JOIN token_accounts ta ON ta.id = cl.account_id
			LEFT JOIN token_accounts parent ON parent.id = ta.parent_account_id
			LEFT JOIN organizations o ON o.id = parent.owner_id`
		selectExpr = `COALESCE(parent.owner_id::text,'') AS key, COALESCE(o.name,'(无归属)') AS label`
		groupBy = "parent.owner_id, o.name"
		orderBy = "credits DESC"

	case models.SummaryDimUser:
		fromClause = `
			FROM token_consumption_logs cl
			LEFT JOIN users u ON u.id = cl.user_id`
		selectExpr = `cl.user_id AS key, COALESCE(u.display_name,'(未知用户)') AS label`
		groupBy = "cl.user_id, u.display_name"
		orderBy = "credits DESC"

	case models.SummaryDimModel:
		fromClause = `FROM token_consumption_logs cl`
		selectExpr = `COALESCE(NULLIF(cl.model_name,''),'(未知模型)') AS key, COALESCE(NULLIF(cl.model_name,''),'(未知模型)') AS label`
		groupBy = "cl.model_name"
		orderBy = "credits DESC"

	case models.SummaryDimScene:
		// scene 中文名在 service 层用 SceneNameMap 翻译，这里只返回原始 scene_code
		fromClause = `FROM token_consumption_logs cl`
		selectExpr = `COALESCE(NULLIF(cl.scene_code,''),'(未知场景)') AS key, COALESCE(NULLIF(cl.scene_code,''),'(未知场景)') AS label`
		groupBy = "cl.scene_code"
		orderBy = "credits DESC"

	case models.SummaryDimTime:
		fromClause = `FROM token_consumption_logs cl`
		selectExpr = `to_char(cl.created_at::date,'YYYY-MM-DD') AS key, to_char(cl.created_at::date,'YYYY-MM-DD') AS label`
		groupBy = "cl.created_at::date"
		orderBy = "key ASC" // 时间正序，柱状图从早到晚

	default:
		return nil, 0, 0, 0, fmt.Errorf("不支持的汇总维度: %s", p.Dimension)
	}

	// ---------- 3. 聚合查询 ----------
	query := fmt.Sprintf(`
		SELECT %s,
		       COALESCE(SUM(cl.credits_consumed),0) AS credits,
		       COALESCE(SUM(cl.cost_usd),0) AS cost_usd,
		       COUNT(*) AS calls
		%s
		WHERE %s
		GROUP BY %s
		ORDER BY %s
	`, selectExpr, fromClause, where, groupBy, orderBy)

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("查询消费汇总失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ConsumptionSummaryRow
	var totalCredits, totalCostUSD float64
	var totalCalls int
	for rows.Next() {
		row := &models.ConsumptionSummaryRow{}
		if err := rows.Scan(&row.Key, &row.Label, &row.Credits, &row.CostUSD, &row.Calls); err != nil {
			return nil, 0, 0, 0, fmt.Errorf("扫描消费汇总行失败: %w", err)
		}
		totalCredits += row.Credits
		totalCostUSD += row.CostUSD
		totalCalls += row.Calls
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("遍历消费汇总结果失败: %w", err)
	}

	// ---------- 4. 计算占比 ----------
	if totalCredits > 0 {
		for _, row := range items {
			row.Percent = row.Credits * 100.0 / totalCredits
		}
	}

	return items, totalCredits, totalCostUSD, totalCalls, nil
}
