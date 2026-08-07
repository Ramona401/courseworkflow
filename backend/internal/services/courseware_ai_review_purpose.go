package services

// courseware_ai_review_purpose.go
//
// 课件AI分析存在两种业务用途：
//
//   1. 正式审核辅助：
//      - review_level = 1 或 2；
//      - 操作者是具有当前L1/L2审核权限的审核员；
//      - 助手场景为courseware_review；
//      - 输出可包含人工审核意见草稿；
//      - AI不得代替审核员提交通过或退回。
//
//   2. 作者课件自审：
//      - review_level = 0；
//      - 操作者只能是课件作者本人；
//      - 助手场景为courseware_self_review；
//      - 输出重点是定位问题、形成修改清单和再次自检；
//      - 自审结果不进入正式审核记录，不修改publish_state。
//
// 两种用途共用页面索引、互动代码分析、顺序分批、连续性账本和最终报告协议。

import "tedna/internal/models"

// isCWAIReviewSelfReview 判断会话是否是作者课件自审。
func isCWAIReviewSelfReview(
	session *models.CoursewareAIReviewSession,
) bool {
	return session != nil &&
		session.ReviewLevel ==
			models.CWAIReviewLevelSelf
}

// cwAIReviewPurposeCode 返回稳定的业务用途代码。
func cwAIReviewPurposeCode(
	session *models.CoursewareAIReviewSession,
) string {
	if isCWAIReviewSelfReview(session) {
		return "courseware_self_review"
	}

	return "formal_courseware_review"
}

// cwAIReviewAssistantHeading 返回助手提示词区块标题。
func cwAIReviewAssistantHeading(
	session *models.CoursewareAIReviewSession,
) string {
	if isCWAIReviewSelfReview(session) {
		return "【作者选择的AI课件自审助手个性化视角】"
	}

	return "【审核员选择的AI课件审核助手个性化视角】"
}

// cwAIReviewTraceScene 返回AI调用追踪场景。
func cwAIReviewTraceScene(
	session *models.CoursewareAIReviewSession,
) string {
	if isCWAIReviewSelfReview(session) {
		return "courseware_ai_self_review"
	}

	return "courseware_ai_review"
}

// cwAIReviewActionLabel 返回当前分析动作中文名称。
func cwAIReviewActionLabel(
	session *models.CoursewareAIReviewSession,
) string {
	if isCWAIReviewSelfReview(session) {
		return "自审"
	}

	return "审核"
}
