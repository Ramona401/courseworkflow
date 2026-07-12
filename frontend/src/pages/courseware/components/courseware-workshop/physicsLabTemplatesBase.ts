/**
 * physicsLabTemplates.ts — 物理实验室模板注册表（第3批A新增，2026-07-09）
 *
 * 本批补齐 PhysicsSceneModal 难以覆盖的非力学主题：
 *   1. 电学：欧姆定律、串并联电路
 *   2. 光学：凸透镜成像、反射与折射
 *   3. 波动：机械波参数与传播
 *   4. 电磁：电磁感应定性演示
 *
 * 模板实现方式：
 *   - 全部为纯 HTML + SVG/Canvas + 原生 JS。
 *   - 每个模板的 buildHTML 输出完整自包含组件。
 *   - 运行时只查询 rootId 内部 DOM，避免同页多个实验互相干扰。
 */
import type { PhysicsLabTemplate, PhysicsLabParamValue } from './physicsLabUtils'

// ============================================================
// 共享辅助
// ============================================================

/** 读取数字参数 */
function num(params: Record<string, PhysicsLabParamValue>, key: string, fallback: number): number {
  const v = params[key]
  return typeof v === 'number' && Number.isFinite(v) ? v : fallback
}

/** 读取布尔参数 */
function bool(params: Record<string, PhysicsLabParamValue>, key: string, fallback: boolean): boolean {
  const v = params[key]
  return typeof v === 'boolean' ? v : fallback
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
    + '#' + rootId + ' .pl-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#E0F2FE,#F0F9FF);border-bottom:1px solid #E5E7EB;box-sizing:border-box;}\n'
    + '#' + rootId + ' .pl-title{font-size:15px;font-weight:800;color:#075985;}\n'
    + '#' + rootId + ' .pl-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .pl-body{height:calc(100% - 46px);display:grid;grid-template-columns:220px 1fr;min-height:0;}\n'
    + '#' + rootId + ' .pl-controls{padding:14px 14px;border-right:1px solid #E5E7EB;background:#F8FAFC;box-sizing:border-box;overflow:auto;}\n'
    + '#' + rootId + ' .pl-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .pl-row{margin-bottom:13px;}\n'
    + '#' + rootId + ' .pl-label{display:flex;justify-content:space-between;gap:8px;font-size:12px;font-weight:700;color:#334155;margin-bottom:6px;}\n'
    + '#' + rootId + ' .pl-value{font-weight:800;color:#0284C7;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9;}\n'
    + '#' + rootId + ' select{width:100%;padding:7px 8px;border-radius:8px;border:1px solid #CBD5E1;background:#fff;font-size:12px;outline:none;}\n'
    + '#' + rootId + ' button{border:none;border-radius:10px;padding:8px 14px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:12px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .pl-result{padding:10px 12px;border-radius:10px;background:#E0F2FE;color:#075985;font-size:12px;line-height:1.6;font-weight:600;}\n'
    + '#' + rootId + ' svg{width:100%;height:100%;display:block;}\n'
    + '#' + rootId + ' canvas{width:100%;height:100%;display:block;}\n'
    + '</style>\n'
}

/** 生成脚本闭合标签，避免源码中出现连续 closing script 字符串 */
const SCRIPT_END = '</' + 'script>'

// ============================================================
// 模板注册表
// ============================================================

export const PHYSICS_LAB_TEMPLATES: PhysicsLabTemplate[] = [
  {
    id: 'lab-ohm-law',
    group: '🔌 电学实验',
    name: '欧姆定律',
    emoji: '🔌',
    desc: '调节电压和电阻，实时观察电流 I=U/R 与功率 P=UI',
    params: [
      { key: 'u', label: '电压 U/V', type: 'number', min: 1, max: 24, step: 1, defaultValue: 6 },
      { key: 'r', label: '电阻 R/Ω', type: 'number', min: 1, max: 50, step: 1, defaultValue: 12 },
    ],
    buildHTML: (params, rootId) => {
      const u = num(params, 'u', 6)
      const r = num(params, 'r', 12)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🔌 欧姆定律：I = U / R</div>
    <div class="pl-note">改变电压或电阻，观察电流变化</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row">
        <div class="pl-label"><span>电压 U</span><span class="pl-value" data-u-val></span></div>
        <input data-u type="range" min="1" max="24" step="1" value="${n(u)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>电阻 R</span><span class="pl-value" data-r-val></span></div>
        <input data-r type="range" min="1" max="50" step="1" value="${n(r)}">
      </div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414" aria-label="欧姆定律电路图">
        <rect x="0" y="0" width="680" height="414" fill="#FFFFFF"/>
        <path d="M150 210 H250 M430 210 H540 M540 210 V310 H150 V210" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
        <path d="M150 210 V110 H540 V210" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
        <line x1="180" y1="92" x2="180" y2="128" stroke="#0F172A" stroke-width="5"/>
        <line x1="203" y1="102" x2="203" y2="118" stroke="#0F172A" stroke-width="5"/>
        <text x="158" y="78" font-size="20" font-weight="800" fill="#0F172A">电源</text>
        <rect x="250" y="176" width="180" height="68" rx="16" fill="#FEE2E2" stroke="#EF4444" stroke-width="4"/>
        <path d="M270 210 q15 -24 30 0 t30 0 t30 0 t30 0" fill="none" stroke="#DC2626" stroke-width="4"/>
        <text x="310" y="166" font-size="20" font-weight="800" fill="#DC2626">电阻 R</text>
        <circle cx="345" cy="310" r="44" fill="#E0F2FE" stroke="#0284C7" stroke-width="4"/>
        <text x="331" y="317" font-size="24" font-weight="900" fill="#0284C7">A</text>
        <g data-arrows fill="#0EA5E9">
          <path d="M322 108 l18 0 l-9 -14z"/>
          <path d="M520 242 l0 18 l14 -9z"/>
          <path d="M286 322 l-18 0 l9 14z"/>
        </g>
        <text x="458" y="124" font-size="17" fill="#64748B">电流方向</text>
        <text x="458" y="152" data-i-text font-size="28" font-weight="900" fill="#0284C7"></text>
        <text x="458" y="188" data-p-text font-size="18" font-weight="700" fill="#475569"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var u=root.querySelector('[data-u]');
    var r=root.querySelector('[data-r]');
    var uv=root.querySelector('[data-u-val]');
    var rv=root.querySelector('[data-r-val]');
    var res=root.querySelector('[data-result]');
    var it=root.querySelector('[data-i-text]');
    var pt=root.querySelector('[data-p-text]');
    var arrows=root.querySelector('[data-arrows]');
    function update(){
      var U=Number(u.value), R=Number(r.value), I=U/R, P=U*I;
      uv.textContent=U.toFixed(0)+' V';
      rv.textContent=R.toFixed(0)+' Ω';
      it.textContent='I = '+I.toFixed(2)+' A';
      pt.textContent='P = UI = '+P.toFixed(2)+' W';
      res.innerHTML='当电阻不变时，电压越大，电流越大；当电压不变时，电阻越大，电流越小。<br><b>I = '+U.toFixed(0)+' / '+R.toFixed(0)+' = '+I.toFixed(2)+' A</b>';
      var s=Math.max(0.45,Math.min(1.8,I/0.6));
      arrows.setAttribute('transform','scale('+s+' '+s+') translate('+((1-s)*340/s)+' '+((1-s)*207/s)+')');
    }
    u.oninput=update;r.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-series-parallel',
    group: '🔌 电学实验',
    name: '串联与并联电路',
    emoji: '🧭',
    desc: '比较串联/并联的等效电阻、总电流和分支电流',
    params: [
      { key: 'u', label: '电源电压 U/V', type: 'number', min: 3, max: 24, step: 1, defaultValue: 12 },
      { key: 'r1', label: '电阻 R1/Ω', type: 'number', min: 1, max: 50, step: 1, defaultValue: 10 },
      { key: 'r2', label: '电阻 R2/Ω', type: 'number', min: 1, max: 50, step: 1, defaultValue: 20 },
      { key: 'parallel', label: '默认并联', type: 'boolean', defaultValue: false },
    ],
    buildHTML: (params, rootId) => {
      const u = num(params, 'u', 12)
      const r1 = num(params, 'r1', 10)
      const r2 = num(params, 'r2', 20)
      const parallel = bool(params, 'parallel', false)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🧭 串联 / 并联电路对比</div>
    <div class="pl-note">切换连接方式，观察等效电阻</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row">
        <div class="pl-label"><span>连接方式</span><span class="pl-value" data-mode-val></span></div>
        <select data-mode>
          <option value="series"${parallel ? '' : ' selected'}>串联</option>
          <option value="parallel"${parallel ? ' selected' : ''}>并联</option>
        </select>
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>电压 U</span><span class="pl-value" data-u-val></span></div>
        <input data-u type="range" min="3" max="24" step="1" value="${n(u)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>R1</span><span class="pl-value" data-r1-val></span></div>
        <input data-r1 type="range" min="1" max="50" step="1" value="${n(r1)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>R2</span><span class="pl-value" data-r2-val></span></div>
        <input data-r2 type="range" min="1" max="50" step="1" value="${n(r2)}">
      </div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#fff"/>
        <g data-series>
          <path d="M120 210 H215 M315 210 H365 M465 210 H560 M560 210 V310 H120 V210 M120 210 V110 H560 V210" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
          <rect x="215" y="178" width="100" height="64" rx="14" fill="#DBEAFE" stroke="#2563EB" stroke-width="4"/>
          <rect x="365" y="178" width="100" height="64" rx="14" fill="#FEE2E2" stroke="#EF4444" stroke-width="4"/>
          <text x="247" y="217" font-size="20" font-weight="900" fill="#2563EB">R1</text>
          <text x="397" y="217" font-size="20" font-weight="900" fill="#EF4444">R2</text>
        </g>
        <g data-parallel style="display:none">
          <path d="M120 210 H210 M470 210 H560 M560 210 V310 H120 V210 M120 210 V110 H560 V210" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
          <path d="M210 145 H470 M210 275 H470 M210 145 V275 M470 145 V275" fill="none" stroke="#334155" stroke-width="5" stroke-linecap="round"/>
          <rect x="285" y="113" width="110" height="64" rx="14" fill="#DBEAFE" stroke="#2563EB" stroke-width="4"/>
          <rect x="285" y="243" width="110" height="64" rx="14" fill="#FEE2E2" stroke="#EF4444" stroke-width="4"/>
          <text x="322" y="153" font-size="20" font-weight="900" fill="#2563EB">R1</text>
          <text x="322" y="283" font-size="20" font-weight="900" fill="#EF4444">R2</text>
        </g>
        <line x1="158" y1="92" x2="158" y2="128" stroke="#0F172A" stroke-width="5"/>
        <line x1="181" y1="102" x2="181" y2="118" stroke="#0F172A" stroke-width="5"/>
        <text x="136" y="78" font-size="18" font-weight="800" fill="#0F172A">电源</text>
        <text x="90" y="370" data-main-text font-size="22" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var mode=root.querySelector('[data-mode]');
    var u=root.querySelector('[data-u]');
    var r1=root.querySelector('[data-r1]');
    var r2=root.querySelector('[data-r2]');
    var res=root.querySelector('[data-result]');
    var mv=root.querySelector('[data-mode-val]');
    var uv=root.querySelector('[data-u-val]');
    var r1v=root.querySelector('[data-r1-val]');
    var r2v=root.querySelector('[data-r2-val]');
    var series=root.querySelector('[data-series]');
    var parallel=root.querySelector('[data-parallel]');
    var main=root.querySelector('[data-main-text]');
    function update(){
      var U=Number(u.value), R1=Number(r1.value), R2=Number(r2.value);
      var isP=mode.value==='parallel';
      var Req=isP ? (R1*R2)/(R1+R2) : (R1+R2);
      var I=U/Req;
      mv.textContent=isP?'并联':'串联';
      uv.textContent=U.toFixed(0)+' V';
      r1v.textContent=R1.toFixed(0)+' Ω';
      r2v.textContent=R2.toFixed(0)+' Ω';
      series.style.display=isP?'none':'block';
      parallel.style.display=isP?'block':'none';
      if(isP){
        res.innerHTML='并联：各支路电压相等，总电流等于各支路电流之和。<br>R等效 = '+Req.toFixed(2)+' Ω；I总 = '+I.toFixed(2)+' A；I1 = '+(U/R1).toFixed(2)+' A，I2 = '+(U/R2).toFixed(2)+' A';
      }else{
        res.innerHTML='串联：电流处处相等，总电阻等于各电阻之和。<br>R等效 = '+Req.toFixed(2)+' Ω；I = '+I.toFixed(2)+' A；U1 = '+(I*R1).toFixed(1)+' V，U2 = '+(I*R2).toFixed(1)+' V';
      }
      main.textContent=(isP?'并联':'串联')+'：R等效 = '+Req.toFixed(2)+' Ω，I总 = '+I.toFixed(2)+' A';
    }
    mode.onchange=update;u.oninput=update;r1.oninput=update;r2.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-convex-lens',
    group: '🔍 光学实验',
    name: '凸透镜成像',
    emoji: '🔍',
    desc: '调节物距和焦距，观察像距、放大率与倒立/正立',
    params: [
      { key: 'f', label: '焦距 f/cm', type: 'number', min: 4, max: 20, step: 1, defaultValue: 10 },
      { key: 'u', label: '物距 u/cm', type: 'number', min: 5, max: 45, step: 1, defaultValue: 25 },
    ],
    buildHTML: (params, rootId) => {
      const f = num(params, 'f', 10)
      const u = num(params, 'u', 25)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🔍 凸透镜成像：1/f = 1/u + 1/v</div>
    <div class="pl-note">调物距，观察实像/虚像与放大率</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row">
        <div class="pl-label"><span>焦距 f</span><span class="pl-value" data-f-val></span></div>
        <input data-f type="range" min="4" max="20" step="1" value="${n(f)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>物距 u</span><span class="pl-value" data-u-val></span></div>
        <input data-u type="range" min="5" max="45" step="1" value="${n(u)}">
      </div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#fff"/>
        <line x1="40" y1="210" x2="640" y2="210" stroke="#CBD5E1" stroke-width="3"/>
        <line x1="340" y1="52" x2="340" y2="366" stroke="#0284C7" stroke-width="4"/>
        <path d="M340 58 C304 130 304 290 340 362 C376 290 376 130 340 58Z" fill="#E0F2FE" stroke="#0284C7" stroke-width="3" opacity="0.95"/>
        <text x="318" y="44" font-size="18" font-weight="900" fill="#0284C7">凸透镜</text>
        <circle data-fl cx="260" cy="210" r="5" fill="#F59E0B"/>
        <circle data-fr cx="420" cy="210" r="5" fill="#F59E0B"/>
        <text data-flt x="250" y="236" font-size="14" font-weight="800" fill="#F59E0B">F</text>
        <text data-frt x="414" y="236" font-size="14" font-weight="800" fill="#F59E0B">F</text>
        <g data-rays stroke-width="3" fill="none" stroke-linecap="round"></g>
        <g data-object></g>
        <g data-image></g>
        <text x="54" y="364" data-formula font-size="20" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var fEl=root.querySelector('[data-f]');
    var uEl=root.querySelector('[data-u]');
    var fv=root.querySelector('[data-f-val]');
    var uv=root.querySelector('[data-u-val]');
    var res=root.querySelector('[data-result]');
    var fl=root.querySelector('[data-fl]');
    var fr=root.querySelector('[data-fr]');
    var flt=root.querySelector('[data-flt]');
    var frt=root.querySelector('[data-frt]');
    var obj=root.querySelector('[data-object]');
    var img=root.querySelector('[data-image]');
    var rays=root.querySelector('[data-rays]');
    var formula=root.querySelector('[data-formula]');
    function arrow(x,y,h,color,label){
      var y2=y-h;
      return '<line x1="'+x+'" y1="'+y+'" x2="'+x+'" y2="'+y2+'" stroke="'+color+'" stroke-width="5"/><polygon points="'+x+','+y2+' '+(x-8)+','+(y2+18)+' '+(x+8)+','+(y2+18)+'" fill="'+color+'"/><text x="'+(x-14)+'" y="'+(y+24)+'" font-size="15" font-weight="800" fill="'+color+'">'+label+'</text>';
    }
    function line(x1,y1,x2,y2,color,dash){
      return '<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2+'" stroke="'+color+'" stroke-width="3" '+(dash?'stroke-dasharray="8 6"':'')+'/>';
    }
    function update(){
      var f=Number(fEl.value), u=Number(uEl.value);
      var scale=5;
      var lensX=340, axisY=210, objH=82;
      var objX=lensX-u*scale;
      var denom=1/f-1/u;
      var v=Math.abs(denom)<0.001 ? 999 : 1/denom;
      var imgX=lensX+v*scale;
      var m=-v/u;
      var imgH=objH*m;
      fv.textContent=f.toFixed(0)+' cm';
      uv.textContent=u.toFixed(0)+' cm';
      var fpx=f*scale;
      fl.setAttribute('cx',String(lensX-fpx)); fr.setAttribute('cx',String(lensX+fpx));
      flt.setAttribute('x',String(lensX-fpx-10)); frt.setAttribute('x',String(lensX+fpx-6));
      obj.innerHTML=arrow(objX,axisY,objH,'#2563EB','物');
      if(u===f){
        img.innerHTML='';
        rays.innerHTML=line(objX,axisY-objH,lensX,axisY-objH,'#DC2626',false)+line(lensX,axisY-objH,630,axisY-objH,'#DC2626',false);
        res.innerHTML='物体放在焦点上：出射光近似平行，像在无穷远处，屏上不能得到清晰像。';
        formula.textContent='u = f，像距趋于无穷远';
        return;
      }
      var real=v>0;
      var safeX=Math.max(40,Math.min(640,imgX));
      var safeH=Math.max(-120,Math.min(120,imgH));
      img.innerHTML=arrow(safeX,axisY,safeH,real?'#EF4444':'#059669',real?'实像':'虚像');
      rays.innerHTML=
        line(objX,axisY-objH,lensX,axisY-objH,'#DC2626',false)+
        line(lensX,axisY-objH,real?safeX:620,real?axisY-safeH:axisY-objH-(620-lensX)*(axisY-safeH-(axisY-objH))/(safeX-lensX),'#DC2626',false)+
        line(objX,axisY-objH,lensX,axisY,'#7C3AED',false)+
        line(lensX,axisY,real?safeX:620,real?axisY-safeH:axisY+(620-lensX)*(axisY-safeH-axisY)/(safeX-lensX),'#7C3AED',false);
      res.innerHTML='像距 v = '+v.toFixed(1)+' cm；放大率 m = '+m.toFixed(2)+'。<br>'+(real?'成倒立实像，可用光屏承接。':'成正立虚像，不能用光屏承接。');
      formula.textContent='f='+f.toFixed(0)+'cm，u='+u.toFixed(0)+'cm，v='+v.toFixed(1)+'cm';
    }
    fEl.oninput=update;uEl.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-refraction',
    group: '🔍 光学实验',
    name: '反射与折射',
    emoji: '🌈',
    desc: '调入射角和介质折射率，观察反射定律、折射定律和全反射',
    params: [
      { key: 'angle', label: '入射角/°', type: 'number', min: 0, max: 80, step: 1, defaultValue: 35 },
      { key: 'n1', label: '上方介质 n1', type: 'number', min: 1, max: 2.4, step: 0.1, defaultValue: 1 },
      { key: 'n2', label: '下方介质 n2', type: 'number', min: 1, max: 2.4, step: 0.1, defaultValue: 1.5 },
    ],
    buildHTML: (params, rootId) => {
      const angle = num(params, 'angle', 35)
      const n1 = num(params, 'n1', 1)
      const n2 = num(params, 'n2', 1.5)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🌈 反射与折射：n₁sinθ₁ = n₂sinθ₂</div>
    <div class="pl-note">调节角度与折射率，观察全反射条件</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row">
        <div class="pl-label"><span>入射角 θ₁</span><span class="pl-value" data-a-val></span></div>
        <input data-a type="range" min="0" max="80" step="1" value="${n(angle)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>n₁</span><span class="pl-value" data-n1-val></span></div>
        <input data-n1 type="range" min="1" max="2.4" step="0.1" value="${n(n1)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>n₂</span><span class="pl-value" data-n2-val></span></div>
        <input data-n2 type="range" min="1" max="2.4" step="0.1" value="${n(n2)}">
      </div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="207" fill="#F8FAFC"/>
        <rect y="207" width="680" height="207" fill="#E0F2FE"/>
        <line x1="0" y1="207" x2="680" y2="207" stroke="#0284C7" stroke-width="4"/>
        <line x1="340" y1="30" x2="340" y2="384" stroke="#64748B" stroke-width="2" stroke-dasharray="8 8"/>
        <text x="356" y="58" font-size="16" font-weight="800" fill="#64748B">法线</text>
        <g data-rays></g>
        <text x="28" y="54" data-n1-text font-size="20" font-weight="900" fill="#334155"></text>
        <text x="28" y="370" data-n2-text font-size="20" font-weight="900" fill="#0369A1"></text>
        <text x="420" y="360" data-law font-size="20" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var a=root.querySelector('[data-a]'), n1=root.querySelector('[data-n1]'), n2=root.querySelector('[data-n2]');
    var av=root.querySelector('[data-a-val]'), n1v=root.querySelector('[data-n1-val]'), n2v=root.querySelector('[data-n2-val]');
    var res=root.querySelector('[data-result]'), rays=root.querySelector('[data-rays]');
    var n1t=root.querySelector('[data-n1-text]'), n2t=root.querySelector('[data-n2-text]'), law=root.querySelector('[data-law]');
    function ray(x1,y1,x2,y2,color,dash,label){
      var ang=Math.atan2(y2-y1,x2-x1);
      var ax=x2-18*Math.cos(ang), ay=y2-18*Math.sin(ang);
      var left=ang+2.55, right=ang-2.55;
      return '<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2+'" stroke="'+color+'" stroke-width="4" stroke-linecap="round" '+(dash?'stroke-dasharray="8 6"':'')+'/><polygon points="'+x2+','+y2+' '+(ax+9*Math.cos(left))+','+(ay+9*Math.sin(left))+' '+(ax+9*Math.cos(right))+','+(ay+9*Math.sin(right))+'" fill="'+color+'"/><text x="'+((x1+x2)/2+8)+'" y="'+((y1+y2)/2-8)+'" font-size="15" font-weight="800" fill="'+color+'">'+label+'</text>';
    }
    function update(){
      var A=Number(a.value), N1=Number(n1.value), N2=Number(n2.value);
      var rad=A*Math.PI/180;
      av.textContent=A.toFixed(0)+'°'; n1v.textContent=N1.toFixed(1); n2v.textContent=N2.toFixed(1);
      n1t.textContent='介质1 n₁='+N1.toFixed(1); n2t.textContent='介质2 n₂='+N2.toFixed(1);
      var cx=340, cy=207, L=185;
      var ix=cx-L*Math.sin(rad), iy=cy-L*Math.cos(rad);
      var rx=cx+L*Math.sin(rad), ry=cy-L*Math.cos(rad);
      var s=N1*Math.sin(rad)/N2;
      var html=ray(ix,iy,cx,cy,'#DC2626',false,'入射光')+ray(cx,cy,rx,ry,'#F59E0B',false,'反射光');
      if(Math.abs(s)>1){
        html += '<path d="M250 207 A90 90 0 0 1 430 207" fill="none" stroke="#EF4444" stroke-width="5" stroke-dasharray="10 6"/>';
        res.innerHTML='发生全反射：当 n₁ > n₂ 且入射角超过临界角时，没有折射光进入下方介质。<br>临界角 θc = '+(Math.asin(N2/N1)*180/Math.PI).toFixed(1)+'°';
        law.textContent='全反射';
      }else{
        var br=Math.asin(s);
        var tx=cx+L*Math.sin(br), ty=cy+L*Math.cos(br);
        html += ray(cx,cy,tx,ty,'#0284C7',false,'折射光');
        res.innerHTML='反射角 = 入射角 = '+A.toFixed(0)+'°。<br>折射角 θ₂ = '+(br*180/Math.PI).toFixed(1)+'°，满足 n₁sinθ₁ = n₂sinθ₂。';
        law.textContent='θ₂='+((br*180/Math.PI).toFixed(1))+'°';
      }
      rays.innerHTML=html;
    }
    a.oninput=update;n1.oninput=update;n2.oninput=update;update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-wave',
    group: '🌊 波动与声学',
    name: '机械波参数',
    emoji: '🌊',
    desc: '调节振幅、波长和波速，观察频率、传播方向和波形变化',
    params: [
      { key: 'amp', label: '振幅 A', type: 'number', min: 10, max: 80, step: 5, defaultValue: 45 },
      { key: 'lambda', label: '波长 λ', type: 'number', min: 60, max: 220, step: 10, defaultValue: 140 },
      { key: 'speed', label: '波速 v', type: 'number', min: 20, max: 180, step: 10, defaultValue: 80 },
    ],
    buildHTML: (params, rootId) => {
      const amp = num(params, 'amp', 45)
      const lambda = num(params, 'lambda', 140)
      const speed = num(params, 'speed', 80)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🌊 机械波：v = λf</div>
    <div class="pl-note">波速固定时，波长越短频率越高</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row">
        <div class="pl-label"><span>振幅 A</span><span class="pl-value" data-amp-val></span></div>
        <input data-amp type="range" min="10" max="80" step="5" value="${n(amp)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>波长 λ</span><span class="pl-value" data-lam-val></span></div>
        <input data-lam type="range" min="60" max="220" step="10" value="${n(lambda)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>波速 v</span><span class="pl-value" data-speed-val></span></div>
        <input data-speed type="range" min="20" max="180" step="10" value="${n(speed)}">
      </div>
      <div style="display:flex;gap:8px;margin-bottom:12px;">
        <button data-toggle>⏸ 暂停</button>
      </div>
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
    var amp=root.querySelector('[data-amp]'), lam=root.querySelector('[data-lam]'), speed=root.querySelector('[data-speed]');
    var av=root.querySelector('[data-amp-val]'), lv=root.querySelector('[data-lam-val]'), sv=root.querySelector('[data-speed-val]');
    var res=root.querySelector('[data-result]'), btn=root.querySelector('[data-toggle]');
    var running=true, t=0, last=0;
    btn.onclick=function(){running=!running;btn.textContent=running?'⏸ 暂停':'▶ 播放';};
    function draw(ts){
      if(!last)last=ts;
      var dt=(ts-last)/1000; last=ts;
      if(running)t+=dt;
      var A=Number(amp.value), L=Number(lam.value), V=Number(speed.value), F=V/L;
      av.textContent=A.toFixed(0);lv.textContent=L.toFixed(0)+' px';sv.textContent=V.toFixed(0)+' px/s';
      res.innerHTML='频率 f = v / λ = '+F.toFixed(2)+' Hz。<br>振幅决定“上下摆动幅度”，波长决定“疏密”，波速决定传播快慢。';
      ctx.clearRect(0,0,680,414);
      ctx.fillStyle='#FFFFFF';ctx.fillRect(0,0,680,414);
      ctx.strokeStyle='#E5E7EB';ctx.lineWidth=1;
      for(var gx=0;gx<680;gx+=40){ctx.beginPath();ctx.moveTo(gx,0);ctx.lineTo(gx,414);ctx.stroke();}
      for(var gy=0;gy<414;gy+=40){ctx.beginPath();ctx.moveTo(0,gy);ctx.lineTo(680,gy);ctx.stroke();}
      ctx.strokeStyle='#94A3B8';ctx.lineWidth=2;ctx.beginPath();ctx.moveTo(30,207);ctx.lineTo(650,207);ctx.stroke();
      ctx.strokeStyle='#0284C7';ctx.lineWidth=5;ctx.beginPath();
      for(var x=30;x<=650;x+=3){
        var y=207 + A*Math.sin(2*Math.PI*((x-30)/L - F*t));
        if(x===30)ctx.moveTo(x,y);else ctx.lineTo(x,y);
      }
      ctx.stroke();
      ctx.fillStyle='#DC2626';
      for(var p=0;p<4;p++){
        var px=80+p*L;
        if(px<640){ctx.beginPath();ctx.arc(px,207,4,0,Math.PI*2);ctx.fill();}
      }
      ctx.fillStyle='#075985';ctx.font='bold 20px sans-serif';
      ctx.fillText('v = λf = '+V.toFixed(0)+' px/s',38,52);
      ctx.fillStyle='#64748B';ctx.font='14px sans-serif';
      ctx.fillText('红点间距表示一个波长 λ',38,80);
      requestAnimationFrame(draw);
    }
    requestAnimationFrame(draw);
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'lab-induction',
    group: '🧲 电磁实验',
    name: '电磁感应',
    emoji: '🧲',
    desc: '磁体穿过线圈，观察感应电流大小与方向随磁通量变化而改变',
    params: [
      { key: 'turns', label: '线圈匝数 N', type: 'number', min: 5, max: 40, step: 1, defaultValue: 18 },
      { key: 'b', label: '磁场强度 B', type: 'number', min: 1, max: 10, step: 1, defaultValue: 5 },
      { key: 'speed', label: '磁体速度', type: 'number', min: 20, max: 180, step: 10, defaultValue: 80 },
    ],
    buildHTML: (params, rootId) => {
      const turns = num(params, 'turns', 18)
      const b = num(params, 'b', 5)
      const speed = num(params, 'speed', 80)
      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="pl-head">
    <div class="pl-title">🧲 电磁感应：磁通量变化产生感应电流</div>
    <div class="pl-note">速度、匝数、磁场越大，感应越明显</div>
  </div>
  <div class="pl-body">
    <div class="pl-controls">
      <div class="pl-row">
        <div class="pl-label"><span>线圈匝数 N</span><span class="pl-value" data-turns-val></span></div>
        <input data-turns type="range" min="5" max="40" step="1" value="${n(turns)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>磁场强度 B</span><span class="pl-value" data-b-val></span></div>
        <input data-b type="range" min="1" max="10" step="1" value="${n(b)}">
      </div>
      <div class="pl-row">
        <div class="pl-label"><span>磁体速度</span><span class="pl-value" data-speed-val></span></div>
        <input data-speed type="range" min="20" max="180" step="10" value="${n(speed)}">
      </div>
      <div style="display:flex;gap:8px;margin-bottom:12px;">
        <button data-toggle>⏸ 暂停</button>
        <button data-reset style="background:#E2E8F0;color:#334155;">↺ 重置</button>
      </div>
      <div class="pl-result" data-result></div>
    </div>
    <div class="pl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#fff"/>
        <g data-coil></g>
        <g data-magnet></g>
        <circle cx="340" cy="328" r="48" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="4"/>
        <path data-needle d="M340 328 L340 290" stroke="#DC2626" stroke-width="5" stroke-linecap="round"/>
        <text x="296" y="392" font-size="16" font-weight="800" fill="#475569">灵敏电流计</text>
        <text x="48" y="54" data-main font-size="20" font-weight="900" fill="#075985"></text>
      </svg>
    </div>
  </div>
  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;
    var turns=root.querySelector('[data-turns]'), b=root.querySelector('[data-b]'), speed=root.querySelector('[data-speed]');
    var tv=root.querySelector('[data-turns-val]'), bv=root.querySelector('[data-b-val]'), sv=root.querySelector('[data-speed-val]');
    var coil=root.querySelector('[data-coil]'), magnet=root.querySelector('[data-magnet]'), needle=root.querySelector('[data-needle]');
    var main=root.querySelector('[data-main]'), res=root.querySelector('[data-result]');
    var btn=root.querySelector('[data-toggle]'), reset=root.querySelector('[data-reset]');
    var running=true, x=70, dir=1, last=0;
    btn.onclick=function(){running=!running;btn.textContent=running?'⏸ 暂停':'▶ 播放';};
    reset.onclick=function(){x=70;dir=1;};
    function update(ts){
      if(!last)last=ts;
      var dt=(ts-last)/1000;last=ts;
      var N=Number(turns.value), B=Number(b.value), V=Number(speed.value);
      if(running){x += dir*V*dt;if(x>590){x=590;dir=-1;}if(x<70){x=70;dir=1;}}
      tv.textContent=N.toFixed(0)+' 匝';bv.textContent=B.toFixed(0);sv.textContent=V.toFixed(0);
      var coilX=340, dist=(x-coilX)/160;
      var flux=Math.exp(-dist*dist);
      var induced=dir*N*B*V/720*flux;
      var angle=Math.max(-60,Math.min(60,induced*42));
      needle.setAttribute('transform','rotate('+angle+' 340 328)');
      var c='';
      for(var i=0;i<N;i++){
        var xx=270+(i%18)*4;
        var yy=120+Math.floor(i/18)*9;
        c+='<ellipse cx="'+xx+'" cy="'+(yy+50)+'" rx="32" ry="92" fill="none" stroke="#0284C7" stroke-width="2" opacity="0.55"/>';
      }
      coil.innerHTML='<rect x="255" y="100" width="170" height="160" rx="24" fill="#E0F2FE" stroke="#0284C7" stroke-width="4" opacity="0.45"/>'+c+'<text x="284" y="92" font-size="18" font-weight="900" fill="#0284C7">线圈</text>';
      magnet.innerHTML='<rect x="'+(x-42)+'" y="164" width="84" height="52" rx="10" fill="#FEE2E2" stroke="#EF4444" stroke-width="3"/><rect x="'+(x-42)+'" y="164" width="42" height="52" rx="10" fill="#EF4444"/><text x="'+(x-31)+'" y="197" font-size="20" font-weight="900" fill="#fff">N</text><text x="'+(x+12)+'" y="197" font-size="20" font-weight="900" fill="#991B1B">S</text>';
      main.textContent='感应强度 ≈ '+Math.abs(induced).toFixed(2)+'，方向：'+(induced>0?'右偏':induced<0?'左偏':'无');
      res.innerHTML='感应电流取决于磁通量变化快慢：磁体越快、磁场越强、线圈匝数越多，电流计偏转越明显。<br>磁体反向运动时，电流方向也随之反向。';
      requestAnimationFrame(update);
    }
    requestAnimationFrame(update);
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
export function getPhysicsLabGroups(): { group: string; items: PhysicsLabTemplate[] }[] {
  const groups: { group: string; items: PhysicsLabTemplate[] }[] = []
  for (const t of PHYSICS_LAB_TEMPLATES) {
    let g = groups.find(x => x.group === t.group)
    if (!g) { g = { group: t.group, items: [] }; groups.push(g) }
    g.items.push(t)
  }
  return groups
}

/** 按ID查模板 */
export function findPhysicsLabTemplate(id: string): PhysicsLabTemplate | undefined {
  return PHYSICS_LAB_TEMPLATES.find(t => t.id === id)
}
