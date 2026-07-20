package services

// video_edit_ffmpeg_helper.go — FFmpeg文件路径、临时文件与数值校验
//
// 这里集中处理所有不访问业务数据库的纯工具：
//   - 资产URL到本地路径的目录边界校验；
//   - 临时清单和随机输出文件名；
//   - FFprobe时长与音轨探测；
//   - 输出文件完整性检查；
//   - 裁剪区间、增益和有限数值校验。

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tedna/internal/models"
)

// resolveAssetPath 从正式资产记录解析安全本地路径。
//
// 只允许访问CWAssetUploadDir/{courseware_id}/...下的真实普通文件，
// 同时校验符号链接解析结果仍位于课件资产根目录内。
func resolveAssetPath(
	asset *models.CoursewareAsset,
) string {
	if asset == nil ||
		strings.TrimSpace(asset.CoursewareID) == "" ||
		!strings.HasPrefix(
			strings.TrimSpace(asset.OssURL),
			CWAssetURLPrefix,
		) {
		return ""
	}

	relativeURL := strings.TrimPrefix(
		strings.TrimSpace(asset.OssURL),
		CWAssetURLPrefix,
	)
	relativePath := filepath.Clean(
		filepath.FromSlash(relativeURL),
	)

	if relativePath == "." ||
		filepath.IsAbs(relativePath) ||
		relativePath == ".." ||
		strings.HasPrefix(
			relativePath,
			".."+string(os.PathSeparator),
		) {
		return ""
	}

	coursewareDir := filepath.Join(
		CWAssetUploadDir,
		asset.CoursewareID,
	)
	fullPath := filepath.Join(
		CWAssetUploadDir,
		relativePath,
	)

	if !pathWithinVideoEditRoot(
		coursewareDir,
		fullPath,
	) {
		return ""
	}

	resolvedPath, err :=
		filepath.EvalSymlinks(
			fullPath,
		)
	if err != nil ||
		!pathWithinVideoEditRoot(
			coursewareDir,
			resolvedPath,
		) {
		return ""
	}

	info, err := os.Stat(resolvedPath)
	if err != nil ||
		!info.Mode().IsRegular() {
		return ""
	}

	return resolvedPath
}

// resolveCoursewareGeneratedURLPath 解析当前课件生成目录内的文件。
//
// MixNarration使用本函数限制TTS文件只能来自：
//
//	CWAssetUploadDir/{courseware_id}/tts/...
func resolveCoursewareGeneratedURLPath(
	coursewareID string,
	rawURL string,
	requiredSubdir string,
) string {
	coursewareID = strings.TrimSpace(
		coursewareID,
	)
	requiredSubdir = strings.TrimSpace(
		requiredSubdir,
	)

	if coursewareID == "" ||
		requiredSubdir == "" ||
		!strings.HasPrefix(
			strings.TrimSpace(rawURL),
			CWAssetURLPrefix,
		) {
		return ""
	}

	relativeURL := strings.TrimPrefix(
		strings.TrimSpace(rawURL),
		CWAssetURLPrefix,
	)
	relativePath := filepath.Clean(
		filepath.FromSlash(relativeURL),
	)

	expectedDir := filepath.Join(
		CWAssetUploadDir,
		coursewareID,
		requiredSubdir,
	)
	fullPath := filepath.Join(
		CWAssetUploadDir,
		relativePath,
	)

	if !pathWithinVideoEditRoot(
		expectedDir,
		fullPath,
	) {
		return ""
	}

	resolvedPath, err :=
		filepath.EvalSymlinks(
			fullPath,
		)
	if err != nil ||
		!pathWithinVideoEditRoot(
			expectedDir,
			resolvedPath,
		) {
		return ""
	}

	info, err := os.Stat(resolvedPath)
	if err != nil ||
		!info.Mode().IsRegular() {
		return ""
	}

	return resolvedPath
}

// pathWithinVideoEditRoot 判断candidate是否位于root内部。
func pathWithinVideoEditRoot(
	root string,
	candidate string,
) bool {
	rootAbs, err := filepath.Abs(
		filepath.Clean(root),
	)
	if err != nil {
		return false
	}

	candidateAbs, err :=
		filepath.Abs(
			filepath.Clean(candidate),
		)
	if err != nil {
		return false
	}

	relative, err :=
		filepath.Rel(
			rootAbs,
			candidateAbs,
		)
	if err != nil {
		return false
	}

	return relative != ".." &&
		!strings.HasPrefix(
			relative,
			".."+string(os.PathSeparator),
		)
}

// createVideoEditOutputPath 创建随机输出文件名并移除空占位文件。
func createVideoEditOutputPath(
	outputDir string,
	pattern string,
) (
	string,
	string,
	error,
) {
	if err := os.MkdirAll(
		outputDir,
		0755,
	); err != nil {
		return "", "",
			fmt.Errorf(
				"创建输出目录失败: %w",
				err,
			)
	}

	tempFile, err :=
		os.CreateTemp(
			outputDir,
			pattern,
		)
	if err != nil {
		return "", "",
			fmt.Errorf(
				"创建随机输出路径失败: %w",
				err,
			)
	}

	outputPath := tempFile.Name()

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(outputPath)

		return "", "",
			fmt.Errorf(
				"关闭输出占位文件失败: %w",
				err,
			)
	}

	if err := os.Remove(outputPath); err != nil {
		return "", "",
			fmt.Errorf(
				"清理输出占位文件失败: %w",
				err,
			)
	}

	return outputPath,
		filepath.Base(outputPath),
		nil
}

// writeVideoEditConcatList 写入FFmpeg concat清单。
func writeVideoEditConcatList(
	paths []string,
) (
	string,
	error,
) {
	listFile, err :=
		os.CreateTemp(
			os.TempDir(),
			"tedna_video_concat_*.txt",
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"创建拼接清单失败: %w",
				err,
			)
	}

	listPath := listFile.Name()
	keep := false

	defer func() {
		_ = listFile.Close()

		if !keep {
			_ = os.Remove(listPath)
		}
	}()

	for _, rawPath := range paths {
		path := strings.TrimSpace(
			rawPath,
		)
		if path == "" ||
			strings.ContainsAny(
				path,
				"'\r\n\x00",
			) {
			return "",
				fmt.Errorf(
					"%w: 拼接文件路径无效",
					ErrVideoEditInputInvalid,
				)
		}

		if _, err := fmt.Fprintf(
			listFile,
			"file '%s'\n",
			path,
		); err != nil {
			return "",
				fmt.Errorf(
					"写入拼接清单失败: %w",
					err,
				)
		}
	}

	if err := listFile.Sync(); err != nil {
		return "",
			fmt.Errorf(
				"同步拼接清单失败: %w",
				err,
			)
	}

	keep = true
	return listPath, nil
}

// validateVideoEditOutput 检查FFmpeg输出是非空普通文件。
func validateVideoEditOutput(
	outputPath string,
) (
	os.FileInfo,
	error,
) {
	info, err := os.Stat(
		outputPath,
	)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Size() <= 0 {
		return nil,
			ErrVideoEditOutputInvalid
	}

	return info, nil
}

// cleanupVideoEditFile 删除本轮未提交成功的输出。
func cleanupVideoEditFile(
	path string,
) {
	if strings.TrimSpace(path) == "" {
		return
	}

	_ = os.Remove(path)
}

// probeMediaDurationSeconds 使用FFprobe读取媒体时长。
func probeMediaDurationSeconds(
	ctx context.Context,
	filePath string,
) (
	float64,
	error,
) {
	command := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	)

	output, err := command.Output()
	if err != nil {
		return 0,
			fmt.Errorf(
				"读取媒体时长失败: %w",
				err,
			)
	}

	duration, err :=
		strconv.ParseFloat(
			strings.TrimSpace(
				string(output),
			),
			64,
		)
	if err != nil ||
		!isFiniteVideoEditNumber(duration) ||
		duration <= 0 {
		return 0,
			fmt.Errorf(
				"媒体时长无效",
			)
	}

	return duration, nil
}

// getMediaDurationLabel 返回前端兼容的“秒数s”文本。
func getMediaDurationLabel(
	ctx context.Context,
	filePath string,
) string {
	duration, err :=
		probeMediaDurationSeconds(
			ctx,
			filePath,
		)
	if err != nil {
		return "未知"
	}

	return strconv.FormatFloat(
		duration,
		'f',
		3,
		64,
	) + "s"
}

// getVideoDuration 保留已有包内调用的兼容签名。
func getVideoDuration(
	filePath string,
) string {
	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			15*time.Second,
		)
	defer cancel()

	return getMediaDurationLabel(
		ctx,
		filePath,
	)
}

// videoHasAudioStream 探测视频是否含音频流。
func videoHasAudioStream(
	ctx context.Context,
	filePath string,
) bool {
	command := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "quiet",
		"-select_streams", "a",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		filePath,
	)

	output, err := command.Output()
	if err != nil {
		videoEditLog.Warn(
			"探测音频流失败，按无原声处理",
			"path", filePath,
			"error", err,
		)
		return false
	}

	return strings.TrimSpace(
		string(output),
	) != ""
}

// validateVideoEditTimeRange 校验裁剪区间。
func validateVideoEditTimeRange(
	startSec float64,
	endSec float64,
	sourceDuration float64,
	minDuration float64,
) error {
	if !isFiniteVideoEditNumber(startSec) ||
		!isFiniteVideoEditNumber(endSec) ||
		!isFiniteVideoEditNumber(sourceDuration) ||
		!isFiniteVideoEditNumber(minDuration) {
		return fmt.Errorf(
			"%w: 时间参数必须是有限数值",
			ErrVideoEditInputInvalid,
		)
	}

	if startSec < 0 {
		return fmt.Errorf(
			"%w: 起始时间不能为负数",
			ErrVideoEditInputInvalid,
		)
	}

	if endSec <= startSec {
		return fmt.Errorf(
			"%w: 结束时间必须大于起始时间",
			ErrVideoEditInputInvalid,
		)
	}

	if endSec-startSec < minDuration {
		return fmt.Errorf(
			"%w: 裁剪后时长至少%.1f秒",
			ErrVideoEditInputInvalid,
			minDuration,
		)
	}

	if sourceDuration <= 0 ||
		endSec > sourceDuration+0.05 {
		return fmt.Errorf(
			"%w: 裁剪结束时间超过源媒体时长",
			ErrVideoEditInputInvalid,
		)
	}

	return nil
}

// normalizeVideoEditGain 规范化旁白增益。
func normalizeVideoEditGain(
	gain float64,
) (
	float64,
	error,
) {
	if gain == 0 {
		return 1.0, nil
	}

	if !isFiniteVideoEditNumber(gain) ||
		gain < 0.1 ||
		gain > 3.0 {
		return 0,
			fmt.Errorf(
				"%w: gain必须在0.1到3.0之间",
				ErrVideoEditInputInvalid,
			)
	}

	return gain, nil
}

// isFiniteVideoEditNumber 判断浮点数是否有限。
func isFiniteVideoEditNumber(
	value float64,
) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0)
}

// fileExists 保留已有包内调用的兼容辅助。
func fileExists(
	path string,
) bool {
	info, err := os.Stat(path)
	return err == nil &&
		info.Mode().IsRegular()
}
