package services

// courseware_ai_review_result_normalize.go
//
// 课件AI审核模型输出解析、R-02维度收敛、教师视图快照归一化
// 与最终报告配置构造。
//
// 安全原则：
//   - 模型不能决定本次允许使用的审核维度；
//   - finding.dimension必须属于会话不可变配置；
//   - 非法、未选择或旧协议维度由后端映射；
//   - 后端修正过的finding强制要求人工复核；
//   - no_lesson模式清空模型生成的教案或大纲依据；
//   - 教师视图快照必须经过后端确定性清洗和完整性补齐；
//   - 最终报告review_config只从数据库会话生成。

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
)

// parseCWAIReviewBatchResult 解析并规范模型的单批输出。
func parseCWAIReviewBatchResult(
	raw string,
	session *models.CoursewareAIReviewSession,
	expectedBatchNo int,
	expectedPageNumbers []int,
	continuityBeforeJSON string,
) (
	*models.CWAIReviewBatchAIResult,
	string,
	string,
	string,
	error,
) {
	if session == nil {
		return nil, "", "", "", errors.New(
			"缺少课件AI审核会话",
		)
	}

	if _, err := cwAIReviewConfigFromSession(session); err != nil {
		return nil, "", "", "", err
	}

	jsonText, ok := ai.ExtractJSON(raw)
	if !ok || strings.TrimSpace(jsonText) == "" {
		jsonText = strings.TrimSpace(raw)
	}

	var result models.CWAIReviewBatchAIResult
	if err := json.Unmarshal(
		[]byte(jsonText),
		&result,
	); err != nil {
		return nil, "", "", "", fmt.Errorf(
			"解析课件AI审核批次JSON失败: %w",
			err,
		)
	}

	result.BatchNo = expectedBatchNo
	result.PageNumbers = append(
		[]int{},
		expectedPageNumbers...,
	)

	if result.Findings == nil {
		result.Findings =
			[]models.CWAIReviewFinding{}
	}
	if result.RiskPages == nil {
		result.RiskPages =
			[]models.CWAIReviewRiskPage{}
	}
	if result.ContinuityLedger == nil {
		result.ContinuityLedger =
			map[string]interface{}{}
	}

	for i := range result.Findings {
		finding := &result.Findings[i]

		if strings.TrimSpace(finding.ID) == "" {
			finding.ID = fmt.Sprintf(
				"B%d-F%d",
				expectedBatchNo,
				i+1,
			)
		}

		finding.Severity =
			normalizeCWAIReviewSeverity(
				finding.Severity,
			)

		finding.PageNumbers =
			normalizeCWAIReviewPageNumbers(
				finding.PageNumbers,
			)

		if finding.Confidence < 0 {
			finding.Confidence = 0
		}
		if finding.Confidence > 100 {
			finding.Confidence = 100
		}

		if err := normalizeCWAIReviewFindingForSession(
			session,
			finding,
		); err != nil {
			return nil, "", "", "", err
		}
	}

	normalizedRiskPages := make(
		[]models.CWAIReviewRiskPage,
		0,
		len(result.RiskPages),
	)

	for i := range result.RiskPages {
		riskPage := result.RiskPages[i]
		if riskPage.PageNumber <= 0 {
			continue
		}

		riskPage.Severity =
			normalizeCWAIReviewSeverity(
				riskPage.Severity,
			)

		normalizedRiskPages = append(
			normalizedRiskPages,
			riskPage,
		)
	}

	result.RiskPages = normalizedRiskPages

	previousLedger := cwAIReviewDecodeJSON(
		continuityBeforeJSON,
		map[string]interface{}{},
	)

	previousMap, _ :=
		previousLedger.(map[string]interface{})

	mergedLedger :=
		mergeCWAIReviewContinuityLedger(
			previousMap,
			result.ContinuityLedger,
		)

	result.ContinuityLedger = mergedLedger

	normalizedResultJSON, err :=
		json.Marshal(&result)
	if err != nil {
		return nil, "", "", "", fmt.Errorf(
			"序列化课件AI审核批次结果失败: %w",
			err,
		)
	}

	continuityJSON, err :=
		json.Marshal(mergedLedger)
	if err != nil {
		return nil, "", "", "", fmt.Errorf(
			"序列化课件AI审核连续性账本失败: %w",
			err,
		)
	}

	riskPagesJSON, err :=
		json.Marshal(result.RiskPages)
	if err != nil {
		return nil, "", "", "", fmt.Errorf(
			"序列化课件AI审核风险页失败: %w",
			err,
		)
	}

	return &result,
		string(normalizedResultJSON),
		string(continuityJSON),
		string(riskPagesJSON),
		nil
}

// normalizeCWAIReviewFindingForSession 把finding收敛到会话已选择维度，
// 并形成可以固化的教师视图快照。
func normalizeCWAIReviewFindingForSession(
	session *models.CoursewareAIReviewSession,
	finding *models.CWAIReviewFinding,
) error {
	if finding == nil {
		return nil
	}

	config, err := cwAIReviewConfigFromSession(session)
	if err != nil {
		return err
	}

	selected := make(
		map[string]bool,
		len(config.ReviewDimensions),
	)
	for _, dimension := range config.ReviewDimensions {
		selected[dimension] = true
	}

	rawDimension := strings.ToLower(
		strings.TrimSpace(finding.Dimension),
	)

	if selected[rawDimension] {
		finding.Dimension = rawDimension
	} else {
		candidates :=
			cwAIReviewDimensionCandidates(
				rawDimension,
				finding,
			)

		mappedDimension := ""
		for _, candidate := range candidates {
			if selected[candidate] {
				mappedDimension = candidate
				break
			}
		}

		if mappedDimension == "" &&
			selected[models.CWAIReviewDimensionCustom] {
			mappedDimension =
				models.CWAIReviewDimensionCustom
		}

		if mappedDimension == "" {
			mappedDimension =
				config.ReviewDimensions[0]
		}

		finding.Dimension = mappedDimension
		finding.ManualReviewRequired = true
	}

	if config.LessonReferenceMode ==
		models.CWAIReviewLessonReferenceNoLesson {
		finding.LessonOrOutlineBasis = ""
	}

	// 教师字段不能直接信任模型输出。
	//
	// 本步骤会清理平台术语、补齐缺失字段、形成2至5条检查项，
	// 并在无法可靠转译时强制要求人工检查。
	normalizeCWAIReviewTeacherViewForFinding(finding)

	return nil
}

// cwAIReviewDimensionCandidates 生成旧维度或自由文本的候选映射顺序。
func cwAIReviewDimensionCandidates(
	rawDimension string,
	finding *models.CWAIReviewFinding,
) []string {
	candidates := make([]string, 0, 12)

	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}

		for _, existing := range candidates {
			if existing == candidate {
				return
			}
		}

		candidates = append(candidates, candidate)
	}

	switch rawDimension {
	case models.CWAIReviewDimensionTeachingLogic:
		appendCandidate(
			models.CWAIReviewDimensionTeachingLogic,
		)

	case models.CWAIReviewDimensionTechnicalImplementation:
		appendCandidate(
			models.CWAIReviewDimensionTechnicalImplementation,
		)

	case models.CWAIReviewDimensionInteractionExperience:
		appendCandidate(
			models.CWAIReviewDimensionInteractionExperience,
		)

	case models.CWAIReviewDimensionLessonAlignment:
		appendCandidate(
			models.CWAIReviewDimensionLessonAlignment,
		)

	case models.CWAIReviewDimensionAuthenticity:
		appendCandidate(
			models.CWAIReviewDimensionAuthenticity,
		)

	case models.CWAIReviewDimensionKnowledgeAccuracy:
		appendCandidate(
			models.CWAIReviewDimensionKnowledgeAccuracy,
		)

	case models.CWAIReviewDimensionPageReadability:
		appendCandidate(
			models.CWAIReviewDimensionPageReadability,
		)

	case models.CWAIReviewDimensionOperationalUsability:
		appendCandidate(
			models.CWAIReviewDimensionOperationalUsability,
		)

	case models.CWAIReviewDimensionCustom:
		appendCandidate(
			models.CWAIReviewDimensionCustom,
		)

	case "grade_fit":
		appendCandidate(
			models.CWAIReviewDimensionTeachingLogic,
		)
		appendCandidate(
			models.CWAIReviewDimensionKnowledgeAccuracy,
		)
		appendCandidate(
			models.CWAIReviewDimensionLessonAlignment,
		)

	case "continuity":
		appendCandidate(
			models.CWAIReviewDimensionTeachingLogic,
		)

	case "interaction":
		appendCandidate(
			models.CWAIReviewDimensionInteractionExperience,
		)
		appendCandidate(
			models.CWAIReviewDimensionTechnicalImplementation,
		)
		appendCandidate(
			models.CWAIReviewDimensionOperationalUsability,
		)

	case "answer_exposure":
		appendCandidate(
			models.CWAIReviewDimensionInteractionExperience,
		)
		appendCandidate(
			models.CWAIReviewDimensionOperationalUsability,
		)
		appendCandidate(
			models.CWAIReviewDimensionTechnicalImplementation,
		)

	case "feedback":
		appendCandidate(
			models.CWAIReviewDimensionInteractionExperience,
		)
		appendCandidate(
			models.CWAIReviewDimensionOperationalUsability,
		)
		appendCandidate(
			models.CWAIReviewDimensionTeachingLogic,
		)

	case "visual_load":
		appendCandidate(
			models.CWAIReviewDimensionPageReadability,
		)

	case "media":
		appendCandidate(
			models.CWAIReviewDimensionAuthenticity,
		)
		appendCandidate(
			models.CWAIReviewDimensionPageReadability,
		)
		appendCandidate(
			models.CWAIReviewDimensionTechnicalImplementation,
		)

	case "runtime_dependency":
		appendCandidate(
			models.CWAIReviewDimensionTechnicalImplementation,
		)
		appendCandidate(
			models.CWAIReviewDimensionOperationalUsability,
		)
	}

	searchText := cwAIReviewFindingSearchText(finding)

	if containsAnyCWAIReviewText(
		searchText,
		"教案",
		"大纲",
		"教学目标",
		"lesson",
		"outline",
		"alignment",
	) {
		appendCandidate(
			models.CWAIReviewDimensionLessonAlignment,
		)
	}

	if containsAnyCWAIReviewText(
		searchText,
		"html",
		"css",
		"javascript",
		"script",
		"函数",
		"代码",
		"加载",
		"依赖",
		"运行时",
		"dom",
	) {
		appendCandidate(
			models.CWAIReviewDimensionTechnicalImplementation,
		)
	}

	if containsAnyCWAIReviewText(
		searchText,
		"互动",
		"交互",
		"点击",
		"按钮",
		"拖拽",
		"反馈",
		"答案暴露",
		"interaction",
	) {
		appendCandidate(
			models.CWAIReviewDimensionInteractionExperience,
		)
	}

	if containsAnyCWAIReviewText(
		searchText,
		"可用",
		"操作",
		"触控",
		"键盘",
		"无法使用",
		"不可达",
		"易用",
		"usability",
	) {
		appendCandidate(
			models.CWAIReviewDimensionOperationalUsability,
		)
	}

	if containsAnyCWAIReviewText(
		searchText,
		"知识",
		"概念",
		"公式",
		"事实",
		"结论错误",
		"科学性",
		"accuracy",
	) {
		appendCandidate(
			models.CWAIReviewDimensionKnowledgeAccuracy,
		)
	}

	if containsAnyCWAIReviewText(
		searchText,
		"真实",
		"真实性",
		"数据来源",
		"案例来源",
		"情境失真",
		"authentic",
	) {
		appendCandidate(
			models.CWAIReviewDimensionAuthenticity,
		)
	}

	if containsAnyCWAIReviewText(
		searchText,
		"可读",
		"字号",
		"排版",
		"拥挤",
		"对比度",
		"视觉负担",
		"readability",
	) {
		appendCandidate(
			models.CWAIReviewDimensionPageReadability,
		)
	}

	if containsAnyCWAIReviewText(
		searchText,
		"逻辑",
		"衔接",
		"流程",
		"节奏",
		"顺序",
		"前后矛盾",
		"continuity",
	) {
		appendCandidate(
			models.CWAIReviewDimensionTeachingLogic,
		)
	}

	return candidates
}

func cwAIReviewFindingSearchText(
	finding *models.CWAIReviewFinding,
) string {
	if finding == nil {
		return ""
	}

	return strings.ToLower(
		strings.Join(
			[]string{
				finding.Title,
				finding.Description,
				finding.TeacherViewSnapshot.TeacherTitle,
				finding.TeacherViewSnapshot.WhatHappened,
				finding.TeacherViewSnapshot.TeachingImpact,
				finding.TeacherViewSnapshot.ImprovementGoal,
				finding.LessonOrOutlineBasis,
				finding.PageEvidence,
				finding.CodeEvidence,
				finding.ContinuityEvidence,
				finding.Suggestion,
				finding.InternalExecutionPlan,
			},
			"\n",
		),
	)
}

func containsAnyCWAIReviewText(
	text string,
	keywords ...string,
) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}

	for _, keyword := range keywords {
		if strings.Contains(
			text,
			strings.ToLower(keyword),
		) {
			return true
		}
	}

	return false
}

// buildCWAIReviewConfigReport 从会话配置构造最终报告配置事实。
func buildCWAIReviewConfigReport(
	session *models.CoursewareAIReviewSession,
) (models.CWAIReviewConfigReport, error) {
	config, err := cwAIReviewConfigFromSession(session)
	if err != nil {
		return models.CWAIReviewConfigReport{}, err
	}

	items := make(
		[]models.CWAIReviewDimensionReportItem,
		0,
		len(config.ReviewDimensions),
	)

	for _, dimension := range config.ReviewDimensions {
		items = append(
			items,
			models.CWAIReviewDimensionReportItem{
				Code: dimension,
				Label: cwAIReviewDimensionLabel(
					dimension,
				),
			},
		)
	}

	return models.CWAIReviewConfigReport{
		SchemaVersion: config.SchemaVersion,

		ReviewDimensions: append(
			[]string{},
			config.ReviewDimensions...,
		),
		ReviewDimensionItems: items,

		CustomDimensionDescription: config.
			CustomDimensionDescription,

		LessonReferenceMode: config.LessonReferenceMode,
		LessonReferenceLabel: cwAIReviewLessonReferenceLabel(
			config.LessonReferenceMode,
		),
		UsesLessonMaterials: cwAIReviewUsesLessonMaterials(
			config,
		),

		ReviewConfigHash: strings.TrimSpace(
			session.ReviewConfigHash,
		),
	}, nil
}
