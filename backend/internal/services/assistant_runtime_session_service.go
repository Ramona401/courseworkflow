package services

// assistant_runtime_session_service.go
//
// 本文件实现教学智能体短时运行会话创建。
//
// 支持两类会话：
//
//   1. external：
//      - 供外部课件平台或公开iframe使用；
//      - 按public_id读取部署；
//      - 必须精确命中部署允许的Origin；
//      - 匿名客户端ID和IP只保存用途隔离的HMAC哈希；
//      - 必须同时通过公开运行总开关。
//
//   2. teacher_preview：
//      - 供部署所有者在TE-DNA课件工坊内部预览；
//      - 使用教师登录JWT完成前置认证；
//      - 只允许部署owner_user_id本人创建；
//      - 不依赖或扩大外部Origin白名单；
//      - 不受公开运行总开关影响；
//      - 仍然使用正式不可变部署版本、正式运行令牌、正式聊天和积分结算链；
//      - AI消费仍从部署所有者个人积分账户扣除。
//
// 创建流程：
//   1. 校验当前会话类型是否允许运行；
//   2. 规范化运行身份和客户端IP；
//   3. 读取并复核部署状态、所有者、当前版本和有效期；
//   4. 读取当前不可变版本，只提取公开欢迎语；
//   5. 生成随机session_id和jti；
//   6. 签发短时独立用途JWT；
//   7. 数据库只保存jti哈希、身份HMAC和IP HMAC；
//   8. Repository在部署锁下再次复核实时状态后创建会话。
//
// 令牌的数据库实时校验拆分到assistant_runtime_authorization.go。
//
// 本文件不调用AI、不执行积分扣费、不注册HTTP路由。

import (
	"context"
	"errors"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	assistantRuntimeSessionCreateMaxAttempts = 3

	assistantRuntimeAnonymousClientHashPurpose = "anonymous-client"

	assistantRuntimeTeacherPreviewHashPurpose = "teacher-preview-user"

	assistantRuntimeIPHashPurpose = "client-ip"

	assistantRuntimeTeacherPreviewOriginSnapshot = "tedna://teacher-preview"
)

var (
	ErrAssistantRuntimeDeploymentUnavailable = errors.New(
		"教学智能体部署当前不可运行",
	)

	ErrAssistantRuntimeOriginDenied = errors.New(
		"当前来源不在教学智能体部署允许范围内",
	)

	ErrAssistantRuntimeSessionInactive = errors.New(
		"教学智能体运行会话已失效",
	)

	ErrAssistantRuntimeDeploymentVersionMismatch = errors.New(
		"教学智能体部署版本已发生变化",
	)

	ErrAssistantRuntimeSnapshotInvalid = errors.New(
		"教学智能体部署快照无效",
	)
)

// AssistantRuntimeSessionService 是短时运行会话服务。
type AssistantRuntimeSessionService struct {
	tokenService *AssistantRuntimeTokenService
	privacyKey   []byte

	// publicRuntimeEnabled只控制external会话。
	//
	// 安全默认值为false，调用方必须在HTTP服务开始接收请求前，
	// 根据可信后端配置显式调用SetPublicRuntimeEnabled。
	//
	// teacher_preview不依赖该字段，因此公开入口关闭时，
	// 教师内部预览仍可以复用正式运行令牌、聊天和计费链。
	publicRuntimeEnabled bool
}

// NewAssistantRuntimeSessionService 创建运行会话服务。
//
// signingSecret通常由JWT_SECRET注入，但会先做用途派生，
// 不会直接使用教师登录JWT签名密钥。
//
// privacySalt必须由独立环境配置注入，
// 用于匿名客户端、教师预览身份和IP的HMAC哈希。
//
// 服务创建后的公开运行默认关闭。
// 公开运行路由必须在开始提供服务前显式注入可信功能开关。
func NewAssistantRuntimeSessionService(
	signingSecret string,
	privacySalt string,
	ttl time.Duration,
) *AssistantRuntimeSessionService {
	service := &AssistantRuntimeSessionService{
		tokenService: newAssistantRuntimeTokenService(
			signingSecret,
			ttl,
		),

		// fail-closed：未显式注入时不得创建或继续external会话。
		publicRuntimeEnabled: false,
	}

	if strings.TrimSpace(privacySalt) != "" {
		service.privacyKey =
			deriveAssistantRuntimeKey(
				privacySalt,
				assistantRuntimePrivacyPurpose,
			)
	}

	return service
}

// SetPublicRuntimeEnabled 设置external公开运行总开关。
//
// 本方法只允许在HTTP服务开始接收请求前调用。
// 当前生产配置通过重启进程生效，不支持进程内动态修改，
// 因而不会发生请求并发读写该字段。
//
// teacher_preview始终不受该开关影响。
func (s *AssistantRuntimeSessionService) SetPublicRuntimeEnabled(
	enabled bool,
) {
	if s == nil {
		return
	}

	s.publicRuntimeEnabled = enabled
}

// validateSessionKindEnabled 校验指定会话类型是否允许进入运行链。
//
// external：必须显式开启公开运行总开关。
// teacher_preview：不受公开运行开关影响。
// 未知类型：按伪造或损坏令牌处理。
func (s *AssistantRuntimeSessionService) validateSessionKindEnabled(
	sessionKind string,
) error {
	switch strings.TrimSpace(sessionKind) {
	case models.AssistantRuntimeSessionKindExternal:
		if s == nil ||
			!s.publicRuntimeEnabled {
			return ErrAssistantRuntimeDeploymentUnavailable
		}

		return nil

	case models.AssistantRuntimeSessionKindTeacherPreview:
		return nil

	default:
		return ErrAssistantRuntimeTokenInvalid
	}
}

// configured 检查签名与隐私哈希配置。
func (s *AssistantRuntimeSessionService) configured() bool {
	return s != nil &&
		s.tokenService != nil &&
		s.tokenService.configured() &&
		len(s.privacyKey) == 32
}

// StartExternalSession 创建外部匿名会话并返回短时运行令牌。
func (s *AssistantRuntimeSessionService) StartExternalSession(
	ctx context.Context,
	publicID string,
	parentOrigin string,
	anonymousClientID string,
	clientIP string,
) (
	*models.AssistantRuntimeStartResponse,
	error,
) {
	if !s.configured() {
		return nil,
			ErrAssistantRuntimeTokenConfiguration
	}

	// 服务层再次执行公开运行总闸门。
	//
	// 即使路由包装被未来代码误绕过，也不能创建external会话。
	if err :=
		s.validateSessionKindEnabled(
			models.AssistantRuntimeSessionKindExternal,
		); err != nil {
		return nil, err
	}

	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	normalizedOrigin, err :=
		normalizeAssistantDeploymentOrigin(
			parentOrigin,
		)
	if err != nil {
		return nil,
			ErrAssistantRuntimeOriginDenied
	}

	normalizedClientID, err :=
		normalizeAssistantRuntimeAnonymousClientID(
			anonymousClientID,
		)
	if err != nil {
		return nil, err
	}

	normalizedIP, err :=
		normalizeAssistantRuntimeClientIP(
			clientIP,
		)
	if err != nil {
		return nil, err
	}

	deployment, err :=
		repository.GetAssistantDeploymentRuntimeByPublicID(
			ctx,
			publicID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrAssistantDeploymentNotFound,
		) {
			return nil,
				ErrAssistantRuntimeDeploymentUnavailable
		}

		return nil, err
	}

	now := time.Now().UTC()

	if err :=
		validateAssistantRuntimeDeploymentForSessionStart(
			deployment,
			now,
		); err != nil {
		return nil, err
	}

	allowedOrigins, err :=
		assistantDeploymentAllowedOriginsFromJSON(
			deployment.AllowedOriginsJSON,
		)
	if err != nil {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	normalizedAllowedOrigins, err :=
		normalizeAssistantDeploymentAllowedOrigins(
			allowedOrigins,
		)
	if err != nil ||
		!assistantRuntimeOriginAllowed(
			normalizedOrigin,
			normalizedAllowedOrigins,
		) {
		return nil,
			ErrAssistantRuntimeOriginDenied
	}

	version, err :=
		repository.GetAssistantDeploymentVersion(
			ctx,
			deployment.ID,
			deployment.CurrentVersion,
		)
	if err != nil {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	welcomeMessage, err :=
		assistantRuntimeWelcomeMessageFromVersion(
			version,
		)
	if err != nil {
		return nil, err
	}

	expiresAt :=
		assistantRuntimeSessionExpiresAt(
			now,
			s.tokenService.TTL(),
			deployment.ValidUntil,
		)
	if !expiresAt.After(now) {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	anonymousClientHash, err :=
		assistantRuntimePrivacyHash(
			s.privacyKey,
			assistantRuntimeAnonymousClientHashPurpose,
			normalizedClientID,
		)
	if err != nil {
		return nil, err
	}

	ipHash, err :=
		assistantRuntimePrivacyHash(
			s.privacyKey,
			assistantRuntimeIPHashPurpose,
			normalizedIP,
		)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt <
		assistantRuntimeSessionCreateMaxAttempts; attempt++ {
		sessionID, randomErr :=
			generateAssistantRuntimeSessionID()
		if randomErr != nil {
			return nil, randomErr
		}

		jti, randomErr :=
			generateAssistantRuntimeRandomID()
		if randomErr != nil {
			return nil, randomErr
		}

		runtimeToken, tokenErr :=
			s.tokenService.Issue(
				sessionID,
				deployment.ID,
				deployment.CurrentVersion,
				models.AssistantRuntimeSessionKindExternal,
				jti,
				expiresAt,
			)
		if tokenErr != nil {
			return nil, tokenErr
		}

		session, createErr :=
			repository.CreateAssistantRuntimeSession(
				ctx,
				sessionID,
				&models.AssistantRuntimeSessionCreateInput{
					DeploymentID:        deployment.ID,
					DeploymentVersion:   deployment.CurrentVersion,
					TokenJTIHash:        assistantRuntimeJTIHash(jti),
					AnonymousClientHash: anonymousClientHash,
					OriginSnapshot:      normalizedOrigin,
					IPHash:              ipHash,
					SessionKind:         models.AssistantRuntimeSessionKindExternal,
					MaxTurns:            deployment.PerSessionTurnLimit,
					ExpiresAt:           expiresAt,
				},
			)
		if createErr != nil {
			if errors.Is(
				createErr,
				repository.ErrAssistantRuntimeSessionTokenConflict,
			) {
				continue
			}

			if errors.Is(
				createErr,
				repository.ErrAssistantRuntimeSessionDeploymentUnavailable,
			) ||
				errors.Is(
					createErr,
					repository.ErrAssistantRuntimeSessionPolicyConflict,
				) {
				return nil,
					ErrAssistantRuntimeDeploymentUnavailable
			}

			return nil, createErr
		}

		return buildAssistantRuntimeStartResponse(
			session,
			runtimeToken,
			welcomeMessage,
		), nil
	}

	return nil,
		repository.ErrAssistantRuntimeSessionTokenConflict
}

// StartTeacherPreviewSession 为部署所有者创建教师内部预览会话。
//
// 教师身份必须来自已认证JWT，调用方只能传当前claims.UserID。
// 本入口不会读取或相信请求正文中的owner、school或education_domain。
//
// 预览会话运行当前不可变部署版本，产生的AI费用仍由deployment.owner_user_id承担。
// 预览会话不需要命中外部Origin白名单，也不受公开运行总开关影响，
// 但部署必须处于active且当前版本有效。
func (s *AssistantRuntimeSessionService) StartTeacherPreviewSession(
	ctx context.Context,
	deploymentID string,
	teacherUserID string,
	clientIP string,
) (
	*models.AssistantRuntimeStartResponse,
	error,
) {
	if !s.configured() {
		return nil,
			ErrAssistantRuntimeTokenConfiguration
	}

	// 明确执行类型闸门，保证未知会话类型无法通过。
	if err :=
		s.validateSessionKindEnabled(
			models.AssistantRuntimeSessionKindTeacherPreview,
		); err != nil {
		return nil, err
	}

	deploymentID = strings.TrimSpace(deploymentID)
	teacherUserID = strings.TrimSpace(teacherUserID)

	if deploymentID == "" ||
		teacherUserID == "" {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	normalizedIP, err :=
		normalizeAssistantRuntimeClientIP(
			clientIP,
		)
	if err != nil {
		return nil, err
	}

	deployment, err :=
		repository.GetAssistantDeploymentForOwner(
			ctx,
			deploymentID,
			teacherUserID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrAssistantDeploymentNotFound,
		) {
			return nil,
				ErrAssistantRuntimeDeploymentUnavailable
		}

		return nil, err
	}

	now := time.Now().UTC()

	if err :=
		validateAssistantRuntimeDeploymentForSessionStart(
			deployment,
			now,
		); err != nil {
		return nil, err
	}

	version, err :=
		repository.GetAssistantDeploymentVersion(
			ctx,
			deployment.ID,
			deployment.CurrentVersion,
		)
	if err != nil {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	welcomeMessage, err :=
		assistantRuntimeWelcomeMessageFromVersion(
			version,
		)
	if err != nil {
		return nil, err
	}

	expiresAt :=
		assistantRuntimeSessionExpiresAt(
			now,
			s.tokenService.TTL(),
			deployment.ValidUntil,
		)
	if !expiresAt.After(now) {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	teacherPreviewHash, err :=
		assistantRuntimePrivacyHash(
			s.privacyKey,
			assistantRuntimeTeacherPreviewHashPurpose,
			teacherUserID,
		)
	if err != nil {
		return nil, err
	}

	ipHash, err :=
		assistantRuntimePrivacyHash(
			s.privacyKey,
			assistantRuntimeIPHashPurpose,
			normalizedIP,
		)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt <
		assistantRuntimeSessionCreateMaxAttempts; attempt++ {
		sessionID, randomErr :=
			generateAssistantRuntimeSessionID()
		if randomErr != nil {
			return nil, randomErr
		}

		jti, randomErr :=
			generateAssistantRuntimeRandomID()
		if randomErr != nil {
			return nil, randomErr
		}

		runtimeToken, tokenErr :=
			s.tokenService.Issue(
				sessionID,
				deployment.ID,
				deployment.CurrentVersion,
				models.AssistantRuntimeSessionKindTeacherPreview,
				jti,
				expiresAt,
			)
		if tokenErr != nil {
			return nil, tokenErr
		}

		session, createErr :=
			repository.CreateAssistantRuntimeTeacherPreviewSession(
				ctx,
				sessionID,
				teacherUserID,
				&models.AssistantRuntimeSessionCreateInput{
					DeploymentID:        deployment.ID,
					DeploymentVersion:   deployment.CurrentVersion,
					TokenJTIHash:        assistantRuntimeJTIHash(jti),
					AnonymousClientHash: teacherPreviewHash,
					OriginSnapshot:      assistantRuntimeTeacherPreviewOriginSnapshot,
					IPHash:              ipHash,
					SessionKind:         models.AssistantRuntimeSessionKindTeacherPreview,
					MaxTurns:            deployment.PerSessionTurnLimit,
					ExpiresAt:           expiresAt,
				},
			)
		if createErr != nil {
			if errors.Is(
				createErr,
				repository.ErrAssistantRuntimeSessionTokenConflict,
			) {
				continue
			}

			if errors.Is(
				createErr,
				repository.ErrAssistantRuntimeSessionDeploymentUnavailable,
			) ||
				errors.Is(
					createErr,
					repository.ErrAssistantRuntimeSessionPolicyConflict,
				) {
				return nil,
					ErrAssistantRuntimeDeploymentUnavailable
			}

			return nil, createErr
		}

		return buildAssistantRuntimeStartResponse(
			session,
			runtimeToken,
			welcomeMessage,
		), nil
	}

	return nil,
		repository.ErrAssistantRuntimeSessionTokenConflict
}

// assistantRuntimeSessionExpiresAt 取运行令牌TTL与部署有效期的较早值。
func assistantRuntimeSessionExpiresAt(
	now time.Time,
	ttl time.Duration,
	deploymentValidUntil *time.Time,
) time.Time {
	expiresAt := now.Add(ttl)

	if deploymentValidUntil != nil &&
		deploymentValidUntil.Before(expiresAt) {
		expiresAt =
			deploymentValidUntil.UTC()
	}

	return expiresAt
}

// buildAssistantRuntimeStartResponse 构造两类会话共用的安全响应。
func buildAssistantRuntimeStartResponse(
	session *models.AssistantRuntimeSession,
	runtimeToken string,
	welcomeMessage string,
) *models.AssistantRuntimeStartResponse {
	if session == nil {
		return nil
	}

	return &models.AssistantRuntimeStartResponse{
		SessionID:      session.ID,
		RuntimeToken:   runtimeToken,
		Status:         session.Status,
		MaxTurns:       session.MaxTurns,
		ExpiresAt:      session.ExpiresAt,
		WelcomeMessage: welcomeMessage,
	}
}

// validateAssistantRuntimeDeploymentForSessionStart 是创建前纯状态校验。
func validateAssistantRuntimeDeploymentForSessionStart(
	deployment *models.AssistantDeployment,
	now time.Time,
) error {
	if deployment == nil ||
		strings.TrimSpace(deployment.ID) == "" ||
		deployment.CurrentVersion <= 0 ||
		deployment.PerSessionTurnLimit < 1 ||
		deployment.PerSessionTurnLimit > 100 ||
		strings.TrimSpace(deployment.Status) !=
			models.AssistantDeploymentStatusActive ||
		strings.TrimSpace(deployment.AccessMode) !=
			models.AssistantDeploymentAccessOriginAllowlist {
		return ErrAssistantRuntimeDeploymentUnavailable
	}

	if deployment.ValidFrom == nil ||
		now.Before(
			deployment.ValidFrom.UTC(),
		) ||
		(deployment.ValidUntil != nil &&
			!deployment.ValidUntil.After(now)) {
		return ErrAssistantRuntimeDeploymentUnavailable
	}

	return nil
}
