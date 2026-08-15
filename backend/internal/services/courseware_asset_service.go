package services

// courseware_asset_service.go — 课件多媒体资产公共定义
//
// 本文件仅保留课件资产服务的公共常量、服务对象和公网URL解析。
// 具体业务已经按职责拆分：
//   - courseware_asset_image_generation.go：普通页面AI图片生成与积分计费；
//   - courseware_asset_library.go：手动上传和素材查询；
//   - courseware_asset_delete_oss.go：删除资产和OSS上云；
//   - courseware_asset_style_insert.go：风格锚点和页面插图。

import (
	"regexp"
	"strings"

	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
)

const (
	// CWAssetUploadDir 课件图片、视频和音频的物理存储根目录。
	CWAssetUploadDir = "/www/wwwroot/tedna/uploads/courseware-assets"

	// CWAssetURLPrefix Nginx对课件资产目录的URL映射前缀。
	CWAssetURLPrefix = "/uploads/courseware-assets/"

	// CWAssetPublicFileMode 是已经完成校验并正式发布到 /uploads/ 的资产文件权限。
	// 临时文件保持私有；只有正式资产在原子落盘完成后才切换为 Nginx 可读的 0644。
	CWAssetPublicFileMode = 0o644

	// CWAssetMaxSize 单张手动上传图片最大5MB。
	CWAssetMaxSize = 5 * 1024 * 1024

	// cwAssetPublicHost 把本地/uploads/路径转换为公网URL时使用的域名。
	cwAssetPublicHost = "https://workflow.pkuailab.com"
)

// 允许手动上传的图片MIME类型。
var cwAssetAllowedMimeTypes = map[string]bool{
	"image/jpeg":    true,
	"image/jpg":     true,
	"image/png":     true,
	"image/webp":    true,
	"image/gif":     true,
	"image/svg+xml": true,
}

// 图片MIME类型到文件扩展名的映射。
var cwAssetMimeToExt = map[string]string{
	"image/jpeg":    ".jpg",
	"image/jpg":     ".jpg",
	"image/png":     ".png",
	"image/webp":    ".webp",
	"image/gif":     ".gif",
	"image/svg+xml": ".svg",
}

// 文件名安全化正则。
var cwAssetSafeNameRe = regexp.MustCompile(`[^a-zA-Z0-9\p{Han}_-]`)

var cwAssetLog = logger.WithModule("courseware_asset")

// CoursewareAssetService 课件多媒体资产服务。
type CoursewareAssetService struct {
	cfg *config.Config
}

// NewCoursewareAssetService 创建课件多媒体资产服务。
func NewCoursewareAssetService(cfg *config.Config) *CoursewareAssetService {
	return &CoursewareAssetService{cfg: cfg}
}

// resolveAssetPublicURL 解析资产的公网可访问URL。
//
// 优先级：
//   - 已写入的OSS公网URL；
//   - 已经是HTTP或HTTPS的地址；
//   - 本地/uploads/路径补充公网域名。
func resolveAssetPublicURL(asset *models.CoursewareAsset) string {
	if asset == nil {
		return ""
	}

	publicURL := strings.TrimSpace(asset.PublicOSSURL)
	if publicURL != "" {
		return publicURL
	}

	assetURL := strings.TrimSpace(asset.OssURL)
	if assetURL == "" {
		return ""
	}

	if strings.HasPrefix(assetURL, "http://") ||
		strings.HasPrefix(assetURL, "https://") {
		return assetURL
	}

	if strings.HasPrefix(assetURL, "/uploads/") {
		return cwAssetPublicHost + assetURL
	}

	return ""
}
