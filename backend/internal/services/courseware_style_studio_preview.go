package services

// courseware_style_studio_preview.go — AI美术风格三类预览
//
// 固定生成三张测试图：
//   - character：人物情境图；
//   - object：知识对象图；
//   - diagram：教学图解图。
//
// 安全规则：
//   - 三张图都只继承当前IAOCI的[A]艺术风格；
//   - 人物图只有在style_character且存在固定[C]时，才使用参考图；
//   - 知识对象和教学图解永远不传人物参考图；
//   - 每张图都是独立课程级图片资产，page_id为NULL；
//   - 单张失败不会复制其它预览图；
//   - 新客户端可显式提交reference_mode，服务端会先持久化模式，
//     再根据该模式规范化IAOCI并生成全部预览。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// GeneratePreviews 保留旧内部调用签名。
//
// 旧调用不提交模式时继续使用会话当前模式。
func (s *CoursewareStyleStudioService) GeneratePreviews(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	actor *CoursewareActorContext,
) (*models.CoursewareStyleStudioState, error) {
	return s.GeneratePreviewsWithRequest(
		ctx,
		coursewareID,
		sessionID,
		nil,
		actor,
	)
}

// GeneratePreviewsWithRequest 按请求显式模式生成或重新生成三类预览。
func (s *CoursewareStyleStudioService) GeneratePreviewsWithRequest(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	request *models.GenerateCoursewareStylePreviewsRequest,
	actor *CoursewareActorContext,
) (*models.CoursewareStyleStudioState, error) {
	courseware, scopedActor, err :=
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

	requestedMode := ""
	operationID := ""

	if request != nil {
		requestedMode =
			request.ReferenceMode

		operationID =
			strings.TrimSpace(
				request.OperationID,
			)
	}

	// 新客户端一次点击只生成一个operation_id。
	// 旧客户端缺省时由服务端补UUID，保持兼容但不信任任意长字符串。
	if operationID == "" {
		operationID =
			uuid.NewString()
	} else {
		parsedOperationID, parseErr :=
			uuid.Parse(
				operationID,
			)
		if parseErr != nil {
			return nil, fmt.Errorf(
				"风格预览operation_id不合法",
			)
		}

		operationID =
			parsedOperationID.String()
	}

	referenceMode, err :=
		resolveStyleStudioReferenceMode(
			session.ReferenceMode,
			requestedMode,
		)
	if err != nil {
		return nil, err
	}

	styleAOCIText, styleAOCI, err :=
		normalizeStyleStudioAOCIForMode(
			session.StyleAOCIText,
			referenceMode,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"请先通过文字或参考图片形成有效美术风格: %w",
			err,
		)
	}

	// 保存规范化后的安全版本和当前显式模式，
	// 避免预览、恢复和最终确认使用不同reference_mode。
	session, err =
		repository.SaveCoursewareStyleSessionDraft(
			ctx,
			courseware.ID,
			session.ID,
			courseware.UserID,
			models.CWStyleSessionStatusPreviewing,
			referenceMode,
			session.ReferenceAssetID,
			styleAOCIText,
			buildStyleStudioSummary(
				styleAOCI,
			),
		)
	if err != nil {
		return nil, err
	}

	imageConfig, err :=
		ai.GetImageConfig(
			s.cfg.GetAESKey(),
		)
	if err != nil {
		return nil, fmt.Errorf(
			"图片生成API未配置: %w",
			err,
		)
	}

	characterReferenceURL :=
		s.resolveStyleStudioCharacterReference(
			ctx,
			courseware,
			session,
			styleAOCI,
		)

	traceUserID :=
		scopedActor.UserID

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			traceUserID,
		)

	traceContext := &ai.TraceContext{
		SceneCode: "courseware_style_preview",
		UserID:    &traceUserID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	successCount := 0
	failures := make([]string, 0)

	for _, previewType := range models.CoursewareStylePreviewTypes {
		prompt :=
			buildStyleStudioPreviewPrompt(
				courseware,
				styleAOCI,
				previewType,
			)

		preview := &models.CoursewareStylePreview{
			SessionID:        session.ID,
			CoursewareID:     courseware.ID,
			PreviewType:      previewType,
			GenerationPrompt: prompt,
			Status: models.
				CWStylePreviewStatusGenerating,
			Version: 1,
		}

		if err :=
			repository.UpsertCoursewareStylePreview(
				ctx,
				courseware.UserID,
				preview,
			); err != nil {
			return nil, err
		}

		referenceURL := ""
		if previewType ==
			models.CWStylePreviewTypeCharacter {
			referenceURL =
				characterReferenceURL
		}

		billingNodeCode :=
			styleStudioPreviewBillingNodeCode(
				previewType,
			)

		result, asset, generationErr :=
			executeBilledCoursewareImage(
				ctx,
				&coursewareImageBillingInput{
					UserID:          traceUserID,
					SchoolID:        schoolIDPtr(schoolID),
					BillingNodeCode: billingNodeCode,
					CoursewareID:    courseware.ID,
					ModelName:       imageConfig.Model,
					IdempotencyKey: fmt.Sprintf(
						"courseware-image:style-preview:%s:%s:%s",
						session.ID,
						operationID,
						previewType,
					),
					Metadata: map[string]interface{}{
						"style_session_id":  session.ID,
						"style_preview_id":  preview.ID,
						"preview_version":   preview.Version,
						"preview_operation": operationID,
						"preview_type":      previewType,
						"reference_mode":    referenceMode,
						"used_reference":    referenceURL != "",
					},
				},
				func() (*ai.ImageGenerateResult, error) {
					return ai.GenerateImage(
						ctx,
						imageConfig,
						prompt,
						"2560x1440",
						1,
						referenceURL,
						traceContext,
					)
				},
				func(
					generated *ai.ImageGenerateResult,
				) (*models.CoursewareAsset, error) {
					return s.saveStyleStudioGeneratedImage(
						ctx,
						courseware.ID,
						session.ID,
						previewType,
						generated.URLs[0],
						prompt,
					)
				},
			)

		if generationErr != nil ||
			result == nil ||
			asset == nil {
			failureMessage :=
				"图片模型未返回有效预览"

			if generationErr != nil {
				failureMessage =
					generationErr.Error()
			}

			// 同一operation_id并发重放时，首个请求仍在生成。
			// 此时保留generating状态，禁止第二个请求误写failed。
			if errors.Is(
				generationErr,
				ErrCoursewareImageBillingInProgress,
			) {
				failures = append(
					failures,
					previewType+
						"："+
						failureMessage,
				)

				continue
			}

			_, _ =
				repository.UpdateCoursewareStylePreviewStatus(
					ctx,
					courseware.UserID,
					courseware.ID,
					session.ID,
					previewType,
					models.CWStylePreviewStatusFailed,
					nil,
					failureMessage,
				)

			failures = append(
				failures,
				previewType+
					"："+
					failureMessage,
			)

			styleStudioLog.Warn(
				"课程美术风格预览生成失败",
				"courseware_id", courseware.ID,
				"session_id", session.ID,
				"preview_type", previewType,
				"reference_mode", referenceMode,
				"error", generationErr,
			)

			continue
		}

		assetID := asset.ID

		if _, err :=
			repository.UpdateCoursewareStylePreviewStatus(
				ctx,
				courseware.UserID,
				courseware.ID,
				session.ID,
				previewType,
				models.CWStylePreviewStatusGenerated,
				&assetID,
				"",
			); err != nil {
			return nil, err
		}

		successCount++

		styleStudioLog.Info(
			"课程美术风格预览生成成功",
			"courseware_id", courseware.ID,
			"session_id", session.ID,
			"preview_type", previewType,
			"asset_id", asset.ID,
			"reference_mode", referenceMode,
			"used_reference",
			referenceURL != "",
			"model", result.ModelUsed,
		)
	}

	state, stateErr :=
		s.loadStyleStudioState(
			ctx,
			courseware,
			session,
		)
	if stateErr != nil {
		return nil, stateErr
	}

	if successCount == 0 {
		return state, fmt.Errorf(
			"三类美术风格预览全部生成失败：%s",
			strings.Join(
				failures,
				"；",
			),
		)
	}

	if len(failures) > 0 {
		styleStudioLog.Warn(
			"课程美术风格预览部分失败",
			"courseware_id", courseware.ID,
			"session_id", session.ID,
			"reference_mode", referenceMode,
			"success_count", successCount,
			"failures",
			strings.Join(
				failures,
				"；",
			),
		)
	}

	return state, nil
}

func styleStudioPreviewBillingNodeCode(
	previewType string,
) string {
	switch previewType {
	case models.CWStylePreviewTypeCharacter:
		return "style_studio_character_preview"
	case models.CWStylePreviewTypeObject:
		return "style_studio_object_preview"
	case models.CWStylePreviewTypeDiagram:
		return "style_studio_diagram_preview"
	default:
		return ""
	}
}

func buildStyleStudioPreviewPrompt(
	courseware *models.Courseware,
	styleAOCI *models.ImageAOCI,
	previewType string,
) string {
	var builder strings.Builder

	builder.WriteString(
		"这是课程美术风格测试图，不是正式课件页面。\n",
	)

	builder.WriteString(
		fmt.Sprintf(
			"课程背景：%s，%s，%s。\n",
			courseware.Subject,
			courseware.Grade,
			courseware.Title,
		),
	)

	builder.WriteString(
		"【统一艺术风格】",
	)
	builder.WriteString(
		styleAOCI.ArtText,
	)
	builder.WriteString("\n")

	switch previewType {
	case models.CWStylePreviewTypeCharacter:
		builder.WriteString(
			"【测试任务】生成一幅自然、亲和的教师与学生共同观察学习材料的教学情境图，用于检验人物造型、表情、材质和整体氛围。不得出现可读文字、Logo、课件界面或固定页面布局。\n",
		)

		if !isStyleStudioEmptySemantic(
			styleAOCI.CharacterText,
		) {
			builder.WriteString(
				"【固定主体】",
			)
			builder.WriteString(
				styleAOCI.CharacterText,
			)
			builder.WriteString(
				"。只保持身份和外貌，不继承参考图动作、场景、位置或镜头。\n",
			)
		} else {
			builder.WriteString(
				"【人物规则】使用普通、自然、非固定身份的教师和学生，不建立全课程固定角色。\n",
			)
		}

	case models.CWStylePreviewTypeObject:
		builder.WriteString(
			"【测试任务】生成一组清晰的教学知识对象静物组合，包括一本书、叶片、地球仪、尺子和简单实验器材，用于检验非人物主体、材质、色彩和细节表现。画面不得出现人物，不得出现可读文字、Logo或水印。\n",
		)

	case models.CWStylePreviewTypeDiagram:
		builder.WriteString(
			"【测试任务】生成一幅清晰的三步骤教学流程图解，用简单形状、箭头、颜色层级和无文字图标表达过程，用于检验图解清晰度和知识表达能力。不得出现人物，不得生成可读文字、Logo或水印，装饰不能影响信息层级。\n",
		)
	}

	builder.WriteString(
		"【输出要求】横向16:9，主体清晰，留白合理，适合教师在课件中继续使用。\n",
	)

	builder.WriteString(
		"【禁止项】",
	)
	builder.WriteString(
		styleAOCI.NegativeText,
	)
	builder.WriteString(
		"；禁止照搬任何参考图片的具体环境、构图、镜头、主体位置、文字、Logo和水印。",
	)

	return builder.String()
}

func (s *CoursewareStyleStudioService) resolveStyleStudioCharacterReference(
	ctx context.Context,
	courseware *models.Courseware,
	session *models.CoursewareStyleSession,
	styleAOCI *models.ImageAOCI,
) string {
	if courseware == nil ||
		session == nil ||
		styleAOCI == nil {
		return ""
	}

	if session.ReferenceMode !=
		models.CWStyleReferenceModeCharacter {
		return ""
	}

	if isStyleStudioEmptySemantic(
		styleAOCI.CharacterText,
	) {
		return ""
	}

	if session.ReferenceAssetID == nil ||
		strings.TrimSpace(
			*session.ReferenceAssetID,
		) == "" {
		return ""
	}

	asset, err :=
		s.loadStyleStudioImageAsset(
			ctx,
			courseware.ID,
			*session.ReferenceAssetID,
		)
	if err != nil {
		styleStudioLog.Warn(
			"读取人物风格参考图失败，预览降级为纯文字生成",
			"courseware_id", courseware.ID,
			"session_id", session.ID,
			"error", err,
		)
		return ""
	}

	return resolveAssetPublicURL(asset)
}

// parseStyleStudioPreviewAOCI 提供定向测试使用。
func parseStyleStudioPreviewAOCI(
	value string,
) (*models.ImageAOCI, error) {
	parsed, err :=
		utils.ParseImageAOCI(value)
	if err != nil {
		return nil, err
	}

	if parsed.IndexType !=
		models.CWImageIndexTypeAnchor {
		return nil, fmt.Errorf(
			"风格预览只能使用课程锚点IAOCI",
		)
	}

	return parsed, nil
}
