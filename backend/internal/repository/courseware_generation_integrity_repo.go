package repository

// courseware_generation_integrity_repo.go — R-04 课件页面生成完整性事实账本
//
// 本文件复用 courseware_assembly_runs.metadata 保存一次受控页面生成运行的不可变方案快照与终态对账，
// 不新增数据库表，也不把完整性状态塞进 Courseware.Status。
//
// 核心边界：
//   - 启动前按稳定 page_id 冻结期望页面清单和方案指纹；
//   - 运行身份继续由 assembly_version + run_id 保护；
//   - 终态重新读取真实页面，逐页按 page_id + 方案指纹 + 有效HTML对账；
//   - 已记录的明确失败优先于取消归类，未被派发且无HTML的页面在取消/中断时归为取消；
//   - 历史运行一旦写入 reconciliation，后续手工编辑不会篡改该次运行的终态结果。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

const coursewareGenerationPageResultFailed = "failed"

type coursewareGenerationExpectedPageSnapshot struct {
	PageID          string `json:"page_id"`
	PageNumber      int    `json:"page_number"`
	Title           string `json:"title"`
	PlanFingerprint string `json:"plan_fingerprint"`
}

type coursewareGenerationRecordedPageResult struct {
	Status     string `json:"status"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

type coursewareGenerationRunMetadata struct {
	IntegritySchemaVersion int                                               `json:"integrity_schema_version"`
	RunKind                string                                            `json:"run_kind"`
	ExpectedPages          []coursewareGenerationExpectedPageSnapshot        `json:"expected_pages"`
	PageResults            map[string]coursewareGenerationRecordedPageResult `json:"page_results"`
	SnapshotCreatedAt      string                                            `json:"snapshot_created_at"`
	Reconciliation         *models.CoursewareGenerationIntegrity             `json:"reconciliation,omitempty"`
}

type coursewareGenerationPlanFingerprintPayload struct {
	PageNumber          int    `json:"page_number"`
	Title               string `json:"title"`
	Purpose             string `json:"purpose"`
	ContentSummary      string `json:"content_summary"`
	InteractionType     string `json:"interaction_type"`
	VisualFormat        string `json:"visual_format"`
	MediaRequirements   string `json:"media_requirements"`
	EstimatedComplexity int    `json:"estimated_complexity"`
	PageIndex           string `json:"page_index"`
	IdxCognitiveLevel   int    `json:"idx_cognitive_level"`
	IdxInteractionLevel int    `json:"idx_interaction_level"`
	IdxVisualFormat     string `json:"idx_visual_format"`
}

func coursewareGenerationPlanFingerprint(
	page *models.CoursewarePage,
) (string, error) {
	if page == nil {
		return "", fmt.Errorf(
			"%w: 页面方案为空",
			ErrCoursewareAssemblyInvalid,
		)
	}

	payload := coursewareGenerationPlanFingerprintPayload{
		PageNumber:          page.PageNumber,
		Title:               page.Title,
		Purpose:             page.Purpose,
		ContentSummary:      page.ContentSummary,
		InteractionType:     page.InteractionType,
		VisualFormat:        page.VisualFormat,
		MediaRequirements:   page.MediaRequirements,
		EstimatedComplexity: page.EstimatedComplexity,
		PageIndex:           page.PageIndex,
		IdxCognitiveLevel:   page.IdxCognitiveLevel,
		IdxInteractionLevel: page.IdxInteractionLevel,
		IdxVisualFormat:     page.IdxVisualFormat,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(
			"序列化课件页面方案指纹失败: %w",
			err,
		)
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// BuildCoursewareGenerationRunMetadata 根据当前真实页面方案生成一次运行的不可变初始metadata。
func BuildCoursewareGenerationRunMetadata(
	runKind string,
	pages []*models.CoursewarePage,
) (string, error) {
	runKind = strings.TrimSpace(runKind)
	if !models.IsValidCoursewareGenerationRunKind(runKind) {
		return "", fmt.Errorf(
			"%w: 生成运行类型无效",
			ErrCoursewareAssemblyInvalid,
		)
	}

	if len(pages) == 0 {
		return "", fmt.Errorf(
			"%w: 课件没有页面方案",
			ErrCoursewareAssemblyInvalid,
		)
	}

	sortedPages := append(
		[]*models.CoursewarePage(nil),
		pages...,
	)
	sort.Slice(
		sortedPages,
		func(i int, j int) bool {
			return sortedPages[i].PageNumber <
				sortedPages[j].PageNumber
		},
	)

	expectedPages := make(
		[]coursewareGenerationExpectedPageSnapshot,
		0,
		len(sortedPages),
	)
	seenIDs := make(map[string]struct{}, len(sortedPages))
	seenNumbers := make(map[int]struct{}, len(sortedPages))

	for _, page := range sortedPages {
		if page == nil {
			return "", fmt.Errorf(
				"%w: 页面方案包含空记录",
				ErrCoursewareAssemblyInvalid,
			)
		}

		pageID := strings.TrimSpace(page.ID)
		if pageID == "" || page.PageNumber <= 0 {
			return "", fmt.Errorf(
				"%w: 页面缺少稳定ID或合法页码",
				ErrCoursewareAssemblyInvalid,
			)
		}

		if _, exists := seenIDs[pageID]; exists {
			return "", fmt.Errorf(
				"%w: 页面稳定ID重复",
				ErrCoursewareAssemblyInvalid,
			)
		}
		seenIDs[pageID] = struct{}{}

		if _, exists := seenNumbers[page.PageNumber]; exists {
			return "", fmt.Errorf(
				"%w: 页面页码重复",
				ErrCoursewareAssemblyInvalid,
			)
		}
		seenNumbers[page.PageNumber] = struct{}{}

		fingerprint, err :=
			coursewareGenerationPlanFingerprint(page)
		if err != nil {
			return "", err
		}

		title := strings.TrimSpace(page.Title)
		if title == "" {
			title = fmt.Sprintf(
				"第 %d 页",
				page.PageNumber,
			)
		}

		expectedPages = append(
			expectedPages,
			coursewareGenerationExpectedPageSnapshot{
				PageID:          pageID,
				PageNumber:      page.PageNumber,
				Title:           title,
				PlanFingerprint: fingerprint,
			},
		)
	}

	metadata := coursewareGenerationRunMetadata{
		IntegritySchemaVersion: models.CoursewareGenerationIntegritySchemaVersion,
		RunKind:                runKind,
		ExpectedPages:          expectedPages,
		PageResults:            map[string]coursewareGenerationRecordedPageResult{},
		SnapshotCreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf(
			"序列化课件生成完整性快照失败: %w",
			err,
		)
	}

	return string(encoded), nil
}

// RecordCoursewareGenerationPageFailure 记录当前版本运行中已经明确发生过的单页HTML失败。
//
// 只记录稳定page_id与失败事实，不保存AI原始错误、提示词、SQL或供应商信息。
// 如果运行已经取消、换版或结束，返回版本冲突，调用方不得因此覆盖页面。
func RecordCoursewareGenerationPageFailure(
	ctx context.Context,
	pageID string,
) error {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return ErrCoursewareAssemblyInvalid
	}

	assembly, ok := coursewareAssemblyWriteContextFrom(ctx)
	if !ok {
		return nil
	}

	recorded := coursewareGenerationRecordedPageResult{
		Status:     coursewareGenerationPageResultFailed,
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	recordedJSON, err := json.Marshal(recorded)
	if err != nil {
		return fmt.Errorf(
			"序列化课件页面失败事实失败: %w",
			err,
		)
	}

	tag, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_assembly_runs
                SET
                        metadata = jsonb_set(
                                metadata,
                                ARRAY['page_results', $1],
                                $2::jsonb,
                                true
                        ),
                        updated_at = NOW()
                WHERE id = $3
                  AND courseware_id = $4
                  AND version = $5
                  AND status = 'running'`,
		pageID,
		string(recordedJSON),
		assembly.RunID,
		assembly.CoursewareID,
		assembly.Version,
	)
	if err != nil {
		return fmt.Errorf(
			"记录课件页面生成失败事实失败: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrCoursewareAssemblyVersionConflict
	}

	return nil
}

func loadCoursewareGenerationRunMetadata(
	ctx context.Context,
	coursewareID string,
	version int64,
	runID string,
) (
	string,
	string,
	coursewareGenerationRunMetadata,
	error,
) {
	coursewareID = strings.TrimSpace(coursewareID)
	runID = strings.TrimSpace(runID)
	if coursewareID == "" || version <= 0 {
		return "", "", coursewareGenerationRunMetadata{}, nil
	}

	var storedRunID string
	var storedStatus string
	var metadataJSON string

	var err error
	if runID == "" {
		err = database.DB.QueryRow(
			ctx,
			`SELECT
                                id,
                                status,
                                metadata::text
                        FROM courseware_assembly_runs
                        WHERE courseware_id = $1
                          AND version = $2`,
			coursewareID,
			version,
		).Scan(
			&storedRunID,
			&storedStatus,
			&metadataJSON,
		)
	} else {
		err = database.DB.QueryRow(
			ctx,
			`SELECT
                                id,
                                status,
                                metadata::text
                        FROM courseware_assembly_runs
                        WHERE id = $1
                          AND courseware_id = $2
                          AND version = $3`,
			runID,
			coursewareID,
			version,
		).Scan(
			&storedRunID,
			&storedStatus,
			&metadataJSON,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", coursewareGenerationRunMetadata{}, nil
		}
		return "", "", coursewareGenerationRunMetadata{}, fmt.Errorf(
			"读取课件生成运行metadata失败: %w",
			err,
		)
	}

	metadata := coursewareGenerationRunMetadata{}
	if strings.TrimSpace(metadataJSON) != "" {
		if err := json.Unmarshal(
			[]byte(metadataJSON),
			&metadata,
		); err != nil {
			return "", "", coursewareGenerationRunMetadata{}, fmt.Errorf(
				"解析课件生成运行metadata失败: %w",
				err,
			)
		}
	}

	return storedRunID, storedStatus, metadata, nil
}

func resolveCoursewareGenerationRunKind(
	metadata coursewareGenerationRunMetadata,
) (string, error) {
	runKind := strings.TrimSpace(metadata.RunKind)
	if runKind == "" {
		// R-04上线前 courseware_assembly_runs 只承载全自动装配。
		return models.CoursewareGenerationRunKindAssembly, nil
	}

	if !models.IsValidCoursewareGenerationRunKind(runKind) {
		return "", fmt.Errorf(
			"%w: 数据库存储的生成运行类型无效",
			ErrCoursewareAssemblyInvalid,
		)
	}

	return runKind, nil
}

func isCoursewareGenerationPageSuccessful(
	page *models.CoursewarePage,
) bool {
	if page == nil ||
		strings.TrimSpace(page.HTMLContent) == "" {
		return false
	}

	switch strings.TrimSpace(page.Status) {
	case models.CWPageStatusGenerated,
		models.CWPageStatusMediaFilling,
		models.CWPageStatusConfirmed:
		return true
	default:
		return false
	}
}

func buildCoursewareGenerationIntegrity(
	runKind string,
	expectedPages []coursewareGenerationExpectedPageSnapshot,
	recordedResults map[string]coursewareGenerationRecordedPageResult,
	currentPages []*models.CoursewarePage,
	finalStatus string,
) (*models.CoursewareGenerationIntegrity, error) {
	currentByID := make(
		map[string]*models.CoursewarePage,
		len(currentPages),
	)
	for _, page := range currentPages {
		if page == nil {
			continue
		}
		pageID := strings.TrimSpace(page.ID)
		if pageID == "" {
			continue
		}
		currentByID[pageID] = page
	}

	integrity := &models.CoursewareGenerationIntegrity{
		SchemaVersion:   models.CoursewareGenerationIntegritySchemaVersion,
		RunKind:         runKind,
		ExpectedCount:   len(expectedPages),
		ActualPageCount: len(currentPages),
		SuccessPages:    []models.CoursewareGenerationPageRef{},
		FailedPages:     []models.CoursewareGenerationPageRef{},
		CancelledPages:  []models.CoursewareGenerationPageRef{},
		MissingPages:    []models.CoursewareGenerationPageRef{},
	}
	now := time.Now().UTC()
	integrity.ReconciledAt = &now

	terminal := models.IsCoursewareAssemblyFinalStatus(
		finalStatus,
	)

	for _, expected := range expectedPages {
		baseRef := models.CoursewareGenerationPageRef{
			PageID:     expected.PageID,
			PageNumber: expected.PageNumber,
			Title:      expected.Title,
		}

		current, exists := currentByID[expected.PageID]
		if !exists {
			baseRef.Reason =
				"当前方案中已找不到该页，请按当前方案重新生成缺失页"
			integrity.MissingPages = append(
				integrity.MissingPages,
				baseRef,
			)
			continue
		}

		currentFingerprint, err :=
			coursewareGenerationPlanFingerprint(current)
		if err != nil {
			return nil, err
		}
		if currentFingerprint != expected.PlanFingerprint {
			baseRef.Reason =
				"本次运行期间页面方案发生变化，旧结果不能作为当前方案的成功页"
			integrity.MissingPages = append(
				integrity.MissingPages,
				baseRef,
			)
			continue
		}

		if isCoursewareGenerationPageSuccessful(current) {
			integrity.SuccessPages = append(
				integrity.SuccessPages,
				baseRef,
			)
			continue
		}

		if recorded, ok :=
			recordedResults[expected.PageID]; ok &&
			recorded.Status ==
				coursewareGenerationPageResultFailed {
			baseRef.Reason =
				"本页HTML生成未成功，可只补生成这一页"
			integrity.FailedPages = append(
				integrity.FailedPages,
				baseRef,
			)
			continue
		}

		if !terminal {
			integrity.PendingCount++
			continue
		}

		switch finalStatus {
		case models.CoursewareAssemblyStatusCancelled,
			models.CoursewareAssemblyStatusInterrupted:
			baseRef.Reason =
				"本次运行在该页成功落库前已停止"
			integrity.CancelledPages = append(
				integrity.CancelledPages,
				baseRef,
			)

		case models.CoursewareAssemblyStatusFailed,
			models.CoursewareAssemblyStatusCompleted:
			// 正常结束或整体失败时，稳定方案页仍存在但没有有效HTML，
			// 说明本轮生成或必要校验没有成功；不能误报成“页面缺失”。
			baseRef.Reason =
				"本页生成或必要校验未成功，可只补生成这一页"
			integrity.FailedPages = append(
				integrity.FailedPages,
				baseRef,
			)

		default:
			baseRef.Reason =
				"方案存在，但数据库或有效结果缺失"
			integrity.MissingPages = append(
				integrity.MissingPages,
				baseRef,
			)
		}
	}

	integrity.SuccessCount = len(integrity.SuccessPages)
	integrity.FailedCount = len(integrity.FailedPages)
	integrity.CancelledCount = len(integrity.CancelledPages)
	integrity.MissingCount = len(integrity.MissingPages)

	classified := integrity.SuccessCount +
		integrity.FailedCount +
		integrity.CancelledCount +
		integrity.MissingCount +
		integrity.PendingCount
	if classified < integrity.ExpectedCount {
		integrity.PendingCount +=
			integrity.ExpectedCount - classified
	}

	integrity.Complete =
		integrity.ExpectedCount > 0 &&
			integrity.ActualPageCount ==
				integrity.ExpectedCount &&
			integrity.SuccessCount ==
				integrity.ExpectedCount &&
			integrity.FailedCount == 0 &&
			integrity.CancelledCount == 0 &&
			integrity.MissingCount == 0

	return integrity, nil
}

// ReconcileCoursewareGenerationIntegrity 按指定终态重新读取真实页面并逐页对账。
func ReconcileCoursewareGenerationIntegrity(
	ctx context.Context,
	coursewareID string,
	version int64,
	runID string,
	finalStatus string,
) (
	string,
	*models.CoursewareGenerationIntegrity,
	error,
) {
	storedRunID, _, metadata, err :=
		loadCoursewareGenerationRunMetadata(
			ctx,
			coursewareID,
			version,
			runID,
		)
	if err != nil {
		return "", nil, err
	}
	if storedRunID == "" {
		return "", nil, ErrCoursewareAssemblyNotFound
	}

	runKind, err :=
		resolveCoursewareGenerationRunKind(metadata)
	if err != nil {
		return "", nil, err
	}

	if metadata.IntegritySchemaVersion == 0 ||
		len(metadata.ExpectedPages) == 0 {
		// R-04上线前历史运行没有冻结快照，只能保留生命周期状态，不能伪造完整性事实。
		return runKind, nil, nil
	}
	if metadata.IntegritySchemaVersion !=
		models.CoursewareGenerationIntegritySchemaVersion {
		return "", nil, fmt.Errorf(
			"%w: 不支持的课件生成完整性协议版本",
			ErrCoursewareAssemblyInvalid,
		)
	}

	currentPages, err :=
		ListCoursewarePages(
			ctx,
			coursewareID,
		)
	if err != nil {
		return "", nil, fmt.Errorf(
			"读取课件当前页面用于完整性对账失败: %w",
			err,
		)
	}

	integrity, err :=
		buildCoursewareGenerationIntegrity(
			runKind,
			metadata.ExpectedPages,
			metadata.PageResults,
			currentPages,
			finalStatus,
		)
	if err != nil {
		return "", nil, err
	}

	return runKind, integrity, nil
}

// ReadCoursewareGenerationIntegrity 读取当前版本运行的教师安全完整性视图。
//
// 终态优先返回运行结束时已经固化的 reconciliation，避免后续手工编辑改写历史运行事实；
// 活动态则按冻结快照和当前页面实时计算成功/失败/缺失/待处理数量。
func ReadCoursewareGenerationIntegrity(
	ctx context.Context,
	coursewareID string,
	version int64,
) (
	string,
	*models.CoursewareGenerationIntegrity,
	error,
) {
	if strings.TrimSpace(coursewareID) == "" ||
		version <= 0 {
		return "", nil, nil
	}

	storedRunID, storedStatus, metadata, err :=
		loadCoursewareGenerationRunMetadata(
			ctx,
			coursewareID,
			version,
			"",
		)
	if err != nil {
		return "", nil, err
	}
	if storedRunID == "" {
		// 兼容R-04前遗留数据：没有运行记录时不能虚构对账结果。
		return "", nil, nil
	}

	runKind, err :=
		resolveCoursewareGenerationRunKind(metadata)
	if err != nil {
		return "", nil, err
	}

	if models.IsCoursewareAssemblyFinalStatus(
		storedStatus,
	) &&
		metadata.Reconciliation != nil {
		result := *metadata.Reconciliation
		result.RunKind = runKind
		return runKind, &result, nil
	}

	_, integrity, err :=
		ReconcileCoursewareGenerationIntegrity(
			ctx,
			coursewareID,
			version,
			storedRunID,
			storedStatus,
		)
	if err != nil {
		return "", nil, err
	}

	return runKind, integrity, nil
}
