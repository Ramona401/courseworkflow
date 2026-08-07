package handlers

// courseware_comic_workflow_mutation_handler.go
//
// 提供教师五步工作流中的两个同步HTTP入口：
//   - 确认第二步分镜；
//   - 保存第三步视觉设置。
//
// 成功响应返回完整项目详情。
// 项目详情中的project.version是下一次CAS请求必须提交的新版本号，
// project.workflow是本次操作后的最新教师工作流。

import (
	"net/http"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// ConfirmStoryboard 确认当前AI分镜并进入第三步。
//
// POST
// /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/confirm-storyboard
func (h *CoursewareComicHandler) ConfirmStoryboard(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
) {
	if r.Method != http.MethodPost {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持POST请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var request models.ConfirmCoursewareComicStoryboardRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPlanRequestMaxBytes,
	) {
		return
	}

	projectService :=
		h.resolveProjectService()

	_, err :=
		projectService.ConfirmProjectStoryboard(
			r.Context(),
			coursewareID,
			projectID,
			actor,
			&request,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	result, err :=
		projectService.GetProjectDetailForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	workflow, err :=
		projectService.GetProjectWorkflowForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		attachCoursewareComicWorkflowToDetail(
			result,
			workflow,
		); err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		result,
	)
}

// UpdateStyleSettings 保存第三步视觉设置。
//
// PUT
// /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/style-settings
func (h *CoursewareComicHandler) UpdateStyleSettings(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
) {
	if r.Method != http.MethodPut {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持PUT请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var request models.UpdateCoursewareComicStyleSettingsRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPlanRequestMaxBytes,
	) {
		return
	}

	projectService :=
		h.resolveProjectService()

	_, err :=
		projectService.UpdateProjectStyleSettings(
			r.Context(),
			coursewareID,
			projectID,
			actor,
			&request,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	result, err :=
		projectService.GetProjectDetailForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	workflow, err :=
		projectService.GetProjectWorkflowForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		attachCoursewareComicWorkflowToDetail(
			result,
			workflow,
		); err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		result,
	)
}
