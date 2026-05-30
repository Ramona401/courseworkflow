package services

// courseware_subtitle_service.go — 课件字幕轨服务
//
// v0.42.8 新增：
//   - SRT 导出（SubtitleSegment[] → SRT 格式文本）
//   - FFmpeg 硬字幕烧录（subtitles filter + libass + 中文字体）
//
// v0.42.9 新增：
//   - TTS 批量配音（逐条字幕调用豆包 seed-tts-2.0 → 音频文件 → URL 回写 segments）
//   - TTS 音色列表查询
//
// 依赖：
//   - ffmpeg 已编译 --enable-libass（已验证 ✅）
//   - 中文字体 Droid Sans Fallback（已验证 ✅）
//   - 临时 SRT 文件写入 /tmp 后由 FFmpeg 读取

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var subtitleLog = logger.WithModule("subtitle")

// ==================== 服务定义 ====================

// CoursewareSubtitleService 字幕轨服务
type CoursewareSubtitleService struct {
	cfg *config.Config
}

// NewCoursewareSubtitleService 创建字幕轨服务
func NewCoursewareSubtitleService(cfg *config.Config) *CoursewareSubtitleService {
	return &CoursewareSubtitleService{cfg: cfg}
}

// ==================== SRT 导出 ====================

// ExportSRT 将字幕轨导出为 SRT 格式文本
// SRT 格式：
//
//	1
//	00:00:01,000 --> 00:00:03,500
//	这是第一条字幕
//
//	2
//	00:00:04,000 --> 00:00:06,500
//	这是第二条字幕
func (s *CoursewareSubtitleService) ExportSRT(ctx context.Context, subtitleID string) (string, error) {
	// 1. 查询字幕轨
	sub, err := repository.GetCoursewareSubtitleByID(ctx, subtitleID)
	if err != nil {
		return "", fmt.Errorf("字幕不存在: %w", err)
	}

	// 2. 解析 segments JSON
	var segments []models.SubtitleSegment
	if err := json.Unmarshal([]byte(sub.Segments), &segments); err != nil {
		return "", fmt.Errorf("解析字幕片段失败: %w", err)
	}

	// 3. 生成 SRT 文本
	return buildSRTContent(segments), nil
}

// buildSRTContent 从 SubtitleSegment 数组生成 SRT 格式文本
func buildSRTContent(segments []models.SubtitleSegment) string {
	var sb strings.Builder
	for i, seg := range segments {
		// 序号（从1开始）
		sb.WriteString(fmt.Sprintf("%d\n", i+1))
		// 时间码
		sb.WriteString(fmt.Sprintf("%s --> %s\n", formatSRTTime(seg.StartSec), formatSRTTime(seg.EndSec)))
		// 文本内容
		sb.WriteString(seg.Text)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// formatSRTTime 将秒数格式化为 SRT 时间码: HH:MM:SS,mmm
func formatSRTTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	totalMs := int(sec * 1000)
	hours := totalMs / 3600000
	totalMs %= 3600000
	minutes := totalMs / 60000
	totalMs %= 60000
	seconds := totalMs / 1000
	ms := totalMs % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, ms)
}

// ==================== FFmpeg 硬字幕烧录 ====================

// BurnInSubtitle 将字幕烧录到视频中（硬字幕）
//
// FFmpeg 命令：
//
//	ffmpeg -i input.mp4 \
//	  -vf "subtitles=subtitle.srt:force_style='FontName=Droid Sans Fallback,FontSize=24,
//	       PrimaryColour=&Hffffff&,OutlineColour=&H40000000&,Outline=2'" \
//	  -c:v libx264 -preset fast -crf 23 \
//	  -c:a copy \
//	  output.mp4
//
// 流程：
//  1. 查询字幕轨 → 解析 segments
//  2. 生成临时 SRT 文件到 /tmp
//  3. 获取视频资产的本地路径
//  4. 执行 FFmpeg 烧录
//  5. 创建新的视频资产记录
//  6. 清理临时 SRT 文件
func (s *CoursewareSubtitleService) BurnInSubtitle(ctx context.Context, subtitleID, videoAssetID, coursewareID, userID string) (*models.BurnInSubtitleResponse, error) {
	// 1. 校验课件权限
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 2. 查询字幕轨
	sub, err := repository.GetCoursewareSubtitleByID(ctx, subtitleID)
	if err != nil {
		return nil, fmt.Errorf("字幕不存在: %w", err)
	}
	if sub.CoursewareID != coursewareID {
		return nil, fmt.Errorf("字幕不属于此课件")
	}

	// 3. 解析 segments
	var segments []models.SubtitleSegment
	if err := json.Unmarshal([]byte(sub.Segments), &segments); err != nil {
		return nil, fmt.Errorf("解析字幕片段失败: %w", err)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("字幕为空，无法烧录")
	}

	// 4. 获取视频资产路径
	asset, err := repository.GetCWAssetByID(ctx, videoAssetID)
	if err != nil {
		return nil, fmt.Errorf("视频资产不存在: %w", err)
	}
	if asset.AssetType != models.CWAssetTypeVideo {
		return nil, fmt.Errorf("该资产不是视频类型")
	}
	if asset.CoursewareID != coursewareID {
		return nil, fmt.Errorf("视频不属于此课件")
	}
	sourcePath := resolveAssetPath(asset)
	if sourcePath == "" {
		return nil, fmt.Errorf("视频文件不存在")
	}

	// 5. 生成临时 SRT 文件
	srtContent := buildSRTContent(segments)
	srtFile := filepath.Join(os.TempDir(), fmt.Sprintf("sub_%s_%d.srt", subtitleID[:8], time.Now().UnixMilli()))
	if err := os.WriteFile(srtFile, []byte(srtContent), 0644); err != nil {
		return nil, fmt.Errorf("写入临时 SRT 文件失败: %w", err)
	}
	defer os.Remove(srtFile)

	// 6. 输出文件路径
	outputDir := filepath.Join(CWAssetUploadDir, coursewareID, "videos")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	outputName := fmt.Sprintf("%d_subtitled.mp4", time.Now().UnixMilli())
	outputPath := filepath.Join(outputDir, outputName)

	// 7. 解析样式配置
	style := models.DefaultSubtitleStyle
	if sub.StyleConfig != nil && *sub.StyleConfig != "" {
		var customStyle models.SubtitleStyleConfig
		if err := json.Unmarshal([]byte(*sub.StyleConfig), &customStyle); err == nil {
			if customStyle.FontSize > 0 {
				style.FontSize = customStyle.FontSize
			}
			if customStyle.FontColor != "" {
				style.FontColor = customStyle.FontColor
			}
			if customStyle.BgColor != "" {
				style.BgColor = customStyle.BgColor
			}
			if customStyle.Outline > 0 {
				style.Outline = customStyle.Outline
			}
			if customStyle.FontFamily != "" {
				style.FontFamily = customStyle.FontFamily
			}
		}
	}

	// 8. 构建 ASS force_style 字符串
	// 颜色格式：ASS 用 &HBBGGRR& 格式（注意 BGR 顺序）
	fontName := style.FontFamily
	if fontName == "" {
		fontName = "Droid Sans Fallback" // 服务器已安装的中文字体
	}
	primaryColor := cssColorToASS(style.FontColor)
	outlineColor := cssColorToASS(style.BgColor)

	// SRT 文件路径中的特殊字符需要转义（冒号、反斜杠等）
	escapedSRT := strings.ReplaceAll(srtFile, ":", "\\:")
	escapedSRT = strings.ReplaceAll(escapedSRT, "'", "\\'")

	forceStyle := fmt.Sprintf(
		"FontName=%s,FontSize=%d,PrimaryColour=%s,OutlineColour=%s,Outline=%d",
		fontName, style.FontSize, primaryColor, outlineColor, style.Outline,
	)

	subtitleFilter := fmt.Sprintf("subtitles=%s:force_style='%s'", escapedSRT, forceStyle)

	// 9. 执行 FFmpeg 烧录
	subtitleLog.Info("开始字幕烧录",
		"courseware_id", coursewareID,
		"subtitle_id", subtitleID,
		"video_asset_id", videoAssetID,
		"segment_count", len(segments),
		"language", sub.Language,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg",
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
	output, err := cmd.CombinedOutput()
	if err != nil {
		subtitleLog.Error("FFmpeg字幕烧录失败", "error", err, "output", string(output))
		return nil, fmt.Errorf("字幕烧录失败: %w", err)
	}

	// 10. 获取时长和文件大小
	duration := getVideoDuration(outputPath)
	var fileSize int64
	if info, statErr := os.Stat(outputPath); statErr == nil {
		fileSize = info.Size()
	}

	// 11. 写入数据库
	localURL := CWAssetURLPrefix + filepath.Join(coursewareID, "videos", outputName)
	newAsset := &models.CoursewareAsset{
		CoursewareID:     coursewareID,
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf("字幕烧录(%s, %d条)", sub.Language, len(segments)),
		OssURL:           localURL,
		FileSize:         fileSize,
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, newAsset); err != nil {
		return nil, fmt.Errorf("记录烧录视频失败: %w", err)
	}

	subtitleLog.Info("字幕烧录完成",
		"subtitle_id", subtitleID,
		"new_asset_id", newAsset.ID,
		"duration", duration,
		"file_size", fileSize,
	)

	return &models.BurnInSubtitleResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Message:  fmt.Sprintf("字幕烧录完成，%s，%d条字幕，时长%s", sub.Language, len(segments), duration),
	}, nil
}

// ==================== v0.42.9 TTS 批量配音 ====================

// GenerateTTS 批量生成字幕 TTS 配音
//
// 流程：
//  1. 查询字幕轨 → 解析 segments
//  2. 筛选指定的 segment IDs（为空则全部）
//  3. 逐条调用豆包 TTS API 生成 mp3
//  4. 更新每条 segment 的 TTS 字段
//  5. 回写更新后的 segments 到数据库
func (s *CoursewareSubtitleService) GenerateTTS(ctx context.Context, subtitleID, coursewareID, userID string, voice string, speed float64, segmentIDs []string) (*models.GenerateTTSResponse, error) {
	// 1. 校验课件权限
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 2. 查询字幕轨
	sub, err := repository.GetCoursewareSubtitleByID(ctx, subtitleID)
	if err != nil {
		return nil, fmt.Errorf("字幕不存在: %w", err)
	}
	if sub.CoursewareID != coursewareID {
		return nil, fmt.Errorf("字幕不属于此课件")
	}

	// 3. 解析 segments
	var segments []models.SubtitleSegment
	if err := json.Unmarshal([]byte(sub.Segments), &segments); err != nil {
		return nil, fmt.Errorf("解析字幕片段失败: %w", err)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("字幕为空，无法生成配音")
	}

	// 4. 构建待处理集合
	targetSet := make(map[string]bool)
	if len(segmentIDs) > 0 {
		for _, id := range segmentIDs {
			targetSet[id] = true
		}
	}

	// 5. 加载 TTS 配置
	ttsCfg, err := ai.GetTTSConfig(s.cfg.GetAESKey())
	if err != nil {
		return nil, fmt.Errorf("加载TTS配置失败: %w", err)
	}

	// 6. 输出目录
	outputDir := filepath.Join(CWAssetUploadDir, coursewareID, "tts")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建TTS输出目录失败: %w", err)
	}

	// 7. 构建追踪上下文
	var traceCtx *ai.TraceContext
	if userID != "" {
		traceCtx = &ai.TraceContext{
			UserID:    &userID,
			SceneCode: "courseware_subtitle_tts",
		}
	}

	// 8. 逐条合成
	successCount := 0
	failCount := 0
	var errors []string

	subtitleLog.Info("开始批量TTS配音",
		"courseware_id", coursewareID,
		"subtitle_id", subtitleID,
		"total_segments", len(segments),
		"target_count", len(segmentIDs),
		"voice", voice,
	)

	for i := range segments {
		seg := &segments[i]

		// 跳过空文本
		if strings.TrimSpace(seg.Text) == "" {
			continue
		}

		// 如果指定了 segmentIDs，只处理指定的
		if len(targetSet) > 0 && !targetSet[seg.ID] {
			continue
		}

		// 生成文件名
		outputName := fmt.Sprintf("tts_%d_%s", time.Now().UnixMilli(), seg.ID[:8])

		// 调用 TTS API
		result, ttsErr := ai.SynthesizeSpeech(ctx, ttsCfg, seg.Text, voice, speed, outputDir, outputName, traceCtx)
		if ttsErr != nil {
			subtitleLog.Error("TTS合成单条失败",
				"segment_id", seg.ID,
				"text_preview", subtitleTruncate(seg.Text, 30),
				"error", ttsErr,
			)
			failCount++
			errors = append(errors, fmt.Sprintf("第%d条(%s): %s", i+1, subtitleTruncate(seg.Text, 15), ttsErr.Error()))
			continue
		}

		// 拼接公网URL
		audioURL := CWAssetURLPrefix + filepath.Join(coursewareID, "tts", outputName+".mp3")

		// 更新 segment 的 TTS 字段
		seg.TTSAudioURL = audioURL
		seg.TTSVoice = voice
		seg.TTSDuration = result.Duration
		seg.TTSGeneratedAt = time.Now().Format(time.RFC3339)

		successCount++
	}

	// 9. 回写更新后的 segments 到数据库
	updatedJSON, err := json.Marshal(segments)
	if err != nil {
		return nil, fmt.Errorf("序列化更新后的字幕失败: %w", err)
	}

	sub.Segments = string(updatedJSON)
	if err := repository.UpsertCoursewareSubtitle(ctx, sub); err != nil {
		return nil, fmt.Errorf("保存TTS结果到数据库失败: %w", err)
	}

	subtitleLog.Info("TTS批量配音完成",
		"subtitle_id", subtitleID,
		"success", successCount,
		"failed", failCount,
	)

	return &models.GenerateTTSResponse{
		SubtitleID:   subtitleID,
		SuccessCount: successCount,
		FailCount:    failCount,
		TotalCount:   successCount + failCount,
		Segments:     string(updatedJSON),
		Errors:       errors,
		Message:      fmt.Sprintf("TTS配音完成：成功%d条，失败%d条", successCount, failCount),
	}, nil
}

// ==================== 颜色格式转换辅助 ====================

// cssColorToASS 将 CSS 颜色 (#RRGGBB 或 #AARRGGBB) 转换为 ASS 格式 (&HAABBGGRR&)
// 输入: "#FFFFFF" → 输出: "&H00FFFFFF&"
// 输入: "#40000000" → 输出: "&H40000000&"（含透明度）
func cssColorToASS(cssColor string) string {
	color := strings.TrimPrefix(cssColor, "#")

	switch len(color) {
	case 6:
		// #RRGGBB → &H00BBGGRR&
		r, g, b := color[0:2], color[2:4], color[4:6]
		return fmt.Sprintf("&H00%s%s%s&", b, g, r)
	case 8:
		// #AARRGGBB → &HAABBGGRR&
		a, r, g, b := color[0:2], color[2:4], color[4:6], color[6:8]
		return fmt.Sprintf("&H%s%s%s%s&", a, b, g, r)
	default:
		// 兜底白色
		return "&H00FFFFFF&"
	}
}

// truncateForLog 截断字符串用于日志输出
func subtitleTruncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
