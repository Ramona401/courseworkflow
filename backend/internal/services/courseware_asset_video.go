package services

// courseware_asset_video.go — 课件视频生成服务（从courseware_asset_service.go拆分）
//
// v0.42.1 新增：AI视频生成（异步提交+状态查询+下载保存）
// 视频锚点轮新增：
//   - GenerateVideoServiceRequest 新增 SourceFrameAssetID（首帧图资产ID，两步流"先出首帧图再生视频"传入）
//   - GenerateVideo 创建视频资产后，若 SourceFrameAssetID 非空，写 metadata 溯源
//     {"source_frame_asset_id":"<首帧图asset_id>"}，建立"视频←首帧图"血缘（失败仅WARN不阻断主流程）
//
// 功能：
//   - GenerateVideo: 提交豆包Seedance视频生成任务（返回task_id）
//   - QueryVideoStatus: 查询任务状态（成功时下载保存到本地）
//   - downloadAndSaveVideo: 下载远程视频并保存到磁盘
//
// 与 courseware_asset_service.go 共享常量和CoursewareAssetService接收器

import (
	"context"
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

// ==================== v0.42.1 AI视频生成（异步模式） ====================

// GenerateVideoServiceRequest AI视频生成请求参数
type GenerateVideoServiceRequest struct {
	CoursewareID       string                  // 课件ID
	PageNumber         int                     // 页码
	Prompt             string                  // 视频描述提示词
	RefImageURL        string                  // 参考图URL（图生视频模式，可选，须公网可访问）
	Actor              *CoursewareActorContext // 可信作者Actor
	SourceFrameAssetID string                  // 首帧图资产ID（两步流可选，非空时写 metadata 溯源；空=直接文字生视频，无溯源）
}

// GenerateVideoServiceResponse AI视频生成任务提交响应
type GenerateVideoServiceResponse struct {
	AssetID   string `json:"asset_id"`   // 资产记录ID（status=generating）
	TaskID    string `json:"task_id"`    // 豆包视频任务ID（用于后续轮询）
	ModelUsed string `json:"model_used"` // 使用的模型
	Message   string `json:"message"`    // 提示信息
}

// validateVideoSourceFrameAsset 校验视频首帧资产的资源边界。
func validateVideoSourceFrameAsset(
	coursewareID string,
	asset *models.CoursewareAsset,
) error {
	if asset == nil {
		return fmt.Errorf("首帧资产不存在")
	}
	if asset.CoursewareID != coursewareID {
		return fmt.Errorf("首帧资产不属于此课件")
	}
	if asset.AssetType != models.CWAssetTypeImage {
		return fmt.Errorf("首帧资产必须是图片")
	}

	return nil
}

// resolveVideoReferenceURL 解析视频任务使用的可信参考图。
//
// 存在SourceFrameAssetID时，忽略客户端同时提交的RefImageURL，
// 只使用数据库内已经完成同课件图片校验的资产地址。
func (s *CoursewareAssetService) resolveVideoReferenceURL(
	ctx context.Context,
	req *GenerateVideoServiceRequest,
) (string, error) {
	sourceFrameAssetID := strings.TrimSpace(
		req.SourceFrameAssetID,
	)

	if sourceFrameAssetID != "" {
		sourceAsset, err := repository.GetCWAssetByID(
			ctx,
			sourceFrameAssetID,
		)
		if err != nil {
			return "", fmt.Errorf(
				"首帧资产不存在: %w",
				err,
			)
		}

		if err := validateVideoSourceFrameAsset(
			req.CoursewareID,
			sourceAsset,
		); err != nil {
			return "", err
		}

		publicURL := resolveAssetPublicURL(
			sourceAsset,
		)
		if publicURL == "" {
			return "", fmt.Errorf(
				"首帧图片没有可用的公网地址",
			)
		}

		return publicURL, nil
	}

	refImageURL := strings.TrimSpace(
		req.RefImageURL,
	)
	if refImageURL == "" {
		return "", nil
	}

	if strings.HasPrefix(
		refImageURL,
		"/uploads/",
	) {
		return cwAssetPublicHost + refImageURL, nil
	}

	return refImageURL, nil
}

// GenerateVideo 提交视频生成任务（异步，返回task_id供前端轮询）
func (s *CoursewareAssetService) GenerateVideo(
	ctx context.Context,
	req *GenerateVideoServiceRequest,
) (*GenerateVideoServiceResponse, error) {
	if req == nil {
		return nil, ErrCoursewareActorRequired
	}

	// 外部视频任务提交前重新加载正式课件并执行作者域二次授权。
	_, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				req.CoursewareID,
				req.Actor,
			)
	if err != nil {
		return nil, err
	}

	userID := scopedActor.UserID

	page, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			req.CoursewareID,
			req.PageNumber,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"页面不存在: 课件=%s 页码=%d",
			req.CoursewareID,
			req.PageNumber,
		)
	}

	// 首帧资产校验必须发生在外部任务提交之前。
	refURL, err := s.resolveVideoReferenceURL(
		ctx,
		req,
	)
	if err != nil {
		return nil, err
	}

	videoConfig, err :=
		ai.GetVideoConfig(s.cfg.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf(
			"视频生成API未配置: %w",
			err,
		)
	}

	traceContext := &ai.TraceContext{
		SceneCode: "courseware_video_gen",
		UserID:    &userID,
	}

	result, err := ai.SubmitVideoTask(
		ctx,
		videoConfig,
		req.Prompt,
		refURL,
		traceContext,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"视频任务提交失败: %w",
			err,
		)
	}

	asset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PageID:           &page.ID,
		PlaceholderID:    result.TaskID,
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: req.Prompt,
		OssURL:           "",
		FileSize:         0,
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusGenerating,
	}

	if err := repository.CreateCWAsset(
		ctx,
		asset,
	); err != nil {
		return nil, fmt.Errorf(
			"记录视频资产失败: %w",
			err,
		)
	}

	sourceFrameAssetID := strings.TrimSpace(
		req.SourceFrameAssetID,
	)
	if sourceFrameAssetID != "" {
		metadataBytes, marshalErr :=
			json.Marshal(
				map[string]string{
					"source_frame_asset_id": sourceFrameAssetID,
				},
			)

		if marshalErr != nil {
			cwAssetLog.Warn(
				"视频首帧溯源JSON序列化失败(跳过溯源,不影响视频生成)",
				"asset_id",
				asset.ID,
				"source_frame_asset_id",
				sourceFrameAssetID,
				"error",
				marshalErr,
			)
		} else if updateErr :=
			repository.UpdateCWAssetMetadata(
				ctx,
				asset.ID,
				string(metadataBytes),
			); updateErr != nil {
			cwAssetLog.Warn(
				"写入视频首帧溯源metadata失败(不影响视频生成)",
				"asset_id",
				asset.ID,
				"source_frame_asset_id",
				sourceFrameAssetID,
				"error",
				updateErr,
			)
		} else {
			cwAssetLog.Info(
				"视频已记录首帧溯源",
				"asset_id",
				asset.ID,
				"source_frame_asset_id",
				sourceFrameAssetID,
			)
		}
	}

	cwAssetLog.Info(
		"视频生成任务已提交",
		"courseware_id",
		req.CoursewareID,
		"page_number",
		req.PageNumber,
		"asset_id",
		asset.ID,
		"task_id",
		result.TaskID,
		"model",
		result.ModelUsed,
		"prompt_len",
		len(req.Prompt),
		"has_ref_image",
		refURL != "",
		"has_source_frame",
		sourceFrameAssetID != "",
	)

	return &GenerateVideoServiceResponse{
		AssetID:   asset.ID,
		TaskID:    result.TaskID,
		ModelUsed: result.ModelUsed,
		Message:   "视频生成任务已提交，通常需要30-120秒完成",
	}, nil
}

// ==================== v0.42.1 视频任务状态查询 ====================

// QueryVideoStatusResponse 视频任务状态查询响应
type QueryVideoStatusResponse struct {
	AssetID    string `json:"asset_id"`   // 资产记录ID
	TaskID     string `json:"task_id"`    // 豆包任务ID
	Status     string `json:"status"`     // 状态：generating/uploaded/failed
	VideoURL   string `json:"video_url"`  // 视频本地URL（成功时有值）
	Duration   int    `json:"duration"`   // 视频时长（秒）
	Resolution string `json:"resolution"` // 分辨率
	Ratio      string `json:"ratio"`      // 画面比例
	ErrorMsg   string `json:"error_msg"`  // 错误信息（失败时有值）
	Message    string `json:"message"`    // 提示信息
}

// QueryVideoStatus 查询视频生成任务状态
// 如果任务已完成，自动下载视频保存到本地并更新数据库
func (s *CoursewareAssetService) QueryVideoStatus(
	ctx context.Context,
	coursewareID string,
	assetID string,
	actor *CoursewareActorContext,
) (*QueryVideoStatusResponse, error) {
	// 先授权路径中的课件，避免通过随机asset_id探测其它课件资产。
	if _, _, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			); err != nil {
		return nil, err
	}

	asset, err := repository.GetCWAssetByID(
		ctx,
		assetID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"视频资产不存在: %w",
			err,
		)
	}

	if asset.CoursewareID != coursewareID {
		return nil, fmt.Errorf(
			"视频资产不属于路径指定课件",
		)
	}
	if asset.AssetType != models.CWAssetTypeVideo {
		return nil, fmt.Errorf(
			"资产不是视频资产",
		)
	}

	if asset.Status == models.CWAssetStatusUploaded ||
		asset.Status == models.CWAssetStatusConfirmed {
		return &QueryVideoStatusResponse{
			AssetID:  asset.ID,
			TaskID:   asset.PlaceholderID,
			Status:   "uploaded",
			VideoURL: asset.OssURL,
			Message:  "视频已生成完成",
		}, nil
	}

	if asset.Status != models.CWAssetStatusGenerating {
		return &QueryVideoStatusResponse{
			AssetID: asset.ID,
			TaskID:  asset.PlaceholderID,
			Status:  "failed",
			ErrorMsg: "视频资产状态异常: " +
				asset.Status,
			Message: "视频生成出现问题",
		}, nil
	}

	taskID := strings.TrimSpace(
		asset.PlaceholderID,
	)
	if taskID == "" {
		return nil, fmt.Errorf(
			"视频任务ID为空",
		)
	}

	videoConfig, err :=
		ai.GetVideoConfig(s.cfg.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf(
			"视频生成API配置加载失败: %w",
			err,
		)
	}

	queryResult, err := ai.QueryVideoTask(
		ctx,
		videoConfig,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询视频任务状态失败: %w",
			err,
		)
	}

	switch queryResult.Status {
	case "running":
		return &QueryVideoStatusResponse{
			AssetID: asset.ID,
			TaskID:  taskID,
			Status:  "generating",
			Message: "视频正在生成中，请稍后重试查询",
		}, nil

	case "succeeded":
		if strings.TrimSpace(
			queryResult.VideoURL,
		) == "" {
			return nil, fmt.Errorf(
				"视频生成成功但未返回视频URL",
			)
		}

		// 外部任务完成后、下载写盘前再次校验课件作者与教育域。
		if _, _, authErr :=
			(&CoursewareService{}).
				LoadCoursewareForOwnerRuntime(
					ctx,
					coursewareID,
					actor,
				); authErr != nil {
			return nil, authErr
		}

		localURL, fileSize, downloadErr :=
			s.downloadAndSaveVideo(
				ctx,
				coursewareID,
				taskID,
				queryResult.VideoURL,
			)
		if downloadErr != nil {
			cwAssetLog.Error(
				"下载视频失败",
				"asset_id",
				asset.ID,
				"task_id",
				taskID,
				"error",
				downloadErr,
			)

			return nil, fmt.Errorf(
				"下载视频文件失败: %w",
				downloadErr,
			)
		}

		if updateErr :=
			repository.UpdateCWAssetOSSURL(
				ctx,
				asset.ID,
				localURL,
				fileSize,
				"video/mp4",
			); updateErr != nil {
			cwAssetLog.Warn(
				"更新视频URL失败",
				"asset_id",
				asset.ID,
				"error",
				updateErr,
			)
		}

		cwAssetLog.Info(
			"视频生成完成并保存到本地",
			"asset_id",
			asset.ID,
			"task_id",
			taskID,
			"duration",
			queryResult.Duration,
			"resolution",
			queryResult.Resolution,
			"file_size",
			fileSize,
			"local_url",
			localURL,
		)

		return &QueryVideoStatusResponse{
			AssetID:    asset.ID,
			TaskID:     taskID,
			Status:     "uploaded",
			VideoURL:   localURL,
			Duration:   queryResult.Duration,
			Resolution: queryResult.Resolution,
			Ratio:      queryResult.Ratio,
			Message: fmt.Sprintf(
				"视频生成完成！时长%d秒，分辨率%s",
				queryResult.Duration,
				queryResult.Resolution,
			),
		}, nil

	case "failed":
		_ = repository.UpdateCWAssetStatus(
			ctx,
			asset.ID,
			models.CWAssetStatusPending,
		)

		cwAssetLog.Warn(
			"视频生成失败",
			"asset_id",
			asset.ID,
			"task_id",
			taskID,
			"error",
			queryResult.ErrorMsg,
		)

		return &QueryVideoStatusResponse{
			AssetID:  asset.ID,
			TaskID:   taskID,
			Status:   "failed",
			ErrorMsg: queryResult.ErrorMsg,
			Message: "视频生成失败: " +
				queryResult.ErrorMsg,
		}, nil

	default:
		return &QueryVideoStatusResponse{
			AssetID: asset.ID,
			TaskID:  taskID,
			Status:  "generating",
			Message: "视频状态: " +
				queryResult.Status,
		}, nil
	}
}

// downloadAndSaveVideo 下载远程视频并保存到本地磁盘
// 返回 (本地URL, 文件大小bytes, error)
func (s *CoursewareAssetService) downloadAndSaveVideo(ctx context.Context, coursewareID string, taskID string, videoURL string) (string, int64, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(videoURL)
	if err != nil {
		return "", 0, fmt.Errorf("下载视频失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("下载视频HTTP错误: %d", resp.StatusCode)
	}

	storedName := fmt.Sprintf("%d_video_%s.mp4", time.Now().UnixMilli(), taskID)

	assetDir := filepath.Join(CWAssetUploadDir, coursewareID, "videos")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return "", 0, fmt.Errorf("创建视频目录失败: %w", err)
	}

	fullPath := filepath.Join(assetDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", 0, fmt.Errorf("创建视频文件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, resp.Body)
	if err != nil {
		_ = os.Remove(fullPath)
		return "", 0, fmt.Errorf("写入视频文件失败: %w", err)
	}

	relativePath := filepath.Join(coursewareID, "videos", storedName)
	localURL := CWAssetURLPrefix + relativePath

	return localURL, written, nil
}
