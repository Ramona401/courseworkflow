package services

// courseware_share_service.go — 课件工坊·发布与共享 + 产权分级（阶段1）
//
// 与 status 生产状态机【正交】的"发布/审核维度"业务逻辑，不碰 courseware_service.go 原有逻辑。
// 本文件提供：
//   - SetPublishState       发布/撤回（按当前发布态 + 角色裁决允许的目标态）
//   - SetCodeShareScope     设置源代码开放范围（产权分级）
//   - ListSharedCoursewares 共享课件库（解析同校/同组作者白名单 + 逐条算 CanCopy）
//   - ForkCourseware        复制到我的（校验 code_share_scope 后深拷贝课件+页面+资产）
//   - resolveCanCopy        内部辅助：按 code_share_scope + 归属关系裁决能否复制源码
//
// 设计要点：
//   1) 可见范围（谁能看渲染效果）与代码复制权（谁能复制源码）双轨解耦：
//      可见范围走 publish_state=published_shared + 同校/同组；
//      代码复制权走 code_share_scope（作者/组织独立设定）。
//   2) 共享可见性不靠 DataScope.SchoolIDs（operator/viewer 那里是空集），
//      而是按"看课件的人是否与作者同校/同组"过滤——与教案库范式一致。
//   3) 归属校验延续 cw.UserID != userID 模式。

import (
	"fmt"

	"context"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// shareServiceLog 模块日志（复用 courseware_service.go 中的 cwServiceLog，同包）
var shareServiceLog = cwServiceLog

// ==================== 发布 / 撤回 ====================

// SetPublishState 设置课件发布态（发布 / 撤回）。
//
// 允许的流转（阶段1，不含审核流——审核在阶段3接入）：
//   - private            → published_personal（个人发布：标记完成，暂不共享）
//   - private            → published_shared  （作者直接共享：同校/同组可见）
//   - published_personal → published_shared  （从个人发布升级为共享）
//   - published_personal → private           （撤回到私有）
//   - published_shared   → private           （撤回共享）
//   - published_shared   → published_personal（收回共享但保留个人发布标记）
//
// submitted/approved/revision 属于审核流中间态：submitted 拒绝（需先撤回审核）；
// approved/revision 允许作者发布为共享或撤回到私有。
//
// 参数 target 必须是 published_personal / published_shared / private 之一。
func (s *CoursewareService) SetPublishState(ctx context.Context, id string, userID string, target string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}

	// 目标态白名单（阶段1 仅允许这三个；审核相关态由阶段3接口管理）
	switch target {
	case models.CWPublishPrivate, models.CWPublishPublishedPersonal, models.CWPublishPublishedShared:
		// ok
	default:
		return fmt.Errorf("不支持的发布目标态: %s", target)
	}

	// 审核中（submitted）禁止用发布接口直接改，避免绕过审核流程
	if cw.PublishState == models.CWPublishSubmitted {
		return fmt.Errorf("课件正在审核中，请先撤回审核再操作")
	}

	// 共享发布要求课件已生成到可展示状态（至少 preview），避免共享半成品
	if target == models.CWPublishPublishedShared {
		order := models.CoursewareStatusOrder[cw.Status]
		if order < models.CoursewareStatusOrder[models.CoursewareStatusPreview] {
			return fmt.Errorf("课件尚未生成完成，暂不能共享（需至少进入预览阶段）")
		}
	}

	// 发布/撤回不涉及审核层级与审核学校，统一清零 review_level、清空 review_school_id
	if err := repository.UpdateCoursewarePublishState(ctx, id, target, 0, nil); err != nil {
		return err
	}
	shareServiceLog.Info("课件发布态变更",
		"courseware_id", id, "from", cw.PublishState, "to", target, "user_id", userID)
	return nil
}

// ==================== 设置代码开放范围（产权分级）====================

// SetCodeShareScope 设置课件源代码开放范围（产权保护，独立于可见范围）。
// 仅作者可设。scope 必须是 none/group/school/region/public 之一。
func (s *CoursewareService) SetCodeShareScope(ctx context.Context, id string, userID string, scope string) error {
	cw, err := repository.GetCoursewareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return fmt.Errorf("无权操作此课件")
	}
	if !models.IsValidCodeShareScope(scope) {
		return fmt.Errorf("无效的代码开放范围: %s", scope)
	}
	if err := repository.UpdateCoursewareCodeShareScope(ctx, id, scope); err != nil {
		return err
	}
	shareServiceLog.Info("课件代码开放范围变更",
		"courseware_id", id, "from", cw.CodeShareScope, "to", scope, "user_id", userID)
	return nil
}

// ==================== 共享课件库 ====================

// ListSharedCoursewares 查询共享课件库（他人共享给"我"的课件）。
//
// 可见性解析（与教案库范式一致，不靠 DataScope.SchoolIDs）：
//   - admin：看全部已共享课件（visibleAuthorIDs 传 nil）。
//   - 其他角色：解析"与当前用户同校 或 同教研组的所有用户"作为作者白名单，
//     只看这些人共享的课件。解析失败/未绑校组 → 至少含本人（看得到自己共享的）。
//
// 逐条计算 CanCopy（当前登录者能否复制该课件源码），供前端显隐"复制到我的"按钮。
func (s *CoursewareService) ListSharedCoursewares(ctx context.Context, userID string, role string, subject string, limit int, offset int) (*models.SharedCoursewareListResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	// 1) 解析可见作者白名单
	var visibleAuthorIDs []string
	if role == models.RoleAdmin {
		visibleAuthorIDs = nil // admin 不限作者
	} else {
		visibleAuthorIDs = s.resolveSameOrgUserIDs(ctx, userID)
	}

	// 2) 查询
	items, total, err := repository.ListSharedCoursewares(ctx, visibleAuthorIDs, subject, limit, offset)
	if err != nil {
		return nil, err
	}

	// 3) 逐条裁决 CanCopy。先把当前用户的学校ID/教研组ID集合算一次，避免每条都查库。
	viewerSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	viewerGroupIDs := s.resolveUserGroupIDSet(ctx, userID)

	for _, it := range items {
		it.CanCopy = s.resolveCanCopy(ctx, role, it.AuthorID, it.CodeShareScope, viewerSchoolID, viewerGroupIDs)
	}

	return &models.SharedCoursewareListResponse{
		Coursewares: items,
		Total:       total,
	}, nil
}

// ==================== Fork：复制到我的 ====================

// ForkCourseware 把一个共享课件复制为当前用户名下的新课件（深拷贝课件+页面+资产）。
//
// 前置校验：
//   - 源课件存在且处于已共享态（published_shared）——只有共享出来的才允许被复制。
//   - 当前用户对源课件的 code_share_scope 有复制权（resolveCanCopy=true）。
//   - 不能 Fork 自己的课件（自己的直接用，无需复制）。
//
// 复制内容：课件主记录（重置为当前用户、私有、流程态沿用源 status）、全部页面、全部资产。
// 新课件 publish_state=private、code_share_scope=none、review_level=0（产权归复制者，默认不外发）。
func (s *CoursewareService) ForkCourseware(ctx context.Context, srcID string, userID string, role string) (*models.Courseware, error) {
	src, err := repository.GetCoursewareByID(ctx, srcID)
	if err != nil {
		return nil, fmt.Errorf("源课件不存在: %w", err)
	}
	if src.UserID == userID {
		return nil, fmt.Errorf("这是您自己的课件，无需复制")
	}
	if src.PublishState != models.CWPublishPublishedShared {
		return nil, fmt.Errorf("该课件未共享，不能复制")
	}

	// 代码复制权裁决
	viewerSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	viewerGroupIDs := s.resolveUserGroupIDSet(ctx, userID)
	if !s.resolveCanCopy(ctx, role, src.UserID, src.CodeShareScope, viewerSchoolID, viewerGroupIDs) {
		return nil, fmt.Errorf("该课件作者未开放源码复制权限")
	}

	// 1) 复制课件主记录（归当前用户，私有，代码不外发；流程态沿用源以便复制者继续编辑/预览）
	newCW := &models.Courseware{
		LessonPlanID: nil, // 复制品不挂源教案，避免误关联
		UserID:       userID,
		Title:        src.Title + "（副本）",
		Subject:      src.Subject,
		Grade:        src.Grade,
		Status:       src.Status,
		SourceType:   src.SourceType,
		PageCount:    src.PageCount,
	}
	if err := repository.CreateCourseware(ctx, newCW); err != nil {
		return nil, fmt.Errorf("创建副本课件失败: %w", err)
	}

	// 复制风格/Logo/机构名/导航栏/概述等展示要素（逐字段写，复用现成 Update 函数）
	if src.StyleConfig != "" {
		_ = repository.UpdateCoursewareStyle(ctx, newCW.ID, src.StyleConfig)
	}
	if src.LogoURL != "" {
		_ = repository.UpdateCoursewareLogo(ctx, newCW.ID, src.LogoURL)
	}
	if src.OrgName != "" {
		_ = repository.UpdateCoursewareOrgName(ctx, newCW.ID, src.OrgName)
	}
	if src.NavTemplateHTML != "" {
		_ = repository.UpdateCoursewareNavTemplate(ctx, newCW.ID, src.NavTemplateHTML)
	}
	if src.IndexOverview != "" {
		_ = repository.UpdateCoursewareOverview(ctx, newCW.ID, src.IndexOverview)
	}

	// 2) 复制全部页面。建立 旧pageID → 新pageID 映射，供资产挂载到新页。
	srcPages, err := repository.ListCoursewarePages(ctx, srcID)
	if err != nil {
		shareServiceLog.Warn("复制课件：读取源页面失败", "src", srcID, "error", err)
		return newCW, nil // 主记录已建，页面复制失败不阻断，返回已建课件
	}
	oldToNewPageID := make(map[string]string, len(srcPages))
	for _, p := range srcPages {
		np := &models.CoursewarePage{
			CoursewareID:        newCW.ID,
			PageNumber:          p.PageNumber,
			Title:               p.Title,
			Purpose:             p.Purpose,
			ContentSummary:      p.ContentSummary,
			InteractionType:     p.InteractionType,
			VisualFormat:        p.VisualFormat,
			MediaRequirements:   p.MediaRequirements,
			EstimatedComplexity: p.EstimatedComplexity,
			PageIndex:           p.PageIndex,
			IdxCognitiveLevel:   p.IdxCognitiveLevel,
			IdxInteractionLevel: p.IdxInteractionLevel,
			IdxVisualFormat:     p.IdxVisualFormat,
			HTMLContent:         p.HTMLContent,
			PlaceholderMap:      p.PlaceholderMap,
			MatchedComponentIDs: p.MatchedComponentIDs,
			Status:              p.Status,
		}
		if err := repository.CreateCoursewarePage(ctx, np); err != nil {
			shareServiceLog.Warn("复制课件：创建副本页面失败", "page_number", p.PageNumber, "error", err)
			continue
		}
		oldToNewPageID[p.ID] = np.ID
	}

	// 3) 复制全部资产（图片/视频），page_id 重映射到新页；课件级资产（page_id=nil）保持 nil。
	srcAssets, aErr := repository.ListCWAssetsByCourseware(ctx, srcID)
	if aErr == nil {
		for _, a := range srcAssets {
			var newPageID *string
			if a.PageID != nil {
				if mapped, ok := oldToNewPageID[*a.PageID]; ok {
					newPageID = &mapped
				}
			}
			na := &models.CoursewareAsset{
				CoursewareID:     newCW.ID,
				PageID:           newPageID,
				PlaceholderID:    a.PlaceholderID,
				AssetType:        a.AssetType,
				GenerationPrompt: a.GenerationPrompt,
				OssURL:           a.OssURL,
				PublicOSSURL:     a.PublicOSSURL,
				FileSize:         a.FileSize,
				MimeType:         a.MimeType,
				Metadata:         a.Metadata,
				Status:           a.Status,
			}
			if err := repository.CreateCWAsset(ctx, na); err != nil {
				shareServiceLog.Warn("复制课件：创建副本资产失败", "placeholder", a.PlaceholderID, "error", err)
			}
		}
	}

	shareServiceLog.Info("课件复制完成",
		"src_courseware_id", srcID, "new_courseware_id", newCW.ID,
		"pages", len(oldToNewPageID), "user_id", userID)
	return newCW, nil
}

// ==================== 内部辅助：可见性 / 复制权解析 ====================

// resolveSameOrgUserIDs 解析"与该用户同校 或 同教研组"的所有用户ID（含其本人），用于共享课件可见白名单。
// 返回非nil切片；解析失败或未绑校组时至少返回[自己]（保证能看到自己共享的）。
func (s *CoursewareService) resolveSameOrgUserIDs(ctx context.Context, userID string) []string {
	idSet := map[string]struct{}{userID: {}}

	// 同校：取本人学校的全体成员
	if schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID); schoolID != "" {
		if memberIDs, err := repository.ListSchoolMemberIDs(ctx, schoolID); err == nil {
			for _, uid := range memberIDs {
				idSet[uid] = struct{}{}
			}
		} else {
			shareServiceLog.Warn("解析同校成员失败", "school", schoolID, "error", err)
		}
	}

	// 同组：取本人所在各教研组的成员（教研组可能跨校协作，单独并入）
	groups, err := repository.GetUserTeachingGroups(ctx, userID)
	if err == nil {
		for _, g := range groups {
			members, mErr := repository.ListTeachingGroupMemberIDs(ctx, g.ID)
			if mErr != nil {
				continue
			}
			for _, uid := range members {
				idSet[uid] = struct{}{}
			}
		}
	}

	out := make([]string, 0, len(idSet))
	for uid := range idSet {
		out = append(out, uid)
	}
	return out
}

// resolveUserGroupIDSet 取用户所属教研组ID集合（供 CanCopy 的 group 级裁决用）。
func (s *CoursewareService) resolveUserGroupIDSet(ctx context.Context, userID string) map[string]struct{} {
	set := make(map[string]struct{})
	groups, err := repository.GetUserTeachingGroups(ctx, userID)
	if err != nil {
		return set
	}
	for _, g := range groups {
		set[g.ID] = struct{}{}
	}
	return set
}

// resolveCanCopy 裁决"当前登录者能否复制 author 这个作者、code_share_scope=scope 的课件源码"。
//
// 规则：
//   - none   ：任何人都不能复制（作者本人不会出现在共享库里，故无需特判）。
//   - public ：任何可见者都能复制。
//   - school ：当前用户与作者同校才能复制。
//   - group  ：当前用户与作者有共同教研组才能复制。
//   - region ：阶段1 用"同校即同区域"近似；跨校区域级在阶段6细化。
//   - admin  ：始终可复制。
func (s *CoursewareService) resolveCanCopy(ctx context.Context, role string, authorID string, scope string, viewerSchoolID string, viewerGroupIDs map[string]struct{}) bool {
	if role == models.RoleAdmin {
		return true
	}
	switch scope {
	case models.CWCodeShareNone:
		return false
	case models.CWCodeSharePublic:
		return true
	case models.CWCodeShareSchool, models.CWCodeShareRegion:
		// 同校判定（region 在阶段1 近似为同校；阶段6 接入区域树后再细分）
		if viewerSchoolID == "" {
			return false
		}
		authorSchoolID, _ := repository.GetSchoolIDByUserID(ctx, authorID)
		return authorSchoolID != "" && authorSchoolID == viewerSchoolID
	case models.CWCodeShareGroup:
		// 与作者有共同教研组
		if len(viewerGroupIDs) == 0 {
			return false
		}
		authorGroups, err := repository.GetUserTeachingGroups(ctx, authorID)
		if err != nil {
			return false
		}
		for _, g := range authorGroups {
			if _, ok := viewerGroupIDs[g.ID]; ok {
				return true
			}
		}
		return false
	}
	return false
}
