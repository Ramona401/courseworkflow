package services

// courseware_index_topic.go — 从主题直接生成课件索引（GenerateIndexFromTopic）
//
// 从 courseware_index_service.go 拆出（超 600 行红线拆分）。
// 与主文件同 package 同 *CoursewareIndexService 接收器，复用主文件的
// broadcastError/parseSchemeJSON/saveAndBroadcast 及包级 cwClamp，
// 并调用 courseware_curriculum.go 的 BuildCurriculumConstraint，调用方零改动。
//
// v0.42：从主题直接生成课件索引（无教案，纯AI规划），跳过层1直接调层2方案翻译。
// 课程知识库轮：若前端传知识点编码，先查 curriculum_standards 构建难度适配约束段落注入。

import (
	"context"
	"fmt"
	"log"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// GenerateIndexFromTopic 从主题直接生成课件索引（无教案，纯AI规划）
// 流程：
//  1. 校验课件状态和权限
//  2. 用主题信息构建提示词，跳过层1（无教案内容可压缩）
//  3. 直接调层2 AI生成方案JSON
//  4. 写入数据库并SSE广播
func (s *CoursewareIndexService) GenerateIndexFromTopic(ctx context.Context, coursewareID string, userID string, req *models.CreateCoursewareFromTopicRequest, preset string) error {
	// ---- 1. 获取课件信息 ----
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		s.broadcastError(coursewareID, "课件不存在: "+err.Error())
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		s.broadcastError(coursewareID, "无权操作此课件")
		return fmt.Errorf("无权操作此课件")
	}
	if cw.Status != models.CoursewareStatusDraft && cw.Status != models.CoursewareStatusIndexing {
		s.broadcastError(coursewareID, "当前状态不允许生成方案: "+cw.Status)
		return fmt.Errorf("当前状态不允许生成方案: %s", cw.Status)
	}

	// ---- 2. 更新课件状态为 indexing ----
	if cw.Status == models.CoursewareStatusDraft {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusIndexing)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"message":       "正在根据主题规划课件方案...",
		},
	})

	// ---- 3. 构建主题直接生成的提示词 ----
	// 课程知识库轮：若前端传了知识点编码，先查 curriculum_standards 构建难度适配约束段落
	// 为空/查询失败/查不到时 constraint 为空串，buildTopicDirectPrompt 退回原有纯主题规划逻辑
	curriculumConstraint := BuildCurriculumConstraint(ctx, req.KPCodes)
	userPrompt := s.buildTopicDirectPrompt(req, preset, curriculumConstraint)

	// ---- 4. 加载提示词模板（复用 courseware_scheme 场景） ----
	schemePrompt, sErr := repository.GetCurrentPromptByKey("prompt_courseware_scheme")
	systemPrompt := ""
	if sErr == nil {
		systemPrompt = schemePrompt.Content
	} else {
		systemPrompt = "你是K12课件规划专家，请按要求输出JSON格式的课件方案。"
	}

	// ---- 5. 调用AI（courseware_scheme场景，降级到scanner） ----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_scheme",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		aiCfg, err = ai.GetEffectiveConfig(
			s.cfg.GetAESKey(), "scanner",
			s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
		)
		if err != nil {
			s.broadcastError(coursewareID, "获取AI配置失败")
			return fmt.Errorf("获取AI配置失败: %w", err)
		}
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "AI正在规划课件结构..."},
	})

	// v198：解析操作者所属学校ID，供模型境内/境外分流判定
	topicSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: "courseware_scheme", UserID: &userID, SchoolID: schoolIDPtr(topicSchoolID)}
	callResult, err := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		s.broadcastError(coursewareID, "AI规划失败: "+err.Error())
		return fmt.Errorf("AI调用失败: %w", err)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEIndexProgress,
		Data:      map[string]interface{}{"message": "方案生成完成，正在整理..."},
	})

	// ---- 6. 解析JSON输出 ----
	schemes, err := s.parseSchemeJSON(callResult.Content)
	if err != nil {
		s.broadcastError(coursewareID, "解析方案失败: "+err.Error())
		return fmt.Errorf("解析方案失败: %w", err)
	}
	if len(schemes) == 0 {
		s.broadcastError(coursewareID, "AI未返回有效方案")
		return fmt.Errorf("AI未返回有效方案")
	}

	// ---- 7. 构建CoursewarePage列表（无层1索引，全部来自层2方案） ----
	var pages []*models.CoursewarePage
	for i, sc := range schemes {
		page := &models.CoursewarePage{
			CoursewareID:        coursewareID,
			PageNumber:          i + 1,
			Title:               strings.TrimSpace(sc.Title),
			Purpose:             strings.TrimSpace(sc.Purpose),
			ContentSummary:      strings.TrimSpace(sc.ContentSummary),
			InteractionType:     strings.TrimSpace(sc.InteractionType),
			VisualFormat:        strings.TrimSpace(sc.VisualFormat),
			MediaRequirements:   strings.TrimSpace(sc.MediaRequirements),
			EstimatedComplexity: cwClamp(sc.EstimatedComplexity, 1, 5),
			Status:              models.CWPageStatusPending,
		}
		if page.InteractionType == "" {
			page.InteractionType = "static"
		}
		if page.VisualFormat == "" {
			page.VisualFormat = "text_heavy"
		}
		pages = append(pages, page)
	}

	// 生成简要概述
	overview := fmt.Sprintf("主题：%s（%s·%s），共%d页课件方案，由AI根据主题直接规划。",
		req.Topic, req.Subject, req.Grade, len(pages))

	log.Printf("[courseware_index] TopicDirect完成: cw=%s pages=%d model=%s tokens=%d topic=%s",
		coursewareID, len(pages), callResult.ModelUsed, callResult.TokensUsed, req.Topic)

	return s.saveAndBroadcast(ctx, coursewareID, overview, pages)
}

// buildTopicDirectPrompt 构建主题直接生成的用户提示词
func (s *CoursewareIndexService) buildTopicDirectPrompt(req *models.CreateCoursewareFromTopicRequest, preset string, curriculumConstraint string) string {
	var sb strings.Builder
	sb.WriteString("你是K12课件规划专家。\n根据以下信息，设计一份完整的课件大纲（每页详细说明）。\n\n")
	sb.WriteString(fmt.Sprintf("学科: %s\n", req.Subject))
	sb.WriteString(fmt.Sprintf("年级: %s\n", req.Grade))
	sb.WriteString(fmt.Sprintf("主题: %s\n", req.Topic))
	if req.PageRange != "" {
		sb.WriteString(fmt.Sprintf("期望页数: %s\n", req.PageRange))
	} else {
		sb.WriteString("期望页数: 按学段默认（小学15-25页，初中20-30页，高中22-35页）\n")
	}
	if req.ExtraNotes != "" {
		sb.WriteString(fmt.Sprintf("额外说明: %s\n", req.ExtraNotes))
	}

	// 课程知识库轮：注入课标知识点与难度适配约束（非空时启用"难度自动适配"）
	if strings.TrimSpace(curriculumConstraint) != "" {
		sb.WriteString(curriculumConstraint)
	}

	// 注入方案结构预设
	if preset != "" {
		presetObj := models.GetSchemePresetByKey(preset)
		if presetObj != nil && presetObj.PromptHint != "" {
			sb.WriteString("\n")
			sb.WriteString(presetObj.PromptHint)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n请输出JSON数组格式。每个元素包含以下字段：\n")
	sb.WriteString("page_number(int), title(string), purpose(string), content_summary(string), ")
	sb.WriteString("interaction_type(string), visual_format(string), media_requirements(string), estimated_complexity(int 1-5)\n\n")
	sb.WriteString("设计原则：\n")
	sb.WriteString("1. 遵循课程标准，知识点覆盖完整\n")
	sb.WriteString("2. 结构：封面(1页) → 学习目标(1页) → 知识讲授(主体) → 练习(2-3页) → 总结(1页)\n")
	sb.WriteString("3. 交互类型分布：纯展示≤40%，简单交互30-40%，复杂交互≤20%\n")
	sb.WriteString("4. 难度递进：前1/3基础 → 中1/3进阶 → 后1/3综合\n\n")
	sb.WriteString("交互类型可选：static/click/drag/input/animation/video/game/quiz\n")
	sb.WriteString("视觉形式可选：text_heavy/image_text/diagram/chart/timeline/comparison/gallery/fullscreen_media\n\n")
	sb.WriteString("请只输出JSON数组，不要有任何额外说明文字。")
	return sb.String()
}
