package services

// lesson_plan_word_image_deletion_download.go — stale Word的图片删除型派生下载
//
// 目标：老师在平台正文中删除无意义图片后，仍可导出保留原表格和版式的Word。
//
// 严格边界：
//   - 仅接受“原Word语义正文删除零个以上Markdown图片后得到当前正文”；
//   - 不接受文字、表格、标题、公式、图片地址或顺序等其它变化；
//   - 只删除word/document.xml中的对应图片容器，不重建整份Word；
//   - 生成后重新使用正式DOCX解析器解析，并再次和当前正文比对；
//   - 表格数量、行列、合并关系、列宽和嵌套关系必须与原结构完全一致；
//   - 临时DOCX打开后立即解除目录项，文件随响应文件描述符关闭而释放；
//   - 不修改lesson_plan_word_documents，不覆盖任何原始或历史Word版本。

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var errLessonPlanWordImageDeletionUnsupported = errors.New(
	"当前正文变化不属于纯图片删除",
)

var (
	lessonPlanWordMarkdownImagePattern = regexp.MustCompile(
		`!\[[^\]\r\n]*\]\([^\)\r\n]+\)`,
	)

	lessonPlanWordImageRepresentationPattern = regexp.MustCompile(
		`!\[[^\]\r\n]*\]\([^\)\r\n]+\)|\[图片：[^\]\r\n]+\]`,
	)

	lessonPlanWordMarkdownImageURLPattern = regexp.MustCompile(
		`^!\[[^\]\r\n]*\]\(([^\)\r\n]+)\)$`,
	)

	lessonPlanWordBlankLinePattern = regexp.MustCompile(
		`\n{3,}`,
	)

	lessonPlanWordDrawingPattern = regexp.MustCompile(
		`(?s)<w:drawing\b.*?</w:drawing>`,
	)

	lessonPlanWordPictPattern = regexp.MustCompile(
		`(?s)<w:pict\b.*?</w:pict>`,
	)

	lessonPlanWordObjectPattern = regexp.MustCompile(
		`(?s)<w:object\b.*?</w:object>`,
	)

	lessonPlanWordDrawingImageReferencePattern = regexp.MustCompile(
		`<a:blip\b[^>]*\br:(?:embed|link)="([^"]+)"[^>]*>`,
	)

	lessonPlanWordVMLImageReferencePattern = regexp.MustCompile(
		`<v:imagedata\b[^>]*\br:id="([^"]+)"[^>]*>`,
	)
)

type lessonPlanWordImageOccurrence struct {
	GlobalIndex    int
	RelationshipID string
	MarkdownToken  string
	ImageURL       string
}

type lessonPlanWordByteSpan struct {
	Start int
	End   int
}

type lessonPlanWordImageReferenceSpan struct {
	Start          int
	RelationshipID string
}

// openLessonPlanWordImageDeletionDownload 创建只删除指定图片的临时DOCX。
func openLessonPlanWordImageDeletionDownload(
	ctx context.Context,
	record *models.LessonPlanWordDocument,
	ownerID string,
) (*LessonPlanWordDownload, error) {
	if record == nil {
		return nil, ErrLessonPlanWordDownloadUnavailable
	}

	plan, err := repository.GetLessonPlanByID(
		ctx,
		record.LessonPlanID,
	)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrLessonPlanWordDownloadNotFound
		}
		return nil, fmt.Errorf("读取当前教案正文失败: %w", err)
	}

	if strings.TrimSpace(plan.AuthorID) != strings.TrimSpace(ownerID) {
		return nil, ErrLessonPlanWordDownloadNotFound
	}

	var originalDocument LessonPlanWordPreviewDocument
	if err := json.Unmarshal(
		[]byte(record.StructureJSON),
		&originalDocument,
	); err != nil {
		return nil, fmt.Errorf("解析原Word结构快照失败: %w", err)
	}

	occurrences, err := collectLessonPlanWordImageOccurrences(
		originalDocument,
	)
	if err != nil {
		return nil, err
	}

	deletedOccurrences, err := selectLessonPlanWordDeletedImageOccurrences(
		record.SemanticMarkdown,
		plan.ContentMarkdown,
		occurrences,
	)
	if err != nil {
		return nil, err
	}

	verifiedSource, err := openVerifiedLessonPlanWordStoredFile(record)
	if err != nil {
		return nil, err
	}
	defer verifiedSource.File.Close()

	archiveReader, err := zip.NewReader(
		verifiedSource.File,
		verifiedSource.FileInfo.Size(),
	)
	if err != nil {
		return nil, fmt.Errorf("打开原格式Word压缩包失败: %w", err)
	}

	entries, err := validateLessonPlanWordArchive(
		archiveReader.File,
	)
	if err != nil {
		return nil, err
	}

	documentXML, err := readLessonPlanWordZipEntry(
		entries,
		"word/document.xml",
		maxLessonPlanWordXMLBytes,
	)
	if err != nil {
		return nil, err
	}

	modifiedDocumentXML, err := removeLessonPlanWordImageOccurrences(
		documentXML,
		deletedOccurrences,
		len(occurrences),
	)
	if err != nil {
		return nil, err
	}

	temporaryPath, err := writeLessonPlanWordDerivedDOCX(
		archiveReader,
		modifiedDocumentXML,
	)
	if err != nil {
		return nil, err
	}

	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	parseResult, err := parseLessonPlanWordDOCX(temporaryPath)
	if err != nil {
		return nil, fmt.Errorf(
			"重新解析删除图片后的Word失败: %w",
			err,
		)
	}

	retainedImageURLs := make(map[string]string)

	for _, occurrence := range occurrences {
		if deletedOccurrences[occurrence.GlobalIndex] ||
			occurrence.ImageURL == "" {
			continue
		}

		existingURL := retainedImageURLs[occurrence.RelationshipID]
		if existingURL != "" && existingURL != occurrence.ImageURL {
			return nil, errLessonPlanWordImageDeletionUnsupported
		}

		retainedImageURLs[occurrence.RelationshipID] = occurrence.ImageURL
	}

	applyLessonPlanWordImportedImageURLs(
		&parseResult.Document,
		retainedImageURLs,
	)

	derivedSemantic := canonicalizeLessonPlanWordSemanticMarkdown(
		buildLessonPlanWordSemanticMarkdown(
			parseResult.Document,
		),
	)

	currentSemantic := canonicalizeLessonPlanWordSemanticMarkdown(
		plan.ContentMarkdown,
	)

	if derivedSemantic == "" ||
		derivedSemantic != currentSemantic {
		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	if err := validateLessonPlanWordTableLayoutPreserved(
		originalDocument,
		parseResult.Document,
	); err != nil {
		return nil, err
	}

	temporaryFile, err := os.Open(temporaryPath)
	if err != nil {
		return nil, fmt.Errorf("打开删除图片后的临时Word失败: %w", err)
	}

	closeWithError := func(result error) (*LessonPlanWordDownload, error) {
		_ = temporaryFile.Close()
		return nil, result
	}

	temporaryInfo, err := temporaryFile.Stat()
	if err != nil {
		return closeWithError(
			fmt.Errorf("读取临时Word文件状态失败: %w", err),
		)
	}

	if !temporaryInfo.Mode().IsRegular() ||
		temporaryInfo.Size() <= 0 ||
		temporaryInfo.Size() > MaxLessonPlanWordFileSize {
		return closeWithError(ErrLessonPlanWordDownloadUnavailable)
	}

	// Ubuntu允许已打开文件被unlink。解除目录项后，响应仍可通过文件描述符读取，
	// 当处理器关闭File时内核自动释放临时文件，不会留下派生DOCX。
	if err := os.Remove(temporaryPath); err != nil {
		return closeWithError(
			fmt.Errorf("释放临时Word目录项失败: %w", err),
		)
	}

	removeTemporary = false

	return &LessonPlanWordDownload{
		File:     temporaryFile,
		FileName: verifiedSource.FileName,
		Size:     temporaryInfo.Size(),
		ModTime:  temporaryInfo.ModTime(),
	}, nil
}

// collectLessonPlanWordImageOccurrences 按原DOCX正文顺序收集图片运行。
//
// 图片运行与block.markdown中的图片表达必须一一对应：
//   - 成功提取的图片表现为![alt](url)；
//   - 未提取的图片表现为[图片：名称]占位符。
func collectLessonPlanWordImageOccurrences(
	document LessonPlanWordPreviewDocument,
) ([]lessonPlanWordImageOccurrence, error) {
	occurrences := make(
		[]lessonPlanWordImageOccurrence,
		0,
	)

	globalIndex := 0

	for _, block := range document.Blocks {
		imageRuns := make(
			[]LessonPlanWordPreviewRun,
			0,
		)

		for _, run := range block.Runs {
			if run.Kind == "image" {
				imageRuns = append(imageRuns, run)
			}
		}

		if len(imageRuns) == 0 {
			continue
		}

		representations := lessonPlanWordImageRepresentationPattern.
			FindAllString(block.Markdown, -1)

		if len(representations) != len(imageRuns) {
			return nil, errLessonPlanWordImageDeletionUnsupported
		}

		for index, run := range imageRuns {
			relationshipID := strings.TrimSpace(run.RelationshipID)
			if relationshipID == "" {
				return nil, errLessonPlanWordImageDeletionUnsupported
			}

			representation := representations[index]
			markdownToken := ""
			imageURL := ""

			if lessonPlanWordMarkdownImagePattern.MatchString(
				representation,
			) {
				matches := lessonPlanWordMarkdownImageURLPattern.
					FindStringSubmatch(representation)

				if len(matches) != 2 ||
					strings.TrimSpace(matches[1]) == "" {
					return nil, errLessonPlanWordImageDeletionUnsupported
				}

				markdownToken = representation
				imageURL = strings.TrimSpace(matches[1])
			}

			occurrences = append(
				occurrences,
				lessonPlanWordImageOccurrence{
					GlobalIndex:    globalIndex,
					RelationshipID: relationshipID,
					MarkdownToken:  markdownToken,
					ImageURL:       imageURL,
				},
			)

			globalIndex++
		}
	}

	if len(occurrences) == 0 {
		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	return occurrences, nil
}

// selectLessonPlanWordDeletedImageOccurrences 验证当前正文只能由旧正文删除图片得到。
func selectLessonPlanWordDeletedImageOccurrences(
	storedSemantic string,
	currentSemantic string,
	occurrences []lessonPlanWordImageOccurrence,
) (map[int]bool, error) {
	storedCanonical := canonicalizeLessonPlanWordSemanticMarkdown(
		storedSemantic,
	)
	currentCanonical := canonicalizeLessonPlanWordSemanticMarkdown(
		currentSemantic,
	)

	if storedCanonical == "" || currentCanonical == "" {
		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	occurrenceTokens := make([]string, 0)
	tokenToOccurrence := make([]int, 0)

	for _, occurrence := range occurrences {
		if occurrence.MarkdownToken == "" {
			continue
		}

		occurrenceTokens = append(
			occurrenceTokens,
			occurrence.MarkdownToken,
		)
		tokenToOccurrence = append(
			tokenToOccurrence,
			occurrence.GlobalIndex,
		)
	}

	storedTokens := lessonPlanWordMarkdownImagePattern.
		FindAllString(storedCanonical, -1)

	if len(storedTokens) == 0 ||
		len(storedTokens) != len(occurrenceTokens) {
		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	for index := range storedTokens {
		if storedTokens[index] != occurrenceTokens[index] {
			return nil, errLessonPlanWordImageDeletionUnsupported
		}
	}

	deletedTokenIndexes, matched :=
		matchLessonPlanWordImageDeletionOnly(
			storedCanonical,
			currentCanonical,
		)

	if !matched || len(deletedTokenIndexes) == 0 {
		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	deletedOccurrences := make(map[int]bool)

	for tokenIndex := range deletedTokenIndexes {
		if tokenIndex < 0 ||
			tokenIndex >= len(tokenToOccurrence) {
			return nil, errLessonPlanWordImageDeletionUnsupported
		}

		deletedOccurrences[tokenToOccurrence[tokenIndex]] = true
	}

	return deletedOccurrences, nil
}

// canonicalizeLessonPlanWordSemanticMarkdown 只归一换行和多余空行。
//
// 不折叠普通空格、不改变文字或Markdown标记，避免把真实正文修改误判成图片删除。
func canonicalizeLessonPlanWordSemanticMarkdown(
	value string,
) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	lines := strings.Split(value, "\n")
	for index := range lines {
		lines[index] = strings.TrimRight(
			lines[index],
			" \t",
		)
	}

	value = strings.TrimSpace(
		strings.Join(lines, "\n"),
	)

	return lessonPlanWordBlankLinePattern.ReplaceAllString(
		value,
		"\n\n",
	)
}

// removeLessonPlanWordImageOccurrences 从原document.xml删除选定图片容器。
func removeLessonPlanWordImageOccurrences(
	documentXML []byte,
	deletedOccurrences map[int]bool,
	expectedOccurrenceCount int,
) ([]byte, error) {
	if len(documentXML) == 0 ||
		len(deletedOccurrences) == 0 ||
		expectedOccurrenceCount <= 0 {
		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	containerSpans, err := collectLessonPlanWordImageContainerSpans(
		documentXML,
	)
	if err != nil {
		return nil, err
	}

	var result bytes.Buffer
	result.Grow(len(documentXML))

	lastEnd := 0
	seenOccurrences := 0
	removedOccurrences := 0

	for _, span := range containerSpans {
		container := documentXML[span.Start:span.End]
		references := collectLessonPlanWordImageReferenceSpans(
			container,
		)

		if len(references) == 0 {
			continue
		}

		selectedInContainer := 0

		for range references {
			if deletedOccurrences[seenOccurrences] {
				selectedInContainer++
			}
			seenOccurrences++
		}

		result.Write(documentXML[lastEnd:span.Start])

		switch {
		case selectedInContainer == 0:
			result.Write(container)

		case selectedInContainer == len(references):
			removedOccurrences += selectedInContainer

		default:
			// 一个Drawing同时携带多张图片但只删除其中部分时，拒绝猜测内部结构。
			return nil, errLessonPlanWordImageDeletionUnsupported
		}

		lastEnd = span.End
	}

	result.Write(documentXML[lastEnd:])

	if seenOccurrences != expectedOccurrenceCount ||
		removedOccurrences != len(deletedOccurrences) {
		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	return result.Bytes(), nil
}

// collectLessonPlanWordImageContainerSpans 收集互不重叠的drawing、pict和object容器。
func collectLessonPlanWordImageContainerSpans(
	documentXML []byte,
) ([]lessonPlanWordByteSpan, error) {
	allSpans := make([]lessonPlanWordByteSpan, 0)

	for _, pattern := range []*regexp.Regexp{
		lessonPlanWordDrawingPattern,
		lessonPlanWordPictPattern,
		lessonPlanWordObjectPattern,
	} {
		for _, indexes := range pattern.FindAllIndex(documentXML, -1) {
			allSpans = append(
				allSpans,
				lessonPlanWordByteSpan{
					Start: indexes[0],
					End:   indexes[1],
				},
			)
		}
	}

	sort.Slice(
		allSpans,
		func(left int, right int) bool {
			if allSpans[left].Start == allSpans[right].Start {
				return allSpans[left].End > allSpans[right].End
			}
			return allSpans[left].Start < allSpans[right].Start
		},
	)

	filtered := make([]lessonPlanWordByteSpan, 0, len(allSpans))

	for _, span := range allSpans {
		if len(filtered) == 0 {
			filtered = append(filtered, span)
			continue
		}

		previous := filtered[len(filtered)-1]

		if span.Start >= previous.End {
			filtered = append(filtered, span)
			continue
		}

		// object可能完整包裹pict；外层已经收集时跳过内部容器。
		if span.End <= previous.End {
			continue
		}

		return nil, errLessonPlanWordImageDeletionUnsupported
	}

	return filtered, nil
}

// collectLessonPlanWordImageReferenceSpans 读取图片容器中的a:blip或v:imagedata引用。
func collectLessonPlanWordImageReferenceSpans(
	container []byte,
) []lessonPlanWordImageReferenceSpan {
	references := make([]lessonPlanWordImageReferenceSpan, 0)

	for _, pattern := range []*regexp.Regexp{
		lessonPlanWordDrawingImageReferencePattern,
		lessonPlanWordVMLImageReferencePattern,
	} {
		matches := pattern.FindAllSubmatchIndex(container, -1)

		for _, match := range matches {
			if len(match) < 4 ||
				match[2] < 0 ||
				match[3] < 0 {
				continue
			}

			references = append(
				references,
				lessonPlanWordImageReferenceSpan{
					Start: match[0],
					RelationshipID: string(
						container[match[2]:match[3]],
					),
				},
			)
		}
	}

	sort.Slice(
		references,
		func(left int, right int) bool {
			return references[left].Start < references[right].Start
		},
	)

	return references
}

// writeLessonPlanWordDerivedDOCX 逐条复制原DOCX，仅替换word/document.xml。
func writeLessonPlanWordDerivedDOCX(
	source *zip.Reader,
	modifiedDocumentXML []byte,
) (string, error) {
	if source == nil || len(modifiedDocumentXML) == 0 {
		return "", ErrLessonPlanWordDownloadUnavailable
	}

	downloadDirectory := filepath.Join(
		LessonPlanWordPrivateRoot,
		"downloads",
	)

	if err := os.MkdirAll(downloadDirectory, 0o700); err != nil {
		return "", fmt.Errorf("创建Word临时下载目录失败: %w", err)
	}
	if err := os.Chmod(downloadDirectory, 0o700); err != nil {
		return "", fmt.Errorf("设置Word临时下载目录权限失败: %w", err)
	}

	temporaryFile, err := os.CreateTemp(
		downloadDirectory,
		"current-image-deletion-*.docx",
	)
	if err != nil {
		return "", fmt.Errorf("创建Word临时文件失败: %w", err)
	}

	temporaryPath := temporaryFile.Name()
	completed := false

	defer func() {
		if completed {
			return
		}

		_ = temporaryFile.Close()
		_ = os.Remove(temporaryPath)
	}()

	zipWriter := zip.NewWriter(temporaryFile)

	for _, sourceEntry := range source.File {
		header := sourceEntry.FileHeader

		// 由zip.Writer重新计算CRC和压缩大小，避免沿用被替换部件的旧值。
		header.CRC32 = 0
		header.CompressedSize = 0
		header.UncompressedSize = 0
		header.CompressedSize64 = 0
		header.UncompressedSize64 = 0

		destinationEntry, err := zipWriter.CreateHeader(&header)
		if err != nil {
			_ = zipWriter.Close()
			return "", fmt.Errorf(
				"创建Word压缩包部件失败: %w",
				err,
			)
		}

		if sourceEntry.Name == "word/document.xml" {
			if _, err := destinationEntry.Write(
				modifiedDocumentXML,
			); err != nil {
				_ = zipWriter.Close()
				return "", fmt.Errorf(
					"写入修改后的Word正文失败: %w",
					err,
				)
			}
			continue
		}

		if sourceEntry.FileInfo().IsDir() {
			continue
		}

		sourceReader, err := sourceEntry.Open()
		if err != nil {
			_ = zipWriter.Close()
			return "", fmt.Errorf(
				"打开原Word压缩部件失败: %w",
				err,
			)
		}

		_, copyErr := io.Copy(destinationEntry, sourceReader)
		closeErr := sourceReader.Close()

		if copyErr != nil {
			_ = zipWriter.Close()
			return "", fmt.Errorf(
				"复制原Word压缩部件失败: %w",
				copyErr,
			)
		}
		if closeErr != nil {
			_ = zipWriter.Close()
			return "", fmt.Errorf(
				"关闭原Word压缩部件失败: %w",
				closeErr,
			)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("完成Word压缩包写入失败: %w", err)
	}

	if err := temporaryFile.Sync(); err != nil {
		return "", fmt.Errorf("同步Word临时文件失败: %w", err)
	}

	if err := temporaryFile.Close(); err != nil {
		return "", fmt.Errorf("关闭Word临时文件失败: %w", err)
	}

	completed = true
	return temporaryPath, nil
}

// validateLessonPlanWordTableLayoutPreserved 确保图片删除没有改变表格结构和版式元数据。
func validateLessonPlanWordTableLayoutPreserved(
	original LessonPlanWordPreviewDocument,
	derived LessonPlanWordPreviewDocument,
) error {
	if len(original.Tables) != len(derived.Tables) {
		return errLessonPlanWordImageDeletionUnsupported
	}

	for tableIndex := range original.Tables {
		originalTable := original.Tables[tableIndex]
		derivedTable := derived.Tables[tableIndex]

		if originalTable.Index != derivedTable.Index ||
			originalTable.Nested != derivedTable.Nested ||
			originalTable.ParentTableIndex != derivedTable.ParentTableIndex ||
			originalTable.ParentRowIndex != derivedTable.ParentRowIndex ||
			originalTable.ParentCellIndex != derivedTable.ParentCellIndex ||
			!equalLessonPlanWordIntSlices(
				originalTable.GridWidths,
				derivedTable.GridWidths,
			) ||
			len(originalTable.Rows) != len(derivedTable.Rows) {
			return errLessonPlanWordImageDeletionUnsupported
		}

		for rowIndex := range originalTable.Rows {
			originalRow := originalTable.Rows[rowIndex]
			derivedRow := derivedTable.Rows[rowIndex]

			if originalRow.Index != derivedRow.Index ||
				len(originalRow.Cells) != len(derivedRow.Cells) {
				return errLessonPlanWordImageDeletionUnsupported
			}

			for cellIndex := range originalRow.Cells {
				originalCell := originalRow.Cells[cellIndex]
				derivedCell := derivedRow.Cells[cellIndex]

				if originalCell.Index != derivedCell.Index ||
					originalCell.GridSpan != derivedCell.GridSpan ||
					originalCell.VerticalMerge != derivedCell.VerticalMerge ||
					originalCell.WidthTwips != derivedCell.WidthTwips ||
					!equalLessonPlanWordIntSlices(
						originalCell.NestedTableIndices,
						derivedCell.NestedTableIndices,
					) {
					return errLessonPlanWordImageDeletionUnsupported
				}
			}
		}
	}

	return nil
}

func equalLessonPlanWordIntSlices(
	left []int,
	right []int,
) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}
