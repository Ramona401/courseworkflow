package services

import (
	"strconv"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCoursewareComicStylePreviewImageSizeMatrix(
	t *testing.T,
) {
	cases := []struct {
		name        string
		aspectRatio string
		quality     string
		expected    string
	}{
		{
			name:        "courseware standard",
			aspectRatio: models.CWComicAspectRatioCourseware,
			quality:     models.CWComicImageQualityStandard,
			expected:    "2560x1440",
		},
		{
			name:        "courseware high",
			aspectRatio: models.CWComicAspectRatioCourseware,
			quality:     models.CWComicImageQualityHigh,
			expected:    "3200x1800",
		},
		{
			name:        "16x9 standard",
			aspectRatio: models.CWComicAspectRatio16x9,
			quality:     models.CWComicImageQualityStandard,
			expected:    "2560x1440",
		},
		{
			name:        "16x9 high",
			aspectRatio: models.CWComicAspectRatio16x9,
			quality:     models.CWComicImageQualityHigh,
			expected:    "3200x1800",
		},
		{
			name:        "4x3 standard",
			aspectRatio: models.CWComicAspectRatio4x3,
			quality:     models.CWComicImageQualityStandard,
			expected:    "2304x1728",
		},
		{
			name:        "4x3 high",
			aspectRatio: models.CWComicAspectRatio4x3,
			quality:     models.CWComicImageQualityHigh,
			expected:    "3072x2304",
		},
		{
			name:        "1x1 standard",
			aspectRatio: models.CWComicAspectRatio1x1,
			quality:     models.CWComicImageQualityStandard,
			expected:    "1920x1920",
		},
		{
			name:        "1x1 high",
			aspectRatio: models.CWComicAspectRatio1x1,
			quality:     models.CWComicImageQualityHigh,
			expected:    "2560x2560",
		},
		{
			name:        "3x4 standard",
			aspectRatio: models.CWComicAspectRatio3x4,
			quality:     models.CWComicImageQualityStandard,
			expected:    "1728x2304",
		},
		{
			name:        "3x4 high",
			aspectRatio: models.CWComicAspectRatio3x4,
			quality:     models.CWComicImageQualityHigh,
			expected:    "2304x3072",
		},
		{
			name:        "9x16 standard",
			aspectRatio: models.CWComicAspectRatio9x16,
			quality:     models.CWComicImageQualityStandard,
			expected:    "1440x2560",
		},
		{
			name:        "9x16 high",
			aspectRatio: models.CWComicAspectRatio9x16,
			quality:     models.CWComicImageQualityHigh,
			expected:    "1800x3200",
		},
	}

	for _, testCase := range cases {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				actual, valid :=
					resolveCoursewareComicStylePreviewImageSize(
						testCase.aspectRatio,
						testCase.quality,
					)

				if !valid {
					t.Fatal(
						"合法比例和清晰度被拒绝",
					)
				}

				if actual != testCase.expected {
					t.Fatalf(
						"尺寸错误：得到%q，期望%q",
						actual,
						testCase.expected,
					)
				}

				width, height, ok :=
					parseCoursewareComicTestImageSize(
						actual,
					)
				if !ok {
					t.Fatalf(
						"尺寸无法解析：%q",
						actual,
					)
				}

				if width*height < 3686400 {
					t.Fatalf(
						"尺寸未达到图片网关最低像素：%s=%d",
						actual,
						width*height,
					)
				}
			},
		)
	}
}

func TestBuildCoursewareComicStylePreviewRenderPlan(
	t *testing.T,
) {
	project :=
		&models.CoursewareComicProject{
			VisualStyle:        models.CWComicVisualChineseInk,
			StyleAOCIText:      "旧规划风格：欧美卡通，鲜艳粗轮廓",
			CharacterBibleJSON: `{"characters":[{"id":"teacher","name":"李老师"}]}`,
		}

	panel :=
		&models.CoursewareComicPanel{
			PanelNo:        1,
			VisualPrompt:   "李老师在课堂中展示知识对象",
			AOCIText:       "本格完整视觉关系",
			NegativePrompt: "禁止文字和水印",
		}

	workflow :=
		&models.CoursewareComicWorkflowState{
			ProjectID:        "project-1",
			Stage:            models.CWComicWorkflowStylePreview,
			AspectRatio:      models.CWComicAspectRatio3x4,
			ImageQuality:     models.CWComicImageQualityHigh,
			InsertionMode:    models.CWComicInsertionSinglePage,
			StyleInstruction: "教师要求：淡雅青绿色，人物衣着符合现代课堂",
		}

	plan, valid :=
		buildCoursewareComicStylePreviewRenderPlan(
			project,
			panel,
			workflow,
		)

	if !valid ||
		plan == nil {
		t.Fatal(
			"合法样张视觉参数未生成渲染计划",
		)
	}

	if plan.ImageSize != "2304x3072" {
		t.Fatalf(
			"样张尺寸错误：%q",
			plan.ImageSize,
		)
	}

	requiredTexts :=
		[]string{
			"现代国风教育插画",
			"竖向3:4画幅",
			"高清精细质量",
			"教师要求：淡雅青绿色",
			"本次只生成第1格完整视觉样张",
			"不得在图片中绘制气泡",
			"不得生成任何文字",
		}

	for _, requiredText := range requiredTexts {
		if !strings.Contains(
			plan.Prompt,
			requiredText,
		) {
			t.Fatalf(
				"样张提示词缺少要求：%q",
				requiredText,
			)
		}
	}

	oldStyleIndex :=
		strings.Index(
			plan.Prompt,
			"旧规划风格：欧美卡通",
		)

	confirmedStyleIndex :=
		strings.Index(
			plan.Prompt,
			"【第三步教师确认画风】",
		)

	teacherInstructionIndex :=
		strings.Index(
			plan.Prompt,
			"【教师补充画风要求】",
		)

	if oldStyleIndex < 0 ||
		confirmedStyleIndex < 0 ||
		teacherInstructionIndex < 0 {
		t.Fatal(
			"样张提示词缺少优先级定位标记",
		)
	}

	if confirmedStyleIndex <= oldStyleIndex {
		t.Fatal(
			"教师确认画风必须位于旧规划风格之后",
		)
	}

	if teacherInstructionIndex <=
		confirmedStyleIndex {
		t.Fatal(
			"教师补充要求必须位于确认画风之后",
		)
	}
}

func TestCoursewareComicStylePreviewRenderPlanRejectsInvalidInput(
	t *testing.T,
) {
	validProject :=
		&models.CoursewareComicProject{
			VisualStyle: models.CWComicVisualModernFlat,
		}

	validPanel :=
		&models.CoursewareComicPanel{
			PanelNo:      1,
			VisualPrompt: "教学场景",
			AOCIText:     "完整IAOCI",
		}

	validWorkflow :=
		&models.CoursewareComicWorkflowState{
			ProjectID:     "project-1",
			Stage:         models.CWComicWorkflowStylePreview,
			AspectRatio:   models.CWComicAspectRatio16x9,
			ImageQuality:  models.CWComicImageQualityStandard,
			InsertionMode: models.CWComicInsertionSinglePage,
		}

	cases := []struct {
		name     string
		project  *models.CoursewareComicProject
		panel    *models.CoursewareComicPanel
		workflow *models.CoursewareComicWorkflowState
	}{
		{
			name:     "nil project",
			panel:    validPanel,
			workflow: validWorkflow,
		},
		{
			name:     "nil panel",
			project:  validProject,
			workflow: validWorkflow,
		},
		{
			name:    "nil workflow",
			project: validProject,
			panel:   validPanel,
		},
		{
			name: "invalid visual style",
			project: &models.CoursewareComicProject{
				VisualStyle: "invalid_style",
			},
			panel:    validPanel,
			workflow: validWorkflow,
		},
		{
			name:    "invalid aspect ratio",
			project: validProject,
			panel:   validPanel,
			workflow: &models.CoursewareComicWorkflowState{
				ProjectID:     "project-1",
				Stage:         models.CWComicWorkflowStylePreview,
				AspectRatio:   "2:1",
				ImageQuality:  models.CWComicImageQualityStandard,
				InsertionMode: models.CWComicInsertionSinglePage,
			},
		},
		{
			name:    "invalid image quality",
			project: validProject,
			panel:   validPanel,
			workflow: &models.CoursewareComicWorkflowState{
				ProjectID:     "project-1",
				Stage:         models.CWComicWorkflowStylePreview,
				AspectRatio:   models.CWComicAspectRatio16x9,
				ImageQuality:  "ultra",
				InsertionMode: models.CWComicInsertionSinglePage,
			},
		},
		{
			name:    "style instruction over limit",
			project: validProject,
			panel:   validPanel,
			workflow: &models.CoursewareComicWorkflowState{
				ProjectID:     "project-1",
				Stage:         models.CWComicWorkflowStylePreview,
				AspectRatio:   models.CWComicAspectRatio16x9,
				ImageQuality:  models.CWComicImageQualityStandard,
				InsertionMode: models.CWComicInsertionSinglePage,
				StyleInstruction: strings.Repeat(
					"画",
					models.CoursewareComicMaxStyleInstructionRunes+1,
				),
			},
		},
	}

	for _, testCase := range cases {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				plan, valid :=
					buildCoursewareComicStylePreviewRenderPlan(
						testCase.project,
						testCase.panel,
						testCase.workflow,
					)

				if valid ||
					plan != nil {
					t.Fatalf(
						"非法样张参数未被拒绝：%+v",
						plan,
					)
				}
			},
		)
	}
}

func TestResolveCoursewareComicStylePreviewImageSizeRejectsInvalid(
	t *testing.T,
) {
	cases := []struct {
		aspectRatio string
		quality     string
	}{
		{
			aspectRatio: "2:1",
			quality:     models.CWComicImageQualityStandard,
		},
		{
			aspectRatio: models.CWComicAspectRatio16x9,
			quality:     "ultra",
		},
		{
			aspectRatio: "",
			quality:     "",
		},
	}

	for _, testCase := range cases {
		size, valid :=
			resolveCoursewareComicStylePreviewImageSize(
				testCase.aspectRatio,
				testCase.quality,
			)

		if valid ||
			size != "" {
			t.Fatalf(
				"非法尺寸参数未被拒绝：比例=%q，质量=%q，结果=%q",
				testCase.aspectRatio,
				testCase.quality,
				size,
			)
		}
	}
}

func parseCoursewareComicTestImageSize(
	value string,
) (int, int, bool) {
	parts :=
		strings.Split(
			strings.ToLower(
				strings.TrimSpace(
					value,
				),
			),
			"x",
		)

	if len(parts) != 2 {
		return 0, 0, false
	}

	width, widthErr :=
		strconv.Atoi(
			strings.TrimSpace(
				parts[0],
			),
		)

	height, heightErr :=
		strconv.Atoi(
			strings.TrimSpace(
				parts[1],
			),
		)

	if widthErr != nil ||
		heightErr != nil ||
		width <= 0 ||
		height <= 0 {
		return 0, 0, false
	}

	return width, height, true
}
