package repository

// courseware_template_repo.go — 课件风格模板数据访问层(v139 新建)
//
// 从 courseware_component_repo.go 拆出,职责分离。
//
// 功能覆盖:
//   1. 模板 CRUD(向后兼容原签名)
//      - ListCWTemplates / GetCWTemplateByID
//      - CreateCWTemplate / UpdateCWTemplate / DeleteCWTemplate
//      - ListCWTemplatesWithUser / CreatePersonalTemplate / DeletePersonalTemplate
//
//   2. v139 新增:4 级 scope 联合查询
//      - ListCWTemplatesWithParams 支持 system/school/group/personal 并集查询 + 草稿过滤
//
//   3. v139 新增:AI 提取草稿
//      - CreateDraftTemplate AI 分析 HTML 后入库为草稿(is_draft=true)
//      - ListMyDrafts 查询我的草稿列表
//
//   4. v139 新增:AI 微调
//      - UpdateTemplateRefined 微调更新,自动写入 refine_history 快照(裁剪到 20 条)
//      - GetTemplateForRefine 加载模板供微调使用
//
//   5. v139 新增:历史回退
//      - GetRefineHistory 读取微调历史数组
//      - RollbackToHistory 把第 N 条历史快照恢复为当前内容
//
//   6. v139 新增:草稿发布为正式
//      - PublishDraft 草稿转正式(设置 scope/scope_target_id/is_draft=false,清空 refine_history 留新空间)

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 模板查询统一 scan(v139 扩 20 列) ====================

// cwTemplateScanColumns v139 模板查询统一 20 列
// 新增 4 列: scope_target_id / is_draft / refine_history / extract_source_meta
const cwTemplateScanColumns = `id, name, COALESCE(description,''), style_category,
	COALESCE(preview_image_url,''), COALESCE(color_scheme::text,''),
	COALESCE(css_variables::text,''), COALESCE(sample_pages::text,''),
	COALESCE(preview_urls::text,''),
	is_active, sort_order,
	user_id, COALESCE(scope,'system'), source_courseware_id,
	scope_target_id, is_draft,
	COALESCE(refine_history::text,''), COALESCE(extract_source_meta::text,''),
	created_at, updated_at`

// scanCWTemplate v139 统一扫描 20 列到模板结构体
func scanCWTemplate(scan func(dest ...interface{}) error) (*models.CoursewareTemplate, error) {
	t := &models.CoursewareTemplate{}
	err := scan(
		&t.ID, &t.Name, &t.Description, &t.StyleCategory,
		&t.PreviewImageURL, &t.ColorScheme, &t.CSSVariables, &t.SamplePages,
		&t.PreviewURLs,
		&t.IsActive, &t.SortOrder,
		&t.UserID, &t.Scope, &t.SourceCoursewareID,
		&t.ScopeTargetID, &t.IsDraft,
		&t.RefineHistory, &t.ExtractSourceMeta,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

// ==================== 模板 CRUD(向后兼容) ====================

// ListCWTemplates 获取所有激活的系统风格模板(按排序) — 向后兼容原有调用
// 注意:默认过滤掉草稿(is_draft=false)
func ListCWTemplates(ctx context.Context, activeOnly bool) ([]*models.CoursewareTemplate, error) {
	conditions := "scope = 'system' AND is_draft = false"
	if activeOnly {
		conditions += " AND is_active = true"
	}
	sql := fmt.Sprintf("SELECT %s FROM courseware_templates WHERE %s ORDER BY sort_order ASC, created_at ASC",
		cwTemplateScanColumns, conditions)

	rows, err := database.DB.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询风格模板列表失败: %w", err)
	}
	defer rows.Close()

	var templates []*models.CoursewareTemplate
	for rows.Next() {
		t, err := scanCWTemplate(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("扫描风格模板行失败: %w", err)
		}
		templates = append(templates, t)
	}
	return templates, nil
}

// ListCWTemplatesWithUser v137 + v139:获取系统模板 + 指定用户的个人模板
// v139:默认过滤草稿;若需要草稿请用 ListMyDrafts 或 ListCWTemplatesWithParams
func ListCWTemplatesWithUser(ctx context.Context, userID string, activeOnly bool) ([]*models.CoursewareTemplate, error) {
	conditions := "is_draft = false AND (scope = 'system'"
	args := []interface{}{}
	argIdx := 1

	if userID != "" {
		conditions += fmt.Sprintf(" OR (scope = 'personal' AND user_id = $%d)", argIdx)
		args = append(args, userID)
		argIdx++
	}
	conditions += ")"

	if activeOnly {
		conditions += " AND is_active = true"
	}

	sql := fmt.Sprintf("SELECT %s FROM courseware_templates WHERE %s ORDER BY scope ASC, sort_order ASC, created_at ASC",
		cwTemplateScanColumns, conditions)

	rows, err := database.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("查询风格模板列表失败: %w", err)
	}
	defer rows.Close()

	var templates []*models.CoursewareTemplate
	for rows.Next() {
		t, err := scanCWTemplate(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("扫描风格模板行失败: %w", err)
		}
		templates = append(templates, t)
	}
	return templates, nil
}

// GetCWTemplateByID 获取风格模板详情
func GetCWTemplateByID(ctx context.Context, id string) (*models.CoursewareTemplate, error) {
	sql := fmt.Sprintf("SELECT %s FROM courseware_templates WHERE id = $1", cwTemplateScanColumns)
	t, err := scanCWTemplate(database.DB.QueryRow(ctx, sql, id).Scan)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// CreateCWTemplate 创建系统模板(admin 使用)
func CreateCWTemplate(ctx context.Context, t *models.CoursewareTemplate) error {
	sql := `INSERT INTO courseware_templates (id, name, description, style_category,
		preview_image_url, color_scheme, css_variables, sample_pages, preview_urls,
		is_active, sort_order, scope, is_draft)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8::jsonb, $9, $10, 'system', false)
		RETURNING id, created_at, updated_at`
	return database.DB.QueryRow(ctx, sql,
		t.Name, t.Description, t.StyleCategory,
		t.PreviewImageURL, nullIfEmpty(t.ColorScheme), nullIfEmpty(t.CSSVariables),
		nullIfEmpty(t.SamplePages), nullIfEmpty(t.PreviewURLs),
		t.IsActive, t.SortOrder,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// CreatePersonalTemplate v137:创建个人模板(老师从课件保存)
func CreatePersonalTemplate(ctx context.Context, t *models.CoursewareTemplate) error {
	sql := `INSERT INTO courseware_templates (id, name, description, style_category,
		preview_image_url, color_scheme, css_variables, sample_pages, preview_urls,
		is_active, sort_order, user_id, scope, source_courseware_id, is_draft)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8::jsonb, $9, $10, $11, 'personal', $12, false)
		RETURNING id, created_at, updated_at`
	return database.DB.QueryRow(ctx, sql,
		t.Name, t.Description, t.StyleCategory,
		t.PreviewImageURL, nullIfEmpty(t.ColorScheme), nullIfEmpty(t.CSSVariables),
		nullIfEmpty(t.SamplePages), nullIfEmpty(t.PreviewURLs),
		t.IsActive, t.SortOrder, t.UserID, t.SourceCoursewareID,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// DeletePersonalTemplate v137:删除个人模板(仅限创建者本人)
func DeletePersonalTemplate(ctx context.Context, id string, userID string) error {
	sql := `DELETE FROM courseware_templates WHERE id = $1 AND user_id = $2 AND scope = 'personal'`
	tag, err := database.DB.Exec(ctx, sql, id, userID)
	if err != nil {
		return fmt.Errorf("删除个人模板失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("模板不存在或无权删除")
	}
	return nil
}

// UpdateCWTemplate 更新风格模板基本信息(name/description/style_category/配色/CSS变量/样例页)
func UpdateCWTemplate(ctx context.Context, id string, req *models.UpdateCWTemplateRequest) error {
	sql := `UPDATE courseware_templates SET name = $1, description = $2, style_category = $3,
		preview_image_url = $4, color_scheme = $5::jsonb, css_variables = $6::jsonb,
		sample_pages = $7::jsonb, preview_urls = $8::jsonb, updated_at = $9
		WHERE id = $10`
	tag, err := database.DB.Exec(ctx, sql,
		req.Name, req.Description, req.StyleCategory,
		req.PreviewImageURL, nullIfEmpty(req.ColorScheme), nullIfEmpty(req.CSSVariables),
		nullIfEmpty(req.SamplePages), nullIfEmpty(req.PreviewURLs),
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("更新风格模板失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("风格模板不存在: %s", id)
	}
	if req.IsActive != nil {
		_, _ = database.DB.Exec(ctx, `UPDATE courseware_templates SET is_active = $1 WHERE id = $2`, *req.IsActive, id)
	}
	if req.SortOrder != nil {
		_, _ = database.DB.Exec(ctx, `UPDATE courseware_templates SET sort_order = $1 WHERE id = $2`, *req.SortOrder, id)
	}
	return nil
}

// DeleteCWTemplate 删除风格模板(物理删除,admin 用)
func DeleteCWTemplate(ctx context.Context, id string) error {
	sql := `DELETE FROM courseware_templates WHERE id = $1`
	tag, err := database.DB.Exec(ctx, sql, id)
	if err != nil {
		return fmt.Errorf("删除风格模板失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("风格模板不存在: %s", id)
	}
	return nil
}

// ==================== v139 新增:4 级 scope 联合查询 ====================

// ListCWTemplatesWithParams v139:支持 system/school/group/personal 4 级 scope 联合查询
//
// 查询逻辑:
//   - 总是包含 scope=system(所有人可见)
//   - 若 SchoolID 非空,包含 scope=school AND scope_target_id=SchoolID
//   - 若 GroupIDs 非空,包含 scope=group AND scope_target_id IN GroupIDs
//   - 若 UserID 非空,包含 scope=personal AND user_id=UserID
//   - 草稿过滤:OnlyMyDrafts=true 只看自己的草稿;否则按 IncludeDrafts 控制
//
// 排序:scope ASC(system/school/group/personal 依次), sort_order ASC, created_at ASC
func ListCWTemplatesWithParams(ctx context.Context, p models.ListCWTemplatesParams) ([]*models.CoursewareTemplate, error) {
	// ---- 构建 scope OR 子句 ----
	var orClauses []string
	args := []interface{}{}
	argIdx := 1

	// 仅查我的草稿:跳过其他 scope 拼接
	if p.OnlyMyDrafts {
		if p.UserID == "" {
			return []*models.CoursewareTemplate{}, nil
		}
		whereSQL := fmt.Sprintf("user_id = $%d AND is_draft = true", argIdx)
		args = append(args, p.UserID)
		argIdx++
		if p.ActiveOnly {
			whereSQL += " AND is_active = true"
		}
		sql := fmt.Sprintf("SELECT %s FROM courseware_templates WHERE %s ORDER BY updated_at DESC, created_at DESC",
			cwTemplateScanColumns, whereSQL)
		return runTemplateQuery(ctx, sql, args)
	}

	// 系统模板永远可见
	if p.ScopeFilter == "" || p.ScopeFilter == models.CWTemplateScopeSystem {
		orClauses = append(orClauses, "scope = 'system'")
	}

	// 学校模板(用户属于该学校)
	if p.SchoolID != "" && (p.ScopeFilter == "" || p.ScopeFilter == models.CWTemplateScopeSchool) {
		orClauses = append(orClauses, fmt.Sprintf("(scope = 'school' AND scope_target_id = $%d)", argIdx))
		args = append(args, p.SchoolID)
		argIdx++
	}

	// 教研组模板(用户属于这些教研组)
	if len(p.GroupIDs) > 0 && (p.ScopeFilter == "" || p.ScopeFilter == models.CWTemplateScopeGroup) {
		// 构建 IN 子句的占位符
		var placeholders []string
		for _, gid := range p.GroupIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
			args = append(args, gid)
			argIdx++
		}
		orClauses = append(orClauses,
			fmt.Sprintf("(scope = 'group' AND scope_target_id IN (%s))", strings.Join(placeholders, ",")))
	}

	// 个人模板(只能看自己的)
	if p.UserID != "" && (p.ScopeFilter == "" || p.ScopeFilter == models.CWTemplateScopePersonal) {
		orClauses = append(orClauses, fmt.Sprintf("(scope = 'personal' AND user_id = $%d)", argIdx))
		args = append(args, p.UserID)
		argIdx++
	}

	// 没有任何可见 scope
	if len(orClauses) == 0 {
		return []*models.CoursewareTemplate{}, nil
	}

	whereSQL := "(" + strings.Join(orClauses, " OR ") + ")"

	// 草稿过滤
	if !p.IncludeDrafts {
		whereSQL += " AND is_draft = false"
	}

	// 激活过滤
	if p.ActiveOnly {
		whereSQL += " AND is_active = true"
	}

	// 排序:scope 字典序刚好是 system < group < personal < school,我们要按业务顺序排
	// 用 CASE 强制业务排序: system(0) → school(1) → group(2) → personal(3)
	orderSQL := `ORDER BY
		CASE scope
			WHEN 'system' THEN 0
			WHEN 'school' THEN 1
			WHEN 'group' THEN 2
			WHEN 'personal' THEN 3
			ELSE 9
		END ASC,
		sort_order ASC, created_at ASC`

	sql := fmt.Sprintf("SELECT %s FROM courseware_templates WHERE %s %s",
		cwTemplateScanColumns, whereSQL, orderSQL)

	return runTemplateQuery(ctx, sql, args)
}

// runTemplateQuery 辅助函数:执行模板查询并扫描结果
func runTemplateQuery(ctx context.Context, sql string, args []interface{}) ([]*models.CoursewareTemplate, error) {
	rows, err := database.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}
	defer rows.Close()

	var templates []*models.CoursewareTemplate
	for rows.Next() {
		t, err := scanCWTemplate(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("扫描模板行失败: %w", err)
		}
		templates = append(templates, t)
	}
	return templates, nil
}

// ==================== v139 新增:AI 提取草稿 ====================

// CreateDraftTemplate v139:创建 AI 提取草稿模板
// 入库时 is_draft=true, scope=personal, user_id=当前用户
// extract_source_meta 存储来源元信息(source_type/original_html_length/extracted_at)
func CreateDraftTemplate(ctx context.Context, t *models.CoursewareTemplate, sourceMeta map[string]interface{}) error {
	sourceMetaJSON, _ := json.Marshal(sourceMeta)

	sql := `INSERT INTO courseware_templates (id, name, description, style_category,
		color_scheme, css_variables, sample_pages, preview_urls,
		is_active, sort_order, user_id, scope, is_draft, extract_source_meta)
		VALUES (gen_random_uuid(), $1, $2, $3,
		$4::jsonb, $5::jsonb, $6::jsonb, $7::jsonb,
		true, 0, $8, 'personal', true, $9::jsonb)
		RETURNING id, created_at, updated_at`
	return database.DB.QueryRow(ctx, sql,
		t.Name, t.Description, t.StyleCategory,
		nullIfEmpty(t.ColorScheme), nullIfEmpty(t.CSSVariables),
		nullIfEmpty(t.SamplePages), nullIfEmpty(t.PreviewURLs),
		t.UserID,
		string(sourceMetaJSON),
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// ListMyDrafts v139:查询当前用户的所有草稿模板
// 按 updated_at 倒序,最近编辑的在最前
func ListMyDrafts(ctx context.Context, userID string) ([]*models.CoursewareTemplate, error) {
	sql := fmt.Sprintf(`SELECT %s FROM courseware_templates
		WHERE user_id = $1 AND is_draft = true
		ORDER BY updated_at DESC, created_at DESC`, cwTemplateScanColumns)
	return runTemplateQuery(ctx, sql, []interface{}{userID})
}

// ==================== v139 新增:AI 微调更新 + 历史快照 ====================

// 微调历史最多保留条数
const maxRefineHistoryEntries = 20

// UpdateTemplateRefined v139:AI 微调后更新模板,自动写入 refine_history 快照
//
// 操作流程:
//   1. 读取当前模板的 sample_pages / css_variables / color_scheme(作为快照)
//   2. 把快照追加到 refine_history 数组的最前面(LIFO,最近的在 [0])
//   3. 数组超过 20 条时裁剪掉最老的
//   4. 用新内容覆盖 sample_pages / css_variables / color_scheme
//   5. 同步 updated_at 和 style_category(若 AI 建议变更)
//
// 注意:此函数只允许更新自己的模板(在 service 层校验权限)
func UpdateTemplateRefined(
	ctx context.Context,
	id string,
	newSamplePages string,
	newCSSVariables string,
	newColorScheme string,
	newStyleCategory string,
	historyEntry models.RefineHistoryEntry,
) error {
	// 1. 读取当前 refine_history
	var currentHistoryJSON string
	err := database.DB.QueryRow(ctx,
		`SELECT COALESCE(refine_history::text,'[]') FROM courseware_templates WHERE id = $1`,
		id,
	).Scan(&currentHistoryJSON)
	if err != nil {
		return fmt.Errorf("读取微调历史失败: %w", err)
	}

	// 2. 解析现有历史 + 追加新快照到最前
	var historyEntries []models.RefineHistoryEntry
	if currentHistoryJSON != "" && currentHistoryJSON != "[]" {
		if err := json.Unmarshal([]byte(currentHistoryJSON), &historyEntries); err != nil {
			// 历史损坏不致命,重置为空
			historyEntries = []models.RefineHistoryEntry{}
		}
	}
	// LIFO 追加到最前
	historyEntries = append([]models.RefineHistoryEntry{historyEntry}, historyEntries...)
	// 裁剪到最多 20 条
	if len(historyEntries) > maxRefineHistoryEntries {
		historyEntries = historyEntries[:maxRefineHistoryEntries]
	}

	newHistoryJSON, err := json.Marshal(historyEntries)
	if err != nil {
		return fmt.Errorf("序列化微调历史失败: %w", err)
	}

	// 3. 一次 UPDATE 完成所有更新
	updateSQL := `UPDATE courseware_templates
		SET sample_pages = $1::jsonb,
		    css_variables = $2::jsonb,
		    color_scheme = $3::jsonb,
		    refine_history = $4::jsonb,
		    updated_at = $5`
	args := []interface{}{
		nullIfEmpty(newSamplePages),
		nullIfEmpty(newCSSVariables),
		nullIfEmpty(newColorScheme),
		string(newHistoryJSON),
		time.Now(),
	}
	argIdx := 6
	if newStyleCategory != "" {
		updateSQL += fmt.Sprintf(", style_category = $%d", argIdx)
		args = append(args, newStyleCategory)
		argIdx++
	}
	updateSQL += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	tag, err := database.DB.Exec(ctx, updateSQL, args...)
	if err != nil {
		return fmt.Errorf("更新微调模板失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("模板不存在: %s", id)
	}
	return nil
}

// GetTemplateForRefine v139:加载模板供微调使用(校验所有权由 service 层负责)
// 实质就是 GetCWTemplateByID 的别名,语义更清晰
func GetTemplateForRefine(ctx context.Context, id string) (*models.CoursewareTemplate, error) {
	return GetCWTemplateByID(ctx, id)
}

// ==================== v139 新增:历史回退 ====================

// GetRefineHistory v139:读取模板的微调历史数组
// 数组按 LIFO 排列,索引 0 = 最近一次微调前的快照
func GetRefineHistory(ctx context.Context, id string) ([]models.RefineHistoryEntry, error) {
	var historyJSON string
	err := database.DB.QueryRow(ctx,
		`SELECT COALESCE(refine_history::text,'[]') FROM courseware_templates WHERE id = $1`,
		id,
	).Scan(&historyJSON)
	if err != nil {
		return nil, fmt.Errorf("读取微调历史失败: %w", err)
	}

	var entries []models.RefineHistoryEntry
	if historyJSON == "" || historyJSON == "[]" {
		return []models.RefineHistoryEntry{}, nil
	}
	if err := json.Unmarshal([]byte(historyJSON), &entries); err != nil {
		return nil, fmt.Errorf("解析微调历史失败: %w", err)
	}
	return entries, nil
}

// RollbackToHistory v139:把第 N 条历史快照恢复为当前内容
//
// 操作流程:
//   1. 读取历史数组
//   2. 取出第 N 条快照
//   3. 把当前内容也作为新快照存入历史(LIFO,继续往前)
//   4. 用第 N 条快照的内容覆盖 sample_pages / css_variables / color_scheme
//   5. 从历史数组中删除第 N 条及之前的所有条目(因为已经回退到那个时点)
//
// 注意:回退也是一种修改,会消耗一个历史名额。但用户在回退后可以继续微调,新的 LIFO 头部会被覆盖
func RollbackToHistory(ctx context.Context, id string, historyIndex int) error {
	// 1. 读取当前模板的 3 个核心字段 + history
	var currentSamplePages, currentCSSVariables, currentColorScheme, historyJSON string
	err := database.DB.QueryRow(ctx,
		`SELECT COALESCE(sample_pages::text,'[]'),
		        COALESCE(css_variables::text,'{}'),
		        COALESCE(color_scheme::text,'{}'),
		        COALESCE(refine_history::text,'[]')
		   FROM courseware_templates WHERE id = $1`,
		id,
	).Scan(&currentSamplePages, &currentCSSVariables, &currentColorScheme, &historyJSON)
	if err != nil {
		return fmt.Errorf("读取模板失败: %w", err)
	}

	// 2. 解析历史
	var entries []models.RefineHistoryEntry
	if err := json.Unmarshal([]byte(historyJSON), &entries); err != nil {
		return fmt.Errorf("解析微调历史失败: %w", err)
	}
	if historyIndex < 0 || historyIndex >= len(entries) {
		return fmt.Errorf("历史索引越界(共 %d 条,请求第 %d 条)", len(entries), historyIndex)
	}

	// 3. 取出目标快照
	targetEntry := entries[historyIndex]
	// 用快照内的 sample_pages 直接序列化成 JSON 字符串数组
	targetSamplePagesJSON, err := json.Marshal(targetEntry.SamplePagesBefore)
	if err != nil {
		return fmt.Errorf("序列化目标快照样例页失败: %w", err)
	}

	// 4. 把当前内容存为新的快照(放在数组最前)
	rollbackEntry := models.RefineHistoryEntry{
		Timestamp:          time.Now().Format(time.RFC3339),
		UserInstruction:    fmt.Sprintf("[回退] 从第 %d 条历史快照恢复", historyIndex+1),
		CSSVariablesBefore: currentCSSVariables,
		ColorSchemeBefore:  currentColorScheme,
		ChangeSummary:      fmt.Sprintf("回退到 %s 的版本", targetEntry.Timestamp),
	}
	// 解析当前 sample_pages 为字符串数组
	var currentPagesArr []string
	if err := json.Unmarshal([]byte(currentSamplePages), &currentPagesArr); err == nil {
		rollbackEntry.SamplePagesBefore = currentPagesArr
	}

	// 5. 新的历史 = 回退记录 + 删除掉 [0..historyIndex] 的旧记录
	//    这样保留了 historyIndex 之前的更老的快照,仍然可以继续往前回退
	var newHistory []models.RefineHistoryEntry
	newHistory = append(newHistory, rollbackEntry)
	if historyIndex+1 < len(entries) {
		newHistory = append(newHistory, entries[historyIndex+1:]...)
	}
	// 裁剪到 20 条
	if len(newHistory) > maxRefineHistoryEntries {
		newHistory = newHistory[:maxRefineHistoryEntries]
	}
	newHistoryJSON, _ := json.Marshal(newHistory)

	// 6. 执行回退更新
	targetSamplePagesStr := string(targetSamplePagesJSON)
	updateSQL := `UPDATE courseware_templates
		SET sample_pages = $1::jsonb,
		    css_variables = $2::jsonb,
		    color_scheme = $3::jsonb,
		    refine_history = $4::jsonb,
		    updated_at = $5
		 WHERE id = $6`
	_, err = database.DB.Exec(ctx, updateSQL,
		nullIfEmpty(targetSamplePagesStr),
		nullIfEmpty(targetEntry.CSSVariablesBefore),
		nullIfEmpty(targetEntry.ColorSchemeBefore),
		string(newHistoryJSON),
		time.Now(),
		id,
	)
	if err != nil {
		return fmt.Errorf("回退模板失败: %w", err)
	}
	return nil
}

// ==================== v139 新增:草稿发布为正式 ====================

// PublishDraft v139:把草稿模板转为正式模板
//
// 操作流程:
//   1. 校验模板存在且 is_draft=true(在 service 层做)
//   2. 更新 scope/scope_target_id/name/description/style_category
//   3. is_draft 改为 false
//   4. 清空 refine_history(发布后微调历史归零,留给新一轮使用)
//   5. 同步 updated_at
//
// 权限校验已在 service 层做完(传入此函数时已确认通过)
func PublishDraft(
	ctx context.Context,
	id string,
	name string,
	description string,
	styleCategory string,
	scope string,
	scopeTargetID string,
) error {
	// scope_target_id 处理:scope=school/group 时非空,其他为 NULL
	var scopeTargetVal interface{}
	if scopeTargetID != "" {
		scopeTargetVal = scopeTargetID
	} else {
		scopeTargetVal = nil
	}

	updateSQL := `UPDATE courseware_templates
		SET name = $1,
		    description = $2,
		    style_category = $3,
		    scope = $4,
		    scope_target_id = $5,
		    is_draft = false,
		    refine_history = NULL,
		    updated_at = $6
		 WHERE id = $7`
	tag, err := database.DB.Exec(ctx, updateSQL,
		name,
		description,
		styleCategory,
		scope,
		scopeTargetVal,
		time.Now(),
		id,
	)
	if err != nil {
		return fmt.Errorf("发布草稿模板失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("草稿模板不存在: %s", id)
	}
	return nil
}

// ==================== v139 新增:删除草稿(本人) ====================


// ==================== v142 新增:撤回已发布模板 ====================

// UnpublishTemplate v142:撤回已发布模板为草稿
//
// 操作流程:
//   1. 将 is_draft 改为 true
//   2. scope 重置为 personal(撤回后变成个人草稿)
//   3. scope_target_id 置空
//   4. 更新 updated_at
//
// 权限校验由 handler 层完成,此处只做数据库操作
func UnpublishTemplate(ctx context.Context, id string) error {
	updateSQL := `UPDATE courseware_templates
		SET is_draft = true,
		    scope = 'personal',
		    scope_target_id = NULL,
		    updated_at = $1
		 WHERE id = $2 AND is_draft = false`
	tag, err := database.DB.Exec(ctx, updateSQL, time.Now(), id)
	if err != nil {
		return fmt.Errorf("撤回模板失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("模板不存在或已是草稿状态")
	}
	return nil
}

// DeleteDraftTemplate v139:删除草稿模板(仅限创建者本人)
func DeleteDraftTemplate(ctx context.Context, id string, userID string) error {
	sql := `DELETE FROM courseware_templates WHERE id = $1 AND user_id = $2 AND is_draft = true`
	tag, err := database.DB.Exec(ctx, sql, id, userID)
	if err != nil {
		return fmt.Errorf("删除草稿失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("草稿不存在或无权删除")
	}
	return nil
}
