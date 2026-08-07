package services

// lesson_plan_word_download_service.go — 原格式Word教案安全下载服务
//
// 下载策略：
//   1. active文档直接下载已经持久化并通过哈希校验的当前DOCX；
//   2. stale文档仅在确认差异完全属于“删除图片”时，按需生成临时派生DOCX；
//   3. 文字、表格或其它正文发生变化时，不得把旧Word冒充当前正文；
//   4. 所有文件均从受控私有根目录读取，下载前重新核验文件身份与SHA-256。
//
// 本服务不把storage_key、哈希或物理路径返回浏览器。

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
	"unicode"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrLessonPlanWordDownloadNotFound 合并“没有保真Word”和“不是作者”两种情况，
	// 防止通过教案ID探测他人的私有Word文档。
	ErrLessonPlanWordDownloadNotFound = errors.New(
		"原格式Word教案不存在",
	)

	// ErrLessonPlanWordDownloadOutOfSync 表示当前正文不只是删除了图片，
	// 无法在不猜测老师意图的情况下安全保持原Word版式。
	ErrLessonPlanWordDownloadOutOfSync = errors.New(
		"原格式Word已与当前正文不同步，请使用普通Word导出当前正文",
	)

	// ErrLessonPlanWordDownloadUnavailable 表示记录存在，但文件、路径或哈希不符合安全要求。
	// 对外不暴露具体物理路径和内部校验细节。
	ErrLessonPlanWordDownloadUnavailable = errors.New(
		"原格式Word文件暂时不可用",
	)
)

// LessonPlanWordDownload 是处理器可安全流式发送的Word文件。
//
// File由调用方关闭。FileName只包含安全文件名，不包含目录。
type LessonPlanWordDownload struct {
	File     *os.File
	FileName string
	Size     int64
	ModTime  time.Time
}

// verifiedLessonPlanWordFile 是已经完成路径、文件身份、大小和哈希校验的私有DOCX。
type verifiedLessonPlanWordFile struct {
	File     *os.File
	FileInfo os.FileInfo
	FileName string
	Path     string
}

// OpenLessonPlanWordDownload 打开作者本人的当前原格式Word文件。
func OpenLessonPlanWordDownload(
	ctx context.Context,
	lessonPlanID string,
	ownerID string,
) (*LessonPlanWordDownload, error) {
	lessonPlanID = strings.TrimSpace(lessonPlanID)
	ownerID = strings.TrimSpace(ownerID)

	if lessonPlanID == "" || ownerID == "" {
		return nil, ErrLessonPlanWordDownloadNotFound
	}

	record, err := repository.GetLessonPlanWordDocumentForOwner(
		ctx,
		lessonPlanID,
		ownerID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanWordDocumentNotFound) {
			return nil, ErrLessonPlanWordDownloadNotFound
		}
		return nil, fmt.Errorf("查询原格式Word文档失败: %w", err)
	}

	switch strings.TrimSpace(record.Status) {
	case models.LessonPlanWordDocumentStatusActive:
		return openActiveLessonPlanWordDownload(record)

	case models.LessonPlanWordDocumentStatusStale:
		download, buildErr := openLessonPlanWordImageDeletionDownload(
			ctx,
			record,
			ownerID,
		)
		if buildErr == nil {
			return download, nil
		}

		switch {
		case errors.Is(
			buildErr,
			errLessonPlanWordImageDeletionUnsupported,
		):
			return nil, ErrLessonPlanWordDownloadOutOfSync

		case errors.Is(
			buildErr,
			ErrLessonPlanWordDownloadUnavailable,
		):
			return nil, ErrLessonPlanWordDownloadUnavailable

		default:
			return nil, fmt.Errorf(
				"生成删除图片后的原格式Word失败: %w",
				buildErr,
			)
		}

	default:
		return nil, ErrLessonPlanWordDownloadUnavailable
	}
}

// openActiveLessonPlanWordDownload 打开经过完整安全校验的active DOCX。
func openActiveLessonPlanWordDownload(
	record *models.LessonPlanWordDocument,
) (*LessonPlanWordDownload, error) {
	verified, err := openVerifiedLessonPlanWordStoredFile(record)
	if err != nil {
		return nil, err
	}

	return &LessonPlanWordDownload{
		File:     verified.File,
		FileName: verified.FileName,
		Size:     verified.FileInfo.Size(),
		ModTime:  verified.FileInfo.ModTime(),
	}, nil
}

// openVerifiedLessonPlanWordStoredFile 打开并校验数据库当前版本指向的私有DOCX。
//
// 返回的File已经Seek回文件起点，调用方负责关闭。
func openVerifiedLessonPlanWordStoredFile(
	record *models.LessonPlanWordDocument,
) (*verifiedLessonPlanWordFile, error) {
	if record == nil {
		return nil, ErrLessonPlanWordDownloadUnavailable
	}

	storageKey := strings.TrimSpace(record.CurrentStorageKey)
	expectedSHA256 := strings.ToLower(
		strings.TrimSpace(record.CurrentFileSHA256),
	)

	if storageKey == "" || len(expectedSHA256) != sha256.Size*2 {
		return nil, ErrLessonPlanWordDownloadUnavailable
	}

	if _, err := hex.DecodeString(expectedSHA256); err != nil {
		return nil, ErrLessonPlanWordDownloadUnavailable
	}

	fullPath, err := resolveLessonPlanWordPrivatePath(storageKey)
	if err != nil {
		return nil, ErrLessonPlanWordDownloadUnavailable
	}

	// Lstat先拒绝符号链接。打开文件后再用SameFile复核，避免检查与打开之间
	// 文件被替换，或把目录、设备文件等非普通对象作为Word返回。
	fileInfo, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrLessonPlanWordDownloadUnavailable
		}
		return nil, fmt.Errorf("读取原格式Word文件状态失败: %w", err)
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 ||
		!fileInfo.Mode().IsRegular() ||
		fileInfo.Size() <= 0 ||
		fileInfo.Size() > MaxLessonPlanWordFileSize {
		return nil, ErrLessonPlanWordDownloadUnavailable
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("打开原格式Word文件失败: %w", err)
	}

	closeWithError := func(result error) (*verifiedLessonPlanWordFile, error) {
		_ = file.Close()
		return nil, result
	}

	openedInfo, err := file.Stat()
	if err != nil {
		return closeWithError(
			fmt.Errorf("复核原格式Word文件状态失败: %w", err),
		)
	}

	if !openedInfo.Mode().IsRegular() ||
		!os.SameFile(fileInfo, openedInfo) ||
		openedInfo.Size() != fileInfo.Size() {
		return closeWithError(ErrLessonPlanWordDownloadUnavailable)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return closeWithError(
			fmt.Errorf("校验原格式Word文件失败: %w", err),
		)
	}

	actualSHA256 := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualSHA256, expectedSHA256) {
		return closeWithError(ErrLessonPlanWordDownloadUnavailable)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return closeWithError(
			fmt.Errorf("重置原格式Word文件读取位置失败: %w", err),
		)
	}

	return &verifiedLessonPlanWordFile{
		File:     file,
		FileInfo: openedInfo,
		FileName: normalizeLessonPlanWordDownloadFileName(record.OriginalFileName),
		Path:     fullPath,
	}, nil
}

// resolveLessonPlanWordPrivatePath 把数据库storage_key安全解析到Word私有根目录。
func resolveLessonPlanWordPrivatePath(
	storageKey string,
) (string, error) {
	storageKey = strings.TrimSpace(storageKey)

	if storageKey == "" ||
		strings.ContainsRune(storageKey, '\x00') ||
		strings.Contains(storageKey, "\\") ||
		strings.HasPrefix(storageKey, "/") {
		return "", ErrLessonPlanWordDownloadUnavailable
	}

	rootPath := filepath.Clean(LessonPlanWordPrivateRoot)
	fullPath := filepath.Clean(
		filepath.Join(
			rootPath,
			filepath.FromSlash(storageKey),
		),
	)

	if fullPath == rootPath ||
		!strings.HasPrefix(
			fullPath,
			rootPath+string(os.PathSeparator),
		) {
		return "", ErrLessonPlanWordDownloadUnavailable
	}

	return fullPath, nil
}

// normalizeLessonPlanWordDownloadFileName 生成Content-Disposition使用的安全文件名。
func normalizeLessonPlanWordDownloadFileName(
	fileName string,
) string {
	fileName = strings.TrimSpace(
		filepath.Base(
			strings.ReplaceAll(fileName, "\x00", ""),
		),
	)

	fileName = strings.Map(
		func(value rune) rune {
			if unicode.IsControl(value) ||
				value == '/' ||
				value == '\\' {
				return -1
			}
			return value
		},
		fileName,
	)

	runes := []rune(strings.TrimSpace(fileName))
	if len(runes) > 200 {
		runes = runes[:200]
		fileName = strings.TrimSpace(string(runes))
	}

	if fileName == "" {
		return "原格式教案.docx"
	}

	if !strings.EqualFold(
		filepath.Ext(fileName),
		".docx",
	) {
		fileName += ".docx"
	}

	return fileName
}
