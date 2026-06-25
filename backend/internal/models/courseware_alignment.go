package models

// courseware_alignment.go — 课件↔教案对齐报告数据模型
//
// 功能背景：
//   课件方案（层2逐页方案）由 AI 从教案翻译生成。本模型承载"方案是否忠实还原教案
//   教学意图"的校验结果，对应 courseware_alignment_reports 表（一课件一报告，重算覆盖）。
//
// 仅当课件来源为 lesson_plan 时才有对齐报告；主题/PPT/Doc 来源无教案可比对，不生成。
//
// 数据流：
//   方案落库后（saveAndBroadcast）→ 异步触发对齐校验 → AI 输出结构化JSON →
//   落 courseware_alignment_reports → 前端 Step1 GET 拉取 + status=generating 时短轮询。

import "time"

// ==================== 对齐报告状态常量 ====================

const (
	// CWAlignStatusGenerating 校验进行中（AI 调用尚未返回）
	CWAlignStatusGenerating = "generating"
	// CWAlignStatusDone 校验完成
	CWAlignStatusDone = "done"
	// CWAlignStatusFailed 校验失败（AI 调用失败 / 解析失败 / 教案内容过少等）
	CWAlignStatusFailed = "failed"
)

// ==================== 整体对齐结论常量 ====================

const (
	// CWAlignOverallAligned 高度一致：核心环节全覆盖、无明显遗漏、无方向性偏移
	CWAlignOverallAligned = "aligned"
	// CWAlignOverallMinor 有小偏差：1-2 处简略或无害新增，但主线完整
	CWAlignOverallMinor = "minor"
	// CWAlignOverallMajor 有明显问题：任一核心环节缺失，或存在方向性的教学意图偏移
	CWAlignOverallMajor = "major"
	// CWAlignOverallFailed 校验未能完成（与 status=failed 配套，前端据此显示"校验失败"）
	CWAlignOverallFailed = "failed"
)

// ==================== 覆盖度状态常量 ====================

const (
	// CWAlignCovCovered 教案该环节有对应课件页且内容相符
	CWAlignCovCovered = "covered"
	// CWAlignCovPartial 有对应页但内容明显不足/简略
	CWAlignCovPartial = "partial"
	// CWAlignCovMissing 教案有该环节但课件方案完全无对应页（最需老师注意）
	CWAlignCovMissing = "missing"
)

// ==================== 数据库实体 ====================

// CoursewareAlignmentReport 对齐报告主记录（对应 courseware_alignment_reports 表）
type CoursewareAlignmentReport struct {
	ID           string  `json:"id"`
	CoursewareID string  `json:"courseware_id"`
	// LessonPlanID 对齐时所基于的教案ID（冗余存，便于追溯；可空）
	LessonPlanID *string `json:"lesson_plan_id"`
	// Overall 整体结论（见 CWAlignOverall* 常量）
	Overall string `json:"overall"`
	// Summary 一句话总览
	Summary string `json:"summary"`
	// ReportJSON 完整结构化报告（coverage/additions/intent_shifts），原始 JSON 文本
	// 读出后由前端直接解析渲染；后端落库前已确保是合法 JSON 对象。
	ReportJSON string `json:"report_json"`
	// Status 校验状态（见 CWAlignStatus* 常量）
	Status string `json:"status"`
	// ErrorMessage 失败原因（status=failed 时填）
	ErrorMessage string `json:"error_message"`
	// ModelUsed / TokensUsed 本次校验消耗（成本追溯）
	ModelUsed  string `json:"model_used"`
	TokensUsed int    `json:"tokens_used"`
	// PageCount 校验时基于的课件页数（用于前端判断"方案改动后报告是否过期"的参考）
	PageCount int        `json:"page_count"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// ==================== AI 输出 JSON 结构（解析用） ====================
//
// 对应 prompt_courseware_alignment 约定的输出格式。AI 返回的 JSON 反序列化到
// AlignmentAIResult，校验/清洗后再把整体结论拆出落库，report_json 存清洗后的完整 JSON。

// AlignmentCoverageItem 单个教学环节的覆盖情况
type AlignmentCoverageItem struct {
	// PlanSegment 教案里的环节名（如"情境导入""新知讲解""课堂练习"）
	PlanSegment string `json:"plan_segment"`
	// Status 覆盖状态（见 CWAlignCov* 常量）
	Status string `json:"status"`
	// PageNums 对应的课件页码数组（missing 时为空数组）
	PageNums []int `json:"page_nums"`
	// Note 简短说明
	Note string `json:"note"`
}

// AlignmentAdditionItem 课件方案中新增的、教案没有的内容
type AlignmentAdditionItem struct {
	PageNum int    `json:"page_num"`
	Desc    string `json:"desc"`
}

// AlignmentIntentShiftItem 教学意图偏移
type AlignmentIntentShiftItem struct {
	PageNum       int    `json:"page_num"`
	PlanIntent    string `json:"plan_intent"`    // 教案对应环节的目标
	SchemePurpose string `json:"scheme_purpose"` // 课件这一页的目的
	Note          string `json:"note"`           // 偏移点说明
}

// AlignmentAIResult AI 返回的完整对齐分析结果（与 prompt 输出结构对齐）
type AlignmentAIResult struct {
	Overall      string                     `json:"overall"`
	Summary      string                     `json:"summary"`
	Coverage     []AlignmentCoverageItem    `json:"coverage"`
	Additions    []AlignmentAdditionItem    `json:"additions"`
	IntentShifts []AlignmentIntentShiftItem `json:"intent_shifts"`
}

// ==================== 响应结构 ====================

// AlignmentReportResponse 对齐报告查询响应
// 当课件无报告（从未校验 / 非教案来源）时，HasReport=false，前端不显示对齐卡片。
type AlignmentReportResponse struct {
	HasReport bool                       `json:"has_report"`
	Report    *CoursewareAlignmentReport `json:"report"`
}
