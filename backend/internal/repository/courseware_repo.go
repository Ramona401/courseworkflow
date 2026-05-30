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
func GetCoursewareByID(ctx context.Context, id string) (*models.Courseware, error) {
sql := `SELECT id, lesson_plan_id, user_id, title, subject, grade, status,
COALESCE(style_config::text, ''), page_count, COALESCE(index_overview, ''),
COALESCE(logo_url, ''), COALESCE(org_name, ''), COALESCE(nav_template_html, ''),
pipeline_id, COALESCE(source_type, 'lesson_plan'), COALESCE(source_file_path, ''),
COALESCE(edu_module_id, ''), COALESCE(published_version, 0),
created_at, updated_at
FROM coursewares WHERE id = $1`
cw := &models.Courseware{}
err := database.DB.QueryRow(ctx, sql, id).Scan(
&cw.ID, &cw.LessonPlanID, &cw.UserID, &cw.Title, &cw.Subject, &cw.Grade,
&cw.Status, &cw.StyleConfig, &cw.PageCount, &cw.IndexOverview,
&cw.LogoURL, &cw.OrgName, &cw.NavTemplateHTML,
&cw.PipelineID, &cw.SourceType, &cw.SourceFilePath,
&cw.EduModuleID, &cw.PublishedVersion,
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

// DeleteCourseware 删除课件（仅draft状态允许）
func DeleteCourseware(ctx context.Context, id string) error {
sql := `DELETE FROM coursewares WHERE id = $1 AND status = 'draft'`
tag, err := database.DB.Exec(ctx, sql, id)
if err != nil {
return fmt.Errorf("删除课件失败: %w", err)
}
if tag.RowsAffected() == 0 {
return fmt.Errorf("课件不存在或状态不允许删除")
}
return nil
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
func scanCWPage(scanner interface{ Scan(dest ...interface{}) error }) (*models.CoursewarePage, error) {
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

// ==================== 辅助函数 ====================

// nullIfEmpty JSONB字段空值处理——空字符串转NULL避免PostgreSQL JSONB解析报错
func nullIfEmpty(s string) interface{} {
if s == "" {
return nil
}
return s
}
