/**
 * lifeScienceLabTemplatesExcretionNephronUrine.ts
 *
 * 平面生命科学实验室：肾单位与尿液形成。
 *
 * 教学目标：
 * 1. 认识肾小球、肾小囊、肾小管和集合管等肾单位相关结构；
 * 2. 理解入球小动脉、肾小球毛细血管和出球小动脉之间的血流关系；
 * 3. 观察肾小球滤过形成原尿的过程；
 * 4. 理解血细胞和大分子蛋白质通常不能通过完整的肾小球滤过屏障；
 * 5. 观察葡萄糖、氨基酸、水和无机盐等物质的选择性重吸收；
 * 6. 观察肾小管分泌以及终尿形成和相对浓缩过程；
 * 7. 比较血浆、原尿和终尿的主要成分差异。
 *
 * 科学边界：
 * 1. 肾小球滤过屏障由有孔内皮、基膜和足细胞裂隙膜等结构共同构成；
 * 2. 水、葡萄糖、氨基酸、尿素和部分无机盐等小分子可进入原尿；
 * 3. 血细胞和大分子蛋白质通常不能进入原尿；
 * 4. 原尿可近似理解为不含血细胞且大分子蛋白质很少的血浆滤液；
 * 5. 在正常生理情况下，滤出的葡萄糖和氨基酸通常几乎全部被重吸收；
 * 6. 水和无机盐的重吸收比例会受到机体状态和激素等多种因素影响；
 * 7. 肾小管可将氢离子、钾离子以及部分药物或代谢物分泌进入管腔；
 * 8. 终尿通常含水、尿素、尿酸、肌酐和一定量无机盐，不应含明显血细胞或大量蛋白质；
 * 9. 图中的滤过、重吸收、分泌、尿量和浓缩指数均为相对教学指标；
 * 10. 本模板只用于生物学教学，不用于肾功能判断、尿检解释或医学诊断。
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

function nephronUrineStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BFDBFE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DBEAFE,#ECFDF5);border-bottom:1px solid #BFDBFE}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#1D4ED8}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FBFF;border-right:1px solid #BFDBFE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#2563EB;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#3B82F6}'
    + '#' + rootId + ' .nu-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#1D4ED8}'
    + '#' + rootId + ' .nu-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .nu-button{min-height:30px;padding:3px;border:1px solid #93C5FD;border-radius:8px;background:#fff;color:#1D4ED8;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .nu-button.active{border-color:#2563EB;background:#DBEAFE;box-shadow:0 3px 9px rgba(37,99,235,.13)}'
    + '#' + rootId + ' .nu-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#60A5FA,#2563EB);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .nu-toggle.off{background:#64748B}'
    + '#' + rootId + ' .nu-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .nu-card{padding:6px 3px;border:1px solid #BFDBFE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .nu-card b{display:block;font-size:13px;color:#1D4ED8}'
    + '#' + rootId + ' .nu-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DBEAFE;color:#1E3A8A;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .nu-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--nu-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .nu-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_EXCRETION_NEPHRON_URINE:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-nephron-urine-formation',
    group: '💧 排泄与内环境稳态',
    name: '肾单位与尿液形成',
    emoji: '🫘',
    desc: '调节肾小球压力、滤过屏障、重吸收、分泌、水重吸收和过程时间，观察原尿与终尿形成',
    params: [
      {
        key: 'glomerularPressure',
        label: '肾小球相对压力',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'barrierIntegrity',
        label: '滤过屏障完整度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 92,
      },
      {
        key: 'reabsorptionCapacity',
        label: '选择性重吸收能力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 84,
      },
      {
        key: 'secretionActivity',
        label: '肾小管分泌活性',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 66,
      },
      {
        key: 'waterReabsorption',
        label: '水重吸收水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'processTime',
        label: '尿液形成过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 52,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const glomerularPressure = num(params, 'glomerularPressure', 72)
      const barrierIntegrity = num(params, 'barrierIntegrity', 92)
      const reabsorptionCapacity = num(
        params,
        'reabsorptionCapacity',
        84,
      )
      const secretionActivity = num(params, 'secretionActivity', 66)
      const waterReabsorption = num(params, 'waterReabsorption', 78)
      const processTime = num(params, 'processTime', 52)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${nephronUrineStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🫘 肾单位与尿液形成</div>
    <div class="bl-note">滤过、重吸收、分泌和尿量均为相对教学指标</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>肾小球相对压力</span>
          <span class="bl-value" data-pressure-value></span>
        </div>
        <input data-pressure type="range" min="20" max="100" step="1" value="${n(glomerularPressure)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>滤过屏障完整度</span>
          <span class="bl-value" data-barrier-value></span>
        </div>
        <input data-barrier type="range" min="20" max="100" step="1" value="${n(barrierIntegrity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>选择性重吸收能力</span>
          <span class="bl-value" data-reabsorption-value></span>
        </div>
        <input data-reabsorption type="range" min="0" max="100" step="1" value="${n(reabsorptionCapacity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>肾小管分泌活性</span>
          <span class="bl-value" data-secretion-value></span>
        </div>
        <input data-secretion type="range" min="0" max="100" step="1" value="${n(secretionActivity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>水重吸收水平</span>
          <span class="bl-value" data-water-value></span>
        </div>
        <input data-water type="range" min="0" max="100" step="1" value="${n(waterReabsorption)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>尿液形成过程时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(processTime)}">
      </div>

      <div class="nu-subtitle">观察方式</div>

      <div class="nu-buttons">
        <button type="button" class="nu-button active" data-mode="filtration">肾小球滤过</button>
        <button type="button" class="nu-button" data-mode="reabsorption">选择性重吸收</button>
        <button type="button" class="nu-button" data-mode="secretion">肾小管分泌</button>
        <button type="button" class="nu-button" data-mode="compare">成分比较</button>
      </div>

      <button type="button" class="nu-toggle${showLabels ? '' : ' off'}" data-label-toggle>${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>
      <button type="button" class="nu-toggle" data-auto>过程推进：运行中</button>

      <div class="nu-status">
        <div class="nu-card">
          <b data-filtration-index></b>
          <span>滤过指数</span>
        </div>
        <div class="nu-card">
          <b data-reabsorption-index></b>
          <span>重吸收指数</span>
        </div>
        <div class="nu-card">
          <b data-urine-volume></b>
          <span>终尿相对量</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg viewBox="0 0 760 430" aria-label="肾单位与尿液形成互动示意图">
        <defs>
          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>
          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>
          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>
          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>
          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#1E3A8A" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>
        <text x="24" y="36" data-title font-size="26" font-weight="900" fill="#1D4ED8"></text>
        <text x="24" y="65" data-summary font-size="14" font-weight="800" fill="#475569"></text>

        <g data-dynamic></g>
        <g data-labels></g>

        <g transform="translate(518 337)">
          <rect width="216" height="66" rx="15" fill="#EFF6FF" stroke="#BFDBFE" stroke-width="2"/>
          <text x="108" y="21" text-anchor="middle" font-size="12" font-weight="900" fill="#1D4ED8">关键边界</text>
          <text x="108" y="40" text-anchor="middle" font-size="10.5" font-weight="800" fill="#1E3A8A">血细胞和大蛋白通常不进入原尿</text>
          <text x="108" y="56" text-anchor="middle" font-size="10.5" font-weight="800" fill="#1E3A8A">全部数值仅用于教学比较</text>
        </g>

        <text x="24" y="407" data-stage-note font-size="14" font-weight="900" fill="#1D4ED8"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var pressureInput=root.querySelector('[data-pressure]');
    var barrierInput=root.querySelector('[data-barrier]');
    var reabsorptionInput=root.querySelector('[data-reabsorption]');
    var secretionInput=root.querySelector('[data-secretion]');
    var waterInput=root.querySelector('[data-water]');
    var timeInput=root.querySelector('[data-time]');

    var pressureValue=root.querySelector('[data-pressure-value]');
    var barrierValue=root.querySelector('[data-barrier-value]');
    var reabsorptionValue=root.querySelector('[data-reabsorption-value]');
    var secretionValue=root.querySelector('[data-secretion-value]');
    var waterValue=root.querySelector('[data-water-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var filtrationText=root.querySelector('[data-filtration-index]');
    var reabsorptionText=root.querySelector('[data-reabsorption-index]');
    var urineVolumeText=root.querySelector('[data-urine-volume]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');
    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='filtration';
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
        timeInput.value=String(next>100?0:next);
        update();
        schedule();
      },800);
    }

    function particle(kind,x,y,scale,opacity){
      if(kind==='rbc'){
        return '<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
          +'<ellipse rx="13" ry="8" fill="#FCA5A5" stroke="#B91C1C" stroke-width="3"/>'
          +'<ellipse rx="5" ry="3" fill="#FEE2E2"/>'
          +'</g>';
      }

      if(kind==='protein'){
        return '<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
          +'<circle r="11" fill="#C4B5FD" stroke="#6D28D9" stroke-width="3"/>'
          +'<path d="M-6 1 Q0 -8 6 1 Q0 8 -6 1" fill="none" stroke="#5B21B6" stroke-width="2"/>'
          +'</g>';
      }

      var color=kind==='glucose'
        ?'#F59E0B'
        :kind==='amino'
          ?'#EC4899'
          :kind==='urea'
            ?'#64748B'
            :kind==='salt'
              ?'#8B5CF6'
              :'#38BDF8';

      var label=kind==='glucose'
        ?'G'
        :kind==='amino'
          ?'A'
          :kind==='urea'
            ?'U'
            :kind==='salt'
              ?'S'
              :'H₂O';

      return '<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle r="8" fill="'+color+'" stroke="#334155" stroke-width="1.5"/>'
        +'<text x="0" y="3" text-anchor="middle" font-size="6.5" font-weight="900" fill="#FFFFFF">'+label+'</text>'
        +'</g>';
    }

    function nephronOutline(){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<circle cx="171" cy="188" r="78" fill="#FFF7ED" stroke="#F59E0B" stroke-width="7"/>'
        +'<circle cx="171" cy="188" r="56" fill="#FEE2E2" stroke="#DC2626" stroke-width="5"/>'
        +'<path d="M139 169 C121 128 171 116 191 145 C220 118 237 166 211 185 C238 207 213 249 185 224 C159 253 119 224 139 194 C112 187 115 159 139 169Z" fill="none" stroke="#DC2626" stroke-width="8" stroke-linecap="round"/>'
        +'<path d="M90 171 H125" stroke="#DC2626" stroke-width="16" stroke-linecap="round"/>'
        +'<path d="M218 150 H274" stroke="#B91C1C" stroke-width="12" stroke-linecap="round"/>'
        +'<path d="M225 213 C271 225 278 265 249 291 C213 324 232 353 296 348 C352 343 347 286 311 267 C278 249 293 219 335 221 C378 222 385 270 362 297 C337 327 358 360 423 350" fill="none" stroke="#60A5FA" stroke-width="22" stroke-linecap="round" stroke-linejoin="round"/>'
        +'<path d="M423 350 C473 337 477 274 442 247 C408 221 421 174 471 167 C520 160 537 201 509 231 C486 255 493 310 548 321" fill="none" stroke="#38BDF8" stroke-width="22" stroke-linecap="round" stroke-linejoin="round"/>'
        +'<path d="M548 321 V376" stroke="#0EA5E9" stroke-width="25" stroke-linecap="round"/>'
        +'<path d="M274 150 C332 137 360 123 406 135 C466 151 521 133 570 109" fill="none" stroke="#FCA5A5" stroke-width="18" stroke-linecap="round"/>'
        +'<path d="M279 160 C337 151 366 148 411 159 C460 172 510 157 556 132" fill="none" stroke="#93C5FD" stroke-width="10" stroke-linecap="round"/>'
        +'</g>';
    }

    function renderFiltration(pressure,barrier,progress){
      var filtration=clamp(
        pressure/100*(.22+.78*progress),
        0,
        1
      );

      var leak=clamp(
        (1-barrier/100)
        *pressure/100
        *(.20+.80*progress),
        0,
        1
      );

      var bloodParticles='';
      var filtrateParticles='';
      var small=['water','glucose','amino','urea','salt'];

      for(var i=0;i<14;i++){
        var angle=Math.PI*2*i/14;
        var x=171+Math.cos(angle)*36;
        var y=188+Math.sin(angle)*30;
        var kind=i%4===0?'rbc':i%4===1?'protein':small[i%small.length];

        bloodParticles+=particle(
          kind,
          x.toFixed(1),
          y.toFixed(1),
          kind==='rbc'?.72:.65,
          .92
        );
      }

      var filteredCount=Math.floor(2+filtration*13);

      for(var j=0;j<filteredCount;j++){
        var fx=228+(j%5)*25;
        var fy=207+Math.floor(j/5)*24;
        var filteredKind=small[j%small.length];

        filtrateParticles+=particle(
          filteredKind,
          fx,
          fy,
          .55,
          .55+.40*filtration
        );
      }

      var leaked='';
      var leakedCount=Math.floor(leak*5);

      for(var k=0;k<leakedCount;k++){
        leaked+=particle(
          k%2===0?'protein':'rbc',
          250+k*22,
          262+k%2*15,
          .48,
          .45+.45*leak
        );
      }

      return ''
        +'<rect x="26" y="86" width="708" height="284" rx="25" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +nephronOutline()
        +bloodParticles
        +filtrateParticles
        +leaked
        +'<path class="nu-flow" d="M197 185 C229 190 245 204 264 221" fill="none" stroke="#2563EB" stroke-width="'+(3+filtration*6)+'" marker-end="url(#${rootId}-arrow-blue)" opacity="'+(.30+.65*filtration)+'"/>'
        +'<text x="69" y="118" font-size="13" font-weight="900" fill="#991B1B">入球小动脉</text>'
        +'<text x="222" y="122" font-size="13" font-weight="900" fill="#991B1B">出球小动脉</text>'
        +'<text x="132" y="287" font-size="13" font-weight="900" fill="#92400E">肾小球与肾小囊</text>'
        +'<g transform="translate(510 92)">'
        +'<rect width="194" height="178" rx="19" fill="#FFFFFF" stroke="#BFDBFE" stroke-width="3"/>'
        +'<text x="97" y="26" text-anchor="middle" font-size="14" font-weight="900" fill="#1D4ED8">滤过屏障选择性</text>'
        +'<text x="17" y="59" font-size="11.5" font-weight="900" fill="#0369A1">可进入原尿</text>'
        +'<text x="17" y="82" font-size="10.5" font-weight="800" fill="#475569">水、葡萄糖、氨基酸</text>'
        +'<text x="17" y="101" font-size="10.5" font-weight="800" fill="#475569">尿素和部分无机盐</text>'
        +'<text x="17" y="130" font-size="11.5" font-weight="900" fill="#991B1B">通常被保留在血液</text>'
        +'<text x="17" y="153" font-size="10.5" font-weight="800" fill="#475569">血细胞和大分子蛋白质</text>'
        +'</g>'
        +'<g transform="translate(486 290)">'
        +'<rect width="223" height="57" rx="15" fill="'+(leak>.15?'#FEE2E2':'#EFF6FF')+'" stroke="'+(leak>.15?'#FCA5A5':'#BFDBFE')+'" stroke-width="2"/>'
        +'<text x="111" y="23" text-anchor="middle" font-size="12" font-weight="900" fill="'+(leak>.15?'#991B1B':'#1E40AF')+'">相对滤过指数 '+(filtration*100).toFixed(0)+'</text>'
        +'<text x="111" y="43" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">屏障异常泄漏指数 '+(leak*100).toFixed(0)+'</text>'
        +'</g>';
    }

    function renderReabsorption(reabsorption,water,progress){
      var selective=clamp(
        reabsorption/100*(.18+.82*progress),
        0,
        1
      );
      var waterFactor=clamp(
        water/100*(.18+.82*progress),
        0,
        1
      );

      var tubuleParticles='';
      var capillaryParticles='';
      var kinds=['glucose','amino','water','salt','urea'];

      for(var i=0;i<18;i++){
        var x=229+(i%9)*37;
        var y=225+Math.sin(i*.88)*63;
        var kind=kinds[i%kinds.length];

        tubuleParticles+=particle(
          kind,
          x.toFixed(1),
          y.toFixed(1),
          .56,
          .78
        );
      }

      var returned=Math.floor(3+selective*11+waterFactor*5);

      for(var j=0;j<returned;j++){
        var rx=270+(j%8)*38;
        var ry=138+Math.floor(j/8)*27;
        var rKind=j%4===0?'water':j%4===1?'glucose':j%4===2?'amino':'salt';

        capillaryParticles+=particle(
          rKind,
          rx,
          ry,
          .48,
          .55+.40*Math.max(selective,waterFactor)
        );
      }

      var arrowWidth=3+selective*5;

      return ''
        +'<rect x="26" y="86" width="708" height="284" rx="25" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +nephronOutline()
        +tubuleParticles
        +capillaryParticles
        +'<path class="nu-flow" d="M284 263 C300 216 311 191 335 160" fill="none" stroke="#16A34A" stroke-width="'+arrowWidth+'" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<path class="nu-flow" d="M393 298 C410 248 422 213 446 171" fill="none" stroke="#16A34A" stroke-width="'+arrowWidth+'" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<path class="nu-flow" d="M511 286 C521 235 529 201 548 158" fill="none" stroke="#38BDF8" stroke-width="'+(3+waterFactor*6)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<g transform="translate(503 91)">'
        +'<rect width="208" height="190" rx="19" fill="#FFFFFF" stroke="#A7F3D0" stroke-width="3"/>'
        +'<text x="104" y="27" text-anchor="middle" font-size="14" font-weight="900" fill="#047857">选择性重吸收</text>'
        +'<text x="16" y="58" font-size="11.5" font-weight="900" fill="#B45309">葡萄糖、氨基酸</text>'
        +'<text x="16" y="79" font-size="10.5" font-weight="800" fill="#475569">正常情况下通常几乎全部重吸收</text>'
        +'<text x="16" y="110" font-size="11.5" font-weight="900" fill="#0369A1">水和无机盐</text>'
        +'<text x="16" y="131" font-size="10.5" font-weight="800" fill="#475569">根据机体需要进行不同比例重吸收</text>'
        +'<text x="16" y="162" font-size="11.5" font-weight="900" fill="#475569">尿素</text>'
        +'<text x="16" y="181" font-size="10.5" font-weight="800" fill="#475569">部分可被重吸收，部分保留在尿液中</text>'
        +'</g>'
        +'<g transform="translate(67 330)">'
        +'<rect width="393" height="34" rx="12" fill="#ECFDF5" stroke="#A7F3D0" stroke-width="2"/>'
        +'<text x="196" y="22" text-anchor="middle" font-size="11.5" font-weight="900" fill="#166534">肾小管上皮细胞将有用物质从管腔侧运回血液</text>'
        +'</g>';
    }

    function renderSecretion(secretion,progress){
      var secretionFactor=clamp(
        secretion/100*(.18+.82*progress),
        0,
        1
      );

      var secretedCount=Math.floor(2+secretionFactor*12);
      var particles='';

      for(var i=0;i<secretedCount;i++){
        var x=318+(i%6)*43;
        var y=150+Math.floor(i/6)*31;
        var kind=i%3===0?'salt':i%3===1?'urea':'protein';

        particles+=particle(
          kind,
          x,
          y,
          .48,
          .62+.35*secretionFactor
        );
      }

      return ''
        +'<rect x="26" y="86" width="708" height="284" rx="25" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +nephronOutline()
        +particles
        +'<path class="nu-flow" d="M349 156 C340 194 328 224 314 257" fill="none" stroke="#F59E0B" stroke-width="'+(3+secretionFactor*6)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path class="nu-flow" d="M449 166 C442 207 431 243 417 281" fill="none" stroke="#F59E0B" stroke-width="'+(3+secretionFactor*6)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path class="nu-flow" d="M541 152 C536 201 536 253 546 318" fill="none" stroke="#F59E0B" stroke-width="'+(3+secretionFactor*6)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<g transform="translate(493 94)">'
        +'<rect width="218" height="190" rx="19" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="109" y="28" text-anchor="middle" font-size="14" font-weight="900" fill="#9A3412">肾小管分泌</text>'
        +'<text x="16" y="60" font-size="11.5" font-weight="900" fill="#92400E">运输方向</text>'
        +'<text x="16" y="82" font-size="10.5" font-weight="800" fill="#475569">管周毛细血管 → 肾小管管腔</text>'
        +'<text x="16" y="113" font-size="11.5" font-weight="900" fill="#92400E">常见示意物质</text>'
        +'<text x="16" y="135" font-size="10.5" font-weight="800" fill="#475569">氢离子、钾离子及部分药物或代谢物</text>'
        +'<text x="16" y="166" font-size="11.5" font-weight="900" fill="#92400E">功能意义</text>'
        +'<text x="16" y="186" font-size="10.5" font-weight="800" fill="#475569">参与酸碱、电解质和代谢废物调节</text>'
        +'</g>'
        +'<g transform="translate(69 331)">'
        +'<rect width="386" height="34" rx="12" fill="#FFFBEB" stroke="#FDE68A" stroke-width="2"/>'
        +'<text x="193" y="22" text-anchor="middle" font-size="11.5" font-weight="900" fill="#92400E">分泌不是滤过，也不是把所有物质都排入尿液</text>'
        +'</g>';
    }

    function compositionBar(x,y,width,value,color,label){
      return ''
        +'<text x="'+x+'" y="'+(y-7)+'" font-size="10.5" font-weight="900" fill="#475569">'+label+'</text>'
        +'<rect x="'+x+'" y="'+y+'" width="'+width+'" height="13" rx="6.5" fill="#E2E8F0"/>'
        +'<rect x="'+x+'" y="'+y+'" width="'+(width*clamp(value,0,100)/100)+'" height="13" rx="6.5" fill="'+color+'"/>';
    }

    function renderCompare(barrier,reabsorption,water,secretion){
      var leak=(100-barrier)*.45;
      var glucoseFinal=clamp(100-reabsorption*1.15,0,100);
      var aminoFinal=clamp(100-reabsorption*1.20,0,100);
      var waterFinal=clamp(100-water*.82,8,100);
      var saltFinal=clamp(55-reabsorption*.42+secretion*.16,8,100);
      var ureaFinal=clamp(42+water*.48+secretion*.18,30,100);

      var columns=[
        {x:42,title:'血浆',fill:'#FFF1F2',stroke:'#FCA5A5'},
        {x:281,title:'原尿',fill:'#EFF6FF',stroke:'#93C5FD'},
        {x:520,title:'终尿',fill:'#ECFDF5',stroke:'#86EFAC'}
      ];

      var html='';

      for(var i=0;i<columns.length;i++){
        var c=columns[i];
        html+='<g transform="translate('+c.x+' 92)">'
          +'<rect width="200" height="262" rx="20" fill="'+c.fill+'" stroke="'+c.stroke+'" stroke-width="3"/>'
          +'<text x="100" y="29" text-anchor="middle" font-size="16" font-weight="900" fill="#334155">'+c.title+'</text>'
          +'</g>';
      }

      html+=compositionBar(59,145,166,100,'#EF4444','血细胞');
      html+=compositionBar(59,187,166,92,'#8B5CF6','大分子蛋白');
      html+=compositionBar(59,229,166,68,'#F59E0B','葡萄糖/氨基酸');
      html+=compositionBar(59,271,166,90,'#38BDF8','水');
      html+=compositionBar(59,313,166,30,'#64748B','尿素');

      html+=compositionBar(298,145,166,leak*.15,'#EF4444','血细胞');
      html+=compositionBar(298,187,166,leak,'#8B5CF6','大分子蛋白');
      html+=compositionBar(298,229,166,65,'#F59E0B','葡萄糖/氨基酸');
      html+=compositionBar(298,271,166,88,'#38BDF8','水');
      html+=compositionBar(298,313,166,35,'#64748B','尿素');

      html+=compositionBar(537,145,166,leak*.22,'#EF4444','血细胞');
      html+=compositionBar(537,187,166,leak*.65,'#8B5CF6','大分子蛋白');
      html+=compositionBar(537,229,166,(glucoseFinal+aminoFinal)/2,'#F59E0B','葡萄糖/氨基酸');
      html+=compositionBar(537,271,166,waterFinal,'#38BDF8','水');
      html+=compositionBar(537,313,166,ureaFinal,'#64748B','尿素');

      return html
        +'<path class="nu-flow" d="M245 224 H272" fill="none" stroke="#2563EB" stroke-width="5" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="nu-flow" d="M484 224 H511" fill="none" stroke="#16A34A" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<text x="259" y="202" text-anchor="middle" font-size="11" font-weight="900" fill="#1D4ED8">滤过</text>'
        +'<text x="498" y="202" text-anchor="middle" font-size="11" font-weight="900" fill="#047857">重吸收与分泌</text>'
        +'<text x="552" y="377" font-size="11" font-weight="900" fill="#475569">无机盐相对指数 '+saltFinal.toFixed(0)+'</text>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='filtration'){
        labels.innerHTML=''
          +'<path d="M171 110 L171 78" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="119" y="72" font-size="13" font-weight="900" fill="#991B1B">肾小球毛细血管</text>'
          +'<path d="M104 229 L48 258" stroke="#F59E0B" stroke-width="2.5"/>'
          +'<text x="24" y="276" font-size="13" font-weight="900" fill="#92400E">肾小囊</text>'
          +'<path d="M258 238 L338 197" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="345" y="195" font-size="13" font-weight="900" fill="#1D4ED8">原尿流入肾小管</text>';
        return;
      }

      if(modeName==='reabsorption'){
        labels.innerHTML=''
          +'<path d="M309 277 L235 319" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="105" y="326" font-size="13" font-weight="900" fill="#166534">近端小管重吸收较活跃</text>'
          +'<path d="M409 153 L468 103" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="476" y="101" font-size="13" font-weight="900" fill="#991B1B">管周毛细血管</text>'
          +'<path d="M547 322 L625 298" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="633" y="301" font-size="13" font-weight="900" fill="#0369A1">集合管</text>';
        return;
      }

      if(modeName==='secretion'){
        labels.innerHTML=''
          +'<path d="M359 161 L437 103" stroke="#F59E0B" stroke-width="2.5"/>'
          +'<text x="445" y="101" font-size="13" font-weight="900" fill="#92400E">由血液进入管腔</text>'
          +'<path d="M410 278 L471 314" stroke="#0EA5E9" stroke-width="2.5"/>'
          +'<text x="479" y="319" font-size="13" font-weight="900" fill="#0369A1">远端小管</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M136 92 L136 73" stroke="#DC2626" stroke-width="2.5"/>'
        +'<text x="93" y="68" font-size="13" font-weight="900" fill="#991B1B">血液成分</text>'
        +'<path d="M381 92 L381 73" stroke="#2563EB" stroke-width="2.5"/>'
        +'<text x="333" y="68" font-size="13" font-weight="900" fill="#1D4ED8">滤过形成原尿</text>'
        +'<path d="M619 92 L619 73" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="567" y="68" font-size="13" font-weight="900" fill="#166534">调节后形成终尿</text>';
    }

    function update(){
      var pressure=Number(pressureInput.value);
      var barrier=Number(barrierInput.value);
      var reabsorption=Number(reabsorptionInput.value);
      var secretion=Number(secretionInput.value);
      var water=Number(waterInput.value);
      var processTime=Number(timeInput.value);

      pressureValue.textContent=pressure.toFixed(0)+'%';
      barrierValue.textContent=barrier.toFixed(0)+'%';
      reabsorptionValue.textContent=reabsorption.toFixed(0)+'%';
      secretionValue.textContent=secretion.toFixed(0)+'%';
      waterValue.textContent=water.toFixed(0)+'%';
      timeValue.textContent=processTime.toFixed(0)+'%';

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

      autoButton.textContent=automatic
        ?'过程推进：运行中'
        :'过程推进：已暂停';

      autoButton.classList.toggle('off',!automatic);

      var progress=processTime/100;

      var filtrationIndex=100*clamp(
        pressure/100*(.25+.75*progress),
        0,
        1
      );

      var reabsorptionIndex=100*Math.sqrt(
        reabsorption/100
        *(.25+.75*water/100)
        *(.20+.80*progress)
      );

      reabsorptionIndex=clamp(reabsorptionIndex,0,100);

      var primaryUrine=filtrationIndex;
      var returned=primaryUrine
        *reabsorptionIndex/100
        *.78;
      var addedBySecretion=primaryUrine
        *secretion/100
        *progress
        *.16;
      var waterSaved=primaryUrine
        *water/100
        *progress
        *.52;

      var finalUrine=clamp(
        primaryUrine-returned-waterSaved+addedBySecretion,
        2,
        100
      );

      var concentrationIndex=clamp(
        25+water*.62+secretion*.13,
        20,
        100
      );

      filtrationText.textContent=filtrationIndex.toFixed(0);
      reabsorptionText.textContent=reabsorptionIndex.toFixed(0);
      urineVolumeText.textContent=finalUrine.toFixed(0);

      root.style.setProperty(
        '--nu-speed',
        clamp(2.45-Math.max(filtrationIndex,reabsorptionIndex)/75,.65,2.35).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='filtration'){
        title.textContent='肾小球滤过与原尿形成';
        summary.textContent='观察血液流经肾小球时，小分子物质进入肾小囊形成原尿。';
        dynamic.innerHTML=renderFiltration(pressure,barrier,progress);
        stageNote.textContent='完整滤过屏障通常保留血细胞和大分子蛋白质。';
        renderLabels(mode);
      }else if(mode==='reabsorption'){
        title.textContent='肾小管的选择性重吸收';
        summary.textContent='观察葡萄糖、氨基酸、水和无机盐从管腔返回血液。';
        dynamic.innerHTML=renderReabsorption(reabsorption,water,progress);
        stageNote.textContent='重吸收把有用物质和大部分水送回血液。';
        renderLabels(mode);
      }else if(mode==='secretion'){
        title.textContent='肾小管分泌与终尿成分调节';
        summary.textContent='观察部分离子、药物和代谢物由血液进入肾小管管腔。';
        dynamic.innerHTML=renderSecretion(secretion,progress);
        stageNote.textContent='肾小管分泌与滤过、重吸收共同决定终尿成分。';
        renderLabels(mode);
      }else{
        title.textContent='血浆、原尿与终尿成分比较';
        summary.textContent='比较滤过、重吸收和分泌前后主要成分的相对变化。';
        dynamic.innerHTML=renderCompare(barrier,reabsorption,water,secretion);
        stageNote.textContent='原尿不是终尿，二者在肾小管中经历大量选择性调节。';
        renderLabels(mode);
      }

      var leakIndex=(100-barrier)*pressure/100;
      var condition='当前滤过、重吸收、分泌和水重吸收处于相对协调状态。';

      if(pressure<30){
        condition='肾小球相对压力较低，原尿形成的相对速度下降。';
      }else if(barrier<45){
        condition='滤过屏障完整度较低，血细胞或大分子蛋白质异常进入滤液的教学风险指数升高。';
      }else if(reabsorption<25&&filtrationIndex>30){
        condition='选择性重吸收能力较低，葡萄糖、氨基酸、水和无机盐返回血液的比例下降。';
      }else if(water<20&&filtrationIndex>30){
        condition='水重吸收水平较低，终尿相对量增大且浓缩程度下降。';
      }else if(secretion<15){
        condition='肾小管分泌活性较低，部分离子和代谢物进入管腔的相对过程减弱。';
      }else if(processTime<15){
        condition='过程时间较短，当前主要显示早期滤过，后续重吸收和分泌尚未充分进行。';
      }

      var principle=mode==='filtration'
        ?'肾小球滤过屏障允许水和多种小分子进入肾小囊，而血细胞和大分子蛋白质通常被保留在血液中。'
        :mode==='reabsorption'
          ?'在正常生理情况下，滤出的葡萄糖和氨基酸通常几乎全部被重吸收，水和无机盐按机体需要进行不同比例重吸收。'
          :mode==='secretion'
            ?'肾小管可将氢离子、钾离子以及部分药物或代谢物由血液分泌到管腔中。'
            :'原尿可近似理解为不含血细胞且大分子蛋白质很少的血浆滤液，经过重吸收和分泌后形成终尿。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前屏障泄漏指数 '+leakIndex.toFixed(0)
        +'，终尿浓缩指数 '+concentrationIndex.toFixed(0)
        +'；所有数值仅用于教学比较，不用于肾功能判断。';
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

    pressureInput.oninput=update;
    barrierInput.oninput=update;
    reabsorptionInput.oninput=update;
    secretionInput.oninput=update;
    waterInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
