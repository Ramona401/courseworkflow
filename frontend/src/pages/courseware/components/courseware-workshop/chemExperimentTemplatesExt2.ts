/**
 * chemExperimentTemplatesExt2.ts — 化学实验第二批扩展模板
 *
 * 第二批新增：
 *   1. 粗盐提纯
 *   2. 燃烧与灭火
 *   3. 酸碱指示剂
 *   4. 金属活动性顺序
 */

import type { ChemExperimentTemplate, ChemExperimentParamValue } from './chemExperimentUtils'

function num(params: Record<string, ChemExperimentParamValue>, key: string, fallback: number): number {
  const v = params[key]
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback
}

function n(v: number): string {
  return parseFloat(v.toFixed(3)).toString()
}

function bool(params: Record<string, ChemExperimentParamValue>, key: string, fallback: boolean): boolean {
  const v = params[key]
  return typeof v === 'boolean' ? v : fallback
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

export const CHEM_EXPERIMENT_TEMPLATES_EXT2: ChemExperimentTemplate[] = [
  {
    id: 'chem-crude-salt-purification',
    group: '🧫 基本实验操作',
    name: '粗盐提纯',
    emoji: '🧂',
    desc: '串联溶解、过滤、蒸发三步，观察杂质分离和晶体获得',
    params: [
      { key: 'progress', label: '流程进度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
      { key: 'impurity', label: '不溶杂质含量', type: 'number', min: 10, max: 80, step: 5, defaultValue: 35 },
    ],
    buildHTML: (params, rootId) => {
      const progress = num(params, 'progress', 45)
      const impurity = num(params, 'impurity', 35)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🧂 粗盐提纯：溶解 → 过滤 → 蒸发</div>
    <div class="ce-note">把可溶性食盐与不溶性泥沙分离</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>流程进度</span><span class="ce-value" data-p-val></span></div><input data-p type="range" min="0" max="100" step="1" value="${n(progress)}"></div>
      <div class="ce-row"><div class="ce-label"><span>不溶杂质</span><span class="ce-value" data-i-val></span></div><input data-i type="range" min="10" max="80" step="5" value="${n(impurity)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <g data-step1></g>
        <g data-step2></g>
        <g data-step3></g>
        <text x="74" y="70" font-size="18" font-weight="900" fill="#075985">① 溶解</text>
        <text x="292" y="70" font-size="18" font-weight="900" fill="#075985">② 过滤</text>
        <text x="502" y="70" font-size="18" font-weight="900" fill="#075985">③ 蒸发</text>
        <path d="M212 202 H260" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <path d="M245 184 L264 202 L245 220" fill="none" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <path d="M432 202 H480" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <path d="M465 184 L484 202 L465 220" fill="none" stroke="#10B981" stroke-width="5" stroke-linecap="round"/>
        <text x="212" y="370" data-state font-size="22" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var p=root.querySelector('[data-p]'), im=root.querySelector('[data-i]');
    var pv=root.querySelector('[data-p-val]'), iv=root.querySelector('[data-i-val]');
    var s1=root.querySelector('[data-step1]'), s2=root.querySelector('[data-step2]'), s3=root.querySelector('[data-step3]');
    var state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function update(){
      var P=Number(p.value), I=Number(im.value);
      pv.textContent=P.toFixed(0)+'%'; iv.textContent=I.toFixed(0)+'%';
      var dissolve=Math.min(1,P/33), filter=Math.max(0,Math.min(1,(P-33)/34)), evap=Math.max(0,(P-67)/33);
      var mud='';
      for(var a=0;a<Math.floor(I/8);a++){
        mud+='<circle cx="'+(95+(a%5)*16)+'" cy="'+(258-Math.floor(a/5)*13)+'" r="5" fill="#92400E" opacity="'+(1-dissolve*0.25)+'"/>';
      }
      s1.innerHTML='<path d="M62 120 L190 120 L166 306 L86 306 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M76 220 C98 240 154 240 176 220 L166 306 L86 306 Z" fill="#BAE6FD" opacity="0.86"/>'+mud+'<text x="82" y="332" font-size="15" font-weight="800" fill="#475569">搅拌溶解</text>';
      var residueH=filter*I*0.45;
      s2.innerHTML='<path d="M284 110 L408 110 L378 232 L314 232 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M302 130 L390 130 L368 212 L324 212 Z" fill="#FDE68A" opacity="0.62"/><path d="M322 '+(212-residueH)+' Q346 '+(200-residueH)+' 370 '+(212-residueH)+' L368 212 L324 212 Z" fill="#92400E" opacity="'+filter+'"/><path d="M346 232 V302" stroke="#60A5FA" stroke-width="'+(4+filter*5)+'" stroke-linecap="round" opacity="'+filter+'"/><path d="M288 302 C288 342 402 342 402 302 L388 356 H302 Z" fill="#E0F2FE" stroke="#0284C7" stroke-width="4"/><text x="308" y="380" font-size="15" font-weight="800" fill="#475569">滤液较澄清</text>';
      var crystals='';
      for(var c=0;c<Math.floor(evap*16);c++){
        var x=520+(c%6)*19, y=302-Math.floor(c/6)*13;
        crystals+='<polygon points="'+x+','+y+' '+(x+8)+','+(y+5)+' '+x+','+(y+11)+' '+(x-8)+','+(y+5)+'" fill="#FACC15" stroke="#B45309" stroke-width="1"/>';
      }
      s3.innerHTML='<ellipse cx="560" cy="306" rx="98" ry="24" fill="#E5E7EB"/><path d="M488 206 C494 316 626 316 632 206 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M506 '+(278-evap*54)+' C532 '+(298-evap*54)+' 588 '+(298-evap*54)+' 614 '+(278-evap*54)+' L628 206 C620 312 500 312 492 206 Z" fill="#BAE6FD" opacity="'+(0.85-evap*0.45)+'"/>'+crystals+'<text x="520" y="380" font-size="15" font-weight="800" fill="#475569">蒸发结晶</text>';
      state.textContent=P<33?'当前：溶解粗盐':P<67?'当前：过滤除去泥沙':'当前：蒸发得到晶体';
      res.innerHTML='粗盐提纯通常按“溶解、过滤、蒸发”进行。<br>过滤除去不溶性泥沙，蒸发滤液后得到氯化钠晶体。';
    }
    p.oninput=update;im.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-combustion-extinguish',
    group: '⚗️ 反应现象',
    name: '燃烧与灭火条件',
    emoji: '🔥',
    desc: '调节可燃物、氧气和温度，理解燃烧三要素与灭火原理',
    params: [
      { key: 'oxygen', label: '氧气浓度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 65 },
      { key: 'temp', label: '温度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 70 },
      { key: 'cover', label: '盖上烧杯隔绝空气', type: 'boolean', defaultValue: false },
    ],
    buildHTML: (params, rootId) => {
      const oxygen = num(params, 'oxygen', 65)
      const temp = num(params, 'temp', 70)
      const cover = bool(params, 'cover', false)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🔥 燃烧三要素：可燃物 + 氧气 + 温度达到着火点</div>
    <div class="ce-note">灭火就是破坏其中一个条件</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>氧气浓度</span><span class="ce-value" data-o-val></span></div><input data-o type="range" min="0" max="100" step="1" value="${n(oxygen)}"></div>
      <div class="ce-row"><div class="ce-label"><span>温度</span><span class="ce-value" data-t-val></span></div><input data-t type="range" min="0" max="100" step="1" value="${n(temp)}"></div>
      <div class="ce-row"><button data-cover>${cover ? '移开烧杯' : '盖上烧杯'}</button></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <ellipse cx="340" cy="322" rx="170" ry="28" fill="#E5E7EB"/>
        <rect x="250" y="286" width="180" height="28" rx="10" fill="#92400E"/>
        <text x="296" y="352" font-size="18" font-weight="900" fill="#475569">可燃物</text>
        <g data-flame></g>
        <g data-cover-g></g>
        <g data-o2></g>
        <text x="74" y="72" data-state font-size="28" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var o=root.querySelector('[data-o]'), t=root.querySelector('[data-t]'), btn=root.querySelector('[data-cover]');
    var ov=root.querySelector('[data-o-val]'), tv=root.querySelector('[data-t-val]');
    var flame=root.querySelector('[data-flame]'), coverG=root.querySelector('[data-cover-g]'), o2=root.querySelector('[data-o2]');
    var state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    var covered=${cover ? 'true' : 'false'};
    btn.onclick=function(){covered=!covered;btn.textContent=covered?'移开烧杯':'盖上烧杯';update();};
    function update(){
      var O=Number(o.value), T=Number(t.value);
      ov.textContent=O.toFixed(0)+'%'; tv.textContent=T.toFixed(0)+'%';
      var effectiveO=covered?O*0.18:O;
      var burning=effectiveO>35 && T>45;
      var strength=burning?Math.min(1,(effectiveO+T-80)/90):0;
      var f='';
      for(var i=0;i<Math.floor(strength*12);i++){
        var x=284+(i%6)*22, base=286-Math.floor(i/6)*18, h=36+strength*58-(i%3)*8;
        f+='<path d="M'+x+' '+base+' C'+(x-18)+' '+(base-h*0.55)+' '+x+' '+(base-h*0.78)+' '+(x+10)+' '+(base-h)+' C'+(x+30)+' '+(base-h*0.55)+' '+(x+18)+' '+(base-h*0.18)+' '+x+' '+base+'Z" fill="'+(i%2?'#F97316':'#FACC15')+'" opacity="0.86"/>';
      }
      flame.innerHTML=f;
      coverG.innerHTML=covered?'<path d="M246 94 L434 94 L404 304 L276 304 Z" fill="#E0F2FE" stroke="#0284C7" stroke-width="5" opacity="0.36"/><text x="270" y="88" font-size="18" font-weight="900" fill="#0284C7">倒扣烧杯</text>':'';
      var dots='';
      for(var j=0;j<Math.floor(effectiveO/8);j++){
        dots+='<circle cx="'+(82+(j*47)%520)+'" cy="'+(110+(j*29)%150)+'" r="5" fill="#38BDF8" opacity="0.45"/>';
      }
      o2.innerHTML=dots;
      state.textContent=burning?'正在燃烧':'燃烧停止';
      res.innerHTML=burning?'燃烧三要素同时满足：有可燃物、有氧气、温度达到着火点。<br>火焰越旺，表示氧气和温度条件越充分。':'燃烧停止：可能因氧气不足、温度低于着火点或隔绝空气。<br>灭火方法本质是破坏燃烧条件之一。';
    }
    o.oninput=update;t.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-indicator-color',
    group: '⚗️ 反应现象',
    name: '酸碱指示剂',
    emoji: '🌈',
    desc: '调节 pH，观察紫色石蕊或酚酞在酸碱环境中的颜色变化',
    params: [
      { key: 'ph', label: 'pH', type: 'number', min: 1, max: 14, step: 0.1, defaultValue: 7 },
      { key: 'phenol', label: '使用酚酞', type: 'boolean', defaultValue: false },
    ],
    buildHTML: (params, rootId) => {
      const ph = num(params, 'ph', 7)
      const phenol = bool(params, 'phenol', false)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🌈 酸碱指示剂：颜色反映溶液酸碱性</div>
    <div class="ce-note">紫色石蕊：酸红碱蓝；酚酞：碱性变红</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>pH</span><span class="ce-value" data-ph-val></span></div><input data-ph type="range" min="1" max="14" step="0.1" value="${n(ph)}"></div>
      <div class="ce-row"><button data-switch>${phenol ? '切换为石蕊' : '切换为酚酞'}</button></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M250 80 L430 80 L396 344 L284 344 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-liquid d="M272 210 C304 238 376 238 408 210 L396 344 L284 344 Z" fill="#A78BFA" opacity="0.9"/>
        <line x1="340" y1="42" x2="340" y2="122" stroke="#059669" stroke-width="7" stroke-linecap="round"/>
        <circle cx="340" cy="132" r="8" data-drop fill="#059669"/>
        <g data-scale></g>
        <text x="62" y="78" data-name font-size="24" font-weight="900" fill="#075985"></text>
        <text x="62" y="114" data-state font-size="20" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var ph=root.querySelector('[data-ph]'), sw=root.querySelector('[data-switch]');
    var phv=root.querySelector('[data-ph-val]'), liq=root.querySelector('[data-liquid]'), scale=root.querySelector('[data-scale]');
    var name=root.querySelector('[data-name]'), state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    var phenol=${phenol ? 'true' : 'false'};
    sw.onclick=function(){phenol=!phenol;sw.textContent=phenol?'切换为石蕊':'切换为酚酞';update();};
    function litmusColor(p){ if(p<6.5)return '#EF4444'; if(p>7.5)return '#2563EB'; return '#8B5CF6'; }
    function phenolColor(p){ return p>8.2 ? '#EC4899' : '#F8FAFC'; }
    function update(){
      var P=Number(ph.value), acid=P<6.5, alk=P>7.5;
      var color=phenol?phenolColor(P):litmusColor(P);
      phv.textContent=P.toFixed(1);
      liq.setAttribute('fill',color);
      name.textContent=phenol?'酚酞指示剂':'紫色石蕊试液';
      state.textContent=acid?'酸性':alk?'碱性':'接近中性';
      var bars='';
      for(var i=1;i<=14;i++){
        var c=phenol?phenolColor(i):litmusColor(i);
        bars+='<rect x="'+(74+i*34)+'" y="326" width="26" height="36" rx="5" fill="'+c+'" opacity="'+(Math.abs(i-P)<0.6?1:0.35)+'"/>';
      }
      scale.innerHTML=bars+'<text x="98" y="388" font-size="13" fill="#64748B">pH 1 ← 酸性　　　中性　　　碱性 → pH 14</text>';
      res.innerHTML=phenol?'酚酞遇酸性和中性溶液通常不变色，遇碱性溶液变红。<br>适合判断碱性是否出现。':'紫色石蕊遇酸变红，遇碱变蓝，中性附近保持紫色。<br>适合整体判断酸性、中性、碱性。';
    }
    ph.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-metal-activity-series',
    group: '⚗️ 反应现象',
    name: '金属活动性顺序',
    emoji: '🥇',
    desc: '选择金属与酸反应，比较产生氢气的快慢，理解活动性强弱',
    params: [
      { key: 'metal', label: '金属选择', type: 'number', min: 0, max: 4, step: 1, defaultValue: 1, hint: '0=Mg, 1=Zn, 2=Fe, 3=Cu, 4=Ag' },
      { key: 'acid', label: '酸浓度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const metal = num(params, 'metal', 1)
      const acid = num(params, 'acid', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🥇 金属活动性：Mg > Zn > Fe > H > Cu > Ag</div>
    <div class="ce-note">排在氢前的金属通常能与酸反应放出氢气</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>金属</span><span class="ce-value" data-m-val></span></div><input data-m type="range" min="0" max="4" step="1" value="${n(metal)}"></div>
      <div class="ce-row"><div class="ce-label"><span>酸浓度</span><span class="ce-value" data-a-val></span></div><input data-a type="range" min="0" max="100" step="1" value="${n(acid)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M258 74 L422 74 L390 340 L290 340 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M276 190 C306 215 374 215 404 190 L390 340 L290 340 Z" fill="#FDE68A" opacity="0.78"/>
        <g data-metal></g>
        <g data-bubbles></g>
        <text x="288" y="58" font-size="17" font-weight="900" fill="#475569">金属 + 稀盐酸</text>
        <text x="470" y="130" data-rate font-size="26" font-weight="900" fill="#047857"></text>
        <text x="54" y="354" data-series font-size="22" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var m=root.querySelector('[data-m]'), a=root.querySelector('[data-a]');
    var mv=root.querySelector('[data-m-val]'), av=root.querySelector('[data-a-val]');
    var metalG=root.querySelector('[data-metal]'), bub=root.querySelector('[data-bubbles]');
    var rateText=root.querySelector('[data-rate]'), series=root.querySelector('[data-series]'), res=root.querySelector('[data-result]');
    var names=['Mg 镁','Zn 锌','Fe 铁','Cu 铜','Ag 银'];
    var strength=[1.0,0.72,0.42,0,0];
    var colors=['#94A3B8','#A3A3A3','#64748B','#B45309','#CBD5E1'];
    function update(){
      var M=Math.round(Number(m.value)), A=Number(a.value);
      mv.textContent=names[M]; av.textContent=A.toFixed(0)+'%';
      var rate=strength[M]*A;
      var mg='';
      for(var i=0;i<10;i++){
        mg+='<rect x="'+(300+(i%5)*16)+'" y="'+(306-Math.floor(i/5)*12)+'" width="22" height="9" rx="3" fill="'+colors[M]+'" stroke="#475569" opacity="0.95"/>';
      }
      metalG.innerHTML=mg;
      var bb='';
      for(var j=0;j<Math.floor(rate/5);j++){
        bb+='<circle cx="'+(310+(j*19)%76)+'" cy="'+(292-(j*15)%140)+'" r="'+(3+j%3)+'" fill="none" stroke="#38BDF8" stroke-width="2" opacity="0.88"/>';
      }
      bub.innerHTML=bb;
      rateText.textContent=rate>60?'反应剧烈':rate>25?'反应明显':rate>5?'反应缓慢':'基本不反应';
      series.textContent='Mg > Zn > Fe > H > Cu > Ag';
      res.innerHTML=rate>0?'该金属排在氢前，能与稀酸反应产生氢气；活动性越强，气泡通常越明显。<br>可用气泡多少和反应快慢比较金属活动性。':'铜、银排在氢后，通常不能与稀盐酸反应放出氢气。<br>这说明金属活动性弱于氢。';
    }
    m.oninput=update;a.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
