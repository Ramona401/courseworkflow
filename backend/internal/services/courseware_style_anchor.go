package services

// courseware_style_anchor.go — 课件课程级图像IAOCI锚点提取服务
//
// 新职责：
//   - 多模态读取锚点图片；
//   - 使用独立prompt_courseware_image_anchor_iaoci提示词；
//   - 清理模型可能附加的说明或代码围栏；
//   - 严格解析IAOCI；
//   - 确定性提纯课程锚点；
//   - 返回规范化IAOCI全文。
//
// 课程锚点只锁定“如何画”和“固定主体是谁”：
//   - [A]艺术风格可供全课件使用；
//   - [C]仅在页面确实出现对应人物、动物或标志性主体时使用；
//   - [L]布局不继承；
//   - [S]环境不继承；
//   - [R]固定为0。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// 使用独立提示词键，避免新代码部署前影响旧运行进程。
const vaociExtractPromptKey = "prompt_courseware_image_anchor_iaoci"

const vaociExtractUserText = `请分析这张课件锚点图片，严格输出系统提示词规定的课程锚点IAOCI。

只提取可复用的艺术风格和固定人物、动物或标志性主体身份。
不得把图片里的教室、课桌、黑板、家具、背景、具体构图、镜头、景别、主体位置和道具位置作为课程级约束。
只输出IAOCI，不要输出JSON、Markdown代码围栏或解释。`

// ExtractVAOCIFromImageURL 保留原方法名和调用契约，返回值已经升级为规范IAOCI。
//
// SetStyleAnchor现有调用方不需要修改：
//   - 本方法返回规范IAOCI；
//   - repository.UpdateCoursewareStyleAnchor继续保存text；
//   - 数据库触发器自动同步@ANCHOR图片索引。
func (s *CoursewareAssetService) ExtractVAOCIFromImageURL(
	ctx context.Context,
	imageURL string,
	userID string,
) (string, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return "", fmt.Errorf(
			"锚点图URL为空，无法提取IAOCI",
		)
	}

	if !strings.HasPrefix(imageURL, "http://") &&
		!strings.HasPrefix(imageURL, "https://") {
		return "", fmt.Errorf(
			"锚点图URL必须是公网http(s)地址，当前为: %s",
			imageURL,
		)
	}

	systemPrompt, err :=
		repository.GetCurrentPromptByKey(
			vaociExtractPromptKey,
		)
	if err != nil {
		return "", fmt.Errorf(
			"加载课程锚点IAOCI提示词失败(%s): %w",
			vaociExtractPromptKey,
			err,
		)
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		sceneCWMediaPrompt,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf(
			"获取AI配置失败: %w",
			err,
		)
	}

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	traceContext := &ai.TraceContext{
		SceneCode: sceneCWMediaPrompt,
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, callErr := ai.CallAIMultimodal(
		aiConfig,
		systemPrompt.Content,
		vaociExtractUserText,
		imageURL,
		traceContext,
	)
	if callErr != nil {
		return "", fmt.Errorf(
			"多模态读图提取课程锚点IAOCI失败: %w",
			callErr,
		)
	}
	if result == nil {
		return "", fmt.Errorf(
			"多模态读图未返回结果",
		)
	}

	rawOutput := strings.TrimSpace(result.Content)
	if rawOutput == "" {
		return "", fmt.Errorf(
			"AI未返回课程锚点IAOCI",
		)
	}

	cleaned := utils.CleanImageAOCIOutput(
		rawOutput,
	)

	parsed, err := utils.ParseImageAOCI(cleaned)
	if err != nil {
		return "", fmt.Errorf(
			"AI返回的课程锚点IAOCI格式无效: %w；输出摘要=%s",
			err,
			truncateStr(cleaned, 240),
		)
	}

	purified, err :=
		utils.PurifyCoursewareAnchorAOCI(
			parsed,
		)
	if err != nil {
		return "", fmt.Errorf(
			"课程锚点IAOCI提纯失败: %w",
			err,
		)
	}

	formatted, err :=
		utils.FormatImageAOCI(purified)
	if err != nil {
		return "", fmt.Errorf(
			"课程锚点IAOCI规范化失败: %w",
			err,
		)
	}

	cwAssetLog.Info(
		"课程锚点IAOCI提取成功",
		"image_url", imageURL,
		"iaoci_len", len([]rune(formatted)),
		"subject_type", purified.SubjectType,
		"has_fixed_subject",
		!strings.HasPrefix(
			strings.TrimSpace(
				purified.CharacterText,
			),
			"Ø",
		),
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
	)

	return formatted, nil
}
