package services

// courseware_alignment_service.go — 课件↔教案对齐校验服务
//
// 职责：比对"课件逐页方案"是否忠实还原"教案教学意图"，产出结构化对齐报告
//       （覆盖度 coverage / 新增 additions / 意图偏移 intent_shifts）落库，
//       供老师在 Step1 确认方案时看到明确信号（哪些环节遗漏、哪些是AI新增、有无目标漂移）。
//
// 设计（仿 courseware_curriculum.go 轻量风格 + 异步范式同 BackfillPageIndexAsync）：
//   - 仅当课件 source_type == lesson_plan 且关联教案存在时才校验；其它来源静默跳过。
//   - 异步执行（go func + context.Background + defer recover），不阻塞方案生成主流程。
//   - 模型走 courseware_alignment 场景（opus），经 applyModelPolicy 按学校授权自动分流，
//     未授权校降级境内 qwen（通过 TraceContext.SchoolID 实现，与课件其它AI调用一致）。
//   - 结果落 courseware_alignment_reports（先 generating 占位 → done/failed），
//     前端 Step1 GET 拉取 + status=generating 时短轮询。失败不影响课件可用。
//
// 触发点（覆盖所有方案变化路径）：
//   - 自动：courseware_index_service.saveAndBroadcast 末尾统一挂载（首次生成/AI改方案/重出方案都覆盖）
//   - 手动：handler 暴露重算端点，老师改完方案可主动重新校验

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 对齐校验服务 ====================

// CoursewareAlignmentService 课件对齐校验服务
type CoursewareAlignmentService struct {
	cfg *config.Config
}

// NewCoursewareAlignmentService 创建对齐校验服务
func NewCoursewareAlignmentService(cfg *config.Config) *CoursewareAlignmentService {
	return &CoursewareAlignmentService{cfg: cfg}
}

// alignMaxLessonContentRunes 教案正文喂AI的截断上限（与索引压缩同量级，控制token）
const alignMaxLessonContentRunes = 18000

// ==================== 异步触发入口 ====================

// TriggerAlignmentAsync 异步触发对齐校验（go func 包裹，调用方零等待）
//
// 由 saveAndBroadcast（方案落库后）与 handler 手动重算端点调用。
// 内部自行判断课件来源/教案是否存在，非教案来源或无教案则静默跳过不留记录。
func (s *CoursewareAlignmentService) TriggerAlignmentAsync(coursewareID string, userID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[courseware_alignment] 对齐校验 panic 已恢复: cw=%s r=%v", coursewareID, r)
			}
		}()
		ctx := context.Background()
		s.runAlignment(ctx, coursewareID, userID)
	}()
}

// ==================== 校验主流程 ====================

// runAlignment 执行一次完整对齐校验（同步，由 TriggerAlignmentAsync 在 goroutine 内调用）
func (s *CoursewareAlignmentService) runAlignment(ctx context.Context, coursewareID string, userID string) {
	// ---- 1. 取课件，判断来源 ----
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		log.Printf("[courseware_alignment] 取课件失败，跳过: cw=%s err=%v", coursewareID, err)
		return
	}
	// 仅教案来源才有对齐意义；其它来源（主题/PPT/Doc/3D）静默跳过，不留记录
	if cw.SourceType != models.CWSourceLessonPlan {
		log.Printf("[courseware_alignment] 非教案来源(%s)，跳过对齐: cw=%s", cw.SourceType, coursewareID)
		return
	}
	if cw.LessonPlanID == nil || *cw.LessonPlanID == "" {
		log.Printf("[courseware_alignment] 课件未关联教案，跳过对齐: cw=%s", coursewareID)
		return
	}
	lessonPlanID := cw.LessonPlanID

	// ---- 2. 取课件页面方案（方案侧输入）----
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		log.Printf("[courseware_alignment] 取课件页面失败或为空，跳过: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 3. 占位：先写 generating 让前端立刻可见"校验中" ----
	if err := repository.UpsertGeneratingReport(ctx, coursewareID, lessonPlanID, len(pages)); err != nil {
		log.Printf("[courseware_alignment] 写 generating 占位失败（继续校验）: cw=%s err=%v", coursewareID, err)
	}

	// ---- 4. 取教案正文 ----
	lp, err := repository.GetLessonPlanByID(ctx, *lessonPlanID)
	if err != nil {
		s.failReport(ctx, coursewareID, lessonPlanID, "关联教案不存在")
		return
	}
	lessonContent := ExtractLessonPlanContentForCW(lp)
	if len(strings.TrimSpace(lessonContent)) < 50 {
		s.failReport(ctx, coursewareID, lessonPlanID, "教案内容过少，无法比对")
		return
	}
	// 截断控制 token
	if len([]rune(lessonContent)) > alignMaxLessonContentRunes {
		lessonContent = string([]rune(lessonContent)[:alignMaxLessonContentRunes]) + "\n\n[教案过长，已截取前部]"
	}

	// ---- 5. 加载对齐提示词 ----
	alignPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_alignment")
	if err != nil {
		s.failReport(ctx, coursewareID, lessonPlanID, "加载对齐提示词失败")
		log.Printf("[courseware_alignment] 加载提示词失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 6. 构建用户提示词（教案正文 + 逐页方案）----
	userPrompt := s.buildAlignmentUserPrompt(lp, lessonContent, pages)

	// ---- 7. 调 AI（courseware_alignment 场景，opus；未授权校经分流降级 qwen）----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_alignment",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.failReport(ctx, coursewareID, lessonPlanID, "获取AI配置失败")
		log.Printf("[courseware_alignment] 获取AI配置失败: cw=%s err=%v", coursewareID, err)
		return
	}

	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: "courseware_alignment", UserID: &userID, SchoolID: schoolIDPtr(schoolID)}
	callResult, err := ai.CallAI(aiCfg, alignPrompt.Content, userPrompt, traceCtx)
	if err != nil {
		s.failReport(ctx, coursewareID, lessonPlanID, "AI对齐分析失败")
		log.Printf("[courseware_alignment] AI调用失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 8. 解析 AI 输出 JSON ----
	result, cleanJSON, parseErr := s.parseAlignmentResult(callResult.Content)
	if parseErr != nil {
		s.failReport(ctx, coursewareID, lessonPlanID, "解析对齐结果失败")
		log.Printf("[courseware_alignment] 解析失败: cw=%s err=%v 原始输出前200字=%s",
			coursewareID, parseErr, cwAlignSafeHead(callResult.Content, 200))
		return
	}

	// ---- 9. 落库 done ----
	overall := s.normalizeOverall(result.Overall)
	if err := repository.UpsertDoneReport(ctx, coursewareID, lessonPlanID,
		overall, strings.TrimSpace(result.Summary), cleanJSON,
		callResult.ModelUsed, callResult.TokensUsed, len(pages)); err != nil {
		log.Printf("[courseware_alignment] 落库 done 失败: cw=%s err=%v", coursewareID, err)
		return
	}

	log.Printf("[courseware_alignment] 对齐校验完成: cw=%s overall=%s pages=%d model=%s tokens=%d coverage=%d additions=%d shifts=%d",
		coursewareID, overall, len(pages), callResult.ModelUsed, callResult.TokensUsed,
		len(result.Coverage), len(result.Additions), len(result.IntentShifts))
}

// ==================== 提示词构建 ====================

// buildAlignmentUserPrompt 拼装"教案正文 + 课件逐页方案"作为对齐比对输入
func (s *CoursewareAlignmentService) buildAlignmentUserPrompt(lp *models.LessonPlan, lessonContent string, pages []*models.CoursewarePage) string {
	var sb strings.Builder
	sb.WriteString("请比对下面这份教案与据此生成的课件逐页方案，输出对齐分析JSON。\n\n")
	sb.WriteString("## 课件基本信息\n")
	sb.WriteString(fmt.Sprintf("- 标题：%s\n- 学科：%s\n- 年级：%s\n- 课件总页数：%d\n\n", lp.Title, lp.Subject, lp.Grade, len(pages)))

	sb.WriteString("## 原始教案内容\n\n")
	sb.WriteString(lessonContent)
	sb.WriteString("\n\n")

	sb.WriteString("## 课件逐页方案（共" + fmt.Sprintf("%d", len(pages)) + "页）\n")
	for _, p := range pages {
		sb.WriteString(fmt.Sprintf("【第%d页】%s\n", p.PageNumber, strings.TrimSpace(p.Title)))
		if strings.TrimSpace(p.Purpose) != "" {
			sb.WriteString("  目的：" + strings.TrimSpace(p.Purpose) + "\n")
		}
		if strings.TrimSpace(p.ContentSummary) != "" {
			sb.WriteString("  内容：" + strings.TrimSpace(p.ContentSummary) + "\n")
		}
	}
	sb.WriteString("\n请严格按提示词约定的JSON格式输出对齐分析，只输出JSON对象，不要任何额外说明、不要markdown代码围栏。")
	return sb.String()
}

// ==================== 结果解析 ====================

// parseAlignmentResult 解析 AI 输出为 AlignmentAIResult，并返回清洗后的合法 JSON 文本
//
// 返回：(结构化结果, 清洗后JSON文本, error)。清洗后JSON供原样落 report_json 列透传前端。
func (s *CoursewareAlignmentService) parseAlignmentResult(raw string) (*models.AlignmentAIResult, string, error) {
	// 复用 ai.ExtractJSON 提取JSON块（剥围栏/定位首个JSON对象）
	jsonText, ok := ai.ExtractJSON(raw)
	if !ok || strings.TrimSpace(jsonText) == "" {
		// 兜底：直接用原始文本去尾空白试一把
		jsonText = strings.TrimSpace(raw)
	}

	var result models.AlignmentAIResult
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return nil, "", fmt.Errorf("JSON反序列化失败: %w", err)
	}

	// 防御：三个数组 nil 归一为空切片，避免前端 .map 崩
	if result.Coverage == nil {
		result.Coverage = []models.AlignmentCoverageItem{}
	}
	if result.Additions == nil {
		result.Additions = []models.AlignmentAdditionItem{}
	}
	if result.IntentShifts == nil {
		result.IntentShifts = []models.AlignmentIntentShiftItem{}
	}
	// 各 coverage 项的 page_nums nil 归一空切片
	for i := range result.Coverage {
		if result.Coverage[i].PageNums == nil {
			result.Coverage[i].PageNums = []int{}
		}
	}

	// 重新序列化为规整 JSON（确保落库的是干净合法 JSON，而非 AI 可能夹带的脏文本）
	cleanBytes, err := json.Marshal(&result)
	if err != nil {
		return nil, "", fmt.Errorf("重新序列化失败: %w", err)
	}
	return &result, string(cleanBytes), nil
}

// normalizeOverall 校验 overall 取值，非法时按是否有 missing/shift 兜底推断
func (s *CoursewareAlignmentService) normalizeOverall(overall string) string {
	o := strings.TrimSpace(strings.ToLower(overall))
	switch o {
	case models.CWAlignOverallAligned, models.CWAlignOverallMinor, models.CWAlignOverallMajor:
		return o
	default:
		// AI 给了非法值，保守归为 minor（既不夸大问题也不掩盖）
		return models.CWAlignOverallMinor
	}
}

// ==================== 辅助 ====================

// failReport 统一失败落库
func (s *CoursewareAlignmentService) failReport(ctx context.Context, coursewareID string, lessonPlanID *string, msg string) {
	if err := repository.MarkReportFailed(ctx, coursewareID, lessonPlanID, msg); err != nil {
		log.Printf("[courseware_alignment] 写 failed 报告失败: cw=%s err=%v", coursewareID, err)
	}
}

// cwAlignSafeHead 安全取字符串前 n 个 rune（日志用，避免中文截半 + 越界）
func cwAlignSafeHead(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
