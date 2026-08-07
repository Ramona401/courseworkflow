package services

// lesson_plan_word_update_service.go — 原格式Word教案正文安全同步服务
//
// 本文件把平台Markdown修改和原DOCX版式修改收口为同一入口：
//   - 普通教案继续走既有正文版本事务；
//   - Word保真教案先把当前语义正文投影为“固定骨架+可编辑块”；
//   - 只允许修改结构中已有的可编辑块，禁止新增、删除段落或改写表格包装；
//   - 重复文字不再按内容搜索，而是按结构生成时的精确位置定位；
//   - 未明确删除的图片和全部公式默认保留；
//   - 明确从Markdown移除的原Word图片会从对应段落的图片运行中删除；
//   - DOCX、平台正文和两类版本历史在同一数据库事务中提交。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	lessonPlanWordMaxEditableBlocks     = 600
	lessonPlanWordNamespaceTransitional = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	lessonPlanWordNamespaceStrict       = "http://purl.oclc.org/ooxml/wordprocessingml/main"
)

var (
	ErrLessonPlanWordStructureChangeUnsupported = errors.New(
		"本次修改涉及新增、删除或无法定位的Word段落，请保持原有段落和表格结构",
	)
	ErrLessonPlanWordCurrentOutOfSync = errors.New(
		"原格式Word已与当前正文不同步，请先恢复到同步版本后再修改",
	)
	ErrLessonPlanWordMetadataChangeUnsupported = errors.New(
		"原格式Word教案暂不支持同时修改标题或课时时长",
	)
	ErrLessonPlanWordImageAdditionUnsupported = errors.New(
		"原格式Word教案暂不支持通过正文编辑新增图片，请保留原图片或另存普通教案",
	)
	ErrLessonPlanWordFormulaChangeUnsupported = errors.New(
		"原格式Word中的公式当前只允许保留，不支持通过Markdown修改或删除",
	)

	lessonPlanWordImageMarkdownPattern = regexp.MustCompile(
		`!\[[^\]]*\]\([^\)]*\)|\[图片：[^\]]*\]`,
	)
	lessonPlanWordFormulaMarkdownPattern = regexp.MustCompile(
		`\{\{FORMULA-[^\}:]+:[^\}]*\}\}`,
	)
	lessonPlanWordLinkMarkdownPattern = regexp.MustCompile(
		`\[([^\]]+)\]\([^\)]*\)`,
	)
	lessonPlanWordInlineCodePattern       = regexp.MustCompile("`([^`]*)`")
	lessonPlanWordMarkdownEmphasisPattern = regexp.MustCompile(`\*\*|__|\*|_`)
	lessonPlanWordHeadingPrefixPattern    = regexp.MustCompile(`^\s{0,3}#{1,6}\s+`)
	lessonPlanWordListPrefixPattern       = regexp.MustCompile(`^\s*(?:[-+*]|\d+[.)])\s+`)
)

type LessonPlanContentMutationInput struct {
	PlanID            string
	CallerID          string
	Title             string
	ContentMarkdown   string
	ContentStructured string
	DurationMinutes   int
	ExpectedVersion   int
	ExpectedContent   string
	ChangeSource      string
	ChangeSummary     string
}

type LessonPlanContentMutationResult struct {
	Changed         bool
	CurrentVersion  int
	ContentMarkdown string
}

type lessonPlanWordParagraphPatch struct {
	TextRuns              []string
	DeleteRelationshipIDs []string
}

// UpdateLessonPlanContentPreservingWord 更新正文；存在Word保真文档时同步生成新DOCX版本。
func UpdateLessonPlanContentPreservingWord(
	ctx context.Context,
	input LessonPlanContentMutationInput,
) (*LessonPlanContentMutationResult, error) {
	input.PlanID = strings.TrimSpace(input.PlanID)
	input.CallerID = strings.TrimSpace(input.CallerID)
	input.Title = strings.TrimSpace(input.Title)
	input.ContentMarkdown = strings.TrimSpace(input.ContentMarkdown)
	input.ChangeSource = strings.TrimSpace(input.ChangeSource)
	input.ChangeSummary = strings.TrimSpace(input.ChangeSummary)

	if input.PlanID == "" || input.CallerID == "" {
		return nil, ErrLPNotFound
	}
	if input.ContentMarkdown == "" {
		return nil, ErrLPContentEmpty
	}
	if !models.IsValidLessonPlanWordChangeSource(input.ChangeSource) {
		input.ChangeSource = models.LessonPlanWordChangeSourceSystem
	}

	plan, err := repository.GetLessonPlanByID(ctx, input.PlanID)
	if err != nil {
		if errors.Is(err, repository.ErrLessonPlanNotFound) {
			return nil, ErrLPNotFound
		}
		return nil, err
	}
	if plan.AuthorID != input.CallerID {
		return nil, ErrLPNotAuthor
	}
	if !isLessonPlanSectionEditableStatusService(plan.Status) {
		return nil, ErrLPCannotEdit
	}
	if input.ExpectedVersion > 0 && input.ExpectedVersion != plan.Version {
		return nil, ErrLPSectionVersionConflict
	}
	if input.ExpectedContent != "" && input.ExpectedContent != plan.ContentMarkdown {
		return nil, ErrLPSectionVersionConflict
	}

	if input.Title == "" {
		input.Title = plan.Title
	}
	if input.DurationMinutes <= 0 {
		input.DurationMinutes = plan.DurationMinutes
	}
	if input.ContentStructured == "" {
		input.ContentStructured = plan.ContentStructured
	}
	if input.ContentStructured == "" {
		input.ContentStructured = "{}"
	}

	if input.Title == plan.Title &&
		input.ContentMarkdown == plan.ContentMarkdown &&
		input.ContentStructured == plan.ContentStructured &&
		input.DurationMinutes == plan.DurationMinutes {
		return &LessonPlanContentMutationResult{
			Changed:         false,
			CurrentVersion:  plan.Version,
			ContentMarkdown: plan.ContentMarkdown,
		}, nil
	}

	wordDocument, wordErr := repository.GetLessonPlanWordDocumentForOwner(
		ctx,
		input.PlanID,
		input.CallerID,
	)
	if wordErr != nil {
		if !errors.Is(wordErr, repository.ErrLessonPlanWordDocumentNotFound) {
			return nil, wordErr
		}
		return updateLessonPlanContentWithoutWord(
			ctx,
			input,
			plan,
		)
	}

	if input.Title != plan.Title || input.DurationMinutes != plan.DurationMinutes {
		return nil, fmt.Errorf(
			"%w: %w",
			ErrLPCannotEdit,
			ErrLessonPlanWordMetadataChangeUnsupported,
		)
	}
	if wordDocument.Status != models.LessonPlanWordDocumentStatusActive ||
		wordDocument.SemanticMarkdown != plan.ContentMarkdown {
		return nil, fmt.Errorf(
			"%w: %w",
			ErrLPCannotEdit,
			ErrLessonPlanWordCurrentOutOfSync,
		)
	}

	var document LessonPlanWordPreviewDocument
	if err := json.Unmarshal([]byte(wordDocument.StructureJSON), &document); err != nil {
		return nil, fmt.Errorf("读取Word结构失败: %w", err)
	}
	if document.SchemaVersion <= 0 || len(document.Blocks) == 0 ||
		len(document.Blocks) > lessonPlanWordMaxEditableBlocks {
		return nil, fmt.Errorf(
			"%w: Word结构缺少可编辑内容块或规模超限",
			ErrLessonPlanWordStructureChangeUnsupported,
		)
	}

	updatedDocument, paragraphPatches, reconciledMarkdown, err :=
		reconcileLessonPlanWordDocument(
			document,
			wordDocument.SemanticMarkdown,
			input.ContentMarkdown,
		)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLPCannotEdit, err)
	}
	if len(paragraphPatches) == 0 {
		return &LessonPlanContentMutationResult{
			Changed:         false,
			CurrentVersion:  plan.Version,
			ContentMarkdown: plan.ContentMarkdown,
		}, nil
	}

	structureBytes, err := json.Marshal(updatedDocument)
	if err != nil {
		return nil, fmt.Errorf("序列化更新后的Word结构失败: %w", err)
	}
	semanticDigest := sha256.Sum256([]byte(reconciledMarkdown))
	structureDigest := sha256.Sum256(structureBytes)

	verified, err := openVerifiedLessonPlanWordStoredFile(wordDocument)
	if err != nil {
		return nil, err
	}
	sourcePath := verified.Path
	_ = verified.File.Close()

	nextWordVersion := wordDocument.Version + 1
	nextStorageKey, nextFullPath, nextFileSHA256, err :=
		createLessonPlanWordPatchedVersion(
			sourcePath,
			input.PlanID,
			nextWordVersion,
			paragraphPatches,
			updatedDocument,
		)
	if err != nil {
		return nil, err
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if removeErr := os.Remove(nextFullPath); removeErr != nil && !os.IsNotExist(removeErr) {
			lpLog.Warn(
				"清理未提交的Word新版本文件失败",
				"plan_id", input.PlanID,
				"path", nextFullPath,
				"error", removeErr,
			)
		}
	}()

	updateResult, err := repository.CommitLessonPlanWordContentUpdate(
		ctx,
		repository.LessonPlanWordContentUpdateInput{
			LessonPlanID: input.PlanID,
			OwnerID:      input.CallerID,

			ExpectedPlanVersion:    plan.Version,
			ExpectedPlanTitle:      plan.Title,
			ExpectedPlanContent:    plan.ContentMarkdown,
			ExpectedPlanStructured: normalizeLessonPlanWordStructuredJSON(plan.ContentStructured),
			ExpectedPlanDuration:   plan.DurationMinutes,
			ExpectedWordDocumentID: wordDocument.ID,
			ExpectedWordVersion:    wordDocument.Version,
			ExpectedWordStorageKey: wordDocument.CurrentStorageKey,
			ExpectedWordFileSHA256: wordDocument.CurrentFileSHA256,
			ExpectedWordSemantic:   wordDocument.SemanticMarkdown,

			NextTitle:             plan.Title,
			NextContentMarkdown:   reconciledMarkdown,
			NextContentStructured: input.ContentStructured,
			NextDurationMinutes:   plan.DurationMinutes,
			NextWordStorageKey:    nextStorageKey,
			NextWordFileSHA256:    nextFileSHA256,
			NextWordStructureJSON: string(structureBytes),
			NextWordSemanticHash:  hex.EncodeToString(semanticDigest[:]),
			NextWordStructureHash: hex.EncodeToString(structureDigest[:]),
			NextWordMetricsJSON:   updateLessonPlanWordMetricsJSON(wordDocument.MetricsJSON, updatedDocument),
			NextWordWarningsJSON:  wordDocument.WarningsJSON,
			ChangeSource:          input.ChangeSource,
			ChangedBy:             lessonPlanSectionStringPtr(input.CallerID),
			ChangeSummary:         input.ChangeSummary,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrLessonPlanNotFound):
			return nil, ErrLPNotFound
		case errors.Is(err, repository.ErrLessonPlanSectionNotAuthor):
			return nil, ErrLPNotAuthor
		case errors.Is(err, repository.ErrLessonPlanSectionNotEditable):
			return nil, ErrLPCannotEdit
		case errors.Is(err, repository.ErrLessonPlanWordContentUpdateConflict),
			errors.Is(err, repository.ErrLessonPlanWordContentUpdateNotReady):
			return nil, ErrLPSectionVersionConflict
		default:
			return nil, err
		}
	}

	committed = true
	return &LessonPlanContentMutationResult{
		Changed:         true,
		CurrentVersion:  updateResult.LessonPlanVersion,
		ContentMarkdown: reconciledMarkdown,
	}, nil
}

func updateLessonPlanContentWithoutWord(
	ctx context.Context,
	input LessonPlanContentMutationInput,
	plan *models.LessonPlan,
) (*LessonPlanContentMutationResult, error) {
	if plan == nil {
		return nil, ErrLPNotFound
	}

	result, err := repository.CommitLessonPlanContentUpdateCAS(
		ctx,
		repository.LessonPlanContentCASInput{
			LessonPlanID: input.PlanID,
			OwnerID:      input.CallerID,

			ExpectedVersion:           plan.Version,
			ExpectedTitle:             plan.Title,
			ExpectedContentMarkdown:   plan.ContentMarkdown,
			ExpectedContentStructured: normalizeLessonPlanWordStructuredJSON(plan.ContentStructured),
			ExpectedDurationMinutes:   plan.DurationMinutes,

			NextTitle:             input.Title,
			NextContentMarkdown:   input.ContentMarkdown,
			NextContentStructured: input.ContentStructured,
			NextDurationMinutes:   input.DurationMinutes,

			ChangeSource:  input.ChangeSource,
			ChangedBy:     lessonPlanSectionStringPtr(input.CallerID),
			ChangeSummary: input.ChangeSummary,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrLessonPlanNotFound):
			return nil, ErrLPNotFound
		case errors.Is(err, repository.ErrLessonPlanSectionNotAuthor):
			return nil, ErrLPNotAuthor
		case errors.Is(err, repository.ErrLessonPlanSectionNotEditable):
			return nil, ErrLPCannotEdit
		case errors.Is(err, repository.ErrLessonPlanWordContentUpdateConflict):
			return nil, ErrLPSectionVersionConflict
		default:
			return nil, err
		}
	}

	return &LessonPlanContentMutationResult{
		Changed:         result.Changed,
		CurrentVersion:  result.LessonPlanVersion,
		ContentMarkdown: result.ContentMarkdown,
	}, nil
}

func normalizeLessonPlanWordStructuredJSON(raw string) string {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return "{}"
	}
	return normalized
}

func reconcileLessonPlanWordDocument(
	source LessonPlanWordPreviewDocument,
	currentSemantic string,
	requestedSemantic string,
) (
	LessonPlanWordPreviewDocument,
	map[int]lessonPlanWordParagraphPatch,
	string,
	error,
) {
	projection, err := projectLessonPlanWordSemanticMarkdown(source)
	if err != nil {
		return source, nil, "", err
	}
	requestedBlocks, err := extractLessonPlanWordRequestedBlocks(
		projection,
		currentSemantic,
		requestedSemantic,
	)
	if err != nil {
		return source, nil, "", err
	}

	updated := cloneLessonPlanWordPreviewDocument(source)
	paragraphPatches := make(map[int]lessonPlanWordParagraphPatch)

	for blockIndex, requestedBlockMarkdown := range requestedBlocks {
		if blockIndex < 0 || blockIndex >= len(updated.Blocks) {
			return source, nil, "", ErrLessonPlanWordStructureChangeUnsupported
		}
		block := &updated.Blocks[blockIndex]
		if requestedBlockMarkdown == block.Markdown {
			continue
		}
		if !block.Editable && !lessonPlanWordBlockHasProtectedRuns(*block) {
			return source, nil, "", ErrLessonPlanWordStructureChangeUnsupported
		}

		patch, changed, err := reconcileLessonPlanWordBlock(
			block,
			requestedBlockMarkdown,
		)
		if err != nil {
			return source, nil, "", err
		}
		if !changed {
			continue
		}
		if block.ParagraphIndex <= 0 {
			return source, nil, "", ErrLessonPlanWordStructureChangeUnsupported
		}
		if _, exists := paragraphPatches[block.ParagraphIndex]; exists {
			return source, nil, "", ErrLessonPlanWordStructureChangeUnsupported
		}
		paragraphPatches[block.ParagraphIndex] = patch
	}

	if len(paragraphPatches) == 0 {
		return source, paragraphPatches, strings.TrimSpace(currentSemantic), nil
	}

	reconciled := strings.TrimSpace(buildLessonPlanWordSemanticMarkdown(updated))
	if reconciled == "" {
		return source, nil, "", ErrLessonPlanWordStructureChangeUnsupported
	}
	return updated, paragraphPatches, reconciled, nil
}

func cloneLessonPlanWordPreviewDocument(
	source LessonPlanWordPreviewDocument,
) LessonPlanWordPreviewDocument {
	data, err := json.Marshal(source)
	if err != nil {
		return source
	}
	var cloned LessonPlanWordPreviewDocument
	if err := json.Unmarshal(data, &cloned); err != nil {
		return source
	}
	return cloned
}

func reconcileLessonPlanWordBlock(
	block *LessonPlanWordPreviewBlock,
	requestedMarkdown string,
) (lessonPlanWordParagraphPatch, bool, error) {
	if block == nil {
		return lessonPlanWordParagraphPatch{}, false,
			ErrLessonPlanWordStructureChangeUnsupported
	}

	oldImageTokens := lessonPlanWordImageMarkdownPattern.FindAllString(block.Markdown, -1)
	newImageTokens := lessonPlanWordImageMarkdownPattern.FindAllString(requestedMarkdown, -1)
	keepImages, err := matchLessonPlanWordImageTokens(oldImageTokens, newImageTokens)
	if err != nil {
		return lessonPlanWordParagraphPatch{}, false, err
	}

	oldFormulaTokens := lessonPlanWordFormulaMarkdownPattern.FindAllString(block.Markdown, -1)
	newFormulaTokens := lessonPlanWordFormulaMarkdownPattern.FindAllString(requestedMarkdown, -1)
	if !equalLessonPlanWordStrings(oldFormulaTokens, newFormulaTokens) {
		return lessonPlanWordParagraphPatch{}, false,
			ErrLessonPlanWordFormulaChangeUnsupported
	}

	newPlainText := lessonPlanWordMarkdownToPlainText(requestedMarkdown)
	oldPlainText := collectLessonPlanWordTextRuns(*block)
	textChanged := oldPlainText != newPlainText
	if textChanged && lessonPlanWordBlockHasTextBreaks(*block) {
		return lessonPlanWordParagraphPatch{}, false,
			ErrLessonPlanWordStructureChangeUnsupported
	}

	textRunValues := []string(nil)
	if textChanged {
		textRunValues = replaceLessonPlanWordRunTextsPreservingStyles(
			block,
			newPlainText,
		)
	}

	deletedRelationshipIDs := make([]string, 0)
	imageRunIndex := 0
	newRuns := make([]LessonPlanWordPreviewRun, 0, len(block.Runs))
	keptImageTokens := make([]string, 0, len(newImageTokens))

	for _, run := range block.Runs {
		if run.Kind != "image" {
			newRuns = append(newRuns, run)
			continue
		}
		if imageRunIndex >= len(oldImageTokens) || imageRunIndex >= len(keepImages) {
			return lessonPlanWordParagraphPatch{}, false,
				ErrLessonPlanWordStructureChangeUnsupported
		}
		if keepImages[imageRunIndex] {
			newRuns = append(newRuns, run)
			keptImageTokens = append(keptImageTokens, oldImageTokens[imageRunIndex])
		} else {
			relationshipID := strings.TrimSpace(run.RelationshipID)
			if relationshipID == "" {
				return lessonPlanWordParagraphPatch{}, false,
					ErrLessonPlanWordStructureChangeUnsupported
			}
			deletedRelationshipIDs = append(deletedRelationshipIDs, relationshipID)
		}
		imageRunIndex++
	}
	if imageRunIndex != len(oldImageTokens) {
		return lessonPlanWordParagraphPatch{}, false,
			ErrLessonPlanWordStructureChangeUnsupported
	}

	block.Runs = newRuns
	rebuildLessonPlanWordBlockMarkdown(block, keptImageTokens)

	changed := textChanged || len(deletedRelationshipIDs) > 0
	if !changed {
		return lessonPlanWordParagraphPatch{}, false, nil
	}

	return lessonPlanWordParagraphPatch{
		TextRuns:              textRunValues,
		DeleteRelationshipIDs: deletedRelationshipIDs,
	}, true, nil
}

func matchLessonPlanWordImageTokens(
	oldTokens []string,
	newTokens []string,
) ([]bool, error) {
	if len(newTokens) > len(oldTokens) {
		return nil, ErrLessonPlanWordImageAdditionUnsupported
	}

	counts := make(map[string]int)
	for _, token := range oldTokens {
		counts[token]++
	}
	for token, count := range counts {
		if count > 1 && countLessonPlanWordString(newTokens, token) != count {
			return nil, ErrLessonPlanWordStructureChangeUnsupported
		}
	}

	keep := make([]bool, len(oldTokens))
	oldCursor := 0
	for _, token := range newTokens {
		found := -1
		for index := oldCursor; index < len(oldTokens); index++ {
			if oldTokens[index] == token {
				found = index
				break
			}
		}
		if found < 0 {
			return nil, ErrLessonPlanWordImageAdditionUnsupported
		}
		keep[found] = true
		oldCursor = found + 1
	}
	return keep, nil
}

func countLessonPlanWordString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}

func equalLessonPlanWordStrings(left []string, right []string) bool {
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

func lessonPlanWordMarkdownToPlainText(markdown string) string {
	value := lessonPlanWordImageMarkdownPattern.ReplaceAllString(markdown, "")
	value = lessonPlanWordFormulaMarkdownPattern.ReplaceAllString(value, "")
	value = lessonPlanWordLinkMarkdownPattern.ReplaceAllString(value, "$1")
	value = lessonPlanWordInlineCodePattern.ReplaceAllString(value, "$1")
	value = lessonPlanWordMarkdownEmphasisPattern.ReplaceAllString(value, "")

	lines := strings.Split(value, "\n")
	cleanedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = lessonPlanWordHeadingPrefixPattern.ReplaceAllString(line, "")
		line = lessonPlanWordListPrefixPattern.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	value = strings.Join(cleanedLines, "；")
	value = strings.Map(func(valueRune rune) rune {
		if unicode.IsControl(valueRune) && valueRune != '\t' {
			return ' '
		}
		return valueRune
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func collectLessonPlanWordTextRuns(block LessonPlanWordPreviewBlock) string {
	var builder strings.Builder
	for _, run := range block.Runs {
		if run.Kind == "text" {
			builder.WriteString(run.Text)
		}
	}
	return builder.String()
}

func lessonPlanWordBlockHasTextBreaks(block LessonPlanWordPreviewBlock) bool {
	for _, run := range block.Runs {
		if run.Kind == "text" && strings.ContainsAny(run.Text, "\n\t") {
			return true
		}
	}
	return false
}

func lessonPlanWordBlockHasProtectedRuns(block LessonPlanWordPreviewBlock) bool {
	for _, run := range block.Runs {
		if run.Kind == "image" || run.Kind == "formula" {
			return true
		}
	}
	return false
}

func replaceLessonPlanWordRunTextsPreservingStyles(
	block *LessonPlanWordPreviewBlock,
	newText string,
) []string {
	textRunIndexes := make([]int, 0)
	oldRunTexts := make([]string, 0)
	for index := range block.Runs {
		if block.Runs[index].Kind != "text" {
			continue
		}
		textRunIndexes = append(textRunIndexes, index)
		oldRunTexts = append(oldRunTexts, block.Runs[index].Text)
	}
	if len(textRunIndexes) == 0 {
		return nil
	}

	oldText := strings.Join(oldRunTexts, "")
	oldRunes := []rune(oldText)
	newRunes := []rune(newText)
	prefix := commonLessonPlanWordPrefixRunes(oldRunes, newRunes)
	suffix := commonLessonPlanWordSuffixRunes(oldRunes[prefix:], newRunes[prefix:])
	oldChangeEnd := len(oldRunes) - suffix
	newChangeEnd := len(newRunes) - suffix
	newMiddle := string(newRunes[prefix:newChangeEnd])

	boundaries := make([]int, len(oldRunTexts)+1)
	for index, text := range oldRunTexts {
		boundaries[index+1] = boundaries[index] + utf8.RuneCountInString(text)
	}

	targetRun := len(oldRunTexts) - 1
	if prefix == 0 && suffix == 0 {
		largestLength := -1
		for index, text := range oldRunTexts {
			length := utf8.RuneCountInString(text)
			if length > largestLength {
				largestLength = length
				targetRun = index
			}
		}
	} else {
		for index := range oldRunTexts {
			if prefix < boundaries[index+1] ||
				(prefix == boundaries[index+1] && prefix == boundaries[index]) {
				targetRun = index
				break
			}
		}
	}

	result := make([]string, len(oldRunTexts))
	for index := range oldRunTexts {
		runStart := boundaries[index]
		runEnd := boundaries[index+1]
		var builder strings.Builder

		if runStart < prefix {
			keepEnd := prefix
			if keepEnd > runEnd {
				keepEnd = runEnd
			}
			builder.WriteString(string(oldRunes[runStart:keepEnd]))
		}
		if index == targetRun {
			builder.WriteString(newMiddle)
		}
		if runEnd > oldChangeEnd {
			keepStart := oldChangeEnd
			if keepStart < runStart {
				keepStart = runStart
			}
			builder.WriteString(string(oldRunes[keepStart:runEnd]))
		}

		result[index] = builder.String()
		block.Runs[textRunIndexes[index]].Text = result[index]
	}
	return result
}

func commonLessonPlanWordPrefixRunes(left []rune, right []rune) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	index := 0
	for index < limit && left[index] == right[index] {
		index++
	}
	return index
}

func commonLessonPlanWordSuffixRunes(left []rune, right []rune) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	count := 0
	for count < limit && left[len(left)-1-count] == right[len(right)-1-count] {
		count++
	}
	return count
}

func rebuildLessonPlanWordBlockMarkdown(
	block *LessonPlanWordPreviewBlock,
	imageTokens []string,
) {
	textParts := make([]string, 0, len(block.Runs))
	markdownParts := make([]string, 0, len(block.Runs))
	imageIndex := 0
	editable := false

	for _, run := range block.Runs {
		switch run.Kind {
		case "text":
			if run.Text == "" {
				continue
			}
			editable = true
			textParts = append(textParts, run.Text)
			markdownParts = append(markdownParts, renderLessonPlanWordRunMarkdown(run))

		case "image":
			label := "图片"
			if strings.TrimSpace(run.MediaTarget) != "" {
				label = path.Base(run.MediaTarget)
			}
			textParts = append(textParts, "[图片："+label+"]")
			if imageIndex < len(imageTokens) {
				markdownParts = append(markdownParts, imageTokens[imageIndex])
			} else {
				markdownParts = append(markdownParts, "[图片："+label+"]")
			}
			imageIndex++

		case "formula":
			textParts = append(textParts, "[公式："+run.Text+"]")
			markdownParts = append(
				markdownParts,
				"{{"+strings.ToUpper(run.FormulaID)+":"+run.Text+"}}",
			)
		}
	}

	block.Text = strings.TrimSpace(strings.Join(textParts, ""))
	block.Markdown = strings.TrimSpace(strings.Join(markdownParts, ""))
	block.Editable = editable
}

func updateLessonPlanWordMetricsJSON(
	raw string,
	document LessonPlanWordPreviewDocument,
) string {
	metrics := make(map[string]any)
	if err := json.Unmarshal([]byte(normalizeLessonPlanWordStructuredJSON(raw)), &metrics); err != nil {
		metrics = make(map[string]any)
	}

	imageCount := 0
	uniqueImages := make(map[string]bool)
	editableCount := 0
	for _, block := range document.Blocks {
		if block.Editable {
			editableCount++
		}
		for _, run := range block.Runs {
			if run.Kind != "image" {
				continue
			}
			imageCount++
			if strings.TrimSpace(run.RelationshipID) != "" {
				uniqueImages[run.RelationshipID] = true
			}
		}
	}
	metrics["image_count"] = imageCount
	metrics["unique_image_count"] = len(uniqueImages)
	metrics["editable_block_count"] = editableCount

	data, err := json.Marshal(metrics)
	if err != nil {
		return "{}"
	}
	return string(data)
}
