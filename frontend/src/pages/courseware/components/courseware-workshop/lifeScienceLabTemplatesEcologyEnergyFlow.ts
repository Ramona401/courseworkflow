/**
 * lifeScienceLabTemplatesEcologyEnergyFlow.ts
 *
 * 平面生命科学实验室：生态系统能量流动。
 *
 * 教学边界：
 * 1. 能量从生产者沿食物链向不同营养级单向流动；
 * 2. 每一营养级均有能量用于生命活动并以热等形式散失；
 * 3. 传递效率不是固定10%，本模型允许在5%—20%范围内比较；
 * 4. 所有能量数值均为相对教学单位，不代表真实生态测量值。
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

function energyStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FDE68A;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .ef-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FEF3C7,#FFFBEB);border-bottom:1px solid #FDE68A}'
    + '#' + rootId + ' .ef-title{font-size:15px;font-weight:800;color:#92400E}'
    + '#' + rootId + ' .ef-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .ef-body{height:calc(100% - 46px);display:grid;grid-template-columns:230px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .ef-controls{padding:13px;overflow:auto;background:#FFFCF5;border-right:1px solid #FDE68A}'
    + '#' + rootId + ' .ef-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .ef-row{margin-bottom:12px}'
    + '#' + rootId + ' .ef-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .ef-value{font-weight:800;color:#D97706;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#F59E0B}'
    + '#' + rootId + ' .ef-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#92400E}'
    + '#' + rootId + ' .ef-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:10px}'
    + '#' + rootId + ' .ef-button{height:32px;border:1px solid #FCD34D;border-radius:8px;background:#fff;color:#92400E;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ef-button.active{border-color:#F59E0B;background:#FEF3C7;box-shadow:0 3px 9px rgba(245,158,11,.15)}'
    + '#' + rootId + ' .ef-result{padding:9px 10px;border-radius:10px;background:#FEF3C7;color:#78350F;font-size:11.5px;line-height:1.52;font-weight:600}'
    + '#' + rootId + ' .ef-legend{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin:9px 0}'
    + '#' + rootId + ' .ef-key{display:flex;align-items:center;gap:5px;font-size:10.5px;color:#475569}'
    + '#' + rootId + ' .ef-dot{width:9px;height:9px;border-radius:50%;flex:none}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ef-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.7s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_ENERGY_FLOW:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-ecosystem-energy-flow',
    group: '🌎 生态系统',
    name: '生态系统能量流动',
    emoji: '🔋',
    desc: '调节生产者能量、营养级数量和传递效率，观察能量金字塔及逐级递减',
    params: [
      {
        key: 'producerEnergy',
        label: '生产者能量',
        type: 'number',
        min: 500,
        max: 5000,
        step: 100,
        defaultValue: 3000,
      },
      {
        key: 'transferEfficiency',
        label: '营养级传递效率/%',
        type: 'number',
        min: 5,
        max: 20,
        step: 1,
        defaultValue: 10,
      },
      {
        key: 'trophicLevels',
        label: '营养级数量',
        type: 'number',
        min: 3,
        max: 5,
        step: 1,
        defaultValue: 4,
      },
    ],

    buildHTML: (params, rootId) => {
      const producerEnergy = num(params, 'producerEnergy', 3000)
      const transferEfficiency = num(params, 'transferEfficiency', 10)
      const trophicLevels = num(params, 'trophicLevels', 4)

      return `
<div id="${rootId}">
${energyStyle(rootId)}
  <div class="ef-head">
    <div class="ef-title">🔋 生态系统能量流动</div>
    <div class="ef-note">能量值为相对教学单位，传递效率并非固定10%</div>
  </div>

  <div class="ef-body">
    <div class="ef-controls">
      <div class="ef-row">
        <div class="ef-label">
          <span>生产者能量</span>
          <span class="ef-value" data-producer-value></span>
        </div>
        <input
          data-producer
          type="range"
          min="500"
          max="5000"
          step="100"
          value="${n(producerEnergy)}"
        >
      </div>

      <div class="ef-row">
        <div class="ef-label">
          <span>营养级传递效率</span>
          <span class="ef-value" data-efficiency-value></span>
        </div>
        <input
          data-efficiency
          type="range"
          min="5"
          max="20"
          step="1"
          value="${n(transferEfficiency)}"
        >
      </div>

      <div class="ef-row">
        <div class="ef-label">
          <span>营养级数量</span>
          <span class="ef-value" data-level-value></span>
        </div>
        <input
          data-levels
          type="range"
          min="3"
          max="5"
          step="1"
          value="${n(trophicLevels)}"
        >
      </div>

      <div class="ef-subtitle">观察模式</div>

      <div class="ef-buttons">
        <button type="button" class="ef-button active" data-mode="pyramid">能量金字塔</button>
        <button type="button" class="ef-button" data-mode="flow">能量流动</button>
      </div>

      <div class="ef-legend">
        <div class="ef-key">
          <span class="ef-dot" style="background:#22C55E"></span>
          输入下一营养级
        </div>
        <div class="ef-key">
          <span class="ef-dot" style="background:#F97316"></span>
          呼吸与热散失
        </div>
      </div>

      <div class="ef-result" data-result></div>
    </div>

    <div class="ef-stage">
      <svg viewBox="0 0 680 414" aria-label="生态系统能量流动互动示意图">
        <defs>
          <marker id="${rootId}-green-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#22C55E"/>
          </marker>

          <marker id="${rootId}-heat-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F97316"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#92400E" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#92400E"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <g data-graphic filter="url(#${rootId}-shadow)"></g>

        <g transform="translate(28 367)">
          <text x="0" y="0" font-size="13" font-weight="800" fill="#475569">进入最高营养级</text>
          <rect x="112" y="-13" width="180" height="16" rx="8" fill="#E2E8F0"/>
          <rect data-top-bar x="112" y="-13" width="0" height="16" rx="8" fill="#EF4444"/>
          <text x="306" y="0" data-top-value font-size="13" font-weight="900" fill="#B91C1C"></text>
        </g>

        <text x="470" y="367" data-loss-value font-size="14" font-weight="900" fill="#C2410C"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var producer=root.querySelector('[data-producer]');
    var efficiency=root.querySelector('[data-efficiency]');
    var levels=root.querySelector('[data-levels]');

    var producerValue=root.querySelector('[data-producer-value]');
    var efficiencyValue=root.querySelector('[data-efficiency-value]');
    var levelValue=root.querySelector('[data-level-value]');

    var buttons=root.querySelectorAll('[data-mode]');
    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var graphic=root.querySelector('[data-graphic]');
    var topBar=root.querySelector('[data-top-bar]');
    var topValue=root.querySelector('[data-top-value]');
    var lossValue=root.querySelector('[data-loss-value]');
    var result=root.querySelector('[data-result]');

    var mode='pyramid';

    var names=[
      '生产者',
      '初级消费者',
      '次级消费者',
      '三级消费者',
      '顶级消费者'
    ];

    var colors=[
      '#22C55E',
      '#84CC16',
      '#F59E0B',
      '#F97316',
      '#EF4444'
    ];

    var emojis=[
      '🌿',
      '🐇',
      '🐸',
      '🐍',
      '🦅'
    ];

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function energyValues(start,rate,count){
      var values=[start];

      for(var i=1;i<count;i++){
        values.push(values[i-1]*rate);
      }

      return values;
    }

    function renderPyramid(values,count){
      var html='';
      var baseY=334;
      var levelHeight=52;
      var maxWidth=490;

      for(var i=0;i<count;i++){
        var value=values[i];
        var width=maxWidth*Math.pow(.68,i);
        var x=340-width/2;
        var y=baseY-(i+1)*levelHeight;
        var label=value>=10
          ?value.toFixed(0)
          :value.toFixed(2);

        html+='<rect x="'+x+'" y="'+y+'" width="'+width
          +'" height="'+(levelHeight-7)+'" rx="11" fill="'+colors[i]
          +'" opacity="'+(.92-i*.07)+'"/>';

        html+='<text x="340" y="'+(y+21)
          +'" text-anchor="middle" font-size="16" font-weight="900" fill="#FFFFFF">'
          +emojis[i]+' '+names[i]+'</text>';

        html+='<text x="340" y="'+(y+39)
          +'" text-anchor="middle" font-size="12" font-weight="800" fill="#FFFFFF">'
          +'相对能量 '+label+'</text>';

        if(i<count-1){
          html+='<path class="ef-flow" d="M340 '+(y-4)+' V'+(y-29)
            +'" fill="none" stroke="#22C55E" stroke-width="4'
            +'" marker-end="url(#${rootId}-green-arrow)"/>';
        }

        html+='<path d="M'+(x+width+5)+' '+(y+20)
          +' H'+Math.min(632,x+width+72)
          +'" fill="none" stroke="#F97316" stroke-width="3'
          +'" marker-end="url(#${rootId}-heat-arrow)" opacity=".75"/>';

        html+='<text x="'+Math.min(638,x+width+78)
          +'" y="'+(y+25)+'" font-size="11" font-weight="800" fill="#C2410C">热散失</text>';
      }

      return html;
    }

    function renderFlow(values,count,rate){
      var html='';
      var startX=78;
      var gap=count===5?122:145;
      var y=212;

      for(var i=0;i<count;i++){
        var x=startX+i*gap;
        var value=values[i];
        var label=value>=10
          ?value.toFixed(0)
          :value.toFixed(2);
        var radius=clamp(39-i*4,22,39);

        html+='<circle cx="'+x+'" cy="'+y+'" r="'+radius
          +'" fill="'+colors[i]+'" opacity="'+(.92-i*.06)+'"/>';

        html+='<text x="'+x+'" y="'+(y-3)
          +'" text-anchor="middle" font-size="24">'+emojis[i]+'</text>';

        html+='<text x="'+x+'" y="'+(y+20)
          +'" text-anchor="middle" font-size="11" font-weight="900" fill="#FFFFFF">'
          +names[i]+'</text>';

        html+='<text x="'+x+'" y="'+(y+67)
          +'" text-anchor="middle" font-size="13" font-weight="900" fill="'+colors[i]+'">'
          +label+'</text>';

        html+='<path d="M'+x+' '+(y-radius-5)
          +' V'+(y-radius-53)
          +'" fill="none" stroke="#F97316" stroke-width="3'
          +'" marker-end="url(#${rootId}-heat-arrow)"/>';

        html+='<text x="'+(x-22)+'" y="'+(y-radius-60)
          +'" font-size="10.5" font-weight="800" fill="#C2410C">散失 '
          +(100-rate*100).toFixed(0)+'%</text>';

        if(i<count-1){
          var nextX=startX+(i+1)*gap;

          html+='<path class="ef-flow" d="M'+(x+radius+5)+' '+y
            +' H'+(nextX-radius-10)
            +'" fill="none" stroke="#22C55E" stroke-width="5'
            +'" marker-end="url(#${rootId}-green-arrow)"/>';

          html+='<text x="'+((x+nextX)/2)
            +'" y="'+(y-13)
            +'" text-anchor="middle" font-size="11" font-weight="900" fill="#15803D">'
            +(rate*100).toFixed(0)+'%</text>';
        }
      }

      html+='<text x="340" y="330" text-anchor="middle" font-size="15" font-weight="900" fill="#475569">'
        +'能量沿食物链单向流动，并在每个营养级逐步减少'
        +'</text>';

      return html;
    }

    function update(){
      var start=Number(producer.value);
      var rate=Number(efficiency.value)/100;
      var count=Math.round(Number(levels.value));
      var values=energyValues(start,rate,count);
      var top=values[count-1];

      producerValue.textContent=start.toFixed(0)+' 单位';
      efficiencyValue.textContent=(rate*100).toFixed(0)+'%';
      levelValue.textContent=count.toFixed(0)+'级';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-mode')===mode
        );
      }

      if(mode==='pyramid'){
        title.textContent='生态系统能量金字塔';
        summary.textContent='营养级越高，可利用的能量通常越少';
        graphic.innerHTML=renderPyramid(values,count);
      }else{
        title.textContent='能量沿营养级单向流动';
        summary.textContent='一部分能量进入下一营养级，其余用于生命活动并以热等形式散失';
        graphic.innerHTML=renderFlow(values,count,rate);
      }

      var topRatio=top/start*100;
      var totalTransferred=0;

      for(var j=1;j<values.length;j++){
        totalTransferred+=values[j];
      }

      var totalLoss=start+totalTransferred-top;
      var visibleLoss=Math.max(0,start-top);

      topBar.setAttribute(
        'width',
        String(180*clamp(topRatio/20,0,1))
      );

      topValue.textContent=top>=10
        ?top.toFixed(0)+' 单位'
        :top.toFixed(2)+' 单位';

      lossValue.textContent='由生产者到最高营养级累计减少 '
        +(100-topRatio).toFixed(1)+'%';

      var condition='当前传递效率处于常见教学比较范围。';

      if(rate<=.07){
        condition='传递效率较低，高营养级可利用的能量迅速减少。';
      }else if(rate>=.17){
        condition='传递效率设置较高，但仍有大量能量未进入下一营养级。';
      }

      result.innerHTML='能量流动通常具有单向流动、逐级递减的特点。'
        +'<br>'+condition
        +' 当前最高营养级约获得生产者能量的 '
        +topRatio.toFixed(2)+'%。'
        +' 减少的能量并非全部消失，其中包括呼吸散热、未被取食和未被同化等部分。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    producer.oninput=update;
    efficiency.oninput=update;
    levels.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
