package routes

// routes_courseware_assistant_test.go
//
// 只验证不连接数据库的教师端教学智能体路径匹配、尾斜杠兼容、
// 非本模块路径透传、非法保留路径和教师端功能关闭状态。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"tedna/internal/models"
)

func TestCoursewareAssistantRouteMatching(
	t *testing.T,
) {
	cases := []struct {
		name         string
		path         string
		kind         string
		coursewareID string
		pageID       string
		slotID       string
		deploymentID string
		matched      bool
	}{
		{
			name:         "插槽列表",
			path:         "/api/v1/coursewares/cw-1/assistant-slots",
			kind:         coursewareAssistantRouteSlots,
			coursewareID: "cw-1",
			matched:      true,
		},
		{
			name:         "插槽列表尾斜杠",
			path:         "/api/v1/coursewares/cw-1/assistant-slots/",
			kind:         coursewareAssistantRouteSlots,
			coursewareID: "cw-1",
			matched:      true,
		},
		{
			name:         "插槽单项",
			path:         "/api/v1/coursewares/cw-1/assistant-slots/slot-1",
			kind:         coursewareAssistantRouteSlotItem,
			coursewareID: "cw-1",
			slotID:       "slot-1",
			matched:      true,
		},
		{
			name:         "页面插槽",
			path:         "/api/v1/coursewares/cw-1/pages/page-1/assistant-slot",
			kind:         coursewareAssistantRoutePageSlot,
			coursewareID: "cw-1",
			pageID:       "page-1",
			matched:      true,
		},
		{
			name:         "上下文预览",
			path:         "/api/v1/coursewares/cw-1/pages/page-1/assistant-context",
			kind:         coursewareAssistantRoutePageContext,
			coursewareID: "cw-1",
			pageID:       "page-1",
			matched:      true,
		},
		{
			name:         "方案生成",
			path:         "/api/v1/coursewares/cw-1/pages/page-1/assistant-plan",
			kind:         coursewareAssistantRoutePagePlan,
			coursewareID: "cw-1",
			pageID:       "page-1",
			matched:      true,
		},
		{
			name:         "首次发布",
			path:         "/api/v1/coursewares/cw-1/pages/page-1/assistant-deployment",
			kind:         coursewareAssistantRoutePublishDeployment,
			coursewareID: "cw-1",
			pageID:       "page-1",
			matched:      true,
		},
		{
			name:         "部署列表",
			path:         "/api/v1/coursewares/cw-1/assistant-deployments",
			kind:         coursewareAssistantRouteDeployments,
			coursewareID: "cw-1",
			matched:      true,
		},
		{
			name:         "版本入口",
			path:         "/api/v1/assistant-deployments/deploy-1/versions",
			kind:         coursewareAssistantRouteVersions,
			deploymentID: "deploy-1",
			matched:      true,
		},
		{
			name:         "暂停入口",
			path:         "/api/v1/assistant-deployments/deploy-1/pause",
			kind:         coursewareAssistantRoutePause,
			deploymentID: "deploy-1",
			matched:      true,
		},
		{
			name:         "恢复入口",
			path:         "/api/v1/assistant-deployments/deploy-1/resume",
			kind:         coursewareAssistantRouteResume,
			deploymentID: "deploy-1",
			matched:      true,
		},
		{
			name:         "撤销入口",
			path:         "/api/v1/assistant-deployments/deploy-1/revoke",
			kind:         coursewareAssistantRouteRevoke,
			deploymentID: "deploy-1",
			matched:      true,
		},
		{
			name:         "策略入口",
			path:         "/api/v1/assistant-deployments/deploy-1/policy",
			kind:         coursewareAssistantRoutePolicy,
			deploymentID: "deploy-1",
			matched:      true,
		},
		{
			name:         "教师预览入口",
			path:         "/api/v1/assistant-deployments/deploy-1/preview-session",
			kind:         coursewareAssistantRoutePreviewSession,
			deploymentID: "deploy-1",
			matched:      true,
		},
		{
			name:         "教师豆包朗读入口",
			path:         "/api/v1/assistant-deployments/deploy-1/tts",
			kind:         coursewareAssistantRouteTTS,
			deploymentID: "deploy-1",
			matched:      true,
		},
		{
			name:    "非法多余路径",
			path:    "/api/v1/coursewares/cw-1/pages/page-1/assistant-plan/extra",
			kind:    coursewareAssistantRouteInvalid,
			matched: true,
		},
		{
			name:    "非法部署根路径",
			path:    "/api/v1/assistant-deployments",
			kind:    coursewareAssistantRouteInvalid,
			matched: true,
		},
		{
			name:    "普通课件路径不拦截",
			path:    "/api/v1/coursewares/cw-1/pages",
			matched: false,
		},
		{
			name:    "健康检查不拦截",
			path:    "/api/v1/health",
			matched: false,
		},
	}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				actual :=
					matchCoursewareAssistantRoute(
						item.path,
					)

				if actual.Kind != item.kind ||
					actual.CoursewareID != item.coursewareID ||
					actual.PageID != item.pageID ||
					actual.SlotID != item.slotID ||
					actual.DeploymentID != item.deploymentID ||
					actual.Matched != item.matched {
					t.Fatalf(
						"路由匹配错误：path=%s result=%+v",
						item.path,
						actual,
					)
				}
			},
		)
	}
}

func TestCoursewareAssistantRouteWrapperPassesThrough(
	t *testing.T,
) {
	baseCalled := false
	authenticatedCalled := false

	base := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			baseCalled = true
			w.WriteHeader(http.StatusAccepted)
		},
	)

	authenticated := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			authenticatedCalled = true
			w.WriteHeader(http.StatusCreated)
		},
	)

	handler :=
		buildCoursewareAssistantRouteHandler(
			base,
			authenticated,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/health",
			nil,
		)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusAccepted ||
		!baseCalled ||
		authenticatedCalled {
		t.Fatalf(
			"非教学智能体路径透传错误：status=%d base=%t auth=%t",
			recorder.Code,
			baseCalled,
			authenticatedCalled,
		)
	}
}

func TestCoursewareAssistantRouteWrapperUsesAuthenticatedRoute(
	t *testing.T,
) {
	baseCalled := false
	authenticatedCalled := false

	base := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			baseCalled = true
			w.WriteHeader(http.StatusAccepted)
		},
	)

	authenticated := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			authenticatedCalled = true
			w.WriteHeader(http.StatusCreated)
		},
	)

	handler :=
		buildCoursewareAssistantRouteHandler(
			base,
			authenticated,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/coursewares/cw-1/assistant-slots",
			nil,
		)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusCreated ||
		baseCalled ||
		!authenticatedCalled {
		t.Fatalf(
			"教学智能体路径分发错误：status=%d base=%t auth=%t",
			recorder.Code,
			baseCalled,
			authenticatedCalled,
		)
	}
}

func TestCoursewareAssistantInvalidReservedPathReturnsNotFound(
	t *testing.T,
) {
	baseCalled := false
	authenticatedCalled := false

	base := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			baseCalled = true
			w.WriteHeader(http.StatusAccepted)
		},
	)

	authenticated := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			authenticatedCalled = true
			w.WriteHeader(http.StatusCreated)
		},
	)

	handler :=
		buildCoursewareAssistantRouteHandler(
			base,
			authenticated,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/coursewares/cw-1/pages/page-1/assistant-plan/extra",
			nil,
		)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusNotFound ||
		baseCalled ||
		authenticatedCalled {
		t.Fatalf(
			"非法保留路径处理错误：status=%d base=%t auth=%t",
			recorder.Code,
			baseCalled,
			authenticatedCalled,
		)
	}
}

func TestCoursewareAssistantDisabledRouteBlocksAssistantPaths(
	t *testing.T,
) {
	cases := []string{
		"/api/v1/coursewares/cw-1/assistant-slots",
		"/api/v1/coursewares/cw-1/pages/page-1/assistant-plan",
		"/api/v1/assistant-deployments/deploy-1/preview-session",
		"/api/v1/assistant-deployments/deploy-1/tts",
		"/api/v1/coursewares/cw-1/pages/page-1/assistant-unknown",
	}

	for _, path := range cases {
		t.Run(
			path,
			func(t *testing.T) {
				baseCalled := false

				base := http.HandlerFunc(
					func(
						w http.ResponseWriter,
						_ *http.Request,
					) {
						baseCalled = true
						w.WriteHeader(
							http.StatusAccepted,
						)
					},
				)

				handler :=
					buildCoursewareAssistantDisabledRouteHandler(
						base,
					)

				request :=
					httptest.NewRequest(
						http.MethodGet,
						path,
						nil,
					)

				recorder :=
					httptest.NewRecorder()

				handler.ServeHTTP(
					recorder,
					request,
				)

				if recorder.Code != http.StatusNotFound ||
					baseCalled {
					t.Fatalf(
						"关闭态没有阻断教学智能体路径：path=%s status=%d base=%t",
						path,
						recorder.Code,
						baseCalled,
					)
				}
			},
		)
	}
}

func TestCoursewareAssistantDisabledRoutePassesThroughUnrelatedPath(
	t *testing.T,
) {
	baseCalled := false

	base := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			baseCalled = true
			w.WriteHeader(http.StatusAccepted)
		},
	)

	handler :=
		buildCoursewareAssistantDisabledRouteHandler(
			base,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/coursewares/cw-1/pages/page-1/save-html",
			nil,
		)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusAccepted ||
		!baseCalled {
		t.Fatalf(
			"关闭态错误拦截普通课件路径：status=%d base=%t",
			recorder.Code,
			baseCalled,
		)
	}
}

func TestCoursewareAssistantPlanSceneRegistered(
	t *testing.T,
) {
	if !models.IsValidSceneCode(
		models.SceneCoursewareAssistantPlan,
	) {
		t.Fatal(
			"课件教学智能体方案场景没有进入有效场景白名单",
		)
	}

	if models.SceneNameMap[models.SceneCoursewareAssistantPlan] == "" {
		t.Fatal(
			"课件教学智能体方案场景缺少管理端名称",
		)
	}

	if models.SceneGroupMap[models.SceneCoursewareAssistantPlan] != "courseware" {
		t.Fatalf(
			"课件教学智能体方案场景分组错误：%s",
			models.SceneGroupMap[models.SceneCoursewareAssistantPlan],
		)
	}
}
