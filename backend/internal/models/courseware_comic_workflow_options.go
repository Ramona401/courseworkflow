package models

// courseware_comic_workflow_options.go — 漫画叙事与视觉选项
//
// 稳定代码用于：
//   - 第二步叙事方式选择；
//   - 第三步美术风格选择；
//   - 服务端请求校验；
//   - 图片提示词构建。

import "strings"

// 漫画叙事方式。
const (
	CWComicNarrativeKnowledgeStory  = "knowledge_story"
	CWComicNarrativeInquiryMystery  = "inquiry_mystery"
	CWComicNarrativeRoleDialogue    = "role_dialogue"
	CWComicNarrativeTravelAdventure = "travel_adventure"
	CWComicNarrativeCivicCase       = "civic_case"
)

// 漫画美术风格。
const (
	CWComicVisualScienceEncyclopedia   = "science_encyclopedia"
	CWComicVisualWarmStorybook         = "warm_storybook"
	CWComicVisualModernFlat            = "modern_flat"
	CWComicVisualChineseInk            = "chinese_ink"
	CWComicVisualCinematic3D           = "cinematic_3d"
	CWComicVisualRealisticIllustration = "realistic_illustration"
)

// CoursewareComicMaxStyleInstructionRunes 是教师风格补充要求业务上限。
const CoursewareComicMaxStyleInstructionRunes = 1200

// IsValidCWComicNarrativeMode 校验叙事方式。
func IsValidCWComicNarrativeMode(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicNarrativeKnowledgeStory,
		CWComicNarrativeInquiryMystery,
		CWComicNarrativeRoleDialogue,
		CWComicNarrativeTravelAdventure,
		CWComicNarrativeCivicCase:
		return true

	default:
		return false
	}
}

// IsValidCWComicVisualStyle 校验美术风格。
func IsValidCWComicVisualStyle(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicVisualScienceEncyclopedia,
		CWComicVisualWarmStorybook,
		CWComicVisualModernFlat,
		CWComicVisualChineseInk,
		CWComicVisualCinematic3D,
		CWComicVisualRealisticIllustration:
		return true

	default:
		return false
	}
}

// CoursewareComicVisualStyleInstruction 返回服务端稳定画风合同。
//
// 除cinematic_3d外，其余风格均明确要求二维媒介，
// 防止图片模型把“柔和、精细、电影光影”等宽泛词自动解释成
// 圆润塑料材质的商业三维动画电影效果。
func CoursewareComicVisualStyleInstruction(
	value string,
) string {
	switch strings.TrimSpace(
		value,
	) {
	case CWComicVisualScienceEncyclopedia:
		return "严格二维科学百科漫画。使用清晰墨线、准确结构、平涂或轻量赛璐璐明暗，" +
			"可使用剖面、观察图和概念对比式构图；人物比例自然，知识对象优先准确。" +
			"禁止三维建模、塑料或橡胶材质、体积渲染、电影级景深、" +
			"圆润大头动画电影角色比例和商业三维动画电影质感。"

	case CWComicVisualWarmStorybook:
		return "严格二维温暖教育绘本。使用手绘水彩、彩铅或粉彩笔触，保留纸张纹理、" +
			"柔和边缘和自然儿童绘本人物比例；画面亲切但不过度幼态。" +
			"禁止三维建模、塑料皮肤、橡胶玩偶材质、体积光、" +
			"圆润大眼动画电影造型和商业三维动画电影质感。"

	case CWComicVisualModernFlat:
		return "严格二维现代扁平教育插画。使用矢量式轮廓、几何色块、有限层级阴影和清楚信息层级，" +
			"人物与知识对象简洁、平面、可读。" +
			"禁止三维建模、真实材质、体积光、塑料高光、复杂景深、" +
			"拟真渲染和商业三维动画电影质感。"

	case CWComicVisualChineseInk:
		return "严格二维现代国风教育插画。使用水墨晕染、工笔线描、宣纸留白和克制设色，" +
			"结合清晰现代教学构图；人物五官与服饰保持东方绘画语言。" +
			"禁止三维建模、塑料材质、西式动画电影角色比例、体积渲染、" +
			"摄影棚灯光和商业三维动画电影质感。"

	case CWComicVisualCinematic3D:
		return "电影级三维动画风格。允许三维角色、真实空间层次、材质、自然光影和景深，" +
			"但人物比例保持教学场景可信，避免过度幼态、夸张大头大眼、塑料玩偶感和品牌化角色造型。"

	case CWComicVisualRealisticIllustration:
		return "严格二维写实教学插画。采用博物馆教育插画、历史地理复原图或写实概念插画语言，" +
			"人物比例、建筑、器物和社会细节可信；允许绘画式明暗与细腻笔触。" +
			"禁止三维渲染、塑料材质、卡通大头比例、动画电影式大眼、" +
			"游戏CG界面感和商业三维动画电影质感。"

	default:
		return ""
	}
}
