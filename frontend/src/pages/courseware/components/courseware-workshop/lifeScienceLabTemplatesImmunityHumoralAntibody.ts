/**
 * lifeScienceLabTemplatesImmunityHumoralAntibody.ts
 *
 * 平面生命科学实验室：体液免疫与抗体形成。
 *
 * 教学目标：
 * 1. 认识抗原呈递、辅助性T细胞信号和B细胞激活之间的关系；
 * 2. 观察特异性B细胞克隆选择、增殖和分化；
 * 3. 理解浆细胞分泌抗体，记忆B细胞参与后续更快的再次应答；
 * 4. 观察抗体的中和、凝集和调理作用；
 * 5. 理解抗体能够特异性识别抗原，但不等于直接消灭所有病原体；
 * 6. 区分体液免疫和以清除感染细胞为主的细胞免疫。
 *
 * 科学边界：
 * 1. B细胞表面受体能够识别特定抗原表位；
 * 2. 许多蛋白质抗原引发充分的B细胞应答时，需要辅助性T细胞提供协同信号；
 * 3. 并非所有抗原都以完全相同的方式激活B细胞，本模型采用典型过程进行教学简化；
 * 4. 被激活的特异性B细胞发生克隆增殖，并分化为浆细胞和记忆B细胞；
 * 5. 浆细胞能够大量分泌具有相同抗原特异性的抗体；
 * 6. 抗体可通过中和、凝集、调理和激活补体等方式协助清除抗原；
 * 7. 抗体与抗原结合不等于抗体直接吞噬或直接杀死所有病原体；
 * 8. 体液免疫主要作用于细胞外抗原、病原体和毒素；
 * 9. 被感染宿主细胞的识别和清除主要依赖细胞免疫等机制；
 * 10. 图中抗原数量、抗体水平和应答强度均为相对教学指标；
 * 11. 本模板只用于生物学教学，不用于疾病诊断、抗体检测解释或医学建议。
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

function humoralAntibodyStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#DBEAFE);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FBFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#8B5CF6}'
    + '#' + rootId + ' .ha-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .ha-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ha-button{min-height:30px;padding:3px;border:1px solid #A78BFA;border-radius:8px;background:#fff;color:#5B21B6;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .ha-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.13)}'
    + '#' + rootId + ' .ha-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ha-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ha-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .ha-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ha-card b{display:block;font-size:13px;color:#6D28D9}'
    + '#' + rootId + ' .ha-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ha-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--ha-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ha-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_HUMORAL_ANTIBODY:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-humoral-immunity-antibody',
    group: '🛡️ 免疫与疾病防御',
    name: '体液免疫与抗体形成',
    emoji: '🧪',
    desc: '调节抗原负荷、辅助性T细胞协同、B细胞激活、浆细胞分化和应答时间，观察抗体形成与作用',
    params: [
      {
        key: 'antigenLoad',
        label: '抗原相对数量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 66,
      },
      {
        key: 'helperTSignal',
        label: '辅助性T细胞协同',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 74,
      },
      {
        key: 'bCellActivation',
        label: 'B细胞激活水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'plasmaDifferentiation',
        label: '浆细胞分化水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'responseTime',
        label: '免疫应答时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 48,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const antigenLoad = num(params, 'antigenLoad', 66)
      const helperTSignal = num(params, 'helperTSignal', 74)
      const bCellActivation = num(params, 'bCellActivation', 78)
      const plasmaDifferentiation = num(
        params,
        'plasmaDifferentiation',
        76,
      )
      const responseTime = num(params, 'responseTime', 48)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${humoralAntibodyStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧪 体液免疫与抗体形成</div>
    <div class="bl-note">抗体水平与应答强度均为相对教学指标</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>抗原相对数量</span>
          <span class="bl-value" data-antigen-value></span>
        </div>
        <input
          data-antigen
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(antigenLoad)}"
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
          value="${n(helperTSignal)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>B细胞激活水平</span>
          <span class="bl-value" data-b-cell-value></span>
        </div>
        <input
          data-b-cell
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(bCellActivation)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>浆细胞分化水平</span>
          <span class="bl-value" data-plasma-value></span>
        </div>
        <input
          data-plasma
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(plasmaDifferentiation)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>免疫应答时间</span>
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

      <div class="ha-subtitle">观察方式</div>

      <div class="ha-buttons">
        <button
          type="button"
          class="ha-button active"
          data-mode="activation"
        >抗原识别与激活</button>

        <button
          type="button"
          class="ha-button"
          data-mode="clonal"
        >克隆增殖与分化</button>

        <button
          type="button"
          class="ha-button"
          data-mode="antibody"
        >抗体形成与作用</button>

        <button
          type="button"
          class="ha-button"
          data-mode="boundary"
        >体液免疫边界</button>
      </div>

      <button
        type="button"
        class="ha-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="ha-toggle"
        data-auto
      >应答推进：运行中</button>

      <div class="ha-status">
        <div class="ha-card">
          <b data-activation-index></b>
          <span>B细胞激活</span>
        </div>

        <div class="ha-card">
          <b data-antibody-index></b>
          <span>抗体水平</span>
        </div>

        <div class="ha-card">
          <b data-remaining></b>
          <span>剩余抗原</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="体液免疫与抗体形成互动示意图"
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
          fill="#5B21B6"
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
            fill="#FAF5FF"
            stroke="#DDD6FE"
            stroke-width="2"
          />

          <text
            x="108"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#5B21B6"
          >关键边界</text>

          <text
            x="108"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#4C1D95"
          >抗体特异性结合抗原</text>

          <text
            x="108"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#4C1D95"
          >不等于直接杀死所有病原体</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#5B21B6"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var antigenInput=root.querySelector('[data-antigen]');
    var helperInput=root.querySelector('[data-helper]');
    var bCellInput=root.querySelector('[data-b-cell]');
    var plasmaInput=root.querySelector('[data-plasma]');
    var timeInput=root.querySelector('[data-time]');

    var antigenValue=root.querySelector('[data-antigen-value]');
    var helperValue=root.querySelector('[data-helper-value]');
    var bCellValue=root.querySelector('[data-b-cell-value]');
    var plasmaValue=root.querySelector('[data-plasma-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var activationText=root.querySelector('[data-activation-index]');
    var antibodyText=root.querySelector('[data-antibody-index]');
    var remainingText=root.querySelector('[data-remaining]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');
    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='activation';
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
      },780);
    }

    function antigenShape(
      x,
      y,
      size,
      color,
      opacity
    ){
      var points='';

      for(var i=0;i<10;i++){
        var angle=-Math.PI/2+Math.PI*2*i/10;
        var radius=i%2===0
          ?size
          :size*.58;

        points+=(x+Math.cos(angle)*radius).toFixed(1)
          +','
          +(y+Math.sin(angle)*radius).toFixed(1)
          +' ';
      }

      return '<polygon points="'+points.trim()
        +'" fill="'+color
        +'" stroke="#991B1B" stroke-width="2.5" opacity="'+opacity+'"/>';
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
        +'<path d="M0 25 V6 M0 8 L-17 -12 M0 8 L17 -12" fill="none" stroke="#8B5CF6" stroke-width="7" stroke-linecap="round"/>'
        +'<circle cx="-18" cy="-14" r="4" fill="#C4B5FD"/>'
        +'<circle cx="18" cy="-14" r="4" fill="#C4B5FD"/>'
        +'</g>';
    }

    function bCellShape(
      x,
      y,
      radius,
      active,
      label
    ){
      var receptors='';

      for(var i=0;i<10;i++){
        var angle=Math.PI*2*i/10;
        var x1=x+Math.cos(angle)*(radius-2);
        var y1=y+Math.sin(angle)*(radius-2);
        var x2=x+Math.cos(angle)*(radius+10);
        var y2=y+Math.sin(angle)*(radius+10);

        receptors+='<path d="M'+x1.toFixed(1)+' '+y1.toFixed(1)
          +' L'+x2.toFixed(1)+' '+y2.toFixed(1)
          +'" stroke="#7C3AED" stroke-width="3" stroke-linecap="round"/>';
      }

      return receptors
        +'<circle cx="'+x+'" cy="'+y+'" r="'+radius
        +'" fill="'+(active?'#EDE9FE':'#F8FAFC')
        +'" stroke="'+(active?'#7C3AED':'#94A3B8')
        +'" stroke-width="'+(active?6:4)+'"/>'
        +'<circle cx="'+x+'" cy="'+y+'" r="'+(radius*.36)
        +'" fill="'+(active?'#A78BFA':'#CBD5E1')+'"/>'
        +'<text x="'+x+'" y="'+(y+radius+25)
        +'" text-anchor="middle" font-size="12" font-weight="900" fill="'
        +(active?'#5B21B6':'#475569')
        +'">'+label+'</text>';
    }

    function helperTCellShape(
      x,
      y,
      signal
    ){
      return ''
        +'<circle cx="'+x+'" cy="'+y+'" r="43" fill="#DCFCE7" stroke="#16A34A" stroke-width="5"/>'
        +'<circle cx="'+x+'" cy="'+y+'" r="16" fill="#86EFAC" stroke="#15803D" stroke-width="3"/>'
        +'<text x="'+x+'" y="'+(y+6)+'" text-anchor="middle" font-size="18" font-weight="900" fill="#166534">Th</text>'
        +'<circle class="ha-pulse" cx="'+x+'" cy="'+y+'" r="'+(49+signal*.13)
        +'" fill="none" stroke="#22C55E" stroke-width="4" stroke-dasharray="7 6" opacity="'+(.25+.70*signal/100)+'"/>';
    }

    function plasmaCellShape(
      x,
      y,
      scale,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<ellipse cx="0" cy="0" rx="43" ry="55" fill="#DBEAFE" stroke="#2563EB" stroke-width="5"/>'
        +'<ellipse cx="-10" cy="1" rx="17" ry="23" fill="#93C5FD" stroke="#1D4ED8" stroke-width="3"/>'
        +'<path d="M9 -28 Q33 -17 17 0 Q36 14 12 29" fill="none" stroke="#60A5FA" stroke-width="6" stroke-linecap="round"/>'
        +'<text x="0" y="78" text-anchor="middle" font-size="12" font-weight="900" fill="#1E40AF">浆细胞</text>'
        +'</g>';
    }

    function memoryCellShape(
      x,
      y,
      scale,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle r="40" fill="#ECFDF5" stroke="#10B981" stroke-width="5"/>'
        +'<circle r="15" fill="#6EE7B7" stroke="#047857" stroke-width="3"/>'
        +'<text x="0" y="6" text-anchor="middle" font-size="17" font-weight="900" fill="#047857">MB</text>'
        +'<text x="0" y="62" text-anchor="middle" font-size="12" font-weight="900" fill="#047857">记忆B细胞</text>'
        +'</g>';
    }

    function renderActivation(
      antigen,
      helper,
      bCell,
      progress
    ){
      var antigenCount=Math.floor(
        4+antigen/10
      );

      var antigenHTML='';

      for(var i=0;i<antigenCount;i++){
        var ax=58+(i%5)*46;
        var ay=129+Math.floor(i/5)*52;

        antigenHTML+=antigenShape(
          ax,
          ay,
          13+i%3,
          i%2===0?'#FCA5A5':'#FDBA74',
          .72
        );
      }

      var activationProgress=clamp(
        progress
        *Math.sqrt(
          (.20+.80*helper/100)
          *(.20+.80*bCell/100)
        ),
        0,
        1
      );

      var selectedX=440;
      var selectedY=211;

      return ''
        +'<rect x="28" y="89" width="255" height="254" rx="24" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="155" y="118" text-anchor="middle" font-size="15" font-weight="900" fill="#9A3412">抗原与抗原呈递信号</text>'
        +antigenHTML
        +'<path class="ha-flow" d="M265 207 C307 189 335 185 372 199" fill="none" stroke="#7C3AED" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<g filter="url(#${rootId}-shadow)">'
        +bCellShape(
          selectedX,
          selectedY,
          48,
          activationProgress>.28,
          '特异性B细胞'
        )
        +helperTCellShape(
          585,
          143,
          helper
        )
        +'</g>'
        +'<path class="ha-flow" d="M551 169 C520 185 507 198 488 207" fill="none" stroke="#16A34A" stroke-width="'+(3+helper/22)+'" marker-end="url(#${rootId}-arrow-green)" opacity="'+(.25+.75*helper/100)+'"/>'
        +'<text x="571" y="219" text-anchor="middle" font-size="11" font-weight="900" fill="#166534">辅助性T细胞协同信号</text>'
        +'<g transform="translate(520 272)">'
        +'<rect width="196" height="66" rx="16" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<text x="98" y="23" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">典型激活条件</text>'
        +'<text x="98" y="43" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">抗原识别 + 协同信号</text>'
        +'<text x="98" y="58" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">本图为教学简化</text>'
        +'</g>'
        +'<circle class="ha-pulse" cx="'+selectedX+'" cy="'+selectedY+'" r="'+(56+activationProgress*18)
        +'" fill="none" stroke="#8B5CF6" stroke-width="5" stroke-dasharray="8 7" opacity="'+activationProgress.toFixed(2)+'"/>';
    }

    function renderClonal(
      activation,
      plasma,
      progress
    ){
      var cloneCount=Math.floor(
        2+activation/11*progress
      );

      cloneCount=clamp(
        cloneCount,
        2,
        10
      );

      var clones='';

      for(var i=0;i<cloneCount;i++){
        var angle=Math.PI*2*i/cloneCount;
        var x=265+Math.cos(angle)*110;
        var y=215+Math.sin(angle)*84;

        clones+=bCellShape(
          x.toFixed(1),
          y.toFixed(1),
          27,
          true,
          ''
        );
      }

      var plasmaOpacity=clamp(
        progress*plasma/100,
        0,
        1
      );

      var memoryOpacity=clamp(
        progress*(.35+.65*activation/100),
        0,
        1
      );

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<circle cx="265" cy="215" r="132" fill="#FAF5FF" stroke="#C4B5FD" stroke-width="4"/>'
        +clones
        +'<circle cx="265" cy="215" r="45" fill="#EDE9FE" stroke="#7C3AED" stroke-width="6"/>'
        +'<text x="265" y="208" text-anchor="middle" font-size="14" font-weight="900" fill="#5B21B6">克隆选择</text>'
        +'<text x="265" y="231" text-anchor="middle" font-size="12" font-weight="900" fill="#5B21B6">与增殖</text>'
        +'</g>'
        +'<path class="ha-flow" d="M397 186 C444 158 464 153 496 167" fill="none" stroke="#2563EB" stroke-width="5" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="ha-flow" d="M394 253 C442 286 464 292 497 280" fill="none" stroke="#16A34A" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>'
        +plasmaCellShape(
          581,
          168,
          .93,
          plasmaOpacity.toFixed(2)
        )
        +memoryCellShape(
          581,
          286,
          .93,
          memoryOpacity.toFixed(2)
        )
        +'<g transform="translate(55 350)">'
        +'<rect width="610" height="36" rx="12" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<text x="305" y="23" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">同一特异性B细胞克隆增殖后，可分化为浆细胞和记忆B细胞</text>'
        +'</g>';
    }

    function renderAntibody(
      antigen,
      antibodyLevel,
      progress
    ){
      var antigenCount=Math.floor(
        4+antigen/9
      );

      var antibodyCount=Math.floor(
        5+antibodyLevel/7
      );

      var antigenHTML='';
      var antibodyHTML='';

      for(var i=0;i<antigenCount;i++){
        var ax=442+(i%5)*49;
        var ay=127+Math.floor(i/5)*62;

        antigenHTML+=antigenShape(
          ax,
          ay,
          13+i%3,
          i%2===0?'#FCA5A5':'#FDBA74',
          .76
        );
      }

      for(var j=0;j<antibodyCount;j++){
        var bx=83+(j%6)*52;
        var by=126+Math.floor(j/6)*60;
        var rotation=(j%5-2)*12;

        antibodyHTML+=antibodyShape(
          bx,
          by,
          .72,
          rotation,
          .45+.5*progress
        );
      }

      var boundCount=Math.floor(
        Math.min(
          antigenCount,
          antibodyLevel/12*progress
        )
      );

      var bindings='';

      for(var k=0;k<boundCount;k++){
        var tx=442+(k%5)*49;
        var ty=127+Math.floor(k/5)*62;

        bindings+=antibodyShape(
          tx-23,
          ty+3,
          .68,
          90,
          .92
        );
      }

      return ''
        +'<rect x="32" y="87" width="350" height="256" rx="24" fill="#FAF5FF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="207" y="116" text-anchor="middle" font-size="15" font-weight="900" fill="#5B21B6">浆细胞分泌抗体</text>'
        +antibodyHTML
        +'<path class="ha-flow" d="M369 215 H418" fill="none" stroke="#7C3AED" stroke-width="6" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<rect x="423" y="87" width="306" height="256" rx="24" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="576" y="116" text-anchor="middle" font-size="15" font-weight="900" fill="#9A3412">抗体与相应抗原结合</text>'
        +antigenHTML
        +bindings
        +'<g transform="translate(55 352)">'
        +'<rect width="640" height="35" rx="12" fill="#EFF6FF" stroke="#BFDBFE" stroke-width="2"/>'
        +'<text x="320" y="23" text-anchor="middle" font-size="11.5" font-weight="900" fill="#1E40AF">抗体可中和、凝集、调理或激活补体，协助其他免疫机制清除抗原</text>'
        +'</g>';
    }

    function renderBoundary(
      antibodyLevel,
      remaining
    ){
      var cards=[
        {
          x:30,
          title:'中和',
          color:'#7C3AED',
          fill:'#F5F3FF',
          line1:'阻断毒素或病毒',
          line2:'与靶细胞结合'
        },
        {
          x:208,
          title:'凝集',
          color:'#DB2777',
          fill:'#FDF2F8',
          line1:'把多个抗原颗粒',
          line2:'连接成较大聚集物'
        },
        {
          x:386,
          title:'调理',
          color:'#D97706',
          fill:'#FFFBEB',
          line1:'标记抗原',
          line2:'促进吞噬细胞识别'
        },
        {
          x:564,
          title:'补体协同',
          color:'#16A34A',
          fill:'#ECFDF5',
          line1:'可参与补体激活',
          line2:'但过程更加复杂'
        }
      ];

      var html='';

      for(var i=0;i<cards.length;i++){
        var card=cards[i];

        html+='<g transform="translate('+card.x+' 96)">'
          +'<rect width="158" height="176" rx="20" fill="'+card.fill
          +'" stroke="'+card.color+'" stroke-width="3"/>'
          +'<circle cx="79" cy="53" r="34" fill="#FFFFFF" stroke="'+card.color+'" stroke-width="4"/>'
          +antibodyShape(
            79,
            55,
            .54,
            0,
            1
          )
          +'<text x="79" y="108" text-anchor="middle" font-size="15" font-weight="900" fill="'+card.color+'">'
          +card.title
          +'</text>'
          +'<text x="79" y="136" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'
          +card.line1
          +'</text>'
          +'<text x="79" y="156" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'
          +card.line2
          +'</text>'
          +'</g>';
      }

      return html
        +'<g transform="translate(63 303)">'
        +'<rect width="306" height="74" rx="18" fill="#EDE9FE" stroke="#C4B5FD" stroke-width="3"/>'
        +'<text x="153" y="25" text-anchor="middle" font-size="13.5" font-weight="900" fill="#5B21B6">体液免疫主要作用对象</text>'
        +'<text x="153" y="48" text-anchor="middle" font-size="11.5" font-weight="800" fill="#4C1D95">细胞外抗原、病原体和毒素</text>'
        +'<text x="153" y="65" text-anchor="middle" font-size="10.5" font-weight="800" fill="#64748B">相对抗体水平 '+antibodyLevel.toFixed(0)+'</text>'
        +'</g>'
        +'<g transform="translate(391 303)">'
        +'<rect width="306" height="74" rx="18" fill="#EFF6FF" stroke="#BFDBFE" stroke-width="3"/>'
        +'<text x="153" y="25" text-anchor="middle" font-size="13.5" font-weight="900" fill="#1E40AF">体液免疫的边界</text>'
        +'<text x="153" y="48" text-anchor="middle" font-size="11.5" font-weight="800" fill="#1E40AF">感染细胞清除主要依赖细胞免疫等机制</text>'
        +'<text x="153" y="65" text-anchor="middle" font-size="10.5" font-weight="800" fill="#64748B">剩余抗原 '+remaining.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='activation'){
        labels.innerHTML=''
          +'<path d="M439 156 L493 103" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="501" y="101" font-size="13" font-weight="900" fill="#5B21B6">B细胞受体</text>'
          +'<path d="M585 100 L656 78" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="662" y="80" font-size="13" font-weight="900" fill="#166534">辅助性T细胞</text>'
          +'<path d="M197 150 L99 97" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="30" y="94" font-size="13" font-weight="900" fill="#991B1B">抗原</text>';
        return;
      }

      if(modeName==='clonal'){
        labels.innerHTML=''
          +'<path d="M265 83 L265 63" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="208" y="58" font-size="13" font-weight="900" fill="#5B21B6">特异性克隆</text>'
          +'<path d="M581 112 L659 84" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="665" y="87" font-size="13" font-weight="900" fill="#1D4ED8">浆细胞</text>'
          +'<path d="M581 327 L659 345" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="665" y="350" font-size="13" font-weight="900" fill="#166534">记忆B细胞</text>';
        return;
      }

      if(modeName==='antibody'){
        labels.innerHTML=''
          +'<path d="M214 138 L272 87" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="280" y="86" font-size="13" font-weight="900" fill="#5B21B6">抗体</text>'
          +'<path d="M574 144 L651 107" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="658" y="108" font-size="13" font-weight="900" fill="#991B1B">相应抗原</text>'
          +'<path d="M486 220 L566 269" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="574" y="275" font-size="13" font-weight="900" fill="#5B21B6">特异性结合</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M111 96 L111 72" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="61" y="67" font-size="13" font-weight="900" fill="#5B21B6">抗体作用方式</text>'
        +'<path d="M544 303 L544 281" stroke="#2563EB" stroke-width="2.5"/>'
        +'<text x="450" y="276" font-size="13" font-weight="900" fill="#1E40AF">与细胞免疫区分</text>';
    }

    function update(){
      var antigen=Number(antigenInput.value);
      var helper=Number(helperInput.value);
      var bCell=Number(bCellInput.value);
      var plasma=Number(plasmaInput.value);
      var responseTime=Number(timeInput.value);

      antigenValue.textContent=antigen.toFixed(0)+'%';
      helperValue.textContent=helper.toFixed(0)+'%';
      bCellValue.textContent=bCell.toFixed(0)+'%';
      plasmaValue.textContent=plasma.toFixed(0)+'%';
      timeValue.textContent=responseTime.toFixed(0)+'%';

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
        (.12+.88*helper/100)
        *(.12+.88*bCell/100)
        *(.22+.78*progress),
        1/3
      );

      activation=clamp(
        activation,
        0,
        100
      );

      var antibodyLevel=100*Math.pow(
        activation/100
        *(.15+.85*plasma/100)
        *(.18+.82*progress),
        1/3
      );

      antibodyLevel=clamp(
        antibodyLevel,
        0,
        100
      );

      var neutralized=antigen
        *antibodyLevel/100
        *(.25+.75*progress)
        *.78;

      var remaining=clamp(
        antigen-neutralized,
        0,
        100
      );

      activationText.textContent=activation.toFixed(0);
      antibodyText.textContent=antibodyLevel.toFixed(0);
      remainingText.textContent=remaining.toFixed(0);

      root.style.setProperty(
        '--ha-speed',
        clamp(
          2.45-antibodyLevel/75,
          .65,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='activation'){
        title.textContent='抗原识别与特异性B细胞激活';
        summary.textContent='观察抗原识别、辅助性T细胞协同和B细胞激活。';

        dynamic.innerHTML=renderActivation(
          antigen,
          helper,
          bCell,
          progress
        );

        stageNote.textContent=
          '许多蛋白质抗原引起充分应答时，需要抗原识别与辅助性T细胞协同信号。';

        renderLabels(mode);
      }else if(mode==='clonal'){
        title.textContent='B细胞克隆增殖与分化';
        summary.textContent='观察特异性B细胞增殖并分化为浆细胞和记忆B细胞。';

        dynamic.innerHTML=renderClonal(
          activation,
          plasma,
          progress
        );

        stageNote.textContent=
          '同一B细胞克隆产生的抗体通常具有相同的抗原特异性。';

        renderLabels(mode);
      }else if(mode==='antibody'){
        title.textContent='浆细胞分泌抗体并特异性结合抗原';
        summary.textContent='观察抗体产生、中和、凝集和调理等协同作用。';

        dynamic.innerHTML=renderAntibody(
          antigen,
          antibodyLevel,
          progress
        );

        stageNote.textContent=
          '抗体结合抗原后，可协助中和、标记或清除，但不等于直接消灭所有病原体。';

        renderLabels(mode);
      }else{
        title.textContent='体液免疫的作用方式与边界';
        summary.textContent='比较抗体作用对象、作用方式及与细胞免疫的分工。';

        dynamic.innerHTML=renderBoundary(
          antibodyLevel,
          remaining
        );

        stageNote.textContent=
          '体液免疫主要作用于细胞外抗原；感染细胞的清除主要依赖细胞免疫等机制。';

        renderLabels(mode);
      }

      var condition=
        '当前辅助性T细胞协同、B细胞激活和浆细胞分化共同支持抗体形成。';

      if(antigen<10){
        condition=
          '抗原负荷较低，当前只形成较弱的B细胞激活和抗体应答。';
      }else if(helper<18&&antigen>20){
        condition=
          '辅助性T细胞协同信号较低，典型蛋白质抗原引起的B细胞充分激活受到限制。';
      }else if(bCell<18&&antigen>20){
        condition=
          'B细胞激活水平较低，克隆增殖和后续分化受到限制。';
      }else if(plasma<18&&activation>25){
        condition=
          '浆细胞分化水平较低，已经激活的B细胞产生抗体的能力受到限制。';
      }else if(responseTime<15){
        condition=
          '免疫应答时间较短，当前主要处于抗原识别和B细胞早期激活阶段。';
      }else if(remaining>55){
        condition=
          '抗原负荷较高，现有抗体水平尚不足以显著降低剩余抗原。';
      }

      var principle=mode==='activation'
        ?'B细胞表面受体识别特定抗原表位；许多蛋白质抗原还需要辅助性T细胞提供协同信号。'
        :mode==='clonal'
          ?'被激活的特异性B细胞发生克隆增殖，并分化为浆细胞和记忆B细胞。'
          :mode==='antibody'
            ?'浆细胞分泌具有相同抗原特异性的抗体，抗体可通过中和、凝集和调理等方式协助清除抗原。'
            :'体液免疫主要针对细胞外抗原、病原体和毒素；被感染宿主细胞的清除主要依赖细胞免疫等机制。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 抗体与抗原结合不等于抗体直接吞噬或直接杀死所有病原体。';
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

    antigenInput.oninput=update;
    helperInput.oninput=update;
    bCellInput.oninput=update;
    plasmaInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
