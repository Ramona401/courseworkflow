package routes

// routes_assistant_runtime.go
//
// 教学智能体公开运行路由包装。
//
// 公开端点：
//   GET  /embed/assistant/{public_id}
//   POST /api/v1/assistant-runtime/deployments/{public_id}/session
//   GET  /api/v1/assistant-runtime/sessions/{session_id}
//   POST /api/v1/assistant-runtime/sessions/{session_id}/chat
//
// 安全边界：
//   - 不套教师JWT；会话和聊天使用独立短时运行令牌；
//   - 签名密钥从JWT_SECRET按用途派生，不直接复用教师JWT用途；
//   - 匿名客户端和IP使用独立隐私盐做HMAC；
//   - 非公开运行路径交给教师端教学智能体、Style Studio和主路由；
//   - 本文件只接线路由，不修改数据库、不启动调度器、不执行部署。
//
// 功能开关语义：
//   - COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED=true时开放全部公开运行入口；
//   - false时阻断embed页面和external会话创建；
//   - 会话读取与聊天入口继续保留，供teacher_preview内部预览复用；
//   - 服务层按session_kind再次执行总闸门；
//   - 已签发external令牌在开关关闭后也会即时拒绝；
//   - teacher_preview不受公开运行开关影响。

import (
	"net/http"
	"strings"

	"tedna/internal/config"
	"tedna/internal/handlers"
	"tedna/internal/services"
)

const (
	assistantRuntimeRouteEmbed        = "embed"
	assistantRuntimeRouteStartSession = "start_session"
	assistantRuntimeRouteGetSession   = "get_session"
	assistantRuntimeRouteChat         = "chat"
	assistantRuntimeRouteInvalid      = "invalid"

	assistantRuntimeEmbedPrefix = "/embed/assistant/"

	assistantRuntimeDeploymentPrefix = "/api/v1/assistant-runtime/deployments/"

	assistantRuntimeSessionPrefix = "/api/v1/assistant-runtime/sessions/"
)

// SetupWithAssistantRuntime 在教师端完整路由外增加公开运行入口。
func SetupWithAssistantRuntime(cfg *config.Config) http.Handler {
	baseHandler := SetupWithCoursewareAssistant(cfg)

	publicRuntimeEnabled :=
		cfg.CoursewareAssistantEnabled &&
			cfg.CoursewareAssistantPublicRuntimeEnabled

	// 即使公开入口关闭，teacher_preview仍需要会话读取和聊天能力，
	// 因此会话、计费和聊天服务仍正常构造。
	sessionService := services.NewAssistantRuntimeSessionService(
		cfg.JWTSecret,
		cfg.AssistantRuntimePrivacySalt,
		cfg.GetAssistantRuntimeTokenTTL(),
	)

	// 必须在HTTP服务开始接收请求前注入可信后端开关。
	//
	// 该开关同时用于：
	//   - StartExternalSession创建防线；
	//   - ValidateRuntimeToken已签发external令牌实时防线。
	sessionService.SetPublicRuntimeEnabled(
		publicRuntimeEnabled,
	)

	billingService := services.NewAssistantRuntimeBillingService(
		sessionService,
		services.NewCreditPolicyService(),
	)

	chatService := services.NewAssistantRuntimeChatService(
		cfg,
		billingService,
	)

	runtimeHandler := handlers.NewAssistantRuntimeHandler(
		sessionService,
		chatService,
	)

	return buildAssistantRuntimeRouteHandler(
		baseHandler,
		runtimeHandler,
		publicRuntimeEnabled,
	)
}

// buildAssistantRuntimeRouteHandler 构造公开运行分发包装。
//
// publicRuntimeEnabledValues采用可选参数，只为兼容已有包内测试或内部调用。
// 省略时按安全默认false处理；生产Setup必须显式传入可信配置值。
func buildAssistantRuntimeRouteHandler(
	baseHandler http.Handler,
	runtimeHandler *handlers.AssistantRuntimeHandler,
	publicRuntimeEnabledValues ...bool,
) http.Handler {
	publicRuntimeEnabled := false

	if len(publicRuntimeEnabledValues) > 0 {
		publicRuntimeEnabled =
			publicRuntimeEnabledValues[0]
	}

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			routeKind, resourceID, matched :=
				matchAssistantRuntimeRoute(
					r.URL.Path,
				)

			if !matched {
				baseHandler.ServeHTTP(w, r)
				return
			}

			// 命中公开运行保留前缀但结构非法时始终404，
			// 不因功能开关状态泄露更多路径信息。
			if routeKind == assistantRuntimeRouteInvalid {
				http.NotFound(w, r)
				return
			}

			// 公开开关关闭时，只在路由层阻断外部入口：
			//   1. 官方embed页面；
			//   2. external新会话创建。
			//
			// GetSession和Chat继续进入正式处理器，
			// 由服务层读取数据库session_kind后：
			//   - external即时拒绝；
			//   - teacher_preview继续运行。
			if !publicRuntimeEnabled &&
				(routeKind == assistantRuntimeRouteEmbed ||
					routeKind == assistantRuntimeRouteStartSession) {
				writeAssistantPublicRuntimeDisabled(
					w,
					r,
					routeKind,
				)
				return
			}

			if runtimeHandler == nil {
				http.Error(
					w,
					"教学智能体服务未就绪",
					http.StatusServiceUnavailable,
				)
				return
			}

			switch routeKind {
			case assistantRuntimeRouteEmbed:
				runtimeHandler.Embed(w, r, resourceID)

			case assistantRuntimeRouteStartSession:
				runtimeHandler.StartSession(w, r, resourceID)

			case assistantRuntimeRouteGetSession:
				runtimeHandler.GetSession(w, r, resourceID)

			case assistantRuntimeRouteChat:
				runtimeHandler.Chat(w, r, resourceID)

			default:
				http.NotFound(w, r)
			}
		},
	)
}

// writeAssistantPublicRuntimeDisabled 输出公开运行关闭状态。
//
// embed页面返回无缓存的503文本；
// API入口返回既有公开API可识别的JSON信封，
// 避免浏览器显示“无效响应”。
func writeAssistantPublicRuntimeDisabled(
	w http.ResponseWriter,
	r *http.Request,
	routeKind string,
) {
	w.Header().Set(
		"Cache-Control",
		"no-store, no-cache, must-revalidate",
	)

	if routeKind == assistantRuntimeRouteEmbed {
		http.Error(
			w,
			"教学智能体公开运行暂未开放",
			http.StatusServiceUnavailable,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	w.WriteHeader(http.StatusServiceUnavailable)

	_, _ = w.Write(
		[]byte(
			`{"code":50301,"message":"教学智能体公开运行暂未开放"}`,
		),
	)
}

// matchAssistantRuntimeRoute 严格匹配公开运行路径。
//
// matched=false表示与运行时无关，应交给教师端完整路由。
// matched=true且routeKind=invalid表示命中运行前缀但结构非法，应直接404。
func matchAssistantRuntimeRoute(
	path string,
) (
	routeKind string,
	resourceID string,
	matched bool,
) {
	path = strings.TrimSpace(path)

	switch {
	case strings.HasPrefix(path, assistantRuntimeEmbedPrefix):
		publicID := strings.TrimPrefix(
			path,
			assistantRuntimeEmbedPrefix,
		)

		if validAssistantRuntimeRouteID(publicID) {
			return assistantRuntimeRouteEmbed, publicID, true
		}

		return assistantRuntimeRouteInvalid, "", true

	case strings.HasPrefix(path, assistantRuntimeDeploymentPrefix):
		rest := strings.TrimPrefix(
			path,
			assistantRuntimeDeploymentPrefix,
		)
		parts := strings.Split(rest, "/")

		if len(parts) == 2 &&
			validAssistantRuntimeRouteID(parts[0]) &&
			parts[1] == "session" {
			return assistantRuntimeRouteStartSession, parts[0], true
		}

		return assistantRuntimeRouteInvalid, "", true

	case strings.HasPrefix(path, assistantRuntimeSessionPrefix):
		rest := strings.TrimPrefix(
			path,
			assistantRuntimeSessionPrefix,
		)
		parts := strings.Split(rest, "/")

		if len(parts) == 1 &&
			validAssistantRuntimeRouteID(parts[0]) {
			return assistantRuntimeRouteGetSession, parts[0], true
		}

		if len(parts) == 2 &&
			validAssistantRuntimeRouteID(parts[0]) &&
			parts[1] == "chat" {
			return assistantRuntimeRouteChat, parts[0], true
		}

		return assistantRuntimeRouteInvalid, "", true

	default:
		return "", "", false
	}
}

// validAssistantRuntimeRouteID 校验单段公开编号或会话编号。
func validAssistantRuntimeRouteID(value string) bool {
	return value != "" &&
		len(value) <= 256 &&
		strings.TrimSpace(value) == value &&
		value != "." &&
		value != ".." &&
		!strings.Contains(value, "/")
}
