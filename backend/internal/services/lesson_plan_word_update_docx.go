package services

// lesson_plan_word_update_docx.go — 原格式Word不可变版本文件生成与XML原位修改
//
// 本文件只处理私有DOCX：
//   - 仅改动structure_json明确定位的段落；
//   - 文字按原Word运行边界写回，未改文字运行保持原样；
//   - 老师明确删除的图片只移除其所在运行，不删除其它图片；
//   - 关系文件和媒体部件可保留为未引用资源，不会在页面或Word中显示；
//   - 其余ZIP部件使用CreateRaw原样复制；
//   - 新文件生成后重新解析，逐段复核文字、图片和公式运行。

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type lessonPlanWordXMLTextSpan struct {
	Start         int
	End           int
	OriginalRunes int
}

type lessonPlanWordXMLRunSpan struct {
	Start           int
	End             int
	TextSpans       []lessonPlanWordXMLTextSpan
	RelationshipIDs []string
}

type lessonPlanWordXMLReplacement struct {
	Start int
	End   int
	Data  []byte
}

func createLessonPlanWordPatchedVersion(
	sourcePath string,
	lessonPlanID string,
	nextVersion int,
	paragraphPatches map[int]lessonPlanWordParagraphPatch,
	expectedDocument LessonPlanWordPreviewDocument,
) (string, string, string, error) {
	if strings.TrimSpace(sourcePath) == "" ||
		strings.TrimSpace(lessonPlanID) == "" ||
		nextVersion <= 1 ||
		len(paragraphPatches) == 0 {
		return "", "", "", ErrLessonPlanWordStructureChangeUnsupported
	}

	storageKey := filepath.ToSlash(filepath.Join(
		"documents",
		lessonPlanID,
		"versions",
		fmt.Sprintf("v%06d-%s.docx", nextVersion, uuid.NewString()),
	))
	fullPath, err := resolveLessonPlanWordPrivatePath(storageKey)
	if err != nil {
		return "", "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
		return "", "", "", fmt.Errorf("创建Word版本目录失败: %w", err)
	}
	if err := os.Chmod(filepath.Dir(fullPath), 0o700); err != nil {
		return "", "", "", fmt.Errorf("设置Word版本目录权限失败: %w", err)
	}

	archiveReader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return "", "", "", fmt.Errorf("打开当前Word版本失败: %w", err)
	}
	defer archiveReader.Close()

	entries, err := validateLessonPlanWordArchive(archiveReader.File)
	if err != nil {
		return "", "", "", err
	}
	documentXML, err := readLessonPlanWordZipEntry(
		entries,
		"word/document.xml",
		maxLessonPlanWordXMLBytes,
	)
	if err != nil {
		return "", "", "", err
	}
	patchedDocumentXML, err := patchLessonPlanWordDocumentXML(
		documentXML,
		paragraphPatches,
	)
	if err != nil {
		return "", "", "", err
	}

	temporaryPath := fullPath + ".tmp-" + uuid.NewString()
	output, err := os.OpenFile(
		temporaryPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("创建Word临时版本失败: %w", err)
	}

	completed := false
	defer func() {
		if completed {
			return
		}
		_ = output.Close()
		_ = os.Remove(temporaryPath)
	}()

	zipWriter := zip.NewWriter(output)
	for _, sourceFile := range archiveReader.File {
		header := sourceFile.FileHeader

		if sourceFile.Name == "word/document.xml" {
			writer, createErr := zipWriter.CreateHeader(&header)
			if createErr != nil {
				return "", "", "", fmt.Errorf("创建Word正文部件失败: %w", createErr)
			}
			if _, writeErr := writer.Write(patchedDocumentXML); writeErr != nil {
				return "", "", "", fmt.Errorf("写入Word正文部件失败: %w", writeErr)
			}
			continue
		}

		rawReader, openErr := sourceFile.OpenRaw()
		if openErr != nil {
			return "", "", "", fmt.Errorf("读取Word原始部件失败: %w", openErr)
		}
		writer, createErr := zipWriter.CreateRaw(&header)
		if createErr != nil {
			return "", "", "", fmt.Errorf("复制Word原始部件失败: %w", createErr)
		}
		if _, copyErr := io.Copy(writer, rawReader); copyErr != nil {
			return "", "", "", fmt.Errorf("复制Word原始部件内容失败: %w", copyErr)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return "", "", "", fmt.Errorf("关闭Word压缩包失败: %w", err)
	}
	if err := output.Sync(); err != nil {
		return "", "", "", fmt.Errorf("同步Word临时版本失败: %w", err)
	}
	if err := output.Close(); err != nil {
		return "", "", "", fmt.Errorf("关闭Word临时版本失败: %w", err)
	}
	if err := os.Rename(temporaryPath, fullPath); err != nil {
		return "", "", "", fmt.Errorf("固化Word不可变版本失败: %w", err)
	}
	if err := os.Chmod(fullPath, 0o600); err != nil {
		_ = os.Remove(fullPath)
		return "", "", "", fmt.Errorf("设置Word版本文件权限失败: %w", err)
	}

	completed = true

	generatedArchive, err := zip.OpenReader(fullPath)
	if err != nil {
		_ = os.Remove(fullPath)
		return "", "", "", fmt.Errorf("复核新Word版本失败: %w", err)
	}
	_, validateErr := validateLessonPlanWordArchive(generatedArchive.File)
	closeErr := generatedArchive.Close()
	if validateErr != nil {
		_ = os.Remove(fullPath)
		return "", "", "", validateErr
	}
	if closeErr != nil {
		_ = os.Remove(fullPath)
		return "", "", "", fmt.Errorf("关闭新Word版本复核文件失败: %w", closeErr)
	}

	if err := verifyLessonPlanWordPatchedVersion(fullPath, expectedDocument); err != nil {
		_ = os.Remove(fullPath)
		return "", "", "", err
	}

	hash, err := hashLessonPlanWordFile(fullPath)
	if err != nil {
		_ = os.Remove(fullPath)
		return "", "", "", err
	}

	return storageKey, fullPath, hash, nil
}

func hashLessonPlanWordFile(fullPath string) (string, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("打开Word版本计算哈希失败: %w", err)
	}
	defer file.Close()

	digest := sha256.New()
	written, err := io.Copy(digest, io.LimitReader(file, MaxLessonPlanWordFileSize+1))
	if err != nil {
		return "", fmt.Errorf("计算Word版本哈希失败: %w", err)
	}
	if written <= 0 || written > MaxLessonPlanWordFileSize {
		return "", ErrLessonPlanWordDownloadUnavailable
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func patchLessonPlanWordDocumentXML(
	documentXML []byte,
	paragraphPatches map[int]lessonPlanWordParagraphPatch,
) ([]byte, error) {
	if len(documentXML) == 0 || len(paragraphPatches) == 0 {
		return nil, ErrLessonPlanWordStructureChangeUnsupported
	}

	paragraphRuns, err := locateLessonPlanWordParagraphRuns(
		documentXML,
		paragraphPatches,
	)
	if err != nil {
		return nil, err
	}

	replacements := make([]lessonPlanWordXMLReplacement, 0)
	targetParagraphs := make([]int, 0, len(paragraphPatches))
	for paragraph := range paragraphPatches {
		targetParagraphs = append(targetParagraphs, paragraph)
	}
	sort.Ints(targetParagraphs)

	for _, paragraph := range targetParagraphs {
		patch := paragraphPatches[paragraph]
		runs := paragraphRuns[paragraph]
		if len(runs) == 0 {
			return nil, fmt.Errorf(
				"%w: 第%d个Word段落不存在",
				ErrLessonPlanWordStructureChangeUnsupported,
				paragraph,
			)
		}

		deleteSet := make(map[string]bool)
		for _, relationshipID := range patch.DeleteRelationshipIDs {
			deleteSet[strings.TrimSpace(relationshipID)] = true
		}

		textRunCursor := 0
		deletedFound := make(map[string]bool)
		for _, run := range runs {
			deleteRun := false
			for _, relationshipID := range run.RelationshipIDs {
				if deleteSet[relationshipID] {
					deleteRun = true
					deletedFound[relationshipID] = true
				}
			}
			if deleteRun {
				replacements = append(replacements, lessonPlanWordXMLReplacement{
					Start: run.Start,
					End:   run.End,
				})
				continue
			}

			if patch.TextRuns == nil || len(run.TextSpans) == 0 {
				continue
			}
			if textRunCursor >= len(patch.TextRuns) {
				return nil, ErrLessonPlanWordStructureChangeUnsupported
			}

			originalLengths := make([]int, len(run.TextSpans))
			for index := range run.TextSpans {
				originalLengths[index] = run.TextSpans[index].OriginalRunes
			}
			distributed := distributeLessonPlanWordTextWithinRun(
				patch.TextRuns[textRunCursor],
				originalLengths,
			)
			for index, span := range run.TextSpans {
				var escaped bytes.Buffer
				if err := xml.EscapeText(&escaped, []byte(distributed[index])); err != nil {
					return nil, fmt.Errorf("转义Word段落文字失败: %w", err)
				}
				replacements = append(replacements, lessonPlanWordXMLReplacement{
					Start: span.Start,
					End:   span.End,
					Data:  escaped.Bytes(),
				})
			}
			textRunCursor++
		}

		if patch.TextRuns != nil && textRunCursor != len(patch.TextRuns) {
			return nil, ErrLessonPlanWordStructureChangeUnsupported
		}
		for relationshipID := range deleteSet {
			if relationshipID == "" || !deletedFound[relationshipID] {
				return nil, fmt.Errorf(
					"%w: Word段落中未找到待删除图片关系%s",
					ErrLessonPlanWordStructureChangeUnsupported,
					relationshipID,
				)
			}
		}
	}

	sort.Slice(replacements, func(i, j int) bool {
		if replacements[i].Start == replacements[j].Start {
			return replacements[i].End > replacements[j].End
		}
		return replacements[i].Start > replacements[j].Start
	})

	result := append([]byte(nil), documentXML...)
	lastStart := len(result) + 1
	for _, replacement := range replacements {
		if replacement.Start < 0 ||
			replacement.End < replacement.Start ||
			replacement.End > len(result) ||
			replacement.End > lastStart {
			return nil, errors.New("Word正文替换位置越界或重叠")
		}
		next := make([]byte, 0,
			len(result)-(replacement.End-replacement.Start)+len(replacement.Data))
		next = append(next, result[:replacement.Start]...)
		next = append(next, replacement.Data...)
		next = append(next, result[replacement.End:]...)
		result = next
		lastStart = replacement.Start
	}

	return result, nil
}

func locateLessonPlanWordParagraphRuns(
	documentXML []byte,
	paragraphPatches map[int]lessonPlanWordParagraphPatch,
) (map[int][]lessonPlanWordXMLRunSpan, error) {
	decoder := xml.NewDecoder(bytes.NewReader(documentXML))
	paragraphIndex := 0
	currentParagraph := 0
	deletedDepth := 0
	inText := false
	var currentRun *lessonPlanWordXMLRunSpan
	result := make(map[int][]lessonPlanWordXMLRunSpan)

	for {
		before := int(decoder.InputOffset())
		token, err := decoder.Token()
		after := int(decoder.InputOffset())
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("解析Word正文定位信息失败: %w", err)
		}

		switch item := token.(type) {
		case xml.StartElement:
			switch item.Name.Local {
			case "p":
				paragraphIndex++
				currentParagraph = paragraphIndex
			case "del":
				deletedDepth++
			case "r":
				if currentParagraph > 0 && deletedDepth == 0 {
					if _, wanted := paragraphPatches[currentParagraph]; wanted {
						currentRun = &lessonPlanWordXMLRunSpan{Start: before}
					}
				}
			case "t":
				if currentRun != nil && deletedDepth == 0 {
					inText = true
				}
			case "blip", "imagedata":
				if currentRun != nil && deletedDepth == 0 {
					relationshipID := strings.TrimSpace(xmlAttrValue(item, "embed"))
					if relationshipID == "" {
						relationshipID = strings.TrimSpace(xmlAttrValue(item, "id"))
					}
					if relationshipID == "" {
						relationshipID = strings.TrimSpace(xmlAttrValue(item, "link"))
					}
					if relationshipID != "" {
						currentRun.RelationshipIDs = append(
							currentRun.RelationshipIDs,
							relationshipID,
						)
					}
				}
			}

		case xml.CharData:
			if currentRun != nil && inText && deletedDepth == 0 {
				currentRun.TextSpans = append(
					currentRun.TextSpans,
					lessonPlanWordXMLTextSpan{
						Start:         before,
						End:           after,
						OriginalRunes: utf8.RuneCount(item),
					},
				)
			}

		case xml.EndElement:
			switch item.Name.Local {
			case "t":
				inText = false
			case "r":
				if currentRun != nil {
					currentRun.End = after
					result[currentParagraph] = append(
						result[currentParagraph],
						*currentRun,
					)
					currentRun = nil
				}
			case "del":
				if deletedDepth > 0 {
					deletedDepth--
				}
			case "p":
				currentParagraph = 0
				currentRun = nil
				inText = false
			}
		}
	}

	return result, nil
}

func distributeLessonPlanWordTextWithinRun(
	value string,
	originalLengths []int,
) []string {
	result := make([]string, len(originalLengths))
	if len(originalLengths) == 0 {
		return result
	}

	runes := []rune(value)
	totalOriginal := 0
	for _, length := range originalLengths {
		if length > 0 {
			totalOriginal += length
		}
	}
	if totalOriginal <= 0 {
		result[0] = value
		return result
	}

	consumed := 0
	cumulativeOriginal := 0
	for index := range originalLengths {
		if index == len(originalLengths)-1 {
			result[index] = string(runes[consumed:])
			break
		}
		cumulativeOriginal += originalLengths[index]
		target := len(runes) * cumulativeOriginal / totalOriginal
		if target < consumed {
			target = consumed
		}
		if target > len(runes) {
			target = len(runes)
		}
		result[index] = string(runes[consumed:target])
		consumed = target
	}
	return result
}

func verifyLessonPlanWordPatchedVersion(
	fullPath string,
	expected LessonPlanWordPreviewDocument,
) error {
	parsed, err := parseLessonPlanWordDOCX(fullPath)
	if err != nil {
		return fmt.Errorf("重新解析新Word版本失败: %w", err)
	}

	actualByParagraph := make(map[int]LessonPlanWordPreviewBlock)
	for _, block := range parsed.Document.Blocks {
		actualByParagraph[block.ParagraphIndex] = block
	}

	for _, expectedBlock := range expected.Blocks {
		expectedRuns := normalizeLessonPlanWordComparableRuns(expectedBlock.Runs)
		if len(expectedRuns) == 0 {
			continue
		}
		actualBlock, ok := actualByParagraph[expectedBlock.ParagraphIndex]
		if !ok {
			return fmt.Errorf(
				"新Word版本缺少第%d个预期段落",
				expectedBlock.ParagraphIndex,
			)
		}
		actualRuns := normalizeLessonPlanWordComparableRuns(actualBlock.Runs)
		if !equalLessonPlanWordComparableRuns(expectedRuns, actualRuns) {
			return fmt.Errorf(
				"新Word版本第%d个段落复核不一致",
				expectedBlock.ParagraphIndex,
			)
		}
	}
	return nil
}

type lessonPlanWordComparableRun struct {
	Kind           string
	Text           string
	RelationshipID string
	FormulaID      string
	Bold           bool
	Italic         bool
	Underline      bool
	VerticalAlign  string
}

func normalizeLessonPlanWordComparableRuns(
	runs []LessonPlanWordPreviewRun,
) []lessonPlanWordComparableRun {
	result := make([]lessonPlanWordComparableRun, 0, len(runs))
	for _, run := range runs {
		if run.Kind == "text" && run.Text == "" {
			continue
		}
		result = append(result, lessonPlanWordComparableRun{
			Kind:           run.Kind,
			Text:           run.Text,
			RelationshipID: run.RelationshipID,
			FormulaID:      run.FormulaID,
			Bold:           run.Bold,
			Italic:         run.Italic,
			Underline:      run.Underline,
			VerticalAlign:  run.VerticalAlign,
		})
	}
	return result
}

func equalLessonPlanWordComparableRuns(
	left []lessonPlanWordComparableRun,
	right []lessonPlanWordComparableRun,
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
