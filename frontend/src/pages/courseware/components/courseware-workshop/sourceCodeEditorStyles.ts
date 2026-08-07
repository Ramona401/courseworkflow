/**
 * sourceCodeEditorStyles.ts — 课件源码编辑器局部样式
 *
 * 样式随懒加载编辑器进入独立资源，不污染全局 CSS。
 *
 * 颜色重点：
 *   1. 大编辑区突出老师最常改的 HTML 正文；
 *   2. 右侧 Minimap 用紫、金、青、灰区分样式、内容、函数和注释；
 *   3. Minimap 顶部使用轻量伪元素图例，不增加 React 节点和运行状态。
 */
import {
  SOURCE_EDITOR_FONT,
  SOURCE_LINE_HEIGHT,
} from './sourceCodeEditorUtils'

export const SOURCE_EDITOR_CSS = `
.tedna-source-editor{position:relative;width:100%;overflow:hidden;border-radius:14px;background:#1e1e1e;font-family:${SOURCE_EDITOR_FONT}}
.tedna-source-editor.is-readonly{border:1px solid #374151}.tedna-source-editor.is-editing{border:2px solid #059669;box-shadow:0 0 0 3px rgba(5,150,105,.08)}
.tedna-source-grid{display:grid;min-height:260px}.tedna-source-lines{box-sizing:border-box;margin:0;padding:12px 9px 12px 4px;overflow:hidden;border-right:1px solid #333;background:#1e1e1e;color:#858585;text-align:right;user-select:none;font:12px/${SOURCE_LINE_HEIGHT}px ${SOURCE_EDITOR_FONT}}
.tedna-source-code-pane{position:relative;min-width:0;height:100%;overflow:hidden;background:#1e1e1e}.tedna-source-highlight,.tedna-source-code-pane textarea{box-sizing:border-box;position:absolute;inset:0;width:100%;height:100%;margin:0;padding:12px 14px;white-space:pre;tab-size:2;font:12px/${SOURCE_LINE_HEIGHT}px ${SOURCE_EDITOR_FONT}}
.tedna-source-highlight{z-index:1;overflow:hidden;border:0;background:#1e1e1e;color:#d4d4d4;pointer-events:none}.tedna-source-code-pane textarea{z-index:2;resize:none;overflow:auto;border:0;outline:0;background:transparent;color:transparent;-webkit-text-fill-color:transparent;caret-color:#f8fafc}.tedna-source-code-pane textarea::selection{background:rgba(38,121,196,.48)}.tedna-source-code-pane textarea.is-disabled{caret-color:transparent}
.tedna-token-plain{color:#d4d4d4}.tedna-token-tag{color:#569cd6;font-weight:600}.tedna-token-attribute{color:#9cdcfe}.tedna-token-string{color:#ce9178}.tedna-token-comment{color:#6a9955;font-style:italic}.tedna-token-content{color:#ffe08a;font-weight:600}.tedna-token-keyword{color:#c586c0;font-weight:600}.tedna-token-function{color:#4ec9b0}.tedna-token-number{color:#b5cea8}.tedna-token-property{color:#9cdcfe}.tedna-token-selector{color:#d7ba7d}.tedna-token-operator{color:#d4d4d4}
.tedna-source-match{background:rgba(250,204,21,.26);box-shadow:inset 0 -1px 0 rgba(250,204,21,.75);border-radius:2px}.tedna-source-match.is-current-match{background:rgba(251,146,60,.58);box-shadow:inset 0 0 0 1px rgba(255,237,213,.82)}
.tedna-source-minimap{position:relative;height:100%;overflow:hidden;border-left:1px solid #333;background:#171717;cursor:pointer;touch-action:none}.tedna-source-minimap canvas{position:relative;z-index:1;display:block;width:100%;height:100%}
.tedna-source-minimap::before{content:"";position:absolute;z-index:3;top:4px;left:5px;right:5px;height:4px;border-radius:3px;background:linear-gradient(90deg,#A78BFA 0 25%,#F6C453 25% 50%,#2DD4BF 50% 75%,#64748B 75% 100%);box-shadow:0 0 0 1px rgba(15,23,42,.75),0 2px 6px rgba(0,0,0,.35);pointer-events:none}
.tedna-source-minimap::after{content:"样　文　函　注";position:absolute;z-index:3;top:10px;left:3px;right:3px;padding:1px 2px;border-radius:4px;background:rgba(23,23,23,.76);color:#E5E7EB;font:8px/1.4 ${SOURCE_EDITOR_FONT};text-align:center;white-space:nowrap;pointer-events:none}
.tedna-source-status{min-height:27px;display:flex;align-items:center;gap:12px;padding:4px 10px;border-top:1px solid #333;background:#181818;color:#a1a1aa;font-size:10px;line-height:1.4;flex-wrap:wrap}.tedna-source-shortcuts{margin-left:auto}
.tedna-source-search{position:absolute;z-index:6;top:8px;width:min(680px,calc(100% - 180px));min-width:390px;padding:8px;border:1px solid #4b5563;border-radius:9px;background:#252526;box-shadow:0 8px 24px rgba(0,0,0,.34)}.tedna-source-search-reopen{position:absolute;z-index:6;top:8px;padding:6px 12px;border:1px solid #4b5563;border-radius:7px;background:#252526;color:#e5e7eb;font-size:12px;cursor:pointer;box-shadow:0 5px 16px rgba(0,0,0,.25)}
.tedna-source-search-row{display:flex;align-items:center;gap:6px}.tedna-source-search-row input{flex:1;min-width:120px;padding:6px 9px;border:1px solid #4b5563;border-radius:5px;outline:0;background:#3c3c3c;color:#f3f4f6;font:12px ${SOURCE_EDITOR_FONT}}.tedna-source-search-row input:focus{border-color:#0e639c;box-shadow:0 0 0 1px #0e639c}
.tedna-source-search-row button{width:27px;height:27px;padding:0;border:1px solid transparent;border-radius:5px;background:transparent;color:#d1d5db;font-size:12px;cursor:pointer}.tedna-source-search-row button:hover:not(:disabled){background:#3c3c3c}.tedna-source-search-row button:disabled{opacity:.35;cursor:default}.tedna-source-search-row button.is-active{background:#0e639c;color:#fff}
.tedna-source-match-count{min-width:42px;color:#d1d5db;font-size:11px;text-align:center}.tedna-source-match-count.is-empty{color:#fca5a5}.tedna-source-replace-row{margin-top:6px}.tedna-source-search-row button.tedna-source-action{width:auto;padding:0 10px;border-color:#4b5563;background:#333;color:#e5e7eb;white-space:nowrap}
@media(max-width:720px){.tedna-source-search{left:8px;right:8px!important;width:auto;min-width:0}.tedna-source-search-reopen{right:8px!important}.tedna-source-shortcuts{width:100%;margin-left:0}.tedna-source-minimap{display:none}.tedna-source-grid{grid-template-columns:50px minmax(0,1fr)!important}}
`
