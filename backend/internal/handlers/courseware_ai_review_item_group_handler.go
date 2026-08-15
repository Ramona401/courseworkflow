package handlers

// courseware_ai_review_item_group_handler.go
//
// R-06 正式问题组HTTP入口。
//
// 路由：
//   GET  /api/v1/courseware-ai-reviews/{session_id}/groups
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups/move-member
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups/merge
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups/split
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups/{group_id}/rename
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups/{group_id}/primary
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups/{group_id}/members
//   POST /api/v1/courseware-ai-reviews/{session_id}/groups/{group_id}/remove-member
//
// 安全边界：
//   1. 操作者身份只从JWT构建，不接受浏览器提交用户身份；
//   2. 浏览器只提交组名、目标ID和乐观并发version；
//   3. 会话归属、整改项可治理性和页面新鲜度由Service重新读取；
//   4. 请求正文限长、拒绝未知字段和尾随JSON；
//   5. 所有动作只治理问题组，不修改页面、确认指令或审核决定。

import (
	"encoding/json"
	"io"
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

const coursewareAIReviewItemGroupBodyMaxBytes = 64 * 1024

type createCWAIReviewItemGroupRequest struct {
	Name          string   `json:"name"`
	ItemIDs       []string `json:"item_ids"`
	PrimaryItemID string   `json:"primary_item_id"`
}

type renameCWAIReviewItemGroupRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Name            string `json:"name"`
}

type setCWAIReviewItemGroupPrimaryRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	PrimaryItemID   string `json:"primary_item_id"`
}

type addCWAIReviewItemGroupMemberRequest struct {
	ExpectedGroupVersion int    `json:"expected_group_version"`
	ItemID               string `json:"item_id"`
}

type removeCWAIReviewItemGroupMemberRequest struct {
	ExpectedGroupVersion  int    `json:"expected_group_version"`
	ExpectedMemberVersion int    `json:"expected_member_version"`
	MemberID              string `json:"member_id"`
	Reason                string `json:"reason"`
}

type moveCWAIReviewItemGroupMemberRequest struct {
	SourceGroupID string `json:"source_group_id"`
	TargetGroupID string `json:"target_group_id"`

	ExpectedSourceVersion int `json:"expected_source_version"`
	ExpectedTargetVersion int `json:"expected_target_version"`

	MemberID              string `json:"member_id"`
	ExpectedMemberVersion int    `json:"expected_member_version"`
	Reason                string `json:"reason"`
}

type mergeCWAIReviewItemGroupsRequest struct {
	SourceGroupID string `json:"source_group_id"`
	TargetGroupID string `json:"target_group_id"`

	ExpectedSourceVersion int    `json:"expected_source_version"`
	ExpectedTargetVersion int    `json:"expected_target_version"`
	Reason                string `json:"reason"`
}

type splitCWAIReviewItemGroupRequest struct {
	SourceGroupID         string `json:"source_group_id"`
	ExpectedSourceVersion int    `json:"expected_source_version"`

	Name          string   `json:"name"`
	ItemIDs       []string `json:"item_ids"`
	PrimaryItemID string   `json:"primary_item_id"`
	Reason        string   `json:"reason"`
}

// isCoursewareAIReviewItemGroupRoute 判断是否属于会话级R-06问题组路由。
func isCoursewareAIReviewItemGroupRoute(parts []string) bool {
	if len(parts) < 2 || parts[0] == "items" || parts[1] != "groups" {
		return false
	}

	if len(parts) == 2 {
		return true
	}

	if len(parts) == 3 {
		switch parts[2] {
		case "move-member", "merge", "split":
			return true
		default:
			return false
		}
	}

	if len(parts) == 4 && parts[2] != "" {
		switch parts[3] {
		case "rename", "primary", "members", "remove-member":
			return true
		default:
			return false
		}
	}

	return false
}

// HandleReviewItemGroupRoute 处理R-06问题组列表和全部人工治理动作。
func (h *CoursewareAIReviewHandler) HandleReviewItemGroupRoute(
	w http.ResponseWriter,
	r *http.Request,
	parts []string,
) {
	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	if h == nil || h.runner == nil {
		utils.InternalError(w, "课件AI审核执行器未初始化")
		return
	}

	sessionID := ""
	if len(parts) > 0 {
		sessionID = parts[0]
	}

	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		h.listReviewItemGroups(w, r, sessionID, actor)

	case len(parts) == 2 && r.Method == http.MethodPost:
		h.createReviewItemGroup(w, r, sessionID, actor)

	case len(parts) == 3 &&
		parts[2] == "move-member" &&
		r.Method == http.MethodPost:
		h.moveReviewItemGroupMember(w, r, sessionID, actor)

	case len(parts) == 3 &&
		parts[2] == "merge" &&
		r.Method == http.MethodPost:
		h.mergeReviewItemGroups(w, r, sessionID, actor)

	case len(parts) == 3 &&
		parts[2] == "split" &&
		r.Method == http.MethodPost:
		h.splitReviewItemGroup(w, r, sessionID, actor)

	case len(parts) == 4 &&
		parts[3] == "rename" &&
		r.Method == http.MethodPost:
		h.renameReviewItemGroup(
			w,
			r,
			sessionID,
			parts[2],
			actor,
		)

	case len(parts) == 4 &&
		parts[3] == "primary" &&
		r.Method == http.MethodPost:
		h.setReviewItemGroupPrimary(
			w,
			r,
			sessionID,
			parts[2],
			actor,
		)

	case len(parts) == 4 &&
		parts[3] == "members" &&
		r.Method == http.MethodPost:
		h.addReviewItemGroupMember(
			w,
			r,
			sessionID,
			parts[2],
			actor,
		)

	case len(parts) == 4 &&
		parts[3] == "remove-member" &&
		r.Method == http.MethodPost:
		h.removeReviewItemGroupMember(
			w,
			r,
			sessionID,
			parts[2],
			actor,
		)

	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"问题组路由或请求方法无效",
		)
	}
}

func (h *CoursewareAIReviewHandler) listReviewItemGroups(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	records, err := h.runner.ListCWAIReviewItemGroups(
		r.Context(),
		sessionID,
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"groups": buildCoursewareAIReviewItemGroupRecordViews(
				records,
			),
		},
	)
}

func (h *CoursewareAIReviewHandler) createReviewItemGroup(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req createCWAIReviewItemGroupRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	record, err := h.runner.CreateCWAIReviewItemGroup(
		r.Context(),
		sessionID,
		&services.CWAIReviewItemGroupCreateInput{
			Name:          req.Name,
			ItemIDs:       req.ItemIDs,
			PrimaryItemID: req.PrimaryItemID,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupRecordView(record),
	)
}

func (h *CoursewareAIReviewHandler) renameReviewItemGroup(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	groupID string,
	actor *services.CoursewareActorContext,
) {
	var req renameCWAIReviewItemGroupRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	record, err := h.runner.RenameCWAIReviewItemGroup(
		r.Context(),
		sessionID,
		groupID,
		&services.CWAIReviewItemGroupRenameInput{
			ExpectedVersion: req.ExpectedVersion,
			Name:            req.Name,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupRecordView(record),
	)
}

func (h *CoursewareAIReviewHandler) setReviewItemGroupPrimary(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	groupID string,
	actor *services.CoursewareActorContext,
) {
	var req setCWAIReviewItemGroupPrimaryRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	record, err := h.runner.SetCWAIReviewItemGroupPrimary(
		r.Context(),
		sessionID,
		groupID,
		&services.CWAIReviewItemGroupPrimaryInput{
			ExpectedVersion: req.ExpectedVersion,
			PrimaryItemID:   req.PrimaryItemID,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupRecordView(record),
	)
}

func (h *CoursewareAIReviewHandler) addReviewItemGroupMember(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	groupID string,
	actor *services.CoursewareActorContext,
) {
	var req addCWAIReviewItemGroupMemberRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	record, err := h.runner.AddCWAIReviewItemGroupMember(
		r.Context(),
		sessionID,
		groupID,
		&services.CWAIReviewItemGroupAddMemberInput{
			ExpectedGroupVersion: req.ExpectedGroupVersion,
			ItemID:               req.ItemID,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupRecordView(record),
	)
}

func (h *CoursewareAIReviewHandler) removeReviewItemGroupMember(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	groupID string,
	actor *services.CoursewareActorContext,
) {
	var req removeCWAIReviewItemGroupMemberRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	record, err := h.runner.RemoveCWAIReviewItemGroupMember(
		r.Context(),
		sessionID,
		groupID,
		&services.CWAIReviewItemGroupRemoveMemberInput{
			ExpectedGroupVersion:  req.ExpectedGroupVersion,
			ExpectedMemberVersion: req.ExpectedMemberVersion,
			MemberID:              req.MemberID,
			Reason:                req.Reason,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupRecordView(record),
	)
}

func (h *CoursewareAIReviewHandler) moveReviewItemGroupMember(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req moveCWAIReviewItemGroupMemberRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	result, err := h.runner.MoveCWAIReviewItemGroupMember(
		r.Context(),
		sessionID,
		&services.CWAIReviewItemGroupMoveMemberInput{
			SourceGroupID:         req.SourceGroupID,
			TargetGroupID:         req.TargetGroupID,
			ExpectedSourceVersion: req.ExpectedSourceVersion,
			ExpectedTargetVersion: req.ExpectedTargetVersion,
			MemberID:              req.MemberID,
			ExpectedMemberVersion: req.ExpectedMemberVersion,
			Reason:                req.Reason,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupPairView(result),
	)
}

func (h *CoursewareAIReviewHandler) mergeReviewItemGroups(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req mergeCWAIReviewItemGroupsRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	result, err := h.runner.MergeCWAIReviewItemGroups(
		r.Context(),
		sessionID,
		&services.CWAIReviewItemGroupMergeInput{
			SourceGroupID:         req.SourceGroupID,
			TargetGroupID:         req.TargetGroupID,
			ExpectedSourceVersion: req.ExpectedSourceVersion,
			ExpectedTargetVersion: req.ExpectedTargetVersion,
			Reason:                req.Reason,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupPairView(result),
	)
}

func (h *CoursewareAIReviewHandler) splitReviewItemGroup(
	w http.ResponseWriter,
	r *http.Request,
	sessionID string,
	actor *services.CoursewareActorContext,
) {
	var req splitCWAIReviewItemGroupRequest
	if !decodeCWAIReviewItemGroupRequest(w, r, &req) {
		return
	}

	result, err := h.runner.SplitCWAIReviewItemGroup(
		r.Context(),
		sessionID,
		&services.CWAIReviewItemGroupSplitInput{
			SourceGroupID:         req.SourceGroupID,
			ExpectedSourceVersion: req.ExpectedSourceVersion,
			Name:                  req.Name,
			ItemIDs:               req.ItemIDs,
			PrimaryItemID:         req.PrimaryItemID,
			Reason:                req.Reason,
		},
		actor,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemGroupPairView(result),
	)
}

func decodeCWAIReviewItemGroupRequest(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareAIReviewItemGroupBodyMaxBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误或内容过大",
		)
		return false
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		utils.BadRequest(
			w,
			"请求正文只能包含一个JSON对象",
		)
		return false
	}

	return true
}
