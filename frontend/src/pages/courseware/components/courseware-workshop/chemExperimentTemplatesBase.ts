/**
 * chemExperimentTemplates.ts — 化学实验过程模板注册表（第4批A新增，2026-07-09）
 *
 * 首批覆盖：
 *   1. 实验基本技能：过滤、蒸发结晶
 *   2. 气体制备：CO₂ 制取与检验
 *   3. 反应现象：酸碱中和、沉淀反应、金属与酸反应
 *   4. 电化学：电解水
 *
 * 实现方式：
 *   - 全部为纯 HTML + SVG/Canvas + 原生 JS。
 *   - 每个模板输出完整自包含组件。
 *   - 运行时只查询 rootId 内部 DOM，避免同页多个实验互相干扰。
 */
import type { ChemExperimentTemplate, ChemExperimentParamValue } from './chemExperimentUtils'

// ============================================================
// 共享辅助
// ============================================================

/** 读取数字参数 */
function num(params: Record<string, ChemExperimentParamValue>, key: string, fallback: number): number {
  const v = params[key]
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback
}

/** 保留两位小数用于内联脚本初值 */
function n(v: number): string {
  return parseFloat(v.toFixed(3)).toString()
}

/** 统一卡片外观 CSS */
function baseStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #E5E7EB;border-radius:16px;background:#FFFFFF;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' .ce-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#DCFCE7,#F0FDF4);border-bottom:1px solid #E5E7EB;box-sizing:border-box;}\n'
    + '#' + rootId + ' .ce-title{font-size:15px;font-weight:800;color:#047857;}\n'
    + '#' + rootId + ' .ce-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .ce-body{height:calc(100% - 46px);display:grid;grid-template-columns:220px 1fr;min-height:0;}\n'
    + '#' + rootId + ' .ce-controls{padding:14px 14px;border-right:1px solid #E5E7EB;background:#F8FAFC;box-sizing:border-box;overflow:auto;}\n'
    + '#' + rootId + ' .ce-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .ce-row{margin-bottom:13px;}\n'
    + '#' + rootId + ' .ce-label{display:flex;justify-content:space-between;gap:8px;font-size:12px;font-weight:700;color:#334155;margin-bottom:6px;}\n'
    + '#' + rootId + ' .ce-value{font-weight:800;color:#059669;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981;}\n'
    + '#' + rootId + ' button{border:none;border-radius:10px;padding:8px 14px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:12px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .ce-result{padding:10px 12px;border-radius:10px;background:#DCFCE7;color:#065F46;font-size:12px;line-height:1.6;font-weight:600;}\n'
    + '#' + rootId + ' svg{width:100%;height:100%;display:block;}\n'
    + '#' + rootId + ' canvas{width:100%;height:100%;display:block;}\n'
    + '</style>\n'
}

/** 生成脚本闭合标签，避免源码中出现连续 closing script 字符串 */
const SCRIPT_END = '</' + 'script>'

// ============================================================
// 模板注册表
// ============================================================

export const CHEM_EXPERIMENT_TEMPLATES: ChemExperimentTemplate[] = [
  {
    id: 'chem-filtration',
    group: '🧫 基本实验操作',
    name: '过滤实验',
    emoji: '🧫',
    desc: '模拟“溶液+不溶固体”的过滤过程，观察滤渣和滤液分离',
    params: [
      { key: 'progress', label: '过滤进度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 35 },
      { key: 'solid', label: '固体含量', type: 'number', min: 10, max: 80, step: 5, defaultValue: 45 },
    ],
    buildHTML: (params, rootId) => {
      const progress = num(params, 'progress', 35)
      const solid = num(params, 'solid', 45)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🧫 过滤：固液分离</div>
    <div class="ce-note">一贴二低三靠，观察滤液与滤渣</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row">
        <div class="ce-label"><span>过滤进度</span><span class="ce-value" data-p-val></span></div>
        <input data-p type="range" min="0" max="100" step="1" value="${n(progress)}">
      </div>
      <div class="ce-row">
        <div class="ce-label"><span>固体含量</span><span class="ce-value" data-s-val></span></div>
        <input data-s type="range" min="10" max="80" step="5" value="${n(solid)}">
      </div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M250 90 L430 90 L385 215 L295 215 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M270 110 L410 110 L372 195 L308 195 Z" fill="#FDE68A" opacity="0.65" stroke="#D97706" stroke-width="2"/>
        <path data-residue d="" fill="#92400E" opacity="0.8"/>
        <path d="M340 215 V300" stroke="#60A5FA" stroke-width="8" stroke-linecap="round" opacity="0.85" data-stream/>
        <path d="M260 300 C260 360 420 360 420 300 L400 388 H280 Z" fill="#E0F2FE" stroke="#0284C7" stroke-width="4"/>
        <rect x="284" y="342" width="112" height="42" fill="#BAE6FD" opacity="0.85" data-liquid/>
        <text x="262" y="72" font-size="18" font-weight="900" fill="#475569">漏斗 + 滤纸</text>
        <text x="278" y="404" font-size="18" font-weight="900" fill="#0284C7">烧杯：滤液</text>
        <g data-particles></g>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var p=root.querySelector('[data-p]'), s=root.querySelector('[data-s]');
    var pv=root.querySelector('[data-p-val]'), sv=root.querySelector('[data-s-val]');
    var res=root.querySelector('[data-result]'), residue=root.querySelector('[data-residue]');
    var stream=root.querySelector('[data-stream]'), liquid=root.querySelector('[data-liquid]'), particles=root.querySelector('[data-particles]');
    function update(){
      var P=Number(p.value), S=Number(s.value);
      pv.textContent=P.toFixed(0)+'%';sv.textContent=S.toFixed(0)+'%';
      var h=42+P*0.72;
      liquid.setAttribute('y',String(384-h));
      liquid.setAttribute('height',String(h));
      stream.setAttribute('opacity',P>=100?'0.15':'0.85');
      var residueH=12+S*0.55*(P/100);
      residue.setAttribute('d','M305 '+(194-residueH)+' Q340 '+(178-residueH*0.7)+' 375 '+(194-residueH)+' L372 195 L308 195 Z');
      var pts='';
      for(var i=0;i<18;i++){
        var x=290+(i%6)*18, y=120+Math.floor(i/6)*22;
        var op=Math.max(0,1-P/100);
        pts+='<circle cx="'+x+'" cy="'+y+'" r="'+(2+S/30)+'" fill="#92400E" opacity="'+op+'"/>';
      }
      particles.innerHTML=pts;
      res.innerHTML='过滤适合分离“不溶性固体 + 液体”。<br>滤纸上留下滤渣，烧杯中得到澄清滤液。操作要点：一贴、二低、三靠。';
    }
    p.oninput=update;s.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-crystallization',
    group: '🧫 基本实验操作',
    name: '蒸发结晶',
    emoji: '💎',
    desc: '模拟加热蒸发溶剂后晶体析出的过程',
    params: [
      { key: 'heat', label: '加热强度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
      { key: 'solute', label: '溶质浓度', type: 'number', min: 10, max: 90, step: 5, defaultValue: 60 },
    ],
    buildHTML: (params, rootId) => {
      const heat = num(params, 'heat', 55)
      const solute = num(params, 'solute', 60)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">💎 蒸发结晶：溶剂减少，晶体析出</div>
    <div class="ce-note">适合溶质溶解度随温度变化不大的溶液</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row">
        <div class="ce-label"><span>加热强度</span><span class="ce-value" data-h-val></span></div>
        <input data-h type="range" min="0" max="100" step="1" value="${n(heat)}">
      </div>
      <div class="ce-row">
        <div class="ce-label"><span>溶质浓度</span><span class="ce-value" data-c-val></span></div>
        <input data-c type="range" min="10" max="90" step="5" value="${n(solute)}">
      </div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <ellipse cx="340" cy="310" rx="150" ry="32" fill="#E5E7EB"/>
        <path d="M220 188 C230 330 450 330 460 188 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-water d="" fill="#BAE6FD" opacity="0.9"/>
        <g data-crystals></g>
        <g data-steam></g>
        <g data-flame></g>
        <rect x="250" y="332" width="180" height="18" rx="8" fill="#64748B"/>
        <text x="250" y="150" font-size="18" font-weight="900" fill="#475569">蒸发皿</text>
        <text x="462" y="116" data-state font-size="18" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var h=root.querySelector('[data-h]'), c=root.querySelector('[data-c]');
    var hv=root.querySelector('[data-h-val]'), cv=root.querySelector('[data-c-val]');
    var water=root.querySelector('[data-water]'), crystals=root.querySelector('[data-crystals]'), steam=root.querySelector('[data-steam]'), flame=root.querySelector('[data-flame]');
    var state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function update(){
      var H=Number(h.value), C=Number(c.value);
      hv.textContent=H.toFixed(0)+'%';cv.textContent=C.toFixed(0)+'%';
      var remain=Math.max(18,100-H*0.62);
      var y=300-remain;
      water.setAttribute('d','M240 '+y+' C270 '+(y+28)+' 410 '+(y+28)+' 440 '+y+' L458 188 C448 318 232 318 222 188 Z');
      var saturation=C+H*0.75;
      var count=Math.max(0,Math.floor((saturation-70)/6));
      var cs='';
      for(var i=0;i<count;i++){
        var x=270+(i%7)*22, yy=288-Math.floor(i/7)*17;
        cs+='<polygon points="'+x+','+yy+' '+(x+8)+','+(yy+6)+' '+x+','+(yy+12)+' '+(x-8)+','+(yy+6)+'" fill="#FACC15" stroke="#B45309" stroke-width="1"/>';
      }
      crystals.innerHTML=cs;
      var st='';
      for(var j=0;j<Math.floor(H/18);j++){
        var sx=290+j*26;
        st+='<path d="M'+sx+' 165 C'+(sx-22)+' 132 '+(sx+22)+' 120 '+sx+' 88" fill="none" stroke="#CBD5E1" stroke-width="4" opacity="0.7"/>';
      }
      steam.innerHTML=st;
      var fl='';
      for(var k=0;k<Math.floor(H/20);k++){
        var fx=290+k*28;
        fl+='<path d="M'+fx+' 355 C'+(fx-12)+' 335 '+fx+' 325 '+(fx+10)+' 307 C'+(fx+22)+' 330 '+(fx+14)+' 348 '+fx+' 355Z" fill="#F97316" opacity="0.85"/>';
      }
      flame.innerHTML=fl;
      state.textContent=count>0?'晶体析出':'继续蒸发';
      res.innerHTML='蒸发过程中溶剂减少，溶液逐渐接近饱和；达到过饱和后晶体析出。<br>当出现较多晶体时应停止加热，利用余热蒸干。';
    }
    h.oninput=update;c.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-co2',
    group: '🧪 气体制取与检验',
    name: 'CO₂ 制取与检验',
    emoji: '🫧',
    desc: '石灰石与稀盐酸反应制取 CO₂，并用澄清石灰水检验',
    params: [
      { key: 'acid', label: '稀盐酸滴加量', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
      { key: 'limestone', label: '石灰石用量', type: 'number', min: 10, max: 90, step: 5, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const acid = num(params, 'acid', 45)
      const limestone = num(params, 'limestone', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🫧 CO₂ 制取与检验</div>
    <div class="ce-note">CaCO₃ + 2HCl → CaCl₂ + H₂O + CO₂↑</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row">
        <div class="ce-label"><span>稀盐酸滴加量</span><span class="ce-value" data-a-val></span></div>
        <input data-a type="range" min="0" max="100" step="1" value="${n(acid)}">
      </div>
      <div class="ce-row">
        <div class="ce-label"><span>石灰石用量</span><span class="ce-value" data-l-val></span></div>
        <input data-l type="range" min="10" max="90" step="5" value="${n(limestone)}">
      </div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M150 125 L250 125 L230 315 L170 315 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M165 245 C180 265 220 265 235 245 L230 315 L170 315 Z" fill="#FDE68A" opacity="0.9"/>
        <g data-stones></g>
        <g data-bubbles></g>
        <path d="M245 138 C340 120 370 160 420 185" fill="none" stroke="#64748B" stroke-width="5"/>
        <path d="M420 185 L545 185 L520 320 L445 320 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-lime d="M438 275 C462 292 512 292 534 275 L522 320 L445 320 Z" fill="#BAE6FD" opacity="0.85"/>
        <text x="136" y="105" font-size="16" font-weight="900" fill="#475569">发生装置</text>
        <text x="430" y="165" font-size="16" font-weight="900" fill="#475569">澄清石灰水</text>
        <text x="300" y="70" data-co2 font-size="20" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var a=root.querySelector('[data-a]'), l=root.querySelector('[data-l]');
    var av=root.querySelector('[data-a-val]'), lv=root.querySelector('[data-l-val]');
    var stones=root.querySelector('[data-stones]'), bubbles=root.querySelector('[data-bubbles]'), lime=root.querySelector('[data-lime]');
    var co2=root.querySelector('[data-co2]'), res=root.querySelector('[data-result]');
    function update(){
      var A=Number(a.value), L=Number(l.value);
      av.textContent=A.toFixed(0)+'%';lv.textContent=L.toFixed(0)+'%';
      var rate=Math.min(100,A*0.75+L*0.45);
      var ss='';
      for(var i=0;i<Math.floor(L/8);i++){
        var x=178+(i%5)*11, y=292-Math.floor(i/5)*10;
        ss+='<polygon points="'+x+','+y+' '+(x+8)+','+(y+4)+' '+(x+4)+','+(y+10)+' '+(x-5)+','+(y+7)+'" fill="#A3A3A3" stroke="#737373" stroke-width="1"/>';
      }
      stones.innerHTML=ss;
      var bb='';
      for(var j=0;j<Math.floor(rate/7);j++){
        var bx=178+(j%5)*12, by=238-(j*11)%110;
        bb+='<circle cx="'+bx+'" cy="'+by+'" r="'+(3+j%3)+'" fill="none" stroke="#38BDF8" stroke-width="2" opacity="0.85"/>';
      }
      bubbles.innerHTML=bb;
      var cloudy=Math.min(1,rate/75);
      lime.setAttribute('fill',cloudy>0.75?'#E5E7EB':'#BAE6FD');
      lime.setAttribute('opacity',String(0.65+cloudy*0.3));
      co2.textContent=rate>20?'CO₂ 正在导入 →':'等待反应开始';
      res.innerHTML='现象：石灰石表面产生气泡，导出的气体使澄清石灰水变浑浊。<br>检验：CO₂ + Ca(OH)₂ → CaCO₃↓ + H₂O。';
    }
    a.oninput=update;l.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-neutralization',
    group: '⚗️ 反应现象',
    name: '酸碱中和与 pH 变化',
    emoji: '⚗️',
    desc: '向盐酸中滴加氢氧化钠，观察 pH 与指示剂颜色变化',
    params: [
      { key: 'base', label: 'NaOH 滴加量/mL', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
      { key: 'acid', label: '盐酸初始量/mL', type: 'number', min: 20, max: 100, step: 5, defaultValue: 50 },
    ],
    buildHTML: (params, rootId) => {
      const base = num(params, 'base', 45)
      const acid = num(params, 'acid', 50)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">⚗️ 酸碱中和：H⁺ + OH⁻ → H₂O</div>
    <div class="ce-note">pH 与颜色同步变化</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row">
        <div class="ce-label"><span>NaOH 滴加量</span><span class="ce-value" data-b-val></span></div>
        <input data-b type="range" min="0" max="100" step="1" value="${n(base)}">
      </div>
      <div class="ce-row">
        <div class="ce-label"><span>盐酸初始量</span><span class="ce-value" data-a-val></span></div>
        <input data-a type="range" min="20" max="100" step="5" value="${n(acid)}">
      </div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M270 96 L410 96 L382 340 L298 340 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-liquid d="M288 250 C310 268 370 268 392 250 L382 340 L298 340 Z" fill="#FCA5A5" opacity="0.9"/>
        <line x1="340" y1="55" x2="340" y2="120" stroke="#10B981" stroke-width="8" stroke-linecap="round"/>
        <circle data-drop cx="340" cy="145" r="7" fill="#10B981"/>
        <text x="286" y="78" font-size="16" font-weight="900" fill="#047857">NaOH 滴定管</text>
        <text x="294" y="370" font-size="16" font-weight="900" fill="#475569">盐酸 + 指示剂</text>
        <g data-scale></g>
        <text x="70" y="80" data-ph-text font-size="28" font-weight="900" fill="#075985"></text>
        <text x="70" y="116" data-state font-size="18" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var b=root.querySelector('[data-b]'), a=root.querySelector('[data-a]');
    var bv=root.querySelector('[data-b-val]'), av=root.querySelector('[data-a-val]');
    var liquid=root.querySelector('[data-liquid]'), drop=root.querySelector('[data-drop]');
    var phText=root.querySelector('[data-ph-text]'), state=root.querySelector('[data-state]');
    var scale=root.querySelector('[data-scale]'), res=root.querySelector('[data-result]');
    function color(pH){
      if(pH<4)return '#FCA5A5';
      if(pH<6.8)return '#FDBA74';
      if(pH<7.2)return '#A7F3D0';
      if(pH<10)return '#93C5FD';
      return '#C4B5FD';
    }
    function update(){
      var B=Number(b.value), A=Number(a.value);
      bv.textContent=B.toFixed(0)+' mL';av.textContent=A.toFixed(0)+' mL';
      var ratio=B/A;
      var pH= ratio<1 ? Math.max(1,7+Math.log10(Math.max(0.001,ratio))) : Math.min(14,7+Math.log10(Math.max(1,(ratio-1)*20+1)));
      var c=color(pH);
      liquid.setAttribute('fill',c);
      drop.setAttribute('cy',String(122+(B%25)*3));
      phText.textContent='pH ≈ '+pH.toFixed(1);
      state.textContent=pH<6.8?'酸性':pH>7.2?'碱性':'接近中性';
      var bars='';
      for(var i=1;i<=14;i++){
        bars+='<rect x="'+(64+i*16)+'" y="316" width="12" height="42" rx="4" fill="'+color(i)+'" opacity="'+(Math.abs(i-pH)<0.6?1:0.35)+'"/>';
      }
      scale.innerHTML=bars+'<text x="80" y="382" font-size="13" fill="#64748B">酸性 ← pH 色阶 → 碱性</text>';
      res.innerHTML='中和反应本质：H⁺ 与 OH⁻ 结合生成水。<br>当酸碱恰好完全反应时，pH 接近 7；继续滴加碱液后溶液转为碱性。';
    }
    b.oninput=update;a.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-precipitation',
    group: '⚗️ 反应现象',
    name: '沉淀反应',
    emoji: '🌫️',
    desc: '混合两种溶液，观察白色沉淀逐渐生成并沉降',
    params: [
      { key: 'mix', label: '混合比例', type: 'number', min: 0, max: 100, step: 1, defaultValue: 50 },
      { key: 'settle', label: '静置时间', type: 'number', min: 0, max: 100, step: 1, defaultValue: 35 },
    ],
    buildHTML: (params, rootId) => {
      const mix = num(params, 'mix', 50)
      const settle = num(params, 'settle', 35)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🌫️ 沉淀反应：Ag⁺ + Cl⁻ → AgCl↓</div>
    <div class="ce-note">以硝酸银与氯化钠反应为例</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row">
        <div class="ce-label"><span>混合比例</span><span class="ce-value" data-m-val></span></div>
        <input data-m type="range" min="0" max="100" step="1" value="${n(mix)}">
      </div>
      <div class="ce-row">
        <div class="ce-label"><span>静置时间</span><span class="ce-value" data-t-val></span></div>
        <input data-t type="range" min="0" max="100" step="1" value="${n(settle)}">
      </div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M250 80 L430 80 L395 350 L285 350 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M270 198 C300 225 380 225 410 198 L395 350 L285 350 Z" fill="#E0F2FE" opacity="0.75"/>
        <g data-cloud></g>
        <g data-bottom></g>
        <text x="235" y="64" font-size="17" font-weight="900" fill="#475569">混合溶液</text>
        <text x="78" y="90" font-size="18" font-weight="900" fill="#2563EB">NaCl(aq)</text>
        <text x="498" y="90" font-size="18" font-weight="900" fill="#7C3AED">AgNO₃(aq)</text>
        <path d="M150 105 C210 125 238 152 278 192" fill="none" stroke="#2563EB" stroke-width="5" stroke-linecap="round"/>
        <path d="M532 105 C470 125 444 152 402 192" fill="none" stroke="#7C3AED" stroke-width="5" stroke-linecap="round"/>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var m=root.querySelector('[data-m]'), t=root.querySelector('[data-t]');
    var mv=root.querySelector('[data-m-val]'), tv=root.querySelector('[data-t-val]');
    var cloud=root.querySelector('[data-cloud]'), bottom=root.querySelector('[data-bottom]'), res=root.querySelector('[data-result]');
    function update(){
      var M=Number(m.value), T=Number(t.value);
      mv.textContent=M.toFixed(0)+'%';tv.textContent=T.toFixed(0)+'%';
      var amount=Math.min(M,100-M)*2;
      var airborne=Math.max(0,amount*(1-T/100));
      var settled=amount-airborne;
      var c='', b='';
      for(var i=0;i<Math.floor(airborne/4);i++){
        var x=290+(i*37)%100, y=215+(i*23)%90;
        c+='<circle cx="'+x+'" cy="'+y+'" r="'+(3+i%4)+'" fill="#E5E7EB" stroke="#CBD5E1" opacity="0.86"/>';
      }
      for(var j=0;j<Math.floor(settled/3);j++){
        var bx=298+(j*17)%84, by=337-Math.floor(j/7)*5;
        b+='<circle cx="'+bx+'" cy="'+by+'" r="4" fill="#F8FAFC" stroke="#CBD5E1" opacity="0.95"/>';
      }
      cloud.innerHTML=c; bottom.innerHTML=b;
      res.innerHTML='离子反应：Ag⁺ 与 Cl⁻ 结合生成难溶 AgCl 白色沉淀。<br>静置后，悬浊的沉淀颗粒逐渐下沉。';
    }
    m.oninput=update;t.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-metal-acid',
    group: '⚗️ 反应现象',
    name: '金属与酸反应',
    emoji: '⚡',
    desc: '比较金属活动性：锌与稀盐酸反应产生氢气',
    params: [
      { key: 'acid', label: '酸浓度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
      { key: 'metal', label: '金属表面积', type: 'number', min: 10, max: 100, step: 5, defaultValue: 60 },
    ],
    buildHTML: (params, rootId) => {
      const acid = num(params, 'acid', 55)
      const metal = num(params, 'metal', 60)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">⚡ 金属与酸：Zn + 2HCl → ZnCl₂ + H₂↑</div>
    <div class="ce-note">酸浓度和表面积影响反应速率</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row">
        <div class="ce-label"><span>酸浓度</span><span class="ce-value" data-a-val></span></div>
        <input data-a type="range" min="0" max="100" step="1" value="${n(acid)}">
      </div>
      <div class="ce-row">
        <div class="ce-label"><span>金属表面积</span><span class="ce-value" data-m-val></span></div>
        <input data-m type="range" min="10" max="100" step="5" value="${n(metal)}">
      </div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M250 70 L430 70 L395 350 L285 350 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M270 180 C300 205 380 205 410 180 L395 350 L285 350 Z" fill="#FDE68A" opacity="0.8"/>
        <g data-metal></g>
        <g data-bubbles></g>
        <text x="284" y="54" font-size="17" font-weight="900" fill="#475569">试管：Zn + 稀盐酸</text>
        <text x="486" y="120" data-rate font-size="24" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var a=root.querySelector('[data-a]'), m=root.querySelector('[data-m]');
    var av=root.querySelector('[data-a-val]'), mv=root.querySelector('[data-m-val]');
    var metal=root.querySelector('[data-metal]'), bubbles=root.querySelector('[data-bubbles]'), rateText=root.querySelector('[data-rate]'), res=root.querySelector('[data-result]');
    function update(){
      var A=Number(a.value), M=Number(m.value);
      av.textContent=A.toFixed(0)+'%';mv.textContent=M.toFixed(0)+'%';
      var rate=A*M/100;
      var met='';
      for(var i=0;i<Math.floor(M/8);i++){
        var x=292+(i%7)*14, y=322-Math.floor(i/7)*10;
        met+='<rect x="'+x+'" y="'+y+'" width="18" height="8" rx="3" fill="#94A3B8" stroke="#64748B"/>';
      }
      metal.innerHTML=met;
      var bub='';
      for(var j=0;j<Math.floor(rate/4);j++){
        var bx=300+(j*19)%86, by=300-(j*13)%130;
        bub+='<circle cx="'+bx+'" cy="'+by+'" r="'+(3+j%3)+'" fill="none" stroke="#38BDF8" stroke-width="2" opacity="0.9"/>';
      }
      bubbles.innerHTML=bub;
      rateText.textContent=rate<20?'反应较慢':rate<55?'反应明显':'反应剧烈';
      res.innerHTML='现象：金属表面产生大量气泡，试管外壁可能略变热。<br>酸浓度越大、金属表面积越大，反应速率通常越快；收集气体可用燃着木条检验 H₂。';
    }
    a.oninput=update;m.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-electrolysis-water',
    group: '🔋 电化学',
    name: '电解水',
    emoji: '🔋',
    desc: '模拟电解水生成氢气和氧气，体积比约 2:1',
    params: [
      { key: 'current', label: '电流强度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
      { key: 'time', label: '通电时间', type: 'number', min: 0, max: 100, step: 1, defaultValue: 40 },
    ],
    buildHTML: (params, rootId) => {
      const current = num(params, 'current', 55)
      const time = num(params, 'time', 40)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🔋 电解水：2H₂O → 2H₂↑ + O₂↑</div>
    <div class="ce-note">负极产氢气，正极产氧气，体积比约 2:1</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row">
        <div class="ce-label"><span>电流强度</span><span class="ce-value" data-c-val></span></div>
        <input data-c type="range" min="0" max="100" step="1" value="${n(current)}">
      </div>
      <div class="ce-row">
        <div class="ce-label"><span>通电时间</span><span class="ce-value" data-t-val></span></div>
        <input data-t type="range" min="0" max="100" step="1" value="${n(time)}">
      </div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="170" y="130" width="340" height="210" rx="24" fill="#E0F2FE" stroke="#0284C7" stroke-width="4" opacity="0.75"/>
        <rect x="235" y="78" width="72" height="220" rx="20" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <rect x="373" y="78" width="72" height="220" rx="20" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <rect x="258" y="220" width="26" height="100" fill="#334155"/>
        <rect x="396" y="220" width="26" height="100" fill="#334155"/>
        <path d="M271 220 V180 H120 V94" fill="none" stroke="#334155" stroke-width="5"/>
        <path d="M409 220 V180 H560 V94" fill="none" stroke="#334155" stroke-width="5"/>
        <rect x="94" y="70" width="80" height="48" rx="12" fill="#F1F5F9" stroke="#64748B" stroke-width="3"/>
        <text x="114" y="101" font-size="22" font-weight="900" fill="#334155">—</text>
        <rect x="532" y="70" width="80" height="48" rx="12" fill="#F1F5F9" stroke="#64748B" stroke-width="3"/>
        <text x="558" y="101" font-size="22" font-weight="900" fill="#334155">＋</text>
        <rect data-h2 x="239" y="86" width="64" height="0" rx="16" fill="#BAE6FD" opacity="0.9"/>
        <rect data-o2 x="377" y="86" width="64" height="0" rx="16" fill="#FED7AA" opacity="0.9"/>
        <g data-bubbles></g>
        <text x="230" y="60" font-size="18" font-weight="900" fill="#0284C7">H₂</text>
        <text x="372" y="60" font-size="18" font-weight="900" fill="#EA580C">O₂</text>
        <text x="252" y="364" data-ratio font-size="24" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var c=root.querySelector('[data-c]'), t=root.querySelector('[data-t]');
    var cv=root.querySelector('[data-c-val]'), tv=root.querySelector('[data-t-val]');
    var h2=root.querySelector('[data-h2]'), o2=root.querySelector('[data-o2]'), bubbles=root.querySelector('[data-bubbles]');
    var ratio=root.querySelector('[data-ratio]'), res=root.querySelector('[data-result]');
    function update(){
      var C=Number(c.value), T=Number(t.value);
      cv.textContent=C.toFixed(0)+'%';tv.textContent=T.toFixed(0)+'%';
      var amount=C*T/100;
      var hH=Math.min(126,amount*1.25);
      var hO=Math.min(63,amount*0.625);
      h2.setAttribute('y',String(298-hH));h2.setAttribute('height',String(hH));
      o2.setAttribute('y',String(298-hO));o2.setAttribute('height',String(hO));
      var bb='';
      for(var i=0;i<Math.floor(amount/5);i++){
        var y1=214-(i*12)%90;
        bb+='<circle cx="'+(260+(i%4)*8)+'" cy="'+y1+'" r="3" fill="none" stroke="#38BDF8" stroke-width="2"/>';
        if(i%2===0)bb+='<circle cx="'+(398+(i%4)*8)+'" cy="'+(225-(i*9)%70)+'" r="3" fill="none" stroke="#FB923C" stroke-width="2"/>';
      }
      bubbles.innerHTML=bb;
      ratio.textContent='H₂ : O₂ ≈ 2 : 1';
      res.innerHTML='负极产生氢气，正极产生氧气；相同条件下，氢气体积约为氧气的 2 倍。<br>可用“带火星木条复燃”检验氧气，用“点燃有爆鸣声”检验氢气。';
    }
    c.oninput=update;t.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]

// ============================================================
// 查询辅助
// ============================================================

/** 模板分组列表 */
export function getChemExperimentGroups(): { group: string; items: ChemExperimentTemplate[] }[] {
  const groups: { group: string; items: ChemExperimentTemplate[] }[] = []
  for (const t of CHEM_EXPERIMENT_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) { g = { group: t.group, items: [] }; groups.push(g) }
    g.items.push(t)
  }
  return groups
}

/** 按ID查模板 */
export function findChemExperimentTemplate(id: string): ChemExperimentTemplate | undefined {
  return CHEM_EXPERIMENT_TEMPLATES.find(t => t.id === id)
}
