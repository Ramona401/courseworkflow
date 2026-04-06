package services

// recipe_service.go — 备课配方业务逻辑层
//
// Phase 7A 新增：
//   - CRUD + Fork + Share + 学情更新
//   - BuildRecipeContext：将配方转化为AI提示词上下文
//   - RecommendRecipe：根据学科+年级智能推荐组件组合
//   - PreviewContext：预览注入给AI的完整上下文文本
// 迭代2 修改：
//   - CreateRecipe：传递 StagesConfig 到模型
//   - ValidateStageFlow：流程完整性校验
//   - GetFlowPresets：返回预设流程模板

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrRecipeNameRequired    = errors.New("配方名称不能为空")
	ErrRecipeSubjectRequired = errors.New("学科不能为空")
	ErrRecipeGradeRequired   = errors.New("年级不能为空")
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
// 迭代2：新增 StagesConfig 字段传递
func (s *RecipeService) CreateRecipe(ctx context.Context, req *models.CreateRecipeRequest, authorID string) (*models.TeachingRecipe, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrRecipeNameRequired
	}
	if strings.TrimSpace(req.Subject) == "" {
		return nil, ErrRecipeSubjectRequired
	}
	if strings.TrimSpace(req.GradeRange) == "" {
		return nil, ErrRecipeGradeRequired
	}

	// 组件ID列表转JSON
	componentJSON := "[]"
	if len(req.ComponentIDs) > 0 {
		b, _ := json.Marshal(req.ComponentIDs)
		componentJSON = string(b)
	}

	// 迭代1：教案结构
	lessonStructure := req.LessonStructure
	if lessonStructure == "" {
		lessonStructure = "[]"
	}

	// 迭代1：备课模式
	promptMode := req.PromptMode
	if promptMode == "" {
		promptMode = models.PromptModeGuided
	}

	// 迭代2：流程配置
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
		StagesConfig:       stagesConfig,       // 迭代2：流程配置穿透
		LessonStructure:    lessonStructure,     // 迭代1：教案结构
		PromptMode:         promptMode,          // 迭代1：备课模式
	}

	if err := repository.CreateRecipe(ctx, r); err != nil {
		recipeLog.Error("创建配方失败", "error", err)
		return nil, err
	}
	recipeLog.Info("创建配方成功", "recipe_id", r.ID, "name", r.Name, "author", authorID)
	return r, nil
}

// ==================== 查询 ====================

// GetRecipe 获取配方详情（含组件摘要）
func (s *RecipeService) GetRecipe(ctx context.Context, recipeID string) (*models.RecipeDetailResponse, error) {
	r, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return nil, ErrRecipeNotFound
		}
		return nil, err
	}

	// 解析组件ID列表
	var componentIDs []string
	_ = json.Unmarshal([]byte(r.ComponentIDs), &componentIDs)

	// 查询组件摘要
	components, _ := repository.GetRecipeComponentBriefs(ctx, componentIDs)

	// 查询作者名
	authorName := ""
	if user, err := repository.FindUserByID(ctx, r.AuthorID); err == nil {
		authorName = user.DisplayName
	}

	return &models.RecipeDetailResponse{
		TeachingRecipe: *r,
		ComponentCount: len(componentIDs),
		Components:     components,
		AuthorName:     authorName,
		ScopeName:      models.RecipeScopeNameMap[r.Scope],
	}, nil
}

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

// ==================== 构建AI上下文 ====================

// BuildRecipeContext 将配方转化为AI系统提示词上下文
// 这是核心函数：把组件+学情+风格+心得+自定义拼成完整的背景知识文本
func (s *RecipeService) BuildRecipeContext(ctx context.Context, recipe *models.TeachingRecipe) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【备课配方：%s v%d】\n", recipe.Name, recipe.Version))

	// 1. 解析并加载绑定组件的完整内容
	var componentIDs []string
	_ = json.Unmarshal([]byte(recipe.ComponentIDs), &componentIDs)

	if len(componentIDs) > 0 {
		groups, err := repository.GetRecipeComponentContents(ctx, componentIDs)
		if err == nil && len(groups) > 0 {
			for _, g := range groups {
				sb.WriteString(fmt.Sprintf("\n== %s ==\n", g.LibraryName))
				for _, c := range g.Components {
					sb.WriteString(fmt.Sprintf("▸ %s\n", c.DisplayLabel))
					if c.DesignLogic != "" {
						sb.WriteString(fmt.Sprintf("  设计逻辑：%s\n", c.DesignLogic))
					}
					if c.FullGuide != "" {
						sb.WriteString(fmt.Sprintf("  完整指引：%s\n", c.FullGuide))
					}
				}
			}
		}
	}

	// 2. 学情档案
	if strings.TrimSpace(recipe.StudentProfile) != "" {
		sb.WriteString(fmt.Sprintf("\n== 学情档案 ==\n%s\n", recipe.StudentProfile))
	}

	// 3. 教学风格偏好
	if strings.TrimSpace(recipe.TeachingStyle) != "" {
		sb.WriteString(fmt.Sprintf("\n== 教师偏好 ==\n%s\n", recipe.TeachingStyle))
	}

	// 4. 学校要求
	if strings.TrimSpace(recipe.SchoolRequirements) != "" {
		sb.WriteString(fmt.Sprintf("\n== 学校要求 ==\n%s\n", recipe.SchoolRequirements))
	}

	// 5. 备课心得
	if strings.TrimSpace(recipe.CustomNotes) != "" {
		sb.WriteString(fmt.Sprintf("\n== 备课心得 ==\n%s\n", recipe.CustomNotes))
	}

	// 6. 自定义提示词（高级用户直接写的指令）
	if strings.TrimSpace(recipe.CustomPrompt) != "" {
		sb.WriteString(fmt.Sprintf("\n== 自定义指令 ==\n%s\n", recipe.CustomPrompt))
	}

	return sb.String()
}

// PreviewContext 预览配方注入给AI的完整上下文文本
func (s *RecipeService) PreviewContext(ctx context.Context, recipeID string) (*models.RecipeContextPreview, error) {
	r, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrRecipeNotFound) {
			return nil, ErrRecipeNotFound
		}
		return nil, err
	}

	contextText := s.BuildRecipeContext(ctx, r)

	return &models.RecipeContextPreview{
		RecipeID:      r.ID,
		RecipeName:    r.Name,
		ContextText:   contextText,
		TokenEstimate: len(contextText) / 2, // 粗估：中文约2字符/token
	}, nil
}

// ==================== 智能推荐 ====================

// RecommendComponents 根据学科+年级，自动匹配推荐的组件组合
// 返回按组件类型分组的推荐列表，老师可以一键采纳创建配方
func (s *RecipeService) RecommendComponents(ctx context.Context, subject string, gradeRange string) ([]*models.MatchedComponentGroup, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, ErrRecipeSubjectRequired
	}

	// 匹配所有注入模式的已审核组件（silent+recommend+on_demand）
	groups, err := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
		Subject:    subject,
		GradeRange: gradeRange,
		Limit:      3, // 每种类型取前3个
	})
	if err != nil {
		recipeLog.Error("智能推荐匹配失败", "subject", subject, "error", err)
		return nil, err
	}
	if groups == nil {
		groups = []*models.MatchedComponentGroup{}
	}
	return groups, nil
}

// ==================== 迭代2新增：流程校验 ====================

// ValidateStageFlow 校验流程配置的完整性
// 返回校验结果，包含三级提示（info/warning/error）
func (s *RecipeService) ValidateStageFlow(stages []models.StageFlowItem) *models.FlowValidationResult {
	result := &models.FlowValidationResult{Valid: true, Messages: []models.FlowValidationMessage{}}

	// 筛选出已启用的阶段，按order排序
	var enabled []models.StageFlowItem
	for _, st := range stages {
		if st.Enabled {
			enabled = append(enabled, st)
		}
	}

	// 规则1（阻断）：流程不能为空
	if len(enabled) == 0 {
		result.Valid = false
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgError, Code: "flow_empty", Message: "流程不能为空，至少需要保留「教案撰写」和「修订定稿」",
		})
		return result
	}

	// 构建已启用阶段的code集合和顺序映射
	enabledSet := make(map[string]bool)
	orderMap := make(map[string]int)
	for _, st := range enabled {
		enabledSet[st.StageCode] = true
		orderMap[st.StageCode] = st.Order
	}

	// 规则2（阻断）：不可移除的阶段必须保留（write和revise）
	for code, rule := range models.SystemStageFlowRules {
		if !rule.Removable && !enabledSet[code] {
			result.Valid = false
			result.Messages = append(result.Messages, models.FlowValidationMessage{
				Level: models.FlowMsgError, Code: "required_stage_missing",
				Message: fmt.Sprintf("「%s」是必须保留的阶段，不可移除", rule.StageName),
			})
		}
	}

	// 规则3（阻断）：修订定稿必须在最后
	if enabledSet["revise"] {
		reviseOrder := orderMap["revise"]
		for _, st := range enabled {
			if st.StageCode != "revise" && st.Order > reviseOrder {
				result.Valid = false
				result.Messages = append(result.Messages, models.FlowValidationMessage{
					Level: models.FlowMsgError, Code: "revise_not_last",
					Message: "「修订定稿」必须是最后一个阶段",
				})
				break
			}
		}
	}

	// 规则4（阻断）：review必须在write之后
	if enabledSet["review"] && enabledSet["write"] {
		if orderMap["review"] < orderMap["write"] {
			result.Valid = false
			result.Messages = append(result.Messages, models.FlowValidationMessage{
				Level: models.FlowMsgError, Code: "review_before_write",
				Message: "「AI评审」必须在「教案撰写」之后",
			})
		}
	}

	// 规则5（阻断）：阶段不能重复
	codeCount := make(map[string]int)
	for _, st := range stages {
		codeCount[st.StageCode]++
	}
	for code, count := range codeCount {
		if count > 1 {
			result.Valid = false
			rule := models.SystemStageFlowRules[code]
			name := code
			if rule.StageName != "" {
				name = rule.StageName
			}
			result.Messages = append(result.Messages, models.FlowValidationMessage{
				Level: models.FlowMsgError, Code: "stage_duplicate",
				Message: fmt.Sprintf("阶段「%s」重复出现", name),
			})
		}
	}

	// 规则6（警告）：跳过分析阶段
	if !enabledSet["analyze"] {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgWarning, Code: "skip_analyze",
			Message: "跳过「教学分析」后，AI缺少学情和课标依据，教案质量可能下降",
		})
	}

	// 规则7（警告）：跳过设计阶段
	if !enabledSet["design"] && enabledSet["write"] {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgWarning, Code: "skip_design",
			Message: "跳过「教学设计」后，AI将直接撰写教案，缺少系统化的教学设计环节",
		})
	}

	// 规则8（警告）：跳过评审阶段
	if !enabledSet["review"] {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgWarning, Code: "skip_review",
			Message: "跳过「AI评审」后，教案将不经过自动质量检查",
		})
	}

	// 规则9（信息）：极简模式提示
	if len(enabled) <= 2 {
		result.Messages = append(result.Messages, models.FlowValidationMessage{
			Level: models.FlowMsgInfo, Code: "minimal_flow",
			Message: "极简流程适合经验丰富的老师快速出稿，新手建议使用更完整的流程",
		})
	}

	return result
}

// ==================== 迭代2新增：预设流程模板 ====================

// GetFlowPresets 返回预设流程模板列表
// 硬编码4个模板，供前端快速选择
func (s *RecipeService) GetFlowPresets() []*models.FlowPreset {
	return []*models.FlowPreset{
		{
			Key: "full_guided", Name: "完整引导", Description: "5步全开，逐步引导，适合新手或重要课程",
			Duration: "15-25分钟", Icon: "🎓", PromptMode: models.PromptModeGuided,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: true, Order: 1},
				{StageCode: "design", Enabled: true, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: true, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
		{
			Key: "standard", Name: "标准协作", Description: "跳过AI评审，分析+设计+撰写+修订",
			Duration: "10-15分钟", Icon: "🤝", PromptMode: models.PromptModeGuided,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: true, Order: 1},
				{StageCode: "design", Enabled: true, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: false, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
		{
			Key: "quick_draft", Name: "快速出稿", Description: "跳过设计和评审，分析+撰写+修订",
			Duration: "5-10分钟", Icon: "⚡", PromptMode: models.PromptModeEfficient,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: true, Order: 1},
				{StageCode: "design", Enabled: false, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: false, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
		{
			Key: "express", Name: "极速模式", Description: "仅撰写+修订，适合老手快速完成",
			Duration: "3-5分钟", Icon: "🚀", PromptMode: models.PromptModeEfficient,
			Stages: []models.StageFlowItem{
				{StageCode: "analyze", Enabled: false, Order: 1},
				{StageCode: "design", Enabled: false, Order: 2},
				{StageCode: "write", Enabled: true, Order: 3},
				{StageCode: "review", Enabled: false, Order: 4},
				{StageCode: "revise", Enabled: true, Order: 5},
			},
		},
	}
}

// ==================== 迭代6新增：配方效果统计 ====================

// RecipeStatsResponse 配方效果统计响应
type RecipeStatsResponse struct {
	RecipeID     string                       `json:"recipe_id"`
	RecipeName   string                       `json:"recipe_name"`
	TotalUsage   int                          `json:"total_usage"`    // 总使用次数
	TotalPlans   int                          `json:"total_plans"`    // 产出教案总数
	AvgScore     float64                      `json:"avg_score"`      // 教案平均分
	RecentUsages []repository.RecipeUsageRow  `json:"recent_usages"` // 最近使用记录
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

// ==================== 迭代6新增：配方市场排行榜 ====================

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
