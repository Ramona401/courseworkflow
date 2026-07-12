/**
 * physicsLabTemplatesExt6.ts — 物理实验室第六批扩展模板
 *
 * 第六批新增：
 *   1. 压强与受力面积
 *   2. 摩擦力影响因素
 *   3. 热传递方式
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

export const PHYSICS_LAB_TEMPLATES_EXT6: PhysicsLabTemplate[] = [
  {
    id: 'lab-pressure-area',
    group: '⚙️ 力与机械',
    name: '压强与受力面积',
    emoji: '🧱',
    desc: '调节压力和受力面积，观察 p=F/S 的变化',
    params: [
      { key: 'force', label: '压力/N', type: 'number', min: 10, max: 500, step: 10, defaultValue: 180 },
      { key: 'area', label: '受力面积/cm²', type: 'number', min: 5, max: 120, step: 5, defaultValue: 45 },
    ],
    buildHTML: (params, rootId) => {
      const force = num(params, 'force', 180)
      const area = num(params, 'area', 45)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🧱 压强：p = F / S</div>
    <div class="pl-note">压力一定时，受力面积越小，压强越大</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>压力</span><span class="pl-value" data-f-val></span></div><input data-f type="range" min="10" max="500" step="10" value="${n(force)}"></div>
      <div class="pl-row"><div class="pl-label"><span>受力面积</span><span class="pl-value" data-a-val></span></div><input data-a type="range" min="5" max="120" step="5" value="${n(area)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="130" y="308" width="420" height="34" rx="10" fill="#E5E7EB" stroke="#CBD5E1" stroke-width="4"/>
        <rect data-block x="290" y="190" width="100" height="118" rx="12" fill="#F97316" stroke="#C2410C" stroke-width="4"/>
        <path data-arrow d="M340 84 V176" stroke="#DC2626" stroke-width="8" stroke-linecap="round"/>
        <path d="M320 156 L340 184 L360 156" fill="none" stroke="#DC2626" stroke-width="8" stroke-linecap="round"/>
        <ellipse data-mark cx="340" cy="312" rx="52" ry="12" fill="#EF4444" opacity="0.28"/>
        <text x="82" y="78" data-p-text font-size="30" font-weight="900" fill="#075985"></text>
        <text x="82" y="120" data-state font-size="20" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var f=root.querySelector('[data-f]'), a=root.querySelector('[data-a]');
    var fv=root.querySelector('[data-f-val]'), av=root.querySelector('[data-a-val]');
    var block=root.querySelector('[data-block]'), arrow=root.querySelector('[data-arrow]'), mark=root.querySelector('[data-mark]');
    var pt=root.querySelector('[data-p-text]'), state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    function update(){
      var F=Number(f.value), A=Number(a.value), P=F/A;
      var w=36+A*1.15, x=340-w/2;
      fv.textContent=F.toFixed(0)+' N'; av.textContent=A.toFixed(0)+' cm²';
      block.setAttribute('x',String(x)); block.setAttribute('width',String(w));
      mark.setAttribute('rx',String(w*0.48));
      mark.setAttribute('opacity',String(Math.min(0.75,0.18+P/18)));
      arrow.setAttribute('stroke-width',String(4+F/70));
      pt.textContent='p≈'+P.toFixed(2)+' N/cm²';
      state.textContent=P>6?'压强较大':P>2?'压强中等':'压强较小';
      res.innerHTML='压强等于压力除以受力面积。<br>压力相同时，刀刃、钉尖面积小，所以压强大；书包肩带加宽可减小压强。';
    }
    f.oninput=update;a.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-friction-force',
    group: '⚙️ 力与机械',
    name: '摩擦力影响因素',
    emoji: '🧲',
    desc: '调节压力和接触面粗糙程度，观察滑动摩擦力变化',
    params: [
      { key: 'normal', label: '压力/N', type: 'number', min: 10, max: 300, step: 10, defaultValue: 120 },
      { key: 'roughness', label: '粗糙程度', type: 'number', min: 0.05, max: 1, step: 0.05, defaultValue: 0.35 },
    ],
    buildHTML: (params, rootId) => {
      const normal = num(params, 'normal', 120)
      const roughness = num(params, 'roughness', 0.35)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🧲 摩擦力：f = μN</div>
    <div class="pl-note">压力越大、接触面越粗糙，摩擦力越大</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>压力</span><span class="pl-value" data-n-val></span></div><input data-n type="range" min="10" max="300" step="10" value="${n(normal)}"></div>
      <div class="pl-row"><div class="pl-label"><span>粗糙程度</span><span class="pl-value" data-r-val></span></div><input data-r type="range" min="0.05" max="1" step="0.05" value="${n(roughness)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="86" y="296" width="508" height="34" rx="10" fill="#E5E7EB" stroke="#CBD5E1" stroke-width="4"/>
        <g data-ground></g>
        <rect x="270" y="214" width="140" height="82" rx="16" fill="#60A5FA" stroke="#2563EB" stroke-width="4"/>
        <path d="M420 256 H552" stroke="#059669" stroke-width="7" stroke-linecap="round"/>
        <path d="M532 236 L560 256 L532 276" fill="none" stroke="#059669" stroke-width="7" stroke-linecap="round"/>
        <path data-fric d="M260 256 H150" stroke="#DC2626" stroke-width="7" stroke-linecap="round"/>
        <path d="M170 236 L142 256 L170 276" fill="none" stroke="#DC2626" stroke-width="7" stroke-linecap="round"/>
        <text x="72" y="86" data-f-text font-size="30" font-weight="900" fill="#075985"></text>
        <text x="72" y="126" data-note font-size="20" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var n=root.querySelector('[data-n]'), r=root.querySelector('[data-r]');
    var nv=root.querySelector('[data-n-val]'), rv=root.querySelector('[data-r-val]');
    var ground=root.querySelector('[data-ground]'), fric=root.querySelector('[data-fric]');
    var ft=root.querySelector('[data-f-text]'), note=root.querySelector('[data-note]'), res=root.querySelector('[data-result]');
    function update(){
      var N=Number(n.value), R=Number(r.value), F=N*R;
      nv.textContent=N.toFixed(0)+' N'; rv.textContent=R.toFixed(2);
      var marks='';
      for(var i=0;i<Math.floor(R*28);i++){
        marks+='<path d="M'+(100+i*18)+' 296 l8 -12 l8 12" stroke="#94A3B8" stroke-width="3" fill="none"/>';
      }
      ground.innerHTML=marks;
      fric.setAttribute('stroke-width',String(4+F/35));
      ft.textContent='f≈'+F.toFixed(1)+' N';
      note.textContent=F>120?'摩擦较大':F>45?'摩擦中等':'摩擦较小';
      res.innerHTML='滑动摩擦力大小与压力和接触面粗糙程度有关。<br>压力越大、接触面越粗糙，摩擦力通常越大。';
    }
    n.oninput=update;r.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-heat-transfer',
    group: '🔥 热学实验',
    name: '热传递方式',
    emoji: '♨️',
    desc: '比较传导、对流、辐射三种热传递方式',
    params: [
      { key: 'mode', label: '方式', type: 'number', min: 0, max: 2, step: 1, defaultValue: 0, hint: '0=传导，1=对流，2=辐射' },
      { key: 'power', label: '热源强度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 60 },
      { key: 'time', label: '加热时间', type: 'number', min: 0, max: 100, step: 1, defaultValue: 50 },
    ],
    buildHTML: (params, rootId) => {
      const mode = num(params, 'mode', 0)
      const power = num(params, 'power', 60)
      const time = num(params, 'time', 50)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">♨️ 热传递：传导 / 对流 / 辐射</div>
    <div class="pl-note">热总是自发地从高温物体传向低温物体</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>方式</span><span class="pl-value" data-m-val></span></div><input data-m type="range" min="0" max="2" step="1" value="${n(mode)}"></div>
      <div class="pl-row"><div class="pl-label"><span>热源强度</span><span class="pl-value" data-p-val></span></div><input data-p type="range" min="0" max="100" step="1" value="${n(power)}"></div>
      <div class="pl-row"><div class="pl-label"><span>加热时间</span><span class="pl-value" data-t-val></span></div><input data-t type="range" min="0" max="100" step="1" value="${n(time)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <g data-scene></g>
        <g data-heat></g>
        <text x="74" y="82" data-title font-size="30" font-weight="900" fill="#075985"></text>
        <text x="74" y="122" data-temp font-size="20" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var m=root.querySelector('[data-m]'), p=root.querySelector('[data-p]'), t=root.querySelector('[data-t]');
    var mv=root.querySelector('[data-m-val]'), pv=root.querySelector('[data-p-val]'), tv=root.querySelector('[data-t-val]');
    var scene=root.querySelector('[data-scene]'), heat=root.querySelector('[data-heat]');
    var title=root.querySelector('[data-title]'), temp=root.querySelector('[data-temp]'), res=root.querySelector('[data-result]');
    var names=['传导','对流','辐射'];
    function update(){
      var M=Math.round(Number(m.value)), P=Number(p.value), T=Number(t.value), Q=P*T/100;
      mv.textContent=names[M]; pv.textContent=P.toFixed(0)+'%'; tv.textContent=T.toFixed(0)+'%';
      title.textContent=names[M];
      temp.textContent='热量传递强度 ≈ '+Q.toFixed(0);
      if(M===0){
        scene.innerHTML='<rect x="180" y="238" width="330" height="30" rx="12" fill="#CBD5E1" stroke="#64748B" stroke-width="4"/><circle cx="190" cy="253" r="28" fill="#EF4444"/><text x="248" y="230" font-size="18" font-weight="900" fill="#475569">金属棒传热</text>';
        var h='';
        for(var i=0;i<8;i++){h+='<circle cx="'+(224+i*36)+'" cy="253" r="'+(8+Math.max(0,Q-i*10)/8)+'" fill="#F97316" opacity="'+Math.max(0.15,0.75-i*0.07)+'"/>'}
        heat.innerHTML=h;
        res.innerHTML='传导主要发生在固体中，热量沿物体从高温部分传向低温部分。金属通常导热较快。';
      }else if(M===1){
        scene.innerHTML='<path d="M230 116 L450 116 L418 340 L262 340 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/><path d="M250 218 C292 248 388 248 430 218 L418 340 L262 340 Z" fill="#BAE6FD" opacity="0.78"/>';
        var c='';
        for(var j=0;j<Math.floor(4+Q/12);j++){c+='<path d="M'+(292+(j%4)*34)+' '+(300-(j*22)%124)+' C'+(260+(j%4)*34)+' '+(276-(j*22)%124)+' '+(304+(j%4)*34)+' '+(250-(j*22)%124)+' '+(282+(j%4)*34)+' '+(226-(j*22)%124)+'" fill="none" stroke="#F97316" stroke-width="4" opacity="0.72"/>'}
        heat.innerHTML=c;
        res.innerHTML='对流主要发生在液体和气体中。受热部分密度变小上升，冷的部分下降，形成循环流动。';
      }else{
        scene.innerHTML='<circle cx="228" cy="238" r="42" fill="#EF4444"/><rect x="460" y="182" width="76" height="112" rx="16" fill="#E5E7EB" stroke="#64748B" stroke-width="4"/><text x="458" y="322" font-size="18" font-weight="900" fill="#475569">物体受热</text>';
        var r='';
        for(var k=0;k<Math.floor(5+Q/12);k++){r+='<path d="M270 238 C'+(320+k*12)+' '+(174+k%3*24)+' '+(384+k*10)+' '+(184+k%4*18)+' 460 '+(210+k%5*11)+'" fill="none" stroke="#F59E0B" stroke-width="4" opacity="0.65"/>'}
        heat.innerHTML=r;
        res.innerHTML='辐射不需要介质，太阳把热传到地球主要依靠热辐射。温度越高，辐射越强。';
      }
    }
    m.oninput=update;p.oninput=update;t.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
