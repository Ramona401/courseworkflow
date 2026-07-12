/**
 * lifeScienceLabTemplatesEcologyFoodWeb.ts
 *
 * 平面生命科学实验室：食物链与食物网。
 *
 * 教学边界：
 * 1. 箭头由被捕食者指向捕食者，表示物质和能量流动方向；
 * 2. 食物网由多条食物链相互交织形成；
 * 3. 生物数量和稳定度均为相对教学指标；
 * 4. 图中物种和关系为简化示意，不对应某一真实生态调查。
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

function foodWebStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BBF7D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .fw-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#F0FDF4);border-bottom:1px solid #BBF7D0}'
    + '#' + rootId + ' .fw-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .fw-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .fw-body{height:calc(100% - 46px);display:grid;grid-template-columns:230px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .fw-controls{padding:13px;overflow:auto;background:#F8FFF9;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .fw-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .fw-row{margin-bottom:12px}'
    + '#' + rootId + ' .fw-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .fw-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#16A34A}'
    + '#' + rootId + ' .fw-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .fw-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:10px}'
    + '#' + rootId + ' .fw-button{height:32px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .fw-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.13)}'
    + '#' + rootId + ' .fw-result{padding:9px 10px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:11.5px;line-height:1.52;font-weight:600}'
    + '#' + rootId + ' .fw-legend{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin:9px 0}'
    + '#' + rootId + ' .fw-key{font-size:10.5px;color:#475569;display:flex;align-items:center;gap:5px}'
    + '#' + rootId + ' .fw-dot{width:9px;height:9px;border-radius:50%;flex:none}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .fw-link{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.8s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_FOOD_WEB:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-food-chain-web',
    group: '🌎 生态系统',
    name: '食物链与食物网',
    emoji: '🕸️',
    desc: '切换食物链和食物网，调节生产者、初级消费者和分解者状态，观察营养关系变化',
    params: [
      {
        key: 'producerLevel',
        label: '生产者丰富度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'herbivoreLevel',
        label: '初级消费者数量',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'decomposerLevel',
        label: '分解者活跃度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 70,
      },
    ],

    buildHTML: (params, rootId) => {
      const producerLevel = num(params, 'producerLevel', 78)
      const herbivoreLevel = num(params, 'herbivoreLevel', 62)
      const decomposerLevel = num(params, 'decomposerLevel', 70)

      return `
<div id="${rootId}">
${foodWebStyle(rootId)}
  <div class="fw-head">
    <div class="fw-title">🕸️ 食物链与食物网</div>
    <div class="fw-note">箭头表示物质和能量由食物指向取食者</div>
  </div>

  <div class="fw-body">
    <div class="fw-controls">
      <div class="fw-row">
        <div class="fw-label">
          <span>生产者丰富度</span>
          <span class="fw-value" data-producer-value></span>
        </div>
        <input data-producer type="range" min="20" max="100" step="1" value="${n(producerLevel)}">
      </div>

      <div class="fw-row">
        <div class="fw-label">
          <span>初级消费者数量</span>
          <span class="fw-value" data-herbivore-value></span>
        </div>
        <input data-herbivore type="range" min="10" max="100" step="1" value="${n(herbivoreLevel)}">
      </div>

      <div class="fw-row">
        <div class="fw-label">
          <span>分解者活跃度</span>
          <span class="fw-value" data-decomposer-value></span>
        </div>
        <input data-decomposer type="range" min="10" max="100" step="1" value="${n(decomposerLevel)}">
      </div>

      <div class="fw-subtitle">观察模式</div>

      <div class="fw-buttons">
        <button type="button" class="fw-button active" data-mode="chain">食物链</button>
        <button type="button" class="fw-button" data-mode="web">食物网</button>
      </div>

      <div class="fw-legend">
        <div class="fw-key"><span class="fw-dot" style="background:#22C55E"></span>生产者</div>
        <div class="fw-key"><span class="fw-dot" style="background:#F59E0B"></span>消费者</div>
        <div class="fw-key"><span class="fw-dot" style="background:#7C3AED"></span>分解者</div>
        <div class="fw-key"><span class="fw-dot" style="background:#0EA5E9"></span>能量流向</div>
      </div>

      <div class="fw-result" data-result></div>
    </div>

    <div class="fw-stage">
      <svg viewBox="0 0 680 414" aria-label="食物链与食物网互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0EA5E9"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#14532D" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#166534"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <g data-links></g>
        <g data-nodes filter="url(#${rootId}-shadow)"></g>

        <g transform="translate(28 367)">
          <text x="0" y="0" font-size="13" font-weight="800" fill="#475569">网络稳定度</text>
          <rect x="92" y="-13" width="175" height="16" rx="8" fill="#E2E8F0"/>
          <rect data-stability-bar x="92" y="-13" width="0" height="16" rx="8" fill="#22C55E"/>
          <text x="280" y="0" data-stability-text font-size="13" font-weight="900" fill="#166534"></text>
        </g>

        <text x="432" y="367" data-chain-count font-size="14" font-weight="900" fill="#0369A1"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var producer=root.querySelector('[data-producer]');
    var herbivore=root.querySelector('[data-herbivore]');
    var decomposer=root.querySelector('[data-decomposer]');

    var producerValue=root.querySelector('[data-producer-value]');
    var herbivoreValue=root.querySelector('[data-herbivore-value]');
    var decomposerValue=root.querySelector('[data-decomposer-value]');

    var buttons=root.querySelectorAll('[data-mode]');
    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var links=root.querySelector('[data-links]');
    var nodes=root.querySelector('[data-nodes]');
    var stabilityBar=root.querySelector('[data-stability-bar]');
    var stabilityText=root.querySelector('[data-stability-text]');
    var chainCount=root.querySelector('[data-chain-count]');
    var result=root.querySelector('[data-result]');

    var mode='chain';

    var species={
      grass:{x:94,y:282,label:'草',emoji:'🌱',color:'#22C55E',type:'producer'},
      shrub:{x:94,y:156,label:'灌木',emoji:'🌿',color:'#16A34A',type:'producer'},
      grasshopper:{x:244,y:294,label:'蝗虫',emoji:'🦗',color:'#F59E0B',type:'consumer'},
      rabbit:{x:244,y:146,label:'兔',emoji:'🐇',color:'#F97316',type:'consumer'},
      mouse:{x:370,y:330,label:'田鼠',emoji:'🐁',color:'#F59E0B',type:'consumer'},
      frog:{x:370,y:238,label:'青蛙',emoji:'🐸',color:'#84CC16',type:'consumer'},
      snake:{x:493,y:218,label:'蛇',emoji:'🐍',color:'#EF4444',type:'consumer'},
      hawk:{x:594,y:121,label:'鹰',emoji:'🦅',color:'#B91C1C',type:'consumer'},
      decomposer:{x:544,y:326,label:'分解者',emoji:'🍄',color:'#7C3AED',type:'decomposer'}
    };

    var chainLinks=[
      ['grass','grasshopper'],
      ['grasshopper','frog'],
      ['frog','snake'],
      ['snake','hawk']
    ];

    var webLinks=[
      ['grass','grasshopper'],
      ['grass','rabbit'],
      ['grass','mouse'],
      ['shrub','rabbit'],
      ['shrub','mouse'],
      ['grasshopper','frog'],
      ['grasshopper','mouse'],
      ['frog','snake'],
      ['mouse','snake'],
      ['rabbit','hawk'],
      ['mouse','hawk'],
      ['snake','hawk'],
      ['grass','decomposer'],
      ['shrub','decomposer'],
      ['grasshopper','decomposer'],
      ['rabbit','decomposer'],
      ['frog','decomposer'],
      ['mouse','decomposer'],
      ['snake','decomposer'],
      ['hawk','decomposer']
    ];

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function node(id,opacity,scale){
      var item=species[id];

      return '<g transform="translate('+item.x+' '+item.y+') scale('+scale+')">'
        +'<circle cx="0" cy="0" r="38" fill="#FFFFFF" stroke="'+item.color
        +'" stroke-width="5" opacity="'+opacity+'"/>'
        +'<circle cx="0" cy="0" r="31" fill="'+item.color+'" opacity="'+(.12+.18*opacity)+'"/>'
        +'<text x="0" y="7" text-anchor="middle" font-size="27">'+item.emoji+'</text>'
        +'<rect x="-43" y="42" width="86" height="25" rx="12" fill="#FFFFFF" stroke="'+item.color+'" stroke-width="2"/>'
        +'<text x="0" y="59" text-anchor="middle" font-size="13" font-weight="900" fill="'+item.color+'">'+item.label+'</text>'
        +'</g>';
    }

    function link(fromId,toId,opacity,width){
      var from=species[fromId];
      var to=species[toId];
      var dx=to.x-from.x;
      var dy=to.y-from.y;
      var length=Math.sqrt(dx*dx+dy*dy) || 1;
      var startX=from.x+dx/length*42;
      var startY=from.y+dy/length*42;
      var endX=to.x-dx/length*46;
      var endY=to.y-dy/length*46;

      return '<path class="fw-link" d="M'+startX+' '+startY+' L'+endX+' '+endY
        +'" fill="none" stroke="#0EA5E9" stroke-width="'+width
        +'" opacity="'+opacity+'" marker-end="url(#${rootId}-arrow)"/>';
    }

    function update(){
      var p=Number(producer.value);
      var h=Number(herbivore.value);
      var d=Number(decomposer.value);

      producerValue.textContent=p.toFixed(0)+'%';
      herbivoreValue.textContent=h.toFixed(0)+'%';
      decomposerValue.textContent=d.toFixed(0)+'%';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-mode')===mode
        );
      }

      var activeLinks=mode==='chain'?chainLinks:webLinks;
      var activeSpecies={};

      for(var l=0;l<activeLinks.length;l++){
        activeSpecies[activeLinks[l][0]]=true;
        activeSpecies[activeLinks[l][1]]=true;
      }

      var producerFactor=p/100;
      var herbivoreFactor=h/100;
      var decomposerFactor=d/100;

      var linkHTML='';

      for(var k=0;k<activeLinks.length;k++){
        var from=species[activeLinks[k][0]];
        var to=species[activeLinks[k][1]];
        var strength=1;

        if(from.type==='producer'){
          strength*=producerFactor;
        }

        if(to.id==='grasshopper'||to.id==='rabbit'||to.id==='mouse'){
          strength*=herbivoreFactor;
        }

        if(to.type==='decomposer'){
          strength*=decomposerFactor;
        }

        linkHTML+=link(
          activeLinks[k][0],
          activeLinks[k][1],
          .28+.72*strength,
          2.5+4*strength
        );
      }

      links.innerHTML=linkHTML;

      var nodeHTML='';

      for(var id in species){
        if(!activeSpecies[id]){
          continue;
        }

        var item=species[id];
        var abundance=.72;

        if(item.type==='producer'){
          abundance=producerFactor;
        }else if(id==='grasshopper'||id==='rabbit'||id==='mouse'){
          abundance=herbivoreFactor;
        }else if(item.type==='decomposer'){
          abundance=decomposerFactor;
        }else{
          abundance=clamp((producerFactor+herbivoreFactor)/2,.25,1);
        }

        nodeHTML+=node(
          id,
          .4+.6*abundance,
          .86+.16*abundance
        );
      }

      nodes.innerHTML=nodeHTML;

      var diversity=mode==='chain'?42:82;
      var balance=100-Math.abs(p-h)*.55-Math.abs(d-65)*.22;
      var stability=clamp(
        diversity*.45+balance*.55,
        10,
        100
      );

      stabilityBar.setAttribute(
        'width',
        String(175*stability/100)
      );

      stabilityText.textContent=stability.toFixed(0)+'%';
      chainCount.textContent=mode==='chain'
        ?'当前展示1条食物链'
        :'当前网络包含多条相互交织的食物链';

      title.textContent=mode==='chain'
        ?'一条典型食物链'
        :'多条食物链构成食物网';

      summary.textContent=mode==='chain'
        ?'草 → 蝗虫 → 青蛙 → 蛇 → 鹰'
        :'同一种生物可能取食多种食物，也可能被多种生物捕食';

      var condition='当前生产者、消费者和分解者之间相对协调。';

      if(p<35){
        condition='生产者较少，进入生态系统的有机物和能量基础减弱。';
      }else if(h>p+20){
        condition='初级消费者相对过多，生产者承受的取食压力较大。';
      }else if(h<25){
        condition='初级消费者较少，部分捕食者的食物来源可能不足。';
      }else if(d<25){
        condition='分解者活跃度较低，遗体和排遗物分解及物质循环受到影响。';
      }

      result.innerHTML='箭头由被捕食者指向捕食者，表示物质和能量流动方向。'
        +'<br>'+condition
        +(mode==='web'
          ?' 食物网中的营养联系通常比单一食物链更复杂。'
          :' 单条食物链只能表示生态系统中的一部分营养关系。');
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    producer.oninput=update;
    herbivore.oninput=update;
    decomposer.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
