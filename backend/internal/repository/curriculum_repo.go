package repository

// curriculum_repo.go — 课程知识库数据访问（课标骨架层 + 教材实例层）
//
// 对应两张表：
//   curriculum_standards 课标骨架层（权威/稳定/版本无关，定义知识点与三档深度）
//   textbook_units       教材实例层（版本相关，各版本教材每年级每册每单元结构）
//
// 原有：从主题创建课件→难度自动适配所需的只读查询（List/GetByCodes/教材单元/出版社）。
// 知识库压缩入库轮新增（本轮）：
//   - InsertCurriculumStandard  commit 灌入：把压缩确认的知识点写入目标表（带 batch_tag）
//   - CountCurriculumByBatch    某批次已灌入条数（供切换前核对）
//   - SwitchCurriculumBatch     蓝绿切换：旧批 active→archived、新批指定→active（单事务）
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

// ==================== 知识库压缩入库：commit 灌入 + 蓝绿切换（本轮新增）====================

// CurriculumInsertRow commit 灌入目标表所需的一行结构化字段
// （由 service 层用 DecodeCurriculumIndex 把 final_line 拆解后填充）
type CurriculumInsertRow struct {
	Subject             string // 学科中文（如"数学"）
	Stage               string // 学段（小学低/小学中/...）
	GradeNum            int    // 年级 1-12，0=学段级
	Domain              string // 领域
	Theme               string // 主题（可空）
	KPCode              string // 知识点编码（唯一键）
	KPName              string // 知识点名称
	ContentRequirement  string // 内容要求/边界
	AcademicRequirement string // 学业要求
	TeachingHint        string // 教学提示
	DepthLevel          int    // 深度档 1-3
	CoreCompetency      string // 核心素养
	SourceRef           string // 出处
	Confidence          int    // 置信度
	SortOrder           int    // 排序
	BatchTag            string // 批次标识
	Status              string // 灌入时的状态（蓝绿切换：新批先非 active，整批切换才转 active）
}

// InsertCurriculumStandard 把一条压缩确认的知识点灌入 curriculum_standards（带 batch_tag）
// 返回新记录 id（供回填 kb_compress_items.committed_ref）。
// kp_code 唯一冲突时按 ON CONFLICT 更新（同批重复 commit 幂等），更新内容并保持该批 status。
func InsertCurriculumStandard(ctx context.Context, row *CurriculumInsertRow) (string, error) {
	if row.Status == "" {
		row.Status = "active" // 兜底，但正常蓝绿流程应由 service 传入候选态
	}
	var id string
	err := database.DB.QueryRow(ctx, `
		INSERT INTO curriculum_standards
		  (subject, stage, grade_num, domain, theme, kp_code, kp_name,
		   content_requirement, academic_requirement, teaching_hint, depth_level,
		   core_competency, source_ref, confidence, sort_order, status, batch_tag)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (kp_code) DO UPDATE SET
		  subject = EXCLUDED.subject, stage = EXCLUDED.stage, grade_num = EXCLUDED.grade_num,
		  domain = EXCLUDED.domain, theme = EXCLUDED.theme, kp_name = EXCLUDED.kp_name,
		  content_requirement = EXCLUDED.content_requirement,
		  academic_requirement = EXCLUDED.academic_requirement,
		  teaching_hint = EXCLUDED.teaching_hint, depth_level = EXCLUDED.depth_level,
		  core_competency = EXCLUDED.core_competency, source_ref = EXCLUDED.source_ref,
		  confidence = EXCLUDED.confidence, sort_order = EXCLUDED.sort_order,
		  status = EXCLUDED.status, batch_tag = EXCLUDED.batch_tag
		RETURNING id
	`,
		row.Subject, row.Stage, row.GradeNum, row.Domain, nullIfEmptyStr(row.Theme),
		row.KPCode, row.KPName, nullIfEmptyStr(row.ContentRequirement),
		nullIfEmptyStr(row.AcademicRequirement), nullIfEmptyStr(row.TeachingHint),
		row.DepthLevel, nullIfEmptyStr(row.CoreCompetency), nullIfEmptyStr(row.SourceRef),
		row.Confidence, row.SortOrder, row.Status, row.BatchTag,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("灌入课标知识点失败: %w", err)
	}
	return id, nil
}

// CountCurriculumByBatch 统计某批次已灌入的条数（切换前核对用）
func CountCurriculumByBatch(ctx context.Context, batchTag string) (int, error) {
	var n int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM curriculum_standards WHERE batch_tag = $1`, batchTag,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("统计批次条数失败: %w", err)
	}
	return n, nil
}

// SwitchCurriculumBatch 蓝绿切换（单事务）：
//  1. 把当前所有 status='active' 的旧数据置为 archived（含遗留批 batch_tag=” 与历史批）
//  2. 把指定 newBatchTag 的数据置为 active
//
// 消费端只读 active，切换瞬时完成；选错可用旧批 batch_tag 反向切回。
// 返回 (归档旧条数, 激活新条数, error)。
func SwitchCurriculumBatch(ctx context.Context, newBatchTag string) (archivedCount int, activatedCount int, err error) {
	if newBatchTag == "" {
		return 0, 0, fmt.Errorf("新批次 batch_tag 不能为空")
	}
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("开启切换事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1) 旧 active（且不属于新批）→ archived
	tagArchive, err := tx.Exec(ctx,
		`UPDATE curriculum_standards SET status = 'archived'
		 WHERE status = 'active' AND batch_tag <> $1`, newBatchTag)
	if err != nil {
		return 0, 0, fmt.Errorf("归档旧批失败: %w", err)
	}
	archivedCount = int(tagArchive.RowsAffected())

	// 2) 新批 → active
	tagActivate, err := tx.Exec(ctx,
		`UPDATE curriculum_standards SET status = 'active' WHERE batch_tag = $1`, newBatchTag)
	if err != nil {
		return 0, 0, fmt.Errorf("激活新批失败: %w", err)
	}
	activatedCount = int(tagActivate.RowsAffected())

	if err = tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("提交切换事务失败: %w", err)
	}
	return archivedCount, activatedCount, nil
}
