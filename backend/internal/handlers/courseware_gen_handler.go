package handlers

// courseware_gen_handler.go — 课件HTML生成HTTP处理器（Phase 4C P0-1~P0-5）
//
// 提供六个接口：
//   1. POST /api/v1/coursewares/{id}/generate-preview   — 仅生成预览页（封面）
//   2. POST /api/v1/coursewares/{id}/save-nav-template   — 保存导航栏HTML模板
//   3. POST /api/v1/coursewares/{id}/generate-pages      — 用固定导航栏批量生成剩余页
//   4. POST /api/v1/coursewares/{id}/refine-nav           — P0-2: 导航栏AI微调
//   5. POST /api/v1/coursewares/{id}/pages/{num}/refine   — P0-4: 单页AI微调
//   6. POST /api/v1/coursewares/{id}/cancel-generate      — P0-5: 中途中断生成

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"time"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 课件生成处理器 ====================

// CoursewareGenHandler 课件HTML生成处理器
type CoursewareGenHandler struct {
	genService *services.CoursewareGenService
	cwService  *services.CoursewareService
}

// NewCoursewareGenHandler 创建课件HTML生成处理器
func NewCoursewareGenHandler(genService *services.CoursewareGenService, cwService *services.CoursewareService) *CoursewareGenHandler {
	return &CoursewareGenHandler{
		genService: genService,
		cwService:  cwService,
	}
}

// ==================== Step 1: 生成预览页 ====================

// GeneratePreview POST /api/v1/coursewares/{id}/generate-preview
func (h *CoursewareGenHandler) GeneratePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/generate-preview")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	userID := claims.UserID
	go func() {
		// 延迟800ms等待前端SSE连接建立，避免立即失败时error事件丢失
		time.Sleep(800 * time.Millisecond)
		asyncCtx := context.Background()
		if err := h.genService.GeneratePreviewPages(asyncCtx, id, userID); err != nil {
			fmt.Printf("[courseware_gen_handler] 预览页生成失败: courseware=%s err=%v\n", id, err)
		}
	}()
	utils.Success(w, map[string]interface{}{
		"message":       "预览页生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}

// ==================== Step 2: 保存导航栏模板 ====================

// SaveNavTemplate POST /api/v1/coursewares/{id}/save-nav-template
func (h *CoursewareGenHandler) SaveNavTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/save-nav-template")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	var req models.SaveNavTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if err := h.cwService.SaveNavTemplate(r.Context(), id, claims.UserID, req.NavTemplateHTML); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]string{"message": "导航栏模板保存成功"})
}

// ==================== Step 3: 批量生成剩余页 ====================

// GeneratePages POST /api/v1/coursewares/{id}/generate-pages
func (h *CoursewareGenHandler) GeneratePages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/generate-pages")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	userID := claims.UserID
	go func() {
		// 延迟800ms等待前端SSE连接建立，避免立即失败时error事件丢失
		time.Sleep(800 * time.Millisecond)
		asyncCtx := context.Background()
		if err := h.genService.GenerateRemainingPages(asyncCtx, id, userID); err != nil {
			fmt.Printf("[courseware_gen_handler] 课件生成失败: courseware=%s err=%v\n", id, err)
		}
	}()
	utils.Success(w, map[string]interface{}{
		"message":       "课件生成已启动（使用固定导航栏），请通过SSE监听进度",
		"courseware_id": id,
	})
}

// ==================== P0-2: 导航栏AI微调 ====================

// RefineNav POST /api/v1/coursewares/{id}/refine-nav
// 请求体: { "instruction": "Logo再大一点" }
// 同步返回微调后的导航栏HTML
func (h *CoursewareGenHandler) RefineNav(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/refine-nav")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	var req struct {
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if strings.TrimSpace(req.Instruction) == "" {
		utils.BadRequest(w, "修改意见不能为空")
		return
	}
	result, err := h.genService.RefineNav(r.Context(), id, claims.UserID, req.Instruction)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"nav_html": result,
		"message":  "导航栏微调完成",
	})
}

// ==================== P0-4: 单页AI微调 ====================

// RefinePage POST /api/v1/coursewares/{id}/pages/{num}/refine
// 请求体: { "instruction": "标题字号再大一些" }
func (h *CoursewareGenHandler) RefinePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	coursewareID, pageNum := extractCWPageRefinePath(r.URL.Path)
	if coursewareID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径格式错误")
		return
	}
	var req struct {
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if strings.TrimSpace(req.Instruction) == "" {
		utils.BadRequest(w, "修改意见不能为空")
		return
	}
	result, err := h.genService.RefinePage(r.Context(), coursewareID, claims.UserID, pageNum, req.Instruction)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"page_number":  pageNum,
		"html_content": result,
		"message":      fmt.Sprintf("第%d页微调完成", pageNum),
	})
}

// ==================== v0.42.11: 3D互动单页生成 ====================

// Generate3DPage POST /api/v1/coursewares/{id}/generate-3d-page
// 一次性生成完整的3D互动HTML单页（Three.js + OrbitControls）
// 异步执行，通过SSE推送进度
func (h *CoursewareGenHandler) Generate3DPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/generate-3d-page")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	userID := claims.UserID
	go func() {
		// 延迟800ms等待前端SSE连接建立
		time.Sleep(800 * time.Millisecond)
		asyncCtx := context.Background()
		if err := h.genService.Generate3DSinglePage(asyncCtx, id, userID); err != nil {
			fmt.Printf("[courseware_gen_handler] 3D单页生成失败: courseware=%s err=%v\n", id, err)
		}
	}()
	utils.Success(w, map[string]interface{}{
		"message":       "3D互动单页生成已启动，请通过SSE监听进度",
		"courseware_id": id,
	})
}

// ==================== P0-5: 中途中断生成 ====================

// CancelGenerate POST /api/v1/coursewares/{id}/cancel-generate
func (h *CoursewareGenHandler) CancelGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/cancel-generate")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}
	h.genService.CancelGenerate(id)
	utils.Success(w, map[string]string{
		"message":       "已发送停止信号",
		"courseware_id": id,
	})
}

// ==================== 路径解析辅助函数 ====================

// extractCWPageRefinePath 从 /api/v1/coursewares/{id}/pages/{num}/refine 提取课件ID和页码
func extractCWPageRefinePath(path string) (string, int) {
	if !strings.HasSuffix(path, "/refine") {
		return "", 0
	}
	trimmed := strings.TrimSuffix(path, "/refine")
	pagesIdx := strings.LastIndex(trimmed, "/pages/")
	if pagesIdx < 0 {
		return "", 0
	}
	numStr := trimmed[pagesIdx+len("/pages/"):]
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return "", 0
	}
	prefix := trimmed[:pagesIdx]
	cwPrefix := "/api/v1/coursewares/"
	if !strings.HasPrefix(prefix, cwPrefix) {
		return "", 0
	}
	coursewareID := prefix[len(cwPrefix):]
	if coursewareID == "" {
		return "", 0
	}
	return coursewareID, num
}
