/**
 * lifeScienceLabTemplatesReproductionAnimalLifeCycle.ts
 *
 * 平面生命科学实验室：动物生命周期与变态发育。
 *
 * 教学边界：
 * 1. 完全变态昆虫通常经历卵→幼虫→蛹→成虫；
 * 2. 不完全变态昆虫通常经历卵→若虫→成虫，没有蛹期；
 * 3. 青蛙通常经历受精卵、蝌蚪、长出四肢、幼蛙和成蛙等阶段；
 * 4. 昆虫蜕皮与变态受蜕皮激素、保幼激素等共同调节；
 * 5. 两栖动物变态发育与甲状腺激素等信号及环境条件有关；
 * 6. 温度、激素和发育进度均为相对教学指标，不代表具体物种数据。
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
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
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

function animalLifeCycleStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BFDBFE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DBEAFE,#ECFDF5);border-bottom:1px solid #BFDBFE}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#1D4ED8}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FBFF;border-right:1px solid #BFDBFE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#2563EB;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#3B82F6}'
    + '#' + rootId + ' .al-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#1D4ED8}'
    + '#' + rootId + ' .al-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .al-stages{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .al-button{min-height:29px;padding:3px;border:1px solid #93C5FD;border-radius:8px;background:#fff;color:#1D4ED8;font-size:9.4px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .al-button.active{border-color:#2563EB;background:#DBEAFE;box-shadow:0 3px 9px rgba(37,99,235,.13)}'
    + '#' + rootId + ' .al-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#60A5FA,#2563EB);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .al-toggle.off{background:#64748B}'
    + '#' + rootId + ' .al-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .al-card{padding:6px 3px;border:1px solid #BFDBFE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .al-card b{display:block;font-size:14px;color:#1D4ED8}'
    + '#' + rootId + ' .al-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DBEAFE;color:#1E3A8A;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .al-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--al-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .al-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_ANIMAL_LIFE_CYCLE:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-animal-life-cycle',
    group: '🌸 生殖与个体发育',
    name: '动物生命周期与变态发育',
    emoji: '🦋',
    desc: '比较蝴蝶、蝗虫和青蛙的生命周期，观察完全变态、不完全变态和两栖动物变态发育',
    params: [
      {
        key: 'animalType',
        label: '动物类型',
        type: 'number',
        min: 0,
        max: 2,
        step: 1,
        defaultValue: 0,
        hint: '0=蝴蝶，1=蝗虫，2=青蛙',
      },
      {
        key: 'environmentTemperature',
        label: '环境温度/℃',
        type: 'number',
        min: 10,
        max: 35,
        step: 1,
        defaultValue: 25,
      },
      {
        key: 'nutritionSupply',
        label: '营养供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'developmentTime',
        label: '发育时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
      },
      {
        key: 'hormoneSignal',
        label: '变态信号强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'showLabels',
        label: '显示阶段标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const animalType = num(params, 'animalType', 0)
      const environmentTemperature = num(params, 'environmentTemperature', 25)
      const nutritionSupply = num(params, 'nutritionSupply', 76)
      const developmentTime = num(params, 'developmentTime', 42)
      const hormoneSignal = num(params, 'hormoneSignal', 68)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${animalLifeCycleStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🦋 动物生命周期与变态发育</div>
    <div class="bl-note">发育时间与激素数值均为相对教学指标</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>动物类型</span>
          <span class="bl-value" data-animal-value></span>
        </div>
        <input
          data-animal
          type="range"
          min="0"
          max="2"
          step="1"
          value="${n(animalType)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>环境温度</span>
          <span class="bl-value" data-temperature-value></span>
        </div>
        <input
          data-temperature
          type="range"
          min="10"
          max="35"
          step="1"
          value="${n(environmentTemperature)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>营养供应</span>
          <span class="bl-value" data-nutrition-value></span>
        </div>
        <input
          data-nutrition
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(nutritionSupply)}"
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

      <div class="bl-row">
        <div class="bl-label">
          <span>变态信号强度</span>
          <span class="bl-value" data-hormone-value></span>
        </div>
        <input
          data-hormone
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(hormoneSignal)}"
        >
      </div>

      <div class="al-subtitle">观察方式</div>

      <div class="al-buttons">
        <button
          type="button"
          class="al-button active"
          data-mode="cycle"
        >生命周期</button>

        <button
          type="button"
          class="al-button"
          data-mode="compare"
        >类型比较</button>

        <button
          type="button"
          class="al-button"
          data-mode="change"
        >关键变化</button>
      </div>

      <div class="al-subtitle">快速查看阶段</div>

      <div class="al-stages">
        <button type="button" class="al-button active" data-stage="0"></button>
        <button type="button" class="al-button" data-stage="1"></button>
        <button type="button" class="al-button" data-stage="2"></button>
        <button type="button" class="al-button" data-stage="3"></button>
        <button type="button" class="al-button" data-stage="4"></button>
      </div>

      <button
        type="button"
        class="al-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '阶段标注：显示' : '阶段标注：隐藏'}</button>

      <button
        type="button"
        class="al-toggle"
        data-auto
      >发育推进：运行中</button>

      <div class="al-status">
        <div class="al-card">
          <b data-progress></b>
          <span>相对发育进度</span>
        </div>

        <div class="al-card">
          <b data-stage-name></b>
          <span>当前阶段</span>
        </div>

        <div class="al-card">
          <b data-type-name></b>
          <span>发育类型</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="动物生命周期与变态发育互动示意图"
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
              flood-color="#1E3A8A"
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
          fill="#1D4ED8"
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

        <g transform="translate(526 337)">
          <rect
            width="208"
            height="66"
            rx="15"
            fill="#EFF6FF"
            stroke="#BFDBFE"
            stroke-width="2"
          />

          <text
            x="104"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#1D4ED8"
          >科学边界</text>

          <text
            x="104"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#1E3A8A"
          >速度和激素为相对指标</text>

          <text
            x="104"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#1E3A8A"
          >不同物种差异很大</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#1D4ED8"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var animalInput=root.querySelector('[data-animal]');
    var temperatureInput=root.querySelector('[data-temperature]');
    var nutritionInput=root.querySelector('[data-nutrition]');
    var timeInput=root.querySelector('[data-time]');
    var hormoneInput=root.querySelector('[data-hormone]');

    var animalValue=root.querySelector('[data-animal-value]');
    var temperatureValue=root.querySelector('[data-temperature-value]');
    var nutritionValue=root.querySelector('[data-nutrition-value]');
    var timeValue=root.querySelector('[data-time-value]');
    var hormoneValue=root.querySelector('[data-hormone-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var stageButtons=root.querySelectorAll('[data-stage]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var progressText=root.querySelector('[data-progress]');
    var stageNameText=root.querySelector('[data-stage-name]');
    var typeNameText=root.querySelector('[data-type-name]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');
    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='cycle';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    var animals=[
      {
        name:'蝴蝶',
        type:'完全变态',
        stages:['卵','幼虫','蛹','羽化','成虫'],
        short:['卵','幼虫','蛹','羽化','成虫'],
        icons:['🥚','🐛','🟤','🦋','🦋'],
        optimum:25,
        color:'#EC4899',
        note:'卵→幼虫→蛹→成虫。幼体与成体形态、食性和生活方式差异明显，蛹期发生显著重组。'
      },
      {
        name:'蝗虫',
        type:'不完全变态',
        stages:['卵','若虫Ⅰ','若虫Ⅱ','若虫Ⅲ','成虫'],
        short:['卵','若Ⅰ','若Ⅱ','若Ⅲ','成虫'],
        icons:['🥚','🦗','🦗','🦗','🦗'],
        optimum:28,
        color:'#65A30D',
        note:'卵→若虫→成虫。没有蛹期，若虫形态和生活方式与成虫较相似，通过多次蜕皮逐渐长大。'
      },
      {
        name:'青蛙',
        type:'两栖动物变态发育',
        stages:['受精卵','蝌蚪','后肢出现','幼蛙','成蛙'],
        short:['卵','蝌蚪','长后肢','幼蛙','成蛙'],
        icons:['◉','◉','🐸','🐸','🐸'],
        optimum:24,
        color:'#0EA5E9',
        note:'受精卵→蝌蚪→长出四肢→幼蛙→成蛙。发育中呼吸器官、运动器官和生活环境发生明显变化。'
      }
    ];

    var descriptions=[
      [
        '受精卵内胚胎开始发育。',
        '幼虫以取食和生长为主，多次蜕皮。',
        '蛹期外部活动减少，内部发生显著重组。',
        '成虫从蛹中羽化，翅逐渐展开。',
        '成虫能够交配产卵，生命周期进入下一轮。'
      ],
      [
        '卵内胚胎开始发育。',
        '早期若虫体形较小，没有成熟翅和生殖能力。',
        '若虫经多次蜕皮，身体和翅芽逐渐增大。',
        '晚期若虫更接近成虫，但尚未完全成熟。',
        '成虫翅和生殖器官成熟，可繁殖产生下一代。'
      ],
      [
        '受精卵在水中进行胚胎发育。',
        '蝌蚪主要生活在水中，以尾游泳并用鳃呼吸。',
        '后肢逐渐出现，变态发育进入明显变化阶段。',
        '前肢形成，肺呼吸增强，尾逐渐缩短。',
        '成蛙主要用肺和皮肤呼吸，可繁殖形成新的受精卵。'
      ]
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
        timeInput.value=String(next>100?0:next);
        update();
        schedule();
      },760);
    }

    function resolveStage(progress){
      if(progress<.16)return 0;
      if(progress<.38)return 1;
      if(progress<.61)return 2;
      if(progress<.82)return 3;
      return 4;
    }

    function currentShape(animalIndex,stageIndex){
      if(animalIndex===2 && stageIndex===0){
        var eggs='';

        for(var e=0;e<8;e++){
          var eggAngle=Math.PI*2*e/8;

          eggs+='<circle cx="'
            +(Math.cos(eggAngle)*28)
            +'" cy="'
            +(Math.sin(eggAngle)*22)
            +'" r="10" fill="#E0F2FE" stroke="#0284C7" stroke-width="2"/>';
        }

        return '<ellipse cx="0" cy="0" rx="53" ry="40" fill="#BAE6FD" stroke="#0284C7" stroke-width="3" opacity=".7"/>'
          +eggs;
      }

      if(animalIndex===2 && stageIndex===1){
        return '<ellipse cx="0" cy="0" rx="32" ry="22" fill="#334155"/>'
          +'<path d="M-28 0 Q-76 24 -96 3" fill="none" stroke="#334155" stroke-width="13" stroke-linecap="round"/>'
          +'<circle cx="11" cy="-6" r="4" fill="#F8FAFC"/>';
      }

      if(animalIndex===2 && stageIndex===2){
        return '<ellipse cx="0" cy="0" rx="34" ry="23" fill="#334155"/>'
          +'<path d="M-30 0 Q-69 18 -86 4" fill="none" stroke="#334155" stroke-width="10" stroke-linecap="round"/>'
          +'<path d="M-6 15 L-27 42 M17 14 L38 41" stroke="#16A34A" stroke-width="7" stroke-linecap="round"/>';
      }

      return '<text x="0" y="18" text-anchor="middle" font-size="76">'
        +animals[animalIndex].icons[stageIndex]
        +'</text>';
    }

    function renderCycle(animalIndex,stageIndex,progress){
      var animal=animals[animalIndex];

      var centers=[
        [177,131],
        [382,108],
        [524,219],
        [398,326],
        [181,309]
      ];

      var arrows=[
        [236,129,317,112],
        [444,132,493,180],
        [493,270,445,306],
        [337,326,244,316],
        [151,256,153,187]
      ];

      var html='';

      for(var i=0;i<5;i++){
        var cx=centers[i][0];
        var cy=centers[i][1];
        var active=i===stageIndex;
        var completed=i<stageIndex;

        html+='<circle cx="'+cx+'" cy="'+cy
          +'" r="56" fill="'
          +(active?'#DBEAFE':completed?'#DCFCE7':'#F8FAFC')
          +'" stroke="'
          +(active?animal.color:completed?'#16A34A':'#CBD5E1')
          +'" stroke-width="'+(active?6:3)+'"/>'
          +'<text x="'+cx+'" y="'+(cy+10)
          +'" text-anchor="middle" font-size="35">'
          +animal.icons[i]
          +'</text>'
          +'<text x="'+cx+'" y="'+(cy+47)
          +'" text-anchor="middle" font-size="11" font-weight="900" fill="#334155">'
          +animal.stages[i]
          +'</text>';

        if(active){
          html+='<circle class="al-pulse" cx="'+cx+'" cy="'+cy
            +'" r="65" fill="none" stroke="'+animal.color
            +'" stroke-width="4" stroke-dasharray="8 7"/>';
        }
      }

      for(var a=0;a<arrows.length;a++){
        html+='<path class="al-flow" d="M'
          +arrows[a][0]+' '+arrows[a][1]
          +' Q'
          +((arrows[a][0]+arrows[a][2])/2)
          +' '
          +((arrows[a][1]+arrows[a][3])/2-8)
          +' '
          +arrows[a][2]+' '+arrows[a][3]
          +'" fill="none" stroke="#60A5FA" stroke-width="4" marker-end="url(#${rootId}-arrow-blue)"/>';
      }

      return html
        +'<g transform="translate(351 216)" filter="url(#${rootId}-shadow)">'
        +'<circle r="82" fill="#FFFFFF" stroke="'+animal.color+'" stroke-width="5"/>'
        +currentShape(animalIndex,stageIndex)
        +'<text x="0" y="68" text-anchor="middle" font-size="14" font-weight="900" fill="'+animal.color+'">'
        +animal.name+' · '+animal.stages[stageIndex]
        +'</text>'
        +'</g>'
        +'<g transform="translate(562 91)">'
        +'<rect width="170" height="85" rx="17" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<text x="85" y="24" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">相对发育进度</text>'
        +'<rect x="17" y="40" width="136" height="14" rx="7" fill="#E2E8F0"/>'
        +'<rect x="17" y="40" width="'+(136*progress)
        +'" height="14" rx="7" fill="'+animal.color+'"/>'
        +'<text x="85" y="73" text-anchor="middle" font-size="11" font-weight="900" fill="#475569">'
        +(progress*100).toFixed(0)+'%'
        +'</text>'
        +'</g>';
    }

    function renderCompare(){
      var cards=[
        [
          '完全变态',
          '卵→幼虫→蛹→成虫',
          '🦋',
          ['有明显蛹期','幼体与成体差异大','蛹期发生显著重组'],
          '#BE185D',
          '#FDF2F8',
          '#F9A8D4'
        ],
        [
          '不完全变态',
          '卵→若虫→成虫',
          '🦗',
          ['没有蛹期','若虫与成虫较相似','多次蜕皮逐渐成熟'],
          '#4D7C0F',
          '#F7FEE7',
          '#BEF264'
        ],
        [
          '两栖动物变态发育',
          '卵→蝌蚪→幼蛙→成蛙',
          '🐸',
          ['四肢逐渐形成','鳃呼吸转向肺呼吸','生活环境发生变化'],
          '#0369A1',
          '#F0F9FF',
          '#7DD3FC'
        ]
      ];

      var html='';

      for(var i=0;i<3;i++){
        var x=28+i*243;
        var card=cards[i];

        html+='<g transform="translate('+x+' 89)">'
          +'<rect width="218" height="252" rx="20" fill="'+card[5]
          +'" stroke="'+card[6]+'" stroke-width="3"/>'
          +'<text x="109" y="31" text-anchor="middle" font-size="16" font-weight="900" fill="'+card[4]+'">'
          +card[0]
          +'</text>'
          +'<text x="109" y="58" text-anchor="middle" font-size="12" font-weight="900" fill="'+card[4]+'">'
          +card[1]
          +'</text>'
          +'<text x="109" y="105" text-anchor="middle" font-size="43">'
          +card[2]
          +'</text>'
          +'<text x="20" y="145" font-size="12" font-weight="800" fill="#475569">• '
          +card[3][0]
          +'</text>'
          +'<text x="20" y="174" font-size="12" font-weight="800" fill="#475569">• '
          +card[3][1]
          +'</text>'
          +'<text x="20" y="203" font-size="12" font-weight="800" fill="#475569">• '
          +card[3][2]
          +'</text>'
          +'</g>';
      }

      return html;
    }

    function renderChange(animalIndex,stageIndex){
      var animal=animals[animalIndex];
      var middle=animalIndex===0?2:animalIndex===1?3:2;

      var notes=animalIndex===0
        ?[
          '幼虫以取食和生长为主',
          '蛹期内部发生显著重组',
          '成虫具翅并可繁殖'
        ]
        :animalIndex===1
          ?[
            '若虫与成虫形态较相似',
            '多次蜕皮，翅芽逐渐增大',
            '成虫翅和生殖器官成熟'
          ]
          :[
            '蝌蚪主要用鳃呼吸',
            '四肢形成，肺呼吸增强',
            '成蛙可水陆活动并繁殖'
          ];

      var stageSet=[1,middle,4];
      var html='';

      for(var i=0;i<3;i++){
        var x=50+i*240;

        html+='<g transform="translate('+x+' 104)">'
          +'<rect width="190" height="214" rx="20" fill="'
          +(i===1?'#EFF6FF':i===2?'#ECFDF5':'#F8FAFC')
          +'" stroke="'
          +(i===1?'#93C5FD':i===2?'#86EFAC':'#CBD5E1')
          +'" stroke-width="3"/>'
          +'<text x="95" y="28" text-anchor="middle" font-size="15" font-weight="900" fill="#334155">'
          +(i===0?'早期幼体':i===1?'变态关键阶段':'成体')
          +'</text>'
          +'<text x="95" y="118" text-anchor="middle" font-size="68">'
          +animal.icons[stageSet[i]]
          +'</text>'
          +'<text x="95" y="188" text-anchor="middle" font-size="11" font-weight="900" fill="#475569">'
          +notes[i]
          +'</text>'
          +'</g>';

        if(i<2){
          html+='<path class="al-flow" d="M'
            +(x+196)
            +' 210 H'
            +(x+230)
            +'" fill="none" stroke="'
            +(i===0?'#2563EB':'#16A34A')
            +'" stroke-width="6" marker-end="url(#${rootId}-arrow-'
            +(i===0?'blue':'green')
            +')"/>';
        }
      }

      return html
        +'<text x="380" y="357" text-anchor="middle" font-size="13" font-weight="900" fill="'
        +animal.color
        +'">当前阶段：'
        +animal.stages[stageIndex]
        +'</text>';
    }

    function renderLabels(animalIndex,stageIndex){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      var animal=animals[animalIndex];

      labels.innerHTML=
        '<path d="M376 117 L476 91" stroke="'+animal.color+'" stroke-width="2.5"/>'
        +'<text x="486" y="95" font-size="13" font-weight="900" fill="'+animal.color+'">'
        +'当前：'+animal.stages[stageIndex]
        +'</text>'
        +'<path d="M373 288 L477 313" stroke="#2563EB" stroke-width="2.5"/>'
        +'<text x="486" y="319" font-size="13" font-weight="900" fill="#1D4ED8">'
        +'成体繁殖后进入下一轮'
        +'</text>';
    }

    function update(){
      var animalIndex=clamp(
        Math.round(Number(animalInput.value)),
        0,
        2
      );

      var temperature=Number(temperatureInput.value);
      var nutrition=Number(nutritionInput.value);
      var time=Number(timeInput.value);
      var hormone=Number(hormoneInput.value);
      var animal=animals[animalIndex];

      animalValue.textContent=animal.name;
      temperatureValue.textContent=temperature.toFixed(0)+'℃';
      nutritionValue.textContent=nutrition.toFixed(0)+'%';
      timeValue.textContent=time.toFixed(0)+'%';
      hormoneValue.textContent=hormone.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      for(var j=0;j<stageButtons.length;j++){
        stageButtons[j].textContent=animal.short[j];
      }

      labelToggle.textContent=showLabels
        ?'阶段标注：显示'
        :'阶段标注：隐藏';

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

      var temperatureFactor=Math.exp(
        -Math.pow(
          (temperature-animal.optimum)/8.5,
          2
        )
      );

      var nutritionFactor=.18+.82*nutrition/100;
      var hormoneFactor=.20+.80*hormone/100;

      var progress=time/100
        *(.24+.76*Math.sqrt(
          temperatureFactor*nutritionFactor
        ))
        *(.35+.65*hormoneFactor);

      progress=clamp(
        progress,
        0,
        1
      );

      var stageIndex=resolveStage(progress);

      for(var k=0;k<stageButtons.length;k++){
        stageButtons[k].classList.toggle(
          'active',
          Number(
            stageButtons[k].getAttribute('data-stage')
          )===stageIndex
        );
      }

      progressText.textContent=(progress*100).toFixed(0)+'%';
      stageNameText.textContent=animal.stages[stageIndex];
      typeNameText.textContent=animal.type;

      root.style.setProperty(
        '--al-speed',
        clamp(
          2.5-progress*1.7,
          .65,
          2.4
        ).toFixed(2)+'s'
      );

      title.textContent=mode==='cycle'
        ?animal.name+'的生命周期'
        :mode==='compare'
          ?'三种变态发育类型比较'
          :animal.name+'发育中的关键变化';

      summary.textContent=mode==='compare'
        ?'比较是否有蛹期、幼体与成体差异以及生活环境变化。'
        :'当前阶段：'
          +animal.stages[stageIndex]
          +'；发育类型：'
          +animal.type
          +'。';

      stageNote.textContent=mode==='compare'
        ?'变态发育是幼体到成体过程中形态和功能发生显著改变的发育方式。'
        :descriptions[animalIndex][stageIndex];

      if(mode==='cycle'){
        dynamic.innerHTML=renderCycle(
          animalIndex,
          stageIndex,
          progress
        );

        renderLabels(
          animalIndex,
          stageIndex
        );
      }else if(mode==='compare'){
        dynamic.innerHTML=renderCompare();
        labels.innerHTML='';
      }else{
        dynamic.innerHTML=renderChange(
          animalIndex,
          stageIndex
        );

        labels.innerHTML='';
      }

      var condition=
        '当前温度、营养和变态信号共同支持相对正常的发育进程。';

      if(temperatureFactor<.25){
        condition=
          '当前温度明显偏离教学模型中的适宜范围，发育速度显著下降。';
      }else if(nutrition<20){
        condition=
          '营养供应不足，幼体生长和后续变态发育受到明显限制。';
      }else if(hormone<18){
        condition=
          '变态信号较弱，阶段转换速度下降；真实生物中的激素调节更加复杂。';
      }else if(time<12){
        condition=
          '发育时间较短，个体仍处于生命周期早期。';
      }

      result.innerHTML=animal.note
        +'<br>'
        +condition
        +' 昆虫和两栖动物的变态均由多种激素、组织状态和环境条件共同调节。';
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

        var stageTimes=[
          5,
          25,
          48,
          72,
          96
        ];

        timeInput.value=String(
          stageTimes[stageIndex]
        );

        automatic=false;
        mode='cycle';

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

    animalInput.oninput=update;
    temperatureInput.oninput=update;
    nutritionInput.oninput=update;
    timeInput.oninput=update;
    hormoneInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
