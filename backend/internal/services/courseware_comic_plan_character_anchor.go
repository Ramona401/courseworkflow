package services

// courseware_comic_plan_character_anchor.go
//
// 知识点漫画人物锚点纯函数：
//   - target_character_id确定目标人物；
//   - default_position确定人物在整格中的主位置；
//   - target_anchor只表示人物自身附近的局部指向方位；
//   - 返回0至1归一化画布坐标。

import (
	"fmt"
	"strings"

	"tedna/internal/models"
)

var coursewareComicCharacterBasePoints = map[string][2]float64{
	models.CWComicAnchorLeftTop: {0.20, 0.32},

	models.CWComicAnchorLeftCenter: {0.20, 0.50},

	models.CWComicAnchorLeftBottom: {0.20, 0.68},

	models.CWComicAnchorCenterTop: {0.50, 0.32},

	models.CWComicAnchorCenter: {0.50, 0.50},

	models.CWComicAnchorCenterBottom: {0.50, 0.68},

	models.CWComicAnchorRightTop: {0.80, 0.32},

	models.CWComicAnchorRightCenter: {0.80, 0.50},

	models.CWComicAnchorRightBottom: {0.80, 0.68},
}

var coursewareComicCharacterTargetOffsets = map[string][2]float64{
	models.CWComicAnchorLeftTop: {-0.032, -0.040},

	models.CWComicAnchorLeftCenter: {-0.038, 0},

	models.CWComicAnchorLeftBottom: {-0.032, 0.040},

	models.CWComicAnchorCenterTop: {0, -0.046},

	models.CWComicAnchorCenter: {0, 0},

	models.CWComicAnchorCenterBottom: {0, 0.046},

	models.CWComicAnchorRightTop: {0.032, -0.040},

	models.CWComicAnchorRightCenter: {0.038, 0},

	models.CWComicAnchorRightBottom: {0.032, 0.040},
}

// buildCoursewareComicPanelCharacterContext 校验每格人物位置并构建
// 本格专用人物映射。全局人物设定保持不变；仅本格副本的
// DefaultPosition被结构化character_positions覆盖。
func buildCoursewareComicPanelCharacterContext(
	panelNo int,
	characterIDs []string,
	sources []coursewareComicAIPanelCharacterPosition,
	characterMap map[string]models.CoursewareComicCharacter,
) (
	[]models.CoursewareComicCharacter,
	map[string]models.CoursewareComicCharacter,
	map[string]bool,
	string,
	error,
) {
	if len(characterIDs) == 0 ||
		len(sources) != len(characterIDs) {
		return nil, nil, nil, "",
			coursewareComicPlanOutputError(
				fmt.Sprintf(
					"漫画第%d格character_positions必须完整覆盖characters",
					panelNo,
				),
				nil,
			)
	}

	panelSet := make(map[string]bool, len(characterIDs))
	for _, rawID := range characterIDs {
		characterID := strings.TrimSpace(rawID)
		if characterID == "" ||
			panelSet[characterID] {
			return nil, nil, nil, "",
				coursewareComicPlanOutputError(
					fmt.Sprintf(
						"漫画第%d格characters存在空值或重复",
						panelNo,
					),
					nil,
				)
		}
		if _, exists := characterMap[characterID]; !exists {
			return nil, nil, nil, "",
				coursewareComicPlanOutputError(
					fmt.Sprintf(
						"漫画第%d格引用未定义角色%s",
						panelNo,
						characterID,
					),
					nil,
				)
		}
		panelSet[characterID] = true
	}

	positions := make(map[string]string, len(sources))
	for _, source := range sources {
		characterID := strings.TrimSpace(source.CharacterID)
		position := strings.TrimSpace(source.Position)

		if !panelSet[characterID] ||
			positions[characterID] != "" ||
			!models.IsValidCWComicCharacterAnchor(position) {
			return nil, nil, nil, "",
				coursewareComicPlanOutputError(
					fmt.Sprintf(
						"漫画第%d格character_positions存在未知、重复或非法位置",
						panelNo,
					),
					nil,
				)
		}
		positions[characterID] = position
	}

	panelMap := make(
		map[string]models.CoursewareComicCharacter,
		len(characterMap),
	)
	for characterID, character := range characterMap {
		panelMap[characterID] = character
	}

	panelCharacters := make(
		[]models.CoursewareComicCharacter,
		0,
		len(characterIDs),
	)
	summaries := make([]string, 0, len(characterIDs))

	for _, characterID := range characterIDs {
		position, exists := positions[characterID]
		if !exists {
			return nil, nil, nil, "",
				coursewareComicPlanOutputError(
					fmt.Sprintf(
						"漫画第%d格缺少角色%s的位置",
						panelNo,
						characterID,
					),
					nil,
				)
		}

		character := characterMap[characterID]
		character.DefaultPosition = position
		panelMap[characterID] = character
		panelCharacters = append(panelCharacters, character)
		summaries = append(
			summaries,
			fmt.Sprintf(
				"%s(%s)=%s",
				character.Name,
				character.ID,
				coursewareComicCharacterAnchorLabel(position),
			),
		)
	}

	return panelCharacters,
		panelMap,
		panelSet,
		strings.Join(summaries, "、"),
		nil
}

func coursewareComicCharacterAnchorLabel(
	position string,
) string {
	labels := map[string]string{
		models.CWComicAnchorLeftTop:      "左上",
		models.CWComicAnchorLeftCenter:   "左中",
		models.CWComicAnchorLeftBottom:   "左下",
		models.CWComicAnchorCenterTop:    "中上",
		models.CWComicAnchorCenter:       "中央",
		models.CWComicAnchorCenterBottom: "中下",
		models.CWComicAnchorRightTop:     "右上",
		models.CWComicAnchorRightCenter:  "右中",
		models.CWComicAnchorRightBottom:  "右下",
	}
	if label := labels[strings.TrimSpace(position)]; label != "" {
		return label
	}
	return strings.TrimSpace(position)
}

// resolveCoursewareComicCharacterTargetPoint
// 根据目标人物和人物局部方位，返回尾巴在整格画布中的目标点。
// characterMap中的DefaultPosition可以是项目默认值，也可以是本格副本覆盖值。
func resolveCoursewareComicCharacterTargetPoint(
	characterMap map[string]models.CoursewareComicCharacter,
	characterID string,
	targetAnchor string,
) (float64, float64) {
	character, exists :=
		characterMap[strings.TrimSpace(
			characterID,
		)]

	if !exists {
		return resolveCoursewareComicAnchorPoint(
			targetAnchor,
		)
	}

	base, exists :=
		coursewareComicCharacterBasePoints[strings.TrimSpace(
			character.DefaultPosition,
		)]

	if !exists {
		base =
			[2]float64{
				0.50,
				0.50,
			}
	}

	offset :=
		coursewareComicCharacterTargetOffsets[strings.TrimSpace(
			targetAnchor,
		)]

	return clampCoursewareComicCharacterCoordinate(
			base[0] +
				offset[0],
		),
		clampCoursewareComicCharacterCoordinate(
			base[1] +
				offset[1],
		)
}

func clampCoursewareComicCharacterCoordinate(
	value float64,
) float64 {
	switch {
	case value < 0.05:
		return 0.05

	case value > 0.95:
		return 0.95

	default:
		return value
	}
}
