package services

// courseware_index_topic.go — 从主题直接生成课件索引
//
// 本模块只处理topic_direct课件方案生成。正式课件中的kp_codes是唯一知识点
// 编码来源；查询课标时使用已经收敛到courseware.education_domain快照的Actor。
// 非K12课件得到空约束，K12数据库错误则在任何状态更新之前终止本次生成。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// GenerateIndexFromTopic 根据主题直接生成课件方案并通过SSE推送进度。
func (s *CoursewareIndexService) GenerateIndexFromTopic(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CreateCoursewareFromTopicRequest,
	preset string,
	customHint string,
) error {
	courseware, scopedActor, err := loadOwnedCoursewareForSchemeMutation(
		ctx,
		coursewareID,
		actor,
	)
	if err != nil {
		s.broadcastError(
			coursewareID,
			"课件授权失败: "+err.Error(),
		)
		return err
	}
	if req == nil {
		s.broadcastError(
			coursewareID,
			"主题方案请求为空",
		)
		return fmt.Errorf("主题方案请求为空")
	}
	if courseware.Status != models.CoursewareStatusDraft &&
		courseware.Status != models.CoursewareStatusIndexing {
		s.broadcastError(
			coursewareID,
			"当前状态不允许生成方案: "+courseware.Status,
		)
		return fmt.Errorf(
			"当前状态不允许生成方案: %s",
			courseware.Status,
		)
	}

	// 请求体中的编码不可信，始终从正式课件快照重新装配。
	req.KPCodes = decodeCoursewareKPCodes(
		courseware.KPCodes,
	)

	constraint, err := BuildCurriculumConstraint(
		ctx,
		scopedActor.EducationDomain,
		req.KPCodes,
	)
	if err != nil {
		s.broadcastError(
			coursewareID,
			"查询课标知识点失败",
		)
		return fmt.Errorf(
			"查询课标知识点失败: %w",
			err,
		)
	}

	// 所有可能失败的课程知识库查询完成后才推进状态，
	// 避免数据库异常留下半途indexing状态。
	if courseware.Status == models.CoursewareStatusDraft {
		_ = repository.UpdateCoursewareStatus(
			ctx,
			coursewareID,
			models.CoursewareStatusIndexing,
		)
	}

	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: CWSSEIndexStart,
			Data: map[string]interface{}{
				"courseware_id": coursewareID,
				"message":       "正在根据主题规划课件方案...",
			},
		},
	)

	systemPrompt := loadTopicSchemeSystemPrompt()

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_scheme",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		aiConfig, err = ai.GetEffectiveConfig(
			s.cfg.GetAESKey(),
			"scanner",
			s.cfg.AIAPIBaseURL,
			s.cfg.AIAPIKey,
			s.cfg.AIDefaultModel,
		)
		if err != nil {
			s.broadcastError(
				coursewareID,
				"获取AI配置失败",
			)
			return fmt.Errorf(
				"获取AI配置失败: %w",
				err,
			)
		}
	}

	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: CWSSEIndexProgress,
			Data: map[string]interface{}{
				"message": "AI正在规划课件结构...",
			},
		},
	)

	userID := scopedActor.UserID
	schoolID, _ := repository.GetSchoolIDByUserID(
		ctx,
		userID,
	)

	traceContext := &ai.TraceContext{
		SceneCode: "courseware_scheme",
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	callResult, err := ai.CallAI(
		aiConfig,
		systemPrompt,
		s.buildTopicDirectPrompt(
			req,
			preset,
			customHint,
			constraint,
		),
		traceContext,
	)
	if err != nil {
		s.broadcastError(
			coursewareID,
			"AI规划失败: "+err.Error(),
		)
		return fmt.Errorf(
			"AI调用失败: %w",
			err,
		)
	}

	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: CWSSEIndexProgress,
			Data: map[string]interface{}{
				"message": "方案生成完成，正在整理...",
			},
		},
	)

	schemes, err := s.parseSchemeJSON(
		callResult.Content,
	)
	if err != nil {
		s.broadcastError(
			coursewareID,
			"解析方案失败: "+err.Error(),
		)
		return fmt.Errorf(
			"解析方案失败: %w",
			err,
		)
	}
	if len(schemes) == 0 {
		s.broadcastError(
			coursewareID,
			"AI未返回有效方案",
		)
		return fmt.Errorf("AI未返回有效方案")
	}

	pages := buildTopicDirectPages(
		coursewareID,
		schemes,
	)

	overview := fmt.Sprintf(
		"主题：%s（%s·%s），共%d页课件方案，由AI根据主题直接规划。",
		req.Topic,
		req.Subject,
		req.Grade,
		len(pages),
	)

	log.Printf(
		"[courseware_index] TopicDirect完成: cw=%s pages=%d model=%s tokens=%d topic=%s",
		coursewareID,
		len(pages),
		callResult.ModelUsed,
		callResult.TokensUsed,
		req.Topic,
	)

	return s.saveAndBroadcast(
		ctx,
		coursewareID,
		scopedActor,
		overview,
		pages,
	)
}

// decodeCoursewareKPCodes 从正式课件JSON快照读取知识点编码；脏JSON按空处理。
func decodeCoursewareKPCodes(
	raw string,
) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var codes []string
	if err := json.Unmarshal(
		[]byte(raw),
		&codes,
	); err != nil {
		return nil
	}

	return codes
}

// loadTopicSchemeSystemPrompt 读取主题方案系统提示词，缺失时使用兼容兜底。
func loadTopicSchemeSystemPrompt() string {
	prompt, err := repository.GetCurrentPromptByKey(
		"prompt_courseware_scheme",
	)
	if err == nil &&
		prompt != nil &&
		strings.TrimSpace(prompt.Content) != "" {
		return prompt.Content
	}

	return "你是K12课件规划专家，请按要求输出JSON格式的课件方案。"
}

// buildTopicDirectPages 把AI方案转换为待生成页面。
func buildTopicDirectPages(
	coursewareID string,
	schemes []cwSchemeItem,
) []*models.CoursewarePage {
	pages := make(
		[]*models.CoursewarePage,
		0,
		len(schemes),
	)

	for index, scheme := range schemes {
		page := &models.CoursewarePage{
			CoursewareID: coursewareID,
			PageNumber:   index + 1,

			Title: strings.TrimSpace(
				scheme.Title,
			),
			Purpose: strings.TrimSpace(
				scheme.Purpose,
			),
			ContentSummary: strings.TrimSpace(
				scheme.ContentSummary,
			),
			InteractionType: strings.TrimSpace(
				scheme.InteractionType,
			),
			VisualFormat: strings.TrimSpace(
				scheme.VisualFormat,
			),
			MediaRequirements: strings.TrimSpace(
				scheme.MediaRequirements,
			),

			EstimatedComplexity: cwClamp(
				scheme.EstimatedComplexity,
				1,
				5,
			),
			Status: models.CWPageStatusPending,
		}

		if page.InteractionType == "" {
			page.InteractionType = "static"
		}
		if page.VisualFormat == "" {
			page.VisualFormat = "text_heavy"
		}

		pages = append(
			pages,
			page,
		)
	}

	return pages
}

// buildTopicDirectPrompt 构建主题课件方案用户提示词。
func (s *CoursewareIndexService) buildTopicDirectPrompt(
	req *models.CreateCoursewareFromTopicRequest,
	preset string,
	customHint string,
	curriculumConstraint string,
) string {
	var builder strings.Builder

	builder.WriteString(
		"你是K12课件规划专家。\n根据以下信息，设计一份完整的课件大纲（每页详细说明）。\n\n",
	)
	builder.WriteString(
		fmt.Sprintf(
			"学科: %s\n年级: %s\n主题: %s\n",
			req.Subject,
			req.Grade,
			req.Topic,
		),
	)

	if req.PageRange != "" {
		builder.WriteString(
			fmt.Sprintf(
				"期望页数: %s\n",
				req.PageRange,
			),
		)
	} else {
		builder.WriteString(
			"期望页数: 按学段默认（小学15-25页，初中20-30页，高中22-35页）\n",
		)
	}

	if req.ExtraNotes != "" {
		builder.WriteString(
			fmt.Sprintf(
				"额外说明: %s\n",
				req.ExtraNotes,
			),
		)
	}

	if strings.TrimSpace(
		curriculumConstraint,
	) != "" {
		builder.WriteString(
			curriculumConstraint,
		)
	}

	if hint := models.ResolveSchemePromptHint(
		preset,
		customHint,
	); hint != "" {
		builder.WriteString("\n")
		builder.WriteString(hint)
		builder.WriteString("\n")
	}

	builder.WriteString(
		"\n请输出JSON数组格式。每个元素包含以下字段：\n",
	)
	builder.WriteString(
		"page_number(int), title(string), purpose(string), content_summary(string), ",
	)
	builder.WriteString(
		"interaction_type(string), visual_format(string), media_requirements(string), estimated_complexity(int 1-5)\n\n",
	)
	builder.WriteString("设计原则：\n")
	builder.WriteString(
		"1. 遵循课程标准，知识点覆盖完整\n",
	)
	builder.WriteString(
		"2. 结构：封面(1页) → 学习目标(1页) → 知识讲授(主体) → 练习(2-3页) → 总结(1页)\n",
	)
	builder.WriteString(
		"3. 交互类型分布：纯展示≤40%，简单交互30-40%，复杂交互≤20%\n",
	)
	builder.WriteString(
		"4. 难度递进：前1/3基础 → 中1/3进阶 → 后1/3综合\n\n",
	)
	builder.WriteString(
		"交互类型可选：static/click/drag/input/animation/video/game/quiz\n",
	)
	builder.WriteString(
		"视觉形式可选：text_heavy/image_text/diagram/chart/timeline/comparison/gallery/fullscreen_media\n\n",
	)
	builder.WriteString(
		"请只输出JSON数组，不要有任何额外说明文字。",
	)

	return builder.String()
}
