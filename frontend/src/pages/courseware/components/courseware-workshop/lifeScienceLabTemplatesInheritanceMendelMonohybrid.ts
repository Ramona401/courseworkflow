/**
 * lifeScienceLabTemplatesInheritanceMendelMonohybrid.ts
 *
 * 平面生命科学实验室：孟德尔一对相对性状。
 *
 * 教学目标：
 * 1. 理解一对等位基因在形成配子时彼此分离；
 * 2. 比较AA×aa、Aa×Aa和Aa×aa三种典型杂交组合；
 * 3. 使用棋盘格分析子代基因型和表现型；
 * 4. 理解完全显性条件下F2常见的1:2:1基因型比例和3:1表现型比例；
 * 5. 理解测交可用于判断显性个体的基因型；
 * 6. 比较理论概率与有限样本模拟结果，认识随机波动。
 *
 * 教学边界：
 * 1. 本模型使用A表示显性等位基因，a表示隐性等位基因；
 * 2. 默认采用完全显性模型，Aa与AA表现为显性性状；
 * 3. 配子形成时，每个配子只获得成对等位基因中的一个；
 * 4. 理论比例是在配子结合机会相等、样本足够大等条件下得到的；
 * 5. 有限样本结果可能偏离理论比例，样本量增大时通常更接近理论值；
 * 6. 真实遗传还可能存在不完全显性、共显性、多基因遗传、致死基因和环境影响；
 * 7. 随机模拟只用于课堂比较，不代表真实育种试验数据。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式或图片；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用.bl-*公共类名，兼容生命科学实验室底部课堂控制条；
 * 5. 支持同一课件页放置多个实例。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

/**
 * 安全读取数值参数。
 */
function num(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: number,
): number {
  const value = params[key]

  return typeof value === 'number' && Number.isFinite(value)
    ? value
    : fallback
}

/**
 * 把数值转换为适合写入HTML属性的短字符串。
 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/**
 * 构建完全限定到当前rootId的样式。
 *
 * 独立预览时保持左侧控制栏；
 * 嵌入课件后由lifeScienceLabUtils.ts中的公共覆盖层
 * 调整为“上方实验主体 + 底部课堂控制条”。
 */
function monohybridStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#FDF2F8);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:244px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#7C3AED}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-buttons{display:grid;grid-template-columns:1fr;gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-button{height:31px;padding:0 6px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .bl-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-auto.paused{background:#64748B}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:14px;color:#6D28D9;min-height:20px}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .mh-gamete{animation:' + rootId + '-gamete var(--mh-speed,1.6s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .mh-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--mh-flow-speed,1.5s) linear infinite}'
    + '@keyframes ' + rootId + '-gamete{from{transform:translateY(4px);opacity:.55}to{transform:translateY(-5px);opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_MENDEL_MONOHYBRID:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-mendel-monohybrid',
    group: '🧬 遗传规律',
    name: '孟德尔一对相对性状',
    emoji: '🌱',
    desc: '比较纯合杂交、自交和测交，使用棋盘格观察基因型、表现型及有限样本随机波动',
    params: [
      {
        key: 'offspringCount',
        label: '模拟子代数量',
        type: 'number',
        min: 20,
        max: 400,
        step: 10,
        defaultValue: 160,
        hint: '样本量越大，模拟比例通常越接近理论概率',
      },
      {
        key: 'randomSeed',
        label: '随机实验编号',
        type: 'number',
        min: 1,
        max: 99,
        step: 1,
        defaultValue: 17,
        hint: '改变编号可得到另一组可重复的模拟结果',
      },
      {
        key: 'animationSpeed',
        label: '自动演示速度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
    ],

    buildHTML: (params, rootId) => {
      const offspringCount = num(
        params,
        'offspringCount',
        160,
      )
      const randomSeed = num(
        params,
        'randomSeed',
        17,
      )
      const animationSpeed = num(
        params,
        'animationSpeed',
        58,
      )

      return `
<div id="${rootId}">
${monohybridStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌱 孟德尔一对相对性状</div>
    <div class="bl-note">完全显性教学模型：A为显性等位基因，a为隐性等位基因</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>模拟子代数量</span>
          <span class="bl-value" data-count-value></span>
        </div>
        <input
          data-count
          type="range"
          min="20"
          max="400"
          step="10"
          value="${n(offspringCount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>随机实验编号</span>
          <span class="bl-value" data-seed-value></span>
        </div>
        <input
          data-seed
          type="range"
          min="1"
          max="99"
          step="1"
          value="${n(randomSeed)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>自动演示速度</span>
          <span class="bl-value" data-speed-value></span>
        </div>
        <input
          data-speed
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(animationSpeed)}"
        >
      </div>

      <div class="bl-subtitle">选择杂交组合</div>

      <div class="bl-buttons">
        <button
          type="button"
          class="bl-button active"
          data-cross="pure"
        >纯合亲本：AA × aa</button>

        <button
          type="button"
          class="bl-button"
          data-cross="self"
        >F₁自交：Aa × Aa</button>

        <button
          type="button"
          class="bl-button"
          data-cross="test"
        >测交：Aa × aa</button>
      </div>

      <button
        type="button"
        class="bl-auto"
        data-auto
      >自动演示：运行中</button>

      <div class="bl-status">
        <div class="bl-card">
          <b data-theory-phenotype></b>
          <span>理论表现型比例</span>
        </div>

        <div class="bl-card">
          <b data-simulated-phenotype></b>
          <span>模拟表现型比例</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="孟德尔一对相对性状杂交互动实验"
      >
        <defs>
          <marker
            id="${rootId}-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <linearGradient
            id="${rootId}-dominant"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#86EFAC"/>
            <stop offset="100%" stop-color="#16A34A"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-recessive"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#FBCFE8"/>
            <stop offset="100%" stop-color="#EC4899"/>
          </linearGradient>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#4C1D95"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="24"
          y="35"
          data-title
          font-size="24"
          font-weight="900"
          fill="#5B21B6"
        ></text>

        <text
          x="24"
          y="63"
          data-summary
          font-size="14"
          font-weight="800"
          fill="#475569"
        ></text>

        <!-- 亲本和配子 -->
        <g data-parent-layer></g>
        <g data-gamete-layer></g>

        <!-- 棋盘格 -->
        <g data-punnett-layer></g>

        <!-- 理论与模拟统计 -->
        <g data-stat-layer></g>

        <g transform="translate(26 388)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text
            x="23"
            y="12"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >A：显性等位基因</text>
        </g>

        <g transform="translate(188 388)">
          <circle cx="7" cy="7" r="7" fill="#EC4899"/>
          <text
            x="23"
            y="12"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >a：隐性等位基因</text>
        </g>

        <g transform="translate(350 388)">
          <circle cx="7" cy="7" r="7" fill="#16A34A"/>
          <text
            x="23"
            y="12"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >显性表现型</text>
        </g>

        <g transform="translate(508 388)">
          <circle cx="7" cy="7" r="7" fill="#EC4899"/>
          <text
            x="23"
            y="12"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >隐性表现型</text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var countInput=root.querySelector('[data-count]');
    var seedInput=root.querySelector('[data-seed]');
    var speedInput=root.querySelector('[data-speed]');

    var countValue=root.querySelector('[data-count-value]');
    var seedValue=root.querySelector('[data-seed-value]');
    var speedValue=root.querySelector('[data-speed-value]');

    var buttons=root.querySelectorAll('[data-cross]');
    var autoButton=root.querySelector('[data-auto]');
    var theoryPhenotype=root.querySelector(
      '[data-theory-phenotype]'
    );
    var simulatedPhenotype=root.querySelector(
      '[data-simulated-phenotype]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var parentLayer=root.querySelector(
      '[data-parent-layer]'
    );
    var gameteLayer=root.querySelector(
      '[data-gamete-layer]'
    );
    var punnettLayer=root.querySelector(
      '[data-punnett-layer]'
    );
    var statLayer=root.querySelector(
      '[data-stat-layer]'
    );

    var cross='pure';
    var automatic=true;
    var timer=null;
    var crossOrder=['pure','self','test'];

    var crossInfo={
      pure:{
        parent1:'AA',
        parent2:'aa',
        title:'纯合显性亲本与纯合隐性亲本杂交',
        summary:'亲本分别只产生A配子和a配子，子代全部为Aa',
        stage:'P代杂交 → F₁',
        teaching:'纯合亲本杂交产生的F₁均为杂合子，并表现显性性状。'
      },
      self:{
        parent1:'Aa',
        parent2:'Aa',
        title:'F₁杂合子自交',
        summary:'两个杂合亲本都产生A和a两类配子，等位基因发生分离',
        stage:'F₁自交 → F₂',
        teaching:'在完全显性条件下，F₂基因型理论比例为1:2:1，表现型理论比例为3:1。'
      },
      test:{
        parent1:'Aa',
        parent2:'aa',
        title:'杂合显性个体与隐性纯合个体测交',
        summary:'测交子代可用于判断待测显性个体产生的配子类型',
        stage:'测交',
        teaching:'杂合子测交时，子代理论上显性与隐性表现型各占一半。'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    /**
     * 把亲本基因型转换为两个等机会配子槽位。
     *
     * AA产生A、A；
     * Aa产生A、a；
     * aa产生a、a。
     */
    function gametesFor(genotype){
      if(genotype==='AA'){
        return ['A','A'];
      }

      if(genotype==='aa'){
        return ['a','a'];
      }

      return ['A','a'];
    }

    /**
     * 统一子代基因型的字母顺序。
     */
    function normalizeGenotype(a1,a2){
      if(a1==='A' && a2==='A'){
        return 'AA';
      }

      if(a1==='a' && a2==='a'){
        return 'aa';
      }

      return 'Aa';
    }

    /**
     * 使用可重复的线性同余伪随机数。
     *
     * 相同实验编号和样本量会得到相同模拟结果，
     * 便于课堂复现和比较。
     */
    function createRandom(seed){
      var state=(
        Math.floor(seed)*2654435761
        +Math.floor(Number(countInput.value))*97
        +crossOrder.indexOf(cross)*7919
      )>>>0;

      return function(){
        state=(
          Math.imul(state,1664525)
          +1013904223
        )>>>0;

        return state/4294967296;
      };
    }

    /**
     * 根据棋盘格的四个等机会组合计算理论概率。
     */
    function calculateTheory(parent1,parent2){
      var g1=gametesFor(parent1);
      var g2=gametesFor(parent2);
      var counts={
        AA:0,
        Aa:0,
        aa:0
      };

      for(var row=0;row<2;row++){
        for(var col=0;col<2;col++){
          var genotype=normalizeGenotype(
            g1[col],
            g2[row]
          );

          counts[genotype]+=1;
        }
      }

      return {
        genotype:counts,
        dominant:counts.AA+counts.Aa,
        recessive:counts.aa
      };
    }

    /**
     * 对有限数量子代进行确定性随机模拟。
     */
    function simulateOffspring(
      parent1,
      parent2,
      count,
      seed
    ){
      var g1=gametesFor(parent1);
      var g2=gametesFor(parent2);
      var random=createRandom(seed);
      var counts={
        AA:0,
        Aa:0,
        aa:0
      };

      for(var i=0;i<count;i++){
        var allele1=g1[
          Math.floor(random()*g1.length)
        ];
        var allele2=g2[
          Math.floor(random()*g2.length)
        ];
        var genotype=normalizeGenotype(
          allele1,
          allele2
        );

        counts[genotype]+=1;
      }

      return {
        genotype:counts,
        dominant:counts.AA+counts.Aa,
        recessive:counts.aa
      };
    }

    function alleleCircle(
      x,
      y,
      allele,
      radius
    ){
      var fill=allele==='A'
        ?'#2563EB'
        :'#EC4899';
      var stroke=allele==='A'
        ?'#1D4ED8'
        :'#BE185D';

      return ''
        +'<circle cx="'+x+'" cy="'+y+'" r="'+radius
        +'" fill="'+fill+'" stroke="'+stroke
        +'" stroke-width="3"/>'
        +'<text x="'+x+'" y="'+(y+5)
        +'" text-anchor="middle" font-size="13"'
        +' font-weight="900" fill="#FFFFFF">'
        +allele+'</text>';
    }

    function phenotypeSymbol(
      x,
      y,
      genotype,
      scale
    ){
      var dominant=genotype!=='aa';

      if(dominant){
        return ''
          +'<circle cx="'+x+'" cy="'+y+'" r="'+(19*scale)
          +'" fill="url(#${rootId}-dominant)"'
          +' stroke="#166534" stroke-width="'+(3*scale)+'"/>'
          +'<path d="M'+(x-10*scale)+' '+(y-2*scale)
          +' Q'+x+' '+(y-12*scale)+' '
          +(x+10*scale)+' '+(y-2*scale)
          +'" fill="none" stroke="#DCFCE7"'
          +' stroke-width="'+(2.5*scale)+'"'
          +' stroke-linecap="round"/>';
      }

      return ''
        +'<path d="M'+(x-19*scale)+' '+y
        +' C'+(x-17*scale)+' '+(y-17*scale)+' '
        +(x-5*scale)+' '+(y-21*scale)+' '+x+' '+(y-13*scale)
        +' C'+(x+8*scale)+' '+(y-23*scale)+' '
        +(x+20*scale)+' '+(y-14*scale)+' '
        +(x+18*scale)+' '+y
        +' C'+(x+22*scale)+' '+(y+13*scale)+' '
        +(x+8*scale)+' '+(y+22*scale)+' '+x+' '+(y+14*scale)
        +' C'+(x-9*scale)+' '+(y+23*scale)+' '
        +(x-22*scale)+' '+(y+13*scale)+' '
        +(x-19*scale)+' '+y+'Z"'
        +' fill="url(#${rootId}-recessive)"'
        +' stroke="#BE185D" stroke-width="'+(3*scale)+'"/>';
    }

    function parentCard(
      x,
      y,
      genotype,
      label
    ){
      var phenotype=genotype==='aa'
        ?'隐性性状'
        :'显性性状';

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="'+x+'" y="'+y+'" width="164"'
        +' height="84" rx="16" fill="#FFFFFF"'
        +' stroke="#C4B5FD" stroke-width="3"/>'
        +phenotypeSymbol(
          x+37,
          y+42,
          genotype,
          .72
        )
        +'<text x="'+(x+70)+'" y="'+(y+27)
        +'" font-size="12" font-weight="800"'
        +' fill="#64748B">'+label+'</text>'
        +'<text x="'+(x+70)+'" y="'+(y+51)
        +'" font-size="22" font-weight="900"'
        +' fill="#5B21B6">'+genotype+'</text>'
        +'<text x="'+(x+70)+'" y="'+(y+69)
        +'" font-size="11" font-weight="800"'
        +' fill="#475569">'+phenotype+'</text>'
        +'</g>';
    }

    function renderParents(info){
      return ''
        +parentCard(
          24,
          82,
          info.parent1,
          '亲本1'
        )
        +'<text x="202" y="132" text-anchor="middle"'
        +' font-size="27" font-weight="900" fill="#7C3AED">×</text>'
        +parentCard(
          216,
          82,
          info.parent2,
          '亲本2'
        )
        +'<text x="24" y="183" font-size="12"'
        +' font-weight="900" fill="#6D28D9">'
        +info.stage+'</text>';
    }

    function renderGametes(info){
      var g1=gametesFor(info.parent1);
      var g2=gametesFor(info.parent2);
      var html='';

      html+='<text x="28" y="206" font-size="11"'
        +' font-weight="800" fill="#64748B">'
        +'亲本1配子</text>';

      html+='<text x="202" y="206" font-size="11"'
        +' font-weight="800" fill="#64748B">'
        +'亲本2配子</text>';

      for(var i=0;i<2;i++){
        var x1=60+i*52;
        var x2=234+i*52;

        html+='<g class="mh-gamete">'
          +alleleCircle(
            x1,
            231,
            g1[i],
            15
          )
          +'</g>';

        html+='<g class="mh-gamete">'
          +alleleCircle(
            x2,
            231,
            g2[i],
            15
          )
          +'</g>';
      }

      html+='<path class="mh-flow"'
        +' d="M142 231 C172 231 183 253 198 271"'
        +' fill="none" stroke="#7C3AED"'
        +' stroke-width="3"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      html+='<path class="mh-flow"'
        +' d="M318 231 C300 248 292 259 280 271"'
        +' fill="none" stroke="#7C3AED"'
        +' stroke-width="3"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      return html;
    }

    function renderPunnett(info){
      var g1=gametesFor(info.parent1);
      var g2=gametesFor(info.parent2);
      var startX=78;
      var startY=273;
      var cell=58;
      var html='';

      html+='<text x="20" y="290" font-size="11"'
        +' font-weight="900" fill="#475569">'
        +'配子结合</text>';

      html+='<rect x="'+startX+'" y="'+startY
        +'" width="'+(cell*3)+'" height="'+(cell*3)
        +'" rx="10" fill="#FFFFFF"'
        +' stroke="#7C3AED" stroke-width="3"/>';

      for(var line=1;line<3;line++){
        html+='<line x1="'+(startX+cell*line)
          +'" y1="'+startY+'"'
          +' x2="'+(startX+cell*line)
          +'" y2="'+(startY+cell*3)
          +'" stroke="#C4B5FD" stroke-width="2"/>';

        html+='<line x1="'+startX
          +'" y1="'+(startY+cell*line)
          +'" x2="'+(startX+cell*3)
          +'" y2="'+(startY+cell*line)
          +'" stroke="#C4B5FD" stroke-width="2"/>';
      }

      html+='<text x="'+(startX+cell/2)
        +'" y="'+(startY+cell/2+5)
        +'" text-anchor="middle" font-size="11"'
        +' font-weight="900" fill="#64748B">×</text>';

      for(var col=0;col<2;col++){
        html+=alleleCircle(
          startX+cell*(col+1)+cell/2,
          startY+cell/2,
          g1[col],
          13
        );
      }

      for(var row=0;row<2;row++){
        html+=alleleCircle(
          startX+cell/2,
          startY+cell*(row+1)+cell/2,
          g2[row],
          13
        );

        for(var innerCol=0;innerCol<2;innerCol++){
          var genotype=normalizeGenotype(
            g1[innerCol],
            g2[row]
          );
          var cx=startX+cell*(innerCol+1)+cell/2;
          var cy=startY+cell*(row+1)+cell/2;

          html+=phenotypeSymbol(
            cx-13,
            cy,
            genotype,
            .42
          );

          html+='<text x="'+(cx+17)+'" y="'+(cy+5)
            +'" text-anchor="middle" font-size="13"'
            +' font-weight="900" fill="'
            +(genotype==='aa'?'#BE185D':'#166534')
            +'">'+genotype+'</text>';
        }
      }

      return html;
    }

    function ratioText(a,b){
      if(b===0){
        return a>0?'全部显性':'0:0';
      }

      if(a===0){
        return '0:1';
      }

      var ratio=a/b;

      return ratio.toFixed(2)+':1';
    }

    function genotypeRatioText(counts){
      return counts.AA
        +':'
        +counts.Aa
        +':'
        +counts.aa;
    }

    function renderStats(
      theory,
      simulated,
      count
    ){
      var theoryDominant=
        theory.dominant/4*100;
      var theoryRecessive=
        theory.recessive/4*100;
      var simulatedDominant=
        simulated.dominant/count*100;
      var simulatedRecessive=
        simulated.recessive/count*100;

      var html='';

      html+='<g transform="translate(416 82)">'
        +'<text x="0" y="0" font-size="14"'
        +' font-weight="900" fill="#334155">'
        +'理论与模拟比较</text>'

        +'<text x="0" y="31" font-size="11"'
        +' font-weight="800" fill="#64748B">'
        +'理论显性 '+theoryDominant.toFixed(0)+'%</text>'
        +'<rect x="0" y="40" width="220" height="15"'
        +' rx="7.5" fill="#E2E8F0"/>'
        +'<rect x="0" y="40" width="'
        +(220*theoryDominant/100)
        +'" height="15" rx="7.5" fill="#16A34A"/>'

        +'<text x="0" y="78" font-size="11"'
        +' font-weight="800" fill="#64748B">'
        +'模拟显性 '+simulatedDominant.toFixed(1)+'%</text>'
        +'<rect x="0" y="87" width="220" height="15"'
        +' rx="7.5" fill="#E2E8F0"/>'
        +'<rect x="0" y="87" width="'
        +(220*simulatedDominant/100)
        +'" height="15" rx="7.5" fill="#22C55E"/>'

        +'<text x="0" y="125" font-size="11"'
        +' font-weight="800" fill="#64748B">'
        +'理论隐性 '+theoryRecessive.toFixed(0)+'%</text>'
        +'<rect x="0" y="134" width="220" height="15"'
        +' rx="7.5" fill="#E2E8F0"/>'
        +'<rect x="0" y="134" width="'
        +(220*theoryRecessive/100)
        +'" height="15" rx="7.5" fill="#EC4899"/>'

        +'<text x="0" y="172" font-size="11"'
        +' font-weight="800" fill="#64748B">'
        +'模拟隐性 '+simulatedRecessive.toFixed(1)+'%</text>'
        +'<rect x="0" y="181" width="220" height="15"'
        +' rx="7.5" fill="#E2E8F0"/>'
        +'<rect x="0" y="181" width="'
        +(220*simulatedRecessive/100)
        +'" height="15" rx="7.5" fill="#F472B6"/>'

        +'<rect x="0" y="218" width="220" height="79"'
        +' rx="13" fill="#F5F3FF"'
        +' stroke="#C4B5FD" stroke-width="2"/>'

        +'<text x="12" y="240" font-size="11"'
        +' font-weight="900" fill="#5B21B6">'
        +'理论基因型：'
        +genotypeRatioText(theory.genotype)
        +'</text>'

        +'<text x="12" y="262" font-size="11"'
        +' font-weight="900" fill="#7C3AED">'
        +'模拟计数：AA '
        +simulated.genotype.AA
        +'，Aa '
        +simulated.genotype.Aa
        +'，aa '
        +simulated.genotype.aa
        +'</text>'

        +'<text x="12" y="284" font-size="11"'
        +' font-weight="800" fill="#475569">'
        +'样本量：'+count+'</text>'
        +'</g>';

      return html;
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(!automatic || !document.body.contains(root)){
        return;
      }

      var speed=Number(speedInput.value);
      var interval=clamp(
        4200-speed*27,
        1200,
        3700
      );

      timer=window.setTimeout(function(){
        var index=crossOrder.indexOf(cross);
        cross=crossOrder[
          (index+1)%crossOrder.length
        ];

        seedInput.value=String(
          Number(seedInput.value)>=99
            ?1
            :Number(seedInput.value)+1
        );

        update();
        schedule();
      },interval);
    }

    function update(){
      var count=Math.round(
        Number(countInput.value)
      );
      var seed=Math.round(
        Number(seedInput.value)
      );
      var speed=Number(speedInput.value);
      var info=crossInfo[cross];

      var theory=calculateTheory(
        info.parent1,
        info.parent2
      );

      var simulated=simulateOffspring(
        info.parent1,
        info.parent2,
        count,
        seed
      );

      countValue.textContent=count.toFixed(0)+' 个';
      seedValue.textContent='第 '+seed+' 组';
      speedValue.textContent=speed.toFixed(0)+'%';

      root.style.setProperty(
        '--mh-speed',
        clamp(
          2.5-speed/60,
          .55,
          2.4
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--mh-flow-speed',
        clamp(
          2.5-speed/58,
          .5,
          2.4
        ).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-cross')===cross
        );
      }

      title.textContent=info.title;
      summary.textContent=info.summary;

      parentLayer.innerHTML=renderParents(info);
      gameteLayer.innerHTML=renderGametes(info);
      punnettLayer.innerHTML=renderPunnett(info);
      statLayer.innerHTML=renderStats(
        theory,
        simulated,
        count
      );

      theoryPhenotype.textContent=
        theory.dominant+':'+theory.recessive;

      simulatedPhenotype.textContent=
        ratioText(
          simulated.dominant,
          simulated.recessive
        );

      var deviation=Math.abs(
        simulated.dominant/count
        -theory.dominant/4
      )*100;

      var sampleNote='';

      if(count<=50){
        sampleNote=
          '当前样本量较小，模拟结果出现明显随机波动是正常现象。';
      }else if(deviation>8){
        sampleNote=
          '本组模拟与理论比例仍有一定偏差，改变实验编号或增大样本量可以继续比较。';
      }else{
        sampleNote=
          '当前模拟结果已较接近理论概率，但有限样本仍不会保证完全相等。';
      }

      var lawNote=
        '形成配子时，成对等位基因彼此分离，每个配子只获得其中一个等位基因。';

      var testNote=cross==='test'
        ?'测交后若出现隐性子代，说明待测显性亲本能够产生a配子。'
        :'配子随机结合后形成子代基因型。';

      result.innerHTML=info.teaching
        +'<br>'+lawNote
        +'<br>'+testNote
        +' '+sampleNote;
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        automatic=false;
        autoButton.textContent='自动演示：已暂停';
        autoButton.classList.add('paused');

        cross=this.getAttribute('data-cross');

        update();
        schedule();
      };
    }

    autoButton.onclick=function(){
      automatic=!automatic;

      autoButton.textContent=automatic
        ?'自动演示：运行中'
        :'自动演示：已暂停';

      autoButton.classList.toggle(
        'paused',
        !automatic
      );

      update();
      schedule();
    };

    countInput.oninput=update;
    seedInput.oninput=update;

    speedInput.oninput=function(){
      update();
      schedule();
    };

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
