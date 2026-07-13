package services

// workshop_stage_components.go — 阶段组件推荐 + 自定义阶段CRUD
//
// v76拆分自 workshop_stage_service.go
// 包含：
//   - GetRecommendedComponents：获取阶段推荐组件列表（迭代12）
//   - getSelectedComponentIDsFromOutput：读取用户选中组件ID
//   - getRecipeComponentsForStage / getAutoMatchedComponentsForStage：组件查询
//   - CreateCustomStage / UpdateCustomStage / DeleteCustomStage / ListCustomStages：自定义阶段CRUD
//
// BugFix（自定义阶段提示词无法保存）：
//   CreateCustomStage / ListCustomStages 构造 CustomStageResponse 时，
//   补填 SystemPrompt / PromptVariants / OutputFormat 三个全文字段（数据来自 workshop_stages 全量列）。
//   此前只回 HasPrompt 布尔值，前端编辑已有阶段拿不到提示词原文，保存时空串覆盖数据库。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// stageComponentTypeMap 阶段→组件类型映射
// revise阶段无组件注入，不在此映射中
var stageComponentTypeMap = map[string][]string{
	"analyze": {"curriculum_standard", "knowledge_graph", "student_profile"},
	"design":  {"pedagogy", "activity_design", "questioning_strategy", "assessment_strategy", "cross_subject"},
	"write":   {"teaching_tool", "scenario_material"},
	"review":  {"quality_rubric", "review_rubric", "design_defect"},
}

// ==================== 获取阶段推荐组件（v76/迭代12新增）====================

// GetRecommendedComponents 获取指定阶段的推荐教学组件列表
//
// 逻辑：
//  1. 验证教案所有权
//  2. 确认目标阶段有组件类型映射（revise阶段无组件，返回空）
//  3. 优先从配方组件中筛选当前阶段类型的组件
//  4. 补充：从组件库自动匹配（按学科+年级+阶段类型），去重后合并
//  5. 返回组件列表，供前端弹窗展示
func (s *WorkshopStageService) GetRecommendedComponents(
	ctx context.Context,
	lessonPlanID string,
	stageCode string,
	callerID string,
) (*models.StageRecommendedComponentsResponse, error) {
	lp, err := repository.GetLessonPlanByID(
		ctx,
		lessonPlanID,
	)
	if err != nil {
		return nil, ErrLPGenPlanNotFound
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	compTypes, hasMapping := stageComponentTypeMap[stageCode]
	if !hasMapping || len(compTypes) == 0 {
		return &models.StageRecommendedComponentsResponse{
			StageCode:  stageCode,
			StageName:  stageCodeToName(stageCode),
			Components: []*models.RecommendedComponentItem{},
		}, nil
	}

	seenIDs := make(map[string]bool)
	var allItems []*models.RecommendedComponentItem

	// 只有严格匹配当前学科和具体年级的配方，
	// 才能向阶段推荐列表提供配方组件。
	//
	// 存量错误关联不修改数据库，只在本轮运行时忽略。
	if lp.RecipeID != nil &&
		strings.TrimSpace(*lp.RecipeID) != "" {
		recipe, recipeErr := loadRecipeForLesson(
			ctx,
			*lp.RecipeID,
			lp.Subject,
			lp.Grade,
		)
		if recipeErr != nil {
			wsLog.Info(
				"阶段组件推荐忽略不适用配方",
				"plan_id", lessonPlanID,
				"stage", stageCode,
				"recipe_id", *lp.RecipeID,
				"subject", lp.Subject,
				"grade", lp.Grade,
			)
		} else {
			recipeItems := s.getRecipeComponentsForStage(
				ctx,
				recipe,
				compTypes,
			)
			for _, item := range recipeItems {
				if seenIDs[item.ID] {
					continue
				}

				seenIDs[item.ID] = true
				item.Source = "recipe"
				allItems = append(
					allItems,
					item,
				)
			}
		}
	}

	// 自动组件仍按当前教案的学科和具体年级匹配。
	autoItems := s.getAutoMatchedComponentsForStage(
		ctx,
		compTypes,
		lp.Subject,
		lp.Grade,
	)
	for _, item := range autoItems {
		if seenIDs[item.ID] {
			continue
		}

		seenIDs[item.ID] = true
		item.Source = "auto"
		allItems = append(
			allItems,
			item,
		)
	}

	if allItems == nil {
		allItems = []*models.RecommendedComponentItem{}
	}

	wsLog.Info(
		"获取阶段推荐组件",
		"plan_id", lessonPlanID,
		"stage", stageCode,
		"total_components", len(allItems),
	)

	return &models.StageRecommendedComponentsResponse{
		StageCode:  stageCode,
		StageName:  stageCodeToName(stageCode),
		Components: allItems,
	}, nil
}

// getRecipeComponentsForStage 从配方已选组件中筛选匹配当前阶段类型的组件
func (s *WorkshopStageService) getRecipeComponentsForStage(ctx context.Context, recipe *models.TeachingRecipe, stageTypes []string) []*models.RecommendedComponentItem {
	var allComponentIDs []string
	if err := json.Unmarshal([]byte(recipe.ComponentIDs), &allComponentIDs); err != nil || len(allComponentIDs) == 0 {
		return nil
	}
	typeSet := make(map[string]bool)
	for _, t := range stageTypes {
		typeSet[t] = true
	}
	groups, err := repository.GetRecipeComponentContents(ctx, allComponentIDs)
	if err != nil || len(groups) == 0 {
		return nil
	}
	var items []*models.RecommendedComponentItem
	for _, g := range groups {
		if !typeSet[g.LibraryType] {
			continue
		}
		for _, c := range g.Components {
			items = append(items, &models.RecommendedComponentItem{
				ID: c.ID, LibraryType: g.LibraryType, LibraryName: g.LibraryName,
				DisplayLabel: c.DisplayLabel, DesignLogic: c.DesignLogic,
				FullGuide: c.FullGuide, ExampleSnippet: c.ExampleSnippet,
				QualityScore: c.QualityScore, Source: "recipe",
			})
		}
	}
	return items
}

// getAutoMatchedComponentsForStage 从组件库自动匹配阶段组件
func (s *WorkshopStageService) getAutoMatchedComponentsForStage(ctx context.Context, stageTypes []string, subject string, grade string) []*models.RecommendedComponentItem {
	normalizedGrade := utils.NormalizeGradeToNumber(grade)
	wsLog.Info("自动匹配组件参数", "subject", subject, "grade", grade, "normalizedGrade", normalizedGrade, "stageTypes", stageTypes)
	groups, err := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
		Subject: subject, GradeRange: normalizedGrade, LibraryTypes: stageTypes, Limit: 3,
	})
	if err != nil {
		wsLog.Warn("自动匹配组件查询失败", "error", err)
		return nil
	}
	wsLog.Info("自动匹配组件结果", "groups_count", len(groups))
	if len(groups) == 0 {
		return nil
	}
	var items []*models.RecommendedComponentItem
	for _, g := range groups {
		for _, c := range g.Components {
			items = append(items, &models.RecommendedComponentItem{
				ID: c.ID, LibraryType: g.LibraryType, LibraryName: g.LibraryName,
				DisplayLabel: c.DisplayLabel, DesignLogic: c.DesignLogic,
				FullGuide: c.FullGuide, ExampleSnippet: c.ExampleSnippet,
				QualityScore: c.QualityScore, Source: "auto",
			})
		}
	}
	return items
}

// getSelectedComponentIDsFromOutput 从阶段产出物中读取用户选中的组件ID列表
func (s *WorkshopStageService) getSelectedComponentIDsFromOutput(ctx context.Context, lessonPlanID string, stageCode string) []string {
	out, err := repository.GetStageOutput(ctx, lessonPlanID, stageCode)
	if err != nil {
		return nil
	}
	if out.StructuredOutput == "" || out.StructuredOutput == "{}" {
		return nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(out.StructuredOutput), &data); err != nil {
		return nil
	}
	idsRaw, ok := data["selected_component_ids"]
	if !ok {
		return nil
	}
	idsArr, ok := idsRaw.([]interface{})
	if !ok {
		return nil
	}
	var ids []string
	for _, v := range idsArr {
		if s, ok := v.(string); ok && s != "" {
			ids = append(ids, s)
		}
	}
	return ids
}

// ==================== 自定义阶段 CRUD ====================

func (s *WorkshopStageService) CreateCustomStage(ctx context.Context, recipeID string, req *models.CreateCustomStageRequest, callerID string) (*models.CustomStageResponse, error) {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return nil, ErrRecipeNotFound
	}
	if recipe.AuthorID != callerID {
		return nil, ErrRecipeUnauthorized
	}
	if strings.TrimSpace(req.StageCode) == "" {
		return nil, errors.New("阶段代码不能为空")
	}
	if strings.TrimSpace(req.StageName) == "" {
		return nil, errors.New("阶段名称不能为空")
	}
	if strings.TrimSpace(req.AIRole) == "" {
		return nil, errors.New("AI角色不能为空")
	}
	for _, ch := range req.StageCode {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return nil, errors.New("阶段代码仅允许小写英文、数字和下划线")
		}
	}
	count, err := repository.CountRecipeStages(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if count >= 10 {
		return nil, ErrCustomStageLimit
	}
	stage, err := repository.CreateRecipeStage(ctx, recipeID, req)
	if err != nil {
		if errors.Is(err, repository.ErrStageCodeConflict) {
			return nil, errors.New("阶段代码已存在，请使用其他代码")
		}
		return nil, err
	}
	wsLog.Info("创建自定义阶段成功", "recipe_id", recipeID, "stage_code", stage.StageCode, "stage_name", stage.StageName)
	// BugFix：回填提示词全文三字段，前端创建后若立刻编辑也能正确回填
	return &models.CustomStageResponse{
		StageCode: stage.StageCode, StageName: stage.StageName, AIRole: stage.AIRole,
		GateMode: stage.GateMode, Skippable: stage.Skippable, HasPrompt: stage.SystemPrompt != "",
		SystemPrompt: stage.SystemPrompt, PromptVariants: stage.PromptVariants, OutputFormat: stage.OutputFormat,
	}, nil
}

func (s *WorkshopStageService) UpdateCustomStage(ctx context.Context, recipeID string, stageCode string, req *models.UpdateCustomStageRequest, callerID string) error {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return ErrRecipeNotFound
	}
	if recipe.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}
	if strings.TrimSpace(req.StageName) == "" {
		return errors.New("阶段名称不能为空")
	}
	if strings.TrimSpace(req.AIRole) == "" {
		return errors.New("AI角色不能为空")
	}
	if err := repository.UpdateRecipeStage(ctx, recipeID, stageCode, req); err != nil {
		return err
	}
	wsLog.Info("更新自定义阶段成功", "recipe_id", recipeID, "stage_code", stageCode)
	return nil
}

func (s *WorkshopStageService) DeleteCustomStage(ctx context.Context, recipeID string, stageCode string, callerID string) error {
	recipe, err := repository.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return ErrRecipeNotFound
	}
	if recipe.AuthorID != callerID {
		return ErrRecipeUnauthorized
	}
	if err := repository.DeleteRecipeStage(ctx, recipeID, stageCode); err != nil {
		return err
	}
	wsLog.Info("删除自定义阶段成功", "recipe_id", recipeID, "stage_code", stageCode)
	return nil
}

func (s *WorkshopStageService) ListCustomStages(ctx context.Context, recipeID string) ([]*models.CustomStageResponse, error) {
	stages, err := repository.GetRecipeStages(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	var items []*models.CustomStageResponse
	for _, st := range stages {
		// BugFix：回填提示词全文三字段，供前端编辑弹窗正确回填，避免空串覆盖数据库已有提示词
		items = append(items, &models.CustomStageResponse{
			StageCode: st.StageCode, StageName: st.StageName, AIRole: st.AIRole,
			GateMode: st.GateMode, Skippable: st.Skippable, HasPrompt: st.SystemPrompt != "",
			SystemPrompt: st.SystemPrompt, PromptVariants: st.PromptVariants, OutputFormat: st.OutputFormat,
		})
	}
	if items == nil {
		items = []*models.CustomStageResponse{}
	}
	return items, nil
}
