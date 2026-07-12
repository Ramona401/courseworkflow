/**
 * lifeScienceLabTemplatesImmunityVaccinationMemory.ts
 *
 * 平面生命科学实验室：疫苗接种与免疫记忆。
 *
 * 教学目标：
 * 1. 理解疫苗通过提供抗原或产生抗原的信息，预先启动适应性免疫；
 * 2. 观察抗原呈递、辅助性T细胞协同、B细胞激活和抗体形成；
 * 3. 理解初次免疫应答通常启动较慢，抗体水平逐步升高后再下降；
 * 4. 认识部分B细胞和T细胞形成记忆细胞；
 * 5. 比较初次应答和再次遇到相同抗原时的二次免疫应答；
 * 6. 理解二次免疫应答通常启动更快、强度更高、维持时间更长；
 * 7. 理解加强免疫可再次激活免疫记忆，但保护效果并非绝对。
 *
 * 科学边界：
 * 1. 疫苗可含灭活病原体、减毒病原体、病原体成分、抗原或产生抗原的信息；
 * 2. 不同技术路线的疫苗启动免疫反应的方式并不完全相同；
 * 3. 疫苗接种后形成保护需要一定时间，并不是接种后立即获得充分保护；
 * 4. 初次免疫应答通常经历抗原识别、淋巴细胞激活、克隆增殖和效应分化；
 * 5. 部分B细胞和T细胞可形成具有抗原特异性的记忆细胞；
 * 6. 再次遇到相同抗原时，记忆细胞可使应答启动更快、强度更高；
 * 7. 血液中抗体水平下降不等于免疫记忆完全消失；
 * 8. 加强免疫可提高抗体水平并扩增记忆细胞，但并非所有疫苗都采用相同程序；
 * 9. 疫苗通常降低感染、发病或重症风险，但不能保证所有接种者绝对不感染；
 * 10. 对不同抗原或发生显著变化的抗原，既有免疫记忆的识别效果可能不同；
 * 11. 图中的抗体水平、记忆指数和保护指数均为相对教学指标；
 * 12. 本模板只用于生物学教学，不提供疫苗选择、接种时机或医学建议。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript，不依赖外部图片、脚本、样式或CDN；
 * 2. 所有DOM查询均限定在rootId内部，支持同页多个独立实例；
 * 3. 使用生命科学统一.bl-*布局协议；
 * 4. 本文件只导出独立模板数组，聚合入口由后续批次统一接入。
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

function vaccinationMemoryStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#EDE9FE);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FAFCFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9}'
    + '#' + rootId + ' .vm-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .vm-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .vm-button{min-height:30px;padding:3px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .vm-button.active{border-color:#0284C7;background:#E0F2FE;box-shadow:0 3px 9px rgba(2,132,199,.13)}'
    + '#' + rootId + ' .vm-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .vm-toggle.off{background:#64748B}'
    + '#' + rootId + ' .vm-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .vm-card{padding:6px 3px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .vm-card b{display:block;font-size:13px;color:#0369A1}'
    + '#' + rootId + ' .vm-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .vm-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--vm-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .vm-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_VACCINATION_MEMORY:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-vaccination-immune-memory',
    group: '🛡️ 免疫与疾病防御',
    name: '疫苗接种与免疫记忆',
    emoji: '💉',
    desc: '调节疫苗抗原、先天免疫信号、辅助性T细胞、记忆形成、应答时间和再次暴露，比较初次与二次免疫应答',
    params: [
      {
        key: 'vaccineAntigenDose',
        label: '疫苗抗原相对剂量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 66,
      },
      {
        key: 'innateSignal',
        label: '先天免疫启动信号',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 70,
      },
      {
        key: 'helperTSupport',
        label: '辅助性T细胞协同',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'memoryFormation',
        label: '记忆细胞形成水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 80,
      },
      {
        key: 'responseTime',
        label: '接种后相对时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 54,
      },
      {
        key: 'challengeExposure',
        label: '再次暴露抗原量',
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
      const vaccineAntigenDose = num(
        params,
        'vaccineAntigenDose',
        66,
      )
      const innateSignal = num(params, 'innateSignal', 70)
      const helperTSupport = num(params, 'helperTSupport', 76)
      const memoryFormation = num(params, 'memoryFormation', 80)
      const responseTime = num(params, 'responseTime', 54)
      const challengeExposure = num(params, 'challengeExposure', 62)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${vaccinationMemoryStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">💉 疫苗接种与免疫记忆</div>
    <div class="bl-note">抗体、记忆和保护均为相对教学指标</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>疫苗抗原相对剂量</span>
          <span class="bl-value" data-dose-value></span>
        </div>
        <input
          data-dose
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(vaccineAntigenDose)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>先天免疫启动信号</span>
          <span class="bl-value" data-innate-value></span>
        </div>
        <input
          data-innate
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(innateSignal)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>辅助性T细胞协同</span>
          <span class="bl-value" data-helper-value></span>
        </div>
        <input
          data-helper
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(helperTSupport)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>记忆细胞形成水平</span>
          <span class="bl-value" data-memory-value></span>
        </div>
        <input
          data-memory
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(memoryFormation)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>接种后相对时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input
          data-time
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(responseTime)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>再次暴露抗原量</span>
          <span class="bl-value" data-challenge-value></span>
        </div>
        <input
          data-challenge
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(challengeExposure)}"
        >
      </div>

      <div class="vm-subtitle">观察方式</div>

      <div class="vm-buttons">
        <button
          type="button"
          class="vm-button active"
          data-mode="vaccination"
        >接种与抗原呈递</button>

        <button
          type="button"
          class="vm-button"
          data-mode="primary"
        >初次免疫应答</button>

        <button
          type="button"
          class="vm-button"
          data-mode="memory"
        >记忆细胞形成</button>

        <button
          type="button"
          class="vm-button"
          data-mode="secondary"
        >二次免疫应答</button>
      </div>

      <button
        type="button"
        class="vm-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="vm-toggle"
        data-auto
      >应答推进：运行中</button>

      <div class="vm-status">
        <div class="vm-card">
          <b data-antibody-index></b>
          <span>抗体水平</span>
        </div>

        <div class="vm-card">
          <b data-memory-index></b>
          <span>免疫记忆</span>
        </div>

        <div class="vm-card">
          <b data-protection-index></b>
          <span>保护指数</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="疫苗接种与免疫记忆互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-blue"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker
            id="${rootId}-arrow-purple"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <marker
            id="${rootId}-arrow-green"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#075985"
              flood-opacity=".13"
            />
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text
          x="24"
          y="36"
          data-title
          font-size="26"
          font-weight="900"
          fill="#075985"
        ></text>

        <text
          x="24"
          y="65"
          data-summary
          font-size="14"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-dynamic></g>
        <g data-labels></g>

        <g transform="translate(518 337)">
          <rect
            width="216"
            height="66"
            rx="15"
            fill="#F0F9FF"
            stroke="#BAE6FD"
            stroke-width="2"
          />

          <text
            x="108"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#075985"
          >关键边界</text>

          <text
            x="108"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#0C4A6E"
          >保护形成需要时间</text>

          <text
            x="108"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#0C4A6E"
          >接种不等于绝对不会感染</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#075985"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var doseInput=root.querySelector('[data-dose]');
    var innateInput=root.querySelector('[data-innate]');
    var helperInput=root.querySelector('[data-helper]');
    var memoryInput=root.querySelector('[data-memory]');
    var timeInput=root.querySelector('[data-time]');
    var challengeInput=root.querySelector('[data-challenge]');

    var doseValue=root.querySelector('[data-dose-value]');
    var innateValue=root.querySelector('[data-innate-value]');
    var helperValue=root.querySelector('[data-helper-value]');
    var memoryValue=root.querySelector('[data-memory-value]');
    var timeValue=root.querySelector('[data-time-value]');
    var challengeValue=root.querySelector('[data-challenge-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var antibodyText=root.querySelector('[data-antibody-index]');
    var memoryText=root.querySelector('[data-memory-index]');
    var protectionText=root.querySelector('[data-protection-index]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');
    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='vaccination';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
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
        var next=Number(timeInput.value)+2;

        timeInput.value=String(
          next>100?0:next
        );

        update();
        schedule();
      },800);
    }

    function antigenShape(
      x,
      y,
      size,
      opacity
    ){
      var points='';

      for(var i=0;i<10;i++){
        var angle=-Math.PI/2+Math.PI*2*i/10;
        var radius=i%2===0
          ?size
          :size*.55;

        points+=(x+Math.cos(angle)*radius).toFixed(1)
          +','
          +(y+Math.sin(angle)*radius).toFixed(1)
          +' ';
      }

      return '<polygon points="'+points.trim()
        +'" fill="#FCA5A5" stroke="#B91C1C" stroke-width="2.5" opacity="'+opacity+'"/>';
    }

    function antibodyShape(
      x,
      y,
      scale,
      rotation,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') rotate('+rotation+') scale('+scale+')" opacity="'+opacity+'">'
        +'<path d="M0 25 V6 M0 8 L-17 -12 M0 8 L17 -12" fill="none" stroke="#7C3AED" stroke-width="7" stroke-linecap="round"/>'
        +'<circle cx="-18" cy="-14" r="4" fill="#C4B5FD"/>'
        +'<circle cx="18" cy="-14" r="4" fill="#C4B5FD"/>'
        +'</g>';
    }

    function apcShape(
      x,
      y,
      scale,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<path d="M-42 -6 C-40 -35 -8 -48 18 -35 C44 -22 49 14 30 37 C9 61 -31 47 -44 21 C-50 9 -49 0 -42 -6Z" fill="#FEF3C7" stroke="#D97706" stroke-width="5"/>'
        +'<ellipse cx="-7" cy="4" rx="18" ry="14" fill="#FBBF24" stroke="#B45309" stroke-width="3"/>'
        +'<rect x="23" y="-14" width="16" height="28" rx="5" fill="#DBEAFE" stroke="#2563EB" stroke-width="3"/>'
        +'<path d="M31 -5 l9 -9 M31 5 l9 9" stroke="#DC2626" stroke-width="3"/>'
        +'<text x="0" y="69" text-anchor="middle" font-size="11" font-weight="900" fill="#92400E">抗原呈递细胞</text>'
        +'</g>';
    }

    function helperTShape(
      x,
      y,
      signal,
      opacity
    ){
      return ''
        +'<g opacity="'+opacity+'">'
        +'<circle cx="'+x+'" cy="'+y+'" r="39" fill="#DCFCE7" stroke="#16A34A" stroke-width="5"/>'
        +'<circle cx="'+x+'" cy="'+y+'" r="15" fill="#86EFAC" stroke="#15803D" stroke-width="3"/>'
        +'<text x="'+x+'" y="'+(y+6)+'" text-anchor="middle" font-size="17" font-weight="900" fill="#166534">Th</text>'
        +'<circle class="vm-pulse" cx="'+x+'" cy="'+y+'" r="'+(45+signal*.12)
        +'" fill="none" stroke="#22C55E" stroke-width="4" stroke-dasharray="7 6" opacity="'+(.25+.7*signal/100)+'"/>'
        +'</g>';
    }

    function memoryCellShape(
      x,
      y,
      type,
      scale,
      opacity
    ){
      var isB=type==='B';
      var stroke=isB
        ?'#0284C7'
        :'#16A34A';
      var fill=isB
        ?'#E0F2FE'
        :'#DCFCE7';

      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle r="35" fill="'+fill+'" stroke="'+stroke+'" stroke-width="5"/>'
        +'<circle r="13" fill="'+(isB?'#7DD3FC':'#86EFAC')+'" stroke="'+stroke+'" stroke-width="3"/>'
        +'<text x="0" y="6" text-anchor="middle" font-size="16" font-weight="900" fill="'+stroke+'">M'+type+'</text>'
        +'</g>';
    }

    function graphAxes(){
      return ''
        +'<line x1="74" y1="327" x2="704" y2="327" stroke="#64748B" stroke-width="3"/>'
        +'<line x1="74" y1="327" x2="74" y2="103" stroke="#64748B" stroke-width="3"/>'
        +'<text x="389" y="359" text-anchor="middle" font-size="12" font-weight="900" fill="#475569">相对时间</text>'
        +'<text x="22" y="222" transform="rotate(-90 22 222)" text-anchor="middle" font-size="12" font-weight="900" fill="#475569">相对抗体水平</text>'
        +'<line x1="74" y1="271" x2="704" y2="271" stroke="#E2E8F0" stroke-width="2" stroke-dasharray="6 6"/>'
        +'<line x1="74" y1="215" x2="704" y2="215" stroke="#E2E8F0" stroke-width="2" stroke-dasharray="6 6"/>'
        +'<line x1="74" y1="159" x2="704" y2="159" stroke="#E2E8F0" stroke-width="2" stroke-dasharray="6 6"/>';
    }

    function primaryCurvePath(
      responseStrength,
      progress
    ){
      var peakX=385;
      var peakY=327-responseStrength*1.65;
      var endY=327-responseStrength*.62;
      var currentX=92+progress*580;

      return {
        full:'M92 327 C158 326 186 315 220 278 C263 230 306 '
          +peakY.toFixed(1)
          +' '+peakX+' '+peakY.toFixed(1)
          +' C493 '+peakY.toFixed(1)
          +' 578 '+endY.toFixed(1)
          +' 687 '+endY.toFixed(1),
        currentX:currentX,
      };
    }

    function secondaryCurvePath(
      primaryStrength,
      secondaryStrength,
      progress
    ){
      var primaryPeakY=327-primaryStrength*1.05;
      var primaryEndY=327-primaryStrength*.35;
      var secondaryPeakY=327-secondaryStrength*1.95;
      var secondaryEndY=327-secondaryStrength*.95;
      var currentX=92+progress*580;

      return {
        primary:'M92 327 C135 326 159 311 184 279 C221 230 261 '
          +primaryPeakY.toFixed(1)
          +' 307 '+primaryPeakY.toFixed(1)
          +' C354 '+primaryPeakY.toFixed(1)
          +' 391 '+primaryEndY.toFixed(1)
          +' 430 '+primaryEndY.toFixed(1),
        secondary:'M430 '+primaryEndY.toFixed(1)
          +' C450 '+primaryEndY.toFixed(1)
          +' 463 '+secondaryPeakY.toFixed(1)
          +' 499 '+secondaryPeakY.toFixed(1)
          +' C558 '+secondaryPeakY.toFixed(1)
          +' 617 '+secondaryEndY.toFixed(1)
          +' 687 '+secondaryEndY.toFixed(1),
        currentX:currentX,
      };
    }

    function renderVaccination(
      dose,
      innate,
      helper,
      progress
    ){
      var antigenCount=Math.floor(
        3+dose/9
      );

      var antigenHTML='';

      for(var i=0;i<antigenCount;i++){
        var ax=58+(i%5)*42;
        var ay=130+Math.floor(i/5)*47;

        antigenHTML+=antigenShape(
          ax,
          ay,
          12+i%3,
          .72
        );
      }

      var uptake=clamp(
        progress
        *Math.sqrt(
          (.18+.82*dose/100)
          *(.18+.82*innate/100)
        ),
        0,
        1
      );

      return ''
        +'<rect x="28" y="88" width="245" height="254" rx="24" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="150" y="117" text-anchor="middle" font-size="15" font-weight="900" fill="#9A3412">疫苗抗原或抗原信息</text>'
        +antigenHTML
        +'<path class="vm-flow" d="M257 207 C302 188 322 184 357 198" fill="none" stroke="#0284C7" stroke-width="5" marker-end="url(#${rootId}-arrow-blue)"/>'
        +apcShape(
          421,
          207,
          .95,
          .45+.55*uptake
        )
        +'<path class="vm-flow" d="M472 204 C505 184 521 172 542 158" fill="none" stroke="#16A34A" stroke-width="'+(3+helper/22)+'" marker-end="url(#${rootId}-arrow-green)" opacity="'+(.25+.75*helper/100)+'"/>'
        +helperTShape(
          606,
          140,
          helper,
          .45+.55*uptake
        )
        +'<path class="vm-flow" d="M468 241 C518 259 547 267 579 277" fill="none" stroke="#7C3AED" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<circle cx="632" cy="283" r="43" fill="#EDE9FE" stroke="#7C3AED" stroke-width="5"/>'
        +'<circle cx="632" cy="283" r="16" fill="#A78BFA"/>'
        +'<text x="632" y="337" text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6">特异性B细胞</text>'
        +'<g transform="translate(65 351)">'
        +'<rect width="586" height="36" rx="12" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<text x="293" y="23" text-anchor="middle" font-size="11.5" font-weight="900" fill="#334155">接种后需要经过抗原呈递、淋巴细胞激活和克隆增殖，保护不会立即形成</text>'
        +'</g>';
    }

    function renderPrimary(
      antibodyLevel,
      progress
    ){
      var curve=primaryCurvePath(
        antibodyLevel,
        progress
      );

      return ''
        +graphAxes()
        +'<path d="'+curve.full+'" fill="none" stroke="#7C3AED" stroke-width="7" stroke-linecap="round"/>'
        +'<circle class="vm-pulse" cx="'+curve.currentX.toFixed(1)
        +'" cy="'+Math.max(122,327-antibodyLevel*1.25).toFixed(1)
        +'" r="11" fill="#7C3AED" stroke="#FFFFFF" stroke-width="4"/>'
        +'<line x1="92" y1="327" x2="92" y2="95" stroke="#0284C7" stroke-width="3" stroke-dasharray="7 6"/>'
        +'<text x="92" y="82" text-anchor="middle" font-size="12" font-weight="900" fill="#0369A1">首次接种</text>'
        +'<text x="223" y="294" font-size="12" font-weight="900" fill="#64748B">潜伏期</text>'
        +'<text x="344" y="139" font-size="12" font-weight="900" fill="#5B21B6">抗体逐步升高</text>'
        +'<text x="546" y="247" font-size="12" font-weight="900" fill="#5B21B6">随后逐渐下降</text>'
        +'<g transform="translate(113 353)">'
        +'<rect width="510" height="34" rx="12" fill="#FAF5FF" stroke="#DDD6FE" stroke-width="2"/>'
        +'<text x="255" y="22" text-anchor="middle" font-size="11.5" font-weight="900" fill="#4C1D95">初次免疫应答通常启动较慢，抗体水平逐步上升后再下降</text>'
        +'</g>';
    }

    function renderMemory(
      memoryIndex,
      antibodyLevel,
      progress
    ){
      var cellCount=Math.floor(
        4+memoryIndex/10
      );

      var cells='';

      for(var i=0;i<cellCount;i++){
        var angle=Math.PI*2*i/cellCount;
        var radius=i%2===0
          ?106
          :73;
        var x=322+Math.cos(angle)*radius;
        var y=211+Math.sin(angle)*radius*.76;
        var type=i%2===0
          ?'B'
          :'T';

        cells+=memoryCellShape(
          x.toFixed(1),
          y.toFixed(1),
          type,
          .78,
          .42+.55*progress
        );
      }

      var antibodyCount=Math.floor(
        2+antibodyLevel/15
      );

      var antibodies='';

      for(var j=0;j<antibodyCount;j++){
        var bx=540+(j%4)*44;
        var by=139+Math.floor(j/4)*54;

        antibodies+=antibodyShape(
          bx,
          by,
          .57,
          (j%3-1)*13,
          .40+.45*antibodyLevel/100
        );
      }

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<circle cx="322" cy="211" r="145" fill="#F0F9FF" stroke="#7DD3FC" stroke-width="4"/>'
        +cells
        +'<circle cx="322" cy="211" r="51" fill="#DBEAFE" stroke="#0284C7" stroke-width="6"/>'
        +'<text x="322" y="202" text-anchor="middle" font-size="15" font-weight="900" fill="#075985">免疫记忆</text>'
        +'<text x="322" y="225" text-anchor="middle" font-size="12" font-weight="900" fill="#075985">抗原特异性</text>'
        +'</g>'
        +'<g transform="translate(505 91)">'
        +'<rect width="214" height="216" rx="22" fill="#FAF5FF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="107" y="28" text-anchor="middle" font-size="15" font-weight="900" fill="#5B21B6">抗体水平与免疫记忆</text>'
        +antibodies
        +'<text x="107" y="172" text-anchor="middle" font-size="11.5" font-weight="900" fill="#5B21B6">抗体可随时间下降</text>'
        +'<text x="107" y="195" text-anchor="middle" font-size="11.5" font-weight="900" fill="#047857">记忆细胞仍可保留</text>'
        +'</g>'
        +'<g transform="translate(79 351)">'
        +'<rect width="566" height="36" rx="12" fill="#ECFDF5" stroke="#A7F3D0" stroke-width="2"/>'
        +'<text x="283" y="23" text-anchor="middle" font-size="11.5" font-weight="900" fill="#166534">血液中抗体水平下降，不等于针对相同抗原的免疫记忆完全消失</text>'
        +'</g>';
    }

    function renderSecondary(
      primaryStrength,
      secondaryStrength,
      progress,
      challenge
    ){
      var curves=secondaryCurvePath(
        primaryStrength,
        secondaryStrength,
        progress
      );

      var remaining=clamp(
        challenge
        *(1-secondaryStrength/100*.82),
        0,
        100
      );

      return ''
        +graphAxes()
        +'<path d="'+curves.primary+'" fill="none" stroke="#94A3B8" stroke-width="6" stroke-linecap="round"/>'
        +'<path d="'+curves.secondary+'" fill="none" stroke="#16A34A" stroke-width="8" stroke-linecap="round"/>'
        +'<line x1="92" y1="327" x2="92" y2="95" stroke="#0284C7" stroke-width="3" stroke-dasharray="7 6"/>'
        +'<line x1="430" y1="327" x2="430" y2="95" stroke="#16A34A" stroke-width="3" stroke-dasharray="7 6"/>'
        +'<text x="92" y="82" text-anchor="middle" font-size="12" font-weight="900" fill="#0369A1">初次接种</text>'
        +'<text x="430" y="82" text-anchor="middle" font-size="12" font-weight="900" fill="#047857">再次遇到相同抗原</text>'
        +'<text x="212" y="273" font-size="12" font-weight="900" fill="#64748B">初次应答</text>'
        +'<text x="512" y="126" font-size="12" font-weight="900" fill="#047857">二次应答更快、更强</text>'
        +'<circle class="vm-pulse" cx="'+curves.currentX.toFixed(1)
        +'" cy="'+Math.max(115,327-secondaryStrength*1.55).toFixed(1)
        +'" r="11" fill="#16A34A" stroke="#FFFFFF" stroke-width="4"/>'
        +'<g transform="translate(95 351)">'
        +'<rect width="532" height="36" rx="12" fill="#ECFDF5" stroke="#A7F3D0" stroke-width="2"/>'
        +'<text x="266" y="23" text-anchor="middle" font-size="11.5" font-weight="900" fill="#166534">相对剩余抗原 '
        +remaining.toFixed(0)
        +'；二次应答通常启动更快、峰值更高并维持更久</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='vaccination'){
        labels.innerHTML=''
          +'<path d="M422 146 L488 95" stroke="#D97706" stroke-width="2.5"/>'
          +'<text x="496" y="94" font-size="13" font-weight="900" fill="#92400E">抗原呈递细胞</text>'
          +'<path d="M606 99 L676 79" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="682" y="81" font-size="13" font-weight="900" fill="#166534">辅助性T细胞</text>'
          +'<path d="M632 240 L696 214" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="702" y="216" font-size="13" font-weight="900" fill="#5B21B6">特异性B细胞</text>';
        return;
      }

      if(modeName==='primary'){
        labels.innerHTML=''
          +'<path d="M290 222 L343 184" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="350" y="181" font-size="13" font-weight="900" fill="#5B21B6">初次抗体峰值</text>'
          +'<path d="M576 253 L638 229" stroke="#64748B" stroke-width="2.5"/>'
          +'<text x="645" y="231" font-size="13" font-weight="900" fill="#475569">抗体逐渐下降</text>';
        return;
      }

      if(modeName==='memory'){
        labels.innerHTML=''
          +'<path d="M244 108 L181 82" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="87" y="80" font-size="13" font-weight="900" fill="#075985">记忆B细胞</text>'
          +'<path d="M407 117 L473 85" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="481" y="85" font-size="13" font-weight="900" fill="#166534">记忆T细胞</text>'
          +'<path d="M590 135 L674 105" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="681" y="107" font-size="13" font-weight="900" fill="#5B21B6">残余抗体</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M430 96 L479 73" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="486" y="75" font-size="13" font-weight="900" fill="#047857">再次暴露</text>'
        +'<path d="M526 132 L598 102" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="605" y="104" font-size="13" font-weight="900" fill="#047857">快速二次应答</text>';
    }

    function update(){
      var dose=Number(doseInput.value);
      var innate=Number(innateInput.value);
      var helper=Number(helperInput.value);
      var memoryFormation=Number(memoryInput.value);
      var responseTime=Number(timeInput.value);
      var challenge=Number(challengeInput.value);

      doseValue.textContent=dose.toFixed(0)+'%';
      innateValue.textContent=innate.toFixed(0)+'%';
      helperValue.textContent=helper.toFixed(0)+'%';
      memoryValue.textContent=memoryFormation.toFixed(0)+'%';
      timeValue.textContent=responseTime.toFixed(0)+'%';
      challengeValue.textContent=challenge.toFixed(0)+'%';

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

      autoButton.textContent=automatic
        ?'应答推进：运行中'
        :'应答推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var progress=responseTime/100;

      var activation=100*Math.pow(
        (.12+.88*dose/100)
        *(.12+.88*innate/100)
        *(.12+.88*helper/100)
        *(.20+.80*progress),
        .25
      );

      activation=clamp(
        activation,
        0,
        100
      );

      var primaryAntibody=100*Math.pow(
        activation/100
        *(.18+.82*progress),
        .5
      );

      primaryAntibody*=1-.48*Math.max(
        0,
        (progress-.62)/.38
      );

      primaryAntibody=clamp(
        primaryAntibody,
        0,
        100
      );

      var memoryIndex=100*Math.pow(
        activation/100
        *(.15+.85*memoryFormation/100)
        *(.22+.78*progress),
        1/3
      );

      memoryIndex=clamp(
        memoryIndex,
        0,
        100
      );

      var secondaryAntibody=100*Math.pow(
        memoryIndex/100
        *(.18+.82*helper/100)
        *(.20+.80*challenge/100),
        1/3
      );

      secondaryAntibody=clamp(
        secondaryAntibody*1.18,
        0,
        100
      );

      var protection=100*Math.sqrt(
        (.30+.70*primaryAntibody/100)
        *(.25+.75*memoryIndex/100)
      );

      protection=clamp(
        protection,
        0,
        100
      );

      antibodyText.textContent=(
        mode==='secondary'
          ?secondaryAntibody
          :primaryAntibody
      ).toFixed(0);

      memoryText.textContent=memoryIndex.toFixed(0);
      protectionText.textContent=protection.toFixed(0);

      root.style.setProperty(
        '--vm-speed',
        clamp(
          2.45-Math.max(
            primaryAntibody,
            secondaryAntibody
          )/75,
          .65,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='vaccination'){
        title.textContent='疫苗抗原进入与免疫系统启动';
        summary.textContent='观察疫苗抗原、先天免疫信号、抗原呈递和淋巴细胞激活。';

        dynamic.innerHTML=renderVaccination(
          dose,
          innate,
          helper,
          progress
        );

        stageNote.textContent=
          '疫苗技术路线不同，但都旨在使免疫系统预先认识特定抗原。';

        renderLabels(mode);
      }else if(mode==='primary'){
        title.textContent='初次免疫应答与抗体变化';
        summary.textContent='观察首次接种后抗体从缓慢升高到逐步下降的相对过程。';

        dynamic.innerHTML=renderPrimary(
          primaryAntibody,
          progress
        );

        stageNote.textContent=
          '保护形成需要经历抗原识别、克隆增殖和效应细胞分化，并非立即完成。';

        renderLabels(mode);
      }else if(mode==='memory'){
        title.textContent='记忆B细胞和记忆T细胞形成';
        summary.textContent='观察抗体水平下降后仍可保留的抗原特异性免疫记忆。';

        dynamic.innerHTML=renderMemory(
          memoryIndex,
          primaryAntibody,
          progress
        );

        stageNote.textContent=
          '抗体水平下降不等于免疫记忆完全消失，部分记忆细胞可长期保留。';

        renderLabels(mode);
      }else{
        title.textContent='再次遇到相同抗原时的二次免疫应答';
        summary.textContent='比较初次应答和记忆细胞参与下的二次应答速度与强度。';

        dynamic.innerHTML=renderSecondary(
          primaryAntibody,
          secondaryAntibody,
          progress,
          challenge
        );

        stageNote.textContent=
          '记忆细胞可使再次应答通常启动更快、强度更高并维持更久。';

        renderLabels(mode);
      }

      var condition=
        '当前抗原、先天免疫信号、辅助性T细胞和记忆形成共同支持免疫应答。';

      if(dose<10){
        condition=
          '疫苗抗原相对剂量较低，抗原呈递和后续淋巴细胞激活信号较弱。';
      }else if(innate<18){
        condition=
          '先天免疫启动信号较低，抗原呈递细胞的激活和协同作用受到限制。';
      }else if(helper<18){
        condition=
          '辅助性T细胞协同较低，典型蛋白质抗原引起的充分适应性免疫应答受到限制。';
      }else if(memoryFormation<18&&activation>25){
        condition=
          '记忆细胞形成水平较低，虽然可以产生初次抗体应答，但二次应答优势有限。';
      }else if(responseTime<15){
        condition=
          '接种后相对时间较短，保护性抗体和免疫记忆尚未充分形成。';
      }else if(mode==='secondary'&&challenge>85&&secondaryAntibody<45){
        condition=
          '再次暴露抗原量较高，而当前二次应答强度相对有限，仍可能有较多抗原未被控制。';
      }

      var principle=mode==='vaccination'
        ?'疫苗可通过提供抗原或产生抗原的信息，使免疫系统在真实暴露前建立针对特定抗原的应答准备。'
        :mode==='primary'
          ?'初次免疫应答通常启动较慢，抗体水平逐步升高并在达到峰值后下降。'
          :mode==='memory'
            ?'部分B细胞和T细胞形成具有抗原特异性的记忆细胞，抗体下降不等于免疫记忆完全消失。'
            :'再次遇到相同抗原时，记忆细胞可使二次免疫应答通常更快、更强并维持更久。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 疫苗通常降低感染、发病或重症风险，但不能保证所有接种者绝对不感染。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    autoButton.onclick=function(){
      automatic=!automatic;
      update();
      schedule();
    };

    doseInput.oninput=update;
    innateInput.oninput=update;
    helperInput.oninput=update;
    memoryInput.oninput=update;
    timeInput.oninput=update;
    challengeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
