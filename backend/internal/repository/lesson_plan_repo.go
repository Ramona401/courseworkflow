package repository

// lesson_plan_repo.go — 教案数据访问层（主文件）
//
// 职责：
//   - 错误常量
//   - 教案CRUD（创建/查询/列表/更新内容/更新状态/更新可见范围/更新评审/Fork/删除）
//   - 教案评审CRUD（创建/列表）
//
// v56修改：CreateLessonPlan/GetLessonPlanByID 增加 recipe_id 字段
// v58修改：GetLessonPlanByID 增加 current_stage + stage_config 字段
// 迭代7B修改：CreateLessonPlan 增加 textbook_page_ids 字段
//             GetLessonPlanByID 增加 textbook_page_ids 字段
// 大单元备课修改：GetLessonPlanByID 增加 unit_plan_id 字段（挂载的单元方案ID）
//                 新增 UpdateLessonPlanUnitPlanID 方法（挂载/解除单元方案）
// 提示词模板+组件萃取+对话记录 → lesson_plan_repo_ext.go
//
// 迭代一 Phase 4（数据隔离收口）修改：
//   ListLessonPlans 新增数据范围白名单参数 scopeUserIDs/scopeIsAdmin，
//   用于把"当前请求者能看到哪些教案"收口到统一的数据范围控制。
//
//   为什么 repo 层只收 (scopeUserIDs []string, scopeIsAdmin bool) 两个原始参数，
//   而不直接收 services.DataScope 结构体？——因为包依赖方向是 services → repository，
//   repository 不能反向 import services（否则构成循环依赖）。这与 token_account_repo.go
//   的 ListTokenAccounts 只收 ownerIDs []string（而非 services.TokenScope）完全一致：
//   由上层 service 从 DataScope 里拆出 UserIDs/IsAdmin 再传进来。
//
//   白名单三态语义（与 token 体系、repository 层其它查询完全一致）：
//     - scopeIsAdmin == true             → 不拼任何可见性过滤（admin 看全部）
//     - scopeIsAdmin == false            → 拼"可见性 OR 子句"（见下）
//       · scopeUserIDs 非空              → author_id = ANY(白名单) 命中本人/管辖成员
//       · scopeUserIDs 为非nil空切片     → author_id = ANY('{}') 匹配空集（fail-closed），
//                                           但共享教案仍可见（OR 的 B 分支）
//
// 「我的教案只显示 1 条」Bug 修复（caller 恒放行）：
//   根因：senior_operator 未绑学校管理员时，service 层 ResolveDataScope 走 senior 分支
//   fail-closed 成 Blocked，scopeUserIDs 传进来是空集 []。我的教案页前端传 author_id=自己，
//   原可见性 OR 子句 (author_id = ANY(空集) OR status IN 共享) 把作者自己的草稿全挡掉，
//   只剩名下 approved 那 1 条命中"共享可见"分支，列表只显示 1 条。
//   修法：ListLessonPlans 新增 callerID string 入参（当前登录用户自己）。
//   仅当前端显式传了 authorID 且 authorID == callerID（即"我的教案页"语义）时，
//   在可见性 OR 子句里额外追加一条"作者是当前用户自己则恒放行"分支：
//       AND ( lp.author_id = $caller                       -- D：自己恒放行（新增）
//          OR lp.author_id = ANY($scope)                   -- A+C：本人 ∪ 管辖范围成员
//          OR lp.status IN ('published_shared','approved') )-- B：共享可见
//   这样一个人无论 scope 怎么算，至少永远能看到自己的全部教案。
//   为什么只在 authorID==callerID 时才加 D 分支、而不是无条件加？——
//   因为教案库页（前端传 status=published_shared/approved、不传 author_id）若也无条件
//   追加 D 分支，会让用户在教案库里额外看到自己的全部草稿，污染教案库可见性。
//   只在"我的教案页"语义下追加 D 分支，教案库页的既有行为完全不变。
//
//   可见性 OR 子句（非 admin 时追加）：
//       AND (
//           lp.author_id = ANY($n)                          -- A+C：本人 ∪ 管辖范围成员
//           OR lp.status IN ('published_shared','approved')  -- B：共享可见
//           [ OR lp.author_id = $caller ]                    -- D：仅"我的教案页"语义下追加
//       )
//   设计依据：经数据库实测，lesson_plans.school_id 长期全为空（归属完全靠 author_id），
//   因此 C 分支（管辖）不能用 school_id 过滤，必须并入 author_id 白名单——
//   而 DataScope.UserIDs 对 senior 恰好是"本校全体成员"、对 region_admin 是"区域树下全体成员"、
//   对 operator/viewer 是"自己"，故 A（本人）与 C（管辖）可合并为同一条 author_id = ANY。
//   B 分支保证任何登录用户都能在教案库浏览已发布/已通过的共享教案（不回退 LibraryPage）。
//   D 分支保证作者在"我的教案页"始终能看到自己全部教案，即便其 scope 被误伤为空集。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrLessonPlanNotFound = errors.New("教案不存在")
	ErrTemplateNotFound   = errors.New("提示词模板不存在")
	ErrExtractionNotFound = errors.New("萃取记录不存在")
)

// ==================== 教案CRUD ====================

// CreateLessonPlan 创建教案
// v56：新增recipe_id字段
// 迭代7B：新增textbook_page_ids字段
//
// 注意：本 INSERT 为显式列名，未列入 unit_plan_id，
// 故新建教案 unit_plan_id 默认 NULL（=未挂载单元方案），符合预期。
// 挂载是后续显式操作，不在创建时进行。
func CreateLessonPlan(ctx context.Context, lp *models.LessonPlan) error {
	query := `
		INSERT INTO lesson_plans (
			title, subject, grade, topic, duration_minutes,
			content_markdown, content_structured, generation_config,
			matched_components, conversation_log,
			status, visibility, author_id, group_id, school_id, template_id, recipe_id,
			textbook_page_ids
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17,
			$18
		)
		RETURNING id, created_at, updated_at
	`
	dur := lp.DurationMinutes
	if dur <= 0 {
		dur = 45
	}
	contentStruct := lp.ContentStructured
	if contentStruct == "" {
		contentStruct = "{}"
	}
	genConfig := lp.GenerationConfig
	if genConfig == "" {
		genConfig = "{}"
	}
	matchedComp := lp.MatchedComponents
	if matchedComp == "" {
		matchedComp = "[]"
	}
	convLog := lp.ConversationLog
	if convLog == "" {
		convLog = "[]"
	}
	status := lp.Status
	if status == "" {
		status = "draft"
	}
	visibility := lp.Visibility
	if visibility == "" {
		visibility = "personal"
	}
	// 迭代7B：课本图片ID列表默认空数组
	textbookIDs := lp.TextbookPageIDs
	if textbookIDs == "" {
		textbookIDs = "[]"
	}

	err := database.DB.QueryRow(ctx, query,
		lp.Title, lp.Subject, lp.Grade, lp.Topic, dur,
		lp.ContentMarkdown, contentStruct, genConfig, matchedComp, convLog,
		status, visibility, lp.AuthorID, lp.GroupID, lp.SchoolID, lp.TemplateID, lp.RecipeID,
		textbookIDs,
	).Scan(&lp.ID, &lp.CreatedAt, &lp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建教案失败: %w", err)
	}
	return nil
}

// GetLessonPlanByID 根据ID查询教案
// v56：新增recipe_id字段
// v58：新增current_stage + stage_config字段
// 迭代7B：新增textbook_page_ids字段
// 大单元备课：新增 unit_plan_id 字段（挂载的单元方案ID，nil=未挂载）
//
// unit_plan_id 是可空 UUID，存量教案全为 NULL。
// 照搬同文件 review_school_id 的成熟 NULL 处理写法：
// SELECT 用 COALESCE(unit_plan_id::text, '') 把 NULL 转空串，
// Scan 进临时字符串变量 unitPlanIDStr，非空才赋给 lp.UnitPlanID（空串=未挂载，留 nil）。
func GetLessonPlanByID(ctx context.Context, id string) (*models.LessonPlan, error) {
	lp := &models.LessonPlan{}
	var reviewSchoolIDStr string
	var unitPlanIDStr string
	query := `
		SELECT id, title, subject, grade, topic, duration_minutes,
		       content_markdown, content_structured, generation_config,
		       matched_components, conversation_log,
		       ai_review_score, ai_review_result, ai_review_history,
		       status, visibility, author_id, group_id, school_id,
		       forked_from, fork_count, template_id, recipe_id,
		       COALESCE(unit_plan_id::text, '') AS unit_plan_id,
		       view_count, use_count, version,
		       current_stage, COALESCE(stage_config::text, '[]'),
		       COALESCE(textbook_page_ids::text, '[]'),
		       COALESCE(lesson_index, ''), idx_cognitive_level, idx_pedagogy_intensity,
		       idx_structure_type, idx_quality_level,
		       review_level, COALESCE(review_school_id::text, ''),
		       created_at, updated_at
		FROM lesson_plans WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&lp.ID, &lp.Title, &lp.Subject, &lp.Grade, &lp.Topic, &lp.DurationMinutes,
		&lp.ContentMarkdown, &lp.ContentStructured, &lp.GenerationConfig,
		&lp.MatchedComponents, &lp.ConversationLog,
		&lp.AIReviewScore, &lp.AIReviewResult, &lp.AIReviewHistory,
		&lp.Status, &lp.Visibility, &lp.AuthorID, &lp.GroupID, &lp.SchoolID,
		&lp.ForkedFrom, &lp.ForkCount, &lp.TemplateID, &lp.RecipeID,
		&unitPlanIDStr,
		&lp.ViewCount, &lp.UseCount, &lp.Version,
		&lp.CurrentStage, &lp.StageConfig,
		&lp.TextbookPageIDs,
		&lp.LessonIndex, &lp.IdxCognitiveLevel, &lp.IdxPedagogyIntensity,
		&lp.IdxStructureType, &lp.IdxQualityLevel,
		&lp.ReviewLevel, &reviewSchoolIDStr,
		&lp.CreatedAt, &lp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLessonPlanNotFound
		}
		return nil, fmt.Errorf("查询教案失败: %w", err)
	}
	if reviewSchoolIDStr != "" {
		lp.ReviewSchoolID = &reviewSchoolIDStr
	}
	// 大单元备课：空串=未挂载，留 nil；非空才赋值
	if unitPlanIDStr != "" {
		lp.UnitPlanID = &unitPlanIDStr
	}
	return lp, nil
}

// ListLessonPlans 获取教案列表（支持多条件筛选+分页+数据范围白名单）
//
// 迭代一 Phase 4（数据隔离收口）：scopeUserIDs/scopeIsAdmin 两个数据范围参数。
//   - scopeIsAdmin=true  → 不追加任何可见性过滤，看全部（admin）。
//   - scopeIsAdmin=false → 追加可见性 OR 子句：
//     AND ( lp.author_id = ANY(scopeUserIDs)            -- 本人 ∪ 管辖范围成员
//     OR lp.status IN ('published_shared','approved')  -- 共享可见
//     [ OR lp.author_id = callerID ] )                 -- 仅"我的教案页"语义下追加
//     其中 scopeUserIDs 三态：非空=名单内作者；非nil空切片=匹配空集(fail-closed，
//     共享教案仍可见)；本函数不接受 nil 非 admin（service 层保证非 admin 必传非 nil 切片）。
//
// 「我的教案只显示 1 条」Bug 修复：
//   新增 callerID string 入参（当前登录用户自己）。当 authorID != "" 且 authorID == callerID
//   （即"我的教案页"，前端按本人 ID 过滤）时，在可见性 OR 子句里追加 D 分支
//   "lp.author_id = callerID"，保证作者无论 scope 怎么算都能看到自己的全部教案。
//   教案库页（不传 authorID）不满足该条件，不追加 D 分支，可见性行为完全不变。
//
// 该可见性子句与前面的 author_id/group_id/status/subject/grade/索引维度等具体筛选是 AND 叠加关系：
//   - 我的教案页（前端传 author_id=自己）：AND 段已限定本人，OR 段 D 分支恒命中，结果即本人全部教案。
//   - 教案库页（前端传 status=published_shared / approved，不传 author_id）：
//     AND 段限定状态，OR 段命中 B 分支（共享可见），任何登录用户都能浏览，不回退。
//
// 注意：本列表查询为显式列名，未查 unit_plan_id（列表无需展示挂载状态），故加列不影响本函数。
func ListLessonPlans(ctx context.Context, callerID string, authorID string, groupID string, status string, subject string, grade string, limit int, offset int, qualityLevel int, structureType int, cognitiveLevel int, pedagogyIntensity int, scopeUserIDs []string, scopeIsAdmin bool) ([]*models.LessonPlanListItem, int, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if authorID != "" {
		where += fmt.Sprintf(" AND lp.author_id = $%d", argIdx)
		args = append(args, authorID)
		argIdx++
	}
	if groupID != "" {
		where += fmt.Sprintf(" AND lp.group_id = $%d", argIdx)
		args = append(args, groupID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND lp.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if subject != "" {
		where += fmt.Sprintf(" AND lp.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}
	if grade != "" {
		where += fmt.Sprintf(" AND lp.grade = $%d", argIdx)
		args = append(args, grade)
		argIdx++
	}

	// v86新增：AOCI索引维度筛选
	if qualityLevel > 0 {
		where += fmt.Sprintf(" AND lp.idx_quality_level >= $%d", argIdx)
		args = append(args, qualityLevel)
		argIdx++
	}
	if structureType > 0 {
		where += fmt.Sprintf(" AND lp.idx_structure_type = $%d", argIdx)
		args = append(args, structureType)
		argIdx++
	}
	if cognitiveLevel > 0 {
		where += fmt.Sprintf(" AND lp.idx_cognitive_level >= $%d", argIdx)
		args = append(args, cognitiveLevel)
		argIdx++
	}
	if pedagogyIntensity > 0 {
		where += fmt.Sprintf(" AND lp.idx_pedagogy_intensity = $%d", argIdx)
		args = append(args, pedagogyIntensity)
		argIdx++
	}

	// 迭代一 Phase 4：数据范围可见性 OR 子句（非 admin 才追加）
	// admin（scopeIsAdmin=true）不拼此段，看全部教案。
	// 非 admin 追加：author_id 命中白名单（本人∪管辖成员） OR 状态为共享可见
	//               [ OR 作者是当前用户自己（仅"我的教案页"语义下追加，见下） ]。
	if !scopeIsAdmin {
		// 「我的教案只显示 1 条」Bug 修复：判定是否为"我的教案页"语义。
		// 仅当前端显式传了 authorID 且该 authorID 就是当前登录用户自己（callerID）时，
		// 才追加 D 分支"lp.author_id = $caller"，让作者无论 scope 怎么算都能看到自己全部教案。
		// 教案库页（不传 authorID，或 authorID 指向他人）不满足，不追加 D 分支，
		// 既有可见性（本人∪管辖 ∪ 共享）行为完全不变，避免教案库被自己的草稿污染。
		isMyPlansView := authorID != "" && callerID != "" && authorID == callerID

		if isMyPlansView {
			// 占两个占位符：$scope（白名单 ANY）+ $caller（自己恒放行）
			scopePlaceholder := argIdx
			callerPlaceholder := argIdx + 1
			where += fmt.Sprintf(
				" AND (lp.author_id = ANY($%d) OR lp.status IN ('published_shared','approved') OR lp.author_id = $%d)",
				scopePlaceholder, callerPlaceholder,
			)
			args = append(args, scopeUserIDs, callerID)
			argIdx += 2
		} else {
			// 非"我的教案页"：保持原有可见性 OR 子句（不含 D 分支）。
			// scopeUserIDs 即使为空切片也照常拼 ANY('{}')，匹配空集（fail-closed），
			// 此时仅靠 B 分支放行共享教案——这是"未绑校 senior 至少能看公开教案"的合理兜底。
			where += fmt.Sprintf(
				" AND (lp.author_id = ANY($%d) OR lp.status IN ('published_shared','approved'))",
				argIdx,
			)
			args = append(args, scopeUserIDs)
			argIdx++
		}
	}

	var total int
	if err := database.DB.QueryRow(ctx, "SELECT COUNT(*) FROM lesson_plans lp"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询教案总数失败: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	listQuery := fmt.Sprintf(`
		SELECT lp.id, lp.title, lp.subject, lp.grade, lp.topic, lp.duration_minutes,
		       lp.status, lp.visibility, lp.author_id,
		       COALESCE(u.display_name, '') AS author_name,
		       lp.ai_review_score, lp.fork_count, lp.view_count,
		       lp.forked_from, lp.recipe_id,
		       COALESCE(tr.name, '') AS recipe_name,
		       COALESCE(lp.lesson_index, '') AS lesson_index,
		       lp.idx_quality_level,
		       lp.review_level,
		       lp.created_at, lp.updated_at
		FROM lesson_plans lp
		LEFT JOIN users u ON u.id = lp.author_id
		LEFT JOIN teaching_recipes tr ON tr.id = lp.recipe_id
		%s
		ORDER BY lp.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询教案列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.LessonPlanListItem
	for rows.Next() {
		item := &models.LessonPlanListItem{}
		err := rows.Scan(
			&item.ID, &item.Title, &item.Subject, &item.Grade, &item.Topic, &item.DurationMinutes,
			&item.Status, &item.Visibility, &item.AuthorID, &item.AuthorName,
			&item.AIReviewScore, &item.ForkCount, &item.ViewCount,
			&item.ForkedFrom, &item.RecipeID, &item.RecipeName, &item.LessonIndex, &item.IdxQualityLevel,
			&item.ReviewLevel,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描教案行失败: %w", err)
		}
		item.StatusName = models.LPStatusNameMap[item.Status]
		items = append(items, item)
	}
	return items, total, nil
}

// UpdateLessonPlanContent 更新教案内容
func UpdateLessonPlanContent(ctx context.Context, id string, title string, contentMd string, contentStruct string, durMinutes int) error {
	// 防御性编程：空字符串不是有效JSON，PostgreSQL的jsonb列会报错
	if contentStruct == "" {
		contentStruct = "{}"
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET title = $1, content_markdown = $2, content_structured = $3,
		    duration_minutes = $4, version = version + 1, updated_at = $5
		WHERE id = $6
	`, title, contentMd, contentStruct, durMinutes, now, id)
	if err != nil {
		return fmt.Errorf("更新教案内容失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// UpdateLessonPlanStatus 更新教案状态
func UpdateLessonPlanStatus(ctx context.Context, id string, status string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET status = $1, updated_at = $2 WHERE id = $3`,
		status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教案状态失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// UpdateLessonPlanVisibility 更新教案可见范围
func UpdateLessonPlanVisibility(ctx context.Context, id string, visibility string, groupID *string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET visibility = $1, group_id = $2, updated_at = $3 WHERE id = $4`,
		visibility, groupID, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教案可见范围失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// UpdateLessonPlanAIReview 更新教案AI评审结果
func UpdateLessonPlanAIReview(ctx context.Context, id string, score float64, result string, history string) error {
	now := time.Now()
	res, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET ai_review_score = $1, ai_review_result = $2, ai_review_history = $3, updated_at = $4
		WHERE id = $5
	`, score, result, history, now, id)
	if err != nil {
		return fmt.Errorf("更新AI评审结果失败: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// ForkLessonPlan 复制教案（fork）
//
// 注意：fork 出的新教案不继承 unit_plan_id（副本是新课，挂不挂单元方案由它自己决定）。
// ForkLessonPlan 走 CreateLessonPlan，而 CreateLessonPlan 不写 unit_plan_id，
// 故新副本 unit_plan_id 默认 NULL（=未挂载），符合预期。
func ForkLessonPlan(ctx context.Context, sourceID string, newAuthorID string) (*models.LessonPlan, error) {
	source, err := GetLessonPlanByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	newLP := &models.LessonPlan{
		Title:             source.Title + "（副本）",
		Subject:           source.Subject,
		Grade:             source.Grade,
		Topic:             source.Topic,
		DurationMinutes:   source.DurationMinutes,
		ContentMarkdown:   source.ContentMarkdown,
		ContentStructured: source.ContentStructured,
		GenerationConfig:  source.GenerationConfig,
		MatchedComponents: source.MatchedComponents,
		Status:            "draft",
		Visibility:        "personal",
		AuthorID:          newAuthorID,
		ForkedFrom:        &sourceID,
		TemplateID:        source.TemplateID,
		RecipeID:          source.RecipeID,
		TextbookPageIDs:   source.TextbookPageIDs, // 迭代7B：fork时保留课本关联
	}
	if err := CreateLessonPlan(ctx, newLP); err != nil {
		return nil, err
	}
	_, err = database.DB.Exec(ctx,
		`UPDATE lesson_plans SET fork_count = fork_count + 1 WHERE id = $1`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("更新fork计数失败: %w", err)
	}
	return newLP, nil
}

// IncrementLessonPlanView 增加教案浏览次数
func IncrementLessonPlanView(ctx context.Context, id string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

// DeleteLessonPlan 删除教案（物理删除，级联删除评审记录）
func DeleteLessonPlan(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `DELETE FROM lesson_plans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除教案失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// ==================== 教案评审CRUD ====================

// CreateLessonPlanReview 创建教案评审记录
func CreateLessonPlanReview(ctx context.Context, review *models.LessonPlanReview) error {
	query := `
		INSERT INTO lesson_plan_reviews (lesson_plan_id, reviewer_id, decision, score, dimensions, comments, suggestions, round)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	dimensions := review.Dimensions
	if dimensions == "" {
		dimensions = "{}"
	}
	suggestions := review.Suggestions
	if suggestions == "" {
		suggestions = "[]"
	}
	err := database.DB.QueryRow(ctx, query,
		review.LessonPlanID, review.ReviewerID, review.Decision,
		review.Score, dimensions, review.Comments, suggestions, review.Round,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建评审记录失败: %w", err)
	}
	return nil
}

// ListLessonPlanReviews 获取教案的评审记录列表
func ListLessonPlanReviews(ctx context.Context, lessonPlanID string) ([]*models.LessonPlanReviewItem, error) {
	query := `
		SELECT r.id, r.reviewer_id, COALESCE(u.display_name, '') AS reviewer_name,
		       r.decision, r.score, r.comments, r.round, r.created_at
		FROM lesson_plan_reviews r
		LEFT JOIN users u ON u.id = r.reviewer_id
		WHERE r.lesson_plan_id = $1
		ORDER BY r.round ASC, r.created_at ASC
	`
	rows, err := database.DB.Query(ctx, query, lessonPlanID)
	if err != nil {
		return nil, fmt.Errorf("查询评审记录失败: %w", err)
	}
	defer rows.Close()

	var items []*models.LessonPlanReviewItem
	for rows.Next() {
		item := &models.LessonPlanReviewItem{}
		err := rows.Scan(
			&item.ID, &item.ReviewerID, &item.ReviewerName,
			&item.Decision, &item.Score, &item.Comments, &item.Round, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描评审记录行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// ==================== v86新增：教案索引写入 ====================

// UpdateLessonPlanIndex 更新教案的AOCI索引（索引文本+冗余列）
func UpdateLessonPlanIndex(ctx context.Context, planID string, indexText string, cogLevel int, pedIntensity int, structType int, qualLevel int) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE lesson_plans
		SET lesson_index = $1,
		    idx_cognitive_level = $2,
		    idx_pedagogy_intensity = $3,
		    idx_structure_type = $4,
		    idx_quality_level = $5,
		    updated_at = $6
		WHERE id = $7
	`, indexText, cogLevel, pedIntensity, structType, qualLevel, now, planID)
	if err != nil {
		return fmt.Errorf("更新教案索引失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}

// ==================== 大单元备课：挂载/解除单元方案 ====================

// UpdateLessonPlanUnitPlanID 挂载或解除教案关联的单元方案
//
// unitPlanID 为 nil → 解除挂载（unit_plan_id 置 NULL）
// unitPlanID 非 nil → 挂载该单元方案（写入 unit_plan_id）
//
// 不做单元方案存在性/可见性校验（交 service 层把关），本函数只负责写列。
// 写列即生效——备课引擎每轮重读 lesson_plans，下一轮对话自动携带单元方案上下文。
//
// 设计说明：unit_plan_id 列无外键约束（镜像 teacher_assistant_prefs.assistant_id 的设计），
// 被挂载的单元方案若被删除，本列不连删、不报错；注入时由 service 层
// 校验单元方案 status==active 兜底（archived 的不注入）。
func UpdateLessonPlanUnitPlanID(ctx context.Context, id string, unitPlanID *string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET unit_plan_id = $1, updated_at = $2 WHERE id = $3`,
		unitPlanID, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教案单元方案关联失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLessonPlanNotFound
	}
	return nil
}
