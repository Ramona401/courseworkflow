package repository

// class_profile_repo.go — 班级学情数据访问（差异化教学·老师私有资料，独立模块）
//
// 操作两张表：
//   class_profiles  班级学情卡（群体结论，注入 AI）
//   class_students  学生个体档案（本地明细，永不注入 AI）
//
// 鉴权口径：纯个人。所有查询/写入都按 created_by / owner_id == 当前用户收窄，
// 不走 unit_plans 那套 group/school/system 可见性（班级是老师自己带的）。
// 归属校验交 service 层（先 GetClassProfileByID 拿 created_by 比对 userID），
// 本层只忠实存取。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrClassProfileNotFound 班级学情卡不存在
var ErrClassProfileNotFound = errors.New("班级学情卡不存在")

// ErrClassStudentNotFound 学生档案不存在
var ErrClassStudentNotFound = errors.New("学生档案不存在")

// ---------- 班级学情卡：统一列与扫描 ----------

// classProfileSelectColumns 单条查询统一列（与 scanClassProfile 对齐）
const classProfileSelectColumns = `id, scope, scope_target_id, subject, grade, class_name, term,
student_count, overall_profile, tier_structure, weak_points, teaching_advice,
last_analyzed_at, last_analyzed_from, created_by, status, created_at, updated_at`

// scanClassProfile 统一扫描单条班级学情卡
func scanClassProfile(row pgx.Row) (*models.ClassProfile, error) {
	p := &models.ClassProfile{}
	err := row.Scan(
		&p.ID, &p.Scope, &p.ScopeTargetID, &p.Subject, &p.Grade, &p.ClassName, &p.Term,
		&p.StudentCount, &p.OverallProfile, &p.TierStructure, &p.WeakPoints, &p.TeachingAdvice,
		&p.LastAnalyzedAt, &p.LastAnalyzedFrom, &p.CreatedBy, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClassProfileNotFound
		}
		return nil, fmt.Errorf("扫描班级学情卡失败: %w", err)
	}
	return p, nil
}

// ---------- 班级学情卡：CRUD ----------

// CreateClassProfile 新建一张班级学情卡（status=active），回填 id/时间
//
// scope 固定 personal，scope_target_id 固定全零占位（v1 纯个人）。
func CreateClassProfile(ctx context.Context, p *models.ClassProfile) error {
	err := database.DB.QueryRow(ctx, `
		INSERT INTO class_profiles
		  (scope, scope_target_id, subject, grade, class_name, term, student_count,
		   overall_profile, tier_structure, weak_points, teaching_advice,
		   last_analyzed_from, created_by, status)
		VALUES ('personal', '00000000-0000-0000-0000-000000000000',
		        $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'active')
		RETURNING id, created_at, updated_at
	`,
		p.Subject, p.Grade, p.ClassName, p.Term, p.StudentCount,
		p.OverallProfile, p.TierStructure, p.WeakPoints, p.TeachingAdvice,
		p.LastAnalyzedFrom, p.CreatedBy,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建班级学情卡失败: %w", err)
	}
	p.Scope = models.ClassProfileScopePersonal
	p.ScopeTargetID = models.ClassProfileSystemTargetID
	p.Status = models.ClassProfileStatusActive
	return nil
}

// GetClassProfileByID 按 ID 查单条班级学情卡（归属校验用）
func GetClassProfileByID(ctx context.Context, id string) (*models.ClassProfile, error) {
	sql := `SELECT ` + classProfileSelectColumns + ` FROM class_profiles WHERE id = $1`
	return scanClassProfile(database.DB.QueryRow(ctx, sql, id))
}

// UpdateClassProfile 更新班级学情卡（定位字段 + 四大段群体学情内容）
//
// from 写入 last_analyzed_from（来源标记）；setAnalyzedNow=true 时同时把 last_analyzed_at 置为 now()。
// 手写编辑场景 setAnalyzedNow 传 false（不算"一次分析"）；AI 总结/导入场景传 true。
func UpdateClassProfile(ctx context.Context, id string, req *models.UpdateClassProfileRequest, from string, setAnalyzedNow bool) error {
	var result pgx.Row
	if setAnalyzedNow {
		result = database.DB.QueryRow(ctx, `
			UPDATE class_profiles
			SET subject=$1, grade=$2, class_name=$3, term=$4, student_count=$5,
			    overall_profile=$6, tier_structure=$7, weak_points=$8, teaching_advice=$9,
			    last_analyzed_from=$10, last_analyzed_at=now(), updated_at=now()
			WHERE id=$11 AND status<>'archived'
			RETURNING id
		`,
			req.Subject, req.Grade, req.ClassName, req.Term, req.StudentCount,
			req.OverallProfile, req.TierStructure, req.WeakPoints, req.TeachingAdvice,
			from, id,
		)
	} else {
		result = database.DB.QueryRow(ctx, `
			UPDATE class_profiles
			SET subject=$1, grade=$2, class_name=$3, term=$4, student_count=$5,
			    overall_profile=$6, tier_structure=$7, weak_points=$8, teaching_advice=$9,
			    last_analyzed_from=$10, updated_at=now()
			WHERE id=$11 AND status<>'archived'
			RETURNING id
		`,
			req.Subject, req.Grade, req.ClassName, req.Term, req.StudentCount,
			req.OverallProfile, req.TierStructure, req.WeakPoints, req.TeachingAdvice,
			from, id,
		)
	}
	var rid string
	if err := result.Scan(&rid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClassProfileNotFound
		}
		return fmt.Errorf("更新班级学情卡失败: %w", err)
	}
	return nil
}

// DeleteClassProfile 软删除班级学情卡（status=archived）
//
// 学生明细（class_students）靠外键 ON DELETE CASCADE 仅在硬删时连带清理；
// 此处是软删，学生明细保留（卡归档后不再展示，数据仍在）。
func DeleteClassProfile(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE class_profiles SET status='archived', updated_at=now() WHERE id=$1 AND status<>'archived'`,
		id)
	if err != nil {
		return fmt.Errorf("删除班级学情卡失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrClassProfileNotFound
	}
	return nil
}

// ListClassProfiles 列出某老师自己的全部班级学情卡（不含 archived）
//
// 纯个人：只按 created_by 过滤。列表轻量，不取四大段正文，
// 仅用 has_profile 表达"四大段是否已有任何内容"。
func ListClassProfiles(ctx context.Context, ownerID string) ([]*models.ClassProfileListItem, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT id, subject, grade, class_name, term, student_count,
		       (overall_profile <> '' OR tier_structure <> '' OR weak_points <> '' OR teaching_advice <> '') AS has_profile,
		       last_analyzed_at, last_analyzed_from, updated_at
		FROM class_profiles
		WHERE created_by = $1 AND status <> 'archived'
		ORDER BY updated_at DESC
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("查询班级学情卡列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ClassProfileListItem
	for rows.Next() {
		it := &models.ClassProfileListItem{}
		if err := rows.Scan(
			&it.ID, &it.Subject, &it.Grade, &it.ClassName, &it.Term, &it.StudentCount,
			&it.HasProfile, &it.LastAnalyzedAt, &it.LastAnalyzedFrom, &it.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描班级学情卡列表行失败: %w", err)
		}
		items = append(items, it)
	}
	if items == nil {
		items = []*models.ClassProfileListItem{}
	}
	return items, nil
}

// ---------- 学生个体档案：统一列与扫描 ----------

// classStudentSelectColumns 单条查询统一列（与 scanClassStudent 对齐）
const classStudentSelectColumns = `id, class_profile_id, owner_id, student_code, tier,
COALESCE(scores::text,'[]'), latest_score, weak_topics, note, created_at, updated_at`

// scanClassStudent 统一扫描单条学生档案
func scanClassStudent(row pgx.Row) (*models.ClassStudent, error) {
	s := &models.ClassStudent{}
	err := row.Scan(
		&s.ID, &s.ClassProfileID, &s.OwnerID, &s.StudentCode, &s.Tier,
		&s.Scores, &s.LatestScore, &s.WeakTopics, &s.Note, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClassStudentNotFound
		}
		return nil, fmt.Errorf("扫描学生档案失败: %w", err)
	}
	return s, nil
}

// ---------- 学生个体档案：CRUD（批次2 才接前端，批次1 先备好数据层）----------

// CreateClassStudent 新建一条学生档案，回填 id/时间
func CreateClassStudent(ctx context.Context, s *models.ClassStudent) error {
	scoresJSON := s.Scores
	if scoresJSON == "" {
		scoresJSON = "[]"
	}
	err := database.DB.QueryRow(ctx, `
		INSERT INTO class_students
		  (class_profile_id, owner_id, student_code, tier, scores, latest_score, weak_topics, note)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$8)
		RETURNING id, created_at, updated_at
	`,
		s.ClassProfileID, s.OwnerID, s.StudentCode, s.Tier, scoresJSON,
		s.LatestScore, s.WeakTopics, s.Note,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建学生档案失败: %w", err)
	}
	return nil
}

// GetClassStudentByID 按 ID 查单条学生档案（归属校验用）
func GetClassStudentByID(ctx context.Context, id string) (*models.ClassStudent, error) {
	sql := `SELECT ` + classStudentSelectColumns + ` FROM class_students WHERE id = $1`
	return scanClassStudent(database.DB.QueryRow(ctx, sql, id))
}

// UpdateClassStudent 更新学生档案（分层/成绩/易错点/备注）
func UpdateClassStudent(ctx context.Context, s *models.ClassStudent) error {
	scoresJSON := s.Scores
	if scoresJSON == "" {
		scoresJSON = "[]"
	}
	result, err := database.DB.Exec(ctx, `
		UPDATE class_students
		SET student_code=$1, tier=$2, scores=$3::jsonb, latest_score=$4,
		    weak_topics=$5, note=$6, updated_at=now()
		WHERE id=$7
	`,
		s.StudentCode, s.Tier, scoresJSON, s.LatestScore, s.WeakTopics, s.Note, s.ID,
	)
	if err != nil {
		return fmt.Errorf("更新学生档案失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrClassStudentNotFound
	}
	return nil
}

// DeleteClassStudent 硬删除一条学生档案
func DeleteClassStudent(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `DELETE FROM class_students WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("删除学生档案失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrClassStudentNotFound
	}
	return nil
}

// ListClassStudents 列出某班级的全部学生档案（按学号代号排序）
func ListClassStudents(ctx context.Context, classProfileID string) ([]*models.ClassStudent, error) {
	sql := `SELECT ` + classStudentSelectColumns + `
	        FROM class_students WHERE class_profile_id = $1
	        ORDER BY tier ASC, student_code ASC`
	rows, err := database.DB.Query(ctx, sql, classProfileID)
	if err != nil {
		return nil, fmt.Errorf("查询学生档案列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.ClassStudent
	for rows.Next() {
		s := &models.ClassStudent{}
		if err := rows.Scan(
			&s.ID, &s.ClassProfileID, &s.OwnerID, &s.StudentCode, &s.Tier,
			&s.Scores, &s.LatestScore, &s.WeakTopics, &s.Note, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描学生档案列表行失败: %w", err)
		}
		items = append(items, s)
	}
	if items == nil {
		items = []*models.ClassStudent{}
	}
	return items, nil
}

// CountClassStudents 统计某班级的学生档案数（供更新 student_count 用）
func CountClassStudents(ctx context.Context, classProfileID string) (int, error) {
	var n int
	err := database.DB.QueryRow(ctx,
		`SELECT count(*) FROM class_students WHERE class_profile_id=$1`, classProfileID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("统计学生档案数失败: %w", err)
	}
	return n, nil
}
