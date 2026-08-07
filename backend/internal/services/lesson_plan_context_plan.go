package services

// lesson_plan_context_plan.go — 教案对话单轮确定性上下文规划器
//
// 设计原则：
//   - “已挂载”只表示本会话可用，不等于每轮自动注入；
//   - 普通讨论只使用阶段骨架、短历史和老师当前问题；
//   - 完整教案、全面评审、修订定稿等正式产物才加载必要证据并启用多证据Harness；
//   - 课本、参考资料中的篇章事实优先于通用课程大纲；
//   - 不新增AI分类调用，全部使用确定性规则。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	lessonPlanLightHistoryMessages = 6
	lessonPlanLightHistoryRunes    = 6000
)

// lessonPlanTurnContextPlan 描述本轮真正需要加载的上下文与质量链。
type lessonPlanTurnContextPlan struct {
	FormalArtifact          bool
	UseRecipe               bool
	UsePriorOutputs         bool
	UseComponents           bool
	UseTextbook             bool
	NeedsTextbookImage      bool
	UseUnitPlan             bool
	UseCourseOutline        bool
	UseRawCourseOutline     bool
	UseKnowledgeLineage     bool
	UseContextCapsule       bool
	UseClassProfile         bool
	UseRefMaterial          bool
	BlockingEvidenceHarness bool
	Reason                  string
}

type lessonPlanTurnContextPlanKey struct{}

func withLessonPlanTurnContextPlan(
	ctx context.Context,
	plan *lessonPlanTurnContextPlan,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if plan == nil {
		return ctx
	}
	return context.WithValue(
		ctx,
		lessonPlanTurnContextPlanKey{},
		plan,
	)
}

func lessonPlanTurnContextPlanFromContext(
	ctx context.Context,
) *lessonPlanTurnContextPlan {
	if ctx == nil {
		return nil
	}
	plan, _ := ctx.Value(
		lessonPlanTurnContextPlanKey{},
	).(*lessonPlanTurnContextPlan)
	return plan
}

// buildLessonPlanTurnContextPlan 根据老师当前任务确定本轮真正需要的资源。
func buildLessonPlanTurnContextPlan(
	lessonPlan *models.LessonPlan,
	request *models.LessonPlanChatRequest,
) *lessonPlanTurnContextPlan {
	plan := &lessonPlanTurnContextPlan{}
	if lessonPlan == nil || request == nil {
		return plan
	}

	text := normalizeLessonPlanTurnText(request.Message)
	stageCode := strings.ToLower(
		strings.TrimSpace(lessonPlan.CurrentStage),
	)

	automaticStageOpening := isStageAutoTriggerContent(
		request.Message,
	)

	revisionMutation :=
		isLessonPlanRevisionMutationIntent(
			stageCode,
			text,
		)

	formalKeywords := []string{
		"完整教案", "完整正文", "整份教案", "全案", "一键生成",
		"生成教案", "写出教案", "正式教案", "完整分析", "完整设计",
		"全面评审", "正式评审", "完整评审", "修订定稿", "最终定稿",
		"发布前检查", "发布前校验", "完整方案", "完整替换", "改一版教案",
		"更新完整教案", "按评审意见修改", "按这个修改", "全部修改",
	}

	formal := request.FullGenerate ||
		revisionMutation ||
		containsAnyLessonPlanTurnText(text, formalKeywords)
	automaticFormalReview := automaticStageOpening && stageCode == "review"

	if stageCode == "review" && containsAnyLessonPlanTurnText(
		text,
		[]string{"评审", "审核", "评分", "全面检查", "完整检查"},
	) {
		formal = true
	}
	if stageCode == "revise" && containsAnyLessonPlanTurnText(
		text,
		[]string{"定稿", "完整修改", "修改教案", "更新正文", "改一版", "按评审"},
	) {
		formal = true
	}
	if automaticStageOpening &&
		!automaticFormalReview &&
		!revisionMutation {
		formal = false
	}
	if automaticFormalReview {
		formal = true
	}

	hasTextbook := strings.TrimSpace(
		lessonPlan.TextbookPageIDs,
	) != "" && strings.TrimSpace(
		lessonPlan.TextbookPageIDs,
	) != "[]"
	hasUnitPlan := lessonPlan.UnitPlanID != nil &&
		strings.TrimSpace(*lessonPlan.UnitPlanID) != ""
	hasOutline := lessonPlan.CourseOutlinePublisher != nil
	hasExactOutline := lessonPlan.CourseOutlineID != nil &&
		strings.TrimSpace(*lessonPlan.CourseOutlineID) != ""
	hasClassProfile := lessonPlan.ClassProfileID != nil &&
		strings.TrimSpace(*lessonPlan.ClassProfileID) != ""
	hasRefMaterial := strings.TrimSpace(
		request.RefMaterial,
	) != ""

	textbookKeywords := []string{
		"课本", "教材", "课文", "原文", "页面", "页码", "第几页",
		"课本图片", "教材图片", "原图", "附图", "插图", "版面",
		"截图", "图中", "图片内容", "文章内容", "这篇文章", "这篇课文",
	}
	imageKeywords := []string{
		"原图", "图片", "版面", "插图", "附图", "截图", "图中",
		"看图", "页面布局", "排版", "图片是不是", "图片内容",
	}
	outlineKeywords := []string{
		"课程大纲", "教学大纲", "大纲", "课程标准", "课标",
		"学段要求", "大纲要求", "出版社版本", "教材版本要求",
	}
	unitPlanKeywords := []string{
		"单元方案", "大单元", "单元目标", "单元任务", "单元评价",
		"单元进阶", "单元位置", "前后课", "单元整体", "单元设计",
	}
	classProfileKeywords := []string{
		"学情", "班级", "学生特点", "学生基础", "分层", "差异化",
		"薄弱点", "基础差异", "特殊学生", "本班", "学生画像",
	}
	refMaterialKeywords := []string{
		"附件", "参考资料", "pdf", "文档", "文件", "上传的资料",
		"刚上传", "这份资料", "这份材料", "根据这份", "按这份",
		"依据材料", "材料中", "附件中", "文档中",
	}
	recipeKeywords := []string{
		"配方", "模板", "九大板块", "教案结构", "按模板", "按配方",
	}
	componentKeywords := []string{
		"组件", "教学组件", "量规", "活动模板", "评价工具", "策略库",
	}
	continuityKeywords := []string{
		"前面", "之前", "上一阶段", "根据分析", "沿用", "继续",
		"评审意见", "前序", "刚才定的", "前面确定",
	}

	explicitTextbook := containsAnyLessonPlanTurnText(
		text,
		textbookKeywords,
	)
	explicitRefMaterial := containsAnyLessonPlanTurnText(
		text,
		refMaterialKeywords,
	)

	// 老师刚附资料后常直接说“帮我分析一下/看看”，这类短指令仍应使用附件。
	if hasRefMaterial && !explicitRefMaterial && len([]rune(text)) <= 40 &&
		containsAnyLessonPlanTurnText(
			text,
			[]string{"分析一下", "帮我分析", "帮我看看", "看一下", "梳理一下", "总结一下", "提炼一下"},
		) {
		explicitRefMaterial = true
	}

	plan.FormalArtifact = formal
	plan.UseRecipe = formal || containsAnyLessonPlanTurnText(
		text,
		recipeKeywords,
	)
	plan.UsePriorOutputs = formal || automaticStageOpening || containsAnyLessonPlanTurnText(
		text,
		continuityKeywords,
	)
	plan.UseComponents = formal ||
		len(request.SelectedComponents) > 0 ||
		containsAnyLessonPlanTurnText(text, componentKeywords)
	plan.UseTextbook = hasTextbook && (formal || explicitTextbook)
	plan.NeedsTextbookImage = plan.UseTextbook &&
		containsAnyLessonPlanTurnText(text, imageKeywords)
	plan.UseUnitPlan = hasUnitPlan &&
		(formal || containsAnyLessonPlanTurnText(text, unitPlanKeywords))
	// 课程大纲全文只保留给老师明确询问大纲原文或版本要求的原始查询。
	// 正式教案、评审和修订不再因为“正式产物”自动注入整份大纲。
	plan.UseRawCourseOutline = hasOutline &&
		containsAnyLessonPlanTurnText(text, outlineKeywords)

	// 精确大纲教案在教学分析确认后，全阶段使用持久化的短版知识脉络。
	// 老师明确查询原始大纲时，本轮允许原始大纲替代短版脉络用于资料查阅，
	// 但不能把查询结果静默写回已经确认的课程锚点。
	plan.UseKnowledgeLineage =
		hasExactOutline &&
			!plan.UseRawCourseOutline

	// 兼容现有提示词装配入口：
	// true表示本轮需要“课程层级上下文”，具体返回原始大纲还是知识脉络，
	// 由BuildLessonPlanCourseOutlineContext根据两个细分开关决定。
	plan.UseCourseOutline =
		plan.UseRawCourseOutline ||
			plan.UseKnowledgeLineage

	plan.UseClassProfile = hasClassProfile &&
		(formal || containsAnyLessonPlanTurnText(text, classProfileKeywords))
	plan.UseRefMaterial = hasRefMaterial &&
		(formal || explicitRefMaterial)

	plan.BlockingEvidenceHarness = formal &&
		(plan.UseTextbook ||
			plan.UseRefMaterial ||
			plan.UseUnitPlan ||
			plan.UseCourseOutline ||
			plan.UseKnowledgeLineage ||
			plan.UseClassProfile)

	switch {
	case automaticFormalReview:
		plan.Reason = "评审阶段自动任务，按正式评审加载证据"
	case revisionMutation:
		plan.Reason = "修订阶段明确要求修改正式正文，按正式产物执行"
	case automaticStageOpening:
		plan.Reason = "阶段过渡开场，使用轻量阶段骨架并承接前序结论"
	case formal:
		plan.Reason = "正式产物任务，加载本轮必要证据"
	default:
		plan.Reason = "普通讨论，按老师当前问题最小化加载"
	}

	return plan
}

// validateLessonPlanTurnContextPlan 在依赖课本的任务开始前确认全部关联页OCR已就绪。
func validateLessonPlanTurnContextPlan(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	plan *lessonPlanTurnContextPlan,
) error {
	if lessonPlan == nil || plan == nil || !plan.UseTextbook {
		return nil
	}

	var pageIDs []string
	if err := json.Unmarshal(
		[]byte(lessonPlan.TextbookPageIDs),
		&pageIDs,
	); err != nil || len(pageIDs) == 0 {
		return fmt.Errorf(
			"课本关联数据异常，本轮已停止生成，请重新关联课本页面",
		)
	}

	pages, err := repository.GetTextbookPagesByIDs(
		ctx,
		pageIDs,
	)
	if err != nil {
		return fmt.Errorf(
			"读取课本页面失败，本轮已停止生成: %w",
			err,
		)
	}
	if len(pages) != len(pageIDs) {
		return fmt.Errorf(
			"部分课本页面已失效，本轮已停止生成，请重新关联课本页面",
		)
	}

	unreadable := make([]string, 0)
	for _, page := range pages {
		if page == nil || strings.TrimSpace(page.OCRText) == "" {
			label := "未识别页面"
			if page != nil {
				label = strings.TrimSpace(page.Chapter)
				if label == "" {
					label = strings.TrimSpace(page.FileName)
				}
				if label == "" {
					label = page.ID
				}
			}
			unreadable = append(unreadable, label)
		}
	}

	if len(unreadable) > 0 {
		return fmt.Errorf(
			"课本文字识别尚未完成（%s），系统已阻止依据不完整课本生成。请完成识别后重试",
			strings.Join(unreadable, "、"),
		)
	}

	return nil
}

// buildLessonPlanSourceAuthorityPrompt 为实际加载的证据声明确定性优先级。
func buildLessonPlanSourceAuthorityPrompt(
	plan *lessonPlanTurnContextPlan,
) string {
	if plan == nil || (!plan.UseTextbook && !plan.UseRefMaterial &&
		!plan.UseCourseOutline && !plan.UseKnowledgeLineage &&
		!plan.UseContextCapsule && !plan.UseUnitPlan &&
		!plan.UseClassProfile) {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n\n【本轮资料证据优先级（系统确定性规则）】\n")
	builder.WriteString("1. 篇名、正文、题目、页码、人物、地点、动物、数据和页面事实：老师挂载的课本页或教师附件优先；课程大纲没有收录某篇材料，不等于该材料不存在或属于模型幻觉。\n")
	builder.WriteString("2. 本课正式知识点、学习深度、前置关系、后续发展、教学边界和教师已确认决定：优先以active核心共识胶囊为统一依据；胶囊尚未形成时使用active知识脉络兜底。已纠正或替代内容不得被前序摘要、配方、组件或模型常识重新激活。\n")
	builder.WriteString("3. 老师明确询问课程大纲原文或版本要求时，才读取原始大纲用于回答该问题；原始大纲不能替代已经确认的本课课程锚点。\n")
	builder.WriteString("4. 单元方案负责本单元位置、进阶、任务与评价衔接；班级学情负责差异化组织，不得改写课本事实或统一知识脉络。\n")
	builder.WriteString("5. 只有无法从老师任务、课本、附件、知识脉络、原始大纲查询或单元方案追溯的模型新增事实，才属于无依据扩写。\n")
	builder.WriteString("6. 资料之间出现冲突时，明确指出冲突与来源，不得静默用模型常识替换老师材料。\n")

	if plan.NeedsTextbookImage {
		builder.WriteString("7. 当前文本生成链只获得课本OCR文字，不能声称已经直接核验图片版式、插图或栏目位置；涉及版面问题时必须明确说明这一限制，不得猜测。\n")
	}

	return builder.String()
}

func buildLightweightLessonPlanContextReceipt(
	lessonPlan *models.LessonPlan,
	request *models.LessonPlanChatRequest,
	assistantReceipt *models.AssistantContextReceipt,
	plan *lessonPlanTurnContextPlan,
) *models.ContextReceipt {
	receipt := &models.ContextReceipt{
		Version:   models.ContextReceiptVersion,
		StageCode: lessonPlan.CurrentStage,
		Assistant: assistantReceipt,
	}

	receipt.Recipe = lightweightMaterialReceipt(
		lessonPlan.RecipeID != nil && strings.TrimSpace(*lessonPlan.RecipeID) != "",
		plan.UseRecipe,
		"配方已挂载，但本轮普通讨论不需要完整配方上下文",
	)
	receipt.Textbook = lightweightMaterialReceipt(
		strings.TrimSpace(lessonPlan.TextbookPageIDs) != "" && strings.TrimSpace(lessonPlan.TextbookPageIDs) != "[]",
		plan.UseTextbook,
		"课本已关联，但本轮问题不依赖课本，未注入",
	)
	receipt.UnitPlan = lightweightMaterialReceipt(
		lessonPlan.UnitPlanID != nil && strings.TrimSpace(*lessonPlan.UnitPlanID) != "",
		plan.UseUnitPlan,
		"单元方案已关联，但本轮问题不需要单元整体证据，未注入",
	)
	receipt.CourseOutline = lightweightMaterialReceipt(
		lessonPlan.CourseOutlinePublisher != nil,
		plan.UseRawCourseOutline,
		"课程大纲已挂载，但本轮没有明确查询原始大纲，未注入全文",
	)
	knowledgeLineageReason := "知识脉络来源已关联，但当前尚未形成可用active快照"
	if plan.UseContextCapsule {
		knowledgeLineageReason = "active知识脉络的课程核心已进入本课共识胶囊，本轮未重复注入"
	}
	receipt.KnowledgeLineage = lightweightMaterialReceipt(
		lessonPlan.CourseOutlineID != nil &&
			strings.TrimSpace(*lessonPlan.CourseOutlineID) != "",
		plan.UseKnowledgeLineage,
		knowledgeLineageReason,
	)
	receipt.ClassProfile = lightweightMaterialReceipt(
		lessonPlan.ClassProfileID != nil && strings.TrimSpace(*lessonPlan.ClassProfileID) != "",
		plan.UseClassProfile,
		"班级学情已关联，但本轮问题不涉及差异化教学，未注入",
	)
	receipt.RefMaterial = lightweightMaterialReceipt(
		strings.TrimSpace(request.RefMaterial) != "",
		plan.UseRefMaterial,
		"参考资料附件可用，但本轮问题没有要求使用，未注入",
	)

	if plan.UseRefMaterial && receipt.RefMaterial != nil {
		receipt.RefMaterial.CharacterCount = len(
			[]rune(strings.TrimSpace(request.RefMaterial)),
		)
	}

	if plan.UseComponents || len(request.SelectedComponents) > 0 {
		receipt.Components = &models.ComponentsContextReceipt{
			Status:        "loaded",
			SelectionMode: "planned",
			Reason:        "本轮任务明确需要专业组件或老师已选择组件",
		}
	} else {
		receipt.Components = &models.ComponentsContextReceipt{
			Status: "deferred",
			Reason: "普通讨论未加载专业组件",
		}
	}

	return receipt
}

func lightweightMaterialReceipt(
	linked bool,
	used bool,
	deferredReason string,
) *models.MaterialContextReceipt {
	switch {
	case used:
		return &models.MaterialContextReceipt{
			Status: "loaded",
			Reason: "本轮确定性上下文规划器判定需要使用",
		}
	case linked:
		return &models.MaterialContextReceipt{
			Status: "deferred",
			Reason: deferredReason,
		}
	default:
		return &models.MaterialContextReceipt{
			Status: "not_linked",
		}
	}
}

// limitLessonPlanWorkingMessages 为普通讨论保留短历史，避免整阶段对话无限膨胀。
func limitLessonPlanWorkingMessages(
	messages []*models.ConversationMessage,
) []*models.ConversationMessage {
	selected := make(
		[]*models.ConversationMessage,
		0,
		lessonPlanLightHistoryMessages,
	)
	usedRunes := 0

	for index := len(messages) - 1; index >= 0; index-- {
		if len(selected) >= lessonPlanLightHistoryMessages ||
			usedRunes >= lessonPlanLightHistoryRunes {
			break
		}

		message := messages[index]
		if message == nil || strings.TrimSpace(message.Content) == "" {
			continue
		}

		remaining := lessonPlanLightHistoryRunes - usedRunes
		if remaining <= 0 {
			break
		}

		messageRunes := []rune(message.Content)
		if len(messageRunes) > remaining {
			copyMessage := *message
			copyMessage.Content = truncateLessonPlanHistoryContent(
				messageRunes,
				remaining,
			)
			selected = append(selected, &copyMessage)
			usedRunes += len([]rune(copyMessage.Content))
			break
		}

		selected = append(selected, message)
		usedRunes += len(messageRunes)
	}

	for left, right := 0, len(selected)-1; left < right; left, right = left+1, right-1 {
		selected[left], selected[right] = selected[right], selected[left]
	}

	return selected
}

// truncateLessonPlanHistoryContent 在严格字符预算内同时保留消息开头和结尾。
func truncateLessonPlanHistoryContent(
	content []rune,
	limit int,
) string {
	if limit <= 0 {
		return ""
	}
	if len(content) <= limit {
		return string(content)
	}

	marker := []rune("\n…历史消息已按上下文预算截断…\n")
	if limit <= len(marker)+2 {
		return string(content[:limit])
	}

	available := limit - len(marker)
	head := available / 2
	tail := available - head

	return string(content[:head]) +
		string(marker) +
		string(content[len(content)-tail:])
}

func normalizeLessonPlanTurnText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		" ", "",
		"\n", "",
		"\r", "",
		"\t", "",
		"，", ",",
		"。", ".",
		"：", ":",
		"（", "(",
		"）", ")",
	)
	return replacer.Replace(value)
}

// isLessonPlanRevisionMutationIntent 判断revise阶段是否在要求真正修改正式正文。
//
// 不能只看单个“改”字，否则“这个修改建议是什么意思”也会被误升级为整稿任务。
// 这里要求同时出现修改动作和明确的教案目标对象；命中后由正式产物链负责生成、
// Word结构校验、正文事务提交和content_update广播。
func isLessonPlanRevisionMutationIntent(
	stageCode string,
	text string,
) bool {
	if strings.ToLower(
		strings.TrimSpace(stageCode),
	) != "revise" {
		return false
	}

	hasMutationVerb :=
		containsAnyLessonPlanTurnText(
			text,
			[]string{
				"修改", "只改", "直接改", "帮我改",
				"调整", "补充", "删除", "删掉",
				"替换", "改成", "改为", "加上",
				"加入", "写入", "写进", "完善",
				"优化", "修订", "同步", "保存",
			},
		)
	if !hasMutationVerb {
		return false
	}

	return containsAnyLessonPlanTurnText(
		text,
		[]string{
			"教案", "正文", "原文", "原有段落",
			"段落文字", "教学目标", "教学重点",
			"教学难点", "教学活动", "教学过程",
			"导入", "作业", "板书", "反思",
			"前面的问题", "上面的内容", "这个版本",
			"右侧", "画布", "正式正文", "正式教案",
		},
	)
}

func containsAnyLessonPlanTurnText(
	text string,
	keywords []string,
) bool {
	for _, keyword := range keywords {
		normalizedKeyword := normalizeLessonPlanTurnText(keyword)
		if normalizedKeyword != "" && strings.Contains(text, normalizedKeyword) {
			return true
		}
	}
	return false
}
