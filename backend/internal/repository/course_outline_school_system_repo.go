package repository

// course_outline_school_system_repo.go — 课程大纲学制感知管理仓储
//
// 存量course_outline_repo.go暂时保留给旧调用方。
// 精确课程大纲管理链统一使用本文件，确保列表、详情、创建和更新
// 都显式读取或写入school_system，不依赖数据库默认值猜测。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

const courseOutlineSchoolSystemSelectColumns = `
	co.id,
	co.scope,
	co.scope_target_id,
	co.subject,
	co.grade,
	co.volume,
	co.publisher,
	co.school_system,
	co.title,
	co.content,
	COALESCE(co.source_file_path, ''),
	co.source_type,
	co.created_by,
	co.status,
	co.created_at,
	co.updated_at
`

func scanCourseOutlineWithSchoolSystem(
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
		&outline.SchoolSystem,
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
			"扫描课程大纲学制记录失败: %w",
			err,
		)
	}

	return outline, nil
}

// GetCourseOutlineByIDWithSchoolSystem 按ID读取完整大纲，包含archived记录。
func GetCourseOutlineByIDWithSchoolSystem(
	ctx context.Context,
	id string,
) (*models.CourseOutline, error) {
	query := `SELECT ` +
		courseOutlineSchoolSystemSelectColumns +
		`
		FROM course_outlines co
		WHERE co.id = $1`

	return scanCourseOutlineWithSchoolSystem(
		database.DB.QueryRow(
			ctx,
			query,
			strings.TrimSpace(id),
		),
	)
}

// ListCourseOutlinesWithSchoolSystem 列出同域、当前用户可见的active大纲。
func ListCourseOutlinesWithSchoolSystem(
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
		    co.school_system,
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
		 AND group_school.type = 'school'
		 AND group_school.status = 'active'
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
		         AND co.scope_target_id = $2::uuid
		       )
		    OR (
		         co.scope = 'group'
		         AND tg.status = 'active'
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
		         AND school_org.type = 'school'
		         AND school_org.status = 'active'
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
		models.CourseOutlineSystemTargetID,
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
		         AND co.scope_target_id = ANY($3)
		       )
		    OR (
		         co.scope = 'school'
		         AND co.scope_target_id = ANY($4)
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
			"查询学制感知课程大纲列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CourseOutlineListItem,
		0,
	)

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
			&item.SchoolSystem,
			&item.Title,
			&item.CreatorName,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描学制感知课程大纲列表失败: %w",
				err,
			)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历学制感知课程大纲列表失败: %w",
			err,
		)
	}

	return items, nil
}

// CreateCourseOutlineWithSchoolSystem 创建学制明确的课程大纲。
func CreateCourseOutlineWithSchoolSystem(
	ctx context.Context,
	outline *models.CourseOutline,
) error {
	if outline == nil {
		return errors.New("课程大纲对象不能为空")
	}

	sourceType := strings.TrimSpace(
		outline.SourceType,
	)
	if sourceType == "" {
		sourceType = models.CourseOutlineSourcePaste
	}

	err := database.DB.QueryRow(
		ctx,
		`
			INSERT INTO course_outlines (
				scope,
				scope_target_id,
				subject,
				grade,
				volume,
				publisher,
				school_system,
				title,
				content,
				source_file_path,
				source_type,
				created_by,
				status
			)
			VALUES (
				$1,$2,$3,$4,$5,$6,$7,
				$8,$9,$10,$11,$12,'active'
			)
			RETURNING
				id,
				school_system,
				created_at,
				updated_at
		`,
		outline.Scope,
		outline.ScopeTargetID,
		outline.Subject,
		outline.Grade,
		outline.Volume,
		outline.Publisher,
		outline.SchoolSystem,
		outline.Title,
		outline.Content,
		nullIfEmptyStr(outline.SourceFilePath),
		sourceType,
		outline.CreatedBy,
	).Scan(
		&outline.ID,
		&outline.SchoolSystem,
		&outline.CreatedAt,
		&outline.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"创建学制感知课程大纲失败: %w",
			err,
		)
	}

	outline.Status = models.CourseOutlineStatusActive
	outline.SourceType = sourceType
	return nil
}

// UpdateCourseOutlineWithSchoolSystem 更新大纲检索标签、学制和正文。
func UpdateCourseOutlineWithSchoolSystem(
	ctx context.Context,
	id string,
	req *models.UpdateCourseOutlineRequest,
) error {
	if req == nil {
		return errors.New("课程大纲更新请求不能为空")
	}

	result, err := database.DB.Exec(
		ctx,
		`
			UPDATE course_outlines
			SET
				subject = $1,
				grade = $2,
				volume = $3,
				publisher = $4,
				school_system = $5,
				title = $6,
				content = $7,
				updated_at = NOW()
			WHERE id = $8
			  AND status = 'active'
		`,
		req.Subject,
		req.Grade,
		req.Volume,
		req.Publisher,
		req.SchoolSystem,
		req.Title,
		req.Content,
		strings.TrimSpace(id),
	)
	if err != nil {
		return fmt.Errorf(
			"更新学制感知课程大纲失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return ErrCourseOutlineNotFound
	}

	return nil
}
