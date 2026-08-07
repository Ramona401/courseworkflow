package handlers

// courseware_ai_review_response.go
//
// 课件AI审核浏览器安全响应模型。
//
// 数据库内部会话保存以下敏感或高体量内容：
//   - 系统提示词完整快照；
//   - 所选AI助手提示词原文；
//   - 来源教案正文；
//   - 课程大纲全文；
//   - 全部页面文字与互动代码索引；
//   - 批次输入清单和连续性账本；
//   - 原始技术证据和AI内部执行计划。
//
// 这些内容只供后端顺序执行AI审核，不直接返回浏览器。
//
// 浏览器允许获得：
//   - 会话与批次进度；
//   - 不可变R-02审核配置代码、模式和哈希；
//   - 教师标题、可观察现象、教学影响、调整目标和检查清单；
//   - 教师化最终综合报告；
//   - 安全的错误状态和时间信息。
//
// 内部模型、Token、原始连续性账本、原始技术证据和内部执行计划
// 均不返回教师端。
//
// 兼容说明：
//   旧前端批次解析器要求continuity_ledger存在且为对象。
//   安全响应只返回固定空对象，不返回数据库中的任何连续性事实。

import (
	"encoding/json"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/services"
)

// coursewareAIReviewConfigView 是浏览器可见的不可变审核配置。
type coursewareAIReviewConfigView struct {
	SchemaVersion int `json:"schema_version"`

	ReviewDimensions []string `json:"review_dimensions"`

	CustomDimensionDescription string `json:"custom_dimension_description"`

	LessonReferenceMode string `json:"lesson_reference_mode"`

	UsesLessonMaterials bool `json:"uses_lesson_materials"`

	ReviewConfigHash string `json:"review_config_hash"`
}

// coursewareAIReviewFindingView 是浏览器可见的教师化审核发现。
//
// 不包含原始教案依据、页面证据、代码证据、连续性证据、置信度或内部执行计划。
type coursewareAIReviewFindingView struct {
	ID string `json:"id"`

	Severity  string `json:"severity"`
	Dimension string `json:"dimension"`

	PageNumbers []int `json:"page_numbers"`

	Title       string `json:"title"`
	Description string `json:"description"`

	TeacherViewSnapshot models.CWAIReviewTeacherViewSnapshot `json:"teacher_view_snapshot"`

	Suggestion string `json:"suggestion"`

	ManualReviewRequired bool `json:"manual_review_required"`
}

// coursewareAIReviewRiskPageView 是浏览器可见的教师化风险页。
type coursewareAIReviewRiskPageView struct {
	PageNumber int    `json:"page_number"`
	Severity   string `json:"severity"`
	Reason     string `json:"reason"`

	ManualReviewRequired bool `json:"manual_review_required"`
}

// coursewareAIReviewBatchResultView 是浏览器可见的批次结果。
//
// 原始连续性账本只供后端跨批继承。
// ContinuityLedger固定为空对象，仅兼容旧前端结构校验。
type coursewareAIReviewBatchResultView struct {
	BatchNo     int   `json:"batch_no"`
	PageNumbers []int `json:"page_numbers"`

	BatchSummary string `json:"batch_summary"`

	Findings []*coursewareAIReviewFindingView `json:"findings"`

	ContinuityLedger map[string]interface{} `json:"continuity_ledger"`

	RiskPages []*coursewareAIReviewRiskPageView `json:"risk_pages"`

	ManualReviewRequired bool `json:"manual_review_required"`
}

// coursewareAIReviewFinalReportView 是浏览器可见的教师化最终报告。
type coursewareAIReviewFinalReportView struct {
	ReviewConfig models.CWAIReviewConfigReport `json:"review_config"`

	OverallRisk string `json:"overall_risk"`
	Summary     string `json:"summary"`

	Strengths []string `json:"strengths"`

	Findings []*coursewareAIReviewFindingView `json:"findings"`

	PriorityActions []models.CWAIReviewPriorityAction `json:"priority_actions"`

	ManualReviewPages []int `json:"manual_review_pages"`

	ReviewCommentDraft string `json:"review_comment_draft"`

	HumanDecisionReminder string `json:"human_decision_reminder"`
}

// coursewareAIReviewSessionView 是浏览器可见会话。
type coursewareAIReviewSessionView struct {
	ID           string  `json:"id"`
	CoursewareID string  `json:"courseware_id"`
	ReviewerID   string  `json:"reviewer_id"`
	AssistantID  *string `json:"assistant_id"`
	LessonPlanID *string `json:"lesson_plan_id"`

	ReviewLevel     int    `json:"review_level"`
	EducationDomain string `json:"education_domain"`
	Subject         string `json:"subject"`
	Grade           string `json:"grade"`

	ReviewConfig *coursewareAIReviewConfigView `json:"review_config"`

	Status         string `json:"status"`
	CurrentStage   string `json:"current_stage"`
	CurrentBatchNo int    `json:"current_batch_no"`
	TotalBatches   int    `json:"total_batches"`

	// 只返回教师化最终报告JSON。
	FinalReportJSON string `json:"final_report_json"`

	// 为兼容旧前端保留字段，但不返回模型、Token和内部错误。
	ModelUsed    string `json:"model_used"`
	TokensUsed   int    `json:"tokens_used"`
	ErrorMessage string `json:"error_message"`

	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// coursewareAIReviewBatchView 是浏览器可见批次。
type coursewareAIReviewBatchView struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	BatchNo   int    `json:"batch_no"`
	Status    string `json:"status"`

	// 只返回教师化审核结果与风险页。
	ResultJSON    string `json:"result_json"`
	RiskPagesJSON string `json:"risk_pages_json"`

	// 为兼容旧前端保留字段，但不返回模型、Token和内部错误。
	ModelUsed    string `json:"model_used"`
	TokensUsed   int    `json:"tokens_used"`
	ErrorMessage string `json:"error_message"`

	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// coursewareAIReviewBundleView 是查询或准备接口响应。
type coursewareAIReviewBundleView struct {
	Session *coursewareAIReviewSessionView `json:"session"`
	Batches []*coursewareAIReviewBatchView `json:"batches"`
}

// coursewareAIReviewRunNextView 是运行下一批接口响应。
type coursewareAIReviewRunNextView struct {
	Session *coursewareAIReviewSessionView     `json:"session"`
	Batch   *coursewareAIReviewBatchView       `json:"batch"`
	Result  *coursewareAIReviewBatchResultView `json:"result"`

	HasMore          bool `json:"has_more"`
	RequiresFinalize bool `json:"requires_finalize"`
}

// coursewareAIReviewFinalizeView 是最终综合接口响应。
type coursewareAIReviewFinalizeView struct {
	Session *coursewareAIReviewSessionView     `json:"session"`
	Report  *coursewareAIReviewFinalReportView `json:"report"`
}

func buildCoursewareAIReviewConfigView(
	session *models.CoursewareAIReviewSession,
) *coursewareAIReviewConfigView {
	if session == nil {
		return nil
	}

	dimensions := make([]string, 0)

	if err := json.Unmarshal(
		[]byte(session.ReviewDimensionsJSON),
		&dimensions,
	); err != nil {
		dimensions = []string{}
	}

	normalizedDimensions := make(
		[]string,
		0,
		len(dimensions),
	)
	seen := make(map[string]bool)

	for _, raw := range dimensions {
		dimension := strings.TrimSpace(raw)
		if dimension == "" ||
			seen[dimension] ||
			!models.IsCWAIReviewDimension(dimension) {
			continue
		}

		seen[dimension] = true
		normalizedDimensions = append(
			normalizedDimensions,
			dimension,
		)
	}

	mode := strings.TrimSpace(
		session.LessonReferenceMode,
	)

	return &coursewareAIReviewConfigView{
		SchemaVersion: session.ReviewConfigSchemaVersion,

		ReviewDimensions: normalizedDimensions,

		CustomDimensionDescription: strings.TrimSpace(
			session.CustomDimensionDescription,
		),

		LessonReferenceMode: mode,

		UsesLessonMaterials: mode !=
			models.CWAIReviewLessonReferenceNoLesson,

		ReviewConfigHash: strings.TrimSpace(
			session.ReviewConfigHash,
		),
	}
}

func buildCoursewareAIReviewFindingView(
	finding *models.CWAIReviewFinding,
) *coursewareAIReviewFindingView {
	safe :=
		services.BuildCWAIReviewBrowserFinding(
			finding,
		)
	if safe == nil {
		return nil
	}

	return &coursewareAIReviewFindingView{
		ID: safe.ID,

		Severity:  safe.Severity,
		Dimension: safe.Dimension,

		PageNumbers: append(
			[]int{},
			safe.PageNumbers...,
		),

		Title:       safe.Title,
		Description: safe.Description,

		TeacherViewSnapshot: safe.
			TeacherViewSnapshot,

		Suggestion: safe.Suggestion,

		ManualReviewRequired: safe.
			ManualReviewRequired,
	}
}

func buildCoursewareAIReviewFindingViews(
	findings []models.CWAIReviewFinding,
) []*coursewareAIReviewFindingView {
	result := make(
		[]*coursewareAIReviewFindingView,
		0,
		len(findings),
	)

	for i := range findings {
		view :=
			buildCoursewareAIReviewFindingView(
				&findings[i],
			)
		if view == nil {
			continue
		}

		result = append(result, view)
	}

	return result
}

func buildCoursewareAIReviewRiskPageViews(
	riskPages []models.CWAIReviewRiskPage,
) []*coursewareAIReviewRiskPageView {
	safeRiskPages :=
		services.BuildCWAIReviewBrowserRiskPages(
			riskPages,
		)

	result := make(
		[]*coursewareAIReviewRiskPageView,
		0,
		len(safeRiskPages),
	)

	for _, risk := range safeRiskPages {
		result = append(
			result,
			&coursewareAIReviewRiskPageView{
				PageNumber: risk.PageNumber,
				Severity:   risk.Severity,
				Reason:     risk.Reason,

				ManualReviewRequired: risk.
					ManualReviewRequired,
			},
		)
	}

	return result
}

func buildCoursewareAIReviewBatchResultView(
	result *models.CWAIReviewBatchAIResult,
) *coursewareAIReviewBatchResultView {
	safe :=
		services.BuildCWAIReviewBrowserBatchResult(
			result,
		)
	if safe == nil {
		return nil
	}

	return &coursewareAIReviewBatchResultView{
		BatchNo: safe.BatchNo,

		PageNumbers: append(
			[]int{},
			safe.PageNumbers...,
		),

		BatchSummary: safe.BatchSummary,

		Findings: buildCoursewareAIReviewFindingViews(
			safe.Findings,
		),

		ContinuityLedger: map[string]interface{}{},

		RiskPages: buildCoursewareAIReviewRiskPageViews(
			safe.RiskPages,
		),

		ManualReviewRequired: safe.
			ManualReviewRequired,
	}
}

func buildCoursewareAIReviewFinalReportView(
	report *models.CWAIReviewFinalReport,
) *coursewareAIReviewFinalReportView {
	safe :=
		services.BuildCWAIReviewBrowserFinalReport(
			report,
		)
	if safe == nil {
		return nil
	}

	return &coursewareAIReviewFinalReportView{
		ReviewConfig: safe.ReviewConfig,

		OverallRisk: safe.OverallRisk,
		Summary:     safe.Summary,

		Strengths: append(
			[]string{},
			safe.Strengths...,
		),

		Findings: buildCoursewareAIReviewFindingViews(
			safe.Findings,
		),

		PriorityActions: append(
			[]models.CWAIReviewPriorityAction{},
			safe.PriorityActions...,
		),

		ManualReviewPages: append(
			[]int{},
			safe.ManualReviewPages...,
		),

		ReviewCommentDraft: safe.ReviewCommentDraft,

		HumanDecisionReminder: safe.
			HumanDecisionReminder,
	}
}

func buildCoursewareAIReviewSafeBatchResultJSON(
	raw string,
) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	var result models.CWAIReviewBatchAIResult
	if err := json.Unmarshal(
		[]byte(raw),
		&result,
	); err != nil {
		return ""
	}

	view :=
		buildCoursewareAIReviewBatchResultView(
			&result,
		)

	encoded, err := json.Marshal(view)
	if err != nil {
		return ""
	}

	return string(encoded)
}

func buildCoursewareAIReviewSafeRiskPagesJSON(
	raw string,
) string {
	if strings.TrimSpace(raw) == "" {
		return "[]"
	}

	var riskPages []models.CWAIReviewRiskPage
	if err := json.Unmarshal(
		[]byte(raw),
		&riskPages,
	); err != nil {
		return "[]"
	}

	encoded, err := json.Marshal(
		buildCoursewareAIReviewRiskPageViews(
			riskPages,
		),
	)
	if err != nil {
		return "[]"
	}

	return string(encoded)
}

func buildCoursewareAIReviewSafeFinalReportJSON(
	raw string,
) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	var report models.CWAIReviewFinalReport
	if err := json.Unmarshal(
		[]byte(raw),
		&report,
	); err != nil {
		return ""
	}

	view :=
		buildCoursewareAIReviewFinalReportView(
			&report,
		)

	encoded, err := json.Marshal(view)
	if err != nil {
		return ""
	}

	return string(encoded)
}

func buildCoursewareAIReviewSessionView(
	session *models.CoursewareAIReviewSession,
) *coursewareAIReviewSessionView {
	if session == nil {
		return nil
	}

	errorMessage := ""
	if strings.TrimSpace(session.ErrorMessage) != "" {
		errorMessage =
			"本次检查未能完成，请稍后重试。"
	}

	return &coursewareAIReviewSessionView{
		ID:           session.ID,
		CoursewareID: session.CoursewareID,
		ReviewerID:   session.ReviewerID,
		AssistantID:  session.AssistantID,
		LessonPlanID: session.LessonPlanID,

		ReviewLevel:     session.ReviewLevel,
		EducationDomain: session.EducationDomain,
		Subject:         session.Subject,
		Grade:           session.Grade,

		ReviewConfig: buildCoursewareAIReviewConfigView(
			session,
		),

		Status:         session.Status,
		CurrentStage:   session.CurrentStage,
		CurrentBatchNo: session.CurrentBatchNo,
		TotalBatches:   session.TotalBatches,

		FinalReportJSON: buildCoursewareAIReviewSafeFinalReportJSON(
			session.FinalReportJSON,
		),

		ModelUsed:    "",
		TokensUsed:   0,
		ErrorMessage: errorMessage,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
		CompletedAt:  session.CompletedAt,
	}
}

func buildCoursewareAIReviewBatchView(
	batch *models.CoursewareAIReviewBatch,
) *coursewareAIReviewBatchView {
	if batch == nil {
		return nil
	}

	errorMessage := ""
	if strings.TrimSpace(batch.ErrorMessage) != "" {
		errorMessage =
			"本批页面检查未能完成，请稍后重试。"
	}

	return &coursewareAIReviewBatchView{
		ID:        batch.ID,
		SessionID: batch.SessionID,
		BatchNo:   batch.BatchNo,
		Status:    batch.Status,

		ResultJSON: buildCoursewareAIReviewSafeBatchResultJSON(
			batch.ResultJSON,
		),
		RiskPagesJSON: buildCoursewareAIReviewSafeRiskPagesJSON(
			batch.RiskPagesJSON,
		),

		ModelUsed:    "",
		TokensUsed:   0,
		ErrorMessage: errorMessage,
		StartedAt:    batch.StartedAt,
		CompletedAt:  batch.CompletedAt,
		CreatedAt:    batch.CreatedAt,
		UpdatedAt:    batch.UpdatedAt,
	}
}

func buildCoursewareAIReviewBatchViews(
	batches []*models.CoursewareAIReviewBatch,
) []*coursewareAIReviewBatchView {
	result := make(
		[]*coursewareAIReviewBatchView,
		0,
		len(batches),
	)

	for _, batch := range batches {
		if batch == nil {
			continue
		}

		result = append(
			result,
			buildCoursewareAIReviewBatchView(batch),
		)
	}

	return result
}

func buildCoursewareAIReviewBundleView(
	session *models.CoursewareAIReviewSession,
	batches []*models.CoursewareAIReviewBatch,
) *coursewareAIReviewBundleView {
	return &coursewareAIReviewBundleView{
		Session: buildCoursewareAIReviewSessionView(
			session,
		),
		Batches: buildCoursewareAIReviewBatchViews(
			batches,
		),
	}
}

func buildCoursewareAIReviewRunNextView(
	response *models.CWAIReviewRunNextResponse,
) *coursewareAIReviewRunNextView {
	if response == nil {
		return nil
	}

	return &coursewareAIReviewRunNextView{
		Session: buildCoursewareAIReviewSessionView(
			response.Session,
		),
		Batch: buildCoursewareAIReviewBatchView(
			response.Batch,
		),
		Result: buildCoursewareAIReviewBatchResultView(
			response.Result,
		),

		HasMore: response.HasMore,

		RequiresFinalize: response.RequiresFinalize,
	}
}

func buildCoursewareAIReviewFinalizeView(
	response *models.CWAIReviewFinalizeResponse,
) *coursewareAIReviewFinalizeView {
	if response == nil {
		return nil
	}

	return &coursewareAIReviewFinalizeView{
		Session: buildCoursewareAIReviewSessionView(
			response.Session,
		),
		Report: buildCoursewareAIReviewFinalReportView(
			response.Report,
		),
	}
}
