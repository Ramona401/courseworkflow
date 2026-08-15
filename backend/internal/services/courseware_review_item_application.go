package services

// courseware_review_item_application.go
//
// 课件审核问题与页面AI微调之间的状态闭环。
//
// 生命周期：
//
//      confirmed -> applying -> applied
//                         ^        |
//                         |        |
//                         +--------+  自审“继续调整”再次进入页面修改
//
// applied表示页面修改已经完成，但不自动等于问题已经解决：
//   - 正式审核问题需要等待审核员复审确认；
//   - 作者自审问题需要等待作者本人检查效果并明确确认。
//
// 安全边界：
//   1. 只有课件作者本人能够应用整改要求或修改方案；
//   2. 正式问题必须已经交付作者；
//   3. 页面应用请求必须携带明确的instruction_version_id；
//   4. 正式问题使用交付版本，自审问题使用当前确认版本；
//   5. 开始应用时在事务内重新校验版本状态、可信正文和页面哈希；
//   6. Begin返回事务确认的稳定page_id、页码和页面哈希；
//   7. 页面AI服务必须绑定该守卫，并在最终写入时执行CAS；
//   8. 首次AI微调失败时允许applying回退confirmed；
//   9. 自审再次调整失败时恢复上一次applied事实，不丢失既有成功修改；
//  10. 页面写入成功后只记录applied，不能自动记录resolved。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrCWReviewItemApplicationPageMismatch = errors.New(
		"整改项与当前课件页面不匹配",
	)
	ErrCWReviewItemApplicationInstructionMissing = errors.New(
		"整改项尚未确认最终修改指令",
	)
	ErrCWReviewItemApplicationInstructionMismatch = errors.New(
		"本次微调指令未完整包含已确认的整改指令",
	)
	ErrCWReviewItemApplicationVersionMismatch = errors.New(
		"页面应用指令版本已变化，请刷新整改项后重试",
	)
)

// CWReviewItemApplicationResult 是页面修改完成后的问题状态结果。
type CWReviewItemApplicationResult struct {
	ItemID          string
	Status          string
	AppliedPageHash string
}

// BeginCWReviewItemApplication 在执行页面AI微调前绑定版本并置为applying。
//
// 返回值是数据库事务确认的页面守卫。
// 调用方必须将其传入RefinePageWithModeGuarded，不能丢弃后改用普通入口。
func BeginCWReviewItemApplication(
	ctx context.Context,
	itemID string,
	coursewareID string,
	pageNumber int,
	instructionVersionID string,
	submittedInstruction string,
	actor *CoursewareActorContext,
) (*CoursewarePageMutationGuard, error) {
	itemID = strings.TrimSpace(itemID)
	coursewareID = strings.TrimSpace(coursewareID)
	instructionVersionID = strings.TrimSpace(instructionVersionID)
	submittedInstruction = strings.TrimSpace(submittedInstruction)

	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}
	if itemID == "" {
		return nil, repository.ErrCoursewareReviewItemNotFound
	}
	if coursewareID == "" || pageNumber <= 0 {
		return nil, ErrCWReviewItemApplicationPageMismatch
	}
	if instructionVersionID == "" {
		return nil, ErrCWReviewItemApplicationVersionMismatch
	}
	if submittedInstruction == "" {
		return nil, ErrCWReviewItemApplicationInstructionMismatch
	}

	item, _, err := loadAuthorizedCWReviewItem(
		ctx,
		itemID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	// 正式审核员可以讨论自己创建的问题，
	// 但只有课件作者能够真正修改课件页面。
	if actor.UserID != item.OwnerID {
		return nil, ErrCWAIReviewNoPermission
	}
	if item.CoursewareID != coursewareID {
		return nil, repository.ErrCoursewareReviewItemNotFound
	}
	if item.IsGlobalIssue() ||
		item.PageID == nil ||
		strings.TrimSpace(*item.PageID) == "" {
		return nil, ErrCWReviewItemApplicationPageMismatch
	}

	startResult, err :=
		repository.BeginCoursewareReviewItemApplicationWithVersion(
			ctx,
			&repository.BeginCoursewareReviewItemApplicationInput{
				ItemID:               item.ID,
				ActorID:              actor.UserID,
				CoursewareID:         coursewareID,
				PageNumber:           pageNumber,
				InstructionVersionID: instructionVersionID,
				SubmittedInstruction: submittedInstruction,
			},
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemApplicationVersionMismatch,
		):
			return nil, ErrCWReviewItemApplicationVersionMismatch

		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemApplicationInstructionMismatch,
		):
			return nil, ErrCWReviewItemApplicationInstructionMismatch

		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemApplicationPageMismatch,
		):
			return nil, ErrCWReviewItemApplicationPageMismatch

		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemApplicationPageStale,
		):
			return nil, ErrCWReviewItemStale

		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemApplicationPageOrphaned,
		):
			return nil, ErrCWReviewItemOrphaned

		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemApplicationNotDelivered,
		):
			return nil, ErrCWReviewItemNotDelivered

		default:
			return nil, err
		}
	}

	if startResult == nil ||
		strings.TrimSpace(startResult.PageID) == "" ||
		startResult.PageNumber != pageNumber ||
		len(strings.TrimSpace(startResult.PageHTMLHash)) != 64 ||
		strings.TrimSpace(startResult.InstructionVersionID) !=
			instructionVersionID {
		return nil, repository.ErrCoursewareReviewItemConflict
	}

	return &CoursewarePageMutationGuard{
		PageID:     strings.TrimSpace(startResult.PageID),
		PageNumber: startResult.PageNumber,
		HTMLHash: strings.ToLower(
			strings.TrimSpace(startResult.PageHTMLHash),
		),
	}, nil
}

// AbortCWReviewItemApplication 在页面AI微调失败时恢复安全的上一个状态。
//
// 首次修改失败恢复confirmed；
// 自审“继续调整”失败则恢复上一次applied事实，并重新检查修改完成后的页面指纹。
func AbortCWReviewItemApplication(
	ctx context.Context,
	itemID string,
	actor *CoursewareActorContext,
) error {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return ErrCWAIReviewActorRequired
	}

	item, _, err := loadAuthorizedCWReviewItem(
		ctx,
		strings.TrimSpace(itemID),
		actor,
	)
	if err != nil {
		return err
	}

	if actor.UserID != item.OwnerID {
		return ErrCWAIReviewNoPermission
	}

	if item.Status == models.CWReviewItemStatusConfirmed {
		return nil
	}
	if item.Status != models.CWReviewItemStatusApplying {
		return repository.ErrCoursewareReviewItemConflict
	}

	if item.SourceType ==
		models.CWReviewItemSourceSelf &&
		strings.TrimSpace(
			item.AppliedPageHash,
		) != "" {
		err :=
			repository.RestoreSelfCoursewareReviewItemAfterReapplyAbort(
				ctx,
				item.ID,
				actor.UserID,
			)
		if err != nil {
			switch {
			case errors.Is(
				err,
				repository.ErrCoursewareReviewItemAppliedPageChanged,
			):
				return ErrCWReviewItemStale

			case errors.Is(
				err,
				repository.ErrCoursewareReviewItemAppliedPageMissing,
			):
				return ErrCWReviewItemOrphaned

			default:
				return err
			}
		}

		return nil
	}

	return repository.AbortCoursewareReviewItemInitialApplication(
		ctx,
		item.ID,
		actor.UserID,
	)
}

// CompleteCWReviewItemApplication 在页面写库成功后验证正式页面并记录applied.
//
// 本入口只确认“作者已经完成页面修改”这一事实。
// 是否真正解决，必须由后续明确检查动作完成。
func CompleteCWReviewItemApplication(
	ctx context.Context,
	itemID string,
	coursewareID string,
	pageNumber int,
	refinedHTML string,
	actor *CoursewareActorContext,
) (*CWReviewItemApplicationResult, error) {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}

	item, _, err := loadAuthorizedCWReviewItem(
		ctx,
		strings.TrimSpace(itemID),
		actor,
	)
	if err != nil {
		return nil, err
	}

	if actor.UserID != item.OwnerID {
		return nil, ErrCWAIReviewNoPermission
	}
	if item.CoursewareID != strings.TrimSpace(coursewareID) {
		return nil, repository.ErrCoursewareReviewItemNotFound
	}
	if item.PageID == nil ||
		strings.TrimSpace(*item.PageID) == "" ||
		pageNumber <= 0 {
		return nil, ErrCWReviewItemApplicationPageMismatch
	}

	// 已由人工确认解决的记录不因重复回调而重新打开。
	if item.Status == models.CWReviewItemStatusResolved {
		return &CWReviewItemApplicationResult{
			ItemID:          item.ID,
			Status:          models.CWReviewItemStatusResolved,
			AppliedPageHash: item.AppliedPageHash,
		}, nil
	}

	page, err := repository.GetCoursewareReviewPageSnapshotByID(
		ctx,
		strings.TrimSpace(*item.PageID),
		item.CoursewareID,
	)
	if err != nil {
		return nil, err
	}

	if page.PageNumber != pageNumber {
		return nil, ErrCWReviewItemApplicationPageMismatch
	}

	currentHash := cwAIReviewHash(page.HTMLContent)
	expectedHash := cwAIReviewHash(refinedHTML)

	if currentHash != expectedHash {
		// 页面在微调写入后又发生变化，不能宣称本次修改已经稳定完成。
		if item.Status == models.CWReviewItemStatusApplying &&
			item.SourceType == models.CWReviewItemSourceSelf &&
			strings.TrimSpace(item.AppliedPageHash) != "" {
			restoreErr :=
				repository.RestoreSelfCoursewareReviewItemAfterReapplyAbort(
					ctx,
					item.ID,
					actor.UserID,
				)
			if restoreErr != nil &&
				!errors.Is(
					restoreErr,
					repository.ErrCoursewareReviewItemAppliedPageChanged,
				) &&
				!errors.Is(
					restoreErr,
					repository.ErrCoursewareReviewItemAppliedPageMissing,
				) {
				return nil, restoreErr
			}
		} else if item.Status == models.CWReviewItemStatusApplying {
			if err :=
				repository.InvalidateCoursewareReviewItemInitialApplication(
					ctx,
					item.ID,
					actor.UserID,
					models.CWReviewItemStatusStale,
				); err != nil {
				return nil, err
			}
		}

		return nil, ErrCoursewarePageMutationConflict
	}

	switch item.Status {
	case models.CWReviewItemStatusApplying:
		if item.AppliedInstructionVersionID == nil ||
			strings.TrimSpace(*item.AppliedInstructionVersionID) == "" {
			return nil, ErrCWReviewItemApplicationVersionMismatch
		}

		if err := repository.MarkCoursewareReviewItemApplied(
			ctx,
			item.ID,
			actor.UserID,
			currentHash,
		); err != nil {
			return nil, err
		}

	case models.CWReviewItemStatusApplied:
		if strings.TrimSpace(item.AppliedPageHash) != "" &&
			item.AppliedPageHash != currentHash {
			return nil, ErrCoursewarePageMutationConflict
		}

	default:
		return nil, repository.ErrCoursewareReviewItemConflict
	}

	return &CWReviewItemApplicationResult{
		ItemID:          item.ID,
		Status:          models.CWReviewItemStatusApplied,
		AppliedPageHash: currentHash,
	}, nil
}
