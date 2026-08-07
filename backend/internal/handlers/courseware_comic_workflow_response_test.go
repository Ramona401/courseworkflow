package handlers

import (
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestAttachCoursewareComicWorkflowResponse(
	t *testing.T,
) {
	t.Run(
		"single project",
		func(t *testing.T) {
			project :=
				&models.CoursewareComicProjectView{
					ID: "project-1",
				}

			workflow :=
				&models.CoursewareComicWorkflowView{
					Stage: models.CWComicWorkflowStoryboard,
				}

			err :=
				attachCoursewareComicWorkflowToProject(
					project,
					workflow,
				)
			if err != nil {
				t.Fatalf(
					"单项目装配失败：%v",
					err,
				)
			}

			if project.Workflow !=
				workflow {
				t.Fatal(
					"单项目工作流没有装配",
				)
			}
		},
	)

	t.Run(
		"project detail",
		func(t *testing.T) {
			detail :=
				&models.CoursewareComicProjectDetailView{
					Project: &models.CoursewareComicProjectView{
						ID: "project-detail",
					},
					Panels: []*models.CoursewareComicPanelView{},
				}

			workflow :=
				&models.CoursewareComicWorkflowView{
					Stage: models.CWComicWorkflowStylePreview,
				}

			err :=
				attachCoursewareComicWorkflowToDetail(
					detail,
					workflow,
				)
			if err != nil {
				t.Fatalf(
					"项目详情装配失败：%v",
					err,
				)
			}

			if detail.Project.Workflow !=
				workflow {
				t.Fatal(
					"项目详情工作流没有装配",
				)
			}
		},
	)

	t.Run(
		"complete project list",
		func(t *testing.T) {
			first :=
				&models.CoursewareComicProjectView{
					ID: "project-1",
				}

			second :=
				&models.CoursewareComicProjectView{
					ID: "project-2",
				}

			firstWorkflow :=
				&models.CoursewareComicWorkflowView{
					Stage: models.CWComicWorkflowStylePreview,
				}

			secondWorkflow :=
				&models.CoursewareComicWorkflowView{
					Stage: models.CWComicWorkflowRefinement,
				}

			result :=
				&models.CoursewareComicProjectListView{
					Projects: []*models.CoursewareComicProjectView{
						first,
						second,
					},
					Total: 2,
				}

			err :=
				attachCoursewareComicWorkflowsToList(
					result,
					map[string]*models.CoursewareComicWorkflowView{
						"project-1": firstWorkflow,
						"project-2": secondWorkflow,
					},
				)
			if err != nil {
				t.Fatalf(
					"列表工作流装配失败：%v",
					err,
				)
			}

			if first.Workflow !=
				firstWorkflow ||
				second.Workflow !=
					secondWorkflow {
				t.Fatal(
					"列表工作流装配结果错误",
				)
			}
		},
	)

	t.Run(
		"missing workflow",
		func(t *testing.T) {
			result :=
				&models.CoursewareComicProjectListView{
					Projects: []*models.CoursewareComicProjectView{
						{
							ID: "project-missing",
						},
					},
					Total: 1,
				}

			err :=
				attachCoursewareComicWorkflowsToList(
					result,
					map[string]*models.CoursewareComicWorkflowView{},
				)

			if !errors.Is(
				err,
				errCoursewareComicWorkflowResponseIncomplete,
			) {
				t.Fatalf(
					"缺少工作流应返回明确错误，实际为：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"empty list",
		func(t *testing.T) {
			err :=
				attachCoursewareComicWorkflowsToList(
					&models.CoursewareComicProjectListView{
						Projects: []*models.CoursewareComicProjectView{},
						Total:    0,
					},
					nil,
				)

			if err != nil {
				t.Fatalf(
					"空项目列表不应失败：%v",
					err,
				)
			}
		},
	)

	t.Run(
		"nil inputs",
		func(t *testing.T) {
			err :=
				attachCoursewareComicWorkflowToProject(
					nil,
					nil,
				)

			if !errors.Is(
				err,
				errCoursewareComicWorkflowResponseIncomplete,
			) {
				t.Fatal(
					"空项目应返回工作流响应错误",
				)
			}

			err =
				attachCoursewareComicWorkflowToDetail(
					nil,
					nil,
				)

			if !errors.Is(
				err,
				errCoursewareComicWorkflowResponseIncomplete,
			) {
				t.Fatal(
					"空项目详情应返回工作流响应错误",
				)
			}

			err =
				attachCoursewareComicWorkflowsToList(
					nil,
					nil,
				)

			if !errors.Is(
				err,
				errCoursewareComicWorkflowResponseIncomplete,
			) {
				t.Fatal(
					"空项目列表应返回工作流响应错误",
				)
			}
		},
	)
}
