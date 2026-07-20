package services

// lesson_plan_import_service.go — 已有教案导入服务
//
// 支持老师将已有教学设计通过以下三种方式导入：
//   - paste：粘贴文本；
//   - docx：浏览器端解析Word后提交纯文本；
//   - pdf：浏览器端解析PDF后提交纯文本。
//
// 文件解析全部在浏览器中完成，后端不接收原始Word/PDF文件，
// 不创建服务器临时文件，也不信任请求体中的任何教育域字段。
//
// 上下文12正式创建链：
//   1. 校验导入来源与正文；
//   2. 实时读取users.role并严格解析唯一具体教育域；
//   3. 校验老师手动选择的配方；
//   4. 只读合并阶段并确认存在review阶段；
//   5. 使用CreateLessonPlanWithEducationDomain显式写入资源域快照；
//   6. 预登记受控AI评审任务；
//   7. 单事务固化阶段快照、跳过状态、review状态和开场消息；
//   8. 受控后台任务重新读取教案教育域快照后执行AI评审。
//
// 任一核心步骤失败时，同步硬清理本次新建的导入教案。
// 异步AI评审失败时保留已有正文和可恢复的review阶段，
// 并通过SSE返回明确错误；老师仍可手动重新触发评审。

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

var (
	// ErrLPGenImportContentRequired 表示导入正文为空。
	ErrLPGenImportContentRequired = errors.New(
		"教案内容不能为空",
	)

	// ErrLPGenImportSourceInvalid 表示来源不属于paste、docx或pdf。
	ErrLPGenImportSourceInvalid = errors.New(
		"导入来源类型无效",
	)

	// ErrLPGenImportReviewStageRequired 表示系统或配方阶段中缺少review阶段。
	ErrLPGenImportReviewStageRequired = errors.New(
		"导入流程缺少AI评审阶段",
	)
)

// lessonPlanImportCreationDeps 是导入创建教育域硬闸的最小依赖集合。
//
// 生产环境使用真实Repository函数；测试可注入脱库函数，
// 验证三个具体教学域与fail-closed规则，不修改全局函数变量。
type lessonPlanImportCreationDeps struct {
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
}

// defaultLessonPlanImportCreationDeps 返回生产环境真实依赖。
func defaultLessonPlanImportCreationDeps() lessonPlanImportCreationDeps {
	return lessonPlanImportCreationDeps{
		findUser: repository.FindUserByID,
		resolveEducationDomain: repository.
			ResolveLessonPlanCreationEducationDomain,
		createWithEducationDomain: repository.
			CreateLessonPlanWithEducationDomain,
	}
}

// ImportExistingPlan 导入已有教案。
func (s *LessonPlanGenService) ImportExistingPlan(
	ctx context.Context,
	req *models.ImportExistingPlanRequest,
	authorID string,
) (
	result *models.ImportExistingPlanResponse,
	resultErr error,
) {
	if IsGlobalBackgroundDraining() {
		return nil, ErrLPGenServiceDraining
	}
	if req == nil {
		return nil, errors.New("导入教案请求不能为空")
	}

	// 统一规范化所有用于创建快照的文本字段。
	req.Subject = strings.TrimSpace(req.Subject)
	req.Grade = strings.TrimSpace(req.Grade)
	req.Topic = strings.TrimSpace(req.Topic)
	req.ContentMarkdown = strings.TrimSpace(
		req.ContentMarkdown,
	)
	req.RecipeID = strings.TrimSpace(req.RecipeID)
	req.GroupID = strings.TrimSpace(req.GroupID)

	if req.Subject == "" {
		return nil, ErrLPGenSubjectRequired
	}
	if req.Grade == "" {
		return nil, ErrLPGenGradeRequired
	}
	if req.Topic == "" {
		return nil, ErrLPGenTopicRequired
	}
	if req.ContentMarkdown == "" {
		return nil, ErrLPGenImportContentRequired
	}

	sourceType, sourceErr := normalizeLessonPlanImportSourceType(
		req.SourceType,
	)
	if sourceErr != nil {
		return nil, sourceErr
	}
	req.SourceType = sourceType

	duration := req.DurationMinutes
	if duration <= 0 {
		duration = 45
	}

	deps := defaultLessonPlanImportCreationDeps()

	// 教育域必须在任何INSERT之前完成严格解析。
	creationDomain, domainErr :=
		resolveImportedLessonPlanCreationDomain(
			ctx,
			authorID,
			deps,
		)
	if domainErr != nil {
		return nil, domainErr
	}

	// 上下文15：导入请求携带课本ID时，必须在任何教案INSERT前
	// 通过K12课本统一硬闸。
	//
	// 职教、成教或其它非K12教案不能通过导入接口写入课本ID；
	// 伪造、归档、重复或属性不一致的课本ID也会整体拒绝。
	if err := ValidateImportedLessonPlanTextbooks(
		ctx,
		creationDomain,
		req,
	); err != nil {
		return nil, err
	}

	// 导入请求中的recipe_id只表示老师手动选择的配方。
	// 必须验证active状态和当前老师的实际使用权限。
	var selectedRecipe *models.TeachingRecipe
	if req.RecipeID != "" {
		var recipeErr error
		selectedRecipe, recipeErr =
			loadRecipeForManualSelection(
				ctx,
				req.RecipeID,
				authorID,
			)
		if recipeErr != nil {
			lpGenLog.Warn(
				"导入已有教案携带的手动配方不可用，拒绝继续导入",
				"author", authorID,
				"subject", req.Subject,
				"grade", req.Grade,
				"recipe_id", req.RecipeID,
				"error", recipeErr,
			)
			return nil, fmt.Errorf(
				"选择的备课配方当前不可用，请重新选择: %w",
				recipeErr,
			)
		}
	}

	// 所有阶段读取与合并都在创建教案之前完成。
	// 缺少review阶段时直接失败，不留下教案记录。
	recipeStagesConfig := ""
	if selectedRecipe != nil {
		recipeStagesConfig =
			selectedRecipe.StagesConfig
	}

	snapshots, mergeErr := s.stageService.MergeStages(
		ctx,
		recipeStagesConfig,
		req.RecipeID,
	)
	if mergeErr != nil {
		return nil, fmt.Errorf(
			"合并导入教案阶段失败: %w",
			mergeErr,
		)
	}

	stageOutputs, skippedStages, outputErr :=
		buildImportedLessonPlanStageOutputs(
			snapshots,
		)
	if outputErr != nil {
		return nil, outputErr
	}

	stageConfigJSON, marshalErr := json.Marshal(
		snapshots,
	)
	if marshalErr != nil {
		return nil, fmt.Errorf(
			"序列化导入教案阶段快照失败: %w",
			marshalErr,
		)
	}

	openingMessage := buildImportOpeningMessage(
		req,
		skippedStages,
	)
	if openingMessage == nil {
		return nil, errors.New(
			"构建导入教案开场消息失败",
		)
	}

	lessonPlan, buildErr := buildImportedLessonPlan(
		req,
		authorID,
		duration,
	)
	if buildErr != nil {
		return nil, buildErr
	}

	// SQL显式写入教育域，Repository事务校验数据库最终快照。
	if createErr := createImportedLessonPlanWithEducationDomain(
		ctx,
		lessonPlan,
		creationDomain,
		deps,
	); createErr != nil {
		return nil, createErr
	}

	lpGenLog.Info(
		"导入已有教案已显式写入教育域",
		"plan_id", lessonPlan.ID,
		"author", authorID,
		"source_type", req.SourceType,
		"content_len", len([]rune(req.ContentMarkdown)),
		"education_domain", lessonPlan.EducationDomain,
	)

	// 从教案INSERT成功开始，直到核心状态全部固化且AI任务启动前，
	// 任一失败路径都必须同步硬清理本次新建记录。
	cleanupRequired := strings.TrimSpace(
		lessonPlan.ID,
	) != ""

	defer func() {
		if !cleanupRequired {
			return
		}

		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		cleanupErr :=
			repository.DeleteIncompleteImportedLessonPlanCreation(
				cleanupCtx,
				lessonPlan.ID,
				authorID,
				creationDomain,
			)
		if cleanupErr == nil {
			lpGenLog.Warn(
				"导入教案失败后的补偿清理已完成",
				"plan_id", lessonPlan.ID,
				"author", authorID,
			)
			return
		}

		lpGenLog.Error(
			"导入教案失败且补偿清理失败",
			"plan_id", lessonPlan.ID,
			"author", authorID,
			"creation_error", resultErr,
			"cleanup_error", cleanupErr,
		)

		if resultErr == nil {
			resultErr = fmt.Errorf(
				"导入教案补偿清理失败: %w",
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

	// 在固化阶段和开场消息之前预登记AI评审任务。
	//
	// draining竞态或任务登记失败时，当前新建教案会被上方补偿守卫清理，
	// 不会返回一个声称“正在AI评审”但实际没有任务的半成品。
	task, taskErr := startLessonPlanAITask(
		lessonPlan.ID,
	)
	if taskErr != nil {
		return nil, taskErr
	}

	taskLaunched := false
	defer func() {
		if !taskLaunched {
			task.Done()
		}
	}()

	for index := range stageOutputs {
		stageOutputs[index].LessonPlanID =
			lessonPlan.ID
	}

	currentStage := "review"

	// 单事务完成：
	//   - 阶段快照；
	//   - review之前阶段的skipped记录；
	//   - review阶段in_progress记录；
	//   - current_stage；
	//   - 导入成功开场消息。
	if finalizeErr :=
		repository.FinalizeImportedLessonPlanCreation(
			ctx,
			lessonPlan.ID,
			authorID,
			creationDomain,
			string(stageConfigJSON),
			currentStage,
			stageOutputs,
			openingMessage,
		); finalizeErr != nil {
		return nil, fmt.Errorf(
			"固化导入教案工作流失败: %w",
			finalizeErr,
		)
	}

	lessonPlan.CurrentStage = currentStage
	lessonPlan.StageConfig = string(stageConfigJSON)

	expectedDomain := lessonPlan.EducationDomain
	planID := lessonPlan.ID

	// 使用统一教案AI任务治理启动异步评审。
	//
	// 后台执行时重新读取正式教案，并重新验证：
	//   - 教案仍然存在；
	//   - 正文仍然非空；
	//   - 教育域快照仍是具体教学域；
	//   - 快照与创建时返回值完全一致。
	s.runLessonPlanAITask(
		task,
		planID,
		"",
		"import_review",
		func() {
			// 给前端保存响应、建立SSE连接留出短暂时间。
			time.Sleep(800 * time.Millisecond)

			bgCtx := context.Background()
			freshPlan, freshErr :=
				repository.GetLessonPlanByID(
					bgCtx,
					planID,
				)
			if freshErr != nil {
				lpGenLog.Error(
					"导入教案异步评审重新读取教案失败",
					"plan_id", planID,
					"error", freshErr,
				)
				s.broadcastError(
					planID,
					"",
					"导入教案已保存，但AI评审启动失败，请稍后手动重试",
				)
				return
			}

			freshDomain := strings.ToLower(
				strings.TrimSpace(
					freshPlan.EducationDomain,
				),
			)
			if !models.IsTeachingEducationDomain(
				freshDomain,
			) ||
				freshDomain != expectedDomain {
				lpGenLog.Error(
					"导入教案异步评审教育域快照异常",
					"plan_id", planID,
					"expected_domain", expectedDomain,
					"stored_domain", freshDomain,
				)
				s.broadcastError(
					planID,
					"",
					"教案教育域快照异常，AI评审已安全停止",
				)
				return
			}

			if strings.TrimSpace(
				freshPlan.ContentMarkdown,
			) == "" {
				lpGenLog.Error(
					"导入教案异步评审检测到正文为空",
					"plan_id", planID,
				)
				s.broadcastError(
					planID,
					"",
					"教案正文为空，AI评审已安全停止",
				)
				return
			}

			lpGenLog.Info(
				"导入已有教案开始受控异步AI评审",
				"plan_id", planID,
				"education_domain", freshDomain,
			)

			s.executeAIReviewAsync(
				bgCtx,
				freshPlan,
			)
		},
	)

	taskLaunched = true
	cleanupRequired = false

	lpGenLog.Info(
		"导入已有教案主流程完成",
		"plan_id", lessonPlan.ID,
		"skipped_stages", skippedStages,
		"current_stage", lessonPlan.CurrentStage,
		"education_domain", lessonPlan.EducationDomain,
	)

	return &models.ImportExistingPlanResponse{
		Plan:           lessonPlan,
		OpeningMessage: openingMessage,
		SkippedStages:  skippedStages,
	}, nil
}
