package handlers

// courseware_assistant_handler_test.go
//
// 本测试只验证HTTP正文防护和路径解析。
// 不连接数据库、不调用AI，也不要求路由已经注册。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// coursewareAssistantHandlerTestRequest 是严格JSON测试协议。
type coursewareAssistantHandlerTestRequest struct {
	Name string `json:"name"`
}

// TestCoursewareAssistantHandlerPathParsing 验证7类接口所需路径。
func TestCoursewareAssistantHandlerPathParsing(
	t *testing.T,
) {
	const coursewareID = "11111111-1111-1111-1111-111111111111"
	const pageID = "22222222-2222-2222-2222-222222222222"
	const slotID = "33333333-3333-3333-3333-333333333333"

	collectionPath :=
		"/api/v1/coursewares/" +
			coursewareID +
			"/assistant-slots"

	if actual :=
		extractCoursewareAssistantSlotCollectionPath(
			collectionPath,
		); actual != coursewareID {
		t.Fatalf(
			"插槽集合路径解析错误: %s",
			actual,
		)
	}

	itemCoursewareID,
		itemSlotID :=
		extractCoursewareAssistantSlotItemPath(
			collectionPath +
				"/" +
				slotID,
		)

	if itemCoursewareID != coursewareID ||
		itemSlotID != slotID {
		t.Fatalf(
			"插槽单项路径解析错误: courseware=%s slot=%s",
			itemCoursewareID,
			itemSlotID,
		)
	}

	actions :=
		[]string{
			"/assistant-slot",
			"/assistant-context",
			"/assistant-plan",
		}

	for _, action := range actions {
		actualCoursewareID,
			actualPageID :=
			extractCoursewareAssistantPageActionPath(
				"/api/v1/coursewares/"+
					coursewareID+
					"/pages/"+
					pageID+
					action,
				action,
			)

		if actualCoursewareID != coursewareID ||
			actualPageID != pageID {
			t.Fatalf(
				"页面动作路径解析错误: action=%s courseware=%s page=%s",
				action,
				actualCoursewareID,
				actualPageID,
			)
		}
	}
}

// TestCoursewareAssistantHandlerRejectsUnknownField 验证未知字段被拒绝。
func TestCoursewareAssistantHandlerRejectsUnknownField(
	t *testing.T,
) {
	recorder :=
		httptest.NewRecorder()
	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/",
			strings.NewReader(
				`{"name":"合法字段","owner_id":"不得接收"}`,
			),
		)

	var target coursewareAssistantHandlerTestRequest

	if decodeCoursewareAssistantJSON(
		recorder,
		request,
		&target,
		1024,
	) {
		t.Fatal(
			"包含未知字段的请求不应解析成功",
		)
	}

	if recorder.Code !=
		http.StatusBadRequest {
		t.Fatalf(
			"未知字段状态码错误: %d",
			recorder.Code,
		)
	}
}

// TestCoursewareAssistantHandlerRejectsMultipleObjects 验证多JSON对象被拒绝。
func TestCoursewareAssistantHandlerRejectsMultipleObjects(
	t *testing.T,
) {
	recorder :=
		httptest.NewRecorder()
	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/",
			strings.NewReader(
				`{"name":"第一份"}{"name":"第二份"}`,
			),
		)

	var target coursewareAssistantHandlerTestRequest

	if decodeCoursewareAssistantJSON(
		recorder,
		request,
		&target,
		1024,
	) {
		t.Fatal(
			"多个JSON对象不应解析成功",
		)
	}

	if recorder.Code !=
		http.StatusBadRequest {
		t.Fatalf(
			"多对象状态码错误: %d",
			recorder.Code,
		)
	}
}

// TestCoursewareAssistantHandlerRejectsOversizedBody 验证正文限长。
func TestCoursewareAssistantHandlerRejectsOversizedBody(
	t *testing.T,
) {
	recorder :=
		httptest.NewRecorder()
	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/",
			strings.NewReader(
				`{"name":"正文超过极小测试上限"}`,
			),
		)

	var target coursewareAssistantHandlerTestRequest

	if decodeCoursewareAssistantJSON(
		recorder,
		request,
		&target,
		8,
	) {
		t.Fatal(
			"超过限制的正文不应解析成功",
		)
	}

	if recorder.Code !=
		http.StatusBadRequest {
		t.Fatalf(
			"超长正文状态码错误: %d",
			recorder.Code,
		)
	}
}

// TestCoursewareAssistantHandlerAcceptsSingleKnownObject 验证合法单对象。
func TestCoursewareAssistantHandlerAcceptsSingleKnownObject(
	t *testing.T,
) {
	recorder :=
		httptest.NewRecorder()
	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/",
			strings.NewReader(
				`{"name":"合法方案"}`,
			),
		)

	var target coursewareAssistantHandlerTestRequest

	if !decodeCoursewareAssistantJSON(
		recorder,
		request,
		&target,
		1024,
	) {
		t.Fatalf(
			"合法JSON不应被拒绝: status=%d body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if target.Name !=
		"合法方案" {
		t.Fatalf(
			"合法字段解析错误: %s",
			target.Name,
		)
	}
}

// TestCoursewareAssistantHandlerRejectsTraversalPath 验证多段和穿透路径。
func TestCoursewareAssistantHandlerRejectsTraversalPath(
	t *testing.T,
) {
	invalidPaths :=
		[]string{
			"/api/v1/coursewares/../assistant-slots",
			"/api/v1/coursewares/cw/pages/../assistant-plan",
			"/api/v1/coursewares/cw/assistant-slots/slot/extra",
		}

	if actual :=
		extractCoursewareAssistantSlotCollectionPath(
			invalidPaths[0],
		); actual != "" {
		t.Fatalf(
			"路径穿透不应解析成功: %s",
			actual,
		)
	}

	if coursewareID, pageID :=
		extractCoursewareAssistantPageActionPath(
			invalidPaths[1],
			"/assistant-plan",
		); coursewareID != "" ||
		pageID != "" {
		t.Fatalf(
			"页面穿透路径不应解析成功: courseware=%s page=%s",
			coursewareID,
			pageID,
		)
	}

	if coursewareID, slotID :=
		extractCoursewareAssistantSlotItemPath(
			invalidPaths[2],
		); coursewareID != "" ||
		slotID != "" {
		t.Fatalf(
			"多段插槽路径不应解析成功: courseware=%s slot=%s",
			coursewareID,
			slotID,
		)
	}
}
