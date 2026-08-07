package handlers

// courseware_page_mutation_handler.go
//
// 课件页级路径解析、历史版本读取与页面回退处理器。
//
// 单页AI微调和重生位于courseware_page_refine_handler.go；
// 源码保存和HTML导入位于courseware_page_source_handler.go。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/utils"
)

// ==================== 路径解析辅助函数 ====================

// extractCWPageRefinePath 从 /api/v1/coursewares/{id}/pages/{num}/refine 提取课件ID和页码
func extractCWPageRefinePath(path string) (string, int) {
	return extractCWPageActionPath(path, "/refine")
}

// extractCWPageRegeneratePath 从 /api/v1/coursewares/{id}/pages/{num}/regenerate 提取课件ID和页码
func extractCWPageRegeneratePath(path string) (string, int) {
	return extractCWPageActionPath(path, "/regenerate")
}

// extractCWPageSaveHTMLPath 从 /api/v1/coursewares/{id}/pages/{num}/save-html 提取课件ID和页码
func extractCWPageSaveHTMLPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/save-html")
}

// extractCWPageImportHTMLPath 从 /api/v1/coursewares/{id}/pages/{num}/import-html 提取课件ID和页码
func extractCWPageImportHTMLPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/import-html")
}

// extractCWPageActionPath 从页级动作路径中提取课件ID和页码。
func extractCWPageActionPath(path string, action string) (string, int) {
	if !strings.HasSuffix(path, action) {
		return "", 0
	}

	trimmed := strings.TrimSuffix(path, action)
	pagesIndex := strings.LastIndex(trimmed, "/pages/")
	if pagesIndex < 0 {
		return "", 0
	}

	numberText := trimmed[pagesIndex+len("/pages/"):]
	pageNumber, err := strconv.Atoi(numberText)
	if err != nil || pageNumber <= 0 {
		return "", 0
	}

	prefix := trimmed[:pagesIndex]
	const coursewarePrefix = "/api/v1/coursewares/"

	if !strings.HasPrefix(prefix, coursewarePrefix) {
		return "", 0
	}

	coursewareID := prefix[len(coursewarePrefix):]
	if coursewareID == "" {
		return "", 0
	}

	return coursewareID, pageNumber
}

// ==================== 页面级版本与回退 ====================

// ListPageVersions GET /api/v1/coursewares/{id}/pages/{num}/versions
// 返回该页的版本列表，按version_no倒序，且不返回html_content。
func (h *CoursewareGenHandler) ListPageVersions(
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
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber := extractCWPageVersionsPath(r.URL.Path)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err := h.authorizeCoursewareOwnerRuntime(
		r.Context(),
		coursewareID,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	items, err := h.genService.ListCWPageVersions(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNumber,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	list := make([]map[string]interface{}, 0, len(items))

	for _, item := range items {
		label := models.CWPageVersionSourceNameMap[item.Source]
		if label == "" {
			label = item.Source
		}

		list = append(
			list,
			map[string]interface{}{
				"id":           item.ID,
				"version_no":   item.VersionNo,
				"source":       item.Source,
				"source_label": label,
				"note":         item.Note,
				"created_at":   item.CreatedAt,
			},
		)
	}

	utils.Success(
		w,
		map[string]interface{}{
			"page_number": pageNumber,
			"versions":    list,
			"total":       len(list),
		},
	)
}

// GetPageVersionDetail GET /api/v1/coursewares/{id}/pages/{num}/versions/{versionId}
func (h *CoursewareGenHandler) GetPageVersionDetail(
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
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber, versionID := extractCWPageVersionDetailPath(
		r.URL.Path,
	)
	if coursewareID == "" || pageNumber <= 0 || versionID == "" {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err := h.authorizeCoursewareOwnerRuntime(
		r.Context(),
		coursewareID,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	htmlContent, versionNumber, source, err := h.genService.GetCWPageVersionHTML(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNumber,
		versionID,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	label := models.CWPageVersionSourceNameMap[source]
	if label == "" {
		label = source
	}

	utils.Success(
		w,
		map[string]interface{}{
			"page_number":  pageNumber,
			"version_id":   versionID,
			"version_no":   versionNumber,
			"source":       source,
			"source_label": label,
			"html_content": htmlContent,
		},
	)
}

// RollbackPage POST /api/v1/coursewares/{id}/pages/{num}/rollback
func (h *CoursewareGenHandler) RollbackPage(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber := extractCWPageRollbackPath(r.URL.Path)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err := h.authorizeCoursewareOwnerRuntime(
		r.Context(),
		coursewareID,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	var req struct {
		VersionID string `json:"version_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	versionID := strings.TrimSpace(req.VersionID)
	if versionID == "" {
		utils.BadRequest(w, "缺少目标版本ID")
		return
	}

	result, err := h.genService.RollbackCWPage(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNumber,
		versionID,
	)
	if err != nil {
		writeCoursewareOwnerRuntimeError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"page_number":  pageNumber,
			"html_content": result,
			"message": fmt.Sprintf(
				"第%d页已回退",
				pageNumber,
			),
		},
	)
}

// extractCWPageVersionsPath 提取页面版本列表路径。
func extractCWPageVersionsPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/versions")
}

// extractCWPageRollbackPath 提取页面回退路径。
func extractCWPageRollbackPath(path string) (string, int) {
	return extractCWPageActionPath(path, "/rollback")
}

// extractCWPageVersionDetailPath 提取课件ID、页码和版本ID。
func extractCWPageVersionDetailPath(
	path string,
) (
	coursewareID string,
	pageNumber int,
	versionID string,
) {
	const marker = "/versions/"

	versionIndex := strings.LastIndex(path, marker)
	if versionIndex < 0 {
		return "", 0, ""
	}

	versionID = strings.TrimSuffix(
		path[versionIndex+len(marker):],
		"/",
	)
	if versionID == "" {
		return "", 0, ""
	}

	front := path[:versionIndex]
	pagesIndex := strings.LastIndex(front, "/pages/")
	if pagesIndex < 0 {
		return "", 0, ""
	}

	numberText := front[pagesIndex+len("/pages/"):]
	parsedNumber, err := strconv.Atoi(numberText)
	if err != nil || parsedNumber <= 0 {
		return "", 0, ""
	}

	prefix := front[:pagesIndex]
	const coursewarePrefix = "/api/v1/coursewares/"

	if !strings.HasPrefix(prefix, coursewarePrefix) {
		return "", 0, ""
	}

	coursewareID = prefix[len(coursewarePrefix):]
	if coursewareID == "" {
		return "", 0, ""
	}

	return coursewareID, parsedNumber, versionID
}
