package repository

// organization_region_scope_repo.go
//
// 本文件专门提供区域管理员学校范围解析所需的数据查询。
//
// 设计目标：
//   区域管理员可见学校必须同时满足以下三个条件：
//     1. 学校确实位于其管辖区域的组织树下；
//     2. 学校状态为 active；
//     3. 学校 education_domain 与区域管理员固定教育域完全一致。
//
// 为什么不直接修改 ListDescendantSchoolIDs：
//   既有函数还被积分、学校管理等其它业务调用，其语义是“区域树下全部 active 学校”。
//   若直接增加教育域过滤，会改变其它调用链的既有行为并造成上下文漂移。
//   因此本文件新增独立、语义明确的查询函数，只供教育域范围解析使用。
//
// 安全原则：
//   - 只接受 k12、vocational、adult 三个具体教学教育域；
//   - mixed、common、空值和非法值全部拒绝；
//   - 不调用 NormalizeEducationDomain，不允许非法值回退为 K12；
//   - 区域不存在、已禁用或没有同域学校时返回非 nil 空切片；
//   - 数据库查询或扫描失败直接返回错误，由上层 fail-closed 收窄为空集。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListDescendantSchoolIDsByEducationDomain 查询指定区域组织树下的同域 active 学校。
//
// 参数：
//   - regionID：区域组织 ID；
//   - educationDomain：区域管理员已经严格解析出的固定教学教育域。
//
// 返回：
//   - 区域存在但没有同域学校时返回 []string{}；
//   - 区域为空时返回 []string{}；
//   - 教育域非法时返回错误，绝不执行数据库查询；
//   - 查询结果按学校 ID 稳定排序。
//
// 递归规则与 ListDescendantSchoolIDs 保持一致：
//   - 起始区域 depth=0，只作为递归根节点；
//   - 递归遍历任意层级的子区域和学校；
//   - 最终只选择 depth>0 的学校节点；
//   - 中间区域只参与递归，不进入学校结果。
func ListDescendantSchoolIDsByEducationDomain(
	ctx context.Context,
	regionID string,
	educationDomain string,
) ([]string, error) {
	regionID = strings.TrimSpace(regionID)
	domain := strings.ToLower(strings.TrimSpace(educationDomain))

	if !models.IsTeachingEducationDomain(domain) {
		return nil, fmt.Errorf("区域管理员固定教育域无效")
	}
	if regionID == "" {
		return []string{}, nil
	}

	rows, err := database.DB.Query(ctx, `
		WITH RECURSIVE org_tree AS (
			-- 基准节点必须是真实、启用的区域。
			-- 区域不存在、类型错误或已禁用时，整个递归结果自然为空。
			SELECT
				id,
				type,
				parent_id,
				status,
				COALESCE(education_domain, '') AS education_domain,
				0 AS depth
			FROM organizations
			WHERE id = $1
			  AND type = 'region'
			  AND status = 'active'

			UNION ALL

			-- 递归遍历所有下级组织。
			SELECT
				o.id,
				o.type,
				o.parent_id,
				o.status,
				COALESCE(o.education_domain, ''),
				t.depth + 1
			FROM organizations o
			JOIN org_tree t
			  ON o.parent_id = t.id
		)
		SELECT id::text
		FROM org_tree
		WHERE depth > 0
		  AND type = 'school'
		  AND status = 'active'
		  AND LOWER(BTRIM(education_domain)) = $2
		ORDER BY id::text
	`, regionID, domain)
	if err != nil {
		return nil, fmt.Errorf("查询区域树下同域学校失败: %w", err)
	}
	defer rows.Close()

	schoolIDs := make([]string, 0)
	for rows.Next() {
		var schoolID string
		if err := rows.Scan(&schoolID); err != nil {
			return nil, fmt.Errorf("扫描区域树同域学校失败: %w", err)
		}
		if strings.TrimSpace(schoolID) != "" {
			schoolIDs = append(schoolIDs, schoolID)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历区域树同域学校失败: %w", err)
	}

	return schoolIDs, nil
}
