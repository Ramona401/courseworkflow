/**
 * lifeScienceLabTemplatesHumanReproductionSystem.ts
 *
 * 平面生命科学实验室：人体生殖系统与生殖细胞运输。
 *
 * 教学目标：
 * 1. 认识男性和女性生殖系统的主要结构及其基本功能；
 * 2. 理解精子的形成、成熟、储存和运输不是同一个环节；
 * 3. 理解卵细胞排出、进入输卵管并向子宫方向运输的基本过程；
 * 4. 比较精子和卵细胞在人体生殖过程中的运输路径；
 * 5. 明确受精通常发生在输卵管，而胚胎着床通常发生在子宫；
 * 6. 区分生殖细胞形成、排出、运输、受精和着床等不同环节。
 *
 * 科学边界：
 * 1. 精子在睾丸曲细精管中形成，并在附睾中进一步成熟和储存；
 * 2. 精子可经附睾、输精管、射精管和尿道排出；
 * 3. 卵细胞由卵巢排出后进入输卵管，并在纤毛运动和平滑肌收缩等作用下运输；
 * 4. 精子进入女性生殖道后，可经过阴道、子宫颈和子宫到达输卵管；
 * 5. 人体受精通常发生在输卵管壶腹部附近，而不是子宫；
 * 6. 受精卵经过早期发育并到达子宫后，才可能发生着床；
 * 7. 图中的细胞数量、运输速度和成功指数均为相对教学指标；
 * 8. 本模板只用于生物学教学，不用于医学诊断或个体生殖能力评价。
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

/**
 * 使用生命科学实验室统一的.bl-*类名。
 * 嵌入课件后，公共布局覆盖层会自动形成：
 * 上方互动主体 + 底部课堂控制条。
 */
function humanReproductionStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#FCE7F3);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#6D28D9}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#8B5CF6}'
    + '#' + rootId + ' .hr-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#6D28D9}'
    + '#' + rootId + ' .hr-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .hr-system-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:5px;margin-bottom:7px}'
    + '#' + rootId + ' .hr-button{min-height:30px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#6D28D9;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .hr-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.13)}'
    + '#' + rootId + ' .hr-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .hr-toggle.off{background:#64748B}'
    + '#' + rootId + ' .hr-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .hr-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .hr-card b{display:block;font-size:13px;color:#6D28D9}'
    + '#' + rootId + ' .hr-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .hr-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--hr-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .hr-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_REPRODUCTION_SYSTEM:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-human-reproductive-system-gametes',
    group: '🧑 人体生殖与发育',
    name: '人体生殖系统与生殖细胞运输',
    emoji: '🧬',
    desc: '切换男性和女性生殖系统，观察精子和卵细胞的形成、排出及运输路径',
    params: [
      {
        key: 'systemType',
        label: '生殖系统',
        type: 'number',
        min: 0,
        max: 1,
        step: 1,
        defaultValue: 1,
        hint: '0=男性生殖系统，1=女性生殖系统',
      },
      {
        key: 'gameteAmount',
        label: '生殖细胞显示量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 70,
        hint: '女性模式仍只突出一个主要卵细胞，数值主要影响精子示意数量',
      },
      {
        key: 'transportEfficiency',
        label: '相对运输条件',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'transportTime',
        label: '运输过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 46,
      },
      {
        key: 'ovulationSide',
        label: '排卵侧别',
        type: 'number',
        min: 0,
        max: 1,
        step: 1,
        defaultValue: 0,
        hint: '0=左侧卵巢，1=右侧卵巢',
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const systemType = num(params, 'systemType', 1)
      const gameteAmount = num(params, 'gameteAmount', 70)
      const transportEfficiency = num(params, 'transportEfficiency', 76)
      const transportTime = num(params, 'transportTime', 46)
      const ovulationSide = num(params, 'ovulationSide', 0)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${humanReproductionStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧬 人体生殖系统与生殖细胞运输</div>
    <div class="bl-note">结构和运输过程均为教学示意</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>生殖系统</span>
          <span class="bl-value" data-system-value></span>
        </div>
        <input
          data-system
          type="range"
          min="0"
          max="1"
          step="1"
          value="${n(systemType)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>生殖细胞显示量</span>
          <span class="bl-value" data-amount-value></span>
        </div>
        <input
          data-amount
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(gameteAmount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>相对运输条件</span>
          <span class="bl-value" data-efficiency-value></span>
        </div>
        <input
          data-efficiency
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(transportEfficiency)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>运输过程时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input
          data-time
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(transportTime)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>女性排卵侧别</span>
          <span class="bl-value" data-side-value></span>
        </div>
        <input
          data-side
          type="range"
          min="0"
          max="1"
          step="1"
          value="${n(ovulationSide)}"
        >
      </div>

      <div class="hr-subtitle">快速切换系统</div>

      <div class="hr-system-buttons">
        <button
          type="button"
          class="hr-button"
          data-system-button="male"
        >男性系统</button>

        <button
          type="button"
          class="hr-button active"
          data-system-button="female"
        >女性系统</button>
      </div>

      <div class="hr-subtitle">观察方式</div>

      <div class="hr-buttons">
        <button
          type="button"
          class="hr-button active"
          data-mode="structure"
        >主要结构</button>

        <button
          type="button"
          class="hr-button"
          data-mode="transport"
        >运输路径</button>

        <button
          type="button"
          class="hr-button"
          data-mode="compare"
        >两性比较</button>
      </div>

      <button
        type="button"
        class="hr-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="hr-toggle"
        data-auto
      >运输推进：运行中</button>

      <div class="hr-status">
        <div class="hr-card">
          <b data-current-system></b>
          <span>当前系统</span>
        </div>

        <div class="hr-card">
          <b data-current-stage></b>
          <span>当前环节</span>
        </div>

        <div class="hr-card">
          <b data-transport-index></b>
          <span>运输进度</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="人体生殖系统与生殖细胞运输互动示意图"
      >
        <defs>
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
            id="${rootId}-arrow-blue"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>

          <marker
            id="${rootId}-arrow-pink"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#DB2777"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#4C1D95"
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
          fill="#6D28D9"
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
        <g data-gametes></g>

        <g transform="translate(519 337)">
          <rect
            width="215"
            height="66"
            rx="15"
            fill="#FAF5FF"
            stroke="#DDD6FE"
            stroke-width="2"
          />

          <text
            x="107"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#6D28D9"
          >关键区分</text>

          <text
            x="107"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#4C1D95"
          >受精通常发生在输卵管</text>

          <text
            x="107"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#4C1D95"
          >着床通常发生在子宫</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#6D28D9"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var systemInput=root.querySelector('[data-system]');
    var amountInput=root.querySelector('[data-amount]');
    var efficiencyInput=root.querySelector('[data-efficiency]');
    var timeInput=root.querySelector('[data-time]');
    var sideInput=root.querySelector('[data-side]');

    var systemValue=root.querySelector('[data-system-value]');
    var amountValue=root.querySelector('[data-amount-value]');
    var efficiencyValue=root.querySelector('[data-efficiency-value]');
    var timeValue=root.querySelector('[data-time-value]');
    var sideValue=root.querySelector('[data-side-value]');

    var systemButtons=root.querySelectorAll('[data-system-button]');
    var modeButtons=root.querySelectorAll('[data-mode]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var currentSystemText=root.querySelector('[data-current-system]');
    var currentStageText=root.querySelector('[data-current-stage]');
    var transportIndexText=root.querySelector('[data-transport-index]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');
    var gametes=root.querySelector('[data-gametes]');

    var mode='structure';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function setVisibleLabelState(){
      labelToggle.textContent=showLabels
        ?'结构标注：显示'
        :'结构标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );
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
      },760);
    }

    function spermShape(
      x,
      y,
      rotation,
      scale,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') rotate('+rotation+') scale('+scale+')" opacity="'+opacity+'">'
        +'<ellipse cx="0" cy="0" rx="7" ry="5" fill="#60A5FA" stroke="#1D4ED8" stroke-width="2"/>'
        +'<path d="M-7 0 Q-18 -8 -30 0 Q-42 8 -51 0" fill="none" stroke="#2563EB" stroke-width="3" stroke-linecap="round"/>'
        +'</g>';
    }

    function maleStructure(){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M287 104 C250 147 251 253 291 305 C317 338 381 338 408 306 C449 257 449 149 411 104Z" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="4"/>'
        +'<ellipse cx="322" cy="305" rx="31" ry="42" fill="#FDE68A" stroke="#B45309" stroke-width="5"/>'
        +'<ellipse cx="376" cy="305" rx="31" ry="42" fill="#FDE68A" stroke="#B45309" stroke-width="5"/>'
        +'<path d="M303 292 Q282 276 294 248 Q310 230 326 246" fill="none" stroke="#F97316" stroke-width="8" stroke-linecap="round"/>'
        +'<path d="M395 292 Q416 276 404 248 Q388 230 372 246" fill="none" stroke="#F97316" stroke-width="8" stroke-linecap="round"/>'
        +'<path d="M295 245 C240 199 253 117 314 126 C332 130 338 155 340 181" fill="none" stroke="#7C3AED" stroke-width="8" stroke-linecap="round"/>'
        +'<path d="M403 245 C458 199 445 117 384 126 C366 130 360 155 358 181" fill="none" stroke="#7C3AED" stroke-width="8" stroke-linecap="round"/>'
        +'<ellipse cx="321" cy="126" rx="29" ry="20" fill="#F9A8D4" stroke="#BE185D" stroke-width="4"/>'
        +'<ellipse cx="377" cy="126" rx="29" ry="20" fill="#F9A8D4" stroke="#BE185D" stroke-width="4"/>'
        +'<path d="M322 139 Q338 158 348 183 M376 139 Q360 158 350 183" fill="none" stroke="#BE185D" stroke-width="6" stroke-linecap="round"/>'
        +'<path d="M349 177 Q319 188 324 220 Q349 248 374 220 Q379 188 349 177Z" fill="#FDBA74" stroke="#C2410C" stroke-width="5"/>'
        +'<path d="M349 220 V346" fill="none" stroke="#94A3B8" stroke-width="10" stroke-linecap="round"/>'
        +'</g>'
        +'<g transform="translate(515 104)">'
        +'<rect width="190" height="181" rx="20" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="95" y="25" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">男性生殖细胞通路</text>'
        +'<text x="18" y="59" font-size="12" font-weight="900" fill="#92400E">1　睾丸形成精子</text>'
        +'<text x="18" y="88" font-size="12" font-weight="900" fill="#C2410C">2　附睾成熟和储存</text>'
        +'<text x="18" y="117" font-size="12" font-weight="900" fill="#6D28D9">3　输精管运输</text>'
        +'<text x="18" y="146" font-size="12" font-weight="900" fill="#BE185D">4　附属腺参与形成精液</text>'
        +'<text x="18" y="175" font-size="12" font-weight="900" fill="#475569">5　经尿道排出</text>'
        +'</g>';
    }

    function femaleStructure(sideIndex){
      var activeLeft=sideIndex===0;

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M349 181 C314 144 267 142 224 174" fill="none" stroke="#DB2777" stroke-width="12" stroke-linecap="round"/>'
        +'<path d="M349 181 C384 144 431 142 474 174" fill="none" stroke="#DB2777" stroke-width="12" stroke-linecap="round"/>'
        +'<path d="M224 174 Q191 157 170 184 M474 174 Q507 157 528 184" fill="none" stroke="#F472B6" stroke-width="5" stroke-linecap="round"/>'
        +'<ellipse cx="157" cy="192" rx="31" ry="23" fill="'+(activeLeft?'#F9A8D4':'#FBCFE8')+'" stroke="#BE185D" stroke-width="'+(activeLeft?6:4)+'"/>'
        +'<ellipse cx="541" cy="192" rx="31" ry="23" fill="'+(!activeLeft?'#F9A8D4':'#FBCFE8')+'" stroke="#BE185D" stroke-width="'+(!activeLeft?6:4)+'"/>'
        +'<path d="M349 177 C304 185 296 236 312 279 C323 309 375 309 386 279 C402 236 394 185 349 177Z" fill="#FBCFE8" stroke="#DB2777" stroke-width="6"/>'
        +'<path d="M349 207 C325 211 325 245 333 273 C338 289 360 289 365 273 C373 245 373 211 349 207Z" fill="#FFF1F2" stroke="#FB7185" stroke-width="4"/>'
        +'<path d="M337 298 Q349 318 361 298 V340 H337Z" fill="#F9A8D4" stroke="#BE185D" stroke-width="5"/>'
        +'<path d="M337 340 V369 M361 340 V369" stroke="#F472B6" stroke-width="7" stroke-linecap="round"/>'
        +'</g>'
        +'<g transform="translate(515 104)">'
        +'<rect width="190" height="181" rx="20" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="95" y="25" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">女性生殖细胞通路</text>'
        +'<text x="18" y="59" font-size="12" font-weight="900" fill="#BE185D">1　卵巢排出卵细胞</text>'
        +'<text x="18" y="88" font-size="12" font-weight="900" fill="#DB2777">2　卵细胞进入输卵管</text>'
        +'<text x="18" y="117" font-size="12" font-weight="900" fill="#7C3AED">3　输卵管内继续运输</text>'
        +'<text x="18" y="146" font-size="12" font-weight="900" fill="#2563EB">4　受精通常位于输卵管</text>'
        +'<text x="18" y="175" font-size="12" font-weight="900" fill="#475569">5　早期胚胎到达子宫</text>'
        +'</g>';
    }

    function maleLabels(){
      if(!showLabels){
        return '';
      }

      return ''
        +'<path d="M322 305 L200 334" stroke="#B45309" stroke-width="2.5"/>'
        +'<text x="137" y="340" font-size="13" font-weight="900" fill="#92400E">睾丸</text>'
        +'<path d="M299 271 L191 268" stroke="#EA580C" stroke-width="2.5"/>'
        +'<text x="121" y="272" font-size="13" font-weight="900" fill="#C2410C">附睾</text>'
        +'<path d="M278 190 L176 157" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="101" y="153" font-size="13" font-weight="900" fill="#6D28D9">输精管</text>'
        +'<path d="M321 126 L218 94" stroke="#BE185D" stroke-width="2.5"/>'
        +'<text x="131" y="90" font-size="13" font-weight="900" fill="#9D174D">精囊等附属腺</text>'
        +'<path d="M324 205 L226 219" stroke="#C2410C" stroke-width="2.5"/>'
        +'<text x="172" y="225" font-size="13" font-weight="900" fill="#9A3412">前列腺</text>'
        +'<path d="M349 320 L432 346" stroke="#64748B" stroke-width="2.5"/>'
        +'<text x="440" y="352" font-size="13" font-weight="900" fill="#475569">尿道</text>';
    }

    function femaleLabels(sideIndex){
      if(!showLabels){
        return '';
      }

      var activeX=sideIndex===0?157:541;

      return ''
        +'<path d="M'+activeX+' 192 L'
        +(sideIndex===0?69:628)+' 216" stroke="#BE185D" stroke-width="2.5"/>'
        +'<text x="'+(sideIndex===0?25:636)+'" y="222" font-size="13" font-weight="900" fill="#9D174D">卵巢</text>'
        +'<path d="M259 161 L177 113" stroke="#DB2777" stroke-width="2.5"/>'
        +'<text x="100" y="109" font-size="13" font-weight="900" fill="#BE185D">输卵管</text>'
        +'<path d="M391 227 L468 240" stroke="#DB2777" stroke-width="2.5"/>'
        +'<text x="477" y="245" font-size="13" font-weight="900" fill="#BE185D">子宫</text>'
        +'<path d="M359 310 L443 312" stroke="#BE185D" stroke-width="2.5"/>'
        +'<text x="451" y="317" font-size="13" font-weight="900" fill="#9D174D">子宫颈</text>'
        +'<path d="M360 356 L442 376" stroke="#F472B6" stroke-width="2.5"/>'
        +'<text x="450" y="382" font-size="13" font-weight="900" fill="#BE185D">阴道</text>';
    }

    function maleTransport(
      progress,
      amount
    ){
      var points=[
        [322,305,-10],
        [299,272,-62],
        [275,229,-92],
        [271,183,-111],
        [289,142,-151],
        [321,126,5],
        [340,163,74],
        [349,214,90],
        [349,273,90],
        [349,337,90]
      ];

      var count=Math.floor(
        4+amount/8
      );

      var html='';

      for(var i=0;i<count;i++){
        var local=clamp(
          progress-i*.045,
          0,
          1
        );

        var scaled=local*(points.length-1);
        var index=Math.min(
          points.length-2,
          Math.floor(scaled)
        );

        var fraction=scaled-index;
        var a=points[index];
        var b=points[index+1];

        var x=a[0]+(b[0]-a[0])*fraction;
        var y=a[1]+(b[1]-a[1])*fraction;
        var rotation=a[2]+(b[2]-a[2])*fraction;

        html+=spermShape(
          x.toFixed(1),
          y.toFixed(1),
          rotation.toFixed(1),
          .72,
          local>0?(.45+.5*local):0
        );
      }

      return html
        +'<path class="hr-flow" d="M322 305 Q285 292 299 250 C249 210 258 139 321 126 Q344 140 349 184 V344" fill="none" stroke="#7C3AED" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)" opacity=".72"/>';
    }

    function femaleTransport(
      sideIndex,
      eggProgress,
      spermProgress,
      amount
    ){
      var left=sideIndex===0;
      var ovaryX=left?157:541;
      var tubeEntryX=left?224:474;
      var meetingX=left?280:418;
      var uterusX=349;

      var eggX;
      var eggY;

      if(eggProgress<.65){
        var first=eggProgress/.65;

        eggX=ovaryX+(meetingX-ovaryX)*first;
        eggY=192+(157-192)*Math.sin(first*Math.PI*.65);
      }else{
        var second=(eggProgress-.65)/.35;

        eggX=meetingX+(uterusX-meetingX)*second;
        eggY=157+(190-157)*second;
      }

      var spermPoints=left
        ?[
          [349,369,-90],
          [349,318,-90],
          [349,270,-90],
          [349,215,-90],
          [326,177,-135],
          [290,157,-165],
          [meetingX,157,180]
        ]
        :[
          [349,369,-90],
          [349,318,-90],
          [349,270,-90],
          [349,215,-90],
          [372,177,-45],
          [408,157,-15],
          [meetingX,157,0]
        ];

      var spermCount=Math.floor(
        3+amount/10
      );

      var spermHTML='';

      for(var i=0;i<spermCount;i++){
        var local=clamp(
          spermProgress-i*.055,
          0,
          1
        );

        var scaled=local*(spermPoints.length-1);
        var index=Math.min(
          spermPoints.length-2,
          Math.floor(scaled)
        );

        var fraction=scaled-index;
        var a=spermPoints[index];
        var b=spermPoints[index+1];

        var sx=a[0]+(b[0]-a[0])*fraction;
        var sy=a[1]+(b[1]-a[1])*fraction;
        var rotation=a[2]+(b[2]-a[2])*fraction;

        spermHTML+=spermShape(
          sx.toFixed(1),
          sy.toFixed(1),
          rotation.toFixed(1),
          .58,
          local>0?(.35+.58*local):0
        );
      }

      var eggHTML=''
        +'<g transform="translate('+eggX.toFixed(1)+' '+eggY.toFixed(1)+')">'
        +'<circle r="17" fill="#FDE68A" stroke="#B45309" stroke-width="4"/>'
        +'<circle r="7" fill="#F9A8D4" stroke="#BE185D" stroke-width="2"/>'
        +'<circle class="hr-pulse" r="25" fill="none" stroke="#F59E0B" stroke-width="3" stroke-dasharray="6 5"/>'
        +'</g>';

      var meetingHTML='';

      if(
        eggProgress>.52
        && spermProgress>.82
      ){
        meetingHTML=''
          +'<circle class="hr-pulse" cx="'+meetingX+'" cy="157" r="34" fill="none" stroke="#16A34A" stroke-width="5" stroke-dasharray="7 6"/>'
          +'<text x="'+meetingX+'" y="116" text-anchor="middle" font-size="12" font-weight="900" fill="#047857">通常受精位置</text>';
      }

      return ''
        +'<path class="hr-flow" d="M'+ovaryX+' 192 Q'+tubeEntryX+' 143 '+meetingX+' 157 Q'
        +((meetingX+uterusX)/2)+' 161 '+uterusX+' 190" fill="none" stroke="#DB2777" stroke-width="5" marker-end="url(#${rootId}-arrow-pink)" opacity=".72"/>'
        +'<path class="hr-flow" d="M349 369 V270 Q349 207 '+meetingX+' 157" fill="none" stroke="#2563EB" stroke-width="4" marker-end="url(#${rootId}-arrow-blue)" opacity=".65"/>'
        +eggHTML
        +spermHTML
        +meetingHTML;
    }

    function compareScene(){
      return ''
        +'<g transform="translate(42 92)">'
        +'<rect width="310" height="255" rx="22" fill="#EFF6FF" stroke="#BFDBFE" stroke-width="3"/>'
        +'<text x="155" y="29" text-anchor="middle" font-size="17" font-weight="900" fill="#1D4ED8">男性生殖系统</text>'
        +'<ellipse cx="82" cy="103" rx="28" ry="38" fill="#FDE68A" stroke="#B45309" stroke-width="4"/>'
        +'<path d="M78 67 Q118 53 133 88 C158 130 137 171 181 190" fill="none" stroke="#7C3AED" stroke-width="7" stroke-linecap="round"/>'
        +'<path d="M181 190 V218" stroke="#64748B" stroke-width="8" stroke-linecap="round"/>'
        +'<text x="28" y="225" font-size="11.5" font-weight="900" fill="#334155">睾丸形成精子</text>'
        +'<text x="28" y="244" font-size="11.5" font-weight="900" fill="#334155">附睾成熟储存→输精管运输</text>'
        +'<path class="hr-flow" d="M83 102 Q126 80 133 88 C158 130 137 171 181 190 V218" fill="none" stroke="#2563EB" stroke-width="4" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'</g>'
        +'<g transform="translate(397 92)">'
        +'<rect width="310" height="255" rx="22" fill="#FDF2F8" stroke="#FBCFE8" stroke-width="3"/>'
        +'<text x="155" y="29" text-anchor="middle" font-size="17" font-weight="900" fill="#BE185D">女性生殖系统</text>'
        +'<ellipse cx="57" cy="103" rx="25" ry="19" fill="#F9A8D4" stroke="#BE185D" stroke-width="4"/>'
        +'<ellipse cx="253" cy="103" rx="25" ry="19" fill="#F9A8D4" stroke="#BE185D" stroke-width="4"/>'
        +'<path d="M70 103 Q104 57 155 89 Q206 57 240 103" fill="none" stroke="#DB2777" stroke-width="8" stroke-linecap="round"/>'
        +'<path d="M155 87 C120 103 123 157 138 188 C146 205 164 205 172 188 C187 157 190 103 155 87Z" fill="#FBCFE8" stroke="#DB2777" stroke-width="5"/>'
        +'<text x="28" y="225" font-size="11.5" font-weight="900" fill="#334155">卵巢排卵→输卵管运输</text>'
        +'<text x="28" y="244" font-size="11.5" font-weight="900" fill="#334155">输卵管受精→早期胚胎到达子宫</text>'
        +'<path class="hr-flow" d="M57 103 Q104 57 155 89" fill="none" stroke="#DB2777" stroke-width="4" marker-end="url(#${rootId}-arrow-pink)"/>'
        +'</g>';
    }

    function maleStage(progress){
      if(progress<.15)return '精子形成';
      if(progress<.33)return '附睾成熟';
      if(progress<.72)return '输精管运输';
      return '经尿道排出';
    }

    function femaleStage(
      eggProgress,
      spermProgress
    ){
      if(eggProgress<.12)return '卵巢排卵';
      if(eggProgress<.58)return '输卵管运输';
      if(
        eggProgress>.52
        && spermProgress>.82
      )return '接近受精位置';

      if(eggProgress<.90)return '向子宫方向运输';
      return '到达子宫附近';
    }

    function update(){
      var systemIndex=clamp(
        Math.round(Number(systemInput.value)),
        0,
        1
      );

      var amount=Number(amountInput.value);
      var efficiency=Number(efficiencyInput.value);
      var time=Number(timeInput.value);

      var sideIndex=clamp(
        Math.round(Number(sideInput.value)),
        0,
        1
      );

      var systemName=systemIndex===0
        ?'男性生殖系统'
        :'女性生殖系统';

      systemValue.textContent=systemName;
      amountValue.textContent=amount.toFixed(0)+'%';
      efficiencyValue.textContent=efficiency.toFixed(0)+'%';
      timeValue.textContent=time.toFixed(0)+'%';
      sideValue.textContent=sideIndex===0?'左侧卵巢':'右侧卵巢';

      for(var i=0;i<systemButtons.length;i++){
        systemButtons[i].classList.toggle(
          'active',
          systemButtons[i].getAttribute('data-system-button')
            ===(systemIndex===0?'male':'female')
        );
      }

      for(var j=0;j<modeButtons.length;j++){
        modeButtons[j].classList.toggle(
          'active',
          modeButtons[j].getAttribute('data-mode')===mode
        );
      }

      setVisibleLabelState();

      autoButton.textContent=automatic
        ?'运输推进：运行中'
        :'运输推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var efficiencyFactor=.18+.82*efficiency/100;

      var progress=clamp(
        time/100*efficiencyFactor,
        0,
        1
      );

      var eggProgress=clamp(
        time/100*(.30+.70*efficiencyFactor),
        0,
        1
      );

      var spermProgress=clamp(
        time/100*(.18+.82*efficiencyFactor),
        0,
        1
      );

      root.style.setProperty(
        '--hr-speed',
        clamp(
          2.45-progress*1.6,
          .65,
          2.35
        ).toFixed(2)+'s'
      );

      currentSystemText.textContent=mode==='compare'
        ?'两性比较'
        :systemIndex===0
          ?'男性'
          :'女性';

      transportIndexText.textContent=(progress*100).toFixed(0)+'%';

      dynamic.innerHTML='';
      labels.innerHTML='';
      gametes.innerHTML='';

      if(mode==='compare'){
        title.textContent='男性与女性生殖系统比较';
        summary.textContent='比较生殖腺、输送管道、生殖细胞及其运输方向。';
        stageNote.textContent='两类系统共同参与有性生殖，但结构和功能分工不同。';
        currentStageText.textContent='结构比较';
        transportIndexText.textContent='—';

        dynamic.innerHTML=compareScene();

        result.innerHTML=
          '男性生殖系统主要负责精子的形成、成熟、储存和排出；女性生殖系统主要负责卵细胞形成与排出，并为受精、胚胎发育和分娩提供相应结构。'
          +'<br>生殖细胞形成、排出、运输、受精和着床是相互衔接但不同的环节。';

        return;
      }

      if(systemIndex===0){
        dynamic.innerHTML=maleStructure();
        labels.innerHTML=maleLabels();

        title.textContent=mode==='structure'
          ?'男性生殖系统主要结构'
          :'精子的形成、成熟与运输';

        summary.textContent=mode==='structure'
          ?'观察睾丸、附睾、输精管、附属腺和尿道。'
          :'精子由睾丸形成，在附睾进一步成熟并沿输精管运输。';

        currentStageText.textContent=mode==='structure'
          ?'结构观察'
          :maleStage(progress);

        stageNote.textContent=mode==='structure'
          ?'睾丸是男性主要生殖器官，能够产生精子并分泌雄性激素。'
          :'精子形成、附睾成熟、输精管运输和排出是不同环节。';

        if(mode==='transport'){
          gametes.innerHTML=maleTransport(
            progress,
            amount
          );
        }

        var maleCondition=efficiency<20
          ?'当前相对运输条件较低，运输进度明显减慢。'
          :amount<15
            ?'当前显示的精子数量较少，只用于观察路径。'
            :time<15
              ?'运输过程时间较短，精子主要位于形成或成熟环节。'
              :'当前参数可清楚观察精子从睾丸、附睾到输精管和尿道的相对运输过程。';

        result.innerHTML=
          '精子在睾丸曲细精管中形成，并在附睾中进一步成熟和储存；随后可经输精管、射精管和尿道排出。'
          +'<br>'+maleCondition
          +' 图中数量和速度不代表真实人体测量值。';

        return;
      }

      dynamic.innerHTML=femaleStructure(sideIndex);
      labels.innerHTML=femaleLabels(sideIndex);

      title.textContent=mode==='structure'
        ?'女性生殖系统主要结构'
        :'卵细胞与精子的体内运输';

      summary.textContent=mode==='structure'
        ?'观察卵巢、输卵管、子宫、子宫颈和阴道。'
        :'卵细胞从卵巢进入输卵管，精子可由女性生殖道到达输卵管。';

      currentStageText.textContent=mode==='structure'
        ?'结构观察'
        :femaleStage(
          eggProgress,
          spermProgress
        );

      stageNote.textContent=mode==='structure'
        ?'卵巢产生卵细胞并分泌雌性激素；子宫为胚胎和胎儿发育提供场所。'
        :'人体受精通常发生在输卵管，而胚胎着床通常发生在子宫。';

      if(mode==='transport'){
        gametes.innerHTML=femaleTransport(
          sideIndex,
          eggProgress,
          spermProgress,
          amount
        );
      }

      var femaleCondition=efficiency<20
        ?'当前相对运输条件较低，卵细胞和精子的运输进度明显减慢。'
        :time<15
          ?'运输时间较短，卵细胞刚从卵巢排出或尚未进入输卵管。'
          :eggProgress>.52&&spermProgress>.82
            ?'卵细胞和部分精子已接近输卵管内通常发生受精的区域。'
            :'卵细胞沿输卵管向子宫方向运输，精子则从女性生殖道向输卵管方向运动。';

      result.innerHTML=
        '卵细胞由卵巢排出后进入输卵管；精子进入女性生殖道后，可经过阴道、子宫颈和子宫到达输卵管。'
        +'<br>'+femaleCondition
        +' 受精通常位于输卵管壶腹部附近，不是在子宫内完成。';
    }

    for(var i=0;i<systemButtons.length;i++){
      systemButtons[i].onclick=function(){
        var selected=this.getAttribute('data-system-button');

        systemInput.value=selected==='male'
          ?'0'
          :'1';

        mode='structure';
        update();
      };
    }

    for(var j=0;j<modeButtons.length;j++){
      modeButtons[j].onclick=function(){
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

    systemInput.oninput=function(){
      mode='structure';
      update();
    };

    amountInput.oninput=update;
    efficiencyInput.oninput=update;
    timeInput.oninput=update;
    sideInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
