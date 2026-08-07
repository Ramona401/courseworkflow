package services

// lesson_plan_shared_read_service.go — 教案列表与详情的共享可见性接入
//
// 教案详情除正文和阶段状态外，还必须读取正式精确课程大纲快照，
// 供备课恢复页面识别当前挂载ID、出版社、册次和学制。
// publisher-only存量教案会返回出版社，但course_outline_id仍为空。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// GetLessonPlan 获取教案详情。
func (s *LessonPlanService) GetLessonPlan(
	ctx context.Context,
	id string,
	callerIDs ...string,
) (*models.LessonPlanDetailResponse, error) {
	callerID := ""
	if len(callerIDs) > 0 {
		callerID = strings.TrimSpace(
			callerIDs[0],
		)
	}

	lessonPlan, err := s.loadLessonPlanForRead(
		ctx,
		id,
		callerID,
		nil,
	)
	if err != nil {
		return nil, err
	}

	outlineSnapshot, err :=
		repository.GetLessonPlanCourseOutlineSnapshot(
			ctx,
			id,
		)
	if err != nil {
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
		reviews =
			[]*models.LessonPlanReviewItem{}
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
		linkedPipelineID =
			&linkedPipeline.ID
	}

	interactionCounts, _ :=
		repository.GetInteractionCounts(
			ctx,
			id,
			callerID,
		)

	likeCount := 0
	favoriteCount := 0
	if interactionCounts != nil {
		likeCount =
			interactionCounts.LikeCount
		favoriteCount =
			interactionCounts.FavoriteCount
	}

	var (
		courseOutlineID        *string
		courseOutlinePublisher *string
		courseOutlineVolume    *string
		schoolSystem           *string
	)
	if outlineSnapshot != nil {
		courseOutlineID =
			outlineSnapshot.CourseOutlineID
		courseOutlinePublisher =
			outlineSnapshot.CourseOutlinePublisher
		courseOutlineVolume =
			outlineSnapshot.CourseOutlineVolume
		schoolSystem =
			outlineSnapshot.SchoolSystem
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

		CourseOutlineID:        courseOutlineID,
		CourseOutlinePublisher: courseOutlinePublisher,
		CourseOutlineVolume:    courseOutlineVolume,
		SchoolSystem:           schoolSystem,

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

	sharedOnly :=
		strings.TrimSpace(authorID) == "" &&
			(status ==
				models.LPStatusPublishedShared ||
				status ==
					models.LPStatusApproved)

	sharedAccess,
		sharedAccessErr :=
		resolveLessonPlanSharedAccessContext(
			ctx,
			callerID,
			scope,
		)
	if sharedAccessErr != nil {
		if errors.Is(
			sharedAccessErr,
			errLPSharedAccessUnavailable,
		) {
			if sharedOnly {
				return &models.LessonPlanListResponse{
					LessonPlans: []*models.LessonPlanListItem{},
					Total:       0,
				}, nil
			}
			sharedAccess = nil
		} else {
			return nil, sharedAccessErr
		}
	}

	var repositorySharedAccess *repository.LessonPlanListSharedAccess
	if sharedAccess != nil {
		repositorySharedAccess =
			&repository.LessonPlanListSharedAccess{
				SharedAuthorIDs: sharedAccess.VisibleAuthorIDs,
				EducationDomain: sharedAccess.CurrentEducationDomain,
				SharedOnly:      sharedOnly,
			}
	} else if sharedOnly {
		return &models.LessonPlanListResponse{
			LessonPlans: []*models.LessonPlanListItem{},
			Total:       0,
		}, nil
	}

	items, total, err :=
		repository.ListLessonPlansWithSharedAccess(
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
			repositorySharedAccess,
		)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items =
			[]*models.LessonPlanListItem{}
	}

	return &models.LessonPlanListResponse{
		LessonPlans: items,
		Total:       total,
	}, nil
}
