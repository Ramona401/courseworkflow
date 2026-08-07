package services

// assistant_runtime_read_service.go
//
// 提供公开iframe描述和运行会话安全视图读取。
//
// 公开描述只包含标题、欢迎语、展示方式和最大轮数；允许嵌入来源仅通过
// json:"-"字段传递给后端HTML响应生成动态frame-ancestors CSP，不进入公开JSON。
//
// 会话视图只包含正式student/assistant消息、轮数和状态。教师、学校、计费、
// 模型、提示词、JTI哈希、匿名客户端哈希和IP哈希永远不会进入响应。

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

// GetPublicDescriptor 读取当前可运行部署的公开展示信息和iframe安全来源。
func (s *AssistantRuntimeSessionService) GetPublicDescriptor(
	ctx context.Context,
	publicID string,
) (*models.AssistantDeploymentPublicDescriptor, error) {
	if s == nil {
		return nil, ErrAssistantRuntimeDeploymentUnavailable
	}

	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return nil, ErrAssistantRuntimeDeploymentUnavailable
	}

	deployment, err := repository.GetAssistantDeploymentRuntimeByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, repository.ErrAssistantDeploymentNotFound) {
			return nil, ErrAssistantRuntimeDeploymentUnavailable
		}
		return nil, err
	}

	if err := validateAssistantRuntimeDeploymentForSessionStart(deployment, time.Now().UTC()); err != nil {
		return nil, err
	}

	allowedOrigins, err := assistantDeploymentAllowedOriginsFromJSON(deployment.AllowedOriginsJSON)
	if err != nil {
		return nil, err
	}

	normalizedOrigins, err := normalizeAssistantDeploymentAllowedOrigins(allowedOrigins)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAssistantDeploymentStoredPolicyInvalid, err)
	}

	version, err := repository.GetAssistantDeploymentVersion(ctx, deployment.ID, deployment.CurrentVersion)
	if err != nil {
		return nil, ErrAssistantRuntimeDeploymentUnavailable
	}

	var teachingPlan models.AssistantDeploymentTeachingPlanSnapshot
	if err := json.Unmarshal([]byte(version.TeachingPlanJSON), &teachingPlan); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAssistantRuntimeSnapshotInvalid, err)
	}

	if strings.TrimSpace(teachingPlan.Title) == "" ||
		strings.TrimSpace(teachingPlan.WelcomeMessage) == "" ||
		!models.IsValidCoursewareAssistantDisplayMode(teachingPlan.DisplayMode) ||
		!models.IsValidCoursewareAssistantPosition(teachingPlan.DisplayPosition) {
		return nil, ErrAssistantRuntimeSnapshotInvalid
	}

	return &models.AssistantDeploymentPublicDescriptor{
		PublicID:            strings.TrimSpace(deployment.PublicID),
		Title:               strings.TrimSpace(teachingPlan.Title),
		WelcomeMessage:      strings.TrimSpace(teachingPlan.WelcomeMessage),
		DisplayMode:         strings.TrimSpace(teachingPlan.DisplayMode),
		DisplayPosition:     strings.TrimSpace(teachingPlan.DisplayPosition),
		MaximumSessionTurns: deployment.PerSessionTurnLimit,
		FrameAncestors:      normalizedOrigins,
	}, nil
}

// GetRuntimeSessionView 使用短时令牌读取正式会话状态和历史。
//
// completed和expired会话仍允许读取，便于前端恢复最终对话；revoked部署、
// 版本变化、JTI不匹配和绑定关系异常全部拒绝。
func (s *AssistantRuntimeSessionService) GetRuntimeSessionView(
	ctx context.Context,
	tokenString string,
	expectedSessionID string,
) (*models.AssistantRuntimeSessionView, error) {
	if !s.configured() {
		return nil, ErrAssistantRuntimeTokenConfiguration
	}

	claims, err := s.tokenService.Parse(tokenString)
	if err != nil {
		return nil, err
	}

	expectedSessionID = strings.TrimSpace(expectedSessionID)
	if expectedSessionID == "" || expectedSessionID != strings.TrimSpace(claims.SessionID) {
		return nil, ErrAssistantRuntimeTokenInvalid
	}

	session, err := repository.GetAssistantRuntimeSessionForToken(
		ctx,
		expectedSessionID,
		assistantRuntimeJTIHash(claims.ID),
	)
	if err != nil {
		if errors.Is(err, repository.ErrAssistantRuntimeSessionTokenMismatch) ||
			errors.Is(err, repository.ErrAssistantRuntimeSessionNotFound) {
			return nil, ErrAssistantRuntimeTokenInvalid
		}
		return nil, err
	}

	deployment, err := repository.GetAssistantDeploymentByID(ctx, claims.DeploymentID)
	if err != nil {
		return nil, ErrAssistantRuntimeDeploymentUnavailable
	}

	if err := validateAssistantRuntimeSessionReadBinding(claims, session, deployment); err != nil {
		if errors.Is(err, ErrAssistantRuntimeDeploymentVersionMismatch) ||
			deployment.Status == models.AssistantDeploymentStatusRevoked {
			_ = repository.RevokeAssistantRuntimeSession(ctx, session.ID)
		}
		return nil, err
	}

	now := time.Now().UTC()
	if session.Status == models.AssistantRuntimeSessionStatusActive &&
		(session.ExpiresAt == nil || !session.ExpiresAt.After(now)) {
		_ = repository.MarkAssistantRuntimeSessionExpired(ctx, session.ID)
		session.Status = models.AssistantRuntimeSessionStatusExpired
		session.ActiveTurnID = nil
		session.ActiveTurnStartedAt = nil
	}

	messages, err := decodeAssistantRuntimeSessionViewMessages(session.MessagesJSON)
	if err != nil {
		return nil, err
	}

	remainingTurns := session.MaxTurns - session.TurnCount
	if remainingTurns < 0 {
		remainingTurns = 0
	}

	return &models.AssistantRuntimeSessionView{
		ID:                session.ID,
		DeploymentVersion: session.DeploymentVersion,
		SessionKind:       session.SessionKind,
		Status:            session.Status,
		TurnCount:         session.TurnCount,
		MaxTurns:          session.MaxTurns,
		RemainingTurns:    remainingTurns,
		Messages:          messages,
		ExpiresAt:         session.ExpiresAt,
		LastActiveAt:      session.LastActiveAt,
	}, nil
}

// validateAssistantRuntimeSessionReadBinding 校验令牌、会话和部署三方绑定。
func validateAssistantRuntimeSessionReadBinding(
	claims *AssistantRuntimeTokenClaims,
	session *models.AssistantRuntimeSession,
	deployment *models.AssistantDeployment,
) error {
	if claims == nil ||
		session == nil ||
		deployment == nil ||
		strings.TrimSpace(claims.SessionID) != strings.TrimSpace(session.ID) ||
		strings.TrimSpace(claims.DeploymentID) != strings.TrimSpace(session.DeploymentID) ||
		strings.TrimSpace(claims.DeploymentID) != strings.TrimSpace(deployment.ID) ||
		claims.DeploymentVersion != session.DeploymentVersion ||
		claims.SessionKind != session.SessionKind ||
		!assistantRuntimeHashEqual(session.TokenJTIHash, assistantRuntimeJTIHash(claims.ID)) {
		return ErrAssistantRuntimeTokenInvalid
	}

	if deployment.Status != models.AssistantDeploymentStatusActive &&
		deployment.Status != models.AssistantDeploymentStatusPaused {
		return ErrAssistantRuntimeDeploymentUnavailable
	}

	if deployment.CurrentVersion != claims.DeploymentVersion {
		return ErrAssistantRuntimeDeploymentVersionMismatch
	}

	if !models.IsValidAssistantRuntimeSessionStatus(session.Status) ||
		session.Status == models.AssistantRuntimeSessionStatusRevoked {
		return ErrAssistantRuntimeSessionInactive
	}

	return nil
}

// decodeAssistantRuntimeSessionViewMessages 解码正式可见消息。
func decodeAssistantRuntimeSessionViewMessages(raw string) ([]models.AssistantRuntimeMessage, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "[]"
	}

	var messages []models.AssistantRuntimeMessage
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return nil, fmt.Errorf("%w: %v", repository.ErrAssistantRuntimeMessagesInvalid, err)
	}

	if messages == nil {
		messages = []models.AssistantRuntimeMessage{}
	}

	for _, message := range messages {
		if !models.IsValidAssistantRuntimeMessageRole(message.Role) ||
			strings.TrimSpace(message.Content) == "" {
			return nil, repository.ErrAssistantRuntimeMessagesInvalid
		}
	}

	return messages, nil
}
