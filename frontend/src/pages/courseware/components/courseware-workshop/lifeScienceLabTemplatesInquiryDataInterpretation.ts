/**
 * lifeScienceLabTemplatesInquiryDataInterpretation.ts
 *
 * 平面生命科学实验室：
 * 实验数据、误差、曲线与结论解释。
 *
 * 教学目标：
 * 1. 区分原始数据、整理后的数据、图表和实验结论；
 * 2. 根据自变量和因变量选择适合的横轴、纵轴和图表形式；
 * 3. 比较线性增长、最适条件和饱和趋势三类常见生命科学关系；
 * 4. 理解实验数据通常会围绕理论趋势发生随机波动；
 * 5. 理解随机误差会造成不同测量点上下波动，
 *    增加样本量通常有助于识别总体趋势；
 * 6. 理解系统误差可能使全部数据整体偏高或偏低，
 *    增加重复不能自动消除系统误差；
 * 7. 识别明显偏离总体趋势的异常值；
 * 8. 理解异常值应先检查记录、测量和实验过程，
 *    不能为了获得理想曲线而直接删除；
 * 9. 理解取值范围过窄可能无法显示完整趋势，
 *    甚至可能把饱和关系误判为线性关系；
 * 10. 使用平均绝对偏差、最大偏差和趋势相关程度
 *     描述模拟数据与理论曲线的接近程度；
 * 11. 区分“数据支持某种趋势”和“已经证明因果关系”；
 * 12. 理解在已观察范围外进行外推需要额外证据；
 * 13. 根据数据质量形成与证据强度匹配的有限结论。
 *
 * 教学边界：
 * 1. 本模型中的自变量、因变量、理论曲线、
 *    测量值和误差指标均为相对教学数据；
 * 2. 理论曲线只是用于比较的假想关系，
 *    不代表某个真实生物实验必然符合该函数；
 * 3. 模型中的误差棒表示测量波动范围示意，
 *    不等同于标准误、置信区间或显著性检验；
 * 4. 相关程度较高不等于已经证明因果关系；
 * 5. 线性相关系数主要描述线性关系，
 *    对最适曲线和饱和曲线不能单独作为趋势质量判断；
 * 6. 异常值可能来自记录错误、测量错误、
 *    个体差异或真实生物变异，应结合实验过程判断；
 * 7. 删除异常值必须有预先规定或可说明的依据，
 *    不能仅因为异常值不符合预期而删除；
 * 8. 系统误差可能来自仪器校准、操作方法、
 *    试剂批次或环境条件的持续偏差；
 * 9. 增加样本量能够减小部分随机波动的影响，
 *    但不能修复错误设计和系统误差；
 * 10. 观察范围过窄时，不应把局部趋势直接推广到全部范围；
 * 11. 本模型不展开正式假设检验、回归诊断、
 *     置信区间、功效分析或多重比较；
 * 12. 本模型不用于医学诊断、药物疗效判断、
 *     人体健康决策或真实科研统计结论。
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
function dataInterpretationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#EFF6FF);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:258px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#7C3AED}'
    + '#' + rootId + ' .di-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .di-relations{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .di-stages{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .di-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .di-button{min-height:31px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:8.7px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .di-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .di-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .di-auto.paused{background:#64748B}'
    + '#' + rootId + ' .di-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .di-toggle.off{background:#64748B}'
    + '#' + rootId + ' .di-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .di-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .di-card b{display:block;min-height:18px;font-size:12.5px;color:#6D28D9}'
    + '#' + rootId + ' .di-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .di-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--di-flow-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .di-point{animation:' + rootId + '-point 1.05s ease-in-out infinite alternate}'
    + '#' + rootId + ' .di-outlier{animation:' + rootId + '-outlier .8s ease-in-out infinite alternate}'
    + '#' + rootId + ' .di-band{animation:' + rootId + '-band 1.35s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-point{from{opacity:.5}to{opacity:1}}'
    + '@keyframes ' + rootId + '-outlier{from{opacity:.35}to{opacity:1}}'
    + '@keyframes ' + rootId + '-band{from{opacity:.3}to{opacity:.68}}'
    + '</style>'
}

/** 避免在外层模板字符串中直接写出脚本结束标签。 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_DATA_INTERPRETATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-data-error-curve-interpretation',
    group: '🧪 实验探究',
    name: '实验数据、误差、曲线与结论解释',
    emoji: '📊',
    desc: '比较理论趋势与模拟测量点，识别随机误差、系统误差、异常值、取值范围和外推边界',
    params: [
      {
        key: 'trendStrength',
        label: '理论效应强度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 70,
      },
      {
        key: 'noiseLevel',
        label: '随机测量波动',
        type: 'number',
        min: 0,
        max: 35,
        step: 1,
        defaultValue: 9,
      },
      {
        key: 'sampleCount',
        label: '测量点数量',
        type: 'number',
        min: 4,
        max: 20,
        step: 1,
        defaultValue: 12,
      },
      {
        key: 'sampleRange',
        label: '自变量覆盖范围',
        type: 'number',
        min: 20,
        max: 100,
        step: 5,
        defaultValue: 100,
      },
      {
        key: 'outlierLevel',
        label: '异常值偏离程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 0,
      },
      {
        key: 'systematicBias',
        label: '系统偏差',
        type: 'number',
        min: -30,
        max: 30,
        step: 1,
        defaultValue: 0,
      },
      {
        key: 'randomSeed',
        label: '随机实验编号',
        type: 'number',
        min: 1,
        max: 99,
        step: 1,
        defaultValue: 41,
      },
      {
        key: 'showLabels',
        label: '显示数据标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const trendStrength = num(
        params,
        'trendStrength',
        70,
      )
      const noiseLevel = num(
        params,
        'noiseLevel',
        9,
      )
      const sampleCount = num(
        params,
        'sampleCount',
        12,
      )
      const sampleRange = num(
        params,
        'sampleRange',
        100,
      )
      const outlierLevel = num(
        params,
        'outlierLevel',
        0,
      )
      const systematicBias = num(
        params,
        'systematicBias',
        0,
      )
      const randomSeed = num(
        params,
        'randomSeed',
        41,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${dataInterpretationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">📊 实验数据、误差、曲线与结论解释</div>
    <div class="bl-note">相关不等于因果；异常值不能为了符合预期而直接删除</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>理论效应强度</span>
          <span class="bl-value" data-strength-value></span>
        </div>
        <input
          data-strength
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(trendStrength)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>随机测量波动</span>
          <span class="bl-value" data-noise-value></span>
        </div>
        <input
          data-noise
          type="range"
          min="0"
          max="35"
          step="1"
          value="${n(noiseLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>测量点数量</span>
          <span class="bl-value" data-count-value></span>
        </div>
        <input
          data-count
          type="range"
          min="4"
          max="20"
          step="1"
          value="${n(sampleCount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>自变量覆盖范围</span>
          <span class="bl-value" data-range-value></span>
        </div>
        <input
          data-range
          type="range"
          min="20"
          max="100"
          step="5"
          value="${n(sampleRange)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>异常值偏离程度</span>
          <span class="bl-value" data-outlier-value></span>
        </div>
        <input
          data-outlier
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(outlierLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>系统偏差</span>
          <span class="bl-value" data-bias-value></span>
        </div>
        <input
          data-bias
          type="range"
          min="-30"
          max="30"
          step="1"
          value="${n(systematicBias)}"
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

      <div class="di-subtitle">理论关系类型</div>

      <div class="di-relations">
        <button
          type="button"
          class="di-button active"
          data-relation="linear"
        >线性增长</button>

        <button
          type="button"
          class="di-button"
          data-relation="optimum"
        >最适条件</button>

        <button
          type="button"
          class="di-button"
          data-relation="saturation"
        >饱和趋势</button>
      </div>

      <div class="di-subtitle">数据分析阶段</div>

      <div class="di-stages">
        <button
          type="button"
          class="di-button active"
          data-stage="collect"
        >1. 收集数据</button>

        <button
          type="button"
          class="di-button"
          data-stage="organize"
        >2. 整理数据</button>

        <button
          type="button"
          class="di-button"
          data-stage="graph"
        >3. 绘制曲线</button>

        <button
          type="button"
          class="di-button"
          data-stage="conclude"
        >4. 解释结论</button>
      </div>

      <div class="di-subtitle">快速比较情境</div>

      <div class="di-scenarios">
        <button
          type="button"
          class="di-button active"
          data-scenario="clear"
        >清晰趋势</button>

        <button
          type="button"
          class="di-button"
          data-scenario="noisy"
        >随机波动大</button>

        <button
          type="button"
          class="di-button"
          data-scenario="outlier"
        >明显异常值</button>

        <button
          type="button"
          class="di-button"
          data-scenario="systematic"
        >系统偏高</button>

        <button
          type="button"
          class="di-button"
          data-scenario="narrow"
        >范围过窄</button>
      </div>

      <button
        type="button"
        class="di-auto"
        data-auto
      >自动演示：运行中</button>

      <button
        type="button"
        class="di-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '数据标注：显示' : '数据标注：隐藏'}</button>

      <div class="di-status">
        <div class="di-card">
          <b data-status-quality></b>
          <span>数据质量评分</span>
        </div>

        <div class="di-card">
          <b data-status-deviation></b>
          <span>平均绝对偏差</span>
        </div>

        <div class="di-card">
          <b data-status-conclusion></b>
          <span>结论强度</span>
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
        aria-label="实验数据误差曲线与结论解释互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-graph"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#FAFAFF"/>
            <stop offset="100%" stop-color="#EEF2FF"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-panel"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#F8FAFC"/>
            <stop offset="100%" stop-color="#ECFDF5"/>
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
              fill="#7C3AED"
            />
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#4C1D95"
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
          fill="#5B21B6"
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
            width="502"
            height="255"
            rx="20"
            fill="url(#${rootId}-graph)"
            stroke="#C4B5FD"
            stroke-width="3"
          />

          <rect
            x="544"
            y="79"
            width="194"
            height="255"
            rx="20"
            fill="url(#${rootId}-panel)"
            stroke="#A7F3D0"
            stroke-width="3"
          />
        </g>

        <text
          x="273"
          y="103"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#5B21B6"
        >理论曲线与模拟测量点</text>

        <text
          x="641"
          y="103"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#047857"
        >数据整理与误差指标</text>

        <g data-grid-layer></g>
        <g data-curve-layer></g>
        <g data-point-layer></g>
        <g data-label-layer></g>
        <g data-panel-layer></g>

        <g transform="translate(22 352)">
          <rect
            width="716"
            height="51"
            rx="15"
            fill="#F8FAFC"
            stroke="#CBD5E1"
            stroke-width="2"
          />

          <text
            x="16"
            y="20"
            data-metric-one
            font-size="10.5"
            font-weight="900"
            fill="#5B21B6"
          ></text>

          <text
            x="250"
            y="20"
            data-metric-two
            font-size="10.5"
            font-weight="900"
            fill="#047857"
          ></text>

          <text
            x="488"
            y="20"
            data-metric-three
            font-size="10.5"
            font-weight="900"
            fill="#B45309"
          ></text>

          <text
            x="16"
            y="41"
            data-panel-note
            font-size="10.5"
            font-weight="900"
            fill="#475569"
          ></text>
        </g>

        <text
          x="22"
          y="423"
          data-footer-note
          font-size="10.5"
          font-weight="900"
          fill="#4C1D95"
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

    var strengthInput=root.querySelector(
      '[data-strength]'
    );
    var noiseInput=root.querySelector(
      '[data-noise]'
    );
    var countInput=root.querySelector(
      '[data-count]'
    );
    var rangeInput=root.querySelector(
      '[data-range]'
    );
    var outlierInput=root.querySelector(
      '[data-outlier]'
    );
    var biasInput=root.querySelector(
      '[data-bias]'
    );
    var seedInput=root.querySelector(
      '[data-seed]'
    );

    var strengthValue=root.querySelector(
      '[data-strength-value]'
    );
    var noiseValue=root.querySelector(
      '[data-noise-value]'
    );
    var countValue=root.querySelector(
      '[data-count-value]'
    );
    var rangeValue=root.querySelector(
      '[data-range-value]'
    );
    var outlierValue=root.querySelector(
      '[data-outlier-value]'
    );
    var biasValue=root.querySelector(
      '[data-bias-value]'
    );
    var seedValue=root.querySelector(
      '[data-seed-value]'
    );

    var relationButtons=root.querySelectorAll(
      '[data-relation]'
    );
    var stageButtons=root.querySelectorAll(
      '[data-stage]'
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

    var statusQuality=root.querySelector(
      '[data-status-quality]'
    );
    var statusDeviation=root.querySelector(
      '[data-status-deviation]'
    );
    var statusConclusion=root.querySelector(
      '[data-status-conclusion]'
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
    var gridLayer=root.querySelector(
      '[data-grid-layer]'
    );
    var curveLayer=root.querySelector(
      '[data-curve-layer]'
    );
    var pointLayer=root.querySelector(
      '[data-point-layer]'
    );
    var labelLayer=root.querySelector(
      '[data-label-layer]'
    );
    var panelLayer=root.querySelector(
      '[data-panel-layer]'
    );

    var metricOne=root.querySelector(
      '[data-metric-one]'
    );
    var metricTwo=root.querySelector(
      '[data-metric-two]'
    );
    var metricThree=root.querySelector(
      '[data-metric-three]'
    );
    var panelNote=root.querySelector(
      '[data-panel-note]'
    );
    var footerNote=root.querySelector(
      '[data-footer-note]'
    );

    var relations=[
      'linear',
      'optimum',
      'saturation'
    ];

    var stages=[
      'collect',
      'organize',
      'graph',
      'conclude'
    ];

    var relation='linear';
    var stage='collect';
    var automatic=true;
    var timer=null;
    var showLabels=${showLabels ? 'true' : 'false'};
    var currentScenario='clear';

    var stageInfo={
      collect:{
        title:'阶段1：在规定范围内收集原始数据',
        summary:'记录每个自变量水平对应的因变量测量值，不预先挑选符合预期的数据',
        note:'原始数据应完整记录，包括看似异常或与预期不一致的测量值。'
      },
      organize:{
        title:'阶段2：整理数据并检查误差来源',
        summary:'比较取值范围、测量点数量、随机波动、系统偏差和异常值',
        note:'整理数据不是修改数据，而是核对记录、单位、缺失值和可能的误差来源。'
      },
      graph:{
        title:'阶段3：选择坐标并绘制数据图',
        summary:'自变量置于横轴，因变量置于纵轴，同时显示理论趋势与模拟测量点',
        note:'坐标范围、刻度和图形形式会影响读图，但不能改变原始数据本身。'
      },
      conclude:{
        title:'阶段4：形成与证据强度匹配的有限结论',
        summary:'结合趋势、离散程度、异常值、系统偏差和观察范围解释结果',
        note:'结论应说明数据支持什么、不能说明什么，以及还存在哪些不确定性。'
      }
    };

    var scenarios={
      clear:{
        relation:'linear',
        strength:70,
        noise:7,
        count:12,
        range:100,
        outlier:0,
        bias:0
      },
      noisy:{
        relation:'linear',
        strength:70,
        noise:30,
        count:10,
        range:100,
        outlier:0,
        bias:0
      },
      outlier:{
        relation:'linear',
        strength:70,
        noise:9,
        count:11,
        range:100,
        outlier:88,
        bias:0
      },
      systematic:{
        relation:'linear',
        strength:70,
        noise:7,
        count:12,
        range:100,
        outlier:0,
        bias:24
      },
      narrow:{
        relation:'saturation',
        strength:76,
        noise:8,
        count:8,
        range:30,
        outlier:0,
        bias:0
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

    function trueValue(
      x,
      relationType,
      strength
    ){
      if(relationType==='optimum'){
        var distance=
          (x-55)/24;

        return 15
          +strength
          *Math.exp(
            -distance*distance
          );
      }

      if(relationType==='saturation'){
        return 12
          +strength
          *x/(x+28);
      }

      return 12
        +strength*x/100;
    }

    function buildData(
      strength,
      noise,
      count,
      range,
      outlier,
      bias,
      seed
    ){
      var random=createRandom(
        seed,
        31
      );

      var minimumX=
        50-range/2;

      var maximumX=
        50+range/2;

      var points=[];

      var outlierIndex=
        outlier>0
          ?Math.floor(
            count*.58
          )
          :-1;

      for(var i=0;i<count;i++){
        var x=
          count<=1
            ?50
            :minimumX
              +(maximumX-minimumX)
              *i/(count-1);

        var theory=
          trueValue(
            x,
            relation,
            strength
          );

        var randomNoise=
          (
            random()
            +random()
            +random()
            -1.5
          )
          *noise;

        var outlierShift=
          i===outlierIndex
            ?(
              seed%2===0
                ?1
                :-1
            )
            *outlier*.42
            :0;

        var observed=
          clamp(
            theory
            +randomNoise
            +bias
            +outlierShift,
            0,
            100
          );

        points.push({
          x:x,
          theory:theory,
          observed:observed,
          residual:
            observed-theory,
          outlier:
            i===outlierIndex
        });
      }

      return points;
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

    function meanAbsoluteDeviation(
      points
    ){
      var values=[];

      for(var i=0;i<points.length;i++){
        values.push(
          Math.abs(
            points[i].residual
          )
        );
      }

      return mean(values);
    }

    function maximumDeviation(
      points
    ){
      var maximum=0;

      for(var i=0;i<points.length;i++){
        maximum=Math.max(
          maximum,
          Math.abs(
            points[i].residual
          )
        );
      }

      return maximum;
    }

    function linearCorrelation(
      points
    ){
      if(points.length<2){
        return 0;
      }

      var xs=[];
      var ys=[];

      for(var i=0;i<points.length;i++){
        xs.push(
          points[i].x
        );

        ys.push(
          points[i].observed
        );
      }

      var meanX=
        mean(xs);

      var meanY=
        mean(ys);

      var numerator=0;
      var denominatorX=0;
      var denominatorY=0;

      for(var j=0;j<points.length;j++){
        var dx=
          xs[j]-meanX;

        var dy=
          ys[j]-meanY;

        numerator+=
          dx*dy;

        denominatorX+=
          dx*dx;

        denominatorY+=
          dy*dy;
      }

      if(
        denominatorX<=0
        ||denominatorY<=0
      ){
        return 0;
      }

      return numerator
        /Math.sqrt(
          denominatorX
          *denominatorY
        );
    }

    function qualityScore(
      count,
      range,
      noise,
      outlier,
      bias
    ){
      var score=100;

      if(count<6){
        score-=20;
      }else if(count<9){
        score-=10;
      }

      if(range<40){
        score-=25;
      }else if(range<65){
        score-=12;
      }

      score-=noise*.65;
      score-=outlier*.18;
      score-=Math.abs(bias)*1.15;

      return clamp(
        score,
        0,
        100
      );
    }

    function patternState(
      points,
      range
    ){
      if(relation==='linear'){
        var correlation=
          linearCorrelation(
            points
          );

        return {
          label:
            correlation>.8
              ?'正相关清晰'
              :correlation>.45
                ?'正相关有限'
                :correlation>-.15
                  ?'趋势不清'
                  :'反向波动',
          detail:
            '线性相关程度 '
            +correlation.toFixed(2),
          correlation:correlation
        };
      }

      if(relation==='optimum'){
        var peak=
          points[0];

        for(var i=1;i<points.length;i++){
          if(
            points[i].observed
            >peak.observed
          ){
            peak=
              points[i];
          }
        }

        return {
          label:
            range<45
              ?'最适位置不确定'
              :Math.abs(peak.x-55)<18
                ?'存在中间峰值'
                :'峰值受噪声影响',
          detail:
            '观察最高点位于自变量 '
            +peak.x.toFixed(0),
          correlation:0
        };
      }

      var first=
        points[0].observed;

      var middle=
        points[
          Math.floor(
            points.length/2
          )
        ].observed;

      var last=
        points[
          points.length-1
        ].observed;

      var earlyGain=
        middle-first;

      var lateGain=
        last-middle;

      return {
        label:
          range<45
            ?'仅见局部趋势'
            :earlyGain>lateGain*1.25
              ?'出现饱和趋势'
              :'饱和趋势不清',
        detail:
          '前半增量 '
          +earlyGain.toFixed(1)
          +'｜后半增量 '
          +lateGain.toFixed(1),
        correlation:0
      };
    }

    function conclusionModel(
      quality,
      pattern,
      range,
      outlier,
      bias
    ){
      var value=quality;

      if(
        relation==='linear'
        &&pattern.correlation>.75
      ){
        value+=8;
      }

      if(
        relation==='linear'
        &&pattern.correlation<.3
      ){
        value-=16;
      }

      if(range<40){
        value-=10;
      }

      if(outlier>55){
        value-=12;
      }

      if(Math.abs(bias)>15){
        value-=15;
      }

      value=clamp(
        value,
        0,
        100
      );

      return {
        value:value,
        label:
          value>=76
            ?'较强支持'
            :value>=50
              ?'有限支持'
              :value>=28
                ?'证据较弱'
                :'暂不能判断'
      };
    }

    function graphBounds(){
      return {
        left:58,
        right:500,
        top:119,
        bottom:309
      };
    }

    function graphX(
      value,
      bounds
    ){
      return bounds.left
        +(bounds.right-bounds.left)
        *value/100;
    }

    function graphY(
      value,
      bounds
    ){
      return bounds.bottom
        -(bounds.bottom-bounds.top)
        *clamp(
          value/100,
          0,
          1
        );
    }

    function drawGrid(){
      var bounds=
        graphBounds();

      var html='';

      html+='<line x1="'+bounds.left
        +'" y1="'+bounds.bottom
        +'" x2="'+bounds.right
        +'" y2="'+bounds.bottom
        +'" stroke="#64748B"'
        +' stroke-width="2.5"/>';

      html+='<line x1="'+bounds.left
        +'" y1="'+bounds.bottom
        +'" x2="'+bounds.left
        +'" y2="'+bounds.top
        +'" stroke="#64748B"'
        +' stroke-width="2.5"/>';

      for(var index=0;index<=5;index++){
        var value=
          index*20;

        var x=
          graphX(
            value,
            bounds
          );

        var y=
          graphY(
            value,
            bounds
          );

        html+='<line x1="'+bounds.left
          +'" y1="'+y
          +'" x2="'+bounds.right
          +'" y2="'+y
          +'" stroke="#E2E8F0"'
          +' stroke-width="1"/>';

        html+='<line x1="'+x
          +'" y1="'+bounds.bottom
          +'" x2="'+x
          +'" y2="'+bounds.top
          +'" stroke="#F1F5F9"'
          +' stroke-width="1"/>';

        html+='<text x="'+(bounds.left-8)
          +'" y="'+(y+4)
          +'" text-anchor="end"'
          +' font-size="9"'
          +' font-weight="700"'
          +' fill="#64748B">'
          +value
          +'</text>';

        html+='<text x="'+x
          +'" y="'+(bounds.bottom+17)
          +'" text-anchor="middle"'
          +' font-size="9"'
          +' font-weight="700"'
          +' fill="#64748B">'
          +value
          +'</text>';
      }

      html+='<text x="'+bounds.right
        +'" y="'+(bounds.bottom+32)
        +'" text-anchor="end"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#475569">'
        +'自变量'
        +'</text>';

      html+='<text x="'+(bounds.left-33)
        +'" y="'+bounds.top
        +'" font-size="10"'
        +' font-weight="900"'
        +' fill="#475569">'
        +'因变量'
        +'</text>';

      gridLayer.innerHTML=html;
    }

    function drawTheoryCurve(
      strength,
      range
    ){
      var bounds=
        graphBounds();

      var path='';
      var bandTop='';
      var bandBottom='';
      var samples=80;

      for(var i=0;i<=samples;i++){
        var xValue=
          100*i/samples;

        var yValue=
          trueValue(
            xValue,
            relation,
            strength
          );

        var x=
          graphX(
            xValue,
            bounds
          );

        var y=
          graphY(
            yValue,
            bounds
          );

        var upper=
          graphY(
            yValue+5,
            bounds
          );

        var lower=
          graphY(
            yValue-5,
            bounds
          );

        path+=(i===0?'M':' L')
          +x+' '+y;

        bandTop+=(i===0?'M':' L')
          +x+' '+upper;

        bandBottom=
          ' L'+x+' '+lower
          +bandBottom;
      }

      var rangeStart=
        50-range/2;

      var rangeEnd=
        50+range/2;

      var rangeXOne=
        graphX(
          rangeStart,
          bounds
        );

      var rangeXTwo=
        graphX(
          rangeEnd,
          bounds
        );

      var html='';

      html+='<rect x="'+rangeXOne
        +'" y="'+bounds.top
        +'" width="'+(rangeXTwo-rangeXOne)
        +'" height="'+(bounds.bottom-bounds.top)
        +'" fill="#DDD6FE"'
        +' opacity=".16"/>';

      html+='<path class="di-band" d="'
        +bandTop
        +bandBottom
        +' Z" fill="#A78BFA"'
        +' opacity=".22"/>';

      html+='<path d="'+path
        +'" fill="none"'
        +' stroke="#7C3AED"'
        +' stroke-width="5"'
        +' stroke-linecap="round"'
        +' stroke-linejoin="round"/>';

      html+='<text x="'+rangeXOne
        +'" y="114"'
        +' font-size="9"'
        +' font-weight="900"'
        +' fill="#6D28D9">'
        +'观察范围起点'
        +'</text>';

      html+='<text x="'+rangeXTwo
        +'" y="114"'
        +' text-anchor="end"'
        +' font-size="9"'
        +' font-weight="900"'
        +' fill="#6D28D9">'
        +'观察范围终点'
        +'</text>';

      curveLayer.innerHTML=html;
    }

    function drawPoints(
      points,
      noise
    ){
      var bounds=
        graphBounds();

      var html='';
      var linePath='';

      for(var i=0;i<points.length;i++){
        var point=
          points[i];

        var x=
          graphX(
            point.x,
            bounds
          );

        var y=
          graphY(
            point.observed,
            bounds
          );

        linePath+=(i===0?'M':' L')
          +x+' '+y;

        var uncertainty=
          Math.max(
            2,
            noise*.35
          );

        var top=
          graphY(
            point.observed
            +uncertainty,
            bounds
          );

        var bottom=
          graphY(
            point.observed
            -uncertainty,
            bounds
          );

        html+='<line x1="'+x
          +'" y1="'+top
          +'" x2="'+x
          +'" y2="'+bottom
          +'" stroke="#94A3B8"'
          +' stroke-width="1.7"/>';

        html+='<line x1="'+(x-4)
          +'" y1="'+top
          +'" x2="'+(x+4)
          +'" y2="'+top
          +'" stroke="#94A3B8"'
          +' stroke-width="1.7"/>';

        html+='<line x1="'+(x-4)
          +'" y1="'+bottom
          +'" x2="'+(x+4)
          +'" y2="'+bottom
          +'" stroke="#94A3B8"'
          +' stroke-width="1.7"/>';

        html+='<circle class="'
          +(point.outlier
            ?'di-outlier'
            :'di-point')
          +'" cx="'+x
          +'" cy="'+y
          +'" r="'
          +(point.outlier?8:5)
          +'" fill="#FFFFFF"'
          +' stroke="'
          +(point.outlier
            ?'#DC2626'
            :'#2563EB')
          +'" stroke-width="'
          +(point.outlier?4:2.8)
          +'"/>';

        if(
          showLabels
          &&(
            stage==='organize'
            ||stage==='conclude'
          )
        ){
          html+='<text x="'+x
            +'" y="'+Math.max(
              bounds.top+10,
              y-12
            )
            +'" text-anchor="middle"'
            +' font-size="8.5"'
            +' font-weight="900"'
            +' fill="'
            +(point.outlier
              ?'#B91C1C'
              :'#1D4ED8')
            +'">'
            +point.observed.toFixed(1)
            +'</text>';
        }
      }

      if(stage!=='collect'){
        html='<path d="'+linePath
          +'" fill="none"'
          +' stroke="#2563EB"'
          +' stroke-width="2.5"'
          +' stroke-dasharray="5 5"'
          +' opacity=".6"/>'
          +html;
      }

      pointLayer.innerHTML=html;
    }

    function drawLabels(
      points,
      bias,
      outlier
    ){
      if(!showLabels){
        labelLayer.innerHTML='';
        return;
      }

      var html='';

      html+='<line x1="70" y1="325"'
        +' x2="99" y2="325"'
        +' stroke="#7C3AED"'
        +' stroke-width="5"/>';

      html+='<text x="107" y="329"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#5B21B6">'
        +'理论趋势'
        +'</text>';

      html+='<circle cx="198" cy="325"'
        +' r="5" fill="#FFFFFF"'
        +' stroke="#2563EB"'
        +' stroke-width="2.5"/>';

      html+='<text x="210" y="329"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'模拟测量值'
        +'</text>';

      html+='<line x1="330" y1="318"'
        +' x2="330" y2="332"'
        +' stroke="#94A3B8"'
        +' stroke-width="2"/>';

      html+='<text x="342" y="329"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#64748B">'
        +'测量波动范围示意'
        +'</text>';

      if(outlier>0){
        html+='<text x="440" y="329"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#B91C1C">'
          +'红圈：需核查的异常值'
          +'</text>';
      }

      if(
        stage==='organize'
        &&Math.abs(bias)>8
      ){
        html+='<path class="di-flow"'
          +' d="M515 162 H536"'
          +' fill="none"'
          +' stroke="#F59E0B"'
          +' stroke-width="3"'
          +' marker-end="url(#${rootId}-arrow)"/>';

        html+='<text x="525" y="150"'
          +' text-anchor="middle"'
          +' font-size="9"'
          +' font-weight="900"'
          +' fill="#B45309">'
          +'整体偏移'
          +'</text>';
      }

      labelLayer.innerHTML=html;
    }

    function drawPanel(
      points,
      pattern,
      averageDeviation,
      maximum,
      quality
    ){
      var residuals=[];

      for(var i=0;i<points.length;i++){
        residuals.push(
          points[i].residual
        );
      }

      var meanResidual=
        mean(
          residuals
        );

      var html='';

      html+='<text x="561" y="133"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'测量点数量'
        +'</text>';

      html+='<text x="720" y="133"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#5B21B6">'
        +points.length
        +'</text>';

      html+='<text x="561" y="158"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'平均残差'
        +'</text>';

      html+='<text x="720" y="158"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="'
        +(Math.abs(meanResidual)<6
          ?'#047857'
          :'#B45309')
        +'">'
        +meanResidual.toFixed(1)
        +'</text>';

      html+='<text x="561" y="183"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'平均绝对偏差'
        +'</text>';

      html+='<text x="720" y="183"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#5B21B6">'
        +averageDeviation.toFixed(1)
        +'</text>';

      html+='<text x="561" y="208"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'最大偏差'
        +'</text>';

      html+='<text x="720" y="208"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="'
        +(maximum<18
          ?'#047857'
          :'#B91C1C')
        +'">'
        +maximum.toFixed(1)
        +'</text>';

      html+='<rect x="561" y="225"'
        +' width="159" height="42"'
        +' rx="10" fill="#FFFFFF"'
        +' stroke="#A7F3D0"'
        +' stroke-width="2"/>';

      html+='<text x="641" y="242"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'观察到的关系'
        +'</text>';

      html+='<text x="641" y="259"'
        +' text-anchor="middle"'
        +' font-size="10.5"'
        +' font-weight="900"'
        +' fill="#047857">'
        +pattern.label
        +'</text>';

      html+='<rect x="561" y="278"'
        +' width="159" height="15"'
        +' rx="7.5" fill="#E2E8F0"/>';

      html+='<rect x="561" y="278"'
        +' width="'+(159*quality/100)
        +'" height="15"'
        +' rx="7.5" fill="'
        +(quality>=70
          ?'#10B981'
          :quality>=40
            ?'#F59E0B'
            :'#EF4444')
        +'"/>';

      html+='<text x="641" y="312"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#475569">'
        +'数据质量 '
        +quality.toFixed(0)
        +'%'
        +'</text>';

      panelLayer.innerHTML=html;
    }

    function qualityExplanation(
      count,
      range,
      noise,
      outlier,
      bias,
      pattern
    ){
      if(Math.abs(bias)>15){
        return '全部测量点整体偏离理论曲线，提示可能存在持续的系统误差。增加测量次数不能自动消除这一偏差。';
      }

      if(outlier>55){
        return '存在一个明显偏离总体趋势的数据点，应核查原始记录、仪器和实验过程，不能只因其不符合预期而删除。';
      }

      if(range<40){
        return '自变量覆盖范围过窄，目前只能看到局部变化，不能据此确认完整曲线形状或进行远距离外推。';
      }

      if(count<6){
        return '测量点数量较少，个别数据点会显著影响曲线判断，应增加覆盖不同自变量水平的测量点。';
      }

      if(noise>24){
        return '随机波动较大，测量点在理论趋势两侧明显分散，当前趋势判断具有较大不确定性。';
      }

      if(pattern.label.indexOf('清晰')>=0){
        return '测量点整体围绕理论趋势分布，当前数据对该关系提供较清晰支持。';
      }

      return '当前数据能够显示一定趋势，但仍应结合误差来源、样本数量和观察范围谨慎解释。';
    }

    function conclusionText(
      conclusion,
      pattern,
      range,
      bias,
      outlier
    ){
      var relationText=
        relation==='linear'
          ?'在当前观察范围内，因变量随自变量增加总体呈上升趋势'
          :relation==='optimum'
            ?'当前数据提示因变量在中间自变量范围可能出现较高值'
            :'当前数据提示因变量随自变量增加后，增幅可能逐渐减小';

      var limitation='';

      if(Math.abs(bias)>15){
        limitation=
          '但全部数据可能受到系统偏差影响，不能直接使用当前数值判断真实效应大小。';
      }else if(outlier>55){
        limitation=
          '但存在需要核查的异常值，异常值的来源尚未明确。';
      }else if(range<40){
        limitation=
          '但取值范围过窄，不能确认完整曲线，更不能把局部趋势推广到全部范围。';
      }else{
        limitation=
          '该结论只适用于当前实验设计、测量范围和模拟数据。';
      }

      return relationText
        +'，结论支持程度为“'
        +conclusion.label
        +'”。'
        +limitation
        +' 数据相关或趋势一致不等于已经证明因果关系。';
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

      relation=
        data.relation;

      strengthInput.value=
        String(data.strength);

      noiseInput.value=
        String(data.noise);

      countInput.value=
        String(data.count);

      rangeInput.value=
        String(data.range);

      outlierInput.value=
        String(data.outlier);

      biasInput.value=
        String(data.bias);

      currentScenario=name;

      if(pauseAutomatic){
        automatic=false;
      }

      update();
    }

    function update(){
      var strength=
        clamp(
          Number(strengthInput.value),
          10,
          100
        );

      var noise=
        clamp(
          Number(noiseInput.value),
          0,
          35
        );

      var count=
        clamp(
          Math.round(
            Number(countInput.value)
          ),
          4,
          20
        );

      var range=
        clamp(
          Number(rangeInput.value),
          20,
          100
        );

      var outlier=
        clamp(
          Number(outlierInput.value),
          0,
          100
        );

      var bias=
        clamp(
          Number(biasInput.value),
          -30,
          30
        );

      var seed=
        clamp(
          Math.round(
            Number(seedInput.value)
          ),
          1,
          99
        );

      var points=
        buildData(
          strength,
          noise,
          count,
          range,
          outlier,
          bias,
          seed
        );

      var averageDeviation=
        meanAbsoluteDeviation(
          points
        );

      var maximum=
        maximumDeviation(
          points
        );

      var quality=
        qualityScore(
          count,
          range,
          noise,
          outlier,
          bias
        );

      var pattern=
        patternState(
          points,
          range
        );

      var conclusion=
        conclusionModel(
          quality,
          pattern,
          range,
          outlier,
          bias
        );

      var info=
        stageInfo[stage];

      strengthValue.textContent=
        strength.toFixed(0)+'%';

      noiseValue.textContent=
        noise.toFixed(0)+'%';

      countValue.textContent=
        count+' 个';

      rangeValue.textContent=
        range.toFixed(0)+'%';

      outlierValue.textContent=
        outlier.toFixed(0)+'%';

      biasValue.textContent=
        (
          bias>0
            ?'+'
            :''
        )
        +bias.toFixed(0);

      seedValue.textContent=
        '第 '+seed+' 组';

      setActive(
        relationButtons,
        'data-relation',
        relation
      );

      setActive(
        stageButtons,
        'data-stage',
        stage
      );

      setScenarioActive(
        currentScenario
      );

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
          ?'数据标注：显示'
          :'数据标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      root.style.setProperty(
        '--di-flow-speed',
        clamp(
          2.7-quality/62,
          .62,
          2.5
        ).toFixed(2)+'s'
      );

      title.textContent=
        info.title;

      summary.textContent=
        info.summary;

      statusQuality.textContent=
        quality.toFixed(0)+'%';

      statusDeviation.textContent=
        averageDeviation.toFixed(1);

      statusConclusion.textContent=
        conclusion.label;

      statusQuality.style.color=
        quality>=70
          ?'#047857'
          :quality>=40
            ?'#B45309'
            :'#B91C1C';

      statusConclusion.style.color=
        conclusion.value>=70
          ?'#047857'
          :conclusion.value>=40
            ?'#B45309'
            :'#B91C1C';

      drawGrid();

      drawTheoryCurve(
        strength,
        range
      );

      drawPoints(
        points,
        noise
      );

      drawLabels(
        points,
        bias,
        outlier
      );

      drawPanel(
        points,
        pattern,
        averageDeviation,
        maximum,
        quality
      );

      metricOne.textContent=
        pattern.detail;

      metricTwo.textContent=
        '平均绝对偏差 '
        +averageDeviation.toFixed(1);

      metricThree.textContent=
        '最大偏差 '
        +maximum.toFixed(1);

      var qualityNote=
        qualityExplanation(
          count,
          range,
          noise,
          outlier,
          bias,
          pattern
        );

      panelNote.textContent=
        qualityNote;

      footerNote.textContent=
        '误差棒仅表示测量波动范围示意，不是置信区间或显著性检验；理论曲线也不是由当前数据自动证明的真实规律。';

      var conclusionNote=
        conclusionText(
          conclusion,
          pattern,
          range,
          bias,
          outlier
        );

      result.innerHTML=
        info.note
        +'<br>'+qualityNote
        +' '+conclusionNote
        +' 本模型不用于医学、药物疗效或真实科研统计结论。';
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
      i<relationButtons.length;
      i++
    ){
      relationButtons[i].onclick=function(){
        automatic=false;

        relation=this.getAttribute(
          'data-relation'
        );

        currentScenario='';

        update();
        schedule();
      };
    }

    for(var j=0;
      j<stageButtons.length;
      j++
    ){
      stageButtons[j].onclick=function(){
        automatic=false;

        stage=this.getAttribute(
          'data-stage'
        );

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

    strengthInput.oninput=function(){
      currentScenario='';
      update();
    };

    noiseInput.oninput=function(){
      currentScenario='';
      update();
    };

    countInput.oninput=function(){
      currentScenario='';
      update();
    };

    rangeInput.oninput=function(){
      currentScenario='';
      update();
    };

    outlierInput.oninput=function(){
      currentScenario='';
      update();
    };

    biasInput.oninput=function(){
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
