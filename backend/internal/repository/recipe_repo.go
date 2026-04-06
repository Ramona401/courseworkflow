package repository

// recipe_repo.go — 备课配方数据访问层
//
// Phase 7A 新增：CRUD+Fork+Share+学情+使用记录+组件查询
// Phase 7B-8 修改：stages_config列读写
// 迭代1 修改：
//   - CreateRecipe：INSERT增加lesson_structure+prompt_mode（$16,$17）
//   - GetRecipeByID：SELECT增加lesson_structure+prompt_mode
//   - UpdateRecipe：UPDATE增加lesson_structure+prompt_mode
//   - ForkRecipe：复制lesson_structure+prompt_mode
//   - ListRecipes：SELECT增加prompt_mode供列表展示
// 迭代2 修改：
//   - UpdateRecipe：UPDATE增加stages_config
//   - ListRecipes：SELECT增加stages_config供列表展示阶段数量

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrRecipeNotFound = errors.New("配方不存在")
)

// ==================== 创建配方 ====================

// CreateRecipe 创建备课配方
// 迭代1：INSERT增加lesson_structure($16)+prompt_mode($17)
func CreateRecipe(ctx context.Context, r *models.TeachingRecipe) error {
	query := `
		INSERT INTO teaching_recipes (
			name, description, subject, grade_range, component_ids,
			student_profile, teaching_style, school_requirements, custom_notes, custom_prompt,
			scope, scope_ref_id, author_id, forked_from, stages_config,
			lesson_structure, prompt_mode
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17
		)
		RETURNING id, created_at, updated_at
	`
	// 组件ID列表序列化为JSON
	componentJSON := "[]"
	if r.ComponentIDs != "" && r.ComponentIDs != "[]" {
		componentJSON = r.ComponentIDs
	}

	// 阶段覆盖配置默认空数组
	stagesConfig := "[]"
	if r.StagesConfig != "" && r.StagesConfig != "[]" {
		stagesConfig = r.StagesConfig
	}

	// 迭代1：教案结构默认空数组
	lessonStructure := "[]"
	if r.LessonStructure != "" && r.LessonStructure != "[]" {
		lessonStructure = r.LessonStructure
	}

	// 迭代1：备课模式默认guided
	promptMode := r.PromptMode
	if promptMode == "" {
		promptMode = models.PromptModeGuided
	}

	scope := r.Scope
	if scope == "" {
		scope = models.RecipeScopePersonal
	}

	err := database.DB.QueryRow(ctx, query,
		r.Name, r.Description, r.Subject, r.GradeRange, componentJSON,
		r.StudentProfile, r.TeachingStyle, r.SchoolRequirements, r.CustomNotes, r.CustomPrompt,
		scope, r.ScopeRefID, r.AuthorID, r.ForkedFrom, stagesConfig,
		lessonStructure, promptMode,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建配方失败: %w", err)
	}
	return nil
}

// ==================== 查询配方 ====================

// GetRecipeByID 根据ID查询配方完整信息
// 迭代1：SELECT增加lesson_structure+prompt_mode
func GetRecipeByID(ctx context.Context, id string) (*models.TeachingRecipe, error) {
	r := &models.TeachingRecipe{}
	query := `
		SELECT id, name, COALESCE(description, ''), subject, grade_range,
		       COALESCE(component_ids::text, '[]'), COALESCE(student_profile, ''),
		       COALESCE(teaching_style, ''), COALESCE(school_requirements, ''),
		       COALESCE(custom_notes, ''), COALESCE(custom_prompt, ''),
		       scope, scope_ref_id, author_id, fork_count, forked_from,
		       use_count, version, status,
		       COALESCE(stages_config::text, '[]'),
		       COALESCE(lesson_structure::text, '[]'),
		       COALESCE(prompt_mode, 'guided'),
		       created_at, updated_at
		FROM teaching_recipes WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&r.ID, &r.Name, &r.Description, &r.Subject, &r.GradeRange,
		&r.ComponentIDs, &r.StudentProfile,
		&r.TeachingStyle, &r.SchoolRequirements,
		&r.CustomNotes, &r.CustomPrompt,
		&r.Scope, &r.ScopeRefID, &r.AuthorID, &r.ForkCount, &r.ForkedFrom,
		&r.UseCount, &r.Version, &r.Status,
		&r.StagesConfig,
		&r.LessonStructure,
		&r.PromptMode,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecipeNotFound
		}
		return nil, fmt.Errorf("查询配方失败: %w", err)
	}
	return r, nil
}

// ==================== 列表查询 ====================

// ListRecipes 查询配方列表（支持多条件筛选）
// 迭代1：SELECT增加prompt_mode供列表展示
// 迭代2：SELECT增加stages_config供列表展示阶段数量
func ListRecipes(ctx context.Context, authorID string, scope string, scopeRefID string, subject string, gradeRange string, limit int, offset int) ([]*models.RecipeListItem, int, error) {
	// 构建WHERE：我的配方 OR 共享给我的配方
	where := " WHERE r.status = 'active' AND ("
	args := []interface{}{}
	argIdx := 1

	// 条件1：我创建的
	where += fmt.Sprintf("r.author_id = $%d", argIdx)
	args = append(args, authorID)
	argIdx++

	// 条件2：共享给指定scope的（教研组或学校）
	if scopeRefID != "" {
		where += fmt.Sprintf(" OR (r.scope IN ('group','school') AND r.scope_ref_id = $%d)", argIdx)
		args = append(args, scopeRefID)
		argIdx++
	}
	where += ")"

	// 额外筛选：仅看特定scope
	if scope != "" {
		where += fmt.Sprintf(" AND r.scope = $%d", argIdx)
		args = append(args, scope)
		argIdx++
	}
	// 学科筛选
	if subject != "" {
		where += fmt.Sprintf(" AND r.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}
	// 年级筛选
	if gradeRange != "" {
		where += fmt.Sprintf(" AND r.grade_range = $%d", argIdx)
		args = append(args, gradeRange)
		argIdx++
	}

	// 查总数
	countQuery := "SELECT COUNT(*) FROM teaching_recipes r" + where
	var total int
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询配方总数失败: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	// 查列表（LEFT JOIN users 取作者名）
	// 迭代1：增加 COALESCE(r.prompt_mode, 'guided')
	// 迭代2：增加 COALESCE(r.stages_config::text, '[]')
	listQuery := fmt.Sprintf(`
		SELECT r.id, r.name, COALESCE(r.description, ''), r.subject, r.grade_range,
		       COALESCE(jsonb_array_length(r.component_ids), 0),
		       r.scope, r.author_id, COALESCE(u.display_name, u.username),
		       r.fork_count, r.use_count, r.version, r.forked_from, r.status,
		       COALESCE(r.prompt_mode, 'guided'),
		       COALESCE(r.stages_config::text, '[]'),
		       r.created_at, r.updated_at
		FROM teaching_recipes r
		LEFT JOIN users u ON u.id = r.author_id
		%s
		ORDER BY r.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询配方列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.RecipeListItem
	for rows.Next() {
		item := &models.RecipeListItem{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.Subject, &item.GradeRange,
			&item.ComponentCount,
			&item.Scope, &item.AuthorID, &item.AuthorName,
			&item.ForkCount, &item.UseCount, &item.Version, &item.ForkedFrom, &item.Status,
			&item.PromptMode,
			&item.StagesConfig,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描配方行失败: %w", err)
		}
		item.ScopeName = models.RecipeScopeNameMap[item.Scope]
		items = append(items, item)
	}
	if items == nil {
		items = []*models.RecipeListItem{}
	}
	return items, total, nil
}

// ==================== 更新配方 ====================

// UpdateRecipe 更新配方（全量更新可编辑字段）
// 迭代1：增加lesson_structure+prompt_mode
// 迭代2：增加stages_config
func UpdateRecipe(ctx context.Context, id string, req *models.UpdateRecipeRequest) error {
	componentJSON, _ := json.Marshal(req.ComponentIDs)
	if req.ComponentIDs == nil {
		componentJSON = []byte("[]")
	}

	// 迭代1：教案结构默认空数组
	lessonStructure := req.LessonStructure
	if lessonStructure == "" {
		lessonStructure = "[]"
	}

	// 迭代1：备课模式默认guided
	promptMode := req.PromptMode
	if promptMode == "" {
		promptMode = models.PromptModeGuided
	}

	// 迭代2：流程配置默认空数组
	stagesConfig := req.StagesConfig
	if stagesConfig == "" {
		stagesConfig = "[]"
	}

	now := time.Now()
	result, err := database.DB.Exec(ctx, `
		UPDATE teaching_recipes
		SET name = $1, description = $2, component_ids = $3,
		    student_profile = $4, teaching_style = $5, school_requirements = $6,
		    custom_notes = $7, custom_prompt = $8,
		    lesson_structure = $9, prompt_mode = $10,
		    stages_config = $11,
		    version = version + 1, updated_at = $12
		WHERE id = $13 AND status = 'active'
	`,
		req.Name, req.Description, string(componentJSON),
		req.StudentProfile, req.TeachingStyle, req.SchoolRequirements,
		req.CustomNotes, req.CustomPrompt,
		lessonStructure, promptMode,
		stagesConfig,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("更新配方失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRecipeNotFound
	}
	return nil
}

// UpdateRecipeStudentProfile 单独更新学情记录
func UpdateRecipeStudentProfile(ctx context.Context, id string, profile string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE teaching_recipes SET student_profile = $1, updated_at = $2 WHERE id = $3 AND status = 'active'`,
		profile, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新学情失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRecipeNotFound
	}
	return nil
}

// ==================== 删除配方 ====================

// DeleteRecipe 删除配方（软删除：archived）
func DeleteRecipe(ctx context.Context, id string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE teaching_recipes SET status = 'archived', updated_at = $1 WHERE id = $2`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("删除配方失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRecipeNotFound
	}
	return nil
}

// ==================== Fork配方 ====================

// ForkRecipe 复制配方到当前用户名下（个人副本）
// 迭代1：复制lesson_structure+prompt_mode
func ForkRecipe(ctx context.Context, sourceID string, newAuthorID string) (*models.TeachingRecipe, error) {
	src, err := GetRecipeByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	forked := &models.TeachingRecipe{
		Name:               src.Name + "（副本）",
		Description:        src.Description,
		Subject:            src.Subject,
		GradeRange:         src.GradeRange,
		ComponentIDs:       src.ComponentIDs,
		StudentProfile:     "", // 学情不复制，因为是个人化的
		TeachingStyle:      src.TeachingStyle,
		SchoolRequirements: src.SchoolRequirements,
		CustomNotes:        src.CustomNotes,
		CustomPrompt:       src.CustomPrompt,
		StagesConfig:       src.StagesConfig,    // 复制流程配置
		LessonStructure:    src.LessonStructure,  // 迭代1：复制教案结构
		PromptMode:         src.PromptMode,       // 迭代1：复制备课模式
		Scope:              models.RecipeScopePersonal,
		AuthorID:           newAuthorID,
		ForkedFrom:         &sourceID,
	}
	if err := CreateRecipe(ctx, forked); err != nil {
		return nil, err
	}

	// 原配方fork_count+1
	_, _ = database.DB.Exec(ctx,
		`UPDATE teaching_recipes SET fork_count = fork_count + 1 WHERE id = $1`, sourceID)

	return forked, nil
}

// ==================== 共享配方 ====================

// ShareRecipe 更新配方的共享范围
func ShareRecipe(ctx context.Context, id string, scope string, scopeRefID string) error {
	now := time.Now()
	result, err := database.DB.Exec(ctx,
		`UPDATE teaching_recipes SET scope = $1, scope_ref_id = $2, updated_at = $3 WHERE id = $4 AND status = 'active'`,
		scope, scopeRefID, now, id,
	)
	if err != nil {
		return fmt.Errorf("共享配方失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRecipeNotFound
	}
	return nil
}

// ==================== 使用记录 ====================

// RecordRecipeUsage 记录配方使用+递增use_count
func RecordRecipeUsage(ctx context.Context, recipeID string, planID string, userID string) error {
	// 写入使用日志
	_, err := database.DB.Exec(ctx,
		`INSERT INTO recipe_usage_log (recipe_id, lesson_plan_id, user_id) VALUES ($1, $2, $3)`,
		recipeID, planID, userID,
	)
	if err != nil {
		return fmt.Errorf("记录配方使用失败: %w", err)
	}
	// 递增use_count
	_, _ = database.DB.Exec(ctx,
		`UPDATE teaching_recipes SET use_count = use_count + 1 WHERE id = $1`, recipeID)
	return nil
}

// ==================== 组件摘要查询 ====================

// GetRecipeComponentBriefs 批量查询配方绑定的组件摘要信息
func GetRecipeComponentBriefs(ctx context.Context, componentIDs []string) ([]*models.RecipeComponentBrief, error) {
	if len(componentIDs) == 0 {
		return []*models.RecipeComponentBrief{}, nil
	}

	query := `
		SELECT id, library_type, display_label, quality_score, status
		FROM lesson_plan_components
		WHERE id = ANY($1)
		ORDER BY library_type, quality_score DESC
	`
	rows, err := database.DB.Query(ctx, query, componentIDs)
	if err != nil {
		return nil, fmt.Errorf("查询组件摘要失败: %w", err)
	}
	defer rows.Close()

	var items []*models.RecipeComponentBrief
	for rows.Next() {
		item := &models.RecipeComponentBrief{}
		if err := rows.Scan(&item.ID, &item.LibraryType, &item.DisplayLabel, &item.QualityScore, &item.Status); err != nil {
			return nil, fmt.Errorf("扫描组件摘要行失败: %w", err)
		}
		item.LibraryName = models.LibraryTypeNameMap[item.LibraryType]
		items = append(items, item)
	}
	if items == nil {
		items = []*models.RecipeComponentBrief{}
	}
	return items, nil
}

// ==================== 获取组件完整内容（用于构建AI上下文）====================

// GetRecipeComponentContents 查询配方绑定组件的完整内容（用于注入AI提示词）
// 返回按library_type分组的组件，包含display_label+design_logic+full_guide
func GetRecipeComponentContents(ctx context.Context, componentIDs []string) ([]*models.MatchedComponentGroup, error) {
	if len(componentIDs) == 0 {
		return []*models.MatchedComponentGroup{}, nil
	}

	query := `
		SELECT library_type, id, display_label, COALESCE(design_logic, ''),
		       COALESCE(example_snippet, ''), COALESCE(full_guide, ''),
		       quality_score, usage_count, select_count
		FROM lesson_plan_components
		WHERE id = ANY($1) AND status = 'active' AND review_status = 'approved'
		ORDER BY library_type, quality_score DESC
	`
	rows, err := database.DB.Query(ctx, query, componentIDs)
	if err != nil {
		return nil, fmt.Errorf("查询配方组件内容失败: %w", err)
	}
	defer rows.Close()

	groupMap := make(map[string]*models.MatchedComponentGroup)
	var groupOrder []string

	for rows.Next() {
		var libraryType string
		mc := &models.MatchedComponent{}
		if err := rows.Scan(
			&libraryType, &mc.ID, &mc.DisplayLabel, &mc.DesignLogic,
			&mc.ExampleSnippet, &mc.FullGuide,
			&mc.QualityScore, &mc.UsageCount, &mc.SelectCount,
		); err != nil {
			return nil, fmt.Errorf("扫描组件内容行失败: %w", err)
		}
		group, exists := groupMap[libraryType]
		if !exists {
			group = &models.MatchedComponentGroup{
				LibraryType: libraryType,
				LibraryName: models.LibraryTypeNameMap[libraryType],
				Components:  []*models.MatchedComponent{},
			}
			groupMap[libraryType] = group
			groupOrder = append(groupOrder, libraryType)
		}
		group.Components = append(group.Components, mc)
	}

	var result []*models.MatchedComponentGroup
	for _, lt := range groupOrder {
		result = append(result, groupMap[lt])
	}
	return result, nil
}

// ==================== 迭代6新增：配方效果统计 ====================

// RecipeStatsRow 配方统计数据行
type RecipeStatsRow struct {
	TotalUsage     int     // 总使用次数
	TotalPlans     int     // 产出教案总数
	AvgScore       float64 // 教案平均分
	RecentUsages   []RecipeUsageRow // 最近使用记录
}

// RecipeUsageRow 单条使用记录
type RecipeUsageRow struct {
	LessonPlanID  *string  `json:"lesson_plan_id"`
	UserName      string   `json:"user_name"`
	AIReviewScore *float64 `json:"ai_review_score"`
	CreatedAt     string   `json:"created_at"`
}

// GetRecipeStats 获取配方效果统计数据
// 从 recipe_usage_log 表聚合使用次数、教案数、平均分，并返回最近20条使用记录
func GetRecipeStats(ctx context.Context, recipeID string) (*RecipeStatsRow, error) {
	stats := &RecipeStatsRow{}

	// 1. 聚合统计（使用次数、有教案的记录数、平均分）
	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(lesson_plan_id),
		       COALESCE(AVG(ai_review_score) FILTER (WHERE ai_review_score IS NOT NULL), 0)
		FROM recipe_usage_log
		WHERE recipe_id = $1
	`, recipeID).Scan(&stats.TotalUsage, &stats.TotalPlans, &stats.AvgScore)
	if err != nil {
		return nil, fmt.Errorf("查询配方统计失败: %w", err)
	}

	// 2. 最近20条使用记录（含用户名和教案分数）
	rows, err := database.DB.Query(ctx, `
		SELECT rul.lesson_plan_id,
		       COALESCE(u.display_name, u.username, '未知'),
		       rul.ai_review_score,
		       TO_CHAR(rul.created_at, 'YYYY-MM-DD HH24:MI')
		FROM recipe_usage_log rul
		LEFT JOIN users u ON u.id = rul.user_id
		WHERE rul.recipe_id = $1
		ORDER BY rul.created_at DESC
		LIMIT 20
	`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("查询配方使用记录失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row RecipeUsageRow
		if err := rows.Scan(&row.LessonPlanID, &row.UserName, &row.AIReviewScore, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描使用记录行失败: %w", err)
		}
		stats.RecentUsages = append(stats.RecentUsages, row)
	}
	if stats.RecentUsages == nil {
		stats.RecentUsages = []RecipeUsageRow{}
	}

	return stats, nil
}

// ==================== 迭代6新增：配方市场排行榜 ====================

// MarketRecipeItem 市场配方列表项（含统计数据）
type MarketRecipeItem struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Subject        string   `json:"subject"`
	GradeRange     string   `json:"grade_range"`
	ComponentCount int      `json:"component_count"`
	Scope          string   `json:"scope"`
	ScopeName      string   `json:"scope_name"`
	AuthorID       string   `json:"author_id"`
	AuthorName     string   `json:"author_name"`
	ForkCount      int      `json:"fork_count"`
	UseCount       int      `json:"use_count"`
	AvgScore       float64  `json:"avg_score"`
	PlanCount      int      `json:"plan_count"`
	PromptMode     string   `json:"prompt_mode"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// ListMarketRecipes 查询配方市场排行榜
// 条件：scope为group或school的已共享配方，按综合分排序
// 综合分 = avg_score * 0.5 + log2(use_count+1) * 0.3 + log2(fork_count+1) * 0.2
func ListMarketRecipes(ctx context.Context, subject string, gradeRange string, sortBy string, limit int, offset int) ([]*MarketRecipeItem, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// 构建WHERE条件
	where := " WHERE r.status = 'active' AND r.scope IN ('group','school')"
	args := []interface{}{}
	argIdx := 1

	if subject != "" {
		where += fmt.Sprintf(" AND r.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}
	if gradeRange != "" {
		where += fmt.Sprintf(" AND r.grade_range = $%d", argIdx)
		args = append(args, gradeRange)
		argIdx++
	}

	// 查总数
	countQuery := "SELECT COUNT(*) FROM teaching_recipes r" + where
	var total int
	if err := database.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询市场配方总数失败: %w", err)
	}

	// 排序方式
	orderClause := "ORDER BY composite_score DESC"
	switch sortBy {
	case "use_count":
		orderClause = "ORDER BY r.use_count DESC"
	case "fork_count":
		orderClause = "ORDER BY r.fork_count DESC"
	case "avg_score":
		orderClause = "ORDER BY avg_score DESC"
	case "newest":
		orderClause = "ORDER BY r.created_at DESC"
	}

	// 查列表
	listQuery := fmt.Sprintf(`
		SELECT r.id, r.name, COALESCE(r.description, ''), r.subject, r.grade_range,
		       COALESCE(jsonb_array_length(r.component_ids), 0),
		       r.scope, r.author_id, COALESCE(u.display_name, u.username, ''),
		       r.fork_count, r.use_count,
		       COALESCE(stats.avg_score, 0),
		       COALESCE(stats.plan_count, 0),
		       COALESCE(r.prompt_mode, 'guided'),
		       TO_CHAR(r.created_at, 'YYYY-MM-DD'),
		       TO_CHAR(r.updated_at, 'YYYY-MM-DD'),
		       (COALESCE(stats.avg_score, 0) * 0.5 + LN(r.use_count + 1)/LN(2) * 0.3 + LN(r.fork_count + 1)/LN(2) * 0.2) AS composite_score
		FROM teaching_recipes r
		LEFT JOIN users u ON u.id = r.author_id
		LEFT JOIN LATERAL (
			SELECT AVG(ai_review_score) FILTER (WHERE ai_review_score IS NOT NULL) AS avg_score,
			       COUNT(lesson_plan_id) AS plan_count
			FROM recipe_usage_log WHERE recipe_id = r.id
		) stats ON true
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, where, orderClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := database.DB.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询市场配方列表失败: %w", err)
	}
	defer rows.Close()

	var items []*MarketRecipeItem
	for rows.Next() {
		item := &MarketRecipeItem{}
		var compositeScore float64
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.Subject, &item.GradeRange,
			&item.ComponentCount,
			&item.Scope, &item.AuthorID, &item.AuthorName,
			&item.ForkCount, &item.UseCount,
			&item.AvgScore, &item.PlanCount,
			&item.PromptMode,
			&item.CreatedAt, &item.UpdatedAt,
			&compositeScore,
		); err != nil {
			return nil, 0, fmt.Errorf("扫描市场配方行失败: %w", err)
		}
		item.ScopeName = models.RecipeScopeNameMap[item.Scope]
		items = append(items, item)
	}
	if items == nil {
		items = []*MarketRecipeItem{}
	}
	return items, total, nil
}
