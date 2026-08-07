package services

// media_billing_service.go — 媒体积分统一预留、结算与释放服务
//
// 调用顺序：Reserve →（异步任务可BindExternalTask）→ Settle或Release。
// 单价未启用或为0时直接拒绝，禁止静默按0积分调用供应商。

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrMediaBillingInvalidRequest     = errors.New("媒体计费请求无效")
	ErrMediaBillingPriceNotConfigured = errors.New("媒体计费尚未配置")
	ErrMediaBillingNodeMismatch       = errors.New("媒体计费业务节点不匹配")
)

var mediaBillingLog = logger.WithModule("services.media_billing")

// MediaBillingService 媒体积分计费服务。
type MediaBillingService struct {
	policyService *CreditPolicyService
}

// NewMediaBillingService 创建媒体积分计费服务。
func NewMediaBillingService() *MediaBillingService {
	return &MediaBillingService{policyService: NewCreditPolicyService()}
}

// Reserve 在供应商调用前冻结预计积分。
func (service *MediaBillingService) Reserve(
	ctx context.Context,
	request *models.MediaBillingReserveRequest,
) (*models.TokenMediaBilling, error) {
	normalized, err := normalizeMediaBillingReserveRequest(request)
	if err != nil {
		return nil, err
	}

	// 幂等重试优先返回原记录，不受后续单价停用或策略变更影响。
	existing, err := repository.GetTokenMediaBillingByKey(ctx, normalized.IdempotencyKey)
	if err == nil {
		existing.ReservationCreated = false
		return existing, nil
	}
	if !errors.Is(err, repository.ErrTokenMediaBillingNotFound) {
		return nil, err
	}

	node, err := repository.GetActiveTokenBillingNode(ctx, normalized.BillingNodeCode)
	if err != nil {
		return nil, err
	}
	if node.Category != normalized.BillingCategory ||
		node.MediaType != normalized.MediaType {
		return nil, ErrMediaBillingNodeMismatch
	}
	if normalized.SceneCode == "" {
		normalized.SceneCode = node.SceneCode
	}

	price, err := repository.GetActiveTokenMediaPrice(
		ctx,
		normalized.MediaType,
		normalized.Provider,
		normalized.ModelName,
		normalized.Variant,
		normalized.MediaUnit,
	)
	if errors.Is(err, repository.ErrTokenMediaPriceUnavailable) {
		return nil, fmt.Errorf(
			"%w: %s/%s/%s/%s",
			ErrMediaBillingPriceNotConfigured,
			normalized.MediaType,
			normalized.Provider,
			normalized.ModelName,
			normalized.MediaUnit,
		)
	}
	if err != nil {
		return nil, err
	}
	if !price.IsActive || price.UnitCostUSD <= 0 {
		return nil, fmt.Errorf(
			"%w: 单价未启用或为0",
			ErrMediaBillingPriceNotConfigured,
		)
	}

	policy := service.policyService.GetEffectivePolicy(ctx, normalized.SchoolID)
	quote := quoteMediaBilling(normalized.EstimatedQuantity, price, policy)
	if quote.CostUSD <= 0 || quote.Credits <= 0 {
		return nil, fmt.Errorf(
			"%w: 计费结果为0",
			ErrMediaBillingPriceNotConfigured,
		)
	}

	if normalized.Metadata == nil {
		normalized.Metadata = map[string]interface{}{}
	}
	normalized.Metadata["media_price_id"] = price.ID
	normalized.Metadata["billing_node_name"] = node.DisplayName
	normalized.Metadata["billable_quantity"] = quote.BillableQuantity

	billing, err := repository.ReserveTokenMediaBilling(
		ctx,
		&models.TokenMediaReserveSnapshot{
			MediaBillingReserveRequest: *normalized,
			Quote:                      quote,
		},
	)
	if err != nil {
		return nil, err
	}

	mediaBillingLog.Info(
		"媒体积分预留完成",
		"idempotency_key", billing.IdempotencyKey,
		"node", billing.BillingNodeCode,
		"media_type", billing.MediaType,
		"reserved_credits", billing.ReservedCredits,
	)
	return billing, nil
}

// BindAsset 绑定已经成功写入业务库的媒体资产。
func (service *MediaBillingService) BindAsset(
	ctx context.Context,
	request *models.MediaBillingBindAssetRequest,
) (*models.TokenMediaBilling, error) {
	if request == nil ||
		strings.TrimSpace(request.IdempotencyKey) == "" ||
		strings.TrimSpace(request.AssetID) == "" {
		return nil, ErrMediaBillingInvalidRequest
	}

	normalized := *request
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.AssetID = strings.TrimSpace(normalized.AssetID)
	return repository.BindTokenMediaBillingAsset(ctx, &normalized)
}

// BindExternalTask 绑定供应商异步任务ID。
func (service *MediaBillingService) BindExternalTask(
	ctx context.Context,
	request *models.MediaBillingBindTaskRequest,
) (*models.TokenMediaBilling, error) {
	if request == nil ||
		strings.TrimSpace(request.IdempotencyKey) == "" ||
		strings.TrimSpace(request.ExternalTaskID) == "" {
		return nil, ErrMediaBillingInvalidRequest
	}

	normalized := *request
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.ExternalTaskID = strings.TrimSpace(normalized.ExternalTaskID)
	return repository.BindTokenMediaBillingExternalTask(ctx, &normalized)
}

// AnnotateReserved 合并更新仍处于reserved状态的补偿元数据。
//
// 本方法不改变冻结金额或计费状态，主要用于保存供应商结果不确定、
// 供应商成功但结算失败等必须跨进程恢复的事实。
func (service *MediaBillingService) AnnotateReserved(
        ctx context.Context,
        request *models.MediaBillingAnnotateRequest,
) (*models.TokenMediaBilling, error) {
        if request == nil ||
                strings.TrimSpace(
                        request.IdempotencyKey,
                ) == "" ||
                len(request.Metadata) == 0 {
                return nil,
                        ErrMediaBillingInvalidRequest
        }

        normalized := *request
        normalized.IdempotencyKey =
                strings.TrimSpace(
                        normalized.IdempotencyKey,
                )

        return repository.AnnotateTokenMediaBilling(
                ctx,
                &normalized,
        )
}

// Settle 在业务结果持久化后按实际数量结算。
func (service *MediaBillingService) Settle(
	ctx context.Context,
	request *models.MediaBillingSettleRequest,
) (*models.TokenMediaBilling, error) {
	if request == nil ||
		strings.TrimSpace(request.IdempotencyKey) == "" ||
		request.ActualQuantity <= 0 ||
		request.LatencyMs < 0 {
		return nil, ErrMediaBillingInvalidRequest
	}

	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	billing, err := repository.GetTokenMediaBillingByKey(ctx, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if billing.Status == models.MediaBillingStatusSettled {
		return billing, nil
	}

	quote := quoteMediaBillingSnapshot(request.ActualQuantity, billing)
	if quote.CostUSD <= 0 || quote.Credits <= 0 {
		return nil, fmt.Errorf(
			"%w: 实际计费结果为0",
			ErrMediaBillingPriceNotConfigured,
		)
	}

	normalized := *request
	normalized.IdempotencyKey = idempotencyKey
	normalized.ExternalTaskID = strings.TrimSpace(normalized.ExternalTaskID)

	settled, err := repository.SettleTokenMediaBilling(
		ctx,
		&models.TokenMediaSettleSnapshot{
			MediaBillingSettleRequest: normalized,
			ActualCostUSD:             quote.CostUSD,
			ActualCredits:             quote.Credits,
		},
	)
	if err != nil {
		return nil, err
	}

	mediaBillingLog.Info(
		"媒体积分结算完成",
		"idempotency_key", settled.IdempotencyKey,
		"node", settled.BillingNodeCode,
		"actual_quantity", settled.ActualQuantity,
		"actual_credits", settled.ActualCredits,
	)
	return settled, nil
}

// Release 释放失败、取消或超时任务的冻结积分。
func (service *MediaBillingService) Release(
	ctx context.Context,
	request *models.MediaBillingReleaseRequest,
) (*models.TokenMediaBilling, error) {
	if request == nil || strings.TrimSpace(request.IdempotencyKey) == "" {
		return nil, ErrMediaBillingInvalidRequest
	}

	status := strings.ToLower(strings.TrimSpace(request.Status))
	if status != models.MediaBillingStatusFailed &&
		status != models.MediaBillingStatusCancelled {
		return nil, ErrMediaBillingInvalidRequest
	}

	normalized := *request
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.Status = status
	normalized.ExternalTaskID = strings.TrimSpace(normalized.ExternalTaskID)
	normalized.FailureReason = truncateMediaBillingReason(normalized.FailureReason)

	released, err := repository.ReleaseTokenMediaBilling(
		ctx,
		&models.TokenMediaReleaseSnapshot{
			MediaBillingReleaseRequest: normalized,
		},
	)
	if err != nil {
		return nil, err
	}

	mediaBillingLog.Info(
		"媒体冻结积分已释放",
		"idempotency_key", released.IdempotencyKey,
		"status", released.Status,
		"reserved_credits", released.ReservedCredits,
	)
	return released, nil
}

func normalizeMediaBillingReserveRequest(
	request *models.MediaBillingReserveRequest,
) (*models.MediaBillingReserveRequest, error) {
	if request == nil {
		return nil, ErrMediaBillingInvalidRequest
	}

	normalized := *request
	normalized.UserID = strings.TrimSpace(normalized.UserID)
	normalized.BillingCategory = strings.ToLower(strings.TrimSpace(normalized.BillingCategory))
	normalized.BillingNodeCode = strings.TrimSpace(normalized.BillingNodeCode)
	normalized.SceneCode = strings.TrimSpace(normalized.SceneCode)
	normalized.MediaType = strings.ToLower(strings.TrimSpace(normalized.MediaType))
	normalized.Provider = strings.ToLower(strings.TrimSpace(normalized.Provider))
	normalized.ModelName = strings.TrimSpace(normalized.ModelName)
	normalized.Variant = strings.TrimSpace(normalized.Variant)
	normalized.MediaUnit = strings.ToLower(strings.TrimSpace(normalized.MediaUnit))
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.ExternalTaskID = strings.TrimSpace(normalized.ExternalTaskID)

	if normalized.Variant == "" {
		normalized.Variant = "default"
	}
	if normalized.BillingCategory == "" {
		normalized.BillingCategory = normalized.MediaType
	}
	if normalized.UserID == "" ||
		normalized.BillingNodeCode == "" ||
		normalized.MediaType == "" ||
		normalized.Provider == "" ||
		normalized.ModelName == "" ||
		normalized.MediaUnit == "" ||
		normalized.IdempotencyKey == "" ||
		normalized.EstimatedQuantity <= 0 ||
		normalized.BillingCategory != normalized.MediaType {
		return nil, ErrMediaBillingInvalidRequest
	}

	normalized.SchoolID = normalizeMediaBillingStringPointer(normalized.SchoolID)
	normalized.CoursewareID = normalizeMediaBillingStringPointer(normalized.CoursewareID)
	normalized.PageID = normalizeMediaBillingStringPointer(normalized.PageID)
	normalized.AssetID = normalizeMediaBillingStringPointer(normalized.AssetID)
	return &normalized, nil
}

func quoteMediaBilling(
	quantity float64,
	price *models.TokenMediaPrice,
	policy *models.CreditPolicy,
) models.MediaBillingQuote {
	if price == nil || policy == nil {
		return models.MediaBillingQuote{}
	}

	billableQuantity := math.Max(quantity, price.MinimumQuantity)
	costUSD := math.Max(
		billableQuantity*price.UnitCostUSD,
		price.MinimumCostUSD,
	)
	return models.MediaBillingQuote{
		BillableQuantity: billableQuantity,
		UnitCostUSD:      price.UnitCostUSD,
		MinimumQuantity:  price.MinimumQuantity,
		MinimumCostUSD:   price.MinimumCostUSD,
		CostUSD:          costUSD,
		ExchangeRate:     policy.ExchangeRate,
		Multiplier:       policy.Multiplier,
		Credits: ceilMediaBillingCredits(
			costUSD * policy.ExchangeRate * policy.Multiplier,
		),
	}
}

func quoteMediaBillingSnapshot(
	quantity float64,
	billing *models.TokenMediaBilling,
) models.MediaBillingQuote {
	if billing == nil {
		return models.MediaBillingQuote{}
	}

	billableQuantity := math.Max(quantity, billing.MinimumQuantity)
	costUSD := math.Max(
		billableQuantity*billing.UnitCostUSD,
		billing.MinimumCostUSD,
	)
	return models.MediaBillingQuote{
		BillableQuantity: billableQuantity,
		UnitCostUSD:      billing.UnitCostUSD,
		MinimumQuantity:  billing.MinimumQuantity,
		MinimumCostUSD:   billing.MinimumCostUSD,
		CostUSD:          costUSD,
		ExchangeRate:     billing.ExchangeRate,
		Multiplier:       billing.Multiplier,
		Credits: ceilMediaBillingCredits(
			costUSD * billing.ExchangeRate * billing.Multiplier,
		),
	}
}

func ceilMediaBillingCredits(value float64) float64 {
	if value <= 0 {
		return 0
	}
	return math.Ceil(value*10000) / 10000
}

func normalizeMediaBillingStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func truncateMediaBillingReason(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= 500 {
		return value
	}
	return string(runes[:500])
}
