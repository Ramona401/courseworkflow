package services

// recipe_service.go — 备课配方业务逻辑层
//
// 本文件包含：
//   - RecipeService 定义 + 错误常量
//   - CRUD（创建/查询/更新/删除）
//   - Fork + 共享
//   - BuildRecipeContext（配方→AI提示词上下文）
//   - PreviewContext（预览上下文）
//   - RecommendComponents（智能推荐）
//   - GetRecipeStats（效果统计）
//   - ListMarketRecipes（市场排行榜）
//
// 流程校验（ValidateStageFlow）和预设模板（GetFlowPresets）
// 已拆分至 recipe_flow_service.go（v92重构）

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrRecipeNameRequired    = errors.New("配方名称不能为空")
	ErrRecipeSubjectRequired = errors.New("配方适用学科不能为空")
	ErrRecipeGradeRequired   = errors.New("配方适用年级必须选择一年级至高三中的一个具体年级")
	ErrRecipeNotFound        = errors.New("配方不存在")
	ErrRecipeUnauthorized    = errors.New("无权操作此配方")
	ErrRecipeShareInvalid    = errors.New("共享范围无效，可选：group/school")
)

// RecipeService 备课配方服务
type RecipeService struct{}

var recipeLog = logger.WithModule("recipe")

// NewRecipeService 创建配方服务实例
func NewRecipeService() *RecipeService {
	return &RecipeService{}
}

// ==================== 创建配方 ====================

// CreateRecipe 创建备课配方
func (s *RecipeService) CreateRecipe(ctx context.Context, req *models.CreateRecipeRequest, authorID string) (*models.TeachingRecipe, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrRecipeNameRequired
	}
	rawSubject := strings.TrimSpace(req.Subject)
	if rawSubject == "" {
		return nil, ErrRecipeSubjectRequired
	}

	normalizedSubject, normalizedGrade, validScope :=
		normalizeStrictResourceScope(
			rawSubject,
			req.GradeRange,
		)
	if !validScope {
		return nil, ErrRecipeGradeRequired
	}

	req.Subject = normalizedSubject
	req.GradeRange = normalizedGrade

	// 组件ID列表转JSON
	componentJSON := "[]"
	if len(req.ComponentIDs) > 0 {
		b, _ := json.Marshal(req.ComponentIDs)
		componentJSON = string(b)
	}

	// 教案结构
	lessonStructure := req.LessonStructure
	if lessonStructure == "" {
		lessonStructure = "[]"
	}

	// 备课模式
	promptMode := req.PromptMode
	if promptMode == "" {
		promptMode = models.PromptModeGuided
	}

	// 流程配置
	stagesConfig := req.StagesConfig
	if stagesConfig == "" {
		stagesConfig = "[]"
	}

	r := &models.TeachingRecipe{
		Name:               strings.TrimSpace(req.Name),
		Description:        req.Description,
		Subject:            req.Subject,
		GradeRange:         req.GradeRange,
		ComponentIDs:       componentJSON,
		StudentProfile:     req.StudentProfile,
		TeachingStyle:      req.TeachingStyle,
		SchoolRequirements: req.SchoolRequirements,
		CustomNotes:        req.CustomNotes,
		CustomPrompt:       req.CustomPrompt,
		Scope:              models.RecipeScopePersonal,
		AuthorID:           authorID,
		StagesConfig:       stagesConfig,
		LessonStructure:    lessonStructure,
		PromptMode:         promptMode,
	}

	if err := repository.CreateRecipe(ctx, r); err != nil {
		recipeLog.Error("创建配方失败", "error", err)
		return nil, err
	}
	recipeLog.Info("创建配方成功", "recipe_id", r.ID, "name", r.Name, "author", authorID)
	return r, nil
}

// ==================== 查询 ====================

// ListRecipes 查询配方列表
func (s *RecipeService) ListRecipes(ctx context.Context, authorID string, scope string, scopeRefID string, subject string, gradeRange string, limit int, offset int) (*models.RecipeListResponse, error) {
	items, total, err := repository.ListRecipes(ctx, authorID, scope, scopeRefID, subject, gradeRange, limit, offset)
	if err != nil {
		return nil, err
	}
	return &models.RecipeListResponse{Recipes: items, Total: total}, nil
}

// ==================== 更新 ====================

// UpdateRecipe 更新配方（需验证所有权）
func (s *RecipeService) UpdateRecipe(ctx context.Context, recipeID string, req *models.UpdateRecipeRequest, callerID string) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrRecipeNameRequired
	}

	rawSubject := strings.TrimSpace(req.Subject)
	if rawSubject == "" {
		return ErrRecipeSubjectRequired
	}

	normalizedSubject, normalizedGrade, validScope :=
		normalizeStrictResourceScope(
			rawSubject,
			req.GradeRange,
		)
	if !validScope {
		return ErrRecipeGradeRequired
	}

	req.Subject = normalizedSubject
	req.GradeRange = normalizedGrade

	r, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return ErrRecipeNotFound
		}
		return err
	}
	if r.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}

	if err := repository.UpdateRecipe(ctx, recipeID, req); err != nil {
		recipeLog.Error("更新配方失败", "recipe_id", recipeID, "error", err)
		return err
	}
	recipeLog.Info("更新配方成功", "recipe_id", recipeID)
	return nil
}

// UpdateStudentProfile 单独更新学情记录
func (s *RecipeService) UpdateStudentProfile(ctx context.Context, recipeID string, profile string, callerID string) error {
	r, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return ErrRecipeNotFound
		}
		return err
	}
	if r.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}
	return repository.UpdateRecipeStudentProfile(ctx, recipeID, profile)
}

// ==================== 删除 ====================

// DeleteRecipe 删除配方（需验证所有权）
func (s *RecipeService) DeleteRecipe(ctx context.Context, recipeID string, callerID string) error {
	r, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return ErrRecipeNotFound
		}
		return err
	}
	if r.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}
	return repository.DeleteRecipe(ctx, recipeID)
}

// ==================== Fork ====================

// ForkRecipe Fork配方到当前用户
func (s *RecipeService) ForkRecipe(ctx context.Context, recipeID string, callerID string) (*models.TeachingRecipe, error) {
	forked, err := repository.ForkRecipe(ctx, recipeID, callerID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return nil, ErrRecipeNotFound
		}
		recipeLog.Error("Fork配方失败", "source_id", recipeID, "error", err)
		return nil, err
	}
	recipeLog.Info("Fork配方成功", "source_id", recipeID, "forked_id", forked.ID, "user", callerID)
	return forked, nil
}

// ==================== 共享 ====================

// ShareRecipe 共享配方到教研组/学校
func (s *RecipeService) ShareRecipe(ctx context.Context, recipeID string, req *models.ShareRecipeRequest, callerID string) error {
	if req.Scope != models.RecipeScopeGroup && req.Scope != models.RecipeScopeSchool {
		return ErrRecipeShareInvalid
	}
	if strings.TrimSpace(req.ScopeRefID) == "" {
		return errors.New("共享目标ID不能为空")
	}

	r, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return ErrRecipeNotFound
		}
		return err
	}
	if r.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}

	if err := repository.ShareRecipe(ctx, recipeID, req.Scope, req.ScopeRefID); err != nil {
		recipeLog.Error("共享配方失败", "recipe_id", recipeID, "error", err)
		return err
	}
	recipeLog.Info("共享配方成功", "recipe_id", recipeID, "scope", req.Scope, "scope_ref_id", req.ScopeRefID)
	return nil
}

// ==================== 配方效果统计 ====================

// RecipeStatsResponse 配方效果统计响应
type RecipeStatsResponse struct {
	RecipeID     string                      `json:"recipe_id"`
	RecipeName   string                      `json:"recipe_name"`
	TotalUsage   int                         `json:"total_usage"`
	TotalPlans   int                         `json:"total_plans"`
	AvgScore     float64                     `json:"avg_score"`
	RecentUsages []repository.RecipeUsageRow `json:"recent_usages"`
}

// GetRecipeStats 获取配方效果统计
func (s *RecipeService) GetRecipeStats(ctx context.Context, recipeID string) (*RecipeStatsResponse, error) {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return nil, ErrRecipeNotFound
		}
		return nil, err
	}

	stats, err := repository.GetRecipeStats(ctx, recipeID)
	if err != nil {
		return nil, err
	}

	return &RecipeStatsResponse{
		RecipeID:     recipeID,
		RecipeName:   recipe.Name,
		TotalUsage:   stats.TotalUsage,
		TotalPlans:   stats.TotalPlans,
		AvgScore:     stats.AvgScore,
		RecentUsages: stats.RecentUsages,
	}, nil
}

// ==================== 配方市场排行榜 ====================

// MarketRecipesResponse 配方市场响应
type MarketRecipesResponse struct {
	Recipes []*repository.MarketRecipeItem `json:"recipes"`
	Total   int                            `json:"total"`
}

// ListMarketRecipes 查询配方市场排行榜
func (s *RecipeService) ListMarketRecipes(ctx context.Context, subject string, gradeRange string, sortBy string, limit int, offset int) (*MarketRecipesResponse, error) {
	items, total, err := repository.ListMarketRecipes(ctx, subject, gradeRange, sortBy, limit, offset)
	if err != nil {
		return nil, err
	}
	return &MarketRecipesResponse{Recipes: items, Total: total}, nil
}
