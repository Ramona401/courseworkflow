package services

// courseware_comic_plan_parse.go — 漫画AI规划严格解析与自动排版
//
// 本文件负责：
//   - 严格解析模型唯一JSON对象，拒绝未知字段和尾随内容；
//   - 校验人物、4至8格分镜、角色引用和跨格连续关系；
//   - 根据项目ID和格号确定性生成稳定图片键；
//   - 根据结构化规划确定性构建项目风格IAOCI和每格图片IAOCI；
//   - 自动生成气泡、旁白、知识卡、题目卡的初始位置、字号和尾巴；
//   - 保留OriginalContent与OriginalQuestion，支持恢复AI初稿；
//   - 不信任模型输出的坐标、HTML或第三方编辑器状态。
//
// 自动排版使用归一化坐标，第一版确保不重叠且可直接使用。
// 后续教师可在前端通过宽松许可证编辑组件调整位置和样式。

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// coursewareComicPlanAIResponse 是模型唯一允许返回的顶层结构。
type coursewareComicPlanAIResponse struct {
	ArtStyleText    string                       `json:"art_style_text"`
	ContinuityNotes []string                     `json:"continuity_notes"`
	Characters      []coursewareComicAICharacter `json:"characters"`
	Panels          []coursewareComicAIPanel     `json:"panels"`
}

type coursewareComicAICharacter struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Role             string   `json:"role"`
	SubjectType      string   `json:"subject_type"`
	Appearance       string   `json:"appearance"`
	DefaultPosition  string   `json:"default_position"`
	FixedFeatures    []string `json:"fixed_features"`
	ForbiddenChanges []string `json:"forbidden_changes"`
}

type coursewareComicAIPanel struct {
	PanelNo               int                                       `json:"panel_no"`
	StoryPurpose          string                                    `json:"story_purpose"`
	KnowledgeClaim        string                                    `json:"knowledge_claim"`
	SceneText             string                                    `json:"scene_text"`
	Characters            []string                                  `json:"characters"`
	CharacterPositions    []coursewareComicAIPanelCharacterPosition `json:"character_positions"`
	ActionText            string                                    `json:"action_text"`
	CameraText            string                                    `json:"camera_text"`
	NarrationText         string                                    `json:"narration_text"`
	KnowledgePresentation string                                    `json:"knowledge_presentation"`
	FocusText             string                                    `json:"focus_text"`
	LayoutText            string                                    `json:"layout_text"`
	VisualPrompt          string                                    `json:"visual_prompt"`
	NegativePrompt        string                                    `json:"negative_prompt"`
	Relations             []coursewareComicAIRelation               `json:"relations"`
	OverlayElements       []coursewareComicAIOverlayElement         `json:"overlay_elements"`
}

type coursewareComicAIPanelCharacterPosition struct {
	CharacterID string `json:"character_id"`
	Position    string `json:"position"`
}

type coursewareComicAIRelation struct {
	TargetPanelNo int    `json:"target_panel_no"`
	RelationCode  string `json:"relation_code"`
	InheritMask   string `json:"inherit_mask"`
	SemanticNote  string `json:"semantic_note"`
}

type coursewareComicAIOverlayElement struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Content           string `json:"content"`
	SpeakerID         string `json:"speaker_id"`
	TargetCharacterID string `json:"target_character_id"`
	TargetAnchor      string `json:"target_anchor"`
	StyleID           string `json:"style_id"`
	PreferredRegion   string `json:"preferred_region"`
	Priority          int    `json:"priority"`

	Question *models.CoursewareComicQuestionContent `json:"question"`
}

// coursewareComicParsedPlan 是严格解析后的内部结果。
type coursewareComicParsedPlan struct {
	StyleAOCIText        string
	CharacterBibleJSON   string
	ContinuityLedgerJSON string
	Panels               []*models.CoursewareComicPanel
}

// parseCoursewareComicPlanAIResult 严格解析并确定性生成完整规划。
func parseCoursewareComicPlanAIResult(
	raw string,
	project *models.CoursewareComicProject,
) (*coursewareComicParsedPlan, error) {
	if project == nil ||
		project.PanelCount < 4 ||
		project.PanelCount > 8 {
		return nil,
			coursewareComicPlanOutputError(
				"漫画项目上下文无效",
				nil,
			)
	}

	jsonText, err :=
		extractCoursewareComicPlanJSON(raw)
	if err != nil {
		return nil, err
	}

	var response coursewareComicPlanAIResponse

	decoder := json.NewDecoder(
		strings.NewReader(jsonText),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&response,
	); err != nil {
		return nil,
			coursewareComicPlanOutputError(
				"JSON结构解析失败",
				err,
			)
	}

	var trailing interface{}
	if err := decoder.Decode(
		&trailing,
	); err != io.EOF {
		if err == nil {
			return nil,
				coursewareComicPlanOutputError(
					"JSON对象后存在额外数据",
					nil,
				)
		}

		return nil,
			coursewareComicPlanOutputError(
				"JSON尾部数据无效",
				err,
			)
	}

	artStyleText := strings.TrimSpace(
		response.ArtStyleText,
	)
	if artStyleText == "" {
		return nil,
			coursewareComicPlanOutputError(
				"art_style_text不能为空",
				nil,
			)
	}

	if len(response.Characters) < 1 ||
		len(response.Characters) > 6 {
		return nil,
			coursewareComicPlanOutputError(
				"characters数量必须为1至6",
				nil,
			)
	}

	characterBible, characterMap, err :=
		buildCoursewareComicCharacterBible(
			response.Characters,
		)
	if err != nil {
		return nil, err
	}

	styleAOCIText, err :=
		buildCoursewareComicStyleAOCI(
			artStyleText,
			characterBible,
		)
	if err != nil {
		return nil, err
	}

	if len(response.Panels) !=
		project.PanelCount {
		return nil,
			coursewareComicPlanOutputError(
				"panels数量与项目panel_count不一致",
				nil,
			)
	}

	sortedPanels := append(
		[]coursewareComicAIPanel{},
		response.Panels...,
	)

	sort.Slice(
		sortedPanels,
		func(left int, right int) bool {
			return sortedPanels[left].PanelNo <
				sortedPanels[right].PanelNo
		},
	)

	imageKeyByPanel := make(map[int]string)

	for panelNo := 1; panelNo <= project.PanelCount; panelNo++ {
		imageKey, keyErr :=
			utils.BuildImageAOCIKey(
				project.ID,
				fmt.Sprintf(
					"COMIC_PANEL_%02d",
					panelNo,
				),
			)
		if keyErr != nil {
			return nil,
				coursewareComicPlanOutputError(
					"生成漫画稳定图片键失败",
					keyErr,
				)
		}

		imageKeyByPanel[panelNo] =
			imageKey
	}

	panels := make(
		[]*models.CoursewareComicPanel,
		0,
		project.PanelCount,
	)

	for index, sourcePanel := range sortedPanels {
		expectedPanelNo := index + 1

		if sourcePanel.PanelNo !=
			expectedPanelNo {
			return nil,
				coursewareComicPlanOutputError(
					"漫画格必须从1开始连续编号",
					nil,
				)
		}

		panel, buildErr :=
			buildCoursewareComicPanel(
				project,
				sourcePanel,
				artStyleText,
				characterMap,
				imageKeyByPanel,
			)
		if buildErr != nil {
			return nil, buildErr
		}

		panels = append(panels, panel)
	}

	characterBibleJSON, err :=
		json.Marshal(characterBible)
	if err != nil {
		return nil,
			coursewareComicPlanOutputError(
				"序列化人物设定失败",
				err,
			)
	}

	continuityNotes :=
		normalizeCoursewareComicStrings(
			response.ContinuityNotes,
			12,
			500,
		)
	if len(continuityNotes) == 0 {
		return nil,
			coursewareComicPlanOutputError(
				"continuity_notes不能为空",
				nil,
			)
	}

	continuityLedger := map[string]interface{}{
		"version": 1,
		"notes":   continuityNotes,
		"character_ids": sortedCoursewareComicCharacterIDs(
			characterMap,
		),
		"panel_count": project.PanelCount,
	}

	continuityLedgerJSON, err :=
		json.Marshal(continuityLedger)
	if err != nil {
		return nil,
			coursewareComicPlanOutputError(
				"序列化连续性账本失败",
				err,
			)
	}

	return &coursewareComicParsedPlan{
		StyleAOCIText: styleAOCIText,
		CharacterBibleJSON: string(
			characterBibleJSON,
		),
		ContinuityLedgerJSON: string(
			continuityLedgerJSON,
		),
		Panels: panels,
	}, nil
}

func buildCoursewareComicCharacterBible(
	sources []coursewareComicAICharacter,
) (
	*models.CoursewareComicCharacterBible,
	map[string]models.CoursewareComicCharacter,
	error,
) {
	characters := make(
		[]models.CoursewareComicCharacter,
		0,
		len(sources),
	)
	characterMap := make(
		map[string]models.CoursewareComicCharacter,
		len(sources),
	)

	for _, source := range sources {
		character := models.CoursewareComicCharacter{
			ID: strings.TrimSpace(
				source.ID,
			),
			Name: strings.TrimSpace(
				source.Name,
			),
			Role: strings.TrimSpace(
				source.Role,
			),
			SubjectType: strings.TrimSpace(
				source.SubjectType,
			),
			Appearance: strings.TrimSpace(
				source.Appearance,
			),
			DefaultPosition: strings.TrimSpace(
				source.DefaultPosition,
			),
			FixedFeatures: normalizeCoursewareComicStrings(
				source.FixedFeatures,
				8,
				300,
			),
			ForbiddenChanges: normalizeCoursewareComicStrings(
				source.ForbiddenChanges,
				8,
				300,
			),
		}

		if !strings.HasPrefix(
			character.ID,
			"CHAR-",
		) ||
			len(character.ID) > 64 {
			return nil, nil,
				coursewareComicPlanOutputError(
					"人物ID必须使用CHAR-开头",
					nil,
				)
		}

		if _, duplicated :=
			characterMap[character.ID]; duplicated {
			return nil, nil,
				coursewareComicPlanOutputError(
					"人物ID重复",
					nil,
				)
		}

		if character.Name == "" ||
			character.Role == "" ||
			character.Appearance == "" {
			return nil, nil,
				coursewareComicPlanOutputError(
					"人物名称、职责和外观不能为空",
					nil,
				)
		}

		if !models.IsValidCWComicCharacterSubjectType(
			character.SubjectType,
		) {
			return nil, nil,
				coursewareComicPlanOutputError(
					"人物subject_type不合法",
					nil,
				)
		}

		if !models.IsValidCWComicCharacterAnchor(
			character.DefaultPosition,
		) {
			return nil, nil,
				coursewareComicPlanOutputError(
					"人物default_position不合法",
					nil,
				)
		}

		if len(character.FixedFeatures) == 0 ||
			len(character.ForbiddenChanges) == 0 {
			return nil, nil,
				coursewareComicPlanOutputError(
					"人物固定特征和禁止变化规则不能为空",
					nil,
				)
		}

		characterMap[character.ID] =
			character
		characters = append(
			characters,
			character,
		)
	}

	return &models.CoursewareComicCharacterBible{
		Version:    1,
		Characters: characters,
	}, characterMap, nil
}

func buildCoursewareComicStyleAOCI(
	artStyleText string,
	bible *models.CoursewareComicCharacterBible,
) (string, error) {
	characterText :=
		buildCoursewareComicCharacterSummary(
			bible.Characters,
		)

	subjectType :=
		resolveCoursewareComicImageSubjectType(
			bible.Characters,
		)

	styleAOCI := &models.ImageAOCI{
		ImageKey:        "@ANCHOR",
		IndexVersion:    1,
		IndexType:       models.CWImageIndexTypeAnchor,
		UsageRole:       models.CWImageUsageBackground,
		ContinuityLevel: 0,
		SubjectType:     subjectType,
		AspectRatio:     models.CWImageAspectFlexible,
		FocusText:       "知识点漫画项目统一视觉风格与固定主体身份",
		LayoutText:      "Ø",
		ArtText:         artStyleText,
		CharacterText:   characterText,
		SceneText:       "Ø",
		ExportText:      "项目级风格锚点，不锁定具体镜头、构图和场景",
		NegativeText: "禁止画面内可读文字、Logo、水印、伪字符；" +
			"禁止人物身份、服装颜色和固定特征跨格漂移；" +
			"禁止从锚点强行继承具体背景和镜头",
		Relations: []models.CoursewareImageRelationSpec{},
	}

	formatted, err :=
		utils.FormatImageAOCI(styleAOCI)
	if err != nil {
		return "",
			coursewareComicPlanOutputError(
				"构建漫画项目风格IAOCI失败",
				err,
			)
	}

	return formatted, nil
}

func buildCoursewareComicPanel(
	project *models.CoursewareComicProject,
	source coursewareComicAIPanel,
	artStyleText string,
	characterMap map[string]models.CoursewareComicCharacter,
	imageKeyByPanel map[int]string,
) (*models.CoursewareComicPanel, error) {
	source.StoryPurpose =
		strings.TrimSpace(source.StoryPurpose)
	source.KnowledgeClaim =
		strings.TrimSpace(source.KnowledgeClaim)
	source.SceneText =
		strings.TrimSpace(source.SceneText)
	source.ActionText =
		strings.TrimSpace(source.ActionText)
	source.CameraText =
		strings.TrimSpace(source.CameraText)
	source.NarrationText =
		strings.TrimSpace(source.NarrationText)
	source.KnowledgePresentation =
		strings.TrimSpace(
			source.KnowledgePresentation,
		)
	source.FocusText =
		strings.TrimSpace(source.FocusText)
	source.LayoutText =
		strings.TrimSpace(source.LayoutText)
	source.VisualPrompt =
		strings.TrimSpace(source.VisualPrompt)
	source.NegativePrompt =
		strings.TrimSpace(source.NegativePrompt)

	if source.StoryPurpose == "" ||
		source.KnowledgeClaim == "" ||
		source.SceneText == "" ||
		source.ActionText == "" ||
		source.CameraText == "" ||
		source.KnowledgePresentation == "" ||
		source.VisualPrompt == "" {
		return nil,
			coursewareComicPlanOutputError(
				fmt.Sprintf(
					"漫画第%d格核心字段不能为空",
					source.PanelNo,
				),
				nil,
			)
	}

	if source.FocusText == "" {
		source.FocusText =
			source.KnowledgeClaim
	}

	characterIDs :=
		normalizeCoursewareComicStrings(
			source.Characters,
			6,
			64,
		)
	if len(characterIDs) == 0 {
		return nil,
			coursewareComicPlanOutputError(
				fmt.Sprintf(
					"漫画第%d格必须出现至少一个角色",
					source.PanelNo,
				),
				nil,
			)
	}

	panelCharacters,
		panelCharacterMap,
		panelCharacterSet,
		panelPositionText,
		err :=
		buildCoursewareComicPanelCharacterContext(
			source.PanelNo,
			characterIDs,
			source.CharacterPositions,
			characterMap,
		)
	if err != nil {
		return nil, err
	}

	relationSpecs, relationJSON, err :=
		buildCoursewareComicPanelRelations(
			source.PanelNo,
			source.Relations,
			imageKeyByPanel,
			len(panelCharacters) > 0,
		)
	if err != nil {
		return nil, err
	}

	overlayDocument,
		dialogues,
		reservedRegions,
		err :=
		buildCoursewareComicOverlayDocument(
			source.PanelNo,
			source.OverlayElements,
			panelCharacterSet,
			panelCharacterMap,
		)
	if err != nil {
		return nil, err
	}

	overlayJSON, err :=
		json.Marshal(overlayDocument)
	if err != nil {
		return nil,
			coursewareComicPlanOutputError(
				"序列化漫画覆盖层失败",
				err,
			)
	}

	dialoguesJSON, err :=
		json.Marshal(dialogues)
	if err != nil {
		return nil,
			coursewareComicPlanOutputError(
				"序列化漫画对白失败",
				err,
			)
	}

	characterIDsJSON, err :=
		json.Marshal(characterIDs)
	if err != nil {
		return nil,
			coursewareComicPlanOutputError(
				"序列化漫画人物引用失败",
				err,
			)
	}

	layoutText := source.LayoutText
	if layoutText == "" {
		layoutText =
			source.CameraText
	}

	layoutText +=
		"；本格人物确定位置：" +
			panelPositionText +
			"；这些位置是底图构图与气泡目标共同使用的最终坐标事实"

	if len(reservedRegions) > 0 {
		layoutText += "；为后期HTML/SVG文字覆盖层预留区域：" +
			strings.Join(
				reservedRegions,
				"、",
			)
	}

	characterText :=
		buildCoursewareComicCharacterSummary(
			panelCharacters,
		) +
			"；本格人物位置：" +
			panelPositionText

	negativeText :=
		source.NegativePrompt
	if negativeText != "" {
		negativeText += "；"
	}
	negativeText +=
		"禁止任何可读文字、公式、标签、Logo、水印和伪字符；" +
			"禁止人物外貌、主体颜色、服装和固定特征漂移；" +
			"禁止人物左右互换、离开本格结构化位置或遮挡预留文字区域"

	imageAOCI := &models.ImageAOCI{
		ImageKey:        imageKeyByPanel[source.PanelNo],
		IndexVersion:    1,
		IndexType:       models.CWImageIndexTypeImage,
		UsageRole:       models.CWImageUsageStory,
		ContinuityLevel: 3,
		SubjectType: resolveCoursewareComicImageSubjectType(
			panelCharacters,
		),
		AspectRatio:   models.CWImageAspectHorizontal,
		FocusText:     source.FocusText,
		LayoutText:    layoutText,
		ArtText:       artStyleText,
		CharacterText: characterText,
		SceneText:     source.SceneText,
		ExportText: "横向16:9教学漫画画面；" +
			"图片内不生成文字；" +
			"对话、旁白、知识卡和题目由后期覆盖层渲染",
		NegativeText: negativeText,
		Relations:    relationSpecs,
	}

	aociText, err :=
		utils.FormatImageAOCI(imageAOCI)
	if err != nil {
		return nil,
			coursewareComicPlanOutputError(
				fmt.Sprintf(
					"构建漫画第%d格IAOCI失败",
					source.PanelNo,
				),
				err,
			)
	}

	visualPrompt := source.VisualPrompt +
		"\n\n【本格结构化人物位置】" +
		panelPositionText +
		"。这些位置优先于项目人物default_position和自然语言中的模糊描述，" +
		"必须严格按本格执行，不得左右互换、移动到其他区域或被文字留白挤走。" +
		"\n\n画面要求：" +
		source.CameraText +
		"。" +
		layoutText +
		"。" +
		"画面中不得出现任何文字、字幕、标签、公式、Logo、水印或伪字符。" +
		"人物、关键道具和知识对象不得进入已预留的文字覆盖区域。"

	return &models.CoursewareComicPanel{
		ProjectID:             project.ID,
		PanelNo:               source.PanelNo,
		ImageKey:              imageKeyByPanel[source.PanelNo],
		StoryPurpose:          source.StoryPurpose,
		KnowledgeClaim:        source.KnowledgeClaim,
		SceneText:             source.SceneText,
		CharacterIDsJSON:      string(characterIDsJSON),
		ActionText:            source.ActionText,
		CameraText:            source.CameraText,
		NarrationText:         source.NarrationText,
		DialoguesJSON:         string(dialoguesJSON),
		KnowledgePresentation: source.KnowledgePresentation,
		VisualPrompt:          visualPrompt,
		NegativePrompt:        negativeText,
		AOCIText:              aociText,
		RelationsJSON:         relationJSON,
		OverlayDocumentJSON:   string(overlayJSON),
		OverlayVersion:        1,
		Status:                models.CWComicPanelStatusPlanned,
		Version:               1,
		LastError:             "",
	}, nil
}

func extractCoursewareComicPlanJSON(
	raw string,
) (string, error) {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return "",
			coursewareComicPlanOutputError(
				"AI返回内容为空",
				nil,
			)
	}

	if extracted, ok :=
		ai.ExtractJSON(cleaned); ok &&
		strings.TrimSpace(
			extracted,
		) != "" {
		return strings.TrimSpace(
			extracted,
		), nil
	}

	if strings.HasPrefix(cleaned, "{") &&
		strings.HasSuffix(cleaned, "}") {
		return cleaned, nil
	}

	return "",
		coursewareComicPlanOutputError(
			"未找到完整合法的JSON对象",
			nil,
		)
}

func coursewareComicPlanOutputError(
	detail string,
	cause error,
) error {
	if cause == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrCoursewareComicPlanInvalidOutput,
			detail,
		)
	}

	return fmt.Errorf(
		"%w: %s: %v",
		ErrCoursewareComicPlanInvalidOutput,
		detail,
		cause,
	)
}
