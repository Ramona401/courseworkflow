package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 课件主表 CRUD ====================
// v0.42 变更：
//   - CreateCourseware: lesson_plan_id 改为可空，新增 source_type 写入
//   - GetCoursewareByID: 扩展到 20 列（新增 source_type/source_file_path/edu_module_id/published_version）
//   - ListCoursewares: 适配可空 lesson_plan_id，新增 source_type 读取
// 风格锚点轮1变更：
//   - GetCoursewareByID: 再扩展2列 style_anchor_asset_id/style_anchor_vaoci（共22列），供前端读当前锚点
// 阶段1（课件审核与协作·发布与共享）变更：
//   - GetCoursewareByID: 再扩展4列 publish_state/review_level/review_school_id/code_share_scope（共26列）
//   - ListCoursewares: SELECT 带上 publish_state/review_level/code_share_scope，供"我的课件"列表显示发布徽章
//   - 新增 UpdateCoursewarePublishState（写发布态+审核层级+学校ID）/ UpdateCoursewareCodeShareScope（写代码范围）
//   - 新增 ListSharedCoursewares（共享课件库查询，按作者白名单+学科筛选，关联作者名/学校名）
// 回收站迭代变更：
//   - DeleteCourseware: 由物理删除改为软删除（UPDATE SET deleted_at）
//   - GetCoursewareByID/ListCoursewares/ListSharedCoursewares: 查询加 deleted_at IS NULL 过滤

// CreateCourseware 创建课件记录
// v0.42: lesson_plan_id 改为可空，新增 source_type 参数
func CreateCourseware(ctx context.Context, cw *models.Courseware) error {
	sql := `INSERT INTO coursewares (id, lesson_plan_id, user_id, title, subject, grade, status, style_config, page_count, source_type, source_file_path)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, created_at, updated_at`
	// v0.42: lesson_plan_id 可空处理——指针为nil或空字符串值都传NULL
	var lpID interface{}
	if cw.LessonPlanID != nil && *cw.LessonPlanID != "" {
		lpID = *cw.LessonPlanID
	}
	sourceType := cw.SourceType
	if sourceType == "" {
		sourceType = models.CWSourceLessonPlan
	}
	return database.DB.QueryRow(ctx, sql,
		lpID, cw.UserID, cw.Title, cw.Subject, cw.Grade,
		cw.Status, nullIfEmpty(cw.StyleConfig), cw.PageCount,
		sourceType, nullIfEmpty(cw.SourceFilePath),
	).Scan(&cw.ID, &cw.CreatedAt, &cw.UpdatedAt)
}

// GetCoursewareByID 根据ID获取课件详情
// v0.42: 扩展到 20 列，含 source_type/source_file_path/edu_module_id/published_version
// 风格锚点轮1: 再加 style_anchor_asset_id/style_anchor_vaoci 两列（共22列）
// 阶段1: 再加 publish_state/review_level/review_school_id/code_share_scope 四列（共26列）
// 回收站迭代: WHERE 加 deleted_at IS NULL 排除已软删课件
func GetCoursewareByID(ctx context.Context, id string) (*models.Courseware, error) {
	sql := `SELECT id, lesson_plan_id, user_id, title, subject, grade, status,
COALESCE(style_config::text, ''), page_count, COALESCE(index_overview, ''),
COALESCE(logo_url, ''), COALESCE(org_name, ''), COALESCE(nav_template_html, ''),
pipeline_id, COALESCE(source_type, 'lesson_plan'), COALESCE(source_file_path, ''),
COALESCE(edu_module_id, ''), COALESCE(published_version, 0),
style_anchor_asset_id, COALESCE(style_anchor_vaoci, ''),
COALESCE(kp_codes::text, ''),
COALESCE(publish_state, 'private'), COALESCE(review_level, 0),
review_school_id, COALESCE(code_share_scope, 'none'),
COALESCE(collab_state, 'idle'),
created_at, updated_at
FROM coursewares WHERE id = $1 AND deleted_at IS NULL`
	cw := &models.Courseware{}
	err := database.DB.QueryRow(ctx, sql, id).Scan(
		&cw.ID, &cw.LessonPlanID, &cw.UserID, &cw.Title, &cw.Subject, &cw.Grade,
		&cw.Status, &cw.StyleConfig, &cw.PageCount, &cw.IndexOverview,
		&cw.LogoURL, &cw.OrgName, &cw.NavTemplateHTML,
		&cw.PipelineID, &cw.SourceType, &cw.SourceFilePath,
		&cw.EduModuleID, &cw.PublishedVersion,
		&cw.StyleAnchorAssetID, &cw.StyleAnchorVAOCI,
		&cw.KPCodes,
		&cw.PublishState, &cw.ReviewLevel,
		&cw.ReviewSchoolID, &cw.CodeShareScope,
		&cw.CollabState,
		&cw.CreatedAt, &cw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return cw, nil
}

// ListCoursewares 查询课件列表（"我的课件"——硬绑当前用户）
// v0.42: 适配可空 lesson_plan_id，新增 source_type 读取
// 阶段1: SELECT 带上 publish_state/review_level/code_share_scope，列表显示发布徽章
// 回收站迭代: 初始条件加 c.deleted_at IS NULL 排除已软删课件
func ListCoursewares(ctx context.Context, userID string, status string, subject string, limit int, offset int) ([]*models.CoursewareListItem, int, error) {
	conditions := []string{"c.user_id = $1", "c.deleted_at IS NULL"}
	args := []interface{}{userID}
	argIdx := 2

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if subject != "" {
		conditions = append(conditions, fmt.Sprintf("c.subject = $%d", argIdx))
		args = append(args, subject)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM coursewares c WHERE %s", whereClause)
	var total int
	if err := database.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询课件总数失败: %w", err)
	}

	// v0.42: LEFT JOIN lesson_plans（lesson_plan_id 可空时 lp.title 为 NULL → COALESCE 兜底）
	// 阶段1: 末尾增 publish_state/review_level/code_share_scope 三列
	listSQL := fmt.Sprintf(`SELECT c.id, c.lesson_plan_id, COALESCE(lp.title, ''), c.title, c.subject, c.grade,
c.status, c.page_count, c.pipeline_id, COALESCE(c.source_type, 'lesson_plan'),
COALESCE(c.publish_state, 'private'), COALESCE(c.review_level, 0), COALESCE(c.code_share_scope, 'none'),
c.created_at, c.updated_at
FROM coursewares c
LEFT JOIN lesson_plans lp ON lp.id = c.lesson_plan_id
WHERE %s
ORDER BY c.created_at DESC
LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课件列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CoursewareListItem
	for rows.Next() {
		item := &models.CoursewareListItem{}
		if err := rows.Scan(
			&item.ID, &item.LessonPlanID, &item.LessonPlanTitle, &item.Title,
			&item.Subject, &item.Grade, &item.Status, &item.PageCount,
			&item.PipelineID, &item.SourceType,
			&item.PublishState, &item.ReviewLevel, &item.CodeShareScope,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描课件列表行失败: %w", err)
		}
		item.StatusName = models.CoursewareStatusNameMap[item.Status]
		item.SourceName = models.CWSourceNameMap[item.SourceType]
		item.PublishStateName = models.CWPublishStateNameMap[item.PublishState]
		items = append(items, item)
	}
	return items, total, nil
}

// ListSharedCoursewares 查询共享课件库（阶段1新增）
//
// 设计（对齐教案库 ListLessonPlans 范式）：repo 只负责"按作者白名单 + 可选学科筛选"查询，
// "谁可见（同校/同组）"的归属解析在 service 层算好 visibleAuthorIDs 白名单再传进来，
// repo 不碰组织关系，职责清晰。
//
// 参数：
//
//	visibleAuthorIDs — 当前登录者可见的作者用户ID白名单（service 层按同校/同组解析）。
//	                   传 nil 表示"不限作者"（仅 admin 等全局可见者），传空切片表示"看不到任何"（fail-closed）。
//	subject          — 可选学科筛选（空串=不筛）
//	limit/offset     — 分页
//
// 只列 publish_state='published_shared' 的课件，按 updated_at 倒序。
// 关联 users 取作者显示名，关联 school_members→organizations 取作者学校名（取一条，无则空）。
// 回收站迭代: 初始条件加 c.deleted_at IS NULL 排除已软删课件
func ListSharedCoursewares(ctx context.Context, visibleAuthorIDs []string, subject string, limit int, offset int) ([]*models.SharedCoursewareListItem, int, error) {
	if limit <= 0 {
		limit = 20
	}

	conditions := []string{"c.publish_state = 'published_shared'", "c.deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	// 作者白名单三态：
	//   nil        → 不加作者条件（全局可见，仅 admin）
	//   非nil空切片 → 注入恒假条件（看不到任何，fail-closed）
	//   非空        → c.user_id = ANY($n)
	if visibleAuthorIDs != nil {
		if len(visibleAuthorIDs) == 0 {
			conditions = append(conditions, "1 = 0")
		} else {
			conditions = append(conditions, fmt.Sprintf("c.user_id = ANY($%d)", argIdx))
			args = append(args, visibleAuthorIDs)
			argIdx++
		}
	}

	if subject != "" {
		conditions = append(conditions, fmt.Sprintf("c.subject = $%d", argIdx))
		args = append(args, subject)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM coursewares c WHERE %s", whereClause)
	var total int
	if err := database.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询共享课件总数失败: %w", err)
	}

	// 作者学校名用 LATERAL 子查询取一条（school_members→organizations，type=school 且 active）
	listSQL := fmt.Sprintf(`SELECT c.id, c.title, c.subject, c.grade, c.page_count,
COALESCE(c.source_type, 'lesson_plan'),
c.user_id, COALESCE(u.display_name, u.username, ''),
COALESCE(sch.name, ''),
COALESCE(c.publish_state, 'private'), COALESCE(c.code_share_scope, 'none'),
c.created_at, c.updated_at
FROM coursewares c
LEFT JOIN users u ON u.id = c.user_id
LEFT JOIN LATERAL (
    SELECT o.name
    FROM school_members sm
    JOIN organizations o ON o.id = sm.school_id AND o.status = 'active'
    WHERE sm.user_id = c.user_id
    LIMIT 1
) sch ON true
WHERE %s
ORDER BY c.updated_at DESC
LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询共享课件列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.SharedCoursewareListItem
	for rows.Next() {
		item := &models.SharedCoursewareListItem{}
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Subject, &item.Grade, &item.PageCount,
			&item.SourceType,
			&item.AuthorID, &item.AuthorName,
			&item.SchoolName,
			&item.PublishState, &item.CodeShareScope,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描共享课件行失败: %w", err)
		}
		item.SourceName = models.CWSourceNameMap[item.SourceType]
		item.PublishStateName = models.CWPublishStateNameMap[item.PublishState]
		// CanCopy 由 service 层按 code_share_scope + 归属关系裁决，repo 不算
		items = append(items, item)
	}
	return items, total, nil
}

// UpdateCoursewareStatus 更新课件状态
func UpdateCoursewareStatus(ctx context.Context, id string, status string) error {
	sql := `UPDATE coursewares SET status = $1, updated_at = $2 WHERE id = $3`
	tag, err := database.DB.Exec(ctx, sql, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新课件状态失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在: %s", id)
	}
	return nil
}

// UpdateCoursewarePublishState 更新课件发布态 + 审核层级 + 审核学校ID（阶段1新增）
//
// 三者一起更新，因为它们在发布/审核流转中经常同步变化（如提交审核时
// publish_state→submitted、review_school_id→作者学校；审核通过时 review_level→1/2）。
//
// 参数：
//
//	publishState   — 目标发布态（service 层已校验合法性）
//	reviewLevel    — 审核层级进度（0/1/2）
//	reviewSchoolID — 审核学校ID指针，nil=写 NULL（未提交/已撤回时清空）
func UpdateCoursewarePublishState(ctx context.Context, id string, publishState string, reviewLevel int, reviewSchoolID *string) error {
	var schoolArg interface{}
	if reviewSchoolID != nil && *reviewSchoolID != "" {
		schoolArg = *reviewSchoolID
	}
	sql := `UPDATE coursewares SET publish_state = $1, review_level = $2, review_school_id = $3, updated_at = $4 WHERE id = $5`
	tag, err := database.DB.Exec(ctx, sql, publishState, reviewLevel, schoolArg, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新课件发布态失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在: %s", id)
	}
	return nil
}

// UpdateCoursewareCodeShareScope 更新课件源代码开放范围（阶段1新增，产权分级）
func UpdateCoursewareCodeShareScope(ctx context.Context, id string, codeShareScope string) error {
	sql := `UPDATE coursewares SET code_share_scope = $1, updated_at = $2 WHERE id = $3`
	tag, err := database.DB.Exec(ctx, sql, codeShareScope, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新课件代码开放范围失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在: %s", id)
	}
	return nil
}

// UpdateCoursewareCollabState 更新课件集体备课态（阶段4新增）
// state 必须是 idle / in_session（service 层已用 models.IsValidCollabState 校验）。
// 与 status/publish_state 正交，仅写 collab_state 一列，不触碰其他维度。
func UpdateCoursewareCollabState(ctx context.Context, id string, state string) error {
	sql := `UPDATE coursewares SET collab_state = $1, updated_at = $2 WHERE id = $3`
	tag, err := database.DB.Exec(ctx, sql, state, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新课件集体备课态失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在: %s", id)
	}
	return nil
}

// UpdateCoursewareTitle 更新课件标题
func UpdateCoursewareTitle(ctx context.Context, id string, title string) error {
	sql := `UPDATE coursewares SET title = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, title, time.Now(), id)
	return err
}

// UpdateCoursewareStyle 保存风格配置（JSONB）
func UpdateCoursewareStyle(ctx context.Context, id string, styleConfig string) error {
	sql := `UPDATE coursewares SET style_config = $1::jsonb, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, styleConfig, time.Now(), id)
	return err
}

// UpdateCoursewarePageCount 更新课件页数
func UpdateCoursewarePageCount(ctx context.Context, id string, count int) error {
	sql := `UPDATE coursewares SET page_count = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, count, time.Now(), id)
	return err
}

// UpdateCoursewareOverview 更新课件脉络概述
func UpdateCoursewareOverview(ctx context.Context, id string, overview string) error {
	sql := `UPDATE coursewares SET index_overview = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, overview, time.Now(), id)
	return err
}

// UpdateCoursewareLogo 更新课件Logo URL
func UpdateCoursewareLogo(ctx context.Context, id string, logoURL string) error {
	sql := `UPDATE coursewares SET logo_url = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, logoURL, time.Now(), id)
	return err
}

// UpdateCoursewareOrgName 更新课件机构名称
func UpdateCoursewareOrgName(ctx context.Context, id string, orgName string) error {
	sql := `UPDATE coursewares SET org_name = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, orgName, time.Now(), id)
	return err
}

// UpdateCoursewareNavTemplate 保存用户确认的导航栏HTML模板
func UpdateCoursewareNavTemplate(ctx context.Context, id string, navHTML string) error {
	sql := `UPDATE coursewares SET nav_template_html = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, navHTML, time.Now(), id)
	return err
}

// UpdateCoursewarePipelineID 回填Pipeline ID
func UpdateCoursewarePipelineID(ctx context.Context, id string, pipelineID string) error {
	sql := `UPDATE coursewares SET pipeline_id = $1, status = $2, updated_at = $3 WHERE id = $4`
	_, err := database.DB.Exec(ctx, sql, pipelineID, models.CoursewareStatusInPipeline, time.Now(), id)
	return err
}

// DeleteCourseware 删除课件（软删除：设置 deleted_at 时间戳，不物理删除）
// 软删除后课件在列表/详情/共享库中均不可见，但数据保留在数据库中可通过回收站恢复。
// 30天后由定时任务 PurgeExpiredTrash 自动物理清理。
// 状态校验已上移到 service 层；此处仅按 id 做软删除。
func DeleteCourseware(ctx context.Context, id string) error {
	sql := `UPDATE coursewares SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := database.DB.Exec(ctx, sql, id)
	if err != nil {
		return fmt.Errorf("删除课件失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在或已被删除")
	}
	return nil
}

// ListUserLogoURLs 查询某用户历史上传过的 Logo URL（去重，按最近使用倒序）
// 需求2：风格页"历史 Logo 复用"——从该用户名下所有课件提取非空 logo_url，
// 同一 URL 只保留一条，按最近一次使用时间 MAX(updated_at) 倒序，最多 limit 条
func ListUserLogoURLs(ctx context.Context, userID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	sql := `SELECT logo_url FROM coursewares
                WHERE user_id = $1 AND COALESCE(logo_url, '') <> ''
                GROUP BY logo_url
                ORDER BY MAX(updated_at) DESC
                LIMIT $2`
	rows, err := database.DB.Query(ctx, sql, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("查询历史Logo失败: %w", err)
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, fmt.Errorf("扫描历史Logo行失败: %w", err)
		}
		urls = append(urls, u)
	}
	return urls, nil
}

// DeleteUserLogoURL 清空某用户名下所有使用指定 logo_url 的课件的 Logo
// 需求2：用于历史 Logo 删除——把 logo_url 置空，使其不再出现在 ListUserLogoURLs 结果中
// 返回受影响行数（即有多少个课件用过这个 Logo）
func DeleteUserLogoURL(ctx context.Context, userID string, logoURL string) (int64, error) {
	sql := `UPDATE coursewares SET logo_url = '',
                style_config = CASE WHEN style_config IS NULL THEN style_config ELSE style_config - 'logo_url' END,
                updated_at = $1
                WHERE user_id = $2 AND logo_url = $3`
	tag, err := database.DB.Exec(ctx, sql, time.Now(), userID, logoURL)
	if err != nil {
		return 0, fmt.Errorf("删除历史Logo失败: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ==================== 课件页面 CRUD ====================

// cwPageSelectColumns 课件页面查询列（19列）
const cwPageSelectColumns = `id, courseware_id, page_number,
COALESCE(title,''), COALESCE(purpose,''), COALESCE(content_summary,''),
COALESCE(interaction_type,''), COALESCE(visual_format,''), COALESCE(media_requirements,''),
estimated_complexity,
COALESCE(page_index,''), idx_cognitive_level, idx_interaction_level, COALESCE(idx_visual_format,''),
COALESCE(html_content,''), COALESCE(placeholder_map::text,''), COALESCE(matched_component_ids::text,''),
status, created_at, updated_at`

// scanCWPage 统一扫描课件页面行（19列）
func scanCWPage(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewarePage, error) {
	p := &models.CoursewarePage{}
	err := scanner.Scan(
		&p.ID, &p.CoursewareID, &p.PageNumber,
		&p.Title, &p.Purpose, &p.ContentSummary,
		&p.InteractionType, &p.VisualFormat, &p.MediaRequirements,
		&p.EstimatedComplexity,
		&p.PageIndex, &p.IdxCognitiveLevel, &p.IdxInteractionLevel, &p.IdxVisualFormat,
		&p.HTMLContent, &p.PlaceholderMap, &p.MatchedComponentIDs,
		&p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// CreateCoursewarePage 创建课件页面
func CreateCoursewarePage(ctx context.Context, page *models.CoursewarePage) error {
	sql := `INSERT INTO courseware_pages (id, courseware_id, page_number, title, purpose,
content_summary, interaction_type, visual_format, media_requirements,
estimated_complexity, page_index, idx_cognitive_level, idx_interaction_level, idx_visual_format,
html_content, placeholder_map, matched_component_ids, status)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16::jsonb, $17)
RETURNING id, created_at, updated_at`
	return database.DB.QueryRow(ctx, sql,
		page.CoursewareID, page.PageNumber, page.Title, page.Purpose,
		page.ContentSummary, page.InteractionType, page.VisualFormat,
		page.MediaRequirements, page.EstimatedComplexity,
		page.PageIndex, page.IdxCognitiveLevel, page.IdxInteractionLevel, page.IdxVisualFormat,
		page.HTMLContent, nullIfEmpty(page.PlaceholderMap), nullIfEmpty(page.MatchedComponentIDs), page.Status,
	).Scan(&page.ID, &page.CreatedAt, &page.UpdatedAt)
}

// BatchCreateCoursewarePages 批量创建课件页面
func BatchCreateCoursewarePages(ctx context.Context, pages []*models.CoursewarePage) error {
	if len(pages) == 0 {
		return nil
	}
	for _, page := range pages {
		if err := CreateCoursewarePage(ctx, page); err != nil {
			return fmt.Errorf("批量创建课件页面失败(page_number=%d): %w", page.PageNumber, err)
		}
	}
	return nil
}

// ListCoursewarePages 获取课件的所有页面
func ListCoursewarePages(ctx context.Context, coursewareID string) ([]*models.CoursewarePage, error) {
	sql := fmt.Sprintf(`SELECT %s FROM courseware_pages WHERE courseware_id = $1 ORDER BY page_number ASC`, cwPageSelectColumns)
	rows, err := database.DB.Query(ctx, sql, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("查询课件页面列表失败: %w", err)
	}
	defer rows.Close()

	var pages []*models.CoursewarePage
	for rows.Next() {
		p, err := scanCWPage(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描课件页面行失败: %w", err)
		}
		pages = append(pages, p)
	}
	return pages, nil
}

// GetCoursewarePageByNumber 获取课件指定页码的页面
func GetCoursewarePageByNumber(ctx context.Context, coursewareID string, pageNumber int) (*models.CoursewarePage, error) {
	sql := fmt.Sprintf(`SELECT %s FROM courseware_pages WHERE courseware_id = $1 AND page_number = $2`, cwPageSelectColumns)
	p, err := scanCWPage(database.DB.QueryRow(ctx, sql, coursewareID, pageNumber))
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateCWPageIndex 更新单页索引说明
func UpdateCWPageIndex(ctx context.Context, coursewareID string, pageNumber int, req *models.UpdateCWPageIndexRequest) error {
	sql := `UPDATE courseware_pages SET title = $1, purpose = $2, content_summary = $3,
interaction_type = $4, visual_format = $5, media_requirements = $6,
estimated_complexity = $7, updated_at = $8
WHERE courseware_id = $9 AND page_number = $10`
	tag, err := database.DB.Exec(ctx, sql,
		req.Title, req.Purpose, req.ContentSummary, req.InteractionType,
		req.VisualFormat, req.MediaRequirements, req.EstimatedComplexity,
		time.Now(), coursewareID, pageNumber,
	)
	if err != nil {
		return fmt.Errorf("更新课件页面索引失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件页面不存在: courseware=%s page=%d", coursewareID, pageNumber)
	}
	return nil
}

// UpdateCWPageIndexFields 仅回填单页的 AOCI 索引列（page_index + 三个 idx 冗余列）
func UpdateCWPageIndexFields(ctx context.Context, coursewareID string, pageNumber int, pageIndex string, cg int, il int, vf string) error {
	sql := `UPDATE courseware_pages SET page_index = $1,
idx_cognitive_level = $2, idx_interaction_level = $3, idx_visual_format = $4,
updated_at = $5
WHERE courseware_id = $6 AND page_number = $7`
	_, err := database.DB.Exec(ctx, sql,
		pageIndex, cg, il, vf, time.Now(), coursewareID, pageNumber,
	)
	if err != nil {
		return fmt.Errorf("回填课件页面索引列失败(courseware=%s page=%d): %w", coursewareID, pageNumber, err)
	}
	return nil
}

// UpdateCWPageHTML 更新页面生成的HTML代码
func UpdateCWPageHTML(ctx context.Context, pageID string, htmlContent string, placeholderMap string, matchedIDs string, status string) error {
	sql := `UPDATE courseware_pages SET html_content = $1, placeholder_map = $2::jsonb,
matched_component_ids = $3::jsonb, status = $4, updated_at = $5
WHERE id = $6`
	_, err := database.DB.Exec(ctx, sql, htmlContent, nullIfEmpty(placeholderMap), nullIfEmpty(matchedIDs), status, time.Now(), pageID)
	return err
}

// UpdateCWPageStatus 更新页面状态
func UpdateCWPageStatus(ctx context.Context, pageID string, status string) error {
	sql := `UPDATE courseware_pages SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, status, time.Now(), pageID)
	return err
}

// DeleteCoursewarePage 删除课件页面
func DeleteCoursewarePage(ctx context.Context, coursewareID string, pageNumber int) error {
	sql := `DELETE FROM courseware_pages WHERE courseware_id = $1 AND page_number = $2`
	_, err := database.DB.Exec(ctx, sql, coursewareID, pageNumber)
	return err
}

// DeleteAllCoursewarePages 删除课件的全部页面
func DeleteAllCoursewarePages(ctx context.Context, coursewareID string) error {
	sql := `DELETE FROM courseware_pages WHERE courseware_id = $1`
	_, err := database.DB.Exec(ctx, sql, coursewareID)
	return err
}

// ReorderCoursewarePages 重新排序课件页面
func ReorderCoursewarePages(ctx context.Context, coursewareID string, pageIDs []string) error {
	// 修正：原实现逐页裸 UPDATE page_number，会撞 UNIQUE(courseware_id,page_number) 约束
	// （把某页改成 N 时表里可能仍有另一页是 N）。改为复用两阶段避撞的事务重排。
	return ResequenceCoursewarePagesByIDs(ctx, coursewareID, pageIDs)
}

// CountCoursewarePages 统计课件页面数
func CountCoursewarePages(ctx context.Context, coursewareID string) (int, error) {
	var count int
	sql := `SELECT COUNT(*) FROM courseware_pages WHERE courseware_id = $1`
	err := database.DB.QueryRow(ctx, sql, coursewareID).Scan(&count)
	return count, err
}

// ==================== 风格锚点字段更新 ====================

// UpdateCoursewareStyleAnchor 设置课件风格锚点（写入资产ID + VAOCI索引文本）
func UpdateCoursewareStyleAnchor(ctx context.Context, id string, assetID string, vaoci string) error {
	sql := `UPDATE coursewares SET style_anchor_asset_id = $1, style_anchor_vaoci = $2, updated_at = $3 WHERE id = $4`
	tag, err := database.DB.Exec(ctx, sql, assetID, vaoci, time.Now(), id)
	if err != nil {
		return fmt.Errorf("设置风格锚点失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在: %s", id)
	}
	return nil
}

// ClearCoursewareStyleAnchor 清除课件风格锚点（两字段置NULL）
func ClearCoursewareStyleAnchor(ctx context.Context, id string) error {
	sql := `UPDATE coursewares SET style_anchor_asset_id = NULL, style_anchor_vaoci = NULL, updated_at = $1 WHERE id = $2`
	_, err := database.DB.Exec(ctx, sql, time.Now(), id)
	return err
}

// ==================== 课程知识库轮：kp_codes 写入 ====================

// UpdateCoursewareKPCodes 写入课件勾选的课标知识点编码数组（JSON文本）
func UpdateCoursewareKPCodes(ctx context.Context, id string, kpCodesJSON string) error {
	sql := `UPDATE coursewares SET kp_codes = $1::jsonb, updated_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(ctx, sql, nullIfEmpty(kpCodesJSON), time.Now(), id)
	if err != nil {
		return fmt.Errorf("写入课件知识点编码失败: %w", err)
	}
	return nil
}

// ==================== 辅助函数 ====================

// nullIfEmpty JSONB字段空值处理——空字符串转NULL避免PostgreSQL JSONB解析报错
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
