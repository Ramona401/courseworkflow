package repository

// curriculum_repo.go — 课程知识库数据访问（课标骨架层 + 教材实例层）
//
// 对应两张表：
//   curriculum_standards 课标骨架层（权威/稳定/版本无关，定义知识点与三档深度）
//   textbook_units       教材实例层（版本相关，各版本教材每年级每册每单元结构）
//
// 本文件仅提供"从主题创建课件→难度自动适配"所需的只读查询，写入由 SQL 脚本/后续PDF导入完成。
// pgx/v5 写法、错误处理风格对齐 courseware_repo.go。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 课标骨架层查询 ====================

// ListCurriculumKPsBySubjectGrade 按学科+年级查询课标知识点清单（供前端勾选 + 难度适配）
// gradeNum<=0 时只按学科查（返回该学科全部年级，慎用）；正常应传具体年级。
// 只返回 status='active' 的知识点，按 sort_order 升序。
func ListCurriculumKPsBySubjectGrade(ctx context.Context, subject string, gradeNum int) ([]*models.CurriculumKP, error) {
	conditions := []string{"subject = $1", "status = 'active'"}
	args := []interface{}{subject}
	argIdx := 2

	if gradeNum > 0 {
		conditions = append(conditions, fmt.Sprintf("grade_num = $%d", argIdx))
		args = append(args, gradeNum)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")
	sql := fmt.Sprintf(`SELECT id, subject, stage, COALESCE(grade_num,0), domain,
COALESCE(theme,''), kp_code, kp_name,
COALESCE(content_requirement,''), COALESCE(academic_requirement,''), COALESCE(teaching_hint,''),
depth_level, COALESCE(core_competency,''), COALESCE(source_ref,''), confidence, sort_order
FROM curriculum_standards
WHERE %s
ORDER BY domain ASC, sort_order ASC`, whereClause)

	rows, err := database.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("查询课标知识点失败: %w", err)
	}
	defer rows.Close()

	var kps []*models.CurriculumKP
	for rows.Next() {
		kp := &models.CurriculumKP{}
		if err := rows.Scan(
			&kp.ID, &kp.Subject, &kp.Stage, &kp.GradeNum, &kp.Domain,
			&kp.Theme, &kp.KPCode, &kp.KPName,
			&kp.ContentRequirement, &kp.AcademicRequirement, &kp.TeachingHint,
			&kp.DepthLevel, &kp.CoreCompetency, &kp.SourceRef, &kp.Confidence, &kp.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("扫描课标知识点行失败: %w", err)
		}
		kps = append(kps, kp)
	}
	return kps, nil
}

// GetCurriculumKPsByCodes 按知识点编码数组批量查询（供生成课件时注入难度约束）
// codes 为空时返回空切片（不报错），调用方据此走"无知识库约束"的原有逻辑。
func GetCurriculumKPsByCodes(ctx context.Context, codes []string) ([]*models.CurriculumKP, error) {
	if len(codes) == 0 {
		return []*models.CurriculumKP{}, nil
	}

	sql := `SELECT id, subject, stage, COALESCE(grade_num,0), domain,
COALESCE(theme,''), kp_code, kp_name,
COALESCE(content_requirement,''), COALESCE(academic_requirement,''), COALESCE(teaching_hint,''),
depth_level, COALESCE(core_competency,''), COALESCE(source_ref,''), confidence, sort_order
FROM curriculum_standards
WHERE kp_code = ANY($1) AND status = 'active'
ORDER BY sort_order ASC`

	rows, err := database.DB.Query(ctx, sql, codes)
	if err != nil {
		return nil, fmt.Errorf("按编码查询课标知识点失败: %w", err)
	}
	defer rows.Close()

	var kps []*models.CurriculumKP
	for rows.Next() {
		kp := &models.CurriculumKP{}
		if err := rows.Scan(
			&kp.ID, &kp.Subject, &kp.Stage, &kp.GradeNum, &kp.Domain,
			&kp.Theme, &kp.KPCode, &kp.KPName,
			&kp.ContentRequirement, &kp.AcademicRequirement, &kp.TeachingHint,
			&kp.DepthLevel, &kp.CoreCompetency, &kp.SourceRef, &kp.Confidence, &kp.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("扫描课标知识点行失败: %w", err)
		}
		kps = append(kps, kp)
	}
	return kps, nil
}

// ==================== 教材实例层查询 ====================

// ListTextbookUnits 按学科+版本+年级查询教材单元（供前端联动单元清单）
// publisher 为空时返回该学科该年级全部版本；semester 为空时返回上下册全部。
func ListTextbookUnits(ctx context.Context, subject string, publisher string, gradeNum int, semester string) ([]*models.TextbookUnit, error) {
	conditions := []string{"subject = $1", "status = 'active'"}
	args := []interface{}{subject}
	argIdx := 2

	if publisher != "" {
		conditions = append(conditions, fmt.Sprintf("publisher = $%d", argIdx))
		args = append(args, publisher)
		argIdx++
	}
	if gradeNum > 0 {
		conditions = append(conditions, fmt.Sprintf("grade_num = $%d", argIdx))
		args = append(args, gradeNum)
		argIdx++
	}
	if semester != "" {
		conditions = append(conditions, fmt.Sprintf("semester = $%d", argIdx))
		args = append(args, semester)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")
	sql := fmt.Sprintf(`SELECT id, subject, publisher, grade_num, semester,
COALESCE(unit_number,0), unit_title, COALESCE(lesson_title,''), COALESCE(content_summary,''),
COALESCE(kp_codes::text,'[]'), idx_depth_level, source_type, confidence, sort_order
FROM textbook_units
WHERE %s
ORDER BY grade_num ASC, semester ASC, sort_order ASC`, whereClause)

	rows, err := database.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("查询教材单元失败: %w", err)
	}
	defer rows.Close()

	var units []*models.TextbookUnit
	for rows.Next() {
		u := &models.TextbookUnit{}
		if err := rows.Scan(
			&u.ID, &u.Subject, &u.Publisher, &u.GradeNum, &u.Semester,
			&u.UnitNumber, &u.UnitTitle, &u.LessonTitle, &u.ContentSummary,
			&u.KPCodesJSON, &u.IdxDepthLevel, &u.SourceType, &u.Confidence, &u.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("扫描教材单元行失败: %w", err)
		}
		units = append(units, u)
	}
	return units, nil
}

// ListTextbookPublishers 查询某学科某年级下都有哪些教材版本（供前端版本下拉）
func ListTextbookPublishers(ctx context.Context, subject string, gradeNum int) ([]string, error) {
	sql := `SELECT DISTINCT publisher FROM textbook_units
WHERE subject = $1 AND grade_num = $2 AND status = 'active'
ORDER BY publisher ASC`
	rows, err := database.DB.Query(ctx, sql, subject, gradeNum)
	if err != nil {
		return nil, fmt.Errorf("查询教材版本失败: %w", err)
	}
	defer rows.Close()

	var publishers []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("扫描教材版本行失败: %w", err)
		}
		publishers = append(publishers, p)
	}
	return publishers, nil
}
