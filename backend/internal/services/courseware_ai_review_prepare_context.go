package services

// courseware_ai_review_prepare_context.go
//
// 课件AI审核准备阶段的材料读取、上下文清单和快照构建。
//
// 本文件负责：
//   - 按页码生成稳定页面摘要；
//   - 根据R-02教案参考模式决定是否读取教案类材料；
//   - no_lesson模式真实跳过教案、大纲和对齐报告查询；
//   - 构造会话context_manifest_json和baseline_json；
//   - 构造课件、页面、教案和大纲确定性快照哈希；
//   - 生成批次规划所需的页面索引和初始账本。
//
// 浏览器不能提交本文件处理的任何正文、页面HTML、材料可用状态或哈希。

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwAIReviewPreparedMaterials 是准备阶段从可信数据库读取的教案类材料。
type cwAIReviewPreparedMaterials struct {
	usesLessonMaterials bool

	lessonPlan    *models.LessonPlan
	lessonContent string

	outlineContext string
	outlineTitles  []string

	alignmentReport  *models.CoursewareAlignmentReport
	alignmentJSON    string
	alignmentSummary string
	alignmentStatus  string

	// no_lesson仍保留课件固化的来源教案ID供内部关联和调用追踪，
	// 但不会读取或保存教案标题和正文。
	lessonPlanID *string
}

// cwAIReviewPreparedSnapshot 是写入会话前构建的全部确定性快照。
type cwAIReviewPreparedSnapshot struct {
	contextManifestJSON string
	baselineJSON        string
	pageIndexJSON       string
	ledgerJSON          string

	coursewareSnapshotHash string
	pagesSnapshotHash      string
	lessonSnapshotHash     string
	outlineSnapshotHash    string

	batches []*models.CoursewareAIReviewBatch
}

// buildCWAIReviewPreparationPageDigests 过滤空页面、按页码排序并构造摘要。
func buildCWAIReviewPreparationPageDigests(
	pages []*models.CoursewarePage,
) []models.CWAIReviewPageDigest {
	orderedPages := make(
		[]*models.CoursewarePage,
		0,
		len(pages),
	)

	for _, page := range pages {
		if page != nil {
			orderedPages = append(orderedPages, page)
		}
	}

	sort.SliceStable(
		orderedPages,
		func(i int, j int) bool {
			return orderedPages[i].PageNumber <
				orderedPages[j].PageNumber
		},
	)

	digests := make(
		[]models.CWAIReviewPageDigest,
		0,
		len(orderedPages),
	)

	for _, page := range orderedPages {
		digests = append(
			digests,
			BuildCWAIReviewPageDigest(page),
		)
	}

	return digests
}

// loadCWAIReviewPreparedMaterials 根据不可变配置读取可信材料。
//
// no_lesson模式会直接返回，不执行以下数据库调用：
//   - GetLessonPlanByID；
//   - BuildLessonPlanCourseOutlineContext；
//   - GetAlignmentReportByCoursewareID。
func loadCWAIReviewPreparedMaterials(
	ctx context.Context,
	courseware *models.Courseware,
	config *CWAIReviewConfigSnapshot,
) (*cwAIReviewPreparedMaterials, error) {
	if courseware == nil {
		return nil, ErrCWAIReviewCoursewareNotFound
	}

	materials := &cwAIReviewPreparedMaterials{
		usesLessonMaterials: cwAIReviewUsesLessonMaterials(config),
		outlineTitles:       []string{},
		alignmentJSON:       "{}",
		lessonPlanID:        courseware.LessonPlanID,
	}

	if !materials.usesLessonMaterials {
		return materials, nil
	}

	lessonPlan, lessonContent, outlineContext, outlineTitles, err :=
		loadCWAIReviewLessonAndOutline(
			ctx,
			courseware,
		)
	if err != nil {
		return nil, err
	}

	materials.lessonPlan = lessonPlan
	materials.lessonContent = lessonContent
	materials.outlineContext = outlineContext
	materials.outlineTitles = outlineTitles

	if lessonPlan != nil {
		lessonPlanID := lessonPlan.ID
		materials.lessonPlanID = &lessonPlanID
	}

	alignmentReport, err :=
		repository.GetAlignmentReportByCoursewareID(
			ctx,
			courseware.ID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"读取课件与教案对齐报告失败: %w",
			err,
		)
	}

	materials.alignmentReport = alignmentReport

	if alignmentReport != nil {
		materials.alignmentJSON = cwAIReviewValidJSONOrString(
			alignmentReport.ReportJSON,
		)
		materials.alignmentSummary = strings.TrimSpace(
			alignmentReport.Summary,
		)
		materials.alignmentStatus = strings.TrimSpace(
			alignmentReport.Status,
		)
	}

	return materials, nil
}

// buildCWAIReviewPreparedSnapshot 构造会话准备结果。
func buildCWAIReviewPreparedSnapshot(
	courseware *models.Courseware,
	detail *models.CoursewareDetailResponse,
	pageDigests []models.CWAIReviewPageDigest,
	materials *cwAIReviewPreparedMaterials,
	config *CWAIReviewConfigSnapshot,
	selectedAssistantID *string,
	assistantID string,
	reviewLevel int,
) (*cwAIReviewPreparedSnapshot, error) {
	if courseware == nil || detail == nil || materials == nil || config == nil {
		return nil, errorsNewCWAIReviewPreparation(
			"缺少课件AI审核准备上下文",
		)
	}
	if len(pageDigests) == 0 {
		return nil, ErrCWAIReviewNoPages
	}

	usesLessonMaterials := materials.usesLessonMaterials
	lessonAvailable := usesLessonMaterials &&
		materials.lessonPlan != nil
	outlineAvailable := usesLessonMaterials &&
		len(materials.outlineTitles) > 0
	alignmentAvailable := usesLessonMaterials &&
		materials.alignmentReport != nil

	lessonContentIncluded := usesLessonMaterials &&
		strings.TrimSpace(materials.lessonContent) != ""
	courseOutlineIncluded := usesLessonMaterials &&
		strings.TrimSpace(materials.outlineContext) != ""
	alignmentReportIncluded := alignmentAvailable

	configManifest := cwAIReviewConfigManifest(config)

	contextManifest := map[string]interface{}{
		"courseware": map[string]interface{}{
			"id":               courseware.ID,
			"title":            courseware.Title,
			"subject":          courseware.Subject,
			"grade":            courseware.Grade,
			"education_domain": courseware.EducationDomain,
			"source_type":      courseware.SourceType,
			"page_count":       len(pageDigests),
			"kp_codes":         cwAIReviewJSONValue(courseware.KPCodes),
		},
		"review_config": configManifest,
		"lesson_material_usage": map[string]interface{}{
			"lesson_content_included":   lessonContentIncluded,
			"course_outline_included":   courseOutlineIncluded,
			"alignment_report_included": alignmentReportIncluded,
		},
		"lesson_plan": map[string]interface{}{
			"available": lessonAvailable,
			"used":      usesLessonMaterials,
			"id": func() string {
				if !lessonAvailable {
					return ""
				}
				return cwAIReviewLessonID(materials.lessonPlan)
			}(),
			"title": func() string {
				if !lessonAvailable {
					return ""
				}
				return cwAIReviewLessonTitle(materials.lessonPlan)
			}(),
		},
		"course_outline": map[string]interface{}{
			"available": outlineAvailable,
			"used":      usesLessonMaterials,
			"count":     len(materials.outlineTitles),
			"titles":    materials.outlineTitles,
		},
		"alignment_report": map[string]interface{}{
			"available": alignmentAvailable,
			"used":      usesLessonMaterials,
			"status":    materials.alignmentStatus,
			"summary":   materials.alignmentSummary,
		},
		"assistant": map[string]interface{}{
			"selected":     selectedAssistantID != nil,
			"assistant_id": strings.TrimSpace(assistantID),
		},
		"analysis_purpose": func() string {
			if reviewLevel == models.CWAIReviewLevelSelf {
				return "courseware_self_review"
			}
			return "formal_courseware_review"
		}(),
		"batch_policy": map[string]interface{}{
			"target_pages":       cwAIReviewBatchTargetPages,
			"minimum_pages":      cwAIReviewBatchMinPages,
			"maximum_pages":      cwAIReviewBatchMaxPages,
			"overlap_pages":      1,
			"execution_order":    "sequential",
			"continuity_carried": true,
		},
	}

	baseline := map[string]interface{}{
		"review_config": configManifest,
		"courseware": map[string]interface{}{
			"title":            courseware.Title,
			"subject":          courseware.Subject,
			"grade":            courseware.Grade,
			"education_domain": courseware.EducationDomain,
			"source_type":      courseware.SourceType,
			"index_overview":   detail.IndexOverview,
			"kp_codes":         cwAIReviewJSONValue(courseware.KPCodes),
		},
		"lesson_plan":      buildCWAIReviewLessonBaseline(materials),
		"course_outline":   buildCWAIReviewOutlineBaseline(materials),
		"alignment_report": buildCWAIReviewAlignmentBaseline(materials),
	}

	initialLedger := cwAIReviewInitialContinuityLedger()

	contextManifestJSON, err := json.Marshal(contextManifest)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化审核上下文清单失败: %w",
			err,
		)
	}

	baselineJSON, err := json.Marshal(baseline)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化审核基准失败: %w",
			err,
		)
	}

	pageIndexJSON, err := json.Marshal(pageDigests)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化课件页面索引失败: %w",
			err,
		)
	}

	ledgerJSON, err := json.Marshal(initialLedger)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化连续性账本失败: %w",
			err,
		)
	}

	coursewareSnapshotJSON, err := json.Marshal(
		map[string]interface{}{
			"id":               courseware.ID,
			"title":            courseware.Title,
			"subject":          courseware.Subject,
			"grade":            courseware.Grade,
			"education_domain": courseware.EducationDomain,
			"source_type":      courseware.SourceType,
			"index_overview":   detail.IndexOverview,
			"kp_codes":         courseware.KPCodes,
			"updated_at":       courseware.UpdatedAt,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化课件审核快照失败: %w",
			err,
		)
	}

	lessonSnapshotJSON, err :=
		buildCWAIReviewLessonSnapshot(materials, config)
	if err != nil {
		return nil, err
	}

	outlineSnapshotJSON, err :=
		buildCWAIReviewOutlineSnapshot(materials, config)
	if err != nil {
		return nil, err
	}

	batches, err := buildCWAIReviewBatches(
		pageDigests,
		string(ledgerJSON),
	)
	if err != nil {
		return nil, err
	}
	if len(batches) == 0 {
		return nil, ErrCWAIReviewNoPages
	}

	return &cwAIReviewPreparedSnapshot{
		contextManifestJSON: string(contextManifestJSON),
		baselineJSON:        string(baselineJSON),
		pageIndexJSON:       string(pageIndexJSON),
		ledgerJSON:          string(ledgerJSON),

		coursewareSnapshotHash: cwAIReviewHash(
			string(coursewareSnapshotJSON),
		),
		pagesSnapshotHash: cwAIReviewHash(
			string(pageIndexJSON),
		),
		lessonSnapshotHash: cwAIReviewHash(
			string(lessonSnapshotJSON),
		),
		outlineSnapshotHash: cwAIReviewHash(
			string(outlineSnapshotJSON),
		),

		batches: batches,
	}, nil
}

func buildCWAIReviewLessonBaseline(
	materials *cwAIReviewPreparedMaterials,
) map[string]interface{} {
	if materials == nil || !materials.usesLessonMaterials {
		return map[string]interface{}{
			"available": false,
			"used":      false,
		}
	}

	return map[string]interface{}{
		"available": materials.lessonPlan != nil,
		"used":      true,
		"id":        cwAIReviewLessonID(materials.lessonPlan),
		"title":     cwAIReviewLessonTitle(materials.lessonPlan),
		"content": cwAIReviewTruncateContext(
			materials.lessonContent,
			cwAIReviewLessonMaxRunes,
		),
	}
}

func buildCWAIReviewOutlineBaseline(
	materials *cwAIReviewPreparedMaterials,
) map[string]interface{} {
	if materials == nil || !materials.usesLessonMaterials {
		return map[string]interface{}{
			"available": false,
			"used":      false,
			"titles":    []string{},
		}
	}

	return map[string]interface{}{
		"available": len(materials.outlineTitles) > 0,
		"used":      true,
		"titles":    materials.outlineTitles,
		"context": cwAIReviewTruncateContext(
			materials.outlineContext,
			cwAIReviewOutlineMaxRunes,
		),
	}
}

func buildCWAIReviewAlignmentBaseline(
	materials *cwAIReviewPreparedMaterials,
) map[string]interface{} {
	if materials == nil || !materials.usesLessonMaterials {
		return map[string]interface{}{
			"available": false,
			"used":      false,
			"status":    "",
			"summary":   "",
		}
	}

	return map[string]interface{}{
		"available": materials.alignmentReport != nil,
		"used":      true,
		"status":    materials.alignmentStatus,
		"summary":   materials.alignmentSummary,
		"report":    json.RawMessage(materials.alignmentJSON),
	}
}

func buildCWAIReviewLessonSnapshot(
	materials *cwAIReviewPreparedMaterials,
	config *CWAIReviewConfigSnapshot,
) ([]byte, error) {
	var snapshot map[string]interface{}

	if materials != nil && materials.usesLessonMaterials {
		snapshot = map[string]interface{}{
			"id":        cwAIReviewLessonID(materials.lessonPlan),
			"title":     cwAIReviewLessonTitle(materials.lessonPlan),
			"content":   materials.lessonContent,
			"available": materials.lessonPlan != nil,
			"used":      true,
		}
	} else {
		mode := ""
		if config != nil {
			mode = config.LessonReferenceMode
		}
		snapshot = map[string]interface{}{
			"available": false,
			"used":      false,
			"mode":      mode,
		}
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化教案审核快照失败: %w",
			err,
		)
	}

	return encoded, nil
}

func buildCWAIReviewOutlineSnapshot(
	materials *cwAIReviewPreparedMaterials,
	config *CWAIReviewConfigSnapshot,
) ([]byte, error) {
	var snapshot map[string]interface{}

	if materials != nil && materials.usesLessonMaterials {
		snapshot = map[string]interface{}{
			"available": len(materials.outlineTitles) > 0,
			"used":      true,
			"titles":    materials.outlineTitles,
			"context":   materials.outlineContext,
		}
	} else {
		mode := ""
		if config != nil {
			mode = config.LessonReferenceMode
		}
		snapshot = map[string]interface{}{
			"available": false,
			"used":      false,
			"mode":      mode,
		}
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化课程大纲审核快照失败: %w",
			err,
		)
	}

	return encoded, nil
}

// loadCWAIReviewLessonAndOutline 读取来源教案和其课程大纲。
//
// 本函数只允许由材料模式已经确认可读取的调用链使用。
// no_lesson不得调用本函数。
func loadCWAIReviewLessonAndOutline(
	ctx context.Context,
	courseware *models.Courseware,
) (
	*models.LessonPlan,
	string,
	string,
	[]string,
	error,
) {
	if courseware == nil ||
		courseware.SourceType != models.CWSourceLessonPlan {
		return nil, "", "", []string{}, nil
	}

	if courseware.LessonPlanID == nil ||
		strings.TrimSpace(*courseware.LessonPlanID) == "" {
		return nil, "", "", nil,
			ErrCWAIReviewLessonPlanMissing
	}

	lessonPlan, err := repository.GetLessonPlanByID(
		ctx,
		strings.TrimSpace(*courseware.LessonPlanID),
	)
	if err != nil || lessonPlan == nil {
		return nil, "", "", nil,
			ErrCWAIReviewLessonPlanMissing
	}

	coursewareDomain := strings.ToLower(
		strings.TrimSpace(courseware.EducationDomain),
	)
	lessonDomain := strings.ToLower(
		strings.TrimSpace(lessonPlan.EducationDomain),
	)

	if coursewareDomain == "" ||
		lessonDomain == "" ||
		coursewareDomain != lessonDomain {
		return nil, "", "", nil,
			ErrCWAIReviewLessonDomainMismatch
	}

	lessonContent := strings.TrimSpace(
		ExtractLessonPlanContentForCW(lessonPlan),
	)

	outlineContext, outlines, err :=
		BuildLessonPlanCourseOutlineContext(
			ctx,
			lessonPlan,
		)
	if err != nil {
		return nil, "", "", nil,
			fmt.Errorf(
				"读取来源教案课程大纲失败: %w",
				err,
			)
	}

	titles := make([]string, 0, len(outlines))

	for _, outline := range outlines {
		if outline == nil {
			continue
		}

		title := strings.TrimSpace(outline.Title)
		if title != "" {
			titles = append(titles, title)
		}
	}

	return lessonPlan,
		lessonContent,
		outlineContext,
		titles,
		nil
}

func cwAIReviewTruncateContext(
	content string,
	maxRunes int,
) string {
	content = strings.TrimSpace(content)
	if content == "" || maxRunes <= 0 {
		return ""
	}

	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}

	return string(runes[:maxRunes]) +
		"\n\n[上下文超过审核基准预算，后续分批调用时将按相关性继续抽取]"
}

func cwAIReviewLessonID(
	lessonPlan *models.LessonPlan,
) string {
	if lessonPlan == nil {
		return ""
	}

	return strings.TrimSpace(lessonPlan.ID)
}

func cwAIReviewLessonTitle(
	lessonPlan *models.LessonPlan,
) string {
	if lessonPlan == nil {
		return ""
	}

	return strings.TrimSpace(lessonPlan.Title)
}

func cwAIReviewJSONValue(
	raw string,
) interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []interface{}{}
	}

	var value interface{}
	if err := json.Unmarshal(
		[]byte(raw),
		&value,
	); err == nil {
		return value
	}

	return raw
}

func cwAIReviewValidJSONOrString(
	raw string,
) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}

	var value interface{}
	if err := json.Unmarshal(
		[]byte(raw),
		&value,
	); err != nil {
		encoded, _ := json.Marshal(
			map[string]interface{}{
				"raw": raw,
			},
		)
		return string(encoded)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}

	return string(encoded)
}

func errorsNewCWAIReviewPreparation(
	message string,
) error {
	return fmt.Errorf("%s", strings.TrimSpace(message))
}
