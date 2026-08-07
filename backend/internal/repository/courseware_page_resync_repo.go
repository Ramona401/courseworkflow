package repository

// courseware_page_resync_repo.go — 课件页号安全重排、严格校准与指定位置原子插页。
//
// courseware_pages存在UNIQUE(courseware_id, page_number)约束。
// 直接逐页修改页码会在处理中途撞号，因此统一使用“两阶段避撞”：
//   1. 在事务内把现有页码整体移动到高位临时区；
//   2. 按最终顺序写回1至N。
//
// 并发安全：
//   - 同一课件的插页、拖拽排序和页码校准共用事务级咨询锁；
//   - 页面行通过FOR UPDATE锁定；
//   - 严格校准写HTML前核对页面updated_at快照，避免旧HTML覆盖新修改。
//
// 完整性：
//   所有重排请求必须完整、无重复地包含数据库中的全部页面。
//   夹带其他课件页面、遗漏页面或重复页面均拒绝执行。
//
// 严格校准会在同一事务中完成：
//   - page_number连续重排；
//   - 导航栏HTML页码更新；
//   - coursewares.page_count总数更新。
//
// 任一步失败均整体回滚，不返回“部分校准成功”。

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tedna/internal/database"
	"tedna/internal/models"
)

// cwPageNumOffset是两阶段重排使用的临时页码偏移量。
// 正常课件不可能达到该页数，能够安全避开最终的1至N区间。
const cwPageNumOffset = 100000

// lockedCoursewarePage是事务锁定后读取的页面并发快照。
//
// UpdatedAt用于判断服务层计算导航HTML之后，页面内容是否又被其他操作修改。
// 一旦发生变化，本次严格校准整体回滚，要求调用方刷新页面后重试。
type lockedCoursewarePage struct {
	ID        string
	UpdatedAt time.Time
}

// CoursewarePageCalibrationHTML描述一页需要写入的校准后HTML。
//
// ExpectedUpdatedAt是服务层读取该页面时的更新时间快照。
// 仓储事务取得行锁后会核对该快照，防止覆盖刚发生的源码编辑、AI微调或重构结果。
type CoursewarePageCalibrationHTML struct {
	HTMLContent      string
	ExpectedUpdatedAt time.Time
}

// lockCoursewarePageSequenceTx为指定课件取得事务级咨询锁。
//
// hashtext返回int4，此处显式转换为bigint，匹配pg_advisory_xact_lock(bigint)。
// 锁会在事务提交或回滚时自动释放。
func lockCoursewarePageSequenceTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
) error {
	if coursewareID == "" {
		return fmt.Errorf("缺少课件ID")
	}

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1)::bigint)`,
		coursewareID,
	); err != nil {
		return fmt.Errorf("取得课件页码事务锁失败: %w", err)
	}

	return nil
}

// loadLockedCoursewarePagesTx锁定并读取课件全部页面。
//
// 返回顺序严格按当前page_number升序。
// FOR UPDATE确保事务期间这些页面不会再被其他写事务同时修改。
func loadLockedCoursewarePagesTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
) ([]lockedCoursewarePage, error) {
	rows, err := tx.Query(
		ctx,
		`SELECT id, updated_at
FROM courseware_pages
WHERE courseware_id = $1
ORDER BY page_number ASC, id ASC
FOR UPDATE`,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf("锁定课件页面失败: %w", err)
	}
	defer rows.Close()

	pages := make(
		[]lockedCoursewarePage,
		0,
	)

	for rows.Next() {
		var page lockedCoursewarePage

		if err := rows.Scan(
			&page.ID,
			&page.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"读取课件页面快照失败: %w",
				err,
			)
		}

		pages = append(
			pages,
			page,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件页面快照失败: %w",
			err,
		)
	}

	return pages, nil
}

// validateOrderedCoursewarePageIDs校验目标页面顺序是否完整且全部属于当前课件。
//
// 返回页面ID到事务内最新快照的映射，供HTML并发校验使用。
func validateOrderedCoursewarePageIDs(
	currentPages []lockedCoursewarePage,
	orderedPageIDs []string,
) (map[string]lockedCoursewarePage, error) {
	if len(orderedPageIDs) != len(currentPages) {
		return nil, fmt.Errorf(
			"页面顺序数据不完整：数据库有%d页，请求包含%d页",
			len(currentPages),
			len(orderedPageIDs),
		)
	}

	currentByID := make(
		map[string]lockedCoursewarePage,
		len(currentPages),
	)

	for _, page := range currentPages {
		currentByID[page.ID] = page
	}

	requestSet := make(
		map[string]struct{},
		len(orderedPageIDs),
	)

	for _, pageID := range orderedPageIDs {
		if pageID == "" {
			return nil, fmt.Errorf(
				"页面顺序数据包含空页面ID",
			)
		}

		if _, duplicated := requestSet[pageID]; duplicated {
			return nil, fmt.Errorf(
				"页面顺序数据包含重复页面ID: %s",
				pageID,
			)
		}

		if _, belongs := currentByID[pageID]; !belongs {
			return nil, fmt.Errorf(
				"页面不属于当前课件或已经不存在: %s",
				pageID,
			)
		}

		requestSet[pageID] = struct{}{}
	}

	return currentByID, nil
}

// moveCoursewarePagesToTemporaryRangeTx把全部现有页码移入高位临时区。
//
// 第一阶段只修改page_number，不修改updated_at。
// 这样严格校准仍能使用原updated_at快照判断HTML是否在准备期间发生变化。
func moveCoursewarePagesToTemporaryRangeTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
) error {
	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_pages
SET page_number = page_number + $1
WHERE courseware_id = $2`,
		cwPageNumOffset,
		coursewareID,
	); err != nil {
		return fmt.Errorf(
			"页号重排第一阶段偏移失败: %w",
			err,
		)
	}

	return nil
}

// updateCoursewarePageCountTx在当前事务中同步课件总页数。
func updateCoursewarePageCountTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	totalPages int,
	now time.Time,
) error {
	commandTag, err := tx.Exec(
		ctx,
		`UPDATE coursewares
SET page_count = $1,
    updated_at = $2
WHERE id = $3
  AND deleted_at IS NULL`,
		totalPages,
		now,
		coursewareID,
	)
	if err != nil {
		return fmt.Errorf(
			"同步课件总页数失败: %w",
			err,
		)
	}

	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf(
			"课件不存在或已被删除",
		)
	}

	return nil
}

// ResequenceCoursewarePagesByIDs在单个事务内按照orderedPageIDs指定的顺序，
// 将课件页号安全重排为1至N，并同步coursewares.page_count。
//
// 本函数只处理页号和总数，不修改页面HTML。
// 需要同时更新导航栏HTML时，应调用ApplyCoursewarePageCalibration。
func ResequenceCoursewarePagesByIDs(
	ctx context.Context,
	coursewareID string,
	orderedPageIDs []string,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启页号重排事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockCoursewarePageSequenceTx(
		ctx,
		tx,
		coursewareID,
	); err != nil {
		return err
	}

	currentPages, err :=
		loadLockedCoursewarePagesTx(
			ctx,
			tx,
			coursewareID,
		)
	if err != nil {
		return err
	}

	if _, err := validateOrderedCoursewarePageIDs(
		currentPages,
		orderedPageIDs,
	); err != nil {
		return err
	}

	now := time.Now()

	if len(currentPages) > 0 {
		if err := moveCoursewarePagesToTemporaryRangeTx(
			ctx,
			tx,
			coursewareID,
		); err != nil {
			return err
		}
	}

	for index, pageID := range orderedPageIDs {
		commandTag, err := tx.Exec(
			ctx,
			`UPDATE courseware_pages
SET page_number = $1,
    updated_at = $2
WHERE id = $3
  AND courseware_id = $4`,
			index+1,
			now,
			pageID,
			coursewareID,
		)
		if err != nil {
			return fmt.Errorf(
				"页号重排第二阶段落位失败(page_id=%s): %w",
				pageID,
				err,
			)
		}

		if commandTag.RowsAffected() != 1 {
			return fmt.Errorf(
				"页号重排期间页面已变化或不存在: %s",
				pageID,
			)
		}
	}

	if err := updateCoursewarePageCountTx(
		ctx,
		tx,
		coursewareID,
		len(orderedPageIDs),
		now,
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交页号重排事务失败: %w",
			err,
		)
	}

	return nil
}

// ApplyCoursewarePageCalibration严格应用一次完整页码校准。
//
// orderedPageIDs表示最终页面顺序。
// htmlByPageID只包含确实需要改写导航页码的页面；
// 没有出现在映射中的页面只重排页号，不覆盖HTML。
//
// 以下内容在同一事务内完成：
//   - 验证页面集合完整；
//   - 校验待写HTML页面的updated_at快照；
//   - 将页面页号重排为1至N；
//   - 写入校准后的导航栏HTML；
//   - 将coursewares.page_count更新为N。
func ApplyCoursewarePageCalibration(
	ctx context.Context,
	coursewareID string,
	orderedPageIDs []string,
	htmlByPageID map[string]CoursewarePageCalibrationHTML,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启页码校准事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockCoursewarePageSequenceTx(
		ctx,
		tx,
		coursewareID,
	); err != nil {
		return err
	}

	currentPages, err :=
		loadLockedCoursewarePagesTx(
			ctx,
			tx,
			coursewareID,
		)
	if err != nil {
		return err
	}

	currentByID, err :=
		validateOrderedCoursewarePageIDs(
			currentPages,
			orderedPageIDs,
		)
	if err != nil {
		return err
	}

	// 必须在移动页码之前检查HTML快照。
	// 此时行已由FOR UPDATE锁定，检查通过后不会再被其他事务修改。
	for pageID, htmlUpdate :=
		range htmlByPageID {
		currentPage, exists :=
			currentByID[pageID]

		if !exists {
			return fmt.Errorf(
				"页码校准HTML映射包含无效页面ID: %s",
				pageID,
			)
		}

		if htmlUpdate.ExpectedUpdatedAt.IsZero() {
			return fmt.Errorf(
				"页码校准缺少页面更新时间快照: %s",
				pageID,
			)
		}

		if !currentPage.UpdatedAt.Equal(
			htmlUpdate.ExpectedUpdatedAt,
		) {
			return fmt.Errorf(
				"页面内容已在校准准备期间发生变化，请刷新后重试(page_id=%s)",
				pageID,
			)
		}
	}

	now := time.Now()

	if len(currentPages) > 0 {
		if err := moveCoursewarePagesToTemporaryRangeTx(
			ctx,
			tx,
			coursewareID,
		); err != nil {
			return err
		}
	}

	for index, pageID := range orderedPageIDs {
		htmlUpdate, updateHTML :=
			htmlByPageID[pageID]

		var commandTag pgconn.CommandTag

		if updateHTML {
			commandTag, err = tx.Exec(
				ctx,
				`UPDATE courseware_pages
SET page_number = $1,
    html_content = $2,
    updated_at = $3
WHERE id = $4
  AND courseware_id = $5`,
				index+1,
				htmlUpdate.HTMLContent,
				now,
				pageID,
				coursewareID,
			)
		} else {
			commandTag, err = tx.Exec(
				ctx,
				`UPDATE courseware_pages
SET page_number = $1,
    updated_at = $2
WHERE id = $3
  AND courseware_id = $4`,
				index+1,
				now,
				pageID,
				coursewareID,
			)
		}

		if err != nil {
			return fmt.Errorf(
				"应用页码校准失败(page_id=%s): %w",
				pageID,
				err,
			)
		}

		if commandTag.RowsAffected() != 1 {
			return fmt.Errorf(
				"页码校准期间页面已变化或不存在: %s",
				pageID,
			)
		}
	}

	if err := updateCoursewarePageCountTx(
		ctx,
		tx,
		coursewareID,
		len(orderedPageIDs),
		now,
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交页码校准事务失败: %w",
			err,
		)
	}

	return nil
}

// InsertCoursewarePageAtPosition在一个事务中把新页面插入指定位置。
//
// insertAt语义：
//   - 1表示插入为新的第一页；
//   - N表示插入为新的第N页，原第N页及其后页面整体后移；
//   - 当前有count页时，允许范围为1至count+1；
//   - count+1表示追加到末尾；
//   - insertAt小于等于0表示旧客户端未指定位置，在事务锁内计算末页。
//
// 事务内完成页面插入、既有页顺延和总页数同步。
// 导航栏HTML校准由服务层在插页完成后调用严格校准方法处理。
func InsertCoursewarePageAtPosition(
	ctx context.Context,
	page *models.CoursewarePage,
	insertAt int,
) error {
	if page == nil {
		return fmt.Errorf(
			"待插入页面不能为空",
		)
	}

	if page.CoursewareID == "" {
		return fmt.Errorf(
			"待插入页面缺少课件ID",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启指定位置插页事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockCoursewarePageSequenceTx(
		ctx,
		tx,
		page.CoursewareID,
	); err != nil {
		return err
	}

	existingPages, err :=
		loadLockedCoursewarePagesTx(
			ctx,
			tx,
			page.CoursewareID,
		)
	if err != nil {
		return err
	}

	currentCount := len(existingPages)

	// 兼容旧客户端：未传insert_at时为0。
	// 必须在取得事务锁、读取真实当前页数之后再计算追加位置。
	if insertAt <= 0 {
		insertAt = currentCount + 1
	}

	if insertAt < 1 ||
		insertAt > currentCount+1 {
		return fmt.Errorf(
			"新增页位置无效：当前共%d页，可插入第1至第%d页",
			currentCount,
			currentCount+1,
		)
	}

	now := time.Now()

	if currentCount > 0 {
		if err := moveCoursewarePagesToTemporaryRangeTx(
			ctx,
			tx,
			page.CoursewareID,
		); err != nil {
			return err
		}
	}

	page.PageNumber = insertAt

	insertSQL := `INSERT INTO courseware_pages (
	id,
	courseware_id,
	page_number,
	title,
	purpose,
	content_summary,
	interaction_type,
	visual_format,
	media_requirements,
	estimated_complexity,
	page_index,
	idx_cognitive_level,
	idx_interaction_level,
	idx_visual_format,
	html_content,
	placeholder_map,
	matched_component_ids,
	status
)
VALUES (
	gen_random_uuid(),
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8,
	$9,
	$10,
	$11,
	$12,
	$13,
	$14,
	$15::jsonb,
	$16::jsonb,
	$17
)
RETURNING id, created_at, updated_at`

	if err := tx.QueryRow(
		ctx,
		insertSQL,
		page.CoursewareID,
		page.PageNumber,
		page.Title,
		page.Purpose,
		page.ContentSummary,
		page.InteractionType,
		page.VisualFormat,
		page.MediaRequirements,
		page.EstimatedComplexity,
		page.PageIndex,
		page.IdxCognitiveLevel,
		page.IdxInteractionLevel,
		page.IdxVisualFormat,
		page.HTMLContent,
		nullIfEmpty(page.PlaceholderMap),
		nullIfEmpty(page.MatchedComponentIDs),
		page.Status,
	).Scan(
		&page.ID,
		&page.CreatedAt,
		&page.UpdatedAt,
	); err != nil {
		return fmt.Errorf(
			"在指定位置创建课件页面失败: %w",
			err,
		)
	}

	nextPageNumber := 1

	for _, existingPage :=
		range existingPages {
		if nextPageNumber == insertAt {
			nextPageNumber++
		}

		commandTag, err := tx.Exec(
			ctx,
			`UPDATE courseware_pages
SET page_number = $1,
    updated_at = $2
WHERE id = $3
  AND courseware_id = $4`,
			nextPageNumber,
			now,
			existingPage.ID,
			page.CoursewareID,
		)
		if err != nil {
			return fmt.Errorf(
				"插页后恢复原页面页码失败(page_id=%s): %w",
				existingPage.ID,
				err,
			)
		}

		if commandTag.RowsAffected() != 1 {
			return fmt.Errorf(
				"插页期间页面发生变化或已经不存在: %s",
				existingPage.ID,
			)
		}

		nextPageNumber++
	}

	if err := updateCoursewarePageCountTx(
		ctx,
		tx,
		page.CoursewareID,
		currentCount+1,
		now,
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交指定位置插页事务失败: %w",
			err,
		)
	}

	return nil
}
