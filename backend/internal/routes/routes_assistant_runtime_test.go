package routes

// routes_assistant_runtime_test.go
//
// 本文件只验证不连接数据库的公开运行路径匹配和包装层功能开关。
// 不启动完整业务路由，不访问AI，不执行积分结算。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAssistantRuntimeRouteMatching(
	t *testing.T,
) {
	cases :=
		[]struct {
			name       string
			path       string
			routeKind  string
			resourceID string
			matched    bool
		}{
			{
				name:       "embed",
				path:       "/embed/assistant/public-id",
				routeKind:  assistantRuntimeRouteEmbed,
				resourceID: "public-id",
				matched:    true,
			},
			{
				name:       "start session",
				path:       "/api/v1/assistant-runtime/deployments/public-id/session",
				routeKind:  assistantRuntimeRouteStartSession,
				resourceID: "public-id",
				matched:    true,
			},
			{
				name:       "get session",
				path:       "/api/v1/assistant-runtime/sessions/session-id",
				routeKind:  assistantRuntimeRouteGetSession,
				resourceID: "session-id",
				matched:    true,
			},
			{
				name:       "chat",
				path:       "/api/v1/assistant-runtime/sessions/session-id/chat",
				routeKind:  assistantRuntimeRouteChat,
				resourceID: "session-id",
				matched:    true,
			},
			{
				name:       "invalid trailing slash",
				path:       "/api/v1/assistant-runtime/sessions/session-id/",
				routeKind:  assistantRuntimeRouteInvalid,
				resourceID: "",
				matched:    true,
			},
			{
				name:       "invalid nested path",
				path:       "/embed/assistant/public-id/extra",
				routeKind:  assistantRuntimeRouteInvalid,
				resourceID: "",
				matched:    true,
			},
			{
				name:       "unrelated",
				path:       "/api/v1/health",
				routeKind:  "",
				resourceID: "",
				matched:    false,
			},
		}

	for _, item := range cases {
		t.Run(
			item.name,
			func(
				t *testing.T,
			) {
				routeKind,
					resourceID,
					matched :=
					matchAssistantRuntimeRoute(
						item.path,
					)

				if routeKind !=
					item.routeKind ||
					resourceID !=
						item.resourceID ||
					matched !=
						item.matched {
					t.Fatalf(
						"路由匹配错误: path=%s route=%s id=%s matched=%t",
						item.path,
						routeKind,
						resourceID,
						matched,
					)
				}
			},
		)
	}
}

func TestAssistantRuntimeRouteWrapperPassesThrough(
	t *testing.T,
) {
	base :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.WriteHeader(
					http.StatusAccepted,
				)
			},
		)

	handler :=
		buildAssistantRuntimeRouteHandler(
			base,
			nil,
			false,
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
		http.StatusAccepted {
		t.Fatalf(
			"非运行时路径没有透传: status=%d",
			recorder.Code,
		)
	}
}

func TestAssistantRuntimePublicEntryBlockedWhenDisabled(
	t *testing.T,
) {
	base :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.WriteHeader(
					http.StatusAccepted,
				)
			},
		)

	handler :=
		buildAssistantRuntimeRouteHandler(
			base,
			nil,
			false,
		)

	cases :=
		[]struct {
			name   string
			method string
			path   string
		}{
			{
				name:   "embed blocked",
				method: http.MethodGet,
				path:   "/embed/assistant/public-id",
			},
			{
				name:   "external session creation blocked",
				method: http.MethodPost,
				path:   "/api/v1/assistant-runtime/deployments/public-id/session",
			},
		}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				request :=
					httptest.NewRequest(
						item.method,
						item.path,
						nil,
					)

				recorder :=
					httptest.NewRecorder()

				handler.ServeHTTP(
					recorder,
					request,
				)

				if recorder.Code !=
					http.StatusServiceUnavailable {
					t.Fatalf(
						"公开入口关闭状态错误: status=%d body=%s",
						recorder.Code,
						recorder.Body.String(),
					)
				}

				if !strings.Contains(
					recorder.Header().Get("Cache-Control"),
					"no-store",
				) {
					t.Fatalf(
						"公开入口关闭响应缺少no-store: header=%q",
						recorder.Header().Get("Cache-Control"),
					)
				}

				if !strings.Contains(
					recorder.Body.String(),
					"教学智能体公开运行暂未开放",
				) {
					t.Fatalf(
						"公开入口关闭文案错误: body=%s",
						recorder.Body.String(),
					)
				}
			},
		)
	}
}

func TestAssistantRuntimeSessionPathsStillReachHandlerWhenPublicDisabled(
	t *testing.T,
) {
	base :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.WriteHeader(
					http.StatusAccepted,
				)
			},
		)

	// runtimeHandler故意传nil。
	//
	// 如果GetSession被路由层错误地当成external入口直接阻断，
	// 响应正文会是“公开运行暂未开放”；
	// 正确行为是继续到运行处理器位置，再因处理器缺失返回“服务未就绪”。
	handler :=
		buildAssistantRuntimeRouteHandler(
			base,
			nil,
			false,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/assistant-runtime/sessions/session-id",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusServiceUnavailable {
		t.Fatalf(
			"会话读取路径状态错误: status=%d",
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		"教学智能体服务未就绪",
	) {
		t.Fatalf(
			"会话读取路径被公开开关错误拦截: body=%s",
			recorder.Body.String(),
		)
	}

	if strings.Contains(
		recorder.Body.String(),
		"公开运行暂未开放",
	) {
		t.Fatalf(
			"会话读取路径不应在路由层按external直接阻断: body=%s",
			recorder.Body.String(),
		)
	}
}

func TestAssistantRuntimeRouteWrapperOmittedFlagFailsClosed(
	t *testing.T,
) {
	base :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.WriteHeader(
					http.StatusAccepted,
				)
			},
		)

	// 兼容旧包内调用的同时，省略第三参数必须安全默认关闭。
	handler :=
		buildAssistantRuntimeRouteHandler(
			base,
			nil,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/embed/assistant/public-id",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusServiceUnavailable {
		t.Fatalf(
			"省略公开开关参数没有fail-closed: status=%d",
			recorder.Code,
		)
	}
}
