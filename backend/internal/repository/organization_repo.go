package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================
//
// 本 package 级错误常量同时被 organization_repo.go 与 organization_group_repo.go 引用，
// 故全部定义在此处不拆分（ErrGroupNotFound/ErrMemberExists/ErrMemberNotFound 供教研组文件使用）。

var (
	ErrOrgNotFound     = errors.New("组织不存在")
	ErrOrgNameExists   = errors.New("同类型下组织名称已存在")
	ErrGroupNotFound   = errors.New("教研组不存在")
	ErrGroupNameExists = errors.New("该学校下教研组名称已存在")
	ErrMemberExists    = errors.New("该用户已是教研组成员")
	ErrMemberNotFound  = errors.New("教研组成员不存在")
)

// ==================== 文件职责说明 ====================
//
// 本文件承载：组织(organizations)CRUD + 门户板块可见性 + 学校直接成员(school_members)。
// 教研组(teaching_groups)及其成员的数据访问已拆至 organization_group_repo.go（超 600 行红线拆分）。
// 组织多管理员(organization_admins)的数据访问在 organization_admin_repo.go。
// 学校成员入校的事务版在 organization_member_tx_repo.go。

// ==================== 组织 CRUD ====================

func CreateOrganization(ctx context.Context, org *models.Organization) error {
	query := `
		INSERT INTO organizations (name, type, parent_id, admin_user_id, settings, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	settings := org.Settings
	if settings == "" {
		settings = "{}"
	}
	err := database.DB.QueryRow(ctx, query,
		org.Name, org.Type, org.ParentID, org.AdminUserID, settings, "active",
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建组织失败: %w", err)
	}
	return nil
}

func GetOrganizationByID(ctx context.Context, id string) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
		SELECT id, name, type, parent_id, admin_user_id, settings, COALESCE(logo_url,''), status, created_at, updated_at
		FROM organizations WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Type, &org.ParentID, &org.AdminUserID,
		&org.Settings, &org.LogoURL, &org.Status, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("查询组织失败: %w", err)
	}
	return org, nil
}

// GetSchoolByAdminUserID 根据学校管理员用户ID获取其管理的学校
//
// 规则：仅返回 type='school' 的组织；若无则返回 ErrOrgNotFound
//
// ⚠ B1 修复（第二学校管理员"未关联学校"根治）：
//
//	本函数是"用户→所属学校"反查的权威入口，被 13+ 处调用（token scope / data_scope /
//	课件与教案多级审核 / 课程大纲 / 单元方案 / 模板发布 / admin_handler / AI助手 全链路）。
//	历史实现只查 organizations.admin_user_id 单字段——而该字段只存"首个/主"学校管理员，
//	一个学校任命的第二个 school_admin 只写进了 organization_admins 表、不进单字段，
//	于是第二校管在此反查落空，连锁表现为：登录后"未关联学校"、进课程大纲被提示"不是学校
//	管理员"、积分 scope 收窄空集、模板发布/课件 L2 审核失效等。
//
//	修法（两级 fail-open 查找链，语义与 token/data_scope 的 school 解析一致）：
//	  ① 先查 organizations.admin_user_id 单字段（旧口径，命中即返，行为与历史完全一致，
//	     对已工作的首个校管零影响）。
//	  ② 单字段查不到时，兜底查 organization_admins(role_type='school_admin')，
//	     JOIN organizations 取该用户被任命管理的 type='school' 组织。
//	     多个学校时按 created_at 升序取第一个（与"首个校管"直觉一致，且此场景极罕见——
//	     同一 school_admin 被任命管理多所学校非常规用法）。
//	  ③ 两级都无 → ErrOrgNotFound（语义不变，调用方错误处理链一字不动）。
//
//	为什么改这一处即修全链：所有依赖"用户是不是某校管理员/属于哪个校"的判断都汇到本函数，
//	补上多管理员表这条链，第二校管在全部下游链路上一次性复活，无需逐个调用点改动。
func GetSchoolByAdminUserID(ctx context.Context, adminUserID string) (*models.Organization, error) {
	if adminUserID == "" {
		return nil, ErrOrgNotFound
	}

	org := &models.Organization{}

	// ① 旧口径：organizations.admin_user_id 单字段（主/首个学校管理员）
	singleFieldQuery := `
		SELECT id, name, type, parent_id, admin_user_id, settings, COALESCE(logo_url,''), status, created_at, updated_at
		FROM organizations
		WHERE admin_user_id = $1 AND type = 'school'
		LIMIT 1
	`
	err := database.DB.QueryRow(ctx, singleFieldQuery, adminUserID).Scan(
		&org.ID, &org.Name, &org.Type, &org.ParentID, &org.AdminUserID,
		&org.Settings, &org.LogoURL, &org.Status, &org.CreatedAt, &org.UpdatedAt,
	)
	if err == nil {
		return org, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		// 真正的 DB 异常直接上抛（不是"查无此人"，不进兜底）
		return nil, fmt.Errorf("查询学校管理员所属学校失败: %w", err)
	}

	// ② 兜底：organization_admins 多管理员表（第二/后续学校管理员在此）
	//   role_type='school_admin' 且组织为 type='school'，JOIN 取组织全字段。
	//   多校时按 created_at 升序取首个（与"首个校管"直觉一致）。
	multiAdminQuery := `
		SELECT o.id, o.name, o.type, o.parent_id, o.admin_user_id, o.settings,
		       COALESCE(o.logo_url,''), o.status, o.created_at, o.updated_at
		FROM organization_admins oa
		JOIN organizations o ON o.id = oa.org_id
		WHERE oa.user_id = $1 AND oa.role_type = 'school_admin' AND o.type = 'school'
		ORDER BY oa.created_at ASC
		LIMIT 1
	`
	org2 := &models.Organization{}
	err2 := database.DB.QueryRow(ctx, multiAdminQuery, adminUserID).Scan(
		&org2.ID, &org2.Name, &org2.Type, &org2.ParentID, &org2.AdminUserID,
		&org2.Settings, &org2.LogoURL, &org2.Status, &org2.CreatedAt, &org2.UpdatedAt,
	)
	if err2 == nil {
		return org2, nil
	}
	if errors.Is(err2, pgx.ErrNoRows) {
		// 两级都查不到 → 语义不变，返回 ErrOrgNotFound
		return nil, ErrOrgNotFound
	}
	return nil, fmt.Errorf("查询学校管理员所属学校(多管理员兜底)失败: %w", err2)
}

func ListOrganizations(ctx context.Context, orgType string, parentID string) ([]*models.OrganizationListItem, error) {
	query := `
		SELECT o.id, o.name, o.type, o.parent_id, o.admin_user_id, COALESCE(o.logo_url,''), o.status, o.created_at,
		       COALESCE(p.name, '') AS parent_name,
		       COALESCE(u.display_name, '') AS admin_user_name,
		       (SELECT COUNT(*) FROM teaching_groups tg WHERE tg.school_id = o.id AND tg.status = 'active') AS group_count,
		       (SELECT COUNT(DISTINCT tgm.user_id) FROM teaching_group_members tgm
		        JOIN teaching_groups tg2 ON tg2.id = tgm.group_id WHERE tg2.school_id = o.id) AS member_count
		FROM organizations o
		LEFT JOIN organizations p ON p.id = o.parent_id
		LEFT JOIN users u ON u.id = o.admin_user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if orgType != "" {
		query += fmt.Sprintf(" AND o.type = $%d", argIdx)
		args = append(args, orgType)
		argIdx++
	}
	if parentID != "" {
		query += fmt.Sprintf(" AND o.parent_id = $%d", argIdx)
		args = append(args, parentID)
	}
	query += " ORDER BY o.type ASC, o.name ASC"

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询组织列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.OrganizationListItem
	for rows.Next() {
		item := &models.OrganizationListItem{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.Type, &item.ParentID, &item.AdminUserID,
			&item.LogoURL, &item.Status, &item.CreatedAt,
			&item.ParentName, &item.AdminUserName,
			&item.GroupCount, &item.MemberCount,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描组织行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func UpdateOrganization(ctx context.Context, id string, req *models.UpdateOrganizationRequest) error {
	query := `
		UPDATE organizations
		SET name = $1, admin_user_id = $2, settings = $3, status = $4, updated_at = $5
		WHERE id = $6
	`
	settings := req.Settings
	if settings == "" {
		settings = "{}"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx, query,
		req.Name, req.AdminUserID, settings, status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新组织失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

func DeleteOrganization(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除组织失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

func CheckOrgNameExists(ctx context.Context, name string, orgType string, excludeID string) (bool, error) {
	query := `SELECT COUNT(*) FROM organizations WHERE name = $1 AND type = $2`
	args := []interface{}{name, orgType}
	if excludeID != "" {
		query += " AND id != $3"
		args = append(args, excludeID)
	}
	var count int
	err := database.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查组织名称失败: %w", err)
	}
	return count > 0, nil
}

func GetSchoolsByRegion(ctx context.Context, regionID string) ([]*models.Organization, error) {
	query := `
		SELECT id, name, type, parent_id, admin_user_id, settings, COALESCE(logo_url,''), status, created_at, updated_at
		FROM organizations WHERE parent_id = $1 AND type = 'school' AND status = 'active'
		ORDER BY name ASC
	`
	rows, err := database.DB.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("查询学校列表失败: %w", err)
	}
	defer rows.Close()

	var orgs []*models.Organization
	for rows.Next() {
		org := &models.Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.Type, &org.ParentID, &org.AdminUserID,
			&org.Settings, &org.LogoURL, &org.Status, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学校行失败: %w", err)
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// ListExistingActiveSchoolIDs 跨校批量导入校验用：
//
// 给定一批 school_id，一次性查库返回其中【真实存在且 type='school' 且 status='active'】的 id 集合。
//
// 用途：
//
//	跨区域多校批量导入时，Excel 每行自带 school_id（前端由"学校名→ID"反查填入）。
//	service 层在逐行建用户前，先把表里去重后的所有 school_id 传入本函数拿到"有效集合"，
//	再逐行用内存集合判定（O(1)），避免每行单独查库，也挡住前端伪造/失效的 school_id
//	往不存在的学校塞人。
//
// 实现：单条 id = ANY($1) 查询，只回真实命中的有效学校 id。
// 入参空切片 → 直接返回空 map（不查库）。返回 map[string]bool，存在即 true，便于调用方判存。
func ListExistingActiveSchoolIDs(ctx context.Context, schoolIDs []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(schoolIDs) == 0 {
		return result, nil
	}
	rows, err := database.DB.Query(ctx, `
		SELECT id FROM organizations
		WHERE id = ANY($1) AND type = 'school' AND status = 'active'
	`, schoolIDs)
	if err != nil {
		return nil, fmt.Errorf("批量校验学校ID失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			result[id] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历有效学校ID结果失败: %w", err)
	}
	return result, nil
}

// ==================== 迭代一 新增：组织树递归查询 ====================

// ListDescendantSchoolIDs 递归查询某区域树下的所有学校ID（WITH RECURSIVE）
//
// 用途：
//   - region_admin 数据范围解析（ResolveDataScope 的 region_admin 分支，Phase 2 接入）
//   - 迭代二积分三级分配（区域→旗下所有学校）
//
// 语义：
//   - 从 regionID 出发，沿 parent_id 链向下递归（支持多级区域嵌套，当前数据仅两级，递归退化为一层）
//   - 只返回 type='school' 且 status='active' 的组织ID
//   - 中间层 region（多级嵌套时）参与递归遍历但不计入返回结果（只要学校）
//   - regionID 本身不是 school，不会出现在结果里
//
// fail-safe：regionID 为空 → 返回空切片（非 nil），由调用方按"空集"语义处理
func ListDescendantSchoolIDs(ctx context.Context, regionID string) ([]string, error) {
	if regionID == "" {
		return []string{}, nil
	}
	// 递归 CTE：org_tree 收集 regionID 及其所有后代组织（不限类型）
	// 最外层 SELECT 再过滤出其中的 school
	query := `
		WITH RECURSIVE org_tree AS (
			-- 基准：起始区域本身，depth=0（仅用于启动递归，不计入结果）
			SELECT id, type, parent_id, status, 0 AS depth
			FROM organizations
			WHERE id = $1
			UNION ALL
			-- 递归：所有 parent_id 指向已在树内节点的下级组织，depth 逐层+1
			SELECT o.id, o.type, o.parent_id, o.status, t.depth + 1
			FROM organizations o
			JOIN org_tree t ON o.parent_id = t.id
		)
		-- depth > 0 排除起点本身：
		--   起点若误传为 school，不会把它自己当成"下级学校"返回（堵住误用缺陷）；
		--   起点若为 region（正常用法），region 自身非 school 本就不会被选，depth 过滤无副作用。
		--   只返回通过 parent_id 链真正向下到达的下级学校。
		SELECT id FROM org_tree
		WHERE type = 'school' AND status = 'active' AND depth > 0
		ORDER BY id
	`
	rows, err := database.DB.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("递归查询区域树下学校失败: %w", err)
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历区域树学校结果失败: %w", err)
	}
	return ids, nil
}

// ==================== v172 新增：门户板块可见性查询 ====================

// parsePortalModulesFromSettings 从组织 settings(JSONB字符串) 解析 portal_modules
//
// 规则（容错优先，保证不波及存量）：
//   - settings 为空 / 非法 JSON / 无 portal_modules 键 → 返回全开
//   - portal_modules 中缺失的板块 key → 该板块按 true 处理（缺省即开启）
//   - 只有显式写成 false 的板块才会被关闭
func parsePortalModulesFromSettings(settings string) map[string]bool {
	result := models.DefaultPortalModules() // 先全开

	settings = strings.TrimSpace(settings)
	if settings == "" || settings == "{}" {
		return result
	}

	// settings 形如 {"portal_modules":{"lesson_plan":true,"workflow":false}, ...}
	var raw struct {
		PortalModules map[string]bool `json:"portal_modules"`
	}
	if err := json.Unmarshal([]byte(settings), &raw); err != nil {
		// 解析失败 → 保持全开
		return result
	}
	if raw.PortalModules == nil {
		// 没有 portal_modules 键 → 保持全开
		return result
	}

	// 用显式配置覆盖默认值（仅覆盖出现的 key，缺失的 key 保持 true）
	for _, k := range models.AllPortalModules {
		if v, ok := raw.PortalModules[k]; ok {
			result[k] = v
		}
	}
	return result
}

// GetUserPortalModules 获取用户所属组织的门户板块可见性配置
//
// 查找链路与 GetUserOrgLogo 一致：
//
//	school_members → 学校 → 学校 settings；school_members 查不到则教研组兜底反查学校
//
// 解析学校 settings 里的 portal_modules；未绑定任何组织 / 查不到 → 返回全开
//
// 注意：admin 的"全开"由 auth_service 层兜底，不在此函数处理（此函数只管按组织配置返回）
func GetUserPortalModules(ctx context.Context, userID string) map[string]bool {
	var settings string

	// 1. 通过 school_members 查用户所属学校的 settings
	err := database.DB.QueryRow(ctx, `
		SELECT COALESCE(o.settings, '{}')
		FROM school_members sm
		JOIN organizations o ON o.id = sm.school_id
		WHERE sm.user_id = $1 AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(&settings)
	if err != nil {
		// 2. 兜底：通过教研组反查学校 settings
		err = database.DB.QueryRow(ctx, `
			SELECT COALESCE(o.settings, '{}')
			FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			JOIN organizations o ON o.id = tg.school_id
			WHERE tgm.user_id = $1 AND o.status = 'active'
			LIMIT 1
		`, userID).Scan(&settings)
		if err != nil {
			// 未绑定任何学校 → 全开
			return models.DefaultPortalModules()
		}
	}

	return parsePortalModulesFromSettings(settings)
}

// ==================== v122 方案B：school_members 直接归属 ====================
//
// school_members 是"学校直接成员名单"的权威来源（v122 新增）。
// 与 teaching_group_members 正交：教研组成员自动算本校成员，但本校成员不一定要在教研组。
// 新建用户入校、加入本校教研组 都会自动写入 school_members。

// AddSchoolMember 将用户加入学校的直接成员名单
// - 幂等：ON CONFLICT 不报错
// - source 记录来源('school_admin_create'/'admin_create'/'group_member'/'migration'/'manual')
func AddSchoolMember(ctx context.Context, schoolID string, userID string, source string) error {
	if schoolID == "" || userID == "" {
		return fmt.Errorf("schoolID 或 userID 为空")
	}
	if source == "" {
		source = "manual"
	}
	_, err := database.DB.Exec(ctx, `
		INSERT INTO school_members (school_id, user_id, joined_at, source)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (school_id, user_id) DO NOTHING
	`, schoolID, userID, source)
	if err != nil {
		return fmt.Errorf("加入学校成员失败: %w", err)
	}
	return nil
}

// RemoveSchoolMember 从学校直接成员名单移除用户
// - 仅当学校管理员显式移除用户时调用
// - 禁用用户不调此函数（禁用只改 users.status）
func RemoveSchoolMember(ctx context.Context, schoolID string, userID string) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM school_members WHERE school_id = $1 AND user_id = $2`,
		schoolID, userID,
	)
	if err != nil {
		return fmt.Errorf("移除学校成员失败: %w", err)
	}
	return nil
}

// IsUserInSchool 检查用户是否属于指定学校（v122 方案B 权威判定）
// 同时兜底查 teaching_group_members，防止回填遗漏或新加入教研组但 school_members 漏写
//
// 注意（迭代一说明）：
//
//	本函数保留教研组兜底，服务于"学校管理员校验本校成员、放行管理操作"等 6 处既有调用点
//	（宁可多放行本校的人，也别漏掉只在教研组的人）。行为与历史一致，不改。
//	而"数据隔离/防跨校越权"场景（教案/审核隔离）应改用下方 IsUserInSchoolStrict（无兜底），
//	两者命名即表达语义差异，杜绝再次误用。
func IsUserInSchool(ctx context.Context, userID string, schoolID string) (bool, error) {
	var count int
	// 主判：school_members
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM school_members WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查学校直接成员失败: %w", err)
	}
	if count > 0 {
		return true, nil
	}
	// 兜底：通过教研组反查（历史兼容 + 防漏）
	err = database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		WHERE tgm.user_id = $1 AND tg.school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户学校归属(教研组兜底)失败: %w", err)
	}
	return count > 0, nil
}

// IsUserInSchoolStrict 严格判定用户是否属于指定学校（迭代一新增：只认 school_members，无教研组兜底）
//
// 与 IsUserInSchool 的区别：
//   - IsUserInSchool      ：school_members 主判 + 教研组兜底（宽松，给"放行管理操作"用）
//   - IsUserInSchoolStrict：只认 school_members（严格，给"数据隔离/防跨校越权"用）
//
// 设计动机（P0-02 根治）：
//
//	P0-02 漏洞的根源是"数据隔离误用了带兜底的归属判断"——只要加个教研组就能看跨校教案。
//	严格版只认 school_members 这一唯一权威归属来源，加教研组也无法越权看跨校数据。
//
// 用途：Phase 4 教案/审核数据隔离收口时使用；以及任何需要严格归属判断的新场景。
func IsUserInSchoolStrict(ctx context.Context, userID string, schoolID string) (bool, error) {
	if userID == "" || schoolID == "" {
		return false, nil
	}
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM school_members WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("严格检查学校直接成员失败: %w", err)
	}
	return count > 0, nil
}

// ListSchoolMemberIDs 返回某学校所有成员的 user_id
// 用于 ListAdminUsers 按学校筛选的 IN 子查询构建
func ListSchoolMemberIDs(ctx context.Context, schoolID string) ([]string, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT user_id FROM school_members WHERE school_id = $1`,
		schoolID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询学校成员ID列表失败: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// IsUserInSchoolByGroup 检查用户是否通过教研组归属于某学校（v110 老接口，保留向后兼容）
// Deprecated: v122 改用 IsUserInSchool（school_members 主判 + 教研组兜底）
// 保留此函数避免未发现的调用点编译失败
func IsUserInSchoolByGroup(ctx context.Context, userID string, schoolID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		WHERE tgm.user_id = $1 AND tg.school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户学校归属失败: %w", err)
	}
	return count > 0, nil
}

// ==================== 组织 Logo ====================

// UpdateOrganizationLogo 更新组织Logo URL
func UpdateOrganizationLogo(ctx context.Context, id string, logoURL string) error {
	sql := `UPDATE organizations SET logo_url = $1, updated_at = $2 WHERE id = $3`
	result, err := database.DB.Exec(ctx, sql, logoURL, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新组织Logo失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// GetUserOrgLogo 获取用户所属组织的Logo和名称
// 查找链路：school_members → 学校 → 学校Logo → 如果没有则取区域Logo
// 返回 (logoURL, orgName)，全部为空表示用户未绑定任何组织
func GetUserOrgLogo(ctx context.Context, userID string) (string, string) {
	// 1. 通过school_members查用户所属学校
	var schoolID, schoolName, schoolLogoURL string
	var parentID *string
	err := database.DB.QueryRow(ctx, `
		SELECT o.id, o.name, COALESCE(o.logo_url, ''), o.parent_id
		FROM school_members sm
		JOIN organizations o ON o.id = sm.school_id
		WHERE sm.user_id = $1 AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(&schoolID, &schoolName, &schoolLogoURL, &parentID)
	if err != nil {
		// 兜底：通过教研组反查学校
		err = database.DB.QueryRow(ctx, `
			SELECT o.id, o.name, COALESCE(o.logo_url, ''), o.parent_id
			FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			JOIN organizations o ON o.id = tg.school_id
			WHERE tgm.user_id = $1 AND o.status = 'active'
			LIMIT 1
		`, userID).Scan(&schoolID, &schoolName, &schoolLogoURL, &parentID)
		if err != nil {
			return "", ""
		}
	}

	// 2. 如果学校有Logo，直接返回
	if schoolLogoURL != "" {
		return schoolLogoURL, schoolName
	}

	// 3. 学校没有Logo，尝试从所属区域获取
	if parentID != nil && *parentID != "" {
		var regionLogoURL, regionName string
		err = database.DB.QueryRow(ctx, `
			SELECT COALESCE(logo_url, ''), name
			FROM organizations WHERE id = $1 AND status = 'active'
		`, *parentID).Scan(&regionLogoURL, &regionName)
		if err == nil && regionLogoURL != "" {
			return regionLogoURL, schoolName
		}
	}

	// 4. 都没有Logo，返回学校名称但空Logo
	return "", schoolName
}
