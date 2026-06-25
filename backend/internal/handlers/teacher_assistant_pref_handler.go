package handlers

// teacher_assistant_pref_handler.go — 老师×学科 AI 助手选择偏好 HTTP 处理器
//
// 对话式备课·助手轻量选择入口 PRD §3 / §5 / §7(Phase 1)
//
// 接口清单：
//   GET  /api/v1/lesson-plans/assistant-prefs?subject=语文              读当前老师在某学科的助手偏好(三态)
//   PUT  /api/v1/lesson-plans/assistant-prefs                          写偏好(body:{subject,assistant_id})
//   GET  /api/v1/lesson-plans/assistant-options?subject=语文&stage=design  列出该学科(该阶段)的可选助手
//
// 设计要点：
//   - 读偏好返回三态(has_record / is_system_default / assistant_id)供前端面板高亮当前生效项。
//   - 写偏好时 assistant_id 空串是合法值(= 显式选择「系统默认」纯骨架);非空时校验该助手对当前
//     老师可见(防止把偏好指向看不到/不存在的助手写脏数据)。
//   - 可选助手列表复用 AIAssistantService.ListAssistants，口径与 skill_router 的 RouteDefaultAssistant
//     完全一致(同样的 场景+学科+可见性 过滤),保证「面板列出的」与「实际会被注入的」是同一批助手。
//   - 取当前用户走 middleware.GetClaims(拿 UserID + Role)，因构造助手可见性 actor 需要 Role，
//     而 getCurrentUserID 只给 UserID。

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

// TeacherAssistantPrefHandler 老师×学科助手偏好处理器
type TeacherAssistantPrefHandler struct {
	assistantService *services.AIAssistantService // 复用:可选助手列表 + 写偏好时的可见性校验
}

// NewTeacherAssistantPrefHandler 创建偏好处理器实例
func NewTeacherAssistantPrefHandler(assistantSvc *services.AIAssistantService) *TeacherAssistantPrefHandler {
	return &TeacherAssistantPrefHandler{assistantService: assistantSvc}
}

// ==================== 请求/响应结构 ====================

// putPrefRequest PUT 写偏好请求体
type putPrefRequest struct {
	Subject     string `json:"subject"`
	AssistantID string `json:"assistant_id"` // 空串=显式选择「系统默认」(纯骨架)
}

// getPrefResponse GET 读偏好响应(三态)
type getPrefResponse struct {
	Subject         string `json:"subject"`
	HasRecord       bool   `json:"has_record"`        // false=从没选过(走学科推荐兜底)
	IsSystemDefault bool   `json:"is_system_default"` // true=显式选了系统默认(纯骨架);仅 HasRecord=true 时有意义
	AssistantID     string `json:"assistant_id"`      // 选定的助手ID;HasRecord=false 或 IsSystemDefault=true 时为空
	AssistantName   string `json:"assistant_name"`    // 选定助手的显示名(仅 AssistantID 非空时填;助手已删则留空)
}

// ==================== GET /assistant-prefs — 读偏好(三态) ====================

func (h *TeacherAssistantPrefHandler) GetPref(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	subject := strings.TrimSpace(r.URL.Query().Get("subject"))
	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}

	assistantID, found, err := repository.GetPref(r.Context(), claims.UserID, subject)
	if err != nil {
		utils.InternalError(w, "查询助手偏好失败")
		return
	}

	resp := getPrefResponse{
		Subject:   subject,
		HasRecord: found,
	}
	if found {
		trimmed := strings.TrimSpace(assistantID)
		if trimmed == "" {
			// 有记录但空串 = 显式系统默认(纯骨架)
			resp.IsSystemDefault = true
			resp.AssistantID = ""
		} else {
			resp.AssistantID = trimmed
			// 助手轻量选择入口 Phase 1 小优化:回填助手真名,供前端指示器直接展示。
			// 查名失败(如助手已被删/停用)不阻塞,留空由前端 fallback 文案兜底。
			actor := services.BuildActorFromClaims(r.Context(), claims.UserID, claims.Role)
			if a, aerr := h.assistantService.GetAssistant(r.Context(), actor, trimmed); aerr == nil && a != nil {
				resp.AssistantName = a.Name
			}
		}
	}
	utils.Success(w, resp)
}

// ==================== PUT /assistant-prefs — 写偏好 ====================

func (h *TeacherAssistantPrefHandler) PutPref(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	var req putPrefRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	assistantID := strings.TrimSpace(req.AssistantID)

	// assistant_id 非空时:校验该助手对当前老师可见(防止写入看不到/不存在的助手)。
	// 空串是合法的「显式系统默认」,无需校验,直接落库。
	if assistantID != "" {
		actor := services.BuildActorFromClaims(r.Context(), claims.UserID, claims.Role)
		if _, err := h.assistantService.GetAssistant(r.Context(), actor, assistantID); err != nil {
			// 助手不存在 / 无权查看 → 视为非法选择,不写脏数据。
			utils.BadRequest(w, "选择的助手不存在或当前账号无权使用")
			return
		}
	}

	if err := repository.UpsertPref(r.Context(), claims.UserID, subject, assistantID); err != nil {
		utils.InternalError(w, "保存助手偏好失败")
		return
	}

	// 回显写入结果(三态),前端据此即时更新面板高亮。
	resp := getPrefResponse{
		Subject:         subject,
		HasRecord:       true,
		IsSystemDefault: assistantID == "",
		AssistantID:     assistantID,
	}
	utils.Success(w, resp)
}

// ==================== GET /assistant-options — 列出可选助手 ====================

func (h *TeacherAssistantPrefHandler) GetOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}

	q := r.URL.Query()
	subject := strings.TrimSpace(q.Get("subject"))
	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	// stage 可选:传了就按对应助手场景过滤(与 RouteDefaultAssistant 同口径);
	// 没传或无法识别 → scene 为空 → 不限场景,列出该学科所有可见助手(更宽容)。
	scene := stageToAssistantScene(strings.TrimSpace(q.Get("stage")))

	actor := services.BuildActorFromClaims(r.Context(), claims.UserID, claims.Role)
	// onlyActive=true:面板只列启用的助手,与默认助手解析一致(停用的不参与)。
	// gradeRange 传空:面板按学科维度列出即可,不按年级进一步收窄(PRD §3.3「只列与当前学科相关的助手」)。
	resp, err := h.assistantService.ListAssistants(r.Context(), actor, scene, subject, "", true)
	if err != nil {
		utils.InternalError(w, "查询可选助手失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 辅助 ====================

// stageToAssistantScene 阶段代码 → AI 助手场景常量映射(handler 侧自包含)。
//
// 与 services/skill_router.go 的 stageCodeToAssistantScene 口径逐字一致(那是 services 包私有函数,
// 跨包调不到,故此处自带一份 5 行映射,避免为一个映射去改 service 增加耦合)。
// 不可识别 / 空串 → 返回空串(ListAssistants 收到空 scene 即不按场景过滤)。
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
