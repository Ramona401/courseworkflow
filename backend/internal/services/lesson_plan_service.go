package services

// lesson_plan_service.go — 教案管理基础业务
//
// 本文件承载：
//   - 教案普通创建入口；
//   - 教案详情、列表、更新和删除；
//   - 提示词模板管理；
//   - 公共错误和基础辅助方法。
//
// 教案状态流转与课件开发位于lesson_plan_service_actions.go。
// Fork教育域继承硬闸位于lesson_plan_fork_service.go。
// 普通创建教育域硬闸位于lesson_plan_creation_service.go。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrLPTitleRequired     = errors.New("教案标题不能为空")
	ErrLPSubjectRequired   = errors.New("学科不能为空")
	ErrLPGradeRequired     = errors.New("年级不能为空")
	ErrLPTopicRequired     = errors.New("课题不能为空")
	ErrLPNotFound          = errors.New("教案不存在")
	ErrLPNotAuthor         = errors.New("只有作者可以操作此教案")
	ErrLPCannotEdit        = errors.New("当前状态不允许编辑")
	ErrLPCannotSubmit      = errors.New("当前状态不允许提交评审")
	ErrLPCannotDevelop     = errors.New("当前状态不允许进入课件开发")
	ErrLPAlreadyDeveloping = errors.New("教案已在课件开发中")
	ErrLPGroupRequired     = errors.New("提交评审需要指定教研组")

	// ErrLPContentEmpty 表示教案正文为空，禁止进入发布或评审流转。
	ErrLPContentEmpty = errors.New(
		"教案正文为空，请先在备课工坊生成完整教案正文后再操作",
	)

	// ErrLPNotPublisher 表示共享发布调用者既不是作者也不是管理员。
	ErrLPNotPublisher = errors.New(
		"只有作者本人或系统管理员可以共享发布此教案",
	)

	// ErrLPForkNotAllowed 表示来源教案状态不允许被复制。
	ErrLPForkNotAllowed = errors.New(
		"只能Fork已共享发布或评审通过的教案",
	)

	// ErrLPForkEducationDomainMismatch 表示调用者教学域与来源资源域不同。
	ErrLPForkEducationDomainMismatch = errors.New(
		"不能跨教育域Fork教案",
	)

	ErrTemplateNotFound     = errors.New("提示词模板不存在")
	ErrTemplateLevelInvalid = errors.New("无效的模板层级")
	ErrTemplateNameRequired = errors.New("模板名称不能为空")
)

// LessonPlanService 教案管理服务。
type LessonPlanService struct {
	compService *ComponentService
}

var lpLog = logger.WithModule("lesson_plan")

// strPtr 字符串指针辅助函数，空字符串返回nil。
func strPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// NewLessonPlanService 创建教案服务实例。
func NewLessonPlanService(
	compService *ComponentService,
) *LessonPlanService {
	return &LessonPlanService{
		compService: compService,
	}
}

// ==================== 教案 CRUD ====================

// CreateLessonPlan 创建普通教案。
//
// 普通创建必须经过教育域严格解析器和显式写域Repository。
func (s *LessonPlanService) CreateLessonPlan(
	ctx context.Context,
	req *models.CreateLessonPlanRequest,
	authorID string,
) (*models.LessonPlan, error) {
	return s.createLessonPlanWithEducationDomainGate(
		ctx,
		req,
		authorID,
	)
}

// GetLessonPlan 获取教案详情。
func (s *LessonPlanService) GetLessonPlan(
	ctx context.Context,
	id string,
) (*models.LessonPlanDetailResponse, error) {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}

	_ = repository.IncrementLessonPlanView(
		ctx,
		id,
	)

	authorName := ""
	if author, authorErr := repository.FindUserByID(
		ctx,
		lessonPlan.AuthorID,
	); authorErr == nil {
		authorName = author.DisplayName
	}

	groupName := ""
	if lessonPlan.GroupID != nil {
		if group, groupErr :=
			repository.GetTeachingGroupByID(
				ctx,
				*lessonPlan.GroupID,
			); groupErr == nil {
			groupName = group.Name
		}
	}

	reviews, reviewErr :=
		repository.ListLessonPlanReviews(
			ctx,
			id,
		)
	if reviewErr != nil {
		reviews = []*models.LessonPlanReviewItem{}
	}

	recipeName := ""
	if lessonPlan.RecipeID != nil &&
		*lessonPlan.RecipeID != "" {
		if recipe, recipeErr :=
			repository.GetRecipeByID(
				ctx,
				*lessonPlan.RecipeID,
			); recipeErr == nil {
			recipeName = recipe.Name
		}
	}

	var linkedPipelineID *string
	linkedPipeline, pipelineErr :=
		repository.GetPipelineByLessonPlanID(
			id,
		)
	if pipelineErr == nil &&
		linkedPipeline != nil {
		linkedPipelineID = &linkedPipeline.ID
	}

	interactionCounts, _ :=
		repository.GetInteractionCounts(
			ctx,
			id,
			lessonPlan.AuthorID,
		)

	likeCount := 0
	favoriteCount := 0
	if interactionCounts != nil {
		likeCount = interactionCounts.LikeCount
		favoriteCount =
			interactionCounts.FavoriteCount
	}

	return &models.LessonPlanDetailResponse{
		ID: lessonPlan.ID,

		Title:           lessonPlan.Title,
		Subject:         lessonPlan.Subject,
		Grade:           lessonPlan.Grade,
		Topic:           lessonPlan.Topic,
		DurationMinutes: lessonPlan.DurationMinutes,

		ContentMarkdown:   lessonPlan.ContentMarkdown,
		ContentStructured: lessonPlan.ContentStructured,
		GenerationConfig:  lessonPlan.GenerationConfig,
		MatchedComponents: lessonPlan.MatchedComponents,

		AIReviewScore:   lessonPlan.AIReviewScore,
		AIReviewResult:  lessonPlan.AIReviewResult,
		AIReviewHistory: lessonPlan.AIReviewHistory,

		Status:     lessonPlan.Status,
		StatusName: models.LPStatusNameMap[lessonPlan.Status],
		Visibility: lessonPlan.Visibility,

		AuthorID:   lessonPlan.AuthorID,
		AuthorName: authorName,
		GroupID:    lessonPlan.GroupID,
		GroupName:  groupName,
		SchoolID:   lessonPlan.SchoolID,

		ForkedFrom: lessonPlan.ForkedFrom,
		ForkCount:  lessonPlan.ForkCount,
		ViewCount:  lessonPlan.ViewCount,
		UseCount:   lessonPlan.UseCount,
		Version:    lessonPlan.Version,

		RecipeID:   lessonPlan.RecipeID,
		RecipeName: recipeName,

		LessonIndex:          lessonPlan.LessonIndex,
		IdxCognitiveLevel:    lessonPlan.IdxCognitiveLevel,
		IdxPedagogyIntensity: lessonPlan.IdxPedagogyIntensity,
		IdxStructureType:     lessonPlan.IdxStructureType,
		IdxQualityLevel:      lessonPlan.IdxQualityLevel,

		CurrentStage: lessonPlan.CurrentStage,
		StageConfig:  lessonPlan.StageConfig,

		LikeCount:        likeCount,
		FavoriteCount:    favoriteCount,
		Reviews:          reviews,
		LinkedPipelineID: linkedPipelineID,

		CreatedAt: lessonPlan.CreatedAt,
		UpdatedAt: lessonPlan.UpdatedAt,
	}, nil
}

// ListLessonPlans 获取教案列表。
//
// DataScope缺失时按非管理员和空白名单处理，绝不放大权限。
func (s *LessonPlanService) ListLessonPlans(
	ctx context.Context,
	callerID string,
	authorID string,
	groupID string,
	status string,
	subject string,
	grade string,
	limit int,
	offset int,
	qualityLevel int,
	structureType int,
	cognitiveLevel int,
	pedagogyIntensity int,
	scope *DataScope,
) (*models.LessonPlanListResponse, error) {
	var scopeUserIDs []string
	scopeIsAdmin := false

	if scope != nil {
		scopeIsAdmin = scope.IsAdmin
		scopeUserIDs = scope.UserIDs
	} else {
		scopeUserIDs = []string{}
	}

	items, total, err := repository.ListLessonPlans(
		ctx,
		callerID,
		authorID,
		groupID,
		status,
		subject,
		grade,
		limit,
		offset,
		qualityLevel,
		structureType,
		cognitiveLevel,
		pedagogyIntensity,
		scopeUserIDs,
		scopeIsAdmin,
	)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []*models.LessonPlanListItem{}
	}

	return &models.LessonPlanListResponse{
		LessonPlans: items,
		Total:       total,
	}, nil
}

// UpdateLessonPlan 更新教案内容。
func (s *LessonPlanService) UpdateLessonPlan(
	ctx context.Context,
	id string,
	callerID string,
	req *models.UpdateLessonPlanRequest,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return ErrLPNotFound
		}
		return err
	}

	if lessonPlan.AuthorID != callerID {
		return ErrLPNotAuthor
	}

	editableStatuses := map[string]bool{
		models.LPStatusDraft:             true,
		models.LPStatusPublishedPersonal: true,
		models.LPStatusRevision:          true,
		models.LPStatusApproved:          true,
		models.LPStatusPublishedShared:   true,
	}
	if !editableStatuses[lessonPlan.Status] {
		return ErrLPCannotEdit
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = lessonPlan.Title
	}

	duration := req.DurationMinutes
	if duration <= 0 {
		duration = lessonPlan.DurationMinutes
	}

	if err := repository.UpdateLessonPlanContent(
		ctx,
		id,
		title,
		req.ContentMarkdown,
		"",
		duration,
		models.LessonPlanVersionMeta{
			ChangeSource:  models.LPVersionSourceManual,
			ChangedBy:     &callerID,
			ChangeSummary: "老师手动编辑教案正文",
		},
	); err != nil {
		lpLog.Error(
			"更新教案失败",
			"plan_id", id,
			"error", err,
		)
		return err
	}

	return nil
}

// DeleteLessonPlan 删除教案。
func (s *LessonPlanService) DeleteLessonPlan(
	ctx context.Context,
	id string,
	callerID string,
) error {
	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		) {
			return ErrLPNotFound
		}
		return err
	}

	if lessonPlan.AuthorID != callerID {
		return ErrLPNotAuthor
	}

	if err := repository.DeleteLessonPlan(
		ctx,
		id,
	); err != nil {
		lpLog.Error(
			"删除教案失败",
			"plan_id", id,
			"error", err,
		)
		return err
	}

	lpLog.Info(
		"删除教案成功",
		"plan_id", id,
	)
	return nil
}

// ==================== 提示词模板管理 ====================

// CreatePromptTemplate 创建提示词模板。
func (s *LessonPlanService) CreatePromptTemplate(
	ctx context.Context,
	req *models.CreatePromptTemplateRequest,
	createdBy string,
) (*models.PromptTemplate, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, ErrTemplateNameRequired
	}
	if !models.IsValidTemplateLevel(req.Level) {
		return nil, ErrTemplateLevelInvalid
	}

	promptTemplate := &models.PromptTemplate{
		Name:             req.Name,
		Description:      strPtr(req.Description),
		Level:            req.Level,
		OwnerID:          req.OwnerID,
		ParentTemplateID: req.ParentTemplateID,

		SystemPrompt:       strPtr(req.SystemPrompt),
		ContextRules:       strPtr(req.ContextRules),
		GenerationRules:    strPtr(req.GenerationRules),
		ReviewRules:        strPtr(req.ReviewRules),
		OutputFormat:       strPtr(req.OutputFormat),
		CustomInstructions: strPtr(req.CustomInstructions),

		Subject:    req.Subject,
		GradeRange: req.GradeRange,
		IsDefault:  req.IsDefault,
		CreatedBy:  &createdBy,
	}

	if err := repository.CreatePromptTemplate(
		ctx,
		promptTemplate,
	); err != nil {
		lpLog.Error(
			"创建模板失败",
			"name", req.Name,
			"error", err,
		)
		return nil, err
	}

	lpLog.Info(
		"创建模板成功",
		"template_id", promptTemplate.ID,
		"name", promptTemplate.Name,
	)
	return promptTemplate, nil
}

// ListPromptTemplates 获取模板列表。
func (s *LessonPlanService) ListPromptTemplates(
	ctx context.Context,
	level string,
	ownerID string,
) (*models.PromptTemplateListResponse, error) {
	items, err := repository.ListPromptTemplates(
		ctx,
		level,
		ownerID,
	)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []*models.PromptTemplateListItem{}
	}

	return &models.PromptTemplateListResponse{
		Templates: items,
		Total:     len(items),
	}, nil
}

// GetPromptTemplate 获取模板详情。
func (s *LessonPlanService) GetPromptTemplate(
	ctx context.Context,
	id string,
) (*models.PromptTemplate, error) {
	promptTemplate, err :=
		repository.GetPromptTemplateByID(
			ctx,
			id,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrTemplateNotFound,
		) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	return promptTemplate, nil
}

// UpdatePromptTemplate 更新模板。
func (s *LessonPlanService) UpdatePromptTemplate(
	ctx context.Context,
	id string,
	req *models.UpdatePromptTemplateRequest,
) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return ErrTemplateNameRequired
	}

	if err := repository.UpdatePromptTemplate(
		ctx,
		id,
		req,
	); err != nil {
		if errors.Is(
			err,
			repository.ErrTemplateNotFound,
		) {
			return ErrTemplateNotFound
		}
		return err
	}

	return nil
}

// ResolvePromptTemplate 解析模板继承链。
func (s *LessonPlanService) ResolvePromptTemplate(
	ctx context.Context,
	templateID string,
) (*models.ResolvedPromptTemplate, error) {
	return repository.ResolvePromptTemplateChain(
		ctx,
		templateID,
	)
}

// mapNotFoundErr 将Repository未找到错误映射为Service错误。
func (s *LessonPlanService) mapNotFoundErr(
	err error,
) error {
	if errors.Is(
		err,
		repository.ErrLessonPlanNotFound,
	) {
		return ErrLPNotFound
	}
	return err
}
