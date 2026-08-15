package services

// courseware_review_history_issue.go
//
// R-03正式交付问题历史读取。
//
// 核心不变量：
//   - 只按本次feedback读取真正交付的问题；
//   - 当前item.status不进入历史DTO；
//   - 教师视图优先使用正式交付时已冻结在EvidenceJSON中的teacher_view_snapshot；
//   - 只有旧记录不存在教师快照时，才按正式交付瞬间status=confirmed走兼容构造；
//   - 指令只读取delivered_instruction_version_id；
//   - 绝不回退current_instruction_version_id；
//   - 作者后续“本次执行补充”只作为可选过程记录展示，不重算旧审核事实。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func buildCWReviewHistoryIssues(
	ctx context.Context,
	courseware *models.Courseware,
	review *models.CoursewareReview,
	feedback *models.CoursewareReviewFeedback,
) (
	[]models.CoursewareReviewHistoryIssue,
	error,
) {
	items, err :=
		repository.ListCoursewareReviewItemsByFeedback(
			ctx,
			feedback.ID,
		)
	if err != nil {
		return nil, err
	}

	historicalPageSnapshots, err :=
		repository.ListCoursewareReviewPageSnapshotsByReviewID(
			ctx,
			review.ID,
		)
	if err != nil {
		return nil, err
	}

	result :=
		make(
			[]models.CoursewareReviewHistoryIssue,
			0,
			len(items),
		)

	for _, item := range items {
		if item == nil {
			continue
		}

		if err :=
			validateCWReviewHistoryDeliveredItem(
				courseware,
				review,
				feedback,
				item,
			); err != nil {
			return nil, err
		}

		stablePageID, err :=
			resolveCWReviewHistoryIssuePageID(
				item,
				historicalPageSnapshots,
			)
		if err != nil {
			return nil, err
		}

		// Attach事务规定只有confirmed项才能正式交付。
		//
		// 后续applied/resolved/stale等状态必须先从只读副本中清除，
		// 防止兼容降级路径读取到今天的整改状态。
		frozenItem :=
			freezeCWReviewItemAtDelivery(
				item,
			)

		// 新记录优先读取正式交付前已经由后端归一化并冻结在
		// EvidenceJSON中的teacher_view_snapshot。
		//
		// 这里刻意不再次调用今天的教师视图规范化规则：
		// 以后即使检查项补齐、教师措辞或安全降级规则调整，
		// 已审核记录仍必须展示当时真正交付给作者的教师视图。
		//
		// 只有旧记录根本没有teacher_view_snapshot时，
		// 才使用既有Builder做确定性兼容降级。
		teacherView :=
			buildCWReviewHistoryTeacherViewSnapshot(
				frozenItem,
			)

		issue :=
			models.CoursewareReviewHistoryIssue{
				ID: item.ID,

				PageID:     stablePageID,
				PageNumber: item.PageNumberSnapshot,
				PageTitle:  item.PageTitleSnapshot,

				Severity:  item.Severity,
				Dimension: item.Dimension,

				TeacherView: teacherView,

				PreviousModificationRecords: []models.CoursewareReviewHistoryModificationRecord{},
			}

		if item.DeliveredInstructionVersionID == nil ||
			strings.TrimSpace(
				*item.DeliveredInstructionVersionID,
			) == "" {
			issue.DeliveredInstructionAvailable =
				false

			issue.DeliveredInstructionUnavailableReason =
				models.
					CWReviewHistoryInstructionUnavailableLegacy
		} else {
			version, versionErr :=
				repository.
					GetCoursewareReviewDeliveredInstructionVersion(
						ctx,
						review.ID,
						item.ID,
						*item.
							DeliveredInstructionVersionID,
					)
			if versionErr != nil {
				return nil, versionErr
			}

			issue.DeliveredInstructionAvailable =
				true

			issue.DeliveredInstruction =
				&models.
					CoursewareReviewHistoryDeliveredInstruction{
					VersionID:   version.ID,
					VersionNo:   version.VersionNo,
					Content:     version.Content,
					SourceType:  version.SourceType,
					ConfirmedAt: version.ConfirmedAt,
				}
		}

		modificationRecords, recordErr :=
			listCWReviewHistoryExecutionNotes(
				ctx,
				item,
			)
		if recordErr != nil {
			return nil, recordErr
		}

		issue.PreviousModificationRecords =
			modificationRecords

		result = append(
			result,
			issue,
		)
	}

	return result, nil
}

// buildCWReviewHistoryTeacherViewSnapshot 返回正式交付时冻结的教师视图。
//
// EvidenceJSON中的teacher_view_snapshot属于审核历史事实。
// 一旦存在，就必须原样使用，不能用当前Builder重新归一化；否则未来教师视图
// 规则调整会反向改变旧审核记录。
//
// 旧记录若不存在该快照，才允许走BuildCWReviewItemTeacherView兼容降级。
// 调用方传入的item必须已经通过freezeCWReviewItemAtDelivery冻结状态。
func buildCWReviewHistoryTeacherViewSnapshot(
	item *models.CoursewareReviewItem,
) models.CWAIReviewTeacherViewSnapshot {
	if item == nil {
		return models.CWAIReviewTeacherViewSnapshot{}
	}

	var evidence struct {
		TeacherViewSnapshot *models.CWAIReviewTeacherViewSnapshot `json:"teacher_view_snapshot"`
	}

	if err := json.Unmarshal(
		[]byte(item.EvidenceJSON),
		&evidence,
	); err == nil &&
		evidence.TeacherViewSnapshot != nil {
		snapshot := *evidence.TeacherViewSnapshot

		// 深拷贝切片，避免DTO后续处理意外修改Evidence反序列化对象。
		snapshot.AcceptanceChecks = append(
			[]string{},
			snapshot.AcceptanceChecks...,
		)

		return snapshot
	}

	teacherView :=
		BuildCWReviewItemTeacherView(
			item,
		)

	return models.CWAIReviewTeacherViewSnapshot{
		TeacherTitle: teacherView.TeacherTitle,

		WhatHappened: teacherView.WhatHappened,

		TeachingImpact: teacherView.TeachingImpact,

		ImprovementGoal: teacherView.ImprovementGoal,

		AcceptanceChecks: append(
			[]string{},
			teacherView.AcceptanceChecks...,
		),

		TeacherContext: teacherView.TeacherContext,

		ManualCheckRequired: teacherView.ManualCheckRequired,
	}
}

func validateCWReviewHistoryDeliveredItem(
	courseware *models.Courseware,
	review *models.CoursewareReview,
	feedback *models.CoursewareReviewFeedback,
	item *models.CoursewareReviewItem,
) error {
	if courseware == nil ||
		review == nil ||
		feedback == nil ||
		item == nil {
		return errors.New(
			"课件审核历史整改项关系不完整",
		)
	}

	if item.CoursewareReviewID == nil ||
		item.FeedbackID == nil ||
		strings.TrimSpace(
			*item.CoursewareReviewID,
		) != strings.TrimSpace(review.ID) ||
		strings.TrimSpace(
			*item.FeedbackID,
		) != strings.TrimSpace(feedback.ID) ||
		strings.TrimSpace(item.CoursewareID) !=
			strings.TrimSpace(review.CoursewareID) ||
		strings.TrimSpace(item.OwnerID) !=
			strings.TrimSpace(courseware.UserID) ||
		strings.TrimSpace(item.CreatedBy) !=
			strings.TrimSpace(review.ReviewerID) ||
		item.SourceType !=
			models.CWReviewItemSourceFormal ||
		item.ReviewLevel != review.ReviewLevel ||
		item.ReviewRound != review.ReviewRound {
		return errors.New(
			"课件审核历史整改项绑定关系异常",
		)
	}

	if feedback.AIReviewSessionID == nil ||
		strings.TrimSpace(
			*feedback.AIReviewSessionID,
		) == "" ||
		strings.TrimSpace(item.SourceSessionID) !=
			strings.TrimSpace(
				*feedback.AIReviewSessionID,
			) {
		return errors.New(
			"课件审核历史整改项来源会话关系异常",
		)
	}

	return nil
}

// freezeCWReviewItemAtDelivery 构造正式交付瞬间的只读副本。
func freezeCWReviewItemAtDelivery(
	item *models.CoursewareReviewItem,
) *models.CoursewareReviewItem {
	if item == nil {
		return nil
	}

	frozen := *item

	frozen.Status =
		models.CWReviewItemStatusConfirmed

	// 浏览器历史展示所需的“当前确认版本”必须固定到当时交付版本。
	frozen.CurrentInstructionVersionID =
		frozen.DeliveredInstructionVersionID

	// 全部后续执行/复审状态清空，避免影响历史教师视图。
	frozen.AppliedInstructionVersionID = nil
	frozen.AppliedPageHash = ""
	frozen.AppliedAt = nil

	frozen.ResubmittedAt = nil
	frozen.ResubmittedReviewLevel = 0
	frozen.ResubmittedReviewRound = 0

	frozen.ResolvedBy = nil
	frozen.ResolvedReviewID = nil
	frozen.ResolvedReviewLevel = 0
	frozen.ResolvedReviewRound = 0
	frozen.ResolutionNote = ""
	frozen.ResolvedAt = nil

	return &frozen
}

func listCWReviewHistoryExecutionNotes(
	ctx context.Context,
	item *models.CoursewareReviewItem,
) (
	[]models.CoursewareReviewHistoryModificationRecord,
	error,
) {
	messages, err :=
		repository.ListCoursewareReviewItemMessages(
			ctx,
			item.ID,
		)
	if err != nil {
		return nil, err
	}

	result :=
		make(
			[]models.CoursewareReviewHistoryModificationRecord,
			0,
		)

	for _, message := range messages {
		if message == nil ||
			message.UserID == nil ||
			strings.TrimSpace(
				*message.UserID,
			) != strings.TrimSpace(item.OwnerID) ||
			strings.TrimSpace(
				message.Role,
			) != "user" ||
			!isCWReviewHistoryExecutionNote(
				message.CitationsJSON,
			) {
			continue
		}

		result = append(
			result,
			models.
				CoursewareReviewHistoryModificationRecord{
				Content:   message.Content,
				CreatedAt: message.CreatedAt,
			},
		)
	}

	return result, nil
}

func isCWReviewHistoryExecutionNote(
	raw string,
) bool {
	var metadata struct {
		Event string `json:"event"`
	}

	if err := json.Unmarshal(
		[]byte(raw),
		&metadata,
	); err != nil {
		return false
	}

	return strings.TrimSpace(
		metadata.Event,
	) == "owner_execution_note"
}
