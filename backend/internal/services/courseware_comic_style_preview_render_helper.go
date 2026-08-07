package services

// courseware_comic_style_preview_render_helper.go
//
// 本文件把教师确认的第三步视觉选择转换为稳定图片渲染参数：
//   - 美术风格代码转换为服务端稳定画风说明；
//   - 图片比例与清晰度转换为豆包可接受的实际尺寸；
//   - 图片比例转换为明确构图要求；
//   - 清晰度转换为稳定细节要求；
//   - 教师补充要求放在旧规划IAOCI之后，拥有更高优先级；
//   - 首格样张、整批分格和后续单格重画共用同一套确认视觉事实源。
//
// 所有尺寸均满足当前图片网关总像素不少于3,686,400的要求。

import (
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
)

type coursewareComicStylePreviewRenderPlan struct {
	ImageSize string
	Prompt    string
}

// buildCoursewareComicStylePreviewRenderPlan 构建第1格样张渲染计划。
func buildCoursewareComicStylePreviewRenderPlan(
	project *models.CoursewareComicProject,
	panel *models.CoursewareComicPanel,
	workflow *models.CoursewareComicWorkflowState,
) (*coursewareComicStylePreviewRenderPlan, bool) {
	plan, valid :=
		buildCoursewareComicConfirmedPanelRenderPlan(
			project,
			panel,
			workflow,
		)

	if !valid ||
		plan == nil {
		return nil, false
	}

	plan.Prompt =
		strings.TrimSpace(
			strings.Join(
				[]string{
					plan.Prompt,
					"【样张专用要求】",
					"本次只生成第1格完整视觉样张。" +
						"画面必须达到可直接叠加HTML与SVG对白、旁白和教学卡片的完成度；" +
						"为覆盖层预留合理留白，但不得在图片中绘制气泡、文字框、" +
						"题目框或任何可读文字。",
				},
				"\n",
			),
		)

	return plan, true
}

// buildCoursewareComicConfirmedPanelRenderPlan 构建严格二选一画风的单格渲染计划。
//
// courseware：不使用六种预设画风，唯一艺术风格来自课件锚点及其派生参考图。
// selected：不读取课件风格锚点，唯一艺术风格来自教师选择的预设画风。
// 两种来源不得混合。
func buildCoursewareComicConfirmedPanelRenderPlan(
	project *models.CoursewareComicProject,
	panel *models.CoursewareComicPanel,
	workflow *models.CoursewareComicWorkflowState,
) (*coursewareComicStylePreviewRenderPlan, bool) {
	if project == nil ||
		panel == nil ||
		workflow == nil {
		return nil, false
	}

	visualStyleSource,
		visualStyle,
		aspectRatio,
		imageQuality,
		styleInstruction,
		valid :=
		normalizeCoursewareComicConfirmedVisualSelection(
			project,
			workflow,
		)

	if !valid {
		return nil, false
	}

	imageSize, valid :=
		resolveCoursewareComicStylePreviewImageSize(
			aspectRatio,
			imageQuality,
		)

	if !valid {
		return nil, false
	}

	sourceInstruction, valid :=
		coursewareComicVisualSourceInstruction(
			visualStyleSource,
			visualStyle,
		)

	aspectInstruction :=
		coursewareComicStylePreviewAspectInstruction(
			aspectRatio,
		)

	qualityInstruction :=
		coursewareComicStylePreviewQualityInstruction(
			imageQuality,
		)

	if !valid ||
		sourceInstruction == "" ||
		aspectInstruction == "" ||
		qualityInstruction == "" {
		return nil, false
	}

	promptParts :=
		[]string{
			"【最高优先级：画风来源严格二选一】",
			sourceInstruction,
			"禁止把课件风格锚点与六种漫画预设画风混合。" +
				"后续内容段落只提供人物、场景、动作、知识事实和连续性；" +
				"其中任何冲突的旧画风、材质或渲染词全部无效。",
			"【教师已确认画幅】",
			aspectInstruction,
			"严格按照所选画幅构图，不得用白边、黑边、模糊背景填充或画中画方式凑比例。",
			"【教师已确认清晰度】",
			qualityInstruction,
			"【人物、场景与知识内容事实】",
			buildCoursewareComicPanelGenerationPrompt(
				project,
				panel,
			),
		}

	if visualStyleSource ==
		models.CWComicVisualStyleSourceSelected &&
		styleInstruction != "" {
		promptParts =
			append(
				promptParts,
				"【教师补充所选画风要求】",
				styleInstruction,
				"教师补充要求只能细化当前所选预设画风，"+
					"不得引入课件风格锚点或第二种艺术媒介。",
			)
	}

	promptParts =
		append(
			promptParts,
			"【最终画风复核】",
			sourceInstruction,
			"输出前再次检查：全图只能存在上述唯一画风来源；"+
				"不得因为高清、柔和光影、人物表情或参考图而自动转成另一种三维动画电影风格。",
			"【完整漫画底图要求】",
			"本格必须形成可直接使用的完整视觉底图。"+
				"为HTML与SVG对白、旁白和教学卡片预留自然安全空间；"+
				"图片中不得绘制气泡、旁白框、题目框、公式、字幕、标签、"+
				"Logo、水印、伪字符或任何可读文字。",
		)

	prompt :=
		strings.TrimSpace(
			strings.Join(
				promptParts,
				"\n",
			),
		)

	if prompt == "" {
		return nil, false
	}

	return &coursewareComicStylePreviewRenderPlan{
		ImageSize: imageSize,
		Prompt:    prompt,
	}, true
}

// buildCoursewareComicConfirmedCharacterSheetRenderPlan
// 构建严格二选一画风的人物设定图渲染计划。
func buildCoursewareComicConfirmedCharacterSheetRenderPlan(
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
) (*coursewareComicStylePreviewRenderPlan, bool) {
	if project == nil ||
		workflow == nil {
		return nil, false
	}

	visualStyleSource,
		visualStyle,
		_,
		imageQuality,
		styleInstruction,
		valid :=
		normalizeCoursewareComicConfirmedVisualSelection(
			project,
			workflow,
		)

	if !valid {
		return nil, false
	}

	imageSize, valid :=
		resolveCoursewareComicStylePreviewImageSize(
			models.CWComicAspectRatio16x9,
			imageQuality,
		)

	if !valid {
		return nil, false
	}

	sourceInstruction, valid :=
		coursewareComicVisualSourceInstruction(
			visualStyleSource,
			visualStyle,
		)

	qualityInstruction :=
		coursewareComicStylePreviewQualityInstruction(
			imageQuality,
		)

	if !valid ||
		sourceInstruction == "" ||
		qualityInstruction == "" {
		return nil, false
	}

	promptParts :=
		[]string{
			"【最高优先级：人物设定图画风来源严格二选一】",
			sourceInstruction,
			"禁止把课件风格锚点与六种漫画预设画风混合。",
			"【人物身份与造型事实】",
			buildCoursewareComicCharacterSheetPrompt(
				project,
			),
			"【人物设定图画幅】",
			"横向16:9人物设定参考图；全部角色分开排列，" +
				"主体完整，不得贴边、遮挡或被裁切。",
			"【教师已确认清晰度】",
			qualityInstruction,
		}

	if visualStyleSource ==
		models.CWComicVisualStyleSourceSelected &&
		styleInstruction != "" {
		promptParts =
			append(
				promptParts,
				"【教师补充所选画风要求】",
				styleInstruction,
			)
	}

	promptParts =
		append(
			promptParts,
			"【最终画风复核】",
			sourceInstruction,
			"人物身份、服饰和固定特征可以继承；"+
				"不得把旧规划中的画风词或另一种艺术媒介带入结果。",
			"【人物设定图硬性约束】",
			"只展示人物、动物或拟人知识对象的固定视觉特征；"+
				"不得生成姓名、编号、标签、对白、公式、Logo、水印、"+
				"伪字符或任何可读文字。",
		)

	prompt :=
		strings.TrimSpace(
			strings.Join(
				promptParts,
				"\n",
			),
		)

	if prompt == "" {
		return nil, false
	}

	return &coursewareComicStylePreviewRenderPlan{
		ImageSize: imageSize,
		Prompt:    prompt,
	}, true
}

func coursewareComicVisualSourceInstruction(
	visualStyleSource string,
	visualStyle string,
) (string, bool) {
	switch strings.TrimSpace(
		visualStyleSource,
	) {
	case models.CWComicVisualStyleSourceCourseware:
		return "只跟随本次请求提供的课件风格锚点图像。" +
				"该参考图是唯一艺术风格来源，只继承其媒介、色彩、线条、" +
				"材质、光影和整体视觉语言；不使用科学百科、温暖绘本、" +
				"现代扁平、现代国风、电影级3D或写实插画中的任何预设画风说明。",
			true

	case models.CWComicVisualStyleSourceSelected:
		instruction :=
			models.CoursewareComicVisualStyleInstruction(
				visualStyle,
			)

		if instruction == "" {
			return "", false
		}

		return "只使用老师选择的漫画预设画风，不读取或模仿课件风格锚点。" +
				instruction,
			true

	default:
		return "", false
	}
}

func normalizeCoursewareComicConfirmedVisualSelection(
	project *models.CoursewareComicProject,
	workflow *models.CoursewareComicWorkflowState,
) (
	visualStyleSource string,
	visualStyle string,
	aspectRatio string,
	imageQuality string,
	styleInstruction string,
	valid bool,
) {
	if project == nil ||
		workflow == nil {
		return "", "", "", "", "", false
	}

	visualStyleSource =
		strings.TrimSpace(
			workflow.VisualStyleSource,
		)

	visualStyle =
		strings.TrimSpace(
			project.VisualStyle,
		)

	aspectRatio =
		strings.TrimSpace(
			workflow.AspectRatio,
		)

	imageQuality =
		strings.TrimSpace(
			workflow.ImageQuality,
		)

	styleInstruction =
		strings.TrimSpace(
			workflow.StyleInstruction,
		)

	if !models.IsValidCWComicVisualStyleSource(
		visualStyleSource,
	) ||
		!models.IsValidCWComicVisualStyle(
			visualStyle,
		) ||
		!models.IsValidCWComicAspectRatio(
			aspectRatio,
		) ||
		!models.IsValidCWComicImageQuality(
			imageQuality,
		) ||
		utf8.RuneCountInString(
			styleInstruction,
		) >
			models.CoursewareComicMaxStyleInstructionRunes {
		return "", "", "", "", "", false
	}

	return visualStyleSource,
		visualStyle,
		aspectRatio,
		imageQuality,
		styleInstruction,
		true
}

func resolveCoursewareComicStylePreviewImageSize(
	aspectRatio string,
	imageQuality string,
) (string, bool) {
	aspectRatio =
		strings.TrimSpace(
			aspectRatio,
		)

	imageQuality =
		strings.TrimSpace(
			imageQuality,
		)

	if !models.IsValidCWComicAspectRatio(
		aspectRatio,
	) ||
		!models.IsValidCWComicImageQuality(
			imageQuality,
		) {
		return "", false
	}

	if imageQuality ==
		models.CWComicImageQualityHigh {
		switch aspectRatio {
		case models.CWComicAspectRatioCourseware,
			models.CWComicAspectRatio16x9:
			return "3200x1800", true

		case models.CWComicAspectRatio4x3:
			return "3072x2304", true

		case models.CWComicAspectRatio1x1:
			return "2560x2560", true

		case models.CWComicAspectRatio3x4:
			return "2304x3072", true

		case models.CWComicAspectRatio9x16:
			return "1800x3200", true
		}
	}

	switch aspectRatio {
	case models.CWComicAspectRatioCourseware,
		models.CWComicAspectRatio16x9:
		return "2560x1440", true

	case models.CWComicAspectRatio4x3:
		return "2304x1728", true

	case models.CWComicAspectRatio1x1:
		return "1920x1920", true

	case models.CWComicAspectRatio3x4:
		return "1728x2304", true

	case models.CWComicAspectRatio9x16:
		return "1440x2560", true

	default:
		return "", false
	}
}

func coursewareComicStylePreviewAspectInstruction(
	aspectRatio string,
) string {
	switch strings.TrimSpace(
		aspectRatio,
	) {
	case models.CWComicAspectRatioCourseware:
		return "横向课件画幅，16:9；主体适合在1920×1080课件页面中完整展示，" +
			"关键人物和知识对象避开画面边缘"

	case models.CWComicAspectRatio16x9:
		return "横向16:9电影式画幅；保持清楚的左右空间关系和横向叙事层次"

	case models.CWComicAspectRatio4x3:
		return "横向4:3经典教学画幅；主体集中，适合人物对话和知识对象并列展示"

	case models.CWComicAspectRatio1x1:
		return "正方形1:1画幅；主体居中稳定，四周保留均衡安全边距"

	case models.CWComicAspectRatio3x4:
		return "竖向3:4画幅；利用纵向空间组织人物、动作和知识对象，避免顶部或底部拥挤"

	case models.CWComicAspectRatio9x16:
		return "竖向9:16画幅；采用清楚的纵深或上下叙事结构，主体保持完整且不贴边"

	default:
		return ""
	}
}

func coursewareComicStylePreviewQualityInstruction(
	imageQuality string,
) string {
	switch strings.TrimSpace(
		imageQuality,
	) {
	case models.CWComicImageQualityStandard:
		return "标准清晰度；教学主体轮廓清楚，人物表情可辨，知识对象结构准确，" +
			"细节适度，避免无意义纹理、噪点和过度锐化"

	case models.CWComicImageQualityHigh:
		return "高清精细质量；根据唯一画风来源呈现人物、知识对象和场景细节，" +
			"边缘完整、层次清楚，避免噪点和过度锐化；" +
			"二维画风不得因为高清要求转成三维材质、塑料高光或体积渲染"

	default:
		return ""
	}
}
