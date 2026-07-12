package handlers

// lesson_plan_ref_handler.go — 备课参考资料附件(PDF/Word)压缩 HTTP 处理器
//
// 单端点：POST /api/v1/lesson-plans/ref-material/compress
//   前端在浏览器端提取出的长参考资料原文(≥3000字)POST 到此，后端用 AI 压成结构化要点返回。
//   短文档(<3000字)前端不调此端点，直接把原文作为 ref_material 注入，省一次 AI 调用。
//   压缩结果不落库——前端拿到后在内存持有，每轮 chat 携带(会话级、用完即走)。

import (
        "encoding/json"
        "net/http"
        "strings"

        "tedna/internal/logger"
        "tedna/internal/models"
        "tedna/internal/services"
        "tedna/internal/utils"
)

// LessonPlanRefHandler 参考资料压缩处理器
type LessonPlanRefHandler struct {
        refService *services.LessonPlanRefService
}

var lpRefHandlerLog = logger.WithModule("lp_ref_handler")

// NewLessonPlanRefHandler 创建参考资料压缩处理器
func NewLessonPlanRefHandler(refService *services.LessonPlanRefService) *LessonPlanRefHandler {
        return &LessonPlanRefHandler{refService: refService}
}

// CompressRefMaterial 处理 POST /ref-material/compress
func (h *LessonPlanRefHandler) CompressRefMaterial(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
                return
        }
        userID := getCurrentUserID(r)
        if userID == "" {
                utils.Unauthorized(w, utils.MsgNotLoggedIn)
                return
        }

        var req models.CompressRefMaterialRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.BadRequest(w, utils.MsgBadRequestBody)
                return
        }
        if strings.TrimSpace(req.Content) == "" {
                utils.BadRequest(w, "参考资料内容为空")
                return
        }

        compressed, origLen, compressedLen, err := h.refService.CompressRefMaterial(
                r.Context(), userID, req.Content, req.FileName, req.Subject, req.Grade,
        )
        if err != nil {
                lpRefHandlerLog.Error("参考资料压缩失败", "user", userID, "file", req.FileName, "error", err)
                utils.InternalError(w, "参考资料压缩失败，请稍后重试")
                return
        }

        utils.Success(w, &models.CompressRefMaterialResponse{
                Compressed:    compressed,
                OriginalLen:   origLen,
                CompressedLen: compressedLen,
        })
}
