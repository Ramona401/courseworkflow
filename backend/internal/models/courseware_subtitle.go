package models

// courseware_subtitle.go — 课件字幕轨数据模型
//
// v0.42.8 新增：字幕轨表 courseware_subtitles
// 核心洞察：字幕和 TTS 共享 SubtitleSegment 数据结构
// 一条数据三个用途：软字幕预览(SRT/VTT) + 硬字幕烧录(FFmpeg burn-in) + TTS配音(v0.42.9)

import (
	"time"
)

// ==================== 字幕片段模型 ====================

// SubtitleSegment 单条字幕片段（存储在 segments JSONB 数组中）
type SubtitleSegment struct {
	ID             string  `json:"id"`               // 唯一标识（前端生成 UUID）
	StartSec       float64 `json:"start_sec"`         // 起始秒数
	EndSec         float64 `json:"end_sec"`           // 结束秒数
	Text           string  `json:"text"`              // 字幕文本内容
	Language       string  `json:"language"`          // BCP-47 语言代码: zh-CN / en-US
	// TTS 相关字段（有值表示已生成，v0.42.9 使用）
	TTSAudioURL    string  `json:"tts_audio_url,omitempty"`    // TTS 音频文件 URL
	TTSVoice       string  `json:"tts_voice,omitempty"`        // 音色代码
	TTSDuration    float64 `json:"tts_duration,omitempty"`     // TTS 音频实际时长（秒）
	TTSGeneratedAt string  `json:"tts_generated_at,omitempty"` // TTS 生成时间
}

// ==================== 字幕轨主表模型 ====================

// CoursewareSubtitle 字幕轨记录（对应 courseware_subtitles 表）
type CoursewareSubtitle struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"`
	ScopeType    string     `json:"scope_type"`     // video_asset / editor_draft / page
	ScopeID      *string    `json:"scope_id"`       // 关联的 asset_id / draft_id / page_id
	Language     string     `json:"language"`        // BCP-47: zh-CN / en-US
	Segments     string     `json:"segments"`        // JSONB 原始字符串（[]SubtitleSegment）
	StyleConfig  *string    `json:"style_config"`    // 样式 JSONB（字号/颜色/位置/字体）
	TTSConfig    *string    `json:"tts_config"`      // TTS 默认配置 JSONB（v0.42.9）
	CreatedBy    *string    `json:"created_by"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// ==================== 字幕样式配置 ====================

// SubtitleStyleConfig 字幕显示样式（存储在 style_config JSONB）
type SubtitleStyleConfig struct {
	FontSize   int    `json:"font_size"`   // 字号，默认 24
	FontColor  string `json:"font_color"`  // 字体颜色，默认 #FFFFFF
	BgColor    string `json:"bg_color"`    // 背景颜色，默认 #40000000（半透明黑）
	Position   string `json:"position"`    // 位置: bottom / top / center，默认 bottom
	FontFamily string `json:"font_family"` // 字体名，默认空（系统默认）
	Outline    int    `json:"outline"`     // 描边宽度，默认 2
}

// DefaultSubtitleStyle 默认字幕样式
var DefaultSubtitleStyle = SubtitleStyleConfig{
	FontSize:   24,
	FontColor:  "#FFFFFF",
	BgColor:    "#40000000",
	Position:   "bottom",
	FontFamily: "",
	Outline:    2,
}

// ==================== 字幕范围类型常量 ====================

const (
	SubScopeVideoAsset  = "video_asset"  // 关联单个视频资产
	SubScopeEditorDraft = "editor_draft" // 关联视频编辑器草稿
	SubScopePage        = "page"         // 关联课件页面
)

// ==================== 请求/响应结构体 ====================

// UpsertSubtitleRequest 创建或更新字幕轨请求
type UpsertSubtitleRequest struct {
	ScopeType   string  `json:"scope_type"`            // 必填: video_asset / editor_draft / page
	ScopeID     *string `json:"scope_id"`              // 关联 ID（可空）
	Language    string  `json:"language"`               // 必填: zh-CN / en-US 等
	Segments    string  `json:"segments"`               // 必填: SubtitleSegment[] JSON 字符串
	StyleConfig *string `json:"style_config,omitempty"` // 可选: 样式 JSON
	TTSConfig   *string `json:"tts_config,omitempty"`   // 可选: TTS 配置 JSON（v0.42.9）
}

// ExportSRTRequest 导出 SRT 文件请求（无额外参数，由 URL path 确定字幕 ID）

// BurnInSubtitleRequest 字幕烧录请求
type BurnInSubtitleRequest struct {
	VideoAssetID string `json:"video_asset_id"` // 要烧录字幕的视频资产 ID
}

// BurnInSubtitleResponse 字幕烧录响应
type BurnInSubtitleResponse struct {
	AssetID  string `json:"asset_id"`  // 烧录后的新视频资产 ID
	URL      string `json:"url"`       // 新视频 URL
	Duration string `json:"duration"`  // 视频时长
	Message  string `json:"message"`
}


// ==================== v0.42.9 TTS 请求/响应结构体 ====================

// GenerateTTSRequest TTS配音请求
type GenerateTTSRequest struct {
	Voice      string   `json:"voice"`                 // 音色代码
	Speed      float64  `json:"speed,omitempty"`       // 语速 0.25-4.0，默认1.0
	SegmentIDs []string `json:"segment_ids,omitempty"` // 指定处理的字幕条ID（为空则全部）
}

// GenerateTTSResponse TTS配音响应
type GenerateTTSResponse struct {
	SubtitleID   string   `json:"subtitle_id"`   // 字幕轨ID
	SuccessCount int      `json:"success_count"`  // 成功条数
	FailCount    int      `json:"fail_count"`     // 失败条数
	TotalCount   int      `json:"total_count"`    // 总处理条数
	Segments     string   `json:"segments"`        // 更新后的完整 segments JSON
	Errors       []string `json:"errors"`          // 失败详情
	Message      string   `json:"message"`
}
