/**
 * lifeScienceLabTemplatesInquiryExperimentalDesign.ts
 *
 * 平面生命科学实验室：
 * 实验设计、对照、重复与变量控制。
 *
 * 教学目标：
 * 1. 区分研究问题、假设、自变量、因变量和控制变量；
 * 2. 理解实验组接受自变量处理，对照组提供比较基准；
 * 3. 理解除自变量外，其他可能影响结果的条件应尽量保持一致；
 * 4. 理解随机分组有助于减少初始差异造成的系统偏倚；
 * 5. 理解设置多个重复能够观察随机波动并提高均值的稳定性；
 * 6. 比较单次测量与多次重复的差异；
 * 7. 使用均值、标准差和误差棒描述数据的集中趋势与离散程度；
 * 8. 理解增加重复通常能够减小随机误差对均值的影响，
 *    但不能自动消除系统误差；
 * 9. 理解无对照组、同时改变多个变量、非随机分组
 *    都会削弱因果解释；
 * 10. 根据实验设计质量和数据重叠程度判断结论强度；
 * 11. 区分“观察到差异”和“证明自变量导致差异”。
 *
 * 教学边界：
 * 1. 本模型以假想植物生长实验为载体，
 *    但实验设计原则可迁移到其他生命科学探究；
 * 2. 所有处理强度、测量值、均值、标准差和结论强度
 *    均为相对教学指标；
 * 3. 对照组不是“什么都不做”，
 *    而是除研究自变量外尽量与实验组保持相同条件；
 * 4. 重复次数增加通常能使样本均值更稳定，
 *    但不能保证每次结果完全相同；
 * 5. 重复不能修复错误测量工具、错误操作、
 *    非随机分组或混杂变量造成的系统偏倚；
 * 6. 两组均值存在差异不等于已经证明因果关系；
 * 7. 本模型使用标准差表示离散程度，
 *    不展开标准误、置信区间和显著性检验；
 * 8. 本模型不提供动物实验、人体实验、
 *    临床试验或生物安全实验方案；
 * 9. 本模型不用于医学诊断、药物效果判断、
 *    人体健康决策或真实科研统计结论。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式、字体、图片或CDN；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用统一.bl-*公共布局协议；
 * 5. 支持同一课件页放置多个独立实例；
 * 6. 不使用document.querySelector或document.querySelectorAll；
 * 7. 本文件只导出独立模板数组，不修改聚合入口。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

/** 安全读取数值参数。 */
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

/** 安全读取布尔参数。 */
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

/** 把数值转换为适合写入HTML属性的短字符串。 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/** 构建完全限定在当前rootId内部的样式。 */
function experimentalDesignStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FCD34D;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FEF3C7,#ECFDF5);border-bottom:1px solid #FCD34D}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#92400E}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:258px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFFCF5;border-right:1px solid #FDE68A}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#D97706;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#D97706}'
    + '#' + rootId + ' .ed-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#92400E}'
    + '#' + rootId + ' .ed-stages{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ed-designs{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ed-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .ed-button{min-height:31px;padding:3px;border:1px solid #FCD34D;border-radius:8px;background:#fff;color:#92400E;font-size:8.7px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .ed-button.active{border-color:#D97706;background:#FEF3C7;box-shadow:0 3px 9px rgba(217,119,6,.14)}'
    + '#' + rootId + ' .ed-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#FBBF24,#D97706);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ed-auto.paused{background:#64748B}'
    + '#' + rootId + ' .ed-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ed-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ed-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .ed-card{padding:6px 3px;border:1px solid #FDE68A;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ed-card b{display:block;min-height:18px;font-size:12.5px;color:#B45309}'
    + '#' + rootId + ' .ed-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FEF3C7;color:#78350F;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ed-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--ed-flow-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ed-plant{animation:' + rootId + '-plant 1.45s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ed-point{animation:' + rootId + '-point 1.05s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ed-warning{animation:' + rootId + '-warning .9s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-plant{from{transform:translateY(3px);opacity:.58}to{transform:translateY(-3px);opacity:1}}'
    + '@keyframes ' + rootId + '-point{from{opacity:.48}to{opacity:1}}'
    + '@keyframes ' + rootId + '-warning{from{opacity:.35}to{opacity:1}}'
    + '</style>'
}

/** 避免在外层模板字符串中直接写出脚本结束标签。 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_EXPERIMENTAL_DESIGN:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-experimental-design-control-repeat',
    group: '🧪 实验探究',
    name: '实验设计、对照与重复',
    emoji: '📋',
    desc: '设置对照组、单一变量、随机分组和重复次数，比较均值、离散程度、随机误差与系统偏倚',
    params: [
      {
        key: 'treatmentStrength',
        label: '自变量处理强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'repeatCount',
        label: '每组重复数量',
        type: 'number',
        min: 1,
        max: 12,
        step: 1,
        defaultValue: 6,
      },
      {
        key: 'randomVariation',
        label: '个体随机差异',
        type: 'number',
        min: 0,
        max: 40,
        step: 1,
        defaultValue: 12,
      },
      {
        key: 'confoundingLevel',
        label: '混杂变量影响',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 8,
      },
      {
        key: 'randomSeed',
        label: '随机实验编号',
        type: 'number',
        min: 1,
        max: 99,
        step: 1,
        defaultValue: 27,
      },
      {
        key: 'showLabels',
        label: '显示变量标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const treatmentStrength = num(
        params,
        'treatmentStrength',
        68,
      )
      const repeatCount = num(
        params,
        'repeatCount',
        6,
      )
      const randomVariation = num(
        params,
        'randomVariation',
        12,
      )
      const confoundingLevel = num(
        params,
        'confoundingLevel',
        8,
      )
      const randomSeed = num(
        params,
        'randomSeed',
        27,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${experimentalDesignStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">📋 实验设计、对照与重复</div>
    <div class="bl-note">增加重复可减小随机误差，但不能消除混杂变量和系统偏倚</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>自变量处理强度</span>
          <span class="bl-value" data-treatment-value></span>
        </div>
        <input
          data-treatment
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(treatmentStrength)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>每组重复数量</span>
          <span class="bl-value" data-repeat-value></span>
        </div>
        <input
          data-repeat
          type="range"
          min="1"
          max="12"
          step="1"
          value="${n(repeatCount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>个体随机差异</span>
          <span class="bl-value" data-variation-value></span>
        </div>
        <input
          data-variation
          type="range"
          min="0"
          max="40"
          step="1"
          value="${n(randomVariation)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>混杂变量影响</span>
          <span class="bl-value" data-confound-value></span>
        </div>
        <input
          data-confound
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(confoundingLevel)}"
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

      <div class="ed-subtitle">实验探究阶段</div>

      <div class="ed-stages">
        <button
          type="button"
          class="ed-button active"
          data-stage="question"
        >1. 提出问题</button>

        <button
          type="button"
          class="ed-button"
          data-stage="design"
        >2. 设计实验</button>

        <button
          type="button"
          class="ed-button"
          data-stage="measure"
        >3. 重复测量</button>

        <button
          type="button"
          class="ed-button"
          data-stage="conclusion"
        >4. 解释结论</button>
      </div>

      <div class="ed-subtitle">设计质量开关</div>

      <div class="ed-designs">
        <button
          type="button"
          class="ed-button active"
          data-design="control"
        >对照组：有</button>

        <button
          type="button"
          class="ed-button active"
          data-design="single"
        >单一变量：是</button>

        <button
          type="button"
          class="ed-button active"
          data-design="random"
        >随机分组：是</button>
      </div>

      <div class="ed-subtitle">快速比较情境</div>

      <div class="ed-scenarios">
        <button
          type="button"
          class="ed-button active"
          data-scenario="standard"
        >规范设计</button>

        <button
          type="button"
          class="ed-button"
          data-scenario="noControl"
        >缺少对照</button>

        <button
          type="button"
          class="ed-button"
          data-scenario="singleRepeat"
        >仅一次测量</button>

        <button
          type="button"
          class="ed-button"
          data-scenario="confounded"
        >多个变量变化</button>

        <button
          type="button"
          class="ed-button"
          data-scenario="nonRandom"
        >非随机分组</button>
      </div>

      <button
        type="button"
        class="ed-auto"
        data-auto
      >自动演示：运行中</button>

      <button
        type="button"
        class="ed-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '变量标注：显示' : '变量标注：隐藏'}</button>

      <div class="ed-status">
        <div class="ed-card">
          <b data-status-score></b>
          <span>实验设计质量</span>
        </div>

        <div class="ed-card">
          <b data-status-difference></b>
          <span>两组均值差</span>
        </div>

        <div class="ed-card">
          <b data-status-strength></b>
          <span>结论支持程度</span>
        </div>
      </div>

      <div
        class="bl-result"
        data-result
      ></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="实验设计、对照与重复互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-control-panel"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#EFF6FF"/>
            <stop offset="100%" stop-color="#F8FAFC"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-experiment-panel"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#ECFDF5"/>
            <stop offset="100%" stop-color="#F0FDF4"/>
          </linearGradient>

          <marker
            id="${rootId}-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path
              d="M0,0 L0,6 L8,3 z"
              fill="#D97706"
            />
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#78350F"
              flood-opacity=".13"
            />
          </filter>
        </defs>

        <rect
          width="760"
          height="430"
          fill="#FFFFFF"
        />

        <text
          x="22"
          y="34"
          data-title
          font-size="24"
          font-weight="900"
          fill="#92400E"
        ></text>

        <text
          x="22"
          y="61"
          data-summary
          font-size="12.5"
          font-weight="800"
          fill="#475569"
        ></text>

        <g filter="url(#${rootId}-shadow)">
          <rect
            x="22"
            y="79"
            width="350"
            height="205"
            rx="20"
            fill="url(#${rootId}-control-panel)"
            stroke="#93C5FD"
            stroke-width="3"
          />

          <rect
            x="390"
            y="79"
            width="348"
            height="205"
            rx="20"
            fill="url(#${rootId}-experiment-panel)"
            stroke="#86EFAC"
            stroke-width="3"
          />
        </g>

        <text
          x="197"
          y="104"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#1D4ED8"
        >对照组</text>

        <text
          x="564"
          y="104"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#15803D"
        >实验组</text>

        <g data-group-layer></g>
        <g data-label-layer></g>
        <g data-warning-layer></g>

        <text
          x="22"
          y="309"
          data-chart-title
          font-size="13"
          font-weight="900"
          fill="#334155"
        ></text>

        <g data-chart-layer></g>

        <text
          x="22"
          y="421"
          data-footer-note
          font-size="10.5"
          font-weight="900"
          fill="#78350F"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');

    if(!root){
      return;
    }

    var treatmentInput=root.querySelector(
      '[data-treatment]'
    );
    var repeatInput=root.querySelector(
      '[data-repeat]'
    );
    var variationInput=root.querySelector(
      '[data-variation]'
    );
    var confoundInput=root.querySelector(
      '[data-confound]'
    );
    var seedInput=root.querySelector(
      '[data-seed]'
    );

    var treatmentValue=root.querySelector(
      '[data-treatment-value]'
    );
    var repeatValue=root.querySelector(
      '[data-repeat-value]'
    );
    var variationValue=root.querySelector(
      '[data-variation-value]'
    );
    var confoundValue=root.querySelector(
      '[data-confound-value]'
    );
    var seedValue=root.querySelector(
      '[data-seed-value]'
    );

    var stageButtons=root.querySelectorAll(
      '[data-stage]'
    );
    var designButtons=root.querySelectorAll(
      '[data-design]'
    );
    var scenarioButtons=root.querySelectorAll(
      '[data-scenario]'
    );

    var autoButton=root.querySelector(
      '[data-auto]'
    );
    var labelToggle=root.querySelector(
      '[data-label-toggle]'
    );

    var statusScore=root.querySelector(
      '[data-status-score]'
    );
    var statusDifference=root.querySelector(
      '[data-status-difference]'
    );
    var statusStrength=root.querySelector(
      '[data-status-strength]'
    );
    var result=root.querySelector(
      '[data-result]'
    );

    var title=root.querySelector(
      '[data-title]'
    );
    var summary=root.querySelector(
      '[data-summary]'
    );
    var groupLayer=root.querySelector(
      '[data-group-layer]'
    );
    var labelLayer=root.querySelector(
      '[data-label-layer]'
    );
    var warningLayer=root.querySelector(
      '[data-warning-layer]'
    );
    var chartTitle=root.querySelector(
      '[data-chart-title]'
    );
    var chartLayer=root.querySelector(
      '[data-chart-layer]'
    );
    var footerNote=root.querySelector(
      '[data-footer-note]'
    );

    var stages=[
      'question',
      'design',
      'measure',
      'conclusion'
    ];

    var stage='question';
    var automatic=true;
    var timer=null;
    var showLabels=${showLabels ? 'true' : 'false'};

    var hasControl=true;
    var singleVariable=true;
    var randomAssignment=true;
    var currentScenario='standard';

    var stageInfo={
      question:{
        title:'阶段1：提出可检验的问题',
        summary:'研究不同光照处理是否会影响同种幼苗在相同时间内的生长量',
        note:'科学问题应明确自变量、观察对象、因变量和比较条件，并能够通过数据检验。'
      },
      design:{
        title:'阶段2：设置实验组、对照组与控制变量',
        summary:'实验组改变光照处理，对照组提供基准，其他条件尽量保持一致',
        note:'对照组不是简单地“不处理”，而是除研究自变量外尽量与实验组保持相同条件。'
      },
      measure:{
        title:'阶段3：设置重复并记录测量值',
        summary:'同一处理设置多个独立重复，比较个体差异、均值和离散程度',
        note:'重复能够显示随机波动，并使样本均值通常比单次测量更稳定。'
      },
      conclusion:{
        title:'阶段4：根据设计质量和数据作出有限结论',
        summary:'同时检查均值差异、组内波动、对照设置、混杂变量和分组方式',
        note:'观察到差异不等于已经证明因果关系，结论强度必须与实验设计质量相匹配。'
      }
    };

    var scenarios={
      standard:{
        control:true,
        single:true,
        random:true,
        repeat:6,
        variation:12,
        confound:8
      },
      noControl:{
        control:false,
        single:true,
        random:true,
        repeat:6,
        variation:12,
        confound:8
      },
      singleRepeat:{
        control:true,
        single:true,
        random:true,
        repeat:1,
        variation:18,
        confound:8
      },
      confounded:{
        control:true,
        single:false,
        random:true,
        repeat:6,
        variation:12,
        confound:72
      },
      nonRandom:{
        control:true,
        single:true,
        random:false,
        repeat:6,
        variation:12,
        confound:42
      }
    };

    function clamp(
      value,
      min,
      max
    ){
      return Math.max(
        min,
        Math.min(
          max,
          value
        )
      );
    }

    function setActive(
      nodes,
      attribute,
      value
    ){
      for(var i=0;i<nodes.length;i++){
        nodes[i].classList.toggle(
          'active',
          nodes[i].getAttribute(attribute)===value
        );
      }
    }

    function setScenarioActive(name){
      setActive(
        scenarioButtons,
        'data-scenario',
        name
      );
    }

    function createRandom(
      seed,
      offset
    ){
      var state=(
        Math.floor(seed)*2654435761
        +Math.floor(offset)*2246822519
        +1013904223
      )>>>0;

      return function(){
        state=(
          Math.imul(
            state,
            1664525
          )
          +1013904223
        )>>>0;

        return state/4294967296;
      };
    }

    function mean(values){
      if(values.length===0){
        return 0;
      }

      var total=0;

      for(var i=0;i<values.length;i++){
        total+=values[i];
      }

      return total/values.length;
    }

    function standardDeviation(
      values,
      average
    ){
      if(values.length<=1){
        return 0;
      }

      var total=0;

      for(var i=0;i<values.length;i++){
        var difference=
          values[i]-average;

        total+=
          difference*difference;
      }

      return Math.sqrt(
        total/(values.length-1)
      );
    }

    function buildExperimentData(
      treatment,
      repeats,
      variation,
      confound,
      seed
    ){
      var controlRandom=
        createRandom(
          seed,
          11
        );

      var experimentRandom=
        createRandom(
          seed,
          37
        );

      var controlBase=
        randomAssignment
          ?52
          :46;

      var treatmentEffect=
        treatment*.28;

      var systematicBias=
        randomAssignment
          ?0
          :confound*.17;

      var extraVariableEffect=
        singleVariable
          ?0
          :confound*.22;

      var controlValues=[];
      var experimentValues=[];

      for(var i=0;i<repeats;i++){
        var controlNoise=
          (
            controlRandom()
            +controlRandom()
            +controlRandom()
            -1.5
          )
          *variation;

        var experimentNoise=
          (
            experimentRandom()
            +experimentRandom()
            +experimentRandom()
            -1.5
          )
          *variation;

        controlValues.push(
          clamp(
            controlBase+controlNoise,
            5,
            100
          )
        );

        experimentValues.push(
          clamp(
            controlBase
            +treatmentEffect
            +systematicBias
            +extraVariableEffect
            +experimentNoise,
            5,
            100
          )
        );
      }

      var controlMean=
        mean(controlValues);

      var experimentMean=
        mean(experimentValues);

      var controlSd=
        standardDeviation(
          controlValues,
          controlMean
        );

      var experimentSd=
        standardDeviation(
          experimentValues,
          experimentMean
        );

      return {
        controlValues:controlValues,
        experimentValues:experimentValues,
        controlMean:controlMean,
        experimentMean:experimentMean,
        controlSd:controlSd,
        experimentSd:experimentSd,
        difference:
          experimentMean-controlMean,
        treatmentEffect:treatmentEffect,
        systematicBias:systematicBias,
        extraVariableEffect:extraVariableEffect
      };
    }

    function designScore(
      repeats
    ){
      var score=100;

      if(!hasControl){
        score-=34;
      }

      if(!singleVariable){
        score-=28;
      }

      if(!randomAssignment){
        score-=20;
      }

      if(repeats<=1){
        score-=22;
      }else if(repeats<4){
        score-=10;
      }

      return clamp(
        score,
        0,
        100
      );
    }

    function conclusionStrength(
      score,
      data
    ){
      if(!hasControl){
        return {
          label:'无法比较',
          value:18
        };
      }

      var pooledSpread=
        (
          data.controlSd
          +data.experimentSd
        )/2;

      var separation=
        Math.abs(
          data.difference
        )
        /Math.max(
          3,
          pooledSpread
        );

      var dataSupport=
        clamp(
          separation/2.5,
          0,
          1
        );

      var value=
        score
        *(.42+.58*dataSupport);

      if(!singleVariable){
        value*=.48;
      }

      if(!randomAssignment){
        value*=.68;
      }

      value=clamp(
        value,
        0,
        100
      );

      return {
        label:
          value>=75
            ?'较强支持'
            :value>=48
              ?'有限支持'
              :value>=25
                ?'证据较弱'
                :'无法判断',
        value:value
      };
    }

    function plantGraphic(
      x,
      y,
      scale,
      color,
      label
    ){
      return ''
        +'<g class="ed-plant" transform="translate('
        +x+' '+y+') scale('+scale+')">'
        +'<rect x="-20" y="34" width="40" height="24"'
        +' rx="7" fill="#D6B47C" stroke="#92400E"'
        +' stroke-width="3"/>'
        +'<path d="M0 34 V-16"'
        +' stroke="#15803D" stroke-width="5"'
        +' stroke-linecap="round"/>'
        +'<ellipse cx="-14" cy="3" rx="18" ry="9"'
        +' fill="'+color+'" stroke="#166534"'
        +' stroke-width="2"'
        +' transform="rotate(-26 -14 3)"/>'
        +'<ellipse cx="14" cy="-5" rx="18" ry="9"'
        +' fill="'+color+'" stroke="#166534"'
        +' stroke-width="2"'
        +' transform="rotate(26 14 -5)"/>'
        +'<circle cx="0" cy="-19" r="5"'
        +' fill="#FDE68A" stroke="#D97706"'
        +' stroke-width="2"/>'
        +'<text x="0" y="76" text-anchor="middle"'
        +' font-size="10" font-weight="900"'
        +' fill="#475569">'
        +label
        +'</text>'
        +'</g>';
    }

    function drawGroups(
      data,
      repeats,
      treatment
    ){
      var html='';
      var shown=
        Math.min(
          repeats,
          6
        );

      if(hasControl){
        for(var i=0;i<shown;i++){
          var controlX=
            72+(i%3)*92;

          var controlY=
            154+Math.floor(i/3)*82;

          var controlScale=
            .62
            +data.controlValues[i]/220;

          html+=plantGraphic(
            controlX,
            controlY,
            controlScale,
            '#60A5FA',
            'C'+(i+1)
          );
        }
      }else{
        html+='<rect class="ed-warning"'
          +' x="53" y="130" width="288" height="116"'
          +' rx="16" fill="#FEE2E2"'
          +' stroke="#DC2626" stroke-width="3"'
          +' stroke-dasharray="9 6"/>';

        html+='<text x="197" y="174"'
          +' text-anchor="middle"'
          +' font-size="15" font-weight="900"'
          +' fill="#B91C1C">'
          +'未设置对照组'
          +'</text>';

        html+='<text x="197" y="204"'
          +' text-anchor="middle"'
          +' font-size="11" font-weight="800"'
          +' fill="#7F1D1D">'
          +'缺少未接受自变量处理的比较基准'
          +'</text>';
      }

      for(var j=0;j<shown;j++){
        var experimentX=
          439+(j%3)*92;

        var experimentY=
          154+Math.floor(j/3)*82;

        var experimentScale=
          .62
          +data.experimentValues[j]/220;

        html+=plantGraphic(
          experimentX,
          experimentY,
          experimentScale,
          '#4ADE80',
          'E'+(j+1)
        );
      }

      if(repeats>6){
        html+='<text x="197" y="269"'
          +' text-anchor="middle"'
          +' font-size="10" font-weight="900"'
          +' fill="#1D4ED8">'
          +'另有 '+(repeats-6)+' 个对照重复未在图中展开'
          +'</text>';

        html+='<text x="564" y="269"'
          +' text-anchor="middle"'
          +' font-size="10" font-weight="900"'
          +' fill="#15803D">'
          +'另有 '+(repeats-6)+' 个实验重复未在图中展开'
          +'</text>';
      }

      if(stage==='question'){
        html+='<path class="ed-flow"'
          +' d="M310 117 C354 91 405 91 449 117"'
          +' fill="none" stroke="#D97706"'
          +' stroke-width="4"'
          +' marker-end="url(#${rootId}-arrow)"/>';

        html+='<text x="380" y="93"'
          +' text-anchor="middle"'
          +' font-size="11" font-weight="900"'
          +' fill="#92400E">'
          +'改变光照处理后，生长量是否变化？'
          +'</text>';
      }

      if(stage==='design'){
        html+='<rect x="407" y="118"'
          +' width="314" height="32"'
          +' rx="10" fill="#FEF3C7"'
          +' stroke="#D97706" stroke-width="2"/>';

        html+='<text x="564" y="139"'
          +' text-anchor="middle"'
          +' font-size="10.5" font-weight="900"'
          +' fill="#92400E">'
          +'实验组接受 '+treatment.toFixed(0)
          +'% 的自变量处理'
          +'</text>';
      }

      groupLayer.innerHTML=html;
    }

    function drawLabels(){
      if(!showLabels){
        labelLayer.innerHTML='';
        return;
      }

      var html='';

      html+='<text x="39" y="124"'
        +' font-size="9.5" font-weight="900"'
        +' fill="#1D4ED8">'
        +'比较基准'
        +'</text>';

      html+='<text x="405" y="124"'
        +' font-size="9.5" font-weight="900"'
        +' fill="#15803D">'
        +'接受自变量处理'
        +'</text>';

      if(stage==='design'){
        html+='<text x="197" y="277"'
          +' text-anchor="middle"'
          +' font-size="9.5" font-weight="900"'
          +' fill="#475569">'
          +'因变量：规定时间内的相对生长量'
          +'</text>';

        html+='<text x="564" y="277"'
          +' text-anchor="middle"'
          +' font-size="9.5" font-weight="900"'
          +' fill="#475569">'
          +(singleVariable
            ?'其他条件：尽量保持一致'
            :'其他条件：存在额外变量变化')
          +'</text>';
      }

      labelLayer.innerHTML=html;
    }

    function drawWarnings(){
      var html='';

      if(!singleVariable){
        html+='<rect class="ed-warning"'
          +' x="396" y="112" width="336" height="151"'
          +' rx="18" fill="none"'
          +' stroke="#DC2626" stroke-width="4"'
          +' stroke-dasharray="9 6"/>';

        html+='<text x="564" y="253"'
          +' text-anchor="middle"'
          +' font-size="10.5" font-weight="900"'
          +' fill="#B91C1C">'
          +'除光照外，温度或水分等条件也发生变化'
          +'</text>';
      }

      if(!randomAssignment){
        html+='<path class="ed-warning"'
          +' d="M34 112 H353"'
          +' fill="none" stroke="#7C3AED"'
          +' stroke-width="5"'
          +' stroke-dasharray="8 6"/>';

        html+='<text x="197" y="121"'
          +' text-anchor="middle"'
          +' font-size="10" font-weight="900"'
          +' fill="#6D28D9">'
          +'初始较弱个体更多进入对照组，形成分组偏倚'
          +'</text>';
      }

      warningLayer.innerHTML=html;
    }

    function drawChart(
      data,
      repeats
    ){
      var left=52;
      var right=708;
      var top=322;
      var bottom=397;
      var chartHeight=
        bottom-top;

      var maxValue=100;
      var html='';

      html+='<line x1="'+left
        +'" y1="'+bottom
        +'" x2="'+right
        +'" y2="'+bottom
        +'" stroke="#64748B"'
        +' stroke-width="2"/>';

      html+='<line x1="'+left
        +'" y1="'+bottom
        +'" x2="'+left
        +'" y2="'+top
        +'" stroke="#64748B"'
        +' stroke-width="2"/>';

      for(var gridIndex=0;
        gridIndex<=4;
        gridIndex++
      ){
        var gridY=
          bottom
          -chartHeight*gridIndex/4;

        var gridValue=
          maxValue*gridIndex/4;

        html+='<line x1="'+left
          +'" y1="'+gridY
          +'" x2="'+right
          +'" y2="'+gridY
          +'" stroke="#E2E8F0"'
          +' stroke-width="1"/>';

        html+='<text x="'+(left-7)
          +'" y="'+(gridY+4)
          +'" text-anchor="end"'
          +' font-size="8.5"'
          +' font-weight="700"'
          +' fill="#64748B">'
          +gridValue.toFixed(0)
          +'</text>';
      }

      function y(value){
        return bottom
          -chartHeight
          *clamp(
            value/maxValue,
            0,
            1
          );
      }

      var groupCenters={
        control:230,
        experiment:535
      };

      if(hasControl){
        var controlMeanY=
          y(data.controlMean);

        var controlTopY=
          y(
            data.controlMean
            +data.controlSd
          );

        var controlBottomY=
          y(
            data.controlMean
            -data.controlSd
          );

        html+='<rect x="184"'
          +' y="'+controlMeanY
          +'" width="92"'
          +' height="'+(bottom-controlMeanY)
          +'" rx="6" fill="#60A5FA"'
          +' opacity=".58"/>';

        html+='<line x1="230"'
          +' y1="'+controlTopY
          +'" x2="230"'
          +' y2="'+controlBottomY
          +'" stroke="#1D4ED8"'
          +' stroke-width="3"/>';

        html+='<line x1="215"'
          +' y1="'+controlTopY
          +'" x2="245"'
          +' y2="'+controlTopY
          +'" stroke="#1D4ED8"'
          +' stroke-width="3"/>';

        html+='<line x1="215"'
          +' y1="'+controlBottomY
          +'" x2="245"'
          +' y2="'+controlBottomY
          +'" stroke="#1D4ED8"'
          +' stroke-width="3"/>';

        for(var i=0;
          i<data.controlValues.length;
          i++
        ){
          var controlPointX=
            groupCenters.control
            +(i-(data.controlValues.length-1)/2)
            *Math.min(
              14,
              70/Math.max(
                1,
                data.controlValues.length
              )
            );

          html+='<circle class="ed-point"'
            +' cx="'+controlPointX
            +'" cy="'+y(data.controlValues[i])
            +'" r="4" fill="#FFFFFF"'
            +' stroke="#2563EB"'
            +' stroke-width="2"/>';
        }

        html+='<text x="230" y="416"'
          +' text-anchor="middle"'
          +' font-size="10" font-weight="900"'
          +' fill="#1D4ED8">'
          +'对照组均值 '
          +data.controlMean.toFixed(1)
          +'</text>';
      }else{
        html+='<rect x="174" y="337"'
          +' width="112" height="43"'
          +' rx="10" fill="#FEE2E2"'
          +' stroke="#DC2626"'
          +' stroke-width="2"/>';

        html+='<text x="230" y="363"'
          +' text-anchor="middle"'
          +' font-size="10" font-weight="900"'
          +' fill="#B91C1C">'
          +'无对照数据'
          +'</text>';
      }

      var experimentMeanY=
        y(data.experimentMean);

      var experimentTopY=
        y(
          data.experimentMean
          +data.experimentSd
        );

      var experimentBottomY=
        y(
          data.experimentMean
          -data.experimentSd
        );

      html+='<rect x="489"'
        +' y="'+experimentMeanY
        +'" width="92"'
        +' height="'+(bottom-experimentMeanY)
        +'" rx="6" fill="#4ADE80"'
        +' opacity=".63"/>';

      html+='<line x1="535"'
        +' y1="'+experimentTopY
        +'" x2="535"'
        +' y2="'+experimentBottomY
        +'" stroke="#15803D"'
        +' stroke-width="3"/>';

      html+='<line x1="520"'
        +' y1="'+experimentTopY
        +'" x2="550"'
        +' y2="'+experimentTopY
        +'" stroke="#15803D"'
        +' stroke-width="3"/>';

      html+='<line x1="520"'
        +' y1="'+experimentBottomY
        +'" x2="550"'
        +' y2="'+experimentBottomY
        +'" stroke="#15803D"'
        +' stroke-width="3"/>';

      for(var j=0;
        j<data.experimentValues.length;
        j++
      ){
        var experimentPointX=
          groupCenters.experiment
          +(j-(data.experimentValues.length-1)/2)
          *Math.min(
            14,
            70/Math.max(
              1,
              data.experimentValues.length
            )
          );

        html+='<circle class="ed-point"'
          +' cx="'+experimentPointX
          +'" cy="'+y(data.experimentValues[j])
          +'" r="4" fill="#FFFFFF"'
          +' stroke="#16A34A"'
          +' stroke-width="2"/>';
      }

      html+='<text x="535" y="416"'
        +' text-anchor="middle"'
        +' font-size="10" font-weight="900"'
        +' fill="#15803D">'
        +'实验组均值 '
        +data.experimentMean.toFixed(1)
        +'</text>';

      if(stage==='measure'){
        html+='<text x="380" y="335"'
          +' text-anchor="middle"'
          +' font-size="9.5" font-weight="900"'
          +' fill="#92400E">'
          +'圆点表示每个重复；柱高表示均值；误差棒表示标准差'
          +'</text>';
      }

      if(stage==='conclusion' && hasControl){
        html+='<path class="ed-flow"'
          +' d="M280 353 H481"'
          +' fill="none" stroke="#D97706"'
          +' stroke-width="3"'
          +' marker-end="url(#${rootId}-arrow)"/>';

        html+='<text x="380" y="375"'
          +' text-anchor="middle"'
          +' font-size="9.5" font-weight="900"'
          +' fill="#92400E">'
          +'均值差 '
          +data.difference.toFixed(1)
          +'</text>';
      }

      chartLayer.innerHTML=html;
    }

    function designButtonText(){
      for(var i=0;
        i<designButtons.length;
        i++
      ){
        var type=
          designButtons[i].getAttribute(
            'data-design'
          );

        if(type==='control'){
          designButtons[i].textContent=
            '对照组：'
            +(hasControl?'有':'无');

          designButtons[i].classList.toggle(
            'active',
            hasControl
          );
        }else if(type==='single'){
          designButtons[i].textContent=
            '单一变量：'
            +(singleVariable?'是':'否');

          designButtons[i].classList.toggle(
            'active',
            singleVariable
          );
        }else{
          designButtons[i].textContent=
            '随机分组：'
            +(randomAssignment?'是':'否');

          designButtons[i].classList.toggle(
            'active',
            randomAssignment
          );
        }
      }
    }

    function applyScenario(
      name,
      pauseAutomatic
    ){
      var data=
        scenarios[name];

      if(!data){
        return;
      }

      hasControl=
        data.control;

      singleVariable=
        data.single;

      randomAssignment=
        data.random;

      repeatInput.value=
        String(data.repeat);

      variationInput.value=
        String(data.variation);

      confoundInput.value=
        String(data.confound);

      currentScenario=name;

      if(pauseAutomatic){
        automatic=false;
      }

      update();
    }

    function update(){
      var treatment=
        clamp(
          Number(treatmentInput.value),
          0,
          100
        );

      var repeats=
        clamp(
          Math.round(
            Number(repeatInput.value)
          ),
          1,
          12
        );

      var variation=
        clamp(
          Number(variationInput.value),
          0,
          40
        );

      var confound=
        clamp(
          Number(confoundInput.value),
          0,
          100
        );

      var seed=
        clamp(
          Math.round(
            Number(seedInput.value)
          ),
          1,
          99
        );

      var data=
        buildExperimentData(
          treatment,
          repeats,
          variation,
          confound,
          seed
        );

      var score=
        designScore(
          repeats
        );

      var strength=
        conclusionStrength(
          score,
          data
        );

      var info=
        stageInfo[stage];

      treatmentValue.textContent=
        treatment.toFixed(0)+'%';

      repeatValue.textContent=
        repeats+' 个';

      variationValue.textContent=
        variation.toFixed(0)+'%';

      confoundValue.textContent=
        confound.toFixed(0)+'%';

      seedValue.textContent=
        '第 '+seed+' 组';

      setActive(
        stageButtons,
        'data-stage',
        stage
      );

      setScenarioActive(
        currentScenario
      );

      designButtonText();

      autoButton.textContent=
        automatic
          ?'自动演示：运行中'
          :'自动演示：已暂停';

      autoButton.classList.toggle(
        'paused',
        !automatic
      );

      labelToggle.textContent=
        showLabels
          ?'变量标注：显示'
          :'变量标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      root.style.setProperty(
        '--ed-flow-speed',
        clamp(
          2.7-score/62,
          .62,
          2.5
        ).toFixed(2)+'s'
      );

      title.textContent=
        info.title;

      summary.textContent=
        info.summary;

      statusScore.textContent=
        score.toFixed(0)+'%';

      statusDifference.textContent=
        hasControl
          ?data.difference.toFixed(1)
          :'—';

      statusStrength.textContent=
        strength.label;

      statusScore.style.color=
        score>=75
          ?'#15803D'
          :score>=45
            ?'#B45309'
            :'#B91C1C';

      statusStrength.style.color=
        strength.value>=70
          ?'#15803D'
          :strength.value>=40
            ?'#B45309'
            :'#B91C1C';

      drawGroups(
        data,
        repeats,
        treatment
      );

      drawLabels();
      drawWarnings();

      chartTitle.textContent=
        stage==='question'
          ?'预期通过数据比较回答研究问题'
          :stage==='design'
            ?'设计质量决定数据能否支持因果解释'
            :stage==='measure'
              ?'重复测量值、均值和组内离散程度'
              :'依据设计质量和数据差异形成有限结论';

      drawChart(
        data,
        repeats
      );

      var designNote='';

      if(!hasControl){
        designNote=
          '没有对照组，无法判断实验组结果与未接受处理时相比是否发生变化。';
      }else if(!singleVariable){
        designNote=
          '实验组除研究自变量外还有其他条件改变，因此均值差不能唯一归因于研究自变量。';
      }else if(!randomAssignment){
        designNote=
          '分组不是随机完成，实验开始前的个体差异可能形成系统偏倚。';
      }else if(repeats<=1){
        designNote=
          '每组只有一次测量，无法判断该结果是否代表稳定趋势，也不能估计组内随机波动。';
      }else if(repeats<4){
        designNote=
          '重复数量较少，样本均值仍容易受到个别极端值影响。';
      }else{
        designNote=
          '当前设置具有对照组、单一变量、随机分组和多个重复，实验设计相对规范。';
      }

      var dataNote='';

      var averageSpread=
        (
          data.controlSd
          +data.experimentSd
        )/2;

      if(!hasControl){
        dataNote=
          '目前只能描述实验组测量值，不能计算可靠的组间效应。';
      }else if(
        Math.abs(data.difference)
        <Math.max(
          3,
          averageSpread*.7
        )
      ){
        dataNote=
          '两组均值差较小，且测量值存在重叠，当前数据对处理效应的支持有限。';
      }else if(
        averageSpread>16
      ){
        dataNote=
          '两组均值存在差异，但组内随机波动较大，应谨慎解释并考虑增加重复。';
      }else{
        dataNote=
          '两组均值存在较明显差异，组内离散程度相对可控。';
      }

      var repeatNote=
        repeats<=1
          ?'单次测量不能展示随机误差。'
          :'增加重复通常能提高均值稳定性，但不能修复系统偏倚或混杂变量。';

      footerNote.textContent=
        '圆点＝独立重复；柱高＝样本均值；误差棒＝标准差。标准差表示离散程度，不等同显著性检验。';

      result.innerHTML=
        info.note
        +'<br>'+designNote
        +' '+dataNote
        +' '+repeatNote
        +' 本模型只用于实验设计教学，不形成真实科研、医学或临床结论。';
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(
        !automatic
        ||!root.isConnected
      ){
        return;
      }

      timer=window.setTimeout(
        function(){
          var index=
            stages.indexOf(stage);

          stage=
            stages[
              (index+1)
              %stages.length
            ];

          update();
          schedule();
        },
        2200
      );
    }

    for(var i=0;
      i<stageButtons.length;
      i++
    ){
      stageButtons[i].onclick=function(){
        automatic=false;

        stage=this.getAttribute(
          'data-stage'
        );

        update();
        schedule();
      };
    }

    for(var j=0;
      j<designButtons.length;
      j++
    ){
      designButtons[j].onclick=function(){
        var type=
          this.getAttribute(
            'data-design'
          );

        automatic=false;
        currentScenario='';

        if(type==='control'){
          hasControl=!hasControl;
        }else if(type==='single'){
          singleVariable=!singleVariable;
        }else{
          randomAssignment=!randomAssignment;
        }

        update();
        schedule();
      };
    }

    for(var k=0;
      k<scenarioButtons.length;
      k++
    ){
      scenarioButtons[k].onclick=function(){
        applyScenario(
          this.getAttribute(
            'data-scenario'
          ),
          true
        );

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

    treatmentInput.oninput=function(){
      currentScenario='';
      update();
    };

    repeatInput.oninput=function(){
      currentScenario='';
      update();
    };

    variationInput.oninput=function(){
      currentScenario='';
      update();
    };

    confoundInput.oninput=function(){
      currentScenario='';
      update();
    };

    seedInput.oninput=function(){
      currentScenario='';
      update();
    };

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
