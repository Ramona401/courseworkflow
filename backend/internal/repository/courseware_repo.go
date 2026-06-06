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
func GetCoursewareByID(ctx context.Context, id string) (*models.Courseware, error) {
	sql := `SELECT id, lesson_plan_id, user_id, title, subject, grade, status,
COALESCE(style_config::text, ''), page_count, COALESCE(index_overview, ''),
COALESCE(logo_url, ''), COALESCE(org_name, ''), COALESCE(nav_template_html, ''),
pipeline_id, COALESCE(source_type, 'lesson_plan'), COALESCE(source_file_path, ''),
COALESCE(edu_module_id, ''), COALESCE(published_version, 0),
style_anchor_asset_id, COALESCE(style_anchor_vaoci, ''),
COALESCE(kp_codes::text, ''),
created_at, updated_at
FROM coursewares WHERE id = $1`
	cw := &models.Courseware{}
	err := database.DB.QueryRow(ctx, sql, id).Scan(
		&cw.ID, &cw.LessonPlanID, &cw.UserID, &cw.Title, &cw.Subject, &cw.Grade,
		&cw.Status, &cw.StyleConfig, &cw.PageCount, &cw.IndexOverview,
		&cw.LogoURL, &cw.OrgName, &cw.NavTemplateHTML,
		&cw.PipelineID, &cw.SourceType, &cw.SourceFilePath,
		&cw.EduModuleID, &cw.PublishedVersion,
		&cw.StyleAnchorAssetID, &cw.StyleAnchorVAOCI,
		&cw.KPCodes,
		&cw.CreatedAt, &cw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return cw, nil
}

// ListCoursewares 查询课件列表
// v0.42: 适配可空 lesson_plan_id，新增 source_type 读取
func ListCoursewares(ctx context.Context, userID string, status string, subject string, limit int, offset int) ([]*models.CoursewareListItem, int, error) {
	conditions := []string{"c.user_id = $1"}
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
	listSQL := fmt.Sprintf(`SELECT c.id, c.lesson_plan_id, COALESCE(lp.title, ''), c.title, c.subject, c.grade,
c.status, c.page_count, c.pipeline_id, COALESCE(c.source_type, 'lesson_plan'),
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
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描课件列表行失败: %w", err)
		}
		item.StatusName = models.CoursewareStatusNameMap[item.Status]
		item.SourceName = models.CWSourceNameMap[item.SourceType]
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

// DeleteCourseware 删除课件（状态校验已上移到 service 层；此处仅按 id 删除，关联子表由 DB 外键 CASCADE 自动清理）
func DeleteCourseware(ctx context.Context, id string) error {
	sql := `DELETE FROM coursewares WHERE id = $1`
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
//
// v0.44 新增：用于「后台异步补索引」与「夜间索引轮询」。
// 严格只更新索引相关4列，绝不触碰 title/purpose/content_summary 等用户方案字段，
// 也不动 html_content / status，避免误伤已生成的方案与课件内容。
//
// 参数：
//
//	pageIndex —— 层1风格的 AOCI 压缩索引原文（PAGE:..|KT:..|CG:..|IL:..|VF:.. + [K][A][I][R][C] 语义行）
//	cg        —— 认知层次 1-6（idx_cognitive_level）
//	il        —— 交互复杂度 1-5（idx_interaction_level）
//	vf        —— 视觉形式编码（idx_visual_format，如 TH/IT/DG...）
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
	// 注意：不校验 RowsAffected==0 —— 后台/夜间回填时该页可能已被删除（老师改了方案），
	// 此时静默跳过即可，不应视为错误中断整批回填。
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
	for i, pid := range pageIDs {
		sql := `UPDATE courseware_pages SET page_number = $1, updated_at = $2
WHERE id = $3 AND courseware_id = $4`
		_, err := database.DB.Exec(ctx, sql, i+1, time.Now(), pid, coursewareID)
		if err != nil {
			return fmt.Errorf("排序课件页面失败(id=%s): %w", pid, err)
		}
	}
	return nil
}

// CountCoursewarePages 统计课件页面数
func CountCoursewarePages(ctx context.Context, coursewareID string) (int, error) {
	var count int
	sql := `SELECT COUNT(*) FROM courseware_pages WHERE courseware_id = $1`
	err := database.DB.QueryRow(ctx, sql, coursewareID).Scan(&count)
	return count, err
}

// ==================== 风格锚点字段更新（VAOCI 课程级风格一致性，轮1新增）====================
// 说明：设/查/清锚点的完整业务接口在轮2实现，本轮先提供底层DB写入函数，
//       供轮2的 service 层调用。查锚点直接复用 GetCoursewareByID（已含两列）。

// UpdateCoursewareStyleAnchor 设置课件风格锚点（写入资产ID + VAOCI索引文本）
// assetID 指向 courseware_assets.id；vaoci 为多模态AI读图提取的VAOCI索引文本
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
// kpCodesJSON 为 JSON 数组文本（如 `["MATH-G3-NA-001","MATH-G3-GG-002"]`）；
// 空串时写 NULL（nullIfEmpty 处理），语义=未勾选。供"从主题创建"勾选后持久化。
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
