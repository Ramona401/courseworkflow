package handlers

// subject_handler.go — 学科字典处理器
//
// 两组端点：
//   公开只读（登录即可）：GET /api/v1/subjects            → 启用学科列表（前端各下拉消费）
//   管理 CRUD（admin）  ：GET    /api/v1/admin/subjects    → 全部学科（含停用）
//                        POST   /api/v1/admin/subjects    → 新建
//                        PUT    /api/v1/admin/subjects/{id}→ 编辑
//                        DELETE /api/v1/admin/subjects/{id}→ 删除（内置学科禁删）
//
// 权限：admin 判定由路由层 adminOnly 中间件兜（对齐 kb_admin_handler），handler 不重复判 Role。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// SubjectHandler 学科字典处理器
type SubjectHandler struct{}

// NewSubjectHandler 创建学科处理器
func NewSubjectHandler() *SubjectHandler {
	return &SubjectHandler{}
}

// ListPublic GET /api/v1/subjects — 公开只读：启用学科列表，供全平台下拉消费
func (h *SubjectHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	items, err := repository.ListActiveSubjects(r.Context())
	if err != nil {
		utils.InternalError(w, "查询学科列表失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"subjects": items, "total": len(items)})
}

// ListAdmin GET /api/v1/admin/subjects — admin：全部学科（含停用）
func (h *SubjectHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	items, err := repository.ListAllSubjects(r.Context())
	if err != nil {
		utils.InternalError(w, "查询学科列表失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"subjects": items, "total": len(items)})
}

// Create POST /api/v1/admin/subjects — admin：新建学科
func (h *SubjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	if req.Name == "" {
		utils.BadRequest(w, "缺少学科名")
		return
	}
	s, err := repository.CreateSubject(r.Context(), &req, claims.UserID)
	if err != nil {
		if err == repository.ErrSubjectNameExists {
			utils.BadRequest(w, "学科名已存在")
			return
		}
		utils.InternalError(w, "新建学科失败: "+err.Error())
		return
	}
	utils.Success(w, s)
}

// Update PUT /api/v1/admin/subjects/{id} — admin：编辑学科
func (h *SubjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	id := extractSubjectID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少学科ID")
		return
	}
	var req models.UpdateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	// 若传了 name，去空格并校验非空
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			utils.BadRequest(w, "学科名不能为空")
			return
		}
		req.Name = &trimmed
	}
	if req.Code != nil {
		trimmed := strings.TrimSpace(*req.Code)
		req.Code = &trimmed
	}
	s, err := repository.UpdateSubject(r.Context(), id, &req, claims.UserID)
	if err != nil {
		switch err {
		case repository.ErrSubjectNotFound:
			utils.Fail(w, http.StatusNotFound, "学科不存在")
		case repository.ErrSubjectNameExists:
			utils.BadRequest(w, "学科名已存在")
		default:
			utils.InternalError(w, "编辑学科失败: "+err.Error())
		}
		return
	}
	utils.Success(w, s)
}

// Delete DELETE /api/v1/admin/subjects/{id} — admin：删除学科（内置学科禁删）
func (h *SubjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodDeleteOnly)
		return
	}
	id := extractSubjectID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少学科ID")
		return
	}
	if err := repository.DeleteSubject(r.Context(), id); err != nil {
		switch err {
		case repository.ErrSubjectNotFound:
			utils.Fail(w, http.StatusNotFound, "学科不存在")
		case repository.ErrSubjectSystemGuard:
			utils.BadRequest(w, "内置学科不可删除，如需隐藏请改为停用")
		default:
			utils.InternalError(w, "删除学科失败: "+err.Error())
		}
		return
	}
	utils.Success(w, map[string]string{"message": "已删除"})
}

// extractSubjectID 从 /api/v1/admin/subjects/{id} 取尾段 id
func extractSubjectID(path string) string {
	p := strings.TrimRight(path, "/")
	idx := strings.LastIndex(p, "/")
	if idx < 0 || idx == len(p)-1 {
		return ""
	}
	return p[idx+1:]
}
