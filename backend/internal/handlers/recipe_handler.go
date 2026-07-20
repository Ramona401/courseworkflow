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

// buildRecipeActor 从当前JWT构造可信配方Actor。
func buildRecipeActor(
	w http.ResponseWriter,
	r *http.Request,
) (*services.AssistantActorContext, bool) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok ||
		claims == nil ||
		strings.TrimSpace(claims.UserID) == "" {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return nil, false
	}

	actor := services.BuildActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	if actor == nil {
		utils.InternalError(
			w,
			"解析当前用户配方权限范围失败",
		)
		return nil, false
	}

	return actor, true
}

// ==================== 列表 ====================

// ListRecipes 查询当前Actor可信范围内的普通配方列表。
func (h *RecipeHandler) ListRecipes(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	resp, err := h.recipeService.ListRecipesForActor(
		r.Context(),
		actor,
		query.Get("scope"),
		query.Get("scope_ref_id"),
		query.Get("subject"),
		query.Get("grade_range"),
		atoi(query.Get("limit")),
		atoi(query.Get("offset")),
	)
	if err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ==================== 创建 ====================

// CreateRecipe 创建当前具体教学域下的个人配方。
func (h *RecipeHandler) CreateRecipe(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	var req models.CreateRecipeRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestArgs,
		)
		return
	}

	recipe, err :=
		h.recipeService.
			CreateRecipeForActor(
				r.Context(),
				actor,
				&req,
			)
	if err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, recipe)
}

// ==================== 详情 ====================

// GetRecipe 获取当前Actor有权查看的配方详情。
func (h *RecipeHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}

	resp, err := h.recipeService.GetRecipeForActor(
		r.Context(),
		actor,
		recipeID,
	)
	if err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ==================== 更新 ====================

// UpdateRecipe 更新作者本人且教育域兼容的配方。
func (h *RecipeHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
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

	if err := h.recipeService.UpdateRecipeForActor(
		r.Context(),
		actor,
		recipeID,
		&req,
	); err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "更新成功"})
}

// ==================== 删除 ====================

// DeleteRecipe 删除作者本人且教育域兼容的配方。
func (h *RecipeHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID := extractRecipeID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}

	if err := h.recipeService.DeleteRecipeForActor(
		r.Context(),
		actor,
		recipeID,
	); err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== Fork ====================

// ForkRecipe Fork当前Actor有权查看的配方。
func (h *RecipeHandler) ForkRecipe(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}

	forked, err := h.recipeService.ForkRecipeForActor(
		r.Context(),
		actor,
		recipeID,
	)
	if err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, forked)
}

// ==================== 共享 ====================

// ShareRecipe 将作者本人的配方共享到合法目标。
func (h *RecipeHandler) ShareRecipe(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
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

	if err := h.recipeService.ShareRecipeForActor(
		r.Context(),
		actor,
		recipeID,
		&req,
	); err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "共享成功"})
}

// ==================== 更新学情 ====================

// UpdateStudentProfile 更新作者本人配方的学情记录。
func (h *RecipeHandler) UpdateStudentProfile(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
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

	if err := h.recipeService.UpdateStudentProfileForActor(
		r.Context(),
		actor,
		recipeID,
		req.StudentProfile,
	); err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "学情更新成功"})
}

// ==================== 预览AI上下文 ====================

// PreviewContext 预览当前Actor有权查看的配方上下文。
func (h *RecipeHandler) PreviewContext(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}

	preview, err := h.recipeService.PreviewContextForActor(
		r.Context(),
		actor,
		recipeID,
	)
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

// GetRecipeStats 获取作者本人或admin有权查看的效果统计。
func (h *RecipeHandler) GetRecipeStats(w http.ResponseWriter, r *http.Request) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID := extractRecipeMiddleID(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, utils.MsgInvalidRecipeID)
		return
	}

	resp, err := h.recipeService.GetRecipeStatsForActor(
		r.Context(),
		actor,
		recipeID,
	)
	if err != nil {
		handleRecipeError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ==================== 配方市场排行榜 ====================

// ListMarketRecipes 查询当前Actor可见的配方市场排行榜。
func (h *RecipeHandler) ListMarketRecipes(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	resp, err :=
		h.recipeService.
			ListMarketRecipesForActor(
				r.Context(),
				actor,
				query.Get("subject"),
				query.Get("grade_range"),
				query.Get("sort_by"),
				atoi(query.Get("limit")),
				atoi(query.Get("offset")),
			)
	if err != nil {
		handleRecipeError(w, err)
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
