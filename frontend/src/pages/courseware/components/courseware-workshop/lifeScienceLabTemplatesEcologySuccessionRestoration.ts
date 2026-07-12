/**
 * lifeScienceLabTemplatesEcologySuccessionRestoration.ts
 *
 * 平面生命科学实验室：群落演替与生态恢复。
 *
 * 教学目标：
 * 1. 理解群落演替是群落组成和结构随时间发生变化的过程；
 * 2. 区分初生演替与次生演替；
 * 3. 理解初生演替通常从缺少土壤的裸岩或新生基质开始，
 *    土壤形成和繁殖体到达往往成为重要限制；
 * 4. 理解次生演替发生在原有群落受到干扰后，
 *    土壤、种子库、根系或邻近繁殖体可能部分保留；
 * 5. 观察土壤条件、繁殖体供应、干扰强度、
 *    恢复措施和时间共同影响群落恢复过程；
 * 6. 比较先锋阶段、草本阶段、灌木阶段、
 *    幼龄林阶段和结构较复杂群落的典型变化；
 * 7. 理解物种丰富度、生物量、土壤发育和群落稳定性
 *    可能在演替过程中发生变化；
 * 8. 理解生态恢复可以改善环境条件、补充繁殖体、
 *    控制持续干扰，但不能保证恢复到唯一或完全相同的历史状态；
 * 9. 理解演替并不一定沿固定路线单向进行，
 *    新的干扰可能使群落停滞、退化或转向其他状态。
 *
 * 教学边界：
 * 1. 演替阶段、物种丰富度、生物量、土壤发育和恢复指数
 *    均为相对教学指标；
 * 2. “裸地—先锋—草本—灌木—幼龄林—复杂群落”
 *    是用于比较的一般化序列，不代表所有生态系统都必须依次经历；
 * 3. 草原、湿地、苔原、荒漠和水生生态系统
 *    不应被简单理解为一定要演替成森林；
 * 4. 初生演替和次生演替的速度没有统一固定值，
 *    真实过程受气候、基质、物种来源和干扰历史等因素影响；
 * 5. 本模型中的“复杂群落”不等于永久不变的终极顶极群落；
 * 6. 中等强度干扰在部分生态系统中可能增加斑块异质性，
 *    但高强度、持续或频繁干扰通常会阻碍恢复；
 * 7. 生态恢复措施包括封育、控制侵蚀、补植乡土物种、
 *    恢复水文和减少污染等，本模型使用综合参数表示；
 * 8. 补植或播种不应被理解为物种越多越好，
 *    真实恢复需要考虑乡土性、遗传来源、种间关系和生态过程；
 * 9. 本模型不用于真实工程设计、生态补偿评估或恢复成效验收。
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
function successionRestorationStyle(rootId: string): string {
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
    + '#' + rootId + ' .sr-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .sr-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .sr-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .sr-button{min-height:31px;padding:3px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:9.3px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .sr-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.14)}'
    + '#' + rootId + ' .sr-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#4ADE80,#16A34A);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .sr-toggle.off{background:#64748B}'
    + '#' + rootId + ' .sr-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .sr-card{padding:6px 3px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .sr-card b{display:block;min-height:18px;font-size:13px;color:#15803D}'
    + '#' + rootId + ' .sr-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .sr-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--sr-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .sr-grow{animation:' + rootId + '-grow var(--sr-grow-speed,1.8s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .sr-seed{animation:' + rootId + '-seed var(--sr-seed-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .sr-pulse{animation:' + rootId + '-pulse 1.2s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-grow{from{opacity:.52}to{opacity:1}}'
    + '@keyframes ' + rootId + '-seed{from{transform:translateY(2px);opacity:.42}to{transform:translateY(-5px);opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.42}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_SUCCESSION_RESTORATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-community-succession-restoration',
    group: '🌎 生态系统',
    name: '群落演替与生态恢复',
    emoji: '🌳',
    desc: '调节土壤条件、繁殖体供应、干扰强度、恢复投入和演替时间，比较初生演替、次生演替与生态恢复',
    params: [
      {
        key: 'soilCondition',
        label: '土壤与基质条件',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'propaguleSupply',
        label: '繁殖体供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
        hint: '包括种子库、根系、孢子和邻近群落来源',
      },
      {
        key: 'disturbanceIntensity',
        label: '持续干扰强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 18,
      },
      {
        key: 'restorationEffort',
        label: '生态恢复投入',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 35,
        hint: '综合表示封育、侵蚀控制、补植和水文恢复',
      },
      {
        key: 'successionTime',
        label: '演替与恢复时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'showLabels',
        label: '显示阶段标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const soilCondition = num(
        params,
        'soilCondition',
        68,
      )
      const propaguleSupply = num(
        params,
        'propaguleSupply',
        72,
      )
      const disturbanceIntensity = num(
        params,
        'disturbanceIntensity',
        18,
      )
      const restorationEffort = num(
        params,
        'restorationEffort',
        35,
      )
      const successionTime = num(
        params,
        'successionTime',
        62,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${successionRestorationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌳 群落演替与生态恢复</div>
    <div class="bl-note">演替路径并非固定，干扰和恢复措施可改变方向与速度</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>土壤与基质条件</span>
          <span class="bl-value" data-soil-value></span>
        </div>
        <input data-soil type="range" min="0" max="100" step="1" value="${n(soilCondition)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>繁殖体供应</span>
          <span class="bl-value" data-propagule-value></span>
        </div>
        <input data-propagule type="range" min="0" max="100" step="1" value="${n(propaguleSupply)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>持续干扰强度</span>
          <span class="bl-value" data-disturbance-value></span>
        </div>
        <input data-disturbance type="range" min="0" max="100" step="1" value="${n(disturbanceIntensity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>生态恢复投入</span>
          <span class="bl-value" data-restoration-value></span>
        </div>
        <input data-restoration type="range" min="0" max="100" step="1" value="${n(restorationEffort)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>演替与恢复时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(successionTime)}">
      </div>

      <div class="sr-subtitle">演替或恢复类型</div>

      <div class="sr-buttons">
        <button type="button" class="sr-button active" data-mode="primary">初生演替</button>
        <button type="button" class="sr-button" data-mode="secondary">次生演替</button>
        <button type="button" class="sr-button" data-mode="restoration">生态恢复</button>
      </div>

      <div class="sr-subtitle">快速比较情境</div>

      <div class="sr-scenarios">
        <button type="button" class="sr-button active" data-scenario="lava">新生裸地</button>
        <button type="button" class="sr-button" data-scenario="abandoned">弃耕地</button>
        <button type="button" class="sr-button" data-scenario="fire">火烧迹地</button>
        <button type="button" class="sr-button" data-scenario="grazing">过度放牧</button>
        <button type="button" class="sr-button" data-scenario="active">主动恢复</button>
      </div>

      <button type="button" class="sr-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '阶段标注：显示' : '阶段标注：隐藏'}
      </button>

      <div class="sr-status">
        <div class="sr-card">
          <b data-stage-state></b>
          <span>当前阶段</span>
        </div>

        <div class="sr-card">
          <b data-recovery-index></b>
          <span>恢复综合指数</span>
        </div>

        <div class="sr-card">
          <b data-main-limit></b>
          <span>主要限制因素</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="群落演替与生态恢复互动示意图"
      >
        <defs>
          <linearGradient id="${rootId}-sky" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#F8FAFC"/>
          </linearGradient>

          <linearGradient id="${rootId}-ground" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#D6B47C"/>
            <stop offset="100%" stop-color="#78350F"/>
          </linearGradient>

          <linearGradient id="${rootId}-rock" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#CBD5E1"/>
            <stop offset="100%" stop-color="#64748B"/>
          </linearGradient>

          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#14532D" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>
        <rect width="760" height="265" fill="url(#${rootId}-sky)" opacity=".72"/>

        <text x="22" y="34" data-title font-size="25" font-weight="900" fill="#166534"></text>
        <text x="22" y="62" data-summary font-size="13" font-weight="800" fill="#475569"></text>

        <!-- 当前生态场景 -->
        <g filter="url(#${rootId}-shadow)">
          <rect x="20" y="82" width="500" height="232" rx="22" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>
          <path data-ground d="M24 238 C125 221 220 245 320 229 C399 216 455 229 516 214 V310 H24Z" fill="url(#${rootId}-ground)"/>
          <g data-substrate-layer></g>
          <g data-vegetation-layer></g>
          <g data-disturbance-layer></g>
          <g data-restoration-layer></g>
          <g data-seed-layer></g>
        </g>

        <g data-label-layer>
          <text x="39" y="105" data-habitat-label font-size="13" font-weight="900" fill="#334155"></text>
          <text x="37" y="292" data-soil-label font-size="11.5" font-weight="900" fill="#78350F"></text>
          <text x="322" y="105" data-community-label font-size="11.5" font-weight="900" fill="#166534"></text>
        </g>

        <!-- 右侧状态面板 -->
        <g transform="translate(540 82)">
          <rect width="198" height="232" rx="21" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>

          <text x="99" y="26" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">群落恢复状态</text>

          <text x="14" y="55" font-size="10.5" font-weight="800" fill="#64748B">土壤发育</text>
          <rect x="78" y="46" width="103" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-soil-bar x="78" y="46" width="0" height="12" rx="6" fill="#92400E"/>

          <text x="14" y="91" font-size="10.5" font-weight="800" fill="#64748B">物种丰富度</text>
          <rect x="78" y="82" width="103" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-richness-bar x="78" y="82" width="0" height="12" rx="6" fill="#22C55E"/>

          <text x="14" y="127" font-size="10.5" font-weight="800" fill="#64748B">群落生物量</text>
          <rect x="78" y="118" width="103" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-biomass-bar x="78" y="118" width="0" height="12" rx="6" fill="#16A34A"/>

          <text x="14" y="163" font-size="10.5" font-weight="800" fill="#64748B">结构稳定性</text>
          <rect x="78" y="154" width="103" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-stability-bar x="78" y="154" width="0" height="12" rx="6" fill="#0EA5E9"/>

          <text x="14" y="199" font-size="10.5" font-weight="800" fill="#64748B">侵蚀退化风险</text>
          <rect x="78" y="190" width="103" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-risk-bar x="78" y="190" width="0" height="12" rx="6" fill="#EF4444"/>

          <text data-panel-note x="99" y="222" text-anchor="middle" font-size="11" font-weight="900" fill="#166534"></text>
        </g>

        <!-- 演替阶段轴 -->
        <g data-timeline></g>

        <text x="22" y="414" data-stage-note font-size="11.5" font-weight="900" fill="#166534"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var soil=root.querySelector('[data-soil]');
    var propagule=root.querySelector('[data-propagule]');
    var disturbance=root.querySelector('[data-disturbance]');
    var restoration=root.querySelector('[data-restoration]');
    var time=root.querySelector('[data-time]');

    var soilValue=root.querySelector('[data-soil-value]');
    var propaguleValue=root.querySelector('[data-propagule-value]');
    var disturbanceValue=root.querySelector('[data-disturbance-value]');
    var restorationValue=root.querySelector('[data-restoration-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var stageState=root.querySelector('[data-stage-state]');
    var recoveryIndexText=root.querySelector('[data-recovery-index]');
    var mainLimitText=root.querySelector('[data-main-limit]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var ground=root.querySelector('[data-ground]');
    var substrateLayer=root.querySelector('[data-substrate-layer]');
    var vegetationLayer=root.querySelector('[data-vegetation-layer]');
    var disturbanceLayer=root.querySelector('[data-disturbance-layer]');
    var restorationLayer=root.querySelector('[data-restoration-layer]');
    var seedLayer=root.querySelector('[data-seed-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');

    var habitatLabel=root.querySelector('[data-habitat-label]');
    var soilLabel=root.querySelector('[data-soil-label]');
    var communityLabel=root.querySelector('[data-community-label]');

    var soilBar=root.querySelector('[data-soil-bar]');
    var richnessBar=root.querySelector('[data-richness-bar]');
    var biomassBar=root.querySelector('[data-biomass-bar]');
    var stabilityBar=root.querySelector('[data-stability-bar]');
    var riskBar=root.querySelector('[data-risk-bar]');
    var panelNote=root.querySelector('[data-panel-note]');
    var timeline=root.querySelector('[data-timeline]');
    var stageNote=root.querySelector('[data-stage-note]');

    var mode='primary';
    var showLabels=${showLabels ? 'true' : 'false'};

    var scenarios={
      lava:{
        mode:'primary',
        soil:6,
        propagule:18,
        disturbance:22,
        restoration:4,
        time:48
      },
      abandoned:{
        mode:'secondary',
        soil:78,
        propagule:82,
        disturbance:18,
        restoration:12,
        time:58
      },
      fire:{
        mode:'secondary',
        soil:56,
        propagule:48,
        disturbance:62,
        restoration:22,
        time:46
      },
      grazing:{
        mode:'secondary',
        soil:38,
        propagule:42,
        disturbance:84,
        restoration:18,
        time:52
      },
      active:{
        mode:'restoration',
        soil:48,
        propagule:58,
        disturbance:38,
        restoration:94,
        time:62
      }
    };

    var stageNames=[
      '裸地或强扰动阶段',
      '先锋生物阶段',
      '草本群落阶段',
      '灌木群落阶段',
      '幼龄木本群落',
      '结构较复杂群落'
    ];

    var shortStageNames=[
      '裸地',
      '先锋',
      '草本',
      '灌木',
      '幼龄林',
      '复杂群落'
    ];

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
     * 根据模式和输入条件计算群落状态。
     *
     * 这些公式只用于比较演替与恢复过程，
     * 不代表真实群落预测模型。
     */
    function calculateState(
      soilLevel,
      propaguleLevel,
      disturbanceLevel,
      restorationLevel,
      timeLevel,
      successionMode
    ){
      var timeFactor=clamp(
        1-Math.exp(-timeLevel/31),
        0,
        1
      );

      var disturbanceFactor=clamp(
        1-disturbanceLevel/126,
        .16,
        1
      );

      var propaguleFactor=clamp(
        .16+.84*propaguleLevel/100,
        .16,
        1
      );

      var initialSoil;

      if(successionMode==='primary'){
        initialSoil=
          5+soilLevel*.27;
      }else if(successionMode==='secondary'){
        initialSoil=
          31+soilLevel*.48;
      }else{
        initialSoil=
          28
          +soilLevel*.44
          +restorationLevel*.12;
      }

      var soilDevelopment=clamp(
        initialSoil
        +42*timeFactor*disturbanceFactor
        +(
          successionMode==='restoration'
            ?restorationLevel*.2
            :restorationLevel*.05
        )
        -disturbanceLevel*.16,
        0,
        100
      );

      var speed=
        successionMode==='primary'
          ?.72
          :successionMode==='secondary'
            ?1
            :1.1+.18*restorationLevel/100;

      var restorationContribution=
        successionMode==='restoration'
          ?restorationLevel*.2
          :restorationLevel*.045;

      var recovery=clamp(
        (
          soilDevelopment*.29
          +propaguleLevel*.25
          +timeLevel*.25
          +restorationContribution
        )
        *disturbanceFactor
        *speed,
        0,
        100
      );

      var stageProgress=clamp(
        recovery
        +(
          successionMode==='secondary'
            ?7
            :successionMode==='restoration'
              ?10
              :0
        ),
        0,
        100
      );

      var stageIndex=clamp(
        Math.floor((stageProgress+4)/18),
        0,
        5
      );

      var richness=clamp(
        3
        +recovery*.42
        +propaguleLevel*.12
        -disturbanceLevel*.08,
        1,
        62
      );

      var biomass=clamp(
        Math.pow(recovery/100,1.34)
        *100
        *(.68+.32*soilDevelopment/100),
        0,
        100
      );

      var stability=clamp(
        recovery*.36
        +soilDevelopment*.27
        +clamp(richness/55*100,0,100)*.2
        +(100-disturbanceLevel)*.17,
        0,
        100
      );

      var erosionRisk=clamp(
        83
        -soilDevelopment*.42
        -biomass*.3
        +disturbanceLevel*.42
        -restorationLevel*.25,
        0,
        100
      );

      return {
        timeFactor:timeFactor,
        disturbanceFactor:disturbanceFactor,
        propaguleFactor:propaguleFactor,
        soilDevelopment:soilDevelopment,
        recovery:recovery,
        stageIndex:stageIndex,
        richness:richness,
        biomass:biomass,
        stability:stability,
        erosionRisk:erosionRisk
      };
    }

    /**
     * 绘制裸岩、土壤和枯落物等基质。
     */
    function buildSubstrate(
      soilDevelopment,
      stageIndex,
      successionMode
    ){
      var html='';
      var rockCount=Math.max(
        1,
        Math.floor(7-soilDevelopment/18)
      );

      for(var i=0;i<rockCount;i++){
        var x=52+(i%6)*73;
        var y=249+(i%3)*15;
        var width=35+(i%3)*9;

        html+='<path d="M'+x+' '+y
          +' L'+(x+width*.42)+' '+(y-19-i%2*7)
          +' L'+(x+width)+' '+(y-3)
          +' L'+(x+width*.78)+' '+(y+14)
          +' L'+(x+8)+' '+(y+13)+'Z"'
          +' fill="url(#${rootId}-rock)"'
          +' stroke="#475569" stroke-width="2.5"'
          +' opacity="'+(.35+.5*(100-soilDevelopment)/100)+'"/>';
      }

      var litterCount=Math.floor(
        stageIndex*2+soilDevelopment/28
      );

      for(var j=0;j<litterCount;j++){
        var lx=58+(j%10)*43;
        var ly=282+(j%3)*7;

        html+='<ellipse cx="'+lx+'" cy="'+ly
          +'" rx="'+(8+j%3)
          +'" ry="'+(3+j%2)
          +'" fill="#92400E" opacity=".68"'
          +' transform="rotate('+(j%2===0?18:-22)
          +' '+lx+' '+ly+')"/>';
      }

      if(successionMode==='primary' && stageIndex<2){
        html+='<text x="275" y="283" text-anchor="middle"'
          +' font-size="12" font-weight="900" fill="#475569">'
          +'裸岩风化和有机质积累逐渐形成土壤'
          +'</text>';
      }

      return html;
    }

    /**
     * 绘制先锋地衣和苔藓斑块。
     */
    function buildPioneers(
      stageIndex,
      richness
    ){
      if(stageIndex<1){
        return '';
      }

      var count=Math.floor(
        4+stageIndex*2+richness/12
      );
      var html='';

      for(var i=0;i<count;i++){
        var x=48+(i%11)*41;
        var y=263+(i%3)*12;
        var color=i%2===0
          ?'#84CC16'
          :'#A3E635';

        html+='<ellipse class="sr-grow" cx="'+x
          +'" cy="'+y+'" rx="'+(8+i%4)
          +'" ry="'+(4+i%3)
          +'" fill="'+color+'" stroke="#4D7C0F"'
          +' stroke-width="1.5" opacity=".82"/>';
      }

      return html;
    }

    /**
     * 绘制草本植物。
     */
    function buildGrasses(
      stageIndex,
      richness,
      biomass
    ){
      if(stageIndex<2){
        return '';
      }

      var count=Math.floor(
        8
        +stageIndex*4
        +richness/5
      );
      var html='';

      for(var i=0;i<count;i++){
        var x=39+(i%18)*26;
        var baseY=274+(i%3)*7;
        var height=18+(i%5)*7+biomass*.12;
        var color=i%3===0
          ?'#65A30D'
          :i%3===1
            ?'#16A34A'
            :'#22C55E';

        html+='<path class="sr-grow" d="M'+x+' '+baseY
          +' Q'+(x-5)+' '+(baseY-height*.55)
          +' '+(x-2)+' '+(baseY-height)
          +'" fill="none" stroke="'+color+'"'
          +' stroke-width="'+(2.2+biomass/85)
          +'" stroke-linecap="round"/>';

        html+='<path class="sr-grow" d="M'+(x+2)+' '+baseY
          +' Q'+(x+9)+' '+(baseY-height*.52)
          +' '+(x+7)+' '+(baseY-height*.88)
          +'" fill="none" stroke="'+color+'"'
          +' stroke-width="2.2" stroke-linecap="round"/>';
      }

      return html;
    }

    /**
     * 绘制灌木。
     */
    function buildShrubs(
      stageIndex,
      richness,
      biomass
    ){
      if(stageIndex<3){
        return '';
      }

      var count=Math.floor(
        2+(stageIndex-2)*2+richness/22
      );
      var html='';

      for(var i=0;i<count;i++){
        var x=76+(i%7)*63;
        var y=249-(i%2)*8;
        var radius=20+(i%3)*5+biomass*.05;

        html+='<g class="sr-grow">'
          +'<path d="M'+x+' '+(y+24)
          +' V'+(y+54)
          +'" stroke="#92400E" stroke-width="5"'
          +' stroke-linecap="round"/>'
          +'<circle cx="'+x+'" cy="'+y
          +'" r="'+radius+'" fill="#22C55E"'
          +' stroke="#15803D" stroke-width="3"/>'
          +'<circle cx="'+(x-15)+'" cy="'+(y+7)
          +'" r="'+(radius*.68)+'" fill="#4ADE80"'
          +' stroke="#15803D" stroke-width="2"/>'
          +'<circle cx="'+(x+15)+'" cy="'+(y+8)
          +'" r="'+(radius*.65)+'" fill="#16A34A"'
          +' stroke="#15803D" stroke-width="2"/>'
          +'</g>';
      }

      return html;
    }

    /**
     * 绘制幼龄和较成熟木本植物。
     */
    function buildTrees(
      stageIndex,
      biomass
    ){
      if(stageIndex<4){
        return '';
      }

      var treeCount=stageIndex===4
        ?3
        :6;
      var html='';

      for(var i=0;i<treeCount;i++){
        var x=58+(i%6)*78;
        var baseY=268-(i%2)*5;
        var trunkHeight=
          stageIndex===4
            ?70+(i%3)*12
            :92+(i%3)*19;
        var crownRadius=
          stageIndex===4
            ?28+(i%2)*6
            :37+(i%3)*7;

        var crownY=
          baseY-trunkHeight;

        html+='<g class="sr-grow">'
          +'<path d="M'+x+' '+baseY
          +' V'+crownY
          +'" stroke="#78350F"'
          +' stroke-width="'+(8+biomass/18)
          +'" stroke-linecap="round"/>'
          +'<circle cx="'+x+'" cy="'+crownY
          +'" r="'+crownRadius+'"'
          +' fill="'+(i%2===0?'#16A34A':'#22C55E')
          +'" stroke="#166534" stroke-width="4"/>'
          +'<circle cx="'+(x-20)+'" cy="'+(crownY+9)
          +'" r="'+(crownRadius*.67)
          +'" fill="#4ADE80" stroke="#166534"'
          +' stroke-width="2.5"/>'
          +'<circle cx="'+(x+20)+'" cy="'+(crownY+10)
          +'" r="'+(crownRadius*.64)
          +'" fill="#15803D" stroke="#166534"'
          +' stroke-width="2.5"/>'
          +'</g>';
      }

      if(stageIndex>=5){
        html+='<path d="M44 165 C130 128 214 151 302 123'
          +' C370 102 433 116 492 92"'
          +' fill="none" stroke="#166534"'
          +' stroke-width="4" opacity=".38"/>';

        html+='<text x="289" y="94" text-anchor="middle"'
          +' font-size="11.5" font-weight="900" fill="#166534">'
          +'冠层分层和空间结构更加复杂'
          +'</text>';
      }

      return html;
    }

    function buildVegetation(
      stageIndex,
      richness,
      biomass
    ){
      return ''
        +buildPioneers(
          stageIndex,
          richness
        )
        +buildGrasses(
          stageIndex,
          richness,
          biomass
        )
        +buildShrubs(
          stageIndex,
          richness,
          biomass
        )
        +buildTrees(
          stageIndex,
          biomass
        );
    }

    /**
     * 绘制火烧、放牧和侵蚀等干扰影响。
     */
    function buildDisturbance(
      level
    ){
      if(level<22){
        return '';
      }

      var html='';
      var stumpCount=Math.floor(
        1+level/22
      );

      for(var i=0;i<stumpCount;i++){
        var x=64+(i%7)*65;
        var y=276-(i%2)*8;

        html+='<g opacity="'+(.35+.65*level/100)+'">'
          +'<path d="M'+x+' '+y
          +' V'+(y-35-i%3*6)
          +'" stroke="#451A03" stroke-width="9"'
          +' stroke-linecap="round"/>'
          +'<path d="M'+(x-12)+' '+(y-18)
          +' L'+(x+12)+' '+(y-18)
          +'" stroke="#451A03" stroke-width="5"/>'
          +'</g>';
      }

      if(level>55){
        html+='<path class="sr-flow" d="M37 289'
          +' C154 315 314 311 496 278"'
          +' fill="none" stroke="#DC2626"'
          +' stroke-width="'+(3+level/20)
          +'" marker-end="url(#${rootId}-arrow-red)"'
          +' opacity=".68"/>';

        html+='<text x="257" y="307" text-anchor="middle"'
          +' font-size="11" font-weight="900" fill="#B91C1C">'
          +'持续或高强度干扰使恢复停滞或退化'
          +'</text>';
      }

      return html;
    }

    /**
     * 绘制封育、补植和侵蚀控制等恢复措施。
     */
    function buildRestoration(
      level,
      visible
    ){
      if(!visible || level<8){
        return '';
      }

      var opacity=.28+.72*level/100;
      var html='';

      html+='<g opacity="'+opacity+'">'
        +'<path d="M42 250 H498" stroke="#B45309"'
        +' stroke-width="5" stroke-dasharray="13 8"/>'
        +'<path d="M65 232 V271 M148 232 V271'
        +' M231 232 V271 M314 232 V271'
        +' M397 232 V271 M480 232 V271"'
        +' stroke="#92400E" stroke-width="5"/>'
        +'<text x="267" y="225" text-anchor="middle"'
        +' font-size="11" font-weight="900" fill="#92400E">'
        +'封育与干扰控制'
        +'</text>'
        +'</g>';

      var seedlingCount=Math.floor(
        1+level/18
      );

      for(var i=0;i<seedlingCount;i++){
        var x=92+(i%7)*57;
        var y=262-(i%2)*9;

        html+='<g class="sr-pulse" opacity="'+opacity+'">'
          +'<path d="M'+x+' '+y
          +' V'+(y-26-i%3*5)
          +'" stroke="#15803D" stroke-width="4"/>'
          +'<ellipse cx="'+(x-7)+'" cy="'+(y-20-i%3*5)
          +'" rx="8" ry="4" fill="#4ADE80"'
          +' stroke="#15803D" stroke-width="1.5"'
          +' transform="rotate(-28 '+(x-7)+' '+(y-20-i%3*5)+')"/>'
          +'<ellipse cx="'+(x+7)+'" cy="'+(y-16-i%3*5)
          +'" rx="8" ry="4" fill="#22C55E"'
          +' stroke="#15803D" stroke-width="1.5"'
          +' transform="rotate(28 '+(x+7)+' '+(y-16-i%3*5)+')"/>'
          +'</g>';
      }

      if(level>60){
        html+='<path d="M50 286 C142 269 236 298 328 278'
          +' C386 266 441 272 494 258"'
          +' fill="none" stroke="#0EA5E9"'
          +' stroke-width="4" stroke-dasharray="8 6"'
          +' opacity=".76"/>';

        html+='<text x="358" y="294" font-size="10.5"'
          +' font-weight="900" fill="#0369A1">'
          +'侵蚀控制与水文过程修复'
          +'</text>';
      }

      return html;
    }

    /**
     * 绘制外来繁殖体、种子和孢子到达。
     */
    function buildSeeds(
      level
    ){
      var count=Math.floor(
        1+level/11
      );
      var html='';

      for(var i=0;i<count;i++){
        var x=48+(i%12)*38;
        var y=119+Math.floor(i/12)*22+(i%3)*8;

        html+='<g class="sr-seed">'
          +'<ellipse cx="'+x+'" cy="'+y
          +'" rx="5" ry="2.8" fill="#D97706"'
          +' stroke="#92400E" stroke-width="1.2"'
          +' transform="rotate('+(i%2===0?28:-32)
          +' '+x+' '+y+')"/>'
          +'<path d="M'+x+' '+(y-2)
          +' Q'+(x+7)+' '+(y-11)
          +' '+(x+13)+' '+(y-7)
          +'" fill="none" stroke="#94A3B8"'
          +' stroke-width="1.3"/>'
          +'</g>';
      }

      return html;
    }

    /**
     * 绘制一般化演替阶段轴。
     */
    function buildTimeline(
      activeStage
    ){
      var html=''
        +'<path class="sr-flow" d="M74 354 H691"'
        +' fill="none" stroke="#16A34A"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-green)"/>';

      for(var i=0;i<shortStageNames.length;i++){
        var x=75+i*119;
        var active=i===activeStage;
        var completed=i<activeStage;
        var fill=active
          ?'#16A34A'
          :completed
            ?'#86EFAC'
            :'#FFFFFF';
        var stroke=active
          ?'#166534'
          :'#64748B';

        html+='<circle cx="'+x+'" cy="354" r="17"'
          +' fill="'+fill+'" stroke="'+stroke+'"'
          +' stroke-width="'+(active?4:2.5)+'"/>';

        html+='<text x="'+x+'" y="359" text-anchor="middle"'
          +' font-size="10" font-weight="900"'
          +' fill="'+(active?'#FFFFFF':'#334155')+'">'
          +(i+1)
          +'</text>';

        html+='<text x="'+x+'" y="385" text-anchor="middle"'
          +' font-size="10" font-weight="900"'
          +' fill="'+(active?'#166534':'#64748B')+'">'
          +shortStageNames[i]
          +'</text>';
      }

      return html;
    }

    function update(){
      var S=Number(soil.value);
      var P=Number(propagule.value);
      var D=Number(disturbance.value);
      var R=Number(restoration.value);
      var T=Number(time.value);

      soilValue.textContent=S.toFixed(0)+'%';
      propaguleValue.textContent=P.toFixed(0)+'%';
      disturbanceValue.textContent=D.toFixed(0)+'%';
      restorationValue.textContent=R.toFixed(0)+'%';
      timeValue.textContent=T.toFixed(0)+'%';

      setModeActive();

      labelToggle.textContent=showLabels
        ?'阶段标注：显示'
        :'阶段标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      var state=calculateState(
        S,
        P,
        D,
        R,
        T,
        mode
      );

      stageState.textContent=
        shortStageNames[
          state.stageIndex
        ];

      recoveryIndexText.textContent=
        state.recovery.toFixed(0);

      var limitingFactors=[
        state.soilDevelopment/100,
        state.propaguleFactor,
        state.disturbanceFactor,
        .25+.75*T/100
      ];

      var limitingNames=[
        '土壤基质',
        '繁殖体来源',
        '持续干扰',
        '恢复时间'
      ];

      if(mode==='restoration'){
        limitingFactors.push(
          .18+.82*R/100
        );
        limitingNames.push(
          '恢复投入'
        );
      }

      var limitIndex=0;

      for(var i=1;i<limitingFactors.length;i++){
        if(limitingFactors[i]<limitingFactors[limitIndex]){
          limitIndex=i;
        }
      }

      mainLimitText.textContent=
        limitingNames[limitIndex];

      soilBar.setAttribute(
        'width',
        String(
          103*state.soilDevelopment/100
        )
      );

      richnessBar.setAttribute(
        'width',
        String(
          103*clamp(
            state.richness/55,
            0,
            1
          )
        )
      );

      biomassBar.setAttribute(
        'width',
        String(
          103*state.biomass/100
        )
      );

      stabilityBar.setAttribute(
        'width',
        String(
          103*state.stability/100
        )
      );

      riskBar.setAttribute(
        'width',
        String(
          103*state.erosionRisk/100
        )
      );

      riskBar.setAttribute(
        'fill',
        state.erosionRisk>65
          ?'#DC2626'
          :state.erosionRisk>35
            ?'#F59E0B'
            :'#22C55E'
      );

      panelNote.textContent=
        state.recovery>72
          ?'恢复程度较高'
          :state.recovery>42
            ?'处于恢复过程中'
            :D>65
              ?'持续干扰占主导'
              :'恢复仍处早期';

      root.style.setProperty(
        '--sr-flow-speed',
        clamp(
          2.5-state.recovery/68,
          .52,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--sr-grow-speed',
        clamp(
          2.6-state.recovery/72,
          .58,
          2.6
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--sr-seed-speed',
        clamp(
          2.6-P/64,
          .62,
          2.6
        ).toFixed(2)+'s'
      );

      var soilDepth=clamp(
        44+state.soilDevelopment*.55,
        44,
        99
      );

      ground.setAttribute(
        'd',
        'M24 '+(310-soilDepth)
        +' C125 '+(294-soilDepth*.75)
        +' 220 '+(314-soilDepth*.78)
        +' 320 '+(298-soilDepth*.72)
        +' C399 '+(286-soilDepth*.62)
        +' 455 '+(294-soilDepth*.65)
        +' 516 '+(278-soilDepth*.55)
        +' V310 H24Z'
      );

      ground.setAttribute(
        'opacity',
        String(
          .45+.55*state.soilDevelopment/100
        )
      );

      substrateLayer.innerHTML=
        buildSubstrate(
          state.soilDevelopment,
          state.stageIndex,
          mode
        );

      vegetationLayer.innerHTML=
        buildVegetation(
          state.stageIndex,
          state.richness,
          state.biomass
        );

      disturbanceLayer.innerHTML=
        buildDisturbance(D);

      restorationLayer.innerHTML=
        buildRestoration(
          R,
          mode==='restoration'
        );

      seedLayer.innerHTML=
        buildSeeds(P);

      timeline.innerHTML=
        buildTimeline(
          state.stageIndex
        );

      habitatLabel.textContent=
        mode==='primary'
          ?'起点：裸岩或新生基质，原有土壤很少'
          :mode==='secondary'
            ?'起点：原群落受扰，但土壤和部分繁殖体仍可能保留'
            :'起点：受损生态系统，在减少干扰基础上实施恢复措施';

      soilLabel.textContent=
        '土壤发育 '
        +state.soilDevelopment.toFixed(0)
        +'｜侵蚀风险 '
        +state.erosionRisk.toFixed(0);

      communityLabel.textContent=
        '物种丰富度约 '
        +state.richness.toFixed(0)
        +'｜生物量 '
        +state.biomass.toFixed(0);

      var explanation='';
      var conditionNote='';

      if(mode==='primary'){
        title.textContent=
          '初生演替：从缺少土壤的新生基质开始';

        summary.textContent=
          '先锋生物促进风化和有机质积累，土壤形成后更多物种才可能建立';

        stageNote.textContent=
          '一般化序列仅用于比较，真实初生演替的路径和速度因环境而异';

        explanation=
          '初生演替通常发生在新形成或原有土壤被彻底破坏的基质上。地衣、苔藓和部分微生物等先锋生物可参与基质风化和有机质积累。';

        if(S<15){
          conditionNote=
            '当前土壤发育程度很低，是群落建立的主要限制。';
        }else if(P<20){
          conditionNote=
            '虽然基质条件有所改善，但繁殖体到达较少，群落扩展仍较缓慢。';
        }else if(D>62){
          conditionNote=
            '持续强干扰不断破坏早期建立的生物和土壤，使初生演替难以推进。';
        }else if(T<20){
          conditionNote=
            '演替时间较短，当前仍主要表现为先锋生物和早期土壤形成。';
        }else{
          conditionNote=
            '随着土壤、有机质和繁殖体来源增加，更多草本、灌木或木本植物可能逐步建立。';
        }
      }else if(mode==='secondary'){
        title.textContent=
          '次生演替：保留土壤基础上的群落重建';

        summary.textContent=
          '种子库、残存根系、土壤生物和邻近群落可加快恢复，但持续干扰仍会阻碍演替';

        stageNote.textContent=
          '次生演替通常比缺少土壤的初生演替更快，但并不存在统一恢复速度';

        explanation=
          '火灾、弃耕、砍伐或风暴等干扰后，如果土壤和部分生物遗存仍在，群落可依靠种子库、萌蘖和外来繁殖体重新建立。';

        if(D>75){
          conditionNote=
            '当前持续干扰过强，已有植被反复受损，群落可能长期停留在退化阶段。';
        }else if(P<22){
          conditionNote=
            '当前繁殖体来源不足，即使保留土壤，也难以快速恢复物种组成。';
        }else if(S<25){
          conditionNote=
            '土壤侵蚀或退化较严重，次生演替的速度明显下降。';
        }else if(state.recovery>68){
          conditionNote=
            '当前土壤、繁殖体和时间条件较好，群落结构正在向更复杂状态恢复。';
        }else{
          conditionNote=
            '当前处于次生演替中期，草本、灌木和木本植物可能形成镶嵌分布。';
        }
      }else{
        title.textContent=
          '生态恢复：减少压力并重建关键生态过程';

        summary.textContent=
          '恢复措施可改善基质、补充乡土繁殖体和控制侵蚀，但不能替代自然过程或保证唯一终点';

        stageNote.textContent=
          '恢复目标应结合当地生态系统类型，不应把所有环境都设定为森林化';

        explanation=
          '生态恢复首先需要识别持续退化原因，再通过封育、减少污染、恢复水文、控制侵蚀或补植适宜乡土物种等措施促进生态过程恢复。';

        if(D>72 && R<45){
          conditionNote=
            '当前持续干扰仍然很强，恢复投入不足以抵消退化压力，应先降低主要干扰。';
        }else if(R>75 && P<25){
          conditionNote=
            '恢复投入较高，但繁殖体和物种来源不足，仍需谨慎补充适宜的乡土物种或连接邻近生境。';
        }else if(R>75 && S<28){
          conditionNote=
            '当前基质和土壤退化严重，单纯补植可能难以维持，应优先改善水土和基质条件。';
        }else if(state.recovery>70){
          conditionNote=
            '当前干扰得到控制，土壤、繁殖体和恢复措施共同促进群落结构及生态过程改善。';
        }else{
          conditionNote=
            '当前恢复已经产生一定效果，但群落组成和生态功能仍需要较长时间发展。';
        }
      }

      var pathNote='';

      if(D>72){
        pathNote=
          '高强度持续干扰可能使群落退化或转向另一种稳定状态。';
      }else if(state.stageIndex>=5){
        pathNote=
          '当前达到结构较复杂阶段，但群落仍会随气候、物种相互作用和新的干扰继续变化。';
      }else{
        pathNote=
          '当前阶段不是不可逆的固定终点，后续条件变化仍可能改变演替方向。';
      }

      var timeNote=T<15
        ?'演替时间较短，累计变化尚不明显。'
        :'时间可以促进演替和恢复，但不能自动弥补严重土壤退化、繁殖体缺乏或持续强干扰。';

      result.innerHTML=explanation
        +'<br>'+conditionNote
        +' '+timeNote
        +' '+pathNote
        +' 当前阶段为“'
        +stageNames[state.stageIndex]
        +'”，所有数值均为相对教学指标。';
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
        soil.value=String(data.soil);
        propagule.value=String(data.propagule);
        disturbance.value=String(data.disturbance);
        restoration.value=String(data.restoration);
        time.value=String(data.time);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    soil.oninput=function(){
      setScenarioActive('');
      update();
    };

    propagule.oninput=function(){
      setScenarioActive('');
      update();
    };

    disturbance.oninput=function(){
      setScenarioActive('');
      update();
    };

    restoration.oninput=function(){
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
