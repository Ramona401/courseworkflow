package services

// lesson_plan_gen_start.go
//
// 本文件专门承载“开始对话 / 专家模式”共用的会话创建编排。
//
// 创建硬闸：
//   - 实时解析唯一具体教学教育域；
//   - 课本ID在INSERT前统一校验；
//   - 精确课程大纲ID在INSERT前校验作者可见性、同域、同学科和具体年级；
//   - 教案与精确大纲ID在同一INSERT事务中原子写入；
//   - 阶段初始化或开场消息失败时同步补偿清理。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

type lessonPlanConversationCreationDeps struct {
	findUser func(
		ctx context.Context,
		userID string,
	) (*models.User, error)

	resolveEducationDomain func(
		ctx context.Context,
		userID string,
		role string,
	) (string, error)

	createWithEducationDomain func(
		ctx context.Context,
		lp *models.LessonPlan,
		educationDomain string,
	) error

	cleanupIncompleteCreation func(
		ctx context.Context,
		lessonPlanID string,
		authorID string,
	) error
}

func defaultLessonPlanConversationCreationDeps() lessonPlanConversationCreationDeps {
	return lessonPlanConversationCreationDeps{
		findUser: repository.FindUserByID,
		resolveEducationDomain: repository.
			ResolveLessonPlanCreationEducationDomain,
		createWithEducationDomain: repository.
			CreateLessonPlanWithEducationDomain,
		cleanupIncompleteCreation: repository.
			DeleteIncompleteLessonPlanConversationCreation,
	}
}

// StartConversation 创建教案、初始化阶段并写入开场消息。
func (s *LessonPlanGenService) StartConversation(
	ctx context.Context,
	req *models.StartConversationRequest,
	authorID string,
) (
	resultPlan *models.LessonPlan,
	resultOpening *models.ConversationMessage,
	resultErr error,
) {
	if IsGlobalBackgroundDraining() {
		return nil, nil, ErrLPGenServiceDraining
	}
	if req == nil {
		return nil, nil, errors.New("开始备课请求不能为空")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return nil, nil, ErrLPGenSubjectRequired
	}
	if strings.TrimSpace(req.Grade) == "" {
		return nil, nil, ErrLPGenGradeRequired
	}
	if strings.TrimSpace(req.Topic) == "" {
		return nil, nil, ErrLPGenTopicRequired
	}

	duration := req.DurationMinutes
	if duration <= 0 {
		duration = 45
	}

	deps := defaultLessonPlanConversationCreationDeps()
	recipeSelectionMode := normalizeStartRecipeSelection(req)

	if recipeSelectionMode == models.RecipeSelectionModeSelected &&
		strings.TrimSpace(req.RecipeID) != "" {
		if _, recipeErr := loadRecipeForManualSelection(
			ctx,
			req.RecipeID,
			authorID,
		); recipeErr != nil {
			lpGenLog.Warn(
				"老师选择的配方不存在、未启用或无权使用，拒绝创建会话",
				"author", authorID,
				"subject", req.Subject,
				"grade", req.Grade,
				"recipe_id", req.RecipeID,
				"error", recipeErr,
			)
			return nil, nil, fmt.Errorf(
				"选择的备课配方当前不可用，请重新选择: %w",
				recipeErr,
			)
		}

		lpGenLog.Info(
			"老师手动选择配方已通过授权校验",
			"author", authorID,
			"subject", req.Subject,
			"grade", req.Grade,
			"recipe_id", req.RecipeID,
		)
	}

	if recipeSelectionMode == models.RecipeSelectionModeAuto {
		if resolvedRecipeID := s.ResolveDefaultRecipe(
			ctx,
			authorID,
			req.Subject,
			req.Grade,
		); resolvedRecipeID != "" {
			req.RecipeID = resolvedRecipeID
			lpGenLog.Info(
				"开始备课自动挂载配方",
				"author", authorID,
				"subject", req.Subject,
				"topic", req.Topic,
				"recipe_id", resolvedRecipeID,
				"recipe_mode", recipeSelectionMode,
			)
		}
	} else if recipeSelectionMode == models.RecipeSelectionModeNone {
		req.RecipeID = ""
		lpGenLog.Info(
			"老师明确选择不使用配方",
			"author", authorID,
			"subject", req.Subject,
			"topic", req.Topic,
			"recipe_mode", recipeSelectionMode,
		)
	}

	createdPlan, createErr := prepareConversationLessonPlan(
		ctx,
		req,
		authorID,
		duration,
		deps,
	)
	resultPlan = createdPlan

	cleanupRequired := createdPlan != nil &&
		strings.TrimSpace(createdPlan.ID) != ""

	defer func() {
		if !cleanupRequired {
			return
		}

		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		cleanupErr := deps.cleanupIncompleteCreation(
			cleanupCtx,
			createdPlan.ID,
			authorID,
		)
		if cleanupErr == nil {
			lpGenLog.Warn(
				"会话创建失败后的补偿清理已完成",
				"plan_id", createdPlan.ID,
				"author", authorID,
			)
			return
		}

		lpGenLog.Error(
			"会话创建失败且补偿清理失败",
			"plan_id", createdPlan.ID,
			"author", authorID,
			"creation_error", resultErr,
			"cleanup_error", cleanupErr,
		)

		if resultErr == nil {
			resultErr = fmt.Errorf(
				"会话创建补偿清理失败: %w",
				cleanupErr,
			)
			return
		}

		resultErr = fmt.Errorf(
			"%w；补偿清理失败: %v",
			resultErr,
			cleanupErr,
		)
	}()

	if createErr != nil {
		return nil, nil, createErr
	}

	lpGenLog.Info(
		"开始备课会话已显式写入教育域与精确大纲",
		"plan_id", createdPlan.ID,
		"topic", req.Topic,
		"author", authorID,
		"recipe_id", req.RecipeID,
		"course_outline_id", req.CourseOutlineID,
		"education_domain", createdPlan.EducationDomain,
	)

	recipeStagesConfig := ""
	if strings.TrimSpace(req.RecipeID) != "" {
		recipe, recipeErr := repository.GetRecipeByID(
			ctx,
			req.RecipeID,
		)
		if recipeErr == nil {
			recipeStagesConfig = recipe.StagesConfig
		} else {
			lpGenLog.Warn(
				"读取配方阶段配置失败，使用系统默认阶段",
				"plan_id", createdPlan.ID,
				"recipe_id", req.RecipeID,
				"error", recipeErr,
			)
		}
	}

	snapshots, stageErr := s.stageService.InitStagesForPlan(
		ctx,
		createdPlan.ID,
		recipeStagesConfig,
		req.RecipeID,
	)
	if stageErr != nil {
		lpGenLog.Error(
			"阶段初始化失败",
			"plan_id", createdPlan.ID,
			"education_domain", createdPlan.EducationDomain,
			"error", stageErr,
		)
		return nil, nil, fmt.Errorf(
			"阶段初始化失败: %w",
			stageErr,
		)
	}

	createdPlan.CurrentStage = snapshots[0].StageCode
	configJSON, marshalErr := json.Marshal(snapshots)
	if marshalErr != nil {
		return nil, nil, fmt.Errorf(
			"阶段配置快照序列化失败: %w",
			marshalErr,
		)
	}
	createdPlan.StageConfig = string(configJSON)

	lpGenLog.Info(
		"阶段初始化成功",
		"plan_id", createdPlan.ID,
		"stages_count", len(snapshots),
		"first_stage", snapshots[0].StageCode,
		"education_domain", createdPlan.EducationDomain,
	)

	openingMsg, openingErr := s.genStageOpeningMessage(
		ctx,
		createdPlan,
		snapshots,
		authorID,
	)
	if openingErr != nil {
		lpGenLog.Warn(
			"阶段开场白生成失败，使用默认开场",
			"plan_id", createdPlan.ID,
			"error", openingErr,
		)
		openingMsg = buildDefaultOpeningMessage(req)
	}

	if openingMsg == nil {
		return nil, nil, errors.New("生成开场消息失败：消息为空")
	}

	if openingMsg.Metadata == nil {
		openingMsg.Metadata = make(map[string]interface{})
	}
	openingMsg.Metadata[recipeSelectionModeMetadataKey] = string(
		recipeSelectionMode,
	)

	s.appendUnrecognizedTextbookNotice(
		ctx,
		req,
		openingMsg,
	)

	if appendErr := s.appendMessage(
		ctx,
		createdPlan.ID,
		openingMsg,
	); appendErr != nil {
		lpGenLog.Error(
			"写入开场消息失败",
			"plan_id", createdPlan.ID,
			"error", appendErr,
		)
		return nil, nil, fmt.Errorf(
			"写入开场消息失败: %w",
			appendErr,
		)
	}

	cleanupRequired = false

	GlobalLPSSEHub.Broadcast(createdPlan.ID, models.LPSSEEvent{
		EventType: models.LPSSEStageStarted,
		PlanID:    createdPlan.ID,
		StageData: &models.StageEventData{
			StageCode:   snapshots[0].StageCode,
			StageName:   snapshots[0].StageName,
			StageOrder:  snapshots[0].StageOrder,
			TotalStages: len(snapshots),
		},
	})

	if strings.TrimSpace(req.RecipeID) != "" {
		if usageErr := repository.RecordRecipeUsage(
			ctx,
			req.RecipeID,
			createdPlan.ID,
			authorID,
		); usageErr != nil {
			lpGenLog.Warn(
				"记录配方使用失败，不影响创建会话",
				"plan_id", createdPlan.ID,
				"recipe_id", req.RecipeID,
				"error", usageErr,
			)
		}
	}

	return createdPlan, openingMsg, nil
}

// prepareConversationLessonPlan 完成教育域、课本与精确大纲校验后原子INSERT。
func prepareConversationLessonPlan(
	ctx context.Context,
	req *models.StartConversationRequest,
	authorID string,
	duration int,
	deps lessonPlanConversationCreationDeps,
) (*models.LessonPlan, error) {
	if req == nil {
		return nil, errors.New("开始备课请求不能为空")
	}

	user, err := deps.findUser(ctx, authorID)
	if err != nil {
		lpGenLog.Error(
			"会话创建读取用户实时角色失败",
			"author", authorID,
			"error", err,
		)
		return nil, fmt.Errorf(
			"%w: 读取用户实时角色失败",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}
	if user == nil || strings.TrimSpace(user.Role) == "" {
		return nil, fmt.Errorf(
			"%w: 用户实时角色为空",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	creationDomain, err := deps.resolveEducationDomain(
		ctx,
		authorID,
		user.Role,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrLessonPlanCreationEducationDomainConflict,
		):
			return nil, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainConflict,
				err,
			)

		case errors.Is(
			err,
			repository.ErrLessonPlanCreationEducationDomainUnavailable,
		),
			errors.Is(
				err,
				repository.ErrRegionAdminEducationDomainNotReady,
			):
			return nil, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainRequired,
				err,
			)

		default:
			lpGenLog.Error(
				"会话创建解析教育域失败",
				"author", authorID,
				"role", user.Role,
				"error", err,
			)
			return nil, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)
		}
	}

	creationDomain = strings.ToLower(
		strings.TrimSpace(creationDomain),
	)
	if !models.IsTeachingEducationDomain(
		creationDomain,
	) {
		return nil, fmt.Errorf(
			"%w: 解析结果不是具体教学域",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	if err := ValidateStartConversationTextbooks(
		ctx,
		creationDomain,
		req,
	); err != nil {
		return nil, err
	}

	if err := ValidateStartConversationCourseOutline(
		ctx,
		creationDomain,
		authorID,
		req,
	); err != nil {
		return nil, err
	}

	title := fmt.Sprintf(
		"%s %s — %s",
		req.Grade,
		req.Subject,
		req.Topic,
	)
	lp := &models.LessonPlan{
		Title:           title,
		Subject:         req.Subject,
		Grade:           req.Grade,
		Topic:           req.Topic,
		DurationMinutes: duration,
		Status:          models.LPStatusDraft,
		Visibility:      models.LPVisibilityPersonal,
		AuthorID:        authorID,
		ConversationLog: "[]",
	}

	if strings.TrimSpace(req.GroupID) != "" {
		groupID := strings.TrimSpace(req.GroupID)
		lp.GroupID = &groupID
	}
	if strings.TrimSpace(req.RecipeID) != "" {
		recipeID := strings.TrimSpace(req.RecipeID)
		lp.RecipeID = &recipeID
	}
	if strings.TrimSpace(req.CourseOutlineID) != "" {
		outlineID := strings.TrimSpace(
			req.CourseOutlineID,
		)
		lp.CourseOutlineID = &outlineID
	}

	if len(req.TextbookPageIDs) > 0 {
		textbookIDsJSON, marshalErr := json.Marshal(
			req.TextbookPageIDs,
		)
		if marshalErr != nil {
			return nil, fmt.Errorf(
				"课本图片ID序列化失败: %w",
				marshalErr,
			)
		}
		lp.TextbookPageIDs = string(textbookIDsJSON)
	}

	if err := deps.createWithEducationDomain(
		ctx,
		lp,
		creationDomain,
	); err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrLessonPlanExplicitEducationDomainRequired,
		),
			errors.Is(
				err,
				repository.ErrLessonPlanExplicitEducationDomainSnapshotMismatch,
			),
			errors.Is(
				err,
				repository.ErrLessonPlanExactCourseOutlineSnapshotMismatch,
			):
			return lp, fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)

		default:
			return lp, fmt.Errorf(
				"创建教案失败: %w",
				err,
			)
		}
	}

	storedDomain := strings.ToLower(
		strings.TrimSpace(lp.EducationDomain),
	)
	if storedDomain != creationDomain ||
		!models.IsTeachingEducationDomain(storedDomain) {
		return lp, fmt.Errorf(
			"%w: service=%s database=%s",
			ErrLPCreationEducationDomainResolveFailed,
			creationDomain,
			storedDomain,
		)
	}

	if strings.TrimSpace(req.CourseOutlineID) != "" {
		if lp.CourseOutlineID == nil ||
			strings.TrimSpace(
				*lp.CourseOutlineID,
			) != strings.TrimSpace(
				req.CourseOutlineID,
			) {
			return lp, fmt.Errorf(
				"%w: 数据库精确大纲ID未正确固化",
				ErrLPCreationEducationDomainResolveFailed,
			)
		}
	}

	return lp, nil
}
