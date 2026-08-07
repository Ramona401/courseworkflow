package services

import (
	"errors"
	"testing"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestFindCoursewareComicStylePreviewPanel(
	t *testing.T,
) {
	panels :=
		[]*models.CoursewareComicPanel{
			{
				ID:      "panel-3",
				PanelNo: 3,
			},
			nil,
			{
				ID:      "panel-1",
				PanelNo: 1,
			},
			{
				ID:      "panel-2",
				PanelNo: 2,
			},
		}

	panel, err :=
		findCoursewareComicStylePreviewPanel(
			panels,
		)

	if err != nil {
		t.Fatalf(
			"未找到合法首格：%v",
			err,
		)
	}

	if panel == nil ||
		panel.ID != "panel-1" {
		t.Fatalf(
			"首格识别错误：%+v",
			panel,
		)
	}

	_, err =
		findCoursewareComicStylePreviewPanel(
			[]*models.CoursewareComicPanel{
				{
					ID:      "panel-2",
					PanelNo: 2,
				},
			},
		)

	if !errors.Is(
		err,
		repository.ErrCoursewareComicPanelNotFound,
	) {
		t.Fatalf(
			"缺少首格时错误类型不正确：%v",
			err,
		)
	}
}

func TestValidateCoursewareComicStylePreviewStart(
	t *testing.T,
) {
	confirmedAt :=
		time.Now()

	validProject :=
		func() *models.CoursewareComicProject {
			return &models.CoursewareComicProject{
				ID:          "project-1",
				VisualStyle: models.CWComicVisualModernFlat,
				Status:      models.CWComicProjectStatusPlanned,
				Version:     8,
			}
		}

	validWorkflow :=
		func() *models.CoursewareComicWorkflowState {
			return &models.CoursewareComicWorkflowState{
				ProjectID:             "project-1",
				Stage:                 models.CWComicWorkflowStylePreview,
				StoryboardConfirmedAt: &confirmedAt,
				AspectRatio:           models.CWComicAspectRatio16x9,
				ImageQuality:          models.CWComicImageQualityHigh,
				InsertionMode:         models.CWComicInsertionSinglePage,
			}
		}

	validPanel :=
		func() *models.CoursewareComicPanel {
			return &models.CoursewareComicPanel{
				ID:             "panel-1",
				ProjectID:      "project-1",
				PanelNo:        1,
				VisualPrompt:   "教师讲解知识对象",
				AOCIText:       "完整视觉关系",
				NegativePrompt: "禁止文字",
				Status:         models.CWComicPanelStatusPlanned,
			}
		}

	if err :=
		validateCoursewareComicStylePreviewStart(
			validProject(),
			validWorkflow(),
			validPanel(),
			8,
		); err != nil {
		t.Fatalf(
			"合法样张启动被拒绝：%v",
			err,
		)
	}

	t.Run(
		"project version conflict",
		func(t *testing.T) {
			err :=
				validateCoursewareComicStylePreviewStart(
					validProject(),
					validWorkflow(),
					validPanel(),
					7,
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
		"project status invalid",
		func(t *testing.T) {
			project :=
				validProject()

			project.Status =
				models.CWComicProjectStatusReady

			err :=
				validateCoursewareComicStylePreviewStart(
					project,
					validWorkflow(),
					validPanel(),
					8,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicProjectNotEditable,
			) {
				t.Fatalf(
					"非法项目状态错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"workflow stage invalid",
		func(t *testing.T) {
			workflow :=
				validWorkflow()

			workflow.Stage =
				models.CWComicWorkflowStoryboard

			err :=
				validateCoursewareComicStylePreviewStart(
					validProject(),
					workflow,
					validPanel(),
					8,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicProjectNotEditable,
			) {
				t.Fatalf(
					"非法工作流步骤错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"storyboard not confirmed",
		func(t *testing.T) {
			workflow :=
				validWorkflow()

			workflow.StoryboardConfirmedAt =
				nil

			err :=
				validateCoursewareComicStylePreviewStart(
					validProject(),
					workflow,
					validPanel(),
					8,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicProjectNotEditable,
			) {
				t.Fatalf(
					"未确认分镜错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"style already confirmed",
		func(t *testing.T) {
			workflow :=
				validWorkflow()

			workflow.StyleConfirmedAt =
				&confirmedAt

			err :=
				validateCoursewareComicStylePreviewStart(
					validProject(),
					workflow,
					validPanel(),
					8,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicProjectNotEditable,
			) {
				t.Fatalf(
					"已确认画风错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"panel already generating",
		func(t *testing.T) {
			panel :=
				validPanel()

			panel.Status =
				models.CWComicPanelStatusGenerating

			err :=
				validateCoursewareComicStylePreviewStart(
					validProject(),
					validWorkflow(),
					panel,
					8,
				)

			if !errors.Is(
				err,
				repository.ErrCoursewareComicPanelNotGeneratable,
			) {
				t.Fatalf(
					"重复启动样张错误不正确：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"invalid visual settings",
		func(t *testing.T) {
			project :=
				validProject()

			project.VisualStyle =
				"invalid_style"

			err :=
				validateCoursewareComicStylePreviewStart(
					project,
					validWorkflow(),
					validPanel(),
					8,
				)

			if !errors.Is(
				err,
				ErrCoursewareComicWorkflowInvalidRequest,
			) {
				t.Fatalf(
					"非法视觉设置错误不正确：%v",
					err,
				)
			}
		},
	)
}

func TestValidateCoursewareComicStylePreviewWorkerState(
	t *testing.T,
) {
	confirmedAt :=
		time.Now()

	project :=
		&models.CoursewareComicProject{
			ID:     "project-1",
			Status: models.CWComicProjectStatusPlanned,
		}

	workflow :=
		&models.CoursewareComicWorkflowState{
			ProjectID:             "project-1",
			Stage:                 models.CWComicWorkflowStylePreview,
			StoryboardConfirmedAt: &confirmedAt,
		}

	panel :=
		&models.CoursewareComicPanel{
			ID:        "panel-1",
			ProjectID: "project-1",
			PanelNo:   1,
			Status:    models.CWComicPanelStatusGenerating,
		}

	if err :=
		validateCoursewareComicStylePreviewWorkerState(
			project,
			workflow,
			panel,
		); err != nil {
		t.Fatalf(
			"合法后台工作状态被拒绝：%v",
			err,
		)
	}

	panel.Status =
		models.CWComicPanelStatusGenerated

	err :=
		validateCoursewareComicStylePreviewWorkerState(
			project,
			workflow,
			panel,
		)

	if !errors.Is(
		err,
		repository.ErrCoursewareComicPanelConflict,
	) {
		t.Fatalf(
			"非generating状态错误不正确：%v",
			err,
		)
	}
}

func TestCoursewareComicStylePreviewGenerationSource(
	t *testing.T,
) {
	if actual :=
		coursewareComicStylePreviewGenerationSource(
			nil,
		); actual !=
		models.CWComicVersionSourceInitial {
		t.Fatalf(
			"空分格来源错误：%q",
			actual,
		)
	}

	panel :=
		&models.CoursewareComicPanel{}

	if actual :=
		coursewareComicStylePreviewGenerationSource(
			panel,
		); actual !=
		models.CWComicVersionSourceInitial {
		t.Fatalf(
			"首次生成来源错误：%q",
			actual,
		)
	}

	assetID :=
		"asset-1"

	panel.CurrentAssetID =
		&assetID

	if actual :=
		coursewareComicStylePreviewGenerationSource(
			panel,
		); actual !=
		models.CWComicVersionSourceRegenerate {
		t.Fatalf(
			"重新生成来源错误：%q",
			actual,
		)
	}
}
