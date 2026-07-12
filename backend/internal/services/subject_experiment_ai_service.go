package services

// subject_experiment_ai_service.go — 物理/化学/生命科学组件 AI 生成服务（第6批A）
//
// 设计原则：
//   1. 不新增 AI 场景码，复用 math_graph_ai_service.go 中的 courseware_media_prompt，
//      继续走已验证的 Gemini 多模态、境内外分流、积分钩子。
//   2. 不新增路由，配合 math_graph_handler.go 的 target 字段复用 /math-graph/generate。
//   3. AI 生成的是“完整 HTML 组件片段”，不是一整页，也不是 React 代码。
//   4. 组件必须使用 __ROOT_ID__ 占位符，前端预览/融入时替换为真实 rootId。
//   5. 组件必须使用平台已有类名协议：
//        物理：pl-head / pl-body / pl-controls / pl-stage / pl-row / pl-result
//        化学：ce-head / ce-body / ce-controls / ce-stage / ce-row / ce-result
//      这样第5批A的底部课堂控制条布局覆盖 CSS 可以自动生效。

import (
	"context"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/repository"
)

const (
	subjectExpDescMaxRunes  = 2400
	subjectExpBaseMaxRunes  = 90000
	subjectExpImageMaxChars = 12000000
)

type SubjectExperimentGenInput struct {
	// Target: physics_lab / chem_experiment / biology_lab
	Target string
	// Mode: adapt / create
	Mode string
	// Description 老师自然语言要求
	Description string
	// BaseCode adapt 模式的底稿 HTML
	BaseCode string
	// TemplateName 底稿模板名称
	TemplateName string
	// Image data URI
	Image string
}

func subjectExperimentTargetName(target string) string {
	switch target {
	case "physics_lab":
		return "物理实验"
	case "chem_experiment":
		return "化学实验"
	case "biology_lab":
		return "生命科学互动观察"
	default:
		return "学科实验"
	}
}

func subjectExperimentClassPrefix(target string) string {
	switch target {
	case "chem_experiment":
		return "ce"
	case "biology_lab":
		return "bl"
	default:
		return "pl"
	}
}

func buildSubjectExperimentSystemPrompt(target string, mode string, hasImage bool) string {
	name := subjectExperimentTargetName(target)
	prefix := subjectExperimentClassPrefix(target)

	var b strings.Builder
	b.WriteString("你是一名精通 K12 互动课件工程的前端组件工程师，正在为 TE-DNA 课件工坊生成" + name + "组件。\n")
	b.WriteString("你的输出将被直接放入课件页面，必须稳定、可交互、可离线运行。\n\n")

	b.WriteString("【输出格式硬约束】\n")
	b.WriteString("1. 只输出一个完整 HTML 片段，不要 Markdown 代码围栏，不要解释文字，不要前言后语。\n")
	b.WriteString("2. 片段根节点必须是：<div id=\"__ROOT_ID__\"> ... </div>，必须保留 __ROOT_ID__ 占位符，前端会替换成真实 id。\n")
	b.WriteString("3. 禁止输出 <html>、<head>、<body>，禁止输出整页文档。\n")
	b.WriteString("4. 组件必须完全自包含：可以包含 <style>、SVG、canvas、原生 JavaScript；禁止外部依赖、禁止网络请求、禁止 fetch/XMLHttpRequest、禁止 import、禁止外链 script/link。\n")
	b.WriteString("5. JavaScript 必须限定在 root 内部查询 DOM：var root=document.getElementById('__ROOT_ID__'); if(!root)return; 后续全部 root.querySelector(...)\n")
	b.WriteString("6. 控件 id / data-* / SVG / canvas / script 必须完整可运行。禁止操作 document.body，禁止 window.location，禁止 localStorage。\n")
	b.WriteString("7. 组件要适配外层尺寸，不要写死 680×414 为唯一可见区域；SVG/canvas 可以用 viewBox，但 CSS 应让其 width/height:100%。\n\n")

	b.WriteString("【平台结构协议，必须使用这些类名】\n")
	b.WriteString("1. 根节点内部必须包含：." + prefix + "-head、." + prefix + "-body、." + prefix + "-controls、." + prefix + "-stage。\n")
	b.WriteString("2. 推荐结构：\n")
	b.WriteString("   <div id=\"__ROOT_ID__\">\n")
	b.WriteString("     <style>所有 CSS 都用 #__ROOT_ID__ 作为前缀选择器</style>\n")
	b.WriteString("     <div class=\"" + prefix + "-head\"><div class=\"" + prefix + "-title\">标题</div><div class=\"" + prefix + "-note\">提示</div></div>\n")
	b.WriteString("     <div class=\"" + prefix + "-body\">\n")
	b.WriteString("       <div class=\"" + prefix + "-controls\">滑杆/按钮/结果说明</div>\n")
	b.WriteString("       <div class=\"" + prefix + "-stage\">SVG 或 canvas 主体</div>\n")
	b.WriteString("     </div>\n")
	b.WriteString("     <script>(function(){ var root=document.getElementById('__ROOT_ID__'); if(!root)return; ... })();</script>\n")
	b.WriteString("   </div>\n")
	b.WriteString("3. 每个控制项用 ." + prefix + "-row；结果说明用 ." + prefix + "-result；标签和值用 ." + prefix + "-label / ." + prefix + "-value。\n")
	b.WriteString("4. 外层系统会把 ." + prefix + "-controls 自动改成底部课堂控制条，所以不要假设 controls 永远在左侧。\n\n")

	if target == "physics_lab" {
		b.WriteString("【物理实验语义要求】\n")
		b.WriteString("1. 适合电学、光学、波动、电磁、热学、力与运动的非 Matter.js 轻量实验。\n")
		b.WriteString("2. 至少包含 1-3 个可调变量，例如电压/电阻/入射角/折射率/焦距/波长/速度/磁场强度等。\n")
		b.WriteString("3. 需要实时读数或结论，例如 I=U/R、n₁sinθ₁=n₂sinθ₂、v=λf、感应强度变化等。\n")
		b.WriteString("4. 图形主体优先使用 SVG；需要连续动画时可用 canvas + requestAnimationFrame。\n")
	} else if target == "biology_lab" {
		b.WriteString("【生命科学组件语义要求】\n")
		b.WriteString("1. 适合显微镜观察、细胞结构、生命过程、人体系统、遗传、生态和微生物等课堂内容。\n")
		b.WriteString("2. 至少包含 1-3 个可调变量或可切换结构，例如观察倍数、焦距、结构选择、过程进度、环境条件等。\n")
		b.WriteString("3. 必须动态呈现结构、位置关系或生命过程，不要只生成静态装饰图。\n")
		b.WriteString("4. 结构名称和功能说明必须符合 K12 生物学常识；示意图需明确注明是模型，不伪装成真实显微照片。\n")
		b.WriteString("5. 优先使用 SVG，使用 bl-head / bl-body / bl-controls / bl-stage / bl-row / bl-result 类名协议。\n")
	} else {
		b.WriteString("【化学实验语义要求】\n")
		b.WriteString("1. 适合实验装置、操作流程、反应现象、变量控制、物质检验、酸碱盐、电化学等。\n")
		b.WriteString("2. 至少包含 1-3 个可调变量，例如滴加量、浓度、加热强度、反应进度、静置时间、电流强度等。\n")
		b.WriteString("3. 需要动态呈现实验现象，例如气泡、沉淀、颜色变化、液面变化、晶体析出、pH变化等。\n")
		b.WriteString("4. 注意安全与教学准确性：不要生成危险操作步骤，只做课堂模拟演示。\n")
	}
	b.WriteString("\n")

	if hasImage {
		b.WriteString("【图片输入要求】\n")
		b.WriteString("用户附了题目/实验装置/教材截图。你要先读图，提取实验对象、变量、装置结构和需要表达的核心现象，再生成对应互动组件。\n")
		b.WriteString("图片模糊时基于可辨认部分生成，不要臆造复杂细节。\n\n")
	}

	if mode == "adapt" {
		b.WriteString("【本次任务：模板改编】\n")
		b.WriteString("用户会给你一个现有组件 HTML 底稿和改编要求。你要在底稿基础上做最小必要修改：\n")
		b.WriteString("1. 保留底稿的 rootId 协议、类名协议、控件结构、脚本运行方式。\n")
		b.WriteString("2. 能不动的布局、风格、控件绑定尽量不动。\n")
		b.WriteString("3. 根据用户要求替换实验主题、变量、读数、SVG/canvas 图形和结论说明。\n")
		b.WriteString("4. 输出修改后的完整 HTML 片段，不是差异片段。\n")
	} else {
		b.WriteString("【本次任务：从零生成】\n")
		b.WriteString("用户会描述一个新实验或上传图片。你要从零生成符合平台结构协议的完整 HTML 片段。\n")
		b.WriteString("必须有清晰实验主体、课堂控制条、动态读数/现象、结论说明。\n")
	}

	return b.String()
}

func buildSubjectExperimentUserPrompt(in *SubjectExperimentGenInput) string {
	var b strings.Builder
	hasImage := strings.TrimSpace(in.Image) != ""
	if in.Mode == "adapt" {
		if strings.TrimSpace(in.TemplateName) != "" {
			b.WriteString("【底稿模板名称】" + strings.TrimSpace(in.TemplateName) + "\n\n")
		}
		b.WriteString("【底稿 HTML 组件，基于它做最小必要修改】\n")
		b.WriteString(in.BaseCode)
		if hasImage {
			b.WriteString("\n\n【用户补充说明，图片也要参考】\n")
		} else {
			b.WriteString("\n\n【用户要求的变化】\n")
		}
		b.WriteString(in.Description)
		b.WriteString("\n\n请输出修改后的完整 HTML 片段。")
	} else {
		if hasImage {
			b.WriteString("【用户补充说明，主要内容见附图】\n")
		} else {
			b.WriteString("【用户描述的新实验】\n")
		}
		b.WriteString(in.Description)
		b.WriteString("\n\n请输出完整 HTML 片段。")
	}
	return b.String()
}

func validateSubjectExperimentHTML(target string, html string) error {
	trimmed := strings.TrimSpace(html)
	low := strings.ToLower(trimmed)
	prefix := subjectExperimentClassPrefix(target)

	if trimmed == "" {
		return fmt.Errorf("生成结果为空")
	}
	if !strings.Contains(trimmed, "__ROOT_ID__") {
		return fmt.Errorf("生成结果缺少 __ROOT_ID__ 占位符")
	}
	if !strings.Contains(trimmed, "<div") || !strings.Contains(trimmed, "</div>") {
		return fmt.Errorf("生成结果不是完整 HTML 片段")
	}
	if strings.Contains(low, "<html") || strings.Contains(low, "<body") || strings.Contains(low, "<head") {
		return fmt.Errorf("生成结果包含整页 HTML 标签，已拦截")
	}
	if strings.Contains(low, "src=\"http") || strings.Contains(low, "href=\"http") || strings.Contains(low, "fetch(") || strings.Contains(low, "xmlhttprequest") || strings.Contains(low, "import ") {
		return fmt.Errorf("生成结果包含外部依赖或网络请求，已拦截")
	}
	if strings.Contains(low, "document.body") || strings.Contains(low, "document.write") || strings.Contains(low, "window.location") || strings.Contains(low, "localstorage") {
		return fmt.Errorf("生成结果包含越界 DOM 或浏览器状态操作，已拦截")
	}
	required := []string{
		prefix + "-head",
		prefix + "-body",
		prefix + "-controls",
		prefix + "-stage",
	}
	for _, cls := range required {
		if !strings.Contains(trimmed, cls) {
			return fmt.Errorf("生成结果缺少必要类名 .%s", cls)
		}
	}
	if !strings.Contains(low, "<script") {
		return fmt.Errorf("生成结果缺少交互脚本 <script>")
	}
	return nil
}

func (s *MathGraphAIService) GenerateSubjectExperiment(ctx context.Context, callerID string, in *SubjectExperimentGenInput) (string, error) {
	if in == nil {
		return "", fmt.Errorf("请求参数为空")
	}

	in.Target = strings.TrimSpace(in.Target)
	if in.Target != "physics_lab" && in.Target != "chem_experiment" && in.Target != "biology_lab" {
		return "", fmt.Errorf("target 必须是 physics_lab、chem_experiment 或 biology_lab")
	}

	in.Mode = strings.TrimSpace(in.Mode)
	if in.Mode != "adapt" && in.Mode != "create" {
		return "", fmt.Errorf("mode 必须是 adapt 或 create")
	}

	in.Description = strings.TrimSpace(in.Description)
	in.Image = strings.TrimSpace(in.Image)
	hasImage := in.Image != ""

	if in.Description == "" && !hasImage {
		return "", fmt.Errorf("实验描述为空")
	}
	if in.Description == "" {
		in.Description = "请根据附图生成对应的互动实验组件。"
	}
	if descRunes := []rune(in.Description); len(descRunes) > subjectExpDescMaxRunes {
		in.Description = string(descRunes[:subjectExpDescMaxRunes])
	}

	if hasImage {
		if !strings.HasPrefix(in.Image, "data:image/") {
			return "", fmt.Errorf("图片格式不正确，需为 data:image/... 的 base64 数据")
		}
		if len(in.Image) > subjectExpImageMaxChars {
			return "", fmt.Errorf("图片过大，请压缩后重试")
		}
	}

	in.BaseCode = strings.TrimSpace(in.BaseCode)
	if in.Mode == "adapt" {
		if in.BaseCode == "" {
			return "", fmt.Errorf("模板改编模式必须提供底稿 HTML base_code")
		}
		if baseRunes := []rune(in.BaseCode); len(baseRunes) > subjectExpBaseMaxRunes {
			return "", fmt.Errorf("底稿 HTML 过长，已超过 %d 字符", subjectExpBaseMaxRunes)
		}
	}

	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), mathGraphSceneCode, "", "", "")
	if err != nil {
		return "", fmt.Errorf("AI配置加载失败: %w", err)
	}

	systemPrompt := buildSubjectExperimentSystemPrompt(in.Target, in.Mode, hasImage)
	userPrompt := buildSubjectExperimentUserPrompt(in)

	schoolID, _ := repository.GetSchoolIDByUserID(ctx, callerID)
	uid := callerID
	traceCtx := &aiClient.TraceContext{
		SceneCode: mathGraphSceneCode,
		UserID:    &uid,
		SchoolID:  schoolIDPtr(schoolID),
	}

	var result *aiClient.CallResult
	if hasImage {
		result, err = aiClient.CallAIMultimodal(aiCfg, systemPrompt, userPrompt, in.Image, traceCtx)
	} else {
		result, err = aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	}
	if err != nil {
		return "", fmt.Errorf("实验组件生成失败: %w", err)
	}

	html := stripMathGraphFences(result.Content)
	if err := validateSubjectExperimentHTML(in.Target, html); err != nil {
		return "", err
	}

	mathGraphLog.Info("学科实验HTML组件生成完成",
		"target", in.Target,
		"mode", in.Mode,
		"caller", callerID,
		"template", in.TemplateName,
		"with_image", hasImage,
		"desc_len", len([]rune(in.Description)),
		"html_len", len([]rune(html)),
		"tokens", result.TokensUsed,
	)

	return html, nil
}
