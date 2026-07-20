package repository

// curriculum_repo.go — K12课程知识库只读查询
//
// curriculum_standards与textbook_units当前均为K12专属基础数据。
//
// 所有读取函数必须显式接收可信educationDomain：
//   - k12正常查询；
//   - vocational、adult、mixed、common、空值和非法值返回类型正确的空切片；
//   - 非K12不访问数据库；
//   - K12数据库错误向上返回。
//
// 写入、批次统计和蓝绿切换位于curriculum_write_repo.go。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// isK12CurriculumEducationDomain 判断是否允许读取K12课程基础数据。
func isK12CurriculumEducationDomain(
	educationDomain string,
) bool {
	return strings.ToLower(
		strings.TrimSpace(educationDomain),
	) == models.EducationDomainK12
}

// ListCurriculumKPsBySubjectGrade 按学科和年级查询课标知识点。
func ListCurriculumKPsBySubjectGrade(
	ctx context.Context,
	educationDomain string,
	subject string,
	gradeNum int,
) ([]*models.CurriculumKP, error) {
	if !isK12CurriculumEducationDomain(
		educationDomain,
	) {
		return []*models.CurriculumKP{}, nil
	}

	conditions := []string{
		"subject = $1",
		"status = 'active'",
	}
	args := []interface{}{subject}
	argIndex := 2

	if gradeNum > 0 {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"grade_num = $%d",
				argIndex,
			),
		)
		args = append(args, gradeNum)
		argIndex++
	}

	whereClause := strings.Join(
		conditions,
		" AND ",
	)
	query := fmt.Sprintf(`
		SELECT
			id,
			subject,
			stage,
			COALESCE(grade_num, 0),
			domain,
			COALESCE(theme, ''),
			kp_code,
			kp_name,
			COALESCE(content_requirement, ''),
			COALESCE(academic_requirement, ''),
			COALESCE(teaching_hint, ''),
			depth_level,
			COALESCE(core_competency, ''),
			COALESCE(source_ref, ''),
			confidence,
			sort_order
		FROM curriculum_standards
		WHERE %s
		ORDER BY domain ASC, sort_order ASC
	`, whereClause)

	rows, err := database.DB.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课标知识点失败: %w",
			err,
		)
	}
	defer rows.Close()

	knowledgePoints :=
		make([]*models.CurriculumKP, 0)

	for rows.Next() {
		knowledgePoint :=
			&models.CurriculumKP{}

		if err := rows.Scan(
			&knowledgePoint.ID,
			&knowledgePoint.Subject,
			&knowledgePoint.Stage,
			&knowledgePoint.GradeNum,
			&knowledgePoint.Domain,
			&knowledgePoint.Theme,
			&knowledgePoint.KPCode,
			&knowledgePoint.KPName,
			&knowledgePoint.ContentRequirement,
			&knowledgePoint.AcademicRequirement,
			&knowledgePoint.TeachingHint,
			&knowledgePoint.DepthLevel,
			&knowledgePoint.CoreCompetency,
			&knowledgePoint.SourceRef,
			&knowledgePoint.Confidence,
			&knowledgePoint.SortOrder,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课标知识点行失败: %w",
				err,
			)
		}

		knowledgePoints = append(
			knowledgePoints,
			knowledgePoint,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课标知识点失败: %w",
			err,
		)
	}

	return knowledgePoints, nil
}

// GetCurriculumKPsByCodes 按知识点编码批量查询。
//
// 此函数是直接编码查询的最终防线，调用者不能省略可信教育域。
func GetCurriculumKPsByCodes(
	ctx context.Context,
	educationDomain string,
	codes []string,
) ([]*models.CurriculumKP, error) {
	if !isK12CurriculumEducationDomain(
		educationDomain,
	) ||
		len(codes) == 0 {
		return []*models.CurriculumKP{}, nil
	}

	rows, err := database.DB.Query(ctx, `
		SELECT
			id,
			subject,
			stage,
			COALESCE(grade_num, 0),
			domain,
			COALESCE(theme, ''),
			kp_code,
			kp_name,
			COALESCE(content_requirement, ''),
			COALESCE(academic_requirement, ''),
			COALESCE(teaching_hint, ''),
			depth_level,
			COALESCE(core_competency, ''),
			COALESCE(source_ref, ''),
			confidence,
			sort_order
		FROM curriculum_standards
		WHERE kp_code = ANY($1)
		  AND status = 'active'
		ORDER BY sort_order ASC
	`, codes)
	if err != nil {
		return nil, fmt.Errorf(
			"按编码查询课标知识点失败: %w",
			err,
		)
	}
	defer rows.Close()

	knowledgePoints :=
		make([]*models.CurriculumKP, 0)

	for rows.Next() {
		knowledgePoint :=
			&models.CurriculumKP{}

		if err := rows.Scan(
			&knowledgePoint.ID,
			&knowledgePoint.Subject,
			&knowledgePoint.Stage,
			&knowledgePoint.GradeNum,
			&knowledgePoint.Domain,
			&knowledgePoint.Theme,
			&knowledgePoint.KPCode,
			&knowledgePoint.KPName,
			&knowledgePoint.ContentRequirement,
			&knowledgePoint.AcademicRequirement,
			&knowledgePoint.TeachingHint,
			&knowledgePoint.DepthLevel,
			&knowledgePoint.CoreCompetency,
			&knowledgePoint.SourceRef,
			&knowledgePoint.Confidence,
			&knowledgePoint.SortOrder,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课标知识点行失败: %w",
				err,
			)
		}

		knowledgePoints = append(
			knowledgePoints,
			knowledgePoint,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课标知识点失败: %w",
			err,
		)
	}

	return knowledgePoints, nil
}

// ListTextbookUnits 按学科、版本、年级和册查询教材单元。
func ListTextbookUnits(
	ctx context.Context,
	educationDomain string,
	subject string,
	publisher string,
	gradeNum int,
	semester string,
) ([]*models.TextbookUnit, error) {
	if !isK12CurriculumEducationDomain(
		educationDomain,
	) {
		return []*models.TextbookUnit{}, nil
	}

	conditions := []string{
		"subject = $1",
		"status = 'active'",
	}
	args := []interface{}{subject}
	argIndex := 2

	if publisher != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"publisher = $%d",
				argIndex,
			),
		)
		args = append(args, publisher)
		argIndex++
	}
	if gradeNum > 0 {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"grade_num = $%d",
				argIndex,
			),
		)
		args = append(args, gradeNum)
		argIndex++
	}
	if semester != "" {
		conditions = append(
			conditions,
			fmt.Sprintf(
				"semester = $%d",
				argIndex,
			),
		)
		args = append(args, semester)
		argIndex++
	}

	whereClause := strings.Join(
		conditions,
		" AND ",
	)
	query := fmt.Sprintf(`
		SELECT
			id,
			subject,
			publisher,
			grade_num,
			semester,
			COALESCE(unit_number, 0),
			unit_title,
			COALESCE(lesson_title, ''),
			COALESCE(content_summary, ''),
			COALESCE(kp_codes::text, '[]'),
			idx_depth_level,
			source_type,
			confidence,
			sort_order
		FROM textbook_units
		WHERE %s
		ORDER BY
			grade_num ASC,
			semester ASC,
			sort_order ASC
	`, whereClause)

	rows, err := database.DB.Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询教材单元失败: %w",
			err,
		)
	}
	defer rows.Close()

	units := make(
		[]*models.TextbookUnit,
		0,
	)

	for rows.Next() {
		unit := &models.TextbookUnit{}

		if err := rows.Scan(
			&unit.ID,
			&unit.Subject,
			&unit.Publisher,
			&unit.GradeNum,
			&unit.Semester,
			&unit.UnitNumber,
			&unit.UnitTitle,
			&unit.LessonTitle,
			&unit.ContentSummary,
			&unit.KPCodesJSON,
			&unit.IdxDepthLevel,
			&unit.SourceType,
			&unit.Confidence,
			&unit.SortOrder,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描教材单元行失败: %w",
				err,
			)
		}

		units = append(units, unit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历教材单元失败: %w",
			err,
		)
	}

	return units, nil
}

// ListTextbookPublishers 查询某学科某年级的教材版本。
func ListTextbookPublishers(
	ctx context.Context,
	educationDomain string,
	subject string,
	gradeNum int,
) ([]string, error) {
	if !isK12CurriculumEducationDomain(
		educationDomain,
	) {
		return []string{}, nil
	}

	rows, err := database.DB.Query(ctx, `
		SELECT DISTINCT publisher
		FROM textbook_units
		WHERE subject = $1
		  AND grade_num = $2
		  AND status = 'active'
		ORDER BY publisher ASC
	`, subject, gradeNum)
	if err != nil {
		return nil, fmt.Errorf(
			"查询教材版本失败: %w",
			err,
		)
	}
	defer rows.Close()

	publishers := make([]string, 0)

	for rows.Next() {
		var publisher string
		if err := rows.Scan(&publisher); err != nil {
			return nil, fmt.Errorf(
				"扫描教材版本行失败: %w",
				err,
			)
		}
		publishers = append(
			publishers,
			publisher,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历教材版本失败: %w",
			err,
		)
	}

	return publishers, nil
}
