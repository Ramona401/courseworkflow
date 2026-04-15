package utils

import (
	"encoding/json"
	"net/http"
)

// ==================== 公共错误消息常量（消除S1192字符串重复） ====================

const (
	// HTTP方法限制消息
	MsgMethodGetOnly      = "仅支持GET请求"
	MsgMethodPostOnly     = "仅支持POST请求"
	MsgMethodPutOnly      = "仅支持PUT请求"
	MsgMethodDeleteOnly   = "仅支持DELETE请求"
	MsgMethodGetPost      = "仅支持GET/POST请求"
	MsgMethodGetPut       = "仅支持GET/PUT请求"
	MsgMethodGetPutDelete = "仅支持GET/PUT/DELETE请求"
	MsgMethodPutDelete    = "仅支持PUT/DELETE请求"

	// 通用校验消息
	MsgBadRequestBody = "请求参数格式错误"
	MsgBadRequestArgs = "请求参数无效"
	MsgUnauthorized   = "未获取到用户信息"
	MsgNotLoggedIn    = "未登录"

	// 通用ID缺失消息
	MsgMissingPipelineID   = "缺少Pipeline ID"
	MsgMissingUserID       = "缺少用户ID"
	MsgMissingLessonPlanID = "缺少教案ID"
	MsgMissingPipelinePage = "缺少Pipeline ID或页码"
	MsgMissingRoleID       = "缺少角色ID"
	MsgMissingComponentID  = "缺少组件ID"
	MsgMissingTemplateID   = "缺少模板ID"
	MsgMissingPromptKey    = "缺少提示词标识"
	MsgMissingCourseCode   = "缺少课程编号"
	MsgMissingOrgID        = "缺少组织ID"
	MsgMissingGroupID      = "缺少教研组ID"
	MsgInvalidPlanID       = "教案ID无效"
	MsgInvalidPlanOrStage  = "教案ID或阶段代码无效"
	MsgInvalidRecipeID     = "配方ID无效"

	// 公共路径前缀常量（消除路径字符串重复）
	PathOrgPrefix   = "/api/v1/lesson-plans/organizations/"
	PathGroupPrefix = "/api/v1/lesson-plans/teaching-groups/"
	PathPromptPrefix = "/api/v1/prompts/"
	PathPagesSegment = "/pages/"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON 返回 JSON 响应
// v94修复errcheck: 检查Encode返回值，编码失败时写入500错误
func JSON(w http.ResponseWriter, statusCode int, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
	}); err != nil {
		_ = err
	}
}

// Success 返回成功响应
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, 0, "success", data)
}

// Fail 返回失败响应
func Fail(w http.ResponseWriter, statusCode int, message string) {
	JSON(w, statusCode, -1, message, nil)
}

// Unauthorized 返回 401
func Unauthorized(w http.ResponseWriter, message string) {
	Fail(w, http.StatusUnauthorized, message)
}

// Forbidden 返回 403
func Forbidden(w http.ResponseWriter, message string) {
	Fail(w, http.StatusForbidden, message)
}

// BadRequest 返回 400
func BadRequest(w http.ResponseWriter, message string) {
	Fail(w, http.StatusBadRequest, message)
}

// InternalError 返回 500
func InternalError(w http.ResponseWriter, message string) {
	Fail(w, http.StatusInternalServerError, message)
}
