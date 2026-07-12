/**
 * lifeScienceLabTemplatesHumanDigestion.ts
 *
 * 人体消化过程互动模型。
 * 所有效率均为相对教学指标，不用于医学诊断。
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

function style(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #FED7AA;border-radius:16px;background:#fff;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' *{box-sizing:border-box;}\n'
    + '#' + rootId + ' .dg-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#FFEDD5,#FFF7ED);border-bottom:1px solid #FED7AA;}\n'
    + '#' + rootId + ' .dg-title{font-size:15px;font-weight:800;color:#9A3412;}\n'
    + '#' + rootId + ' .dg-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .dg-body{height:calc(100% - 46px);display:grid;grid-template-columns:230px minmax(0,1fr);min-height:0;}\n'
    + '#' + rootId + ' .dg-controls{padding:13px;border-right:1px solid #FED7AA;background:#FFFBF7;overflow:auto;}\n'
    + '#' + rootId + ' .dg-stage{position:relative;min-width:0;min-height:0;background:#fff;}\n'
    + '#' + rootId + ' .dg-row{margin-bottom:11px;}\n'
    + '#' + rootId + ' .dg-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155;}\n'
    + '#' + rootId + ' .dg-value{font-weight:800;color:#C2410C;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#EA580C;}\n'
    + '#' + rootId + ' .dg-stage-title{margin:9px 0 7px;font-size:12px;font-weight:800;color:#7C2D12;}\n'
    + '#' + rootId + ' .dg-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:10px;}\n'
    + '#' + rootId + ' .dg-button{height:31px;border:1px solid #FDBA74;border-radius:8px;background:#fff;color:#9A3412;font-size:11px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .dg-button.active{border-color:#EA580C;background:#FFEDD5;color:#7C2D12;box-shadow:0 3px 9px rgba(194,65,12,.12);}\n'
    + '#' + rootId + ' .dg-result{padding:9px 10px;border-radius:10px;background:#FFEDD5;color:#7C2D12;font-size:11.5px;line-height:1.5;font-weight:600;}\n'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%;}\n'
    + '#' + rootId + ' .dg-flow{stroke-dasharray:8 8;animation:' + rootId + '-flow 1.6s linear infinite;}\n'
    + '#' + rootId + ' .dg-organ{transition:opacity .2s ease,transform .2s ease;transform-box:fill-box;transform-origin:center;}\n'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32;}}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_DIGESTION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-human-digestion',
    group: '🫀 人体生命活动',
    name: '消化过程',
    emoji: '🍽️',
    desc: '调节咀嚼、消化酶、胆汁乳化和小肠吸收状态，观察营养物质的消化与吸收',
    params: [
      {
        key: 'chewing',
        label: '咀嚼充分度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'enzyme',
        label: '消化酶活性',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'bile',
        label: '胆汁乳化作用',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 70,
      },
      {
        key: 'absorption',
        label: '小肠吸收状态',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
    ],

    buildHTML: (params, rootId) => {
      const chewing = num(params, 'chewing', 72)
      const enzyme = num(params, 'enzyme', 78)
      const bile = num(params, 'bile', 70)
      const absorption = num(params, 'absorption', 82)

      return `
<div id="${rootId}">
${style(rootId)}
  <div class="dg-head">
    <div class="dg-title">🍽️ 人体消化与营养物质吸收</div>
    <div class="dg-note">相对教学模型，不代表人体真实消化效率</div>
  </div>

  <div class="dg-body">
    <div class="dg-controls">
      <div class="dg-row">
        <div class="dg-label">
          <span>咀嚼充分度</span>
          <span class="dg-value" data-chewing-value></span>
        </div>
        <input data-chewing type="range" min="0" max="100" step="1" value="${n(chewing)}">
      </div>

      <div class="dg-row">
        <div class="dg-label">
          <span>消化酶活性</span>
          <span class="dg-value" data-enzyme-value></span>
        </div>
        <input data-enzyme type="range" min="0" max="100" step="1" value="${n(enzyme)}">
      </div>

      <div class="dg-row">
        <div class="dg-label">
          <span>胆汁乳化作用</span>
          <span class="dg-value" data-bile-value></span>
        </div>
        <input data-bile type="range" min="0" max="100" step="1" value="${n(bile)}">
      </div>

      <div class="dg-row">
        <div class="dg-label">
          <span>小肠吸收状态</span>
          <span class="dg-value" data-absorption-value></span>
        </div>
        <input data-absorption type="range" min="0" max="100" step="1" value="${n(absorption)}">
      </div>

      <div class="dg-stage-title">选择消化阶段</div>

      <div class="dg-buttons">
        <button type="button" class="dg-button active" data-stage-button="mouth">口腔</button>
        <button type="button" class="dg-button" data-stage-button="stomach">胃</button>
        <button type="button" class="dg-button" data-stage-button="smallIntestine">小肠</button>
        <button type="button" class="dg-button" data-stage-button="largeIntestine">大肠</button>
      </div>

      <div class="dg-result" data-result></div>
    </div>

    <div class="dg-stage">
      <svg viewBox="0 0 680 414" aria-label="人体消化过程互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#EA580C"/>
          </marker>
          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#9A3412" flood-opacity=".16"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="30" y="37" data-stage-name font-size="25" font-weight="900" fill="#9A3412"></text>
        <text x="30" y="66" data-stage-function font-size="15" font-weight="800" fill="#475569"></text>

        <g data-organ="mouth" class="dg-organ" filter="url(#${rootId}-shadow)">
          <ellipse cx="86" cy="139" rx="47" ry="31" fill="#FECACA" stroke="#BE123C" stroke-width="4"/>
          <path d="M53 139 Q86 164 119 139" fill="none" stroke="#FFFFFF" stroke-width="8" stroke-linecap="round"/>
          <path d="M72 129 Q86 119 100 129" fill="none" stroke="#E11D48" stroke-width="5" stroke-linecap="round"/>
          <text x="66" y="190" font-size="15" font-weight="900" fill="#881337">口腔</text>
        </g>

        <path d="M128 143 C168 146 172 170 190 192" fill="none" stroke="#FDBA74" stroke-width="18" stroke-linecap="round"/>
        <path class="dg-flow" d="M130 143 C168 146 172 170 190 192" fill="none" stroke="#EA580C" stroke-width="4" marker-end="url(#${rootId}-arrow)"/>

        <g data-organ="stomach" class="dg-organ" filter="url(#${rootId}-shadow)">
          <path d="M200 164 C252 142 306 160 300 212 C296 256 250 283 214 252 C190 231 194 196 200 164Z"
            fill="#FDBA74" stroke="#C2410C" stroke-width="5"/>
          <path d="M216 194 C246 177 274 190 278 218 C266 249 230 257 210 233"
            fill="none" stroke="#FFEDD5" stroke-width="7" stroke-linecap="round"/>
          <text x="226" y="304" font-size="15" font-weight="900" fill="#9A3412">胃</text>
        </g>

        <g>
          <path d="M340 119 C398 93 464 108 474 148 C430 171 374 168 334 148Z"
            fill="#B45309" stroke="#78350F" stroke-width="4"/>
          <ellipse cx="432" cy="164" rx="13" ry="21" fill="#65A30D" stroke="#3F6212" stroke-width="3"/>
          <path data-bile-flow d="M432 184 C422 207 410 219 398 233"
            fill="none" stroke="#65A30D" stroke-width="6" stroke-linecap="round" marker-end="url(#${rootId}-arrow)"/>
          <text x="356" y="91" font-size="14" font-weight="900" fill="#78350F">肝脏和胆囊</text>
        </g>

        <g data-organ="smallIntestine" class="dg-organ" filter="url(#${rootId}-shadow)">
          <rect x="345" y="205" width="170" height="145" rx="39"
            fill="#FFEDD5" stroke="#EA580C" stroke-width="6"/>
          <path d="M372 232 C475 220 480 252 385 262 C350 269 362 291 469 287 C506 286 496 317 382 318"
            fill="none" stroke="#FB923C" stroke-width="15" stroke-linecap="round"/>
          <path class="dg-flow"
            d="M372 232 C475 220 480 252 385 262 C350 269 362 291 469 287 C506 286 496 317 382 318"
            fill="none" stroke="#FFF7ED" stroke-width="4" marker-end="url(#${rootId}-arrow)"/>
          <text x="402" y="380" font-size="15" font-weight="900" fill="#9A3412">小肠</text>
        </g>

        <g data-organ="largeIntestine" class="dg-organ" filter="url(#${rootId}-shadow)">
          <path d="M552 198 C608 198 630 230 624 274 V336 C624 362 603 374 580 374 H548"
            fill="none" stroke="#A78BFA" stroke-width="25" stroke-linecap="round" stroke-linejoin="round"/>
          <path class="dg-flow"
            d="M552 198 C608 198 630 230 624 274 V336 C624 362 603 374 580 374 H548"
            fill="none" stroke="#F5F3FF" stroke-width="5" marker-end="url(#${rootId}-arrow)"/>
          <text x="566" y="398" font-size="15" font-weight="900" fill="#6D28D9">大肠</text>
        </g>

        <path class="dg-flow" d="M292 242 C320 247 328 244 351 238"
          fill="none" stroke="#EA580C" stroke-width="6" marker-end="url(#${rootId}-arrow)"/>
        <path class="dg-flow" d="M510 312 C535 304 542 265 552 230"
          fill="none" stroke="#EA580C" stroke-width="6" marker-end="url(#${rootId}-arrow)"/>

        <g data-food-particles></g>
        <g data-absorption-particles></g>

        <g transform="translate(26 337)">
          <text x="0" y="0" font-size="13" font-weight="800" fill="#475569">糖类消化</text>
          <rect x="76" y="-13" width="150" height="15" rx="7" fill="#E2E8F0"/>
          <rect data-carb-bar x="76" y="-13" width="0" height="15" rx="7" fill="#F59E0B"/>

          <text x="0" y="27" font-size="13" font-weight="800" fill="#475569">蛋白质消化</text>
          <rect x="76" y="14" width="150" height="15" rx="7" fill="#E2E8F0"/>
          <rect data-protein-bar x="76" y="14" width="0" height="15" rx="7" fill="#E11D48"/>

          <text x="0" y="54" font-size="13" font-weight="800" fill="#475569">脂肪消化</text>
          <rect x="76" y="41" width="150" height="15" rx="7" fill="#E2E8F0"/>
          <rect data-fat-bar x="76" y="41" width="0" height="15" rx="7" fill="#65A30D"/>
        </g>

        <text x="286" y="397" data-absorption-text font-size="15" font-weight="900" fill="#0369A1"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var chewing=root.querySelector('[data-chewing]');
    var enzyme=root.querySelector('[data-enzyme]');
    var bile=root.querySelector('[data-bile]');
    var absorption=root.querySelector('[data-absorption]');

    var chewingValue=root.querySelector('[data-chewing-value]');
    var enzymeValue=root.querySelector('[data-enzyme-value]');
    var bileValue=root.querySelector('[data-bile-value]');
    var absorptionValue=root.querySelector('[data-absorption-value]');

    var buttons=root.querySelectorAll('[data-stage-button]');
    var organs=root.querySelectorAll('[data-organ]');
    var result=root.querySelector('[data-result]');
    var stageName=root.querySelector('[data-stage-name]');
    var stageFunction=root.querySelector('[data-stage-function]');

    var foodParticles=root.querySelector('[data-food-particles]');
    var absorptionParticles=root.querySelector('[data-absorption-particles]');
    var bileFlow=root.querySelector('[data-bile-flow]');

    var carbBar=root.querySelector('[data-carb-bar]');
    var proteinBar=root.querySelector('[data-protein-bar]');
    var fatBar=root.querySelector('[data-fat-bar]');
    var absorptionText=root.querySelector('[data-absorption-text]');

    var selectedStage='mouth';

    var stages={
      mouth:{
        name:'第一站：口腔',
        fn:'牙齿切断磨碎食物，唾液淀粉酶开始消化部分淀粉。',
        note:'口腔中的咀嚼属于物理性消化，唾液淀粉酶参与化学性消化。'
      },
      stomach:{
        name:'第二站：胃',
        fn:'胃的蠕动混合食物，胃液主要参与蛋白质的初步消化。',
        note:'胃具有暂时储存、机械搅拌和初步消化蛋白质等作用。'
      },
      smallIntestine:{
        name:'第三站：小肠',
        fn:'多种消化液共同作用，是消化和吸收的主要场所。',
        note:'小肠很长，内表面有绒毛，可显著扩大营养物质吸收面积。'
      },
      largeIntestine:{
        name:'第四站：大肠',
        fn:'吸收部分水和无机盐，形成并暂时储存粪便。',
        note:'大肠不是主要消化场所，其重要作用之一是回收部分水分。'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function selectStage(stage){
      selectedStage=stage;

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage-button')===stage
        );
      }

      for(var j=0;j<organs.length;j++){
        var active=organs[j].getAttribute('data-organ')===stage;
        organs[j].style.opacity=active?'1':'0.38';
        organs[j].style.transform=active?'scale(1.045)':'scale(1)';
      }

      update();
    }

    function update(){
      var c=Number(chewing.value);
      var e=Number(enzyme.value);
      var b=Number(bile.value);
      var a=Number(absorption.value);

      var chewingFactor=.3+.7*c/100;
      var enzymeFactor=.25+.75*e/100;
      var bileFactor=.25+.75*b/100;

      var carbohydrate=clamp(100*chewingFactor*enzymeFactor,0,100);
      var protein=clamp(100*(.22+.78*enzymeFactor),0,100);
      var fat=clamp(100*bileFactor*(.35+.65*enzymeFactor),0,100);
      var average=(carbohydrate+protein+fat)/3;
      var absorbed=average*a/100;

      chewingValue.textContent=c.toFixed(0)+'%';
      enzymeValue.textContent=e.toFixed(0)+'%';
      bileValue.textContent=b.toFixed(0)+'%';
      absorptionValue.textContent=a.toFixed(0)+'%';

      carbBar.setAttribute('width',String(1.5*carbohydrate));
      proteinBar.setAttribute('width',String(1.5*protein));
      fatBar.setAttribute('width',String(1.5*fat));

      absorptionText.textContent='相对吸收水平 '+absorbed.toFixed(0);
      bileFlow.setAttribute('stroke-width',String(2+b/16));
      bileFlow.setAttribute('opacity',String(.25+b/135));

      var foodHTML='';
      var foodCount=Math.floor(clamp(13-c/10,3,12));

      for(var p=0;p<foodCount;p++){
        var angle=p*2.399;
        var radius=4+(p%4)*5;
        var x=86+Math.cos(angle)*radius;
        var y=139+Math.sin(angle)*radius;

        foodHTML+='<circle cx="'+x+'" cy="'+y+'" r="'
          +(3+(100-c)/28)
          +'" fill="#F59E0B" stroke="#92400E" stroke-width="1.5"/>';
      }

      foodParticles.innerHTML=foodHTML;

      var absorbedHTML='';
      var absorbedCount=Math.floor(absorbed/9);

      for(var q=0;q<absorbedCount;q++){
        var ax=374+(q%7)*18;
        var ay=219+Math.floor(q/7)*28;
        var color=q%3===0?'#F59E0B':q%3===1?'#E11D48':'#65A30D';

        absorbedHTML+='<circle cx="'+ax+'" cy="'+ay+'" r="5" fill="'
          +color+'" opacity=".82"/>'
          +'<path d="M'+ax+' '+(ay+6)
          +' v18" stroke="#0284C7" stroke-width="2" opacity=".65"/>';
      }

      absorptionParticles.innerHTML=absorbedHTML;

      var info=stages[selectedStage];
      stageName.textContent=info.name;
      stageFunction.textContent=info.fn;

      var condition='当前各项条件较协调，消化和吸收过程相对顺畅。';

      if(c<25){
        condition='咀嚼不充分，食物颗粒较大，与消化液接触的表面积较小。';
      }else if(e<25){
        condition='消化酶活性较低，营养物质的化学性消化受到限制。';
      }else if(b<25){
        condition='胆汁乳化作用较弱，脂肪分散程度较低，脂肪消化受到影响。';
      }else if(a<30){
        condition='小肠吸收状态较低，已经分解的小分子营养物质吸收不足。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' 各项数值仅用于比较变量影响。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        selectStage(this.getAttribute('data-stage-button'));
      };
    }

    chewing.oninput=update;
    enzyme.oninput=update;
    bile.oninput=update;
    absorption.oninput=update;

    selectStage('mouth');
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
