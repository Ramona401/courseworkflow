package handlers

// courseware_page_refine_handler.go
//
// 课件单页AI微调与重新生成处理器。
//
// 整改项页面应用请求必须同时提交：
//   - review_item_id；
//   - instruction_version_id。
//
// 后端不会信任浏览器声明的版本状态或页面哈希。
// Begin事务会返回可信page_id、页码和页面HTML哈希守卫，
// Handler必须把该守卫传入RefinePageWithModeGuarded。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// RefinePage POST /api/v1/coursewares/{id}/pages/{num}/refine。
func (h *CoursewareGenHandler) RefinePage(
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

	coursewareID, pageNumber := extractCWPageRefinePath(r.URL.Path)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	// 截图正文可能很大，必须在解析请求正文前完成微调授权。
	scopedActor, err := h.authorizeCoursewareRefineForHandler(
		r.Context(),
		coursewareID,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareRefineError(w, err)
		return
	}

	var req struct {
		Instruction          string `json:"instruction"`
		Image                string `json:"image"`
		Mode                 string `json:"mode"`
		ReviewItemID         string `json:"review_item_id"`
		InstructionVersionID string `json:"instruction_version_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	instruction := strings.TrimSpace(req.Instruction)
	image := strings.TrimSpace(req.Image)
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	reviewItemID := strings.TrimSpace(req.ReviewItemID)
	instructionVersionID := strings.TrimSpace(req.InstructionVersionID)

	if mode == "" {
		mode = "preserve"
	}
	if mode != "preserve" && mode != "rebuild" {
		utils.BadRequest(
			w,
			"修改模式无效，仅支持preserve或rebuild",
		)
		return
	}

	if instruction == "" && image == "" {
		utils.BadRequest(w, "请提供修改意见或粘贴截图")
		return
	}

	if reviewItemID != "" && instructionVersionID == "" {
		utils.BadRequest(
			w,
			"应用整改项时缺少instruction_version_id",
		)
		return
	}
	if reviewItemID == "" && instructionVersionID != "" {
		utils.BadRequest(
			w,
			"instruction_version_id必须与review_item_id同时提交",
		)
		return
	}

	if image != "" {
		if !strings.HasPrefix(image, "data:image/") {
			utils.BadRequest(
				w,
				"截图格式无效，请直接粘贴图片",
			)
			return
		}

		const maxImageLength = 12 * 1024 * 1024

		if len(image) > maxImageLength {
			utils.BadRequest(
				w,
				"截图过大，请压缩后重试（建议不超过8MB）",
			)
			return
		}
	}

	if instruction == "" {
		if mode == "rebuild" {
			instruction =
				"请参考截图重新设计本页内容区，保留导航栏和模板风格。"
		} else {
			instruction =
				"请参考截图修复页面版面问题，其余结构、内容和交互保持不变。"
		}
	}

	applicationStarted := false
	var mutationGuard *services.CoursewarePageMutationGuard

	if reviewItemID != "" {
		mutationGuard, err =
			beginCoursewareReviewItemRefineApplication(
				r.Context(),
				reviewItemID,
				coursewareID,
				pageNumber,
				instructionVersionID,
				instruction,
				scopedActor,
			)
		if err != nil {
			writeCoursewareRefineError(w, err)
			return
		}

		applicationStarted = true
	}

	result, err := h.genService.RefinePageWithModeGuarded(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNumber,
		instruction,
		image,
		mode,
		mutationGuard,
	)
	if err != nil {
		if applicationStarted {
			abortCoursewareReviewItemRefineApplication(
				r.Context(),
				reviewItemID,
				coursewareID,
				pageNumber,
				scopedActor,
			)
		}

		writeCoursewareRefineError(w, err)
		return
	}

	reviewItemStatus := ""
	reviewItemWarning := ""

	if applicationStarted {
		reviewItemStatus, reviewItemWarning =
			completeCoursewareReviewItemRefineApplication(
				r.Context(),
				reviewItemID,
				coursewareID,
				pageNumber,
				result,
				scopedActor,
			)
	}

	message := fmt.Sprintf(
		"第%d页微调完成",
		pageNumber,
	)
	if mode == "rebuild" {
		message = fmt.Sprintf(
			"第%d页全页重构完成",
			pageNumber,
		)
	}

	response := map[string]interface{}{
		"page_number":  pageNumber,
		"html_content": result,
		"mode":         mode,
		"message":      message,
	}

	if reviewItemID != "" {
		response["review_item_id"] = reviewItemID
		response["instruction_version_id"] = instructionVersionID
		response["review_item_status"] = reviewItemStatus

		if reviewItemWarning != "" {
			response["review_item_warning"] = reviewItemWarning
		}
	}

	utils.Success(w, response)
}

// RegeneratePage POST /api/v1/coursewares/{id}/pages/{num}/regenerate。
func (h *CoursewareGenHandler) RegeneratePage(
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

	coursewareID, pageNumber := extractCWPageRegeneratePath(
		r.URL.Path,
	)
	if coursewareID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}

	scopedActor, err := h.authorizeCoursewareRefineForHandler(
		r.Context(),
		coursewareID,
		claims.UserID,
		claims.Role,
	)
	if err != nil {
		writeCoursewareRefineError(w, err)
		return
	}

	result, err := h.genService.RegenerateSinglePage(
		r.Context(),
		coursewareID,
		scopedActor,
		pageNumber,
	)
	if err != nil {
		writeCoursewareRefineError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"page_number":  pageNumber,
			"html_content": result,
			"message": fmt.Sprintf(
				"第%d页重新生成完成",
				pageNumber,
			),
		},
	)
}
