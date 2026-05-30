package services

// courseware_export_service.go — 课件离线打包导出服务（编排层）
//
// 功能：把一套课件的所有已生成页面 + 引用到的图片/视频/音频资源，
//       整合打包成一个 zip。解压后是「单一文件夹」，老师拷到任意电脑、
//       断网双击 index.html 即可全屏授课、左右翻页。
//
// 设计要点：
//   1. 常规课件页 html_content 是 1920×1080 的 <div> 片段（不含 <html>），
//      离线时由本服务包装成独立 HTML 文档（缩放自适应 + 翻页导航）。
//   2. 3D/导入页 html_content 是完整 <html> 文档，直接使用，仅在多页时注入导航。
//   3. 资源（图片/视频/音频）通过扫描页面 HTML 发现，本地读盘 / 远程下载，
//      流式写入 zip 的 assets/ 目录，并把页面里的 URL 改写成相对路径。
//   4. zip 临时文件落在 /www/wwwroot/tedna/tmp（大磁盘、非 Nginx 暴露目录），
//      由 handler 流式返回后立即删除。
//
// 资源发现 / 改写 / HTML 包装等细节拆分在 courseware_export_assets.go。

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

// cwExportLog 模块级结构化日志器（assets 文件也复用此变量）
var cwExportLog = logger.WithModule("courseware_export")

const (
	// cwExportRoot 本地上传文件的磁盘根目录（/uploads/xxx 映射到此处）
	cwExportRoot = "/www/wwwroot/tedna"
	// cwExportTmpDir zip 临时文件目录（非 Nginx 暴露，避免外泄）
	cwExportTmpDir = cwExportRoot + "/tmp"
	// cwExportMaxAssetBytes 单个资源大小上限（超过则跳过该资源，保留原链接）
	cwExportMaxAssetBytes = 500 * 1024 * 1024 // 500MB
	// cwExportMaxTotalBytes 整包大小上限（超过则中止打包，防止磁盘被打爆）
	cwExportMaxTotalBytes = 3 * 1024 * 1024 * 1024 // 3GB
)

// CoursewareExportService 课件离线打包导出服务（无状态，可随用随建）
type CoursewareExportService struct{}

// NewCoursewareExportService 创建导出服务实例
func NewCoursewareExportService() *CoursewareExportService {
	return &CoursewareExportService{}
}

// ExportBundle 生成课件离线包
//
// 参数:
//   - coursewareID: 课件ID
//   - userID:       当前登录用户ID（用于归属校验）
//
// 返回:
//   - zipPath:      生成的临时 zip 文件磁盘路径（调用方负责返回后删除）
//   - downloadName: 建议的下载文件名（含 .zip）
//   - err:          错误
func (s *CoursewareExportService) ExportBundle(ctx context.Context, coursewareID, userID string) (string, string, error) {
	// 1. 加载课件 + 归属校验（只能导出自己的课件）
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", "", fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		return "", "", fmt.Errorf("无权导出该课件")
	}

	// 2. 加载全部页面（已按 page_number 升序）
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil {
		return "", "", fmt.Errorf("加载页面失败: %w", err)
	}

	// 只保留已经生成了 HTML 的页面（跳过尚未生成的空页）
	var valid []*models.CoursewarePage
	for _, p := range pages {
		if strings.TrimSpace(p.HTMLContent) != "" {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return "", "", fmt.Errorf("课件尚未生成页面内容，无法导出")
	}

	// 3. 准备临时 zip 文件（落在大磁盘的非暴露目录）
	if err := os.MkdirAll(cwExportTmpDir, 0o755); err != nil {
		return "", "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	tmp, err := os.CreateTemp(cwExportTmpDir, "cw-bundle-*.zip")
	if err != nil {
		return "", "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	zw := zip.NewWriter(tmp)

	// 解压后的顶层文件夹名（保证「单一文件夹」体验）
	rootDir := sanitizeBundleName(cw.Title)
	if rootDir == "" {
		rootDir = "courseware"
	}

	bundle := &cwExportBundle{
		zw:         zw,
		rootDir:    rootDir,
		assetCache: make(map[string]string),
	}

	// 出错时统一清理：关闭 writer、关闭并删除临时文件
	failClose := func(e error) (string, string, error) {
		_ = zw.Close()
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", "", e
	}

	total := len(valid)

	// 4. 逐页处理
	for i, p := range valid {
		pageNum := i + 1

		title := strings.TrimSpace(p.Title)
		if title == "" {
			title = fmt.Sprintf("第%d页", pageNum)
		}

		// 4.1 改写资源 URL —— 同时把资源流式写入 zip 的 assets/ 目录
		//      （此步骤会在 zip 里创建若干 assets 条目，但此刻还没有打开页面条目，
		//        符合 archive/zip「同一时刻只能有一个打开的条目」的约束）
		rewritten := bundle.rewriteAssets(p.HTMLContent)

		// 4.2 包装成独立可双击打开的 HTML 文档
		doc := buildOfflinePageDoc(rewritten, pageNum, total, title, cw.Title)

		// 4.3 写入 pN.html
		if err := bundle.writeText(fmt.Sprintf("p%d.html", pageNum), doc); err != nil {
			return failClose(fmt.Errorf("写入页面失败: %w", err))
		}

		// 整包大小护栏
		if bundle.totalBytes > cwExportMaxTotalBytes {
			return failClose(fmt.Errorf("课件资源总大小超出上限(3GB)，无法打包"))
		}
	}

	// 5. 入口页 + 使用说明
	if err := bundle.writeText("index.html", buildOfflineIndexDoc(cw.Title, total)); err != nil {
		return failClose(fmt.Errorf("写入入口页失败: %w", err))
	}
	if err := bundle.writeText("使用说明.txt", buildOfflineReadme(cw.Title, total)); err != nil {
		return failClose(fmt.Errorf("写入说明失败: %w", err))
	}

	// 6. 收尾
	if err := zw.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("打包失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	downloadName := rootDir + ".zip"
	cwExportLog.Info("课件离线包生成成功",
		"courseware_id", coursewareID,
		"pages", total,
		"assets", len(bundle.assetCache),
		"bytes", bundle.totalBytes,
	)
	return tmpPath, downloadName, nil
}
