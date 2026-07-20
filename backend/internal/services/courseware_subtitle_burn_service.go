package services

// courseware_subtitle_burn_service.go — 作者专属硬字幕烧录
//
// 安全顺序：
//
//   初始作者授权
//   → 字幕与视频复合读取
//   → FFmpeg前重新授权和版本校验
//   → 执行FFmpeg
//   → 资产写库前再次授权、版本和视频校验
//   → 创建新视频资产
//
// 任一后置步骤失败时自动删除本轮输出视频。

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

// BurnInSubtitle 将字幕烧录到课件作者自己的视频资产中。
func (s *CoursewareSubtitleService) BurnInSubtitle(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	videoAssetID string,
	actor *CoursewareActorContext,
) (
	*models.BurnInSubtitleResponse,
	error,
) {
	videoAssetID = strings.TrimSpace(videoAssetID)
	if videoAssetID == "" {
		return nil,
			fmt.Errorf(
				"%w: video_asset_id不能为空",
				ErrCoursewareSubtitleInputInvalid,
			)
	}

	scopedActor,
		subtitle,
		err :=
		loadOwnedCoursewareSubtitleMediaInputs(
			ctx,
			coursewareID,
			subtitleID,
			actor,
		)
	if err != nil {
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
				"解析字幕片段失败: %w",
				err,
			)
	}
	if len(segments) == 0 {
		return nil,
			fmt.Errorf("字幕为空，无法烧录")
	}

	if _, _, err =
		loadCoursewareSubtitleVideoAsset(
			ctx,
			coursewareID,
			videoAssetID,
		); err != nil {
		return nil, err
	}

	subtitleToken :=
		coursewareSubtitleSafeFileToken(
			subtitleID,
		)

	srtFile := filepath.Join(
		os.TempDir(),
		fmt.Sprintf(
			"sub_%s_%d.srt",
			subtitleToken,
			time.Now().UnixNano(),
		),
	)

	if err := os.WriteFile(
		srtFile,
		[]byte(buildSRTContent(segments)),
		0644,
	); err != nil {
		return nil,
			fmt.Errorf(
				"写入临时SRT文件失败: %w",
				err,
			)
	}
	defer os.Remove(srtFile)

	outputDir := filepath.Join(
		CWAssetUploadDir,
		coursewareID,
		"videos",
	)
	if err := os.MkdirAll(
		outputDir,
		0755,
	); err != nil {
		return nil,
			fmt.Errorf(
				"创建输出目录失败: %w",
				err,
			)
	}

	outputName := fmt.Sprintf(
		"%d_subtitled.mp4",
		time.Now().UnixNano(),
	)
	outputPath := filepath.Join(
		outputDir,
		outputName,
	)

	keepOutput := false

	defer func() {
		if !keepOutput {
			_ = os.Remove(outputPath)
		}
	}()

	scopedActor,
		_,
		err =
		reloadOwnedCoursewareSubtitleMediaInputs(
			ctx,
			coursewareID,
			subtitleID,
			scopedActor,
			subtitle,
		)
	if err != nil {
		return nil, err
	}

	asset,
		sourcePath,
		err :=
		loadCoursewareSubtitleVideoAsset(
			ctx,
			coursewareID,
			videoAssetID,
		)
	if err != nil {
		return nil, err
	}

	style := models.DefaultSubtitleStyle

	if subtitle.StyleConfig != nil &&
		strings.TrimSpace(
			*subtitle.StyleConfig,
		) != "" {
		var customStyle models.SubtitleStyleConfig

		if err := json.Unmarshal(
			[]byte(*subtitle.StyleConfig),
			&customStyle,
		); err == nil {
			if customStyle.FontSize > 0 {
				style.FontSize =
					customStyle.FontSize
			}
			if customStyle.FontColor != "" {
				style.FontColor =
					customStyle.FontColor
			}
			if customStyle.BgColor != "" {
				style.BgColor =
					customStyle.BgColor
			}
			if customStyle.Outline > 0 {
				style.Outline =
					customStyle.Outline
			}
			if customStyle.FontFamily != "" {
				style.FontFamily =
					customStyle.FontFamily
			}
		}
	}

	fontName := style.FontFamily
	if fontName == "" {
		fontName = "Droid Sans Fallback"
	}

	escapedSRT := strings.ReplaceAll(
		srtFile,
		":",
		"\\:",
	)
	escapedSRT = strings.ReplaceAll(
		escapedSRT,
		"'",
		"\\'",
	)

	forceStyle := fmt.Sprintf(
		"FontName=%s,FontSize=%d,PrimaryColour=%s,OutlineColour=%s,Outline=%d",
		fontName,
		style.FontSize,
		cssColorToASS(style.FontColor),
		cssColorToASS(style.BgColor),
		style.Outline,
	)

	subtitleFilter := fmt.Sprintf(
		"subtitles=%s:force_style='%s'",
		escapedSRT,
		forceStyle,
	)

	subtitleLog.Info(
		"开始字幕烧录",
		"courseware_id", coursewareID,
		"subtitle_id", subtitleID,
		"video_asset_id", videoAssetID,
		"segment_count", len(segments),
		"language", subtitle.Language,
	)

	command := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", sourcePath,
		"-vf", subtitleFilter,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "copy",
		"-movflags", "+faststart",
		outputPath,
	)

	output, err := command.CombinedOutput()
	if err != nil {
		subtitleLog.Error(
			"FFmpeg字幕烧录失败",
			"error", err,
			"output", string(output),
		)

		return nil,
			fmt.Errorf(
				"字幕烧录失败: %w",
				err,
			)
	}

	scopedActor,
		_,
		err =
		reloadOwnedCoursewareSubtitleMediaInputs(
			ctx,
			coursewareID,
			subtitleID,
			scopedActor,
			subtitle,
		)
	if err != nil {
		return nil, err
	}

	latestAsset,
		_,
		err :=
		loadCoursewareSubtitleVideoAsset(
			ctx,
			coursewareID,
			videoAssetID,
		)
	if err != nil {
		return nil, err
	}

	if latestAsset.ID != asset.ID ||
		latestAsset.CoursewareID != asset.CoursewareID ||
		latestAsset.AssetType != asset.AssetType ||
		latestAsset.OssURL != asset.OssURL ||
		latestAsset.PublicOSSURL != asset.PublicOSSURL ||
		!coursewareSubtitleStringPointerEqual(
			latestAsset.PageID,
			asset.PageID,
		) {
		return nil,
			ErrCoursewareSubtitleMutationConflict
	}

	duration := getVideoDuration(outputPath)

	var fileSize int64

	if info, statErr :=
		os.Stat(outputPath); statErr == nil {
		fileSize = info.Size()
	}

	localURL :=
		CWAssetURLPrefix +
			filepath.Join(
				coursewareID,
				"videos",
				outputName,
			)

	newAsset := &models.CoursewareAsset{
		CoursewareID:  coursewareID,
		PageID:        asset.PageID,
		PlaceholderID: "",
		AssetType:     models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf(
			"字幕烧录(%s, %d条)",
			subtitle.Language,
			len(segments),
		),
		OssURL:   localURL,
		FileSize: fileSize,
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
				"记录烧录视频失败: %w",
				err,
			)
	}

	keepOutput = true

	return &models.BurnInSubtitleResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Message: fmt.Sprintf(
			"字幕烧录完成，%s，%d条字幕，时长%s",
			subtitle.Language,
			len(segments),
			duration,
		),
	}, nil
}
