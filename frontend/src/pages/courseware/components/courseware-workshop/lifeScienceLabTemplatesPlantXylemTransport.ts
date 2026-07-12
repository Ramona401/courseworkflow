/**
 * lifeScienceLabTemplatesPlantXylemTransport.ts
 *
 * 平面生命科学实验室：木质部中的水和无机盐运输。
 *
 * 教学目标：
 * 1. 观察根吸收的水和无机盐如何进入木质部并主要向上运输；
 * 2. 理解叶片蒸腾产生的拉力是木质部远距离运输的重要动力；
 * 3. 理解水分子间的内聚力以及水与管壁间的附着作用，
 *    有助于维持木质部中连续的水柱；
 * 4. 理解根压可以辅助水分向上移动，但通常不是高大植物
 *    白天长距离运输的主要动力；
 * 5. 观察导管连续性下降或发生气穴时，运输效率如何降低；
 * 6. 理解矿质离子溶解在木质部汁液中，可随水流向上运输。
 *
 * 教学边界：
 * 1. 水分供应、气孔开放度、导管连续性、根压和运输速率
 *    均为相对教学指标；
 * 2. 本模型重点呈现蒸腾拉力、根压和水柱连续性三类因素，
 *    不计算真实水势、压力势或流体力学数值；
 * 3. 木质部运输通常以向上为主，但真实植物中局部水分运动
 *    还会受到器官状态、环境和压力梯度共同影响；
 * 4. 根压在蒸腾较弱、土壤水分较充足时可能更明显，
 *    可与吐水现象联系，但不能解释高大树木全部运输高度；
 * 5. 气穴或栓塞会破坏导管内连续水柱，不同植物具有不同的
 *    修复、绕行和安全输水机制，本模型只作简化演示；
 * 6. 无机盐进入根细胞的跨膜机制已在独立模板中演示，
 *    本模板只呈现其进入木质部后随水流运输的过程。
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
 * 构建完全限定在当前rootId内的样式。
 *
 * 独立预览时保留左侧控制区；
 * 嵌入课件后由公共布局覆盖层转换为
 * “上方实验主体 + 底部课堂控制条”。
 */
function xylemTransportStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#DCFCE7);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:246px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FDFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9}'
    + '#' + rootId + ' .xt-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .xt-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .xt-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .xt-button{min-height:30px;padding:3px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:9.5px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .xt-button.active{border-color:#0EA5E9;background:#E0F2FE;box-shadow:0 3px 9px rgba(14,165,233,.14)}'
    + '#' + rootId + ' .xt-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .xt-toggle.off{background:#64748B}'
    + '#' + rootId + ' .xt-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .xt-card{padding:6px 3px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .xt-card b{display:block;min-height:18px;font-size:13px;color:#0369A1}'
    + '#' + rootId + ' .xt-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .xt-water{animation:' + rootId + '-water var(--xt-water-speed,1.4s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .xt-mineral{animation:' + rootId + '-mineral var(--xt-mineral-speed,1.6s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .xt-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--xt-flow-speed,1.2s) linear infinite}'
    + '#' + rootId + ' .xt-vapor{animation:' + rootId + '-vapor var(--xt-vapor-speed,1.8s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .xt-pressure{animation:' + rootId + '-pressure 1.2s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-water{from{opacity:.38}to{opacity:1}}'
    + '@keyframes ' + rootId + '-mineral{from{opacity:.48}to{opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-vapor{from{opacity:.3;transform:translateY(0)}to{opacity:1;transform:translateY(-7px)}}'
    + '@keyframes ' + rootId + '-pressure{from{opacity:.45}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_PLANT_XYLEM_TRANSPORT:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-xylem-water-mineral-transport',
    group: '🌿 植物生理',
    name: '木质部中的水和无机盐运输',
    emoji: '🌳',
    desc: '调节根部水分供应、气孔开放度、导管连续性、根压和作用时间，观察木质部水流及无机盐向上运输',
    params: [
      {
        key: 'rootWaterSupply',
        label: '根部水分供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'stomatalOpening',
        label: '气孔开放度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'vesselContinuity',
        label: '导管水柱连续性',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 92,
        hint: '数值降低可模拟气穴或栓塞影响',
      },
      {
        key: 'rootPressure',
        label: '根压相对水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 38,
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
      const rootWaterSupply = num(
        params,
        'rootWaterSupply',
        76,
      )
      const stomatalOpening = num(
        params,
        'stomatalOpening',
        72,
      )
      const vesselContinuity = num(
        params,
        'vesselContinuity',
        92,
      )
      const rootPressure = num(
        params,
        'rootPressure',
        38,
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
${xylemTransportStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌳 木质部中的水和无机盐运输</div>
    <div class="bl-note">蒸腾拉力为主，根压辅助，连续水柱保障传递</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>根部水分供应</span>
          <span class="bl-value" data-water-value></span>
        </div>
        <input data-water type="range" min="0" max="100" step="1" value="${n(rootWaterSupply)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>气孔开放度</span>
          <span class="bl-value" data-stomata-value></span>
        </div>
        <input data-stomata type="range" min="0" max="100" step="1" value="${n(stomatalOpening)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>导管水柱连续性</span>
          <span class="bl-value" data-continuity-value></span>
        </div>
        <input data-continuity type="range" min="0" max="100" step="1" value="${n(vesselContinuity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>根压相对水平</span>
          <span class="bl-value" data-pressure-value></span>
        </div>
        <input data-pressure type="range" min="0" max="100" step="1" value="${n(rootPressure)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>作用时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="xt-subtitle">重点观察</div>

      <div class="xt-buttons">
        <button type="button" class="xt-button active" data-mode="combined">综合运输</button>
        <button type="button" class="xt-button" data-mode="transpiration">蒸腾拉力</button>
        <button type="button" class="xt-button" data-mode="rootPressure">根压作用</button>
      </div>

      <div class="xt-subtitle">快速比较情境</div>

      <div class="xt-scenarios">
        <button type="button" class="xt-button active" data-scenario="day">晴天</button>
        <button type="button" class="xt-button" data-scenario="humid">湿润</button>
        <button type="button" class="xt-button" data-scenario="drought">干旱</button>
        <button type="button" class="xt-button" data-scenario="embolism">气穴</button>
        <button type="button" class="xt-button" data-scenario="night">夜间</button>
      </div>

      <button type="button" class="xt-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '结构标注：显示' : '结构标注：隐藏'}
      </button>

      <div class="xt-status">
        <div class="xt-card">
          <b data-flow-rate></b>
          <span>相对上升流速</span>
        </div>

        <div class="xt-card">
          <b data-mineral-rate></b>
          <span>无机盐运输</span>
        </div>

        <div class="xt-card">
          <b data-main-force></b>
          <span>主要动力</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="木质部中水和无机盐运输互动示意图"
      >
        <defs>
          <linearGradient id="${rootId}-stem" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stop-color="#86EFAC"/>
            <stop offset="48%" stop-color="#16A34A"/>
            <stop offset="100%" stop-color="#86EFAC"/>
          </linearGradient>

          <linearGradient id="${rootId}-xylem" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#7DD3FC"/>
          </linearGradient>

          <linearGradient id="${rootId}-soil" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#FDE68A"/>
            <stop offset="100%" stop-color="#B45309"/>
          </linearGradient>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker id="${rootId}-arrow-purple" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#075985" flood-opacity=".14"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text x="24" y="36" data-title font-size="27" font-weight="900" fill="#075985"></text>
        <text x="24" y="65" data-summary font-size="14" font-weight="800" fill="#475569"></text>

        <rect x="0" y="344" width="514" height="86" fill="url(#${rootId}-soil)" opacity=".82"/>

        <g filter="url(#${rootId}-shadow)">
          <path d="M245 351 C242 292 246 228 248 111" fill="none" stroke="url(#${rootId}-stem)" stroke-width="72" stroke-linecap="round"/>
          <path d="M248 111 C181 92 132 116 98 163 C161 186 217 171 250 130Z" fill="#4ADE80" stroke="#15803D" stroke-width="5"/>
          <path d="M248 111 C315 91 373 112 409 158 C346 185 287 171 247 130Z" fill="#22C55E" stroke="#15803D" stroke-width="5"/>
          <path d="M248 345 C214 365 180 383 145 410" fill="none" stroke="#92400E" stroke-width="16" stroke-linecap="round"/>
          <path d="M248 345 C278 367 316 389 356 412" fill="none" stroke="#92400E" stroke-width="16" stroke-linecap="round"/>
          <path d="M248 345 C246 374 246 397 246 424" fill="none" stroke="#92400E" stroke-width="16" stroke-linecap="round"/>
        </g>

        <rect
          x="226"
          y="105"
          width="44"
          height="247"
          rx="20"
          fill="url(#${rootId}-xylem)"
          stroke="#0369A1"
          stroke-width="5"
        />

        <path d="M237 126 V332 M248 126 V332 M259 126 V332" stroke="#FFFFFF" stroke-width="4" stroke-linecap="round" opacity=".72"/>

        <g data-vessel-breaks></g>
        <g data-water-flow></g>
        <g data-mineral-flow></g>
        <g data-leaf-vapor></g>
        <g data-root-pressure></g>
        <g data-guttation></g>

        <g data-label-layer>
          <path d="M215 169 L153 194" stroke="#64748B" stroke-width="2.5"/>
          <text x="72" y="202" font-size="13" font-weight="900" fill="#475569">叶片与气孔</text>

          <path d="M270 222 L345 222" stroke="#64748B" stroke-width="2.5"/>
          <text x="354" y="227" font-size="13" font-weight="900" fill="#0369A1">木质部导管</text>

          <path d="M215 362 L150 338" stroke="#64748B" stroke-width="2.5"/>
          <text x="69" y="335" font-size="13" font-weight="900" fill="#475569">根吸收区</text>

          <path d="M265 127 L332 91" stroke="#64748B" stroke-width="2.5"/>
          <text x="340" y="92" font-size="13" font-weight="900" fill="#475569">水分蒸发出口</text>
        </g>

        <g transform="translate(514 91)">
          <rect width="222" height="247" rx="22" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>

          <text x="111" y="27" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">木质部运输条件</text>

          <text x="17" y="56" font-size="11" font-weight="800" fill="#64748B">蒸腾拉力</text>
          <rect x="17" y="66" width="184" height="15" rx="7.5" fill="#E2E8F0"/>
          <rect data-pull-bar x="17" y="66" width="0" height="15" rx="7.5" fill="#38BDF8"/>

          <text x="17" y="103" font-size="11" font-weight="800" fill="#64748B">根压贡献</text>
          <rect x="17" y="113" width="184" height="15" rx="7.5" fill="#E2E8F0"/>
          <rect data-pressure-bar x="17" y="113" width="0" height="15" rx="7.5" fill="#F59E0B"/>

          <text x="17" y="150" font-size="11" font-weight="800" fill="#64748B">水柱连续性</text>
          <rect x="17" y="160" width="184" height="15" rx="7.5" fill="#E2E8F0"/>
          <rect data-continuity-bar x="17" y="160" width="0" height="15" rx="7.5" fill="#10B981"/>

          <text x="17" y="197" font-size="11" font-weight="800" fill="#64748B">根部水分供应</text>
          <rect x="17" y="207" width="184" height="15" rx="7.5" fill="#E2E8F0"/>
          <rect data-supply-bar x="17" y="207" width="0" height="15" rx="7.5" fill="#0EA5E9"/>

          <text data-condition-label x="111" y="239" text-anchor="middle" font-size="12" font-weight="900" fill="#075985"></text>
        </g>

        <g transform="translate(521 355)">
          <circle cx="7" cy="7" r="7" fill="#38BDF8"/>
          <text x="22" y="12" font-size="12" font-weight="800" fill="#475569">水分子</text>

          <circle cx="107" cy="7" r="7" fill="#8B5CF6"/>
          <text x="122" y="12" font-size="12" font-weight="800" fill="#475569">矿质离子</text>
        </g>

        <text x="513" y="402" data-stage-note font-size="12.5" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var water=root.querySelector('[data-water]');
    var stomata=root.querySelector('[data-stomata]');
    var continuity=root.querySelector('[data-continuity]');
    var pressure=root.querySelector('[data-pressure]');
    var time=root.querySelector('[data-time]');

    var waterValue=root.querySelector('[data-water-value]');
    var stomataValue=root.querySelector('[data-stomata-value]');
    var continuityValue=root.querySelector('[data-continuity-value]');
    var pressureValue=root.querySelector('[data-pressure-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var flowRateText=root.querySelector('[data-flow-rate]');
    var mineralRateText=root.querySelector('[data-mineral-rate]');
    var mainForceText=root.querySelector('[data-main-force]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var vesselBreaks=root.querySelector('[data-vessel-breaks]');
    var waterFlow=root.querySelector('[data-water-flow]');
    var mineralFlow=root.querySelector('[data-mineral-flow]');
    var leafVapor=root.querySelector('[data-leaf-vapor]');
    var rootPressureLayer=root.querySelector('[data-root-pressure]');
    var guttation=root.querySelector('[data-guttation]');
    var labelLayer=root.querySelector('[data-label-layer]');

    var pullBar=root.querySelector('[data-pull-bar]');
    var pressureBar=root.querySelector('[data-pressure-bar]');
    var continuityBar=root.querySelector('[data-continuity-bar]');
    var supplyBar=root.querySelector('[data-supply-bar]');
    var conditionLabel=root.querySelector('[data-condition-label]');

    var mode='combined';
    var showLabels=${showLabels ? 'true' : 'false'};

    var scenarios={
      day:{
        water:76,
        stomata:84,
        continuity:94,
        pressure:34,
        time:66
      },
      humid:{
        water:82,
        stomata:28,
        continuity:95,
        pressure:48,
        time:66
      },
      drought:{
        water:18,
        stomata:32,
        continuity:90,
        pressure:16,
        time:66
      },
      embolism:{
        water:72,
        stomata:74,
        continuity:22,
        pressure:36,
        time:66
      },
      night:{
        water:84,
        stomata:8,
        continuity:96,
        pressure:76,
        time:66
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
     * 绘制木质部水柱中的水分子。
     */
    function buildWaterParticles(rate,continuityLevel){
      var html='';
      var count=Math.floor(2+rate/8);
      var opacity=.32+.68*continuityLevel/100;

      for(var i=0;i<count;i++){
        var y=329-(i*23)%205;
        var x=237+(i%3)*11;

        html+='<circle class="xt-water" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%3)
          +'" fill="#38BDF8" stroke="#0284C7"'
          +' stroke-width="1.5" opacity="'+opacity+'"/>';
      }

      if(rate>2){
        html+='<path class="xt-flow" d="M248 333 V118"'
          +' fill="none" stroke="#0284C7"'
          +' stroke-width="'+(4+rate/24)
          +'" marker-end="url(#${rootId}-arrow-blue)"'
          +' opacity="'+(.35+.65*continuityLevel/100)+'"/>';
      }

      return html;
    }

    /**
     * 绘制随木质部汁液向上运输的矿质离子。
     */
    function buildMineralParticles(rate){
      var html='';
      var count=Math.floor(1+rate/15);

      for(var i=0;i<count;i++){
        var y=320-(i*37)%190;
        var x=241+(i%2)*15;

        html+='<circle class="xt-mineral" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%2)
          +'" fill="#8B5CF6" stroke="#5B21B6"'
          +' stroke-width="1.5"/>';

        if(i%2===0){
          html+='<text x="'+(x+7)+'" y="'+(y+4)
            +'" font-size="8" font-weight="900"'
            +' fill="#5B21B6">M</text>';
        }
      }

      if(rate>3){
        html+='<path class="xt-flow" d="M259 327 V126"'
          +' fill="none" stroke="#7C3AED"'
          +' stroke-width="'+(3+rate/30)
          +'" marker-end="url(#${rootId}-arrow-purple)"'
          +' opacity=".8"/>';
      }

      return html;
    }

    /**
     * 根据气孔开放度和蒸腾拉力绘制叶片水汽散失。
     */
    function buildLeafVapor(stomatalLevel,pull){
      var html='';
      var count=Math.floor(stomatalLevel/10+pull/24);

      for(var i=0;i<count;i++){
        var side=i%2===0?-1:1;
        var baseX=side<0
          ?176-(i%4)*20
          :320+(i%4)*22;
        var baseY=123+(i%3)*13;

        html+='<circle class="xt-vapor" cx="'+baseX
          +'" cy="'+(baseY-18-Math.floor(i/4)*14)
          +'" r="'+(4+i%3)
          +'" fill="#BAE6FD" stroke="#0284C7"'
          +' stroke-width="1.3" opacity=".8"/>';
      }

      if(pull>5){
        html+='<path class="xt-flow" d="M176 137'
          +' C148 110 139 88 136 72"'
          +' fill="none" stroke="#0EA5E9"'
          +' stroke-width="'+(3+pull/28)
          +'" marker-end="url(#${rootId}-arrow-blue)" opacity=".74"/>';

        html+='<path class="xt-flow" d="M319 137'
          +' C350 109 364 87 369 69"'
          +' fill="none" stroke="#0EA5E9"'
          +' stroke-width="'+(3+pull/28)
          +'" marker-end="url(#${rootId}-arrow-blue)" opacity=".74"/>';
      }

      return html;
    }

    /**
     * 绘制根压对木质部水柱的向上推力。
     */
    function buildRootPressureGraphic(level,visible){
      if(!visible || level<2){
        return '';
      }

      var width=3+level/22;

      return ''
        +'<path class="xt-pressure" d="M202 384'
        +' C218 364 233 352 246 338"'
        +' fill="none" stroke="#F59E0B"'
        +' stroke-width="'+width
        +'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path class="xt-pressure" d="M292 386'
        +' C278 365 264 353 251 338"'
        +' fill="none" stroke="#F59E0B"'
        +' stroke-width="'+width
        +'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<text x="291" y="382" font-size="11"'
        +' font-weight="900" fill="#B45309">'
        +'根压辅助推升'
        +'</text>';
    }

    /**
     * 低蒸腾且根压较高时，用叶缘水滴提示吐水现象。
     */
    function buildGuttationGraphic(
      stomatalLevel,
      pressureLevel,
      visible
    ){
      if(
        !visible
        || stomatalLevel>24
        || pressureLevel<46
      ){
        return '';
      }

      var count=Math.floor(
        1+(pressureLevel-46)/18
      );
      var html='';

      for(var i=0;i<count;i++){
        var left=i%2===0;
        var x=left
          ?103+i*18
          :399-i*17;
        var y=156-i*5;

        html+='<path class="xt-water" d="M'+x+' '+y
          +' C'+(x-7)+' '+(y+11)+' '+x+' '+(y+21)
          +' '+x+' '+(y+21)
          +' C'+x+' '+(y+21)+' '+(x+7)+' '+(y+11)
          +' '+x+' '+y+'Z" fill="#38BDF8"'
          +' stroke="#0284C7" stroke-width="1.5"/>';
      }

      html+='<text x="348" y="188" font-size="11"'
        +' font-weight="900" fill="#0369A1">'
        +'根压较高时可能出现吐水'
        +'</text>';

      return html;
    }

    /**
     * 连续性降低时在导管中绘制气泡和断裂区。
     */
    function buildVesselBreaks(continuityLevel){
      var damage=clamp(
        (100-continuityLevel)/100,
        0,
        1
      );

      if(damage<.12){
        return '';
      }

      var count=Math.max(
        1,
        Math.floor(damage*6)
      );
      var html='';

      for(var i=0;i<count;i++){
        var y=149+i*34;
        var radius=5+damage*8+(i%2)*3;

        html+='<ellipse cx="'+(248+(i%2===0?-4:5))
          +'" cy="'+y+'" rx="'+radius
          +'" ry="'+(radius*.72)
          +'" fill="#FFFFFF" stroke="#64748B"'
          +' stroke-width="2.5" opacity=".95"/>';
      }

      if(continuityLevel<35){
        html+='<rect x="226" y="236" width="44" height="20"'
          +' fill="#FFFFFF" stroke="#DC2626"'
          +' stroke-width="3" stroke-dasharray="5 4"/>';

        html+='<text x="281" y="251" font-size="11"'
          +' font-weight="900" fill="#DC2626">'
          +'水柱连续性明显破坏'
          +'</text>';
      }

      return html;
    }

    function update(){
      var W=Number(water.value);
      var S=Number(stomata.value);
      var C=Number(continuity.value);
      var P=Number(pressure.value);
      var T=Number(time.value);

      waterValue.textContent=W.toFixed(0)+'%';
      stomataValue.textContent=S.toFixed(0)+'%';
      continuityValue.textContent=C.toFixed(0)+'%';
      pressureValue.textContent=P.toFixed(0)+'%';
      timeValue.textContent=T.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      labelToggle.textContent=showLabels
        ?'结构标注：显示'
        :'结构标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      var supplyFactor=clamp(
        W/(W+22)*1.2,
        0,
        1
      );

      var stomatalFactor=clamp(
        S/100,
        0,
        1
      );

      var continuityFactor=clamp(
        C/100,
        0,
        1
      );

      var pressureFactor=clamp(
        P/(P+28),
        0,
        1
      );

      var timeFactor=clamp(
        1-Math.exp(-T/27),
        0,
        1
      );

      var droughtProtection=W<25
        ?clamp(.35+W/50,.35,.85)
        :1;

      var transpirationPull=100
        *stomatalFactor
        *supplyFactor
        *continuityFactor
        *droughtProtection
        *(.38+.62*timeFactor);

      var rootPressureContribution=58
        *pressureFactor
        *supplyFactor
        *continuityFactor
        *(.42+.58*timeFactor);

      transpirationPull=clamp(
        transpirationPull,
        0,
        100
      );

      rootPressureContribution=clamp(
        rootPressureContribution,
        0,
        58
      );

      var transportRate=0;

      if(mode==='transpiration'){
        transportRate=transpirationPull;
      }else if(mode==='rootPressure'){
        transportRate=rootPressureContribution;
      }else{
        transportRate=transpirationPull
          +rootPressureContribution*.42;
      }

      transportRate=clamp(
        transportRate,
        0,
        100
      );

      var mineralRate=clamp(
        transportRate
        *(.34+.66*supplyFactor)
        *(.55+.45*continuityFactor),
        0,
        100
      );

      var mainForce='蒸腾拉力';

      if(mode==='rootPressure'){
        mainForce='根压';
      }else if(
        transpirationPull<12
        && rootPressureContribution>transpirationPull
      ){
        mainForce='根压辅助';
      }else if(
        transpirationPull>rootPressureContribution*1.35
      ){
        mainForce='蒸腾拉力';
      }else{
        mainForce='共同作用';
      }

      flowRateText.textContent=
        transportRate.toFixed(0);

      mineralRateText.textContent=
        mineralRate.toFixed(0);

      mainForceText.textContent=
        mainForce;

      pullBar.setAttribute(
        'width',
        String(184*transpirationPull/100)
      );

      pressureBar.setAttribute(
        'width',
        String(184*rootPressureContribution/58)
      );

      continuityBar.setAttribute(
        'width',
        String(184*C/100)
      );

      supplyBar.setAttribute(
        'width',
        String(184*W/100)
      );

      pullBar.setAttribute(
        'fill',
        transpirationPull>55
          ?'#0EA5E9'
          :'#7DD3FC'
      );

      continuityBar.setAttribute(
        'fill',
        C<35
          ?'#EF4444'
          :C<70
            ?'#F59E0B'
            :'#10B981'
      );

      conditionLabel.textContent=
        C<35
          ?'导管气穴明显'
          :W<24
            ?'根部供水不足'
            :S<16
              ?'蒸腾拉力较弱'
              :transportRate>58
                ?'向上运输较活跃'
                :'向上运输较缓慢';

      root.style.setProperty(
        '--xt-water-speed',
        clamp(
          2.5-transportRate/66,
          .5,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--xt-mineral-speed',
        clamp(
          2.5-mineralRate/68,
          .55,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--xt-flow-speed',
        clamp(
          2.3-transportRate/70,
          .48,
          2.3
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--xt-vapor-speed',
        clamp(
          2.5-transpirationPull/70,
          .6,
          2.5
        ).toFixed(2)+'s'
      );

      vesselBreaks.innerHTML=
        buildVesselBreaks(C);

      waterFlow.innerHTML=
        buildWaterParticles(
          transportRate,
          C
        );

      mineralFlow.innerHTML=
        buildMineralParticles(
          mineralRate
        );

      var showTranspiration=
        mode==='combined'
        || mode==='transpiration';

      var showPressure=
        mode==='combined'
        || mode==='rootPressure';

      leafVapor.innerHTML=
        buildLeafVapor(
          showTranspiration?S:0,
          showTranspiration
            ?transpirationPull
            :0
        );

      rootPressureLayer.innerHTML=
        buildRootPressureGraphic(
          P,
          showPressure
        );

      guttation.innerHTML=
        buildGuttationGraphic(
          S,
          P,
          showPressure
        );

      var explanation='';
      var conditionNote='';

      if(mode==='transpiration'){
        title.textContent=
          '蒸腾拉力驱动木质部水柱上升';

        summary.textContent=
          '叶片失水产生拉力，连续水柱把这种拉力向根部方向传递';

        stageNote.textContent=
          '蓝色水流向上，叶片水汽向外散失';

        explanation=
          '叶片蒸腾使叶肉细胞和叶脉木质部中的水势降低，形成向上的拉力。水分子间的内聚作用以及水与导管壁间的附着作用，有助于维持连续水柱。';

        if(S<16){
          conditionNote=
            '当前气孔开放度较低，蒸腾较弱，蒸腾拉力有限。';
        }else if(W<24){
          conditionNote=
            '虽然气孔仍有开放，但根部供水不足会限制木质部水流，并增加植物失水风险。';
        }else if(C<35){
          conditionNote=
            '导管水柱连续性明显下降，蒸腾产生的拉力难以有效传递。';
        }else{
          conditionNote=
            '当前气孔开放、根部供水和导管连续性共同支持向上运输。';
        }
      }else if(mode==='rootPressure'){
        title.textContent=
          '根压对木质部运输的辅助作用';

        summary.textContent=
          '根部离子积累和水分进入可产生正压力，在低蒸腾条件下推动木质部汁液上升';

        stageNote.textContent=
          '橙色箭头表示根部对水柱的向上推力';

        explanation=
          '根部离子主动积累可降低木质部汁液水势，促进水分进入并形成一定正压力。根压可辅助水和无机盐向上移动。';

        if(P<18){
          conditionNote=
            '当前根压较低，对木质部水流的推动作用较弱。';
        }else if(W<24){
          conditionNote=
            '根部水分供应不足，即使根压参数较高，也难以形成明显的持续水流。';
        }else if(C<35){
          conditionNote=
            '导管连续性较差会削弱根压向上传递的效果。';
        }else if(S<24 && P>46){
          conditionNote=
            '蒸腾较弱而根压较高时，叶缘可能出现吐水，但根压不能解释高大植物全部运输高度。';
        }else{
          conditionNote=
            '根压可以辅助运输，但在蒸腾旺盛的高大植物中通常不是主要动力。';
        }
      }else{
        title.textContent=
          '木质部水和无机盐的综合运输';

        summary.textContent=
          '蒸腾拉力通常为主要动力，根压提供辅助，连续水柱保障动力传递';

        stageNote.textContent=
          '无机盐溶解在木质部汁液中随水流向上移动';

        explanation=
          '根吸收的水和矿质离子进入木质部后形成木质部汁液。叶片蒸腾产生的拉力通常是远距离向上运输的重要动力，根压可在一定条件下提供辅助。';

        if(C<35){
          conditionNote=
            '当前导管连续性是主要限制，气穴或栓塞破坏了连续水柱，水和无机盐运输明显下降。';
        }else if(W<24){
          conditionNote=
            '当前根部供水不足，木质部可获得的水量有限，即使蒸腾拉力较强也难以维持运输。';
        }else if(S<16 && P<24){
          conditionNote=
            '蒸腾拉力和根压都较弱，木质部汁液向上运输较慢。';
        }else if(S<20 && P>48){
          conditionNote=
            '当前蒸腾较弱、根压相对较高，根压贡献更容易被观察。';
        }else{
          conditionNote=
            '当前根部供水、气孔开放和导管连续性总体较好，蒸腾拉力主导向上运输。';
        }
      }

      var timeNote=T<15
        ?'作用时间较短，累计运输效果尚不明显。'
        :'作用时间增加会使当前条件下的累计运输更明显，但不能弥补供水不足或水柱断裂。';

      result.innerHTML=explanation
        +'<br>'+conditionNote
        +' '+timeNote
        +' 所有速率均为相对教学指标。';
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

        water.value=String(data.water);
        stomata.value=String(data.stomata);
        continuity.value=String(data.continuity);
        pressure.value=String(data.pressure);
        time.value=String(data.time);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    water.oninput=function(){
      setScenarioActive('');
      update();
    };

    stomata.oninput=function(){
      setScenarioActive('');
      update();
    };

    continuity.oninput=function(){
      setScenarioActive('');
      update();
    };

    pressure.oninput=function(){
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
