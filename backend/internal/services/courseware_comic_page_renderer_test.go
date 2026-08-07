package services

// courseware_comic_page_renderer_test.go
//
// 只测试确定性渲染纯函数。
// 不连接数据库、不调用AI、不读写图片文件。

import (
	"encoding/json"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCoursewareComicPageRendererEscapesTeacherTextAndBuildsMarkers(
	t *testing.T,
) {
	overlayJSON :=
		buildCoursewareComicRendererTestOverlay(
			t,
		)

	project :=
		&models.CoursewareComicProject{
			ID:
				"project-1",
			Title:
				`电解质<script>alert("x")</script>`,
			Subject:
				"化学",
			Grade:
				"高一",
			PublisherSnapshot:
				"人教版",
			SemesterSnapshot:
				"必修第一册",
		}

	courseware :=
		&models.Courseware{
			ID:
				"courseware-1",
			Title:
				"化学课件",
		}

	renderData := make(
		[]coursewareComicPanelRenderData,
		0,
		4,
	)

	for panelNo := 1;
		panelNo <= 4;
		panelNo++ {
		panel :=
			&models.CoursewareComicPanel{
				ID:
					"panel-" +
						string(
							rune(
								'0'+panelNo,
							),
						),
				PanelNo:
					panelNo,
				KnowledgeClaim:
					"知识结论",
				OverlayDocumentJSON:
					overlayJSON,
			}

		renderData = append(
			renderData,
			coursewareComicPanelRenderData{
				Panel:
					panel,
				ImageURL:
					"https://example.com/panel.png",
			},
		)
	}

	rendered, err :=
		renderCoursewareComicPageHTML(
			courseware,
			project,
			renderData,
			2,
			6,
		)
	if err != nil {
		t.Fatalf(
			"漫画页面渲染失败: %v",
			err,
		)
	}

	required := []string{
		coursewareComicProjectStartMarker(
			project.ID,
		),
		coursewareComicProjectEndMarker(
			project.ID,
		),
		coursewareComicPanelStartMarker(
			project.ID,
			"panel-1",
		),
		"tedna-comic-layout--4",
		"data-tedna-answer-target",
		"查看答案",
		"&lt;script&gt;",
	}

	for _, value := range required {
		if !strings.Contains(
			rendered,
			value,
		) {
			t.Fatalf(
				"渲染结果缺少内容: %s",
				value,
			)
		}
	}

	if strings.Contains(
		rendered,
		`<script>alert("x")</script>`,
	) {
		t.Fatal(
			"教师文本没有执行HTML转义",
		)
	}
}

func buildCoursewareComicRendererTestOverlay(
	t *testing.T,
) string {
	t.Helper()

	document :=
		models.CoursewareComicOverlayDocument{
			Version:
				1,
			Canvas:
				models.CoursewareComicOverlayCanvas{
					Width:
						1920,
					Height:
						1080,
				},
			Elements:
				[]models.CoursewareComicOverlayElement{
					{
						ID:
							"question-1",
						Type:
							models.CWComicElementQuestionCard,
						Content:
							"想一想",
						StyleID:
							"question_purple",
						X:
							0.08,
						Y:
							0.08,
						Width:
							0.46,
						Height:
							0.36,
						ZIndex:
							30,
						TextStyle:
							models.CoursewareComicTextStyle{
								FontSize:
									28,
								FontWeight:
									600,
								LineHeight:
									1.4,
								Align:
									"left",
								Color:
									"#3B0764",
							},
						Question:
							&models.CoursewareComicQuestionContent{
								Question:
									"下列哪项属于电解质？",
								Options:
									[]string{
										"氯化钠",
										"蔗糖",
									},
								AnswerIndex:
									0,
								Explanation:
									"氯化钠在水溶液中能够电离。",
								AnswerMode:
									models.CWComicAnswerModeClickReveal,
							},
					},
				},
		}

	encoded, err :=
		json.Marshal(document)
	if err != nil {
		t.Fatalf(
			"构造测试覆盖层失败: %v",
			err,
		)
	}

	return string(encoded)
}
