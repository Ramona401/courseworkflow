package handlers

// assistant_deployment_handler.go
//
// 教师端课件教学智能体部署HTTP入口。
// 所有接口要求教师JWT；部署身份、学校、教育域和快照均由Service重读。
// 浏览器只得到部署元数据和版本哈希，不得到提示词、上下文正文或模型配置。

import (
	"errors"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const assistantDeploymentRequestMaxBytes int64 = 64 * 1024

// AssistantDeploymentHandler 管理首发、版本、状态和实时策略。
type AssistantDeploymentHandler struct {
	service *services.AssistantDeploymentService
}

// NewAssistantDeploymentHandler 创建部署管理处理器。
func NewAssistantDeploymentHandler(
	service *services.AssistantDeploymentService,
) *AssistantDeploymentHandler {
	return &AssistantDeploymentHandler{
		service: service,
	}
}

// Publish 首次发布当前页面已经确认的教学智能体插槽。
func (h *AssistantDeploymentHandler) Publish(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	pageID string,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	actor, ok :=
		assistantDeploymentActorFromRequest(
			w,
			r,
		)

	if !ok ||
		!assistantDeploymentServiceReady(
			w,
			h,
		) {
		return
	}

	// 发布会读取JSON并创建正式部署，先完成作者运行通道预检。
	scopedActor, err :=
		authorizeCoursewareOwnerRuntimeForHandler(
			r.Context(),
			coursewareID,
			actor.UserID,
			actor.Role,
		)
	if err != nil {
		writeAssistantDeploymentHandlerError(
			w,
			err,
		)
		return
	}

	var request models.CreateAssistantDeploymentRequest

	if !decodeCoursewareAssistantJSON(
		w,
		r,
		&request,
		assistantDeploymentRequestMaxBytes,
	) {
		return
	}

	response, err :=
		h.service.PublishAssistantDeployment(
			r.Context(),
			coursewareID,
			pageID,
			scopedActor,
			&request,
		)
	if err != nil {
		writeAssistantDeploymentHandlerError(
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

// List 返回作者当前课件的全部部署历史。
func (h *AssistantDeploymentHandler) List(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	actor, ok :=
		assistantDeploymentActorFromRequest(
			w,
			r,
		)

	if !ok ||
		!assistantDeploymentServiceReady(
			w,
			h,
		) {
		return
	}

	response, err :=
		h.service.ListAssistantDeployments(
			r.Context(),
			coursewareID,
			actor,
		)
	if err != nil {
		writeAssistantDeploymentHandlerError(
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

// PublishVersion 从当前正式插槽追加不可变版本。
func (h *AssistantDeploymentHandler) PublishVersion(
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

	actor, ok :=
		assistantDeploymentActorFromRequest(
			w,
			r,
		)

	if !ok ||
		!assistantDeploymentServiceReady(
			w,
			h,
		) {
		return
	}

	response, err :=
		h.service.PublishAssistantDeploymentVersion(
			r.Context(),
			deploymentID,
			actor,
		)
	if err != nil {
		writeAssistantDeploymentHandlerError(
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

// ListVersions 返回版本哈希历史，不返回快照正文。
func (h *AssistantDeploymentHandler) ListVersions(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodGetOnly,
		)
		return
	}

	actor, ok :=
		assistantDeploymentActorFromRequest(
			w,
			r,
		)

	if !ok ||
		!assistantDeploymentServiceReady(
			w,
			h,
		) {
		return
	}

	response, err :=
		h.service.ListAssistantDeploymentVersions(
			r.Context(),
			deploymentID,
			actor,
		)
	if err != nil {
		writeAssistantDeploymentHandlerError(
			w,
			err,
		)
		return
	}

	if response == nil {
		response =
			[]*models.AssistantDeploymentVersionView{}
	}

	utils.Success(
		w,
		response,
	)
}

// Pause 暂停部署。
func (h *AssistantDeploymentHandler) Pause(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
) {
	h.transition(
		w,
		r,
		deploymentID,
		"pause",
	)
}

// Resume 恢复暂停部署。
func (h *AssistantDeploymentHandler) Resume(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
) {
	h.transition(
		w,
		r,
		deploymentID,
		"resume",
	)
}

// Revoke 永久撤销部署。
func (h *AssistantDeploymentHandler) Revoke(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
) {
	h.transition(
		w,
		r,
		deploymentID,
		"revoke",
	)
}

// transition 执行固定状态机动作。
func (h *AssistantDeploymentHandler) transition(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
	action string,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	actor, ok :=
		assistantDeploymentActorFromRequest(
			w,
			r,
		)

	if !ok ||
		!assistantDeploymentServiceReady(
			w,
			h,
		) {
		return
	}

	var (
		response *models.AssistantDeploymentView
		err      error
	)

	switch action {
	case "pause":
		response, err =
			h.service.PauseAssistantDeployment(
				r.Context(),
				deploymentID,
				actor,
			)

	case "resume":
		response, err =
			h.service.ResumeAssistantDeployment(
				r.Context(),
				deploymentID,
				actor,
			)

	case "revoke":
		response, err =
			h.service.RevokeAssistantDeployment(
				r.Context(),
				deploymentID,
				actor,
			)

	default:
		err =
			repository.ErrAssistantDeploymentStateConflict
	}

	if err != nil {
		writeAssistantDeploymentHandlerError(
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

// UpdatePolicy 更新Origin、每日额度、单会话轮数和有效期。
func (h *AssistantDeploymentHandler) UpdatePolicy(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPutOnly,
		)
		return
	}

	actor, ok :=
		assistantDeploymentActorFromRequest(
			w,
			r,
		)

	if !ok ||
		!assistantDeploymentServiceReady(
			w,
			h,
		) {
		return
	}

	var request models.UpdateAssistantDeploymentPolicyRequest

	if !decodeCoursewareAssistantJSON(
		w,
		r,
		&request,
		assistantDeploymentRequestMaxBytes,
	) {
		return
	}

	response, err :=
		h.service.UpdateAssistantDeploymentPolicy(
			r.Context(),
			deploymentID,
			actor,
			&request,
		)
	if err != nil {
		writeAssistantDeploymentHandlerError(
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

// assistantDeploymentServiceReady 检查Handler依赖。
func assistantDeploymentServiceReady(
	w http.ResponseWriter,
	h *AssistantDeploymentHandler,
) bool {
	if h != nil &&
		h.service != nil {
		return true
	}

	utils.Fail(
		w,
		http.StatusServiceUnavailable,
		"课件教学智能体部署服务暂不可用",
	)

	return false
}

// assistantDeploymentActorFromRequest 只从认证Claims构造可信Actor。
func assistantDeploymentActorFromRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	*services.CoursewareActorContext,
	bool,
) {
	if r == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return nil, false
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)

	if !ok ||
		claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return nil, false
	}

	actor :=
		services.BuildCoursewareActorFromClaims(
			r.Context(),
			claims.UserID,
			claims.Role,
		)

	if actor == nil {
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权执行此课件教学智能体部署操作",
		)
		return nil, false
	}

	return actor, true
}

// writeAssistantDeploymentHandlerError 将部署错误收敛为稳定公开响应。
func writeAssistantDeploymentHandlerError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrAssistantDeploymentPolicyInvalid,
	),
		errors.Is(
			err,
			services.ErrAssistantDeploymentOriginInvalid,
		):
		utils.BadRequest(
			w,
			"课件教学智能体部署策略无效",
		)

	case errors.Is(
		err,
		services.ErrAssistantDeploymentActorRequired,
	):
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权执行此课件教学智能体部署操作",
		)

	case errors.Is(
		err,
		repository.ErrAssistantDeploymentNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"课件教学智能体部署不存在",
		)

	case errors.Is(
		err,
		repository.ErrAssistantDeploymentVersionNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"课件教学智能体部署版本不存在",
		)

	case isAssistantDeploymentConflict(
		err,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			assistantDeploymentConflictMessage(
				err,
			),
		)

	case errors.Is(
		err,
		services.ErrAssistantDeploymentSnapshotInvalid,
	),
		errors.Is(
			err,
			services.ErrAssistantDeploymentStoredPolicyInvalid,
		),
		errors.Is(
			err,
			repository.ErrAssistantDeploymentInvalidRecord,
		),
		errors.Is(
			err,
			repository.ErrAssistantDeploymentPublicIDConflict,
		):
		utils.InternalError(
			w,
			"课件教学智能体部署数据异常，请稍后重试",
		)

	default:
		// 插槽、上下文、助手选择和课件访问复用已有安全映射。
		writeCoursewareAssistantHandlerError(
			w,
			err,
		)
	}
}

// isAssistantDeploymentConflict 判断可向教师解释的状态冲突。
func isAssistantDeploymentConflict(
	err error,
) bool {
	return errors.Is(
		err,
		services.ErrAssistantDeploymentCoursewareNotPublishable,
	) ||
		errors.Is(
			err,
			services.ErrAssistantDeploymentSlotRequired,
		) ||
		errors.Is(
			err,
			services.ErrAssistantDeploymentSlotInactive,
		) ||
		errors.Is(
			err,
			services.ErrAssistantDeploymentAssistantRequired,
		) ||
		errors.Is(
			err,
			services.ErrAssistantDeploymentAssistantPromptRequired,
		) ||
		errors.Is(
			err,
			services.ErrAssistantDeploymentSchoolRequired,
		) ||
		errors.Is(
			err,
			services.ErrAssistantDeploymentSlotChanged,
		) ||
		errors.Is(
			err,
			repository.ErrAssistantDeploymentPageAlreadyLive,
		) ||
		errors.Is(
			err,
			repository.ErrAssistantDeploymentRevoked,
		) ||
		errors.Is(
			err,
			repository.ErrAssistantDeploymentStateConflict,
		)
}

// assistantDeploymentConflictMessage 返回不含内部信息的冲突文案。
func assistantDeploymentConflictMessage(
	err error,
) string {
	switch {
	case errors.Is(
		err,
		repository.ErrAssistantDeploymentPageAlreadyLive,
	):
		return "当前页面已经存在未撤销的教学智能体部署"

	case errors.Is(
		err,
		repository.ErrAssistantDeploymentRevoked,
	):
		return "教学智能体部署已永久撤销"

	case errors.Is(
		err,
		services.ErrAssistantDeploymentSlotChanged,
	):
		return "部署关联的教学智能体插槽已删除或替换，请重新发布"

	case errors.Is(
		err,
		services.ErrAssistantDeploymentSchoolRequired,
	):
		return "当前教师未绑定可用学校，不能发布教学智能体"

	case errors.Is(
		err,
		services.ErrAssistantDeploymentCoursewareNotPublishable,
	):
		return "课件当前状态不允许发布教学智能体"

	case errors.Is(
		err,
		services.ErrAssistantDeploymentSlotInactive,
	):
		return "当前页面教学智能体插槽已停用"

	case errors.Is(
		err,
		services.ErrAssistantDeploymentAssistantPromptRequired,
	):
		return "已选择的AI助手当前不可发布，请清除该助手后保存，或重新选择可用助手"

	case errors.Is(
		err,
		services.ErrAssistantDeploymentAssistantRequired,
	):
		return "当前页面教学智能体发布信息不完整，请重新保存方案"

	case errors.Is(
		err,
		services.ErrAssistantDeploymentSlotRequired,
	):
		return "当前页面缺少完整可发布的教学智能体方案"

	default:
		return "课件教学智能体部署状态冲突，请刷新后重试"
	}
}
