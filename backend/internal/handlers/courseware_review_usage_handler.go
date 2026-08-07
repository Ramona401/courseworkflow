package handlers

// courseware_review_usage_handler.go
//
// 课件评审体验的最小使用事件入口。
//
// 隐私与安全边界：
//   - 身份只取JWT，不接受客户端提交用户、角色、学校或教育域；
//   - event、mode、shortcut均使用固定白名单；
//   - 只接受数量、页码和筛选数量，不接受搜索词、问题正文、整改要求或教案内容；
//   - 请求体限制为4KB并拒绝未知字段；
//   - 审计写入沿用best-effort机制，失败不阻断评审主业务。

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

const (
	maxCoursewareReviewUsageBodyBytes = 4 << 10
	maxCoursewareReviewUsageCount     = 10000
	maxCoursewareReviewUsagePage      = 10000
	maxCoursewareReviewFilterCount    = 8
)

var allowedCoursewareReviewUsageEvents = map[string]struct{}{
	"filter_applied":     {},
	"link_copied":        {},
	"deep_link_restored": {},
	"workspace_opened":   {},
	"keyboard_shortcut":  {},
	"lesson_plan_opened": {},
	"auto_advanced":      {},
}

var allowedCoursewareReviewUsageModes = map[string]struct{}{
	"formal":      {},
	"self":        {},
	"remediation": {},
}

var allowedCoursewareReviewUsageShortcuts = map[string]struct{}{
	"focus_search":     {},
	"open_next_task":   {},
	"previous_item":    {},
	"next_item":        {},
	"back_to_list":     {},
	"open_lesson_plan": {},
}

type coursewareReviewUsageEventRequest struct {
	Event             string `json:"event"`
	Mode              string `json:"mode"`
	Shortcut          string `json:"shortcut,omitempty"`
	TotalCount        int    `json:"total_count,omitempty"`
	VisibleCount      int    `json:"visible_count,omitempty"`
	ActiveFilterCount int    `json:"active_filter_count,omitempty"`
	PageNumber        int    `json:"page_number,omitempty"`
}

// CoursewareReviewUsageHandler 记录课件评审体验的白名单使用事件。
type CoursewareReviewUsageHandler struct{}

// NewCoursewareReviewUsageHandler 创建无状态处理器。
func NewCoursewareReviewUsageHandler() *CoursewareReviewUsageHandler {
	return &CoursewareReviewUsageHandler{}
}

// Record POST /api/v1/courseware-review-usage。
func (h *CoursewareReviewUsageHandler) Record(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil ||
		strings.TrimSpace(claims.UserID) == "" {
		utils.Unauthorized(w, "未登录")
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxCoursewareReviewUsageBodyBytes,
	)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req coursewareReviewUsageEventRequest
	if err := decoder.Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"使用事件参数格式错误",
		)
		return
	}

	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		utils.BadRequest(
			w,
			"请求体只能包含一个JSON对象",
		)
		return
	}

	normalizeCoursewareReviewUsageRequest(
		&req,
	)

	if err := validateCoursewareReviewUsageRequest(
		&req,
	); err != nil {
		utils.BadRequest(
			w,
			err.Error(),
		)
		return
	}

	detail := map[string]interface{}{
		"schema_version":      1,
		"event":               req.Event,
		"mode":                req.Mode,
		"total_count":         req.TotalCount,
		"visible_count":       req.VisibleCount,
		"active_filter_count": req.ActiveFilterCount,
		"page_number":         req.PageNumber,
	}
	if req.Shortcut != "" {
		detail["shortcut"] =
			req.Shortcut
	}

	repository.WriteAuditLog(
		claims.UserID,
		repository.ActionCoursewareReviewUsage,
		detail,
		repository.GetClientIP(
			r.RemoteAddr,
		),
	)

	utils.Success(
		w,
		map[string]bool{
			"accepted": true,
		},
	)
}

func normalizeCoursewareReviewUsageRequest(
	req *coursewareReviewUsageEventRequest,
) {
	if req == nil {
		return
	}

	req.Event = strings.ToLower(
		strings.TrimSpace(
			req.Event,
		),
	)
	req.Mode = strings.ToLower(
		strings.TrimSpace(
			req.Mode,
		),
	)
	req.Shortcut = strings.ToLower(
		strings.TrimSpace(
			req.Shortcut,
		),
	)
}

func validateCoursewareReviewUsageRequest(
	req *coursewareReviewUsageEventRequest,
) error {
	if req == nil {
		return errors.New(
			"使用事件请求不能为空",
		)
	}

	if _, ok :=
		allowedCoursewareReviewUsageEvents[req.Event]; !ok {
		return errors.New(
			"不支持的使用事件",
		)
	}

	if _, ok :=
		allowedCoursewareReviewUsageModes[req.Mode]; !ok {
		return errors.New(
			"不支持的评审模式",
		)
	}

	if req.TotalCount < 0 ||
		req.TotalCount >
			maxCoursewareReviewUsageCount {
		return errors.New(
			"问题总数超出允许范围",
		)
	}

	if req.VisibleCount < 0 ||
		req.VisibleCount >
			maxCoursewareReviewUsageCount {
		return errors.New(
			"可见问题数量超出允许范围",
		)
	}

	if req.VisibleCount >
		req.TotalCount {
		return errors.New(
			"可见问题数量不能大于问题总数",
		)
	}

	if req.ActiveFilterCount < 0 ||
		req.ActiveFilterCount >
			maxCoursewareReviewFilterCount {
		return errors.New(
			"筛选条件数量超出允许范围",
		)
	}

	if req.PageNumber < 0 ||
		req.PageNumber >
			maxCoursewareReviewUsagePage {
		return errors.New(
			"页码超出允许范围",
		)
	}

	if req.Event ==
		"keyboard_shortcut" {
		if _, ok :=
			allowedCoursewareReviewUsageShortcuts[req.Shortcut]; !ok {
			return errors.New(
				"不支持的快捷键事件",
			)
		}
	} else if req.Shortcut != "" {
		return errors.New(
			"非快捷键事件不能携带shortcut",
		)
	}

	return nil
}
