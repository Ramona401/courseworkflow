/**
 * lifeScienceLabTemplatesReproductionFlower.ts
 *
 * 平面生命科学实验室：花的结构、传粉与受精。
 *
 * 教学边界：
 * 1. 传粉是花粉到达柱头，受精是精子与雌性生殖细胞融合；
 * 2. 被子植物通常发生双受精：一个精子与卵细胞结合形成受精卵，
 *    另一个精子与中央细胞结合形成初生胚乳核；
 * 3. 两性花、雄花、雌花及各项数值均为一般化教学模型；
 * 4. 子房通常发育成果实，胚珠通常发育成种子。
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

function flowerStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FBCFE8;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#F0FDF4);border-bottom:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFF9FC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#EC4899}'
    + '#' + rootId + ' .fr-sub{margin:6px 0;font-size:11.5px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .fr-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .fr-buttons.five{grid-template-columns:repeat(5,1fr)}'
    + '#' + rootId + ' .fr-btn{min-height:29px;padding:3px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:9.6px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .fr-btn.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.13)}'
    + '#' + rootId + ' .fr-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .fr-toggle.off{background:#64748B}'
    + '#' + rootId + ' .fr-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .fr-card{padding:6px 3px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .fr-card b{display:block;font-size:14px;color:#BE185D}'
    + '#' + rootId + ' .fr-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .fr-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--fr-speed,1.5s) linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REPRODUCTION_FLOWER:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-flower-pollination-fertilization',
    group: '🌸 生殖与个体发育',
    name: '花的结构、传粉与受精',
    emoji: '🌸',
    desc: '比较花的主要结构，模拟自花或异花传粉、花粉管生长和被子植物双受精',
    params: [
      { key: 'flowerType', label: '花的结构类型', type: 'number', min: 0, max: 2, step: 1, defaultValue: 0, hint: '0=两性花，1=雄花，2=雌花' },
      { key: 'pollinationMode', label: '传粉方式', type: 'number', min: 0, max: 1, step: 1, defaultValue: 1, hint: '0=自花传粉，1=异花传粉' },
      { key: 'pollinationAgent', label: '传粉媒介', type: 'number', min: 0, max: 2, step: 1, defaultValue: 0, hint: '0=昆虫，1=风力，2=人工' },
      { key: 'pollenAmount', label: '花粉数量', type: 'number', min: 0, max: 100, step: 1, defaultValue: 72 },
      { key: 'compatibility', label: '花粉-柱头亲和性', type: 'number', min: 0, max: 100, step: 1, defaultValue: 82 },
      { key: 'elapsedTime', label: '过程时间', type: 'number', min: 0, max: 100, step: 1, defaultValue: 45 },
      { key: 'showLabels', label: '显示结构标注', type: 'boolean', defaultValue: true },
    ],

    buildHTML: (params, rootId) => {
      const flowerType = num(params, 'flowerType', 0)
      const pollinationMode = num(params, 'pollinationMode', 1)
      const pollinationAgent = num(params, 'pollinationAgent', 0)
      const pollenAmount = num(params, 'pollenAmount', 72)
      const compatibility = num(params, 'compatibility', 82)
      const elapsedTime = num(params, 'elapsedTime', 45)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${flowerStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌸 花的结构、传粉与受精</div>
    <div class="bl-note">传粉与受精是两个不同过程</div>
  </div>
  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row"><div class="bl-label"><span>花的结构类型</span><span class="bl-value" data-flower-value></span></div><input data-flower type="range" min="0" max="2" step="1" value="${n(flowerType)}"></div>
      <div class="bl-row"><div class="bl-label"><span>传粉方式</span><span class="bl-value" data-mode-value></span></div><input data-pollination-mode type="range" min="0" max="1" step="1" value="${n(pollinationMode)}"></div>
      <div class="bl-row"><div class="bl-label"><span>传粉媒介</span><span class="bl-value" data-agent-value></span></div><input data-agent type="range" min="0" max="2" step="1" value="${n(pollinationAgent)}"></div>
      <div class="bl-row"><div class="bl-label"><span>花粉数量</span><span class="bl-value" data-pollen-value></span></div><input data-pollen type="range" min="0" max="100" step="1" value="${n(pollenAmount)}"></div>
      <div class="bl-row"><div class="bl-label"><span>花粉-柱头亲和性</span><span class="bl-value" data-compatibility-value></span></div><input data-compatibility type="range" min="0" max="100" step="1" value="${n(compatibility)}"></div>
      <div class="bl-row"><div class="bl-label"><span>过程时间</span><span class="bl-value" data-time-value></span></div><input data-time type="range" min="0" max="100" step="1" value="${n(elapsedTime)}"></div>

      <div class="fr-sub">观察过程</div>
      <div class="fr-buttons">
        <button type="button" class="fr-btn active" data-mode="structure">花的结构</button>
        <button type="button" class="fr-btn" data-mode="pollination">传粉过程</button>
        <button type="button" class="fr-btn" data-mode="fertilization">受精过程</button>
      </div>

      <div class="fr-sub">重点结构</div>
      <div class="fr-buttons five">
        <button type="button" class="fr-btn" data-part="sepal">萼片</button>
        <button type="button" class="fr-btn active" data-part="petal">花瓣</button>
        <button type="button" class="fr-btn" data-part="stamen">雄蕊</button>
        <button type="button" class="fr-btn" data-part="pistil">雌蕊</button>
        <button type="button" class="fr-btn" data-part="ovule">胚珠</button>
      </div>

      <button type="button" class="fr-toggle${showLabels ? '' : ' off'}" data-label-toggle>${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>
      <button type="button" class="fr-toggle" data-auto>过程推进：运行中</button>

      <div class="fr-status">
        <div class="fr-card"><b data-pollination-score></b><span>传粉到达指数</span></div>
        <div class="fr-card"><b data-tube-progress></b><span>花粉管进度</span></div>
        <div class="fr-card"><b data-fertilization-state></b><span>受精状态</span></div>
      </div>
      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg viewBox="0 0 760 430" aria-label="花的结构、传粉与受精互动示意图">
        <defs>
          <marker id="${rootId}-arrow-pink" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L0,6 L8,3 z" fill="#DB2777"/></marker>
          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/></marker>
          <filter id="${rootId}-shadow"><feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#831843" flood-opacity=".13"/></filter>
          <linearGradient id="${rootId}-petal" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="#FBCFE8"/><stop offset="100%" stop-color="#F472B6"/></linearGradient>
        </defs>
        <rect width="760" height="430" fill="#FFFFFF"/>
        <text x="24" y="36" data-title font-size="26" font-weight="900" fill="#9D174D"></text>
        <text x="24" y="65" data-summary font-size="14" font-weight="800" fill="#475569"></text>

        <g data-source-flower visibility="hidden" transform="translate(73 173) scale(.55)">
          <path d="M0 140 V250" stroke="#15803D" stroke-width="18" stroke-linecap="round"/>
          <ellipse cx="0" cy="82" rx="45" ry="78" fill="#F9A8D4" stroke="#DB2777" stroke-width="5" transform="rotate(-25 0 82)"/>
          <ellipse cx="0" cy="82" rx="45" ry="78" fill="#FBCFE8" stroke="#DB2777" stroke-width="5" transform="rotate(25 0 82)"/>
          <path d="M-36 120 Q-55 75 -40 43 M36 120 Q55 75 40 43" fill="none" stroke="#F59E0B" stroke-width="7"/>
          <ellipse cx="-40" cy="36" rx="19" ry="11" fill="#FBBF24" stroke="#B45309" stroke-width="4"/>
          <ellipse cx="40" cy="36" rx="19" ry="11" fill="#FBBF24" stroke="#B45309" stroke-width="4"/>
          <text x="0" y="278" text-anchor="middle" font-size="24" font-weight="900" fill="#9D174D">供粉花</text>
        </g>

        <g data-main-flower filter="url(#${rootId}-shadow)">
          <path d="M350 330 V235" stroke="#15803D" stroke-width="22" stroke-linecap="round"/>
          <g data-sepals>
            <path d="M350 247 C302 268 282 312 302 338 C328 316 344 285 350 247Z" fill="#4ADE80" stroke="#15803D" stroke-width="4"/>
            <path d="M350 247 C398 268 418 312 398 338 C372 316 356 285 350 247Z" fill="#22C55E" stroke="#15803D" stroke-width="4"/>
          </g>
          <g data-petals>
            <ellipse cx="350" cy="184" rx="54" ry="103" fill="url(#${rootId}-petal)" stroke="#DB2777" stroke-width="5"/>
            <ellipse cx="350" cy="184" rx="54" ry="103" fill="#F9A8D4" stroke="#DB2777" stroke-width="5" transform="rotate(72 350 214)"/>
            <ellipse cx="350" cy="184" rx="54" ry="103" fill="#FBCFE8" stroke="#DB2777" stroke-width="5" transform="rotate(144 350 214)"/>
            <ellipse cx="350" cy="184" rx="54" ry="103" fill="#F9A8D4" stroke="#DB2777" stroke-width="5" transform="rotate(216 350 214)"/>
            <ellipse cx="350" cy="184" rx="54" ry="103" fill="#FBCFE8" stroke="#DB2777" stroke-width="5" transform="rotate(288 350 214)"/>
          </g>
          <circle cx="350" cy="220" r="71" fill="#FEF3C7" stroke="#F59E0B" stroke-width="4" opacity=".94"/>
          <g data-stamens>
            <path d="M350 225 Q292 180 300 121 M350 225 Q316 170 325 108 M350 225 Q384 170 375 108 M350 225 Q408 180 400 121" fill="none" stroke="#F59E0B" stroke-width="7" stroke-linecap="round"/>
            <ellipse cx="300" cy="113" rx="23" ry="12" fill="#FBBF24" stroke="#B45309" stroke-width="4"/>
            <ellipse cx="325" cy="100" rx="23" ry="12" fill="#FBBF24" stroke="#B45309" stroke-width="4"/>
            <ellipse cx="375" cy="100" rx="23" ry="12" fill="#FBBF24" stroke="#B45309" stroke-width="4"/>
            <ellipse cx="400" cy="113" rx="23" ry="12" fill="#FBBF24" stroke="#B45309" stroke-width="4"/>
          </g>
          <g data-pistil>
            <ellipse cx="350" cy="102" rx="31" ry="16" fill="#A3E635" stroke="#3F6212" stroke-width="5"/>
            <path d="M350 118 V242" fill="none" stroke="#65A30D" stroke-width="17" stroke-linecap="round"/>
            <ellipse cx="350" cy="277" rx="62" ry="54" fill="#86EFAC" stroke="#15803D" stroke-width="5"/>
            <g data-ovules>
              <ellipse cx="325" cy="275" rx="16" ry="23" fill="#FEF3C7" stroke="#CA8A04" stroke-width="3"/>
              <ellipse cx="350" cy="289" rx="16" ry="23" fill="#FEF3C7" stroke="#CA8A04" stroke-width="3"/>
              <ellipse cx="375" cy="275" rx="16" ry="23" fill="#FEF3C7" stroke="#CA8A04" stroke-width="3"/>
            </g>
          </g>
        </g>

        <g data-pollen-flow></g>
        <g data-pollen-tube></g>
        <g data-agent-icon></g>
        <g data-highlight></g>
        <g data-labels></g>

        <g data-zoom-panel transform="translate(520 105)" visibility="hidden">
          <rect width="214" height="218" rx="22" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>
          <text x="107" y="25" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">胚珠与双受精放大示意</text>
          <ellipse cx="107" cy="119" rx="67" ry="75" fill="#FEF3C7" stroke="#CA8A04" stroke-width="4"/>
          <ellipse cx="107" cy="124" rx="43" ry="53" fill="#FFF7ED" stroke="#FDBA74" stroke-width="3"/>
          <circle cx="107" cy="155" r="14" fill="#F9A8D4" stroke="#BE185D" stroke-width="3"/>
          <text x="132" y="159" font-size="10.5" font-weight="800" fill="#9D174D">卵细胞</text>
          <circle cx="98" cy="108" r="8" fill="#C4B5FD" stroke="#7C3AED" stroke-width="2"/>
          <circle cx="116" cy="108" r="8" fill="#C4B5FD" stroke="#7C3AED" stroke-width="2"/>
          <text x="132" y="112" font-size="10.5" font-weight="800" fill="#6D28D9">中央细胞</text>
          <path data-zoom-tube d="M107 42 V42" fill="none" stroke="#22C55E" stroke-width="8" stroke-linecap="round"/>
          <g data-sperm-cells></g>
          <text x="107" y="204" text-anchor="middle" data-zoom-note font-size="10.5" font-weight="900" fill="#475569"></text>
        </g>

        <g transform="translate(520 341)">
          <rect width="214" height="61" rx="15" fill="#FFF1F2" stroke="#FECDD3" stroke-width="2"/>
          <text x="107" y="20" text-anchor="middle" font-size="12" font-weight="900" fill="#9F1239">关键区分</text>
          <text x="107" y="38" text-anchor="middle" font-size="10.5" font-weight="800" fill="#881337">传粉：花粉到达柱头</text>
          <text x="107" y="53" text-anchor="middle" font-size="10.5" font-weight="800" fill="#881337">受精：精子与雌性细胞融合</text>
        </g>

        <text x="24" y="405" data-stage-note font-size="14" font-weight="900" fill="#9D174D"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var flowerInput=root.querySelector('[data-flower]');
    var pollinationModeInput=root.querySelector('[data-pollination-mode]');
    var agentInput=root.querySelector('[data-agent]');
    var pollenInput=root.querySelector('[data-pollen]');
    var compatibilityInput=root.querySelector('[data-compatibility]');
    var timeInput=root.querySelector('[data-time]');
    var values={
      flower:root.querySelector('[data-flower-value]'),
      mode:root.querySelector('[data-mode-value]'),
      agent:root.querySelector('[data-agent-value]'),
      pollen:root.querySelector('[data-pollen-value]'),
      compatibility:root.querySelector('[data-compatibility-value]'),
      time:root.querySelector('[data-time-value]')
    };
    var modeButtons=root.querySelectorAll('[data-mode]');
    var partButtons=root.querySelectorAll('[data-part]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');
    var scoreText=root.querySelector('[data-pollination-score]');
    var tubeText=root.querySelector('[data-tube-progress]');
    var stateText=root.querySelector('[data-fertilization-state]');
    var result=root.querySelector('[data-result]');
    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');
    var sourceFlower=root.querySelector('[data-source-flower]');
    var stamens=root.querySelector('[data-stamens]');
    var pistil=root.querySelector('[data-pistil]');
    var pollenFlow=root.querySelector('[data-pollen-flow]');
    var pollenTube=root.querySelector('[data-pollen-tube]');
    var agentIcon=root.querySelector('[data-agent-icon]');
    var highlight=root.querySelector('[data-highlight]');
    var labels=root.querySelector('[data-labels]');
    var zoomPanel=root.querySelector('[data-zoom-panel]');
    var zoomTube=root.querySelector('[data-zoom-tube]');
    var spermCells=root.querySelector('[data-sperm-cells]');
    var zoomNote=root.querySelector('[data-zoom-note]');

    var mode='structure';
    var selectedPart='petal';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;
    var flowerNames=['两性花','雄花','雌花'];
    var pollinationNames=['自花传粉','异花传粉'];
    var agentNames=['昆虫传粉','风力传粉','人工传粉'];
    var agentFactors=[.86,.62,.96];
    var partNotes={
      sepal:'萼片通常位于花的最外层，可在花开放前保护内部结构。',
      petal:'花瓣常通过颜色、气味或蜜腺等特征吸引传粉动物，风媒花的花瓣可能不明显。',
      stamen:'雄蕊通常由花药和花丝组成，花药中产生并释放花粉。',
      pistil:'雌蕊通常包括柱头、花柱和子房；柱头接受花粉，子房内含胚珠。',
      ovule:'胚珠位于子房内，受精后通常发育成种子；子房通常发育成果实。'
    };

    function clamp(v,min,max){return Math.max(min,Math.min(max,v));}
    function visible(el,on){el.setAttribute('visibility',on?'visible':'hidden');}

    function schedule(){
      if(timer){window.clearTimeout(timer);timer=null;}
      if(!automatic||!document.body.contains(root))return;
      timer=window.setTimeout(function(){
        var next=Number(timeInput.value)+2;
        timeInput.value=String(next>100?0:next);
        update();
        schedule();
      },720);
    }

    function renderLabels(hasStamens,hasPistil){
      if(!showLabels){labels.innerHTML='';return;}
      var html='<path d="M287 295 L205 331" stroke="#15803D" stroke-width="2.5"/><text x="134" y="338" font-size="13" font-weight="900" fill="#15803D">萼片</text>'
        +'<path d="M281 164 L187 118" stroke="#DB2777" stroke-width="2.5"/><text x="114" y="114" font-size="13" font-weight="900" fill="#BE185D">花瓣</text>';
      if(hasStamens)html+='<path d="M302 120 L214 83" stroke="#D97706" stroke-width="2.5"/><text x="142" y="79" font-size="13" font-weight="900" fill="#B45309">花药</text>';
      if(hasPistil)html+='<path d="M350 102 L446 80" stroke="#65A30D" stroke-width="2.5"/><text x="454" y="84" font-size="13" font-weight="900" fill="#3F6212">柱头</text>'
        +'<path d="M350 183 L456 159" stroke="#65A30D" stroke-width="2.5"/><text x="464" y="163" font-size="13" font-weight="900" fill="#3F6212">花柱</text>'
        +'<path d="M399 278 L468 278" stroke="#15803D" stroke-width="2.5"/><text x="476" y="283" font-size="13" font-weight="900" fill="#166534">子房</text>'
        +'<path d="M371 286 L457 315" stroke="#CA8A04" stroke-width="2.5"/><text x="465" y="321" font-size="13" font-weight="900" fill="#A16207">胚珠</text>';
      labels.innerHTML=html;
    }

    function renderHighlight(part,hasStamens,hasPistil){
      var html='';
      if(part==='sepal')html='<ellipse cx="350" cy="300" rx="83" ry="54" fill="none" stroke="#16A34A" stroke-width="6" stroke-dasharray="8 6"/>';
      if(part==='petal')html='<circle cx="350" cy="188" r="137" fill="none" stroke="#DB2777" stroke-width="6" stroke-dasharray="8 6"/>';
      if(part==='stamen'&&hasStamens)html='<ellipse cx="350" cy="151" rx="101" ry="78" fill="none" stroke="#F59E0B" stroke-width="6" stroke-dasharray="8 6"/>';
      if(part==='pistil'&&hasPistil)html='<path d="M318 91 Q350 68 382 91 V281 Q350 338 318 281Z" fill="none" stroke="#65A30D" stroke-width="6" stroke-dasharray="8 6"/>';
      if(part==='ovule'&&hasPistil)html='<ellipse cx="350" cy="281" rx="49" ry="37" fill="none" stroke="#CA8A04" stroke-width="6" stroke-dasharray="8 6"/>';
      highlight.innerHTML=html;
    }

    function renderAgent(agentIndex,pollinationMode){
      var startX=pollinationMode===0?400:168;
      var startY=pollinationMode===0?116:125;
      var icon=agentIndex===0?'🐝':agentIndex===1?'💨':'🖌️';
      agentIcon.innerHTML='<text x="'+(startX-26)+'" y="'+(startY+8)+'" font-size="38">'+icon+'</text>'
        +'<path class="fr-flow" d="M'+startX+' '+startY+' C'+(startX+52)+' '+(startY-36)+' 315 92 348 103" fill="none" stroke="'+(agentIndex===1?'#38BDF8':'#DB2777')+'" stroke-width="4" marker-end="url(#${rootId}-arrow-pink)"/>';
    }

    function renderPollen(pollinationMode,pollenLevel,timeFactor,hasPistil){
      if(!hasPistil||pollenLevel<=0){pollenFlow.innerHTML='';return;}
      var count=Math.floor(2+pollenLevel/9),html='',sx=pollinationMode===0?400:118,sy=pollinationMode===0?116:166;
      for(var i=0;i<count;i++){
        var p=clamp((i+1)/(count+1)*timeFactor*1.45,0,1);
        var x=sx+(350-sx)*p,y=sy+(102-sy)*p+Math.sin(p*Math.PI)*-52;
        if(p>.92){x=332+(i%5)*9;y=94-Math.floor(i/5)*7;}
        html+='<circle cx="'+x.toFixed(1)+'" cy="'+y.toFixed(1)+'" r="'+(4+i%3)+'" fill="#FBBF24" stroke="#B45309" stroke-width="2"/>';
      }
      pollenFlow.innerHTML=html;
    }

    function update(){
      var flowerType=clamp(Math.round(Number(flowerInput.value)),0,2);
      var pollinationMode=clamp(Math.round(Number(pollinationModeInput.value)),0,1);
      var agentIndex=clamp(Math.round(Number(agentInput.value)),0,2);
      var pollenLevel=Number(pollenInput.value);
      var compatibility=Number(compatibilityInput.value);
      var timeLevel=Number(timeInput.value),timeFactor=timeLevel/100;
      var hasStamens=flowerType!==2,hasPistil=flowerType!==1;

      values.flower.textContent=flowerNames[flowerType];
      values.mode.textContent=pollinationNames[pollinationMode];
      values.agent.textContent=agentNames[agentIndex];
      values.pollen.textContent=pollenLevel.toFixed(0)+'%';
      values.compatibility.textContent=compatibility.toFixed(0)+'%';
      values.time.textContent=timeLevel.toFixed(0)+'%';
      for(var i=0;i<modeButtons.length;i++)modeButtons[i].classList.toggle('active',modeButtons[i].getAttribute('data-mode')===mode);
      for(var j=0;j<partButtons.length;j++)partButtons[j].classList.toggle('active',partButtons[j].getAttribute('data-part')===selectedPart);
      labelToggle.textContent=showLabels?'结构标注：显示':'结构标注：隐藏';
      labelToggle.classList.toggle('off',!showLabels);
      autoButton.textContent=automatic?'过程推进：运行中':'过程推进：已暂停';
      autoButton.classList.toggle('off',!automatic);
      visible(stamens,hasStamens);visible(pistil,hasPistil);renderLabels(hasStamens,hasPistil);

      var feasible=hasPistil&&(pollinationMode===1||flowerType===0);
      var score=feasible?pollenLevel*agentFactors[agentIndex]*clamp(timeFactor*1.55,0,1):0;
      var tube=clamp((timeFactor-.18)/.72,0,1)*clamp(score/100*compatibility/100*1.45,0,1);
      var fertilized=tube>.88&&hasPistil;
      scoreText.textContent=score.toFixed(0);
      tubeText.textContent=(tube*100).toFixed(0)+'%';
      stateText.textContent=fertilized?'双受精可发生':tube>.05?'花粉管生长':'尚未开始';
      root.style.setProperty('--fr-speed',clamp(2.4-agentFactors[agentIndex]-timeFactor,.55,2.2).toFixed(2)+'s');

      pollenFlow.innerHTML='';
      pollenTube.innerHTML='';
      agentIcon.innerHTML='';
      highlight.innerHTML='';
      sourceFlower.setAttribute('visibility','hidden');
      zoomPanel.setAttribute('visibility','hidden');

      if(mode==='structure'){
        title.textContent='花的结构：'+flowerNames[flowerType];
        summary.textContent='当前重点：'+selectedPart+'；比较雄蕊和雌蕊是否存在。';
        stageNote.textContent=partNotes[selectedPart];
        renderHighlight(selectedPart,hasStamens,hasPistil);

        var typeNote=flowerType===0
          ?'两性花同时具有雄蕊和雌蕊。'
          :flowerType===1
            ?'雄花可提供花粉，但没有柱头、子房和胚珠。'
            :'雌花可接受外来花粉，但自身不能提供花粉。';

        result.innerHTML=partNotes[selectedPart]
          +'<br>'+typeNote
          +' 实际植物的花结构具有丰富多样性。';
        return;
      }

      sourceFlower.setAttribute(
        'visibility',
        pollinationMode===1?'visible':'hidden'
      );

      if(mode==='pollination'){
        renderAgent(agentIndex,pollinationMode);
        renderPollen(
          pollinationMode,
          pollenLevel,
          timeFactor,
          hasPistil
        );

        title.textContent=pollinationNames[pollinationMode];
        summary.textContent=agentNames[agentIndex]
          +'把花粉从花药带到柱头。';
        stageNote.textContent='传粉成功并不等于已经完成受精。';

        var pNote=!hasPistil
          ?'当前为雄花，没有柱头，不能作为受粉花。'
          :pollinationMode===0&&flowerType!==0
            ?'自花传粉需要同一朵花同时具有花药和柱头。'
            :score>65
              ?'较多花粉已经到达柱头。'
              :score>10
                ?'部分花粉能够到达柱头。'
                :'到达柱头的花粉很少。';

        result.innerHTML=pNote
          +' 传粉是花粉到达柱头；花粉是否萌发还与亲和性等因素有关。';
        return;
      }

      zoomPanel.setAttribute('visibility','visible');

      if(hasPistil&&score>0){
        pollenTube.innerHTML=
          '<path class="fr-flow" d="M350 102 C354 138 344 171 350 205 C356 236 348 254 350 '
          +(102+175*tube).toFixed(1)
          +'" fill="none" stroke="#22C55E" stroke-width="8" stroke-linecap="round" marker-end="url(#${rootId}-arrow-green)"/>'
          +'<circle cx="350" cy="102" r="8" fill="#FBBF24" stroke="#B45309" stroke-width="2"/>';
      }

      zoomTube.setAttribute(
        'd',
        'M107 42 V'+(42+37*tube).toFixed(1)
      );

      var sperm='';

      if(tube>.65){
        var sp=clamp((tube-.65)/.35,0,1);

        sperm='<circle cx="'+(107-9*sp).toFixed(1)
          +'" cy="'+(80+67*sp).toFixed(1)
          +'" r="7" fill="#60A5FA" stroke="#1D4ED8" stroke-width="2"/>'
          +'<circle cx="'+(107+9*sp).toFixed(1)
          +'" cy="'+(80+28*sp).toFixed(1)
          +'" r="7" fill="#A78BFA" stroke="#6D28D9" stroke-width="2"/>';
      }

      spermCells.innerHTML=sperm;

      zoomNote.textContent=fertilized
        ?'两个精子分别参与两次融合'
        :tube>.05
          ?'花粉管正在向胚珠生长'
          :'需先有亲和花粉萌发';

      title.textContent='花粉萌发、花粉管生长与双受精';
      summary.textContent='亲和花粉萌发后，花粉管沿花柱向胚珠生长。';
      stageNote.textContent='花粉管负责运送精子；传粉与受精不能混为一谈。';

      var fNote=!hasPistil
        ?'当前为雄花，不能完成受精。'
        :pollinationMode===0&&flowerType!==0
          ?'当前结构不能完成自花传粉。'
          :score<12
            ?'到达柱头的花粉太少。'
            :compatibility<20
              ?'亲和性较低，花粉萌发受到限制。'
              :tube<.35
                ?'花粉管仍在花柱中生长。'
                :tube<.9
                  ?'花粉管逐渐接近胚珠。'
                  :'花粉管进入胚珠并释放两个精子。';

      result.innerHTML=fNote
        +' 被子植物通常一个精子与卵细胞结合形成受精卵，另一个精子与中央细胞结合形成初生胚乳核。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<partButtons.length;j++){
      partButtons[j].onclick=function(){
        selectedPart=this.getAttribute('data-part');
        mode='structure';
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    autoButton.onclick=function(){
      automatic=!automatic;
      update();
      schedule();
    };

    flowerInput.oninput=update;
    pollinationModeInput.oninput=update;
    agentInput.oninput=update;
    pollenInput.oninput=update;
    compatibilityInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
