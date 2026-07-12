package services

// courseware_asset_audio_upload.go — 课件音频手动上传服务
//
// 设计背景:
//   - 音频上传不走AI生成，纯手动上传到本地磁盘
//   - 上传后可走既有 UploadToOSS 端点上传到阿里云OSS获取公网链接
//   - 老师复制公网链接在微调里把音频嵌入课件HTML（如 <audio> 标签）
//
// 与视频上传(courseware_asset_video_upload.go)的差异:
//   - MIME 白名单不同: audio/mpeg(mp3)|audio/wav|audio/ogg|audio/aac|audio/flac|audio/x-m4a
//   - 大小上限不同: 音频 20MB（比视频50MB小，比图片5MB大）
//   - 存储路径不同: 音频 audios/（区别于图片 p{num}/ 和视频 videos/）
//   - 无 ffprobe 元数据提取（音频不需要宽高等视频元数据）
//   - 无 placeholder_id（音频不替换占位符，直接加入素材库）
//
// 路由: POST /api/v1/coursewares/{id}/pages/{num}/upload-audio

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwAudioUploadLog 音频上传服务专用 logger
var cwAudioUploadLog = logger.WithModule("cw_audio_upload")

// ==================== 音频上传相关常量 ====================

const (
	// CWAudioMaxSize 单个上传音频最大 20MB
	CWAudioMaxSize = 20 * 1024 * 1024
)

// cwAudioAllowedMimeTypes 允许的音频 MIME 类型白名单
// 浏览器 <audio> 标签可直接播放的常见格式
var cwAudioAllowedMimeTypes = map[string]bool{
	"audio/mpeg":    true, // .mp3 — 最常见
	"audio/wav":     true, // .wav — 无损格式
	"audio/ogg":     true, // .ogg — 开源格式
	"audio/aac":     true, // .aac — 苹果系常用
	"audio/flac":    true, // .flac — 无损格式
	"audio/x-m4a":   true, // .m4a — 苹果音频容器
	"audio/mp4":     true, // .m4a — 部分浏览器返回此MIME
}

// cwAudioMimeToExt 音频 MIME → 文件扩展名映射
// 用于在客户端未提供 Content-Type 时按扩展名兜底识别
var cwAudioMimeToExt = map[string]string{
	"audio/mpeg":  ".mp3",
	"audio/wav":   ".wav",
	"audio/ogg":   ".ogg",
	"audio/aac":   ".aac",
	"audio/flac":  ".flac",
	"audio/x-m4a": ".m4a",
	"audio/mp4":   ".m4a",
}

// ==================== 请求/响应结构 ====================

// UploadAudioAssetRequest 音频上传请求参数
type UploadAudioAssetRequest struct {
	CoursewareID string // 课件 ID
	PageNumber   int    // 关联页码（用于 page_id 外键写入，便于按页统计）
	UserID       string // 操作者 ID（用于权限校验）
}

// UploadAudioAssetResponse 音频上传响应
// 字段与视频/图片上传响应保持一致，前端 TypeScript 类型可共用
type UploadAudioAssetResponse struct {
	AssetID  string `json:"asset_id"`  // 数据库资产记录 ID
	URL      string `json:"url"`       // 公开访问 URL（前端 <audio src> 直接用）
	FileName string `json:"file_name"` // 服务器端实际存储的文件名
	FileSize int64  `json:"file_size"` // 文件大小（字节）
	MimeType string `json:"mime_type"` // MIME 类型
}

// ==================== 上传服务主方法 ====================

// UploadAudioAsset 手动上传音频到本地磁盘并记录到数据库
//
// 处理流程:
//  1. 校验课件存在性 + 用户所有权
//  2. 校验文件大小（20MB 上限）
//  3. 检测 MIME 类型（优先 header，回退按扩展名）
//  4. 校验页面存在性
//  5. 生成安全文件名（rune 截断防止中文 UTF-8 多字节断裂）
//  6. 磁盘空间预检查（复用视频上传的 checkDiskSpace）
//  7. 创建 audios/ 子目录
//  8. 保存文件到磁盘
//  9. 写入 courseware_assets 表（asset_type=audio, status=uploaded）
//
// 错误时已写盘的文件会被回滚（os.Remove），保证数据库与磁盘一致
func (s *CoursewareAssetService) UploadAudioAsset(
	ctx context.Context,
	req *UploadAudioAssetRequest,
	file multipart.File,
	header *multipart.FileHeader,
) (*UploadAudioAssetResponse, error) {
	// ========== 1. 校验课件存在性和用户所有权 ==========
	cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
	if err != nil {
		return nil, fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != req.UserID {
		return nil, fmt.Errorf("无权操作此课件")
	}

	// ========== 2. 校验文件大小（20MB） ==========
	if header.Size > CWAudioMaxSize {
		return nil, fmt.Errorf("音频文件过大，最大支持20MB（当前%.1fMB）",
			float64(header.Size)/(1024*1024))
	}
	if header.Size <= 0 {
		return nil, fmt.Errorf("音频文件为空")
	}

	// ========== 3. 检测 MIME 类型 ==========
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		// 浏览器未提供 Content-Type 时，按扩展名兜底
		ext := strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".mp3":
			mimeType = "audio/mpeg"
		case ".wav":
			mimeType = "audio/wav"
		case ".ogg":
			mimeType = "audio/ogg"
		case ".aac":
			mimeType = "audio/aac"
		case ".flac":
			mimeType = "audio/flac"
		case ".m4a":
			mimeType = "audio/x-m4a"
		default:
			return nil, fmt.Errorf("不支持的音频格式，支持 MP3/WAV/OGG/AAC/FLAC/M4A")
		}
	}
	if !cwAudioAllowedMimeTypes[mimeType] {
		return nil, fmt.Errorf("不支持的音频格式 %s，仅支持 MP3/WAV/OGG/AAC/FLAC/M4A", mimeType)
	}

	// ========== 4. 校验页面存在性 ==========
	page, err := repository.GetCoursewarePageByNumber(ctx, req.CoursewareID, req.PageNumber)
	if err != nil {
		return nil, fmt.Errorf("页面不存在: 课件=%s 页码=%d", req.CoursewareID, req.PageNumber)
	}

	// ========== 5. 生成安全文件名 ==========
	ext := cwAudioMimeToExt[mimeType]
	if ext == "" {
		ext = ".mp3"
	}
	baseName := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	// 复用图片服务的安全名正则 cwAssetSafeNameRe（声明在 courseware_asset_service.go）
	baseName = cwAssetSafeNameRe.ReplaceAllString(baseName, "_")
	// 折叠连续下划线
	for strings.Contains(baseName, "__") {
		baseName = strings.ReplaceAll(baseName, "__", "_")
	}
	baseName = strings.Trim(baseName, "_")
	// rune 截断防止 UTF-8 多字节序列被切断（中文一字三字节）
	baseRunes := []rune(baseName)
	if len(baseRunes) > 40 {
		baseName = string(baseRunes[:40])
	}
	if baseName == "" {
		baseName = "audio"
	}
	storedName := fmt.Sprintf("%d_upload_%s%s", time.Now().UnixMilli(), baseName, ext)

	// ========== 6. 磁盘空间预检查（复用视频上传的 checkDiskSpace） ==========
	diskInfo, diskErr := checkDiskSpace(CWAssetUploadDir, header.Size)
	if diskErr != nil && diskInfo != nil {
		// 有 diskInfo = statfs 成功了但空间不足 → 拒绝上传
		cwAudioUploadLog.Warn("磁盘空间不足，拒绝音频上传",
			"courseware_id", req.CoursewareID,
			"file", header.Filename,
			"file_size", header.Size,
			"available_mb", diskInfo.AvailableBytes/(1024*1024),
			"used_percent", diskInfo.UsedPercent,
			"error", diskErr,
		)
		return nil, diskErr
	}
	if diskErr != nil && diskInfo == nil {
		// 没有 diskInfo = statfs 系统调用本身失败 → WARN 但放行
		cwAudioUploadLog.Warn("磁盘空间检查失败，降级放行",
			"courseware_id", req.CoursewareID,
			"dir", CWAssetUploadDir,
			"error", diskErr,
		)
	}

	// ========== 7. 创建 audios/ 子目录 ==========
	assetDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "audios")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return nil, fmt.Errorf("创建音频目录失败: %w", err)
	}

	// ========== 8. 保存音频文件到磁盘 ==========
	fullPath := filepath.Join(assetDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建音频文件失败: %w", err)
	}
	defer dst.Close()

	// io.Copy 流式写入，避免一次性把音频加载到内存
	written, err := io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(fullPath) // 写入失败回滚已写部分
		return nil, fmt.Errorf("保存音频文件失败: %w", err)
	}

	// ========== 9. 构建 URL 并写入数据库 ==========
	relativePath := filepath.Join(req.CoursewareID, "audios", storedName)
	assetURL := CWAssetURLPrefix + relativePath

	asset := &models.CoursewareAsset{
		CoursewareID:     req.CoursewareID,
		PageID:           &page.ID,  // 关联页面便于按页统计
		PlaceholderID:    "",        // 音频不替换占位符
		AssetType:        models.CWAssetTypeAudio,
		GenerationPrompt: "",        // 手动上传无生成提示词
		OssURL:           assetURL,
		FileSize:         written,
		MimeType:         mimeType,
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, asset); err != nil {
		_ = os.Remove(fullPath) // 数据库写入失败回滚磁盘文件
		return nil, fmt.Errorf("记录音频资产失败: %w", err)
	}

	// 结构化日志（与图片/视频上传保持相同字段，便于统一查询）
	logFields := []interface{}{
		"courseware_id", req.CoursewareID,
		"page_number", req.PageNumber,
		"asset_id", asset.ID,
		"file", header.Filename,
		"size", written,
		"mime", mimeType,
		"url", assetURL,
	}
	if diskInfo != nil {
		logFields = append(logFields,
			"disk_used_pct", diskInfo.UsedPercent,
			"disk_available_mb", diskInfo.AvailableBytes/(1024*1024),
		)
	}
	cwAssetLog.Info("课件音频上传成功", logFields...)

	return &UploadAudioAssetResponse{
		AssetID:  asset.ID,
		URL:      assetURL,
		FileName: storedName,
		FileSize: written,
		MimeType: mimeType,
	}, nil
}
