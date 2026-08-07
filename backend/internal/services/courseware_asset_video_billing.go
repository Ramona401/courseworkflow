package services

// courseware_asset_video_billing.go — 异步视频积分预留、恢复、结算与释放
//
// 状态机：
//   1. 提交供应商前按预计provider_token冻结积分；
//   2. 供应商明确拒绝时释放预留；
//   3. 提交结果不确定时保持reserved，禁止自动重复提交；
//   4. 供应商返回task_id后绑定外部任务；
//   5. 创建generating视频资产并绑定asset_id；
//   6. running保持reserved；
//   7. succeeded按usage.total_tokens结算；
//   8. failed释放预留；
//   9. 浏览器重试或刷新时通过operation_id、计费记录和资产恢复原任务。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	coursewareVideoBillingNodeCode = "courseware_video_generate"
	coursewareVideoBillingProvider = "volcengine"
	coursewareVideoBillingVariant  = "silent"

	// 固定生成5秒720p无声视频。
	// 预留量包含安全余量，最终按供应商真实total_tokens结算。
	coursewareVideoEstimatedProviderTokens = 130000.0

	coursewareVideoBillingDBTimeout = 8 * time.Second
)

var (
	ErrCoursewareVideoBillingInProgress = errors.New(
		"视频生成任务正在处理中",
	)
	ErrCoursewareVideoBillingTerminal = errors.New(
		"视频生成计费任务已经终态",
	)
	ErrCoursewareVideoBillingIdentityMismatch = errors.New(
		"视频生成计费身份不匹配",
	)
	ErrCoursewareVideoBillingOutputMissing = errors.New(
		"视频任务已提交但业务资产不存在",
	)
)

// coursewareVideoBillingIdentity 保存一次视频操作的不可变身份。
type coursewareVideoBillingIdentity struct {
	UserID             string
	SchoolID           *string
	CoursewareID       string
	PageID             string
	PageNumber         int
	ModelName          string
	OperationID        string
	IdempotencyKey     string
	RequestFingerprint string
	SourceFrameAssetID string
	HasReferenceImage  bool
}

func newCoursewareVideoBillingIdentity(
	userID string,
	schoolID string,
	coursewareID string,
	pageID string,
	pageNumber int,
	modelName string,
	operationID string,
	prompt string,
	refURL string,
	sourceFrameAssetID string,
) (*coursewareVideoBillingIdentity, error) {
	userID = strings.TrimSpace(userID)
	coursewareID = strings.TrimSpace(coursewareID)
	pageID = strings.TrimSpace(pageID)
	modelName = strings.TrimSpace(modelName)
	operationID = strings.TrimSpace(operationID)
	sourceFrameAssetID = strings.TrimSpace(sourceFrameAssetID)
	refURL = strings.TrimSpace(refURL)

	if _, err := uuid.Parse(operationID); err != nil {
		return nil, ErrMediaBillingInvalidRequest
	}

	if userID == "" ||
		coursewareID == "" ||
		pageID == "" ||
		pageNumber <= 0 ||
		modelName == "" ||
		strings.TrimSpace(prompt) == "" {
		return nil, ErrMediaBillingInvalidRequest
	}

	return &coursewareVideoBillingIdentity{
		UserID:       userID,
		SchoolID:     schoolIDPtr(strings.TrimSpace(schoolID)),
		CoursewareID: coursewareID,
		PageID:       pageID,
		PageNumber:   pageNumber,
		ModelName:    modelName,
		OperationID:  operationID,
		IdempotencyKey: "courseware-video:" +
			operationID,
		RequestFingerprint: coursewareVideoRequestFingerprint(
			coursewareID,
			pageID,
			coursewareVideoBillingProvider,
			modelName,
			coursewareVideoBillingVariant,
			models.MediaUnitProviderToken,
			"generate_audio=false",
			prompt,
			refURL,
			sourceFrameAssetID,
		),
		SourceFrameAssetID: sourceFrameAssetID,
		HasReferenceImage:  refURL != "",
	}, nil
}

func reserveCoursewareVideoBilling(
	ctx context.Context,
	identity *coursewareVideoBillingIdentity,
) (*models.TokenMediaBilling, error) {
	if identity == nil {
		return nil, ErrMediaBillingInvalidRequest
	}

	return NewMediaBillingService().Reserve(
		ctx,
		&models.MediaBillingReserveRequest{
			UserID:            identity.UserID,
			SchoolID:          identity.SchoolID,
			BillingCategory:   models.BillingCategoryVideo,
			BillingNodeCode:   coursewareVideoBillingNodeCode,
			MediaType:         models.MediaTypeVideo,
			Provider:          coursewareVideoBillingProvider,
			ModelName:         identity.ModelName,
			Variant:           coursewareVideoBillingVariant,
			MediaUnit:         models.MediaUnitProviderToken,
			EstimatedQuantity: coursewareVideoEstimatedProviderTokens,
			IdempotencyKey:    identity.IdempotencyKey,
			CoursewareID: coursewareVideoStringPointer(
				identity.CoursewareID,
			),
			PageID: coursewareVideoStringPointer(
				identity.PageID,
			),
			Metadata: coursewareVideoBillingMetadata(
				identity,
				"",
			),
		},
	)
}

// recoverCoursewareVideoSubmission 恢复同一个operation_id对应的原任务。
func (s *CoursewareAssetService) recoverCoursewareVideoSubmission(
	billing *models.TokenMediaBilling,
	identity *coursewareVideoBillingIdentity,
	req *GenerateVideoServiceRequest,
	page *models.CoursewarePage,
) (*GenerateVideoServiceResponse, error) {
	if err := validateCoursewareVideoBillingIdentity(
		billing,
		identity,
	); err != nil {
		return nil, err
	}

	switch billing.Status {
	case models.MediaBillingStatusFailed,
		models.MediaBillingStatusCancelled:
		return nil, ErrCoursewareVideoBillingTerminal

	case models.MediaBillingStatusReserved,
		models.MediaBillingStatusSettled:

	default:
		return nil, ErrCoursewareVideoBillingTerminal
	}

	taskID := strings.TrimSpace(
		billing.ExternalTaskID,
	)

	var asset *models.CoursewareAsset

	if billing.AssetID != nil &&
		strings.TrimSpace(*billing.AssetID) != "" {
		loaded, err := repository.GetCWAssetByID(
			context.Background(),
			strings.TrimSpace(*billing.AssetID),
		)
		if err != nil {
			return nil,
				ErrCoursewareVideoBillingOutputMissing
		}

		if loaded.CoursewareID !=
			identity.CoursewareID ||
			loaded.AssetType !=
				models.CWAssetTypeVideo {
			return nil,
				ErrCoursewareVideoBillingIdentityMismatch
		}

		asset = loaded

		if taskID == "" {
			taskID =
				resolveCoursewareVideoTaskID(
					billing,
					asset,
				)
		}
	}

	if taskID == "" {
		if billing.Status ==
			models.MediaBillingStatusSettled {
			return nil,
				ErrCoursewareVideoBillingOutputMissing
		}

		return nil,
			ErrCoursewareVideoBillingInProgress
	}

	if asset == nil {
		if billing.Status ==
			models.MediaBillingStatusSettled {
			return nil,
				ErrCoursewareVideoBillingOutputMissing
		}

		created, err :=
			s.createGeneratingVideoAsset(
				req,
				page,
				identity,
				taskID,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"恢复视频资产失败: %w",
				err,
			)
		}

		asset = created
	}

	if billing.Status ==
		models.MediaBillingStatusReserved {
		service :=
			NewMediaBillingService()

		if err :=
			bindCoursewareVideoExternalTask(
				service,
				identity.IdempotencyKey,
				taskID,
			); err != nil {
			cwAssetLog.Warn(
				"恢复视频时绑定供应商任务失败",
				"asset_id",
				asset.ID,
				"task_id",
				taskID,
				"error",
				err,
			)
		}

		if err :=
			bindCoursewareVideoAsset(
				service,
				identity.IdempotencyKey,
				asset.ID,
			); err != nil {
			cwAssetLog.Warn(
				"恢复视频时绑定资产失败",
				"asset_id",
				asset.ID,
				"task_id",
				taskID,
				"error",
				err,
			)
		}
	}

	return &GenerateVideoServiceResponse{
		AssetID:   asset.ID,
		TaskID:    taskID,
		ModelUsed: billing.ModelName,
		Message:   "已恢复原视频生成任务，请继续等待生成结果",
	}, nil
}

func (s *CoursewareAssetService) createGeneratingVideoAsset(
	req *GenerateVideoServiceRequest,
	page *models.CoursewarePage,
	identity *coursewareVideoBillingIdentity,
	taskID string,
) (*models.CoursewareAsset, error) {
	if req == nil ||
		page == nil ||
		identity == nil ||
		strings.TrimSpace(taskID) == "" {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	metadata :=
		coursewareVideoBillingMetadata(
			identity,
			taskID,
		)

	metadata["billing_idempotency_key"] =
		identity.IdempotencyKey

	asset :=
		&models.CoursewareAsset{
			CoursewareID:     identity.CoursewareID,
			PageID:           &page.ID,
			PlaceholderID:    strings.TrimSpace(taskID),
			AssetType:        models.CWAssetTypeVideo,
			GenerationPrompt: strings.TrimSpace(req.Prompt),
			OssURL:           "",
			FileSize:         0,
			MimeType:         "video/mp4",
			Metadata: coursewareVideoMetadataString(
				metadata,
			),
			Status: models.CWAssetStatusGenerating,
		}

	createCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareVideoBillingDBTimeout,
		)
	defer cancel()

	if err :=
		repository.CreateCWAsset(
			createCtx,
			asset,
		); err != nil {
		return nil, err
	}

	return asset, nil
}

func bindCoursewareVideoExternalTask(
	service *MediaBillingService,
	idempotencyKey string,
	taskID string,
) error {
	if service == nil {
		return ErrMediaBillingInvalidRequest
	}

	bindCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareVideoBillingDBTimeout,
		)
	defer cancel()

	_, err :=
		service.BindExternalTask(
			bindCtx,
			&models.MediaBillingBindTaskRequest{
				IdempotencyKey: strings.TrimSpace(
					idempotencyKey,
				),
				ExternalTaskID: strings.TrimSpace(
					taskID,
				),
				Metadata: map[string]interface{}{
					"external_task_bound": true,
				},
			},
		)

	return err
}

func bindCoursewareVideoAsset(
	service *MediaBillingService,
	idempotencyKey string,
	assetID string,
) error {
	if service == nil {
		return ErrMediaBillingInvalidRequest
	}

	bindCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareVideoBillingDBTimeout,
		)
	defer cancel()

	_, err :=
		service.BindAsset(
			bindCtx,
			&models.MediaBillingBindAssetRequest{
				IdempotencyKey: strings.TrimSpace(
					idempotencyKey,
				),
				AssetID: strings.TrimSpace(
					assetID,
				),
				Metadata: map[string]interface{}{
					"asset_id": strings.TrimSpace(
						assetID,
					),
					"asset_bound": true,
				},
			},
		)

	return err
}

func loadCoursewareVideoBilling(
	ctx context.Context,
	asset *models.CoursewareAsset,
) (*models.TokenMediaBilling, error) {
	if asset == nil {
		return nil,
			ErrMediaBillingInvalidRequest
	}

	metadata :=
		coursewareVideoMetadataMap(
			asset.Metadata,
		)

	idempotencyKey :=
		coursewareVideoMetadataText(
			metadata,
			"billing_idempotency_key",
		)

	// 上线统一视频计费前的旧资产没有此字段。
	if idempotencyKey == "" {
		return nil, nil
	}

	billing, err :=
		repository.GetTokenMediaBillingByKey(
			ctx,
			idempotencyKey,
		)
	if err != nil {
		return nil, err
	}

	billingCoursewareID := ""

	if billing.CoursewareID != nil {
		billingCoursewareID =
			strings.TrimSpace(
				*billing.CoursewareID,
			)
	}

	if billingCoursewareID !=
		asset.CoursewareID ||
		strings.TrimSpace(
			billing.MediaType,
		) != models.MediaTypeVideo {
		return nil,
			ErrCoursewareVideoBillingIdentityMismatch
	}

	return billing, nil
}

func resolveCoursewareVideoTaskID(
	billing *models.TokenMediaBilling,
	asset *models.CoursewareAsset,
) string {
	if billing != nil &&
		strings.TrimSpace(
			billing.ExternalTaskID,
		) != "" {
		return strings.TrimSpace(
			billing.ExternalTaskID,
		)
	}

	if asset == nil {
		return ""
	}

	metadata :=
		coursewareVideoMetadataMap(
			asset.Metadata,
		)

	if value :=
		coursewareVideoMetadataText(
			metadata,
			"external_task_id",
		); value != "" {
		return value
	}

	return strings.TrimSpace(
		asset.PlaceholderID,
	)
}

func updateCoursewareVideoAssetResultMetadata(
	asset *models.CoursewareAsset,
	result *ai.VideoQueryResult,
) error {
	if asset == nil ||
		result == nil {
		return ErrMediaBillingInvalidRequest
	}

	metadata :=
		coursewareVideoMetadataMap(
			asset.Metadata,
		)

	metadata["provider_status"] =
		result.Status
	metadata["provider_total_tokens"] =
		result.TotalTokens
	metadata["video_duration"] =
		result.Duration
	metadata["video_resolution"] =
		result.Resolution
	metadata["video_ratio"] =
		result.Ratio
	metadata["video_fps"] =
		result.FPS
	metadata["external_task_id"] =
		result.TaskID

	content :=
		coursewareVideoMetadataString(
			metadata,
		)

	updateCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareVideoBillingDBTimeout,
		)
	defer cancel()

	if err :=
		repository.UpdateCWAssetMetadata(
			updateCtx,
			asset.ID,
			content,
		); err != nil {
		return err
	}

	asset.Metadata = content

	return nil
}

func settleCoursewareVideoBilling(
	billing *models.TokenMediaBilling,
	asset *models.CoursewareAsset,
	result *ai.VideoQueryResult,
) error {
	if billing == nil ||
		asset == nil ||
		result == nil {
		return ErrMediaBillingInvalidRequest
	}

	if billing.Status ==
		models.MediaBillingStatusSettled {
		return nil
	}

	if billing.Status !=
		models.MediaBillingStatusReserved {
		return ErrCoursewareVideoBillingTerminal
	}

	if result.TotalTokens <= 0 {
		return fmt.Errorf(
			"视频供应商未返回有效token用量",
		)
	}

	service :=
		NewMediaBillingService()

	_ =
		bindCoursewareVideoExternalTask(
			service,
			billing.IdempotencyKey,
			result.TaskID,
		)

	_ =
		bindCoursewareVideoAsset(
			service,
			billing.IdempotencyKey,
			asset.ID,
		)

	assetID :=
		strings.TrimSpace(
			asset.ID,
		)

	latencyMS :=
		int(
			time.Since(
				billing.CreatedAt,
			).Milliseconds(),
		)

	if latencyMS < 0 {
		latencyMS = 0
	}

	settleCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareVideoBillingDBTimeout,
		)
	defer cancel()

	_, err :=
		service.Settle(
			settleCtx,
			&models.MediaBillingSettleRequest{
				IdempotencyKey: billing.IdempotencyKey,
				ActualQuantity: float64(
					result.TotalTokens,
				),
				AssetID:        &assetID,
				ExternalTaskID: result.TaskID,
				LatencyMs:      latencyMS,
				Metadata: map[string]interface{}{
					"provider_succeeded":    true,
					"provider_total_tokens": result.TotalTokens,
					"video_duration":        result.Duration,
					"video_resolution":      result.Resolution,
					"video_ratio":           result.Ratio,
					"video_fps":             result.FPS,
				},
			},
		)

	return err
}

func settleCoursewareVideoBillingFromAsset(
	billing *models.TokenMediaBilling,
	asset *models.CoursewareAsset,
) error {
	if billing == nil ||
		asset == nil {
		return ErrMediaBillingInvalidRequest
	}

	if billing.Status ==
		models.MediaBillingStatusSettled {
		return nil
	}

	metadata :=
		coursewareVideoMetadataMap(
			asset.Metadata,
		)

	totalTokens :=
		coursewareVideoMetadataInt(
			metadata,
			"provider_total_tokens",
		)

	if totalTokens <= 0 {
		return fmt.Errorf(
			"视频资产缺少供应商token用量",
		)
	}

	return settleCoursewareVideoBilling(
		billing,
		asset,
		&ai.VideoQueryResult{
			TaskID: resolveCoursewareVideoTaskID(
				billing,
				asset,
			),
			Status: "succeeded",
			Duration: coursewareVideoMetadataInt(
				metadata,
				"video_duration",
			),
			Resolution: coursewareVideoMetadataText(
				metadata,
				"video_resolution",
			),
			Ratio: coursewareVideoMetadataText(
				metadata,
				"video_ratio",
			),
			FPS: coursewareVideoMetadataInt(
				metadata,
				"video_fps",
			),
			TotalTokens: totalTokens,
		},
	)
}

func releaseCoursewareVideoBilling(
	billing *models.TokenMediaBilling,
	taskID string,
	reason string,
	metadata map[string]interface{},
) error {
	if billing == nil {
		return ErrMediaBillingInvalidRequest
	}

	if billing.Status ==
		models.MediaBillingStatusFailed ||
		billing.Status ==
			models.MediaBillingStatusCancelled {
		return nil
	}

	if billing.Status !=
		models.MediaBillingStatusReserved {
		return ErrCoursewareVideoBillingTerminal
	}

	releaseCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			coursewareVideoBillingDBTimeout,
		)
	defer cancel()

	_, err :=
		NewMediaBillingService().
			Release(
				releaseCtx,
				&models.MediaBillingReleaseRequest{
					IdempotencyKey: billing.IdempotencyKey,
					Status:         models.MediaBillingStatusFailed,
					FailureReason: truncateMediaBillingReason(
						reason,
					),
					ExternalTaskID: strings.TrimSpace(
						taskID,
					),
					Metadata: metadata,
				},
			)

	return err
}

func validateCoursewareVideoBillingIdentity(
	billing *models.TokenMediaBilling,
	identity *coursewareVideoBillingIdentity,
) error {
	if billing == nil ||
		identity == nil {
		return ErrMediaBillingInvalidRequest
	}

	billingCoursewareID := ""

	if billing.CoursewareID != nil {
		billingCoursewareID =
			strings.TrimSpace(
				*billing.CoursewareID,
			)
	}

	billingPageID := ""

	if billing.PageID != nil {
		billingPageID =
			strings.TrimSpace(
				*billing.PageID,
			)
	}

	metadata :=
		coursewareVideoBillingRawMetadata(
			billing.Metadata,
		)

	if strings.TrimSpace(
		billing.UserID,
	) != identity.UserID ||
		strings.TrimSpace(
			billing.BillingNodeCode,
		) != coursewareVideoBillingNodeCode ||
		strings.TrimSpace(
			billing.MediaType,
		) != models.MediaTypeVideo ||
		strings.TrimSpace(
			billing.Provider,
		) != coursewareVideoBillingProvider ||
		strings.TrimSpace(
			billing.ModelName,
		) != identity.ModelName ||
		strings.TrimSpace(
			billing.Variant,
		) != coursewareVideoBillingVariant ||
		strings.TrimSpace(
			billing.MediaUnit,
		) != models.MediaUnitProviderToken ||
		billingCoursewareID !=
			identity.CoursewareID ||
		billingPageID !=
			identity.PageID ||
		coursewareVideoMetadataText(
			metadata,
			"request_fingerprint",
		) != identity.RequestFingerprint {
		return ErrCoursewareVideoBillingIdentityMismatch
	}

	return nil
}

func coursewareVideoRequestFingerprint(
	values ...string,
) string {
	normalized :=
		make(
			[]string,
			0,
			len(values),
		)

	for _, value := range values {
		normalized =
			append(
				normalized,
				strings.TrimSpace(
					value,
				),
			)
	}

	sum :=
		sha256.Sum256(
			[]byte(
				strings.Join(
					normalized,
					"\x1f",
				),
			),
		)

	return hex.EncodeToString(
		sum[:12],
	)
}

func coursewareVideoBillingMetadata(
	identity *coursewareVideoBillingIdentity,
	taskID string,
) map[string]interface{} {
	metadata :=
		map[string]interface{}{
			"video_operation_id":        identity.OperationID,
			"request_fingerprint":       identity.RequestFingerprint,
			"page_number":               identity.PageNumber,
			"source_frame_asset_id":     identity.SourceFrameAssetID,
			"has_reference_image":       identity.HasReferenceImage,
			"requested_resolution":      "720p",
			"requested_ratio":           "16:9",
			"requested_duration":        5,
			"generate_audio":            false,
			"billing_variant":           coursewareVideoBillingVariant,
			"estimated_provider_tokens": coursewareVideoEstimatedProviderTokens,
		}

	if strings.TrimSpace(
		taskID,
	) != "" {
		metadata["external_task_id"] =
			strings.TrimSpace(
				taskID,
			)
	}

	return metadata
}

func coursewareVideoStringPointer(
	value string,
) *string {
	normalized :=
		strings.TrimSpace(
			value,
		)

	if normalized == "" {
		return nil
	}

	return &normalized
}

func coursewareVideoMetadataMap(
	raw string,
) map[string]interface{} {
	result :=
		map[string]interface{}{}

	if strings.TrimSpace(
		raw,
	) != "" {
		_ =
			json.Unmarshal(
				[]byte(raw),
				&result,
			)
	}

	return result
}

func coursewareVideoBillingRawMetadata(
	raw json.RawMessage,
) map[string]interface{} {
	result :=
		map[string]interface{}{}

	if len(raw) > 0 {
		_ =
			json.Unmarshal(
				raw,
				&result,
			)
	}

	return result
}

func coursewareVideoMetadataString(
	metadata map[string]interface{},
) string {
	content, err :=
		json.Marshal(
			metadata,
		)

	if err != nil {
		return "{}"
	}

	return string(content)
}

func coursewareVideoMetadataText(
	metadata map[string]interface{},
	key string,
) string {
	value, exists :=
		metadata[key]

	if !exists {
		return ""
	}

	text, ok :=
		value.(string)

	if !ok {
		return ""
	}

	return strings.TrimSpace(
		text,
	)
}

func coursewareVideoMetadataInt(
	metadata map[string]interface{},
	key string,
) int {
	value, exists :=
		metadata[key]

	if !exists {
		return 0
	}

	switch typed :=
		value.(type) {
	case float64:
		return int(typed)

	case int:
		return typed

	case json.Number:
		parsed, _ :=
			typed.Int64()

		return int(parsed)

	default:
		return 0
	}
}
