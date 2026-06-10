package handlers

// courseware_background_handler.go — 课件背景图库处理器（批次1新建，批次3扩展生产入口）
//
// 路由（图库子树统一经 HandleLibrary 分发）：
//   GET    /api/v1/courseware-backgrounds                 — 图集列表（系统+我的，登录即可）
//   POST   /api/v1/courseware-backgrounds/generate        — 批次3：AI生成一套背景（封面+内页）
//   POST   /api/v1/courseware-backgrounds/upload          — 批次3：上传一套背景（multipart两图）
//   DELETE /api/v1/courseware-backgrounds/{id}            — 批次3：归档个人集（本人/admin）
//   POST   /api/v1/courseware-backgrounds/{id}/promote    — 批次3：升级为系统图库（仅admin）
//
// 课件侧（独立挂载，不经 HandleLibrary）：
//   GET /api/v1/coursewares/{id}/background       — 课件当前背景选择
//   PUT /api/v1/coursewares/{id}/background       — 选择({set_id})/清除({clear:true})背景并秒换已生成页

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/config"
	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
)

// CoursewareBackgroundHandler 背景图库处理器
type CoursewareBackgroundHandler struct {
	svc *services.CoursewareBackgroundService
}

// NewCoursewareBackgroundHandler 创建背景图库处理器
// 批次3：服务需要 cfg（AES密钥+OSS配置）；config.Load() 读环境变量幂等无副作用，
// 在此内部加载可保持 routes 注册签名零改动
func NewCoursewareBackgroundHandler() *CoursewareBackgroundHandler {
	cfg := config.Load()
	return &CoursewareBackgroundHandler{svc: services.NewCoursewareBackgroundService(cfg)}
}

// bgWriteJSON 统一JSON响应（与系统 {code,data,message} 格式对齐，自包含不依赖外部签名）
func bgWriteJSON(w http.ResponseWriter, httpStatus int, code int, data interface{}, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": code, "data": data, "message": msg})
}

// bgWriteErr 业务错误统一映射：含"无权"映射403，其余400
func bgWriteErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if strings.Contains(err.Error(), "无权") {
		status = http.StatusForbidden
	}
	bgWriteJSON(w, status, -1, nil, err.Error())
}

// extractBgCoursewareID 从 /api/v1/coursewares/{id}/background 提取课件ID
func extractBgCoursewareID(path string) string {
	p := strings.TrimPrefix(path, "/api/v1/coursewares/")
	if i := strings.Index(p, "/"); i > 0 {
		return p[:i]
	}
	return ""
}

// extractBgSetID 从 /api/v1/courseware-backgrounds/{id}[/suffix] 提取图集ID
func extractBgSetID(path string, suffix string) string {
	p := strings.TrimPrefix(path, "/api/v1/courseware-backgrounds/")
	if suffix != "" {
		p = strings.TrimSuffix(p, suffix)
	}
	return strings.Trim(p, "/")
}

// ==================== 图库子树统一分发（批次3） ====================

// HandleLibrary /api/v1/courseware-backgrounds[/*] 按路径与方法分发
func (h *CoursewareBackgroundHandler) HandleLibrary(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r.Context())
	if claims == nil {
		bgWriteJSON(w, http.StatusUnauthorized, -1, nil, "未登录")
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	// 列表：GET /api/v1/courseware-backgrounds
	case path == "/api/v1/courseware-backgrounds":
		if r.Method != http.MethodGet {
			bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
			return
		}
		h.ListSets(w, r)

	// AI生成一套：POST /api/v1/courseware-backgrounds/generate
	case path == "/api/v1/courseware-backgrounds/generate":
		if r.Method != http.MethodPost {
			bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
			return
		}
		h.handleGenerateSet(w, r, claims.UserID)

	// 上传一套：POST /api/v1/courseware-backgrounds/upload
	case path == "/api/v1/courseware-backgrounds/upload":
		if r.Method != http.MethodPost {
			bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
			return
		}
		h.handleUploadSet(w, r, claims.UserID)

	// 升级系统图库：POST /api/v1/courseware-backgrounds/{id}/promote（仅admin）
	case strings.HasSuffix(path, "/promote"):
		if r.Method != http.MethodPost {
			bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
			return
		}
		if claims.Role != "admin" {
			bgWriteJSON(w, http.StatusForbidden, -1, nil, "仅系统管理员可将图集存为系统图库")
			return
		}
		setID := extractBgSetID(r.URL.Path, "/promote")
		if setID == "" {
			bgWriteJSON(w, http.StatusBadRequest, -1, nil, "无效的图集ID")
			return
		}
		set, err := h.svc.PromoteSet(r.Context(), setID)
		if err != nil {
			bgWriteErr(w, err)
			return
		}
		bgWriteJSON(w, http.StatusOK, 0, set, "已存为系统图库，全体用户可见")

	// 归档删除：DELETE /api/v1/courseware-backgrounds/{id}
	default:
		if r.Method != http.MethodDelete {
			bgWriteJSON(w, http.StatusNotFound, -1, nil, "未找到路由")
			return
		}
		setID := extractBgSetID(r.URL.Path, "")
		if setID == "" || strings.Contains(setID, "/") {
			bgWriteJSON(w, http.StatusBadRequest, -1, nil, "无效的图集ID")
			return
		}
		if err := h.svc.DeleteSet(r.Context(), claims.UserID, claims.Role, setID); err != nil {
			bgWriteErr(w, err)
			return
		}
		bgWriteJSON(w, http.StatusOK, 0, map[string]interface{}{"id": setID}, "图集已删除（已选用该背景的课件不受影响）")
	}
}

// handleGenerateSet 批次3：AI生成一套背景
func (h *CoursewareBackgroundHandler) handleGenerateSet(w http.ResponseWriter, r *http.Request, userID string) {
	var req models.GenerateBackgroundSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		bgWriteJSON(w, http.StatusBadRequest, -1, nil, "请求体解析失败")
		return
	}
	result, err := h.svc.GenerateSet(r.Context(), userID, &req)
	if err != nil {
		bgWriteErr(w, err)
		return
	}
	msg := "背景图集已生成"
	if result.Selection != nil {
		msg = "背景图集已生成并自动应用到本课件"
	}
	bgWriteJSON(w, http.StatusOK, 0, result, msg)
}

// handleUploadSet 批次3：上传一套背景（multipart字段：name/courseware_id/cover/content）
func (h *CoursewareBackgroundHandler) handleUploadSet(w http.ResponseWriter, r *http.Request, userID string) {
	// 两张≤5MB图：内存上限16MB，超出自动落临时盘
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		bgWriteJSON(w, http.StatusBadRequest, -1, nil, "上传内容解析失败: "+err.Error())
		return
	}
	name := r.FormValue("name")
	coursewareID := r.FormValue("courseware_id")
	coverFile, coverHdr, err := r.FormFile("cover")
	if err != nil {
		bgWriteJSON(w, http.StatusBadRequest, -1, nil, "缺少封面图(cover)")
		return
	}
	defer coverFile.Close()
	contentFile, contentHdr, err := r.FormFile("content")
	if err != nil {
		bgWriteJSON(w, http.StatusBadRequest, -1, nil, "缺少内页图(content)")
		return
	}
	defer contentFile.Close()

	result, err := h.svc.UploadSet(r.Context(), userID, name, coursewareID, coverFile, coverHdr, contentFile, contentHdr)
	if err != nil {
		bgWriteErr(w, err)
		return
	}
	msg := "背景图集已上传"
	if result.Selection != nil {
		msg = "背景图集已上传并自动应用到本课件"
	}
	bgWriteJSON(w, http.StatusOK, 0, result, msg)
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
			bgWriteErr(w, err)
			return
		}
		bgWriteJSON(w, http.StatusOK, 0, result, "背景已更新")

	default:
		bgWriteJSON(w, http.StatusMethodNotAllowed, -1, nil, "Method not allowed")
	}
}
