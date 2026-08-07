package services

// courseware_ai_review_item_relation_governance.go
//
// 课件AI审核问题清单的直接关系治理服务。
//
// 与全局讨论AI关系建议的区别：
//   1. 本服务不依赖全局assistant消息；
//   2. 关系类型、方向和说明均由当前用户明确填写；
//   3. source_global_message_id保持为空，明确表示人工直接建立；
//   4. 关系仍复用现有版本、取消、重新启用和追加式事件模型；
//   5. 不自动确认指令、忽略问题、修改页面或提交审核决定。
//
// 安全边界：
//   1. 只允许已完成AI审核会话的原创建者操作；
//   2. 两端必须属于同一课件和同一审核会话；
//   3. 两端必须尚未正式交付、仍可处理且各自页面快照有效；
//   4. 不相关页面发生变化时，不应阻断仍然有效的两条问题建立关系；
//   5. conflict关系由后端按UUID文本升序规范化；
//   6. 关系说明由用户明确填写，后端限制为1至500字。

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const cwAIReviewManualRelationExplanationMaxRunes = 500

var (
	ErrCWAIReviewManualRelationInvalid = errors.New(
		"人工整改项关系参数无效",
	)

	ErrCWAIReviewManualRelationExplanationInvalid = errors.New(
		"人工整改项关系说明不能为空或超过500字",
	)
)

// CWAIReviewManualRelationConfirmInput 是用户在问题清单中直接建立关系的输入。
//
// 浏览器必须明确提交关系方向。对于conflict，后端会统一规范端点顺序。
type CWAIReviewManualRelationConfirmInput struct {
	RelationType string
	SourceItemID string
	TargetItemID string
	Explanation  string
}

// ConfirmCWAIReviewManualRelation 明确建立或重新启用一条人工问题关系。
func (s *CoursewareAIReviewRunner) ConfirmCWAIReviewManualRelation(
	ctx context.Context,
	sessionID string,
	input *CWAIReviewManualRelationConfirmInput,
	actor *CoursewareActorContext,
) (*CWAIReviewGlobalRelationRecord, error) {
	if input == nil {
		return nil, ErrCWAIReviewManualRelationInvalid
	}

	// 这里只执行会话所有权、课件访问权限和会话完成状态校验。
	//
	// 不要求整课快照完全未变化，因为关系只涉及两个明确问题；
	// 下方loadCWAIReviewGlobalSelectedItems会分别复核两端归属、
	// 未交付状态、可处理状态以及各自稳定页面快照。
	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			false,
		)
	if err != nil {
		return nil, err
	}

	relationType := strings.TrimSpace(input.RelationType)
	sourceItemID := strings.TrimSpace(input.SourceItemID)
	targetItemID := strings.TrimSpace(input.TargetItemID)
	explanation := strings.TrimSpace(input.Explanation)

	if !models.IsCWReviewItemRelationType(relationType) ||
		sourceItemID == "" ||
		targetItemID == "" ||
		sourceItemID == targetItemID {
		return nil, ErrCWAIReviewManualRelationInvalid
	}

	if explanation == "" ||
		utf8.RuneCountInString(explanation) >
			cwAIReviewManualRelationExplanationMaxRunes {
		return nil, ErrCWAIReviewManualRelationExplanationInvalid
	}

	// conflict没有方向，数据库要求两个UUID按文本升序保存。
	if relationType == models.CWReviewItemRelationConflict &&
		sourceItemID > targetItemID {
		sourceItemID, targetItemID =
			targetItemID, sourceItemID
	}

	// 复用既有严格端点校验：
	// 同会话、同课件、来源类型一致、未交付、可处理且页面快照有效。
	if _, err :=
		loadCWAIReviewGlobalSelectedItems(
			ctx,
			session,
			[]string{
				sourceItemID,
				targetItemID,
			},
			actor,
		); err != nil {
		return nil, err
	}

	relation, err :=
		repository.ConfirmCoursewareReviewItemRelation(
			ctx,
			&models.CoursewareReviewItemRelation{
				CoursewareID:    session.CoursewareID,
				SourceSessionID: session.ID,
				SourceItemID:    sourceItemID,
				TargetItemID:    targetItemID,
				RelationType:    relationType,
				Explanation:     explanation,

				// 人工直接建立的关系不伪造全局讨论来源消息。
				SourceGlobalMessageID: nil,
				CreatedBy:             actor.UserID,
			},
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewGlobalRelationRecord(
		ctx,
		relation,
		actor.UserID,
	)
}
