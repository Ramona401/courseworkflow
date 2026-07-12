/**
 * physicsLabTemplatesExt.ts — 物理实验室扩展模板库
 *
 * 定位：
 *   补充 PhysicsLabModal 的非力学实验资源。
 *
 * 接入方式：
 *   下一批在 PhysicsLabModal.tsx 中聚合：
 *   [...PHYSICS_LAB_TEMPLATES, ...PHYSICS_LAB_TEMPLATES_EXT]
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
    + '#' + rootId + ' canvas{width:100%;height:100%;display:block;}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const PHYSICS_LAB_TEMPLATES_EXT: PhysicsLabTemplate[] = [
  {
    id: 'lab-joule-law',
    group: '🔌 电学实验',
    name: '焦耳定律',
    emoji: '🔥',
    desc: '调节电流、电阻和通电时间，观察电热 Q=I²Rt',
    params: [
      { key: 'i', label: '电流 I/A', type: 'number', min: 0.2, max: 5, step: 0.1, defaultValue: 2 },
      { key: 'r', label: '电阻 R/Ω', type: 'number', min: 1, max: 30, step: 1, defaultValue: 10 },
      { key: 't', label: '时间 t/s', type: 'number', min: 1, max: 60, step: 1, defaultValue: 20 },
    ],
    buildHTML: (params, rootId) => {
      const i = num(params, 'i', 2)
      const r = num(params, 'r', 10)
      const t = num(params, 't', 20)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🔥 焦耳定律：Q = I²Rt</div>
    <div class="pl-note">电流影响是平方关系</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>电流 I</span><span class="pl-value" data-i-val></span></div><input data-i type="range" min="0.2" max="5" step="0.1" value="${n(i)}"></div>
      <div class="pl-row"><div class="pl-label"><span>电阻 R</span><span class="pl-value" data-r-val></span></div><input data-r type="range" min="1" max="30" step="1" value="${n(r)}"></div>
      <div class="pl-row"><div class="pl-label"><span>通电时间 t</span><span class="pl-value" data-t-val></span></div><input data-t type="range" min="1" max="60" step="1" value="${n(t)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M120 190 H220 M460 190 H560 M560 190 V300 H120 V190 M120 190 V92 H560 V190" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
        <line x1="160" y1="75" x2="160" y2="110" stroke="#0F172A" stroke-width="5"/>
        <line x1="185" y1="84" x2="185" y2="102" stroke="#0F172A" stroke-width="5"/>
        <rect x="220" y="148" width="240" height="84" rx="22" fill="#FEE2E2" stroke="#EF4444" stroke-width="4"/>
        <path d="M246 190 q22 -34 44 0 t44 0 t44 0 t44 0" fill="none" stroke="#DC2626" stroke-width="5"/>
        <g data-heat></g>
        <text x="258" y="128" font-size="20" font-weight="900" fill="#DC2626">电热丝</text>
        <text x="212" y="354" data-q-text font-size="28" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var ie=root.querySelector('[data-i]'), re=root.querySelector('[data-r]'), te=root.querySelector('[data-t]');
    var iv=root.querySelector('[data-i-val]'), rv=root.querySelector('[data-r-val]'), tv=root.querySelector('[data-t-val]');
    var heat=root.querySelector('[data-heat]'), qt=root.querySelector('[data-q-text]'), res=root.querySelector('[data-result]');
    function update(){
      var I=Number(ie.value), R=Number(re.value), T=Number(te.value), Q=I*I*R*T;
      iv.textContent=I.toFixed(1)+' A'; rv.textContent=R.toFixed(0)+' Ω'; tv.textContent=T.toFixed(0)+' s';
      var count=Math.min(18,Math.floor(Q/120));
      var h='';
      for(var k=0;k<count;k++){
        var x=245+(k%6)*34, y=142-Math.floor(k/6)*28;
        h+='<path d="M'+x+' '+y+' C'+(x-12)+' '+(y-22)+' '+x+' '+(y-30)+' '+(x+10)+' '+(y-48)+' C'+(x+24)+' '+(y-22)+' '+(x+12)+' '+(y-10)+' '+x+' '+y+'Z" fill="#F97316" opacity="0.55"/>';
      }
      heat.innerHTML=h;
      qt.textContent='Q = '+Q.toFixed(0)+' J';
      res.innerHTML='焦耳定律：Q = I²Rt。电阻和时间一定时，电流变为 2 倍，产生的热量变为 4 倍。<br>可用于解释电热器工作、导线发热和保险丝熔断。';
    }
    ie.oninput=update;re.oninput=update;te.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-sound-wave',
    group: '🌊 波动与声学',
    name: '响度与音调',
    emoji: '🎵',
    desc: '调节振幅和频率，比较响度与音调的区别',
    params: [
      { key: 'amp', label: '振幅', type: 'number', min: 10, max: 90, step: 5, defaultValue: 45 },
      { key: 'freq', label: '频率/Hz', type: 'number', min: 1, max: 12, step: 1, defaultValue: 5 },
    ],
    buildHTML: (params, rootId) => {
      const amp = num(params, 'amp', 45)
      const freq = num(params, 'freq', 5)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🎵 声波：振幅影响响度，频率影响音调</div>
    <div class="pl-note">看波形判断声音特征</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>振幅</span><span class="pl-value" data-a-val></span></div><input data-a type="range" min="10" max="90" step="5" value="${n(amp)}"></div>
      <div class="pl-row"><div class="pl-label"><span>频率</span><span class="pl-value" data-f-val></span></div><input data-f type="range" min="1" max="12" step="1" value="${n(freq)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <canvas data-canvas width="680" height="414"></canvas>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var cv=root.querySelector('[data-canvas]'), ctx=cv.getContext('2d');
    var ae=root.querySelector('[data-a]'), fe=root.querySelector('[data-f]');
    var av=root.querySelector('[data-a-val]'), fv=root.querySelector('[data-f-val]'), res=root.querySelector('[data-result]');
    function draw(){
      var A=Number(ae.value), F=Number(fe.value);
      av.textContent=A.toFixed(0); fv.textContent=F.toFixed(0)+' Hz';
      ctx.clearRect(0,0,680,414); ctx.fillStyle='#fff'; ctx.fillRect(0,0,680,414);
      ctx.strokeStyle='#E5E7EB'; ctx.lineWidth=1;
      for(var x=40;x<=640;x+=40){ctx.beginPath();ctx.moveTo(x,70);ctx.lineTo(x,340);ctx.stroke();}
      for(var y=90;y<=330;y+=40){ctx.beginPath();ctx.moveTo(40,y);ctx.lineTo(640,y);ctx.stroke();}
      ctx.strokeStyle='#94A3B8'; ctx.lineWidth=2; ctx.beginPath(); ctx.moveTo(40,207); ctx.lineTo(640,207); ctx.stroke();
      ctx.strokeStyle='#0284C7'; ctx.lineWidth=5; ctx.beginPath();
      for(var px=40;px<=640;px+=2){
        var yy=207 + A*Math.sin((px-40)/600*Math.PI*2*F);
        if(px===40)ctx.moveTo(px,yy); else ctx.lineTo(px,yy);
      }
      ctx.stroke();
      ctx.fillStyle='#075985'; ctx.font='bold 22px sans-serif'; ctx.fillText('振幅越大 → 响度越大',54,52);
      ctx.fillText('频率越高 → 音调越高',374,52);
      res.innerHTML='振幅表示声源振动幅度，决定响度；频率表示每秒振动次数，决定音调。<br>同样响度可以有不同音调，同样音调也可以有不同响度。';
    }
    ae.oninput=draw;fe.oninput=draw;draw();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-plane-mirror',
    group: '🔍 光学实验',
    name: '平面镜成像',
    emoji: '🪞',
    desc: '调节物体到镜面的距离，观察像与物等大、等距、左右相反',
    params: [
      { key: 'dist', label: '物距/cm', type: 'number', min: 2, max: 24, step: 1, defaultValue: 10 },
    ],
    buildHTML: (params, rootId) => {
      const dist = num(params, 'dist', 10)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🪞 平面镜成像：像与物关于镜面对称</div>
    <div class="pl-note">像距 = 物距，虚像不可用光屏承接</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>物距</span><span class="pl-value" data-d-val></span></div><input data-d type="range" min="2" max="24" step="1" value="${n(dist)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#fff"/>
        <line x1="340" y1="54" x2="340" y2="360" stroke="#0284C7" stroke-width="6"/>
        <text x="314" y="42" font-size="18" font-weight="900" fill="#0284C7">平面镜</text>
        <g data-object></g>
        <g data-image></g>
        <g data-rays></g>
        <text x="64" y="360" data-text font-size="22" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var d=root.querySelector('[data-d]'), dv=root.querySelector('[data-d-val]');
    var obj=root.querySelector('[data-object]'), img=root.querySelector('[data-image]'), rays=root.querySelector('[data-rays]');
    var text=root.querySelector('[data-text]'), res=root.querySelector('[data-result]');
    function person(x,op,label,color){
      return '<circle cx="'+x+'" cy="150" r="20" fill="'+color+'" opacity="'+op+'"/><line x1="'+x+'" y1="170" x2="'+x+'" y2="250" stroke="'+color+'" stroke-width="8" opacity="'+op+'"/><line x1="'+x+'" y1="190" x2="'+(x-38)+'" y2="224" stroke="'+color+'" stroke-width="7" opacity="'+op+'"/><line x1="'+x+'" y1="190" x2="'+(x+38)+'" y2="224" stroke="'+color+'" stroke-width="7" opacity="'+op+'"/><line x1="'+x+'" y1="250" x2="'+(x-34)+'" y2="310" stroke="'+color+'" stroke-width="7" opacity="'+op+'"/><line x1="'+x+'" y1="250" x2="'+(x+34)+'" y2="310" stroke="'+color+'" stroke-width="7" opacity="'+op+'"/><text x="'+(x-16)+'" y="336" font-size="17" font-weight="900" fill="'+color+'" opacity="'+op+'">'+label+'</text>';
    }
    function update(){
      var D=Number(d.value), px=340-D*8, ix=340+D*8;
      dv.textContent=D.toFixed(0)+' cm';
      obj.innerHTML=person(px,1,'物','#2563EB');
      img.innerHTML=person(ix,0.38,'像','#EF4444');
      rays.innerHTML='<line x1="'+px+'" y1="150" x2="340" y2="132" stroke="#F59E0B" stroke-width="3"/><line x1="340" y1="132" x2="'+(px+50)+'" y2="80" stroke="#F59E0B" stroke-width="3"/><line x1="340" y1="132" x2="'+ix+'" y2="150" stroke="#EF4444" stroke-width="2" stroke-dasharray="8 6"/>';
      text.textContent='物距 '+D.toFixed(0)+' cm，像距 '+D.toFixed(0)+' cm';
      res.innerHTML='平面镜成像特点：像与物大小相等，像距等于物距，连线垂直镜面，所成像是虚像。';
    }
    d.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
