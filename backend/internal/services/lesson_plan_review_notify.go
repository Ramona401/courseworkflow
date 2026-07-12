package services

// lesson_plan_review_notify.go — 教案审核·通知中心旁路（阶段5 收尾·教案审核接线）
//
// 镜像 courseware_review_notify.go，把教案多级审核（L1/L2）的三个流转事件接进通知中心。
// 与课件审核旁路的结构性差异（也是本文件单独存在的原因）：
//   课件审核三动作（SubmitForReview/ReviewL1/ReviewL2）同属一个 CoursewareReviewService，
//   故课件那边把旁路方法挂在该结构体上即可。
//   教案则跨两个 service：
//     - 提交审核 SubmitForReview        在 LessonPlanService（lesson_plan_service.go）
//     - L1/L2 审核 ReviewL1/ReviewL2     在 ReviewV2Service（review_v2_service.go）
//   两个结构体若各写一遍方法会重复，因此本文件提供【包级函数】（不挂任何结构体），
//   两个 service 都能直接调用，这是跨 service 共享旁路逻辑的更自然写法。
//
// 全程 best-effort（经 GlobalNotificationService 异步写入），写失败仅记日志，
// 绝不阻断、绝不回滚审核/提交主业务（与 audit_repo.WriteAuditLog 同范式）。
//
// 三个接线点（在各 service 对应方法尾部调用）：
//   - notifyLPReviewersOnSubmit  提交成功 → 通知作者所属教研组的 L1 审核员（lp_review_submitted）
//   - notifyLPAuthorReviewResult approved/revision → 通知作者（lp_review_approved / lp_review_revision）
//
// 收件人解析的关键差异（相比课件）：
//   课件作者可能在多个教研组，需遍历所有组收集 L1 审核员。
//   教案提交审核时已绑定唯一 group_id（SubmitForReview 写入 lp.GroupID），
//   故 L1 审核员就是【那一个组】的 lead/backbone——更精确，直接用 lp.GroupID + ListGroupMembers。

import (
	"context"
	"fmt"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// lpNotifyLog 教案审核通知旁路独立日志器（与主审核服务日志区分，便于排查通知问题）。
var lpNotifyLog = logger.WithModule("lesson_plan_review_notify")

// notifyLPReviewersOnSubmit 教案提交审核成功后，给"该教案绑定教研组的 lead/backbone（L1 审核员）"
// 旁路发"有新教案待审"通知。
//
// 与课件不同：教案提交时已确定唯一 group_id（lp.GroupID），故收件人就是那一个组的
// lead/backbone，无需遍历作者所有教研组。
//
// 全程 best-effort：任一步查询失败仅记日志，不阻断提交（提交已成功落库）。
// 教案无 group_id（理论上 SubmitForReview 已保证非空，此处仍防御）、或组内无 lead/backbone、
// 或唯一审核员就是作者本人时，无人可通知则静默跳过。
func notifyLPReviewersOnSubmit(ctx context.Context, lp *models.LessonPlan) {
	if lp == nil {
		return
	}
	if lp.GroupID == nil || *lp.GroupID == "" {
		// 提交审核必绑教研组（SubmitForReview 已校验 groupID 非空），此处为防御性兜底
		lpNotifyLog.Warn("提交通知：教案无关联教研组，跳过通知", "plan_id", lp.ID)
		return
	}

	reviewerIDs := collectLPL1ReviewerIDs(ctx, *lp.GroupID, lp.AuthorID)
	if len(reviewerIDs) == 0 {
		return // 组内无 L1 审核员（或唯一审核员是作者本人），静默跳过
	}

	actorName := resolveLPReviewerName(ctx, lp.AuthorID) // 作者名（提交人）
	var title string
	if actorName != "" {
		title = fmt.Sprintf("「%s」提交了教案《%s》，请审核", actorName, lp.Title)
	} else {
		title = fmt.Sprintf("有新教案《%s》提交审核，请审核", lp.Title)
	}

	GlobalNotificationService.EmitNotificationBatch(reviewerIDs, models.EmitNotificationInput{
		Type:       models.NotifLPReviewSubmitted,
		Title:      title,
		EntityType: models.NotifEntityLessonPlan,
		EntityID:   lp.ID,
		ActorID:    lp.AuthorID,
		ActorName:  actorName,
		// 审核员点通知应进入教案多级审核工作台（与课件审核指向审核中心同理）
		Link: "/lesson-plans/review-v2",
	})
}

// collectLPL1ReviewerIDs 解析"指定教研组内全体 lead/backbone"的去重 user_id 列表（剔除作者本人）。
//
// 步骤：ListGroupMembers(groupID) 取该组全体成员含 role → 过滤 role∈(lead,backbone)
//
//	→ 剔除作者本人 → 去重。
//
// 查询失败仅记日志返回 nil（best-effort），绝不阻断。
func collectLPL1ReviewerIDs(ctx context.Context, groupID string, authorID string) []string {
	members, err := repository.ListGroupMembers(ctx, groupID)
	if err != nil {
		lpNotifyLog.Warn("提交通知：拉取教研组成员失败（跳过通知，不阻断）",
			"group_id", groupID, "error", err)
		return nil
	}

	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, m := range members {
		if m == nil {
			continue
		}
		// 仅 lead/backbone 是 L1 审核员
		if m.Role != "lead" && m.Role != "backbone" {
			continue
		}
		// 剔除作者本人 + 去重 + 防空ID
		if m.UserID == authorID || m.UserID == "" {
			continue
		}
		if _, dup := seen[m.UserID]; dup {
			continue
		}
		seen[m.UserID] = struct{}{}
		out = append(out, m.UserID)
	}
	return out
}

// notifyLPAuthorReviewResult 教案审核决策后给作者发结果通知（approved 通过 / revision 退回）。
//
// decision=approved → lp_review_approved，文案"审核通过了"，Body 空。
// decision=revision → lp_review_revision，文案"被退回，请修改后重新提交"，审核意见放进 Body。
// 其它决策（如旧版 rejected）不在此发送——本函数只接 review_v2 的 approved/revision 两态。
//
// 旁路 best-effort；作者ID取 lp.AuthorID，审核员名取 reviewerID 解析（用于 ActorName）。
func notifyLPAuthorReviewResult(ctx context.Context, lp *models.LessonPlan, reviewerID string, decision string, comment string) {
	if lp == nil || lp.AuthorID == "" {
		return
	}
	reviewerName := resolveLPReviewerName(ctx, reviewerID)

	var notifType, title, body string
	switch decision {
	case models.ReviewDecisionApproved:
		notifType = models.NotifLPReviewApproved
		title = fmt.Sprintf("你的教案《%s》审核通过了", lp.Title)
		body = ""
	case models.ReviewDecisionRevision:
		notifType = models.NotifLPReviewRevision
		title = fmt.Sprintf("你的教案《%s》被退回，请修改后重新提交", lp.Title)
		body = comment // 审核意见进 Body，作者点开能看到为什么被退
	default:
		return // 其它决策不通知
	}

	GlobalNotificationService.EmitNotification(models.EmitNotificationInput{
		RecipientID: lp.AuthorID,
		Type:        notifType,
		Title:       title,
		Body:        body,
		EntityType:  models.NotifEntityLessonPlan,
		EntityID:    lp.ID,
		ActorID:     reviewerID,
		ActorName:   reviewerName,
		// 作者点通知进入该教案详情页（查看审核意见、继续修改）
		Link: "/lesson-plans/plans/" + lp.ID,
	})
}

// resolveLPReviewerName 解析用户显示名，best-effort。
//
// 复用 repository.ListUsersBasicByIDs（传单个ID）。注意该接口会过滤 admin/viewer 角色——
// 审核员若为 admin、或作者若为 viewer，会查不到返回空串，调用方文案自动退化为不带名版本，不报错。
// 教案审核场景里作者通常是 operator、L1 审核员通常是 lead/backbone(operator)、L2 是 senior_operator，
// 均不被过滤能正常拿到名字；admin 审核时退化为不带名的文案。
func resolveLPReviewerName(ctx context.Context, userID string) string {
	if userID == "" {
		return ""
	}
	users, err := repository.ListUsersBasicByIDs(ctx, []string{userID})
	if err != nil || len(users) == 0 {
		return ""
	}
	if users[0].DisplayName != "" {
		return users[0].DisplayName
	}
	return users[0].Username
}
