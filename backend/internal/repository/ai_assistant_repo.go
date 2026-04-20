package repository

// ai_assistant_repo.go — AI 助手数据访问层
//
// 职责:
//   - CRUD:CreateAIAssistant / GetAIAssistantByID / ListAIAssistants / UpdateAIAssistant / DeleteAIAssistant
//   - 使用量统计:IncrementAIAssistantUseCount
//
// 可见性规则(由 Service 层组装 params 后传入):
//   - 所有用户可见:source='system' AND is_active=true
//   - 本校用户可见:source='group' AND organization_id=<用户学校>
//   - 个人可见:source='personal' AND created_by=<当前用户>
//
// 列表 SQL 通过 OR 三条件实现上述可见性的并集查询
//
// v115改动(2026-04-20 学段匹配修复):
//   GradeRange 过滤逻辑从"精确字符串匹配"升级为"学段级匹配":
//   前端传任意格式的年级输入("七年级"/"初一"/"7"/"7-9"),
//   Repository 先调用 utils.NormalizeGradeToSegment 归一化为学段("初中"/"小学"/"高中"/""),
//   再与数据库 grade_range 字段做精确匹配。
//   数据库里专业助手的 grade_range 存学段标签("小学"/"初中"/"高中"),
//   通用兜底助手的 grade_range 保持空字符串,SQL 的 OR 条件同时匹配学段和通用两种情况。

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
// 调用方负责设置 Source / CreatedBy / OrganizationID 等归属字段
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

	query := `
		INSERT INTO ai_assistants (
			name, avatar_emoji, description,
			source, created_by, organization_id, group_id,
			full_prompt, knowledge_refs,
			subject, grade_range, scenes,
			forked_from,
			sort_order, is_default_for_scene, is_active
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9,
			$10, $11, $12,
			$13,
			$14, $15, $16
		)
		RETURNING id, created_at, updated_at
	`
	err := database.DB.QueryRow(ctx, query,
		a.Name, a.AvatarEmoji, a.Description,
		a.Source, a.CreatedBy, a.OrganizationID, a.GroupID,
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
// 可见性:system(所有人) + group(本校) + personal(自己)
//
// v115改动:GradeRange 过滤前先调 NormalizeGradeToSegment 归一化为学段
func ListAIAssistants(ctx context.Context, params *models.ListAIAssistantsParams) ([]*models.AIAssistantListItem, int, error) {
	// 构建可见性子句:三种 source 用 OR 连接
	visibilityClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	// (A) 所有人可见 system
	visibilityClauses = append(visibilityClauses, "a.source = 'system'")

	// (B) 本校可见 group(当且仅当 CurrentSchoolID 非空)
	if params.CurrentSchoolID != "" {
		visibilityClauses = append(visibilityClauses,
			fmt.Sprintf("(a.source = 'group' AND a.organization_id = $%d)", argIdx))
		args = append(args, params.CurrentSchoolID)
		argIdx++
	}

	// (C) 个人可见 personal(只看自己的)
	if params.CurrentUserID != "" {
		visibilityClauses = append(visibilityClauses,
			fmt.Sprintf("(a.source = 'personal' AND a.created_by = $%d)", argIdx))
		args = append(args, params.CurrentUserID)
		argIdx++
	}

	where := " WHERE (" + strings.Join(visibilityClauses, " OR ") + ")"

	// 附加过滤
	if params.OnlyActive {
		where += " AND a.is_active = true"
	}
	if params.Subject != "" {
		where += fmt.Sprintf(" AND (a.subject = $%d OR a.subject IS NULL OR a.subject = '')", argIdx)
		args = append(args, params.Subject)
		argIdx++
	}
	// v115:Grade 过滤升级为学段级匹配
	// 前端传"七年级"→ NormalizeGradeToSegment 归一化为"初中"→ 匹配 grade_range='初中' 或空
	// 数据库里专业助手存学段标签,通用助手存空串,OR 条件同时命中
	if params.GradeRange != "" {
		segment := utils.NormalizeGradeToSegment(params.GradeRange)
		where += fmt.Sprintf(" AND (a.grade_range = $%d OR a.grade_range IS NULL OR a.grade_range = '')", argIdx)
		args = append(args, segment)
		argIdx++
	}
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

	// 查询列表(带创建者名和学校名)
	listQuery := `
		SELECT a.id, a.name, a.avatar_emoji, COALESCE(a.description, ''),
			a.source,
			COALESCE(a.subject, ''), COALESCE(a.grade_range, ''),
			COALESCE(a.scenes::text, '[]'),
			a.use_count, a.avg_score,
			a.is_active, COALESCE(a.is_default_for_scene::text, '[]'),
			a.created_by, a.organization_id,
			COALESCE(u.display_name, '')  AS creator_name,
			COALESCE(o.name, '')          AS school_name,
			a.created_at, a.updated_at
		FROM ai_assistants a
		LEFT JOIN users u        ON u.id = a.created_by
		LEFT JOIN organizations o ON o.id = a.organization_id
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
		var createdBy, orgID *string
		err := rows.Scan(
			&item.ID, &item.Name, &item.AvatarEmoji, &item.Description,
			&item.Source,
			&item.Subject, &item.GradeRange,
			&scenesJSON,
			&item.UseCount, &item.AvgScore,
			&item.IsActive, &defaultJSON,
			&createdBy, &orgID,
			&item.CreatorName,
			&item.SchoolName,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描 AI 助手行失败: %w", err)
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

		// 编辑/删除权限判断
		item.CanEdit = canEditAssistant(item.Source, createdBy, orgID, params)
		item.CanDelete = item.CanEdit && item.Source != models.AssistantSourceSystem

		items = append(items, item)
	}

	if items == nil {
		items = []*models.AIAssistantListItem{}
	}
	return items, total, nil
}

// canEditAssistant 判断当前用户能否编辑该助手
// - system:仅 admin 可编辑
// - group: 仅本校 senior_operator 可编辑(管理本校学校的管理员)
// - personal:仅创建者本人可编辑
func canEditAssistant(source string, createdBy *string, orgID *string, params *models.ListAIAssistantsParams) bool {
	switch source {
	case models.AssistantSourceSystem:
		return params.CurrentUserRole == models.RoleAdmin
	case models.AssistantSourceGroup:
		if params.CurrentUserRole != models.RoleSeniorOperator {
			return false
		}
		if orgID == nil || params.CurrentSchoolID == "" {
			return false
		}
		return *orgID == params.CurrentSchoolID
	case models.AssistantSourcePersonal:
		if createdBy == nil || params.CurrentUserID == "" {
			return false
		}
		return *createdBy == params.CurrentUserID
	}
	return false
}

// ==================== 更新 ====================

// UpdateAIAssistant 更新助手
// 只允许修改:name/avatar/description/full_prompt/subject/grade_range/scenes/is_active
// 不允许修改:source/created_by/organization_id(归属永久不变)
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
