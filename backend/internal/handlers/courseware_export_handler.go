package handlers

// courseware_export_handler.go — 课件离线打包下载 HTTP 处理器
//
// 路由：GET /api/v1/coursewares/{id}/export-bundle
// 行为：生成 zip 临时文件 → 以 application/zip 流式返回 → 返回后删除临时文件
// 鉴权：复用现有 authMW（cwMux 已包裹），仅课件归属者可下载
//
// 说明：本方法挂在现有 CoursewareHandler 上（同包另起文件），
//       不改动 NewCoursewareHandler 构造签名与 routes.go 主文件。

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ExportBundle GET /api/v1/coursewares/{id}/export-bundle — 导出课件离线包(zip)
func (h *CoursewareHandler) ExportBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}
	id := extractCoursewareMiddleID(r.URL.Path, "/export-bundle")
	if id == "" {
		utils.BadRequest(w, "缺少课件ID")
		return
	}

	// 生成离线包（临时 zip 文件）
	exportSvc := services.NewCoursewareExportService()
	zipPath, downloadName, err := exportSvc.ExportBundle(r.Context(), id, claims.UserID)
	if err != nil {
		utils.InternalError(w, "生成离线包失败: "+err.Error())
		return
	}
	// 无论成功失败，响应结束后删除临时文件
	defer func() { _ = os.Remove(zipPath) }()

	f, err := os.Open(zipPath)
	if err != nil {
		utils.InternalError(w, "读取离线包失败: "+err.Error())
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		utils.InternalError(w, "读取离线包信息失败: "+err.Error())
		return
	}

	// Content-Disposition：同时给 ASCII 兜底名和 RFC5987 中文名
	encoded := url.PathEscape(downloadName)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"courseware.zip\"; filename*=UTF-8''%s", encoded))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}
