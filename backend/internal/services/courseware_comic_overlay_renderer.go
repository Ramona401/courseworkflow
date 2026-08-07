package services

// courseware_comic_overlay_renderer.go — 漫画覆盖层确定性渲染
//
// 本文件负责气泡、尾巴、卡片、题目和文字的HTML/SVG输出。
// 说话气泡主体与尾巴使用同一个闭合SVG路径，统一填充、透明度和描边；
// 自动模式保留人物语义目标点，并为课件显示选择自然短尾巴和最近外边缘。

import (
	"fmt"
	htmlstd "html"
	"math"
	"strconv"
	"strings"

	"tedna/internal/models"
)

// coursewareComicRenderPalette 是气泡和教学卡片的安全配色。
type coursewareComicRenderPalette struct {
	Fill   string
	Stroke string
	Text   string
}

const (
	coursewareComicRenderCanvasWidth         = 1920.0
	coursewareComicRenderCanvasHeight        = 1080.0
	coursewareComicRenderTailOriginEdgeInset = 0.12
	coursewareComicRenderTailInsideEpsilon   = 0.0001
)

type coursewareComicRenderTailDirection struct {
	X float64
	Y float64
}

type coursewareComicRenderInsideTailPlacement struct {
	Origin    coursewareComicRenderPoint
	Direction coursewareComicRenderTailDirection
}

func renderCoursewareComicOverlayElement(
	projectID string,
	panelID string,
	element models.CoursewareComicOverlayElement,
) string {
	elementType := sanitizeCoursewareComicElementType(element.Type)
	palette := resolveCoursewareComicRenderPalette(element.StyleID, elementType)
	baseFill := palette.Fill

	palette.Fill = applyCoursewareComicBackgroundOpacity(
		palette.Fill,
		normalizeCoursewareComicBackgroundOpacity(element.TextStyle.BackgroundOpacity),
	)

	x := clampCoursewareComicUnit(element.X)
	y := clampCoursewareComicUnit(element.Y)
	width := clampCoursewareComicSize(element.Width)
	height := clampCoursewareComicSize(element.Height)

	fontSize := element.TextStyle.FontSize
	if fontSize < 12 {
		fontSize = 28
	}
	if fontSize > 120 {
		fontSize = 120
	}

	fontWeight := element.TextStyle.FontWeight
	if fontWeight < 300 || fontWeight > 900 {
		fontWeight = 600
	}

	lineHeight := element.TextStyle.LineHeight
	if lineHeight < 1 || lineHeight > 2.2 {
		lineHeight = 1.4
	}

	textColor := resolveCoursewareComicRenderTextColor(
		element.TextStyle,
		baseFill,
		palette.Text,
	)

	zIndex := element.ZIndex
	if zIndex < 1 {
		zIndex = 20
	}
	if zIndex > 999 {
		zIndex = 999
	}

	elementID := safeCoursewareComicDOMID(projectID + "-" + panelID + "-" + element.ID)
	var builder strings.Builder

	builder.WriteString(`<div class="tedna-comic-overlay tedna-comic-overlay--`)
	builder.WriteString(elementType)
	builder.WriteString(`" id="`)
	builder.WriteString(htmlstd.EscapeString(elementID))
	builder.WriteString(`" style="left:`)
	builder.WriteString(formatCoursewareComicPercent(x))
	builder.WriteString(`;top:`)
	builder.WriteString(formatCoursewareComicPercent(y))
	builder.WriteString(`;width:`)
	builder.WriteString(formatCoursewareComicPercent(width))
	builder.WriteString(`;height:`)
	builder.WriteString(formatCoursewareComicPercent(height))
	builder.WriteString(`;min-height:0;transform:rotate(`)
	builder.WriteString(strconv.FormatFloat(element.Rotation, 'f', 2, 64))
	builder.WriteString(`deg);z-index:`)
	builder.WriteString(strconv.Itoa(zIndex))
	builder.WriteString(`;">`)

	if elementType == models.CWComicElementSpeechBubble &&
		element.Tail != nil {
		builder.WriteString(
			renderCoursewareComicSpeechBubbleShape(
				element,
				palette,
			),
		)
	} else {
		builder.WriteString(
			renderCoursewareComicBubbleShape(
				elementType,
				element.StyleID,
				palette,
			),
		)
	}
	builder.WriteString(`<div class="tedna-comic-overlay-text" style="font-size:`)
	builder.WriteString(strconv.FormatFloat(fontSize, 'f', 1, 64))
	builder.WriteString(`px;font-weight:`)
	builder.WriteString(strconv.Itoa(fontWeight))
	builder.WriteString(`;line-height:`)
	builder.WriteString(strconv.FormatFloat(lineHeight, 'f', 2, 64))
	builder.WriteString(`;text-align:`)
	builder.WriteString(sanitizeCoursewareComicTextAlign(element.TextStyle.Align))
	builder.WriteString(`;color:`)
	builder.WriteString(textColor)
	builder.WriteString(`;">`)

	if elementType == models.CWComicElementQuestionCard && element.Question != nil {
		builder.WriteString(renderCoursewareComicQuestion(elementID, element.Content, element.Question))
	} else {
		builder.WriteString(htmlstd.EscapeString(element.Content))
	}

	builder.WriteString(`</div></div>`)
	return builder.String()
}

// renderCoursewareComicBubbleShape 渲染气泡或卡片SVG轮廓。
func renderCoursewareComicBubbleShape(
	elementType string,
	styleID string,
	palette coursewareComicRenderPalette,
) string {
	styleID =
		strings.ToLower(
			strings.TrimSpace(
				styleID,
			),
		)

	strokeWidth := "2.2"
	shape := ""
	closingTag := "</rect>"

	switch {
	case elementType ==
		models.CWComicElementThoughtBubble:
		strokeWidth = "2.0"

		if styleID ==
			"thought_outline" {
			strokeWidth = "2.8"
		}

		shape =
			`<path d="M16 29 C10 17 25 7 38 14 C46 3 65 7 67 19 C83 14 94 26 87 39 C99 49 91 65 78 66 C78 82 61 91 50 80 C38 92 20 85 21 71 C6 69 2 51 13 43 C8 38 9 33 16 29 Z"`
		closingTag = "</path>"

	case styleID ==
		"speech_capsule":
		strokeWidth = "2.0"
		shape =
			`<rect x="1.5" y="1.5" width="97" height="97" rx="48"`

	case styleID ==
		"speech_pop":
		strokeWidth = "2.8"
		shape =
			`<path d="M11 4 C27 1 36 6 48 4 C66 1 86 3 94 14 C99 27 94 39 97 51 C101 66 94 88 80 94 C64 99 52 94 39 97 C23 101 6 93 3 78 C0 63 5 51 3 38 C1 23 2 9 11 4 Z"`
		closingTag = "</path>"

	default:
		radius := "18"

		if styleID ==
			"speech_soft" {
			radius = "23"
			strokeWidth = "1.5"
		}

		if styleID ==
			"speech_outline" {
			radius = "17"
			strokeWidth = "2.8"
		}

		if elementType ==
			models.CWComicElementNarration ||
			elementType ==
				models.CWComicElementCaption {
			radius = "6"
		}

		shape =
			`<rect x="1.5" y="1.5" width="97" height="97" rx="` +
				radius + `"`
	}

	return `<svg class="tedna-comic-bubble-shape" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">` +
		shape +
		` fill="` +
		palette.Fill +
		`" stroke="` +
		palette.Stroke +
		`" stroke-width="` +
		strokeWidth +
		`" stroke-linejoin="round">` +
		closingTag +
		`</svg>`
}

type coursewareComicRenderPoint struct {
	X float64
	Y float64
}

func formatCoursewareComicRenderPoint(
	point coursewareComicRenderPoint,
) string {
	return fmt.Sprintf(
		"%.3f,%.3f",
		point.X,
		point.Y,
	)
}

// resolveCoursewareComicRenderTailGeometryPoints 返回连接点和人物目标点。
//
// manual模式使用教师保存的连接点；auto模式只重新投影连接边。
// 两种模式都严格保留TargetX和TargetY，不再为了固定长度改写人物目标。
func resolveCoursewareComicRenderTailGeometryPoints(
	element models.CoursewareComicOverlayElement,
) (coursewareComicRenderPoint, coursewareComicRenderPoint) {
	target := coursewareComicRenderPoint{
		X: clampCoursewareComicUnit(element.Tail.TargetX),
		Y: clampCoursewareComicUnit(element.Tail.TargetY),
	}
	var origin coursewareComicRenderPoint

	if coursewareComicRenderTailTargetInsideBody(
		element,
		target,
	) {
		origin =
			coursewareComicRenderInsideTargetPlacement(
				element,
				target,
			).Origin
	} else {
		direction :=
			resolveCoursewareComicRenderTailDirection(
				element,
				target,
			)
		origin =
			resolveCoursewareComicRenderTailOriginForDirection(
				element,
				direction,
			)
	}

	if strings.EqualFold(
		strings.TrimSpace(element.Tail.Type),
		"manual",
	) &&
		element.Tail.OriginX != nil &&
		element.Tail.OriginY != nil {
		origin = coursewareComicRenderPoint{
			X: clampCoursewareComicUnit(
				*element.Tail.OriginX,
			),
			Y: clampCoursewareComicUnit(
				*element.Tail.OriginY,
			),
		}
	}

	return origin, target
}

func resolveCoursewareComicRenderTailDirection(
	element models.CoursewareComicOverlayElement,
	target coursewareComicRenderPoint,
) coursewareComicRenderTailDirection {
	if coursewareComicRenderTailTargetInsideBody(
		element,
		target,
	) {
		return normalizeCoursewareComicRenderTailDirection(
			coursewareComicRenderInsideTargetDirection(
				element,
				target,
			),
		)
	}

	centerX := element.X + element.Width/2
	centerY := element.Y + element.Height/2
	direction := coursewareComicRenderTailDirection{
		X: (target.X - centerX) * coursewareComicRenderCanvasWidth,
		Y: (target.Y - centerY) * coursewareComicRenderCanvasHeight,
	}
	if math.Hypot(direction.X, direction.Y) < 1 {
		direction =
			coursewareComicRenderFallbackTailDirection(
				element.TargetAnchor,
			)
	}
	return normalizeCoursewareComicRenderTailDirection(direction)
}

// coursewareComicRenderInsideTargetPlacement
// 在人物语义目标落入文本框内部时，选择最近外边缘，并把连接点对齐到
// 目标在该边上的投影。连接点不会再固定落在边缘中点。
func coursewareComicRenderInsideTargetPlacement(
	element models.CoursewareComicOverlayElement,
	target coursewareComicRenderPoint,
) coursewareComicRenderInsideTailPlacement {
	type candidate struct {
		Edge     coursewareComicRenderTailEdge
		Distance float64
	}

	candidates := []candidate{
		{
			Edge: coursewareComicRenderTailEdgeLeft,
			Distance: (target.X - element.X) *
				coursewareComicRenderCanvasWidth,
		},
		{
			Edge: coursewareComicRenderTailEdgeRight,
			Distance: (element.X + element.Width - target.X) *
				coursewareComicRenderCanvasWidth,
		},
		{
			Edge: coursewareComicRenderTailEdgeTop,
			Distance: (target.Y - element.Y) *
				coursewareComicRenderCanvasHeight,
		},
		{
			Edge: coursewareComicRenderTailEdgeBottom,
			Distance: (element.Y + element.Height - target.Y) *
				coursewareComicRenderCanvasHeight,
		},
	}

	selected := candidates[0]
	for _, current := range candidates[1:] {
		if current.Distance < selected.Distance {
			selected = current
		}
	}

	localX := math.Max(
		0,
		math.Min(
			1,
			(target.X-element.X)/
				math.Max(element.Width, 0.001),
		),
	)
	localY := math.Max(
		0,
		math.Min(
			1,
			(target.Y-element.Y)/
				math.Max(element.Height, 0.001),
		),
	)

	switch selected.Edge {
	case coursewareComicRenderTailEdgeLeft:
		return coursewareComicRenderInsideTailPlacement{
			Origin: coursewareComicRenderPoint{
				X: 0,
				Y: math.Max(
					coursewareComicRenderTailOriginEdgeInset,
					math.Min(
						1-coursewareComicRenderTailOriginEdgeInset,
						localY,
					),
				),
			},
			Direction: coursewareComicRenderTailDirection{X: -1, Y: 0},
		}

	case coursewareComicRenderTailEdgeRight:
		return coursewareComicRenderInsideTailPlacement{
			Origin: coursewareComicRenderPoint{
				X: 1,
				Y: math.Max(
					coursewareComicRenderTailOriginEdgeInset,
					math.Min(
						1-coursewareComicRenderTailOriginEdgeInset,
						localY,
					),
				),
			},
			Direction: coursewareComicRenderTailDirection{X: 1, Y: 0},
		}

	case coursewareComicRenderTailEdgeTop:
		return coursewareComicRenderInsideTailPlacement{
			Origin: coursewareComicRenderPoint{
				X: math.Max(
					coursewareComicRenderTailOriginEdgeInset,
					math.Min(
						1-coursewareComicRenderTailOriginEdgeInset,
						localX,
					),
				),
				Y: 0,
			},
			Direction: coursewareComicRenderTailDirection{X: 0, Y: -1},
		}

	default:
		return coursewareComicRenderInsideTailPlacement{
			Origin: coursewareComicRenderPoint{
				X: math.Max(
					coursewareComicRenderTailOriginEdgeInset,
					math.Min(
						1-coursewareComicRenderTailOriginEdgeInset,
						localX,
					),
				),
				Y: 1,
			},
			Direction: coursewareComicRenderTailDirection{X: 0, Y: 1},
		}
	}
}

func coursewareComicRenderInsideTargetDirection(
	element models.CoursewareComicOverlayElement,
	target coursewareComicRenderPoint,
) coursewareComicRenderTailDirection {
	return coursewareComicRenderInsideTargetPlacement(
		element,
		target,
	).Direction
}

// coursewareComicRenderOutwardDirectionForOrigin
// 框内目标必须沿实际连接边的外法线绘制，manual连接点也不会指回框内。
func coursewareComicRenderOutwardDirectionForOrigin(
	origin coursewareComicRenderPoint,
) coursewareComicRenderTailDirection {
	switch {
	case origin.Y == 0:
		return coursewareComicRenderTailDirection{X: 0, Y: -1}
	case origin.X == 1:
		return coursewareComicRenderTailDirection{X: 1, Y: 0}
	case origin.Y == 1:
		return coursewareComicRenderTailDirection{X: 0, Y: 1}
	default:
		return coursewareComicRenderTailDirection{X: -1, Y: 0}
	}
}

func coursewareComicRenderFallbackTailDirection(anchor string) coursewareComicRenderTailDirection {
	switch strings.TrimSpace(anchor) {
	case models.CWComicAnchorLeftTop:
		return coursewareComicRenderTailDirection{X: -1, Y: -1}
	case models.CWComicAnchorLeftCenter:
		return coursewareComicRenderTailDirection{X: -1, Y: 0}
	case models.CWComicAnchorLeftBottom:
		return coursewareComicRenderTailDirection{X: -1, Y: 1}
	case models.CWComicAnchorCenterTop:
		return coursewareComicRenderTailDirection{X: 0, Y: -1}
	case models.CWComicAnchorCenter:
		return coursewareComicRenderTailDirection{X: 0, Y: 1}
	case models.CWComicAnchorRightTop:
		return coursewareComicRenderTailDirection{X: 1, Y: -1}
	case models.CWComicAnchorRightCenter:
		return coursewareComicRenderTailDirection{X: 1, Y: 0}
	case models.CWComicAnchorRightBottom:
		return coursewareComicRenderTailDirection{X: 1, Y: 1}
	default:
		return coursewareComicRenderTailDirection{X: 0, Y: 1}
	}
}

func normalizeCoursewareComicRenderTailDirection(
	direction coursewareComicRenderTailDirection,
) coursewareComicRenderTailDirection {
	length := math.Hypot(direction.X, direction.Y)
	if length < 0.001 {
		return coursewareComicRenderTailDirection{X: 0, Y: 1}
	}
	return coursewareComicRenderTailDirection{
		X: direction.X / length,
		Y: direction.Y / length,
	}
}

func resolveCoursewareComicRenderTailOriginForDirection(
	element models.CoursewareComicOverlayElement,
	direction coursewareComicRenderTailDirection,
) coursewareComicRenderPoint {
	direction = normalizeCoursewareComicRenderTailDirection(direction)
	maxX := math.Inf(1)
	maxY := math.Inf(1)

	if math.Abs(direction.X) > 0.0001 {
		maxX = element.Width * coursewareComicRenderCanvasWidth / 2 / math.Abs(direction.X)
	}
	if math.Abs(direction.Y) > 0.0001 {
		maxY = element.Height * coursewareComicRenderCanvasHeight / 2 / math.Abs(direction.Y)
	}

	distance := math.Min(maxX, maxY)
	globalX := element.X + element.Width/2 +
		direction.X*distance/coursewareComicRenderCanvasWidth
	globalY := element.Y + element.Height/2 +
		direction.Y*distance/coursewareComicRenderCanvasHeight
	localX := (globalX - element.X) / math.Max(element.Width, 0.001)
	localY := (globalY - element.Y) / math.Max(element.Height, 0.001)

	if localX <= 0.001 || localX >= 0.999 {
		localX = math.Round(localX)
		localY = math.Max(
			coursewareComicRenderTailOriginEdgeInset,
			math.Min(1-coursewareComicRenderTailOriginEdgeInset, localY),
		)
	} else {
		localX = math.Max(
			coursewareComicRenderTailOriginEdgeInset,
			math.Min(1-coursewareComicRenderTailOriginEdgeInset, localX),
		)
		localY = math.Round(localY)
	}

	return coursewareComicRenderPoint{X: localX, Y: localY}
}

func coursewareComicRenderTailOriginGlobal(
	element models.CoursewareComicOverlayElement,
	origin coursewareComicRenderPoint,
) coursewareComicRenderPoint {
	return coursewareComicRenderPoint{
		X: element.X + element.Width*origin.X,
		Y: element.Y + element.Height*origin.Y,
	}
}

func coursewareComicRenderTailTargetInsideBody(
	element models.CoursewareComicOverlayElement,
	target coursewareComicRenderPoint,
) bool {
	return target.X >
		element.X+coursewareComicRenderTailInsideEpsilon &&
		target.X <
			element.X+element.Width-
				coursewareComicRenderTailInsideEpsilon &&
		target.Y >
			element.Y+coursewareComicRenderTailInsideEpsilon &&
		target.Y <
			element.Y+element.Height-
				coursewareComicRenderTailInsideEpsilon
}

// renderCoursewareComicQuestion 渲染题目、选项、答案和解析。
func renderCoursewareComicQuestion(
	elementID string,
	label string,
	question *models.CoursewareComicQuestionContent,
) string {
	if question == nil {
		return ""
	}

	answerID :=
		elementID + "-answer"

	var builder strings.Builder

	if strings.TrimSpace(label) != "" {
		builder.WriteString(
			`<div class="tedna-comic-question-label">`,
		)
		builder.WriteString(
			htmlstd.EscapeString(label),
		)
		builder.WriteString(`</div>`)
	}

	builder.WriteString(
		`<div class="tedna-comic-question-title">`,
	)
	builder.WriteString(
		htmlstd.EscapeString(
			question.Question,
		),
	)
	builder.WriteString(`</div>`)

	builder.WriteString(
		`<ol class="tedna-comic-options">`,
	)

	for _, option := range question.Options {
		builder.WriteString(`<li>`)
		builder.WriteString(
			htmlstd.EscapeString(option),
		)
		builder.WriteString(`</li>`)
	}

	builder.WriteString(`</ol>`)

	if question.AnswerMode ==
		models.CWComicAnswerModeClickReveal {
		builder.WriteString(
			`<button type="button" class="tedna-comic-answer-button" data-tedna-answer-target="`,
		)
		builder.WriteString(
			htmlstd.EscapeString(
				answerID,
			),
		)
		builder.WriteString(
			`">查看答案</button>`,
		)
	}

	builder.WriteString(
		`<div class="tedna-comic-answer" data-tedna-answer-id="`,
	)
	builder.WriteString(
		htmlstd.EscapeString(
			answerID,
		),
	)
	builder.WriteString(`"`)

	if question.AnswerMode ==
		models.CWComicAnswerModeClickReveal {
		builder.WriteString(` hidden`)
	}

	builder.WriteString(`>`)

	if question.AnswerIndex >= 0 &&
		question.AnswerIndex <
			len(question.Options) {
		answerText :=
			question.Options[question.AnswerIndex]

		builder.WriteString(
			`<strong>答案：</strong>`,
		)
		builder.WriteString(
			htmlstd.EscapeString(
				answerText,
			),
		)
		builder.WriteString(`<br>`)
	}

	builder.WriteString(
		`<strong>解析：</strong>`,
	)
	builder.WriteString(
		htmlstd.EscapeString(
			question.Explanation,
		),
	)
	builder.WriteString(`</div>`)

	return builder.String()
}

// replaceCoursewareComicPanelFragment 只替换一个稳定漫画格区间。
