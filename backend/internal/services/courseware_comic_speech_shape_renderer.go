package services

// courseware_comic_speech_shape_renderer.go — 说话气泡整体SVG外轮廓
//
// 本文件把说话气泡主体和尾巴合成为一个闭合路径：
//   - 统一填充、透明度和描边；
//   - 尾巴根部替换主体边缘的一小段，不产生内部接缝；
//   - 圆角按1920×1080稳定设计像素计算，横向或纵向缩放时不再使用百分比椭圆角；
//   - 描边线宽读取教师可调的text_style.outline_width；
//   - 自动尾巴显示长度限制为30至58设计像素；
//   - 框内语义目标从最近外边缘向外显示36像素短尾巴；
//   - 与前端编辑画布和普通预览使用相同几何口径。
//
// SVG viewBox采用气泡自身的设计像素宽高，不再固定为100×100。
// 在标准覆盖层画布中，拖动改变长宽只延长直线边，四角圆弧半径保持一致。

import (
	"fmt"
	"math"
	"strings"

	"tedna/internal/models"
)

type coursewareComicRenderTailEdge string

const (
	coursewareComicRenderTailEdgeTop    coursewareComicRenderTailEdge = "top"
	coursewareComicRenderTailEdgeRight  coursewareComicRenderTailEdge = "right"
	coursewareComicRenderTailEdgeBottom coursewareComicRenderTailEdge = "bottom"
	coursewareComicRenderTailEdgeLeft   coursewareComicRenderTailEdge = "left"

	coursewareComicRenderSpeechBodyInset = 1.0

	coursewareComicRenderAutoTailMinimumPixels = 30.0
	coursewareComicRenderAutoTailMaximumPixels = 58.0
	coursewareComicRenderInsideTailPixels      = 36.0
)

func resolveCoursewareComicRenderTailEdge(
	origin coursewareComicRenderPoint,
) coursewareComicRenderTailEdge {
	switch {
	case origin.Y == 0:
		return coursewareComicRenderTailEdgeTop
	case origin.X == 1:
		return coursewareComicRenderTailEdgeRight
	case origin.Y == 1:
		return coursewareComicRenderTailEdgeBottom
	default:
		return coursewareComicRenderTailEdgeLeft
	}
}

func resolveCoursewareComicRenderSpeechDesignSize(
	element models.CoursewareComicOverlayElement,
) (float64, float64) {
	return math.Max(
			32,
			element.Width*coursewareComicRenderCanvasWidth,
		),
		math.Max(
			24,
			element.Height*coursewareComicRenderCanvasHeight,
		)
}

// resolveCoursewareComicRenderSpeechRadius 返回设计像素中的正圆圆角半径。
//
// 同一个半径同时用于横向和纵向圆弧，避免100×100百分比路径在
// 长方形气泡中被拉成椭圆。胶囊样式仍保持更大的圆角，但保留足够
// 直线边供尾巴自然接入。
func resolveCoursewareComicRenderSpeechRadius(
	styleID string,
	width float64,
	height float64,
) float64 {
	radius := 30.0

	switch strings.ToLower(
		strings.TrimSpace(styleID),
	) {
	case "speech_capsule":
		radius = 42
	case "speech_cloud":
		radius = 38
	case "speech_soft":
		radius = 34
	case "speech_outline":
		radius = 26
	case "speech_pop":
		radius = 28
	}

	maximum := math.Max(
		6,
		math.Min(width, height)/2-4,
	)

	return math.Max(
		6,
		math.Min(radius, maximum),
	)
}

// resolveCoursewareComicRenderSpeechStrokeWidth 统一读取整体描边线宽。
//
// 历史文档没有outline_width时按1px显示；教师保存后的合法范围为
// 0.5至3px。vector-effect保证气泡被缩放时线宽仍保持CSS像素稳定。
func resolveCoursewareComicRenderSpeechStrokeWidth(
	element models.CoursewareComicOverlayElement,
) float64 {
	value := element.TextStyle.OutlineWidth

	if math.IsNaN(value) ||
		math.IsInf(value, 0) ||
		value == 0 {
		return 1
	}

	return math.Max(
		0.5,
		math.Min(3, value),
	)
}

func resolveCoursewareComicRenderTailHalfBase(
	width float64,
	height float64,
) float64 {
	shortSide := math.Min(width, height)

	return math.Max(
		5,
		math.Min(
			9,
			shortSide*0.028,
		),
	)
}

func resolveCoursewareComicRenderTailBase(
	origin coursewareComicRenderPoint,
	edge coursewareComicRenderTailEdge,
	halfBase float64,
	radius float64,
	width float64,
	height float64,
) (coursewareComicRenderPoint, coursewareComicRenderPoint) {
	left := coursewareComicRenderSpeechBodyInset
	top := coursewareComicRenderSpeechBodyInset
	right := width - coursewareComicRenderSpeechBodyInset
	bottom := height - coursewareComicRenderSpeechBodyInset

	if edge == coursewareComicRenderTailEdgeTop ||
		edge == coursewareComicRenderTailEdgeBottom {
		centerX := math.Max(
			left+radius+halfBase,
			math.Min(
				right-radius-halfBase,
				origin.X*width,
			),
		)
		y := bottom
		if edge == coursewareComicRenderTailEdgeTop {
			y = top
		}

		first := coursewareComicRenderPoint{
			X: centerX - halfBase,
			Y: y,
		}
		second := coursewareComicRenderPoint{
			X: centerX + halfBase,
			Y: y,
		}

		if edge == coursewareComicRenderTailEdgeTop {
			return first, second
		}
		return second, first
	}

	centerY := math.Max(
		top+radius+halfBase,
		math.Min(
			bottom-radius-halfBase,
			origin.Y*height,
		),
	)
	x := left
	if edge == coursewareComicRenderTailEdgeRight {
		x = right
	}

	first := coursewareComicRenderPoint{
		X: x,
		Y: centerY - halfBase,
	}
	second := coursewareComicRenderPoint{
		X: x,
		Y: centerY + halfBase,
	}

	if edge == coursewareComicRenderTailEdgeRight {
		return first, second
	}
	return second, first
}

func resolveCoursewareComicRenderLocalTarget(
	element models.CoursewareComicOverlayElement,
	target coursewareComicRenderPoint,
	width float64,
	height float64,
) coursewareComicRenderPoint {
	localX :=
		(target.X - element.X) *
			coursewareComicRenderCanvasWidth

	localY :=
		(target.Y - element.Y) *
			coursewareComicRenderCanvasHeight

	return coursewareComicRenderPoint{
		X: math.Max(
			-width*2,
			math.Min(
				width*3,
				localX,
			),
		),
		Y: math.Max(
			-height*2,
			math.Min(
				height*3,
				localY,
			),
		),
	}
}

// coursewareComicRenderPointFromOrigin
// 从整格画布中的连接点沿指定方向生成一个设计像素长度的可见尖端。
func coursewareComicRenderPointFromOrigin(
	origin coursewareComicRenderPoint,
	direction coursewareComicRenderTailDirection,
	lengthPixels float64,
) coursewareComicRenderPoint {
	direction =
		normalizeCoursewareComicRenderTailDirection(
			direction,
		)

	return coursewareComicRenderPoint{
		X: clampCoursewareComicUnit(
			origin.X +
				direction.X*
					lengthPixels/
					coursewareComicRenderCanvasWidth,
		),
		Y: clampCoursewareComicUnit(
			origin.Y +
				direction.Y*
					lengthPixels/
					coursewareComicRenderCanvasHeight,
		),
	}
}

// resolveCoursewareComicRenderVisibleTailTarget
// 只计算课件页面实际绘制的尾巴尖端，不修改稳定文档中的人物语义目标。
//
// 自动模式把尾巴限制为30至58像素；manual框外目标保持教师设置；
// auto框内目标从目标最近外边缘生长；manual框内目标沿教师连接边向外；
// 两种模式都显示36像素短尾巴。
func resolveCoursewareComicRenderVisibleTailTarget(
	element models.CoursewareComicOverlayElement,
	origin coursewareComicRenderPoint,
	semanticTarget coursewareComicRenderPoint,
) coursewareComicRenderPoint {
	globalOrigin :=
		coursewareComicRenderTailOriginGlobal(
			element,
			origin,
		)

	if coursewareComicRenderTailTargetInsideBody(
		element,
		semanticTarget,
	) {
		return coursewareComicRenderPointFromOrigin(
			globalOrigin,
			coursewareComicRenderOutwardDirectionForOrigin(
				origin,
			),
			coursewareComicRenderInsideTailPixels,
		)
	}

	if strings.EqualFold(
		strings.TrimSpace(element.Tail.Type),
		"manual",
	) {
		return semanticTarget
	}

	delta := coursewareComicRenderTailDirection{
		X: (semanticTarget.X - globalOrigin.X) *
			coursewareComicRenderCanvasWidth,
		Y: (semanticTarget.Y - globalOrigin.Y) *
			coursewareComicRenderCanvasHeight,
	}
	distance := math.Hypot(delta.X, delta.Y)

	if distance < 1 {
		delta =
			resolveCoursewareComicRenderTailDirection(
				element,
				semanticTarget,
			)
	}

	visibleLength := math.Max(
		coursewareComicRenderAutoTailMinimumPixels,
		math.Min(
			coursewareComicRenderAutoTailMaximumPixels,
			distance,
		),
	)

	return coursewareComicRenderPointFromOrigin(
		globalOrigin,
		delta,
		visibleLength,
	)
}

func buildCoursewareComicRenderTailSegment(
	entry coursewareComicRenderPoint,
	exit coursewareComicRenderPoint,
	target coursewareComicRenderPoint,
) string {
	center := coursewareComicRenderPoint{
		X: (entry.X + exit.X) / 2,
		Y: (entry.Y + exit.Y) / 2,
	}
	deltaX := target.X - center.X
	deltaY := target.Y - center.Y
	distance := math.Max(
		1,
		math.Hypot(
			deltaX,
			deltaY,
		),
	)
	directionX := deltaX / distance
	directionY := deltaY / distance
	tangentX := exit.X - entry.X
	tangentY := exit.Y - entry.Y
	tangentLength := math.Max(
		0.001,
		math.Hypot(
			tangentX,
			tangentY,
		),
	)
	tangentX /= tangentLength
	tangentY /= tangentLength

	baseControl := math.Min(
		distance*0.30,
		70,
	)
	tipControl := math.Min(
		distance*0.13,
		24,
	)
	tipHalf := 0.8

	firstControl := coursewareComicRenderPoint{
		X: entry.X +
			directionX*
				baseControl,
		Y: entry.Y +
			directionY*
				baseControl,
	}
	secondControl := coursewareComicRenderPoint{
		X: exit.X +
			directionX*
				baseControl,
		Y: exit.Y +
			directionY*
				baseControl,
	}
	firstTipControl := coursewareComicRenderPoint{
		X: target.X -
			directionX*
				tipControl -
			tangentX*
				tipHalf,
		Y: target.Y -
			directionY*
				tipControl -
			tangentY*
				tipHalf,
	}
	secondTipControl := coursewareComicRenderPoint{
		X: target.X -
			directionX*
				tipControl +
			tangentX*
				tipHalf,
		Y: target.Y -
			directionY*
				tipControl +
			tangentY*
				tipHalf,
	}

	return "C " +
		formatCoursewareComicRenderPoint(
			firstControl,
		) +
		" " +
		formatCoursewareComicRenderPoint(
			firstTipControl,
		) +
		" " +
		formatCoursewareComicRenderPoint(
			target,
		) +
		" C " +
		formatCoursewareComicRenderPoint(
			secondTipControl,
		) +
		" " +
		formatCoursewareComicRenderPoint(
			secondControl,
		) +
		" " +
		formatCoursewareComicRenderPoint(
			exit,
		)
}

func buildCoursewareComicRenderUnifiedSpeechPath(
	edge coursewareComicRenderTailEdge,
	entry coursewareComicRenderPoint,
	exit coursewareComicRenderPoint,
	target coursewareComicRenderPoint,
	radius float64,
	width float64,
	height float64,
) string {
	left := coursewareComicRenderSpeechBodyInset
	top := coursewareComicRenderSpeechBodyInset
	right := width - coursewareComicRenderSpeechBodyInset
	bottom := height - coursewareComicRenderSpeechBodyInset
	tail := buildCoursewareComicRenderTailSegment(
		entry,
		exit,
		target,
	)

	switch edge {
	case coursewareComicRenderTailEdgeTop:
		return fmt.Sprintf(
			"M %.3f,%.3f H %.3f %s H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f Q %.3f,%.3f %.3f,%.3f H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f Q %.3f,%.3f %.3f,%.3f Z",
			left+radius,
			top,
			entry.X,
			tail,
			right-radius,
			right,
			top,
			right,
			top+radius,
			bottom-radius,
			right,
			bottom,
			right-radius,
			bottom,
			left+radius,
			left,
			bottom,
			left,
			bottom-radius,
			top+radius,
			left,
			top,
			left+radius,
			top,
		)

	case coursewareComicRenderTailEdgeRight:
		return fmt.Sprintf(
			"M %.3f,%.3f H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f %s V %.3f Q %.3f,%.3f %.3f,%.3f H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f Q %.3f,%.3f %.3f,%.3f Z",
			left+radius,
			top,
			right-radius,
			right,
			top,
			right,
			top+radius,
			entry.Y,
			tail,
			bottom-radius,
			right,
			bottom,
			right-radius,
			bottom,
			left+radius,
			left,
			bottom,
			left,
			bottom-radius,
			top+radius,
			left,
			top,
			left+radius,
			top,
		)

	case coursewareComicRenderTailEdgeBottom:
		return fmt.Sprintf(
			"M %.3f,%.3f H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f Q %.3f,%.3f %.3f,%.3f H %.3f %s H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f Q %.3f,%.3f %.3f,%.3f Z",
			left+radius,
			top,
			right-radius,
			right,
			top,
			right,
			top+radius,
			bottom-radius,
			right,
			bottom,
			right-radius,
			bottom,
			entry.X,
			tail,
			left+radius,
			left,
			bottom,
			left,
			bottom-radius,
			top+radius,
			left,
			top,
			left+radius,
			top,
		)

	default:
		return fmt.Sprintf(
			"M %.3f,%.3f H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f Q %.3f,%.3f %.3f,%.3f H %.3f Q %.3f,%.3f %.3f,%.3f V %.3f %s V %.3f Q %.3f,%.3f %.3f,%.3f Z",
			left+radius,
			top,
			right-radius,
			right,
			top,
			right,
			top+radius,
			bottom-radius,
			right,
			bottom,
			right-radius,
			bottom,
			left+radius,
			left,
			bottom,
			left,
			bottom-radius,
			entry.Y,
			tail,
			top+radius,
			left,
			top,
			left+radius,
			top,
		)
	}
}

func renderCoursewareComicSpeechBubbleShape(
	element models.CoursewareComicOverlayElement,
	palette coursewareComicRenderPalette,
) string {
	width, height :=
		resolveCoursewareComicRenderSpeechDesignSize(
			element,
		)

	origin, target :=
		resolveCoursewareComicRenderTailGeometryPoints(
			element,
		)

	edge :=
		resolveCoursewareComicRenderTailEdge(
			origin,
		)

	radius :=
		resolveCoursewareComicRenderSpeechRadius(
			element.StyleID,
			width,
			height,
		)

	halfBase :=
		resolveCoursewareComicRenderTailHalfBase(
			width,
			height,
		)

	entry, exit :=
		resolveCoursewareComicRenderTailBase(
			origin,
			edge,
			halfBase,
			radius,
			width,
			height,
		)

	visibleTarget :=
		resolveCoursewareComicRenderVisibleTailTarget(
			element,
			origin,
			target,
		)

	localTarget :=
		resolveCoursewareComicRenderLocalTarget(
			element,
			visibleTarget,
			width,
			height,
		)

	path :=
		buildCoursewareComicRenderUnifiedSpeechPath(
			edge,
			entry,
			exit,
			localTarget,
			radius,
			width,
			height,
		)

	return fmt.Sprintf(
		`<svg class="tedna-comic-bubble-shape" viewBox="0 0 %.3f %.3f" preserveAspectRatio="none" style="overflow:visible" aria-hidden="true"><path d="%s" fill="%s" stroke="%s" stroke-width="%.2f" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></path></svg>`,
		width,
		height,
		path,
		palette.Fill,
		palette.Stroke,
		resolveCoursewareComicRenderSpeechStrokeWidth(
			element,
		),
	)
}
