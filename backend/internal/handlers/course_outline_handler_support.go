package handlers

// course_outline_handler_support.go — 双模式候选、响应组装和错误映射
//
// GET /api/v1/course-outlines/candidates 查询参数：
//   - subject：必填，当前课程或学科；
//   - grade：必填，当前具体年级或学习层级；
//   - mode：exact或manual，缺省为exact。
//
// exact用于自动匹配，只返回具体年级完全相等候选；
// manual用于自动匹配失败后的教师手动选择，返回年级或学段相交候选。

import (
        "errors"
        "net/http"
        "strings"

        "tedna/internal/models"
        "tedna/internal/repository"
        "tedna/internal/services"
        "tedna/internal/utils"
)

const (
        courseOutlineCandidateModeExact  = "exact"
        courseOutlineCandidateModeManual = "manual"
)

func (h *CourseOutlineHandler) listExactCandidates(
        w http.ResponseWriter,
        r *http.Request,
        userID string,
) {
        subject := strings.TrimSpace(
                r.URL.Query().Get("subject"),
        )

        grade := strings.TrimSpace(
                r.URL.Query().Get("grade"),
        )

        if subject == "" || grade == "" {
                utils.BadRequest(
                        w,
                        "缺少学科或当前年级参数",
                )
                return
        }

        mode := strings.ToLower(
                strings.TrimSpace(
                        r.URL.Query().Get("mode"),
                ),
        )

        if mode == "" {
                mode =
                        courseOutlineCandidateModeExact
        }

        var (
                items  []*models.CourseOutlineListItem
                domain string
                err    error
        )

        switch mode {
        case courseOutlineCandidateModeExact:
                items, domain, err =
                        h.svc.ListExactCandidates(
                                r.Context(),
                                userID,
                                subject,
                                grade,
                        )

        case courseOutlineCandidateModeManual:
                items, domain, err =
                        h.svc.ListManualCandidates(
                                r.Context(),
                                userID,
                                subject,
                                grade,
                        )

        default:
                utils.BadRequest(
                        w,
                        "课程大纲候选模式仅支持exact或manual",
                )
                return
        }

        if err != nil {
                h.mapError(w, err)
                return
        }

        responseItems := make(
                []map[string]interface{},
                0,
                len(items),
        )

        for _, item := range items {
                responseItems = append(
                        responseItems,
                        courseOutlineListItemResponse(
                                item,
                                domain,
                        ),
                )
        }

        utils.Success(
                w,
                map[string]interface{}{
                        "candidates": responseItems,
                        "total":      len(responseItems),
                        "match_mode": mode,
                },
        )
}

func courseOutlineListItemResponse(
        item *models.CourseOutlineListItem,
        educationDomain string,
) map[string]interface{} {
        if item == nil {
                return map[string]interface{}{}
        }

        response := map[string]interface{}{
                "id":              item.ID,
                "scope":           item.Scope,
                "scope_target_id": item.ScopeTargetID,
                "scope_name":      item.ScopeName,
                "subject":         item.Subject,
                "grade":           item.Grade,
                "volume":          item.Volume,
                "title":           item.Title,
                "creator_name":    item.CreatorName,
                "updated_at":      item.UpdatedAt,
        }

        if educationDomain ==
                models.EducationDomainK12 {
                response["publisher"] =
                        item.Publisher
                response["school_system"] =
                        item.SchoolSystem
        }

        return response
}

func courseOutlineDetailResponse(
        outline *models.CourseOutline,
        educationDomain string,
) map[string]interface{} {
        if outline == nil {
                return map[string]interface{}{}
        }

        response := map[string]interface{}{
                "id":               outline.ID,
                "scope":            outline.Scope,
                "scope_target_id":  outline.ScopeTargetID,
                "subject":          outline.Subject,
                "grade":            outline.Grade,
                "volume":           outline.Volume,
                "title":            outline.Title,
                "content":          outline.Content,
                "source_file_path": outline.SourceFilePath,
                "source_type":      outline.SourceType,
                "created_by":       outline.CreatedBy,
                "status":           outline.Status,
                "created_at":       outline.CreatedAt,
                "updated_at":       outline.UpdatedAt,
        }

        if educationDomain ==
                models.EducationDomainK12 {
                response["publisher"] =
                        outline.Publisher
                response["school_system"] =
                        outline.SchoolSystem
        }

        return response
}

func (h *CourseOutlineHandler) mapError(
        w http.ResponseWriter,
        err error,
) {
        switch {
        case errors.Is(
                err,
                services.ErrOutlineFieldRequired,
        ),
                errors.Is(
                        err,
                        services.ErrOutlineScopeInvalid,
                ),
                errors.Is(
                        err,
                        services.ErrOutlinePublisherNotAllowed,
                ),
                errors.Is(
                        err,
                        services.ErrOutlinePublisherUnavailable,
                ),
                errors.Is(
                        err,
                        services.ErrOutlineSchoolSystemInvalid,
                ):
                utils.BadRequest(
                        w,
                        err.Error(),
                )

        case errors.Is(
                err,
                services.ErrOutlineNoPermission,
        ),
                errors.Is(
                        err,
                        services.ErrOutlineEducationDomainRequired,
                ),
                errors.Is(
                        err,
                        services.ErrOutlineEducationDomainConflict,
                ),
                errors.Is(
                        err,
                        services.ErrOutlineEducationDomainMismatch,
                ):
                utils.Fail(
                        w,
                        http.StatusForbidden,
                        err.Error(),
                )

        case errors.Is(
                err,
                repository.ErrCourseOutlineNotFound,
        ):
                utils.Fail(
                        w,
                        http.StatusNotFound,
                        err.Error(),
                )

        case errors.Is(
                err,
                services.ErrOutlineEducationDomainResolveFailed,
        ):
                utils.InternalError(
                        w,
                        services.
                                ErrOutlineEducationDomainResolveFailed.
                                Error(),
                )

        default:
                utils.InternalError(
                        w,
                        "课程大纲操作失败，请稍后重试",
                )
        }
}
