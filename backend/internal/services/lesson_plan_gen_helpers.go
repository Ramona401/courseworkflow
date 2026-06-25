package services

// lesson_plan_gen_helpers.go — 教案生成服务的内部辅助方法（从 lesson_plan_gen_service.go 拆出）
//
// 助手轻量选择入口 Phase 1 拆分：本文件承接主文件搬出的一组纯辅助方法，使主文件回到 600 行红线内。
// 本次为纯位置搬移，逻辑零改动；所有方法仍挂在 *LessonPlanGenService 接收器上、仍属 services 包，
// 调用方与行为完全不变。
//
// 本文件方法清单：
//   - appendUnrecognizedTextbookNotice — 开场白末尾拼接「未识别课本」确定性提醒
//   - checkPlanEditable                — 教案存在性/归属/可编辑状态校验
//   - appendMessage                    — 追加消息到对话历史（薄封装）
//   - loadConversation                 — 加载全量对话历史（薄封装）
//   - resolveTemplateForReview         — 解析评审模板（系统提示词 + 评审规则）
//   - broadcastError                   — SSE 推送错误消息
//   - broadcastSoftRetryNotice         — 空流软兜底：友好消息替代开发者报错
//   - parseAIReply                     — 解析 AI 回复类型（文本/教案内容/组件推荐）

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// appendUnrecognizedTextbookNotice 检查勾选的课本图中有几张未识别文字（OCR 为空），
// 若有，则在开场白消息末尾拼接一句确定性的点名提醒，让老师明确知道哪些课本无法被参考。
// 设计要点：
//   - 纯代码判断 + 字符串拼接，不经过 AI，保证"必然出现、措辞精确"；
//   - 用图在勾选列表中的序号（第X张）定位，老师在课本区按勾选顺序即可对应；
//   - 任何异常（无关联/查询失败/全部已识别）都静默返回，不影响开场白。
func (s *LessonPlanGenService) appendUnrecognizedTextbookNotice(
	ctx context.Context,
	req *models.StartConversationRequest,
	openingMsg *models.ConversationMessage,
) {
	if openingMsg == nil || len(req.TextbookPageIDs) == 0 {
		return
	}

	pages, err := repository.GetTextbookPagesByIDs(ctx, req.TextbookPageIDs)
	if err != nil || len(pages) == 0 {
		return
	}

	// GetTextbookPagesByIDs 返回顺序按 textbook_name+page_number 排，
	// 与前端勾选顺序未必一致，这里按返回顺序编号"第X张"，并尽量带上章节/教材名帮助老师定位。
	var unrecognized []string
	for i, p := range pages {
		if strings.TrimSpace(p.OCRText) == "" {
			label := strings.TrimSpace(p.Chapter)
			if label == "" {
				label = strings.TrimSpace(p.TextbookName)
			}
			if label == "" {
				unrecognized = append(unrecognized, fmt.Sprintf("第%d张", i+1))
			} else {
				unrecognized = append(unrecognized, fmt.Sprintf("第%d张（%s）", i+1, label))
			}
		}
	}

	if len(unrecognized) == 0 {
		return
	}

	notice := fmt.Sprintf(
		"\n\n---\n📷 **课本识别提醒**：你关联的 %d 张课本图中，%s 尚未成功识别文字，本次备课**无法参考这些页面的内容**。建议返回「课本管理」对这些图重新点击「AI识别」，识别成功后再关联进来。",
		len(pages), strings.Join(unrecognized, "、"),
	)
	openingMsg.Content += notice

	lpGenLog.Info("开场白已拼接未识别课本提醒",
		"plan_id", req.Topic, "total", len(pages), "unrecognized", len(unrecognized))
}

// checkPlanEditable 检查教案是否存在、归属正确、且处于可编辑状态
func (s *LessonPlanGenService) checkPlanEditable(ctx context.Context, planID string, callerID string) (*models.LessonPlan, error) {
	lp, err := repository.GetLessonPlanByID(ctx, planID)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrLPGenPlanNotFound
		}
		return nil, err
	}
	if lp.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}
	if lp.Status != models.LPStatusDraft &&
		lp.Status != models.LPStatusPublishedPersonal &&
		lp.Status != models.LPStatusRevision &&
		lp.Status != models.LPStatusDeveloping {
		return nil, ErrLPGenNotEditable
	}
	return lp, nil
}

// appendMessage 追加消息到教案对话历史
func (s *LessonPlanGenService) appendMessage(ctx context.Context, planID string, msg *models.ConversationMessage) error {
	return repository.AppendConversationMessage(ctx, planID, msg)
}

// loadConversation 加载教案全量对话历史（前端展示用，不用于AI上下文）
func (s *LessonPlanGenService) loadConversation(ctx context.Context, planID string) ([]*models.ConversationMessage, error) {
	return repository.GetConversationLog(ctx, planID)
}

// resolveTemplateForReview 解析评审模板
func (s *LessonPlanGenService) resolveTemplateForReview(ctx context.Context, subject string) (systemPrompt string, reviewRules string) {
	return buildReviewSystemPrompt(subject), buildDefaultReviewRules(subject)
}

// broadcastError 通过SSE推送错误消息给前端
//
// 子轮一·B：新增 turnID 参数。本轮 chat 触发的错误传本轮 turnID（前端按轮次过滤，
// 作废轮次的迟到错误不污染新轮）；系统旁路（评审/手动按钮等）调用处传 "" 即可（前端不过滤）。
func (s *LessonPlanGenService) broadcastError(planID string, turnID string, msg string) {
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType:    models.LPSSEError,
		PlanID:       planID,
		ClientTurnID: turnID,
		Error:        msg,
	})
}

// broadcastSoftRetryNotice 空流软兜底：AI 整轮没吐内容、且自动重试仍空时，
// 给老师一条看得懂、能动手的友好消息，而不是"AI流式返回内容为空"这类开发者报错。
// 以正常 assistant 消息落库 + message_done 下发，避免触发前端红色错误样式；
// 并明确告诉老师“内容没丢”，消除“是不是要刷新”的慌乱。
func (s *LessonPlanGenService) broadcastSoftRetryNotice(ctx context.Context, planID string, turnID string) {
	softMsg := &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   "我这边刚才没接上话，可能是网络打了个盹。你把刚才那句再发一次、或点下面的按钮继续就好——之前的内容都还在，不会丢。",
		CreatedAt: time.Now(),
		Metadata:  map[string]interface{}{"soft_retry": true},
	}
	if err := s.appendMessage(ctx, planID, softMsg); err != nil {
		lpGenLog.Warn("软兜底消息写入失败", "plan_id", planID, "error", err)
	}
	GlobalLPSSEHub.Broadcast(planID, models.LPSSEEvent{
		EventType:    models.LPSSEMessageDone,
		PlanID:       planID,
		ClientTurnID: turnID,
		MessageID:    softMsg.ID,
		Message:      softMsg,
	})
}

// parseAIReply 解析AI回复，判断消息类型（普通文本/教案内容/组件推荐）
func (s *LessonPlanGenService) parseAIReply(ctx context.Context, content string, lp *models.LessonPlan) *models.ConversationMessage {
	msg := &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		CreatedAt: time.Now(),
	}

	if strings.Contains(content, "## 教学目标") || strings.Contains(content, "# 教案") {
		msg.Type = models.ConvMsgTypeContent
		msg.Content = content
		return msg
	}

	if strings.Contains(content, "【推荐组件】") || strings.Contains(content, "推荐以下教学方案") {
		msg.Type = models.ConvMsgTypeComponents
		msg.Content = cleanComponentMarkers(content)
		groups, _ := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
			Subject:       lp.Subject,
			GradeRange:    lp.Grade,
			InjectionMode: "recommend",
			Limit:         3,
		})
		msg.Components = convertGroupsToConvComponents(groups)
		return msg
	}

	msg.Type = models.ConvMsgTypeText
	msg.Content = content
	return msg
}
