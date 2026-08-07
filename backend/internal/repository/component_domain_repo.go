package repository

// component_domain_repo.go — 教案组件教育域数据访问底座。
//
// 本文件专门承载组件资源教育域相关的查询和写入，避免继续扩大已经超过
// 600行的component_repo.go。旧Repository函数会在后续接线批次逐步退出
// 运行时主链，本文件提供统一、fail-closed的新入口。
//
// 设计原则：
//   1. 组件创建显式写入可信education_domain；
//   2. 具体教学域只读取同域或common；
//   3. mixed只允许合法管理Actor跨域管理；
//   4. 空值、common当前域、非法当前域全部查询不到数据；
//   5. 历史配方和阶段输出中的异域ID通过过滤查询安全忽略；
//   6. 新提交的直接ID由Service对查询结果做整组严格校验。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ComponentAccessRecord 是直接ID严格校验所需的最小数据库快照。
//
// Repository只提供客观状态，不在此决定是返回404、400还是静默过滤；
// 新提交与历史引用的不同处理策略由Service统一决定。
type ComponentAccessRecord struct {
	ID              string
	EducationDomain string
	LibraryType     string
	Status          string
	ReviewStatus    string
}

// CreateComponentWithEducationDomain 创建带显式资源教育域的组件。
//
// 新版应用必须使用本函数，不依赖数据库触发器猜测资源域。
// 数据库触发器只用于旧二进制、系统种子和运维修复兜底。
func CreateComponentWithEducationDomain(
	ctx context.Context,
	component *models.LessonPlanComponent,
) error {
	query := `
		INSERT INTO lesson_plan_components (
			education_domain,
			library_type,
			subject,
			grade_range,
			tags,
			injection_mode,
			display_label,
			design_logic,
			example_snippet,
			full_guide,
			content,
			source,
			source_ref,
			scope,
			scope_ref_id,
			created_by,
			review_status,
			status
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18
		)
		RETURNING id, education_domain, created_at, updated_at
	`

	subject := component.Subject
	if subject == "" {
		subject = "general"
	}

	tags := component.Tags
	if tags == "" {
		tags = "[]"
	}

	injectionMode := component.InjectionMode
	if injectionMode == "" {
		injectionMode = models.InjectionOnDemand
	}

	content := component.Content
	if content == "" {
		content = "{}"
	}

	source := component.Source
	if source == "" {
		source = "manual"
	}

	scope := component.Scope
	if scope == "" {
		scope = models.ScopeGlobal
	}

	reviewStatus := component.ReviewStatus
	if reviewStatus == "" {
		reviewStatus = models.ComponentReviewApproved
	}

	status := component.Status
	if status == "" {
		status = "active"
	}

	err := database.DB.QueryRow(
		ctx,
		query,
		component.EducationDomain,
		component.LibraryType,
		subject,
		component.GradeRange,
		tags,
		injectionMode,
		component.DisplayLabel,
		component.DesignLogic,
		component.ExampleSnippet,
		component.FullGuide,
		content,
		source,
		component.SourceRef,
		scope,
		component.ScopeRefID,
		component.CreatedBy,
		reviewStatus,
		status,
	).Scan(
		&component.ID,
		&component.EducationDomain,
		&component.CreatedAt,
		&component.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建教育域组件失败: %w", err)
	}

	return nil
}

// GetComponentByIDForEducationDomain 按可信当前域读取组件详情。
//
// currentDomain：
//   - k12/vocational/adult：同域或common；
//   - mixed：所有合法资源域；
//   - 其它值：查询不到，统一表现为组件不存在。
func GetComponentByIDForEducationDomain(
	ctx context.Context,
	id string,
	currentDomain string,
) (*models.LessonPlanComponent, error) {
	component := &models.LessonPlanComponent{}

	query := `
		SELECT
			c.id,
			c.education_domain,
			c.library_type,
			c.subject,
			COALESCE(c.grade_range, ''),
			c.tags,
			c.injection_mode,
			c.display_label,
			COALESCE(c.design_logic, ''),
			COALESCE(c.example_snippet, ''),
			COALESCE(c.full_guide, ''),
			c.content,
			c.source,
			COALESCE(c.source_ref, ''),
			c.quality_score,
			c.usage_count,
			c.select_count,
			c.like_count,
			c.dislike_count,
			c.scope,
			c.scope_ref_id,
			c.created_by,
			c.review_status,
			c.reviewed_by,
			c.reviewed_at,
			c.status,
			c.created_at,
			c.updated_at
		FROM lesson_plan_components c
		WHERE c.id = $1
		  AND (
			(
				$2 = 'mixed'
				AND c.education_domain IN (
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
					c.education_domain = $2
					OR c.education_domain = 'common'
				)
			)
		  )
	`

	err := database.DB.QueryRow(
		ctx,
		query,
		id,
		currentDomain,
	).Scan(
		&component.ID,
		&component.EducationDomain,
		&component.LibraryType,
		&component.Subject,
		&component.GradeRange,
		&component.Tags,
		&component.InjectionMode,
		&component.DisplayLabel,
		&component.DesignLogic,
		&component.ExampleSnippet,
		&component.FullGuide,
		&component.Content,
		&component.Source,
		&component.SourceRef,
		&component.QualityScore,
		&component.UsageCount,
		&component.SelectCount,
		&component.LikeCount,
		&component.DislikeCount,
		&component.Scope,
		&component.ScopeRefID,
		&component.CreatedBy,
		&component.ReviewStatus,
		&component.ReviewedBy,
		&component.ReviewedAt,
		&component.Status,
		&component.CreatedAt,
		&component.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrComponentNotFound
		}

		return nil, fmt.Errorf(
			"按教育域查询组件失败: %w",
			err,
		)
	}

	return component, nil
}

// GetComponentAccessRecordsByIDs 读取直接ID的最小校验快照。
//
// 本函数不会把组件正文返回给调用方。Service使用这些客观字段完成
// “存在、active、approved、阶段类型、教育域”五项整组验证。
func GetComponentAccessRecordsByIDs(
	ctx context.Context,
	componentIDs []string,
) ([]*ComponentAccessRecord, error) {
	if len(componentIDs) == 0 {
		return []*ComponentAccessRecord{}, nil
	}

	rows, err := database.DB.Query(
		ctx,
		`
			SELECT
				id,
				education_domain,
				library_type,
				status,
				review_status
			FROM lesson_plan_components
			WHERE id = ANY($1::uuid[])
		`,
		componentIDs,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询组件直接ID校验快照失败: %w",
			err,
		)
	}
	defer rows.Close()

	records := make(
		[]*ComponentAccessRecord,
		0,
		len(componentIDs),
	)

	for rows.Next() {
		record := &ComponentAccessRecord{}

		if err := rows.Scan(
			&record.ID,
			&record.EducationDomain,
			&record.LibraryType,
			&record.Status,
			&record.ReviewStatus,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描组件直接ID校验快照失败: %w",
				err,
			)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历组件直接ID校验快照失败: %w",
			err,
		)
	}

	return records, nil
}

// GetComponentContentsForEducationDomain 按教案教育域过滤历史组件ID。
//
// 适用于历史配方component_ids和历史阶段selected_component_ids：
//   - 同域或common正常返回；
//   - 不存在、停用、未审核、异域或非法域组件静默忽略；
//   - 不返回任何被过滤组件的标题、正文或其它可识别信息。
func GetComponentContentsForEducationDomain(
	ctx context.Context,
	componentIDs []string,
	lessonDomain string,
) ([]*models.MatchedComponentGroup, error) {
	if len(componentIDs) == 0 {
		return []*models.MatchedComponentGroup{}, nil
	}

	query := `
		SELECT
			c.library_type,
			c.id,
			c.education_domain,
			c.display_label,
			COALESCE(c.design_logic, ''),
			COALESCE(c.example_snippet, ''),
			COALESCE(c.full_guide, ''),
			c.quality_score,
			c.usage_count,
			c.select_count,
			COALESCE(c.tags::text, '[]'),
			COALESCE(c.component_index, '')
		FROM lesson_plan_components c
		WHERE c.id = ANY($1::uuid[])
		  AND c.status = 'active'
		  AND c.review_status = 'approved'
		  AND $2 IN (
			'k12',
			'vocational',
			'adult'
		  )
		  AND (
			c.education_domain = $2
			OR c.education_domain = 'common'
		  )
		ORDER BY
			c.library_type,
			c.quality_score DESC,
			c.select_count DESC
	`

	rows, err := database.DB.Query(
		ctx,
		query,
		componentIDs,
		lessonDomain,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"按教案教育域查询组件内容失败: %w",
			err,
		)
	}
	defer rows.Close()

	groupMap := make(
		map[string]*models.MatchedComponentGroup,
	)
	groupOrder := make([]string, 0)

	for rows.Next() {
		var libraryType string

		component := &models.MatchedComponent{}

		if err := rows.Scan(
			&libraryType,
			&component.ID,
			&component.EducationDomain,
			&component.DisplayLabel,
			&component.DesignLogic,
			&component.ExampleSnippet,
			&component.FullGuide,
			&component.QualityScore,
			&component.UsageCount,
			&component.SelectCount,
			&component.Tags,
			&component.ComponentIndex,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描教案教育域组件内容失败: %w",
				err,
			)
		}

		group, exists := groupMap[libraryType]
		if !exists {
			group = &models.MatchedComponentGroup{
				LibraryType: libraryType,
				LibraryName: models.LibraryTypeNameMap[libraryType],
				Components:  []*models.MatchedComponent{},
			}
			groupMap[libraryType] = group
			groupOrder = append(
				groupOrder,
				libraryType,
			)
		}

		group.Components = append(
			group.Components,
			component,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历教案教育域组件内容失败: %w",
			err,
		)
	}

	result := make(
		[]*models.MatchedComponentGroup,
		0,
		len(groupOrder),
	)

	for _, libraryType := range groupOrder {
		result = append(
			result,
			groupMap[libraryType],
		)
	}

	return result, nil
}
