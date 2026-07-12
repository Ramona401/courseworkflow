/**
 * lifeScienceLabTemplatesPlantRootAbsorption.ts
 *
 * 平面生命科学实验室：根对水和无机盐的吸收。
 *
 * 教学目标：
 * 1. 观察根毛如何扩大根与土壤溶液的接触面积；
 * 2. 理解水分进入根毛细胞与细胞内外水势差有关；
 * 3. 理解矿质离子跨膜运输需要具有选择性的运输蛋白；
 * 4. 理解逆浓度梯度吸收矿质离子需要细胞呼吸提供能量；
 * 5. 比较干旱、盐渍、缺氧和矿质不足等典型土壤条件；
 * 6. 区分水分净移动、矿质离子吸收和根内进一步运输。
 *
 * 教学边界：
 * 1. 土壤含水量、矿质离子、土壤氧气和吸收速率均为相对教学指标；
 * 2. 本模型用综合指数简化表示土壤溶液与根毛细胞之间的水势差；
 * 3. 矿质离子可以通过不同运输方式进入根细胞，本模型重点呈现
 *    运输蛋白参与以及逆浓度梯度时对能量的需求；
 * 4. 土壤矿质离子过低会限制离子来源，过高则可能降低土壤水势，
 *    形成盐胁迫并抑制根吸水；
 * 5. 水淹土壤虽然含水量高，但土壤氧气不足会抑制根细胞呼吸，
 *    从而影响主动吸收矿质离子；
 * 6. 本模板只演示根表吸收和进入根内部的起始过程，
 *    木质部远距离运输将在独立模板中进一步呈现。
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
 * 嵌入课件后由lifeScienceLabUtils.ts中的公共覆盖层
 * 转换为“上方实验主体 + 底部课堂控制条”。
 */
function rootAbsorptionStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BBF7D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#E0F2FE);border-bottom:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:246px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FAFFFB;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#22C55E}'
    + '#' + rootId + ' .ra-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .ra-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ra-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .ra-button{min-height:30px;padding:3px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:9.5px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .ra-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.13)}'
    + '#' + rootId + ' .ra-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#4ADE80,#16A34A);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ra-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ra-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .ra-card{padding:6px 3px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ra-card b{display:block;min-height:18px;font-size:13px;color:#15803D}'
    + '#' + rootId + ' .ra-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ra-water{animation:' + rootId + '-water var(--ra-water-speed,1.6s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .ra-mineral{animation:' + rootId + '-mineral var(--ra-mineral-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .ra-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--ra-flow-speed,1.3s) linear infinite}'
    + '#' + rootId + ' .ra-atp{animation:' + rootId + '-atp 1.1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-water{from{opacity:.38}to{opacity:1}}'
    + '@keyframes ' + rootId + '-mineral{from{opacity:.5}to{opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-atp{from{opacity:.45}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_PLANT_ROOT_ABSORPTION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-root-water-mineral-absorption',
    group: '🌿 植物生理',
    name: '根对水和无机盐的吸收',
    emoji: '🌱',
    desc: '调节土壤含水量、矿质离子、土壤氧气、根毛密度和作用时间，观察根毛吸水及矿质离子吸收',
    params: [
      {
        key: 'soilMoisture',
        label: '土壤含水量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'mineralLevel',
        label: '土壤矿质离子',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 45,
        hint: '相对教学单位；过高可用于模拟盐胁迫',
      },
      {
        key: 'soilOxygen',
        label: '土壤氧气供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'rootHairDensity',
        label: '根毛相对密度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'observationTime',
        label: '作用时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const soilMoisture = num(
        params,
        'soilMoisture',
        72,
      )
      const mineralLevel = num(
        params,
        'mineralLevel',
        45,
      )
      const soilOxygen = num(
        params,
        'soilOxygen',
        78,
      )
      const rootHairDensity = num(
        params,
        'rootHairDensity',
        68,
      )
      const observationTime = num(
        params,
        'observationTime',
        58,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${rootAbsorptionStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌱 根对水和无机盐的吸收</div>
    <div class="bl-note">相对教学模型：吸收条件与运输机制比较</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>土壤含水量</span>
          <span class="bl-value" data-moisture-value></span>
        </div>
        <input data-moisture type="range" min="0" max="100" step="1" value="${n(soilMoisture)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>土壤矿质离子</span>
          <span class="bl-value" data-mineral-value></span>
        </div>
        <input data-mineral type="range" min="0" max="100" step="1" value="${n(mineralLevel)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>土壤氧气供应</span>
          <span class="bl-value" data-oxygen-value></span>
        </div>
        <input data-oxygen type="range" min="0" max="100" step="1" value="${n(soilOxygen)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>根毛相对密度</span>
          <span class="bl-value" data-hair-value></span>
        </div>
        <input data-hair type="range" min="0" max="100" step="1" value="${n(rootHairDensity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>作用时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="ra-subtitle">重点观察</div>

      <div class="ra-buttons">
        <button type="button" class="ra-button active" data-mode="combined">综合过程</button>
        <button type="button" class="ra-button" data-mode="water">水分吸收</button>
        <button type="button" class="ra-button" data-mode="mineral">离子吸收</button>
      </div>

      <div class="ra-subtitle">快速比较情境</div>

      <div class="ra-scenarios">
        <button type="button" class="ra-button active" data-scenario="normal">适宜</button>
        <button type="button" class="ra-button" data-scenario="drought">干旱</button>
        <button type="button" class="ra-button" data-scenario="waterlogged">水淹</button>
        <button type="button" class="ra-button" data-scenario="deficient">缺肥</button>
        <button type="button" class="ra-button" data-scenario="saline">盐渍</button>
      </div>

      <button type="button" class="ra-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '结构标注：显示' : '结构标注：隐藏'}
      </button>

      <div class="ra-status">
        <div class="ra-card">
          <b data-water-rate></b>
          <span>相对净水流</span>
        </div>

        <div class="ra-card">
          <b data-mineral-rate></b>
          <span>矿质吸收</span>
        </div>

        <div class="ra-card">
          <b data-limit></b>
          <span>主要限制</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="根对水和无机盐吸收互动示意图"
      >
        <defs>
          <linearGradient id="${rootId}-soil" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stop-color="#D6B47C"/>
            <stop offset="100%" stop-color="#FDE7B2"/>
          </linearGradient>

          <linearGradient id="${rootId}-cell" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#DCFCE7"/>
            <stop offset="100%" stop-color="#BBF7D0"/>
          </linearGradient>

          <linearGradient id="${rootId}-xylem" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#7DD3FC"/>
          </linearGradient>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker id="${rootId}-arrow-purple" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#14532D" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>
        <rect x="0" y="82" width="252" height="348" fill="url(#${rootId}-soil)"/>
        <rect x="252" y="82" width="508" height="348" fill="#F8FFFC"/>

        <text x="25" y="38" data-title font-size="27" font-weight="900" fill="#166534"></text>
        <text x="25" y="67" data-summary font-size="14" font-weight="800" fill="#475569"></text>

        <text x="24" y="108" font-size="14" font-weight="900" fill="#854D0E">土壤溶液</text>
        <text x="285" y="108" font-size="14" font-weight="900" fill="#166534">根毛区与根内部</text>

        <g data-soil-water></g>
        <g data-soil-minerals></g>
        <g data-soil-oxygen></g>

        <g filter="url(#${rootId}-shadow)">
          <path
            d="M92 230
               C142 214 188 212 250 224
               L250 284
               C188 296 142 294 92 276
               C70 268 70 238 92 230Z"
            fill="url(#${rootId}-cell)"
            stroke="#166534"
            stroke-width="6"
          />

          <rect
            x="246"
            y="188"
            width="174"
            height="126"
            rx="24"
            fill="url(#${rootId}-cell)"
            stroke="#166534"
            stroke-width="7"
          />

          <rect
            x="260"
            y="202"
            width="146"
            height="98"
            rx="18"
            fill="#F0FDF4"
            stroke="#22C55E"
            stroke-width="4"
          />

          <ellipse
            cx="337"
            cy="252"
            rx="48"
            ry="30"
            fill="#DBEAFE"
            stroke="#3B82F6"
            stroke-width="3"
            opacity=".78"
          />

          <circle
            cx="292"
            cy="231"
            r="14"
            fill="#DDD6FE"
            stroke="#7C3AED"
            stroke-width="4"
          />
        </g>

        <g data-root-hairs></g>

        <g data-cortex-cells>
          <ellipse cx="447" cy="213" rx="31" ry="39" fill="#ECFCCB" stroke="#65A30D" stroke-width="4"/>
          <ellipse cx="447" cy="291" rx="31" ry="39" fill="#ECFCCB" stroke="#65A30D" stroke-width="4"/>
          <ellipse cx="497" cy="252" rx="31" ry="39" fill="#DCFCE7" stroke="#16A34A" stroke-width="4"/>
        </g>

        <g filter="url(#${rootId}-shadow)">
          <rect x="536" y="145" width="89" height="220" rx="28" fill="url(#${rootId}-xylem)" stroke="#0369A1" stroke-width="6"/>
          <path d="M559 177 V333 M581 177 V333 M603 177 V333" stroke="#FFFFFF" stroke-width="5" stroke-linecap="round" opacity=".72"/>
        </g>

        <g data-water-flow></g>
        <g data-mineral-flow></g>
        <g data-atp-layer></g>

        <g data-label-layer>
          <path d="M128 212 L104 159" stroke="#64748B" stroke-width="2.5"/>
          <text x="42" y="153" font-size="13" font-weight="900" fill="#475569">根毛</text>

          <path d="M252 193 L276 147" stroke="#64748B" stroke-width="2.5"/>
          <text x="264" y="140" font-size="13" font-weight="900" fill="#475569">表皮细胞</text>

          <path d="M452 176 L464 136" stroke="#64748B" stroke-width="2.5"/>
          <text x="422" y="128" font-size="13" font-weight="900" fill="#475569">皮层</text>

          <path d="M582 144 L604 112" stroke="#64748B" stroke-width="2.5"/>
          <text x="586" y="104" font-size="13" font-weight="900" fill="#0369A1">木质部</text>
        </g>

        <g transform="translate(642 132)">
          <rect width="100" height="192" rx="18" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>

          <text x="50" y="25" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">条件状态</text>

          <text x="12" y="53" font-size="10.5" font-weight="800" fill="#64748B">水分条件</text>
          <rect x="12" y="62" width="76" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-water-bar x="12" y="62" width="0" height="12" rx="6" fill="#38BDF8"/>

          <text x="12" y="94" font-size="10.5" font-weight="800" fill="#64748B">矿质来源</text>
          <rect x="12" y="103" width="76" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-mineral-bar x="12" y="103" width="0" height="12" rx="6" fill="#8B5CF6"/>

          <text x="12" y="135" font-size="10.5" font-weight="800" fill="#64748B">呼吸供能</text>
          <rect x="12" y="144" width="76" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-energy-bar x="12" y="144" width="0" height="12" rx="6" fill="#F59E0B"/>

          <text data-potential-label x="50" y="180" text-anchor="middle" font-size="10.5" font-weight="900" fill="#0369A1"></text>
        </g>

        <g transform="translate(30 395)">
          <circle cx="7" cy="7" r="7" fill="#38BDF8"/>
          <text x="22" y="12" font-size="12" font-weight="800" fill="#475569">水分子</text>
        </g>

        <g transform="translate(142 395)">
          <circle cx="7" cy="7" r="7" fill="#8B5CF6"/>
          <text x="22" y="12" font-size="12" font-weight="800" fill="#475569">矿质离子</text>
        </g>

        <g transform="translate(274 395)">
          <polygon points="7,0 14,6 11,15 3,15 0,6" fill="#FACC15" stroke="#CA8A04" stroke-width="2"/>
          <text x="24" y="12" font-size="12" font-weight="800" fill="#475569">ATP能量</text>
        </g>

        <text x="415" y="407" data-stage-note font-size="13" font-weight="900" fill="#166534"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var moisture=root.querySelector('[data-moisture]');
    var mineral=root.querySelector('[data-mineral]');
    var oxygen=root.querySelector('[data-oxygen]');
    var hair=root.querySelector('[data-hair]');
    var time=root.querySelector('[data-time]');

    var moistureValue=root.querySelector('[data-moisture-value]');
    var mineralValue=root.querySelector('[data-mineral-value]');
    var oxygenValue=root.querySelector('[data-oxygen-value]');
    var hairValue=root.querySelector('[data-hair-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var waterRateText=root.querySelector('[data-water-rate]');
    var mineralRateText=root.querySelector('[data-mineral-rate]');
    var limitText=root.querySelector('[data-limit]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var soilWater=root.querySelector('[data-soil-water]');
    var soilMinerals=root.querySelector('[data-soil-minerals]');
    var soilOxygenLayer=root.querySelector('[data-soil-oxygen]');
    var rootHairs=root.querySelector('[data-root-hairs]');
    var waterFlow=root.querySelector('[data-water-flow]');
    var mineralFlow=root.querySelector('[data-mineral-flow]');
    var atpLayer=root.querySelector('[data-atp-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');

    var waterBar=root.querySelector('[data-water-bar]');
    var mineralBar=root.querySelector('[data-mineral-bar]');
    var energyBar=root.querySelector('[data-energy-bar]');
    var potentialLabel=root.querySelector('[data-potential-label]');

    var mode='combined';
    var showLabels=${showLabels ? 'true' : 'false'};

    var scenarios={
      normal:{
        moisture:72,
        mineral:45,
        oxygen:78,
        hair:68,
        time:58
      },
      drought:{
        moisture:18,
        mineral:35,
        oxygen:82,
        hair:68,
        time:58
      },
      waterlogged:{
        moisture:94,
        mineral:45,
        oxygen:8,
        hair:68,
        time:58
      },
      deficient:{
        moisture:72,
        mineral:7,
        oxygen:78,
        hair:68,
        time:58
      },
      saline:{
        moisture:60,
        mineral:96,
        oxygen:72,
        hair:68,
        time:58
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

    function factorLabel(value){
      if(value<.22)return '很低';
      if(value<.48)return '偏低';
      if(value<.72)return '中等';
      return '较好';
    }

    function buildSoilWater(count){
      var html='';

      for(var i=0;i<count;i++){
        var x=28+(i%7)*30+(Math.floor(i/7)%2)*9;
        var y=132+Math.floor(i/7)*45+(i%2)*10;

        if(y>365){
          continue;
        }

        html+='<path class="ra-water" d="M'+x+' '+y
          +' C'+(x-7)+' '+(y+11)+' '+x+' '+(y+21)
          +' '+x+' '+(y+21)
          +' C'+x+' '+(y+21)+' '+(x+7)+' '+(y+11)
          +' '+x+' '+y+'Z" fill="#38BDF8" opacity=".78"/>';
      }

      return html;
    }

    function buildSoilMinerals(count){
      var html='';

      for(var i=0;i<count;i++){
        var x=42+(i%6)*34+(Math.floor(i/6)%2)*8;
        var y=158+Math.floor(i/6)*48+(i%2)*8;

        if(y>370){
          continue;
        }

        html+='<circle class="ra-mineral" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%3)
          +'" fill="#8B5CF6" stroke="#5B21B6"'
          +' stroke-width="1.6"/>';

        if(i%4===0){
          html+='<text x="'+(x+7)+'" y="'+(y+4)
            +'" font-size="8" font-weight="900"'
            +' fill="#5B21B6">M</text>';
        }
      }

      return html;
    }

    function buildSoilOxygen(count){
      var html='';

      for(var i=0;i<count;i++){
        var x=34+(i%7)*31;
        var y=352-Math.floor(i/7)*38-(i%2)*8;

        html+='<circle cx="'+x+'" cy="'+y
          +'" r="'+(4+i%3)
          +'" fill="#FFFFFF" stroke="#0284C7"'
          +' stroke-width="1.7" opacity=".86"/>';

        if(i%3===0){
          html+='<text x="'+(x+7)+'" y="'+(y+4)
            +'" font-size="8" font-weight="900"'
            +' fill="#0284C7">O₂</text>';
        }
      }

      return html;
    }

    function buildRootHairs(density){
      var html='';
      var count=Math.max(1,Math.floor(1+density/16));

      for(var i=0;i<count;i++){
        var y=210+i*12;
        var length=48+(i%3)*18+density*.24;
        var endX=246-length;
        var curveY=y+(i%2===0?-15:15);

        html+='<path d="M252 '+y
          +' C220 '+curveY+' '+(endX+24)+' '+curveY
          +' '+endX+' '+(y+(i%2===0?-8:8))
          +'" fill="none" stroke="#166534"'
          +' stroke-width="'+(3+density/55)
          +'" stroke-linecap="round"/>';

        html+='<path d="M249 '+y
          +' C220 '+curveY+' '+(endX+24)+' '+curveY
          +' '+endX+' '+(y+(i%2===0?-8:8))
          +'" fill="none" stroke="#86EFAC"'
          +' stroke-width="1.8" stroke-linecap="round"/>';
      }

      return html;
    }

    function buildWaterFlow(direction,rate,visible){
      if(!visible){
        return '';
      }

      if(direction==='balanced'){
        return ''
          +'<path class="ra-flow" d="M174 192 C218 184 246 198 282 218"'
          +' fill="none" stroke="#0284C7" stroke-width="4"'
          +' marker-end="url(#${rootId}-arrow-blue)" opacity=".62"/>'
          +'<path class="ra-flow" d="M282 242 C246 260 218 266 174 256"'
          +' fill="none" stroke="#0284C7" stroke-width="4"'
          +' marker-end="url(#${rootId}-arrow-blue)" opacity=".62"/>'
          +'<text x="178" y="171" font-size="11" font-weight="900" fill="#0369A1">'
          +'双向运动近似平衡</text>';
      }

      var inward=direction==='in';
      var startX=inward?142:314;
      var startY=inward?188:222;
      var endX=inward?306:140;
      var endY=inward?222:188;
      var color=inward?'#0284C7':'#DC2626';
      var marker=inward
        ?'url(#${rootId}-arrow-blue)'
        :'url(#${rootId}-arrow-red)';
      var thickness=3.5+Math.abs(rate)/24;

      var html=''
        +'<path class="ra-flow" d="M'+startX+' '+startY
        +' C220 180 246 218 '+endX+' '+endY
        +'" fill="none" stroke="'+color+'"'
        +' stroke-width="'+thickness
        +'" marker-end="'+marker+'"/>';

      if(inward){
        html+='<path class="ra-flow" d="M316 228'
          +' C384 218 444 220 548 220"'
          +' fill="none" stroke="#0284C7"'
          +' stroke-width="'+Math.max(3,thickness-1)
          +'" marker-end="url(#${rootId}-arrow-blue)" opacity=".82"/>';
      }

      html+='<text x="152" y="168" font-size="11" font-weight="900" fill="'+color+'">'
        +(inward?'水分净进入根毛细胞':'根细胞可能净失水')
        +'</text>';

      return html;
    }

    function buildMineralFlow(rate,visible){
      if(!visible){
        return '';
      }

      var thickness=3.5+rate/28;

      return ''
        +'<path class="ra-flow" d="M144 286'
        +' C204 296 252 282 304 262"'
        +' fill="none" stroke="#7C3AED"'
        +' stroke-width="'+thickness
        +'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<path class="ra-flow" d="M316 266'
        +' C390 286 452 278 548 260"'
        +' fill="none" stroke="#7C3AED"'
        +' stroke-width="'+Math.max(3,thickness-1)
        +'" marker-end="url(#${rootId}-arrow-purple)" opacity=".84"/>'
        +'<text x="145" y="319" font-size="11" font-weight="900" fill="#6D28D9">'
        +'运输蛋白参与矿质离子跨膜吸收'
        +'</text>';
    }

    function buildATP(level,visible){
      if(!visible){
        return '';
      }

      var html='';
      var count=Math.max(1,Math.floor(level/17));

      for(var i=0;i<count;i++){
        var x=282+(i%4)*31;
        var y=332+Math.floor(i/4)*27;

        html+='<g class="ra-atp" transform="translate('
          +x+' '+y+')">'
          +'<polygon points="0,-9 8,-3 5,7 -5,7 -8,-3"'
          +' fill="#FACC15" stroke="#CA8A04" stroke-width="2"/>'
          +'<text x="0" y="3" text-anchor="middle"'
          +' font-size="7" font-weight="900" fill="#854D0E">ATP</text>'
          +'</g>';
      }

      return html;
    }

    function update(){
      var M=Number(moisture.value);
      var N=Number(mineral.value);
      var O=Number(oxygen.value);
      var H=Number(hair.value);
      var T=Number(time.value);

      moistureValue.textContent=M.toFixed(0)+'%';
      mineralValue.textContent=N.toFixed(0)+'%';
      oxygenValue.textContent=O.toFixed(0)+'%';
      hairValue.textContent=H.toFixed(0)+'%';
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

      labelToggle.classList.toggle('off',!showLabels);
      labelLayer.style.display=showLabels?'':'none';

      var waterAvailability=clamp(
        M/(M+20)*1.16,
        0,
        1
      );

      var mineralAvailability=clamp(
        N/(N+24),
        0,
        1
      );

      var oxygenEnergy=clamp(
        .16+.84*(1-Math.exp(-O/27)),
        0,
        1
      );

      var surfaceFactor=clamp(
        .22+.78*H/100,
        .22,
        1
      );

      var timeFactor=clamp(
        1-Math.exp(-T/28),
        0,
        1
      );

      var salinityPenalty=clamp(
        1-Math.max(0,N-72)/28*.78,
        .12,
        1
      );

      var soilWaterPotential=M-N*.55;
      var rootCellPotential=38;
      var potentialDifference=
        soilWaterPotential-rootCellPotential;

      var waterDirection=Math.abs(potentialDifference)<4
        ?'balanced'
        :potentialDifference>0
          ?'in'
          :'out';

      var directionSign=waterDirection==='in'
        ?1
        :waterDirection==='out'
          ?-1
          :0;

      var waterMagnitude=100
        *waterAvailability
        *surfaceFactor
        *(.42+.58*timeFactor)
        *salinityPenalty;

      if(waterDirection==='out'){
        waterMagnitude=100
          *clamp(Math.abs(potentialDifference)/60,0,1)
          *surfaceFactor
          *(.35+.65*timeFactor);
      }

      if(waterDirection==='balanced'){
        waterMagnitude=0;
      }

      var signedWater=clamp(
        directionSign*waterMagnitude,
        -100,
        100
      );

      var mineralRate=100
        *mineralAvailability
        *surfaceFactor
        *oxygenEnergy
        *(.4+.6*timeFactor);

      mineralRate=clamp(mineralRate,0,100);

      var waterFactors=[
        waterAvailability,
        surfaceFactor,
        salinityPenalty,
        .42+.58*timeFactor
      ];

      var waterFactorNames=[
        '土壤水分',
        '根毛面积',
        '盐胁迫',
        '作用时间'
      ];

      var mineralFactors=[
        mineralAvailability,
        surfaceFactor,
        oxygenEnergy,
        .4+.6*timeFactor
      ];

      var mineralFactorNames=[
        '矿质来源',
        '根毛面积',
        '呼吸供能',
        '作用时间'
      ];

      var limitingFactors=mode==='water'
        ?waterFactors
        :mode==='mineral'
          ?mineralFactors
          :[
            waterAvailability,
            mineralAvailability,
            surfaceFactor,
            oxygenEnergy,
            salinityPenalty,
            .4+.6*timeFactor
          ];

      var limitingNames=mode==='water'
        ?waterFactorNames
        :mode==='mineral'
          ?mineralFactorNames
          :[
            '土壤水分',
            '矿质来源',
            '根毛面积',
            '呼吸供能',
            '盐胁迫',
            '作用时间'
          ];

      var limitIndex=0;

      for(var j=1;j<limitingFactors.length;j++){
        if(limitingFactors[j]<limitingFactors[limitIndex]){
          limitIndex=j;
        }
      }

      var mainLimit=limitingNames[limitIndex];

      waterRateText.textContent=
        signedWater>0
          ?'+'+signedWater.toFixed(0)
          :signedWater.toFixed(0);

      waterRateText.style.color=
        signedWater<0
          ?'#DC2626'
          :signedWater>0
            ?'#0284C7'
            :'#15803D';

      mineralRateText.textContent=mineralRate.toFixed(0);
      limitText.textContent=mainLimit;

      waterBar.setAttribute(
        'width',
        String(76*M/100)
      );

      mineralBar.setAttribute(
        'width',
        String(76*N/100)
      );

      energyBar.setAttribute(
        'width',
        String(76*oxygenEnergy)
      );

      potentialLabel.textContent=
        waterDirection==='in'
          ?'土壤侧水势较高'
          :waterDirection==='out'
            ?'根细胞侧水势较高'
            :'两侧水势接近';

      root.style.setProperty(
        '--ra-water-speed',
        clamp(
          2.5-Math.abs(signedWater)/65,
          .55,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--ra-mineral-speed',
        clamp(
          2.5-mineralRate/68,
          .55,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--ra-flow-speed',
        clamp(
          2.4-Math.max(Math.abs(signedWater),mineralRate)/70,
          .5,
          2.4
        ).toFixed(2)+'s'
      );

      soilWater.innerHTML=buildSoilWater(
        Math.floor(2+M/8)
      );

      soilMinerals.innerHTML=buildSoilMinerals(
        Math.floor(1+N/10)
      );

      soilOxygenLayer.innerHTML=buildSoilOxygen(
        Math.floor(O/13)
      );

      rootHairs.innerHTML=buildRootHairs(H);

      var showWater=mode==='combined' || mode==='water';
      var showMineral=mode==='combined' || mode==='mineral';

      waterFlow.innerHTML=buildWaterFlow(
        waterDirection,
        signedWater,
        showWater
      );

      mineralFlow.innerHTML=buildMineralFlow(
        mineralRate,
        showMineral
      );

      atpLayer.innerHTML=buildATP(
        O,
        showMineral
      );

      var explanation='';
      var conditionNote='';

      if(mode==='water'){
        title.textContent='根毛细胞吸水与水势差';
        summary.textContent='水分的净移动方向取决于土壤溶液与根毛细胞之间的相对水势差';
        stageNote.textContent='蓝色路径表示水分净移动方向';

        if(waterDirection==='in'){
          explanation=
            '当前土壤侧综合水势较高，水分能够由土壤溶液净进入根毛细胞，并继续向根内部移动。';
        }else if(waterDirection==='out'){
          explanation=
            '当前土壤过干或溶质浓度过高，土壤侧综合水势较低，根细胞可能难以吸水甚至发生净失水。';
        }else{
          explanation=
            '当前两侧综合水势接近，水分子仍可双向运动，但净移动接近零。';
        }

        conditionNote=
          '根毛数量增加可扩大接触面积，但不能改变水分净移动必须符合水势差这一基本条件。';
      }else if(mode==='mineral'){
        title.textContent='根细胞对矿质离子的选择性吸收';
        summary.textContent='矿质离子跨膜需要运输蛋白，逆浓度梯度运输还需要细胞呼吸提供能量';
        stageNote.textContent='紫色路径表示矿质离子进入根内';

        explanation=
          '土壤中的矿质离子先与根毛表面接触，再通过具有选择性的膜运输系统进入根细胞。';

        if(N<12){
          conditionNote=
            '当前土壤矿质离子来源不足，即使根毛和能量条件较好，吸收量仍会受到限制。';
        }else if(O<18){
          conditionNote=
            '土壤缺氧限制根细胞呼吸和ATP供应，主动吸收矿质离子的能力明显下降。';
        }else{
          conditionNote=
            '运输蛋白、矿质来源、根毛接触面积和呼吸供能共同影响矿质离子吸收。';
        }
      }else{
        title.textContent='根毛吸收水和无机盐的综合过程';
        summary.textContent='水势差、根毛面积、矿质来源和根细胞呼吸共同决定根表吸收状态';
        stageNote.textContent='水和矿质离子进入根内后将继续向输导组织移动';

        explanation=
          '根毛扩大了根与土壤溶液的接触面积，水分和矿质离子随后通过不同机制进入根细胞。';

        if(M<22){
          conditionNote=
            '土壤水分不足是当前主要问题，根毛与土壤溶液接触减少，净吸水受到明显限制。';
        }else if(N>82){
          conditionNote=
            '土壤溶质浓度过高形成盐胁迫，可能降低土壤水势并抑制根吸水。';
        }else if(O<18){
          conditionNote=
            '水淹环境虽然含水量高，但土壤缺氧会抑制根细胞呼吸和主动吸收矿质离子。';
        }else if(N<12){
          conditionNote=
            '当前矿质离子来源不足，水分条件较好也不能弥补矿质营养缺乏。';
        }else if(H<18){
          conditionNote=
            '根毛较少使吸收表面积减小，水分和矿质离子吸收均受到限制。';
        }else{
          conditionNote=
            '当前水分、矿质来源、根毛面积和呼吸供能总体较适宜。';
        }
      }

      var timeNote=T<15
        ?'作用时间较短，累计吸收效应尚不明显。'
        :'作用时间增加会使当前条件的影响更加明显，但不能把不适宜条件自动变为适宜条件。';

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

        moisture.value=String(data.moisture);
        mineral.value=String(data.mineral);
        oxygen.value=String(data.oxygen);
        hair.value=String(data.hair);
        time.value=String(data.time);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    moisture.oninput=function(){
      setScenarioActive('');
      update();
    };

    mineral.oninput=function(){
      setScenarioActive('');
      update();
    };

    oxygen.oninput=function(){
      setScenarioActive('');
      update();
    };

    hair.oninput=function(){
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
