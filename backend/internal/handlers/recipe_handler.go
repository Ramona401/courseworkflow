package handlers

// recipe_handler.go — 备课配方HTTP处理器
//
// Phase 7A 新增：10个REST接口
// 迭代2 新增：
//   GET    /api/v1/lesson-plans/recipes/flow-presets    — 预设流程模板
//   POST   /api/v1/lesson-plans/recipes/validate-flow   — 校验流程完整性
// 迭代4B-2 新增：
//   POST   /api/v1/lesson-plans/recipes/smart-recommend — 画像感知智能推荐

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// RecipeHandler 配方HTTP处理器
type RecipeHandler struct {
	recipeService *services.RecipeService
	compService   *services.ComponentService // 迭代4B-2：用于调用SmartRecommendComponents
}

// NewRecipeHandler 创建配方处理器
// 迭代4B-2：新增compService参数
func NewRecipeHandler(rs *services.RecipeService, cs *services.ComponentService) *RecipeHandler {
	return &RecipeHandler{
		recipeService: rs,
		compService:   cs,
	}
}

// ==================== 列表 ====================

// ListRecipes GET /api/v1/lesson-plans/recipes
func (h *RecipeHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	q := r.URL.Query()
	scope := q.Get("scope")
	scopeRefID := q.Get("scope_ref_id")
	subject := q.Get("subject")
	gradeRange := q.Get("grade_range")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	resp, err := h.recipeService.ListRecipes(r.Context(), claims.UserID, scope, scopeRefID, subject, gradeRange, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询配方列表失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 创建 ====================

// CreateRecipe POST /api/v1/lesson-plans/recipes
func (h *RecipeHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	recipe, err := h.recipeService.CreateRecipe(r.Context(), &req, claims.UserID)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, recipe)
}

// ==================== 详情 ====================

// GetRecipe GET /api/v1/lesson-plans/recipes/{id}
func (h *RecipeHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	resp, err := h.recipeService.GetRecipe(r.Context(), recipeID)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== 更新 ====================

// UpdateRecipe PUT /api/v1/lesson-plans/recipes/{id}
func (h *RecipeHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	var req models.UpdateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	if err := h.recipeService.UpdateRecipe(r.Context(), recipeID, &req, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除 ====================

// DeleteRecipe DELETE /api/v1/lesson-plans/recipes/{id}
func (h *RecipeHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	if err := h.recipeService.DeleteRecipe(r.Context(), recipeID, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== Fork ====================

// ForkRecipe POST /api/v1/lesson-plans/recipes/{id}/fork
func (h *RecipeHandler) ForkRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	forked, err := h.recipeService.ForkRecipe(r.Context(), recipeID, claims.UserID)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, forked)
}

// ==================== 共享 ====================

// ShareRecipe PUT /api/v1/lesson-plans/recipes/{id}/share
func (h *RecipeHandler) ShareRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	var req models.ShareRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	if err := h.recipeService.ShareRecipe(r.Context(), recipeID, &req, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "共享成功"})
}

// ==================== 更新学情 ====================

// UpdateStudentProfile PUT /api/v1/lesson-plans/recipes/{id}/student-profile
func (h *RecipeHandler) UpdateStudentProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	var req models.UpdateStudentProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	if err := h.recipeService.UpdateStudentProfile(r.Context(), recipeID, req.StudentProfile, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "学情更新成功"})
}

// ==================== 预览AI上下文 ====================

// PreviewContext GET /api/v1/lesson-plans/recipes/{id}/preview-context
func (h *RecipeHandler) PreviewContext(w http.ResponseWriter, r *http.Request) {
	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	preview, err := h.recipeService.PreviewContext(r.Context(), recipeID)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, preview)
}

// ==================== 智能推荐（原始版本） ====================

// RecommendComponents POST /api/v1/lesson-plans/recipes/recommend
func (h *RecipeHandler) RecommendComponents(w http.ResponseWriter, r *http.Request) {
	var req models.RecipeRecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	groups, err := h.recipeService.RecommendComponents(r.Context(), req.Subject, req.GradeRange)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{"groups": groups})
}

// ==================== 迭代4B-2新增：画像感知智能推荐 ====================

// SmartRecommendComponents POST /api/v1/lesson-plans/recipes/smart-recommend
// 根据当前用户的teaching_profile+学科+年级进行加权推荐
// 无teaching_profile时降级为普通推荐
func (h *RecipeHandler) SmartRecommendComponents(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	var req models.RecipeRecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	// 从数据库读取当前用户的teaching_profile
	var profile *models.TeachingProfile
	tp, err := repository.GetTeachingProfile(r.Context(), claims.UserID)
	if err == nil && tp != nil {
		profile = tp
	}

	groups, err := h.compService.SmartRecommendComponents(r.Context(), req.Subject, req.GradeRange, profile)
	if err != nil {
		utils.InternalError(w, "智能推荐失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"groups": groups})
}

// ==================== 迭代2新增：预设流程模板 ====================

// GetFlowPresets GET /api/v1/lesson-plans/recipes/flow-presets
func (h *RecipeHandler) GetFlowPresets(w http.ResponseWriter, r *http.Request) {
	presets := h.recipeService.GetFlowPresets()
	utils.Success(w, map[string]interface{}{"presets": presets})
}

// ==================== 迭代2新增：校验流程完整性 ====================

// ValidateFlow POST /api/v1/lesson-plans/recipes/validate-flow
func (h *RecipeHandler) ValidateFlow(w http.ResponseWriter, r *http.Request) {
	var req models.ValidateFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	result := h.recipeService.ValidateStageFlow(req.Stages)
	utils.Success(w, result)
}

// ==================== 辅助函数 ====================

// extractRecipeID 从路径 /api/v1/lesson-plans/recipes/{id} 提取末尾ID
func extractRecipeID(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) < 1 {
		return ""
	}
	id := parts[len(parts)-1]
	if len(id) < 10 {
		return ""
	}
	return id
}

// extractRecipeMiddleID 从路径 /api/v1/lesson-plans/recipes/{id}/xxx 提取中间ID
func extractRecipeMiddleID(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "recipes" && i+1 < len(parts) {
			id := parts[i+1]
			if len(id) >= 10 {
				return id
			}
		}
	}
	return ""
}

// handleRecipeError 统一错误处理
func handleRecipeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrRecipeNotFound):
		utils.Fail(w, 404, "配方不存在")
	case errors.Is(err, services.ErrRecipeUnauthorized):
		utils.Fail(w, 403, "无权操作此配方")
	case errors.Is(err, services.ErrRecipeNameRequired):
		utils.BadRequest(w, "配方名称不能为空")
	case errors.Is(err, services.ErrRecipeSubjectRequired):
		utils.BadRequest(w, "学科不能为空")
	case errors.Is(err, services.ErrRecipeGradeRequired):
		utils.BadRequest(w, "年级不能为空")
	case errors.Is(err, services.ErrRecipeShareInvalid):
		utils.BadRequest(w, err.Error())
	default:
		utils.InternalError(w, "操作失败: "+err.Error())
	}
}

// ==================== 迭代6新增：配方效果统计 ====================

// GetRecipeStats GET /api/v1/lesson-plans/recipes/{id}/stats
// 获取配方的效果统计数据（使用次数/教案数/均分/最近使用记录）
func (h *RecipeHandler) GetRecipeStats(w http.ResponseWriter, r *http.Request) {
	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	resp, err := h.recipeService.GetRecipeStats(r.Context(), recipeID)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== 迭代6新增：配方市场排行榜 ====================

// ListMarketRecipes GET /api/v1/lesson-plans/recipes/market
// 查询配方市场（已共享的配方排行榜）
func (h *RecipeHandler) ListMarketRecipes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	subject := q.Get("subject")
	gradeRange := q.Get("grade_range")
	sortBy := q.Get("sort_by") // composite(默认)/use_count/fork_count/avg_score/newest
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	resp, err := h.recipeService.ListMarketRecipes(r.Context(), subject, gradeRange, sortBy, limit, offset)
	if err != nil {
		utils.InternalError(w, "查询配方市场失败")
		return
	}
	utils.Success(w, resp)
}
