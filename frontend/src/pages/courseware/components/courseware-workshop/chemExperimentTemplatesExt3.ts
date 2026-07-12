/**
 * chemExperimentTemplatesExt3.ts — 化学实验第三批扩展模板
 *
 * 第三批新增：
 *   1. 气体收集方法选择
 *   2. pH 试纸比色
 *   3. 碳酸盐检验
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

export const CHEM_EXPERIMENT_TEMPLATES_EXT3: ChemExperimentTemplate[] = [
  {
    id: 'chem-gas-collection-methods',
    group: '🧪 气体制取与检验',
    name: '气体收集方法选择',
    emoji: '🫙',
    desc: '根据气体密度和溶解性，判断排水法、向上排空气法、向下排空气法',
    params: [
      { key: 'gas', label: '气体选择', type: 'number', min: 0, max: 3, step: 1, defaultValue: 0, hint: '0=O₂, 1=CO₂, 2=H₂, 3=NH₃' },
      { key: 'purity', label: '收集进度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const gas = num(params, 'gas', 0)
      const purity = num(params, 'purity', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🫙 气体收集：由密度与溶解性决定方法</div>
    <div class="ce-note">排水法/向上排空气/向下排空气</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>气体</span><span class="ce-value" data-g-val></span></div><input data-g type="range" min="0" max="3" step="1" value="${n(gas)}"></div>
      <div class="ce-row"><div class="ce-label"><span>收集进度</span><span class="ce-value" data-p-val></span></div><input data-p type="range" min="0" max="100" step="1" value="${n(purity)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <g data-method></g>
        <g data-gas></g>
        <text x="52" y="58" data-name font-size="28" font-weight="900" fill="#047857"></text>
        <text x="52" y="94" data-rule font-size="18" font-weight="900" fill="#475569"></text>
        <text x="52" y="364" data-choice font-size="24" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var ge=root.querySelector('[data-g]'), pe=root.querySelector('[data-p]');
    var gv=root.querySelector('[data-g-val]'), pv=root.querySelector('[data-p-val]');
    var method=root.querySelector('[data-method]'), gasG=root.querySelector('[data-gas]');
    var name=root.querySelector('[data-name]'), rule=root.querySelector('[data-rule]'), choice=root.querySelector('[data-choice]'), res=root.querySelector('[data-result]');
    var gases=[
      {n:'O₂ 氧气', m:'排水法或向上排空气法', r:'不易溶于水，密度比空气大', color:'#38BDF8', type:'water'},
      {n:'CO₂ 二氧化碳', m:'向上排空气法', r:'能溶于水且密度比空气大', color:'#94A3B8', type:'up'},
      {n:'H₂ 氢气', m:'排水法或向下排空气法', r:'难溶于水，密度比空气小', color:'#60A5FA', type:'down'},
      {n:'NH₃ 氨气', m:'向下排空气法', r:'极易溶于水且密度比空气小', color:'#A78BFA', type:'down'}
    ];
    function update(){
      var G=Math.round(Number(ge.value)), P=Number(pe.value), info=gases[G];
      gv.textContent=info.n; pv.textContent=P.toFixed(0)+'%';
      name.textContent=info.n;
      rule.textContent=info.r;
      choice.textContent='推荐：'+info.m;
      var base='';
      if(info.type==='water'){
        base='<rect x="210" y="250" width="300" height="90" rx="18" fill="#BAE6FD" stroke="#0284C7" stroke-width="4" opacity="0.8"/><path d="M320 110 L420 110 L400 300 L340 300 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M190 220 C250 198 300 218 340 250" fill="none" stroke="#64748B" stroke-width="5"/><text x="250" y="386" font-size="17" font-weight="900" fill="#0284C7">排水集气</text>';
      }else if(info.type==='up'){
        base='<path d="M310 104 L450 104 L424 330 L336 330 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M176 266 C236 240 280 260 336 300" fill="none" stroke="#64748B" stroke-width="5"/><text x="284" y="386" font-size="17" font-weight="900" fill="#475569">向上排空气</text>';
      }else{
        base='<path d="M310 330 L450 330 L424 104 L336 104 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M176 180 C238 206 286 184 336 142" fill="none" stroke="#64748B" stroke-width="5"/><text x="284" y="386" font-size="17" font-weight="900" fill="#475569">向下排空气</text>';
      }
      method.innerHTML=base;
      var dots='';
      for(var i=0;i<Math.floor(P/5);i++){
        var x=344+(i*23)%78;
        var y=info.type==='down' ? 300-(i*11)%180 : 126+(i*13)%172;
        dots+='<circle cx="'+x+'" cy="'+y+'" r="'+(4+i%3)+'" fill="'+info.color+'" opacity="0.58"/>';
      }
      gasG.innerHTML=dots;
      res.innerHTML='判断气体收集方法要看两点：是否易溶于水、密度与空气相比大小。<br>不易溶于水可用排水法；密度大用向上排空气法，密度小用向下排空气法。';
    }
    ge.oninput=update;pe.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-ph-paper',
    group: '⚗️ 反应现象',
    name: 'pH 试纸比色',
    emoji: '📏',
    desc: '把试纸颜色与标准比色卡对照，估测溶液 pH',
    params: [
      { key: 'ph', label: '溶液 pH', type: 'number', min: 1, max: 14, step: 1, defaultValue: 6 },
      { key: 'wet', label: '试纸先用水润湿', type: 'boolean', defaultValue: false, hint: '错误操作：可能稀释待测液，导致读数偏差' },
    ],
    buildHTML: (params, rootId) => {
      const ph = num(params, 'ph', 6)
      const wet = bool(params, 'wet', false)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">📏 pH 试纸：蘸取待测液后与标准色卡比色</div>
    <div class="ce-note">不能把试纸直接伸入试剂瓶</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>真实 pH</span><span class="ce-value" data-ph-val></span></div><input data-ph type="range" min="1" max="14" step="1" value="${n(ph)}"></div>
      <div class="ce-row"><button data-wet>${wet ? '改为干试纸' : '先润湿试纸'}</button></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="122" y="118" width="436" height="54" rx="12" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="4"/>
        <rect x="190" y="218" width="300" height="50" rx="8" data-paper fill="#A7F3D0" stroke="#64748B" stroke-width="3"/>
        <text x="244" y="207" font-size="18" font-weight="900" fill="#475569">蘸取后的 pH 试纸</text>
        <g data-card></g>
        <text x="184" y="328" data-reading font-size="28" font-weight="900" fill="#075985"></text>
        <text x="184" y="366" data-state font-size="19" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var ph=root.querySelector('[data-ph]'), btn=root.querySelector('[data-wet]');
    var phv=root.querySelector('[data-ph-val]'), paper=root.querySelector('[data-paper]'), card=root.querySelector('[data-card]');
    var reading=root.querySelector('[data-reading]'), state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    var wet=${wet ? 'true' : 'false'};
    function color(p){
      if(p<=2)return '#EF4444';
      if(p<=4)return '#F97316';
      if(p<=6)return '#FACC15';
      if(p===7)return '#22C55E';
      if(p<=9)return '#38BDF8';
      if(p<=11)return '#2563EB';
      return '#7C3AED';
    }
    btn.onclick=function(){wet=!wet;btn.textContent=wet?'改为干试纸':'先润湿试纸';update();};
    function update(){
      var P=Number(ph.value);
      var read=wet ? Math.round(P + (7-P)*0.35) : P;
      phv.textContent=P.toFixed(0);
      paper.setAttribute('fill',color(read));
      var html='';
      for(var i=1;i<=14;i++){
        html+='<rect x="'+(132+(i-1)*30)+'" y="130" width="24" height="30" rx="4" fill="'+color(i)+'" opacity="'+(i===read?1:0.62)+'"/><text x="'+(136+(i-1)*30)+'" y="112" font-size="10" fill="#64748B">'+i+'</text>';
      }
      card.innerHTML=html;
      reading.textContent='读数约 pH = '+read;
      state.textContent=read<7?'酸性':read>7?'碱性':'中性';
      res.innerHTML=wet?'错误提醒：pH 试纸不能先润湿。润湿会稀释待测液，酸/碱溶液读数可能向 7 靠近。':'正确操作：用玻璃棒蘸取待测液点到干燥 pH 试纸上，再与标准比色卡对照。';
    }
    ph.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'chem-carbonate-test',
    group: '🧪 气体制取与检验',
    name: '碳酸盐检验',
    emoji: '🪨',
    desc: '碳酸盐遇酸产生 CO₂，通入澄清石灰水变浑浊',
    params: [
      { key: 'acid', label: '稀盐酸滴加量', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
      { key: 'carbonate', label: '碳酸盐样品量', type: 'number', min: 10, max: 100, step: 5, defaultValue: 50 },
    ],
    buildHTML: (params, rootId) => {
      const acid = num(params, 'acid', 55)
      const carbonate = num(params, 'carbonate', 50)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="ce-head">
    <div class="ce-title">🪨 碳酸盐检验：酸 + 碳酸盐 → CO₂↑</div>
    <div class="ce-note">CO₂ 使澄清石灰水变浑浊</div>
  </div>
  <div class="ce-body">
    <div class="ce-controls">
      <div class="ce-row"><div class="ce-label"><span>稀盐酸</span><span class="ce-value" data-a-val></span></div><input data-a type="range" min="0" max="100" step="1" value="${n(acid)}"></div>
      <div class="ce-row"><div class="ce-label"><span>样品量</span><span class="ce-value" data-c-val></span></div><input data-c type="range" min="10" max="100" step="5" value="${n(carbonate)}"></div>
      <div class="ce-result" data-result></div>
    </div>
    <div class="ce-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M132 94 L262 94 L238 310 L156 310 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M150 210 C176 230 220 230 246 210 L238 310 L156 310 Z" fill="#FDE68A" opacity="0.82"/>
        <g data-stones></g>
        <g data-bubbles></g>
        <path d="M260 112 C350 94 390 144 430 190" fill="none" stroke="#64748B" stroke-width="5"/>
        <path d="M430 190 L560 190 L536 318 L456 318 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-lime d="M444 266 C470 286 520 286 548 266 L536 318 L456 318 Z" fill="#BAE6FD" opacity="0.86"/>
        <text x="114" y="72" font-size="17" font-weight="900" fill="#475569">样品 + 稀盐酸</text>
        <text x="420" y="166" font-size="17" font-weight="900" fill="#475569">澄清石灰水</text>
        <text x="270" y="58" data-main font-size="24" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var a=root.querySelector('[data-a]'), c=root.querySelector('[data-c]');
    var av=root.querySelector('[data-a-val]'), cv=root.querySelector('[data-c-val]');
    var stones=root.querySelector('[data-stones]'), bubbles=root.querySelector('[data-bubbles]'), lime=root.querySelector('[data-lime]');
    var main=root.querySelector('[data-main]'), res=root.querySelector('[data-result]');
    function update(){
      var A=Number(a.value), C=Number(c.value), rate=Math.min(100,A*0.7+C*0.5);
      av.textContent=A.toFixed(0)+'%'; cv.textContent=C.toFixed(0)+'%';
      var ss='';
      for(var i=0;i<Math.floor(C/8);i++){
        ss+='<polygon points="'+(170+(i%5)*13)+','+(290-Math.floor(i/5)*12)+' '+(181+(i%5)*13)+','+(294-Math.floor(i/5)*12)+' '+(176+(i%5)*13)+','+(304-Math.floor(i/5)*12)+' '+(164+(i%5)*13)+','+(300-Math.floor(i/5)*12)+'" fill="#A3A3A3" stroke="#737373" stroke-width="1"/>';
      }
      stones.innerHTML=ss;
      var bb='';
      for(var j=0;j<Math.floor(rate/6);j++){
        bb+='<circle cx="'+(178+(j*17)%54)+'" cy="'+(204-(j*13)%112)+'" r="'+(3+j%3)+'" fill="none" stroke="#38BDF8" stroke-width="2" opacity="0.9"/>';
      }
      bubbles.innerHTML=bb;
      lime.setAttribute('fill',rate>45?'#E5E7EB':'#BAE6FD');
      lime.setAttribute('opacity',String(0.7+Math.min(0.25,rate/300)));
      main.textContent=rate>35?'产生 CO₂ 并使石灰水浑浊':'等待明显现象';
      res.innerHTML='碳酸盐遇酸产生二氧化碳；二氧化碳通入澄清石灰水，生成碳酸钙白色沉淀而变浑浊。<br>常用于检验 CO₃²⁻ 或碳酸盐样品。';
    }
    a.oninput=update;c.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
