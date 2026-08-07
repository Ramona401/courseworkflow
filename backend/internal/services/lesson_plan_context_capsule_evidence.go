package services

// lesson_plan_context_capsule_evidence.go — 胶囊稳定语义哈希与原文召回路由
//
// 稳定语义哈希故意忽略：
//   - summary展示措辞；
//   - stage_focus；
//   - updated_by_turn_id；
//   - source_manifest；
//   - last_turn_id。
//
// 这些字段变化不代表课程核心或教师共识变化，不应制造新版本。
//
// 原文证据路由只保存：
//   - 来源类型和来源ID；
//   - 页码、章节、消息ID或阶段代码等定位信息；
//   - 来源哈希和短证据片段；
//   - 不保存整份课本、课程大纲或附件全文。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"tedna/internal/models"
)

// hashLessonPlanContextCapsuleVersion 生成稳定语义哈希。
//
// 通过深拷贝清除展示摘要、阶段焦点和轮次标记，确保这些非核心变化不会产生新版本。
func hashLessonPlanContextCapsuleVersion(
	document *models.LessonPlanContextCapsuleDocument,
) (string, error) {
	if document == nil {
		return "", fmt.Errorf(
			"生成胶囊语义哈希失败：文档为空",
		)
	}

	encoded, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf(
			"生成胶囊语义哈希失败: %w",
			err,
		)
	}

	stable := &models.LessonPlanContextCapsuleDocument{}
	if err := json.Unmarshal(
		encoded,
		stable,
	); err != nil {
		return "", fmt.Errorf(
			"构建胶囊稳定副本失败: %w",
			err,
		)
	}

	stable.Summary = ""
	stable.StageFocus =
		models.LessonPlanContextCapsuleStageFocus{}

	clearTurnIDs := func(
		items []models.LessonPlanContextCapsuleItem,
	) {
		for index := range items {
			items[index].UpdatedByTurnID = ""
		}
	}

	clearTurnIDs(stable.CourseCore)
	clearTurnIDs(stable.TeachingConsensus)
	clearTurnIDs(stable.Constraints)
	clearTurnIDs(stable.OpenQuestions)
	clearTurnIDs(stable.DeferredItems)
	clearTurnIDs(stable.SupersededItems)

	stableJSON, err := json.Marshal(stable)
	if err != nil {
		return "", fmt.Errorf(
			"序列化胶囊稳定副本失败: %w",
			err,
		)
	}

	sum := sha256.Sum256(stableJSON)
	return hex.EncodeToString(sum[:]), nil
}

// buildLessonPlanContextCapsuleEvidence 构建当前版本证据路由。
func buildLessonPlanContextCapsuleEvidence(
	lessonPlanID string,
	document *models.LessonPlanContextCapsuleDocument,
	sourceEntries []lessonPlanContextCapsuleSourceEntry,
) []models.LessonPlanContextCapsuleEvidence {
	sourceByKey := make(
		map[string]lessonPlanContextCapsuleSourceEntry,
		len(sourceEntries),
	)

	teacherKeys := make([]string, 0, 1)

	for _, entry := range sourceEntries {
		sourceByKey[entry.Key] = entry

		if entry.SourceType ==
			models.LessonPlanContextCapsuleSourceTeacherMessage {
			teacherKeys = append(
				teacherKeys,
				entry.Key,
			)
		}
	}

	items := collectLessonPlanContextCapsuleItems(
		document,
	)

	evidence := make(
		[]models.LessonPlanContextCapsuleEvidence,
		0,
	)

	for _, item := range items {
		sourceKeys := item.SourceKeys

		if len(sourceKeys) == 0 &&
			item.Authority ==
				models.LessonPlanContextCapsuleAuthorityTeacherExplicit {
			sourceKeys = teacherKeys
		}

		for _, sourceKey := range sourceKeys {
			entry, exists := sourceByKey[sourceKey]

			if !exists ||
				!models.IsValidLessonPlanContextCapsuleSourceType(
					entry.SourceType,
				) {
				continue
			}

			excerpt := truncateLessonPlanCapsuleSource(
				entry.Excerpt,
				lessonPlanCapsuleEvidenceMaxRunes,
			)

			excerptHash := ""
			if excerpt != "" {
				excerptHash =
					hashLessonPlanContextCapsuleText(
						excerpt,
					)
			}

			authority := item.Authority

			if !models.IsValidLessonPlanContextCapsuleAuthority(
				authority,
			) {
				authority = entry.Authority
			}

			evidence = append(
				evidence,
				models.LessonPlanContextCapsuleEvidence{
					LessonPlanID: lessonPlanID,
					ItemKey:      item.Key,
					SourceType:   entry.SourceType,
					SourceID:     entry.SourceID,
					SourceTitle:  entry.Title,
					Locator:      entry.Locator,
					SourceHash:   entry.SourceHash,
					ExcerptHash:  excerptHash,
					EvidenceExcerpt:
						excerpt,
					Authority: authority,
				},
			)
		}
	}

	return evidence
}

// collectLessonPlanContextCapsuleItems 汇总全部原子条目用于证据绑定。
func collectLessonPlanContextCapsuleItems(
	document *models.LessonPlanContextCapsuleDocument,
) []models.LessonPlanContextCapsuleItem {
	if document == nil {
		return nil
	}

	output := make(
		[]models.LessonPlanContextCapsuleItem,
		0,
	)

	output = append(
		output,
		document.CourseCore...,
	)

	output = append(
		output,
		document.TeachingConsensus...,
	)

	output = append(
		output,
		document.Constraints...,
	)

	output = append(
		output,
		document.OpenQuestions...,
	)

	output = append(
		output,
		document.DeferredItems...,
	)

	output = append(
		output,
		document.SupersededItems...,
	)

	return output
}
