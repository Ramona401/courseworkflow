package services

// courseware_export_assets.go — 课件离线包资源发现、URL改写和HTML包装
//
// 本文件负责：
//   - ZIP写入上下文与资源去重；
//   - 扫描页面HTML中的src、href、poster和CSS url；
//   - 本地读取或远程下载媒体资源；
//   - 把媒体URL改写为ZIP内相对路径；
//   - 把页面片段或完整文档包装为可双击运行的独立HTML；
//   - 生成统一iframe播放器入口和纯离线使用说明。
//
// 离线ZIP不注入教学智能体。历史导出助手痕迹由编排服务在资源扫描前清除，
// 防止public_id、embed地址或悬浮入口进入纯离线交付物。

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// cwExportBundle 单次打包上下文。
type cwExportBundle struct {
	zw         *zip.Writer
	rootDir    string
	assetCache map[string]string
	totalBytes int64
}

// writeText 向ZIP写入文本条目。
func (bundle *cwExportBundle) writeText(relativePath string, content string) error {
	writer, err := bundle.zw.Create(bundle.rootDir + "/" + relativePath)
	if err != nil {
		return err
	}

	written, err := io.WriteString(writer, content)
	bundle.totalBytes += int64(written)
	return err
}

// ==================== 资源URL扫描与改写 ====================

var (
	cwExpAttrDouble = regexp.MustCompile(`(?i)(src|href|poster)\s*=\s*"([^"]*)"`)
	cwExpAttrSingle = regexp.MustCompile(`(?i)(src|href|poster)\s*=\s*'([^']*)'`)
	cwExpCSSURL     = regexp.MustCompile(`(?i)url\(\s*['"]?([^)'"]+)['"]?\s*\)`)
)

// rewriteAssets 扫描页面HTML，将可下载媒体改写为ZIP内相对路径。
func (bundle *cwExportBundle) rewriteAssets(document string) string {
	document = cwExpAttrDouble.ReplaceAllStringFunc(document, func(match string) string {
		submatches := cwExpAttrDouble.FindStringSubmatch(match)
		if submatches == nil {
			return match
		}

		rewritten, ok := bundle.processAssetURL(submatches[2])
		if !ok {
			return match
		}

		return submatches[1] + `="` + rewritten + `"`
	})

	document = cwExpAttrSingle.ReplaceAllStringFunc(document, func(match string) string {
		submatches := cwExpAttrSingle.FindStringSubmatch(match)
		if submatches == nil {
			return match
		}

		rewritten, ok := bundle.processAssetURL(submatches[2])
		if !ok {
			return match
		}

		return submatches[1] + `='` + rewritten + `'`
	})

	document = cwExpCSSURL.ReplaceAllStringFunc(document, func(match string) string {
		submatches := cwExpCSSURL.FindStringSubmatch(match)
		if submatches == nil {
			return match
		}

		rewritten, ok := bundle.processAssetURL(submatches[1])
		if !ok {
			return match
		}

		return "url('" + rewritten + "')"
	})

	return document
}

// processAssetURL 下载单个媒体并返回ZIP内相对路径。
func (bundle *cwExportBundle) processAssetURL(raw string) (string, bool) {
	assetURL := strings.TrimSpace(raw)
	if assetURL == "" {
		return "", false
	}

	lowerURL := strings.ToLower(assetURL)
	if strings.HasPrefix(lowerURL, "data:") ||
		strings.HasPrefix(lowerURL, "blob:") ||
		strings.HasPrefix(assetURL, "#") ||
		strings.HasPrefix(lowerURL, "javascript:") ||
		strings.HasPrefix(lowerURL, "mailto:") ||
		strings.Contains(assetURL, "{{") {
		return "", false
	}

	if cached, exists := bundle.assetCache[assetURL]; exists {
		return cached, true
	}

	if !isDownloadableAsset(assetURL) {
		return "", false
	}

	reader, err := resolveAssetReader(assetURL)
	if err != nil {
		cwExportLog.Warn(
			"资源无法获取，保留原链接",
			"url", assetURL,
			"error", err,
		)
		return "", false
	}
	defer reader.Close()

	relativePath := "assets/" + cwAssetBundleName(assetURL)
	writer, err := bundle.zw.Create(bundle.rootDir + "/" + relativePath)
	if err != nil {
		cwExportLog.Warn(
			"创建资源条目失败，保留原链接",
			"url", assetURL,
			"error", err,
		)
		return "", false
	}

	written, err := io.Copy(writer, reader)
	bundle.totalBytes += written
	if err != nil {
		cwExportLog.Warn(
			"写入资源失败，保留原链接",
			"url", assetURL,
			"error", err,
		)
		return "", false
	}

	bundle.assetCache[assetURL] = relativePath
	return relativePath, true
}

var cwMediaExtensions = []string{
	".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg", ".bmp", ".ico",
	".mp4", ".webm", ".mov", ".avi", ".m4v",
	".mp3", ".wav", ".ogg", ".oga", ".m4a", ".aac", ".flac",
}

// isDownloadableAsset 判断URL是否应进入离线包。
func isDownloadableAsset(assetURL string) bool {
	path := assetURL
	if index := strings.IndexAny(path, "?#"); index >= 0 {
		path = path[:index]
	}

	lowerPath := strings.ToLower(path)
	if strings.HasPrefix(path, "/uploads/") {
		return true
	}

	remote := strings.HasPrefix(lowerPath, "http://") ||
		strings.HasPrefix(lowerPath, "https://") ||
		strings.HasPrefix(path, "//")
	if !remote {
		return false
	}

	if strings.Contains(lowerPath, "/uploads/") {
		return true
	}

	for _, extension := range cwMediaExtensions {
		if strings.HasSuffix(lowerPath, extension) {
			return true
		}
	}

	return false
}

// resolveAssetReader 将媒体URL解析为本地文件流或远程HTTP流。
func resolveAssetReader(assetURL string) (io.ReadCloser, error) {
	path := assetURL
	if index := strings.IndexAny(path, "?#"); index >= 0 {
		path = path[:index]
	}

	diskPath := ""
	if strings.HasPrefix(path, "/uploads/") {
		diskPath = cwExportRoot + path
	} else if index := strings.Index(path, "/uploads/"); index >= 0 {
		diskPath = cwExportRoot + path[index:]
	}

	if diskPath != "" {
		fileInfo, err := os.Stat(diskPath)
		if err != nil {
			return nil, fmt.Errorf("本地文件不存在: %s", diskPath)
		}
		if fileInfo.Size() > cwExportMaxAssetBytes {
			return nil, fmt.Errorf("文件超过单文件上限: %s", diskPath)
		}

		file, err := os.Open(diskPath)
		if err != nil {
			return nil, err
		}

		return file, nil
	}

	fetchURL := assetURL
	if strings.HasPrefix(fetchURL, "//") {
		fetchURL = "https:" + fetchURL
	}

	client := &http.Client{Timeout: 120 * time.Second}
	response, err := client.Get(fetchURL)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	if response.ContentLength > cwExportMaxAssetBytes {
		_ = response.Body.Close()
		return nil, fmt.Errorf("远程文件超过单文件上限")
	}

	return response.Body, nil
}

// cwAssetBundleName 生成资源去重文件名。
func cwAssetBundleName(assetURL string) string {
	sum := md5.Sum([]byte(assetURL))
	prefix := hex.EncodeToString(sum[:])[:8]
	return prefix + "_" + safeAssetBase(assetURL)
}

// safeAssetBase 提取并清洗安全文件名。
func safeAssetBase(assetURL string) string {
	path := assetURL
	if index := strings.IndexAny(path, "?#"); index >= 0 {
		path = path[:index]
	}

	base := path
	if index := strings.LastIndex(base, "/"); index >= 0 {
		base = base[index+1:]
	}

	base = strings.TrimSpace(base)
	if base == "" {
		base = "asset"
	}

	var builder strings.Builder
	for _, character := range base {
		switch character {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ' ':
			builder.WriteRune('_')
		default:
			builder.WriteRune(character)
		}
	}

	result := builder.String()
	runes := []rune(result)
	if len(runes) > 60 {
		result = string(runes[len(runes)-60:])
	}

	return result
}

// ==================== HTML包装 ====================

const cwOflPageCSS = `*{margin:0;padding:0;box-sizing:border-box;}
html,body{width:100%;height:100%;overflow:hidden;background:#0f172a;font-family:'Inter','PingFang SC','Microsoft YaHei',system-ui,sans-serif;}
#cw-stage{position:absolute;top:50%;left:50%;width:1920px;height:1080px;transform:translate(-50%,-50%);transform-origin:center center;background:#fff;box-shadow:0 0 60px rgba(0,0,0,0.5);}
.cw-ofl-nav{display:none;position:fixed;bottom:16px;left:50%;transform:translateX(-50%);align-items:center;gap:10px;z-index:2147483647;background:rgba(15,23,42,0.78);padding:8px 14px;border-radius:999px;box-shadow:0 4px 20px rgba(0,0,0,0.35);}
.cw-ofl-btn{display:inline-block;padding:6px 14px;border-radius:999px;border:none;background:rgba(255,255,255,0.14);color:#fff;font-size:14px;line-height:1.4;cursor:pointer;}
.cw-ofl-btn:hover{background:rgba(255,255,255,0.26);}
.cw-ofl-disabled{opacity:0.35;cursor:default;}
.cw-ofl-count{color:#fff;font-size:14px;min-width:54px;text-align:center;}
@media print{.cw-ofl-nav{display:none;}}`

// buildOfflinePageDoc 将页面内容包装为可独立打开的HTML文档。
func buildOfflinePageDoc(
	inner string,
	pageNumber int,
	totalPages int,
	title string,
	coursewareTitle string,
) string {
	trimmed := strings.TrimSpace(inner)
	lower := strings.ToLower(trimmed)

	// 完整文档型页面保留原始文档，只在body结束前注入框架感知导航。
	if strings.HasPrefix(lower, "<!doctype") ||
		strings.HasPrefix(lower, "<html") {
		navigation := buildFloatingNav(pageNumber, totalPages)
		if bodyClose := strings.LastIndex(lower, "</body>"); bodyClose >= 0 {
			return trimmed[:bodyClose] + navigation + trimmed[bodyClose:]
		}
		return trimmed + navigation
	}

	previousPage := ""
	nextPage := ""

	if pageNumber > 1 {
		previousPage = "p" + strconv.Itoa(pageNumber-1) + ".html"
	}
	if pageNumber < totalPages {
		nextPage = "p" + strconv.Itoa(pageNumber+1) + ".html"
	}

	var builder strings.Builder
	builder.WriteString("<!DOCTYPE html>\n<html lang=\"zh-CN\">\n<head>\n<meta charset=\"UTF-8\">\n")
	builder.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	builder.WriteString("<title>" + htmlEscape(coursewareTitle) + " · 第" + strconv.Itoa(pageNumber) + "页</title>\n")
	builder.WriteString("<style>\n" + cwOflPageCSS + "\n</style>\n</head>\n<body>\n")
	builder.WriteString("<div id=\"cw-stage\">")
	builder.WriteString(trimmed)
	builder.WriteString("</div>\n")
	builder.WriteString(buildStageNav(previousPage, nextPage))
	builder.WriteString("\n")
	builder.WriteString(buildPageScript(previousPage, nextPage))
	builder.WriteString("\n</body>\n</html>")

	return builder.String()
}

// buildStageNav 构造单页直接打开时使用的底部导航。
func buildStageNav(previousPage string, nextPage string) string {
	var builder strings.Builder
	builder.WriteString(`<div class="cw-ofl-nav" id="cw-ofl-nav">`)

	if previousPage != "" {
		builder.WriteString(`<button class="cw-ofl-btn" onclick="cwNav('prev')">← 上一页</button>`)
	} else {
		builder.WriteString(`<span class="cw-ofl-btn cw-ofl-disabled">← 上一页</span>`)
	}

	if nextPage != "" {
		builder.WriteString(`<button class="cw-ofl-btn" onclick="cwNav('next')">下一页 →</button>`)
	} else {
		builder.WriteString(`<span class="cw-ofl-btn cw-ofl-disabled">下一页 →</span>`)
	}

	builder.WriteString(`<button class="cw-ofl-btn" onclick="cwFull()">⛶ 全屏</button>`)
	builder.WriteString(`</div>`)

	return builder.String()
}

// buildPageScript 构造缩放、翻页、全屏和键盘脚本。
func buildPageScript(previousPage string, nextPage string) string {
	return "<script>\n(function(){\n" +
		"var framed=(window.self!==window.top);\n" +
		"function fit(){var s=Math.min(window.innerWidth/1920,window.innerHeight/1080);var el=document.getElementById('cw-stage');if(el){el.style.transform='translate(-50%,-50%) scale('+s+')';}}\n" +
		"window.addEventListener('resize',fit);window.addEventListener('load',fit);fit();\n" +
		"var PREV='" + previousPage + "',NEXT='" + nextPage + "';\n" +
		"window.cwNav=function(dir){if(framed){window.parent.postMessage({__cwNav:dir},'*');}else{var h=(dir==='next'?NEXT:PREV);if(h)location.href=h;}};\n" +
		"window.cwFull=function(){if(framed){window.parent.postMessage({__cwFull:1},'*');return;}var el=document.documentElement;if(!document.fullscreenElement&&!document.webkitFullscreenElement){(el.requestFullscreen||el.webkitRequestFullscreen||function(){}).call(el);}else{(document.exitFullscreen||document.webkitExitFullscreen||function(){}).call(document);}};\n" +
		"var nav=document.getElementById('cw-ofl-nav');if(nav&&!framed){nav.style.display='flex';}\n" +
		"document.addEventListener('keydown',function(e){\n" +
		"if(e.key==='ArrowRight'||e.key==='PageDown'||e.key===' '){e.preventDefault();window.cwNav('next');}\n" +
		"else if(e.key==='ArrowLeft'||e.key==='PageUp'){e.preventDefault();window.cwNav('prev');}\n" +
		"else if(e.key==='f'||e.key==='F'){window.cwFull();}\n" +
		"});\n})();\n</script>"
}

// buildFloatingNav 为完整文档型页面注入框架感知导航。
func buildFloatingNav(pageNumber int, totalPages int) string {
	previousPage := ""
	nextPage := ""

	if pageNumber > 1 {
		previousPage = "p" + strconv.Itoa(pageNumber-1) + ".html"
	}
	if pageNumber < totalPages {
		nextPage = "p" + strconv.Itoa(pageNumber+1) + ".html"
	}

	var builder strings.Builder
	builder.WriteString(`<style>#cw-ofl-nav{display:none;position:fixed;bottom:16px;left:50%;transform:translateX(-50%);align-items:center;gap:10px;z-index:2147483647;background:rgba(15,23,42,0.8);padding:8px 14px;border-radius:999px;}#cw-ofl-nav button,#cw-ofl-nav span{padding:6px 14px;border-radius:999px;border:none;color:#fff;font-size:14px;background:rgba(255,255,255,0.16);cursor:pointer;}#cw-ofl-nav .d{opacity:.35;cursor:default;}</style>`)
	builder.WriteString(`<div id="cw-ofl-nav">`)

	if previousPage != "" {
		builder.WriteString(`<button onclick="cwNav('prev')">← 上一页</button>`)
	} else {
		builder.WriteString(`<span class="d">← 上一页</span>`)
	}

	if nextPage != "" {
		builder.WriteString(`<button onclick="cwNav('next')">下一页 →</button>`)
	} else {
		builder.WriteString(`<span class="d">下一页 →</span>`)
	}

	builder.WriteString(`<button onclick="cwFull()">⛶ 全屏</button>`)
	builder.WriteString(`</div>`)
	builder.WriteString(buildPageScript(previousPage, nextPage))

	return builder.String()
}

const cwOflShellCSS = `*{margin:0;padding:0;box-sizing:border-box;}
html,body{width:100%;height:100%;overflow:hidden;background:#0f172a;font-family:'PingFang SC','Microsoft YaHei',system-ui,sans-serif;}
#cw-frame{position:fixed;inset:0;width:100%;height:100%;border:0;}
.cw-shell-nav{position:fixed;bottom:16px;left:50%;transform:translateX(-50%);display:flex;align-items:center;gap:10px;z-index:2147483647;background:rgba(15,23,42,0.8);padding:8px 14px;border-radius:999px;box-shadow:0 4px 20px rgba(0,0,0,0.35);}
.cw-shell-nav button{padding:6px 14px;border-radius:999px;border:none;background:rgba(255,255,255,0.14);color:#fff;font-size:14px;cursor:pointer;}
.cw-shell-nav button:hover{background:rgba(255,255,255,0.26);}
.cw-shell-nav button:disabled{opacity:0.35;cursor:default;}
.cw-shell-count{color:#fff;font-size:14px;min-width:54px;text-align:center;}`

// buildOfflineIndexDoc 构造保持全屏状态的iframe播放器入口。
func buildOfflineIndexDoc(coursewareTitle string, totalPages int) string {
	total := strconv.Itoa(totalPages)

	var builder strings.Builder
	builder.WriteString("<!DOCTYPE html>\n<html lang=\"zh-CN\">\n<head>\n<meta charset=\"UTF-8\">\n")
	builder.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	builder.WriteString("<title>" + htmlEscape(coursewareTitle) + "</title>\n")
	builder.WriteString("<style>\n" + cwOflShellCSS + "\n</style>\n</head>\n<body>\n")
	builder.WriteString(`<iframe id="cw-frame" src="p1.html"></iframe>` + "\n")
	builder.WriteString(`<div class="cw-shell-nav">`)
	builder.WriteString(`<button id="cw-prev">← 上一页</button>`)
	builder.WriteString(`<span class="cw-shell-count" id="cw-count">1 / ` + total + `</span>`)
	builder.WriteString(`<button id="cw-next">下一页 →</button>`)
	builder.WriteString(`<button id="cw-full">⛶ 全屏</button>`)
	builder.WriteString(`</div>` + "\n")
	builder.WriteString("<script>\n(function(){\n")
	builder.WriteString("var total=" + total + ",cur=1;\n")
	builder.WriteString("var frame=document.getElementById('cw-frame');\n")
	builder.WriteString("var prevBtn=document.getElementById('cw-prev'),nextBtn=document.getElementById('cw-next'),cnt=document.getElementById('cw-count');\n")
	builder.WriteString("function upd(){cnt.textContent=cur+' / '+total;prevBtn.disabled=(cur<=1);nextBtn.disabled=(cur>=total);}\n")
	builder.WriteString("function go(n){if(n<1||n>total)return;cur=n;frame.src='p'+n+'.html';upd();}\n")
	builder.WriteString("function toggleFull(){var el=document.documentElement;if(!document.fullscreenElement&&!document.webkitFullscreenElement){(el.requestFullscreen||el.webkitRequestFullscreen||function(){}).call(el);}else{(document.exitFullscreen||document.webkitExitFullscreen||function(){}).call(document);}}\n")
	builder.WriteString("prevBtn.onclick=function(){go(cur-1);};nextBtn.onclick=function(){go(cur+1);};\n")
	builder.WriteString("document.getElementById('cw-full').onclick=toggleFull;\n")
	builder.WriteString("window.addEventListener('message',function(e){\n")
	builder.WriteString("var d=e.data;if(!d||typeof d!=='object')return;\n")
	builder.WriteString("if(d.__cwNav==='next'){go(cur+1);return;}\n")
	builder.WriteString("if(d.__cwNav==='prev'){go(cur-1);return;}\n")
	builder.WriteString("if(d.__cwFull){toggleFull();return;}\n")
	builder.WriteString("if(d.action==='navigate'){if(d.direction==='prev'){go(cur-1);}else{go(cur+1);}return;}\n")
	builder.WriteString("if(d.action==='next'){go(cur+1);return;}\n")
	builder.WriteString("if(d.action==='prev'){go(cur-1);return;}\n")
	builder.WriteString("});\n")
	builder.WriteString("window.addEventListener('keydown',function(e){if(e.key==='ArrowRight'||e.key==='PageDown'||e.key===' '){e.preventDefault();go(cur+1);}else if(e.key==='ArrowLeft'||e.key==='PageUp'){e.preventDefault();go(cur-1);}else if(e.key==='f'||e.key==='F'){toggleFull();}});\n")
	builder.WriteString("upd();\n})();\n</script>\n</body>\n</html>")

	return builder.String()
}

// buildOfflineReadme 生成导出包使用说明。
func buildOfflineReadme(
	coursewareTitle string,
	totalPages int,
) string {
	var builder strings.Builder

	builder.WriteString("课件离线包使用说明\n")
	builder.WriteString("====================\n\n")
	builder.WriteString("课件名称：" + coursewareTitle + "\n")
	builder.WriteString("页面数量：" + strconv.Itoa(totalPages) + " 页\n")
	builder.WriteString("教学智能体：不包含\n\n")

	builder.WriteString("【如何打开】\n")
	builder.WriteString("1. 将本文件夹完整拷贝到电脑任意位置，不能只复制index.html。\n")
	builder.WriteString("2. 双击文件夹内的index.html，用Chrome或Edge打开。\n")
	builder.WriteString("3. index.html是离线播放器，首屏直接显示第1页。\n\n")

	builder.WriteString("【如何翻页和全屏】\n")
	builder.WriteString("· 点击底部上一页或下一页，或使用键盘方向键、PageUp、PageDown和空格。\n")
	builder.WriteString("· 点击全屏按钮或按F进入全屏，翻页不会退出全屏。\n")
	builder.WriteString("· 按Esc退出全屏。\n\n")

	builder.WriteString("【教学智能体】\n")
	builder.WriteString("· 本ZIP是纯离线课件包，不包含教学智能体悬浮入口。\n")
	builder.WriteString("· ZIP只包含课件页面、离线播放器和已打包媒体，不携带任何教学智能体运行数据、联网代码或在线服务地址。\n")
	builder.WriteString("· 通过file://打开的本地页面不会尝试连接教学智能体服务，也不会显示无法使用的助手按钮。\n")
	builder.WriteString("· 需要使用教学智能体时，请回到TE-DNA平台使用登录态预览，或使用未来独立的HTTPS在线发布链路。\n")
	builder.WriteString("· 在线教学智能体与本离线ZIP是两种独立交付方式，不应把本文件夹直接当作在线助手发布包。\n\n")

	builder.WriteString("【文件夹结构】\n")
	builder.WriteString("· index.html：离线播放器入口。\n")
	builder.WriteString("· p1.html至pN.html：各页课件，也可以单独打开。\n")
	builder.WriteString("· assets/：已打包的图片、视频和音频。\n\n")

	builder.WriteString("【离线说明】\n")
	builder.WriteString("· 已成功打包的图片、视频和音频断网仍可使用。\n")
	builder.WriteString("· 原课件自身若依赖在线AI、在线语音、远程网页或第三方接口，断网时相关原生功能可能不可用。\n")
	builder.WriteString("· 3D互动课件若依赖在线3D引擎，断网时可能无法显示。\n")

	return builder.String()
}

// ==================== 通用工具 ====================

// sanitizeBundleName 清洗ZIP顶层文件夹名。
func sanitizeBundleName(name string) string {
	name = strings.TrimSpace(name)

	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)

	name = strings.TrimSpace(replacer.Replace(name))
	runes := []rune(name)
	if len(runes) > 80 {
		name = string(runes[:80])
	}

	return strings.Trim(name, ". ")
}

// htmlEscape 转义HTML文本和属性中的特殊字符。
func htmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)

	return replacer.Replace(value)
}
