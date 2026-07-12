package services

// courseware_review_service.go — 课件多级审核核心业务逻辑（阶段3）
//
// 镜像 review_v2_service.go，三大本质改写：
//
//  1. 状态机载体：教案用 lesson_plans.status，课件用与 status 正交的 coursewares.publish_state。
//     全程绝不改动 courseware.status（生产状态机）。状态流转：
//       提交审核     publish_state→submitted, review_level→0, review_school_id→作者学校
//       L1通过(无L2) publish_state→approved,  review_level→1（待发布）
//       L1通过(有L2) publish_state 保持 submitted, review_level→1（进入L2）
//       L2通过       publish_state→approved,  review_level→2
//       退回         publish_state→revision,  review_level→0
//
//  2. 审核记录表：courseware_reviews（本期新建），写入走 repository.CreateCoursewareReview。
//
//  3. 审核流程配置：【复用】教案 review_flow_configs 表（按 school_id），
//     直接调 repository.GetReviewFlowConfig（review_v2_repo.go 已有），同校 l2_enabled 教案课件共用。
//
// 决策落地：
//   - 决策一（L2 沿用教案 school 级 l2_enabled）：L1 通过后查 GetReviewFlowConfig.L2Enabled 决定是否进 L2。
//   - 决策二（审核台联动批注）：GetReviewDetail 返回课件详情 + 全部批注 + 审核历史，审核员边看边决策。
//   - 决策三（退回回流）：退回置 publish_state=revision，作者改完可重新提交（SubmitForReview 放行 revision）。
//
// 权限：L1=作者所属教研组的 lead/backbone，或本校 senior_operator 兜底（方案B），或 admin；
//       L2=作者学校的 senior_operator（或 admin）。
//
// 审核级别常量 ReviewLevelL1/L2、决策常量 ReviewDecisionApproved/Revision 复用 models 包（review_v2.go 定义）。
//
// ★ 阶段5c：接入通知中心 ★
//   审核流转的关键节点向相关用户旁路推送站内信（best-effort 异步，写失败仅记日志，绝不阻断审核主业务）：
//     - SubmitForReview 提交成功 → 给【作者所属教研组的全体 lead/backbone（L1 审核员）】发 cw_review_submitted
//         （去重 + 剔除作者本人；让审核员第一时间知道"有新课件待审"）
//     - ReviewL1/L2 决策 approved 终审通过（publish_state→approved）→ 给【作者】发 cw_review_approved
//     - ReviewL1/L2 决策 revision 退回 → 给【作者】发 cw_review_revision（审核意见 comment 放进 Body）
//   设计取舍（本版）：L1 通过进 L2 的内部流转【不】单独推送 L2 审核员——senior_operator 本就会去
//     审核中心待审列表（GetPendingReviews 给 senior 返 L1+L2）主动查看，无需为此再加一个"列某校
//     senior_operator"的查询。cw_review_submitted 只在"作者提交"那一刻发给 L1 审核员，语义完整。
//
// ★ B3 修复（账户与权限·第一批）★
//   1. GetPendingReviews 新增 region_admin 分支：经唯一数据范围解析器 ResolveDataScope
//      取"辖区学校 ∪ 本校"（B2 双来源/双重身份并集直接生效），L1+L2 待审均按
//      review_school_id ∈ 辖区收窄（repository.ListCWPendingReviewsBySchools）。只读视图——
//      审核决策权不变（ReviewL1 组内 lead/backbone 或本校 senior、ReviewL2 本校 senior 或 admin）。
//   2. GetReviewStats 按角色装配学校白名单，堵住统计卡全局泄漏（历史上区域/学校管理员
//      能看到全系统待审数），口径与各角色待审列表严格一致。
//   3. canReviewCourseware 放行 region_admin 对辖区课件（review_school_id ∈ 辖区）的
//      只读审核详情，供辖区视图点开查看。
//
// ★ 方案B（2026-07-03，学校管理员审核盲区修复）★
//   问题：senior_operator 若不兼任教研组 lead/backbone，看不到本校 L1 待审课件（课件先落 L1，
//   未过 L1 不进 L2 列表）；组无组长/骨干时课件卡死在 L1。产品决策：本校 senior 对本校 L1
//   "可见 + 可审（兜底代审）"。落地四处：
//     1. GetPendingReviews senior 分支：L1 从"组内口径"升级为"本校全量口径"（复用 B3 的
//        ListCWPendingReviewsBySchools 传本校单元素白名单，repo 零改动）；学校解析失败降级回老口径。
//     2. GetReviewStats senior 分支：L1/L2 统一按本校白名单计数，与列表新口径严格一致。
//     3. ReviewL1 权限阶梯：admin → 本校 senior（isSeniorOfReviewSchool）→ 组内 lead/backbone。
//     4. canReviewCourseware：本校 senior 可打开本校待审课件详情（在 courseware_review_access.go）。
//   权限裁决类辅助方法（canReviewCourseware / isSeniorOfReviewSchool /
//   isReviewerInAuthorGroupAsLeadOrBackbone / resolveReviewSchoolID）已拆至
//   courseware_review_access.go（同包同接收器），主文件守 600 行红线。

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrCWReviewCoursewareNotFound = errors.New("课件不存在")
	ErrCWReviewNotSubmitted       = errors.New("只有已提交审核的课件可以审核")
	ErrCWReviewNotL2Status        = errors.New("该课件不在L2待审核状态")
	ErrCWReviewNoPermission       = errors.New("您没有审核此课件的权限")
	ErrCWReviewInvalidDecision    = errors.New("审核决策无效，可选值：approved/revision")
	// 提交审核相关
	ErrCWSubmitNotOwner   = errors.New("只有课件作者本人可以提交审核")
	ErrCWSubmitNotReady   = errors.New("课件尚未生成完成，请先完成课件（至少进入预览阶段）再提交审核")
	ErrCWSubmitWrongState = errors.New("当前状态不可提交审核（仅私有/个人发布/已退回的课件可提交）")
	ErrCWSubmitNoSchool   = errors.New("无法确定您所属的学校，请联系管理员配置组织归属后再提交审核")
)

// cwReviewLog 模块日志
var cwReviewLog = logger.WithModule("courseware_review")

// CoursewareReviewService 课件多级审核服务
// 不持有外部依赖（审核无自动萃取等副作用），构造简单。
type CoursewareReviewService struct{}

// NewCoursewareReviewService 创建课件多级审核服务实例
func NewCoursewareReviewService() *CoursewareReviewService {
	return &CoursewareReviewService{}
}

// ==================== 提交审核（阶段3状态机起点，作者发起）====================

// SubmitForReview 作者提交课件进入审核流。
//
// 前置：
//   - 仅作者本人可提交。
//   - 课件须已生成到可展示（status≥preview），避免送审半成品。
//   - 当前 publish_state 须 ∈ {private, published_personal, revision}（revision=被退回后重新提交，回流）。
//   - 反查作者所属学校 school_id（写入 review_school_id 供 L2 按校过滤）；查不到则拒绝。
//
// 成功后：publish_state→submitted, review_level→0, review_school_id→作者学校。
//
// 阶段5c：提交成功后，旁路给作者教研组的全体 L1 审核员（lead/backbone）发"有新课件待审"通知。
func (s *CoursewareReviewService) SubmitForReview(ctx context.Context, coursewareID string, userID string) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return ErrCWReviewCoursewareNotFound
	}
	if cw.UserID != userID {
		return ErrCWSubmitNotOwner
	}

	// 生产态校验：至少 preview
	if models.CoursewareStatusOrder[cw.Status] < models.CoursewareStatusOrder[models.CoursewareStatusPreview] {
		return ErrCWSubmitNotReady
	}

	// 发布态校验：仅 private / published_personal / revision 可提交
	switch cw.PublishState {
	case models.CWPublishPrivate, models.CWPublishPublishedPersonal, models.CWPublishRevision:
		// ok
	default:
		return ErrCWSubmitWrongState
	}

	// 反查作者学校
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	if schoolID == "" {
		return ErrCWSubmitNoSchool
	}

	if err := repository.UpdateCoursewarePublishState(ctx, coursewareID, models.CWPublishSubmitted, 0, &schoolID); err != nil {
		return err
	}
	cwReviewLog.Info("课件提交审核",
		"courseware_id", coursewareID, "author", userID, "school_id", schoolID)

	// 阶段5c：旁路通知 L1 审核员（作者教研组 lead/backbone）
	s.notifyL1ReviewersOnSubmit(ctx, cw)
	return nil
}

// ==================== L1 教研组审核 ====================

// ReviewL1 L1 教研组审核决策。
//
// 权限阶梯（方案B，2026-07-03）：
//   admin 直放 → 本校 senior_operator 兜底可审（isSeniorOfReviewSchool，组长缺位时流程不死锁）
//   → 作者教研组的 lead/backbone。
// 角色经 FindUserByID 现查（沿用既有模式，避免改 handler 签名与路由）。
func (s *CoursewareReviewService) ReviewL1(ctx context.Context, coursewareID string, reviewerID string, req *models.CWReviewDecisionRequest) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return ErrCWReviewCoursewareNotFound
	}

	if cw.PublishState != models.CWPublishSubmitted || cw.ReviewLevel != 0 {
		return ErrCWReviewNotSubmitted
	}

	// 权限阶梯：admin → 本校 senior 兜底 → 组内 lead/backbone
	reviewerRole := ""
	if u, uErr := repository.FindUserByID(ctx, reviewerID); uErr == nil {
		reviewerRole = u.Role
	}
	allowed := reviewerRole == models.RoleAdmin
	if !allowed && reviewerRole == models.RoleSeniorOperator {
		// 方案B：学校管理员对"审核学校=本校"的课件享有 L1 兜底审核权
		allowed = s.isSeniorOfReviewSchool(ctx, cw, reviewerID)
	}
	if !allowed {
		ok, permErr := s.isReviewerInAuthorGroupAsLeadOrBackbone(ctx, cw.UserID, reviewerID)
		if permErr != nil {
			return fmt.Errorf("校验审核权限失败: %w", permErr)
		}
		allowed = ok
	}
	if !allowed {
		return ErrCWReviewNoPermission
	}

	if req.Decision != models.ReviewDecisionApproved && req.Decision != models.ReviewDecisionRevision {
		return ErrCWReviewInvalidDecision
	}

	// 轮次 = 该课件 L1 已有记录数 + 1
	existing, _ := repository.CountCoursewareReviewsByLevel(ctx, coursewareID, models.ReviewLevelL1)
	round := existing + 1

	review := &models.CoursewareReview{
		CoursewareID: coursewareID,
		ReviewLevel:  models.ReviewLevelL1,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}
	if err := repository.CreateCoursewareReview(ctx, review); err != nil {
		cwReviewLog.Error("创建L1审核记录失败", "courseware_id", coursewareID, "error", err)
		return err
	}

	switch req.Decision {
	case models.ReviewDecisionApproved:
		schoolID := s.resolveReviewSchoolID(ctx, cw)
		var schoolIDPtr *string
		if schoolID != "" {
			schoolIDPtr = &schoolID
		}

		needL2 := false
		if schoolID != "" {
			cfg, cfgErr := repository.GetReviewFlowConfig(ctx, schoolID)
			if cfgErr == nil && cfg.L2Enabled {
				needL2 = true
			}
		}

		if needL2 {
			// 进入 L2：publish_state 保持 submitted，review_level→1
			_ = repository.UpdateCoursewarePublishState(ctx, coursewareID, models.CWPublishSubmitted, models.ReviewLevelL1, schoolIDPtr)
			cwReviewLog.Info("课件L1通过，进入L2待审核",
				"courseware_id", coursewareID, "school_id", schoolID, "round", round)
			// 阶段5c：进 L2 是内部流转，本版不单独推送 L2 审核员（senior 主动看列表）
		} else {
			// 无 L2：直接终审通过 → approved（待发布），review_level→1
			_ = repository.UpdateCoursewarePublishState(ctx, coursewareID, models.CWPublishApproved, models.ReviewLevelL1, schoolIDPtr)
			cwReviewLog.Info("课件L1通过，直接终审（待发布）",
				"courseware_id", coursewareID, "round", round)
			// 阶段5c：终审通过 → 通知作者
			s.notifyAuthorReviewResult(ctx, cw, reviewerID, models.ReviewDecisionApproved, "")
		}

	case models.ReviewDecisionRevision:
		// 退回：publish_state→revision，review_level→0，清空审核学校；批注不动（作者改时仍需参考）
		_ = repository.UpdateCoursewarePublishState(ctx, coursewareID, models.CWPublishRevision, 0, nil)
		cwReviewLog.Info("课件L1退回", "courseware_id", coursewareID, "round", round)
		// 阶段5c：退回 → 通知作者（审核意见进 Body）
		s.notifyAuthorReviewResult(ctx, cw, reviewerID, models.ReviewDecisionRevision, req.Comment)
	}

	return nil
}

// ==================== L2 学校审核 ====================

// ReviewL2 L2 学校审核决策。
// 权限：admin 或 senior_operator；senior_operator 须是 review_school_id 对应学校的管理员。
func (s *CoursewareReviewService) ReviewL2(ctx context.Context, coursewareID string, reviewerID string, reviewerRole string, req *models.CWReviewDecisionRequest) error {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return ErrCWReviewCoursewareNotFound
	}

	if cw.PublishState != models.CWPublishSubmitted || cw.ReviewLevel != models.ReviewLevelL1 {
		return ErrCWReviewNotL2Status
	}

	if reviewerRole != models.RoleSeniorOperator && reviewerRole != models.RoleAdmin {
		return ErrCWReviewNoPermission
	}
	if reviewerRole == models.RoleSeniorOperator {
		school, sErr := repository.GetSchoolByAdminUserID(ctx, reviewerID)
		if sErr != nil {
			return ErrCWReviewNoPermission
		}
		if cw.ReviewSchoolID == nil || *cw.ReviewSchoolID != school.ID {
			return ErrCWReviewNoPermission
		}
	}

	if req.Decision != models.ReviewDecisionApproved && req.Decision != models.ReviewDecisionRevision {
		return ErrCWReviewInvalidDecision
	}

	existing, _ := repository.CountCoursewareReviewsByLevel(ctx, coursewareID, models.ReviewLevelL2)
	round := existing + 1

	review := &models.CoursewareReview{
		CoursewareID: coursewareID,
		ReviewLevel:  models.ReviewLevelL2,
		ReviewerID:   reviewerID,
		Decision:     req.Decision,
		Score:        req.Score,
		Comment:      req.Comment,
		Dimensions:   req.Dimensions,
		ReviewRound:  round,
	}
	if err := repository.CreateCoursewareReview(ctx, review); err != nil {
		cwReviewLog.Error("创建L2审核记录失败", "courseware_id", coursewareID, "error", err)
		return err
	}

	switch req.Decision {
	case models.ReviewDecisionApproved:
		// L2 通过 → approved（待发布），review_level→2
		_ = repository.UpdateCoursewarePublishState(ctx, coursewareID, models.CWPublishApproved, models.ReviewLevelL2, cw.ReviewSchoolID)
		cwReviewLog.Info("课件L2通过（待发布）", "courseware_id", coursewareID, "round", round)
		// 阶段5c：终审通过 → 通知作者
		s.notifyAuthorReviewResult(ctx, cw, reviewerID, models.ReviewDecisionApproved, "")

	case models.ReviewDecisionRevision:
		_ = repository.UpdateCoursewarePublishState(ctx, coursewareID, models.CWPublishRevision, 0, nil)
		cwReviewLog.Info("课件L2退回", "courseware_id", coursewareID, "round", round)
		// 阶段5c：退回 → 通知作者（审核意见进 Body）
		s.notifyAuthorReviewResult(ctx, cw, reviewerID, models.ReviewDecisionRevision, req.Comment)
	}

	return nil
}

// ==================== 审核历史 ====================

// GetReviewHistory 获取某课件的审核历史
func (s *CoursewareReviewService) GetReviewHistory(ctx context.Context, coursewareID string) (*models.CWReviewHistoryResponse, error) {
	reviews, err := repository.ListCoursewareReviewsByCourseware(ctx, coursewareID)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []*models.CWReviewListItem{}
	}

	currentLevel := 0
	if cw, cErr := repository.GetCoursewareByID(ctx, coursewareID); cErr == nil {
		currentLevel = cw.ReviewLevel
	}

	return &models.CWReviewHistoryResponse{
		Reviews:      reviews,
		Total:        len(reviews),
		CurrentLevel: currentLevel,
	}, nil
}

// ==================== 待审核列表 ====================

// GetPendingReviews 获取当前用户的课件待审核列表（按角色分流，口径与教案一致）
//
// B3：新增 region_admin 分支——辖区只读视图，L1+L2 待审均按 review_school_id ∈ 辖区收窄。
// 方案B：senior_operator 的 L1 待审升级为"本校全量口径"（详见分支内注释）。
func (s *CoursewareReviewService) GetPendingReviews(ctx context.Context, userID string, userRole string, limit int, offset int) (*models.CWPendingReviewListResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	switch userRole {
	case models.RoleOperator, models.RoleViewer:
		items, total, err := repository.ListCWPendingReviewsL1(ctx, userID, limit, offset)
		if err != nil {
			return nil, err
		}
		return &models.CWPendingReviewListResponse{Items: items, Total: total}, nil

	case models.RoleSeniorOperator:
		// 方案B（2026-07-03）：学校管理员的 L1 待审从"组内口径"升级为"本校全量口径"——
		// 复用 B3 的 ListCWPendingReviewsBySchools（按 review_school_id 白名单过滤，
		// SubmitForReview 提交时即写入该列，L1/L2 待审阶段都有值），传本校单元素白名单。
		// 效果：不兼任组长/骨干的校管也能看到本校全部 L1 待审课件（并可兜底代审，见 ReviewL1）。
		schoolID := ""
		if school, err := repository.GetSchoolByAdminUserID(ctx, userID); err == nil && school != nil {
			schoolID = school.ID
		}
		if schoolID != "" {
			l1Items, _, _ := repository.ListCWPendingReviewsBySchools(ctx, []string{schoolID}, models.ReviewLevelL1, 100, 0)
			l2Items, _, _ := repository.ListCWPendingReviewsBySchools(ctx, []string{schoolID}, models.ReviewLevelL2, 100, 0)
			all := append(l1Items, l2Items...)
			return &models.CWPendingReviewListResponse{Items: all, Total: len(all)}, nil
		}
		// 学校解析失败（senior 未绑校等异常）：降级回老口径——仅组内 L1，无 L2（行为与改造前一致）
		l1Items, _, _ := repository.ListCWPendingReviewsL1(ctx, userID, 100, 0)
		if l1Items == nil {
			l1Items = []*models.CWPendingReviewItem{}
		}
		return &models.CWPendingReviewListResponse{Items: l1Items, Total: len(l1Items)}, nil

	case models.RoleRegionAdmin:
		// B3：区域管理员辖区视图（只读）——经唯一数据范围解析器取"辖区学校 ∪ 本校"白名单
		// （B2 后 ResolveDataScope 的 region 分支已是双来源+双重身份并集）。
		// 只读语义：本分支只提供可见性；审核决策权不变（ReviewL1/L2 的权限校验原封不动）。
		scope := ResolveDataScope(ctx, userRole, userID)
		if scope.Blocked || len(scope.SchoolIDs) == 0 {
			return &models.CWPendingReviewListResponse{Items: []*models.CWPendingReviewItem{}, Total: 0}, nil
		}
		l1Items, _, _ := repository.ListCWPendingReviewsBySchools(ctx, scope.SchoolIDs, models.ReviewLevelL1, 100, 0)
		l2Items, _, _ := repository.ListCWPendingReviewsBySchools(ctx, scope.SchoolIDs, models.ReviewLevelL2, 100, 0)
		all := append(l1Items, l2Items...)
		return &models.CWPendingReviewListResponse{Items: all, Total: len(all)}, nil

	case models.RoleAdmin:
		l1Items, _, _ := repository.ListCWPendingReviewsL1All(ctx, 100, 0)
		l2Items, _, _ := repository.ListCWPendingReviewsL2(ctx, "", 100, 0)
		all := append(l1Items, l2Items...)
		return &models.CWPendingReviewListResponse{Items: all, Total: len(all)}, nil

	default:
		return &models.CWPendingReviewListResponse{Items: []*models.CWPendingReviewItem{}, Total: 0}, nil
	}
}

// ==================== 审核统计 ====================

// GetReviewStats 获取课件审核统计（B3 修复版：按角色装配白名单，口径与待审列表严格一致）
//
// 各角色统计口径（与 GetPendingReviews 各分支一一对应）：
//   - admin           → 全局（memberIDs/schoolIDs 均不传）
//   - senior_operator → 方案B：L1/L2 统一按本校白名单（与其待审列表新口径一致）；
//                       学校解析失败且 level=L1 时降级为 lead/backbone 组员口径（与列表降级路径一致）
//   - region_admin    → L1/L2 均按辖区学校（ResolveDataScope，B2 双来源+双重身份并集）
//   - operator/viewer → L1 按其 lead/backbone 教研组成员；L2 无（计 0）
//   - 白名单装配失败/落空 → repo 层 fail-closed 计 0，绝不退化为全局
func (s *CoursewareReviewService) GetReviewStats(ctx context.Context, reviewerID string, userRole string, level int) (*models.CWReviewStatsResponse, error) {
	isAdmin := userRole == models.RoleAdmin

	var memberIDs []string
	var schoolIDs []string
	if !isAdmin {
		switch userRole {
		case models.RoleRegionAdmin:
			// 辖区学校白名单（L1/L2 通用）
			scope := ResolveDataScope(ctx, userRole, reviewerID)
			if !scope.Blocked {
				schoolIDs = scope.SchoolIDs
			}
		case models.RoleSeniorOperator:
			// 方案B：L1、L2 统一按本校白名单计数（与其待审列表新口径严格一致）
			if school, err := repository.GetSchoolByAdminUserID(ctx, reviewerID); err == nil && school != nil {
				schoolIDs = []string{school.ID}
			} else if level == models.ReviewLevelL1 {
				// 学校解析失败降级：回老口径（lead/backbone 组员），与列表降级路径一致
				memberIDs, _ = repository.GetCWReviewableMemberIDs(ctx, reviewerID)
			}
		default:
			// operator/viewer：仅 L1 有口径（本组成员）；L2 计 0
			if level == models.ReviewLevelL1 {
				memberIDs, _ = repository.GetCWReviewableMemberIDs(ctx, reviewerID)
			}
		}
	}

	return repository.GetCWReviewStats(ctx, reviewerID, level, isAdmin, memberIDs, schoolIDs)
}

// ==================== 已审核记录 ====================

// GetReviewedRecords 获取课件已审核记录列表
func (s *CoursewareReviewService) GetReviewedRecords(ctx context.Context, reviewerID string, userRole string, level int, decision string, limit int, offset int) (*models.CWReviewedListResponse, error) {
	isAdmin := userRole == models.RoleAdmin
	items, total, err := repository.ListCWReviewedRecords(ctx, reviewerID, level, decision, isAdmin, limit, offset)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.CWReviewedListItem{}
	}
	return &models.CWReviewedListResponse{Items: items, Total: total}, nil
}

// ==================== 审核详情（决策二：联动批注）====================

// GetReviewDetail 获取课件审核详情（供审核台：课件详情含页面 + 全部批注 + 审核历史）。
//
// 权限：admin 放行；L1 审核员（作者教研组 lead/backbone）放行；
//       本校 senior_operator 放行（方案B：L1/L2 阶段均可，见 canReviewCourseware）；
//       region_admin 对辖区课件只读放行（B3）。普通无关用户拒绝。
// 复用 cwService 装配课件详情、复用阶段2批注列表 —— 通过传入的服务引用调用，
// 避免本服务直接依赖 courseware_service 的内部装配逻辑。
//
// 参数 cwService 提供课件详情装配；批注直接走 repository（只读全集，无需可见性二次裁决，
// 因为能进审核详情的人已通过审核权限校验）。
func (s *CoursewareReviewService) GetReviewDetail(ctx context.Context, coursewareID string, userID string, userRole string, cwService *CoursewareService) (*models.CWReviewDetailResponse, error) {
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return nil, ErrCWReviewCoursewareNotFound
	}

	// 权限裁决（canReviewCourseware 在 courseware_review_access.go）
	if !s.canReviewCourseware(ctx, cw, userID, userRole) {
		return nil, ErrCWReviewNoPermission
	}

	// 课件详情（含 pages），复用 cwService.GetCourseware
	detail, err := cwService.GetCourseware(ctx, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("装配课件详情失败: %w", err)
	}

	// 全部批注（阶段2复用）
	annotations, aErr := repository.ListCWAnnotationsByCoursewareID(ctx, coursewareID)
	if aErr != nil {
		cwReviewLog.Warn("审核详情拉取批注失败（不阻断）", "courseware_id", coursewareID, "error", aErr)
		annotations = []*models.CoursewareAnnotation{}
	}
	if annotations == nil {
		annotations = []*models.CoursewareAnnotation{}
	}

	// 审核历史
	reviews, rErr := repository.ListCoursewareReviewsByCourseware(ctx, coursewareID)
	if rErr != nil {
		reviews = []*models.CWReviewListItem{}
	}
	if reviews == nil {
		reviews = []*models.CWReviewListItem{}
	}

	return &models.CWReviewDetailResponse{
		Courseware:  detail,
		Annotations: annotations,
		Reviews:     reviews,
	}, nil
}

// ==================== 内部辅助 ====================
// 权限裁决类辅助方法（canReviewCourseware / isSeniorOfReviewSchool /
// isReviewerInAuthorGroupAsLeadOrBackbone / resolveReviewSchoolID）
// 已拆至同包 courseware_review_access.go，本文件不再重复定义。
