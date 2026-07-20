package handlers

// courseware_index_tracked.go — 课件方案异步任务的受控启动入口
//
// 本文件不改变原Handler方法，提供路由专用Tracked包装：
//   - GenerateIndexWithPresetTracked
//   - GenerateIndexFromTopicTracked
//   - RefineIndexTracked
//   - GenerateIndexFromPPTTracked
//   - GenerateIndexFromDocTracked
//
// 五条路径统一使用任务键：courseware_scheme:<coursewareID>。
// 因此同一课件不能同时生成、修改或重新生成多个方案，避免重复扣费和页面写库竞争。

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const trackedCoursewareSchemeTaskType = "courseware_scheme"

// authorizeCoursewareSchemeMutationForHandler 在正文解析和任务登记前
// 执行作者运行域、历史教育域与审核锁预检。
func authorizeCoursewareSchemeMutationForHandler(
	ctx context.Context,
	coursewareID string,
	userID string,
	role string,
) (*services.CoursewareActorContext, error) {
	actor := services.BuildCoursewareActorFromClaims(
		ctx,
		userID,
		role,
	)

	_, scopedActor, err :=
		(&services.CoursewareService{}).
			LoadCoursewareForOwnerControlMutation(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return nil, err
	}

	return scopedActor, nil
}

// GenerateIndexWithPresetTracked 带预设参数生成教案来源课件方案。
func (h *CoursewareIndexHandler) GenerateIndexWithPresetTracked(
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

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	scopedActor, err :=
		authorizeCoursewareSchemeMutationForHandler(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	var reqBody struct {
		Preset           string `json:"preset"`
		CustomPromptHint string `json:"custom_prompt_hint"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareSchemeTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件正在生成或修改方案，请等待当前任务完成",
	)
	if !started {
		return
	}

	preset := reqBody.Preset
	customHint := reqBody.CustomPromptHint

	runTrackedBackgroundTask(
		task,
		trackedCoursewareSchemeTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.indexService.GenerateIndex(
				context.Background(),
				id,
				asyncActor,
				preset,
				customHint,
			)
		},
	)

	utils.Success(w, map[string]interface{}{
		"message":       "课件索引生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}

// GenerateIndexFromTopicTracked 从主题生成课件方案。
func (h *CoursewareIndexHandler) GenerateIndexFromTopicTracked(
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

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index-topic")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	scopedActor, err :=
		authorizeCoursewareSchemeMutationForHandler(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	var reqBody struct {
		Subject          string `json:"subject"`
		Grade            string `json:"grade"`
		Topic            string `json:"topic"`
		PageRange        string `json:"page_range"`
		ExtraNotes       string `json:"extra_notes"`
		Preset           string `json:"preset"`
		CustomPromptHint string `json:"custom_prompt_hint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	req := &models.CreateCoursewareFromTopicRequest{
		Subject:    reqBody.Subject,
		Grade:      reqBody.Grade,
		Topic:      reqBody.Topic,
		PageRange:  reqBody.PageRange,
		ExtraNotes: reqBody.ExtraNotes,
	}

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareSchemeTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件正在生成或修改方案，请等待当前任务完成",
	)
	if !started {
		return
	}

	preset := reqBody.Preset
	customHint := reqBody.CustomPromptHint

	runTrackedBackgroundTask(
		task,
		trackedCoursewareSchemeTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.indexService.GenerateIndexFromTopic(
				context.Background(),
				id,
				asyncActor,
				req,
				preset,
				customHint,
			)
		},
	)

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":       "主题课件方案生成已启动，请通过SSE接收进度",
	})
}

// RefineIndexTracked 根据老师意见修改课件方案。
func (h *CoursewareIndexHandler) RefineIndexTracked(
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

	id := extractCoursewareMiddleID(r.URL.Path, "/refine-index")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	scopedActor, err :=
		authorizeCoursewareSchemeMutationForHandler(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	var req models.RefineIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if strings.TrimSpace(req.Feedback) == "" {
		utils.BadRequest(w, "修改意见不能为空")
		return
	}

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareSchemeTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件正在生成或修改方案，请等待当前任务完成",
	)
	if !started {
		return
	}

	feedback := req.Feedback

	runTrackedBackgroundTask(
		task,
		trackedCoursewareSchemeTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.indexService.RefineIndex(
				context.Background(),
				id,
				asyncActor,
				feedback,
			)
		},
	)

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":       "AI修改方案已启动，请通过SSE接收进度",
	})
}

// GenerateIndexFromPPTTracked 从已上传PPT生成课件方案。
func (h *CoursewareIndexHandler) GenerateIndexFromPPTTracked(
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

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index-ppt")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if h.pptService == nil {
		utils.InternalError(w, "PPT解析服务未初始化")
		return
	}

	scopedActor, err :=
		authorizeCoursewareSchemeMutationForHandler(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	var reqBody struct {
		Preset           string `json:"preset"`
		CustomPromptHint string `json:"custom_prompt_hint"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareSchemeTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件正在生成或修改方案，请等待当前任务完成",
	)
	if !started {
		return
	}

	preset := reqBody.Preset
	customHint := reqBody.CustomPromptHint

	runTrackedBackgroundTask(
		task,
		trackedCoursewareSchemeTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.pptService.GenerateIndexFromPPT(
				context.Background(),
				id,
				asyncActor,
				preset,
				customHint,
			)
		},
	)

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":       "PPT课件方案生成已启动，请通过SSE接收进度",
	})
}

// GenerateIndexFromDocTracked 从已上传Word文档生成课件方案。
func (h *CoursewareIndexHandler) GenerateIndexFromDocTracked(
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

	id := extractCoursewareMiddleID(r.URL.Path, "/generate-index-doc")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	if h.pptService == nil {
		utils.InternalError(w, "文档解析服务未初始化")
		return
	}

	scopedActor, err :=
		authorizeCoursewareSchemeMutationForHandler(
			r.Context(),
			id,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareControlError(w, err)
		return
	}

	asyncActor :=
		services.CloneCoursewareActorContext(
			scopedActor,
		)

	var reqBody struct {
		Preset           string `json:"preset"`
		CustomPromptHint string `json:"custom_prompt_hint"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	task, started := startTrackedBackgroundTask(
		w,
		trackedCoursewareSchemeTaskType,
		id,
		services.BackgroundTaskCritical,
		nil,
		"该课件正在生成或修改方案，请等待当前任务完成",
	)
	if !started {
		return
	}

	preset := reqBody.Preset
	customHint := reqBody.CustomPromptHint

	runTrackedBackgroundTask(
		task,
		trackedCoursewareSchemeTaskType,
		id,
		800*time.Millisecond,
		func() error {
			return h.pptService.GenerateIndexFromDoc(
				context.Background(),
				id,
				asyncActor,
				preset,
				customHint,
			)
		},
	)

	utils.Success(w, map[string]string{
		"courseware_id": id,
		"message":       "教案文档课件方案生成已启动，请通过SSE接收进度",
	})
}
