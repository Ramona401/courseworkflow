package handlers

// courseware_collab_handler.go — 课件工坊·集体备课（阶段4）HTTP 处理器
//
// 方法挂在已有的 CoursewareHandler 上（与 courseware_share_handler.go 同结构体），
// 复用其 h.cwService 与 courseware_handler.go 的 extractCoursewareMiddleID 路径解析。
//
// 端点（全部走 cwMux→dispatchCoursewareSubRoutes，登录即可，细粒度权限在 service 层裁决）：
//   POST   /api/v1/coursewares/{id}/collab/start          发起集体备课（仅作者，可选带首批参与者）
//   POST   /api/v1/coursewares/{id}/collab/end            结束集体备课（仅作者）
//   GET    /api/v1/coursewares/{id}/collab                查状态+参与者列表+我能否微调
//   POST   /api/v1/coursewares/{id}/collab/members        加参与者（仅作者，body: {user_id}）
//   DELETE /api/v1/coursewares/{id}/collab/members/{uid}  移除参与者（仅作者）
//
// 集体备课设计（最小增量）：议课走现有页级批注，留痕走现有版本快照，本处理器只管"标记态 + 参与者名单"。

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// StartCollab POST /api/v1/coursewares/{id}/collab/start — 发起集体备课
func (h *CoursewareHandler) StartCollab(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/collab/start")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// body 可选：{members:[...]}（首批参与者）。允许空 body（只发起不拉人）。
	var req models.StartCollabRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // 解析失败按空处理，不阻断
	}

	if err := h.cwService.StartCollab(r.Context(), id, claims.UserID, req.Members); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已发起集体备课"})
}

// EndCollab POST /api/v1/coursewares/{id}/collab/end — 结束集体备课
func (h *CoursewareHandler) EndCollab(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/collab/end")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	if err := h.cwService.EndCollab(r.Context(), id, claims.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已结束集体备课"})
}

// GetCollabStatus GET /api/v1/coursewares/{id}/collab — 查集体备课状态
func (h *CoursewareHandler) GetCollabStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	// 这里 path 形如 .../{id}/collab（无更深后缀），用 TrimSuffix 取 id
	id := extractCollabCoursewareID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	resp, err := h.cwService.GetCollabStatus(r.Context(), id, claims.UserID, claims.Role)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// AddCollabMember POST /api/v1/coursewares/{id}/collab/members — 加参与者
func (h *CoursewareHandler) AddCollabMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/collab/members")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	var req models.AddCollabMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		utils.BadRequest(w, "参与者用户ID不能为空")
		return
	}
	if err := h.cwService.AddCollabMember(r.Context(), id, claims.UserID, req.UserID); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已添加参与者"})
}

// RemoveCollabMember DELETE /api/v1/coursewares/{id}/collab/members/{uid} — 移除参与者
func (h *CoursewareHandler) RemoveCollabMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	// path: /api/v1/coursewares/{id}/collab/members/{uid}
	//   id  = /collab/members/ 之前的中间段
	//   uid = path 末段
	id := extractCoursewareMiddleID(r.URL.Path, "/collab/members/")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	uid := path.Base(r.URL.Path)
	if uid == "" || uid == "members" {
		utils.BadRequest(w, "缺少参与者用户ID")
		return
	}
	if err := h.cwService.RemoveCollabMember(r.Context(), id, claims.UserID, uid); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "已移除参与者"})
}

// extractCollabCoursewareID 从 /api/v1/coursewares/{id}/collab[/] 提取课件ID（GET 状态查询用）。
// 与 extractCoursewareMiddleID 不同：这里 /collab 是 path 末段（后面没有更多内容），
// 故用 TrimPrefix + TrimSuffix 取中间的 id。
func extractCollabCoursewareID(p string) string {
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(p, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(p, prefix)
	rest = strings.TrimSuffix(rest, "/")     // 去掉可能的尾斜杠
	rest = strings.TrimSuffix(rest, "/collab") // 去掉 /collab 末段
	// 此时 rest 应当就是纯 id（不含更多斜杠）
	if rest == "" || strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

// ListCollabCandidates GET /api/v1/coursewares/collab/candidates — 列候选成员（同校同组）
// 集合级路径（不带课件ID），供"加参与者"下拉选人。任何登录者可查（只返回与自己同校同组的人）。
func (h *CoursewareHandler) ListCollabCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	resp, err := h.cwService.ListCollabCandidates(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}

// ListJoinedCollab GET /api/v1/coursewares/collab/joined — 我参与的集体备课
// 集合级路径（不带课件ID）。列出当前用户被拉入、且仍在 in_session 的课件，供参与者入口。
func (h *CoursewareHandler) ListJoinedCollab(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	resp, err := h.cwService.ListJoinedCollab(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, resp)
}
