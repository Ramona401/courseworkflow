package handlers

// video_edit_handler.go — 课件视频与音频FFmpeg处理器
//
// 正式路由：
//   POST /api/v1/coursewares/{id}/videos/concat
//   POST /api/v1/coursewares/{id}/videos/trim
//   POST /api/v1/coursewares/{id}/videos/advanced-concat
//   POST /api/v1/coursewares/{id}/videos/mute
//   POST /api/v1/coursewares/{id}/videos/extract-audio
//   POST /api/v1/coursewares/{id}/videos/mix-narration
//   POST /api/v1/coursewares/{id}/videos/trim-audio
//
// Handler先执行作者控制预检，再解析受限JSON正文；Service仍会重新
// 加载正式课件，并在FFmpeg执行前后和资产写库前再次授权。

import (
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// VideoEditHandler 视频与音频编辑处理器。
type VideoEditHandler struct {
	editService *services.VideoEditService
}

// NewVideoEditHandler 创建视频编辑处理器。
func NewVideoEditHandler(
	editService *services.VideoEditService,
) *VideoEditHandler {
	return &VideoEditHandler{
		editService: editService,
	}
}

// ConcatVideos 顺序拼接多个视频。
func (h *VideoEditHandler) ConcatVideos(
	w http.ResponseWriter,
	r *http.Request,
) {
	coursewareID, ok :=
		h.prepareVideoEditRequest(
			w,
			r,
			"/videos/concat",
		)
	if !ok {
		return
	}

	actor, ok :=
		h.preflightVideoEditOwner(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var body struct {
		AssetIDs []string `json:"asset_ids"`
	}

	if err := decodeVideoEditJSON(
		w,
		r,
		&body,
	); err != nil {
		writeVideoEditDecodeError(w, err)
		return
	}

	response, err :=
		h.editService.ConcatVideos(
			r.Context(),
			&services.ConcatVideosRequest{
				CoursewareID: coursewareID,
				AssetIDs:     body.AssetIDs,
				Actor:        actor,
			},
		)
	if err != nil {
		writeVideoEditError(w, err)
		return
	}

	utils.Success(w, response)
}

// TrimVideo 裁剪单个视频。
func (h *VideoEditHandler) TrimVideo(
	w http.ResponseWriter,
	r *http.Request,
) {
	coursewareID, ok :=
		h.prepareVideoEditRequest(
			w,
			r,
			"/videos/trim",
		)
	if !ok {
		return
	}

	actor, ok :=
		h.preflightVideoEditOwner(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var body struct {
		AssetID  string  `json:"asset_id"`
		StartSec float64 `json:"start_sec"`
		EndSec   float64 `json:"end_sec"`
	}

	if err := decodeVideoEditJSON(
		w,
		r,
		&body,
	); err != nil {
		writeVideoEditDecodeError(w, err)
		return
	}

	response, err :=
		h.editService.TrimVideo(
			r.Context(),
			&services.TrimVideoRequest{
				CoursewareID: coursewareID,
				AssetID:      body.AssetID,
				StartSec:     body.StartSec,
				EndSec:       body.EndSec,
				Actor:        actor,
			},
		)
	if err != nil {
		writeVideoEditError(w, err)
		return
	}

	utils.Success(w, response)
}

// AdvancedConcat 执行独立裁剪与转场拼接。
func (h *VideoEditHandler) AdvancedConcat(
	w http.ResponseWriter,
	r *http.Request,
) {
	coursewareID, ok :=
		h.prepareVideoEditRequest(
			w,
			r,
			"/videos/advanced-concat",
		)
	if !ok {
		return
	}

	actor, ok :=
		h.preflightVideoEditOwner(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var body struct {
		Clips []services.VideoClip `json:"clips"`
	}

	if err := decodeVideoEditJSON(
		w,
		r,
		&body,
	); err != nil {
		writeVideoEditDecodeError(w, err)
		return
	}

	response, err :=
		h.editService.AdvancedConcat(
			r.Context(),
			&services.AdvancedConcatRequest{
				CoursewareID: coursewareID,
				Clips:        body.Clips,
				Actor:        actor,
			},
		)
	if err != nil {
		writeVideoEditError(w, err)
		return
	}

	utils.Success(w, response)
}

// MuteVideo 为视频替换静默音轨。
func (h *VideoEditHandler) MuteVideo(
	w http.ResponseWriter,
	r *http.Request,
) {
	coursewareID, actor, assetID, ok :=
		h.prepareSingleAssetVideoEdit(
			w,
			r,
			"/videos/mute",
		)
	if !ok {
		return
	}

	response, err :=
		h.editService.MuteVideo(
			r.Context(),
			&services.MuteVideoRequest{
				CoursewareID: coursewareID,
				AssetID:      assetID,
				Actor:        actor,
			},
		)
	if err != nil {
		writeVideoEditError(w, err)
		return
	}

	utils.Success(w, response)
}

// ExtractAudio 从视频中提取MP3音轨。
func (h *VideoEditHandler) ExtractAudio(
	w http.ResponseWriter,
	r *http.Request,
) {
	coursewareID, actor, assetID, ok :=
		h.prepareSingleAssetVideoEdit(
			w,
			r,
			"/videos/extract-audio",
		)
	if !ok {
		return
	}

	response, err :=
		h.editService.ExtractAudio(
			r.Context(),
			&services.ExtractAudioRequest{
				CoursewareID: coursewareID,
				AssetID:      assetID,
				Actor:        actor,
			},
		)
	if err != nil {
		writeVideoEditError(w, err)
		return
	}

	utils.Success(w, response)
}

// MixNarration 将字幕TTS旁白混入视频。
func (h *VideoEditHandler) MixNarration(
	w http.ResponseWriter,
	r *http.Request,
) {
	coursewareID, ok :=
		h.prepareVideoEditRequest(
			w,
			r,
			"/videos/mix-narration",
		)
	if !ok {
		return
	}

	actor, ok :=
		h.preflightVideoEditOwner(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var body struct {
		AssetID    string  `json:"asset_id"`
		SubtitleID string  `json:"subtitle_id"`
		Gain       float64 `json:"gain"`
	}

	if err := decodeVideoEditJSON(
		w,
		r,
		&body,
	); err != nil {
		writeVideoEditDecodeError(w, err)
		return
	}

	response, err :=
		h.editService.MixNarration(
			r.Context(),
			&services.MixNarrationRequest{
				CoursewareID: coursewareID,
				AssetID:      body.AssetID,
				SubtitleID:   body.SubtitleID,
				Gain:         body.Gain,
				Actor:        actor,
			},
		)
	if err != nil {
		writeVideoEditError(w, err)
		return
	}

	utils.Success(w, response)
}

// TrimAudio 裁剪课件音频资产。
func (h *VideoEditHandler) TrimAudio(
	w http.ResponseWriter,
	r *http.Request,
) {
	coursewareID, ok :=
		h.prepareVideoEditRequest(
			w,
			r,
			"/videos/trim-audio",
		)
	if !ok {
		return
	}

	actor, ok :=
		h.preflightVideoEditOwner(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var body struct {
		AssetID  string  `json:"asset_id"`
		StartSec float64 `json:"start_sec"`
		EndSec   float64 `json:"end_sec"`
	}

	if err := decodeVideoEditJSON(
		w,
		r,
		&body,
	); err != nil {
		writeVideoEditDecodeError(w, err)
		return
	}

	response, err :=
		h.editService.TrimAudio(
			r.Context(),
			&services.TrimAudioRequest{
				CoursewareID: coursewareID,
				AssetID:      body.AssetID,
				StartSec:     body.StartSec,
				EndSec:       body.EndSec,
				Actor:        actor,
			},
		)
	if err != nil {
		writeVideoEditError(w, err)
		return
	}

	utils.Success(w, response)
}
