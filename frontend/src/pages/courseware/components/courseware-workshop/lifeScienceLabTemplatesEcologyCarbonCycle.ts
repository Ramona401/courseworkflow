/**
 * lifeScienceLabTemplatesEcologyCarbonCycle.ts
 *
 * 平面生命科学实验室：生态系统中的碳循环。
 *
 * 教学目标：
 * 1. 识别大气、生产者、消费者、土壤与分解者、
 *    海洋和长期地质储库等主要碳库；
 * 2. 理解光合作用把大气中的二氧化碳转化为有机物；
 * 3. 理解生产者、消费者和分解者的呼吸作用
 *    会把部分有机碳重新释放为二氧化碳；
 * 4. 理解遗体、排遗物和枯落物可进入土壤有机碳库，
 *    并在分解过程中释放二氧化碳；
 * 5. 理解化石燃料燃烧可使长期地质碳较快进入大气；
 * 6. 理解海洋与大气之间存在双向碳交换；
 * 7. 比较植被恢复、森林破坏、升温和化石燃料大量燃烧
 *    对生态系统碳收支的影响；
 * 8. 区分物质循环和能量流动：
 *    碳元素可在生态系统中循环，能量则通常单向流动并逐级散失。
 *
 * 教学边界：
 * 1. 所有碳库大小、碳通量和净变化均为相对教学指标；
 * 2. 本模型不使用真实的GtC、ppm或年度碳排放数据；
 * 3. 光合作用、呼吸作用、分解作用和燃烧作用
 *    均由多个环境及生物因素共同决定，本模型使用综合参数简化表达；
 * 4. 海洋碳交换实际受温度、盐度、风速、环流、
 *    生物泵和碳酸盐平衡等因素影响，本模型只显示净吸收与释放；
 * 5. “大气碳增加”表示当前设置下进入大气的相对通量
 *    大于离开大气的相对通量，不代表真实浓度预测；
 * 6. 森林和土壤既可能是碳汇，也可能在干扰、火灾、
 *    砍伐或分解增强时成为碳源；
 * 7. 燃烧包括自然火灾和人为燃烧，本模型的高燃烧情境
 *    主要用于演示化石燃料碳快速进入大气；
 * 8. 长期地质碳库形成需要非常长的时间尺度，
 *    不能把短期生态过程简单视为化石燃料的快速再生；
 * 9. 本模型不用于气候预测、碳核算或政策评估。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式、字体、图片或CDN；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用统一.bl-*公共布局协议；
 * 5. 支持同一课件页放置多个独立实例；
 * 6. 不使用document.querySelector或document.querySelectorAll；
 * 7. 本文件只导出独立模板数组，聚合接入由第27批C1完成。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

/**
 * 安全读取数值参数。
 */
function num(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: number,
): number {
  const value = params[key]

  return typeof value === 'number' && Number.isFinite(value)
    ? value
    : fallback
}

/**
 * 安全读取布尔参数。
 */
function bool(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]

  return typeof value === 'boolean'
    ? value
    : fallback
}

/**
 * 把数值转换为适合写入HTML属性的短字符串。
 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/**
 * 构建完全限定在当前rootId内的样式。
 *
 * 独立预览时保留左侧控制区；
 * 嵌入课件后由lifeScienceLabUtils.ts中的公共覆盖层
 * 转换为“上方实验主体 + 底部课堂控制条”。
 */
function carbonCycleStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #A7F3D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#D1FAE5,#E0F2FE);border-bottom:1px solid #A7F3D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FFFC;border-right:1px solid #A7F3D0}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#059669;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981}'
    + '#' + rootId + ' .cc-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .cc-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .cc-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .cc-button{min-height:30px;padding:3px;border:1px solid #6EE7B7;border-radius:8px;background:#fff;color:#065F46;font-size:9.5px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .cc-button.active{border-color:#10B981;background:#D1FAE5;box-shadow:0 3px 9px rgba(16,185,129,.14)}'
    + '#' + rootId + ' .cc-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .cc-toggle.off{background:#64748B}'
    + '#' + rootId + ' .cc-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .cc-card{padding:6px 3px;border:1px solid #A7F3D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .cc-card b{display:block;min-height:18px;font-size:13px;color:#047857}'
    + '#' + rootId + ' .cc-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#D1FAE5;color:#064E3B;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .cc-carbon{animation:' + rootId + '-carbon var(--cc-carbon-speed,1.6s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .cc-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--cc-flow-speed,1.3s) linear infinite}'
    + '#' + rootId + ' .cc-cloud{animation:' + rootId + '-cloud 2.3s ease-in-out infinite alternate}'
    + '#' + rootId + ' .cc-pulse{animation:' + rootId + '-pulse 1.2s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-carbon{from{opacity:.42}to{opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-cloud{from{transform:translateX(-4px);opacity:.72}to{transform:translateX(5px);opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.42}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_CARBON_CYCLE:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-ecosystem-carbon-cycle',
    group: '🌎 生态系统',
    name: '生态系统中的碳循环',
    emoji: '♻️',
    desc: '调节光合作用、呼吸作用、分解作用、燃烧排放和海洋交换，观察主要碳库、碳通量及大气碳收支',
    params: [
      {
        key: 'photosynthesisActivity',
        label: '光合作用活跃度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'respirationActivity',
        label: '生物呼吸活跃度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 48,
      },
      {
        key: 'decompositionActivity',
        label: '分解作用活跃度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 45,
      },
      {
        key: 'combustionEmission',
        label: '燃烧排放水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 10,
      },
      {
        key: 'oceanExchange',
        label: '海洋碳交换强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
      {
        key: 'observationTime',
        label: '作用时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 65,
      },
      {
        key: 'showLabels',
        label: '显示过程标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const photosynthesisActivity = num(
        params,
        'photosynthesisActivity',
        78,
      )
      const respirationActivity = num(
        params,
        'respirationActivity',
        48,
      )
      const decompositionActivity = num(
        params,
        'decompositionActivity',
        45,
      )
      const combustionEmission = num(
        params,
        'combustionEmission',
        10,
      )
      const oceanExchange = num(
        params,
        'oceanExchange',
        58,
      )
      const observationTime = num(
        params,
        'observationTime',
        65,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${carbonCycleStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">♻️ 生态系统中的碳循环</div>
    <div class="bl-note">碳元素循环利用，能量单向流动并逐级散失</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>光合作用活跃度</span>
          <span class="bl-value" data-photo-value></span>
        </div>
        <input data-photo type="range" min="0" max="100" step="1" value="${n(photosynthesisActivity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>生物呼吸活跃度</span>
          <span class="bl-value" data-respiration-value></span>
        </div>
        <input data-respiration type="range" min="0" max="100" step="1" value="${n(respirationActivity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>分解作用活跃度</span>
          <span class="bl-value" data-decomposition-value></span>
        </div>
        <input data-decomposition type="range" min="0" max="100" step="1" value="${n(decompositionActivity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>燃烧排放水平</span>
          <span class="bl-value" data-combustion-value></span>
        </div>
        <input data-combustion type="range" min="0" max="100" step="1" value="${n(combustionEmission)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>海洋碳交换强度</span>
          <span class="bl-value" data-ocean-value></span>
        </div>
        <input data-ocean type="range" min="0" max="100" step="1" value="${n(oceanExchange)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>作用时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="cc-subtitle">重点观察</div>

      <div class="cc-buttons">
        <button type="button" class="cc-button active" data-mode="system">完整循环</button>
        <button type="button" class="cc-button" data-mode="biological">生物过程</button>
        <button type="button" class="cc-button" data-mode="human">人为影响</button>
      </div>

      <div class="cc-subtitle">快速比较情境</div>

      <div class="cc-scenarios">
        <button type="button" class="cc-button active" data-scenario="balanced">相对平衡</button>
        <button type="button" class="cc-button" data-scenario="restoration">植被恢复</button>
        <button type="button" class="cc-button" data-scenario="deforestation">森林破坏</button>
        <button type="button" class="cc-button" data-scenario="warming">升温</button>
        <button type="button" class="cc-button" data-scenario="fossil">高排放</button>
      </div>

      <button type="button" class="cc-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '过程标注：显示' : '过程标注：隐藏'}
      </button>

      <div class="cc-status">
        <div class="cc-card">
          <b data-atmosphere-state></b>
          <span>大气碳趋势</span>
        </div>

        <div class="cc-card">
          <b data-net-flux></b>
          <span>净大气通量</span>
        </div>

        <div class="cc-card">
          <b data-main-process></b>
          <span>主要影响过程</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="生态系统碳循环互动示意图"
      >
        <defs>
          <linearGradient id="${rootId}-sky" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#F8FAFC"/>
          </linearGradient>

          <linearGradient id="${rootId}-land" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#DCFCE7"/>
            <stop offset="100%" stop-color="#D6B47C"/>
          </linearGradient>

          <linearGradient id="${rootId}-ocean" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#BAE6FD"/>
            <stop offset="100%" stop-color="#0284C7"/>
          </linearGradient>

          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F97316"/>
          </marker>

          <marker id="${rootId}-arrow-purple" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <marker id="${rootId}-arrow-brown" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#92400E"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#064E3B" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>
        <rect x="0" y="0" width="760" height="220" fill="url(#${rootId}-sky)" opacity=".72"/>
        <rect x="0" y="220" width="515" height="210" fill="url(#${rootId}-land)" opacity=".72"/>
        <path d="M515 198 C581 177 650 188 760 167 V430 H515Z" fill="url(#${rootId}-ocean)" opacity=".82"/>

        <text x="22" y="34" data-title font-size="25" font-weight="900" fill="#065F46"></text>
        <text x="22" y="62" data-summary font-size="13" font-weight="800" fill="#475569"></text>

        <!-- 大气碳库 -->
        <g class="cc-cloud" filter="url(#${rootId}-shadow)">
          <path
            d="M246 80
               C251 51 279 42 302 56
               C316 31 359 32 372 60
               C401 52 427 73 421 100
               H258
               C238 100 229 87 246 80Z"
            fill="#FFFFFF"
            stroke="#64748B"
            stroke-width="4"
          />

          <text x="329" y="75" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">大气碳库</text>
          <text x="329" y="94" text-anchor="middle" font-size="12" font-weight="900" fill="#475569">CO₂</text>
        </g>

        <!-- 生产者 -->
        <g filter="url(#${rootId}-shadow)">
          <path d="M155 281 V174" stroke="#15803D" stroke-width="17" stroke-linecap="round"/>
          <path d="M155 200 C113 168 74 176 48 207 C87 236 126 230 155 200Z" fill="#4ADE80" stroke="#15803D" stroke-width="4"/>
          <path d="M155 188 C196 159 239 167 263 199 C225 226 187 221 155 188Z" fill="#22C55E" stroke="#15803D" stroke-width="4"/>
          <path d="M155 279 C128 300 109 319 89 344 M155 279 C181 301 205 319 227 344" fill="none" stroke="#92400E" stroke-width="8" stroke-linecap="round"/>
          <rect x="72" y="286" width="168" height="45" rx="18" fill="#FFFFFF" stroke="#22C55E" stroke-width="3"/>
          <text x="156" y="305" text-anchor="middle" font-size="13" font-weight="900" fill="#166534">生产者碳库</text>
          <rect x="91" y="314" width="130" height="10" rx="5" fill="#E2E8F0"/>
          <rect data-plant-bar x="91" y="314" width="0" height="10" rx="5" fill="#22C55E"/>
        </g>

        <!-- 消费者 -->
        <g filter="url(#${rootId}-shadow)">
          <ellipse cx="333" cy="222" rx="55" ry="34" fill="#FEF3C7" stroke="#D97706" stroke-width="4"/>
          <circle cx="379" cy="211" r="23" fill="#FDE68A" stroke="#D97706" stroke-width="4"/>
          <ellipse cx="388" cy="193" rx="8" ry="17" fill="#FDE68A" stroke="#D97706" stroke-width="3" transform="rotate(21 388 193)"/>
          <circle cx="386" cy="207" r="3" fill="#111827"/>
          <path d="M299 246 L287 263 M330 253 L323 270 M355 250 L365 267" stroke="#92400E" stroke-width="5" stroke-linecap="round"/>
          <rect x="267" y="276" width="153" height="45" rx="18" fill="#FFFFFF" stroke="#F59E0B" stroke-width="3"/>
          <text x="343" y="295" text-anchor="middle" font-size="13" font-weight="900" fill="#92400E">消费者碳库</text>
          <rect x="281" y="304" width="125" height="10" rx="5" fill="#E2E8F0"/>
          <rect data-consumer-bar x="281" y="304" width="0" height="10" rx="5" fill="#F59E0B"/>
        </g>

        <!-- 土壤有机碳与分解者 -->
        <g filter="url(#${rootId}-shadow)">
          <rect x="87" y="352" width="344" height="61" rx="24" fill="#D6B47C" stroke="#92400E" stroke-width="4"/>
          <circle cx="128" cy="376" r="13" fill="#7C3AED" opacity=".78"/>
          <circle cx="165" cy="391" r="10" fill="#8B5CF6" opacity=".78"/>
          <circle cx="204" cy="373" r="12" fill="#6D28D9" opacity=".78"/>
          <text x="267" y="372" font-size="13" font-weight="900" fill="#78350F">土壤有机碳与分解者</text>
          <rect x="257" y="383" width="147" height="11" rx="5.5" fill="#E2E8F0"/>
          <rect data-soil-bar x="257" y="383" width="0" height="11" rx="5.5" fill="#92400E"/>
          <text x="267" y="408" font-size="10.5" font-weight="800" fill="#78350F">遗体、排遗物、枯落物进入土壤</text>
        </g>

        <!-- 海洋碳库 -->
        <g filter="url(#${rootId}-shadow)">
          <ellipse cx="637" cy="256" rx="82" ry="40" fill="#E0F2FE" stroke="#0369A1" stroke-width="4"/>
          <path d="M568 250 Q590 231 612 250 T656 250 T700 250" fill="none" stroke="#38BDF8" stroke-width="5"/>
          <text x="637" y="276" text-anchor="middle" font-size="13" font-weight="900" fill="#075985">海洋碳库</text>
          <rect x="579" y="288" width="116" height="10" rx="5" fill="#E2E8F0"/>
          <rect data-ocean-bar x="579" y="288" width="0" height="10" rx="5" fill="#0284C7"/>
        </g>

        <!-- 长期地质碳库 -->
        <g filter="url(#${rootId}-shadow)">
          <path d="M520 349 H720 L693 412 H547Z" fill="#334155" stroke="#0F172A" stroke-width="4"/>
          <path d="M551 373 Q584 348 617 374 T683 373" fill="none" stroke="#64748B" stroke-width="8"/>
          <text x="620" y="397" text-anchor="middle" font-size="13" font-weight="900" fill="#FFFFFF">长期地质碳库</text>
          <rect x="567" y="404" width="106" height="9" rx="4.5" fill="#64748B"/>
          <rect data-fossil-bar x="567" y="404" width="0" height="9" rx="4.5" fill="#F59E0B"/>
        </g>

        <!-- 动态碳通量 -->
        <g data-flow-layer></g>
        <g data-particle-layer></g>

        <!-- 过程标注 -->
        <g data-label-layer>
          <text x="95" y="134" font-size="11.5" font-weight="900" fill="#166534">光合作用固定大气CO₂</text>
          <text x="340" y="133" font-size="11.5" font-weight="900" fill="#C2410C">呼吸作用释放CO₂</text>
          <text x="65" y="346" font-size="11.5" font-weight="900" fill="#78350F">枯落物与遗体</text>
          <text x="356" y="350" font-size="11.5" font-weight="900" fill="#6D28D9">分解释放CO₂</text>
          <text x="529" y="127" font-size="11.5" font-weight="900" fill="#0369A1">海气双向交换</text>
          <text x="550" y="337" font-size="11.5" font-weight="900" fill="#B91C1C">燃烧使长期碳进入大气</text>
        </g>

        <!-- 右上角碳收支面板 -->
        <g transform="translate(530 40)">
          <rect width="205" height="108" rx="18" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>

          <text x="102" y="25" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">当前大气碳收支</text>

          <text x="14" y="50" font-size="10.5" font-weight="800" fill="#64748B">进入大气</text>
          <rect x="76" y="41" width="112" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-input-bar x="76" y="41" width="0" height="12" rx="6" fill="#F97316"/>

          <text x="14" y="75" font-size="10.5" font-weight="800" fill="#64748B">离开大气</text>
          <rect x="76" y="66" width="112" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-output-bar x="76" y="66" width="0" height="12" rx="6" fill="#10B981"/>

          <text data-budget-label x="102" y="98" text-anchor="middle" font-size="11.5" font-weight="900" fill="#065F46"></text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var photo=root.querySelector('[data-photo]');
    var respiration=root.querySelector('[data-respiration]');
    var decomposition=root.querySelector('[data-decomposition]');
    var combustion=root.querySelector('[data-combustion]');
    var ocean=root.querySelector('[data-ocean]');
    var time=root.querySelector('[data-time]');

    var photoValue=root.querySelector('[data-photo-value]');
    var respirationValue=root.querySelector('[data-respiration-value]');
    var decompositionValue=root.querySelector('[data-decomposition-value]');
    var combustionValue=root.querySelector('[data-combustion-value]');
    var oceanValue=root.querySelector('[data-ocean-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var atmosphereState=root.querySelector('[data-atmosphere-state]');
    var netFluxText=root.querySelector('[data-net-flux]');
    var mainProcessText=root.querySelector('[data-main-process]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var flowLayer=root.querySelector('[data-flow-layer]');
    var particleLayer=root.querySelector('[data-particle-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');

    var plantBar=root.querySelector('[data-plant-bar]');
    var consumerBar=root.querySelector('[data-consumer-bar]');
    var soilBar=root.querySelector('[data-soil-bar]');
    var oceanBar=root.querySelector('[data-ocean-bar]');
    var fossilBar=root.querySelector('[data-fossil-bar]');

    var inputBar=root.querySelector('[data-input-bar]');
    var outputBar=root.querySelector('[data-output-bar]');
    var budgetLabel=root.querySelector('[data-budget-label]');

    var mode='system';
    var showLabels=${showLabels ? 'true' : 'false'};

    var scenarios={
      balanced:{
        photo:78,
        respiration:48,
        decomposition:45,
        combustion:10,
        ocean:58,
        time:65
      },
      restoration:{
        photo:96,
        respiration:48,
        decomposition:42,
        combustion:7,
        ocean:62,
        time:72
      },
      deforestation:{
        photo:24,
        respiration:58,
        decomposition:74,
        combustion:34,
        ocean:54,
        time:72
      },
      warming:{
        photo:62,
        respiration:82,
        decomposition:88,
        combustion:18,
        ocean:44,
        time:72
      },
      fossil:{
        photo:70,
        respiration:52,
        decomposition:50,
        combustion:96,
        ocean:70,
        time:72
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function setScenarioActive(name){
      for(var i=0;i<scenarioButtons.length;i++){
        scenarioButtons[i].classList.toggle(
          'active',
          scenarioButtons[i].getAttribute('data-scenario')===name
        );
      }
    }

    /**
     * 生成一条带方向箭头的碳通量路径。
     */
    function flow(
      path,
      color,
      marker,
      width,
      opacity
    ){
      return '<path class="cc-flow" d="'+path
        +'" fill="none" stroke="'+color
        +'" stroke-width="'+width
        +'" marker-end="url(#${rootId}-'+marker+')'
        +'" opacity="'+opacity+'"/>';
    }

    /**
     * 沿两点之间生成动态碳颗粒。
     */
    function particles(
      x1,
      y1,
      x2,
      y2,
      count,
      color,
      symbol
    ){
      var html='';

      for(var i=0;i<count;i++){
        var ratio=(i+1)/(count+1);
        var x=x1+(x2-x1)*ratio;
        var y=y1+(y2-y1)*ratio;
        var offset=(i%2===0?-1:1)*(4+i%3);

        html+='<circle class="cc-carbon" cx="'
          +(x+offset).toFixed(1)
          +'" cy="'+(y-offset*.35).toFixed(1)
          +'" r="'+(3.5+i%2)
          +'" fill="'+color+'" opacity=".84"/>';

        if(symbol && i%3===0){
          html+='<text x="'+(x+offset+6).toFixed(1)
            +'" y="'+(y-offset*.35+3).toFixed(1)
            +'" font-size="7.5" font-weight="900"'
            +' fill="'+color+'">'+symbol+'</text>';
        }
      }

      return html;
    }

    /**
     * 根据当前相对参数计算各类碳通量。
     *
     * 这些公式只用于呈现过程间的相对关系，
     * 不代表真实生态系统碳核算模型。
     */
    function calculateFluxes(
      photoLevel,
      respirationLevel,
      decompositionLevel,
      combustionLevel,
      oceanLevel,
      timeLevel
    ){
      var timeFactor=clamp(
        .32+.68*(1-Math.exp(-timeLevel/28)),
        .32,
        1
      );

      var photoFactor=
        photoLevel/(photoLevel+24);

      var respirationFactor=
        respirationLevel/(respirationLevel+30);

      var decompositionFactor=
        decompositionLevel/(decompositionLevel+30);

      var oceanFactor=
        oceanLevel/(oceanLevel+34);

      var photosynthesis=104
        *photoFactor
        *timeFactor;

      var respirationFlux=72
        *respirationFactor
        *(.45+.55*timeFactor);

      var decompositionFlux=62
        *decompositionFactor
        *(.42+.58*timeFactor);

      var combustionFlux=78
        *combustionLevel/100
        *(.38+.62*timeFactor);

      var oceanUptake=53
        *oceanFactor
        *(.52+.48*combustionLevel/100)
        *timeFactor;

      var oceanRelease=29
        *oceanFactor
        *(.95-.34*combustionLevel/100)
        *timeFactor;

      var herbivory=34
        *photoFactor
        *(.4+.6*timeFactor);

      var detritusInput=26
        *photoFactor
        *(.5+.5*respirationFactor)
        *timeFactor;

      return {
        photosynthesis:photosynthesis,
        respiration:respirationFlux,
        decomposition:decompositionFlux,
        combustion:combustionFlux,
        oceanUptake:oceanUptake,
        oceanRelease:oceanRelease,
        herbivory:herbivory,
        detritusInput:detritusInput
      };
    }

    function update(){
      var P=Number(photo.value);
      var R=Number(respiration.value);
      var D=Number(decomposition.value);
      var C=Number(combustion.value);
      var O=Number(ocean.value);
      var T=Number(time.value);

      photoValue.textContent=P.toFixed(0)+'%';
      respirationValue.textContent=R.toFixed(0)+'%';
      decompositionValue.textContent=D.toFixed(0)+'%';
      combustionValue.textContent=C.toFixed(0)+'%';
      oceanValue.textContent=O.toFixed(0)+'%';
      timeValue.textContent=T.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      labelToggle.textContent=showLabels
        ?'过程标注：显示'
        :'过程标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      var flux=calculateFluxes(
        P,
        R,
        D,
        C,
        O,
        T
      );

      var atmosphereInputs=
        flux.respiration
        +flux.decomposition
        +flux.combustion
        +flux.oceanRelease;

      var atmosphereOutputs=
        flux.photosynthesis
        +flux.oceanUptake;

      var netAtmosphere=
        atmosphereInputs-atmosphereOutputs;

      var atmosphericStock=clamp(
        50+netAtmosphere*.58,
        8,
        96
      );

      var plantStock=clamp(
        43
        +(flux.photosynthesis
          -flux.respiration*.38
          -flux.herbivory*.58)
        *.48,
        8,
        96
      );

      var consumerStock=clamp(
        25
        +(flux.herbivory
          -flux.respiration*.22)
        *.34,
        8,
        82
      );

      var soilStock=clamp(
        44
        +(flux.detritusInput
          -flux.decomposition*.44)
        *.48,
        8,
        96
      );

      var oceanStock=clamp(
        48
        +(flux.oceanUptake
          -flux.oceanRelease)
        *.72,
        8,
        96
      );

      var fossilStock=clamp(
        92-flux.combustion*.72,
        16,
        94
      );

      var atmosphereTrend=
        netAtmosphere>8
          ?'明显增加'
          :netAtmosphere>2
            ?'缓慢增加'
            :netAtmosphere<-8
              ?'明显减少'
              :netAtmosphere<-2
                ?'缓慢减少'
                :'近似平衡';

      atmosphereState.textContent=
        atmosphereTrend;

      atmosphereState.style.color=
        netAtmosphere>2
          ?'#DC2626'
          :netAtmosphere<-2
            ?'#059669'
            :'#475569';

      netFluxText.textContent=
        (netAtmosphere>0?'+':'')
        +netAtmosphere.toFixed(0);

      netFluxText.style.color=
        netAtmosphere>2
          ?'#DC2626'
          :netAtmosphere<-2
            ?'#059669'
            :'#475569';

      var processValues=[
        flux.photosynthesis,
        flux.respiration,
        flux.decomposition,
        flux.combustion,
        Math.abs(
          flux.oceanUptake-flux.oceanRelease
        )
      ];

      var processNames=[
        '光合作用',
        '生物呼吸',
        '分解作用',
        '燃烧排放',
        '海洋交换'
      ];

      var mainIndex=0;

      for(var j=1;j<processValues.length;j++){
        if(processValues[j]>processValues[mainIndex]){
          mainIndex=j;
        }
      }

      mainProcessText.textContent=
        processNames[mainIndex];

      plantBar.setAttribute(
        'width',
        String(130*plantStock/100)
      );

      consumerBar.setAttribute(
        'width',
        String(125*consumerStock/100)
      );

      soilBar.setAttribute(
        'width',
        String(147*soilStock/100)
      );

      oceanBar.setAttribute(
        'width',
        String(116*oceanStock/100)
      );

      fossilBar.setAttribute(
        'width',
        String(106*fossilStock/100)
      );

      var maxBudget=Math.max(
        1,
        atmosphereInputs,
        atmosphereOutputs
      );

      inputBar.setAttribute(
        'width',
        String(112*atmosphereInputs/maxBudget)
      );

      outputBar.setAttribute(
        'width',
        String(112*atmosphereOutputs/maxBudget)
      );

      budgetLabel.textContent=
        atmosphereTrend
        +'｜净通量 '
        +(netAtmosphere>0?'+':'')
        +netAtmosphere.toFixed(0);

      budgetLabel.setAttribute(
        'fill',
        netAtmosphere>2
          ?'#B91C1C'
          :netAtmosphere<-2
            ?'#047857'
            :'#475569'
      );

      root.style.setProperty(
        '--cc-carbon-speed',
        clamp(
          2.5-Math.max(
            flux.photosynthesis,
            atmosphereInputs
          )/72,
          .5,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--cc-flow-speed',
        clamp(
          2.4-Math.max(
            flux.photosynthesis,
            atmosphereInputs
          )/75,
          .46,
          2.4
        ).toFixed(2)+'s'
      );

      var showBiological=
        mode==='system'
        ||mode==='biological';

      var showHuman=
        mode==='system'
        ||mode==='human';

      var showOcean=
        mode==='system'
        ||mode==='human';

      var flowHTML='';
      var particleHTML='';

      if(showBiological){
        flowHTML+=flow(
          'M275 106 C235 123 199 144 170 177',
          '#16A34A',
          'arrow-green',
          3.5+flux.photosynthesis/24,
          .86
        );

        flowHTML+=flow(
          'M181 175 C226 142 274 122 309 105',
          '#F97316',
          'arrow-orange',
          3+flux.respiration/28,
          .72
        );

        flowHTML+=flow(
          'M219 213 C251 213 278 216 300 220',
          '#D97706',
          'arrow-orange',
          3+flux.herbivory/18,
          .75
        );

        flowHTML+=flow(
          'M360 192 C361 151 354 125 344 106',
          '#F97316',
          'arrow-orange',
          3+flux.respiration/30,
          .72
        );

        flowHTML+=flow(
          'M178 281 C179 314 196 337 224 355',
          '#92400E',
          'arrow-brown',
          3+flux.detritusInput/18,
          .76
        );

        flowHTML+=flow(
          'M343 253 C343 309 326 335 304 355',
          '#92400E',
          'arrow-brown',
          3+flux.detritusInput/20,
          .68
        );

        flowHTML+=flow(
          'M365 353 C399 292 407 188 371 108',
          '#7C3AED',
          'arrow-purple',
          3.5+flux.decomposition/22,
          .8
        );

        particleHTML+=particles(
          276,108,171,176,
          Math.floor(2+flux.photosynthesis/16),
          '#16A34A',
          'C'
        );

        particleHTML+=particles(
          368,348,373,112,
          Math.floor(1+flux.decomposition/18),
          '#7C3AED',
          'CO₂'
        );

        particleHTML+=particles(
          219,212,300,220,
          Math.floor(1+flux.herbivory/15),
          '#D97706',
          'C'
        );
      }

      if(showOcean){
        flowHTML+=flow(
          'M422 90 C500 86 559 120 606 205',
          '#0284C7',
          'arrow-blue',
          3+flux.oceanUptake/18,
          .75
        );

        flowHTML+=flow(
          'M626 207 C589 143 523 102 424 96',
          '#0EA5E9',
          'arrow-blue',
          3+flux.oceanRelease/18,
          .62
        );

        particleHTML+=particles(
          433,95,604,207,
          Math.floor(1+flux.oceanUptake/14),
          '#0284C7',
          'C'
        );
      }

      if(showHuman){
        flowHTML+=flow(
          'M621 349 C590 276 520 176 407 105',
          '#DC2626',
          'arrow-red',
          3.5+flux.combustion/18,
          .9
        );

        particleHTML+=particles(
          621,346,410,108,
          Math.floor(1+flux.combustion/13),
          '#DC2626',
          'CO₂'
        );
      }

      flowLayer.innerHTML=flowHTML;
      particleLayer.innerHTML=particleHTML;

      var explanation='';
      var conditionNote='';

      if(mode==='biological'){
        title.textContent=
          '生物过程推动碳在生态系统中循环';

        summary.textContent=
          '光合作用固定碳，取食传递有机碳，呼吸和分解把部分碳释放回大气';

        explanation=
          '生产者通过光合作用把大气中的二氧化碳转化为有机物。有机碳可沿食物关系进入消费者，也可随枯落物、遗体和排遗物进入土壤。';

        if(P<20){
          conditionNote=
            '当前光合作用很弱，大气碳进入生物群落的通量明显降低。';
        }else if(D>78){
          conditionNote=
            '分解作用较强，土壤有机碳向大气释放的相对通量增大。';
        }else if(R>78){
          conditionNote=
            '生物呼吸作用较强，较多有机碳以二氧化碳形式返回大气。';
        }else if(netAtmosphere<-2){
          conditionNote=
            '当前生物固定碳的相对通量高于呼吸和分解释放，陆地生态系统表现出较强吸碳趋势。';
        }else{
          conditionNote=
            '当前光合作用、呼吸作用和分解作用共同维持碳在生物与非生物环境之间流动。';
        }
      }else if(mode==='human'){
        title.textContent=
          '燃烧与土地利用改变碳收支';

        summary.textContent=
          '长期地质碳快速进入大气后，陆地和海洋只能吸收其中一部分';

        explanation=
          '化石燃料燃烧把长期储存在地质碳库中的碳较快释放到大气。森林破坏还会减少光合作用固定碳的能力，并可能增强土壤有机碳损失。';

        if(C>78){
          conditionNote=
            '当前燃烧排放很高，即使海洋交换和植被吸收较强，大气碳仍可能明显增加。';
        }else if(P<28){
          conditionNote=
            '当前植被固定碳能力很弱，生态系统对大气碳的吸收明显下降。';
        }else if(O>70 && netAtmosphere>2){
          conditionNote=
            '海洋能够吸收部分新增大气碳，但不能据此认为排放已经被完全抵消。';
        }else if(netAtmosphere<-2){
          conditionNote=
            '当前燃烧较低且生物、海洋吸收较强，大气碳呈相对减少趋势。';
        }else{
          conditionNote=
            '当前燃烧排放与陆地、海洋吸收共同决定大气碳变化。';
        }
      }else{
        title.textContent=
          '生态系统主要碳库与碳通量';

        summary.textContent=
          '碳在大气、生物群落、土壤、海洋和长期地质储库之间不断转移';

        explanation=
          '生态系统中的碳不会只停留在一个碳库。光合作用、取食、呼吸、分解、燃烧以及海气交换共同构成碳循环。';

        if(netAtmosphere>8){
          conditionNote=
            '当前进入大气的相对碳通量明显高于离开大气的通量，大气碳呈明显增加趋势。';
        }else if(netAtmosphere>2){
          conditionNote=
            '当前大气碳呈缓慢增加趋势，释放过程略强于固定和吸收过程。';
        }else if(netAtmosphere<-8){
          conditionNote=
            '当前光合作用和海洋净吸收较强，大气碳呈明显减少趋势。';
        }else if(netAtmosphere<-2){
          conditionNote=
            '当前离开大气的相对通量略高于进入大气的通量。';
        }else{
          conditionNote=
            '当前进入和离开大气的相对碳通量接近，系统处于教学意义上的近似平衡。';
        }
      }

      var timeNote=T<15
        ?'作用时间较短，各碳库的累计变化尚不明显。'
        :'作用时间增加会放大当前碳收支的累计结果，但不会自动使不平衡状态恢复平衡。';

      result.innerHTML=explanation
        +'<br>'+conditionNote
        +' '+timeNote
        +' 碳元素可以循环利用，但能量不能循环返回生产者，能量流动通常是单向的。所有数值均为相对教学指标。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<scenarioButtons.length;j++){
      scenarioButtons[j].onclick=function(){
        var name=this.getAttribute('data-scenario');
        var data=scenarios[name];

        if(!data){
          return;
        }

        photo.value=String(data.photo);
        respiration.value=String(data.respiration);
        decomposition.value=String(data.decomposition);
        combustion.value=String(data.combustion);
        ocean.value=String(data.ocean);
        time.value=String(data.time);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    photo.oninput=function(){
      setScenarioActive('');
      update();
    };

    respiration.oninput=function(){
      setScenarioActive('');
      update();
    };

    decomposition.oninput=function(){
      setScenarioActive('');
      update();
    };

    combustion.oninput=function(){
      setScenarioActive('');
      update();
    };

    ocean.oninput=function(){
      setScenarioActive('');
      update();
    };

    time.oninput=function(){
      setScenarioActive('');
      update();
    };

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
