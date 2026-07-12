/**
 * musicScoreUtils.ts — 课件五线谱/乐谱编辑器工具函数模块
 *
 * 职责：
 * 1. 生成自包含的五线谱 HTML 代码片段（abcjs + SVG 渲染 + MIDI 播放）
 * 2. 拼接用于 RefinePage 的微调指令文本
 * 3. ABC 记谱法常用模板与校验
 *
 * 技术选型：
 *   - abcjs（MIT 协议，最轻量的 JS 乐谱渲染库）
 *   - 输入格式：ABC 记谱法（一种简洁的文本音乐格式，适合 K12 音乐课）
 *   - 输出格式：SVG 五线谱 + 可选 MIDI 播放按钮
 *   - 运行时动态加载，不进主包（已自托管，见下方"自托管说明"）
 *
 * 自托管说明（批次0b 换源，2026-07-08）：
 *   abcjs 已从 jsdelivr CDN 迁移为本服务器自托管，版本 6.4.4 与原 CDN 完全一致，
 *   纯换源零行为变化。文件位于 /www/wwwroot/tedna/uploads/courseware-assets/libs/abcjs/6.4.4/，
 *   走 Nginx /uploads/courseware-assets/ 映射（CORS 头 + 30 天缓存），与音源同款链路。
 *   文件名保持 abcjs-basic-min.js 不变，生成片段内 script[src*="abcjs"] 的去重逻辑不受影响。
 *
 * 声音说明（重要）：
 *   abcjs 的合成器本身不含任何音色数据，点播放时需要在运行时下载
 *   SoundFont 音符采样（一批钢琴音符 mp3）。它的默认音源地址是
 *   paulrosen.github.io（GitHub Pages），国内浏览器基本无法访问，
 *   会静默失败导致"谱子画得出来、点播放没声音"。
 *   因此钢琴音色 88 个 mp3 已自托管到本服务器（见 SOUNDFONT_URL），
 *   编辑器试听与课件内播放按钮均显式指定 soundFontUrl 指向自托管地址。
 *
 * ABC 记谱法速查：
 *   X:1          — 曲目编号
 *   T:小星星      — 曲目标题
 *   M:4/4        — 拍号
 *   L:1/4        — 默认音符时值
 *   K:C          — 调号（C大调）
 *   CDEF GABc    — 音符（大写=低八度, 小写=高八度）
 *   z            — 休止符
 *   |            — 小节线
 *   |]           — 终止线
 *
 * 路径: frontend/src/pages/courseware/components/courseware-workshop/musicScoreUtils.ts
 * 依赖: 无外部依赖，纯函数模块
 */

// ============================================================
// 类型定义
// ============================================================

/** 乐谱配置参数 */
export interface MusicScoreConfig {
  /** ABC 记谱法文本 */
  abc: string
  /** 乐谱宽度（px，默认 600） */
  width: number
  /** 是否显示播放按钮（默认 true） */
  showPlayer: boolean
  /** 自定义标题（覆盖 ABC 内的 T: 行，可选） */
  title?: string
}

/** 微调指令配置 */
export interface MusicRefineConfig {
  /** 要融入的乐谱列表 */
  scores: MusicScoreConfig[]
  /** 融入位置偏好提示（可选） */
  positionHint?: string
}

// ============================================================
// 常量
// ============================================================

/** 自托管前端库基础地址（详见文件头"自托管说明"） */
const LIBS_BASE = 'https://workflow.pkuailab.com/uploads/courseware-assets/libs'

/** abcjs 自托管地址（原 jsdelivr abcjs@6.4.4 纯换源，含渲染 + MIDI 播放；常量名保留 _CDN 后缀避免调用方改名） */
export const ABCJS_CDN = LIBS_BASE + '/abcjs/6.4.4/abcjs-basic-min.js'

/**
 * 自托管 SoundFont 音源基础地址（钢琴音色 88 个 mp3 已下载至本服务器）
 *
 * 背景：abcjs 默认音源在 paulrosen.github.io，国内浏览器不可达导致播放无声。
 * 自托管目录走 Nginx 的 /uploads/courseware-assets/ 映射（已带 CORS 头 + 30 天缓存），
 * 与课件字体方案（CWFontBaseURL）同款链路。
 *
 * 注意：
 * 1. 地址必须以 / 结尾，abcjs 会在其后自动拼接 "acoustic_grand_piano-mp3/音名.mp3"
 * 2. 用绝对 URL（含域名）是刻意的——课件预览 iframe 为 sandbox 独立源，
 *    绝对地址在预览、放映、审核工作台等所有场景下均可稳定解析
 * 3. 音源文件位于 /www/wwwroot/tedna/uploads/courseware-assets/soundfonts/abcjs/
 */
export const SOUNDFONT_URL = 'https://workflow.pkuailab.com/uploads/courseware-assets/soundfonts/abcjs/'

/** 默认配置 */
export const MUSIC_DEFAULTS: Omit<MusicScoreConfig, 'abc'> = {
  width: 600,
  showPlayer: true,
}

/** 常用调号选项 */
export const KEY_OPTIONS = [
  { key: 'C', label: 'C大调' },
  { key: 'G', label: 'G大调' },
  { key: 'D', label: 'D大调' },
  { key: 'F', label: 'F大调' },
  { key: 'Bb', label: 'Bb大调' },
  { key: 'Am', label: 'A小调' },
  { key: 'Em', label: 'E小调' },
  { key: 'Dm', label: 'D小调' },
] as const

/** 常用拍号选项 */
export const METER_OPTIONS = [
  { meter: '4/4', label: '4/4 拍' },
  { meter: '3/4', label: '3/4 拍' },
  { meter: '2/4', label: '2/4 拍' },
  { meter: '6/8', label: '6/8 拍' },
  { meter: '2/2', label: '2/2 拍' },
] as const

/**
 * 常用乐谱模板——老师可直接点选
 */
export const MUSIC_TEMPLATES: { label: string; abc: string }[] = [
  {
    label: '🌟 小星星（C大调）',
    abc: 'X:1\nT:小星星\nM:4/4\nL:1/4\nK:C\nC C G G | A A G2 | F F E E | D D C2 |\nG G F F | E E D2 | G G F F | E E D2 |\nC C G G | A A G2 | F F E E | D D C2 |]',
  },
  {
    label: '🎵 欢乐颂（D大调）',
    abc: 'X:1\nT:欢乐颂\nM:4/4\nL:1/4\nK:D\nF F G A | A G F E | D D E F | F E E2 |\nF F G A | A G F E | D D E F | E D D2 |]',
  },
  {
    label: '🎶 两只老虎（C大调）',
    abc: 'X:1\nT:两只老虎\nM:4/4\nL:1/4\nK:C\nC D E C | C D E C | E F G2 | E F G2 |\nG A G F E C | G A G F E C | C G, C2 | C G, C2 |]',
  },
  {
    label: '🎼 C大调音阶',
    abc: 'X:1\nT:C大调音阶\nM:4/4\nL:1/4\nK:C\nC D E F | G A B c | c B A G | F E D C |]',
  },
  {
    label: '🎹 简单和弦进行',
    abc: 'X:1\nT:和弦进行 I-V-vi-IV\nM:4/4\nL:1/4\nK:C\n"C"C E G c | "G"B, D G B | "Am"A, C E A | "F"F, A, C F |]',
  },
  {
    label: '🥁 基础节奏练习（2/4拍）',
    abc: 'X:1\nT:节奏练习\nM:2/4\nL:1/8\nK:C\nC C C C | C2 C2 | C C C2 | C2 z2 |\nC/C/ C C C | C C C/C/ C | C2 C C | C4 |]',
  },
  {
    label: '🎵 空白模板',
    abc: 'X:1\nT:我的乐谱\nM:4/4\nL:1/4\nK:C\nz4 | z4 | z4 | z4 |]',
  },
]

// ============================================================
// HTML 代码片段生成
// ============================================================

/**
 * 为乐谱生成唯一 DOM ID
 */
function makeElementId(): string {
  const suffix = Date.now().toString(36).slice(-6) + Math.random().toString(36).slice(-4)
  return 'music-' + suffix
}

/**
 * 为单个乐谱生成完整的自包含 HTML 代码片段
 *
 * 生成的代码包含：
 * - 外层容器 div（带样式）
 * - 乐谱渲染目标 div
 * - 可选播放按钮
 * - abcjs 库引用（script 标签，去重）
 * - 初始化脚本（IIFE 封装）
 *
 * 播放按钮的合成器初始化显式传入 soundFontUrl 指向自托管音源，
 * 保证国内网络环境下点播放能出声（否则走 abcjs 默认 github.io 音源会静默无声）。
 */
export function generateMusicEmbed(config: MusicScoreConfig): string {
  const id = makeElementId()
  const renderTargetId = id + '-render'
  const playerTargetId = id + '-player'

  // 转义 ABC 文本中的引号和反斜杠
  const escapedAbc = config.abc
    .replace(/\\/g, '\\\\')
    .replace(/'/g, "\\'")
    .replace(/\n/g, '\\n')

  const titleHtml = config.title
    ? '<div style="font-size:18px;font-weight:700;color:#1F2937;margin-bottom:8px;text-align:center;">' + escapeHtml(config.title) + '</div>'
    : ''

  const playerHtml = config.showPlayer
    ? '<div id="' + playerTargetId + '" style="margin-top:8px;text-align:center;"></div>'
    : ''

  return '<div id="' + id + '" style="text-align:center;padding:16px 20px;background:#FFFDF7;border:1px solid #E5E0D5;border-radius:12px;">\n'
    + '  ' + titleHtml + '\n'
    + '  <div id="' + renderTargetId + '" style="max-width:' + config.width + 'px;margin:0 auto;"></div>\n'
    + '  ' + playerHtml + '\n'
    + '  <script>\n'
    + '  (function(){\n'
    + '    function loadAbcjs(cb){\n'
    + '      if(typeof ABCJS!=="undefined"){cb();return;}\n'
    + '      if(!document.querySelector("script[src*=\\"abcjs\\"]")){\n'
    + '        var s=document.createElement("script");s.src="' + ABCJS_CDN + '";\n'
    + '        s.onload=cb;document.head.appendChild(s);\n'
    + '      } else { setTimeout(function(){loadAbcjs(cb);},200); }\n'
    + '    }\n'
    + '    loadAbcjs(function(){\n'
    + '      var abc=\'' + escapedAbc + '\';\n'
    + '      try{\n'
    + '        ABCJS.renderAbc("' + renderTargetId + '",abc,{responsive:"resize",staffwidth:' + config.width + '});\n'
    + (config.showPlayer
      ? '        if(ABCJS.synth&&ABCJS.synth.supportsAudio()){\n'
        + '          var playerEl=document.getElementById("' + playerTargetId + '");\n'
        + '          if(playerEl){\n'
        + '            var btn=document.createElement("button");\n'
        + '            btn.textContent="▶ 播放";\n'
        + '            btn.style.cssText="padding:8px 20px;border-radius:8px;border:1px solid #D4A574;background:#FFF8F0;color:#8B6914;font-size:13px;font-weight:600;cursor:pointer;";\n'
        + '            var playing=false,synthCtrl=null;\n'
        + '            btn.onclick=function(){\n'
        + '              if(playing){if(synthCtrl)synthCtrl.stop();btn.textContent="▶ 播放";playing=false;return;}\n'
        + '              btn.textContent="⏹ 停止";playing=true;\n'
        + '              var vis=ABCJS.renderAbc("*",abc)[0];\n'
        + '              synthCtrl=new ABCJS.synth.CreateSynth();\n'
        + '              synthCtrl.init({visualObj:vis,options:{soundFontUrl:"' + SOUNDFONT_URL + '"}}).then(function(){return synthCtrl.prime();}).then(function(){synthCtrl.start();synthCtrl.addEventListener("finished",function(){btn.textContent="▶ 播放";playing=false;});}).catch(function(){btn.textContent="▶ 播放";playing=false;});\n'
        + '            };\n'
        + '            playerEl.appendChild(btn);\n'
        + '          }\n'
        + '        }\n'
      : '')
    + '      }catch(e){document.getElementById("' + renderTargetId + '").textContent="乐谱渲染失败: "+e.message;}\n'
    + '    });\n'
    + '  })();\n'
    + '  </' + 'script>\n'
    + '</div>'
}

/** HTML 转义辅助 */
function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

/**
 * 为多个乐谱批量生成代码片段
 */
export function generateMultiMusicEmbed(scores: MusicScoreConfig[]): string {
  if (scores.length === 0) return ''
  if (scores.length === 1) return generateMusicEmbed(scores[0])
  const embeds = scores.map(s => generateMusicEmbed(s))
  return '<div style="display:flex;flex-direction:column;gap:24px;padding:16px 0;">\n  ' + embeds.join('\n  ') + '\n</div>'
}

// ============================================================
// 微调指令拼接
// ============================================================

/**
 * 根据乐谱配置生成 RefinePage 的微调指令文本
 */
export function buildMusicRefineInstruction(config: MusicRefineConfig): string {
  const { scores, positionHint } = config
  if (scores.length === 0) return ''

  const embedCode = generateMultiMusicEmbed(scores)
  const count = scores.length

  const posDesc = positionHint
    ? '位置偏好: ' + positionHint + '。'
    : '请根据页面现有内容布局，在最合适的位置放置乐谱区域。'

  return '请在当前课件页面中融入五线谱乐谱组件。\n\n'
    + '【融入要求】\n'
    + '1. ' + posDesc + '\n'
    + '2. 共 ' + count + ' 段乐谱展示。\n'
    + '3. 保持与页面整体视觉风格协调——配色、圆角、间距与现有元素和谐统一。\n'
    + '4. 乐谱区域应有清晰的视觉边界（如卡片容器），保留已有的标题和播放按钮。\n'
    + '5. 不要删除或大幅改动页面现有的其他内容和布局结构。\n'
    + '6. 以下代码中的 <script> 标签和 JavaScript 逻辑必须完整保留，不要修改脚本内容，尤其是 soundFontUrl 音源地址和库加载地址一个字符都不能改（改了播放会没声音）。\n'
    + '7. abcjs 库引用会自动去重加载，保持 script 标签在代码片段内即可。\n\n'
    + '【以下是需要融入的完整代码（含 abcjs 库引用、SVG 渲染和播放按钮脚本）】\n\n'
    + embedCode
}
