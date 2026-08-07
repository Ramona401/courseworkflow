package services

// courseware_ai_review_teacher_view.go
//
// 课件AI审核发现的教师视图快照归一化与浏览器安全转换。
//
// 设计目标：
//   1. 新模型输出优先使用结构化教师字段；
//   2. 历史或字段缺失记录使用确定性规则降级；
//   3. 教师默认字段不得包含平台实现术语；
//   4. 每条问题始终形成2至5条可观察检查项；
//   5. 无法可靠从技术证据转译时明确要求人工检查；
//   6. 技术证据和内部执行计划继续保留在后端事实层；
//   7. 浏览器安全转换只返回教师表达，不返回内部执行计划和技术证据；
//   8. 本文件不调用AI、不写数据库、不改变审核决定。

import (
	"encoding/json"
	"strings"

	"tedna/internal/models"
)

const (
	cwAIReviewTeacherTitleMaxRunes = 120
	cwAIReviewTeacherTextMaxRunes  = 800
	cwAIReviewTeacherCheckMaxRunes = 260

	cwAIReviewTeacherCheckMin = 2
	cwAIReviewTeacherCheckMax = 5

	// cwAIReviewManualBrowserCheck 是无法仅凭静态证据确认时，
	// 向教师补充的统一实际操作检查项。
	//
	// 已有检查项不会被替换；总数量仍受2至5条边界约束。
	cwAIReviewManualBrowserCheck = "请在实际浏览器中完整操作或播放一次，确认页面表现与预期一致。"
)

// cwAIReviewTeacherPlatformTerms 是教师默认字段禁止出现的平台实现术语。
//
// “代码”没有加入全局禁词，因为课程本身可能属于编程学科；
// 但HTML、CSS、JavaScript、DOM和TE-DNA内部状态等实现细节仍必须隐藏。
var cwAIReviewTeacherPlatformTerms = []string{
	"html",
	"css",
	"javascript",
	"localstorage",
	"dom",
	"console",
	"element id",
	"element_id",
	"runtime",
	"confirmed",
	"superseded",
	"stale",
	"orphaned",
	"applied",
	"json",
	"token",
	"prompt",
	"selector",
	"script",
	"function",
	"函数名",
	"变量名",
	"脚本",
	"选择器",
	"元素id",
	"元素 id",
	"控制台",
	"哈希",
	"输入账本",
	"模型供应商",
	"提示词",
	"数据库表",
	"内部状态",
	"执行追踪",
	"错误堆栈",
	"解析错误",
	"脚本闭合",
	"本地存储",
}

// normalizeCWAIReviewTeacherViewForFinding 为finding形成稳定教师视图。
//
// 该函数可重复调用。第一次调用完成模型结果归一化，后续物化或最终综合
// 再次调用时不会破坏已经符合要求的教师字段。
func normalizeCWAIReviewTeacherViewForFinding(
	finding *models.CWAIReviewFinding,
) {
	if finding == nil {
		return
	}

	finding.Title = strings.TrimSpace(finding.Title)
	finding.Description = strings.TrimSpace(finding.Description)
	finding.Suggestion = strings.TrimSpace(finding.Suggestion)
	finding.InternalExecutionPlan = strings.TrimSpace(
		finding.InternalExecutionPlan,
	)

	// 旧协议没有InternalExecutionPlan时，先保存原始suggestion作为内部兼容计划。
	// 随后Suggestion会被收敛为教师可读的ImprovementGoal。
	if finding.InternalExecutionPlan == "" {
		finding.InternalExecutionPlan = finding.Suggestion
	}

	defaults := cwAIReviewTeacherDefaultsForDimension(
		finding.Dimension,
	)

	snapshot := finding.TeacherViewSnapshot

	var titleFallback bool
	snapshot.TeacherTitle, titleFallback =
		cwAIReviewNormalizeTeacherText(
			snapshot.TeacherTitle,
			cwAIReviewFirstSafeTeacherText(
				finding.Title,
				defaults.Title,
			),
			cwAIReviewTeacherTitleMaxRunes,
		)

	var happenedFallback bool
	snapshot.WhatHappened, happenedFallback =
		cwAIReviewNormalizeTeacherText(
			snapshot.WhatHappened,
			cwAIReviewFirstSafeTeacherText(
				finding.Description,
				defaults.WhatHappened,
			),
			cwAIReviewTeacherTextMaxRunes,
		)

	var impactFallback bool
	snapshot.TeachingImpact, impactFallback =
		cwAIReviewNormalizeTeacherText(
			snapshot.TeachingImpact,
			defaults.TeachingImpact,
			cwAIReviewTeacherTextMaxRunes,
		)

	var goalFallback bool
	snapshot.ImprovementGoal, goalFallback =
		cwAIReviewNormalizeTeacherText(
			snapshot.ImprovementGoal,
			cwAIReviewFirstSafeTeacherText(
				finding.Suggestion,
				defaults.ImprovementGoal,
			),
			cwAIReviewTeacherTextMaxRunes,
		)

	var contextFallback bool
	snapshot.TeacherContext, contextFallback =
		cwAIReviewNormalizeOptionalTeacherText(
			snapshot.TeacherContext,
			cwAIReviewTeacherTextMaxRunes,
		)

	var checksFallback bool
	snapshot.AcceptanceChecks, checksFallback =
		cwAIReviewNormalizeTeacherChecks(
			snapshot.AcceptanceChecks,
			defaults.AcceptanceChecks,
			finding.ManualReviewRequired ||
				snapshot.ManualCheckRequired,
		)

	snapshot.ManualCheckRequired =
		snapshot.ManualCheckRequired ||
			finding.ManualReviewRequired ||
			titleFallback ||
			happenedFallback ||
			impactFallback ||
			goalFallback ||
			contextFallback ||
			checksFallback

	finding.ManualReviewRequired =
		finding.ManualReviewRequired ||
			snapshot.ManualCheckRequired

	finding.TeacherViewSnapshot = snapshot

	// 旧浏览器仍可能读取suggestion，因此兼容字段也必须是教师语言。
	finding.Suggestion = snapshot.ImprovementGoal
}

// BuildCWAIReviewBrowserFinding 将一条内部finding转换为浏览器安全finding。
//
// 返回值保留ID、页码、维度、风险级别和教师视图；原始教案依据、页面证据、
// 代码证据、连续性证据和内部执行计划全部清空。
func BuildCWAIReviewBrowserFinding(
	finding *models.CWAIReviewFinding,
) *models.CWAIReviewFinding {
	if finding == nil {
		return nil
	}

	safe := *finding
	safe.PageNumbers = append(
		[]int{},
		finding.PageNumbers...,
	)

	normalizeCWAIReviewTeacherViewForFinding(&safe)

	snapshot := safe.TeacherViewSnapshot
	snapshot.AcceptanceChecks = append(
		[]string{},
		snapshot.AcceptanceChecks...,
	)
	safe.TeacherViewSnapshot = snapshot

	safe.Title = snapshot.TeacherTitle
	safe.Description = snapshot.WhatHappened
	safe.Suggestion = snapshot.ImprovementGoal

	safe.LessonOrOutlineBasis = ""
	safe.PageEvidence = ""
	safe.CodeEvidence = ""
	safe.ContinuityEvidence = ""
	safe.InternalExecutionPlan = ""

	return &safe
}

// BuildCWAIReviewBrowserRiskPages 将内部风险页转换为教师可读风险页。
func BuildCWAIReviewBrowserRiskPages(
	input []models.CWAIReviewRiskPage,
) []models.CWAIReviewRiskPage {
	result := make(
		[]models.CWAIReviewRiskPage,
		0,
		len(input),
	)

	for _, raw := range input {
		if raw.PageNumber <= 0 {
			continue
		}

		risk := raw
		risk.Severity = normalizeCWAIReviewSeverity(
			risk.Severity,
		)

		var fallback bool
		risk.Reason, fallback =
			cwAIReviewNormalizeTeacherText(
				risk.Reason,
				"该页面需要教师打开并完成实际操作检查。",
				cwAIReviewTeacherTextMaxRunes,
			)

		risk.EvidenceType = ""
		risk.ManualReviewRequired =
			risk.ManualReviewRequired ||
				fallback

		result = append(result, risk)
	}

	return result
}

// BuildCWAIReviewBrowserBatchResult 将批次结果收敛为浏览器安全结果。
//
// 连续性账本只供后端跨批继承，浏览器响应固定为空对象。
func BuildCWAIReviewBrowserBatchResult(
	result *models.CWAIReviewBatchAIResult,
) *models.CWAIReviewBatchAIResult {
	if result == nil {
		return nil
	}

	safe := *result
	safe.PageNumbers = append(
		[]int{},
		result.PageNumbers...,
	)

	var summaryFallback bool
	safe.BatchSummary, summaryFallback =
		cwAIReviewNormalizeTeacherText(
			result.BatchSummary,
			"本批页面已经完成检查，请结合下方问题逐项确认。",
			cwAIReviewTeacherTextMaxRunes,
		)

	safe.Findings = make(
		[]models.CWAIReviewFinding,
		0,
		len(result.Findings),
	)
	for i := range result.Findings {
		finding :=
			BuildCWAIReviewBrowserFinding(
				&result.Findings[i],
			)
		if finding == nil {
			continue
		}

		if finding.ManualReviewRequired {
			safe.ManualReviewRequired = true
		}

		safe.Findings = append(
			safe.Findings,
			*finding,
		)
	}

	safe.ContinuityLedger =
		map[string]interface{}{}

	safe.RiskPages =
		BuildCWAIReviewBrowserRiskPages(
			result.RiskPages,
		)

	if summaryFallback {
		safe.ManualReviewRequired = true
	}

	for _, risk := range safe.RiskPages {
		if risk.ManualReviewRequired {
			safe.ManualReviewRequired = true
			break
		}
	}

	return &safe
}

// BuildCWAIReviewBrowserFinalReport 将最终报告收敛为浏览器安全报告。
func BuildCWAIReviewBrowserFinalReport(
	report *models.CWAIReviewFinalReport,
) *models.CWAIReviewFinalReport {
	if report == nil {
		return nil
	}

	safe := *report

	safe.ReviewConfig.ReviewDimensions = append(
		[]string{},
		report.ReviewConfig.ReviewDimensions...,
	)
	safe.ReviewConfig.ReviewDimensionItems = append(
		[]models.CWAIReviewDimensionReportItem{},
		report.ReviewConfig.ReviewDimensionItems...,
	)

	safe.OverallRisk =
		normalizeCWAIReviewSeverity(
			report.OverallRisk,
		)

	safe.Summary, _ =
		cwAIReviewNormalizeTeacherText(
			report.Summary,
			"本次检查发现了需要教师进一步确认和完善的内容。",
			cwAIReviewTeacherTextMaxRunes,
		)

	safe.Strengths = make(
		[]string,
		0,
		len(report.Strengths),
	)
	for _, rawStrength := range report.Strengths {
		strength, fallback :=
			cwAIReviewNormalizeOptionalTeacherText(
				rawStrength,
				cwAIReviewTeacherTextMaxRunes,
			)
		if fallback || strength == "" {
			continue
		}

		safe.Strengths = append(
			safe.Strengths,
			strength,
		)
	}

	safe.Findings = make(
		[]models.CWAIReviewFinding,
		0,
		len(report.Findings),
	)
	for i := range report.Findings {
		finding :=
			BuildCWAIReviewBrowserFinding(
				&report.Findings[i],
			)
		if finding == nil {
			continue
		}

		safe.Findings = append(
			safe.Findings,
			*finding,
		)
	}

	safe.PriorityActions = make(
		[]models.CWAIReviewPriorityAction,
		0,
		len(report.PriorityActions),
	)
	for _, rawAction := range report.PriorityActions {
		action := rawAction
		action.PageNumbers = append(
			[]int{},
			rawAction.PageNumbers...,
		)

		var titleFallback bool
		action.Title, titleFallback =
			cwAIReviewNormalizeTeacherText(
				rawAction.Title,
				"优先检查并完善相关页面",
				cwAIReviewTeacherTitleMaxRunes,
			)

		var descriptionFallback bool
		action.Description, descriptionFallback =
			cwAIReviewNormalizeTeacherText(
				rawAction.Description,
				"按照问题卡中的调整目标逐页完成修改，并使用检查清单确认结果。",
				cwAIReviewTeacherTextMaxRunes,
			)

		var reasonFallback bool
		action.Reason, reasonFallback =
			cwAIReviewNormalizeOptionalTeacherText(
				rawAction.Reason,
				cwAIReviewTeacherTextMaxRunes,
			)

		action.ManualReviewRequired =
			action.ManualReviewRequired ||
				titleFallback ||
				descriptionFallback ||
				reasonFallback

		safe.PriorityActions = append(
			safe.PriorityActions,
			action,
		)
	}

	safe.ManualReviewPages = append(
		[]int{},
		report.ManualReviewPages...,
	)

	safe.ReviewCommentDraft, _ =
		cwAIReviewNormalizeOptionalTeacherText(
			report.ReviewCommentDraft,
			cwAIReviewTeacherTextMaxRunes,
		)

	safe.HumanDecisionReminder, _ =
		cwAIReviewNormalizeTeacherText(
			report.HumanDecisionReminder,
			"AI检查仅提供辅助，是否要求修改和最终审核决定仍由教师确认。",
			cwAIReviewTeacherTextMaxRunes,
		)

	return &safe
}

// BuildCWReviewItemTeacherView 从整改项内部事实构造教师视图。
//
// 新记录优先读取固化的teacher_view_snapshot；旧记录没有快照时，使用
// 整改项兼容字段和内部证据执行确定性降级，不调用AI、不回写数据库。
func BuildCWReviewItemTeacherView(
	item *models.CoursewareReviewItem,
) models.CWAIReviewTeacherViewSnapshot {
	if item == nil {
		return models.CWAIReviewTeacherViewSnapshot{
			AcceptanceChecks: []string{},
		}
	}

	finding := models.CWAIReviewFinding{
		Dimension:   item.Dimension,
		Title:       item.Title,
		Description: item.Description,
		Suggestion:  item.OriginalSuggestion,
	}

	var evidence struct {
		TeacherViewSnapshot models.CWAIReviewTeacherViewSnapshot `json:"teacher_view_snapshot"`

		RawTitle       string `json:"raw_title"`
		RawDescription string `json:"raw_description"`
		RawSuggestion  string `json:"raw_suggestion"`

		LessonOrOutlineBasis string `json:"lesson_or_outline_basis"`
		PageEvidence         string `json:"page_evidence"`
		CodeEvidence         string `json:"code_evidence"`
		ContinuityEvidence   string `json:"continuity_evidence"`

		InternalExecutionPlan string `json:"internal_execution_plan"`

		ManualReviewRequired bool `json:"manual_review_required"`
	}

	if err := json.Unmarshal(
		[]byte(item.EvidenceJSON),
		&evidence,
	); err == nil {
		finding.TeacherViewSnapshot =
			evidence.TeacherViewSnapshot

		if strings.TrimSpace(evidence.RawTitle) != "" {
			finding.Title = evidence.RawTitle
		}
		if strings.TrimSpace(evidence.RawDescription) != "" {
			finding.Description =
				evidence.RawDescription
		}
		if strings.TrimSpace(evidence.RawSuggestion) != "" {
			finding.Suggestion =
				evidence.RawSuggestion
		}

		finding.LessonOrOutlineBasis =
			evidence.LessonOrOutlineBasis
		finding.PageEvidence =
			evidence.PageEvidence
		finding.CodeEvidence =
			evidence.CodeEvidence
		finding.ContinuityEvidence =
			evidence.ContinuityEvidence
		finding.InternalExecutionPlan =
			evidence.InternalExecutionPlan
		finding.ManualReviewRequired =
			evidence.ManualReviewRequired
	}

	if item.Status == models.CWReviewItemStatusStale ||
		item.Status == models.CWReviewItemStatusOrphaned {
		finding.ManualReviewRequired = true
	}

	normalizeCWAIReviewTeacherViewForFinding(
		&finding,
	)

	snapshot := finding.TeacherViewSnapshot
	snapshot.AcceptanceChecks = append(
		[]string{},
		snapshot.AcceptanceChecks...,
	)

	return snapshot
}

// BuildCWReviewItemTeacherEvidenceJSON 构造浏览器可见的最小证据对象。
//
// 该JSON只包含教师视图与人工检查标记，不返回原始证据、配置哈希、
// 内部执行计划或模型输出。
func BuildCWReviewItemTeacherEvidenceJSON(
	snapshot models.CWAIReviewTeacherViewSnapshot,
) string {
	snapshot.AcceptanceChecks = append(
		[]string{},
		snapshot.AcceptanceChecks...,
	)

	payload := struct {
		TeacherViewSnapshot  models.CWAIReviewTeacherViewSnapshot `json:"teacher_view_snapshot"`
		ManualReviewRequired bool                                 `json:"manual_review_required"`
	}{
		TeacherViewSnapshot: snapshot,
		ManualReviewRequired: snapshot.
			ManualCheckRequired,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}

	return string(encoded)
}

type cwAIReviewTeacherDefaults struct {
	Title            string
	WhatHappened     string
	TeachingImpact   string
	ImprovementGoal  string
	AcceptanceChecks []string
}

// cwAIReviewTeacherDefaultsForDimension 提供不依赖AI的安全教师表达。
func cwAIReviewTeacherDefaultsForDimension(
	dimension string,
) cwAIReviewTeacherDefaults {
	switch strings.TrimSpace(dimension) {
	case models.CWAIReviewDimensionTechnicalImplementation,
		models.CWAIReviewDimensionInteractionExperience,
		models.CWAIReviewDimensionOperationalUsability:
		return cwAIReviewTeacherDefaults{
			Title:           "这一页的课堂操作可能不够稳定",
			WhatHappened:    "页面中的部分操作可能不能稳定呈现预期结果。",
			TeachingImpact:  "讲解或学生操作时可能出现中断，影响课堂节奏和参与。",
			ImprovementGoal: "让页面中的关键操作能够稳定完成，并在每次操作后给出清楚的可见反馈。",
			AcceptanceChecks: []string{
				"逐一操作页面上的主要按钮，确认每次都有清楚且一致的结果。",
				"重复操作并切换前后步骤，确认页面不会中断或停留在错误状态。",
				"在实际播放环境中完整演示一次，确认讲解流程可以顺利完成。",
			},
		}

	case models.CWAIReviewDimensionPageReadability:
		return cwAIReviewTeacherDefaults{
			Title:           "这一页的内容可能不够清楚",
			WhatHappened:    "部分文字、图示或重点信息可能不容易快速看清。",
			TeachingImpact:  "投影或后排观看时可能增加阅读负担，影响学生跟随讲解。",
			ImprovementGoal: "提高重点内容的清晰度和层级，让学生能够快速找到需要关注的信息。",
			AcceptanceChecks: []string{
				"在常用投影尺寸下查看页面，确认主要文字和图示都能清楚辨认。",
				"确认标题、重点和补充内容的层级清楚，不需要反复寻找。",
				"检查文字与背景的区分度，确认后排观看时仍然易读。",
			},
		}

	case models.CWAIReviewDimensionKnowledgeAccuracy:
		return cwAIReviewTeacherDefaults{
			Title:           "这一页的知识表述需要进一步核对",
			WhatHappened:    "页面中的概念、事实、公式或结论可能存在不够准确或不够完整的地方。",
			TeachingImpact:  "学生可能形成错误理解，或在后续练习中使用不准确的结论。",
			ImprovementGoal: "核对知识表述和结论，使页面内容准确、完整并符合当前学习层级。",
			AcceptanceChecks: []string{
				"逐项核对页面中的概念、事实、公式和结论是否准确。",
				"确认例子、答案和解释之间没有互相矛盾。",
				"确认表述难度符合当前年级，且不会产生明显歧义。",
			},
		}

	case models.CWAIReviewDimensionLessonAlignment:
		return cwAIReviewTeacherDefaults{
			Title:           "这一页与本课教学目标需要再核对",
			WhatHappened:    "当前内容或活动与本课重点、教学环节或知识边界可能不完全一致。",
			TeachingImpact:  "讲解重点可能被分散，学生不容易理解这一页与整节课的关系。",
			ImprovementGoal: "让本页内容服务于本课核心目标，并与前后教学环节自然衔接。",
			AcceptanceChecks: []string{
				"确认本页内容能够直接支持本课的核心教学目标。",
				"确认页面活动与前后教学环节衔接自然，不会突然改变学习重点。",
				"确认没有加入超出本课知识边界且无法解释的内容。",
			},
		}

	case models.CWAIReviewDimensionTeachingLogic:
		return cwAIReviewTeacherDefaults{
			Title:           "这一页的讲解顺序可能不够连贯",
			WhatHappened:    "页面内容、案例或活动的先后关系可能不够清楚。",
			TeachingImpact:  "学生可能难以跟上推理过程，教师也需要额外解释页面之间的关系。",
			ImprovementGoal: "理清讲解顺序和前后关系，让学生能够沿着清楚的学习路径理解内容。",
			AcceptanceChecks: []string{
				"从上一页进入本页时，确认学生能够理解为什么学习这一内容。",
				"确认本页内容按照由浅入深或由问题到结论的顺序展开。",
				"进入下一页前，确认本页已经形成清楚的阶段结论。",
			},
		}

	case models.CWAIReviewDimensionAuthenticity:
		return cwAIReviewTeacherDefaults{
			Title:           "这一页的情境或材料需要进一步核对",
			WhatHappened:    "页面中的案例、数据、人物或情境可能缺少足够依据。",
			TeachingImpact:  "学生可能把不可靠的材料当作事实，影响对知识和真实情境的理解。",
			ImprovementGoal: "使用来源清楚、符合实际且与教学目标相关的材料。",
			AcceptanceChecks: []string{
				"核对页面中的数据、案例和结论是否有可靠依据。",
				"确认情境描述符合真实情况，不会造成明显误解。",
				"确认材料与当前教学目标相关，而不是仅作装饰。",
			},
		}

	default:
		return cwAIReviewTeacherDefaults{
			Title:           "这一页需要进一步检查和完善",
			WhatHappened:    "页面当前呈现可能还没有完全达到预期的教学效果。",
			TeachingImpact:  "可能影响学生理解、课堂操作或教师讲解，需要进一步确认。",
			ImprovementGoal: "围绕本页教学目标完成必要调整，并保持其他已符合要求的内容不变。",
			AcceptanceChecks: []string{
				"打开当前页面，确认主要内容和操作结果符合本页教学目标。",
				"完整演示一次本页内容，确认讲解和学生操作可以顺利进行。",
				"确认调整没有影响页面中原本正确且需要保留的内容。",
			},
		}
	}
}

// cwAIReviewFirstSafeTeacherText 优先使用不含平台术语的历史内容。
func cwAIReviewFirstSafeTeacherText(
	candidate string,
	fallback string,
) string {
	candidate = cwAIReviewCollapseTeacherWhitespace(candidate)

	if candidate == "" ||
		cwAIReviewTeacherViewContainsPlatformTerm(candidate) {
		return fallback
	}

	return candidate
}

// cwAIReviewNormalizeTeacherText 规范必填教师字段。
//
// 第二个返回值表示使用了安全降级，调用方应把问题标记为需要人工检查。
func cwAIReviewNormalizeTeacherText(
	value string,
	fallback string,
	maxRunes int,
) (string, bool) {
	value = cwAIReviewCollapseTeacherWhitespace(value)
	fallback = cwAIReviewCollapseTeacherWhitespace(fallback)

	if value == "" ||
		cwAIReviewTeacherViewContainsPlatformTerm(value) {
		return cwAIReviewTruncateTeacherText(
			fallback,
			maxRunes,
		), true
	}

	return cwAIReviewTruncateTeacherText(
		value,
		maxRunes,
	), false
}

// cwAIReviewNormalizeOptionalTeacherText 规范允许为空的教师字段。
func cwAIReviewNormalizeOptionalTeacherText(
	value string,
	maxRunes int,
) (string, bool) {
	value = cwAIReviewCollapseTeacherWhitespace(value)

	if value == "" {
		return "", false
	}

	if cwAIReviewTeacherViewContainsPlatformTerm(value) {
		return "", true
	}

	return cwAIReviewTruncateTeacherText(
		value,
		maxRunes,
	), false
}

// cwAIReviewNormalizeTeacherChecks 形成2至5条可观察检查项。
func cwAIReviewNormalizeTeacherChecks(
	input []string,
	defaults []string,
	manualCheckRequired bool,
) ([]string, bool) {
	result := make([]string, 0, cwAIReviewTeacherCheckMax)
	seen := make(map[string]bool)
	usedFallback := false

	appendCheck := func(raw string, fallback bool) {
		value := cwAIReviewCollapseTeacherWhitespace(
			strings.TrimLeft(
				strings.TrimSpace(raw),
				"-*•·0123456789.、)） ",
			),
		)

		if value == "" {
			return
		}

		if cwAIReviewTeacherViewContainsPlatformTerm(value) {
			usedFallback = true
			return
		}

		value = cwAIReviewTruncateTeacherText(
			value,
			cwAIReviewTeacherCheckMaxRunes,
		)

		key := strings.ToLower(value)
		if seen[key] {
			return
		}

		seen[key] = true
		result = append(result, value)

		if fallback {
			usedFallback = true
		}
	}

	for _, check := range input {
		appendCheck(check, false)

		if len(result) >= cwAIReviewTeacherCheckMax {
			break
		}
	}

	for _, check := range defaults {
		if len(result) >= cwAIReviewTeacherCheckMin {
			break
		}

		appendCheck(check, true)
	}

	if manualCheckRequired &&
		len(result) < cwAIReviewTeacherCheckMax {
		appendCheck(
			cwAIReviewManualBrowserCheck,
			true,
		)
	}

	for _, check := range defaults {
		if len(result) >= cwAIReviewTeacherCheckMin {
			break
		}

		appendCheck(check, true)
	}

	if len(result) > cwAIReviewTeacherCheckMax {
		result = result[:cwAIReviewTeacherCheckMax]
	}

	return result, usedFallback
}

// cwAIReviewTeacherViewContainsPlatformTerm 判断教师字段是否含平台实现术语。
func cwAIReviewTeacherViewContainsPlatformTerm(
	value string,
) bool {
	normalized := strings.ToLower(
		strings.TrimSpace(value),
	)

	if normalized == "" {
		return false
	}

	for _, term := range cwAIReviewTeacherPlatformTerms {
		if strings.Contains(
			normalized,
			strings.ToLower(term),
		) {
			return true
		}
	}

	return false
}

// cwAIReviewCollapseTeacherWhitespace 清理代码围栏和连续空白。
func cwAIReviewCollapseTeacherWhitespace(
	value string,
) string {
	value = strings.ReplaceAll(value, "`", "")
	return strings.Join(
		strings.Fields(value),
		" ",
	)
}

// cwAIReviewTruncateTeacherText 按Unicode字符截断并保留省略标记。
func cwAIReviewTruncateTeacherText(
	value string,
	maxRunes int,
) string {
	value = strings.TrimSpace(value)

	if maxRunes <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}

	if maxRunes == 1 {
		return string(runes[:1])
	}

	return strings.TrimSpace(
		string(runes[:maxRunes-1]),
	) + "…"
}
