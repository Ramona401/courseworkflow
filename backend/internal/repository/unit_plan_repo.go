package repository

// unit_plan_repo.go — 单元方案数据访问（大单元备课·独立模块）
//
// 操作 unit_plans 表。风格对齐 course_outline_repo.go。
// 可见性：本人草稿(draft) ∪ 按可见性可见的 active。
//   admin → 所有 active；非 admin → 全局 system + 自己组 group + 本校 school 的 active。
// 归属名/建立者名 LEFT JOIN 回填。
//
// 大单元挂载（前端入口）新增：
//   ListActiveUnitPlansForMount —— 专供「教案挂载单元方案选择器」的查询。
//   与 ListUnitPlans 的关键区别：只列 active，绝不列草稿(draft)。
//   原因：后端注入层（workshop_stage_service.go）只注入 status==active 的单元方案，
//   草稿挂上去备课时不生效（鬼打墙）。挂载选择器的口径必须与注入层焊死一致，
//   故单独开一个"只列 active"的查询，而不是复用 ListUnitPlans 后在前端过滤——
//   把"能挂什么"的判断焊在后端，前端无脑列即可。
//
// v232修复（保存撞唯一约束）：
//   FinalizeUnitPlan 保存前先归档同归属+同维度的旧 active 方案，再激活新方案——
//   "新版覆盖旧版"语义，根治 uq_unit_plans_active 唯一约束冲突(SQLSTATE 23505)。
//
// v233 变更（课程大纲教材版本绑定，对齐备课工坊）：
//   1) unitPlanSelectColumns 与 scanUnitPlan 新增 course_outline_publisher 列——
//      直接扫进 *string（pgx v5 对指针目标原生支持：列 NULL → nil / 非 NULL → 指向值），
//      还原三态语义：nil=未关联 / ""=通用版 / 具名=该版本精确匹配；
//   2) CreateUnitPlanDraft 的 INSERT 新增该列，*string 直接透传（pgx 对 nil 自动写 NULL），
//      老客户端不传该字段时解码为 nil → 落 NULL，天然兼容零回归。
//   其余方法与查询保持不变。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrUnitPlanNotFound 单元方案不存在
var ErrUnitPlanNotFound = errors.New("单元方案不存在")

// unitPlanSelectColumns 单条查询统一列（与 scanUnitPlan 对齐）
// v233：末尾新增 course_outline_publisher（可空，不加 COALESCE——NULL 需保留以还原"未关联"态）
const unitPlanSelectColumns = `id, scope, scope_target_id, subject, grade, volume, unit, unit_theme,
title, content, atlas, COALESCE(conversation_log::text,'[]'), source_type, created_by, status,
created_at, updated_at, course_outline_publisher`

// scanUnitPlan 统一扫描单条
// v233：末尾新增 &p.CourseOutlinePublisher（**string）——pgx v5 原生支持：
//   列为 NULL → 字段置 nil（未关联大纲）；列非 NULL → 字段指向实际值（""=通用版 / 具名版本）
func scanUnitPlan(row pgx.Row) (*models.UnitPlan, error) {
	p := &models.UnitPlan{}
	err := row.Scan(
		&p.ID, &p.Scope, &p.ScopeTargetID, &p.Subject, &p.Grade, &p.Volume, &p.Unit, &p.UnitTheme,
		&p.Title, &p.Content, &p.Atlas, &p.ConversationLog, &p.SourceType, &p.CreatedBy, &p.Status,
		&p.CreatedAt, &p.UpdatedAt, &p.CourseOutlinePublisher,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnitPlanNotFound
		}
		return nil, fmt.Errorf("扫描单元方案失败: %w", err)
	}
	return p, nil
}

// CreateUnitPlanDraft 建一份草稿（status=draft），回填 id/时间
// v233：INSERT 新增 course_outline_publisher（$11）——*string 直接透传，
// nil 写 NULL（未关联/老客户端）、"" 写空串（通用版）、具名写实际值（版本精确匹配）。
func CreateUnitPlanDraft(ctx context.Context, p *models.UnitPlan) error {
	sourceType := p.SourceType
	if sourceType == "" {
		sourceType = models.UnitPlanSourceGenerated
	}
	err := database.DB.QueryRow(ctx, `
		INSERT INTO unit_plans
		  (scope, scope_target_id, subject, grade, volume, unit, unit_theme, title,
		   source_type, created_by, course_outline_publisher, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'draft')
		RETURNING id, created_at, updated_at
	`,
		p.Scope, p.ScopeTargetID, p.Subject, p.Grade, p.Volume, p.Unit, p.UnitTheme, p.Title,
		sourceType, p.CreatedBy, p.CourseOutlinePublisher,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建单元方案草稿失败: %w", err)
	}
	p.Status = models.UnitPlanStatusDraft
	p.SourceType = sourceType
	return nil
}

// GetUnitPlanByID 按 ID 查单条（含 conversation_log，归属/续作校验用）
func GetUnitPlanByID(ctx context.Context, id string) (*models.UnitPlan, error) {
	sql := `SELECT ` + unitPlanSelectColumns + ` FROM unit_plans WHERE id = $1`
	return scanUnitPlan(database.DB.QueryRow(ctx, sql, id))
}

// AppendUnitPlanMessage 往对话日志末尾原子追加一条消息（jsonb 数组拼接）
func AppendUnitPlanMessage(ctx context.Context, id, role, content string) error {
	_, err := database.DB.Exec(ctx, `
		UPDATE unit_plans
		SET conversation_log = conversation_log || jsonb_build_array(
		      jsonb_build_object('role', $2::text, 'content', $3::text, 'created_at', now())
		    ),
		    updated_at = now()
		WHERE id = $1
	`, id, role, content)
	if err != nil {
		return fmt.Errorf("追加单元方案对话失败: %w", err)
	}
	return nil
}

// FinalizeUnitPlan 定稿：写方案/图谱/主题/标题并置 active
//
// v232修复（保存撞唯一约束）：
// 唯一约束 uq_unit_plans_active 保证同归属+学科+年级+册次+单元只有一份 active。
// 若该组合已有旧 active 方案，先将旧方案归档(archived)再激活新方案——
// "新版覆盖旧版"语义，符合老师预期（同一个单元只保留最新定稿）。
// 归档操作按 scope+scope_target_id+subject+grade+volume+unit 精确匹配，
// 且排除本方案自身（防止自己归档自己）。
func FinalizeUnitPlan(ctx context.Context, id string, req *models.SaveUnitPlanRequest) error {
	// 先查出本方案的归属维度，用于定位可能冲突的旧 active 方案
	var scope, scopeTargetID, subject, grade, volume, unit string
	lookupErr := database.DB.QueryRow(ctx, `
		SELECT scope, scope_target_id, subject, grade, volume, unit
		FROM unit_plans WHERE id = $1
	`, id).Scan(&scope, &scopeTargetID, &subject, &grade, &volume, &unit)
	if lookupErr != nil {
		return fmt.Errorf("查询单元方案归属失败: %w", lookupErr)
	}

	// 将同归属+同学科+同年级+同册次+同单元的旧 active 方案归档（排除自身）
	// 归档失败不阻断保存——最坏情况下面 UPDATE 会撞唯一约束，走原有报错
	_, archiveErr := database.DB.Exec(ctx, `
		UPDATE unit_plans
		SET status = 'archived', updated_at = now()
		WHERE scope = $1 AND scope_target_id = $2
		  AND subject = $3 AND grade = $4 AND volume = $5 AND unit = $6
		  AND status = 'active' AND id <> $7
	`, scope, scopeTargetID, subject, grade, volume, unit, id)
	if archiveErr != nil {
		fmt.Printf("[WARN] 归档旧单元方案失败: %v\n", archiveErr)
	}

	// 激活本方案
	result, err := database.DB.Exec(ctx, `
		UPDATE unit_plans
		SET title = $1, unit_theme = $2, content = $3, atlas = $4,
		    status = 'active', updated_at = now()
		WHERE id = $5 AND status <> 'archived'
	`, req.Title, req.UnitTheme, req.Content, req.Atlas, id)
	if err != nil {
		return fmt.Errorf("保存单元方案失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUnitPlanNotFound
	}
	return nil
}

// DeleteUnitPlan 软删除（archived）
func DeleteUnitPlan(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE unit_plans SET status='archived', updated_at=now() WHERE id=$1 AND status<>'archived'`,
		id)
	if err != nil {
		return fmt.Errorf("删除单元方案失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUnitPlanNotFound
	}
	return nil
}

// ListUnitPlans 列出可见单元方案（资源界面用）
//
// 可见 = 本人草稿(draft) ∪ 按可见性可见的 active。不含 archived。
func ListUnitPlans(ctx context.Context, scopeIsAdmin bool, groupIDs, schoolIDs []string, ownerID string) ([]*models.UnitPlanListItem, error) {
	baseSQL := `
		SELECT up.id, up.scope, up.scope_target_id,
		       COALESCE(CASE up.scope
		                  WHEN 'group'  THEN tg.name
		                  WHEN 'school' THEN org.name
		                  WHEN 'system' THEN '全局（所有学校通用）'
		                END,'') AS scope_name,
		       up.subject, up.grade, up.volume, up.unit, up.unit_theme, up.title, up.status,
		       COALESCE(u.display_name,'') AS creator_name, up.updated_at
		FROM unit_plans up
		LEFT JOIN teaching_groups tg ON tg.id = up.scope_target_id AND up.scope = 'group'
		LEFT JOIN organizations  org ON org.id = up.scope_target_id AND up.scope = 'school'
		LEFT JOIN users u ON u.id = up.created_by
		WHERE up.status <> 'archived'`

	var args []interface{}
	if scopeIsAdmin {
		// admin：本人草稿 OR 任意 active
		args = append(args, ownerID)
		baseSQL += `
		  AND ( (up.status = 'draft' AND up.created_by = $1)
		     OR  up.status = 'active' )`
	} else {
		// 非 admin：本人草稿 OR （active 且按可见性：全局/自己组/本校）
		args = append(args, ownerID, groupIDs, schoolIDs)
		baseSQL += `
		  AND ( (up.status = 'draft' AND up.created_by = $1)
		     OR ( up.status = 'active' AND (
		            up.scope = 'system'
		         OR (up.scope = 'group'  AND up.scope_target_id = ANY($2))
		         OR (up.scope = 'school' AND up.scope_target_id = ANY($3))
		        )) )`
	}
	baseSQL += ` ORDER BY up.updated_at DESC`

	rows, err := database.DB.Query(ctx, baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("查询单元方案列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.UnitPlanListItem
	for rows.Next() {
		it := &models.UnitPlanListItem{}
		if err := rows.Scan(
			&it.ID, &it.Scope, &it.ScopeTargetID, &it.ScopeName,
			&it.Subject, &it.Grade, &it.Volume, &it.Unit, &it.UnitTheme, &it.Title, &it.Status,
			&it.CreatorName, &it.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描单元方案列表行失败: %w", err)
		}
		items = append(items, it)
	}
	if items == nil {
		items = []*models.UnitPlanListItem{}
	}
	return items, nil
}

// ListActiveUnitPlansForMount 列出「可被教案挂载」的单元方案（挂载选择器专用）
//
// 与 ListUnitPlans 的唯一区别：只列 active，绝不列任何 draft。
//   - 后端注入层只注入 status==active 的单元方案；草稿挂上去备课不生效。
//   - 故挂载选择器只应展示 active，口径与注入层焊死一致，老师看到什么就能挂什么、挂了就生效。
//
// 可见性规则与 ListUnitPlans 的 active 分支完全相同（不含草稿分支）：
//   - admin            → 所有 active
//   - 非 admin         → 全局 system ∪ 自己组 group ∪ 本校 school 的 active
//   不含 archived（软删的不可挂）。
//
// 可按 subject 选填收窄（传空串=不按学科过滤）。挂载选择器通常按"当前教案学科"传入，
// 让老师只看到同学科的单元方案，减少噪音；但不强制（service 层决定传不传）。
func ListActiveUnitPlansForMount(ctx context.Context, scopeIsAdmin bool, groupIDs, schoolIDs []string, subject string) ([]*models.UnitPlanListItem, error) {
	baseSQL := `
		SELECT up.id, up.scope, up.scope_target_id,
		       COALESCE(CASE up.scope
		                  WHEN 'group'  THEN tg.name
		                  WHEN 'school' THEN org.name
		                  WHEN 'system' THEN '全局（所有学校通用）'
		                END,'') AS scope_name,
		       up.subject, up.grade, up.volume, up.unit, up.unit_theme, up.title, up.status,
		       COALESCE(u.display_name,'') AS creator_name, up.updated_at
		FROM unit_plans up
		LEFT JOIN teaching_groups tg ON tg.id = up.scope_target_id AND up.scope = 'group'
		LEFT JOIN organizations  org ON org.id = up.scope_target_id AND up.scope = 'school'
		LEFT JOIN users u ON u.id = up.created_by
		WHERE up.status = 'active'`

	var args []interface{}
	argIdx := 1

	if scopeIsAdmin {
		// admin：任意 active（无可见性收窄），不追加白名单条件
	} else {
		// 非 admin：active 且按可见性（全局 system / 自己组 group / 本校 school）
		groupPlaceholder := argIdx
		schoolPlaceholder := argIdx + 1
		args = append(args, groupIDs, schoolIDs)
		argIdx += 2
		baseSQL += fmt.Sprintf(`
		  AND (
		        up.scope = 'system'
		     OR (up.scope = 'group'  AND up.scope_target_id = ANY($%d))
		     OR (up.scope = 'school' AND up.scope_target_id = ANY($%d))
		  )`, groupPlaceholder, schoolPlaceholder)
	}

	// 选填：按学科收窄（传空串则不过滤）
	if subject != "" {
		baseSQL += fmt.Sprintf(` AND up.subject = $%d`, argIdx)
		args = append(args, subject)
		argIdx++
	}

	baseSQL += ` ORDER BY up.updated_at DESC`

	rows, err := database.DB.Query(ctx, baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("查询可挂载单元方案列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.UnitPlanListItem
	for rows.Next() {
		it := &models.UnitPlanListItem{}
		if err := rows.Scan(
			&it.ID, &it.Scope, &it.ScopeTargetID, &it.ScopeName,
			&it.Subject, &it.Grade, &it.Volume, &it.Unit, &it.UnitTheme, &it.Title, &it.Status,
			&it.CreatorName, &it.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描可挂载单元方案列表行失败: %w", err)
		}
		items = append(items, it)
	}
	if items == nil {
		items = []*models.UnitPlanListItem{}
	}
	return items, nil
}
