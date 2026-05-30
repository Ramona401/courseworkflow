package services

// courseware_export_assets.go — 课件离线包：资源发现/下载/URL改写 + 页面HTML包装
//
// 拆分自 courseware_export_service.go，集中处理：
//   - cwExportBundle：打包上下文（zip writer + 资源去重缓存 + 计字节）
//   - rewriteAssets：扫描页面 HTML，把资源 URL 改写为相对路径，并把资源流式写入 zip
//   - resolveAssetReader：把一个 URL 解析为可读流（本地读盘 / 远程下载）
//   - buildOfflinePageDoc：把页面片段/完整文档包装成可双击打开的离线 HTML（缩放自适应）
//   - buildOfflineIndexDoc：iframe 播放器外壳（统一风格 + 翻页时全屏不丢失）
//   - buildOfflineReadme：使用说明
//
// 复用 courseware_export_service.go 中定义的 cwExportLog / cwExportRoot 等常量。
//
// v2 改进（修复两个体验问题）：
//   1. index.html 不再是深色"跳转过渡页"，而是内嵌 iframe 的播放器外壳，
//      首屏即真实第 1 页，外观与课件完全一致。
//   2. 翻页改为只切换 iframe 的 src（不跳转父文档），全屏作用于外壳，
//      因此「先点一次全屏，之后翻页一直保持全屏」，不再每翻一页就退出。
//      键盘方向键由页面通过 postMessage 转发给外壳，翻页/全屏连贯。
//      每页仍可单独打开（脱离外壳时回退到自身的翻页与全屏）。
//
// v3 改进（修复封面 CTA 翻页不响应）：
//   - 外壳的 message 监听同时兼容两种翻页协议：
//       新协议 {__cwNav:'next'|'prev'} / {__cwFull:1}（本系统页面脚本发出）
//       旧协议 {action:'navigate', direction:'next'|'prev'}（部分导入型/封面 CTA 按钮发出，
//              例如封面"开启之旅"按钮内联的 postMessage({action:'navigate',direction:'next'})）
//     这样无论页面用哪种协议，都能在播放器外壳里正确翻页。

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

// cwExportBundle 单次打包的上下文
type cwExportBundle struct {
	zw         *zip.Writer       // zip 写入器
	rootDir    string            // 顶层文件夹名（所有条目前缀）
	assetCache map[string]string // 资源去重缓存：原始URL -> 包内相对路径(assets/xxx)
	totalBytes int64             // 已写入字节累计（用于整包大小护栏）
}

// writeText 向 zip 写入一个文本条目（路径会自动加上顶层文件夹前缀）
func (b *cwExportBundle) writeText(rel, content string) error {
	w, err := b.zw.Create(b.rootDir + "/" + rel)
	if err != nil {
		return err
	}
	n, err := io.WriteString(w, content)
	b.totalBytes += int64(n)
	return err
}

// ==================== 资源 URL 扫描与改写 ====================

// 匹配 src/href/poster="..." 与 '...'，以及 CSS url(...)
var (
	cwExpAttrDouble = regexp.MustCompile(`(?i)(src|href|poster)\s*=\s*"([^"]*)"`)
	cwExpAttrSingle = regexp.MustCompile(`(?i)(src|href|poster)\s*=\s*'([^']*)'`)
	cwExpCSSUrl     = regexp.MustCompile(`(?i)url\(\s*['"]?([^)'"]+)['"]?\s*\)`)
)

// rewriteAssets 扫描页面 HTML，把可下载的资源 URL 改写为包内相对路径，
// 同时把资源本体流式写进 zip 的 assets/ 目录。无法获取的资源保留原链接（在线仍可用）。
func (b *cwExportBundle) rewriteAssets(html string) string {
	html = cwExpAttrDouble.ReplaceAllStringFunc(html, func(m string) string {
		sub := cwExpAttrDouble.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		nu, ok := b.processAssetURL(sub[2])
		if !ok {
			return m
		}
		return sub[1] + `="` + nu + `"`
	})
	html = cwExpAttrSingle.ReplaceAllStringFunc(html, func(m string) string {
		sub := cwExpAttrSingle.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		nu, ok := b.processAssetURL(sub[2])
		if !ok {
			return m
		}
		return sub[1] + `='` + nu + `'`
	})
	html = cwExpCSSUrl.ReplaceAllStringFunc(html, func(m string) string {
		sub := cwExpCSSUrl.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		nu, ok := b.processAssetURL(sub[1])
		if !ok {
			return m
		}
		return "url('" + nu + "')"
	})
	return html
}

// processAssetURL 处理单个 URL：若是可下载资源，则写入 zip 并返回包内相对路径。
func (b *cwExportBundle) processAssetURL(raw string) (string, bool) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", false
	}
	lu := strings.ToLower(u)
	if strings.HasPrefix(lu, "data:") || strings.HasPrefix(lu, "blob:") ||
		strings.HasPrefix(u, "#") || strings.HasPrefix(lu, "javascript:") ||
		strings.HasPrefix(lu, "mailto:") || strings.Contains(u, "{{") {
		return "", false
	}
	if cached, ok := b.assetCache[u]; ok {
		return cached, true
	}
	if !isDownloadableAsset(u) {
		return "", false
	}

	rc, err := resolveAssetReader(u)
	if err != nil {
		cwExportLog.Warn("资源无法获取，保留原链接", "url", u, "error", err)
		return "", false
	}
	defer rc.Close()

	rel := "assets/" + cwAssetBundleName(u)
	w, err := b.zw.Create(b.rootDir + "/" + rel)
	if err != nil {
		cwExportLog.Warn("创建资源条目失败，保留原链接", "url", u, "error", err)
		return "", false
	}
	n, err := io.Copy(w, rc)
	b.totalBytes += n
	if err != nil {
		cwExportLog.Warn("写入资源失败，保留原链接", "url", u, "error", err)
		return "", false
	}

	b.assetCache[u] = rel
	return rel, true
}

// 可识别的媒体扩展名
var cwMediaExts = []string{
	".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg", ".bmp", ".ico",
	".mp4", ".webm", ".mov", ".avi", ".m4v",
	".mp3", ".wav", ".ogg", ".oga", ".m4a", ".aac", ".flac",
}

// isDownloadableAsset 判断一个 URL 是否是「应当打进包里的资源」
func isDownloadableAsset(u string) bool {
	p := u
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	lp := strings.ToLower(p)
	if strings.HasPrefix(p, "/uploads/") {
		return true
	}
	isRemote := strings.HasPrefix(lp, "http://") || strings.HasPrefix(lp, "https://") || strings.HasPrefix(p, "//")
	if isRemote {
		if strings.Contains(lp, "/uploads/") {
			return true
		}
		for _, e := range cwMediaExts {
			if strings.HasSuffix(lp, e) {
				return true
			}
		}
	}
	return false
}

// resolveAssetReader 把资源 URL 解析为可读流：本地路径直接读盘，远程 URL 走 HTTP。
func resolveAssetReader(u string) (io.ReadCloser, error) {
	p := u
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}

	disk := ""
	if strings.HasPrefix(p, "/uploads/") {
		disk = cwExportRoot + p
	} else if idx := strings.Index(p, "/uploads/"); idx >= 0 {
		disk = cwExportRoot + p[idx:]
	}

	if disk != "" {
		fi, err := os.Stat(disk)
		if err != nil {
			return nil, fmt.Errorf("本地文件不存在: %s", disk)
		}
		if fi.Size() > cwExportMaxAssetBytes {
			return nil, fmt.Errorf("文件超过单文件上限: %s", disk)
		}
		f, err := os.Open(disk)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	fetchURL := u
	if strings.HasPrefix(fetchURL, "//") {
		fetchURL = "https:" + fetchURL
	}
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(fetchURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > cwExportMaxAssetBytes {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("远程文件超过单文件上限")
	}
	return resp.Body, nil
}

// cwAssetBundleName 生成包内资源文件名：md5前缀(去重) + 安全的原始文件名
func cwAssetBundleName(u string) string {
	sum := md5.Sum([]byte(u))
	prefix := hex.EncodeToString(sum[:])[:8]
	return prefix + "_" + safeAssetBase(u)
}

// safeAssetBase 从 URL 提取并清洗出安全的文件名（保留扩展名）
func safeAssetBase(u string) string {
	p := u
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	base := p
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = "asset"
	}
	var sb strings.Builder
	for _, c := range base {
		switch c {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ' ':
			sb.WriteRune('_')
		default:
			sb.WriteRune(c)
		}
	}
	res := sb.String()
	r := []rune(res)
	if len(r) > 60 {
		res = string(r[len(r)-60:])
	}
	return res
}

// ==================== HTML 包装 ====================

// 离线页面外壳 CSS（1920×1080 居中缩放 + 翻页导航条；导航默认隐藏，单独打开时由JS显示）
const cwOflPageCSS = `*{margin:0;padding:0;box-sizing:border-box;}
html,body{width:100%;height:100%;overflow:hidden;background:#0f172a;font-family:'Inter','PingFang SC','Microsoft YaHei',system-ui,sans-serif;}
#cw-stage{position:absolute;top:50%;left:50%;width:1920px;height:1080px;transform:translate(-50%,-50%);transform-origin:center center;background:#fff;box-shadow:0 0 60px rgba(0,0,0,0.5);}
.cw-ofl-nav{display:none;position:fixed;bottom:16px;left:50%;transform:translateX(-50%);align-items:center;gap:10px;z-index:2147483647;background:rgba(15,23,42,0.78);padding:8px 14px;border-radius:999px;box-shadow:0 4px 20px rgba(0,0,0,0.35);}
.cw-ofl-btn{display:inline-block;padding:6px 14px;border-radius:999px;border:none;background:rgba(255,255,255,0.14);color:#fff;font-size:14px;line-height:1.4;cursor:pointer;}
.cw-ofl-btn:hover{background:rgba(255,255,255,0.26);}
.cw-ofl-disabled{opacity:0.35;cursor:default;}
.cw-ofl-count{color:#fff;font-size:14px;min-width:54px;text-align:center;}
@media print{.cw-ofl-nav{display:none;}}`

// buildOfflinePageDoc 把页面内容包装成一个可双击打开的独立 HTML 文档
func buildOfflinePageDoc(inner string, pageNum, total int, title, cwTitle string) string {
	t := strings.TrimSpace(inner)
	low := strings.ToLower(t)

	// 完整文档（3D / HTML导入）：注入「框架感知」的浮动导航
	if strings.HasPrefix(low, "<!doctype") || strings.HasPrefix(low, "<html") {
		nav := buildFloatingNav(pageNum, total)
		if idx := strings.LastIndex(low, "</body>"); idx >= 0 {
			return t[:idx] + nav + t[idx:]
		}
		return t + nav
	}

	// 常规片段：缩放外壳 + 翻页导航（框架感知）
	prev, next := "", ""
	if pageNum > 1 {
		prev = "p" + strconv.Itoa(pageNum-1) + ".html"
	}
	if pageNum < total {
		next = "p" + strconv.Itoa(pageNum+1) + ".html"
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"zh-CN\">\n<head>\n<meta charset=\"UTF-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>" + htmlEscape(cwTitle) + " · 第" + strconv.Itoa(pageNum) + "页</title>\n")
	b.WriteString("<style>\n" + cwOflPageCSS + "\n</style>\n</head>\n<body>\n")
	b.WriteString("<div id=\"cw-stage\">")
	b.WriteString(t)
	b.WriteString("</div>\n")
	b.WriteString(buildStageNav(prev, next))
	b.WriteString("\n")
	b.WriteString(buildPageScript(prev, next))
	b.WriteString("\n</body>\n</html>")
	return b.String()
}

// buildStageNav 页面底部翻页导航（默认隐藏；单独打开页面时由脚本显示）
func buildStageNav(prev, next string) string {
	var b strings.Builder
	b.WriteString(`<div class="cw-ofl-nav" id="cw-ofl-nav">`)
	if prev != "" {
		b.WriteString(`<button class="cw-ofl-btn" onclick="cwNav('prev')">← 上一页</button>`)
	} else {
		b.WriteString(`<span class="cw-ofl-btn cw-ofl-disabled">← 上一页</span>`)
	}
	if next != "" {
		b.WriteString(`<button class="cw-ofl-btn" onclick="cwNav('next')">下一页 →</button>`)
	} else {
		b.WriteString(`<span class="cw-ofl-btn cw-ofl-disabled">下一页 →</span>`)
	}
	b.WriteString(`<button class="cw-ofl-btn" onclick="cwFull()">⛶ 全屏</button>`)
	b.WriteString(`</div>`)
	return b.String()
}

// buildPageScript 页面脚本：缩放自适应 + 框架感知的翻页/全屏 + 键盘
//   - 在 iframe 播放器里（framed）：翻页/全屏通过 postMessage 转发给外壳，
//     并隐藏自身导航（外壳已提供），从而保证全屏不丢、风格统一。
//   - 单独打开页面时：显示自身导航，直接跳转/自行全屏。
func buildPageScript(prev, next string) string {
	return "<script>\n(function(){\n" +
		"var framed=(window.self!==window.top);\n" +
		"function fit(){var s=Math.min(window.innerWidth/1920,window.innerHeight/1080);var el=document.getElementById('cw-stage');if(el){el.style.transform='translate(-50%,-50%) scale('+s+')';}}\n" +
		"window.addEventListener('resize',fit);window.addEventListener('load',fit);fit();\n" +
		"var PREV='" + prev + "',NEXT='" + next + "';\n" +
		"window.cwNav=function(dir){if(framed){window.parent.postMessage({__cwNav:dir},'*');}else{var h=(dir==='next'?NEXT:PREV);if(h)location.href=h;}};\n" +
		"window.cwFull=function(){if(framed){window.parent.postMessage({__cwFull:1},'*');return;}var el=document.documentElement;if(!document.fullscreenElement&&!document.webkitFullscreenElement){(el.requestFullscreen||el.webkitRequestFullscreen||function(){}).call(el);}else{(document.exitFullscreen||document.webkitExitFullscreen||function(){}).call(document);}};\n" +
		"var nav=document.getElementById('cw-ofl-nav');if(nav&&!framed){nav.style.display='flex';}\n" +
		"document.addEventListener('keydown',function(e){\n" +
		"if(e.key==='ArrowRight'||e.key==='PageDown'||e.key===' '){e.preventDefault();window.cwNav('next');}\n" +
		"else if(e.key==='ArrowLeft'||e.key==='PageUp'){e.preventDefault();window.cwNav('prev');}\n" +
		"else if(e.key==='f'||e.key==='F'){window.cwFull();}\n" +
		"});\n})();\n</script>"
}

// buildFloatingNav 给「完整文档型」页面（3D/导入）注入的框架感知浮动导航
func buildFloatingNav(pageNum, total int) string {
	prev, next := "", ""
	if pageNum > 1 {
		prev = "p" + strconv.Itoa(pageNum-1) + ".html"
	}
	if pageNum < total {
		next = "p" + strconv.Itoa(pageNum+1) + ".html"
	}
	var b strings.Builder
	b.WriteString(`<style>#cw-ofl-nav{display:none;position:fixed;bottom:16px;left:50%;transform:translateX(-50%);align-items:center;gap:10px;z-index:2147483647;background:rgba(15,23,42,0.8);padding:8px 14px;border-radius:999px;}#cw-ofl-nav button,#cw-ofl-nav span{padding:6px 14px;border-radius:999px;border:none;color:#fff;font-size:14px;background:rgba(255,255,255,0.16);cursor:pointer;}#cw-ofl-nav .d{opacity:.35;cursor:default;}</style>`)
	b.WriteString(`<div id="cw-ofl-nav">`)
	if prev != "" {
		b.WriteString(`<button onclick="cwNav('prev')">← 上一页</button>`)
	} else {
		b.WriteString(`<span class="d">← 上一页</span>`)
	}
	if next != "" {
		b.WriteString(`<button onclick="cwNav('next')">下一页 →</button>`)
	} else {
		b.WriteString(`<span class="d">下一页 →</span>`)
	}
	b.WriteString(`<button onclick="cwFull()">⛶ 全屏</button>`)
	b.WriteString(`</div>`)
	b.WriteString(buildPageScript(prev, next))
	return b.String()
}

// 播放器外壳 CSS
const cwOflShellCSS = `*{margin:0;padding:0;box-sizing:border-box;}
html,body{width:100%;height:100%;overflow:hidden;background:#0f172a;font-family:'PingFang SC','Microsoft YaHei',system-ui,sans-serif;}
#cw-frame{position:fixed;inset:0;width:100%;height:100%;border:0;}
.cw-shell-nav{position:fixed;bottom:16px;left:50%;transform:translateX(-50%);display:flex;align-items:center;gap:10px;z-index:2147483647;background:rgba(15,23,42,0.8);padding:8px 14px;border-radius:999px;box-shadow:0 4px 20px rgba(0,0,0,0.35);}
.cw-shell-nav button{padding:6px 14px;border-radius:999px;border:none;background:rgba(255,255,255,0.14);color:#fff;font-size:14px;cursor:pointer;}
.cw-shell-nav button:hover{background:rgba(255,255,255,0.26);}
.cw-shell-nav button:disabled{opacity:0.35;cursor:default;}
.cw-shell-count{color:#fff;font-size:14px;min-width:54px;text-align:center;}`

// buildOfflineIndexDoc 入口页 = iframe 播放器外壳
//   - 首屏即真实第 1 页（p1.html），外观与课件一致（解决"风格不一致"）。
//   - 翻页只换 iframe 的 src；全屏作用于外壳，故翻页时全屏不丢（解决"跳来跳去"）。
//   - v3：message 监听同时兼容新协议 {__cwNav/__cwFull} 与旧协议 {action:'navigate',direction}，
//     使封面/导入型页面里内联的 postMessage({action:'navigate',direction:'next'}) 也能翻页。
func buildOfflineIndexDoc(cwTitle string, total int) string {
	tot := strconv.Itoa(total)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"zh-CN\">\n<head>\n<meta charset=\"UTF-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>" + htmlEscape(cwTitle) + "</title>\n")
	b.WriteString("<style>\n" + cwOflShellCSS + "\n</style>\n</head>\n<body>\n")
	b.WriteString(`<iframe id="cw-frame" src="p1.html"></iframe>` + "\n")
	b.WriteString(`<div class="cw-shell-nav">`)
	b.WriteString(`<button id="cw-prev">← 上一页</button>`)
	b.WriteString(`<span class="cw-shell-count" id="cw-count">1 / ` + tot + `</span>`)
	b.WriteString(`<button id="cw-next">下一页 →</button>`)
	b.WriteString(`<button id="cw-full">⛶ 全屏</button>`)
	b.WriteString(`</div>` + "\n")
	b.WriteString("<script>\n(function(){\n")
	b.WriteString("var total=" + tot + ",cur=1;\n")
	b.WriteString("var frame=document.getElementById('cw-frame');\n")
	b.WriteString("var prevBtn=document.getElementById('cw-prev'),nextBtn=document.getElementById('cw-next'),cnt=document.getElementById('cw-count');\n")
	b.WriteString("function upd(){cnt.textContent=cur+' / '+total;prevBtn.disabled=(cur<=1);nextBtn.disabled=(cur>=total);}\n")
	b.WriteString("function go(n){if(n<1||n>total)return;cur=n;frame.src='p'+n+'.html';upd();}\n")
	b.WriteString("function toggleFull(){var el=document.documentElement;if(!document.fullscreenElement&&!document.webkitFullscreenElement){(el.requestFullscreen||el.webkitRequestFullscreen||function(){}).call(el);}else{(document.exitFullscreen||document.webkitExitFullscreen||function(){}).call(document);}}\n")
	b.WriteString("prevBtn.onclick=function(){go(cur-1);};nextBtn.onclick=function(){go(cur+1);};\n")
	b.WriteString("document.getElementById('cw-full').onclick=toggleFull;\n")
	// v3：兼容新旧两种翻页协议
	b.WriteString("window.addEventListener('message',function(e){\n")
	b.WriteString("var d=e.data;if(!d||typeof d!=='object')return;\n")
	b.WriteString("// 新协议（本系统页面脚本）\n")
	b.WriteString("if(d.__cwNav==='next'){go(cur+1);return;}\n")
	b.WriteString("if(d.__cwNav==='prev'){go(cur-1);return;}\n")
	b.WriteString("if(d.__cwFull){toggleFull();return;}\n")
	b.WriteString("// 旧协议（封面/导入型 CTA 按钮）：{action:'navigate',direction:'next'|'prev'}\n")
	b.WriteString("if(d.action==='navigate'){if(d.direction==='prev'){go(cur-1);}else{go(cur+1);}return;}\n")
	b.WriteString("// 兜底：部分页面只发 {action:'next'|'prev'} 或 {type:'nav',...}\n")
	b.WriteString("if(d.action==='next'){go(cur+1);return;}\n")
	b.WriteString("if(d.action==='prev'){go(cur-1);return;}\n")
	b.WriteString("});\n")
	b.WriteString("window.addEventListener('keydown',function(e){if(e.key==='ArrowRight'||e.key==='PageDown'||e.key===' '){e.preventDefault();go(cur+1);}else if(e.key==='ArrowLeft'||e.key==='PageUp'){e.preventDefault();go(cur-1);}else if(e.key==='f'||e.key==='F'){toggleFull();}});\n")
	b.WriteString("upd();\n})();\n</script>\n</body>\n</html>")
	return b.String()
}

// buildOfflineReadme 使用说明（纯文本）
func buildOfflineReadme(cwTitle string, total int) string {
	var b strings.Builder
	b.WriteString("课件离线包使用说明\n====================\n\n")
	b.WriteString("课件名称：" + cwTitle + "\n")
	b.WriteString("页面数量：" + strconv.Itoa(total) + " 页\n\n")
	b.WriteString("【如何打开】\n")
	b.WriteString("1. 将本文件夹完整拷贝到电脑任意位置（U盘、桌面均可）。\n")
	b.WriteString("2. 双击文件夹内的 index.html，用浏览器（推荐 Chrome / Edge）打开。\n")
	b.WriteString("   index.html 是播放器，首屏即第 1 页，外观与课件一致。\n\n")
	b.WriteString("【如何翻页 / 全屏】\n")
	b.WriteString("· 点击底部「上一页 / 下一页」按钮，或用键盘 → / 空格（下一页）、←（上一页）。\n")
	b.WriteString("· 先点一次「⛶ 全屏」按钮（或按 F）进入全屏，之后翻页会一直保持全屏，不会退出。\n")
	b.WriteString("· 按 Esc 退出全屏。\n\n")
	b.WriteString("【文件夹结构】\n")
	b.WriteString("· index.html        播放器入口（双击它）\n")
	b.WriteString("· p1.html ~ pN.html   各页课件（也可单独打开）\n")
	b.WriteString("· assets/            课件用到的图片、视频、音频\n\n")
	b.WriteString("【离线说明】\n")
	b.WriteString("· 图片、视频、音频已全部打包，断网可正常显示与播放。\n")
	b.WriteString("· 若课件含「AI 对话」「语音合成(TTS)」等需联网的智能互动，断网时不可用，其余内容不受影响。\n")
	b.WriteString("· 3D 互动课件依赖在线 3D 引擎，断网时可能无法显示，请联网使用。\n")
	return b.String()
}

// ==================== 通用小工具 ====================

// sanitizeBundleName 清洗课件标题为合法的文件夹名（保留中文）
func sanitizeBundleName(name string) string {
	name = strings.TrimSpace(name)
	repl := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
		"\n", " ", "\r", " ", "\t", " ",
	)
	name = repl.Replace(name)
	name = strings.TrimSpace(name)
	r := []rune(name)
	if len(r) > 80 {
		name = string(r[:80])
	}
	return strings.Trim(name, ". ")
}

// htmlEscape 转义文本中的 HTML 特殊字符（用于写入 <title> 等文本位置）
func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}
