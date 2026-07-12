/**
 * chemExperimentTemplatesExt5.ts — 化学实验第五批扩展模板
 *
 * 第五批新增：
 *   1. 配制一定质量分数溶液
 *   2. 水的净化
 *   3. 铁与硫酸铜置换反应
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

export const CHEM_EXPERIMENT_TEMPLATES_EXT5: ChemExperimentTemplate[] = [
  {
    id: 'chem-solution-mass-fraction',
    group: '🧫 基本实验操作',
    name: '配制一定质量分数溶液',
    emoji: '⚖️',
    desc: '调节溶质质量和水的质量，观察质量分数变化和配制步骤',
    params: [
      { key: 'solute', label: '溶质质量/g', type: 'number', min: 1, max: 60, step: 1, defaultValue: 10 },
      { key: 'water', label: '水的质量/g', type: 'number', min: 20, max: 300, step: 5, defaultValue: 90 },
      { key: 'progress', label: '配制进度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const solute = num(params, 'solute', 10)
      const water = num(params, 'water', 90)
      const progress = num(params, 'progress', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">⚖️ 配制溶液：溶质质量分数 = 溶质质量 / 溶液质量</div>
    <div class="ce-note">称量、量取、溶解、转移</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>溶质质量</span><span class="ce-value" data-s-val></span></div><input data-s type="range" min="1" max="60" step="1" value="${n(solute)}"></div>
      <div class="ce-row"><div class="ce-label"><span>水的质量</span><span class="ce-value" data-w-val></span></div><input data-w type="range" min="20" max="300" step="5" value="${n(water)}"></div>
      <div class="ce-row"><div class="ce-label"><span>配制进度</span><span class="ce-value" data-p-val></span></div><input data-p type="range" min="0" max="100" step="1" value="${n(progress)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="72" y="274" width="190" height="34" rx="12" fill="#CBD5E1" stroke="#64748B" stroke-width="4"/>
        <rect x="128" y="176" width="78" height="98" rx="10" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <text x="104" y="154" font-size="17" font-weight="900" fill="#475569">称量溶质</text>
        <text x="112" y="252" data-balance font-size="21" font-weight="900" fill="#075985"></text>
        <path d="M320 86 L460 86 L436 340 L344 340 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-liquid d="M340 250 C366 270 414 270 440 250 L436 340 L344 340 Z" fill="#BAE6FD" opacity="0.8"/>
        <g data-solute></g>
        <g data-step></g>
        <text x="314" y="66" font-size="17" font-weight="900" fill="#475569">烧杯溶解</text>
        <text x="238" y="378" data-fraction font-size="28" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var s=root.querySelector('[data-s]'), w=root.querySelector('[data-w]'), p=root.querySelector('[data-p]');
    var sv=root.querySelector('[data-s-val]'), wv=root.querySelector('[data-w-val]'), pv=root.querySelector('[data-p-val]');
    var balance=root.querySelector('[data-balance]'), liquid=root.querySelector('[data-liquid]'), soluteG=root.querySelector('[data-solute]');
    var step=root.querySelector('[data-step]'), fraction=root.querySelector('[data-fraction]'), res=root.querySelector('[data-result]');
    function update(){
      var S=Number(s.value), W=Number(w.value), P=Number(p.value);
      var frac=S/(S+W)*100;
      sv.textContent=S.toFixed(0)+' g'; wv.textContent=W.toFixed(0)+' g'; pv.textContent=P.toFixed(0)+'%';
      balance.textContent=S.toFixed(0)+' g';
      var h=Math.min(180,64+W*0.38), y=340-h;
      liquid.setAttribute('d','M340 '+y+' C366 '+(y+20)+' 414 '+(y+20)+' 440 '+y+' L436 340 L344 340 Z');
      var dots='';
      for(var i=0;i<Math.floor(S/3);i++){
        var x=358+(i*29)%62;
        var yy=P<35 ? 214+(i*17)%76 : y+28+(i*23)%Math.max(28,h-54);
        dots+='<circle cx="'+x+'" cy="'+yy+'" r="4" fill="#FACC15" opacity="'+(P<35?'0.95':'0.55')+'"/>';
      }
      soluteG.innerHTML=dots;
      step.innerHTML='<text x="494" y="138" font-size="20" font-weight="900" fill="#075985">'+(P<25?'① 称量':P<50?'② 量取水':P<75?'③ 溶解':'④ 转移定容')+'</text>';
      fraction.textContent='质量分数 ≈ '+frac.toFixed(1)+'%';
      res.innerHTML='配制一定质量分数溶液时，先计算所需溶质和溶剂质量，再称量、量取、溶解。<br>当前质量分数 = '+S.toFixed(0)+' / ('+S.toFixed(0)+' + '+W.toFixed(0)+') ×100% = '+frac.toFixed(1)+'%。';
    }
    s.oninput=update;w.oninput=update;p.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-water-purification',
    group: '🧫 基本实验操作',
    name: '水的净化',
    emoji: '🚰',
    desc: '模拟沉淀、过滤、吸附三个净水步骤，观察浑浊度降低',
    params: [
      { key: 'turbidity', label: '初始浑浊度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 75 },
      { key: 'filter', label: '过滤效果', type: 'number', min: 0, max: 100, step: 1, defaultValue: 65 },
      { key: 'carbon', label: '活性炭吸附', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const turbidity = num(params, 'turbidity', 75)
      const filter = num(params, 'filter', 65)
      const carbon = num(params, 'carbon', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🚰 水的净化：沉淀 → 过滤 → 吸附</div>
    <div class="ce-note">净化能降低杂质，但不一定杀菌</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>初始浑浊度</span><span class="ce-value" data-t-val></span></div><input data-t type="range" min="0" max="100" step="1" value="${n(turbidity)}"></div>
      <div class="ce-row"><div class="ce-label"><span>过滤效果</span><span class="ce-value" data-f-val></span></div><input data-f type="range" min="0" max="100" step="1" value="${n(filter)}"></div>
      <div class="ce-row"><div class="ce-label"><span>活性炭吸附</span><span class="ce-value" data-c-val></span></div><input data-c type="range" min="0" max="100" step="1" value="${n(carbon)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <g data-before></g>
        <g data-filter></g>
        <g data-after></g>
        <path d="M214 208 H262" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <path d="M247 190 L266 208 L247 226" fill="none" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <path d="M428 208 H476" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <path d="M461 190 L480 208 L461 226" fill="none" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <text x="82" y="72" font-size="18" font-weight="900" fill="#475569">原水</text>
        <text x="286" y="72" font-size="18" font-weight="900" fill="#475569">过滤/吸附</text>
        <text x="504" y="72" font-size="18" font-weight="900" fill="#475569">净化后</text>
        <text x="202" y="374" data-state font-size="24" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var t=root.querySelector('[data-t]'), f=root.querySelector('[data-f]'), c=root.querySelector('[data-c]');
    var tv=root.querySelector('[data-t-val]'), fv=root.querySelector('[data-f-val]'), cv=root.querySelector('[data-c-val]');
    var before=root.querySelector('[data-before]'), filterG=root.querySelector('[data-filter]'), after=root.querySelector('[data-after]');
    var state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function cup(x,op,count,label){
      var pts='';
      for(var i=0;i<count;i++){
        pts+='<circle cx="'+(x+28+(i*23)%78)+'" cy="'+(190+(i*31)%94)+'" r="'+(3+i%4)+'" fill="#92400E" opacity="'+op+'"/>';
      }
      return '<path d="M'+x+' 116 L'+(x+132)+' 116 L'+(x+112)+' 326 L'+(x+20)+' 326 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M'+(x+18)+' 210 C'+(x+46)+' 234 '+(x+88)+' 234 '+(x+116)+' 210 L'+(x+112)+' 326 L'+(x+20)+' 326 Z" fill="#BAE6FD" opacity="0.76"/>'+pts+'<text x="'+(x+24)+'" y="350" font-size="15" font-weight="900" fill="#475569">'+label+'</text>';
    }
    function update(){
      var T=Number(t.value), F=Number(f.value), C=Number(c.value);
      var afterT=Math.max(0,T*(1-F/130)*(1-C/160));
      tv.textContent=T.toFixed(0)+'%'; fv.textContent=F.toFixed(0)+'%'; cv.textContent=C.toFixed(0)+'%';
      before.innerHTML=cup(58,0.85,Math.floor(T/4),'浑浊水');
      filterG.innerHTML='<path d="M282 104 L414 104 L378 250 L318 250 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M304 126 L392 126 L370 226 L326 226 Z" fill="#FDE68A" opacity="0.62"/><rect x="322" y="168" width="54" height="26" fill="#1F2937" opacity="'+(C/120)+'"/><path d="M348 250 V318" stroke="#60A5FA" stroke-width="'+(4+F/16)+'" stroke-linecap="round" opacity="0.8"/><text x="302" y="350" font-size="15" font-weight="900" fill="#475569">滤纸+活性炭</text>';
      after.innerHTML=cup(492,0.28,Math.floor(afterT/6),'较澄清');
      state.textContent=afterT<20?'水较澄清':'仍有明显杂质';
      res.innerHTML='沉淀可除去较大颗粒，过滤可除去不溶性固体，活性炭可吸附色素和异味。<br>净化后的水不一定是纯水，也不一定已经消毒。';
    }
    t.oninput=update;f.oninput=update;c.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-iron-copper-sulfate',
    group: '⚗️ 反应现象',
    name: '铁与硫酸铜置换反应',
    emoji: '🔵',
    desc: '铁钉放入硫酸铜溶液，观察红色铜析出和蓝色溶液变浅',
    params: [
      { key: 'time', label: '反应时间', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
      { key: 'conc', label: '硫酸铜浓度', type: 'number', min: 10, max: 100, step: 5, defaultValue: 65 },
    ],
    buildHTML: (params, rootId) => {
      const time = num(params, 'time', 45)
      const conc = num(params, 'conc', 65)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🔵 置换反应：Fe + CuSO₄ → FeSO₄ + Cu</div>
    <div class="ce-note">活动性强的金属能把活动性弱的金属置换出来</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>反应时间</span><span class="ce-value" data-t-val></span></div><input data-t type="range" min="0" max="100" step="1" value="${n(time)}"></div>
      <div class="ce-row"><div class="ce-label"><span>CuSO₄浓度</span><span class="ce-value" data-c-val></span></div><input data-c type="range" min="10" max="100" step="5" value="${n(conc)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M238 72 L442 72 L404 342 L276 342 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-liquid d="M260 190 C296 216 384 216 420 190 L404 342 L276 342 Z" fill="#2563EB" opacity="0.75"/>
        <g data-nail></g>
        <g data-copper></g>
        <text x="244" y="54" font-size="18" font-weight="900" fill="#475569">铁钉 + 硫酸铜溶液</text>
        <text x="74" y="92" data-state font-size="27" font-weight="900" fill="#075985"></text>
        <text x="74" y="132" data-note font-size="18" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var t=root.querySelector('[data-t]'), c=root.querySelector('[data-c]');
    var tv=root.querySelector('[data-t-val]'), cv=root.querySelector('[data-c-val]');
    var liquid=root.querySelector('[data-liquid]'), nail=root.querySelector('[data-nail]'), copper=root.querySelector('[data-copper]');
    var state=root.querySelector('[data-state]'), note=root.querySelector('[data-note]'), res=root.querySelector('[data-result]');
    function update(){
      var T=Number(t.value), C=Number(c.value), rate=T*C/100;
      tv.textContent=T.toFixed(0)+'%'; cv.textContent=C.toFixed(0)+'%';
      liquid.setAttribute('opacity',String(Math.max(0.22,0.82-rate/150)));
      nail.innerHTML='<rect x="330" y="112" width="34" height="200" rx="12" fill="#94A3B8" stroke="#64748B" stroke-width="4" transform="rotate(15 347 212)"/>';
      var cu='';
      for(var i=0;i<Math.floor(rate/5);i++){
        cu+='<circle cx="'+(318+(i*21)%62)+'" cy="'+(150+(i*27)%132)+'" r="'+(4+i%4)+'" fill="#B45309" opacity="0.9"/>';
      }
      copper.innerHTML=cu;
      state.textContent=rate>38?'红色铜析出':'现象逐渐出现';
      note.textContent='蓝色溶液逐渐变浅';
      res.innerHTML='铁比铜活泼，能把硫酸铜溶液中的铜置换出来。<br>现象：铁钉表面出现红色物质，蓝色溶液逐渐变浅。';
    }
    t.oninput=update;c.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
