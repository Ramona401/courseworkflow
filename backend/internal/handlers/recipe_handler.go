package handlers

// recipe_handler.go — 备课配方HTTP处理器

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
	compService   *services.ComponentService
}

// NewRecipeHandler 创建配方处理器
func NewRecipeHandler(rs *services.RecipeService, cs *services.ComponentService) *RecipeHandler {
	return &RecipeHandler{recipeService: rs, compService: cs}
}

// ==================== 列表 ====================

func (h *RecipeHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	q := r.URL.Query()
	resp, err := h.recipeService.ListRecipes(r.Context(), claims.UserID, q.Get("scope"), q.Get("scope_ref_id"), q.Get("subject"), q.Get("grade_range"), atoi(q.Get("limit")), atoi(q.Get("offset")))
	if err != nil {
		utils.InternalError(w, "查询配方列表失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 创建 ====================

func (h *RecipeHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestArgs)
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

func (h *RecipeHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
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

func (h *RecipeHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}
	var req models.UpdateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestArgs)
		return
	}
	if err := h.recipeService.UpdateRecipe(r.Context(), recipeID, &req, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除 ====================

func (h *RecipeHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}
	if err := h.recipeService.DeleteRecipe(r.Context(), recipeID, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== Fork ====================

func (h *RecipeHandler) ForkRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
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

func (h *RecipeHandler) ShareRecipe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}
	var req models.ShareRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestArgs)
		return
	}
	if err := h.recipeService.ShareRecipe(r.Context(), recipeID, &req, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "共享成功"})
}

// ==================== 更新学情 ====================

func (h *RecipeHandler) UpdateStudentProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}
	var req models.UpdateStudentProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestArgs)
		return
	}
	if err := h.recipeService.UpdateStudentProfile(r.Context(), recipeID, req.StudentProfile, claims.UserID); err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "学情更新成功"})
}

// ==================== 预览AI上下文 ====================

func (h *RecipeHandler) PreviewContext(w http.ResponseWriter, r *http.Request) {
	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}
	preview, err := h.recipeService.PreviewContext(r.Context(), recipeID)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, preview)
}

// ==================== 智能推荐 ====================

func (h *RecipeHandler) RecommendComponents(w http.ResponseWriter, r *http.Request) {
	var req models.RecipeRecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestArgs)
		return
	}
	groups, err := h.recipeService.RecommendComponents(r.Context(), req.Subject, req.GradeRange)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{"groups": groups})
}

// ==================== 画像感知智能推荐 ====================

func (h *RecipeHandler) SmartRecommendComponents(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req models.RecipeRecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestArgs)
		return
	}
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

// ==================== 预设流程模板 ====================

func (h *RecipeHandler) GetFlowPresets(w http.ResponseWriter, r *http.Request) {
	presets := h.recipeService.GetFlowPresets()
	utils.Success(w, map[string]interface{}{"presets": presets})
}

// ==================== 校验流程 ====================

func (h *RecipeHandler) ValidateFlow(w http.ResponseWriter, r *http.Request) {
	var req models.ValidateFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestArgs)
		return
	}
	result := h.recipeService.ValidateStageFlow(req.Stages)
	utils.Success(w, result)
}

// ==================== 配方效果统计 ====================

func (h *RecipeHandler) GetRecipeStats(w http.ResponseWriter, r *http.Request) {
	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}
	resp, err := h.recipeService.GetRecipeStats(r.Context(), recipeID)
	if err != nil {
		handleRecipeError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== 配方市场排行榜 ====================

func (h *RecipeHandler) ListMarketRecipes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resp, err := h.recipeService.ListMarketRecipes(r.Context(), q.Get("subject"), q.Get("grade_range"), q.Get("sort_by"), atoi(q.Get("limit")), atoi(q.Get("offset")))
	if err != nil {
		utils.InternalError(w, "查询配方市场失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 辅助函数 ====================

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

// atoi 安全转换字符串为整数，失败返回0
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

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
