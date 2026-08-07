package routes

// routes_courseware_comic_test.go
//
// 验证知识点漫画项目、参考资源、五步工作流、生成、插页、
// 覆盖层编辑、单格重画和页面同步路径。
//
// 安全边界：
//   - 图片提示词与IAOCI编辑接口已经关闭；
//   - /panels/{panel_id}/prompt属于漫画保留路径，但必须被识别为非法路径；
//   - 参考资源只开放集合GET/POST和单项DELETE；
//   - 非漫画路径继续透传既有主路由；
//   - 非法漫画保留路径统一返回404，不能下沉到其他处理器。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"tedna/internal/models"
)

func TestCoursewareComicRouteMatching(
	t *testing.T,
) {
	cases := []struct {
		name         string
		path         string
		kind         string
		coursewareID string
		projectID    string
		panelID      string
		referenceID  string
		matched      bool
	}{
		{
			name:         "项目列表",
			path:         "/api/v1/coursewares/cw-1/comic-projects",
			kind:         coursewareComicRouteProjects,
			coursewareID: "cw-1",
			matched:      true,
		},
		{
			name:         "项目详情",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1",
			kind:         coursewareComicRouteProject,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "AI规划",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/plan",
			kind:         coursewareComicRoutePlan,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "参考资源集合",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/references",
			kind:         coursewareComicRouteReferences,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "参考资源单项",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/references/reference-1",
			kind:         coursewareComicRouteReference,
			coursewareID: "cw-1",
			projectID:    "project-1",
			referenceID:  "reference-1",
			matched:      true,
		},
		{
			name:         "确认分镜",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/confirm-storyboard",
			kind:         coursewareComicRouteConfirmStoryboard,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "保存视觉设置",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/style-settings",
			kind:         coursewareComicRouteStyleSettings,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "生成首格样张",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/generate-style-preview",
			kind:         coursewareComicRouteGenerateStylePreview,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "确认首格样张",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/confirm-style-preview",
			kind:         coursewareComicRouteConfirmStylePreview,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "整批图片生成",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/generate",
			kind:         coursewareComicRouteGenerate,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "插入课件页面",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/insert-page",
			kind:         coursewareComicRouteInsertPage,
			coursewareID: "cw-1",
			projectID:    "project-1",
			matched:      true,
		},
		{
			name:         "保存覆盖层",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/panels/panel-1/overlay",
			kind:         coursewareComicRouteOverlay,
			coursewareID: "cw-1",
			projectID:    "project-1",
			panelID:      "panel-1",
			matched:      true,
		},
		{
			name:    "图片提示词与IAOCI接口已经关闭",
			path:    "/api/v1/coursewares/cw-1/comic-projects/project-1/panels/panel-1/prompt",
			kind:    coursewareComicRouteInvalid,
			matched: true,
		},
		{
			name:         "单格重新生成",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/panels/panel-1/regenerate",
			kind:         coursewareComicRouteRegenerate,
			coursewareID: "cw-1",
			projectID:    "project-1",
			panelID:      "panel-1",
			matched:      true,
		},
		{
			name:         "同步单格到页面",
			path:         "/api/v1/coursewares/cw-1/comic-projects/project-1/panels/panel-1/sync-page",
			kind:         coursewareComicRouteSyncPage,
			coursewareID: "cw-1",
			projectID:    "project-1",
			panelID:      "panel-1",
			matched:      true,
		},
		{
			name:    "非法项目动作",
			path:    "/api/v1/coursewares/cw-1/comic-projects/project-1/delete",
			kind:    coursewareComicRouteInvalid,
			matched: true,
		},
		{
			name:    "非法参考资源深层动作",
			path:    "/api/v1/coursewares/cw-1/comic-projects/project-1/references/reference-1/delete",
			kind:    coursewareComicRouteInvalid,
			matched: true,
		},
		{
			name:    "非法漫画格动作",
			path:    "/api/v1/coursewares/cw-1/comic-projects/project-1/panels/panel-1/delete",
			kind:    coursewareComicRouteInvalid,
			matched: true,
		},
		{
			name:    "普通课件页面路径",
			path:    "/api/v1/coursewares/cw-1/pages",
			matched: false,
		},
		{
			name:    "健康检查",
			path:    "/api/v1/health",
			matched: false,
		},
	}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				actual :=
					matchCoursewareComicRoute(
						item.path,
					)

				if actual.Kind !=
					item.kind ||
					actual.CoursewareID !=
						item.coursewareID ||
					actual.ProjectID !=
						item.projectID ||
					actual.PanelID !=
						item.panelID ||
					actual.ReferenceID !=
						item.referenceID ||
					actual.Matched !=
						item.matched {
					t.Fatalf(
						"漫画路由匹配错误: path=%s result=%+v",
						item.path,
						actual,
					)
				}
			},
		)
	}
}

func TestCoursewareComicRouteWrapperPassesThrough(
	t *testing.T,
) {
	baseCalled := false
	authenticatedCalled := false

	base :=
		http.HandlerFunc(
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

	authenticated :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				authenticatedCalled = true

				w.WriteHeader(
					http.StatusCreated,
				)
			},
		)

	handler :=
		buildCoursewareComicRouteHandler(
			base,
			authenticated,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/health",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusAccepted ||
		!baseCalled ||
		authenticatedCalled {
		t.Fatalf(
			"非漫画路径透传错误: status=%d base=%t auth=%t",
			recorder.Code,
			baseCalled,
			authenticatedCalled,
		)
	}
}

func TestCoursewareComicRouteWrapperUsesAuthenticatedRoute(
	t *testing.T,
) {
	baseCalled := false
	authenticatedCalled := false

	base :=
		http.HandlerFunc(
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

	authenticated :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				authenticatedCalled = true

				w.WriteHeader(
					http.StatusCreated,
				)
			},
		)

	handler :=
		buildCoursewareComicRouteHandler(
			base,
			authenticated,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/coursewares/cw-1/comic-projects/project-1/references",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusCreated ||
		baseCalled ||
		!authenticatedCalled {
		t.Fatalf(
			"漫画路径认证分发错误: status=%d base=%t auth=%t",
			recorder.Code,
			baseCalled,
			authenticatedCalled,
		)
	}
}

func TestCoursewareComicInvalidReservedPathReturnsNotFound(
	t *testing.T,
) {
	baseCalled := false
	authenticatedCalled := false

	base :=
		http.HandlerFunc(
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

	authenticated :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				authenticatedCalled = true

				w.WriteHeader(
					http.StatusCreated,
				)
			},
		)

	handler :=
		buildCoursewareComicRouteHandler(
			base,
			authenticated,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/coursewares/cw-1/comic-projects/project-1/references/reference-1/delete",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusNotFound ||
		baseCalled ||
		authenticatedCalled {
		t.Fatalf(
			"非法漫画保留路径处理错误: status=%d base=%t auth=%t",
			recorder.Code,
			baseCalled,
			authenticatedCalled,
		)
	}
}

func TestCoursewareComicPromptRouteReturnsNotFound(
	t *testing.T,
) {
	baseCalled := false
	authenticatedCalled := false

	base :=
		http.HandlerFunc(
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

	authenticated :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				authenticatedCalled = true

				w.WriteHeader(
					http.StatusCreated,
				)
			},
		)

	handler :=
		buildCoursewareComicRouteHandler(
			base,
			authenticated,
		)

	request :=
		httptest.NewRequest(
			http.MethodPut,
			"/api/v1/coursewares/cw-1/comic-projects/project-1/panels/panel-1/prompt",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusNotFound ||
		baseCalled ||
		authenticatedCalled {
		t.Fatalf(
			"提示词与IAOCI接口没有安全关闭: status=%d base=%t auth=%t",
			recorder.Code,
			baseCalled,
			authenticatedCalled,
		)
	}
}

func TestCoursewareComicPlanSceneRegistered(
	t *testing.T,
) {
	if !models.IsValidSceneCode(
		models.SceneCoursewareComicPlan,
	) {
		t.Fatal(
			"知识点漫画规划场景没有进入AI配置场景白名单",
		)
	}

	if models.SceneNameMap[
		models.SceneCoursewareComicPlan,
	] == "" {
		t.Fatal(
			"知识点漫画规划场景缺少管理端名称",
		)
	}

	if models.SceneGroupMap[
		models.SceneCoursewareComicPlan,
	] !=
		"courseware" {
		t.Fatalf(
			"知识点漫画规划场景分组错误: %s",
			models.SceneGroupMap[
				models.SceneCoursewareComicPlan,
			],
		)
	}
}
