package utils

import (
	"encoding/json"
	"net/http"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON 返回 JSON 响应
func JSON(w http.ResponseWriter, statusCode int, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
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
