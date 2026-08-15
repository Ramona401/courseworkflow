package services

// courseware_review_comment_candidate.go
//
// R-08 正式课件审核意见重新汇总服务。
//
// 本文件负责：
//
//   1. 接收教师当前“本次修改清单”ID集合和当前审核意见；
//   2. 重新读取服务端正式整改项，拒绝无效、未确认或已交付条目；
//   3. 对每条整改项重新读取current_instruction_version；
//   4. 重新读取R-06 active问题组及成员version；
//   5. 构建稳定排序的可信输入快照并计算SHA-256；
//   6. 调AI生成一份新的完整审核意见候选；
//   7. 由代码确定性计算原意见与候选的added/removed/adjusted差异；
//   8. 保存不可变候选；
//   9. 教师选择replace或append时重新构建当前事实快照；
//  10. 任何可信输入发生变化都fail-closed为stale；
//  11. 验证通过后只返回新的输入框文本，不直接提交正式审核决定。
//
// 安全边界：
//
//   - 浏览器可以提交“当前选择”和“教师自己正在编辑的comment”，因为它们是教师意图；
//   - 浏览器不得提交candidate_text、input_hash、指令正文、组version或AI差异作为事实源；
//   - current instruction、组和成员version全部由后端重新读取；
//   - 候选AI调用失败不得把已经done的审核Session改成failed；
//   - R-08不复用R-07 impact plan，也不执行页面修改和整改项治理。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwReviewCommentCandidateMaxSelectedItems = 100
	cwReviewCommentCandidateMaxCommentRunes  = 20000
	cwReviewCommentCandidateMaxOutputRunes   = 20000
	cwReviewCommentDiffMaxLines              = 300
)

var (
	ErrCWReviewCommentCandidateInvalid = errors.New(
		"审核意见重新整理参数无效",
	)

	ErrCWReviewCommentCandidateNoItems = errors.New(
		"本次修改清单为空，无法重新整理审核意见",
	)

	ErrCWReviewCommentCandidateStale = errors.New(
		"本次修改清单、修改要求、问题组或审核意见已经变化，需要重新整理审核意见",
	)

	ErrCWReviewCommentCandidateOutputInvalid = errors.New(
		"AI返回的审核意见候选格式无效",
	)
)

// CWReviewCommentCandidateGenerateInput 是教师发起重新整理时允许提交的输入。
//
// SelectedItemIDs只是教师当前选择意图；Service会重新读取每个ID对应的正式整改事实。
// OriginalComment是教师本人尚未正式提交的本地草稿，不属于AI可信正文。
type CWReviewCommentCandidateGenerateInput struct {
	SelectedItemIDs []string
	OriginalComment string
}

// CWReviewCommentCandidateApplyInput 是教师确认replace/append时允许提交的输入。
//
// CurrentComment和SelectedItemIDs必须重新提交，用于检测候选生成后的本地变化。
// Candidate正文和input hash绝不能由浏览器提交。
type CWReviewCommentCandidateApplyInput struct {
	CandidateID     string
	Action          string
	SelectedItemIDs []string
	CurrentComment  string
}

// CWReviewCommentCandidateApplyResult 是通过stale复核后的输入框新值。
//
// 本结果仍不是正式审核事实，必须继续经过既有reviewCWL1/reviewCWL2人工提交事务。
type CWReviewCommentCandidateApplyResult struct {
	CandidateID string
	Action      string
	NextComment string
}

// cwReviewCommentCandidateItemSnapshot 是一条正式交付候选整改项的可信冻结事实。
type cwReviewCommentCandidateItemSnapshot struct {
	ItemID string `json:"item_id"`

	PageNumber int    `json:"page_number"`
	PageTitle  string `json:"page_title"`

	Severity    string `json:"severity"`
	Dimension   string `json:"dimension"`
	Title       string `json:"title"`
	Description string `json:"description"`

	ItemStatus string `json:"item_status"`

	InstructionVersionID string `json:"instruction_version_id"`
	InstructionVersionNo int    `json:"instruction_version_no"`
	InstructionContent   string `json:"instruction_content"`
	InstructionHash      string `json:"instruction_hash"`
	InstructionPageHash  string `json:"instruction_page_hash"`
	InstructionStatus    string `json:"instruction_status"`
}

// cwReviewCommentCandidateGroupMemberSnapshot 冻结R-06有效成员身份及version。
type cwReviewCommentCandidateGroupMemberSnapshot struct {
	MemberID string `json:"member_id"`
	ItemID   string `json:"item_id"`
	Version  int    `json:"version"`

	SelectedForDelivery bool `json:"selected_for_delivery"`
}

// cwReviewCommentCandidateGroupSnapshot 冻结影响当前交付清单的问题组事实。
//
// 只冻结至少包含一条当前selected item的active组；
// 但组内会保存全部active成员，从而让后续成员移动、加入、移除能够触发stale。
type cwReviewCommentCandidateGroupSnapshot struct {
	GroupID string `json:"group_id"`
	Name    string `json:"name"`

	PrimaryItemID string `json:"primary_item_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	Members []cwReviewCommentCandidateGroupMemberSnapshot `json:"members"`
}

// cwReviewCommentCandidateInputSnapshot 是R-08 input_hash的唯一规范化结构。
//
// 必须保持字段和排序确定性；禁止在其中使用map作为hash事实源。
type cwReviewCommentCandidateInputSnapshot struct {
	SchemaVersion int `json:"schema_version"`

	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`
	ReviewerID      string `json:"reviewer_id"`
	ReviewLevel     int    `json:"review_level"`

	OriginalCommentHash string `json:"original_comment_hash"`

	SelectedItemIDs []string                                `json:"selected_item_ids"`
	Items           []cwReviewCommentCandidateItemSnapshot  `json:"items"`
	Groups          []cwReviewCommentCandidateGroupSnapshot `json:"groups"`
}

// cwReviewCommentCandidateAIRequest 是发送给AI的最小教师化输入。
type cwReviewCommentCandidateAIRequest struct {
	OriginalComment string `json:"original_comment"`

	Items  []cwReviewCommentCandidateItemSnapshot  `json:"selected_items"`
	Groups []cwReviewCommentCandidateGroupSnapshot `json:"relevant_problem_groups"`
}

// cwReviewCommentCandidateAIResponse 是AI唯一允许返回的结构。
type cwReviewCommentCandidateAIResponse struct {
	CandidateText string `json:"candidate_text"`
}

// GenerateCWReviewCommentCandidate 根据当前可信事实生成新的不可变审核意见候选。
func (s *CoursewareAIReviewRunner) GenerateCWReviewCommentCandidate(
	ctx context.Context,
	sessionID string,
	input *CWReviewCommentCandidateGenerateInput,
	actor *CoursewareActorContext,
) (*models.CoursewareReviewCommentCandidate, error) {
	if s == nil || s.cfg == nil {
		return nil, errors.New("课件AI审核执行器未初始化")
	}
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}
	if input == nil {
		return nil, ErrCWReviewCommentCandidateInvalid
	}

	originalComment, err := normalizeCWReviewCommentCandidateText(
		input.OriginalComment,
		cwReviewCommentCandidateMaxCommentRunes,
	)
	if err != nil {
		return nil, err
	}

	session, err := s.authorizeCWReviewCommentCandidateSession(
		ctx,
		sessionID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	snapshot, snapshotJSON, inputHash, selectedIDsJSON, err :=
		buildCurrentCWReviewCommentCandidateSnapshot(
			ctx,
			session,
			input.SelectedItemIDs,
			originalComment,
			actor,
		)
	if err != nil {
		return nil, err
	}

	systemPrompt := buildCWReviewCommentCandidateSystemPrompt()

	userPrompt, err := buildCWReviewCommentCandidateUserPrompt(
		originalComment,
		snapshot,
	)
	if err != nil {
		return nil, err
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		"courseware_ai_review",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"获取审核意见重新整理模型配置失败: %w",
			err,
		)
	}

	userID := actor.UserID
	schoolID, _ := repository.GetSchoolIDByUserID(
		ctx,
		actor.UserID,
	)

	traceContext := &ai.TraceContext{
		SceneCode: cwAIReviewTraceScene(session),
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	callResult, err := ai.CallAI(
		aiConfig,
		systemPrompt,
		userPrompt,
		traceContext,
	)
	if err != nil {
		// R-08是done Session之后的辅助候选生成。
		// 这里绝不能把原AI审核Session改写为failed。
		return nil, fmt.Errorf(
			"审核意见重新整理失败: %w",
			err,
		)
	}

	candidateText, err := parseCWReviewCommentCandidateAIResponse(
		callResult.Content,
	)
	if err != nil {
		return nil, err
	}

	diff := buildCWReviewCommentDiff(
		originalComment,
		candidateText,
	)

	diffJSONBytes, err := json.Marshal(diff)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化审核意见差异失败: %w",
			err,
		)
	}

	originalCommentHash := hashCWReviewCommentCandidateText(
		originalComment,
	)

	candidate, err := repository.CreateCoursewareReviewCommentCandidate(
		ctx,
		&repository.CreateCoursewareReviewCommentCandidateInput{
			CoursewareID:    session.CoursewareID,
			SourceSessionID: session.ID,
			CreatedBy:       actor.UserID,
			ReviewLevel:     session.ReviewLevel,

			CandidateSchemaVersion: models.CWReviewCommentCandidateSchemaVersion,
			CandidateText:          candidateText,

			OriginalCommentSnapshot: originalComment,
			OriginalCommentHash:     originalCommentHash,

			SelectedItemIDsJSON: selectedIDsJSON,

			InputSnapshotSchemaVersion: models.CWReviewCommentInputSnapshotSchemaVersion,
			InputSnapshotJSON:          snapshotJSON,
			InputHash:                  inputHash,

			DiffSchemaVersion: models.CWReviewCommentDiffSchemaVersion,
			DiffJSON:          string(diffJSONBytes),

			ModelUsed:  callResult.ModelUsed,
			TokensUsed: callResult.TokensUsed,
		},
	)
	if err != nil {
		return nil, err
	}

	return candidate, nil
}

// ApplyCWReviewCommentCandidate 在教师明确选择replace或append前执行完整stale复核。
//
// 成功只返回输入框新文本，不写courseware_reviews，也不改变课件状态。
func (s *CoursewareAIReviewRunner) ApplyCWReviewCommentCandidate(
	ctx context.Context,
	sessionID string,
	input *CWReviewCommentCandidateApplyInput,
	actor *CoursewareActorContext,
) (*CWReviewCommentCandidateApplyResult, error) {
	if s == nil {
		return nil, errors.New("课件AI审核执行器未初始化")
	}
	if actor == nil || strings.TrimSpace(actor.UserID) == "" {
		return nil, ErrCWAIReviewActorRequired
	}
	if input == nil {
		return nil, ErrCWReviewCommentCandidateInvalid
	}

	candidateID := strings.TrimSpace(input.CandidateID)
	action := strings.TrimSpace(input.Action)

	if candidateID == "" ||
		!models.IsCWReviewCommentCandidateApplyAction(action) {
		return nil, ErrCWReviewCommentCandidateInvalid
	}

	currentComment, err := normalizeCWReviewCommentCandidateText(
		input.CurrentComment,
		cwReviewCommentCandidateMaxCommentRunes,
	)
	if err != nil {
		return nil, err
	}

	session, err := s.authorizeCWReviewCommentCandidateSession(
		ctx,
		sessionID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	candidate, err := repository.GetCoursewareReviewCommentCandidate(
		ctx,
		candidateID,
		session.CoursewareID,
		session.ID,
		actor.UserID,
	)
	if err != nil {
		return nil, err
	}

	if candidate.ReviewLevel != session.ReviewLevel ||
		candidate.CandidateSchemaVersion !=
			models.CWReviewCommentCandidateSchemaVersion ||
		candidate.InputSnapshotSchemaVersion !=
			models.CWReviewCommentInputSnapshotSchemaVersion ||
		candidate.DiffSchemaVersion !=
			models.CWReviewCommentDiffSchemaVersion {
		return nil, ErrCWReviewCommentCandidateStale
	}

	_, _, currentInputHash, _, err :=
		buildCurrentCWReviewCommentCandidateSnapshot(
			ctx,
			session,
			input.SelectedItemIDs,
			currentComment,
			actor,
		)
	if err != nil {
		if isCWReviewCommentCandidateSnapshotStaleError(err) {
			return nil, ErrCWReviewCommentCandidateStale
		}

		return nil, err
	}

	if currentInputHash != candidate.InputHash {
		return nil, ErrCWReviewCommentCandidateStale
	}

	nextComment := candidate.CandidateText

	if action == models.CWReviewCommentCandidateApplyAppend {
		if currentComment == "" {
			nextComment = candidate.CandidateText
		} else {
			nextComment = strings.TrimSpace(
				currentComment + "\n\n" + candidate.CandidateText,
			)
		}
	}

	return &CWReviewCommentCandidateApplyResult{
		CandidateID: candidate.ID,
		Action:      action,
		NextComment: nextComment,
	}, nil
}

// isCWReviewCommentCandidateSnapshotStaleError 判断重新核对过程中哪些业务变化必须统一视为stale。
//
// 这些错误代表候选生成时曾经成立的事实现在已经不再成立；
// 不向浏览器泄露到底是指令版本、成员、组还是选择集合发生了变化。
//
// 数据库连接错误等真正基础设施故障不会被吞成stale，仍按原错误上抛。
func isCWReviewCommentCandidateSnapshotStaleError(
	err error,
) bool {
	switch {
	case errors.Is(
		err,
		ErrCWReviewCommentCandidateNoItems,
	):
		return true

	case errors.Is(
		err,
		ErrCWReviewCommentCandidateInvalid,
	):
		return true

	case errors.Is(
		err,
		repository.ErrCoursewareReviewInstructionVersionNotFound,
	):
		return true

	case errors.Is(
		err,
		repository.ErrCoursewareReviewInstructionVersionConflict,
	):
		return true

	case errors.Is(
		err,
		repository.ErrCoursewareReviewItemGroupNotFound,
	):
		return true

	case errors.Is(
		err,
		repository.ErrCoursewareReviewItemGroupMemberNotFound,
	):
		return true

	case errors.Is(
		err,
		repository.ErrCoursewareReviewItemGroupConflict,
	):
		return true

	default:
		return false
	}
}

// authorizeCWReviewCommentCandidateSession 复用R-06/R-07同一审核会话授权边界。
func (s *CoursewareAIReviewRunner) authorizeCWReviewCommentCandidateSession(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) (*models.CoursewareAIReviewSession, error) {
	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			strings.TrimSpace(sessionID),
			actor,
			false,
		)
	if err != nil {
		return nil, err
	}

	if session == nil ||
		session.Status != models.CWAIReviewStatusDone ||
		(session.ReviewLevel != models.ReviewLevelL1 &&
			session.ReviewLevel != models.ReviewLevelL2) {
		return nil, ErrCWReviewCommentCandidateInvalid
	}

	return session, nil
}

// buildCurrentCWReviewCommentCandidateSnapshot 重新读取当前全部可信输入并生成稳定hash。
func buildCurrentCWReviewCommentCandidateSnapshot(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	rawSelectedItemIDs []string,
	originalComment string,
	actor *CoursewareActorContext,
) (
	*cwReviewCommentCandidateInputSnapshot,
	string,
	string,
	string,
	error,
) {
	if session == nil ||
		actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, "", "", "", ErrCWReviewCommentCandidateInvalid
	}

	selectedItemIDs := normalizeCWReviewCommentCandidateItemIDs(
		rawSelectedItemIDs,
	)

	if len(selectedItemIDs) == 0 {
		return nil, "", "", "", ErrCWReviewCommentCandidateNoItems
	}
	if len(selectedItemIDs) >
		cwReviewCommentCandidateMaxSelectedItems {
		return nil, "", "", "", ErrCWReviewCommentCandidateInvalid
	}

	allItems, err :=
		repository.ListCoursewareReviewItemsBySessionForCreator(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, "", "", "", err
	}

	itemsByID := make(
		map[string]*models.CoursewareReviewItem,
		len(allItems),
	)

	for _, item := range allItems {
		if item != nil {
			itemsByID[item.ID] = item
		}
	}

	itemSnapshots := make(
		[]cwReviewCommentCandidateItemSnapshot,
		0,
		len(selectedItemIDs),
	)

	selectedSet := make(
		map[string]bool,
		len(selectedItemIDs),
	)

	for _, itemID := range selectedItemIDs {
		selectedSet[itemID] = true

		item := itemsByID[itemID]
		if !isSelectableCWReviewCommentCandidateItem(
			item,
			session,
		) {
			return nil, "", "", "", ErrCWReviewCommentCandidateInvalid
		}

		version, versionErr :=
			repository.GetCurrentCoursewareReviewInstructionVersion(
				ctx,
				item.ID,
				actor.UserID,
			)
		if versionErr != nil {
			return nil, "", "", "", versionErr
		}

		if version == nil ||
			version.ItemID != item.ID ||
			version.Status !=
				models.CWReviewInstructionVersionStatusConfirmed ||
			strings.TrimSpace(version.Content) == "" ||
			strings.TrimSpace(version.ContentHash) == "" {
			return nil, "", "", "", ErrCWReviewCommentCandidateInvalid
		}

		itemSnapshots = append(
			itemSnapshots,
			cwReviewCommentCandidateItemSnapshot{
				ItemID: item.ID,

				PageNumber: item.PageNumberSnapshot,
				PageTitle: strings.TrimSpace(
					item.PageTitleSnapshot,
				),

				Severity: strings.TrimSpace(
					item.Severity,
				),
				Dimension: strings.TrimSpace(
					item.Dimension,
				),
				Title: strings.TrimSpace(
					item.Title,
				),
				Description: strings.TrimSpace(
					item.Description,
				),

				ItemStatus: item.Status,

				InstructionVersionID: version.ID,
				InstructionVersionNo: version.VersionNo,
				InstructionContent: strings.TrimSpace(
					version.Content,
				),
				InstructionHash: strings.TrimSpace(
					version.ContentHash,
				),
				InstructionPageHash: strings.TrimSpace(
					version.PageSnapshotHash,
				),
				InstructionStatus: version.Status,
			},
		)
	}

	sort.Slice(
		itemSnapshots,
		func(i int, j int) bool {
			return itemSnapshots[i].ItemID <
				itemSnapshots[j].ItemID
		},
	)

	groupSnapshots, err :=
		buildCurrentCWReviewCommentCandidateGroupSnapshots(
			ctx,
			session,
			selectedSet,
			actor,
		)
	if err != nil {
		return nil, "", "", "", err
	}

	snapshot := &cwReviewCommentCandidateInputSnapshot{
		SchemaVersion: models.CWReviewCommentInputSnapshotSchemaVersion,

		CoursewareID:    session.CoursewareID,
		SourceSessionID: session.ID,
		ReviewerID:      actor.UserID,
		ReviewLevel:     session.ReviewLevel,

		OriginalCommentHash: hashCWReviewCommentCandidateText(
			originalComment,
		),

		SelectedItemIDs: selectedItemIDs,
		Items:           itemSnapshots,
		Groups:          groupSnapshots,
	}

	snapshotJSONBytes, err := json.Marshal(snapshot)
	if err != nil {
		return nil, "", "", "", fmt.Errorf(
			"序列化审核意见可信输入快照失败: %w",
			err,
		)
	}

	selectedIDsJSONBytes, err := json.Marshal(
		selectedItemIDs,
	)
	if err != nil {
		return nil, "", "", "", fmt.Errorf(
			"序列化本次修改清单失败: %w",
			err,
		)
	}

	snapshotJSON := string(snapshotJSONBytes)

	return snapshot,
		snapshotJSON,
		hashCWReviewCommentCandidateText(snapshotJSON),
		string(selectedIDsJSONBytes),
		nil
}

// buildCurrentCWReviewCommentCandidateGroupSnapshots 冻结与当前清单相关的R-06事实。
func buildCurrentCWReviewCommentCandidateGroupSnapshots(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	selectedSet map[string]bool,
	actor *CoursewareActorContext,
) ([]cwReviewCommentCandidateGroupSnapshot, error) {
	groups, err :=
		repository.ListCoursewareReviewItemGroupsBySession(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	result := make(
		[]cwReviewCommentCandidateGroupSnapshot,
		0,
		len(groups),
	)

	for _, group := range groups {
		if group == nil ||
			group.Status != models.CWReviewItemGroupStatusActive {
			continue
		}

		record, buildErr :=
			buildCWAIReviewItemGroupRecord(
				ctx,
				group,
				actor.UserID,
			)
		if buildErr != nil {
			return nil, buildErr
		}

		activeMembers := make(
			[]cwReviewCommentCandidateGroupMemberSnapshot,
			0,
			len(record.Members),
		)

		intersectsSelected := false

		for _, member := range record.Members {
			if member == nil ||
				member.Status !=
					models.CWReviewItemGroupMemberStatusActive {
				continue
			}

			selected := selectedSet[member.ItemID]
			if selected {
				intersectsSelected = true
			}

			activeMembers = append(
				activeMembers,
				cwReviewCommentCandidateGroupMemberSnapshot{
					MemberID: member.ID,
					ItemID:   member.ItemID,
					Version:  member.Version,

					SelectedForDelivery: selected,
				},
			)
		}

		if !intersectsSelected {
			continue
		}

		sort.Slice(
			activeMembers,
			func(i int, j int) bool {
				return activeMembers[i].MemberID <
					activeMembers[j].MemberID
			},
		)

		primaryItemID := ""
		if group.PrimaryItemID != nil {
			primaryItemID = strings.TrimSpace(
				*group.PrimaryItemID,
			)
		}

		result = append(
			result,
			cwReviewCommentCandidateGroupSnapshot{
				GroupID: group.ID,
				Name: strings.TrimSpace(
					group.Name,
				),
				PrimaryItemID: primaryItemID,
				Status:        group.Status,
				Version:       group.Version,
				Members:       activeMembers,
			},
		)
	}

	sort.Slice(
		result,
		func(i int, j int) bool {
			return result[i].GroupID <
				result[j].GroupID
		},
	)

	return result, nil
}

// isSelectableCWReviewCommentCandidateItem 校验当前清单中的整改项仍可进入正式退回。
func isSelectableCWReviewCommentCandidateItem(
	item *models.CoursewareReviewItem,
	session *models.CoursewareAIReviewSession,
) bool {
	if item == nil || session == nil {
		return false
	}

	if item.CoursewareID != session.CoursewareID ||
		item.SourceSessionID != session.ID ||
		item.SourceType != models.CWReviewItemSourceFormal ||
		item.ReviewLevel != session.ReviewLevel {
		return false
	}

	if item.CoursewareReviewID != nil ||
		item.FeedbackID != nil {
		return false
	}

	return item.Status ==
		models.CWReviewItemStatusConfirmed
}

// normalizeCWReviewCommentCandidateItemIDs 将清单按集合语义规范化。
//
// 排序后hash不受前端展示顺序变化影响，但成员增删一定改变hash。
func normalizeCWReviewCommentCandidateItemIDs(
	input []string,
) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))

	for _, raw := range input {
		itemID := strings.TrimSpace(raw)
		if itemID == "" || seen[itemID] {
			continue
		}

		seen[itemID] = true
		result = append(result, itemID)
	}

	sort.Strings(result)

	return result
}

// normalizeCWReviewCommentCandidateText 统一换行并限制教师意见或AI候选长度。
func normalizeCWReviewCommentCandidateText(
	value string,
	maxRunes int,
) (string, error) {
	normalized := strings.ReplaceAll(
		value,
		"\r\n",
		"\n",
	)
	normalized = strings.ReplaceAll(
		normalized,
		"\r",
		"\n",
	)
	normalized = strings.TrimSpace(normalized)

	if maxRunes > 0 &&
		utf8.RuneCountInString(normalized) > maxRunes {
		return "", ErrCWReviewCommentCandidateInvalid
	}

	return normalized, nil
}

// hashCWReviewCommentCandidateText 生成稳定SHA-256十六进制指纹。
func hashCWReviewCommentCandidateText(
	value string,
) string {
	sum := sha256.Sum256(
		[]byte(value),
	)

	return hex.EncodeToString(
		sum[:],
	)
}

// buildCWReviewCommentCandidateSystemPrompt 构建R-08严格系统提示。
func buildCWReviewCommentCandidateSystemPrompt() string {
	return strings.TrimSpace(`
你是正式课件人工审核中的“审核意见整理助手”。

你的唯一任务是：
根据审核员已经明确选择的“本次修改清单”、每条问题当前已确认的修改要求、
教师已有审核意见，以及相关问题组的人工治理结构，
整理一份新的完整审核意见候选。

必须遵守：

1. 只能使用输入中提供的事实，不能新增问题、页码、修改要求或审核结论。
2. selected_items中的instruction_content是当前已确认修改要求，优先级高于旧问题描述。
3. relevant_problem_groups只用于帮助归并表达和减少重复。
4. 问题组中selected_for_delivery=false的成员不是本次正式退回清单，
   不得因为它们属于同组就把其修改要求写入本次候选。
5. 如果original_comment已有教师判断、语气或重点，应尽量保留其有效语义，
   但可以重新组织、去重和提高可执行性。
6. 不得替人工审核员自行新增“通过”“退回”决定。
7. 不得声称已经修改页面、已经解决问题或已经完成复核。
8. 输出应适合作为教师正式审核意见：清晰、专业、简洁、可执行。
9. 只输出一个合法JSON对象，不要Markdown代码围栏，不要额外说明。

输出结构严格为：

{
  "candidate_text": "新的完整审核意见候选"
}
`)
}

// buildCWReviewCommentCandidateUserPrompt 构建最小可信AI输入。
func buildCWReviewCommentCandidateUserPrompt(
	originalComment string,
	snapshot *cwReviewCommentCandidateInputSnapshot,
) (string, error) {
	if snapshot == nil {
		return "", ErrCWReviewCommentCandidateInvalid
	}

	payload := cwReviewCommentCandidateAIRequest{
		OriginalComment: originalComment,
		Items:           snapshot.Items,
		Groups:          snapshot.Groups,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(
			"序列化审核意见重新整理输入失败: %w",
			err,
		)
	}

	return "请根据以下可信审核事实生成新的完整审核意见候选：\n\n" +
			string(encoded),
		nil
}

// parseCWReviewCommentCandidateAIResponse 解析AI候选且只接受candidate_text。
func parseCWReviewCommentCandidateAIResponse(
	raw string,
) (string, error) {
	jsonText, ok := ai.ExtractJSON(raw)
	if !ok || strings.TrimSpace(jsonText) == "" {
		jsonText = strings.TrimSpace(raw)
	}

	var response cwReviewCommentCandidateAIResponse

	if err := json.Unmarshal(
		[]byte(jsonText),
		&response,
	); err != nil {
		return "", fmt.Errorf(
			"%w: %v",
			ErrCWReviewCommentCandidateOutputInvalid,
			err,
		)
	}

	candidateText, err :=
		normalizeCWReviewCommentCandidateText(
			response.CandidateText,
			cwReviewCommentCandidateMaxOutputRunes,
		)
	if err != nil {
		return "", ErrCWReviewCommentCandidateOutputInvalid
	}

	if candidateText == "" {
		return "", ErrCWReviewCommentCandidateOutputInvalid
	}

	return candidateText, nil
}

// buildCWReviewCommentDiff 以行级确定性算法生成added/removed/adjusted。
//
// 最多对300行执行LCS，避免异常输入造成O(n*m)内存放大。
// 超过限制时退化为整段adjusted，仍保持正确的“原文已变化”语义。
func buildCWReviewCommentDiff(
	original string,
	candidate string,
) models.CWReviewCommentDiff {
	originalLines :=
		splitCWReviewCommentDiffLines(
			original,
		)

	candidateLines :=
		splitCWReviewCommentDiffLines(
			candidate,
		)

	diff := models.CWReviewCommentDiff{
		Added:    []string{},
		Removed:  []string{},
		Adjusted: []models.CWReviewCommentDiffAdjustment{},
	}

	if strings.TrimSpace(original) ==
		strings.TrimSpace(candidate) {
		return diff
	}

	if len(originalLines) >
		cwReviewCommentDiffMaxLines ||
		len(candidateLines) >
			cwReviewCommentDiffMaxLines {
		switch {
		case len(originalLines) == 0:
			diff.Added = append(
				diff.Added,
				strings.TrimSpace(candidate),
			)
		case len(candidateLines) == 0:
			diff.Removed = append(
				diff.Removed,
				strings.TrimSpace(original),
			)
		default:
			diff.Adjusted = append(
				diff.Adjusted,
				models.CWReviewCommentDiffAdjustment{
					Before: strings.TrimSpace(
						original,
					),
					After: strings.TrimSpace(
						candidate,
					),
				},
			)
		}

		return diff
	}

	lcs := buildCWReviewCommentLineLCS(
		originalLines,
		candidateLines,
	)

	i := 0
	j := 0

	for i < len(originalLines) ||
		j < len(candidateLines) {
		if i < len(originalLines) &&
			j < len(candidateLines) &&
			originalLines[i] == candidateLines[j] {
			i++
			j++
			continue
		}

		removed := make([]string, 0)
		added := make([]string, 0)

		for i < len(originalLines) ||
			j < len(candidateLines) {
			if i < len(originalLines) &&
				j < len(candidateLines) &&
				originalLines[i] == candidateLines[j] {
				break
			}

			if j >= len(candidateLines) ||
				(i < len(originalLines) &&
					lcs[i+1][j] >= lcs[i][j+1]) {
				removed = append(
					removed,
					originalLines[i],
				)
				i++
				continue
			}

			added = append(
				added,
				candidateLines[j],
			)
			j++
		}

		paired := len(removed)
		if len(added) < paired {
			paired = len(added)
		}

		for index := 0; index < paired; index++ {
			diff.Adjusted = append(
				diff.Adjusted,
				models.CWReviewCommentDiffAdjustment{
					Before: removed[index],
					After:  added[index],
				},
			)
		}

		if paired < len(removed) {
			diff.Removed = append(
				diff.Removed,
				removed[paired:]...,
			)
		}

		if paired < len(added) {
			diff.Added = append(
				diff.Added,
				added[paired:]...,
			)
		}
	}

	return diff
}

// splitCWReviewCommentDiffLines 将意见转为适合教师查看差异的非空行集合。
func splitCWReviewCommentDiffLines(
	value string,
) []string {
	normalized := strings.ReplaceAll(
		value,
		"\r\n",
		"\n",
	)
	normalized = strings.ReplaceAll(
		normalized,
		"\r",
		"\n",
	)

	rawLines := strings.Split(
		normalized,
		"\n",
	)

	result := make(
		[]string,
		0,
		len(rawLines),
	)

	for _, raw := range rawLines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		result = append(
			result,
			line,
		)
	}

	return result
}

// buildCWReviewCommentLineLCS 构建行级最长公共子序列矩阵。
func buildCWReviewCommentLineLCS(
	original []string,
	candidate []string,
) [][]int {
	lcs := make(
		[][]int,
		len(original)+1,
	)

	for i := range lcs {
		lcs[i] = make(
			[]int,
			len(candidate)+1,
		)
	}

	for i := len(original) - 1; i >= 0; i-- {
		for j := len(candidate) - 1; j >= 0; j-- {
			if original[i] == candidate[j] {
				lcs[i][j] =
					lcs[i+1][j+1] + 1
				continue
			}

			if lcs[i+1][j] >=
				lcs[i][j+1] {
				lcs[i][j] =
					lcs[i+1][j]
			} else {
				lcs[i][j] =
					lcs[i][j+1]
			}
		}
	}

	return lcs
}
