package services

// video_edit_narration.go — 作者专属TTS旁白混入成片
//
// 安全顺序：
//   作者控制授权与课件级任务登记
//   → 视频资产和字幕复合读取
//   → editor_draft字幕个人归属校验
//   → TTS文件限制在当前课件tts目录
//   → FFmpeg前重新授权、字幕版本和全部文件复验
//   → 执行FFmpeg
//   → FFmpeg后、资产写库前再次执行相同复验
//   → 创建继承源视频页面的新资产
//
// 失权、字幕并发修改、源视频变化、TTS文件变化或写库失败时，
// 本轮输出视频都会被删除。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// MixNarrationRequest 配音混音请求。
type MixNarrationRequest struct {
	CoursewareID string
	AssetID      string
	SubtitleID   string
	Gain         float64
	Actor        *CoursewareActorContext
}

// MixNarrationResponse 配音混音响应。
type MixNarrationResponse struct {
	AssetID        string `json:"asset_id"`
	URL            string `json:"url"`
	Duration       string `json:"duration"`
	NarrationCount int    `json:"narration_count"`
	SkippedCount   int    `json:"skipped_count"`
	Message        string `json:"message"`
}

// narrationInput 单条旁白文件及时间轴位置。
type narrationInput struct {
	Path     string
	StartSec float64
	Size     int64
	ModTime  time.Time
	Mode     os.FileMode
}

// MixNarration 把字幕TTS旁白按时间轴混入视频。
func (s *VideoEditService) MixNarration(
	ctx context.Context,
	req *MixNarrationRequest,
) (
	*MixNarrationResponse,
	error,
) {
	if req == nil {
		return nil, ErrVideoEditInputInvalid
	}

	assetID := strings.TrimSpace(
		req.AssetID,
	)
	subtitleID := strings.TrimSpace(
		req.SubtitleID,
	)
	if assetID == "" ||
		subtitleID == "" {
		return nil,
			fmt.Errorf(
				"%w: asset_id和subtitle_id不能为空",
				ErrVideoEditInputInvalid,
			)
	}

	gain, err :=
		normalizeVideoEditGain(
			req.Gain,
		)
	if err != nil {
		return nil, err
	}

	courseware,
		scopedActor,
		release,
		err :=
		s.beginVideoEditOperation(
			ctx,
			req.CoursewareID,
			req.Actor,
			"配音混入成片",
		)
	if err != nil {
		return nil, err
	}
	defer release()

	videoInput, err :=
		loadVideoEditInput(
			ctx,
			req.CoursewareID,
			assetID,
			models.CWAssetTypeVideo,
		)
	if err != nil {
		return nil, err
	}

	subtitle, err :=
		repository.GetCoursewareSubtitleForCourseware(
			ctx,
			req.CoursewareID,
			subtitleID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareSubtitleNotFound,
				err,
			)
	}

	if err :=
		validateCoursewareEditorDraftSubtitleOwner(
			ctx,
			courseware,
			scopedActor,
			subtitle,
		); err != nil {
		return nil, err
	}

	if subtitle.UpdatedAt == nil {
		return nil,
			ErrCoursewareSubtitleMutationConflict
	}

	var segments []models.SubtitleSegment

	if err := json.Unmarshal(
		[]byte(subtitle.Segments),
		&segments,
	); err != nil {
		return nil,
			fmt.Errorf(
				"%w: 字幕片段不是合法JSON数组",
				ErrVideoEditInputInvalid,
			)
	}

	if len(segments) == 0 {
		return nil,
			fmt.Errorf(
				"%w: 字幕轨没有可用片段",
				ErrVideoEditInputInvalid,
			)
	}

	narrations := make(
		[]narrationInput,
		0,
		len(segments),
	)
	skipped := 0

	for _, segment := range segments {
		if strings.TrimSpace(
			segment.TTSAudioURL,
		) == "" {
			continue
		}

		if !isFiniteVideoEditNumber(
			segment.StartSec,
		) {
			return nil,
				fmt.Errorf(
					"%w: 字幕起始时间无效",
					ErrVideoEditInputInvalid,
				)
		}

		path :=
			resolveCoursewareGeneratedURLPath(
				req.CoursewareID,
				segment.TTSAudioURL,
				"tts",
			)
		if path == "" {
			videoEditLog.Warn(
				"TTS文件不属于当前课件或已丢失",
				"segment_id", segment.ID,
			)
			skipped++
			continue
		}

		info, statErr := os.Stat(path)
		if statErr != nil ||
			!info.Mode().IsRegular() {
			skipped++
			continue
		}

		startSec := segment.StartSec
		if startSec < 0 {
			startSec = 0
		}

		narrations = append(
			narrations,
			narrationInput{
				Path:     path,
				StartSec: startSec,
				Size:     info.Size(),
				ModTime:  info.ModTime(),
				Mode:     info.Mode(),
			},
		)
	}

	if len(narrations) == 0 {
		return nil,
			fmt.Errorf(
				"%w: 该字幕轨没有当前课件可用的TTS音频",
				ErrVideoEditInputInvalid,
			)
	}

	if len(narrations) > 100 {
		return nil,
			fmt.Errorf(
				"%w: 配音条数不能超过100条",
				ErrVideoEditInputInvalid,
			)
	}

	pageID, err :=
		validateVideoEditInheritedPage(
			ctx,
			req.CoursewareID,
			videoInput.Asset.PageID,
		)
	if err != nil {
		return nil, err
	}

	hasAudio := videoHasAudioStream(
		ctx,
		videoInput.Path,
	)

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
			"narrated_*.mp4",
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

	scopedActor, err =
		reloadNarrationInputs(
			ctx,
			req.CoursewareID,
			subtitleID,
			scopedActor,
			subtitle,
			videoInput,
			narrations,
		)
	if err != nil {
		return nil, err
	}

	args := []string{
		"-y",
		"-i", videoInput.Path,
	}

	for _, narration := range narrations {
		args = append(
			args,
			"-i", narration.Path,
		)
	}

	filterParts := make(
		[]string,
		0,
		len(narrations)+1,
	)
	narrationLabels := strings.Builder{}

	for index, narration := range narrations {
		delayMilliseconds :=
			int(
				narration.StartSec *
					1000,
			)

		filterParts = append(
			filterParts,
			fmt.Sprintf(
				"[%d:a]volume=%.3f,adelay=%d:all=1[na%d]",
				index+1,
				gain,
				delayMilliseconds,
				index,
			),
		)
		narrationLabels.WriteString(
			fmt.Sprintf(
				"[na%d]",
				index,
			),
		)
	}

	switch {
	case hasAudio:
		filterParts = append(
			filterParts,
			fmt.Sprintf(
				"[0:a]%samix=inputs=%d:normalize=0[aout]",
				narrationLabels.String(),
				len(narrations)+1,
			),
		)

	case len(narrations) == 1:
		filterParts = append(
			filterParts,
			"[na0]anull[aout]",
		)

	default:
		filterParts = append(
			filterParts,
			fmt.Sprintf(
				"%samix=inputs=%d:normalize=0[aout]",
				narrationLabels.String(),
				len(narrations),
			),
		)
	}

	args = append(
		args,
		"-filter_complex",
		strings.Join(
			filterParts,
			";",
		),
		"-map", "0:v",
		"-map", "[aout]",
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "128k",
		"-shortest",
		"-movflags", "+faststart",
		outputPath,
	)

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		args...,
	)

	output, err :=
		command.CombinedOutput()
	if err != nil {
		videoEditLog.Error(
			"FFmpeg配音混音失败",
			"error", err,
			"output", string(output),
		)

		return nil,
			fmt.Errorf(
				"配音混音失败: %w",
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

	scopedActor, err =
		reloadNarrationInputs(
			ctx,
			req.CoursewareID,
			subtitleID,
			scopedActor,
			subtitle,
			videoInput,
			narrations,
		)
	if err != nil {
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
		CoursewareID:  req.CoursewareID,
		PageID:        pageID,
		PlaceholderID: "",
		AssetType:     models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf(
			"配音合成(%d条旁白)",
			len(narrations),
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
				"记录配音视频失败: %w",
				err,
			)
	}

	keepOutput = true

	message := fmt.Sprintf(
		"配音合成完成：混入%d条旁白，时长%s",
		len(narrations),
		duration,
	)
	if skipped > 0 {
		message += fmt.Sprintf(
			"（%d条因文件缺失或归属异常被跳过）",
			skipped,
		)
	}

	return &MixNarrationResponse{
		AssetID:        newAsset.ID,
		URL:            localURL,
		Duration:       duration,
		NarrationCount: len(narrations),
		SkippedCount:   skipped,
		Message:        message,
	}, nil
}

// reloadNarrationInputs 重新授权并复验视频、字幕和TTS文件。
func reloadNarrationInputs(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	actor *CoursewareActorContext,
	expectedSubtitle *models.CoursewareSubtitle,
	videoInput *videoEditInputSnapshot,
	narrations []narrationInput,
) (
	*CoursewareActorContext,
	error,
) {
	courseware,
		scopedActor,
		err :=
		reauthorizeVideoEditOwner(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		reloadVideoEditInputUnchanged(
			ctx,
			coursewareID,
			videoInput,
			models.CWAssetTypeVideo,
		); err != nil {
		return nil, err
	}

	latestSubtitle, err :=
		repository.GetCoursewareSubtitleForCourseware(
			ctx,
			coursewareID,
			subtitleID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareSubtitleNotFound,
				err,
			)
	}

	if !coursewareSubtitleRevisionUnchanged(
		expectedSubtitle,
		latestSubtitle,
	) {
		return nil,
			ErrCoursewareSubtitleMutationConflict
	}

	if err :=
		validateCoursewareEditorDraftSubtitleOwner(
			ctx,
			courseware,
			scopedActor,
			latestSubtitle,
		); err != nil {
		return nil, err
	}

	for _, narration := range narrations {
		info, statErr :=
			os.Stat(
				narration.Path,
			)
		if statErr != nil ||
			!info.Mode().IsRegular() ||
			info.Size() !=
				narration.Size ||
			!info.ModTime().Equal(
				narration.ModTime,
			) ||
			info.Mode() !=
				narration.Mode {
			return nil,
				ErrVideoEditSourceChanged
		}
	}

	return scopedActor, nil
}
