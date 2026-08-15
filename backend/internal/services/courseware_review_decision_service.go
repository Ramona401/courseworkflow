package services

// courseware_review_decision_service.go
//
// 课件L1/L2正式审核决定服务。
//
// 本文件负责：
//
//   1. 重新读取课件并校验教育域；
//   2. 校验L1或L2审核权限；
//   3. 重新读取后端AI最终报告；
//   4. 重新读取审核员选中的本轮新整改项；
//   5. 接收审核员明确确认解决的上一轮问题ID；
//   6. 构建不可变反馈快照；
//   7. 调用事务仓储原子提交审核记录、旧问题解决、新问题交付和课件状态；
//   8. 事务成功后记录日志并旁路通知作者。
//
// 复审规则：
//
//   - 审核通过：本级、本轮所有旧问题必须全部确认解决；
//   - 继续退回：允许只确认一部分旧问题，未确认部分继续保留；
//   - 旧问题解决说明使用本轮人工审核意见，不接受前端单独伪造；
//   - 前端不能提交AI报告正文或旧问题状态。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwFormalReviewFeedbackSnapshot 是正式审核事务使用的AI反馈快照。
type cwFormalReviewFeedbackSnapshot struct {
	AIReviewSessionID *string
	ReviewItemIDs     []string

	OverallRisk         string
	OverallSummary      string
	StrengthsJSON       string
	ObviousProblemsJSON string
}

// cwReviewNextState 是一次审核决定成功后的目标审核状态。
type cwReviewNextState struct {
	PublishState   string
	ReviewLevel    int
	ReviewSchoolID *string
	NeedL2         bool
}

// CWReviewDecisionResult 是一次正式课件审核提交成功后的可信结果。
// DeliveredItemCount 只在事务成功后返回，代表本次真正绑定到正式反馈的整改项数量。
type CWReviewDecisionResult struct {
	DeliveredItemCount int
}

// reviewCoursewareAtLevel 保留既有服务入口，只返回错误，避免影响已有调用方。
func (s *CoursewareReviewService) reviewCoursewareAtLevel(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
	reviewLevel int,
) error {
	_, err := s.reviewCoursewareAtLevelWithResult(ctx, coursewareID, actor, req, reviewLevel)
	return err
}

// ReviewL1WithResult 执行L1正式审核并返回本次真实交付数量。
func (s *CoursewareReviewService) ReviewL1WithResult(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
) (*CWReviewDecisionResult, error) {
	return s.reviewCoursewareAtLevelWithResult(ctx, coursewareID, actor, req, models.ReviewLevelL1)
}

// ReviewL2WithResult 执行L2正式审核并返回本次真实交付数量。
func (s *CoursewareReviewService) ReviewL2WithResult(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
) (*CWReviewDecisionResult, error) {
	return s.reviewCoursewareAtLevelWithResult(ctx, coursewareID, actor, req, models.ReviewLevelL2)
}

// reviewCoursewareAtLevelWithResult 执行L1或L2正式审核并返回可信交付数量。
func (s *CoursewareReviewService) reviewCoursewareAtLevelWithResult(
	ctx context.Context,
	coursewareID string,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
	reviewLevel int,
) (*CWReviewDecisionResult, error) {
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCoursewareActorRequired
	}

	courseware, err := repository.GetCoursewareByID(ctx, strings.TrimSpace(coursewareID))
	if err != nil || courseware == nil {
		return nil, ErrCWReviewCoursewareNotFound
	}

	if err := ValidateCoursewareReviewEducationDomain(actor, courseware); err != nil {
		return nil, err
	}

	if !isValidCWReviewDecision(req) {
		return nil, ErrCWReviewInvalidDecision
	}

	req.Comment = strings.TrimSpace(req.Comment)
	if req.Comment == "" {
		return nil, ErrCWReviewFeedbackInvalid
	}

	if err := s.authorizeCWReviewLevel(ctx, courseware, actor, reviewLevel); err != nil {
		return nil, err
	}

	feedback, err := buildCWFormalReviewFeedbackSnapshot(ctx, courseware, actor, req, reviewLevel)
	if err != nil {
		return nil, err
	}

	nextState, err := s.resolveCWReviewNextState(ctx, courseware, req.Decision, reviewLevel)
	if err != nil {
		return nil, err
	}

	resolvedReviewItemIDs := normalizeCWFormalReviewItemIDs(req.ResolvedReviewItemIDs)

	commitInput := &repository.CoursewareReviewDecisionCommitInput{
		CoursewareID: courseware.ID,
		ReviewerID:   actor.UserID,
		ReviewLevel:  reviewLevel,

		Decision:   req.Decision,
		Score:      req.Score,
		Comment:    req.Comment,
		Dimensions: strings.TrimSpace(req.Dimensions),

		ExpectedPublishState: models.CWPublishSubmitted,
		ExpectedReviewLevel:  courseware.ReviewLevel,

		NextPublishState:   nextState.PublishState,
		NextReviewLevel:    nextState.ReviewLevel,
		NextReviewSchoolID: nextState.ReviewSchoolID,

		AIReviewSessionID: feedback.AIReviewSessionID,
		ReviewItemIDs:     feedback.ReviewItemIDs,

		ResolvedReviewItemIDs: resolvedReviewItemIDs,

		OverallRisk:         feedback.OverallRisk,
		OverallSummary:      feedback.OverallSummary,
		StrengthsJSON:       feedback.StrengthsJSON,
		ObviousProblemsJSON: feedback.ObviousProblemsJSON,
	}

	review, _, err := repository.CommitCoursewareReviewDecision(ctx, commitInput)
	if err != nil {
		return nil, mapCWReviewCommitError(err, reviewLevel)
	}

	deliveredItemCount := len(feedback.ReviewItemIDs)
	if req.Decision != models.ReviewDecisionRevision {
		deliveredItemCount = 0
	}

	s.afterCWReviewDecisionCommitted(ctx, courseware, actor.UserID, review, nextState.NeedL2)

	return &CWReviewDecisionResult{DeliveredItemCount: deliveredItemCount}, nil
}

// isValidCWReviewDecision 校验正式审核请求与决策值。
func isValidCWReviewDecision(
	req *models.CWReviewDecisionRequest,
) bool {
	if req == nil {
		return false
	}

	return req.Decision ==
		models.ReviewDecisionApproved ||
		req.Decision ==
			models.ReviewDecisionRevision
}

// authorizeCWReviewLevel 校验当前审核层级状态和人员权限。
func (s *CoursewareReviewService) authorizeCWReviewLevel(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	reviewLevel int,
) error {
	switch reviewLevel {
	case models.ReviewLevelL1:
		if courseware.PublishState !=
			models.CWPublishSubmitted ||
			courseware.ReviewLevel != 0 {
			return ErrCWReviewNotSubmitted
		}

		return s.authorizeCWReviewL1(
			ctx,
			courseware,
			actor,
		)

	case models.ReviewLevelL2:
		if courseware.PublishState !=
			models.CWPublishSubmitted ||
			courseware.ReviewLevel !=
				models.ReviewLevelL1 {
			return ErrCWReviewNotL2Status
		}

		return s.authorizeCWReviewL2(
			ctx,
			courseware,
			actor,
		)

	default:
		return ErrCWReviewInvalidDecision
	}
}

// resolveCWReviewNextState 计算正式审核完成后的目标状态。
func (s *CoursewareReviewService) resolveCWReviewNextState(
	ctx context.Context,
	courseware *models.Courseware,
	decision string,
	reviewLevel int,
) (
	*cwReviewNextState,
	error,
) {
	if decision ==
		models.ReviewDecisionRevision {
		return &cwReviewNextState{
			PublishState: models.CWPublishRevision,
			ReviewLevel:  0,
		}, nil
	}

	if decision !=
		models.ReviewDecisionApproved {
		return nil,
			ErrCWReviewInvalidDecision
	}

	switch reviewLevel {
	case models.ReviewLevelL1:
		return s.resolveCWReviewL1ApprovedState(
			ctx,
			courseware,
		)

	case models.ReviewLevelL2:
		return &cwReviewNextState{
			PublishState:   models.CWPublishApproved,
			ReviewLevel:    models.ReviewLevelL2,
			ReviewSchoolID: courseware.ReviewSchoolID,
		}, nil

	default:
		return nil,
			ErrCWReviewInvalidDecision
	}
}

// resolveCWReviewL1ApprovedState 判断L1通过后是否进入L2。
func (s *CoursewareReviewService) resolveCWReviewL1ApprovedState(
	ctx context.Context,
	courseware *models.Courseware,
) (
	*cwReviewNextState,
	error,
) {
	schoolID :=
		s.resolveReviewSchoolID(
			ctx,
			courseware,
		)

	state :=
		&cwReviewNextState{
			PublishState: models.CWPublishApproved,
			ReviewLevel:  models.ReviewLevelL1,
		}

	if schoolID == "" {
		return state, nil
	}

	schoolIDCopy := schoolID
	state.ReviewSchoolID =
		&schoolIDCopy

	config, err :=
		repository.GetReviewFlowConfig(
			ctx,
			schoolID,
		)
	if err == nil &&
		config != nil &&
		config.L2Enabled {
		state.PublishState =
			models.CWPublishSubmitted
		state.NeedL2 = true
	}

	return state, nil
}

// authorizeCWReviewL1 校验L1审核权限。
func (s *CoursewareReviewService) authorizeCWReviewL1(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) error {
	allowed :=
		actor.Role ==
			models.RoleAdmin

	if !allowed &&
		actor.Role ==
			models.RoleSeniorOperator {
		allowed =
			s.isSeniorOfReviewSchool(
				ctx,
				courseware,
				actor.UserID,
			)
	}

	if !allowed {
		hasPermission, err :=
			s.isReviewerInAuthorGroupAsLeadOrBackbone(
				ctx,
				courseware.UserID,
				actor.UserID,
			)
		if err != nil {
			return fmt.Errorf(
				"校验审核权限失败: %w",
				err,
			)
		}

		allowed = hasPermission
	}

	if !allowed {
		return ErrCWReviewNoPermission
	}

	return nil
}

// authorizeCWReviewL2 校验L2审核权限。
func (s *CoursewareReviewService) authorizeCWReviewL2(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
) error {
	if actor.Role !=
		models.RoleSeniorOperator &&
		actor.Role !=
			models.RoleAdmin {
		return ErrCWReviewNoPermission
	}

	if actor.Role ==
		models.RoleAdmin {
		return nil
	}

	school, err :=
		repository.GetSchoolByAdminUserID(
			ctx,
			actor.UserID,
		)
	if err != nil ||
		school == nil {
		return ErrCWReviewNoPermission
	}

	if courseware.ReviewSchoolID == nil ||
		*courseware.ReviewSchoolID !=
			school.ID {
		return ErrCWReviewNoPermission
	}

	return nil
}

// buildCWFormalReviewFeedbackSnapshot 重新读取AI报告和选中的本轮新整改项。
func buildCWFormalReviewFeedbackSnapshot(
	ctx context.Context,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	req *models.CWReviewDecisionRequest,
	reviewLevel int,
) (
	*cwFormalReviewFeedbackSnapshot,
	error,
) {
	result :=
		&cwFormalReviewFeedbackSnapshot{
			OverallRisk: models.CWReviewSeverityInfo,
			OverallSummary: strings.TrimSpace(
				req.Comment,
			),
			StrengthsJSON:       "[]",
			ObviousProblemsJSON: "[]",
		}

	sessionID :=
		strings.TrimSpace(
			req.AIReviewSessionID,
		)

	itemIDs :=
		normalizeCWFormalReviewItemIDs(
			req.ReviewItemIDs,
		)

	if sessionID == "" {
		if len(itemIDs) > 0 {
			return nil,
				ErrCWReviewFeedbackInvalid
		}

		return result, nil
	}

	// 通过决定不交付本轮新整改项，避免作者收到与通过结果冲突的新任务。
	//
	// 上一轮旧问题通过ResolvedReviewItemIDs单独确认解决，
	// 不受本规则影响。
	if req.Decision ==
		models.ReviewDecisionApproved &&
		len(itemIDs) > 0 {
		return nil,
			ErrCWReviewFeedbackInvalid
	}

	session, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			sessionID,
		)
	if err != nil {
		return nil, err
	}

	if !isMatchingCWFormalReviewSession(
		session,
		courseware,
		actor,
		reviewLevel,
	) {
		return nil,
			ErrCWReviewFeedbackInvalid
	}

	var report models.CWAIReviewFinalReport
	if err := json.Unmarshal(
		[]byte(
			session.FinalReportJSON,
		),
		&report,
	); err != nil {
		return nil, fmt.Errorf(
			"解析正式课件AI审核报告失败: %w",
			err,
		)
	}

	sessionIDCopy :=
		session.ID
	result.AIReviewSessionID =
		&sessionIDCopy
	result.OverallRisk =
		normalizeCWAIReviewSeverity(
			report.OverallRisk,
		)

	if strings.TrimSpace(
		report.Summary,
	) != "" {
		result.OverallSummary =
			strings.TrimSpace(
				report.Summary,
			)
	}

	strengthsJSON, err :=
		json.Marshal(
			report.Strengths,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化课件审核优点失败: %w",
			err,
		)
	}

	result.StrengthsJSON =
		string(strengthsJSON)

	if len(itemIDs) == 0 {
		return result, nil
	}

	items, err :=
		repository.ListCoursewareReviewItemsBySessionForCreator(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	itemsByID :=
		make(
			map[string]*models.CoursewareReviewItem,
			len(items),
		)

	for _, item := range items {
		if item != nil {
			itemsByID[item.ID] =
				item
		}
	}

	problems :=
		make(
			[]map[string]interface{},
			0,
			len(itemIDs),
		)

	for _, itemID := range itemIDs {
		item, exists :=
			itemsByID[itemID]

		if !exists ||
			!isSelectableCWFormalReviewItem(
				item,
				courseware,
				session,
				reviewLevel,
			) {
			return nil,
				ErrCWReviewFeedbackInvalid
		}

		problems =
			append(
				problems,
				map[string]interface{}{
					"review_item_id":       item.ID,
					"source_finding_id":    item.SourceFindingID,
					"page_id":              item.PageID,
					"page_number_snapshot": item.PageNumberSnapshot,
					"page_title_snapshot":  item.PageTitleSnapshot,
					"page_html_hash":       item.PageHTMLHash,
					"severity":             item.Severity,
					"dimension":            item.Dimension,
					"title":                item.Title,
					"description":          item.Description,
					"evidence": decodeCWReviewEvidence(
						item.EvidenceJSON,
					),
					"original_suggestion":   item.OriginalSuggestion,
					"confirmed_instruction": item.ConfirmedInstruction,
					"status":                item.Status,
				},
			)
	}

	problemsJSON, err :=
		json.Marshal(
			problems,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化课件正式整改问题失败: %w",
			err,
		)
	}

	result.ReviewItemIDs =
		itemIDs
	result.ObviousProblemsJSON =
		string(problemsJSON)

	return result, nil
}

// isMatchingCWFormalReviewSession 校验正式AI审核会话归属。
func isMatchingCWFormalReviewSession(
	session *models.CoursewareAIReviewSession,
	courseware *models.Courseware,
	actor *CoursewareActorContext,
	reviewLevel int,
) bool {
	if session == nil ||
		courseware == nil ||
		actor == nil {
		return false
	}

	return session.CoursewareID ==
		courseware.ID &&
		session.ReviewerID ==
			actor.UserID &&
		session.ReviewLevel ==
			reviewLevel &&
		session.Status ==
			models.CWAIReviewStatusDone
}

// isSelectableCWFormalReviewItem 校验本轮新问题能否正式交付。
func isSelectableCWFormalReviewItem(
	item *models.CoursewareReviewItem,
	courseware *models.Courseware,
	session *models.CoursewareAIReviewSession,
	reviewLevel int,
) bool {
	if item == nil ||
		courseware == nil ||
		session == nil {
		return false
	}

	if item.CoursewareID !=
		courseware.ID ||
		item.SourceSessionID !=
			session.ID ||
		item.SourceType !=
			models.CWReviewItemSourceFormal ||
		item.ReviewLevel !=
			reviewLevel {
		return false
	}

	if item.CoursewareReviewID != nil ||
		item.FeedbackID != nil {
		return false
	}

	if item.Status !=
		models.CWReviewItemStatusConfirmed {
		return false
	}

	return strings.TrimSpace(
		item.ConfirmedInstruction,
	) != ""
}

// decodeCWReviewEvidence 将证据JSON转换为快照对象。
func decodeCWReviewEvidence(
	raw string,
) interface{} {
	var value interface{}

	if err := json.Unmarshal(
		[]byte(raw),
		&value,
	); err == nil {
		return value
	}

	return map[string]interface{}{
		"raw": raw,
	}
}

// normalizeCWFormalReviewItemIDs 去空、去重并保留原顺序。
func normalizeCWFormalReviewItemIDs(
	input []string,
) []string {
	result :=
		make(
			[]string,
			0,
			len(input),
		)

	seen :=
		make(
			map[string]bool,
		)

	for _, raw := range input {
		value :=
			strings.TrimSpace(
				raw,
			)

		if value == "" ||
			seen[value] {
			continue
		}

		seen[value] = true
		result =
			append(
				result,
				value,
			)
	}

	return result
}

// mapCWReviewCommitError 将事务仓储错误映射为稳定服务错误。
func mapCWReviewCommitError(
	err error,
	reviewLevel int,
) error {
	switch {
	case errors.Is(
		err,
		repository.ErrCWReviewDecisionStateConflict,
	):
		if reviewLevel ==
			models.ReviewLevelL1 {
			return ErrCWReviewNotSubmitted
		}

		return ErrCWReviewNotL2Status

	case errors.Is(
		err,
		repository.ErrCWReviewDecisionSessionInvalid,
	),
		errors.Is(
			err,
			repository.ErrCWReviewDecisionItemsInvalid,
		),
		errors.Is(
			err,
			repository.ErrCWReviewDecisionCarryoverInvalid,
		):
		return ErrCWReviewFeedbackInvalid

	default:
		return err
	}
}

// afterCWReviewDecisionCommitted 在事务提交后记录日志和发送旁路通知。
func (s *CoursewareReviewService) afterCWReviewDecisionCommitted(
	ctx context.Context,
	courseware *models.Courseware,
	reviewerID string,
	review *models.CoursewareReview,
	needL2 bool,
) {
	if courseware == nil ||
		review == nil {
		return
	}

	switch review.Decision {
	case models.ReviewDecisionApproved:
		if review.ReviewLevel ==
			models.ReviewLevelL1 &&
			needL2 {
			cwReviewLog.Info(
				"课件L1审核通过，进入L2",
				"courseware_id",
				courseware.ID,
				"round",
				review.ReviewRound,
			)
			return
		}

		cwReviewLog.Info(
			"课件审核通过",
			"courseware_id",
			courseware.ID,
			"review_level",
			review.ReviewLevel,
			"round",
			review.ReviewRound,
		)

		s.notifyAuthorReviewResult(
			ctx,
			courseware,
			reviewerID,
			models.ReviewDecisionApproved,
			"",
		)

	case models.ReviewDecisionRevision:
		cwReviewLog.Info(
			"课件审核退回",
			"courseware_id",
			courseware.ID,
			"review_level",
			review.ReviewLevel,
			"round",
			review.ReviewRound,
		)

		s.notifyAuthorReviewResult(
			ctx,
			courseware,
			reviewerID,
			models.ReviewDecisionRevision,
			review.Comment,
		)
	}
}
