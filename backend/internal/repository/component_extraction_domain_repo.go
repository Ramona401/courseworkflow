package repository

// component_extraction_domain_repo.go — 组件萃取教育域仓储。
//
// 本文件提供三组原子数据能力：
//
//  1. 组件与萃取记录在同一事务创建。
//  2. 待审队列按来源教案教育域过滤。
//  3. 萃取确认与组件审核状态在同一事务更新。
//
// 安全规则：
//   - 组件教育域必须由上层从lesson_plans.education_domain快照取得；
//   - 待审记录必须同时存在来源教案和目标组件；
//   - 来源教案域必须是k12、vocational或adult；
//   - 来源教案域必须与组件教育域完全一致；
//   - common不通过自动萃取审核队列生成；
//   - 异域、缺失关联和脏数据统一表现为ErrExtractionNotFound。

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

// ComponentExtractionQueueRecord 是待审队列查询需要的完整快照。
type ComponentExtractionQueueRecord struct {
	ID              string
	SourceType      string
	SourcePlanID    string
	SourceContent   string
	ComponentID     string
	ExtractionType  string
	Status          string
	CreatedBy       *string
	CreatedAt       time.Time
	EducationDomain string
	PlanTitle       string
	CreatedByName   string
}

// CreateExtractedComponentWithRecord 在同一事务中创建组件与萃取记录。
//
// 任一INSERT失败时整个事务回滚，避免孤立组件或孤立萃取记录。
func CreateExtractedComponentWithRecord(
	ctx context.Context,
	component *models.LessonPlanComponent,
	extraction *models.ComponentExtraction,
) error {
	transaction, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始组件萃取事务失败: %w",
			err,
		)
	}

	defer func() {
		_ = transaction.Rollback(ctx)
	}()

	subject := strings.TrimSpace(component.Subject)
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
		source = "ai_extracted"
	}

	scope := component.Scope
	if scope == "" {
		scope = models.ScopeGroup
	}

	reviewStatus := component.ReviewStatus
	if reviewStatus == "" {
		reviewStatus = models.ComponentReviewPending
	}

	status := component.Status
	if status == "" {
		status = "active"
	}

	var componentCreatedAt time.Time
	var componentUpdatedAt time.Time

	err = transaction.QueryRow(
		ctx,
		`
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
			RETURNING
				id,
				education_domain,
				created_at,
				updated_at
		`,
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
		&componentCreatedAt,
		&componentUpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"事务创建萃取组件失败: %w",
			err,
		)
	}

	component.CreatedAt = &componentCreatedAt
	component.UpdatedAt = &componentUpdatedAt

	extraction.ExtractedComponentID = &component.ID

	extractionStatus := extraction.Status
	if extractionStatus == "" {
		extractionStatus = "pending"
	}

	var extractionCreatedAt time.Time

	err = transaction.QueryRow(
		ctx,
		`
			INSERT INTO component_extractions (
				source_type,
				source_lesson_plan_id,
				source_content,
				extracted_component_id,
				extraction_type,
				status,
				created_by
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7
			)
			RETURNING
				id,
				created_at
		`,
		extraction.SourceType,
		extraction.SourceLessonPlanID,
		extraction.SourceContent,
		extraction.ExtractedComponentID,
		extraction.ExtractionType,
		extractionStatus,
		extraction.CreatedBy,
	).Scan(
		&extraction.ID,
		&extractionCreatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"事务创建组件萃取记录失败: %w",
			err,
		)
	}

	extraction.CreatedAt = &extractionCreatedAt

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交组件萃取事务失败: %w",
			err,
		)
	}

	return nil
}

// ListPendingExtractionsForEducationDomain 查询当前Actor域可见的待审萃取。
//
// currentDomain：
//   - k12/vocational/adult：只返回完全同域记录；
//   - mixed：返回三个具体教学域记录；
//   - 其它值：返回空集。
func ListPendingExtractionsForEducationDomain(
	ctx context.Context,
	currentDomain string,
	limit int,
) ([]*ComponentExtractionQueueRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := database.DB.Query(
		ctx,
		`
			SELECT
				ce.id,
				ce.source_type,
				ce.source_lesson_plan_id,
				COALESCE(ce.source_content, ''),
				ce.extracted_component_id,
				COALESCE(ce.extraction_type, ''),
				ce.status,
				ce.created_by,
				COALESCE(ce.created_at, now()),
				lp.education_domain,
				lp.title,
				COALESCE(u.display_name, '')
			FROM component_extractions ce
			JOIN lesson_plans lp
			  ON lp.id = ce.source_lesson_plan_id
			 AND lp.deleted_at IS NULL
			JOIN lesson_plan_components c
			  ON c.id = ce.extracted_component_id
			LEFT JOIN users u
			  ON u.id = ce.created_by
			WHERE ce.status = 'pending'
			  AND c.status = 'active'
			  AND c.review_status IN (
				'captured',
				'pending'
			  )
			  AND lp.education_domain IN (
				'k12',
				'vocational',
				'adult'
			  )
			  AND c.education_domain =
			      lp.education_domain
			  AND (
				$1 = 'mixed'
				OR (
					$1 IN (
						'k12',
						'vocational',
						'adult'
					)
					AND lp.education_domain = $1
				)
			  )
			ORDER BY ce.created_at DESC
			LIMIT $2
		`,
		currentDomain,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"按教育域查询待审萃取失败: %w",
			err,
		)
	}
	defer rows.Close()

	records := make(
		[]*ComponentExtractionQueueRecord,
		0,
	)

	for rows.Next() {
		record := &ComponentExtractionQueueRecord{}

		if err := rows.Scan(
			&record.ID,
			&record.SourceType,
			&record.SourcePlanID,
			&record.SourceContent,
			&record.ComponentID,
			&record.ExtractionType,
			&record.Status,
			&record.CreatedBy,
			&record.CreatedAt,
			&record.EducationDomain,
			&record.PlanTitle,
			&record.CreatedByName,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描教育域萃取队列失败: %w",
				err,
			)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历教育域萃取队列失败: %w",
			err,
		)
	}

	return records, nil
}

// ConfirmExtractionForEducationDomain 在同一事务中确认萃取并更新组件。
func ConfirmExtractionForEducationDomain(
	ctx context.Context,
	extractionID string,
	currentDomain string,
	confirmerID string,
	decision string,
) error {
	transaction, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始萃取确认事务失败: %w",
			err,
		)
	}

	defer func() {
		_ = transaction.Rollback(ctx)
	}()

	var storedStatus string
	var lessonDomain string
	var componentID string
	var componentDomain string
	var componentStatus string
	var componentReview string

	err = transaction.QueryRow(
		ctx,
		`
			SELECT
				ce.status,
				lp.education_domain,
				c.id,
				c.education_domain,
				c.status,
				c.review_status
			FROM component_extractions ce
			JOIN lesson_plans lp
			  ON lp.id = ce.source_lesson_plan_id
			 AND lp.deleted_at IS NULL
			JOIN lesson_plan_components c
			  ON c.id = ce.extracted_component_id
			WHERE ce.id = $1
			  AND (
				$2 = 'mixed'
				OR (
					$2 IN (
						'k12',
						'vocational',
						'adult'
					)
					AND lp.education_domain = $2
				)
			  )
			FOR UPDATE OF ce, c
		`,
		extractionID,
		currentDomain,
	).Scan(
		&storedStatus,
		&lessonDomain,
		&componentID,
		&componentDomain,
		&componentStatus,
		&componentReview,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrExtractionNotFound
		}

		return fmt.Errorf(
			"读取待确认萃取失败: %w",
			err,
		)
	}

	validLessonDomain :=
		lessonDomain == models.EducationDomainK12 ||
			lessonDomain == models.EducationDomainVocational ||
			lessonDomain == models.EducationDomainAdult

	validComponentReview :=
		componentReview == models.ComponentReviewCaptured ||
			componentReview == models.ComponentReviewPending

	validRecord :=
		storedStatus == "pending" &&
			validLessonDomain &&
			componentDomain == lessonDomain &&
			componentStatus == "active" &&
			validComponentReview

	if !validRecord {
		return ErrExtractionNotFound
	}

	componentDecision := models.ComponentReviewApproved

	if decision == "rejected" {
		componentDecision = models.ComponentReviewRejected
	}

	now := time.Now()

	extractionResult, err := transaction.Exec(
		ctx,
		`
			UPDATE component_extractions
			SET
				status = $1,
				confirmed_by = $2,
				confirmed_at = $3
			WHERE id = $4
			  AND status = 'pending'
		`,
		decision,
		confirmerID,
		now,
		extractionID,
	)
	if err != nil {
		return fmt.Errorf(
			"事务更新萃取状态失败: %w",
			err,
		)
	}

	if extractionResult.RowsAffected() != 1 {
		return ErrExtractionNotFound
	}

	componentResult, err := transaction.Exec(
		ctx,
		`
			UPDATE lesson_plan_components
			SET
				review_status = $1,
				reviewed_by = $2,
				reviewed_at = $3,
				updated_at = $3
			WHERE id = $4
			  AND education_domain = $5
			  AND status = 'active'
			  AND review_status IN (
				'captured',
				'pending'
			  )
		`,
		componentDecision,
		confirmerID,
		now,
		componentID,
		lessonDomain,
	)
	if err != nil {
		return fmt.Errorf(
			"事务更新萃取组件审核状态失败: %w",
			err,
		)
	}

	if componentResult.RowsAffected() != 1 {
		return ErrExtractionNotFound
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交萃取确认事务失败: %w",
			err,
		)
	}

	return nil
}
