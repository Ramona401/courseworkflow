package repository

// course_outline_exact_repo.go — 精确课程大纲读取与双模式候选查询
//
// 本文件承载唯一course_outline_id链路：
//   - 按ID和教育域读取唯一active大纲；
//   - 自动匹配候选要求同学科、同具体年级完全相等；
//   - 手动选择候选只在同学科、同教育域和可见范围内召回，
//     年级或学段相交过滤由Service层统一执行；
//   - group资源必须依附active、type=school的正式学校；
//   - 所有查询都显式读取school_system；
//   - 不使用出版社兜底、跨教育域匹配或前端身份字段扩权。

import (
        "context"
        "errors"
        "fmt"
        "strings"

        "github.com/jackc/pgx/v5"
        "tedna/internal/database"
        "tedna/internal/models"
)

const exactCourseOutlineSelectColumns = `
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

func scanExactCourseOutline(
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
                        "扫描精确课程大纲失败: %w",
                        err,
                )
        }

        return outline, nil
}

// GetActiveCourseOutlineByIDAndEducationDomain
// 按唯一ID读取同教育域的active课程大纲。
//
// 本函数只校验资源状态与教育域，不单独裁决用户可见范围；
// 用户可见性必须由Service在调用本函数前重新校验。
func GetActiveCourseOutlineByIDAndEducationDomain(
        ctx context.Context,
        id string,
        educationDomain string,
) (*models.CourseOutline, error) {
        id = strings.TrimSpace(id)
        educationDomain = strings.ToLower(
                strings.TrimSpace(educationDomain),
        )

        if id == "" ||
                !models.IsTeachingEducationDomain(
                        educationDomain,
                ) {
                return nil, ErrCourseOutlineNotFound
        }

        query := `SELECT ` +
                exactCourseOutlineSelectColumns +
                `
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
                WHERE co.id = $1
                  AND co.status = 'active'
                  AND (
                       (
                         co.scope = 'system'
                         AND co.scope_target_id = $2::uuid
                         AND $3 = 'k12'
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
                             ) = $3
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
                             ) = $3
                       )
                  )`

        return scanExactCourseOutline(
                database.DB.QueryRow(
                        ctx,
                        query,
                        id,
                        models.CourseOutlineSystemTargetID,
                        educationDomain,
                ),
        )
}

// listVisibleCourseOutlineCandidates
// 查询当前Actor可见的同学科课程大纲。
//
// exactGradeOnly=true：
//   数据库直接要求大纲grade与当前grade完全相等，用于自动匹配。
//
// exactGradeOnly=false：
//   数据库不按grade过滤，用于手动选择候选召回；
//   Service随后使用统一年级或学段相交规则做二次过滤，避免SQL和Go重复维护
//   两套复杂中文学段解析逻辑。
func listVisibleCourseOutlineCandidates(
        ctx context.Context,
        scopeIsAdmin bool,
        groupIDs []string,
        schoolIDs []string,
        educationDomain string,
        subject string,
        grade string,
        exactGradeOnly bool,
) ([]*models.CourseOutlineListItem, error) {
        educationDomain = strings.ToLower(
                strings.TrimSpace(educationDomain),
        )
        subject = strings.TrimSpace(subject)
        grade = strings.TrimSpace(grade)

        if !models.IsTeachingEducationDomain(
                educationDomain,
        ) ||
                subject == "" ||
                grade == "" {
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
                  AND BTRIM(co.subject) = $2
                  AND (
                       (
                         co.scope = 'system'
                         AND co.scope_target_id = $3::uuid
                         AND $1 = 'k12'
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
                subject,
                models.CourseOutlineSystemTargetID,
        }

        if exactGradeOnly {
                gradeArgIndex := len(args) + 1
                query += fmt.Sprintf(
                        `
                  AND BTRIM(co.grade) = $%d`,
                        gradeArgIndex,
                )
                args = append(
                        args,
                        grade,
                )
        }

        if !scopeIsAdmin {
                groupArgIndex := len(args) + 1
                schoolArgIndex := len(args) + 2

                query += fmt.Sprintf(
                        `
                  AND (
                       co.scope = 'system'
                    OR (
                         co.scope = 'group'
                         AND co.scope_target_id = ANY($%d)
                       )
                    OR (
                         co.scope = 'school'
                         AND co.scope_target_id = ANY($%d)
                       )
                  )`,
                        groupArgIndex,
                        schoolArgIndex,
                )

                args = append(
                        args,
                        groupIDs,
                        schoolIDs,
                )
        }

        query += `
                ORDER BY
                    co.publisher,
                    co.school_system,
                    co.volume,
                    co.grade,
                    co.title,
                    co.updated_at DESC`

        rows, err := database.DB.Query(
                ctx,
                query,
                args...,
        )
        if err != nil {
                if exactGradeOnly {
                        return nil, fmt.Errorf(
                                "查询自动匹配课程大纲候选失败: %w",
                                err,
                        )
                }

                return nil, fmt.Errorf(
                        "查询手动选择课程大纲候选失败: %w",
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
                                "扫描课程大纲候选失败: %w",
                                err,
                        )
                }

                items = append(
                        items,
                        item,
                )
        }

        if err := rows.Err(); err != nil {
                return nil, fmt.Errorf(
                        "遍历课程大纲候选失败: %w",
                        err,
                )
        }

        return items, nil
}

// ListVisibleExactCourseOutlineCandidates
// 返回当前用户可见的同学科、同具体年级自动匹配候选。
func ListVisibleExactCourseOutlineCandidates(
        ctx context.Context,
        scopeIsAdmin bool,
        groupIDs []string,
        schoolIDs []string,
        educationDomain string,
        subject string,
        grade string,
) ([]*models.CourseOutlineListItem, error) {
        return listVisibleCourseOutlineCandidates(
                ctx,
                scopeIsAdmin,
                groupIDs,
                schoolIDs,
                educationDomain,
                subject,
                grade,
                true,
        )
}

// ListVisibleManualCourseOutlineCandidates
// 召回当前用户可见的同学科手动选择候选。
//
// 本层不按grade过滤；Service必须继续执行年级或学段相交过滤，
// 不得把本函数结果直接返回浏览器。
func ListVisibleManualCourseOutlineCandidates(
        ctx context.Context,
        scopeIsAdmin bool,
        groupIDs []string,
        schoolIDs []string,
        educationDomain string,
        subject string,
        grade string,
) ([]*models.CourseOutlineListItem, error) {
        return listVisibleCourseOutlineCandidates(
                ctx,
                scopeIsAdmin,
                groupIDs,
                schoolIDs,
                educationDomain,
                subject,
                grade,
                false,
        )
}
