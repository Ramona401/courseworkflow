package models

import (
	"encoding/json"
	"time"
)

// ============================================================
// 知识库压缩入库系统 · 数据模型
// 对应表：kb_compress_jobs / kb_compress_items / kb_authorized_users
//        （textbook_unit_images 教材子系统迭代二再用，本文件预留常量）
//
// 设计要点：
//   - 与全平台风格一致：snake_case json tag、常量分组+NameMap、可空用指针、
//     JSONB 列在 Go 侧存为原始 JSON 字符串 + ToJSON/Parse 辅助函数。
//   - 中间真相层语义：半成品全程驻留 kb_compress_items，draft_rounds 存多轮草稿，
//     arbitration 存仲裁结论，final_line 存最终采纳的一行制索引（成品真相）。
//   - 蓝绿切换：成品确认后才 commit 写目标表(curriculum_standards/textbook_units)，
//     带 batch_tag；整批确认后旧批 status→archived、新批→active。
// ============================================================

// ==================== 压缩任务主表模型（kb_compress_jobs，16列） ====================

// KBCompressJob 知识库压缩任务（一次上传对应一条）
// 状态机：uploaded → parsing → compressing → arbitrating → reviewing → done/failed
type KBCompressJob struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`          // curriculum / textbook
	BatchTag     string     `json:"batch_tag"`     // 批次标识（蓝绿切换单位）
	SourceFile   string     `json:"source_file"`   // 上传来源文件路径（图片/文本落盘路径，可空）
	CompressMode string     `json:"compress_mode"` // fast(急速,只读文字) / precise(精准,AI读图)
	Subject      string     `json:"subject"`       // 学科（定位字段，可空）
	Publisher    string     `json:"publisher"`     // 出版社（教材用，可空）
	GradeNum     *int       `json:"grade_num"`     // 年级（可空）
	Semester     string     `json:"semester"`      // 学期（教材用，可空）
	UnitNumber   *int       `json:"unit_number"`   // 单元号（教材用，可空）
	Status       string     `json:"status"`        // 任务状态
	TotalItems   int        `json:"total_items"`   // 拆出的待压缩单元总数
	DoneItems    int        `json:"done_items"`    // 已完成压缩数
	CreatedBy    *string    `json:"created_by"`    // 创建人（可空）
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// ==================== 压缩单元表模型（kb_compress_items，21列，中间真相层核心） ====================

// KBCompressItem 单个待压缩单元（一个任务拆出的每个知识点/教材单元一条）
// 半成品全程驻留本结构：经多轮压缩(draft_rounds)+仲裁(arbitration)定 confidence，
// 高置信自动 final_line+auto_passed，低置信进人工审核。
type KBCompressItem struct {
	ID            string     `json:"id"`
	JobID         string     `json:"job_id"`
	Kind          string     `json:"kind"`           // curriculum / textbook（冗余自 job 便于查询）
	Seq           int        `json:"seq"`            // 任务内序号（与 job_id 组成唯一约束）
	SourceExcerpt string     `json:"source_excerpt"` // 原文片段（供审核比对，不展示索引时仍可看原文）
	PageLabel     string     `json:"page_label"`     // 页码标签（可空）
	DraftRounds   string     `json:"draft_rounds"`   // JSONB原文：多轮压缩草稿数组，见 KBDraftRound
	Confidence    string     `json:"confidence"`     // high / low（仲裁后填，可空）
	Arbitration   string     `json:"arbitration"`    // JSONB原文：仲裁结论，见 KBArbitration（可空）
	FinalLine     string     `json:"final_line"`     // 最终采纳的一行制索引（成品真相，可空直到定稿）
	ReviewStatus  string     `json:"review_status"`  // pending/auto_passed/need_review/approved/rejected/archived
	ReviewerID    *string    `json:"reviewer_id"`    // 审核人（可空）
	ReviewedAt    *time.Time `json:"reviewed_at"`    // 审核时间（可空）
	ReviewNote    string     `json:"review_note"`    // 审核意见（专利校准锚点，可空）
	AttemptCount  int        `json:"attempt_count"`  // 压缩尝试次数（失败重试累计）
	LastError     string     `json:"last_error"`     // 最后一次失败原因（可空）
	TokensTotal   int64      `json:"tokens_total"`   // 累计 token 消耗（成本追溯）
	Committed     bool       `json:"committed"`      // 是否已 commit 写入目标成品表
	CommittedRef  *string    `json:"committed_ref"`  // 目标表记录 id（可空）
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// ==================== draft_rounds JSONB 子结构 ====================

// KBDraftRound 单轮独立压缩草稿（draft_rounds 数组的元素）
// 各轮独立产出、不共享中间态，全部留痕供仲裁与审核选版。
type KBDraftRound struct {
	Round  int    `json:"round"`           // 轮次序号（从1起）
	Line   string `json:"line"`            // 本轮压缩产出的一行制索引
	Model  string `json:"model"`           // 本轮使用的模型
	Tokens int    `json:"tokens"`          // 本轮 token 消耗
	At     string `json:"at"`              // 产出时间（RFC3339字符串）
	Error  string `json:"error,omitempty"` // 本轮失败原因（成功为空）
}

// ParseDraftRounds 从 JSONB 文本解析多轮草稿数组（空/损坏均返回空切片，不报错）
func ParseDraftRounds(jsonStr string) []KBDraftRound {
	if jsonStr == "" {
		return []KBDraftRound{}
	}
	var rounds []KBDraftRound
	if err := json.Unmarshal([]byte(jsonStr), &rounds); err != nil {
		return []KBDraftRound{}
	}
	return rounds
}

// DraftRoundsToJSON 将多轮草稿数组序列化为 JSONB 文本（失败返回 "[]"）
func DraftRoundsToJSON(rounds []KBDraftRound) string {
	if rounds == nil {
		return "[]"
	}
	data, err := json.Marshal(rounds)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// ==================== arbitration JSONB 子结构 ====================

// KBArbitration 语义一致性仲裁结论（arbitration 列）
// Consistent=true 表示多轮语义一致(高置信)，取 ChosenRound 作 final_line；
// false 表示存在关键事实矛盾(低置信)，Conflicts 列出冲突点供人工审核高亮。
type KBArbitration struct {
	Consistent  bool     `json:"consistent"`   // 多轮是否语义/逻辑一致
	Conflicts   []string `json:"conflicts"`    // 冲突点描述（低置信时非空，供审核界面高亮）
	ChosenRound int      `json:"chosen_round"` // 选中的最优轮次序号（高置信时指向最完整的一轮）
	Reason      string   `json:"reason"`       // 仲裁理由（中文说明）
	Model       string   `json:"model"`        // 仲裁使用的模型
	Tokens      int      `json:"tokens"`       // 仲裁 token 消耗
}

// ParseArbitration 从 JSONB 文本解析仲裁结论（空/损坏返回 nil，不报错）
func ParseArbitration(jsonStr string) *KBArbitration {
	if jsonStr == "" {
		return nil
	}
	var arb KBArbitration
	if err := json.Unmarshal([]byte(jsonStr), &arb); err != nil {
		return nil
	}
	return &arb
}

// ArbitrationToJSON 将仲裁结论序列化为 JSONB 文本（nil 返回空串语义=NULL）
func ArbitrationToJSON(arb *KBArbitration) string {
	if arb == nil {
		return ""
	}
	data, err := json.Marshal(arb)
	if err != nil {
		return ""
	}
	return string(data)
}

// ==================== 白名单模型（kb_authorized_users，4列） ====================

// KBAuthorizedUser 知识库压缩子系统访问白名单条目
type KBAuthorizedUser struct {
	UserID    string     `json:"user_id"`
	GrantedBy *string    `json:"granted_by"` // 授权操作人（可空）
	Note      string     `json:"note"`       // 授权备注
	CreatedAt *time.Time `json:"created_at"`
}

// KBAuthorizedUserItem 白名单列表展示项（JOIN users 后带用户名）
type KBAuthorizedUserItem struct {
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`     // 被授权人登录名
	DisplayName string     `json:"display_name"` // 被授权人显示名
	Role        string     `json:"role"`         // 被授权人当前角色（仅展示，不影响授权）
	GrantedBy   string     `json:"granted_by"`   // 授权人显示名（COALESCE 兜底）
	Note        string     `json:"note"`
	CreatedAt   *time.Time `json:"created_at"`
}

// ==================== 任务/单元 状态与种类常量 ====================

const (
	// 任务种类
	KBKindCurriculum = "curriculum" // 课标
	KBKindTextbook   = "textbook"   // 教材

	// 压缩模式
	KBCompressModeFast    = "fast"    // 急速：只读文字
	KBCompressModePrecise = "precise" // 精准：AI 读图（教材用）

	// 任务状态
	KBJobStatusUploaded    = "uploaded"
	KBJobStatusParsing     = "parsing"
	KBJobStatusCompressing = "compressing"
	KBJobStatusArbitrating = "arbitrating"
	KBJobStatusReviewing   = "reviewing"
	KBJobStatusDone        = "done"
	KBJobStatusFailed      = "failed"

	// 单元审核状态
	KBReviewStatusPending    = "pending"     // 未压缩/未仲裁
	KBReviewStatusAutoPassed = "auto_passed" // 高置信自动通过候选
	KBReviewStatusNeedReview = "need_review" // 低置信进人工
	KBReviewStatusApproved   = "approved"    // 人工确认通过
	KBReviewStatusRejected   = "rejected"    // 人工退回
	KBReviewStatusArchived   = "archived"    // 已归档（蓝绿切换旧批）

	// 置信分级
	KBConfidenceHigh = "high"
	KBConfidenceLow  = "low"
)

// KBJobStatusNameMap 任务状态中文名
var KBJobStatusNameMap = map[string]string{
	KBJobStatusUploaded:    "已上传",
	KBJobStatusParsing:     "解析中",
	KBJobStatusCompressing: "压缩中",
	KBJobStatusArbitrating: "仲裁中",
	KBJobStatusReviewing:   "待审核",
	KBJobStatusDone:        "已完成",
	KBJobStatusFailed:      "失败",
}

// KBReviewStatusNameMap 单元审核状态中文名
var KBReviewStatusNameMap = map[string]string{
	KBReviewStatusPending:    "待处理",
	KBReviewStatusAutoPassed: "自动通过",
	KBReviewStatusNeedReview: "待人工审核",
	KBReviewStatusApproved:   "已确认",
	KBReviewStatusRejected:   "已退回",
	KBReviewStatusArchived:   "已归档",
}

// IsValidKBKind 校验任务种类
func IsValidKBKind(kind string) bool {
	return kind == KBKindCurriculum || kind == KBKindTextbook
}

// IsValidKBCompressMode 校验压缩模式
func IsValidKBCompressMode(mode string) bool {
	return mode == KBCompressModeFast || mode == KBCompressModePrecise
}

// ==================== 请求/响应结构体 ====================

// KBCreateJobRequest 创建压缩任务请求
// 课标先行：输入为图片(多模态)或纯文本粘贴；Rounds 为可配置轮数(默认3)。
type KBCreateJobRequest struct {
	Kind         string `json:"kind"`          // curriculum / textbook
	BatchTag     string `json:"batch_tag"`     // 批次标识（必填，蓝绿切换单位）
	CompressMode string `json:"compress_mode"` // fast / precise
	Subject      string `json:"subject"`       // 学科（可选定位）
	GradeNum     *int   `json:"grade_num"`     // 年级（可选定位）
	Rounds       int    `json:"rounds"`        // 多轮压缩轮数（可配置，<=0 时服务层兜底为3）
	// 输入内容（二选一或都有）：
	TextContent   string   `json:"text_content"`    // 纯文本粘贴（课标原文）
	ImageDataURIs []string `json:"image_data_uris"` // 图片 dataURI 数组（多模态识别）
}

// KBDefaultRounds 多轮压缩默认轮数
const KBDefaultRounds = 3

// KBReviewActionRequest 审核动作请求（确认/选版/退回三选一）
type KBReviewActionRequest struct {
	Action      string `json:"action"`       // confirm / select / reject
	ChosenRound int    `json:"chosen_round"` // action=select 时指定采纳的轮次序号
	ReviewNote  string `json:"review_note"`  // 审核意见（可选）
}

// 审核动作常量
const (
	KBReviewActionConfirm = "confirm" // 确认通过（采纳仲裁选中的版本）
	KBReviewActionSelect  = "select"  // 选版（从多轮中选指定一版）
	KBReviewActionReject  = "reject"  // 退回（几版都不对，退回重压）
)

// KBAddAuthorizedRequest 添加白名单成员请求
type KBAddAuthorizedRequest struct {
	UserID string `json:"user_id"` // 被授权用户ID（必填）
	Note   string `json:"note"`    // 授权备注（可选）
}

// ==================== 解码展示结构体（供审核界面，索引原文不暴露） ====================

// KBDecodedField 解码后的单个语义字段（人话）
type KBDecodedField struct {
	Label   string `json:"label"`   // 中文标签名（如"学业要求"）
	Tag     string `json:"tag"`     // 原标签字母（如"E"，仅供调试，前端可不显示）
	Content string `json:"content"` // 字段内容文本
}

// KBDecodedIndex 一行索引解码后的人话卡片（审核员所见，绝不含索引原文符号）
type KBDecodedIndex struct {
	KPCode       string           `json:"kp_code"`       // 知识点编码（标识，非机密符号）
	SubjectName  string           `json:"subject_name"`  // 学科中文（SJ 解码）
	StageName    string           `json:"stage_name"`    // 学段中文（SG 解码）
	GradeName    string           `json:"grade_name"`    // 年级中文（GR 解码）
	DepthName    string           `json:"depth_name"`    // 深度档中文（DP 解码）
	Fields       []KBDecodedField `json:"fields"`        // 语义字段人话列表
	DecodeFailed bool             `json:"decode_failed"` // 解码是否遇到无法识别项（降级展示，不阻断）
}

// KBReviewItemView 审核队列单项视图（低置信项含多轮并排解码）
type KBReviewItemView struct {
	ItemID        string        `json:"item_id"`
	Seq           int           `json:"seq"`
	Confidence    string        `json:"confidence"` // high / low
	ReviewStatus  string        `json:"review_status"`
	SourceExcerpt string        `json:"source_excerpt"` // 原文片段（供比对）
	PageLabel     string        `json:"page_label"`
	Conflicts     []string      `json:"conflicts"`    // 冲突点（低置信时供高亮）
	ChosenRound   int           `json:"chosen_round"` // 仲裁选中的轮次
	Rounds        []KBRoundView `json:"rounds"`       // 各轮解码后的人话并排
}

// KBRoundView 单轮草稿的解码视图（人话，不含索引原文）
type KBRoundView struct {
	Round   int             `json:"round"`
	Model   string          `json:"model"`
	Decoded *KBDecodedIndex `json:"decoded"` // 该轮索引解码后的人话卡片
	Error   string          `json:"error,omitempty"`
}
