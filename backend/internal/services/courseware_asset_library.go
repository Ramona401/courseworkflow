package services

// courseware_asset_library.go — 课件图片上传与素材查询

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// UploadAssetRequest 手动上传图片请求。
type UploadAssetRequest struct {
	CoursewareID  string
	PageNumber    int
	PlaceholderID string
	Actor         *CoursewareActorContext
}

// UploadAssetResponse 手动上传图片响应。
type UploadAssetResponse struct {
	AssetID  string `json:"asset_id"`
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
}

// UploadAsset 手动上传图片到本地磁盘并写入课件资产表。
func (s *CoursewareAssetService) UploadAsset(
	ctx context.Context,
	request *UploadAssetRequest,
	file multipart.File,
	header *multipart.FileHeader,
) (*UploadAssetResponse, error) {
	if request == nil || file == nil || header == nil {
		return nil, ErrCoursewareActorRequired
	}

	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			request.CoursewareID,
			request.Actor,
		); err != nil {
		return nil, err
	}

	if header.Size > CWAssetMaxSize {
		return nil, fmt.Errorf(
			"图片文件过大，最大支持5MB（当前%.1fMB）",
			float64(header.Size)/(1024*1024),
		)
	}

	mimeType, err := resolveCoursewareUploadImageMIME(header)
	if err != nil {
		return nil, err
	}

	page, err := repository.GetCoursewarePageByNumber(
		ctx,
		request.CoursewareID,
		request.PageNumber,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"页面不存在: 课件=%s 页码=%d",
			request.CoursewareID,
			request.PageNumber,
		)
	}

	extension := cwAssetMimeToExt[mimeType]
	if extension == "" {
		extension = ".png"
	}

	storedName := buildCoursewareUploadImageName(
		header.Filename,
		extension,
	)

	assetDirectory := filepath.Join(
		CWAssetUploadDir,
		request.CoursewareID,
		fmt.Sprintf("p%d", request.PageNumber),
	)
	if err := os.MkdirAll(assetDirectory, 0755); err != nil {
		return nil, fmt.Errorf("创建图片目录失败: %w", err)
	}

	fullPath := filepath.Join(assetDirectory, storedName)
	destination, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer destination.Close()

	written, err := io.Copy(destination, file)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	relativePath := filepath.Join(
		request.CoursewareID,
		fmt.Sprintf("p%d", request.PageNumber),
		storedName,
	)
	assetURL := CWAssetURLPrefix + relativePath

	asset := &models.CoursewareAsset{
		CoursewareID:     request.CoursewareID,
		PageID:           &page.ID,
		PlaceholderID:    strings.TrimSpace(request.PlaceholderID),
		AssetType:        models.CWAssetTypeImage,
		GenerationPrompt: "",
		OssURL:           assetURL,
		FileSize:         written,
		MimeType:         mimeType,
		Status:           models.CWAssetStatusUploaded,
	}
	if err := repository.CreateCWAsset(ctx, asset); err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("记录图片资产失败: %w", err)
	}

	cwAssetLog.Info(
		"课件图片上传成功",
		"courseware_id", request.CoursewareID,
		"page_number", request.PageNumber,
		"asset_id", asset.ID,
		"file", header.Filename,
		"size", written,
		"url", assetURL,
	)

	return &UploadAssetResponse{
		AssetID:  asset.ID,
		URL:      assetURL,
		FileName: storedName,
		FileSize: written,
		MimeType: mimeType,
	}, nil
}

// resolveCoursewareUploadImageMIME 解析并校验上传图片的MIME类型。
func resolveCoursewareUploadImageMIME(
	header *multipart.FileHeader,
) (string, error) {
	if header == nil {
		return "", fmt.Errorf("图片文件信息为空")
	}

	mimeType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mimeType == "" {
		switch strings.ToLower(filepath.Ext(header.Filename)) {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".webp":
			mimeType = "image/webp"
		case ".gif":
			mimeType = "image/gif"
		case ".svg":
			mimeType = "image/svg+xml"
		default:
			return "", fmt.Errorf(
				"不支持的图片格式，支持JPG/PNG/WEBP/GIF/SVG",
			)
		}
	}

	if !cwAssetAllowedMimeTypes[mimeType] {
		return "", fmt.Errorf(
			"不支持的图片格式，支持JPG/PNG/WEBP/GIF/SVG",
		)
	}

	return mimeType, nil
}

// buildCoursewareUploadImageName 构造安全、可读且不会切断中文的文件名。
func buildCoursewareUploadImageName(
	originalName string,
	extension string,
) string {
	baseName := strings.TrimSuffix(
		filepath.Base(originalName),
		filepath.Ext(originalName),
	)
	baseName = cwAssetSafeNameRe.ReplaceAllString(baseName, "_")

	for strings.Contains(baseName, "__") {
		baseName = strings.ReplaceAll(baseName, "__", "_")
	}

	baseName = strings.Trim(baseName, "_")
	baseRunes := []rune(baseName)
	if len(baseRunes) > 40 {
		baseName = string(baseRunes[:40])
	}
	if baseName == "" {
		baseName = "image"
	}

	return fmt.Sprintf(
		"%d_%s%s",
		time.Now().UnixMilli(),
		baseName,
		extension,
	)
}

// ListPageAssets 获取指定页面的全部图片、视频和音频资产。
func (s *CoursewareAssetService) ListPageAssets(
	ctx context.Context,
	coursewareID string,
	pageNumber int,
	actor *CoursewareActorContext,
) ([]*models.CoursewareAsset, error) {
	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return nil, err
	}

	page, err := repository.GetCoursewarePageByNumber(
		ctx,
		coursewareID,
		pageNumber,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"页面不存在: 课件=%s 页码=%d",
			coursewareID,
			pageNumber,
		)
	}

	return repository.ListCWAssetsByPage(ctx, page.ID)
}

// ListCoursewareAssets 获取课件的全部图片、视频和音频资产。
func (s *CoursewareAssetService) ListCoursewareAssets(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) ([]*models.CoursewareAsset, error) {
	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return nil, err
	}

	return repository.ListCWAssetsByCourseware(ctx, coursewareID)
}
