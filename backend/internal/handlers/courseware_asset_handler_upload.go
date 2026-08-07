package handlers

// courseware_asset_handler_upload.go — 课件视频和音频手动上传接口

import (
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// UploadVideo 手动上传视频文件。
func (h *CoursewareAssetHandler) UploadVideo(
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

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/upload-video",
		)
	if coursewareID == "" ||
		pageNumber <= 0 {
		utils.BadRequest(
			w,
			"路径参数错误",
		)
		return
	}

	// 大体积视频必须先授权，再解析multipart。
	actor, allowed :=
		requireCoursewareAssetOwnerActor(
			w,
			r,
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if !allowed {
		return
	}

	if err :=
		r.ParseMultipartForm(
			10 << 20,
		); err != nil {
		utils.BadRequest(
			w,
			"视频文件解析失败: "+
				err.Error(),
		)
		return
	}

	file, header, err :=
		r.FormFile("file")
	if err != nil {
		utils.BadRequest(
			w,
			"缺少文件字段 file",
		)
		return
	}
	defer file.Close()

	originalFilename :=
		header.Filename

	response, err :=
		h.assetService.UploadVideoAsset(
			r.Context(),
			&services.UploadVideoAssetRequest{
				CoursewareID: coursewareID,
				PageNumber:   pageNumber,
				Actor:        actor,
			},
			file,
			header,
		)
	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	repository.WriteAuditLog(
		claims.UserID,
		"courseware.video_upload",
		map[string]interface{}{
			"courseware_id":     coursewareID,
			"page_number":       pageNumber,
			"asset_id":          response.AssetID,
			"file_size":         response.FileSize,
			"mime_type":         response.MimeType,
			"original_filename": originalFilename,
			"stored_filename":   response.FileName,
		},
		repository.GetClientIP(
			r.RemoteAddr,
		),
	)

	utils.Success(w, response)
}

// UploadAudio 手动上传音频文件。
func (h *CoursewareAssetHandler) UploadAudio(
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

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	coursewareID, pageNumber :=
		extractCWAssetPageActionPath(
			r.URL.Path,
			"/upload-audio",
		)
	if coursewareID == "" ||
		pageNumber <= 0 {
		utils.BadRequest(
			w,
			"路径参数错误",
		)
		return
	}

	// 音频也必须先授权，再解析multipart。
	actor, allowed :=
		requireCoursewareAssetOwnerActor(
			w,
			r,
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if !allowed {
		return
	}

	if err :=
		r.ParseMultipartForm(
			10 << 20,
		); err != nil {
		utils.BadRequest(
			w,
			"音频文件解析失败: "+
				err.Error(),
		)
		return
	}

	file, header, err :=
		r.FormFile("file")
	if err != nil {
		utils.BadRequest(
			w,
			"缺少文件字段 file",
		)
		return
	}
	defer file.Close()

	originalFilename :=
		header.Filename

	response, err :=
		h.assetService.UploadAudioAsset(
			r.Context(),
			&services.UploadAudioAssetRequest{
				CoursewareID: coursewareID,
				PageNumber:   pageNumber,
				Actor:        actor,
			},
			file,
			header,
		)
	if err != nil {
		handleCoursewareAssetServiceError(
			w,
			err,
		)
		return
	}

	repository.WriteAuditLog(
		claims.UserID,
		"courseware.audio_upload",
		map[string]interface{}{
			"courseware_id":     coursewareID,
			"page_number":       pageNumber,
			"asset_id":          response.AssetID,
			"file_size":         response.FileSize,
			"mime_type":         response.MimeType,
			"original_filename": originalFilename,
			"stored_filename":   response.FileName,
		},
		repository.GetClientIP(
			r.RemoteAddr,
		),
	)

	utils.Success(w, response)
}
