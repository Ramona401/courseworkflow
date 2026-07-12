package services

// courseware_review_access.go — 课件多级审核·访问与权限裁决辅助（从 courseware_review_service.go 拆出）
//
// 拆分原因：主服务文件因"方案B（学校管理员本校 L1 可见+可审）"改造逼近 600 行红线，
// 将纯权限裁决类辅助方法集中到本文件，主文件只保留业务流程编排。
// 所有方法均挂 *CoursewareReviewService 接收器，与主文件同包同实例，调用方式零变化。
//
// ★ 方案B（2026-07-03，学校管理员审核盲区修复）★
//   背景：此前学校管理员（senior_operator）若不兼任教研组 lead/backbone，则完全看不到
//   本校 L1 待审课件（课件提交后先落 L1，未过 L1 不会出现在 L2 列表），形成管理盲区；
//   若某教研组恰好没有组长/骨干，课件会卡死在 L1 无人能审（仅 admin 可救）。
//   现开放：本校 senior_operator 对"审核学校=本校"的课件享有 L1 兜底审核权——
//     - canReviewCourseware（本文件）：本校 senior 可打开本校待审课件（L1/L2 阶段均可）的审核详情；
//     - ReviewL1（主文件）：权限阶梯加入"本校 senior"分支（经本文件 isSeniorOfReviewSchool 判定）。
//   职责分离说明：组长/骨干在岗时仍应由其完成 L1；senior 的 L1 权限定位为"兜底代审"。
//   开启两级审核（l2_enabled）的学校，senior 若代审 L1，L2 仍由其复审——L2 本就归 senior，
//   相当于提前介入，管理上可接受（在意分离的学校靠行为约定：组长在岗则由组长审 L1）。

import (
	"context"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 审核详情访问裁决 ====================

// canReviewCourseware 裁决"当前用户能否查看/审核该课件的审核详情"（审核详情访问控制）。
//   - admin → true
//   - region_admin → 课件 review_school_id ∈ 辖区学校（B3：辖区只读；不授予 L1/L2 决策权，
//     决策权限由 ReviewL1/ReviewL2 各自的校验把关，本函数放行仅意味着"能看详情"）
//   - senior_operator → 课件审核学校 = 本校（方案B：本校 L1/L2 阶段待审详情均可打开；
//     不匹配时不直接拒绝——senior 也可能兼任某组 lead/backbone，继续落到下方组内判定）
//   - 课件处于 L1 待审核（review_level=0）：作者教研组 lead/backbone → true
//   - 其它一律 false
//
// 说明：原"L2 阶段（review_level=1）本校 senior 单独放行"分支已被上方 senior 统一分支覆盖
// （isSeniorOfReviewSchool 经 resolveReviewSchoolID 优先取 review_school_id，同一比对口径且
// 多一层作者学校回退，语义只宽不严），故不再重复保留。
func (s *CoursewareReviewService) canReviewCourseware(ctx context.Context, cw *models.Courseware, userID string, userRole string) bool {
	if userRole == models.RoleAdmin {
		return true
	}
	// B3：区域管理员——辖区课件只读详情
	// 待审课件（submitted）的 review_school_id 在提交时即写入，L1/L2 阶段都有值；
	// 该列为空（如已退回 revision）则不放行，fail-closed。
	if userRole == models.RoleRegionAdmin {
		if cw.ReviewSchoolID == nil || *cw.ReviewSchoolID == "" {
			return false
		}
		scope := ResolveDataScope(ctx, userRole, userID)
		if scope.Blocked {
			return false
		}
		for _, sid := range scope.SchoolIDs {
			if sid == *cw.ReviewSchoolID {
				return true
			}
		}
		return false
	}
	// 方案B：学校管理员——本校待审课件（L1/L2 阶段）详情放行
	if userRole == models.RoleSeniorOperator {
		if s.isSeniorOfReviewSchool(ctx, cw, userID) {
			return true
		}
		// 不匹配不直接拒绝：senior 可能兼任某组 lead/backbone，继续走下方组内判定
	}
	// L1 阶段：作者教研组 lead/backbone
	if cw.ReviewLevel == 0 {
		ok, _ := s.isReviewerInAuthorGroupAsLeadOrBackbone(ctx, cw.UserID, userID)
		return ok
	}
	return false
}

// ==================== 学校管理员本校判定（方案B新增） ====================

// isSeniorOfReviewSchool 判断 userID（调用方应保证其角色为 senior_operator）管理的学校
// 是否等于该课件的审核学校。
//
// 审核学校经 resolveReviewSchoolID 解析：优先课件已写入的 review_school_id，
// 该列为空时回退反查作者所属学校（与 L1 通过时"是否进 L2"的判定走同一条解析链，口径一致）。
// 学校反查失败 / 解析为空 / 不匹配均返回 false（fail-closed，绝不放大权限）。
func (s *CoursewareReviewService) isSeniorOfReviewSchool(ctx context.Context, cw *models.Courseware, userID string) bool {
	school, err := repository.GetSchoolByAdminUserID(ctx, userID)
	if err != nil || school == nil {
		return false
	}
	reviewSchoolID := s.resolveReviewSchoolID(ctx, cw)
	return reviewSchoolID != "" && reviewSchoolID == school.ID
}

// ==================== 组内 lead/backbone 判定 ====================

// isReviewerInAuthorGroupAsLeadOrBackbone 判断 reviewer 是否在"作者所属某教研组"中担任 lead/backbone。
// 即：存在某教研组 g，使得 作者 是 g 的成员，且 reviewer 在 g 中 role ∈ (lead, backbone)。
func (s *CoursewareReviewService) isReviewerInAuthorGroupAsLeadOrBackbone(ctx context.Context, authorID string, reviewerID string) (bool, error) {
	// 取作者所属全部教研组
	authorGroups, err := repository.GetUserTeachingGroups(ctx, authorID)
	if err != nil {
		return false, err
	}
	if len(authorGroups) == 0 {
		return false, nil
	}
	// 逐组判断 reviewer 是否在该组任 lead/backbone
	for _, g := range authorGroups {
		ok, lbErr := repository.IsGroupLeadOrBackbone(ctx, g.ID, reviewerID)
		if lbErr != nil {
			continue
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// ==================== 审核学校解析 ====================

// resolveReviewSchoolID 解析课件审核学校ID：优先已写入的 review_school_id，否则反查作者学校。
func (s *CoursewareReviewService) resolveReviewSchoolID(ctx context.Context, cw *models.Courseware) string {
	if cw.ReviewSchoolID != nil && *cw.ReviewSchoolID != "" {
		return *cw.ReviewSchoolID
	}
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, cw.UserID)
	return schoolID
}
