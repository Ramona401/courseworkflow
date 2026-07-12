/**
 * lifeScienceLabTemplatesMicrobiologyImmunity.ts
 *
 * 平面生命科学实验室：人体免疫防御。
 *
 * 教学边界：
 * 1. 展示屏障防御、先天性免疫、体液免疫、细胞免疫和免疫记忆；
 * 2. 先天性免疫反应较快，但特异性较弱；
 * 3. 抗体可特异性识别抗原，不等同于直接消灭所有病原体；
 * 4. 细胞毒性T细胞主要识别并清除被感染的宿主细胞；
 * 5. 本模型仅用于教学，不用于疾病判断或医学建议。
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

function immunityStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BFDBFE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .im-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DBEAFE,#EFF6FF);border-bottom:1px solid #BFDBFE}'
    + '#' + rootId + ' .im-title{font-size:15px;font-weight:800;color:#1E40AF}'
    + '#' + rootId + ' .im-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .im-body{height:calc(100% - 46px);display:grid;grid-template-columns:240px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .im-controls{padding:13px;overflow:auto;background:#F8FAFF;border-right:1px solid #BFDBFE}'
    + '#' + rootId + ' .im-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .im-row{margin-bottom:10px}'
    + '#' + rootId + ' .im-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .im-value{font-weight:800;color:#2563EB;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#2563EB}'
    + '#' + rootId + ' .im-subtitle{margin:7px 0;font-size:12px;font-weight:800;color:#1E40AF}'
    + '#' + rootId + ' .im-buttons{display:grid;grid-template-columns:1fr 1fr;gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .im-button{height:31px;padding:0 4px;border:1px solid #93C5FD;border-radius:8px;background:#fff;color:#1E40AF;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .im-button.active{border-color:#2563EB;background:#DBEAFE;box-shadow:0 3px 9px rgba(37,99,235,.13)}'
    + '#' + rootId + ' .im-result{padding:9px 10px;border-radius:10px;background:#DBEAFE;color:#1E3A8A;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' .im-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .im-card{padding:7px;border:1px solid #BFDBFE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .im-card b{display:block;font-size:16px;color:#1D4ED8}'
    + '#' + rootId + ' .im-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .im-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.5s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_IMMUNITY:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-human-immune-defense',
    group: '🦠 微生物与免疫',
    name: '人体免疫防御',
    emoji: '🛡️',
    desc: '观察屏障、先天性免疫、体液免疫、细胞免疫和免疫记忆的协同作用',
    params: [
      {
        key: 'pathogenLoad',
        label: '病原体相对数量',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 65,
      },
      {
        key: 'barrierIntegrity',
        label: '屏障完整度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'innateActivity',
        label: '先天免疫活跃度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 70,
      },
      {
        key: 'adaptiveActivity',
        label: '适应性免疫活跃度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
    ],

    buildHTML: (params, rootId) => {
      const pathogenLoad = num(params, 'pathogenLoad', 65)
      const barrierIntegrity = num(params, 'barrierIntegrity', 82)
      const innateActivity = num(params, 'innateActivity', 70)
      const adaptiveActivity = num(params, 'adaptiveActivity', 76)

      return `
<div id="${rootId}">
${immunityStyle(rootId)}
  <div class="im-head">
    <div class="im-title">🛡️ 人体免疫防御系统</div>
    <div class="im-note">教学示意，不用于疾病判断或医学建议</div>
  </div>

  <div class="im-body">
    <div class="im-controls">
      <div class="im-row">
        <div class="im-label">
          <span>病原体相对数量</span>
          <span class="im-value" data-pathogen-value></span>
        </div>
        <input data-pathogen type="range" min="10" max="100" step="1" value="${n(pathogenLoad)}">
      </div>

      <div class="im-row">
        <div class="im-label">
          <span>屏障完整度</span>
          <span class="im-value" data-barrier-value></span>
        </div>
        <input data-barrier type="range" min="20" max="100" step="1" value="${n(barrierIntegrity)}">
      </div>

      <div class="im-row">
        <div class="im-label">
          <span>先天免疫活跃度</span>
          <span class="im-value" data-innate-value></span>
        </div>
        <input data-innate type="range" min="20" max="100" step="1" value="${n(innateActivity)}">
      </div>

      <div class="im-row">
        <div class="im-label">
          <span>适应性免疫活跃度</span>
          <span class="im-value" data-adaptive-value></span>
        </div>
        <input data-adaptive type="range" min="20" max="100" step="1" value="${n(adaptiveActivity)}">
      </div>

      <div class="im-subtitle">选择防御阶段</div>

      <div class="im-buttons">
        <button type="button" class="im-button active" data-stage="barrier">屏障防御</button>
        <button type="button" class="im-button" data-stage="innate">先天性免疫</button>
        <button type="button" class="im-button" data-stage="humoral">体液免疫</button>
        <button type="button" class="im-button" data-stage="cellular">细胞免疫</button>
        <button type="button" class="im-button" data-stage="memory">免疫记忆</button>
      </div>

      <div class="im-status">
        <div class="im-card">
          <b data-entered></b>
          <span>突破屏障</span>
        </div>
        <div class="im-card">
          <b data-remaining></b>
          <span>防御后剩余</span>
        </div>
      </div>

      <div class="im-result" data-result></div>
    </div>

    <div class="im-stage">
      <svg viewBox="0 0 680 414" aria-label="人体免疫防御互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#1E40AF" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#1E40AF"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <g data-graphic filter="url(#${rootId}-shadow)"></g>

        <g transform="translate(28 370)">
          <circle cx="7" cy="7" r="7" fill="#EF4444"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">病原体</text>
        </g>

        <g transform="translate(142 370)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">吞噬细胞</text>
        </g>

        <g transform="translate(284 370)">
          <circle cx="7" cy="7" r="7" fill="#8B5CF6"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">抗体</text>
        </g>

        <g transform="translate(398 370)">
          <circle cx="7" cy="7" r="7" fill="#10B981"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">T细胞</text>
        </g>

        <text x="510" y="382" data-stage-note font-size="14" font-weight="900" fill="#1D4ED8"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var pathogen=root.querySelector('[data-pathogen]');
    var barrier=root.querySelector('[data-barrier]');
    var innate=root.querySelector('[data-innate]');
    var adaptive=root.querySelector('[data-adaptive]');

    var pathogenValue=root.querySelector('[data-pathogen-value]');
    var barrierValue=root.querySelector('[data-barrier-value]');
    var innateValue=root.querySelector('[data-innate-value]');
    var adaptiveValue=root.querySelector('[data-adaptive-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var enteredText=root.querySelector('[data-entered]');
    var remainingText=root.querySelector('[data-remaining]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var graphic=root.querySelector('[data-graphic]');
    var stageNote=root.querySelector('[data-stage-note]');

    var stage='barrier';

    var information={
      barrier:{
        title:'第一道防线：屏障防御',
        summary:'皮肤和黏膜等结构阻挡病原体进入机体',
        note:'屏障防御属于非特异性防御，是阻止病原体进入的重要第一道防线。'
      },
      innate:{
        title:'第二道防线：先天性免疫',
        summary:'吞噬细胞等快速识别并处理进入机体的异物',
        note:'先天性免疫反应较快，但通常不针对某一种特定抗原产生高度特异性的反应。'
      },
      humoral:{
        title:'体液免疫',
        summary:'B细胞增殖分化，产生能够特异性识别抗原的抗体',
        note:'抗体可与相应抗原特异性结合，协助中和、标记或清除病原体。'
      },
      cellular:{
        title:'细胞免疫',
        summary:'细胞毒性T细胞识别并清除被感染的宿主细胞',
        note:'细胞毒性T细胞主要作用于被感染细胞，而不是直接吞噬游离病毒。'
      },
      memory:{
        title:'免疫记忆',
        summary:'部分B细胞和T细胞形成记忆细胞，使再次应答更快更强',
        note:'免疫记忆具有特异性，再次遇到相同抗原时可更快启动适应性免疫。'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function pathogenParticle(x,y,size,opacity){
      var html='<g transform="translate('+x+' '+y+')" opacity="'+opacity+'">';

      for(var i=0;i<8;i++){
        var a=Math.PI*2*i/8;
        html+='<line x1="'+Math.cos(a)*size+'" y1="'+Math.sin(a)*size
          +'" x2="'+Math.cos(a)*(size+8)+'" y2="'+Math.sin(a)*(size+8)
          +'" stroke="#EF4444" stroke-width="2.5"/>';
      }

      html+='<circle cx="0" cy="0" r="'+size+'" fill="#FECACA" stroke="#DC2626" stroke-width="3"/>'
        +'</g>';

      return html;
    }

    function pathogens(count,startX,startY,spreadX,spreadY,opacity){
      var html='';

      for(var i=0;i<count;i++){
        var x=startX+(i%6)*spreadX;
        var y=startY+Math.floor(i/6)*spreadY;

        html+=pathogenParticle(
          x+(i%2)*6,
          y,
          9+i%3,
          opacity
        );
      }

      return html;
    }

    function renderBarrier(load,integrity){
      var incoming=Math.floor(4+load/10);
      var passed=Math.floor(incoming*(1-integrity/100));
      var html='';

      html+=pathogens(incoming,72,145,35,47,.86);

      html+='<rect x="298" y="106" width="50" height="225" rx="22'
        +'" fill="#FDE68A" stroke="#D97706" stroke-width="5"/>';

      for(var i=0;i<7;i++){
        var y=125+i*30;
        var gap=integrity<55 && i%3===1 ?14:2;

        html+='<rect x="'+(303+gap)+'" y="'+y
          +'" width="'+(40-gap*2)+'" height="21" rx="8" fill="#F59E0B"/>';
      }

      html+=pathogens(
        passed,
        394,
        165,
        42,
        48,
        .35+.65*(1-integrity/100)
      );

      html+='<path class="im-flow" d="M228 220 H286'
        +'" fill="none" stroke="#2563EB" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="250" y="350" font-size="15" font-weight="900" fill="#92400E">'
        +'皮肤和黏膜屏障</text>';

      return html;
    }

    function renderInnate(entered,activity){
      var html=pathogens(
        Math.max(2,Math.floor(entered/7)),
        390,
        150,
        42,
        55,
        .78
      );

      var cellCount=Math.floor(2+activity/28);

      for(var i=0;i<cellCount;i++){
        var x=150+(i%3)*105;
        var y=175+Math.floor(i/3)*95;

        html+='<circle cx="'+x+'" cy="'+y+'" r="42'
          +'" fill="#FEF3C7" stroke="#F59E0B" stroke-width="5"/>';

        html+='<path d="M'+(x-20)+' '+y
          +' Q'+x+' '+(y-28)+' '+(x+20)+' '+y
          +' Q'+x+' '+(y+26)+' '+(x-20)+' '+y
          +'" fill="#FBBF24" opacity=".72"/>';
      }

      html+='<path class="im-flow" d="M410 220 C354 220 325 220 286 220'
        +'" fill="none" stroke="#F59E0B" stroke-width="5'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="178" y="335" font-size="15" font-weight="900" fill="#B45309">'
        +'吞噬和炎症反应示意</text>';

      return html;
    }

    function renderHumoral(remaining,activity){
      var html=pathogens(
        Math.max(2,Math.floor(remaining/8)),
        375,
        145,
        46,
        55,
        .72
      );

      var antibodyCount=Math.floor(4+activity/10);

      for(var i=0;i<antibodyCount;i++){
        var x=145+(i%5)*47;
        var y=145+Math.floor(i/5)*58;

        html+='<path d="M'+x+' '+(y+24)
          +' V'+(y+8)
          +' M'+x+' '+(y+10)
          +' L'+(x-12)+' '+(y-4)
          +' M'+x+' '+(y+10)
          +' L'+(x+12)+' '+(y-4)
          +'" fill="none" stroke="#8B5CF6" stroke-width="6" stroke-linecap="round"/>';
      }

      html+='<path class="im-flow" d="M326 215 H362'
        +'" fill="none" stroke="#8B5CF6" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="130" y="335" font-size="15" font-weight="900" fill="#6D28D9">'
        +'抗体与相应抗原特异性结合</text>';

      return html;
    }

    function renderCellular(activity){
      var html='';

      html+='<ellipse cx="430" cy="222" rx="100" ry="78'
        +'" fill="#FEE2E2" stroke="#DC2626" stroke-width="5"/>';

      html+='<circle cx="430" cy="222" r="34'
        +'" fill="#FCA5A5" stroke="#B91C1C" stroke-width="4"/>';

      html+=pathogenParticle(457,190,11,.85);
      html+=pathogenParticle(398,248,10,.75);

      var cells=Math.floor(2+activity/30);

      for(var i=0;i<cells;i++){
        var x=140+(i%3)*78;
        var y=185+Math.floor(i/3)*90;

        html+='<circle cx="'+x+'" cy="'+y+'" r="34'
          +'" fill="#D1FAE5" stroke="#10B981" stroke-width="5"/>';

        html+='<text x="'+x+'" y="'+(y+7)
          +'" text-anchor="middle" font-size="21" font-weight="900" fill="#047857">T</text>';
      }

      html+='<path class="im-flow" d="M300 220 H322'
        +'" fill="none" stroke="#10B981" stroke-width="5'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="308" y="335" font-size="15" font-weight="900" fill="#047857">'
        +'识别并清除被感染细胞</text>';

      return html;
    }

    function renderMemory(activity){
      var html='';
      var memoryCount=Math.floor(4+activity/14);

      for(var i=0;i<memoryCount;i++){
        var angle=Math.PI*2*i/memoryCount;
        var x=340+Math.cos(angle)*125;
        var y=220+Math.sin(angle)*78;
        var color=i%2===0?'#2563EB':'#10B981';
        var label=i%2===0?'B':'T';

        html+='<circle cx="'+x+'" cy="'+y+'" r="30'
          +'" fill="#EFF6FF" stroke="'+color+'" stroke-width="5"/>';

        html+='<text x="'+x+'" y="'+(y+7)
          +'" text-anchor="middle" font-size="20" font-weight="900" fill="'+color+'">'
          +label+'</text>';
      }

      html+='<circle cx="340" cy="220" r="63'
        +'" fill="#DBEAFE" stroke="#2563EB" stroke-width="5"/>';

      html+='<text x="340" y="211" text-anchor="middle" font-size="18" font-weight="900" fill="#1E40AF">'
        +'再次遇到</text>';

      html+='<text x="340" y="238" text-anchor="middle" font-size="18" font-weight="900" fill="#1E40AF">'
        +'相同抗原</text>';

      html+='<text x="235" y="335" font-size="15" font-weight="900" fill="#1D4ED8">'
        +'记忆细胞使再次应答更快、更强</text>';

      return html;
    }

    function update(){
      var load=Number(pathogen.value);
      var barrierLevel=Number(barrier.value);
      var innateLevel=Number(innate.value);
      var adaptiveLevel=Number(adaptive.value);

      pathogenValue.textContent=load.toFixed(0)+'%';
      barrierValue.textContent=barrierLevel.toFixed(0)+'%';
      innateValue.textContent=innateLevel.toFixed(0)+'%';
      adaptiveValue.textContent=adaptiveLevel.toFixed(0)+'%';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var entered=load*(1-barrierLevel/100);
      var afterInnate=Math.max(
        0,
        entered-entered*innateLevel/100*.55
      );

      var remaining=Math.max(
        0,
        afterInnate-afterInnate*adaptiveLevel/100*.75
      );

      enteredText.textContent=entered.toFixed(0);
      remainingText.textContent=remaining.toFixed(0);

      var info=information[stage];

      title.textContent=info.title;
      summary.textContent=info.summary;

      if(stage==='barrier'){
        graphic.innerHTML=renderBarrier(load,barrierLevel);
        stageNote.textContent='非特异性防御';
      }else if(stage==='innate'){
        graphic.innerHTML=renderInnate(entered,innateLevel);
        stageNote.textContent='反应较快';
      }else if(stage==='humoral'){
        graphic.innerHTML=renderHumoral(afterInnate,adaptiveLevel);
        stageNote.textContent='抗原特异性';
      }else if(stage==='cellular'){
        graphic.innerHTML=renderCellular(adaptiveLevel);
        stageNote.textContent='清除感染细胞';
      }else{
        graphic.innerHTML=renderMemory(adaptiveLevel);
        stageNote.textContent='再次应答更快';
      }

      var condition='当前多层免疫防御能够共同降低病原体负荷。';

      if(barrierLevel<35){
        condition='屏障完整度较低，较多病原体可能进入机体内部。';
      }else if(innateLevel<35){
        condition='先天性免疫活跃度较低，早期控制能力相对有限。';
      }else if(adaptiveLevel<35){
        condition='适应性免疫活跃度较低，特异性应答和免疫记忆形成受到限制。';
      }else if(load>90){
        condition='病原体负荷很高，即使多层防御同时参与，仍可能有较多病原体剩余。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' 免疫系统由多种细胞、分子和器官协同作用，本图为简化模型。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        stage=this.getAttribute('data-stage');
        update();
      };
    }

    pathogen.oninput=update;
    barrier.oninput=update;
    innate.oninput=update;
    adaptive.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
