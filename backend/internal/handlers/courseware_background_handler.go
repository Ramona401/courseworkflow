package handlers

// courseware_background_handler.go — 课件背景图库处理器（批次1新建）
//
// 路由：
//   GET /api/v1/courseware-backgrounds            — 图集列表（系统+我的，登录即可）
//   GET /api/v1/coursewares/{id}/background       — 课件当前背景选择
//   PUT /api/v1/coursewares/{id}/background       — 选择({set_id})/清除({clear:true})背景并秒换已生成页

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
)

// CoursewareBackgroundHandler 背景图库处理器
type CoursewareBackgroundHandler struct {
	svc *services.CoursewareBackgroundService
}

// NewCoursewareBackgroundHandler 创建背景图库处理器
func NewCoursewareBackgroundHandler() *CoursewareBackgroundHandler {
	return &CoursewareBackgroundHandler{svc: services.NewCoursewareBackgroundService()}
}

// bgWriteJSON 统一JSON响应（与系统 {code,data,message} 格式对齐，自包含不依赖外部签名）
func bgWriteJSON(w http.ResponseWriter, httpStatus int, code int, data interface{}, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": code, "data": data, "message": msg})
}

// extractBgCoursewareID 从 /api/v1/coursewares/{id}/background 提取课件ID
func extractBgCoursewareID(path string) string {
	p := strings.TrimPrefix(path, "/api/v1/coursewares/")
	if i := strings.Index(p, "/"); i > 0 {
		return p[:i]
	}
	return ""
}

// ListSets GET /api/v1/courseware-backgrounds — 图集列表
func (h *CoursewareBackgroundHandler) ListSets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	if claims == nil {
		bgWriteJSON(w, http.StatusUnauthorized, -1, nil, "未登录")
		return
	}
	sets, err := h.svc.ListSets(r.Context(), claims.UserID)
	if err != nil {
		bgWriteJSON(w, http.StatusInternalServerError, -1, nil, "查询背景图集失败: "+err.Error())
		return
	}
	if sets == nil {
		sets = []*models.CoursewareBackgroundSet{}
	}
	bgWriteJSON(w, http.StatusOK, 0, map[string]interface{}{"sets": sets, "total": len(sets)}, "")
}

// HandleCoursewareBackground /api/v1/coursewares/{id}/background 按方法分发
func (h *CoursewareBackgroundHandler) HandleCoursewareBackground(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())
	if claims == nil {
		bgWriteJSON(w, http.StatusUnauthorized, -1, nil, "未登录")
		return
	}
	cwID := extractBgCoursewareID(r.URL.Path)
	if cwID == "" {
		bgWriteJSON(w, http.StatusBadRequest, -1, nil, "无效的课件ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		sel, err := h.svc.GetSelection(r.Context(), cwID)
		if err != nil {
			bgWriteJSON(w, http.StatusNotFound, -1, nil, "查询课件背景失败: "+err.Error())
			return
		}
		bgWriteJSON(w, http.StatusOK, 0, sel, "")

	case http.MethodPut:
		var req models.SelectCoursewareBackgroundRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			bgWriteJSON(w, http.StatusBadRequest, -1, nil, "请求体解析失败")
			return
		}
		if !req.Clear && strings.TrimSpace(req.SetID) == "" {
			bgWriteJSON(w, http.StatusBadRequest, -1, nil, "请提供 set_id 或 clear:true")
			return
		}
		var result *models.BackgroundSelectionResult
		var err error
		if req.Clear {
			result, err = h.svc.ClearBackground(r.Context(), cwID, claims.UserID)
		} else {
			result, err = h.svc.SelectBackground(r.Context(), cwID, claims.UserID, req.SetID)
		}
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "无权") {
				status = http.StatusForbidden
			}
			bgWriteJSON(w, status, -1, nil, err.Error())
			return
		}
		bgWriteJSON(w, http.StatusOK, 0, result, "背景已更新")

	default:
		bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
	}
}
