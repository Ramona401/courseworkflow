/**
 * chemExperimentTemplatesExt6.ts — 化学实验第六批扩展模板
 *
 * 第六批新增：
 *   1. 过氧化氢分解与催化剂
 *   2. 反应速率影响因素
 *   3. 溶液导电性
 */

import type { ChemExperimentTemplate, ChemExperimentParamValue } from './chemExperimentUtils'

function num(params: Record<string, ChemExperimentParamValue>, key: string, fallback: number): number {
  const v = params[key]
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback
}

function n(v: number): string {
  return parseFloat(v.toFixed(3)).toString()
}

function baseStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #E5E7EB;border-radius:16px;background:#FFFFFF;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' .ce-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#DCFCE7,#F0FDF4);border-bottom:1px solid #E5E7EB;box-sizing:border-box;}\n'
    + '#' + rootId + ' .ce-title{font-size:15px;font-weight:800;color:#047857;}\n'
    + '#' + rootId + ' .ce-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .ce-body{height:calc(100% - 46px);display:grid;grid-template-columns:220px 1fr;min-height:0;}\n'
    + '#' + rootId + ' .ce-controls{padding:14px;border-right:1px solid #E5E7EB;background:#F8FAFC;box-sizing:border-box;overflow:auto;}\n'
    + '#' + rootId + ' .ce-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .ce-row{margin-bottom:13px;}\n'
    + '#' + rootId + ' .ce-label{display:flex;justify-content:space-between;gap:8px;font-size:12px;font-weight:700;color:#334155;margin-bottom:6px;}\n'
    + '#' + rootId + ' .ce-value{font-weight:800;color:#059669;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981;}\n'
    + '#' + rootId + ' button{border:none;border-radius:10px;padding:8px 14px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:12px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .ce-result{padding:10px 12px;border-radius:10px;background:#DCFCE7;color:#065F46;font-size:12px;line-height:1.6;font-weight:600;}\n'
    + '#' + rootId + ' svg{width:100%;height:100%;display:block;}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const CHEM_EXPERIMENT_TEMPLATES_EXT6: ChemExperimentTemplate[] = [
  {
    id: 'chem-h2o2-catalyst',
    group: '⚗️ 反应现象',
    name: '过氧化氢分解与催化剂',
    emoji: '🫧',
    desc: '比较无催化剂、二氧化锰、酶催化下氧气产生快慢',
    params: [
      { key: 'catalyst', label: '催化剂', type: 'number', min: 0, max: 2, step: 1, defaultValue: 1, hint: '0=无，1=MnO₂，2=酶' },
      { key: 'conc', label: 'H₂O₂浓度', type: 'number', min: 5, max: 30, step: 1, defaultValue: 15 },
      { key: 'time', label: '反应时间', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
    ],
    buildHTML: (params, rootId) => {
      const catalyst = num(params, 'catalyst', 1)
      const conc = num(params, 'conc', 15)
      const time = num(params, 'time', 45)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🫧 H₂O₂ 分解：2H₂O₂ → 2H₂O + O₂↑</div>
    <div class="ce-note">催化剂改变反应速率，不改变生成物</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>催化剂</span><span class="ce-value" data-cat-val></span></div><input data-cat type="range" min="0" max="2" step="1" value="${n(catalyst)}"></div>
      <div class="ce-row"><div class="ce-label"><span>H₂O₂浓度</span><span class="ce-value" data-c-val></span></div><input data-c type="range" min="5" max="30" step="1" value="${n(conc)}"></div>
      <div class="ce-row"><div class="ce-label"><span>反应时间</span><span class="ce-value" data-t-val></span></div><input data-t type="range" min="0" max="100" step="1" value="${n(time)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M238 84 L442 84 L404 342 L276 342 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M260 210 C294 236 386 236 420 210 L404 342 L276 342 Z" fill="#BAE6FD" opacity="0.78"/>
        <g data-catalyst></g>
        <g data-bubbles></g>
        <text x="246" y="62" font-size="18" font-weight="900" fill="#475569">过氧化氢溶液</text>
        <text x="72" y="84" data-state font-size="27" font-weight="900" fill="#075985"></text>
        <text x="72" y="124" data-rate font-size="19" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var cat=root.querySelector('[data-cat]'), c=root.querySelector('[data-c]'), t=root.querySelector('[data-t]');
    var catv=root.querySelector('[data-cat-val]'), cv=root.querySelector('[data-c-val]'), tv=root.querySelector('[data-t-val]');
    var catalyst=root.querySelector('[data-catalyst]'), bubbles=root.querySelector('[data-bubbles]');
    var state=root.querySelector('[data-state]'), rateText=root.querySelector('[data-rate]'), res=root.querySelector('[data-result]');
    var names=['无催化剂','MnO₂','酶'];
    var factors=[0.22,1.0,0.78];
    function update(){
      var K=Math.round(Number(cat.value)), C=Number(c.value), T=Number(t.value);
      var rate=C*T*factors[K]/18;
      catv.textContent=names[K]; cv.textContent=C.toFixed(0)+'%'; tv.textContent=T.toFixed(0)+'%';
      catalyst.innerHTML=K===0?'':'<g><circle cx="340" cy="315" r="8" fill="'+(K===1?'#1F2937':'#22C55E')+'"/><circle cx="362" cy="306" r="7" fill="'+(K===1?'#374151':'#16A34A')+'"/><circle cx="320" cy="302" r="7" fill="'+(K===1?'#111827':'#4ADE80')+'"/></g>';
      var bb='';
      for(var i=0;i<Math.min(34,Math.floor(rate));i++){
        bb+='<circle cx="'+(300+(i*23)%88)+'" cy="'+(292-(i*17)%172)+'" r="'+(3+i%4)+'" fill="none" stroke="#38BDF8" stroke-width="2" opacity="0.9"/>';
      }
      bubbles.innerHTML=bb;
      state.textContent=rate>18?'氧气大量产生':rate>5?'缓慢产生氧气':'现象较弱';
      rateText.textContent='反应速率：'+(rate>18?'快':rate>5?'中等':'慢');
      res.innerHTML='催化剂能加快过氧化氢分解产生氧气，但反应前后催化剂质量和化学性质基本不变。<br>可用带火星木条检验生成的氧气。';
    }
    cat.oninput=update;c.oninput=update;t.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-reaction-rate-factors',
    group: '⚗️ 反应现象',
    name: '反应速率影响因素',
    emoji: '⏱️',
    desc: '调节温度、浓度和接触面积，观察气泡产生速率变化',
    params: [
      { key: 'temp', label: '温度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
      { key: 'conc', label: '反应物浓度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 60 },
      { key: 'area', label: '接触面积', type: 'number', min: 10, max: 100, step: 1, defaultValue: 65 },
    ],
    buildHTML: (params, rootId) => {
      const temp = num(params, 'temp', 55)
      const conc = num(params, 'conc', 60)
      const area = num(params, 'area', 65)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">⏱️ 反应速率：温度、浓度、接触面积</div>
    <div class="ce-note">以固体和酸反应产生气体为例</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>温度</span><span class="ce-value" data-t-val></span></div><input data-t type="range" min="0" max="100" step="1" value="${n(temp)}"></div>
      <div class="ce-row"><div class="ce-label"><span>浓度</span><span class="ce-value" data-c-val></span></div><input data-c type="range" min="0" max="100" step="1" value="${n(conc)}"></div>
      <div class="ce-row"><div class="ce-label"><span>接触面积</span><span class="ce-value" data-a-val></span></div><input data-a type="range" min="10" max="100" step="1" value="${n(area)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M240 80 L440 80 L404 340 L276 340 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M262 204 C298 232 382 232 418 204 L404 340 L276 340 Z" fill="#FDE68A" opacity="0.82"/>
        <g data-solid></g>
        <g data-bubbles></g>
        <g data-heat></g>
        <text x="260" y="62" font-size="18" font-weight="900" fill="#475569">固体 + 酸</text>
        <text x="78" y="90" data-state font-size="28" font-weight="900" fill="#075985"></text>
        <text x="78" y="130" data-speed font-size="19" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var t=root.querySelector('[data-t]'), c=root.querySelector('[data-c]'), a=root.querySelector('[data-a]');
    var tv=root.querySelector('[data-t-val]'), cv=root.querySelector('[data-c-val]'), av=root.querySelector('[data-a-val]');
    var solid=root.querySelector('[data-solid]'), bubbles=root.querySelector('[data-bubbles]'), heat=root.querySelector('[data-heat]');
    var state=root.querySelector('[data-state]'), speed=root.querySelector('[data-speed]'), res=root.querySelector('[data-result]');
    function update(){
      var T=Number(t.value), C=Number(c.value), A=Number(a.value);
      var rate=(0.25+T/100)*(0.2+C/100)*(0.3+A/100)*42;
      tv.textContent=T.toFixed(0)+'%'; cv.textContent=C.toFixed(0)+'%'; av.textContent=A.toFixed(0)+'%';
      var pieces=Math.floor(3+A/9), ss='';
      for(var i=0;i<pieces;i++){
        ss+='<polygon points="'+(296+(i*17)%88)+','+(310-Math.floor(i/6)*13)+' '+(308+(i*17)%88)+','+(315-Math.floor(i/6)*13)+' '+(302+(i*17)%88)+','+(326-Math.floor(i/6)*13)+' '+(288+(i*17)%88)+','+(320-Math.floor(i/6)*13)+'" fill="#A3A3A3" stroke="#737373" stroke-width="1"/>';
      }
      solid.innerHTML=ss;
      var bb='';
      for(var j=0;j<Math.min(42,Math.floor(rate));j++){
        bb+='<circle cx="'+(300+(j*19)%82)+'" cy="'+(292-(j*13)%156)+'" r="'+(3+j%3)+'" fill="none" stroke="#38BDF8" stroke-width="2" opacity="0.88"/>';
      }
      bubbles.innerHTML=bb;
      var hh='';
      for(var k=0;k<Math.floor(T/24);k++){
        var x=286+k*30;
        hh+='<path d="M'+x+' 364 C'+(x-12)+' 342 '+x+' 332 '+(x+10)+' 312 C'+(x+24)+' 338 '+(x+14)+' 356 '+x+' 364Z" fill="#F97316" opacity="0.65"/>';
      }
      heat.innerHTML=hh;
      state.textContent=rate>32?'反应很快':rate>15?'反应明显':'反应较慢';
      speed.textContent='速率指数 ≈ '+rate.toFixed(0);
      res.innerHTML='升高温度、增大反应物浓度、增大固体接触面积，通常都能加快反应速率。<br>课堂可用“控制变量法”分别比较每个因素。';
    }
    t.oninput=update;c.oninput=update;a.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-solution-conductivity',
    group: '🔋 电化学',
    name: '溶液导电性',
    emoji: '💡',
    desc: '比较纯水、食盐水、糖水和酸溶液的导电性差异',
    params: [
      { key: 'solute', label: '溶质选择', type: 'number', min: 0, max: 3, step: 1, defaultValue: 1, hint: '0=纯水，1=食盐，2=蔗糖，3=盐酸' },
      { key: 'conc', label: '浓度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const solute = num(params, 'solute', 1)
      const conc = num(params, 'conc', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">💡 溶液导电性：是否产生自由移动离子</div>
    <div class="ce-note">电解质溶液能导电，非电解质溶液通常不导电</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>溶质</span><span class="ce-value" data-s-val></span></div><input data-s type="range" min="0" max="3" step="1" value="${n(solute)}"></div>
      <div class="ce-row"><div class="ce-label"><span>浓度</span><span class="ce-value" data-c-val></span></div><input data-c type="range" min="0" max="100" step="1" value="${n(conc)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M230 160 L450 160 L410 342 L270 342 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M250 226 C292 252 388 252 430 226 L410 342 L270 342 Z" fill="#BAE6FD" opacity="0.78"/>
        <line x1="302" y1="126" x2="302" y2="278" stroke="#334155" stroke-width="7" stroke-linecap="round"/>
        <line x1="378" y1="126" x2="378" y2="278" stroke="#334155" stroke-width="7" stroke-linecap="round"/>
        <path d="M302 126 V78 H160 V126" fill="none" stroke="#334155" stroke-width="5"/>
        <path d="M378 126 V78 H520 V126" fill="none" stroke="#334155" stroke-width="5"/>
        <circle cx="160" cy="148" r="36" data-lamp fill="#F8FAFC" stroke="#F59E0B" stroke-width="4"/>
        <path d="M142 150 q18 -26 36 0" stroke="#92400E" stroke-width="4" fill="none"/>
        <g data-ions></g>
        <text x="84" y="72" data-name font-size="26" font-weight="900" fill="#075985"></text>
        <text x="84" y="204" data-light font-size="21" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var s=root.querySelector('[data-s]'), c=root.querySelector('[data-c]');
    var sv=root.querySelector('[data-s-val]'), cv=root.querySelector('[data-c-val]');
    var ions=root.querySelector('[data-ions]'), lamp=root.querySelector('[data-lamp]');
    var name=root.querySelector('[data-name]'), light=root.querySelector('[data-light]'), res=root.querySelector('[data-result]');
    var list=[
      {n:'纯水', k:0.04, desc:'纯水中离子很少，导电性很弱。'},
      {n:'食盐水', k:0.85, desc:'NaCl 溶于水产生 Na⁺ 和 Cl⁻，能导电。'},
      {n:'蔗糖溶液', k:0.02, desc:'蔗糖溶于水主要以分子存在，几乎不导电。'},
      {n:'盐酸', k:1.0, desc:'HCl 在水中电离产生 H⁺ 和 Cl⁻，导电性强。'}
    ];
    function update(){
      var S=Math.round(Number(s.value)), C=Number(c.value), info=list[S], power=info.k*C;
      sv.textContent=info.n; cv.textContent=C.toFixed(0)+'%';
      name.textContent=info.n;
      lamp.setAttribute('fill',power>55?'#FDE68A':power>18?'#FEF3C7':'#F8FAFC');
      lamp.setAttribute('opacity',String(0.45+Math.min(0.55,power/100)));
      var html='';
      for(var i=0;i<Math.floor(power/5);i++){
        html+='<circle cx="'+(278+(i*29)%122)+'" cy="'+(238+(i*17)%74)+'" r="5" fill="'+(i%2?'#2563EB':'#EF4444')+'" opacity="0.74"/><text x="'+(274+(i*29)%122)+'" y="'+(242+(i*17)%74)+'" font-size="8" fill="#fff">'+(i%2?'-':'+')+'</text>';
      }
      ions.innerHTML=html;
      light.textContent=power>55?'灯泡较亮':power>18?'灯泡微亮':'灯泡几乎不亮';
      res.innerHTML=info.desc+'<br>溶液能否导电，关键看是否存在大量自由移动的离子。';
    }
    s.oninput=update;c.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
