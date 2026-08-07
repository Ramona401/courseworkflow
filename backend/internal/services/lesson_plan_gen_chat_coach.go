package services

// lesson_plan_gen_chat_coach.go
// 历史停滞检测与教练建议兼容函数。
//
// 当前正式对话主链不自动调用本函数；
// 保留它仅用于紧急回滚或后续显式启用，避免主异步文件继续超过900行。

import (
        "context"
        "time"

        "tedna/internal/models"
)

// checkAndInsertCoachAdvice 对话完成后检测停滞并插入教练建议。
func (s *LessonPlanGenService) checkAndInsertCoachAdvice(
        ctx context.Context,
        planID string,
        stageCode string,
        turnID string,
) {
        time.Sleep(
                500 *
                        time.Millisecond,
        )

        stagnation := DetectStagnation(
                ctx,
                planID,
                stageCode,
        )
        if stagnation == nil ||
                !stagnation.IsStagnant {
                return
        }

        suggestion := GenerateCoachSuggestion(
                stagnation,
        )
        if suggestion == "" {
                return
        }

        coachMessage := &models.ConversationMessage{
                ID:        generateMsgID(),
                Role:      models.ConvRoleAssistant,
                Type:      models.ConvMsgTypeText,
                Content:   suggestion,
                CreatedAt: time.Now(),
        }

        if err := s.appendMessage(
                ctx,
                planID,
                coachMessage,
        ); err != nil {
                lpGenLog.Warn(
                        "教练建议写入消息失败",
                        "plan_id", planID,
                        "error", err,
                )
                return
        }

        GlobalLPSSEHub.Broadcast(
                planID,
                models.LPSSEEvent{
                        EventType:    models.LPSSEMessageDone,
                        PlanID:       planID,
                        ClientTurnID: turnID,
                        MessageID:    coachMessage.ID,
                        Message:      coachMessage,
                },
        )

        lpGenLog.Info(
                "教练建议已插入",
                "plan_id", planID,
                "stage", stageCode,
                "user_rounds",
                stagnation.ConsecutiveRounds,
        )
}
