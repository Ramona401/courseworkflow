package handlers

// assistant_runtime_handler_support.go
//
// 提供公开运行HTTP入口共用的严格JSON、Bearer令牌、客户端IP、SSE事件、
// 外部会话来源三方绑定、动态frame-ancestors iframe安全HTML和公开错误映射。
//
// 所有公开文案都经过收敛，不返回内部堆栈、模型、供应商、教师账户、
// 提示词、学校信息、允许来源列表或数据库细节。

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	assistantRuntimeStartBodyMaxBytes int64 = 16 * 1024
	assistantRuntimeChatBodyMaxBytes  int64 = 16 * 1024

	assistantRuntimeEmbedContextPrefix = "/embed/assistant/"
)

// decodeAssistantRuntimeJSON 严格解析单个JSON值。
func decodeAssistantRuntimeJSON(w http.ResponseWriter, r *http.Request, target interface{}, maxBytes int64) error {
	if r == nil || r.Body == nil || target == nil || maxBytes <= 0 {
		return errors.New("运行时请求体无效")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}

	var trailing interface{}
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("请求体包含多个JSON值")
	}

	return err
}

// writeAssistantRuntimeDecodeError 映射正文解析错误。
func writeAssistantRuntimeDecodeError(w http.ResponseWriter, err error) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		utils.Fail(w, http.StatusRequestEntityTooLarge, "教学智能体请求体过大")
		return
	}

	utils.BadRequest(w, "教学智能体请求参数格式错误")
}

// extractAssistantRuntimeBearerToken 只接受Authorization: Bearer。
func extractAssistantRuntimeBearerToken(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("请求不存在")
	}

	parts := strings.Fields(strings.TrimSpace(r.Header.Get("Authorization")))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("Bearer运行令牌格式错误")
	}

	return strings.TrimSpace(parts[1]), nil
}

// assistantRuntimeClientIP 只信任Nginx覆盖的X-Real-IP，其次使用RemoteAddr。
//
// 不读取X-Forwarded-For，避免客户端伪造链首地址。
func assistantRuntimeClientIP(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("请求不存在")
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if parsed := net.ParseIP(realIP); parsed != nil {
		return parsed.String(), nil
	}

	remoteAddress := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		if parsed := net.ParseIP(host); parsed != nil {
			return parsed.String(), nil
		}
	}

	if parsed := net.ParseIP(remoteAddress); parsed != nil {
		return parsed.String(), nil
	}

	return "", errors.New("客户端IP无效")
}

// assistantRuntimeCanonicalHTTPOrigin 严格规范化HTTP或HTTPS Origin。
//
// HTTP只允许localhost或回环地址。公网来源必须使用HTTPS。
// 默认端口会被移除，IPv6地址会恢复为标准方括号形式。
func assistantRuntimeCanonicalHTTPOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len([]rune(raw)) > 512 || strings.ContainsAny(raw, " \t\r\n;\"'") {
		return "", errors.New("运行来源格式无效")
	}

	parsed, err := url.Parse(raw)
	if err != nil ||
		parsed == nil ||
		parsed.Opaque != "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.RawPath != "" {
		return "", errors.New("运行来源结构无效")
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	hostname := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	port := strings.TrimSpace(parsed.Port())

	if hostname == "" || (scheme != "https" && scheme != "http") {
		return "", errors.New("运行来源协议无效")
	}

	if scheme == "http" {
		ip := net.ParseIP(hostname)
		if hostname != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", errors.New("公网运行来源必须使用HTTPS")
		}
	}

	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}

	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}

	return scheme + "://" + host, nil
}

// assistantRuntimeExpectedRequestOrigin 根据受信反向代理字段构造当前运行站点Origin。
//
// 生产Nginx会覆盖X-Forwarded-Proto并使用规范化$host传递Host。
// 直接本机调用时根据TLS状态或HTTP回环地址构造Origin。
func assistantRuntimeExpectedRequestOrigin(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("请求不存在")
	}

	scheme := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	if scheme != "https" && scheme != "http" {
		return "", errors.New("运行请求协议无效")
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		return "", errors.New("运行请求主机无效")
	}

	return assistantRuntimeCanonicalHTTPOrigin(scheme + "://" + host)
}

// validateAssistantRuntimeStartRequestContext 校验官方embed页面、API站点和public_id绑定。
//
// 必须同时满足：
//  1. 浏览器HTTP Origin等于当前TE-DNA运行站点；
//  2. Referer与当前TE-DNA运行站点同源；
//  3. Referer路径精确为当前public_id的官方embed路径；
//  4. Referer不携带查询参数、片段或用户凭据。
//
// 真正的外部父页面Origin由请求正文parent_origin携带，并在Service中继续与
// 部署allowed_origins精确比较。本函数不信任或替代该白名单校验。
func validateAssistantRuntimeStartRequestContext(r *http.Request, publicID string) error {
	if r == nil {
		return errors.New("运行请求不存在")
	}

	publicID = strings.TrimSpace(publicID)
	if publicID == "" || strings.Contains(publicID, "/") {
		return errors.New("运行公开编号无效")
	}

	expectedOrigin, err := assistantRuntimeExpectedRequestOrigin(r)
	if err != nil {
		return err
	}

	requestOrigin, err := assistantRuntimeCanonicalHTTPOrigin(r.Header.Get("Origin"))
	if err != nil || requestOrigin != expectedOrigin {
		return errors.New("运行请求Origin不匹配")
	}

	refererRaw := strings.TrimSpace(r.Header.Get("Referer"))
	if refererRaw == "" {
		return errors.New("运行请求Referer缺失")
	}

	referer, err := url.Parse(refererRaw)
	if err != nil ||
		referer == nil ||
		referer.Opaque != "" ||
		referer.User != nil ||
		referer.RawQuery != "" ||
		referer.Fragment != "" ||
		referer.RawPath != "" {
		return errors.New("运行请求Referer无效")
	}

	refererOrigin, err := assistantRuntimeCanonicalHTTPOrigin(referer.Scheme + "://" + referer.Host)
	if err != nil || refererOrigin != expectedOrigin {
		return errors.New("运行请求Referer来源不匹配")
	}

	expectedPath := assistantRuntimeEmbedContextPrefix + publicID
	if referer.Path != expectedPath {
		return errors.New("运行请求Referer路径不匹配")
	}

	return nil
}

// prepareAssistantRuntimeSSE 设置公开运行流式响应头。
//
// 学生端运行API与embed页面同源，不需要开放跨站CORS。
// 外部课件平台只能作为iframe父页面，不能直接跨站调用运行API。
func prepareAssistantRuntimeSSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Add("Vary", "Origin")
}

// writeAssistantRuntimeSSEEvent 写入并刷新一个SSE事件。
func writeAssistantRuntimeSSEEvent(w http.ResponseWriter, eventType string, data interface{}) error {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return errors.New("SSE事件类型不能为空")
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("编码教学智能体SSE事件失败: %w", err)
	}

	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, encoded); err != nil {
		return err
	}

	return http.NewResponseController(w).Flush()
}

// assistantRuntimeFrameAncestorsCSP 对Service已规范化的Origin再次做响应边界校验。
//
// 只输出部署明确保存的精确Origin，不隐式加入self。
// HTTP只允许localhost或回环地址，公网来源必须使用HTTPS。
func assistantRuntimeFrameAncestorsCSP(origins []string) (string, error) {
	if len(origins) == 0 || len(origins) > 20 {
		return "", errors.New("iframe允许来源数量无效")
	}

	result := make([]string, 0, len(origins))
	seen := make(map[string]struct{}, len(origins))

	for _, raw := range origins {
		origin := strings.TrimSpace(raw)

		canonical, err := assistantRuntimeCanonicalHTTPOrigin(origin)
		if err != nil {
			return "", errors.New("iframe允许来源格式无效")
		}

		if origin != canonical && origin != canonical+"/" {
			return "", errors.New("iframe允许来源不是精确Origin")
		}

		if _, exists := seen[canonical]; exists {
			continue
		}

		seen[canonical] = struct{}{}
		result = append(result, canonical)
	}

	if len(result) == 0 {
		return "", errors.New("iframe允许来源为空")
	}

	return strings.Join(result, " "), nil
}

// writeAssistantRuntimeEmbedHTML 返回动态CSP保护的独立学生端HTML壳。
//
// HTML本身不含内联脚本、不含教师身份、不含允许来源列表。
// 固定模块/assets/assistant-embed.js由Vite多入口构建生成。
//
// same-origin只允许当前embed页面向同源运行API发送完整Referer，
// 不会向外部父页面或其他跨源地址泄露embed URL。
func writeAssistantRuntimeEmbedHTML(w http.ResponseWriter, descriptor *models.AssistantDeploymentPublicDescriptor) {
	if descriptor == nil {
		utils.InternalError(w, "教学智能体公开信息不可用")
		return
	}

	frameAncestors, err := assistantRuntimeFrameAncestorsCSP(descriptor.FrameAncestors)
	if err != nil {
		utils.InternalError(w, "教学智能体公开安全策略不可用")
		return
	}

	title := html.EscapeString(strings.TrimSpace(descriptor.Title))
	welcome := html.EscapeString(strings.TrimSpace(descriptor.WelcomeMessage))
	publicID := html.EscapeString(strings.TrimSpace(descriptor.PublicID))
	displayMode := html.EscapeString(strings.TrimSpace(descriptor.DisplayMode))
	displayPosition := html.EscapeString(strings.TrimSpace(descriptor.DisplayPosition))

	contentSecurityPolicy := strings.Join([]string{
		"default-src 'none'",
		"script-src 'self'",
		"connect-src 'self'",
		"style-src 'unsafe-inline'",
		"img-src 'none'",
		"font-src 'none'",
		"object-src 'none'",
		"worker-src 'none'",
		"base-uri 'none'",
		"form-action 'none'",
		"frame-ancestors " + frameAncestors,
	}, "; ")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Content-Security-Policy", contentSecurityPolicy)

	_, _ = fmt.Fprintf(
		w,
		`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light">
<meta name="referrer" content="same-origin">
<title>%s</title>
<style>
html,body{margin:0;min-height:100%%;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f8fafc;color:#1f2937}
body{min-width:280px}
#assistant-embed-root{min-height:100vh}
.embed-fallback{box-sizing:border-box;max-width:680px;margin:0 auto;padding:24px}
.embed-fallback-card{border:1px solid #e2e8f0;border-radius:16px;background:#fff;padding:22px;box-shadow:0 8px 24px rgba(15,23,42,.06)}
.embed-fallback-title{margin:0 0 10px;font-size:20px}
.embed-fallback-text{margin:0;line-height:1.7;color:#64748b}
</style>
</head>
<body>
<div
 id="assistant-embed-root"
 data-public-id="%s"
 data-title="%s"
 data-welcome-message="%s"
 data-display-mode="%s"
 data-display-position="%s"
 data-maximum-session-turns="%d"
>
<main class="embed-fallback" aria-live="polite">
<section class="embed-fallback-card">
<h1 class="embed-fallback-title">%s</h1>
<p class="embed-fallback-text">%s</p>
<p class="embed-fallback-text">正在建立安全学生会话…</p>
</section>
</main>
</div>
<script type="module" src="/assets/assistant-embed.js"></script>
</body>
</html>`,
		title,
		publicID,
		title,
		welcome,
		displayMode,
		displayPosition,
		descriptor.MaximumSessionTurns,
		title,
		welcome,
	)
}

// assistantRuntimeHTTPStatusAndMessage 将内部错误映射为稳定HTTP响应。
func assistantRuntimeHTTPStatusAndMessage(err error) (int, string) {
	switch {
	case errors.Is(err, services.ErrAssistantRuntimeChatInvalidRequest),
		errors.Is(err, repository.ErrAssistantRuntimeSessionInputInvalid):
		return http.StatusBadRequest, "教学智能体请求参数无效"

	case errors.Is(err, services.ErrAssistantRuntimeTokenInvalid),
		errors.Is(err, services.ErrAssistantRuntimeTokenExpired):
		return http.StatusUnauthorized, "运行令牌无效或已过期"

	case errors.Is(err, services.ErrAssistantRuntimeOriginDenied):
		return http.StatusForbidden, "当前页面来源未获授权"

	case errors.Is(err, repository.ErrAssistantRuntimeDailyQuotaExceeded):
		return http.StatusTooManyRequests, "教学智能体今日调用额度已用尽"

	case errors.Is(err, repository.ErrAssistantRuntimeTurnInProgress):
		return http.StatusConflict, "当前会话已有正在处理的消息"

	case errors.Is(err, repository.ErrAssistantRuntimeTurnLimitReached):
		return http.StatusConflict, "当前会话轮数已用尽"

	case errors.Is(err, services.ErrAssistantRuntimeDeploymentVersionMismatch),
		errors.Is(err, services.ErrAssistantRuntimeSessionInactive),
		errors.Is(err, repository.ErrAssistantRuntimeTurnSessionUnavailable):
		return http.StatusConflict, "教学智能体会话状态已变化，请重新建立会话"

	case errors.Is(err, services.ErrAssistantRuntimeDeploymentUnavailable),
		errors.Is(err, repository.ErrAssistantDeploymentNotFound):
		return http.StatusNotFound, "教学智能体部署不存在或当前不可用"

	case errors.Is(err, repository.ErrAssistantRuntimeBillingAccountUnavailable),
		errors.Is(err, repository.ErrAssistantRuntimeMessagesInvalid),
		errors.Is(err, services.ErrAssistantRuntimeBillingContextInvalid),
		errors.Is(err, services.ErrAssistantRuntimeBillingResultInvalid),
		errors.Is(err, services.ErrAssistantRuntimeTokenConfiguration),
		errors.Is(err, services.ErrAssistantRuntimeSnapshotInvalid),
		errors.Is(err, services.ErrAssistantRuntimeChatSnapshotInvalid),
		errors.Is(err, services.ErrAssistantRuntimeChatUnavailable),
		errors.Is(err, services.ErrAssistantDeploymentStoredPolicyInvalid):
		return http.StatusServiceUnavailable, "教学智能体暂时不可用，请稍后重试"

	default:
		return http.StatusInternalServerError, "教学智能体服务异常，请稍后重试"
	}
}

// assistantRuntimePublicErrorMessage 返回SSE可见错误文案。
func assistantRuntimePublicErrorMessage(err error) string {
	_, message := assistantRuntimeHTTPStatusAndMessage(err)
	return message
}

// writeAssistantRuntimeHTTPError 写入统一公开错误。
func writeAssistantRuntimeHTTPError(w http.ResponseWriter, err error) {
	statusCode, message := assistantRuntimeHTTPStatusAndMessage(err)
	utils.Fail(w, statusCode, message)
}
