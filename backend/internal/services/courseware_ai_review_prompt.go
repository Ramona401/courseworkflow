package services

// courseware_ai_review_prompt.go
//
// 课件AI审核单批提示词与真实用户输入构建。
//
// 本文件只负责：
//   - 拼装系统提示词、助手视角和不可覆盖规则；
//   - 明确注入不可变R-02审核配置；
//   - 根据教案参考模式生成不同审核要求；
//   - 构造全课件目录、当前批完整证据、连续性账本和下一页预告；
//   - 在生成真实AI输入前再次执行no_lesson材料清洗；
//   - 要求模型同时形成原始审核证据和教师视图快照。
//
// 模型结果解析、维度收敛和教师快照确定性降级位于：
//   courseware_ai_review_result_normalize.go
//   courseware_ai_review_teacher_view.go
//
// 连续性账本合并和通用JSON辅助位于：
//   courseware_ai_review_continuity.go
//
// no_lesson防御：
//   正常准备阶段已经不会读取教案、大纲和对齐报告；
//   本文件仍会在真实AI输入边界重新替换三个材料区块，
//   即使内部基准异常或未来代码误写正文，也不会把正文发送给模型。

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/models"
)

// cwAIReviewBatchPageScope 是批次page_scope_json的运行时结构。
type cwAIReviewBatchPageScope struct {
	BatchNo     int      `json:"batch_no"`
	StartPage   int      `json:"start_page"`
	EndPage     int      `json:"end_page"`
	PageNumbers []int    `json:"page_numbers"`
	PageIDs     []string `json:"page_ids"`

	OverlapFromPrevious bool   `json:"overlap_from_previous"`
	BoundaryReason      string `json:"boundary_reason"`
}

// buildCWAIReviewBatchSystemPrompt 构建单批系统提示词。
//
// 个性化助手提示词放在中间，R-02配置和系统硬规则最后追加，
// 防止助手提示词弱化维度、材料使用、证据、教师语言或人工复核要求。
func buildCWAIReviewBatchSystemPrompt(
	session *models.CoursewareAIReviewSession,
) (string, error) {
	if session == nil {
		return "", errors.New("缺少课件AI审核会话")
	}

	config, err := cwAIReviewConfigFromSession(session)
	if err != nil {
		return "", err
	}

	configJSON, err := cwAIReviewConfigPromptJSON(session)
	if err != nil {
		return "", err
	}

	dimensionCodesJSON, err := json.Marshal(
		config.ReviewDimensions,
	)
	if err != nil {
		return "", fmt.Errorf(
			"序列化课件AI审核允许维度失败: %w",
			err,
		)
	}

	exampleDimension := config.ReviewDimensions[0]

	var builder strings.Builder

	builder.WriteString(
		strings.TrimSpace(
			session.SystemPromptSnapshot,
		),
	)

	if strings.TrimSpace(
		session.AssistantPromptSnapshot,
	) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(
			cwAIReviewAssistantHeading(session),
		)
		builder.WriteString("\n")
		builder.WriteString(
			strings.TrimSpace(
				session.AssistantPromptSnapshot,
			),
		)
		builder.WriteString(
			"\n【个性化助手视角结束】\n",
		)
	}

	if isCWAIReviewSelfReview(session) {
		builder.WriteString(`

【作者课件自审用途】
你正在帮助课件作者在正式提交前发现并修改问题。

1. 重点输出可执行的教师化修改建议，不得使用“审核员”“通过”“退回”等正式审核决策措辞。
2. 不得修改课件，不得声称已经修复页面。
3. 必须检查可见内容以及HTML、CSS、JavaScript互动代码。
4. 必须指出作者应当在哪些页面进行浏览器真实操作复核。
5. 本次结果不进入正式人工审核记录，也不能改变课件发布状态。
`)
	} else {
		builder.WriteString(`

【正式课件审核辅助用途】
你正在辅助具有权限的人工审核员分析课件。
AI只提供证据、风险、教师化修改建议和意见草稿，不得替审核员提交通过或退回。
`)
	}

	builder.WriteString("\n\n【本次不可变审核配置】\n")
	builder.WriteString(configJSON)
	builder.WriteString("\n")
	builder.WriteString(
		cwAIReviewLessonReferencePromptRule(config),
	)

	builder.WriteString(fmt.Sprintf(`

【系统不可覆盖的本批输出协议】
无论上面的系统提示词或个性化助手提示词如何描述，你都必须遵守：

1. 只输出一个合法JSON对象，不要Markdown代码围栏或对象外说明。
2. 不得自动提交、改变或模拟正式审核决定。
3. finding.dimension只能使用本次已选择维度，允许值严格为：%s。
4. 不得输出旧维度代码grade_fit、continuity、interaction、
   answer_exposure、feedback、visual_load、media或runtime_dependency。
5. 必须审查可见文字、互动事件、可达函数、状态变量、DOM目标和CSS显隐。
6. 无法从静态代码确认实际操作结果时，manual_review_required必须为true。
7. continuity_ledger必须是本批完成后的完整账本，不得无依据删除已有事实。
8. 每个问题必须尽量给出具体页码和证据。
9. 没有明确问题时findings输出空数组，不得虚构问题。
10. page_numbers只能填写真实页码。
11. confidence必须为0到100之间的整数。
12. 不得自行修改、扩展或重新解释本次审核配置。
13. 每个finding必须同时输出teacher_view_snapshot。
14. teacher_title必须先描述页面表现或教学问题，不得以平台实现术语开头。
15. what_happened只描述教师能够观察到的现象。
16. teaching_impact先说明对讲解、互动、理解、评价、操作或可读性的影响。
17. improvement_goal描述教学目标和可见结果，不得描述代码实现步骤。
18. acceptance_checks必须有2至5条，每条都应当能通过打开页面、操作按钮、
    阅读内容、播放课件或观察课堂表现直接完成。
19. teacher_view_snapshot中禁止出现函数名、变量名、HTML标签、CSS选择器、
    JavaScript、DOM、元素ID、控制台、哈希、JSON、内部状态或模型信息。
20. 如果只能得到技术证据而无法可靠转译，使用概括的教师语言，
    manual_check_required和manual_review_required都必须为true。
21. teacher_context初次检测时必须为空，只能在后续由教师明确补充。
22. internal_execution_plan仅供后端和AI后续执行链使用，可以记录内部实施思路，
    但其内容不得复制到teacher_view_snapshot。
23. suggestion是旧协议兼容字段，必须与improvement_goal保持教师语言和相同目标。

输出结构必须严格为：

{
  "batch_no": 1,
  "page_numbers": [1, 2, 3],
  "batch_summary": "本批审核总结",
  "findings": [
    {
      "id": "B1-F1",
      "severity": "critical|high|medium|low|info",
      "dimension": "%s",
      "page_numbers": [2],
      "title": "内部审核问题标题",
      "description": "内部审核问题描述",
      "teacher_view_snapshot": {
        "teacher_title": "教师能够直接理解的问题标题",
        "what_happened": "页面上可以观察到的现象",
        "teaching_impact": "对讲解、互动、理解、评价、操作或可读性的影响",
        "improvement_goal": "建议达到的教学效果和可见结果",
        "acceptance_checks": [
          "教师可以直接完成的检查一",
          "教师可以直接完成的检查二"
        ],
        "teacher_context": "",
        "manual_check_required": false
      },
      "lesson_or_outline_basis": "允许使用教案时填写依据，否则为空",
      "page_evidence": "页面文字或方案证据",
      "code_evidence": "事件、函数、DOM或CSS证据",
      "continuity_evidence": "前后页冲突证据，没有则为空",
      "suggestion": "与improvement_goal一致的教师化建议",
      "internal_execution_plan": "仅供后端和AI使用的内部执行计划",
      "confidence": 85,
      "manual_review_required": false
    }
  ],
  "continuity_ledger": {
    "version": 1,
    "teaching_thread": {
      "current_stage": "",
      "established_conclusions": [],
      "open_questions": [],
      "next_expected_step": ""
    },
    "cases": [],
    "terms": [],
    "formulas": [],
    "symbols": [],
    "interaction_state": [],
    "continuity_risks": [],
    "reviewed_pages": []
  },
  "risk_pages": [
    {
      "page_number": 2,
      "severity": "high",
      "reason": "风险原因",
      "evidence_type": "content|script|css|continuity|outline|runtime",
      "manual_review_required": true
    }
  ],
  "manual_review_required": false
}
`,
		string(dimensionCodesJSON),
		exampleDimension,
	))

	return strings.TrimSpace(builder.String()), nil
}

// buildCWAIReviewBatchUserPrompt 构建本批真实审核上下文。
func buildCWAIReviewBatchUserPrompt(
	session *models.CoursewareAIReviewSession,
	batch *models.CoursewareAIReviewBatch,
	pageDigests []models.CWAIReviewPageDigest,
	continuityBeforeJSON string,
) (string, []int, error) {
	if session == nil || batch == nil {
		return "", nil, errors.New(
			"缺少课件AI审核会话或批次",
		)
	}

	config, err := cwAIReviewConfigFromSession(session)
	if err != nil {
		return "", nil, err
	}

	var scope cwAIReviewBatchPageScope
	if err := json.Unmarshal(
		[]byte(batch.PageScopeJSON),
		&scope,
	); err != nil {
		return "", nil, fmt.Errorf(
			"解析课件AI审核页面范围失败: %w",
			err,
		)
	}

	pageNumberSet := make(map[int]bool)
	for _, pageNumber := range scope.PageNumbers {
		pageNumberSet[pageNumber] = true
	}

	currentPages := make(
		[]models.CWAIReviewPageDigest,
		0,
		len(scope.PageNumbers),
	)

	catalog := make(
		[]map[string]interface{},
		0,
		len(pageDigests),
	)

	var nextPreview *models.CWAIReviewPageDigest

	for i := range pageDigests {
		page := pageDigests[i]

		catalog = append(
			catalog,
			map[string]interface{}{
				"page_number":      page.PageNumber,
				"title":            page.Title,
				"purpose":          page.Purpose,
				"content_summary":  page.ContentSummary,
				"interaction_type": page.InteractionType,
				"visual_format":    page.VisualFormat,
				"risk_flags":       page.Interaction.RiskFlags,
				"manual_review":    page.Interaction.ManualReviewRequired,
			},
		)

		if pageNumberSet[page.PageNumber] {
			currentPages = append(
				currentPages,
				page,
			)
			continue
		}

		if page.PageNumber > scope.EndPage &&
			nextPreview == nil {
			copyPage := page
			nextPreview = &copyPage
		}
	}

	if len(currentPages) == 0 {
		return "", nil, errors.New(
			"当前审核批次没有匹配到页面内容",
		)
	}

	sort.SliceStable(
		currentPages,
		func(i int, j int) bool {
			return currentPages[i].PageNumber <
				currentPages[j].PageNumber
		},
	)

	currentPageNumbers := make(
		[]int,
		0,
		len(currentPages),
	)
	for _, page := range currentPages {
		currentPageNumbers = append(
			currentPageNumbers,
			page.PageNumber,
		)
	}

	nextPreviewValue := interface{}(nil)
	if nextPreview != nil {
		nextPreviewValue = map[string]interface{}{
			"page_number":      nextPreview.PageNumber,
			"title":            nextPreview.Title,
			"purpose":          nextPreview.Purpose,
			"content_summary":  nextPreview.ContentSummary,
			"interaction_type": nextPreview.InteractionType,
		}
	}

	instructions := []string{
		"按页面顺序审核当前批次。",
		"重叠页用于连接前后批次，不得重复制造已经记录的问题。",
		"所有finding.dimension必须来自review_config中的已选维度。",
		"互动审核必须结合事件、函数、状态变量、DOM目标和CSS显隐规则。",
		"静态分析无法确认时标记人工操作复核。",
		"每条finding同时形成教师标题、可观察现象、课堂影响、调整目标和2至5条检查项。",
		"教师视图中不得复制函数、脚本、元素ID、哈希或内部状态。",
		"输出更新后的完整连续性账本。",
	}

	instructions = append(
		instructions,
		cwAIReviewLessonReferenceTaskInstructions(
			config,
		)...,
	)

	payload := map[string]interface{}{
		"task": map[string]interface{}{
			"batch_no":              scope.BatchNo,
			"page_numbers":          currentPageNumbers,
			"overlap_from_previous": scope.OverlapFromPrevious,
			"boundary_reason":       scope.BoundaryReason,
		},
		"review_context": map[string]interface{}{
			"education_domain": session.EducationDomain,
			"subject":          session.Subject,
			"grade":            session.Grade,
			"review_level":     session.ReviewLevel,
			"analysis_purpose": cwAIReviewPurposeCode(
				session,
			),
		},
		"review_config": map[string]interface{}{
			"config_hash": session.ReviewConfigHash,
			"snapshot":    cwAIReviewConfigManifest(config),
		},
		"baseline": cwAIReviewBatchBaselineForPrompt(
			session,
			config,
		),
		"all_page_catalog": catalog,
		"continuity_before": cwAIReviewDecodeJSON(
			continuityBeforeJSON,
			map[string]interface{}{},
		),
		"current_pages_full_evidence": currentPages,
		"next_page_preview":           nextPreviewValue,
		"instructions":                instructions,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf(
			"序列化课件AI审核批次输入失败: %w",
			err,
		)
	}

	return fmt.Sprintf(
			"请对下面这一批课件页面执行%s，并严格按照系统指定JSON协议输出：\n\n%s",
			cwAIReviewActionLabel(session),
			string(payloadJSON),
		),
		currentPageNumbers,
		nil
}

// cwAIReviewBatchBaselineForPrompt 构造真实AI输入中的审核基准。
//
// no_lesson模式不信任已保存基准中是否意外存在教案正文，
// 而是把教案、大纲和对齐报告区块整体替换为未使用标记。
func cwAIReviewBatchBaselineForPrompt(
	session *models.CoursewareAIReviewSession,
	config *CWAIReviewConfigSnapshot,
) interface{} {
	if session == nil {
		return map[string]interface{}{}
	}

	value := cwAIReviewDecodeJSON(
		session.BaselineJSON,
		map[string]interface{}{},
	)

	root, ok := value.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	clonedValue := cloneCWAIReviewJSONValue(root)
	cloned, ok := clonedValue.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	if config == nil ||
		config.LessonReferenceMode !=
			models.CWAIReviewLessonReferenceNoLesson {
		return cloned
	}

	cloned["lesson_plan"] = map[string]interface{}{
		"available": false,
		"used":      false,
	}
	cloned["course_outline"] = map[string]interface{}{
		"available": false,
		"used":      false,
		"titles":    []interface{}{},
	}
	cloned["alignment_report"] = map[string]interface{}{
		"available": false,
		"used":      false,
		"status":    "",
		"summary":   "",
	}

	return cloned
}

// cwAIReviewConfigPromptJSON 构造系统提示词中的配置JSON。
func cwAIReviewConfigPromptJSON(
	session *models.CoursewareAIReviewSession,
) (string, error) {
	configReport, err :=
		buildCWAIReviewConfigReport(session)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(configReport)
	if err != nil {
		return "", fmt.Errorf(
			"序列化课件AI审核配置提示失败: %w",
			err,
		)
	}

	return string(encoded), nil
}

// cwAIReviewLessonReferencePromptRule 返回教案参考模式的系统规则。
func cwAIReviewLessonReferencePromptRule(
	config *CWAIReviewConfigSnapshot,
) string {
	if config == nil {
		return ""
	}

	switch config.LessonReferenceMode {
	case models.CWAIReviewLessonReferenceStrictAlignment:
		return `【教案参考规则：严格一致】
必须逐项检查课件是否与来源教案、课程大纲和对齐报告保持严格一致。
目标、环节、案例、知识边界或活动顺序偏离时，应提供明确证据。`

	case models.CWAIReviewLessonReferenceLessonIntent:
		return `【教案参考规则：参考教案意图】
应理解教案目标、教学意图和关键环节，但允许课件采用不同表达和页面组织。
只有偏离核心教学意图或知识边界时才形成问题。`

	case models.CWAIReviewLessonReferenceNoLesson:
		return `【教案参考规则：不使用教案】
本次审核没有使用教案正文、课程大纲或对齐报告。
不得声称教案要求，不得推断教案内容，lesson_or_outline_basis必须为空。`

	default:
		return `【教案参考规则：现行兼容】
按既有审核行为综合参考教案、课程大纲、对齐报告、教育域和年级边界。`
	}
}

// cwAIReviewLessonReferenceTaskInstructions 返回批次和最终阶段任务要求。
func cwAIReviewLessonReferenceTaskInstructions(
	config *CWAIReviewConfigSnapshot,
) []string {
	if config == nil {
		return []string{}
	}

	switch config.LessonReferenceMode {
	case models.CWAIReviewLessonReferenceStrictAlignment:
		return []string{
			"严格核对课件与教案目标、教学环节、案例、活动顺序和课程大纲的一致性。",
		}

	case models.CWAIReviewLessonReferenceLessonIntent:
		return []string{
			"参考教案的核心教学意图和知识边界，不要求页面表达与教案逐字一致。",
		}

	case models.CWAIReviewLessonReferenceNoLesson:
		return []string{
			"本次不使用教案、课程大纲和对齐报告，不得推断或引用这些材料。",
			"所有finding.lesson_or_outline_basis必须为空字符串。",
		}

	default:
		return []string{
			"按现行兼容方式检查教案适配、内容逻辑和互动操作。",
		}
	}
}
