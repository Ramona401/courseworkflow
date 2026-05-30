package services

// courseware_asset_video_upload.go — 课件视频手动上传服务（v0.42.5 新增,v0.42.6+ P2 磁盘预检查）
//
// 设计背景:
//   - AI 视频生成(courseware_asset_video.go)走豆包 Seedance 异步任务+轮询
//   - 手动上传视频(本文件)走 multipart/form-data 直接保存
//   - 两者共用 /uploads/courseware-assets/{cwID}/videos/ 存储路径,
//     在前端素材库中无缝混合展示,删除走同一份 DeleteAsset 逻辑
//
// 与图片上传(UploadAsset in courseware_asset_service.go)的差异:
//   - MIME 白名单不同: video/mp4|webm|quicktime|x-msvideo
//   - 大小上限不同: 图片 5MB,视频 50MB
//   - 存储路径不同: 图片 p{num}/,视频 videos/(与 AI 生成视频一致)
//   - 不支持 placeholder_id(视频不放占位符,直接加入素材库)
//
// v0.42.6+ P2 优化:
//   - P2.5 磁盘空间预检查: 写盘前用 syscall.Statfs 查上传目录所在分区可用空间
//     要求至少为文件大小 × 2(双倍冗余防并发上传)
//     这样能在写盘前快速失败,避免 io.Copy 中途磁盘满留下半截损坏文件
//
// 路由: POST /api/v1/coursewares/{id}/pages/{num}/upload-video
// Nginx 已配置 client_max_body_size 55M 支持 50MB 视频上传

import (
        "context"
        "encoding/json"
        "fmt"
        "io"
        "mime/multipart"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
        "syscall"
        "time"

        "tedna/internal/logger"
        "tedna/internal/models"
        "tedna/internal/repository"
)

// cwVideoUploadLog 视频上传服务专用 logger (v0.42.5+)
var cwVideoUploadLog = logger.WithModule("cw_video_upload")

// ==================== 视频上传相关常量 ====================

const (
        // CWVideoMaxSize 单个上传视频最大 50MB
        // 与 Nginx client_max_body_size 55M 保持兼容(多 5MB 用于 multipart 边界开销)
        CWVideoMaxSize = 50 * 1024 * 1024

        // cwVideoDiskSafetyFactor 磁盘可用空间相对于文件大小的冗余倍数 (P2.5)
        //
        // 设置为 2 的原因:
        //   1. 并发上传保护: 同时多个 50MB 视频写入时单文件检查可能错过累计占用
        //   2. ffprobe/FFmpeg 操作冗余: 后续可能产生临时文件(如分离音轨/拼接)
        //   3. 系统层文件系统元数据/inode 开销
        //   4. 实践经验: 1.5 倍勉强够,2 倍给运维留余地告警
        //
        // 若 50MB 视频上传,需要至少 100MB 可用空间才放行
        cwVideoDiskSafetyFactor = 2
)

// cwVideoAllowedMimeTypes 允许的视频 MIME 类型白名单
// 浏览器内置 <video> 标签可直接播放,FFmpeg 也都能处理
var cwVideoAllowedMimeTypes = map[string]bool{
        "video/mp4":       true, // .mp4 - 最常见
        "video/webm":      true, // .webm - 现代浏览器原生支持
        "video/quicktime": true, // .mov - 苹果设备录制默认格式
        "video/x-msvideo": true, // .avi - 老格式但仍在使用
}

// cwVideoMimeToExt 视频 MIME → 文件扩展名映射
// 用于在客户端未提供 Content-Type 时按扩展名兜底识别
var cwVideoMimeToExt = map[string]string{
        "video/mp4":       ".mp4",
        "video/webm":      ".webm",
        "video/quicktime": ".mov",
        "video/x-msvideo": ".avi",
}

// ==================== 请求/响应结构 ====================

// UploadVideoAssetRequest 视频上传请求参数
// 注意: 不含 PlaceholderID 字段(视频不替换占位符,直接进素材库)
type UploadVideoAssetRequest struct {
        CoursewareID string // 课件 ID
        PageNumber   int    // 关联页码(用于 page_id 外键写入,便于按页统计)
        UserID       string // 操作者 ID(用于权限校验)
}

// UploadVideoAssetResponse 视频上传响应
// 字段与图片 UploadAssetResponse 保持一致,前端 TypeScript 类型可共用
type UploadVideoAssetResponse struct {
        AssetID  string `json:"asset_id"`  // 数据库资产记录 ID
        URL      string `json:"url"`       // 公开访问 URL(前端 <video src> 直接用)
        FileName string `json:"file_name"` // 服务器端实际存储的文件名
        FileSize int64  `json:"file_size"` // 文件大小(字节)
        MimeType string `json:"mime_type"` // MIME 类型
}

// ==================== v0.42.6+ P2.5: 磁盘空间预检查 ====================

// diskSpaceInfo 磁盘空间信息(供日志输出和错误消息使用)
type diskSpaceInfo struct {
        AvailableBytes uint64 // 当前用户可用字节数(非 root 普通用户视角)
        TotalBytes     uint64 // 分区总字节数(用于计算使用率)
        UsedPercent    int    // 使用率百分比(整数,便于日志展示)
}

// checkDiskSpace 检查指定目录所在分区是否有足够空间容纳即将写入的文件
//
// 参数:
//   - dir: 即将写入文件的目录路径(本服务总是传 CWAssetUploadDir 上传根目录)
//   - requiredBytes: 单次需求字节数(本服务即将写入的视频文件大小)
//
// 返回值:
//   - *diskSpaceInfo: 当前磁盘统计(用于日志和错误消息,即使无错误也返回供成功路径记日志)
//   - error: 空间不足时返回明确错误,statfs 调用失败时返回包装错误
//
// 实现原理: syscall.Statfs(Linux only) 读取分区元数据
//   - statfs.Bavail = 普通用户可用块数(已扣除 root 保留块)
//   - statfs.Bsize = 文件系统块大小(通常 4096)
//   - 可用字节 = Bavail × Bsize
//
// 该函数失败时(如 statfs 系统调用本身失败)只记 WARN 不阻塞,
// 因为大部分情况是测试环境路径问题,直接拦截会让上传整体瘫痪
func checkDiskSpace(dir string, requiredBytes int64) (*diskSpaceInfo, error) {
        var stat syscall.Statfs_t
        if err := syscall.Statfs(dir, &stat); err != nil {
                // statfs 失败常见原因: 目录不存在/权限不足/非 Linux 平台
                // 这种情况返回 nil info + error,调用方决定是否降级放行
                return nil, fmt.Errorf("无法获取磁盘信息: %w", err)
        }

        // 计算可用/总空间(注意 uint64 防止 int 溢出,50MB×2 在 32 位机也安全)
        available := stat.Bavail * uint64(stat.Bsize)
        total := stat.Blocks * uint64(stat.Bsize)
        var usedPct int
        if total > 0 {
                usedPct = int((total - available) * 100 / total)
        }

        info := &diskSpaceInfo{
                AvailableBytes: available,
                TotalBytes:     total,
                UsedPercent:    usedPct,
        }

        // 需求 = 文件大小 × 冗余系数
        needed := uint64(requiredBytes) * uint64(cwVideoDiskSafetyFactor)
        if available < needed {
                return info, fmt.Errorf(
                        "磁盘空间不足: 当前可用 %.1f MB,本次上传需要至少 %.1f MB(文件 %.1f MB × %d 倍冗余),磁盘使用率 %d%%",
                        float64(available)/(1024*1024),
                        float64(needed)/(1024*1024),
                        float64(requiredBytes)/(1024*1024),
                        cwVideoDiskSafetyFactor,
                        usedPct,
                )
        }

        return info, nil
}

// ==================== 上传服务主方法 ====================

// UploadVideoAsset 手动上传视频到本地磁盘并记录到数据库
//
// 处理流程(v0.42.6+ 加入 P2.5 磁盘预检查):
//   1. 校验课件存在性 + 用户所有权
//   2. 校验文件大小(50MB 上限)
//   3. 检测 MIME 类型(优先 header,回退按扩展名)
//   4. 校验页面存在性
//   5. 生成安全文件名(rune 截断防止中文 UTF-8 多字节断裂)
//   6. 【P2.5 新增】磁盘空间预检查 — 写盘前快速失败,避免 io.Copy 中途半残
//   7. 创建 videos/ 目录(与 AI 视频生成共用)
//   8. 保存文件到磁盘
//   9. 写入 courseware_assets 表(asset_type=video, status=uploaded)
//
// 错误时已写盘的文件会被回滚(os.Remove),保证数据库与磁盘一致
func (s *CoursewareAssetService) UploadVideoAsset(
        ctx context.Context,
        req *UploadVideoAssetRequest,
        file multipart.File,
        header *multipart.FileHeader,
) (*UploadVideoAssetResponse, error) {
        // ========== 1. 校验课件存在性和用户所有权 ==========
        cw, err := repository.GetCoursewareByID(ctx, req.CoursewareID)
        if err != nil {
                return nil, fmt.Errorf("课件不存在: %w", err)
        }
        if cw.UserID != req.UserID {
                return nil, fmt.Errorf("无权操作此课件")
        }

        // ========== 2. 校验文件大小(50MB) ==========
        if header.Size > CWVideoMaxSize {
                return nil, fmt.Errorf("视频文件过大,最大支持50MB(当前%.1fMB)",
                        float64(header.Size)/(1024*1024))
        }
        if header.Size <= 0 {
                return nil, fmt.Errorf("视频文件为空")
        }

        // ========== 3. 检测 MIME 类型 ==========
        mimeType := header.Header.Get("Content-Type")
        if mimeType == "" {
                // 浏览器未提供 Content-Type 时,按扩展名兜底
                ext := strings.ToLower(filepath.Ext(header.Filename))
                switch ext {
                case ".mp4":
                        mimeType = "video/mp4"
                case ".webm":
                        mimeType = "video/webm"
                case ".mov":
                        mimeType = "video/quicktime"
                case ".avi":
                        mimeType = "video/x-msvideo"
                default:
                        return nil, fmt.Errorf("不支持的视频格式,支持 MP4/WebM/MOV/AVI")
                }
        }
        if !cwVideoAllowedMimeTypes[mimeType] {
                return nil, fmt.Errorf("不支持的视频格式 %s,仅支持 MP4/WebM/MOV/AVI", mimeType)
        }

        // ========== 4. 校验页面存在性 ==========
        page, err := repository.GetCoursewarePageByNumber(ctx, req.CoursewareID, req.PageNumber)
        if err != nil {
                return nil, fmt.Errorf("页面不存在: 课件=%s 页码=%d", req.CoursewareID, req.PageNumber)
        }

        // ========== 5. 生成安全文件名 ==========
        ext := cwVideoMimeToExt[mimeType]
        if ext == "" {
                ext = ".mp4"
        }
        baseName := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
        // 复用图片服务的安全名正则 cwAssetSafeNameRe(声明在 courseware_asset_service.go)
        baseName = cwAssetSafeNameRe.ReplaceAllString(baseName, "_")
        // 折叠连续下划线
        for strings.Contains(baseName, "__") {
                baseName = strings.ReplaceAll(baseName, "__", "_")
        }
        baseName = strings.Trim(baseName, "_")
        // rune 截断防止 UTF-8 多字节序列被切断(中文一字三字节)
        baseRunes := []rune(baseName)
        if len(baseRunes) > 40 {
                baseName = string(baseRunes[:40])
        }
        if baseName == "" {
                baseName = "video"
        }
        storedName := fmt.Sprintf("%d_upload_%s%s", time.Now().UnixMilli(), baseName, ext)

        // ========== 6. 【P2.5】磁盘空间预检查 ==========
        // 在创建目录、写文件前就快速失败,把"磁盘满"问题拦截在最早一步,
        // 避免 io.Copy 中途失败导致半截文件留在磁盘+数据库记录不一致
        //
        // 检查目录: 用上传根目录 CWAssetUploadDir(必然存在,且与实际写入是同一分区)
        // 失败处理: 检查本身失败(statfs 调用错)只 WARN 不阻塞,允许继续写盘
        //          空间不足才返回错误拒绝上传
        diskInfo, diskErr := checkDiskSpace(CWAssetUploadDir, header.Size)
        if diskErr != nil && diskInfo != nil {
                // 有 diskInfo = statfs 成功了但空间不足 → 拒绝上传
                cwVideoUploadLog.Warn("磁盘空间不足,拒绝视频上传",
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
                // 常见原因: 测试环境路径问题、跨平台兼容,不应阻塞业务
                cwVideoUploadLog.Warn("磁盘空间检查失败,降级放行",
                        "courseware_id", req.CoursewareID,
                        "dir", CWAssetUploadDir,
                        "error", diskErr,
                )
        }

        // ========== 7. 创建 videos/ 子目录 ==========
        // 与 AI 视频生成 (courseware_asset_video.go downloadAndSaveVideo) 共用同一路径
        assetDir := filepath.Join(CWAssetUploadDir, req.CoursewareID, "videos")
        if err := os.MkdirAll(assetDir, 0755); err != nil {
                return nil, fmt.Errorf("创建视频目录失败: %w", err)
        }

        // ========== 8. 保存视频文件到磁盘 ==========
        fullPath := filepath.Join(assetDir, storedName)
        dst, err := os.Create(fullPath)
        if err != nil {
                return nil, fmt.Errorf("创建视频文件失败: %w", err)
        }
        defer dst.Close()

        // io.Copy 流式写入,避免一次性把 50MB 视频加载到内存
        written, err := io.Copy(dst, file)
        if err != nil {
                _ = os.Remove(fullPath) // 写入失败回滚已写部分
                return nil, fmt.Errorf("保存视频文件失败: %w", err)
        }

        // ========== 9. 构建 URL 并写入数据库 ==========
        relativePath := filepath.Join(req.CoursewareID, "videos", storedName)
        assetURL := CWAssetURLPrefix + relativePath

        asset := &models.CoursewareAsset{
                CoursewareID:     req.CoursewareID,
                PageID:           &page.ID, // 关联页面便于按页统计
                PlaceholderID:    "",       // 视频不替换占位符
                AssetType:        models.CWAssetTypeVideo,
                GenerationPrompt: "", // 手动上传无生成提示词
                OssURL:           assetURL,
                FileSize:         written,
                MimeType:         mimeType,
                Status:           models.CWAssetStatusUploaded,
        }
        if err := repository.CreateCWAsset(ctx, asset); err != nil {
                _ = os.Remove(fullPath) // 数据库写入失败回滚磁盘文件
                return nil, fmt.Errorf("记录视频资产失败: %w", err)
        }

        // 结构化日志(与图片上传保持相同字段,便于统一查询)
        // P2.5: 日志中加入磁盘使用率,便于运维监控趋势(diskInfo 可能为 nil)
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
        cwAssetLog.Info("课件视频上传成功", logFields...)

        // v0.42.6+ P0.2: 异步提取视频元数据(本次仅日志验证,下次接通 metadata 列写入)
        saveVideoMetadataAsync(asset.ID, fullPath)

        return &UploadVideoAssetResponse{
                AssetID:  asset.ID,
                URL:      assetURL,
                FileName: storedName,
                FileSize: written,
                MimeType: mimeType,
        }, nil
}

// ==================== v0.42.6+ P0.2: 视频元数据提取 ====================

// videoMetadata 视频元数据结构(JSONB 序列化)
type videoMetadata struct {
        Duration string `json:"duration,omitempty"` // 秒(字符串,如 "12.345")
        Width    int    `json:"width,omitempty"`    // 宽度像素
        Height   int    `json:"height,omitempty"`   // 高度像素
        Codec    string `json:"codec,omitempty"`    // 视频编码(h264/vp9等)
        FPS      string `json:"fps,omitempty"`      // 帧率(分数,如 "30000/1001")
        BitRate  string `json:"bit_rate,omitempty"` // 比特率(bps,字符串)
}

// extractVideoMetadata 用 ffprobe 提取视频完整元数据
// 参考 video_edit_service.go 的 ffprobe 用法,扩展支持多字段
func extractVideoMetadata(filePath string) (*videoMetadata, error) {
        cmd := exec.Command("ffprobe",
                "-v", "quiet",
                "-print_format", "json",
                "-show_format",
                "-show_streams",
                filePath,
        )
        output, err := cmd.Output()
        if err != nil {
                return nil, fmt.Errorf("ffprobe 执行失败: %w", err)
        }

        // ffprobe JSON 输出结构(只声明用到的字段)
        var probe struct {
                Format struct {
                        Duration string `json:"duration"`
                        BitRate  string `json:"bit_rate"`
                } `json:"format"`
                Streams []struct {
                        CodecType string `json:"codec_type"`
                        CodecName string `json:"codec_name"`
                        Width     int    `json:"width"`
                        Height    int    `json:"height"`
                        FrameRate string `json:"r_frame_rate"`
                } `json:"streams"`
        }

        if err := json.Unmarshal(output, &probe); err != nil {
                return nil, fmt.Errorf("ffprobe 输出解析失败: %w", err)
        }

        meta := &videoMetadata{
                Duration: probe.Format.Duration,
                BitRate:  probe.Format.BitRate,
        }
        // 找到视频流(忽略音频/字幕等其他流)
        for _, s := range probe.Streams {
                if s.CodecType == "video" {
                        meta.Width = s.Width
                        meta.Height = s.Height
                        meta.Codec = s.CodecName
                        meta.FPS = s.FrameRate
                        break
                }
        }
        return meta, nil
}

// saveVideoMetadataAsync 异步提取元数据并记录日志
//
// v0.42.6+ P0.2 阶段1实现:
//   - 仅 ffprobe 提取 + 日志记录,验证管道工作正常
//   - metadata 列已通过 ALTER TABLE 加入数据库
//   - 下次对话接通 repository.UpdateCWAssetMetadata 写入 metadata 列
//
// 失败仅记日志,不阻塞主流程(用户上传已经成功返回)
// 用 goroutine 异步处理避免阻塞 HTTP 响应
func saveVideoMetadataAsync(assetID, filePath string) {
        go func() {
                meta, err := extractVideoMetadata(filePath)
                if err != nil {
                        cwVideoUploadLog.Warn("视频元数据提取失败,跳过",
                                "asset_id", assetID, "file", filePath, "error", err)
                        return
                }

                metaJSON, _ := json.Marshal(meta)

                // P0.2 阶段1: 仅日志输出,下次接通数据库写入(需 repository 加 UpdateCWAssetMetadata 函数)
                cwVideoUploadLog.Info("✓ 视频元数据已提取(下次接通 metadata 列写入)",
                        "asset_id", assetID,
                        "duration", meta.Duration,
                        "resolution", fmt.Sprintf("%dx%d", meta.Width, meta.Height),
                        "codec", meta.Codec,
                        "fps", meta.FPS,
                        "bit_rate", meta.BitRate,
                        "metadata_json", string(metaJSON),
                )
        }()
}
