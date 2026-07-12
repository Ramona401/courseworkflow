/**
 * lifeScienceLabTemplatesRegulationBloodGlucose.ts
 *
 * 平面生命科学实验室：血糖调节。
 *
 * 教学边界：
 * 1. 血糖升高时，胰岛β细胞分泌胰岛素增多；
 * 2. 胰岛素促进细胞摄取和利用葡萄糖，并促进糖原合成；
 * 3. 血糖降低时，胰岛α细胞分泌胰高血糖素增多；
 * 4. 胰高血糖素促进肝糖原分解等过程，使血糖回升；
 * 5. 模型使用相对血糖指数，不代表临床检测值。
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

function bloodGlucoseStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FDE68A;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bg-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FEF3C7,#FFF7ED);border-bottom:1px solid #FDE68A}'
    + '#' + rootId + ' .bg-title{font-size:15px;font-weight:800;color:#92400E}'
    + '#' + rootId + ' .bg-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bg-body{height:calc(100% - 46px);display:grid;grid-template-columns:240px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bg-controls{padding:13px;overflow:auto;background:#FFFCF5;border-right:1px solid #FDE68A}'
    + '#' + rootId + ' .bg-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bg-row{margin-bottom:11px}'
    + '#' + rootId + ' .bg-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bg-value{font-weight:800;color:#D97706;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#F59E0B}'
    + '#' + rootId + ' .bg-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#92400E}'
    + '#' + rootId + ' .bg-buttons{display:grid;grid-template-columns:1fr 1fr;gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bg-button{height:31px;padding:0 4px;border:1px solid #FCD34D;border-radius:8px;background:#fff;color:#92400E;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bg-button.active{border-color:#F59E0B;background:#FEF3C7;box-shadow:0 3px 9px rgba(245,158,11,.14)}'
    + '#' + rootId + ' .bg-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bg-card{padding:7px;border:1px solid #FDE68A;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bg-card b{display:block;font-size:16px;color:#B45309}'
    + '#' + rootId + ' .bg-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bg-result{padding:9px 10px;border-radius:10px;background:#FEF3C7;color:#78350F;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .bg-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.5s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_BLOOD_GLUCOSE:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-blood-glucose-regulation',
    group: '🧠 稳态与调节',
    name: '血糖调节',
    emoji: '🍬',
    desc: '调节进食、运动和激素敏感性，观察胰岛素与胰高血糖素的负反馈调节',
    params: [
      {
        key: 'foodIntake',
        label: '进食影响',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'exerciseLevel',
        label: '运动强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 28,
      },
      {
        key: 'insulinSensitivity',
        label: '胰岛素敏感性',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'glucagonResponse',
        label: '胰高血糖素反应',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
    ],

    buildHTML: (params, rootId) => {
      const foodIntake = num(params, 'foodIntake', 68)
      const exerciseLevel = num(params, 'exerciseLevel', 28)
      const insulinSensitivity = num(params, 'insulinSensitivity', 82)
      const glucagonResponse = num(params, 'glucagonResponse', 76)

      return `
<div id="${rootId}">
${bloodGlucoseStyle(rootId)}
  <div class="bg-head">
    <div class="bg-title">🍬 血糖稳态与激素调节</div>
    <div class="bg-note">相对血糖指数，不代表临床检测值</div>
  </div>

  <div class="bg-body">
    <div class="bg-controls">
      <div class="bg-row">
        <div class="bg-label">
          <span>进食影响</span>
          <span class="bg-value" data-food-value></span>
        </div>
        <input data-food type="range" min="0" max="100" step="1" value="${n(foodIntake)}">
      </div>

      <div class="bg-row">
        <div class="bg-label">
          <span>运动强度</span>
          <span class="bg-value" data-exercise-value></span>
        </div>
        <input data-exercise type="range" min="0" max="100" step="1" value="${n(exerciseLevel)}">
      </div>

      <div class="bg-row">
        <div class="bg-label">
          <span>胰岛素敏感性</span>
          <span class="bg-value" data-insulin-value></span>
        </div>
        <input data-insulin type="range" min="20" max="100" step="1" value="${n(insulinSensitivity)}">
      </div>

      <div class="bg-row">
        <div class="bg-label">
          <span>胰高血糖素反应</span>
          <span class="bg-value" data-glucagon-value></span>
        </div>
        <input data-glucagon type="range" min="20" max="100" step="1" value="${n(glucagonResponse)}">
      </div>

      <div class="bg-subtitle">选择调节情境</div>

      <div class="bg-buttons">
        <button type="button" class="bg-button active" data-stage="meal">进食后</button>
        <button type="button" class="bg-button" data-stage="insulin">胰岛素作用</button>
        <button type="button" class="bg-button" data-stage="fasting">空腹时</button>
        <button type="button" class="bg-button" data-stage="glucagon">胰高血糖素作用</button>
      </div>

      <div class="bg-status">
        <div class="bg-card">
          <b data-glucose-index></b>
          <span>相对血糖指数</span>
        </div>

        <div class="bg-card">
          <b data-feedback-state></b>
          <span>反馈状态</span>
        </div>
      </div>

      <div class="bg-result" data-result></div>
    </div>

    <div class="bg-stage">
      <svg viewBox="0 0 680 414" aria-label="血糖调节互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#D97706"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#92400E" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#92400E"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <g filter="url(#${rootId}-shadow)">
          <path
            d="M74 147 C108 122 153 130 175 163 C197 196 173 231 136 237 C96 243 62 215 66 179 C67 166 69 156 74 147Z"
            fill="#FECACA"
            stroke="#DC2626"
            stroke-width="5"
          />

          <text x="121" y="188" text-anchor="middle" font-size="16" font-weight="900" fill="#991B1B">胰腺</text>

          <circle cx="100" cy="207" r="10" fill="#2563EB"/>
          <circle cx="143" cy="205" r="10" fill="#10B981"/>

          <text x="75" y="265" font-size="13" font-weight="900" fill="#1D4ED8">β细胞</text>
          <text x="135" y="265" font-size="13" font-weight="900" fill="#047857">α细胞</text>

          <path
            d="M270 150 C310 113 383 117 414 159 C446 202 411 249 360 255 C311 260 267 224 265 184 C264 171 265 160 270 150Z"
            fill="#FDE68A"
            stroke="#B45309"
            stroke-width="5"
          />

          <text x="342" y="193" text-anchor="middle" font-size="20" font-weight="900" fill="#92400E">肝脏</text>

          <rect x="500" y="132" width="120" height="128" rx="34" fill="#DBEAFE" stroke="#2563EB" stroke-width="5"/>
          <text x="560" y="181" text-anchor="middle" font-size="18" font-weight="900" fill="#1E40AF">组织细胞</text>
          <text x="560" y="209" text-anchor="middle" font-size="13" font-weight="800" fill="#475569">摄取和利用</text>
          <text x="560" y="230" text-anchor="middle" font-size="13" font-weight="800" fill="#475569">葡萄糖</text>
        </g>

        <g data-flows></g>
        <g data-glucose></g>

        <g transform="translate(34 318)">
          <text x="0" y="0" font-size="13" font-weight="800" fill="#475569">相对血糖指数</text>
          <rect x="112" y="-13" width="330" height="18" rx="9" fill="#E2E8F0"/>
          <rect data-glucose-bar x="112" y="-13" width="0" height="18" rx="9" fill="#F59E0B"/>
          <line x1="277" y1="-19" x2="277" y2="12" stroke="#10B981" stroke-width="4"/>
          <text x="258" y="31" font-size="11" font-weight="900" fill="#047857">稳态参考</text>
          <text x="456" y="0" data-bar-value font-size="14" font-weight="900" fill="#B45309"></text>
        </g>

        <g transform="translate(28 376)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">胰岛素</text>
        </g>

        <g transform="translate(158 376)">
          <circle cx="7" cy="7" r="7" fill="#10B981"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">胰高血糖素</text>
        </g>

        <text x="360" y="388" data-stage-note font-size="14" font-weight="900" fill="#92400E"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var food=root.querySelector('[data-food]');
    var exercise=root.querySelector('[data-exercise]');
    var insulin=root.querySelector('[data-insulin]');
    var glucagon=root.querySelector('[data-glucagon]');

    var foodValue=root.querySelector('[data-food-value]');
    var exerciseValue=root.querySelector('[data-exercise-value]');
    var insulinValue=root.querySelector('[data-insulin-value]');
    var glucagonValue=root.querySelector('[data-glucagon-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var glucoseIndex=root.querySelector('[data-glucose-index]');
    var feedbackState=root.querySelector('[data-feedback-state]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var flows=root.querySelector('[data-flows]');
    var glucose=root.querySelector('[data-glucose]');
    var glucoseBar=root.querySelector('[data-glucose-bar]');
    var barValue=root.querySelector('[data-bar-value]');
    var stageNote=root.querySelector('[data-stage-note]');

    var stage='meal';

    var information={
      meal:{
        title:'进食后：血糖升高',
        summary:'食物中的糖类经消化吸收，使血液中葡萄糖相对增加',
        note:'血糖升高可刺激胰岛β细胞分泌胰岛素增多。'
      },
      insulin:{
        title:'胰岛素降低血糖',
        summary:'促进组织细胞摄取和利用葡萄糖，并促进肝糖原合成',
        note:'胰岛素通过多种作用使升高的血糖逐渐回到相对稳定范围。'
      },
      fasting:{
        title:'空腹或运动后：血糖降低',
        summary:'未及时补充葡萄糖，组织细胞仍不断消耗血液中的葡萄糖',
        note:'血糖降低可刺激胰岛α细胞分泌胰高血糖素增多。'
      },
      glucagon:{
        title:'胰高血糖素升高血糖',
        summary:'促进肝糖原分解等过程，使葡萄糖释放进入血液',
        note:'胰高血糖素的作用有助于使偏低的血糖逐渐回升。'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function glucoseParticles(count,level){
      var html='';

      for(var i=0;i<count;i++){
        var x=210+(i%9)*46;
        var y=92+Math.floor(i/9)*28;

        html+='<circle cx="'+x+'" cy="'+y+'" r="'
          +(4+i%3)+'" fill="#F59E0B" opacity="'
          +(.35+.65*level/100)+'"/>';

        html+='<text x="'+(x+7)+'" y="'+(y+4)
          +'" font-size="9" font-weight="900" fill="#92400E">G</text>';
      }

      return html;
    }

    function update(){
      var foodLevel=Number(food.value);
      var exerciseLevel=Number(exercise.value);
      var insulinLevel=Number(insulin.value);
      var glucagonLevel=Number(glucagon.value);

      foodValue.textContent=foodLevel.toFixed(0)+'%';
      exerciseValue.textContent=exerciseLevel.toFixed(0)+'%';
      insulinValue.textContent=insulinLevel.toFixed(0)+'%';
      glucagonValue.textContent=glucagonLevel.toFixed(0)+'%';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var rise=foodLevel*.62;
      var use=exerciseLevel*.34;
      var insulinEffect=insulinLevel*.42;
      var glucagonEffect=glucagonLevel*.36;

      var index=100;

      if(stage==='meal'){
        index=100+rise-use*.25;
      }else if(stage==='insulin'){
        index=100+rise-insulinEffect-use*.22;
      }else if(stage==='fasting'){
        index=100-use*.55-foodLevel*.08;
      }else{
        index=100-use*.45+glucagonEffect-foodLevel*.05;
      }

      index=clamp(index,45,165);

      glucoseIndex.textContent=index.toFixed(0);
      barValue.textContent=index.toFixed(0);

      glucoseBar.setAttribute(
        'width',
        String(330*clamp(index/165,0,1))
      );

      glucoseBar.setAttribute(
        'fill',
        index>120
          ?'#EF4444'
          :index<80
            ?'#3B82F6'
            :'#F59E0B'
      );

      var particleCount=Math.floor(
        clamp(index/8,6,20)
      );

      glucose.innerHTML=glucoseParticles(
        particleCount,
        index
      );

      var flowHTML='';

      if(stage==='meal'){
        flowHTML+='<path class="bg-flow" d="M70 104 C120 75 177 82 218 105'
          +'" fill="none" stroke="#F59E0B" stroke-width="5'
          +'" marker-end="url(#${rootId}-arrow)"/>';

        stageNote.textContent='血糖升高 → β细胞兴奋';
      }else if(stage==='insulin'){
        flowHTML+='<path class="bg-flow" d="M122 208 C205 274 281 270 329 232'
          +'" fill="none" stroke="#2563EB" stroke-width="6'
          +'" marker-end="url(#${rootId}-arrow)"/>';

        flowHTML+='<path class="bg-flow" d="M162 207 C300 97 445 98 500 161'
          +'" fill="none" stroke="#2563EB" stroke-width="5'
          +'" marker-end="url(#${rootId}-arrow)"/>';

        stageNote.textContent='促进摄取利用与糖原合成';
      }else if(stage==='fasting'){
        flowHTML+='<path class="bg-flow" d="M610 283 C540 312 474 310 424 277'
          +'" fill="none" stroke="#64748B" stroke-width="5'
          +'" marker-end="url(#${rootId}-arrow)"/>';

        stageNote.textContent='血糖降低 → α细胞兴奋';
      }else{
        flowHTML+='<path class="bg-flow" d="M150 211 C218 291 310 292 350 252'
          +'" fill="none" stroke="#10B981" stroke-width="6'
          +'" marker-end="url(#${rootId}-arrow)"/>';

        flowHTML+='<path class="bg-flow" d="M405 202 C462 160 498 152 526 166'
          +'" fill="none" stroke="#10B981" stroke-width="5'
          +'" marker-end="url(#${rootId}-arrow)"/>';

        stageNote.textContent='促进肝糖原分解等过程';
      }

      flows.innerHTML=flowHTML;

      var feedback='接近稳态';

      if(index>120){
        feedback='胰岛素调节';
      }else if(index<80){
        feedback='胰高血糖素调节';
      }

      feedbackState.textContent=feedback;

      var condition='当前进食、运动和激素反应相对协调。';

      if(index>130){
        condition='相对血糖指数较高，胰岛素调节作用应增强。';
      }else if(index<70){
        condition='相对血糖指数较低，胰高血糖素调节作用应增强。';
      }else if(insulinLevel<35 && foodLevel>65){
        condition='进食影响较强而胰岛素敏感性较低，血糖回落速度受到限制。';
      }else if(glucagonLevel<35 && exerciseLevel>65){
        condition='运动消耗较强而胰高血糖素反应较低，血糖回升能力受到限制。';
      }

      var info=information[stage];

      title.textContent=info.title;
      summary.textContent=info.summary;

      result.innerHTML=info.note
        +'<br>'+condition
        +' 胰岛素与胰高血糖素通过负反馈共同维持血糖的相对稳定。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        stage=this.getAttribute('data-stage');
        update();
      };
    }

    food.oninput=update;
    exercise.oninput=update;
    insulin.oninput=update;
    glucagon.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
