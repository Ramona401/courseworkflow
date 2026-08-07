package services

// lesson_plan_word_revision_stream.go — 原格式Word正式修订的流式执行策略
//
// 仅对active且与正式正文同步的原Word修订启用真实流式候选。
// 课本、教师附件、单元方案、原始大纲和班级学情等硬证据仍走阻塞Harness。
// 流式结束后先把“表格N · 第N行/第N列”原Word模板提取成正式候选，
// 再通过原Word段落、表格、图片、公式与双版本事务校验。
// 任何提取或校验失败都不会写入正式正文、消息历史或新的DOCX版本。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrLessonPlanWordRevisionCandidateMissing 表示流式回复没有包含可定位的完整原Word模板。
	ErrLessonPlanWordRevisionCandidateMissing = errors.New(
		"流式回复没有包含可定位的完整原Word模板",
	)

	// ErrLessonPlanWordProtectedImageChanged 表示自动修订漏掉、重排或改写了原图片标记。
	ErrLessonPlanWordProtectedImageChanged = errors.New(
		"自动修订改动了原Word受保护图片标记",
	)
)

// canStreamLessonPlanWordRevision 是不访问数据库的纯策略判断。
func canStreamLessonPlanWordRevision(
	currentStage string,
	turnPlan *lessonPlanTurnContextPlan,
	hasActiveSynchronizedWord bool,
) bool {
	if !hasActiveSynchronizedWord ||
		turnPlan == nil ||
		!turnPlan.FormalArtifact ||
		!strings.EqualFold(
			strings.TrimSpace(currentStage),
			"revise",
		) {
		return false
	}

	return !turnPlan.UseTextbook &&
		!turnPlan.UseRefMaterial &&
		!turnPlan.UseUnitPlan &&
		!turnPlan.UseRawCourseOutline &&
		!turnPlan.UseClassProfile
}

// shouldStreamLessonPlanWordRevision 查询当前教案是否存在active且同步的原Word。
func (s *LessonPlanGenService) shouldStreamLessonPlanWordRevision(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	currentStage string,
	turnPlan *lessonPlanTurnContextPlan,
) bool {
	if lessonPlan == nil ||
		strings.TrimSpace(lessonPlan.ID) == "" ||
		strings.TrimSpace(lessonPlan.AuthorID) == "" ||
		!canStreamLessonPlanWordRevision(
			currentStage,
			turnPlan,
			true,
		) {
		return false
	}

	wordDocument, err :=
		repository.GetLessonPlanWordDocumentForOwner(
			ctx,
			lessonPlan.ID,
			lessonPlan.AuthorID,
		)
	if err != nil {
		if !errors.Is(
			err,
			repository.ErrLessonPlanWordDocumentNotFound,
		) {
			lpGenLog.Warn(
				"解析原格式Word流式修订策略失败，继续使用既有Harness",
				"plan_id", lessonPlan.ID,
				"error", err,
			)
		}
		return false
	}

	synchronized :=
		wordDocument.Status ==
			models.LessonPlanWordDocumentStatusActive &&
			strings.TrimSpace(
				wordDocument.SemanticMarkdown,
			) != "" &&
			wordDocument.SemanticMarkdown ==
				lessonPlan.ContentMarkdown

	enabled :=
		canStreamLessonPlanWordRevision(
			currentStage,
			turnPlan,
			synchronized,
		)
	if enabled {
		lpGenLog.Info(
			"原格式Word修订启用流式候选与确定性提交",
			"plan_id", lessonPlan.ID,
			"stage", currentStage,
		)
	}
	return enabled
}

// extractLessonPlanStageArtifact 统一选择普通阶段提取或原Word流式提取。
//
// 原Word流式修订必须在这里得到正式候选；提取失败直接返回错误，调用方不得
// appendMessage、message_done、建议芯片或更新共识胶囊。
func (s *LessonPlanGenService) extractLessonPlanStageArtifact(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	rawContent string,
	streamWordRevision bool,
) (
	string,
	string,
	bool,
	error,
) {
	if !streamWordRevision {
		structuredJSON,
			narrative,
			hasContent :=
			ExtractStructuredFromNaturalReply(
				stageCode,
				rawContent,
			)
		return structuredJSON,
			narrative,
			hasContent,
			nil
	}

	structuredJSON,
		narrative,
		err :=
		s.extractStreamedLessonPlanWordRevision(
			ctx,
			lessonPlan,
			rawContent,
		)
	if err != nil {
		return "", "", false, err
	}

	return structuredJSON, narrative, true, nil
}

// extractStreamedLessonPlanWordRevision 从流式回复中提取并校验原Word正式候选。
func (s *LessonPlanGenService) extractStreamedLessonPlanWordRevision(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	rawContent string,
) (
	string,
	string,
	error,
) {
	if lessonPlan == nil {
		return "", "",
			ErrLessonPlanWordCurrentOutOfSync
	}

	wordDocument, err :=
		repository.GetLessonPlanWordDocumentForOwner(
			ctx,
			lessonPlan.ID,
			lessonPlan.AuthorID,
		)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrLessonPlanWordDocumentNotFound,
		) {
			return "", "",
				ErrLessonPlanWordCurrentOutOfSync
		}
		return "", "", err
	}

	if wordDocument.Status !=
		models.LessonPlanWordDocumentStatusActive ||
		wordDocument.SemanticMarkdown !=
			lessonPlan.ContentMarkdown {
		return "", "",
			ErrLessonPlanWordCurrentOutOfSync
	}

	candidate,
		suggestionText :=
		extractLessonPlanWordRevisionCandidate(
			rawContent,
			wordDocument.SemanticMarkdown,
		)
	if candidate == "" {
		return "", "",
			ErrLessonPlanWordRevisionCandidateMissing
	}

	currentImages :=
		lessonPlanWordImageMarkdownPattern.
			FindAllString(
				wordDocument.SemanticMarkdown,
				-1,
			)
	candidateImages :=
		lessonPlanWordImageMarkdownPattern.
			FindAllString(
				candidate,
				-1,
			)
	if !equalLessonPlanWordStrings(
		currentImages,
		candidateImages,
	) {
		return "", "",
			ErrLessonPlanWordProtectedImageChanged
	}

	structuredJSONBytes, marshalErr :=
		json.Marshal(
			map[string]interface{}{
				"content_markdown": candidate,
			},
		)
	if marshalErr != nil {
		return "", "",
			fmt.Errorf(
				"序列化原Word流式修订候选失败: %w",
				marshalErr,
			)
	}

	narrative :=
		fmt.Sprintf(
			"已生成原格式Word修订候选（%d字符），正在执行结构与版本提交。",
			utf8.RuneCountInString(candidate),
		)
	narrative =
		appendSuggestionToNarrative(
			narrative,
			suggestionText,
		)

	return string(structuredJSONBytes),
		narrative,
		nil
}

// extractLessonPlanWordRevisionCandidate 从带说明文字的回复中截取原Word模板。
//
// 先以当前正式Word模板的首个结构锚点定位，避免通用教案提取器从“课题”
// 中途截断并丢掉表格包装；找不到结构锚点时才使用通用完整教案检测兜底。
func extractLessonPlanWordRevisionCandidate(
	rawContent string,
	baseline string,
) (
	string,
	string,
) {
	pureContent,
		suggestionText :=
		splitSuggestionBlock(
			rawContent,
		)
	pureContent =
		strings.TrimSpace(
			StripSuggestedActionsBlock(
				pureContent,
			),
		)
	if pureContent == "" {
		return "", suggestionText
	}

	candidate :=
		extractLessonPlanWordCandidateFromAnchor(
			pureContent,
			baseline,
		)
	if candidate == "" {
		candidate =
			strings.TrimSpace(
				DetectLessonPlanContent(
					pureContent,
				),
			)
	}
	if candidate == "" {
		return "", suggestionText
	}

	candidate =
		trimLessonPlanWordRevisionTrailingNarrative(
			candidate,
		)
	candidate =
		collapseLessonPlanWordEmptySlotContinuations(
			baseline,
			candidate,
		)

	return strings.TrimSpace(candidate),
		suggestionText
}

// extractLessonPlanWordCandidateFromAnchor 从当前Word模板首个结构行开始截取。
func extractLessonPlanWordCandidateFromAnchor(
	content string,
	baseline string,
) string {
	anchors :=
		lessonPlanWordCandidateAnchors(
			baseline,
		)

	lines :=
		strings.Split(
			content,
			"\n",
		)
	for index, line := range lines {
		normalized :=
			normalizeLessonPlanWordAnchorLine(
				line,
			)
		if normalized == "" {
			continue
		}
		if _, ok :=
			anchors[normalized]; ok {
			return strings.TrimSpace(
				strings.Join(
					lines[index:],
					"\n",
				),
			)
		}
	}

	// 最后兜底：AI保留了通用“表格N · 第N行”包装，但首行文字略有漂移。
	for index, line := range lines {
		if isLessonPlanWordTableAnchorLine(
			normalizeLessonPlanWordAnchorLine(
				line,
			),
		) {
			return strings.TrimSpace(
				strings.Join(
					lines[index:],
					"\n",
				),
			)
		}
	}

	return ""
}

// lessonPlanWordCandidateAnchors 返回优先级最高的少量基线结构锚点。
func lessonPlanWordCandidateAnchors(
	baseline string,
) map[string]struct{} {
	result :=
		make(
			map[string]struct{},
		)

	for _, line := range strings.Split(
		baseline,
		"\n",
	) {
		normalized :=
			normalizeLessonPlanWordAnchorLine(
				line,
			)
		if normalized == "" {
			continue
		}
		if isLessonPlanWordTableAnchorLine(
			normalized,
		) {
			result[normalized] =
				struct{}{}
			return result
		}
	}

	// 非表格Word使用第一条非空正式行兜底。
	if len(result) == 0 {
		for _, line := range strings.Split(
			baseline,
			"\n",
		) {
			normalized :=
				normalizeLessonPlanWordAnchorLine(
					line,
				)
			if normalized != "" {
				result[normalized] =
					struct{}{}
				break
			}
		}
	}

	return result
}

func normalizeLessonPlanWordAnchorLine(
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)
	value =
		strings.TrimSpace(
			strings.TrimLeft(
				value,
				"#",
			),
		)
	return strings.Join(
		strings.Fields(value),
		"",
	)
}

func isLessonPlanWordTableAnchorLine(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"表格",
	) &&
		strings.Contains(
			value,
			"·第",
		) &&
		strings.HasSuffix(
			value,
			"行",
		)
}

// trimLessonPlanWordRevisionTrailingNarrative 去掉模板后的说明或UI文字。
func trimLessonPlanWordRevisionTrailingNarrative(
	candidate string,
) string {
	lines :=
		strings.Split(
			candidate,
			"\n",
		)
	cut := len(lines)

	for index, line := range lines {
		trimmed :=
			strings.TrimSpace(
				line,
			)
		if index == 0 {
			continue
		}
		for _, prefix := range []string{
			"💡 我的补充建议",
			"🔄 重新回答",
			"🎉 完成并发布",
		} {
			if strings.HasPrefix(
				trimmed,
				prefix,
			) {
				cut = index
				break
			}
		}
		if cut != len(lines) {
			break
		}
	}

	return trimTrailingChatter(
		strings.Join(
			lines[:cut],
			"\n",
		),
	)
}

// collapseLessonPlanWordEmptySlotContinuations 把空字段新增内容收回原段落。
//
// 典型原Word结构为“课前预习阶段：/课堂教学阶段：/课后提升阶段：”
// 三个空段落。模型常把补写内容另起一段，导致Word结构数增加。这里仅对
// 基线中确认为空槽的短冒号标签进行确定性合并，不改其它段落和表格包装。
func collapseLessonPlanWordEmptySlotContinuations(
	baseline string,
	candidate string,
) string {
	emptyLabels :=
		lessonPlanWordEmptySlotLabels(
			baseline,
		)
	if len(emptyLabels) == 0 {
		return candidate
	}

	baselineLines :=
		make(
			map[string]struct{},
		)
	for _, line := range strings.Split(
		baseline,
		"\n",
	) {
		trimmed :=
			strings.TrimSpace(
				line,
			)
		if trimmed != "" {
			baselineLines[trimmed] =
				struct{}{}
		}
	}

	lines :=
		strings.Split(
			candidate,
			"\n",
		)
	result :=
		make(
			[]string,
			0,
			len(lines),
		)

	for index := 0; index < len(lines); {
		trimmed :=
			strings.TrimSpace(
				lines[index],
			)
		if _, ok :=
			emptyLabels[trimmed]; !ok {
			result = append(
				result,
				lines[index],
			)
			index++
			continue
		}

		continuation :=
			make(
				[]string,
				0,
			)
		next := index + 1

		for next < len(lines) {
			nextTrimmed :=
				strings.TrimSpace(
					lines[next],
				)
			if nextTrimmed == "" {
				next++
				continue
			}
			if _, isEmptyLabel :=
				emptyLabels[nextTrimmed]; isEmptyLabel {
				break
			}
			if _, isBaselineLine :=
				baselineLines[nextTrimmed]; isBaselineLine {
				break
			}

			continuation = append(
				continuation,
				nextTrimmed,
			)
			next++
		}

		if len(continuation) == 0 {
			result = append(
				result,
				lines[index],
			)
			index++
			continue
		}

		result = append(
			result,
			strings.TrimSpace(
				lines[index],
			)+strings.Join(
				continuation,
				" ",
			),
		)
		index = next
	}

	return strings.Join(
		result,
		"\n",
	)
}

// lessonPlanWordEmptySlotLabels 找出基线中后面没有正文的短冒号字段。
func lessonPlanWordEmptySlotLabels(
	baseline string,
) map[string]struct{} {
	lines :=
		strings.Split(
			baseline,
			"\n",
		)
	result :=
		make(
			map[string]struct{},
		)

	for index, line := range lines {
		trimmed :=
			strings.TrimSpace(
				line,
			)
		if trimmed == "" ||
			utf8.RuneCountInString(trimmed) > 40 ||
			(!strings.HasSuffix(trimmed, "：") &&
				!strings.HasSuffix(trimmed, ":")) {
			continue
		}

		nextNonEmpty := ""
		for next := index + 1; next < len(lines); next++ {
			nextNonEmpty =
				strings.TrimSpace(
					lines[next],
				)
			if nextNonEmpty != "" {
				break
			}
		}

		if nextNonEmpty == "" ||
			(strings.HasSuffix(nextNonEmpty, "：") &&
				utf8.RuneCountInString(nextNonEmpty) <= 40) ||
			(strings.HasSuffix(nextNonEmpty, ":") &&
				utf8.RuneCountInString(nextNonEmpty) <= 40) {
			result[trimmed] =
				struct{}{}
		}
	}

	return result
}
