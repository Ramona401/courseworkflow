/**
 * lifeScienceLabTemplatesPlantHormoneGrowth.ts
 *
 * 平面生命科学实验室：植物激素与生长调节。
 *
 * 教学目标：
 * 1. 比较根和茎对生长素浓度的不同敏感性；
 * 2. 理解适宜浓度可促进生长，浓度过高可能抑制生长；
 * 3. 观察顶芽存在或去除时侧芽生长的变化，理解顶端优势；
 * 4. 观察单侧光照下茎两侧相对伸长差异与向光弯曲；
 * 5. 认识植物生长通常由多种激素和环境条件共同调节。
 *
 * 教学边界：
 * 1. 本模板中的生长素浓度、伸长量和抑制程度均为相对教学指标；
 * 2. 根通常比茎对生长素更敏感，同一浓度对不同器官的作用可能不同；
 * 3. 顶端优势与顶芽产生的生长素及其他信号共同相关，本模型作教学简化；
 * 4. 茎的向光性表现为生长方向改变，不是植物主动朝光源移动；
 * 5. 植物生长不是由单一激素独立决定，而是多种激素、组织状态和环境共同作用。
 *
 * 工程约束：
 * 1. 纯 HTML + SVG + 原生 JavaScript，不依赖外部图片、脚本、样式或 CDN；
 * 2. 所有 DOM 查询均限定在 rootId 内，支持同页多个组件实例；
 * 3. 使用生命科学统一 .bl-* 布局协议，嵌入课件后自动转换为底部课堂控制条；
 * 4. 本文件只导出独立模板数组，聚合接入由后续批次完成。
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

  return typeof value === 'boolean'
    ? value
    : fallback
}

function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

function hormoneStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BBF7D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#F5F3FF);border-bottom:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:242px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#FAFFF9;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:10px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#16A34A}'
    + '#' + rootId + ' .ph-subtitle{margin:7px 0;font-size:12px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .ph-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .ph-buttons.two{grid-template-columns:repeat(2,1fr)}'
    + '#' + rootId + ' .ph-button{height:31px;padding:0 4px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ph-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.13)}'
    + '#' + rootId + ' .ph-toggle{width:100%;height:32px;margin-bottom:8px;border:0;border-radius:8px;background:linear-gradient(135deg,#4ADE80,#16A34A);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ph-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ph-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:8px}'
    + '#' + rootId + ' .ph-card{padding:7px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ph-card b{display:block;font-size:16px;color:#15803D}'
    + '#' + rootId + ' .ph-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ph-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.5s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_PLANT_HORMONE_GROWTH:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-plant-hormone-growth',
    group: '🌱 植物生长与发育',
    name: '植物激素与生长调节',
    emoji: '🌿',
    desc: '调节生长素相对浓度、器官类型、光照方向、顶芽状态和生长时间，观察浓度效应、顶端优势与向光弯曲',
    params: [
      {
        key: 'auxinConcentration',
        label: '生长素相对浓度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 38,
      },
      {
        key: 'organType',
        label: '器官类型',
        type: 'number',
        min: 0,
        max: 1,
        step: 1,
        defaultValue: 1,
        hint: '0=根，1=茎',
      },
      {
        key: 'lightDirection',
        label: '光照方向',
        type: 'number',
        min: 0,
        max: 2,
        step: 1,
        defaultValue: 0,
        hint: '0=左侧，1=右侧，2=上方',
      },
      {
        key: 'growthTime',
        label: '生长时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 70,
      },
      {
        key: 'apicalBud',
        label: '保留顶芽',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showHormone',
        label: '显示激素分布',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const auxinConcentration = num(params, 'auxinConcentration', 38)
      const organType = num(params, 'organType', 1)
      const lightDirection = num(params, 'lightDirection', 0)
      const growthTime = num(params, 'growthTime', 70)
      const apicalBud = bool(params, 'apicalBud', true)
      const showHormone = bool(params, 'showHormone', true)

      return `
<div id="${rootId}">
${hormoneStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌿 植物激素与生长调节</div>
    <div class="bl-note">相对教学模型，不代表真实浓度或实验测量</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>生长素相对浓度</span>
          <span class="bl-value" data-auxin-value></span>
        </div>
        <input data-auxin-range type="range" min="0" max="100" step="1" value="${n(auxinConcentration)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>器官类型</span>
          <span class="bl-value" data-organ-value></span>
        </div>
        <input data-organ-range type="range" min="0" max="1" step="1" value="${n(organType)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>光照方向</span>
          <span class="bl-value" data-light-value></span>
        </div>
        <input data-light type="range" min="0" max="2" step="1" value="${n(lightDirection)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>生长时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(growthTime)}">
      </div>

      <div class="ph-subtitle">观察主题</div>

      <div class="ph-buttons">
        <button type="button" class="ph-button active" data-mode="concentration">浓度效应</button>
        <button type="button" class="ph-button" data-mode="apical">顶端优势</button>
        <button type="button" class="ph-button" data-mode="phototropism">向光调节</button>
      </div>

      <div class="ph-buttons two">
        <button type="button" class="ph-button${apicalBud ? ' active' : ''}" data-bud="intact">保留顶芽</button>
        <button type="button" class="ph-button${apicalBud ? '' : ' active'}" data-bud="removed">去除顶芽</button>
      </div>

      <button type="button" class="ph-toggle${showHormone ? '' : ' off'}" data-hormone-toggle>
        ${showHormone ? '激素分布：显示' : '激素分布：隐藏'}
      </button>

      <div class="ph-status">
        <div class="ph-card">
          <b data-response-value></b>
          <span>相对生长效应</span>
        </div>

        <div class="ph-card">
          <b data-state-value></b>
          <span>当前主要状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg viewBox="0 0 680 414" aria-label="植物激素与生长调节互动示意图">
        <defs>
          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker id="${rootId}-arrow-purple" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#14532D" flood-opacity=".12"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#166534"></text>
        <text x="28" y="68" data-summary font-size="14" font-weight="800" fill="#475569"></text>

        <g data-concentration-scene>
          <line x1="92" y1="326" x2="626" y2="326" stroke="#94A3B8" stroke-width="3"/>
          <line x1="92" y1="98" x2="92" y2="326" stroke="#94A3B8" stroke-width="3"/>
          <line x1="92" y1="212" x2="626" y2="212" stroke="#CBD5E1" stroke-width="2" stroke-dasharray="7 7"/>
          <text x="609" y="351" font-size="12" font-weight="800" fill="#64748B">浓度</text>
          <text x="28" y="111" font-size="12" font-weight="800" fill="#64748B">促进</text>
          <text x="28" y="316" font-size="12" font-weight="800" fill="#64748B">抑制</text>
          <text x="100" y="204" font-size="11" font-weight="800" fill="#64748B">0</text>
          <path data-root-curve fill="none" stroke="#B45309" stroke-width="5" stroke-linecap="round"/>
          <path data-stem-curve fill="none" stroke="#16A34A" stroke-width="5" stroke-linecap="round"/>
          <g data-current-marker></g>
          <g transform="translate(408 92)">
            <rect width="218" height="76" rx="16" fill="#F8FAFC" stroke="#D1D5DB" stroke-width="2"/>
            <line x1="18" y1="25" x2="56" y2="25" stroke="#B45309" stroke-width="5"/>
            <text x="68" y="30" font-size="13" font-weight="900" fill="#92400E">根：更敏感</text>
            <line x1="18" y1="53" x2="56" y2="53" stroke="#16A34A" stroke-width="5"/>
            <text x="68" y="58" font-size="13" font-weight="900" fill="#166534">茎：适宜范围较高</text>
          </g>
          <text x="108" y="382" data-concentration-note font-size="14" font-weight="900" fill="#166534"></text>
        </g>

        <g data-apical-scene visibility="hidden" filter="url(#${rootId}-shadow)">
          <rect x="70" y="328" width="540" height="45" rx="20" fill="#FEF3C7" stroke="#D97706" stroke-width="3"/>
          <path data-main-stem d="M340 330 C340 272 340 202 340 112" fill="none" stroke="#15803D" stroke-width="19" stroke-linecap="round"/>
          <g data-branches></g>
          <g data-apical-bud></g>
          <g data-auxin-stream></g>
          <text x="78" y="92" data-apical-label font-size="23" font-weight="900" fill="#166534"></text>
          <text x="78" y="121" data-apical-summary font-size="14" font-weight="800" fill="#475569"></text>
          <g transform="translate(78 248)">
            <rect width="165" height="76" rx="15" fill="#F0FDF4" stroke="#86EFAC" stroke-width="2"/>
            <text x="18" y="27" font-size="12" font-weight="800" fill="#64748B">主茎相对伸长</text>
            <rect x="18" y="43" width="128" height="15" rx="7" fill="#D1FAE5"/>
            <rect data-main-growth x="18" y="43" width="0" height="15" rx="7" fill="#16A34A"/>
          </g>
          <g transform="translate(438 248)">
            <rect width="165" height="76" rx="15" fill="#F5F3FF" stroke="#C4B5FD" stroke-width="2"/>
            <text x="18" y="27" font-size="12" font-weight="800" fill="#64748B">侧芽相对生长</text>
            <rect x="18" y="43" width="128" height="15" rx="7" fill="#EDE9FE"/>
            <rect data-lateral-growth x="18" y="43" width="0" height="15" rx="7" fill="#7C3AED"/>
          </g>
        </g>

        <g data-photo-scene visibility="hidden">
          <rect x="0" y="350" width="680" height="64" fill="#FEF3C7" opacity=".8"/>
          <path data-photo-stem d="M340 354 C340 296 340 224 340 142" fill="none" stroke="#15803D" stroke-width="22" stroke-linecap="round"/>
          <g data-photo-leaves></g>
          <g data-light-source></g>
          <g data-side-growth></g>
          <g data-photo-hormone></g>
          <text x="68" y="96" data-photo-label font-size="24" font-weight="900" fill="#166534"></text>
          <text x="68" y="126" data-photo-summary font-size="14" font-weight="800" fill="#475569"></text>
          <g transform="translate(468 230)">
            <rect width="172" height="98" rx="16" fill="#F8FAFC" stroke="#D1D5DB" stroke-width="2"/>
            <text x="18" y="25" font-size="12" font-weight="800" fill="#64748B">两侧相对伸长</text>
            <text x="18" y="50" data-side-a-label font-size="11" font-weight="900" fill="#166534"></text>
            <text x="18" y="76" data-side-b-label font-size="11" font-weight="900" fill="#166534"></text>
            <rect x="82" y="39" width="74" height="12" rx="6" fill="#D1FAE5"/>
            <rect x="82" y="65" width="74" height="12" rx="6" fill="#D1FAE5"/>
            <rect data-side-a-bar x="82" y="39" width="0" height="12" rx="6" fill="#16A34A"/>
            <rect data-side-b-bar x="82" y="65" width="0" height="12" rx="6" fill="#65A30D"/>
          </g>
        </g>

        <text x="28" y="398" data-stage-note font-size="13" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var auxin=root.querySelector('[data-auxin-range]');
    var organRange=root.querySelector('[data-organ-range]');
    var light=root.querySelector('[data-light]');
    var time=root.querySelector('[data-time]');

    var auxinValue=root.querySelector('[data-auxin-value]');
    var organValue=root.querySelector('[data-organ-value]');
    var lightValue=root.querySelector('[data-light-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var budButtons=root.querySelectorAll('[data-bud]');
    var hormoneToggle=root.querySelector('[data-hormone-toggle]');

    var responseValue=root.querySelector('[data-response-value]');
    var stateValue=root.querySelector('[data-state-value]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var concentrationScene=root.querySelector('[data-concentration-scene]');
    var apicalScene=root.querySelector('[data-apical-scene]');
    var photoScene=root.querySelector('[data-photo-scene]');

    var rootCurve=root.querySelector('[data-root-curve]');
    var stemCurve=root.querySelector('[data-stem-curve]');
    var currentMarker=root.querySelector('[data-current-marker]');
    var concentrationNote=root.querySelector('[data-concentration-note]');

    var branches=root.querySelector('[data-branches]');
    var apicalBudGraphic=root.querySelector('[data-apical-bud]');
    var auxinStream=root.querySelector('[data-auxin-stream]');
    var apicalLabel=root.querySelector('[data-apical-label]');
    var apicalSummary=root.querySelector('[data-apical-summary]');
    var mainGrowth=root.querySelector('[data-main-growth]');
    var lateralGrowth=root.querySelector('[data-lateral-growth]');

    var photoStem=root.querySelector('[data-photo-stem]');
    var photoLeaves=root.querySelector('[data-photo-leaves]');
    var lightSource=root.querySelector('[data-light-source]');
    var sideGrowth=root.querySelector('[data-side-growth]');
    var photoHormone=root.querySelector('[data-photo-hormone]');
    var photoLabel=root.querySelector('[data-photo-label]');
    var photoSummary=root.querySelector('[data-photo-summary]');
    var sideALabel=root.querySelector('[data-side-a-label]');
    var sideBLabel=root.querySelector('[data-side-b-label]');
    var sideABar=root.querySelector('[data-side-a-bar]');
    var sideBBar=root.querySelector('[data-side-b-bar]');

    var mode='concentration';
    var budIntact=${apicalBud ? 'true' : 'false'};
    var showHormone=${showHormone ? 'true' : 'false'};

    var lightNames=['左侧','右侧','上方'];

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function responseAt(concentration,organ){
      var optimum=organ==='root'?18:45;
      var inhibitionStart=organ==='root'?35:72;
      var promote=concentration<=0
        ?0
        :100*(concentration/optimum)*Math.exp(1-concentration/optimum);
      var inhibition=concentration<=inhibitionStart
        ?0
        :(organ==='root'?150:115)*Math.pow(
          (concentration-inhibitionStart)/(100-inhibitionStart),
          1.35
        );

      return clamp(promote-inhibition,-100,100);
    }

    function curvePath(organ){
      var path='';

      for(var c=0;c<=100;c+=2){
        var x=92+c*5.34;
        var response=responseAt(c,organ);
        var y=212-response*1.04;
        path+=(c===0?'M':' L')+x.toFixed(1)+' '+y.toFixed(1);
      }

      return path;
    }

    function responseLabel(value){
      if(value>55)return '明显促进';
      if(value>15)return '促进生长';
      if(value>=-15)return '影响较小';
      if(value>-55)return '抑制生长';
      return '明显抑制';
    }

    function lightGraphic(direction){
      if(direction===0){
        return '<circle cx="64" cy="176" r="35" fill="#FACC15" stroke="#D97706" stroke-width="4"/>'
          +'<path class="ph-flow" d="M108 176 H238" fill="none" stroke="#F59E0B" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>';
      }

      if(direction===1){
        return '<circle cx="616" cy="176" r="35" fill="#FACC15" stroke="#D97706" stroke-width="4"/>'
          +'<path class="ph-flow" d="M572 176 H442" fill="none" stroke="#F59E0B" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>';
      }

      return '<circle cx="340" cy="88" r="35" fill="#FACC15" stroke="#D97706" stroke-width="4"/>'
        +'<path class="ph-flow" d="M340 132 V182" fill="none" stroke="#F59E0B" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>';
    }

    function updateConcentration(concentration,organ){
      concentrationScene.setAttribute('visibility','visible');
      apicalScene.setAttribute('visibility','hidden');
      photoScene.setAttribute('visibility','hidden');

      rootCurve.setAttribute('d',curvePath('root'));
      stemCurve.setAttribute('d',curvePath('stem'));

      var rootResponse=responseAt(concentration,'root');
      var stemResponse=responseAt(concentration,'stem');
      var selected=organ==='root'?rootResponse:stemResponse;
      var x=92+concentration*5.34;
      var y=212-selected*1.04;
      var color=organ==='root'?'#B45309':'#16A34A';

      currentMarker.innerHTML=''
        +'<line x1="'+x.toFixed(1)+'" y1="92" x2="'+x.toFixed(1)+'" y2="326" stroke="#7C3AED" stroke-width="2" stroke-dasharray="5 5"/>'
        +'<circle cx="'+x.toFixed(1)+'" cy="'+y.toFixed(1)+'" r="9" fill="'+color+'" stroke="#FFFFFF" stroke-width="3"/>';

      responseValue.textContent=(selected>=0?'+':'')+selected.toFixed(0);
      stateValue.textContent=responseLabel(selected);
      title.textContent='生长素浓度效应曲线';
      summary.textContent='比较同一相对浓度对根和茎生长的不同影响';
      concentrationNote.textContent='当前观察：'+(organ==='root'?'根':'茎')
        +'，相对效应 '+(selected>=0?'+':'')+selected.toFixed(0);
      stageNote.textContent='根通常比茎对生长素更敏感，适宜浓度范围也不同。';

      var concentrationState=concentration<8
        ?'浓度很低，促进作用较弱。'
        :selected>15
          ?'当前浓度处于该器官的促进范围。'
          :selected<-15
            ?'当前浓度偏高，对该器官生长表现为抑制。'
            :'当前浓度接近促进与抑制的过渡范围。';

      result.innerHTML=concentrationState
        +' 同一生长素浓度对根和茎的作用可能不同，不能只用“促进”或“抑制”作绝对判断。';
    }

    function updateApical(concentration,timeLevel){
      concentrationScene.setAttribute('visibility','hidden');
      apicalScene.setAttribute('visibility','visible');
      photoScene.setAttribute('visibility','hidden');

      var concentrationFactor=clamp(concentration/60,0,1.35);
      var timeFactor=timeLevel/100;
      var mainValue=budIntact
        ?clamp((42+42*concentrationFactor)*timeFactor,0,100)
        :clamp((22+20*concentrationFactor)*timeFactor,0,100);
      var lateralValue=budIntact
        ?clamp((58-38*concentrationFactor)*timeFactor,0,100)
        :clamp((46+48*timeFactor)*timeFactor,0,100);

      mainGrowth.setAttribute('width',String(128*mainValue/100));
      lateralGrowth.setAttribute('width',String(128*lateralValue/100));

      var branchLength=24+lateralValue*.85;
      var branchHTML='';
      var levels=[164,211,258];

      for(var i=0;i<levels.length;i++){
        var y=levels[i];
        var offset=i%2===0?1:-1;
        var leftX=340-branchLength*(.72+Math.abs(offset)*.08);
        var rightX=340+branchLength*(.72+Math.abs(offset)*.08);
        branchHTML+='<path d="M340 '+y+' Q'+(300-branchLength*.25)+' '+(y-15)+' '+leftX.toFixed(1)+' '+(y-36)+'" fill="none" stroke="#16A34A" stroke-width="'+(7+lateralValue/28)+'" stroke-linecap="round"/>';
        branchHTML+='<path d="M340 '+(y+9)+' Q'+(380+branchLength*.25)+' '+(y-4)+' '+rightX.toFixed(1)+' '+(y-25)+'" fill="none" stroke="#22C55E" stroke-width="'+(7+lateralValue/28)+'" stroke-linecap="round"/>';
        branchHTML+='<ellipse cx="'+leftX.toFixed(1)+'" cy="'+(y-38)+'" rx="18" ry="9" fill="#4ADE80" stroke="#15803D" stroke-width="3"/>';
        branchHTML+='<ellipse cx="'+rightX.toFixed(1)+'" cy="'+(y-27)+'" rx="18" ry="9" fill="#4ADE80" stroke="#15803D" stroke-width="3"/>';
      }

      branches.innerHTML=branchHTML;
      apicalBudGraphic.innerHTML=budIntact
        ?'<circle cx="340" cy="104" r="16" fill="#84CC16" stroke="#3F6212" stroke-width="4"/><path d="M340 87 L340 67" stroke="#65A30D" stroke-width="5" stroke-linecap="round"/>'
        :'<path d="M319 111 L361 91 M319 91 L361 111" stroke="#DC2626" stroke-width="7" stroke-linecap="round"/><rect x="325" y="106" width="30" height="8" rx="4" fill="#92400E"/>';

      var streamHTML='';

      if(showHormone && budIntact){
        for(var q=0;q<9;q++){
          streamHTML+='<circle cx="'+(334+(q%2)*12)+'" cy="'+(125+q*20)+'" r="5" fill="#8B5CF6" opacity=".78"/>';
        }
        streamHTML+='<path class="ph-flow" d="M340 118 V302" fill="none" stroke="#7C3AED" stroke-width="3" marker-end="url(#${rootId}-arrow-purple)"/>';
      }

      auxinStream.innerHTML=streamHTML;
      apicalLabel.textContent=budIntact?'顶芽完整：顶端优势较明显':'去除顶芽：侧芽生长释放';
      apicalSummary.textContent=budIntact
        ?'顶芽产生的信号沿主茎向下传递，侧芽生长相对受抑。'
        :'顶芽去除后，原有抑制减弱，多个侧芽可继续生长。';
      responseValue.textContent=lateralValue.toFixed(0);
      stateValue.textContent=budIntact?'侧芽受抑':'侧芽生长';
      title.textContent='顶端优势与侧芽生长';
      summary.textContent='比较保留顶芽与去除顶芽后的株形变化';
      stageNote.textContent=showHormone
        ?'紫色小点表示顶芽来源的相对生长素信号。'
        :'激素分布已隐藏，株形变化仍按当前参数计算。';

      var timeCondition=timeLevel<18
        ?'生长时间较短，株形差异尚不明显。'
        :budIntact
          ?'顶芽完整时主茎生长占优势，侧芽相对受抑。'
          :'去除顶芽后侧芽生长增强，植株更容易形成分枝。';

      result.innerHTML=timeCondition
        +' 顶端优势不是单一激素孤立完成的过程，还受到其他激素、营养和环境条件影响。';
    }

    function updatePhototropism(concentration,lightDirection,timeLevel){
      concentrationScene.setAttribute('visibility','hidden');
      apicalScene.setAttribute('visibility','hidden');
      photoScene.setAttribute('visibility','visible');

      var responsePotential=Math.max(0,responseAt(concentration,'stem'))/100;
      var timeFactor=timeLevel/100;
      var unilateral=lightDirection===2?0:1;
      var direction=lightDirection===0?-1:lightDirection===1?1:0;
      var bend=unilateral*direction*clamp(responsePotential*timeFactor,0,1)*102;
      var endX=340+bend;
      var controlX=340+bend*.34;
      var endY=140-timeFactor*26;

      photoStem.setAttribute(
        'd',
        'M340 354 C340 296 '+controlX.toFixed(1)+' 220 '+endX.toFixed(1)+' '+endY.toFixed(1)
      );

      photoLeaves.innerHTML=''
        +'<ellipse cx="'+(endX-21).toFixed(1)+'" cy="'+(endY+12).toFixed(1)+'" rx="30" ry="14" fill="#4ADE80" stroke="#15803D" stroke-width="4" transform="rotate(-25 '+(endX-21).toFixed(1)+' '+(endY+12).toFixed(1)+')"/>'
        +'<ellipse cx="'+(endX+21).toFixed(1)+'" cy="'+(endY+12).toFixed(1)+'" rx="30" ry="14" fill="#22C55E" stroke="#15803D" stroke-width="4" transform="rotate(25 '+(endX+21).toFixed(1)+' '+(endY+12).toFixed(1)+')"/>'
        +'<circle cx="'+endX.toFixed(1)+'" cy="'+endY.toFixed(1)+'" r="11" fill="#84CC16" stroke="#3F6212" stroke-width="3"/>';

      lightSource.innerHTML=lightGraphic(lightDirection);

      var asymmetry=unilateral*responsePotential*timeFactor*36;
      var shaded=50+asymmetry;
      var lit=50-asymmetry;
      var sideAName=lightDirection===0?'右侧（背光）':lightDirection===1?'左侧（背光）':'两侧接近';
      var sideBName=lightDirection===0?'左侧（向光）':lightDirection===1?'右侧（向光）':'两侧接近';

      sideALabel.textContent=sideAName;
      sideBLabel.textContent=sideBName;
      sideABar.setAttribute('width',String(74*shaded/100));
      sideBBar.setAttribute('width',String(74*lit/100));

      sideGrowth.innerHTML=unilateral
        ?'<path d="M340 260 C'+(340-direction*35)+' 232 '+(340-direction*54)+' 204 '+(340-direction*64)+' 176" fill="none" stroke="#16A34A" stroke-width="6" marker-end="url(#${rootId}-arrow-green)" opacity=".78"/>'
        :'<path d="M316 242 V188 M364 242 V188" fill="none" stroke="#16A34A" stroke-width="5" marker-end="url(#${rootId}-arrow-green)" opacity=".7"/>';

      var hormoneHTML='';

      if(showHormone){
        var dotCount=10;
        for(var i=0;i<dotCount;i++){
          var progress=(i+1)/(dotCount+1);
          var x=340+(endX-340)*progress;
          var y=340+(endY-340)*progress;
          var offset=unilateral
            ?direction*(7+(i%3)*3)
            :(i%2===0?-7:7);
          hormoneHTML+='<circle cx="'+(x+offset).toFixed(1)+'" cy="'+y.toFixed(1)+'" r="'+(4+i%2)+'" fill="#8B5CF6" opacity=".78"/>';
        }
      }

      photoHormone.innerHTML=hormoneHTML;
      photoLabel.textContent=unilateral?'茎向光弯曲':'顶部光照：两侧伸长接近';
      photoSummary.textContent=unilateral
        ?'背光侧生长素相对较多并促进该侧伸长，茎向光弯曲。'
        :'光从上方照射时，茎两侧没有明显的单侧光差异。';
      responseValue.textContent=Math.abs(bend).toFixed(0)+'°';
      stateValue.textContent=unilateral?'向光生长':'近直立生长';
      title.textContent='单侧光照下的生长调节';
      summary.textContent='观察生长素相对分布、两侧伸长差异和茎的弯曲方向';
      stageNote.textContent=showHormone
        ?'紫色小点表示相对生长素分布，箭头表示伸长较快的一侧。'
        :'激素分布已隐藏，仍可比较两侧相对伸长。';

      var condition=timeLevel<18
        ?'生长时间很短，尚未形成明显弯曲。'
        :responsePotential<.18
          ?'当前生长素浓度对茎伸长的促进效应较弱，弯曲不明显。'
          :unilateral
            ?'两侧伸长速度不同，逐渐形成向光弯曲。'
            :'顶部光照下两侧伸长接近，茎保持近直立生长。';

      result.innerHTML=condition
        +' 向光性是生长方向改变，不是植物主动朝光源移动。';
    }

    function update(){
      var concentration=Number(auxin.value);
      var organ=Math.round(Number(organRange.value))===0?'root':'stem';
      var lightDirection=Math.round(Number(light.value));
      var timeLevel=Number(time.value);

      auxinValue.textContent=concentration.toFixed(0);
      organValue.textContent=organ==='root'?'根':'茎';
      lightValue.textContent=lightNames[lightDirection];
      timeValue.textContent=timeLevel.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      for(var j=0;j<budButtons.length;j++){
        budButtons[j].classList.toggle(
          'active',
          (budButtons[j].getAttribute('data-bud')==='intact')===budIntact
        );
      }

      hormoneToggle.textContent=showHormone
        ?'激素分布：显示'
        :'激素分布：隐藏';
      hormoneToggle.classList.toggle('off',!showHormone);

      if(mode==='concentration'){
        updateConcentration(concentration,organ);
      }else if(mode==='apical'){
        updateApical(concentration,timeLevel);
      }else{
        updatePhototropism(concentration,lightDirection,timeLevel);
      }
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<budButtons.length;j++){
      budButtons[j].onclick=function(){
        budIntact=this.getAttribute('data-bud')==='intact';
        update();
      };
    }

    hormoneToggle.onclick=function(){
      showHormone=!showHormone;
      update();
    };

    auxin.oninput=update;
    organRange.oninput=update;
    light.oninput=update;
    time.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
