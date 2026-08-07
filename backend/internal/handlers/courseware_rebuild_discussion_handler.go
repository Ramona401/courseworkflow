package handlers

// courseware_rebuild_discussion_handler.go — 课件页面讨论接口。
//
// 现有全页重构讨论接口：
//   POST /api/v1/coursewares/{id}/pages/{num}/rebuild-discussion
//
// 新增页面需求讨论接口：
//   POST /api/v1/coursewares/{id}/pages/new/rebuild-discussion
//
// 两条路径复用现有路由分发中的/rebuild-discussion长后缀匹配，
// 因此不需要继续扩张庞大的routes_courseware.go。
//
// 全页重构action取值：
//   - load：读取当前页面活动讨论；不调用AI。
//   - message：追加老师消息并让AI只讨论方案，不生成HTML、不写页面。
//   - confirm：老师点击独立确认按钮后执行既有全页重构链路。
//   - cancel：取消讨论，不修改页面。
//
// 新增页讨论使用项目原有无状态契约：
//   - 前端提交message、messages、current_plan和insert_at；
//   - 后端返回reply、summary、plan与ready_for_confirmation；
//   - 页面创建和HTML生成仍由前端独立按钮调用现有接口。
//
// 自然语言中的“开始”“确认”均不会直接触发页面执行动作。

import (
        "encoding/json"
        "errors"
        "net/http"
        "strings"

        "tedna/internal/middleware"
        "tedna/internal/repository"
        "tedna/internal/services"
        "tedna/internal/utils"
)

const (
        coursewareRebuildDiscussionBodyMaxBytes =
                13 * 1024 * 1024

        coursewareAddPageDiscussionBodyMaxBytes =
                256 * 1024
)

type coursewareRebuildDiscussionRequest struct {
        Action           string `json:"action"`
        DiscussionID     string `json:"discussion_id"`
        Content          string `json:"content"`
        ReferenceContext string `json:"reference_context"`
        Image            string `json:"image"`
}

// RebuildDiscussion处理全页重构讨论和新增页需求讨论。
func (h *CoursewareGenHandler) RebuildDiscussion(
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
        if !ok || claims == nil {
                utils.Unauthorized(
                        w,
                        "未登录",
                )
                return
        }

        // 特殊路径/pages/new/rebuild-discussion不对应已有页面，
        // 必须在按数字页码解析之前单独处理。
        if coursewareID :=
                extractCWAddPageDiscussionPath(
                        r.URL.Path,
                ); coursewareID != "" {
                h.handleAddPageDiscussion(
                        w,
                        r,
                        coursewareID,
                        claims.UserID,
                        claims.Role,
                )
                return
        }

        coursewareID, pageNumber :=
                extractCWPageActionPath(
                        r.URL.Path,
                        "/rebuild-discussion",
                )
        if coursewareID == "" ||
                pageNumber <= 0 {
                utils.BadRequest(
                        w,
                        "路径格式错误",
                )
                return
        }

        // 图片和参考代码可能较大，必须在读取请求体前完成教研微调授权。
        scopedActor, err :=
                h.authorizeCoursewareRefineForHandler(
                        r.Context(),
                        coursewareID,
                        claims.UserID,
                        claims.Role,
                )
        if err != nil {
                writeCoursewareRefineError(
                        w,
                        err,
                )
                return
        }

        r.Body = http.MaxBytesReader(
                w,
                r.Body,
                coursewareRebuildDiscussionBodyMaxBytes,
        )

        decoder := json.NewDecoder(
                r.Body,
        )
        decoder.DisallowUnknownFields()

        var req coursewareRebuildDiscussionRequest
        if err := decoder.Decode(
                &req,
        ); err != nil {
                utils.BadRequest(
                        w,
                        "请求参数格式错误或内容过大",
                )
                return
        }

        action := strings.ToLower(
                strings.TrimSpace(
                        req.Action,
                ),
        )
        if action == "" {
                action = "message"
        }

        image := strings.TrimSpace(
                req.Image,
        )
        if image != "" {
                if !strings.HasPrefix(
                        image,
                        "data:image/",
                ) {
                        utils.BadRequest(
                                w,
                                "截图格式无效，请直接粘贴图片",
                        )
                        return
                }

                if len(image) >
                        12*1024*1024 {
                        utils.BadRequest(
                                w,
                                "截图过大，请压缩后重试（建议不超过8MB）",
                        )
                        return
                }
        }

        service :=
                services.NewCoursewareRebuildDiscussionService(
                        h.genService,
                )

        switch action {
        case "load":
                discussion, loadErr :=
                        service.Load(
                                r.Context(),
                                coursewareID,
                                scopedActor,
                                pageNumber,
                        )
                if loadErr != nil {
                        writeCoursewareRebuildDiscussionError(
                                w,
                                loadErr,
                        )
                        return
                }

                utils.Success(
                        w,
                        map[string]interface{}{
                                "discussion":
                                        discussion,
                        },
                )

        case "message":
                discussion, messageErr :=
                        service.Message(
                                r.Context(),
                                coursewareID,
                                scopedActor,
                                pageNumber,
                                strings.TrimSpace(
                                        req.DiscussionID,
                                ),
                                req.Content,
                                req.ReferenceContext,
                                image,
                        )
                if messageErr != nil {
                        writeCoursewareRebuildDiscussionError(
                                w,
                                messageErr,
                        )
                        return
                }

                utils.Success(
                        w,
                        map[string]interface{}{
                                "discussion":
                                        discussion,
                                "message":
                                        "AI已回复讨论方案，尚未修改页面",
                        },
                )

        case "confirm":
                result, confirmErr :=
                        service.Confirm(
                                r.Context(),
                                coursewareID,
                                scopedActor,
                                pageNumber,
                                strings.TrimSpace(
                                        req.DiscussionID,
                                ),
                        )
                if confirmErr != nil {
                        writeCoursewareRebuildDiscussionError(
                                w,
                                confirmErr,
                        )
                        return
                }

                utils.Success(
                        w,
                        result,
                )

        case "cancel":
                discussion, cancelErr :=
                        service.Cancel(
                                r.Context(),
                                coursewareID,
                                scopedActor,
                                pageNumber,
                                strings.TrimSpace(
                                        req.DiscussionID,
                                ),
                        )
                if cancelErr != nil {
                        writeCoursewareRebuildDiscussionError(
                                w,
                                cancelErr,
                        )
                        return
                }

                utils.Success(
                        w,
                        map[string]interface{}{
                                "discussion":
                                        discussion,
                                "message":
                                        "讨论已取消，页面未发生修改",
                        },
                )

        default:
                utils.BadRequest(
                        w,
                        "action无效，仅支持load、message、confirm或cancel",
                )
        }
}

// handleAddPageDiscussion处理新增页面前的无状态需求讨论。
func (h *CoursewareGenHandler) handleAddPageDiscussion(
        w http.ResponseWriter,
        r *http.Request,
        coursewareID string,
        userID string,
        role string,
) {
        // 先执行作者运行域预检，再读取对话正文。
        scopedActor, err :=
                h.authorizeCoursewareOwnerRuntime(
                        r.Context(),
                        coursewareID,
                        userID,
                        role,
                )
        if err != nil {
                writeCoursewareControlError(
                        w,
                        err,
                )
                return
        }

        r.Body = http.MaxBytesReader(
                w,
                r.Body,
                coursewareAddPageDiscussionBodyMaxBytes,
        )

        decoder := json.NewDecoder(
                r.Body,
        )
        decoder.DisallowUnknownFields()

        var req services.CoursewareAddPageDiscussionRequest
        if err := decoder.Decode(
                &req,
        ); err != nil {
                utils.BadRequest(
                        w,
                        "请求参数格式错误或内容过大",
                )
                return
        }

        service :=
                services.NewCoursewareAddPageDiscussionService(
                        h.genService,
                )

        result, err := service.Discuss(
                r.Context(),
                coursewareID,
                scopedActor,
                &req,
        )
        if err != nil {
                writeCoursewareAddPageDiscussionError(
                        w,
                        err,
                )
                return
        }

        utils.Success(
                w,
               	map[string]interface{}{
                        "discussion":
                                result,
                        "message":
                                "AI已回复新增页方案，尚未创建页面",
                },
        )
}

// extractCWAddPageDiscussionPath解析新增页需求讨论特殊路径。
//
// 有效路径：
//   /api/v1/coursewares/{id}/pages/new/rebuild-discussion
func extractCWAddPageDiscussionPath(
        path string,
) string {
        const prefix =
                "/api/v1/coursewares/"
        const suffix =
                "/pages/new/rebuild-discussion"

        normalized := strings.TrimSuffix(
                path,
                "/",
        )

        if !strings.HasPrefix(
                normalized,
                prefix,
        ) ||
                !strings.HasSuffix(
                        normalized,
                        suffix,
                ) {
                return ""
        }

        coursewareID :=
                strings.TrimSuffix(
                        strings.TrimPrefix(
                                normalized,
                                prefix,
                        ),
                        suffix,
                )

        if coursewareID == "" ||
                strings.Contains(
                        coursewareID,
                        "/",
                ) {
                return ""
        }

        return coursewareID
}

// writeCoursewareAddPageDiscussionError映射新增页讨论的稳定业务错误。
func writeCoursewareAddPageDiscussionError(
        w http.ResponseWriter,
        err error,
) {
        switch {
        case errors.Is(
                err,
                services.ErrCWAddPageDiscussionInvalidRequest,
        ),
                errors.Is(
                        err,
                        services.ErrCWAddPageDiscussionTooLong,
                ):
                utils.BadRequest(
                        w,
                        err.Error(),
                )

        default:
                writeCoursewareControlError(
                        w,
                        err,
                )
        }
}

func writeCoursewareRebuildDiscussionError(
        w http.ResponseWriter,
        err error,
) {
        switch {
        case errors.Is(
                err,
                repository.ErrCWRebuildDiscussionNotFound,
        ):
                utils.Fail(
                        w,
                        http.StatusNotFound,
                        "重构讨论不存在",
                )

        case errors.Is(
                err,
                repository.ErrCWRebuildDiscussionConflict,
        ),
                errors.Is(
                        err,
                        services.ErrCWRebuildDiscussionStale,
                ),
                errors.Is(
                        err,
                        services.ErrCWRebuildDiscussionNotReady,
                ),
                errors.Is(
                        err,
                        services.ErrCWRebuildDiscussionBusy,
                ),
                errors.Is(
                        err,
                        services.ErrCWRebuildDiscussionReferenceChanged,
                ),
                errors.Is(
                        err,
                        services.ErrCWRebuildDiscussionTooManyMessages,
                ):
                utils.Fail(
                        w,
                        http.StatusConflict,
                        err.Error(),
                )

        case errors.Is(
                err,
                services.ErrCoursewareAccessNotFound,
        ),
                errors.Is(
                        err,
                        services.ErrCoursewarePageNotFound,
                ):
                utils.Fail(
                        w,
                        http.StatusNotFound,
                        err.Error(),
                )

        case errors.Is(
                err,
                services.ErrCoursewareActorRequired,
        ),
                errors.Is(
                        err,
                        services.ErrCoursewareEducationDomainMismatch,
                ):
                utils.Fail(
                        w,
                        http.StatusForbidden,
                        err.Error(),
                )

        default:
                writeCoursewareRefineError(
                        w,
                        err,
                )
        }
}
