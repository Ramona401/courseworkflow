/**
 * lifeScienceLabTemplatesEcologySymbiosisParasitism.ts
 *
 * 平面生命科学实验室：
 * 互利共生、偏利共生与寄生种间关系。
 *
 * 教学目标：
 * 1. 使用“+、0、−”描述种间关系对双方相对适合度的影响；
 * 2. 理解互利共生中双方都能获得净收益，表示为“+/+”；
 * 3. 区分专性互利与兼性互利：
 *    依赖程度越高，失去共生伙伴后的负面影响通常越明显；
 * 4. 理解偏利共生中一方获得收益，另一方净影响较小，
 *    一般简化表示为“+/0”；
 * 5. 理解偏利共生中的“0”表示当前尺度上
 *    未观察到明显净收益或净损害，不等于双方没有接触；
 * 6. 理解寄生关系中寄生者获得资源，宿主受到损害，
 *    一般表示为“+/−”；
 * 7. 理解寄生者通常在一段时间内利用宿主，
 *    不应简单等同于捕食者立即杀死并取食猎物；
 * 8. 观察资源供应、相互作用强度、依赖程度、
 *    寄生负荷、宿主抗性和时间对双方状态的影响；
 * 9. 理解种间关系可能随环境、生命阶段和观察尺度变化，
 *    不能仅凭物种名称永久固定其关系类型。
 *
 * 教学边界：
 * 1. 所有种群状态、净效应、寄生负荷和关系强度
 *    均为相对教学指标；
 * 2. “+/+、+/0、+/−”表示相对于缺少该种间关系时
 *    双方适合度或增长表现的相对变化；
 * 3. 互利关系也可能存在交换成本，
 *    当环境资源变化或伙伴回报下降时，净收益可能减弱；
 * 4. 互利共生不等于双方永远完全依赖，
 *    有些互利关系是专性的，有些是兼性的；
 * 5. 偏利共生中的宿主可能存在轻微成本或收益，
 *    只是当前模型把较小净效应近似处理为0；
 * 6. 寄生者数量增加不必然使宿主立即死亡，
 *    但持续高寄生负荷可能降低宿主生长、繁殖或存活；
 * 7. 宿主抗性可以降低寄生者成功率，
 *    但免疫、防御或修复也可能消耗宿主资源；
 * 8. 寄生关系、病原感染和捕食关系之间存在概念差异，
 *    本模型不讨论医学诊断和具体疾病治疗；
 * 9. 豆科植物与根瘤菌、开花植物与传粉者、
 *    附生植物与支持植物等仅作为关系示例，
 *    不代表所有相关物种在所有环境中具有完全相同的结果；
 * 10. 本模型不用于农业用菌、疾病防控、寄生虫治疗
 *     或真实生态风险评估。
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
function symbiosisParasitismStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#DCFCE7);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:250px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#8B5CF6}'
    + '#' + rootId + ' .sp-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .sp-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .sp-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .sp-button{min-height:31px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:9.1px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .sp-button.active{border-color:#8B5CF6;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .sp-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .sp-toggle.off{background:#64748B}'
    + '#' + rootId + ' .sp-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .sp-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .sp-card b{display:block;min-height:18px;font-size:13px;color:#6D28D9}'
    + '#' + rootId + ' .sp-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .sp-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--sp-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .sp-a{animation:' + rootId + '-a var(--sp-a-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .sp-b{animation:' + rootId + '-b var(--sp-b-speed,1.9s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .sp-pulse{animation:' + rootId + '-pulse 1.1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-a{from{transform:translateY(2px);opacity:.52}to{transform:translateY(-4px);opacity:1}}'
    + '@keyframes ' + rootId + '-b{from{transform:translateX(-3px);opacity:.5}to{transform:translateX(4px);opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.38}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_SYMBIOSIS_PARASITISM:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-symbiosis-parasitism-interactions',
    group: '🌎 生态系统',
    name: '互利共生、偏利共生与寄生',
    emoji: '🤝',
    desc: '调节资源、相互作用强度、依赖程度、寄生负荷、宿主抗性和时间，比较+/+、+/0与+/−关系',
    params: [
      {
        key: 'resourceSupply',
        label: '环境资源供应',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'interactionStrength',
        label: '种间作用强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'dependencyLevel',
        label: '伙伴依赖程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 55,
        hint: '互利模式中表示双方对伙伴服务的依赖程度',
      },
      {
        key: 'parasiteLoad',
        label: '寄生者相对负荷',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
      },
      {
        key: 'hostResistance',
        label: '宿主防御与抗性',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'observationTime',
        label: '作用时间',
        type: 'number',
        min: 5,
        max: 100,
        step: 1,
        defaultValue: 70,
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
        72,
      )
      const interactionStrength = num(
        params,
        'interactionStrength',
        68,
      )
      const dependencyLevel = num(
        params,
        'dependencyLevel',
        55,
      )
      const parasiteLoad = num(
        params,
        'parasiteLoad',
        42,
      )
      const hostResistance = num(
        params,
        'hostResistance',
        62,
      )
      const observationTime = num(
        params,
        'observationTime',
        70,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${symbiosisParasitismStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🤝 互利共生、偏利共生与寄生</div>
    <div class="bl-note">关系符号表示双方相对适合度的净变化，不代表永久不变的物种标签</div>
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
          <span>种间作用强度</span>
          <span class="bl-value" data-interaction-value></span>
        </div>
        <input data-interaction type="range" min="0" max="100" step="1" value="${n(interactionStrength)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>伙伴依赖程度</span>
          <span class="bl-value" data-dependency-value></span>
        </div>
        <input data-dependency type="range" min="0" max="100" step="1" value="${n(dependencyLevel)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>寄生者相对负荷</span>
          <span class="bl-value" data-parasite-value></span>
        </div>
        <input data-parasite type="range" min="0" max="100" step="1" value="${n(parasiteLoad)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>宿主防御与抗性</span>
          <span class="bl-value" data-resistance-value></span>
        </div>
        <input data-resistance type="range" min="0" max="100" step="1" value="${n(hostResistance)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>作用时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="5" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="sp-subtitle">种间关系类型</div>

      <div class="sp-buttons">
        <button type="button" class="sp-button active" data-mode="mutualism">互利共生</button>
        <button type="button" class="sp-button" data-mode="commensalism">偏利共生</button>
        <button type="button" class="sp-button" data-mode="parasitism">寄生关系</button>
      </div>

      <div class="sp-subtitle">快速比较情境</div>

      <div class="sp-scenarios">
        <button type="button" class="sp-button active" data-scenario="rootNodule">根瘤共生</button>
        <button type="button" class="sp-button" data-scenario="pollination">传粉互利</button>
        <button type="button" class="sp-button" data-scenario="epiphyte">附生关系</button>
        <button type="button" class="sp-button" data-scenario="infection">高寄生负荷</button>
        <button type="button" class="sp-button" data-scenario="resistant">高宿主抗性</button>
      </div>

      <button type="button" class="sp-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '关系标注：显示' : '关系标注：隐藏'}
      </button>

      <div class="sp-status">
        <div class="sp-card">
          <b data-a-effect></b>
          <span data-a-card-label>种群甲净效应</span>
        </div>

        <div class="sp-card">
          <b data-b-effect></b>
          <span data-b-card-label>种群乙净效应</span>
        </div>

        <div class="sp-card">
          <b data-relation-sign></b>
          <span>关系符号</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="共生与寄生种间关系互动模型"
      >
        <defs>
          <linearGradient id="${rootId}-scene" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#F0FDF4"/>
            <stop offset="100%" stop-color="#F5F3FF"/>
          </linearGradient>

          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#4C1D95" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text x="22" y="34" data-title font-size="25" font-weight="900" fill="#5B21B6"></text>
        <text x="22" y="62" data-summary font-size="13" font-weight="800" fill="#475569"></text>

        <g filter="url(#${rootId}-shadow)">
          <rect x="22" y="79" width="400" height="184" rx="21" fill="url(#${rootId}-scene)" stroke="#C4B5FD" stroke-width="3"/>
        </g>

        <g data-scene-layer></g>
        <g data-relation-layer></g>

        <g transform="translate(442 79)">
          <rect width="294" height="184" rx="21" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>

          <text x="147" y="26" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">双方相对状态</text>

          <text x="15" y="57" data-a-panel-label font-size="10.5" font-weight="800" fill="#64748B">种群甲</text>
          <rect x="93" y="48" width="177" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-a-bar x="93" y="48" width="0" height="12" rx="6" fill="#22C55E"/>

          <text x="15" y="91" data-b-panel-label font-size="10.5" font-weight="800" fill="#64748B">种群乙</text>
          <rect x="93" y="82" width="177" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-b-bar x="93" y="82" width="0" height="12" rx="6" fill="#8B5CF6"/>

          <text x="15" y="125" data-factor-label font-size="10.5" font-weight="800" fill="#64748B">关系收益</text>
          <rect x="93" y="116" width="177" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-factor-bar x="93" y="116" width="0" height="12" rx="6" fill="#0EA5E9"/>

          <text x="147" y="154" data-panel-note text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6"></text>
          <text x="147" y="174" data-context-note text-anchor="middle" font-size="9.8" font-weight="800" fill="#64748B"></text>
        </g>

        <text x="22" y="292" data-chart-title font-size="13" font-weight="900" fill="#334155"></text>

        <line x1="62" y1="389" x2="704" y2="389" stroke="#64748B" stroke-width="2.5"/>
        <line x1="62" y1="389" x2="62" y2="307" stroke="#64748B" stroke-width="2.5"/>

        <text x="708" y="393" font-size="10.5" font-weight="800" fill="#64748B">时间</text>
        <text x="18" y="316" font-size="10.5" font-weight="800" fill="#64748B">状态</text>

        <g data-grid></g>

        <path data-a-area fill="#86EFAC" opacity=".22"></path>
        <path data-b-area fill="#C4B5FD" opacity=".2"></path>

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
          stroke="#7C3AED"
          stroke-width="5"
          stroke-linecap="round"
          stroke-linejoin="round"
        ></path>

        <g data-points></g>

        <g data-label-layer>
          <g transform="translate(82 414)">
            <circle cx="7" cy="0" r="6" fill="#16A34A"/>
            <text x="20" y="5" data-a-legend font-size="11" font-weight="900" fill="#475569"></text>
          </g>

          <g transform="translate(260 414)">
            <circle cx="7" cy="0" r="6" fill="#7C3AED"/>
            <text x="20" y="5" data-b-legend font-size="11" font-weight="900" fill="#475569"></text>
          </g>

          <text x="485" y="418" data-stage-note font-size="11" font-weight="900" fill="#5B21B6"></text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var resource=root.querySelector('[data-resource]');
    var interaction=root.querySelector('[data-interaction]');
    var dependency=root.querySelector('[data-dependency]');
    var parasite=root.querySelector('[data-parasite]');
    var resistance=root.querySelector('[data-resistance]');
    var time=root.querySelector('[data-time]');

    var resourceValue=root.querySelector('[data-resource-value]');
    var interactionValue=root.querySelector('[data-interaction-value]');
    var dependencyValue=root.querySelector('[data-dependency-value]');
    var parasiteValue=root.querySelector('[data-parasite-value]');
    var resistanceValue=root.querySelector('[data-resistance-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var aEffectText=root.querySelector('[data-a-effect]');
    var bEffectText=root.querySelector('[data-b-effect]');
    var relationSignText=root.querySelector('[data-relation-sign]');
    var aCardLabel=root.querySelector('[data-a-card-label]');
    var bCardLabel=root.querySelector('[data-b-card-label]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var sceneLayer=root.querySelector('[data-scene-layer]');
    var relationLayer=root.querySelector('[data-relation-layer]');

    var aPanelLabel=root.querySelector('[data-a-panel-label]');
    var bPanelLabel=root.querySelector('[data-b-panel-label]');
    var factorLabel=root.querySelector('[data-factor-label]');
    var aBar=root.querySelector('[data-a-bar]');
    var bBar=root.querySelector('[data-b-bar]');
    var factorBar=root.querySelector('[data-factor-bar]');
    var panelNote=root.querySelector('[data-panel-note]');
    var contextNote=root.querySelector('[data-context-note]');

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

    var mode='mutualism';
    var showLabels=${showLabels ? 'true' : 'false'};
    var steps=32;

    var scenarios={
      rootNodule:{
        mode:'mutualism',
        resource:56,
        interaction:96,
        dependency:78,
        parasite:8,
        resistance:62,
        time:74
      },
      pollination:{
        mode:'mutualism',
        resource:76,
        interaction:84,
        dependency:48,
        parasite:8,
        resistance:62,
        time:68
      },
      epiphyte:{
        mode:'commensalism',
        resource:70,
        interaction:72,
        dependency:32,
        parasite:8,
        resistance:62,
        time:72
      },
      infection:{
        mode:'parasitism',
        resource:68,
        interaction:86,
        dependency:55,
        parasite:94,
        resistance:24,
        time:80
      },
      resistant:{
        mode:'parasitism',
        resource:68,
        interaction:66,
        dependency:42,
        parasite:72,
        resistance:94,
        time:80
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function signedEffect(value){
      if(value>6){
        return '+';
      }

      if(value<-6){
        return '−';
      }

      return '0';
    }

    function signedNumber(value){
      if(Math.abs(value)<.5){
        return '0';
      }

      return (value>0?'+':'')
        +value.toFixed(0);
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

    function calculateState(
      relationshipMode,
      resourceLevel,
      interactionLevel,
      dependencyLevel,
      parasiteLevel,
      resistanceLevel,
      progress
    ){
      var resourceFactor=
        resourceLevel/(resourceLevel+28);

      var interactionFactor=
        interactionLevel/100;

      var dependencyFactor=
        dependencyLevel/100;

      var parasiteFactor=
        parasiteLevel/100;

      var resistanceFactor=
        resistanceLevel/100;

      var timeFactor=clamp(
        progress,
        0,
        1
      );

      var baselineA=
        30+52*resourceFactor;

      var baselineB=
        27+47*resourceFactor;

      var a=baselineA;
      var b=baselineB;
      var effectA=0;
      var effectB=0;
      var factor=0;

      if(relationshipMode==='mutualism'){
        var exchangeBenefitA=
          51
          *interactionFactor
          *(.42+.58*dependencyFactor)
          *timeFactor;

        var exchangeBenefitB=
          47
          *interactionFactor
          *(.48+.52*dependencyFactor)
          *timeFactor;

        var exchangeCostA=
          10
          *interactionFactor
          *(1-resourceFactor)
          *timeFactor;

        var exchangeCostB=
          9
          *interactionFactor
          *(1-resourceFactor)
          *timeFactor;

        var missingPartnerCost=
          31
          *dependencyFactor
          *(1-interactionFactor)
          *timeFactor;

        effectA=
          exchangeBenefitA
          -exchangeCostA
          -missingPartnerCost*.55;

        effectB=
          exchangeBenefitB
          -exchangeCostB
          -missingPartnerCost*.48;

        a=clamp(
          baselineA+effectA,
          0,
          100
        );

        b=clamp(
          baselineB+effectB,
          0,
          100
        );

        factor=clamp(
          (
            exchangeBenefitA
            +exchangeBenefitB
          )/1.05,
          0,
          100
        );
      }else if(relationshipMode==='commensalism'){
        var commensalBenefit=
          58
          *interactionFactor
          *(.5+.5*resourceFactor)
          *timeFactor;

        var smallHostCost=
          5
          *interactionFactor
          *dependencyFactor
          *(1-resourceFactor)
          *timeFactor;

        var smallHostBenefit=
          2.5
          *interactionFactor
          *resourceFactor
          *timeFactor;

        effectA=
          smallHostBenefit
          -smallHostCost;

        effectB=
          commensalBenefit;

        a=clamp(
          baselineA+effectA,
          0,
          100
        );

        b=clamp(
          baselineB+effectB,
          0,
          100
        );

        factor=clamp(
          commensalBenefit,
          0,
          100
        );
      }else{
        var effectiveExploitation=
          interactionFactor
          *parasiteFactor
          *(1-.72*resistanceFactor);

        var hostDamage=
          71
          *effectiveExploitation
          *timeFactor;

        var defenseCost=
          17
          *resistanceFactor
          *parasiteFactor
          *timeFactor;

        var parasiteBenefit=
          69
          *effectiveExploitation
          *(.55+.45*baselineA/100)
          *timeFactor;

        var resistanceLoss=
          27
          *resistanceFactor
          *parasiteFactor
          *timeFactor;

        effectA=
          -hostDamage
          -defenseCost;

        effectB=
          parasiteBenefit
          -resistanceLoss;

        a=clamp(
          baselineA+effectA,
          0,
          100
        );

        b=clamp(
          12
          +20*resourceFactor
          +effectB,
          0,
          100
        );

        factor=clamp(
          effectiveExploitation*100,
          0,
          100
        );
      }

      return {
        a:a,
        b:b,
        effectA:effectA,
        effectB:effectB,
        factor:factor,
        baselineA:baselineA,
        baselineB:baselineB
      };
    }

    function calculateRecords(
      relationshipMode,
      resourceLevel,
      interactionLevel,
      dependencyLevel,
      parasiteLevel,
      resistanceLevel,
      timeLevel
    ){
      var records=[];
      var finalTime=
        clamp(
          .12+.88*timeLevel/100,
          .12,
          1
        );

      for(var i=0;i<=steps;i++){
        var progress=
          finalTime*i/steps;

        var state=calculateState(
          relationshipMode,
          resourceLevel,
          interactionLevel,
          dependencyLevel,
          parasiteLevel,
          resistanceLevel,
          progress
        );

        records.push({
          a:state.a,
          b:state.b,
          effectA:state.effectA,
          effectB:state.effectB,
          factor:state.factor
        });
      }

      return records;
    }

    function buildGrid(){
      var html='';
      var left=62;
      var right=704;
      var top=307;
      var bottom=389;
      var width=right-left;
      var height=bottom-top;

      for(var yIndex=0;yIndex<=4;yIndex++){
        var y=
          bottom-height*yIndex/4;

        var value=
          yIndex*25;

        html+='<line x1="'+left+'" y1="'+y
          +'" x2="'+right+'" y2="'+y
          +'" stroke="#E2E8F0" stroke-width="1.2"/>';

        html+='<text x="'+(left-8)+'" y="'+(y+4)
          +'" text-anchor="end" font-size="9.5"'
          +' font-weight="700" fill="#64748B">'
          +value
          +'</text>';
      }

      for(var xIndex=0;xIndex<=6;xIndex++){
        var x=
          left+width*xIndex/6;

        var index=
          Math.round(
            steps*xIndex/6
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

    function curveData(
      records,
      key,
      color
    ){
      var left=62;
      var right=704;
      var top=307;
      var bottom=389;
      var width=right-left;
      var height=bottom-top;
      var path='';
      var pointHTML='';

      function x(index){
        return left
          +width*index/(records.length-1);
      }

      function y(value){
        return bottom
          -height*clamp(
            value/100,
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
            +'" cy="'+py+'" r="3.7"'
            +' fill="#FFFFFF" stroke="'+color+'"'
            +' stroke-width="2.3"/>';
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

    function buildMutualismScene(
      aLevel,
      bLevel
    ){
      var noduleCount=clamp(
        Math.floor(2+bLevel/18),
        2,
        8
      );

      var html=''
        +'<g class="sp-a">'
        +'<path d="M116 224 V125"'
        +' stroke="#15803D" stroke-width="16"'
        +' stroke-linecap="round"/>'
        +'<path d="M116 151 C78 120 48 131 31 159'
        +' C67 181 94 176 116 151Z"'
        +' fill="#4ADE80" stroke="#15803D"'
        +' stroke-width="3"/>'
        +'<path d="M116 140 C151 113 185 124 204 152'
        +' C171 176 141 171 116 140Z"'
        +' fill="#22C55E" stroke="#15803D"'
        +' stroke-width="3"/>'
        +'<path d="M116 222 C91 235 74 246 58 258'
        +' M116 222 C142 237 160 247 180 258"'
        +' fill="none" stroke="#92400E"'
        +' stroke-width="7" stroke-linecap="round"/>'
        +'</g>';

      for(var i=0;i<noduleCount;i++){
        var x=72+(i%4)*29;
        var y=238+Math.floor(i/4)*17+(i%2)*5;

        html+='<circle class="sp-b" cx="'+x
          +'" cy="'+y+'" r="'+(5+i%2)
          +'" fill="#C4B5FD" stroke="#6D28D9"'
          +' stroke-width="2"/>';
      }

      html+='<g class="sp-b">'
        +'<circle cx="299" cy="162" r="43"'
        +' fill="#EDE9FE" stroke="#7C3AED"'
        +' stroke-width="4"/>'
        +'<ellipse cx="286" cy="157" rx="13" ry="8"'
        +' fill="#A78BFA" stroke="#5B21B6"'
        +' stroke-width="2"/>'
        +'<ellipse cx="311" cy="174" rx="14" ry="8"'
        +' fill="#8B5CF6" stroke="#5B21B6"'
        +' stroke-width="2"/>'
        +'<ellipse cx="309" cy="143" rx="11" ry="7"'
        +' fill="#C4B5FD" stroke="#5B21B6"'
        +' stroke-width="2"/>'
        +'<text x="299" y="218" text-anchor="middle"'
        +' font-size="12" font-weight="900" fill="#5B21B6">'
        +'共生伙伴'
        +'</text>'
        +'</g>';

      return html;
    }

    function buildCommensalismScene(
      aLevel,
      bLevel
    ){
      var epiphyteCount=clamp(
        Math.floor(2+bLevel/20),
        2,
        7
      );

      var html=''
        +'<g class="sp-a">'
        +'<path d="M153 237 V125"'
        +' stroke="#78350F" stroke-width="22"'
        +' stroke-linecap="round"/>'
        +'<circle cx="153" cy="116" r="61"'
        +' fill="#22C55E" stroke="#166534"'
        +' stroke-width="4"/>'
        +'<circle cx="113" cy="130" r="40"'
        +' fill="#4ADE80" stroke="#166534"'
        +' stroke-width="3"/>'
        +'<circle cx="195" cy="132" r="42"'
        +' fill="#16A34A" stroke="#166534"'
        +' stroke-width="3"/>'
        +'</g>';

      for(var i=0;i<epiphyteCount;i++){
        var angle=
          -120+i*37;

        var radians=
          angle*Math.PI/180;

        var x=
          153+Math.cos(radians)*61;

        var y=
          125+Math.sin(radians)*45;

        html+='<g class="sp-b"'
          +' transform="translate('+x.toFixed(1)
          +' '+y.toFixed(1)+') rotate('+angle+')">'
          +'<path d="M0 0 Q10 -16 19 -4"'
          +' fill="none" stroke="#7C3AED"'
          +' stroke-width="4" stroke-linecap="round"/>'
          +'<ellipse cx="10" cy="-9" rx="9" ry="4"'
          +' fill="#C4B5FD" stroke="#6D28D9"'
          +' stroke-width="1.5"/>'
          +'</g>';
      }

      html+='<text x="153" y="255" text-anchor="middle"'
        +' font-size="12" font-weight="900" fill="#166534">'
        +'支持植物'
        +'</text>'
        +'<text x="307" y="183" text-anchor="middle"'
        +' font-size="12" font-weight="900" fill="#6D28D9">'
        +'附生者获得空间与光照条件'
        +'</text>';

      return html;
    }

    function buildParasitismScene(
      hostLevel,
      parasiteLevel
    ){
      var parasiteCount=clamp(
        Math.floor(1+parasiteLevel/15),
        1,
        8
      );

      var hostOpacity=
        .35+.65*hostLevel/100;

      var html=''
        +'<g class="sp-a" opacity="'+hostOpacity+'">'
        +'<ellipse cx="144" cy="165" rx="83" ry="54"'
        +' fill="#FDE68A" stroke="#D97706"'
        +' stroke-width="4"/>'
        +'<circle cx="211" cy="153" r="31"'
        +' fill="#FACC15" stroke="#D97706"'
        +' stroke-width="4"/>'
        +'<ellipse cx="222" cy="129" rx="8" ry="18"'
        +' fill="#FDE68A" stroke="#D97706"'
        +' stroke-width="3"/>'
        +'<circle cx="218" cy="149" r="3" fill="#111827"/>'
        +'<path d="M104 208 L91 232 M143 218 L137 242'
        +' M177 213 L188 237"'
        +' stroke="#92400E" stroke-width="6"'
        +' stroke-linecap="round"/>'
        +'<text x="146" y="254" text-anchor="middle"'
        +' font-size="12" font-weight="900" fill="#92400E">'
        +'宿主'
        +'</text>'
        +'</g>';

      for(var i=0;i<parasiteCount;i++){
        var x=95+(i%5)*31;
        var y=145+Math.floor(i/5)*35+(i%2)*16;

        html+='<g class="sp-b"'
          +' transform="translate('+x+' '+y+')">'
          +'<ellipse cx="0" cy="0" rx="8" ry="5"'
          +' fill="#FCA5A5" stroke="#B91C1C"'
          +' stroke-width="2"/>'
          +'<path d="M-7 -4 L-12 -9 M-7 4 L-12 9'
          +' M7 -4 L12 -9 M7 4 L12 9"'
          +' stroke="#B91C1C" stroke-width="1.5"/>'
          +'</g>';
      }

      html+='<g class="sp-b">'
        +'<circle cx="315" cy="166" r="39"'
        +' fill="#FEE2E2" stroke="#DC2626"'
        +' stroke-width="4"/>'
        +'<text x="315" y="160" text-anchor="middle"'
        +' font-size="23">🦠</text>'
        +'<text x="315" y="187" text-anchor="middle"'
        +' font-size="12" font-weight="900" fill="#B91C1C">'
        +'寄生者'
        +'</text>'
        +'</g>';

      return html;
    }

    function buildMutualismRelations(){
      return ''
        +'<path class="sp-flow" d="M205 131'
        +' C235 104 266 104 286 124"'
        +' fill="none" stroke="#16A34A"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-green)"/>'
        +'<text x="246" y="102" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#166534">'
        +'有机物或栖息条件'
        +'</text>'
        +'<path class="sp-flow" d="M285 204'
        +' C252 229 211 229 180 207"'
        +' fill="none" stroke="#7C3AED"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="237" y="242" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#5B21B6">'
        +'矿质营养、传粉或其他服务'
        +'</text>';
    }

    function buildCommensalismRelations(){
      return ''
        +'<path class="sp-flow" d="M216 142'
        +' C251 127 279 137 298 158"'
        +' fill="none" stroke="#0284C7"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="269" y="123" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#0369A1">'
        +'一方获得空间、运输或残余资源'
        +'</text>'
        +'<path d="M217 204 C250 217 280 213 301 193"'
        +' fill="none" stroke="#94A3B8"'
        +' stroke-width="3" stroke-dasharray="5 6"/>'
        +'<text x="270" y="230" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#64748B">'
        +'另一方净影响接近0'
        +'</text>';
    }

    function buildParasitismRelations(){
      return ''
        +'<path class="sp-flow" d="M221 126'
        +' C251 105 284 111 303 134"'
        +' fill="none" stroke="#F59E0B"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<text x="265" y="104" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#B45309">'
        +'宿主资源被寄生者利用'
        +'</text>'
        +'<path class="sp-flow" d="M302 204'
        +' C271 230 228 229 196 206"'
        +' fill="none" stroke="#DC2626"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-red)"/>'
        +'<text x="251" y="244" text-anchor="middle"'
        +' font-size="10.5" font-weight="900" fill="#B91C1C">'
        +'寄生使宿主生长或繁殖受损'
        +'</text>';
    }

    function classifyMutualism(
      effectA,
      effectB,
      dependencyLevel,
      interactionLevel
    ){
      if(effectA>12 && effectB>12){
        return dependencyLevel>70
          ?'强互利依赖'
          :'双方净受益';
      }

      if(
        dependencyLevel>70
        &&interactionLevel<30
      ){
        return '伙伴不足受限';
      }

      if(effectA>2 && effectB>2){
        return '互利作用较弱';
      }

      return '净收益不明显';
    }

    function classifyCommensalism(
      effectA,
      effectB
    ){
      if(effectB>12 && Math.abs(effectA)<=6){
        return '典型+/0';
      }

      if(effectB>5 && effectA<-6){
        return '宿主出现成本';
      }

      if(effectB>5){
        return '偏利收益较弱';
      }

      return '关系影响较小';
    }

    function classifyParasitism(
      effectA,
      effectB,
      hostLevel
    ){
      if(hostLevel<22){
        return '宿主严重受损';
      }

      if(effectB<=0){
        return '寄生受到抑制';
      }

      if(effectA<-20 && effectB>8){
        return '典型+/−';
      }

      return '寄生作用较弱';
    }

    function update(){
      var R=Number(resource.value);
      var I=Number(interaction.value);
      var D=Number(dependency.value);
      var P=Number(parasite.value);
      var H=Number(resistance.value);
      var T=Number(time.value);

      resourceValue.textContent=R.toFixed(0)+'%';
      interactionValue.textContent=I.toFixed(0)+'%';
      dependencyValue.textContent=D.toFixed(0)+'%';
      parasiteValue.textContent=P.toFixed(0)+'%';
      resistanceValue.textContent=H.toFixed(0)+'%';
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

      var records=calculateRecords(
        mode,
        R,
        I,
        D,
        P,
        H,
        T
      );

      var finalState=
        records[
          records.length-1
        ];

      grid.innerHTML=
        buildGrid();

      var aData=
        curveData(
          records,
          'a',
          '#16A34A'
        );

      var bData=
        curveData(
          records,
          'b',
          '#7C3AED'
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

      aBar.setAttribute(
        'width',
        String(
          177*finalState.a/100
        )
      );

      bBar.setAttribute(
        'width',
        String(
          177*finalState.b/100
        )
      );

      factorBar.setAttribute(
        'width',
        String(
          177*finalState.factor/100
        )
      );

      aEffectText.textContent=
        signedNumber(
          finalState.effectA
        );

      bEffectText.textContent=
        signedNumber(
          finalState.effectB
        );

      var signA=
        signedEffect(
          finalState.effectA
        );

      var signB=
        signedEffect(
          finalState.effectB
        );

      var relationSign=
        signA+'/'+signB;

      relationSignText.textContent=
        relationSign;

      var relationshipState='';
      var explanation='';
      var conditionNote='';

      if(mode==='mutualism'){
        relationshipState=
          classifyMutualism(
            finalState.effectA,
            finalState.effectB,
            D,
            I
          );

        title.textContent=
          '互利共生：双方获得净收益';

        summary.textContent=
          '双方交换资源或生态服务，但维持互利关系本身也可能需要成本';

        chartTitle.textContent=
          '互利伙伴双方的相对状态随时间变化';

        aCardLabel.textContent=
          '伙伴甲净效应';

        bCardLabel.textContent=
          '伙伴乙净效应';

        aPanelLabel.textContent=
          '伙伴甲状态';

        bPanelLabel.textContent=
          '伙伴乙状态';

        factorLabel.textContent=
          '互利交换收益';

        aLegend.textContent=
          '互利伙伴甲';

        bLegend.textContent=
          '互利伙伴乙';

        panelNote.textContent=
          relationshipState;

        contextNote.textContent=
          '通常简化为+/+';

        stageNote.textContent=
          D>70
            ?'较高依赖使伙伴缺失的代价增大'
            :'兼性互利中双方可保留一定独立生活能力';

        sceneLayer.innerHTML=
          buildMutualismScene(
            finalState.a,
            finalState.b
          );

        relationLayer.innerHTML=
          showLabels
            ?buildMutualismRelations()
            :'';

        explanation=
          '互利共生中，双方通过营养、传粉、保护、栖息空间或其他生态服务获得净收益，因此常表示为“+/+”。';

        if(I<20 && D>70){
          conditionNote=
            '当前双方依赖较高，但实际相互作用很弱，伙伴服务不足使双方状态受到限制。';
        }else if(R<28 && I>72){
          conditionNote=
            '环境资源较少时，交换成本相对增加，即使关系较强，互利净收益也可能下降。';
        }else if(I>75 && finalState.effectA>10 && finalState.effectB>10){
          conditionNote=
            '当前双方交换较充分，两个伙伴相对于缺少关系时都获得明显净收益。';
        }else if(D<30){
          conditionNote=
            '当前依赖程度较低，关系更接近兼性互利，伙伴分离后仍可能保持一定生存能力。';
        }else{
          conditionNote=
            '当前双方存在互利收益，但净效果仍受到资源供应、交换强度和依赖程度影响。';
        }
      }else if(mode==='commensalism'){
        relationshipState=
          classifyCommensalism(
            finalState.effectA,
            finalState.effectB
          );

        title.textContent=
          '偏利共生：一方受益，另一方净影响较小';

        summary.textContent=
          '受益方利用另一方提供的空间、运输或残余资源，宿主净效应近似为0';

        chartTitle.textContent=
          '支持方与受益方的相对状态随时间变化';

        aCardLabel.textContent=
          '支持方净效应';

        bCardLabel.textContent=
          '受益方净效应';

        aPanelLabel.textContent=
          '支持方状态';

        bPanelLabel.textContent=
          '受益方状态';

        factorLabel.textContent=
          '受益方获得资源';

        aLegend.textContent=
          '支持方';

        bLegend.textContent=
          '受益方';

        panelNote.textContent=
          relationshipState;

        contextNote.textContent=
          '通常简化为+/0';

        stageNote.textContent=
          Math.abs(finalState.effectA)<=6
            ?'支持方净效应接近0'
            :'资源紧张时支持方可能出现轻微成本';

        sceneLayer.innerHTML=
          buildCommensalismScene(
            finalState.a,
            finalState.b
          );

        relationLayer.innerHTML=
          showLabels
            ?buildCommensalismRelations()
            :'';

        explanation=
          '偏利共生中一方获得空间、运输、庇护或残余资源，另一方在当前观察尺度上的净影响较小，常简化表示为“+/0”。';

        if(R<25 && I>75){
          conditionNote=
            '环境资源紧张且作用较强时，支持方可能出现可观察成本，此时关系未必仍是严格的+/0。';
        }else if(I<18){
          conditionNote=
            '当前种间接触较弱，受益方获得的空间或资源有限，关系效应不明显。';
        }else if(
          finalState.effectB>12
          &&Math.abs(finalState.effectA)<=6
        ){
          conditionNote=
            '当前受益方获得明显收益，而支持方净变化较小，符合偏利共生的简化特征。';
        }else{
          conditionNote=
            '当前一方获得一定收益，但另一方也可能存在轻微、难以观察的成本或收益。';
        }
      }else{
        relationshipState=
          classifyParasitism(
            finalState.effectA,
            finalState.effectB,
            finalState.a
          );

        title.textContent=
          '寄生关系：寄生者受益，宿主受到损害';

        summary.textContent=
          '寄生者持续利用宿主营养或组织，宿主防御可降低寄生成功率但也需要资源';

        chartTitle.textContent=
          '宿主与寄生者的相对状态随时间变化';

        aCardLabel.textContent=
          '宿主净效应';

        bCardLabel.textContent=
          '寄生者净效应';

        aPanelLabel.textContent=
          '宿主状态';

        bPanelLabel.textContent=
          '寄生者状态';

        factorLabel.textContent=
          '有效寄生强度';

        aLegend.textContent=
          '宿主';

        bLegend.textContent=
          '寄生者';

        panelNote.textContent=
          relationshipState;

        contextNote.textContent=
          '通常简化为+/−';

        stageNote.textContent=
          H>75
            ?'宿主抗性降低寄生者成功率'
            :P>75
              ?'高寄生负荷使宿主损害加重'
              :'宿主与寄生者持续相互作用';

        sceneLayer.innerHTML=
          buildParasitismScene(
            finalState.a,
            finalState.b
          );

        relationLayer.innerHTML=
          showLabels
            ?buildParasitismRelations()
            :'';

        explanation=
          '寄生者从宿主获得营养、空间或繁殖条件，宿主的生长、繁殖或存活受到损害，因此常表示为“+/−”。';

        if(P>78 && H<30){
          conditionNote=
            '寄生负荷较高且宿主抗性较弱，寄生者获得较多资源，宿主状态明显下降。';
        }else if(H>82){
          conditionNote=
            '宿主防御与抗性较强，有效寄生强度下降；但维持防御也会产生一定资源成本。';
        }else if(I<18 || P<15){
          conditionNote=
            '当前寄生者负荷或作用强度很低，宿主损害和寄生者收益都不明显。';
        }else if(finalState.b<=20){
          conditionNote=
            '宿主抗性和寄生压力的组合使寄生者增长受到明显限制。';
        }else{
          conditionNote=
            '当前寄生者获得一定收益，宿主承担持续损害，但这不等同于捕食者立即杀死猎物。';
        }
      }

      aEffectText.style.color=
        finalState.effectA>6
          ?'#059669'
          :finalState.effectA<-6
            ?'#DC2626'
            :'#64748B';

      bEffectText.style.color=
        finalState.effectB>6
          ?'#059669'
          :finalState.effectB<-6
            ?'#DC2626'
            :'#64748B';

      relationSignText.style.color=
        mode==='parasitism'
          ?'#B91C1C'
          :mode==='mutualism'
            ?'#047857'
            :'#475569';

      root.style.setProperty(
        '--sp-flow-speed',
        clamp(
          2.5-I/62,
          .52,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--sp-a-speed',
        clamp(
          2.5-finalState.a/72,
          .62,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--sp-b-speed',
        clamp(
          2.7-finalState.b/68,
          .65,
          2.7
        ).toFixed(2)+'s'
      );

      var timeNote=T<20
        ?'作用时间较短，当前净效应尚未充分积累。'
        :'作用时间增加会放大当前关系的累计结果，但不会使关系类型永久固定。';

      var signNote=
        ' 当前模型根据净效应计算关系符号为“'
        +relationSign
        +'”。';

      result.innerHTML=
        explanation
        +'<br>'+conditionNote
        +' '+timeNote
        +signNote
        +' “0”表示当前尺度上净效应较小，不代表双方完全没有相互作用。所有数值均为相对教学指标。';
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
        interaction.value=String(data.interaction);
        dependency.value=String(data.dependency);
        parasite.value=String(data.parasite);
        resistance.value=String(data.resistance);
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

    interaction.oninput=function(){
      setScenarioActive('');
      update();
    };

    dependency.oninput=function(){
      setScenarioActive('');
      update();
    };

    parasite.oninput=function(){
      setScenarioActive('');
      update();
    };

    resistance.oninput=function(){
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
