package repository

// ai_assistant_repo.go — AI 助手数据访问层
//
// 职责:
//   - CRUD:CreateAIAssistant / GetAIAssistantByID / ListAIAssistants / UpdateAIAssistant / DeleteAIAssistant
//   - 使用量统计:IncrementAIAssistantUseCount
//
// ──────────────────────────────────────────────────────────────────────
// 可见性规则(由 Service 层组装 params 后传入):
//   - 所有用户可见: source='system'
//   - 教研组级可见: source='group' AND group_id = ANY(我所属的教研组集合)
//   - 全校级可见:   source='group' AND group_id IS NULL AND organization_id=<我的学校>
//   - 个人可见:     source='personal' AND created_by=<当前用户>
//
//   列表 SQL 通过 OR 多条件实现上述可见性的并集查询。
//   里程碑一关键:group 来源用 group_id 是否为空区分两档——
//     非空=教研组级(仅同组可见)、空=全校级(同校可见),两档互斥不重叠。
// ──────────────────────────────────────────────────────────────────────
//
// ──────────────────────────────────────────────────────────────────────
// share_policy(分享权限策略,本次新增)在本层的处理:
//   - CreateAIAssistant 的 INSERT 增加 share_policy 列(第 17 个字段)
//   - GetAIAssistantByID 的 SELECT 增加读取 share_policy(供 service 判 fork/edit)
//   - ListAIAssistants 的列表 SELECT 增加读取 share_policy,回填到列表项,
//     并据此计算 CanFork(列表层"按钮显隐提示",最终拦截仍在 service.ForkAssistant)
//   - UpdateAIAssistant 支持按需更新 share_policy(req.SharePolicy 非 nil 时才改)
//
//   ⚠ 本层只忠实存取 share_policy + 计算列表层 CanFork 提示,
//     真正的 fork/edit 权限拦截在 service 层(那里有完整的 MyLeadGroupIDs)。
// ──────────────────────────────────────────────────────────────────────
//
// v115改动(2026-04-20 学段匹配修复):
//   GradeRange 过滤逻辑从"精确字符串匹配"升级为"学段级匹配":
//   前端传任意格式的年级输入("七年级"/"初一"/"7"/"7-9"),
//   Repository 先调用 utils.NormalizeGradeToSegment 归一化为学段("初中"/"小学"/"高中"/""),
//   再与数据库 grade_range 字段做精确匹配。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// ==================== 错误常量 ====================

var (
	ErrAIAssistantNotFound = errors.New("AI 助手不存在")
	ErrAIAssistantInactive = errors.New("AI 助手已停用")
)

// ==================== 创建 ====================

// CreateAIAssistant 创建助手记录
// 调用方负责设置 Source / CreatedBy / OrganizationID / GroupID / SharePolicy 等归属字段
func CreateAIAssistant(ctx context.Context, a *models.AIAssistant) error {
	// 兜底默认值
	if a.AvatarEmoji == "" {
		a.AvatarEmoji = "🤖"
	}
	if a.KnowledgeRefs == "" {
		a.KnowledgeRefs = "[]"
	}
	if a.Scenes == "" {
		a.Scenes = "[]"
	}
	if a.IsDefaultForScene == "" {
		a.IsDefaultForScene = "[]"
	}
	// share_policy 兜底:为空或非法时统一回落默认 use_only(与 DB 默认值一致)
	if !models.IsValidSharePolicy(a.SharePolicy) {
		a.SharePolicy = models.SharePolicyUseOnly
	}

	query := `
		INSERT INTO ai_assistants (
			name, avatar_emoji, description,
			source, created_by, organization_id, group_id,
			share_policy,
			full_prompt, knowledge_refs,
			subject, grade_range, scenes,
			forked_from,
			sort_order, is_default_for_scene, is_active
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8,
			$9, $10,
			$11, $12, $13,
			$14,
			$15, $16, $17
		)
		RETURNING id, created_at, updated_at
	`
	err := database.DB.QueryRow(ctx, query,
		a.Name, a.AvatarEmoji, a.Description,
		a.Source, a.CreatedBy, a.OrganizationID, a.GroupID,
		a.SharePolicy,
		a.FullPrompt, a.KnowledgeRefs,
		a.Subject, a.GradeRange, a.Scenes,
		a.ForkedFrom,
		a.SortOrder, a.IsDefaultForScene, a.IsActive,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建 AI 助手失败: %w", err)
	}
	return nil
}

// ==================== 查询单个 ====================

// GetAIAssistantByID 根据 ID 获取助手(不判断可见性,由 Service 层校验)
func GetAIAssistantByID(ctx context.Context, id string) (*models.AIAssistant, error) {
	a := &models.AIAssistant{}
	query := `
		SELECT id, name, avatar_emoji, COALESCE(description, ''),
			source, created_by, organization_id, group_id,
			COALESCE(share_policy, 'use_only'),
			full_prompt, COALESCE(knowledge_refs::text, '[]'),
			COALESCE(subject, ''), COALESCE(grade_range, ''),
			COALESCE(scenes::text, '[]'),
			creation_conversation::text,
			forked_from,
			use_count, avg_score,
			sort_order, COALESCE(is_default_for_scene::text, '[]'),
			is_active, created_at, updated_at
		FROM ai_assistants
		WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.Name, &a.AvatarEmoji, &a.Description,
		&a.Source, &a.CreatedBy, &a.OrganizationID, &a.GroupID,
		&a.SharePolicy,
		&a.FullPrompt, &a.KnowledgeRefs,
		&a.Subject, &a.GradeRange, &a.Scenes,
		&a.CreationConversation,
		&a.ForkedFrom,
		&a.UseCount, &a.AvgScore,
		&a.SortOrder, &a.IsDefaultForScene,
		&a.IsActive, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAIAssistantNotFound
		}
		return nil, fmt.Errorf("查询 AI 助手失败: %w", err)
	}
	return a, nil
}

// ==================== 列表查询(含可见性过滤) ====================

// ListAIAssistants 列表查询,按可见性规则返回当前用户可见的助手
//
// 里程碑一可见性(OR 并集):
//
//	system(所有人) + 教研组级 group(我所属的组) + 全校级 group(同校) + personal(自己)
//
// v115改动:GradeRange 过滤前先调 NormalizeGradeToSegment 归一化为学段
//
// share_policy 本次新增:读取每行 share_policy 回填列表项,并计算 CanFork
func ListAIAssistants(ctx context.Context, params *models.ListAIAssistantsParams) ([]*models.AIAssistantListItem, int, error) {
	// 构建可见性子句,各 source 用 OR 连接
	visibilityClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	// (A) 所有人可见 system
	visibilityClauses = append(visibilityClauses, "a.source = 'system'")

	// (B) 教研组级可见:source='group' 且 group_id ∈ 我所属的教研组集合
	//     用 = ANY($n) 接收 []string,pgx 自动编码为数组;空集合匹配空,即不在任何组就看不到
	if len(params.CurrentGroupIDs) > 0 {
		visibilityClauses = append(visibilityClauses,
			fmt.Sprintf("(a.source = 'group' AND a.group_id IS NOT NULL AND a.group_id::text = ANY($%d))", argIdx))
		args = append(args, params.CurrentGroupIDs)
		argIdx++
	}

	// (C) 全校级可见:source='group' 且 group_id 为空(全校档) 且 organization_id = 我的学校
	if params.CurrentSchoolID != "" {
		visibilityClauses = append(visibilityClauses,
			fmt.Sprintf("(a.source = 'group' AND a.group_id IS NULL AND a.organization_id = $%d)", argIdx))
		args = append(args, params.CurrentSchoolID)
		argIdx++
	}

	// (D) 个人可见 personal(只看自己的)
	if params.CurrentUserID != "" {
		visibilityClauses = append(visibilityClauses,
			fmt.Sprintf("(a.source = 'personal' AND a.created_by = $%d)", argIdx))
		args = append(args, params.CurrentUserID)
		argIdx++
	}

	where := " WHERE (" + strings.Join(visibilityClauses, " OR ") + ")"

	// ── share_policy=locked 的收紧:非属主非 admin 不可见(即使因 source 落入可见集) ──
	//   locked 语义=仅属主/admin 可见可用。这里在可见性 WHERE 之上再叠加一层过滤:
	//   把"locked 且(不是我建的)"的助手剔除;admin 不加此限制(可见全部)。
	//   注意:此过滤只影响"看得到与否",不影响 source 本身的归属判断。
	if params.CurrentUserRole != models.RoleAdmin {
		// 非 admin:排除 locked 且非本人创建的助手
		if params.CurrentUserID != "" {
			where += fmt.Sprintf(" AND NOT (a.share_policy = 'locked' AND (a.created_by IS NULL OR a.created_by <> $%d))", argIdx)
			args = append(args, params.CurrentUserID)
			argIdx++
		} else {
			// 没有用户 ID(理论不会发生):保守把所有 locked 排除
			where += " AND a.share_policy <> 'locked'"
		}
	}

	// 附加过滤
	if params.OnlyActive {
		where += " AND a.is_active = true"
	}
	if params.Subject != "" {
		// 学科严格匹配：空学科通用助手不再参与具体课程匹配。
		where += fmt.Sprintf(" AND a.subject = $%d", argIdx)
		args = append(args, params.Subject)
		argIdx++
	}
	// 具体年级不直接在SQL中比较：
	// 数据库存量可能使用“高三/十二年级/12年级/12”等同义表达，
	// 扫描结果后统一调用utils.IsStrictGradeMatch。
	// 学段、空年级和跨年级范围都会被严格排除。
	if params.Scene != "" {
		// scenes 是 JSONB 数组,使用 @> 判断包含
		where += fmt.Sprintf(" AND a.scenes @> $%d::jsonb", argIdx)
		args = append(args, fmt.Sprintf(`["%s"]`, params.Scene))
		argIdx++
	}

	// 查询总数
	countQuery := `SELECT COUNT(*) FROM ai_assistants a` + where
	var total int
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计 AI 助手数量失败: %w", err)
	}

	// 查询列表(带创建者名、学校名、教研组名)
	listQuery := `
		SELECT a.id, a.name, a.avatar_emoji, COALESCE(a.description, ''),
			a.source,
			COALESCE(a.share_policy, 'use_only'),
			COALESCE(a.subject, ''), COALESCE(a.grade_range, ''),
			COALESCE(a.scenes::text, '[]'),
			a.use_count, a.avg_score,
			a.is_active, COALESCE(a.is_default_for_scene::text, '[]'),
			a.created_by, a.organization_id, a.group_id,
			COALESCE(u.display_name, '')  AS creator_name,
			COALESCE(o.name, '')          AS school_name,
			COALESCE(tg.name, '')         AS group_name,
			a.created_at, a.updated_at
		FROM ai_assistants a
		LEFT JOIN users u            ON u.id = a.created_by
		LEFT JOIN organizations o    ON o.id = a.organization_id
		LEFT JOIN teaching_groups tg ON tg.id = a.group_id
	` + where + `
		ORDER BY
			CASE a.source WHEN 'system' THEN 0 WHEN 'group' THEN 1 ELSE 2 END,
			a.sort_order DESC,
			a.created_at ASC
	`

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询 AI 助手列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.AIAssistantListItem
	for rows.Next() {
		item := &models.AIAssistantListItem{}
		var scenesJSON, defaultJSON string
		var createdBy, orgID, groupID *string
		err := rows.Scan(
			&item.ID, &item.Name, &item.AvatarEmoji, &item.Description,
			&item.Source,
			&item.SharePolicy,
			&item.Subject, &item.GradeRange,
			&scenesJSON,
			&item.UseCount, &item.AvgScore,
			&item.IsActive, &defaultJSON,
			&createdBy, &orgID, &groupID,
			&item.CreatorName,
			&item.SchoolName,
			&item.GroupName,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描 AI 助手行失败: %w", err)
		}

		// 回填教研组归属(供前端区分教研组级/全校级展示)
		item.GroupID = groupID

		// 具体年级严格过滤：高三只接受高三、12、十二年级等同义表达。
		// 高中、高一、高二、范围或空年级助手均不参与匹配。
		if params.GradeRange != "" &&
			!utils.IsStrictGradeMatch(
				item.GradeRange,
				params.GradeRange,
			) {
			continue
		}

		// 解析 scenes
		var scenes []string
		_ = json.Unmarshal([]byte(scenesJSON), &scenes)
		item.Scenes = scenes

		// 当前场景是否默认
		if params.Scene != "" {
			var defaults []string
			_ = json.Unmarshal([]byte(defaultJSON), &defaults)
			for _, d := range defaults {
				if d == params.Scene {
					item.IsDefaultHere = true
					break
				}
			}
		}

		// source 标签
		if label, ok := models.SourceLabelMap[item.Source]; ok {
			item.SourceLabel = label
		} else {
			item.SourceLabel = item.Source
		}

		// 编辑/删除权限判断(列表层只做按钮显隐提示,最终拦截在 service.canEdit)
		item.CanEdit = canEditAssistant(item.Source, item.SharePolicy, createdBy, orgID, groupID, params)
		item.CanDelete = item.CanEdit && item.Source != models.AssistantSourceSystem

		// Fork 权限判断(列表层提示,最终拦截在 service.ForkAssistant)
		item.CanFork = canForkAssistant(item.SharePolicy, createdBy, params)

		// 查看原文权限判断(列表层提示,最终拦截在 service.GetAssistant)
		//   前端据此显隐"丢给 AI 分析"按钮,避免老师点了才发现拿不到 use_only 助手的原文
		item.CanViewPrompt = canViewPromptAssistant(item.Source, item.SharePolicy, createdBy, orgID, groupID, params)

		items = append(items, item)
	}

	if items == nil {
		items = []*models.AIAssistantListItem{}
	}

	// 年级在Go层完成严格过滤时，以过滤后的数量作为真实total。
	if params.GradeRange != "" {
		total = len(items)
	}

	return items, total, nil
}

// canEditAssistant 判断当前用户能否编辑该助手(列表层快速提示)
//
// ⚠ 说明:本函数仅用于列表项 CanEdit/CanDelete 的"按钮显隐提示",
//
//	不是权限的最终防线。真正的编辑/删除拦截在 service 层 canEdit(),
//	那里有完整的 MyLeadGroupIDs,能精确判定"组长可改本组其他人建的助手"。
//
//	本函数为保持改动收敛,对教研组级助手只放行"创建者本人 + admin",
//	不放行"组长改组员建的组助手"(此情况按钮不显示,但组长仍可经正常编辑入口操作,
//	service 层会放行)。这是 UI 提示偏保守、不影响安全与功能正确性的有意取舍。
//
//	share_policy 叠加(本次新增):
//	  use_only / locked → 即使 source 规则允许,非属主非 admin 也不显示编辑按钮
//	    (防"标准被改坏":只有属主、组长、admin 能改;组员看到的是只读)
//	  open              → 不额外收紧,沿用 source 规则
//
//	规则(综合 source + share_policy):
//	  admin      → 任何助手都可编辑(不受 share_policy 限制)
//	  system     → 非 admin 一律不可
//	  group      → 创建者本人可编辑;use_only/locked 时非创建者(组长除外,组长由 service 兜底)不可
//	  personal   → 仅创建者本人(personal 本就只有自己看,share_policy 不额外影响)
func canEditAssistant(source, sharePolicy string, createdBy *string, orgID *string, groupID *string, params *models.ListAIAssistantsParams) bool {
	// admin 可编辑任何助手(share_policy 对 admin 不设限)
	if params.CurrentUserRole == models.RoleAdmin {
		return true
	}

	// 是否本人创建
	isOwner := createdBy != nil && params.CurrentUserID != "" && *createdBy == params.CurrentUserID

	switch source {
	case models.AssistantSourceSystem:
		return false
	case models.AssistantSourceGroup:
		// 创建者本人恒可编辑(教研组级 / 全校级通用)
		if isOwner {
			return true
		}
		// 非创建者:若我是该教研组的组长(lead),则可编辑本组助手——
		//   与 service 层 canEdit 的组长逻辑对齐(本次新增 CurrentLeadGroupIDs 后,列表层也能正确认组长)。
		//   share_policy 收紧:locked 连组长也挡(locked=仅属主);use_only/open 组长可改。
		if groupID != nil && *groupID != "" && containsStrRepo(params.CurrentLeadGroupIDs, *groupID) {
			if sharePolicy == models.SharePolicyLocked {
				return false
			}
			return true
		}
		return false
	case models.AssistantSourcePersonal:
		return isOwner
	}
	return false
}

// containsStrRepo 判断字符串切片是否包含目标值(repository 层内部小工具)
//
//	(与 service 层 containsStr 同名不同包,这里独立一份避免跨层依赖)
func containsStrRepo(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

// canViewPromptAssistant 判断当前用户能否查看该助手的 full_prompt 原文(列表层快速提示,本次新增)
//
// ⚠ 与 canEditAssistant 同理:仅用于列表项 CanViewPrompt 的"按钮显隐提示",
//
//	最终拦截在 service 层 GetAssistant(无权时把 full_prompt 置空)。
//
//	"能看原文" = "能编辑"同款闸门(Yuhan 拍板:看原文权限与编辑权限对齐):
//	  admin / 属主本人 / open 助手任何可见者 / 本组组长(对本组 use_only/open 助手) → 可看
//	  其余(包括能看到 use_only 助手的普通组员) → 不可看(防"丢给 AI 分析"绕开 fork 拿原文)
//
//	注意:能进到列表里的助手已通过可见性过滤(locked 非属主已被剔除),
//	  所以这里 locked 只剩属主自己,直接复用 canEditAssistant 的判定即可:
//	    open  → 任何可见者可看(单独放行)
//	    其他  → 落到 canEditAssistant(admin/属主/组长)
func canViewPromptAssistant(source, sharePolicy string, createdBy *string, orgID *string, groupID *string, params *models.ListAIAssistantsParams) bool {
	// open:任何能看到的人都能看原文(open 本就允许 fork 带走,看原文更无妨)
	if sharePolicy == models.SharePolicyOpen {
		return true
	}
	// 其余(use_only / locked):与"能否编辑"同闸门
	return canEditAssistant(source, sharePolicy, createdBy, orgID, groupID, params)
}

// canForkAssistant 判断当前用户能否把该助手 fork 成自己的(列表层快速提示,本次新增)
//
// ⚠ 与 canEditAssistant 同理:仅用于列表项 CanFork 的"按钮显隐提示",
//
//	最终拦截在 service.ForkAssistant。
//
//	fork 语义=复制一份成自己的 personal 助手带走并可改。share_policy 决定能否带走:
//	  open      → 谁能看到(已被可见性 WHERE 过滤过)就能 fork
//	  use_only  → 仅属主/admin 可 fork(防产权流失:别人只能用不能复制带走)
//	  locked    → 仅属主/admin 可 fork(且 locked 本就只有属主/admin 可见)
//
//	注意:能进到列表里的助手已通过可见性过滤,所以这里只需判 share_policy + 属主/admin。
func canForkAssistant(sharePolicy string, createdBy *string, params *models.ListAIAssistantsParams) bool {
	// admin 可 fork 任何可见助手
	if params.CurrentUserRole == models.RoleAdmin {
		return true
	}
	// open:任何能看到的人都能 fork
	if sharePolicy == models.SharePolicyOpen {
		return true
	}
	// use_only / locked:仅属主本人可 fork
	isOwner := createdBy != nil && params.CurrentUserID != "" && *createdBy == params.CurrentUserID
	return isOwner
}

// ==================== 更新 ====================

// UpdateAIAssistant 更新助手
// 只允许修改:name/avatar/description/full_prompt/subject/grade_range/scenes/is_active/share_policy
// 不允许修改:source/created_by/organization_id/group_id(归属永久不变)
//
// share_policy(本次新增):req.SharePolicy 为 nil 时不改(保持原值),
//
//	非 nil 且合法时更新;非 nil 但非法值则忽略不改(防脏数据)。
func UpdateAIAssistant(ctx context.Context, id string, req *models.UpdateAIAssistantRequest) error {
	// 将 scenes 数组转换为 JSONB 字符串
	scenesJSON, err := json.Marshal(req.Scenes)
	if err != nil {
		return fmt.Errorf("序列化场景列表失败: %w", err)
	}
	if len(req.Scenes) == 0 {
		scenesJSON = []byte("[]")
	}

	// 动态构建 SET 子句
	setParts := []string{
		"name = $1",
		"avatar_emoji = $2",
		"description = $3",
		"full_prompt = $4",
		"subject = $5",
		"grade_range = $6",
		"scenes = $7::jsonb",
		"updated_at = now()",
	}
	args := []interface{}{
		req.Name, req.AvatarEmoji, req.Description,
		req.FullPrompt, req.Subject, req.GradeRange, string(scenesJSON),
	}
	argIdx := 8

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	// share_policy:仅当请求显式带了合法值时才更新(nil=不改)
	if req.SharePolicy != nil && models.IsValidSharePolicy(*req.SharePolicy) {
		setParts = append(setParts, fmt.Sprintf("share_policy = $%d", argIdx))
		args = append(args, *req.SharePolicy)
		argIdx++
	}

	// 最后一个参数是 WHERE id
	query := fmt.Sprintf(
		`UPDATE ai_assistants SET %s WHERE id = $%d`,
		strings.Join(setParts, ", "), argIdx,
	)
	args = append(args, id)

	result, err := database.DB.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("更新 AI 助手失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrAIAssistantNotFound
	}
	return nil
}

// ==================== 删除 ====================

// DeleteAIAssistant 硬删除助手(软删除用 UpdateAIAssistant 把 is_active=false)
// 调用方负责确认 source != 'system'(handler 层已做校验)
func DeleteAIAssistant(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM ai_assistants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除 AI 助手失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrAIAssistantNotFound
	}
	return nil
}

// ==================== 使用量统计 ====================

// IncrementAIAssistantUseCount 增加助手使用次数(每次被调用时 +1)
// P0 埋点,P2 数据飞轮功能启用
func IncrementAIAssistantUseCount(ctx context.Context, id string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE ai_assistants SET use_count = use_count + 1, updated_at = now() WHERE id = $1`,
		id,
	)
	return err
}
