/**
 * lifeScienceLabTemplatesInheritancePedigreeAnalysis.ts
 *
 * 平面生命科学实验室：遗传系谱分析。
 *
 * 教学目标：
 * 1. 认识遗传系谱图中的男性、女性、患病、正常和携带者符号；
 * 2. 根据亲子关系、性别分布和代际连续性提取遗传证据；
 * 3. 比较常染色体显性、常染色体隐性和伴X染色体隐性遗传；
 * 4. 根据系谱证据推断部分个体的可能基因型；
 * 5. 理解“未患病”不一定意味着“不携带隐性致病等位基因”；
 * 6. 理解小型家系可能不能唯一确定遗传方式，需要结合更多证据判断。
 *
 * 教学边界：
 * 1. 本模型只比较常染色体显性、常染色体隐性和伴X隐性三种典型方式；
 * 2. 三个案例均为教学构造家系，不代表真实患者资料；
 * 3. 默认完全外显，不讨论外显率不全、迟发、嵌合和新发突变；
 * 4. 默认没有表型误判、收养、非亲生等影响系谱解释的因素；
 * 5. 携带者和基因型标记属于教学答案，可由开关显示或隐藏；
 * 6. 真实遗传咨询不能仅凭简化系谱图作出临床结论；
 * 7. 伴X隐性遗传中不存在父亲把X染色体直接传给儿子的过程。
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
 * 安全读取布尔参数。
 */
function bool(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]

  return typeof value === 'boolean'
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
function pedigreeStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FBCFE8;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#EDE9FE);border-bottom:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#FFFAFC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#DB2777}'
    + '#' + rootId + ' input[type=checkbox]{width:15px;height:15px;accent-color:#DB2777;cursor:pointer}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .bl-case-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-answer-buttons{display:grid;grid-template-columns:1fr;gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .bl-button{height:31px;padding:0 5px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.14)}'
    + '#' + rootId + ' .bl-button.correct{border-color:#10B981;background:#D1FAE5;color:#065F46}'
    + '#' + rootId + ' .bl-button.wrong{border-color:#EF4444;background:#FEE2E2;color:#991B1B}'
    + '#' + rootId + ' .bl-checks{display:grid;grid-template-columns:1fr;gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-check{display:flex;align-items:center;justify-content:space-between;gap:8px;padding:6px 8px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;font-size:10.5px;font-weight:800;color:#475569}'
    + '#' + rootId + ' .bl-verify{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:13px;color:#BE185D;min-height:20px}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .pa-evidence{stroke-dasharray:8 7;animation:' + rootId + '-evidence 1.4s linear infinite}'
    + '#' + rootId + ' .pa-halo{animation:' + rootId + '-halo 1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-evidence{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-halo{from{opacity:.25}to{opacity:.82}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_PEDIGREE_ANALYSIS:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-pedigree-analysis',
    group: '🧬 遗传规律',
    name: '遗传系谱分析',
    emoji: '👨‍👩‍👧‍👦',
    desc: '分析三个匿名家系案例，比较常染色体显性、常染色体隐性和伴X隐性遗传',
    params: [
      {
        key: 'caseIndex',
        label: '初始案例编号',
        type: 'number',
        min: 1,
        max: 3,
        step: 1,
        defaultValue: 1,
      },
      {
        key: 'showGenotypes',
        label: '显示基因型答案',
        type: 'boolean',
        defaultValue: false,
      },
      {
        key: 'showCarriers',
        label: '显示已知携带者',
        type: 'boolean',
        defaultValue: false,
      },
      {
        key: 'highlightEvidence',
        label: '突出关键证据',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const caseIndex = num(params, 'caseIndex', 1)
      const showGenotypes = bool(
        params,
        'showGenotypes',
        false,
      )
      const showCarriers = bool(
        params,
        'showCarriers',
        false,
      )
      const highlightEvidence = bool(
        params,
        'highlightEvidence',
        true,
      )

      return `
<div id="${rootId}">
${pedigreeStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">👨‍👩‍👧‍👦 遗传系谱分析</div>
    <div class="bl-note">教学构造家系：先判断，再显示证据和基因型</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>初始案例编号</span>
          <span class="bl-value" data-case-value></span>
        </div>
        <input
          data-case-range
          type="range"
          min="1"
          max="3"
          step="1"
          value="${n(caseIndex)}"
        >
      </div>

      <div class="bl-subtitle">选择匿名家系案例</div>

      <div class="bl-case-buttons">
        <button
          type="button"
          class="bl-button"
          data-case="1"
        >案例A</button>

        <button
          type="button"
          class="bl-button"
          data-case="2"
        >案例B</button>

        <button
          type="button"
          class="bl-button"
          data-case="3"
        >案例C</button>
      </div>

      <div class="bl-subtitle">选择你的判断</div>

      <div class="bl-answer-buttons">
        <button
          type="button"
          class="bl-button"
          data-answer="autosomalDominant"
        >常染色体显性</button>

        <button
          type="button"
          class="bl-button"
          data-answer="autosomalRecessive"
        >常染色体隐性</button>

        <button
          type="button"
          class="bl-button"
          data-answer="xLinkedRecessive"
        >伴X染色体隐性</button>
      </div>

      <button
        type="button"
        class="bl-verify"
        data-verify
      >验证遗传方式判断</button>

      <div class="bl-checks">
        <label class="bl-check">
          <span>显示基因型答案</span>
          <input
            data-show-genotypes
            type="checkbox"
            ${showGenotypes ? 'checked' : ''}
          >
        </label>

        <label class="bl-check">
          <span>显示已知携带者</span>
          <input
            data-show-carriers
            type="checkbox"
            ${showCarriers ? 'checked' : ''}
          >
        </label>

        <label class="bl-check">
          <span>突出关键证据</span>
          <input
            data-highlight-evidence
            type="checkbox"
            ${highlightEvidence ? 'checked' : ''}
          >
        </label>
      </div>

      <div class="bl-status">
        <div class="bl-card">
          <b data-selected-answer></b>
          <span>当前选择</span>
        </div>

        <div class="bl-card">
          <b data-check-state></b>
          <span>验证状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="遗传系谱分析互动图"
      >
        <defs>
          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#831843"
              flood-opacity=".14"
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
          fill="#9D174D"
        ></text>

        <text
          x="22"
          y="62"
          data-summary
          font-size="13"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-pedigree-layer></g>
        <g data-evidence-layer></g>

        <g transform="translate(22 386)">
          <rect x="0" y="-11" width="18" height="18" fill="#FFFFFF" stroke="#334155" stroke-width="2.5"/>
          <text x="27" y="3" font-size="11" font-weight="800" fill="#475569">男性</text>
        </g>

        <g transform="translate(94 386)">
          <circle cx="9" cy="-2" r="9" fill="#FFFFFF" stroke="#334155" stroke-width="2.5"/>
          <text x="27" y="3" font-size="11" font-weight="800" fill="#475569">女性</text>
        </g>

        <g transform="translate(166 386)">
          <rect x="0" y="-11" width="18" height="18" fill="#334155" stroke="#334155" stroke-width="2.5"/>
          <text x="27" y="3" font-size="11" font-weight="800" fill="#475569">患病</text>
        </g>

        <g transform="translate(248 386)">
          <circle cx="9" cy="-2" r="9" fill="#FFFFFF" stroke="#334155" stroke-width="2.5"/>
          <circle cx="9" cy="-2" r="3.5" fill="#F59E0B"/>
          <text x="27" y="3" font-size="11" font-weight="800" fill="#475569">已知携带者</text>
        </g>

        <text
          x="425"
          y="390"
          data-footer-note
          font-size="11"
          font-weight="900"
          fill="#9D174D"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var caseRange=root.querySelector(
      '[data-case-range]'
    );
    var caseValue=root.querySelector(
      '[data-case-value]'
    );

    var caseButtons=root.querySelectorAll(
      '[data-case]'
    );
    var answerButtons=root.querySelectorAll(
      '[data-answer]'
    );
    var verifyButton=root.querySelector(
      '[data-verify]'
    );

    var showGenotypesInput=root.querySelector(
      '[data-show-genotypes]'
    );
    var showCarriersInput=root.querySelector(
      '[data-show-carriers]'
    );
    var highlightEvidenceInput=root.querySelector(
      '[data-highlight-evidence]'
    );

    var selectedAnswerValue=root.querySelector(
      '[data-selected-answer]'
    );
    var checkState=root.querySelector(
      '[data-check-state]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var pedigreeLayer=root.querySelector(
      '[data-pedigree-layer]'
    );
    var evidenceLayer=root.querySelector(
      '[data-evidence-layer]'
    );
    var footerNote=root.querySelector(
      '[data-footer-note]'
    );

    var currentCase=Math.max(
      1,
      Math.min(
        3,
        Math.round(Number(caseRange.value))
      )
    );

    var selectedAnswer='';
    var verified=false;

    var answerNames={
      autosomalDominant:'常染色体显性',
      autosomalRecessive:'常染色体隐性',
      xLinkedRecessive:'伴X染色体隐性'
    };

    var cases={
      1:{
        caseName:'案例A',
        title:'案例A：患病性状连续出现在多代',
        summary:'观察男女患病分布、代际连续性和父子传递',
        correct:'autosomalDominant',
        conclusion:'常染色体显性',
        evidence:[
          '患病个体在连续多代中出现',
          '男性和女性都可以患病',
          '存在患病父亲向儿子传递'
        ],
        teaching:'患病父亲能够把性状传给儿子，说明该致病基因不位于X染色体上；连续多代出现更符合常染色体显性遗传。',
        footer:'关键：连续遗传，并存在父子传递'
      },
      2:{
        caseName:'案例B',
        title:'案例B：正常父母生出患病子女',
        summary:'观察性状是否跳代，以及父母表型与子代表型的关系',
        correct:'autosomalRecessive',
        conclusion:'常染色体隐性',
        evidence:[
          '表型正常的父母可以生出患病子女',
          '男性和女性都可以患病',
          '性状可能出现隔代或跳代'
        ],
        teaching:'两个表型正常但携带隐性致病等位基因的亲本，可以生出隐性纯合患病子女，更符合常染色体隐性遗传。',
        footer:'关键：正常父母可生患病子女'
      },
      3:{
        caseName:'案例C',
        title:'案例C：患病者主要为男性',
        summary:'观察携带者母亲、患病儿子以及父子之间的传递关系',
        correct:'xLinkedRecessive',
        conclusion:'伴X染色体隐性',
        evidence:[
          '患病个体主要为男性',
          '携带者女性可以生出患病儿子',
          '没有父亲把致病X染色体传给儿子'
        ],
        teaching:'儿子的X染色体来自母亲，父亲把Y染色体传给儿子；患病男性较多且没有父子直接传递，更符合伴X隐性遗传。',
        footer:'关键：男性多见，无父子直接传递'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function line(
      x1,
      y1,
      x2,
      y2,
      width,
      color,
      className
    ){
      return '<line x1="'+x1+'" y1="'+y1
        +'" x2="'+x2+'" y2="'+y2
        +'" stroke="'+(color||'#64748B')
        +'" stroke-width="'+(width||2.5)+'"'
        +(className?' class="'+className+'"':'')
        +'/>';
    }

    function halo(x,y,sex){
      var shape=sex==='male'
        ?'<rect x="'+(x-20)+'" y="'+(y-20)
          +'" width="40" height="40" rx="7"'
          +' fill="none" stroke="#F59E0B"'
          +' stroke-width="5"/>'
        :'<circle cx="'+x+'" cy="'+y+'" r="21"'
          +' fill="none" stroke="#F59E0B"'
          +' stroke-width="5"/>';

      return '<g class="pa-halo">'+shape+'</g>';
    }

    /**
     * 绘制一个系谱个体。
     *
     * male使用方形，female使用圆形；
     * affected使用深色填充；
     * carrier在显示携带者时使用中心橙点；
     * genotype在显示答案时写在符号下方。
     */
    function person(
      x,
      y,
      sex,
      affected,
      carrier,
      id,
      genotype,
      highlighted
    ){
      var showGenotypes=
        showGenotypesInput.checked;
      var showCarriers=
        showCarriersInput.checked;
      var highlight=
        highlightEvidenceInput.checked
        &&highlighted;

      var fill=affected
        ?'#334155'
        :'#FFFFFF';
      var stroke=highlight
        ?'#F59E0B'
        :'#334155';
      var html='';

      if(highlight){
        html+=halo(x,y,sex);
      }

      if(sex==='male'){
        html+='<rect x="'+(x-15)+'" y="'+(y-15)
          +'" width="30" height="30" rx="3"'
          +' fill="'+fill+'" stroke="'+stroke
          +'" stroke-width="'+(highlight?4:3)+'"/>';
      }else{
        html+='<circle cx="'+x+'" cy="'+y+'" r="15"'
          +' fill="'+fill+'" stroke="'+stroke
          +'" stroke-width="'+(highlight?4:3)+'"/>';
      }

      if(carrier && showCarriers){
        html+='<circle cx="'+x+'" cy="'+y+'" r="5"'
          +' fill="#F59E0B" stroke="#B45309"'
          +' stroke-width="1.5"/>';
      }

      html+='<text x="'+x+'" y="'+(y+31)
        +'" text-anchor="middle" font-size="9.5"'
        +' font-weight="900" fill="#475569">'
        +id+'</text>';

      if(showGenotypes){
        html+='<text x="'+x+'" y="'+(y+44)
          +'" text-anchor="middle" font-size="9.5"'
          +' font-weight="900" fill="#9D174D">'
          +genotype+'</text>';
      }

      return html;
    }

    function generationLabel(
      label,
      y
    ){
      return '<text x="24" y="'+(y+5)
        +'" font-size="14" font-weight="900"'
        +' fill="#9D174D">'+label+'</text>';
    }

    function marriage(
      x1,
      x2,
      y
    ){
      return line(
        x1+15,
        y,
        x2-15,
        y,
        3,
        '#64748B'
      );
    }

    function offspringStem(
      centerX,
      parentY,
      branchY
    ){
      return line(
        centerX,
        parentY,
        centerX,
        branchY,
        2.5,
        '#64748B'
      );
    }

    function childBranch(
      firstX,
      lastX,
      branchY
    ){
      return line(
        firstX,
        branchY,
        lastX,
        branchY,
        2.5,
        '#64748B'
      );
    }

    function childStem(
      x,
      branchY,
      childY
    ){
      return line(
        x,
        branchY,
        x,
        childY-15,
        2.5,
        '#64748B'
      );
    }

    /**
     * 案例A：常染色体显性。
     *
     * 关键证据：
     * 1. 连续多代出现；
     * 2. 男女均可患病；
     * 3. II-3患病父亲把致病等位基因传给III-4儿子。
     */
    function renderCaseOne(){
      var html='';

      html+=generationLabel('Ⅰ',102);
      html+=generationLabel('Ⅱ',202);
      html+=generationLabel('Ⅲ',307);

      html+=marriage(190,280,102);
      html+=offspringStem(235,102,142);
      html+=childBranch(105,385,142);
      html+=childStem(105,142,202);
      html+=childStem(245,142,202);
      html+=childStem(385,142,202);

      html+=marriage(55,105,202);
      html+=offspringStem(80,202,246);
      html+=childBranch(55,105,246);
      html+=childStem(55,246,307);
      html+=childStem(105,246,307);

      html+=marriage(385,435,202);
      html+=offspringStem(410,202,246);
      html+=childBranch(385,435,246);
      html+=childStem(385,246,307);
      html+=childStem(435,246,307);

      html+=person(
        190,102,'male',true,false,
        'Ⅰ-1','Aa',true
      );
      html+=person(
        280,102,'female',false,false,
        'Ⅰ-2','aa',false
      );

      html+=person(
        55,202,'male',false,false,
        'Ⅱ-0','aa',false
      );
      html+=person(
        105,202,'female',true,false,
        'Ⅱ-1','Aa',true
      );
      html+=person(
        245,202,'male',false,false,
        'Ⅱ-2','aa',false
      );
      html+=person(
        385,202,'male',true,false,
        'Ⅱ-3','Aa',true
      );
      html+=person(
        435,202,'female',false,false,
        'Ⅱ-4','aa',false
      );

      html+=person(
        55,307,'male',true,false,
        'Ⅲ-1','Aa',false
      );
      html+=person(
        105,307,'female',false,false,
        'Ⅲ-2','aa',false
      );
      html+=person(
        385,307,'female',false,false,
        'Ⅲ-3','aa',false
      );
      html+=person(
        435,307,'male',true,false,
        'Ⅲ-4','Aa',true
      );

      return html;
    }

    /**
     * 案例B：常染色体隐性。
     *
     * 关键证据：
     * 1. Ⅰ-1和Ⅰ-2均表型正常，却生出患病女儿；
     * 2. Ⅱ-2和配偶均表型正常，却生出患病儿子；
     * 3. 男女均可患病。
     */
    function renderCaseTwo(){
      var html='';

      html+=generationLabel('Ⅰ',102);
      html+=generationLabel('Ⅱ',202);
      html+=generationLabel('Ⅲ',307);

      html+=marriage(190,280,102);
      html+=offspringStem(235,102,142);
      html+=childBranch(105,395,142);
      html+=childStem(105,142,202);
      html+=childStem(255,142,202);
      html+=childStem(395,142,202);

      html+=marriage(255,315,202);
      html+=offspringStem(285,202,246);
      html+=childBranch(250,320,246);
      html+=childStem(250,246,307);
      html+=childStem(320,246,307);

      html+=person(
        190,102,'male',false,true,
        'Ⅰ-1','Aa',true
      );
      html+=person(
        280,102,'female',false,true,
        'Ⅰ-2','Aa',true
      );

      html+=person(
        105,202,'female',true,false,
        'Ⅱ-1','aa',true
      );
      html+=person(
        255,202,'male',false,true,
        'Ⅱ-2','Aa',true
      );
      html+=person(
        315,202,'female',false,true,
        'Ⅱ-3','Aa',true
      );
      html+=person(
        395,202,'female',false,true,
        'Ⅱ-4','Aa',false
      );

      html+=person(
        250,307,'male',true,false,
        'Ⅲ-1','aa',true
      );
      html+=person(
        320,307,'female',false,false,
        'Ⅲ-2','AA或Aa',false
      );

      return html;
    }

    /**
     * 案例C：伴X染色体隐性。
     *
     * 关键证据：
     * 1. 患病者主要为男性；
     * 2. 携带者女性可生出患病儿子；
     * 3. 患病父亲的儿子不直接获得父亲的X染色体。
     */
    function renderCaseThree(){
      var html='';

      html+=generationLabel('Ⅰ',102);
      html+=generationLabel('Ⅱ',202);
      html+=generationLabel('Ⅲ',307);

      html+=marriage(190,280,102);
      html+=offspringStem(235,102,142);
      html+=childBranch(75,425,142);
      html+=childStem(75,142,202);
      html+=childStem(195,142,202);
      html+=childStem(315,142,202);
      html+=childStem(425,142,202);

      html+=marriage(75,125,202);
      html+=offspringStem(100,202,246);
      html+=childBranch(75,125,246);
      html+=childStem(75,246,307);
      html+=childStem(125,246,307);

      html+=marriage(195,245,202);
      html+=offspringStem(220,202,246);
      html+=childBranch(185,255,246);
      html+=childStem(185,246,307);
      html+=childStem(220,246,307);
      html+=childStem(255,246,307);

      html+=person(
        190,102,'male',false,false,
        'Ⅰ-1','XᴬY',false
      );
      html+=person(
        280,102,'female',false,true,
        'Ⅰ-2','XᴬXᵃ',true
      );

      html+=person(
        75,202,'male',true,false,
        'Ⅱ-1','XᵃY',true
      );
      html+=person(
        125,202,'female',false,false,
        'Ⅱ-0','XᴬXᴬ',false
      );
      html+=person(
        195,202,'female',false,true,
        'Ⅱ-2','XᴬXᵃ',true
      );
      html+=person(
        245,202,'male',false,false,
        'Ⅱ-5','XᴬY',false
      );
      html+=person(
        315,202,'male',false,false,
        'Ⅱ-3','XᴬY',false
      );
      html+=person(
        425,202,'female',false,false,
        'Ⅱ-4','XᴬXᴬ或XᴬXᵃ',false
      );

      html+=person(
        75,307,'female',false,true,
        'Ⅲ-1','XᴬXᵃ',true
      );
      html+=person(
        125,307,'male',false,false,
        'Ⅲ-2','XᴬY',true
      );

      html+=person(
        185,307,'male',true,false,
        'Ⅲ-3','XᵃY',true
      );
      html+=person(
        220,307,'female',false,true,
        'Ⅲ-4','XᴬXᵃ',false
      );
      html+=person(
        255,307,'male',false,false,
        'Ⅲ-5','XᴬY',false
      );

      return html;
    }

    function renderPedigree(){
      if(currentCase===1){
        return renderCaseOne();
      }

      if(currentCase===2){
        return renderCaseTwo();
      }

      return renderCaseThree();
    }

    function evidencePanel(info){
      var show=
        highlightEvidenceInput.checked;
      var html='';

      html+='<g transform="translate(488 84)">';

      html+='<rect x="0" y="0" width="170" height="258"'
        +' rx="15" fill="#FFF7FB"'
        +' stroke="#F9A8D4" stroke-width="2.5"/>';

      html+='<text x="13" y="25" font-size="13"'
        +' font-weight="900" fill="#9D174D">'
        +'系谱证据清单</text>';

      for(var i=0;i<info.evidence.length;i++){
        var y=54+i*62;
        var color=show
          ?'#DB2777'
          :'#94A3B8';

        html+='<circle cx="18" cy="'+(y-3)+'" r="9"'
          +' fill="'+color+'" opacity="'
          +(show?'1':'.45')+'"/>';

        html+='<text x="18" y="'+(y+1)+'"'
          +' text-anchor="middle" font-size="9"'
          +' font-weight="900" fill="#FFFFFF">'
          +(i+1)+'</text>';

        var words=info.evidence[i];
        var first=words.slice(0,12);
        var second=words.slice(12,24);
        var third=words.slice(24);

        html+='<text x="34" y="'+(y-7)+'"'
          +' font-size="10.5" font-weight="800"'
          +' fill="#475569">'+first+'</text>';

        if(second){
          html+='<text x="34" y="'+(y+9)+'"'
            +' font-size="10.5" font-weight="800"'
            +' fill="#475569">'+second+'</text>';
        }

        if(third){
          html+='<text x="34" y="'+(y+25)+'"'
            +' font-size="10.5" font-weight="800"'
            +' fill="#475569">'+third+'</text>';
        }
      }

      html+='<rect x="11" y="216" width="148" height="30"'
        +' rx="9" fill="'
        +(verified?'#FCE7F3':'#F1F5F9')
        +'"/>';

      html+='<text x="85" y="236" text-anchor="middle"'
        +' font-size="11" font-weight="900" fill="'
        +(verified?'#9D174D':'#64748B')
        +'">'
        +(verified
          ?'结论：'+info.conclusion
          :'先选择并验证遗传方式')
        +'</text>';

      html+='</g>';

      return html;
    }

    function updateAnswerButtons(){
      var info=cases[currentCase];

      for(var i=0;i<answerButtons.length;i++){
        var answer=answerButtons[i].getAttribute(
          'data-answer'
        );

        answerButtons[i].classList.toggle(
          'active',
          answer===selectedAnswer
        );

        answerButtons[i].classList.remove(
          'correct',
          'wrong'
        );

        if(verified && answer===selectedAnswer){
          answerButtons[i].classList.add(
            answer===info.correct
              ?'correct'
              :'wrong'
          );
        }

        if(
          verified
          &&selectedAnswer
          &&answer===info.correct
        ){
          answerButtons[i].classList.add(
            'correct'
          );
        }
      }
    }

    function update(){
      currentCase=clamp(
        Math.round(Number(caseRange.value)),
        1,
        3
      );

      var info=cases[currentCase];

      caseValue.textContent='案例 '
        +String.fromCharCode(64+currentCase);

      for(var i=0;i<caseButtons.length;i++){
        caseButtons[i].classList.toggle(
          'active',
          Number(
            caseButtons[i].getAttribute('data-case')
          )===currentCase
        );
      }

      updateAnswerButtons();

      selectedAnswerValue.textContent=
        selectedAnswer
          ?answerNames[selectedAnswer]
          :'尚未选择';

      if(!verified){
        checkState.textContent='等待验证';
      }else if(selectedAnswer===info.correct){
        checkState.textContent='判断正确';
      }else{
        checkState.textContent='需要修正';
      }

      title.textContent=info.title;
      summary.textContent=info.summary;
      footerNote.textContent=info.footer;

      pedigreeLayer.innerHTML=renderPedigree();
      evidenceLayer.innerHTML=evidencePanel(info);

      if(!selectedAnswer){
        result.innerHTML=
          '请先观察患病个体的性别、亲子关系和代际分布，再选择一种遗传方式。'
          +'<br>小型家系有时不能唯一确定遗传方式，应结合更多家系成员和遗传检测。';
      }else if(!verified){
        result.innerHTML=
          '你当前选择了“'
          +answerNames[selectedAnswer]
          +'”。请点击“验证遗传方式判断”，再对照关键证据。';
      }else if(selectedAnswer===info.correct){
        result.innerHTML=
          '判断正确：'+info.teaching
          +'<br>显示基因型和携带者后，可继续检查每一组亲子关系是否符合该遗传方式。';
      }else{
        result.innerHTML=
          '当前判断与本案例设计不一致。'
          +'<br>'+info.teaching
          +' 请重新比较患病者性别、父子传递和正常父母能否生出患病子女。';
      }
    }

    for(var i=0;i<caseButtons.length;i++){
      caseButtons[i].onclick=function(){
        currentCase=Number(
          this.getAttribute('data-case')
        );
        caseRange.value=String(currentCase);
        selectedAnswer='';
        verified=false;
        update();
      };
    }

    for(var j=0;j<answerButtons.length;j++){
      answerButtons[j].onclick=function(){
        selectedAnswer=this.getAttribute(
          'data-answer'
        );
        verified=false;
        update();
      };
    }

    verifyButton.onclick=function(){
      if(!selectedAnswer){
        verified=false;
        update();
        return;
      }

      verified=true;
      update();
    };

    caseRange.oninput=function(){
      currentCase=Math.round(
        Number(caseRange.value)
      );
      selectedAnswer='';
      verified=false;
      update();
    };

    showGenotypesInput.onchange=update;
    showCarriersInput.onchange=update;
    highlightEvidenceInput.onchange=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
