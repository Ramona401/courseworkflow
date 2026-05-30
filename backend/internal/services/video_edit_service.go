package services

// video_edit_service.go — 课件视频编辑服务
//
// v0.42.5 修复：
//   - MuteVideo: 改为添加静默音频轨道替代-an移除音轨
//     根因：-an产出的video-only MP4在浏览器<video>元素中canplay/seeking行为异常导致卡顿
//     修复：-f lavfi -i anullsrc + -c:a aac -b:a 1k 生成极低码率静默音轨
//     效果：浏览器看到正常双轨道MP4，解码行为与原始视频一致，几乎不增加文件体积
//
// v0.42.4 新增：
//   - MuteVideo: 去除视频音轨（保留画面，输出静音MP4）
//   - ExtractAudio: 从视频中分离音频轨道（输出MP3文件）
//
// v0.42.1 原有：
//   - ConcatVideos: 多视频顺序拼接（帧动画串联）
//   - TrimVideo: 视频裁剪（截取指定起止时间段）
//
// 依赖：系统已安装 ffmpeg（apt install ffmpeg）
// 存储：输出文件保存到 /uploads/courseware-assets/{courseware_id}/videos/
// 限制：单次拼接最多10个视频，裁剪后时长至少1秒

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var videoEditLog = logger.WithModule("video_edit")

// ==================== 服务定义 ====================

// VideoEditService 视频编辑服务
type VideoEditService struct {
	cfg *config.Config
}

// NewVideoEditService 创建视频编辑服务
func NewVideoEditService(cfg *config.Config) *VideoEditService {
	return &VideoEditService{cfg: cfg}
}

// ==================== 视频拼接 ====================

// ConcatVideosRequest 视频拼接请求
type ConcatVideosRequest struct {
	CoursewareID string   // 课件ID
	AssetIDs     []string // 要拼接的视频资产ID列表（按顺序）
	UserID       string   // 操作者ID
}

// ConcatVideosResponse 视频拼接响应
type ConcatVideosResponse struct {
	AssetID  string `json:"asset_id"`  // 新生成的资产ID
	URL      string `json:"url"`       // 拼接后视频URL
	Duration string `json:"duration"`  // 总时长
	Message  string `json:"message"`   // 提示信息
}

// ConcatVideos 多视频顺序拼接
func (s *VideoEditService) ConcatVideos(ctx context.Context, req *ConcatVideosRequest) (*ConcatVideosResponse, error) {
	// 1. 校验参数
	if len(req.AssetIDs) < 2 {
		return nil, fmt.Errorf("至少需要2个视频才能拼接")
	}
	if len(req.AssetIDs) > 10 {
		return nil, fmt.Errorf("单次最多拼接10个视频")
	}

	// 2. 校验课件权限
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 3. 获取所有视频资产的本地文件路径
	var filePaths []string
	for _, assetID := range req.AssetIDs {
		asset, err := repository.GetCWAssetByID(ctx, assetID)
		if err != nil {
			return nil, fmt.Errorf("视频资产 %s 不存在: %w", assetID, err)
		}
		if asset.AssetType != models.CWAssetTypeVideo {
			return nil, fmt.Errorf("资产 %s 不是视频类型", assetID)
		}
		if asset.CoursewareID != req.CoursewareID {
			return nil, fmt.Errorf("资产 %s 不属于此课件", assetID)
		}
		if asset.OssURL == "" || !strings.HasPrefix(asset.OssURL, CWAssetURLPrefix) {
			return nil, fmt.Errorf("资产 %s 无有效视频文件", assetID)
		}
		relativePath := asset.OssURL[len(CWAssetURLPrefix):]
		fullPath := filepath.Join(CWAssetUploadDir, relativePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("视频文件不存在: %s", assetID)
		}
		filePaths = append(filePaths, fullPath)
	}

	// 4. 创建FFmpeg拼接清单文件
	listFile := filepath.Join(os.TempDir(), fmt.Sprintf("concat_%d.txt", time.Now().UnixMilli()))
	var listContent strings.Builder
	for _, fp := range filePaths {
		listContent.WriteString(fmt.Sprintf("file '%s'\n", fp))
	}
	if err := os.WriteFile(listFile, []byte(listContent.String()), 0644); err != nil {
		return nil, fmt.Errorf("创建拼接清单失败: %w", err)
	}
	defer os.Remove(listFile)

	// 5. 输出文件路径
	outputDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "videos")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	outputName := fmt.Sprintf("%d_concat_%d.mp4", time.Now().UnixMilli(), len(req.AssetIDs))
	outputPath := filepath.Join(outputDir, outputName)

	// 6. 执行FFmpeg拼接（同编码直接拷贝，不重编码，速度很快）
	videoEditLog.Info("开始视频拼接",
		"courseware_id", req.CoursewareID,
		"video_count", len(req.AssetIDs),
		"output", outputPath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",           // 覆盖已有文件
		"-f", "concat", // 拼接模式
		"-safe", "0",   // 允许绝对路径
		"-i", listFile, // 拼接清单
		"-c", "copy",   // 直接拷贝流，不重编码
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		videoEditLog.Error("FFmpeg拼接失败", "error", err, "output", string(output))
		return nil, fmt.Errorf("视频拼接失败: %w", err)
	}

	// 7. 获取输出视频时长
	duration := getVideoDuration(outputPath)

	// 8. 写入数据库
	localURL := CWAssetURLPrefix + filepath.Join(req.CoursewareID, "videos", outputName)
	asset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf("拼接%d个视频片段", len(req.AssetIDs)),
		OssURL:           localURL,
		FileSize:         0,
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, asset); err != nil {
		return nil, fmt.Errorf("记录拼接视频失败: %w", err)
	}

	videoEditLog.Info("视频拼接完成",
		"asset_id", asset.ID,
		"duration", duration,
		"video_count", len(req.AssetIDs),
	)

	return &ConcatVideosResponse{
		AssetID:  asset.ID,
		URL:      localURL,
		Duration: duration,
		Message:  fmt.Sprintf("成功拼接%d个视频，总时长%s", len(req.AssetIDs), duration),
	}, nil
}

// ==================== 视频裁剪 ====================

// TrimVideoRequest 视频裁剪请求
type TrimVideoRequest struct {
	CoursewareID string  // 课件ID
	AssetID      string  // 原视频资产ID
	StartSec     float64 // 起始秒数（如 1.5）
	EndSec       float64 // 结束秒数（如 4.0）
	UserID       string  // 操作者ID
}

// TrimVideoResponse 视频裁剪响应
type TrimVideoResponse struct {
	AssetID  string `json:"asset_id"`  // 新生成的资产ID
	URL      string `json:"url"`       // 裁剪后视频URL
	Duration string `json:"duration"`  // 裁剪后时长
	Message  string `json:"message"`   // 提示信息
}

// TrimVideo 视频裁剪（截取指定时间段）
func (s *VideoEditService) TrimVideo(ctx context.Context, req *TrimVideoRequest) (*TrimVideoResponse, error) {
	// 1. 校验参数
	if req.StartSec < 0 {
		return nil, fmt.Errorf("起始时间不能为负数")
	}
	if req.EndSec <= req.StartSec {
		return nil, fmt.Errorf("结束时间必须大于起始时间")
	}
	if req.EndSec-req.StartSec < 1.0 {
		return nil, fmt.Errorf("裁剪后时长至少1秒")
	}

	// 2. 校验课件权限
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 3. 获取原视频文件路径
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

	sourcePath := ""
	if asset.OssURL != "" && strings.HasPrefix(asset.OssURL, CWAssetURLPrefix) {
		relativePath := asset.OssURL[len(CWAssetURLPrefix):]
		sourcePath = filepath.Join(CWAssetUploadDir, relativePath)
	}
	if sourcePath == "" || !fileExists(sourcePath) {
		return nil, fmt.Errorf("原视频文件不存在")
	}

	// 4. 输出文件路径
	outputDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "videos")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	outputName := fmt.Sprintf("%d_trim_%.0f-%.0f.mp4", time.Now().UnixMilli(), req.StartSec, req.EndSec)
	outputPath := filepath.Join(outputDir, outputName)

	// 5. 执行FFmpeg裁剪
	duration := req.EndSec - req.StartSec
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-ss", fmt.Sprintf("%.2f", req.StartSec), // 起始时间
		"-i", sourcePath,
		"-t", fmt.Sprintf("%.2f", duration), // 持续时长
		"-c", "copy",                        // 不重编码
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		videoEditLog.Error("FFmpeg裁剪失败", "error", err, "output", string(output))
		return nil, fmt.Errorf("视频裁剪失败: %w", err)
	}

	// 6. 获取输出视频时长
	actualDuration := getVideoDuration(outputPath)

	// 7. 写入数据库
	localURL := CWAssetURLPrefix + filepath.Join(req.CoursewareID, "videos", outputName)
	newAsset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: fmt.Sprintf("裁剪 %.1fs-%.1fs", req.StartSec, req.EndSec),
		OssURL:           localURL,
		FileSize:         0,
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, newAsset); err != nil {
		return nil, fmt.Errorf("记录裁剪视频失败: %w", err)
	}

	videoEditLog.Info("视频裁剪完成",
		"source_asset", req.AssetID,
		"new_asset", newAsset.ID,
		"start", req.StartSec,
		"end", req.EndSec,
		"duration", actualDuration,
	)

	return &TrimVideoResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: actualDuration,
		Message:  fmt.Sprintf("裁剪完成，截取 %.1f-%.1f 秒，时长%s", req.StartSec, req.EndSec, actualDuration),
	}, nil
}

// ==================== 视频静音（v0.42.4新增，v0.42.5修复） ====================

// MuteVideoRequest 视频静音请求
type MuteVideoRequest struct {
	CoursewareID string // 课件ID
	AssetID      string // 原视频资产ID
	UserID       string // 操作者ID
}

// MuteVideoResponse 视频静音响应
type MuteVideoResponse struct {
	AssetID  string `json:"asset_id"`  // 新生成的静音视频资产ID
	URL      string `json:"url"`       // 静音视频URL
	Duration string `json:"duration"`  // 视频时长（不变）
	Message  string `json:"message"`   // 提示信息
}

// MuteVideo 去除视频原始音轨，替换为静默音轨，生成新的静音视频
//
// v0.42.5修复：改为添加静默音频轨道替代-an移除音轨
// 根因：-an产出的video-only MP4在浏览器<video>元素中存在兼容性问题——
//   Chrome/Edge对无音频轨道的MP4文件canplay/canplaythrough事件触发时机不一致，
//   seeking时缺少音频时钟参考导致帧定位延迟，表现为前端换源后播放卡顿/断续。
// 修复：用-f lavfi -i anullsrc生成静默音频源 + -c:a aac -b:a 1k极低码率编码，
//   -map 0:v:0 -map 1:a:0 精确映射输入0的视频+输入1的音频，
//   产出正常双轨道MP4，浏览器解码行为与原始视频完全一致。
//   文件体积仅增加约1KB/秒（静默AAC极低码率），速度与纯-c:v copy几乎相同。
func (s *VideoEditService) MuteVideo(ctx context.Context, req *MuteVideoRequest) (*MuteVideoResponse, error) {
	// 1. 参数校验
	if req.AssetID == "" {
		return nil, fmt.Errorf("asset_id不能为空")
	}

	// 2. 课件权限校验
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 3. 获取原视频文件路径
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

	sourcePath := resolveAssetPath(asset)
	if sourcePath == "" {
		return nil, fmt.Errorf("原视频文件不存在")
	}

	// 4. 输出文件路径
	outputDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "videos")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	outputName := fmt.Sprintf("%d_muted.mp4", time.Now().UnixMilli())
	outputPath := filepath.Join(outputDir, outputName)

	// 5. 执行FFmpeg：用静默音频替代原始音轨
	// v0.42.5: 不再使用-an（移除音轨），改为添加静默音频轨道
	// -f lavfi -i anullsrc: 生成无限静默音频源
	// -map 0:v:0: 取第一输入（原视频）的视频流
	// -map 1:a:0: 取第二输入（静默源）的音频流
	// -c:v copy: 视频流直接拷贝不重编码（极快）
	// -c:a aac -b:a 1k: 静默音频用AAC极低码率编码（几乎不增加体积）
	// -shortest: 静默音频长度对齐视频长度（anullsrc是无限的）
	// -avoid_negative_ts make_zero: 确保时间戳从零开始
	// -movflags +faststart: moov原子前置，浏览器可渐进加载
	videoEditLog.Info("开始视频静音（静默音轨替换模式）",
		"courseware_id", req.CoursewareID,
		"source_asset", req.AssetID,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", sourcePath,                                          // 输入0: 原始视频
		"-f", "lavfi", "-i", "anullsrc=channel_layout=stereo:sample_rate=44100", // 输入1: 静默音频源
		"-map", "0:v:0",      // 取原视频的视频流
		"-map", "1:a:0",      // 取静默源的音频流（替代原始音轨）
		"-c:v", "copy",       // 视频不重编码
		"-c:a", "aac",        // 静默音频AAC编码
		"-b:a", "1k",         // 极低码率（静默音频不需要质量）
		"-shortest",          // 音频长度对齐视频长度
		"-avoid_negative_ts", "make_zero",
		"-movflags", "+faststart",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		videoEditLog.Error("FFmpeg静音失败", "error", err, "output", string(output))
		return nil, fmt.Errorf("视频静音失败: %w", err)
	}

	// 6. 获取时长和文件大小
	duration := getVideoDuration(outputPath)
	var fileSize int64
	if info, statErr := os.Stat(outputPath); statErr == nil {
		fileSize = info.Size()
	}

	// 7. 写入数据库
	localURL := CWAssetURLPrefix + filepath.Join(req.CoursewareID, "videos", outputName)
	newAsset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PlaceholderID:    "",
		AssetType:        models.CWAssetTypeVideo,
		GenerationPrompt: "静音处理（静默音轨替换）",
		OssURL:           localURL,
		FileSize:         fileSize,
		MimeType:         "video/mp4",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, newAsset); err != nil {
		return nil, fmt.Errorf("记录静音视频失败: %w", err)
	}

	videoEditLog.Info("视频静音完成",
		"source_asset", req.AssetID,
		"new_asset", newAsset.ID,
		"duration", duration,
		"file_size", fileSize,
	)

	return &MuteVideoResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Message:  fmt.Sprintf("静音完成，已替换为静默音轨，时长%s", duration),
	}, nil
}

// ==================== 音轨分离（v0.42.4新增） ====================

// ExtractAudioRequest 音轨分离请求
type ExtractAudioRequest struct {
	CoursewareID string // 课件ID
	AssetID      string // 原视频资产ID
	UserID       string // 操作者ID
}

// ExtractAudioResponse 音轨分离响应
type ExtractAudioResponse struct {
	AssetID  string `json:"asset_id"`  // 新生成的音频资产ID
	URL      string `json:"url"`       // 音频文件URL
	Duration string `json:"duration"`  // 音频时长
	Format   string `json:"format"`    // 输出格式（mp3）
	FileSize int64  `json:"file_size"` // 文件大小（字节）
	Message  string `json:"message"`   // 提示信息
}

// ExtractAudio 从视频中分离音频轨道，输出MP3文件
//
// FFmpeg命令: ffmpeg -y -i input.mp4 -vn -acodec libmp3lame -q:a 2 output.mp3
//   -vn: 去除视频流
//   -acodec libmp3lame: 使用LAME编码器输出MP3
//   -q:a 2: 高质量VBR（约190kbps）
func (s *VideoEditService) ExtractAudio(ctx context.Context, req *ExtractAudioRequest) (*ExtractAudioResponse, error) {
	// 1. 参数校验
	if req.AssetID == "" {
		return nil, fmt.Errorf("asset_id不能为空")
	}

	// 2. 课件权限校验
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// 3. 获取原视频文件路径
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

	sourcePath := resolveAssetPath(asset)
	if sourcePath == "" {
		return nil, fmt.Errorf("原视频文件不存在")
	}

	// 4. 先检查视频是否含有音频流（用ffprobe探测）
	probeCmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-select_streams", "a",       // 只选音频流
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		sourcePath,
	)
	probeOut, _ := probeCmd.Output()
	if len(strings.TrimSpace(string(probeOut))) == 0 {
		return nil, fmt.Errorf("该视频没有音频轨道，无法提取")
	}

	// 5. 输出文件路径（音频存在videos同目录下）
	outputDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "videos")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}
	outputName := fmt.Sprintf("%d_audio.mp3", time.Now().UnixMilli())
	outputPath := filepath.Join(outputDir, outputName)

	// 6. 执行FFmpeg提取音轨
	videoEditLog.Info("开始音轨分离",
		"courseware_id", req.CoursewareID,
		"source_asset", req.AssetID,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",                   // 覆盖已有文件
		"-i", sourcePath,
		"-vn",                  // 去除视频流
		"-acodec", "libmp3lame", // MP3编码
		"-q:a", "2",            // 高质量VBR（约190kbps）
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		videoEditLog.Error("FFmpeg音轨分离失败", "error", err, "output", string(output))
		return nil, fmt.Errorf("音轨分离失败: %w", err)
	}

	// 7. 获取时长和文件大小
	duration := getVideoDuration(outputPath)
	var fileSize int64
	if info, statErr := os.Stat(outputPath); statErr == nil {
		fileSize = info.Size()
	}

	// 8. 写入数据库（asset_type = audio）
	localURL := CWAssetURLPrefix + filepath.Join(req.CoursewareID, "videos", outputName)
	newAsset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PlaceholderID:    "",
		AssetType:        "audio", // 音频类型
		GenerationPrompt: "从视频提取音轨",
		OssURL:           localURL,
		FileSize:         fileSize,
		MimeType:         "audio/mpeg",
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, newAsset); err != nil {
		return nil, fmt.Errorf("记录音频资产失败: %w", err)
	}

	videoEditLog.Info("音轨分离完成",
		"source_asset", req.AssetID,
		"new_asset", newAsset.ID,
		"duration", duration,
		"file_size", fileSize,
	)

	return &ExtractAudioResponse{
		AssetID:  newAsset.ID,
		URL:      localURL,
		Duration: duration,
		Format:   "mp3",
		FileSize: fileSize,
		Message:  fmt.Sprintf("音轨分离完成，格式MP3，时长%s", duration),
	}, nil
}

// ==================== 辅助函数 ====================

// resolveAssetPath 从资产记录解析本地文件完整路径
// 返回空字符串表示文件不存在
func resolveAssetPath(asset *models.CoursewareAsset) string {
	if asset.OssURL == "" || !strings.HasPrefix(asset.OssURL, CWAssetURLPrefix) {
		return ""
	}
	relativePath := asset.OssURL[len(CWAssetURLPrefix):]
	fullPath := filepath.Join(CWAssetUploadDir, relativePath)
	if !fileExists(fullPath) {
		return ""
	}
	return fullPath
}

// getVideoDuration 使用ffprobe获取视频/音频时长
func getVideoDuration(filePath string) string {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath,
	)
	output, err := cmd.Output()
	if err != nil {
		return "未知"
	}
	// 简单解析duration字段
	s := string(output)
	idx := strings.Index(s, `"duration"`)
	if idx < 0 {
		return "未知"
	}
	rest := s[idx:]
	q1 := strings.Index(rest, `": "`)
	if q1 < 0 {
		return "未知"
	}
	rest = rest[q1+4:]
	q2 := strings.Index(rest, `"`)
	if q2 < 0 {
		return "未知"
	}
	return rest[:q2] + "s"
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
