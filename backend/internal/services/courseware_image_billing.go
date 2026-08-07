package services

// courseware_image_billing.go — 课件图片统一积分预留与资产落库后结算
//
// 所有图片业务遵循同一边界：
//   1. 调用图片供应商前预留一张图片的预计积分；
//   2. 供应商明确失败时释放预留；
//   3. 供应商成功后即确认真实成本，下载或资产落库失败仍按一张图片结算；
//   4. 课程资产成功持久化后先绑定asset_id，再结算；
//   5. 同一幂等键重复请求不得再次调用供应商；
//   6. 结算数据库异常时保留reserved，供后续重试补结算；
//   7. 资产后的IAOCI回绑、预览状态或漫画状态更新不参与费用释放判断。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
)

const (
	coursewareImageBillingProvider  = "volcengine"
	coursewareImageBillingVariant   = "default"
	coursewareImageBillingDBTimeout = 8 * time.Second
)

var (
	ErrCoursewareImageBillingInProgress       = errors.New("图片生成任务正在处理中")
	ErrCoursewareImageBillingTerminal         = errors.New("图片生成计费任务已经失败或取消")
	ErrCoursewareImageBillingAssetLost        = errors.New("图片计费记录关联资产不存在")
	ErrCoursewareImageBillingOutputMissing    = errors.New("图片调用已结算但业务资产未形成")
	ErrCoursewareImageBillingIdentityMismatch = errors.New("图片计费幂等身份不匹配")
)

var coursewareImageBillingLog = logger.WithModule("services.courseware_image_billing")

// coursewareImageBillingInput 描述一次图片业务调用的不可变计费身份。
type coursewareImageBillingInput struct {
	UserID          string
	SchoolID        *string
	BillingNodeCode string
	CoursewareID    string
	PageID          *string
	ModelName       string
	IdempotencyKey  string
	Metadata        map[string]interface{}
}

// coursewareImageBillingSession 保存一次预留的进程内终态保护。
type coursewareImageBillingSession struct {
	service        *MediaBillingService
	idempotencyKey string
	startedAt      time.Time

	mu               sync.Mutex
	terminal         bool
	preserveReserved bool
}

// coursewareImageGenerateFunc 执行一次真实图片供应商调用。
type coursewareImageGenerateFunc func() (*ai.ImageGenerateResult, error)

// coursewareImagePersistFunc 下载图片并创建课程资产。
type coursewareImagePersistFunc func(
	result *ai.ImageGenerateResult,
) (*models.CoursewareAsset, error)

// executeBilledCoursewareImage 执行“预留→生图→资产持久化→绑定→结算”。
//
// 同一幂等键重放时：
//   - reserved且无asset_id：返回处理中，禁止再次调用供应商；
//   - reserved且已有asset_id：直接补结算并返回原资产；
//   - settled：直接返回原资产；
//   - failed/cancelled：返回终态错误。
func executeBilledCoursewareImage(
	ctx context.Context,
	input *coursewareImageBillingInput,
	generate coursewareImageGenerateFunc,
	persist coursewareImagePersistFunc,
) (
	*ai.ImageGenerateResult,
	*models.CoursewareAsset,
	error,
) {
	normalized, err := normalizeCoursewareImageBillingInput(input)
	if err != nil {
		return nil, nil, err
	}
	if generate == nil || persist == nil {
		return nil, nil, ErrMediaBillingInvalidRequest
	}

	service := NewMediaBillingService()

	billing, err := service.Reserve(
		ctx,
		&models.MediaBillingReserveRequest{
			UserID:            normalized.UserID,
			SchoolID:          normalized.SchoolID,
			BillingCategory:   models.BillingCategoryImage,
			BillingNodeCode:   normalized.BillingNodeCode,
			MediaType:         models.MediaTypeImage,
			Provider:          coursewareImageBillingProvider,
			ModelName:         normalized.ModelName,
			Variant:           coursewareImageBillingVariant,
			MediaUnit:         models.MediaUnitImage,
			EstimatedQuantity: 1,
			IdempotencyKey:    normalized.IdempotencyKey,
			CoursewareID:      coursewareImageStringPointer(normalized.CoursewareID),
			PageID:            normalized.PageID,
			Metadata:          normalized.Metadata,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if billing == nil {
		return nil, nil, ErrMediaBillingInvalidRequest
	}

	session := &coursewareImageBillingSession{
		service:        service,
		idempotencyKey: billing.IdempotencyKey,
		startedAt:      time.Now(),
	}

	if !billing.ReservationCreated {
		return recoverBilledCoursewareImage(
			ctx,
			billing,
			session,
			normalized,
		)
	}

	releaseReason := "courseware_image_operation_failed"
	defer func() {
		session.releasePending(
			releaseReason,
			map[string]interface{}{
				"billing_node_code": normalized.BillingNodeCode,
				"courseware_id":     normalized.CoursewareID,
			},
		)
	}()

	result, err := generate()
	if err != nil {
		releaseReason = "courseware_image_provider_failed"
		return nil, nil, err
	}

	// 供应商已成功返回，从此真实成本已经发生。
	// 后续即使没有URL、下载失败或资产落库失败，也不能释放预留。
	session.preserve()

	generatedURLCount := 0
	modelUsed := ""
	if result != nil {
		generatedURLCount = len(result.URLs)
		modelUsed = result.ModelUsed
	}

	if result == nil || len(result.URLs) == 0 {
		settleErr := session.settleProviderUsage(
			nil,
			map[string]interface{}{
				"provider_succeeded":  true,
				"business_result":     "provider_result_empty",
				"generated_url_count": generatedURLCount,
				"model_used":          modelUsed,
			},
		)
		if settleErr != nil {
			coursewareImageBillingLog.Error(
				"图片供应商已成功但空结果结算失败，预留保持待补偿",
				"idempotency_key", session.idempotencyKey,
				"billing_node_code", normalized.BillingNodeCode,
				"courseware_id", normalized.CoursewareID,
				"error", settleErr,
			)
		}

		return result, nil, fmt.Errorf("图片模型未返回有效图片")
	}

	asset, err := persist(result)
	if err != nil {
		settleErr := session.settleProviderUsage(
			nil,
			map[string]interface{}{
				"provider_succeeded":  true,
				"business_result":     "asset_persist_failed",
				"generated_url_count": len(result.URLs),
				"model_used":          result.ModelUsed,
				"persist_error":       truncateMediaBillingReason(err.Error()),
			},
		)
		if settleErr != nil {
			coursewareImageBillingLog.Error(
				"图片供应商已成功但资产持久化失败后的积分结算失败，预留保持待补偿",
				"idempotency_key", session.idempotencyKey,
				"billing_node_code", normalized.BillingNodeCode,
				"courseware_id", normalized.CoursewareID,
				"error", settleErr,
			)
		}

		return result, nil, err
	}
	if asset == nil || strings.TrimSpace(asset.ID) == "" {
		invalidErr := fmt.Errorf("图片资产持久化结果无效")

		settleErr := session.settleProviderUsage(
			nil,
			map[string]interface{}{
				"provider_succeeded":  true,
				"business_result":     "asset_result_invalid",
				"generated_url_count": len(result.URLs),
				"model_used":          result.ModelUsed,
			},
		)
		if settleErr != nil {
			coursewareImageBillingLog.Error(
				"图片供应商已成功但资产结果无效后的积分结算失败，预留保持待补偿",
				"idempotency_key", session.idempotencyKey,
				"billing_node_code", normalized.BillingNodeCode,
				"courseware_id", normalized.CoursewareID,
				"error", settleErr,
			)
		}

		return result, nil, invalidErr
	}

	bindErr := session.bindAsset(
		asset,
		map[string]interface{}{
			"asset_id":            asset.ID,
			"generated_url_count": len(result.URLs),
			"model_used":          result.ModelUsed,
		},
	)
	if bindErr != nil {
		coursewareImageBillingLog.Error(
			"图片资产已持久化但计费资产绑定失败，将继续通过结算事务绑定",
			"idempotency_key", session.idempotencyKey,
			"billing_node_code", normalized.BillingNodeCode,
			"courseware_id", normalized.CoursewareID,
			"asset_id", asset.ID,
			"error", bindErr,
		)
	}

	settleErr := session.settleProviderUsage(
		asset,
		map[string]interface{}{
			"asset_id":            asset.ID,
			"asset_bind_failed":   bindErr != nil,
			"generated_url_count": len(result.URLs),
			"model_used":          result.ModelUsed,
		},
	)
	if settleErr != nil {
		coursewareImageBillingLog.Error(
			"图片资产已持久化但积分结算失败，预留保持待补偿",
			"idempotency_key", session.idempotencyKey,
			"billing_node_code", normalized.BillingNodeCode,
			"courseware_id", normalized.CoursewareID,
			"asset_id", asset.ID,
			"error", settleErr,
		)
	}

	return result, asset, nil
}

// preserve 标记供应商成本及业务资产已经发生，禁止失败释放。
func (session *coursewareImageBillingSession) preserve() {
	if session == nil {
		return
	}

	session.mu.Lock()
	session.preserveReserved = true
	session.mu.Unlock()
}

// bindAsset 把已持久化的课程资产绑定到预留记录。
func (session *coursewareImageBillingSession) bindAsset(
	asset *models.CoursewareAsset,
	metadata map[string]interface{},
) error {
	if session == nil || session.service == nil || asset == nil {
		return ErrMediaBillingInvalidRequest
	}

	assetID := strings.TrimSpace(asset.ID)
	if assetID == "" {
		return ErrMediaBillingInvalidRequest
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		coursewareImageBillingDBTimeout,
	)
	defer cancel()

	_, err := session.service.BindAsset(
		ctx,
		&models.MediaBillingBindAssetRequest{
			IdempotencyKey: session.idempotencyKey,
			AssetID:        assetID,
			Metadata:       metadata,
		},
	)
	return err
}

// settleProviderUsage 按一次真实图片供应商成功调用结算。
//
// asset可以为空：供应商已经成功但下载、落盘或业务资产创建失败时，
// 仍需扣减真实成本，并在统一消费流水中保留失败原因。
func (session *coursewareImageBillingSession) settleProviderUsage(
	asset *models.CoursewareAsset,
	metadata map[string]interface{},
) error {
	if session == nil || session.service == nil {
		return ErrMediaBillingInvalidRequest
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.terminal {
		return nil
	}

	session.preserveReserved = true

	var assetID *string
	if asset != nil {
		normalizedAssetID := strings.TrimSpace(asset.ID)
		if normalizedAssetID == "" {
			return ErrMediaBillingInvalidRequest
		}
		assetID = &normalizedAssetID
	}

	latencyMS := int(time.Since(session.startedAt).Milliseconds())
	if latencyMS < 0 {
		latencyMS = 0
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		coursewareImageBillingDBTimeout,
	)
	defer cancel()

	_, err := session.service.Settle(
		ctx,
		&models.MediaBillingSettleRequest{
			IdempotencyKey: session.idempotencyKey,
			ActualQuantity: 1,
			AssetID:        assetID,
			LatencyMs:      latencyMS,
			Metadata:       metadata,
		},
	)
	if err != nil {
		return err
	}

	session.terminal = true
	session.preserveReserved = false
	return nil
}

// releasePending 释放资产持久化前失败操作的预留积分。
func (session *coursewareImageBillingSession) releasePending(
	reason string,
	metadata map[string]interface{},
) {
	if session == nil || session.service == nil {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.terminal || session.preserveReserved {
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		coursewareImageBillingDBTimeout,
	)
	defer cancel()

	_, err := session.service.Release(
		ctx,
		&models.MediaBillingReleaseRequest{
			IdempotencyKey: session.idempotencyKey,
			Status:         models.MediaBillingStatusFailed,
			FailureReason:  reason,
			Metadata:       metadata,
		},
	)
	if err != nil {
		coursewareImageBillingLog.Error(
			"释放图片媒体积分预留失败",
			"idempotency_key", session.idempotencyKey,
			"reason", reason,
			"error", err,
		)
		return
	}

	session.terminal = true
}

// normalizeCoursewareImageBillingInput 标准化图片计费身份。
func normalizeCoursewareImageBillingInput(
	input *coursewareImageBillingInput,
) (*coursewareImageBillingInput, error) {
	if input == nil {
		return nil, ErrMediaBillingInvalidRequest
	}

	normalized := *input
	normalized.UserID = strings.TrimSpace(normalized.UserID)
	normalized.BillingNodeCode = strings.TrimSpace(normalized.BillingNodeCode)
	normalized.CoursewareID = strings.TrimSpace(normalized.CoursewareID)
	normalized.ModelName = strings.TrimSpace(normalized.ModelName)
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.SchoolID = normalizeMediaBillingStringPointer(normalized.SchoolID)
	normalized.PageID = normalizeMediaBillingStringPointer(normalized.PageID)

	if normalized.IdempotencyKey == "" {
		normalized.IdempotencyKey = fmt.Sprintf(
			"courseware-image:%s:%s",
			normalized.BillingNodeCode,
			uuid.NewString(),
		)
	}

	if normalized.UserID == "" ||
		normalized.BillingNodeCode == "" ||
		normalized.CoursewareID == "" ||
		normalized.ModelName == "" {
		return nil, ErrMediaBillingInvalidRequest
	}

	metadata := map[string]interface{}{}
	for key, value := range normalized.Metadata {
		metadata[key] = value
	}
	metadata["billing_node_code"] = normalized.BillingNodeCode
	metadata["courseware_id"] = normalized.CoursewareID
	normalized.Metadata = metadata

	return &normalized, nil
}

func coursewareImageStringPointer(value string) *string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}
	return &normalized
}
