/**
 * chemExperimentTemplatesExt.ts — 化学实验扩展模板库
 *
 * 定位：
 *   不替代 chemExperimentTemplates.ts 的首批实验，
 *   只补充更常见、更适合课堂演示的化学实验过程组件。
 *
 * 接入方式：
 *   下一批在 ChemExperimentModal.tsx 中聚合：
 *   [...CHEM_EXPERIMENT_TEMPLATES, ...CHEM_EXPERIMENT_TEMPLATES_EXT]
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

export const CHEM_EXPERIMENT_TEMPLATES_EXT: ChemExperimentTemplate[] = [
  {
    id: 'chem-oxygen-prep',
    group: '🧪 气体制取与检验',
    name: '氧气制取与检验',
    emoji: '🫧',
    desc: '模拟加热高锰酸钾制取氧气，并用带火星木条检验氧气',
    params: [
      { key: 'heat', label: '加热强度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
      { key: 'time', label: '反应时间', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
    ],
    buildHTML: (params, rootId) => {
      const heat = num(params, 'heat', 55)
      const time = num(params, 'time', 45)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🫧 氧气制取：2KMnO₄ → K₂MnO₄ + MnO₂ + O₂↑</div>
    <div class="ce-note">带火星木条复燃说明氧气支持燃烧</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>加热强度</span><span class="ce-value" data-h-val></span></div><input data-h type="range" min="0" max="100" step="1" value="${n(heat)}"></div>
      <div class="ce-row"><div class="ce-label"><span>反应时间</span><span class="ce-value" data-t-val></span></div><input data-t type="range" min="0" max="100" step="1" value="${n(time)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M96 238 L330 190" stroke="#64748B" stroke-width="20" stroke-linecap="round"/>
        <path d="M116 234 L310 194" stroke="#FDE68A" stroke-width="14" stroke-linecap="round"/>
        <text x="104" y="166" font-size="17" font-weight="900" fill="#475569">试管：高锰酸钾</text>
        <path d="M325 192 C410 160 436 178 468 210" fill="none" stroke="#64748B" stroke-width="5"/>
        <path d="M468 210 L550 210 L532 332 L486 332 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-water d="M474 270 C494 286 526 286 546 270 L532 332 L486 332 Z" fill="#BAE6FD" opacity="0.85"/>
        <g data-bubbles></g>
        <g data-flame></g>
        <g data-stick></g>
        <text x="452" y="188" font-size="16" font-weight="900" fill="#0284C7">集气瓶</text>
        <text x="440" y="370" data-state font-size="22" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var h=root.querySelector('[data-h]'), t=root.querySelector('[data-t]');
    var hv=root.querySelector('[data-h-val]'), tv=root.querySelector('[data-t-val]');
    var bubbles=root.querySelector('[data-bubbles]'), flame=root.querySelector('[data-flame]'), stick=root.querySelector('[data-stick]');
    var state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function update(){
      var H=Number(h.value), T=Number(t.value), amount=H*T/100;
      hv.textContent=H.toFixed(0)+'%'; tv.textContent=T.toFixed(0)+'%';
      var bb='';
      for(var i=0;i<Math.floor(amount/5);i++){
        bb+='<circle cx="'+(492+(i%5)*9)+'" cy="'+(262-(i*13)%92)+'" r="'+(3+i%3)+'" fill="none" stroke="#38BDF8" stroke-width="2" opacity="0.9"/>';
      }
      bubbles.innerHTML=bb;
      var fh=Math.floor(H/18);
      var ff='';
      for(var j=0;j<fh;j++){
        var x=130+j*26;
        ff+='<path d="M'+x+' 290 C'+(x-12)+' 268 '+x+' 258 '+(x+10)+' 238 C'+(x+24)+' 264 '+(x+14)+' 282 '+x+' 290Z" fill="#F97316" opacity="0.85"/>';
      }
      flame.innerHTML=ff;
      var bright=amount>35;
      stick.innerHTML='<line x1="575" y1="214" x2="630" y2="184" stroke="#92400E" stroke-width="7" stroke-linecap="round"/>'
        + '<circle cx="572" cy="216" r="'+(bright?14:7)+'" fill="'+(bright?'#F97316':'#991B1B')+'" opacity="'+(bright?'0.95':'0.55')+'"/>'
        + (bright?'<circle cx="572" cy="216" r="24" fill="#FDBA74" opacity="0.25"/>':'');
      state.textContent=bright?'木条复燃':'氧气较少';
      res.innerHTML='氧气可用排水法收集；检验氧气时，带火星木条伸入集气瓶会复燃。<br>课堂提醒：试管口略向下倾斜，防止冷凝水倒流导致试管炸裂。';
    }
    h.oninput=update;t.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-mass-conservation',
    group: '⚗️ 反应现象',
    name: '质量守恒',
    emoji: '⚖️',
    desc: '反应前后原子种类和数目不变，总质量守恒',
    params: [
      { key: 'progress', label: '反应进度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 50 },
      { key: 'sealed', label: '密闭体系', type: 'boolean', defaultValue: true, hint: '密闭体系中总质量保持不变；开放体系若气体逸出会造成读数变化' },
    ],
    buildHTML: (params, rootId) => {
      const progress = num(params, 'progress', 50)
      const sealed = params.sealed === false ? false : true
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">⚖️ 质量守恒：反应前后总质量不变</div>
    <div class="ce-note">宏观质量守恒，微观原子重组</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>反应进度</span><span class="ce-value" data-p-val></span></div><input data-p type="range" min="0" max="100" step="1" value="${n(progress)}"></div>
      <div class="ce-row"><button data-seal>${sealed ? '密闭体系' : '开放体系'}</button></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="96" y="96" width="190" height="220" rx="22" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <rect x="394" y="96" width="190" height="220" rx="22" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <text x="142" y="76" font-size="20" font-weight="900" fill="#475569">反应前</text>
        <text x="440" y="76" font-size="20" font-weight="900" fill="#475569">反应后</text>
        <g data-before></g>
        <g data-after></g>
        <path d="M307 206 H370" stroke="#059669" stroke-width="6" stroke-linecap="round"/>
        <path d="M354 186 L374 206 L354 226" fill="none" stroke="#059669" stroke-width="6" stroke-linecap="round"/>
        <text x="126" y="358" data-m1 font-size="22" font-weight="900" fill="#075985"></text>
        <text x="424" y="358" data-m2 font-size="22" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var p=root.querySelector('[data-p]'), seal=root.querySelector('[data-seal]');
    var pv=root.querySelector('[data-p-val]'), before=root.querySelector('[data-before]'), after=root.querySelector('[data-after]');
    var m1=root.querySelector('[data-m1]'), m2=root.querySelector('[data-m2]'), res=root.querySelector('[data-result]');
    var sealed=${sealed ? 'true' : 'false'};
    seal.onclick=function(){sealed=!sealed;seal.textContent=sealed?'密闭体系':'开放体系';update();};
    function atom(x,y,c,t){return '<circle cx="'+x+'" cy="'+y+'" r="18" fill="'+c+'" opacity="0.95"/><text x="'+(x-6)+'" y="'+(y+6)+'" font-size="16" font-weight="900" fill="#fff">'+t+'</text>';}
    function update(){
      var P=Number(p.value); pv.textContent=P.toFixed(0)+'%';
      before.innerHTML=atom(150,160,'#2563EB','A')+atom(196,160,'#EF4444','B')+atom(150,220,'#2563EB','A')+atom(196,220,'#EF4444','B')+atom(238,190,'#F59E0B','C');
      var shift=P/100*28;
      after.innerHTML=atom(446-shift,168,'#2563EB','A')+atom(492-shift,168,'#EF4444','B')+atom(538-shift,168,'#F59E0B','C')+atom(456+shift,235,'#2563EB','A')+atom(502+shift,235,'#EF4444','B');
      m1.textContent='总质量：100.0 g';
      m2.textContent='总质量：'+(sealed?100:(100-P*0.08)).toFixed(1)+' g';
      res.innerHTML=sealed?'密闭体系中，参加反应的各物质质量总和等于生成物质量总和。<br>微观解释：反应只是原子重新组合，原子种类和数目不变。':'开放体系中若有气体逸出，天平读数可能变小；这不是质量不守恒，而是部分物质离开了称量体系。';
    }
    p.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-solubility',
    group: '🧫 基本实验操作',
    name: '溶解度与饱和溶液',
    emoji: '🧂',
    desc: '调节温度和溶质加入量，观察不饱和、饱和与晶体析出',
    params: [
      { key: 'temp', label: '温度/℃', type: 'number', min: 0, max: 90, step: 1, defaultValue: 40 },
      { key: 'solute', label: '加入溶质/g', type: 'number', min: 0, max: 120, step: 1, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const temp = num(params, 'temp', 40)
      const solute = num(params, 'solute', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🧂 溶解度：温度影响可溶解的最大溶质量</div>
    <div class="ce-note">以硝酸钾溶解度随温度升高明显增大为例</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>温度</span><span class="ce-value" data-t-val></span></div><input data-t type="range" min="0" max="90" step="1" value="${n(temp)}"></div>
      <div class="ce-row"><div class="ce-label"><span>溶质加入量</span><span class="ce-value" data-s-val></span></div><input data-s type="range" min="0" max="120" step="1" value="${n(solute)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M242 88 L438 88 L402 350 L278 350 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-liquid d="M262 210 C292 236 388 236 418 210 L402 350 L278 350 Z" fill="#BAE6FD" opacity="0.82"/>
        <g data-solute></g>
        <g data-crystal></g>
        <path d="M510 326 V116" stroke="#CBD5E1" stroke-width="10" stroke-linecap="round"/>
        <circle cx="510" cy="336" r="24" fill="#FEE2E2" stroke="#EF4444" stroke-width="4"/>
        <rect data-mercury x="505" y="210" width="10" height="116" rx="5" fill="#EF4444"/>
        <text x="538" y="176" data-limit font-size="20" font-weight="900" fill="#075985"></text>
        <text x="300" y="380" data-state font-size="24" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var t=root.querySelector('[data-t]'), s=root.querySelector('[data-s]');
    var tv=root.querySelector('[data-t-val]'), sv=root.querySelector('[data-s-val]');
    var sol=root.querySelector('[data-solute]'), cry=root.querySelector('[data-crystal]'), mercury=root.querySelector('[data-mercury]');
    var limit=root.querySelector('[data-limit]'), state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function update(){
      var T=Number(t.value), S=Number(s.value);
      var L=20+T*0.95;
      tv.textContent=T.toFixed(0)+'℃'; sv.textContent=S.toFixed(0)+' g';
      mercury.setAttribute('y',String(326-T*2.2)); mercury.setAttribute('height',String(T*2.2));
      limit.textContent='最多约 '+L.toFixed(0)+' g';
      var dissolved=Math.min(S,L), left=Math.max(0,S-L);
      var dots='', crystals='';
      for(var i=0;i<Math.floor(dissolved/4);i++){
        dots+='<circle cx="'+(286+(i*31)%104)+'" cy="'+(220+(i*19)%88)+'" r="4" fill="#38BDF8" opacity="0.75"/>';
      }
      for(var j=0;j<Math.floor(left/4);j++){
        var x=300+(j*17)%86, y=336-Math.floor(j/6)*10;
        crystals+='<polygon points="'+x+','+y+' '+(x+8)+','+(y+5)+' '+x+','+(y+11)+' '+(x-8)+','+(y+5)+'" fill="#FACC15" stroke="#B45309" stroke-width="1"/>';
      }
      sol.innerHTML=dots; cry.innerHTML=crystals;
      state.textContent=left>0?'饱和，有晶体剩余':'不饱和或恰好饱和';
      res.innerHTML='该模型体现“温度越高，硝酸钾溶解度越大”。<br>当加入溶质超过该温度下的溶解度时，多余固体不能继续溶解，会留在杯底。';
    }
    t.oninput=update;s.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
