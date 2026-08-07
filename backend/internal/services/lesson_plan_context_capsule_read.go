package services

// lesson_plan_context_capsule_read.go — 本课共识胶囊安全读取
//
// 本文件只负责教师端冷启动读取：
//   - 先复用GetConversation完成教案存在性与作者归属校验；
//   - 再读取当前active胶囊；
//   - 只解析并返回教师端安全display_json；
//   - 不返回capsule_json、context_text、source_manifest、证据原文或内部错误；
//   - 胶囊缺失、stale、failed或安全视图损坏时，正常返回对话且胶囊为空。
//
// 该能力复用既有conversation接口，不增加新的路由和授权面。

import (
	"context"
	"encoding/json"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// GetConversationWithContextCapsule 返回对话历史和可选的教师端安全胶囊。
//
// 对话历史仍是主结果。胶囊读取或解析失败时记录日志并静默降级，
// 避免一个增强视图阻断老师恢复备课。
func (s *LessonPlanGenService) GetConversationWithContextCapsule(
	ctx context.Context,
	planID string,
	callerID string,
) (
	[]*models.ConversationMessage,
	*models.LessonPlanContextCapsuleEventData,
	error,
) {
	messages, err := s.GetConversation(
		ctx,
		planID,
		callerID,
	)
	if err != nil {
		return nil, nil, err
	}

	capsule, err := repository.GetActiveLessonPlanContextCapsule(
		ctx,
		planID,
	)
	if err != nil {
		lpGenLog.Warn(
			"恢复对话时读取本课共识胶囊失败，已静默降级",
			"plan_id", planID,
			"error", err,
		)
		return messages, nil, nil
	}

	return messages,
		buildLessonPlanContextCapsuleEventData(
			planID,
			capsule,
		),
		nil
}

// buildLessonPlanContextCapsuleEventData 将数据库记录转换为教师端安全事件数据。
func buildLessonPlanContextCapsuleEventData(
	lessonPlanID string,
	capsule *models.LessonPlanContextCapsule,
) *models.LessonPlanContextCapsuleEventData {
	if capsule == nil ||
		!capsule.IsActiveUsable() ||
		strings.TrimSpace(capsule.DisplayJSON) == "" ||
		strings.TrimSpace(capsule.DisplayJSON) == "{}" {
		return nil
	}

	display := &models.LessonPlanContextCapsuleDisplayView{}
	if err := json.Unmarshal(
		[]byte(capsule.DisplayJSON),
		display,
	); err != nil {
		lpGenLog.Warn(
			"恢复对话时解析本课共识安全视图失败，已静默降级",
			"plan_id", lessonPlanID,
			"capsule_version", capsule.Version,
			"error", err,
		)
		return nil
	}

	if strings.TrimSpace(display.Summary) == "" &&
		len(display.Sections) == 0 {
		return nil
	}

	return &models.LessonPlanContextCapsuleEventData{
		Version: capsule.Version,
		Status:  capsule.Status,
		Display: display,
	}
}
