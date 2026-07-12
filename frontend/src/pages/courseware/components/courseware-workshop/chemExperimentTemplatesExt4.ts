/**
 * chemExperimentTemplatesExt4.ts — 化学实验第四批扩展模板
 *
 * 第四批新增：
 *   1. 铁钉生锈条件
 *   2. 蜡烛燃烧产物检验
 *   3. 溶液稀释与浓度变化
 */

import type { ChemExperimentTemplate, ChemExperimentParamValue } from './chemExperimentUtils'

function num(params: Record<string, ChemExperimentParamValue>, key: string, fallback: number): number {
  const v = params[key]
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback
}

function bool(params: Record<string, ChemExperimentParamValue>, key: string, fallback: boolean): boolean {
  const v = params[key]
  return typeof v === 'boolean' ? v : fallback
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

export const CHEM_EXPERIMENT_TEMPLATES_EXT4: ChemExperimentTemplate[] = [
  {
    id: 'chem-rust-conditions',
    group: '⚗️ 反应现象',
    name: '铁钉生锈条件',
    emoji: '🔩',
    desc: '比较水、氧气和盐分对铁钉生锈的影响',
    params: [
      { key: 'water', label: '水分', type: 'number', min: 0, max: 100, step: 1, defaultValue: 70 },
      { key: 'oxygen', label: '氧气', type: 'number', min: 0, max: 100, step: 1, defaultValue: 75 },
      { key: 'salt', label: '盐分', type: 'number', min: 0, max: 100, step: 1, defaultValue: 25 },
    ],
    buildHTML: (params, rootId) => {
      const water = num(params, 'water', 70)
      const oxygen = num(params, 'oxygen', 75)
      const salt = num(params, 'salt', 25)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🔩 铁钉生锈：铁 + 水 + 氧气</div>
    <div class="ce-note">盐分会加快铁生锈</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>水分</span><span class="ce-value" data-w-val></span></div><input data-w type="range" min="0" max="100" step="1" value="${n(water)}"></div>
      <div class="ce-row"><div class="ce-label"><span>氧气</span><span class="ce-value" data-o-val></span></div><input data-o type="range" min="0" max="100" step="1" value="${n(oxygen)}"></div>
      <div class="ce-row"><div class="ce-label"><span>盐分</span><span class="ce-value" data-s-val></span></div><input data-s type="range" min="0" max="100" step="1" value="${n(salt)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M220 70 L460 70 L420 348 L260 348 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-water d="M242 230 C286 258 394 258 438 230 L420 348 L260 348 Z" fill="#BAE6FD" opacity="0.72"/>
        <g data-nail></g>
        <g data-rust></g>
        <g data-o2></g>
        <text x="250" y="54" font-size="18" font-weight="900" fill="#475569">试管中的铁钉</text>
        <text x="74" y="82" data-state font-size="28" font-weight="900" fill="#075985"></text>
        <text x="74" y="122" data-speed font-size="19" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var w=root.querySelector('[data-w]'), o=root.querySelector('[data-o]'), s=root.querySelector('[data-s]');
    var wv=root.querySelector('[data-w-val]'), ov=root.querySelector('[data-o-val]'), sv=root.querySelector('[data-s-val]');
    var water=root.querySelector('[data-water]'), nail=root.querySelector('[data-nail]'), rust=root.querySelector('[data-rust]'), o2=root.querySelector('[data-o2]');
    var state=root.querySelector('[data-state]'), speed=root.querySelector('[data-speed]'), res=root.querySelector('[data-result]');
    function update(){
      var W=Number(w.value), O=Number(o.value), S=Number(s.value);
      var rate=(W/100)*(O/100)*(1+S/90)*100;
      wv.textContent=W.toFixed(0)+'%'; ov.textContent=O.toFixed(0)+'%'; sv.textContent=S.toFixed(0)+'%';
      water.setAttribute('opacity',String(0.18+W/140));
      nail.innerHTML='<rect x="320" y="118" width="34" height="190" rx="12" fill="#94A3B8" stroke="#64748B" stroke-width="4" transform="rotate(18 337 213)"/>';
      var rr='';
      for(var i=0;i<Math.floor(rate/7);i++){
        rr+='<circle cx="'+(314+(i*23)%56)+'" cy="'+(146+(i*31)%132)+'" r="'+(4+i%4)+'" fill="#B45309" opacity="0.84"/>';
      }
      rust.innerHTML=rr;
      var oo='';
      for(var j=0;j<Math.floor(O/8);j++){
        oo+='<circle cx="'+(250+(j*37)%168)+'" cy="'+(92+(j*29)%110)+'" r="5" fill="#38BDF8" opacity="0.36"/>';
      }
      o2.innerHTML=oo;
      state.textContent=rate>45?'生锈明显':rate>12?'缓慢生锈':'几乎不生锈';
      speed.textContent='生锈趋势 ≈ '+rate.toFixed(0)+'%';
      res.innerHTML='铁生锈需要同时接触水和氧气；盐分会提高腐蚀速度。<br>保持干燥、隔绝氧气、刷漆或镀层都能减缓生锈。';
    }
    w.oninput=update;o.oninput=update;s.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-candle-products',
    group: '🧪 气体制取与检验',
    name: '蜡烛燃烧产物检验',
    emoji: '🕯️',
    desc: '观察蜡烛燃烧生成水和二氧化碳的检验现象',
    params: [
      { key: 'burn', label: '燃烧强度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 65 },
      { key: 'cool', label: '冷玻璃片温度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 25 },
    ],
    buildHTML: (params, rootId) => {
      const burn = num(params, 'burn', 65)
      const cool = num(params, 'cool', 25)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🕯️ 蜡烛燃烧产物：水 + 二氧化碳</div>
    <div class="ce-note">冷玻璃片检验水，石灰水检验 CO₂</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>燃烧强度</span><span class="ce-value" data-b-val></span></div><input data-b type="range" min="0" max="100" step="1" value="${n(burn)}"></div>
      <div class="ce-row"><div class="ce-label"><span>玻璃片温度</span><span class="ce-value" data-c-val></span></div><input data-c type="range" min="0" max="100" step="1" value="${n(cool)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="296" y="228" width="88" height="108" rx="12" fill="#FEF3C7" stroke="#D97706" stroke-width="4"/>
        <line x1="340" y1="224" x2="340" y2="194" stroke="#92400E" stroke-width="5"/>
        <g data-flame></g>
        <rect x="242" y="82" width="196" height="42" rx="12" data-glass fill="#E0F2FE" stroke="#0284C7" stroke-width="4" opacity="0.75"/>
        <g data-drops></g>
        <path d="M474 210 L588 210 L566 334 L496 334 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-lime d="M486 274 C510 292 552 292 576 274 L566 334 L496 334 Z" fill="#BAE6FD" opacity="0.85"/>
        <text x="244" y="72" font-size="17" font-weight="900" fill="#0284C7">冷玻璃片</text>
        <text x="466" y="188" font-size="17" font-weight="900" fill="#475569">澄清石灰水</text>
        <text x="86" y="92" data-state font-size="24" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var b=root.querySelector('[data-b]'), c=root.querySelector('[data-c]');
    var bv=root.querySelector('[data-b-val]'), cv=root.querySelector('[data-c-val]');
    var flame=root.querySelector('[data-flame]'), drops=root.querySelector('[data-drops]'), lime=root.querySelector('[data-lime]');
    var state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function update(){
      var B=Number(b.value), C=Number(c.value);
      var water=Math.max(0,B*(100-C)/100);
      var co2=B;
      bv.textContent=B.toFixed(0)+'%'; cv.textContent=C.toFixed(0)+'%';
      flame.innerHTML=B>8?'<path d="M340 198 C306 152 336 132 350 96 C388 144 378 176 340 198Z" fill="#F97316" opacity="'+(0.35+B/120)+'"/><path d="M340 188 C320 158 340 144 350 124 C372 160 366 178 340 188Z" fill="#FACC15" opacity="0.9"/>':'';
      var dd='';
      for(var i=0;i<Math.floor(water/8);i++){
        dd+='<ellipse cx="'+(268+(i*29)%140)+'" cy="'+(134+(i*17)%42)+'" rx="4" ry="7" fill="#38BDF8" opacity="0.72"/>';
      }
      drops.innerHTML=dd;
      lime.setAttribute('fill',co2>45?'#E5E7EB':'#BAE6FD');
      state.textContent=B>20?'水雾 + 石灰水变浑浊':'现象不明显';
      res.innerHTML='冷玻璃片上出现水雾，说明燃烧生成水；燃烧气体使澄清石灰水变浑浊，说明生成二氧化碳。<br>蜡烛主要含碳、氢元素，充分燃烧生成 CO₂ 和 H₂O。';
    }
    b.oninput=update;c.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-dilution-concentration',
    group: '🧫 基本实验操作',
    name: '溶液稀释与浓度变化',
    emoji: '💧',
    desc: '加入水稀释溶液，观察浓度降低但溶质质量不变',
    params: [
      { key: 'solute', label: '溶质质量/g', type: 'number', min: 1, max: 50, step: 1, defaultValue: 12 },
      { key: 'water', label: '加水量/mL', type: 'number', min: 20, max: 300, step: 10, defaultValue: 120 },
      { key: 'color', label: '显示颜色深浅', type: 'boolean', defaultValue: true },
    ],
    buildHTML: (params, rootId) => {
      const solute = num(params, 'solute', 12)
      const water = num(params, 'water', 120)
      const color = bool(params, 'color', true)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">💧 溶液稀释：加水后浓度降低</div>
    <div class="ce-note">稀释前后溶质质量不变</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>溶质质量</span><span class="ce-value" data-s-val></span></div><input data-s type="range" min="1" max="50" step="1" value="${n(solute)}"></div>
      <div class="ce-row"><div class="ce-label"><span>加水量</span><span class="ce-value" data-w-val></span></div><input data-w type="range" min="20" max="300" step="10" value="${n(water)}"></div>
      <div class="ce-row"><button data-color>${color ? '隐藏颜色提示' : '显示颜色提示'}</button></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M160 96 L300 96 L276 336 L184 336 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M380 96 L520 96 L496 336 L404 336 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-before d="M178 232 C204 252 256 252 282 232 L276 336 L184 336 Z" fill="#2563EB" opacity="0.75"/>
        <path data-after d="M398 202 C424 222 476 222 502 202 L496 336 L404 336 Z" fill="#2563EB" opacity="0.38"/>
        <text x="178" y="76" font-size="18" font-weight="900" fill="#475569">稀释前</text>
        <text x="398" y="76" font-size="18" font-weight="900" fill="#475569">稀释后</text>
        <path d="M315 206 H358" stroke="#059669" stroke-width="6" stroke-linecap="round"/>
        <path d="M342 188 L362 206 L342 224" fill="none" stroke="#059669" stroke-width="6" stroke-linecap="round"/>
        <text x="126" y="374" data-c1 font-size="21" font-weight="900" fill="#075985"></text>
        <text x="376" y="374" data-c2 font-size="21" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var s=root.querySelector('[data-s]'), w=root.querySelector('[data-w]'), btn=root.querySelector('[data-color]');
    var sv=root.querySelector('[data-s-val]'), wv=root.querySelector('[data-w-val]');
    var before=root.querySelector('[data-before]'), after=root.querySelector('[data-after]');
    var c1=root.querySelector('[data-c1]'), c2=root.querySelector('[data-c2]'), res=root.querySelector('[data-result]');
    var showColor=${color ? 'true' : 'false'};
    btn.onclick=function(){showColor=!showColor;btn.textContent=showColor?'隐藏颜色提示':'显示颜色提示';update();};
    function update(){
      var S=Number(s.value), W=Number(w.value);
      var V1=80, V2=80+W, C1=S/V1*100, C2=S/V2*100;
      sv.textContent=S.toFixed(0)+' g'; wv.textContent=W.toFixed(0)+' mL';
      before.setAttribute('opacity',showColor?String(Math.min(0.9,0.25+C1/30)):'0.55');
      after.setAttribute('opacity',showColor?String(Math.min(0.9,0.25+C2/30)):'0.55');
      c1.textContent='C前≈'+C1.toFixed(1)+'%';
      c2.textContent='C后≈'+C2.toFixed(1)+'%';
      res.innerHTML='稀释时加入的是溶剂，溶质质量不变；溶液体积增大，所以质量分数降低。<br>可以类比 C1V1 = C2V2 的稀释关系。';
    }
    s.oninput=update;w.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
