package services

// video_edit_advanced.go — 课件视频高级编辑服务
//
// v0.42.3 更新：转场效果从5种扩展到15种
//   - 基础: fade / fadeblack / fadewhite / dissolve
//   - 擦除: wipeleft / wiperight / wipeup / wipedown
//   - 滑动: slideleft / slideright / slideup / slidedown
//   - 图形: circleopen / circleclose
//   所有转场名与 FFmpeg xfade 滤镜原生支持的 transition 参数完全一致
//
// v0.42.2 新增：高级拼接功能（从video_edit_service.go拆分）
//
// 功能：
//   - AdvancedConcat: 多视频高级拼接（支持每段独立裁剪+转场效果）
//   - concatWithTransitions: FFmpeg xfade滤镜转场拼接（含降级兜底）
//
// 依赖：系统已安装 ffmpeg 6.1.1+

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 支持的转场效果列表（与前端TRANS常量保持一致） ====================
// 所有 key 均为 FFmpeg xfade 原生支持的 transition 名称
// 参考: https://ffmpeg.org/ffmpeg-filters.html#xfade

var validTransitions = map[string]bool{
	"none":        true,
	"fade":        true,
	"fadeblack":   true,
	"fadewhite":   true,
	"dissolve":    true,
	"wipeleft":    true,
	"wiperight":   true,
	"wipeup":      true,
	"wipedown":    true,
	"slideleft":   true,
	"slideright":  true,
	"slideup":     true,
	"slidedown":   true,
	"circleopen":  true,
	"circleclose": true,
}

// normalizeTransition 规范化转场名称，不支持的转场降级为fade
func normalizeTransition(trans string) string {
	if trans == "" {
		return "none"
	}
	if validTransitions[trans] {
		return trans
	}
	// 不识别的转场名降级为fade
	videoEditLog.Warn("不支持的转场效果，降级为fade", "transition", trans)
	return "fade"
}

// ==================== 高级拼接（支持独立裁剪+转场） ====================

// VideoClip 单个视频片段配置
type VideoClip struct {
	AssetID    string  `json:"asset_id"`            // 视频资产ID
	StartSec   float64 `json:"start_sec"`           // 裁剪起始秒数（0=从头开始）
	EndSec     float64 `json:"end_sec"`             // 裁剪结束秒数（0=到末尾）
	Transition string  `json:"transition,omitempty"` // 与下一段的转场效果
	TransDur   float64 `json:"trans_dur,omitempty"`  // 转场时长（秒），默认0.5
}

// AdvancedConcatRequest 高级拼接请求
type AdvancedConcatRequest struct {
	CoursewareID string      `json:"courseware_id"` // 课件ID
	Clips        []VideoClip `json:"clips"`         // 片段列表（按顺序）
	UserID       string      `json:"user_id"`       // 操作者ID
}

// AdvancedConcatResponse 高级拼接响应
type AdvancedConcatResponse struct {
	AssetID  string `json:"asset_id"`  // 新生成的资产ID
	URL      string `json:"url"`       // 输出视频URL
	Duration string `json:"duration"`  // 总时长
	Message  string `json:"message"`   // 提示信息
}

// AdvancedConcat 高级多视频拼接（支持每段独立裁剪+转场效果）
//
// 工作流程：
//  1. 逐段裁剪为临时文件（如有裁剪需求）
//  2. 如果无转场效果，用concat协议快速拼接（不重编码）
//  3. 如果有转场效果，用FFmpeg xfade滤镜（需要重编码，较慢但效果好）
func (s *VideoEditService) AdvancedConcat(ctx context.Context, req *AdvancedConcatRequest) (*AdvancedConcatResponse, error) {
	// 1. 参数校验
	if len(req.Clips) < 1 {
		return nil, fmt.Errorf("至少需要1个视频片段")
	}
	if len(req.Clips) > 10 {
		return nil, fmt.Errorf("单次最多处理10个视频片段")
	}

	// 2. 课件权限校验
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 3. 解析所有片段的本地文件路径 + 准备裁剪
	type preparedClip struct {
		sourcePath string
		startSec   float64
		endSec     float64
		transition string
		transDur   float64
	}
	var clips []preparedClip
	hasTransition := false

	for i, clip := range req.Clips {
		asset, err := repository.GetCWAssetByID(ctx, clip.AssetID)
		if err != nil {
			return nil, fmt.Errorf("片段%d: 视频资产不存在: %w", i+1, err)
		}
		if asset.AssetType != models.CWAssetTypeVideo {
			return nil, fmt.Errorf("片段%d: 不是视频类型", i+1)
		}
		if asset.CoursewareID != req.CoursewareID {
			return nil, fmt.Errorf("片段%d: 不属于此课件", i+1)
		}
		if asset.OssURL == "" || !strings.HasPrefix(asset.OssURL, CWAssetURLPrefix) {
			return nil, fmt.Errorf("片段%d: 无有效视频文件", i+1)
		}
		relativePath := asset.OssURL[len(CWAssetURLPrefix):]
		fullPath := filepath.Join(CWAssetUploadDir, relativePath)
		if !fileExists(fullPath) {
			return nil, fmt.Errorf("片段%d: 视频文件不存在", i+1)
		}

		// v0.42.3: 规范化转场名称
		trans := normalizeTransition(clip.Transition)
		transDur := clip.TransDur
		if transDur <= 0 {
			transDur = 0.5
		}
		if trans != "none" {
			hasTransition = true
		}

		clips = append(clips, preparedClip{
			sourcePath: fullPath,
			startSec:   clip.StartSec,
			endSec:     clip.EndSec,
			transition: trans,
			transDur:   transDur,
		})
	}

	// 4. 逐段裁剪为临时文件（如果有裁剪需求）
	var tempFiles []string
	defer func() {
		for _, f := range tempFiles {
			os.Remove(f)
		}
	}()

	var segmentPaths []string
	for i, clip := range clips {
		needsTrim := clip.startSec > 0 || clip.endSec > 0
		if !needsTrim {
			segmentPaths = append(segmentPaths, clip.sourcePath)
			continue
		}

		// 裁剪为临时文件
		tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("adv_seg_%d_%d.mp4", time.Now().UnixMilli(), i))
		tempFiles = append(tempFiles, tmpPath)

		args := []string{"-y"}
		if clip.startSec > 0 {
			args = append(args, "-ss", fmt.Sprintf("%.2f", clip.startSec))
		}
		args = append(args, "-i", clip.sourcePath)
		if clip.endSec > 0 {
			dur := clip.endSec - clip.startSec
			if dur < 0.5 {
				return nil, fmt.Errorf("片段%d: 裁剪后时长不足0.5秒", i+1)
			}
			args = append(args, "-t", fmt.Sprintf("%.2f", dur))
		}
		args = append(args, "-c", "copy", tmpPath)

		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			videoEditLog.Error("片段裁剪失败", "clip", i+1, "error", err, "output", string(output))
			return nil, fmt.Errorf("片段%d裁剪失败: %w", i+1, err)
		}
		segmentPaths = append(segmentPaths, tmpPath)
	}

	// 5. 输出文件准备
	outputDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "videos")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	outputName := fmt.Sprintf("%d_adv_concat_%d.mp4", time.Now().UnixMilli(), len(clips))
	outputPath := filepath.Join(outputDir, outputName)

	// 6. 执行拼接
	videoEditLog.Info("开始高级拼接",
		"courseware_id", req.CoursewareID,
		"clip_count", len(clips),
		"has_transition", hasTransition,
	)

	if len(segmentPaths) == 1 {
		// 单片段：直接复制（可能只是裁剪）
		cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", segmentPaths[0], "-c", "copy", outputPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			videoEditLog.Error("单片段复制失败", "error", err, "output", string(output))
			return nil, fmt.Errorf("处理失败: %w", err)
		}
	} else if !hasTransition {
		// 无转场：用concat协议快速拼接（不重编码）
		listFile := filepath.Join(os.TempDir(), fmt.Sprintf("adv_list_%d.txt", time.Now().UnixMilli()))
		var listContent strings.Builder
		for _, fp := range segmentPaths {
			listContent.WriteString(fmt.Sprintf("file '%s'\n", fp))
		}
		if err := os.WriteFile(listFile, []byte(listContent.String()), 0644); err != nil {
			return nil, fmt.Errorf("创建拼接清单失败: %w", err)
		}
		defer os.Remove(listFile)

		cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			videoEditLog.Error("无转场拼接失败", "error", err, "output", string(output))
			return nil, fmt.Errorf("拼接失败: %w", err)
		}
	} else {
		// 有转场：用xfade滤镜（需要重编码）
		if err := s.concatWithTransitions(ctx, segmentPaths, req.Clips, outputPath); err != nil {
			return nil, err
		}
	}

	// 7. 获取输出时长
	duration := getVideoDuration(outputPath)

	// 8. 获取文件大小
	var fileSize int64
	if info, err := os.Stat(outputPath); err == nil {
		fileSize = info.Size()
	}

	// 9. 写入数据库
	localURL := CWAssetURLPrefix + filepath.Join(req.CoursewareID, "videos", outputName)
	newAsset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf("高级拼接%d个片段", len(clips)),
		OssURL:           localURL,
		FileSize:         fileSize,
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, newAsset); err != nil {
		return nil, fmt.Errorf("记录视频失败: %w", err)
	}

	videoEditLog.Info("高级拼接完成",
		"asset_id", newAsset.ID,
		"duration", duration,
		"clip_count", len(clips),
		"has_transition", hasTransition,
	)

	return &AdvancedConcatResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Message:  fmt.Sprintf("成功处理%d个片段，总时长%s", len(clips), duration),
	}, nil
}

// concatWithTransitions 使用FFmpeg xfade滤镜拼接带转场效果的视频
// xfade需要逐对处理：每两个相邻片段用一次xfade
// v0.42.3: 转场名已通过normalizeTransition校验，均为FFmpeg原生支持的名称
func (s *VideoEditService) concatWithTransitions(ctx context.Context, segmentPaths []string, clips []VideoClip, outputPath string) error {
	if len(segmentPaths) < 2 {
		return fmt.Errorf("转场拼接至少需要2个片段")
	}

	// 构建FFmpeg复杂滤镜图
	var args []string
	args = append(args, "-y")

	// 添加所有输入
	for _, fp := range segmentPaths {
		args = append(args, "-i", fp)
	}

	// 构建filter_complex
	var filterParts []string
	n := len(segmentPaths)

	// 获取每段时长（用于计算xfade offset）
	durations := make([]float64, n)
	for i, fp := range segmentPaths {
		durStr := getVideoDuration(fp)
		durStr = strings.TrimSuffix(durStr, "s")
		var d float64
		fmt.Sscanf(durStr, "%f", &d)
		if d <= 0 {
			d = 5.0 // 兜底
		}
		durations[i] = d
	}

	// 逐对构建xfade：[0][1]→[v01], [v01][2]→[v012], ...
	offset := 0.0
	prevLabel := "[0:v]"
	prevALabel := "[0:a]"

	for i := 1; i < n; i++ {
		// v0.42.3: 使用normalizeTransition确保转场名有效
		trans := normalizeTransition(clips[i-1].Transition)
		transDur := clips[i-1].TransDur
		if trans == "none" || trans == "" {
			trans = "fade"
			transDur = 0.3
		}
		if transDur <= 0 {
			transDur = 0.5
		}

		// xfade offset = 上一段累计时长 - 转场时长
		offset += durations[i-1]
		xfadeOffset := offset - transDur
		if xfadeOffset < 0 {
			xfadeOffset = 0
		}

		// 视频xfade
		outVLabel := fmt.Sprintf("[v%d]", i)
		if i == n-1 {
			outVLabel = "[vout]"
		}
		filterParts = append(filterParts,
			fmt.Sprintf("%s[%d:v]xfade=transition=%s:duration=%.2f:offset=%.2f%s",
				prevLabel, i, trans, transDur, xfadeOffset, outVLabel))

		// 音频acrossfade
		outALabel := fmt.Sprintf("[a%d]", i)
		if i == n-1 {
			outALabel = "[aout]"
		}
		filterParts = append(filterParts,
			fmt.Sprintf("%s[%d:a]acrossfade=d=%.2f%s",
				prevALabel, i, transDur, outALabel))

		prevLabel = outVLabel
		prevALabel = outALabel
		offset -= transDur // 转场重叠部分扣除
	}

	filterComplex := strings.Join(filterParts, ";")
	args = append(args, "-filter_complex", filterComplex)
	args = append(args, "-map", "[vout]", "-map", "[aout]")
	args = append(args, "-c:v", "libx264", "-preset", "fast", "-crf", "23")
	args = append(args, "-c:a", "aac", "-b:a", "128k")
	args = append(args, outputPath)

	videoEditLog.Info("执行转场拼接",
		"filter", filterComplex,
		"segment_count", n,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		videoEditLog.Error("转场拼接失败", "error", err, "output", string(output))
		// 降级：无转场直接拼接
		videoEditLog.Info("降级为无转场拼接")
		listFile := filepath.Join(os.TempDir(), fmt.Sprintf("fallback_list_%d.txt", time.Now().UnixMilli()))
		var listContent strings.Builder
		for _, fp := range segmentPaths {
			listContent.WriteString(fmt.Sprintf("file '%s'\n", fp))
		}
		if err2 := os.WriteFile(listFile, []byte(listContent.String()), 0644); err2 != nil {
			return fmt.Errorf("降级拼接失败: %w", err2)
		}
		defer os.Remove(listFile)
		cmd2 := exec.CommandContext(ctx, "ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputPath)
		if output2, err2 := cmd2.CombinedOutput(); err2 != nil {
			videoEditLog.Error("降级拼接也失败", "error", err2, "output", string(output2))
			return fmt.Errorf("拼接失败: %w", err2)
		}
	}
	return nil
}
