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
func (h *CoursewareBackgroundHandler) HandleCoursewareBackground(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		bgWriteJSON(
			w,
			http.StatusUnauthorized,
			-1,
			nil,
			"未登录",
		)
		return
	}

	coursewareID :=
		extractBgCoursewareID(r.URL.Path)
	if coursewareID == "" {
		bgWriteJSON(
			w,
			http.StatusBadRequest,
			-1,
			nil,
			"无效的课件ID",
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if err := authorizeCoursewareViewForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		); err != nil {
			handleCoursewareAccessError(
				w,
				err,
				"查询课件背景失败",
			)
			return
		}

		selection, err := h.svc.GetSelection(
			r.Context(),
			coursewareID,
		)
		if err != nil {
			bgWriteJSON(
				w,
				http.StatusNotFound,
				-1,
				nil,
				"查询课件背景失败: "+err.Error(),
			)
			return
		}

		bgWriteJSON(
			w,
			http.StatusOK,
			0,
			selection,
			"",
		)

	case http.MethodPut:
		actor, err :=
			authorizeCoursewareOwnerRuntimeForHandler(
				r.Context(),
				coursewareID,
				claims.UserID,
				claims.Role,
			)
		if err != nil {
			writeCoursewareControlError(w, err)
			return
		}

		var req models.SelectCoursewareBackgroundRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			bgWriteJSON(
				w,
				http.StatusBadRequest,
				-1,
				nil,
				"请求体解析失败",
			)
			return
		}

		if !req.Clear &&
			strings.TrimSpace(req.SetID) == "" {
			bgWriteJSON(
				w,
				http.StatusBadRequest,
				-1,
				nil,
				"请提供 set_id 或 clear:true",
			)
			return
		}

		var result *models.BackgroundSelectionResult
		if req.Clear {
			result, err = h.svc.ClearBackgroundForActor(
				r.Context(),
				coursewareID,
				actor,
			)
		} else {
			result, err = h.svc.SelectBackgroundForActor(
				r.Context(),
				coursewareID,
				actor,
				req.SetID,
			)
		}
		if err != nil {
			writeCoursewareControlError(w, err)
			return
		}

		bgWriteJSON(
			w,
			http.StatusOK,
			0,
			result,
			"背景已更新",
		)

	default:
		bgWriteJSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			nil,
			"Method not allowed",
		)
	}
}

// ==================== 页级背景覆盖端点 ====================

// HandlePageBackground 页级背景设置——GET/PUT/DELETE /api/v1/coursewares/{id}/pages/{num}/page-bg
func (h *CoursewareBackgroundHandler) HandlePageBackground(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		bgWriteJSON(
			w,
			http.StatusUnauthorized,
			-1,
			nil,
			"未登录",
		)
		return
	}

	coursewareID, pageNumber :=
		extractPageBgPath(r.URL.Path)
	if coursewareID == "" || pageNumber <= 0 {
		bgWriteJSON(
			w,
			http.StatusBadRequest,
			-1,
			nil,
			"无效的课件ID或页码",
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if err := authorizeCoursewareViewForHandler(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		); err != nil {
			handleCoursewareAccessError(
				w,
				err,
				"查询页级背景失败",
			)
			return
		}

		setting, err := h.svc.GetPageBackground(
			r.Context(),
			coursewareID,
			pageNumber,
		)
		if err != nil {
			bgWriteJSON(
				w,
				http.StatusNotFound,
				-1,
				nil,
				"查询页级背景失败: "+err.Error(),
			)
			return
		}

		bgWriteJSON(
			w,
			http.StatusOK,
			0,
			setting,
			"",
		)

	case http.MethodPut:
		actor, err :=
			authorizeCoursewareOwnerRuntimeForHandler(
				r.Context(),
				coursewareID,
				claims.UserID,
				claims.Role,
			)
		if err != nil {
			writeCoursewareControlError(w, err)
			return
		}

		var req PageBgRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			bgWriteJSON(
				w,
				http.StatusBadRequest,
				-1,
				nil,
				"请求体解析失败",
			)
			return
		}

		result, err := h.svc.SetPageBackgroundForActor(
			r.Context(),
			coursewareID,
			actor,
			pageNumber,
			req.URL,
			req.Opacity,
			req.Mode,
		)
		if err != nil {
			writeCoursewareControlError(w, err)
			return
		}

		bgWriteJSON(
			w,
			http.StatusOK,
			0,
			result,
			"页级背景已设置",
		)

	case http.MethodDelete:
		actor, err :=
			authorizeCoursewareOwnerRuntimeForHandler(
				r.Context(),
				coursewareID,
				claims.UserID,
				claims.Role,
			)
		if err != nil {
			writeCoursewareControlError(w, err)
			return
		}

		result, err := h.svc.ClearPageBackgroundForActor(
			r.Context(),
			coursewareID,
			actor,
			pageNumber,
		)
		if err != nil {
			writeCoursewareControlError(w, err)
			return
		}

		bgWriteJSON(
			w,
			http.StatusOK,
			0,
			result,
			"页级背景已清除",
		)

	default:
		bgWriteJSON(
			w,
			http.StatusMethodNotAllowed,
			-1,
			nil,
			"Method not allowed",
		)
	}
}

// PageBgRequest 页级背景设置请求
type PageBgRequest struct {
	URL     string   `json:"url"`     // 页级背景图URL（空=仅改蒙版模式不换图）
	Opacity *float64 `json:"opacity"` // 蒙版透明度 0.0~1.0（null=跟随默认）
	Mode    string   `json:"mode"`    // default/custom/none
}

// extractPageBgPath 从 /api/v1/coursewares/{cwID}/pages/{num}/page-bg 提取课件ID和页码
func extractPageBgPath(path string) (string, int) {
	// 格式: /api/v1/coursewares/{cwID}/pages/{num}/page-bg
	p := strings.TrimPrefix(path, "/api/v1/coursewares/")
	// p = "{cwID}/pages/{num}/page-bg"
	parts := strings.Split(p, "/")
	if len(parts) < 4 || parts[1] != "pages" {
		return "", 0
	}
	cwID := parts[0]
	pageNum := 0
	for _, c := range parts[2] {
		if c >= '0' && c <= '9' {
			pageNum = pageNum*10 + int(c-'0')
		} else {
			return "", 0
		}
	}
	return cwID, pageNum
}
