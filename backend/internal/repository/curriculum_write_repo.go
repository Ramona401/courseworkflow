package repository

// curriculum_write_repo.go — 课程知识库写入与批次切换
//
// 本文件从curriculum_repo.go拆出，保持原有写入语义：
//   - 审核通过的课标知识点写入候选批次；
//   - 按批次统计已灌入条数；
//   - 在单事务中归档旧批并激活新批。
//
// 本文件不参与上下文14的HTTP只读教育域判断；
// 查询端的K12限制位于curriculum_repo.go。

import (
	"context"
	"fmt"

	"tedna/internal/database"
)

// CurriculumInsertRow 描述课标灌入目标表的一行结构化数据。
type CurriculumInsertRow struct {
	Subject             string
	Stage               string
	GradeNum            int
	Domain              string
	Theme               string
	KPCode              string
	KPName              string
	ContentRequirement  string
	AcademicRequirement string
	TeachingHint        string
	DepthLevel          int
	CoreCompetency      string
	SourceRef           string
	Confidence          int
	SortOrder           int
	BatchTag            string
	Status              string
}

// InsertCurriculumStandard 写入一条课标知识点。
func InsertCurriculumStandard(
	ctx context.Context,
	row *CurriculumInsertRow,
) (string, error) {
	if row == nil {
		return "", fmt.Errorf(
			"灌入课标知识点失败: 写入行为空",
		)
	}

	if row.Status == "" {
		row.Status = "active"
	}

	var id string
	err := database.DB.QueryRow(ctx, `
		INSERT INTO curriculum_standards (
			subject,
			stage,
			grade_num,
			domain,
			theme,
			kp_code,
			kp_name,
			content_requirement,
			academic_requirement,
			teaching_hint,
			depth_level,
			core_competency,
			source_ref,
			confidence,
			sort_order,
			status,
			batch_tag
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17
		)
		ON CONFLICT (kp_code) DO UPDATE SET
			subject = EXCLUDED.subject,
			stage = EXCLUDED.stage,
			grade_num = EXCLUDED.grade_num,
			domain = EXCLUDED.domain,
			theme = EXCLUDED.theme,
			kp_name = EXCLUDED.kp_name,
			content_requirement =
				EXCLUDED.content_requirement,
			academic_requirement =
				EXCLUDED.academic_requirement,
			teaching_hint =
				EXCLUDED.teaching_hint,
			depth_level =
				EXCLUDED.depth_level,
			core_competency =
				EXCLUDED.core_competency,
			source_ref =
				EXCLUDED.source_ref,
			confidence =
				EXCLUDED.confidence,
			sort_order =
				EXCLUDED.sort_order,
			status =
				EXCLUDED.status,
			batch_tag =
				EXCLUDED.batch_tag
		RETURNING id
	`,
		row.Subject,
		row.Stage,
		row.GradeNum,
		row.Domain,
		nullIfEmptyStr(row.Theme),
		row.KPCode,
		row.KPName,
		nullIfEmptyStr(
			row.ContentRequirement,
		),
		nullIfEmptyStr(
			row.AcademicRequirement,
		),
		nullIfEmptyStr(
			row.TeachingHint,
		),
		row.DepthLevel,
		nullIfEmptyStr(
			row.CoreCompetency,
		),
		nullIfEmptyStr(
			row.SourceRef,
		),
		row.Confidence,
		row.SortOrder,
		row.Status,
		row.BatchTag,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf(
			"灌入课标知识点失败: %w",
			err,
		)
	}

	return id, nil
}

// CountCurriculumByBatch 统计某批次已经灌入的条数。
func CountCurriculumByBatch(
	ctx context.Context,
	batchTag string,
) (int, error) {
	var count int

	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM curriculum_standards
		WHERE batch_tag = $1
	`, batchTag).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf(
			"统计批次条数失败: %w",
			err,
		)
	}

	return count, nil
}

// SwitchCurriculumBatch 单事务切换课标数据批次。
//
// 原子规则：
//  1. 归档不属于新批次的旧active数据；
//  2. 激活指定的新批次；
//  3. 两步全部成功才提交。
func SwitchCurriculumBatch(
	ctx context.Context,
	newBatchTag string,
) (
	archivedCount int,
	activatedCount int,
	err error,
) {
	if newBatchTag == "" {
		return 0, 0, fmt.Errorf(
			"新批次 batch_tag 不能为空",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf(
			"开启切换事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	archiveResult, err := tx.Exec(ctx, `
		UPDATE curriculum_standards
		SET status = 'archived'
		WHERE status = 'active'
		  AND batch_tag <> $1
	`, newBatchTag)
	if err != nil {
		return 0, 0, fmt.Errorf(
			"归档旧批失败: %w",
			err,
		)
	}

	archivedCount =
		int(archiveResult.RowsAffected())

	activateResult, err := tx.Exec(ctx, `
		UPDATE curriculum_standards
		SET status = 'active'
		WHERE batch_tag = $1
	`, newBatchTag)
	if err != nil {
		return 0, 0, fmt.Errorf(
			"激活新批失败: %w",
			err,
		)
	}

	activatedCount =
		int(activateResult.RowsAffected())

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf(
			"提交切换事务失败: %w",
			err,
		)
	}

	return archivedCount, activatedCount, nil
}
