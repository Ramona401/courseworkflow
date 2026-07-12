package models

// courseware_normalized.go — 教案规整缓存表 courseware_normalized_lessons 数据模型
//
// 对应表 courseware_normalized_lessons（一课件一条规整结果，UNIQUE courseware_id）。
// 规整层把"又长又乱、排版不规范"的教案原文规整成"结构清晰、去噪保核、预置清单一字不差"
// 的干净教案，存本表供逐页生成时注入，根治"课件对教案还原度不高、跨页共享案例对不上"。

import "time"

// 规整状态常量
const (
	CWNormalizeStatusPending    = "pending"    // 待规整（占位，当前流程一般直接进 generating）
	CWNormalizeStatusGenerating = "generating" // 规整中（AI 调用进行中）
	CWNormalizeStatusDone       = "done"       // 规整成功（normalized_content 可用）
	CWNormalizeStatusFailed     = "failed"     // 规整失败（error_message 记原因，下游退回原文）
)

// CoursewareNormalizedLesson 教案规整缓存记录
type CoursewareNormalizedLesson struct {
	ID                string    `json:"id"`
	CoursewareID      string    `json:"courseware_id"`      // 归属课件（唯一键）
	SourceType        string    `json:"source_type"`        // 冗余记来源(lesson_plan/doc_upload)
	SourceRef         string    `json:"source_ref"`         // 冗余记原文出处(教案id 或 docx文件名)
	NormalizedContent string    `json:"normalized_content"` // 规整后的干净教案正文
	Status            string    `json:"status"`             // pending/generating/done/failed
	ErrorMessage      string    `json:"error_message"`      // 规整失败原因
	ModelUsed         string    `json:"model_used"`         // 实际规整模型
	TokensUsed        int       `json:"tokens_used"`        // 消耗token
	RawCharCount      int       `json:"raw_char_count"`     // 原文字符数
	NormCharCount     int       `json:"norm_char_count"`    // 规整后字符数
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// HasUsableContent 判断该规整记录是否有可注入下游的正文。
//   仅当 status=done 且正文非空时为真；供注入层决定"用规整结果 还是 退回原文"。
func (n *CoursewareNormalizedLesson) HasUsableContent() bool {
	return n != nil && n.Status == CWNormalizeStatusDone && len(n.NormalizedContent) > 0
}
