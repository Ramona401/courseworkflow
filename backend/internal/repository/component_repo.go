package repository

// component_repo.go — 组件Repository共享辅助与统计。
//
// 旧无教育域CRUD、详情、列表和匹配函数已删除。
// 正式数据访问入口分别位于：
//   - component_domain_repo.go
//   - component_management_domain_repo.go
//   - component_match_domain_repo.go
//   - component_smart_match_domain_repo.go
//   - component_extraction_domain_repo.go
//
// 本文件只保留：
//   - 跨域Repository共同使用的ErrComponentNotFound；
//   - AOCI冗余索引列筛选SQL构造器；
//   - 使用、选中和质量分统计函数。

import (
	"context"
	"errors"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrComponentNotFound 对不存在和教育域不可见组件使用统一错误。
var ErrComponentNotFound = errors.New(
	"组件不存在",
)

// buildIndexColumnConditions 构建AOCI索引冗余列的可选WHERE条件。
//
// 未传筛选项时不附加条件；冗余列值为0表示尚未维护索引，
// 为兼容存量数据仍保留在候选集中。
func buildIndexColumnConditions(
	req *models.MatchComponentsRequest,
	args []interface{},
	argIndex int,
) (
	string,
	[]interface{},
	int,
) {
	conditions := ""

	if len(req.CognitiveLevel) > 0 {
		conditions += fmt.Sprintf(
			" AND (c.idx_cognitive_level = ANY($%d) OR c.idx_cognitive_level = 0)",
			argIndex,
		)
		args = append(
			args,
			req.CognitiveLevel,
		)
		argIndex++
	}

	if len(req.StageTiming) > 0 {
		conditions += fmt.Sprintf(
			" AND (c.idx_stage_timing = ANY($%d) OR c.idx_stage_timing = 0)",
			argIndex,
		)
		args = append(
			args,
			req.StageTiming,
		)
		argIndex++
	}

	if len(req.PedagogyIntensity) > 0 {
		conditions += fmt.Sprintf(
			" AND (c.idx_pedagogy_intensity = ANY($%d) OR c.idx_pedagogy_intensity = 0)",
			argIndex,
		)
		args = append(
			args,
			req.PedagogyIntensity,
		)
		argIndex++
	}

	return conditions,
		args,
		argIndex
}

// IncrementComponentUsage 增加组件使用次数。
func IncrementComponentUsage(
	ctx context.Context,
	componentID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
			UPDATE lesson_plan_components
			SET
				usage_count = usage_count + 1,
				updated_at = now()
			WHERE id = $1
		`,
		componentID,
	)

	return err
}

// IncrementComponentSelect 增加组件选中次数。
func IncrementComponentSelect(
	ctx context.Context,
	componentID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
			UPDATE lesson_plan_components
			SET
				select_count = select_count + 1,
				updated_at = now()
			WHERE id = $1
		`,
		componentID,
	)

	return err
}

// UpdateComponentQualityScore 更新组件质量分。
//
// 公式保持原有行为：
//   - 选中率占40%；
//   - 关联教案平均AI评分占40%；
//   - 点赞净值占20%。
func UpdateComponentQualityScore(
	ctx context.Context,
	componentID string,
	averageLinkedPlanScore float64,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
			UPDATE lesson_plan_components
			SET
				quality_score = (
					(
						CAST(select_count AS NUMERIC)
						/
						GREATEST(usage_count, 1)
					) * 0.4
					+
					($1 / 10.0) * 0.4
					+
					(
						CAST(
							like_count - dislike_count
							AS NUMERIC
						)
						/
						GREATEST(
							like_count + dislike_count,
							1
						)
					) * 0.2
				),
				updated_at = now()
			WHERE id = $2
		`,
		averageLinkedPlanScore,
		componentID,
	)
	if err != nil {
		return fmt.Errorf(
			"更新组件质量分失败: %w",
			err,
		)
	}

	return nil
}

// GetComponentLinkedPlanAvgScore 计算组件关联教案的平均AI评审分。
func GetComponentLinkedPlanAvgScore(
	ctx context.Context,
	componentID string,
) (float64, error) {
	var averageScore float64

	err := database.DB.QueryRow(
		ctx,
		`
			SELECT COALESCE(
				AVG(lp.ai_review_score),
				0
			)
			FROM component_extractions ce
			JOIN lesson_plans lp
			  ON lp.id = ce.source_lesson_plan_id
			WHERE ce.extracted_component_id = $1
			  AND lp.ai_review_score IS NOT NULL
		`,
		componentID,
	).Scan(
		&averageScore,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"查询组件关联教案均分失败: %w",
			err,
		)
	}

	return averageScore, nil
}
