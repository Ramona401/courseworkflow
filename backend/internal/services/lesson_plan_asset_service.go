package services

// lesson_plan_asset_service.go — 教案附属资产业务逻辑层
//
// v123 新增:教案图片上传/列表/详情/更新/删除
//
// 设计说明:
//   - 文件物理存储路径:/www/wwwroot/tedna/uploads/lesson-plans/{plan_id}/{ts}_{name}
//   - URL 暴露路径:/uploads/lesson-plans/{plan_id}/{ts}_{name}(由 Nginx alias 映射)
//   - 上传时校验 MIME + 扩展名 + 大小,通过后保存物理文件并写数据库
//   - 删除时先删数据库行(成功)再删物理文件(失败仅记日志,不阻塞)
//   - 权限控制:只有教案作者本人可上传/删除,任何登录用户可查看(图片是公开 URL)

import (
        "context"
        "errors"
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

// ==================== 错误常量 ====================

var (
        ErrAssetNotFound        = errors.New("教案资产不存在")
        ErrAssetUnauthorized    = errors.New("无权操作此资产")
        ErrAssetFileInvalid     = errors.New("文件格式无效,仅支持 JPG/PNG/WEBP/GIF 图片")
        ErrAssetFileTooLarge    = errors.New("文件过大,最大支持 5MB")
        ErrAssetPlanNotFound    = errors.New("教案不存在")
        ErrAssetNotPlanAuthor   = errors.New("仅教案作者可上传/删除资产")
)

// ==================== 常量 ====================

const (
        // MaxAssetFileSize 单图最大 5MB(教案配图通常远小于 10MB,限严点防滥用)
        MaxAssetFileSize = 5 * 1024 * 1024

        // AssetUploadDir 资产存储根目录(物理路径)
        AssetUploadDir = "/www/wwwroot/tedna/uploads/lesson-plans"

        // AssetURLPrefix 资产 URL 前缀(由 Nginx alias 映射到 AssetUploadDir)
        AssetURLPrefix = "/uploads/lesson-plans/"
)

// 允许的 MIME 类型(教案配图比课本图片多一个 GIF 格式,因为美术老师可能需要 GIF 演示)
var allowedAssetMimeTypes = map[string]bool{
        "image/jpeg": true,
        "image/jpg":  true,
        "image/png":  true,
        "image/webp": true,
        "image/gif":  true,
}

// MIME 类型 → 扩展名映射
var assetMimeToExt = map[string]string{
        "image/jpeg": ".jpg",
        "image/jpg":  ".jpg",
        "image/png":  ".png",
        "image/webp": ".webp",
        "image/gif":  ".gif",
}

// ==================== 服务结构体 ====================

// LessonPlanAssetService 教案资产服务
type LessonPlanAssetService struct{}

var assetLog = logger.WithModule("lesson_plan_asset")

// NewLessonPlanAssetService 创建资产服务实例
func NewLessonPlanAssetService() *LessonPlanAssetService {
        return &LessonPlanAssetService{}
}

// ==================== 上传 ====================

// UploadAsset 上传教案图片
// 1. 校验教案存在 + 调用者是教案作者
// 2. 校验文件大小、MIME 类型
// 3. 生成安全文件名 → 保存物理文件 → 写数据库 → 返回响应
func (s *LessonPlanAssetService) UploadAsset(
        ctx context.Context,
        planID string,
        file multipart.File,
        header *multipart.FileHeader,
        altText string,
        callerID string,
) (*models.AssetUploadResponse, error) {
        // ---- 1. 校验教案存在 + 权限 ----
        plan, err := repository.GetLessonPlanByID(ctx, planID)
        if err != nil {
                if errors.Is(err, repository.ErrLessonPlanNotFound) {
                        return nil, ErrAssetPlanNotFound
                }
                return nil, err
        }
        if plan.AuthorID != callerID {
                return nil, ErrAssetNotPlanAuthor
        }

        // ---- 2. 校验文件 ----
        if header.Size > MaxAssetFileSize {
                return nil, ErrAssetFileTooLarge
        }
        mimeType := header.Header.Get("Content-Type")
        if mimeType == "" {
                // 从扩展名推断
                ext := strings.ToLower(filepath.Ext(header.Filename))
                switch ext {
                case ".jpg", ".jpeg":
                        mimeType = "image/jpeg"
                case ".png":
                        mimeType = "image/png"
                case ".webp":
                        mimeType = "image/webp"
                case ".gif":
                        mimeType = "image/gif"
                default:
                        return nil, ErrAssetFileInvalid
                }
        }
        if !allowedAssetMimeTypes[mimeType] {
                return nil, ErrAssetFileInvalid
        }

        // ---- 3. 生成安全文件名 ----
        ext := assetMimeToExt[mimeType]
        if ext == "" {
                ext = ".jpg"
        }
        // 文件名安全化:去空格+剥离 URL 特殊字符+只保留基础名+加时间戳前缀防冲突
        // v123 增强(2026-04-26):图片 URL 会进入 Markdown ![alt](url) 语法,
        // 必须剥离 URL 中有歧义的字符,否则前端 markdown 渲染会因解析失败而崩。
        // 触发 bug 的真实文件名:"u=1854298473,327852352&fm=253&app=138&f=JPEG.jpg"
        safeOrigName := strings.ReplaceAll(header.Filename, " ", "_")
        baseName := strings.TrimSuffix(filepath.Base(safeOrigName), filepath.Ext(safeOrigName))
        // 防 path traversal
        baseName = strings.ReplaceAll(baseName, "..", "_")
        baseName = strings.ReplaceAll(baseName, "/", "_")
        baseName = strings.ReplaceAll(baseName, "\\", "_")
        // URL 安全化:剥离会破坏 Markdown URL 语法的字符
        for _, ch := range []string{"&", "=", ",", "#", "?", "%", "+", "(", ")", "[", "]", "<", ">", "\"", "'", "`", ";", ":"} {
                baseName = strings.ReplaceAll(baseName, ch, "_")
        }
        // 多个连续下划线压缩为单个
        for strings.Contains(baseName, "__") {
                baseName = strings.ReplaceAll(baseName, "__", "_")
        }
        baseName = strings.Trim(baseName, "_")
        if len(baseName) > 80 {
                baseName = baseName[:80]
        }
        if baseName == "" {
                baseName = "image"
        }
        storedName := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), baseName, ext)

        // ---- 4. 创建教案专属目录 ----
        planDir := filepath.Join(AssetUploadDir, planID)
        if err := os.MkdirAll(planDir, 0755); err != nil {
                return nil, fmt.Errorf("创建上传目录失败: %w", err)
        }

        // ---- 5. 保存物理文件 ----
        fullPath := filepath.Join(planDir, storedName)
        dst, err := os.Create(fullPath)
        if err != nil {
                return nil, fmt.Errorf("创建文件失败: %w", err)
        }
        defer dst.Close()

        written, err := io.Copy(dst, file)
        if err != nil {
                _ = os.Remove(fullPath) // 清理失败的半成品文件
                return nil, fmt.Errorf("保存文件失败: %w", err)
        }

        // 相对路径:planID/storedName(URL 拼接和数据库存储都用这个)
        relativePath := filepath.Join(planID, storedName)

        // ---- 6. alt 文本兜底 ----
        if strings.TrimSpace(altText) == "" {
                // 没填 alt 时,用原始文件名(去扩展名)作为兜底
                altText = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
                if len(altText) > 100 {
                        altText = altText[:100]
                }
        }

        // ---- 7. 写数据库 ----
        asset := &models.LessonPlanAsset{
                LessonPlanID: planID,
                UploaderID:   callerID,
                AssetType:    models.AssetTypeImage,
                FileName:     header.Filename,
                FilePath:     relativePath,
                FileSize:     written,
                MimeType:     mimeType,
                AltText:      altText,
        }
        if err := repository.CreateLessonPlanAsset(ctx, asset); err != nil {
                _ = os.Remove(fullPath) // 数据库写失败时清理已保存的物理文件
                return nil, err
        }

        // ---- 8. 拼装响应(URL + 已拼好的 Markdown) ----
        url := AssetURLPrefix + relativePath
        markdown := fmt.Sprintf("![%s](%s)", altText, url)

        assetLog.Info("教案图片上传成功",
                "asset_id", asset.ID,
                "plan_id", planID,
                "file", header.Filename,
                "size", written,
                "uploader", callerID,
        )

        return &models.AssetUploadResponse{
                ID:       asset.ID,
                FileName: header.Filename,
                FileSize: written,
                URL:      url,
                Markdown: markdown,
        }, nil
}

// ==================== 列表 ====================

// ListAssets 按教案 ID 列出所有资产
// 需校验调用者对教案有访问权限(暂时简化:任意登录用户可查看,未来按可见性控制)
func (s *LessonPlanAssetService) ListAssets(ctx context.Context, planID string) (*models.AssetListResponse, error) {
        // 校验教案存在
        if _, err := repository.GetLessonPlanByID(ctx, planID); err != nil {
                if errors.Is(err, repository.ErrLessonPlanNotFound) {
                        return nil, ErrAssetPlanNotFound
                }
                return nil, err
        }

        items, total, err := repository.ListLessonPlanAssets(ctx, planID)
        if err != nil {
                return nil, err
        }
        if items == nil {
                items = []*models.LessonPlanAssetListItem{}
        }
        // 给每项填充可访问 URL
        for _, it := range items {
                it.URL = AssetURLPrefix + it.FilePath
        }
        return &models.AssetListResponse{Assets: items, Total: total}, nil
}

// ==================== 详情 ====================

// GetAsset 获取单条资产详情
func (s *LessonPlanAssetService) GetAsset(ctx context.Context, id string) (*models.LessonPlanAssetListItem, error) {
        a, err := repository.GetLessonPlanAssetByID(ctx, id)
        if err != nil {
                if errors.Is(err, repository.ErrLessonPlanAssetNotFound) {
                        return nil, ErrAssetNotFound
                }
                return nil, err
        }
        item := &models.LessonPlanAssetListItem{
                LessonPlanAsset: *a,
                URL:             AssetURLPrefix + a.FilePath,
        }
        return item, nil
}

// ==================== 更新 alt 文本 ====================

// UpdateAssetAltText 更新资产 alt 文本(只允许教案作者)
func (s *LessonPlanAssetService) UpdateAssetAltText(ctx context.Context, id string, altText string, callerID string) error {
        a, err := repository.GetLessonPlanAssetByID(ctx, id)
        if err != nil {
                if errors.Is(err, repository.ErrLessonPlanAssetNotFound) {
                        return ErrAssetNotFound
                }
                return err
        }
        // 校验权限:必须是教案作者
        plan, err := repository.GetLessonPlanByID(ctx, a.LessonPlanID)
        if err != nil {
                return err
        }
        if plan.AuthorID != callerID {
                return ErrAssetUnauthorized
        }
        return repository.UpdateLessonPlanAssetAltText(ctx, id, altText)
}

// ==================== 删除 ====================

// DeleteAsset 删除资产(数据库行 + 物理文件)
// 只允许教案作者操作
func (s *LessonPlanAssetService) DeleteAsset(ctx context.Context, id string, callerID string) error {
        a, err := repository.GetLessonPlanAssetByID(ctx, id)
        if err != nil {
                if errors.Is(err, repository.ErrLessonPlanAssetNotFound) {
                        return ErrAssetNotFound
                }
                return err
        }
        plan, err := repository.GetLessonPlanByID(ctx, a.LessonPlanID)
        if err != nil {
                return err
        }
        if plan.AuthorID != callerID {
                return ErrAssetUnauthorized
        }

        // 1. 删数据库行(优先,因为这是用户感知的"删除")
        if err := repository.DeleteLessonPlanAsset(ctx, id); err != nil {
                return err
        }
        // 2. 异步删物理文件(失败仅记日志,不阻塞)
        fullPath := filepath.Join(AssetUploadDir, a.FilePath)
        if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
                assetLog.Warn("物理文件删除失败(数据库已删除)",
                        "asset_id", id, "path", fullPath, "error", err,
                )
        }
        assetLog.Info("教案资产已删除",
                "asset_id", id, "plan_id", a.LessonPlanID, "file", a.FileName,
        )
        return nil
}
