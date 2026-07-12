/**
 * lifeScienceLabTemplatesImmunityInnateInflammation.ts
 *
 * 平面生命科学实验室：先天免疫与炎症反应。
 *
 * 教学目标：
 * 1. 认识皮肤和黏膜屏障在阻止病原体进入中的作用；
 * 2. 观察病原体突破屏障后，受损细胞和免疫细胞释放炎症介质；
 * 3. 理解毛细血管扩张和通透性增加与局部红、肿、热、痛的关系；
 * 4. 观察中性粒细胞、巨噬细胞趋化、迁出血管和吞噬病原体；
 * 5. 区分局部炎症反应与全身发热等全身反应；
 * 6. 理解炎症既有防御作用，也可能在过强或持续时造成组织损伤。
 *
 * 科学边界：
 * 1. 皮肤和黏膜屏障属于机体重要的第一道防线；
 * 2. 屏障被破坏后，病原体及组织损伤信号可启动先天免疫反应；
 * 3. 组织细胞、肥大细胞和免疫细胞可释放多种炎症介质；
 * 4. 毛细血管扩张可增加局部血流，通透性增加有利于液体和免疫细胞进入组织；
 * 5. 中性粒细胞通常较早到达炎症部位，巨噬细胞可吞噬病原体和细胞碎片；
 * 6. 先天免疫反应启动较快，但缺乏针对特定抗原的高度特异性；
 * 7. 局部红、肿、热、痛是多种血管、神经和组织变化共同作用的结果；
 * 8. 局部炎症不等同于全身发热，发热涉及全身性信号和体温调节中枢改变；
 * 9. 炎症具有清除病原体、限制感染和促进修复等作用；
 * 10. 过强或持续的炎症反应也可能造成组织损伤；
 * 11. 图中的病原体数量、介质水平和炎症指数均为相对教学指标；
 * 12. 本模板只用于生物学教学，不用于疾病诊断、用药建议或个体健康评价。
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

function innateInflammationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FED7AA;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FFEDD5,#FEE2E2);border-bottom:1px solid #FED7AA}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9A3412}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFFBF7;border-right:1px solid #FED7AA}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#EA580C;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#F97316}'
    + '#' + rootId + ' .ii-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#9A3412}'
    + '#' + rootId + ' .ii-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ii-button{min-height:30px;padding:3px;border:1px solid #FDBA74;border-radius:8px;background:#fff;color:#9A3412;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .ii-button.active{border-color:#EA580C;background:#FFEDD5;box-shadow:0 3px 9px rgba(234,88,12,.13)}'
    + '#' + rootId + ' .ii-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#FB923C,#EA580C);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ii-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ii-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .ii-card{padding:6px 3px;border:1px solid #FED7AA;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ii-card b{display:block;font-size:13px;color:#C2410C}'
    + '#' + rootId + ' .ii-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FFEDD5;color:#7C2D12;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ii-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--ii-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ii-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_IMMUNITY_INNATE_INFLAMMATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-innate-immunity-inflammation',
    group: '🛡️ 免疫与疾病防御',
    name: '先天免疫与炎症反应',
    emoji: '🔥',
    desc: '调节病原体、屏障完整度、炎症介质、吞噬细胞活性和反应时间，观察局部炎症与先天免疫防御',
    params: [
      {
        key: 'pathogenLoad',
        label: '病原体相对数量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 64,
      },
      {
        key: 'barrierIntegrity',
        label: '屏障完整度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'mediatorRelease',
        label: '炎症介质释放',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'phagocyteActivity',
        label: '吞噬细胞活跃度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'responseTime',
        label: '反应过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 44,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const pathogenLoad = num(params, 'pathogenLoad', 64)
      const barrierIntegrity = num(params, 'barrierIntegrity', 72)
      const mediatorRelease = num(params, 'mediatorRelease', 68)
      const phagocyteActivity = num(params, 'phagocyteActivity', 76)
      const responseTime = num(params, 'responseTime', 44)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${innateInflammationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🔥 先天免疫与炎症反应</div>
    <div class="bl-note">炎症指标均为相对教学示意</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>病原体相对数量</span>
          <span class="bl-value" data-pathogen-value></span>
        </div>
        <input
          data-pathogen
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(pathogenLoad)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>屏障完整度</span>
          <span class="bl-value" data-barrier-value></span>
        </div>
        <input
          data-barrier
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(barrierIntegrity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>炎症介质释放</span>
          <span class="bl-value" data-mediator-value></span>
        </div>
        <input
          data-mediator
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(mediatorRelease)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>吞噬细胞活跃度</span>
          <span class="bl-value" data-phagocyte-value></span>
        </div>
        <input
          data-phagocyte
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(phagocyteActivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>反应过程时间</span>
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

      <div class="ii-subtitle">观察方式</div>

      <div class="ii-buttons">
        <button
          type="button"
          class="ii-button active"
          data-mode="barrier"
        >屏障突破</button>

        <button
          type="button"
          class="ii-button"
          data-mode="vascular"
        >血管反应</button>

        <button
          type="button"
          class="ii-button"
          data-mode="phagocytosis"
        >趋化与吞噬</button>

        <button
          type="button"
          class="ii-button"
          data-mode="effects"
        >红肿热痛</button>
      </div>

      <button
        type="button"
        class="ii-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="ii-toggle"
        data-auto
      >反应推进：运行中</button>

      <div class="ii-status">
        <div class="ii-card">
          <b data-entered></b>
          <span>突破屏障</span>
        </div>

        <div class="ii-card">
          <b data-inflammation></b>
          <span>炎症指数</span>
        </div>

        <div class="ii-card">
          <b data-remaining></b>
          <span>剩余病原体</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="先天免疫与炎症反应互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-orange"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#EA580C"/>
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
              flood-color="#7C2D12"
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
          fill="#9A3412"
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
            fill="#FFF7ED"
            stroke="#FED7AA"
            stroke-width="2"
          />

          <text
            x="108"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#9A3412"
          >关键边界</text>

          <text
            x="108"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#7C2D12"
          >先天免疫快速但特异性较弱</text>

          <text
            x="108"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#7C2D12"
          >过强或持续炎症可损伤组织</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#9A3412"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var pathogenInput=root.querySelector('[data-pathogen]');
    var barrierInput=root.querySelector('[data-barrier]');
    var mediatorInput=root.querySelector('[data-mediator]');
    var phagocyteInput=root.querySelector('[data-phagocyte]');
    var timeInput=root.querySelector('[data-time]');

    var pathogenValue=root.querySelector('[data-pathogen-value]');
    var barrierValue=root.querySelector('[data-barrier-value]');
    var mediatorValue=root.querySelector('[data-mediator-value]');
    var phagocyteValue=root.querySelector('[data-phagocyte-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var enteredText=root.querySelector('[data-entered]');
    var inflammationText=root.querySelector('[data-inflammation]');
    var remainingText=root.querySelector('[data-remaining]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');
    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='barrier';
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

    function pathogenShape(
      x,
      y,
      size,
      opacity
    ){
      var html='<g transform="translate('+x+' '+y+')" opacity="'+opacity+'">';

      for(var i=0;i<8;i++){
        var angle=Math.PI*2*i/8;
        var x1=Math.cos(angle)*(size+2);
        var y1=Math.sin(angle)*(size+2);
        var x2=Math.cos(angle)*(size+9);
        var y2=Math.sin(angle)*(size+9);

        html+='<line x1="'+x1+'" y1="'+y1
          +'" x2="'+x2+'" y2="'+y2
          +'" stroke="#DC2626" stroke-width="2"/>';
      }

      html+='<circle cx="0" cy="0" r="'+size
        +'" fill="#FECACA" stroke="#B91C1C" stroke-width="3"/>'
        +'</g>';

      return html;
    }

    function neutrophilShape(
      x,
      y,
      scale,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle r="34" fill="#FEF3C7" stroke="#D97706" stroke-width="5"/>'
        +'<circle cx="-11" cy="-2" r="10" fill="#F59E0B"/>'
        +'<circle cx="6" cy="-9" r="10" fill="#F59E0B"/>'
        +'<circle cx="13" cy="8" r="10" fill="#F59E0B"/>'
        +'<text x="0" y="53" text-anchor="middle" font-size="11" font-weight="900" fill="#92400E">中性粒细胞</text>'
        +'</g>';
    }

    function macrophageShape(
      x,
      y,
      scale,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<path d="M-36 -6 C-34 -32 -4 -43 16 -31 C39 -17 42 12 25 31 C6 51 -28 40 -39 18 C-44 9 -43 0 -36 -6Z" fill="#DCFCE7" stroke="#16A34A" stroke-width="5"/>'
        +'<ellipse cx="-4" cy="4" rx="16" ry="12" fill="#86EFAC" stroke="#15803D" stroke-width="3"/>'
        +'<text x="0" y="57" text-anchor="middle" font-size="11" font-weight="900" fill="#166534">巨噬细胞</text>'
        +'</g>';
    }

    function renderBarrier(
      pathogen,
      barrier,
      progress
    ){
      var incomingCount=Math.floor(
        5+pathogen/9
      );

      var enteredFraction=clamp(
        (1-barrier/100)
        *(.22+.78*progress),
        0,
        1
      );

      var passedCount=Math.floor(
        incomingCount*enteredFraction
      );

      var incoming='';
      var passed='';

      for(var i=0;i<incomingCount;i++){
        var ix=58+(i%6)*35;
        var iy=119+Math.floor(i/6)*44;

        incoming+=pathogenShape(
          ix,
          iy,
          8+i%3,
          .76
        );
      }

      for(var j=0;j<passedCount;j++){
        var px=445+(j%5)*42;
        var py=178+Math.floor(j/5)*48;

        passed+=pathogenShape(
          px,
          py,
          8+j%3,
          .42+.50*enteredFraction
        );
      }

      var gapSize=4+(100-barrier)*.30;

      var skinCells='';

      for(var c=0;c<7;c++){
        var y=105+c*38;
        var broken=barrier<58 && c===3;
        var leftWidth=broken
          ?Math.max(5,42-gapSize)
          :42;

        skinCells+='<rect x="306" y="'+y
          +'" width="'+leftWidth
          +'" height="28" rx="9" fill="#FDBA74" stroke="#C2410C" stroke-width="3"/>';

        if(broken){
          skinCells+='<rect x="'+(306+42+gapSize)
            +'" y="'+y
            +'" width="'+Math.max(5,42-gapSize)
            +'" height="28" rx="9" fill="#FDBA74" stroke="#C2410C" stroke-width="3"/>';
        }else{
          skinCells+='<rect x="352" y="'+y
            +'" width="42" height="28" rx="9" fill="#FDBA74" stroke="#C2410C" stroke-width="3"/>';
        }
      }

      var wound=barrier<65
        ?'<path d="M332 210 L350 230 L368 204" fill="none" stroke="#B91C1C" stroke-width="7" stroke-linecap="round"/>'
        :'';

      return ''
        +'<rect x="25" y="89" width="244" height="262" rx="24" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="147" y="118" text-anchor="middle" font-size="15" font-weight="900" fill="#9A3412">外界病原体</text>'
        +incoming
        +'<g filter="url(#${rootId}-shadow)">'
        +skinCells
        +wound
        +'</g>'
        +'<text x="350" y="355" text-anchor="middle" font-size="15" font-weight="900" fill="#9A3412">皮肤和黏膜屏障</text>'
        +'<rect x="426" y="89" width="306" height="262" rx="24" fill="#FFF1F2" stroke="#FCA5A5" stroke-width="3"/>'
        +'<text x="579" y="118" text-anchor="middle" font-size="15" font-weight="900" fill="#9F1239">组织内部</text>'
        +passed
        +'<path class="ii-flow" d="M248 220 H292" fill="none" stroke="#EA580C" stroke-width="5" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path class="ii-flow" d="M402 220 H438" fill="none" stroke="#DC2626" stroke-width="5" marker-end="url(#${rootId}-arrow-orange)" opacity="'+enteredFraction.toFixed(2)+'"/>'
        +'<g transform="translate(482 291)">'
        +'<rect width="208" height="39" rx="12" fill="#FFFFFF" stroke="#FCA5A5" stroke-width="2"/>'
        +'<text x="104" y="25" text-anchor="middle" font-size="12" font-weight="900" fill="#991B1B">突破比例 '+(enteredFraction*100).toFixed(0)+'%</text>'
        +'</g>';
    }

    function renderVascular(
      mediator,
      inflammation,
      progress
    ){
      var dilation=28+mediator*.38;
      var permeability=clamp(
        mediator/100*(.25+.75*progress),
        0,
        1
      );

      var vesselTop=178-dilation*.35;
      var vesselBottom=178+dilation*.35;

      var mediatorParticles='';

      var mediatorCount=Math.floor(
        4+mediator/9
      );

      for(var i=0;i<mediatorCount;i++){
        var mx=118+(i%7)*43;
        var my=103+Math.floor(i/7)*31;

        mediatorParticles+='<circle cx="'+mx+'" cy="'+my
          +'" r="'+(4+i%3)
          +'" fill="#F97316" opacity=".75"/>'
          +'<text x="'+(mx+7)+'" y="'+(my+4)
          +'" font-size="8" font-weight="900" fill="#9A3412">M</text>';
      }

      var plasma='';

      var plasmaCount=Math.floor(
        2+permeability*10
      );

      for(var j=0;j<plasmaCount;j++){
        var px=284+(j%6)*42;
        var py=246+Math.floor(j/6)*27;

        plasma+='<path d="M'+px+' '+py
          +' C'+(px-7)+' '+(py+12)+' '+px+' '+(py+21)
          +' '+px+' '+(py+21)
          +' C'+px+' '+(py+21)+' '+(px+7)+' '+(py+12)
          +' '+px+' '+py
          +'Z" fill="#60A5FA" opacity=".64"/>';
      }

      var leukocytes='';

      var leukocyteCount=Math.floor(
        2+inflammation/24
      );

      for(var k=0;k<leukocyteCount;k++){
        var lx=170+k*93;
        var ly=178+(k%2===0?-8:9);

        leukocytes+='<circle cx="'+lx+'" cy="'+ly
          +'" r="17" fill="#FEF3C7" stroke="#D97706" stroke-width="4"/>'
          +'<circle cx="'+(lx-5)+'" cy="'+ly
          +'" r="5" fill="#F59E0B"/>'
          +'<circle cx="'+(lx+5)+'" cy="'+ly
          +'" r="5" fill="#F59E0B"/>';
      }

      return ''
        +'<rect x="41" y="83" width="678" height="278" rx="25" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="71" y="112" font-size="15" font-weight="900" fill="#9A3412">受损组织和免疫细胞释放炎症介质</text>'
        +mediatorParticles
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M78 '+vesselTop.toFixed(1)
        +' C230 '+(vesselTop-9).toFixed(1)
        +' 473 '+(vesselTop+8).toFixed(1)
        +' 684 '+vesselTop.toFixed(1)
        +' L684 '+vesselBottom.toFixed(1)
        +' C473 '+(vesselBottom+9).toFixed(1)
        +' 230 '+(vesselBottom-8).toFixed(1)
        +' 78 '+vesselBottom.toFixed(1)
        +'Z" fill="#FECACA" stroke="#DC2626" stroke-width="5"/>'
        +'<path class="ii-flow" d="M106 178 H653" fill="none" stroke="#EF4444" stroke-width="'+(7+mediator/16)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +leukocytes
        +'</g>'
        +'<g opacity="'+permeability.toFixed(2)+'">'
        +'<path d="M258 '+vesselBottom.toFixed(1)
        +' V240 M350 '+vesselBottom.toFixed(1)
        +' V240 M442 '+vesselBottom.toFixed(1)
        +' V240 M534 '+vesselBottom.toFixed(1)
        +' V240" stroke="#2563EB" stroke-width="4" stroke-dasharray="7 6"/>'
        +plasma
        +'</g>'
        +'<text x="107" y="315" font-size="13" font-weight="900" fill="#B91C1C">血管扩张：局部血流增加</text>'
        +'<text x="407" y="315" font-size="13" font-weight="900" fill="#1D4ED8">通透性增加：液体和细胞进入组织</text>'
        +'<g transform="translate(244 326)">'
        +'<rect width="278" height="29" rx="11" fill="#FFFFFF" stroke="#FED7AA" stroke-width="2"/>'
        +'<text x="139" y="20" text-anchor="middle" font-size="11.5" font-weight="900" fill="#9A3412">相对炎症指数 '+inflammation.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderPhagocytosis(
      entered,
      phagocyte,
      progress
    ){
      var pathogenCount=Math.floor(
        3+entered/7
      );

      var pathogens='';

      for(var i=0;i<pathogenCount;i++){
        var px=500+(i%5)*42;
        var py=134+Math.floor(i/5)*48;

        pathogens+=pathogenShape(
          px,
          py,
          8+i%3,
          .73
        );
      }

      var movement=clamp(
        progress*(.24+.76*phagocyte/100),
        0,
        1
      );

      var neutrophilX=150+movement*275;
      var macrophageX=121+movement*260;

      var phagosome='';

      if(movement>.72){
        phagosome=''
          +'<circle class="ii-pulse" cx="536" cy="210" r="41" fill="none" stroke="#16A34A" stroke-width="5" stroke-dasharray="8 7"/>'
          +'<text x="536" y="269" text-anchor="middle" font-size="12" font-weight="900" fill="#166534">吞噬病原体</text>';
      }

      return ''
        +'<rect x="33" y="84" width="694" height="279" rx="25" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<path d="M47 126 C180 87 307 102 414 142 C511 179 625 172 713 122" fill="none" stroke="#FECACA" stroke-width="58" stroke-linecap="round"/>'
        +'<path class="ii-flow" d="M61 126 C187 92 308 107 410 145 C506 181 612 175 699 128" fill="none" stroke="#DC2626" stroke-width="8" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path d="M226 114 Q235 172 276 198 Q305 217 323 258" fill="none" stroke="#D97706" stroke-width="5" stroke-dasharray="7 6"/>'
        +'<path d="M352 130 Q362 186 397 213 Q426 235 449 268" fill="none" stroke="#16A34A" stroke-width="5" stroke-dasharray="7 6"/>'
        +neutrophilShape(
          neutrophilX.toFixed(1),
          (171+movement*77).toFixed(1),
          .92,
          1
        )
        +macrophageShape(
          macrophageX.toFixed(1),
          (246+movement*48).toFixed(1),
          .88,
          1
        )
        +pathogens
        +phagosome
        +'<path class="ii-flow" d="M278 243 C357 213 413 206 488 190" fill="none" stroke="#D97706" stroke-width="5" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path class="ii-flow" d="M308 309 C377 298 432 271 492 231" fill="none" stroke="#16A34A" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<text x="186" y="345" font-size="13" font-weight="900" fill="#92400E">中性粒细胞通常较早到达</text>'
        +'<text x="452" y="345" font-size="13" font-weight="900" fill="#166534">巨噬细胞吞噬病原体和细胞碎片</text>';
    }

    function renderEffects(
      inflammation,
      mediator,
      responseTime
    ){
      var cards=[
        {
          x:35,
          title:'红',
          color:'#DC2626',
          fill:'#FEE2E2',
          text:'血管扩张、局部血流增加'
        },
        {
          x:211,
          title:'肿',
          color:'#2563EB',
          fill:'#DBEAFE',
          text:'通透性增加、组织液增多'
        },
        {
          x:387,
          title:'热',
          color:'#EA580C',
          fill:'#FFEDD5',
          text:'局部血流和代谢活动增强'
        },
        {
          x:563,
          title:'痛',
          color:'#7C3AED',
          fill:'#EDE9FE',
          text:'炎症介质和组织压力刺激感觉神经'
        }
      ];

      var html='';

      for(var i=0;i<cards.length;i++){
        var card=cards[i];

        html+='<g transform="translate('+card.x+' 101)">'
          +'<rect width="151" height="161" rx="20" fill="'+card.fill
          +'" stroke="'+card.color+'" stroke-width="3"/>'
          +'<circle cx="75" cy="53" r="34" fill="#FFFFFF" stroke="'+card.color+'" stroke-width="4"/>'
          +'<text x="75" y="65" text-anchor="middle" font-size="32" font-weight="900" fill="'+card.color+'">'
          +card.title
          +'</text>'
          +'<text x="75" y="113" text-anchor="middle" font-size="11" font-weight="900" fill="#334155">'
          +card.text.substring(0,10)
          +'</text>'
          +'<text x="75" y="135" text-anchor="middle" font-size="11" font-weight="900" fill="#334155">'
          +card.text.substring(10)
          +'</text>'
          +'</g>';
      }

      var excessive=inflammation>72
        &&(
          mediator>75
          ||responseTime>78
        );

      return html
        +'<g transform="translate(81 291)">'
        +'<rect width="278" height="78" rx="18" fill="#ECFDF5" stroke="#86EFAC" stroke-width="3"/>'
        +'<text x="139" y="25" text-anchor="middle" font-size="14" font-weight="900" fill="#166534">局部炎症的防御意义</text>'
        +'<text x="139" y="48" text-anchor="middle" font-size="11.5" font-weight="800" fill="#166534">限制感染、募集免疫细胞、促进清除与修复</text>'
        +'<text x="139" y="67" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">属于局部组织反应</text>'
        +'</g>'
        +'<g transform="translate(401 291)">'
        +'<rect width="278" height="78" rx="18" fill="'
        +(excessive?'#FEE2E2':'#EFF6FF')
        +'" stroke="'
        +(excessive?'#FCA5A5':'#93C5FD')
        +'" stroke-width="3"/>'
        +'<text x="139" y="25" text-anchor="middle" font-size="14" font-weight="900" fill="'
        +(excessive?'#991B1B':'#1E40AF')
        +'">'
        +(excessive?'炎症反应过强或持续':'局部炎症与全身发热')
        +'</text>'
        +'<text x="139" y="48" text-anchor="middle" font-size="11.5" font-weight="800" fill="'
        +(excessive?'#991B1B':'#1E40AF')
        +'">'
        +(excessive?'可能增加组织损伤风险':'发热涉及全身信号和体温调节中枢')
        +'</text>'
        +'<text x="139" y="67" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'
        +(excessive?'防御与损伤需要保持平衡':'二者不是同一概念')
        +'</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='barrier'){
        labels.innerHTML=''
          +'<path d="M350 128 L443 94" stroke="#C2410C" stroke-width="2.5"/>'
          +'<text x="452" y="97" font-size="13" font-weight="900" fill="#9A3412">上皮屏障</text>'
          +'<path d="M357 220 L448 224" stroke="#B91C1C" stroke-width="2.5"/>'
          +'<text x="456" y="229" font-size="13" font-weight="900" fill="#991B1B">损伤或缺口</text>'
          +'<path d="M518 178 L610 143" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="619" y="143" font-size="13" font-weight="900" fill="#991B1B">进入组织的病原体</text>';
        return;
      }

      if(modeName==='vascular'){
        labels.innerHTML=''
          +'<path d="M373 177 L475 127" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="483" y="126" font-size="13" font-weight="900" fill="#991B1B">扩张的毛细血管</text>'
          +'<path d="M352 241 L444 278" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="452" y="284" font-size="13" font-weight="900" fill="#1D4ED8">血管通透性增加</text>'
          +'<path d="M191 105 L102 86" stroke="#EA580C" stroke-width="2.5"/>'
          +'<text x="27" y="84" font-size="13" font-weight="900" fill="#9A3412">炎症介质</text>';
        return;
      }

      if(modeName==='phagocytosis'){
        labels.innerHTML=''
          +'<path d="M306 205 L214 168" stroke="#D97706" stroke-width="2.5"/>'
          +'<text x="98" y="164" font-size="13" font-weight="900" fill="#92400E">中性粒细胞迁出血管</text>'
          +'<path d="M360 297 L267 335" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="141" y="341" font-size="13" font-weight="900" fill="#166534">巨噬细胞趋化</text>'
          +'<path d="M545 171 L627 121" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="635" y="121" font-size="13" font-weight="900" fill="#991B1B">病原体</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M111 102 L111 75" stroke="#DC2626" stroke-width="2.5"/>'
        +'<text x="68" y="70" font-size="13" font-weight="900" fill="#991B1B">局部表现</text>'
        +'<path d="M522 310 L522 277" stroke="#2563EB" stroke-width="2.5"/>'
        +'<text x="455" y="273" font-size="13" font-weight="900" fill="#1D4ED8">全身反应需另行区分</text>';
    }

    function update(){
      var pathogen=Number(pathogenInput.value);
      var barrier=Number(barrierInput.value);
      var mediator=Number(mediatorInput.value);
      var phagocyte=Number(phagocyteInput.value);
      var responseTime=Number(timeInput.value);

      pathogenValue.textContent=pathogen.toFixed(0)+'%';
      barrierValue.textContent=barrier.toFixed(0)+'%';
      mediatorValue.textContent=mediator.toFixed(0)+'%';
      phagocyteValue.textContent=phagocyte.toFixed(0)+'%';
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
        ?'反应推进：运行中'
        :'反应推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var progress=responseTime/100;

      var entered=pathogen
        *(1-barrier/100)
        *(.22+.78*progress);

      entered=clamp(
        entered,
        0,
        100
      );

      var inflammation=100*Math.sqrt(
        entered/100
        *(.20+.80*mediator/100)
      );

      inflammation=clamp(
        inflammation,
        0,
        100
      );

      var clearance=entered
        *phagocyte/100
        *(.18+.82*progress)
        *.78;

      var remaining=clamp(
        entered-clearance,
        0,
        100
      );

      var excessiveDamage=clamp(
        inflammation
        *(mediator/100)
        *(.25+.75*progress)
        -56,
        0,
        44
      );

      enteredText.textContent=entered.toFixed(0);
      inflammationText.textContent=inflammation.toFixed(0);
      remainingText.textContent=remaining.toFixed(0);

      root.style.setProperty(
        '--ii-speed',
        clamp(
          2.45-inflammation/75,
          .65,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='barrier'){
        title.textContent='第一道防线：屏障与病原体进入';
        summary.textContent='观察屏障完整度如何影响病原体突破和进入组织。';

        dynamic.innerHTML=renderBarrier(
          pathogen,
          barrier,
          progress
        );

        stageNote.textContent=
          '皮肤和黏膜屏障可在病原体进入机体前发挥阻挡作用。';

        renderLabels(mode);
      }else if(mode==='vascular'){
        title.textContent='炎症介质与局部血管反应';
        summary.textContent='观察毛细血管扩张、局部血流和通透性变化。';

        dynamic.innerHTML=renderVascular(
          mediator,
          inflammation,
          progress
        );

        stageNote.textContent=
          '血管扩张和通透性增加有利于免疫细胞及液体进入受损组织。';

        renderLabels(mode);
      }else if(mode==='phagocytosis'){
        title.textContent='吞噬细胞的趋化、迁出与吞噬';
        summary.textContent='观察中性粒细胞和巨噬细胞向炎症部位移动。';

        dynamic.innerHTML=renderPhagocytosis(
          entered,
          phagocyte,
          progress
        );

        stageNote.textContent=
          '中性粒细胞通常较早到达，巨噬细胞参与吞噬病原体和细胞碎片。';

        renderLabels(mode);
      }else{
        title.textContent='局部红、肿、热、痛及炎症边界';
        summary.textContent='比较炎症的防御作用、局部表现和过强反应的损伤风险。';

        dynamic.innerHTML=renderEffects(
          inflammation,
          mediator,
          responseTime
        );

        stageNote.textContent=
          '局部炎症不等同于全身发热，二者涉及的范围和调节机制不同。';

        renderLabels(mode);
      }

      var condition=
        '当前屏障、炎症介质和吞噬细胞活动共同限制病原体扩散。';

      if(pathogen<10){
        condition=
          '病原体负荷较低，当前仅形成较弱的先天免疫和炎症信号。';
      }else if(barrier>88){
        condition=
          '屏障较完整，大部分病原体在进入组织前被阻挡。';
      }else if(mediator<18&&entered>20){
        condition=
          '进入组织的病原体较多，但炎症介质释放较低，免疫细胞募集受到限制。';
      }else if(phagocyte<18&&entered>20){
        condition=
          '吞噬细胞活跃度较低，病原体和组织碎片的清除速度下降。';
      }else if(excessiveDamage>12){
        condition=
          '炎症介质水平较高且反应持续时间较长，组织损伤风险上升。';
      }

      var principle=mode==='barrier'
        ?'皮肤和黏膜屏障属于重要的第一道防线；屏障被破坏后，病原体更容易进入组织。'
        :mode==='vascular'
          ?'炎症介质可促进局部血管扩张和通透性增加，从而增加血流并募集免疫细胞。'
          :mode==='phagocytosis'
            ?'先天免疫细胞可通过趋化到达炎症部位，并吞噬病原体和细胞碎片。'
            :'红、肿、热、痛是多种血管、神经和组织变化共同作用的结果。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 先天免疫反应启动较快，但缺乏针对特定抗原的高度特异性；过强或持续的炎症反应也可能造成组织损伤。';
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

    pathogenInput.oninput=update;
    barrierInput.oninput=update;
    mediatorInput.oninput=update;
    phagocyteInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
