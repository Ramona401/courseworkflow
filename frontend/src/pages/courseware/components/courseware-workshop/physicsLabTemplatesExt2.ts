/**
 * physicsLabTemplatesExt2.ts — 物理实验室第二批扩展模板
 *
 * 第二批新增：
 *   1. 滑动变阻器
 *   2. 伏安法测电阻
 *   3. 电磁铁
 *   4. 家庭电路安全
 */

import type { PhysicsLabTemplate, PhysicsLabParamValue } from './physicsLabUtils'

function num(params: Record<string, PhysicsLabParamValue>, key: string, fallback: number): number {
  const v = params[key]
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback
}

function n(v: number): string {
  return parseFloat(v.toFixed(3)).toString()
}

function bool(params: Record<string, PhysicsLabParamValue>, key: string, fallback: boolean): boolean {
  const v = params[key]
  return typeof v === 'boolean' ? v : fallback
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

export const PHYSICS_LAB_TEMPLATES_EXT2: PhysicsLabTemplate[] = [
  {
    id: 'lab-rheostat',
    group: '🔌 电学实验',
    name: '滑动变阻器',
    emoji: '🎚️',
    desc: '移动滑片改变接入电阻，观察电流和灯泡亮度变化',
    params: [
      { key: 'u', label: '电源电压/V', type: 'number', min: 3, max: 12, step: 1, defaultValue: 6 },
      { key: 'pos', label: '滑片位置/%', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
      { key: 'lampR', label: '灯泡电阻/Ω', type: 'number', min: 2, max: 30, step: 1, defaultValue: 10 },
    ],
    buildHTML: (params, rootId) => {
      const u = num(params, 'u', 6)
      const pos = num(params, 'pos', 45)
      const lampR = num(params, 'lampR', 10)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🎚️ 滑动变阻器：改变接入电阻</div>
    <div class="pl-note">滑片移动 → 电阻变化 → 电流变化 → 灯泡亮度变化</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>电压 U</span><span class="pl-value" data-u-val></span></div><input data-u type="range" min="3" max="12" step="1" value="${n(u)}"></div>
      <div class="pl-row"><div class="pl-label"><span>滑片位置</span><span class="pl-value" data-p-val></span></div><input data-p type="range" min="0" max="100" step="1" value="${n(pos)}"></div>
      <div class="pl-row"><div class="pl-label"><span>灯泡电阻</span><span class="pl-value" data-l-val></span></div><input data-l type="range" min="2" max="30" step="1" value="${n(lampR)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M110 212 H220 M460 212 H560 M560 212 V316 H110 V212 M110 212 V96 H560 V212" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
        <line x1="150" y1="78" x2="150" y2="114" stroke="#0F172A" stroke-width="5"/>
        <line x1="174" y1="88" x2="174" y2="106" stroke="#0F172A" stroke-width="5"/>
        <circle cx="340" cy="316" r="42" data-lamp fill="#FEF3C7" stroke="#F59E0B" stroke-width="4"/>
        <path d="M316 316 q24 -34 48 0" fill="none" stroke="#92400E" stroke-width="4"/>
        <rect x="220" y="178" width="240" height="68" rx="18" fill="#F1F5F9" stroke="#64748B" stroke-width="4"/>
        <path d="M244 212 q18 -28 36 0 t36 0 t36 0 t36 0 t36 0" fill="none" stroke="#DC2626" stroke-width="4"/>
        <line data-slider x1="340" y1="160" x2="340" y2="252" stroke="#0284C7" stroke-width="6" stroke-linecap="round"/>
        <circle data-knob cx="340" cy="160" r="12" fill="#0284C7"/>
        <text x="232" y="162" font-size="18" font-weight="900" fill="#475569">滑动变阻器</text>
        <text x="452" y="120" data-current font-size="24" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var u=root.querySelector('[data-u]'), p=root.querySelector('[data-p]'), l=root.querySelector('[data-l]');
    var uv=root.querySelector('[data-u-val]'), pv=root.querySelector('[data-p-val]'), lv=root.querySelector('[data-l-val]');
    var slider=root.querySelector('[data-slider]'), knob=root.querySelector('[data-knob]'), lamp=root.querySelector('[data-lamp]');
    var cur=root.querySelector('[data-current]'), res=root.querySelector('[data-result]');
    function update(){
      var U=Number(u.value), P=Number(p.value), L=Number(l.value);
      var Rv=1+P/100*49, I=U/(Rv+L), brightness=Math.min(1,I/0.7);
      uv.textContent=U.toFixed(0)+' V'; pv.textContent=P.toFixed(0)+'%'; lv.textContent=L.toFixed(0)+' Ω';
      var x=220+P/100*240;
      slider.setAttribute('x1',x);slider.setAttribute('x2',x);knob.setAttribute('cx',x);
      lamp.setAttribute('fill',brightness>0.66?'#FDE68A':brightness>0.3?'#FEF3C7':'#F8FAFC');
      lamp.setAttribute('opacity',String(0.45+brightness*0.55));
      cur.textContent='I = '+I.toFixed(2)+' A';
      res.innerHTML='滑片移动改变接入电路的电阻。接入电阻越大，总电阻越大，电流越小，灯泡越暗。<br>当前变阻器接入约 '+Rv.toFixed(1)+' Ω。';
    }
    u.oninput=update;p.oninput=update;l.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-volt-amp-resistance',
    group: '🔌 电学实验',
    name: '伏安法测电阻',
    emoji: '📟',
    desc: '读取电压表和电流表，用 R=U/I 计算未知电阻',
    params: [
      { key: 'r', label: '未知电阻/Ω', type: 'number', min: 2, max: 50, step: 1, defaultValue: 18 },
      { key: 'u', label: '电压/V', type: 'number', min: 1, max: 12, step: 0.5, defaultValue: 6 },
      { key: 'error', label: '读数误差/%', type: 'number', min: 0, max: 10, step: 1, defaultValue: 2 },
    ],
    buildHTML: (params, rootId) => {
      const r = num(params, 'r', 18)
      const u = num(params, 'u', 6)
      const error = num(params, 'error', 2)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">📟 伏安法测电阻：R = U / I</div>
    <div class="pl-note">电压表并联，电流表串联</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>未知电阻</span><span class="pl-value" data-r-val></span></div><input data-r type="range" min="2" max="50" step="1" value="${n(r)}"></div>
      <div class="pl-row"><div class="pl-label"><span>电压</span><span class="pl-value" data-u-val></span></div><input data-u type="range" min="1" max="12" step="0.5" value="${n(u)}"></div>
      <div class="pl-row"><div class="pl-label"><span>读数误差</span><span class="pl-value" data-e-val></span></div><input data-e type="range" min="0" max="10" step="1" value="${n(error)}"></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M120 210 H250 M430 210 H560 M560 210 V315 H120 V210 M120 210 V96 H560 V210" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
        <rect x="250" y="176" width="180" height="68" rx="16" fill="#FEE2E2" stroke="#EF4444" stroke-width="4"/>
        <text x="314" y="218" font-size="24" font-weight="900" fill="#EF4444">Rx</text>
        <circle cx="340" cy="315" r="44" fill="#E0F2FE" stroke="#0284C7" stroke-width="4"/>
        <text x="326" y="322" font-size="25" font-weight="900" fill="#0284C7">A</text>
        <circle cx="340" cy="112" r="44" fill="#EEF2FF" stroke="#7C3AED" stroke-width="4"/>
        <text x="326" y="119" font-size="25" font-weight="900" fill="#7C3AED">V</text>
        <path d="M250 190 C230 138 286 112 296 112 M384 112 C394 112 450 138 430 190" fill="none" stroke="#7C3AED" stroke-width="4"/>
        <text x="450" y="118" data-u-text font-size="22" font-weight="900" fill="#7C3AED"></text>
        <text x="450" y="152" data-i-text font-size="22" font-weight="900" fill="#0284C7"></text>
        <text x="224" y="372" data-r-text font-size="28" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var re=root.querySelector('[data-r]'), ue=root.querySelector('[data-u]'), ee=root.querySelector('[data-e]');
    var rv=root.querySelector('[data-r-val]'), uv=root.querySelector('[data-u-val]'), ev=root.querySelector('[data-e-val]');
    var ut=root.querySelector('[data-u-text]'), it=root.querySelector('[data-i-text]'), rt=root.querySelector('[data-r-text]'), res=root.querySelector('[data-result]');
    function update(){
      var R=Number(re.value), U=Number(ue.value), E=Number(ee.value);
      var I=U/R;
      var Uread=U*(1+E/100*0.4), Iread=I*(1-E/100*0.35), Rread=Uread/Iread;
      rv.textContent=R.toFixed(0)+' Ω'; uv.textContent=U.toFixed(1)+' V'; ev.textContent=E.toFixed(0)+'%';
      ut.textContent='U = '+Uread.toFixed(2)+' V';
      it.textContent='I = '+Iread.toFixed(2)+' A';
      rt.textContent='测得 R ≈ '+Rread.toFixed(1)+' Ω';
      res.innerHTML='伏安法测电阻：电压表并联在被测电阻两端，电流表串联在电路中。<br>根据读数计算 R = U/I，多次测量取平均可减小误差。';
    }
    re.oninput=update;ue.oninput=update;ee.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-electromagnet',
    group: '🧲 电磁实验',
    name: '电磁铁',
    emoji: '🧲',
    desc: '调节电流、线圈匝数和铁芯，观察磁性强弱变化',
    params: [
      { key: 'turns', label: '线圈匝数', type: 'number', min: 5, max: 60, step: 1, defaultValue: 28 },
      { key: 'current', label: '电流/A', type: 'number', min: 0, max: 5, step: 0.1, defaultValue: 2.2 },
      { key: 'core', label: '插入铁芯', type: 'boolean', defaultValue: true },
    ],
    buildHTML: (params, rootId) => {
      const turns = num(params, 'turns', 28)
      const current = num(params, 'current', 2.2)
      const core = bool(params, 'core', true)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🧲 电磁铁：线圈通电产生磁性</div>
    <div class="pl-note">电流越大、匝数越多、有铁芯，磁性越强</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>线圈匝数</span><span class="pl-value" data-n-val></span></div><input data-n type="range" min="5" max="60" step="1" value="${n(turns)}"></div>
      <div class="pl-row"><div class="pl-label"><span>电流</span><span class="pl-value" data-i-val></span></div><input data-i type="range" min="0" max="5" step="0.1" value="${n(current)}"></div>
      <div class="pl-row"><button data-core>${core ? '移除铁芯' : '插入铁芯'}</button></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <rect x="240" y="170" width="200" height="70" rx="24" data-core-rect fill="#CBD5E1" stroke="#64748B" stroke-width="4"/>
        <g data-coil></g>
        <g data-field></g>
        <g data-nails></g>
        <text x="270" y="152" font-size="18" font-weight="900" fill="#475569">线圈 + 铁芯</text>
        <text x="466" y="104" data-strength font-size="26" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var ne=root.querySelector('[data-n]'), ie=root.querySelector('[data-i]'), btn=root.querySelector('[data-core]');
    var nv=root.querySelector('[data-n-val]'), iv=root.querySelector('[data-i-val]');
    var coil=root.querySelector('[data-coil]'), field=root.querySelector('[data-field]'), nails=root.querySelector('[data-nails]'), coreRect=root.querySelector('[data-core-rect]');
    var st=root.querySelector('[data-strength]'), res=root.querySelector('[data-result]');
    var hasCore=${core ? 'true' : 'false'};
    btn.onclick=function(){hasCore=!hasCore;btn.textContent=hasCore?'移除铁芯':'插入铁芯';update();};
    function update(){
      var N=Number(ne.value), I=Number(ie.value), S=N*I*(hasCore?1.8:1);
      nv.textContent=N.toFixed(0)+' 匝'; iv.textContent=I.toFixed(1)+' A';
      coreRect.setAttribute('opacity',hasCore?'1':'0.22');
      var c='';
      for(var k=0;k<N;k++){
        var x=246+(k%24)*8;
        c+='<ellipse cx="'+x+'" cy="205" rx="18" ry="48" fill="none" stroke="#0284C7" stroke-width="2" opacity="0.62"/>';
      }
      coil.innerHTML=c;
      var f='';
      for(var j=0;j<Math.min(10,Math.floor(S/20));j++){
        f+='<ellipse cx="340" cy="205" rx="'+(130+j*18)+'" ry="'+(54+j*11)+'" fill="none" stroke="#38BDF8" stroke-width="3" opacity="'+(0.52-j*0.035)+'"/>';
      }
      field.innerHTML=f;
      var count=Math.min(18,Math.floor(S/12));
      var ns='';
      for(var i=0;i<count;i++){
        var x=160+(i%9)*42, y=310-Math.floor(i/9)*34;
        ns+='<path d="M'+x+' '+y+' L'+(x+22)+' '+(y-14)+'" stroke="#64748B" stroke-width="5" stroke-linecap="round"/><circle cx="'+(x+23)+'" cy="'+(y-15)+'" r="4" fill="#94A3B8"/>';
      }
      nails.innerHTML=ns;
      st.textContent='磁性 '+(S<40?'较弱':S<120?'明显':'很强');
      res.innerHTML='电磁铁磁性强弱与电流大小、线圈匝数、有无铁芯有关。<br>插入铁芯后磁性显著增强，断电后磁性明显减弱或消失。';
    }
    ne.oninput=update;ie.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-home-circuit-safety',
    group: '🔌 电学实验',
    name: '家庭电路安全',
    emoji: '🏠',
    desc: '观察大功率用电器、过载和漏电保护的触发条件',
    params: [
      { key: 'power', label: '总功率/kW', type: 'number', min: 0.2, max: 9, step: 0.1, defaultValue: 3.2 },
      { key: 'limit', label: '空气开关额定电流/A', type: 'number', min: 5, max: 40, step: 1, defaultValue: 16 },
      { key: 'leak', label: '模拟漏电', type: 'boolean', defaultValue: false },
    ],
    buildHTML: (params, rootId) => {
      const power = num(params, 'power', 3.2)
      const limit = num(params, 'limit', 16)
      const leak = bool(params, 'leak', false)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🏠 家庭电路安全：过载与漏电保护</div>
    <div class="pl-note">家庭电路电压按 220V 估算电流</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row"><div class="pl-label"><span>总功率</span><span class="pl-value" data-p-val></span></div><input data-p type="range" min="0.2" max="9" step="0.1" value="${n(power)}"></div>
      <div class="pl-row"><div class="pl-label"><span>额定电流</span><span class="pl-value" data-l-val></span></div><input data-l type="range" min="5" max="40" step="1" value="${n(limit)}"></div>
      <div class="pl-row"><button data-leak>${leak ? '取消漏电' : '模拟漏电'}</button></div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <path d="M100 104 H580 V314 H100 Z" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="4"/>
        <line x1="130" y1="166" x2="550" y2="166" stroke="#DC2626" stroke-width="5"/>
        <line x1="130" y1="250" x2="550" y2="250" stroke="#2563EB" stroke-width="5"/>
        <rect x="152" y="128" width="92" height="76" rx="14" data-breaker fill="#E0F2FE" stroke="#0284C7" stroke-width="4"/>
        <text x="166" y="174" font-size="17" font-weight="900" fill="#075985">空开</text>
        <rect x="304" y="128" width="92" height="76" rx="14" fill="#FEF3C7" stroke="#F59E0B" stroke-width="4"/>
        <text x="318" y="174" font-size="17" font-weight="900" fill="#92400E">用电器</text>
        <rect x="450" y="128" width="92" height="76" rx="14" data-leakbox fill="#F8FAFC" stroke="#64748B" stroke-width="4"/>
        <text x="462" y="174" font-size="17" font-weight="900" fill="#475569">漏保</text>
        <g data-spark></g>
        <text x="135" y="346" data-current font-size="27" font-weight="900" fill="#075985"></text>
        <text x="420" y="346" data-state font-size="27" font-weight="900" fill="#DC2626"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var p=root.querySelector('[data-p]'), l=root.querySelector('[data-l]'), btn=root.querySelector('[data-leak]');
    var pv=root.querySelector('[data-p-val]'), lv=root.querySelector('[data-l-val]');
    var breaker=root.querySelector('[data-breaker]'), leakbox=root.querySelector('[data-leakbox]'), spark=root.querySelector('[data-spark]');
    var cur=root.querySelector('[data-current]'), state=root.querySelector('[data-state]'), res=root.querySelector('[data-result]');
    var leak=${leak ? 'true' : 'false'};
    btn.onclick=function(){leak=!leak;btn.textContent=leak?'取消漏电':'模拟漏电';update();};
    function update(){
      var P=Number(p.value), L=Number(l.value), I=P*1000/220, overload=I>L;
      pv.textContent=P.toFixed(1)+' kW'; lv.textContent=L.toFixed(0)+' A';
      breaker.setAttribute('fill',overload?'#FEE2E2':'#E0F2FE');
      leakbox.setAttribute('fill',leak?'#FEE2E2':'#F8FAFC');
      var sp='';
      if(overload||leak){
        for(var i=0;i<8;i++){
          sp+='<path d="M'+(240+i*42)+' '+(112+(i%2)*150)+' l14 -22 l8 18 l16 -26" fill="none" stroke="#F59E0B" stroke-width="4" stroke-linecap="round" opacity="0.9"/>';
        }
      }
      spark.innerHTML=sp;
      cur.textContent='估算电流 I ≈ '+I.toFixed(1)+' A';
      state.textContent=leak?'漏保跳闸':overload?'过载跳闸':'正常工作';
      res.innerHTML=leak?'漏电保护器检测到火线和零线电流不平衡，会迅速切断电路，保护人身安全。':'总功率越大，电流越大。若电流超过空气开关额定值，可能过载跳闸。<br>家庭电路应避免多个大功率用电器同时使用同一线路。';
    }
    p.oninput=update;l.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
