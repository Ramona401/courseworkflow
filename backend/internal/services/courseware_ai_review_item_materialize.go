package services

// courseware_ai_review_item_materialize.go
//
// 将已完成的课件AI最终报告物化为可持续处理的整改项。
//
// 主要职责：
//   1. 只允许AI会话创建者选择本会话中的finding；
//   2. 使用会话PageIndexJSON把AI页码确定性映射为稳定page_id；
//   3. 一条跨多页finding拆成多个页级整改项；
//   4. 保存审核时页码、标题和HTML哈希快照；
//   5. 页面已经变化时直接生成stale整改项；
//   6. 页面已经删除时生成orphaned整改项；
//   7. 自审整改项仅作者可见；
//   8. 正式整改项在绑定人工审核反馈前仅审核员可见；
//   9. 物化前再次把dimension收敛到会话已选择的R-02维度；
//   10. 证据快照保存配置哈希、教案模式、维度事实和原始技术证据；
//   11. 同时固化teacher_view_snapshot，后续历史打开不临时调用AI改写；
//   12. 整改项兼容标题、说明和建议字段使用教师语言，原始事实保留在内部证据中。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrCWReviewItemSessionNotDone = errors.New(
		"课件AI审核尚未生成最终报告",
	)
	ErrCWReviewItemFindingRequired = errors.New(
		"请至少选择一条AI审核发现",
	)
	ErrCWReviewItemFindingNotFound = errors.New(
		"选择的AI审核发现不存在或已变化",
	)
	ErrCWReviewItemNotDelivered = errors.New(
		"该正式审核整改项尚未随人工审核结果交付",
	)
	ErrCWReviewItemStale = errors.New(
		"整改意见对应页面已发生变化，需要重新审核或重新确认",
	)
	ErrCWReviewItemOrphaned = errors.New(
		"整改意见对应的原页面已被删除",
	)
	ErrCWReviewItemNotActionable = errors.New(
		"该整改项当前状态不可继续讨论或确认",
	)
	ErrCWReviewItemContentTooLong = errors.New(
		"单次整改讨论内容过长",
	)
	ErrCWReviewItemInstructionInvalid = errors.New(
		"确认的修改指令不能为空或内容过长",
	)
)

// MaterializeCWAIReviewFindings 把人工选择的最终finding物化为整改项。
func (s *CoursewareAIReviewService) MaterializeCWAIReviewFindings(
	ctx context.Context,
	sessionID string,
	findingIDs []string,
	actor *CoursewareActorContext,
) ([]*models.CoursewareReviewItem, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}
	if s == nil ||
		s.reviewService == nil ||
		s.coursewareService == nil {
		return nil, errors.New(
			"课件AI审核服务未初始化",
		)
	}

	session, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			strings.TrimSpace(sessionID),
		)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrCWAIReviewSessionNotFound
	}
	if session.ReviewerID != actor.UserID {
		return nil, ErrCWAIReviewSessionOwnerMismatch
	}
	if session.Status != models.CWAIReviewStatusDone {
		return nil, ErrCWReviewItemSessionNotDone
	}

	configSnapshot, err :=
		cwAIReviewConfigFromSession(session)
	if err != nil {
		return nil, err
	}

	selectedIDs :=
		normalizeCWReviewFindingIDs(findingIDs)
	if len(selectedIDs) == 0 {
		return nil, ErrCWReviewItemFindingRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			session.CoursewareID,
		)
	if err != nil ||
		courseware == nil {
		return nil, ErrCWAIReviewCoursewareNotFound
	}

	if err := s.authorizeCWReviewItemMaterialization(
		ctx,
		session,
		courseware,
		actor,
	); err != nil {
		return nil, err
	}

	var report models.CWAIReviewFinalReport
	if err := json.Unmarshal(
		[]byte(session.FinalReportJSON),
		&report,
	); err != nil {
		return nil, fmt.Errorf(
			"解析课件AI最终报告失败: %w",
			err,
		)
	}

	reportConfigHash := strings.TrimSpace(
		report.ReviewConfig.ReviewConfigHash,
	)
	if reportConfigHash != "" &&
		reportConfigHash != session.ReviewConfigHash {
		return nil, errors.New(
			"课件AI最终报告配置与会话快照不一致",
		)
	}

	findingsByID := make(
		map[string]models.CWAIReviewFinding,
		len(report.Findings),
	)
	for _, rawFinding := range report.Findings {
		finding := rawFinding

		if err := normalizeCWAIReviewFindingForSession(
			session,
			&finding,
		); err != nil {
			return nil, err
		}

		id := strings.TrimSpace(finding.ID)
		if id == "" {
			continue
		}

		findingsByID[id] = finding
	}

	var pageDigests []models.CWAIReviewPageDigest
	if err := json.Unmarshal(
		[]byte(session.PageIndexJSON),
		&pageDigests,
	); err != nil {
		return nil, fmt.Errorf(
			"解析课件AI页面快照索引失败: %w",
			err,
		)
	}

	digestsByNumber := make(
		map[int]models.CWAIReviewPageDigest,
		len(pageDigests),
	)
	for _, digest := range pageDigests {
		if digest.PageNumber <= 0 {
			continue
		}

		digestsByNumber[digest.PageNumber] =
			digest
	}

	existing, err :=
		repository.ListCoursewareReviewItemsBySessionForCreator(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	existingByKey := make(
		map[string]*models.CoursewareReviewItem,
		len(existing),
	)
	for _, item := range existing {
		if item == nil {
			continue
		}

		existingByKey[cwReviewMaterializedItemKey(
			item.SourceFindingID,
			item.PageID,
		)] = item
	}

	result := make(
		[]*models.CoursewareReviewItem,
		0,
		len(selectedIDs),
	)

	for _, findingID := range selectedIDs {
		finding, exists :=
			findingsByID[findingID]
		if !exists {
			return nil,
				fmt.Errorf(
					"%w：%s",
					ErrCWReviewItemFindingNotFound,
					findingID,
				)
		}

		pageNumbers :=
			normalizeCWAIReviewPageNumbers(
				finding.PageNumbers,
			)

		if len(pageNumbers) == 0 {
			item, buildErr :=
				s.materializeCWReviewFindingPage(
					ctx,
					session,
					courseware,
					actor,
					configSnapshot,
					finding,
					finding.ID,
					nil,
					0,
				)
			if buildErr != nil {
				return nil, buildErr
			}

			key := cwReviewMaterializedItemKey(
				item.SourceFindingID,
				item.PageID,
			)
			if saved, ok := existingByKey[key]; ok {
				result = append(result, saved)
				continue
			}

			if err := repository.CreateCoursewareReviewItem(
				ctx,
				item,
			); err != nil {
				return nil, err
			}
			existingByKey[key] = item
			result = append(result, item)
			continue
		}

		for _, pageNumber := range pageNumbers {
			digest, digestExists :=
				digestsByNumber[pageNumber]

			storageFindingID :=
				strings.TrimSpace(finding.ID)

			var digestPointer *models.CWAIReviewPageDigest
			if digestExists &&
				strings.TrimSpace(digest.PageID) != "" {
				copyDigest := digest
				digestPointer = &copyDigest
			} else {
				// 页面索引缺失时使用带页码的存储ID，
				// 避免同一finding多个缺失页面发生唯一索引冲突。
				storageFindingID =
					fmt.Sprintf(
						"%s@p%d",
						finding.ID,
						pageNumber,
					)
			}

			item, buildErr :=
				s.materializeCWReviewFindingPage(
					ctx,
					session,
					courseware,
					actor,
					configSnapshot,
					finding,
					storageFindingID,
					digestPointer,
					pageNumber,
				)
			if buildErr != nil {
				return nil, buildErr
			}

			key := cwReviewMaterializedItemKey(
				item.SourceFindingID,
				item.PageID,
			)
			if saved, ok := existingByKey[key]; ok {
				result = append(result, saved)
				continue
			}

			if err := repository.CreateCoursewareReviewItem(
				ctx,
				item,
			); err != nil {
				return nil, err
			}
			existingByKey[key] = item
			result = append(result, item)
		}
	}

	return result, nil
}

// ListCWAIReviewSessionItems 返回会话创建者已经物化的整改项。
func (s *CoursewareAIReviewService) ListCWAIReviewSessionItems(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) ([]*models.CoursewareReviewItem, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}

	session, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			strings.TrimSpace(sessionID),
		)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrCWAIReviewSessionNotFound
	}
	if session.ReviewerID != actor.UserID {
		return nil, ErrCWAIReviewSessionOwnerMismatch
	}

	// 正式AI审核完成后，自动把严重、高风险和中风险的明确页级发现
	// 建立为detected整改草稿。创建过程幂等，刷新可安全重试。
	if err := s.ensureAutoMaterializedFormalReviewFindings(
		ctx,
		session,
		actor,
	); err != nil {
		return nil, err
	}

	return repository.
		ListCoursewareReviewItemsBySessionForCreator(
			ctx,
			session.ID,
			actor.UserID,
		)
}

// ListCWOwnerReviewItems 返回作者可见的课件整改项。
//
// 正式整改项只有绑定正式反馈后才向作者开放。
func (s *CoursewareAIReviewService) ListCWOwnerReviewItems(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
) ([]*models.CoursewareReviewItem, error) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			strings.TrimSpace(coursewareID),
		)
	if err != nil ||
		courseware == nil {
		return nil, ErrCWAIReviewCoursewareNotFound
	}

	if err := ValidateCoursewareReviewEducationDomain(
		actor,
		courseware,
	); err != nil {
		return nil, err
	}
	if courseware.UserID != actor.UserID {
		return nil, ErrCWAIReviewNoPermission
	}

	items, err :=
		repository.ListCoursewareReviewItemsForOwner(
			ctx,
			courseware.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	visible := make(
		[]*models.CoursewareReviewItem,
		0,
		len(items),
	)
	for _, item := range items {
		if item == nil {
			continue
		}

		if item.SourceType ==
			models.CWReviewItemSourceSelf {
			visible = append(visible, item)
			continue
		}

		if item.FeedbackID != nil {
			visible = append(visible, item)
		}
	}

	return visible, nil
}

// ensureAutoMaterializedFormalReviewFindings 自动建立正式审核的重点页级整改草稿。
func (s *CoursewareAIReviewService) ensureAutoMaterializedFormalReviewFindings(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	actor *CoursewareActorContext,
) error {
	if session == nil ||
		session.Status != models.CWAIReviewStatusDone ||
		isCWAIReviewSelfReview(session) {
		return nil
	}

	var report models.CWAIReviewFinalReport
	if err := json.Unmarshal(
		[]byte(session.FinalReportJSON),
		&report,
	); err != nil {
		return fmt.Errorf(
			"解析自动页级整改草稿报告失败: %w",
			err,
		)
	}

	findingIDs := make(
		[]string,
		0,
		len(report.Findings),
	)

	for _, rawFinding := range report.Findings {
		finding := rawFinding

		if err := normalizeCWAIReviewFindingForSession(
			session,
			&finding,
		); err != nil {
			return err
		}

		findingID := strings.TrimSpace(finding.ID)
		if findingID == "" {
			continue
		}

		pageNumbers :=
			normalizeCWAIReviewPageNumbers(
				finding.PageNumbers,
			)
		if len(pageNumbers) == 0 {
			continue
		}

		severity :=
			normalizeCWAIReviewSeverity(
				finding.Severity,
			)

		switch severity {
		case "critical",
			"high",
			"medium":
			findingIDs = append(
				findingIDs,
				findingID,
			)
		}
	}

	findingIDs =
		normalizeCWReviewFindingIDs(
			findingIDs,
		)
	if len(findingIDs) == 0 {
		return nil
	}

	_, err :=
		s.MaterializeCWAIReviewFindings(
			ctx,
			session.ID,
			findingIDs,
			actor,
		)

	return err
}

func (s *CoursewareAIReviewService) authorizeCWReviewItemMaterialization(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) error {
	if session.ReviewLevel ==
		models.CWAIReviewLevelSelf {
		if courseware.UserID != actor.UserID {
			return ErrCWAIReviewNoPermission
		}

		return ValidateCoursewareReviewEducationDomain(
			actor,
			courseware,
		)
	}

	allowed, err :=
		s.reviewService.CanReviewLoadedCourseware(
			ctx,
			courseware,
			actor,
		)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrCWAIReviewNoPermission
	}

	return nil
}

func (s *CoursewareAIReviewService) materializeCWReviewFindingPage(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	config *CWAIReviewConfigSnapshot,
	finding models.CWAIReviewFinding,
	storageFindingID string,
	digest *models.CWAIReviewPageDigest,
	fallbackPageNumber int,
) (*models.CoursewareReviewItem, error) {
	if config == nil {
		return nil, ErrCWAIReviewConfigInvalid
	}

	if err := normalizeCWAIReviewFindingForSession(
		session,
		&finding,
	); err != nil {
		return nil, err
	}

	sourceType :=
		models.CWReviewItemSourceFormal
	if session.ReviewLevel ==
		models.CWAIReviewLevelSelf {
		sourceType =
			models.CWReviewItemSourceSelf
	}

	status :=
		models.CWReviewItemStatusDetected

	var pageID *string
	pageNumber := fallbackPageNumber
	pageTitle := ""
	pageHTMLHash := ""

	var pageUpdatedAt = (*repository.CoursewareReviewPageSnapshot)(nil)

	if digest != nil {
		pageNumber = digest.PageNumber
		pageTitle = strings.TrimSpace(digest.Title)
		pageHTMLHash = strings.TrimSpace(digest.HTMLHash)

		currentPage, err :=
			repository.GetCoursewareReviewPageSnapshotByID(
				ctx,
				digest.PageID,
				courseware.ID,
			)
		if err != nil {
			if errors.Is(
				err,
				repository.ErrCoursewareReviewPageSnapshotNotFound,
			) {
				status =
					models.CWReviewItemStatusOrphaned
			} else {
				return nil, err
			}
		} else {
			pageUpdatedAt = currentPage
			currentPageID := currentPage.ID
			pageID = &currentPageID

			if pageHTMLHash != "" &&
				cwAIReviewHash(
					currentPage.HTMLContent,
				) != pageHTMLHash {
				status =
					models.CWReviewItemStatusStale
			}
		}
	} else if pageNumber > 0 {
		status =
			models.CWReviewItemStatusOrphaned
	}

	teacherView := finding.TeacherViewSnapshot
	teacherView.AcceptanceChecks = append(
		[]string{},
		teacherView.AcceptanceChecks...,
	)

	evidence := map[string]interface{}{
		"finding_id":       finding.ID,
		"all_page_numbers": finding.PageNumbers,
		"dimension":        finding.Dimension,

		"raw_title":       finding.Title,
		"raw_description": finding.Description,
		"raw_suggestion":  finding.InternalExecutionPlan,

		"lesson_or_outline_basis": finding.LessonOrOutlineBasis,
		"page_evidence":           finding.PageEvidence,
		"code_evidence":           finding.CodeEvidence,
		"continuity_evidence":     finding.ContinuityEvidence,

		"internal_execution_plan": finding.InternalExecutionPlan,

		"teacher_view_schema_version": 1,
		"teacher_view_snapshot":       teacherView,

		"confidence":             finding.Confidence,
		"manual_review_required": finding.ManualReviewRequired,

		"review_config_hash": session.ReviewConfigHash,
		"review_dimensions": append(
			[]string{},
			config.ReviewDimensions...,
		),
		"lesson_reference_mode": config.LessonReferenceMode,
	}

	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化课件整改项证据失败: %w",
			err,
		)
	}

	item := &models.CoursewareReviewItem{
		CoursewareID: courseware.ID,

		SourceSessionID: session.ID,
		SourceFindingID: strings.TrimSpace(
			storageFindingID,
		),

		SourceType:  sourceType,
		ReviewLevel: session.ReviewLevel,
		ReviewRound: 0,

		CreatedBy: actor.UserID,
		OwnerID:   courseware.UserID,

		PageID:             pageID,
		PageNumberSnapshot: pageNumber,
		PageTitleSnapshot:  pageTitle,
		PageHTMLHash:       pageHTMLHash,

		Severity:  normalizeCWAIReviewSeverity(finding.Severity),
		Dimension: finding.Dimension,

		Title:       teacherView.TeacherTitle,
		Description: teacherView.WhatHappened,

		EvidenceJSON: string(evidenceJSON),

		OriginalSuggestion: teacherView.ImprovementGoal,

		Status: status,
	}

	if pageUpdatedAt != nil {
		item.PageUpdatedAtSnapshot =
			pageUpdatedAt.UpdatedAt
	}

	return item, nil
}

func normalizeCWReviewFindingIDs(
	input []string,
) []string {
	result := make([]string, 0, len(input))
	seen := make(map[string]bool)

	for _, raw := range input {
		value := strings.TrimSpace(raw)
		if value == "" || seen[value] {
			continue
		}

		seen[value] = true
		result = append(result, value)
	}

	return result
}

func cwReviewMaterializedItemKey(
	sourceFindingID string,
	pageID *string,
) string {
	pageKey := "global"
	if pageID != nil &&
		strings.TrimSpace(*pageID) != "" {
		pageKey = strings.TrimSpace(*pageID)
	}

	return strings.TrimSpace(sourceFindingID) +
		"|" +
		pageKey +
		"|" +
		strconv.FormatBool(pageID != nil)
}
