package handlers

// courseware_refine_review_item_helper.go
//
// 页面AI微调与审核问题进展之间的编排辅助。
//
// 页面修改开始：
//   - 复核浏览器提交的instruction_version_id；
//   - 事务内绑定applied_instruction_version_id；
//   - confirmed -> applying；
//   - 返回数据库确认的page_id、页码和页面HTML哈希守卫。
//
// 页面修改失败：
//   - applying -> confirmed；
//   - 数据库守卫清除尚未完成的临时应用版本引用。
//
// 页面修改成功：
//   - applying -> applied。
//
// applied只表示页面修改已经完成：
//   - 正式审核问题等待审核员复审；
//   - 作者自审问题等待作者本人检查效果。
//
// 状态收尾使用WithoutCancel上下文，避免浏览器断开后留下长期applying状态。

import (
	"context"
	"log/slog"
	"strings"

	"tedna/internal/services"
)

const cwReviewItemSyncWarning = "页面修改已完成，但整改项状态同步失败，请刷新整改中心检查"

// beginCoursewareReviewItemRefineApplication 开始本次问题对应的页面修改。
//
// 返回值必须直接传给RefinePageWithModeGuarded。
func beginCoursewareReviewItemRefineApplication(
	ctx context.Context,
	reviewItemID string,
	coursewareID string,
	pageNumber int,
	instructionVersionID string,
	instruction string,
	actor *services.CoursewareActorContext,
) (*services.CoursewarePageMutationGuard, error) {
	reviewItemID = strings.TrimSpace(reviewItemID)
	if reviewItemID == "" {
		return nil, nil
	}

	return services.BeginCWReviewItemApplication(
		ctx,
		reviewItemID,
		coursewareID,
		pageNumber,
		instructionVersionID,
		instruction,
		actor,
	)
}

// abortCoursewareReviewItemRefineApplication 在页面微调失败时恢复confirmed。
func abortCoursewareReviewItemRefineApplication(
	ctx context.Context,
	reviewItemID string,
	coursewareID string,
	pageNumber int,
	actor *services.CoursewareActorContext,
) {
	reviewItemID = strings.TrimSpace(reviewItemID)
	if reviewItemID == "" {
		return
	}

	cleanupContext := context.WithoutCancel(ctx)

	if err := services.AbortCWReviewItemApplication(
		cleanupContext,
		reviewItemID,
		actor,
	); err != nil {
		slog.ErrorContext(
			cleanupContext,
			"页面微调失败后恢复整改项状态失败",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNumber,
			"review_item_id",
			reviewItemID,
			"error",
			err,
		)
	}
}

// completeCoursewareReviewItemRefineApplication 完成页面写入后的进展记录。
//
// 成功后只记录applied，不在这里判断问题是否真正解决。
func completeCoursewareReviewItemRefineApplication(
	ctx context.Context,
	reviewItemID string,
	coursewareID string,
	pageNumber int,
	refinedHTML string,
	actor *services.CoursewareActorContext,
) (string, string) {
	reviewItemID = strings.TrimSpace(reviewItemID)
	if reviewItemID == "" {
		return "", ""
	}

	cleanupContext := context.WithoutCancel(ctx)

	result, err := services.CompleteCWReviewItemApplication(
		cleanupContext,
		reviewItemID,
		coursewareID,
		pageNumber,
		refinedHTML,
		actor,
	)
	if err != nil {
		slog.ErrorContext(
			cleanupContext,
			"页面微调完成后记录整改进展失败",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNumber,
			"review_item_id",
			reviewItemID,
			"error",
			err,
		)

		return "", cwReviewItemSyncWarning
	}

	if result == nil {
		slog.ErrorContext(
			cleanupContext,
			"页面微调整改项状态结果为空",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNumber,
			"review_item_id",
			reviewItemID,
		)

		return "", cwReviewItemSyncWarning
	}

	return result.Status, ""
}
