package repository

// recipe_repo.go — 备课配方数据访问层
//
// 本文件包含：
//   - 错误常量（ErrRecipeNotFound）
//   - CRUD（CreateRecipe / GetRecipeByID / ListRecipes / UpdateRecipe / DeleteRecipe）
//   - Fork + Share
//   - 使用记录（RecordRecipeUsage）
// //
// 效果统计（GetRecipeStats）和市场排行榜（ListMarketRecipes）
// 已拆分至 recipe_repo_market.go（v92重构）

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
//
// B2-C：配方仓储教育域快照。
// education_domain由数据库BEFORE INSERT触发器确定：
//   - 普通创建按作者当前教学域写入；
//   - Fork创建按来源配方教育域继承。
//
// 创建完成后通过RETURNING回填数据库最终快照。
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
		RETURNING id, education_domain, created_at, updated_at
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

	// 教案结构默认空数组
	lessonStructure := "[]"
	if r.LessonStructure != "" && r.LessonStructure != "[]" {
		lessonStructure = r.LessonStructure
	}

	// 备课模式默认guided
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
	).Scan(
		&r.ID,
		&r.EducationDomain,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建配方失败: %w", err)
	}
	return nil
}

// ==================== 查询配方 ====================

// GetRecipeByID 根据ID查询配方完整信息
func GetRecipeByID(ctx context.Context, id string) (*models.TeachingRecipe, error) {
	r := &models.TeachingRecipe{}
	query := `
		SELECT id, name, COALESCE(description, ''), subject, grade_range,
		       COALESCE(component_ids::text, '[]'), COALESCE(student_profile, ''),
		       COALESCE(teaching_style, ''), COALESCE(school_requirements, ''),
		       COALESCE(custom_notes, ''), COALESCE(custom_prompt, ''),
		       scope, scope_ref_id, author_id, education_domain, fork_count, forked_from,
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
		&r.Scope, &r.ScopeRefID, &r.AuthorID, &r.EducationDomain, &r.ForkCount, &r.ForkedFrom,
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
	listQuery := fmt.Sprintf(`
		SELECT r.id, r.name, COALESCE(r.description, ''), r.subject, r.grade_range,
		       COALESCE(jsonb_array_length(r.component_ids), 0),
		       r.scope, r.author_id, r.education_domain, COALESCE(u.display_name, u.username),
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
			&item.Scope, &item.AuthorID, &item.EducationDomain, &item.AuthorName,
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
func UpdateRecipe(ctx context.Context, id string, req *models.UpdateRecipeRequest) error {
	componentJSON, _ := json.Marshal(req.ComponentIDs)
	if req.ComponentIDs == nil {
		componentJSON = []byte("[]")
	}

	// 教案结构默认空数组
	lessonStructure := req.LessonStructure
	if lessonStructure == "" {
		lessonStructure = "[]"
	}

	// 备课模式默认guided
	promptMode := req.PromptMode
	if promptMode == "" {
		promptMode = models.PromptModeGuided
	}

	// 流程配置默认空数组
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
		    subject = $12, grade_range = $13,
		    version = version + 1, updated_at = $14
		WHERE id = $15 AND status = 'active'
	`,
		req.Name, req.Description, string(componentJSON),
		req.StudentProfile, req.TeachingStyle, req.SchoolRequirements,
		req.CustomNotes, req.CustomPrompt,
		lessonStructure, promptMode,
		stagesConfig,
		req.Subject, req.GradeRange,
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
		StagesConfig:       src.StagesConfig,
		LessonStructure:    src.LessonStructure,
		PromptMode:         src.PromptMode,
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
