package handlers

// courseware_comic_handler.go — 知识点漫画作者端HTTP处理器
//
// 项目主体视图与教师五步工作流分别读取，
// 在返回浏览器前执行安全装配。
//
// AI规划成功后，教师工作流从source推进到storyboard，
// 后台项目status仍由规划服务维护为planned。
//
// 教师端只允许保存文字、题目与气泡覆盖层。
// 图片视觉提示词、负面提示词、IAOCI和图片关系索引由后端维护，
// 本处理器不再提供浏览器直接修改入口。

import (
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

type CoursewareComicHandler struct {
	projectService *services.CoursewareComicProjectService
	planService    *services.CoursewareComicPlanService
}

func NewCoursewareComicHandler(
	projectService *services.CoursewareComicProjectService,
	planService *services.CoursewareComicPlanService,
) *CoursewareComicHandler {
	return &CoursewareComicHandler{
		projectService: projectService,
		planService:    planService,
	}
}

func (h *CoursewareComicHandler) CreateProject(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
) {
	if r.Method != http.MethodPost {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持POST请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var request models.CreateCoursewareComicProjectRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicCreateRequestMaxBytes,
	) {
		return
	}

	projectService :=
		h.resolveProjectService()

	result, err :=
		projectService.CreateProject(
			r.Context(),
			coursewareID,
			actor,
			&request,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	workflow, err :=
		projectService.GetProjectWorkflowForBrowser(
			r.Context(),
			coursewareID,
			result.ID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		attachCoursewareComicWorkflowToProject(
			result,
			workflow,
		); err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

func (h *CoursewareComicHandler) ListProjects(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
) {
	if r.Method != http.MethodGet {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持GET请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	projectService :=
		h.resolveProjectService()

	result, err :=
		projectService.ListProjects(
			r.Context(),
			coursewareID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	workflows, err :=
		projectService.ListProjectWorkflowsForBrowser(
			r.Context(),
			coursewareID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		attachCoursewareComicWorkflowsToList(
			result,
			workflows,
		); err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

func (h *CoursewareComicHandler) GetProject(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
) {
	if r.Method != http.MethodGet {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持GET请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	projectService :=
		h.resolveProjectService()

	result, err :=
		projectService.GetProjectDetailForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	workflow, err :=
		projectService.GetProjectWorkflowForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		attachCoursewareComicWorkflowToDetail(
			result,
			workflow,
		); err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

func (h *CoursewareComicHandler) PlanProject(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
) {
	if r.Method != http.MethodPost {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持POST请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var request models.PlanCoursewareComicRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPlanRequestMaxBytes,
	) {
		return
	}

	planService :=
		h.resolvePlanService()

	if planService == nil {
		writeCoursewareComicHandlerError(
			w,
			services.
				ErrCoursewareComicPlanServiceUnavailable,
		)
		return
	}

	_, err :=
		planService.PlanProject(
			r.Context(),
			coursewareID,
			projectID,
			actor,
			&request,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	projectService :=
		h.resolveProjectService()

	workflow, err :=
		projectService.AdvanceWorkflowAfterPlanning(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	result, err :=
		projectService.GetProjectDetailForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		attachCoursewareComicWorkflowToDetail(
			result,
			workflow,
		); err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

func (h *CoursewareComicHandler) UpdatePanelOverlay(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
	panelID string,
) {
	if r.Method != http.MethodPut {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持PUT请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var request models.UpdateCoursewareComicPanelOverlayRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicOverlayRequestMaxBytes,
	) {
		return
	}

	result, err :=
		h.resolveProjectService().
			UpdatePanelOverlayForWorkshop(
				r.Context(),
				coursewareID,
				projectID,
				panelID,
				actor,
				&request,
			)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

func authorizeCoursewareComicActor(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
) (*services.CoursewareActorContext, bool) {
	if r == nil {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return nil, false
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok ||
		claims == nil ||
		strings.TrimSpace(
			claims.UserID,
		) == "" {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return nil, false
	}

	actor, err :=
		authorizeCoursewareOwnerRuntimeForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return nil, false
	}

	return actor, true
}

func coursewareComicMethodNotAllowed(
	w http.ResponseWriter,
	message string,
) {
	utils.Fail(
		w,
		http.StatusMethodNotAllowed,
		message,
	)
}

func (h *CoursewareComicHandler) resolveProjectService() *services.CoursewareComicProjectService {
	if h != nil &&
		h.projectService != nil {
		return h.projectService
	}

	return services.
		NewCoursewareComicProjectService()
}

func (h *CoursewareComicHandler) resolvePlanService() *services.CoursewareComicPlanService {
	if h != nil {
		return h.planService
	}

	return nil
}
