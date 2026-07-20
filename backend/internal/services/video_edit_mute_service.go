package services

// video_edit_mute_service.go — 作者专属视频静音
//
// 使用静默AAC音轨替换原音轨，保留浏览器双轨MP4兼容性。
// FFmpeg前后均重新授权并复验正式源视频；写库失败会删除输出。

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// MuteVideoRequest 视频静音请求。
type MuteVideoRequest struct {
	CoursewareID string
	AssetID      string
	Actor        *CoursewareActorContext
}

// MuteVideoResponse 视频静音响应。
type MuteVideoResponse struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
	Message  string `json:"message"`
}

// MuteVideo 用静默AAC音轨替换原视频音轨。
func (s *VideoEditService) MuteVideo(
	ctx context.Context,
	req *MuteVideoRequest,
) (
	*MuteVideoResponse,
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
			"视频静音",
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
			"muted_*.mp4",
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

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", input.Path,
		"-f", "lavfi",
		"-i", "anullsrc=channel_layout=stereo:sample_rate=44100",
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "1k",
		"-shortest",
		"-avoid_negative_ts", "make_zero",
		"-movflags", "+faststart",
		outputPath,
	)

	output, err :=
		command.CombinedOutput()
	if err != nil {
		videoEditLog.Error(
			"FFmpeg静音失败",
			"error", err,
			"output", string(output),
		)

		return nil,
			fmt.Errorf(
				"视频静音失败: %w",
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

	newAsset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PageID:           pageID,
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: "静音处理（静默音轨替换）",
		OssURL:           localURL,
		FileSize:         info.Size(),
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			newAsset,
		); err != nil {
		return nil,
			fmt.Errorf(
				"记录静音视频失败: %w",
				err,
			)
	}

	keepOutput = true

	return &MuteVideoResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Message: fmt.Sprintf(
			"静音完成，已替换为静默音轨，时长%s",
			duration,
		),
	}, nil
}
