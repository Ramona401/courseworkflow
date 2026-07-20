package handlers

import (
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

const sseServiceDrainingMessage = "系统正在升级，暂不接受新的流式连接，请稍后重试"

// beginSSEHandshake 在写入任何SSE响应头之前取得全局握手许可。
//
// accepted=false时，本函数已经写入标准HTTP 503 JSON响应。
// accepted=true时，调用方必须调用finish；finish允许重复调用。
func beginSSEHandshake(w http.ResponseWriter) (finish func(), accepted bool) {
	finish, accepted = services.BeginGlobalSSEHandshake()
	if accepted {
		return finish, true
	}

	writeSSEServiceUnavailable(w)
	return nil, false
}

// writeSSEServiceUnavailable 写入统一的SSE排空响应。
func writeSSEServiceUnavailable(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "30")
	utils.Fail(
		w,
		http.StatusServiceUnavailable,
		sseServiceDrainingMessage,
	)
}
