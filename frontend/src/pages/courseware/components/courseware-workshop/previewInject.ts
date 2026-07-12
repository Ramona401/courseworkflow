/**
 * 课件预览降级注入 (previewInject.ts)
 *
 * 从 CoursewareWorkshopPage.tsx 抽出。TE-DNA 工坊 iframe 预览中，课件 HTML 调用 /api/*
 * 会失败(edu 平台 API 不可用)。本模块为 srcDoc 注入脚本：拦截 /api/* fetch 返回降级 JSON、
 * 归零 html/body margin 消除缩放白缝、置 window.__TEDNA_PREVIEW_MODE__=true 供课件 JS 检测。
 * 所有预览/全屏/放映 iframe 的 srcDoc 统一经 injectPreviewMode 处理。
 *
 * 【v5.5 焦点穿透修复】iframe 内部用户点击互动元素后，浏览器键盘焦点移入 iframe，
 *   此后 keydown 事件在 iframe 内部触发不会冒泡到父 window，导致父组件的翻页键盘监听失效。
 *   修复方案：在注入脚本中增加 keydown 监听，把导航相关按键（方向键/ESC/空格/PageUp/PageDown）
 *   通过 parent.postMessage 转发到父窗口，父组件（SlideshowPlayer/CWFullscreenPreview）
 *   增加 message 监听器接收并执行翻页。消息格式 { type: '__tedna_nav_key', key: string }。
 *   同时在 iframe 内对这些导航键调用 preventDefault 防止 iframe 内部滚动。
 */

/** 需要从 iframe 转发到父窗口的导航按键集合（供父组件校验消息合法性） */
export const NAV_KEYS = new Set([
  'ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown',
  'PageUp', 'PageDown', 'Escape', ' ',
])

/** postMessage 的消息类型标识，父组件据此识别来自 iframe 的导航按键转发 */
export const NAV_KEY_MSG_TYPE = '__tedna_nav_key'

/** 注入到课件 HTML 的降级脚本（margin 归零 + fetch 拦截 + 预览模式标记 + 导航键转发） */
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

// 【v5.5 焦点穿透修复】iframe 内部导航按键转发到父窗口
// 用户点击课件互动元素后焦点进入 iframe，此后 keydown 不冒泡到父 window，
// 父组件的翻页键盘监听失效。这里捕获导航相关按键经 postMessage 转发给父窗口。
(function() {
  // 需要转发的按键集合
  var navKeys = {
    'ArrowLeft': true, 'ArrowRight': true, 'ArrowUp': true, 'ArrowDown': true,
    'PageUp': true, 'PageDown': true, 'Escape': true, ' ': true
  };
  document.addEventListener('keydown', function(e) {
    if (!navKeys[e.key]) return;
    // 如果用户正在 iframe 内的输入框/文本域中打字，不转发（允许空格等正常输入）
    var tag = e.target && e.target.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
    // 如果目标元素是 contentEditable，也不转发
    if (e.target && e.target.isContentEditable) return;
    // 阻止 iframe 内部默认行为（如空格导致页面滚动、方向键移动滚动条）
    e.preventDefault();
    // 转发给父窗口
    try {
      if (window.parent && window.parent !== window) {
        window.parent.postMessage({
          type: '__tedna_nav_key',
          key: e.key
        }, '*');
      }
    } catch(err) {
      // sandbox 限制下 postMessage 可能失败，静默忽略
    }
  }, true); // 使用捕获阶段，确保在课件自身的 keydown 处理之前拦截
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
