/**
 * lifeScienceLabTemplatesEvolutionBiodiversityClassification.ts
 *
 * 平面生命科学实验室：生物多样性与分类检索。
 *
 * 教学目标：
 * 1. 理解生物多样性包括遗传多样性、物种多样性和生态系统多样性；
 * 2. 理解物种数量只是描述生物多样性的一个方面；
 * 3. 理解调查中观察到的物种数会受到取样努力和检测概率影响；
 * 4. 理解栖息地完整度会影响种群、物种及生态系统层面的多样性；
 * 5. 根据可观察或可检验特征使用二歧检索表鉴定生物；
 * 6. 比较种子植物、蕨类、真菌、鸟类、昆虫和两栖动物的典型特征；
 * 7. 理解二歧检索表是鉴定工具，不等于完整的系统发育关系。
 *
 * 教学边界：
 * 1. 本模型中的物种数和综合指数均为相对教学数据；
 * 2. 遗传、物种和生态系统三个层次彼此相关，但不能相互替代；
 * 3. 调查观察到的物种数不一定等于环境中实际存在的物种总数；
 * 4. 增加取样努力通常会提高发现物种的机会，但不会无限增加真实物种数；
 * 5. 二歧检索每一步只提供两个互斥选项；
 * 6. 分类检索依据特征进行鉴定，不代表完整展示生物的亲缘关系；
 * 7. 真菌不属于植物，本模型把蘑菇作为真菌代表；
 * 8. 分类体系可能随着形态、分子和进化证据增加而修订；
 * 9. 不使用“高等生物”“低等生物”等带有线性等级含义的表述；
 * 10. 本模型不用于真实野外物种鉴定和保护等级评估。
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
function biodiversityStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BBF7D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#E0F2FE);border-bottom:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#F8FFF9;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#16A34A}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-mode-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-specimen-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-key-buttons{display:grid;grid-template-columns:1fr 1fr 1fr;gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-button{height:32px;padding:0 4px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.14)}'
    + '#' + rootId + ' .bl-button.correct{border-color:#10B981;background:#D1FAE5;color:#065F46}'
    + '#' + rootId + ' .bl-button.wrong{border-color:#EF4444;background:#FEE2E2;color:#991B1B}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:14px;color:#15803D;min-height:20px}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .bc-species{animation:' + rootId + '-species var(--bc-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .bc-path{stroke-dasharray:8 7;animation:' + rootId + '-path 1.5s linear infinite}'
    + '#' + rootId + ' .bc-pulse{animation:' + rootId + '-pulse 1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-species{from{transform:translateY(3px);opacity:.58}to{transform:translateY(-4px);opacity:1}}'
    + '@keyframes ' + rootId + '-path{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.3}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_BIODIVERSITY_CLASSIFICATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-biodiversity-classification',
    group: '🦋 进化与生物多样性',
    name: '生物多样性与分类检索',
    emoji: '🌏',
    desc: '比较遗传、物种和生态系统多样性，并使用二歧检索表鉴定六种代表生物',
    params: [
      {
        key: 'habitatIntegrity',
        label: '栖息地完整度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 76,
        hint: '表示栖息地结构和生态过程保持完整的相对程度',
      },
      {
        key: 'samplingEffort',
        label: '调查取样努力',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 62,
        hint: '取样努力会影响调查中观察到的物种数',
      },
      {
        key: 'geneticVariation',
        label: '种内遗传变异',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'initialSpecimen',
        label: '初始检索标本',
        type: 'number',
        min: 1,
        max: 6,
        step: 1,
        defaultValue: 1,
      },
    ],

    buildHTML: (params, rootId) => {
      const habitatIntegrity = num(
        params,
        'habitatIntegrity',
        76,
      )
      const samplingEffort = num(
        params,
        'samplingEffort',
        62,
      )
      const geneticVariation = num(
        params,
        'geneticVariation',
        68,
      )
      const initialSpecimen = num(
        params,
        'initialSpecimen',
        1,
      )

      return `
<div id="${rootId}">
${biodiversityStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌏 生物多样性与分类检索</div>
    <div class="bl-note">多样性不只等于物种数量；二歧检索是鉴定工具</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row" data-diversity-row>
        <div class="bl-label">
          <span>栖息地完整度</span>
          <span class="bl-value" data-habitat-value></span>
        </div>
        <input
          data-habitat
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(habitatIntegrity)}"
        >
      </div>

      <div class="bl-row" data-diversity-row>
        <div class="bl-label">
          <span>调查取样努力</span>
          <span class="bl-value" data-sampling-value></span>
        </div>
        <input
          data-sampling
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(samplingEffort)}"
        >
      </div>

      <div class="bl-row" data-diversity-row>
        <div class="bl-label">
          <span>种内遗传变异</span>
          <span class="bl-value" data-genetic-value></span>
        </div>
        <input
          data-genetic
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(geneticVariation)}"
        >
      </div>

      <div class="bl-row" data-specimen-row>
        <div class="bl-label">
          <span>初始检索标本</span>
          <span class="bl-value" data-specimen-value></span>
        </div>
        <input
          data-specimen-range
          type="range"
          min="1"
          max="6"
          step="1"
          value="${n(initialSpecimen)}"
        >
      </div>

      <div class="bl-subtitle">观察模式</div>

      <div class="bl-mode-buttons">
        <button
          type="button"
          class="bl-button active"
          data-mode="diversity"
        >多样性层次</button>

        <button
          type="button"
          class="bl-button"
          data-mode="classification"
        >二歧检索</button>
      </div>

      <div data-specimen-controls>
        <div class="bl-subtitle">选择待检索标本</div>

        <div class="bl-specimen-buttons">
          <button type="button" class="bl-button" data-specimen="pine">松树</button>
          <button type="button" class="bl-button" data-specimen="fern">蕨</button>
          <button type="button" class="bl-button" data-specimen="mushroom">蘑菇</button>
          <button type="button" class="bl-button" data-specimen="bird">鸟</button>
          <button type="button" class="bl-button" data-specimen="butterfly">蝴蝶</button>
          <button type="button" class="bl-button" data-specimen="frog">青蛙</button>
        </div>
      </div>

      <div data-key-controls>
        <div class="bl-subtitle">回答当前二歧问题</div>

        <div class="bl-key-buttons">
          <button
            type="button"
            class="bl-button"
            data-key-answer="yes"
          >是</button>

          <button
            type="button"
            class="bl-button"
            data-key-answer="no"
          >否</button>

          <button
            type="button"
            class="bl-button"
            data-key-reset
          >重新开始</button>
        </div>
      </div>

      <div class="bl-status">
        <div class="bl-card">
          <b data-primary-status></b>
          <span data-primary-label></span>
        </div>

        <div class="bl-card">
          <b data-secondary-status></b>
          <span data-secondary-label></span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="生物多样性与二歧分类检索互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-green-card"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#ECFDF5"/>
            <stop offset="100%" stop-color="#D1FAE5"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-blue-card"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#EFF6FF"/>
            <stop offset="100%" stop-color="#DBEAFE"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-amber-card"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#FFFBEB"/>
            <stop offset="100%" stop-color="#FEF3C7"/>
          </linearGradient>

          <marker
            id="${rootId}-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#14532D"
              flood-opacity=".13"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="22"
          y="34"
          data-title
          font-size="24"
          font-weight="900"
          fill="#166534"
        ></text>

        <text
          x="22"
          y="62"
          data-summary
          font-size="13"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-graphic></g>

        <text
          x="24"
          y="397"
          data-footer
          font-size="11"
          font-weight="900"
          fill="#166534"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var habitatInput=root.querySelector(
      '[data-habitat]'
    );
    var samplingInput=root.querySelector(
      '[data-sampling]'
    );
    var geneticInput=root.querySelector(
      '[data-genetic]'
    );
    var specimenRange=root.querySelector(
      '[data-specimen-range]'
    );

    var habitatValue=root.querySelector(
      '[data-habitat-value]'
    );
    var samplingValue=root.querySelector(
      '[data-sampling-value]'
    );
    var geneticValue=root.querySelector(
      '[data-genetic-value]'
    );
    var specimenValue=root.querySelector(
      '[data-specimen-value]'
    );

    var diversityRows=root.querySelectorAll(
      '[data-diversity-row]'
    );
    var specimenRow=root.querySelector(
      '[data-specimen-row]'
    );
    var specimenControls=root.querySelector(
      '[data-specimen-controls]'
    );
    var keyControls=root.querySelector(
      '[data-key-controls]'
    );

    var modeButtons=root.querySelectorAll(
      '[data-mode]'
    );
    var specimenButtons=root.querySelectorAll(
      '[data-specimen]'
    );
    var answerButtons=root.querySelectorAll(
      '[data-key-answer]'
    );
    var resetButton=root.querySelector(
      '[data-key-reset]'
    );

    var primaryStatus=root.querySelector(
      '[data-primary-status]'
    );
    var secondaryStatus=root.querySelector(
      '[data-secondary-status]'
    );
    var primaryLabel=root.querySelector(
      '[data-primary-label]'
    );
    var secondaryLabel=root.querySelector(
      '[data-secondary-label]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var graphic=root.querySelector('[data-graphic]');
    var footer=root.querySelector('[data-footer]');

    var mode='diversity';

    var specimenOrder=[
      'pine',
      'fern',
      'mushroom',
      'bird',
      'butterfly',
      'frog'
    ];

    var initialIndex=Math.max(
      1,
      Math.min(
        6,
        Math.round(Number(specimenRange.value))
      )
    );

    var selectedSpecimen=
      specimenOrder[initialIndex-1];

    var keyNode='1';
    var keyPath=[];
    var identifiedSpecimen='';
    var keyComplete=false;

    var specimens={
      pine:{
        name:'松树',
        emoji:'🌲',
        group:'种子植物',
        result:'松树',
        traits:[
          '具有叶绿体，能进行光合作用',
          '能够形成种子',
          '常见木本种子植物'
        ],
        color:'#16A34A'
      },
      fern:{
        name:'蕨',
        emoji:'🌿',
        group:'蕨类植物',
        result:'蕨',
        traits:[
          '具有叶绿体，能进行光合作用',
          '不形成种子，以孢子繁殖',
          '具有根、茎、叶的分化'
        ],
        color:'#22C55E'
      },
      mushroom:{
        name:'蘑菇',
        emoji:'🍄',
        group:'真菌',
        result:'蘑菇',
        traits:[
          '不具有叶绿体',
          '通过菌丝等结构吸收营养',
          '真菌不属于植物'
        ],
        color:'#7C3AED'
      },
      bird:{
        name:'鸟',
        emoji:'🐦',
        group:'鸟类',
        result:'鸟',
        traits:[
          '属于动物',
          '体表具有羽毛',
          '具有脊柱'
        ],
        color:'#0EA5E9'
      },
      butterfly:{
        name:'蝴蝶',
        emoji:'🦋',
        group:'昆虫',
        result:'蝴蝶',
        traits:[
          '属于动物',
          '成体具有三对足',
          '具有外骨骼'
        ],
        color:'#F59E0B'
      },
      frog:{
        name:'青蛙',
        emoji:'🐸',
        group:'两栖动物',
        result:'青蛙',
        traits:[
          '属于动物',
          '成体通常具有四肢',
          '皮肤裸露而湿润'
        ],
        color:'#84CC16'
      }
    };

    var keyNodes={
      1:{
        question:'该生物具有叶绿体，能够进行光合作用吗？',
        yes:'2',
        no:'3',
        hint:'先判断是否属于能够进行光合作用的植物类群。'
      },
      2:{
        question:'该生物能够形成种子吗？',
        yes:'leaf:pine',
        no:'leaf:fern',
        hint:'种子的有无可区分本检索表中的松树和蕨。'
      },
      3:{
        question:'该生物通常固定生活，并通过菌丝等结构吸收营养吗？',
        yes:'leaf:mushroom',
        no:'4',
        hint:'真菌不含叶绿体，营养方式与动物不同。'
      },
      4:{
        question:'该动物体表具有羽毛吗？',
        yes:'leaf:bird',
        no:'5',
        hint:'羽毛是鸟类的重要识别特征。'
      },
      5:{
        question:'该动物成体具有三对足和外骨骼吗？',
        yes:'leaf:butterfly',
        no:'leaf:frog',
        hint:'昆虫成体具有三对足；青蛙属于两栖动物。'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    /**
     * 栖息地完整度影响教学模型中的潜在物种数。
     *
     * 这里的潜在物种数不是野外真实调查结果。
     */
    function calculatePotentialSpecies(
      habitatIntegrity
    ){
      return Math.round(
        6+habitatIntegrity*.14
      );
    }

    /**
     * 调查观察到的物种数同时受到：
     * 1. 环境中潜在物种数；
     * 2. 调查取样努力和检测机会。
     *
     * 取样努力增加时，观察数逐渐接近潜在数，
     * 但不会无限增加。
     */
    function calculateObservedSpecies(
      potentialSpecies,
      samplingEffort
    ){
      var detection=
        .2+.8*Math.sqrt(
          samplingEffort/100
        );

      return Math.min(
        potentialSpecies,
        Math.max(
          1,
          Math.round(
            potentialSpecies*detection
          )
        )
      );
    }

    /**
     * 综合教学指数只用于比较三个层次的共同变化，
     * 不对应真实生态学中的某个统一指数。
     */
    function calculateDiversityIndex(
      geneticVariation,
      habitatIntegrity,
      observedSpecies,
      potentialSpecies
    ){
      var genetic=
        geneticVariation;
      var species=
        potentialSpecies>0
          ?observedSpecies/potentialSpecies*100
          :0;
      var ecosystem=
        habitatIntegrity;

      return clamp(
        genetic*.3
        +species*.35
        +ecosystem*.35,
        0,
        100
      );
    }

    function levelCard(
      x,
      titleText,
      value,
      note,
      fill,
      color,
      emoji
    ){
      var barWidth=
        116*clamp(value/100,0,1);

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="'+x+'" y="82" width="145"'
        +' height="112" rx="15" fill="'+fill+'"'
        +' stroke="'+color+'" stroke-width="2.5"/>'
        +'<text x="'+(x+14)+'" y="107"'
        +' font-size="20">'+emoji+'</text>'
        +'<text x="'+(x+43)+'" y="107"'
        +' font-size="12.5" font-weight="900"'
        +' fill="'+color+'">'+titleText+'</text>'
        +'<text x="'+(x+14)+'" y="139"'
        +' font-size="23" font-weight="900"'
        +' fill="'+color+'">'+value.toFixed(0)+'</text>'
        +'<rect x="'+(x+14)+'" y="151"'
        +' width="116" height="11" rx="5.5"'
        +' fill="#E2E8F0"/>'
        +'<rect x="'+(x+14)+'" y="151"'
        +' width="'+barWidth+'" height="11" rx="5.5"'
        +' fill="'+color+'"/>'
        +'<text x="'+(x+14)+'" y="180"'
        +' font-size="9.5" font-weight="800"'
        +' fill="#475569">'+note+'</text>'
        +'</g>';
    }

    function speciesIcon(
      x,
      y,
      index,
      observed,
      habitatIntegrity
    ){
      var emojis=[
        '🌲','🌿','🌾','🌼','🍄',
        '🦋','🐝','🐞','🐸','🐟',
        '🐦','🦆','🐁','🦔','🐍',
        '🦎','🕷️','🐌','🌱','🪲'
      ];
      var emoji=emojis[
        index%emojis.length
      ];
      var opacity=observed
        ?1
        :.18;
      var scale=observed
        ?1
        :.82;

      return ''
        +'<g class="bc-species"'
        +' transform="translate('+x+' '+y+')'
        +' scale('+scale+')" opacity="'+opacity+'">'
        +'<circle cx="0" cy="0" r="16"'
        +' fill="#FFFFFF" stroke="'
        +(observed?'#16A34A':'#CBD5E1')
        +'" stroke-width="2"/>'
        +'<text x="0" y="6" text-anchor="middle"'
        +' font-size="20">'+emoji+'</text>'
        +'</g>';
    }

    function samplingCurve(
      potentialSpecies,
      currentEffort
    ){
      var left=428;
      var right=646;
      var top=231;
      var bottom=352;
      var width=right-left;
      var height=bottom-top;
      var path='';
      var points='';

      for(var effort=0;effort<=100;effort+=5){
        var observed=
          calculateObservedSpecies(
            potentialSpecies,
            Math.max(1,effort)
          );
        var x=left+width*effort/100;
        var y=bottom
          -height*observed/20;

        path+=(effort===0?'M':' L')
          +x+' '+y;

        if(effort%20===0){
          points+='<circle cx="'+x+'" cy="'+y
            +'" r="3.5" fill="#FFFFFF"'
            +' stroke="#0284C7" stroke-width="2"/>';
        }
      }

      var currentObserved=
        calculateObservedSpecies(
          potentialSpecies,
          currentEffort
        );
      var currentX=
        left+width*currentEffort/100;
      var currentY=
        bottom-height*currentObserved/20;

      return ''
        +'<text x="420" y="211" font-size="12"'
        +' font-weight="900" fill="#334155">'
        +'取样努力与观察物种数'
        +'</text>'
        +'<line x1="'+left+'" y1="'+bottom
        +'" x2="'+right+'" y2="'+bottom
        +'" stroke="#64748B" stroke-width="2.3"/>'
        +'<line x1="'+left+'" y1="'+bottom
        +'" x2="'+left+'" y2="'+top
        +'" stroke="#64748B" stroke-width="2.3"/>'
        +'<line x1="'+left+'" y1="'+(bottom-height*.5)
        +'" x2="'+right+'" y2="'+(bottom-height*.5)
        +'" stroke="#E2E8F0" stroke-width="1.2"/>'
        +'<line x1="'+left+'" y1="'+top
        +'" x2="'+right+'" y2="'+top
        +'" stroke="#E2E8F0" stroke-width="1.2"/>'
        +'<text x="'+(left-8)+'" y="'+(bottom+4)
        +'" text-anchor="end" font-size="9"'
        +' font-weight="700" fill="#64748B">0</text>'
        +'<text x="'+(left-8)+'" y="'+(bottom-height*.5+4)
        +'" text-anchor="end" font-size="9"'
        +' font-weight="700" fill="#64748B">10</text>'
        +'<text x="'+(left-8)+'" y="'+(top+4)
        +'" text-anchor="end" font-size="9"'
        +' font-weight="700" fill="#64748B">20</text>'
        +'<text x="'+left+'" y="'+(bottom+18)
        +'" text-anchor="middle" font-size="9"'
        +' font-weight="700" fill="#64748B">0</text>'
        +'<text x="'+right+'" y="'+(bottom+18)
        +'" text-anchor="middle" font-size="9"'
        +' font-weight="700" fill="#64748B">100%</text>'
        +'<path d="'+path+'" fill="none"'
        +' stroke="#0284C7" stroke-width="4"'
        +' stroke-linecap="round"'
        +' stroke-linejoin="round"/>'
        +points
        +'<circle class="bc-pulse" cx="'+currentX
        +'" cy="'+currentY+'" r="7"'
        +' fill="#FFFFFF" stroke="#EF4444"'
        +' stroke-width="4"/>'
        +'<text x="'+currentX+'" y="'+(currentY-12)
        +'" text-anchor="middle" font-size="9.5"'
        +' font-weight="900" fill="#B91C1C">'
        +currentObserved+'</text>';
    }

    function renderDiversity(
      habitatIntegrity,
      samplingEffort,
      geneticVariation
    ){
      var potentialSpecies=
        calculatePotentialSpecies(
          habitatIntegrity
        );
      var observedSpecies=
        calculateObservedSpecies(
          potentialSpecies,
          samplingEffort
        );
      var diversityIndex=
        calculateDiversityIndex(
          geneticVariation,
          habitatIntegrity,
          observedSpecies,
          potentialSpecies
        );

      var speciesLevel=
        potentialSpecies/20*100;
      var ecosystemLevel=
        habitatIntegrity;

      var html='';

      html+=levelCard(
        22,
        '遗传多样性',
        geneticVariation,
        '同一物种内部的遗传差异',
        'url(#${rootId}-green-card)',
        '#15803D',
        '🧬'
      );

      html+=levelCard(
        178,
        '物种多样性',
        speciesLevel,
        '物种丰富度与组成差异',
        'url(#${rootId}-blue-card)',
        '#1D4ED8',
        '🦋'
      );

      html+=levelCard(
        334,
        '生态系统多样性',
        ecosystemLevel,
        '不同生境及生态过程',
        'url(#${rootId}-amber-card)',
        '#B45309',
        '🌏'
      );

      html+='<text x="22" y="219" font-size="12"'
        +' font-weight="900" fill="#334155">'
        +'模拟调查记录：已观察 '
        +observedSpecies
        +' 种 / 潜在 '
        +potentialSpecies
        +' 种'
        +'</text>';

      var iconCount=Math.min(
        20,
        potentialSpecies
      );

      for(var i=0;i<iconCount;i++){
        var col=i%7;
        var row=Math.floor(i/7);
        var x=44+col*49;
        var y=252+row*45;
        var observed=i<observedSpecies;

        html+=speciesIcon(
          x,
          y,
          i,
          observed,
          habitatIntegrity
        );
      }

      html+=samplingCurve(
        potentialSpecies,
        samplingEffort
      );

      html+='<rect x="22" y="366" width="368"'
        +' height="22" rx="11" fill="#F1F5F9"/>';

      html+='<rect x="22" y="366" width="'
        +(368*diversityIndex/100)
        +'" height="22" rx="11" fill="#16A34A"/>';

      html+='<text x="206" y="381" text-anchor="middle"'
        +' font-size="11" font-weight="900" fill="#FFFFFF">'
        +'综合教学指数 '
        +diversityIndex.toFixed(0)
        +'</text>';

      return {
        html:html,
        potentialSpecies:potentialSpecies,
        observedSpecies:observedSpecies,
        diversityIndex:diversityIndex
      };
    }

    function specimenCard(
      specimen
    ){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="22" y="83" width="180"'
        +' height="270" rx="18" fill="#FFFFFF"'
        +' stroke="'+specimen.color+'"'
        +' stroke-width="3"/>'
        +'<circle cx="112" cy="145" r="48"'
        +' fill="'+specimen.color+'" opacity=".13"'
        +' stroke="'+specimen.color+'"'
        +' stroke-width="3"/>'
        +'<text x="112" y="161" text-anchor="middle"'
        +' font-size="54">'+specimen.emoji+'</text>'
        +'<text x="112" y="218" text-anchor="middle"'
        +' font-size="22" font-weight="900"'
        +' fill="'+specimen.color+'">'
        +specimen.name+'</text>'
        +'<text x="112" y="241" text-anchor="middle"'
        +' font-size="12" font-weight="900"'
        +' fill="#475569">'+specimen.group+'</text>';

      for(var i=0;i<specimen.traits.length;i++){
        var y=272+i*23;

        html+='<circle cx="38" cy="'+(y-4)
          +'" r="5" fill="'+specimen.color+'"/>';

        html+='<text x="51" y="'+y+'"'
          +' font-size="9.5" font-weight="800"'
          +' fill="#475569">'
          +specimen.traits[i]
          +'</text>';
      }

      html+='</g>';

      return html;
    }

    function pathText(answer){
      return answer==='yes'
        ?'是'
        :'否';
    }

    function renderPathHistory(){
      var html='';

      html+='<rect x="220" y="258" width="250"'
        +' height="95" rx="14" fill="#F8FAFC"'
        +' stroke="#CBD5E1" stroke-width="2"/>';

      html+='<text x="234" y="280"'
        +' font-size="11.5" font-weight="900"'
        +' fill="#334155">检索路径</text>';

      if(keyPath.length===0){
        html+='<text x="234" y="310"'
          +' font-size="10.5" font-weight="800"'
          +' fill="#64748B">'
          +'尚未回答问题'
          +'</text>';

        return html;
      }

      for(var i=0;i<keyPath.length;i++){
        var item=keyPath[i];
        var x=237+i*54;

        html+='<circle cx="'+x+'" cy="318" r="15"'
          +' fill="'
          +(item.answer==='yes'
            ?'#10B981'
            :'#F59E0B')
          +'" stroke="#FFFFFF" stroke-width="2"/>';

        html+='<text x="'+x+'" y="322"'
          +' text-anchor="middle" font-size="9"'
          +' font-weight="900" fill="#FFFFFF">'
          +pathText(item.answer)
          +'</text>';

        if(i<keyPath.length-1){
          html+='<path class="bc-path"'
            +' d="M'+(x+17)+' 318 H'
            +(x+37)
            +'" stroke="#16A34A" stroke-width="2.5"'
            +' marker-end="url(#${rootId}-arrow)"/>';
        }
      }

      return html;
    }

    function renderTraitReference(
      specimen
    ){
      var html='';

      html+='<rect x="490" y="83" width="168"'
        +' height="270" rx="16" fill="#FFFBEB"'
        +' stroke="#FCD34D" stroke-width="2.5"/>';

      html+='<text x="574" y="108" text-anchor="middle"'
        +' font-size="12.5" font-weight="900"'
        +' fill="#92400E">观察特征提示</text>';

      for(var i=0;i<specimen.traits.length;i++){
        var y=139+i*56;

        html+='<circle cx="510" cy="'+(y-4)
          +'" r="9" fill="'+specimen.color+'"/>';

        html+='<text x="510" y="'+y+'"'
          +' text-anchor="middle" font-size="9"'
          +' font-weight="900" fill="#FFFFFF">'
          +(i+1)+'</text>';

        var text=specimen.traits[i];
        var first=text.slice(0,11);
        var second=text.slice(11,22);
        var third=text.slice(22);

        html+='<text x="527" y="'+(y-9)
          +'" font-size="10" font-weight="800"'
          +' fill="#475569">'+first+'</text>';

        if(second){
          html+='<text x="527" y="'+(y+7)
            +'" font-size="10" font-weight="800"'
            +' fill="#475569">'+second+'</text>';
        }

        if(third){
          html+='<text x="527" y="'+(y+23)
            +'" font-size="10" font-weight="800"'
            +' fill="#475569">'+third+'</text>';
        }
      }

      html+='<text x="574" y="326" text-anchor="middle"'
        +' font-size="10" font-weight="900"'
        +' fill="#B45309">'
        +'每一步只选择“是”或“否”'
        +'</text>';

      return html;
    }

    function renderClassification(){
      var specimen=
        specimens[selectedSpecimen];
      var html='';

      html+=specimenCard(specimen);

      html+='<rect x="220" y="83" width="250"'
        +' height="155" rx="16" fill="url(#${rootId}-green-card)"'
        +' stroke="#86EFAC" stroke-width="2.5"/>';

      if(!keyComplete){
        var node=keyNodes[keyNode];

        html+='<text x="345" y="111"'
          +' text-anchor="middle" font-size="12"'
          +' font-weight="900" fill="#166534">'
          +'二歧检索问题 '+keyNode
          +'</text>';

        var question=node.question;
        var line1=question.slice(0,17);
        var line2=question.slice(17,34);
        var line3=question.slice(34);

        html+='<text x="345" y="147"'
          +' text-anchor="middle" font-size="13"'
          +' font-weight="900" fill="#334155">'
          +line1+'</text>';

        if(line2){
          html+='<text x="345" y="169"'
            +' text-anchor="middle" font-size="13"'
            +' font-weight="900" fill="#334155">'
            +line2+'</text>';
        }

        if(line3){
          html+='<text x="345" y="191"'
            +' text-anchor="middle" font-size="13"'
            +' font-weight="900" fill="#334155">'
            +line3+'</text>';
        }

        html+='<text x="345" y="221"'
          +' text-anchor="middle" font-size="10"'
          +' font-weight="800" fill="#64748B">'
          +node.hint+'</text>';
      }else{
        var identified=
          specimens[identifiedSpecimen];
        var correct=
          identifiedSpecimen===selectedSpecimen;

        html+='<text x="345" y="114"'
          +' text-anchor="middle" font-size="12"'
          +' font-weight="900" fill="'
          +(correct?'#047857':'#B91C1C')
          +'">'
          +(correct
            ?'检索完成：鉴定正确'
            :'检索完成：需要修正')
          +'</text>';

        html+='<text x="345" y="172"'
          +' text-anchor="middle" font-size="50">'
          +identified.emoji+'</text>';

        html+='<text x="345" y="211"'
          +' text-anchor="middle" font-size="20"'
          +' font-weight="900" fill="'
          +identified.color+'">'
          +'鉴定结果：'+identified.name
          +'</text>';
      }

      html+=renderPathHistory();
      html+=renderTraitReference(specimen);

      return html;
    }

    function resetKey(){
      keyNode='1';
      keyPath=[];
      identifiedSpecimen='';
      keyComplete=false;
    }

    function followKey(answer){
      if(keyComplete){
        return;
      }

      var node=keyNodes[keyNode];

      if(!node){
        return;
      }

      keyPath.push({
        node:keyNode,
        answer:answer
      });

      var next=answer==='yes'
        ?node.yes
        :node.no;

      if(next.indexOf('leaf:')===0){
        identifiedSpecimen=
          next.slice(5);
        keyComplete=true;
      }else{
        keyNode=next;
      }
    }

    function updateControlVisibility(){
      var diversityVisible=
        mode==='diversity';

      for(var i=0;i<diversityRows.length;i++){
        diversityRows[i].style.display=
          diversityVisible?'':'none';
      }

      specimenRow.style.display=
        diversityVisible?'none':'';

      specimenControls.style.display=
        diversityVisible?'none':'';

      keyControls.style.display=
        diversityVisible?'none':'';
    }

    function update(){
      var habitatIntegrity=Number(
        habitatInput.value
      );
      var samplingEffort=Number(
        samplingInput.value
      );
      var geneticVariation=Number(
        geneticInput.value
      );
      var specimenIndex=Math.max(
        1,
        Math.min(
          6,
          Math.round(Number(specimenRange.value))
        )
      );

      habitatValue.textContent=
        habitatIntegrity.toFixed(0)+'%';
      samplingValue.textContent=
        samplingEffort.toFixed(0)+'%';
      geneticValue.textContent=
        geneticVariation.toFixed(0)+'%';
      specimenValue.textContent=
        '第 '+specimenIndex+' 个';

      root.style.setProperty(
        '--bc-speed',
        clamp(
          2.5-habitatIntegrity/65,
          .55,
          2.4
        ).toFixed(2)+'s'
      );

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      for(var j=0;j<specimenButtons.length;j++){
        specimenButtons[j].classList.toggle(
          'active',
          specimenButtons[j].getAttribute(
            'data-specimen'
          )===selectedSpecimen
        );
      }

      for(var k=0;k<answerButtons.length;k++){
        answerButtons[k].classList.remove(
          'correct',
          'wrong'
        );
      }

      updateControlVisibility();

      if(mode==='diversity'){
        var diversity=renderDiversity(
          habitatIntegrity,
          samplingEffort,
          geneticVariation
        );

        title.textContent=
          '生物多样性的三个层次';

        summary.textContent=
          '遗传多样性、物种多样性和生态系统多样性彼此关联，但不能相互替代';

        graphic.innerHTML=
          diversity.html;

        primaryStatus.textContent=
          diversity.observedSpecies
          +' / '
          +diversity.potentialSpecies;

        primaryLabel.textContent=
          '观察种数 / 潜在种数';

        secondaryStatus.textContent=
          diversity.diversityIndex.toFixed(0);

        secondaryLabel.textContent=
          '综合教学指数';

        footer.textContent=
          '调查观察数受取样努力影响，不等于环境中的真实物种总数';

        var condition='';

        if(habitatIntegrity<30){
          condition=
            '栖息地完整度较低，多个层次的生物多样性都可能受到影响。';
        }else if(samplingEffort<25){
          condition=
            '取样努力较低，尚未观察到的物种不应直接判断为不存在。';
        }else if(geneticVariation<25){
          condition=
            '种内遗传变异较低，种群面对环境变化时可供选择作用的遗传差异可能较少。';
        }else{
          condition=
            '当前栖息地、取样和种内变异条件处于中等或较高水平。';
        }

        result.innerHTML=
          '生物多样性包括遗传、物种和生态系统三个层次，不能只用物种数量概括。'
          +'<br>'+condition
          +' 增加取样努力通常会发现更多物种，但观察数会逐渐接近环境中的潜在物种数，而不是无限增加。';
      }else{
        var specimen=
          specimens[selectedSpecimen];

        title.textContent=
          '二歧检索：根据特征逐步鉴定';

        summary.textContent=
          '每一步从两个互斥选项中选择一个，沿检索路径到达鉴定结果';

        graphic.innerHTML=
          renderClassification();

        primaryStatus.textContent=
          specimen.name;

        primaryLabel.textContent=
          '当前待检索标本';

        secondaryStatus.textContent=
          keyComplete
            ?(
              identifiedSpecimen===selectedSpecimen
                ?'正确'
                :'需修正'
            )
            :'第 '+keyNode+' 步';

        secondaryLabel.textContent=
          keyComplete
            ?'鉴定状态'
            :'当前检索步骤';

        footer.textContent=
          '二歧检索表用于鉴定，不等于完整的系统发育树或亲缘关系图';

        if(!keyComplete){
          result.innerHTML=
            '请观察“'
            +specimen.name
            +'”的特征，并回答当前问题。'
            +'<br>二歧检索每一步只给出两个互斥选项；选择错误时可能到达不正确的鉴定结果。';
        }else if(
          identifiedSpecimen===selectedSpecimen
        ){
          result.innerHTML=
            '鉴定正确：检索结果为“'
            +specimen.name
            +'”，所属示例类群为“'
            +specimen.group
            +'”。'
            +'<br>二歧检索是依据特征进行鉴定的工具，不代表完整展示生物的进化亲缘关系。';
        }else{
          var wrong=
            specimens[identifiedSpecimen];

          result.innerHTML=
            '当前路径把标本鉴定为“'
            +wrong.name
            +'”，与待检索标本“'
            +specimen.name
            +'”不一致。'
            +'<br>请重新开始，并逐项核对叶绿体、种子、营养方式、羽毛、足的数量和外骨骼等特征。';
        }
      }
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');

        if(mode==='classification'){
          var rangeIndex=Math.max(
            1,
            Math.min(
              6,
              Math.round(
                Number(specimenRange.value)
              )
            )
          );

          selectedSpecimen=
            specimenOrder[rangeIndex-1];

          resetKey();
        }

        update();
      };
    }

    for(var j=0;j<specimenButtons.length;j++){
      specimenButtons[j].onclick=function(){
        selectedSpecimen=this.getAttribute(
          'data-specimen'
        );

        specimenRange.value=String(
          specimenOrder.indexOf(
            selectedSpecimen
          )+1
        );

        resetKey();
        update();
      };
    }

    for(var k=0;k<answerButtons.length;k++){
      answerButtons[k].onclick=function(){
        var answer=this.getAttribute(
          'data-key-answer'
        );

        followKey(answer);
        update();
      };
    }

    resetButton.onclick=function(){
      resetKey();
      update();
    };

    habitatInput.oninput=update;
    samplingInput.oninput=update;
    geneticInput.oninput=update;

    specimenRange.oninput=function(){
      var index=Math.max(
        1,
        Math.min(
          6,
          Math.round(
            Number(specimenRange.value)
          )
        )
      );

      selectedSpecimen=
        specimenOrder[index-1];

      resetKey();
      update();
    };

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
