package handlers

// courseware_assistant_response.go
//
// 本文件限定教师端教学智能体接口能够返回的成功响应类型。
//
// 只接受浏览器安全模型：
//   - CoursewareAssistantSlotView；
//   - CoursewareAssistantSlotListResponse；
//   - CoursewareAssistantContextPreview；
//   - CoursewareAssistantPlanResult。
//
// 不接受部署版本内部快照、助手完整实体、页面HTML或教案全文。

import (
	"net/http"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// coursewareAssistantDeleteResponse 是插槽删除成功响应。
type coursewareAssistantDeleteResponse struct {
	Message string `json:"message"`
}

// writeCoursewareAssistantSlotResponse 返回单个安全插槽。
func writeCoursewareAssistantSlotResponse(
	w http.ResponseWriter,
	response *models.CoursewareAssistantSlotView,
) {
	utils.Success(
		w,
		response,
	)
}

// writeCoursewareAssistantSlotListResponse 返回安全插槽列表。
func writeCoursewareAssistantSlotListResponse(
	w http.ResponseWriter,
	response *models.CoursewareAssistantSlotListResponse,
) {
	utils.Success(
		w,
		response,
	)
}

// writeCoursewareAssistantContextResponse 返回受限上下文预览。
func writeCoursewareAssistantContextResponse(
	w http.ResponseWriter,
	response *models.CoursewareAssistantContextPreview,
) {
	utils.Success(
		w,
		response,
	)
}

// writeCoursewareAssistantPlanResponse 返回尚未保存的可编辑方案。
func writeCoursewareAssistantPlanResponse(
	w http.ResponseWriter,
	response *models.CoursewareAssistantPlanResult,
) {
	utils.Success(
		w,
		response,
	)
}

// writeCoursewareAssistantDeleteResponse 返回删除确认。
func writeCoursewareAssistantDeleteResponse(
	w http.ResponseWriter,
) {
	utils.Success(
		w,
		&coursewareAssistantDeleteResponse{
			Message: "课件教学智能体插槽已删除",
		},
	)
}
