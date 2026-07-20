package services

// video_edit_trim_service.go — 作者专属视频裁剪
//
// 裁剪会在FFmpeg前后重新授权、重新绑定源资产并比较本地文件版本；
// 新资产继承源视频页面。失败或写库冲突时删除本轮输出。

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// TrimVideoRequest 视频裁剪请求。
type TrimVideoRequest struct {
	CoursewareID string
	AssetID      string
	StartSec     float64
	EndSec       float64
	Actor        *CoursewareActorContext
}

// TrimVideoResponse 视频裁剪响应。
type TrimVideoResponse struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
	Message  string `json:"message"`
}

// TrimVideo 截取指定视频时间段。
func (s *VideoEditService) TrimVideo(
	ctx context.Context,
	req *TrimVideoRequest,
) (
	*TrimVideoResponse,
	error,
) {
	if req == nil {
		return nil, ErrVideoEditInputInvalid
	}

	_,
		scopedActor,
		release,
		err :=
		s.beginVideoEditOperation(
			ctx,
			req.CoursewareID,
			req.Actor,
			"视频裁剪",
		)
	if err != nil {
		return nil, err
	}
	defer release()

	input, err :=
		loadVideoEditInput(
			ctx,
			req.CoursewareID,
			req.AssetID,
			models.CWAssetTypeVideo,
		)
	if err != nil {
		return nil, err
	}

	sourceDuration, err :=
		probeMediaDurationSeconds(
			ctx,
			input.Path,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"读取源视频时长失败: %w",
				err,
			)
	}

	if err :=
		validateVideoEditTimeRange(
			req.StartSec,
			req.EndSec,
			sourceDuration,
			1.0,
		); err != nil {
		return nil, err
	}

	pageID, err :=
		validateVideoEditInheritedPage(
			ctx,
			req.CoursewareID,
			input.Asset.PageID,
		)
	if err != nil {
		return nil, err
	}

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
			"trim_*.mp4",
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
		reloadVideoEditInputUnchanged(
			ctx,
			req.CoursewareID,
			input,
			models.CWAssetTypeVideo,
		); err != nil {
		return nil, err
	}

	duration := req.EndSec - req.StartSec

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-ss", fmt.Sprintf(
			"%.3f",
			req.StartSec,
		),
		"-i", input.Path,
		"-t", fmt.Sprintf(
			"%.3f",
			duration,
		),
		"-c", "copy",
		outputPath,
	)

	output, err :=
		command.CombinedOutput()
	if err != nil {
		videoEditLog.Error(
			"FFmpeg视频裁剪失败",
			"error", err,
			"output", string(output),
		)

		return nil,
			fmt.Errorf(
				"视频裁剪失败: %w",
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
		reloadVideoEditInputUnchanged(
			ctx,
			req.CoursewareID,
			input,
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

	actualDuration :=
		getMediaDurationLabel(
			ctx,
			outputPath,
		)
	localURL := videoEditLocalURL(
		req.CoursewareID,
		"videos",
		outputName,
	)

	newAsset := &models.CoursewareAsset{
		CoursewareID:  req.CoursewareID,
		PageID:        pageID,
		PlaceholderID: "",
		AssetType:     models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf(
			"裁剪 %.1fs-%.1fs",
			req.StartSec,
			req.EndSec,
		),
		OssURL:   localURL,
		FileSize: info.Size(),
		MimeType: "video/mp4",
		Status:   models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			newAsset,
		); err != nil {
		return nil,
			fmt.Errorf(
				"记录裁剪视频失败: %w",
				err,
			)
	}

	keepOutput = true

	return &TrimVideoResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: actualDuration,
		Message: fmt.Sprintf(
			"裁剪完成，截取 %.1f-%.1f 秒，时长%s",
			req.StartSec,
			req.EndSec,
			actualDuration,
		),
	}, nil
}
