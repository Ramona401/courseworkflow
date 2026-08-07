package handlers

// courseware_comic_storyboard_edit_handler.go — 第二步单格分镜编辑入口
//
// 浏览器只能提交教师可见的教学分镜字段和 panel.version。
// 图片提示词、负面提示词、IAOCI、内部图片键和关系索引不进入请求协议。

import (
	"net/http"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// UpdatePanelStoryboard 保存第二步中的单格教学分镜。
//
// PUT
// /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/panels/{panel_id}/storyboard
func (h *CoursewareComicHandler) UpdatePanelStoryboard(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
	panelID string,
) {
	if r.Method != http.MethodPut {
		coursewareComicMethodNotAllowed(w, "仅支持PUT请求")
		return
	}

	actor, ok := authorizeCoursewareComicActor(w, r, coursewareID)
	if !ok {
		return
	}

	var request models.UpdateCoursewareComicStoryboardPanelRequest
	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPlanRequestMaxBytes,
	) {
		return
	}

	result, err := h.resolveProjectService().UpdatePanelStoryboard(
		r.Context(),
		coursewareID,
		projectID,
		panelID,
		actor,
		&request,
	)
	if err != nil {
		writeCoursewareComicHandlerError(w, err)
		return
	}

	utils.Success(w, result)
}
