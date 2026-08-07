package services

// lesson_plan_import_service.go — 已有教案导入服务
//
// 支持老师将已有教学设计通过以下四种方式导入：
//   - paste：粘贴文本；
//   - docx：浏览器端解析Word后提交纯文本；
//   - pdf：浏览器端解析PDF后提交纯文本；
//   - docx_fidelity：后端从可信短时会话读取原Word结构和语义正文。
//
// 普通docx/pdf继续保留浏览器纯文本兼容路径；保留原格式Word通过
// 独立预解析端点私有落盘，确认时只读取当前用户可信会话中的正文、
// 结构和服务端文件哈希。两种路径都不信任请求体中的教育域字段。
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

	// ErrLPGenImportSourceInvalid 表示来源不属于paste、docx、pdf或docx_fidelity。
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
		return nil, errors.New(
			"导入教案请求不能为空",
		)
	}

	req.Subject = strings.TrimSpace(
		req.Subject,
	)
	req.Grade = strings.TrimSpace(
		req.Grade,
	)
	req.Topic = strings.TrimSpace(
		req.Topic,
	)
	req.ContentMarkdown = strings.TrimSpace(
		req.ContentMarkdown,
	)
	req.RecipeID = strings.TrimSpace(
		req.RecipeID,
	)
	req.GroupID = strings.TrimSpace(
		req.GroupID,
	)

	// Word保真路径只能使用当前用户可信短时会话中的正文。
	// 浏览器重新提交的content_markdown会被强制覆盖。
	wordImportSession, wordSessionErr :=
		resolveTrustedLessonPlanWordImportSession(
			ctx,
			req,
			authorID,
		)
	if wordSessionErr != nil {
		return nil, wordSessionErr
	}

	if req.Subject == "" {
		return nil,
			ErrLPGenSubjectRequired
	}
	if req.Grade == "" {
		return nil,
			ErrLPGenGradeRequired
	}
	if req.Topic == "" {
		return nil,
			ErrLPGenTopicRequired
	}
	if req.ContentMarkdown == "" {
		return nil,
			ErrLPGenImportContentRequired
	}

	sourceType, sourceErr :=
		normalizeLessonPlanImportSourceType(
			req.SourceType,
		)
	if sourceErr != nil {
		return nil, sourceErr
	}

	if sourceType ==
		"docx_fidelity" &&
		wordImportSession == nil {
		return nil, fmt.Errorf(
			"%w: 保留原Word格式导入必须先完成文件预解析",
			ErrLPGenImportSourceInvalid,
		)
	}

	req.SourceType =
		sourceType

	duration :=
		req.DurationMinutes
	if duration <= 0 {
		duration = 45
	}

	deps :=
		defaultLessonPlanImportCreationDeps()

	// 教育域必须在任何正式教案INSERT之前实时解析。
	creationDomain, domainErr :=
		resolveImportedLessonPlanCreationDomain(
			ctx,
			authorID,
			deps,
		)
	if domainErr != nil {
		return nil, domainErr
	}

	// 预解析时的教育域快照必须与确认时实时教育域完全一致。
	if wordImportSession != nil &&
		wordImportSession.EducationDomain !=
			creationDomain {
		return nil, fmt.Errorf(
			"%w: Word导入会话教育域已经变化，请重新上传",
			ErrLPGenImportSourceInvalid,
		)
	}

	if err :=
		ValidateImportedLessonPlanTextbooks(
			ctx,
			creationDomain,
			req,
		); err != nil {
		return nil, err
	}

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
				"author",
				authorID,
				"subject",
				req.Subject,
				"grade",
				req.Grade,
				"recipe_id",
				req.RecipeID,
				"error",
				recipeErr,
			)

			return nil, fmt.Errorf(
				"选择的备课配方当前不可用，请重新选择: %w",
				recipeErr,
			)
		}
	}

	recipeStagesConfig := ""
	if selectedRecipe != nil {
		recipeStagesConfig =
			selectedRecipe.StagesConfig
	}

	snapshots, mergeErr :=
		s.stageService.MergeStages(
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

	stageOutputs,
		skippedStages,
		outputErr :=
		buildImportedLessonPlanStageOutputs(
			snapshots,
		)
	if outputErr != nil {
		return nil, outputErr
	}

	stageConfigJSON, marshalErr :=
		json.Marshal(
			snapshots,
		)
	if marshalErr != nil {
		return nil, fmt.Errorf(
			"序列化导入教案阶段快照失败: %w",
			marshalErr,
		)
	}

	openingMessage :=
		buildImportOpeningMessage(
			req,
			skippedStages,
		)
	if openingMessage == nil {
		return nil, errors.New(
			"构建导入教案开场消息失败",
		)
	}

	lessonPlan, buildErr :=
		buildImportedLessonPlan(
			req,
			authorID,
			duration,
		)
	if buildErr != nil {
		return nil, buildErr
	}

	if createErr :=
		createImportedLessonPlanWithEducationDomain(
			ctx,
			lessonPlan,
			creationDomain,
			deps,
		); createErr != nil {
		return nil, createErr
	}

	lpGenLog.Info(
		"导入已有教案已显式写入教育域",
		"plan_id",
		lessonPlan.ID,
		"author",
		authorID,
		"source_type",
		req.SourceType,
		"content_len",
		len(
			[]rune(
				req.ContentMarkdown,
			),
		),
		"education_domain",
		lessonPlan.EducationDomain,
	)

	var (
		wordPermanentStorageKey string
		wordPermanentFullPath   string
		wordPermanentSHA256     string
		wordDocumentBound       bool
		wordDocument            *models.LessonPlanWordDocument
		wordImportedAssetPaths  []string
	)

	cleanupRequired :=
		strings.TrimSpace(
			lessonPlan.ID,
		) != ""

	defer func() {
		if !cleanupRequired {
			return
		}

		cleanupCtx, cancel :=
			context.WithTimeout(
				context.Background(),
				10*time.Second,
			)
		defer cancel()

		cleanupErr :=
			cleanupFailedImportedLessonPlanWithWord(
				cleanupCtx,
				lessonPlan.ID,
				authorID,
				creationDomain,
				wordImportSession,
				wordPermanentFullPath,
				wordDocumentBound,
				wordImportedAssetPaths,
			)

		if cleanupErr == nil {
			lpGenLog.Warn(
				"导入教案失败后的补偿清理已完成",
				"plan_id",
				lessonPlan.ID,
				"author",
				authorID,
				"word_fidelity",
				wordImportSession != nil,
			)
			return
		}

		if wordDocumentBound &&
			wordPermanentFullPath != "" {
			lpGenLog.Error(
				"Word文档已绑定且教案补偿可能失败，为避免断链保留正式文件",
				"plan_id",
				lessonPlan.ID,
				"path",
				wordPermanentFullPath,
			)
		}

		lpGenLog.Error(
			"导入教案失败且补偿清理失败",
			"plan_id",
			lessonPlan.ID,
			"author",
			authorID,
			"creation_error",
			resultErr,
			"cleanup_error",
			cleanupErr,
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

	// 在创建Word当前文档前，从可信DOCX中提取安全图片资产。
	//
	// 本步骤会同步更新：
	//   - lesson_plan_assets；
	//   - lesson_plans.content_markdown；
	//   - Word短时会话的structure_json和semantic_markdown。
	//
	// 后续任一步失败时，上方补偿逻辑会删除教案、资产记录和物理图片。
	if wordImportSession != nil {
		mediaResult, mediaErr :=
			hydrateLessonPlanWordImportMediaAssets(
				ctx,
				wordImportSession,
				lessonPlan.ID,
				authorID,
				creationDomain,
			)
		if mediaErr != nil {
			return nil, fmt.Errorf(
				"提取原Word图片资产失败: %w",
				mediaErr,
			)
		}

		wordImportedAssetPaths =
			mediaResult.AssetFullPaths

		req.ContentMarkdown =
			mediaResult.SemanticMarkdown
		lessonPlan.ContentMarkdown =
			mediaResult.SemanticMarkdown

		lpGenLog.Info(
			"原Word图片已同步为教案资产",
			"plan_id",
			lessonPlan.ID,
			"session_id",
			wordImportSession.ID,
			"image_count",
			mediaResult.ImportedImageCount,
		)
	}

	// 生成正式教案ID后，复制并复核不可变DOCX版本1。
	if wordImportSession != nil {
		var permanentErr error

		wordPermanentStorageKey,
			wordPermanentFullPath,
			wordPermanentSHA256,
			permanentErr =
			prepareLessonPlanWordPermanentFile(
				wordImportSession,
				lessonPlan.ID,
			)
		if permanentErr != nil {
			return nil, fmt.Errorf(
				"固化原格式Word文件失败: %w",
				permanentErr,
			)
		}
	}

	// 在写入阶段和开场消息前预登记AI任务。
	// 任务登记失败时不会留下半成品教案或正式Word文件。
	task, taskErr :=
		startLessonPlanAITask(
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

	// 在阶段固化前绑定Word文档。
	// 如果后续阶段固化失败，现有教案补偿删除可以安全级联清理Word记录。
	if wordImportSession != nil {
		var wordBindErr error

		wordDocument, wordBindErr =
			repository.ConfirmLessonPlanWordImport(
				ctx,
				models.ConfirmLessonPlanWordImportInput{
					ImportSessionID:     wordImportSession.ID,
					LessonPlanID:        lessonPlan.ID,
					OwnerID:             authorID,
					EducationDomain:     creationDomain,
					PermanentStorageKey: wordPermanentStorageKey,
					PermanentFileSHA256: wordPermanentSHA256,
					ChangeSummary:       "保留原Word格式导入并创建首个版本",
				},
			)
		if wordBindErr != nil {
			return nil, fmt.Errorf(
				"绑定原格式Word文档失败: %w",
				wordBindErr,
			)
		}

		wordDocumentBound = true
	}

	for index := range stageOutputs {
		stageOutputs[index].LessonPlanID =
			lessonPlan.ID
	}

	currentStage := "review"

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

	lessonPlan.CurrentStage =
		currentStage
	lessonPlan.StageConfig =
		string(stageConfigJSON)

	expectedDomain :=
		lessonPlan.EducationDomain
	planID :=
		lessonPlan.ID
	isWordFidelity :=
		wordImportSession != nil

	s.runLessonPlanAITask(
		task,
		planID,
		"",
		"import_review",
		func() {
			time.Sleep(
				800 *
					time.Millisecond,
			)

			bgCtx :=
				context.Background()

			freshPlan, freshErr :=
				repository.GetLessonPlanByID(
					bgCtx,
					planID,
				)
			if freshErr != nil {
				lpGenLog.Error(
					"导入教案异步评审重新读取教案失败",
					"plan_id",
					planID,
					"error",
					freshErr,
				)
				s.broadcastError(
					planID,
					"",
					"导入教案已保存，但AI评审启动失败，请稍后手动重试",
				)
				return
			}

			freshDomain :=
				strings.ToLower(
					strings.TrimSpace(
						freshPlan.EducationDomain,
					),
				)

			if !models.IsTeachingEducationDomain(
				freshDomain,
			) ||
				freshDomain !=
					expectedDomain {
				lpGenLog.Error(
					"导入教案异步评审教育域快照异常",
					"plan_id",
					planID,
					"expected_domain",
					expectedDomain,
					"stored_domain",
					freshDomain,
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
					"plan_id",
					planID,
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
				"plan_id",
				planID,
				"education_domain",
				freshDomain,
				"word_fidelity",
				isWordFidelity,
			)

			s.executeAIReviewAsync(
				bgCtx,
				freshPlan,
			)
		},
	)

	taskLaunched = true
	cleanupRequired = false

	// 所有数据库状态成功后，删除短时imports副本。
	// 删除失败只记录日志，不回滚已经成功的正式教案。
	if wordImportSession != nil {
		if removeErr :=
			removeLessonPlanWordImportSourceFile(
				wordImportSession,
			); removeErr != nil {
			lpGenLog.Warn(
				"删除已确认Word短时导入副本失败",
				"session_id",
				wordImportSession.ID,
				"plan_id",
				lessonPlan.ID,
				"error",
				removeErr,
			)
		}
	}

	lpGenLog.Info(
		"导入已有教案主流程完成",
		"plan_id",
		lessonPlan.ID,
		"skipped_stages",
		skippedStages,
		"current_stage",
		lessonPlan.CurrentStage,
		"education_domain",
		lessonPlan.EducationDomain,
		"word_fidelity",
		wordDocument != nil,
	)

	return &models.ImportExistingPlanResponse{
		Plan:           lessonPlan,
		OpeningMessage: openingMessage,
		SkippedStages:  skippedStages,
		WordDocument:   wordDocument,
	}, nil
}
