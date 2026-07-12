/**
 * lifeScienceLabTemplatesRegulationTemperature.ts
 *
 * 平面生命科学实验室：体温调节。
 *
 * 教学边界：
 * 1. 温度感受器感受体内外温度变化；
 * 2. 下丘脑体温调节中枢对信息进行整合；
 * 3. 高温时皮肤血管舒张、出汗增强，有利于散热；
 * 4. 低温时皮肤血管收缩、骨骼肌战栗，有利于减少散热和增加产热；
 * 5. 体温调节属于负反馈调节，本模型不用于健康或疾病判断。
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

function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

function temperatureStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .tp-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#FFF7ED);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .tp-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .tp-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .tp-body{height:calc(100% - 46px);display:grid;grid-template-columns:240px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .tp-controls{padding:13px;overflow:auto;background:#FAFCFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .tp-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .tp-row{margin-bottom:10px}'
    + '#' + rootId + ' .tp-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .tp-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0284C7}'
    + '#' + rootId + ' .tp-subtitle{margin:7px 0;font-size:12px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .tp-buttons{display:grid;grid-template-columns:1fr 1fr;gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .tp-button{height:31px;padding:0 4px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .tp-button.active{border-color:#0284C7;background:#E0F2FE;box-shadow:0 3px 9px rgba(2,132,199,.13)}'
    + '#' + rootId + ' .tp-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .tp-card{padding:7px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .tp-card b{display:block;font-size:16px;color:#0369A1}'
    + '#' + rootId + ' .tp-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .tp-result{padding:9px 10px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .tp-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.5s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_TEMPERATURE:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-body-temperature-regulation',
    group: '🧠 稳态与调节',
    name: '体温调节',
    emoji: '🌡️',
    desc: '调节环境温度、活动产热、出汗效率和战栗能力，观察体温负反馈调节',
    params: [
      {
        key: 'ambientTemperature',
        label: '环境温度/℃',
        type: 'number',
        min: 0,
        max: 42,
        step: 1,
        defaultValue: 25,
      },
      {
        key: 'activityHeat',
        label: '活动产热水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 30,
      },
      {
        key: 'sweatEfficiency',
        label: '出汗散热效率',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'shiveringCapacity',
        label: '战栗产热能力',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 75,
      },
    ],

    buildHTML: (params, rootId) => {
      const ambientTemperature = num(params, 'ambientTemperature', 25)
      const activityHeat = num(params, 'activityHeat', 30)
      const sweatEfficiency = num(params, 'sweatEfficiency', 78)
      const shiveringCapacity = num(params, 'shiveringCapacity', 75)

      return `
<div id="${rootId}">
${temperatureStyle(rootId)}
  <div class="tp-head">
    <div class="tp-title">🌡️ 体温稳态与负反馈调节</div>
    <div class="tp-note">教学示意，不用于健康或疾病判断</div>
  </div>

  <div class="tp-body">
    <div class="tp-controls">
      <div class="tp-row">
        <div class="tp-label">
          <span>环境温度</span>
          <span class="tp-value" data-ambient-value></span>
        </div>
        <input data-ambient type="range" min="0" max="42" step="1" value="${n(ambientTemperature)}">
      </div>

      <div class="tp-row">
        <div class="tp-label">
          <span>活动产热水平</span>
          <span class="tp-value" data-activity-value></span>
        </div>
        <input data-activity type="range" min="0" max="100" step="1" value="${n(activityHeat)}">
      </div>

      <div class="tp-row">
        <div class="tp-label">
          <span>出汗散热效率</span>
          <span class="tp-value" data-sweat-value></span>
        </div>
        <input data-sweat type="range" min="20" max="100" step="1" value="${n(sweatEfficiency)}">
      </div>

      <div class="tp-row">
        <div class="tp-label">
          <span>战栗产热能力</span>
          <span class="tp-value" data-shivering-value></span>
        </div>
        <input data-shivering type="range" min="20" max="100" step="1" value="${n(shiveringCapacity)}">
      </div>

      <div class="tp-subtitle">选择调节环节</div>

      <div class="tp-buttons">
        <button type="button" class="tp-button active" data-stage="receptor">温度感受</button>
        <button type="button" class="tp-button" data-stage="center">调节中枢</button>
        <button type="button" class="tp-button" data-stage="heatLoss">散热反应</button>
        <button type="button" class="tp-button" data-stage="heatProduction">产热反应</button>
      </div>

      <div class="tp-status">
        <div class="tp-card">
          <b data-core-temperature></b>
          <span>核心体温教学示意</span>
        </div>

        <div class="tp-card">
          <b data-regulation-state></b>
          <span>当前调节方向</span>
        </div>
      </div>

      <div class="tp-result" data-result></div>
    </div>

    <div class="tp-stage">
      <svg viewBox="0 0 680 414" aria-label="体温调节互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#075985" flood-opacity=".13"/>
          </filter>

          <linearGradient id="${rootId}-thermometer" x1="0" y1="1" x2="0" y2="0">
            <stop offset="0%" stop-color="#2563EB"/>
            <stop offset="55%" stop-color="#F59E0B"/>
            <stop offset="100%" stop-color="#EF4444"/>
          </linearGradient>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#075985"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <g filter="url(#${rootId}-shadow)">
          <circle cx="122" cy="202" r="68" fill="#DBEAFE" stroke="#2563EB" stroke-width="5"/>
          <circle cx="122" cy="202" r="27" fill="#93C5FD" stroke="#1D4ED8" stroke-width="4"/>
          <path d="M83 166 Q122 135 161 166 M83 238 Q122 269 161 238"
            fill="none" stroke="#60A5FA" stroke-width="5" stroke-linecap="round"/>
          <text x="122" y="300" text-anchor="middle" font-size="15" font-weight="900" fill="#1E40AF">温度感受器</text>

          <ellipse cx="340" cy="188" rx="90" ry="66" fill="#EDE9FE" stroke="#7C3AED" stroke-width="5"/>
          <path d="M286 188 Q340 128 394 188 Q340 248 286 188Z" fill="#C4B5FD"/>
          <circle cx="340" cy="188" r="24" fill="#8B5CF6" opacity=".72"/>
          <text x="340" y="282" text-anchor="middle" font-size="15" font-weight="900" fill="#6D28D9">下丘脑调节中枢</text>

          <rect x="500" y="113" width="124" height="87" rx="30" fill="#FEE2E2" stroke="#E11D48" stroke-width="5"/>
          <text x="562" y="150" text-anchor="middle" font-size="16" font-weight="900" fill="#BE123C">散热反应</text>
          <text x="562" y="176" text-anchor="middle" font-size="12" font-weight="800" fill="#475569">出汗、血管舒张</text>

          <rect x="500" y="235" width="124" height="87" rx="30" fill="#FEF3C7" stroke="#D97706" stroke-width="5"/>
          <text x="562" y="272" text-anchor="middle" font-size="16" font-weight="900" fill="#92400E">产热反应</text>
          <text x="562" y="298" text-anchor="middle" font-size="12" font-weight="800" fill="#475569">战栗、代谢产热</text>
        </g>

        <path class="tp-flow" d="M190 202 H242"
          fill="none" stroke="#0284C7" stroke-width="5"
          marker-end="url(#${rootId}-arrow)"/>

        <path class="tp-flow" d="M430 166 C462 146 474 146 493 151"
          fill="none" stroke="#E11D48" stroke-width="5"
          marker-end="url(#${rootId}-arrow)"/>

        <path class="tp-flow" d="M430 219 C461 247 478 265 493 276"
          fill="none" stroke="#D97706" stroke-width="5"
          marker-end="url(#${rootId}-arrow)"/>

        <g data-highlight></g>
        <g data-effects></g>

        <g transform="translate(32 345)">
          <rect x="0" y="0" width="390" height="17" rx="8" fill="#E2E8F0"/>
          <rect data-temperature-bar x="0" y="0" width="195" height="17" rx="8" fill="url(#${rootId}-thermometer)"/>
          <line x1="195" y1="-7" x2="195" y2="25" stroke="#10B981" stroke-width="4"/>
          <text x="168" y="43" font-size="11" font-weight="900" fill="#047857">稳态参考</text>
        </g>

        <text x="445" y="360" data-balance-note font-size="14" font-weight="900" fill="#075985"></text>
        <text x="445" y="386" data-stage-note font-size="13" font-weight="800" fill="#475569"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var ambient=root.querySelector('[data-ambient]');
    var activity=root.querySelector('[data-activity]');
    var sweat=root.querySelector('[data-sweat]');
    var shivering=root.querySelector('[data-shivering]');

    var ambientValue=root.querySelector('[data-ambient-value]');
    var activityValue=root.querySelector('[data-activity-value]');
    var sweatValue=root.querySelector('[data-sweat-value]');
    var shiveringValue=root.querySelector('[data-shivering-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var coreTemperature=root.querySelector('[data-core-temperature]');
    var regulationState=root.querySelector('[data-regulation-state]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var highlight=root.querySelector('[data-highlight]');
    var effects=root.querySelector('[data-effects]');
    var temperatureBar=root.querySelector('[data-temperature-bar]');
    var balanceNote=root.querySelector('[data-balance-note]');
    var stageNote=root.querySelector('[data-stage-note]');

    var stage='receptor';

    var information={
      receptor:{
        title:'温度感受器感受变化',
        summary:'皮肤和体内温度感受器把温度变化转换为神经信号',
        note:'体温调节首先需要感受器检测体内外温度变化。'
      },
      center:{
        title:'下丘脑整合调节信息',
        summary:'体温调节中枢比较温度变化并发出相应调节指令',
        note:'下丘脑是体温调节的重要中枢，可协调神经和体液调节。'
      },
      heatLoss:{
        title:'高温条件下增强散热',
        summary:'皮肤血管舒张、皮肤血流增加并促进出汗散热',
        note:'出汗后水分蒸发能够带走热量，皮肤血管舒张也有利于散热。'
      },
      heatProduction:{
        title:'低温条件下增加产热',
        summary:'皮肤血管收缩减少散热，骨骼肌战栗增加产热',
        note:'低温时减少皮肤散热并增加产热，有助于维持核心体温相对稳定。'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function update(){
      var ambientLevel=Number(ambient.value);
      var activityLevel=Number(activity.value);
      var sweatLevel=Number(sweat.value);
      var shiveringLevel=Number(shivering.value);

      ambientValue.textContent=ambientLevel.toFixed(0)+'℃';
      activityValue.textContent=activityLevel.toFixed(0)+'%';
      sweatValue.textContent=sweatLevel.toFixed(0)+'%';
      shiveringValue.textContent=shiveringLevel.toFixed(0)+'%';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var environmentalLoad=(ambientLevel-25)*.055;
      var activityLoad=activityLevel*.012;
      var hotSignal=Math.max(0,environmentalLoad+activityLoad);
      var coldSignal=Math.max(0,-environmentalLoad-activityLoad*.25);

      var heatLoss=hotSignal*sweatLevel/100*.72;
      var heatGain=coldSignal*shiveringLevel/100*.68;

      var core=37
        +environmentalLoad
        +activityLoad
        -heatLoss
        +heatGain;

      core=clamp(core,35.2,39.6);

      coreTemperature.textContent=core.toFixed(1)+'℃';

      var direction='接近稳态';

      if(core>37.4){
        direction='增强散热';
      }else if(core<36.6){
        direction='减少散热、增加产热';
      }

      regulationState.textContent=direction;

      temperatureBar.setAttribute(
        'width',
        String(390*clamp((core-35.2)/(39.6-35.2),0,1))
      );

      var info=information[stage];

      title.textContent=info.title;
      summary.textContent=info.summary;

      var highlightHTML='';
      var effectHTML='';

      if(stage==='receptor'){
        highlightHTML='<circle cx="122" cy="202" r="82" fill="none" stroke="#2563EB" stroke-width="5"/>';
        stageNote.textContent='检测体内外温度变化';
      }else if(stage==='center'){
        highlightHTML='<ellipse cx="340" cy="188" rx="105" ry="80" fill="none" stroke="#7C3AED" stroke-width="5"/>';
        stageNote.textContent='整合信息并发出调节指令';
      }else if(stage==='heatLoss'){
        highlightHTML='<rect x="488" y="101" width="148" height="111" rx="39" fill="none" stroke="#E11D48" stroke-width="5"/>';

        var sweatCount=Math.floor(2+sweatLevel/14);

        for(var s=0;s<sweatCount;s++){
          var x=505+(s%6)*22;
          var y=82-Math.floor(s/6)*22;

          effectHTML+='<path d="M'+x+' '+y
            +' C'+(x-7)+' '+(y+12)+' '+x+' '+(y+21)
            +' '+x+' '+(y+21)
            +' C'+x+' '+(y+21)+' '+(x+7)+' '+(y+12)+' '+x+' '+y
            +'Z" fill="#38BDF8" opacity=".8"/>';
        }

        stageNote.textContent='出汗和皮肤血管舒张';
      }else{
        highlightHTML='<rect x="488" y="223" width="148" height="111" rx="39" fill="none" stroke="#D97706" stroke-width="5"/>';

        var lines=Math.floor(2+shiveringLevel/18);

        for(var q=0;q<lines;q++){
          var y=335+q*9;

          effectHTML+='<path d="M500 '+y
            +' l8 -6 l8 12 l8 -12 l8 12 l8 -12 l8 6'
            +'" fill="none" stroke="#D97706" stroke-width="3"/>';
        }

        stageNote.textContent='血管收缩和骨骼肌战栗';
      }

      highlight.innerHTML=highlightHTML;
      effects.innerHTML=effectHTML;

      var balance='产热与散热相对平衡';

      if(core>37.4){
        balance='产热暂时大于散热';
      }else if(core<36.6){
        balance='散热暂时大于产热';
      }

      balanceNote.textContent=balance;

      var condition='当前环境、活动和调节能力能够使核心体温接近稳态参考值。';

      if(ambientLevel>35 && sweatLevel<40){
        condition='环境温度较高而出汗散热效率较低，核心体温可能偏高。';
      }else if(activityLevel>75 && sweatLevel<50){
        condition='活动产热较强而散热能力不足，体内热量容易积累。';
      }else if(ambientLevel<8 && shiveringLevel<40){
        condition='环境温度较低而战栗产热能力较弱，核心体温可能偏低。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' 体温偏离稳态时，相应调节反应会减小原来的偏差，体现负反馈调节。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        stage=this.getAttribute('data-stage');
        update();
      };
    }

    ambient.oninput=update;
    activity.oninput=update;
    sweat.oninput=update;
    shivering.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
