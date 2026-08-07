package handlers

// courseware_page_source_handler.go
//
// 课件页面源码保存与HTML导入处理器。
//
// 本文件由courseware_gen_handler.go按页面源码写入职责拆出，
// 保持既有HTTP路径、作者权限和页面版本快照行为不变。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// SavePageHTML POST /api/v1/coursewares/{id}/pages/{num}/save-html。
func (h *CoursewareGenHandler) SavePageHTML(
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

	coursewareID, pageNumber := extractCWPageSaveHTMLPath(
		r.URL.Path,
	)
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
		HTMLContent string `json:"html_content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if strings.TrimSpace(req.HTMLContent) == "" {
		utils.BadRequest(w, "编辑后的内容为空，未保存")
		return
	}

	if len(req.HTMLContent) > services.CoursewarePageHTMLMaxBytes {
		utils.BadRequest(w, "页面内容过大，无法保存")
		return
	}

	result, err := h.genService.SaveManualEditedPage(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNumber,
		req.HTMLContent,
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
				"第%d页修改已保存",
				pageNumber,
			),
		},
	)
}

// ImportPageHTML POST /api/v1/coursewares/{id}/pages/{num}/import-html。
func (h *CoursewareGenHandler) ImportPageHTML(
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

	coursewareID, pageNumber := extractCWPageImportHTMLPath(
		r.URL.Path,
	)
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
		HTMLContent string `json:"html_content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if strings.TrimSpace(req.HTMLContent) == "" {
		utils.BadRequest(w, "粘贴的内容为空，未导入")
		return
	}

	if len(req.HTMLContent) > services.CoursewarePageHTMLMaxBytes {
		utils.BadRequest(w, "粘贴的内容过大，无法导入")
		return
	}

	result, err := h.genService.ImportPageHTML(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNumber,
		req.HTMLContent,
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
				"第%d页HTML导入完成",
				pageNumber,
			),
		},
	)
}
