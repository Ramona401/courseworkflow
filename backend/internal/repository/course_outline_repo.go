package repository

// course_outline_repo.go — 课程大纲数据访问
//
// 操作course_outlines表。content字段保存原文整块，本层不解析正文。
//
// 上下文16教育域收口：
//   1. ListCourseOutlines显式接收可信educationDomain，只返回同域资源；
//   2. K12可读取system全局资源，以及K12学校/教研组资源；
//   3. vocational/adult只读取本域group或school资源，不读取K12 system资源；
//   4. ListActiveOutlinesBySubjectAndEducationDomain供出版社列表、教案注入、
//      单元方案注入和上下文回执统一使用；
//   5. ResolveCourseOutlineScopeEducationDomain通过正式组织关系解析资源域；
//   6. 原ListActiveOutlinesBySubject暂时保留给尚未迁移的旧调用方，
//      后续批次会逐一替换，正式权限链不得新增对旧函数的调用。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrCourseOutlineNotFound 大纲不存在。
var ErrCourseOutlineNotFound = errors.New(
	"课程大纲不存在",
)

// ErrCourseOutlineScopeDomainUnavailable 表示资源归属无法解析为具体教学域。
var ErrCourseOutlineScopeDomainUnavailable = errors.New(
	"课程大纲归属教育域不可用",
)

// courseOutlineSelectColumns 单表、无别名查询统一列。
const courseOutlineSelectColumns = `id, scope, scope_target_id, subject, grade, volume, publisher, title,
content, COALESCE(source_file_path,''), source_type, created_by, status, created_at, updated_at`

// courseOutlineQualifiedSelectColumns
// 带co别名的查询统一列。
//
// 必须显式写出每一列，不能通过字符串替换自动加前缀；
// COALESCE等SQL表达式若被机械改写会形成co.COALESCE非法SQL。
const courseOutlineQualifiedSelectColumns = `co.id, co.scope, co.scope_target_id, co.subject, co.grade, co.volume, co.publisher, co.title,
co.content, COALESCE(co.source_file_path,''), co.source_type, co.created_by, co.status, co.created_at, co.updated_at`

// scanCourseOutline 统一扫描单条大纲。
func scanCourseOutline(
	row pgx.Row,
) (*models.CourseOutline, error) {
	outline := &models.CourseOutline{}

	err := row.Scan(
		&outline.ID,
		&outline.Scope,
		&outline.ScopeTargetID,
		&outline.Subject,
		&outline.Grade,
		&outline.Volume,
		&outline.Publisher,
		&outline.Title,
		&outline.Content,
		&outline.SourceFilePath,
		&outline.SourceType,
		&outline.CreatedBy,
		&outline.Status,
		&outline.CreatedAt,
		&outline.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCourseOutlineNotFound
		}

		return nil, fmt.Errorf(
			"扫描课程大纲失败: %w",
			err,
		)
	}

	return outline, nil
}

// CreateCourseOutline 创建课程大纲。
func CreateCourseOutline(
	ctx context.Context,
	outline *models.CourseOutline,
) error {
	sourceType := outline.SourceType
	if sourceType == "" {
		sourceType =
			models.CourseOutlineSourcePaste
	}

	err := database.DB.QueryRow(
		ctx,
		`
			INSERT INTO course_outlines
			  (
			    scope,
			    scope_target_id,
			    subject,
			    grade,
			    volume,
			    publisher,
			    title,
			    content,
			    source_file_path,
			    source_type,
			    created_by,
			    status
			  )
			VALUES (
			    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'active'
			)
			RETURNING id, created_at, updated_at
		`,
		outline.Scope,
		outline.ScopeTargetID,
		outline.Subject,
		outline.Grade,
		outline.Volume,
		outline.Publisher,
		outline.Title,
		outline.Content,
		nullIfEmptyStr(outline.SourceFilePath),
		sourceType,
		outline.CreatedBy,
	).Scan(
		&outline.ID,
		&outline.CreatedAt,
		&outline.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"创建课程大纲失败: %w",
			err,
		)
	}

	outline.Status =
		models.CourseOutlineStatusActive
	outline.SourceType = sourceType

	return nil
}

// GetCourseOutlineByID 按ID查询单条，包含archived记录。
func GetCourseOutlineByID(
	ctx context.Context,
	id string,
) (*models.CourseOutline, error) {
	query := `SELECT ` +
		courseOutlineSelectColumns +
		` FROM course_outlines WHERE id = $1`

	return scanCourseOutline(
		database.DB.QueryRow(
			ctx,
			query,
			id,
		),
	)
}

// ResolveCourseOutlineScopeEducationDomain
// 通过正式归属关系解析课程大纲或单元方案的具体教学教育域。
//
// 规则：
//   - system维持现有K12全局资源语义；
//   - school读取organizations.education_domain；
//   - group读取teaching_groups.school_id对应学校的education_domain；
//   - 归属不存在、未启用、组织类型不正确或教育域非法时返回错误。
func ResolveCourseOutlineScopeEducationDomain(
	ctx context.Context,
	scope string,
	targetID string,
) (string, error) {
	scope = strings.ToLower(
		strings.TrimSpace(scope),
	)
	targetID = strings.TrimSpace(targetID)

	switch scope {
	case models.CourseOutlineScopeSystem:
		if targetID != "" &&
			targetID !=
				models.CourseOutlineSystemTargetID {
			return "", fmt.Errorf(
				"%w: system归属ID非法",
				ErrCourseOutlineScopeDomainUnavailable,
			)
		}

		return models.EducationDomainK12, nil

	case models.CourseOutlineScopeSchool:
		var domain string

		err := database.DB.QueryRow(
			ctx,
			`
				SELECT LOWER(BTRIM(education_domain))
				FROM organizations
				WHERE id = $1
				  AND type = 'school'
				  AND status = 'active'
			`,
			targetID,
		).Scan(&domain)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", fmt.Errorf(
					"%w: 学校归属不存在或未启用",
					ErrCourseOutlineScopeDomainUnavailable,
				)
			}

			return "", fmt.Errorf(
				"查询学校归属教育域失败: %w",
				err,
			)
		}

		if !models.IsTeachingEducationDomain(
			domain,
		) {
			return "", fmt.Errorf(
				"%w: 学校教育域非法",
				ErrCourseOutlineScopeDomainUnavailable,
			)
		}

		return domain, nil

	case models.CourseOutlineScopeGroup:
		var domain string

		err := database.DB.QueryRow(
			ctx,
			`
				SELECT LOWER(BTRIM(org.education_domain))
				FROM teaching_groups tg
				JOIN organizations org
				  ON org.id = tg.school_id
				 AND org.type = 'school'
				 AND org.status = 'active'
				WHERE tg.id = $1
				  AND tg.status = 'active'
			`,
			targetID,
		).Scan(&domain)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", fmt.Errorf(
					"%w: 教研组归属不存在、未启用或未绑定有效学校",
					ErrCourseOutlineScopeDomainUnavailable,
				)
			}

			return "", fmt.Errorf(
				"查询教研组归属教育域失败: %w",
				err,
			)
		}

		if !models.IsTeachingEducationDomain(
			domain,
		) {
			return "", fmt.Errorf(
				"%w: 教研组学校教育域非法",
				ErrCourseOutlineScopeDomainUnavailable,
			)
		}

		return domain, nil

	default:
		return "", fmt.Errorf(
			"%w: 不支持的归属类型%s",
			ErrCourseOutlineScopeDomainUnavailable,
			scope,
		)
	}
}

// ListCourseOutlines 列出同教育域、且当前用户可见的active大纲。
//
// scopeIsAdmin只表示是否跳过group/school可见白名单，
// 不表示可以跳过educationDomain过滤。
func ListCourseOutlines(
	ctx context.Context,
	scopeIsAdmin bool,
	groupIDs []string,
	schoolIDs []string,
	educationDomain string,
) ([]*models.CourseOutlineListItem, error) {
	educationDomain = strings.ToLower(
		strings.TrimSpace(educationDomain),
	)
	if !models.IsTeachingEducationDomain(
		educationDomain,
	) {
		return []*models.CourseOutlineListItem{}, nil
	}

	query := `
		SELECT
		    co.id,
		    co.scope,
		    co.scope_target_id,
		    COALESCE(
		      CASE co.scope
		        WHEN 'group' THEN tg.name
		        WHEN 'school' THEN school_org.name
		        WHEN 'system' THEN '全局（所有K12学校通用）'
		      END,
		      ''
		    ) AS scope_name,
		    co.subject,
		    co.grade,
		    co.volume,
		    co.publisher,
		    co.title,
		    COALESCE(creator.display_name, '')
		      AS creator_name,
		    co.updated_at
		FROM course_outlines co
		LEFT JOIN teaching_groups tg
		  ON tg.id = co.scope_target_id
		 AND co.scope = 'group'
		LEFT JOIN organizations group_school
		  ON group_school.id = tg.school_id
		 AND co.scope = 'group'
		LEFT JOIN organizations school_org
		  ON school_org.id = co.scope_target_id
		 AND co.scope = 'school'
		LEFT JOIN users creator
		  ON creator.id = co.created_by
		WHERE co.status = 'active'
		  AND (
		       (
		         co.scope = 'system'
		         AND $1 = 'k12'
		       )
		    OR (
		         co.scope = 'group'
		         AND LOWER(
		               BTRIM(
		                 COALESCE(
		                   group_school.education_domain,
		                   ''
		                 )
		               )
		             ) = $1
		       )
		    OR (
		         co.scope = 'school'
		         AND LOWER(
		               BTRIM(
		                 COALESCE(
		                   school_org.education_domain,
		                   ''
		                 )
		               )
		             ) = $1
		       )
		  )`

	args := []interface{}{
		educationDomain,
	}

	if !scopeIsAdmin {
		args = append(
			args,
			groupIDs,
			schoolIDs,
		)

		query += `
		  AND (
		       co.scope = 'system'
		    OR (
		         co.scope = 'group'
		         AND co.scope_target_id =
		             ANY($2)
		       )
		    OR (
		         co.scope = 'school'
		         AND co.scope_target_id =
		             ANY($3)
		       )
		  )`
	}

	query += `
		ORDER BY co.updated_at DESC`

	rows, err := database.DB.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课程大纲列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items :=
		make([]*models.CourseOutlineListItem, 0)

	for rows.Next() {
		item := &models.CourseOutlineListItem{}

		if err := rows.Scan(
			&item.ID,
			&item.Scope,
			&item.ScopeTargetID,
			&item.ScopeName,
			&item.Subject,
			&item.Grade,
			&item.Volume,
			&item.Publisher,
			&item.Title,
			&item.CreatorName,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课程大纲列表行失败: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课程大纲列表失败: %w",
			err,
		)
	}

	return items, nil
}

// UpdateCourseOutline 全量更新课程大纲正文及检索标签。
func UpdateCourseOutline(
	ctx context.Context,
	id string,
	req *models.UpdateCourseOutlineRequest,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
			UPDATE course_outlines
			SET
			    subject = $1,
			    grade = $2,
			    volume = $3,
			    publisher = $4,
			    title = $5,
			    content = $6,
			    updated_at = NOW()
			WHERE id = $7
			  AND status = 'active'
		`,
		req.Subject,
		req.Grade,
		req.Volume,
		req.Publisher,
		req.Title,
		req.Content,
		id,
	)
	if err != nil {
		return fmt.Errorf(
			"更新课程大纲失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrCourseOutlineNotFound
	}

	return nil
}

// DeleteCourseOutline 软删除课程大纲。
func DeleteCourseOutline(
	ctx context.Context,
	id string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
			UPDATE course_outlines
			SET
			    status = 'archived',
			    updated_at = NOW()
			WHERE id = $1
			  AND status = 'active'
		`,
		id,
	)
	if err != nil {
		return fmt.Errorf(
			"删除课程大纲失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrCourseOutlineNotFound
	}

	return nil
}

// ListActiveOutlinesBySubjectAndEducationDomain
// 查询指定教育域、指定学科下的全部active大纲。
//
// K12可命中K12 group/school及system；
// vocational/adult只命中本域group/school，绝不读取system。
func ListActiveOutlinesBySubjectAndEducationDomain(
	ctx context.Context,
	subject string,
	educationDomain string,
) ([]*models.CourseOutline, error) {
	subject = strings.TrimSpace(subject)
	educationDomain = strings.ToLower(
		strings.TrimSpace(educationDomain),
	)

	if subject == "" ||
		!models.IsTeachingEducationDomain(
			educationDomain,
		) {
		return []*models.CourseOutline{}, nil
	}

	query := `SELECT ` +
		courseOutlineQualifiedSelectColumns +
		`
		FROM course_outlines co
		LEFT JOIN teaching_groups tg
		  ON tg.id = co.scope_target_id
		 AND co.scope = 'group'
		LEFT JOIN organizations group_school
		  ON group_school.id = tg.school_id
		 AND co.scope = 'group'
		LEFT JOIN organizations school_org
		  ON school_org.id = co.scope_target_id
		 AND co.scope = 'school'
		WHERE co.subject = $1
		  AND co.status = 'active'
		  AND (
		       (
		         co.scope = 'system'
		         AND $2 = 'k12'
		       )
		    OR (
		         co.scope = 'group'
		         AND LOWER(
		               BTRIM(
		                 COALESCE(
		                   group_school.education_domain,
		                   ''
		                 )
		               )
		             ) = $2
		       )
		    OR (
		         co.scope = 'school'
		         AND LOWER(
		               BTRIM(
		                 COALESCE(
		                   school_org.education_domain,
		                   ''
		                 )
		               )
		             ) = $2
		       )
		  )
		ORDER BY co.updated_at DESC`

	rows, err := database.DB.Query(
		ctx,
		query,
		subject,
		educationDomain,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"按教育域查询学科课程大纲失败: %w",
			err,
		)
	}
	defer rows.Close()

	outlines :=
		make([]*models.CourseOutline, 0)

	for rows.Next() {
		outline := &models.CourseOutline{}

		if err := rows.Scan(
			&outline.ID,
			&outline.Scope,
			&outline.ScopeTargetID,
			&outline.Subject,
			&outline.Grade,
			&outline.Volume,
			&outline.Publisher,
			&outline.Title,
			&outline.Content,
			&outline.SourceFilePath,
			&outline.SourceType,
			&outline.CreatedBy,
			&outline.Status,
			&outline.CreatedAt,
			&outline.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描分域课程大纲行失败: %w",
				err,
			)
		}

		outlines = append(
			outlines,
			outline,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历分域课程大纲失败: %w",
			err,
		)
	}

	return outlines, nil
}

// ListActiveOutlinesBySubject
// 旧的不分教育域查询，仅为尚未迁移的历史调用方保留。
//
// 新增代码禁止调用本函数。
// 上下文16后续批次会把单元方案、工坊注入和上下文回执全部迁到
// ListActiveOutlinesBySubjectAndEducationDomain。
func ListActiveOutlinesBySubject(
	ctx context.Context,
	subject string,
) ([]*models.CourseOutline, error) {
	query := `SELECT ` +
		courseOutlineSelectColumns +
		`
			FROM course_outlines
			WHERE subject = $1
			  AND status = 'active'
			ORDER BY updated_at DESC`

	rows, err := database.DB.Query(
		ctx,
		query,
		subject,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询学科大纲失败: %w",
			err,
		)
	}
	defer rows.Close()

	outlines :=
		make([]*models.CourseOutline, 0)

	for rows.Next() {
		outline := &models.CourseOutline{}

		if err := rows.Scan(
			&outline.ID,
			&outline.Scope,
			&outline.ScopeTargetID,
			&outline.Subject,
			&outline.Grade,
			&outline.Volume,
			&outline.Publisher,
			&outline.Title,
			&outline.Content,
			&outline.SourceFilePath,
			&outline.SourceType,
			&outline.CreatedBy,
			&outline.Status,
			&outline.CreatedAt,
			&outline.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描学科大纲行失败: %w",
				err,
			)
		}

		outlines = append(
			outlines,
			outline,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历学科大纲失败: %w",
			err,
		)
	}

	return outlines, nil
}
