/**
 * 课件预览降级注入 (previewInject.ts)
 *
 * 从 CoursewareWorkshopPage.tsx 抽出。TE-DNA 工坊 iframe 预览中，课件 HTML 调用 /api/*
 * 会失败(edu 平台 API 不可用)。本模块为 srcDoc 注入脚本：拦截 /api/* fetch 返回降级 JSON、
 * 归零 html/body margin 消除缩放白缝、置 window.__TEDNA_PREVIEW_MODE__=true 供课件 JS 检测。
 * 所有预览/全屏/放映 iframe 的 srcDoc 统一经 injectPreviewMode 处理。
 */

/** 注入到课件 HTML 的降级脚本（margin 归零 + fetch 拦截 + 预览模式标记） */
export const PREVIEW_INJECT_SCRIPT = `
<style>
/* 强制归零 iframe 内 html/body 默认 margin（浏览器默认 body margin:8px，
   经 scale 缩小后会在课件左/上侧露出约 1-2px 画布底色缝隙）。
   课件内容本就按 1920x1080 从 (0,0) 铺满，归零后缝隙消除。 */
html, body { margin: 0 !important; padding: 0 !important; }
</style>
<script>
// TE-DNA 预览模式标记
window.__TEDNA_PREVIEW_MODE__ = true;

// 拦截 fetch，对 /api/* 请求返回降级响应
(function() {
  var originalFetch = window.fetch;
  window.fetch = function(url, options) {
    if (typeof url === 'string' && url.startsWith('/api/')) {
      console.log('[预览模式] API 调用已降级:', url);
      return Promise.resolve(new Response(JSON.stringify({
        success: false,
        _preview_mode: true,
        message: '预览模式下 API 不可用，请在授课平台查看完整效果',
        data: { content: '预览模式：AI功能需在授课平台使用', audio_url: '' }
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      }));
    }
    return originalFetch.call(this, url, options);
  };
})();
</script>
`

/**
 * injectPreviewMode — 为课件 HTML 注入预览降级脚本
 * 在 </head> 或第一个 <script 前注入，确保在课件 JS 执行前生效；
 * 纯展示页(无 /api/ 调用)注入也无副作用。三级兜底：</head> → <script → 最前。
 */
export function injectPreviewMode(html: string): string {
  if (!html || !html.trim()) return html

  // 策略1: 如果有 </head>，在其前面注入
  const headClose = html.indexOf('</head>')
  if (headClose >= 0) {
    return html.slice(0, headClose) + PREVIEW_INJECT_SCRIPT + html.slice(headClose)
  }

  // 策略2: 如果有 <script，在第一个 <script 前注入
  const firstScript = html.indexOf('<script')
  if (firstScript >= 0) {
    return html.slice(0, firstScript) + PREVIEW_INJECT_SCRIPT + html.slice(firstScript)
  }

  // 策略3: 在HTML最前面注入（兜底）
  return PREVIEW_INJECT_SCRIPT + html
}
