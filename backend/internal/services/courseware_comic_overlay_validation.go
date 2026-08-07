package services

// courseware_comic_overlay_validation.go — 漫画覆盖层稳定协议规范化与校验
//
// 本文件负责：
//   - 保留服务端已经固化的AI原始文字和题目；
//   - 兼容历史缺失的颜色模式、行距、透明度和自动尾巴字段；
//   - 校验坐标、尺寸、字号、字重、行距、对齐、颜色、透明度和描边线宽；
//   - 校验气泡尾巴auto/manual稳定协议；
//   - 自动模式保存前修复过短或落入文本框内部的尾巴；
//   - 派生可编辑对白列表。
//
// 所有规范化均发生在服务端接收覆盖层文档之后、持久化之前。
// manual尾巴的连接点和指向点不会被自动算法改写。

import (
	"encoding/json"
	"math"
	"regexp"
	"strings"

	"tedna/internal/models"
)

var coursewareComicOverlayHexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// preserveCoursewareComicOriginalOverlay 保留服务端已经固化的AI初稿。
//
// 客户端可以修改Content和Question，但不能覆盖OriginalContent和
// OriginalQuestion。新增元素以首次保存内容作为自己的原始值。
func preserveCoursewareComicOriginalOverlay(
	storedJSON string,
	document *models.CoursewareComicOverlayDocument,
) error {
	if document == nil {
		return ErrCoursewareComicOverlayInvalid
	}

	var stored models.CoursewareComicOverlayDocument

	if err := json.Unmarshal(
		[]byte(storedJSON),
		&stored,
	); err != nil {
		return ErrCoursewareComicOverlayInvalid
	}

	storedByID := make(
		map[string]models.CoursewareComicOverlayElement,
		len(stored.Elements),
	)

	for _, element := range stored.Elements {
		storedByID[element.ID] = element
	}

	for index := range document.Elements {
		element := &document.Elements[index]

		storedElement, exists := storedByID[element.ID]
		if exists {
			element.OriginalContent = storedElement.OriginalContent
			if element.OriginalContent == "" {
				element.OriginalContent = storedElement.Content
			}

			element.OriginalQuestion =
				copyCoursewareComicQuestion(
					storedElement.OriginalQuestion,
				)
			if element.OriginalQuestion == nil {
				element.OriginalQuestion =
					copyCoursewareComicQuestion(
						storedElement.Question,
					)
			}
			continue
		}

		element.OriginalContent = element.Content
		element.OriginalQuestion =
			copyCoursewareComicQuestion(
				element.Question,
			)
	}

	return nil
}

func isValidCoursewareComicFiniteNumber(
	value float64,
) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0)
}

// isValidCoursewareComicUnitCoordinate 校验0至1归一化坐标。
func isValidCoursewareComicUnitCoordinate(
	value float64,
) bool {
	return isValidCoursewareComicFiniteNumber(value) &&
		value >= 0 &&
		value <= 1
}

func normalizeCoursewareComicTextStyle(
	elementType string,
	style *models.CoursewareComicTextStyle,
) bool {
	if style == nil ||
		!isValidCoursewareComicFiniteNumber(
			style.FontSize,
		) ||
		style.FontSize < 12 ||
		style.FontSize > 120 {
		return false
	}

	style.FontFamily =
		strings.TrimSpace(
			style.FontFamily,
		)
	if style.FontFamily == "" {
		style.FontFamily =
			"Noto Sans SC, sans-serif"
	}

	if style.FontWeight == 0 {
		style.FontWeight = 600
	}
	if style.FontWeight < 300 ||
		style.FontWeight > 900 {
		return false
	}

	if style.LineHeight == 0 {
		style.LineHeight = 1.35
	}
	if !isValidCoursewareComicFiniteNumber(
		style.LineHeight,
	) ||
		style.LineHeight < 1 ||
		style.LineHeight > 2.2 {
		return false
	}

	style.Align =
		strings.ToLower(
			strings.TrimSpace(
				style.Align,
			),
		)
	if style.Align == "" {
		switch elementType {
		case models.CWComicElementNarration,
			models.CWComicElementKnowledgeCard,
			models.CWComicElementWarningCard,
			models.CWComicElementQuestionCard,
			models.CWComicElementAnswerCard,
			models.CWComicElementCaption:
			style.Align = "left"

		default:
			style.Align = "center"
		}
	}

	switch style.Align {
	case "left", "center", "right", "justify":
	default:
		return false
	}

	style.ColorMode =
		strings.ToLower(
			strings.TrimSpace(
				style.ColorMode,
			),
		)
	if style.ColorMode == "" {
		style.ColorMode =
			models.CWComicTextColorModeAuto
	}
	if !models.IsValidCWComicTextColorMode(
		style.ColorMode,
	) {
		return false
	}

	style.Color =
		strings.ToUpper(
			strings.TrimSpace(
				style.Color,
			),
		)
	if style.ColorMode ==
		models.CWComicTextColorModeManual {
		if !coursewareComicOverlayHexColorPattern.
			MatchString(
				style.Color,
			) {
			return false
		}
	} else if !coursewareComicOverlayHexColorPattern.
		MatchString(
			style.Color,
		) {
		style.Color = "#111827"
	}

	if !isValidCoursewareComicFiniteNumber(
		style.BackgroundOpacity,
	) {
		return false
	}
	if style.BackgroundOpacity == 0 {
		style.BackgroundOpacity = 1
	}
	if style.BackgroundOpacity < 0.2 ||
		style.BackgroundOpacity > 1 {
		return false
	}

	if !isValidCoursewareComicFiniteNumber(
		style.OutlineWidth,
	) {
		return false
	}
	if style.OutlineWidth == 0 {
		style.OutlineWidth = 1
	}
	if style.OutlineWidth < 0.5 ||
		style.OutlineWidth > 3 {
		return false
	}

	return true
}

// validateCoursewareComicBubbleTail 校验并规范气泡尾巴协议。
//
// 历史空值和single统一收敛成auto；manual必须同时保存连接点。
// OriginX与OriginY任意模式都必须成对出现，避免保存半份几何数据。
func validateCoursewareComicBubbleTail(
	tail *models.CoursewareComicBubbleTail,
) bool {
	if tail == nil {
		return false
	}

	tail.Type =
		strings.ToLower(
			strings.TrimSpace(
				tail.Type,
			),
		)

	switch tail.Type {
	case "", "single":
		tail.Type = "auto"

	case "auto", "manual":

	default:
		return false
	}

	if !isValidCoursewareComicUnitCoordinate(
		tail.TargetX,
	) ||
		!isValidCoursewareComicUnitCoordinate(
			tail.TargetY,
		) {
		return false
	}

	hasOriginX := tail.OriginX != nil
	hasOriginY := tail.OriginY != nil

	if hasOriginX != hasOriginY {
		return false
	}
	if tail.Type == "manual" &&
		!hasOriginX {
		return false
	}
	if hasOriginX &&
		(!isValidCoursewareComicUnitCoordinate(
			*tail.OriginX,
		) ||
			!isValidCoursewareComicUnitCoordinate(
				*tail.OriginY,
			)) {
		return false
	}

	return true
}

// normalizeCoursewareComicAutomaticSpeechTail 把自动尾巴持久化为安全几何。
//
// 渲染器和编辑器已经使用相同的80像素最短、130像素默认长度算法。
// 这里在保存前复用后端确定性几何，使刷新、普通预览和插入课件
// 读取到的JSON本身也不会继续保留重叠目标点。
func normalizeCoursewareComicAutomaticSpeechTail(
	element *models.CoursewareComicOverlayElement,
) {
	if element == nil ||
		element.Type !=
			models.CWComicElementSpeechBubble ||
		element.Tail == nil ||
		element.Tail.Type != "auto" {
		return
	}

	origin, target :=
		resolveCoursewareComicRenderTailGeometryPoints(
			*element,
		)

	originX := origin.X
	originY := origin.Y

	element.Tail.TargetX = target.X
	element.Tail.TargetY = target.Y
	element.Tail.OriginX = &originX
	element.Tail.OriginY = &originY
}

func validateCoursewareComicQuestionContent(
	question *models.CoursewareComicQuestionContent,
) bool {
	if question == nil {
		return false
	}

	question.Question =
		strings.TrimSpace(
			question.Question,
		)
	question.Explanation =
		strings.TrimSpace(
			question.Explanation,
		)
	question.AnswerMode =
		strings.TrimSpace(
			question.AnswerMode,
		)

	if question.Question == "" ||
		len(question.Options) < 2 ||
		len(question.Options) > 6 ||
		question.AnswerIndex < 0 ||
		question.AnswerIndex >=
			len(question.Options) ||
		!models.IsValidCWComicAnswerMode(
			question.AnswerMode,
		) {
		return false
	}

	for index := range question.Options {
		question.Options[index] =
			strings.TrimSpace(
				question.Options[index],
			)
		if question.Options[index] == "" {
			return false
		}
	}

	return true
}

func validateCoursewareComicOverlayDocument(
	document *models.CoursewareComicOverlayDocument,
) error {
	if document == nil ||
		document.Version < 1 ||
		document.Canvas.Width <= 0 ||
		document.Canvas.Height <= 0 ||
		len(document.Elements) < 1 ||
		len(document.Elements) > 8 {
		return ErrCoursewareComicOverlayInvalid
	}

	seen := make(map[string]bool)

	for index := range document.Elements {
		element := &document.Elements[index]

		element.ID =
			strings.TrimSpace(
				element.ID,
			)
		element.Type =
			strings.TrimSpace(
				element.Type,
			)
		element.Content =
			strings.TrimSpace(
				element.Content,
			)
		element.StyleID =
			strings.TrimSpace(
				element.StyleID,
			)
		element.SpeakerID =
			strings.TrimSpace(
				element.SpeakerID,
			)
		element.TargetCharacterID =
			strings.TrimSpace(
				element.TargetCharacterID,
			)
		element.TargetAnchor =
			strings.TrimSpace(
				element.TargetAnchor,
			)
		element.AutoLayoutRegion =
			strings.TrimSpace(
				element.AutoLayoutRegion,
			)

		if element.ID == "" ||
			seen[element.ID] ||
			!models.IsValidCWComicElementType(
				element.Type,
			) ||
			element.StyleID == "" ||
			!isValidCoursewareComicFiniteNumber(
				element.X,
			) ||
			!isValidCoursewareComicFiniteNumber(
				element.Y,
			) ||
			!isValidCoursewareComicFiniteNumber(
				element.Width,
			) ||
			!isValidCoursewareComicFiniteNumber(
				element.Height,
			) ||
			!isValidCoursewareComicFiniteNumber(
				element.Rotation,
			) ||
			element.Width <= 0 ||
			element.Height <= 0 ||
			element.X < 0 ||
			element.Y < 0 ||
			element.X+element.Width > 1.001 ||
			element.Y+element.Height > 1.001 ||
			!normalizeCoursewareComicTextStyle(
				element.Type,
				&element.TextStyle,
			) {
			return ErrCoursewareComicOverlayInvalid
		}

		seen[element.ID] = true

		if element.Type !=
			models.CWComicElementQuestionCard &&
			element.Content == "" {
			return ErrCoursewareComicOverlayInvalid
		}

		if element.AutoLayoutRegion != "" &&
			!models.IsValidCWComicLayoutRegion(
				element.AutoLayoutRegion,
			) {
			return ErrCoursewareComicOverlayInvalid
		}

		isBubble :=
			element.Type ==
				models.CWComicElementSpeechBubble ||
				element.Type ==
					models.CWComicElementThoughtBubble

		if isBubble {
			if element.SpeakerID == "" ||
				element.TargetCharacterID == "" ||
				!validateCoursewareComicBubbleTail(
					element.Tail,
				) {
				return ErrCoursewareComicOverlayInvalid
			}

			if element.TargetAnchor == "" {
				element.TargetAnchor =
					models.CWComicAnchorCenter
			}
			if !models.IsValidCWComicCharacterAnchor(
				element.TargetAnchor,
			) {
				return ErrCoursewareComicOverlayInvalid
			}

			normalizeCoursewareComicAutomaticSpeechTail(
				element,
			)
		}

		if element.Type ==
			models.CWComicElementQuestionCard &&
			!validateCoursewareComicQuestionContent(
				element.Question,
			) {
			return ErrCoursewareComicOverlayInvalid
		}
	}

	return nil
}

func deriveCoursewareComicDialogues(
	elements []models.CoursewareComicOverlayElement,
) []models.CoursewareComicDialogue {
	result := make(
		[]models.CoursewareComicDialogue,
		0,
		len(elements),
	)

	for _, element := range elements {
		if element.Type !=
			models.CWComicElementSpeechBubble &&
			element.Type !=
				models.CWComicElementThoughtBubble {
			continue
		}

		result = append(
			result,
			models.CoursewareComicDialogue{
				ID:          element.ID,
				CharacterID: element.SpeakerID,
				Content:     element.Content,
				BubbleStyle: element.StyleID,
			},
		)
	}

	return result
}
