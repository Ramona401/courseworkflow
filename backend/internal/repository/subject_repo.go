package repository

// subject_repo.go — 学科字典（subjects 表）数据访问层
//
// 对应表：subjects（id / name唯一 / code / sort_order / is_active / is_system / note / updated_by / 时间戳）
// 风格对齐 kb_authorized_repo.go 与 model_alias_repo.go：database.DB、ctx、fmt.Errorf("...: %w")。
//
// 语义：
//   - ListActiveSubjects  公开只读（GET /api/v1/subjects）：只返 is_active=true，按 sort_order 排。
//   - ListAllSubjects     admin 管理用：返全部（含停用），按 sort_order 排。
//   - Create/Update/Delete admin 管理用。is_system=true 的学科禁止删除（在 service 层拦截，repo 层也兜一道）。

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// 学科相关错误常量
var (
	ErrSubjectNotFound    = errors.New("学科不存在")
	ErrSubjectNameExists  = errors.New("学科名已存在")
	ErrSubjectSystemGuard = errors.New("内置学科不可删除")
)

// subjectSelectColumns 统一列顺序，供扫描复用
const subjectSelectColumns = `id, name, code, sort_order, is_active, is_system, COALESCE(note,''), updated_by, created_at, updated_at`

// scanSubject 统一扫描一行
func scanSubject(rows interface {
	Scan(dest ...interface{}) error
}) (*models.Subject, error) {
	s := &models.Subject{}
	if err := rows.Scan(
		&s.ID, &s.Name, &s.Code, &s.SortOrder, &s.IsActive, &s.IsSystem,
		&s.Note, &s.UpdatedBy, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return s, nil
}

// ListActiveSubjects 公开只读：仅启用学科，按 sort_order 升序
func ListActiveSubjects(ctx context.Context) ([]*models.Subject, error) {
	query := `SELECT ` + subjectSelectColumns + ` FROM subjects WHERE is_active = true ORDER BY sort_order ASC, name ASC`
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询启用学科列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.Subject{}
	for rows.Next() {
		s, err := scanSubject(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描学科行失败: %w", err)
		}
		items = append(items, s)
	}
	return items, nil
}

// ListAllSubjects admin 管理用：返回全部（含停用），按 sort_order 升序
func ListAllSubjects(ctx context.Context) ([]*models.Subject, error) {
	query := `SELECT ` + subjectSelectColumns + ` FROM subjects ORDER BY sort_order ASC, name ASC`
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询全部学科列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.Subject{}
	for rows.Next() {
		s, err := scanSubject(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描学科行失败: %w", err)
		}
		items = append(items, s)
	}
	return items, nil
}

// GetSubjectByID 按 ID 查单个学科
func GetSubjectByID(ctx context.Context, id string) (*models.Subject, error) {
	query := `SELECT ` + subjectSelectColumns + ` FROM subjects WHERE id = $1`
	row := database.DB.QueryRow(ctx, query, id)
	s, err := scanSubject(row)
	if err != nil {
		// pgx 查不到返回 no rows；统一翻译为 ErrSubjectNotFound
		return nil, ErrSubjectNotFound
	}
	return s, nil
}

// CreateSubject 新建学科。撞唯一名返回 ErrSubjectNameExists。
func CreateSubject(ctx context.Context, req *models.CreateSubjectRequest, updatedBy string) (*models.Subject, error) {
	sortOrder := req.SortOrder
	if sortOrder <= 0 {
		sortOrder = 100 // 默认排序值
	}
	var updatedByArg interface{}
	if updatedBy == "" {
		updatedByArg = nil
	} else {
		updatedByArg = updatedBy
	}

	query := `
		INSERT INTO subjects (name, code, sort_order, note, is_active, is_system, updated_by)
		VALUES ($1, $2, $3, $4, true, false, $5)
		RETURNING ` + subjectSelectColumns
	row := database.DB.QueryRow(ctx, query, req.Name, req.Code, sortOrder, req.Note, updatedByArg)
	s, err := scanSubject(row)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrSubjectNameExists
		}
		return nil, fmt.Errorf("新建学科失败: %w", err)
	}
	return s, nil
}

// UpdateSubject 编辑学科（部分更新，指针字段 nil=不改）。撞唯一名返回 ErrSubjectNameExists。
func UpdateSubject(ctx context.Context, id string, req *models.UpdateSubjectRequest, updatedBy string) (*models.Subject, error) {
	// 先取现值（同时校验存在）
	cur, err := GetSubjectByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 逐字段回填现值（nil=不改）
	name := cur.Name
	if req.Name != nil {
		name = *req.Name
	}
	code := cur.Code
	if req.Code != nil {
		code = *req.Code
	}
	sortOrder := cur.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	isActive := cur.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	note := cur.Note
	if req.Note != nil {
		note = *req.Note
	}

	var updatedByArg interface{}
	if updatedBy == "" {
		updatedByArg = nil
	} else {
		updatedByArg = updatedBy
	}

	query := `
		UPDATE subjects
		SET name = $1, code = $2, sort_order = $3, is_active = $4, note = $5,
		    updated_by = $6, updated_at = now()
		WHERE id = $7
		RETURNING ` + subjectSelectColumns
	row := database.DB.QueryRow(ctx, query, name, code, sortOrder, isActive, note, updatedByArg, id)
	s, err := scanSubject(row)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrSubjectNameExists
		}
		return nil, fmt.Errorf("编辑学科失败: %w", err)
	}
	return s, nil
}

// DeleteSubject 删除学科。is_system=true 的内置学科禁止删除（返回 ErrSubjectSystemGuard）。
func DeleteSubject(ctx context.Context, id string) error {
	cur, err := GetSubjectByID(ctx, id)
	if err != nil {
		return err
	}
	if cur.IsSystem {
		return ErrSubjectSystemGuard
	}
	result, err := database.DB.Exec(ctx, `DELETE FROM subjects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除学科失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSubjectNotFound
	}
	return nil
}
