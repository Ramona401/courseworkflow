package services

// assistant_runtime_authorization.go
//
// 本文件实现短时运行令牌的数据库实时校验。
//
// 校验流程：
//   1. 验证独立签名、issuer、audience、用途和过期时间；
//   2. 校验路径session_id与令牌声明一致；
//   3. 使用JTI哈希读取数据库会话；
//   4. 按数据库session_kind执行实时功能总闸门；
//   5. 重新读取部署实时状态；
//   6. 拒绝暂停、撤销、过期、会话终态和current_version不匹配；
//   7. 读取令牌绑定的不可变版本供后续额度和聊天服务使用。
//
// 当公开运行总开关关闭时：
//   - external会话即使持有尚未过期的有效令牌，也会在步骤4即时拒绝；
//   - teacher_preview继续进入正式授权、聊天和计费链；
//   - 被开关临时拒绝的external会话不自动改为revoked，重新开启后在原令牌
//     仍有效且其他状态均合法时可以继续使用。
//
// 验证失败时可以把数据库活动会话收敛为expired或revoked，
// 但不会修改部署、版本快照或使用流水。
//
// AssistantRuntimeAuthorization包含敏感版本快照，禁止直接返回浏览器。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// AssistantRuntimeAuthorization 是后续额度和聊天服务使用的后端授权结果。
type AssistantRuntimeAuthorization struct {
	Claims     *AssistantRuntimeTokenClaims
	Session    *models.AssistantRuntimeSession
	Deployment *models.AssistantDeployment
	Version    *models.AssistantDeploymentVersion
}

// ValidateRuntimeToken 验证短时令牌及其数据库实时状态。
func (s *AssistantRuntimeSessionService) ValidateRuntimeToken(
	ctx context.Context,
	tokenString string,
	expectedSessionID string,
) (
	*AssistantRuntimeAuthorization,
	error,
) {
	if !s.configured() {
		return nil,
			ErrAssistantRuntimeTokenConfiguration
	}

	claims, err :=
		s.tokenService.Parse(
			tokenString,
		)
	if err != nil {
		return nil, err
	}

	expectedSessionID =
		strings.TrimSpace(expectedSessionID)
	if expectedSessionID == "" ||
		expectedSessionID !=
			strings.TrimSpace(
				claims.SessionID,
			) {
		return nil,
			ErrAssistantRuntimeTokenInvalid
	}

	jtiHash :=
		assistantRuntimeJTIHash(
			claims.ID,
		)

	session, err :=
		repository.GetAssistantRuntimeSessionForToken(
			ctx,
			expectedSessionID,
			jtiHash,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrAssistantRuntimeSessionTokenMismatch,
		) ||
			errors.Is(
				err,
				repository.ErrAssistantRuntimeSessionNotFound,
			) {
			return nil,
				ErrAssistantRuntimeTokenInvalid
		}

		return nil, err
	}

	// 功能总闸门必须使用数据库会话类型，而不是单独相信令牌声明。
	//
	// 这样即使未来令牌结构或调用入口发生变化：
	//   - external仍会在公开开关关闭时即时拒绝；
	//   - teacher_preview仍可继续使用正式聊天和计费链；
	//   - 未知或损坏的会话类型按无效令牌处理。
	if err :=
		s.validateSessionKindEnabled(
			session.SessionKind,
		); err != nil {
		return nil, err
	}

	deployment, err :=
		repository.GetAssistantDeploymentByID(
			ctx,
			claims.DeploymentID,
		)
	if err != nil {
		return nil,
			ErrAssistantRuntimeDeploymentUnavailable
	}

	now := time.Now().UTC()

	stateErr :=
		validateAssistantRuntimeAuthorizationState(
			claims,
			session,
			deployment,
			now,
		)
	if stateErr != nil {
		switch {
		case errors.Is(
			stateErr,
			ErrAssistantRuntimeTokenExpired,
		):
			_ = repository.MarkAssistantRuntimeSessionExpired(
				ctx,
				session.ID,
			)

		case errors.Is(
			stateErr,
			ErrAssistantRuntimeDeploymentVersionMismatch,
		),
			deployment.Status ==
				models.AssistantDeploymentStatusRevoked:
			_ = repository.RevokeAssistantRuntimeSession(
				ctx,
				session.ID,
			)
		}

		return nil, stateErr
	}

	version, err :=
		repository.GetAssistantDeploymentVersion(
			ctx,
			deployment.ID,
			claims.DeploymentVersion,
		)
	if err != nil {
		_ = repository.RevokeAssistantRuntimeSession(
			ctx,
			session.ID,
		)

		return nil,
			ErrAssistantRuntimeDeploymentVersionMismatch
	}

	return &AssistantRuntimeAuthorization{
		Claims:     claims,
		Session:    session,
		Deployment: deployment,
		Version:    version,
	}, nil
}

// validateAssistantRuntimeAuthorizationState 校验令牌、会话和部署三方绑定。
func validateAssistantRuntimeAuthorizationState(
	claims *AssistantRuntimeTokenClaims,
	session *models.AssistantRuntimeSession,
	deployment *models.AssistantDeployment,
	now time.Time,
) error {
	if claims == nil ||
		session == nil ||
		deployment == nil ||
		strings.TrimSpace(claims.SessionID) !=
			strings.TrimSpace(session.ID) ||
		strings.TrimSpace(claims.DeploymentID) !=
			strings.TrimSpace(session.DeploymentID) ||
		strings.TrimSpace(claims.DeploymentID) !=
			strings.TrimSpace(deployment.ID) ||
		claims.DeploymentVersion !=
			session.DeploymentVersion ||
		claims.SessionKind !=
			session.SessionKind ||
		!assistantRuntimeHashEqual(
			session.TokenJTIHash,
			assistantRuntimeJTIHash(claims.ID),
		) {
		return ErrAssistantRuntimeTokenInvalid
	}

	if session.Status !=
		models.AssistantRuntimeSessionStatusActive {
		return ErrAssistantRuntimeSessionInactive
	}

	if session.ExpiresAt == nil ||
		!session.ExpiresAt.After(now) {
		return ErrAssistantRuntimeTokenExpired
	}

	if deployment.Status !=
		models.AssistantDeploymentStatusActive ||
		deployment.ValidFrom == nil ||
		now.Before(
			deployment.ValidFrom.UTC(),
		) ||
		(deployment.ValidUntil != nil &&
			!deployment.ValidUntil.After(now)) {
		return ErrAssistantRuntimeDeploymentUnavailable
	}

	if deployment.CurrentVersion !=
		claims.DeploymentVersion {
		return ErrAssistantRuntimeDeploymentVersionMismatch
	}

	return nil
}

// assistantRuntimeOriginAllowed 判断当前Origin是否精确命中。
func assistantRuntimeOriginAllowed(
	origin string,
	allowed []string,
) bool {
	for _, candidate := range allowed {
		if candidate == origin {
			return true
		}
	}

	return false
}

// assistantRuntimeWelcomeMessageFromVersion 只提取公开欢迎语。
func assistantRuntimeWelcomeMessageFromVersion(
	version *models.AssistantDeploymentVersion,
) (
	string,
	error,
) {
	if version == nil ||
		strings.TrimSpace(
			version.TeachingPlanJSON,
		) == "" {
		return "",
			ErrAssistantRuntimeSnapshotInvalid
	}

	var teachingPlan models.AssistantDeploymentTeachingPlanSnapshot
	if err := json.Unmarshal(
		[]byte(version.TeachingPlanJSON),
		&teachingPlan,
	); err != nil {
		return "",
			fmt.Errorf(
				"%w: %v",
				ErrAssistantRuntimeSnapshotInvalid,
				err,
			)
	}

	welcomeMessage :=
		strings.TrimSpace(
			teachingPlan.WelcomeMessage,
		)
	if welcomeMessage == "" {
		return "",
			ErrAssistantRuntimeSnapshotInvalid
	}

	return welcomeMessage, nil
}
