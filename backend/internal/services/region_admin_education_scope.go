package services

// region_admin_education_scope.go
//
// 本文件是“区域管理员学校范围同域过滤”的唯一统一解析入口。
//
// 输入：
//   - 当前区域管理员 userID。
//
// 解析过程：
//   1. 通过 repository.ResolveUserEducationContext 严格解析管理员固定教育域；
//   2. 通过 repository.ListRegionIDsByAdmin 解析其真实管辖区域；
//   3. 对每个管辖区域递归查询与固定教育域一致的 active 学校；
//   4. 对区域和学校 ID 去空、去重、稳定排序后返回。
//
// 输出：
//   - 固定具体教育域；
//   - 管辖区域 ID 白名单；
//   - 管辖区域树下同域 active 学校 ID 白名单。
//
// 业务边界：
//   学校必须满足：
//     属于管辖区域
//     AND status='active'
//     AND education_domain=区域管理员固定教育域
//
// 特别说明：
//   不再把“本人兼任学校管理员的本校”无条件添加到区域管理员范围。
//   该学校只有在本身位于管辖区域树下、为 active 且教育域一致时，才会由递归查询自然命中。
//   这保证范围严格符合 PRD，而不是“辖区学校与本人本校的并集”。
//
// fail-closed 原则：
//   - 无任命；
//   - 任命教育域为空、非法或冲突；
//   - 没有管辖区域；
//   - 任一数据库查询失败；
//   均返回 ErrRegionAdminEducationScopeNotReady，上层必须收窄为空集，不能回退全量或 K12。

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ErrRegionAdminEducationScopeNotReady 表示无法安全构造区域管理员同域学校范围。
var ErrRegionAdminEducationScopeNotReady = errors.New(
	"区域管理员教育域或管辖范围尚未正确配置",
)

// RegionAdminEducationScope 是区域管理员的统一同域学校范围结果。
type RegionAdminEducationScope struct {
	EducationDomain string   // k12、vocational 或 adult
	RegionIDs       []string // 真实管辖的 active 区域
	SchoolIDs       []string // 管辖区域树下同域 active 学校
}

var regionAdminEducationScopeLog = logger.WithModule(
	"services.region_admin_education_scope",
)

// normalizeRegionEducationScopeIDs 对范围 ID 去空、去重并稳定排序。
//
// 统一排序不是权限要求，但可以保证：
//   - 测试结果稳定；
//   - 日志和 SQL 参数稳定；
//   - 多次请求的返回顺序不会受 map 或 UNION 执行顺序影响。
func normalizeRegionEducationScopeIDs(values []string) []string {
	idSet := make(map[string]struct{})

	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		idSet[id] = struct{}{}
	}

	result := make([]string, 0, len(idSet))
	for id := range idSet {
		result = append(result, id)
	}
	sort.Strings(result)

	return result
}

// ResolveRegionAdminEducationScope 解析区域管理员统一同域学校范围。
func ResolveRegionAdminEducationScope(
	ctx context.Context,
	userID string,
) (*RegionAdminEducationScope, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf(
			"%w: 用户ID为空",
			ErrRegionAdminEducationScopeNotReady,
		)
	}

	// 1. 严格解析固定教育域。
	//
	// 该入口内部会进入 region_admin 专用解析器：
	//   - 只读取 organization_admins.education_domain；
	//   - 多区域任命必须全部同域；
	//   - 遗留 admin_user_id 单字段任命视为未配置；
	//   - 空值、mixed、common、非法值和数据库错误全部拒绝；
	//   - 绝不回退 K12。
	educationContext, err := repository.ResolveUserEducationContext(
		ctx,
		userID,
		models.RoleRegionAdmin,
	)
	if err != nil {
		regionAdminEducationScopeLog.Warn(
			"解析区域管理员固定教育域失败",
			"user_id",
			userID,
			"error",
			err,
		)
		return nil, fmt.Errorf(
			"%w: 固定教育域解析失败",
			ErrRegionAdminEducationScopeNotReady,
		)
	}

	if educationContext == nil ||
		!models.IsTeachingEducationDomain(educationContext.EducationDomain) {
		regionAdminEducationScopeLog.Warn(
			"区域管理员固定教育域结果无效",
			"user_id",
			userID,
		)
		return nil, fmt.Errorf(
			"%w: 固定教育域为空或非法",
			ErrRegionAdminEducationScopeNotReady,
		)
	}

	educationDomain := strings.ToLower(
		strings.TrimSpace(educationContext.EducationDomain),
	)

	// 2. 解析真实管辖区域。
	regionIDs, err := repository.ListRegionIDsByAdmin(ctx, userID)
	if err != nil {
		regionAdminEducationScopeLog.Warn(
			"查询区域管理员管辖区域失败",
			"user_id",
			userID,
			"error",
			err,
		)
		return nil, fmt.Errorf(
			"%w: 查询管辖区域失败",
			ErrRegionAdminEducationScopeNotReady,
		)
	}

	regionIDs = normalizeRegionEducationScopeIDs(regionIDs)
	if len(regionIDs) == 0 {
		return nil, fmt.Errorf(
			"%w: 没有有效区域任命",
			ErrRegionAdminEducationScopeNotReady,
		)
	}

	// 3. 逐区域查询同域 active 学校。
	//
	// 任意一个区域查询失败即整体失败，不能忽略失败区域后返回不完整结果，
	// 防止调用方误以为返回值是完整授权范围。
	schoolIDSet := make(map[string]struct{})
	for _, regionID := range regionIDs {
		schoolIDs, queryErr :=
			repository.ListDescendantSchoolIDsByEducationDomain(
				ctx,
				regionID,
				educationDomain,
			)
		if queryErr != nil {
			regionAdminEducationScopeLog.Warn(
				"查询区域树下同域学校失败",
				"user_id",
				userID,
				"region_id",
				regionID,
				"education_domain",
				educationDomain,
				"error",
				queryErr,
			)
			return nil, fmt.Errorf(
				"%w: 查询同域学校失败",
				ErrRegionAdminEducationScopeNotReady,
			)
		}

		for _, schoolID := range schoolIDs {
			schoolID = strings.TrimSpace(schoolID)
			if schoolID != "" {
				schoolIDSet[schoolID] = struct{}{}
			}
		}
	}

	schoolIDs := make([]string, 0, len(schoolIDSet))
	for schoolID := range schoolIDSet {
		schoolIDs = append(schoolIDs, schoolID)
	}
	schoolIDs = normalizeRegionEducationScopeIDs(schoolIDs)

	return &RegionAdminEducationScope{
		EducationDomain: educationDomain,
		RegionIDs:       regionIDs,
		SchoolIDs:       schoolIDs,
	}, nil
}
