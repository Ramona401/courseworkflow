/**
 * lifeScienceLabTemplatesPlantSeedGermination.ts
 *
 * 平面生命科学实验室：种子萌发条件。
 *
 * 教学目标：
 * 1. 区分“萌发率”和“萌发速度”两个不同指标；
 * 2. 观察水分、适宜温度和充足氧气对种子萌发的共同影响；
 * 3. 观察吸胀、种皮破裂、胚根伸出、胚芽生长和幼苗建立等阶段；
 * 4. 比较缺水、缺氧、低温和高温等典型情境；
 * 5. 明确光照并不是所有种子萌发的普遍必要条件，不同种类种子对光的反应可能不同。
 *
 * 科学边界：
 * - 所有数值均为相对教学指标，不代表某一真实物种的精确实验数据；
 * - 本模型只演示主要环境条件，不替代真实种子萌发实验；
 * - “普通种子、需光种子、避光种子”是用于比较光反应差异的教学模式。
 */
import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'
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
function bool(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]
  return typeof value === 'boolean' ? value : fallback
}
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}
/**
 * 使用生命科学实验室统一的 .bl-* 类名。
 * 这样模板被嵌入课件后，可以自动应用“上方实验主体 + 底部课堂控制条”的公共布局覆盖层。
 */
function seedGerminationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BBF7D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#FEFCE8);border-bottom:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:244px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FAFFFB;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#22C55E}'
    + '#' + rootId + ' .sg-subtitle{margin:5px 0 6px;font-size:11.5px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .sg-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px}'
    + '#' + rootId + ' .sg-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px}'
    + '#' + rootId + ' .sg-button{min-height:29px;padding:3px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:9.5px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .sg-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.13)}'
    + '#' + rootId + ' .sg-auto{width:100%;height:31px;border:0;border-radius:8px;background:linear-gradient(135deg,#4ADE80,#16A34A);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .sg-auto.paused{background:#64748B}'
    + '#' + rootId + ' .sg-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:8px 0}'
    + '#' + rootId + ' .sg-card{padding:6px 3px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .sg-card b{display:block;font-size:14px;color:#15803D}'
    + '#' + rootId + ' .sg-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .sg-pulse{animation:' + rootId + '-pulse var(--sg-speed,1.8s) ease-in-out infinite}'
    + '#' + rootId + ' .sg-drift{animation:' + rootId + '-drift 2.4s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.45}50%{opacity:1}}'
    + '@keyframes ' + rootId + '-drift{from{transform:translateY(0)}to{transform:translateY(-7px)}}'
    + '</style>'
}
const SCRIPT_END = '</' + 'script>'
export const LIFE_SCIENCE_LAB_TEMPLATES_PLANT_SEED_GERMINATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-seed-germination',
    group: '🌱 植物生长与发育',
    name: '种子萌发条件',
    emoji: '🌱',
    desc: '调节温度、水分、氧气、光照和观察时间，比较萌发率、萌发速度与发育阶段',
    params: [
      {
        key: 'temperature',
        label: '环境温度/℃',
        type: 'number',
        min: 5,
        max: 40,
        step: 1,
        defaultValue: 24,
      },
      {
        key: 'water',
        label: '水分供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 75,
      },
      {
        key: 'oxygen',
        label: '氧气供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 85,
      },
      {
        key: 'light',
        label: '光照强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 50,
      },
      {
        key: 'observationTime',
        label: '观察时间/小时',
        type: 'number',
        min: 0,
        max: 120,
        step: 2,
        defaultValue: 48,
      },
      {
        key: 'lightSensitive',
        label: '初始使用需光种子',
        type: 'boolean',
        defaultValue: false,
        hint: '关闭时默认使用普通种子；组件内仍可切换需光种子和避光种子',
      },
    ],
    buildHTML: (params, rootId) => {
      const temperature = num(params, 'temperature', 24)
      const water = num(params, 'water', 75)
      const oxygen = num(params, 'oxygen', 85)
      const light = num(params, 'light', 50)
      const observationTime = num(params, 'observationTime', 48)
      const lightSensitive = bool(params, 'lightSensitive', false)
      return `
<div id="${rootId}">
${seedGerminationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌱 种子萌发条件与发育阶段</div>
    <div class="bl-note">相对教学模型：萌发率 ≠ 萌发速度</div>
  </div>
  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label"><span>环境温度</span><span class="bl-value" data-temperature-value></span></div>
        <input data-temperature type="range" min="5" max="40" step="1" value="${n(temperature)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>水分供应</span><span class="bl-value" data-water-value></span></div>
        <input data-water type="range" min="0" max="100" step="1" value="${n(water)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>氧气供应</span><span class="bl-value" data-oxygen-value></span></div>
        <input data-oxygen type="range" min="0" max="100" step="1" value="${n(oxygen)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>光照强度</span><span class="bl-value" data-light-value></span></div>
        <input data-light type="range" min="0" max="100" step="1" value="${n(light)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>观察时间</span><span class="bl-value" data-time-value></span></div>
        <input data-time type="range" min="0" max="120" step="2" value="${n(observationTime)}">
      </div>
      <div class="sg-subtitle">种子光反应类型</div>
      <div class="sg-buttons" data-mode-buttons>
        <button type="button" class="sg-button${lightSensitive ? '' : ' active'}" data-mode="common">普通种子</button>
        <button type="button" class="sg-button${lightSensitive ? ' active' : ''}" data-mode="positive">需光种子</button>
        <button type="button" class="sg-button" data-mode="negative">避光种子</button>
      </div>
      <div class="sg-subtitle">快速比较情境</div>
      <div class="sg-scenarios" data-scenario-buttons>
        <button type="button" class="sg-button active" data-scenario="standard">适宜</button>
        <button type="button" class="sg-button" data-scenario="dry">缺水</button>
        <button type="button" class="sg-button" data-scenario="hypoxia">缺氧</button>
        <button type="button" class="sg-button" data-scenario="cold">低温</button>
        <button type="button" class="sg-button" data-scenario="hot">高温</button>
      </div>
      <div class="bl-row" style="margin-top:7px">
        <button type="button" class="sg-auto" data-auto>时间推进：运行中</button>
      </div>
      <div class="sg-status">
        <div class="sg-card"><b data-rate></b><span>群体萌发率</span></div>
        <div class="sg-card"><b data-speed></b><span>萌发速度指数</span></div>
        <div class="sg-card"><b data-stage-card></b><span>当前发育阶段</span></div>
      </div>
      <div class="bl-result" data-result></div>
    </div>
    <div class="bl-stage">
      <svg viewBox="0 0 760 430" aria-label="种子萌发条件互动示意图">
        <defs>
          <linearGradient id="${rootId}-soil" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#D6B47C"/>
            <stop offset="100%" stop-color="#8B5A2B"/>
          </linearGradient>
          <linearGradient id="${rootId}-seed" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#FDE68A"/>
            <stop offset="100%" stop-color="#D97706"/>
          </linearGradient>
          <marker id="${rootId}-arrow" markerWidth="8" markerHeight="8" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>
          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#14532D" flood-opacity=".14"/>
          </filter>
        </defs>
        <rect width="760" height="430" fill="#FFFFFF"/>
        <text x="26" y="38" data-title font-size="27" font-weight="900" fill="#166534"></text>
        <text x="26" y="67" data-summary font-size="14" font-weight="800" fill="#475569"></text>
        <g data-sun transform="translate(68 104)">
          <circle cx="0" cy="0" r="31" fill="#FACC15" stroke="#F59E0B" stroke-width="4"/>
          <g data-light-rays></g>
        </g>
        <g filter="url(#${rootId}-shadow)">
          <rect x="40" y="132" width="438" height="238" rx="42" fill="url(#${rootId}-soil)" stroke="#7C4A1D" stroke-width="5"/>
          <rect x="40" y="132" width="438" height="58" rx="42" fill="#E8C999" opacity=".82"/>
          <path d="M40 188 H478" stroke="#A16207" stroke-width="3" stroke-dasharray="8 8" opacity=".6"/>
        </g>
        <g data-water-flow></g>
        <g data-oxygen-flow></g>
        <g data-seed transform="translate(0 0)" filter="url(#${rootId}-shadow)">
          <ellipse data-testa cx="270" cy="230" rx="69" ry="48" fill="url(#${rootId}-seed)" stroke="#92400E" stroke-width="7"/>
          <path data-cotyledon-left d="M265 203 C225 195 209 226 229 249 C244 267 263 252 270 231Z" fill="#FDE68A" stroke="#B45309" stroke-width="3"/>
          <path data-cotyledon-right d="M275 203 C315 195 331 226 311 249 C296 267 277 252 270 231Z" fill="#FCD34D" stroke="#B45309" stroke-width="3"/>
          <path data-radicle pathLength="100" d="M270 251 C273 278 258 302 264 344" fill="none" stroke="#F8FAFC" stroke-width="10" stroke-linecap="round" stroke-dasharray="100" stroke-dashoffset="100"/>
          <path data-root-tip d="M264 344 q-12 17 -2 29" fill="none" stroke="#F8FAFC" stroke-width="7" stroke-linecap="round" opacity="0"/>
          <path data-plumule pathLength="100" d="M270 211 C269 190 282 174 280 143 C281 127 289 114 301 104" fill="none" stroke="#22C55E" stroke-width="10" stroke-linecap="round" stroke-dasharray="100" stroke-dashoffset="100"/>
          <g data-leaves opacity="0">
            <ellipse cx="309" cy="100" rx="21" ry="11" fill="#4ADE80" stroke="#15803D" stroke-width="3" transform="rotate(-28 309 100)"/>
            <ellipse cx="290" cy="105" rx="21" ry="11" fill="#86EFAC" stroke="#15803D" stroke-width="3" transform="rotate(30 290 105)"/>
          </g>
          <g data-root-hairs></g>
        </g>
        <g data-labels>
          <path d="M219 212 L150 172" stroke="#92400E" stroke-width="2.5"/>
          <text x="84" y="168" font-size="13" font-weight="900" fill="#92400E">种皮</text>
          <path d="M238 240 L154 263" stroke="#B45309" stroke-width="2.5"/>
          <text x="87" y="270" font-size="13" font-weight="900" fill="#B45309">子叶</text>
          <path data-radicle-label d="M267 310 L168 327" stroke="#64748B" stroke-width="2.5" opacity="0"/>
          <text data-radicle-label x="98" y="333" font-size="13" font-weight="900" fill="#475569" opacity="0">胚根</text>
          <path data-plumule-label d="M284 158 L370 122" stroke="#15803D" stroke-width="2.5" opacity="0"/>
          <text data-plumule-label x="378" y="126" font-size="13" font-weight="900" fill="#15803D" opacity="0">胚芽</text>
        </g>
        <g transform="translate(500 102)">
          <rect x="0" y="0" width="230" height="170" rx="24" fill="#F0FDF4" stroke="#86EFAC" stroke-width="4"/>
          <text x="18" y="28" font-size="16" font-weight="900" fill="#166534">群体萌发率：12粒种子</text>
          <g data-cohort></g>
          <text x="18" y="151" data-cohort-note font-size="12" font-weight="800" fill="#475569"></text>
        </g>
        <g transform="translate(500 292)">
          <rect x="0" y="0" width="230" height="82" rx="20" fill="#FFFBEB" stroke="#FCD34D" stroke-width="4"/>
          <text x="16" y="24" font-size="14" font-weight="900" fill="#92400E">萌发进程</text>
          <line x1="18" y1="50" x2="210" y2="50" stroke="#D6D3D1" stroke-width="7" stroke-linecap="round"/>
          <line data-progress-line x1="18" y1="50" x2="18" y2="50" stroke="#22C55E" stroke-width="7" stroke-linecap="round"/>
          <circle data-progress-marker cx="18" cy="50" r="10" fill="#16A34A" stroke="#FFFFFF" stroke-width="3"/>
          <text x="16" y="72" font-size="10" font-weight="800" fill="#78716C">吸胀</text>
          <text x="75" y="72" font-size="10" font-weight="800" fill="#78716C">胚根</text>
          <text x="133" y="72" font-size="10" font-weight="800" fill="#78716C">胚芽</text>
          <text x="184" y="72" font-size="10" font-weight="800" fill="#78716C">幼苗</text>
        </g>
        <text x="40" y="401" data-stage-note font-size="17" font-weight="900" fill="#166534"></text>
        <text x="500" y="402" data-limit-note font-size="13" font-weight="800" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var temperature=root.querySelector('[data-temperature]');
    var water=root.querySelector('[data-water]');
    var oxygen=root.querySelector('[data-oxygen]');
    var light=root.querySelector('[data-light]');
    var time=root.querySelector('[data-time]');
    var temperatureValue=root.querySelector('[data-temperature-value]');
    var waterValue=root.querySelector('[data-water-value]');
    var oxygenValue=root.querySelector('[data-oxygen-value]');
    var lightValue=root.querySelector('[data-light-value]');
    var timeValue=root.querySelector('[data-time-value]');
    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var autoButton=root.querySelector('[data-auto]');
    var rateText=root.querySelector('[data-rate]');
    var speedText=root.querySelector('[data-speed]');
    var stageCard=root.querySelector('[data-stage-card]');
    var result=root.querySelector('[data-result]');
    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var lightRays=root.querySelector('[data-light-rays]');
    var waterFlow=root.querySelector('[data-water-flow]');
    var oxygenFlow=root.querySelector('[data-oxygen-flow]');
    var seed=root.querySelector('[data-seed]');
    var testa=root.querySelector('[data-testa]');
    var cotyledonLeft=root.querySelector('[data-cotyledon-left]');
    var cotyledonRight=root.querySelector('[data-cotyledon-right]');
    var radicle=root.querySelector('[data-radicle]');
    var rootTip=root.querySelector('[data-root-tip]');
    var plumule=root.querySelector('[data-plumule]');
    var leaves=root.querySelector('[data-leaves]');
    var rootHairs=root.querySelector('[data-root-hairs]');
    var radicleLabels=root.querySelectorAll('[data-radicle-label]');
    var plumuleLabels=root.querySelectorAll('[data-plumule-label]');
    var cohort=root.querySelector('[data-cohort]');
    var cohortNote=root.querySelector('[data-cohort-note]');
    var progressLine=root.querySelector('[data-progress-line]');
    var progressMarker=root.querySelector('[data-progress-marker]');
    var stageNote=root.querySelector('[data-stage-note]');
    var limitNote=root.querySelector('[data-limit-note]');
    var mode=${lightSensitive ? "'positive'" : "'common'"};
    var automatic=true;
    var timer=null;
    var stageNames=['干燥种子','吸胀启动','种皮破裂','胚根伸出','胚芽生长','幼苗建立'];
    var stageNotes=[
      '种子尚未充分吸水，代谢活动维持在较低水平。',
      '种子吸水膨胀，酶和呼吸等生命活动逐渐增强。',
      '种皮破裂，萌发进入可观察的结构变化阶段。',
      '胚根通常先伸出，逐步形成幼根并向下生长。',
      '胚芽继续生长，子叶中的储藏物质支持早期发育。',
      '幼苗逐渐建立，叶片展开后开始增强光合作用能力。'
    ];
    var scenarios={
      standard:{temperature:24,water:75,oxygen:85,light:50,time:48},
      dry:{temperature:24,water:10,oxygen:85,light:50,time:48},
      hypoxia:{temperature:24,water:85,oxygen:8,light:50,time:48},
      cold:{temperature:6,water:75,oxygen:85,light:50,time:72},
      hot:{temperature:39,water:75,oxygen:85,light:50,time:48}
    };
    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }
    function setOpacity(nodes,value){
      for(var i=0;i<nodes.length;i++){
        nodes[i].setAttribute('opacity',String(value));
      }
    }
    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }
      if(!automatic || !document.body.contains(root)){
        return;
      }
      timer=window.setTimeout(function(){
        var next=Number(time.value)+2;
        time.value=String(next>120?0:next);
        update();
        schedule();
      },760);
    }
    function resolveStage(development){
      if(development<.15)return 0;
      if(development<.5)return 1;
      if(development<1)return 2;
      if(development<1.8)return 3;
      if(development<2.8)return 4;
      return 5;
    }
    function updateScenarioActive(name){
      for(var i=0;i<scenarioButtons.length;i++){
        scenarioButtons[i].classList.toggle(
          'active',
          scenarioButtons[i].getAttribute('data-scenario')===name
        );
      }
    }
    function update(){
      var T=Number(temperature.value);
      var W=Number(water.value);
      var O=Number(oxygen.value);
      var L=Number(light.value);
      var H=Number(time.value);
      temperatureValue.textContent=T.toFixed(0)+'℃';
      waterValue.textContent=W.toFixed(0)+'%';
      oxygenValue.textContent=O.toFixed(0)+'%';
      lightValue.textContent=L.toFixed(0)+'%';
      timeValue.textContent=H.toFixed(0)+' h';
      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }
      var temperatureFactor=Math.exp(-Math.pow((T-25)/9.5,2));
      var waterFactor=W<=75
        ?W/(W+18)*1.18
        :1-(W-75)*.012;
      waterFactor=clamp(waterFactor,0,1);
      var oxygenFactor=1-Math.exp(-O/24);
      oxygenFactor=clamp(oxygenFactor,0,1);
      var lightFactor=1;
      var lightExplanation='普通种子模式下，光照不作为萌发的普遍必要条件。';
      if(mode==='positive'){
        lightFactor=.25+.75*L/100;
        lightExplanation='需光种子模式下，适度光照可促进萌发。';
      }else if(mode==='negative'){
        lightFactor=.25+.75*(1-L/100);
        lightExplanation='避光种子模式下，强光会抑制萌发，较暗条件更有利。';
      }
      var factors=[temperatureFactor,waterFactor,oxygenFactor,lightFactor];
      var factorNames=['温度','水分','氧气','光照反应'];
      var limitIndex=0;
      for(var f=1;f<factors.length;f++){
        if(factors[f]<factors[limitIndex])limitIndex=f;
      }
      var condition=Math.pow(
        clamp(temperatureFactor*waterFactor*oxygenFactor*lightFactor,0,1),
        .55
      );
      var speedIndex=100*condition;
      var maximumRate=100*clamp(.08+.92*Math.min.apply(null,factors),0,1);
      var timeConstant=18+62*(1-condition);
      var germinationRate=maximumRate*(1-Math.exp(-H/timeConstant));
      germinationRate=clamp(germinationRate,0,100);
      var development=H/18*condition;
      var stageIndex=resolveStage(development);
      var stageProgress=clamp(development/3.25,0,1);
      rateText.textContent=germinationRate.toFixed(0)+'%';
      speedText.textContent=speedIndex.toFixed(0);
      stageCard.textContent=stageNames[stageIndex];
      title.textContent=stageNames[stageIndex];
      summary.textContent=stageNotes[stageIndex];
      stageNote.textContent='当前阶段：'+stageNames[stageIndex]+'｜'+stageNotes[stageIndex];
      limitNote.textContent='主要限制：'+factorNames[limitIndex];
      root.style.setProperty(
        '--sg-speed',
        clamp(2.6-speedIndex/65,.65,2.5).toFixed(2)+'s'
      );
      var swell=1+clamp(development,0,.5)*.1;
      seed.setAttribute(
        'transform',
        'translate('+(270-270*swell)+' '+(230-230*swell)+') scale('+swell+')'
      );
      testa.setAttribute('opacity',stageIndex>=2?'.52':'1');
      testa.setAttribute('stroke-dasharray',stageIndex>=2?'18 8':'0');
      cotyledonLeft.setAttribute('opacity',stageIndex>=5?'.62':'1');
      cotyledonRight.setAttribute('opacity',stageIndex>=5?'.62':'1');
      var rootRatio=clamp((development-.78)/1.5,0,1);
      var shootRatio=clamp((development-1.55)/1.45,0,1);
      radicle.setAttribute('stroke-dashoffset',String(100-rootRatio*100));
      plumule.setAttribute('stroke-dashoffset',String(100-shootRatio*100));
      rootTip.setAttribute('opacity',rootRatio>.78?'1':'0');
      leaves.setAttribute('opacity',stageIndex>=5?'1':'0');
      leaves.classList.toggle('sg-pulse',stageIndex>=5);
      setOpacity(radicleLabels,rootRatio>.34?1:0);
      setOpacity(plumuleLabels,shootRatio>.35?1:0);
      var hairHTML='';
      if(rootRatio>.62){
        var hairCount=Math.floor(3+rootRatio*8);
        for(var h=0;h<hairCount;h++){
          var hy=296+h*7;
          var side=h%2===0?-1:1;
          hairHTML+='<path d="M264 '+hy+' q'+(side*18)+' 7 '+(side*25)+' 15" fill="none" stroke="#F8FAFC" stroke-width="2.5"/>';
        }
      }
      rootHairs.innerHTML=hairHTML;
      var waterHTML='';
      var waterCount=Math.floor(W/12);
      for(var w=0;w<waterCount;w++){
        var wx=74+(w%7)*52;
        var wy=162+Math.floor(w/7)*42+(w%2)*10;
        waterHTML+='<path class="sg-drift" d="M'+wx+' '+wy+' C'+(wx-7)+' '+(wy+11)+' '+wx+' '+(wy+21)+' '+wx+' '+(wy+21)+' C'+wx+' '+(wy+21)+' '+(wx+7)+' '+(wy+11)+' '+wx+' '+wy+'Z" fill="#38BDF8" opacity=".74"/>';
      }
      waterFlow.innerHTML=waterHTML;
      var oxygenHTML='';
      var oxygenCount=Math.floor(O/13);
      for(var o=0;o<oxygenCount;o++){
        var ox=88+(o%8)*46;
        var oy=350-Math.floor(o/8)*28;
        oxygenHTML+='<circle cx="'+ox+'" cy="'+oy+'" r="'+(4+o%3)+'" fill="#E0F2FE" stroke="#0284C7" stroke-width="1.8" opacity=".84"/>';
        if(o%3===0){
          oxygenHTML+='<text x="'+(ox+8)+'" y="'+(oy+4)+'" font-size="9" font-weight="900" fill="#0284C7">O₂</text>';
        }
      }
      oxygenFlow.innerHTML=oxygenHTML;
      var rayHTML='';
      var rayCount=Math.floor(2+L/18);
      for(var r=0;r<rayCount;r++){
        var angle=-.45+r*.18;
        var x1=40*Math.cos(angle),y1=40*Math.sin(angle);
        var x2=(70+L*.45)*Math.cos(angle),y2=(70+L*.45)*Math.sin(angle);
        rayHTML+='<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2+'" stroke="#F59E0B" stroke-width="'+(2+L/35)+'" stroke-linecap="round" opacity=".78"/>';
      }
      lightRays.innerHTML=rayHTML;
      var cohortHTML='';
      var germinatedCount=Math.round(germinationRate/100*12);
      for(var c=0;c<12;c++){
        var cx=30+(c%6)*34;
        var cy=58+Math.floor(c/6)*48;
        var germinated=c<germinatedCount;
        cohortHTML+='<ellipse cx="'+cx+'" cy="'+cy+'" rx="12" ry="9" fill="'+(germinated?'#86EFAC':'#FDE68A')+'" stroke="'+(germinated?'#15803D':'#B45309')+'" stroke-width="2.5"/>';
        if(germinated){
          cohortHTML+='<path d="M'+cx+' '+(cy+8)+' q3 13 -1 23" fill="none" stroke="#F8FAFC" stroke-width="3" stroke-linecap="round"/>';
        }
      }
      cohort.innerHTML=cohortHTML;
      cohortNote.textContent='约 '+germinatedCount+' / 12 粒已萌发；比例反映萌发率，不等同于单粒种子的生长速度。';
      var progressX=18+192*stageProgress;
      progressLine.setAttribute('x2',String(progressX));
      progressMarker.setAttribute('cx',String(progressX));
      var conditionText='当前主要条件较适宜，种子能够较快进入后续萌发阶段。';
      if(W<20){
        conditionText='水分不足，种子难以充分吸胀，萌发率和萌发速度都会明显降低。';
      }else if(O<18){
        conditionText='氧气不足会限制细胞呼吸和能量供应，萌发进程明显变慢。';
      }else if(T<10){
        conditionText='低温使酶促反应和代谢活动减慢，延长观察时间也不等于条件已适宜。';
      }else if(T>36){
        conditionText='温度过高会使萌发相关生命活动受到抑制，极端条件还可能造成损伤。';
      }else if(W>92){
        conditionText='水分过多时，种子周围空气可能减少，真实实验中常会伴随缺氧风险。';
      }
      result.innerHTML='萌发率表示一批种子中已萌发的比例；萌发速度表示萌发进程快慢，两者不能混为一谈。'
        +'<br>'+conditionText+' '+lightExplanation
        +' 光照并不是所有种子萌发的普遍必要条件。';
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
        if(!data)return;
        temperature.value=String(data.temperature);
        water.value=String(data.water);
        oxygen.value=String(data.oxygen);
        light.value=String(data.light);
        time.value=String(data.time);
        updateScenarioActive(name);
        update();
      };
    }
    autoButton.onclick=function(){
      automatic=!automatic;
      autoButton.textContent=automatic?'时间推进：运行中':'时间推进：已暂停';
      autoButton.classList.toggle('paused',!automatic);
      schedule();
    };
    temperature.oninput=function(){updateScenarioActive('');update();};
    water.oninput=function(){updateScenarioActive('');update();};
    oxygen.oninput=function(){updateScenarioActive('');update();};
    light.oninput=function(){updateScenarioActive('');update();};
    time.oninput=function(){updateScenarioActive('');update();};
    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
