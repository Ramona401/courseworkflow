package handlers

// courseware_ai_review_instruction_version_handler.go
//
// 课件审核整改指令版本HTTP响应与精确子路由处理。
//
// 路由：
//   GET  /api/v1/courseware-ai-reviews/items/{item_id}/instruction-versions
//   GET  /api/v1/courseware-ai-reviews/items/{item_id}/instruction-versions/current
//   GET  /api/v1/courseware-ai-reviews/items/{item_id}/instruction-versions/{version_id}
//   POST /api/v1/courseware-ai-reviews/items/{item_id}/instruction-versions/confirm
//
// 同时接管旧POST /items/{item_id}/confirm，使旧路径也必须提交
// expected_current_version_id并走不可变版本事务。其他整改项路径直接委托原HandleItem。
//
// 浏览器安全边界：
//   1. 只返回版本业务字段，不返回created_by和confirmed_by内部用户ID；
//   2. 保存并确认只接收instruction和expected_current_version_id；
//   3. 创建人、确认人、来源、页面快照、内容哈希、版本号和状态全部由后端生成；
//   4. 当前版本为空时返回null，历史列表始终返回空数组而不是null。

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

type confirmCWReviewInstructionVersionRequest struct {
	Instruction              string `json:"instruction"`
	ExpectedCurrentVersionID string `json:"expected_current_version_id"`
}

// coursewareAIReviewInstructionVersionView 是浏览器安全的不可变版本视图。
type coursewareAIReviewInstructionVersionView struct {
	ID        string `json:"id"`
	ItemID    string `json:"item_id"`
	VersionNo int    `json:"version_no"`

	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
	SourceType  string `json:"source_type"`

	CreatedAt   *time.Time `json:"created_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`

	PageSnapshotHash string `json:"page_snapshot_hash"`
	Status           string `json:"status"`
	IsCurrent        bool   `json:"is_current"`
}

func buildCoursewareAIReviewInstructionVersionView(
	version *models.CoursewareReviewInstructionVersion,
	currentVersionID string,
) *coursewareAIReviewInstructionVersionView {
	if version == nil {
		return nil
	}

	return &coursewareAIReviewInstructionVersionView{
		ID:               version.ID,
		ItemID:           version.ItemID,
		VersionNo:        version.VersionNo,
		Content:          version.Content,
		ContentHash:      version.ContentHash,
		SourceType:       version.SourceType,
		CreatedAt:        version.CreatedAt,
		ConfirmedAt:      version.ConfirmedAt,
		PageSnapshotHash: version.PageSnapshotHash,
		Status:           version.Status,
		IsCurrent: version.ID ==
			strings.TrimSpace(currentVersionID),
	}
}

func buildCoursewareAIReviewInstructionVersionViews(
	versions []*models.CoursewareReviewInstructionVersion,
	currentVersionID string,
) []*coursewareAIReviewInstructionVersionView {
	result := make(
		[]*coursewareAIReviewInstructionVersionView,
		0,
		len(versions),
	)

	for _, version := range versions {
		view :=
			buildCoursewareAIReviewInstructionVersionView(
				version,
				currentVersionID,
			)
		if view != nil {
			result = append(result, view)
		}
	}

	return result
}

func optionalCWReviewInstructionVersionID(
	value string,
) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}

// HandleReviewInstructionVersionRoute 处理更具体的/items/路径。
// 非版本路径委托原处理器，避免复制既有讨论、状态治理和列表逻辑。
func (h *CoursewareAIReviewHandler) HandleReviewInstructionVersionRoute(
	w http.ResponseWriter,
	r *http.Request,
) {
	parts :=
		parseCoursewareAIReviewPathParts(
			r.URL.Path,
		)

	if !isCWReviewInstructionVersionRoute(parts) {
		h.HandleItem(w, r)
		return
	}

	actor, ok := buildCoursewareAIReviewActor(r)
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	itemID := parts[1]

	switch {
	case len(parts) == 3 &&
		parts[2] == "confirm" &&
		r.Method == http.MethodPost:
		h.confirmReviewItemInstructionVersion(
			w,
			r,
			itemID,
			actor,
		)

	case len(parts) == 3 &&
		parts[2] == "instruction-versions" &&
		r.Method == http.MethodGet:
		h.listReviewItemInstructionVersions(
			w,
			r,
			itemID,
			actor,
		)

	case len(parts) == 4 &&
		parts[2] == "instruction-versions" &&
		parts[3] == "current" &&
		r.Method == http.MethodGet:
		h.getCurrentReviewItemInstructionVersion(
			w,
			r,
			itemID,
			actor,
		)

	case len(parts) == 4 &&
		parts[2] == "instruction-versions" &&
		parts[3] == "confirm" &&
		r.Method == http.MethodPost:
		h.confirmReviewItemInstructionVersion(
			w,
			r,
			itemID,
			actor,
		)

	case len(parts) == 4 &&
		parts[2] == "instruction-versions" &&
		r.Method == http.MethodGet:
		h.getReviewItemInstructionVersion(
			w,
			r,
			itemID,
			parts[3],
			actor,
		)

	default:
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"指令版本路由或请求方法无效",
		)
	}
}

func isCWReviewInstructionVersionRoute(
	parts []string,
) bool {
	if len(parts) < 3 ||
		parts[0] != "items" ||
		strings.TrimSpace(parts[1]) == "" {
		return false
	}

	if len(parts) == 3 &&
		parts[2] == "confirm" {
		return true
	}

	return parts[2] ==
		"instruction-versions"
}

func (h *CoursewareAIReviewHandler) listReviewItemInstructionVersions(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(
			w,
			"课件AI审核执行器未初始化",
		)
		return
	}

	result, err :=
		h.runner.ListCWReviewItemInstructionVersions(
			r.Context(),
			itemID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	versions :=
		[]*coursewareAIReviewInstructionVersionView{}

	currentVersionID := ""

	if result != nil {
		currentVersionID =
			result.CurrentVersionID

		versions =
			buildCoursewareAIReviewInstructionVersionViews(
				result.Versions,
				currentVersionID,
			)
	}

	utils.Success(
		w,
		map[string]interface{}{
			"current_instruction_version_id": optionalCWReviewInstructionVersionID(
				currentVersionID,
			),
			"versions": versions,
			"total":    len(versions),
		},
	)
}

func (h *CoursewareAIReviewHandler) getCurrentReviewItemInstructionVersion(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(
			w,
			"课件AI审核执行器未初始化",
		)
		return
	}

	version, err :=
		h.runner.GetCurrentCWReviewItemInstructionVersion(
			r.Context(),
			itemID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	currentVersionID := ""
	if version != nil {
		currentVersionID = version.ID
	}

	utils.Success(
		w,
		map[string]interface{}{
			"current_instruction_version_id": optionalCWReviewInstructionVersionID(
				currentVersionID,
			),
			"version": buildCoursewareAIReviewInstructionVersionView(
				version,
				currentVersionID,
			),
		},
	)
}

func (h *CoursewareAIReviewHandler) getReviewItemInstructionVersion(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	versionID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(
			w,
			"课件AI审核执行器未初始化",
		)
		return
	}

	version, err :=
		h.runner.GetCWReviewItemInstructionVersion(
			r.Context(),
			itemID,
			versionID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"version": buildCoursewareAIReviewInstructionVersionView(
				version,
				"",
			),
		},
	)
}

func (h *CoursewareAIReviewHandler) confirmReviewItemInstructionVersion(
	w http.ResponseWriter,
	r *http.Request,
	itemID string,
	actor *services.CoursewareActorContext,
) {
	if h == nil || h.runner == nil {
		utils.InternalError(
			w,
			"课件AI审核执行器未初始化",
		)
		return
	}

	var req confirmCWReviewInstructionVersionRequest

	if !decodeCWReviewInstructionVersionRequest(
		w,
		r,
		&req,
	) {
		return
	}

	result, err :=
		h.runner.ConfirmCWReviewItemInstructionVersion(
			r.Context(),
			itemID,
			req.Instruction,
			req.ExpectedCurrentVersionID,
			actor,
		)
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.Success(
		w,
		buildCoursewareAIReviewItemDiscussionView(
			result,
		),
	)
}

func decodeCWReviewInstructionVersionRequest(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
) bool {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			coursewareAIReviewItemBodyMaxBytes,
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

	return true
}
