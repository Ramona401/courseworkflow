package services

// lesson_plan_gen_start.go
//
// 本文件专门承载“开始对话 / 专家模式”共用的会话创建编排。
//
// 上下文11的核心规则：
//   1. 不信任JWT中的历史角色，也不接受前端传入教育域；
//   2. 创建教案前实时读取users.role；
//   3. 调用统一解析器ResolveLessonPlanCreationEducationDomain；
//   4. 只接受k12、vocational、adult三个具体教学域；
//   5. 使用CreateLessonPlanWithEducationDomain显式写入资源快照；
//   6. 阶段初始化或开场消息持久化失败时，执行同步补偿清理；
//   7. 配方使用记录与SSE事件只在核心资源完整持久化后产生。
//
// 普通对话模式与专家模式都只通过StartConversation进入。
// 专家模式传入的recipe_id只能影响配方选择，不能影响教育域解析或数据库快照。

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

// lessonPlanConversationCreationDeps 是会话创建阶段的最小依赖集合。
//
// 生产代码使用defaultLessonPlanConversationCreationDeps返回真实Repository函数；
// 测试代码可注入脱库实现，验证三个具体教育域与fail-closed规则，
// 不需要修改全局函数变量，也不会产生并发测试污染。
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

// defaultLessonPlanConversationCreationDeps 返回生产环境真实依赖。
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
//
// 失败原子性采用“显式写域 + 同步补偿链”：
//   - 教育域解析失败：任何INSERT前直接返回；
//   - 教案INSERT失败：数据库事务自行回滚；
//   - 阶段初始化失败：硬清理新教案及已产生的阶段记录；
//   - 开场消息写入失败：硬清理新教案及阶段记录；
//   - 补偿清理自身失败：返回500并记录高优先级错误，绝不伪装成功。
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

	// 配方三态解析与自动挂载。
	//
	// recipe_mode:
	//   auto     -> 根据学校默认、教研组共享和学科规则自动选择；
	//   selected -> 使用老师明确传入的recipe_id；
	//   none     -> 老师明确不使用配方，禁止自动匹配。
	//
	// 旧客户端不传recipe_mode时保持兼容：
	// 有recipe_id视为selected，无recipe_id视为auto。
	recipeSelectionMode := normalizeStartRecipeSelection(req)

	// 老师明确选择的配方只校验active状态与当前老师使用权限。
	// 该校验只读数据库，不会在教育域异常时留下任何资源。
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
		// 明确不用配方时清空旧客户端可能残留的recipe_id，
		// 防止“none + recipe_id”组合意外进入专家模式路径。
		req.RecipeID = ""
		lpGenLog.Info(
			"老师明确选择不使用配方",
			"author", authorID,
			"subject", req.Subject,
			"topic", req.Topic,
			"recipe_mode", recipeSelectionMode,
		)
	}

	// 统一教育域硬闸与显式写域。
	//
	// prepareConversationLessonPlan会实时读取users.role并调用统一解析器。
	// mixed管理身份、无有效组织、非法域和跨域冲突都会在INSERT前失败。
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

	// 创建成功后直到“阶段初始化 + 开场消息持久化”全部完成前，
	// 任一返回路径都会触发同步补偿。使用后台独立超时上下文，
	// 避免原HTTP请求取消后清理也被立即取消。
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

		// 保留原错误作为errors.Is判断链，同时附加清理失败详情。
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
		"开始备课会话已显式写入教育域",
		"plan_id", createdPlan.ID,
		"topic", req.Topic,
		"author", authorID,
		"recipe_id", req.RecipeID,
		"education_domain", createdPlan.EducationDomain,
	)

	// 读取配方阶段配置。配方详情读取失败时保持旧行为：
	// 使用系统默认阶段，不把配方查询瞬时故障扩大成空教案残留。
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

	// AI开场白失败时使用确定性默认开场。
	// 默认开场同样必须持久化成功，否则整次会话创建失败并补偿清理。
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

	// 将本次配方选择方式写入开场消息metadata。
	// ConversationMessage整体进入conversation_log，无需新增数据库字段。
	if openingMsg.Metadata == nil {
		openingMsg.Metadata = make(map[string]interface{})
	}
	openingMsg.Metadata[recipeSelectionModeMetadataKey] = string(
		recipeSelectionMode,
	)

	// 若关联课本图但存在未完成OCR的页面，确定性追加提醒。
	s.appendUnrecognizedTextbookNotice(
		ctx,
		req,
		openingMsg,
	)

	// 开场消息是完整会话不可缺少的核心状态。
	// 不再像旧逻辑一样只Warn后返回成功。
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

	// 至此教案、阶段快照、首阶段产出和开场消息均已持久化。
	// 关闭补偿开关后，后续旁路失败不能删除一个完整可用会话。
	cleanupRequired = false

	// 推送阶段开始事件。只有完整会话才允许对外广播。
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

	// 配方使用记录属于旁路统计，只能在完整会话持久化后写入。
	// 写入失败不影响主业务，也不会产生失败会话的孤立使用记录。
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

// prepareConversationLessonPlan 完成会话教案的教育域解析与显式INSERT。
//
// 返回约定：
//   - INSERT前失败：返回nil,error；
//   - INSERT事务失败：通常返回带空ID的lp,error；
//   - INSERT已提交但Service防御性复核失败：返回带ID的lp,error，
//     由StartConversation的补偿守卫负责硬清理。
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
	if !models.IsTeachingEducationDomain(creationDomain) {
		return nil, fmt.Errorf(
			"%w: 解析结果不是具体教学域",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	// 上下文15：请求携带课本ID时，必须在任何教案INSERT前
	// 通过K12课本统一硬闸。
	//
	// 非K12携带课本ID返回403；
	// 不存在、归档、重复或学科年级不匹配的ID返回400；
	// 数据库错误保持错误链并由Handler映射为5xx。
	if err := ValidateStartConversationTextbooks(
		ctx,
		creationDomain,
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
				repository.
					ErrLessonPlanExplicitEducationDomainSnapshotMismatch,
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

	return lp, nil
}
