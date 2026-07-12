/**
 * physicsLabTemplatesExt5.ts — 物理实验室第五批扩展模板
 *
 * 第五批新增：
 *   1. 液体压强
 *   2. 滑轮组
 *   3. 比热容升温
 */

import type { PhysicsLabTemplate, PhysicsLabParamValue } from './physicsLabUtils'

function num(params: Record<string, PhysicsLabParamValue>, key: string, fallback: number): number {
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
    + '#' + rootId + ' .pl-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#E0F2FE,#F0F9FF);border-bottom:1px solid #E5E7EB;box-sizing:border-box;}\n'
    + '#' + rootId + ' .pl-title{font-size:15px;font-weight:800;color:#075985;}\n'
    + '#' + rootId + ' .pl-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .pl-body{height:calc(100% - 46px);display:grid;grid-template-columns:220px 1fr;min-height:0;}\n'
    + '#' + rootId + ' .pl-controls{padding:14px;border-right:1px solid #E5E7EB;background:#F8FAFC;box-sizing:border-box;overflow:auto;}\n'
    + '#' + rootId + ' .pl-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .pl-row{margin-bottom:13px;}\n'
    + '#' + rootId + ' .pl-label{display:flex;justify-content:space-between;gap:8px;font-size:12px;font-weight:700;color:#334155;margin-bottom:6px;}\n'
    + '#' + rootId + ' .pl-value{font-weight:800;color:#0284C7;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9;}\n'
    + '#' + rootId + ' button{border:none;border-radius:10px;padding:8px 14px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:12px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .pl-result{padding:10px 12px;border-radius:10px;background:#E0F2FE;color:#075985;font-size:12px;line-height:1.6;font-weight:600;}\n'
    + '#' + rootId + ' svg{width:100%;height:100%;display:block;}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const PHYSICS_LAB_TEMPLATES_EXT5: PhysicsLabTemplate[] = [
  {
    id: 'lab-liquid-pressure',
    group: '⚙️ 力与机械',
    name: '液体压强',
    emoji: '🌊',
    desc: '调节液体密度和深度，观察 p=ρgh 随深度增大',
    params: [
      { key: 'depth', label: '深度/cm', type: 'number', min: 5, max: 100, step: 1, defaultValue: 45 },
      { key: 'rho', label: '液体密度', type: 'number', min: 0.6, max: 1.4, step: 0.1, defaultValue: 1 },
    ],
    buildHTML: (params, rootId) => {
      const depth = num(params, 'depth', 45)
      const rho = num(params, 'rho', 1)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🌊 液体压强：p = ρgh</div>
    <div class="pl-note">同种液体中，深度越大压强越大</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>深度</span><span class="pl-value" data-d-val></span></div><input data-d type="range" min="5" max="100" step="1" value="${n(depth)}"></div>
      <div class="pl-row"><div class="pl-label"><span>液体密度</span><span class="pl-value" data-r-val></span></div><input data-r type="range" min="0.6" max="1.4" step="0.1" value="${n(rho)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M188 74 L492 74 L450 344 L230 344 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path d="M210 142 C270 176 410 176 470 142 L450 344 L230 344 Z" fill="#BAE6FD" opacity="0.82"/>
        <circle data-point cx="340" cy="224" r="9" fill="#DC2626"/>
        <line x1="512" y1="142" x2="512" y2="340" stroke="#64748B" stroke-width="3"/>
        <line data-depth x1="512" y1="142" x2="512" y2="224" stroke="#0284C7" stroke-width="7" stroke-linecap="round"/>
        <path data-jet d="" fill="none" stroke="#38BDF8" stroke-width="6" stroke-linecap="round"/>
        <text x="532" y="238" data-p-text font-size="22" font-weight="900" fill="#075985"></text>
        <text x="80" y="82" data-state font-size="24" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var d=root.querySelector('[data-d]'), r=root.querySelector('[data-r]');
    var dv=root.querySelector('[data-d-val]'), rv=root.querySelector('[data-r-val]');
    var point=root.querySelector('[data-point]'), dep=root.querySelector('[data-depth]'), jet=root.querySelector('[data-jet]');
    var pt=root.querySelector('[data-p-text]'), state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function update(){
      var D=Number(d.value), R=Number(r.value), P=R*9.8*D/100;
      var y=142+D*1.92;
      dv.textContent=D.toFixed(0)+' cm'; rv.textContent=R.toFixed(1);
      point.setAttribute('cy',String(y));
      dep.setAttribute('y2',String(y));
      var len=20+P*20;
      jet.setAttribute('d','M340 '+y+' C'+(380+len*0.4)+' '+(y-18)+' '+(430+len)+' '+(y+8)+' '+(466+len)+' '+(y+22));
      pt.textContent='p≈'+P.toFixed(1)+' kPa';
      state.textContent=D>70?'深处压强较大':'浅处压强较小';
      res.innerHTML='液体内部压强随深度和液体密度增大而增大。<br>同一深度处，液体向各个方向都有压强。';
    }
    d.oninput=update;r.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-pulley-system',
    group: '⚙️ 力与机械',
    name: '滑轮组',
    emoji: '🪝',
    desc: '调节重物、绳子段数和摩擦，观察省力与费距离',
    params: [
      { key: 'load', label: '重物重力/N', type: 'number', min: 20, max: 300, step: 10, defaultValue: 120 },
      { key: 'segments', label: '承重绳段数', type: 'number', min: 1, max: 5, step: 1, defaultValue: 3 },
      { key: 'friction', label: '摩擦损失/%', type: 'number', min: 0, max: 40, step: 1, defaultValue: 10 },
    ],
    buildHTML: (params, rootId) => {
      const load = num(params, 'load', 120)
      const segments = num(params, 'segments', 3)
      const friction = num(params, 'friction', 10)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🪝 滑轮组：省力但费距离</div>
    <div class="pl-note">理想情况下 F = G / n</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>重物重力</span><span class="pl-value" data-g-val></span></div><input data-g type="range" min="20" max="300" step="10" value="${n(load)}"></div>
      <div class="pl-row"><div class="pl-label"><span>承重绳段数</span><span class="pl-value" data-n-val></span></div><input data-n type="range" min="1" max="5" step="1" value="${n(segments)}"></div>
      <div class="pl-row"><div class="pl-label"><span>摩擦损失</span><span class="pl-value" data-f-val></span></div><input data-f type="range" min="0" max="40" step="1" value="${n(friction)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="160" y="64" width="360" height="18" rx="8" fill="#CBD5E1" stroke="#64748B" stroke-width="3"/>
        <g data-pulleys></g>
        <g data-ropes></g>
        <rect x="292" y="278" width="96" height="66" rx="12" fill="#F97316" stroke="#C2410C" stroke-width="4"/>
        <text x="308" y="318" font-size="20" font-weight="900" fill="#fff">重物</text>
        <text x="70" y="88" data-force font-size="28" font-weight="900" fill="#075985"></text>
        <text x="70" y="128" data-save font-size="19" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var g=root.querySelector('[data-g]'), n=root.querySelector('[data-n]'), f=root.querySelector('[data-f]');
    var gv=root.querySelector('[data-g-val]'), nv=root.querySelector('[data-n-val]'), fv=root.querySelector('[data-f-val]');
    var pul=root.querySelector('[data-pulleys]'), ropes=root.querySelector('[data-ropes]');
    var force=root.querySelector('[data-force]'), save=root.querySelector('[data-save]'), res=root.querySelector('[data-result]');
    function update(){
      var G=Number(g.value), N=Math.round(Number(n.value)), F=Number(f.value);
      var ideal=G/N, actual=ideal*(1+F/100);
      gv.textContent=G.toFixed(0)+' N'; nv.textContent=N.toFixed(0)+' 段'; fv.textContent=F.toFixed(0)+'%';
      var ps='', rs='';
      for(var i=0;i<N;i++){
        var x=220+i*58;
        ps+='<circle cx="'+x+'" cy="'+(N%2?122:118)+'" r="26" fill="#E0F2FE" stroke="#0284C7" stroke-width="4"/><circle cx="'+x+'" cy="'+(N%2?122:118)+'" r="7" fill="#0284C7"/>';
        rs+='<path d="M'+x+' 82 V278" stroke="#64748B" stroke-width="5" fill="none"/>';
      }
      pul.innerHTML=ps;
      ropes.innerHTML=rs+'<path d="M'+(220+(N-1)*58)+' 82 C520 140 554 214 554 306" stroke="#64748B" stroke-width="5" fill="none"/><path d="M540 288 L554 312 L568 288" fill="none" stroke="#64748B" stroke-width="5" stroke-linecap="round"/>';
      force.textContent='拉力≈'+actual.toFixed(1)+' N';
      save.textContent='理论省力约 '+N+' 倍，需拉更长绳';
      res.innerHTML='承重绳段数越多越省力，但重物上升同样高度时，需要拉动更长的绳子。<br>实际滑轮组有摩擦和绳重，所以实际拉力大于理想值。';
    }
    g.oninput=update;n.oninput=update;f.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-specific-heat',
    group: '🔥 热学实验',
    name: '比热容升温',
    emoji: '🌡️',
    desc: '比较水、沙子、铜块在相同加热下温度变化快慢',
    params: [
      { key: 'material', label: '材料', type: 'number', min: 0, max: 2, step: 1, defaultValue: 0, hint: '0=水, 1=沙子, 2=铜' },
      { key: 'power', label: '加热功率', type: 'number', min: 50, max: 500, step: 10, defaultValue: 220 },
      { key: 'time', label: '加热时间/s', type: 'number', min: 0, max: 180, step: 5, defaultValue: 60 },
    ],
    buildHTML: (params, rootId) => {
      const material = num(params, 'material', 0)
      const power = num(params, 'power', 220)
      const time = num(params, 'time', 60)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🌡️ 比热容：Q = cmΔt</div>
    <div class="pl-note">相同吸热下，比热容越大升温越慢</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>材料</span><span class="pl-value" data-m-val></span></div><input data-m type="range" min="0" max="2" step="1" value="${n(material)}"></div>
      <div class="pl-row"><div class="pl-label"><span>加热功率</span><span class="pl-value" data-p-val></span></div><input data-p type="range" min="50" max="500" step="10" value="${n(power)}"></div>
      <div class="pl-row"><div class="pl-label"><span>加热时间</span><span class="pl-value" data-t-val></span></div><input data-t type="range" min="0" max="180" step="5" value="${n(time)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M250 130 L430 130 L402 330 L278 330 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-sample d="M268 224 C300 246 380 246 412 224 L402 330 L278 330 Z" fill="#BAE6FD" opacity="0.82"/>
        <g data-flame></g>
        <path d="M510 318 V118" stroke="#CBD5E1" stroke-width="10" stroke-linecap="round"/>
        <circle cx="510" cy="330" r="24" fill="#FEE2E2" stroke="#EF4444" stroke-width="4"/>
        <rect data-tempbar x="505" y="260" width="10" height="58" rx="5" fill="#EF4444"/>
        <text x="546" y="178" data-temp font-size="27" font-weight="900" fill="#075985"></text>
        <text x="74" y="88" data-name font-size="27" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var m=root.querySelector('[data-m]'), p=root.querySelector('[data-p]'), t=root.querySelector('[data-t]');
    var mv=root.querySelector('[data-m-val]'), pv=root.querySelector('[data-p-val]'), tv=root.querySelector('[data-t-val]');
    var sample=root.querySelector('[data-sample]'), flame=root.querySelector('[data-flame]'), bar=root.querySelector('[data-tempbar]');
    var temp=root.querySelector('[data-temp]'), name=root.querySelector('[data-name]'), res=root.querySelector('[data-result]');
    var mats=[
      {n:'水', c:4.2, color:'#BAE6FD'},
      {n:'沙子', c:0.9, color:'#FDE68A'},
      {n:'铜', c:0.39, color:'#F97316'}
    ];
    function update(){
      var M=Math.round(Number(m.value)), P=Number(p.value), T=Number(t.value), mat=mats[M];
      var delta=P*T/(100*mat.c)/10;
      var finalT=20+delta;
      mv.textContent=mat.n; pv.textContent=P.toFixed(0)+' W'; tv.textContent=T.toFixed(0)+' s';
      sample.setAttribute('fill',mat.color);
      var h=Math.min(180,20+finalT*1.8);
      bar.setAttribute('y',String(318-h)); bar.setAttribute('height',String(h));
      temp.textContent=finalT.toFixed(1)+'℃';
      name.textContent=mat.n+'：c≈'+mat.c;
      var flames='';
      for(var i=0;i<Math.floor(P/70);i++){
        var x=282+i*28;
        flames+='<path d="M'+x+' 352 C'+(x-12)+' 330 '+x+' 320 '+(x+10)+' 300 C'+(x+24)+' 326 '+(x+14)+' 344 '+x+' 352Z" fill="#F97316" opacity="0.85"/>';
      }
      flame.innerHTML=flames;
      res.innerHTML='相同质量、相同吸热条件下，比热容越大的物质温度升高越慢。<br>水的比热容较大，所以吸热后温度变化较慢。';
    }
    m.oninput=update;p.oninput=update;t.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
