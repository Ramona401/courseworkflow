package models

// price_sync.go — 文本及多媒体模型价格自动同步领域协议。
//
// 本文件只定义同步目标、外部价格、预览批次、应用结果和管理配置；
// 不访问数据库，不调用供应商，不参与图片、视频或TTS业务结算。
//
// 安全边界：
//   - 文本价格按精确模型名匹配；
//   - 媒体价格沿用现有五元组身份；
//   - 预览与应用严格分离；
//   - 定时任务只处理AutoSyncEnabled=true的目标；
//   - 手动应用必须明确选择“全部”或指定明细ID，禁止空选择误应用全部。

import (
	"encoding/json"
	"time"
)

const (
	PriceSyncSourceMainGateway     = "main_gateway"
	PriceSyncSourceDomesticGateway = "domestic_gateway"
	PriceSyncSourceMediaGateway    = "media_gateway"
	PriceSyncSourceTTSGateway      = "tts_gateway"
)

const (
	PriceSyncTriggerManual    = "manual"
	PriceSyncTriggerScheduler = "scheduler"
)

const (
	PriceSyncRunPreviewed = "previewed"
	PriceSyncRunApplied   = "applied"
	PriceSyncRunFailed    = "failed"
)

const (
	PriceSyncTargetText  = "text"
	PriceSyncTargetMedia = "media"
)

const (
	PriceSyncActionUpdate    = "update"
	PriceSyncActionUnchanged = "unchanged"
	PriceSyncActionSkipped   = "skipped"
	PriceSyncActionApplied   = "applied"
	PriceSyncActionStale     = "stale"
)

// TextPriceSyncTarget 是同步服务内部使用的文本价格目标。
type TextPriceSyncTarget struct {
	ID              string
	ModelName       string
	Provider        string
	CostPer1kInput  float64
	CostPer1kOutput float64
	DisplayName     string
	IsActive        bool

	AutoSyncEnabled bool
	SyncSource      string
	SyncModelName   string
	LastSyncedAt    *time.Time
	LastSyncStatus  string
	LastSyncMessage string
}

// MediaPriceSyncTarget 是同步服务内部使用的媒体价格目标。
type MediaPriceSyncTarget struct {
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

	AutoSyncEnabled bool
	SyncSource      string
	SyncModelName   string
	LastSyncedAt    *time.Time
	LastSyncStatus  string
	LastSyncMessage string
}

// GatewayPrice 是价格来源返回的单个模型规范化记录。
type GatewayPrice struct {
	ModelName              string             `json:"model_name"`
	Provider               string             `json:"provider"`
	Description            string             `json:"description"`
	QuotaType              int                `json:"quota_type"`
	ModelRatio             float64            `json:"model_ratio"`
	CompletionRatio        float64            `json:"completion_ratio"`
	ModelPriceUSD          float64            `json:"model_price_usd"`
	EnableGroups           []string           `json:"enable_groups"`
	SupportedEndpointPaths []string           `json:"supported_endpoint_paths"`
	UnitPricesUSD          map[string]float64 `json:"unit_prices_usd"`
	RawPayload             json.RawMessage    `json:"raw_payload"`
}

// GatewayPriceCatalog 是一次价格接口读取结果。
type GatewayPriceCatalog struct {
	Source      string
	PricingURL  string
	GroupRatios map[string]float64
	Prices      []GatewayPrice
}

// PriceSyncRun 是一次同步预览或应用批次。
type PriceSyncRun struct {
	ID            string          `json:"id"`
	TriggerType   string          `json:"trigger_type"`
	Status        string          `json:"status"`
	SourceKind    string          `json:"source_kind"`
	SourceBaseURL string          `json:"source_base_url"`
	PreviewOnly   bool            `json:"preview_only"`
	Summary       json.RawMessage `json:"summary"`
	ErrorMessage  string          `json:"error_message"`
	CreatedBy     *string         `json:"created_by"`
	StartedAt     time.Time       `json:"started_at"`
	FinishedAt    *time.Time      `json:"finished_at"`
}

// PriceSyncItem 是单个本地价格与上游价格的比对结果。
type PriceSyncItem struct {
	ID         string `json:"id"`
	RunID      string `json:"run_id"`
	TargetKind string `json:"target_kind"`
	TargetID   string `json:"target_id"`

	Provider   string `json:"provider"`
	ModelName  string `json:"model_name"`
	SyncSource string `json:"sync_source"`

	MediaType string `json:"media_type"`
	Variant   string `json:"variant"`
	MediaUnit string `json:"media_unit"`

	OldInputUSD    float64 `json:"old_input_usd"`
	NewInputUSD    float64 `json:"new_input_usd"`
	OldOutputUSD   float64 `json:"old_output_usd"`
	NewOutputUSD   float64 `json:"new_output_usd"`
	OldUnitCostUSD float64 `json:"old_unit_cost_usd"`
	NewUnitCostUSD float64 `json:"new_unit_cost_usd"`

	Action        string          `json:"action"`
	Reason        string          `json:"reason"`
	SourcePayload json.RawMessage `json:"source_payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

// PriceSyncSummary 汇总一次同步结果。
type PriceSyncSummary struct {
	TotalCount     int `json:"total_count"`
	UpdateCount    int `json:"update_count"`
	UnchangedCount int `json:"unchanged_count"`
	SkippedCount   int `json:"skipped_count"`
	AppliedCount   int `json:"applied_count"`
	StaleCount     int `json:"stale_count"`
}

// PriceSyncPreviewRequest 发起同步预览。
type PriceSyncPreviewRequest struct {
	TriggerType      string  `json:"trigger_type"`
	Group            string  `json:"group"`
	IncludeText      bool    `json:"include_text"`
	IncludeMedia     bool    `json:"include_media"`
	MaxChangePercent float64 `json:"max_change_percent"`
}

// PriceSyncPreviewResponse 返回同步批次及全部比对明细。
type PriceSyncPreviewResponse struct {
	Run     *PriceSyncRun    `json:"run"`
	Items   []PriceSyncItem  `json:"items"`
	Summary PriceSyncSummary `json:"summary"`
}

// PriceSyncApplyRequest 应用已预览的价格变化。
//
// ApplyAll=true表示应用本批全部update明细。
// ApplyAll=false时ItemIDs必须至少包含一项，只应用管理员明确勾选的明细。
type PriceSyncApplyRequest struct {
	RunID    string   `json:"run_id"`
	ApplyAll bool     `json:"apply_all"`
	ItemIDs  []string `json:"item_ids"`
}

// PriceSyncApplyResponse 返回实际应用结果。
type PriceSyncApplyResponse struct {
	Run     *PriceSyncRun    `json:"run"`
	Items   []PriceSyncItem  `json:"items"`
	Summary PriceSyncSummary `json:"summary"`
}

// PriceSyncSettings 是价格同步全局配置。
type PriceSyncSettings struct {
	Enabled          bool    `json:"enabled"`
	AutoApply        bool    `json:"auto_apply"`
	Group            string  `json:"group"`
	IntervalHours    int     `json:"interval_hours"`
	MaxChangePercent float64 `json:"max_change_percent"`

	MainPricingURL     string `json:"main_pricing_url"`
	DomesticPricingURL string `json:"domestic_pricing_url"`
	MediaPricingURL    string `json:"media_pricing_url"`
	TTSPricingURL      string `json:"tts_pricing_url"`
}

// UpdatePriceSyncSettingsRequest 支持按字段更新全局同步配置。
type UpdatePriceSyncSettingsRequest struct {
	Enabled          *bool    `json:"enabled"`
	AutoApply        *bool    `json:"auto_apply"`
	Group            *string  `json:"group"`
	IntervalHours    *int     `json:"interval_hours"`
	MaxChangePercent *float64 `json:"max_change_percent"`

	MainPricingURL     *string `json:"main_pricing_url"`
	DomesticPricingURL *string `json:"domestic_pricing_url"`
	MediaPricingURL    *string `json:"media_pricing_url"`
	TTSPricingURL      *string `json:"tts_pricing_url"`
}

// PriceSyncTargetConfig 是管理界面中的单个同步目标。
type PriceSyncTargetConfig struct {
	ID         string `json:"id"`
	TargetKind string `json:"target_kind"`

	Provider    string `json:"provider"`
	ModelName   string `json:"model_name"`
	DisplayName string `json:"display_name"`
	IsActive    bool   `json:"is_active"`

	MediaType string `json:"media_type"`
	Variant   string `json:"variant"`
	MediaUnit string `json:"media_unit"`

	CurrentInputUSD    float64 `json:"current_input_usd"`
	CurrentOutputUSD   float64 `json:"current_output_usd"`
	CurrentUnitCostUSD float64 `json:"current_unit_cost_usd"`

	AutoSyncEnabled bool   `json:"auto_sync_enabled"`
	SyncSource      string `json:"sync_source"`
	SyncModelName   string `json:"sync_model_name"`

	LastSyncedAt    *time.Time `json:"last_synced_at"`
	LastSyncStatus  string     `json:"last_sync_status"`
	LastSyncMessage string     `json:"last_sync_message"`
}

// UpdatePriceSyncTargetRequest 更新同步目标，不直接修改正式价格。
type UpdatePriceSyncTargetRequest struct {
	AutoSyncEnabled *bool   `json:"auto_sync_enabled"`
	SyncSource      *string `json:"sync_source"`
	SyncModelName   *string `json:"sync_model_name"`
}

// PriceSyncSettingsResponse 返回全局配置及全部同步目标。
type PriceSyncSettingsResponse struct {
	Settings     PriceSyncSettings       `json:"settings"`
	TextTargets  []PriceSyncTargetConfig `json:"text_targets"`
	MediaTargets []PriceSyncTargetConfig `json:"media_targets"`
}
