package services

// courseware_image_iaoci_generator.go — IAOCI条件锚点图片生成
//
// 规则：
//   - 数据库中的图片索引是生成事实源；
//   - 默认不传课程锚点原图；
//   - 锚点[A]始终通过文字提示词继承；
//   - 只有本图[C]明确引用锚点C1/A1/O1时才传锚点原图；
//   - 显式R关系参考图优先于课程锚点图；
//   - 一次调用只生成一个槽位的一张图片；
//   - 本生成器服务于全自动装配IAOCI图片流水线，
//     统一使用courseware_auto_assembly_image计费节点。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// GenerateImageIAOCIRequest IAOCI单槽位生图请求。
type GenerateImageIAOCIRequest struct {
	CoursewareID        string
	PageNumber          int
	PlaceholderID       string
	ImageKey            string
	Prompt              string
	Size                string
	RelationRefImageURL string
	Actor               *CoursewareActorContext
}

// GenerateImageFromIAOCI 按已落库的单槽位IAOCI生成图片。
func (s *CoursewareAssetService) GenerateImageFromIAOCI(
	ctx context.Context,
	request *GenerateImageIAOCIRequest,
) (*GenerateImageServiceResponse, error) {
	if request == nil {
		return nil, ErrCoursewareActorRequired
	}

	courseware, scopedActor, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			request.CoursewareID,
			request.Actor,
		)
	if err != nil {
		return nil, err
	}

	page, err := repository.GetCoursewarePageByNumber(
		ctx,
		request.CoursewareID,
		request.PageNumber,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"页面不存在: 课件=%s 页码=%d",
			request.CoursewareID,
			request.PageNumber,
		)
	}

	imageIndex, err :=
		repository.GetCoursewareImageIndexByKey(
			ctx,
			request.CoursewareID,
			request.ImageKey,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"图片IAOCI索引不存在: %w",
			err,
		)
	}

	if imageIndex.PageID == nil ||
		*imageIndex.PageID != page.ID {
		return nil, fmt.Errorf(
			"图片IAOCI索引不属于当前页面",
		)
	}

	if imageIndex.PlaceholderID !=
		strings.TrimSpace(
			request.PlaceholderID,
		) {
		return nil, fmt.Errorf(
			"图片IAOCI索引与placeholder_id不一致",
		)
	}

	imageAOCI, err :=
		utils.ParseImageAOCI(
			imageIndex.AOCIText,
		)
	if err != nil {
		s.markImageIAOCIFailed(
			ctx,
			request.CoursewareID,
			request.ImageKey,
			err,
		)

		return nil, fmt.Errorf(
			"数据库中的图片IAOCI无效: %w",
			err,
		)
	}

	if imageAOCI.ImageKey !=
		request.ImageKey {
		err := fmt.Errorf(
			"图片索引中的image_key与IAOCI不一致",
		)

		s.markImageIAOCIFailed(
			ctx,
			request.CoursewareID,
			request.ImageKey,
			err,
		)

		return nil, err
	}

	// 正式调用图片模型前，先把单槽位状态切到generating。
	if err := repository.UpdateCoursewareImageIndexAssetStatus(
		ctx,
		imageIndex.ID,
		nil,
		models.CWImageIndexStatusGenerating,
		"",
	); err != nil {
		return nil, fmt.Errorf(
			"更新图片槽位生成状态失败: %w",
			err,
		)
	}

	anchorAOCI :=
		cwParseCoursewareAnchorAOCI(
			courseware,
		)

	effectivePrompt :=
		strings.TrimSpace(
			imageIndex.GenerationPrompt,
		)

	if effectivePrompt == "" {
		effectivePrompt =
			strings.TrimSpace(
				request.Prompt,
			)
	}

	if effectivePrompt == "" {
		effectivePrompt =
			cwCompileImageGenerationPrompt(
				imageAOCI,
				anchorAOCI,
			)
	}

	imageSize :=
		strings.TrimSpace(
			request.Size,
		)

	if imageSize == "" {
		imageSize =
			cwImageAOCISize(
				imageAOCI,
			)
	}

	referenceURL :=
		normalizeIAOCIReferenceURL(
			request.RelationRefImageURL,
		)

	usedRelationReference :=
		referenceURL != ""

	usedAnchorSubjectReference := false

	if referenceURL == "" &&
		anchorAOCI != nil &&
		cwImageAOCIUsesAnchorSubject(
			imageAOCI,
			anchorAOCI,
		) {
		referenceURL =
			s.resolveCoursewareAnchorSubjectReference(
				ctx,
				courseware,
			)

		usedAnchorSubjectReference =
			referenceURL != ""
	}

	imageConfig, err :=
		ai.GetImageConfig(
			s.cfg.GetAESKey(),
		)
	if err != nil {
		s.markImageIAOCIFailed(
			ctx,
			request.CoursewareID,
			request.ImageKey,
			err,
		)

		return nil, fmt.Errorf(
			"图片生成API未配置: %w",
			err,
		)
	}

	userID := scopedActor.UserID

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	traceContext := &ai.TraceContext{
		SceneCode: "courseware_image_gen",
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, asset, err := executeBilledCoursewareImage(
		ctx,
		&coursewareImageBillingInput{
			UserID:          userID,
			SchoolID:        schoolIDPtr(schoolID),
			BillingNodeCode: "courseware_auto_assembly_image",
			CoursewareID:    request.CoursewareID,
			PageID:          &page.ID,
			ModelName:       imageConfig.Model,
			IdempotencyKey: fmt.Sprintf(
				"courseware-image:iaoci:%s:v%d",
				imageIndex.ID,
				imageIndex.Version,
			),
			Metadata: map[string]interface{}{
				"image_index_id":                imageIndex.ID,
				"image_index_version":           imageIndex.Version,
				"image_key":                     request.ImageKey,
				"placeholder_id":                request.PlaceholderID,
				"used_relation_reference":       usedRelationReference,
				"used_anchor_subject_reference": usedAnchorSubjectReference,
				"requested_image_size":          imageSize,
				"has_reference_image":           referenceURL != "",
			},
		},
		func() (*ai.ImageGenerateResult, error) {
			return ai.GenerateImage(
				ctx,
				imageConfig,
				effectivePrompt,
				imageSize,
				1,
				referenceURL,
				traceContext,
			)
		},
		func(
			generated *ai.ImageGenerateResult,
		) (*models.CoursewareAsset, error) {
			saved, saveErr := s.downloadAndSaveImageWithMetadata(
				ctx,
				request.CoursewareID,
				request.PageNumber,
				generated.URLs[0],
				imageAOCI.FocusText,
			)
			if saveErr != nil {
				return nil, fmt.Errorf(
					"下载生成图片失败: %w",
					saveErr,
				)
			}

			if metadataErr :=
				validateCoursewareGeneratedImageFile(
					saved,
				); metadataErr != nil {
				return nil,
					fmt.Errorf(
						"生成图片文件元数据无效: %w",
						metadataErr,
					)
			}

			persisted := &models.CoursewareAsset{
				CoursewareID: request.CoursewareID,
				PageID:       &page.ID,
				PlaceholderID: strings.TrimSpace(
					request.PlaceholderID,
				),
				AssetType:        models.CWAssetTypeImage,
				GenerationPrompt: effectivePrompt,
				OssURL:           saved.URL,
				FileSize:         saved.FileSize,
				MimeType:         saved.MimeType,
				Status:           models.CWAssetStatusUploaded,
			}

			if createErr := repository.CreateCWAsset(
				ctx,
				persisted,
			); createErr != nil {
				return nil, fmt.Errorf(
					"记录图片资产失败: %w",
					createErr,
				)
			}

			return persisted, nil
		},
	)
	if err != nil {
		// 同一稳定幂等键的并发重放只返回“处理中”，
		// 不得把首个仍在生成的图片索引误标记为失败。
		if !errors.Is(
			err,
			ErrCoursewareImageBillingInProgress,
		) {
			s.markImageIAOCIFailed(
				ctx,
				request.CoursewareID,
				request.ImageKey,
				err,
			)
		}

		return nil, fmt.Errorf(
			"图片生成或资产保存失败: %w",
			err,
		)
	}

	localURL := asset.OssURL

	if err :=
		repository.UpdateCoursewareImageIndexAssetStatus(
			ctx,
			imageIndex.ID,
			&asset.ID,
			models.CWImageIndexStatusGenerated,
			"",
		); err != nil {
		return nil, fmt.Errorf(
			"绑定图片资产到IAOCI索引失败: %w",
			err,
		)
	}

	cwAssetLog.Info(
		"IAOCI单槽位图片生成成功",
		"courseware_id", request.CoursewareID,
		"page_number", request.PageNumber,
		"placeholder_id", request.PlaceholderID,
		"image_key", request.ImageKey,
		"asset_id", asset.ID,
		"billing_node_code",
		"courseware_auto_assembly_image",
		"model", result.ModelUsed,
		"used_relation_reference",
		usedRelationReference,
		"used_anchor_subject_reference",
		usedAnchorSubjectReference,
	)

	return &GenerateImageServiceResponse{
		AssetID:       asset.ID,
		URL:           localURL,
		OriginalURLs:  result.URLs,
		ModelUsed:     result.ModelUsed,
		RevisedPrompt: result.RevisedPrompt,
	}, nil
}

func normalizeIAOCIReferenceURL(
	value string,
) string {
	value = strings.TrimSpace(value)

	if strings.HasPrefix(
		value,
		"/uploads/",
	) {
		return cwAssetPublicHost + value
	}

	if strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") {
		return value
	}

	return ""
}

// resolveCoursewareAnchorSubjectReference 只在明确复用锚点主体时解析锚点图。
func (s *CoursewareAssetService) resolveCoursewareAnchorSubjectReference(
	ctx context.Context,
	courseware *models.Courseware,
) string {
	if courseware == nil ||
		courseware.StyleAnchorAssetID == nil ||
		strings.TrimSpace(
			*courseware.StyleAnchorAssetID,
		) == "" {
		return ""
	}

	anchorAsset, err :=
		repository.GetCWAssetByID(
			ctx,
			*courseware.StyleAnchorAssetID,
		)
	if err != nil ||
		anchorAsset == nil {
		return ""
	}

	if anchorAsset.CoursewareID !=
		courseware.ID ||
		anchorAsset.AssetType !=
			models.CWAssetTypeImage {
		return ""
	}

	return resolveAssetPublicURL(
		anchorAsset,
	)
}

func (s *CoursewareAssetService) markImageIAOCIFailed(
	ctx context.Context,
	coursewareID string,
	imageKey string,
	failure error,
) {
	index, err :=
		repository.GetCoursewareImageIndexByKey(
			ctx,
			coursewareID,
			imageKey,
		)
	if err != nil ||
		index == nil {
		return
	}

	message := "图片生成失败"

	if failure != nil {
		message = failure.Error()
	}

	_ = repository.UpdateCoursewareImageIndexAssetStatus(
		ctx,
		index.ID,
		nil,
		models.CWImageIndexStatusFailed,
		message,
	)
}
