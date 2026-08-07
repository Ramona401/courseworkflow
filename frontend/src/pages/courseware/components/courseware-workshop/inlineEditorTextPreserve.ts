/**
 * inlineEditorTextPreserve.ts — 就地编辑器文字内容替换的格式保留运行时代码
 *
 * 背景：
 * 旧实现使用 element.textContent = newText 替换整段文字。该赋值会删除元素内部
 * 所有 strong、span、em、b、i、font 等行内节点，导致原有加粗、颜色、字体等
 * 格式在第一次修改后整体丢失。
 *
 * 本模块不直接访问父页面DOM，而是生成一段注入 iframe 的纯 JavaScript 运行时代码。
 * 运行时算法只更新既有 Text 节点的 nodeValue，不删除、不新建、不重排行内元素，
 * 因而能够保留原DOM标签、class、style、事件属性与格式边界。
 *
 * 算法概要：
 * 1. 按DOM顺序收集目标元素内全部文本节点，并记录每个节点在旧全文中的结束偏移；
 * 2. 计算旧文本和新文本的最长公共前缀、最长公共后缀，定位真实改动区；
 * 3. 改动区前的格式边界保持原位，改动区后的边界按字符增减量平移；
 * 4. 落在改动区内部的边界按比例映射，保证跨多个格式片段修改时仍保留全部节点；
 * 5. 按新边界把新文本重新写入原文本节点，最后一个节点接收剩余内容。
 *
 * 该方案适合改错字、增删词句和整段重写。即使整段完全替换，也会保留原有行内
 * 格式节点，并按原格式片段的大致比例分配新文字，不再退化为单一纯文本节点。
 */

/**
 * buildInlineTextPreserveRuntime 返回可直接放入 iframe <script> 的运行时代码。
 *
 * 使用静态字符串而不是 Function.toString()，避免不同构建目标对函数源码序列化
 * 产生差异；代码仅使用 ES5 级 var/function 语法，兼容课件 iframe 运行环境。
 */
export function buildInlineTextPreserveRuntime(): string {
  return `
  /* ===== v5.8 文字替换格式保留 ===== */

  /**
   * 收集目标元素内按DOM顺序排列的文本节点。
   *
   * 编辑器自身注入的节点带 data-tedna-editor 标记，必须排除；
   * script/style/noscript 内容不是教师可编辑正文，也必须排除。
   */
  function collectTednaEditableTextNodes(root) {
    var nodes = [];
    if (!root) return nodes;

    var walker = document.createTreeWalker(
      root,
      NodeFilter.SHOW_TEXT,
      null
    );

    var node = walker.nextNode();
    while (node) {
      var parentEl = node.parentElement;
      var skip = false;

      if (!parentEl) {
        skip = true;
      } else {
        var tag = parentEl.tagName;
        if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'NOSCRIPT') {
          skip = true;
        }

        if (!skip && parentEl.closest && parentEl.closest('[data-tedna-editor]')) {
          skip = true;
        }
      }

      if (!skip) nodes.push(node);
      node = walker.nextNode();
    }

    return nodes;
  }

  /** 计算两个字符串的最长公共前缀长度（UTF-16偏移，与Text.nodeValue一致）。 */
  function tednaCommonPrefixLength(a, b) {
    var max = Math.min(a.length, b.length);
    var i = 0;
    while (i < max && a.charCodeAt(i) === b.charCodeAt(i)) i++;
    return i;
  }

  /**
   * 计算最长公共后缀长度。
   *
   * prefixLength用于防止前后缀区间重叠，确保改动区起止偏移合法。
   */
  function tednaCommonSuffixLength(a, b, prefixLength) {
    var max = Math.min(a.length, b.length) - prefixLength;
    var i = 0;
    while (
      i < max
      && a.charCodeAt(a.length - 1 - i) === b.charCodeAt(b.length - 1 - i)
    ) {
      i++;
    }
    return i;
  }

  /**
   * 防止映射后的格式边界落在UTF-16代理对中间。
   *
   * 中文普通字符不受影响；包含Emoji等扩展字符时，边界会向后移动一个码元，
   * 避免把一个字符拆成两个无效半字符。
   */
  function normalizeTednaTextBoundary(text, index) {
    var next = Math.max(0, Math.min(text.length, index));
    if (next <= 0 || next >= text.length) return next;

    var previousCode = text.charCodeAt(next - 1);
    var currentCode = text.charCodeAt(next);
    var previousIsHighSurrogate = previousCode >= 0xD800 && previousCode <= 0xDBFF;
    var currentIsLowSurrogate = currentCode >= 0xDC00 && currentCode <= 0xDFFF;

    if (previousIsHighSurrogate && currentIsLowSurrogate) {
      return next + 1;
    }
    return next;
  }

  /**
   * 在不破坏行内DOM结构的前提下替换目标元素的可见文本。
   *
   * 与 element.textContent = newText 的关键差异：
   * - 不删除任何子元素；
   * - 不删除 strong/span/em 等格式节点；
   * - 不修改 class/style/id/事件属性；
   * - 仅重写原有文本节点的 nodeValue。
   */
  function replaceTednaTextPreservingInlineFormat(root, newText) {
    if (!root || typeof newText !== 'string') return false;

    var textNodes = collectTednaEditableTextNodes(root);
    if (textNodes.length === 0) {
      root.appendChild(document.createTextNode(newText));
      return true;
    }

    var oldTextParts = [];
    var oldBoundaries = [];
    var oldLength = 0;

    for (var i = 0; i < textNodes.length; i++) {
      var value = textNodes[i].nodeValue || '';
      oldTextParts.push(value);
      oldLength += value.length;
      oldBoundaries.push(oldLength);
    }

    var oldText = oldTextParts.join('');
    if (oldText === newText) return false;

    var prefixLength = tednaCommonPrefixLength(oldText, newText);
    var suffixLength = tednaCommonSuffixLength(
      oldText,
      newText,
      prefixLength
    );

    var oldChangeStart = prefixLength;
    var oldChangeEnd = oldText.length - suffixLength;
    var newChangeEnd = newText.length - suffixLength;
    var oldChangeLength = oldChangeEnd - oldChangeStart;
    var newChangeLength = newChangeEnd - oldChangeStart;
    var totalDelta = newText.length - oldText.length;

    var previousNewBoundary = 0;

    for (var n = 0; n < textNodes.length; n++) {
      var mappedBoundary;

      if (n === textNodes.length - 1) {
        mappedBoundary = newText.length;
      } else {
        var oldBoundary = oldBoundaries[n];

        if (oldBoundary <= oldChangeStart) {
          // 改动区之前的格式边界保持原位置。
          mappedBoundary = oldBoundary;
        } else if (oldBoundary >= oldChangeEnd) {
          // 改动区之后的格式边界整体按字符增减量平移。
          mappedBoundary = oldBoundary + totalDelta;
        } else if (oldChangeLength > 0) {
          // 改动区内部的格式边界按相对位置映射到新改动区。
          var ratio = (oldBoundary - oldChangeStart) / oldChangeLength;
          mappedBoundary = oldChangeStart + Math.round(ratio * newChangeLength);
        } else {
          // 纯插入场景理论上不会进入此分支，保守落在插入点。
          mappedBoundary = oldChangeStart;
        }

        mappedBoundary = normalizeTednaTextBoundary(
          newText,
          mappedBoundary
        );

        mappedBoundary = Math.max(
          previousNewBoundary,
          Math.min(newText.length, mappedBoundary)
        );
      }

      textNodes[n].nodeValue = newText.slice(
        previousNewBoundary,
        mappedBoundary
      );
      previousNewBoundary = mappedBoundary;
    }

    return true;
  }
`
}

