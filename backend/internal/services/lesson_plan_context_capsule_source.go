package services

// lesson_plan_context_capsule_source.go — 胶囊权威来源快照装配
//
// 本模块只做确定性读取、校验、截断和哈希，不调用AI、不写数据库。
//
// 来源优先级：
//   1. 教师本轮明确表达；
//   2. active课程大纲知识脉络；
//   3. 已挂载且重新通过运行时硬闸的课本页面；
//   4. 已完成阶段的正式结构化产出和摘要；
//   5. 教案课程定位元数据。
//
// 课本与课程大纲原文不会整体复制进胶囊表。这里只为旁路提取器准备受限输入，
// 并生成source_manifest和后续精准回源所需的来源键、定位信息与内容哈希。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	lessonPlanCapsuleTextbookPageMaxRunes = 6000
	lessonPlanCapsuleTextbookTotalRunes   = 30000
	lessonPlanCapsuleStageOutputMaxRunes  = 3000
	lessonPlanCapsuleEvidenceMaxRunes     = 2000
)

type lessonPlanContextCapsuleSourceEntry struct {
	Key        string                 `json:"key"`
	SourceType string                 `json:"source_type"`
	SourceID   string                 `json:"source_id"`
	Title      string                 `json:"title"`
	Locator    map[string]interface{} `json:"locator"`
	SourceHash string                 `json:"source_hash"`
	Excerpt    string                 `json:"excerpt"`
	Authority  string                 `json:"authority"`
}

type lessonPlanContextCapsuleSourceSnapshot struct {
	Manifest models.LessonPlanContextCapsuleSourceManifest `json:"manifest"`
	Entries  []lessonPlanContextCapsuleSourceEntry         `json:"entries"`
}

func hashLessonPlanContextCapsuleText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func truncateLessonPlanCapsuleSource(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 || value == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}

	return string(runes[:maxRunes])
}

func buildLessonPlanCapsuleTextbookTitle(page *models.TextbookPage) string {
	if page == nil {
		return ""
	}

	textbookName := strings.TrimSpace(page.TextbookName)
	chapter := strings.TrimSpace(page.Chapter)

	switch {
	case textbookName != "" && chapter != "":
		return textbookName + " · " + chapter
	case chapter != "":
		return chapter
	case textbookName != "":
		return textbookName
	default:
		return strings.TrimSpace(page.FileName)
	}
}

func appendLessonPlanCapsuleSourceEntry(
	snapshot *lessonPlanContextCapsuleSourceSnapshot,
	entry lessonPlanContextCapsuleSourceEntry,
) {
	if snapshot == nil || strings.TrimSpace(entry.Key) == "" {
		return
	}

	entry.Key = strings.TrimSpace(entry.Key)
	entry.SourceType = strings.TrimSpace(entry.SourceType)
	entry.SourceID = strings.TrimSpace(entry.SourceID)
	entry.Title = strings.TrimSpace(entry.Title)
	entry.Excerpt = strings.TrimSpace(entry.Excerpt)

	for _, existing := range snapshot.Entries {
		if existing.Key == entry.Key {
			return
		}
	}

	snapshot.Entries = append(snapshot.Entries, entry)
	snapshot.Manifest.Sources = append(
		snapshot.Manifest.Sources,
		models.LessonPlanContextCapsuleSourceRef{
			Key:        entry.Key,
			SourceType: entry.SourceType,
			SourceID:   entry.SourceID,
			Title:      entry.Title,
			Locator:    entry.Locator,
			SourceHash: entry.SourceHash,
		},
	)
}

func loadLessonPlanContextCapsuleSource(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	teacherMessage *models.ConversationMessage,
	turnID string,
) (*lessonPlanContextCapsuleSourceSnapshot, error) {
	if lessonPlan == nil || strings.TrimSpace(lessonPlan.ID) == "" {
		return nil, errors.New("构建胶囊来源失败：教案为空")
	}

	snapshot := &lessonPlanContextCapsuleSourceSnapshot{
		Manifest: models.LessonPlanContextCapsuleSourceManifest{
			LessonPlanID:     lessonPlan.ID,
			Subject:          strings.TrimSpace(lessonPlan.Subject),
			Grade:            strings.TrimSpace(lessonPlan.Grade),
			Topic:            strings.TrimSpace(lessonPlan.Topic),
			EducationDomain:  strings.TrimSpace(lessonPlan.EducationDomain),
			CurrentStageCode: strings.TrimSpace(lessonPlan.CurrentStage),
			Sources:          []models.LessonPlanContextCapsuleSourceRef{},
		},
		Entries: []lessonPlanContextCapsuleSourceEntry{},
	}

	metadataExcerpt := fmt.Sprintf(
		"学科：%s\n年级：%s\n课题：%s\n当前阶段：%s",
		lessonPlan.Subject,
		lessonPlan.Grade,
		lessonPlan.Topic,
		lessonPlan.CurrentStage,
	)
	appendLessonPlanCapsuleSourceEntry(
		snapshot,
		lessonPlanContextCapsuleSourceEntry{
			Key:        "system.lesson_plan",
			SourceType: models.LessonPlanContextCapsuleSourceSystem,
			SourceID:   lessonPlan.ID,
			Title:      "本课基本信息",
			Locator: map[string]interface{}{
				"lesson_plan_id": lessonPlan.ID,
			},
			SourceHash: hashLessonPlanContextCapsuleText(metadataExcerpt),
			Excerpt:    metadataExcerpt,
			Authority:  models.LessonPlanContextCapsuleAuthoritySourceVerified,
		},
	)

	lineage, err := repository.GetActiveLessonPlanKnowledgeLineage(ctx, lessonPlan.ID)
	if err != nil {
		return nil, fmt.Errorf("读取active课程大纲知识脉络失败: %w", err)
	}
	if lineage != nil && lineage.IsActiveUsable() {
		appendLessonPlanCapsuleSourceEntry(
			snapshot,
			lessonPlanContextCapsuleSourceEntry{
				Key:        "course_outline.knowledge_lineage",
				SourceType: models.LessonPlanContextCapsuleSourceCourseOutline,
				SourceID:   lineage.CourseOutlineID,
				Title:      "教师确认后的本课统一知识脉络",
				Locator: map[string]interface{}{
					"knowledge_lineage_id": lineage.ID,
					"course_outline_id":     lineage.CourseOutlineID,
					"confirmed_stage_code": lineage.ConfirmedStageCode,
				},
				SourceHash: lineage.OutlineHash,
				Excerpt: truncateLessonPlanCapsuleSource(
					lineage.ContextText,
					12000,
				),
				Authority: models.LessonPlanContextCapsuleAuthoritySourceVerified,
			},
		)
	}

	rawPageIDs := strings.TrimSpace(lessonPlan.TextbookPageIDs)
	if rawPageIDs != "" && rawPageIDs != "[]" {
		var pageIDs []string
		if err := json.Unmarshal([]byte(rawPageIDs), &pageIDs); err != nil {
			return nil, fmt.Errorf("解析胶囊课本关联失败: %w", err)
		}

		selection, err := validateLessonPlanTextbookSelection(
			ctx,
			lessonPlan.EducationDomain,
			lessonPlan.Subject,
			lessonPlan.Grade,
			pageIDs,
		)
		if err != nil {
			return nil, fmt.Errorf("胶囊课本来源未通过运行时硬闸: %w", err)
		}

		usedTextbookRunes := 0
		for index, pageID := range selection.PageIDs {
			page := selection.PagesByID[pageID]
			if page == nil {
				return nil, ErrLPTextbookSelectionInvalid
			}

			ocrText := strings.TrimSpace(page.OCRText)
			if ocrText == "" {
				continue
			}

			remaining := lessonPlanCapsuleTextbookTotalRunes - usedTextbookRunes
			if remaining <= 0 {
				break
			}

			pageLimit := lessonPlanCapsuleTextbookPageMaxRunes
			if remaining < pageLimit {
				pageLimit = remaining
			}

			excerpt := truncateLessonPlanCapsuleSource(ocrText, pageLimit)
			usedTextbookRunes += len([]rune(excerpt))

			appendLessonPlanCapsuleSourceEntry(
				snapshot,
				lessonPlanContextCapsuleSourceEntry{
					Key:        "textbook." + page.ID,
					SourceType: models.LessonPlanContextCapsuleSourceTextbookPage,
					SourceID:   page.ID,
					Title:      buildLessonPlanCapsuleTextbookTitle(page),
					Locator: map[string]interface{}{
						"textbook_page_id": page.ID,
						"selection_order":  index + 1,
						"chapter":          strings.TrimSpace(page.Chapter),
					},
					SourceHash: hashLessonPlanContextCapsuleText(ocrText),
					Excerpt:    excerpt,
					Authority:  models.LessonPlanContextCapsuleAuthoritySourceVerified,
				},
			)
		}
	}

	outputs, err := repository.ListStageOutputs(ctx, lessonPlan.ID)
	if err != nil {
		return nil, fmt.Errorf("读取胶囊前序阶段产出失败: %w", err)
	}

	for _, output := range outputs {
		if output == nil || output.Status != models.StageOutputCompleted {
			continue
		}

		content := strings.TrimSpace(output.StructuredOutput)
		narrative := strings.TrimSpace(output.NarrativeOutput)
		if content == "" || content == "{}" {
			content = narrative
		} else if narrative != "" {
			content += "\n\n阶段摘要：\n" + narrative
		}
		content = truncateLessonPlanCapsuleSource(
			content,
			lessonPlanCapsuleStageOutputMaxRunes,
		)
		if content == "" {
			continue
		}

		appendLessonPlanCapsuleSourceEntry(
			snapshot,
			lessonPlanContextCapsuleSourceEntry{
				Key:        "stage." + strings.TrimSpace(output.StageCode),
				SourceType: models.LessonPlanContextCapsuleSourceStageOutput,
				SourceID:   lessonPlan.ID + ":" + strings.TrimSpace(output.StageCode),
				Title:      stageCodeToName(output.StageCode) + "已确认产出",
				Locator: map[string]interface{}{
					"lesson_plan_id": lessonPlan.ID,
					"stage_code":     output.StageCode,
					"stage_order":    output.StageOrder,
				},
				SourceHash: hashLessonPlanContextCapsuleText(content),
				Excerpt:    content,
				Authority:  models.LessonPlanContextCapsuleAuthorityTeacherSourceConfirmed,
			},
		)
	}

	if teacherMessage != nil && strings.TrimSpace(teacherMessage.Content) != "" {
		messageID := strings.TrimSpace(teacherMessage.ID)
		if messageID == "" {
			messageID = strings.TrimSpace(turnID)
		}
		if messageID == "" {
			messageID = "current_turn"
		}

		appendLessonPlanCapsuleSourceEntry(
			snapshot,
			lessonPlanContextCapsuleSourceEntry{
				Key:        "teacher." + messageID,
				SourceType: models.LessonPlanContextCapsuleSourceTeacherMessage,
				SourceID:   messageID,
				Title:      "教师本轮明确表达",
				Locator: map[string]interface{}{
					"message_id": messageID,
					"turn_id":    strings.TrimSpace(turnID),
					"stage_code": lessonPlan.CurrentStage,
				},
				SourceHash: hashLessonPlanContextCapsuleText(teacherMessage.Content),
				Excerpt: truncateLessonPlanCapsuleSource(
					teacherMessage.Content,
					lessonPlanCapsuleEvidenceMaxRunes,
				),
				Authority: models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
			},
		)
	}

	return snapshot, nil
}
