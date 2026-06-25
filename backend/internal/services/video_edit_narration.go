package services

// video_edit_narration.go — 课件视频「配音混入成片」服务（S-V1）
//
// 背景（迭代3.5子专项S）：
//   TTS批量配音生成后只存在于字幕轨各 SubtitleSegment 的 tts_audio_url 字段，
//   导出成片（AdvancedConcat）不混入配音音频——老师配了音，导出的视频里没有旁白。
//   本文件补上"配音功能闭环的最后一公里：把旁白混进成片"。
//
// 方案（两段式，不动现有拼接逻辑）：
//   第一段：AdvancedConcat 照常产出成片（含各片段原声、转场）。
//   第二段：本服务 MixNarration 读字幕轨中 tts_audio_url 非空的字幕条，
//           按 segment.start_sec 在全局时间轴定位，FFmpeg 一次合成：
//             - 每条旁白: [i:a]volume=增益,adelay=起始毫秒:all=1[naX]
//             - 与原声:   [0:a][na0][na1]...amix=inputs=N+1:normalize=0[aout]
//             - 输出:     -map 0:v -map "[aout]" -c:v copy（视频零重编码，增量秒级）
//
// 关键设计点：
//   - adelay 用 all=1 参数对所有声道统一延迟（兼容单声道mp3），毫秒取 start_sec×1000
//   - amix 关闭归一化（normalize=0），保证原声音量不被自动压低
//   - 旁白增益默认1.0，可调参（0.1~3.0），由前端透传
//   - 配音超出片尾按 -shortest 截断（视频流长度为准）
//   - 不变速不截断单条配音，只按 start 定位（朗读自然溢出比机械截断体验好，
//     时长冲突的提示标记列入 V3）
//   - 全片段无原声轨（如全静音视频）时只 mix 旁白
//   - 个别 TTS 音频文件丢失时跳过该条并记警告，不整体失败
//
// 依赖：系统已安装 ffmpeg 6.1.1+（amix normalize 参数需 ffmpeg 5.1+，已满足）

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

// ==================== 请求/响应结构 ====================

// MixNarrationRequest 配音混音请求
type MixNarrationRequest struct {
	CoursewareID string  // 课件ID
	AssetID      string  // 目标视频资产ID（通常是 AdvancedConcat 产出的成片）
	SubtitleID   string  // 字幕轨ID（其中 tts_audio_url 非空的条目将被混入）
	Gain         float64 // 旁白增益，默认1.0，范围0.1~3.0
	UserID       string  // 操作者ID
}

// MixNarrationResponse 配音混音响应
type MixNarrationResponse struct {
	AssetID        string `json:"asset_id"`        // 新生成的资产ID
	URL            string `json:"url"`             // 输出视频URL
	Duration       string `json:"duration"`        // 视频时长
	NarrationCount int    `json:"narration_count"` // 实际混入的旁白条数
	SkippedCount   int    `json:"skipped_count"`   // 因音频文件缺失被跳过的条数
	Message        string `json:"message"`         // 提示信息
}

// narrationItem 单条待混入的旁白（内部结构）
type narrationItem struct {
	localPath string  // mp3 本地完整路径
	startSec  float64 // 在全局时间轴的起始秒数
}

// ==================== 核心方法 ====================

// MixNarration 把字幕轨中已生成的TTS配音按时间轴混入指定视频，产出新视频资产
func (s *VideoEditService) MixNarration(ctx context.Context, req *MixNarrationRequest) (*MixNarrationResponse, error) {
	// 1. 参数校验
	if req.AssetID == "" {
		return nil, fmt.Errorf("asset_id不能为空")
	}
	if req.SubtitleID == "" {
		return nil, fmt.Errorf("subtitle_id不能为空")
	}
	// 增益默认值与范围钳制
	gain := req.Gain
	if gain <= 0 {
		gain = 1.0
	}
	if gain < 0.1 {
		gain = 0.1
	}
	if gain > 3.0 {
		gain = 3.0
	}

	// 2. 课件权限校验
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 3. 获取目标视频资产并解析本地路径
	asset, err := repository.GetCWAssetByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("视频资产不存在: %w", err)
	}
	if asset.AssetType != models.CWAssetTypeVideo {
		return nil, fmt.Errorf("该资产不是视频类型")
	}
	if asset.CoursewareID != req.CoursewareID {
		return nil, fmt.Errorf("资产不属于此课件")
	}
	videoPath := resolveAssetPath(asset)
	if videoPath == "" {
		return nil, fmt.Errorf("视频文件不存在")
	}

	// 4. 查询字幕轨并校验归属
	sub, err := repository.GetCoursewareSubtitleByID(ctx, req.SubtitleID)
	if err != nil {
		return nil, fmt.Errorf("字幕不存在: %w", err)
	}
	if sub.CoursewareID != req.CoursewareID {
		return nil, fmt.Errorf("字幕不属于此课件")
	}

	// 5. 解析 segments，收集已配音条目（tts_audio_url 非空且文件真实存在）
	var segments []models.SubtitleSegment
	if err := json.Unmarshal([]byte(sub.Segments), &segments); err != nil {
		return nil, fmt.Errorf("解析字幕片段失败: %w", err)
	}

	var narrations []narrationItem
	skipped := 0
	for _, seg := range segments {
		if seg.TTSAudioURL == "" {
			continue
		}
		// tts_audio_url 形如 /uploads/courseware-assets/{cwID}/tts/xxx.mp3
		// 与资产 oss_url 同前缀规则，复用前缀剥离逻辑解析本地路径
		if !strings.HasPrefix(seg.TTSAudioURL, CWAssetURLPrefix) {
			videoEditLog.Warn("TTS音频URL前缀异常，跳过该条",
				"segment_id", seg.ID, "url", seg.TTSAudioURL)
			skipped++
			continue
		}
		relativePath := seg.TTSAudioURL[len(CWAssetURLPrefix):]
		fullPath := filepath.Join(CWAssetUploadDir, relativePath)
		if !fileExists(fullPath) {
			videoEditLog.Warn("TTS音频文件不存在，跳过该条",
				"segment_id", seg.ID, "path", fullPath)
			skipped++
			continue
		}
		start := seg.StartSec
		if start < 0 {
			start = 0
		}
		narrations = append(narrations, narrationItem{
			localPath: fullPath,
			startSec:  start,
		})
	}

	if len(narrations) == 0 {
		return nil, fmt.Errorf("该字幕轨没有可用的配音音频（请先在TTS配音中生成，或检查音频文件是否完整）")
	}
	// FFmpeg 输入数量护栏（每条旁白一个输入）
	if len(narrations) > 100 {
		return nil, fmt.Errorf("配音条数过多（%d条，上限100条），请分段处理", len(narrations))
	}

	// 6. 探测目标视频是否含原声轨（决定 amix 输入构成）
	hasAudio := videoHasAudioStream(ctx, videoPath)

	// 7. 输出文件准备
	outputDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "videos")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	outputName := fmt.Sprintf("%d_narrated_%d.mp4", time.Now().UnixMilli(), len(narrations))
	outputPath := filepath.Join(outputDir, outputName)

	// 8. 构建 FFmpeg 命令
	//    输入0 = 成片视频；输入1..N = 各条旁白mp3
	args := []string{"-y", "-i", videoPath}
	for _, n := range narrations {
		args = append(args, "-i", n.localPath)
	}

	// filter_complex 构建：
	//   每条旁白: [i:a]volume=G,adelay=MS:all=1[naX]
	//   汇流: ([0:a])[na0][na1]...amix=inputs=K:normalize=0[aout]
	var filterParts []string
	narrLabels := ""
	for i, n := range narrations {
		delayMs := int(n.startSec * 1000)
		filterParts = append(filterParts,
			fmt.Sprintf("[%d:a]volume=%.2f,adelay=%d:all=1[na%d]", i+1, gain, delayMs, i))
		narrLabels += fmt.Sprintf("[na%d]", i)
	}

	if hasAudio {
		// 原声 + 全部旁白混音，normalize=0 保证原声音量不被压低
		filterParts = append(filterParts,
			fmt.Sprintf("[0:a]%samix=inputs=%d:normalize=0[aout]", narrLabels, len(narrations)+1))
	} else if len(narrations) == 1 {
		// 无原声且只有1条旁白：anull 直通即可，无需 amix
		filterParts = append(filterParts, "[na0]anull[aout]")
	} else {
		// 无原声多条旁白：旁白之间互相混音
		filterParts = append(filterParts,
			fmt.Sprintf("%samix=inputs=%d:normalize=0[aout]", narrLabels, len(narrations)))
	}

	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "0:v",      // 视频流取自成片
		"-map", "[aout]",   // 音频流取混音结果
		"-c:v", "copy",     // 视频零重编码（增量秒级）
		"-c:a", "aac",      // 混音后音频重编码为AAC
		"-b:a", "128k",
		"-shortest",        // 配音超出片尾按视频长度截断
		"-movflags", "+faststart", // moov前置，浏览器可渐进加载
		outputPath,
	)

	videoEditLog.Info("开始配音混音",
		"courseware_id", req.CoursewareID,
		"asset_id", req.AssetID,
		"subtitle_id", req.SubtitleID,
		"narration_count", len(narrations),
		"skipped", skipped,
		"has_origin_audio", hasAudio,
		"gain", gain,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		videoEditLog.Error("FFmpeg配音混音失败", "error", err, "output", string(output))
		return nil, fmt.Errorf("配音混音失败: %w", err)
	}

	// 9. 获取时长与文件大小
	duration := getVideoDuration(outputPath)
	var fileSize int64
	if info, statErr := os.Stat(outputPath); statErr == nil {
		fileSize = info.Size()
	}

	// 10. 写入数据库（新视频资产）
	localURL := CWAssetURLPrefix + filepath.Join(req.CoursewareID, "videos", outputName)
	newAsset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PageID:           asset.PageID, // 继承目标视频的页归属（目标通常是已继承源片段页归属的拼接成片）
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf("配音合成(%d条旁白)", len(narrations)),
		OssURL:           localURL,
		FileSize:         fileSize,
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, newAsset); err != nil {
		return nil, fmt.Errorf("记录配音视频失败: %w", err)
	}

	videoEditLog.Info("配音混音完成",
		"new_asset", newAsset.ID,
		"duration", duration,
		"narration_count", len(narrations),
		"file_size", fileSize,
	)

	msg := fmt.Sprintf("配音合成完成：混入%d条旁白，时长%s", len(narrations), duration)
	if skipped > 0 {
		msg += fmt.Sprintf("（%d条因音频文件缺失被跳过）", skipped)
	}

	return &MixNarrationResponse{
		AssetID:        newAsset.ID,
		URL:            localURL,
		Duration:       duration,
		NarrationCount: len(narrations),
		SkippedCount:   skipped,
		Message:        msg,
	}, nil
}

// ==================== 辅助函数 ====================

// videoHasAudioStream 用 ffprobe 探测视频文件是否含音频流
// 探测失败保守返回 false（按无原声处理，只混旁白，不会导致整体失败）
func videoHasAudioStream(ctx context.Context, filePath string) bool {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-select_streams", "a", // 只选音频流
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		videoEditLog.Warn("探测音频流失败，按无原声处理", "path", filePath, "error", err)
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}
