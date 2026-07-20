package services

// video_edit_service.go — 作者专属视频顺序拼接
//
// 安全顺序：
//   初始Service授权与课件级任务登记
//   → 复合加载正式视频资产
//   → FFmpeg前重新授权和输入复验
//   → 执行FFmpeg
//   → FFmpeg后、写库前重新授权和输入复验
//   → 复验继承页面并创建新资产
//
// 任一失败、失权、冲突、请求取消或写库失败都会删除本轮输出文件。

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var videoEditLog = logger.WithModule(
	"video_edit",
)

// VideoEditService 视频编辑服务。
type VideoEditService struct {
	cfg *config.Config
}

// NewVideoEditService 创建视频编辑服务。
func NewVideoEditService(
	cfg *config.Config,
) *VideoEditService {
	return &VideoEditService{
		cfg: cfg,
	}
}

// ConcatVideosRequest 视频顺序拼接请求。
type ConcatVideosRequest struct {
	CoursewareID string
	AssetIDs     []string
	Actor        *CoursewareActorContext
}

// ConcatVideosResponse 视频拼接响应。
type ConcatVideosResponse struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
	Message  string `json:"message"`
}

// ConcatVideos 多视频顺序拼接。
func (s *VideoEditService) ConcatVideos(
	ctx context.Context,
	req *ConcatVideosRequest,
) (
	*ConcatVideosResponse,
	error,
) {
	if req == nil ||
		len(req.AssetIDs) < 2 ||
		len(req.AssetIDs) > 10 {
		return nil,
			fmt.Errorf(
				"%w: 单次拼接需要2到10个视频",
				ErrVideoEditInputInvalid,
			)
	}

	_,
		scopedActor,
		release,
		err :=
		s.beginVideoEditOperation(
			ctx,
			req.CoursewareID,
			req.Actor,
			"视频拼接",
		)
	if err != nil {
		return nil, err
	}
	defer release()

	inputs := make(
		[]*videoEditInputSnapshot,
		0,
		len(req.AssetIDs),
	)
	filePaths := make(
		[]string,
		0,
		len(req.AssetIDs),
	)

	for _, assetID := range req.AssetIDs {
		input, err :=
			loadVideoEditInput(
				ctx,
				req.CoursewareID,
				assetID,
				models.CWAssetTypeVideo,
			)
		if err != nil {
			return nil, err
		}

		inputs = append(
			inputs,
			input,
		)
		filePaths = append(
			filePaths,
			input.Path,
		)
	}

	pageID, err :=
		validateVideoEditInheritedPage(
			ctx,
			req.CoursewareID,
			inputs[0].Asset.PageID,
		)
	if err != nil {
		return nil, err
	}

	listFile, err :=
		writeVideoEditConcatList(
			filePaths,
		)
	if err != nil {
		return nil, err
	}
	defer cleanupVideoEditFile(listFile)

	outputDir := filepath.Join(
		CWAssetUploadDir,
		req.CoursewareID,
		"videos",
	)
	outputPath,
		outputName,
		err :=
		createVideoEditOutputPath(
			outputDir,
			"concat_*.mp4",
		)
	if err != nil {
		return nil, err
	}

	keepOutput := false

	defer func() {
		if !keepOutput {
			cleanupVideoEditFile(
				outputPath,
			)
		}
	}()

	_,
		scopedActor,
		err =
		reauthorizeVideoEditOwner(
			ctx,
			req.CoursewareID,
			scopedActor,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		reloadVideoEditInputsUnchanged(
			ctx,
			req.CoursewareID,
			inputs,
			models.CWAssetTypeVideo,
		); err != nil {
		return nil, err
	}

	videoEditLog.Info(
		"开始视频拼接",
		"courseware_id", req.CoursewareID,
		"video_count", len(inputs),
		"output", outputPath,
	)

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		outputPath,
	)

	output, err :=
		command.CombinedOutput()
	if err != nil {
		videoEditLog.Error(
			"FFmpeg拼接失败",
			"error", err,
			"output", string(output),
		)

		return nil,
			fmt.Errorf(
				"视频拼接失败: %w",
				err,
			)
	}

	info, err :=
		validateVideoEditOutput(
			outputPath,
		)
	if err != nil {
		return nil, err
	}

	_,
		scopedActor,
		err =
		reauthorizeVideoEditOwner(
			ctx,
			req.CoursewareID,
			scopedActor,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		reloadVideoEditInputsUnchanged(
			ctx,
			req.CoursewareID,
			inputs,
			models.CWAssetTypeVideo,
		); err != nil {
		return nil, err
	}

	pageID, err =
		validateVideoEditInheritedPage(
			ctx,
			req.CoursewareID,
			pageID,
		)
	if err != nil {
		return nil, err
	}

	duration :=
		getMediaDurationLabel(
			ctx,
			outputPath,
		)
	localURL := videoEditLocalURL(
		req.CoursewareID,
		"videos",
		outputName,
	)

	asset := &models.CoursewareAsset{
		CoursewareID:  req.CoursewareID,
		PageID:        pageID,
		PlaceholderID: "",
		AssetType:     models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf(
			"拼接%d个视频片段",
			len(inputs),
		),
		OssURL:   localURL,
		FileSize: info.Size(),
		MimeType: "video/mp4",
		Status:   models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			asset,
		); err != nil {
		return nil,
			fmt.Errorf(
				"记录拼接视频失败: %w",
				err,
			)
	}

	keepOutput = true

	return &ConcatVideosResponse{
		AssetID:  asset.ID,
		URL:      localURL,
		Duration: duration,
		Message: fmt.Sprintf(
			"成功拼接%d个视频，总时长%s",
			len(inputs),
			duration,
		),
	}, nil
}

// videoEditLocalURL 构造统一斜杠的本地资源URL。
func videoEditLocalURL(
	coursewareID string,
	subdir string,
	fileName string,
) string {
	return CWAssetURLPrefix +
		filepath.ToSlash(
			filepath.Join(
				coursewareID,
				subdir,
				fileName,
			),
		)
}
