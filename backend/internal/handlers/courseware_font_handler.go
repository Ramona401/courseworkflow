package handlers

// courseware_font_handler.go — 课件字体方案处理器（字体F1新建）
//
// 路由：
//   GET /api/v1/courseware-fonts          — 字体方案列表（5套系统预设，登录即可）
//   GET /api/v1/coursewares/{id}/font     — 课件当前字体选择
//   PUT /api/v1/coursewares/{id}/font     — 选择({scheme_code})/清除({clear:true})字体并秒换已生成页
//
// 复用 courseware_background_handler.go 同包的 bgWriteJSON / bgWriteErr / extractBgCoursewareID
//（同响应格式 {code,data,message}，路径结构同为 /coursewares/{id}/xxx，避免重复造轮子）。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
)

// CoursewareFontHandler 字体方案处理器
type CoursewareFontHandler struct {
	svc *services.CoursewareFontService
}

// NewCoursewareFontHandler 创建字体方案处理器（服务无状态零依赖）
func NewCoursewareFontHandler() *CoursewareFontHandler {
	return &CoursewareFontHandler{svc: services.NewCoursewareFontService()}
}

// ListSchemes GET /api/v1/courseware-fonts — 字体方案列表
// 响应附 base_url：前端用 base_url + faces[].file 现场构建 @font-face 渲染各方案真实字样预览
func (h *CoursewareFontHandler) ListSchemes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
		return
	}
	claims, _ := middleware.GetClaims(r.Context())
	if claims == nil {
		bgWriteJSON(w, http.StatusUnauthorized, -1, nil, "未登录")
		return
	}
	schemes := h.svc.ListSchemes()
	bgWriteJSON(w, http.StatusOK, 0, map[string]interface{}{
		"schemes":  schemes,
		"total":    len(schemes),
		"base_url": services.CWFontBaseURL,
	}, "")
}

// HandleCoursewareFont /api/v1/coursewares/{id}/font 按方法分发
func (h *CoursewareFontHandler) HandleCoursewareFont(w http.ResponseWriter, r *http.Request) {
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
			bgWriteJSON(w, http.StatusNotFound, -1, nil, "查询课件字体失败: "+err.Error())
			return
		}
		bgWriteJSON(w, http.StatusOK, 0, sel, "")

	case http.MethodPut:
		var req models.SelectCoursewareFontRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			bgWriteJSON(w, http.StatusBadRequest, -1, nil, "请求体解析失败")
			return
		}
		if !req.Clear && strings.TrimSpace(req.SchemeCode) == "" {
			bgWriteJSON(w, http.StatusBadRequest, -1, nil, "请提供 scheme_code 或 clear:true")
			return
		}
		var result *models.FontSelectionResult
		var err error
		if req.Clear {
			result, err = h.svc.ClearFont(r.Context(), cwID, claims.UserID)
		} else {
			result, err = h.svc.SelectFont(r.Context(), cwID, claims.UserID, req.SchemeCode)
		}
		if err != nil {
			bgWriteErr(w, err)
			return
		}
		bgWriteJSON(w, http.StatusOK, 0, result, "字体已更新")

	default:
		bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
	}
}
