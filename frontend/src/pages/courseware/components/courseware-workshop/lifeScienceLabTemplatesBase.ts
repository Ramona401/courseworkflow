/**
 * lifeScienceLabTemplates.ts — 生命科学实验室首批模板
 *
 * 首批覆盖：
 *   1. 显微镜观察
 *   2. 植物细胞结构
 *   3. 动物细胞结构
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

function num(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: number,
): number {
  const value = params[key]
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function bool(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]
  return typeof value === 'boolean' ? value : fallback
}

function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

function baseStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #D1FAE5;border-radius:16px;background:#FFFFFF;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' .bl-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#D1FAE5,#F0FDF4);border-bottom:1px solid #D1FAE5;box-sizing:border-box;}\n'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#065F46;}\n'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:220px 1fr;min-height:0;}\n'
    + '#' + rootId + ' .bl-controls{padding:14px;border-right:1px solid #D1FAE5;background:#F8FAFC;box-sizing:border-box;overflow:auto;}\n'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .bl-row{margin-bottom:13px;}\n'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;font-size:12px;font-weight:700;color:#334155;margin-bottom:6px;}\n'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#059669;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981;}\n'
    + '#' + rootId + ' button{border:none;border-radius:10px;padding:8px 12px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:12px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .bl-result{padding:9px 11px;border-radius:10px;background:#D1FAE5;color:#065F46;font-size:12px;line-height:1.55;font-weight:600;}\n'
    + '#' + rootId + ' svg{width:100%;height:100%;display:block;}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES: LifeScienceLabTemplate[] = [
  {
    id: 'biology-microscope-observation',
    group: '🔬 显微观察',
    name: '显微镜观察',
    emoji: '🔬',
    desc: '调节目镜、物镜、光线和焦距，理解显微镜成像与规范操作',
    params: [
      {
        key: 'eyepiece',
        label: '目镜倍数',
        type: 'number',
        min: 5,
        max: 20,
        step: 5,
        defaultValue: 10,
      },
      {
        key: 'objective',
        label: '物镜倍数',
        type: 'number',
        min: 4,
        max: 40,
        step: 4,
        defaultValue: 10,
      },
      {
        key: 'focus',
        label: '焦距调节',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 55,
      },
      {
        key: 'light',
        label: '光线强度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
    ],
    buildHTML: (params, rootId) => {
      const eyepiece = num(params, 'eyepiece', 10)
      const objective = num(params, 'objective', 10)
      const focus = num(params, 'focus', 55)
      const light = num(params, 'light', 68)

      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🔬 显微镜观察与成像</div>
    <div class="bl-note">总放大倍数 = 目镜倍数 × 物镜倍数</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label"><span>目镜倍数</span><span class="bl-value" data-e-val></span></div>
        <input data-e type="range" min="5" max="20" step="5" value="${n(eyepiece)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>物镜倍数</span><span class="bl-value" data-o-val></span></div>
        <input data-o type="range" min="4" max="40" step="4" value="${n(objective)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>焦距</span><span class="bl-value" data-f-val></span></div>
        <input data-f type="range" min="0" max="100" step="1" value="${n(focus)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>光线</span><span class="bl-value" data-l-val></span></div>
        <input data-l type="range" min="10" max="100" step="1" value="${n(light)}">
      </div>
      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>

        <g transform="translate(40 30)">
          <path d="M120 56 L188 56 L206 126 L164 142 L142 92 L98 92 Z" fill="#CBD5E1" stroke="#475569" stroke-width="4"/>
          <rect x="145" y="10" width="48" height="70" rx="12" fill="#334155"/>
          <rect x="154" y="0" width="30" height="25" rx="8" fill="#111827"/>
          <path d="M181 126 C238 158 250 260 196 304" fill="none" stroke="#334155" stroke-width="24" stroke-linecap="round"/>
          <rect x="84" y="204" width="180" height="24" rx="8" fill="#64748B"/>
          <rect x="122" y="225" width="104" height="16" rx="5" fill="#94A3B8"/>
          <circle cx="170" cy="168" r="24" fill="#475569"/>
          <circle cx="170" cy="168" r="10" fill="#CBD5E1"/>
          <path d="M88 334 Q178 294 272 334 L286 366 H70 Z" fill="#334155"/>
          <circle cx="174" cy="280" r="26" fill="#F8FAFC" stroke="#64748B" stroke-width="5"/>
          <path d="M174 306 L174 334" stroke="#64748B" stroke-width="6"/>
          <path data-light-ray d="M174 280 L174 228" stroke="#FACC15" stroke-width="10" opacity="0.7"/>
        </g>

        <circle cx="500" cy="202" r="142" fill="#F0FDF4" stroke="#059669" stroke-width="8"/>
        <circle cx="500" cy="202" r="126" data-field-bg fill="#ECFDF5"/>
        <g data-cells></g>
        <circle cx="500" cy="202" r="126" fill="none" data-blur-ring stroke="#FFFFFF" stroke-width="25" opacity="0"/>

        <text x="426" y="52" font-size="17" font-weight="900" fill="#065F46">目镜视野</text>
        <text x="424" y="374" data-total font-size="24" font-weight="900" fill="#065F46"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var e=root.querySelector('[data-e]');
    var o=root.querySelector('[data-o]');
    var f=root.querySelector('[data-f]');
    var l=root.querySelector('[data-l]');
    var ev=root.querySelector('[data-e-val]');
    var ov=root.querySelector('[data-o-val]');
    var fv=root.querySelector('[data-f-val]');
    var lv=root.querySelector('[data-l-val]');
    var cells=root.querySelector('[data-cells]');
    var bg=root.querySelector('[data-field-bg]');
    var blur=root.querySelector('[data-blur-ring]');
    var ray=root.querySelector('[data-light-ray]');
    var total=root.querySelector('[data-total]');
    var result=root.querySelector('[data-result]');

    function update(){
      var E=Number(e.value);
      var O=Number(o.value);
      var F=Number(f.value);
      var L=Number(l.value);
      var M=E*O;
      var clarity=Math.max(0,1-Math.abs(F-55)/55);

      ev.textContent=E+'×';
      ov.textContent=O+'×';
      fv.textContent=F+'%';
      lv.textContent=L+'%';
      total.textContent='总放大倍数：'+M+'×';

      bg.setAttribute('opacity',String(0.25+L/135));
      ray.setAttribute('opacity',String(0.18+L/120));
      ray.setAttribute('stroke-width',String(4+L/12));
      blur.setAttribute('opacity',String((1-clarity)*0.82));

      var count=Math.max(4,Math.round(34-M/18));
      var size=Math.min(34,10+M/35);
      var html='';

      for(var i=0;i<count;i++){
        var angle=i*2.399;
        var radius=18+(i*29)%96;
        var x=500+Math.cos(angle)*radius;
        var y=202+Math.sin(angle)*radius;
        html+='<ellipse cx="'+x+'" cy="'+y+'" rx="'+(size*1.3)+'" ry="'+size+'" fill="#A7F3D0" stroke="#059669" stroke-width="2" opacity="'+(0.38+clarity*0.58)+'"/>';
        html+='<circle cx="'+(x+size*0.25)+'" cy="'+y+'" r="'+Math.max(3,size*0.23)+'" fill="#7C3AED" opacity="'+(0.4+clarity*0.55)+'"/>';
      }

      cells.innerHTML=html;

      result.innerHTML='显微镜成倒立、放大的像。总放大倍数为目镜与物镜倍数的乘积。<br>'
        +(clarity>0.78?'当前焦距合适，图像清晰。':'当前焦距偏离最佳位置，图像较模糊。');
    }

    e.oninput=update;
    o.oninput=update;
    f.oninput=update;
    l.oninput=update;
    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'biology-plant-cell',
    group: '🧫 细胞结构',
    name: '植物细胞结构',
    emoji: '🌿',
    desc: '观察细胞壁、细胞膜、细胞核、液泡和叶绿体的位置与作用',
    params: [
      {
        key: 'zoom',
        label: '观察倍数',
        type: 'number',
        min: 60,
        max: 140,
        step: 5,
        defaultValue: 100,
      },
      {
        key: 'highlight',
        label: '结构选择',
        type: 'number',
        min: 0,
        max: 4,
        step: 1,
        defaultValue: 0,
        hint: '0=细胞壁，1=细胞膜，2=细胞核，3=液泡，4=叶绿体',
      },
      {
        key: 'labels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: (params, rootId) => {
      const zoom = num(params, 'zoom', 100)
      const highlight = num(params, 'highlight', 0)
      const labels = bool(params, 'labels', true)

      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌿 植物细胞结构</div>
    <div class="bl-note">植物细胞具有细胞壁、大液泡，绿色部分常含叶绿体</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label"><span>观察倍数</span><span class="bl-value" data-z-val></span></div>
        <input data-z type="range" min="60" max="140" step="5" value="${n(zoom)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>结构选择</span><span class="bl-value" data-h-val></span></div>
        <input data-h type="range" min="0" max="4" step="1" value="${n(highlight)}">
      </div>
      <div class="bl-row">
        <button data-label-btn>${labels ? '隐藏标注' : '显示标注'}</button>
      </div>
      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <g data-cell transform="translate(0 0) scale(1)">
          <rect x="142" y="72" width="396" height="270" rx="54" data-wall fill="#DCFCE7" stroke="#166534" stroke-width="15"/>
          <rect x="158" y="88" width="364" height="238" rx="44" data-membrane fill="#ECFDF5" stroke="#10B981" stroke-width="6"/>
          <ellipse cx="334" cy="210" rx="128" ry="78" data-vacuole fill="#DBEAFE" stroke="#60A5FA" stroke-width="5" opacity="0.78"/>
          <circle cx="230" cy="164" r="39" data-nucleus fill="#DDD6FE" stroke="#7C3AED" stroke-width="6"/>
          <circle cx="230" cy="164" r="14" fill="#8B5CF6"/>
          <g data-chloroplasts></g>
        </g>
        <g data-labels></g>
        <text x="210" y="382" data-name font-size="25" font-weight="900" fill="#065F46"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var z=root.querySelector('[data-z]');
    var h=root.querySelector('[data-h]');
    var zv=root.querySelector('[data-z-val]');
    var hv=root.querySelector('[data-h-val]');
    var cell=root.querySelector('[data-cell]');
    var wall=root.querySelector('[data-wall]');
    var membrane=root.querySelector('[data-membrane]');
    var nucleus=root.querySelector('[data-nucleus]');
    var vacuole=root.querySelector('[data-vacuole]');
    var chloroplasts=root.querySelector('[data-chloroplasts]');
    var labelGroup=root.querySelector('[data-labels]');
    var labelBtn=root.querySelector('[data-label-btn]');
    var name=root.querySelector('[data-name]');
    var result=root.querySelector('[data-result]');

    var showLabels=${labels ? 'true' : 'false'};
    var names=['细胞壁','细胞膜','细胞核','液泡','叶绿体'];
    var descriptions=[
      '细胞壁具有保护和支持作用，使植物细胞形态较稳定。',
      '细胞膜控制物质进出细胞，也具有保护作用。',
      '细胞核内含遗传物质，是细胞生命活动的重要控制中心。',
      '液泡内含细胞液，与细胞吸水和维持形态有关。',
      '叶绿体是绿色植物细胞进行光合作用的重要场所。'
    ];

    labelBtn.onclick=function(){
      showLabels=!showLabels;
      labelBtn.textContent=showLabels?'隐藏标注':'显示标注';
      update();
    };

    function update(){
      var Z=Number(z.value);
      var H=Math.round(Number(h.value));
      var scale=Z/100;

      zv.textContent=Z+'%';
      hv.textContent=names[H];
      name.textContent='当前观察：'+names[H];

      cell.setAttribute('transform','translate('+(340-340*scale)+' '+(207-207*scale)+') scale('+scale+')');

      wall.setAttribute('stroke-width',H===0?'24':'15');
      membrane.setAttribute('stroke-width',H===1?'13':'6');
      nucleus.setAttribute('stroke-width',H===2?'13':'6');
      vacuole.setAttribute('stroke-width',H===3?'12':'5');

      var cp='';
      var positions=[[430,145],[458,196],[425,260],[280,286],[196,245],[310,116]];
      for(var i=0;i<positions.length;i++){
        cp+='<ellipse cx="'+positions[i][0]+'" cy="'+positions[i][1]+'" rx="28" ry="13" fill="#22C55E" stroke="'+(H===4?'#14532D':'#15803D')+'" stroke-width="'+(H===4?'8':'4')+'" transform="rotate('+(i*27)+' '+positions[i][0]+' '+positions[i][1]+')"/>';
        cp+='<path d="M'+(positions[i][0]-17)+' '+positions[i][1]+' H'+(positions[i][0]+17)+'" stroke="#BBF7D0" stroke-width="3"/>';
      }
      chloroplasts.innerHTML=cp;

      labelGroup.innerHTML=showLabels
        ? '<path d="M142 110 L74 70" stroke="#166534" stroke-width="3"/><text x="18" y="66" font-size="16" font-weight="800" fill="#166534">细胞壁</text>'
          + '<path d="M164 286 L84 326" stroke="#059669" stroke-width="3"/><text x="18" y="348" font-size="16" font-weight="800" fill="#059669">细胞膜</text>'
          + '<path d="M230 164 L112 174" stroke="#7C3AED" stroke-width="3"/><text x="38" y="168" font-size="16" font-weight="800" fill="#7C3AED">细胞核</text>'
          + '<path d="M430 210 L580 210" stroke="#2563EB" stroke-width="3"/><text x="588" y="216" font-size="16" font-weight="800" fill="#2563EB">液泡</text>'
          + '<path d="M458 196 L580 136" stroke="#15803D" stroke-width="3"/><text x="574" y="126" font-size="16" font-weight="800" fill="#15803D">叶绿体</text>'
        : '';

      result.innerHTML=descriptions[H]+'<br>植物细胞通常还具有细胞质等共同结构。';
    }

    z.oninput=update;
    h.oninput=update;
    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },

  {
    id: 'biology-animal-cell',
    group: '🧫 细胞结构',
    name: '动物细胞结构',
    emoji: '🧬',
    desc: '观察细胞膜、细胞质、细胞核和线粒体，比较与植物细胞的差异',
    params: [
      {
        key: 'zoom',
        label: '观察倍数',
        type: 'number',
        min: 60,
        max: 140,
        step: 5,
        defaultValue: 100,
      },
      {
        key: 'highlight',
        label: '结构选择',
        type: 'number',
        min: 0,
        max: 3,
        step: 1,
        defaultValue: 0,
        hint: '0=细胞膜，1=细胞质，2=细胞核，3=线粒体',
      },
      {
        key: 'compare',
        label: '显示植物细胞对比',
        type: 'boolean',
        defaultValue: false,
      },
    ],
    buildHTML: (params, rootId) => {
      const zoom = num(params, 'zoom', 100)
      const highlight = num(params, 'highlight', 0)
      const compare = bool(params, 'compare', false)

      return `
<div id="${rootId}">
${baseStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧬 动物细胞结构</div>
    <div class="bl-note">动物细胞没有细胞壁和叶绿体，形态通常较灵活</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label"><span>观察倍数</span><span class="bl-value" data-z-val></span></div>
        <input data-z type="range" min="60" max="140" step="5" value="${n(zoom)}">
      </div>
      <div class="bl-row">
        <div class="bl-label"><span>结构选择</span><span class="bl-value" data-h-val></span></div>
        <input data-h type="range" min="0" max="3" step="1" value="${n(highlight)}">
      </div>
      <div class="bl-row">
        <button data-compare-btn>${compare ? '隐藏植物细胞对比' : '显示植物细胞对比'}</button>
      </div>
      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg viewBox="0 0 680 414">
        <rect width="680" height="414" fill="#FFFFFF"/>
        <g data-animal>
          <path data-membrane d="M118 214 C118 112 212 68 322 88 C432 58 544 132 542 228 C552 328 444 354 340 330 C232 362 114 318 118 214Z" fill="#FCE7F3" stroke="#DB2777" stroke-width="8"/>
          <path data-cytoplasm d="M134 214 C134 126 218 88 322 104 C420 78 526 144 526 226 C532 306 438 336 340 314 C238 344 130 302 134 214Z" fill="#FDF2F8" opacity="0.9"/>
          <circle cx="286" cy="204" r="58" data-nucleus fill="#DDD6FE" stroke="#7C3AED" stroke-width="7"/>
          <circle cx="286" cy="204" r="19" fill="#8B5CF6"/>
          <g data-mito></g>
        </g>
        <g data-compare></g>
        <text x="206" y="382" data-name font-size="25" font-weight="900" fill="#065F46"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var z=root.querySelector('[data-z]');
    var h=root.querySelector('[data-h]');
    var zv=root.querySelector('[data-z-val]');
    var hv=root.querySelector('[data-h-val]');
    var animal=root.querySelector('[data-animal]');
    var membrane=root.querySelector('[data-membrane]');
    var cytoplasm=root.querySelector('[data-cytoplasm]');
    var nucleus=root.querySelector('[data-nucleus]');
    var mito=root.querySelector('[data-mito]');
    var compareGroup=root.querySelector('[data-compare]');
    var compareBtn=root.querySelector('[data-compare-btn]');
    var name=root.querySelector('[data-name]');
    var result=root.querySelector('[data-result]');

    var showCompare=${compare ? 'true' : 'false'};
    var names=['细胞膜','细胞质','细胞核','线粒体'];
    var descriptions=[
      '细胞膜保护细胞，并控制物质进出。',
      '细胞质是许多生命活动进行的场所。',
      '细胞核内含遗传物质，控制细胞的生命活动。',
      '线粒体参与细胞呼吸，为生命活动释放能量。'
    ];

    compareBtn.onclick=function(){
      showCompare=!showCompare;
      compareBtn.textContent=showCompare?'隐藏植物细胞对比':'显示植物细胞对比';
      update();
    };

    function update(){
      var Z=Number(z.value);
      var H=Math.round(Number(h.value));
      var scale=Z/100;

      zv.textContent=Z+'%';
      hv.textContent=names[H];
      name.textContent='当前观察：'+names[H];

      var offsetX=showCompare?-90:0;
      animal.setAttribute('transform','translate('+(340-340*scale+offsetX)+' '+(207-207*scale)+') scale('+scale+')');

      membrane.setAttribute('stroke-width',H===0?'16':'8');
      cytoplasm.setAttribute('fill',H===1?'#FBCFE8':'#FDF2F8');
      nucleus.setAttribute('stroke-width',H===2?'14':'7');

      var positions=[[410,164,-18],[438,252,24],[206,288,-12],[366,294,16],[424,112,10]];
      var html='';
      for(var i=0;i<positions.length;i++){
        var x=positions[i][0];
        var y=positions[i][1];
        html+='<ellipse cx="'+x+'" cy="'+y+'" rx="34" ry="16" fill="#FDBA74" stroke="'+(H===3?'#C2410C':'#EA580C')+'" stroke-width="'+(H===3?'9':'4')+'" transform="rotate('+positions[i][2]+' '+x+' '+y+')"/>';
        html+='<path d="M'+(x-22)+' '+y+' q11 -10 22 0 t22 0" fill="none" stroke="#FFF7ED" stroke-width="3"/>';
      }
      mito.innerHTML=html;

      compareGroup.innerHTML=showCompare
        ? '<g transform="translate(470 244) scale(0.42)">'
          + '<rect x="0" y="0" width="350" height="260" rx="45" fill="#DCFCE7" stroke="#166534" stroke-width="15"/>'
          + '<rect x="18" y="18" width="314" height="224" rx="35" fill="#ECFDF5" stroke="#10B981" stroke-width="6"/>'
          + '<ellipse cx="190" cy="130" rx="98" ry="58" fill="#DBEAFE" stroke="#60A5FA" stroke-width="5"/>'
          + '<circle cx="90" cy="94" r="34" fill="#DDD6FE" stroke="#7C3AED" stroke-width="6"/>'
          + '<ellipse cx="260" cy="66" rx="27" ry="13" fill="#22C55E" stroke="#15803D" stroke-width="4"/>'
          + '<ellipse cx="278" cy="172" rx="27" ry="13" fill="#22C55E" stroke="#15803D" stroke-width="4"/>'
          + '</g>'
          + '<text x="500" y="374" font-size="15" font-weight="900" fill="#166534">植物细胞对比</text>'
        : '';

      result.innerHTML=descriptions[H]+'<br>'
        +(showCompare
          ? '对比可见：动物细胞没有细胞壁、叶绿体和中央大液泡。'
          : '可打开植物细胞对比，观察两类细胞的共同点和差异。');
    }

    z.oninput=update;
    h.oninput=update;
    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]

export function getLifeScienceLabGroups(): {
  group: string
  items: LifeScienceLabTemplate[]
}[] {
  const groups: { group: string; items: LifeScienceLabTemplate[] }[] = []

  for (const template of LIFE_SCIENCE_LAB_TEMPLATES) {
    let group = groups.find(item => item.group === template.group)

    if (!group) {
      group = { group: template.group, items: [] }
      groups.push(group)
    }

    group.items.push(template)
  }

  return groups
}

export function findLifeScienceLabTemplate(
  id: string,
): LifeScienceLabTemplate | undefined {
  return LIFE_SCIENCE_LAB_TEMPLATES.find(template => template.id === id)
}
