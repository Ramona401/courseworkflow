/**
 * lifeScienceLabTemplatesInheritanceMendelDihybrid.ts
 *
 * 平面生命科学实验室：孟德尔两对相对性状。
 *
 * 教学目标：
 * 1. 理解两对等位基因在形成配子时分别发生分离；
 * 2. 理解位于非同源染色体上的两对基因可以独立分配；
 * 3. 比较AABB×aabb、AaBb×AaBb和AaBb×aabb三种典型组合；
 * 4. 使用4×4棋盘格分析两对相对性状的子代组合；
 * 5. 理解完全显性和独立分配条件下F2常见的9:3:3:1表现型比例；
 * 6. 理解双杂合子测交常见的1:1:1:1表现型比例；
 * 7. 比较理论概率与有限样本模拟结果，认识随机波动。
 *
 * 教学边界：
 * 1. 本模型使用A/a表示第一对等位基因，B/b表示第二对等位基因；
 * 2. A和B分别为对应性状的显性等位基因；
 * 3. 默认采用完全显性模型；
 * 4. 默认假设两对基因位于非同源染色体上，形成配子时独立分配；
 * 5. AaBb可形成AB、Ab、aB、ab四类等机会配子；
 * 6. 9:3:3:1和1:1:1:1比例需要满足相关遗传假设和足够大的样本量；
 * 7. 若两基因连锁、存在互作、致死或环境影响，实际结果可能不同；
 * 8. 随机模拟仅用于课堂概率比较，不代表真实育种试验数据。
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
 */
function dihybridStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C7D2FE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0E7FF,#ECFDF5);border-bottom:1px solid #C7D2FE}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:244px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#FAFAFF;border-right:1px solid #C7D2FE}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#4F46E5;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#4F46E5}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .bl-buttons{display:grid;grid-template-columns:1fr;gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-button{height:31px;padding:0 6px;border:1px solid #A5B4FC;border-radius:8px;background:#fff;color:#3730A3;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#4F46E5;background:#E0E7FF;box-shadow:0 3px 9px rgba(79,70,229,.14)}'
    + '#' + rootId + ' .bl-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#818CF8,#4F46E5);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-auto.paused{background:#64748B}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #C7D2FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:13px;color:#4338CA;min-height:20px}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#E0E7FF;color:#312E81;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .dh-gamete{animation:' + rootId + '-gamete var(--dh-speed,1.6s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .dh-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--dh-flow-speed,1.5s) linear infinite}'
    + '@keyframes ' + rootId + '-gamete{from{transform:translateY(4px);opacity:.55}to{transform:translateY(-5px);opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_MENDEL_DIHYBRID:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-mendel-dihybrid',
    group: '🧬 遗传规律',
    name: '孟德尔两对相对性状',
    emoji: '🌾',
    desc: '比较双纯合杂交、双杂合自交和双杂合测交，观察独立分配、4×4棋盘格与9:3:3:1比例',
    params: [
      {
        key: 'offspringCount',
        label: '模拟子代数量',
        type: 'number',
        min: 40,
        max: 800,
        step: 20,
        defaultValue: 320,
        hint: '样本量越大，模拟比例通常越接近理论概率',
      },
      {
        key: 'randomSeed',
        label: '随机实验编号',
        type: 'number',
        min: 1,
        max: 99,
        step: 1,
        defaultValue: 23,
        hint: '改变编号可获得另一组可重复的模拟结果',
      },
      {
        key: 'animationSpeed',
        label: '自动演示速度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 56,
      },
    ],

    buildHTML: (params, rootId) => {
      const offspringCount = num(
        params,
        'offspringCount',
        320,
      )
      const randomSeed = num(
        params,
        'randomSeed',
        23,
      )
      const animationSpeed = num(
        params,
        'animationSpeed',
        56,
      )

      return `
<div id="${rootId}">
${dihybridStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌾 孟德尔两对相对性状</div>
    <div class="bl-note">完全显性、独立分配教学模型</div>
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
          min="40"
          max="800"
          step="20"
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
        >双纯合亲本：AABB × aabb</button>

        <button
          type="button"
          class="bl-button"
          data-cross="self"
        >F₁自交：AaBb × AaBb</button>

        <button
          type="button"
          class="bl-button"
          data-cross="test"
        >双杂合测交：AaBb × aabb</button>
      </div>

      <button
        type="button"
        class="bl-auto"
        data-auto
      >自动演示：运行中</button>

      <div class="bl-status">
        <div class="bl-card">
          <b data-theory-ratio></b>
          <span>理论表现型比例</span>
        </div>

        <div class="bl-card">
          <b data-simulated-ratio></b>
          <span>模拟表现型比例</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="孟德尔两对相对性状杂交互动实验"
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
            <path d="M0,0 L0,6 L8,3 z" fill="#4F46E5"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#312E81"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="22"
          y="33"
          data-title
          font-size="23"
          font-weight="900"
          fill="#3730A3"
        ></text>

        <text
          x="22"
          y="60"
          data-summary
          font-size="13"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-parent-layer></g>
        <g data-gamete-layer></g>
        <g data-punnett-layer></g>
        <g data-stat-layer></g>

        <g transform="translate(20 392)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="22" y="12" font-size="11" font-weight="800" fill="#475569">
            A-：第一性状显性
          </text>
        </g>

        <g transform="translate(178 392)">
          <circle cx="7" cy="7" r="7" fill="#EC4899"/>
          <text x="22" y="12" font-size="11" font-weight="800" fill="#475569">
            aa：第一性状隐性
          </text>
        </g>

        <g transform="translate(340 392)">
          <rect x="0" y="0" width="14" height="14" rx="3" fill="#10B981"/>
          <text x="22" y="12" font-size="11" font-weight="800" fill="#475569">
            B-：第二性状显性
          </text>
        </g>

        <g transform="translate(504 392)">
          <rect x="0" y="0" width="14" height="14" rx="3" fill="#F59E0B"/>
          <text x="22" y="12" font-size="11" font-weight="800" fill="#475569">
            bb：第二性状隐性
          </text>
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
    var theoryRatio=root.querySelector('[data-theory-ratio]');
    var simulatedRatio=root.querySelector(
      '[data-simulated-ratio]'
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
        parent1:'AABB',
        parent2:'aabb',
        title:'双纯合显性亲本与双纯合隐性亲本杂交',
        summary:'亲本分别只产生AB和ab配子，F₁全部为AaBb',
        stage:'P代杂交 → F₁',
        teaching:'双纯合亲本杂交形成的F₁均为AaBb，并同时表现两种显性性状。'
      },
      self:{
        parent1:'AaBb',
        parent2:'AaBb',
        title:'F₁双杂合子自交',
        summary:'AaBb分别产生AB、Ab、aB、ab四类等机会配子',
        stage:'F₁自交 → F₂',
        teaching:'在完全显性和独立分配条件下，F₂四类表现型理论比例为9:3:3:1。'
      },
      test:{
        parent1:'AaBb',
        parent2:'aabb',
        title:'双杂合显性个体与双隐性纯合个体测交',
        summary:'双杂合亲本产生四类配子，双隐性亲本只产生ab配子',
        stage:'双杂合测交',
        teaching:'两对基因独立分配时，双杂合子测交的四类表现型理论上各占四分之一。'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    /**
     * 返回四个等机会配子槽位。
     *
     * 重复槽位保留，是为了让所有棋盘格统一使用4×4结构：
     * AABB → AB、AB、AB、AB；
     * aabb → ab、ab、ab、ab；
     * AaBb → AB、Ab、aB、ab。
     */
    function gametesFor(genotype){
      if(genotype==='AABB'){
        return ['AB','AB','AB','AB'];
      }

      if(genotype==='aabb'){
        return ['ab','ab','ab','ab'];
      }

      return ['AB','Ab','aB','ab'];
    }

    function normalizeLocus(a1,a2,upper){
      var lower=upper.toLowerCase();

      if(a1===upper && a2===upper){
        return upper+upper;
      }

      if(a1===lower && a2===lower){
        return lower+lower;
      }

      return upper+lower;
    }

    /**
     * 将两个配子组合成标准顺序的双基因型。
     */
    function normalizeDihybrid(gamete1,gamete2){
      var locusA=normalizeLocus(
        gamete1.charAt(0),
        gamete2.charAt(0),
        'A'
      );

      var locusB=normalizeLocus(
        gamete1.charAt(1),
        gamete2.charAt(1),
        'B'
      );

      return locusA+locusB;
    }

    /**
     * 将基因型归为四类表现型。
     *
     * bothDominant：A-B-
     * firstDominant：A-bb
     * secondDominant：aaB-
     * bothRecessive：aabb
     */
    function phenotypeClass(genotype){
      var firstDominant=
        genotype.charAt(0)==='A'
        ||genotype.charAt(1)==='A';

      var secondDominant=
        genotype.charAt(2)==='B'
        ||genotype.charAt(3)==='B';

      if(firstDominant && secondDominant){
        return 'bothDominant';
      }

      if(firstDominant){
        return 'firstDominant';
      }

      if(secondDominant){
        return 'secondDominant';
      }

      return 'bothRecessive';
    }

    function createEmptyPhenotypes(){
      return {
        bothDominant:0,
        firstDominant:0,
        secondDominant:0,
        bothRecessive:0
      };
    }

    function createRandom(seed){
      var state=(
        Math.floor(seed)*2246822519
        +Math.floor(Number(countInput.value))*131
        +crossOrder.indexOf(cross)*104729
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
     * 根据16个等机会棋盘格单元计算理论表现型计数。
     */
    function calculateTheory(parent1,parent2){
      var g1=gametesFor(parent1);
      var g2=gametesFor(parent2);
      var phenotypeCounts=createEmptyPhenotypes();
      var genotypeCounts={};

      for(var row=0;row<4;row++){
        for(var col=0;col<4;col++){
          var genotype=normalizeDihybrid(
            g1[col],
            g2[row]
          );
          var phenotype=phenotypeClass(genotype);

          phenotypeCounts[phenotype]+=1;
          genotypeCounts[genotype]=
            (genotypeCounts[genotype]||0)+1;
        }
      }

      return {
        phenotypes:phenotypeCounts,
        genotypes:genotypeCounts
      };
    }

    function simulateOffspring(
      parent1,
      parent2,
      count,
      seed
    ){
      var g1=gametesFor(parent1);
      var g2=gametesFor(parent2);
      var random=createRandom(seed);
      var phenotypeCounts=createEmptyPhenotypes();
      var genotypeCounts={};

      for(var i=0;i<count;i++){
        var gamete1=g1[
          Math.floor(random()*g1.length)
        ];
        var gamete2=g2[
          Math.floor(random()*g2.length)
        ];
        var genotype=normalizeDihybrid(
          gamete1,
          gamete2
        );
        var phenotype=phenotypeClass(genotype);

        phenotypeCounts[phenotype]+=1;
        genotypeCounts[genotype]=
          (genotypeCounts[genotype]||0)+1;
      }

      return {
        phenotypes:phenotypeCounts,
        genotypes:genotypeCounts
      };
    }

    function gcd(a,b){
      var x=Math.abs(Math.round(a));
      var y=Math.abs(Math.round(b));

      while(y!==0){
        var temp=x%y;
        x=y;
        y=temp;
      }

      return x||1;
    }

    function phenotypeRatioText(counts){
      var values=[
        counts.bothDominant,
        counts.firstDominant,
        counts.secondDominant,
        counts.bothRecessive
      ];

      var divisor=0;

      for(var i=0;i<values.length;i++){
        divisor=divisor===0
          ?values[i]
          :gcd(divisor,values[i]);
      }

      divisor=divisor||1;

      return values.map(function(value){
        return Math.round(value/divisor);
      }).join(':');
    }

    function simulatedRatioText(counts){
      var base=counts.bothRecessive;

      if(base<=0){
        return counts.bothDominant
          +':'
          +counts.firstDominant
          +':'
          +counts.secondDominant
          +':'
          +counts.bothRecessive;
      }

      return [
        counts.bothDominant/base,
        counts.firstDominant/base,
        counts.secondDominant/base,
        1
      ].map(function(value){
        return value.toFixed(1);
      }).join(':');
    }

    function gameteChip(x,y,gamete,radius){
      var firstColor=gamete.charAt(0)==='A'
        ?'#2563EB'
        :'#EC4899';
      var secondColor=gamete.charAt(1)==='B'
        ?'#10B981'
        :'#F59E0B';

      return ''
        +'<circle cx="'+(x-radius*.35)+'" cy="'+y
        +'" r="'+radius+'" fill="'+firstColor
        +'" stroke="#FFFFFF" stroke-width="2"/>'
        +'<rect x="'+(x-radius*.1)+'" y="'+(y-radius)
        +'" width="'+(radius*2)+'" height="'+(radius*2)
        +'" rx="'+(radius*.45)+'" fill="'+secondColor
        +'" stroke="#FFFFFF" stroke-width="2"/>'
        +'<text x="'+(x+radius*.45)+'" y="'+(y+4)
        +'" text-anchor="middle" font-size="10"'
        +' font-weight="900" fill="#FFFFFF">'
        +gamete+'</text>';
    }

    function phenotypeSymbol(
      x,
      y,
      genotype,
      scale
    ){
      var phenotype=phenotypeClass(genotype);
      var firstDominant=
        phenotype==='bothDominant'
        ||phenotype==='firstDominant';
      var secondDominant=
        phenotype==='bothDominant'
        ||phenotype==='secondDominant';

      var firstFill=firstDominant
        ?'#2563EB'
        :'#EC4899';
      var secondFill=secondDominant
        ?'#10B981'
        :'#F59E0B';

      return ''
        +'<circle cx="'+(x-7*scale)+'" cy="'+y
        +'" r="'+(12*scale)+'" fill="'+firstFill
        +'" stroke="#FFFFFF" stroke-width="'+(2*scale)+'"/>'
        +'<rect x="'+(x+1*scale)+'" y="'+(y-12*scale)
        +'" width="'+(24*scale)+'" height="'+(24*scale)
        +'" rx="'+(5*scale)+'" fill="'+secondFill
        +'" stroke="#FFFFFF" stroke-width="'+(2*scale)+'"/>';
    }

    function parentCard(x,y,genotype,label){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="'+x+'" y="'+y+'" width="166"'
        +' height="75" rx="15" fill="#FFFFFF"'
        +' stroke="#A5B4FC" stroke-width="3"/>'
        +phenotypeSymbol(
          x+35,
          y+38,
          genotype,
          .85
        )
        +'<text x="'+(x+70)+'" y="'+(y+24)
        +'" font-size="11" font-weight="800"'
        +' fill="#64748B">'+label+'</text>'
        +'<text x="'+(x+70)+'" y="'+(y+49)
        +'" font-size="20" font-weight="900"'
        +' fill="#3730A3">'+genotype+'</text>'
        +'<text x="'+(x+70)+'" y="'+(y+65)
        +'" font-size="9.5" font-weight="800"'
        +' fill="#475569">'
        +phenotypeClass(genotype)
          .replace('bothDominant','A-B-')
          .replace('firstDominant','A-bb')
          .replace('secondDominant','aaB-')
          .replace('bothRecessive','aabb')
        +'</text>'
        +'</g>';
    }

    function renderParents(info){
      return ''
        +parentCard(
          18,
          74,
          info.parent1,
          '亲本1'
        )
        +'<text x="198" y="119" text-anchor="middle"'
        +' font-size="25" font-weight="900" fill="#4F46E5">×</text>'
        +parentCard(
          212,
          74,
          info.parent2,
          '亲本2'
        )
        +'<text x="18" y="168" font-size="11"'
        +' font-weight="900" fill="#4338CA">'
        +info.stage+'</text>';
    }

    function renderGametes(info){
      var g1=gametesFor(info.parent1);
      var g2=gametesFor(info.parent2);
      var html='';

      html+='<text x="20" y="188" font-size="10.5"'
        +' font-weight="800" fill="#64748B">'
        +'亲本1配子</text>';

      html+='<text x="205" y="188" font-size="10.5"'
        +' font-weight="800" fill="#64748B">'
        +'亲本2配子</text>';

      for(var i=0;i<4;i++){
        var x1=40+i*39;
        var x2=225+i*39;

        html+='<g class="dh-gamete">'
          +gameteChip(
            x1,
            211,
            g1[i],
            11
          )
          +'</g>';

        html+='<g class="dh-gamete">'
          +gameteChip(
            x2,
            211,
            g2[i],
            11
          )
          +'</g>';
      }

      html+='<path class="dh-flow"'
        +' d="M172 211 C189 219 199 225 211 235"'
        +' fill="none" stroke="#4F46E5"'
        +' stroke-width="3"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      return html;
    }

    function renderPunnett(info){
      var g1=gametesFor(info.parent1);
      var g2=gametesFor(info.parent2);
      var startX=24;
      var startY=239;
      var cell=43;
      var html='';

      html+='<rect x="'+startX+'" y="'+startY
        +'" width="'+(cell*5)+'" height="'+(cell*5)
        +'" rx="9" fill="#FFFFFF"'
        +' stroke="#4F46E5" stroke-width="3"/>';

      for(var line=1;line<5;line++){
        html+='<line x1="'+(startX+cell*line)
          +'" y1="'+startY+'"'
          +' x2="'+(startX+cell*line)
          +'" y2="'+(startY+cell*5)
          +'" stroke="#C7D2FE" stroke-width="1.5"/>';

        html+='<line x1="'+startX
          +'" y1="'+(startY+cell*line)
          +'" x2="'+(startX+cell*5)
          +'" y2="'+(startY+cell*line)
          +'" stroke="#C7D2FE" stroke-width="1.5"/>';
      }

      html+='<text x="'+(startX+cell/2)
        +'" y="'+(startY+cell/2+4)
        +'" text-anchor="middle" font-size="10"'
        +' font-weight="900" fill="#64748B">×</text>';

      for(var col=0;col<4;col++){
        html+=gameteChip(
          startX+cell*(col+1)+cell/2-4,
          startY+cell/2,
          g1[col],
          8
        );
      }

      for(var row=0;row<4;row++){
        html+=gameteChip(
          startX+cell/2-4,
          startY+cell*(row+1)+cell/2,
          g2[row],
          8
        );

        for(var innerCol=0;innerCol<4;innerCol++){
          var genotype=normalizeDihybrid(
            g1[innerCol],
            g2[row]
          );
          var cx=
            startX+cell*(innerCol+1)+cell/2;
          var cy=
            startY+cell*(row+1)+cell/2;

          html+=phenotypeSymbol(
            cx-3,
            cy-7,
            genotype,
            .48
          );

          html+='<text x="'+cx+'" y="'+(cy+15)
            +'" text-anchor="middle" font-size="7.5"'
            +' font-weight="900" fill="#3730A3">'
            +genotype+'</text>';
        }
      }

      return html;
    }

    function phenotypeMeta(){
      return [
        {
          key:'bothDominant',
          label:'A-B-',
          color:'#4F46E5'
        },
        {
          key:'firstDominant',
          label:'A-bb',
          color:'#2563EB'
        },
        {
          key:'secondDominant',
          label:'aaB-',
          color:'#10B981'
        },
        {
          key:'bothRecessive',
          label:'aabb',
          color:'#F59E0B'
        }
      ];
    }

    function renderStats(
      theory,
      simulated,
      count
    ){
      var meta=phenotypeMeta();
      var html='';

      html+='<g transform="translate(424 76)">'
        +'<text x="0" y="0" font-size="13"'
        +' font-weight="900" fill="#334155">'
        +'四类表现型比较</text>';

      for(var i=0;i<meta.length;i++){
        var item=meta[i];
        var y=30+i*57;
        var theoryPercent=
          theory.phenotypes[item.key]/16*100;
        var simulatedPercent=
          simulated.phenotypes[item.key]/count*100;

        html+='<text x="0" y="'+y+'" font-size="10.5"'
          +' font-weight="900" fill="'+item.color+'">'
          +item.label
          +' 理论 '
          +theoryPercent.toFixed(1)
          +'% / 模拟 '
          +simulatedPercent.toFixed(1)
          +'%</text>';

        html+='<rect x="0" y="'+(y+9)
          +'" width="220" height="12" rx="6"'
          +' fill="#E2E8F0"/>';

        html+='<rect x="0" y="'+(y+9)
          +'" width="'+(220*theoryPercent/100)
          +'" height="12" rx="6"'
          +' fill="'+item.color+'" opacity=".42"/>';

        html+='<rect x="0" y="'+(y+25)
          +'" width="220" height="12" rx="6"'
          +' fill="#E2E8F0"/>';

        html+='<rect x="0" y="'+(y+25)
          +'" width="'+(220*simulatedPercent/100)
          +'" height="12" rx="6"'
          +' fill="'+item.color+'"/>';
      }

      html+='<rect x="0" y="264" width="220" height="67"'
        +' rx="12" fill="#EEF2FF"'
        +' stroke="#A5B4FC" stroke-width="2"/>';

      html+='<text x="11" y="285" font-size="10.5"'
        +' font-weight="900" fill="#3730A3">'
        +'理论比例：'
        +phenotypeRatioText(theory.phenotypes)
        +'</text>';

      html+='<text x="11" y="306" font-size="10.5"'
        +' font-weight="900" fill="#4F46E5">'
        +'模拟计数：'
        +simulated.phenotypes.bothDominant
        +' / '
        +simulated.phenotypes.firstDominant
        +' / '
        +simulated.phenotypes.secondDominant
        +' / '
        +simulated.phenotypes.bothRecessive
        +'</text>';

      html+='<text x="11" y="325" font-size="10"'
        +' font-weight="800" fill="#475569">'
        +'样本量：'+count
        +'</text>';

      html+='</g>';

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
        4300-speed*28,
        1200,
        3800
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
        '--dh-speed',
        clamp(
          2.5-speed/60,
          .55,
          2.4
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--dh-flow-speed',
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

      theoryRatio.textContent=
        phenotypeRatioText(
          theory.phenotypes
        );

      simulatedRatio.textContent=
        simulatedRatioText(
          simulated.phenotypes
        );

      var maxDeviation=0;
      var keys=[
        'bothDominant',
        'firstDominant',
        'secondDominant',
        'bothRecessive'
      ];

      for(var keyIndex=0;keyIndex<keys.length;keyIndex++){
        var key=keys[keyIndex];
        var theoryFrequency=
          theory.phenotypes[key]/16;
        var simulatedFrequency=
          simulated.phenotypes[key]/count;
        var deviation=Math.abs(
          theoryFrequency-simulatedFrequency
        );

        maxDeviation=Math.max(
          maxDeviation,
          deviation
        );
      }

      var sampleNote='';

      if(count<=100){
        sampleNote=
          '当前样本量较小，四类表现型计数出现较明显随机波动是正常现象。';
      }else if(maxDeviation>.08){
        sampleNote=
          '本组模拟与理论概率仍有一定偏差，可改变实验编号或继续增加样本量。';
      }else{
        sampleNote=
          '当前模拟结果已较接近理论概率，但有限样本不会保证严格等于理论比例。';
      }

      var lawNote=
        '形成配子时，A与a彼此分离，B与b也彼此分离；本模型假设两对基因独立分配。';

      var crossNote=cross==='self'
        ?'AaBb自交时，四类配子随机结合形成16个等机会棋盘格单元。'
        :cross==='test'
          ?'测交把双杂合亲本产生的四类配子直接反映为四类子代表现型。'
          :'双纯合亲本分别只形成一种配子，因此F₁全部为AaBb。';

      result.innerHTML=info.teaching
        +'<br>'+lawNote
        +'<br>'+crossNote
        +' '+sampleNote
        +' 若两基因连锁或存在基因互作，实际比例可能发生改变。';
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
