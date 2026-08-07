package services

// courseware_ai_review_runner.go
//
// 课件 AI 审核真实批次执行器。
//
// 一次 RunNextBatch 只执行一个批次：
//   1. 重新验证审核员权限和课件审核状态；
//   2. 重新计算课件与页面快照，阻止旧会话审核已修改页面；
//   3. 复核不可变R-02审核配置；
//   4. 原子领取下一条可执行批次；
//   5. 继承上一批完整连续性账本；
//   6. 构建当前批页面文字、互动代码和CSS证据；
//   7. 保存配置哈希、教案材料使用事实和提示词哈希；
//   8. 调用AI并执行后端维度收敛；
//   9. 批次结果、风险页、模型和Token统一落库。
//
// 本执行器不会修改人工审核决定。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrCWAIReviewSessionNotFound = errors.New(
		"课件AI审核会话不存在",
	)
	ErrCWAIReviewSessionOwnerMismatch = errors.New(
		"无权执行此课件AI审核会话",
	)
	ErrCWAIReviewSessionNotRunnable = errors.New(
		"课件AI审核会话当前不可执行",
	)
	ErrCWAIReviewSnapshotExpired = errors.New(
		"课件内容已发生变化，当前AI审核会话已过期，请重新开始审核",
	)
	ErrCWAIReviewBatchBusy = errors.New(
		"当前批次正在执行，或前序批次尚未完成",
	)
)

// CoursewareAIReviewRunner 批次执行和最终综合共用的AI运行器。
type CoursewareAIReviewRunner struct {
	cfg           *config.Config
	reviewService *CoursewareReviewService
}

// NewCoursewareAIReviewRunner 创建真实AI执行器。
func NewCoursewareAIReviewRunner(
	cfg *config.Config,
	reviewService *CoursewareReviewService,
) *CoursewareAIReviewRunner {
	return &CoursewareAIReviewRunner{
		cfg:           cfg,
		reviewService: reviewService,
	}
}

// RunNextBatch 运行当前会话的下一条顺序批次。
func (s *CoursewareAIReviewRunner) RunNextBatch(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) (*models.CWAIReviewRunNextResponse, error) {
	session, courseware, pageDigests, err :=
		s.authorizeRunnableSession(
			ctx,
			sessionID,
			actor,
			true,
		)
	if err != nil {
		return nil, err
	}

	configSnapshot, err := cwAIReviewConfigFromSession(session)
	if err != nil {
		return nil, err
	}

	if session.Status == models.CWAIReviewStatusAggregating {
		return &models.CWAIReviewRunNextResponse{
			Session:          session,
			Batch:            nil,
			Result:           nil,
			HasMore:          false,
			RequiresFinalize: true,
		}, nil
	}

	batch, err :=
		repository.ClaimNextCoursewareAIReviewBatch(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, err
	}
	if batch == nil {
		return nil, ErrCWAIReviewBatchBusy
	}

	failBatch := func(cause error) {
		if cause == nil {
			return
		}

		_ = repository.FailCoursewareAIReviewBatch(
			context.Background(),
			batch.ID,
			cause.Error(),
		)
	}

	continuityBefore := strings.TrimSpace(
		session.ContinuityLedgerJSON,
	)

	previousBatch, err :=
		repository.GetPreviousCompletedCoursewareAIReviewBatch(
			ctx,
			session.ID,
			batch.BatchNo,
		)
	if err != nil {
		failBatch(err)
		return nil, err
	}

	if previousBatch != nil &&
		strings.TrimSpace(
			previousBatch.ContinuityAfterJSON,
		) != "" &&
		strings.TrimSpace(
			previousBatch.ContinuityAfterJSON,
		) != "{}" {
		continuityBefore =
			previousBatch.ContinuityAfterJSON
	}

	if continuityBefore == "" {
		continuityBefore = "{}"
	}

	systemPrompt, err :=
		buildCWAIReviewBatchSystemPrompt(session)
	if err != nil {
		failBatch(err)
		return nil, err
	}

	userPrompt, pageNumbers, err :=
		buildCWAIReviewBatchUserPrompt(
			session,
			batch,
			pageDigests,
			continuityBefore,
		)
	if err != nil {
		failBatch(err)
		return nil, err
	}

	materialUsage :=
		loadCWAIReviewSessionMaterialUsage(
			session,
		)

	inputManifest := map[string]interface{}{
		"session_id": session.ID,
		"batch_id":   batch.ID,
		"batch_no":   batch.BatchNo,

		"page_numbers": pageNumbers,

		"courseware_id": courseware.ID,

		"courseware_snapshot_hash": session.CoursewareSnapshotHash,
		"pages_snapshot_hash":      session.PagesSnapshotHash,

		"review_config_hash": session.ReviewConfigHash,
		"review_config":      cwAIReviewConfigManifest(configSnapshot),

		"selected_review_dimensions": append(
			[]string{},
			configSnapshot.ReviewDimensions...,
		),

		"lesson_reference_mode": configSnapshot.LessonReferenceMode,

		"lesson_content_included": materialUsage.LessonContentIncluded,
		"course_outline_included": materialUsage.CourseOutlineIncluded,
		"alignment_report_included": materialUsage.
			AlignmentReportIncluded,

		"system_prompt_hash":     cwAIReviewHash(systemPrompt),
		"user_prompt_hash":       cwAIReviewHash(userPrompt),
		"continuity_before_hash": cwAIReviewHash(continuityBefore),

		"system_prompt_key":     session.SystemPromptKey,
		"system_prompt_version": session.SystemPromptVersion,

		"assistant_selected": session.AssistantID != nil,
	}

	inputManifestJSON, err :=
		json.Marshal(inputManifest)
	if err != nil {
		failBatch(err)
		return nil, fmt.Errorf(
			"序列化课件AI审核真实输入清单失败: %w",
			err,
		)
	}

	inputHash := cwAIReviewHash(
		systemPrompt +
			"\n\n" +
			userPrompt,
	)

	if err := repository.UpdateCoursewareAIReviewBatchInput(
		ctx,
		batch.ID,
		continuityBefore,
		inputHash,
		string(inputManifestJSON),
	); err != nil {
		failBatch(err)
		return nil, err
	}

	if s == nil || s.cfg == nil {
		err := errors.New(
			"课件AI审核模型配置未初始化",
		)
		failBatch(err)
		return nil, err
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_ai_review",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		failBatch(err)
		return nil, fmt.Errorf(
			"获取课件AI审核模型配置失败: %w",
			err,
		)
	}

	userID := actor.UserID
	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			actor.UserID,
		)

	var traceLessonPlanID *string
	if configSnapshot.LessonReferenceMode !=
		models.CWAIReviewLessonReferenceNoLesson {
		traceLessonPlanID = session.LessonPlanID
	}

	traceContext := &ai.TraceContext{
		SceneCode: cwAIReviewTraceScene(session),
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),

		// no_lesson模式不把来源教案ID写入AI调用追踪。
		LessonPlanID: traceLessonPlanID,
	}

	callResult, err := ai.CallAI(
		aiConfig,
		systemPrompt,
		userPrompt,
		traceContext,
	)
	if err != nil {
		failBatch(err)
		return nil, fmt.Errorf(
			"课件AI审核第%d批调用失败: %w",
			batch.BatchNo,
			err,
		)
	}

	parsedResult,
		resultJSON,
		continuityAfterJSON,
		riskPagesJSON,
		err := parseCWAIReviewBatchResult(
		callResult.Content,
		session,
		batch.BatchNo,
		pageNumbers,
		continuityBefore,
	)
	if err != nil {
		failBatch(err)
		return nil, err
	}

	if err := repository.CompleteCoursewareAIReviewBatch(
		ctx,
		batch.ID,
		resultJSON,
		continuityAfterJSON,
		riskPagesJSON,
		callResult.ModelUsed,
		callResult.TokensUsed,
	); err != nil {
		failBatch(err)
		return nil, err
	}

	updatedSession, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, err
	}
	if updatedSession == nil {
		return nil, ErrCWAIReviewSessionNotFound
	}

	updatedBatch, err :=
		repository.GetCoursewareAIReviewBatchByID(
			ctx,
			batch.ID,
		)
	if err != nil {
		return nil, err
	}

	hasMore, err :=
		repository.HasRemainingCoursewareAIReviewBatches(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, err
	}

	return &models.CWAIReviewRunNextResponse{
		Session: updatedSession,
		Batch:   updatedBatch,
		Result:  parsedResult,

		HasMore: hasMore,

		RequiresFinalize: !hasMore &&
			updatedSession.Status ==
				models.CWAIReviewStatusAggregating,
	}, nil
}

// authorizeRunnableSession 统一校验会话所有权、人工审核权限和快照。
func (s *CoursewareAIReviewRunner) authorizeRunnableSession(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
	requirePageSnapshot bool,
) (
	*models.CoursewareAIReviewSession,
	*models.Courseware,
	[]models.CWAIReviewPageDigest,
	error,
) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, nil, nil,
			ErrCWAIReviewActorRequired
	}
	if s == nil {
		return nil, nil, nil,
			errors.New(
				"课件AI审核执行服务未初始化",
			)
	}

	session, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			strings.TrimSpace(sessionID),
		)
	if err != nil {
		return nil, nil, nil, err
	}
	if session == nil {
		return nil, nil, nil,
			ErrCWAIReviewSessionNotFound
	}

	if session.ReviewerID != actor.UserID &&
		actor.Role != models.RoleAdmin {
		return nil, nil, nil,
			ErrCWAIReviewSessionOwnerMismatch
	}

	switch session.Status {
	case models.CWAIReviewStatusReviewing,
		models.CWAIReviewStatusAggregating:
	default:
		return nil, nil, nil,
			ErrCWAIReviewSessionNotRunnable
	}

	// 在读取页面和生成真实输入之前先复核不可变配置。
	if _, err := cwAIReviewConfigFromSession(session); err != nil {
		return nil, nil, nil, err
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			session.CoursewareID,
		)
	if err != nil || courseware == nil {
		return nil, nil, nil,
			ErrCWAIReviewCoursewareNotFound
	}

	if session.ReviewLevel ==
		models.CWAIReviewLevelSelf {
		loadedCourseware, _, accessErr :=
			(&CoursewareService{}).
				LoadCoursewareForOwnerRuntime(
					ctx,
					courseware.ID,
					actor,
				)
		if accessErr != nil {
			return nil, nil, nil,
				accessErr
		}

		courseware = loadedCourseware
	} else {
		if s.reviewService == nil {
			return nil, nil, nil,
				errors.New(
					"课件AI审核权限服务未初始化",
				)
		}

		allowed, reviewErr :=
			s.reviewService.
				CanReviewLoadedCourseware(
					ctx,
					courseware,
					actor,
				)
		if reviewErr != nil {
			return nil, nil, nil,
				reviewErr
		}
		if !allowed {
			return nil, nil, nil,
				ErrCWAIReviewNoPermission
		}
	}

	if err := validateCWAIReviewLevel(
		courseware,
		session.ReviewLevel,
	); err != nil {
		return nil, nil, nil, err
	}

	pageDigests, err :=
		loadCurrentCWAIReviewPageDigests(
			ctx,
			courseware.ID,
		)
	if err != nil {
		return nil, nil, nil, err
	}

	if requirePageSnapshot {
		pageSnapshotJSON, err :=
			json.Marshal(pageDigests)
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf(
					"序列化当前课件页面快照失败: %w",
					err,
				)
		}

		if cwAIReviewHash(
			string(pageSnapshotJSON),
		) != session.PagesSnapshotHash {
			return nil, nil, nil,
				ErrCWAIReviewSnapshotExpired
		}

		coursewareSnapshotJSON, _ :=
			json.Marshal(
				map[string]interface{}{
					"id": courseware.ID,

					"title": courseware.Title,

					"subject": courseware.Subject,
					"grade":   courseware.Grade,

					"education_domain": courseware.EducationDomain,

					"source_type": courseware.SourceType,

					"index_overview": courseware.IndexOverview,

					"kp_codes": courseware.KPCodes,

					"updated_at": courseware.UpdatedAt,
				},
			)

		if cwAIReviewHash(
			string(coursewareSnapshotJSON),
		) != session.CoursewareSnapshotHash {
			return nil, nil, nil,
				ErrCWAIReviewSnapshotExpired
		}
	}

	return session,
		courseware,
		pageDigests,
		nil
}

// loadCurrentCWAIReviewPageDigests 读取当前页面并重建相同快照。
func loadCurrentCWAIReviewPageDigests(
	ctx context.Context,
	coursewareID string,
) ([]models.CWAIReviewPageDigest, error) {
	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			coursewareID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"读取课件AI审核页面失败: %w",
			err,
		)
	}
	if len(pages) == 0 {
		return nil, ErrCWAIReviewNoPages
	}

	sort.SliceStable(
		pages,
		func(i int, j int) bool {
			return pages[i].PageNumber <
				pages[j].PageNumber
		},
	)

	digests := make(
		[]models.CWAIReviewPageDigest,
		0,
		len(pages),
	)
	for _, page := range pages {
		if page == nil {
			continue
		}
		digests = append(
			digests,
			BuildCWAIReviewPageDigest(page),
		)
	}

	if len(digests) == 0 {
		return nil, ErrCWAIReviewNoPages
	}

	return digests, nil
}

// cwAIReviewSessionMaterialUsage 是批次真实输入账本中的材料使用事实。
type cwAIReviewSessionMaterialUsage struct {
	LessonContentIncluded   bool
	CourseOutlineIncluded   bool
	AlignmentReportIncluded bool
}

// loadCWAIReviewSessionMaterialUsage 读取准备阶段写入的材料使用清单。
//
// 对R-02上线前已经存在的会话，若context_manifest_json缺少新字段，
// 会从baseline_json实际存在的长材料字段进行兼容推断。
func loadCWAIReviewSessionMaterialUsage(
	session *models.CoursewareAIReviewSession,
) cwAIReviewSessionMaterialUsage {
	usage := cwAIReviewSessionMaterialUsage{}

	if session == nil ||
		session.LessonReferenceMode ==
			models.CWAIReviewLessonReferenceNoLesson {
		return usage
	}

	var manifest struct {
		LessonMaterialUsage struct {
			LessonContentIncluded   *bool `json:"lesson_content_included"`
			CourseOutlineIncluded   *bool `json:"course_outline_included"`
			AlignmentReportIncluded *bool `json:"alignment_report_included"`
		} `json:"lesson_material_usage"`
	}

	if err := json.Unmarshal(
		[]byte(session.ContextManifestJSON),
		&manifest,
	); err == nil {
		materialUsage := manifest.LessonMaterialUsage

		if materialUsage.LessonContentIncluded != nil {
			usage.LessonContentIncluded =
				*materialUsage.LessonContentIncluded
		}
		if materialUsage.CourseOutlineIncluded != nil {
			usage.CourseOutlineIncluded =
				*materialUsage.CourseOutlineIncluded
		}
		if materialUsage.AlignmentReportIncluded != nil {
			usage.AlignmentReportIncluded =
				*materialUsage.AlignmentReportIncluded
		}

		if materialUsage.LessonContentIncluded != nil &&
			materialUsage.CourseOutlineIncluded != nil &&
			materialUsage.AlignmentReportIncluded != nil {
			return usage
		}
	}

	baselineValue := cwAIReviewDecodeJSON(
		session.BaselineJSON,
		map[string]interface{}{},
	)
	baseline, _ :=
		baselineValue.(map[string]interface{})

	usage.LessonContentIncluded =
		cwAIReviewBaselineHasMaterialField(
			baseline,
			"lesson_plan",
			"content",
		)
	usage.CourseOutlineIncluded =
		cwAIReviewBaselineHasMaterialField(
			baseline,
			"course_outline",
			"context",
		)
	usage.AlignmentReportIncluded =
		cwAIReviewBaselineHasMaterialField(
			baseline,
			"alignment_report",
			"report",
		)

	return usage
}

func cwAIReviewBaselineHasMaterialField(
	baseline map[string]interface{},
	section string,
	field string,
) bool {
	if baseline == nil {
		return false
	}

	sectionValue, ok :=
		baseline[section].(map[string]interface{})
	if !ok {
		return false
	}

	value, exists := sectionValue[field]
	if !exists || value == nil {
		return false
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case map[string]interface{}:
		return len(typed) > 0
	case []interface{}:
		return len(typed) > 0
	default:
		return true
	}
}
