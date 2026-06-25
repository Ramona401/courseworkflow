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
// 调用方负责设置 Source / CreatedBy / OrganizationID / GroupID 等归属字段
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
//
// 里程碑一可见性(OR 并集):
//   system(所有人) + 教研组级 group(我所属的组) + 全校级 group(同校) + personal(自己)
//
// v115改动:GradeRange 过滤前先调 NormalizeGradeToSegment 归一化为学段
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

        // 查询列表(带创建者名、学校名、教研组名)
        listQuery := `
                SELECT a.id, a.name, a.avatar_emoji, COALESCE(a.description, ''),
                        a.source,
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
                item.CanEdit = canEditAssistant(item.Source, createdBy, orgID, groupID, params)
                item.CanDelete = item.CanEdit && item.Source != models.AssistantSourceSystem

                items = append(items, item)
        }

        if items == nil {
                items = []*models.AIAssistantListItem{}
        }
        return items, total, nil
}

// canEditAssistant 判断当前用户能否编辑该助手(列表层快速提示)
//
// ⚠ 说明:本函数仅用于列表项 CanEdit/CanDelete 的"按钮显隐提示",
//   不是权限的最终防线。真正的编辑/删除拦截在 service 层 canEdit(),
//   那里有完整的 MyLeadGroupIDs,能精确判定"组长可改本组其他人建的助手"。
//
//   本函数为保持改动收敛,对教研组级助手只放行"创建者本人 + admin",
//   不放行"组长改组员建的组助手"(此情况按钮不显示,但组长仍可经正常编辑入口操作,
//   service 层会放行)。这是 UI 提示偏保守、不影响安全与功能正确性的有意取舍。
//
//   规则:
//     system     → 仅 admin
//     group      → admin / 创建者本人(教研组级组长的额外编辑权由 service 层兜底)
//     personal   → 仅创建者本人
func canEditAssistant(source string, createdBy *string, orgID *string, groupID *string, params *models.ListAIAssistantsParams) bool {
        if params.CurrentUserRole == models.RoleAdmin {
                return true
        }
        switch source {
        case models.AssistantSourceSystem:
                return false
        case models.AssistantSourceGroup:
                // 创建者本人可编辑(教研组级 / 全校级通用)
                if createdBy != nil && params.CurrentUserID != "" && *createdBy == params.CurrentUserID {
                        return true
                }
                return false
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
// 不允许修改:source/created_by/organization_id/group_id(归属永久不变)
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
