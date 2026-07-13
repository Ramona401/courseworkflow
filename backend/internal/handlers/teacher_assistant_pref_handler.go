package handlers

// teacher_assistant_pref_handler.go — 老师×学科 AI 助手选择偏好 HTTP 处理器
//
// 偏好表仍保持 user_id+subject，不操作数据库结构。
// grade和stage是每次请求的运行时适用性条件：
//   - 存量偏好不删除；
//   - 在其它年级或不适用阶段下，视为当前没有有效偏好；
//   - 切回原适用年级和阶段时仍可重新生效。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// TeacherAssistantPrefHandler 老师助手偏好处理器。
type TeacherAssistantPrefHandler struct {
	assistantService *services.AIAssistantService
}

// NewTeacherAssistantPrefHandler 创建处理器。
func NewTeacherAssistantPrefHandler(
	assistantSvc *services.AIAssistantService,
) *TeacherAssistantPrefHandler {
	return &TeacherAssistantPrefHandler{
		assistantService: assistantSvc,
	}
}

// putPrefRequest 写偏好请求。
type putPrefRequest struct {
	Subject     string `json:"subject"`
	Grade       string `json:"grade"`
	Stage       string `json:"stage"`
	AssistantID string `json:"assistant_id"`
}

// getPrefResponse 偏好三态响应。
type getPrefResponse struct {
	Subject         string `json:"subject"`
	Grade           string `json:"grade"`
	HasRecord       bool   `json:"has_record"`
	IsSystemDefault bool   `json:"is_system_default"`
	AssistantID     string `json:"assistant_id"`
	AssistantName   string `json:"assistant_name"`
}

// GetPref GET /assistant-prefs?subject=&grade=&stage=
//
// 存量偏好若不适用于本次具体年级或阶段，返回has_record=false，
// 但不删除数据库中的原偏好。
func (h *TeacherAssistantPrefHandler) GetPref(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	query := r.URL.Query()
	subject := strings.TrimSpace(query.Get("subject"))
	grade := strings.TrimSpace(query.Get("grade"))
	scene := stageToAssistantScene(
		strings.TrimSpace(query.Get("stage")),
	)

	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	if grade == "" {
		utils.BadRequest(w, "缺少 grade 参数")
		return
	}

	assistantID, found, err := repository.GetPref(
		r.Context(),
		claims.UserID,
		subject,
	)
	if err != nil {
		utils.InternalError(w, "查询助手偏好失败")
		return
	}

	response := getPrefResponse{
		Subject:   subject,
		Grade:     grade,
		HasRecord: found,
	}

	if !found {
		utils.Success(w, response)
		return
	}

	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		response.IsSystemDefault = true
		utils.Success(w, response)
		return
	}

	actor := services.BuildActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	assistant, validateErr :=
		h.assistantService.ValidateAssistantForLesson(
			r.Context(),
			actor,
			assistantID,
			subject,
			grade,
			scene,
		)
	if validateErr != nil || assistant == nil {
		// 原偏好仍保留在数据库，但本年级/本阶段不生效。
		response.HasRecord = false
		response.AssistantID = ""
		response.AssistantName = ""
		utils.Success(w, response)
		return
	}

	response.AssistantID = assistant.ID
	response.AssistantName = assistant.Name
	utils.Success(w, response)
}

// PutPref PUT /assistant-prefs
//
// 具体助手必须与请求中的学科、具体年级及阶段匹配。
// assistant_id为空仍表示老师明确选择系统默认。
func (h *TeacherAssistantPrefHandler) PutPref(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPut {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持PUT请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var request putPrefRequest
	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}

	subject := strings.TrimSpace(request.Subject)
	grade := strings.TrimSpace(request.Grade)
	assistantID := strings.TrimSpace(
		request.AssistantID,
	)
	scene := stageToAssistantScene(
		strings.TrimSpace(request.Stage),
	)

	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	if grade == "" {
		utils.BadRequest(w, "缺少 grade 参数")
		return
	}

	assistantName := ""
	if assistantID != "" {
		actor := services.BuildActorFromClaims(
			r.Context(),
			claims.UserID,
			claims.Role,
		)

		assistant, err :=
			h.assistantService.ValidateAssistantForLesson(
				r.Context(),
				actor,
				assistantID,
				subject,
				grade,
				scene,
			)
		if err != nil || assistant == nil {
			utils.BadRequest(
				w,
				"选择的助手不适用于当前学科、具体年级或阶段",
			)
			return
		}
		assistantName = assistant.Name
	}

	if err := repository.UpsertPref(
		r.Context(),
		claims.UserID,
		subject,
		assistantID,
	); err != nil {
		utils.InternalError(w, "保存助手偏好失败")
		return
	}

	utils.Success(w, getPrefResponse{
		Subject:         subject,
		Grade:           grade,
		HasRecord:       true,
		IsSystemDefault: assistantID == "",
		AssistantID:     assistantID,
		AssistantName:   assistantName,
	})
}

// GetOptions GET /assistant-options?subject=&grade=&stage=
//
// 只返回同学科、同具体年级、当前场景可用的助手。
func (h *TeacherAssistantPrefHandler) GetOptions(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	query := r.URL.Query()
	subject := strings.TrimSpace(query.Get("subject"))
	grade := strings.TrimSpace(query.Get("grade"))
	scene := stageToAssistantScene(
		strings.TrimSpace(query.Get("stage")),
	)

	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	if grade == "" {
		utils.BadRequest(w, "缺少 grade 参数")
		return
	}

	actor := services.BuildActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)

	response, err := h.assistantService.ListAssistants(
		r.Context(),
		actor,
		scene,
		subject,
		grade,
		true,
	)
	if err != nil {
		utils.InternalError(w, "查询可选助手失败")
		return
	}

	utils.Success(w, response)
}

// stageToAssistantScene 阶段代码转助手场景。
// 空值或自定义阶段暂不额外限制场景，但仍严格校验学科与年级。
func stageToAssistantScene(stageCode string) string {
	switch stageCode {
	case "analyze":
		return models.SceneWorkshopAnalyze
	case "design":
		return models.SceneWorkshopDesign
	case "write":
		return models.SceneWorkshopWrite
	case "review":
		return models.SceneWorkshopReview
	case "revise":
		return models.SceneWorkshopRevise
	default:
		return ""
	}
}
