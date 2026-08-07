package services

// courseware_ai_review_finalize.go
//
// 课件 AI 审核最终综合与风险回看。
//
// Finalize 只在所有批次均完成后运行：
//   - 汇总每批 findings；
//   - 重点回看 risk_pages 和人工操作复核项；
//   - 合并重复问题；
//   - 形成优先修改动作；
//   - 生成可由审核员编辑的审核意见草稿；
//   - 由后端覆盖写入不可变R-02审核配置；
//   - 再次把全部finding收敛到本次已选择维度。
//
// 为控制长上下文，最终综合不重复注入完整教案、完整大纲或页面代码。
// no_lesson模式下，最终输入也不会重新加载或推断教案材料。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// Finalize 生成全课件最终综合报告。
func (s *CoursewareAIReviewRunner) Finalize(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) (*models.CWAIReviewFinalizeResponse, error) {
	session, _, pageDigests, err :=
		s.authorizeRunnableSession(
			ctx,
			sessionID,
			actor,
			true,
		)
	if err != nil {
		return nil, err
	}

	configSnapshot, err := cwAIReviewConfigFromSession(session)
	if err != nil {
		return nil, err
	}

	if session.Status !=
		models.CWAIReviewStatusAggregating {
		return nil, errors.New(
			"课件AI审核尚未完成全部分批",
		)
	}

	hasRemaining, err :=
		repository.HasRemainingCoursewareAIReviewBatches(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, err
	}
	if hasRemaining {
		return nil, errors.New(
			"课件AI审核仍有未完成批次",
		)
	}

	batches, err :=
		repository.ListCoursewareAIReviewBatches(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, err
	}
	if len(batches) == 0 {
		return nil, errors.New(
			"课件AI审核没有可综合的批次结果",
		)
	}

	for _, batch := range batches {
		if batch == nil ||
			batch.Status !=
				models.CWAIReviewBatchDone {
			return nil, errors.New(
				"课件AI审核存在未完成批次",
			)
		}
	}

	systemPrompt, err :=
		buildCWAIReviewFinalSystemPrompt(session)
	if err != nil {
		return nil, err
	}

	userPrompt, err :=
		buildCWAIReviewFinalUserPrompt(
			session,
			batches,
			pageDigests,
		)
	if err != nil {
		return nil, err
	}

	if s == nil || s.cfg == nil {
		return nil, errors.New(
			"课件AI审核模型配置未初始化",
		)
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_ai_review",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"获取课件AI审核综合模型配置失败: %w",
			err,
		)
	}

	userID := actor.UserID
	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			actor.UserID,
		)

	var traceLessonPlanID *string
	if configSnapshot.LessonReferenceMode !=
		models.CWAIReviewLessonReferenceNoLesson {
		traceLessonPlanID = session.LessonPlanID
	}

	traceContext := &ai.TraceContext{
		SceneCode: cwAIReviewTraceScene(session),
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),

		LessonPlanID: traceLessonPlanID,
	}

	callResult, err := ai.CallAI(
		aiConfig,
		systemPrompt,
		userPrompt,
		traceContext,
	)
	if err != nil {
		_ = repository.MarkCoursewareAIReviewSessionFailed(
			context.Background(),
			session.ID,
			err.Error(),
		)
		return nil, fmt.Errorf(
			"课件AI审核综合报告生成失败: %w",
			err,
		)
	}

	report, reportJSON, err :=
		parseCWAIReviewFinalReport(
			callResult.Content,
			session,
		)
	if err == nil &&
		isCWAIReviewSelfReview(session) {
		report.HumanDecisionReminder =
			"AI课件自审仅帮助作者发现和修改问题，不等于正式审核，也不会自动提交课件。"

		normalized, marshalErr :=
			json.Marshal(report)
		if marshalErr != nil {
			err = marshalErr
		} else {
			reportJSON = string(normalized)
		}
	}
	if err != nil {
		_ = repository.MarkCoursewareAIReviewSessionFailed(
			context.Background(),
			session.ID,
			err.Error(),
		)
		return nil, err
	}

	if err := repository.CompleteCoursewareAIReviewSession(
		ctx,
		session.ID,
		reportJSON,
		callResult.ModelUsed,
		callResult.TokensUsed,
	); err != nil {
		return nil, err
	}

	updatedSession, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			session.ID,
		)
	if err != nil {
		return nil, err
	}
	if updatedSession == nil {
		return nil, ErrCWAIReviewSessionNotFound
	}

	return &models.CWAIReviewFinalizeResponse{
		Session: updatedSession,
		Report:  report,
	}, nil
}

// buildCWAIReviewFinalSystemPrompt 构建综合报告系统提示词。
func buildCWAIReviewFinalSystemPrompt(
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

	dimensionCodesJSON, err :=
		json.Marshal(config.ReviewDimensions)
	if err != nil {
		return "", fmt.Errorf(
			"序列化最终报告审核维度失败: %w",
			err,
		)
	}

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

【作者课件自审最终报告用途】
1. 报告面向课件作者，重点形成按优先级排序的修改清单。
2. review_comment_draft字段改作“作者自审修改摘要草稿”，供作者复制参考，不进入正式审核。
3. human_decision_reminder必须说明：AI自审不等于正式审核，也不会自动提交课件。
4. 不得输出“建议通过”“建议退回”或模拟审核员决定。
5. 每个高风险问题尽量给出具体页码、代码证据和修复方向。
`)
	} else {
		builder.WriteString(`

【正式课件审核最终报告用途】
报告面向人工审核员。review_comment_draft只是一份可编辑草稿，
AI不得自动提交通过或退回。
`)
	}

	builder.WriteString("\n\n【本次不可变审核配置】\n")
	builder.WriteString(configJSON)
	builder.WriteString("\n")
	builder.WriteString(
		cwAIReviewLessonReferencePromptRule(config),
	)

	builder.WriteString(fmt.Sprintf(`

【最终综合不可覆盖规则】
1. 只输出一个合法JSON对象，不要Markdown代码围栏。
2. 只表达风险和修改优先级，不得输出“建议通过”“建议退回”或替人工审核员作决定。
3. findings中的dimension只能使用：%s。
4. 合并重复问题，但不得删除不同页码或不同证据支持的问题。
5. risk_pages和manual_review_required项目必须进入综合回看。
6. review_comment_draft只是供人工编辑的草稿，不得自动提交。
7. 没有问题时findings和priority_actions可以为空，不得虚构问题。
8. manual_review_pages必须列出所有需要浏览器真实操作复核的页面。
9. findings继续使用分批协议中的完整证据结构。
10. review_config字段由后端覆盖写入，模型不得自行编造或改变配置。

输出结构必须为：

{
  "review_config": {},
  "overall_risk": "critical|high|medium|low|info",
  "summary": "全课件综合总结",
  "strengths": ["明确观察到的优点"],
  "findings": [],
  "priority_actions": [
    {
      "priority": 1,
      "title": "优先动作标题",
      "description": "具体要做什么",
      "page_numbers": [3, 4],
      "reason": "为什么优先",
      "manual_review_required": false
    }
  ],
  "manual_review_pages": [5],
  "review_comment_draft": "供审核员编辑的审核意见草稿",
  "human_decision_reminder": "AI结果仅供辅助，最终决定由人工审核员作出。"
}
`,
		string(dimensionCodesJSON),
	))

	return strings.TrimSpace(builder.String()), nil
}

// buildCWAIReviewFinalUserPrompt 构建精简综合输入。
func buildCWAIReviewFinalUserPrompt(
	session *models.CoursewareAIReviewSession,
	batches []*models.CoursewareAIReviewBatch,
	pageDigests []models.CWAIReviewPageDigest,
) (string, error) {
	config, err := cwAIReviewConfigFromSession(session)
	if err != nil {
		return "", err
	}

	compactPages := make(
		[]map[string]interface{},
		0,
		len(pageDigests),
	)

	for _, page := range pageDigests {
		compactPages = append(
			compactPages,
			map[string]interface{}{
				"page_number": page.PageNumber,

				"title": page.Title,

				"purpose": page.Purpose,

				"content_summary": page.ContentSummary,

				"interaction_type": page.InteractionType,

				"risk_flags": page.Interaction.RiskFlags,

				"manual_review_required": page.Interaction.
					ManualReviewRequired,
			},
		)
	}

	batchResults := make(
		[]map[string]interface{},
		0,
		len(batches),
	)

	for _, batch := range batches {
		if batch == nil {
			continue
		}

		batchResults = append(
			batchResults,
			map[string]interface{}{
				"batch_no": batch.BatchNo,

				"page_scope": cwAIReviewDecodeJSON(
					batch.PageScopeJSON,
					map[string]interface{}{},
				),

				"result": cwAIReviewDecodeJSON(
					batch.ResultJSON,
					map[string]interface{}{},
				),

				"risk_pages": cwAIReviewDecodeJSON(
					batch.RiskPagesJSON,
					[]interface{}{},
				),
			},
		)
	}

	finalTasks := []string{
		"合并跨批次重复发现。",
		"保留不同页码和不同证据支持的问题。",
		"所有最终finding.dimension必须属于本次已选择审核维度。",
		"重点复核互动脚本、答案提前暴露和需要人工操作的风险页。",
		"检查最终连续性账本是否揭示案例、数字、符号或结论冲突。",
		"按风险和修改依赖关系生成优先动作。",
		"生成一段可由审核员编辑的审核意见草稿，但不得替人工决定通过或退回。",
	}

	finalTasks = append(
		finalTasks,
		cwAIReviewLessonReferenceTaskInstructions(config)...,
	)

	payload := map[string]interface{}{
		"review_config": map[string]interface{}{
			"config_hash": session.ReviewConfigHash,
			"snapshot":    cwAIReviewConfigManifest(config),
		},

		"review_context": cwAIReviewDecodeJSON(
			session.ContextManifestJSON,
			map[string]interface{}{},
		),

		"compact_baseline": compactCWAIReviewBaseline(
			session.BaselineJSON,
		),

		"all_page_catalog": compactPages,

		"final_continuity_ledger": cwAIReviewDecodeJSON(
			session.ContinuityLedgerJSON,
			map[string]interface{}{},
		),

		"completed_batches": batchResults,

		"analysis_purpose": cwAIReviewPurposeCode(session),

		"final_tasks": finalTasks,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(
			"序列化课件AI审核综合输入失败: %w",
			err,
		)
	}

	return fmt.Sprintf(
			"请对下面的全部分批%s结果执行最终综合与风险回看：\n\n%s",
			cwAIReviewActionLabel(session),
			string(encoded),
		),
		nil
}

// compactCWAIReviewBaseline 删除最终综合不需要重复注入的长正文。
func compactCWAIReviewBaseline(
	raw string,
) interface{} {
	value := cwAIReviewDecodeJSON(
		raw,
		map[string]interface{}{},
	)

	root, ok :=
		value.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	clonedValue :=
		cloneCWAIReviewJSONValue(root)

	cloned, ok :=
		clonedValue.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	if lesson, ok :=
		cloned["lesson_plan"].(map[string]interface{}); ok {
		delete(lesson, "content")
	}

	if outline, ok :=
		cloned["course_outline"].(map[string]interface{}); ok {
		delete(outline, "context")
	}

	if alignment, ok :=
		cloned["alignment_report"].(map[string]interface{}); ok {
		delete(alignment, "report")
	}

	return cloned
}

// parseCWAIReviewFinalReport 解析并规范综合报告。
//
// review_config无论模型返回什么内容，最终都由后端根据Session覆盖。
func parseCWAIReviewFinalReport(
	raw string,
	session *models.CoursewareAIReviewSession,
) (
	*models.CWAIReviewFinalReport,
	string,
	error,
) {
	if session == nil {
		return nil, "", errors.New(
			"缺少课件AI审核会话",
		)
	}

	configReport, err :=
		buildCWAIReviewConfigReport(session)
	if err != nil {
		return nil, "", err
	}

	jsonText, ok := ai.ExtractJSON(raw)
	if !ok || strings.TrimSpace(jsonText) == "" {
		jsonText = strings.TrimSpace(raw)
	}

	var report models.CWAIReviewFinalReport

	if err := json.Unmarshal(
		[]byte(jsonText),
		&report,
	); err != nil {
		return nil, "", fmt.Errorf(
			"解析课件AI审核综合报告失败: %w",
			err,
		)
	}

	// 不接受AI生成的配置声明。
	report.ReviewConfig = configReport

	report.OverallRisk =
		normalizeCWAIReviewSeverity(
			report.OverallRisk,
		)

	if report.Strengths == nil {
		report.Strengths = []string{}
	}
	report.Strengths =
		cwAIReviewDedupeStrings(
			report.Strengths,
		)

	if report.Findings == nil {
		report.Findings =
			[]models.CWAIReviewFinding{}
	}

	for i := range report.Findings {
		finding := &report.Findings[i]

		if strings.TrimSpace(finding.ID) == "" {
			finding.ID = fmt.Sprintf(
				"FINAL-F%d",
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
			return nil, "", err
		}
	}

	if report.PriorityActions == nil {
		report.PriorityActions =
			[]models.CWAIReviewPriorityAction{}
	}

	for i := range report.PriorityActions {
		action := &report.PriorityActions[i]

		if action.Priority <= 0 {
			action.Priority = i + 1
		}

		action.PageNumbers =
			normalizeCWAIReviewPageNumbers(
				action.PageNumbers,
			)
	}

	sort.SliceStable(
		report.PriorityActions,
		func(i int, j int) bool {
			return report.PriorityActions[i].
				Priority <
				report.PriorityActions[j].
					Priority
		},
	)

	report.ManualReviewPages =
		normalizeCWAIReviewPageNumbers(
			report.ManualReviewPages,
		)

	if strings.TrimSpace(
		report.HumanDecisionReminder,
	) == "" {
		report.HumanDecisionReminder =
			"AI结果仅供辅助，最终通过或退回决定由人工审核员作出。"
	}

	encoded, err := json.Marshal(&report)
	if err != nil {
		return nil, "", fmt.Errorf(
			"序列化课件AI审核综合报告失败: %w",
			err,
		)
	}

	return &report, string(encoded), nil
}

// normalizeCWAIReviewPageNumbers 页码去重、去非法值并排序。
func normalizeCWAIReviewPageNumbers(
	input []int,
) []int {
	seen := make(map[int]bool)
	output := make([]int, 0, len(input))

	for _, pageNumber := range input {
		if pageNumber <= 0 ||
			seen[pageNumber] {
			continue
		}

		seen[pageNumber] = true
		output = append(
			output,
			pageNumber,
		)
	}

	sort.Ints(output)
	return output
}
