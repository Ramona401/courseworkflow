package handlers

// courseware_comic_workflow_response.go — 漫画项目工作流响应装配
//
// 项目主体视图和教师工作流分别从独立仓储读取，
// 在HTTP响应前才进行浏览器安全装配。
//
// 数据库迁移后，每一个漫画项目都必须拥有工作流状态。
// 如果装配阶段缺失工作流，HTTP层应返回错误，
// 不能静默返回workflow:null并让前端错误回退到第一步。

import (
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
)

var errCoursewareComicWorkflowResponseIncomplete = errors.New(
	"知识点漫画工作流响应不完整",
)

func attachCoursewareComicWorkflowToProject(
	project *models.CoursewareComicProjectView,
	workflow *models.CoursewareComicWorkflowView,
) error {
	if project == nil {
		return fmt.Errorf(
			"%w：项目视图为空",
			errCoursewareComicWorkflowResponseIncomplete,
		)
	}

	if strings.TrimSpace(
		project.ID,
	) == "" {
		return fmt.Errorf(
			"%w：项目ID为空",
			errCoursewareComicWorkflowResponseIncomplete,
		)
	}

	if workflow == nil {
		return fmt.Errorf(
			"%w：项目%s缺少工作流",
			errCoursewareComicWorkflowResponseIncomplete,
			project.ID,
		)
	}

	project.Workflow = workflow

	return nil
}

func attachCoursewareComicWorkflowToDetail(
	result *models.CoursewareComicProjectDetailView,
	workflow *models.CoursewareComicWorkflowView,
) error {
	if result == nil {
		return fmt.Errorf(
			"%w：项目详情为空",
			errCoursewareComicWorkflowResponseIncomplete,
		)
	}

	return attachCoursewareComicWorkflowToProject(
		result.Project,
		workflow,
	)
}

func attachCoursewareComicWorkflowsToList(
	result *models.CoursewareComicProjectListView,
	workflows map[string]*models.CoursewareComicWorkflowView,
) error {
	if result == nil {
		return fmt.Errorf(
			"%w：项目列表为空",
			errCoursewareComicWorkflowResponseIncomplete,
		)
	}

	for _, project := range result.Projects {
		if project == nil {
			return fmt.Errorf(
				"%w：项目列表包含空项目",
				errCoursewareComicWorkflowResponseIncomplete,
			)
		}

		workflow :=
			workflows[project.ID]

		if err :=
			attachCoursewareComicWorkflowToProject(
				project,
				workflow,
			); err != nil {
			return err
		}
	}

	return nil
}
