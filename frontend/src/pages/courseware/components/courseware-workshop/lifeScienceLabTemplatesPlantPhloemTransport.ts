/**
 * lifeScienceLabTemplatesPlantPhloemTransport.ts
 *
 * 平面生命科学实验室：韧皮部中的有机物运输。
 *
 * 教学目标：
 * 1. 区分有机物运输中的“源”和“库”；
 * 2. 理解成熟叶片通常可作为有机物输出的源；
 * 3. 理解生长旺盛或储藏有机物的器官可作为库；
 * 4. 观察蔗糖装载、筛管压力差和库端卸载之间的联系；
 * 5. 理解韧皮部汁液可沿筛管由源流向库；
 * 6. 观察韧皮部连续性下降或环剥时有机物运输受阻；
 * 7. 理解整株植物中不同筛管可同时向不同方向运输，
 *    但同一段筛管中的净流动由当前源库关系决定。
 *
 * 教学边界：
 * 1. 光合产物、库需求、装载活性、压力和运输速率
 *    均为相对教学指标；
 * 2. 韧皮部运输的主要有机物通常以蔗糖等可溶性有机物形式存在，
 *    本模型统一用“蔗糖”代表可运输有机物；
 * 3. 不同植物的韧皮部装载方式不同，可能包括主动装载、
 *    被动装载或二者组合，本模型用装载活性作综合简化；
 * 4. 压力流学说强调源端装载后吸水形成较高膨压，
 *    库端卸载后压力较低，从而形成筛管中的整体流动；
 * 5. 水可以在源端由木质部进入韧皮部，并在库端部分返回木质部，
 *    本模型只演示两者之间的简化联系；
 * 6. 环剥主要破坏树皮中的韧皮部，短期内木质部水分运输仍可继续，
 *    但有机物向环剥下方运输会受阻；
 * 7. 本模板不展开筛板孔径、原生质联络丝、聚合物陷阱、
 *    韧皮部电信号和不同糖类的精细代谢过程。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式、字体、图片或CDN；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用统一.bl-*公共布局协议；
 * 5. 支持同一课件页放置多个独立实例；
 * 6. 不使用document.querySelector或document.querySelectorAll；
 * 7. 本文件只导出独立模板数组，聚合接入由第26批C1完成。
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
 * 构建完全限定在当前rootId内部的样式。
 *
 * 独立预览时保留左侧控制栏；
 * 嵌入课件后由公共布局覆盖层转换为
 * “上方实验主体 + 底部课堂控制条”。
 */
function phloemTransportStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #DDD6FE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#F5F3FF,#DCFCE7);border-bottom:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#8B5CF6}'
    + '#' + rootId + ' .pt-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .pt-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .pt-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .pt-button{min-height:30px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:9.5px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .pt-button.active{border-color:#8B5CF6;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .pt-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pt-toggle.off{background:#64748B}'
    + '#' + rootId + ' .pt-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .pt-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .pt-card b{display:block;min-height:18px;font-size:13px;color:#6D28D9}'
    + '#' + rootId + ' .pt-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ph-sugar{animation:' + rootId + '-sugar var(--ph-sugar-speed,1.5s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .ph-water{animation:' + rootId + '-water var(--ph-water-speed,1.8s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .ph-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--ph-flow-speed,1.2s) linear infinite}'
    + '#' + rootId + ' .ph-atp{animation:' + rootId + '-atp 1.1s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ph-pulse{animation:' + rootId + '-pulse 1.4s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-sugar{from{opacity:.46}to{opacity:1}}'
    + '@keyframes ' + rootId + '-water{from{opacity:.35}to{opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-atp{from{opacity:.42}to{opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.52}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_PLANT_PHLOEM_TRANSPORT:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-phloem-organic-transport',
    group: '🌿 植物生理',
    name: '韧皮部中的有机物运输',
    emoji: '🍬',
    desc: '调节源端有机物生产、库端需求、装载活性、筛管连续性和作用时间，观察韧皮部压力流与源库运输',
    params: [
      {
        key: 'sourceProduction',
        label: '源端有机物生产',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'sinkDemand',
        label: '库端有机物需求',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'loadingActivity',
        label: '装载与卸载活性',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 82,
        hint: '综合表示运输蛋白和能量条件',
      },
      {
        key: 'sieveContinuity',
        label: '筛管连续性',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 94,
        hint: '降低可模拟韧皮部损伤或环剥',
      },
      {
        key: 'observationTime',
        label: '作用时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const sourceProduction = num(
        params,
        'sourceProduction',
        78,
      )
      const sinkDemand = num(
        params,
        'sinkDemand',
        72,
      )
      const loadingActivity = num(
        params,
        'loadingActivity',
        82,
      )
      const sieveContinuity = num(
        params,
        'sieveContinuity',
        94,
      )
      const observationTime = num(
        params,
        'observationTime',
        62,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${phloemTransportStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🍬 韧皮部中的有机物运输</div>
    <div class="bl-note">源端装载形成高压，库端卸载维持低压</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>源端有机物生产</span>
          <span class="bl-value" data-source-value></span>
        </div>
        <input data-source type="range" min="0" max="100" step="1" value="${n(sourceProduction)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>库端有机物需求</span>
          <span class="bl-value" data-sink-value></span>
        </div>
        <input data-sink type="range" min="0" max="100" step="1" value="${n(sinkDemand)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>装载与卸载活性</span>
          <span class="bl-value" data-loading-value></span>
        </div>
        <input data-loading type="range" min="0" max="100" step="1" value="${n(loadingActivity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>筛管连续性</span>
          <span class="bl-value" data-continuity-value></span>
        </div>
        <input data-continuity type="range" min="0" max="100" step="1" value="${n(sieveContinuity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>作用时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="pt-subtitle">重点观察</div>

      <div class="pt-buttons">
        <button type="button" class="pt-button active" data-mode="combined">综合运输</button>
        <button type="button" class="pt-button" data-mode="loading">源库装卸</button>
        <button type="button" class="pt-button" data-mode="pressure">压力流动</button>
      </div>

      <div class="pt-subtitle">当前主要库器官</div>

      <div class="pt-buttons">
        <button type="button" class="pt-button active" data-sink-type="fruit">果实</button>
        <button type="button" class="pt-button" data-sink-type="root">储藏根</button>
        <button type="button" class="pt-button" data-sink-type="bud">生长芽</button>
      </div>

      <div class="pt-subtitle">快速比较情境</div>

      <div class="pt-scenarios">
        <button type="button" class="pt-button active" data-scenario="sunlight">强光</button>
        <button type="button" class="pt-button" data-scenario="shade">遮阴</button>
        <button type="button" class="pt-button" data-scenario="fruiting">结果</button>
        <button type="button" class="pt-button" data-scenario="girdling">环剥</button>
        <button type="button" class="pt-button" data-scenario="storage">储藏</button>
      </div>

      <button type="button" class="pt-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '结构标注：显示' : '结构标注：隐藏'}
      </button>

      <div class="pt-status">
        <div class="pt-card">
          <b data-rate></b>
          <span>相对运输速率</span>
        </div>

        <div class="pt-card">
          <b data-gradient></b>
          <span>源库压力差</span>
        </div>

        <div class="pt-card">
          <b data-state></b>
          <span>当前状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="韧皮部有机物运输互动示意图"
      >
        <defs>
          <linearGradient id="${rootId}-stem" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stop-color="#86EFAC"/>
            <stop offset="50%" stop-color="#16A34A"/>
            <stop offset="100%" stop-color="#86EFAC"/>
          </linearGradient>

          <linearGradient id="${rootId}-phloem" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#F3E8FF"/>
            <stop offset="100%" stop-color="#C4B5FD"/>
          </linearGradient>

          <linearGradient id="${rootId}-xylem" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#7DD3FC"/>
          </linearGradient>

          <linearGradient id="${rootId}-soil" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#FDE68A"/>
            <stop offset="100%" stop-color="#B45309"/>
          </linearGradient>

          <marker id="${rootId}-arrow-purple" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#4C1D95" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text x="24" y="36" data-title font-size="27" font-weight="900" fill="#5B21B6"></text>
        <text x="24" y="65" data-summary font-size="14" font-weight="800" fill="#475569"></text>

        <rect x="0" y="354" width="505" height="76" fill="url(#${rootId}-soil)" opacity=".82"/>

        <g filter="url(#${rootId}-shadow)">
          <path d="M294 359 C294 298 295 217 296 112" fill="none" stroke="url(#${rootId}-stem)" stroke-width="86" stroke-linecap="round"/>

          <path
            d="M291 112
               C221 82 152 99 101 151
               C164 190 235 177 298 132Z"
            fill="#4ADE80"
            stroke="#15803D"
            stroke-width="5"
          />

          <path
            d="M299 122
               C361 91 421 100 458 141
               C407 171 350 169 298 139Z"
            fill="#22C55E"
            stroke="#15803D"
            stroke-width="5"
          />

          <path d="M295 352 C254 371 217 390 177 420" fill="none" stroke="#92400E" stroke-width="17" stroke-linecap="round"/>
          <path d="M295 352 C332 376 367 395 401 421" fill="none" stroke="#92400E" stroke-width="17" stroke-linecap="round"/>
          <path d="M295 352 C294 381 294 405 294 429" fill="none" stroke="#92400E" stroke-width="17" stroke-linecap="round"/>
        </g>

        <!-- 木质部与韧皮部并列显示 -->
        <rect
          x="282"
          y="108"
          width="28"
          height="248"
          rx="13"
          fill="url(#${rootId}-xylem)"
          stroke="#0369A1"
          stroke-width="4"
        />

        <rect
          x="315"
          y="108"
          width="31"
          height="248"
          rx="13"
          fill="url(#${rootId}-phloem)"
          stroke="#7C3AED"
          stroke-width="4"
        />

        <g data-sieve-plates></g>
        <g data-continuity-damage></g>

        <!-- 源叶中的有机物生产与装载 -->
        <g data-source-sugar></g>
        <g data-loading-layer></g>

        <!-- 筛管压力流和蔗糖颗粒 -->
        <g data-sugar-flow></g>

        <!-- 木质部与韧皮部之间的水交换 -->
        <g data-water-exchange></g>

        <!-- 三类库器官 -->
        <g data-fruit-layer>
          <path d="M345 225 C405 223 450 225 504 236" fill="none" stroke="#7C3AED" stroke-width="13" stroke-linecap="round"/>
          <circle cx="541" cy="246" r="38" fill="#FCA5A5" stroke="#DC2626" stroke-width="5"/>
          <path d="M541 208 C534 195 540 183 552 178" fill="none" stroke="#15803D" stroke-width="6" stroke-linecap="round"/>
          <ellipse cx="558" cy="178" rx="18" ry="9" fill="#4ADE80" stroke="#15803D" stroke-width="3" transform="rotate(-25 558 178)"/>
          <text x="541" y="251" text-anchor="middle" font-size="13" font-weight="900" fill="#7F1D1D">果实</text>
        </g>

        <g data-root-layer opacity=".26">
          <ellipse cx="294" cy="393" rx="55" ry="25" fill="#F59E0B" stroke="#92400E" stroke-width="5"/>
          <text x="294" y="399" text-anchor="middle" font-size="12" font-weight="900" fill="#78350F">储藏根</text>
        </g>

        <g data-bud-layer opacity=".26">
          <path d="M338 145 C389 124 424 101 457 78" fill="none" stroke="#7C3AED" stroke-width="12" stroke-linecap="round"/>
          <path d="M457 79 C472 61 488 59 497 75 C489 94 473 99 457 79Z" fill="#A3E635" stroke="#3F6212" stroke-width="4"/>
          <text x="506" y="80" font-size="12" font-weight="900" fill="#3F6212">生长芽</text>
        </g>

        <g data-unloading-layer></g>
        <g data-accumulation-layer></g>

        <g data-label-layer>
          <path d="M203 127 L143 87" stroke="#64748B" stroke-width="2.5"/>
          <text x="62" y="83" font-size="13" font-weight="900" fill="#166534">源叶：输出有机物</text>

          <path d="M315 202 L383 202" stroke="#64748B" stroke-width="2.5"/>
          <text x="391" y="207" font-size="13" font-weight="900" fill="#6D28D9">韧皮部筛管</text>

          <path d="M282 283 L213 283" stroke="#64748B" stroke-width="2.5"/>
          <text x="116" y="288" font-size="13" font-weight="900" fill="#0369A1">木质部水流</text>

          <path data-sink-label-line d="M505 246 L614 246" stroke="#64748B" stroke-width="2.5"/>
          <text data-sink-label x="620" y="251" font-size="13" font-weight="900" fill="#475569">库器官</text>
        </g>

        <g transform="translate(546 100)">
          <rect width="190" height="220" rx="20" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>

          <text x="95" y="27" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">源—库运输状态</text>

          <text x="15" y="56" font-size="11" font-weight="800" fill="#64748B">源端压力</text>
          <rect x="15" y="66" width="160" height="14" rx="7" fill="#E2E8F0"/>
          <rect data-source-pressure-bar x="15" y="66" width="0" height="14" rx="7" fill="#A78BFA"/>

          <text x="15" y="100" font-size="11" font-weight="800" fill="#64748B">库端压力</text>
          <rect x="15" y="110" width="160" height="14" rx="7" fill="#E2E8F0"/>
          <rect data-sink-pressure-bar x="15" y="110" width="0" height="14" rx="7" fill="#C4B5FD"/>

          <text x="15" y="144" font-size="11" font-weight="800" fill="#64748B">装载卸载活性</text>
          <rect x="15" y="154" width="160" height="14" rx="7" fill="#E2E8F0"/>
          <rect data-loading-bar x="15" y="154" width="0" height="14" rx="7" fill="#F59E0B"/>

          <text x="15" y="188" font-size="11" font-weight="800" fill="#64748B">筛管连续性</text>
          <rect x="15" y="198" width="160" height="14" rx="7" fill="#E2E8F0"/>
          <rect data-continuity-bar x="15" y="198" width="0" height="14" rx="7" fill="#10B981"/>
        </g>

        <g transform="translate(522 352)">
          <circle cx="7" cy="7" r="7" fill="#8B5CF6"/>
          <text x="22" y="12" font-size="12" font-weight="800" fill="#475569">蔗糖等有机物</text>

          <circle cx="143" cy="7" r="7" fill="#38BDF8"/>
          <text x="158" y="12" font-size="12" font-weight="800" fill="#475569">水分子</text>
        </g>

        <text x="520" y="402" data-stage-note font-size="12.5" font-weight="900" fill="#5B21B6"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var source=root.querySelector('[data-source]');
    var sink=root.querySelector('[data-sink]');
    var loading=root.querySelector('[data-loading]');
    var continuity=root.querySelector('[data-continuity]');
    var time=root.querySelector('[data-time]');

    var sourceValue=root.querySelector('[data-source-value]');
    var sinkValue=root.querySelector('[data-sink-value]');
    var loadingValue=root.querySelector('[data-loading-value]');
    var continuityValue=root.querySelector('[data-continuity-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var sinkButtons=root.querySelectorAll('[data-sink-type]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var rateText=root.querySelector('[data-rate]');
    var gradientText=root.querySelector('[data-gradient]');
    var stateText=root.querySelector('[data-state]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var sievePlates=root.querySelector('[data-sieve-plates]');
    var continuityDamage=root.querySelector('[data-continuity-damage]');
    var sourceSugar=root.querySelector('[data-source-sugar]');
    var loadingLayer=root.querySelector('[data-loading-layer]');
    var sugarFlow=root.querySelector('[data-sugar-flow]');
    var waterExchange=root.querySelector('[data-water-exchange]');
    var unloadingLayer=root.querySelector('[data-unloading-layer]');
    var accumulationLayer=root.querySelector('[data-accumulation-layer]');

    var fruitLayer=root.querySelector('[data-fruit-layer]');
    var rootLayer=root.querySelector('[data-root-layer]');
    var budLayer=root.querySelector('[data-bud-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');
    var sinkLabelLine=root.querySelector('[data-sink-label-line]');
    var sinkLabel=root.querySelector('[data-sink-label]');

    var sourcePressureBar=root.querySelector(
      '[data-source-pressure-bar]'
    );
    var sinkPressureBar=root.querySelector(
      '[data-sink-pressure-bar]'
    );
    var loadingBar=root.querySelector('[data-loading-bar]');
    var continuityBar=root.querySelector(
      '[data-continuity-bar]'
    );

    var mode='combined';
    var sinkType='fruit';
    var showLabels=${showLabels ? 'true' : 'false'};

    var scenarios={
      sunlight:{
        source:88,
        sink:72,
        loading:86,
        continuity:96,
        time:65,
        sinkType:'fruit'
      },
      shade:{
        source:18,
        sink:70,
        loading:74,
        continuity:95,
        time:65,
        sinkType:'fruit'
      },
      fruiting:{
        source:78,
        sink:97,
        loading:90,
        continuity:95,
        time:65,
        sinkType:'fruit'
      },
      girdling:{
        source:82,
        sink:74,
        loading:84,
        continuity:16,
        time:65,
        sinkType:'root'
      },
      storage:{
        source:72,
        sink:94,
        loading:84,
        continuity:96,
        time:65,
        sinkType:'root'
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

    function updateSinkButtons(){
      for(var i=0;i<sinkButtons.length;i++){
        sinkButtons[i].classList.toggle(
          'active',
          sinkButtons[i].getAttribute('data-sink-type')===sinkType
        );
      }
    }

    /**
     * 绘制筛管中的筛板。
     */
    function buildSievePlates(){
      var html='';

      for(var i=0;i<6;i++){
        var y=137+i*35;

        html+='<line x1="317" y1="'+y
          +'" x2="344" y2="'+y
          +'" stroke="#6D28D9" stroke-width="3"/>';

        for(var j=0;j<4;j++){
          html+='<circle cx="'+(321+j*6.3)
            +'" cy="'+y+'" r="1.8" fill="#FFFFFF"/>';
        }
      }

      return html;
    }

    /**
     * 绘制源叶产生的蔗糖颗粒。
     */
    function buildSourceSugar(level){
      var html='';
      var count=Math.floor(2+level/8);

      for(var i=0;i<count;i++){
        var x=121+(i%7)*22;
        var y=120+Math.floor(i/7)*26+(i%2)*7;

        html+='<circle class="ph-sugar" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%3)
          +'" fill="#8B5CF6" stroke="#5B21B6"'
          +' stroke-width="1.5"/>';

        if(i%4===0){
          html+='<text x="'+(x+7)+'" y="'+(y+4)
            +'" font-size="8" font-weight="900"'
            +' fill="#5B21B6">糖</text>';
        }
      }

      return html;
    }

    /**
     * 绘制源端从叶肉细胞向筛管的装载过程。
     */
    function buildLoadingGraphic(
      production,
      loadingLevel,
      visible
    ){
      if(!visible){
        return '';
      }

      var activity=clamp(
        production/100*loadingLevel/100,
        0,
        1
      );

      var width=3.5+activity*5;
      var html=''
        +'<path class="ph-flow" d="M218 145'
        +' C255 143 286 151 322 165"'
        +' fill="none" stroke="#7C3AED"'
        +' stroke-width="'+width
        +'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<text x="219" y="133" font-size="11"'
        +' font-weight="900" fill="#6D28D9">'
        +'蔗糖装载进入筛管'
        +'</text>';

      var atpCount=Math.max(
        1,
        Math.floor(loadingLevel/19)
      );

      for(var i=0;i<atpCount;i++){
        var x=224+(i%5)*24;
        var y=182+Math.floor(i/5)*23;

        html+='<g class="ph-atp" transform="translate('
          +x+' '+y+')">'
          +'<polygon points="0,-8 7,-2 4,7 -4,7 -7,-2"'
          +' fill="#FACC15" stroke="#CA8A04" stroke-width="1.8"/>'
          +'<text x="0" y="3" text-anchor="middle"'
          +' font-size="6.5" font-weight="900" fill="#854D0E">ATP</text>'
          +'</g>';
      }

      return html;
    }

    /**
     * 返回从源端到所选库器官的路径信息。
     */
    function resolveSinkPath(type){
      if(type==='root'){
        return {
          path:'M330 165 C330 220 330 292 302 381',
          endX:302,
          endY:381,
          label:'储藏根'
        };
      }

      if(type==='bud'){
        return {
          path:'M330 165 C355 136 402 108 465 79',
          endX:465,
          endY:79,
          label:'生长芽'
        };
      }

      return {
        path:'M330 165 C343 192 405 221 526 243',
        endX:526,
        endY:243,
        label:'果实'
      };
    }

    /**
     * 绘制筛管内的有机物压力流。
     */
    function buildSugarFlow(rate,type,visible){
      if(!visible){
        return '';
      }

      var path=resolveSinkPath(type);
      var thickness=3.5+rate/22;
      var html='';

      if(rate>2){
        html+='<path class="ph-flow" d="'+path.path+'"'
          +' fill="none" stroke="#7C3AED"'
          +' stroke-width="'+thickness
          +'" marker-end="url(#${rootId}-arrow-purple)"'
          +' opacity=".88"/>';
      }

      var count=Math.floor(2+rate/9);

      for(var i=0;i<count;i++){
        var progress=(i+1)/(count+1);
        var x;
        var y;

        if(type==='root'){
          x=330-28*Math.pow(progress,2);
          y=165+216*progress;
        }else if(type==='bud'){
          x=330+135*progress;
          y=165-86*progress;
        }else{
          x=330+196*progress;
          y=165+78*progress;
        }

        html+='<circle class="ph-sugar" cx="'+x.toFixed(1)
          +'" cy="'+y.toFixed(1)+'" r="'+(4+i%3)
          +'" fill="#8B5CF6" stroke="#5B21B6"'
          +' stroke-width="1.5"/>';
      }

      return html;
    }

    /**
     * 绘制源端木质部水进入韧皮部、
     * 库端韧皮部水部分返回木质部的过程。
     */
    function buildWaterExchange(
      sourcePressure,
      sinkDemandLevel,
      type,
      visible
    ){
      if(!visible){
        return '';
      }

      var path=resolveSinkPath(type);
      var sourceWidth=3+sourcePressure/30;
      var sinkWidth=3+sinkDemandLevel/35;

      return ''
        +'<path class="ph-flow" d="M292 153 C305 151 313 154 322 162"'
        +' fill="none" stroke="#0284C7"'
        +' stroke-width="'+sourceWidth
        +'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="208" y="108" font-size="10.5"'
        +' font-weight="900" fill="#0369A1">'
        +'源端：水进入筛管'
        +'</text>'
        +'<path class="ph-flow" d="M'+path.endX+' '+path.endY
        +' C'+(path.endX-26)+' '+(path.endY+17)
        +' 360 310 302 300"'
        +' fill="none" stroke="#0284C7"'
        +' stroke-width="'+sinkWidth
        +'" marker-end="url(#${rootId}-arrow-blue)"'
        +' opacity=".7"/>'
        +'<text x="382" y="326" font-size="10.5"'
        +' font-weight="900" fill="#0369A1">'
        +'库端：部分水返回木质部'
        +'</text>';
    }

    /**
     * 绘制库端卸载和利用、储藏过程。
     */
    function buildUnloadingGraphic(
      rate,
      type,
      visible
    ){
      if(!visible){
        return '';
      }

      var path=resolveSinkPath(type);
      var radius=18+rate*.17;
      var color=type==='fruit'
        ?'#DC2626'
        :type==='root'
          ?'#B45309'
          :'#65A30D';

      return ''
        +'<circle class="ph-pulse" cx="'+path.endX
        +'" cy="'+path.endY+'" r="'+radius.toFixed(1)
        +'" fill="none" stroke="'+color
        +'" stroke-width="4" stroke-dasharray="7 6"/>'
        +'<text x="'+path.endX+'" y="'+(path.endY-48)
        +'" text-anchor="middle" font-size="11"'
        +' font-weight="900" fill="'+color+'">'
        +'卸载、利用或储藏'
        +'</text>';
    }

    /**
     * 连续性下降时绘制筛管损伤和环剥阻断区。
     */
    function buildContinuityDamage(level){
      var damage=clamp(
        (100-level)/100,
        0,
        1
      );

      if(damage<.12){
        return '';
      }

      var count=Math.max(
        1,
        Math.floor(damage*5)
      );
      var html='';

      for(var i=0;i<count;i++){
        var y=198+i*29;

        html+='<rect x="315" y="'+y
          +'" width="31" height="'+(7+damage*9)
          +'" fill="#FFFFFF" stroke="#DC2626"'
          +' stroke-width="2" stroke-dasharray="4 3"/>';
      }

      if(level<30){
        html+='<rect x="265" y="274" width="100" height="23"'
          +' fill="#FEE2E2" stroke="#DC2626"'
          +' stroke-width="4" stroke-dasharray="7 5"/>';

        html+='<text x="371" y="290" font-size="11"'
          +' font-weight="900" fill="#DC2626">'
          +'环剥或韧皮部严重受损'
          +'</text>';
      }

      return html;
    }

    /**
     * 韧皮部阻断时在损伤上方显示有机物积累。
     */
    function buildAccumulation(
      production,
      continuityLevel
    ){
      if(continuityLevel>55){
        return '';
      }

      var severity=clamp(
        (55-continuityLevel)/55,
        0,
        1
      );

      var count=Math.floor(
        2+production/16+severity*7
      );
      var html='';

      for(var i=0;i<count;i++){
        var x=351+(i%5)*17;
        var y=248-Math.floor(i/5)*18-(i%2)*4;

        html+='<circle class="ph-sugar" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%2)
          +'" fill="#A855F7" stroke="#6B21A8"'
          +' stroke-width="1.5"/>';
      }

      html+='<text x="371" y="222" font-size="11"'
        +' font-weight="900" fill="#7E22CE">'
        +'阻断上方有机物积累'
        +'</text>';

      return html;
    }

    function update(){
      var P=Number(source.value);
      var D=Number(sink.value);
      var L=Number(loading.value);
      var C=Number(continuity.value);
      var T=Number(time.value);

      sourceValue.textContent=P.toFixed(0)+'%';
      sinkValue.textContent=D.toFixed(0)+'%';
      loadingValue.textContent=L.toFixed(0)+'%';
      continuityValue.textContent=C.toFixed(0)+'%';
      timeValue.textContent=T.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      updateSinkButtons();

      labelToggle.textContent=showLabels
        ?'结构标注：显示'
        :'结构标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      var sourceFactor=clamp(
        P/(P+24),
        0,
        1
      );

      var sinkFactor=clamp(
        D/(D+22),
        0,
        1
      );

      var loadingFactor=clamp(
        L/(L+20),
        0,
        1
      );

      var continuityFactor=clamp(
        C/100,
        0,
        1
      );

      var timeFactor=clamp(
        1-Math.exp(-T/26),
        0,
        1
      );

      var sourcePressure=clamp(
        16
        +82*sourceFactor*loadingFactor,
        0,
        100
      );

      var sinkPressure=clamp(
        63
        -43*sinkFactor
        +10*(1-loadingFactor),
        8,
        76
      );

      var rawGradient=Math.max(
        0,
        sourcePressure-sinkPressure
      );

      var effectiveGradient=
        rawGradient*continuityFactor;

      var loadingRate=100
        *sourceFactor
        *loadingFactor
        *(.42+.58*timeFactor);

      var pressureRate=100
        *clamp(rawGradient/78,0,1)
        *continuityFactor
        *(.4+.6*timeFactor);

      var combinedRate=100
        *sourceFactor
        *sinkFactor
        *loadingFactor
        *continuityFactor
        *(.4+.6*timeFactor);

      var transportRate=mode==='loading'
        ?loadingRate
        :mode==='pressure'
          ?pressureRate
          :combinedRate;

      transportRate=clamp(
        transportRate,
        0,
        100
      );

      var state='运输中等';

      if(C<28){
        state='筛管受阻';
      }else if(P<18){
        state='源端不足';
      }else if(D<18){
        state='库需求低';
      }else if(L<18){
        state='装卸受限';
      }else if(transportRate>58){
        state='运输活跃';
      }else if(transportRate<20){
        state='运输较弱';
      }

      rateText.textContent=
        transportRate.toFixed(0);

      gradientText.textContent=
        effectiveGradient.toFixed(0);

      stateText.textContent=state;

      sourcePressureBar.setAttribute(
        'width',
        String(160*sourcePressure/100)
      );

      sinkPressureBar.setAttribute(
        'width',
        String(160*sinkPressure/100)
      );

      loadingBar.setAttribute(
        'width',
        String(160*L/100)
      );

      continuityBar.setAttribute(
        'width',
        String(160*C/100)
      );

      continuityBar.setAttribute(
        'fill',
        C<30
          ?'#EF4444'
          :C<65
            ?'#F59E0B'
            :'#10B981'
      );

      root.style.setProperty(
        '--ph-sugar-speed',
        clamp(
          2.5-transportRate/68,
          .52,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--ph-water-speed',
        clamp(
          2.6-effectiveGradient/55,
          .62,
          2.6
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--ph-flow-speed',
        clamp(
          2.4-transportRate/70,
          .48,
          2.4
        ).toFixed(2)+'s'
      );

      sievePlates.innerHTML=buildSievePlates();

      continuityDamage.innerHTML=
        buildContinuityDamage(C);

      sourceSugar.innerHTML=
        buildSourceSugar(P);

      var showLoading=
        mode==='combined'
        || mode==='loading';

      var showPressure=
        mode==='combined'
        || mode==='pressure';

      loadingLayer.innerHTML=
        buildLoadingGraphic(
          P,
          L,
          showLoading
        );

      sugarFlow.innerHTML=
        buildSugarFlow(
          transportRate,
          sinkType,
          showPressure || mode==='combined'
        );

      waterExchange.innerHTML=
        buildWaterExchange(
          sourcePressure,
          D,
          sinkType,
          showPressure
        );

      unloadingLayer.innerHTML=
        buildUnloadingGraphic(
          transportRate,
          sinkType,
          showLoading
        );

      accumulationLayer.innerHTML=
        buildAccumulation(P,C);

      fruitLayer.setAttribute(
        'opacity',
        sinkType==='fruit'?'1':'.24'
      );

      rootLayer.setAttribute(
        'opacity',
        sinkType==='root'?'1':'.24'
      );

      budLayer.setAttribute(
        'opacity',
        sinkType==='bud'?'1':'.24'
      );

      var sinkPath=resolveSinkPath(sinkType);

      if(sinkType==='root'){
        sinkLabelLine.setAttribute(
          'd',
          'M320 390 L470 390'
        );
        sinkLabel.setAttribute('x','477');
        sinkLabel.setAttribute('y','395');
      }else if(sinkType==='bud'){
        sinkLabelLine.setAttribute(
          'd',
          'M480 80 L590 80'
        );
        sinkLabel.setAttribute('x','597');
        sinkLabel.setAttribute('y','85');
      }else{
        sinkLabelLine.setAttribute(
          'd',
          'M505 246 L614 246'
        );
        sinkLabel.setAttribute('x','620');
        sinkLabel.setAttribute('y','251');
      }

      sinkLabel.textContent=
        '库器官：'+sinkPath.label;

      var explanation='';
      var conditionNote='';

      if(mode==='loading'){
        title.textContent=
          '源端装载与库端卸载';

        summary.textContent=
          '源叶把蔗糖装入筛管，库器官卸载后用于生长、呼吸或储藏';

        stageNote.textContent=
          '黄色ATP符号表示某些装载和卸载过程需要代谢能量';

        explanation=
          '成熟叶片制造的有机物可转化为蔗糖等可运输形式，并通过伴胞和筛管系统完成装载；到达库器官后再卸载、利用或储藏。';

        if(P<18){
          conditionNote=
            '当前源端有机物生产很少，可供装载和输出的蔗糖不足。';
        }else if(L<18){
          conditionNote=
            '装载与卸载活性较低，源端有机物难以高效进入筛管，库端接收也受到限制。';
        }else if(D<18){
          conditionNote=
            '库端需求较低，卸载和利用速度下降，源库之间的有效运输减弱。';
        }else{
          conditionNote=
            '当前源端生产、装载活性和库端需求共同支持有机物转运。';
        }
      }else if(mode==='pressure'){
        title.textContent=
          '筛管中的压力流动';

        summary.textContent=
          '源端装载和吸水形成较高压力，库端卸载维持较低压力';

        stageNote.textContent=
          '紫色路径表示同一筛管中由源端流向当前库端的净流动';

        explanation=
          '源端蔗糖装载使筛管溶液浓度升高，水可由邻近木质部进入筛管并形成较高膨压；库端卸载后压力较低，压力差推动筛管汁液整体流动。';

        if(C<28){
          conditionNote=
            '筛管连续性严重下降，压力差无法有效传递，有机物在阻断上方积累。';
        }else if(effectiveGradient<12){
          conditionNote=
            '当前有效源库压力差较小，筛管中的整体流动较弱。';
        }else if(P<18){
          conditionNote=
            '源端有机物不足，难以通过装载和吸水建立较高源端压力。';
        }else if(D<18){
          conditionNote=
            '库端卸载需求较弱，库端压力下降不明显，源库压力差减小。';
        }else{
          conditionNote=
            '源端高压、库端低压和连续筛管共同支持压力流动。';
        }
      }else{
        title.textContent=
          '韧皮部源—库有机物运输';

        summary.textContent=
          '有机物由源端装载进入筛管，并在压力差推动下流向生长或储藏器官';

        stageNote.textContent=
          '同一段筛管的净方向取决于当前源库关系';

        explanation=
          '成熟叶片通常作为源输出有机物，生长芽、果实和储藏根等器官可作为库。蔗糖装载、源库压力差、库端需求和筛管连续性共同决定运输效果。';

        if(C<28){
          conditionNote=
            '韧皮部严重受损或发生环剥，有机物向阻断下方运输明显下降，并在阻断上方积累。';
        }else if(P<18){
          conditionNote=
            '遮阴等条件使源叶有机物生产不足，库器官获得的有机物减少。';
        }else if(D>88 && transportRate<45){
          conditionNote=
            '库器官需求很高，但当前源端供给或运输能力不足，形成明显的供需矛盾。';
        }else if(L<18){
          conditionNote=
            '装载与卸载活性不足，是当前韧皮部运输的主要限制。';
        }else if(transportRate>58){
          conditionNote=
            '当前源端供给、库端需求、装卸活性和筛管连续性总体较好，运输较活跃。';
        }else{
          conditionNote=
            '当前有机物运输处于较低或中等水平，源、库和运输通路均可能参与限制。';
        }
      }

      var timeNote=T<15
        ?'作用时间较短，源端积累或库端获得有机物的差异尚不明显。'
        :'作用时间增加会使当前条件的累计影响更加明显，但不能修复筛管阻断或弥补源端供给不足。';

      result.innerHTML=explanation
        +'<br>'+conditionNote
        +' '+timeNote
        +' 整株植物可由不同筛管同时向不同库器官运输，所有数值均为相对教学指标。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<sinkButtons.length;j++){
      sinkButtons[j].onclick=function(){
        sinkType=this.getAttribute(
          'data-sink-type'
        );
        setScenarioActive('');
        update();
      };
    }

    for(var k=0;k<scenarioButtons.length;k++){
      scenarioButtons[k].onclick=function(){
        var name=this.getAttribute('data-scenario');
        var data=scenarios[name];

        if(!data){
          return;
        }

        source.value=String(data.source);
        sink.value=String(data.sink);
        loading.value=String(data.loading);
        continuity.value=String(data.continuity);
        time.value=String(data.time);
        sinkType=data.sinkType;

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    source.oninput=function(){
      setScenarioActive('');
      update();
    };

    sink.oninput=function(){
      setScenarioActive('');
      update();
    };

    loading.oninput=function(){
      setScenarioActive('');
      update();
    };

    continuity.oninput=function(){
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
