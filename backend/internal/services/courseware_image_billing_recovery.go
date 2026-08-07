package services

// courseware_image_billing_recovery.go — 图片媒体计费幂等恢复与身份校验
//
// 本文件只负责已有计费记录的恢复：
//   - 校验用户、课件、业务节点、模型和媒体类型一致；
//   - reserved且有资产时补结算；
//   - settled且有资产时直接返回原资产；
//   - reserved无资产时返回处理中；
//   - 已结算但未形成资产时返回明确业务错误。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// recoverBilledCoursewareImage 恢复同一幂等键的已有图片计费记录。
func recoverBilledCoursewareImage(
	ctx context.Context,
	billing *models.TokenMediaBilling,
	session *coursewareImageBillingSession,
	input *coursewareImageBillingInput,
) (
	*ai.ImageGenerateResult,
	*models.CoursewareAsset,
	error,
) {
	if billing == nil || session == nil || input == nil {
		return nil, nil, ErrMediaBillingInvalidRequest
	}
	if err := validateRecoveredCoursewareImageIdentity(
		billing,
		input,
	); err != nil {
		return nil, nil, err
	}

	switch billing.Status {
	case models.MediaBillingStatusFailed,
		models.MediaBillingStatusCancelled:
		return nil, nil, fmt.Errorf(
			"%w: %s",
			ErrCoursewareImageBillingTerminal,
			billing.Status,
		)

	case models.MediaBillingStatusReserved:
		if billing.AssetID == nil ||
			strings.TrimSpace(*billing.AssetID) == "" {
			return nil, nil, ErrCoursewareImageBillingInProgress
		}

	case models.MediaBillingStatusSettled:

	default:
		return nil, nil, fmt.Errorf(
			"%w: 未知状态%s",
			ErrCoursewareImageBillingTerminal,
			billing.Status,
		)
	}

	if billing.AssetID == nil || strings.TrimSpace(*billing.AssetID) == "" {
		if billing.Status == models.MediaBillingStatusSettled {
			return nil, nil, ErrCoursewareImageBillingOutputMissing
		}
		return nil, nil, ErrCoursewareImageBillingAssetLost
	}

	assetID := strings.TrimSpace(*billing.AssetID)
	asset, err := repository.GetCWAssetByID(
		ctx,
		assetID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%w: %s: %v",
			ErrCoursewareImageBillingAssetLost,
			assetID,
			err,
		)
	}

	result := recoveredCoursewareImageResult(
		billing,
		asset,
	)

	if billing.Status == models.MediaBillingStatusReserved {
		session.preserve()

		settleErr := session.settleProviderUsage(
			asset,
			map[string]interface{}{
				"asset_id":          asset.ID,
				"recovered_billing": true,
				"recovered_from":    "reserved_with_asset",
				"billing_node_code": billing.BillingNodeCode,
			},
		)
		if settleErr != nil {
			coursewareImageBillingLog.Error(
				"恢复图片计费记录时补结算失败，预留继续保持",
				"idempotency_key", billing.IdempotencyKey,
				"asset_id", asset.ID,
				"error", settleErr,
			)
		}
	}

	return result, asset, nil
}

// validateRecoveredCoursewareImageIdentity 防止同一幂等键跨用户、课件或业务节点复用。
func validateRecoveredCoursewareImageIdentity(
	billing *models.TokenMediaBilling,
	input *coursewareImageBillingInput,
) error {
	if billing == nil || input == nil {
		return ErrMediaBillingInvalidRequest
	}

	billingCoursewareID := ""
	if billing.CoursewareID != nil {
		billingCoursewareID = strings.TrimSpace(
			*billing.CoursewareID,
		)
	}

	if strings.TrimSpace(billing.UserID) != input.UserID ||
		strings.TrimSpace(billing.BillingNodeCode) != input.BillingNodeCode ||
		billingCoursewareID != input.CoursewareID ||
		strings.TrimSpace(billing.ModelName) != input.ModelName ||
		strings.TrimSpace(billing.MediaType) != models.MediaTypeImage {
		return fmt.Errorf(
			"%w: %s",
			ErrCoursewareImageBillingIdentityMismatch,
			billing.IdempotencyKey,
		)
	}

	return nil
}

// recoveredCoursewareImageResult 为幂等重放构造业务兼容结果。
func recoveredCoursewareImageResult(
	billing *models.TokenMediaBilling,
	asset *models.CoursewareAsset,
) *ai.ImageGenerateResult {
	urls := make([]string, 0, 1)

	if asset != nil {
		url := strings.TrimSpace(asset.OssURL)
		if url != "" {
			urls = append(urls, url)
		}
	}

	modelName := ""
	if billing != nil {
		modelName = billing.ModelName
	}

	return &ai.ImageGenerateResult{
		URLs:      urls,
		ModelUsed: modelName,
	}
}
