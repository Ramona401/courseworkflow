package services

// courseware_style_studio_asset.go — 风格工作室课程级图片资产
//
// 本文件负责：
//   - 上传老师的参考图片；
//   - 图片保存到课程级style-studio目录；
//   - 参考图片和预览图片均使用page_id=NULL；
//   - 下载图片模型返回的临时图片；
//   - 按真实文件签名识别预览图片格式，避免扩展名与文件内容不一致；
//   - 创建courseware_assets记录；
//   - 校验图片必须属于当前课件。
//
// 重要说明：
// 图片供应商不一定返回可靠的Content-Type。
// 若把实际JPEG或WEBP文件错误保存为PNG，同时Nginx返回image/png并启用nosniff，
// 浏览器可能拒绝加载。因此预览图必须先保存临时文件，再按文件签名确定MIME和扩展名。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwStyleStudioAssetMaxSize   = 8 * 1024 * 1024
	cwStyleStudioPreviewMaxSize = 24 * 1024 * 1024
	cwStyleStudioImageMinSize   = 100
)

var cwStyleStudioAllowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

// CoursewareStyleStudioAssetResult 风格工作室图片上传结果。
type CoursewareStyleStudioAssetResult struct {
	AssetID string `json:"asset_id"`
	URL     string `json:"url"`

	PublicURL string `json:"public_url"`

	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
}

// UploadReferenceImage 上传课程级风格参考图片。
func (s *CoursewareStyleStudioService) UploadReferenceImage(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	file multipart.File,
	header *multipart.FileHeader,
) (*CoursewareStyleStudioAssetResult, error) {
	courseware, _, err :=
		s.loadStyleStudioCourseware(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if file == nil ||
		header == nil {
		return nil, fmt.Errorf(
			"缺少风格参考图片",
		)
	}

	if header.Size <= 0 {
		return nil, fmt.Errorf(
			"参考图片为空",
		)
	}

	if header.Size >
		cwStyleStudioAssetMaxSize {
		return nil, fmt.Errorf(
			"参考图片过大，最大支持8MB（当前%.1fMB）",
			float64(header.Size)/(1024*1024),
		)
	}

	mimeType :=
		strings.ToLower(
			strings.TrimSpace(
				header.Header.Get(
					"Content-Type",
				),
			),
		)

	if mimeType == "" {
		mimeType =
			styleStudioMimeFromExtension(
				header.Filename,
			)
	}

	if !cwStyleStudioAllowedMimeTypes[mimeType] {
		return nil, fmt.Errorf(
			"风格参考图仅支持JPG、PNG和WEBP",
		)
	}

	extension :=
		styleStudioExtensionForMime(
			mimeType,
		)

	baseName :=
		strings.TrimSuffix(
			filepath.Base(
				header.Filename,
			),
			filepath.Ext(
				header.Filename,
			),
		)

	baseName =
		cwAssetSafeNameRe.
			ReplaceAllString(
				baseName,
				"_",
			)

	baseName =
		strings.Trim(
			baseName,
			"_",
		)

	if baseName == "" {
		baseName =
			"style_reference"
	}

	baseName =
		safeStyleStudioRunes(
			baseName,
			40,
		)

	storedName := fmt.Sprintf(
		"%d_reference_%s%s",
		time.Now().UnixMilli(),
		baseName,
		extension,
	)

	assetDir := filepath.Join(
		CWAssetUploadDir,
		courseware.ID,
		"style-studio",
		"references",
	)

	if err := os.MkdirAll(
		assetDir,
		0755,
	); err != nil {
		return nil, fmt.Errorf(
			"创建风格参考图目录失败: %w",
			err,
		)
	}

	fullPath :=
		filepath.Join(
			assetDir,
			storedName,
		)

	destination, err :=
		os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf(
			"创建风格参考图片失败: %w",
			err,
		)
	}

	written, copyErr :=
		io.Copy(
			destination,
			file,
		)

	closeErr :=
		destination.Close()

	if copyErr != nil {
		_ = os.Remove(fullPath)

		return nil, fmt.Errorf(
			"保存风格参考图片失败: %w",
			copyErr,
		)
	}

	if closeErr != nil {
		_ = os.Remove(fullPath)

		return nil, fmt.Errorf(
			"关闭风格参考图片失败: %w",
			closeErr,
		)
	}

	relativePath := filepath.Join(
		courseware.ID,
		"style-studio",
		"references",
		storedName,
	)

	localURL :=
		CWAssetURLPrefix +
			filepath.ToSlash(
				relativePath,
			)

	metadataJSON, _ :=
		json.Marshal(
			map[string]interface{}{
				"style_studio_role": "reference",
				"original_filename": header.Filename,
			},
		)

	asset := &models.CoursewareAsset{
		CoursewareID:     courseware.ID,
		PageID:           nil,
		PlaceholderID:    "style-reference",
		AssetType:        models.CWAssetTypeImage,
		GenerationPrompt: "",
		OssURL:           localURL,
		FileSize:         written,
		MimeType:         mimeType,
		Metadata: string(
			metadataJSON,
		),
		Status: models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			asset,
		); err != nil {
		_ = os.Remove(fullPath)

		return nil, fmt.Errorf(
			"记录风格参考图片失败: %w",
			err,
		)
	}

	publicURL :=
		resolveAssetPublicURL(
			asset,
		)

	styleStudioLog.Info(
		"上传课程美术风格参考图",
		"courseware_id", courseware.ID,
		"asset_id", asset.ID,
		"mime_type", mimeType,
		"file_size", written,
	)

	return &CoursewareStyleStudioAssetResult{
		AssetID:   asset.ID,
		URL:       localURL,
		PublicURL: publicURL,
		FileName:  storedName,
		FileSize:  written,
		MimeType:  mimeType,
	}, nil
}

// saveStyleStudioGeneratedImage 下载并保存一张风格验证图。
//
// 下载过程使用.part临时文件：
//   - 限制最大体积，避免异常响应无限写盘；
//   - 下载完成后按真实文件签名识别JPEG、PNG或WEBP；
//   - 根据真实MIME确定最终扩展名；
//   - 最后原子Rename到正式文件名。
//
// 不能根据上游Content-Type直接决定扩展名，因为部分图片网关会返回
// application/octet-stream、空Content-Type，甚至错误的image/png。
func (s *CoursewareStyleStudioService) saveStyleStudioGeneratedImage(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	previewType string,
	remoteURL string,
	generationPrompt string,
) (*models.CoursewareAsset, error) {
	remoteURL =
		strings.TrimSpace(remoteURL)

	if remoteURL == "" {
		return nil, fmt.Errorf(
			"预览图片远程地址为空",
		)
	}

	client := &http.Client{
		Timeout: 45 * time.Second,
	}

	request, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			remoteURL,
			nil,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"创建预览图片下载请求失败: %w",
			err,
		)
	}

	response, err :=
		client.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"下载风格预览图片失败: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode !=
		http.StatusOK {
		return nil, fmt.Errorf(
			"下载风格预览图片HTTP错误: %d",
			response.StatusCode,
		)
	}

	assetDir := filepath.Join(
		CWAssetUploadDir,
		coursewareID,
		"style-studio",
		"previews",
		sessionID,
	)

	if err := os.MkdirAll(
		assetDir,
		0755,
	); err != nil {
		return nil, fmt.Errorf(
			"创建风格预览目录失败: %w",
			err,
		)
	}

	safePreviewType :=
		cwAssetSafeNameRe.
			ReplaceAllString(
				strings.TrimSpace(
					previewType,
				),
				"_",
			)

	safePreviewType =
		strings.Trim(
			safePreviewType,
			"_",
		)

	if safePreviewType == "" {
		safePreviewType =
			"unknown"
	}

	fileBase := fmt.Sprintf(
		"%d_preview_%s",
		time.Now().UnixMilli(),
		safePreviewType,
	)

	tempPath :=
		filepath.Join(
			assetDir,
			fileBase+".part",
		)

	destination, err :=
		os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf(
			"创建风格预览临时文件失败: %w",
			err,
		)
	}

	limitedReader :=
		io.LimitReader(
			response.Body,
			cwStyleStudioPreviewMaxSize+1,
		)

	written, copyErr :=
		io.Copy(
			destination,
			limitedReader,
		)

	closeErr :=
		destination.Close()

	if copyErr != nil {
		_ = os.Remove(tempPath)

		return nil, fmt.Errorf(
			"保存风格预览图片失败: %w",
			copyErr,
		)
	}

	if closeErr != nil {
		_ = os.Remove(tempPath)

		return nil, fmt.Errorf(
			"关闭风格预览临时文件失败: %w",
			closeErr,
		)
	}

	if written >
		cwStyleStudioPreviewMaxSize {
		_ = os.Remove(tempPath)

		return nil, fmt.Errorf(
			"风格预览图片超过24MB限制",
		)
	}

	if written <
		cwStyleStudioImageMinSize {
		_ = os.Remove(tempPath)

		return nil, fmt.Errorf(
			"风格预览图片内容为空或不完整",
		)
	}

	mimeType, err :=
		detectStyleStudioImageMIME(
			tempPath,
		)
	if err != nil {
		_ = os.Remove(tempPath)

		return nil, err
	}

	extension :=
		styleStudioExtensionForMime(
			mimeType,
		)

	storedName :=
		fileBase +
			extension

	fullPath :=
		filepath.Join(
			assetDir,
			storedName,
		)

	if err := os.Rename(
		tempPath,
		fullPath,
	); err != nil {
		_ = os.Remove(tempPath)

		return nil, fmt.Errorf(
			"确认风格预览图片文件失败: %w",
			err,
		)
	}

	relativePath := filepath.Join(
		coursewareID,
		"style-studio",
		"previews",
		sessionID,
		storedName,
	)

	localURL :=
		CWAssetURLPrefix +
			filepath.ToSlash(
				relativePath,
			)

	metadataJSON, _ :=
		json.Marshal(
			map[string]interface{}{
				"style_studio_role": "preview",
				"style_session_id":  sessionID,
				"preview_type":      previewType,
				"detected_mime_type": mimeType,
			},
		)

	asset := &models.CoursewareAsset{
		CoursewareID: coursewareID,
		PageID:       nil,
		PlaceholderID: "style-preview:" +
			previewType,
		AssetType: models.CWAssetTypeImage,
		GenerationPrompt: strings.TrimSpace(
			generationPrompt,
		),
		OssURL:   localURL,
		FileSize: written,
		MimeType: mimeType,
		Metadata: string(
			metadataJSON,
		),
		Status: models.CWAssetStatusUploaded,
	}

	if err :=
		repository.CreateCWAsset(
			ctx,
			asset,
		); err != nil {
		_ = os.Remove(fullPath)

		return nil, fmt.Errorf(
			"记录风格预览资产失败: %w",
			err,
		)
	}

	styleStudioLog.Info(
		"风格验证图下载并按真实格式落盘",
		"courseware_id", coursewareID,
		"session_id", sessionID,
		"preview_type", previewType,
		"asset_id", asset.ID,
		"mime_type", mimeType,
		"file_size", written,
	)

	return asset, nil
}

// detectStyleStudioImageMIME 根据正式图片文件签名识别MIME。
//
// 不依赖文件扩展名或供应商响应头：
//   - JPEG：FF D8 FF；
//   - PNG：标准8字节签名；
//   - WEBP：RIFF....WEBP。
func detectStyleStudioImageMIME(
	filePath string,
) (string, error) {
	file, err :=
		os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf(
			"读取风格预览图片失败: %w",
			err,
		)
	}
	defer file.Close()

	header :=
		make([]byte, 16)

	readCount, readErr :=
		io.ReadFull(
			file,
			header,
		)

	if readErr != nil &&
		readErr != io.ErrUnexpectedEOF {
		return "", fmt.Errorf(
			"读取风格预览图片签名失败: %w",
			readErr,
		)
	}

	header =
		header[:readCount]

	if len(header) >= 3 &&
		header[0] == 0xFF &&
		header[1] == 0xD8 &&
		header[2] == 0xFF {
		return "image/jpeg", nil
	}

	pngSignature := []byte{
		0x89, 0x50, 0x4E, 0x47,
		0x0D, 0x0A, 0x1A, 0x0A,
	}

	if len(header) >=
		len(pngSignature) &&
		bytes.Equal(
			header[:len(pngSignature)],
			pngSignature,
		) {
		return "image/png", nil
	}

	if len(header) >= 12 &&
		string(header[0:4]) == "RIFF" &&
		string(header[8:12]) == "WEBP" {
		return "image/webp", nil
	}

	detected :=
		http.DetectContentType(
			header,
		)

	return "", fmt.Errorf(
		"风格预览返回的不是受支持的图片文件（检测类型：%s）",
		detected,
	)
}

func (s *CoursewareStyleStudioService) loadStyleStudioImageAsset(
	ctx context.Context,
	coursewareID string,
	assetID string,
) (*models.CoursewareAsset, error) {
	assetID =
		strings.TrimSpace(assetID)

	if assetID == "" {
		return nil, fmt.Errorf(
			"风格图片资产ID不能为空",
		)
	}

	asset, err :=
		repository.GetCWAssetByID(
			ctx,
			assetID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"风格图片资产不存在: %w",
			err,
		)
	}

	if asset.CoursewareID !=
		coursewareID {
		return nil, fmt.Errorf(
			"风格图片不属于当前课件",
		)
	}

	if asset.AssetType !=
		models.CWAssetTypeImage {
		return nil, fmt.Errorf(
			"风格工作室仅支持图片资产",
		)
	}

	return asset, nil
}

func styleStudioMimeFromExtension(
	fileName string,
) string {
	switch strings.ToLower(
		filepath.Ext(fileName),
	) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

func styleStudioExtensionForMime(
	mimeType string,
) string {
	switch strings.ToLower(
		strings.TrimSpace(mimeType),
	) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}
