/**
 * lifeScienceLabTemplatesHumanFertilizationImplantation.ts
 *
 * 平面生命科学实验室：受精、卵裂与着床。
 *
 * 教学目标：
 * 1. 观察精子接近、识别并进入卵细胞的受精过程；
 * 2. 理解正常情况下通常只有一个精子与卵细胞完成融合；
 * 3. 观察受精卵经过二细胞期、四细胞期、桑椹胚和囊胚等早期阶段；
 * 4. 理解卵裂过程中细胞数增加，但胚胎总体积通常不明显增大；
 * 5. 观察早期胚胎由输卵管向子宫方向运输并最终着床；
 * 6. 区分受精、卵裂、囊胚形成和着床等不同环节。
 *
 * 科学边界：
 * 1. 人体受精通常发生在输卵管壶腹部附近，而不是子宫；
 * 2. 精子获能、顶体反应、精卵识别和膜融合等过程在图中作教学简化；
 * 3. 一个精子进入后，卵细胞发生的皮质反应等机制有助于阻止多精入卵；
 * 4. 受精卵形成后进行连续卵裂，细胞数增加而总体积通常不明显增大；
 * 5. 桑椹胚继续发育形成囊胚，囊胚通常在脱离透明带后开始着床；
 * 6. 着床通常发生在子宫内膜，不等同于受精；
 * 7. 受精卵、卵裂胚和囊胚均属于早期胚胎阶段，不应称为胎儿；
 * 8. 图中的时间、细胞数量、成功指数和运输速度均为相对教学指标；
 * 9. 本模板只用于生物学教学，不用于医学诊断、生育预测或个体健康评价。
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

function fertilizationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FBCFE8;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#EDE9FE);border-bottom:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFF9FC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#EC4899}'
    + '#' + rootId + ' .fi-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .fi-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .fi-stages{display:grid;grid-template-columns:repeat(7,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .fi-button{min-height:29px;padding:3px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:9.2px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .fi-button.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.13)}'
    + '#' + rootId + ' .fi-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .fi-toggle.off{background:#64748B}'
    + '#' + rootId + ' .fi-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .fi-card{padding:6px 3px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .fi-card b{display:block;font-size:13px;color:#BE185D}'
    + '#' + rootId + ' .fi-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .fi-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--fi-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .fi-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_FERTILIZATION_IMPLANTATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-fertilization-cleavage-implantation',
    group: '🧑 人体生殖与发育',
    name: '受精、卵裂与着床',
    emoji: '🧫',
    desc: '调节精子活力、卵细胞成熟度、输卵管运输、子宫内膜状态和发育时间，观察受精、卵裂与着床',
    params: [
      {
        key: 'spermActivity',
        label: '精子相对活力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'oocyteMaturity',
        label: '卵细胞成熟度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 84,
      },
      {
        key: 'tubeTransport',
        label: '输卵管运输条件',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'endometriumReceptivity',
        label: '子宫内膜容受状态',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 80,
      },
      {
        key: 'developmentTime',
        label: '过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 46,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const spermActivity = num(params, 'spermActivity', 76)
      const oocyteMaturity = num(params, 'oocyteMaturity', 84)
      const tubeTransport = num(params, 'tubeTransport', 78)
      const endometriumReceptivity = num(
        params,
        'endometriumReceptivity',
        80,
      )
      const developmentTime = num(params, 'developmentTime', 46)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${fertilizationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧫 受精、卵裂与着床</div>
    <div class="bl-note">受精位置、卵裂过程和着床位置均为教学示意</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>精子相对活力</span>
          <span class="bl-value" data-sperm-value></span>
        </div>
        <input
          data-sperm
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(spermActivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>卵细胞成熟度</span>
          <span class="bl-value" data-oocyte-value></span>
        </div>
        <input
          data-oocyte
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(oocyteMaturity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>输卵管运输条件</span>
          <span class="bl-value" data-tube-value></span>
        </div>
        <input
          data-tube
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(tubeTransport)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>子宫内膜容受状态</span>
          <span class="bl-value" data-endometrium-value></span>
        </div>
        <input
          data-endometrium
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(endometriumReceptivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>过程时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input
          data-time
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(developmentTime)}"
        >
      </div>

      <div class="fi-subtitle">观察方式</div>

      <div class="fi-buttons">
        <button
          type="button"
          class="fi-button active"
          data-mode="fertilization"
        >受精过程</button>

        <button
          type="button"
          class="fi-button"
          data-mode="cleavage"
        >卵裂过程</button>

        <button
          type="button"
          class="fi-button"
          data-mode="implantation"
        >着床过程</button>
      </div>

      <div class="fi-subtitle">快速查看阶段</div>

      <div class="fi-stages">
        <button type="button" class="fi-button active" data-stage="0">接近</button>
        <button type="button" class="fi-button" data-stage="1">识别</button>
        <button type="button" class="fi-button" data-stage="2">入卵</button>
        <button type="button" class="fi-button" data-stage="3">受精卵</button>
        <button type="button" class="fi-button" data-stage="4">卵裂</button>
        <button type="button" class="fi-button" data-stage="5">囊胚</button>
        <button type="button" class="fi-button" data-stage="6">着床</button>
      </div>

      <button
        type="button"
        class="fi-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="fi-toggle"
        data-auto
      >过程推进：运行中</button>

      <div class="fi-status">
        <div class="fi-card">
          <b data-stage-name></b>
          <span>当前阶段</span>
        </div>

        <div class="fi-card">
          <b data-cell-count></b>
          <span>相对细胞数</span>
        </div>

        <div class="fi-card">
          <b data-outcome></b>
          <span>当前状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="受精、卵裂与着床互动示意图"
      >
        <defs>
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
              flood-color="#831843"
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
          fill="#9D174D"
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

        <g transform="translate(520 337)">
          <rect
            width="214"
            height="66"
            rx="15"
            fill="#FFF1F2"
            stroke="#FBCFE8"
            stroke-width="2"
          />

          <text
            x="107"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#9D174D"
          >关键区分</text>

          <text
            x="107"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#831843"
          >受精通常发生在输卵管</text>

          <text
            x="107"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#831843"
          >着床通常发生在子宫内膜</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#9D174D"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var spermInput=root.querySelector('[data-sperm]');
    var oocyteInput=root.querySelector('[data-oocyte]');
    var tubeInput=root.querySelector('[data-tube]');
    var endometriumInput=root.querySelector('[data-endometrium]');
    var timeInput=root.querySelector('[data-time]');

    var spermValue=root.querySelector('[data-sperm-value]');
    var oocyteValue=root.querySelector('[data-oocyte-value]');
    var tubeValue=root.querySelector('[data-tube-value]');
    var endometriumValue=root.querySelector('[data-endometrium-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var stageButtons=root.querySelectorAll('[data-stage]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var stageNameText=root.querySelector('[data-stage-name]');
    var cellCountText=root.querySelector('[data-cell-count]');
    var outcomeText=root.querySelector('[data-outcome]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='fertilization';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    var stageNames=[
      '精子接近',
      '精卵识别',
      '单精入卵',
      '受精卵形成',
      '卵裂进行',
      '囊胚形成',
      '开始着床'
    ];

    var stageTimes=[
      5,
      18,
      31,
      43,
      59,
      78,
      96
    ];

    var stageNotes=[
      '多个精子向卵细胞移动，但只有少数能够接近卵细胞表面。',
      '精子与卵细胞外层结构发生识别，顶体反应等过程开始。',
      '通常只有一个精子与卵细胞膜融合，随后阻止其他精子进入。',
      '雌、雄原核形成并结合，形成一个受精卵。',
      '受精卵连续卵裂，细胞数增加但总体积通常不明显增大。',
      '桑椹胚继续发育为具有囊胚腔的囊胚，并逐渐脱离透明带。',
      '囊胚与子宫内膜接触并逐渐嵌入，开始建立着床关系。'
    ];

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
      },760);
    }

    function resolveStage(progress){
      if(progress<.12)return 0;
      if(progress<.25)return 1;
      if(progress<.38)return 2;
      if(progress<.50)return 3;
      if(progress<.68)return 4;
      if(progress<.86)return 5;
      return 6;
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
        +'<path d="M-7 0 Q-20 -8 -33 0 Q-45 8 -57 0" fill="none" stroke="#2563EB" stroke-width="3" stroke-linecap="round"/>'
        +'</g>';
    }

    function embryoCells(
      cx,
      cy,
      count,
      radius,
      outerRadius
    ){
      var html='';
      var rows=Math.ceil(Math.sqrt(count));
      var spacing=radius*1.65;

      for(var i=0;i<count;i++){
        var row=Math.floor(i/rows);
        var column=i%rows;
        var usedInRow=Math.min(
          rows,
          count-row*rows
        );

        var x=cx
          +(column-(usedInRow-1)/2)*spacing;

        var totalRows=Math.ceil(
          count/rows
        );

        var y=cy
          +(row-(totalRows-1)/2)*spacing;

        html+='<circle cx="'+x.toFixed(1)
          +'" cy="'+y.toFixed(1)
          +'" r="'+radius
          +'" fill="'
          +(i%2===0?'#FBCFE8':'#DDD6FE')
          +'" stroke="'
          +(i%2===0?'#BE185D':'#7C3AED')
          +'" stroke-width="2.5"/>';
      }

      return '<circle cx="'+cx+'" cy="'+cy
        +'" r="'+outerRadius
        +'" fill="#FFF7ED" stroke="#F59E0B" stroke-width="5"/>'
        +html;
    }

    function renderFertilization(
      stageIndex,
      fertilizationPotential
    ){
      var ovumX=443;
      var ovumY=218;
      var spermHTML='';
      var spermCount=Math.floor(
        6+fertilizationPotential*8
      );

      for(var i=0;i<spermCount;i++){
        var angle=-1.25+i*.31;
        var distance=200-i%4*18;

        if(stageIndex>=1){
          distance=140-i%4*15;
        }

        if(stageIndex>=2){
          distance=104-i%4*12;
        }

        var x=ovumX-Math.cos(angle)*distance;
        var y=ovumY+Math.sin(angle)*distance*.72;
        var rotation=Math.atan2(
          ovumY-y,
          ovumX-x
        )*180/Math.PI;

        var opacity=.40+.50*fertilizationPotential;

        spermHTML+=spermShape(
          x.toFixed(1),
          y.toFixed(1),
          rotation.toFixed(1),
          .72,
          opacity.toFixed(2)
        );
      }

      var entering='';

      if(stageIndex>=2){
        entering=spermShape(
          370,
          218,
          0,
          .84,
          1
        );
      }

      var pronuclei='';

      if(stageIndex>=3){
        pronuclei=''
          +'<circle cx="426" cy="210" r="14" fill="#60A5FA" stroke="#1D4ED8" stroke-width="3"/>'
          +'<circle cx="460" cy="226" r="14" fill="#F9A8D4" stroke="#BE185D" stroke-width="3"/>'
          +'<path class="fi-flow" d="M426 210 Q443 188 460 226" fill="none" stroke="#7C3AED" stroke-width="4" marker-end="url(#${rootId}-arrow-pink)"/>'
          +'<text x="443" y="271" text-anchor="middle" font-size="12" font-weight="900" fill="#6D28D9">雌、雄原核接近</text>';
      }

      var blockRing=stageIndex>=2
        ?'<circle class="fi-pulse" cx="'+ovumX+'" cy="'+ovumY+'" r="90" fill="none" stroke="#16A34A" stroke-width="6" stroke-dasharray="8 7"/>'
        :'';

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M42 153 C160 78 262 90 351 159 C430 221 519 227 708 145" fill="none" stroke="#FBCFE8" stroke-width="91" stroke-linecap="round"/>'
        +'<path d="M42 153 C160 78 262 90 351 159 C430 221 519 227 708 145" fill="none" stroke="#DB2777" stroke-width="7" stroke-linecap="round"/>'
        +'</g>'
        +'<text x="84" y="99" font-size="14" font-weight="900" fill="#9D174D">输卵管壶腹部附近</text>'
        +'<circle cx="'+ovumX+'" cy="'+ovumY+'" r="78" fill="#FEF3C7" stroke="#F59E0B" stroke-width="7"/>'
        +'<circle cx="'+ovumX+'" cy="'+ovumY+'" r="59" fill="#FFF7ED" stroke="#FDBA74" stroke-width="5"/>'
        +'<circle cx="'+ovumX+'" cy="'+ovumY+'" r="24" fill="#F9A8D4" stroke="#BE185D" stroke-width="4"/>'
        +'<circle cx="'+ovumX+'" cy="'+ovumY+'" r="86" fill="none" stroke="#FDE68A" stroke-width="9" opacity=".62"/>'
        +spermHTML
        +entering
        +blockRing
        +pronuclei
        +'<g transform="translate(548 293)">'
        +'<rect width="176" height="81" rx="16" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<text x="88" y="24" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">正常受精关键点</text>'
        +'<text x="15" y="47" font-size="10.5" font-weight="800" fill="#475569">• 通常一个精子进入</text>'
        +'<text x="15" y="66" font-size="10.5" font-weight="800" fill="#475569">• 形成阻止多精入卵的机制</text>'
        +'</g>';
    }

    function renderCleavage(stageIndex){
      var stages=[
        {
          count:1,
          label:'受精卵',
          size:43
        },
        {
          count:2,
          label:'二细胞期',
          size:25
        },
        {
          count:4,
          label:'四细胞期',
          size:20
        },
        {
          count:8,
          label:'八细胞期',
          size:14
        },
        {
          count:16,
          label:'桑椹胚',
          size:9
        },
        {
          count:22,
          label:'囊胚',
          size:7
        }
      ];

      var selected=Math.max(
        0,
        Math.min(
          stages.length-1,
          stageIndex-1
        )
      );

      if(stageIndex<=2){
        selected=0;
      }else if(stageIndex===3){
        selected=0;
      }else if(stageIndex===4){
        selected=3;
      }else if(stageIndex===5){
        selected=5;
      }else{
        selected=5;
      }

      var sequenceHTML='';

      for(var i=0;i<stages.length;i++){
        var x=74+i*111;
        var visible=i<=selected;

        sequenceHTML+='<g opacity="'+(visible?1:.22)+'">'
          +embryoCells(
            x,
            181,
            stages[i].count,
            Math.min(stages[i].size,15),
            43
          )
          +'<text x="'+x+'" y="247" text-anchor="middle" font-size="11" font-weight="900" fill="#831843">'
          +stages[i].label
          +'</text>'
          +'</g>';

        if(i<stages.length-1){
          sequenceHTML+='<path class="fi-flow" d="M'
            +(x+47)+' 181 H'
            +(x+64)
            +'" fill="none" stroke="#DB2777" stroke-width="4" marker-end="url(#${rootId}-arrow-pink)"/>';
        }
      }

      var selectedData=stages[selected];

      return ''
        +sequenceHTML
        +'<g transform="translate(130 292)">'
        +'<rect width="480" height="78" rx="18" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="240" y="25" text-anchor="middle" font-size="14" font-weight="900" fill="#9A3412">'
        +'当前：'+selectedData.label
        +'</text>'
        +'<text x="240" y="49" text-anchor="middle" font-size="12" font-weight="800" fill="#7C2D12">'
        +'细胞数增加，但早期胚胎总体积通常不明显增大'
        +'</text>'
        +'<text x="240" y="68" text-anchor="middle" font-size="11" font-weight="800" fill="#64748B">'
        +'每次卵裂使原有细胞分成体积更小的卵裂球'
        +'</text>'
        +'</g>';
    }

    function renderImplantation(
      stageIndex,
      implantationProgress,
      endometrium
    ){
      var blastocystX=248
        +implantationProgress*220;

      var blastocystY=174
        +implantationProgress*91;

      var embedDepth=clamp(
        (implantationProgress-.55)/.45,
        0,
        1
      );

      var endometriumHeight=38
        +endometrium*.34;

      var cells='';

      for(var i=0;i<20;i++){
        var angle=Math.PI*2*i/20;
        var x=blastocystX+Math.cos(angle)*35;
        var y=blastocystY+Math.sin(angle)*29;

        cells+='<circle cx="'+x.toFixed(1)
          +'" cy="'+y.toFixed(1)
          +'" r="6" fill="'
          +(i%2===0?'#F9A8D4':'#C4B5FD')
          +'" stroke="'
          +(i%2===0?'#BE185D':'#7C3AED')
          +'" stroke-width="1.7"/>';
      }

      var innerMass=''
        +'<g>'
        +'<circle cx="'+(blastocystX-12).toFixed(1)
        +'" cy="'+(blastocystY+5).toFixed(1)
        +'" r="9" fill="#F472B6" stroke="#BE185D" stroke-width="2"/>'
        +'<circle cx="'+(blastocystX+1).toFixed(1)
        +'" cy="'+(blastocystY+11).toFixed(1)
        +'" r="9" fill="#F472B6" stroke="#BE185D" stroke-width="2"/>'
        +'<circle cx="'+(blastocystX-1).toFixed(1)
        +'" cy="'+(blastocystY-2).toFixed(1)
        +'" r="9" fill="#F472B6" stroke="#BE185D" stroke-width="2"/>'
        +'</g>';

      var contact='';

      if(implantationProgress>.50){
        contact='<path class="fi-pulse" d="M'
          +(blastocystX-35).toFixed(1)+' '
          +(blastocystY+26).toFixed(1)
          +' Q'
          +blastocystX.toFixed(1)+' '
          +(blastocystY+50+embedDepth*18).toFixed(1)
          +' '
          +(blastocystX+35).toFixed(1)+' '
          +(blastocystY+26).toFixed(1)
          +'" fill="none" stroke="#16A34A" stroke-width="6" stroke-dasharray="7 6"/>';
      }

      return ''
        +'<path d="M80 104 C190 66 340 71 475 105 C587 133 654 197 676 293" fill="none" stroke="#FBCFE8" stroke-width="58" stroke-linecap="round"/>'
        +'<path class="fi-flow" d="M92 105 C204 72 332 78 452 107" fill="none" stroke="#DB2777" stroke-width="6" marker-end="url(#${rootId}-arrow-pink)"/>'
        +'<path d="M102 298 Q380 230 666 298 V378 H102Z" fill="#F9A8D4" stroke="#BE185D" stroke-width="5"/>'
        +'<path d="M102 '+(298-endometriumHeight*.20)
        +' Q380 '+(230-endometriumHeight*.22)
        +' 666 '+(298-endometriumHeight*.20)
        +' V'
        +(298+endometriumHeight*.30)
        +' Q380 '
        +(230+endometriumHeight*.35)
        +' 102 '
        +(298+endometriumHeight*.30)
        +'Z" fill="#FBCFE8" stroke="#DB2777" stroke-width="3"/>'
        +'<g filter="url(#${rootId}-shadow)">'
        +'<circle cx="'+blastocystX.toFixed(1)
        +'" cy="'+blastocystY.toFixed(1)
        +'" r="45" fill="#FFF7ED" stroke="#F59E0B" stroke-width="5"/>'
        +cells
        +innerMass
        +'</g>'
        +contact
        +'<g transform="translate(515 100)">'
        +'<rect width="195" height="158" rx="20" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="97" y="27" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">着床所需环节</text>'
        +'<text x="18" y="59" font-size="11.5" font-weight="900" fill="#9D174D">1　囊胚到达子宫</text>'
        +'<text x="18" y="86" font-size="11.5" font-weight="900" fill="#C2410C">2　脱离透明带</text>'
        +'<text x="18" y="113" font-size="11.5" font-weight="900" fill="#047857">3　接触并附着子宫内膜</text>'
        +'<text x="18" y="140" font-size="11.5" font-weight="900" fill="#6D28D9">4　逐渐嵌入内膜</text>'
        +'</g>'
        +'<text x="164" y="367" font-size="13" font-weight="900" fill="#9D174D">子宫内膜</text>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='fertilization'){
        labels.innerHTML=''
          +'<path d="M443 132 L514 94" stroke="#F59E0B" stroke-width="2.5"/>'
          +'<text x="522" y="98" font-size="13" font-weight="900" fill="#A16207">透明带</text>'
          +'<path d="M443 218 L543 217" stroke="#BE185D" stroke-width="2.5"/>'
          +'<text x="551" y="222" font-size="13" font-weight="900" fill="#9D174D">卵细胞</text>'
          +'<path d="M316 218 L231 244" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="173" y="250" font-size="13" font-weight="900" fill="#1D4ED8">精子</text>';
        return;
      }

      if(modeName==='cleavage'){
        labels.innerHTML=''
          +'<path d="M74 133 L74 88" stroke="#F59E0B" stroke-width="2.5"/>'
          +'<text x="35" y="82" font-size="13" font-weight="900" fill="#A16207">透明带</text>'
          +'<path d="M518 135 L565 91" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="573" y="88" font-size="13" font-weight="900" fill="#6D28D9">卵裂球</text>'
          +'<path d="M629 181 L695 181" stroke="#BE185D" stroke-width="2.5"/>'
          +'<text x="649" y="206" font-size="13" font-weight="900" fill="#9D174D">囊胚</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M388 259 L454 207" stroke="#F59E0B" stroke-width="2.5"/>'
        +'<text x="463" y="205" font-size="13" font-weight="900" fill="#A16207">囊胚</text>'
        +'<path d="M398 291 L477 317" stroke="#DB2777" stroke-width="2.5"/>'
        +'<text x="486" y="322" font-size="13" font-weight="900" fill="#BE185D">子宫内膜</text>';
    }

    function update(){
      var sperm=Number(spermInput.value);
      var oocyte=Number(oocyteInput.value);
      var tube=Number(tubeInput.value);
      var endometrium=Number(endometriumInput.value);
      var time=Number(timeInput.value);

      spermValue.textContent=sperm.toFixed(0)+'%';
      oocyteValue.textContent=oocyte.toFixed(0)+'%';
      tubeValue.textContent=tube.toFixed(0)+'%';
      endometriumValue.textContent=endometrium.toFixed(0)+'%';
      timeValue.textContent=time.toFixed(0)+'%';

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
        ?'过程推进：运行中'
        :'过程推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var fertilizationPotential=Math.sqrt(
        sperm/100
        *oocyte/100
        *(.20+.80*tube/100)
      );

      var developmentalPotential=.25
        +.75*Math.sqrt(
          oocyte/100
          *(.30+.70*tube/100)
        );

      var progress=time/100
        *(.22+.78*fertilizationPotential)
        *developmentalPotential;

      progress=clamp(
        progress,
        0,
        1
      );

      var stageIndex=resolveStage(progress);

      for(var j=0;j<stageButtons.length;j++){
        stageButtons[j].classList.toggle(
          'active',
          Number(stageButtons[j].getAttribute('data-stage'))===stageIndex
        );
      }

      var cellCount=1;

      if(stageIndex===4){
        cellCount=8;
      }else if(stageIndex===5){
        cellCount=32;
      }else if(stageIndex===6){
        cellCount=64;
      }

      var implantationProgress=clamp(
        (progress-.68)/.32
        *(.25+.75*endometrium/100),
        0,
        1
      );

      var outcome=stageIndex<2
        ?'接近卵细胞'
        :stageIndex===2
          ?'单精受精'
          :stageIndex===3
            ?'形成受精卵'
            :stageIndex===4
              ?'卵裂进行'
              :stageIndex===5
                ?'囊胚形成'
                :implantationProgress>.72
                  ?'着床进行'
                  :'等待附着';

      stageNameText.textContent=stageNames[stageIndex];
      cellCountText.textContent=stageIndex<4
        ?'1'
        :stageIndex===4
          ?'2–16'
          :stageIndex===5
            ?'数十'
            :'持续增加';

      outcomeText.textContent=outcome;

      root.style.setProperty(
        '--fi-speed',
        clamp(
          2.45-progress*1.7,
          .65,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='fertilization'){
        title.textContent='输卵管内的受精过程';
        summary.textContent='观察精子接近、精卵识别、单精入卵和受精卵形成。';
        stageNote.textContent=stageNotes[Math.min(stageIndex,3)];

        dynamic.innerHTML=renderFertilization(
          stageIndex,
          fertilizationPotential
        );

        renderLabels(mode);
      }else if(mode==='cleavage'){
        title.textContent='受精卵的连续卵裂';
        summary.textContent='观察受精卵、卵裂球、桑椹胚和囊胚的形成。';
        stageNote.textContent=stageIndex<3
          ?'完成受精后，受精卵才会进入后续卵裂阶段。'
          :stageNotes[Math.max(3,Math.min(stageIndex,5))];

        dynamic.innerHTML=renderCleavage(stageIndex);
        renderLabels(mode);
      }else{
        title.textContent='囊胚运输与子宫内膜着床';
        summary.textContent='观察囊胚到达子宫、脱离透明带、附着和嵌入内膜。';
        stageNote.textContent=stageIndex<5
          ?'早期胚胎尚未发育到能够着床的囊胚阶段。'
          :stageNotes[Math.min(stageIndex,6)];

        dynamic.innerHTML=renderImplantation(
          stageIndex,
          implantationProgress,
          endometrium
        );

        renderLabels(mode);
      }

      var condition='当前精子活力、卵细胞成熟度、输卵管运输和子宫内膜状态相对协调。';

      if(sperm<18){
        condition='精子相对活力较低，接近和识别卵细胞的教学成功指数下降。';
      }else if(oocyte<18){
        condition='卵细胞成熟度较低，精卵识别、融合和后续发育受到限制。';
      }else if(tube<18){
        condition='输卵管运输条件较低，生殖细胞和早期胚胎的相对运输进度下降。';
      }else if(
        mode==='implantation'
        &&endometrium<20
      ){
        condition='子宫内膜容受状态较低，囊胚附着和嵌入的相对进度受到限制。';
      }else if(time<12){
        condition='过程时间较短，当前主要处于精子接近或精卵识别阶段。';
      }

      var principle=mode==='fertilization'
        ?'人体受精通常发生在输卵管壶腹部附近；一个精子进入后，皮质反应等机制有助于阻止多精入卵。'
        :mode==='cleavage'
          ?'卵裂过程中细胞数增加，但早期胚胎总体积通常不明显增大；桑椹胚继续发育形成囊胚。'
          :'囊胚通常在脱离透明带后与子宫内膜接触并开始着床；着床与受精不是同一过程。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 受精卵、卵裂胚和囊胚均属于早期胚胎阶段，不应称为胎儿。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<stageButtons.length;j++){
      stageButtons[j].onclick=function(){
        var stageIndex=Number(
          this.getAttribute('data-stage')
        );

        timeInput.value=String(
          stageTimes[stageIndex]
        );

        if(stageIndex<=3){
          mode='fertilization';
        }else if(stageIndex<=5){
          mode='cleavage';
        }else{
          mode='implantation';
        }

        automatic=false;
        update();
        schedule();
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

    spermInput.oninput=update;
    oocyteInput.oninput=update;
    tubeInput.oninput=update;
    endometriumInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
