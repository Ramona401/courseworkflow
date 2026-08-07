package repository

// lesson_plan_context_capsule_repo.go — 备课核心共识胶囊事务仓储
//
// 写入顺序：
//   1. 对lesson_plan_id取得事务级咨询锁；
//   2. FOR UPDATE读取当前胶囊；
//   3. 内容哈希相同则幂等返回，不产生无意义新版本；
//   4. 首次写入version=1，后续写入version=current+1；
//   5. 数据库触发器自动保存不可变版本；
//   6. 在同一事务内写入本版本的原文证据路由；
//   7. 提交后新版本才对运行时可见。
//
// 旁路更新失败不得破坏已经存在的active胶囊。错误记录只更新error_message，
// 不递增版本，也不把已有active胶囊降级为failed。

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrLessonPlanContextCapsuleNotFound        = errors.New("备课核心共识胶囊不存在")
	ErrLessonPlanContextCapsuleVersionConflict = errors.New("备课核心共识胶囊版本冲突")
	ErrLessonPlanContextCapsuleInvalid         = errors.New("备课核心共识胶囊数据无效")
)

const lessonPlanContextCapsuleSelectColumns = `
        id::text,
        lesson_plan_id::text,
        status,
        version,
        schema_version,
        current_stage_code,
        capsule_json::text,
        display_json::text,
        context_text,
        source_manifest::text,
        source_hash,
        last_turn_id,
        last_update_reason,
        error_message,
        generated_at,
        created_at,
        updated_at
`

// lessonPlanContextCapsuleSelectColumnsFromCapsule 用于包含JOIN的查询。
//
// 该查询必须显式限定主表别名；lesson_plans和胶囊表都包含id、created_at、
// updated_at等同名列，使用未限定列名会在PostgreSQL解析阶段直接报
// “column reference is ambiguous”，导致冷启动读取和旁路更新同时失效。
const lessonPlanContextCapsuleSelectColumnsFromCapsule = `
        capsule.id::text,
        capsule.lesson_plan_id::text,
        capsule.status,
        capsule.version,
        capsule.schema_version,
        capsule.current_stage_code,
        capsule.capsule_json::text,
        capsule.display_json::text,
        capsule.context_text,
        capsule.source_manifest::text,
        capsule.source_hash,
        capsule.last_turn_id,
        capsule.last_update_reason,
        capsule.error_message,
        capsule.generated_at,
        capsule.created_at,
        capsule.updated_at
`

// UpsertLessonPlanContextCapsuleInput 是一次完整新版本写入。
type UpsertLessonPlanContextCapsuleInput struct {
	LessonPlanID     string
	SchemaVersion    int
	CurrentStageCode string
	CapsuleJSON      string
	DisplayJSON      string
	ContextText      string
	SourceManifest   string
	SourceHash       string
	LastTurnID       string
	UpdateReason     string
	Evidence         []models.LessonPlanContextCapsuleEvidence
}

func scanLessonPlanContextCapsule(row pgx.Row) (*models.LessonPlanContextCapsule, error) {
	record := &models.LessonPlanContextCapsule{}
	var generatedAt, createdAt, updatedAt sql.NullTime

	err := row.Scan(
		&record.ID,
		&record.LessonPlanID,
		&record.Status,
		&record.Version,
		&record.SchemaVersion,
		&record.CurrentStageCode,
		&record.CapsuleJSON,
		&record.DisplayJSON,
		&record.ContextText,
		&record.SourceManifest,
		&record.SourceHash,
		&record.LastTurnID,
		&record.LastUpdateReason,
		&record.ErrorMessage,
		&generatedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if generatedAt.Valid {
		value := generatedAt.Time
		record.GeneratedAt = &value
	}
	if createdAt.Valid {
		value := createdAt.Time
		record.CreatedAt = &value
	}
	if updatedAt.Valid {
		value := updatedAt.Time
		record.UpdatedAt = &value
	}

	return record, nil
}

// GetLessonPlanContextCapsule 读取当前胶囊，无记录返回nil。
func GetLessonPlanContextCapsule(
	ctx context.Context,
	lessonPlanID string,
) (*models.LessonPlanContextCapsule, error) {
	record, err := scanLessonPlanContextCapsule(
		database.DB.QueryRow(
			ctx,
			`SELECT `+lessonPlanContextCapsuleSelectColumns+`
                         FROM lesson_plan_context_capsules
                         WHERE lesson_plan_id = $1`,
			strings.TrimSpace(lessonPlanID),
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取备课核心共识胶囊失败: %w", err)
	}

	return record, nil
}

// GetActiveLessonPlanContextCapsule 只返回可用于运行时的active胶囊。
func GetActiveLessonPlanContextCapsule(
	ctx context.Context,
	lessonPlanID string,
) (*models.LessonPlanContextCapsule, error) {
	record, err := scanLessonPlanContextCapsule(
		database.DB.QueryRow(
			ctx,
			`SELECT `+lessonPlanContextCapsuleSelectColumnsFromCapsule+`
                         FROM lesson_plan_context_capsules capsule
                         JOIN lesson_plans lesson_plan
                           ON lesson_plan.id = capsule.lesson_plan_id
                          AND lesson_plan.deleted_at IS NULL
                         WHERE capsule.lesson_plan_id = $1
                           AND capsule.status = 'active'`,
			strings.TrimSpace(lessonPlanID),
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取active备课核心共识胶囊失败: %w", err)
	}

	return record, nil
}

func validateLessonPlanContextCapsuleUpsertInput(
	input *UpsertLessonPlanContextCapsuleInput,
) error {
	if input == nil ||
		strings.TrimSpace(input.LessonPlanID) == "" ||
		input.SchemaVersion < 1 ||
		strings.TrimSpace(input.CapsuleJSON) == "" ||
		strings.TrimSpace(input.CapsuleJSON) == "{}" ||
		strings.TrimSpace(input.DisplayJSON) == "" ||
		strings.TrimSpace(input.ContextText) == "" ||
		strings.TrimSpace(input.SourceManifest) == "" ||
		strings.TrimSpace(input.SourceManifest) == "{}" ||
		len(strings.TrimSpace(input.SourceHash)) != 64 {
		return ErrLessonPlanContextCapsuleInvalid
	}

	for _, raw := range []string{
		input.CapsuleJSON,
		input.DisplayJSON,
		input.SourceManifest,
	} {
		var value map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return fmt.Errorf("%w: JSON解析失败: %v", ErrLessonPlanContextCapsuleInvalid, err)
		}
	}

	return nil
}

// UpsertActiveLessonPlanContextCapsule 原子写入一个完整active新版本。
func UpsertActiveLessonPlanContextCapsule(
	ctx context.Context,
	input *UpsertLessonPlanContextCapsuleInput,
) (*models.LessonPlanContextCapsule, bool, error) {
	if err := validateLessonPlanContextCapsuleUpsertInput(input); err != nil {
		return nil, false, err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("开始胶囊写入事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	lessonPlanID := strings.TrimSpace(input.LessonPlanID)

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		lessonPlanID,
	); err != nil {
		return nil, false, fmt.Errorf("取得胶囊写入事务锁失败: %w", err)
	}

	current, currentErr := scanLessonPlanContextCapsule(
		tx.QueryRow(
			ctx,
			`SELECT `+lessonPlanContextCapsuleSelectColumns+`
                         FROM lesson_plan_context_capsules
                         WHERE lesson_plan_id = $1
                         FOR UPDATE`,
			lessonPlanID,
		),
	)
	if errors.Is(currentErr, pgx.ErrNoRows) {
		current = nil
	} else if currentErr != nil {
		return nil, false, fmt.Errorf("锁定当前胶囊失败: %w", currentErr)
	}

	// 胶囊更新是异步旁路，相邻轮次可能乱序完成。
	//
	// 当active胶囊和待写入胶囊都具有可比较的轮次时间序号时，
	// 较旧轮次不得覆盖较新的active状态。这里在事务咨询锁和
	// FOR UPDATE行锁之后判断，保证并发写入时结论稳定。
	//
	// 被拦截的旧轮次按幂等成功返回，不记录错误、不产生版本，
	// 也不广播SSE更新。
	if current != nil &&
		current.Status == models.LessonPlanContextCapsuleStatusActive &&
		lessonPlanContextCapsuleIncomingTurnIsOlder(
			current.LastTurnID,
			input.LastTurnID,
		) {
		if err := tx.Commit(ctx); err != nil {
			return nil, false, fmt.Errorf(
				"提交胶囊旧轮次拦截事务失败: %w",
				err,
			)
		}

		return current, false, nil
	}

	// 语义内容没有变化时保持当前版本，避免每轮对话制造空版本。
	if current != nil &&
		current.Status == models.LessonPlanContextCapsuleStatusActive &&
		current.SourceHash == strings.TrimSpace(input.SourceHash) {
		if err := tx.Commit(ctx); err != nil {
			return nil, false, fmt.Errorf("提交胶囊幂等事务失败: %w", err)
		}
		return current, false, nil
	}

	version := 1
	if current != nil {
		version = current.Version + 1
	}

	now := time.Now()
	if current == nil {
		_, err = tx.Exec(
			ctx,
			`
                                INSERT INTO lesson_plan_context_capsules (
                                        id,
                                        lesson_plan_id,
                                        status,
                                        version,
                                        schema_version,
                                        current_stage_code,
                                        capsule_json,
                                        display_json,
                                        context_text,
                                        source_manifest,
                                        source_hash,
                                        last_turn_id,
                                        last_update_reason,
                                        error_message,
                                        generated_at,
                                        created_at,
                                        updated_at
                                )
                                VALUES (
                                        gen_random_uuid(),
                                        $1,
                                        'active',
                                        1,
                                        $2,
                                        $3,
                                        $4::jsonb,
                                        $5::jsonb,
                                        $6,
                                        $7::jsonb,
                                        $8,
                                        $9,
                                        $10,
                                        '',
                                        $11,
                                        $11,
                                        $11
                                )
                        `,
			lessonPlanID,
			input.SchemaVersion,
			strings.TrimSpace(input.CurrentStageCode),
			input.CapsuleJSON,
			input.DisplayJSON,
			input.ContextText,
			input.SourceManifest,
			input.SourceHash,
			strings.TrimSpace(input.LastTurnID),
			strings.TrimSpace(input.UpdateReason),
			now,
		)
	} else {
		tag, updateErr := tx.Exec(
			ctx,
			`
                                UPDATE lesson_plan_context_capsules
                                SET
                                        status = 'active',
                                        version = $2,
                                        schema_version = $3,
                                        current_stage_code = $4,
                                        capsule_json = $5::jsonb,
                                        display_json = $6::jsonb,
                                        context_text = $7,
                                        source_manifest = $8::jsonb,
                                        source_hash = $9,
                                        last_turn_id = $10,
                                        last_update_reason = $11,
                                        error_message = '',
                                        generated_at = $12,
                                        updated_at = $12
                                WHERE lesson_plan_id = $1
                                  AND version = $13
                        `,
			lessonPlanID,
			version,
			input.SchemaVersion,
			strings.TrimSpace(input.CurrentStageCode),
			input.CapsuleJSON,
			input.DisplayJSON,
			input.ContextText,
			input.SourceManifest,
			input.SourceHash,
			strings.TrimSpace(input.LastTurnID),
			strings.TrimSpace(input.UpdateReason),
			now,
			current.Version,
		)
		if updateErr != nil {
			return nil, false, fmt.Errorf("更新备课核心共识胶囊失败: %w", updateErr)
		}
		if tag.RowsAffected() != 1 {
			return nil, false, ErrLessonPlanContextCapsuleVersionConflict
		}
	}
	if err != nil {
		return nil, false, fmt.Errorf("写入备课核心共识胶囊失败: %w", err)
	}

	for _, evidence := range input.Evidence {
		locatorJSON, marshalErr := json.Marshal(evidence.Locator)
		if marshalErr != nil {
			return nil, false, fmt.Errorf("序列化胶囊证据定位信息失败: %w", marshalErr)
		}

		if _, insertErr := tx.Exec(
			ctx,
			`
                                INSERT INTO lesson_plan_context_capsule_evidence (
                                        id,
                                        lesson_plan_id,
                                        capsule_version,
                                        item_key,
                                        source_type,
                                        source_id,
                                        source_title,
                                        locator_json,
                                        source_hash,
                                        excerpt_hash,
                                        evidence_excerpt,
                                        authority,
                                        created_at
                                )
                                VALUES (
                                        gen_random_uuid(),
                                        $1,
                                        $2,
                                        $3,
                                        $4,
                                        $5,
                                        $6,
                                        $7::jsonb,
                                        $8,
                                        $9,
                                        $10,
                                        $11,
                                        NOW()
                                )
                                ON CONFLICT (
                                        lesson_plan_id,
                                        capsule_version,
                                        item_key,
                                        source_type,
                                        source_id,
                                        excerpt_hash
                                )
                                DO NOTHING
                        `,
			lessonPlanID,
			version,
			strings.TrimSpace(evidence.ItemKey),
			strings.TrimSpace(evidence.SourceType),
			strings.TrimSpace(evidence.SourceID),
			strings.TrimSpace(evidence.SourceTitle),
			string(locatorJSON),
			strings.TrimSpace(evidence.SourceHash),
			strings.TrimSpace(evidence.ExcerptHash),
			strings.TrimSpace(evidence.EvidenceExcerpt),
			strings.TrimSpace(evidence.Authority),
		); insertErr != nil {
			return nil, false, fmt.Errorf("写入胶囊原文证据路由失败: %w", insertErr)
		}
	}

	saved, err := scanLessonPlanContextCapsule(
		tx.QueryRow(
			ctx,
			`SELECT `+lessonPlanContextCapsuleSelectColumns+`
                         FROM lesson_plan_context_capsules
                         WHERE lesson_plan_id = $1`,
			lessonPlanID,
		),
	)
	if err != nil {
		return nil, false, fmt.Errorf("回读新胶囊版本失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("提交胶囊写入事务失败: %w", err)
	}

	return saved, true, nil
}

// RecordLessonPlanContextCapsuleUpdateError 记录旁路更新错误，但不破坏已有active版本。
func RecordLessonPlanContextCapsuleUpdateError(
	ctx context.Context,
	lessonPlanID string,
	errorMessage string,
) error {
	errorMessage = strings.TrimSpace(errorMessage)
	if len([]rune(errorMessage)) > 4000 {
		errorMessage = string([]rune(errorMessage)[:4000])
	}

	_, err := database.DB.Exec(
		ctx,
		`
                        UPDATE lesson_plan_context_capsules
                        SET
                                status = CASE
                                        WHEN status = 'active' THEN 'active'
                                        ELSE 'failed'
                                END,
                                error_message = $2,
                                updated_at = NOW()
                        WHERE lesson_plan_id = $1
                `,
		strings.TrimSpace(lessonPlanID),
		errorMessage,
	)
	if err != nil {
		return fmt.Errorf("记录胶囊旁路更新错误失败: %w", err)
	}

	return nil
}

// MarkLessonPlanContextCapsuleStale 主动使当前胶囊失效。
func MarkLessonPlanContextCapsuleStale(
	ctx context.Context,
	lessonPlanID string,
	reason string,
) error {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) > 4000 {
		reason = string([]rune(reason)[:4000])
	}

	_, err := database.DB.Exec(
		ctx,
		`
                        UPDATE lesson_plan_context_capsules
                        SET
                                status = 'stale',
                                error_message = $2,
                                updated_at = NOW()
                        WHERE lesson_plan_id = $1
                `,
		strings.TrimSpace(lessonPlanID),
		reason,
	)
	if err != nil {
		return fmt.Errorf("标记备课核心共识胶囊失效失败: %w", err)
	}

	return nil
}
