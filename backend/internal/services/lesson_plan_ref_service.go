package services

// lesson_plan_ref_service.go — 备课参考资料附件处理服务
//
// 同时承担：
//   1. 长文本参考资料压缩；
//   2. 扫描 PDF 单页图片的多模态忠实转录。
//
// 原文件和页面图不落库。页面图仅作为单次 Data URI 请求发送给视觉模型。
// 视觉转录必须逐页执行，禁止把多页拼成长图，也禁止用常识补写图片中不存在的事实。

import (
	"context"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/repository"
)

var lpRefLog = logger.WithModule("lp_ref_material")

const (
	refCompressSceneCode     = "lesson_plan"
	refVisionSceneCode       = "scanner"
	refCompressInputMaxRunes = 40000
	refVisionOutputMaxRunes  = 12000
)

// LessonPlanRefService 参考资料处理服务。
type LessonPlanRefService struct {
	cfg interface{ GetAESKey() string }
}

// NewLessonPlanRefService 创建参考资料处理服务。
func NewLessonPlanRefService(
	cfg interface{ GetAESKey() string },
) *LessonPlanRefService {
	return &LessonPlanRefService{cfg: cfg}
}

// buildRefCompressSystemPrompt 构建长文本压缩提示词。
func (s *LessonPlanRefService) buildRefCompressSystemPrompt(
	subject string,
	grade string,
) string {
	var builder strings.Builder

	builder.WriteString(
		"你是一名严谨的教研资料整理员。请把较长的备课参考资料压缩成便于后续检索和引用的结构化要点。\n\n",
	)
	builder.WriteString("【真实性铁律】\n")
	builder.WriteString(
		"1. 只能整理原文已经出现的内容，禁止补充常识、纠错式改写或推测原文意图。\n",
	)
	builder.WriteString(
		"2. 所有专有名词、篇名、人名、地名、动物名、数字、数据、题干、选项和原文结论必须保持原样，不得替换成相近概念。\n",
	)
	builder.WriteString(
		"3. 原文中的【第N页】页码标记必须保留，便于后续追溯来源。\n",
	)
	builder.WriteString(
		"4. 看不清、缺失或互相矛盾的内容要明确保留不确定性，绝不能自行补齐。\n\n",
	)

	builder.WriteString("【压缩目标】\n")
	builder.WriteString(
		"1. 保留核心知识点、教学要求、重点难点、关键定义、事实、数据、例子、题目与页面结构。\n",
	)
	builder.WriteString(
		"2. 去除页眉页脚、重复套话、纯装饰性描述和明显排版噪声。\n",
	)
	builder.WriteString(
		"3. 相同内容可以合并，但不能合并或改写事实实体。\n\n",
	)

	builder.WriteString("【输出格式】\n")
	builder.WriteString(
		"- 直接输出结构化要点纯文本，可用小标题和分条；不要输出前言、代码围栏或JSON。\n",
	)
	builder.WriteString(
		"- 篇幅原则上控制在原文二分之一以内；事实密集、题目密集或课文原文不应为了追求短而删掉关键信息。\n",
	)

	if strings.TrimSpace(subject) != "" ||
		strings.TrimSpace(grade) != "" {
		builder.WriteString("\n【聚焦范围】\n")
		builder.WriteString(fmt.Sprintf(
			"本资料服务于【%s】【%s】备课。只调整信息组织顺序，不得因学科年级判断而删除或改写原文事实。\n",
			upDashRef(subject),
			upDashRef(grade),
		))
	}

	return builder.String()
}

// buildRefVisionSystemPrompt 构建扫描页忠实转录提示词。
func (s *LessonPlanRefService) buildRefVisionSystemPrompt(
	subject string,
	grade string,
) string {
	var builder strings.Builder

	builder.WriteString(
		"你是教材和教学资料的逐页视觉转录器，不是问答助手，也不是内容创作者。\n",
	)
	builder.WriteString(
		"你的唯一任务是把当前这一页图片中真实可见的文字与版面层级忠实转成纯文本。\n\n",
	)

	builder.WriteString("【绝对禁止】\n")
	builder.WriteString(
		"1. 禁止补充图片中没有出现的动物、人名、地名、数据、知识点、教材版本或页码。\n",
	)
	builder.WriteString(
		"2. 禁止根据常识、上下文或文件名猜测缺失文字；禁止改写、概括、纠错、续写和解释。\n",
	)
	builder.WriteString(
		"3. 禁止把插图内容想象成正文，禁止把旁栏、注释、练习题和正文混为一段。\n",
	)
	builder.WriteString(
		"4. 禁止输出LaTeX、HTML、Markdown代码块或任何分析过程。\n\n",
	)

	builder.WriteString("【转录规则】\n")
	builder.WriteString(
		"1. 按从上到下、从左到右的可见阅读顺序输出。\n",
	)
	builder.WriteString(
		"2. 标题、正文、旁栏/注释、图表文字、练习题分别换行；可用【标题】【正文】【旁栏】【图表】【练习】标记真实可见的版块。\n",
	)
	builder.WriteString(
		"3. 原文标点、数字、专有名词和题目选项尽量逐字保留。\n",
	)
	builder.WriteString(
		"4. 某个字确实无法辨认时写【无法辨认】；整块看不清时写【此处无法辨认】，不要猜。\n",
	)
	builder.WriteString(
		"5. 只输出转录结果，不要说“图片中显示”“识别结果如下”等套话。\n",
	)

	if strings.TrimSpace(subject) != "" ||
		strings.TrimSpace(grade) != "" {
		builder.WriteString(
			"\n【资料标签，仅用于识别语境，不得据此补写】\n",
		)
		builder.WriteString(fmt.Sprintf(
			"学科：%s；年级：%s。\n",
			upDashRef(subject),
			upDashRef(grade),
		))
	}

	return builder.String()
}

// CompressRefMaterial 把长参考资料原文压缩为结构化要点。
func (s *LessonPlanRefService) CompressRefMaterial(
	ctx context.Context,
	callerID string,
	content string,
	fileName string,
	subject string,
	grade string,
) (string, int, int, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", 0, 0, fmt.Errorf("参考资料内容为空")
	}

	originalRunes := []rune(content)
	originalLen := len(originalRunes)
	if originalLen > refCompressInputMaxRunes {
		content = string(originalRunes[:refCompressInputMaxRunes])
		lpRefLog.Info(
			"参考资料原文超上限，已截断后压缩",
			"file", fileName,
			"orig_runes", originalLen,
			"truncated_to", refCompressInputMaxRunes,
		)
	}

	aiConfig, err := aiClient.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		refCompressSceneCode,
		"",
		"",
		"",
	)
	if err != nil {
		return "", originalLen, 0, fmt.Errorf(
			"AI配置加载失败: %w",
			err,
		)
	}

	traceContext := buildLessonPlanRefTraceContext(
		ctx,
		callerID,
		refCompressSceneCode,
	)
	result, err := aiClient.CallAI(
		aiConfig,
		s.buildRefCompressSystemPrompt(subject, grade),
		s.buildRefCompressUserPrompt(content, fileName),
		traceContext,
	)
	if err != nil {
		return "", originalLen, 0, fmt.Errorf(
			"参考资料压缩失败: %w",
			err,
		)
	}
	if result == nil {
		return "", originalLen, 0, fmt.Errorf(
			"参考资料压缩结果为空",
		)
	}

	compressed := strings.TrimSpace(result.Content)
	if compressed == "" {
		return "", originalLen, 0, fmt.Errorf(
			"压缩结果为空，请重试",
		)
	}

	compressedLen := len([]rune(compressed))
	lpRefLog.Info(
		"参考资料压缩完成",
		"file", fileName,
		"caller", callerID,
		"orig_len", originalLen,
		"compressed_len", compressedLen,
		"tokens", result.TokensUsed,
	)
	return compressed, originalLen, compressedLen, nil
}

// TranscribeRefMaterialPage 忠实转录扫描 PDF 的单页图片。
func (s *LessonPlanRefService) TranscribeRefMaterialPage(
	ctx context.Context,
	callerID string,
	imageDataURI string,
	fileName string,
	pageNumber int,
	totalPages int,
	subject string,
	grade string,
) (string, error) {
	imageDataURI = strings.TrimSpace(imageDataURI)
	if imageDataURI == "" {
		return "", fmt.Errorf("扫描页图片为空")
	}

	aiConfig, err := aiClient.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		refVisionSceneCode,
		"",
		"",
		"",
	)
	if err != nil {
		return "", fmt.Errorf(
			"视觉识别配置加载失败: %w",
			err,
		)
	}

	userPrompt := fmt.Sprintf(
		"资料文件：%s\n当前页：第%d页，共%d页。\n请严格按系统规则，只转录这一页图片中真实可见的内容。",
		upDashRef(fileName),
		pageNumber,
		totalPages,
	)
	result, err := aiClient.CallAIMultimodal(
		aiConfig,
		s.buildRefVisionSystemPrompt(subject, grade),
		userPrompt,
		imageDataURI,
		buildLessonPlanRefTraceContext(
			ctx,
			callerID,
			refVisionSceneCode,
		),
	)
	if err != nil {
		return "", fmt.Errorf(
			"扫描页视觉转录失败: %w",
			err,
		)
	}
	if result == nil {
		return "", fmt.Errorf(
			"扫描页视觉转录结果为空",
		)
	}

	text := strings.TrimSpace(result.Content)
	if text == "" {
		return "", fmt.Errorf(
			"扫描页视觉转录结果为空",
		)
	}

	outputRunes := []rune(text)
	if len(outputRunes) > refVisionOutputMaxRunes {
		text = string(outputRunes[:refVisionOutputMaxRunes])
		lpRefLog.Warn(
			"扫描页转录结果超过单页上限，已安全截断",
			"file", fileName,
			"page", pageNumber,
			"original_runes", len(outputRunes),
			"max_runes", refVisionOutputMaxRunes,
		)
	}

	lpRefLog.Info(
		"参考资料扫描页视觉转录完成",
		"file", fileName,
		"page", pageNumber,
		"total_pages", totalPages,
		"caller", callerID,
		"model", result.ModelUsed,
		"text_runes", len([]rune(text)),
		"tokens", result.TokensUsed,
	)
	return text, nil
}

func (s *LessonPlanRefService) buildRefCompressUserPrompt(
	content string,
	fileName string,
) string {
	var builder strings.Builder
	if strings.TrimSpace(fileName) != "" {
		builder.WriteString(fmt.Sprintf(
			"【资料文件名】%s\n\n",
			fileName,
		))
	}
	builder.WriteString("【待压缩的参考资料原文】\n")
	builder.WriteString(content)
	builder.WriteString(
		"\n\n请严格按系统指令整理。任何原文没有的事实都不得添加。",
	)
	return builder.String()
}

func buildLessonPlanRefTraceContext(
	ctx context.Context,
	callerID string,
	sceneCode string,
) *aiClient.TraceContext {
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, callerID)
	userID := callerID
	return &aiClient.TraceContext{
		SceneCode: sceneCode,
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}
}

func upDashRef(value string) string {
	if strings.TrimSpace(value) == "" {
		return "未指定"
	}
	return value
}
