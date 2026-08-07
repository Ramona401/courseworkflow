package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestBuildCoursewareComicConfirmedPanelRenderPlan(
	t *testing.T,
) {
	confirmedAt :=
		time.Now()

	project :=
		&models.CoursewareComicProject{
			ID:                 "project-1",
			VisualStyle:        models.CWComicVisualCinematic3D,
			StyleAOCIText:      "旧规划风格：简笔平面图",
			CharacterBibleJSON: `{"characters":[{"id":"student","name":"学生"}]}`,
		}

	panel :=
		&models.CoursewareComicPanel{
			ProjectID:      "project-1",
			PanelNo:        2,
			VisualPrompt:   "学生观察知识对象",
			AOCIText:       "完整视觉关系",
			NegativePrompt: "禁止文字和水印",
		}

	previewPanelID :=
		"panel-1"

	workflow :=
		&models.CoursewareComicWorkflowState{
			ProjectID:             "project-1",
			Stage:                 models.CWComicWorkflowBatchGeneration,
			StoryboardConfirmedAt: &confirmedAt,
			StyleConfirmedAt:      &confirmedAt,
			StylePreviewPanelID:   &previewPanelID,
			AspectRatio:           models.CWComicAspectRatio4x3,
			ImageQuality:          models.CWComicImageQualityHigh,
			InsertionMode:         models.CWComicInsertionSinglePage,
			StyleInstruction:      "使用蓝紫色自然光，人物保持真实材质",
		}

	plan, valid :=
		buildCoursewareComicConfirmedPanelRenderPlan(
			project,
			panel,
			workflow,
		)

	if !valid ||
		plan == nil {
		t.Fatal(
			"合法确认画风未生成整批渲染计划",
		)
	}

	if plan.ImageSize !=
		"3072x2304" {
		t.Fatalf(
			"整批分格尺寸错误：%q",
			plan.ImageSize,
		)
	}

	required :=
		[]string{
			"电影级三维动画风格",
			"横向4:3经典教学画幅",
			"高清精细质量",
			"使用蓝紫色自然光",
			"完整漫画底图要求",
			"不得绘制气泡",
			"不得生成任何文字",
		}

	for _, value := range required {
		if !strings.Contains(
			plan.Prompt,
			value,
		) {
			t.Fatalf(
				"整批提示词缺少：%q",
				value,
			)
		}
	}

	if strings.Contains(
		plan.Prompt,
		"本次只生成第1格完整视觉样张",
	) {
		t.Fatal(
			"整批生成提示词不应包含样张专用语句",
		)
	}

	oldIndex :=
		strings.Index(
			plan.Prompt,
			"旧规划风格：简笔平面图",
		)

	confirmedIndex :=
		strings.Index(
			plan.Prompt,
			"【教师已确认画风】",
		)

	if oldIndex < 0 ||
		confirmedIndex <= oldIndex {
		t.Fatal(
			"确认画风必须位于旧规划风格之后",
		)
	}
}

func TestBuildCoursewareComicConfirmedCharacterSheetRenderPlan(
	t *testing.T,
) {
	project :=
		&models.CoursewareComicProject{
			VisualStyle:        models.CWComicVisualWarmStorybook,
			CharacterBibleJSON: `{"characters":[{"id":"child","name":"儿童"}]}`,
		}

	workflow :=
		&models.CoursewareComicWorkflowState{
			ProjectID:        "project-1",
			Stage:            models.CWComicWorkflowBatchGeneration,
			AspectRatio:      models.CWComicAspectRatio9x16,
			ImageQuality:     models.CWComicImageQualityHigh,
			InsertionMode:    models.CWComicInsertionSinglePage,
			StyleInstruction: "柔和纸张肌理，表情自然",
		}

	plan, valid :=
		buildCoursewareComicConfirmedCharacterSheetRenderPlan(
			project,
			workflow,
		)

	if !valid ||
		plan == nil {
		t.Fatal(
			"合法人物设定图参数被拒绝",
		)
	}

	if plan.ImageSize !=
		"3200x1800" {
		t.Fatalf(
			"人物设定图应固定为高清16:9：%q",
			plan.ImageSize,
		)
	}

	required :=
		[]string{
			"温暖教育绘本风格",
			"横向16:9人物设定参考图",
			"柔和纸张肌理",
			"不得生成姓名",
		}

	for _, value := range required {
		if !strings.Contains(
			plan.Prompt,
			value,
		) {
			t.Fatalf(
				"人物设定图提示词缺少：%q",
				value,
			)
		}
	}
}

func TestValidateCoursewareComicBatchGenerationStart(
	t *testing.T,
) {
	confirmedAt :=
		time.Now()

	previewPanelID :=
		"panel-1"

	validProject :=
		func() *models.CoursewareComicProject {
			return &models.CoursewareComicProject{
				ID:          "project-1",
				VisualStyle: models.CWComicVisualModernFlat,
				Status:      models.CWComicProjectStatusPlanned,
				Version:     12,
			}
		}

	validWorkflow :=
		func() *models.CoursewareComicWorkflowState {
			return &models.CoursewareComicWorkflowState{
				ProjectID:             "project-1",
				Stage:                 models.CWComicWorkflowBatchGeneration,
				StoryboardConfirmedAt: &confirmedAt,
				StyleConfirmedAt:      &confirmedAt,
				StylePreviewPanelID:   &previewPanelID,
				AspectRatio:           models.CWComicAspectRatio16x9,
				ImageQuality:          models.CWComicImageQualityHigh,
				InsertionMode:         models.CWComicInsertionSinglePage,
			}
		}

	if err :=
		validateCoursewareComicBatchGenerationStart(
			validProject(),
			validWorkflow(),
			12,
		); err != nil {
		t.Fatalf(
			"合法整批生成启动被拒绝：%v",
			err,
		)
	}

	failedProject :=
		validProject()

	failedProject.Status =
		models.CWComicProjectStatusFailed

	if err :=
		validateCoursewareComicBatchGenerationStart(
			failedProject,
			validWorkflow(),
			12,
		); err != nil {
		t.Fatalf(
			"失败项目恢复生成被拒绝：%v",
			err,
		)
	}

	t.Run(
		"version conflict",
		func(t *testing.T) {
			err :=
				validateCoursewareComicBatchGenerationStart(
					validProject(),
					validWorkflow(),
					11,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicProjectConflict,
			) {
				t.Fatalf(
					"项目版本冲突错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"preview not confirmed",
		func(t *testing.T) {
			workflow :=
				validWorkflow()

			workflow.StyleConfirmedAt =
				nil

			err :=
				validateCoursewareComicBatchGenerationStart(
					validProject(),
					workflow,
					12,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicProjectNotEditable,
			) {
				t.Fatalf(
					"未确认样张错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"wrong stage",
		func(t *testing.T) {
			workflow :=
				validWorkflow()

			workflow.Stage =
				models.CWComicWorkflowStylePreview

			err :=
				validateCoursewareComicBatchGenerationStart(
					validProject(),
					workflow,
					12,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicProjectNotEditable,
			) {
				t.Fatalf(
					"错误工作流步骤错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"invalid visual selection",
		func(t *testing.T) {
			project :=
				validProject()

			project.VisualStyle =
				"invalid_style"

			err :=
				validateCoursewareComicBatchGenerationStart(
					project,
					validWorkflow(),
					12,
				)

			if !errors.Is(
				err,
				ErrCoursewareComicWorkflowInvalidRequest,
			) {
				t.Fatalf(
					"非法确认画风错误不正确：%v",
					err,
				)
			}
		},
	)
}

func TestValidateCoursewareComicBatchGenerationWorkerState(
	t *testing.T,
) {
	confirmedAt :=
		time.Now()

	previewPanelID :=
		"panel-1"

	project :=
		&models.CoursewareComicProject{
			ID:          "project-1",
			VisualStyle: models.CWComicVisualScienceEncyclopedia,
			Status:      models.CWComicProjectStatusGenerating,
		}

	workflow :=
		&models.CoursewareComicWorkflowState{
			ProjectID:             "project-1",
			Stage:                 models.CWComicWorkflowBatchGeneration,
			StoryboardConfirmedAt: &confirmedAt,
			StyleConfirmedAt:      &confirmedAt,
			StylePreviewPanelID:   &previewPanelID,
			AspectRatio:           models.CWComicAspectRatioCourseware,
			ImageQuality:          models.CWComicImageQualityStandard,
			InsertionMode:         models.CWComicInsertionSinglePage,
		}

	if err :=
		validateCoursewareComicBatchGenerationWorkerState(
			project,
			workflow,
		); err != nil {
		t.Fatalf(
			"合法整批后台状态被拒绝：%v",
			err,
		)
	}

	project.Status =
		models.CWComicProjectStatusReady

	err :=
		validateCoursewareComicBatchGenerationWorkerState(
			project,
			workflow,
		)

	if !errors.Is(
		err,
		repository.ErrCoursewareComicProjectNotEditable,
	) {
		t.Fatalf(
			"非generating项目错误不正确：%v",
			err,
		)
	}
}
