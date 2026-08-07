package models

// media_billing.go — 图片、视频、TTS和ASR统一积分计费模型
//
// 本文件只定义媒体计费领域对象和内部命令，不承载数据库或供应商调用。
// 媒体调用采用“预留→结算/释放”状态机，并用idempotency_key保证一次业务操作
// 最多产生一条积分消费流水。

import (
	"encoding/json"
	"time"
)

const (
	BillingCategoryImage = "image"
	BillingCategoryVideo = "video"
	BillingCategoryTTS   = "tts"
	BillingCategoryASR   = "asr"
)

const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeTTS   = "tts"
	MediaTypeASR   = "asr"
)

const (
	MediaUnitImage         = "image"
	MediaUnitRequest       = "request"
	MediaUnitProviderToken = "provider_token"
	MediaUnitSecond        = "second"
	MediaUnitCharacter     = "character"
	MediaUnitAudioSecond   = "audio_second"
)

const (
	MediaBillingStatusReserved  = "reserved"
	MediaBillingStatusSettled   = "settled"
	MediaBillingStatusFailed    = "failed"
	MediaBillingStatusCancelled = "cancelled"
)

// TokenBillingNode 统一积分中心业务节点。
type TokenBillingNode struct {
	NodeCode    string
	Category    string
	DisplayName string
	SceneCode   string
	MediaType   string
	Description string
	SortOrder   int
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TokenMediaPrice 媒体模型单价及最低计费规则。
type TokenMediaPrice struct {
	ID              string
	MediaType       string
	Provider        string
	ModelName       string
	Variant         string
	MediaUnit       string
	UnitCostUSD     float64
	MinimumQuantity float64
	MinimumCostUSD  float64
	DisplayName     string
	IsActive        bool
	UpdatedBy       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TokenMediaBilling 媒体预留与结算记录。
type TokenMediaBilling struct {
	ID                string
	IdempotencyKey    string
	Status            string
	AccountID         string
	UserID            string
	SchoolID          *string
	BillingCategory   string
	BillingNodeCode   string
	SceneCode         string
	MediaType         string
	Provider          string
	ModelName         string
	Variant           string
	MediaUnit         string
	EstimatedQuantity float64
	ActualQuantity    float64
	UnitCostUSD       float64
	MinimumQuantity   float64
	MinimumCostUSD    float64
	EstimatedCostUSD  float64
	ActualCostUSD     float64
	ExchangeRate      float64
	Multiplier        float64
	ReservedCredits   float64
	ActualCredits     float64
	CoursewareID      *string
	PageID            *string
	AssetID           *string
	ExternalTaskID    string
	ConsumptionLogID  *string
	Metadata          json.RawMessage
	FailureReason     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	SettledAt         *time.Time

	// ReservationCreated 仅表示本次Reserve是否新建记录，不写入数据库。
	ReservationCreated bool `json:"-"`
}

// IsTerminal 判断媒体计费记录是否已经终态。
func (billing *TokenMediaBilling) IsTerminal() bool {
	if billing == nil {
		return false
	}

	return billing.Status == MediaBillingStatusSettled ||
		billing.Status == MediaBillingStatusFailed ||
		billing.Status == MediaBillingStatusCancelled
}

// MediaBillingReserveRequest 供应商调用前的预留请求。
type MediaBillingReserveRequest struct {
	UserID            string
	SchoolID          *string
	BillingCategory   string
	BillingNodeCode   string
	SceneCode         string
	MediaType         string
	Provider          string
	ModelName         string
	Variant           string
	MediaUnit         string
	EstimatedQuantity float64
	IdempotencyKey    string
	CoursewareID      *string
	PageID            *string
	AssetID           *string
	ExternalTaskID    string
	Metadata          map[string]interface{}
}

// MediaBillingAnnotateRequest 合并更新reserved媒体计费记录的补偿元数据。
type MediaBillingAnnotateRequest struct {
        IdempotencyKey string
        Metadata       map[string]interface{}
}

// MediaBillingSettleRequest 供应商成功且业务结果持久化后的结算请求。
type MediaBillingSettleRequest struct {
	IdempotencyKey string
	ActualQuantity float64
	AssetID        *string
	ExternalTaskID string
	LatencyMs      int
	Metadata       map[string]interface{}
}

// MediaBillingReleaseRequest 供应商失败、取消或超时后的释放请求。
type MediaBillingReleaseRequest struct {
	IdempotencyKey string
	Status         string
	FailureReason  string
	ExternalTaskID string
	Metadata       map[string]interface{}
}

// MediaBillingBindAssetRequest 业务资产持久化后的绑定请求。
type MediaBillingBindAssetRequest struct {
	IdempotencyKey string
	AssetID        string
	Metadata       map[string]interface{}
}

// MediaBillingBindTaskRequest 异步任务提交成功后的外部任务绑定请求。
type MediaBillingBindTaskRequest struct {
	IdempotencyKey string
	ExternalTaskID string
	Metadata       map[string]interface{}
}

// MediaBillingQuote 保存一次媒体调用的计费快照。
type MediaBillingQuote struct {
	BillableQuantity float64
	UnitCostUSD      float64
	MinimumQuantity  float64
	MinimumCostUSD   float64
	CostUSD          float64
	ExchangeRate     float64
	Multiplier       float64
	Credits          float64
}

// TokenMediaReserveSnapshot 仓储层执行预留所需的完整不可变快照。
type TokenMediaReserveSnapshot struct {
	MediaBillingReserveRequest
	Quote MediaBillingQuote
}

// TokenMediaSettleSnapshot 仓储层执行成功结算所需的最终快照。
type TokenMediaSettleSnapshot struct {
	MediaBillingSettleRequest
	ActualCostUSD float64
	ActualCredits float64
}

// TokenMediaReleaseSnapshot 仓储层执行失败释放所需的终态快照。
type TokenMediaReleaseSnapshot struct {
	MediaBillingReleaseRequest
}
