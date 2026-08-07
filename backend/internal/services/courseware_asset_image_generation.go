package services

// courseware_asset_image_generation.go — 普通课件页面图片生成与积分计费
//
// 本模块负责普通页面图片、自动装配图片和视频首帧图片的统一生成入口。
// 计费边界：
//   - 调用图片供应商前预留积分；
//   - 供应商明确失败时释放预留；
//   - 供应商成功后即确认真实成本；
//   - 下载或资产落库失败仍结算一次图片积分；
//   - 同一operation_id重放不得再次调用供应商；
//   - 真实模型、供应商原始URL不向普通用户JSON接口暴露。

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// GenerateImageServiceRequest AI图片生成请求。
type GenerateImageServiceRequest struct {
	CoursewareID  string
	PageNumber    int
	PlaceholderID string
	Prompt        string
	Size          string
	RefImageURL   string

	// OperationID由前端一次点击生成一个UUID。
	// 同一次HTTP网络重放必须沿用同一个值。
	// 旧内部调用缺省时生成新UUID以保持兼容，
	// 正式HTTP链路将在下一批补齐稳定值。
	OperationID string

	Actor *CoursewareActorContext

	// 以下字段只允许可信内部编排设置。
	// HTTP处理器不得接收客户端自定义的节点或幂等键。
	BillingNodeCode       string
	BillingIdempotencyKey string
}

// GenerateImageServiceResponse AI图片生成业务响应。
//
// OriginalURLs和ModelUsed仅供后端内部编排及日志使用，
// JSON序列化时明确隐藏，普通用户不会看到真实模型或供应商URL。
type GenerateImageServiceResponse struct {
	AssetID      string   `json:"asset_id"`
	URL          string   `json:"url"`
	OriginalURLs []string `json:"-"`
	ModelUsed    string   `json:"-"`

	RevisedPrompt string `json:"revised_prompt"`
}

// GenerateImage 生成普通课件页面图片并通过统一媒体计费状态机扣积分。
func (s *CoursewareAssetService) GenerateImage(
	ctx context.Context,
	request *GenerateImageServiceRequest,
) (*GenerateImageServiceResponse, error) {
	if request == nil {
		return nil, ErrCoursewareActorRequired
	}

	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("图片生成提示词不能为空")
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

	imageConfig, err := ai.GetImageConfig(s.cfg.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf("图片生成API未配置: %w", err)
	}

	userID := scopedActor.UserID
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)

	traceContext := &ai.TraceContext{
		SceneCode: "courseware_image_gen",
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	imageSize := strings.TrimSpace(request.Size)
	if imageSize == "" {
		imageSize = "1920x1920"
	}

	effectivePrompt, referenceURL := s.resolveCoursewareGenerateImageStyle(
		ctx,
		courseware,
		request,
		prompt,
	)

	billingNodeCode, idempotencyKey, operationID, err :=
		resolveCoursewareGenerateImageBillingIdentity(
			request,
			page.ID,
		)
	if err != nil {
		return nil, err
	}

	result, asset, err := executeBilledCoursewareImage(
		ctx,
		&coursewareImageBillingInput{
			UserID:          userID,
			SchoolID:        schoolIDPtr(schoolID),
			BillingNodeCode: billingNodeCode,
			CoursewareID:    request.CoursewareID,
			PageID:          &page.ID,
			ModelName:       imageConfig.Model,
			IdempotencyKey:  idempotencyKey,
			Metadata: map[string]interface{}{
				"page_number":          request.PageNumber,
				"page_id":              page.ID,
				"placeholder_id":       strings.TrimSpace(request.PlaceholderID),
				"operation_id":         operationID,
				"requested_image_size": imageSize,
				"has_reference_image":  referenceURL != "",
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
		func(generated *ai.ImageGenerateResult) (*models.CoursewareAsset, error) {
			saved, saveErr := s.downloadAndSaveImageWithMetadata(
				ctx,
				request.CoursewareID,
				request.PageNumber,
				generated.URLs[0],
				prompt,
			)
			if saveErr != nil {
				return nil, fmt.Errorf("下载生成图片失败: %w", saveErr)
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
				CoursewareID:     request.CoursewareID,
				PageID:           &page.ID,
				PlaceholderID:    strings.TrimSpace(request.PlaceholderID),
				AssetType:        models.CWAssetTypeImage,
				GenerationPrompt: prompt,
				OssURL:           saved.URL,
				FileSize:         saved.FileSize,
				MimeType:         saved.MimeType,
				Status:           models.CWAssetStatusUploaded,
			}
			if createErr := repository.CreateCWAsset(
				ctx,
				persisted,
			); createErr != nil {
				return nil, fmt.Errorf("记录图片资产失败: %w", createErr)
			}

			return persisted, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("图片生成或资产保存失败: %w", err)
	}

	cwAssetLog.Info(
		"AI图片生成并保存成功",
		"courseware_id", request.CoursewareID,
		"page_number", request.PageNumber,
		"asset_id", asset.ID,
		"billing_node_code", billingNodeCode,
		"model", result.ModelUsed,
		"prompt_len", len(prompt),
		"file_size", asset.FileSize,
		"mime_type", asset.MimeType,
	)

	return &GenerateImageServiceResponse{
		AssetID:       asset.ID,
		URL:           asset.OssURL,
		OriginalURLs:  result.URLs,
		ModelUsed:     result.ModelUsed,
		RevisedPrompt: result.RevisedPrompt,
	}, nil
}

// resolveCoursewareGenerateImageStyle 解析普通生图所需的提示词和参考图。
func (s *CoursewareAssetService) resolveCoursewareGenerateImageStyle(
	ctx context.Context,
	courseware *models.Courseware,
	request *GenerateImageServiceRequest,
	prompt string,
) (string, string) {
	effectivePrompt := prompt
	referenceURL := ""

	requestReferenceURL := strings.TrimSpace(request.RefImageURL)
	if requestReferenceURL != "" {
		if strings.HasPrefix(requestReferenceURL, "/uploads/") {
			return effectivePrompt, cwAssetPublicHost + requestReferenceURL
		}
		return effectivePrompt, requestReferenceURL
	}

	if courseware == nil ||
		courseware.StyleAnchorAssetID == nil ||
		strings.TrimSpace(*courseware.StyleAnchorAssetID) == "" {
		return effectivePrompt, referenceURL
	}

	anchorURL, vaoci := s.resolveStyleAnchorForGen(
		ctx,
		courseware,
		request.PlaceholderID,
	)
	if anchorURL == "" {
		return effectivePrompt, referenceURL
	}

	referenceURL = anchorURL
	if vaoci != "" {
		effectivePrompt = prompt +
			"\n\n【风格与人物一致性约束】" +
			"请严格保持与参考图一致的视觉风格，" +
			"并保持画面中人物或主体角色的发型、脸型、" +
			"服装和配色与参考图一致：" + vaoci
	}

	cwAssetLog.Info(
		"生成图片自动套用风格锚点",
		"courseware_id", request.CoursewareID,
		"page_number", request.PageNumber,
		"anchor_asset_id", *courseware.StyleAnchorAssetID,
		"has_vaoci", vaoci != "",
	)

	return effectivePrompt, referenceURL
}

// resolveCoursewareGenerateImageBillingIdentity 解析可信计费节点和幂等键。
func resolveCoursewareGenerateImageBillingIdentity(
	request *GenerateImageServiceRequest,
	pageID string,
) (string, string, string, error) {
	if request == nil {
		return "", "", "", ErrMediaBillingInvalidRequest
	}

	nodeCode := strings.TrimSpace(request.BillingNodeCode)
	if nodeCode == "" {
		nodeCode = "courseware_page_image"
	}

	switch nodeCode {
	case "courseware_page_image",
		"courseware_auto_assembly_image",
		"video_first_frame":
	default:
		return "", "", "", fmt.Errorf(
			"%w: 图片业务节点%s不允许",
			ErrMediaBillingInvalidRequest,
			nodeCode,
		)
	}

	internalKey := strings.TrimSpace(request.BillingIdempotencyKey)
	if internalKey != "" {
		return nodeCode, internalKey, "", nil
	}

	operationID := strings.TrimSpace(request.OperationID)
	if operationID == "" {
		operationID = uuid.NewString()
	}

	parsedOperationID, err := uuid.Parse(operationID)
	if err != nil {
		return "", "", "", fmt.Errorf("图片生成operation_id不合法")
	}
	operationID = parsedOperationID.String()

	return nodeCode,
		fmt.Sprintf(
			"courseware-image:%s:%s:%s",
			nodeCode,
			strings.TrimSpace(pageID),
			operationID,
		),
		operationID,
		nil
}

// resolveStyleAnchorForGen 获取风格锚点图片公网URL和VAOCI。
//
// 当前目标占位符与锚点资产占位符相同时，不允许图片自我引用。
func (s *CoursewareAssetService) resolveStyleAnchorForGen(
	ctx context.Context,
	courseware *models.Courseware,
	currentPlaceholderID string,
) (string, string) {
	if courseware == nil ||
		courseware.StyleAnchorAssetID == nil ||
		strings.TrimSpace(*courseware.StyleAnchorAssetID) == "" {
		return "", ""
	}

	anchorAsset, err := repository.GetCWAssetByID(
		ctx,
		*courseware.StyleAnchorAssetID,
	)
	if err != nil {
		cwAssetLog.Warn(
			"风格锚点资产查询失败，本次生成跳过套用",
			"courseware_id", courseware.ID,
			"anchor_asset_id", *courseware.StyleAnchorAssetID,
			"error", err,
		)
		return "", ""
	}

	if strings.TrimSpace(currentPlaceholderID) != "" &&
		anchorAsset.PlaceholderID == strings.TrimSpace(currentPlaceholderID) {
		return "", ""
	}

	anchorURL := resolveAssetPublicURL(anchorAsset)
	if anchorURL == "" {
		return "", ""
	}

	return anchorURL, strings.TrimSpace(courseware.StyleAnchorVAOCI)
}

// downloadAndSaveImage 保留旧字符串返回协议。
//
// 旧内部调用方继续只接收本地URL；实际下载、文件签名校验和
// 文件元数据计算统一委托给downloadAndSaveImageWithMetadata。
func (s *CoursewareAssetService) downloadAndSaveImage(
	ctx context.Context,
	coursewareID string,
	pageNumber int,
	imageURL string,
	prompt string,
) (string, error) {
	saved, err :=
		s.downloadAndSaveImageWithMetadata(
			ctx,
			coursewareID,
			pageNumber,
			imageURL,
			prompt,
		)
	if err != nil {
		return "", err
	}

	if err :=
		validateCoursewareGeneratedImageFile(
			saved,
		); err != nil {
		return "", err
	}

	return saved.URL, nil
}

// downloadAndSaveImageWithMetadata 下载图片并返回完整本地文件元数据。
//
// 文件先写入同目录临时文件，关闭成功后按实际文件签名识别MIME，
// 再使用匹配扩展名原子改名。最终file_size以os.Stat结果为准。
func (s *CoursewareAssetService) downloadAndSaveImageWithMetadata(
	ctx context.Context,
	coursewareID string,
	pageNumber int,
	imageURL string,
	prompt string,
) (*coursewareGeneratedImageFile, error) {
	request, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			imageURL,
			nil,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"构造图片下载请求失败: %w",
				err,
			)
	}

	client :=
		&http.Client{
			Timeout: 30 * time.Second,
		}

	response, err :=
		client.Do(
			request,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"下载图片失败: %w",
				err,
			)
	}
	defer response.Body.Close()

	if response.StatusCode !=
		http.StatusOK {
		return nil,
			fmt.Errorf(
				"下载图片HTTP错误: %d",
				response.StatusCode,
			)
	}

	assetDirectory :=
		filepath.Join(
			CWAssetUploadDir,
			coursewareID,
			fmt.Sprintf(
				"p%d",
				pageNumber,
			),
		)

	if err :=
		os.MkdirAll(
			assetDirectory,
			0755,
		); err != nil {
		return nil,
			fmt.Errorf(
				"创建图片目录失败: %w",
				err,
			)
	}

	temporaryFile, err :=
		os.CreateTemp(
			assetDirectory,
			".generated-image-*.part",
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"创建图片临时文件失败: %w",
				err,
			)
	}

	temporaryPath :=
		temporaryFile.Name()

	removeTemporary :=
		true

	defer func() {
		if removeTemporary {
			_ =
				os.Remove(
					temporaryPath,
				)
		}
	}()

	written, copyErr :=
		io.Copy(
			temporaryFile,
			response.Body,
		)

	closeErr :=
		temporaryFile.Close()

	if copyErr != nil {
		return nil,
			fmt.Errorf(
				"写入图片文件失败: %w",
				copyErr,
			)
	}

	if closeErr != nil {
		return nil,
			fmt.Errorf(
				"关闭图片文件失败: %w",
				closeErr,
			)
	}

	if written <= 0 {
		return nil,
			fmt.Errorf(
				"图片供应商返回空文件",
			)
	}

	mimeType, err :=
		detectGeneratedImageMIMEType(
			temporaryPath,
			response.Header.Get(
				"Content-Type",
			),
		)
	if err != nil {
		return nil, err
	}

	extension, err :=
		generatedImageExtension(
			mimeType,
		)
	if err != nil {
		return nil, err
	}

	storedName :=
		buildGeneratedImageName(
			prompt,
			extension,
		)

	fullPath :=
		filepath.Join(
			assetDirectory,
			storedName,
		)

	if err :=
		os.Rename(
			temporaryPath,
			fullPath,
		); err != nil {
		return nil,
			fmt.Errorf(
				"确认图片文件失败: %w",
				err,
			)
	}

	removeTemporary =
		false

	fileInfo, err :=
		os.Stat(
			fullPath,
		)
	if err != nil {
		_ =
			os.Remove(
				fullPath,
			)

		return nil,
			fmt.Errorf(
				"读取图片文件信息失败: %w",
				err,
			)
	}

	if !fileInfo.Mode().
		IsRegular() ||
		fileInfo.Size() <= 0 {
		_ =
			os.Remove(
				fullPath,
			)

		return nil,
			fmt.Errorf(
				"生成图片文件状态异常",
			)
	}

	relativePath :=
		filepath.Join(
			coursewareID,
			fmt.Sprintf(
				"p%d",
				pageNumber,
			),
			storedName,
		)

	return &coursewareGeneratedImageFile{
		URL: CWAssetURLPrefix +
			relativePath,
		FileSize: fileInfo.Size(),
		MimeType: mimeType,
	}, nil
}

// buildGeneratedImageName 构造AI图片的安全本地文件名。
func buildGeneratedImageName(prompt string, extension string) string {
	nameHint := cwAssetSafeNameRe.ReplaceAllString(prompt, "_")
	nameRunes := []rune(nameHint)
	if len(nameRunes) > 20 {
		nameHint = string(nameRunes[:20])
	}
	for strings.Contains(nameHint, "__") {
		nameHint = strings.ReplaceAll(nameHint, "__", "_")
	}
	nameHint = strings.Trim(nameHint, "_")
	if nameHint == "" {
		nameHint = "ai_gen"
	}

	return fmt.Sprintf(
		"%d_ai_%s%s",
		time.Now().UnixMilli(),
		nameHint,
		extension,
	)
}
