package repository

// course_outline_repo.go — 课程大纲数据访问（大单元备课能力·批次一 + 教材版本增强）
//
// 操作 course_outlines 表。pgx/v5 写法、错误处理风格对齐 curriculum_repo.go。
// content 字段存原文整块，本层不做任何解析。
// 列表查询 LEFT JOIN 回填归属名称（教研组名 / 学校名）与建立者显示名。
//
// publisher（教材版本）：空串=通用/不限版本；CRUD 全程读写本列，
// 注入匹配候选(ListActiveOutlinesBySubject)也带出本列，由 service 层按版本筛选。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrCourseOutlineNotFound 大纲不存在
var ErrCourseOutlineNotFound = errors.New("课程大纲不存在")

// courseOutlineSelectColumns 单条查询统一列（与 scanCourseOutline 对齐）
const courseOutlineSelectColumns = `id, scope, scope_target_id, subject, grade, volume, publisher, title,
content, COALESCE(source_file_path,''), source_type, created_by, status, created_at, updated_at`

// scanCourseOutline 统一扫描单条大纲
func scanCourseOutline(row pgx.Row) (*models.CourseOutline, error) {
	o := &models.CourseOutline{}
	err := row.Scan(
		&o.ID, &o.Scope, &o.ScopeTargetID, &o.Subject, &o.Grade, &o.Volume, &o.Publisher, &o.Title,
		&o.Content, &o.SourceFilePath, &o.SourceType, &o.CreatedBy, &o.Status,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCourseOutlineNotFound
		}
		return nil, fmt.Errorf("扫描课程大纲失败: %w", err)
	}
	return o, nil
}

// CreateCourseOutline 创建一份课程大纲（原文整块）
func CreateCourseOutline(ctx context.Context, o *models.CourseOutline) error {
	sourceType := o.SourceType
	if sourceType == "" {
		sourceType = models.CourseOutlineSourcePaste
	}
	err := database.DB.QueryRow(ctx, `
		INSERT INTO course_outlines
		  (scope, scope_target_id, subject, grade, volume, publisher, title, content,
		   source_file_path, source_type, created_by, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'active')
		RETURNING id, created_at, updated_at
	`,
		o.Scope, o.ScopeTargetID, o.Subject, o.Grade, o.Volume, o.Publisher, o.Title, o.Content,
		nullIfEmptyStr(o.SourceFilePath), sourceType, o.CreatedBy,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建课程大纲失败: %w", err)
	}
	o.Status = models.CourseOutlineStatusActive
	o.SourceType = sourceType
	return nil
}

// GetCourseOutlineByID 按 ID 查单条（含 archived，归属校验需要读出 scope）
func GetCourseOutlineByID(ctx context.Context, id string) (*models.CourseOutline, error) {
	sql := `SELECT ` + courseOutlineSelectColumns + ` FROM course_outlines WHERE id = $1`
	return scanCourseOutline(database.DB.QueryRow(ctx, sql, id))
}

// ListCourseOutlines 列出大纲（管理界面用）
//
// 三态白名单语义（对齐 data_scope）：
//
//	scopeIsAdmin=true            → 不过滤，列全部 active（含全局/学校/教研组）
//	否则                          → 全局(system)人人可见 + 自己组的 group + 本校的 school
//	groupIDs 与 schoolIDs 均为空  → 仍能看到全局(system)，但看不到任何组/校大纲
//
// 只返回 status='active'。LEFT JOIN 回填归属名称与建立者名。
func ListCourseOutlines(ctx context.Context, scopeIsAdmin bool, groupIDs []string, schoolIDs []string) ([]*models.CourseOutlineListItem, error) {
	// 归属名：group 取 teaching_groups.name，school 取 organizations.name，system 显示固定文案
	baseSQL := `
		SELECT co.id, co.scope, co.scope_target_id,
		       COALESCE(CASE co.scope
		                  WHEN 'group'  THEN tg.name
		                  WHEN 'school' THEN org.name
		                  WHEN 'system' THEN '全局（所有学校通用）'
		                END, '') AS scope_name,
		       co.subject, co.grade, co.volume, co.publisher, co.title,
		       COALESCE(u.display_name, '') AS creator_name,
		       co.updated_at
		FROM course_outlines co
		LEFT JOIN teaching_groups tg ON tg.id = co.scope_target_id AND co.scope = 'group'
		LEFT JOIN organizations  org ON org.id = co.scope_target_id AND co.scope = 'school'
		LEFT JOIN users u ON u.id = co.created_by
		WHERE co.status = 'active'`

	args := []interface{}{}
	if !scopeIsAdmin {
		// 非 admin：全局(system)恒可见；group/school 按白名单。两白名单都空时仅剩全局可见。
		args = append(args, groupIDs, schoolIDs)
		baseSQL += `
		  AND (
		        co.scope = 'system'
		     OR (co.scope = 'group'  AND co.scope_target_id = ANY($1))
		     OR (co.scope = 'school' AND co.scope_target_id = ANY($2))
		      )`
	}
	baseSQL += ` ORDER BY co.updated_at DESC`

	rows, err := database.DB.Query(ctx, baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("查询课程大纲列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CourseOutlineListItem
	for rows.Next() {
		it := &models.CourseOutlineListItem{}
		if err := rows.Scan(
			&it.ID, &it.Scope, &it.ScopeTargetID, &it.ScopeName,
			&it.Subject, &it.Grade, &it.Volume, &it.Publisher, &it.Title,
			&it.CreatorName, &it.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描课程大纲列表行失败: %w", err)
		}
		items = append(items, it)
	}
	if items == nil {
		items = []*models.CourseOutlineListItem{}
	}
	return items, nil
}

// UpdateCourseOutline 更新大纲内容（学科/年级/册次/版本/标题/正文全量更新）
func UpdateCourseOutline(ctx context.Context, id string, req *models.UpdateCourseOutlineRequest) error {
	result, err := database.DB.Exec(ctx, `
		UPDATE course_outlines
		SET subject = $1, grade = $2, volume = $3, publisher = $4, title = $5, content = $6, updated_at = now()
		WHERE id = $7 AND status = 'active'
	`, req.Subject, req.Grade, req.Volume, req.Publisher, req.Title, req.Content, id)
	if err != nil {
		return fmt.Errorf("更新课程大纲失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCourseOutlineNotFound
	}
	return nil
}

// DeleteCourseOutline 软删除（status → archived）
func DeleteCourseOutline(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE course_outlines SET status = 'archived', updated_at = now() WHERE id = $1 AND status = 'active'`,
		id)
	if err != nil {
		return fmt.Errorf("删除课程大纲失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCourseOutlineNotFound
	}
	return nil
}

// ==================== 备课注入用:按学科查同学科所有 active 大纲 ====================

// ListActiveOutlinesBySubject 查某学科下所有 active 大纲（供注入时打分匹配）
//
// 不在 SQL 里做年级/学段/版本过滤——年级写法五花八门（"小学低段"/"三年级"/"小学一至六年级"），
// SQL 等值匹配会漏；版本筛选也交给 service 层（空版本=通用要兜底命中）。
// 改为 SQL 只按学科粗筛，捞回同学科全部 active 大纲（含全局/学校/教研组），
// 交给 service 层的「学段归一化打分 + 版本筛选」函数挑最贴合的一份。
// 同学科大纲通常很少，全捞回内存打分开销可忽略。按 updated_at 倒序便于同分取最新。
func ListActiveOutlinesBySubject(ctx context.Context, subject string) ([]*models.CourseOutline, error) {
	sql := `SELECT ` + courseOutlineSelectColumns + `
		FROM course_outlines
		WHERE subject = $1 AND status = 'active'
		ORDER BY updated_at DESC`
	rows, err := database.DB.Query(ctx, sql, subject)
	if err != nil {
		return nil, fmt.Errorf("查询学科大纲失败: %w", err)
	}
	defer rows.Close()

	var outlines []*models.CourseOutline
	for rows.Next() {
		o := &models.CourseOutline{}
		if err := rows.Scan(
			&o.ID, &o.Scope, &o.ScopeTargetID, &o.Subject, &o.Grade, &o.Volume, &o.Publisher, &o.Title,
			&o.Content, &o.SourceFilePath, &o.SourceType, &o.CreatedBy, &o.Status,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描学科大纲行失败: %w", err)
		}
		outlines = append(outlines, o)
	}
	return outlines, nil
}
