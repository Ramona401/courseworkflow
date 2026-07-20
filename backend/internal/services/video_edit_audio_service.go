package services

// video_edit_audio_service.go — 作者专属音轨提取与音频裁剪
//
// 两个入口均使用可信Actor、课件级并发保护、FFmpeg前后重新授权、
// 正式资产复合绑定、源文件版本复验、页面归属继承和失败输出清理。

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ExtractAudioRequest 音轨提取请求。
type ExtractAudioRequest struct {
	CoursewareID string
	AssetID      string
	Actor        *CoursewareActorContext
}

// ExtractAudioResponse 音轨提取响应。
type ExtractAudioResponse struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
	Format   string `json:"format"`
	FileSize int64  `json:"file_size"`
	Message  string `json:"message"`
}

// ExtractAudio 从视频中提取MP3音轨。
func (s *VideoEditService) ExtractAudio(
	ctx context.Context,
	req *ExtractAudioRequest,
) (
	*ExtractAudioResponse,
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
			"音轨提取",
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

	if !videoHasAudioStream(
		ctx,
		input.Path,
	) {
		return nil,
			fmt.Errorf(
				"%w: 该视频没有音频轨道",
				ErrVideoEditInputInvalid,
			)
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
		"audios",
	)
	outputPath,
		outputName,
		err :=
		createVideoEditOutputPath(
			outputDir,
			"extracted_*.mp3",
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

	if !videoHasAudioStream(
		ctx,
		input.Path,
	) {
		return nil,
			ErrVideoEditSourceChanged
	}

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", input.Path,
		"-vn",
		"-acodec", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)

	output, err :=
		command.CombinedOutput()
	if err != nil {
		videoEditLog.Error(
			"FFmpeg音轨提取失败",
			"error", err,
			"output", string(output),
		)

		return nil,
			fmt.Errorf(
				"音轨提取失败: %w",
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
		"audios",
		outputName,
	)

	newAsset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PageID:           pageID,
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeAudio,
		GenerationPrompt: "从视频提取音轨",
		OssURL:           localURL,
		FileSize:         info.Size(),
		MimeType:         "audio/mpeg",
		Status:           models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			newAsset,
		); err != nil {
		return nil,
			fmt.Errorf(
				"记录音频资产失败: %w",
				err,
			)
	}

	keepOutput = true

	return &ExtractAudioResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Format:   "mp3",
		FileSize: info.Size(),
		Message: fmt.Sprintf(
			"音轨提取完成，格式MP3，时长%s",
			duration,
		),
	}, nil
}

// TrimAudioRequest 音频裁剪请求。
type TrimAudioRequest struct {
	CoursewareID string
	AssetID      string
	StartSec     float64
	EndSec       float64
	Actor        *CoursewareActorContext
}

// TrimAudioResponse 音频裁剪响应。
type TrimAudioResponse struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
	Message  string `json:"message"`
}

// TrimAudio 裁剪正式课件音频资产。
func (s *VideoEditService) TrimAudio(
	ctx context.Context,
	req *TrimAudioRequest,
) (
	*TrimAudioResponse,
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
			"音频裁剪",
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
			models.CWAssetTypeAudio,
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
				"读取源音频时长失败: %w",
				err,
			)
	}

	if err :=
		validateVideoEditTimeRange(
			req.StartSec,
			req.EndSec,
			sourceDuration,
			0.5,
		); err != nil {
		return nil, err
	}

	extension,
		mimeType,
		err :=
		resolveVideoEditAudioFormat(
			input.Path,
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
		"audios",
	)
	outputPath,
		outputName,
		err :=
		createVideoEditOutputPath(
			outputDir,
			"trim_*"+extension,
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
			models.CWAssetTypeAudio,
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
			"FFmpeg音频裁剪失败",
			"error", err,
			"output", string(output),
		)

		return nil,
			fmt.Errorf(
				"音频裁剪失败: %w",
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
			models.CWAssetTypeAudio,
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
		"audios",
		outputName,
	)

	newAsset := &models.CoursewareAsset{
		CoursewareID:  req.CoursewareID,
		PageID:        pageID,
		PlaceholderID: "",
		AssetType:     models.CWAssetTypeAudio,
		GenerationPrompt: fmt.Sprintf(
			"裁剪 %.1fs-%.1fs",
			req.StartSec,
			req.EndSec,
		),
		OssURL:   localURL,
		FileSize: info.Size(),
		MimeType: mimeType,
		Status:   models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			newAsset,
		); err != nil {
		return nil,
			fmt.Errorf(
				"记录裁剪音频失败: %w",
				err,
			)
	}

	keepOutput = true

	return &TrimAudioResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: actualDuration,
		FileName: outputName,
		FileSize: info.Size(),
		MimeType: mimeType,
		Message: fmt.Sprintf(
			"裁剪完成，截取 %.1f-%.1f 秒，时长%s",
			req.StartSec,
			req.EndSec,
			actualDuration,
		),
	}, nil
}

// resolveVideoEditAudioFormat 校验并返回允许的音频扩展名和MIME。
func resolveVideoEditAudioFormat(
	sourcePath string,
) (
	string,
	string,
	error,
) {
	extension := strings.ToLower(
		filepath.Ext(sourcePath),
	)

	switch extension {
	case ".mp3":
		return extension, "audio/mpeg", nil
	case ".wav":
		return extension, "audio/wav", nil
	case ".ogg":
		return extension, "audio/ogg", nil
	case ".aac":
		return extension, "audio/aac", nil
	case ".flac":
		return extension, "audio/flac", nil
	case ".m4a":
		return extension, "audio/x-m4a", nil
	default:
		return "", "",
			fmt.Errorf(
				"%w: 不支持的音频格式",
				ErrVideoEditInputInvalid,
			)
	}
}
