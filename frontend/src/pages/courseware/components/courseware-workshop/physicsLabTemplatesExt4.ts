/**
 * physicsLabTemplatesExt4.ts — 物理实验室第四批扩展模板
 *
 * 第四批新增：
 *   1. 杠杆平衡
 *   2. 密度测量
 *   3. 小孔成像
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

export const PHYSICS_LAB_TEMPLATES_EXT4: PhysicsLabTemplate[] = [
  {
    id: 'lab-lever-balance',
    group: '⚙️ 力与机械',
    name: '杠杆平衡',
    emoji: '⚖️',
    desc: '调节力和力臂，观察杠杆平衡条件 F1L1=F2L2',
    params: [
      { key: 'f1', label: '左侧力/N', type: 'number', min: 1, max: 20, step: 1, defaultValue: 8 },
      { key: 'l1', label: '左力臂/cm', type: 'number', min: 1, max: 10, step: 1, defaultValue: 6 },
      { key: 'f2', label: '右侧力/N', type: 'number', min: 1, max: 20, step: 1, defaultValue: 6 },
      { key: 'l2', label: '右力臂/cm', type: 'number', min: 1, max: 10, step: 1, defaultValue: 8 },
    ],
    buildHTML: (params, rootId) => {
      const f1 = num(params, 'f1', 8)
      const l1 = num(params, 'l1', 6)
      const f2 = num(params, 'f2', 6)
      const l2 = num(params, 'l2', 8)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">⚖️ 杠杆平衡：F₁L₁ = F₂L₂</div>
    <div class="pl-note">比较左右两侧力矩大小</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>左侧力</span><span class="pl-value" data-f1-val></span></div><input data-f1 type="range" min="1" max="20" step="1" value="${n(f1)}"></div>
      <div class="pl-row"><div class="pl-label"><span>左力臂</span><span class="pl-value" data-l1-val></span></div><input data-l1 type="range" min="1" max="10" step="1" value="${n(l1)}"></div>
      <div class="pl-row"><div class="pl-label"><span>右侧力</span><span class="pl-value" data-f2-val></span></div><input data-f2 type="range" min="1" max="20" step="1" value="${n(f2)}"></div>
      <div class="pl-row"><div class="pl-label"><span>右力臂</span><span class="pl-value" data-l2-val></span></div><input data-l2 type="range" min="1" max="10" step="1" value="${n(l2)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <polygon points="340,224 306,330 374,330" fill="#CBD5E1" stroke="#64748B" stroke-width="4"/>
        <line data-beam x1="130" y1="210" x2="550" y2="210" stroke="#92400E" stroke-width="12" stroke-linecap="round"/>
        <circle cx="340" cy="210" r="14" fill="#475569"/>
        <g data-left></g>
        <g data-right></g>
        <text x="96" y="76" data-state font-size="28" font-weight="900" fill="#075985"></text>
        <text x="96" y="114" data-moment font-size="19" font-weight="900" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var f1=root.querySelector('[data-f1]'), l1=root.querySelector('[data-l1]'), f2=root.querySelector('[data-f2]'), l2=root.querySelector('[data-l2]');
    var f1v=root.querySelector('[data-f1-val]'), l1v=root.querySelector('[data-l1-val]'), f2v=root.querySelector('[data-f2-val]'), l2v=root.querySelector('[data-l2-val]');
    var beam=root.querySelector('[data-beam]'), left=root.querySelector('[data-left]'), right=root.querySelector('[data-right]');
    var state=root.querySelector('[data-state]'), moment=root.querySelector('[data-moment]'), res=root.querySelector('[data-result]');
    function update(){
      var F1=Number(f1.value), L1=Number(l1.value), F2=Number(f2.value), L2=Number(l2.value);
      var M1=F1*L1, M2=F2*L2, diff=M2-M1, angle=Math.max(-12,Math.min(12,diff/5));
      f1v.textContent=F1.toFixed(0)+' N'; l1v.textContent=L1.toFixed(0)+' cm'; f2v.textContent=F2.toFixed(0)+' N'; l2v.textContent=L2.toFixed(0)+' cm';
      beam.setAttribute('transform','rotate('+angle+' 340 210)');
      var lx=340-L1*28, rx=340+L2*28;
      left.innerHTML='<line x1="'+lx+'" y1="210" x2="'+lx+'" y2="'+(210+F1*5)+'" stroke="#2563EB" stroke-width="6" stroke-linecap="round"/><path d="M'+(lx-14)+' '+(198+F1*5)+' L'+lx+' '+(218+F1*5)+' L'+(lx+14)+' '+(198+F1*5)+'" fill="#2563EB"/><text x="'+(lx-34)+'" y="'+(238+F1*5)+'" font-size="15" font-weight="900" fill="#2563EB">F₁</text>';
      right.innerHTML='<line x1="'+rx+'" y1="210" x2="'+rx+'" y2="'+(210+F2*5)+'" stroke="#EF4444" stroke-width="6" stroke-linecap="round"/><path d="M'+(rx-14)+' '+(198+F2*5)+' L'+rx+' '+(218+F2*5)+' L'+(rx+14)+' '+(198+F2*5)+'" fill="#EF4444"/><text x="'+(rx-14)+'" y="'+(238+F2*5)+'" font-size="15" font-weight="900" fill="#EF4444">F₂</text>';
      state.textContent=Math.abs(diff)<3?'接近平衡':diff>0?'右端下沉':'左端下沉';
      moment.textContent='左力矩 '+M1.toFixed(0)+'，右力矩 '+M2.toFixed(0);
      res.innerHTML='杠杆平衡条件：动力 × 动力臂 = 阻力 × 阻力臂。<br>比较两侧力矩大小，可以判断哪一端下沉。';
    }
    f1.oninput=update;l1.oninput=update;f2.oninput=update;l2.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-density-measurement',
    group: '⚙️ 力与机械',
    name: '密度测量',
    emoji: '🧪',
    desc: '读取天平质量和量筒体积，计算密度 ρ=m/V',
    params: [
      { key: 'mass', label: '质量/g', type: 'number', min: 10, max: 300, step: 5, defaultValue: 120 },
      { key: 'volume', label: '体积/mL', type: 'number', min: 10, max: 200, step: 5, defaultValue: 80 },
    ],
    buildHTML: (params, rootId) => {
      const mass = num(params, 'mass', 120)
      const volume = num(params, 'volume', 80)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🧪 密度测量：ρ = m / V</div>
    <div class="pl-note">质量用天平，体积可用量筒排水法</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>质量</span><span class="pl-value" data-m-val></span></div><input data-m type="range" min="10" max="300" step="5" value="${n(mass)}"></div>
      <div class="pl-row"><div class="pl-label"><span>体积</span><span class="pl-value" data-v-val></span></div><input data-v type="range" min="10" max="200" step="5" value="${n(volume)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="92" y="266" width="210" height="34" rx="12" fill="#CBD5E1" stroke="#64748B" stroke-width="4"/>
        <rect x="174" y="168" width="46" height="98" rx="8" fill="#94A3B8"/>
        <rect x="126" y="132" width="142" height="40" rx="12" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <text x="144" y="158" data-mass-text font-size="20" font-weight="900" fill="#075985"></text>
        <path d="M420 82 L540 82 L514 334 L446 334 Z" fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <path data-water d="M436 224 C458 240 502 240 524 224 L514 334 L446 334 Z" fill="#BAE6FD" opacity="0.82"/>
        <rect data-object x="468" y="236" width="28" height="28" rx="6" fill="#F97316" stroke="#C2410C" stroke-width="3"/>
        <text x="428" y="62" font-size="18" font-weight="900" fill="#475569">量筒排水</text>
        <text x="244" y="360" data-density font-size="30" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var m=root.querySelector('[data-m]'), v=root.querySelector('[data-v]');
    var mv=root.querySelector('[data-m-val]'), vv=root.querySelector('[data-v-val]');
    var mt=root.querySelector('[data-mass-text]'), water=root.querySelector('[data-water]'), obj=root.querySelector('[data-object]');
    var den=root.querySelector('[data-density]'), res=root.querySelector('[data-result]');
    function update(){
      var M=Number(m.value), V=Number(v.value), D=M/V;
      mv.textContent=M.toFixed(0)+' g'; vv.textContent=V.toFixed(0)+' mL';
      mt.textContent=M.toFixed(0)+' g';
      var h=Math.min(160,42+V*0.62), y=334-h;
      water.setAttribute('d','M436 '+y+' C458 '+(y+16)+' 502 '+(y+16)+' 524 '+y+' L514 334 L446 334 Z');
      obj.setAttribute('y',String(Math.max(126,322-h*0.55)));
      den.textContent='ρ = '+D.toFixed(2)+' g/mL';
      res.innerHTML='密度计算公式：ρ = m / V。<br>不规则固体体积可用排水法测量，读数时视线应与液面凹面最低处相平。';
    }
    m.oninput=update;v.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-pinhole-camera',
    group: '🔍 光学实验',
    name: '小孔成像',
    emoji: '📷',
    desc: '观察光沿直线传播形成倒立实像，小孔大小影响清晰度',
    params: [
      { key: 'distance', label: '物孔距离', type: 'number', min: 80, max: 300, step: 10, defaultValue: 180 },
      { key: 'hole', label: '小孔大小', type: 'number', min: 2, max: 30, step: 1, defaultValue: 8 },
    ],
    buildHTML: (params, rootId) => {
      const distance = num(params, 'distance', 180)
      const hole = num(params, 'hole', 8)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">📷 小孔成像：光沿直线传播</div>
    <div class="pl-note">像是倒立的，小孔过大则模糊</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>物屏距离</span><span class="pl-value" data-d-val></span></div><input data-d type="range" min="80" max="300" step="10" value="${n(distance)}"></div>
      <div class="pl-row"><div class="pl-label"><span>小孔大小</span><span class="pl-value" data-h-val></span></div><input data-h type="range" min="2" max="30" step="1" value="${n(hole)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <g data-object></g>
        <rect x="330" y="72" width="22" height="270" rx="6" fill="#334155"/>
        <circle cx="341" cy="207" data-hole r="8" fill="#FFFFFF"/>
        <rect x="534" y="92" width="28" height="230" rx="8" fill="#E5E7EB" stroke="#64748B" stroke-width="4"/>
        <g data-rays></g>
        <g data-image></g>
        <text x="84" y="354" font-size="18" font-weight="900" fill="#475569">物体</text>
        <text x="304" y="360" font-size="18" font-weight="900" fill="#475569">小孔</text>
        <text x="512" y="354" font-size="18" font-weight="900" fill="#475569">光屏</text>
        <text x="388" y="62" data-note font-size="20" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var d=root.querySelector('[data-d]'), h=root.querySelector('[data-h]');
    var dv=root.querySelector('[data-d-val]'), hv=root.querySelector('[data-h-val]');
    var obj=root.querySelector('[data-object]'), img=root.querySelector('[data-image]'), rays=root.querySelector('[data-rays]'), hole=root.querySelector('[data-hole]');
    var note=root.querySelector('[data-note]'), res=root.querySelector('[data-result]');
    function candle(x,y,scale,flip,op){
      var fy=flip?-1:1;
      return '<rect x="'+(x-10*scale)+'" y="'+(y-70*scale*fy)+'" width="'+(20*scale)+'" height="'+(70*scale*fy)+'" rx="'+(6*scale)+'" fill="#FEF3C7" stroke="#D97706" opacity="'+op+'"/><path d="M'+x+' '+(y-78*scale*fy)+' C'+(x-20*scale)+' '+(y-116*scale*fy)+' '+x+' '+(y-128*scale*fy)+' '+(x+10*scale)+' '+(y-152*scale*fy)+' C'+(x+28*scale)+' '+(y-112*scale*fy)+' '+(x+18*scale)+' '+(y-92*scale*fy)+' '+x+' '+(y-78*scale*fy)+'Z" fill="#F97316" opacity="'+op+'"/>';
    }
    function update(){
      var D=Number(d.value), H=Number(h.value);
      var pinX=341, axis=207, objH=124;
      var ox=Math.max(56, pinX-D*0.9);
      var screenX=548;
      var mag=(screenX-pinX)/Math.max(40,(pinX-ox));
      var imgScale=Math.max(0.35, Math.min(1.05, 0.42+mag*0.18));
      var blur=Math.min(1,H/24);
      dv.textContent=D.toFixed(0)+' px'; hv.textContent=H.toFixed(0)+' px';
      hole.setAttribute('r',String(H/2));
      obj.innerHTML=candle(ox,286,1,0,1);
      rays.innerHTML='<line x1="'+ox+'" y1="'+(286-124)+'" x2="'+pinX+'" y2="'+axis+'" stroke="#F59E0B" stroke-width="3"/><line x1="'+pinX+'" y1="'+axis+'" x2="'+screenX+'" y2="'+(axis+124*mag)+'" stroke="#F59E0B" stroke-width="3"/><line x1="'+ox+'" y1="286" x2="'+pinX+'" y2="'+axis+'" stroke="#2563EB" stroke-width="3"/><line x1="'+pinX+'" y1="'+axis+'" x2="'+screenX+'" y2="'+(axis-78*mag)+'" stroke="#2563EB" stroke-width="3"/>';
      img.innerHTML=candle(screenX,axis-78*mag,imgScale,1,0.9-blur*0.35);
      note.textContent=H>18?'小孔偏大，像较模糊':'倒立实像较清晰';
      res.innerHTML='小孔成像说明光沿直线传播。物体上方的光线通过小孔到达光屏下方，所以像是倒立的。<br>小孔太大时，多束光线重叠，像会变模糊。';
    }
    d.oninput=update;h.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
