package services

// courseware_collab_service.go — 课件工坊·集体备课（阶段4）
//
// 方法挂在 CoursewareService 上（与 courseware_share_service.go 同包同结构体），
// 不新建带依赖的结构体，沿用本包无状态服务 + repository 包级函数的既有风格。
//
// 集体备课设计（最小增量，线下聚集场景）：
//   系统只做三件事 —— ① 标记态 collab_state ② 共享微调权 ③ 留痕（走现有版本快照）。
//   不做实时在线协同、不做讨论楼层（议课走现有页级批注 courseware_annotations）。
//
// 本文件提供：
//   - CanEditCourseware  统一微调权限判定（作者/admin/集体备课参与者，且非锁定态）—— refine 三处卡点统一调用它
//   - StartCollab        发起集体备课（仅作者，标记 in_session，可选带首批参与者）
//   - EndCollab          结束集体备课（仅作者，回 idle + 清空参与者名单，彻底收权）
//   - AddCollabMember    加参与者（仅作者）
//   - RemoveCollabMember 移除参与者（仅作者）
//   - GetCollabStatus    查集体备课状态 + 参与者列表 + 当前请求者能否微调
//
// 权限边界（重要）：集体备课只共享"改页面内容（微调/重生/批注）"的权，
//   绝不共享"改课件状态 / 删课件 / 发布 / 提交审核"的权 —— 那些仍是作者专属。
//
// ★ 阶段5b：接入通知中心 ★
//   集体备课的四个动作（加人/移人/发起带人/结束）向相关用户旁路推送站内信：
//     - 加参与者 / 发起时拉入参与者 → 给【被拉的人】发 cw_collab_invited（"你被邀请参与某课件集体备课"）
//     - 移除参与者                  → 给【被移除的人】发 cw_collab_removed
//     - 结束集体备课                → 给【结束前的全体参与者】发 cw_collab_ended（批量）
//   去重：加人前先 IsCollabMember 探测，已是成员（重复拉）不重复发 invited。
//   旁路：全部走 GlobalNotificationService 的 best-effort 异步发送，写失败仅记日志，
//        绝不阻断也绝不回滚集体备课主业务（拉人成功了通知没发出去，业务照样成功）。
//   设计取舍：发起集体备课带的首批参与者与后续单独加的参与者，对被拉者体验一致，
//        统一发 cw_collab_invited；不区分"开局在场/中途被拉"，cw_collab_started 类型本版不接线。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// collabServiceLog 模块日志（复用 courseware_service.go 中的 cwServiceLog，同包）
var collabServiceLog = cwServiceLog

// ==================== 统一微调权限判定 ====================

// CanEditCourseware 判定"此刻 userID(role) 能否微调这个课件"。
//
// 这是集体备课放宽微调权的唯一裁决点，被 RefinePage/RegenerateSinglePage/RefineNav 三处统一调用，
// 取代它们原先硬编码的 `cw.UserID != userID`。
//
// 判定顺序（先卡锁定态，再判身份）：
//  1. 锁定态闸门：课件处于 status=in_pipeline（生产态·审核中）或 publish_state=submitted（发布态·已提交审核）
//     → 任何人都不能改（含作者本人、含参与者），防止改动审核中的内容。直接返回 false。
//  2. 身份判定：
//     - 作者本人（cw.UserID == userID）→ true
//     - 平台管理员（role == admin）→ true
//     - 集体备课进行中（cw.CollabState == in_session）且 userID 在参与者名单里 → true
//     - 其余 → false
//
// 入参 cw 由调用方先查好传入（避免本函数重复查库）；查参与者名单仅在前两条不命中时才查（省一次库）。
func (s *CoursewareService) CanEditCourseware(
	ctx context.Context,
	courseware *models.Courseware,
	userID string,
	role string,
) (bool, error) {
	if courseware == nil {
		return false, fmt.Errorf("课件不存在")
	}

	actor := BuildCoursewareActorFromClaims(
		ctx,
		userID,
		role,
	)
	return s.CanEditLoadedCourseware(
		ctx,
		courseware,
		actor,
	)
}

// canRefineCourseware 保持教研微调语义：作者或集体备课参与者。
// admin不因平台角色自动放行，但同样必须经过课件教育域校验。
func (s *CoursewareService) canRefineCourseware(
	ctx context.Context,
	courseware *models.Courseware,
	userID string,
) (bool, error) {
	if courseware == nil {
		return false, fmt.Errorf("课件不存在")
	}

	user, err := repository.FindUserByID(
		ctx,
		userID,
	)
	if err != nil {
		return false, err
	}

	actor := BuildCoursewareActorFromClaims(
		ctx,
		userID,
		user.Role,
	)
	return s.CanRefineLoadedCourseware(
		ctx,
		courseware,
		actor,
	)
}

// ==================== 发起 / 结束集体备课（仅作者）====================

// StartCollab 发起集体备课：把课件标记为 in_session，可选一次性拉入首批参与者。
//
// 仅作者可发起。锁定态（审核中）不允许发起（审核中的课件不该被集体改）。
// 幂等：已在 in_session 再次发起不报错，只把新传入的 members 追加进名单。
//
// 参数 members 为可选首批参与者用户ID数组（可为空，发起后再逐个加）。
//
// 阶段5b：每成功拉入一名首批参与者，旁路给其发 cw_collab_invited 通知（与 AddCollabMember 同口径）。
func (s *CoursewareService) StartCollab(ctx context.Context, coursewareID string, userID string, members []string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("只有课件作者可以发起集体备课")
	}
	// 锁定态不允许发起集体备课
	if cw.Status == models.CoursewareStatusInPipeline {
		return fmt.Errorf("课件已提交审核，不能发起集体备课")
	}
	if cw.PublishState == models.CWPublishSubmitted {
		return fmt.Errorf("课件正在审核中，不能发起集体备课")
	}

	// 标记 in_session（幂等：本就 in_session 也无妨）
	if err := repository.UpdateCoursewareCollabState(ctx, coursewareID, models.CWCollabInSession); err != nil {
		return err
	}

	// 可选：拉入首批参与者（逐个加，幂等；作者本人无需加入名单，跳过）
	added := 0
	for _, uid := range members {
		if uid == "" || uid == userID {
			continue // 跳过空值与作者本人
		}
		// 阶段5b：加之前先探重，决定是否需要发"被邀请"通知（已是成员则不重复发）
		wasMember, _ := repository.IsCollabMember(ctx, coursewareID, uid)
		if err := repository.AddCollabMember(ctx, coursewareID, uid, userID); err != nil {
			collabServiceLog.Warn("发起集体备课时加参与者失败（跳过该人，不阻断）",
				"courseware_id", coursewareID, "member", uid, "error", err)
			continue
		}
		added++
		// 仅"真新增"的参与者才发邀请通知（重复拉同一人不打扰）
		if !wasMember {
			s.emitCollabInvite(ctx, cw, uid)
		}
	}

	collabServiceLog.Info("发起集体备课",
		"courseware_id", coursewareID, "initiator", userID, "first_batch_added", added)
	return nil
}

// EndCollab 结束集体备课：课件回 idle，并清空参与者名单（彻底收回所有共享微调权）。
//
// 仅作者可结束。结束后参与者立即失去微调权（权限实时按 collab_state + 名单判定）。
// 清空名单是有意为之：本场集体备课的参与记录不长期保留，下次再发起重新拉人，语义干净。
//
// 阶段5b：在【清空名单之前】先抓全体参与者ID，结束后批量给他们发 cw_collab_ended 通知。
func (s *CoursewareService) EndCollab(ctx context.Context, coursewareID string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("只有课件作者可以结束集体备课")
	}

	// 阶段5b：清空名单前先抓参与者ID（清空后就拿不到了），用于结束后批量通知
	endRecipients := s.collectMemberRecipientIDs(ctx, coursewareID, userID)

	// 回 idle
	if err := repository.UpdateCoursewareCollabState(ctx, coursewareID, models.CWCollabIdle); err != nil {
		return err
	}
	// 清空参与者名单（收权）
	if err := repository.ClearCollabMembers(ctx, coursewareID); err != nil {
		// 名单清理失败不阻断"结束"主流程（collab_state 已回 idle，参与者已因状态失权），仅记日志
		collabServiceLog.Warn("结束集体备课时清空参与者名单失败（状态已回idle，参与者已失权）",
			"courseware_id", coursewareID, "error", err)
	}

	// 阶段5b：批量给结束前的全体参与者发"集体备课已结束"通知（旁路，失败不影响主流程）
	if len(endRecipients) > 0 {
		GlobalNotificationService.EmitNotificationBatch(endRecipients, models.EmitNotificationInput{
			Type:       models.NotifCWCollabEnded,
			Title:      fmt.Sprintf("课件《%s》的集体备课已结束", cw.Title),
			EntityType: models.NotifEntityCourseware,
			EntityID:   cw.ID,
			ActorID:    userID,
			ActorName:  s.resolveActorName(ctx, cw.UserID),
			Link:       "/courseware/" + cw.ID,
		})
	}

	collabServiceLog.Info("结束集体备课", "courseware_id", coursewareID, "initiator", userID,
		"notified", len(endRecipients))
	return nil
}

// ==================== 加 / 移 参与者（仅作者）====================

// AddCollabMember 给集体备课加一名参与者（仅作者）。
//
// 要求课件已处于 in_session（不能给"没在集体备课"的课件加人）。
// 幂等：重复加同一人不报错（仓储层 ON CONFLICT DO NOTHING）。
//
// 阶段5b：加成功且确为"真新增"（之前不在名单）时，旁路给被拉的人发 cw_collab_invited 通知。
func (s *CoursewareService) AddCollabMember(ctx context.Context, coursewareID string, userID string, targetUserID string) error {
	if targetUserID == "" {
		return fmt.Errorf("参与者用户ID不能为空")
	}
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("只有课件作者可以添加集体备课参与者")
	}
	if cw.CollabState != models.CWCollabInSession {
		return fmt.Errorf("请先发起集体备课，再添加参与者")
	}
	if targetUserID == userID {
		return fmt.Errorf("作者本人无需加入参与者名单（作者本就有微调权）")
	}
	// 阶段5b：插入前先探重，决定是否需要发"被邀请"通知（已是成员则不重复发）
	wasMember, _ := repository.IsCollabMember(ctx, coursewareID, targetUserID)
	if err := repository.AddCollabMember(ctx, coursewareID, targetUserID, userID); err != nil {
		return err
	}
	if !wasMember {
		s.emitCollabInvite(ctx, cw, targetUserID)
	}
	collabServiceLog.Info("集体备课加参与者",
		"courseware_id", coursewareID, "member", targetUserID, "by", userID)
	return nil
}

// RemoveCollabMember 移除一名参与者（仅作者）。移除后该用户立即失去微调权。
//
// 阶段5b：移除成功后，旁路给被移除的人发 cw_collab_removed 通知（语气温和，不点名移除者）。
func (s *CoursewareService) RemoveCollabMember(ctx context.Context, coursewareID string, userID string, targetUserID string) error {
	if targetUserID == "" {
		return fmt.Errorf("参与者用户ID不能为空")
	}
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("只有课件作者可以移除集体备课参与者")
	}
	if err := repository.RemoveCollabMember(ctx, coursewareID, targetUserID); err != nil {
		return err
	}
	// 阶段5b：旁路通知被移除者（不点名是谁移除的，语气温和）
	GlobalNotificationService.EmitNotification(models.EmitNotificationInput{
		RecipientID: targetUserID,
		Type:        models.NotifCWCollabRemoved,
		Title:       fmt.Sprintf("你已退出课件《%s》的集体备课", cw.Title),
		EntityType:  models.NotifEntityCourseware,
		EntityID:    cw.ID,
		ActorID:     userID,
		ActorName:   s.resolveActorName(ctx, cw.UserID),
		Link:        "/courseware/" + cw.ID,
	})
	collabServiceLog.Info("集体备课移除参与者",
		"courseware_id", coursewareID, "member", targetUserID, "by", userID)
	return nil
}

// ==================== 查询集体备课状态 ====================

// GetCollabStatus 查询某课件的集体备课状态：当前态 + 作者ID + 当前请求者能否微调 + 参与者列表。
//
// 任何登录者都可查（前端进工坊就要知道"这课件是不是在集体备课、我能不能改"）。
// CanEdit 复用 CanEditCourseware，保证"前端显隐微调入口"与"后端实际放行"完全同口径。
func (s *CoursewareService) GetCollabStatus(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) (*models.CollabStatusResponse, error) {
	cw, err := s.LoadCoursewareForView(
		ctx,
		coursewareID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	members, err := repository.ListCollabMembers(
		ctx,
		coursewareID,
	)
	if err != nil {
		return nil, err
	}

	canEdit := false
	domain := strings.ToLower(
		strings.TrimSpace(cw.EducationDomain),
	)

	// common资源可以被合法用户只读查看，但不能进入编辑或运行链。
	if models.IsTeachingEducationDomain(domain) {
		canEdit, err = s.CanEditLoadedCourseware(
			ctx,
			cw,
			actor,
		)
		if err != nil {
			return nil, err
		}
	}

	return &models.CollabStatusResponse{
		CoursewareID: cw.ID,
		CollabState:  cw.CollabState,
		OwnerID:      cw.UserID,
		CanEdit:      canEdit,
		Members:      members,
		Total:        len(members),
	}, nil
}

// ==================== 集体备课候选成员 ====================

// ListCollabCandidates 列出当前用户可拉入集体备课的候选成员（同校 + 同教研组）。
//
// 复用 resolveSameOrgUserIDs（share 服务的同校同组解析零件）拿到候选用户ID集合，
// 再批量查基本信息（仓储层已过滤 admin/viewer/非active、排序）。
// 最后剔除本人（作者本就有微调权，无需加入名单）。
func (s *CoursewareService) ListCollabCandidates(ctx context.Context, userID string) (*models.CollabCandidateListResponse, error) {
	// 1) 解析同校同组成员ID（含本人）
	ids := s.resolveSameOrgUserIDs(ctx, userID)

	// 2) 批量查基本信息（仓储层过滤 admin/viewer/非active）
	cands, err := repository.ListUsersBasicByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// 3) 剔除本人
	out := make([]*models.CollabCandidate, 0, len(cands))
	for _, c := range cands {
		if c.UserID == userID {
			continue
		}
		out = append(out, c)
	}

	return &models.CollabCandidateListResponse{
		Candidates: out,
		Total:      len(out),
	}, nil
}

// ==================== 我参与的集体备课（参与者入口）====================

// ListJoinedCollab 列出当前用户作为参与者被拉入、且仍在 in_session 的课件。
// 供参与者（非作者）的"我参与的集体备课"入口使用。薄封装仓储查询。
func (s *CoursewareService) ListJoinedCollab(ctx context.Context, userID string) (*models.JoinedCollabListResponse, error) {
	items, err := repository.ListJoinedCollabCoursewares(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &models.JoinedCollabListResponse{
		Coursewares: items,
		Total:       len(items),
	}, nil
}

// ==================== 阶段5b：通知中心旁路辅助 ====================

// emitCollabInvite 给被拉入集体备课的参与者发一条"你被邀请参与集体备课"通知。
//
// 旁路 best-effort（GlobalNotificationService 内部异步、失败仅记日志），调用方无需处理返回。
// 作者名取不到时（如作者是 admin 被 ListUsersBasicByIDs 过滤），文案自动退化为不带名字的版本，
// 通知照常发出，不影响可用性。
//
// 入参 cw 由调用方传入（已在手，避免重复查库）；targetUserID 为被邀请人。
func (s *CoursewareService) emitCollabInvite(ctx context.Context, cw *models.Courseware, targetUserID string) {
	if cw == nil || targetUserID == "" {
		return
	}
	actorName := s.resolveActorName(ctx, cw.UserID)

	// 文案：能拿到作者名则"「X」邀请你…"，拿不到则退化为"你被邀请…"
	var title string
	if actorName != "" {
		title = fmt.Sprintf("「%s」邀请你参与课件《%s》的集体备课", actorName, cw.Title)
	} else {
		title = fmt.Sprintf("你被邀请参与课件《%s》的集体备课", cw.Title)
	}

	GlobalNotificationService.EmitNotification(models.EmitNotificationInput{
		RecipientID: targetUserID,
		Type:        models.NotifCWCollabInvited,
		Title:       title,
		EntityType:  models.NotifEntityCourseware,
		EntityID:    cw.ID,
		ActorID:     cw.UserID,
		ActorName:   actorName,
		Link:        "/courseware/" + cw.ID,
	})
}

// collectMemberRecipientIDs 抓取某课件当前全体参与者的用户ID（用于结束集体备课时批量通知）。
//
// 必须在 ClearCollabMembers 之前调用（清空后就查不到了）。
// 剔除作者本人（作者发起的结束，不必通知自己）。查询失败仅记日志返回空切片，绝不阻断主流程。
func (s *CoursewareService) collectMemberRecipientIDs(ctx context.Context, coursewareID string, ownerID string) []string {
	members, err := repository.ListCollabMembers(ctx, coursewareID)
	if err != nil {
		collabServiceLog.Warn("结束集体备课时拉取参与者名单失败（跳过结束通知，不阻断）",
			"courseware_id", coursewareID, "error", err)
		return nil
	}
	out := make([]string, 0, len(members))
	for _, m := range members {
		if m == nil || m.UserID == "" || m.UserID == ownerID {
			continue
		}
		out = append(out, m.UserID)
	}
	return out
}

// resolveActorName 解析触发人（通常是课件作者）的显示名，best-effort。
//
// 复用现成的 repository.ListUsersBasicByIDs 批量查接口（传单个ID）。注意该接口会过滤掉
// admin/viewer 角色，故作者若为 admin 会查不到——此时返回空串，调用方文案自动退化，不报错。
func (s *CoursewareService) resolveActorName(ctx context.Context, actorID string) string {
	if actorID == "" {
		return ""
	}
	users, err := repository.ListUsersBasicByIDs(ctx, []string{actorID})
	if err != nil || len(users) == 0 {
		return ""
	}
	if users[0].DisplayName != "" {
		return users[0].DisplayName
	}
	return users[0].Username
}
