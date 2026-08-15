package handlers

// identity_account_link_handler.go — TE-DNA Identity Account Linking HTTP边界。
//
// HTTP合同：
//   GET /api/v1/auth/identity/link-url
//     - 必须经过现有TE-DNA JWT AuthMiddleware；
//     - 本地账号只能取claims.UserID；
//     - 返回authorization_url并设置Secure HttpOnly Flow Cookie。
//
//   GET /api/v1/auth/identity/unlink-url
//     - 与link-url相同，但Flow purpose固定为unlink。
//
//   GET /api/v1/auth/identity/callback
//     - OIDC公开回调，不要求Bearer JWT；
//     - 本地账号身份只来自AES-GCM认证的Flow Cookie；
//     - callback完成后只302到固定前端/identity/callback。
//
// 本Handler绝不向浏览器输出：
//   - Client Secret；
//   - code_verifier / OIDC nonce；
//   - global_person_id；
//   - local_account_id / users.id；
//   - Backchannel Replay Nonce / Idempotency Key；
//   - Identity原始错误正文。

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	identityFrontendCallbackPath = "/identity/callback"

	identityCallbackStateMaxBytes       = 256
	identityCallbackCodeMaxBytes        = 4096
	identityCallbackErrorMaxBytes       = 256
	identityCallbackDescriptionMaxBytes = 2048
)

// IdentityAccountLinkHandler 只负责HTTP边界。
// Service通过惰性Provider取得，避免Identity Client Secret未安装时破坏TE-DNA启动。
type IdentityAccountLinkHandler struct {
	serviceProvider services.IdentityAccountLinkServiceProvider
}

// NewIdentityAccountLinkHandler 创建Phase 1账号关联Handler。
func NewIdentityAccountLinkHandler(
	serviceProvider services.IdentityAccountLinkServiceProvider,
) *IdentityAccountLinkHandler {
	return &IdentityAccountLinkHandler{
		serviceProvider: serviceProvider,
	}
}

// GetLinkURL 为当前已登录TE-DNA账号发起Identity Link授权。
func (h *IdentityAccountLinkHandler) GetLinkURL(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.startAuthorization(
		w,
		r,
		services.IdentityFlowPurposeLink,
	)
}

// GetUnlinkURL 为当前已登录TE-DNA账号发起Identity Unlink授权。
func (h *IdentityAccountLinkHandler) GetUnlinkURL(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.startAuthorization(
		w,
		r,
		services.IdentityFlowPurposeUnlink,
	)
}

func (h *IdentityAccountLinkHandler) startAuthorization(
	w http.ResponseWriter,
	r *http.Request,
	purpose string,
) {
	setIdentityAccountLinkSensitiveHeaders(w)

	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(r.Context())
	if !ok ||
		claims == nil ||
		strings.TrimSpace(claims.UserID) == "" {
		utils.Unauthorized(
			w,
			"未找到有效认证信息",
		)
		return
	}

	service, err := h.getService()
	if err != nil {
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"Identity账号关联服务暂时不可用",
		)
		return
	}

	start, err := service.StartAuthorization(
		r.Context(),
		purpose,
		claims.UserID,
	)
	if err != nil {
		writeIdentityAccountLinkServiceError(
			w,
			err,
		)
		return
	}

	if start.AuthorizationURL == "" ||
		start.FlowToken == "" ||
		start.CookieName !=
			services.IdentityFlowCookieName {
		utils.InternalError(
			w,
			"Identity授权流程初始化失败",
		)
		return
	}

	maxAge := int(
		time.Until(start.ExpiresAt).
			Seconds(),
	)
	if maxAge <= 0 {
		utils.InternalError(
			w,
			"Identity授权流程已经失效",
		)
		return
	}

	// __Host- Cookie安全要求：
	//   - Secure；
	//   - Path=/；
	//   - 不设置Domain。
	//
	// SameSite=Lax允许浏览器从Identity Center顶层GET导航回callback时携带Cookie，
	// 同时不放宽到SameSite=None。
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     start.CookieName,
			Value:    start.FlowToken,
			Path:     "/",
			Expires:  start.ExpiresAt,
			MaxAge:   maxAge,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		},
	)

	utils.Success(
		w,
		map[string]string{
			"authorization_url": start.AuthorizationURL,
		},
	)
}

// Callback 是Identity Center OIDC公开回调。
//
// “公开”只表示不要求Bearer JWT；真正的本地账号授权能力仍由
// Secure HttpOnly AES-GCM Flow Cookie和state共同保护。
func (h *IdentityAccountLinkHandler) Callback(
	w http.ResponseWriter,
	r *http.Request,
) {
	setIdentityAccountLinkSensitiveHeaders(w)

	if r.Method != http.MethodGet {
		redirectIdentityAccountLinkError(
			w,
			r,
			"IDENTITY_CALLBACK_INVALID",
		)
		return
	}

	callback, ok :=
		parseIdentityAccountLinkCallback(
			r,
		)
	if !ok {
		clearIdentityAccountLinkFlowCookie(w)

		redirectIdentityAccountLinkError(
			w,
			r,
			"IDENTITY_CALLBACK_INVALID",
		)
		return
	}

	flowCookie, cookieErr :=
		r.Cookie(
			services.IdentityFlowCookieName,
		)

	// Flow是单次授权凭据。无论成功、拒绝、协议错误还是上游故障，
	// callback到达后都立即清除浏览器Cookie，避免重复消费。
	clearIdentityAccountLinkFlowCookie(w)

	if callback.ProviderError != "" {
		redirectIdentityAccountLinkError(
			w,
			r,
			"IDENTITY_AUTHORIZATION_DENIED",
		)
		return
	}

	if cookieErr != nil ||
		flowCookie == nil ||
		strings.TrimSpace(flowCookie.Value) == "" {
		redirectIdentityAccountLinkError(
			w,
			r,
			"IDENTITY_FLOW_MISSING",
		)
		return
	}

	service, err := h.getService()
	if err != nil {
		redirectIdentityAccountLinkError(
			w,
			r,
			"IDENTITY_UNAVAILABLE",
		)
		return
	}

	result, err :=
		service.CompleteAuthorization(
			r.Context(),
			flowCookie.Value,
			callback.State,
			callback.Code,
		)
	if err != nil {
		var serviceErr *services.IdentityAccountLinkServiceError

		if errors.As(
			err,
			&serviceErr,
		) &&
			serviceErr != nil &&
			isSafeIdentityAccountLinkErrorCode(
				serviceErr.Code,
			) {
			redirectIdentityAccountLinkError(
				w,
				r,
				serviceErr.Code,
			)
			return
		}

		redirectIdentityAccountLinkError(
			w,
			r,
			"IDENTITY_OPERATION_FAILED",
		)
		return
	}

	if !validIdentityAccountLinkCompletionResult(
		result,
	) {
		redirectIdentityAccountLinkError(
			w,
			r,
			"IDENTITY_PROTOCOL_ERROR",
		)
		return
	}

	query := url.Values{}
	query.Set(
		"operation",
		result.Operation,
	)
	query.Set(
		"state",
		result.State,
	)
	query.Set(
		"changed",
		strconv.FormatBool(
			result.Changed,
		),
	)

	http.Redirect(
		w,
		r,
		identityFrontendCallbackPath+
			"?"+
			query.Encode(),
		http.StatusFound,
	)
}

type identityAccountLinkCallbackQuery struct {
	State         string
	Code          string
	ProviderError string
}

// parseIdentityAccountLinkCallback 严格限制OIDC callback query。
// error_description只接受有限长度后丢弃，绝不进入日志、响应或前端URL。
func parseIdentityAccountLinkCallback(
	r *http.Request,
) (identityAccountLinkCallbackQuery, bool) {
	if r == nil ||
		r.URL == nil {
		return identityAccountLinkCallbackQuery{},
			false
	}

	values := r.URL.Query()

	allowed := map[string]bool{
		"state":             true,
		"code":              true,
		"error":             true,
		"error_description": true,
	}

	for key, items := range values {
		if !allowed[key] ||
			len(items) != 1 {
			return identityAccountLinkCallbackQuery{},
				false
		}
	}

	stateValues, exists :=
		values["state"]
	if !exists ||
		len(stateValues) != 1 {
		return identityAccountLinkCallbackQuery{},
			false
	}

	state := stateValues[0]
	if state == "" ||
		state != strings.TrimSpace(state) ||
		len(state) >
			identityCallbackStateMaxBytes {
		return identityAccountLinkCallbackQuery{},
			false
	}

	codeItems, hasCode := values["code"]
	code := ""
	if hasCode {
		if len(codeItems) != 1 {
			return identityAccountLinkCallbackQuery{},
				false
		}

		code = codeItems[0]
	}

	errorItems, hasProviderError := values["error"]
	providerError := ""
	if hasProviderError {
		if len(errorItems) != 1 {
			return identityAccountLinkCallbackQuery{},
				false
		}

		providerError = errorItems[0]
	}

	description, hasDescription :=
		values["error_description"]
	if hasDescription {
		if len(description) != 1 ||
			len(description[0]) >
				identityCallbackDescriptionMaxBytes {
			return identityAccountLinkCallbackQuery{},
				false
		}
	}

	// OIDC success与error response必须按参数“存在性”严格二选一。
	//
	// 不能只看code/error正文是否为空：例如code=...&error=或
	// code=...&error_description=...仍然同时表达了两种响应形态，必须拒绝。
	switch {
	case hasProviderError:
		if providerError == "" ||
			providerError !=
				strings.TrimSpace(providerError) ||
			len(providerError) >
				identityCallbackErrorMaxBytes ||
			hasCode {
			return identityAccountLinkCallbackQuery{},
				false
		}

	case hasCode:
		if code == "" ||
			code != strings.TrimSpace(code) ||
			len(code) >
				identityCallbackCodeMaxBytes ||
			hasDescription {
			return identityAccountLinkCallbackQuery{},
				false
		}

	default:
		return identityAccountLinkCallbackQuery{},
			false
	}

	return identityAccountLinkCallbackQuery{
		State:         state,
		Code:          code,
		ProviderError: providerError,
	}, true
}

func (
	h *IdentityAccountLinkHandler,
) getService() (
	*services.IdentityAccountLinkService,
	error,
) {
	if h == nil ||
		h.serviceProvider == nil {
		return nil,
			errors.New(
				"Identity账号关联Handler未初始化",
			)
	}

	service, err :=
		h.serviceProvider()
	if err != nil {
		return nil, err
	}

	if service == nil {
		return nil,
			errors.New(
				"Identity账号关联Service不可用",
			)
	}

	return service, nil
}

func writeIdentityAccountLinkServiceError(
	w http.ResponseWriter,
	err error,
) {
	var serviceErr *services.IdentityAccountLinkServiceError

	if errors.As(
		err,
		&serviceErr,
	) &&
		serviceErr != nil {

		status := serviceErr.StatusCode
		if status < 400 ||
			status > 599 {
			status =
				http.StatusInternalServerError
		}

		utils.Fail(
			w,
			status,
			serviceErr.Message,
		)
		return
	}

	utils.InternalError(
		w,
		"Identity账号关联操作失败",
	)
}

func clearIdentityAccountLinkFlowCookie(
	w http.ResponseWriter,
) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name: services.IdentityFlowCookieName,

			Value: "",

			Path: "/",

			Expires: time.Unix(
				1,
				0,
			).UTC(),

			MaxAge: -1,

			HttpOnly: true,
			Secure:   true,

			SameSite: http.SameSiteLaxMode,
		},
	)
}

func redirectIdentityAccountLinkError(
	w http.ResponseWriter,
	r *http.Request,
	code string,
) {
	if !isSafeIdentityAccountLinkErrorCode(
		code,
	) {
		code =
			"IDENTITY_OPERATION_FAILED"
	}

	query := url.Values{}
	query.Set("error", code)

	http.Redirect(
		w,
		r,
		identityFrontendCallbackPath+
			"?"+
			query.Encode(),
		http.StatusFound,
	)
}

// callback只允许这一组服务器生成的固定非敏感错误码进入前端URL。
func isSafeIdentityAccountLinkErrorCode(
	code string,
) bool {
	switch code {
	case "IDENTITY_CALLBACK_INVALID",
		"IDENTITY_AUTHORIZATION_DENIED",
		"IDENTITY_FLOW_MISSING",
		"IDENTITY_FLOW_INVALID",
		"IDENTITY_STATE_MISMATCH",
		"IDENTITY_INVALID_OPERATION",
		"IDENTITY_UNAVAILABLE",
		"IDENTITY_NOT_INITIALIZED",
		"IDENTITY_CONTEXT_INVALID",
		"IDENTITY_UPSTREAM_ERROR",
		"IDENTITY_PROTOCOL_ERROR",
		"IDENTITY_PERSON_UNAVAILABLE",
		"IDENTITY_LINK_CONFLICT",
		"IDENTITY_LINK_STATE_CHANGED",
		"IDENTITY_OPERATION_FAILED",
		"ACCOUNT_UNAVAILABLE",
		"ACCOUNT_DISABLED",
		"UNAUTHORIZED",
		"DATABASE_ERROR":
		return true

	default:
		return false
	}
}

func validIdentityAccountLinkCompletionResult(
	result services.IdentityAccountLinkCompletionResult,
) bool {
	switch result.Operation {
	case services.IdentityBackchannelOperationLink:
		return result.State == "linked"

	case services.IdentityBackchannelOperationUnlink:
		return result.State == "unlinked"

	default:
		return false
	}
}

func setIdentityAccountLinkSensitiveHeaders(
	w http.ResponseWriter,
) {
	w.Header().Set(
		"Cache-Control",
		"no-store",
	)
	w.Header().Set(
		"Pragma",
		"no-cache",
	)
	w.Header().Set(
		"Referrer-Policy",
		"no-referrer",
	)
	w.Header().Set(
		"X-Content-Type-Options",
		"nosniff",
	)
}
