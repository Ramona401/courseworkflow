package repository

// subject_repo.go — 统一课程定义数据访问层
//
// 对应数据表：
//   - subjects：全平台统一课程定义；
//   - subject_catalog_entries：课程在教育域和学校中的可见目录。
//
// 管理端行为：
//   - 列表同时返回课程定义及其全部目录配置；
//   - 新建课程必须在同一事务中写入至少一条目录配置；
//   - 编辑课程可以只更新课程定义，也可以完整替换目录配置；
//   - 任一步失败时整体回滚，禁止产生只有课程定义、教师却看不到的孤立记录。
//
// 公开课程目录查询仍由education_domain_repo.go负责，
// 本文件不改变普通教师按教育域和学校读取课程的既有规则。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

/* ==================== 课程定义错误 ==================== */

var (
	ErrSubjectNotFound = errors.New(
		"课程不存在",
	)
	ErrSubjectNameExists = errors.New(
		"课程名称已存在",
	)
	ErrSubjectSystemGuard = errors.New(
		"内置课程不可删除",
	)
)

/* ==================== 通用扫描 ==================== */

// subjectSelectColumns 统一subjects字段顺序。
//
// 查询和RETURNING必须复用该顺序，避免新增字段后不同函数扫描错位。
const subjectSelectColumns = `
	id,
	name,
	code,
	sort_order,
	is_active,
	is_system,
	COALESCE(note, ''),
	updated_by,
	created_at,
	updated_at
`

// subjectRowScanner 描述pgx.Row和pgx.Rows共同具备的扫描能力。
type subjectRowScanner interface {
	Scan(dest ...interface{}) error
}

// scanSubject 扫描一条统一课程定义。
func scanSubject(
	row subjectRowScanner,
) (*models.Subject, error) {
	item := &models.Subject{}

	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Code,
		&item.SortOrder,
		&item.IsActive,
		&item.IsSystem,
		&item.Note,
		&item.UpdatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return item, nil
}

// subjectUpdatedByArg 将空操作人转换为数据库NULL。
func subjectUpdatedByArg(
	updatedBy string,
) interface{} {
	if updatedBy == "" {
		return nil
	}

	return updatedBy
}

/* ==================== 公开与管理列表 ==================== */

// ListActiveSubjects 返回全部启用课程定义。
//
// 该函数保留给仍然只需要统一定义的内部调用点。
// 普通教师正式课程下拉应使用分域课程目录查询。
func ListActiveSubjects(
	ctx context.Context,
) ([]*models.Subject, error) {
	rows, err := database.DB.Query(
		ctx,
		`
		SELECT `+subjectSelectColumns+`
		FROM subjects
		WHERE is_active = true
		ORDER BY sort_order ASC, name ASC
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询启用课程列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.Subject, 0)

	for rows.Next() {
		item, scanErr := scanSubject(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课程定义失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历启用课程列表失败: %w",
			err,
		)
	}

	return items, nil
}

// ListAllSubjects 返回后台课程管理列表。
//
// 每条记录都包含catalog_entries，前端可以直接展示、编辑教育域和适用学校。
// 课程定义和目录配置分别使用两次批量查询，不执行逐课程N+1查询。
func ListAllSubjects(
	ctx context.Context,
) ([]*models.SubjectAdminItem, error) {
	rows, err := database.DB.Query(
		ctx,
		`
		SELECT `+subjectSelectColumns+`
		FROM subjects
		ORDER BY sort_order ASC, name ASC
		`,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询全部课程列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	subjects := make([]*models.Subject, 0)

	for rows.Next() {
		item, scanErr := scanSubject(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课程定义失败: %w",
				scanErr,
			)
		}

		subjects = append(subjects, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历全部课程列表失败: %w",
			err,
		)
	}

	catalogsBySubjectID, err :=
		listAllSubjectCatalogEntries(ctx)
	if err != nil {
		return nil, err
	}

	items := make(
		[]*models.SubjectAdminItem,
		0,
		len(subjects),
	)

	for _, subject := range subjects {
		catalogEntries :=
			catalogsBySubjectID[subject.ID]

		if catalogEntries == nil {
			catalogEntries =
				[]*models.SubjectCatalogEntry{}
		}

		items = append(
			items,
			&models.SubjectAdminItem{
				Subject:        *subject,
				CatalogEntries: catalogEntries,
			},
		)
	}

	return items, nil
}

/* ==================== 单条读取 ==================== */

// GetSubjectByID 从数据库连接池读取一门课程定义。
func GetSubjectByID(
	ctx context.Context,
	id string,
) (*models.Subject, error) {
	row := database.DB.QueryRow(
		ctx,
		`
		SELECT `+subjectSelectColumns+`
		FROM subjects
		WHERE id = $1
		`,
		id,
	)

	item, err := scanSubject(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubjectNotFound
		}

		return nil, fmt.Errorf(
			"查询课程定义失败: %w",
			err,
		)
	}

	return item, nil
}

// getSubjectByIDTx 在当前事务中读取并锁定课程定义。
//
// 编辑期间使用FOR UPDATE，避免两个管理员同时保存时相互覆盖。
func getSubjectByIDTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (*models.Subject, error) {
	row := tx.QueryRow(
		ctx,
		`
		SELECT `+subjectSelectColumns+`
		FROM subjects
		WHERE id = $1
		FOR UPDATE
		`,
		id,
	)

	item, err := scanSubject(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubjectNotFound
		}

		return nil, fmt.Errorf(
			"查询课程定义失败: %w",
			err,
		)
	}

	return item, nil
}

/* ==================== 新建课程 ==================== */

// CreateSubject 在单个事务中创建课程定义和课程目录。
//
// 新建请求必须包含至少一条目录配置。
// 这样课程创建成功后，至少会在一个教育域或学校中真正可见。
func CreateSubject(
	ctx context.Context,
	req *models.CreateSubjectRequest,
	updatedBy string,
) (*models.SubjectAdminItem, error) {
	if req == nil ||
		len(req.CatalogEntries) == 0 {
		return nil, ErrSubjectCatalogRequired
	}

	sortOrder := req.SortOrder
	if sortOrder <= 0 {
		sortOrder = 100
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始新建课程事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row := tx.QueryRow(
		ctx,
		`
		INSERT INTO subjects (
			name,
			code,
			sort_order,
			note,
			is_active,
			is_system,
			updated_by
		)
		VALUES ($1, $2, $3, $4, true, false, $5)
		RETURNING `+subjectSelectColumns,
		req.Name,
		req.Code,
		sortOrder,
		req.Note,
		subjectUpdatedByArg(updatedBy),
	)

	subject, err := scanSubject(row)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrSubjectNameExists
		}

		return nil, fmt.Errorf(
			"新建课程定义失败: %w",
			err,
		)
	}

	catalogEntries, err :=
		replaceSubjectCatalogEntries(
			ctx,
			tx,
			subject.ID,
			subject.Name,
			subject.SortOrder,
			req.CatalogEntries,
		)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交新建课程事务失败: %w",
			err,
		)
	}

	return &models.SubjectAdminItem{
		Subject:        *subject,
		CatalogEntries: catalogEntries,
	}, nil
}

/* ==================== 编辑课程 ==================== */

// UpdateSubject 部分更新课程定义，并可选择完整替换目录配置。
//
// CatalogEntries规则：
//   - nil：保留原目录配置，适用于列表行内启停；
//   - 非nil：先完整校验，再使用提交数组完整替换旧目录。
func UpdateSubject(
	ctx context.Context,
	id string,
	req *models.UpdateSubjectRequest,
	updatedBy string,
) (*models.SubjectAdminItem, error) {
	if req == nil {
		return nil, ErrSubjectNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始编辑课程事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := getSubjectByIDTx(
		ctx,
		tx,
		id,
	)
	if err != nil {
		return nil, err
	}

	name := current.Name
	if req.Name != nil {
		name = *req.Name
	}

	code := current.Code
	if req.Code != nil {
		code = *req.Code
	}

	sortOrder := current.SortOrder
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
		if sortOrder < 0 {
			sortOrder = 0
		}
	}

	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	note := current.Note
	if req.Note != nil {
		note = *req.Note
	}

	row := tx.QueryRow(
		ctx,
		`
		UPDATE subjects
		SET
			name = $1,
			code = $2,
			sort_order = $3,
			is_active = $4,
			note = $5,
			updated_by = $6,
			updated_at = now()
		WHERE id = $7
		RETURNING `+subjectSelectColumns,
		name,
		code,
		sortOrder,
		isActive,
		note,
		subjectUpdatedByArg(updatedBy),
		id,
	)

	subject, err := scanSubject(row)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrSubjectNameExists
		}

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubjectNotFound
		}

		return nil, fmt.Errorf(
			"编辑课程定义失败: %w",
			err,
		)
	}

	var catalogEntries []*models.SubjectCatalogEntry

	if req.CatalogEntries != nil {
		catalogEntries, err =
			replaceSubjectCatalogEntries(
				ctx,
				tx,
				subject.ID,
				subject.Name,
				subject.SortOrder,
				*req.CatalogEntries,
			)
	} else {
		catalogEntries, err =
			listSubjectCatalogEntriesBySubjectTx(
				ctx,
				tx,
				subject.ID,
			)
	}
	if err != nil {
		return nil, err
	}

	if catalogEntries == nil {
		catalogEntries =
			[]*models.SubjectCatalogEntry{}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交编辑课程事务失败: %w",
			err,
		)
	}

	return &models.SubjectAdminItem{
		Subject:        *subject,
		CatalogEntries: catalogEntries,
	}, nil
}

/* ==================== 删除课程 ==================== */

// DeleteSubject 删除非内置课程。
//
// subject_catalog_entries通过外键ON DELETE CASCADE自动清理。
// 已有教案和课件保存的是课程名称快照，不依赖该外键。
func DeleteSubject(
	ctx context.Context,
	id string,
) error {
	current, err := GetSubjectByID(ctx, id)
	if err != nil {
		return err
	}

	if current.IsSystem {
		return ErrSubjectSystemGuard
	}

	result, err := database.DB.Exec(
		ctx,
		`DELETE FROM subjects WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf(
			"删除课程失败: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrSubjectNotFound
	}

	return nil
}
