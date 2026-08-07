package services

// lesson_plan_word_confirm_service.go — 原格式Word确认导入辅助服务
//
// 本文件负责保留原格式Word教案从“短时预解析会话”进入正式教案时的
// 可信会话解析、私有文件固化和失败补偿。
//
// 安全原则：
//   - 不信任浏览器再次提交的正文、文件路径、文件大小或文件哈希；
//   - 正文只能从当前用户自己的parsed会话中读取；
//   - 正式DOCX复制后重新计算SHA-256，并与预解析快照严格核对；
//   - 文件使用不可预测路径、0700目录和0600文件权限；
//   - 任一步失败都删除无数据库引用的正式文件半成品；
//   - Word文档已绑定数据库后，只有教案补偿删除成功才删除正式文件；
//   - 教案补偿删除成功后，把confirmed会话恢复成parsed，允许老师重试确认。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// resolveTrustedLessonPlanWordImportSession 从当前用户的短时会话中读取可信正文。
//
// 请求携带word_import_session_id时：
//   - 强制把source_type改成docx_fidelity；
//   - 强制使用数据库中的semantic_markdown；
//   - 忽略浏览器再次提交的content_markdown；
//   - 会话不存在、越权、过期或状态不正确时拒绝继续。
func resolveTrustedLessonPlanWordImportSession(
	ctx context.Context,
	req *models.ImportExistingPlanRequest,
	authorID string,
) (*models.LessonPlanWordImportSession, error) {
	if req == nil {
		return nil, errors.New(
			"导入教案请求不能为空",
		)
	}

	req.WordImportSessionID = strings.TrimSpace(
		req.WordImportSessionID,
	)
	if req.WordImportSessionID == "" {
		return nil, nil
	}

	session, err :=
		repository.GetLessonPlanWordImportSessionForUser(
			ctx,
			req.WordImportSessionID,
			authorID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: Word导入会话不存在或无权访问，请重新上传",
			ErrLPGenImportSourceInvalid,
		)
	}

	if session.Status !=
		models.LessonPlanWordImportStatusParsed ||
		session.ParsedAt == nil ||
		!session.ExpiresAt.After(time.Now()) {
		return nil, fmt.Errorf(
			"%w: Word导入会话已失效或尚未解析完成，请重新上传",
			ErrLPGenImportSourceInvalid,
		)
	}

	trustedContent := strings.TrimSpace(
		session.SemanticMarkdown,
	)
	if trustedContent == "" {
		return nil,
			ErrLPGenImportContentRequired
	}

	if strings.TrimSpace(
		session.StorageKey,
	) == "" ||
		session.FileSize <= 0 ||
		session.FileSize >
			MaxLessonPlanWordFileSize ||
		len(strings.TrimSpace(
			session.FileSHA256,
		)) != 64 ||
		strings.TrimSpace(
			session.StructureJSON,
		) == "" ||
		strings.TrimSpace(
			session.StructureJSON,
		) == "{}" {
		return nil, fmt.Errorf(
			"%w: Word导入会话数据不完整，请重新上传",
			ErrLPGenImportSourceInvalid,
		)
	}

	req.ContentMarkdown =
		trustedContent
	req.SourceType =
		"docx_fidelity"

	return session, nil
}

// prepareLessonPlanWordPermanentFile 将已解析DOCX复制为正式不可变版本1。
//
// 正式文件与短时imports文件使用不同路径。复制过程中重新计算哈希，
// 避免预览完成后源文件被替换或损坏。
func prepareLessonPlanWordPermanentFile(
	session *models.LessonPlanWordImportSession,
	lessonPlanID string,
) (
	storageKey string,
	fullPath string,
	fileSHA256 string,
	resultErr error,
) {
	if session == nil {
		return "", "", "", errors.New(
			"Word导入会话为空",
		)
	}

	lessonPlanID = strings.TrimSpace(
		lessonPlanID,
	)
	if _, err := uuid.Parse(
		lessonPlanID,
	); err != nil {
		return "", "", "", errors.New(
			"正式教案ID无效",
		)
	}

	if session.Status !=
		models.LessonPlanWordImportStatusParsed ||
		!session.ExpiresAt.After(time.Now()) {
		return "", "", "",
			repository.ErrLessonPlanWordImportConflict
	}

	sourcePath, err :=
		resolveLessonPlanWordPrivateStoragePath(
			session.StorageKey,
		)
	if err != nil {
		return "", "", "", err
	}

	sourceInfo, err := os.Lstat(
		sourcePath,
	)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", "", errors.New(
				"Word导入源文件已经失效，请重新上传",
			)
		}

		return "", "", "", fmt.Errorf(
			"读取Word导入源文件失败: %w",
			err,
		)
	}

	if sourceInfo.Mode()&os.ModeSymlink != 0 ||
		!sourceInfo.Mode().IsRegular() {
		return "", "", "", errors.New(
			"Word导入源文件不是安全的普通文件",
		)
	}

	if sourceInfo.Size() != session.FileSize {
		return "", "", "", errors.New(
			"Word导入源文件大小与解析快照不一致",
		)
	}

	storageKey = filepath.ToSlash(
		filepath.Join(
			"documents",
			lessonPlanID,
			"versions",
			fmt.Sprintf(
				"v1_%s.docx",
				uuid.NewString(),
			),
		),
	)

	fullPath, err =
		resolveLessonPlanWordPrivateStoragePath(
			storageKey,
		)
	if err != nil {
		return "", "", "", err
	}

	parentDirectory := filepath.Dir(
		fullPath,
	)
	if err := os.MkdirAll(
		parentDirectory,
		0o700,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"创建Word正式版本目录失败: %w",
			err,
		)
	}

	if err := os.Chmod(
		LessonPlanWordPrivateRoot,
		0o700,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"设置Word私有根目录权限失败: %w",
			err,
		)
	}

	if err := os.Chmod(
		parentDirectory,
		0o700,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"设置Word正式版本目录权限失败: %w",
			err,
		)
	}

	if _, statErr := os.Lstat(
		fullPath,
	); statErr == nil {
		return "", "", "", errors.New(
			"Word正式版本文件发生随机路径冲突",
		)
	} else if !os.IsNotExist(
		statErr,
	) {
		return "", "", "", fmt.Errorf(
			"检查Word正式版本路径失败: %w",
			statErr,
		)
	}

	sourceFile, err := os.Open(
		sourcePath,
	)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"打开Word导入源文件失败: %w",
			err,
		)
	}
	defer sourceFile.Close()

	temporaryPath := fullPath +
		".tmp_" +
		uuid.NewString()

	destinationFile, err := os.OpenFile(
		temporaryPath,
		os.O_WRONLY|
			os.O_CREATE|
			os.O_EXCL,
		0o600,
	)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"创建Word正式版本临时文件失败: %w",
			err,
		)
	}

	copyCompleted := false
	defer func() {
		if copyCompleted {
			return
		}

		_ = destinationFile.Close()
		_ = os.Remove(
			temporaryPath,
		)
		_ = os.Remove(
			fullPath,
		)
	}()

	hash := sha256.New()

	written, copyErr := io.Copy(
		io.MultiWriter(
			destinationFile,
			hash,
		),
		io.LimitReader(
			sourceFile,
			MaxLessonPlanWordFileSize+1,
		),
	)
	if copyErr == nil {
		copyErr =
			destinationFile.Sync()
	}

	closeErr :=
		destinationFile.Close()

	if copyErr != nil {
		return "", "", "", fmt.Errorf(
			"复制Word正式版本失败: %w",
			copyErr,
		)
	}
	if closeErr != nil {
		return "", "", "", fmt.Errorf(
			"关闭Word正式版本文件失败: %w",
			closeErr,
		)
	}
	if written != session.FileSize {
		return "", "", "", errors.New(
			"Word正式版本文件大小与导入会话不一致",
		)
	}
	if written >
		MaxLessonPlanWordFileSize {
		return "", "", "",
			ErrLessonPlanWordFileTooLarge
	}

	fileSHA256 = hex.EncodeToString(
		hash.Sum(nil),
	)
	if fileSHA256 !=
		strings.TrimSpace(
			session.FileSHA256,
		) {
		return "", "", "", errors.New(
			"Word正式版本文件哈希与预解析快照不一致",
		)
	}

	if err := os.Rename(
		temporaryPath,
		fullPath,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"原子固化Word正式版本失败: %w",
			err,
		)
	}

	if err := os.Chmod(
		fullPath,
		0o600,
	); err != nil {
		return "", "", "", fmt.Errorf(
			"设置Word正式版本文件权限失败: %w",
			err,
		)
	}

	copyCompleted = true

	return storageKey,
		fullPath,
		fileSHA256,
		nil
}

// resolveLessonPlanWordPrivateStoragePath 将数据库受控相对键解析成私有绝对路径。
//
// 数据库已经有路径约束，文件操作前仍重复执行应用层校验，
// 防止错误迁移、人工修复或脏数据造成目录越界。
func resolveLessonPlanWordPrivateStoragePath(
	storageKey string,
) (string, error) {
	storageKey = filepath.ToSlash(
		strings.TrimSpace(
			storageKey,
		),
	)

	if storageKey == "" ||
		strings.HasPrefix(
			storageKey,
			"/",
		) ||
		strings.Contains(
			storageKey,
			"..",
		) ||
		strings.Contains(
			storageKey,
			`\`,
		) ||
		strings.Contains(
			storageKey,
			"//",
		) {
		return "", errors.New(
			"Word私有存储键无效",
		)
	}

	rootPath := filepath.Clean(
		LessonPlanWordPrivateRoot,
	)

	rootInfo, err := os.Lstat(
		rootPath,
	)
	if err != nil {
		return "", fmt.Errorf(
			"读取Word私有根目录失败: %w",
			err,
		)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 ||
		!rootInfo.IsDir() {
		return "", errors.New(
			"Word私有根目录不是安全目录",
		)
	}

	fullPath := filepath.Clean(
		filepath.Join(
			rootPath,
			filepath.FromSlash(
				storageKey,
			),
		),
	)

	if fullPath == rootPath ||
		!strings.HasPrefix(
			fullPath,
			rootPath+
				string(
					os.PathSeparator,
				),
		) {
		return "", errors.New(
			"Word私有存储路径越界",
		)
	}

	return fullPath, nil
}

// removeLessonPlanWordPrivateFile 删除已通过私有根目录边界检查的文件。
func removeLessonPlanWordPrivateFile(
	fullPath string,
) error {
	fullPath = filepath.Clean(
		strings.TrimSpace(
			fullPath,
		),
	)
	if fullPath == "" {
		return nil
	}

	rootPath := filepath.Clean(
		LessonPlanWordPrivateRoot,
	)
	if fullPath == rootPath ||
		!strings.HasPrefix(
			fullPath,
			rootPath+
				string(
					os.PathSeparator,
				),
		) {
		return errors.New(
			"拒绝删除Word私有根目录之外的文件",
		)
	}

	if err := os.Remove(
		fullPath,
	); err != nil &&
		!os.IsNotExist(err) {
		return err
	}

	return nil
}

// removeLessonPlanWordImportSourceFile 删除确认成功后的短时imports副本。
//
// 正式不可变版本已经完成独立复制和哈希复核，短时副本无需继续占用磁盘。
// 导入会话数据库记录仍保留，用于追溯导入来源和状态。
func removeLessonPlanWordImportSourceFile(
	session *models.LessonPlanWordImportSession,
) error {
	if session == nil {
		return nil
	}

	fullPath, err :=
		resolveLessonPlanWordPrivateStoragePath(
			session.StorageKey,
		)
	if err != nil {
		return err
	}

	return removeLessonPlanWordPrivateFile(
		fullPath,
	)
}

// cleanupFailedImportedLessonPlanWithWord 清理正式导入失败后的数据库和文件。
//
// Word文档尚未绑定数据库时，正式文件可直接删除。
// Word文档已经绑定后，必须先成功删除教案，使数据库级联清理Word记录，
// 才能删除正式文件，避免数据库记录指向不存在的DOCX。
func cleanupFailedImportedLessonPlanWithWord(
	ctx context.Context,
	lessonPlanID string,
	authorID string,
	educationDomain string,
	session *models.LessonPlanWordImportSession,
	permanentFullPath string,
	wordDocumentBound bool,
	importedAssetFullPaths []string,
) error {
	planCleanupErr :=
		repository.DeleteIncompleteImportedLessonPlanCreation(
			ctx,
			lessonPlanID,
			authorID,
			educationDomain,
		)

	planCleanupSucceeded :=
		planCleanupErr == nil

	combinedErr :=
		planCleanupErr

	if planCleanupSucceeded &&
		session != nil &&
		wordDocumentBound {
		resetErr :=
			repository.ResetConfirmedLessonPlanWordImportSession(
				ctx,
				session.ID,
				authorID,
			)

		combinedErr =
			combineLessonPlanWordCleanupError(
				combinedErr,
				"恢复Word导入会话失败",
				resetErr,
			)
	}

	if permanentFullPath != "" &&
		(planCleanupSucceeded ||
			!wordDocumentBound) {
		removeErr :=
			removeLessonPlanWordPrivateFile(
				permanentFullPath,
			)

		combinedErr =
			combineLessonPlanWordCleanupError(
				combinedErr,
				"删除Word正式半成品失败",
				removeErr,
			)
	}

	if planCleanupSucceeded {
		assetCleanupErr :=
			removeLessonPlanWordImportedAssetFiles(
				lessonPlanID,
				importedAssetFullPaths,
			)

		combinedErr =
			combineLessonPlanWordCleanupError(
				combinedErr,
				"删除Word图片资产失败",
				assetCleanupErr,
			)
	}

	return combinedErr
}

func combineLessonPlanWordCleanupError(
	current error,
	message string,
	next error,
) error {
	if next == nil {
		return current
	}

	if current == nil {
		return fmt.Errorf(
			"%s: %w",
			message,
			next,
		)
	}

	return fmt.Errorf(
		"%w；%s: %v",
		current,
		message,
		next,
	)
}
