/**
 * lifeScienceLabTemplatesEcologyDensityDependence.ts
 *
 * 平面生命科学实验室：
 * 环境容纳量、种群密度与密度制约。
 *
 * 教学目标：
 * 1. 理解种群密度是单位空间或单位生境中的个体数量；
 * 2. 理解环境容纳量K是在特定环境条件下，
 *    环境能够相对长期维持的种群数量水平；
 * 3. 理解K值会随食物、空间、水分、栖息地质量、
 *    气候和人类活动变化，不是永久固定常数；
 * 4. 理解种群密度增加时，种内竞争、疾病传播、
 *    寄生和部分捕食压力可能增强；
 * 5. 理解密度制约因素的作用强度通常与种群密度有关；
 * 6. 理解干旱、寒潮、火灾、洪水和极端风暴等因素
 *    可在不同密度下对种群造成明显影响；
 * 7. 区分密度制约因素与非密度制约因素；
 * 8. 观察种群低于K值、接近K值和超过K值后的数量变化；
 * 9. 理解真实种群常在变化的K值附近波动，
 *    不一定形成平滑、稳定和完全重复的S型曲线。
 *
 * 教学边界：
 * 1. 所有种群数量、种群密度、环境容纳量和作用强度
 *    均为相对教学指标；
 * 2. 环境容纳量K不是环境中可以容纳个体的绝对物理上限，
 *    而是特定时间和条件下的相对维持水平；
 * 3. 种群超过K值后不一定立即崩溃，
 *    但资源消耗、竞争和死亡压力通常会增强；
 * 4. 种群低于K值也不代表一定快速增长，
 *    出生率、死亡率、年龄结构、迁入和迁出仍会影响结果；
 * 5. 种内竞争、疾病传播和寄生常具有密度制约特征，
 *    但具体关系可能受接触方式和空间分布影响；
 * 6. 捕食作用是否呈明显密度制约，
 *    取决于捕食者反应、猎物隐蔽和替代食物等条件；
 * 7. 非密度制约因素并非完全不受密度影响，
 *    这里仅表示其发生强度不主要由当前种群密度决定；
 * 8. 极端干扰后K值本身也可能下降，
 *    不应只把干扰理解为一次性减少个体数量；
 * 9. 本模型不用于种群保护、狩猎限额、
 *    渔业配额、害虫控制或真实生态预测。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式、字体、图片或CDN；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用统一.bl-*公共布局协议；
 * 5. 支持同一课件页放置多个独立实例；
 * 6. 不使用document.querySelector或document.querySelectorAll；
 * 7. 本文件只导出独立模板数组，聚合接入由第28批C1完成。
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
 */
function densityDependenceStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#DCFCE7);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:250px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FDFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9}'
    + '#' + rootId + ' .dd-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .dd-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .dd-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .dd-button{min-height:31px;padding:3px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:9.1px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .dd-button.active{border-color:#0EA5E9;background:#E0F2FE;box-shadow:0 3px 9px rgba(14,165,233,.14)}'
    + '#' + rootId + ' .dd-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .dd-toggle.off{background:#64748B}'
    + '#' + rootId + ' .dd-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .dd-card{padding:6px 3px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .dd-card b{display:block;min-height:18px;font-size:13px;color:#0369A1}'
    + '#' + rootId + ' .dd-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .dd-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--dd-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .dd-organism{animation:' + rootId + '-organism var(--dd-organism-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .dd-pulse{animation:' + rootId + '-pulse 1.1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-organism{from{transform:translateY(2px);opacity:.48}to{transform:translateY(-4px);opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.36}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_DENSITY_DEPENDENCE:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-density-dependence-carrying-capacity',
    group: '🌎 生态系统',
    name: '环境容纳量与密度制约',
    emoji: '🐇',
    desc: '调节初始数量、资源、竞争、疾病、捕食、环境干扰和时间，比较密度制约与非密度制约因素',
    params: [
      {
        key: 'initialPopulation',
        label: '初始种群数量',
        type: 'number',
        min: 10,
        max: 180,
        step: 5,
        defaultValue: 70,
      },
      {
        key: 'resourceSupply',
        label: '食物与空间资源',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'competitionStrength',
        label: '种内竞争强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
      {
        key: 'diseaseTransmission',
        label: '疾病传播潜力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
      },
      {
        key: 'predationPressure',
        label: '捕食压力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 35,
      },
      {
        key: 'disturbanceIntensity',
        label: '非密度制约干扰',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 12,
        hint: '综合表示极端天气、火灾或洪水等影响',
      },
      {
        key: 'observationTime',
        label: '观察时间',
        type: 'number',
        min: 5,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'showLabels',
        label: '显示过程标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const initialPopulation = num(
        params,
        'initialPopulation',
        70,
      )
      const resourceSupply = num(
        params,
        'resourceSupply',
        72,
      )
      const competitionStrength = num(
        params,
        'competitionStrength',
        58,
      )
      const diseaseTransmission = num(
        params,
        'diseaseTransmission',
        42,
      )
      const predationPressure = num(
        params,
        'predationPressure',
        35,
      )
      const disturbanceIntensity = num(
        params,
        'disturbanceIntensity',
        12,
      )
      const observationTime = num(
        params,
        'observationTime',
        72,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${densityDependenceStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🐇 环境容纳量、种群密度与密度制约</div>
    <div class="bl-note">K值随环境变化，真实种群常在变化的容纳量附近波动</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>初始种群数量</span>
          <span class="bl-value" data-initial-value></span>
        </div>
        <input data-initial type="range" min="10" max="180" step="5" value="${n(initialPopulation)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>食物与空间资源</span>
          <span class="bl-value" data-resource-value></span>
        </div>
        <input data-resource type="range" min="10" max="100" step="1" value="${n(resourceSupply)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>种内竞争强度</span>
          <span class="bl-value" data-competition-value></span>
        </div>
        <input data-competition type="range" min="0" max="100" step="1" value="${n(competitionStrength)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>疾病传播潜力</span>
          <span class="bl-value" data-disease-value></span>
        </div>
        <input data-disease type="range" min="0" max="100" step="1" value="${n(diseaseTransmission)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>捕食压力</span>
          <span class="bl-value" data-predation-value></span>
        </div>
        <input data-predation type="range" min="0" max="100" step="1" value="${n(predationPressure)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>非密度制约干扰</span>
          <span class="bl-value" data-disturbance-value></span>
        </div>
        <input data-disturbance type="range" min="0" max="100" step="1" value="${n(disturbanceIntensity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>观察时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="5" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="dd-subtitle">观察模式</div>

      <div class="dd-buttons">
        <button type="button" class="dd-button active" data-mode="dependent">密度制约</button>
        <button type="button" class="dd-button" data-mode="independent">非密度制约</button>
        <button type="button" class="dd-button" data-mode="compare">综合比较</button>
      </div>

      <div class="dd-subtitle">快速比较情境</div>

      <div class="dd-scenarios">
        <button type="button" class="dd-button active" data-scenario="sparse">低密度</button>
        <button type="button" class="dd-button" data-scenario="nearK">接近K值</button>
        <button type="button" class="dd-button" data-scenario="crowded">超过K值</button>
        <button type="button" class="dd-button" data-scenario="outbreak">高密度传播</button>
        <button type="button" class="dd-button" data-scenario="storm">极端干扰</button>
      </div>

      <button type="button" class="dd-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '过程标注：显示' : '过程标注：隐藏'}
      </button>

      <div class="dd-status">
        <div class="dd-card">
          <b data-k-value></b>
          <span>当前环境容纳量K</span>
        </div>

        <div class="dd-card">
          <b data-final-population></b>
          <span>末期种群数量</span>
        </div>

        <div class="dd-card">
          <b data-density-state></b>
          <span>密度状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="环境容纳量与密度制约互动模型"
      >
        <defs>
          <linearGradient id="${rootId}-habitat" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#ECFDF5"/>
          </linearGradient>

          <linearGradient id="${rootId}-population-area" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#0EA5E9" stop-opacity=".38"/>
            <stop offset="100%" stop-color="#0EA5E9" stop-opacity=".04"/>
          </linearGradient>

          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#075985" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text x="22" y="34" data-title font-size="25" font-weight="900" fill="#075985"></text>
        <text x="22" y="62" data-summary font-size="13" font-weight="800" fill="#475569"></text>

        <!-- 左上：种群密度场景 -->
        <g filter="url(#${rootId}-shadow)">
          <rect x="22" y="79" width="343" height="166" rx="20" fill="url(#${rootId}-habitat)" stroke="#7DD3FC" stroke-width="3"/>
        </g>

        <g data-habitat-layer></g>
        <g data-organism-layer></g>
        <g data-pressure-layer></g>

        <!-- 右上：制约因素强度 -->
        <g transform="translate(385 79)">
          <rect width="351" height="166" rx="20" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>

          <text x="175" y="25" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">种群限制因素</text>

          <text x="14" y="53" font-size="10.5" font-weight="800" fill="#64748B">种内竞争</text>
          <rect x="89" y="44" width="238" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-competition-bar x="89" y="44" width="0" height="12" rx="6" fill="#F59E0B"/>

          <text x="14" y="82" font-size="10.5" font-weight="800" fill="#64748B">疾病传播</text>
          <rect x="89" y="73" width="238" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-disease-bar x="89" y="73" width="0" height="12" rx="6" fill="#DC2626"/>

          <text x="14" y="111" font-size="10.5" font-weight="800" fill="#64748B">捕食影响</text>
          <rect x="89" y="102" width="238" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-predation-bar x="89" y="102" width="0" height="12" rx="6" fill="#7C3AED"/>

          <text x="14" y="140" font-size="10.5" font-weight="800" fill="#64748B">环境干扰</text>
          <rect x="89" y="131" width="238" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-disturbance-bar x="89" y="131" width="0" height="12" rx="6" fill="#64748B"/>

          <text x="175" y="158" data-panel-note text-anchor="middle" font-size="10.5" font-weight="900" fill="#075985"></text>
        </g>

        <!-- 下方曲线 -->
        <text x="22" y="275" data-chart-title font-size="13" font-weight="900" fill="#334155"></text>

        <line x1="62" y1="389" x2="704" y2="389" stroke="#64748B" stroke-width="2.5"/>
        <line x1="62" y1="389" x2="62" y2="291" stroke="#64748B" stroke-width="2.5"/>

        <text x="708" y="393" font-size="10.5" font-weight="800" fill="#64748B">时间</text>
        <text x="17" y="300" font-size="10.5" font-weight="800" fill="#64748B">数量</text>

        <g data-grid></g>

        <line
          data-k-line
          x1="62"
          y1="330"
          x2="704"
          y2="330"
          stroke="#F59E0B"
          stroke-width="3"
          stroke-dasharray="9 7"
        ></line>

        <text
          data-k-label
          x="575"
          y="320"
          font-size="10.5"
          font-weight="900"
          fill="#B45309"
        ></text>

        <path
          data-population-area
          fill="url(#${rootId}-population-area)"
        ></path>

        <path
          data-population-curve
          fill="none"
          stroke="#0284C7"
          stroke-width="5"
          stroke-linecap="round"
          stroke-linejoin="round"
        ></path>

        <g data-points></g>

        <g data-label-layer>
          <g transform="translate(82 416)">
            <circle cx="7" cy="0" r="6" fill="#0284C7"/>
            <text x="20" y="5" font-size="11" font-weight="900" fill="#475569">种群数量</text>
          </g>

          <g transform="translate(228 416)">
            <circle cx="7" cy="0" r="6" fill="#F59E0B"/>
            <text x="20" y="5" font-size="11" font-weight="900" fill="#475569">环境容纳量K</text>
          </g>

          <text x="452" y="420" data-stage-note font-size="11" font-weight="900" fill="#075985"></text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var initial=root.querySelector('[data-initial]');
    var resource=root.querySelector('[data-resource]');
    var competition=root.querySelector('[data-competition]');
    var disease=root.querySelector('[data-disease]');
    var predation=root.querySelector('[data-predation]');
    var disturbance=root.querySelector('[data-disturbance]');
    var time=root.querySelector('[data-time]');

    var initialValue=root.querySelector('[data-initial-value]');
    var resourceValue=root.querySelector('[data-resource-value]');
    var competitionValue=root.querySelector('[data-competition-value]');
    var diseaseValue=root.querySelector('[data-disease-value]');
    var predationValue=root.querySelector('[data-predation-value]');
    var disturbanceValue=root.querySelector('[data-disturbance-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var kValueText=root.querySelector('[data-k-value]');
    var finalPopulationText=root.querySelector('[data-final-population]');
    var densityStateText=root.querySelector('[data-density-state]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var habitatLayer=root.querySelector('[data-habitat-layer]');
    var organismLayer=root.querySelector('[data-organism-layer]');
    var pressureLayer=root.querySelector('[data-pressure-layer]');

    var competitionBar=root.querySelector('[data-competition-bar]');
    var diseaseBar=root.querySelector('[data-disease-bar]');
    var predationBar=root.querySelector('[data-predation-bar]');
    var disturbanceBar=root.querySelector('[data-disturbance-bar]');
    var panelNote=root.querySelector('[data-panel-note]');

    var chartTitle=root.querySelector('[data-chart-title]');
    var grid=root.querySelector('[data-grid]');
    var kLine=root.querySelector('[data-k-line]');
    var kLabel=root.querySelector('[data-k-label]');
    var populationArea=root.querySelector('[data-population-area]');
    var populationCurve=root.querySelector('[data-population-curve]');
    var points=root.querySelector('[data-points]');

    var labelLayer=root.querySelector('[data-label-layer]');
    var stageNote=root.querySelector('[data-stage-note]');

    var mode='dependent';
    var showLabels=${showLabels ? 'true' : 'false'};
    var steps=36;

    var scenarios={
      sparse:{
        mode:'dependent',
        initial:35,
        resource:82,
        competition:48,
        disease:28,
        predation:22,
        disturbance:5,
        time:72
      },
      nearK:{
        mode:'dependent',
        initial:120,
        resource:62,
        competition:64,
        disease:48,
        predation:35,
        disturbance:8,
        time:72
      },
      crowded:{
        mode:'dependent',
        initial:180,
        resource:42,
        competition:94,
        disease:72,
        predation:52,
        disturbance:8,
        time:78
      },
      outbreak:{
        mode:'dependent',
        initial:155,
        resource:68,
        competition:68,
        disease:98,
        predation:28,
        disturbance:5,
        time:82
      },
      storm:{
        mode:'independent',
        initial:110,
        resource:74,
        competition:52,
        disease:32,
        predation:28,
        disturbance:96,
        time:70
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

    function setModeActive(){
      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }
    }

    /**
     * 根据资源和干扰计算动态环境容纳量。
     *
     * 极端干扰不仅会减少个体，
     * 也可能暂时降低栖息地质量和K值。
     */
    function calculateCarryingCapacity(
      resourceLevel,
      disturbanceLevel,
      progress
    ){
      var baseCapacity=
        70+resourceLevel*3.4;

      var disturbanceDamage=
        disturbanceLevel/100
        *(.35+.65*progress);

      return clamp(
        baseCapacity
        *(1-.52*disturbanceDamage),
        38,
        420
      );
    }

    /**
     * 模拟密度制约和非密度制约共同作用。
     *
     * 密度制约：
     * - 种内竞争；
     * - 疾病传播；
     * - 部分捕食影响。
     *
     * 非密度制约：
     * - 极端天气、火灾或洪水等环境干扰。
     */
    function simulatePopulation(
      initialPopulation,
      resourceLevel,
      competitionLevel,
      diseaseLevel,
      predationLevel,
      disturbanceLevel,
      observationMode
    ){
      var population=
        initialPopulation;

      var records=[{
        population:population,
        capacity:
          calculateCarryingCapacity(
            resourceLevel,
            disturbanceLevel,
            0
          ),
        competitionPressure:0,
        diseasePressure:0,
        predationPressure:0,
        disturbancePressure:0
      }];

      for(var i=1;i<=steps;i++){
        var progress=
          i/steps;

        var capacity=
          calculateCarryingCapacity(
            resourceLevel,
            disturbanceLevel,
            progress
          );

        var densityRatio=
          population/Math.max(
            1,
            capacity
          );

        var intrinsicGrowth=
          .15+.055*resourceLevel/100;

        var logisticGrowth=
          intrinsicGrowth
          *population
          *(1-densityRatio);

        var competitionPressure=
          competitionLevel/100
          *Math.pow(
            Math.max(0,densityRatio),
            1.45
          )
          *.085
          *population;

        var diseasePressure=
          diseaseLevel/100
          *Math.pow(
            Math.max(0,densityRatio),
            1.8
          )
          *.07
          *population;

        var predationDensityResponse=
          .28
          +.72*clamp(
            densityRatio,
            0,
            1.4
          );

        var predationPressure=
          predationLevel/100
          *predationDensityResponse
          *.045
          *population;

        var disturbancePulse=
          disturbanceLevel/100
          *(
            progress>.36
            &&progress<.52
              ?1
              :.16
          );

        var disturbancePressure=
          disturbancePulse
          *.15
          *population;

        var densityLoss=
          competitionPressure
          +diseasePressure
          +predationPressure;

        var selectedDensityLoss=
          observationMode==='independent'
            ?densityLoss*.18
            :densityLoss;

        var selectedDisturbanceLoss=
          observationMode==='dependent'
            ?disturbancePressure*.18
            :disturbancePressure;

        if(observationMode==='compare'){
          selectedDensityLoss=
            densityLoss;

          selectedDisturbanceLoss=
            disturbancePressure;
        }

        var nextPopulation=
          population
          +logisticGrowth
          -selectedDensityLoss
          -selectedDisturbanceLoss;

        population=clamp(
          nextPopulation,
          0,
          capacity*1.55
        );

        records.push({
          population:population,
          capacity:capacity,
          competitionPressure:
            competitionPressure,
          diseasePressure:
            diseasePressure,
          predationPressure:
            predationPressure,
          disturbancePressure:
            disturbancePressure
        });
      }

      return records;
    }

    function visibleIndex(timeLevel){
      return clamp(
        Math.round(
          3+steps*timeLevel/100
        ),
        3,
        steps
      );
    }

    function buildGrid(
      maxValue,
      recordCount
    ){
      var html='';
      var left=62;
      var right=704;
      var top=291;
      var bottom=389;
      var width=right-left;
      var height=bottom-top;

      for(var yIndex=0;yIndex<=4;yIndex++){
        var y=
          bottom-height*yIndex/4;

        var value=
          maxValue*yIndex/4;

        html+='<line x1="'+left+'" y1="'+y
          +'" x2="'+right+'" y2="'+y
          +'" stroke="#E2E8F0" stroke-width="1.2"/>';

        html+='<text x="'+(left-8)+'" y="'+(y+4)
          +'" text-anchor="end" font-size="9.5"'
          +' font-weight="700" fill="#64748B">'
          +value.toFixed(0)
          +'</text>';
      }

      var ticks=Math.min(
        6,
        Math.max(
          1,
          recordCount-1
        )
      );

      for(var xIndex=0;xIndex<=ticks;xIndex++){
        var x=
          left+width*xIndex/ticks;

        var index=
          Math.round(
            (recordCount-1)
            *xIndex/ticks
          );

        html+='<line x1="'+x+'" y1="'+bottom
          +'" x2="'+x+'" y2="'+top
          +'" stroke="#F1F5F9" stroke-width="1"/>';

        html+='<text x="'+x+'" y="'+(bottom+16)
          +'" text-anchor="middle" font-size="9.5"'
          +' font-weight="700" fill="#64748B">'
          +index
          +'</text>';
      }

      return html;
    }

    function buildCurve(
      records,
      maxValue
    ){
      var left=62;
      var right=704;
      var top=291;
      var bottom=389;
      var width=right-left;
      var height=bottom-top;
      var path='';
      var pointHTML='';

      function x(index){
        return records.length<=1
          ?left
          :left
            +width*index/(records.length-1);
      }

      function y(value){
        return bottom
          -height*clamp(
            value/maxValue,
            0,
            1
          );
      }

      for(var i=0;i<records.length;i++){
        var px=x(i);
        var py=y(
          records[i].population
        );

        path+=(i===0?'M':' L')
          +px+' '+py;

        if(
          i===0
          ||i===records.length-1
          ||i%Math.max(
            1,
            Math.floor(records.length/6)
          )===0
        ){
          pointHTML+='<circle cx="'+px
            +'" cy="'+py+'" r="3.7"'
            +' fill="#FFFFFF" stroke="#0284C7"'
            +' stroke-width="2.3"/>';
        }
      }

      return {
        path:path,
        area:path
          +' L'+right+' '+bottom
          +' L'+left+' '+bottom
          +' Z',
        points:pointHTML,
        y:y
      };
    }

    function buildHabitat(
      resourceLevel,
      disturbanceLevel
    ){
      var plantCount=clamp(
        Math.floor(
          2+resourceLevel/14
        ),
        2,
        9
      );

      var html='';

      html+='<rect x="37" y="202" width="313" height="29"'
        +' rx="14" fill="#D6B47C" stroke="#92400E"'
        +' stroke-width="3"/>';

      for(var i=0;i<plantCount;i++){
        var x=52+(i%8)*38;
        var y=202-(i%3)*8;
        var height=
          23+(i%4)*8;

        html+='<path d="M'+x+' '+y
          +' Q'+(x-4)+' '+(y-height*.55)
          +' '+x+' '+(y-height)
          +'" fill="none" stroke="#16A34A"'
          +' stroke-width="4" stroke-linecap="round"/>';

        html+='<ellipse cx="'+(x-6)
          +'" cy="'+(y-height*.55)
          +'" rx="8" ry="4" fill="#4ADE80"'
          +' stroke="#15803D" stroke-width="1.5"'
          +' transform="rotate(-28 '+(x-6)
          +' '+(y-height*.55)+')"/>';
      }

      if(disturbanceLevel>45){
        html+='<path class="dd-flow" d="M40 105'
          +' C115 79 188 119 257 91'
          +' C291 78 321 83 347 104"'
          +' fill="none" stroke="#64748B"'
          +' stroke-width="'+(3+disturbanceLevel/20)
          +'" marker-end="url(#${rootId}-arrow-red)"'
          +' opacity="'+(.35+.6*disturbanceLevel/100)+'"/>';

        html+='<text x="194" y="82" text-anchor="middle"'
          +' font-size="10.5" font-weight="900" fill="#475569">'
          +'极端天气或环境干扰'
          +'</text>';
      }

      return html;
    }

    function buildOrganisms(
      population,
      capacity
    ){
      var densityRatio=
        population/Math.max(
          1,
          capacity
        );

      var count=clamp(
        Math.floor(
          3+population/24
        ),
        3,
        18
      );

      var html='';

      for(var i=0;i<count;i++){
        var columns=
          densityRatio>1.05
            ?7
            :6;

        var x=
          57+(i%columns)*42;

        var y=
          127
          +Math.floor(i/columns)*35
          +(i%2)*5;

        var scale=
          densityRatio>1.15
            ?.78
            :.88;

        html+='<g class="dd-organism"'
          +' transform="translate('+x+' '+y+')'
          +' scale('+scale+')">'
          +'<ellipse cx="0" cy="0" rx="15" ry="10"'
          +' fill="#FDE68A" stroke="#D97706"'
          +' stroke-width="2.5"/>'
          +'<circle cx="13" cy="-3" r="7"'
          +' fill="#FACC15" stroke="#D97706"'
          +' stroke-width="2"/>'
          +'<ellipse cx="16" cy="-13" rx="3" ry="8"'
          +' fill="#FDE68A" stroke="#D97706"'
          +' stroke-width="1.5"/>'
          +'<circle cx="16" cy="-5" r="1.5" fill="#111827"/>'
          +'</g>';
      }

      return html;
    }

    function buildPressures(
      competitionPressure,
      diseasePressure,
      predationPressure,
      disturbancePressure,
      visible
    ){
      if(!visible){
        return '';
      }

      var html='';

      if(competitionPressure>2){
        html+='<path class="dd-flow" d="M66 194'
          +' C111 169 158 171 197 192"'
          +' fill="none" stroke="#F59E0B"'
          +' stroke-width="'+(2+competitionPressure/12)
          +'" marker-end="url(#${rootId}-arrow-orange)"/>';

        html+='<text x="132" y="164" text-anchor="middle"'
          +' font-size="10" font-weight="900" fill="#B45309">'
          +'资源竞争'
          +'</text>';
      }

      if(diseasePressure>2){
        var diseaseCount=clamp(
          Math.floor(
            1+diseasePressure/8
          ),
          1,
          8
        );

        for(var i=0;i<diseaseCount;i++){
          var x=85+(i%6)*37;
          var y=113+Math.floor(i/6)*28+(i%2)*9;

          html+='<circle class="dd-pulse" cx="'+x
            +'" cy="'+y+'" r="'+(4+i%2)
            +'" fill="#FCA5A5" stroke="#B91C1C"'
            +' stroke-width="1.5"/>';
        }
      }

      if(predationPressure>2){
        html+='<path class="dd-flow" d="M322 119'
          +' C291 112 269 123 247 143"'
          +' fill="none" stroke="#7C3AED"'
          +' stroke-width="'+(2+predationPressure/12)
          +'" marker-end="url(#${rootId}-arrow-blue)"/>';

        html+='<text x="296" y="105" text-anchor="middle"'
          +' font-size="10" font-weight="900" fill="#6D28D9">'
          +'捕食影响'
          +'</text>';
      }

      if(disturbancePressure>2){
        html+='<path d="M44 220 C120 204 211 234 343 210"'
          +' fill="none" stroke="#DC2626"'
          +' stroke-width="'+(2+disturbancePressure/8)
          +'" opacity=".65"/>';
      }

      return html;
    }

    function classifyDensity(
      population,
      capacity
    ){
      var ratio=
        population/Math.max(
          1,
          capacity
        );

      if(ratio<.35){
        return '低密度';
      }

      if(ratio<.8){
        return '增长区间';
      }

      if(ratio<=1.08){
        return '接近K值';
      }

      return '超过K值';
    }

    function update(){
      var I=Number(initial.value);
      var R=Number(resource.value);
      var C=Number(competition.value);
      var D=Number(disease.value);
      var P=Number(predation.value);
      var E=Number(disturbance.value);
      var T=Number(time.value);

      initialValue.textContent=
        I.toFixed(0);

      resourceValue.textContent=
        R.toFixed(0)+'%';

      competitionValue.textContent=
        C.toFixed(0)+'%';

      diseaseValue.textContent=
        D.toFixed(0)+'%';

      predationValue.textContent=
        P.toFixed(0)+'%';

      disturbanceValue.textContent=
        E.toFixed(0)+'%';

      timeValue.textContent=
        T.toFixed(0)+'%';

      setModeActive();

      labelToggle.textContent=showLabels
        ?'过程标注：显示'
        :'过程标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      var allRecords=
        simulatePopulation(
          I,
          R,
          C,
          D,
          P,
          E,
          mode
        );

      var endIndex=
        visibleIndex(T);

      var records=
        allRecords.slice(
          0,
          endIndex+1
        );

      var finalRecord=
        records[
          records.length-1
        ];

      var maxValue=100;

      for(var i=0;i<records.length;i++){
        maxValue=Math.max(
          maxValue,
          records[i].population,
          records[i].capacity
        );
      }

      maxValue*=1.12;

      grid.innerHTML=
        buildGrid(
          maxValue,
          records.length
        );

      var curve=
        buildCurve(
          records,
          maxValue
        );

      populationCurve.setAttribute(
        'd',
        curve.path
      );

      populationArea.setAttribute(
        'd',
        curve.area
      );

      points.innerHTML=
        curve.points;

      var kY=
        curve.y(
          finalRecord.capacity
        );

      kLine.setAttribute(
        'y1',
        String(kY)
      );

      kLine.setAttribute(
        'y2',
        String(kY)
      );

      kLabel.setAttribute(
        'y',
        String(
          Math.max(
            302,
            kY-8
          )
        )
      );

      kLabel.textContent=
        '当前K值 '
        +finalRecord.capacity.toFixed(0);

      var densityRatio=
        finalRecord.population
        /Math.max(
          1,
          finalRecord.capacity
        );

      var effectiveCompetition=
        clamp(
          finalRecord.competitionPressure
          /Math.max(
            1,
            finalRecord.population
          )
          *1000,
          0,
          100
        );

      var effectiveDisease=
        clamp(
          finalRecord.diseasePressure
          /Math.max(
            1,
            finalRecord.population
          )
          *1200,
          0,
          100
        );

      var effectivePredation=
        clamp(
          finalRecord.predationPressure
          /Math.max(
            1,
            finalRecord.population
          )
          *1500,
          0,
          100
        );

      var effectiveDisturbance=
        clamp(
          E,
          0,
          100
        );

      competitionBar.setAttribute(
        'width',
        String(
          238*effectiveCompetition/100
        )
      );

      diseaseBar.setAttribute(
        'width',
        String(
          238*effectiveDisease/100
        )
      );

      predationBar.setAttribute(
        'width',
        String(
          238*effectivePredation/100
        )
      );

      disturbanceBar.setAttribute(
        'width',
        String(
          238*effectiveDisturbance/100
        )
      );

      var densityState=
        classifyDensity(
          finalRecord.population,
          finalRecord.capacity
        );

      kValueText.textContent=
        finalRecord.capacity.toFixed(0);

      finalPopulationText.textContent=
        finalRecord.population.toFixed(0);

      densityStateText.textContent=
        densityState;

      densityStateText.style.color=
        densityState==='超过K值'
          ?'#DC2626'
          :densityState==='接近K值'
            ?'#B45309'
            :'#0369A1';

      habitatLayer.innerHTML=
        buildHabitat(
          R,
          E
        );

      organismLayer.innerHTML=
        buildOrganisms(
          finalRecord.population,
          finalRecord.capacity
        );

      pressureLayer.innerHTML=
        buildPressures(
          finalRecord.competitionPressure,
          finalRecord.diseasePressure,
          finalRecord.predationPressure,
          finalRecord.disturbancePressure,
          showLabels
        );

      root.style.setProperty(
        '--dd-flow-speed',
        clamp(
          2.5-Math.max(
            effectiveCompetition,
            effectiveDisease,
            effectivePredation,
            effectiveDisturbance
          )/64,
          .52,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--dd-organism-speed',
        clamp(
          2.6-finalRecord.population/180,
          .58,
          2.6
        ).toFixed(2)+'s'
      );

      var mainPressureValues=[
        effectiveCompetition,
        effectiveDisease,
        effectivePredation,
        effectiveDisturbance
      ];

      var mainPressureNames=[
        '种内竞争',
        '疾病传播',
        '捕食影响',
        '环境干扰'
      ];

      var mainIndex=0;

      for(var j=1;j<mainPressureValues.length;j++){
        if(mainPressureValues[j]>mainPressureValues[mainIndex]){
          mainIndex=j;
        }
      }

      panelNote.textContent=
        '主要限制：'
        +mainPressureNames[mainIndex];

      var explanation='';
      var conditionNote='';

      if(mode==='dependent'){
        title.textContent=
          '密度制约：种群越密集，部分限制作用越强';

        summary.textContent=
          '种内竞争、疾病传播和部分捕食压力会随种群密度变化';

        chartTitle.textContent=
          '密度制约条件下的种群数量与动态K值';

        stageNote.textContent=
          '密度升高后，单位时间内的竞争、接触和传播机会增加';

        explanation=
          '密度制约因素的作用强度通常与种群密度有关。个体越密集，食物和空间竞争、个体接触以及疾病传播机会往往越多。';

        if(densityRatio>1.08){
          conditionNote=
            '当前种群超过环境容纳量，资源竞争和死亡压力增强，数量通常难以长期维持在这一水平。';
        }else if(densityRatio>.8){
          conditionNote=
            '当前种群接近K值，环境阻力增强，净增长速度逐渐下降。';
        }else if(D>82 && densityRatio>.65){
          conditionNote=
            '疾病传播潜力较高且种群较密集，接触传播造成的损失明显增强。';
        }else if(C>82 && densityRatio>.65){
          conditionNote=
            '种内竞争较强，资源和空间限制成为当前主要密度制约因素。';
        }else if(densityRatio<.35){
          conditionNote=
            '当前种群密度较低，竞争和传播压力相对较小，但种群是否增长还取决于出生率和其他因素。';
        }else{
          conditionNote=
            '当前种群仍低于K值，但密度上升已经使部分环境阻力逐步增强。';
        }
      }else if(mode==='independent'){
        title.textContent=
          '非密度制约：环境干扰不主要由当前密度决定';

        summary.textContent=
          '极端天气、火灾或洪水可在不同种群密度下造成明显影响';

        chartTitle.textContent=
          '非密度制约干扰后的种群数量与K值变化';

        stageNote.textContent=
          '干扰不仅减少个体，也可能降低栖息地质量和环境容纳量';

        explanation=
          '非密度制约因素的发生强度不主要由当前种群密度决定，例如极端天气、火灾或洪水可以同时影响低密度和高密度种群。';

        if(E>82){
          conditionNote=
            '当前环境干扰很强，种群数量明显下降，同时栖息地受损使K值也降低。';
        }else if(E>45){
          conditionNote=
            '当前干扰造成一定个体损失，并使环境能够维持的种群数量下降。';
        }else if(E<15){
          conditionNote=
            '当前非密度制约干扰较弱，种群变化主要仍由资源和自身增长过程决定。';
        }else{
          conditionNote=
            '当前存在中等环境干扰，但其影响并不是因为种群密度升高才发生。';
        }
      }else{
        title.textContent=
          '密度制约与非密度制约共同影响种群';

        summary.textContent=
          '种群内部的密度效应与外部环境干扰可以同时发生并相互叠加';

        chartTitle.textContent=
          '多类限制因素共同作用下的种群变化';

        stageNote.textContent=
          '真实种群通常同时受到密度、气候、迁移和种间关系影响';

        explanation=
          '真实种群很少只受到单一因素控制。种内竞争、疾病传播和捕食影响可以与极端天气、火灾或洪水同时发生。';

        if(E>75 && densityRatio>1){
          conditionNote=
            '当前种群超过K值，同时又遭遇强环境干扰，个体损失和栖息地退化共同加重种群下降。';
        }else if(
          effectiveDisease>65
          &&effectiveCompetition>65
        ){
          conditionNote=
            '当前高密度同时增强竞争与疾病传播，两类密度制约压力共同限制种群。';
        }else if(E>65){
          conditionNote=
            '当前环境干扰是主要限制因素，但干扰后的低K值又会进一步提高密度压力。';
        }else if(densityRatio>.8){
          conditionNote=
            '当前种群接近环境容纳量，密度制约因素成为主要限制。';
        }else{
          conditionNote=
            '当前多类限制因素均存在，但没有单一因素完全决定种群变化。';
        }
      }

      var kNote=
        ' 当前K值为'
        +finalRecord.capacity.toFixed(0)
        +'，它由资源和环境干扰共同决定，并不是永久不变的常数。';

      var timeNote=T<20
        ?'观察时间较短，当前变化不能代表长期结果。'
        :'观察时间增加可显示更多累计变化，但真实种群通常会在变化的K值附近波动。';

      result.innerHTML=
        explanation
        +'<br>'+conditionNote
        +kNote
        +' '+timeNote
        +' 所有数值均为相对教学指标。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        setScenarioActive('');
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

        mode=data.mode;
        initial.value=String(data.initial);
        resource.value=String(data.resource);
        competition.value=String(data.competition);
        disease.value=String(data.disease);
        predation.value=String(data.predation);
        disturbance.value=String(data.disturbance);
        time.value=String(data.time);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    initial.oninput=function(){
      setScenarioActive('');
      update();
    };

    resource.oninput=function(){
      setScenarioActive('');
      update();
    };

    competition.oninput=function(){
      setScenarioActive('');
      update();
    };

    disease.oninput=function(){
      setScenarioActive('');
      update();
    };

    predation.oninput=function(){
      setScenarioActive('');
      update();
    };

    disturbance.oninput=function(){
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
