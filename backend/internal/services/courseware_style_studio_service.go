package services

// courseware_style_studio_service.go — AI美术风格工作室核心服务
//
// 本文件负责：
//   - 创建新的风格共创会话；
//   - 恢复当前活动会话；
//   - 读取指定会话完整状态；
//   - 归档未完成会话；
//   - 确认课程正式风格锚点；
//   - 统一解析前端显式提交的参考图模式；
//   - 统一执行作者权限和课件教育域校验。
//
// AI对话、参考图上传和三类预览分别位于独立服务文件。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// CoursewareStyleStudioService AI美术风格工作室服务。
type CoursewareStyleStudioService struct {
	cfg          *config.Config
	assetService *CoursewareAssetService
	ossService   *OSSService
}

// NewCoursewareStyleStudioService 创建风格工作室服务。
func NewCoursewareStyleStudioService(
	cfg *config.Config,
	assetService *CoursewareAssetService,
	ossService *OSSService,
) *CoursewareStyleStudioService {
	if assetService == nil {
		assetService =
			NewCoursewareAssetService(cfg)
	}

	if ossService == nil {
		ossService =
			NewOSSService(cfg)
	}

	return &CoursewareStyleStudioService{
		cfg:          cfg,
		assetService: assetService,
		ossService:   ossService,
	}
}

// CreateSession 创建新的风格共创会话。
//
// 创建时会由仓储自动归档同一课件原有的draft或previewing会话。
func (s *CoursewareStyleStudioService) CreateSession(
	ctx context.Context,
	coursewareID string,
	request *models.CreateCoursewareStyleSessionRequest,
	actor *CoursewareActorContext,
) (*models.CoursewareStyleStudioState, error) {
	courseware, _, err :=
		s.loadStyleStudioCourseware(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	referenceMode :=
		models.CWStyleReferenceModeStyleOnly

	var referenceAssetID *string

	if request != nil {
		if strings.TrimSpace(
			request.ReferenceMode,
		) != "" {
			referenceMode =
				strings.TrimSpace(
					request.ReferenceMode,
				)
		}

		referenceAssetID =
			normalizeStyleStudioStringPointer(
				request.ReferenceAssetID,
			)
	}

	if !models.IsValidCWStyleReferenceMode(
		referenceMode,
	) {
		return nil, fmt.Errorf(
			"参考图模式不合法: %s",
			referenceMode,
		)
	}

	if referenceAssetID != nil {
		if _, err :=
			s.loadStyleStudioImageAsset(
				ctx,
				courseware.ID,
				*referenceAssetID,
			); err != nil {
			return nil, err
		}
	}

	session := &models.CoursewareStyleSession{
		CoursewareID: courseware.ID,
		UserID:       courseware.UserID,
		Status: models.
			CWStyleSessionStatusDraft,
		ReferenceMode:    referenceMode,
		ReferenceAssetID: referenceAssetID,
		Version:          1,
	}

	if err :=
		repository.CreateCoursewareStyleSession(
			ctx,
			session,
		); err != nil {
		return nil, err
	}

	styleStudioLog.Info(
		"创建课程美术风格会话",
		"courseware_id", courseware.ID,
		"session_id", session.ID,
		"reference_mode", referenceMode,
		"has_reference",
		referenceAssetID != nil,
	)

	return s.loadStyleStudioState(
		ctx,
		courseware,
		session,
	)
}

// GetActiveState 恢复当前活动风格会话。
//
// 当前课件还没有活动会话时返回空状态，而不是返回错误。
func (s *CoursewareStyleStudioService) GetActiveState(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.CoursewareStyleStudioState, error) {
	courseware, _, err :=
		s.loadStyleStudioCourseware(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	session, err :=
		repository.GetActiveCoursewareStyleSession(
			ctx,
			courseware.ID,
			courseware.UserID,
		)
	if errors.Is(
		err,
		repository.ErrCoursewareStyleSessionNotFound,
	) {
		return &models.CoursewareStyleStudioState{
			Session:  nil,
			Messages: []*models.CoursewareStyleMessage{},
			Previews: []*models.CoursewareStylePreview{},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return s.loadStyleStudioState(
		ctx,
		courseware,
		session,
	)
}

// GetState 读取指定风格会话完整状态。
func (s *CoursewareStyleStudioService) GetState(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	actor *CoursewareActorContext,
) (*models.CoursewareStyleStudioState, error) {
	courseware, _, err :=
		s.loadStyleStudioCourseware(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	session, err :=
		repository.GetCoursewareStyleSessionByID(
			ctx,
			courseware.ID,
			strings.TrimSpace(sessionID),
			courseware.UserID,
		)
	if err != nil {
		return nil, err
	}

	return s.loadStyleStudioState(
		ctx,
		courseware,
		session,
	)
}

// ArchiveSession 归档未确认的风格会话。
func (s *CoursewareStyleStudioService) ArchiveSession(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	actor *CoursewareActorContext,
) error {
	courseware, _, err :=
		s.loadStyleStudioCourseware(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	if err :=
		repository.ArchiveCoursewareStyleSession(
			ctx,
			courseware.ID,
			strings.TrimSpace(sessionID),
			courseware.UserID,
		); err != nil {
		return err
	}

	styleStudioLog.Info(
		"归档课程美术风格会话",
		"courseware_id", courseware.ID,
		"session_id", sessionID,
	)

	return nil
}

// ConfirmSession 确认当前IAOCI和选定图片为课程正式风格锚点。
//
// 安全约束最终由仓储事务再次执行：
//   - 当前会话已成功生成的预览图可以确认；
//   - 仅style_character可确认当前会话参考图；
//   - 其它同课件图片不可被借用为本会话锚点；
//   - 模式变化后旧模式预览不可直接确认。
func (s *CoursewareStyleStudioService) ConfirmSession(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	request *models.ConfirmCoursewareStyleSessionRequest,
	actor *CoursewareActorContext,
) (*models.CoursewareStyleStudioState, error) {
	if request == nil ||
		strings.TrimSpace(
			request.AssetID,
		) == "" {
		return nil, fmt.Errorf(
			"确认风格必须选择参考图或预览图",
		)
	}

	courseware, _, err :=
		s.loadStyleStudioCourseware(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	session, err :=
		repository.GetCoursewareStyleSessionByID(
			ctx,
			courseware.ID,
			strings.TrimSpace(sessionID),
			courseware.UserID,
		)
	if err != nil {
		return nil, err
	}

	if !models.IsEditableCWStyleSessionStatus(
		session.Status,
	) {
		return nil,
			repository.
				ErrCoursewareStyleSessionNotEditable
	}

	referenceMode, err :=
		resolveStyleStudioReferenceMode(
			session.ReferenceMode,
			request.ReferenceMode,
		)
	if err != nil {
		return nil, err
	}

	assetID :=
		strings.TrimSpace(request.AssetID)

	if _, err :=
		s.loadStyleStudioImageAsset(
			ctx,
			courseware.ID,
			assetID,
		); err != nil {
		return nil, err
	}

	styleAOCIText, parsedAOCI, err :=
		normalizeStyleStudioAOCIForMode(
			session.StyleAOCIText,
			referenceMode,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"当前风格草稿无效，不能确认: %w",
			err,
		)
	}

	styleSummary :=
		strings.TrimSpace(
			session.StyleSummary,
		)
	if styleSummary == "" ||
		referenceMode !=
			session.ReferenceMode {
		styleSummary =
			buildStyleStudioSummary(
				parsedAOCI,
			)
	}

	confirmedSession, err :=
		repository.
			ConfirmCoursewareStyleSessionWithMode(
				ctx,
				courseware.ID,
				session.ID,
				courseware.UserID,
				assetID,
				referenceMode,
				styleAOCIText,
				styleSummary,
			)
	if err != nil {
		return nil, err
	}

	// 资产状态仅用于素材库展示。
	// 正式锚点与@ANCHOR的一致性已经由上面的数据库事务保证。
	_ = repository.UpdateCWAssetStatus(
		ctx,
		assetID,
		models.CWAssetStatusConfirmed,
	)

	styleStudioLog.Info(
		"确认课程AI定制美术风格",
		"courseware_id", courseware.ID,
		"session_id", session.ID,
		"asset_id", assetID,
		"reference_mode",
		referenceMode,
		"subject_type",
		parsedAOCI.SubjectType,
	)

	return s.loadStyleStudioState(
		ctx,
		courseware,
		confirmedSession,
	)
}

// resolveStyleStudioReferenceMode 解析当前会话模式和请求显式模式。
//
// 请求为空时兼容旧客户端。
// 请求非空时必须属于三个合法模式。
func resolveStyleStudioReferenceMode(
	currentMode string,
	requestedMode string,
) (string, error) {
	currentMode =
		strings.TrimSpace(currentMode)

	requestedMode =
		strings.TrimSpace(requestedMode)

	if requestedMode == "" {
		requestedMode = currentMode
	}

	if !models.IsValidCWStyleReferenceMode(
		requestedMode,
	) {
		return "", fmt.Errorf(
			"参考图模式不合法: %s",
			requestedMode,
		)
	}

	return requestedMode, nil
}

func (s *CoursewareStyleStudioService) loadStyleStudioCourseware(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (
	*models.Courseware,
	*CoursewareActorContext,
	error,
) {
	coursewareID =
		strings.TrimSpace(coursewareID)
	if coursewareID == "" {
		return nil, nil,
			fmt.Errorf("课件ID不能为空")
	}

	return (&CoursewareService{}).
		LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
}

func (s *CoursewareStyleStudioService) loadStyleStudioState(
	ctx context.Context,
	courseware *models.Courseware,
	session *models.CoursewareStyleSession,
) (*models.CoursewareStyleStudioState, error) {
	if courseware == nil ||
		session == nil {
		return nil, fmt.Errorf(
			"风格工作室状态对象为空",
		)
	}

	messages, err :=
		repository.ListCoursewareStyleMessages(
			ctx,
			courseware.ID,
			session.ID,
			courseware.UserID,
		)
	if err != nil {
		return nil, err
	}

	previews, err :=
		repository.ListCoursewareStylePreviews(
			ctx,
			courseware.ID,
			session.ID,
			courseware.UserID,
		)
	if err != nil {
		return nil, err
	}

	if messages == nil {
		messages =
			[]*models.CoursewareStyleMessage{}
	}

	if previews == nil {
		previews =
			[]*models.CoursewareStylePreview{}
	}

	return &models.CoursewareStyleStudioState{
		Session:  session,
		Messages: messages,
		Previews: previews,
	}, nil
}

func normalizeStyleStudioStringPointer(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	normalized :=
		strings.TrimSpace(*value)

	if normalized == "" {
		return nil
	}

	return &normalized
}

// validateStyleStudioConfirmedAOCI 提供给后续Handler或测试使用。
func validateStyleStudioConfirmedAOCI(
	value string,
) error {
	parsed, err :=
		utils.ParseImageAOCI(value)
	if err != nil {
		return err
	}

	if parsed.ImageKey != "@ANCHOR" ||
		parsed.IndexType !=
			models.CWImageIndexTypeAnchor ||
		len(parsed.Relations) != 0 {
		return fmt.Errorf(
			"正式风格锚点必须使用@ANCHOR、IT=A且无R关系",
		)
	}

	return nil
}
