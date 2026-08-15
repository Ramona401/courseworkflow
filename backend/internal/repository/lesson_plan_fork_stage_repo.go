package repository

// lesson_plan_fork_stage_repo.go — Fork副本阶段运行态的事务内初始化
//
// Fork只继承来源教案的阶段“配置模板”，不继承任何阶段进度：
//   - 优先复用来源已固化的 stage_config 快照；
//   - 老旧来源没有可用快照时，在同一事务读取当前系统默认阶段；
//   - Fork已经复制完整正文，因此复用“导入已有完整教案”的正式语义；
//   - review之前阶段统一标记skipped，review作为唯一in_progress当前阶段；
//   - 任一步失败都由外层Fork事务整体回滚，禁止留下“有正文、无合法运行态”的半成品副本。

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/models"
)

type lessonPlanForkStageBootstrap struct {
	ConfigJSON   string
	CurrentStage models.StageConfigSnapshot
	StageOutputs []models.WorkshopStageOutput
}

func resolveLessonPlanForkStageBootstrapTx(
	ctx context.Context,
	tx pgx.Tx,
	sourceStageConfig string,
) (*lessonPlanForkStageBootstrap, error) {
	snapshots, ok := normalizeLessonPlanForkStageSnapshots(
		sourceStageConfig,
	)
	if !ok {
		var err error
		snapshots, err = loadLessonPlanForkDefaultStageSnapshotsTx(
			ctx,
			tx,
		)
		if err != nil {
			return nil, err
		}
	}

	stageOutputs, currentStage, err :=
		buildLessonPlanForkCompleteStageOutputs(
			snapshots,
		)
	if err != nil {
		return nil, err
	}

	configBytes, err := json.Marshal(snapshots)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化Fork阶段配置失败: %w",
			err,
		)
	}

	return &lessonPlanForkStageBootstrap{
		ConfigJSON:   string(configBytes),
		CurrentStage: currentStage,
		StageOutputs: stageOutputs,
	}, nil
}

func buildLessonPlanForkCompleteStageOutputs(
	snapshots []models.StageConfigSnapshot,
) (
	[]models.WorkshopStageOutput,
	models.StageConfigSnapshot,
	error,
) {
	reviewIndex := -1
	for index := range snapshots {
		if strings.TrimSpace(
			snapshots[index].StageCode,
		) == "review" {
			reviewIndex = index
			break
		}
	}

	if reviewIndex < 0 {
		return nil,
			models.StageConfigSnapshot{},
			fmt.Errorf(
				"Fork完整教案阶段配置缺少review阶段",
			)
	}

	outputs := make(
		[]models.WorkshopStageOutput,
		0,
		reviewIndex+1,
	)

	for index := 0; index <= reviewIndex; index++ {
		snapshot := snapshots[index]
		stageCode := strings.TrimSpace(
			snapshot.StageCode,
		)
		if stageCode == "" {
			return nil,
				models.StageConfigSnapshot{},
				fmt.Errorf(
					"Fork完整教案阶段配置存在空阶段代码",
				)
		}

		status := models.StageOutputSkipped
		if index == reviewIndex {
			status = models.StageOutputInProgress
		}

		outputs = append(
			outputs,
			models.WorkshopStageOutput{
				StageCode:            stageCode,
				StageOrder:           snapshot.StageOrder,
				StructuredOutput:     "{}",
				NarrativeOutput:      "",
				ConversationSnapshot: "[]",
				Status:               status,
			},
		)
	}

	return outputs, snapshots[reviewIndex], nil
}

func normalizeLessonPlanForkStageSnapshots(
	raw string,
) ([]models.StageConfigSnapshot, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil, false
	}

	var snapshots []models.StageConfigSnapshot
	if err := json.Unmarshal(
		[]byte(raw),
		&snapshots,
	); err != nil || len(snapshots) == 0 {
		return nil, false
	}

	seen := make(map[string]struct{}, len(snapshots))
	for index := range snapshots {
		snapshots[index].StageCode =
			strings.TrimSpace(
				snapshots[index].StageCode,
			)

		if snapshots[index].StageCode == "" ||
			snapshots[index].StageOrder <= 0 {
			return nil, false
		}

		if _, exists := seen[snapshots[index].StageCode]; exists {
			return nil, false
		}
		seen[snapshots[index].StageCode] = struct{}{}
	}

	sort.SliceStable(
		snapshots,
		func(left int, right int) bool {
			return snapshots[left].StageOrder <
				snapshots[right].StageOrder
		},
	)

	return snapshots, true
}

func loadLessonPlanForkDefaultStageSnapshotsTx(
	ctx context.Context,
	tx pgx.Tx,
) ([]models.StageConfigSnapshot, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			stage_code,
			stage_name,
			stage_order,
			ai_role,
			gate_mode,
			skippable
		FROM workshop_stages
		WHERE source = $1
		  AND status = 'active'
		ORDER BY stage_order
	`, models.StageSourceSystem)
	if err != nil {
		return nil, fmt.Errorf(
			"读取Fork默认阶段失败: %w",
			err,
		)
	}
	defer rows.Close()

	snapshots :=
		make(
			[]models.StageConfigSnapshot,
			0,
		)

	for rows.Next() {
		var snapshot models.StageConfigSnapshot
		if err := rows.Scan(
			&snapshot.StageCode,
			&snapshot.StageName,
			&snapshot.StageOrder,
			&snapshot.AIRole,
			&snapshot.GateMode,
			&snapshot.Skippable,
		); err != nil {
			return nil, fmt.Errorf(
				"读取Fork默认阶段记录失败: %w",
				err,
			)
		}

		snapshots = append(
			snapshots,
			snapshot,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历Fork默认阶段失败: %w",
			err,
		)
	}

	normalized, ok :=
		normalizeLessonPlanForkStageSnapshotsFromValues(
			snapshots,
		)
	if !ok {
		return nil, fmt.Errorf(
			"Fork默认阶段配置为空或非法",
		)
	}

	return normalized, nil
}

func normalizeLessonPlanForkStageSnapshotsFromValues(
	snapshots []models.StageConfigSnapshot,
) ([]models.StageConfigSnapshot, bool) {
	if len(snapshots) == 0 {
		return nil, false
	}

	seen := make(map[string]struct{}, len(snapshots))
	for index := range snapshots {
		snapshots[index].StageCode =
			strings.TrimSpace(
				snapshots[index].StageCode,
			)

		if snapshots[index].StageCode == "" ||
			snapshots[index].StageOrder <= 0 {
			return nil, false
		}

		if _, exists := seen[snapshots[index].StageCode]; exists {
			return nil, false
		}
		seen[snapshots[index].StageCode] = struct{}{}
	}

	sort.SliceStable(
		snapshots,
		func(left int, right int) bool {
			return snapshots[left].StageOrder <
				snapshots[right].StageOrder
		},
	)

	return snapshots, true
}

func createLessonPlanForkStageOutputsTx(
	ctx context.Context,
	tx pgx.Tx,
	lessonPlanID string,
	stageOutputs []models.WorkshopStageOutput,
) error {
	if strings.TrimSpace(lessonPlanID) == "" {
		return fmt.Errorf(
			"创建Fork阶段产出失败: 教案ID为空",
		)
	}
	if len(stageOutputs) == 0 {
		return fmt.Errorf(
			"创建Fork阶段产出失败: 阶段列表为空",
		)
	}

	for index := range stageOutputs {
		output := stageOutputs[index]

		structuredOutput := strings.TrimSpace(
			output.StructuredOutput,
		)
		if structuredOutput == "" {
			structuredOutput = "{}"
		}

		conversationSnapshot := strings.TrimSpace(
			output.ConversationSnapshot,
		)
		if conversationSnapshot == "" {
			conversationSnapshot = "[]"
		}

		var completedAt *time.Time
		if output.Status == models.StageOutputSkipped {
			now := time.Now()
			completedAt = &now
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO workshop_stage_outputs (
				lesson_plan_id,
				stage_code,
				stage_order,
				structured_output,
				narrative_output,
				conversation_snapshot,
				model_used,
				tokens_used,
				status,
				completed_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4::jsonb,
				$5,
				$6::jsonb,
				$7,
				$8,
				$9,
				$10
			)
		`,
			lessonPlanID,
			output.StageCode,
			output.StageOrder,
			structuredOutput,
			output.NarrativeOutput,
			conversationSnapshot,
			output.ModelUsed,
			output.TokensUsed,
			output.Status,
			completedAt,
		)
		if err != nil {
			return fmt.Errorf(
				"创建Fork阶段产出失败（%s）: %w",
				output.StageCode,
				err,
			)
		}
	}

	return nil
}
