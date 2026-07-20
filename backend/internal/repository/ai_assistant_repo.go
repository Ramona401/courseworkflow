package repository

// ai_assistant_repo.go — AI助手数据访问层
//
// 职责：
//   - CRUD：CreateAIAssistant / GetAIAssistantByID / ListAIAssistants /
//     UpdateAIAssistant / DeleteAIAssistant；
//   - 使用量统计：IncrementAIAssistantUseCount；
//   - 在候选列表数据库查询阶段执行教育域第一层硬隔离。
//
// 可见性规则由两个彼此正交的维度共同决定：
//
//  1. 原有来源与组织可见性：
//     - system：系统助手；
//     - group且group_id非空：当前用户所属教研组；
//     - group且group_id为空：当前用户所属学校；
//     - personal：当前用户本人创建。
//
//  2. 教育域可见性：
//     - k12、vocational、adult教学上下文：
//       只允许资源education_domain等于当前教学域，或者等于common；
//     - mixed跨域管理上下文：
//       不增加教育域过滤，保留跨域管理能力；
//     - 空值、非法值或common作为操作者上下文：
//       使用AND 1=0严格拒绝，避免错误回退K12。
//
// system只是助手来源，不再代表跨教育域通用。
// 真正跨教育域通用的教学资源必须明确保存为education_domain=common。
//
// share_policy与教育域隔离继续正交：
//   - open / use_only / locked只决定可复制、可编辑和原文保护；
//   - education_domain决定资源能否进入当前教学域候选集合。
//
// 列表层权限字段CanEdit、CanDelete、CanFork和CanViewPrompt只用于前端按钮提示，
// 最终权限仍由Service层执行。

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

// CreateAIAssistant 创建助手记录。
//
// education_domain不由本函数显式写入，继续交给数据库BEFORE INSERT触发器确定：
//   - Fork继承来源助手教育域；
//   - system助手默认k12；
//   - 其它助手按创建者当前教学教育域。
//
// INSERT完成后通过RETURNING回填最终education_domain，保证返回对象与数据库快照一致。
func CreateAIAssistant(
	ctx context.Context,
	a *models.AIAssistant,
) error {
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

	// share_policy为空或非法时统一回落use_only，与数据库默认值保持一致。
	if !models.IsValidSharePolicy(a.SharePolicy) {
		a.SharePolicy = models.SharePolicyUseOnly
	}

	query := `
		INSERT INTO ai_assistants (
			name,
			avatar_emoji,
			description,
			source,
			created_by,
			organization_id,
			group_id,
			share_policy,
			full_prompt,
			knowledge_refs,
			subject,
			grade_range,
			scenes,
			forked_from,
			sort_order,
			is_default_for_scene,
			is_active
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17
		)
		RETURNING
			id,
			education_domain,
			created_at,
			updated_at
	`

	err := database.DB.QueryRow(
		ctx,
		query,
		a.Name,
		a.AvatarEmoji,
		a.Description,
		a.Source,
		a.CreatedBy,
		a.OrganizationID,
		a.GroupID,
		a.SharePolicy,
		a.FullPrompt,
		a.KnowledgeRefs,
		a.Subject,
		a.GradeRange,
		a.Scenes,
		a.ForkedFrom,
		a.SortOrder,
		a.IsDefaultForScene,
		a.IsActive,
	).Scan(
		&a.ID,
		&a.EducationDomain,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建 AI 助手失败: %w", err)
	}

	return nil
}

// ==================== 查询单个 ====================

// GetAIAssistantByID 根据ID读取完整助手实体。
//
// 本函数只忠实读取数据，不判断组织可见性和教育域权限。
// 所有按ID使用路径必须由Service层再次执行可见性和资源教育域校验。
func GetAIAssistantByID(
	ctx context.Context,
	id string,
) (*models.AIAssistant, error) {
	a := &models.AIAssistant{}

	query := `
		SELECT
			id,
			name,
			avatar_emoji,
			COALESCE(description, ''),
			source,
			created_by,
			organization_id,
			group_id,
			education_domain,
			COALESCE(share_policy, 'use_only'),
			full_prompt,
			COALESCE(knowledge_refs::text, '[]'),
			COALESCE(subject, ''),
			COALESCE(grade_range, ''),
			COALESCE(scenes::text, '[]'),
			creation_conversation::text,
			forked_from,
			use_count,
			avg_score,
			sort_order,
			COALESCE(is_default_for_scene::text, '[]'),
			is_active,
			created_at,
			updated_at
		FROM ai_assistants
		WHERE id = $1
	`

	err := database.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&a.ID,
		&a.Name,
		&a.AvatarEmoji,
		&a.Description,
		&a.Source,
		&a.CreatedBy,
		&a.OrganizationID,
		&a.GroupID,
		&a.EducationDomain,
		&a.SharePolicy,
		&a.FullPrompt,
		&a.KnowledgeRefs,
		&a.Subject,
		&a.GradeRange,
		&a.Scenes,
		&a.CreationConversation,
		&a.ForkedFrom,
		&a.UseCount,
		&a.AvgScore,
		&a.SortOrder,
		&a.IsDefaultForScene,
		&a.IsActive,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAIAssistantNotFound
		}
		return nil, fmt.Errorf("查询 AI 助手失败: %w", err)
	}

	return a, nil
}

// ==================== 列表查询 ====================

// ListAIAssistants 按原有组织可见性和教育域可见性返回候选助手。
//
// 教育域规则：
//   - 当前域为k12、vocational或adult：当前域资源 + common资源；
//   - 当前域为mixed：不增加域过滤，供跨域管理页面使用；
//   - 当前域为空、非法或common：AND 1=0，严格返回空集合。
//
// 具体教案运行时，调用方必须把Actor.EducationDomain覆盖为教案快照域，
// 因而即使操作者本身是mixed管理员，也只会得到该教案域和common资源。
func ListAIAssistants(
	ctx context.Context,
	params *models.ListAIAssistantsParams,
) (
	[]*models.AIAssistantListItem,
	int,
	error,
) {
	if params == nil {
		return []*models.AIAssistantListItem{}, 0, nil
	}

	visibilityClauses := make([]string, 0, 4)
	args := make([]interface{}, 0)
	argIdx := 1

	// system仍参与来源可见性并集，但后面必须继续经过education_domain过滤。
	visibilityClauses = append(
		visibilityClauses,
		"a.source = 'system'",
	)

	// 教研组级助手：只对当前用户所属教研组开放。
	if len(params.CurrentGroupIDs) > 0 {
		visibilityClauses = append(
			visibilityClauses,
			fmt.Sprintf(
				"(a.source = 'group' AND a.group_id IS NOT NULL AND a.group_id::text = ANY($%d))",
				argIdx,
			),
		)
		args = append(args, params.CurrentGroupIDs)
		argIdx++
	}

	// 全校级助手：只对当前用户所属学校开放。
	if strings.TrimSpace(params.CurrentSchoolID) != "" {
		visibilityClauses = append(
			visibilityClauses,
			fmt.Sprintf(
				"(a.source = 'group' AND a.group_id IS NULL AND a.organization_id = $%d)",
				argIdx,
			),
		)
		args = append(args, strings.TrimSpace(params.CurrentSchoolID))
		argIdx++
	}

	// 个人助手：只对创建者本人开放。
	if strings.TrimSpace(params.CurrentUserID) != "" {
		visibilityClauses = append(
			visibilityClauses,
			fmt.Sprintf(
				"(a.source = 'personal' AND a.created_by = $%d)",
				argIdx,
			),
		)
		args = append(args, strings.TrimSpace(params.CurrentUserID))
		argIdx++
	}

	where := " WHERE (" + strings.Join(visibilityClauses, " OR ") + ")"

	// 第一层资源教育域硬隔离。
	//
	// mixed只用于跨域管理页面，因此不拼域条件。
	// 具体教学运行时，mixed管理员Actor会被调用方覆盖为教案快照域，
	// 届时自然进入下面的具体教学域分支。
	currentEducationDomain := strings.ToLower(
		strings.TrimSpace(params.CurrentEducationDomain),
	)

	switch {
	case currentEducationDomain == models.EducationDomainMixed:
		// mixed管理上下文允许跨教育域查看，不增加过滤。

	case models.IsTeachingEducationDomain(currentEducationDomain):
		where += fmt.Sprintf(
			" AND (a.education_domain = $%d OR a.education_domain = $%d)",
			argIdx,
			argIdx+1,
		)
		args = append(
			args,
			currentEducationDomain,
			models.EducationDomainCommon,
		)
		argIdx += 2

	default:
		// 空值、非法值和common都不是合法的操作者教学上下文。
		// 严格匹配空集合，绝不回退K12。
		where += " AND 1 = 0"
	}

	// locked表示仅属主或admin可见。
	// mixed只控制教育域范围，不改变原有产权保护。
	if params.CurrentUserRole != models.RoleAdmin {
		if strings.TrimSpace(params.CurrentUserID) != "" {
			where += fmt.Sprintf(
				" AND NOT (a.share_policy = 'locked' AND (a.created_by IS NULL OR a.created_by <> $%d))",
				argIdx,
			)
			args = append(
				args,
				strings.TrimSpace(params.CurrentUserID),
			)
			argIdx++
		} else {
			where += " AND a.share_policy <> 'locked'"
		}
	}

	if params.OnlyActive {
		where += " AND a.is_active = true"
	}

	if strings.TrimSpace(params.Subject) != "" {
		where += fmt.Sprintf(
			" AND a.subject = $%d",
			argIdx,
		)
		args = append(
			args,
			strings.TrimSpace(params.Subject),
		)
		argIdx++
	}

	// 年级保留Go层严格同义归一匹配，
	// 避免“高三/十二年级/12年级/12”等合法同义值在SQL层误伤。
	if strings.TrimSpace(params.Scene) != "" {
		where += fmt.Sprintf(
			" AND a.scenes @> $%d::jsonb",
			argIdx,
		)
		args = append(
			args,
			fmt.Sprintf(
				`["%s"]`,
				strings.TrimSpace(params.Scene),
			),
		)
		argIdx++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM ai_assistants a
	` + where

	var total int
	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"统计 AI 助手数量失败: %w",
			err,
		)
	}

	listQuery := `
		SELECT
			a.id,
			a.name,
			a.avatar_emoji,
			COALESCE(a.description, ''),
			a.source,
			a.education_domain,
			COALESCE(a.share_policy, 'use_only'),
			COALESCE(a.subject, ''),
			COALESCE(a.grade_range, ''),
			COALESCE(a.scenes::text, '[]'),
			a.use_count,
			a.avg_score,
			a.is_active,
			COALESCE(a.is_default_for_scene::text, '[]'),
			a.created_by,
			a.organization_id,
			a.group_id,
			COALESCE(u.display_name, '') AS creator_name,
			COALESCE(o.name, '') AS school_name,
			COALESCE(tg.name, '') AS group_name,
			a.created_at,
			a.updated_at
		FROM ai_assistants a
		LEFT JOIN users u
			ON u.id = a.created_by
		LEFT JOIN organizations o
			ON o.id = a.organization_id
		LEFT JOIN teaching_groups tg
			ON tg.id = a.group_id
	` + where + `
		ORDER BY
			CASE a.source
				WHEN 'system' THEN 0
				WHEN 'group' THEN 1
				ELSE 2
			END,
			a.sort_order DESC,
			a.created_at ASC
	`

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"查询 AI 助手列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.AIAssistantListItem,
		0,
	)

	for rows.Next() {
		item := &models.AIAssistantListItem{}
		var scenesJSON string
		var defaultJSON string
		var createdBy *string
		var organizationID *string
		var groupID *string

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.AvatarEmoji,
			&item.Description,
			&item.Source,
			&item.EducationDomain,
			&item.SharePolicy,
			&item.Subject,
			&item.GradeRange,
			&scenesJSON,
			&item.UseCount,
			&item.AvgScore,
			&item.IsActive,
			&defaultJSON,
			&createdBy,
			&organizationID,
			&groupID,
			&item.CreatorName,
			&item.SchoolName,
			&item.GroupName,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描 AI 助手行失败: %w",
				err,
			)
		}

		item.GroupID = groupID

		// 具体年级严格过滤：
		// 高三只接受高三、十二年级、12年级、12等同义表达。
		if strings.TrimSpace(params.GradeRange) != "" &&
			!utils.IsStrictGradeMatch(
				item.GradeRange,
				params.GradeRange,
			) {
			continue
		}

		var scenes []string
		_ = json.Unmarshal(
			[]byte(scenesJSON),
			&scenes,
		)
		if scenes == nil {
			scenes = []string{}
		}
		item.Scenes = scenes

		if strings.TrimSpace(params.Scene) != "" {
			var defaults []string
			_ = json.Unmarshal(
				[]byte(defaultJSON),
				&defaults,
			)
			for _, candidate := range defaults {
				if strings.TrimSpace(candidate) ==
					strings.TrimSpace(params.Scene) {
					item.IsDefaultHere = true
					break
				}
			}
		}

		if label, ok := models.SourceLabelMap[item.Source]; ok {
			item.SourceLabel = label
		} else {
			item.SourceLabel = item.Source
		}

		item.CanEdit = canEditAssistant(
			item.Source,
			item.SharePolicy,
			createdBy,
			organizationID,
			groupID,
			params,
		)
		item.CanDelete =
			item.CanEdit &&
				item.Source != models.AssistantSourceSystem

		item.CanFork = canForkAssistant(
			item.SharePolicy,
			createdBy,
			params,
		)

		item.CanViewPrompt = canViewPromptAssistant(
			item.Source,
			item.SharePolicy,
			createdBy,
			organizationID,
			groupID,
			params,
		)

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历 AI 助手列表失败: %w",
			err,
		)
	}

	// 年级在Go层过滤，因此以过滤后的切片长度作为真实total。
	if strings.TrimSpace(params.GradeRange) != "" {
		total = len(items)
	}

	return items, total, nil
}

// canEditAssistant 判断当前用户能否编辑助手。
//
// 本函数只计算列表按钮提示，最终权限由Service层canEdit执行。
func canEditAssistant(
	source string,
	sharePolicy string,
	createdBy *string,
	_ *string,
	groupID *string,
	params *models.ListAIAssistantsParams,
) bool {
	if params == nil {
		return false
	}

	if params.CurrentUserRole == models.RoleAdmin {
		return true
	}

	isOwner :=
		createdBy != nil &&
			strings.TrimSpace(params.CurrentUserID) != "" &&
			*createdBy == strings.TrimSpace(params.CurrentUserID)

	switch source {
	case models.AssistantSourceSystem:
		return false

	case models.AssistantSourceGroup:
		if isOwner {
			return true
		}

		if groupID != nil &&
			strings.TrimSpace(*groupID) != "" &&
			containsStrRepo(
				params.CurrentLeadGroupIDs,
				*groupID,
			) {
			return sharePolicy != models.SharePolicyLocked
		}

		return false

	case models.AssistantSourcePersonal:
		return isOwner
	}

	return false
}

// containsStrRepo 判断字符串切片是否包含目标值。
func containsStrRepo(
	list []string,
	target string,
) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// canViewPromptAssistant 判断当前用户能否查看full_prompt原文。
//
// open助手对所有已通过候选过滤的人开放原文；
// 其它策略复用编辑权限闸门。
func canViewPromptAssistant(
	source string,
	sharePolicy string,
	createdBy *string,
	organizationID *string,
	groupID *string,
	params *models.ListAIAssistantsParams,
) bool {
	if sharePolicy == models.SharePolicyOpen {
		return true
	}

	return canEditAssistant(
		source,
		sharePolicy,
		createdBy,
		organizationID,
		groupID,
		params,
	)
}

// canForkAssistant 判断当前用户能否把助手复制为个人副本。
//
// 本函数只计算列表按钮提示，Service层ForkAssistant仍是最终权限防线。
func canForkAssistant(
	sharePolicy string,
	createdBy *string,
	params *models.ListAIAssistantsParams,
) bool {
	if params == nil {
		return false
	}

	if params.CurrentUserRole == models.RoleAdmin {
		return true
	}

	if sharePolicy == models.SharePolicyOpen {
		return true
	}

	return createdBy != nil &&
		strings.TrimSpace(params.CurrentUserID) != "" &&
		*createdBy == strings.TrimSpace(params.CurrentUserID)
}

// ==================== 更新 ====================

// UpdateAIAssistant 更新助手内容和匹配维度。
//
// 不允许修改source、created_by、organization_id、group_id和education_domain。
// education_domain是创建时快照，后续编辑不得重分类。
func UpdateAIAssistant(
	ctx context.Context,
	id string,
	req *models.UpdateAIAssistantRequest,
) error {
	scenesJSON, err := json.Marshal(req.Scenes)
	if err != nil {
		return fmt.Errorf(
			"序列化场景列表失败: %w",
			err,
		)
	}
	if len(req.Scenes) == 0 {
		scenesJSON = []byte("[]")
	}

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
		req.Name,
		req.AvatarEmoji,
		req.Description,
		req.FullPrompt,
		req.Subject,
		req.GradeRange,
		string(scenesJSON),
	}
	argIdx := 8

	if req.IsActive != nil {
		setParts = append(
			setParts,
			fmt.Sprintf(
				"is_active = $%d",
				argIdx,
			),
		)
		args = append(args, *req.IsActive)
		argIdx++
	}

	if req.SharePolicy != nil &&
		models.IsValidSharePolicy(*req.SharePolicy) {
		setParts = append(
			setParts,
			fmt.Sprintf(
				"share_policy = $%d",
				argIdx,
			),
		)
		args = append(
			args,
			*req.SharePolicy,
		)
		argIdx++
	}

	query := fmt.Sprintf(
		`UPDATE ai_assistants SET %s WHERE id = $%d`,
		strings.Join(setParts, ", "),
		argIdx,
	)
	args = append(args, id)

	result, err := database.DB.Exec(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return fmt.Errorf(
			"更新 AI 助手失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrAIAssistantNotFound
	}

	return nil
}

// ==================== 删除 ====================

// DeleteAIAssistant 硬删除助手。
// 调用方负责确认助手允许删除。
func DeleteAIAssistant(
	ctx context.Context,
	id string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`DELETE FROM ai_assistants WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf(
			"删除 AI 助手失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrAIAssistantNotFound
	}

	return nil
}

// ==================== 使用量统计 ====================

// IncrementAIAssistantUseCount 增加助手真实使用次数。
func IncrementAIAssistantUseCount(
	ctx context.Context,
	id string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
			UPDATE ai_assistants
			SET
				use_count = use_count + 1,
				updated_at = now()
			WHERE id = $1
		`,
		id,
	)
	return err
}
