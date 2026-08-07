package services

// lesson_plan_word_import_service.go — 原格式Word教案上传、私有落盘与安全预解析
//
// 本阶段只建立“上传→解析→安全预览”的独立闭环：
//   - 不创建正式lesson_plans记录；
//   - 不启动AI评审；
//   - 不改变现有普通Word/PDF纯文本导入；
//   - 老师后续确认后，才会进入现有ImportExistingPlan工作流。
//
// 文件安全原则：
//   - DOCX只写入private目录，不生成Nginx公开URL；
//   - 浏览器文件名、MIME和哈希均不可信，服务端重新规范化和计算；
//   - 文件使用随机不可预测存储键和0600权限；目录使用0700权限；
//   - 解析失败的文件立即删除，只保留限长错误状态，避免恶意文件长期占用空间。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	// MaxLessonPlanWordFileSize 第一版单个DOCX最大30MB。
	// 数据库约束使用相同上限，防止应用与数据库口径漂移。
	MaxLessonPlanWordFileSize = int64(30 * 1024 * 1024)

	// LessonPlanWordPrivateRoot 不在Nginx公开uploads目录下。
	LessonPlanWordPrivateRoot = "/www/wwwroot/tedna/private/lesson-plan-word"

	lessonPlanWordImportTTL = 24 * time.Hour
)

var (
	ErrLessonPlanWordFileRequired = errors.New(
		"请选择要上传的Word文档",
	)
	ErrLessonPlanWordFileTooLarge = errors.New(
		"Word文档过大，最大支持30MB",
	)
	ErrLessonPlanWordFileInvalid = errors.New(
		"文件格式无效，仅支持有效的.docx文档",
	)
	ErrLessonPlanWordParseFailed = errors.New(
		"Word文档解析失败",
	)
)

// LessonPlanWordImportPreviewResponse 是浏览器安全的预解析响应。
//
// 明确不返回：
//   - storage_key；
//   - 服务端物理路径；
//   - 文件SHA-256；
//   - 原始OOXML；
//   - 数据库内部结构哈希。
type LessonPlanWordImportPreviewResponse struct {
	SessionID              string                         `json:"session_id"`
	Status                 string                         `json:"status"`
	OriginalFileName       string                         `json:"original_file_name"`
	FileSize               int64                          `json:"file_size"`
	ParserVersion          string                         `json:"parser_version"`
	StructureSchemaVersion int                            `json:"structure_schema_version"`
	SemanticMarkdown       string                         `json:"semantic_markdown"`
	Metrics                LessonPlanWordPreviewMetrics   `json:"metrics"`
	Warnings               []LessonPlanWordPreviewWarning `json:"warnings"`
	Document               LessonPlanWordPreviewDocument  `json:"document"`
	ExpiresAt              time.Time                      `json:"expires_at"`
	CanConfirm             bool                           `json:"can_confirm"`
}

// PreviewLessonPlanWordImport 保存并解析一份待确认的DOCX。
func (s *LessonPlanGenService) PreviewLessonPlanWordImport(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	callerID string,
) (*LessonPlanWordImportPreviewResponse, error) {
	if IsGlobalBackgroundDraining() {
		return nil, ErrLPGenServiceDraining
	}
	if file == nil || header == nil {
		return nil, ErrLessonPlanWordFileRequired
	}

	callerID = strings.TrimSpace(callerID)
	if callerID == "" {
		return nil, ErrLPGenUnauthorized
	}

	originalFileName, err := normalizeLessonPlanWordOriginalFileName(
		header.Filename,
	)
	if err != nil {
		return nil, err
	}

	if header.Size > MaxLessonPlanWordFileSize {
		return nil, ErrLessonPlanWordFileTooLarge
	}

	creationDomain, err := resolveImportedLessonPlanCreationDomain(
		ctx,
		callerID,
		defaultLessonPlanImportCreationDeps(),
	)
	if err != nil {
		return nil, err
	}

	storageKey, fullPath, err := createLessonPlanWordImportStoragePath()
	if err != nil {
		return nil, err
	}

	fileSize, fileSHA256, err := saveLessonPlanWordUpload(
		file,
		fullPath,
	)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}

	expiresAt := time.Now().Add(lessonPlanWordImportTTL)
	importSession, err := repository.CreateLessonPlanWordImportSession(
		ctx,
		models.CreateLessonPlanWordImportSessionInput{
			CreatedBy:              callerID,
			EducationDomain:        creationDomain,
			OriginalFileName:       originalFileName,
			StorageKey:             storageKey,
			FileSize:               fileSize,
			MimeType:               models.LessonPlanWordMimeDOCX,
			FileSHA256:             fileSHA256,
			StructureSchemaVersion: models.LessonPlanWordStructureSchemaVersion,
			ExpiresAt:              expiresAt,
		},
	)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("创建Word导入会话失败: %w", err)
	}

	parseResult, parseErr := parseLessonPlanWordDOCX(fullPath)
	if parseErr != nil {
		publicMessage := lessonPlanWordParsePublicMessage(parseErr)

		if markErr := repository.MarkLessonPlanWordImportSessionFailed(
			ctx,
			importSession.ID,
			callerID,
			publicMessage,
		); markErr != nil {
			lpGenLog.Error(
				"保存Word解析失败状态失败",
				"session_id", importSession.ID,
				"error", markErr,
			)
		}

		// 失败文件不参与后续确认，立即删除可显著降低恶意压缩包占用风险。
		if removeErr := os.Remove(fullPath); removeErr != nil && !os.IsNotExist(removeErr) {
			lpGenLog.Warn(
				"删除Word解析失败文件失败",
				"session_id", importSession.ID,
				"path", fullPath,
				"error", removeErr,
			)
		}

		lpGenLog.Warn(
			"Word保真预解析失败",
			"session_id", importSession.ID,
			"file_name", originalFileName,
			"file_size", fileSize,
			"error", parseErr,
		)

		return nil, fmt.Errorf(
			"%w: %s",
			ErrLessonPlanWordParseFailed,
			publicMessage,
		)
	}

	parsedSession, err := repository.MarkLessonPlanWordImportSessionParsed(
		ctx,
		importSession.ID,
		callerID,
		fileSHA256,
		parseResult.Payload,
	)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("保存Word解析结果失败: %w", err)
	}

	lpGenLog.Info(
		"Word保真预解析完成",
		"session_id", parsedSession.ID,
		"caller_id", callerID,
		"education_domain", creationDomain,
		"file_name", originalFileName,
		"file_size", fileSize,
		"table_count", parseResult.Metrics.TableCount,
		"block_count", parseResult.Metrics.BlockCount,
		"image_count", parseResult.Metrics.ImageCount,
		"formula_count", parseResult.Metrics.FormulaCount,
		"warning_count", len(parseResult.Warnings),
	)

	return &LessonPlanWordImportPreviewResponse{
		SessionID:              parsedSession.ID,
		Status:                 parsedSession.Status,
		OriginalFileName:       parsedSession.OriginalFileName,
		FileSize:               parsedSession.FileSize,
		ParserVersion:          parsedSession.ParserVersion,
		StructureSchemaVersion: parsedSession.StructureSchemaVersion,
		SemanticMarkdown:       parseResult.Payload.SemanticMarkdown,
		Metrics:                parseResult.Metrics,
		Warnings:               parseResult.Warnings,
		Document:               parseResult.Document,
		ExpiresAt:              parsedSession.ExpiresAt,
		CanConfirm:             parsedSession.Status == models.LessonPlanWordImportStatusParsed,
	}, nil
}

func normalizeLessonPlanWordOriginalFileName(
	fileName string,
) (string, error) {
	fileName = strings.ReplaceAll(fileName, "\x00", "")
	fileName = strings.TrimSpace(filepath.Base(fileName))

	if fileName == "" ||
		!strings.EqualFold(filepath.Ext(fileName), ".docx") {
		return "", ErrLessonPlanWordFileInvalid
	}

	runes := []rune(fileName)
	if len(runes) > 255 {
		extension := filepath.Ext(fileName)
		baseRunes := []rune(strings.TrimSuffix(fileName, extension))
		maxBaseLength := 255 - utf8.RuneCountInString(extension)
		if maxBaseLength < 1 {
			return "", ErrLessonPlanWordFileInvalid
		}
		if len(baseRunes) > maxBaseLength {
			baseRunes = baseRunes[:maxBaseLength]
		}
		fileName = string(baseRunes) + extension
	}

	return fileName, nil
}

func createLessonPlanWordImportStoragePath() (
	string,
	string,
	error,
) {
	dateDirectory := time.Now().Format("20060102")
	storageKey := filepath.ToSlash(filepath.Join(
		"imports",
		dateDirectory,
		uuid.NewString()+".docx",
	))
	fullPath := filepath.Join(
		LessonPlanWordPrivateRoot,
		filepath.FromSlash(storageKey),
	)

	rootPath := filepath.Clean(LessonPlanWordPrivateRoot)
	cleanFullPath := filepath.Clean(fullPath)
	if cleanFullPath == rootPath ||
		!strings.HasPrefix(cleanFullPath, rootPath+string(os.PathSeparator)) {
		return "", "", errors.New("Word私有存储路径越界")
	}

	if err := os.MkdirAll(filepath.Dir(cleanFullPath), 0o700); err != nil {
		return "", "", fmt.Errorf("创建Word私有目录失败: %w", err)
	}
	if err := os.Chmod(LessonPlanWordPrivateRoot, 0o700); err != nil {
		return "", "", fmt.Errorf("设置Word私有根目录权限失败: %w", err)
	}
	if err := os.Chmod(filepath.Dir(cleanFullPath), 0o700); err != nil {
		return "", "", fmt.Errorf("设置Word导入目录权限失败: %w", err)
	}

	return storageKey, cleanFullPath, nil
}

func saveLessonPlanWordUpload(
	source multipart.File,
	destinationPath string,
) (int64, string, error) {
	destination, err := os.OpenFile(
		destinationPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return 0, "", fmt.Errorf("创建Word私有文件失败: %w", err)
	}

	hash := sha256.New()
	written, copyErr := io.Copy(
		io.MultiWriter(destination, hash),
		io.LimitReader(source, MaxLessonPlanWordFileSize+1),
	)
	if copyErr == nil {
		copyErr = destination.Sync()
	}
	closeErr := destination.Close()

	if copyErr != nil {
		return 0, "", fmt.Errorf("保存Word文件失败: %w", copyErr)
	}
	if closeErr != nil {
		return 0, "", fmt.Errorf("关闭Word文件失败: %w", closeErr)
	}
	if written <= 0 {
		return 0, "", ErrLessonPlanWordFileInvalid
	}
	if written > MaxLessonPlanWordFileSize {
		return 0, "", ErrLessonPlanWordFileTooLarge
	}

	storedFile, err := os.Open(destinationPath)
	if err != nil {
		return 0, "", fmt.Errorf("复核Word文件失败: %w", err)
	}
	defer storedFile.Close()

	var signature [4]byte
	if _, err := io.ReadFull(storedFile, signature[:]); err != nil ||
		string(signature[:]) != "PK\x03\x04" {
		return 0, "", ErrLessonPlanWordFileInvalid
	}

	return written, hex.EncodeToString(hash.Sum(nil)), nil
}

func lessonPlanWordParsePublicMessage(err error) string {
	message := strings.TrimSpace(err.Error())

	switch {
	case strings.Contains(message, "带宏"):
		return "不支持带宏的Word文档，请另存为普通.docx后再上传"
	case strings.Contains(message, "加密"):
		return "该Word文档已加密，请取消密码保护后再上传"
	case strings.Contains(message, "压缩炸弹"),
		strings.Contains(message, "解压后总体积过大"),
		strings.Contains(message, "单个部件过大"):
		return "文档内部体积或压缩比异常，已为安全起见停止解析"
	case strings.Contains(message, "不安全路径"),
		strings.Contains(message, "符号链接"),
		strings.Contains(message, "重复部件"):
		return "文档内部结构不安全，已停止解析"
	case strings.Contains(message, "缺少word/document.xml"),
		strings.Contains(message, "缺少[Content_Types].xml"),
		strings.Contains(message, "不是有效的DOCX"):
		return "该文件不是有效的DOCX文档，请在Word中重新另存为.docx"
	case strings.Contains(message, "未提取到可用教学内容"):
		return "文档中没有识别到可用的教案文字或表格内容"
	default:
		return "暂时无法解析该Word文档，请确认文件未损坏并重新另存为.docx"
	}
}
