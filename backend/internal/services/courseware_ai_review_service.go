package services

// courseware_ai_review_service.go
//
// 课件AI审核会话准备服务的入口与权限编排。
//
// 本文件负责：
//   1. 校验作者自审或正式审核权限；
//   2. 规范化R-02审核维度和教案参考模式；
//   3. 重新读取课件和完整页面；
//   4. 选择并复核课件审核AI助手；
//   5. 调用材料、快照和批次构建辅助；
//   6. 创建不可变配置会话并保存批次；
//   7. 保持旧会话取消、新会话创建和准备状态推进的既有语义。
//
// 具体职责拆分：
//   - courseware_ai_review_prepare_context.go：
//     页面摘要、教案材料隔离、上下文清单、基准和哈希快照；
//   - courseware_ai_review_prepare_batches.go：
//     页面批次规划、教学环节边界和初始连续性账本。
//
// R-02安全边界：
//   - 浏览器只提交配置选择意图；
//   - 课件、页面、身份、教育域、教案和大纲均由后端重新读取；
//   - no_lesson模式不会调用教案、大纲和对齐报告读取链；
//   - 配置在会话创建时固化，创建后不能原地修改。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	coursewareAIReviewPromptKey = "prompt_courseware_ai_review"

	cwAIReviewLessonMaxRunes  = 50000
	cwAIReviewOutlineMaxRunes = 60000

	cwAIReviewBatchTargetPages = 5
	cwAIReviewBatchMinPages    = 3
	cwAIReviewBatchMaxPages    = 6
)

var (
	ErrCWAIReviewActorRequired = errors.New(
		"缺少课件AI审核操作者",
	)
	ErrCWAIReviewCoursewareNotFound = errors.New(
		"正在审核的课件不存在",
	)
	ErrCWAIReviewNoPermission = errors.New(
		"没有使用AI审核此课件的权限",
	)
	ErrCWAIReviewInvalidLevel = errors.New(
		"课件当前状态与审核级别不一致",
	)
	ErrCWAIReviewNoPages = errors.New(
		"课件没有可审核的页面",
	)
	ErrCWAIReviewLessonPlanMissing = errors.New(
		"课件关联的来源教案不存在",
	)
	ErrCWAIReviewLessonDomainMismatch = errors.New(
		"课件与来源教案的教育域不一致",
	)
	ErrCWAIReviewPromptUnavailable = errors.New(
		"课件AI审核系统提示词不可用",
	)
)

// CoursewareAIReviewService 课件AI审核会话准备服务。
type CoursewareAIReviewService struct {
	reviewService     *CoursewareReviewService
	coursewareService *CoursewareService
	assistantService  *AIAssistantService
}

// NewCoursewareAIReviewService 创建课件AI审核服务。
func NewCoursewareAIReviewService(
	reviewService *CoursewareReviewService,
	coursewareService *CoursewareService,
	assistantService *AIAssistantService,
) *CoursewareAIReviewService {
	return &CoursewareAIReviewService{
		reviewService:     reviewService,
		coursewareService: coursewareService,
		assistantService:  assistantService,
	}
}

// PrepareSession 重新准备一次课件AI审核会话。
//
// assistantID为空时只使用版本化系统提示词；
// 非空时叠加审核员明确选择且有权使用的AI助手提示词。
//
// configInput为nil或字段缺失时使用现行兼容预设。
// 明确提交空维度数组、重复维度、自定义说明冲突或非法模式时返回配置错误。
func (s *CoursewareAIReviewService) PrepareSession(
	ctx context.Context,
	coursewareID string,
	reviewLevel int,
	actor *CoursewareActorContext,
	assistantID string,
	configInput *CWAIReviewConfigInput,
) (*models.CoursewareAIReviewSession, error) {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}
	if s == nil || s.reviewService == nil || s.coursewareService == nil {
		return nil, errors.New("课件AI审核服务未初始化")
	}

	configSnapshot, err := NormalizeCWAIReviewConfig(configInput)
	if err != nil {
		return nil, err
	}

	courseware, err := repository.GetCoursewareByID(
		ctx,
		strings.TrimSpace(coursewareID),
	)
	if err != nil || courseware == nil {
		return nil, ErrCWAIReviewCoursewareNotFound
	}

	scopedActor := actor

	if reviewLevel == models.CWAIReviewLevelSelf {
		loadedCourseware, ownerActor, accessErr :=
			s.coursewareService.LoadCoursewareForOwnerRuntime(
				ctx,
				courseware.ID,
				actor,
			)
		if accessErr != nil {
			return nil, accessErr
		}

		courseware = loadedCourseware
		scopedActor = ownerActor
	} else {
		allowed, reviewErr := s.reviewService.CanReviewLoadedCourseware(
			ctx,
			courseware,
			actor,
		)
		if reviewErr != nil {
			return nil, reviewErr
		}
		if !allowed {
			return nil, ErrCWAIReviewNoPermission
		}
	}

	if err := validateCWAIReviewLevel(courseware, reviewLevel); err != nil {
		return nil, err
	}

	// 作者自审授权后使用已经收敛到课件教育域快照的Actor。
	actor = scopedActor

	detail, err := s.coursewareService.GetCourseware(ctx, courseware.ID)
	if err != nil {
		return nil, fmt.Errorf("读取课件完整页面失败: %w", err)
	}
	if detail == nil {
		return nil, ErrCWAIReviewNoPages
	}

	pageDigests := buildCWAIReviewPreparationPageDigests(detail.Pages)
	if len(pageDigests) == 0 {
		return nil, ErrCWAIReviewNoPages
	}

	materials, err := loadCWAIReviewPreparedMaterials(
		ctx,
		courseware,
		configSnapshot,
	)
	if err != nil {
		return nil, err
	}

	systemPrompt, err := repository.GetCurrentPromptByKey(
		coursewareAIReviewPromptKey,
	)
	if err != nil ||
		systemPrompt == nil ||
		strings.TrimSpace(systemPrompt.Content) == "" {
		return nil, ErrCWAIReviewPromptUnavailable
	}

	assistantPrompt, selectedAssistantID, err :=
		s.resolveCWAIReviewAssistantPrompt(
			ctx,
			courseware,
			reviewLevel,
			actor,
			assistantID,
		)
	if err != nil {
		return nil, err
	}

	prepared, err := buildCWAIReviewPreparedSnapshot(
		courseware,
		detail,
		pageDigests,
		materials,
		configSnapshot,
		selectedAssistantID,
		assistantID,
		reviewLevel,
	)
	if err != nil {
		return nil, err
	}

	session := &models.CoursewareAIReviewSession{
		CoursewareID: courseware.ID,
		ReviewerID:   actor.UserID,
		AssistantID:  selectedAssistantID,
		LessonPlanID: materials.lessonPlanID,

		ReviewLevel:     reviewLevel,
		EducationDomain: courseware.EducationDomain,
		Subject:         courseware.Subject,
		Grade:           courseware.Grade,

		ReviewConfigSchemaVersion:  configSnapshot.SchemaVersion,
		ReviewDimensionsJSON:       configSnapshot.ReviewDimensionsJSON,
		CustomDimensionDescription: configSnapshot.CustomDimensionDescription,
		LessonReferenceMode:        configSnapshot.LessonReferenceMode,

		Status:         models.CWAIReviewStatusPreparing,
		CurrentStage:   models.CWAIReviewStageBaseline,
		CurrentBatchNo: 0,
		TotalBatches:   len(prepared.batches),

		CoursewareSnapshotHash:    prepared.coursewareSnapshotHash,
		PagesSnapshotHash:         prepared.pagesSnapshotHash,
		LessonPlanSnapshotHash:    prepared.lessonSnapshotHash,
		CourseOutlineSnapshotHash: prepared.outlineSnapshotHash,

		SystemPromptKey:         systemPrompt.PromptKey,
		SystemPromptVersion:     systemPrompt.Version,
		SystemPromptSnapshot:    systemPrompt.Content,
		AssistantPromptSnapshot: assistantPrompt,

		ContextManifestJSON:  prepared.contextManifestJSON,
		BaselineJSON:         prepared.baselineJSON,
		PageIndexJSON:        prepared.pageIndexJSON,
		ContinuityLedgerJSON: prepared.ledgerJSON,
		FinalReportJSON:      "{}",
	}

	cancelReason := "审核员重新开始课件AI审核"
	if reviewLevel == models.CWAIReviewLevelSelf {
		cancelReason = "作者重新开始课件AI自审"
	}

	if err := repository.CancelActiveCoursewareAIReviewSessions(
		ctx,
		courseware.ID,
		actor.UserID,
		reviewLevel,
		cancelReason,
	); err != nil {
		return nil, err
	}

	if err := repository.CreateCoursewareAIReviewSession(
		ctx,
		session,
	); err != nil {
		return nil, err
	}

	if err := repository.ReplaceCoursewareAIReviewBatches(
		ctx,
		session.ID,
		prepared.batches,
	); err != nil {
		_ = repository.MarkCoursewareAIReviewSessionFailed(
			ctx,
			session.ID,
			err.Error(),
		)
		return nil, err
	}

	if err := repository.UpdateCoursewareAIReviewPrepared(
		ctx,
		session,
	); err != nil {
		_ = repository.MarkCoursewareAIReviewSessionFailed(
			ctx,
			session.ID,
			err.Error(),
		)
		return nil, err
	}

	return session, nil
}

// resolveCWAIReviewAssistantPrompt 读取审核员明确选择且有权使用的AI助手提示词。
func (s *CoursewareAIReviewService) resolveCWAIReviewAssistantPrompt(
	ctx context.Context,
	courseware *models.Courseware,
	reviewLevel int,
	actor *CoursewareActorContext,
	assistantID string,
) (string, *string, error) {
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		return "", nil, nil
	}
	if s == nil || s.assistantService == nil {
		return "", nil, errors.New("AI助手服务未初始化")
	}
	if courseware == nil || actor == nil {
		return "", nil, ErrCWAIReviewNoPermission
	}

	assistantActor := BuildActorFromClaims(
		ctx,
		actor.UserID,
		actor.Role,
	)
	if assistantActor == nil {
		return "", nil, ErrCWAIReviewNoPermission
	}

	assistantActor.EducationDomain = strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)

	assistantScene := models.SceneCoursewareReview
	if reviewLevel == models.CWAIReviewLevelSelf {
		assistantScene = models.SceneCoursewareSelfReview
	}

	// 课件审核助手由审核者明确手动选择。
	//
	// 运行时继续校验教育域、可见性、启用状态、学科和审核场景，
	// 但不把助手适用年级作为课件审核的使用门槛。
	assistant, err :=
		s.assistantService.LoadActiveAssistantForManualLessonUse(
			ctx,
			assistantActor,
			assistantID,
			courseware.Subject,
			assistantScene,
		)
	if err != nil {
		return "", nil, err
	}
	if assistant == nil || strings.TrimSpace(assistant.FullPrompt) == "" {
		return "", nil, errors.New("选择的AI助手没有可用提示词")
	}

	selectedAssistantID := assistant.ID

	return assistant.FullPrompt, &selectedAssistantID, nil
}

// GetSessionForReviewer 返回审核员自己的会话及批次。
func (s *CoursewareAIReviewService) GetSessionForReviewer(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) (
	*models.CoursewareAIReviewSession,
	[]*models.CoursewareAIReviewBatch,
	error,
) {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, nil, ErrCWAIReviewActorRequired
	}

	session, err := repository.GetCoursewareAIReviewSessionByID(
		ctx,
		strings.TrimSpace(sessionID),
	)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, errors.New("课件AI审核会话不存在")
	}
	if session.ReviewerID != actor.UserID && actor.Role != models.RoleAdmin {
		return nil, nil, ErrCWAIReviewNoPermission
	}

	batches, err := repository.ListCoursewareAIReviewBatches(
		ctx,
		session.ID,
	)
	if err != nil {
		return nil, nil, err
	}

	return session, batches, nil
}

// validateCWAIReviewLevel 校验当前课件状态是否允许对应级别的AI审核。
func validateCWAIReviewLevel(
	courseware *models.Courseware,
	reviewLevel int,
) error {
	if courseware == nil {
		return ErrCWAIReviewInvalidLevel
	}

	// 作者自审只允许在课件完成生成且仍可修改时执行。
	if reviewLevel == models.CWAIReviewLevelSelf {
		switch strings.TrimSpace(courseware.PublishState) {
		case "",
			"private",
			"published_personal",
			"revision":
		default:
			return ErrCWAIReviewInvalidLevel
		}

		switch strings.TrimSpace(courseware.Status) {
		case "preview", "confirmed":
			return nil
		default:
			return ErrCWAIReviewInvalidLevel
		}
	}

	if courseware.PublishState != models.CWPublishSubmitted {
		return ErrCWAIReviewInvalidLevel
	}

	switch reviewLevel {
	case models.ReviewLevelL1:
		if courseware.ReviewLevel != 0 {
			return ErrCWAIReviewInvalidLevel
		}

	case models.ReviewLevelL2:
		if courseware.ReviewLevel != models.ReviewLevelL1 {
			return ErrCWAIReviewInvalidLevel
		}

	default:
		return ErrCWAIReviewInvalidLevel
	}

	return nil
}

// cwAIReviewHash 返回与数据库约束一致的UTF-8字节SHA-256。
func cwAIReviewHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
