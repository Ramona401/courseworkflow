package handlers

// lesson_plan_ref_handler.go — 备课参考资料附件 HTTP 处理器
//
// 保持既有路由不变：POST /api/v1/lesson-plans/ref-material/compress
//
// 请求 mode：
//   - compress_text（默认）：压缩长文本；
//   - vision_transcribe：忠实转录扫描 PDF 单页图片。
//
// 页面图仅随当前请求传输，不落盘、不落库。

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	refMaterialModeCompress = "compress_text"
	refMaterialModeVision   = "vision_transcribe"

	refMaterialRequestMaxBytes     int64 = 16 * 1024 * 1024
	refMaterialImageBase64MaxChars       = 14 * 1024 * 1024
	refMaterialVisionMaxPages            = 12
)

type lessonPlanRefProcessRequest struct {
	Mode         string `json:"mode"`
	Content      string `json:"content"`
	FileName     string `json:"file_name"`
	Subject      string `json:"subject"`
	Grade        string `json:"grade"`
	ImageDataURI string `json:"image_data_uri"`
	PageNumber   int    `json:"page_number"`
	TotalPages   int    `json:"total_pages"`
}

// LessonPlanRefHandler 参考资料处理器。
type LessonPlanRefHandler struct {
	refService *services.LessonPlanRefService
}

var lpRefHandlerLog = logger.WithModule("lp_ref_handler")

// NewLessonPlanRefHandler 创建参考资料处理器。
func NewLessonPlanRefHandler(
	refService *services.LessonPlanRefService,
) *LessonPlanRefHandler {
	return &LessonPlanRefHandler{refService: refService}
}

// CompressRefMaterial 处理参考资料文本压缩或扫描页转录。
func (h *LessonPlanRefHandler) CompressRefMaterial(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		utils.Fail(
			writer,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	userID := getCurrentUserID(request)
	if userID == "" {
		utils.Unauthorized(writer, utils.MsgNotLoggedIn)
		return
	}

	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		refMaterialRequestMaxBytes,
	)

	var payload lessonPlanRefProcessRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		lpRefHandlerLog.Warn(
			"参考资料请求体解析失败",
			"user", userID,
			"error", err,
		)
		utils.BadRequest(
			writer,
			"参考资料请求无效或数据过大",
		)
		return
	}

	mode := strings.ToLower(strings.TrimSpace(payload.Mode))
	if mode == "" {
		mode = refMaterialModeCompress
	}

	switch mode {
	case refMaterialModeCompress:
		h.handleCompressText(writer, request, userID, &payload)
	case refMaterialModeVision:
		h.handleVisionTranscribe(writer, request, userID, &payload)
	default:
		utils.BadRequest(writer, "不支持的参考资料处理模式")
	}
}

func (h *LessonPlanRefHandler) handleCompressText(
	writer http.ResponseWriter,
	request *http.Request,
	userID string,
	payload *lessonPlanRefProcessRequest,
) {
	if payload == nil || strings.TrimSpace(payload.Content) == "" {
		utils.BadRequest(writer, "参考资料内容为空")
		return
	}

	compressed, originalLen, compressedLen, err :=
		h.refService.CompressRefMaterial(
			request.Context(),
			userID,
			payload.Content,
			payload.FileName,
			payload.Subject,
			payload.Grade,
		)
	if err != nil {
		lpRefHandlerLog.Error(
			"参考资料压缩失败",
			"user", userID,
			"file", payload.FileName,
			"error", err,
		)
		utils.InternalError(
			writer,
			"参考资料压缩失败，请稍后重试",
		)
		return
	}

	utils.Success(writer, &models.CompressRefMaterialResponse{
		Compressed:    compressed,
		OriginalLen:   originalLen,
		CompressedLen: compressedLen,
	})
}

func (h *LessonPlanRefHandler) handleVisionTranscribe(
	writer http.ResponseWriter,
	request *http.Request,
	userID string,
	payload *lessonPlanRefProcessRequest,
) {
	if payload == nil {
		utils.BadRequest(writer, "扫描页请求为空")
		return
	}
	if payload.TotalPages <= 0 ||
		payload.TotalPages > refMaterialVisionMaxPages ||
		payload.PageNumber <= 0 ||
		payload.PageNumber > payload.TotalPages {
		utils.BadRequest(writer, fmt.Sprintf(
			"扫描PDF页码无效，当前最多支持%d页",
			refMaterialVisionMaxPages,
		))
		return
	}

	imageDataURI, err := validateRefMaterialImageDataURI(
		payload.ImageDataURI,
	)
	if err != nil {
		utils.BadRequest(writer, err.Error())
		return
	}

	text, err := h.refService.TranscribeRefMaterialPage(
		request.Context(),
		userID,
		imageDataURI,
		payload.FileName,
		payload.PageNumber,
		payload.TotalPages,
		payload.Subject,
		payload.Grade,
	)
	if err != nil {
		lpRefHandlerLog.Error(
			"参考资料扫描页转录失败",
			"user", userID,
			"file", payload.FileName,
			"page", payload.PageNumber,
			"total_pages", payload.TotalPages,
			"error", err,
		)
		utils.InternalError(writer, fmt.Sprintf(
			"PDF第%d页识别失败，请稍后重试",
			payload.PageNumber,
		))
		return
	}

	utils.Success(writer, map[string]interface{}{
		"text":        text,
		"page_number": payload.PageNumber,
		"total_pages": payload.TotalPages,
	})
}

func validateRefMaterialImageDataURI(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	allowedPrefix := ""

	for _, prefix := range []string{
		"data:image/jpeg;base64,",
		"data:image/jpg;base64,",
		"data:image/png;base64,",
		"data:image/webp;base64,",
	} {
		if strings.HasPrefix(raw, prefix) {
			allowedPrefix = prefix
			break
		}
	}

	if allowedPrefix == "" {
		return "", fmt.Errorf(
			"扫描页图片格式无效，仅支持JPEG/PNG/WEBP",
		)
	}

	encoded := strings.TrimPrefix(raw, allowedPrefix)
	if encoded == "" {
		return "", fmt.Errorf("扫描页图片内容为空")
	}
	if len(encoded) > refMaterialImageBase64MaxChars {
		return "", fmt.Errorf(
			"扫描页图片数据过大，请降低PDF页面清晰度后重试",
		)
	}

	decoder := base64.NewDecoder(
		base64.StdEncoding,
		strings.NewReader(encoded),
	)
	if _, err := io.Copy(io.Discard, decoder); err != nil {
		return "", fmt.Errorf("扫描页图片编码无效")
	}

	return raw, nil
}
