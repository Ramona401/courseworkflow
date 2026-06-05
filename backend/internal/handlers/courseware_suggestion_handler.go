package handlers

// courseware_suggestion_handler.go — 课件页"AI物料建议"持久化读取/保存处理器
//
// 配套 repository/courseware_suggestion_repo.go, 实现"先读库、没有才调AI"省 token 策略中的
// 【读】与【显式保存】两端:
//   GET  /api/v1/coursewares/{id}/pages/{num}/image-suggestions  — 读已存生图建议(不调AI、不计token)
//   GET  /api/v1/coursewares/{id}/pages/{num}/video-storyboards  — 读已存视频分镜(不调AI)
//   POST /api/v1/coursewares/{id}/pages/{num}/video-storyboards  — 保存视频分镜(老师手动编辑/拆镜结果)
//
// 【写入的另一半】由两个已有 handler 出结果后顺手完成(见 courseware_asset_handler.go 的
// SuggestImagePrompt / SuggestVideoPrompt), 故 AI 每次产出都会自动落库, 进页只在库空时才回退调AI。
//
// 三个 handler 都挂在已有的 *CoursewareAssetHandler 上(Go 允许同类型方法跨文件定义), 因此路由
// 复用 dispatchCoursewareSubRoutes 里的 assetH, 无需改 registerCoursewareRoutes 的签名与调用方。
// 均做课件归属校验(只能读写自己课件的物料)。

import (
	"encoding/json"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// cwSuggestionsJSONOrEmpty 把"从 jsonb 列读出的 JSON 文本"转成可直接塞进响应体的值:
//   - 空白(列 NULL 或空串) → 返回空数组 []，前端据此判定"库里没有"→ 回退调AI;
//   - 非空 → 作为 json.RawMessage 原样透传(列里存的本就是合法 JSON 数组, 避免在 handlers 包反序列化 service 私有类型)。
func cwSuggestionsJSONOrEmpty(stored string) interface{} {
	if strings.TrimSpace(stored) == "" {
		return []interface{}{}
	}
	return json.RawMessage(stored)
}

// GetStoredImageSuggestions GET /api/v1/coursewares/{id}/pages/{num}/image-suggestions
// 只读已存生图建议, 返回 {"prompts": [...]}, 与 SuggestImagePrompt 响应同构, 前端可共用解析;
// 库里没有时 prompts 为空数组, 前端据此回退调用 suggest-image-prompt。
func (h *CoursewareAssetHandler) GetStoredImageSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/image-suggestions")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}
	cw, err := repository.GetCoursewareByID(r.Context(), cwID)
	if err != nil {
		utils.InternalError(w, "课件不存在: "+err.Error())
		return
	}
	if cw.UserID != claims.UserID {
		utils.Fail(w, http.StatusForbidden, "无权访问此课件")
		return
	}
	stored, err := repository.GetPageImageSuggestions(r.Context(), cwID, pageNum)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"prompts": cwSuggestionsJSONOrEmpty(stored)})
}

// GetStoredVideoStoryboards GET /api/v1/coursewares/{id}/pages/{num}/video-storyboards
// 只读已存视频分镜, 返回 {"storyboards": [...]}, 与 SuggestVideoPrompt 响应同构。
func (h *CoursewareAssetHandler) GetStoredVideoStoryboards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/video-storyboards")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}
	cw, err := repository.GetCoursewareByID(r.Context(), cwID)
	if err != nil {
		utils.InternalError(w, "课件不存在: "+err.Error())
		return
	}
	if cw.UserID != claims.UserID {
		utils.Fail(w, http.StatusForbidden, "无权访问此课件")
		return
	}
	stored, err := repository.GetPageVideoStoryboards(r.Context(), cwID, pageNum)
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"storyboards": cwSuggestionsJSONOrEmpty(stored)})
}

// SaveVideoStoryboards POST /api/v1/coursewares/{id}/pages/{num}/video-storyboards
// 保存视频分镜(老师手动编辑各镜提示词/台词后, 或前端防抖自动保存)。整列覆盖式写入;
// 传空数组/null 等价清空(写 NULL)。请求体: {"storyboards":[{scene,image_prompt,video_prompt,narration},...]}
func (h *CoursewareAssetHandler) SaveVideoStoryboards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	cwID, pageNum := extractCWAssetPageActionPath(r.URL.Path, "/video-storyboards")
	if cwID == "" || pageNum <= 0 {
		utils.BadRequest(w, "路径参数错误")
		return
	}
	cw, err := repository.GetCoursewareByID(r.Context(), cwID)
	if err != nil {
		utils.InternalError(w, "课件不存在: "+err.Error())
		return
	}
	if cw.UserID != claims.UserID {
		utils.Fail(w, http.StatusForbidden, "无权操作此课件")
		return
	}
	var req struct {
		Storyboards json.RawMessage `json:"storyboards"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	// 入库前确认是合法 JSON 数组(或空), 避免脏数据写进 jsonb 列; 空/null/空数组一律按"清空"处理。
	jsonStr := strings.TrimSpace(string(req.Storyboards))
	if jsonStr != "" && jsonStr != "null" {
		var probe []interface{}
		if err := json.Unmarshal(req.Storyboards, &probe); err != nil {
			utils.BadRequest(w, "storyboards 必须是 JSON 数组")
			return
		}
		if len(probe) == 0 {
			jsonStr = ""
		}
	} else {
		jsonStr = ""
	}
	if err := repository.UpdatePageVideoStoryboards(r.Context(), cwID, pageNum, jsonStr); err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{"message": "视频分镜已保存", "page_number": pageNum})
}
