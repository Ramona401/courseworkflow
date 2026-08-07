package services

// courseware_style_studio_chat.go — AI美术风格对话与参考图提取
//
// 本文件负责：
//   - 文字多轮对话更新课程锚点IAOCI；
//   - 上传参考图后的多模态风格提取；
//   - 在后端解析回复和完整IAOCI；
//   - 根据reference_mode强制清除或保留固定主体；
//   - 将老师消息、AI回复和IAOCI草稿原子保存；
//   - 风格变化后自动使旧预览失效。
//
// 保密规则：
//   - 完整IAOCI仅用于后端解析、校验、存储和图片生成；
//   - 返回给前端的内容只有自然语言回复和经过保密序列化的状态；
//   - AI格式错误时不得把原始模型输出或索引摘要写入HTTP错误正文。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

const (
	cwStyleStudioChatPromptKey = "prompt_courseware_style_studio_chat"

	cwStyleStudioImagePromptKey = "prompt_courseware_style_studio_image"

	cwStyleStudioSceneCode = "courseware_style_studio"
)

var styleStudioLog = logger.WithModule("courseware_style_studio")

// CoursewareStyleTurnResult 一轮风格共创结果。
type CoursewareStyleTurnResult struct {
	Reply string `json:"reply"`

	State *models.CoursewareStyleStudioState `json:"state"`
}

// SendTurn 发送一轮文字要求或参考图要求。
func (s *CoursewareStyleStudioService) SendTurn(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	request *models.CoursewareStyleTurnRequest,
	actor *CoursewareActorContext,
) (*CoursewareStyleTurnResult, error) {
	if request == nil {
		return nil, fmt.Errorf(
			"风格对话请求不能为空",
		)
	}

	courseware, scopedActor, err :=
		s.loadStyleStudioCourseware(
			ctx,
			coursewareID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	session, err :=
		repository.GetCoursewareStyleSessionByID(
			ctx,
			courseware.ID,
			strings.TrimSpace(sessionID),
			courseware.UserID,
		)
	if err != nil {
		return nil, err
	}

	if !models.IsEditableCWStyleSessionStatus(
		session.Status,
	) {
		return nil,
			repository.
				ErrCoursewareStyleSessionNotEditable
	}

	referenceMode :=
		strings.TrimSpace(
			request.ReferenceMode,
		)
	if referenceMode == "" {
		referenceMode =
			session.ReferenceMode
	}

	if !models.IsValidCWStyleReferenceMode(
		referenceMode,
	) {
		return nil, fmt.Errorf(
			"参考图模式不合法: %s",
			referenceMode,
		)
	}

	referenceAssetID :=
		normalizeStyleStudioStringPointer(
			request.ReferenceAssetID,
		)
	if referenceAssetID == nil {
		referenceAssetID =
			normalizeStyleStudioStringPointer(
				session.ReferenceAssetID,
			)
	}

	userContent :=
		strings.TrimSpace(
			request.Content,
		)

	if userContent == "" &&
		referenceAssetID == nil {
		return nil, fmt.Errorf(
			"请输入风格要求或上传参考图片",
		)
	}

	messages, err :=
		repository.ListCoursewareStyleMessages(
			ctx,
			courseware.ID,
			session.ID,
			courseware.UserID,
		)
	if err != nil {
		return nil, err
	}

	userInput :=
		buildCoursewareStyleStudioInput(
			courseware,
			session,
			messages,
			referenceMode,
			userContent,
		)

	systemPromptKey :=
		cwStyleStudioChatPromptKey

	var imageURL string

	if referenceAssetID != nil {
		asset, assetErr :=
			s.loadStyleStudioImageAsset(
				ctx,
				courseware.ID,
				*referenceAssetID,
			)
		if assetErr != nil {
			return nil, assetErr
		}

		imageURL =
			resolveAssetPublicURL(asset)
		if imageURL == "" {
			return nil, fmt.Errorf(
				"参考图片无法转换为公网地址",
			)
		}

		systemPromptKey =
			cwStyleStudioImagePromptKey
	}

	systemPrompt, err :=
		repository.GetCurrentPromptByKey(
			systemPromptKey,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"加载风格工作室提示词失败(%s): %w",
			systemPromptKey,
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
		return nil, fmt.Errorf(
			"获取风格工作室AI配置失败: %w",
			err,
		)
	}

	traceUserID :=
		scopedActor.UserID

	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			traceUserID,
		)

	traceContext := &ai.TraceContext{
		SceneCode: cwStyleStudioSceneCode,
		UserID:    &traceUserID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	var (
		rawOutput  string
		modelUsed  string
		tokensUsed int
	)

	if imageURL != "" {
		result, callErr :=
			ai.CallAIMultimodal(
				aiConfig,
				systemPrompt.Content,
				userInput,
				imageURL,
				traceContext,
			)
		if callErr != nil {
			return nil, fmt.Errorf(
				"参考图风格提取失败: %w",
				callErr,
			)
		}
		if result == nil {
			return nil, fmt.Errorf(
				"参考图风格提取未返回结果",
			)
		}

		rawOutput = result.Content
		modelUsed = result.ModelUsed
		tokensUsed = result.TokensUsed
	} else {
		result, callErr :=
			ai.CallAI(
				aiConfig,
				systemPrompt.Content,
				userInput,
				traceContext,
			)
		if callErr != nil {
			return nil, fmt.Errorf(
				"AI美术风格对话失败: %w",
				callErr,
			)
		}
		if result == nil {
			return nil, fmt.Errorf(
				"AI美术风格对话未返回结果",
			)
		}

		rawOutput = result.Content
		modelUsed = result.ModelUsed
		tokensUsed = result.TokensUsed
	}

	reply, styleAOCIText, parsedAOCI, err :=
		parseStyleStudioAIOutput(
			rawOutput,
			referenceMode,
		)
	if err != nil {
		// 原始模型输出可能包含完整内部索引，
		// 只在服务端日志记录错误类型和输出长度，不记录正文。
		styleStudioLog.Warn(
			"AI美术风格索引解析失败",
			"courseware_id", courseware.ID,
			"session_id", session.ID,
			"reference_mode", referenceMode,
			"output_length",
			len([]rune(rawOutput)),
			"error", err,
		)

		return nil, fmt.Errorf(
			"AI未能形成有效美术风格，请重新描述要求或更换参考图片",
		)
	}

	styleSummary :=
		buildStyleStudioSummary(
			parsedAOCI,
		)

	userMessage := &models.CoursewareStyleMessage{
		SessionID:        session.ID,
		CoursewareID:     courseware.ID,
		Role:             models.CWStyleMessageRoleUser,
		Content:          userContent,
		ReferenceAssetID: referenceAssetID,
	}

	assistantMessage :=
		&models.CoursewareStyleMessage{
			SessionID:    session.ID,
			CoursewareID: courseware.ID,
			Role: models.
				CWStyleMessageRoleAssistant,
			Content:       reply,
			StyleAOCIText: styleAOCIText,
		}

	updatedSession, err :=
		repository.AppendCoursewareStyleTurn(
			ctx,
			courseware.UserID,
			userMessage,
			assistantMessage,
			referenceMode,
			referenceAssetID,
			styleAOCIText,
			styleSummary,
		)
	if err != nil {
		return nil, err
	}

	state, err :=
		s.loadStyleStudioState(
			ctx,
			courseware,
			updatedSession,
		)
	if err != nil {
		return nil, err
	}

	styleStudioLog.Info(
		"AI美术风格对话完成",
		"courseware_id", courseware.ID,
		"session_id", session.ID,
		"reference_mode", referenceMode,
		"has_reference", imageURL != "",
		"subject_type",
		parsedAOCI.SubjectType,
		"model", modelUsed,
		"tokens", tokensUsed,
	)

	return &CoursewareStyleTurnResult{
		Reply: reply,
		State: state,
	}, nil
}

func buildCoursewareStyleStudioInput(
	courseware *models.Courseware,
	session *models.CoursewareStyleSession,
	messages []*models.CoursewareStyleMessage,
	referenceMode string,
	userContent string,
) string {
	var builder strings.Builder

	builder.WriteString("## 课程信息\n")
	builder.WriteString(
		fmt.Sprintf(
			"- 标题：%s\n- 学科：%s\n- 年级：%s\n",
			courseware.Title,
			courseware.Subject,
			courseware.Grade,
		),
	)

	builder.WriteString("\n## 本轮参考图模式\n")
	builder.WriteString(referenceMode)
	builder.WriteString("\n")

	if strings.TrimSpace(
		session.StyleAOCIText,
	) != "" {
		builder.WriteString(
			"\n## 当前课程锚点IAOCI草稿\n",
		)
		builder.WriteString(
			session.StyleAOCIText,
		)
		builder.WriteString("\n")
	} else {
		builder.WriteString(
			"\n## 当前课程锚点IAOCI草稿\n尚未形成，请从老师要求建立第一版。\n",
		)
	}

	recentMessages :=
		messages
	if len(recentMessages) > 10 {
		recentMessages =
			recentMessages[len(recentMessages)-10:]
	}

	if len(recentMessages) > 0 {
		builder.WriteString(
			"\n## 最近对话\n",
		)

		for _, message := range recentMessages {
			if message == nil {
				continue
			}

			roleName := "老师"
			if message.Role ==
				models.CWStyleMessageRoleAssistant {
				roleName = "AI"
			}

			builder.WriteString(
				fmt.Sprintf(
					"%s：%s\n",
					roleName,
					strings.TrimSpace(
						message.Content,
					),
				),
			)
		}
	}

	builder.WriteString(
		"\n## 老师本轮要求\n",
	)

	if strings.TrimSpace(
		userContent,
	) == "" {
		builder.WriteString(
			"老师本轮只上传了参考图片，请根据reference_mode提取或更新风格。\n",
		)
	} else {
		builder.WriteString(userContent)
		builder.WriteString("\n")
	}

	builder.WriteString(
		"\n请严格输出一行[REPLY]和完整九行课程锚点IAOCI，不得输出JSON。",
	)

	return builder.String()
}

func parseStyleStudioAIOutput(
	rawOutput string,
	referenceMode string,
) (
	string,
	string,
	*models.ImageAOCI,
	error,
) {
	rawOutput =
		strings.TrimSpace(rawOutput)
	if rawOutput == "" {
		return "", "", nil,
			fmt.Errorf("AI未返回风格内容")
	}

	reply := ""

	normalized :=
		strings.ReplaceAll(
			rawOutput,
			"\r\n",
			"\n",
		)
	normalized =
		strings.ReplaceAll(
			normalized,
			"\r",
			"\n",
		)

	for _, line := range strings.Split(
		normalized,
		"\n",
	) {
		line =
			strings.TrimSpace(line)

		if strings.HasPrefix(
			line,
			"[REPLY]",
		) {
			reply =
				strings.TrimSpace(
					strings.TrimPrefix(
						line,
						"[REPLY]",
					),
				)
			break
		}
	}

	cleaned :=
		utils.CleanImageAOCIOutput(
			rawOutput,
		)

	styleAOCIText, parsed, err :=
		normalizeStyleStudioAOCIForMode(
			cleaned,
			referenceMode,
		)
	if err != nil {
		return "", "", nil,
			fmt.Errorf(
				"AI返回的美术风格索引无效: %w",
				err,
			)
	}

	if reply == "" {
		reply =
			buildStyleStudioDefaultReply(
				referenceMode,
				parsed,
			)
	}

	reply =
		safeStyleStudioRunes(
			reply,
			120,
		)

	return reply,
		styleAOCIText,
		parsed,
		nil
}

// normalizeStyleStudioAOCIForMode 对AI输出执行确定性安全收敛。
func normalizeStyleStudioAOCIForMode(
	styleAOCIText string,
	referenceMode string,
) (
	string,
	*models.ImageAOCI,
	error,
) {
	if !models.IsValidCWStyleReferenceMode(
		referenceMode,
	) {
		return "", nil, fmt.Errorf(
			"参考图模式不合法: %s",
			referenceMode,
		)
	}

	parsed, err :=
		utils.ParseImageAOCI(
			strings.TrimSpace(
				styleAOCIText,
			),
		)
	if err != nil {
		return "", nil, err
	}

	purified, err :=
		utils.PurifyCoursewareAnchorAOCI(
			parsed,
		)
	if err != nil {
		return "", nil, err
	}

	// style_only和inspiration绝不保留参考图中的人物或物体身份。
	if referenceMode ==
		models.CWStyleReferenceModeStyleOnly ||
		referenceMode ==
			models.CWStyleReferenceModeInspiration {
		purified.CharacterText = "Ø"
		purified.SubjectType =
			models.CWImageSubjectNone
	}

	requiredNegative :=
		"禁止继承参考图中的文字、Logo、水印和机构标识"

	if referenceMode ==
		models.CWStyleReferenceModeInspiration {
		requiredNegative +=
			"；禁止精确复刻参考图角色、构图和品牌识别元素"
	}

	purified.NegativeText =
		appendStyleStudioSemantic(
			purified.NegativeText,
			requiredNegative,
		)

	formatted, err :=
		utils.FormatImageAOCI(
			purified,
		)
	if err != nil {
		return "", nil, err
	}

	finalParsed, err :=
		utils.ParseImageAOCI(
			formatted,
		)
	if err != nil {
		return "", nil, err
	}

	return formatted,
		finalParsed,
		nil
}

func buildStyleStudioSummary(
	aoci *models.ImageAOCI,
) string {
	if aoci == nil {
		return ""
	}

	summary :=
		"艺术风格：" +
			strings.TrimSpace(
				aoci.ArtText,
			)

	if !isStyleStudioEmptySemantic(
		aoci.CharacterText,
	) {
		summary +=
			"；固定主体：" +
				strings.TrimSpace(
					aoci.CharacterText,
				)
	} else {
		summary +=
			"；不固定人物或主体"
	}

	return safeStyleStudioRunes(
		summary,
		260,
	)
}

func buildStyleStudioDefaultReply(
	referenceMode string,
	aoci *models.ImageAOCI,
) string {
	switch referenceMode {
	case models.CWStyleReferenceModeCharacter:
		if aoci != nil &&
			!isStyleStudioEmptySemantic(
				aoci.CharacterText,
			) {
			return "已更新课程艺术风格，并保留您明确要求复用的固定主体；具体场景和构图不会被锁定。"
		}

		return "已提取课程艺术风格；参考图中没有适合全课程固定复用的主体，因此未锁定人物或物体。"

	case models.CWStyleReferenceModeInspiration:
		return "已把参考图抽象为通用视觉语言，不会复刻原图角色、环境、构图、文字或品牌元素。"

	default:
		return "已更新课程艺术风格，只保留绘制方式，不继承参考图中的人物、环境、构图、文字或标识。"
	}
}

func appendStyleStudioSemantic(
	current string,
	required string,
) string {
	current =
		strings.TrimSpace(current)
	required =
		strings.TrimSpace(required)

	if current == "" ||
		isStyleStudioEmptySemantic(
			current,
		) {
		return required
	}

	if required == "" ||
		strings.Contains(
			current,
			required,
		) {
		return current
	}

	return current + "；" + required
}

func isStyleStudioEmptySemantic(
	value string,
) bool {
	switch strings.ToLower(
		strings.TrimSpace(value),
	) {
	case "",
		"ø",
		"无",
		"none":
		return true
	default:
		return false
	}
}

func safeStyleStudioRunes(
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

	return string(runes[:maxLength])
}
