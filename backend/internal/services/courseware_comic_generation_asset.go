package services

// courseware_comic_generation_asset.go — 漫画图片资产、参考图与统一积分计费
//
// 本文件负责：
//   - 生成人物设定参考图；
//   - 生成或重新生成一个漫画格；
//   - 所有真实图片调用统一执行积分预留、结算、释放和幂等恢复；
//   - 应用教师已确认的画风、比例、清晰度和补充要求；
//   - 优先使用已确认样张或上一格作为下一格参考图；
//   - 下载临时图片到课程级漫画目录；
//   - 创建page_id=NULL的courseware_assets记录；
//   - 写入漫画项目、分格、图片键、版本来源和参考资产元数据；
//   - 尽力上传OSS并回写稳定公网地址。
//
// 计费节点：
//   - comic_character_sheet：人物设定图；
//   - comic_panel_generate：漫画分格首次生成；
//   - comic_panel_regenerate：漫画分格重新生成。
//
// 幂等规则：
//   - 人物设定图使用项目ID和确认后渲染内容摘要构造稳定键；
//   - 漫画格使用分格ID和领取后的分格version构造稳定键；
//   - 供应商失败后，失败事务会增加分格version，下一次领取得到新键；
//   - 供应商成功但漫画格业务绑定失败时，分格保持generating且version不变，
//     下次执行恢复同一计费记录和原图片资产，不重复调用供应商。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// coursewareComicBillingDigest 构造短且稳定的漫画图片业务摘要。
func coursewareComicBillingDigest(
	values ...string,
) string {
	normalized := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		normalized = append(
			normalized,
			strings.TrimSpace(value),
		)
	}

	sum := sha256.Sum256(
		[]byte(
			strings.Join(
				normalized,
				"\x1f",
			),
		),
	)

	return hex.EncodeToString(
		sum[:12],
	)
}

// generateAndCompletePanel 调用统一图片计费状态机、保存资产并创建不可变版本。
//
// 返回值约定：
//   - asset=nil且error!=nil：供应商或资产持久化未形成可恢复资产；
//   - asset!=nil且error!=nil：图片资产已经形成并完成计费，
//     但漫画格版本或当前资产绑定失败；调用方不得把分格版本推进为failed，
//     应保留generating状态供下一次同键恢复。
func (s *CoursewareComicGenerationService) generateAndCompletePanel(
	ctx context.Context,
	courseware *models.Courseware,
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
	panel *models.CoursewareComicPanel,
	characterSheet *models.CoursewareAsset,
	previousAsset *models.CoursewareAsset,
	imageConfig *ai.ImageConfig,
	traceContext *ai.TraceContext,
	userID string,
	generationSource string,
	regenerationInstruction string,
) (*models.CoursewareAsset, error) {
	if courseware == nil ||
		project == nil ||
		workflow == nil ||
		panel == nil ||
		imageConfig == nil ||
		traceContext == nil {
		return nil,
			fmt.Errorf(
				"漫画图片生成上下文不完整",
			)
	}

	baseRenderPlan, valid :=
		buildCoursewareComicConfirmedPanelRenderPlan(
			project,
			panel,
			workflow,
		)

	if !valid ||
		baseRenderPlan == nil {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	renderPlan :=
		baseRenderPlan

	regenerationInstruction =
		strings.TrimSpace(
			regenerationInstruction,
		)

	if generationSource ==
		models.CWComicVersionSourceRegenerate {
		// 教师主动单格重画会携带非空要求；
		// 整批流程对已有旧图片分格的生成或恢复允许为空，
		// 此时辅助函数返回基础渲染计划副本。
		renderPlan, valid =
			applyCoursewareComicPanelRegenerationInstruction(
				baseRenderPlan,
				regenerationInstruction,
			)

		if !valid ||
			renderPlan == nil {
			return nil,
				ErrCoursewareComicWorkflowInvalidRequest
		}
	} else {
		regenerationInstruction =
			""
	}

	// 已确认样张或上一格优先于人物设定图，
	// 使后续分格直接继承老师刚确认的视觉语言和构图质感。
	referenceAsset :=
		previousAsset

	referenceRole :=
		"previous_panel"

	if referenceAsset == nil {
		referenceAsset =
			characterSheet

		referenceRole =
			"character_sheet"
	}

	referenceURL :=
		resolveAssetPublicURL(
			referenceAsset,
		)

	actualReferenceRole :=
		"none"

	referenceAssetID := ""

	if referenceURL != "" {
		actualReferenceRole =
			referenceRole

		if referenceAsset != nil {
			referenceAssetID =
				strings.TrimSpace(
					referenceAsset.ID,
				)
		}
	}

	billingNodeCode :=
		"comic_panel_generate"

	if generationSource ==
		models.CWComicVersionSourceRegenerate {
		billingNodeCode =
			"comic_panel_regenerate"
	}

	// 幂等键只使用已领取分格版本和数据库可重建的稳定事实。
	//
	// 本次教师微调文字不得进入幂等键：
	//   - 正常失败后仓储会增加分格version，下一次要求自然得到新键；
	//   - 供应商已成功但业务绑定失败时，分格保持原version，
	//     即使浏览器刷新或教师修改输入，也必须恢复原计费资产，
	//     不能以新文字生成第二个键并重复调用供应商。
	idempotencyKey :=
		fmt.Sprintf(
			"courseware-image:%s:%s:v%d:%s",
			billingNodeCode,
			panel.ID,
			panel.Version,
			coursewareComicBillingDigest(
				panel.ID,
				fmt.Sprintf(
					"%d",
					panel.Version,
				),
				baseRenderPlan.Prompt,
				baseRenderPlan.ImageSize,
				generationSource,
				referenceAssetID,
			),
		)

	metadata :=
		map[string]interface{}{
			"comic_project_id":    project.ID,
			"comic_panel_id":      panel.ID,
			"panel_number":        panel.PanelNo,
			"panel_version":       panel.Version,
			"image_key":           panel.ImageKey,
			"generation_source":   generationSource,
			"visual_style_source": workflow.VisualStyleSource,
			"effective_style": coursewareComicEffectiveStyleMetadata(
				project,
				workflow,
			),
			"reference_role":       actualReferenceRole,
			"aspect_ratio":         workflow.AspectRatio,
			"image_quality":        workflow.ImageQuality,
			"requested_image_size": renderPlan.ImageSize,
		}

	if referenceAssetID != "" {
		metadata["reference_asset_id"] =
			referenceAssetID
	}

	if regenerationInstruction != "" {
		// 只记录存在本次微调要求，不保存原文或摘要。
		// 恢复同一已领取版本时，调用方可能提交不同文字；
		// 计费审计不能把恢复请求误记为最初实际生成要求。
		metadata["has_regeneration_instruction"] =
			true
	}

	s.broadcastGeneration(
		courseware.ID,
		"panel_generating",
		map[string]interface{}{
			"project_id":     project.ID,
			"panel_id":       panel.ID,
			"panel_no":       panel.PanelNo,
			"reference_role": actualReferenceRole,
			"image_size":     renderPlan.ImageSize,
			"aspect_ratio":   workflow.AspectRatio,
			"image_quality":  workflow.ImageQuality,
			"message": fmt.Sprintf(
				"正在生成第%d格漫画",
				panel.PanelNo,
			),
		},
	)

	_, asset, err :=
		executeBilledCoursewareImage(
			ctx,
			&coursewareImageBillingInput{
				UserID:          userID,
				SchoolID:        traceContext.SchoolID,
				BillingNodeCode: billingNodeCode,
				CoursewareID:    courseware.ID,
				ModelName:       imageConfig.Model,
				IdempotencyKey:  idempotencyKey,
				Metadata:        metadata,
			},
			func() (*ai.ImageGenerateResult, error) {
				return ai.GenerateImage(
					ctx,
					imageConfig,
					renderPlan.Prompt,
					renderPlan.ImageSize,
					1,
					referenceURL,
					traceContext,
				)
			},
			func(
				generated *ai.ImageGenerateResult,
			) (*models.CoursewareAsset, error) {
				return s.saveComicGeneratedImage(
					ctx,
					courseware.ID,
					project.ID,
					panel,
					"panel",
					generated.URLs[0],
					renderPlan.Prompt,
					generationSource,
					referenceAsset,
				)
			},
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"第%d格图片生成或资产保存失败: %w",
				panel.PanelNo,
				err,
			)
	}

	updatedPanel, err :=
		repository.CompleteCoursewareComicPanelGeneration(
			ctx,
			courseware.ID,
			project.ID,
			panel.ID,
			userID,
			asset.ID,
			panel.AOCIText,
			generationSource,
		)

	if err != nil {
		// 图片供应商已经产生费用，资产也已经持久化并绑定计费记录。
		// 返回非空asset，通知上层保留generating状态等待同键恢复。
		return asset, err
	}

	s.broadcastGeneration(
		courseware.ID,
		"panel_done",
		map[string]interface{}{
			"project_id":    project.ID,
			"panel_id":      updatedPanel.ID,
			"panel_no":      updatedPanel.PanelNo,
			"panel_version": updatedPanel.Version,
			"asset_id":      asset.ID,
			"asset_url":     asset.OssURL,
			"public_url": resolveAssetPublicURL(
				asset,
			),
			"image_size": renderPlan.ImageSize,
			"message": fmt.Sprintf(
				"第%d格漫画已生成",
				panel.PanelNo,
			),
		},
	)

	return asset, nil
}

// ensureCharacterSheet 读取已有设定图，缺失时通过统一图片计费生成。
//
// courseware模式必须加载有效课件风格锚点，禁止降级为纯文字或预设画风。
// selected模式完全不读取课件风格锚点，只使用教师选择的漫画画风。
func (s *CoursewareComicGenerationService) ensureCharacterSheet(
	ctx context.Context,
	courseware *models.Courseware,
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
	imageConfig *ai.ImageConfig,
	traceContext *ai.TraceContext,
) (*models.CoursewareAsset, error) {
	if courseware == nil ||
		project == nil ||
		workflow == nil ||
		imageConfig == nil ||
		traceContext == nil {
		return nil,
			fmt.Errorf(
				"人物设定图生成上下文不完整",
			)
	}

	visualStyleSource :=
		strings.TrimSpace(
			workflow.VisualStyleSource,
		)

	referenceURL := ""
	referenceAssetID := ""
	referenceRole := "none"

	switch visualStyleSource {
	case models.CWComicVisualStyleSourceCourseware:
		if courseware.StyleAnchorAssetID == nil ||
			strings.TrimSpace(
				*courseware.StyleAnchorAssetID,
			) == "" {
			return nil,
				fmt.Errorf(
					"%w: 跟随课件画风需要先为课件设置有效的风格锚点",
					ErrCoursewareComicWorkflowInvalidRequest,
				)
		}

		anchor, err :=
			s.loadComicImageAsset(
				ctx,
				courseware.ID,
				*courseware.StyleAnchorAssetID,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"%w: 课件风格锚点不可用",
					ErrCoursewareComicWorkflowInvalidRequest,
				)
		}

		referenceURL =
			resolveAssetPublicURL(
				anchor,
			)

		if referenceURL == "" {
			return nil,
				fmt.Errorf(
					"%w: 课件风格锚点缺少可用图片地址",
					ErrCoursewareComicWorkflowInvalidRequest,
				)
		}

		referenceAssetID =
			strings.TrimSpace(
				anchor.ID,
			)

		referenceRole =
			"courseware_style_anchor"

	case models.CWComicVisualStyleSourceSelected:
		// 严格selected模式不读取课件StyleAnchorAssetID。
		referenceURL = ""
		referenceAssetID = ""
		referenceRole = "selected_visual_style"

	default:
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	if existing :=
		s.loadProjectCharacterSheet(
			ctx,
			courseware.ID,
			project,
		); existing != nil {
		return existing, nil
	}

	renderPlan, valid :=
		buildCoursewareComicConfirmedCharacterSheetRenderPlan(
			project,
			workflow,
		)

	if !valid ||
		renderPlan == nil {
		return nil,
			ErrCoursewareComicWorkflowInvalidRequest
	}

	idempotencyKey :=
		fmt.Sprintf(
			"courseware-image:comic-character-sheet:%s:%s",
			project.ID,
			coursewareComicBillingDigest(
				project.ID,
				renderPlan.Prompt,
				renderPlan.ImageSize,
				workflow.ImageQuality,
				referenceAssetID,
			),
		)

	metadata :=
		map[string]interface{}{
			"comic_project_id":    project.ID,
			"project_version":     project.Version,
			"visual_style_source": visualStyleSource,
			"effective_style": coursewareComicEffectiveStyleMetadata(
				project,
				workflow,
			),
			"reference_role":       referenceRole,
			"image_quality":        workflow.ImageQuality,
			"requested_image_size": renderPlan.ImageSize,
			"has_reference_image":  referenceURL != "",
		}

	if referenceAssetID != "" {
		metadata["reference_asset_id"] =
			referenceAssetID
	}

	s.broadcastGeneration(
		courseware.ID,
		"character_sheet_generating",
		map[string]interface{}{
			"project_id":    project.ID,
			"image_size":    renderPlan.ImageSize,
			"image_quality": workflow.ImageQuality,
			"message":       "正在按确认画风生成人物设定参考图",
		},
	)

	_, asset, err :=
		executeBilledCoursewareImage(
			ctx,
			&coursewareImageBillingInput{
				UserID:          project.CreatedBy,
				SchoolID:        traceContext.SchoolID,
				BillingNodeCode: "comic_character_sheet",
				CoursewareID:    courseware.ID,
				ModelName:       imageConfig.Model,
				IdempotencyKey:  idempotencyKey,
				Metadata:        metadata,
			},
			func() (*ai.ImageGenerateResult, error) {
				return ai.GenerateImage(
					ctx,
					imageConfig,
					renderPlan.Prompt,
					renderPlan.ImageSize,
					1,
					referenceURL,
					traceContext,
				)
			},
			func(
				generated *ai.ImageGenerateResult,
			) (*models.CoursewareAsset, error) {
				return s.saveComicGeneratedImage(
					ctx,
					courseware.ID,
					project.ID,
					nil,
					"character_sheet",
					generated.URLs[0],
					renderPlan.Prompt,
					models.CWComicVersionSourceInitial,
					nil,
				)
			},
		)

	if err != nil {
		return nil, err
	}

	updatedProject, updateErr :=
		repository.UpdateCoursewareComicProjectCharacterSheet(
			ctx,
			courseware.ID,
			project.ID,
			project.CreatedBy,
			asset.ID,
			project.Version,
		)

	if updateErr != nil {
		// 计费资产已经形成。
		// 版本CAS冲突或瞬时写库失败时，再执行“仅在未绑定时补绑”的恢复操作。
		updatedProject, updateErr =
			repository.AttachCoursewareComicProjectCharacterSheetIfMissing(
				ctx,
				courseware.ID,
				project.ID,
				project.CreatedBy,
				asset.ID,
			)
	}

	if updateErr != nil {
		return asset,
			fmt.Errorf(
				"人物设定图已经生成，但项目资产绑定失败: %w",
				updateErr,
			)
	}

	*project =
		*updatedProject

	s.broadcastGeneration(
		courseware.ID,
		"character_sheet_done",
		map[string]interface{}{
			"project_id": project.ID,
			"asset_id":   asset.ID,
			"asset_url":  asset.OssURL,
			"public_url": resolveAssetPublicURL(
				asset,
			),
			"message": "人物设定参考图已生成",
		},
	)

	return asset, nil
}

func coursewareComicEffectiveStyleMetadata(
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
) string {
	if project == nil ||
		workflow == nil {
		return ""
	}

	if workflow.VisualStyleSource ==
		models.CWComicVisualStyleSourceCourseware {
		return "courseware_style_anchor"
	}

	return strings.TrimSpace(
		project.VisualStyle,
	)
}

// loadComicImageRuntime 加载图片网关配置和真实用户、学校追踪上下文。
func (s *CoursewareComicGenerationService) loadComicImageRuntime(
	ctx context.Context,
	userID string,
) (*ai.ImageConfig, *ai.TraceContext, error) {
	imageConfig, err :=
		ai.GetImageConfig(
			s.cfg.GetAESKey(),
		)

	if err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"图片生成API未配置: %w",
				err,
			)
	}

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	userIDSnapshot :=
		strings.TrimSpace(
			userID,
		)

	schoolIDSnapshot :=
		strings.TrimSpace(
			schoolID,
		)

	return imageConfig,
		&ai.TraceContext{
			SceneCode: "courseware_image_gen",
			UserID:    &userIDSnapshot,
			SchoolID: schoolIDPtr(
				schoolIDSnapshot,
			),
		},
		nil
}

// loadProjectCharacterSheet 返回项目当前有效人物设定图。
func (s *CoursewareComicGenerationService) loadProjectCharacterSheet(
	ctx context.Context,
	coursewareID string,
	project *models.CoursewareComicProject,
) *models.CoursewareAsset {
	if project == nil ||
		project.CharacterSheetAssetID == nil ||
		strings.TrimSpace(
			*project.CharacterSheetAssetID,
		) == "" {
		return nil
	}

	asset, err :=
		s.loadComicImageAsset(
			ctx,
			coursewareID,
			*project.CharacterSheetAssetID,
		)

	if err != nil {
		return nil
	}

	return asset
}

// findPreviousPanelAsset 返回目标格之前一格的有效当前图片。
func (s *CoursewareComicGenerationService) findPreviousPanelAsset(
	ctx context.Context,
	coursewareID string,
	panels []*models.CoursewareComicPanel,
	panelNo int,
) *models.CoursewareAsset {
	for _, candidate := range panels {
		if candidate == nil ||
			candidate.PanelNo !=
				panelNo-1 ||
			candidate.CurrentAssetID == nil {
			continue
		}

		asset, err :=
			s.loadComicImageAsset(
				ctx,
				coursewareID,
				*candidate.CurrentAssetID,
			)

		if err == nil {
			return asset
		}
	}

	return nil
}

// loadComicImageAsset 校验一个图片资产属于路径课件。
func (s *CoursewareComicGenerationService) loadComicImageAsset(
	ctx context.Context,
	coursewareID string,
	assetID string,
) (*models.CoursewareAsset, error) {
	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			strings.TrimSpace(
				assetID,
			),
		)

	if err != nil {
		return nil, err
	}

	if asset == nil ||
		asset.CoursewareID !=
			coursewareID ||
		asset.AssetType !=
			models.CWAssetTypeImage {
		return nil,
			repository.ErrCoursewareComicAssetInvalid
	}

	return asset, nil
}

// saveComicGeneratedImage 保存图片模型临时地址对应的课程级图片资产。
func (s *CoursewareComicGenerationService) saveComicGeneratedImage(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panel *models.CoursewareComicPanel,
	role string,
	remoteURL string,
	prompt string,
	generationSource string,
	referenceAsset *models.CoursewareAsset,
) (*models.CoursewareAsset, error) {
	remoteURL =
		strings.TrimSpace(
			remoteURL,
		)

	if remoteURL == "" {
		return nil,
			fmt.Errorf(
				"漫画图片远程地址为空",
			)
	}

	request, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			remoteURL,
			nil,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"创建漫画图片下载请求失败: %w",
				err,
			)
	}

	response, err :=
		(&http.Client{
			Timeout: 60 * time.Second,
		}).Do(
			request,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"下载漫画图片失败: %w",
				err,
			)
	}

	defer response.Body.Close()

	if response.StatusCode !=
		http.StatusOK {
		return nil,
			fmt.Errorf(
				"下载漫画图片HTTP错误: %d",
				response.StatusCode,
			)
	}

	mimeType :=
		strings.ToLower(
			strings.TrimSpace(
				strings.Split(
					response.Header.Get(
						"Content-Type",
					),
					";",
				)[0],
			),
		)

	extension :=
		".png"

	switch mimeType {
	case "image/jpeg",
		"image/jpg":
		extension =
			".jpg"

	case "image/webp":
		extension =
			".webp"

	default:
		mimeType =
			"image/png"
	}

	subDirectory :=
		"character-sheet"

	panelNumber :=
		0

	placeholderID :=
		"comic-character-sheet:" +
			projectID

	if panel != nil {
		panelNumber =
			panel.PanelNo

		subDirectory =
			filepath.Join(
				"panels",
				fmt.Sprintf(
					"p%02d",
					panel.PanelNo,
				),
			)

		placeholderID =
			fmt.Sprintf(
				"comic-panel:%s:%02d",
				projectID,
				panel.PanelNo,
			)
	}

	assetDirectory :=
		filepath.Join(
			CWAssetUploadDir,
			coursewareID,
			"comics",
			projectID,
			subDirectory,
		)

	if err :=
		os.MkdirAll(
			assetDirectory,
			0755,
		); err != nil {
		return nil,
			fmt.Errorf(
				"创建漫画图片目录失败: %w",
				err,
			)
	}

	storedName :=
		fmt.Sprintf(
			"%d_%s%s",
			time.Now().
				UnixMilli(),
			role,
			extension,
		)

	fullPath :=
		filepath.Join(
			assetDirectory,
			storedName,
		)

	destination, err :=
		os.Create(
			fullPath,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"创建漫画图片文件失败: %w",
				err,
			)
	}

	written, copyErr :=
		io.Copy(
			destination,
			io.LimitReader(
				response.Body,
				coursewareComicMaxGeneratedFileSize+1,
			),
		)

	closeErr :=
		destination.Close()

	if copyErr != nil ||
		closeErr != nil ||
		written >
			coursewareComicMaxGeneratedFileSize {
		_ =
			os.Remove(
				fullPath,
			)

		switch {
		case copyErr != nil:
			return nil,
				fmt.Errorf(
					"保存漫画图片失败: %w",
					copyErr,
				)

		case closeErr != nil:
			return nil,
				fmt.Errorf(
					"关闭漫画图片失败: %w",
					closeErr,
				)

		default:
			return nil,
				fmt.Errorf(
					"漫画图片文件超过20MB限制",
				)
		}
	}

	relativePath :=
		filepath.Join(
			coursewareID,
			"comics",
			projectID,
			subDirectory,
			storedName,
		)

	localURL :=
		CWAssetURLPrefix +
			filepath.ToSlash(
				relativePath,
			)

	metadata :=
		map[string]interface{}{
			"comic_role":        role,
			"comic_project_id":  projectID,
			"panel_number":      panelNumber,
			"generation_source": generationSource,
		}

	if panel != nil {
		metadata["comic_panel_id"] =
			panel.ID

		metadata["image_key"] =
			panel.ImageKey

		metadata["panel_version"] =
			panel.Version
	}

	if referenceAsset != nil {
		metadata["reference_asset_id"] =
			referenceAsset.ID
	}

	metadataJSON, _ :=
		json.Marshal(
			metadata,
		)

	asset :=
		&models.CoursewareAsset{
			CoursewareID:  coursewareID,
			PageID:        nil,
			PlaceholderID: placeholderID,
			AssetType:     models.CWAssetTypeImage,
			GenerationPrompt: strings.TrimSpace(
				prompt,
			),
			OssURL:   localURL,
			FileSize: written,
			MimeType: mimeType,
			Metadata: string(
				metadataJSON,
			),
			Status: models.CWAssetStatusUploaded,
		}

	if err :=
		repository.CreateCWAsset(
			ctx,
			asset,
		); err != nil {
		_ =
			os.Remove(
				fullPath,
			)

		return nil,
			fmt.Errorf(
				"记录漫画图片资产失败: %w",
				err,
			)
	}

	publicURL, uploadErr :=
		s.ossService.
			UploadAssetToOSS(
				localURL,
			)

	if uploadErr == nil &&
		strings.TrimSpace(
			publicURL,
		) != "" {
		updateErr :=
			repository.UpdateCWAssetPublicURL(
				ctx,
				asset.ID,
				publicURL,
			)

		if updateErr == nil {
			asset.PublicOSSURL =
				publicURL
		}
	}

	return asset, nil
}
