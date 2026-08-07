package services

// courseware_asset_style_insert.go — 课件风格锚点与页面插图

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// SetStyleAnchorResult 设置风格锚点后的业务响应。
type SetStyleAnchorResult struct {
	AssetID   string `json:"asset_id"`
	AnchorURL string `json:"anchor_url"`
	VAOCI     string `json:"vaoci"`
}

// SetStyleAnchor 设置课件风格锚点。
//
// 流程：
//   - 校验作者和资产归属；
//   - 必要时把本地图片上传OSS；
//   - 多模态读取锚点图片并提取VAOCI；
//   - 保存资产ID和VAOCI到课件。
func (s *CoursewareAssetService) SetStyleAnchor(
	ctx context.Context,
	coursewareID string,
	assetID string,
	actor *CoursewareActorContext,
) (*SetStyleAnchorResult, error) {
	_, scopedActor, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
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
	if asset.AssetType != models.CWAssetTypeImage {
		return nil, fmt.Errorf("仅图片可设为风格锚点")
	}

	if strings.TrimSpace(asset.PublicOSSURL) == "" &&
		strings.HasPrefix(asset.OssURL, "/uploads/") {
		ossService := NewOSSService(s.cfg)
		publicURL, uploadErr := ossService.UploadAssetToOSS(asset.OssURL)
		if uploadErr != nil {
			return nil, fmt.Errorf(
				"锚点图上传云盘失败（设锚点需稳定公网地址）: %w",
				uploadErr,
			)
		}

		if updateErr := repository.UpdateCWAssetPublicURL(
			ctx,
			assetID,
			publicURL,
		); updateErr != nil {
			cwAssetLog.Warn(
				"锚点图OSS地址回写失败，不阻断设锚点",
				"asset_id", assetID,
				"error", updateErr,
			)
		}

		asset.PublicOSSURL = publicURL
		cwAssetLog.Info(
			"设锚点：锚点图已自动上云",
			"asset_id", assetID,
			"public_oss_url", publicURL,
		)
	}

	anchorURL := resolveAssetPublicURL(asset)
	if anchorURL == "" {
		return nil, fmt.Errorf(
			"无法解析锚点图的公网URL，请确认图片已正确保存",
		)
	}

	vaoci, err := s.ExtractVAOCIFromImageURL(
		ctx,
		anchorURL,
		scopedActor.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("提取风格索引失败: %w", err)
	}

	if err := repository.UpdateCoursewareStyleAnchor(
		ctx,
		coursewareID,
		assetID,
		vaoci,
	); err != nil {
		return nil, fmt.Errorf("保存风格锚点失败: %w", err)
	}

	cwAssetLog.Info(
		"设置课件风格锚点成功",
		"courseware_id", coursewareID,
		"anchor_asset_id", assetID,
		"vaoci_len", len([]rune(vaoci)),
	)

	return &SetStyleAnchorResult{
		AssetID:   assetID,
		AnchorURL: anchorURL,
		VAOCI:     vaoci,
	}, nil
}

// ClearStyleAnchor 清除课件风格锚点。
func (s *CoursewareAssetService) ClearStyleAnchor(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) error {
	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return err
	}

	if err := repository.ClearCoursewareStyleAnchor(
		ctx,
		coursewareID,
	); err != nil {
		return fmt.Errorf("清除风格锚点失败: %w", err)
	}

	cwAssetLog.Info(
		"清除课件风格锚点成功",
		"courseware_id", coursewareID,
	)
	return nil
}

// InsertImageToPage 将图片插入到页面HTML。
//
// 两种模式：
//   - placeholder_id非空：替换匹配的占位符；
//   - placeholder_id为空：追加到页面内容区末尾。
func (s *CoursewareAssetService) InsertImageToPage(
	ctx context.Context,
	coursewareID string,
	pageNumber int,
	assetID string,
	actor *CoursewareActorContext,
) (string, error) {
	if _, _, err :=
		(&CoursewareService{}).LoadCoursewareForOwnerRuntime(
			ctx,
			coursewareID,
			actor,
		); err != nil {
		return "", err
	}

	asset, err := repository.GetCWAssetByID(ctx, assetID)
	if err != nil {
		return "", fmt.Errorf("资产不存在: %w", err)
	}
	if asset.CoursewareID != coursewareID {
		return "", fmt.Errorf("资产不属于此课件")
	}

	page, err := repository.GetCoursewarePageByNumber(
		ctx,
		coursewareID,
		pageNumber,
	)
	if err != nil {
		return "", fmt.Errorf("页面不存在")
	}
	if page.HTMLContent == "" {
		return "", fmt.Errorf("页面尚未生成HTML，请先生成课件")
	}

	html := page.HTMLContent
	imageTag := fmt.Sprintf(
		`<img src="%s" alt="课件图片" `+
			`style="max-width:100%%;height:auto;`+
			`border-radius:var(--cw-radius,12px);`+
			`margin:12px 0" />`,
		asset.OssURL,
	)

	if asset.PlaceholderID != "" {
		placeholderPattern := fmt.Sprintf(
			`<div[^>]*data-placeholder-id="%s"[^>]*>[\s\S]*?</div>`,
			regexp.QuoteMeta(asset.PlaceholderID),
		)
		compiled, compileErr := regexp.Compile(placeholderPattern)

		if compileErr == nil && compiled.MatchString(html) {
			html = compiled.ReplaceAllString(html, imageTag)
			cwAssetLog.Info(
				"替换占位符为图片",
				"courseware_id", coursewareID,
				"page_number", pageNumber,
				"placeholder_id", asset.PlaceholderID,
			)
		} else {
			cwAssetLog.Warn(
				"未找到占位符，降级为追加模式",
				"placeholder_id", asset.PlaceholderID,
				"page_number", pageNumber,
			)
			html = appendImageToHTML(html, imageTag)
		}
	} else {
		html = appendImageToHTML(html, imageTag)
	}

	if err := repository.UpdateCWPageHTML(
		ctx,
		page.ID,
		html,
		page.PlaceholderMap,
		page.MatchedComponentIDs,
		page.Status,
	); err != nil {
		return "", fmt.Errorf("更新页面HTML失败: %w", err)
	}

	_ = repository.UpdateCWAssetStatus(
		ctx,
		assetID,
		models.CWAssetStatusConfirmed,
	)

	cwAssetLog.Info(
		"图片已插入页面HTML",
		"courseware_id", coursewareID,
		"page_number", pageNumber,
		"asset_id", assetID,
	)

	return html, nil
}

// appendImageToHTML 在HTML内容区末尾插入图片。
func appendImageToHTML(html string, imageTag string) string {
	wrappedImage := fmt.Sprintf(
		`<div style="text-align:center;padding:16px 40px">%s</div>`,
		imageTag,
	)
	lastClose := strings.LastIndex(html, "</div>")
	if lastClose < 0 {
		return html + "\n" + wrappedImage
	}
	return html[:lastClose] +
		"\n" + wrappedImage + "\n" +
		html[lastClose:]
}
