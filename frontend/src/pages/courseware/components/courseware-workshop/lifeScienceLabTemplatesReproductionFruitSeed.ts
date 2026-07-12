/**
 * lifeScienceLabTemplatesReproductionFruitSeed.ts
 *
 * 平面生命科学实验室：果实和种子的形成。
 *
 * 教学目标：
 * 1. 建立子房、胚珠、受精卵和初生胚乳核与发育结果之间的对应关系；
 * 2. 观察受精后子房逐渐膨大、果实形成以及种子发育的过程；
 * 3. 比较受精成功率、胚珠数量、营养和水分供应对相对发育结果的影响；
 * 4. 观察果皮、种皮、胚和胚乳等结构的来源；
 * 5. 区分果实形成、种子形成和种子内部结构形成三个层次。
 *
 * 教学边界：
 * 1. 子房通常发育成果实，子房壁通常发育成果皮；
 * 2. 胚珠通常发育成种子，珠被通常发育成种皮；
 * 3. 受精卵通常发育成胚，初生胚乳核通常发育成胚乳；
 * 4. 有些成熟种子中的胚乳会被子叶吸收，本模型采用保留胚乳的结构进行示意；
 * 5. 少数植物能够发生单性结实，本模型不展开该例外；
 * 6. 模板中的数量、大小、成熟速度均为相对教学指标，不代表真实物种的精确数据。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript，不依赖外部图片、脚本、样式或CDN；
 * 2. 所有DOM查询均限定在rootId内部，支持同页多个组件实例；
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
 * 使用.bl-*公共类名，使模板进入课件后自动转为：
 * 上方互动主体 + 底部课堂控制条。
 */
function fruitSeedStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FED7AA;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FFEDD5,#F0FDF4);border-bottom:1px solid #FED7AA}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9A3412}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFFCF8;border-right:1px solid #FED7AA}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#EA580C;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#F97316}'
    + '#' + rootId + ' .fs-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#9A3412}'
    + '#' + rootId + ' .fs-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .fs-buttons.four{grid-template-columns:repeat(4,1fr)}'
    + '#' + rootId + ' .fs-stages{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .fs-button{min-height:29px;padding:3px;border:1px solid #FDBA74;border-radius:8px;background:#fff;color:#9A3412;font-size:9.4px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .fs-button.active{border-color:#EA580C;background:#FFEDD5;box-shadow:0 3px 9px rgba(234,88,12,.13)}'
    + '#' + rootId + ' .fs-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#FB923C,#EA580C);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .fs-toggle.off{background:#64748B}'
    + '#' + rootId + ' .fs-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .fs-card{padding:6px 3px;border:1px solid #FED7AA;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .fs-card b{display:block;font-size:14px;color:#C2410C}'
    + '#' + rootId + ' .fs-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FFEDD5;color:#7C2D12;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .fs-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--fs-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .fs-pulse{animation:' + rootId + '-pulse 1.7s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.48}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_FRUIT_SEED:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-fruit-seed-development',
    group: '🌸 生殖与个体发育',
    name: '果实和种子的形成',
    emoji: '🍎',
    desc: '调节胚珠数量、受精成功率、营养、水分和发育时间，观察子房成果实、胚珠成种子的过程',
    params: [
      {
        key: 'ovuleCount',
        label: '胚珠数量',
        type: 'number',
        min: 1,
        max: 8,
        step: 1,
        defaultValue: 4,
      },
      {
        key: 'fertilizationRate',
        label: '受精成功率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'nutrientSupply',
        label: '营养供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'waterSupply',
        label: '水分供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'developmentTime',
        label: '发育时间',
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
      const ovuleCount = num(params, 'ovuleCount', 4)
      const fertilizationRate = num(params, 'fertilizationRate', 78)
      const nutrientSupply = num(params, 'nutrientSupply', 76)
      const waterSupply = num(params, 'waterSupply', 72)
      const developmentTime = num(params, 'developmentTime', 52)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${fruitSeedStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🍎 果实和种子的形成</div>
    <div class="bl-note">来源、发育过程和成熟剖面三种观察方式</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>胚珠数量</span>
          <span class="bl-value" data-ovule-value></span>
        </div>
        <input
          data-ovule-count
          type="range"
          min="1"
          max="8"
          step="1"
          value="${n(ovuleCount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>受精成功率</span>
          <span class="bl-value" data-fertilization-value></span>
        </div>
        <input
          data-fertilization
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(fertilizationRate)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>营养供应</span>
          <span class="bl-value" data-nutrient-value></span>
        </div>
        <input
          data-nutrient
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(nutrientSupply)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>水分供应</span>
          <span class="bl-value" data-water-value></span>
        </div>
        <input
          data-water
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(waterSupply)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>发育时间</span>
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

      <div class="fs-subtitle">观察方式</div>

      <div class="fs-buttons">
        <button
          type="button"
          class="fs-button active"
          data-mode="mapping"
        >来源对应</button>

        <button
          type="button"
          class="fs-button"
          data-mode="development"
        >连续发育</button>

        <button
          type="button"
          class="fs-button"
          data-mode="section"
        >成熟剖面</button>
      </div>

      <div class="fs-subtitle">重点来源</div>

      <div class="fs-buttons four">
        <button
          type="button"
          class="fs-button active"
          data-part="ovary"
        >子房壁</button>

        <button
          type="button"
          class="fs-button"
          data-part="ovule"
        >胚珠</button>

        <button
          type="button"
          class="fs-button"
          data-part="zygote"
        >受精卵</button>

        <button
          type="button"
          class="fs-button"
          data-part="endosperm"
        >胚乳核</button>
      </div>

      <div class="fs-subtitle">快速查看发育阶段</div>

      <div class="fs-stages">
        <button
          type="button"
          class="fs-button active"
          data-stage="0"
        >受精后</button>

        <button
          type="button"
          class="fs-button"
          data-stage="1"
        >幼果</button>

        <button
          type="button"
          class="fs-button"
          data-stage="2"
        >膨大</button>

        <button
          type="button"
          class="fs-button"
          data-stage="3"
        >种熟</button>

        <button
          type="button"
          class="fs-button"
          data-stage="4"
        >果熟</button>
      </div>

      <button
        type="button"
        class="fs-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="fs-toggle"
        data-auto
      >发育推进：运行中</button>

      <div class="fs-status">
        <div class="fs-card">
          <b data-fruit-set></b>
          <span>果实形成指数</span>
        </div>

        <div class="fs-card">
          <b data-seed-count></b>
          <span>预计种子数</span>
        </div>

        <div class="fs-card">
          <b data-stage-name></b>
          <span>当前阶段</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="果实和种子的形成互动示意图"
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

          <linearGradient
            id="${rootId}-young-fruit"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#BEF264"/>
            <stop offset="100%" stop-color="#4D7C0F"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-mature-fruit"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#FDBA74"/>
            <stop offset="100%" stop-color="#EA580C"/>
          </linearGradient>
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
        <g data-highlight></g>
        <g data-labels></g>

        <g transform="translate(522 337)">
          <rect
            width="212"
            height="66"
            rx="15"
            fill="#FFF7ED"
            stroke="#FED7AA"
            stroke-width="2"
          />

          <text
            x="106"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#9A3412"
          >科学边界</text>

          <text
            x="106"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#7C2D12"
          >少数植物可单性结实</text>

          <text
            x="106"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#7C2D12"
          >本模型不展开该例外</text>
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

    var ovuleInput=root.querySelector('[data-ovule-count]');
    var fertilizationInput=root.querySelector('[data-fertilization]');
    var nutrientInput=root.querySelector('[data-nutrient]');
    var waterInput=root.querySelector('[data-water]');
    var timeInput=root.querySelector('[data-time]');

    var ovuleValue=root.querySelector('[data-ovule-value]');
    var fertilizationValue=root.querySelector('[data-fertilization-value]');
    var nutrientValue=root.querySelector('[data-nutrient-value]');
    var waterValue=root.querySelector('[data-water-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var partButtons=root.querySelectorAll('[data-part]');
    var stageButtons=root.querySelectorAll('[data-stage]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var fruitSetText=root.querySelector('[data-fruit-set]');
    var seedCountText=root.querySelector('[data-seed-count]');
    var stageNameText=root.querySelector('[data-stage-name]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var dynamic=root.querySelector('[data-dynamic]');
    var highlight=root.querySelector('[data-highlight]');
    var labels=root.querySelector('[data-labels]');

    var mode='mapping';
    var selectedPart='ovary';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    var stageNames=[
      '受精完成',
      '幼果形成',
      '果实膨大',
      '种子成熟',
      '果实成熟'
    ];

    var stageNotes=[
      '受精完成后，受精卵、初生胚乳核和子房等结构开始进入后续发育。',
      '子房开始膨大，胚珠内部的胚和胚乳逐渐形成。',
      '果实继续膨大，果皮和种子内部结构不断发育。',
      '胚、种皮和胚乳等结构逐渐成熟，种子含水量可能下降。',
      '果实达到成熟阶段，果皮的颜色、质地等可能发生明显变化。'
    ];

    var partNotes={
      ovary:'子房通常发育成果实，子房壁通常发育成果皮。',
      ovule:'胚珠通常发育成种子，珠被通常发育成种皮。',
      zygote:'受精卵通常经过细胞分裂和分化发育成胚。',
      endosperm:'初生胚乳核通常发育成胚乳，为胚的发育或种子萌发提供营养。'
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function resolveStage(development){
      if(development<.10)return 0;
      if(development<.30)return 1;
      if(development<.58)return 2;
      if(development<.82)return 3;
      return 4;
    }

    function setActiveStage(stageIndex){
      for(var i=0;i<stageButtons.length;i++){
        stageButtons[i].classList.toggle(
          'active',
          Number(stageButtons[i].getAttribute('data-stage'))===stageIndex
        );
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
        var next=Number(timeInput.value)+2;
        timeInput.value=String(next>100?0:next);
        update();
        schedule();
      },760);
    }

    function seedHTML(
      cx,
      cy,
      size,
      mature,
      index
    ){
      var coatColor=mature
        ?'#92400E'
        :'#CA8A04';

      var innerColor=mature
        ?'#FEF3C7'
        :'#FEF9C3';

      var embryoColor=mature
        ?'#22C55E'
        :'#86EFAC';

      var rotation=(index%2===0?-14:14);

      return ''
        +'<g transform="rotate('+rotation+' '+cx+' '+cy+')">'
        +'<ellipse cx="'+cx+'" cy="'+cy+'" rx="'+size
        +'" ry="'+(size*.68)+'" fill="'+innerColor
        +'" stroke="'+coatColor+'" stroke-width="'+Math.max(2,size*.16)+'"/>'
        +'<path d="M'+(cx-size*.28)+' '+(cy+size*.10)
        +' Q'+cx+' '+(cy-size*.34)+' '+(cx+size*.30)+' '+(cy+size*.08)
        +'" fill="none" stroke="'+embryoColor
        +'" stroke-width="'+Math.max(2,size*.16)
        +'" stroke-linecap="round"/>'
        +'<circle cx="'+(cx-size*.16)+'" cy="'+(cy-size*.07)
        +'" r="'+Math.max(2,size*.10)+'" fill="#FACC15"/>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='mapping'){
        labels.innerHTML=''
          +'<path d="M175 262 L88 290" stroke="#15803D" stroke-width="2.5"/>'
          +'<text x="28" y="297" font-size="13" font-weight="900" fill="#166534">子房壁</text>'
          +'<path d="M206 228 L95 205" stroke="#CA8A04" stroke-width="2.5"/>'
          +'<text x="32" y="201" font-size="13" font-weight="900" fill="#A16207">胚珠</text>'
          +'<path d="M500 222 L585 183" stroke="#EA580C" stroke-width="2.5"/>'
          +'<text x="594" y="180" font-size="13" font-weight="900" fill="#C2410C">果实</text>'
          +'<path d="M480 259 L582 285" stroke="#92400E" stroke-width="2.5"/>'
          +'<text x="591" y="291" font-size="13" font-weight="900" fill="#92400E">种子</text>';
        return;
      }

      if(modeName==='development'){
        labels.innerHTML=''
          +'<path d="M365 250 L470 228" stroke="#EA580C" stroke-width="2.5"/>'
          +'<text x="478" y="232" font-size="13" font-weight="900" fill="#C2410C">发育中的果皮</text>'
          +'<path d="M365 279 L470 302" stroke="#92400E" stroke-width="2.5"/>'
          +'<text x="478" y="308" font-size="13" font-weight="900" fill="#92400E">发育中的种子</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M420 198 L510 155" stroke="#EA580C" stroke-width="2.5"/>'
        +'<text x="519" y="154" font-size="13" font-weight="900" fill="#C2410C">果皮</text>'
        +'<path d="M390 261 L510 255" stroke="#92400E" stroke-width="2.5"/>'
        +'<text x="519" y="260" font-size="13" font-weight="900" fill="#92400E">种皮</text>'
        +'<path d="M356 249 L510 292" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="519" y="298" font-size="13" font-weight="900" fill="#166534">胚</text>'
        +'<path d="M356 224 L510 333" stroke="#CA8A04" stroke-width="2.5"/>'
        +'<text x="519" y="339" font-size="13" font-weight="900" fill="#A16207">胚乳</text>';
    }

    function renderMapping(
      ovules,
      seedCount,
      stageIndex
    ){
      var ovuleHTML='';
      var seedResultHTML='';

      for(var i=0;i<ovules;i++){
        var angle=Math.PI*2*i/ovules-Math.PI/2;
        var ox=205+Math.cos(angle)*35;
        var oy=232+Math.sin(angle)*25;

        ovuleHTML+='<ellipse cx="'+ox.toFixed(1)+'" cy="'+oy.toFixed(1)
          +'" rx="10" ry="15" fill="#FEF3C7" stroke="#CA8A04" stroke-width="3"/>';

        if(i<seedCount){
          var fruitAngle=Math.PI*2*i/Math.max(1,seedCount)-Math.PI/2;
          var sx=435+Math.cos(fruitAngle)*42;
          var sy=241+Math.sin(fruitAngle)*31;

          seedResultHTML+=seedHTML(
            sx,
            sy,
            15,
            stageIndex>=3,
            i
          );
        }
      }

      var fruitFill=stageIndex>=4
        ?'url(#${rootId}-mature-fruit)'
        :'url(#${rootId}-young-fruit)';

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M205 104 V168" stroke="#65A30D" stroke-width="14" stroke-linecap="round"/>'
        +'<ellipse cx="205" cy="94" rx="26" ry="14" fill="#A3E635" stroke="#3F6212" stroke-width="4"/>'
        +'<ellipse cx="205" cy="231" rx="65" ry="59" fill="#86EFAC" stroke="#15803D" stroke-width="5"/>'
        +'<path d="M205 108 V214" stroke="#65A30D" stroke-width="12" stroke-linecap="round"/>'
        +ovuleHTML
        +'</g>'
        +'<path class="fs-flow" d="M282 231 H343" fill="none" stroke="#EA580C" stroke-width="6" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<text x="312" y="208" text-anchor="middle" font-size="13" font-weight="900" fill="#9A3412">受精后发育</text>'
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M435 139 Q446 110 468 99" fill="none" stroke="#15803D" stroke-width="10" stroke-linecap="round"/>'
        +'<ellipse cx="474" cy="94" rx="28" ry="13" fill="#22C55E" stroke="#15803D" stroke-width="4" transform="rotate(-25 474 94)"/>'
        +'<ellipse cx="435" cy="239" rx="92" ry="102" fill="'+fruitFill+'" stroke="#C2410C" stroke-width="6"/>'
        +'<ellipse cx="435" cy="241" rx="64" ry="74" fill="#FFF7ED" stroke="#FDBA74" stroke-width="4"/>'
        +seedResultHTML
        +'</g>'
        +'<g transform="translate(90 335)">'
        +'<rect width="510" height="45" rx="14" fill="#FFF7ED" stroke="#FED7AA" stroke-width="2"/>'
        +'<text x="20" y="28" font-size="13" font-weight="900" fill="#7C2D12">'
        +'子房→果实　子房壁→果皮　胚珠→种子　受精卵→胚　初生胚乳核→胚乳'
        +'</text>'
        +'</g>';
    }

    function renderDevelopment(
      stageIndex,
      seedCount,
      fruitSet,
      development
    ){
      var stageColors=[
        '#A3E635',
        '#84CC16',
        '#65A30D',
        '#F59E0B',
        '#F97316'
      ];

      var stageHTML='';

      for(var i=0;i<5;i++){
        var x=88+i*112;
        var size=18+i*7;
        var opacity=i<=stageIndex?1:.28;

        stageHTML+='<g opacity="'+opacity+'">'
          +'<path d="M'+x+' 139 V112" stroke="#15803D" stroke-width="6" stroke-linecap="round"/>'
          +'<ellipse cx="'+x+'" cy="'+(174-i*2)+'" rx="'+size
          +'" ry="'+(size*1.12)+'" fill="'+stageColors[i]
          +'" stroke="#C2410C" stroke-width="4"/>'
          +'<text x="'+x+'" y="226" text-anchor="middle" font-size="11" font-weight="900" fill="#7C2D12">'
          +stageNames[i]
          +'</text>'
          +'</g>';

        if(i<4){
          stageHTML+='<path d="M'+(x+35)+' 174 H'+(x+77)
            +'" fill="none" stroke="#FDBA74" stroke-width="4" marker-end="url(#${rootId}-arrow-orange)"/>';
        }
      }

      var fruitSize=34+96*development;
      var currentSeeds='';
      var visibleSeeds=Math.max(0,seedCount);

      for(var s=0;s<visibleSeeds;s++){
        var angle=Math.PI*2*s/Math.max(1,visibleSeeds)-Math.PI/2;
        var sx=350+Math.cos(angle)*fruitSize*.38;
        var sy=292+Math.sin(angle)*fruitSize*.28;

        currentSeeds+=seedHTML(
          sx,
          sy,
          Math.max(8,fruitSize*.10),
          stageIndex>=3,
          s
        );
      }

      var fruitFill=stageIndex>=4
        ?'url(#${rootId}-mature-fruit)'
        :'url(#${rootId}-young-fruit)';

      return stageHTML
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M350 '+(292-fruitSize*.92)+' Q360 '
          +(292-fruitSize-26)+' 382 '+(292-fruitSize-36)
          +'" fill="none" stroke="#15803D" stroke-width="9" stroke-linecap="round"/>'
        +'<ellipse cx="391" cy="'+(292-fruitSize-41)
          +'" rx="25" ry="11" fill="#22C55E" stroke="#15803D" stroke-width="3" transform="rotate(-24 391 '
          +(292-fruitSize-41)+')"/>'
        +'<ellipse cx="350" cy="292" rx="'+fruitSize
          +'" ry="'+(fruitSize*.92)+'" fill="'+fruitFill
          +'" stroke="#C2410C" stroke-width="6"/>'
        +'<ellipse cx="350" cy="292" rx="'+(fruitSize*.69)
          +'" ry="'+(fruitSize*.61)+'" fill="#FFF7ED" stroke="#FDBA74" stroke-width="4"/>'
        +currentSeeds
        +'</g>'
        +'<g transform="translate(520 110)">'
        +'<rect width="190" height="158" rx="20" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="95" y="27" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">当前相对发育</text>'
        +'<text x="20" y="62" font-size="12" font-weight="800" fill="#64748B">果实形成</text>'
        +'<rect x="20" y="73" width="150" height="13" rx="6" fill="#E2E8F0"/>'
        +'<rect x="20" y="73" width="'+(150*fruitSet/100)
          +'" height="13" rx="6" fill="#F97316"/>'
        +'<text x="20" y="111" font-size="12" font-weight="800" fill="#64748B">发育进度</text>'
        +'<rect x="20" y="122" width="150" height="13" rx="6" fill="#E2E8F0"/>'
        +'<rect x="20" y="122" width="'+(150*development)
          +'" height="13" rx="6" fill="#65A30D"/>'
        +'<text x="95" y="151" text-anchor="middle" font-size="11" font-weight="900" fill="#475569">'
        +stageNames[stageIndex]
        +'</text>'
        +'</g>';
    }

    function renderSection(
      seedCount,
      stageIndex
    ){
      var seeds='';
      var count=Math.max(1,seedCount);

      for(var i=0;i<count;i++){
        var angle=Math.PI*2*i/count-Math.PI/2;
        var x=330+Math.cos(angle)*63;
        var y=232+Math.sin(angle)*47;

        seeds+=seedHTML(
          x,
          y,
          24,
          stageIndex>=3,
          i
        );
      }

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M330 100 Q342 70 365 59" fill="none" stroke="#15803D" stroke-width="10" stroke-linecap="round"/>'
        +'<ellipse cx="378" cy="53" rx="28" ry="12" fill="#22C55E" stroke="#15803D" stroke-width="4" transform="rotate(-24 378 53)"/>'
        +'<ellipse cx="330" cy="223" rx="150" ry="139" fill="url(#${rootId}-mature-fruit)" stroke="#C2410C" stroke-width="7"/>'
        +'<ellipse cx="330" cy="223" rx="118" ry="106" fill="#FFEDD5" stroke="#FB923C" stroke-width="6"/>'
        +'<ellipse cx="330" cy="223" rx="84" ry="76" fill="#FFF7ED" stroke="#FDBA74" stroke-width="4"/>'
        +seeds
        +'</g>'
        +'<g transform="translate(505 87)">'
        +'<rect width="205" height="205" rx="22" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="102" y="27" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">种子内部放大示意</text>'
        +'<ellipse cx="102" cy="116" rx="68" ry="54" fill="#FEF3C7" stroke="#92400E" stroke-width="6"/>'
        +'<ellipse cx="102" cy="116" rx="49" ry="39" fill="#FDE68A" stroke="#CA8A04" stroke-width="3"/>'
        +'<path d="M72 125 Q102 75 135 118 Q113 126 101 153 Q87 139 72 125Z" fill="#4ADE80" stroke="#15803D" stroke-width="4"/>'
        +'<circle cx="92" cy="116" r="7" fill="#FACC15"/>'
        +'<text x="102" y="187" text-anchor="middle" font-size="11" font-weight="900" fill="#475569">种皮 · 胚乳 · 胚</text>'
        +'</g>';
    }

    function renderHighlight(
      modeName,
      part
    ){
      var html='';

      if(modeName==='mapping'){
        if(part==='ovary'){
          html='<ellipse cx="205" cy="231" rx="77" ry="71" fill="none" stroke="#EA580C" stroke-width="6" stroke-dasharray="8 6"/>';
        }else if(part==='ovule'){
          html='<ellipse cx="205" cy="231" rx="51" ry="40" fill="none" stroke="#CA8A04" stroke-width="6" stroke-dasharray="8 6"/>';
        }else if(part==='zygote'){
          html='<circle cx="435" cy="241" r="38" fill="none" stroke="#16A34A" stroke-width="6" stroke-dasharray="8 6"/>';
        }else{
          html='<circle cx="435" cy="241" r="66" fill="none" stroke="#F59E0B" stroke-width="6" stroke-dasharray="8 6"/>';
        }
      }else if(modeName==='section'){
        if(part==='ovary'){
          html='<ellipse cx="330" cy="223" rx="159" ry="148" fill="none" stroke="#EA580C" stroke-width="6" stroke-dasharray="8 6"/>';
        }else if(part==='ovule'){
          html='<ellipse cx="330" cy="223" rx="92" ry="84" fill="none" stroke="#92400E" stroke-width="6" stroke-dasharray="8 6"/>';
        }else if(part==='zygote'){
          html='<path d="M562 212 Q607 155 650 206 Q625 220 608 257 Q587 237 562 212Z" fill="none" stroke="#16A34A" stroke-width="6" stroke-dasharray="8 6"/>';
        }else{
          html='<ellipse cx="607" cy="203" rx="58" ry="49" fill="none" stroke="#F59E0B" stroke-width="6" stroke-dasharray="8 6"/>';
        }
      }

      highlight.innerHTML=html;
    }

    function update(){
      var ovules=clamp(
        Math.round(Number(ovuleInput.value)),
        1,
        8
      );

      var fertilization=Number(fertilizationInput.value);
      var nutrient=Number(nutrientInput.value);
      var water=Number(waterInput.value);
      var time=Number(timeInput.value);

      ovuleValue.textContent=ovules.toFixed(0)+' 个';
      fertilizationValue.textContent=fertilization.toFixed(0)+'%';
      nutrientValue.textContent=nutrient.toFixed(0)+'%';
      waterValue.textContent=water.toFixed(0)+'%';
      timeValue.textContent=time.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      for(var j=0;j<partButtons.length;j++){
        partButtons[j].classList.toggle(
          'active',
          partButtons[j].getAttribute('data-part')===selectedPart
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
        ?'发育推进：运行中'
        :'发育推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var fertilizationFactor=fertilization/100;
      var nutrientFactor=.35+.65*nutrient/100;
      var waterFactor=.35+.65*water/100;
      var resourceFactor=Math.sqrt(
        nutrientFactor*waterFactor
      );

      var fruitSet=100
        *fertilizationFactor
        *(.55+.25*nutrient/100+.20*water/100);

      fruitSet=clamp(
        fruitSet,
        0,
        100
      );

      var seedCount=Math.round(
        ovules*fertilizationFactor
      );

      var development=time/100
        *(.18+.82*fruitSet/100)
        *resourceFactor;

      development=clamp(
        development,
        0,
        1
      );

      var stageIndex=resolveStage(development);

      fruitSetText.textContent=fruitSet.toFixed(0);
      seedCountText.textContent=seedCount.toFixed(0)+' / '+ovules;
      stageNameText.textContent=stageNames[stageIndex];

      setActiveStage(stageIndex);

      root.style.setProperty(
        '--fs-speed',
        clamp(
          2.5-development*1.6,
          .65,
          2.4
        ).toFixed(2)+'s'
      );

      title.textContent=mode==='mapping'
        ?'生殖结构与发育结果的对应关系'
        :mode==='development'
          ?'果实和种子的连续发育'
          :'成熟果实与种子剖面';

      summary.textContent='当前阶段：'
        +stageNames[stageIndex]
        +'；预计形成 '
        +seedCount
        +' 粒种子。';

      stageNote.textContent=stageNotes[stageIndex];

      if(mode==='mapping'){
        dynamic.innerHTML=renderMapping(
          ovules,
          seedCount,
          stageIndex
        );
      }else if(mode==='development'){
        dynamic.innerHTML=renderDevelopment(
          stageIndex,
          seedCount,
          fruitSet,
          development
        );
      }else{
        dynamic.innerHTML=renderSection(
          seedCount,
          stageIndex
        );
      }

      renderLabels(mode);
      renderHighlight(mode,selectedPart);

      var condition='当前受精、营养和水分条件能够支持相对正常的果实与种子发育。';

      if(fertilization<10){
        condition='受精成功率很低，种子难以形成，本模型中的子房通常不能继续正常膨大。';
      }else if(nutrient<20){
        condition='营养供应不足，果实膨大和种子内部物质积累受到明显限制。';
      }else if(water<20){
        condition='水分供应不足，细胞生长和物质运输受到限制，发育速度下降。';
      }else if(seedCount===0){
        condition='当前没有胚珠完成受精，因此不能形成正常种子。';
      }else if(time<18){
        condition='发育时间较短，子房和胚珠尚未表现出明显的成熟变化。';
      }

      result.innerHTML=partNotes[selectedPart]
        +'<br>'+condition
        +' 少数植物能够发生单性结实，但本模型不展开该例外。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<partButtons.length;j++){
      partButtons[j].onclick=function(){
        selectedPart=this.getAttribute('data-part');

        if(mode==='development'){
          mode='mapping';
        }

        update();
      };
    }

    for(var k=0;k<stageButtons.length;k++){
      stageButtons[k].onclick=function(){
        var stageIndex=Number(
          this.getAttribute('data-stage')
        );

        var stageTimes=[
          5,
          25,
          50,
          75,
          100
        ];

        timeInput.value=String(
          stageTimes[stageIndex]
        );

        mode='development';
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

    ovuleInput.oninput=update;
    fertilizationInput.oninput=update;
    nutrientInput.oninput=update;
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
