/**
 * inlineEditorInject.ts — 就地编辑器的 iframe 注入脚本生成模块
 *
 * 独立抽出，降低 InlineTextEditor.tsx 体量。
 * 导出两个函数：
 *   buildEditorInject(token, isFullDoc) → 注入脚本 HTML 字符串
 *   injectEditor(html, token, isFullDoc) → 注入后的完整 HTML
 *
 * 支持的能力：
 *   - 文字元素选中（放宽版 isEditableTextEl，支持含行内格式子元素）
 *   - 图片选中（含视频占位智能检测 isVideoPlaceholder）
 *   - 视频 <video> 选中
 *   - 可拖拽块选中（绝对定位元素，含8点手柄拖拽缩放）
 *   - mouseup 文字选区检测 + savedRange 保存
 *   - 指令处理：apply / format / replace_image / resize_image /
 *               replace_video / replace_img_with_video / update_block / export
 *
 * 【v5.6 热点可选修复】block 模式点击判定增加向上冒泡查找（最多 5 层），
 *   让点到热点圆点子元素时能识别到父级绝对定位容器作为可拖拽块。
 *
 * 【v5.7 四模式互切修复】解决"选了模块切不出去 / 模块内的文字选不中改不了"：
 *   1. 点选改为「就近优先」统一向上查找 resolveSelectable：从点击目标逐层
 *      向上（最多5层），每层先判文字再判模块，谁先命中谁赢——模块内部的
 *      文字元素离点击点更近，天然优先选中文字；点在模块壳/装饰圆点上才
 *      选中模块。（v5.6 的"模块冒泡永远抢先"是切不出去的根因之一）
 *   2. 同一元素既满足文字又满足模块（如绝对定位的纯文字标签）：首次点击
 *      选为模块（保留拖拽定位能力），在已选中该模块的状态下再点一次自动
 *      切换为文字编辑（双击语义的单击版，兼顾两种能力）。
 *   3. 模块选中后的内部 mousedown 增加 4px 拖拽死区：位移未超过死区视为
 *      点击（click 正常触发选中切换），超过才真正移动模块；真正发生过
 *      拖拽移动的那一次 click 被抑制（suppressNextClick），防止拖完松手
 *      时误切换选中目标。
 *   4. 点击空白处（未命中任何可选元素）或按 ESC → 清除全部选中并向外层
 *      post('deselect')，外层面板回到未选中提示态。
 *   5. hover 提示同步改为就近优先，且每次 mouseover 先全局清一遍 hover
 *      标记，杜绝残留虚线框。
 */

/**
 * 生成注入 iframe 的完整编辑脚本（CSS + JS）
 * @param token  通信令牌（用于 postMessage 校验）
 * @param isFullDoc  课件 HTML 是否为完整文档（含 <html>）
 */
export function buildEditorInject(token: string, isFullDoc: boolean): string {
  return `
<style data-tedna-editor="1">
  html, body { margin: 0 !important; padding: 0 !important; }
  /* 文字 hover/selected */
  [data-tedna-hover] { outline: 2px dashed rgba(124,58,237,0.5) !important; outline-offset: 1px !important; cursor: pointer !important; }
  [data-tedna-selected] { outline: 2px solid #7C3AED !important; outline-offset: 1px !important; }
  /* 图片 hover/selected */
  [data-tedna-img-hover] { outline: 2px dashed rgba(14,165,233,0.6) !important; outline-offset: 2px !important; cursor: pointer !important; }
  [data-tedna-img-selected] { outline: 3px solid #0EA5E9 !important; outline-offset: 2px !important; }
  /* 视频 hover/selected */
  [data-tedna-video-hover] { outline: 2px dashed rgba(249,115,22,0.6) !important; outline-offset: 2px !important; cursor: pointer !important; }
  [data-tedna-video-selected] { outline: 3px solid #F59E0B !important; outline-offset: 2px !important; }
  /* 可拖拽块 hover/selected */
  [data-tedna-block-hover] { outline: 2px dashed rgba(16,185,129,0.5) !important; outline-offset: 1px !important; cursor: move !important; }
  [data-tedna-block-selected] { outline: 2px solid #10B981 !important; outline-offset: 1px !important; }
  /* 拖拽手柄：8个控制点 */
  .tedna-handle { position:absolute !important; width:10px !important; height:10px !important; background:#10B981 !important; border:1px solid #fff !important; border-radius:2px !important; z-index:99999 !important; pointer-events:auto !important; box-sizing:border-box !important; }
  .tedna-handle-tl { top:-5px !important; left:-5px !important; cursor:nw-resize !important; }
  .tedna-handle-tc { top:-5px !important; left:50% !important; margin-left:-5px !important; cursor:n-resize !important; }
  .tedna-handle-tr { top:-5px !important; right:-5px !important; cursor:ne-resize !important; }
  .tedna-handle-ml { top:50% !important; left:-5px !important; margin-top:-5px !important; cursor:w-resize !important; }
  .tedna-handle-mr { top:50% !important; right:-5px !important; margin-top:-5px !important; cursor:e-resize !important; }
  .tedna-handle-bl { bottom:-5px !important; left:-5px !important; cursor:sw-resize !important; }
  .tedna-handle-bc { bottom:-5px !important; left:50% !important; margin-left:-5px !important; cursor:s-resize !important; }
  .tedna-handle-br { bottom:-5px !important; right:-5px !important; cursor:se-resize !important; }
</style>
<script data-tedna-editor="1">
(function() {
  var TOKEN = ${JSON.stringify(token)};
  var IS_FULL_DOC = ${isFullDoc ? 'true' : 'false'};

  /* 预览降级：拦截 /api/ 请求 */
  var _fetch = window.fetch;
  window.fetch = function(url, opt) {
    if (typeof url === 'string' && url.indexOf('/api/') === 0) {
      return Promise.resolve(new Response(JSON.stringify({ success:false, _preview_mode:true, data:{} }), { status:200, headers:{'Content-Type':'application/json'} }));
    }
    return _fetch.call(this, url, opt);
  };
  window.__TEDNA_PREVIEW_MODE__ = true;

  /* ===== 状态变量 ===== */
  var selectedEl = null;   // 当前选中的文字元素
  var selectedImg = null;  // 当前选中的图片元素
  var currentMode = '';    // 'text' | 'image' | 'video' | 'block'
  var savedRange = null;   // 保存用户拖选的 Range 对象（防焦点离开iframe后Selection被清空）

  /* v5.7: 拖拽死区与点击抑制 */
  var DRAG_THRESHOLD = 4;        // 拖拽死区（px）：位移未超过视为点击，超过才算拖拽
  var dragMoved = false;         // 本次拖拽是否真正发生过移动（超过死区）
  var suppressNextClick = false; // 真正拖拽移动后，抑制随之而来的那一次 click（防误切换选中）

  /* ===== 行内标签白名单 ===== */
  var INLINE_TAGS = {
    'BR':1,'SPAN':1,'STRONG':1,'EM':1,'B':1,'I':1,'U':1,'S':1,
    'A':1,'SUB':1,'SUP':1,'MARK':1,'SMALL':1,'ABBR':1,'CODE':1,
    'DEL':1,'INS':1,'Q':1,'CITE':1,'DFN':1,'KBD':1,'SAMP':1,'VAR':1,
    'RUBY':1,'RT':1,'RP':1,'BDI':1,'BDO':1,'WBR':1,'DATA':1,'TIME':1,
    'FONT':1,'LABEL':1
  };

  /* ===== 不可选的标签 ===== */
  var SKIP_TAGS = {
    'SCRIPT':1,'STYLE':1,'HTML':1,'BODY':1,'HEAD':1,'META':1,
    'LINK':1,'TITLE':1,'NOSCRIPT':1,'SVG':1,'CANVAS':1,
    'AUDIO':1,'IFRAME':1,'OBJECT':1,'EMBED':1
  };

  /* ===== 元素判定函数 ===== */

  /** 可编辑文字元素（放宽版：允许行内格式子元素） */
  function isEditableTextEl(el) {
    if (!el || el.nodeType !== 1) return false;
    if (el.hasAttribute && el.hasAttribute('data-tedna-editor')) return false;
    var tag = el.tagName;
    if (SKIP_TAGS[tag]) return false;
    if (tag === 'IMG' || tag === 'VIDEO') return false;
    var children = el.children;
    for (var i = 0; i < children.length; i++) {
      var childTag = children[i].tagName;
      if (!INLINE_TAGS[childTag]) return false;
    }
    var t = (el.textContent || '').trim();
    if (t.length === 0) return false;
    if (t.length > 500) return false;
    var rect = el.getBoundingClientRect();
    if (rect.width * rect.height > 1920 * 1080 * 0.4) return false;
    return true;
  }

  /** 可选中的图片 */
  function isEditableImg(el) {
    if (!el || el.nodeType !== 1) return false;
    if (el.tagName !== 'IMG') return false;
    if (el.hasAttribute && el.hasAttribute('data-tedna-editor')) return false;
    var rect = el.getBoundingClientRect();
    if (rect.width < 24 || rect.height < 24) return false;
    return true;
  }

  /** 可选中的视频 */
  function isEditableVideo(el) {
    if (!el || el.nodeType !== 1) return false;
    if (el.tagName === 'VIDEO') return true;
    if (el.tagName === 'SOURCE' && el.parentElement && el.parentElement.tagName === 'VIDEO') return true;
    return false;
  }
  function resolveVideoEl(el) {
    if (el.tagName === 'VIDEO') return el;
    if (el.tagName === 'SOURCE' && el.parentElement && el.parentElement.tagName === 'VIDEO') return el.parentElement;
    return null;
  }
  function getVideoSrc(videoEl) {
    if (videoEl.src) return videoEl.src;
    var sources = videoEl.querySelectorAll('source');
    for (var i = 0; i < sources.length; i++) { if (sources[i].src) return sources[i].src; }
    return '';
  }

  /** 可拖拽缩放的块元素（绝对定位 + 有明确宽高） */
  function isDraggableBlock(el) {
    if (!el || el.nodeType !== 1) return false;
    if (el.hasAttribute && el.hasAttribute('data-tedna-editor')) return false;
    var tag = el.tagName;
    if (SKIP_TAGS[tag]) return false;
    if (tag === 'IMG' || tag === 'VIDEO' || tag === 'SOURCE') return false;
    var cs = window.getComputedStyle(el);
    if (cs.position !== 'absolute' && cs.position !== 'fixed') return false;
    var rect = el.getBoundingClientRect();
    if (rect.width < 20 || rect.height < 20) return false;
    if (el === document.body || el === document.documentElement) return false;
    if (el.classList && el.classList.contains('cw-page')) return false;
    if (parseInt(cs.zIndex, 10) >= 999) return false;
    return true;
  }

  /**
   * 【v5.7 核心】就近优先的统一选中解析：
   * 从点击目标逐层向上（最多 maxUp 层），每层先判文字再判模块，
   * 谁先命中谁赢（离点击点最近的可选元素优先）。
   *
   * 同一元素既满足文字又满足模块（如绝对定位的纯文字标签）时：
   *   - 该元素当前已是选中模块（dragTarget）→ 返回文字（再点一次切换为文字编辑）
   *   - 否则 → 返回模块（首次点击保留拖拽定位能力）
   *
   * 返回 { kind: 'text' | 'block', el: Element } 或 null（未命中任何可选元素）
   */
  function resolveSelectable(el, maxUp) {
    var node = el;
    var depth = maxUp || 5;
    while (node && depth > 0) {
      var isTxt = isEditableTextEl(node);
      var isBlk = isDraggableBlock(node);
      if (isTxt && isBlk) {
        /* 双重身份元素：已选中该模块时再点一次 → 切换为文字编辑 */
        if (node === dragTarget && currentMode === 'block') {
          return { kind: 'text', el: node };
        }
        return { kind: 'block', el: node };
      }
      if (isTxt) return { kind: 'text', el: node };
      if (isBlk) return { kind: 'block', el: node };
      node = node.parentElement;
      depth--;
    }
    return null;
  }

  /* ===== 拖拽手柄管理 ===== */
  var dragHandles = [];
  var dragTarget = null;
  var dragType = '';
  var dragStartX = 0, dragStartY = 0;
  var dragStartLeft = 0, dragStartTop = 0, dragStartW = 0, dragStartH = 0;

  function removeHandles() {
    for (var h = 0; h < dragHandles.length; h++) {
      if (dragHandles[h].parentNode) dragHandles[h].parentNode.removeChild(dragHandles[h]);
    }
    dragHandles = [];
  }

  function addHandles(el) {
    removeHandles();
    var positions = ['tl','tc','tr','ml','mr','bl','bc','br'];
    for (var p = 0; p < positions.length; p++) {
      var hd = document.createElement('div');
      hd.className = 'tedna-handle tedna-handle-' + positions[p];
      hd.setAttribute('data-tedna-editor', '1');
      hd.setAttribute('data-handle', positions[p]);
      el.appendChild(hd);
      dragHandles.push(hd);
    }
  }

  /* ===== 路径计算 ===== */
  function getPath(el) {
    var parts = [];
    var node = el;
    while (node && node.nodeType === 1 && node !== document.documentElement) {
      var parent = node.parentNode;
      if (!parent) break;
      var idx = 0, found = -1;
      for (var i = 0; i < parent.childNodes.length; i++) {
        var c = parent.childNodes[i];
        if (c.nodeType === 1) { if (c === node) { found = idx; } idx++; }
      }
      parts.unshift(found);
      node = parent;
    }
    return parts.join('-');
  }
  function elByPath(path) {
    if (path === '') return document.documentElement;
    var parts = path.split('-').map(function(x){ return parseInt(x,10); });
    var node = document.documentElement;
    for (var p = 0; p < parts.length; p++) {
      var wantIdx = parts[p];
      var elIdx = 0, next = null;
      for (var i = 0; i < node.childNodes.length; i++) {
        var c = node.childNodes[i];
        if (c.nodeType === 1) { if (elIdx === wantIdx) { next = c; break; } elIdx++; }
      }
      if (!next) return null;
      node = next;
    }
    return node;
  }

  function post(type, payload) {
    parent.postMessage({ __tedna: TOKEN, type: type, payload: payload }, '*');
  }

  /* ===== 清除所有选中态 ===== */
  function clearAllSelection() {
    if (selectedEl) { selectedEl.removeAttribute('data-tedna-selected'); selectedEl = null; }
    if (selectedImg) { selectedImg.removeAttribute('data-tedna-img-selected'); selectedImg = null; }
    var allVidSel = document.querySelectorAll('[data-tedna-video-selected]');
    for (var v = 0; v < allVidSel.length; v++) allVidSel[v].removeAttribute('data-tedna-video-selected');
    var allBlkSel = document.querySelectorAll('[data-tedna-block-selected]');
    for (var b = 0; b < allBlkSel.length; b++) allBlkSel[b].removeAttribute('data-tedna-block-selected');
    removeHandles();
    dragTarget = null;
    currentMode = '';
    savedRange = null;
  }

  /* ===== v5.7: 全局清除所有 hover 标记（杜绝残留虚线框） ===== */
  function clearAllHovers() {
    var hovered = document.querySelectorAll('[data-tedna-hover],[data-tedna-img-hover],[data-tedna-video-hover],[data-tedna-block-hover]');
    for (var i = 0; i < hovered.length; i++) {
      hovered[i].removeAttribute('data-tedna-hover');
      hovered[i].removeAttribute('data-tedna-img-hover');
      hovered[i].removeAttribute('data-tedna-video-hover');
      hovered[i].removeAttribute('data-tedna-block-hover');
    }
  }

  /* ===== hover 提示（v5.7：先全清再按就近优先设置） ===== */
  document.addEventListener('mouseover', function(e) {
    var el = e.target;
    clearAllHovers();
    /* 手柄不做 hover 提示 */
    if (el.getAttribute && el.getAttribute('data-handle')) return;
    /* 图片 / 视频优先（直接命中判定） */
    if (isEditableImg(el)) {
      if (el !== selectedImg) el.setAttribute('data-tedna-img-hover', '1');
      return;
    }
    if (isEditableVideo(el)) {
      var vid = resolveVideoEl(el);
      if (vid) vid.setAttribute('data-tedna-video-hover', '1');
      return;
    }
    /* 文字 / 模块：就近优先解析（与点选逻辑完全一致） */
    var hit = resolveSelectable(el, 5);
    if (!hit) return;
    if (hit.kind === 'text') {
      if (hit.el !== selectedEl) hit.el.setAttribute('data-tedna-hover', '1');
    } else {
      if (hit.el !== dragTarget) hit.el.setAttribute('data-tedna-block-hover', '1');
    }
  }, true);

  document.addEventListener('mouseout', function() {
    /* v5.7: 简化为全局清除（mouseover 会重新设置正确的 hover） */
    clearAllHovers();
  }, true);

  /* ===== 点选（v5.7：就近优先 + 拖拽后抑制 + 空白取消选中） ===== */
  document.addEventListener('click', function(e) {
    /* v5.7: 刚发生过真实拖拽移动 → 抑制这一次 click，防止拖完松手误切换选中 */
    if (suppressNextClick) { suppressNextClick = false; return; }

    var el = e.target;
    /* 点在手柄上不做新选中 */
    if (el.getAttribute && el.getAttribute('data-handle')) return;

    /* 图片 */
    if (isEditableImg(el)) {
      e.preventDefault(); e.stopPropagation();
      clearAllSelection();
      el.removeAttribute('data-tedna-img-hover');
      el.setAttribute('data-tedna-img-selected', '1');
      selectedImg = el;
      currentMode = 'image';
      var rect = el.getBoundingClientRect();
      /* 检测是否为视频占位/首帧图 */
      var isVidPlaceholder = false;
      var checkEl = el.parentElement;
      var checkDepth = 5;
      while (checkEl && checkDepth > 0) {
        var checkHtml = checkEl.innerHTML || '';
        if (checkHtml.indexOf('暂停') >= 0 || checkHtml.indexOf('播放') >= 0 ||
            checkHtml.indexOf('play') >= 0 || checkHtml.indexOf('Play') >= 0 ||
            checkHtml.indexOf('playIcon') >= 0 || checkHtml.indexOf('controlPlay') >= 0 ||
            checkHtml.indexOf('video-poster') >= 0 || checkHtml.indexOf('video-frame') >= 0 ||
            checkHtml.indexOf('视频') >= 0 || checkHtml.indexOf('首帧') >= 0 ||
            checkHtml.indexOf('时长:') >= 0 || checkHtml.indexOf('时长：') >= 0) {
          isVidPlaceholder = true; break;
        }
        checkEl = checkEl.parentElement; checkDepth--;
      }
      post('select', {
        mode: 'image', path: getPath(el), src: el.src || '',
        width: Math.round(rect.width), height: Math.round(rect.height),
        naturalWidth: el.naturalWidth || 0, naturalHeight: el.naturalHeight || 0,
        isVideoPlaceholder: isVidPlaceholder
      });
      return;
    }

    /* 视频 */
    if (isEditableVideo(el)) {
      e.preventDefault(); e.stopPropagation();
      clearAllSelection();
      var vid = resolveVideoEl(el);
      if (!vid) return;
      vid.removeAttribute('data-tedna-video-hover');
      vid.setAttribute('data-tedna-video-selected', '1');
      currentMode = 'video';
      var vRect = vid.getBoundingClientRect();
      post('select', { mode: 'video', path: getPath(vid), src: getVideoSrc(vid), width: Math.round(vRect.width), height: Math.round(vRect.height) });
      return;
    }

    /*
     * 【v5.7 核心修改】文字 / 模块统一走就近优先解析：
     * 旧逻辑：模块冒泡判定在前，模块内部的文字永远选不中（切不出去的根因）。
     * 新逻辑：从点击目标逐层向上，每层先判文字再判模块，最近者胜——
     *   点模块内的文字 → 选中文字；点模块壳/装饰圆点 → 选中模块；
     *   双重身份元素首次点选模块，再点一次切换为文字编辑。
     */
    var hit = resolveSelectable(el, 5);

    if (hit && hit.kind === 'block') {
      e.preventDefault(); e.stopPropagation();
      clearAllSelection();
      var blockEl = hit.el;
      blockEl.removeAttribute('data-tedna-block-hover');
      blockEl.setAttribute('data-tedna-block-selected', '1');
      dragTarget = blockEl;
      currentMode = 'block';
      addHandles(blockEl);
      var bcs = window.getComputedStyle(blockEl);
      var bRect = blockEl.getBoundingClientRect();
      post('select', {
        mode: 'block', path: getPath(blockEl), tagName: blockEl.tagName,
        left: parseInt(bcs.left, 10) || 0, top: parseInt(bcs.top, 10) || 0,
        width: Math.round(bRect.width), height: Math.round(bRect.height),
        hasText: (blockEl.textContent || '').trim().length > 0,
        elementId: blockEl.id || '',
        elementClass: blockEl.className || ''
      });
      return;
    }

    if (hit && hit.kind === 'text') {
      e.preventDefault(); e.stopPropagation();
      clearAllSelection();
      var textEl = hit.el;
      textEl.removeAttribute('data-tedna-hover');
      textEl.setAttribute('data-tedna-selected', '1');
      selectedEl = textEl;
      currentMode = 'text';
      var cs = window.getComputedStyle(textEl);
      post('select', {
        mode: 'text', path: getPath(textEl), text: textEl.textContent || '',
        fontSizePx: Math.round(parseFloat(cs.fontSize) || 16),
        color: cs.color || 'rgb(0,0,0)', hasSelection: false
      });
      return;
    }

    /* v5.7: 未命中任何可选元素（点空白处）→ 清除选中并通知外层面板 */
    if (currentMode !== '') {
      clearAllSelection();
      post('deselect', {});
    }
  }, true);

  /* ===== v5.7: ESC 键取消选中 ===== */
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && currentMode !== '') {
      clearAllSelection();
      post('deselect', {});
    }
  }, true);

  /* ===== mouseup 检测文字选区并保存 Range ===== */
  document.addEventListener('mouseup', function() {
    if (currentMode !== 'text' || !selectedEl) return;
    setTimeout(function() {
      var sel = window.getSelection();
      savedRange = null;
      var has = false;
      if (sel && sel.rangeCount > 0 && !sel.isCollapsed) {
        var range = sel.getRangeAt(0);
        if (selectedEl.contains(range.commonAncestorContainer)) {
          has = true;
          savedRange = range.cloneRange();
        }
      }
      post('selection_change', { hasSelection: has });
    }, 10);
  }, true);

  /* ===== 拖拽事件处理（v5.7：加入 4px 死区判别点击 vs 拖拽） ===== */
  function onDragMouseDown(e) {
    var handleType = e.target.getAttribute && e.target.getAttribute('data-handle');
    if (handleType) {
      e.preventDefault(); e.stopPropagation();
      dragType = handleType;
    } else if (dragTarget && (e.target === dragTarget || dragTarget.contains(e.target)) && currentMode === 'block') {
      /* 点在已选中的块内部 → 潜在拖拽移动（死区内视为点击，click 正常触发切换选中） */
      e.preventDefault(); e.stopPropagation();
      dragType = 'move';
    } else {
      return;
    }
    if (!dragTarget) return;
    dragMoved = false;
    var cs = window.getComputedStyle(dragTarget);
    dragStartX = e.clientX; dragStartY = e.clientY;
    dragStartLeft = parseInt(cs.left, 10) || 0;
    dragStartTop = parseInt(cs.top, 10) || 0;
    dragStartW = parseInt(cs.width, 10) || dragTarget.getBoundingClientRect().width;
    dragStartH = parseInt(cs.height, 10) || dragTarget.getBoundingClientRect().height;
    document.addEventListener('mousemove', onDragMouseMove, true);
    document.addEventListener('mouseup', onDragMouseUp, true);
  }

  function onDragMouseMove(e) {
    if (!dragTarget || !dragType) return;
    e.preventDefault();
    var dx = e.clientX - dragStartX, dy = e.clientY - dragStartY;
    /* v5.7: 死区判别——位移未超过阈值不算拖拽，不动元素（松手后 click 正常处理切换选中） */
    if (!dragMoved) {
      if (Math.abs(dx) < DRAG_THRESHOLD && Math.abs(dy) < DRAG_THRESHOLD) return;
      dragMoved = true;
    }
    var newL = dragStartLeft, newT = dragStartTop, newW = dragStartW, newH = dragStartH;
    if (dragType === 'move') {
      newL = dragStartLeft + dx; newT = dragStartTop + dy;
    } else {
      if (dragType.indexOf('l') >= 0) { newL = dragStartLeft + dx; newW = dragStartW - dx; }
      if (dragType.indexOf('r') >= 0) { newW = dragStartW + dx; }
      if (dragType === 'tl' || dragType === 'tc') { newT = dragStartTop + dy; newH = dragStartH - dy; }
      if (dragType === 'tr') { newT = dragStartTop + dy; newH = dragStartH - dy; newW = dragStartW + dx; }
      if (dragType.indexOf('b') >= 0) { newH = dragStartH + dy; }
      if (newW < 30) { newW = 30; if (dragType.indexOf('l') >= 0) newL = dragStartLeft + dragStartW - 30; }
      if (newH < 30) { newH = 30; if (dragType === 'tl' || dragType === 'tc' || dragType === 'tr') newT = dragStartTop + dragStartH - 30; }
    }
    dragTarget.style.left = newL + 'px'; dragTarget.style.top = newT + 'px';
    dragTarget.style.width = newW + 'px'; dragTarget.style.height = newH + 'px';
    post('block_update', { left: newL, top: newT, width: newW, height: newH });
  }

  function onDragMouseUp() {
    document.removeEventListener('mousemove', onDragMouseMove, true);
    document.removeEventListener('mouseup', onDragMouseUp, true);
    /* v5.7: 真正发生过拖拽移动 → 抑制随之而来的 click（防拖完松手误切换）
       兜底：若 click 因松手位置特殊未触发，250ms 后自动复位标志防吃掉下一次点击 */
    if (dragMoved) {
      suppressNextClick = true;
      setTimeout(function() { suppressNextClick = false; }, 250);
    }
    dragMoved = false;
    dragType = '';
  }
  document.addEventListener('mousedown', onDragMouseDown, true);

  /* ===== 接收外层指令 ===== */
  window.addEventListener('message', function(e) {
    var d = e.data;
    if (!d || d.__tedna !== TOKEN) return;

    if (d.type === 'apply') {
      var el = elByPath(d.payload.path);
      if (!el) return;
      if (typeof d.payload.text === 'string') el.textContent = d.payload.text;
      if (d.payload.fontSizePx) el.style.fontSize = d.payload.fontSizePx + 'px';
      if (d.payload.color) el.style.color = d.payload.color;

    } else if (d.type === 'format') {
      if (!savedRange) return;
      var range = savedRange;
      if (!selectedEl || !selectedEl.contains(range.commonAncestorContainer)) return;
      var action = d.payload.action;
      if (action === 'bold') {
        var parentStrong = range.commonAncestorContainer;
        while (parentStrong && parentStrong !== selectedEl) {
          if (parentStrong.nodeType === 1 && (parentStrong.tagName === 'STRONG' || parentStrong.tagName === 'B')) break;
          parentStrong = parentStrong.parentNode;
        }
        if (parentStrong && parentStrong !== selectedEl && (parentStrong.tagName === 'STRONG' || parentStrong.tagName === 'B')) {
          var frag = document.createDocumentFragment();
          while (parentStrong.firstChild) frag.appendChild(parentStrong.firstChild);
          parentStrong.parentNode.insertBefore(frag, parentStrong);
          parentStrong.parentNode.removeChild(parentStrong);
        } else {
          try { var s1 = document.createElement('strong'); range.surroundContents(s1); }
          catch(ex) { try { var s2 = document.createElement('strong'); s2.appendChild(range.extractContents()); range.insertNode(s2); } catch(ex2) {} }
        }
        savedRange = null;
      } else if (action === 'color' && d.payload.value) {
        try { var cs1 = document.createElement('span'); cs1.style.color = d.payload.value; range.surroundContents(cs1); }
        catch(ex) { try { var cs2 = document.createElement('span'); cs2.style.color = d.payload.value; cs2.appendChild(range.extractContents()); range.insertNode(cs2); } catch(ex2) {} }
        savedRange = null;
      } else if (action === 'font' && d.payload.value) {
        try { var fs1 = document.createElement('span'); fs1.style.fontFamily = d.payload.value; range.surroundContents(fs1); }
        catch(ex) { try { var fs2 = document.createElement('span'); fs2.style.fontFamily = d.payload.value; fs2.appendChild(range.extractContents()); range.insertNode(fs2); } catch(ex2) {} }
        savedRange = null;
      }

    } else if (d.type === 'replace_image') {
      var img = elByPath(d.payload.path);
      if (!img || img.tagName !== 'IMG') return;
      if (d.payload.src) img.src = d.payload.src;

    } else if (d.type === 'replace_img_with_video') {
      var imgEl = elByPath(d.payload.path);
      if (!imgEl || imgEl.tagName !== 'IMG') return;
      var videoTag = document.createElement('video');
      videoTag.src = d.payload.src;
      videoTag.setAttribute('controls', '');
      videoTag.setAttribute('preload', 'metadata');
      videoTag.style.width = imgEl.style.width || (imgEl.getBoundingClientRect().width + 'px');
      videoTag.style.height = imgEl.style.height || (imgEl.getBoundingClientRect().height + 'px');
      videoTag.style.borderRadius = window.getComputedStyle(imgEl).borderRadius || '12px';
      videoTag.style.objectFit = 'cover'; videoTag.style.display = 'block';
      var imgParent = imgEl.parentElement;
      if (imgParent) {
        var sibCount = 0;
        for (var sc = 0; sc < imgParent.childNodes.length; sc++) { if (imgParent.childNodes[sc].nodeType === 1) sibCount++; }
        if (sibCount <= 4) { imgParent.innerHTML = ''; imgParent.appendChild(videoTag); }
        else { imgEl.parentNode.replaceChild(videoTag, imgEl); }
      } else { imgEl.parentNode.replaceChild(videoTag, imgEl); }

    } else if (d.type === 'resize_image') {
      var img2 = elByPath(d.payload.path);
      if (!img2 || img2.tagName !== 'IMG') return;
      if (d.payload.width > 0) img2.style.width = d.payload.width + 'px';
      if (d.payload.height > 0) img2.style.height = d.payload.height + 'px';

    } else if (d.type === 'replace_video') {
      var videoEl = elByPath(d.payload.path);
      if (!videoEl || videoEl.tagName !== 'VIDEO') return;
      var newSrc = d.payload.src;
      if (!newSrc) return;
      var sources = videoEl.querySelectorAll('source');
      if (sources.length > 0) {
        sources[0].src = newSrc;
        for (var si = 1; si < sources.length; si++) sources[si].parentNode.removeChild(sources[si]);
      } else { videoEl.src = newSrc; }
      videoEl.load();

    } else if (d.type === 'update_block') {
      var blk = elByPath(d.payload.path);
      if (!blk) return;
      if (d.payload.left !== undefined) blk.style.left = d.payload.left + 'px';
      if (d.payload.top !== undefined) blk.style.top = d.payload.top + 'px';
      if (d.payload.width !== undefined && d.payload.width >= 30) blk.style.width = d.payload.width + 'px';
      if (d.payload.height !== undefined && d.payload.height >= 30) blk.style.height = d.payload.height + 'px';

    } else if (d.type === 'export') {
      var injected = document.querySelectorAll('[data-tedna-editor]');
      for (var i = 0; i < injected.length; i++) injected[i].parentNode && injected[i].parentNode.removeChild(injected[i]);
      var marked = document.querySelectorAll('[data-tedna-hover],[data-tedna-selected],[data-tedna-img-hover],[data-tedna-img-selected],[data-tedna-video-hover],[data-tedna-video-selected],[data-tedna-block-hover],[data-tedna-block-selected]');
      for (var j = 0; j < marked.length; j++) {
        marked[j].removeAttribute('data-tedna-hover'); marked[j].removeAttribute('data-tedna-selected');
        marked[j].removeAttribute('data-tedna-img-hover'); marked[j].removeAttribute('data-tedna-img-selected');
        marked[j].removeAttribute('data-tedna-video-hover'); marked[j].removeAttribute('data-tedna-video-selected');
        marked[j].removeAttribute('data-tedna-block-hover'); marked[j].removeAttribute('data-tedna-block-selected');
      }
      var html = IS_FULL_DOC ? '<!DOCTYPE html>' + document.documentElement.outerHTML : document.body.innerHTML;
      post('exported', { html: html });
    }
  });

  post('ready', {});
})();
</script>
`
}

/**
 * 把编辑脚本注入课件 HTML（三级兜底注入位置）
 */
export function injectEditor(html: string, token: string, isFullDoc: boolean): string {
  const script = buildEditorInject(token, isFullDoc)
  const headClose = html.indexOf('</head>')
  if (headClose >= 0) return html.slice(0, headClose) + script + html.slice(headClose)
  const firstScript = html.indexOf('<script')
  if (firstScript >= 0) return html.slice(0, firstScript) + script + html.slice(firstScript)
  return script + html
}
