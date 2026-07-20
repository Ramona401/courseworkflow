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
	"strings"

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
func (s *CoursewareSubtitleService) ExportSRT(
	ctx context.Context,
	coursewareID string,
	subtitleID string,
	actor *CoursewareActorContext,
) (string, error) {
	if _, err :=
		(&CoursewareService{}).
			LoadCoursewareForView(
				ctx,
				coursewareID,
				actor,
			); err != nil {
		return "", err
	}

	subtitle, err :=
		repository.GetCoursewareSubtitleForCourseware(
			ctx,
			coursewareID,
			subtitleID,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"%w: %v",
				ErrCoursewareSubtitleNotFound,
				err,
			)
	}

	var segments []models.SubtitleSegment

	if err := json.Unmarshal(
		[]byte(subtitle.Segments),
		&segments,
	); err != nil {
		return "",
			fmt.Errorf(
				"解析字幕片段失败: %w",
				err,
			)
	}

	return buildSRTContent(
		segments,
	), nil
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

// ==================== 字幕媒体衍生 ====================
//
// BurnInSubtitle与GenerateTTS已经拆分到独立文件：
//   - courseware_subtitle_burn_service.go
//   - courseware_subtitle_tts_service.go

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
