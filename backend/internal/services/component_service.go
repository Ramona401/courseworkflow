package services

// 组件库管理业务逻辑层
// 负责组件CRUD、匹配引擎调用、审核、统计更新
//
// Phase5新增：
//   - AutoExtractFromLessonPlan  — 从评审通过的教案自动萃取组件（通道二）
//   - SaveExtractionFromChat     — 保存对话中的萃取记录（通道一）
//   - ConfirmExtractionByID      — 教研组长/骨干确认萃取记录
//   - ListPendingExtractionItems — 获取待审萃取队列
//   - RefreshQualityScore        — 刷新组件质量分
// 迭代4B-2新增：
//   - SmartRecommendComponents   — 画像感知智能推荐（根据teaching_profile加权匹配）
// v82变更：normalizeGradeForMatch 抽取为 utils.NormalizeGradeToNumber 统一工具函数
// v89-2变更：AutoExtractFromLessonPlan的AI调用传入真实TraceContext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 错误常量 ====================

var (
	ErrComponentLibTypeRequired  = errors.New("组件库类型不能为空")
	ErrComponentLibTypeInvalid   = errors.New("无效的组件库类型")
	ErrComponentLabelRequired    = errors.New("组件展示标签不能为空")
	ErrComponentNotFound         = errors.New("组件不存在")
	ErrComponentReviewInvalid    = errors.New("审核决策无效，可选值：approved/rejected")
	ErrExtractionNotFound        = errors.New("萃取记录不存在")
	ErrExtractionDecisionInvalid = errors.New("萃取决策无效，可选值：confirmed/rejected")
)

// ComponentService 组件库管理服务
type ComponentService struct {
	cfg interface{ GetAESKey() string }
}

var compLog = logger.WithModule("component")

// NewComponentService 创建组件服务实例（Phase5：需传入cfg用于AI萃取）
func NewComponentService(cfg interface{ GetAESKey() string }) *ComponentService {
	return &ComponentService{cfg: cfg}
}

// ==================== 组件 CRUD ====================

// CreateComponent 创建组件
func (s *ComponentService) CreateComponent(ctx context.Context, req *models.CreateComponentRequest, createdBy string) (*models.LessonPlanComponent, error) {
	req.DisplayLabel = strings.TrimSpace(req.DisplayLabel)
	if req.LibraryType == "" {
		return nil, ErrComponentLibTypeRequired
	}
	if !models.IsValidLibraryType(req.LibraryType) {
		return nil, ErrComponentLibTypeInvalid
	}
	if req.DisplayLabel == "" {
		return nil, ErrComponentLabelRequired
	}
	if req.InjectionMode != "" && !models.IsValidInjectionMode(req.InjectionMode) {
		return nil, errors.New("无效的注入模式，可选值：silent/recommend/on_demand")
	}
	if req.Scope != "" && !models.IsValidScope(req.Scope) {
		return nil, errors.New("无效的可见范围，可选值：global/region/school/group/personal")
	}

	c := &models.LessonPlanComponent{
		LibraryType:    req.LibraryType,
		Subject:        req.Subject,
		GradeRange:     req.GradeRange,
		Tags:           req.Tags,
		InjectionMode:  req.InjectionMode,
		DisplayLabel:   req.DisplayLabel,
		DesignLogic:    req.DesignLogic,
		ExampleSnippet: req.ExampleSnippet,
		FullGuide:      req.FullGuide,
		Content:        req.Content,
		Source:         "manual",
		Scope:          req.Scope,
		ScopeRefID:     req.ScopeRefID,
		CreatedBy:      &createdBy,
		ReviewStatus:   models.ComponentReviewApproved,
	}

	if err := repository.CreateComponent(ctx, c); err != nil {
		compLog.Error("创建组件失败", "library_type", req.LibraryType, "error", err)
		return nil, err
	}
	compLog.Info("创建组件成功", "component_id", c.ID, "library_type", c.LibraryType)
	return c, nil
}

// GetComponent 获取组件详情
func (s *ComponentService) GetComponent(ctx context.Context, id string) (*models.LessonPlanComponent, error) {
	c, err := repository.GetComponentByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrComponentNotFound) {
			return nil, ErrComponentNotFound
		}
		return nil, err
	}
	return c, nil
}

// ListComponents 获取组件列表
func (s *ComponentService) ListComponents(ctx context.Context, libraryType string, subject string, reviewStatus string, scope string, limit int, offset int) (*models.ComponentListResponse, error) {
	items, total, err := repository.ListComponents(ctx, libraryType, subject, reviewStatus, scope, limit, offset)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.ComponentListItem{}
	}
	return &models.ComponentListResponse{Components: items, Total: total}, nil
}

// UpdateComponent 更新组件
func (s *ComponentService) UpdateComponent(ctx context.Context, id string, req *models.UpdateComponentRequest) error {
	req.DisplayLabel = strings.TrimSpace(req.DisplayLabel)
	if req.DisplayLabel == "" {
		return ErrComponentLabelRequired
	}
	if err := repository.UpdateComponent(ctx, id, req); err != nil {
		if errors.Is(err, repository.ErrComponentNotFound) {
			return ErrComponentNotFound
		}
		compLog.Error("更新组件失败", "component_id", id, "error", err)
		return err
	}
	compLog.Info("更新组件成功", "component_id", id)
	return nil
}

// DeleteComponent 删除组件（软删除）
func (s *ComponentService) DeleteComponent(ctx context.Context, id string) error {
	if err := repository.DeleteComponent(ctx, id); err != nil {
		if errors.Is(err, repository.ErrComponentNotFound) {
			return ErrComponentNotFound
		}
		compLog.Error("删除组件失败", "component_id", id, "error", err)
		return err
	}
	compLog.Info("删除组件成功", "component_id", id)
	return nil
}

// ==================== 审核 ====================

// ReviewComponent 审核组件（教研组长/骨干操作）
func (s *ComponentService) ReviewComponent(ctx context.Context, id string, reviewerID string, req *models.ReviewComponentRequest) error {
	if req.Decision != models.ComponentReviewApproved && req.Decision != models.ComponentReviewRejected {
		return ErrComponentReviewInvalid
	}
	c, err := repository.GetComponentByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrComponentNotFound) {
			return ErrComponentNotFound
		}
		return err
	}
	if c.ReviewStatus != models.ComponentReviewCaptured && c.ReviewStatus != models.ComponentReviewPending {
		return errors.New("该组件不在待审核状态")
	}
	if err := repository.ReviewComponent(ctx, id, reviewerID, req.Decision); err != nil {
		compLog.Error("审核组件失败", "component_id", id, "error", err)
		return err
	}
	compLog.Info("审核组件成功", "component_id", id, "decision", req.Decision)
	return nil
}

// ==================== 匹配引擎 ====================

// MatchComponents 匹配组件（原始匹配，按学科+年级+质量分）
func (s *ComponentService) MatchComponents(ctx context.Context, req *models.MatchComponentsRequest) (*models.MatchComponentsResponse, error) {
	if req.Subject == "" {
		return nil, errors.New("学科不能为空")
	}
	groups, err := repository.MatchComponents(ctx, req)
	if err != nil {
		compLog.Error("匹配组件失败", "subject", req.Subject, "error", err)
		return nil, err
	}
	if groups == nil {
		groups = []*models.MatchedComponentGroup{}
	}
	return &models.MatchComponentsResponse{Groups: groups}, nil
}

// ==================== 迭代4B-2新增：画像感知智能推荐 ====================

// SmartRecommendComponents 根据老师画像+学科+年级智能推荐组件
// 从teaching_profile中提取风格标签，传给SmartMatchComponents进行加权匹配
// 无画像时降级为普通匹配
// v82变更：年级转换改用统一工具函数 utils.NormalizeGradeToNumber
func (s *ComponentService) SmartRecommendComponents(ctx context.Context, subject string, gradeRange string, profile *models.TeachingProfile) ([]*models.MatchedComponentGroup, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, errors.New("学科不能为空")
	}

	// 将中文年级转换为数字格式，确保SQL年级范围匹配能正常工作
	// 例如 "三年级" → "3"、"七年级" → "7"、"高一" → "10"
	normalizedGrade := utils.NormalizeGradeToNumber(gradeRange)
	compLog.Info("智能推荐-年级转换", "original", gradeRange, "normalized", normalizedGrade)

	req := &models.MatchComponentsRequest{
		Subject:    subject,
		GradeRange: normalizedGrade,
		Limit:      5, // 每种类型取前5个
	}

	// 从teaching_profile提取画像标签
	profileTags := buildProfileTags(profile)

	compLog.Info("智能推荐组件",
		"subject", subject,
		"grade", gradeRange,
		"profile_tags_count", len(profileTags),
		"profile_tags", strings.Join(profileTags, ","),
	)

	groups, err := repository.SmartMatchComponents(ctx, req, profileTags)
	if err != nil {
		compLog.Error("智能推荐匹配失败", "subject", subject, "error", err)
		return nil, err
	}
	if groups == nil {
		groups = []*models.MatchedComponentGroup{}
	}
	return groups, nil
}

// buildProfileTags 从teaching_profile中提取匹配标签列表
// 返回如 ["style:growing", "collab:collaborative", "priority:activity_detail", "priority:student_response"]
func buildProfileTags(profile *models.TeachingProfile) []string {
	if profile == nil {
		return nil
	}

	var tags []string

	// 教学风格 → style:xxx 标签
	if profile.TeachingStyle != "" {
		tags = append(tags, "style:"+profile.TeachingStyle)
	}

	// AI协作偏好 → collab:xxx 标签
	if profile.AICollaboration != "" {
		tags = append(tags, "collab:"+profile.AICollaboration)
	}

	// 质量关注点 → priority:xxx 标签（最多取4个，避免过多加权）
	if len(profile.Priorities) > 0 {
		maxPriorities := 4
		for i, p := range profile.Priorities {
			if i >= maxPriorities {
				break
			}
			tags = append(tags, "priority:"+p)
		}
	}

	return tags
}

// ==================== 统计更新 ====================

// RecordComponentUsage 记录组件被使用
func (s *ComponentService) RecordComponentUsage(ctx context.Context, componentID string) error {
	return repository.IncrementComponentUsage(ctx, componentID)
}

// RecordComponentSelect 记录组件被选中，并异步刷新质量分
func (s *ComponentService) RecordComponentSelect(ctx context.Context, componentID string) error {
	if err := repository.IncrementComponentSelect(ctx, componentID); err != nil {
		return err
	}
	go func() {
		bgCtx := context.Background()
		if err := s.RefreshQualityScore(bgCtx, componentID); err != nil {
			compLog.Warn("异步刷新质量分失败", "component_id", componentID, "error", err)
		}
	}()
	return nil
}

// RefreshQualityScore 刷新组件质量分
func (s *ComponentService) RefreshQualityScore(ctx context.Context, componentID string) error {
	avgScore, err := repository.GetComponentLinkedPlanAvgScore(ctx, componentID)
	if err != nil {
		avgScore = 0
		compLog.Warn("获取关联教案均分失败，使用0", "component_id", componentID, "error", err)
	}
	if err := repository.UpdateComponentQualityScore(ctx, componentID, avgScore); err != nil {
		return fmt.Errorf("刷新质量分失败: %w", err)
	}
	compLog.Info("刷新质量分完成", "component_id", componentID, "avg_linked_score", avgScore)
	return nil
}

// ==================== Phase5：通道一——对话萃取 ====================

// SaveExtractionFromChat 保存对话中AI识别的高价值设计片段
func (s *ComponentService) SaveExtractionFromChat(
	ctx context.Context,
	planID string,
	sourceContent string,
	extractionType string,
	displayLabel string,
	designLogic string,
	createdBy string,
) error {
	comp := &models.LessonPlanComponent{
		LibraryType:  extractionType,
		DisplayLabel: displayLabel,
		DesignLogic:  designLogic,
		Source:       "ai_extracted",
		ReviewStatus: models.ComponentReviewCaptured,
		Scope:        models.ScopeGroup,
		CreatedBy:    &createdBy,
	}
	if err := repository.CreateComponent(ctx, comp); err != nil {
		return fmt.Errorf("创建萃取组件失败: %w", err)
	}

	planIDPtr := planID
	ce := &models.ComponentExtraction{
		SourceType:           "conversation",
		SourceLessonPlanID:   &planIDPtr,
		SourceContent:        sourceContent,
		ExtractedComponentID: &comp.ID,
		ExtractionType:       extractionType,
		Status:               "pending",
		CreatedBy:            &createdBy,
	}
	if err := repository.CreateComponentExtraction(ctx, ce); err != nil {
		compLog.Error("创建萃取记录失败", "plan_id", planID, "error", err)
		return err
	}

	compLog.Info("保存对话萃取成功",
		"extraction_id", ce.ID,
		"component_id", comp.ID,
		"type", extractionType,
	)
	return nil
}

// ==================== Phase5：通道二——评审自动萃取 ====================

// AutoExtractFromLessonPlan 从评审通过的高分教案自动萃取可复用设计逻辑
// v89-2变更：传入真实TraceContext，记录planID用于成本追踪
func (s *ComponentService) AutoExtractFromLessonPlan(
	ctx context.Context,
	planID string,
	planContent string,
	subject string,
	grade string,
	reviewerID string,
) error {
	if strings.TrimSpace(planContent) == "" {
		compLog.Warn("教案内容为空，跳过自动萃取", "plan_id", planID)
		return nil
	}

	compLog.Info("开始自动萃取", "plan_id", planID, "subject", subject)

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), "", "", "", "")
	if err != nil {
		return fmt.Errorf("获取AI配置失败: %w", err)
	}

	systemPrompt := `你是一位教育专家，负责从优秀教案中提取可复用的教学设计逻辑片段。
请从给定教案中识别2-4个高价值的教学设计片段，这些片段应当：
1. 有明确的教学意图和逻辑（不只是内容描述）
2. 具有可复用性（其他老师可以参考用于类似课程）
3. 描述具体可操作的教学活动或策略

严格输出以下JSON数组格式（不要有其他文字、不要有markdown代码块）：
[
  {
    "extraction_type": "activity_design",
    "display_label": "🎯 简短标签（20字内，emoji+大白话）",
    "design_logic": "核心设计逻辑（100-200字，说明为什么这样设计）",
    "source_snippet": "从教案摘录的原始片段（50-150字）"
  }
]

extraction_type 只能从以下值中选择：
activity_design, questioning_strategy, pedagogy, assessment_strategy, cross_subject, scenario_material`

	userPrompt := fmt.Sprintf(
		"请从以下%s课教案（%s年级）中提取可复用的教学设计逻辑：\n\n%s",
		subject, grade, planContent,
	)

	// v89-2：构建真实TraceContext，关联教案ID
	pid := planID
	traceCtx := &aiClient.TraceContext{
		SceneCode:    "scanner", // 萃取属于scanner场景范畴
		LessonPlanID: &pid,
	}

	result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		return fmt.Errorf("AI萃取调用失败: %w", err)
	}

	extractions, err := parseAutoExtractionResult(result.Content)
	if err != nil {
		compLog.Warn("解析萃取结果失败", "plan_id", planID, "error", err)
		return err
	}

	validTypes := map[string]bool{
		"activity_design": true, "questioning_strategy": true,
		"pedagogy": true, "assessment_strategy": true,
		"cross_subject": true, "scenario_material": true,
	}

	successCount := 0
	for _, ext := range extractions {
		if !validTypes[ext.ExtractionType] {
			continue
		}
		if strings.TrimSpace(ext.DisplayLabel) == "" || strings.TrimSpace(ext.DesignLogic) == "" {
			continue
		}

		comp := &models.LessonPlanComponent{
			LibraryType:  ext.ExtractionType,
			Subject:      subject,
			GradeRange:   grade,
			DisplayLabel: ext.DisplayLabel,
			DesignLogic:  ext.DesignLogic,
			Source:       "ai_extracted",
			ReviewStatus: models.ComponentReviewPending,
			Scope:        models.ScopeGroup,
			CreatedBy:    &reviewerID,
		}
		if err := repository.CreateComponent(ctx, comp); err != nil {
			compLog.Warn("创建萃取组件失败", "plan_id", planID, "error", err)
			continue
		}

		planIDPtr := planID
		ce := &models.ComponentExtraction{
			SourceType:           "lesson_plan",
			SourceLessonPlanID:   &planIDPtr,
			SourceContent:        ext.SourceSnippet,
			ExtractedComponentID: &comp.ID,
			ExtractionType:       ext.ExtractionType,
			Status:               "pending",
			CreatedBy:            &reviewerID,
		}
		if err := repository.CreateComponentExtraction(ctx, ce); err != nil {
			compLog.Warn("创建萃取记录失败", "plan_id", planID, "error", err)
			continue
		}
		successCount++
	}

	compLog.Info("自动萃取完成", "plan_id", planID, "found", len(extractions), "saved", successCount)
	return nil
}

// ==================== Phase5：萃取确认 ====================

// ConfirmExtractionByID 确认或拒绝萃取记录
func (s *ComponentService) ConfirmExtractionByID(
	ctx context.Context,
	extractionID string,
	confirmerID string,
	decision string,
) error {
	if decision != "confirmed" && decision != "rejected" {
		return ErrExtractionDecisionInvalid
	}

	if err := repository.ConfirmExtraction(ctx, extractionID, confirmerID, decision); err != nil {
		if errors.Is(err, repository.ErrExtractionNotFound) {
			return ErrExtractionNotFound
		}
		return fmt.Errorf("更新萃取状态失败: %w", err)
	}

	ce, err := repository.GetExtractionByID(ctx, extractionID)
	if err == nil && ce.ExtractedComponentID != nil {
		componentDecision := models.ComponentReviewApproved
		if decision == "rejected" {
			componentDecision = models.ComponentReviewRejected
		}
		_ = repository.ReviewComponent(ctx, *ce.ExtractedComponentID, confirmerID, componentDecision)
		compLog.Info("萃取确认，同步更新组件状态",
			"extraction_id", extractionID,
			"component_id", *ce.ExtractedComponentID,
			"decision", decision,
		)
	}
	return nil
}

// ListPendingExtractionItems 获取待审萃取列表
func (s *ComponentService) ListPendingExtractionItems(ctx context.Context, limit int) (*models.ExtractionListResponse, error) {
	extractions, err := repository.ListPendingExtractions(ctx, limit)
	if err != nil {
		return nil, err
	}

	var items []*models.ExtractionListItem
	for _, ce := range extractions {
		item := &models.ExtractionListItem{
			ID:             ce.ID,
			SourceType:     ce.SourceType,
			SourceContent:  ce.SourceContent,
			ExtractionType: ce.ExtractionType,
			LibraryName:    models.LibraryTypeNameMap[ce.ExtractionType],
			Status:         ce.Status,
			CreatedAt:      ce.CreatedAt.Format(time.RFC3339),
		}

		if ce.SourceLessonPlanID != nil {
			if lp, err := repository.GetLessonPlanByID(ctx, *ce.SourceLessonPlanID); err == nil {
				item.PlanTitle = lp.Title
			}
		}

		if ce.CreatedBy != nil {
			if user, err := repository.FindUserByID(ctx, *ce.CreatedBy); err == nil {
				item.CreatedByName = user.DisplayName
			}
		}

		items = append(items, item)
	}

	if items == nil {
		items = []*models.ExtractionListItem{}
	}
	return &models.ExtractionListResponse{Extractions: items, Total: len(items)}, nil
}

// ==================== 内部辅助 ====================

// autoExtractionItem AI萃取返回的单条记录
type autoExtractionItem struct {
	ExtractionType string `json:"extraction_type"`
	DisplayLabel   string `json:"display_label"`
	DesignLogic    string `json:"design_logic"`
	SourceSnippet  string `json:"source_snippet"`
}

// parseAutoExtractionResult 解析AI自动萃取结果（JSON数组）
func parseAutoExtractionResult(content string) ([]autoExtractionItem, error) {
	jsonStr, ok := aiClient.ExtractJSON(content)
	if !ok {
		jsonStr = strings.TrimSpace(content)
	}

	if !strings.HasPrefix(jsonStr, "[") {
		jsonStr = "[" + jsonStr + "]"
	}

	var items []autoExtractionItem
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}
	return items, nil
}
