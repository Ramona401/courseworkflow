package handlers

// courseware_gen_tracked.go — 课件HTML生成与装配任务的受控启动入口
//
// 四条路径统一使用任务键：courseware_render:<coursewareID>。
// 同一课件不能同时执行封面预览、批量页面生成、全自动装配或3D页面生成。
//
// GeneratePages额外登记onDrain钩子：服务进入排空时调用CancelGenerate，
// 停止继续派发尚未开始的页面；已经发出的AI请求仍自然完成并写库。

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const trackedCoursewareRenderTaskType = "courseware_render"

// GeneratePreviewTracked 异步生成封面预览页。
func (h *CoursewareGenHandler) GeneratePreviewTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/generate-preview",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// 必须在登记后台任务前完成作者专属教育域预检。
	// 否则无权调用者也能短暂占用courseware_render任务锁。
	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareRenderTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件已有页面生成或装配任务正在执行",
	)
	if !started {
		return
	}

	runTrackedBackgroundTask(
		task,
		trackedCoursewareRenderTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.genService.GeneratePreviewPages(
				context.Background(),
				id,
				asyncActor,
			)
		},
	)

	utils.Success(w, map[string]interface{}{
		"message":       "预览页生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}

// GeneratePagesTracked 异步批量生成剩余课件页。
func (h *CoursewareGenHandler) GeneratePagesTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/generate-pages",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	// 同一份收敛Actor同时供后台生成与服务排空取消使用。
	// Actor只读，可安全由两个闭包共享。
	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareRenderTaskType,
		id,
		services.BackgroundTaskCritical,
		func() {
			_ = h.genService.CancelGenerate(
				context.Background(),
				id,
				asyncActor,
			)
		},
		"该课件已有页面生成或装配任务正在执行",
	)
	if !started {
		return
	}

	runTrackedBackgroundTask(
		task,
		trackedCoursewareRenderTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.genService.GenerateRemainingPages(
				context.Background(),
				id,
				asyncActor,
			)
		},
	)

	utils.Success(w, map[string]interface{}{
		"message":       "课件生成已启动（使用固定导航栏），请通过SSE监听进度",
		"courseware_id": id,
	})
}

// AutoAssembleTracked 异步执行全自动装配。
func (h *CoursewareGenHandler) AutoAssembleTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/auto-assemble",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	var req struct {
		SkipVideo bool `json:"skip_video"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)
	skipVideo := req.SkipVideo

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareRenderTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件已有页面生成或装配任务正在执行",
	)
	if !started {
		return
	}

	runTrackedBackgroundTask(
		task,
		trackedCoursewareRenderTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.autoAssemblyService.AutoAssemble(
				context.Background(),
				id,
				asyncActor,
				skipVideo,
			)
		},
	)

	utils.Success(w, map[string]interface{}{
		"message":       "全自动装配已启动，请通过SSE监听 assembly_* 进度事件",
		"courseware_id": id,
		"skip_video":    skipVideo,
	})
}

// Generate3DPageTracked 异步生成3D互动单页。
func (h *CoursewareGenHandler) Generate3DPageTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	id := extractCoursewareMiddleID(
		r.URL.Path,
		"/generate-3d-page",
	)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	scopedActor, err :=
		h.authorizeCoursewareOwnerRuntime(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareOwnerRuntimeError(
			w,
			err,
		)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareRenderTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件已有页面生成或装配任务正在执行",
	)
	if !started {
		return
	}

	runTrackedBackgroundTask(
		task,
		trackedCoursewareRenderTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.genService.Generate3DSinglePage(
				context.Background(),
				id,
				asyncActor,
			)
		},
	)

	utils.Success(w, map[string]interface{}{
		"message":       "3D互动单页生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}
