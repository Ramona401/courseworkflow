package repository

// lesson_plan_creation_domain_repo.go — 普通教案创建教育域严格解析
//
// 本文件专门服务普通教案创建，不复用登录场景的兼容解析结果。
//
// 登录与创建的安全目标不同：
//   - 登录需要兼容存量普通账号，部分异常状态可能使用兼容展示值；
//   - 创建教学资源必须fail-closed，任何不确定都不能创建教案。
//
// 普通教学身份只读取以下确定性关系：
//   1. organizations.admin_user_id学校主管理员字段；
//   2. organization_admins中的school_admin正式任命；
//   3. school_members学校直接校籍；
//   4. teaching_group_members所在启用教研组对应的启用学校。
//
// region_admin使用既有严格固定域解析器：
//   - 固定教育域来自organization_admins.education_domain；
//   - 多个同域区域任命允许；
//   - 无任命、空值、非法值和跨域冲突全部拒绝。
//
// 返回值只可能是k12、vocational或adult。
// mixed、common、空值、非法字符串、无教学组织和跨域冲突全部拒绝。
// 数据库查询、扫描和遍历错误向上返回，绝不回退K12。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrLessonPlanCreationEducationDomainUnavailable 表示没有可用于创建教案的具体教学域。
	//
	// 典型原因：
	//   - 没有有效教学组织；
	//   - 教育域为空、mixed、common或非法；
	//   - mixed管理身份尝试直接创建教学教案；
	//   - 用户ID或角色无效。
	ErrLessonPlanCreationEducationDomainUnavailable = errors.New(
		"教案创建教育域不可用",
	)

	// ErrLessonPlanCreationEducationDomainConflict 表示用户同时关联多个具体教学域。
	ErrLessonPlanCreationEducationDomainConflict = errors.New(
		"教案创建教育域冲突",
	)
)

// LessonPlanCreationEducationDomainCandidate 是一条教学组织教育域候选。
//
// 该类型导出是为了让Service层测试可以覆盖纯解析规则，
// 正式数据库调用仍由ResolveLessonPlanCreationEducationDomain统一完成。
type LessonPlanCreationEducationDomainCandidate struct {
	OrganizationID  string
	EducationDomain string
}

// ResolveLessonPlanCreationEducationDomainFromCandidates
// 将教学组织候选解析为唯一具体教学域。
//
// 本函数不访问数据库，不调用NormalizeEducationDomain。
// 同域多组织允许；任一非法候选或多个不同具体域均直接拒绝。
func ResolveLessonPlanCreationEducationDomainFromCandidates(
	candidates []LessonPlanCreationEducationDomainCandidate,
) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf(
			"%w: 没有有效教学组织",
			ErrLessonPlanCreationEducationDomainUnavailable,
		)
	}

	domainSet := make(map[string]struct{})

	for _, candidate := range candidates {
		domain := strings.ToLower(
			strings.TrimSpace(candidate.EducationDomain),
		)

		// 教案属于具体教学资源，只接受三个具体教学域。
		// 禁止调用NormalizeEducationDomain，避免非法值静默回退K12。
		if !models.IsTeachingEducationDomain(domain) {
			return "", fmt.Errorf(
				"%w: 组织%s的教育域为空或非法",
				ErrLessonPlanCreationEducationDomainUnavailable,
				candidate.OrganizationID,
			)
		}

		domainSet[domain] = struct{}{}
	}

	if len(domainSet) != 1 {
		return "", fmt.Errorf(
			"%w: 同时存在多个具体教学教育域",
			ErrLessonPlanCreationEducationDomainConflict,
		)
	}

	for domain := range domainSet {
		return domain, nil
	}

	// 防御性出口，理论上不会到达。
	return "", ErrLessonPlanCreationEducationDomainUnavailable
}

// ResolveLessonPlanCreationEducationDomain
// 解析用户创建普通教案时的具体教学教育域。
//
// role必须由Service从users表实时读取，不能接受前端角色参数。
//
// 角色规则：
//   - region_admin：按固定教育域任命严格解析；
//   - senior_operator/operator/viewer：按学校任命、校籍和教研组解析；
//   - admin/district_inspector：属于mixed管理身份，不能直接创建教学教案；
//   - 未知角色：拒绝。
func ResolveLessonPlanCreationEducationDomain(
	ctx context.Context,
	userID string,
	role string,
) (string, error) {
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)

	if userID == "" {
		return "", fmt.Errorf(
			"%w: 用户ID为空",
			ErrLessonPlanCreationEducationDomainUnavailable,
		)
	}

	switch role {
	case models.RoleRegionAdmin:
		educationContext, err :=
			resolveRegionAdminEducationContext(
				ctx,
				userID,
			)
		if err != nil {
			return "", err
		}
		if educationContext == nil {
			return "", fmt.Errorf(
				"%w: 区域管理员教育上下文为空",
				ErrLessonPlanCreationEducationDomainUnavailable,
			)
		}

		domain := strings.ToLower(
			strings.TrimSpace(
				educationContext.EducationDomain,
			),
		)
		if !models.IsTeachingEducationDomain(domain) {
			return "", fmt.Errorf(
				"%w: 区域管理员固定教育域非法",
				ErrLessonPlanCreationEducationDomainUnavailable,
			)
		}

		return domain, nil

	case models.RoleSeniorOperator,
		models.RoleOperator,
		models.RoleViewer:
		// 继续查询普通教学组织候选。

	case models.RoleAdmin,
		models.RoleDistrictInspector:
		return "", fmt.Errorf(
			"%w: mixed管理身份不能直接创建教学教案",
			ErrLessonPlanCreationEducationDomainUnavailable,
		)

	default:
		return "", fmt.Errorf(
			"%w: 未知或不支持的用户角色%s",
			ErrLessonPlanCreationEducationDomainUnavailable,
			role,
		)
	}

	rows, err := database.DB.Query(ctx, `
		WITH candidate_links AS (
			-- 1A. 学校主管理员字段。
			SELECT
				o.id::text AS organization_id,
				COALESCE(o.education_domain, '')
					AS education_domain,
				1 AS source_priority,
				COALESCE(o.created_at, now())
					AS linked_at
			FROM organizations o
			WHERE o.admin_user_id = $1
			  AND o.type = 'school'
			  AND o.status = 'active'

			UNION ALL

			-- 1B. 正式学校管理员任命。
			SELECT
				o.id::text,
				COALESCE(o.education_domain, ''),
				1,
				COALESCE(
					oa.created_at,
					o.created_at,
					now()
				)
			FROM organization_admins oa
			JOIN organizations o
			  ON o.id = oa.org_id
			 AND o.type = 'school'
			 AND o.status = 'active'
			WHERE oa.user_id = $1
			  AND oa.role_type = 'school_admin'

			UNION ALL

			-- 2. 学校直接校籍。
			SELECT
				o.id::text,
				COALESCE(o.education_domain, ''),
				2,
				COALESCE(
					sm.joined_at,
					o.created_at,
					now()
				)
			FROM school_members sm
			JOIN organizations o
			  ON o.id = sm.school_id
			 AND o.type = 'school'
			 AND o.status = 'active'
			WHERE sm.user_id = $1

			UNION ALL

			-- 3. 启用教研组成员归属兜底。
			SELECT
				o.id::text,
				COALESCE(o.education_domain, ''),
				3,
				COALESCE(
					tgm.joined_at,
					tg.created_at,
					o.created_at,
					now()
				)
			FROM teaching_group_members tgm
			JOIN teaching_groups tg
			  ON tg.id = tgm.group_id
			 AND tg.status = 'active'
			JOIN organizations o
			  ON o.id = tg.school_id
			 AND o.type = 'school'
			 AND o.status = 'active'
			WHERE tgm.user_id = $1
		)
		SELECT
			organization_id,
			education_domain
		FROM candidate_links
		ORDER BY
			source_priority,
			linked_at,
			organization_id
	`, userID)
	if err != nil {
		return "", fmt.Errorf(
			"查询教案创建教育域候选失败: %w",
			err,
		)
	}
	defer rows.Close()

	candidates := make(
		[]LessonPlanCreationEducationDomainCandidate,
		0,
	)
	seenOrganizations := make(map[string]struct{})

	for rows.Next() {
		var candidate LessonPlanCreationEducationDomainCandidate

		if err := rows.Scan(
			&candidate.OrganizationID,
			&candidate.EducationDomain,
		); err != nil {
			return "", fmt.Errorf(
				"扫描教案创建教育域候选失败: %w",
				err,
			)
		}

		// 同一学校可能被管理员任命、直接校籍和教研组同时命中。
		// 保留稳定排序后的首条，避免重复候选影响诊断。
		if _, exists :=
			seenOrganizations[candidate.OrganizationID]; exists {
			continue
		}

		seenOrganizations[candidate.OrganizationID] = struct{}{}
		candidates = append(candidates, candidate)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf(
			"遍历教案创建教育域候选失败: %w",
			err,
		)
	}

	return ResolveLessonPlanCreationEducationDomainFromCandidates(
		candidates,
	)
}
