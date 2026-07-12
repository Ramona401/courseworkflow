package services

// courseware_review_notify.go — 课件审核·通知中心旁路（阶段5c）
//
// 从 courseware_review_service.go 拆出（主文件加通知后破 600 行红线，按铁律拆分）。
// 本文件只承载"审核流转 → 站内信"的旁路逻辑，挂在 *CoursewareReviewService 上，
// 与主审核逻辑同包同结构体。全程 best-effort 异步（经 GlobalNotificationService），
// 写失败仅记日志，绝不阻断也绝不回滚审核主业务。
//
// 三个接线点（在主文件 SubmitForReview / ReviewL1 / ReviewL2 尾部调用）：
//   - notifyL1ReviewersOnSubmit  提交成功 → 通知作者教研组全体 lead/backbone（cw_review_submitted）
//   - notifyAuthorReviewResult   approved/revision → 通知作者（cw_review_approved / cw_review_revision）
//
// 设计取舍（本版）：L1 通过进 L2 的内部流转不单独推送 L2 审核员（senior 主动看审核中心列表），
//   cw_review_submitted 仅在"作者提交"那一刻发给 L1 审核员，语义完整。

import (
        "context"
        "fmt"

        "tedna/internal/models"
        "tedna/internal/repository"
)

// notifyL1ReviewersOnSubmit 提交审核成功后，给"作者所属教研组的全体 lead/backbone（L1 审核员）"
// 旁路发"有新课件待审"通知。
//
// 收件人解析：作者可能在多个教研组，逐组列成员、过滤 role∈(lead,backbone)、跨组去重、剔除作者本人。
// 全程 best-effort：任一步查询失败仅记日志，不阻断提交主业务（提交已成功落库）。
// 没有任何 L1 审核员（如作者不在任何教研组、或组里无 lead/backbone）时静默跳过，不发通知。
func (s *CoursewareReviewService) notifyL1ReviewersOnSubmit(ctx context.Context, cw *models.Courseware) {
        if cw == nil {
                return
        }
        reviewerIDs := s.collectL1ReviewerIDs(ctx, cw.UserID)
        if len(reviewerIDs) == 0 {
                return // 无审核员可通知，静默跳过
        }

        actorName := s.resolveUserName(ctx, cw.UserID) // 作者名（提交人）
        var title string
        if actorName != "" {
                title = fmt.Sprintf("「%s」提交了课件《%s》，请审核", actorName, cw.Title)
        } else {
                title = fmt.Sprintf("有新课件《%s》提交审核，请审核", cw.Title)
        }

        GlobalNotificationService.EmitNotificationBatch(reviewerIDs, models.EmitNotificationInput{
                Type:       models.NotifCWReviewSubmitted,
                Title:      title,
                EntityType: models.NotifEntityCourseware,
                EntityID:   cw.ID,
                ActorID:    cw.UserID,
                ActorName:  actorName,
                // 审核员点通知应进入审核中心，而非课件工坊；此处统一指向审核中心待审入口
                Link: "/courseware/review",
        })
}

// collectL1ReviewerIDs 解析"作者所属教研组的全体 lead/backbone"的去重 user_id 列表（剔除作者本人）。
//
// 步骤：GetUserTeachingGroups(作者) 拿作者所有 active 教研组 → 逐组 ListGroupMembers 取成员含 role
//   → 过滤 role∈(lead,backbone) → map 去重 → 剔除作者本人。
// 任一步失败仅记日志返回已收集到的部分（best-effort），绝不阻断。
func (s *CoursewareReviewService) collectL1ReviewerIDs(ctx context.Context, authorID string) []string {
        groups, err := repository.GetUserTeachingGroups(ctx, authorID)
        if err != nil {
                cwReviewLog.Warn("提交通知：拉取作者教研组失败（跳过通知，不阻断）",
                        "author", authorID, "error", err)
                return nil
        }
        if len(groups) == 0 {
                return nil
        }

        seen := make(map[string]struct{})
        out := make([]string, 0)
        for _, g := range groups {
                members, mErr := repository.ListGroupMembers(ctx, g.ID)
                if mErr != nil {
                        cwReviewLog.Warn("提交通知：拉取教研组成员失败（跳过该组，不阻断）",
                                "group_id", g.ID, "error", mErr)
                        continue
                }
                for _, m := range members {
                        if m == nil {
                                continue
                        }
                        // 仅 lead/backbone 是 L1 审核员
                        if m.Role != "lead" && m.Role != "backbone" {
                                continue
                        }
                        // 剔除作者本人 + 去重
                        if m.UserID == authorID || m.UserID == "" {
                                continue
                        }
                        if _, dup := seen[m.UserID]; dup {
                                continue
                        }
                        seen[m.UserID] = struct{}{}
                        out = append(out, m.UserID)
                }
        }
        return out
}

// notifyAuthorReviewResult 审核决策后给作者发结果通知（approved 通过 / revision 退回）。
//
// decision=approved → cw_review_approved，文案"审核通过了"，Body 空。
// decision=revision → cw_review_revision，文案"被退回，请修改后重新提交"，审核意见放进 Body。
// 旁路 best-effort；作者ID取 cw.UserID，审核员名取 reviewerID（用于 ActorName）。
func (s *CoursewareReviewService) notifyAuthorReviewResult(ctx context.Context, cw *models.Courseware, reviewerID string, decision string, comment string) {
        if cw == nil || cw.UserID == "" {
                return
        }
        reviewerName := s.resolveUserName(ctx, reviewerID)

        var notifType, title, body string
        switch decision {
        case models.ReviewDecisionApproved:
                notifType = models.NotifCWReviewApproved
                title = fmt.Sprintf("你的课件《%s》审核通过了", cw.Title)
                body = ""
        case models.ReviewDecisionRevision:
                notifType = models.NotifCWReviewRevision
                title = fmt.Sprintf("你的课件《%s》被退回，请修改后重新提交", cw.Title)
                body = comment // 审核意见进 Body，作者点开能看到为什么被退
        default:
                return // 其它决策不通知
        }

        GlobalNotificationService.EmitNotification(models.EmitNotificationInput{
                RecipientID: cw.UserID,
                Type:        notifType,
                Title:       title,
                Body:        body,
                EntityType:  models.NotifEntityCourseware,
                EntityID:    cw.ID,
                ActorID:     reviewerID,
                ActorName:   reviewerName,
                Link:        "/courseware/" + cw.ID,
        })
}

// resolveUserName 解析用户显示名，best-effort。
//
// 复用 repository.ListUsersBasicByIDs（传单个ID）。注意该接口会过滤 admin/viewer 角色——
// 审核员若为 admin、或作者若为 viewer，会查不到返回空串，调用方文案自动退化，不报错。
// 课件审核场景里作者通常是 operator、审核员通常是 lead/backbone(operator) 或 senior_operator，
// 均不被过滤，能正常拿到名字；admin 审核时退化为不带名的文案。
func (s *CoursewareReviewService) resolveUserName(ctx context.Context, userID string) string {
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
