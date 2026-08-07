package services

import (
	"testing"
	"time"

	"tedna/internal/models"
)

func TestBuildCoursewareComicWorkflowView(
	t *testing.T,
) {
	now :=
		time.Date(
			2026,
			time.July,
			27,
			23,
			59,
			0,
			0,
			time.UTC,
		)

	panelID :=
		"panel-1"

	view, err :=
		buildCoursewareComicWorkflowView(
			&models.CoursewareComicWorkflowState{
				ProjectID:             "project-1",
				Stage:                 models.CWComicWorkflowStylePreview,
				StoryboardConfirmedAt: &now,
				StylePreviewPanelID:   &panelID,
				AspectRatio:           models.CWComicAspectRatio16x9,
				ImageQuality:          models.CWComicImageQualityHigh,
				InsertionMode:         models.CWComicInsertionSinglePage,
				StyleInstruction:      "  颜色更明亮，人物更像中学生  ",
			},
		)
	if err != nil {
		t.Fatalf(
			"构建浏览器工作流失败：%v",
			err,
		)
	}

	if view.Stage !=
		models.CWComicWorkflowStylePreview {
		t.Fatalf(
			"工作流步骤错误：%q",
			view.Stage,
		)
	}

	if view.AspectRatio !=
		models.CWComicAspectRatio16x9 {
		t.Fatalf(
			"图片比例错误：%q",
			view.AspectRatio,
		)
	}

	if view.ImageQuality !=
		models.CWComicImageQualityHigh {
		t.Fatalf(
			"图片清晰度错误：%q",
			view.ImageQuality,
		)
	}

	if view.InsertionMode !=
		models.CWComicInsertionSinglePage {
		t.Fatalf(
			"使用方式错误：%q",
			view.InsertionMode,
		)
	}

	if view.StylePreviewPanelID == nil ||
		*view.StylePreviewPanelID !=
			panelID {
		t.Fatal(
			"首格样张定位丢失",
		)
	}

	if view.StoryboardConfirmedAt == nil {
		t.Fatal(
			"分镜确认时间丢失",
		)
	}

	if view.StyleInstruction !=
		"颜色更明亮，人物更像中学生" {
		t.Fatalf(
			"风格补充要求错误：%q",
			view.StyleInstruction,
		)
	}

	defaultView, err :=
		buildCoursewareComicWorkflowView(
			&models.CoursewareComicWorkflowState{
				ProjectID: "project-2",
			},
		)
	if err != nil {
		t.Fatalf(
			"默认工作流构建失败：%v",
			err,
		)
	}

	if defaultView.Stage !=
		models.CWComicWorkflowSource {
		t.Fatalf(
			"默认工作流步骤错误：%q",
			defaultView.Stage,
		)
	}

	if defaultView.AspectRatio !=
		models.CWComicAspectRatioCourseware {
		t.Fatalf(
			"默认图片比例错误：%q",
			defaultView.AspectRatio,
		)
	}

	if defaultView.ImageQuality !=
		models.CWComicImageQualityHigh {
		t.Fatalf(
			"默认图片清晰度错误：%q",
			defaultView.ImageQuality,
		)
	}

	if defaultView.InsertionMode !=
		models.CWComicInsertionSinglePage {
		t.Fatalf(
			"默认使用方式错误：%q",
			defaultView.InsertionMode,
		)
	}

	if defaultView.StyleInstruction != "" {
		t.Fatalf(
			"默认风格补充要求应为空：%q",
			defaultView.StyleInstruction,
		)
	}

	if _, err :=
		buildCoursewareComicWorkflowView(
			nil,
		); err == nil {
		t.Fatal(
			"空工作流状态不应通过",
		)
	}

	if _, err :=
		buildCoursewareComicWorkflowView(
			&models.CoursewareComicWorkflowState{
				ProjectID: "project-invalid",
				Stage:     "invalid-stage",
			},
		); err == nil {
		t.Fatal(
			"非法工作流状态不应通过",
		)
	}
}
