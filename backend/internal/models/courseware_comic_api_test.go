package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCoursewareComicBrowserJSONContract(
	t *testing.T,
) {
	project :=
		&CoursewareComicProjectView{
			ID:
				"project-1",
			CharacterSheetAssetID:
				stringPointerForComicTest(
					"asset-sheet",
				),
			CharacterSheetURL:
				"https://example.com/character-sheet.png",
		}

	panel :=
		&CoursewareComicPanelView{
			ID:
				"panel-1",
			CurrentAssetID:
				stringPointerForComicTest(
					"asset-panel",
				),
			CurrentAssetURL:
				"https://example.com/panel-1.png",
		}

	raw, err :=
		json.Marshal(
			&CoursewareComicProjectDetailView{
				Project:
					project,
				Panels:
					[]*CoursewareComicPanelView{
						panel,
					},
			},
		)
	if err != nil {
		t.Fatalf(
			"序列化漫画浏览器协议失败: %v",
			err,
		)
	}

	payload :=
		string(raw)

	required := []string{
		`"character_sheet_asset_id":"asset-sheet"`,
		`"character_sheet_url":"https://example.com/character-sheet.png"`,
		`"current_asset_id":"asset-panel"`,
		`"current_asset_url":"https://example.com/panel-1.png"`,
	}

	for _, expected :=
		range required {
		if !strings.Contains(
			payload,
			expected,
		) {
			t.Fatalf(
				"漫画浏览器协议缺少字段: %s；实际内容: %s",
				expected,
				payload,
			)
		}
	}
}

func stringPointerForComicTest(
	value string,
) *string {
	return &value
}
