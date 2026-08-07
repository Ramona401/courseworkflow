package handlers

// courseware_assistant_preview_handler.go
//
// 本文件提供教师端教学智能体内部预览会话创建入口：
//
//   POST /api/v1/assistant-deployments/{deployment_id}/preview-session
//
// 安全边界：
//   - 路由必须经过现有教师JWT认证；
//   - 教师身份只取claims.UserID；
//   - 请求正文不能提交owner_user_id、school_id、education_domain或计费账户；
//   - Service与Repository同时绑定deployment_id和owner_user_id；
//   - 只有部署所有者本人可以创建预览会话；
//   - admin角色不会自动取得其他教师部署的预览权；
//   - 返回的是短时assistant_runtime运行令牌，不是教师JWT；
//   - 后续聊天复用正式运行接口和正式积分结算；
//   - 教师预览扣部署所有者积分，但不占外部学生每日额度。

import (
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CoursewareAssistantPreviewHandler 是教师内部预览会话处理器。
type CoursewareAssistantPreviewHandler struct {
	sessionService *services.AssistantRuntimeSessionService
}

// NewCoursewareAssistantPreviewHandler 创建教师内部预览处理器。
func NewCoursewareAssistantPreviewHandler(
	sessionService *services.AssistantRuntimeSessionService,
) *CoursewareAssistantPreviewHandler {
	return &CoursewareAssistantPreviewHandler{
		sessionService: sessionService,
	}
}

// StartPreviewSession 创建绑定当前部署版本的教师预览会话。
func (h *CoursewareAssistantPreviewHandler) StartPreviewSession(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil ||
		strings.TrimSpace(claims.UserID) == "" {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	deploymentID =
		strings.TrimSpace(
			deploymentID,
		)
	if deploymentID == "" {
		utils.BadRequest(
			w,
			"教学智能体部署ID无效",
		)
		return
	}

	if h == nil ||
		h.sessionService == nil {
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"教学智能体预览服务未就绪",
		)
		return
	}

	clientIP, err :=
		assistantRuntimeClientIP(
			r,
		)
	if err != nil {
		utils.BadRequest(
			w,
			"无法识别客户端网络地址",
		)
		return
	}

	response, err :=
		h.sessionService.StartTeacherPreviewSession(
			r.Context(),
			deploymentID,
			claims.UserID,
			clientIP,
		)
	if err != nil {
		writeAssistantRuntimeHTTPError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		response,
	)
}
