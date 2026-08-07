package services

// lesson_plan_word_restore_service.go — 教案正文历史与原格式Word版本恢复服务
//
// 恢复策略：
//   - 普通教案继续使用统一正文CAS更新入口；
//   - Word保真教案必须找到与目标Markdown完全一致的不可变Word版本；
//   - 优先选择Word版本号与正文历史version_number相同的快照；
//   - 没有同号快照时，仅允许选择文件哈希、结构哈希和语义哈希完全等价的快照；
//   - 同一Markdown对应多份不同版式或图片的Word时拒绝自动猜测；
//   - 历史DOCX先复制为新的不可变版本文件，再与正文一起原子提交；
//   - 当前Word即使处于stale，也可通过可信历史快照恢复为active。
//
// Word保真教案暂不允许通过正文历史恢复标题或课时时长，
// 因为Word版本快照没有独立保存这些元信息，不能证明目标DOCX与它们一致。

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrLessonPlanWordRestoreSnapshotNotFound = errors.New(
		"历史正文没有对应的原格式Word版本",
	)
	ErrLessonPlanWordRestoreSnapshotAmbiguous = errors.New(
		"历史正文对应多份不同的原格式Word版本",
	)
	ErrLessonPlanWordRestoreMetadataUnsupported = errors.New(
		"原格式Word历史恢复暂不支持改变标题或课时时长",
	)
	ErrLessonPlanWordRestoreMetadataStale = errors.New(
		"原格式Word因课程元信息变化而失步，无法证明历史快照与当前课程定位一致",
	)
)

// RestoreLessonPlanContentVersionPreservingWord 恢复正文历史，并在需要时恢复Word快照。
func RestoreLessonPlanContentVersionPreservingWord(
	ctx context.Context,
	plan *models.LessonPlan,
	target *models.LessonPlanContentVersion,
	callerID string,
) (*LessonPlanContentMutationResult, error) {
	callerID = strings.TrimSpace(callerID)

	if plan == nil || target == nil ||
		strings.TrimSpace(plan.ID) == "" ||
		callerID == "" {
		return nil, ErrLPNotFound
	}
	if plan.AuthorID != callerID {
		return nil, ErrLPNotAuthor
	}
	if !isLessonPlanSectionEditableStatusService(plan.Status) {
		return nil, ErrLPCannotEdit
	}

	wordDocument, err :=
		repository.GetLessonPlanWordDocumentForOwner(
			ctx,
			plan.ID,
			callerID,
		)
	if err != nil {
		if !errors.Is(
			err,
			repository.ErrLessonPlanWordDocumentNotFound,
		) {
			return nil, err
		}

		return UpdateLessonPlanContentPreservingWord(
			ctx,
			LessonPlanContentMutationInput{
				PlanID:            plan.ID,
				CallerID:          callerID,
				Title:             target.Title,
				ContentMarkdown:   target.ContentMarkdown,
				ContentStructured: target.ContentStructured,
				DurationMinutes:   target.DurationMinutes,
				ExpectedVersion:   plan.Version,
				ExpectedContent:   plan.ContentMarkdown,
				ChangeSource:      models.LessonPlanWordChangeSourceRestore,
				ChangeSummary: fmt.Sprintf(
					"恢复历史版本 v%d",
					target.VersionNumber,
				),
			},
		)
	}

	switch wordDocument.Status {
	case models.LessonPlanWordDocumentStatusActive,
		models.LessonPlanWordDocumentStatusStale:
		// active和正文失步stale可以从可信历史快照恢复。
	default:
		return nil, fmt.Errorf(
			"%w: 原格式Word当前状态不可恢复",
			ErrLPCannotEdit,
		)
	}

	if err := validateLessonPlanWordRestoreStaleReason(
		wordDocument,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrLPCannotEdit,
			err,
		)
	}

	if target.Title != plan.Title ||
		target.DurationMinutes != plan.DurationMinutes {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrLPCannotEdit,
			ErrLessonPlanWordRestoreMetadataUnsupported,
		)
	}

	candidates, err :=
		repository.ListLessonPlanWordVersionsBySemanticForOwner(
			ctx,
			plan.ID,
			callerID,
			target.ContentMarkdown,
		)
	if err != nil {
		return nil, err
	}

	targetWord, err :=
		selectLessonPlanWordRestoreVersion(
			candidates,
			target.VersionNumber,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrLPCannotEdit,
			err,
		)
	}

	targetStructured :=
		normalizeLessonPlanWordStructuredJSON(
			target.ContentStructured,
		)
	currentStructured :=
		normalizeLessonPlanWordStructuredJSON(
			plan.ContentStructured,
		)

	// 正文、结构和Word文件已经是目标版本时属于幂等成功。
	if wordDocument.Status ==
		models.LessonPlanWordDocumentStatusActive &&
		plan.Title == target.Title &&
		plan.ContentMarkdown ==
			target.ContentMarkdown &&
		currentStructured == targetStructured &&
		plan.DurationMinutes ==
			target.DurationMinutes &&
		strings.EqualFold(
			wordDocument.CurrentFileSHA256,
			targetWord.FileSHA256,
		) &&
		wordDocument.StructureHash ==
			targetWord.StructureHash &&
		wordDocument.SemanticMarkdown ==
			targetWord.SemanticMarkdown {
		return &LessonPlanContentMutationResult{
			Changed:         false,
			CurrentVersion:  plan.Version,
			ContentMarkdown: plan.ContentMarkdown,
		}, nil
	}

	nextWordVersion := wordDocument.Version + 1

	nextStorageKey,
		nextFullPath,
		nextFileSHA256,
		err :=
		copyLessonPlanWordRestoreVersion(
			targetWord,
			wordDocument,
			plan.ID,
			nextWordVersion,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: 对应的原格式Word历史文件不可用: %v",
			ErrLPCannotEdit,
			err,
		)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if removeErr := os.Remove(nextFullPath); removeErr != nil &&
			!os.IsNotExist(removeErr) {
			lpLog.Warn(
				"清理未提交的Word恢复版本文件失败",
				"plan_id", plan.ID,
				"path", nextFullPath,
				"error", removeErr,
			)
		}
	}()

	summary := fmt.Sprintf(
		"恢复历史版本 v%d 并同步原格式Word",
		target.VersionNumber,
	)

	result, err :=
		repository.CommitLessonPlanWordVersionRestore(
			ctx,
			repository.LessonPlanWordRestoreInput{
				LessonPlanID: plan.ID,
				OwnerID:      callerID,

				ExpectedPlanVersion:    plan.Version,
				ExpectedPlanTitle:      plan.Title,
				ExpectedPlanContent:    plan.ContentMarkdown,
				ExpectedPlanStructured: currentStructured,
				ExpectedPlanDuration:   plan.DurationMinutes,

				ExpectedWordDocumentID: wordDocument.ID,
				ExpectedWordStatus:     wordDocument.Status,
				ExpectedWordVersion:    wordDocument.Version,
				ExpectedWordStorageKey: wordDocument.CurrentStorageKey,
				ExpectedWordFileSHA256: wordDocument.CurrentFileSHA256,
				ExpectedWordSemantic:   wordDocument.SemanticMarkdown,

				NextTitle:               target.Title,
				NextContentMarkdown:     target.ContentMarkdown,
				NextContentStructured:   targetStructured,
				NextDurationMinutes:     target.DurationMinutes,
				NextWordStorageKey:      nextStorageKey,
				NextWordFileSHA256:      nextFileSHA256,
				NextWordParserVersion:   targetWord.ParserVersion,
				NextWordStructureSchema: targetWord.StructureSchemaVersion,
				NextWordStructureJSON:   targetWord.StructureJSON,
				NextWordSemanticHash:    targetWord.SemanticMarkdownHash,
				NextWordStructureHash:   targetWord.StructureHash,
				NextWordMetricsJSON:     targetWord.MetricsJSON,
				NextWordWarningsJSON:    targetWord.WarningsJSON,
				ChangedBy:               lessonPlanSectionStringPtr(callerID),
				ChangeSummary:           summary,
			},
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		):
			return nil, ErrLPNotFound

		case errors.Is(
			err,
			repository.ErrLessonPlanSectionNotAuthor,
		):
			return nil, ErrLPNotAuthor

		case errors.Is(
			err,
			repository.ErrLessonPlanSectionNotEditable,
		):
			return nil, ErrLPCannotEdit

		case errors.Is(
			err,
			repository.ErrLessonPlanWordRestoreConflict,
		):
			return nil, ErrLPSectionVersionConflict

		default:
			return nil, err
		}
	}

	committed = true

	return &LessonPlanContentMutationResult{
		Changed:         true,
		CurrentVersion:  result.LessonPlanVersion,
		ContentMarkdown: target.ContentMarkdown,
	}, nil
}

// validateLessonPlanWordRestoreStaleReason 区分可恢复的正文失步和不可证明的元信息失步。
//
// 数据库触发器对两类漂移写入不同错误说明：
//   - 平台语义正文变化：可以通过历史Word快照恢复；
//   - 标题、课程定位或课时时长变化：Word版本没有独立保存完整课程元信息，
//     因此无法证明历史DOCX仍与当前学科、年级和课题一致，必须拒绝自动恢复。
//
// 未知或空原因同样fail-closed，避免把其它异常stale误恢复为active。
func validateLessonPlanWordRestoreStaleReason(
	document *models.LessonPlanWordDocument,
) error {
	if document == nil {
		return ErrLessonPlanWordRestoreMetadataStale
	}
	if document.Status !=
		models.LessonPlanWordDocumentStatusStale {
		return nil
	}

	reason := strings.TrimSpace(document.ErrorMessage)
	if strings.Contains(
		reason,
		"平台语义正文已由其它链路修改",
	) {
		return nil
	}

	return ErrLessonPlanWordRestoreMetadataStale
}

// selectLessonPlanWordRestoreVersion 选择正文历史对应的可信Word版本。
func selectLessonPlanWordRestoreVersion(
	candidates []*models.LessonPlanWordDocumentVersion,
	targetContentVersion int,
) (*models.LessonPlanWordDocumentVersion, error) {
	var exact *models.LessonPlanWordDocumentVersion

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if candidate.Version == targetContentVersion {
			if exact != nil {
				return nil,
					ErrLessonPlanWordRestoreSnapshotAmbiguous
			}
			exact = candidate
		}
	}

	if exact != nil {
		return exact, nil
	}

	nonNil := make(
		[]*models.LessonPlanWordDocumentVersion,
		0,
		len(candidates),
	)
	for _, candidate := range candidates {
		if candidate != nil {
			nonNil = append(nonNil, candidate)
		}
	}
	if len(nonNil) == 0 {
		return nil, ErrLessonPlanWordRestoreSnapshotNotFound
	}

	selected := nonNil[0]
	for _, candidate := range nonNil[1:] {
		if !strings.EqualFold(
			candidate.FileSHA256,
			selected.FileSHA256,
		) ||
			candidate.StructureHash !=
				selected.StructureHash ||
			candidate.SemanticMarkdownHash !=
				selected.SemanticMarkdownHash {
			return nil,
				ErrLessonPlanWordRestoreSnapshotAmbiguous
		}
	}

	return selected, nil
}

// copyLessonPlanWordRestoreVersion 把历史不可变DOCX复制为新的不可变版本文件。
func copyLessonPlanWordRestoreVersion(
	sourceVersion *models.LessonPlanWordDocumentVersion,
	currentDocument *models.LessonPlanWordDocument,
	lessonPlanID string,
	nextWordVersion int,
) (string, string, string, error) {
	if sourceVersion == nil ||
		currentDocument == nil ||
		strings.TrimSpace(lessonPlanID) == "" ||
		nextWordVersion <= 1 {
		return "", "", "",
			ErrLessonPlanWordStructureChangeUnsupported
	}

	verified, err :=
		openVerifiedLessonPlanWordStoredFile(
			&models.LessonPlanWordDocument{
				OriginalFileName:  currentDocument.OriginalFileName,
				CurrentStorageKey: sourceVersion.StorageKey,
				CurrentFileSHA256: sourceVersion.FileSHA256,
			},
		)
	if err != nil {
		return "", "", "", err
	}
	defer verified.File.Close()

	storageKey := filepath.ToSlash(
		filepath.Join(
			"documents",
			lessonPlanID,
			"versions",
			fmt.Sprintf(
				"v%06d-%s.docx",
				nextWordVersion,
				uuid.NewString(),
			),
		),
	)

	fullPath, err :=
		resolveLessonPlanWordPrivatePath(storageKey)
	if err != nil {
		return "", "", "", err
	}

	if err := os.MkdirAll(
		filepath.Dir(fullPath),
		0o700,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"创建Word恢复版本目录失败: %w",
			err,
		)
	}
	if err := os.Chmod(
		filepath.Dir(fullPath),
		0o700,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"设置Word恢复版本目录权限失败: %w",
			err,
		)
	}

	temporaryPath :=
		fullPath + ".tmp-" + uuid.NewString()

	output, err := os.OpenFile(
		temporaryPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"创建Word恢复临时文件失败: %w",
			err,
		)
	}

	completed := false
	defer func() {
		if completed {
			return
		}
		_ = output.Close()
		_ = os.Remove(temporaryPath)
		_ = os.Remove(fullPath)
	}()

	written, err := io.Copy(
		output,
		io.LimitReader(
			verified.File,
			MaxLessonPlanWordFileSize+1,
		),
	)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"复制Word历史版本失败: %w",
			err,
		)
	}
	if written <= 0 ||
		written > MaxLessonPlanWordFileSize ||
		written != verified.FileInfo.Size() {
		return "", "", "",
			ErrLessonPlanWordDownloadUnavailable
	}

	if err := output.Sync(); err != nil {
		return "", "", "", fmt.Errorf(
			"同步Word恢复临时文件失败: %w",
			err,
		)
	}
	if err := output.Close(); err != nil {
		return "", "", "", fmt.Errorf(
			"关闭Word恢复临时文件失败: %w",
			err,
		)
	}
	if err := os.Rename(
		temporaryPath,
		fullPath,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"固化Word恢复版本失败: %w",
			err,
		)
	}
	if err := os.Chmod(fullPath, 0o600); err != nil {
		return "", "", "", fmt.Errorf(
			"设置Word恢复版本文件权限失败: %w",
			err,
		)
	}

	archive, err := zip.OpenReader(fullPath)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"复核Word恢复版本失败: %w",
			err,
		)
	}
	_, validateErr :=
		validateLessonPlanWordArchive(archive.File)
	closeErr := archive.Close()
	if validateErr != nil {
		return "", "", "", validateErr
	}
	if closeErr != nil {
		return "", "", "", fmt.Errorf(
			"关闭Word恢复版本复核文件失败: %w",
			closeErr,
		)
	}

	fileSHA256, err :=
		hashLessonPlanWordFile(fullPath)
	if err != nil {
		return "", "", "", err
	}
	if !strings.EqualFold(
		fileSHA256,
		sourceVersion.FileSHA256,
	) {
		return "", "", "",
			ErrLessonPlanWordDownloadUnavailable
	}

	completed = true
	return storageKey, fullPath, fileSHA256, nil
}
