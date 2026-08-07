package repository

// component_management_domain_repo.go — 组件管理CRUD教育域仓储。
//
// 本文件仅负责组件管理页面的列表、更新、删除和审核。
// 创建与详情读取复用component_domain_repo.go中已有的显式写域和域感知详情函数。
//
// 读取规则：
//   - k12/vocational/adult：可以读取同域或common；
//   - mixed：可以读取全部四种合法资源域；
//   - 其它当前域：返回空结果。
//
// 修改规则：
//   - 具体教学域只能修改完全同域资源，不能修改common；
//   - mixed管理上下文可以治理全部合法资源域；
//   - 不存在和无权操作统一返回ErrComponentNotFound。

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ListComponentsForEducationDomain 按可信Actor教育域查询组件列表。
//
// currentDomain必须由服务端Actor解析得出。
// targetDomain仅供mixed管理页面精确筛选；普通Actor不会把客户端筛选传到这里。
func ListComponentsForEducationDomain(
	ctx context.Context,
	currentDomain string,
	targetDomain string,
	libraryType string,
	subject string,
	reviewStatus string,
	scope string,
	limit int,
	offset int,
) ([]*models.ComponentListItem, int, error) {
	where := `
		WHERE c.status = 'active'
		  AND (
			(
				$1 = 'mixed'
				AND c.education_domain IN (
					'k12',
					'vocational',
					'adult',
					'common'
				)
			)
			OR
			(
				$1 IN (
					'k12',
					'vocational',
					'adult'
				)
				AND (
					c.education_domain = $1
					OR c.education_domain = 'common'
				)
			)
		  )
		  AND (
			$2 = ''
			OR c.education_domain = $2
		  )
	`

	args := []interface{}{
		currentDomain,
		targetDomain,
	}
	argIndex := 3

	if libraryType != "" {
		where += fmt.Sprintf(
			" AND c.library_type = $%d",
			argIndex,
		)
		args = append(args, libraryType)
		argIndex++
	}

	if subject != "" {
		where += fmt.Sprintf(
			" AND (c.subject = $%d OR c.subject = 'general')",
			argIndex,
		)
		args = append(args, subject)
		argIndex++
	}

	if reviewStatus != "" {
		where += fmt.Sprintf(
			" AND c.review_status = $%d",
			argIndex,
		)
		args = append(args, reviewStatus)
		argIndex++
	}

	if scope != "" {
		where += fmt.Sprintf(
			" AND c.scope = $%d",
			argIndex,
		)
		args = append(args, scope)
		argIndex++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM lesson_plan_components c
	` + where

	var total int

	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"按教育域查询组件总数失败: %w",
			err,
		)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	listQuery := `
		SELECT
			c.id,
			c.education_domain,
			c.library_type,
			c.subject,
			COALESCE(c.grade_range, ''),
			c.injection_mode,
			c.display_label,
			c.quality_score,
			c.usage_count,
			c.select_count,
			c.source,
			c.review_status,
			c.scope,
			c.status,
			c.created_at
		FROM lesson_plan_components c
	` + where + fmt.Sprintf(
		`
			ORDER BY
				c.quality_score DESC,
				c.created_at DESC
			LIMIT $%d
			OFFSET $%d
		`,
		argIndex,
		argIndex+1,
	)

	listArgs := append(
		append([]interface{}{}, args...),
		limit,
		offset,
	)

	rows, err := database.DB.Query(
		ctx,
		listQuery,
		listArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"按教育域查询组件列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.ComponentListItem, 0)

	for rows.Next() {
		item := &models.ComponentListItem{}

		if err := rows.Scan(
			&item.ID,
			&item.EducationDomain,
			&item.LibraryType,
			&item.Subject,
			&item.GradeRange,
			&item.InjectionMode,
			&item.DisplayLabel,
			&item.QualityScore,
			&item.UsageCount,
			&item.SelectCount,
			&item.Source,
			&item.ReviewStatus,
			&item.Scope,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描教育域组件列表失败: %w",
				err,
			)
		}

		item.LibraryName =
			models.LibraryTypeNameMap[item.LibraryType]

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历教育域组件列表失败: %w",
			err,
		)
	}

	return items, total, nil
}

// UpdateComponentForEducationDomain 按可信管理域更新组件。
func UpdateComponentForEducationDomain(
	ctx context.Context,
	id string,
	currentDomain string,
	req *models.UpdateComponentRequest,
) error {
	tags := req.Tags
	if tags == "" {
		tags = "[]"
	}

	content := req.Content
	if content == "" {
		content = "{}"
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	result, err := database.DB.Exec(
		ctx,
		`
			UPDATE lesson_plan_components c
			SET
				subject = $1,
				grade_range = $2,
				tags = $3,
				injection_mode = $4,
				display_label = $5,
				design_logic = $6,
				example_snippet = $7,
				full_guide = $8,
				content = $9,
				scope = $10,
				scope_ref_id = $11,
				status = $12,
				updated_at = $13
			WHERE c.id = $14
			  AND (
				(
					$15 = 'mixed'
					AND c.education_domain IN (
						'k12',
						'vocational',
						'adult',
						'common'
					)
				)
				OR
				(
					$15 IN (
						'k12',
						'vocational',
						'adult'
					)
					AND c.education_domain = $15
				)
			  )
		`,
		req.Subject,
		req.GradeRange,
		tags,
		req.InjectionMode,
		req.DisplayLabel,
		req.DesignLogic,
		req.ExampleSnippet,
		req.FullGuide,
		content,
		req.Scope,
		req.ScopeRefID,
		status,
		time.Now(),
		id,
		currentDomain,
	)
	if err != nil {
		return fmt.Errorf(
			"按教育域更新组件失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrComponentNotFound
	}

	return nil
}

// DeleteComponentForEducationDomain 按可信管理域软删除组件。
func DeleteComponentForEducationDomain(
	ctx context.Context,
	id string,
	currentDomain string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
			UPDATE lesson_plan_components c
			SET
				status = 'archived',
				updated_at = $1
			WHERE c.id = $2
			  AND (
				(
					$3 = 'mixed'
					AND c.education_domain IN (
						'k12',
						'vocational',
						'adult',
						'common'
					)
				)
				OR
				(
					$3 IN (
						'k12',
						'vocational',
						'adult'
					)
					AND c.education_domain = $3
				)
			  )
		`,
		time.Now(),
		id,
		currentDomain,
	)
	if err != nil {
		return fmt.Errorf(
			"按教育域删除组件失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrComponentNotFound
	}

	return nil
}

// ReviewComponentForEducationDomain 按可信管理域审核组件。
func ReviewComponentForEducationDomain(
	ctx context.Context,
	id string,
	currentDomain string,
	reviewerID string,
	decision string,
) error {
	now := time.Now()

	result, err := database.DB.Exec(
		ctx,
		`
			UPDATE lesson_plan_components c
			SET
				review_status = $1,
				reviewed_by = $2,
				reviewed_at = $3,
				updated_at = $3
			WHERE c.id = $4
			  AND (
				(
					$5 = 'mixed'
					AND c.education_domain IN (
						'k12',
						'vocational',
						'adult',
						'common'
					)
				)
				OR
				(
					$5 IN (
						'k12',
						'vocational',
						'adult'
					)
					AND c.education_domain = $5
				)
			  )
		`,
		decision,
		reviewerID,
		now,
		id,
		currentDomain,
	)
	if err != nil {
		return fmt.Errorf(
			"按教育域审核组件失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrComponentNotFound
	}

	return nil
}
