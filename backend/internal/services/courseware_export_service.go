package services

// courseware_export_service.go — 课件纯离线打包导出服务（编排层）
//
// 功能：把一套课件的所有已生成页面及页面引用的媒体资源整合成ZIP。
// 解压后老师可以断网双击index.html完成课件展示、动画、测验和翻页。
//
// 产品边界：
//   1. 离线ZIP是纯课件交付物，不提供教学智能体运行能力；
//   2. 导出过程不查询教学智能体部署，不写public_id、部署ID、运行令牌或embed地址；
//   3. 教学智能体仅存在于TE-DNA平台登录态预览，以及未来独立的HTTPS在线发布链路；
//   4. 常规课件页html_content是1920×1080片段，由本服务包装为独立文档；
//   5. 3D或导入页可能是完整HTML文档，保留正文并注入离线导航；
//   6. 媒体扫描、下载、相对路径改写和页面包装位于courseware_export_assets.go；
//   7. courseware_export_assistant.go只负责清除历史版本可能残留的旧导出助手片段；
//   8. ZIP临时文件位于非Nginx暴露目录，由Handler返回后立即删除。

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwExportLog 模块级结构化日志器，导出子模块共用。
var cwExportLog = logger.WithModule("courseware_export")

const (
	// cwExportRoot 本地上传文件磁盘根目录。
	cwExportRoot = "/www/wwwroot/tedna"

	// cwExportTmpDir ZIP临时目录，不由Nginx直接暴露。
	cwExportTmpDir = cwExportRoot + "/tmp"

	// cwExportMaxAssetBytes 单个资源大小上限。
	cwExportMaxAssetBytes = 500 * 1024 * 1024

	// cwExportMaxTotalBytes 整个导出包大小上限。
	cwExportMaxTotalBytes = 3 * 1024 * 1024 * 1024
)

// CoursewareExportService 课件离线打包导出服务。
type CoursewareExportService struct{}

// NewCoursewareExportService 创建导出服务。
func NewCoursewareExportService() *CoursewareExportService {
	return &CoursewareExportService{}
}

// ExportBundle 生成纯离线课件包。
//
// 安全边界：
//   - 只能导出当前登录教师自己的课件；
//   - 不查询或读取任何教学智能体部署；
//   - 不把public_id、内部部署ID、教师ID、提示词、上下文、运行令牌或embed地址写入ZIP；
//   - 导出前防御性清除历史TEDNA-ASSISTANT-EXPORT片段；
//   - 清理后若仍检测到在线助手残留，立即停止导出，绝不生成不合规ZIP。
func (s *CoursewareExportService) ExportBundle(
	ctx context.Context,
	coursewareID string,
	userID string,
) (
	string,
	string,
	error,
) {
	// 1. 加载课件并执行归属校验。
	courseware, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", "", fmt.Errorf("课件不存在: %w", err)
	}
	if strings.TrimSpace(courseware.UserID) != strings.TrimSpace(userID) {
		return "", "", fmt.Errorf("无权导出该课件")
	}

	// 2. 加载全部页面，仓储已按page_number升序返回。
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil {
		return "", "", fmt.Errorf("加载页面失败: %w", err)
	}

	validPages := make([]*models.CoursewarePage, 0, len(pages))
	for _, page := range pages {
		if page != nil && strings.TrimSpace(page.HTMLContent) != "" {
			validPages = append(validPages, page)
		}
	}
	if len(validPages) == 0 {
		return "", "", fmt.Errorf("课件尚未生成页面内容，无法导出")
	}

	// 3. 离线ZIP不读取教学智能体部署，直接创建临时文件。
	if err := os.MkdirAll(cwExportTmpDir, 0o755); err != nil {
		return "", "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	tmp, err := os.CreateTemp(cwExportTmpDir, "cw-bundle-*.zip")
	if err != nil {
		return "", "", fmt.Errorf("创建临时文件失败: %w", err)
	}

	tmpPath := tmp.Name()
	zipWriter := zip.NewWriter(tmp)

	rootDir := sanitizeBundleName(courseware.Title)
	if rootDir == "" {
		rootDir = "courseware"
	}

	bundle := &cwExportBundle{
		zw:         zipWriter,
		rootDir:    rootDir,
		assetCache: make(map[string]string),
	}

	failClose := func(cause error) (string, string, error) {
		_ = zipWriter.Close()
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", "", cause
	}

	totalPages := len(validPages)

	// 4. 逐页清理历史助手痕迹、改写资源并包装独立离线文档。
	for index, page := range validPages {
		pageNumber := index + 1

		title := strings.TrimSpace(page.Title)
		if title == "" {
			title = fmt.Sprintf("第%d页", pageNumber)
		}

		// 先防御性清除历史导出版本可能留下的助手注入片段。
		//
		// 正常数据库页面不会保存导出时注入内容；此清理用于防止老师把旧导出HTML
		// 再次导入或手工粘贴回平台后，在线助手编号随下一次离线ZIP继续传播。
		sanitized := stripCoursewareExportAssistantArtifacts(page.HTMLContent)
		if containsCoursewareExportAssistantArtifacts(sanitized) {
			return failClose(fmt.Errorf(
				"第%d页仍包含在线教学智能体残留，已停止离线导出",
				pageNumber,
			))
		}

		// 再处理页面原有媒体资源。
		rewritten := bundle.rewriteAssets(sanitized)

		// 再包装为完整独立HTML文档。
		document := buildOfflinePageDoc(
			rewritten,
			pageNumber,
			totalPages,
			title,
			courseware.Title,
		)

		// 包装完成后再次执行边界检查，确保导航和页面包装不会带入任何在线助手内容。
		if containsCoursewareExportAssistantArtifacts(document) {
			return failClose(fmt.Errorf(
				"第%d页离线文档仍包含教学智能体内容，已停止导出",
				pageNumber,
			))
		}

		if err := bundle.writeText(
			fmt.Sprintf("p%d.html", pageNumber),
			document,
		); err != nil {
			return failClose(fmt.Errorf("写入页面失败: %w", err))
		}

		if bundle.totalBytes > cwExportMaxTotalBytes {
			return failClose(fmt.Errorf("课件资源总大小超出上限(3GB)，无法打包"))
		}
	}

	// 5. 写入播放器入口和说明文件。
	if err := bundle.writeText(
		"index.html",
		buildOfflineIndexDoc(courseware.Title, totalPages),
	); err != nil {
		return failClose(fmt.Errorf("写入入口页失败: %w", err))
	}

	if err := bundle.writeText(
		"使用说明.txt",
		buildOfflineReadme(
			courseware.Title,
			totalPages,
		),
	); err != nil {
		return failClose(fmt.Errorf("写入说明失败: %w", err))
	}

	// 6. 收尾。
	if err := zipWriter.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("打包失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	downloadName := rootDir + ".zip"

	cwExportLog.Info(
		"课件离线包生成成功",
		"courseware_id", coursewareID,
		"pages", totalPages,
		"assets", len(bundle.assetCache),
		"bytes", bundle.totalBytes,
	)

	return tmpPath, downloadName, nil
}
