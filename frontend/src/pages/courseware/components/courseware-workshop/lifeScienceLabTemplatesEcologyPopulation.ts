/**
 * lifeScienceLabTemplatesEcologyPopulation.ts
 *
 * 平面生命科学实验室：种群数量变化。
 *
 * 教学边界：
 * 1. J型增长表示资源和空间近似无限等理想条件下的指数增长；
 * 2. S型增长表示环境阻力存在时，种群数量逐渐接近环境容纳量；
 * 3. 真实种群常在环境容纳量附近波动，不一定形成平滑曲线；
 * 4. 模型数值为相对教学单位，不用于真实种群预测。
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

function populationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #A7F3D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .pg-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#D1FAE5,#ECFDF5);border-bottom:1px solid #A7F3D0}'
    + '#' + rootId + ' .pg-title{font-size:15px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .pg-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .pg-body{height:calc(100% - 46px);display:grid;grid-template-columns:232px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .pg-controls{padding:13px;overflow:auto;background:#F8FFFC;border-right:1px solid #A7F3D0}'
    + '#' + rootId + ' .pg-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .pg-row{margin-bottom:11px}'
    + '#' + rootId + ' .pg-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .pg-value{font-weight:800;color:#059669;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981}'
    + '#' + rootId + ' .pg-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .pg-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:10px}'
    + '#' + rootId + ' .pg-button{height:32px;border:1px solid #6EE7B7;border-radius:8px;background:#fff;color:#065F46;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pg-button.active{border-color:#10B981;background:#D1FAE5;box-shadow:0 3px 9px rgba(16,185,129,.14)}'
    + '#' + rootId + ' .pg-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .pg-card{padding:7px;border:1px solid #A7F3D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .pg-card b{display:block;font-size:16px;color:#047857}'
    + '#' + rootId + ' .pg-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .pg-result{padding:9px 10px;border-radius:10px;background:#D1FAE5;color:#064E3B;font-size:11.5px;line-height:1.52;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_POPULATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-population-growth',
    group: '🌎 生态系统',
    name: '种群数量变化',
    emoji: '📈',
    desc: '调节初始数量、出生率、死亡率和环境容纳量，比较J型增长与S型增长',
    params: [
      {
        key: 'initialPopulation',
        label: '初始种群数量',
        type: 'number',
        min: 10,
        max: 150,
        step: 5,
        defaultValue: 45,
      },
      {
        key: 'birthRate',
        label: '相对出生率/%',
        type: 'number',
        min: 0,
        max: 50,
        step: 1,
        defaultValue: 28,
      },
      {
        key: 'deathRate',
        label: '相对死亡率/%',
        type: 'number',
        min: 0,
        max: 40,
        step: 1,
        defaultValue: 8,
      },
      {
        key: 'carryingCapacity',
        label: '环境容纳量K',
        type: 'number',
        min: 100,
        max: 1000,
        step: 50,
        defaultValue: 500,
      },
    ],

    buildHTML: (params, rootId) => {
      const initialPopulation = num(params, 'initialPopulation', 45)
      const birthRate = num(params, 'birthRate', 28)
      const deathRate = num(params, 'deathRate', 8)
      const carryingCapacity = num(params, 'carryingCapacity', 500)

      return `
<div id="${rootId}">
${populationStyle(rootId)}
  <div class="pg-head">
    <div class="pg-title">📈 种群数量变化模型</div>
    <div class="pg-note">相对教学模型，不用于真实种群预测</div>
  </div>

  <div class="pg-body">
    <div class="pg-controls">
      <div class="pg-row">
        <div class="pg-label">
          <span>初始种群数量</span>
          <span class="pg-value" data-initial-value></span>
        </div>
        <input
          data-initial
          type="range"
          min="10"
          max="150"
          step="5"
          value="${n(initialPopulation)}"
        >
      </div>

      <div class="pg-row">
        <div class="pg-label">
          <span>相对出生率</span>
          <span class="pg-value" data-birth-value></span>
        </div>
        <input
          data-birth
          type="range"
          min="0"
          max="50"
          step="1"
          value="${n(birthRate)}"
        >
      </div>

      <div class="pg-row">
        <div class="pg-label">
          <span>相对死亡率</span>
          <span class="pg-value" data-death-value></span>
        </div>
        <input
          data-death
          type="range"
          min="0"
          max="40"
          step="1"
          value="${n(deathRate)}"
        >
      </div>

      <div class="pg-row">
        <div class="pg-label">
          <span>环境容纳量K</span>
          <span class="pg-value" data-capacity-value></span>
        </div>
        <input
          data-capacity
          type="range"
          min="100"
          max="1000"
          step="50"
          value="${n(carryingCapacity)}"
        >
      </div>

      <div class="pg-subtitle">增长模式</div>

      <div class="pg-buttons">
        <button
          type="button"
          class="pg-button active"
          data-mode="exponential"
        >J型增长</button>

        <button
          type="button"
          class="pg-button"
          data-mode="logistic"
        >S型增长</button>
      </div>

      <div class="pg-status">
        <div class="pg-card">
          <b data-net-rate></b>
          <span>净增长率</span>
        </div>

        <div class="pg-card">
          <b data-final-population></b>
          <span>末期种群数量</span>
        </div>
      </div>

      <div class="pg-result" data-result></div>
    </div>

    <div class="pg-stage">
      <svg viewBox="0 0 680 414" aria-label="种群数量变化互动曲线">
        <defs>
          <linearGradient id="${rootId}-area" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#34D399" stop-opacity=".42"/>
            <stop offset="100%" stop-color="#34D399" stop-opacity=".04"/>
          </linearGradient>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#065F46"
              flood-opacity=".12"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="28"
          y="38"
          data-title
          font-size="27"
          font-weight="900"
          fill="#065F46"
        ></text>

        <text
          x="28"
          y="68"
          data-summary
          font-size="15"
          font-weight="800"
          fill="#475569"
        ></text>

        <line
          x1="74"
          y1="335"
          x2="630"
          y2="335"
          stroke="#64748B"
          stroke-width="3"
        />

        <line
          x1="74"
          y1="335"
          x2="74"
          y2="92"
          stroke="#64748B"
          stroke-width="3"
        />

        <text
          x="635"
          y="340"
          font-size="13"
          font-weight="800"
          fill="#475569"
        >时间</text>

        <text
          x="30"
          y="103"
          font-size="13"
          font-weight="800"
          fill="#475569"
        >种群数量</text>

        <g data-grid></g>

        <line
          data-capacity-line
          x1="74"
          y1="180"
          x2="630"
          y2="180"
          stroke="#F59E0B"
          stroke-width="3"
          stroke-dasharray="9 7"
        />

        <text
          data-capacity-label
          x="535"
          y="169"
          font-size="13"
          font-weight="900"
          fill="#B45309"
        ></text>

        <path
          data-area
          fill="url(#${rootId}-area)"
          opacity=".9"
        ></path>

        <path
          data-curve
          fill="none"
          stroke="#059669"
          stroke-width="6"
          stroke-linecap="round"
          stroke-linejoin="round"
          filter="url(#${rootId}-shadow)"
        ></path>

        <g data-points></g>

        <g transform="translate(76 374)">
          <circle cx="7" cy="7" r="7" fill="#059669"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            模拟种群数量
          </text>
        </g>

        <g transform="translate(248 374)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            环境容纳量K
          </text>
        </g>

        <text
          x="438"
          y="386"
          data-phase-note
          font-size="13"
          font-weight="900"
          fill="#047857"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var initial=root.querySelector('[data-initial]');
    var birth=root.querySelector('[data-birth]');
    var death=root.querySelector('[data-death]');
    var capacity=root.querySelector('[data-capacity]');

    var initialValue=root.querySelector('[data-initial-value]');
    var birthValue=root.querySelector('[data-birth-value]');
    var deathValue=root.querySelector('[data-death-value]');
    var capacityValue=root.querySelector('[data-capacity-value]');

    var buttons=root.querySelectorAll('[data-mode]');
    var netRate=root.querySelector('[data-net-rate]');
    var finalPopulation=root.querySelector('[data-final-population]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var grid=root.querySelector('[data-grid]');
    var capacityLine=root.querySelector('[data-capacity-line]');
    var capacityLabel=root.querySelector('[data-capacity-label]');
    var area=root.querySelector('[data-area]');
    var curve=root.querySelector('[data-curve]');
    var points=root.querySelector('[data-points]');
    var phaseNote=root.querySelector('[data-phase-note]');

    var mode='exponential';
    var steps=24;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function calculateValues(start,r,k,type){
      var values=[start];

      for(var i=1;i<=steps;i++){
        var previous=values[i-1];
        var next;

        if(type==='exponential'){
          next=previous*(1+r);
        }else{
          next=previous+r*previous*(1-previous/k);
        }

        values.push(Math.max(0,next));
      }

      return values;
    }

    function update(){
      var start=Number(initial.value);
      var birthRate=Number(birth.value)/100;
      var deathRate=Number(death.value)/100;
      var k=Number(capacity.value);
      var r=birthRate-deathRate;

      var values=calculateValues(start,r,k,mode);
      var finalValue=values[values.length-1];

      initialValue.textContent=start.toFixed(0);
      birthValue.textContent=(birthRate*100).toFixed(0)+'%';
      deathValue.textContent=(deathRate*100).toFixed(0)+'%';
      capacityValue.textContent=k.toFixed(0);

      netRate.textContent=(r*100).toFixed(0)+'%';
      finalPopulation.textContent=finalValue>=1000
        ?finalValue.toExponential(1)
        :finalValue.toFixed(0);

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-mode')===mode
        );
      }

      var maxValue=Math.max.apply(null,values.concat([k]));
      maxValue=Math.max(maxValue,100)*1.12;

      var left=74;
      var right=630;
      var top=92;
      var bottom=335;
      var width=right-left;
      var height=bottom-top;

      function x(index){
        return left+width*index/steps;
      }

      function y(value){
        return bottom-height*clamp(value/maxValue,0,1);
      }

      var gridHTML='';

      for(var g=0;g<=4;g++){
        var gy=bottom-height*g/4;
        var label=maxValue*g/4;

        gridHTML+='<line x1="'+left+'" y1="'+gy
          +'" x2="'+right+'" y2="'+gy
          +'" stroke="#E2E8F0" stroke-width="1.5"/>';

        gridHTML+='<text x="'+(left-10)+'" y="'+(gy+5)
          +'" text-anchor="end" font-size="11" font-weight="700" fill="#64748B">'
          +(label>=1000?label.toExponential(1):label.toFixed(0))
          +'</text>';
      }

      for(var t=0;t<=steps;t+=4){
        var tx=x(t);

        gridHTML+='<line x1="'+tx+'" y1="'+bottom
          +'" x2="'+tx+'" y2="'+top
          +'" stroke="#F1F5F9" stroke-width="1"/>';

        gridHTML+='<text x="'+tx+'" y="'+(bottom+20)
          +'" text-anchor="middle" font-size="11" font-weight="700" fill="#64748B">'
          +t+'</text>';
      }

      grid.innerHTML=gridHTML;

      var path='';
      var pointHTML='';

      for(var p=0;p<values.length;p++){
        var px=x(p);
        var py=y(values[p]);

        path+=(p===0?'M':' L')+px+' '+py;

        if(p%3===0 || p===values.length-1){
          pointHTML+='<circle cx="'+px+'" cy="'+py
            +'" r="4.5" fill="#FFFFFF" stroke="#059669" stroke-width="3"/>';
        }
      }

      curve.setAttribute('d',path);
      points.innerHTML=pointHTML;

      area.setAttribute(
        'd',
        path+' L'+right+' '+bottom+' L'+left+' '+bottom+' Z'
      );

      var capacityY=y(k);

      capacityLine.setAttribute('y1',String(capacityY));
      capacityLine.setAttribute('y2',String(capacityY));
      capacityLine.setAttribute(
        'opacity',
        mode==='logistic'?'1':'.28'
      );

      capacityLabel.setAttribute(
        'y',
        String(Math.max(top+15,capacityY-9))
      );

      capacityLabel.textContent='环境容纳量 K = '+k.toFixed(0);
      capacityLabel.setAttribute(
        'opacity',
        mode==='logistic'?'1':'.38'
      );

      if(mode==='exponential'){
        title.textContent='J型增长：理想条件下的指数增长';
        summary.textContent='资源和空间近似无限，种群增长不受环境容纳量限制';
      }else{
        title.textContent='S型增长：环境阻力限制种群增长';
        summary.textContent='随着种群密度增大，资源和空间限制逐渐增强';
      }

      var condition='';
      var note='';

      if(r<0){
        condition='死亡率高于出生率，种群数量持续下降。';
        note='种群衰退';
      }else if(r===0){
        condition='出生率与死亡率相等，本模型中种群数量基本保持稳定。';
        note='相对稳定';
      }else if(mode==='exponential'){
        condition='净增长率为正，在理想条件下种群数量呈指数增加。';
        note='指数增长';
      }else if(start>k){
        condition='初始种群数量超过环境容纳量，环境阻力使数量逐渐下降。';
        note='超过K值';
      }else if(finalValue>k*.9){
        condition='种群数量逐渐接近环境容纳量，增长速度趋于减慢。';
        note='接近K值';
      }else{
        condition='种群数量仍低于环境容纳量，当前处于增长阶段。';
        note='增长阶段';
      }

      phaseNote.textContent=note;

      result.innerHTML=mode==='exponential'
        ?'J型增长只适用于资源、空间等条件近似无限的理想情境。'
          +'<br>'+condition
          +' 真实种群不可能长期保持无限指数增长。'
        :'S型增长体现了环境阻力和环境容纳量对种群增长的限制。'
          +'<br>'+condition
          +' 真实种群通常会在K值附近发生波动，而不是形成绝对平滑曲线。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    initial.oninput=update;
    birth.oninput=update;
    death.oninput=update;
    capacity.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
