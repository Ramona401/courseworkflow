package services

// geography_experiment_ai_service.go — 地理互动探究组件AI生成服务
//
// 设计原则：
//   1. 复用courseware_media_prompt场景，不新增AI场景码。
//   2. 复用SubjectExperimentGenInput请求结构。
//   3. 输出自包含HTML片段，不输出React或完整HTML文档。
//   4. 地理组件统一使用gl-*结构协议。
//   5. 禁止在线地图、外部脚本、外部数据和网络请求。
//   6. 所有地理模型必须明确属于课堂教学简化模型。

import (
	"context"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/repository"
)

func buildGeographyExperimentSystemPrompt(
	mode string,
	hasImage bool,
) string {
	var b strings.Builder

	b.WriteString("你是一名精通K12地理教学和互动课件工程的前端组件工程师。\n")
	b.WriteString("你正在为TE-DNA课件工坊生成一个地理互动探究组件。\n")
	b.WriteString("组件会直接嵌入课件页面，必须稳定、可交互、可离线运行。\n\n")

	b.WriteString("【输出格式硬约束】\n")
	b.WriteString("1. 只输出一个完整HTML片段，不输出Markdown围栏、解释、前言或后语。\n")
	b.WriteString("2. 根节点必须为<div id=\"__ROOT_ID__\">...</div>，必须保留__ROOT_ID__占位符。\n")
	b.WriteString("3. 禁止输出html、head、body等整页标签。\n")
	b.WriteString("4. 可以使用HTML、CSS、SVG、canvas和原生JavaScript。\n")
	b.WriteString("5. 禁止React、Vue、外部图片、在线地图、地图瓦片、在线字体、CDN、外链脚本和外链样式。\n")
	b.WriteString("6. 禁止fetch、XMLHttpRequest、WebSocket、import和任何网络请求。\n")
	b.WriteString("7. 脚本必须先执行var root=document.getElementById('__ROOT_ID__'); if(!root)return;。\n")
	b.WriteString("8. 后续DOM查询只能使用root.querySelector或root.querySelectorAll，不得使用document.querySelector或document.querySelectorAll。\n")
	b.WriteString("9. 禁止document.body、document.write、window.location和localStorage。\n")
	b.WriteString("10. 必须支持同一课件页出现多个独立实例，控件和动画不得互相干扰。\n")
	b.WriteString("11. 动画或定时器继续执行前必须检查root.isConnected，组件移除后应停止更新。\n")
	b.WriteString("12. SVG和canvas应使用响应式尺寸，不能把固定680×414作为唯一可见区域。\n\n")

	b.WriteString("【地理结构协议】\n")
	b.WriteString("根节点内部必须包含以下结构类名：\n")
	b.WriteString("- gl-head：标题与教学提示\n")
	b.WriteString("- gl-body：互动主体总容器\n")
	b.WriteString("- gl-controls：参数和模式控制区\n")
	b.WriteString("- gl-stage：地图、剖面、过程或图表主体\n")
	b.WriteString("- gl-row：单个控制项目\n")
	b.WriteString("- gl-result：读数、规律和结论区\n")
	b.WriteString("- gl-label：参数名称\n")
	b.WriteString("- gl-value：实时参数值\n\n")

	b.WriteString("推荐结构：\n")
	b.WriteString("<div id=\"__ROOT_ID__\">\n")
	b.WriteString("  <style>所有选择器使用#__ROOT_ID__作为前缀</style>\n")
	b.WriteString("  <div class=\"gl-head\">标题和教学提示</div>\n")
	b.WriteString("  <div class=\"gl-body\">\n")
	b.WriteString("    <div class=\"gl-controls\">滑杆、按钮、模式与结果</div>\n")
	b.WriteString("    <div class=\"gl-stage\">SVG或canvas互动主体</div>\n")
	b.WriteString("  </div>\n")
	b.WriteString("  <script>(function(){var root=document.getElementById('__ROOT_ID__');if(!root)return;...})();</script>\n")
	b.WriteString("</div>\n\n")

	b.WriteString("外层系统会把gl-controls改造成底部课堂控制条，因此不能假设控制区永远位于左侧。\n\n")

	b.WriteString("【地理教学语义要求】\n")
	b.WriteString("1. 适用于经纬网、地图判读、等高线与地形剖面、地球运动、大气运动、天气气候、水循环、河流、地质地貌、海洋、人口城市、产业区位、区域发展、遥感和GIS等内容。\n")
	b.WriteString("2. 至少包含1至3个可调参数、模式按钮、阶段按钮或图层开关。\n")
	b.WriteString("3. 必须动态呈现空间位置、方向、剖面、环流、过程变化、图层叠加或数据关系，不能只做静态装饰图。\n")
	b.WriteString("4. 必须包含实时读数、现象解释或教学结论。\n")
	b.WriteString("5. 优先使用自绘SVG；连续过程可以使用canvas和requestAnimationFrame。\n")
	b.WriteString("6. 不得虚构精确测绘数据、实时天气数据、真实遥感数据或真实行政边界。\n")
	b.WriteString("7. 必须在组件中明确说明这是教学简化模型，比例、时间和空间关系属于教学示意。\n")
	b.WriteString("8. 不得用于导航、防灾决策、工程选址或真实行政边界判定。\n")
	b.WriteString("9. 涉及地震、火山、洪水、台风等过程时，只演示地理原理，不提供现实风险决策结论。\n\n")

	if hasImage {
		b.WriteString("【图片输入要求】\n")
		b.WriteString("用户附有地图、地理图表、教材截图或题目图片。\n")
		b.WriteString("请先识别图中的地理对象、图例、方向、变量和空间关系，再生成互动组件。\n")
		b.WriteString("图片模糊时只使用可辨认内容，不臆造行政边界、坐标和数据。\n\n")
	}

	if mode == "adapt" {
		b.WriteString("【本次任务：模板改编】\n")
		b.WriteString("1. 在现有底稿基础上完成最小必要修改。\n")
		b.WriteString("2. 保留rootId协议、gl类名结构、布局和已有控件绑定。\n")
		b.WriteString("3. 根据老师要求替换地理主题、参数、SVG图形、数据读数和结论。\n")
		b.WriteString("4. 输出修改后的完整HTML片段，不输出差异代码。\n")
	} else {
		b.WriteString("【本次任务：从零生成】\n")
		b.WriteString("从零生成完整的地理互动探究组件。\n")
		b.WriteString("必须包含清晰主体、参数控制、动态图示、数据读数和教学结论。\n")
	}

	return b.String()
}

func validateGeographyExperimentHTML(html string) error {
	trimmed := strings.TrimSpace(html)
	low := strings.ToLower(trimmed)

	if trimmed == "" {
		return fmt.Errorf("生成结果为空")
	}

	if !strings.Contains(trimmed, "__ROOT_ID__") {
		return fmt.Errorf("生成结果缺少 __ROOT_ID__ 占位符")
	}

	if !strings.Contains(low, "<div") ||
		!strings.Contains(low, "</div>") {
		return fmt.Errorf("生成结果不是完整HTML片段")
	}

	if strings.Contains(low, "<html") ||
		strings.Contains(low, "<head") ||
		strings.Contains(low, "<body") {
		return fmt.Errorf("生成结果包含整页HTML标签，已拦截")
	}

	forbidden := []string{
		"src=\"http",
		"src='http",
		"href=\"http",
		"href='http",
		"fetch(",
		"xmlhttprequest",
		"websocket",
		"import ",
		"document.queryselector",
		"document.queryselectorall",
		"document.body",
		"document.write",
		"window.location",
		"localstorage",
	}

	for _, item := range forbidden {
		if strings.Contains(low, item) {
			return fmt.Errorf("生成结果包含禁止的外部依赖或越界操作：%s", item)
		}
	}

	required := []string{
		"gl-head",
		"gl-body",
		"gl-controls",
		"gl-stage",
		"gl-result",
	}

	for _, className := range required {
		if !strings.Contains(trimmed, className) {
			return fmt.Errorf("生成结果缺少必要类名 .%s", className)
		}
	}

	if !strings.Contains(low, "<script") {
		return fmt.Errorf("生成结果缺少交互脚本")
	}

	hasAnimation := strings.Contains(low, "requestanimationframe") ||
		strings.Contains(low, "setinterval(") ||
		strings.Contains(low, "settimeout(")

	if hasAnimation && !strings.Contains(trimmed, "root.isConnected") {
		return fmt.Errorf("动画或定时器缺少 root.isConnected 存活检查")
	}

	return nil
}

func (s *MathGraphAIService) GenerateGeographyExperiment(
	ctx context.Context,
	callerID string,
	in *SubjectExperimentGenInput,
) (string, error) {
	if in == nil {
		return "", fmt.Errorf("请求参数为空")
	}

	in.Target = "geography_lab"
	in.Mode = strings.TrimSpace(in.Mode)

	if in.Mode != "adapt" && in.Mode != "create" {
		return "", fmt.Errorf("mode必须是adapt或create")
	}

	in.Description = strings.TrimSpace(in.Description)
	in.Image = strings.TrimSpace(in.Image)

	hasImage := in.Image != ""

	if in.Description == "" && !hasImage {
		return "", fmt.Errorf("地理探究描述为空")
	}

	if in.Description == "" {
		in.Description = "请根据附图生成对应的地理互动探究组件。"
	}

	descriptionRunes := []rune(in.Description)
	if len(descriptionRunes) > subjectExpDescMaxRunes {
		in.Description = string(descriptionRunes[:subjectExpDescMaxRunes])
	}

	if hasImage {
		if !strings.HasPrefix(in.Image, "data:image/") {
			return "", fmt.Errorf("图片必须为data:image/...格式")
		}

		if len(in.Image) > subjectExpImageMaxChars {
			return "", fmt.Errorf("图片过大，请压缩后重试")
		}
	}

	in.BaseCode = strings.TrimSpace(in.BaseCode)

	if in.Mode == "adapt" {
		if in.BaseCode == "" {
			return "", fmt.Errorf("模板改编模式必须提供底稿HTML")
		}

		if len([]rune(in.BaseCode)) > subjectExpBaseMaxRunes {
			return "", fmt.Errorf(
				"底稿HTML过长，已超过%d字符",
				subjectExpBaseMaxRunes,
			)
		}
	}

	aiCfg, err := aiClient.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		mathGraphSceneCode,
		"",
		"",
		"",
	)
	if err != nil {
		return "", fmt.Errorf("AI配置加载失败: %w", err)
	}

	systemPrompt := buildGeographyExperimentSystemPrompt(
		in.Mode,
		hasImage,
	)

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
		result, err = aiClient.CallAIMultimodal(
			aiCfg,
			systemPrompt,
			userPrompt,
			in.Image,
			traceCtx,
		)
	} else {
		result, err = aiClient.CallAI(
			aiCfg,
			systemPrompt,
			userPrompt,
			traceCtx,
		)
	}

	if err != nil {
		return "", fmt.Errorf("地理互动组件生成失败: %w", err)
	}

	html := stripMathGraphFences(result.Content)

	if err := validateGeographyExperimentHTML(html); err != nil {
		return "", err
	}

	mathGraphLog.Info(
		"地理互动HTML组件生成完成",
		"target", "geography_lab",
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
