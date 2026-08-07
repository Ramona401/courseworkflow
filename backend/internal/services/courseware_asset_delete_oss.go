package services

// courseware_asset_delete_oss.go — 课件资产删除与OSS上云

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// DeleteAsset 删除图片、视频或音频资产。
//
// 删除范围包括：
//   - 本地物理文件；
//   - 数据库资产记录；
//   - 已上传OSS的云盘副本；
//   - 当前资产恰好为课件风格锚点时，清理锚点引用。
func (s *CoursewareAssetService) DeleteAsset(
	ctx context.Context,
	coursewareID string,
	assetID string,
	actor *CoursewareActorContext,
) error {
	courseware, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return err
	}

	asset, err := repository.GetCWAssetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("资产不存在: %w", err)
	}
	if err := validateCoursewareAssetForCourseware(
		coursewareID,
		asset,
	); err != nil {
		return err
	}

	if courseware.StyleAnchorAssetID != nil &&
		*courseware.StyleAnchorAssetID == assetID {
		if clearErr := repository.ClearCoursewareStyleAnchor(
			ctx,
			courseware.ID,
		); clearErr != nil {
			cwAssetLog.Warn(
				"删除锚点资产时清空课件锚点引用失败，继续删除资产",
				"asset_id", assetID,
				"courseware_id", courseware.ID,
				"error", clearErr,
			)
		} else {
			cwAssetLog.Info(
				"删除锚点资产，已连带清空课件锚点引用",
				"asset_id", assetID,
				"courseware_id", courseware.ID,
			)
		}
	}

	removeCoursewareAssetLocalFile(asset)
	s.deleteCoursewareAssetOSSObject(asset)

	if err := repository.DeleteCWAsset(ctx, assetID); err != nil {
		return fmt.Errorf("删除资产记录失败: %w", err)
	}

	cwAssetLog.Info(
		"课件资产删除成功",
		"asset_id", assetID,
		"asset_type", asset.AssetType,
		"courseware_id", asset.CoursewareID,
	)

	return nil
}

// removeCoursewareAssetLocalFile 删除课件资产对应的本地文件。
func removeCoursewareAssetLocalFile(asset *models.CoursewareAsset) {
	if asset == nil ||
		!strings.HasPrefix(asset.OssURL, CWAssetURLPrefix) {
		return
	}

	relativePath := strings.TrimPrefix(
		asset.OssURL,
		CWAssetURLPrefix,
	)
	if strings.TrimSpace(relativePath) == "" {
		return
	}

	fullPath := filepath.Join(CWAssetUploadDir, relativePath)
	if err := os.Remove(fullPath); err != nil &&
		!os.IsNotExist(err) {
		cwAssetLog.Warn(
			"删除课件资产本地文件失败",
			"asset_id", asset.ID,
			"path", fullPath,
			"error", err,
		)
	}
}

// deleteCoursewareAssetOSSObject 尽力删除资产对应的OSS对象。
func (s *CoursewareAssetService) deleteCoursewareAssetOSSObject(
	asset *models.CoursewareAsset,
) {
	if asset == nil || strings.TrimSpace(asset.PublicOSSURL) == "" {
		return
	}

	ossService := NewOSSService(s.cfg)
	if err := ossService.DeleteObjectFromOSS(
		asset.PublicOSSURL,
	); err != nil {
		cwAssetLog.Warn(
			"删除OSS云盘副本失败，本地资产仍继续删除",
			"asset_id", asset.ID,
			"public_oss_url", asset.PublicOSSURL,
			"error", err,
		)
		return
	}

	cwAssetLog.Info(
		"OSS云盘副本已删除",
		"asset_id", asset.ID,
		"public_oss_url", asset.PublicOSSURL,
	)
}

// UploadCoursewareAssetOSSResult 课件资产上云结果。
type UploadCoursewareAssetOSSResult struct {
	AssetID      string `json:"asset_id"`
	LocalURL     string `json:"local_url"`
	OSSPublicURL string `json:"oss_public_url"`
	Message      string `json:"message"`
}

// validateCoursewareAssetForCourseware 校验资产真实归属于路径课件。
func validateCoursewareAssetForCourseware(
	coursewareID string,
	asset *models.CoursewareAsset,
) error {
	if asset == nil {
		return fmt.Errorf("资产不存在")
	}
	if asset.CoursewareID != coursewareID {
		return fmt.Errorf("资产不属于路径指定课件")
	}
	return nil
}

// UploadCoursewareAssetToOSS 在作者专属运行通道中上传课件资产到OSS。
//
// 低层OSSService保持无身份语义，供装配、背景生产等可信内部流程复用；
// HTTP作者授权、路径课件归属和资产数据库校验统一在本方法完成。
func (s *CoursewareAssetService) UploadCoursewareAssetToOSS(
	ctx context.Context,
	coursewareID string,
	assetID string,
	actor *CoursewareActorContext,
) (*UploadCoursewareAssetOSSResult, error) {
	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return nil, err
	}

	asset, err := repository.GetCWAssetByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("资产不存在: %w", err)
	}
	if err := validateCoursewareAssetForCourseware(
		coursewareID,
		asset,
	); err != nil {
		return nil, err
	}

	localURL := strings.TrimSpace(asset.OssURL)
	if localURL == "" || !strings.HasPrefix(localURL, "/uploads/") {
		return nil, fmt.Errorf("资产没有本地文件或已经是外部URL")
	}

	ossService := NewOSSService(s.cfg)
	publicURL, err := ossService.UploadAssetToOSS(localURL)
	if err != nil {
		return nil, fmt.Errorf("上传云盘失败: %w", err)
	}

	message := "上传云盘成功"
	if updateErr := repository.UpdateCWAssetPublicURL(
		ctx,
		assetID,
		publicURL,
	); updateErr != nil {
		message = "上传云盘成功(URL持久化失败,不影响使用)"
		cwAssetLog.Warn(
			"课件资产OSS地址回写失败",
			"courseware_id", coursewareID,
			"asset_id", assetID,
			"public_url", publicURL,
			"error", updateErr,
		)
	}

	return &UploadCoursewareAssetOSSResult{
		AssetID:      assetID,
		LocalURL:     localURL,
		OSSPublicURL: publicURL,
		Message:      message,
	}, nil
}
