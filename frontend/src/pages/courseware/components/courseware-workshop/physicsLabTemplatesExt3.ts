/**
 * physicsLabTemplatesExt3.ts — 物理实验室第三批扩展模板
 *
 * 第三批新增：
 *   1. 电功率与灯泡亮度
 *   2. 三棱镜色散
 *   3. 浮力与排开液体
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

export const PHYSICS_LAB_TEMPLATES_EXT3: PhysicsLabTemplate[] = [
  {
    id: 'lab-electric-power-brightness',
    group: '🔌 电学实验',
    name: '电功率与灯泡亮度',
    emoji: '💡',
    desc: '调节电压和灯泡电阻，观察功率 P=U²/R 与亮度变化',
    params: [
      { key: 'u', label: '电压 U/V', type: 'number', min: 1, max: 12, step: 0.5, defaultValue: 6 },
      { key: 'r', label: '灯泡电阻 R/Ω', type: 'number', min: 2, max: 40, step: 1, defaultValue: 12 },
    ],
    buildHTML: (params, rootId) => {
      const u = num(params, 'u', 6)
      const r = num(params, 'r', 12)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">💡 电功率：P = UI = U²/R</div>
    <div class="pl-note">功率越大，灯泡通常越亮</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>电压 U</span><span class="pl-value" data-u-val></span></div><input data-u type="range" min="1" max="12" step="0.5" value="${n(u)}"></div>
      <div class="pl-row"><div class="pl-label"><span>电阻 R</span><span class="pl-value" data-r-val></span></div><input data-r type="range" min="2" max="40" step="1" value="${n(r)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M116 210 H246 M434 210 H560 M560 210 V318 H116 V210 M116 210 V98 H560 V210" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
        <line x1="154" y1="80" x2="154" y2="116" stroke="#0F172A" stroke-width="5"/>
        <line x1="178" y1="90" x2="178" y2="108" stroke="#0F172A" stroke-width="5"/>
        <circle cx="340" cy="210" r="72" data-glow fill="#FEF3C7" opacity="0.35"/>
        <circle cx="340" cy="210" r="44" data-bulb fill="#FEF3C7" stroke="#F59E0B" stroke-width="5"/>
        <path d="M318 214 q22 -34 44 0" fill="none" stroke="#92400E" stroke-width="4"/>
        <text x="470" y="122" data-p-text font-size="27" font-weight="900" fill="#075985"></text>
        <text x="470" y="160" data-i-text font-size="20" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var u=root.querySelector('[data-u]'), r=root.querySelector('[data-r]');
    var uv=root.querySelector('[data-u-val]'), rv=root.querySelector('[data-r-val]');
    var bulb=root.querySelector('[data-bulb]'), glow=root.querySelector('[data-glow]');
    var pt=root.querySelector('[data-p-text]'), it=root.querySelector('[data-i-text]'), res=root.querySelector('[data-result]');
    function update(){
      var U=Number(u.value), R=Number(r.value), I=U/R, P=U*I, b=Math.min(1,P/6);
      uv.textContent=U.toFixed(1)+' V'; rv.textContent=R.toFixed(0)+' Ω';
      bulb.setAttribute('fill',b>0.7?'#FDE68A':b>0.3?'#FEF3C7':'#F8FAFC');
      glow.setAttribute('opacity',String(0.15+b*0.65));
      glow.setAttribute('r',String(52+b*54));
      pt.textContent='P = '+P.toFixed(2)+' W';
      it.textContent='I = '+I.toFixed(2)+' A';
      res.innerHTML='灯泡亮度通常与实际电功率有关。电阻一定时，电压越大，功率按 U²/R 增大。<br>当前 P = '+U.toFixed(1)+'² / '+R.toFixed(0)+' = '+P.toFixed(2)+' W。';
    }
    u.oninput=update;r.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-prism-dispersion',
    group: '🔍 光学实验',
    name: '三棱镜色散',
    emoji: '🌈',
    desc: '白光通过三棱镜后发生色散，不同颜色偏折程度不同',
    params: [
      { key: 'angle', label: '入射角/°', type: 'number', min: 5, max: 50, step: 1, defaultValue: 24 },
      { key: 'spread', label: '色散程度', type: 'number', min: 0, max: 100, step: 1, defaultValue: 55 },
    ],
    buildHTML: (params, rootId) => {
      const angle = num(params, 'angle', 24)
      const spread = num(params, 'spread', 55)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🌈 三棱镜色散：白光分解为多种色光</div>
    <div class="pl-note">紫光偏折较大，红光偏折较小</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>入射角</span><span class="pl-value" data-a-val></span></div><input data-a type="range" min="5" max="50" step="1" value="${n(angle)}"></div>
      <div class="pl-row"><div class="pl-label"><span>色散程度</span><span class="pl-value" data-s-val></span></div><input data-s type="range" min="0" max="100" step="1" value="${n(spread)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <polygon points="310,92 210,300 420,300" fill="#E0F2FE" stroke="#0284C7" stroke-width="5" opacity="0.75"/>
        <line x1="56" y1="206" x2="238" y2="206" stroke="#F8FAFC" stroke-width="15" stroke-linecap="round"/>
        <line x1="56" y1="206" x2="238" y2="206" stroke="#CBD5E1" stroke-width="4" stroke-linecap="round"/>
        <g data-rays></g>
        <text x="86" y="176" font-size="18" font-weight="900" fill="#475569">白光</text>
        <text x="246" y="328" font-size="18" font-weight="900" fill="#0284C7">三棱镜</text>
        <text x="458" y="346" data-label font-size="21" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var ae=root.querySelector('[data-a]'), se=root.querySelector('[data-s]');
    var av=root.querySelector('[data-a-val]'), sv=root.querySelector('[data-s-val]');
    var rays=root.querySelector('[data-rays]'), label=root.querySelector('[data-label]'), res=root.querySelector('[data-result]');
    function update(){
      var A=Number(ae.value), S=Number(se.value);
      av.textContent=A.toFixed(0)+'°'; sv.textContent=S.toFixed(0)+'%';
      var colors=['#EF4444','#F97316','#FACC15','#22C55E','#38BDF8','#2563EB','#7C3AED'];
      var html='';
      for(var i=0;i<colors.length;i++){
        var dy=(i-3)*S*0.34 + A*0.55;
        html+='<path d="M372 210 C430 '+(194+dy*0.25)+' 488 '+(176+dy)+' 620 '+(158+dy)+'" fill="none" stroke="'+colors[i]+'" stroke-width="5" stroke-linecap="round" opacity="0.88"/>';
      }
      rays.innerHTML=html;
      label.textContent='红橙黄绿蓝靛紫';
      res.innerHTML='白光通过三棱镜后，不同颜色光的折射程度不同，于是分散成连续色带。<br>通常红光偏折较小，紫光偏折较大。';
    }
    ae.oninput=update;se.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-buoyancy-displacement',
    group: '🌊 波动与声学',
    name: '浮力与排开液体',
    emoji: '🛟',
    desc: '调节物体体积、密度和液体密度，观察浮沉与排开液体体积',
    params: [
      { key: 'objDensity', label: '物体密度', type: 'number', min: 0.2, max: 2.5, step: 0.1, defaultValue: 0.8 },
      { key: 'liquidDensity', label: '液体密度', type: 'number', min: 0.6, max: 1.4, step: 0.1, defaultValue: 1 },
      { key: 'volume', label: '物体体积', type: 'number', min: 20, max: 100, step: 5, defaultValue: 60 },
    ],
    buildHTML: (params, rootId) => {
      const objDensity = num(params, 'objDensity', 0.8)
      const liquidDensity = num(params, 'liquidDensity', 1)
      const volume = num(params, 'volume', 60)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🛟 浮力：F浮 = ρ液 g V排</div>
    <div class="pl-note">浮沉由物体密度与液体密度比较决定</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>物体密度</span><span class="pl-value" data-od-val></span></div><input data-od type="range" min="0.2" max="2.5" step="0.1" value="${n(objDensity)}"></div>
      <div class="pl-row"><div class="pl-label"><span>液体密度</span><span class="pl-value" data-ld-val></span></div><input data-ld type="range" min="0.6" max="1.4" step="0.1" value="${n(liquidDensity)}"></div>
      <div class="pl-row"><div class="pl-label"><span>物体体积</span><span class="pl-value" data-v-val></span></div><input data-v type="range" min="20" max="100" step="5" value="${n(volume)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M200 86 L500 86 L456 348 L244 348 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-water d="M220 184 C270 214 430 214 480 184 L456 348 L244 348 Z" fill="#BAE6FD" opacity="0.82"/>
        <rect data-object x="310" y="196" width="60" height="60" rx="12" fill="#F97316" stroke="#C2410C" stroke-width="4"/>
        <path data-arrow-up d="M532 306 V222" stroke="#0284C7" stroke-width="6" stroke-linecap="round"/>
        <path d="M516 238 L532 216 L548 238" fill="none" stroke="#0284C7" stroke-width="6" stroke-linecap="round"/>
        <path data-arrow-down d="M576 196 V280" stroke="#DC2626" stroke-width="6" stroke-linecap="round"/>
        <path d="M560 264 L576 286 L592 264" fill="none" stroke="#DC2626" stroke-width="6" stroke-linecap="round"/>
        <text x="510" y="334" font-size="16" font-weight="900" fill="#0284C7">浮力</text>
        <text x="552" y="176" font-size="16" font-weight="900" fill="#DC2626">重力</text>
        <text x="78" y="78" data-state font-size="28" font-weight="900" fill="#075985"></text>
        <text x="78" y="116" data-force font-size="18" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var od=root.querySelector('[data-od]'), ld=root.querySelector('[data-ld]'), vol=root.querySelector('[data-v]');
    var odv=root.querySelector('[data-od-val]'), ldv=root.querySelector('[data-ld-val]'), vv=root.querySelector('[data-v-val]');
    var obj=root.querySelector('[data-object]'), state=root.querySelector('[data-state]'), force=root.querySelector('[data-force]'), res=root.querySelector('[data-result]');
    function update(){
      var OD=Number(od.value), LD=Number(ld.value), V=Number(vol.value);
      var ratio=OD/LD, side=36+V*0.62, x=340-side/2;
      var y= ratio<1 ? 184 - side*(1-ratio)*0.6 : 286 - side*0.5 + Math.min(62,(ratio-1)*42);
      var displaced=ratio<1?V*ratio:V;
      var f=LD*displaced, g=OD*V;
      odv.textContent=OD.toFixed(1); ldv.textContent=LD.toFixed(1); vv.textContent=V.toFixed(0);
      obj.setAttribute('x',String(x)); obj.setAttribute('y',String(y)); obj.setAttribute('width',String(side)); obj.setAttribute('height',String(side));
      state.textContent=ratio<0.96?'上浮/漂浮':ratio>1.04?'下沉':'悬浮';
      force.textContent='F浮≈'+f.toFixed(1)+'，G≈'+g.toFixed(1);
      res.innerHTML='浮力大小等于液体密度、重力加速度和排开液体体积的乘积。<br>当物体密度小于液体密度时易上浮；大于液体密度时易下沉；接近时可悬浮。';
    }
    od.oninput=update;ld.oninput=update;vol.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
