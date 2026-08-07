package handlers

// courseware_gen_tracked.go — 课件HTML生成与装配任务的受控启动入口
//
// 四条路径统一使用任务键：courseware_render:<coursewareID>。
// 同一课件不能同时执行封面预览、批量页面生成、全自动装配或3D页面生成。
//
// 快速部署断点续生：
//   - GeneratePages在关停时调用CancelGenerate，停止继续派发未开始页面；
//   - AutoAssemble在关停时调用CancelAutoAssemblyVersioned，先冻结数据库写回再停止继续派发；
//   - 已发出的同步AI请求不等待完整返回；
//   - 已成功落库页面保留；
//   - 新进程再次启动任务时只处理数据库中未完成页面。

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
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
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

	utils.Success(
		w,
		map[string]interface{}{
			"message":       "预览页生成已启动，请通过SSE监听进度",
			"courseware_id": id,
		},
	)
}

// GeneratePagesTracked 异步生成尚未完成的课件页面。
//
// GenerateRemainingPages会重新读取数据库，
// 只选择html_content为空的页面，因此本接口同时也是断点续生入口。
func (h *CoursewareGenHandler) GeneratePagesTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
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

	utils.Success(
		w,
		map[string]interface{}{
			"message":       "课件生成或断点续生已启动，请通过SSE监听进度",
			"courseware_id": id,
		},
	)
}

// AutoAssembleTracked 异步执行全自动装配。
//
// AutoAssemble会跳过已有HTML页面；
// 图片生成使用IAOCI稳定索引和媒体计费幂等键恢复。
func (h *CoursewareGenHandler) AutoAssembleTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
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

	var request struct {
		SkipVideo bool `json:"skip_video"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(
			r.Body,
		).Decode(
			&request,
		)
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
	skipVideo := request.SkipVideo

	// 必须先建立精确启动票据，再登记Tracker任务。
	//
	// 这样部署排空或用户取消发生在800毫秒缓冲期时，
	// 能只取消本次真实启动；空闲课件不会遗留取消状态。
	launchToken, launchErr :=
		services.PrepareCoursewareAutoAssemblyLaunch(
			id,
			skipVideo,
		)
	if launchErr != nil {
		utils.Fail(
			w,
			http.StatusConflict,
			"该课件已有自动装配正在启动或运行",
		)
		return
	}

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareRenderTaskType,
		id,
		services.BackgroundTaskCritical,
		func() {
			_ = h.autoAssemblyService.CancelAutoAssemblyVersioned(
				context.Background(),
				id,
				asyncActor,
			)
		},
		"该课件已有页面生成或装配任务正在执行",
	)
	if !started {
		services.AbortCoursewareAutoAssemblyLaunch(
			id,
			launchToken,
		)
		return
	}

	runTrackedBackgroundTask(
		task,
		trackedCoursewareRenderTaskType,
		id,
		800*time.Millisecond,
		func() error {
			// preflight或数据库领取失败时也必须清理尚未消费的票据。
			defer services.AbortCoursewareAutoAssemblyLaunch(
				id,
				launchToken,
			)

			return h.autoAssemblyService.
				AutoAssembleVersionedWithLaunch(
					context.Background(),
					id,
					asyncActor,
					skipVideo,
					launchToken,
				)
		},
	)

	utils.Success(
		w,
		map[string]interface{}{
			"message":       "全自动装配或断点续装已启动，请通过SSE监听进度",
			"courseware_id": id,
			"skip_video":    skipVideo,
		},
	)
}

// Generate3DPageTracked 异步生成3D互动单页。
func (h *CoursewareGenHandler) Generate3DPageTracked(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
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

	utils.Success(
		w,
		map[string]interface{}{
			"message":       "3D互动单页生成已启动，请通过SSE监听进度",
			"courseware_id": id,
		},
	)
}
