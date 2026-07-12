/**
 * lifeScienceLabTemplatesRegulationReflex.ts
 *
 * 平面生命科学实验室：反射弧与神经调节。
 *
 * 教学边界：
 * 1. 反射弧包括感受器、传入神经、神经中枢、传出神经和效应器；
 * 2. 神经冲动在突触处通常具有单向传递特点；
 * 3. 脊髓反射可快速完成，同时信息也可上传至大脑形成感觉；
 * 4. 反应时间和反应强度均为相对教学指标。
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

function reflexStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C7D2FE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .rf-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0E7FF,#EEF2FF);border-bottom:1px solid #C7D2FE}'
    + '#' + rootId + ' .rf-title{font-size:15px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .rf-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .rf-body{height:calc(100% - 46px);display:grid;grid-template-columns:240px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .rf-controls{padding:13px;overflow:auto;background:#FAFAFF;border-right:1px solid #C7D2FE}'
    + '#' + rootId + ' .rf-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .rf-row{margin-bottom:11px}'
    + '#' + rootId + ' .rf-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .rf-value{font-weight:800;color:#4F46E5;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#4F46E5}'
    + '#' + rootId + ' .rf-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .rf-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .rf-button{height:31px;padding:0 4px;border:1px solid #A5B4FC;border-radius:8px;background:#fff;color:#3730A3;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .rf-button.active{border-color:#4F46E5;background:#E0E7FF;box-shadow:0 3px 9px rgba(79,70,229,.13)}'
    + '#' + rootId + ' .rf-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#818CF8,#4F46E5);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .rf-auto.paused{background:#64748B}'
    + '#' + rootId + ' .rf-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .rf-card{padding:7px;border:1px solid #C7D2FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .rf-card b{display:block;font-size:16px;color:#4338CA}'
    + '#' + rootId + ' .rf-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .rf-result{padding:9px 10px;border-radius:10px;background:#E0E7FF;color:#312E81;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .rf-signal{stroke-dasharray:8 7;animation:' + rootId + '-signal var(--rf-speed,1.5s) linear infinite}'
    + '@keyframes ' + rootId + '-signal{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_REFLEX:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-reflex-arc',
    group: '🧠 稳态与调节',
    name: '反射弧与神经调节',
    emoji: '⚡',
    desc: '观察感受器、传入神经、神经中枢、传出神经和效应器，模拟缩手反射',
    params: [
      {
        key: 'stimulusIntensity',
        label: '刺激强度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'synapseEfficiency',
        label: '突触传递效率',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'muscleFatigue',
        label: '肌肉疲劳程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 18,
      },
    ],

    buildHTML: (params, rootId) => {
      const stimulusIntensity = num(params, 'stimulusIntensity', 68)
      const synapseEfficiency = num(params, 'synapseEfficiency', 82)
      const muscleFatigue = num(params, 'muscleFatigue', 18)

      return `
<div id="${rootId}">
${reflexStyle(rootId)}
  <div class="rf-head">
    <div class="rf-title">⚡ 反射弧与神经调节</div>
    <div class="rf-note">缩手反射示意：信息也可上传至大脑形成感觉</div>
  </div>

  <div class="rf-body">
    <div class="rf-controls">
      <div class="rf-row">
        <div class="rf-label">
          <span>刺激强度</span>
          <span class="rf-value" data-stimulus-value></span>
        </div>
        <input data-stimulus type="range" min="10" max="100" step="1" value="${n(stimulusIntensity)}">
      </div>

      <div class="rf-row">
        <div class="rf-label">
          <span>突触传递效率</span>
          <span class="rf-value" data-synapse-value></span>
        </div>
        <input data-synapse type="range" min="20" max="100" step="1" value="${n(synapseEfficiency)}">
      </div>

      <div class="rf-row">
        <div class="rf-label">
          <span>肌肉疲劳程度</span>
          <span class="rf-value" data-fatigue-value></span>
        </div>
        <input data-fatigue type="range" min="0" max="100" step="1" value="${n(muscleFatigue)}">
      </div>

      <div class="rf-subtitle">观察反射弧环节</div>

      <div class="rf-buttons">
        <button type="button" class="rf-button active" data-stage="receptor">1. 感受器</button>
        <button type="button" class="rf-button" data-stage="sensory">2. 传入神经</button>
        <button type="button" class="rf-button" data-stage="center">3. 神经中枢</button>
        <button type="button" class="rf-button" data-stage="motor">4. 传出神经</button>
        <button type="button" class="rf-button" data-stage="effector">5. 效应器</button>
      </div>

      <button type="button" class="rf-auto" data-auto>自动演示：运行中</button>

      <div class="rf-status">
        <div class="rf-card">
          <b data-time></b>
          <span>相对反应时间</span>
        </div>
        <div class="rf-card">
          <b data-strength></b>
          <span>相对反应强度</span>
        </div>
      </div>

      <div class="rf-result" data-result></div>
    </div>

    <div class="rf-stage">
      <svg viewBox="0 0 680 414" aria-label="反射弧互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#4F46E5"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#312E81" flood-opacity=".14"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#3730A3"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <g filter="url(#${rootId}-shadow)">
          <rect x="66" y="166" width="122" height="112" rx="42" fill="#FDE68A" stroke="#D97706" stroke-width="5"/>
          <path d="M76 215 C105 191 143 190 178 215" fill="none" stroke="#F59E0B" stroke-width="12" stroke-linecap="round"/>
          <circle cx="154" cy="206" r="12" fill="#EF4444" stroke="#B91C1C" stroke-width="3"/>

          <path d="M188 206 C250 124 315 142 340 201" fill="none" stroke="#2563EB" stroke-width="10" stroke-linecap="round"/>
          <path d="M341 235 C303 305 230 323 170 267" fill="none" stroke="#10B981" stroke-width="10" stroke-linecap="round"/>

          <rect x="330" y="139" width="104" height="140" rx="43" fill="#EDE9FE" stroke="#7C3AED" stroke-width="6"/>
          <path d="M382 155 V264" stroke="#A78BFA" stroke-width="17" stroke-linecap="round"/>
          <path d="M350 193 Q382 164 414 193 Q382 222 350 193Z" fill="#C4B5FD"/>
          <path d="M350 230 Q382 201 414 230 Q382 259 350 230Z" fill="#C4B5FD"/>

          <ellipse cx="555" cy="237" rx="84" ry="52" fill="#FECACA" stroke="#E11D48" stroke-width="5"/>
          <path d="M493 237 Q555 194 617 237 Q555 280 493 237Z" fill="#FB7185" opacity=".7"/>

          <path d="M382 139 C382 98 412 84 450 80" fill="none" stroke="#94A3B8" stroke-width="5" stroke-dasharray="8 7"/>
          <ellipse cx="486" cy="75" rx="45" ry="30" fill="#DBEAFE" stroke="#2563EB" stroke-width="4"/>
          <text x="486" y="81" text-anchor="middle" font-size="15" font-weight="900" fill="#1E40AF">大脑</text>
        </g>

        <g data-signals></g>
        <g data-highlight></g>

        <text x="89" y="309" font-size="14" font-weight="900" fill="#92400E">感受器</text>
        <text x="213" y="126" font-size="14" font-weight="900" fill="#1D4ED8">传入神经</text>
        <text x="346" y="308" font-size="14" font-weight="900" fill="#6D28D9">脊髓中枢</text>
        <text x="215" y="344" font-size="14" font-weight="900" fill="#047857">传出神经</text>
        <text x="524" y="318" font-size="14" font-weight="900" fill="#BE123C">效应器</text>

        <text x="442" y="116" font-size="12" font-weight="800" fill="#64748B">信息上传形成感觉</text>

        <g transform="translate(28 373)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">传入神经冲动</text>
        </g>

        <g transform="translate(204 373)">
          <circle cx="7" cy="7" r="7" fill="#10B981"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">传出神经冲动</text>
        </g>

        <text x="440" y="385" data-stage-note font-size="14" font-weight="900" fill="#4338CA"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var stimulus=root.querySelector('[data-stimulus]');
    var synapse=root.querySelector('[data-synapse]');
    var fatigue=root.querySelector('[data-fatigue]');

    var stimulusValue=root.querySelector('[data-stimulus-value]');
    var synapseValue=root.querySelector('[data-synapse-value]');
    var fatigueValue=root.querySelector('[data-fatigue-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var autoButton=root.querySelector('[data-auto]');
    var timeText=root.querySelector('[data-time]');
    var strengthText=root.querySelector('[data-strength]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var signals=root.querySelector('[data-signals]');
    var highlight=root.querySelector('[data-highlight]');
    var stageNote=root.querySelector('[data-stage-note]');

    var stages=['receptor','sensory','center','motor','effector'];

    var information={
      receptor:{
        title:'感受器接受刺激',
        summary:'皮肤中的感受器将刺激转换为神经冲动',
        note:'感受器能够接受适宜刺激，并把刺激信息转换为神经信号。'
      },
      sensory:{
        title:'传入神经传递信息',
        summary:'神经冲动由感受器沿传入神经到达脊髓',
        note:'传入神经把感受器产生的神经冲动传向神经中枢。'
      },
      center:{
        title:'神经中枢分析整合',
        summary:'脊髓中的神经元通过突触完成信息传递和初步整合',
        note:'神经冲动在突触处通常单向传递，反射中枢发出相应指令。'
      },
      motor:{
        title:'传出神经传递指令',
        summary:'神经冲动由神经中枢沿传出神经到达效应器',
        note:'传出神经把神经中枢发出的指令传递给肌肉等效应器。'
      },
      effector:{
        title:'效应器产生反应',
        summary:'骨骼肌收缩，使手迅速避开有害刺激',
        note:'效应器接受传出神经的指令并作出反应，完成反射活动。'
      }
    };

    var stage='receptor';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function circle(x,y,color,size,opacity){
      return '<circle cx="'+x+'" cy="'+y+'" r="'+size
        +'" fill="'+color+'" stroke="#FFFFFF" stroke-width="2" opacity="'+opacity+'"/>';
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(!automatic || !document.body.contains(root)){
        return;
      }

      var efficiency=Number(synapse.value);
      var interval=clamp(2600-efficiency*13,850,2300);

      timer=window.setTimeout(function(){
        var index=stages.indexOf(stage);
        stage=stages[(index+1)%stages.length];
        update();
        schedule();
      },interval);
    }

    function update(){
      var stimulusLevel=Number(stimulus.value);
      var synapseLevel=Number(synapse.value);
      var fatigueLevel=Number(fatigue.value);

      stimulusValue.textContent=stimulusLevel.toFixed(0)+'%';
      synapseValue.textContent=synapseLevel.toFixed(0)+'%';
      fatigueValue.textContent=fatigueLevel.toFixed(0)+'%';

      var responseStrength=
        stimulusLevel
        *synapseLevel/100
        *(1-fatigueLevel/100);

      var reactionTime=
        100
        -stimulusLevel*.25
        -synapseLevel*.38
        +fatigueLevel*.18;

      reactionTime=clamp(reactionTime,18,100);
      responseStrength=clamp(responseStrength,0,100);

      timeText.textContent=reactionTime.toFixed(0);
      strengthText.textContent=responseStrength.toFixed(0);

      root.style.setProperty(
        '--rf-speed',
        clamp(2.5-synapseLevel/55,.55,2.3).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var info=information[stage];

      title.textContent=info.title;
      summary.textContent=info.summary;

      var signalHTML='';
      var highlightHTML='';

      if(stage==='receptor'){
        signalHTML+=circle(154,206,'#EF4444',8+stimulusLevel/18,.45+stimulusLevel/180);
        highlightHTML='<circle cx="154" cy="206" r="27" fill="none" stroke="#EF4444" stroke-width="5"/>';
        stageNote.textContent='刺激转为神经冲动';
      }else if(stage==='sensory'){
        signalHTML+='<path class="rf-signal" d="M174 200 C250 120 315 143 345 198'
          +'" fill="none" stroke="#2563EB" stroke-width="'+(4+stimulusLevel/22)
          +'" marker-end="url(#${rootId}-arrow)"/>';
        highlightHTML='<path d="M188 206 C250 124 315 142 340 201" fill="none" stroke="#60A5FA" stroke-width="18" opacity=".28"/>';
        stageNote.textContent='传向神经中枢';
      }else if(stage==='center'){
        signalHTML+=circle(382,193,'#8B5CF6',11,.85);
        signalHTML+=circle(382,230,'#8B5CF6',11,.85);
        highlightHTML='<rect x="321" y="130" width="122" height="158" rx="49" fill="none" stroke="#7C3AED" stroke-width="5"/>';
        stageNote.textContent='突触单向传递';
      }else if(stage==='motor'){
        signalHTML+='<path class="rf-signal" d="M341 235 C303 305 230 323 170 267'
          +'" fill="none" stroke="#10B981" stroke-width="'+(4+synapseLevel/22)
          +'" marker-end="url(#${rootId}-arrow)"/>';
        highlightHTML='<path d="M341 235 C303 305 230 323 170 267" fill="none" stroke="#34D399" stroke-width="18" opacity=".28"/>';
        stageNote.textContent='指令传向效应器';
      }else{
        var contraction=8+responseStrength/8;

        signalHTML+='<path d="M500 237 Q555 '+(237-contraction)
          +' 610 237 Q555 '+(237+contraction)+' 500 237Z'
          +'" fill="#E11D48" opacity=".56"/>';

        highlightHTML='<ellipse cx="555" cy="237" rx="96" ry="63" fill="none" stroke="#E11D48" stroke-width="5"/>';
        stageNote.textContent='肌肉收缩产生反应';
      }

      signals.innerHTML=signalHTML;
      highlight.innerHTML=highlightHTML;

      var condition='当前刺激、突触传递和肌肉状态能够形成较明显的反射反应。';

      if(stimulusLevel<25){
        condition='刺激较弱，感受器产生的神经冲动和最终反应相对较弱。';
      }else if(synapseLevel<35){
        condition='突触传递效率较低，神经中枢内的信息传递受到明显影响。';
      }else if(fatigueLevel>75){
        condition='肌肉疲劳程度较高，即使神经指令到达，效应器反应也会减弱。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' 反射弧是完成反射活动的结构基础。脊髓可快速完成缩手反射，同时相关信息也可上传至大脑形成感觉。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        automatic=false;
        autoButton.textContent='自动演示：已暂停';
        autoButton.classList.add('paused');
        stage=this.getAttribute('data-stage');
        update();
        schedule();
      };
    }

    autoButton.onclick=function(){
      automatic=!automatic;

      autoButton.textContent=automatic
        ?'自动演示：运行中'
        :'自动演示：已暂停';

      autoButton.classList.toggle('paused',!automatic);

      update();
      schedule();
    };

    stimulus.oninput=update;
    fatigue.oninput=update;

    synapse.oninput=function(){
      update();
      schedule();
    };

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
