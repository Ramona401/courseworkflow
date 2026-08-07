package repository

// courseware_component_domain_repo.go — 课件组件教育域CRUD仓储。
//
// 本文件只承载管理面的创建、列表、详情、更新和删除。
// 运行时匹配及页面组件ID复核位于：
// courseware_component_domain_runtime_repo.go。
//
// 读取规则：
//   - k12/vocational/adult：只可读取同域或common；
//   - mixed管理上下文：可读取四种合法资源域；
//   - 空值、common当前域和非法当前域：查询不到任何数据。
//
// 修改规则：
//   - 具体教学域只能修改完全同域资源，不能修改common；
//   - mixed管理上下文可以治理合法资源域；
//   - 更新语句绝不修改education_domain；
//   - 不存在和异域统一返回ErrCWComponentDomainNotFound。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrCWComponentDomainNotFound 对外统一表示组件不存在或当前域不可见。
var ErrCWComponentDomainNotFound = errors.New("课件组件不存在")

// normalizeCWComponentRepositoryDomain 只做格式归一，不做非法值回退。
func normalizeCWComponentRepositoryDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

// CreateCWComponentWithEducationDomain 显式写入可信资源教育域。
func CreateCWComponentWithEducationDomain(
	ctx context.Context,
	component *models.CoursewareComponent,
	educationDomain string,
) error {
	educationDomain = normalizeCWComponentRepositoryDomain(
		educationDomain,
	)

	if component == nil ||
		!models.IsResourceEducationDomain(educationDomain) {
		return fmt.Errorf("创建课件组件失败：教育域无效")
	}

	query := `
		INSERT INTO courseware_components (
			id,
			education_domain,
			name,
			description,
			component_type,
			code_content,
			preview_image_url,
			preview_html,
			subject_scope,
			grade_scope,
			component_index,
			idx_interaction_level,
			idx_visual_format,
			idx_tech_tag,
			tech_dependencies,
			tags,
			is_active,
			review_status
		) VALUES (
			gen_random_uuid(),
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14::jsonb, $15::jsonb,
			$16, $17
		)
		RETURNING
			id,
			created_at,
			updated_at
	`

	err := database.DB.QueryRow(
		ctx,
		query,
		educationDomain,
		component.Name,
		component.Description,
		component.ComponentType,
		component.CodeContent,
		component.PreviewImageURL,
		component.PreviewHTML,
		component.SubjectScope,
		component.GradeScope,
		component.ComponentIndex,
		component.IdxInteractionLevel,
		component.IdxVisualFormat,
		component.IdxTechTag,
		nullIfEmpty(component.TechDependencies),
		nullIfEmpty(component.Tags),
		component.IsActive,
		component.ReviewStatus,
	).Scan(
		&component.ID,
		&component.CreatedAt,
		&component.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建课件组件失败: %w", err)
	}

	return nil
}

// GetCWComponentByIDForEducationDomain 按可信当前域读取直接ID详情。
//
// 使用id::text进行匹配，使格式错误的ID与不存在ID一样得到空结果，
// 不会因UUID强制转换错误泄漏内部数据库细节。
func GetCWComponentByIDForEducationDomain(
	ctx context.Context,
	id string,
	currentDomain string,
) (*models.CWComponentResource, error) {
	resource := &models.CWComponentResource{
		CoursewareComponent: &models.CoursewareComponent{},
	}

	query := `
		SELECT
			id,
			education_domain,
			name,
			COALESCE(description, ''),
			component_type,
			code_content,
			COALESCE(preview_image_url, ''),
			COALESCE(preview_html, ''),
			COALESCE(subject_scope, 'ALL'),
			COALESCE(grade_scope, 'ALL'),
			COALESCE(component_index, ''),
			idx_interaction_level,
			COALESCE(idx_visual_format, ''),
			COALESCE(idx_tech_tag, ''),
			COALESCE(tech_dependencies::text, ''),
			COALESCE(tags::text, ''),
			is_active,
			review_status,
			created_at,
			updated_at
		FROM courseware_components
		WHERE id::text = $1
		  AND (
			(
				$2 = 'mixed'
				AND education_domain IN (
					'k12',
					'vocational',
					'adult',
					'common'
				)
			)
			OR
			(
				$2 IN (
					'k12',
					'vocational',
					'adult'
				)
				AND (
					education_domain = $2
					OR education_domain = 'common'
				)
			)
		  )
	`

	err := database.DB.QueryRow(
		ctx,
		query,
		strings.TrimSpace(id),
		normalizeCWComponentRepositoryDomain(currentDomain),
	).Scan(
		&resource.ID,
		&resource.EducationDomain,
		&resource.Name,
		&resource.Description,
		&resource.ComponentType,
		&resource.CodeContent,
		&resource.PreviewImageURL,
		&resource.PreviewHTML,
		&resource.SubjectScope,
		&resource.GradeScope,
		&resource.ComponentIndex,
		&resource.IdxInteractionLevel,
		&resource.IdxVisualFormat,
		&resource.IdxTechTag,
		&resource.TechDependencies,
		&resource.Tags,
		&resource.IsActive,
		&resource.ReviewStatus,
		&resource.CreatedAt,
		&resource.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCWComponentDomainNotFound
		}

		return nil, fmt.Errorf(
			"按教育域读取课件组件失败: %w",
			err,
		)
	}

	return resource, nil
}

// ListCWComponentsForEducationDomain 按可信Actor域查询组件列表。
//
// targetDomain只供mixed管理页面精确筛选；普通教学Actor由Service
// 强制传空值，客户端不能利用筛选参数扩大读取范围。
func ListCWComponentsForEducationDomain(
	ctx context.Context,
	currentDomain string,
	targetDomain string,
	componentType string,
	subjectScope string,
	gradeScope string,
	isActive *bool,
	limit int,
	offset int,
) ([]*models.CWComponentDomainListItem, int, error) {
	conditions := []string{
		`(
			(
				$1 = 'mixed'
				AND education_domain IN (
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
					education_domain = $1
					OR education_domain = 'common'
				)
			)
		)`,
		`(
			$2 = ''
			OR education_domain = $2
		)`,
	}

	args := []interface{}{
		normalizeCWComponentRepositoryDomain(currentDomain),
		normalizeCWComponentRepositoryDomain(targetDomain),
	}
	argIndex := 3

	if componentType != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"component_type = $%d",
				argIndex,
			),
		)
		args = append(args, componentType)
		argIndex++
	}

	if subjectScope != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"(subject_scope = $%d OR subject_scope = 'ALL')",
				argIndex,
			),
		)
		args = append(args, subjectScope)
		argIndex++
	}

	if gradeScope != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"(grade_scope = $%d OR grade_scope = 'ALL')",
				argIndex,
			),
		)
		args = append(args, gradeScope)
		argIndex++
	}

	if isActive != nil {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"is_active = $%d",
				argIndex,
			),
		)
		args = append(args, *isActive)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(
		`
			SELECT COUNT(*)
			FROM courseware_components
			WHERE %s
		`,
		whereClause,
	)

	var total int

	if err := database.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"按教育域查询课件组件总数失败: %w",
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

	listQuery := fmt.Sprintf(
		`
			SELECT
				id,
				education_domain,
				name,
				COALESCE(description, ''),
				component_type,
				COALESCE(preview_image_url, ''),
				COALESCE(subject_scope, 'ALL'),
				COALESCE(grade_scope, 'ALL'),
				COALESCE(component_index, ''),
				idx_interaction_level,
				is_active,
				review_status,
				created_at
			FROM courseware_components
			WHERE %s
			ORDER BY
				education_domain,
				created_at DESC
			LIMIT $%d
			OFFSET $%d
		`,
		whereClause,
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
			"按教育域查询课件组件列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CWComponentDomainListItem,
		0,
	)

	for rows.Next() {
		item := &models.CWComponentDomainListItem{
			CWComponentListItem: &models.CWComponentListItem{},
		}

		if err := rows.Scan(
			&item.ID,
			&item.EducationDomain,
			&item.Name,
			&item.Description,
			&item.ComponentType,
			&item.PreviewImageURL,
			&item.SubjectScope,
			&item.GradeScope,
			&item.ComponentIndex,
			&item.IdxInteractionLevel,
			&item.IsActive,
			&item.ReviewStatus,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf(
				"扫描教育域课件组件列表失败: %w",
				err,
			)
		}

		item.ComponentTypeName =
			models.CWComponentTypeNameMap[item.ComponentType]

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"遍历教育域课件组件列表失败: %w",
			err,
		)
	}

	return items, total, nil
}

// UpdateCWComponentForEducationDomain 按可信管理域更新组件内容。
//
// 本函数不更新education_domain，禁止通过普通内容编辑原地迁移资源。
func UpdateCWComponentForEducationDomain(
	ctx context.Context,
	id string,
	currentDomain string,
	request *models.UpdateCWComponentRequest,
) error {
	var activeValue interface{}

	if request.IsActive != nil {
		activeValue = *request.IsActive
	}

	result, err := database.DB.Exec(
		ctx,
		`
			UPDATE courseware_components
			SET
				name = $1,
				description = $2,
				component_type = $3,
				code_content = $4,
				preview_image_url = $5,
				preview_html = $6,
				subject_scope = $7,
				grade_scope = $8,
				tech_dependencies = $9::jsonb,
				tags = $10::jsonb,
				is_active = COALESCE(
					$11::boolean,
					is_active
				),
				review_status = CASE
					WHEN $12 = ''
					THEN review_status
					ELSE $12
				END,
				updated_at = $13
			WHERE id::text = $14
			  AND (
				(
					$15 = 'mixed'
					AND education_domain IN (
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
					AND education_domain = $15
				)
			  )
		`,
		request.Name,
		request.Description,
		request.ComponentType,
		request.CodeContent,
		request.PreviewImageURL,
		request.PreviewHTML,
		request.SubjectScope,
		request.GradeScope,
		nullIfEmpty(request.TechDependencies),
		nullIfEmpty(request.Tags),
		activeValue,
		request.ReviewStatus,
		time.Now(),
		strings.TrimSpace(id),
		normalizeCWComponentRepositoryDomain(currentDomain),
	)
	if err != nil {
		return fmt.Errorf(
			"按教育域更新课件组件失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrCWComponentDomainNotFound
	}

	return nil
}

// DeleteCWComponentForEducationDomain 按可信管理域物理删除组件。
func DeleteCWComponentForEducationDomain(
	ctx context.Context,
	id string,
	currentDomain string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
			DELETE FROM courseware_components
			WHERE id::text = $1
			  AND (
				(
					$2 = 'mixed'
					AND education_domain IN (
						'k12',
						'vocational',
						'adult',
						'common'
					)
				)
				OR
				(
					$2 IN (
						'k12',
						'vocational',
						'adult'
					)
					AND education_domain = $2
				)
			  )
		`,
		strings.TrimSpace(id),
		normalizeCWComponentRepositoryDomain(currentDomain),
	)
	if err != nil {
		return fmt.Errorf(
			"按教育域删除课件组件失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrCWComponentDomainNotFound
	}

	return nil
}
