/**
 * lifeScienceLabTemplatesInheritanceSexLinked.ts
 *
 * 平面生命科学实验室：
 * 伴性遗传与性染色体传递。
 *
 * 教学目标：
 * 1. 理解简化XX—XY模型中，母亲的卵细胞通常携带X染色体，
 *    父亲的精子可能携带X或Y染色体；
 * 2. 理解父亲把X染色体传给女儿，把Y染色体传给儿子；
 * 3. 理解父亲不会把X染色体直接传给儿子；
 * 4. 使用棋盘格比较伴X隐性遗传、伴X显性遗传
 *    和伴Y遗传的典型传递路径；
 * 5. 理解伴X隐性遗传中，杂合女性可能表现正常但作为携带者；
 * 6. 理解伴X隐性男性只要其唯一X染色体携带隐性变异，
 *    就会在完全外显模型中表现相应性状；
 * 7. 理解伴X显性遗传中，患病父亲可以把相关X染色体
 *    传给所有女儿，但不能直接传给儿子；
 * 8. 理解伴Y遗传只沿父系传递，父亲可以把Y染色体传给儿子，
 *    不能传给女儿；
 * 9. 比较理论概率和有限样本模拟结果，认识随机波动；
 * 10. 理解伴性遗传不等于所有性别差异都由性染色体基因决定。
 *
 * 教学边界：
 * 1. 本模型使用简化的XX—XY性染色体决定系统，
 *    不代表所有生物都采用该系统；
 * 2. 模型中的“女儿”和“儿子”只表示简化棋盘格中的
 *    XX与XY染色体组合，不讨论性别发育的复杂生物学情况；
 * 3. 伴X隐性模式默认完全外显，
 *    不讨论外显率不全、嵌合、X染色体失活偏倚或新发变异；
 * 4. 伴X显性模式同样采用简化完全外显模型；
 * 5. 伴Y遗传只用于展示Y染色体上的基因随父系传递，
 *    不代表所有男性特征都是伴Y遗传；
 * 6. 父亲把X染色体传给女儿、把Y染色体传给儿子，
 *    因此不存在父亲把X染色体直接传给儿子的路径；
 * 7. “携带者”主要用于伴X隐性杂合女性的教学表达，
 *    不能替代真实遗传检测；
 * 8. 理论比例假设各类配子形成与结合机会相等、
 *    样本足够大且没有选择、致死或生殖差异；
 * 9. 有限样本模拟可能偏离理论比例，
 *    样本量增加时通常更接近理论概率；
 * 10. 本模板中的家系和基因型均为教学构造，
 *     不对应真实患者或家庭；
 * 11. 本模型不用于疾病诊断、遗传咨询、
 *     生育决策、个人风险判断或临床解释。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式、字体、图片或CDN；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用统一.bl-*公共布局协议；
 * 5. 支持同一课件页放置多个独立实例；
 * 6. 不使用document.querySelector或document.querySelectorAll；
 * 7. 本文件只导出独立模板数组，聚合接入由第29批C1完成。
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
 * 构建完全限定在当前rootId内部的样式。
 */
function sexLinkedStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #F9A8D4;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#E0E7FF);border-bottom:1px solid #F9A8D4}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFFAFC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#DB2777}'
    + '#' + rootId + ' .sl-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .sl-modes{display:grid;grid-template-columns:repeat(4,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .sl-crosses{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .sl-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .sl-button{min-height:31px;padding:3px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:8.9px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .sl-button.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.14)}'
    + '#' + rootId + ' .sl-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .sl-auto.paused{background:#64748B}'
    + '#' + rootId + ' .sl-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .sl-toggle.off{background:#64748B}'
    + '#' + rootId + ' .sl-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .sl-card{padding:6px 3px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .sl-card b{display:block;min-height:18px;font-size:12.5px;color:#BE185D}'
    + '#' + rootId + ' .sl-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .sl-gamete{animation:' + rootId + '-gamete var(--sl-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .sl-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--sl-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .sl-highlight{animation:' + rootId + '-highlight 1.05s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-gamete{from{transform:translateY(3px);opacity:.52}to{transform:translateY(-4px);opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-highlight{from{opacity:.35}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_SEX_LINKED:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-sex-linked-inheritance',
    group: '🧬 遗传规律',
    name: '伴性遗传与性染色体传递',
    emoji: '⚥',
    desc: '比较XX—XY传递、伴X隐性、伴X显性和伴Y遗传，观察父母配子、子代性别和理论概率',
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
        defaultValue: 29,
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
      {
        key: 'showLabels',
        label: '显示传递标注',
        type: 'boolean',
        defaultValue: true,
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
        29,
      )
      const animationSpeed = num(
        params,
        'animationSpeed',
        56,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${sexLinkedStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">⚥ 伴性遗传与性染色体传递</div>
    <div class="bl-note">父亲把X传给女儿、把Y传给儿子，不存在父亲把X直接传给儿子的路径</div>
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

      <div class="sl-subtitle">遗传观察模式</div>

      <div class="sl-modes">
        <button type="button" class="sl-button active" data-mode="chromosome">XX—XY传递</button>
        <button type="button" class="sl-button" data-mode="xRecessive">伴X隐性</button>
        <button type="button" class="sl-button" data-mode="xDominant">伴X显性</button>
        <button type="button" class="sl-button" data-mode="yLinked">伴Y遗传</button>
      </div>

      <div class="sl-subtitle">选择亲本组合或观察路径</div>

      <div class="sl-crosses">
        <button type="button" class="sl-button active" data-cross="cross1"></button>
        <button type="button" class="sl-button" data-cross="cross2"></button>
        <button type="button" class="sl-button" data-cross="cross3"></button>
      </div>

      <div class="sl-subtitle">快速比较情境</div>

      <div class="sl-scenarios">
        <button type="button" class="sl-button active" data-scenario="basic">XX×XY</button>
        <button type="button" class="sl-button" data-scenario="carrierMother">携带者母亲</button>
        <button type="button" class="sl-button" data-scenario="recessiveFather">隐性患病父亲</button>
        <button type="button" class="sl-button" data-scenario="dominantFather">显性患病父亲</button>
        <button type="button" class="sl-button" data-scenario="yFather">伴Y父系传递</button>
      </div>

      <button type="button" class="sl-auto" data-auto>
        自动演示：运行中
      </button>

      <button
        type="button"
        class="sl-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '传递标注：显示' : '传递标注：隐藏'}</button>

      <div class="sl-status">
        <div class="sl-card">
          <b data-status-one></b>
          <span data-status-one-label></span>
        </div>

        <div class="sl-card">
          <b data-status-two></b>
          <span data-status-two-label></span>
        </div>

        <div class="sl-card">
          <b data-status-three></b>
          <span data-status-three-label></span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="伴性遗传与性染色体传递互动模型"
      >
        <defs>
          <marker
            id="${rootId}-arrow-pink"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#DB2777"/>
          </marker>

          <marker
            id="${rootId}-arrow-blue"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>

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

        <rect width="760" height="430" fill="#FFFFFF"/>

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
          y="61"
          data-summary
          font-size="12.5"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-parent-layer></g>
        <g data-gamete-layer></g>
        <g data-punnett-layer></g>
        <g data-stat-layer></g>
        <g data-label-layer></g>

        <text
          x="22"
          y="306"
          font-size="13"
          font-weight="900"
          fill="#334155"
        >理论概率与有限样本模拟比较</text>

        <g data-chart-layer></g>

        <g transform="translate(22 414)">
          <circle cx="7" cy="0" r="6" fill="#F472B6"/>
          <text x="20" y="5" font-size="10.5" font-weight="800" fill="#475569">
            女儿相关结果
          </text>
        </g>

        <g transform="translate(163 414)">
          <circle cx="7" cy="0" r="6" fill="#60A5FA"/>
          <text x="20" y="5" font-size="10.5" font-weight="800" fill="#475569">
            儿子相关结果
          </text>
        </g>

        <g transform="translate(304 414)">
          <rect x="1" y="-6" width="13" height="12" rx="3" fill="#CBD5E1"/>
          <text x="22" y="5" font-size="10.5" font-weight="800" fill="#475569">
            浅色：理论
          </text>
        </g>

        <g transform="translate(416 414)">
          <rect x="1" y="-6" width="13" height="12" rx="3" fill="#BE185D"/>
          <text x="22" y="5" font-size="10.5" font-weight="800" fill="#475569">
            深色：模拟
          </text>
        </g>

        <text
          x="548"
          y="419"
          data-footer-note
          font-size="10.5"
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

    var countInput=root.querySelector('[data-count]');
    var seedInput=root.querySelector('[data-seed]');
    var speedInput=root.querySelector('[data-speed]');

    var countValue=root.querySelector('[data-count-value]');
    var seedValue=root.querySelector('[data-seed-value]');
    var speedValue=root.querySelector('[data-speed-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var crossButtons=root.querySelectorAll('[data-cross]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var autoButton=root.querySelector('[data-auto]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var statusOne=root.querySelector('[data-status-one]');
    var statusTwo=root.querySelector('[data-status-two]');
    var statusThree=root.querySelector('[data-status-three]');
    var statusOneLabel=root.querySelector('[data-status-one-label]');
    var statusTwoLabel=root.querySelector('[data-status-two-label]');
    var statusThreeLabel=root.querySelector('[data-status-three-label]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var parentLayer=root.querySelector('[data-parent-layer]');
    var gameteLayer=root.querySelector('[data-gamete-layer]');
    var punnettLayer=root.querySelector('[data-punnett-layer]');
    var statLayer=root.querySelector('[data-stat-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');
    var chartLayer=root.querySelector('[data-chart-layer]');
    var footerNote=root.querySelector('[data-footer-note]');

    var mode='chromosome';
    var cross='cross1';
    var automatic=true;
    var timer=null;
    var showLabels=${showLabels ? 'true' : 'false'};

    var modeOrder=[
      'chromosome',
      'xRecessive',
      'xDominant',
      'yLinked'
    ];

    var crossOrder=[
      'cross1',
      'cross2',
      'cross3'
    ];

    var scenarios={
      basic:{
        mode:'chromosome',
        cross:'cross1'
      },
      carrierMother:{
        mode:'xRecessive',
        cross:'cross1'
      },
      recessiveFather:{
        mode:'xRecessive',
        cross:'cross2'
      },
      dominantFather:{
        mode:'xDominant',
        cross:'cross2'
      },
      yFather:{
        mode:'yLinked',
        cross:'cross1'
      }
    };

    var crossLabels={
      chromosome:[
        '完整XX×XY',
        '重点观察女儿',
        '重点观察儿子'
      ],
      xRecessive:[
        '携带者母亲×正常父亲',
        '正常母亲×患病父亲',
        '患病母亲×正常父亲'
      ],
      xDominant:[
        '杂合患病母亲×正常父亲',
        '正常母亲×患病父亲',
        '纯合患病母亲×正常父亲'
      ],
      yLinked:[
        '伴Y父亲×正常母亲',
        '正常父亲×正常母亲',
        '观察父系连续传递'
      ]
    };

    var crossData={
      chromosome:{
        cross1:{
          mother:'XX',
          father:'XY',
          maternal:['X','X'],
          paternal:['X','Y'],
          focus:'',
          title:'XX—XY性染色体传递',
          summary:'母亲的卵细胞携带X，父亲的精子可能携带X或Y',
          teaching:'父亲提供X时形成XX组合，父亲提供Y时形成XY组合。',
          footer:'父亲的X进入女儿，Y进入儿子'
        },
        cross2:{
          mother:'XX',
          father:'XY',
          maternal:['X','X'],
          paternal:['X','Y'],
          focus:'female',
          title:'条件观察：女儿获得父亲的X染色体',
          summary:'女儿的两个X染色体分别来自母亲和父亲',
          teaching:'本模式只突出已经形成女儿后的传递路径，不表示父亲只产生X精子。',
          footer:'女儿：母方X + 父方X'
        },
        cross3:{
          mother:'XX',
          father:'XY',
          maternal:['X','X'],
          paternal:['X','Y'],
          focus:'male',
          title:'条件观察：儿子获得父亲的Y染色体',
          summary:'儿子的X来自母亲，Y来自父亲',
          teaching:'父亲不会把X染色体直接传给儿子；儿子的父方性染色体是Y。',
          footer:'儿子：母方X + 父方Y'
        }
      },

      xRecessive:{
        cross1:{
          mother:'XᴬXᵃ',
          father:'XᴬY',
          maternal:['Xᴬ','Xᵃ'],
          paternal:['Xᴬ','Y'],
          focus:'',
          title:'伴X隐性：携带者母亲与正常父亲',
          summary:'母亲可形成Xᴬ和Xᵃ卵细胞，父亲形成Xᴬ和Y精子',
          teaching:'女儿可能正常或为携带者；儿子是否表现相关性状取决于从母亲获得哪一种X染色体。',
          footer:'儿子的X全部来自母亲'
        },
        cross2:{
          mother:'XᴬXᴬ',
          father:'XᵃY',
          maternal:['Xᴬ','Xᴬ'],
          paternal:['Xᵃ','Y'],
          focus:'',
          title:'伴X隐性：正常母亲与患病父亲',
          summary:'患病父亲把Xᵃ传给所有女儿，把Y传给所有儿子',
          teaching:'在这一简化组合中，女儿均为携带者，儿子不从父亲获得Xᵃ。',
          footer:'患病父亲不能把Xᵃ直接传给儿子'
        },
        cross3:{
          mother:'XᵃXᵃ',
          father:'XᴬY',
          maternal:['Xᵃ','Xᵃ'],
          paternal:['Xᴬ','Y'],
          focus:'',
          title:'伴X隐性：患病母亲与正常父亲',
          summary:'母亲形成的卵细胞都携带Xᵃ',
          teaching:'在这一简化组合中，女儿均为携带者，儿子均从母亲获得Xᵃ并表现相关性状。',
          footer:'母亲把X传给女儿和儿子'
        }
      },

      xDominant:{
        cross1:{
          mother:'XᴰXᵈ',
          father:'XᵈY',
          maternal:['Xᴰ','Xᵈ'],
          paternal:['Xᵈ','Y'],
          focus:'',
          title:'伴X显性：杂合患病母亲与正常父亲',
          summary:'母亲可能把Xᴰ或Xᵈ传给女儿和儿子',
          teaching:'在完全外显模型中，女儿和儿子都有一半机会从母亲获得Xᴰ。',
          footer:'母亲的X可以传给两种性别的子代'
        },
        cross2:{
          mother:'XᵈXᵈ',
          father:'XᴰY',
          maternal:['Xᵈ','Xᵈ'],
          paternal:['Xᴰ','Y'],
          focus:'',
          title:'伴X显性：正常母亲与患病父亲',
          summary:'父亲把Xᴰ传给所有女儿，把Y传给所有儿子',
          teaching:'在这一简化组合中，女儿均表现相关性状，儿子不从父亲获得Xᴰ。',
          footer:'患病父亲：女儿全获得Xᴰ，儿子不获得'
        },
        cross3:{
          mother:'XᴰXᴰ',
          father:'XᵈY',
          maternal:['Xᴰ','Xᴰ'],
          paternal:['Xᵈ','Y'],
          focus:'',
          title:'伴X显性：纯合患病母亲与正常父亲',
          summary:'母亲形成的卵细胞都携带Xᴰ',
          teaching:'在完全外显的简化模型中，所有女儿和儿子都从母亲获得Xᴰ。',
          footer:'母方Xᴰ可进入女儿和儿子'
        }
      },

      yLinked:{
        cross1:{
          mother:'XX',
          father:'XYᵞ',
          maternal:['X','X'],
          paternal:['X','Yᵞ'],
          focus:'',
          title:'伴Y遗传：父亲携带Y染色体相关基因',
          summary:'父亲把X传给女儿，把Yᵞ传给儿子',
          teaching:'女儿不获得父亲的Y染色体；儿子均从父亲获得Yᵞ。',
          footer:'伴Y遗传只沿父系传递'
        },
        cross2:{
          mother:'XX',
          father:'XY',
          maternal:['X','X'],
          paternal:['X','Y'],
          focus:'',
          title:'伴Y遗传对照：父亲的Y不携带示意变异',
          summary:'父亲仍把Y传给儿子，但本模型中的Y不含目标变异',
          teaching:'没有目标Y染色体变异时，子代不会表现该伴Y性状。',
          footer:'Y染色体仍沿父系传递'
        },
        cross3:{
          mother:'XX',
          father:'XYᵞ',
          maternal:['X','X'],
          paternal:['X','Yᵞ'],
          focus:'male',
          title:'伴Y遗传：父系连续传递路径',
          summary:'携带Yᵞ的父亲传给儿子，儿子成年后还可传给自己的儿子',
          teaching:'本图只展示一代棋盘格，但父子之间的Y染色体传递可在父系连续出现。',
          footer:'父亲 → 儿子 → 孙子'
        }
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function setScenarioActive(name){
      for(var i=0;i<scenarioButtons.length;i++){
        scenarioButtons[i].classList.toggle(
          'active',
          scenarioButtons[i].getAttribute('data-scenario')===name
        );
      }
    }

    function setModeActive(){
      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }
    }

    function setCrossActive(){
      var labels=crossLabels[mode];

      for(var i=0;i<crossButtons.length;i++){
        crossButtons[i].textContent=labels[i];

        crossButtons[i].classList.toggle(
          'active',
          crossButtons[i].getAttribute('data-cross')===cross
        );
      }
    }

    function createRandom(seed,count){
      var state=(
        Math.floor(seed)*2654435761
        +Math.floor(count)*131
        +modeOrder.indexOf(mode)*104729
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

    function isFemalePaternalGamete(gamete){
      return gamete.charAt(0)==='X';
    }

    function combineOutcome(
      maternalGamete,
      paternalGamete
    ){
      var female=
        isFemalePaternalGamete(
          paternalGamete
        );

      var genotype=female
        ?maternalGamete+paternalGamete
        :maternalGamete+paternalGamete;

      var affected=false;
      var carrier=false;
      var category='';
      var status='';

      if(mode==='chromosome'){
        status=female
          ?'女儿（XX）'
          :'儿子（XY）';

        category=female
          ?'femaleNormal'
          :'maleNormal';
      }else if(mode==='xRecessive'){
        if(female){
          var recessiveCount=
            (
              maternalGamete.indexOf('ᵃ')>=0
                ?1
                :0
            )
            +(
              paternalGamete.indexOf('ᵃ')>=0
                ?1
                :0
            );

          affected=
            recessiveCount===2;

          carrier=
            recessiveCount===1;

          status=affected
            ?'患病女儿'
            :carrier
              ?'携带者女儿'
              :'正常女儿';

          category=affected
            ?'femaleAffected'
            :carrier
              ?'femaleCarrier'
              :'femaleNormal';
        }else{
          affected=
            maternalGamete.indexOf('ᵃ')>=0;

          status=affected
            ?'患病儿子'
            :'正常儿子';

          category=affected
            ?'maleAffected'
            :'maleNormal';
        }
      }else if(mode==='xDominant'){
        affected=
          maternalGamete.indexOf('ᴰ')>=0
          ||paternalGamete.indexOf('ᴰ')>=0;

        status=female
          ?affected
            ?'患病女儿'
            :'正常女儿'
          :affected
            ?'患病儿子'
            :'正常儿子';

        category=female
          ?affected
            ?'femaleAffected'
            :'femaleNormal'
          :affected
            ?'maleAffected'
            :'maleNormal';
      }else{
        affected=
          !female
          &&paternalGamete.indexOf('ᵞ')>=0;

        status=female
          ?'女儿：不获得父方Y'
          :affected
            ?'儿子：获得Yᵞ'
            :'儿子：获得普通Y';

        category=female
          ?'femaleNormal'
          :affected
            ?'maleAffected'
            :'maleNormal';
      }

      return {
        maternal:maternalGamete,
        paternal:paternalGamete,
        genotype:genotype,
        female:female,
        affected:affected,
        carrier:carrier,
        category:category,
        status:status
      };
    }

    function emptyCounts(){
      return {
        femaleNormal:0,
        femaleCarrier:0,
        femaleAffected:0,
        maleNormal:0,
        maleAffected:0,
        femaleTotal:0,
        maleTotal:0
      };
    }

    function addOutcome(counts,outcome){
      counts[outcome.category]+=1;

      if(outcome.female){
        counts.femaleTotal+=1;
      }else{
        counts.maleTotal+=1;
      }
    }

    function calculateTheory(data){
      var outcomes=[];
      var counts=emptyCounts();

      for(var row=0;row<2;row++){
        for(var col=0;col<2;col++){
          var outcome=combineOutcome(
            data.maternal[col],
            data.paternal[row]
          );

          outcomes.push(outcome);
          addOutcome(counts,outcome);
        }
      }

      return {
        outcomes:outcomes,
        counts:counts
      };
    }

    function simulateOffspring(
      data,
      count,
      seed
    ){
      var random=createRandom(
        seed,
        count
      );

      var counts=emptyCounts();

      for(var i=0;i<count;i++){
        var maternalGamete=
          data.maternal[
            Math.floor(
              random()*data.maternal.length
            )
          ];

        var paternalGamete=
          data.paternal[
            Math.floor(
              random()*data.paternal.length
            )
          ];

        var outcome=combineOutcome(
          maternalGamete,
          paternalGamete
        );

        addOutcome(counts,outcome);
      }

      return counts;
    }

    function percent(
      value,
      total
    ){
      return total>0
        ?value/total*100
        :0;
    }

    function gameteColor(gamete){
      if(gamete.charAt(0)==='Y'){
        return '#2563EB';
      }

      if(
        gamete.indexOf('ᵃ')>=0
        ||gamete.indexOf('ᵈ')>=0
      ){
        return '#F9A8D4';
      }

      if(
        gamete.indexOf('ᴰ')>=0
        ||gamete.indexOf('ᵞ')>=0
      ){
        return '#DB2777';
      }

      return '#8B5CF6';
    }

    function gameteChip(
      x,
      y,
      gamete,
      radius
    ){
      var color=gameteColor(gamete);

      return ''
        +'<circle cx="'+x+'" cy="'+y
        +'" r="'+radius+'" fill="'+color
        +'" stroke="#FFFFFF" stroke-width="2.5"/>'
        +'<text x="'+x+'" y="'+(y+4)
        +'" text-anchor="middle" font-size="'
        +(radius<12?'8.5':'11')
        +'" font-weight="900" fill="#FFFFFF">'
        +gamete+'</text>';
    }

    function parentCard(
      x,
      y,
      genotype,
      label,
      female
    ){
      var symbol=female
        ?'<circle cx="'+(x+34)+'" cy="'+(y+39)
          +'" r="20" fill="#FCE7F3" stroke="#DB2777"'
          +' stroke-width="3"/>'
        :'<rect x="'+(x+14)+'" y="'+(y+19)
          +'" width="40" height="40" rx="5"'
          +' fill="#DBEAFE" stroke="#2563EB"'
          +' stroke-width="3"/>';

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="'+x+'" y="'+y+'" width="145"'
        +' height="78" rx="15" fill="#FFFFFF"'
        +' stroke="#F9A8D4" stroke-width="2.5"/>'
        +symbol
        +'<text x="'+(x+65)+'" y="'+(y+25)
        +'" font-size="11" font-weight="800" fill="#64748B">'
        +label+'</text>'
        +'<text x="'+(x+65)+'" y="'+(y+52)
        +'" font-size="19" font-weight="900" fill="'
        +(female?'#9D174D':'#1D4ED8')
        +'">'+genotype+'</text>'
        +'<text x="'+(x+65)+'" y="'+(y+68)
        +'" font-size="9.5" font-weight="800" fill="#475569">'
        +(female?'形成X型卵细胞':'形成X型或Y型精子')
        +'</text>'
        +'</g>';
    }

    function renderParents(data){
      return ''
        +parentCard(
          22,
          80,
          data.mother,
          '母亲',
          true
        )
        +'<text x="177" y="127" text-anchor="middle"'
        +' font-size="25" font-weight="900" fill="#7C3AED">×</text>'
        +parentCard(
          190,
          80,
          data.father,
          '父亲',
          false
        );
    }

    function renderGametes(data){
      var html='';

      html+='<text x="35" y="182" font-size="10.5"'
        +' font-weight="900" fill="#9D174D">母方配子</text>';

      html+='<text x="199" y="182" font-size="10.5"'
        +' font-weight="900" fill="#1D4ED8">父方配子</text>';

      for(var i=0;i<2;i++){
        html+='<g class="sl-gamete">'
          +gameteChip(
            72+i*55,
            211,
            data.maternal[i],
            16
          )
          +'</g>';

        html+='<g class="sl-gamete">'
          +gameteChip(
            236+i*55,
            211,
            data.paternal[i],
            16
          )
          +'</g>';
      }

      html+='<path class="sl-flow"'
        +' d="M150 211 C218 234 270 237 325 250"'
        +' fill="none" stroke="#DB2777"'
        +' stroke-width="3"'
        +' marker-end="url(#${rootId}-arrow-pink)"/>';

      html+='<path class="sl-flow"'
        +' d="M314 211 C344 224 355 232 368 246"'
        +' fill="none" stroke="#2563EB"'
        +' stroke-width="3"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>';

      return html;
    }

    function outcomeSymbol(
      x,
      y,
      outcome,
      focused
    ){
      var stroke=outcome.female
        ?'#DB2777'
        :'#2563EB';

      var fill=outcome.affected
        ?'#334155'
        :outcome.carrier
          ?'#FCE7F3'
          :outcome.female
            ?'#FFFFFF'
            :'#EFF6FF';

      var opacity=focused
        ?1
        :.34;

      var html='<g opacity="'+opacity+'"'
        +(focused?' class="sl-highlight"':'')
        +'>';

      if(outcome.female){
        html+='<circle cx="'+x+'" cy="'+y+'" r="18"'
          +' fill="'+fill+'" stroke="'+stroke+'"'
          +' stroke-width="3"/>';

        if(outcome.carrier){
          html+='<circle cx="'+x+'" cy="'+y+'" r="5"'
            +' fill="#F59E0B" stroke="#B45309"'
            +' stroke-width="1.5"/>';
        }
      }else{
        html+='<rect x="'+(x-18)+'" y="'+(y-18)
          +'" width="36" height="36" rx="4"'
          +' fill="'+fill+'" stroke="'+stroke+'"'
          +' stroke-width="3"/>';
      }

      html+='<text x="'+x+'" y="'+(y+34)
        +'" text-anchor="middle" font-size="8.7"'
        +' font-weight="900" fill="#475569">'
        +outcome.genotype+'</text>';

      html+='<text x="'+x+'" y="'+(y+47)
        +'" text-anchor="middle" font-size="8.5"'
        +' font-weight="800" fill="'
        +(outcome.affected?'#B91C1C':'#64748B')
        +'">'+outcome.status+'</text>';

      html+='</g>';

      return html;
    }

    function renderPunnett(
      data,
      theory
    ){
      var startX=338;
      var startY=91;
      var cell=61;
      var html='';

      html+='<text x="'+(startX+cell*1.5)
        +'" y="80" text-anchor="middle"'
        +' font-size="11.5" font-weight="900" fill="#5B21B6">'
        +'2×2棋盘格'
        +'</text>';

      html+='<rect x="'+startX+'" y="'+startY
        +'" width="'+(cell*3)+'" height="'+(cell*3)
        +'" rx="11" fill="#FFFFFF"'
        +' stroke="#C4B5FD" stroke-width="3"/>';

      for(var line=1;line<3;line++){
        html+='<line x1="'+(startX+cell*line)
          +'" y1="'+startY
          +'" x2="'+(startX+cell*line)
          +'" y2="'+(startY+cell*3)
          +'" stroke="#DDD6FE" stroke-width="2"/>';

        html+='<line x1="'+startX
          +'" y1="'+(startY+cell*line)
          +'" x2="'+(startX+cell*3)
          +'" y2="'+(startY+cell*line)
          +'" stroke="#DDD6FE" stroke-width="2"/>';
      }

      html+='<text x="'+(startX+cell/2)
        +'" y="'+(startY+cell/2+4)
        +'" text-anchor="middle" font-size="10"'
        +' font-weight="900" fill="#64748B">×</text>';

      for(var col=0;col<2;col++){
        html+=gameteChip(
          startX+cell*(col+1)+cell/2,
          startY+cell/2,
          data.maternal[col],
          13
        );
      }

      for(var row=0;row<2;row++){
        html+=gameteChip(
          startX+cell/2,
          startY+cell*(row+1)+cell/2,
          data.paternal[row],
          13
        );

        for(var innerCol=0;innerCol<2;innerCol++){
          var outcome=
            theory.outcomes[
              row*2+innerCol
            ];

          var focused=
            !data.focus
            ||(
              data.focus==='female'
              &&outcome.female
            )
            ||(
              data.focus==='male'
              &&!outcome.female
            );

          var cx=
            startX
            +cell*(innerCol+1)
            +cell/2;

          var cy=
            startY
            +cell*(row+1)
            +cell/2-8;

          html+=outcomeSymbol(
            cx,
            cy,
            outcome,
            focused
          );
        }
      }

      return html;
    }

    function renderTopStats(
      theory
    ){
      var counts=theory.counts;
      var daughterAffected=percent(
        counts.femaleAffected,
        counts.femaleTotal
      );
      var daughterCarrier=percent(
        counts.femaleCarrier,
        counts.femaleTotal
      );
      var sonAffected=percent(
        counts.maleAffected,
        counts.maleTotal
      );

      var html='<g transform="translate(548 82)">';

      html+='<rect x="0" y="0" width="190" height="190"'
        +' rx="17" fill="#FFFAFC"'
        +' stroke="#F9A8D4" stroke-width="2.5"/>';

      html+='<text x="95" y="25" text-anchor="middle"'
        +' font-size="12.5" font-weight="900" fill="#9D174D">'
        +'理论传递结果'
        +'</text>';

      if(mode==='chromosome'){
        html+='<text x="13" y="54" font-size="10.5"'
          +' font-weight="900" fill="#DB2777">'
          +'XX组合 '
          +percent(
            counts.femaleTotal,
            4
          ).toFixed(0)
          +'%'
          +'</text>';

        html+='<text x="13" y="83" font-size="10.5"'
          +' font-weight="900" fill="#2563EB">'
          +'XY组合 '
          +percent(
            counts.maleTotal,
            4
          ).toFixed(0)
          +'%'
          +'</text>';

        html+='<text x="13" y="116" font-size="10.5"'
          +' font-weight="800" fill="#475569">'
          +'女儿：母方X + 父方X'
          +'</text>';

        html+='<text x="13" y="143" font-size="10.5"'
          +' font-weight="800" fill="#475569">'
          +'儿子：母方X + 父方Y'
          +'</text>';

        html+='<text x="95" y="174" text-anchor="middle"'
          +' font-size="10.5" font-weight="900" fill="#5B21B6">'
          +'父方配子决定XX或XY组合'
          +'</text>';
      }else{
        var items=[
          {
            label:'患病女儿',
            value:daughterAffected,
            color:'#DB2777'
          },
          {
            label:'携带者女儿',
            value:daughterCarrier,
            color:'#F59E0B'
          },
          {
            label:'患病儿子',
            value:sonAffected,
            color:'#2563EB'
          }
        ];

        for(var i=0;i<items.length;i++){
          var item=items[i];
          var y=49+i*45;

          html+='<text x="13" y="'+y
            +'" font-size="10" font-weight="900"'
            +' fill="'+item.color+'">'
            +item.label+' '
            +item.value.toFixed(0)+'%'
            +'</text>';

          html+='<rect x="13" y="'+(y+8)
            +'" width="164" height="11" rx="5.5"'
            +' fill="#E2E8F0"/>';

          html+='<rect x="13" y="'+(y+8)
            +'" width="'+(164*item.value/100)
            +'" height="11" rx="5.5"'
            +' fill="'+item.color+'"/>';
        }

        html+='<text x="95" y="178" text-anchor="middle"'
          +' font-size="9.5" font-weight="900" fill="#5B21B6">'
          +(mode==='yLinked'
            ?'女儿不获得父亲的Y染色体'
            :'比例按女儿或儿子分别计算')
          +'</text>';
      }

      html+='</g>';

      return html;
    }

    function chartMeta(){
      if(mode==='xRecessive'){
        return [
          {
            key:'femaleNormal',
            label:'正常女儿',
            color:'#F9A8D4'
          },
          {
            key:'femaleCarrier',
            label:'携带者女儿',
            color:'#F59E0B'
          },
          {
            key:'maleNormal',
            label:'正常儿子',
            color:'#93C5FD'
          },
          {
            key:'maleAffected',
            label:'患病儿子',
            color:'#2563EB'
          }
        ];
      }

      if(mode==='xDominant'){
        return [
          {
            key:'femaleNormal',
            label:'正常女儿',
            color:'#F9A8D4'
          },
          {
            key:'femaleAffected',
            label:'患病女儿',
            color:'#DB2777'
          },
          {
            key:'maleNormal',
            label:'正常儿子',
            color:'#93C5FD'
          },
          {
            key:'maleAffected',
            label:'患病儿子',
            color:'#2563EB'
          }
        ];
      }

      if(mode==='yLinked'){
        return [
          {
            key:'femaleNormal',
            label:'女儿',
            color:'#F472B6'
          },
          {
            key:'femaleAffected',
            label:'伴Y女儿',
            color:'#DB2777'
          },
          {
            key:'maleNormal',
            label:'普通Y儿子',
            color:'#93C5FD'
          },
          {
            key:'maleAffected',
            label:'Yᵞ儿子',
            color:'#2563EB'
          }
        ];
      }

      return [
        {
          key:'femaleNormal',
          label:'XX组合',
          color:'#F472B6'
        },
        {
          key:'femaleCarrier',
          label:'其他女儿',
          color:'#F9A8D4'
        },
        {
          key:'maleNormal',
          label:'XY组合',
          color:'#60A5FA'
        },
        {
          key:'maleAffected',
          label:'其他儿子',
          color:'#93C5FD'
        }
      ];
    }

    function renderChart(
      theoryCounts,
      simulatedCounts,
      total
    ){
      var meta=chartMeta();
      var html='';
      var baseline=394;
      var top=326;
      var chartHeight=65;

      html+='<line x1="55" y1="'+baseline
        +'" x2="720" y2="'+baseline
        +'" stroke="#64748B" stroke-width="2"/>';

      for(var i=0;i<meta.length;i++){
        var item=meta[i];
        var x=105+i*165;
        var theoryPercent=
          theoryCounts[item.key]/4*100;
        var simulatedPercent=
          simulatedCounts[item.key]/total*100;

        var theoryHeight=
          chartHeight*theoryPercent/100;

        var simulatedHeight=
          chartHeight*simulatedPercent/100;

        html+='<rect x="'+(x-28)
          +'" y="'+(baseline-theoryHeight)
          +'" width="24" height="'+theoryHeight
          +'" rx="4" fill="'+item.color
          +'" opacity=".34"/>';

        html+='<rect x="'+(x+4)
          +'" y="'+(baseline-simulatedHeight)
          +'" width="24" height="'+simulatedHeight
          +'" rx="4" fill="'+item.color+'"/>';

        html+='<text x="'+x+'" y="'+(baseline+15)
          +'" text-anchor="middle" font-size="9"'
          +' font-weight="900" fill="#475569">'
          +item.label+'</text>';

        html+='<text x="'+x+'" y="'+(top-3)
          +'" text-anchor="middle" font-size="8.7"'
          +' font-weight="800" fill="#64748B">'
          +'理 '+theoryPercent.toFixed(0)
          +'% / 模 '+simulatedPercent.toFixed(1)
          +'%'
          +'</text>';
      }

      return html;
    }

    function updateStatus(theory){
      var counts=theory.counts;

      if(mode==='chromosome'){
        statusOne.textContent=
          percent(
            counts.femaleTotal,
            4
          ).toFixed(0)+'%';

        statusTwo.textContent=
          percent(
            counts.maleTotal,
            4
          ).toFixed(0)+'%';

        statusThree.textContent='0%';

        statusOneLabel.textContent='XX组合';
        statusTwoLabel.textContent='XY组合';
        statusThreeLabel.textContent='父亲X直传儿子';
        return;
      }

      var daughterAffected=percent(
        counts.femaleAffected,
        counts.femaleTotal
      );

      var sonAffected=percent(
        counts.maleAffected,
        counts.maleTotal
      );

      var daughterCarrier=percent(
        counts.femaleCarrier,
        counts.femaleTotal
      );

      statusOne.textContent=
        daughterAffected.toFixed(0)+'%';

      statusTwo.textContent=
        sonAffected.toFixed(0)+'%';

      statusOneLabel.textContent=
        '女儿表现比例';

      statusTwoLabel.textContent=
        '儿子表现比例';

      if(mode==='xRecessive'){
        statusThree.textContent=
          daughterCarrier.toFixed(0)+'%';

        statusThreeLabel.textContent=
          '女儿携带比例';
      }else if(mode==='xDominant'){
        statusThree.textContent='0%';
        statusThreeLabel.textContent=
          '父亲X直传儿子';
      }else{
        statusThree.textContent=
          sonAffected.toFixed(0)+'%';

        statusThreeLabel.textContent=
          '儿子获得Yᵞ';
      }
    }

    function renderLabels(data){
      if(!showLabels){
        return '';
      }

      var html='';

      html+='<text x="182" y="259"'
        +' text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#9D174D">'
        +'母亲的配子只提供X型性染色体'
        +'</text>';

      html+='<text x="635" y="291"'
        +' text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#1D4ED8">'
        +(mode==='yLinked'
          ?'Y染色体只由父亲传给儿子'
          :'父亲的X进入女儿，Y进入儿子')
        +'</text>';

      if(data.focus==='female'){
        html+='<path class="sl-flow"'
          +' d="M503 126 C548 95 592 91 632 109"'
          +' fill="none" stroke="#DB2777"'
          +' stroke-width="3"'
          +' marker-end="url(#${rootId}-arrow-pink)"/>';
      }else if(data.focus==='male'){
        html+='<path class="sl-flow"'
          +' d="M503 221 C550 249 592 253 631 232"'
          +' fill="none" stroke="#2563EB"'
          +' stroke-width="3"'
          +' marker-end="url(#${rootId}-arrow-blue)"/>';
      }

      return html;
    }

    function update(){
      var count=Math.round(
        Number(countInput.value)
      );

      var seed=Math.round(
        Number(seedInput.value)
      );

      var speed=Number(
        speedInput.value
      );

      var data=
        crossData[mode][cross];

      var theory=
        calculateTheory(data);

      var simulated=
        simulateOffspring(
          data,
          count,
          seed
        );

      countValue.textContent=
        count+' 个';

      seedValue.textContent=
        '第 '+seed+' 组';

      speedValue.textContent=
        speed.toFixed(0)+'%';

      labelToggle.textContent=
        showLabels
          ?'传递标注：显示'
          :'传递标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      autoButton.textContent=
        automatic
          ?'自动演示：运行中'
          :'自动演示：已暂停';

      autoButton.classList.toggle(
        'paused',
        !automatic
      );

      root.style.setProperty(
        '--sl-speed',
        clamp(
          2.6-speed/58,
          .58,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--sl-flow-speed',
        clamp(
          2.5-speed/60,
          .52,
          2.4
        ).toFixed(2)+'s'
      );

      setModeActive();
      setCrossActive();

      title.textContent=
        data.title;

      summary.textContent=
        data.summary;

      parentLayer.innerHTML=
        renderParents(data);

      gameteLayer.innerHTML=
        renderGametes(data);

      punnettLayer.innerHTML=
        renderPunnett(
          data,
          theory
        );

      statLayer.innerHTML=
        renderTopStats(theory);

      labelLayer.innerHTML=
        renderLabels(data);

      chartLayer.innerHTML=
        renderChart(
          theory.counts,
          simulated,
          count
        );

      footerNote.textContent=
        data.footer;

      updateStatus(theory);

      var deviation=0;
      var keys=[
        'femaleNormal',
        'femaleCarrier',
        'femaleAffected',
        'maleNormal',
        'maleAffected'
      ];

      for(var i=0;i<keys.length;i++){
        var key=keys[i];
        var theoryFrequency=
          theory.counts[key]/4;

        var simulatedFrequency=
          simulated[key]/count;

        deviation=Math.max(
          deviation,
          Math.abs(
            theoryFrequency
            -simulatedFrequency
          )
        );
      }

      var sampleNote='';

      if(count<=100){
        sampleNote=
          '当前样本量较小，模拟比例出现明显随机波动是正常现象。';
      }else if(deviation>.08){
        sampleNote=
          '本组有限样本与理论概率仍有一定偏差，可改变实验编号或增大样本量。';
      }else{
        sampleNote=
          '当前模拟结果已较接近理论概率，但有限样本不会保证严格相等。';
      }

      var boundaryNote='';

      if(mode==='chromosome'){
        boundaryNote=
          '父亲把X传给女儿，把Y传给儿子，因此父亲不会把X染色体直接传给儿子。';
      }else if(mode==='xRecessive'){
        boundaryNote=
          '伴X隐性杂合女性可作为携带者；男性只有一条X染色体，是否表现相关性状取决于其母方X。';
      }else if(mode==='xDominant'){
        boundaryNote=
          '伴X显性患病父亲可把Xᴰ传给所有女儿，但不能直接传给儿子。';
      }else{
        boundaryNote=
          '伴Y遗传只沿父系传递，父亲可以把Y传给儿子，不能传给女儿。';
      }

      result.innerHTML=
        data.teaching
        +'<br>'+boundaryNote
        +' '+sampleNote
        +' 本模型采用完全外显和配子机会相等的简化假设，不用于真实家庭风险判断。';
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(
        !automatic
        ||!document.body.contains(root)
      ){
        return;
      }

      var speed=Number(
        speedInput.value
      );

      var interval=clamp(
        4500-speed*29,
        1250,
        4000
      );

      timer=window.setTimeout(function(){
        var crossIndex=
          crossOrder.indexOf(cross);

        if(crossIndex<crossOrder.length-1){
          cross=
            crossOrder[crossIndex+1];
        }else{
          cross='cross1';

          var modeIndex=
            modeOrder.indexOf(mode);

          mode=
            modeOrder[
              (modeIndex+1)%modeOrder.length
            ];
        }

        seedInput.value=String(
          Number(seedInput.value)>=99
            ?1
            :Number(seedInput.value)+1
        );

        setScenarioActive('');
        update();
        schedule();
      },interval);
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        automatic=false;
        mode=this.getAttribute(
          'data-mode'
        );
        cross='cross1';

        setScenarioActive('');
        update();
        schedule();
      };
    }

    for(var j=0;j<crossButtons.length;j++){
      crossButtons[j].onclick=function(){
        automatic=false;
        cross=this.getAttribute(
          'data-cross'
        );

        setScenarioActive('');
        update();
        schedule();
      };
    }

    for(var k=0;k<scenarioButtons.length;k++){
      scenarioButtons[k].onclick=function(){
        var name=this.getAttribute(
          'data-scenario'
        );

        var data=scenarios[name];

        if(!data){
          return;
        }

        automatic=false;
        mode=data.mode;
        cross=data.cross;

        setScenarioActive(name);
        update();
        schedule();
      };
    }

    autoButton.onclick=function(){
      automatic=!automatic;
      update();
      schedule();
    };

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    countInput.oninput=function(){
      setScenarioActive('');
      update();
    };

    seedInput.oninput=function(){
      setScenarioActive('');
      update();
    };

    speedInput.oninput=function(){
      setScenarioActive('');
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
