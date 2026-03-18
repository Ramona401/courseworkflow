package repository

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrCourseNotFound      = errors.New("课程不存在")
	ErrCourseCodeExists    = errors.New("课程编号已存在")
	ErrCourseIndexNotFound = errors.New("课程索引不存在")
)

// ==================== 课程 CRUD ====================

// ListCourses 获取所有课程列表（含索引摘要信息）
// 通过LEFT JOIN course_indexes获取索引状态
func ListCourses() ([]*models.CourseListItem, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT c.id, c.course_code, c.course_name, c.external_module_id,
		        c.grade_num, c.stage, c.semester, c.status, c.created_at, c.updated_at,
		        ci.page_count, ci.total_length, ci.fetched_at
		 FROM courses c
		 LEFT JOIN course_indexes ci ON ci.course_id = c.id
		 ORDER BY c.course_code ASC`)
	if err != nil {
		return nil, fmt.Errorf("查询课程列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CourseListItem
	for rows.Next() {
		item := &models.CourseListItem{}
		var pageCount, totalLength *int
		var fetchedAt *interface{}
		// 使用临时变量接收可能为NULL的索引字段
		var pgCnt, ttLen *int
		var ftAt interface{}

		err := rows.Scan(
			&item.ID, &item.CourseCode, &item.CourseName, &item.ExternalModuleID,
			&item.GradeNum, &item.Stage, &item.Semester, &item.Status,
			&item.CreatedAt, &item.UpdatedAt,
			&pgCnt, &ttLen, &ftAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描课程行失败: %w", err)
		}

		// 处理索引字段（LEFT JOIN可能为NULL）
		if pgCnt != nil {
			item.HasIndex = true
			item.IndexPageCount = *pgCnt
		}
		if ttLen != nil {
			item.IndexTotalLength = *ttLen
		}
		// fetched_at需要特殊处理：pgx返回time.Time或nil
		_ = pageCount
		_ = totalLength
		_ = fetchedAt

		items = append(items, item)
	}
	return items, nil
}

// ListCoursesForUser 获取指定用户可见的课程列表
// admin看所有课程，非admin只看分配给自己的
func ListCoursesForUser(userID string, role string) ([]*models.CourseListItem, error) {
	ctx := context.Background()
	var query string
	var args []interface{}

	if role == "admin" {
		// admin可以看所有课程
		query = `SELECT c.id, c.course_code, c.course_name, c.external_module_id,
		                c.grade_num, c.stage, c.semester, c.status, c.created_at, c.updated_at,
		                ci.page_count, ci.total_length, ci.fetched_at
		         FROM courses c
		         LEFT JOIN course_indexes ci ON ci.course_id = c.id
		         ORDER BY c.course_code ASC`
	} else {
		// 非admin只能看分配给自己的课程
		query = `SELECT c.id, c.course_code, c.course_name, c.external_module_id,
		                c.grade_num, c.stage, c.semester, c.status, c.created_at, c.updated_at,
		                ci.page_count, ci.total_length, ci.fetched_at
		         FROM courses c
		         INNER JOIN user_course_assignments uca ON uca.course_code = c.course_code AND uca.user_id = $1
		         LEFT JOIN course_indexes ci ON ci.course_id = c.id
		         ORDER BY c.course_code ASC`
		args = append(args, userID)
	}

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询用户课程列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.CourseListItem
	for rows.Next() {
		item := &models.CourseListItem{}
		var pgCnt, ttLen *int
		var ftAt interface{}

		err := rows.Scan(
			&item.ID, &item.CourseCode, &item.CourseName, &item.ExternalModuleID,
			&item.GradeNum, &item.Stage, &item.Semester, &item.Status,
			&item.CreatedAt, &item.UpdatedAt,
			&pgCnt, &ttLen, &ftAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描课程行失败: %w", err)
		}

		if pgCnt != nil {
			item.HasIndex = true
			item.IndexPageCount = *pgCnt
		}
		if ttLen != nil {
			item.IndexTotalLength = *ttLen
		}

		items = append(items, item)
	}
	return items, nil
}

// GetCourseByCode 根据课程编号查询课程
func GetCourseByCode(code string) (*models.Course, error) {
	ctx := context.Background()
	c := &models.Course{}
	err := database.DB.QueryRow(ctx,
		`SELECT id, course_code, course_name, external_module_id,
		        grade_num, stage, semester, status, created_at, updated_at
		 FROM courses WHERE course_code = $1`, code).Scan(
		&c.ID, &c.CourseCode, &c.CourseName, &c.ExternalModuleID,
		&c.GradeNum, &c.Stage, &c.Semester, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	return c, nil
}

// GetCourseByModuleID 根据外部模块ID查询课程
func GetCourseByModuleID(moduleID int) (*models.Course, error) {
	ctx := context.Background()
	c := &models.Course{}
	err := database.DB.QueryRow(ctx,
		`SELECT id, course_code, course_name, external_module_id,
		        grade_num, stage, semester, status, created_at, updated_at
		 FROM courses WHERE external_module_id = $1`, moduleID).Scan(
		&c.ID, &c.CourseCode, &c.CourseName, &c.ExternalModuleID,
		&c.GradeNum, &c.Stage, &c.Semester, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	return c, nil
}

// CheckCourseCodeExists 检查课程编号是否已存在
func CheckCourseCodeExists(code string) (bool, error) {
	ctx := context.Background()
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM courses WHERE course_code = $1`, code).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查课程编号失败: %w", err)
	}
	return count > 0, nil
}

// CreateCourse 创建课程记录
func CreateCourse(c *models.Course) error {
	ctx := context.Background()
	err := database.DB.QueryRow(ctx,
		`INSERT INTO courses (course_code, course_name, external_module_id, grade_num, stage, semester, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		c.CourseCode, c.CourseName, c.ExternalModuleID, c.GradeNum, c.Stage, c.Semester, c.Status,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建课程失败: %w", err)
	}
	return nil
}

// GetAllRegisteredModuleIDs 获取所有已注册课程的外部模块ID列表
// 用于OSS目录展示时标记哪些模块已注册
func GetAllRegisteredModuleIDs() (map[int]string, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT external_module_id, course_code FROM courses WHERE external_module_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("查询已注册模块ID失败: %w", err)
	}
	defer rows.Close()

	result := make(map[int]string)
	for rows.Next() {
		var moduleID int
		var courseCode string
		if err := rows.Scan(&moduleID, &courseCode); err != nil {
			return nil, fmt.Errorf("扫描模块ID行失败: %w", err)
		}
		result[moduleID] = courseCode
	}
	return result, nil
}

// ==================== 课程索引 CRUD ====================

// GetCourseIndex 获取课程索引（完整内容）
func GetCourseIndex(courseID string) (*models.CourseIndex, error) {
	ctx := context.Background()
	idx := &models.CourseIndex{}
	err := database.DB.QueryRow(ctx,
		`SELECT id, course_id, index_content, index_hash, page_count, total_length, fetched_at
		 FROM course_indexes WHERE course_id = $1`, courseID).Scan(
		&idx.ID, &idx.CourseID, &idx.IndexContent, &idx.IndexHash,
		&idx.PageCount, &idx.TotalLength, &idx.FetchedAt)
	if err != nil {
		return nil, ErrCourseIndexNotFound
	}
	return idx, nil
}

// UpsertCourseIndex 插入或更新课程索引
// 如果已存在则更新内容，不存在则插入
func UpsertCourseIndex(idx *models.CourseIndex) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`INSERT INTO course_indexes (course_id, index_content, index_hash, page_count, total_length, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (course_id) DO UPDATE SET
		   index_content = EXCLUDED.index_content,
		   index_hash = EXCLUDED.index_hash,
		   page_count = EXCLUDED.page_count,
		   total_length = EXCLUDED.total_length,
		   fetched_at = NOW()`,
		idx.CourseID, idx.IndexContent, idx.IndexHash, idx.PageCount, idx.TotalLength)
	if err != nil {
		return fmt.Errorf("保存课程索引失败: %w", err)
	}
	return nil
}

// DeleteCourseIndex 删除课程索引
func DeleteCourseIndex(courseID string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`DELETE FROM course_indexes WHERE course_id = $1`, courseID)
	if err != nil {
		return fmt.Errorf("删除课程索引失败: %w", err)
	}
	return nil
}
