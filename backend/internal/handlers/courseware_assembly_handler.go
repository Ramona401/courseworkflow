package handlers

// courseware_assembly_handler.go — 自动装配运行状态与显式取消接口
//
// 该Handler只处理两个作者专属生命周期端点：
//   GET  /api/v1/coursewares/{id}/assembly-state
//   POST /api/v1/coursewares/{id}/cancel-auto-assemble
//
// 正式启动仍由既有AutoAssembleTracked负责；本文件不复制生成逻辑。

import (
	"errors"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// CoursewareAssemblyHandler 自动装配生命周期Handler。
type CoursewareAssemblyHandler struct {
	coursewareService *services.CoursewareService
}

// NewCoursewareAssemblyHandler 创建生命周期Handler。
func NewCoursewareAssemblyHandler(
	coursewareService *services.CoursewareService,
) *CoursewareAssemblyHandler {
	if coursewareService == nil {
		coursewareService =
			services.NewCoursewareService()
	}

	return &CoursewareAssemblyHandler{
		coursewareService: coursewareService,
	}
}

// authorizeOwner 构造可信Actor并校验课件作者运行权限。
func (h *CoursewareAssemblyHandler) authorizeOwner(
	r *http.Request,
	coursewareID string,
) (
	*services.CoursewareActorContext,
	error,
) {
	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		return nil,
			services.ErrCoursewareActorRequired
	}

	actor :=
		services.BuildCoursewareActorFromClaims(
			r.Context(),
			claims.UserID,
			claims.Role,
		)

	_, scopedActor, err :=
		h.coursewareService.
			LoadCoursewareForOwnerRuntime(
				r.Context(),
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// GetState 返回课件当前数据库装配生命周期状态。
func (h *CoursewareAssemblyHandler) GetState(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	scopedActor, err :=
		h.authorizeOwner(
			r,
			coursewareID,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	// 明确使用变量，表示授权结果不能被编译器优化为无意义调用。
	if scopedActor == nil {
		writeCoursewareOwnerRuntimeError(
			w,
			services.ErrCoursewareActorRequired,
		)
		return
	}

	state, err :=
		repository.GetCoursewareAssemblyState(
			r.Context(),
			coursewareID,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrCoursewareAssemblyNotFound,
		):
			utils.Fail(
				w,
				http.StatusNotFound,
				"课件不存在",
			)

		default:
			utils.InternalError(
				w,
				err.Error(),
			)
		}
		return
	}

	launchState :=
		services.GetCoursewareAutoAssemblyLaunchState(
			coursewareID,
		)

	databaseActive :=
		state.ActiveRunID != nil &&
			(state.Status ==
				models.CoursewareAssemblyStatusRunning ||
				state.Status ==
					models.CoursewareAssemblyStatusCancelRequested)

	runtimeStatus := state.Status
	effectiveSkipVideo :=
		state.SkipVideo
	var launchStartedAt interface{}

	if launchState.Pending {
		runtimeStatus = "starting"
		effectiveSkipVideo =
			launchState.SkipVideo
		launchStartedAt =
			launchState.StartedAt
	}

	utils.Success(
		w,
		map[string]interface{}{
			"courseware_id":     state.CoursewareID,
			"assembly_version":  state.Version,
			"assembly_status":   state.Status,
			"runtime_status":    runtimeStatus,
			"active_run_id":     state.ActiveRunID,
			"started_by":        state.StartedBy,
			"skip_video":        effectiveSkipVideo,
			"started_at":        state.StartedAt,
			"finished_at":       state.FinishedAt,
			"is_starting":       launchState.Pending,
			"launch_started_at": launchStartedAt,
			"is_active": databaseActive ||
				launchState.Pending,
		},
	)
}

// Cancel 显式请求取消当前自动装配。
func (h *CoursewareAssemblyHandler) Cancel(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	scopedActor, err :=
		h.authorizeOwner(
			r,
			coursewareID,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	err =
		services.CancelCoursewareAutoAssemblyVersioned(
			r.Context(),
			coursewareID,
			scopedActor,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrCoursewareAssemblyVersionConflict,
		):
			utils.Fail(
				w,
				http.StatusConflict,
				"装配运行状态已经变化，请刷新后重试",
			)

		default:
			writeCoursewareOwnerRuntimeError(
				w,
				err,
			)
		}
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"courseware_id": coursewareID,
			"message":       "已发送停止信号；已完成页面保留，迟到结果不会覆盖当前课件",
		},
	)
}
