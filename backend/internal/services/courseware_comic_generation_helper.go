package services

// courseware_comic_generation_helper.go — 漫画图片生产纯辅助
//
// 本文件集中保存：
//   - 后台任务类型；
//   - 图片尺寸与文件上限；
//   - 人物设定图提示词；
//   - 单格图片提示词；
//   - 图片生成专用Unicode安全截断；
//   - 后台任务登记结果映射；
//   - 项目失败状态收敛；
//   - 课件SSE漫画事件广播。
//
// 图片生成侧使用truncateCoursewareComicGenerationRunes。
// 页面渲染侧使用truncateCoursewareComicRunes。
// 两者职责分离，避免同一个services包内出现同名函数。

import (
	"context"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	backgroundTaskTypeCoursewareComicGenerate = "courseware_comic_generate"

	backgroundTaskTypeCoursewareComicPanelRegenerate = "courseware_comic_panel_regenerate"

	coursewareComicGenerationEvent = "comic_generation"

	coursewareComicImageSize = "2560x1440"

	coursewareComicMaxGeneratedFileSize = 20 * 1024 * 1024

	// coursewareComicPanelRegenerationInstructionMaxRunes
	// 限制教师单格画面微调要求的最大Unicode字符数，
	// 防止超长请求挤占内部人物、知识事实与IAOCI约束。
	coursewareComicPanelRegenerationInstructionMaxRunes = 1200
)

// buildCoursewareComicCharacterSheetPrompt 构建无文字人物设定图内容事实。
//
// 艺术风格由第三步visual_style_source严格决定，
// 此处不再注入AI规划阶段的旧StyleAOCIText，避免旧画风污染。
func buildCoursewareComicCharacterSheetPrompt(
	project *models.CoursewareComicProject,
) string {
	if project == nil {
		return ""
	}

	return strings.Join(
		[]string{
			"【人物设定图内容任务】",
			"生成知识点漫画项目的人物设定参考图。",
			"横向16:9，中性浅色背景，" +
				"把全部角色清晰分开排列，" +
				"展示全身或接近全身的标准造型。",
			"每个角色保持自然站姿和清楚正面特征，" +
				"人物之间不得遮挡。",
			"只绘制人物、动物或拟人知识对象，" +
				"不得生成角色姓名、编号、标签、对白、" +
				"公式、Logo、水印或任何可读文字。",
			"输入参考图只用于继承艺术媒介、色彩、线条、材质和光影语言，" +
				"不得继承参考图的场景、动作、构图、主体位置、" +
				"文字、Logo或机构标识。",
			"【人物身份与固定特征】",
			truncateCoursewareComicGenerationRunes(
				project.CharacterBibleJSON,
				16000,
			),
		},
		"\n",
	)
}

// buildCoursewareComicPanelGenerationPrompt 构建无文字单格内容事实。
//
// 项目旧StyleAOCIText不再进入实际图片提示词。
// VisualPrompt与AOCIText只作为人物、场景、动作、镜头、知识对象和连续性事实，
// 其中任何艺术风格、渲染媒介、材质或品牌化描述都必须忽略。
func buildCoursewareComicPanelGenerationPrompt(
	project *models.CoursewareComicProject,
	panel *models.CoursewareComicPanel,
) string {
	if project == nil ||
		panel == nil {
		return ""
	}

	return strings.Join(
		[]string{
			"【本格内容事实】",
			truncateCoursewareComicGenerationRunes(
				panel.VisualPrompt,
				12000,
			),
			"上段只用于提取人物、场景、动作、镜头和知识对象；" +
				"如包含任何画风、媒介、材质、灯光渲染或品牌化描述，一律忽略。",
			"【跨格人物身份与固定特征】",
			truncateCoursewareComicGenerationRunes(
				project.CharacterBibleJSON,
				16000,
			),
			"人物设定中的default_position只用于项目级默认和人物设定图；" +
				"本格VisualPrompt与AOCI中明确写出的结构化人物位置优先级更高，" +
				"不得因为项目默认位置而把本格人物移到其他区域。",
			"【本格构图与连续性事实】",
			truncateCoursewareComicGenerationRunes(
				panel.AOCIText,
				12000,
			),
			"上段只用于提取构图关系、主体位置、人物连续性和知识事实；" +
				"如包含任何旧画风、三维、材质或渲染描述，一律忽略。" +
				"其中“本格人物确定位置”是最终构图约束，必须与气泡目标使用同一九宫格位置，" +
				"不得左右互换、自动居中或移动到其他区域。",
			"【本格禁止项】",
			truncateCoursewareComicGenerationRunes(
				panel.NegativePrompt,
				6000,
			),
			"【硬性输出约束】",
			"画面只包含人物、情节、场景、道具和知识对象；" +
				"不得生成任何文字、题目、公式、字幕、标签、" +
				"Logo、水印或伪字符；" +
				"不得把气泡、旁白框或题目卡直接画进图片。",
		},
		"\n",
	)
}

// applyCoursewareComicPanelRegenerationInstruction
// 把可选的教师单格画面微调要求追加到已确认视觉渲染计划。
//
// 教师主动单格重画入口已经强制要求非空；
// 整批生成内部对旧图片分格的生成或恢复可以传入空要求，
// 此时返回基础渲染计划的安全副本。
//
// 非空要求只参与本次实际图片Prompt：
//   - 不覆盖数据库中的visual_prompt、IAOCI或人物设定；
//   - 不改变知识事实、人物身份、跨格连续性、画风、画幅与清晰度；
//   - 不进入计费幂等键，恢复同一已领取版本时始终复用原计费资产。
func applyCoursewareComicPanelRegenerationInstruction(
	plan *coursewareComicStylePreviewRenderPlan,
	instruction string,
) (*coursewareComicStylePreviewRenderPlan, bool) {
	if plan == nil {
		return nil, false
	}

	instruction =
		strings.TrimSpace(
			instruction,
		)

	if instruction == "" {
		cloned :=
			*plan

		return &cloned,
			strings.TrimSpace(
				cloned.Prompt,
			) != ""
	}

	if len(
		[]rune(
			instruction,
		),
	) >
		coursewareComicPanelRegenerationInstructionMaxRunes {
		return nil, false
	}

	cloned :=
		*plan

	cloned.Prompt =
		strings.TrimSpace(
			strings.Join(
				[]string{
					cloned.Prompt,
					"【教师本次画面微调要求】",
					instruction,
					"【本次微调边界】",
					"只调整本格画面的构图、镜头、人物动作与表情、场景、道具、光线、色彩和视觉细节；" +
						"不得改变知识事实、人物身份、人物固定特征、跨格连续性、教师已确认画风、画幅或清晰度；" +
						"不得生成文字、公式、字幕、气泡、题目框、Logo、水印或伪字符。" +
						"若教师微调要求与上述硬约束冲突，必须以硬约束为准。",
				},
				"\n",
			),
		)

	if cloned.Prompt == "" {
		return nil, false
	}

	return &cloned, true
}

// truncateCoursewareComicGenerationRunes 按Unicode字符安全截断图片提示词。
func truncateCoursewareComicGenerationRunes(
	value string,
	maxLength int,
) string {
	value =
		strings.TrimSpace(value)

	if maxLength <= 0 {
		return ""
	}

	runes := []rune(value)

	if len(runes) <= maxLength {
		return value
	}

	return string(
		runes[:maxLength],
	)
}

// mapCoursewareComicTaskStartResult 映射后台任务登记结果。
func mapCoursewareComicTaskStartResult(
	result BackgroundStartResult,
) error {
	switch result {
	case BackgroundStarted:
		return nil

	case BackgroundAlreadyRunning:
		return repository.
			ErrCoursewareComicProjectConflict

	case BackgroundRejectedDraining:
		return ErrCoursewareComicProjectServiceUnavailable

	default:
		return ErrCoursewareComicProjectInvalidRequest
	}
}

// markProjectGenerationFailed 把仍为generating的项目收敛为failed。
//
// 单格失败事务可能已经把项目改成failed，
// 因此本操作是best-effort，不覆盖原始错误。
func (s *CoursewareComicGenerationService) markProjectGenerationFailed(
	coursewareID string,
	projectID string,
	userID string,
	cause error,
) {
	message :=
		"知识点漫画图片生成失败"

	if cause != nil &&
		strings.TrimSpace(
			cause.Error(),
		) != "" {
		message =
			strings.TrimSpace(
				cause.Error(),
			)
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
	defer cancel()

	_, _ =
		repository.TransitionCoursewareComicProjectStatus(
			ctx,
			coursewareID,
			projectID,
			userID,
			[]string{
				models.CWComicProjectStatusGenerating,
			},
			models.CWComicProjectStatusFailed,
			message,
		)
}

// broadcastGeneration 广播知识点漫画图片生产事件。
func (s *CoursewareComicGenerationService) broadcastGeneration(
	coursewareID string,
	stage string,
	data map[string]interface{},
) {
	if data == nil {
		data =
			map[string]interface{}{}
	}

	data["stage"] = stage

	GlobalCWSSEHub.Broadcast(
		coursewareID,
		CWSSEEvent{
			EventType: coursewareComicGenerationEvent,
			Data:      data,
		},
	)
}
