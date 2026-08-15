package repository

// courseware_assembly_repo.go — 课件自动装配数据库业务版本与布局验收仓储
//
// 本仓储是进程级BackgroundTaskTracker之外的数据库业务防线。
//
// 进程级Tracker解决：
//   - 同一进程内任务去重；
//   - 部署排空和停止继续派发；
//   - goroutine完成、panic恢复和运维观察。
//
// 本仓储解决：
//   - 每次装配获得单调递增版本；
//   - 同一课件数据库层最多一个活动运行；
//   - 取消后页面写回立即失效；
//   - 新版本领取后旧任务不能覆盖新页面；
//   - HTML写入后旧布局报告自动重置；
//   - 服务重启时把旧进程遗留活动状态收敛为interrupted。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrCoursewareAssemblyNotFound 合并课件不存在和装配运行不存在。
	ErrCoursewareAssemblyNotFound = errors.New(
		"课件装配记录不存在",
	)

	// ErrCoursewareAssemblyAlreadyRunning 表示同一课件已有活动装配运行。
	ErrCoursewareAssemblyAlreadyRunning = errors.New(
		"课件已有装配任务正在运行",
	)

	// ErrCoursewareAssemblyVersionConflict 表示运行版本、RunID或状态已经失效。
	ErrCoursewareAssemblyVersionConflict = errors.New(
		"课件装配版本已经失效",
	)

	// ErrCoursewareAssemblyInvalid 表示调用参数或状态不符合装配契约。
	ErrCoursewareAssemblyInvalid = errors.New(
		"课件装配参数无效",
	)
)

// CoursewareAssemblyPageWrite 是装配任务写回单页HTML的完整身份。
// CoursewareID、Version和RunID必须与数据库当前活动运行完全一致。
type CoursewareAssemblyPageWrite struct {
	PageID              string
	CoursewareID        string
	Version             int64
	RunID               string
	HTMLContent         string
	PlaceholderMap      string
	MatchedComponentIDs string
	PageStatus          string
}

// CoursewareAssemblyLayoutWrite 是浏览器布局验收结果写回身份。
type CoursewareAssemblyLayoutWrite struct {
	PageID       string
	CoursewareID string
	Version      int64
	RunID        string
	LayoutStatus string
	HTMLHash     string
	AuditJSON    string
}

// normalizeCoursewareAssemblyJSON 校验并规范装配JSON对象。
func normalizeCoursewareAssemblyJSON(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "{}", nil
	}

	if !json.Valid([]byte(value)) {
		return "", fmt.Errorf(
			"%w: JSON格式错误",
			ErrCoursewareAssemblyInvalid,
		)
	}

	var object map[string]interface{}
	if err := json.Unmarshal(
		[]byte(value),
		&object,
	); err != nil || object == nil {
		return "", fmt.Errorf(
			"%w: JSON必须是对象",
			ErrCoursewareAssemblyInvalid,
		)
	}

	normalized, err := json.Marshal(object)
	if err != nil {
		return "", fmt.Errorf(
			"%w: JSON规范化失败",
			ErrCoursewareAssemblyInvalid,
		)
	}

	return string(normalized), nil
}

// BeginCoursewareAssembly 原子领取一个新的课件装配版本。
//
// 事务顺序：
//  1. FOR UPDATE锁定课件；
//  2. 拒绝仍处于running/cancel_requested的活动运行；
//  3. assembly_version加一；
//  4. 创建不可混淆的运行记录；
//  5. 把课件当前活动RunID指向新运行；
//  6. 提交事务。
func BeginCoursewareAssembly(
	ctx context.Context,
	coursewareID string,
	startedBy string,
	skipVideo bool,
) (*models.CoursewareAssemblyRun, error) {
	return beginCoursewareAssemblyWithMetadata(
		ctx,
		coursewareID,
		startedBy,
		skipVideo,
		"{}",
	)
}

// BeginCoursewareGenerationRun 原子领取带R-04完整性快照metadata的新生成运行。
//
// 与BeginCoursewareAssembly共用同一数据库版本与互斥边界，区别仅在于调用方可在
// 运行记录首次可见时就写入run_kind和稳定页面方案快照，避免状态查询短暂把batch误判为assembly。
func BeginCoursewareGenerationRun(
	ctx context.Context,
	coursewareID string,
	startedBy string,
	skipVideo bool,
	initialMetadataJSON string,
) (*models.CoursewareAssemblyRun, error) {
	return beginCoursewareAssemblyWithMetadata(
		ctx,
		coursewareID,
		startedBy,
		skipVideo,
		initialMetadataJSON,
	)
}

func beginCoursewareAssemblyWithMetadata(
	ctx context.Context,
	coursewareID string,
	startedBy string,
	skipVideo bool,
	initialMetadataJSON string,
) (*models.CoursewareAssemblyRun, error) {
	coursewareID = strings.TrimSpace(coursewareID)
	startedBy = strings.TrimSpace(startedBy)

	if coursewareID == "" || startedBy == "" {
		return nil, fmt.Errorf(
			"%w: 缺少课件或操作者",
			ErrCoursewareAssemblyInvalid,
		)
	}

	normalizedInitialMetadata, err :=
		normalizeCoursewareAssemblyJSON(
			initialMetadataJSON,
		)
	if err != nil {
		return nil, err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启课件装配事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var currentVersion int64
	var currentStatus string

	lockErr := tx.QueryRow(
		ctx,
		`SELECT
                        assembly_version,
                        assembly_status
                FROM coursewares
                WHERE id = $1
                  AND deleted_at IS NULL
                FOR UPDATE`,
		coursewareID,
	).Scan(
		&currentVersion,
		&currentStatus,
	)
	if lockErr != nil {
		if errors.Is(lockErr, pgx.ErrNoRows) {
			return nil, ErrCoursewareAssemblyNotFound
		}

		return nil, fmt.Errorf(
			"锁定课件装配状态失败: %w",
			lockErr,
		)
	}

	if currentStatus == models.CoursewareAssemblyStatusRunning ||
		currentStatus == models.CoursewareAssemblyStatusCancelRequested {
		return nil, ErrCoursewareAssemblyAlreadyRunning
	}

	nextVersion := currentVersion + 1
	run := &models.CoursewareAssemblyRun{}

	insertErr := tx.QueryRow(
		ctx,
		`INSERT INTO courseware_assembly_runs (
                        courseware_id,
                        version,
                        started_by,
                        skip_video,
                        status,
                        metadata
                )
                VALUES (
                        $1,
                        $2,
                        $3,
                        $4,
                        'running',
                        $5::jsonb
                )
                RETURNING
                        id,
                        courseware_id,
                        version,
                        started_by,
                        skip_video,
                        status,
                        error_message,
                        metadata::text,
                        started_at,
                        updated_at,
                        finished_at`,
		coursewareID,
		nextVersion,
		startedBy,
		skipVideo,
		normalizedInitialMetadata,
	).Scan(
		&run.ID,
		&run.CoursewareID,
		&run.Version,
		&run.StartedBy,
		&run.SkipVideo,
		&run.Status,
		&run.ErrorMessage,
		&run.MetadataJSON,
		&run.StartedAt,
		&run.UpdatedAt,
		&run.FinishedAt,
	)
	if insertErr != nil {
		return nil, fmt.Errorf(
			"创建课件装配运行失败: %w",
			insertErr,
		)
	}

	tag, updateErr := tx.Exec(
		ctx,
		`UPDATE coursewares
                SET
                        assembly_version = $1,
                        assembly_status = 'running',
                        active_assembly_run_id = $2,
                        assembly_started_at = NOW(),
                        assembly_finished_at = NULL,
                        assembly_started_by = $3,
                        assembly_skip_video = $4,
                        updated_at = NOW()
                WHERE id = $5
                  AND deleted_at IS NULL`,
		nextVersion,
		run.ID,
		startedBy,
		skipVideo,
		coursewareID,
	)
	if updateErr != nil {
		return nil, fmt.Errorf(
			"更新课件活动装配状态失败: %w",
			updateErr,
		)
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrCoursewareAssemblyVersionConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交课件装配事务失败: %w",
			err,
		)
	}

	return run, nil
}

// RecoverInterruptedCoursewareAssemblies 把旧进程遗留活动运行收敛为interrupted。
//
// 该函数应在新进程完成数据库初始化后、接受外部装配请求前调用。
// 单实例systemd环境中，新进程启动时旧进程已经停止，因此数据库中仍为
// running/cancel_requested的运行均属于旧进程遗留。
func RecoverInterruptedCoursewareAssemblies(
	ctx context.Context,
) (int64, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"开启装配恢复事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_assembly_runs
                SET
                        status = 'interrupted',
                        error_message = CASE
                                WHEN BTRIM(error_message) = ''
                                        THEN '服务重启，旧进程装配运行已中断'
                                ELSE error_message
                        END,
                        updated_at = NOW(),
                        finished_at = COALESCE(finished_at, NOW())
                WHERE status IN ('running', 'cancel_requested')`,
	); err != nil {
		return 0, fmt.Errorf(
			"收敛旧装配运行失败: %w",
			err,
		)
	}

	tag, err := tx.Exec(
		ctx,
		`UPDATE coursewares
                SET
                        assembly_status = 'interrupted',
                        active_assembly_run_id = NULL,
                        assembly_finished_at = COALESCE(
                                assembly_finished_at,
                                NOW()
                        ),
                        updated_at = NOW()
                WHERE assembly_status IN (
                        'running',
                        'cancel_requested'
                )`,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"收敛课件活动装配状态失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(
			"提交装配恢复事务失败: %w",
			err,
		)
	}

	return tag.RowsAffected(), nil
}

// GetCoursewareAssemblyState 读取课件当前装配状态。
func GetCoursewareAssemblyState(
	ctx context.Context,
	coursewareID string,
) (*models.CoursewareAssemblyState, error) {
	coursewareID = strings.TrimSpace(coursewareID)
	if coursewareID == "" {
		return nil, ErrCoursewareAssemblyInvalid
	}

	state := &models.CoursewareAssemblyState{
		CoursewareID: coursewareID,
	}

	err := database.DB.QueryRow(
		ctx,
		`SELECT
                        assembly_version,
                        assembly_status,
                        active_assembly_run_id,
                        assembly_started_by,
                        assembly_skip_video,
                        assembly_started_at,
                        assembly_finished_at
                FROM coursewares
                WHERE id = $1
                  AND deleted_at IS NULL`,
		coursewareID,
	).Scan(
		&state.Version,
		&state.Status,
		&state.ActiveRunID,
		&state.StartedBy,
		&state.SkipVideo,
		&state.StartedAt,
		&state.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareAssemblyNotFound
		}

		return nil, fmt.Errorf(
			"读取课件装配状态失败: %w",
			err,
		)
	}

	runKind, integrity, integrityErr :=
		ReadCoursewareGenerationIntegrity(
			ctx,
			coursewareID,
			state.Version,
		)
	if integrityErr != nil {
		return nil, fmt.Errorf(
			"读取课件生成完整性状态失败: %w",
			integrityErr,
		)
	}

	state.RunKind = runKind
	state.Integrity = integrity

	return state, nil
}

// RequestCoursewareAssemblyCancel 原子请求取消当前装配版本。
//
// 状态一旦变为cancel_requested，版本化页面写回SQL会立即拒绝后续HTML写入。
func RequestCoursewareAssemblyCancel(
	ctx context.Context,
	coursewareID string,
	version int64,
	runID string,
) error {
	coursewareID = strings.TrimSpace(coursewareID)
	runID = strings.TrimSpace(runID)

	if coursewareID == "" || version <= 0 || runID == "" {
		return ErrCoursewareAssemblyInvalid
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启取消装配事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(
		ctx,
		`UPDATE coursewares
                SET
                        assembly_status = 'cancel_requested',
                        updated_at = NOW()
                WHERE id = $1
                  AND assembly_version = $2
                  AND active_assembly_run_id = $3
                  AND assembly_status = 'running'`,
		coursewareID,
		version,
		runID,
	)
	if err != nil {
		return fmt.Errorf(
			"请求取消课件装配失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareAssemblyVersionConflict
	}

	tag, err = tx.Exec(
		ctx,
		`UPDATE courseware_assembly_runs
                SET
                        status = 'cancel_requested',
                        updated_at = NOW()
                WHERE id = $1
                  AND courseware_id = $2
                  AND version = $3
                  AND status = 'running'`,
		runID,
		coursewareID,
		version,
	)
	if err != nil {
		return fmt.Errorf(
			"更新装配运行取消状态失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareAssemblyVersionConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交取消装配事务失败: %w",
			err,
		)
	}

	return nil
}

// FinishCoursewareAssembly 原子结束当前装配运行。
func FinishCoursewareAssembly(
	ctx context.Context,
	coursewareID string,
	version int64,
	runID string,
	finalStatus string,
	errorMessage string,
	metadataJSON string,
) error {
	coursewareID = strings.TrimSpace(coursewareID)
	runID = strings.TrimSpace(runID)
	finalStatus = strings.TrimSpace(finalStatus)
	errorMessage = strings.TrimSpace(errorMessage)

	if coursewareID == "" ||
		version <= 0 ||
		runID == "" ||
		!models.IsCoursewareAssemblyFinalStatus(finalStatus) {
		return ErrCoursewareAssemblyInvalid
	}

	normalizedMetadata, err :=
		normalizeCoursewareAssemblyJSON(metadataJSON)
	if err != nil {
		return err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启完成装配事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var storedVersion int64
	var storedStatus string
	var storedRunID *string

	err = tx.QueryRow(
		ctx,
		`SELECT
                        assembly_version,
                        assembly_status,
                        active_assembly_run_id
                FROM coursewares
                WHERE id = $1
                  AND deleted_at IS NULL
                FOR UPDATE`,
		coursewareID,
	).Scan(
		&storedVersion,
		&storedStatus,
		&storedRunID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareAssemblyNotFound
		}

		return fmt.Errorf(
			"锁定完成装配状态失败: %w",
			err,
		)
	}

	if storedVersion != version ||
		storedRunID == nil ||
		*storedRunID != runID ||
		(storedStatus != models.CoursewareAssemblyStatusRunning &&
			storedStatus != models.CoursewareAssemblyStatusCancelRequested) {
		return ErrCoursewareAssemblyVersionConflict
	}

	tag, err := tx.Exec(
		ctx,
		`UPDATE courseware_assembly_runs
                SET
                        status = $1,
                        error_message = $2,
                        metadata = metadata || $3::jsonb,
                        updated_at = NOW(),
                        finished_at = NOW()
                WHERE id = $4
                  AND courseware_id = $5
                  AND version = $6
                  AND status IN ('running', 'cancel_requested')`,
		finalStatus,
		errorMessage,
		normalizedMetadata,
		runID,
		coursewareID,
		version,
	)
	if err != nil {
		return fmt.Errorf(
			"结束装配运行失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareAssemblyVersionConflict
	}

	tag, err = tx.Exec(
		ctx,
		`UPDATE coursewares
                SET
                        assembly_status = $1,
                        active_assembly_run_id = NULL,
                        assembly_finished_at = NOW(),
                        updated_at = NOW()
                WHERE id = $2
                  AND assembly_version = $3
                  AND active_assembly_run_id = $4`,
		finalStatus,
		coursewareID,
		version,
		runID,
	)
	if err != nil {
		return fmt.Errorf(
			"结束课件活动装配状态失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareAssemblyVersionConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交完成装配事务失败: %w",
			err,
		)
	}

	return nil
}

// UpdateCWPageHTMLForAssembly 仅允许当前活动装配运行写入页面HTML。
//
// 每次HTML发生变化时同步：
//   - 写入页面assembly_version；
//   - 重置layout_status为unchecked；
//   - 清空旧布局报告、HTML哈希和检查时间。
func UpdateCWPageHTMLForAssembly(
	ctx context.Context,
	input CoursewareAssemblyPageWrite,
) error {
	input.PageID = strings.TrimSpace(input.PageID)
	input.CoursewareID = strings.TrimSpace(input.CoursewareID)
	input.RunID = strings.TrimSpace(input.RunID)
	input.PageStatus = strings.TrimSpace(input.PageStatus)

	if input.PageID == "" ||
		input.CoursewareID == "" ||
		input.RunID == "" ||
		input.Version <= 0 ||
		strings.TrimSpace(input.HTMLContent) == "" {
		return ErrCoursewareAssemblyInvalid
	}

	tag, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_pages AS page
                SET
                        html_content = $1,
                        placeholder_map = NULLIF($2, '')::jsonb,
                        matched_component_ids = NULLIF($3, '')::jsonb,
                        status = $4,
                        assembly_version = $5,
                        layout_status = 'unchecked',
                        layout_audit_json = '{}'::jsonb,
                        layout_html_hash = NULL,
                        layout_checked_at = NULL,
                        updated_at = NOW()
                FROM coursewares AS courseware
                WHERE page.id = $6
                  AND page.courseware_id = $7
                  AND courseware.id = page.courseware_id
                  AND courseware.assembly_version = $5
                  AND courseware.assembly_status = 'running'
                  AND courseware.active_assembly_run_id = $8`,
		input.HTMLContent,
		input.PlaceholderMap,
		input.MatchedComponentIDs,
		input.PageStatus,
		input.Version,
		input.PageID,
		input.CoursewareID,
		input.RunID,
	)
	if err != nil {
		return fmt.Errorf(
			"按装配版本写回页面HTML失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareAssemblyVersionConflict
	}

	return nil
}

// UpdateCoursewarePageLayoutAuditForAssembly 写回当前装配版本的浏览器布局验收结果。
func UpdateCoursewarePageLayoutAuditForAssembly(
	ctx context.Context,
	input CoursewareAssemblyLayoutWrite,
) error {
	input.PageID = strings.TrimSpace(input.PageID)
	input.CoursewareID = strings.TrimSpace(input.CoursewareID)
	input.RunID = strings.TrimSpace(input.RunID)
	input.LayoutStatus = strings.TrimSpace(input.LayoutStatus)
	input.HTMLHash = strings.ToLower(
		strings.TrimSpace(input.HTMLHash),
	)

	if input.PageID == "" ||
		input.CoursewareID == "" ||
		input.RunID == "" ||
		input.Version <= 0 ||
		!models.IsValidCoursewareLayoutStatus(input.LayoutStatus) {
		return ErrCoursewareAssemblyInvalid
	}

	if len(input.HTMLHash) != 64 {
		return fmt.Errorf(
			"%w: HTML哈希必须为64位SHA-256十六进制文本",
			ErrCoursewareAssemblyInvalid,
		)
	}

	for _, character := range input.HTMLHash {
		if !strings.ContainsRune(
			"0123456789abcdef",
			character,
		) {
			return fmt.Errorf(
				"%w: HTML哈希包含非法字符",
				ErrCoursewareAssemblyInvalid,
			)
		}
	}

	normalizedAudit, err :=
		normalizeCoursewareAssemblyJSON(input.AuditJSON)
	if err != nil {
		return err
	}

	tag, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_pages AS page
                SET
                        layout_status = $1,
                        layout_audit_json = $2::jsonb,
                        layout_html_hash = $3,
                        layout_checked_at = NOW(),
                        updated_at = NOW()
                FROM coursewares AS courseware
                WHERE page.id = $4
                  AND page.courseware_id = $5
                  AND page.assembly_version = $6
                  AND courseware.id = page.courseware_id
                  AND courseware.assembly_version = $6
                  AND courseware.assembly_status = 'running'
                  AND courseware.active_assembly_run_id = $7`,
		input.LayoutStatus,
		normalizedAudit,
		input.HTMLHash,
		input.PageID,
		input.CoursewareID,
		input.Version,
		input.RunID,
	)
	if err != nil {
		return fmt.Errorf(
			"写回页面布局验收结果失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareAssemblyVersionConflict
	}

	return nil
}

// IsCoursewareAssemblyVersionCurrent 判断运行身份是否仍为当前活动装配。
func IsCoursewareAssemblyVersionCurrent(
	ctx context.Context,
	coursewareID string,
	version int64,
	runID string,
) (bool, error) {
	coursewareID = strings.TrimSpace(coursewareID)
	runID = strings.TrimSpace(runID)

	if coursewareID == "" || version <= 0 || runID == "" {
		return false, ErrCoursewareAssemblyInvalid
	}

	var current bool
	err := database.DB.QueryRow(
		ctx,
		`SELECT EXISTS (
                        SELECT 1
                        FROM coursewares
                        WHERE id = $1
                          AND deleted_at IS NULL
                          AND assembly_version = $2
                          AND active_assembly_run_id = $3
                          AND assembly_status = 'running'
                )`,
		coursewareID,
		version,
		runID,
	).Scan(&current)
	if err != nil {
		return false, fmt.Errorf(
			"检查课件装配版本失败: %w",
			err,
		)
	}

	return current, nil
}
