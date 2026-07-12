/**
 * lifeScienceLabTemplatesExt.ts — 生命科学实验室第一批扩展模板
 *
 * 本批覆盖生命过程：
 *   1. 光合作用：光照、二氧化碳、温度和水分共同影响相对速率。
 *   2. 细胞呼吸：有机物、氧气和温度共同影响有氧呼吸相对速率。
 *   3. 蒸腾作用：气孔开放度、温度、湿度和风速共同影响相对速率。
 *
 * 说明：
 *   - 三个模板均为教学示意模型，不把相对速率伪装成真实实验测量值。
 *   - 全部使用纯 HTML + SVG + 原生 JavaScript，可离线运行。
 *   - 所有 DOM 查询均限定在 rootId 内，支持同页放置多个组件。
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

function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

function baseStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #D1FAE5;border-radius:16px;background:#FFFFFF;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' .bl-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#D1FAE5,#F0FDF4);border-bottom:1px solid #D1FAE5;box-sizing:border-box;}\n'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#065F46;}\n'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:220px 1fr;min-height:0;}\n'
    + '#' + rootId + ' .bl-controls{padding:14px;border-right:1px solid #D1FAE5;background:#F8FAFC;box-sizing:border-box;overflow:auto;}\n'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .bl-row{margin-bottom:13px;}\n'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;font-size:12px;font-weight:700;color:#334155;margin-bottom:6px;}\n'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#059669;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981;}\n'
    + '#' + rootId + ' button{border:none;border-radius:10px;padding:8px 12px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:12px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .bl-result{padding:9px 11px;border-radius:10px;background:#D1FAE5;color:#065F46;font-size:12px;line-height:1.55;font-weight:600;}\n'
    + '#' + rootId + ' svg{width:100%;height:100%;display:block;}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_EXT: LifeScienceLabTemplate[] = [
  {
    id: 'biology-photosynthesis',
    group: '🌿 植物生理',
    name: '光合作用',
    emoji: '☀️',
    desc: '调节光照、二氧化碳、温度和水分，观察限制因素与相对光合速率',
    params: [
      { key: 'light', label: '光照强度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 70 },
      { key: 'co2', label: 'CO₂浓度/ppm', type: 'number', min: 100, max: 1200, step: 50, defaultValue: 450 },
      { key: 'temperature', label: '温度/℃', type: 'number', min: 5, max: 45, step: 1, defaultValue: 25 },
      { key: 'water', label: '水分供应', type: 'number', min: 0, max: 100, step: 1, defaultValue: 75 },
    ],
    buildHTML: (params, rootId) => {
      const light = num(params, 'light', 70)
      const co2 = num(params, 'co2', 450)
      const temperature = num(params, 'temperature', 25)
      const water = num(params, 'water', 75)

      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">☀️ 光合作用条件与限制因素</div>
    <div class="bl-note">相对速率教学模型，不代表真实定量测量</div>
  </div>
  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row"><div class="bl-label"><span>光照强度</span><span class="bl-value" data-light-val></span></div><input data-light type="range" min="0" max="100" step="1" value="${n(light)}"></div>
      <div class="bl-row"><div class="bl-label"><span>CO₂浓度</span><span class="bl-value" data-co2-val></span></div><input data-co2 type="range" min="100" max="1200" step="50" value="${n(co2)}"></div>
      <div class="bl-row"><div class="bl-label"><span>温度</span><span class="bl-value" data-temp-val></span></div><input data-temp type="range" min="5" max="45" step="1" value="${n(temperature)}"></div>
      <div class="bl-row"><div class="bl-label"><span>水分供应</span><span class="bl-value" data-water-val></span></div><input data-water type="range" min="0" max="100" step="1" value="${n(water)}"></div>
      <div class="bl-result" data-result></div>
    </div>
    <div class="bl-stage">
      <svg viewBox="0 0 680 414">
        <defs>
          <linearGradient id="${rootId}-leaf" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stop-color="#86EFAC"/>
            <stop offset="1" stop-color="#16A34A"/>
          </linearGradient>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#64748B"/>
          </marker>
        </defs>
        <rect width="680" height="414" fill="#FFFFFF"/>
        <circle cx="92" cy="84" r="42" data-sun fill="#FACC15"/>
        <g data-rays stroke="#F59E0B" stroke-linecap="round"></g>
        <path d="M340 338 C338 286 340 242 344 198" stroke="#15803D" stroke-width="22" stroke-linecap="round"/>
        <path d="M344 225 C280 178 216 152 158 180 C206 250 286 272 344 225Z" data-leaf fill="url(#${rootId}-leaf)" stroke="#166534" stroke-width="5"/>
        <path d="M344 214 C416 154 510 154 558 200 C494 265 408 267 344 214Z" data-leaf fill="url(#${rootId}-leaf)" stroke="#166534" stroke-width="5"/>
        <path d="M180 182 C232 202 284 216 340 224 M526 198 C466 210 410 214 348 216" fill="none" stroke="#DCFCE7" stroke-width="4"/>
        <path d="M340 336 C310 360 280 374 244 382 M340 336 C368 360 406 376 448 384 M340 336 C338 366 338 386 338 402" fill="none" stroke="#92400E" stroke-width="9" stroke-linecap="round"/>
        <rect x="0" y="360" width="680" height="54" fill="#FEF3C7" opacity="0.75"/>
        <g data-water-flow></g>
        <g data-co2-flow></g>
        <g data-o2></g>
        <text x="76" y="150" data-light-text font-size="17" font-weight="900" fill="#B45309"></text>
        <text x="410" y="72" data-rate font-size="29" font-weight="900" fill="#065F46"></text>
        <text x="410" y="108" data-limit font-size="18" font-weight="800" fill="#475569"></text>
        <text x="382" y="332" font-size="16" font-weight="800" fill="#166534">水由根吸收并向上运输</text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var light=root.querySelector('[data-light]'),co2=root.querySelector('[data-co2]');
    var temp=root.querySelector('[data-temp]'),water=root.querySelector('[data-water]');
    var lightVal=root.querySelector('[data-light-val]'),co2Val=root.querySelector('[data-co2-val]');
    var tempVal=root.querySelector('[data-temp-val]'),waterVal=root.querySelector('[data-water-val]');
    var sun=root.querySelector('[data-sun]'),rays=root.querySelector('[data-rays]');
    var leaves=root.querySelectorAll('[data-leaf]'),waterFlow=root.querySelector('[data-water-flow]');
    var co2Flow=root.querySelector('[data-co2-flow]'),o2=root.querySelector('[data-o2]');
    var rateText=root.querySelector('[data-rate]'),limitText=root.querySelector('[data-limit]');
    var lightText=root.querySelector('[data-light-text]'),result=root.querySelector('[data-result]');

    function clamp(v,min,max){return Math.max(min,Math.min(max,v));}
    function update(){
      var L=Number(light.value),C=Number(co2.value),T=Number(temp.value),W=Number(water.value);
      var lf=1-Math.exp(-L/32);
      var cf=C/(C+300);
      var tf=Math.exp(-Math.pow((T-25)/13,2));
      var wf=clamp(W/65,0,1);
      var factors=[lf,cf,tf,wf],names=['光照','二氧化碳','温度','水分'];
      var limitIndex=0;
      for(var i=1;i<factors.length;i++){if(factors[i]<factors[limitIndex])limitIndex=i;}
      var rate=100*lf*cf*tf*wf;

      lightVal.textContent=L.toFixed(0)+'%';
      co2Val.textContent=C.toFixed(0)+' ppm';
      tempVal.textContent=T.toFixed(0)+'℃';
      waterVal.textContent=W.toFixed(0)+'%';
      rateText.textContent='相对光合速率 '+rate.toFixed(0);
      limitText.textContent='当前主要限制：'+names[limitIndex];
      lightText.textContent='光照 '+L.toFixed(0)+'%';
      sun.setAttribute('opacity',String(0.25+L/135));

      var rayHTML='';
      for(var r=0;r<Math.floor(2+L/14);r++){
        var angle=r*Math.PI/4;
        var x1=92+52*Math.cos(angle),y1=84+52*Math.sin(angle);
        var x2=92+(68+L*0.35)*Math.cos(angle),y2=84+(68+L*0.35)*Math.sin(angle);
        rayHTML+='<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2+'" stroke-width="'+(3+L/35)+'"/>';
      }
      rays.innerHTML=rayHTML;

      for(var j=0;j<leaves.length;j++){
        leaves[j].setAttribute('opacity',String(0.45+rate/180));
      }

      var waterHTML='';
      var waterCount=Math.floor(W/14);
      for(var k=0;k<waterCount;k++){
        var wy=352-k*21;
        waterHTML+='<circle cx="'+(340+(k%2?7:-7))+'" cy="'+wy+'" r="6" fill="#38BDF8" opacity="0.78"/>';
      }
      waterFlow.innerHTML=waterHTML;

      var co2HTML='';
      var co2Count=Math.floor(2+C/180);
      for(var m=0;m<co2Count;m++){
        var cy=118+m*27;
        co2HTML+='<text x="'+(584-(m%2)*28)+'" y="'+cy+'" font-size="14" font-weight="900" fill="#475569">CO₂</text>';
        co2HTML+='<path d="M'+(576-(m%2)*28)+' '+(cy+7)+' C540 '+(cy+4)+' 526 188 500 202" fill="none" stroke="#64748B" stroke-width="2.5" marker-end="url(#${rootId}-arrow)"/>';
      }
      co2Flow.innerHTML=co2HTML;

      var o2HTML='';
      var bubbleCount=Math.floor(rate/10);
      for(var q=0;q<bubbleCount;q++){
        var ox=184+(q%5)*74,oy=126-Math.floor(q/5)*30;
        o2HTML+='<circle cx="'+ox+'" cy="'+oy+'" r="'+(5+q%3)+'" fill="#BAE6FD" stroke="#0284C7" stroke-width="2"/>';
        if(q%2===0)o2HTML+='<text x="'+(ox+9)+'" y="'+(oy+5)+'" font-size="12" font-weight="900" fill="#0284C7">O₂</text>';
      }
      o2.innerHTML=o2HTML;

      result.innerHTML='光合作用需要光、二氧化碳和水，并受温度等条件影响。'
        +'<br>当前相对速率为 '+rate.toFixed(0)+'，主要限制因素是'+names[limitIndex]+'。该数值仅用于比较条件变化。';
    }

    light.oninput=update;co2.oninput=update;temp.oninput=update;water.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'biology-cellular-respiration',
    group: '⚡ 细胞代谢',
    name: '细胞呼吸',
    emoji: '⚡',
    desc: '调节有机物、氧气和温度，观察有氧呼吸与能量释放的相对变化',
    params: [
      { key: 'substrate', label: '有机物供应', type: 'number', min: 0, max: 100, step: 1, defaultValue: 72 },
      { key: 'oxygen', label: '氧气供应', type: 'number', min: 0, max: 100, step: 1, defaultValue: 80 },
      { key: 'temperature', label: '温度/℃', type: 'number', min: 5, max: 45, step: 1, defaultValue: 30 },
    ],
    buildHTML: (params, rootId) => {
      const substrate = num(params, 'substrate', 72)
      const oxygen = num(params, 'oxygen', 80)
      const temperature = num(params, 'temperature', 30)

      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">⚡ 细胞呼吸与能量释放</div>
    <div class="bl-note">主要演示有氧呼吸；最适温度因生物种类而异</div>
  </div>
  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row"><div class="bl-label"><span>有机物供应</span><span class="bl-value" data-sub-val></span></div><input data-sub type="range" min="0" max="100" step="1" value="${n(substrate)}"></div>
      <div class="bl-row"><div class="bl-label"><span>氧气供应</span><span class="bl-value" data-o2-val></span></div><input data-o2 type="range" min="0" max="100" step="1" value="${n(oxygen)}"></div>
      <div class="bl-row"><div class="bl-label"><span>温度</span><span class="bl-value" data-temp-val></span></div><input data-temp type="range" min="5" max="45" step="1" value="${n(temperature)}"></div>
      <div class="bl-result" data-result></div>
    </div>
    <div class="bl-stage">
      <svg viewBox="0 0 680 414">
        <defs>
          <marker id="${rootId}-resp-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#64748B"/>
          </marker>
        </defs>
        <rect width="680" height="414" fill="#FFFFFF"/>
        <ellipse cx="350" cy="210" rx="176" ry="112" fill="#FED7AA" stroke="#EA580C" stroke-width="7"/>
        <path d="M208 198 C250 126 300 294 344 182 C388 90 438 292 492 198" fill="none" stroke="#FB923C" stroke-width="10" stroke-linecap="round"/>
        <path d="M216 246 C258 162 304 318 354 226 C398 142 448 308 486 232" fill="none" stroke="#FDBA74" stroke-width="8" stroke-linecap="round"/>
        <text x="292" y="76" font-size="21" font-weight="900" fill="#9A3412">线粒体示意</text>
        <g data-inputs></g>
        <g data-outputs></g>
        <g data-atp></g>
        <text x="76" y="64" data-rate font-size="29" font-weight="900" fill="#065F46"></text>
        <text x="76" y="98" data-state font-size="17" font-weight="800" fill="#475569"></text>
        <text x="104" y="374" font-size="18" font-weight="900" fill="#9A3412">有机物 + 氧气 → 二氧化碳 + 水 + 能量（简式）</text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var sub=root.querySelector('[data-sub]'),o2=root.querySelector('[data-o2]'),temp=root.querySelector('[data-temp]');
    var subVal=root.querySelector('[data-sub-val]'),o2Val=root.querySelector('[data-o2-val]'),tempVal=root.querySelector('[data-temp-val]');
    var inputs=root.querySelector('[data-inputs]'),outputs=root.querySelector('[data-outputs]'),atp=root.querySelector('[data-atp]');
    var rateText=root.querySelector('[data-rate]'),stateText=root.querySelector('[data-state]'),result=root.querySelector('[data-result]');

    function update(){
      var S=Number(sub.value),O=Number(o2.value),T=Number(temp.value);
      var sf=S/(S+35);
      var of=O/(O+28);
      var tf=Math.exp(-Math.pow((T-30)/12,2));
      var rate=100*sf*of*tf;

      subVal.textContent=S.toFixed(0)+'%';
      o2Val.textContent=O.toFixed(0)+'%';
      tempVal.textContent=T.toFixed(0)+'℃';
      rateText.textContent='相对有氧呼吸速率 '+rate.toFixed(0);

      var state=O<18?'氧气不足，有氧呼吸明显受限':T<12||T>40?'温度偏离适宜范围，酶促反应受限':rate>45?'条件较适宜，能量释放较活跃':'有一个或多个条件限制速率';
      stateText.textContent=state;

      var inHTML='';
      var inputCount=Math.floor((S+O)/28);
      for(var i=0;i<inputCount;i++){
        var iy=132+i*24;
        var label=i%2===0?'O₂':'有机物';
        var color=i%2===0?'#0284C7':'#7C3AED';
        inHTML+='<text x="62" y="'+iy+'" font-size="14" font-weight="900" fill="'+color+'">'+label+'</text>';
        inHTML+='<path d="M118 '+(iy-5)+' C154 '+(iy-5)+' 170 174 194 190" fill="none" stroke="'+color+'" stroke-width="2.5" marker-end="url(#${rootId}-resp-arrow)"/>';
      }
      inputs.innerHTML=inHTML;

      var outHTML='';
      var outputCount=Math.floor(rate/14);
      for(var j=0;j<outputCount;j++){
        var oy=132+j*30;
        var outLabel=j%2===0?'CO₂':'H₂O';
        var outColor=j%2===0?'#64748B':'#0EA5E9';
        outHTML+='<path d="M506 198 C534 178 546 '+(oy-8)+' 566 '+(oy-8)+'" fill="none" stroke="'+outColor+'" stroke-width="2.5" marker-end="url(#${rootId}-resp-arrow)"/>';
        outHTML+='<text x="578" y="'+oy+'" font-size="14" font-weight="900" fill="'+outColor+'">'+outLabel+'</text>';
      }
      outputs.innerHTML=outHTML;

      var atpHTML='';
      var atpCount=Math.floor(rate/8);
      for(var k=0;k<atpCount;k++){
        var angle=k*2.399,radius=30+(k*19)%76;
        var x=350+Math.cos(angle)*radius,y=210+Math.sin(angle)*radius;
        atpHTML+='<polygon points="'+x+','+(y-8)+' '+(x+8)+','+y+' '+x+','+(y+8)+' '+(x-8)+','+y+'" fill="#FACC15" stroke="#CA8A04" stroke-width="2"/>';
      }
      atp.innerHTML=atpHTML;

      result.innerHTML='细胞通过呼吸作用释放有机物中的能量，供生命活动利用。'
        +'<br>'+state+'。低氧时可能发生其他代谢途径，本模型不展开其细节。';
    }

    sub.oninput=update;o2.oninput=update;temp.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'biology-transpiration',
    group: '🌿 植物生理',
    name: '蒸腾作用',
    emoji: '💧',
    desc: '调节气孔开放度、温度、空气湿度和风速，观察水分散失与运输',
    params: [
      { key: 'stomata', label: '气孔开放度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 65 },
      { key: 'temperature', label: '温度/℃', type: 'number', min: 10, max: 40, step: 1, defaultValue: 26 },
      { key: 'humidity', label: '空气湿度', type: 'number', min: 20, max: 100, step: 1, defaultValue: 55 },
      { key: 'wind', label: '风速', type: 'number', min: 0, max: 100, step: 1, defaultValue: 35 },
    ],
    buildHTML: (params, rootId) => {
      const stomata = num(params, 'stomata', 65)
      const temperature = num(params, 'temperature', 26)
      const humidity = num(params, 'humidity', 55)
      const wind = num(params, 'wind', 35)

      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">💧 蒸腾作用与气孔调节</div>
    <div class="bl-note">相对速率示意：植物状态与环境条件共同影响</div>
  </div>
  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row"><div class="bl-label"><span>气孔开放度</span><span class="bl-value" data-stoma-val></span></div><input data-stoma-range type="range" min="0" max="100" step="1" value="${n(stomata)}"></div>
      <div class="bl-row"><div class="bl-label"><span>温度</span><span class="bl-value" data-temp-val></span></div><input data-temp type="range" min="10" max="40" step="1" value="${n(temperature)}"></div>
      <div class="bl-row"><div class="bl-label"><span>空气湿度</span><span class="bl-value" data-hum-val></span></div><input data-hum type="range" min="20" max="100" step="1" value="${n(humidity)}"></div>
      <div class="bl-row"><div class="bl-label"><span>风速</span><span class="bl-value" data-wind-val></span></div><input data-wind type="range" min="0" max="100" step="1" value="${n(wind)}"></div>
      <div class="bl-result" data-result></div>
    </div>
    <div class="bl-stage">
      <svg viewBox="0 0 680 414">
        <defs>
          <marker id="${rootId}-water-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0EA5E9"/>
          </marker>
        </defs>
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M112 322 C196 322 212 246 300 246 C388 246 424 310 568 304" fill="none" stroke="#16A34A" stroke-width="48" stroke-linecap="round"/>
        <path d="M112 322 C196 322 212 246 300 246 C388 246 424 310 568 304" fill="none" stroke="#86EFAC" stroke-width="36" stroke-linecap="round"/>
        <g data-stoma-graphic>
          <ellipse data-left cx="308" cy="260" rx="54" ry="23" fill="#22C55E" stroke="#166534" stroke-width="5" transform="rotate(-24 308 260)"/>
          <ellipse data-right cx="372" cy="260" rx="54" ry="23" fill="#22C55E" stroke="#166534" stroke-width="5" transform="rotate(24 372 260)"/>
          <ellipse cx="340" cy="260" data-pore rx="12" ry="30" fill="#FFFFFF" stroke="#059669" stroke-width="4"/>
        </g>
        <text x="276" y="348" font-size="17" font-weight="900" fill="#166534">叶片下表皮气孔示意</text>
        <path d="M94 390 V300 C94 230 116 198 158 176" fill="none" stroke="#0284C7" stroke-width="13" stroke-linecap="round"/>
        <path d="M94 390 V300 C94 230 116 198 158 176" fill="none" stroke="#BAE6FD" stroke-width="5" stroke-linecap="round"/>
        <g data-xylem></g>
        <g data-vapor></g>
        <g data-wind-lines></g>
        <text x="52" y="70" data-rate font-size="29" font-weight="900" fill="#065F46"></text>
        <text x="52" y="104" data-state font-size="17" font-weight="800" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var stomaInput=root.querySelector('[data-stoma-range]');
    var temp=root.querySelector('[data-temp]'),hum=root.querySelector('[data-hum]'),wind=root.querySelector('[data-wind]');
    var stomaVal=root.querySelector('[data-stoma-val]'),tempVal=root.querySelector('[data-temp-val]');
    var humVal=root.querySelector('[data-hum-val]'),windVal=root.querySelector('[data-wind-val]');
    var pore=root.querySelector('[data-pore]'),left=root.querySelector('[data-left]'),right=root.querySelector('[data-right]');
    var xylem=root.querySelector('[data-xylem]'),vapor=root.querySelector('[data-vapor]'),windLines=root.querySelector('[data-wind-lines]');
    var rateText=root.querySelector('[data-rate]'),stateText=root.querySelector('[data-state]'),result=root.querySelector('[data-result]');

    function update(){
      var A=Number(stomaInput.value),T=Number(temp.value),H=Number(hum.value),W=Number(wind.value);
      var af=A/100;
      var tf=0.35+0.65*(T-10)/30;
      var hf=Math.max(0,(100-H)/80);
      var wf=0.65+0.35*W/100;
      var rate=100*af*tf*hf*wf;

      stomaVal.textContent=A.toFixed(0)+'%';
      tempVal.textContent=T.toFixed(0)+'℃';
      humVal.textContent=H.toFixed(0)+'%';
      windVal.textContent=W.toFixed(0)+'%';
      rateText.textContent='相对蒸腾速率 '+rate.toFixed(0);

      var state=A<15?'气孔接近关闭，水分散失明显减少':H>88?'空气湿度很高，叶内外水汽梯度较小':rate>45?'蒸腾较强，需注意水分供应':'蒸腾处于较低或中等水平';
      stateText.textContent=state;

      var gap=4+A*0.22;
      pore.setAttribute('rx',String(gap));
      left.setAttribute('cx',String(340-gap-38));
      right.setAttribute('cx',String(340+gap+38));

      var xylemHTML='';
      var waterCount=Math.floor(2+rate/10);
      for(var i=0;i<waterCount;i++){
        var y=378-i*22;
        xylemHTML+='<circle cx="'+(94+(i%2?4:-4))+'" cy="'+y+'" r="6" fill="#38BDF8"/>';
      }
      xylem.innerHTML=xylemHTML;

      var vaporHTML='';
      var vaporCount=Math.floor(rate/6);
      for(var j=0;j<vaporCount;j++){
        var col=j%6,row=Math.floor(j/6);
        var x=300+col*18+(row%2)*8,y=212-row*28-col%2*8;
        vaporHTML+='<circle cx="'+x+'" cy="'+y+'" r="'+(4+j%3)+'" fill="#BAE6FD" stroke="#0284C7" stroke-width="1.5" opacity="0.82"/>';
        if(j%3===0)vaporHTML+='<path d="M'+x+' '+(y-7)+' V'+(y-28)+'" stroke="#0EA5E9" stroke-width="2" marker-end="url(#${rootId}-water-arrow)"/>';
      }
      vapor.innerHTML=vaporHTML;

      var windHTML='';
      var lineCount=Math.floor(W/18);
      for(var k=0;k<lineCount;k++){
        var wy=128+k*18;
        windHTML+='<path d="M430 '+wy+' H'+(500+W*0.8)+'" stroke="#94A3B8" stroke-width="3" stroke-linecap="round" marker-end="url(#${rootId}-water-arrow)" opacity="0.75"/>';
      }
      windLines.innerHTML=windHTML;

      result.innerHTML='水主要通过气孔以水蒸气形式散失。温度升高、空气较干、风速增大或气孔开放，通常会增强蒸腾。'
        +'<br>蒸腾拉力有助于水和无机盐运输，但水分运输还受其他因素共同影响。';
    }

    stomaInput.oninput=update;temp.oninput=update;hum.oninput=update;wind.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
