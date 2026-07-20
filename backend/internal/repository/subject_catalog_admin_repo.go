package repository

// subject_catalog_admin_repo.go — 后台课程目录归属数据访问
//
// 本文件只负责subject_catalog_entries的后台管理能力：
//   - 批量读取课程目录配置；
//   - 校验教育域与指定学校是否一致；
//   - 在事务中完整替换一门课程的目录配置。
//
// 与subject_repo.go拆分的原因：
//   课程定义CRUD和目录归属校验属于两个清晰模块，拆分后单文件保持在
//   600行以内，也便于未来独立扩展“教育域公共课程”和“学校专属课程”。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

/* ==================== 目录错误 ==================== */

var (
	ErrSubjectCatalogRequired = errors.New(
		"新建课程至少需要一条教育域或学校目录配置",
	)
	ErrSubjectCatalogInvalidDomain = errors.New(
		"课程目录教育域无效",
	)
	ErrSubjectCatalogDuplicate = errors.New(
		"同一教育域或学校存在重复课程目录配置",
	)
	ErrSubjectCatalogOrganizationNotFound = errors.New(
		"课程目录指定的学校不存在或未启用",
	)
	ErrSubjectCatalogOrganizationMismatch = errors.New(
		"课程目录指定学校的教育域与目录教育域不一致",
	)
)

/* ==================== 扫描与批量读取 ==================== */

// scanSubjectCatalogEntry 扫描一条带组织名称的课程目录记录。
func scanSubjectCatalogEntry(row interface {
	Scan(dest ...interface{}) error
}) (*models.SubjectCatalogEntry, error) {
	item := &models.SubjectCatalogEntry{}

	if err := row.Scan(
		&item.ID,
		&item.SubjectID,
		&item.EducationDomain,
		&item.OrganizationID,
		&item.OrganizationName,
		&item.DisplayName,
		&item.SortOrder,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return item, nil
}

// listAllSubjectCatalogEntries 按课程ID分组读取全部目录配置。
//
// 管理列表使用一次查询完成批量回填，避免逐课程N+1查询。
func listAllSubjectCatalogEntries(
	ctx context.Context,
) (map[string][]*models.SubjectCatalogEntry, error) {
	rows, err := database.DB.Query(
		ctx,
		`
		SELECT
			sce.id::text,
			sce.subject_id::text,
			sce.education_domain,
			CASE
				WHEN sce.organization_id IS NULL THEN NULL
				ELSE sce.organization_id::text
			END,
			CASE
				WHEN sce.organization_id IS NULL THEN '教育域公共课程'
				ELSE COALESCE(o.name, '')
			END,
			sce.display_name,
			sce.sort_order,
			sce.is_active,
			sce.created_at,
			sce.updated_at
		FROM subject_catalog_entries sce
		LEFT JOIN organizations o
		  ON o.id = sce.organization_id
		ORDER BY
			sce.subject_id,
			sce.education_domain,
			CASE WHEN sce.organization_id IS NULL THEN 0 ELSE 1 END,
			COALESCE(o.name, ''),
			sce.sort_order,
			sce.display_name
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("查询课程目录配置失败: %w", err)
	}
	defer rows.Close()

	itemsBySubjectID := make(
		map[string][]*models.SubjectCatalogEntry,
	)

	for rows.Next() {
		item, scanErr := scanSubjectCatalogEntry(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描课程目录配置失败: %w", scanErr)
		}

		itemsBySubjectID[item.SubjectID] = append(
			itemsBySubjectID[item.SubjectID],
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历课程目录配置失败: %w", err)
	}

	return itemsBySubjectID, nil
}

// listSubjectCatalogEntriesBySubjectTx 在当前事务中读取一门课程的目录配置。
func listSubjectCatalogEntriesBySubjectTx(
	ctx context.Context,
	tx pgx.Tx,
	subjectID string,
) ([]*models.SubjectCatalogEntry, error) {
	rows, err := tx.Query(
		ctx,
		`
		SELECT
			sce.id::text,
			sce.subject_id::text,
			sce.education_domain,
			CASE
				WHEN sce.organization_id IS NULL THEN NULL
				ELSE sce.organization_id::text
			END,
			CASE
				WHEN sce.organization_id IS NULL THEN '教育域公共课程'
				ELSE COALESCE(o.name, '')
			END,
			sce.display_name,
			sce.sort_order,
			sce.is_active,
			sce.created_at,
			sce.updated_at
		FROM subject_catalog_entries sce
		LEFT JOIN organizations o
		  ON o.id = sce.organization_id
		WHERE sce.subject_id = $1
		ORDER BY
			sce.education_domain,
			CASE WHEN sce.organization_id IS NULL THEN 0 ELSE 1 END,
			COALESCE(o.name, ''),
			sce.sort_order,
			sce.display_name
		`,
		subjectID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询课程目录配置失败: %w", err)
	}
	defer rows.Close()

	items := make([]*models.SubjectCatalogEntry, 0)

	for rows.Next() {
		item, scanErr := scanSubjectCatalogEntry(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("扫描课程目录配置失败: %w", scanErr)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历课程目录配置失败: %w", err)
	}

	return items, nil
}

/* ==================== 写入前规范化 ==================== */

// normalizedSubjectCatalogEntry 是写入前已完成清洗和学校校验的目录配置。
type normalizedSubjectCatalogEntry struct {
	EducationDomain  string
	OrganizationID   *string
	OrganizationName string
	DisplayName      string
	SortOrder        int
	IsActive         bool
}

// normalizeSubjectCatalogEntries 对管理端提交的目录配置执行最终校验。
//
// 规则：
//   - 只接受k12、vocational、adult三个具体教学域；
//   - organization_id为空表示教育域公共课程；
//   - 指定组织必须是启用学校，且学校教育域必须与目录教育域一致；
//   - 同一课程不能重复配置同一域公共目录或同一学校目录；
//   - 展示名和排序为空时回退课程名称与课程基础排序。
func normalizeSubjectCatalogEntries(
	ctx context.Context,
	tx pgx.Tx,
	subjectName string,
	subjectSortOrder int,
	requests []models.SubjectCatalogEntryRequest,
) ([]normalizedSubjectCatalogEntry, error) {
	items := make([]normalizedSubjectCatalogEntry, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))

	for _, request := range requests {
		domain := strings.ToLower(
			strings.TrimSpace(request.EducationDomain),
		)
		if !models.IsTeachingEducationDomain(domain) {
			return nil, ErrSubjectCatalogInvalidDomain
		}

		var organizationID *string
		organizationName := "教育域公共课程"

		if request.OrganizationID != nil {
			trimmedOrganizationID := strings.TrimSpace(
				*request.OrganizationID,
			)
			if trimmedOrganizationID != "" {
				organizationID = &trimmedOrganizationID
			}
		}

		duplicateKey := domain + "::public"
		if organizationID != nil {
			duplicateKey = domain + "::" + *organizationID
		}
		if _, exists := seen[duplicateKey]; exists {
			return nil, ErrSubjectCatalogDuplicate
		}
		seen[duplicateKey] = struct{}{}

		if organizationID != nil {
			var organizationType string
			var organizationDomain string

			err := tx.QueryRow(
				ctx,
				`
				SELECT
					name,
					type,
					COALESCE(education_domain, '')
				FROM organizations
				WHERE id = $1
				  AND status = 'active'
				`,
				*organizationID,
			).Scan(
				&organizationName,
				&organizationType,
				&organizationDomain,
			)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, ErrSubjectCatalogOrganizationNotFound
				}
				return nil, fmt.Errorf("查询课程目录学校失败: %w", err)
			}

			organizationDomain = strings.ToLower(
				strings.TrimSpace(organizationDomain),
			)
			if organizationType != "school" ||
				organizationDomain != domain {
				return nil, ErrSubjectCatalogOrganizationMismatch
			}
		}

		displayName := strings.TrimSpace(request.DisplayName)
		if displayName == "" {
			displayName = subjectName
		}

		sortOrder := request.SortOrder
		if sortOrder <= 0 {
			sortOrder = subjectSortOrder
		}
		if sortOrder < 0 {
			sortOrder = 0
		}

		items = append(items, normalizedSubjectCatalogEntry{
			EducationDomain:  domain,
			OrganizationID:   organizationID,
			OrganizationName: organizationName,
			DisplayName:      displayName,
			SortOrder:        sortOrder,
			IsActive:         request.IsActive,
		})
	}

	return items, nil
}

/* ==================== 事务完整替换 ==================== */

// replaceSubjectCatalogEntries 在当前事务中完整替换课程目录配置。
//
// 先完成全部规范化和学校校验，再删除旧配置并写入新配置；
// 任一步失败均由外层事务整体回滚，不会留下半套数据。
func replaceSubjectCatalogEntries(
	ctx context.Context,
	tx pgx.Tx,
	subjectID string,
	subjectName string,
	subjectSortOrder int,
	requests []models.SubjectCatalogEntryRequest,
) ([]*models.SubjectCatalogEntry, error) {
	normalized, err := normalizeSubjectCatalogEntries(
		ctx,
		tx,
		subjectName,
		subjectSortOrder,
		requests,
	)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM subject_catalog_entries WHERE subject_id = $1`,
		subjectID,
	); err != nil {
		return nil, fmt.Errorf("清理旧课程目录失败: %w", err)
	}

	entries := make([]*models.SubjectCatalogEntry, 0, len(normalized))

	for _, item := range normalized {
		row := tx.QueryRow(
			ctx,
			`
			INSERT INTO subject_catalog_entries (
				subject_id,
				education_domain,
				organization_id,
				display_name,
				sort_order,
				is_active
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING
				id::text,
				subject_id::text,
				education_domain,
				CASE
					WHEN organization_id IS NULL THEN NULL
					ELSE organization_id::text
				END,
				display_name,
				sort_order,
				is_active,
				created_at,
				updated_at
			`,
			subjectID,
			item.EducationDomain,
			item.OrganizationID,
			item.DisplayName,
			item.SortOrder,
			item.IsActive,
		)

		entry := &models.SubjectCatalogEntry{
			OrganizationName: item.OrganizationName,
		}
		if err := row.Scan(
			&entry.ID,
			&entry.SubjectID,
			&entry.EducationDomain,
			&entry.OrganizationID,
			&entry.DisplayName,
			&entry.SortOrder,
			&entry.IsActive,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		); err != nil {
			if IsUniqueViolation(err) {
				return nil, ErrSubjectCatalogDuplicate
			}
			return nil, fmt.Errorf("写入课程目录失败: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
