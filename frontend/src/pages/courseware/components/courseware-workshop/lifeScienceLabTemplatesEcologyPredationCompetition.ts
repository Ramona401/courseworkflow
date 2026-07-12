/**
 * lifeScienceLabTemplatesEcologyPredationCompetition.ts
 *
 * 平面生命科学实验室：捕食与竞争的种群关系。
 *
 * 教学目标：
 * 1. 理解捕食关系会同时影响猎物种群和捕食者种群；
 * 2. 理解猎物增加可为捕食者提供更多食物，
 *    捕食者增加又会提高猎物受到的捕食压力；
 * 3. 观察部分捕食—被捕食系统中可能出现的种群波动；
 * 4. 理解捕食者种群数量高峰常可能滞后于猎物种群数量高峰；
 * 5. 理解竞争发生在两个种群共同利用有限资源时；
 * 6. 观察资源供应和生态位重叠程度对竞争结果的影响；
 * 7. 理解生态位分化可以减弱种间竞争并提高共存可能性；
 * 8. 区分捕食、种间竞争与食物链能量传递的不同含义；
 * 9. 理解真实种群还会受到气候、疾病、迁移、
 *    人类活动和其他种间关系共同影响。
 *
 * 教学边界：
 * 1. 本模型使用简化的捕食—被捕食模型和种间竞争模型；
 * 2. 所有种群数量、捕食强度、资源水平和竞争指数
 *    均为相对教学指标；
 * 3. 捕食者和猎物并不一定形成规则、永久和完全重复的周期；
 * 4. 捕食者数量高峰滞后于猎物高峰是部分系统中的常见现象，
 *    不是所有真实捕食系统都必须出现的固定规律；
 * 5. 捕食并不等同于把猎物种群全部消灭，
 *    捕食者同样受到猎物数量和其他资源条件限制；
 * 6. 竞争可能降低双方增长率，也可能在生态位分化后形成共存；
 * 7. 竞争排斥并不是由一次短期观察即可确认，
 *    真实结局取决于时间尺度、空间异质性和环境变化；
 * 8. 生态位重叠高不等于两个物种所有需求完全相同；
 * 9. 图中“猎物向捕食者提供食物”和
 *    “捕食者对猎物形成捕食压力”是种群作用方向，
 *    不等同于食物链中物质和能量流向的单一箭头；
 * 10. 本模型不用于野生动物管理、渔业预测或害虫防治决策。
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
function predationCompetitionStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BBF7D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#FEF3C7);border-bottom:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:250px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FAFFFB;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#22C55E}'
    + '#' + rootId + ' .pc-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .pc-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .pc-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .pc-button{min-height:31px;padding:3px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:9.2px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .pc-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.14)}'
    + '#' + rootId + ' .pc-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#4ADE80,#16A34A);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pc-toggle.off{background:#64748B}'
    + '#' + rootId + ' .pc-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .pc-card{padding:6px 3px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .pc-card b{display:block;min-height:18px;font-size:13px;color:#15803D}'
    + '#' + rootId + ' .pc-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .pc-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--pc-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .pc-prey{animation:' + rootId + '-prey var(--pc-prey-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .pc-predator{animation:' + rootId + '-predator var(--pc-predator-speed,1.9s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .pc-pulse{animation:' + rootId + '-pulse 1.1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-prey{from{transform:translateY(2px);opacity:.52}to{transform:translateY(-4px);opacity:1}}'
    + '@keyframes ' + rootId + '-predator{from{transform:translateX(-3px);opacity:.5}to{transform:translateX(4px);opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.38}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_PREDATION_COMPETITION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-predation-competition-dynamics',
    group: '🌎 生态系统',
    name: '捕食与竞争的种群关系',
    emoji: '🐺',
    desc: '调节资源、两个种群初始数量、捕食强度、生态位重叠和观察时间，比较捕食波动与种间竞争',
    params: [
      {
        key: 'resourceSupply',
        label: '环境资源供应',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'populationAInitial',
        label: '种群甲初始数量',
        type: 'number',
        min: 5,
        max: 100,
        step: 1,
        defaultValue: 64,
        hint: '捕食模式中表示猎物种群',
      },
      {
        key: 'populationBInitial',
        label: '种群乙初始数量',
        type: 'number',
        min: 5,
        max: 100,
        step: 1,
        defaultValue: 34,
        hint: '捕食模式中表示捕食者种群',
      },
      {
        key: 'predationStrength',
        label: '捕食作用强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
      {
        key: 'nicheOverlap',
        label: '生态位重叠程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 52,
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
        label: '显示关系标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const resourceSupply = num(
        params,
        'resourceSupply',
        76,
      )
      const populationAInitial = num(
        params,
        'populationAInitial',
        64,
      )
      const populationBInitial = num(
        params,
        'populationBInitial',
        34,
      )
      const predationStrength = num(
        params,
        'predationStrength',
        58,
      )
      const nicheOverlap = num(
        params,
        'nicheOverlap',
        52,
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
${predationCompetitionStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🐺 捕食与竞争的种群关系</div>
    <div class="bl-note">种群相互作用会改变双方增长，但真实生态系统还受多种因素共同影响</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>环境资源供应</span>
          <span class="bl-value" data-resource-value></span>
        </div>
        <input data-resource type="range" min="10" max="100" step="1" value="${n(resourceSupply)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>种群甲初始数量</span>
          <span class="bl-value" data-a-value></span>
        </div>
        <input data-a type="range" min="5" max="100" step="1" value="${n(populationAInitial)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>种群乙初始数量</span>
          <span class="bl-value" data-b-value></span>
        </div>
        <input data-b type="range" min="5" max="100" step="1" value="${n(populationBInitial)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>捕食作用强度</span>
          <span class="bl-value" data-predation-value></span>
        </div>
        <input data-predation type="range" min="0" max="100" step="1" value="${n(predationStrength)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>生态位重叠程度</span>
          <span class="bl-value" data-overlap-value></span>
        </div>
        <input data-overlap type="range" min="0" max="100" step="1" value="${n(nicheOverlap)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>观察时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="5" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="pc-subtitle">种群关系模式</div>

      <div class="pc-buttons">
        <button type="button" class="pc-button active" data-mode="predation">捕食关系</button>
        <button type="button" class="pc-button" data-mode="competition">资源竞争</button>
        <button type="button" class="pc-button" data-mode="compare">综合比较</button>
      </div>

      <div class="pc-subtitle">快速比较情境</div>

      <div class="pc-scenarios">
        <button type="button" class="pc-button active" data-scenario="balanced">捕食协调</button>
        <button type="button" class="pc-button" data-scenario="predatorBoom">捕食者增多</button>
        <button type="button" class="pc-button" data-scenario="preyScarcity">猎物匮乏</button>
        <button type="button" class="pc-button" data-scenario="overlap">高度重叠</button>
        <button type="button" class="pc-button" data-scenario="partition">生态位分化</button>
      </div>

      <button type="button" class="pc-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '关系标注：显示' : '关系标注：隐藏'}
      </button>

      <div class="pc-status">
        <div class="pc-card">
          <b data-a-final></b>
          <span data-a-label>种群甲末期</span>
        </div>

        <div class="pc-card">
          <b data-b-final></b>
          <span data-b-label>种群乙末期</span>
        </div>

        <div class="pc-card">
          <b data-state></b>
          <span>关系状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="捕食与竞争种群关系互动模型"
      >
        <defs>
          <linearGradient id="${rootId}-habitat" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#EFF6FF"/>
            <stop offset="100%" stop-color="#ECFDF5"/>
          </linearGradient>

          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#14532D" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text x="22" y="34" data-title font-size="25" font-weight="900" fill="#166534"></text>
        <text x="22" y="62" data-summary font-size="13" font-weight="800" fill="#475569"></text>

        <!-- 左上：种群关系场景 -->
        <g filter="url(#${rootId}-shadow)">
          <rect x="22" y="78" width="304" height="144" rx="19" fill="url(#${rootId}-habitat)" stroke="#A7F3D0" stroke-width="3"/>
        </g>

        <g data-scene-layer></g>
        <g data-relation-layer></g>

        <!-- 右上：相对资源利用 -->
        <g transform="translate(346 78)">
          <rect width="390" height="144" rx="19" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>

          <text x="195" y="25" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">资源与相互作用状态</text>

          <text x="16" y="55" font-size="10.5" font-weight="800" fill="#64748B">环境资源</text>
          <rect x="88" y="46" width="270" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-resource-bar x="88" y="46" width="0" height="12" rx="6" fill="#22C55E"/>

          <text x="16" y="85" data-pressure-label font-size="10.5" font-weight="800" fill="#64748B">捕食压力</text>
          <rect x="88" y="76" width="270" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-pressure-bar x="88" y="76" width="0" height="12" rx="6" fill="#EF4444"/>

          <text x="16" y="115" data-overlap-label font-size="10.5" font-weight="800" fill="#64748B">生态位重叠</text>
          <rect x="88" y="106" width="270" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-overlap-bar x="88" y="106" width="0" height="12" rx="6" fill="#F59E0B"/>

          <text x="195" y="136" data-panel-note text-anchor="middle" font-size="10.5" font-weight="900" fill="#166534"></text>
        </g>

        <!-- 下方种群曲线 -->
        <text x="22" y="253" data-chart-title font-size="13" font-weight="900" fill="#334155"></text>

        <line x1="62" y1="380" x2="704" y2="380" stroke="#64748B" stroke-width="2.5"/>
        <line x1="62" y1="380" x2="62" y2="270" stroke="#64748B" stroke-width="2.5"/>

        <text x="708" y="384" font-size="10.5" font-weight="800" fill="#64748B">时间</text>
        <text x="18" y="280" font-size="10.5" font-weight="800" fill="#64748B">数量</text>

        <g data-grid></g>

        <path data-a-area fill="#86EFAC" opacity=".23"></path>
        <path data-b-area fill="#FCA5A5" opacity=".18"></path>

        <path
          data-a-curve
          fill="none"
          stroke="#16A34A"
          stroke-width="5"
          stroke-linecap="round"
          stroke-linejoin="round"
        ></path>

        <path
          data-b-curve
          fill="none"
          stroke="#DC2626"
          stroke-width="5"
          stroke-linecap="round"
          stroke-linejoin="round"
        ></path>

        <g data-points></g>

        <g data-label-layer>
          <g transform="translate(80 405)">
            <circle cx="7" cy="0" r="6" fill="#16A34A"/>
            <text x="20" y="5" data-a-legend font-size="11" font-weight="900" fill="#475569"></text>
          </g>

          <g transform="translate(255 405)">
            <circle cx="7" cy="0" r="6" fill="#DC2626"/>
            <text x="20" y="5" data-b-legend font-size="11" font-weight="900" fill="#475569"></text>
          </g>

          <text x="470" y="410" data-stage-note font-size="11" font-weight="900" fill="#166534"></text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var resource=root.querySelector('[data-resource]');
    var populationA=root.querySelector('[data-a]');
    var populationB=root.querySelector('[data-b]');
    var predation=root.querySelector('[data-predation]');
    var overlap=root.querySelector('[data-overlap]');
    var time=root.querySelector('[data-time]');

    var resourceValue=root.querySelector('[data-resource-value]');
    var populationAValue=root.querySelector('[data-a-value]');
    var populationBValue=root.querySelector('[data-b-value]');
    var predationValue=root.querySelector('[data-predation-value]');
    var overlapValue=root.querySelector('[data-overlap-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var aFinalText=root.querySelector('[data-a-final]');
    var bFinalText=root.querySelector('[data-b-final]');
    var aStatusLabel=root.querySelector('[data-a-label]');
    var bStatusLabel=root.querySelector('[data-b-label]');
    var stateText=root.querySelector('[data-state]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var sceneLayer=root.querySelector('[data-scene-layer]');
    var relationLayer=root.querySelector('[data-relation-layer]');

    var resourceBar=root.querySelector('[data-resource-bar]');
    var pressureBar=root.querySelector('[data-pressure-bar]');
    var overlapBar=root.querySelector('[data-overlap-bar]');
    var pressureLabel=root.querySelector('[data-pressure-label]');
    var overlapLabel=root.querySelector('[data-overlap-label]');
    var panelNote=root.querySelector('[data-panel-note]');

    var chartTitle=root.querySelector('[data-chart-title]');
    var grid=root.querySelector('[data-grid]');
    var aArea=root.querySelector('[data-a-area]');
    var bArea=root.querySelector('[data-b-area]');
    var aCurve=root.querySelector('[data-a-curve]');
    var bCurve=root.querySelector('[data-b-curve]');
    var points=root.querySelector('[data-points]');

    var labelLayer=root.querySelector('[data-label-layer]');
    var aLegend=root.querySelector('[data-a-legend]');
    var bLegend=root.querySelector('[data-b-legend]');
    var stageNote=root.querySelector('[data-stage-note]');

    var mode='predation';
    var showLabels=${showLabels ? 'true' : 'false'};
    var steps=36;

    var scenarios={
      balanced:{
        mode:'predation',
        resource:76,
        a:64,
        b:34,
        predation:58,
        overlap:45,
        time:72
      },
      predatorBoom:{
        mode:'predation',
        resource:72,
        a:62,
        b:88,
        predation:78,
        overlap:45,
        time:68
      },
      preyScarcity:{
        mode:'predation',
        resource:24,
        a:28,
        b:64,
        predation:82,
        overlap:45,
        time:76
      },
      overlap:{
        mode:'competition',
        resource:58,
        a:68,
        b:62,
        predation:20,
        overlap:96,
        time:78
      },
      partition:{
        mode:'competition',
        resource:76,
        a:62,
        b:58,
        predation:20,
        overlap:16,
        time:78
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
     * 简化捕食—被捕食模型。
     *
     * 猎物受到环境容纳量限制；
     * 捕食者依赖捕食获得的资源增长；
     * 捕食者本身也存在基础死亡。
     */
    function simulatePredation(
      resourceLevel,
      preyInitial,
      predatorInitial,
      predationLevel
    ){
      var carryingCapacity=
        90+resourceLevel*4.5;

      var prey=
        18+preyInitial*3.7;

      var predator=
        6+predatorInitial*1.9;

      var preyGrowthRate=
        .13+.08*resourceLevel/100;

      var attackRate=
        .00055
        +predationLevel*.000018;

      var conversion=
        .08+.1*predationLevel/100;

      var predatorDeath=
        .065
        +(100-resourceLevel)*.00018;

      var records=[{
        a:prey,
        b:predator
      }];

      for(var i=1;i<=steps;i++){
        var preyGrowth=
          preyGrowthRate
          *prey
          *(1-prey/carryingCapacity);

        var consumed=
          attackRate
          *prey
          *predator;

        var nextPrey=
          prey+preyGrowth-consumed;

        var nextPredator=
          predator
          +conversion*consumed
          -predatorDeath*predator;

        prey=clamp(
          nextPrey,
          0,
          carryingCapacity*1.4
        );

        predator=clamp(
          nextPredator,
          0,
          carryingCapacity*.9
        );

        records.push({
          a:prey,
          b:predator
        });
      }

      return {
        records:records,
        carryingCapacity:carryingCapacity
      };
    }

    /**
     * 简化种间竞争模型。
     *
     * 两个种群共享有限资源；
     * 生态位重叠越高，彼此对对方增长的抑制越强；
     * 生态位分化可降低竞争系数。
     */
    function simulateCompetition(
      resourceLevel,
      speciesAInitial,
      speciesBInitial,
      overlapLevel
    ){
      var carryingCapacity=
        85+resourceLevel*4.3;

      var a=
        16+speciesAInitial*3.2;

      var b=
        16+speciesBInitial*3.2;

      var overlapFactor=
        overlapLevel/100;

      var alpha=
        .12+1.22*overlapFactor;

      var beta=
        .14+1.08*overlapFactor;

      var growthA=
        .115+.035*resourceLevel/100;

      var growthB=
        .108+.032*resourceLevel/100;

      var capacityA=
        carryingCapacity;

      var capacityB=
        carryingCapacity*.92;

      var records=[{
        a:a,
        b:b
      }];

      for(var i=1;i<=steps;i++){
        var nextA=
          a
          +growthA*a
          *(
            1-(a+alpha*b)/capacityA
          );

        var nextB=
          b
          +growthB*b
          *(
            1-(b+beta*a)/capacityB
          );

        a=clamp(
          nextA,
          0,
          capacityA*1.25
        );

        b=clamp(
          nextB,
          0,
          capacityB*1.25
        );

        records.push({
          a:a,
          b:b
        });
      }

      return {
        records:records,
        carryingCapacity:
          Math.max(
            capacityA,
            capacityB
          )
      };
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

    function recordSlice(
      records,
      endIndex
    ){
      return records.slice(
        0,
        endIndex+1
      );
    }

    function maximumIndex(
      records,
      key
    ){
      var maxIndex=0;

      for(var i=1;i<records.length;i++){
        if(records[i][key]>records[maxIndex][key]){
          maxIndex=i;
        }
      }

      return maxIndex;
    }

    function buildGrid(
      maxValue,
      recordCount
    ){
      var html='';
      var left=62;
      var right=704;
      var top=270;
      var bottom=380;
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

        html+='<text x="'+x+'" y="'+(bottom+17)
          +'" text-anchor="middle" font-size="9.5"'
          +' font-weight="700" fill="#64748B">'
          +index
          +'</text>';
      }

      return html;
    }

    function curveData(
      records,
      key,
      maxValue
    ){
      var left=62;
      var right=704;
      var top=270;
      var bottom=380;
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
        var py=y(records[i][key]);

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
            +'" cy="'+py+'" r="3.8"'
            +' fill="#FFFFFF" stroke="'
            +(key==='a'
              ?'#16A34A'
              :'#DC2626')
            +'" stroke-width="2.4"/>';
        }
      }

      return {
        path:path,
        area:path
          +' L'+right+' '+bottom
          +' L'+left+' '+bottom
          +' Z',
        points:pointHTML
      };
    }

    function buildPredationScene(
      preyLevel,
      predatorLevel
    ){
      var preyCount=clamp(
        Math.floor(2+preyLevel/42),
        2,
        10
      );

      var predatorCount=clamp(
        Math.floor(1+predatorLevel/48),
        1,
        7
      );

      var html='';

      for(var i=0;i<preyCount;i++){
        var x=56+(i%5)*48;
        var y=136+Math.floor(i/5)*48+(i%2)*8;

        html+='<g class="pc-prey"'
          +' transform="translate('+x+' '+y+')">'
          +'<ellipse cx="0" cy="0" rx="17" ry="11"'
          +' fill="#FDE68A" stroke="#D97706"'
          +' stroke-width="2.5"/>'
          +'<circle cx="15" cy="-3" r="8"'
          +' fill="#FACC15" stroke="#D97706"'
          +' stroke-width="2"/>'
          +'<ellipse cx="18" cy="-13" rx="3.5" ry="8"'
          +' fill="#FDE68A" stroke="#D97706"'
          +' stroke-width="1.5"/>'
          +'<circle cx="18" cy="-5" r="1.5" fill="#111827"/>'
          +'</g>';
      }

      for(var j=0;j<predatorCount;j++){
        var px=208+(j%3)*34;
        var py=118+Math.floor(j/3)*43+(j%2)*7;

        html+='<g class="pc-predator"'
          +' transform="translate('+px+' '+py+')">'
          +'<ellipse cx="0" cy="0" rx="18" ry="10"'
          +' fill="#CBD5E1" stroke="#475569"'
          +' stroke-width="2.5"/>'
          +'<circle cx="-16" cy="-3" r="8"'
          +' fill="#94A3B8" stroke="#475569"'
          +' stroke-width="2"/>'
          +'<path d="M-21 -9 L-17 -18 L-11 -9"'
          +' fill="#94A3B8" stroke="#475569"'
          +' stroke-width="1.5"/>'
          +'<circle cx="-19" cy="-4" r="1.5" fill="#111827"/>'
          +'</g>';
      }

      return html;
    }

    function buildCompetitionScene(
      aLevel,
      bLevel,
      overlapLevel
    ){
      var aCount=clamp(
        Math.floor(2+aLevel/45),
        2,
        9
      );

      var bCount=clamp(
        Math.floor(2+bLevel/45),
        2,
        9
      );

      var html='';

      html+='<ellipse cx="153" cy="184" rx="112" ry="23"'
        +' fill="#D1FAE5" stroke="#16A34A"'
        +' stroke-width="3" opacity=".82"/>';

      html+='<text x="153" y="190" text-anchor="middle"'
        +' font-size="12" font-weight="900" fill="#166534">'
        +'共同利用的有限资源'
        +'</text>';

      for(var i=0;i<aCount;i++){
        var x=48+(i%5)*37;
        var y=119+Math.floor(i/5)*37+(i%2)*5;

        html+='<g class="pc-prey"'
          +' transform="translate('+x+' '+y+')">'
          +'<circle cx="0" cy="0" r="10"'
          +' fill="#60A5FA" stroke="#1D4ED8"'
          +' stroke-width="2.5"/>'
          +'<text x="0" y="4" text-anchor="middle"'
          +' font-size="8" font-weight="900" fill="#FFFFFF">甲</text>'
          +'</g>';
      }

      for(var j=0;j<bCount;j++){
        var bx=178+(j%4)*35;
        var by=112+Math.floor(j/4)*38+(j%2)*6;

        html+='<g class="pc-predator"'
          +' transform="translate('+bx+' '+by+')">'
          +'<circle cx="0" cy="0" r="10"'
          +' fill="#FDBA74" stroke="#C2410C"'
          +' stroke-width="2.5"/>'
          +'<text x="0" y="4" text-anchor="middle"'
          +' font-size="8" font-weight="900" fill="#FFFFFF">乙</text>'
          +'</g>';
      }

      html+='<path class="pc-flow" d="M104 169'
        +' C119 153 132 146 145 145"'
        +' fill="none" stroke="#0284C7"'
        +' stroke-width="'+(2+overlapLevel/30)
        +'" marker-end="url(#${rootId}-arrow-blue)"/>';

      html+='<path class="pc-flow" d="M210 166'
        +' C193 150 178 145 162 145"'
        +' fill="none" stroke="#F59E0B"'
        +' stroke-width="'+(2+overlapLevel/30)
        +'" marker-end="url(#${rootId}-arrow-orange)"/>';

      return html;
    }

    function buildPredationRelations(){
      return ''
        +'<path class="pc-flow" d="M104 112'
        +' C143 89 192 89 226 111"'
        +' fill="none" stroke="#16A34A"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-green)"/>'
        +'<text x="165" y="93" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#166534">'
        +'猎物为捕食者提供食物'
        +'</text>'
        +'<path class="pc-flow" d="M226 198'
        +' C189 219 139 219 104 199"'
        +' fill="none" stroke="#DC2626"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-red)"/>'
        +'<text x="165" y="218" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#B91C1C">'
        +'捕食者对猎物形成捕食压力'
        +'</text>';
    }

    function buildCompetitionRelations(
      overlapLevel
    ){
      return ''
        +'<text x="153" y="96" text-anchor="middle"'
        +' font-size="11" font-weight="900" fill="#475569">'
        +'生态位重叠 '
        +overlapLevel.toFixed(0)
        +'%'
        +'</text>'
        +'<path d="M72 103 C111 82 138 83 153 103"'
        +' fill="none" stroke="#0284C7" stroke-width="3"/>'
        +'<path d="M234 103 C196 82 168 83 153 103"'
        +' fill="none" stroke="#F59E0B" stroke-width="3"/>';
    }

    function classifyPredation(
      records,
      carryingCapacity
    ){
      var final=
        records[records.length-1];

      if(final.a<carryingCapacity*.04){
        return '猎物极少';
      }

      if(final.b<carryingCapacity*.025){
        return '捕食者受限';
      }

      var preyPeak=
        maximumIndex(records,'a');

      var predatorPeak=
        maximumIndex(records,'b');

      if(predatorPeak>preyPeak){
        return '滞后波动';
      }

      return '相互制约';
    }

    function classifyCompetition(
      records,
      carryingCapacity
    ){
      var final=
        records[records.length-1];

      var aRatio=
        final.a/carryingCapacity;

      var bRatio=
        final.b/carryingCapacity;

      if(aRatio>.12 && bRatio>.12){
        return '共同存在';
      }

      if(aRatio>.12 && bRatio<=.12){
        return '种群甲占优';
      }

      if(bRatio>.12 && aRatio<=.12){
        return '种群乙占优';
      }

      return '双方受限';
    }

    function update(){
      var R=Number(resource.value);
      var A=Number(populationA.value);
      var B=Number(populationB.value);
      var P=Number(predation.value);
      var O=Number(overlap.value);
      var T=Number(time.value);

      resourceValue.textContent=R.toFixed(0)+'%';
      populationAValue.textContent=A.toFixed(0)+'%';
      populationBValue.textContent=B.toFixed(0)+'%';
      predationValue.textContent=P.toFixed(0)+'%';
      overlapValue.textContent=O.toFixed(0)+'%';
      timeValue.textContent=T.toFixed(0)+'%';

      setModeActive();

      labelToggle.textContent=showLabels
        ?'关系标注：显示'
        :'关系标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      var predationData=
        simulatePredation(
          R,
          A,
          B,
          P
        );

      var competitionData=
        simulateCompetition(
          R,
          A,
          B,
          O
        );

      var endIndex=
        visibleIndex(T);

      var activeData=
        mode==='competition'
          ?competitionData
          :mode==='predation'
            ?predationData
            :(
              O>=P
                ?competitionData
                :predationData
            );

      var visibleRecords=
        recordSlice(
          activeData.records,
          endIndex
        );

      var finalRecord=
        visibleRecords[
          visibleRecords.length-1
        ];

      var maxValue=
        activeData.carryingCapacity*1.18;

      for(var i=0;i<visibleRecords.length;i++){
        maxValue=Math.max(
          maxValue,
          visibleRecords[i].a,
          visibleRecords[i].b
        );
      }

      maxValue=Math.max(
        100,
        maxValue*1.05
      );

      grid.innerHTML=
        buildGrid(
          maxValue,
          visibleRecords.length
        );

      var aData=
        curveData(
          visibleRecords,
          'a',
          maxValue
        );

      var bData=
        curveData(
          visibleRecords,
          'b',
          maxValue
        );

      aCurve.setAttribute(
        'd',
        aData.path
      );

      bCurve.setAttribute(
        'd',
        bData.path
      );

      aArea.setAttribute(
        'd',
        aData.area
      );

      bArea.setAttribute(
        'd',
        bData.area
      );

      points.innerHTML=
        aData.points+bData.points;

      resourceBar.setAttribute(
        'width',
        String(270*R/100)
      );

      pressureBar.setAttribute(
        'width',
        String(
          270
          *(mode==='competition'
            ?O
            :P)
          /100
        )
      );

      overlapBar.setAttribute(
        'width',
        String(270*O/100)
      );

      var relationshipState='';
      var explanation='';
      var conditionNote='';

      if(mode==='predation'){
        relationshipState=
          classifyPredation(
            visibleRecords,
            predationData.carryingCapacity
          );

        title.textContent=
          '捕食关系中的种群数量变化';

        summary.textContent=
          '猎物为捕食者提供食物，捕食者增加又会提高猎物受到的捕食压力';

        chartTitle.textContent=
          '猎物与捕食者相对数量随时间变化';

        aLegend.textContent=
          '猎物种群';

        bLegend.textContent=
          '捕食者种群';

        aStatusLabel.textContent=
          '猎物末期数量';

        bStatusLabel.textContent=
          '捕食者末期数量';

        pressureLabel.textContent=
          '捕食压力';

        overlapLabel.textContent=
          '生态位重叠（本模式次要）';

        panelNote.textContent=
          relationshipState;

        sceneLayer.innerHTML=
          buildPredationScene(
            finalRecord.a,
            finalRecord.b
          );

        relationLayer.innerHTML=
          showLabels
            ?buildPredationRelations()
            :'';

        var preyPeak=
          maximumIndex(
            visibleRecords,
            'a'
          );

        var predatorPeak=
          maximumIndex(
            visibleRecords,
            'b'
          );

        var lag=
          predatorPeak-preyPeak;

        stageNote.textContent=
          lag>0
            ?'捕食者峰值比猎物峰值滞后约 '+lag+' 个模拟阶段'
            :'当前条件下未形成明显的峰值滞后';

        explanation=
          '猎物数量增加可以提高捕食者获得食物的机会；捕食者数量增加后，又会提高猎物死亡压力，因此两个种群可能出现相互制约和时间滞后的变化。';

        if(R<28){
          conditionNote=
            '环境资源不足首先限制猎物增长，捕食者随后也会因食物减少而受到限制。';
        }else if(B>78 && P>65){
          conditionNote=
            '捕食者初始数量和捕食强度都较高，猎物短期承受较大压力，捕食者随后也可能因猎物减少而下降。';
        }else if(P<12){
          conditionNote=
            '捕食作用很弱，猎物变化主要由资源和自身密度制约决定。';
        }else if(
          predatorPeak>preyPeak
        ){
          conditionNote=
            '当前教学模型中捕食者高峰晚于猎物高峰，表现出一定的时间滞后。';
        }else{
          conditionNote=
            '当前参数下两个种群相互影响，但没有形成明显而规则的滞后周期。';
        }
      }else if(mode==='competition'){
        relationshipState=
          classifyCompetition(
            visibleRecords,
            competitionData.carryingCapacity
          );

        title.textContent=
          '种间竞争与生态位重叠';

        summary.textContent=
          '两个种群共同利用有限资源，生态位重叠越高，彼此增长受到的抑制通常越强';

        chartTitle.textContent=
          '竞争种群甲与种群乙相对数量变化';

        aLegend.textContent=
          '竞争种群甲';

        bLegend.textContent=
          '竞争种群乙';

        aStatusLabel.textContent=
          '种群甲末期';

        bStatusLabel.textContent=
          '种群乙末期';

        pressureLabel.textContent=
          '竞争压力';

        overlapLabel.textContent=
          '生态位重叠';

        panelNote.textContent=
          relationshipState;

        sceneLayer.innerHTML=
          buildCompetitionScene(
            finalRecord.a,
            finalRecord.b,
            O
          );

        relationLayer.innerHTML=
          showLabels
            ?buildCompetitionRelations(O)
            :'';

        stageNote.textContent=
          O<30
            ?'生态位分化降低了直接竞争'
            :O>80
              ?'高度重叠使竞争作用明显增强'
              :'两个种群存在中等程度资源竞争';

        explanation=
          '当两个种群共同利用有限资源时，双方的实际增长率都可能下降。生态位重叠越高，竞争作用通常越强；利用不同资源、空间或时间可以减弱直接竞争。';

        if(R<28){
          conditionNote=
            '资源总量较低，即使生态位重叠不高，两个种群也可能同时受到资源不足限制。';
        }else if(O>82){
          conditionNote=
            '两个种群生态位高度重叠，竞争抑制明显增强，长期结果可能表现为一方占优或双方数量下降。';
        }else if(O<25){
          conditionNote=
            '生态位分化降低了直接竞争，在资源较充足时两个种群更可能共同存在。';
        }else if(
          relationshipState==='种群甲占优'
          ||relationshipState==='种群乙占优'
        ){
          conditionNote=
            '当前模型中一方逐渐占优，但短期模拟结果不能直接等同于真实生态系统中的永久竞争排斥。';
        }else{
          conditionNote=
            '当前资源供应和生态位重叠允许两个种群维持一定数量。';
        }
      }else{
        var predationFinal=
          predationData.records[
            endIndex
          ];

        var competitionFinal=
          competitionData.records[
            endIndex
          ];

        var predationState=
          classifyPredation(
            recordSlice(
              predationData.records,
              endIndex
            ),
            predationData.carryingCapacity
          );

        var competitionState=
          classifyCompetition(
            recordSlice(
              competitionData.records,
              endIndex
            ),
            competitionData.carryingCapacity
          );

        relationshipState=
          P>=O
            ?predationState
            :competitionState;

        title.textContent=
          '捕食与竞争的作用机制比较';

        summary.textContent=
          '捕食通常使一方获得食物而另一方承受死亡压力，竞争通常使双方因共享有限资源受到抑制';

        chartTitle.textContent=
          P>=O
            ?'当前以捕食作用较强的模拟曲线为主'
            :'当前以竞争作用较强的模拟曲线为主';

        aLegend.textContent=
          P>=O
            ?'猎物种群'
            :'竞争种群甲';

        bLegend.textContent=
          P>=O
            ?'捕食者种群'
            :'竞争种群乙';

        aStatusLabel.textContent=
          '捕食猎物 / 竞争甲';

        bStatusLabel.textContent=
          '捕食者 / 竞争乙';

        pressureLabel.textContent=
          '捕食作用强度';

        overlapLabel.textContent=
          '生态位重叠程度';

        panelNote.textContent=
          '捕食 '+predationState
          +'｜竞争 '+competitionState;

        sceneLayer.innerHTML=
          P>=O
            ?buildPredationScene(
              predationFinal.a,
              predationFinal.b
            )
            :buildCompetitionScene(
              competitionFinal.a,
              competitionFinal.b,
              O
            );

        relationLayer.innerHTML=
          showLabels
            ?(
              P>=O
                ?buildPredationRelations()
                :buildCompetitionRelations(O)
            )
            :'';

        stageNote.textContent=
          '捕食和竞争都可影响种群数量，但双方受益或受损的方向不同';

        explanation=
          '捕食关系中，捕食者通过猎物获得食物，猎物受到死亡压力；种间竞争中，两个种群因共同利用有限资源而都可能受到抑制。';

        if(P>75 && O>75){
          conditionNote=
            '当前同时设置了较强捕食和高度生态位重叠，真实群落中的结果还会受到食物网结构和空间异质性影响。';
        }else if(P>O){
          conditionNote=
            '当前捕食作用强于竞争重叠，曲线主要显示捕食—被捕食种群的相互制约。';
        }else if(O>P){
          conditionNote=
            '当前生态位重叠高于捕食强度，曲线主要显示共享资源带来的竞争抑制。';
        }else{
          conditionNote=
            '当前捕食和竞争参数接近，需要结合具体物种关系判断主要作用机制。';
        }

        finalRecord={
          a:(
            predationFinal.a
            +competitionFinal.a
          )/2,
          b:(
            predationFinal.b
            +competitionFinal.b
          )/2
        };
      }

      aFinalText.textContent=
        finalRecord.a.toFixed(0);

      bFinalText.textContent=
        finalRecord.b.toFixed(0);

      stateText.textContent=
        relationshipState;

      root.style.setProperty(
        '--pc-flow-speed',
        clamp(
          2.5-Math.max(P,O)/62,
          .52,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--pc-prey-speed',
        clamp(
          2.5-finalRecord.a/190,
          .62,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--pc-predator-speed',
        clamp(
          2.7-finalRecord.b/160,
          .68,
          2.7
        ).toFixed(2)+'s'
      );

      var timeNote=T<20
        ?'观察时间较短，暂时的数量差异不能代表长期结局。'
        :'观察时间增加可以显示更多累积变化，但本模型仍不是现实种群预测。';

      result.innerHTML=
        explanation
        +'<br>'+conditionNote
        +' '+timeNote
        +' 捕食和竞争都不应只根据单一时刻的数量判断，所有数值均为相对教学指标。';
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
        resource.value=String(data.resource);
        populationA.value=String(data.a);
        populationB.value=String(data.b);
        predation.value=String(data.predation);
        overlap.value=String(data.overlap);
        time.value=String(data.time);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    resource.oninput=function(){
      setScenarioActive('');
      update();
    };

    populationA.oninput=function(){
      setScenarioActive('');
      update();
    };

    populationB.oninput=function(){
      setScenarioActive('');
      update();
    };

    predation.oninput=function(){
      setScenarioActive('');
      update();
    };

    overlap.oninput=function(){
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
