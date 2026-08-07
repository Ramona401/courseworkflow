package handlers

// lesson_plan_interaction_subpath_test.go — 上下文17互动保留路径回归测试
//
// 错误HTTP方法落入通配路由普通CRUD分支时，必须在调用Service或解析正文前
// 返回405，不能读取、更新或删除教案。

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLessonPlanCRUDRejectsInteractionSubpathFallback(
	t *testing.T,
) {
	handler := &LessonPlanHandler{}

	testCases := []struct {
		name       string
		method     string
		path       string
		invoke     func(http.ResponseWriter, *http.Request)
		wantStatus int
	}{
		{
			name:   "GET_interact不能回落详情",
			method: http.MethodGet,
			path: "/api/v1/lesson-plans/plans/" +
				"11111111-1111-1111-1111-111111111111/interact",
			invoke:     handler.GetLessonPlan,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "PUT_interact不能回落更新",
			method: http.MethodPut,
			path: "/api/v1/lesson-plans/plans/" +
				"11111111-1111-1111-1111-111111111111/interact",
			invoke:     handler.UpdateLessonPlan,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "DELETE_interactions不能回落删除",
			method: http.MethodDelete,
			path: "/api/v1/lesson-plans/plans/" +
				"11111111-1111-1111-1111-111111111111/interactions/",
			invoke:     handler.DeleteLessonPlan,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(
				testCase.method,
				testCase.path,
				nil,
			)
			response := httptest.NewRecorder()

			testCase.invoke(response, request)

			if response.Code != testCase.wantStatus {
				t.Fatalf(
					"状态码=%d，期望=%d，响应=%s",
					response.Code,
					testCase.wantStatus,
					response.Body.String(),
				)
			}
		})
	}
}
