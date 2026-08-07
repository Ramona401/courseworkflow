package services

// courseware_comic_plan_layout.go — 漫画覆盖层初始布局、尾巴安全走廊与通用规划辅助
//
// 本文件负责：
//   - 漫画格跨页连续关系校验与序列化；
//   - 覆盖层元素的确定性区域选择和初始尺寸；
//   - 说话气泡尾巴的默认长度、边缘连接点和安全分离；
//   - 问题卡、文字样式及通用规划辅助函数；
//   - 新生成数据默认采用自动高对比文字与轻透明背景。
//
// 尾巴长度以1920×1080稳定覆盖层画布的像素距离计算。
// 自动模式始终保留人物目标点，通过选择邻近区域和移动气泡缩短尾巴；
// 教师后续切换为manual后不经过本文件重算。

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
)

const (
	coursewareComicPlanCanvasWidth         = 1920.0
	coursewareComicPlanCanvasHeight        = 1080.0
	coursewareComicPlanTailMinimumPixels   = 30.0
	coursewareComicPlanTailMaximumPixels   = 68.0
	coursewareComicPlanTailBoxGapPixels    = 10.0
	coursewareComicPlanTailOriginEdgeInset = 0.12
)

type coursewareComicLayoutBox struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

type coursewareComicPlanTailPoint struct {
	X float64
	Y float64
}

type coursewareComicPlanTailDirection struct {
	X float64
	Y float64
}

func buildCoursewareComicPanelRelations(
	panelNo int,
	sources []coursewareComicAIRelation,
	imageKeyByPanel map[int]string,
	hasCharacters bool,
) ([]models.CoursewareImageRelationSpec, string, error) {
	if panelNo == 1 && len(sources) > 0 {
		return nil, "", coursewareComicPlanOutputError("漫画第1格relations必须为空", nil)
	}
	if panelNo > 1 && len(sources) == 0 {
		return nil, "", coursewareComicPlanOutputError(fmt.Sprintf("漫画第%d格缺少连续关系", panelNo), nil)
	}

	specs := make([]models.CoursewareImageRelationSpec, 0, len(sources))
	publicRelations := make([]models.CoursewareComicPanelRelation, 0, len(sources))
	seen := make(map[string]bool)
	hasPreviousCharacterContinuity := false

	for _, source := range sources {
		if source.TargetPanelNo < 1 || source.TargetPanelNo >= panelNo {
			return nil, "", coursewareComicPlanOutputError("漫画关系只能引用更早的格", nil)
		}

		relationCode := strings.TrimSpace(source.RelationCode)
		if !models.IsValidCWImageRelationCode(relationCode) {
			return nil, "", coursewareComicPlanOutputError("漫画关系relation_code不合法", nil)
		}

		mask, ok := models.NormalizeCWImageInheritMask(source.InheritMask)
		if !ok {
			return nil, "", coursewareComicPlanOutputError("漫画关系inherit_mask不合法", nil)
		}

		targetKey := imageKeyByPanel[source.TargetPanelNo]
		uniqueKey := targetKey + "|" + relationCode + "|" + mask
		if seen[uniqueKey] {
			return nil, "", coursewareComicPlanOutputError("漫画关系重复", nil)
		}
		seen[uniqueKey] = true

		note := cwComicPlanTruncateRunes(source.SemanticNote, 500)
		specs = append(specs, models.CoursewareImageRelationSpec{
			TargetImageKey: targetKey,
			RelationCode:   relationCode,
			InheritMask:    mask,
			SemanticNote:   note,
		})
		publicRelations = append(publicRelations, models.CoursewareComicPanelRelation{
			TargetImageKey: targetKey,
			RelationCode:   relationCode,
			InheritMask:    mask,
			SemanticNote:   note,
		})

		if source.TargetPanelNo == panelNo-1 && relationCode == models.CWImageRelationContinue &&
			(!hasCharacters || strings.Contains(mask, "C")) {
			hasPreviousCharacterContinuity = true
		}
	}

	if panelNo > 1 && !hasPreviousCharacterContinuity {
		return nil, "", coursewareComicPlanOutputError(
			fmt.Sprintf("漫画第%d格必须用>关系延续上一格人物", panelNo),
			nil,
		)
	}

	encoded, err := json.Marshal(publicRelations)
	if err != nil {
		return nil, "", coursewareComicPlanOutputError("序列化漫画关系失败", err)
	}
	return specs, string(encoded), nil
}

func buildCoursewareComicOverlayDocument(
	panelNo int,
	sources []coursewareComicAIOverlayElement,
	panelCharacterSet map[string]bool,
	characterMaps ...map[string]models.CoursewareComicCharacter,
) (*models.CoursewareComicOverlayDocument, []models.CoursewareComicDialogue, []string, error) {
	var characterMap map[string]models.CoursewareComicCharacter
	if len(characterMaps) > 0 {
		characterMap = characterMaps[0]
	}

	if len(sources) < 1 || len(sources) > 8 {
		return nil, nil, nil, coursewareComicPlanOutputError(
			fmt.Sprintf("漫画第%d格覆盖层元素数量必须为1至8", panelNo),
			nil,
		)
	}

	sortedSources := append([]coursewareComicAIOverlayElement{}, sources...)
	sort.SliceStable(sortedSources, func(left int, right int) bool {
		leftPriority, rightPriority := sortedSources[left].Priority, sortedSources[right].Priority
		if leftPriority <= 0 {
			leftPriority = 100 + left
		}
		if rightPriority <= 0 {
			rightPriority = 100 + right
		}
		return leftPriority < rightPriority
	})

	usedIDs := make(map[string]bool)
	usedRegions := make(map[string]bool)
	elements := make([]models.CoursewareComicOverlayElement, 0, len(sortedSources))
	dialogues := make([]models.CoursewareComicDialogue, 0, len(sortedSources))
	reservedRegions := make([]string, 0, len(sortedSources))

	for index, source := range sortedSources {
		source.ID = strings.TrimSpace(source.ID)
		source.Type = strings.TrimSpace(source.Type)
		source.Content = strings.TrimSpace(source.Content)
		source.SpeakerID = strings.TrimSpace(source.SpeakerID)
		source.TargetCharacterID = strings.TrimSpace(source.TargetCharacterID)
		source.TargetAnchor = strings.TrimSpace(source.TargetAnchor)
		source.StyleID = strings.TrimSpace(source.StyleID)
		source.PreferredRegion = strings.TrimSpace(source.PreferredRegion)

		if source.ID == "" || len(source.ID) > 64 || usedIDs[source.ID] {
			return nil, nil, nil, coursewareComicPlanOutputError(
				fmt.Sprintf("漫画第%d格覆盖层元素ID为空或重复", panelNo),
				nil,
			)
		}
		usedIDs[source.ID] = true

		if !models.IsValidCWComicElementType(source.Type) {
			return nil, nil, nil, coursewareComicPlanOutputError("覆盖层元素type不合法", nil)
		}
		if !models.IsValidCWComicLayoutRegion(source.PreferredRegion) {
			return nil, nil, nil, coursewareComicPlanOutputError("覆盖层preferred_region不合法", nil)
		}
		if source.StyleID == "" {
			return nil, nil, nil, coursewareComicPlanOutputError("覆盖层style_id不能为空", nil)
		}

		if source.Type == models.CWComicElementQuestionCard {
			if err := validateCoursewareComicQuestion(source.Question); err != nil {
				return nil, nil, nil, err
			}
			if source.Content == "" {
				source.Content = "想一想"
			}
		} else {
			if source.Question != nil {
				return nil, nil, nil, coursewareComicPlanOutputError("非问题卡不能携带question对象", nil)
			}
			if source.Content == "" {
				return nil, nil, nil, coursewareComicPlanOutputError("覆盖层文字不能为空", nil)
			}
		}

		if utf8.RuneCountInString(source.Content) > 500 {
			return nil, nil, nil, coursewareComicPlanOutputError("覆盖层单项文字超过500字符", nil)
		}

		isBubble := source.Type == models.CWComicElementSpeechBubble ||
			source.Type == models.CWComicElementThoughtBubble
		targetX, targetY := 0.0, 0.0

		if isBubble {
			if source.TargetCharacterID == "" || !panelCharacterSet[source.TargetCharacterID] {
				return nil, nil, nil, coursewareComicPlanOutputError("对话气泡必须指向本格真实角色", nil)
			}
			if source.SpeakerID == "" || !panelCharacterSet[source.SpeakerID] {
				return nil, nil, nil, coursewareComicPlanOutputError("对话气泡speaker_id必须引用本格角色", nil)
			}
			if !models.IsValidCWComicCharacterAnchor(source.TargetAnchor) {
				return nil, nil, nil, coursewareComicPlanOutputError("气泡target_anchor不合法", nil)
			}

			targetX, targetY = resolveCoursewareComicCharacterTargetPoint(
				characterMap,
				source.TargetCharacterID,
				source.TargetAnchor,
			)
		}

		region := ""
		if isBubble {
			region = chooseCoursewareComicBubbleLayoutRegion(
				source.PreferredRegion,
				usedRegions,
				source.Type,
				targetX,
				targetY,
			)
		} else {
			region = chooseCoursewareComicLayoutRegion(
				source.PreferredRegion,
				usedRegions,
			)
		}
		if region == "" {
			return nil, nil, nil, coursewareComicPlanOutputError("覆盖层元素过多，无法完成无重叠自动排版", nil)
		}
		usedRegions[region] = true
		reservedRegions = append(reservedRegions, region)

		box := coursewareComicLayoutBoxForRegion(region, source.Type)
		var tail *models.CoursewareComicBubbleTail
		if isBubble {
			box, tail = prepareCoursewareComicAutomaticTail(box, targetX, targetY, source.TargetAnchor)
			dialogues = append(dialogues, models.CoursewareComicDialogue{
				ID:          source.ID,
				CharacterID: source.SpeakerID,
				Content:     source.Content,
				BubbleStyle: source.StyleID,
				Emotion:     "",
			})
		}

		priority := source.Priority
		if priority <= 0 {
			priority = index + 1
		}

		questionCopy := copyCoursewareComicQuestion(source.Question)
		elements = append(elements, models.CoursewareComicOverlayElement{
			ID:              source.ID,
			Type:            source.Type,
			Content:         source.Content,
			OriginalContent: source.Content,

			SpeakerID:         source.SpeakerID,
			TargetCharacterID: source.TargetCharacterID,
			TargetAnchor:      source.TargetAnchor,
			StyleID:           source.StyleID,

			AutoLayoutRegion: region,
			Priority:         priority,

			X:        box.X,
			Y:        box.Y,
			Width:    box.Width,
			Height:   box.Height,
			Rotation: 0,
			ZIndex:   20 + index,

			Tail:      tail,
			TextStyle: defaultCoursewareComicTextStyle(source.Type, source.Content, source.Question),

			Question:         questionCopy,
			OriginalQuestion: copyCoursewareComicQuestion(source.Question),

			Locked:       false,
			ContentDirty: false,
			LayoutDirty:  false,
		})
	}

	return &models.CoursewareComicOverlayDocument{
		Version: 1,
		Canvas: models.CoursewareComicOverlayCanvas{
			Width:  1920,
			Height: 1080,
		},
		Elements: elements,
	}, dialogues, reservedRegions, nil
}

// prepareCoursewareComicAutomaticTail 保留人物目标点，并通过移动气泡
// 把尾巴控制在约30至68像素。画布边界不足时接受实际距离，
// 不再把目标点替换成与人物无关的任意安全点。
func prepareCoursewareComicAutomaticTail(
	box coursewareComicLayoutBox,
	targetX float64,
	targetY float64,
	targetAnchor string,
) (coursewareComicLayoutBox, *models.CoursewareComicBubbleTail) {
	target := coursewareComicPlanTailPoint{
		X: clampCoursewareComicPlanUnit(targetX),
		Y: clampCoursewareComicPlanUnit(targetY),
	}

	for iteration := 0; iteration < 4; iteration++ {
		direction := resolveCoursewareComicPlanTailDirection(box, target, targetAnchor)
		origin := resolveCoursewareComicPlanTailOrigin(box, direction)
		distance := coursewareComicPlanTailDistancePixels(
			coursewareComicPlanTailOriginGlobal(box, origin),
			target,
		)
		desired := resolveCoursewareComicPlanTailDesiredLength(box)

		if !coursewareComicPlanPointInsideBox(target, box) &&
			math.Abs(distance-desired) <= 2 {
			break
		}

		translation := distance - desired
		if coursewareComicPlanPointInsideBox(target, box) {
			translation = -(coursewareComicPlanTailMinimumPixels -
				distance + coursewareComicPlanTailBoxGapPixels)
		}
		translation = math.Max(-140, math.Min(220, translation))

		previousX, previousY := box.X, box.Y
		box.X = math.Max(0, math.Min(
			1-box.Width,
			box.X+direction.X*translation/coursewareComicPlanCanvasWidth,
		))
		box.Y = math.Max(0, math.Min(
			1-box.Height,
			box.Y+direction.Y*translation/coursewareComicPlanCanvasHeight,
		))
		if math.Abs(box.X-previousX) < 0.0001 &&
			math.Abs(box.Y-previousY) < 0.0001 {
			break
		}
	}

	direction := resolveCoursewareComicPlanTailDirection(box, target, targetAnchor)
	origin := resolveCoursewareComicPlanTailOrigin(box, direction)
	originX, originY := origin.X, origin.Y

	return box, &models.CoursewareComicBubbleTail{
		Type:    "auto",
		TargetX: target.X,
		TargetY: target.Y,
		OriginX: &originX,
		OriginY: &originY,
	}
}

func resolveCoursewareComicPlanTailDesiredLength(
	box coursewareComicLayoutBox,
) float64 {
	shortSide := math.Min(
		box.Width*coursewareComicPlanCanvasWidth,
		box.Height*coursewareComicPlanCanvasHeight,
	)
	return math.Max(
		coursewareComicPlanTailMinimumPixels,
		math.Min(coursewareComicPlanTailMaximumPixels, shortSide*0.18),
	)
}

func resolveCoursewareComicPlanTailDirection(
	box coursewareComicLayoutBox,
	target coursewareComicPlanTailPoint,
	targetAnchor string,
) coursewareComicPlanTailDirection {
	centerX := box.X + box.Width/2
	centerY := box.Y + box.Height/2
	direction := coursewareComicPlanTailDirection{
		X: (target.X - centerX) * coursewareComicPlanCanvasWidth,
		Y: (target.Y - centerY) * coursewareComicPlanCanvasHeight,
	}
	if math.Hypot(direction.X, direction.Y) < 1 {
		direction = coursewareComicPlanFallbackTailDirection(targetAnchor)
	}
	return normalizeCoursewareComicPlanTailDirection(direction)
}

func coursewareComicPlanFallbackTailDirection(anchor string) coursewareComicPlanTailDirection {
	switch strings.TrimSpace(anchor) {
	case models.CWComicAnchorLeftTop:
		return coursewareComicPlanTailDirection{X: -1, Y: -1}
	case models.CWComicAnchorLeftCenter:
		return coursewareComicPlanTailDirection{X: -1, Y: 0}
	case models.CWComicAnchorLeftBottom:
		return coursewareComicPlanTailDirection{X: -1, Y: 1}
	case models.CWComicAnchorCenterTop:
		return coursewareComicPlanTailDirection{X: 0, Y: -1}
	case models.CWComicAnchorCenter:
		return coursewareComicPlanTailDirection{X: 0, Y: 1}
	case models.CWComicAnchorRightTop:
		return coursewareComicPlanTailDirection{X: 1, Y: -1}
	case models.CWComicAnchorRightCenter:
		return coursewareComicPlanTailDirection{X: 1, Y: 0}
	case models.CWComicAnchorRightBottom:
		return coursewareComicPlanTailDirection{X: 1, Y: 1}
	default:
		return coursewareComicPlanTailDirection{X: 0, Y: 1}
	}
}

func normalizeCoursewareComicPlanTailDirection(
	direction coursewareComicPlanTailDirection,
) coursewareComicPlanTailDirection {
	length := math.Hypot(direction.X, direction.Y)
	if length < 0.001 {
		return coursewareComicPlanTailDirection{X: 0, Y: 1}
	}
	return coursewareComicPlanTailDirection{X: direction.X / length, Y: direction.Y / length}
}

func resolveCoursewareComicPlanTailOrigin(
	box coursewareComicLayoutBox,
	direction coursewareComicPlanTailDirection,
) coursewareComicPlanTailPoint {
	direction = normalizeCoursewareComicPlanTailDirection(direction)
	maxX := math.Inf(1)
	maxY := math.Inf(1)

	if math.Abs(direction.X) > 0.0001 {
		maxX = box.Width * coursewareComicPlanCanvasWidth / 2 / math.Abs(direction.X)
	}
	if math.Abs(direction.Y) > 0.0001 {
		maxY = box.Height * coursewareComicPlanCanvasHeight / 2 / math.Abs(direction.Y)
	}

	distance := math.Min(maxX, maxY)
	globalX := box.X + box.Width/2 + direction.X*distance/coursewareComicPlanCanvasWidth
	globalY := box.Y + box.Height/2 + direction.Y*distance/coursewareComicPlanCanvasHeight
	localX := (globalX - box.X) / math.Max(box.Width, 0.001)
	localY := (globalY - box.Y) / math.Max(box.Height, 0.001)

	if localX <= 0.001 || localX >= 0.999 {
		localX = math.Round(localX)
		localY = math.Max(
			coursewareComicPlanTailOriginEdgeInset,
			math.Min(1-coursewareComicPlanTailOriginEdgeInset, localY),
		)
	} else {
		localX = math.Max(
			coursewareComicPlanTailOriginEdgeInset,
			math.Min(1-coursewareComicPlanTailOriginEdgeInset, localX),
		)
		localY = math.Round(localY)
	}

	return coursewareComicPlanTailPoint{X: localX, Y: localY}
}

func coursewareComicPlanTailOriginGlobal(
	box coursewareComicLayoutBox,
	origin coursewareComicPlanTailPoint,
) coursewareComicPlanTailPoint {
	return coursewareComicPlanTailPoint{
		X: box.X + box.Width*origin.X,
		Y: box.Y + box.Height*origin.Y,
	}
}

func coursewareComicPlanTailDistancePixels(
	left coursewareComicPlanTailPoint,
	right coursewareComicPlanTailPoint,
) float64 {
	return math.Hypot(
		(right.X-left.X)*coursewareComicPlanCanvasWidth,
		(right.Y-left.Y)*coursewareComicPlanCanvasHeight,
	)
}

func coursewareComicPlanPointInsideBox(
	point coursewareComicPlanTailPoint,
	box coursewareComicLayoutBox,
) bool {
	return point.X >= box.X && point.X <= box.X+box.Width &&
		point.Y >= box.Y && point.Y <= box.Y+box.Height
}

func clampCoursewareComicPlanUnit(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}

func validateCoursewareComicQuestion(question *models.CoursewareComicQuestionContent) error {
	if question == nil {
		return coursewareComicPlanOutputError("问题卡必须携带question对象", nil)
	}

	question.Question = strings.TrimSpace(question.Question)
	question.Explanation = strings.TrimSpace(question.Explanation)
	question.AnswerMode = strings.TrimSpace(question.AnswerMode)
	question.Options = normalizeCoursewareComicStrings(question.Options, 6, 300)

	if question.Question == "" || question.Explanation == "" {
		return coursewareComicPlanOutputError("题干和答案解析不能为空", nil)
	}
	if len(question.Options) < 2 {
		return coursewareComicPlanOutputError("问题卡至少需要两个选项", nil)
	}
	if question.AnswerIndex < 0 || question.AnswerIndex >= len(question.Options) {
		return coursewareComicPlanOutputError("问题卡answer_index越界", nil)
	}
	if !models.IsValidCWComicAnswerMode(question.AnswerMode) {
		return coursewareComicPlanOutputError("问题卡answer_mode不合法", nil)
	}
	return nil
}

// chooseCoursewareComicBubbleLayoutRegion 优先选择靠近人物、又不覆盖人物的
// 未使用区域，避免尾巴横跨大半画面。preferred_region仍作为软偏好。
func chooseCoursewareComicBubbleLayoutRegion(
	preferred string,
	used map[string]bool,
	elementType string,
	targetX float64,
	targetY float64,
) string {
	candidates := []string{
		preferred,
		models.CWComicRegionTopLeft,
		models.CWComicRegionTopCenter,
		models.CWComicRegionTopRight,
		models.CWComicRegionMiddleLeft,
		models.CWComicRegionMiddleRight,
		models.CWComicRegionBottomLeft,
		models.CWComicRegionBottomCenter,
		models.CWComicRegionBottomRight,
	}
	target := coursewareComicPlanTailPoint{
		X: clampCoursewareComicPlanUnit(targetX),
		Y: clampCoursewareComicPlanUnit(targetY),
	}
	bestRegion, bestScore := "", math.Inf(1)
	seenCandidates := make(map[string]bool)

	for _, candidate := range candidates {
		if candidate == "" || seenCandidates[candidate] || used[candidate] ||
			!models.IsValidCWComicLayoutRegion(candidate) {
			continue
		}
		seenCandidates[candidate] = true

		box := coursewareComicLayoutBoxForRegion(candidate, elementType)
		distance := coursewareComicPlanPointToBoxDistancePixels(target, box)
		score := math.Abs(
			distance - resolveCoursewareComicPlanTailDesiredLength(box),
		)
		if coursewareComicPlanPointInsideBox(target, box) {
			score += 10000
		}
		if candidate == preferred {
			score -= 16
		}
		if score < bestScore {
			bestRegion, bestScore = candidate, score
		}
	}

	return bestRegion
}

func coursewareComicPlanPointToBoxDistancePixels(
	point coursewareComicPlanTailPoint,
	box coursewareComicLayoutBox,
) float64 {
	deltaX, deltaY := 0.0, 0.0
	if point.X < box.X {
		deltaX = (box.X - point.X) * coursewareComicPlanCanvasWidth
	} else if point.X > box.X+box.Width {
		deltaX = (point.X - box.X - box.Width) * coursewareComicPlanCanvasWidth
	}
	if point.Y < box.Y {
		deltaY = (box.Y - point.Y) * coursewareComicPlanCanvasHeight
	} else if point.Y > box.Y+box.Height {
		deltaY = (point.Y - box.Y - box.Height) * coursewareComicPlanCanvasHeight
	}
	return math.Hypot(deltaX, deltaY)
}

func chooseCoursewareComicLayoutRegion(preferred string, used map[string]bool) string {
	candidates := []string{
		preferred,
		models.CWComicRegionTopLeft,
		models.CWComicRegionTopRight,
		models.CWComicRegionBottomLeft,
		models.CWComicRegionBottomRight,
		models.CWComicRegionTopCenter,
		models.CWComicRegionBottomCenter,
		models.CWComicRegionMiddleLeft,
		models.CWComicRegionMiddleRight,
	}
	for _, candidate := range candidates {
		if candidate != "" && !used[candidate] && models.IsValidCWComicLayoutRegion(candidate) {
			return candidate
		}
	}
	return ""
}

func coursewareComicLayoutBoxForRegion(region string, elementType string) coursewareComicLayoutBox {
	boxes := map[string]coursewareComicLayoutBox{
		models.CWComicRegionTopLeft:      {X: 0.04, Y: 0.05, Width: 0.36, Height: 0.21},
		models.CWComicRegionTopCenter:    {X: 0.30, Y: 0.04, Width: 0.40, Height: 0.20},
		models.CWComicRegionTopRight:     {X: 0.60, Y: 0.05, Width: 0.36, Height: 0.21},
		models.CWComicRegionMiddleLeft:   {X: 0.03, Y: 0.36, Width: 0.34, Height: 0.24},
		models.CWComicRegionMiddleRight:  {X: 0.63, Y: 0.36, Width: 0.34, Height: 0.24},
		models.CWComicRegionBottomLeft:   {X: 0.04, Y: 0.70, Width: 0.40, Height: 0.24},
		models.CWComicRegionBottomCenter: {X: 0.25, Y: 0.70, Width: 0.50, Height: 0.25},
		models.CWComicRegionBottomRight:  {X: 0.56, Y: 0.70, Width: 0.40, Height: 0.24},
	}

	box := boxes[region]
	switch elementType {
	case models.CWComicElementQuestionCard:
		box.Height = 0.29
	case models.CWComicElementNarration, models.CWComicElementCaption:
		box.Height = 0.16
	case models.CWComicElementEmphasis:
		box.Width *= 0.75
		box.Height = 0.15
	}
	return box
}

func defaultCoursewareComicTextStyle(
	elementType string,
	content string,
	question *models.CoursewareComicQuestionContent,
) models.CoursewareComicTextStyle {
	textLength := utf8.RuneCountInString(content)
	if question != nil {
		textLength += utf8.RuneCountInString(question.Question)
		for _, option := range question.Options {
			textLength += utf8.RuneCountInString(option)
		}
	}

	fontSize := 40.0
	switch {
	case textLength > 180:
		fontSize = 25
	case textLength > 120:
		fontSize = 28
	case textLength > 80:
		fontSize = 31
	case textLength > 45:
		fontSize = 35
	}

	fontWeight, align := 600, "center"
	switch elementType {
	case models.CWComicElementNarration, models.CWComicElementCaption:
		fontWeight, align = 500, "left"
	case models.CWComicElementKnowledgeCard,
		models.CWComicElementWarningCard,
		models.CWComicElementQuestionCard,
		models.CWComicElementAnswerCard:
		align = "left"
	}

	return models.CoursewareComicTextStyle{
		FontFamily:        "Noto Sans SC, sans-serif",
		FontSize:          fontSize,
		FontWeight:        fontWeight,
		LineHeight:        1.45,
		Align:             align,
		Color:             "#1F2937",
		ColorMode:         models.CWComicTextColorModeAuto,
		BackgroundOpacity: 0.94,
		OutlineWidth:      1,
	}
}

func resolveCoursewareComicAnchorPoint(anchor string) (float64, float64) {
	points := map[string][2]float64{
		models.CWComicAnchorLeftTop:      {0.20, 0.25},
		models.CWComicAnchorLeftCenter:   {0.20, 0.50},
		models.CWComicAnchorLeftBottom:   {0.20, 0.76},
		models.CWComicAnchorCenterTop:    {0.50, 0.25},
		models.CWComicAnchorCenter:       {0.50, 0.50},
		models.CWComicAnchorCenterBottom: {0.50, 0.76},
		models.CWComicAnchorRightTop:     {0.80, 0.25},
		models.CWComicAnchorRightCenter:  {0.80, 0.50},
		models.CWComicAnchorRightBottom:  {0.80, 0.76},
	}
	point := points[anchor]
	return point[0], point[1]
}

func buildCoursewareComicCharacterSummary(characters []models.CoursewareComicCharacter) string {
	parts := make([]string, 0, len(characters))
	for _, character := range characters {
		parts = append(parts, fmt.Sprintf(
			"%s(%s)：%s；固定特征：%s",
			character.ID,
			character.Name,
			character.Appearance,
			strings.Join(character.FixedFeatures, "、"),
		))
	}
	if len(parts) == 0 {
		return "Ø"
	}
	return strings.Join(parts, "；")
}

func resolveCoursewareComicImageSubjectType(characters []models.CoursewareComicCharacter) string {
	if len(characters) == 0 {
		return models.CWImageSubjectNone
	}

	types := make(map[string]bool)
	for _, character := range characters {
		types[character.SubjectType] = true
	}
	if len(types) > 1 {
		return models.CWImageSubjectMixed
	}

	for subjectType := range types {
		switch subjectType {
		case models.CWComicCharacterSubjectPerson:
			return models.CWImageSubjectPerson
		case models.CWComicCharacterSubjectAnimal:
			return models.CWImageSubjectAnimal
		case models.CWComicCharacterSubjectObject:
			return models.CWImageSubjectObject
		}
	}
	return models.CWImageSubjectMixed
}

func normalizeCoursewareComicStrings(values []string, maxItems int, maxRunes int) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]bool)

	for _, value := range values {
		value = cwComicPlanTruncateRunes(value, maxRunes)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
		if len(result) >= maxItems {
			break
		}
	}
	return result
}

func sortedCoursewareComicCharacterIDs(
	characters map[string]models.CoursewareComicCharacter,
) []string {
	ids := make([]string, 0, len(characters))
	for id := range characters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func copyCoursewareComicQuestion(
	source *models.CoursewareComicQuestionContent,
) *models.CoursewareComicQuestionContent {
	if source == nil {
		return nil
	}

	return &models.CoursewareComicQuestionContent{
		Question:    source.Question,
		Options:     append([]string{}, source.Options...),
		AnswerIndex: source.AnswerIndex,
		Explanation: source.Explanation,
		AnswerMode:  source.AnswerMode,
	}
}
