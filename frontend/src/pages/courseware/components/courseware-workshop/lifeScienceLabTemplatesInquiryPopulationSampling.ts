/**
 * lifeScienceLabTemplatesInquiryPopulationSampling.ts
 *
 * 平面生命科学实验室：
 * 样方法与标志重捕法种群调查。
 *
 * 教学目标：
 * 1. 理解种群数量调查通常只能获得估计值，
 *    单次估计不等于真实种群数量；
 * 2. 理解样方法适合调查植物、活动能力弱或位置相对固定的生物；
 * 3. 理解样方应尽量随机或系统布设，
 *    避免只选择个体密集、稀疏或方便观察的位置；
 * 4. 根据样方平均个体数和调查面积比例估算种群数量；
 * 5. 理解增加样方数量通常能够提高样本代表性，
 *    但不能自动修复取样位置偏倚；
 * 6. 理解聚集分布会使不同样方之间的个体数差异增大；
 * 7. 理解标志重捕法适合调查活动能力较强、
 *    个体边界较清楚且能够重复捕获的动物；
 * 8. 根据首次标记数M、第二次捕获数C和重捕标记数R，
 *    使用N≈M×C÷R估算种群数量；
 * 9. 理解标记个体需要充分混合，
 *    并应与未标记个体具有近似相同的被捕获概率；
 * 10. 理解标记脱落、标记影响行为、逃避捕获、
 *     偏好再次进入捕获装置等情况都会造成估计偏差；
 * 11. 理解R过小或等于0时，估计结果非常不稳定或无法计算；
 * 12. 比较随机样方、样方过少、聚集分布、
 *     标记脱落和再捕获偏差等情境。
 *
 * 教学边界：
 * 1. 所有种群数量、样方面积、捕获数量和估计误差
 *    均为相对教学指标；
 * 2. 样方法估算式为：
 *    估计种群数量≈样方平均个体数×总区域样方单元数；
 * 3. 本模型把调查区域简化为24个等面积样方单元；
 * 4. 聚集分布会增加样方计数的离散程度，
 *    但实际空间分布还可能呈均匀分布或随机分布；
 * 5. 增加样方数量通常能够减小随机抽样波动，
 *    但若样方布设本身有偏，增加数量也可能持续产生偏差；
 * 6. 标志重捕法估算式N≈M×C÷R建立在若干近似条件上；
 * 7. 两次捕获之间应尽量满足种群相对封闭，
 *    出生、死亡、迁入和迁出均可能影响估计；
 * 8. 标志不能明显影响个体存活、行为和被捕获概率；
 * 9. 标记脱落会使可识别的重捕个体减少，
 *    常导致种群数量被高估；
 * 10. 标记个体逃避再次捕获会使R偏小，
 *     常导致种群数量被高估；
 * 11. 标记个体更容易再次被捕获会使R偏大，
 *     常导致种群数量被低估；
 * 12. 本模型不提供真实野生动物捕捉、
 *     标记、麻醉、运输或放归操作方法；
 * 13. 本模型不用于保护区管理、狩猎限额、
 *     渔业配额、害虫防治或真实生态决策。
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
function populationSamplingStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #6EE7B7;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#D1FAE5,#ECFEFF);border-bottom:1px solid #6EE7B7}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:260px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FFFC;border-right:1px solid #A7F3D0}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#059669;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981}'
    + '#' + rootId + ' .ps-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .ps-methods{display:grid;grid-template-columns:1fr 1fr;gap:5px;margin-bottom:7px}'
    + '#' + rootId + ' .ps-stages{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ps-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .ps-button{min-height:31px;padding:3px;border:1px solid #6EE7B7;border-radius:8px;background:#fff;color:#065F46;font-size:8.7px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .ps-button.active{border-color:#10B981;background:#D1FAE5;box-shadow:0 3px 9px rgba(16,185,129,.14)}'
    + '#' + rootId + ' .ps-quality{width:100%;height:31px;margin-bottom:7px;border:1px solid #6EE7B7;border-radius:8px;background:#D1FAE5;color:#065F46;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ps-quality.off{border-color:#FCA5A5;background:#FEE2E2;color:#991B1B}'
    + '#' + rootId + ' .ps-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ps-auto.paused{background:#64748B}'
    + '#' + rootId + ' .ps-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ps-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ps-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .ps-card{padding:6px 3px;border:1px solid #A7F3D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ps-card b{display:block;min-height:18px;font-size:12.5px;color:#047857}'
    + '#' + rootId + ' .ps-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#D1FAE5;color:#064E3B;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ps-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--ps-flow-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ps-organism{animation:' + rootId + '-organism 1.55s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ps-mark{animation:' + rootId + '-mark 1.05s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ps-warning{animation:' + rootId + '-warning .9s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-organism{from{transform:translateY(2px);opacity:.52}to{transform:translateY(-3px);opacity:1}}'
    + '@keyframes ' + rootId + '-mark{from{opacity:.38}to{opacity:1}}'
    + '@keyframes ' + rootId + '-warning{from{opacity:.35}to{opacity:1}}'
    + '</style>'
}

/** 避免在外层模板字符串中直接写出脚本结束标签。 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_POPULATION_SAMPLING:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-population-sampling-estimation',
    group: '🧪 实验探究',
    name: '样方法与标志重捕法种群调查',
    emoji: '🗺️',
    desc: '比较样方随机取样和标志重捕估算，观察样本量、聚集分布、标记脱落与再捕获偏差',
    params: [
      {
        key: 'actualPopulation',
        label: '模型实际种群数量',
        type: 'number',
        min: 40,
        max: 240,
        step: 5,
        defaultValue: 120,
      },
      {
        key: 'quadratCount',
        label: '抽取样方数量',
        type: 'number',
        min: 1,
        max: 12,
        step: 1,
        defaultValue: 6,
      },
      {
        key: 'patchiness',
        label: '空间聚集程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 35,
      },
      {
        key: 'firstMarked',
        label: '首次捕获并标记M',
        type: 'number',
        min: 10,
        max: 100,
        step: 5,
        defaultValue: 40,
      },
      {
        key: 'secondCapture',
        label: '第二次捕获数量C',
        type: 'number',
        min: 10,
        max: 100,
        step: 5,
        defaultValue: 50,
      },
      {
        key: 'markRetention',
        label: '标记保留率',
        type: 'number',
        min: 50,
        max: 100,
        step: 1,
        defaultValue: 92,
      },
      {
        key: 'recaptureBias',
        label: '标记个体再捕倾向',
        type: 'number',
        min: 40,
        max: 160,
        step: 5,
        defaultValue: 100,
        hint: '100表示与未标记个体近似等概率',
      },
      {
        key: 'randomSeed',
        label: '随机调查编号',
        type: 'number',
        min: 1,
        max: 99,
        step: 1,
        defaultValue: 23,
      },
      {
        key: 'showLabels',
        label: '显示调查标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const actualPopulation = num(
        params,
        'actualPopulation',
        120,
      )
      const quadratCount = num(
        params,
        'quadratCount',
        6,
      )
      const patchiness = num(
        params,
        'patchiness',
        35,
      )
      const firstMarked = num(
        params,
        'firstMarked',
        40,
      )
      const secondCapture = num(
        params,
        'secondCapture',
        50,
      )
      const markRetention = num(
        params,
        'markRetention',
        92,
      )
      const recaptureBias = num(
        params,
        'recaptureBias',
        100,
      )
      const randomSeed = num(
        params,
        'randomSeed',
        23,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${populationSamplingStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🗺️ 样方法与标志重捕法种群调查</div>
    <div class="bl-note">种群调查得到的是估计值；方法选择和抽样条件决定估计质量</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>模型实际种群数量</span>
          <span class="bl-value" data-population-value></span>
        </div>
        <input
          data-population
          type="range"
          min="40"
          max="240"
          step="5"
          value="${n(actualPopulation)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>抽取样方数量</span>
          <span class="bl-value" data-quadrat-value></span>
        </div>
        <input
          data-quadrat
          type="range"
          min="1"
          max="12"
          step="1"
          value="${n(quadratCount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>空间聚集程度</span>
          <span class="bl-value" data-patch-value></span>
        </div>
        <input
          data-patch
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(patchiness)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>首次捕获并标记M</span>
          <span class="bl-value" data-marked-value></span>
        </div>
        <input
          data-marked
          type="range"
          min="10"
          max="100"
          step="5"
          value="${n(firstMarked)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>第二次捕获数量C</span>
          <span class="bl-value" data-capture-value></span>
        </div>
        <input
          data-capture
          type="range"
          min="10"
          max="100"
          step="5"
          value="${n(secondCapture)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>标记保留率</span>
          <span class="bl-value" data-retention-value></span>
        </div>
        <input
          data-retention
          type="range"
          min="50"
          max="100"
          step="1"
          value="${n(markRetention)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>标记个体再捕倾向</span>
          <span class="bl-value" data-bias-value></span>
        </div>
        <input
          data-bias
          type="range"
          min="40"
          max="160"
          step="5"
          value="${n(recaptureBias)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>随机调查编号</span>
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

      <div class="ps-subtitle">调查方法</div>

      <div class="ps-methods">
        <button
          type="button"
          class="ps-button active"
          data-method="quadrat"
        >样方法</button>

        <button
          type="button"
          class="ps-button"
          data-method="markRecapture"
        >标志重捕法</button>
      </div>

      <div class="ps-subtitle">调查阶段</div>

      <div class="ps-stages">
        <button
          type="button"
          class="ps-button active"
          data-stage="design"
        >1. 选择方法</button>

        <button
          type="button"
          class="ps-button"
          data-stage="sample"
        >2. 抽样调查</button>

        <button
          type="button"
          class="ps-button"
          data-stage="calculate"
        >3. 计算估计</button>

        <button
          type="button"
          class="ps-button"
          data-stage="evaluate"
        >4. 评价误差</button>
      </div>

      <div class="ps-subtitle">快速比较情境</div>

      <div class="ps-scenarios">
        <button
          type="button"
          class="ps-button active"
          data-scenario="standardQuadrat"
        >规范样方</button>

        <button
          type="button"
          class="ps-button"
          data-scenario="fewQuadrats"
        >样方过少</button>

        <button
          type="button"
          class="ps-button"
          data-scenario="clumped"
        >聚集分布</button>

        <button
          type="button"
          class="ps-button"
          data-scenario="markLoss"
        >标记脱落</button>

        <button
          type="button"
          class="ps-button"
          data-scenario="captureBias"
        >再捕偏差</button>
      </div>

      <button
        type="button"
        class="ps-quality"
        data-quality-toggle
      >样方布设：随机</button>

      <button
        type="button"
        class="ps-auto"
        data-auto
      >自动演示：运行中</button>

      <button
        type="button"
        class="ps-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '调查标注：显示' : '调查标注：隐藏'}</button>

      <div class="ps-status">
        <div class="ps-card">
          <b data-status-method></b>
          <span>当前调查方法</span>
        </div>

        <div class="ps-card">
          <b data-status-estimate></b>
          <span>估计种群数量</span>
        </div>

        <div class="ps-card">
          <b data-status-error></b>
          <span>相对估计误差</span>
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
        aria-label="样方法与标志重捕法种群调查互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-habitat"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#ECFDF5"/>
            <stop offset="100%" stop-color="#D1FAE5"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-panel"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#F8FAFC"/>
            <stop offset="100%" stop-color="#EFF6FF"/>
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
              fill="#059669"
            />
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#064E3B"
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
          fill="#065F46"
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
            width="446"
            height="220"
            rx="20"
            fill="url(#${rootId}-habitat)"
            stroke="#6EE7B7"
            stroke-width="3"
          />

          <rect
            x="488"
            y="79"
            width="250"
            height="220"
            rx="20"
            fill="url(#${rootId}-panel)"
            stroke="#BFDBFE"
            stroke-width="3"
          />
        </g>

        <text
          x="245"
          y="103"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#065F46"
        >模拟调查区域</text>

        <text
          x="613"
          y="103"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#1D4ED8"
        >调查记录与计算</text>

        <g data-habitat-layer></g>
        <g data-population-layer></g>
        <g data-sampling-layer></g>
        <g data-label-layer></g>
        <g data-calculation-layer></g>

        <g transform="translate(22 321)">
          <rect
            width="716"
            height="81"
            rx="17"
            fill="#F8FAFC"
            stroke="#CBD5E1"
            stroke-width="2"
          />

          <text
            x="16"
            y="21"
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          >模型实际数量</text>

          <rect
            x="111"
            y="12"
            width="210"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-actual-bar
            x="111"
            y="12"
            width="0"
            height="13"
            rx="6.5"
            fill="#64748B"
          />

          <text
            x="375"
            y="21"
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          >调查估计数量</text>

          <rect
            x="470"
            y="12"
            width="218"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-estimate-bar
            x="470"
            y="12"
            width="0"
            height="13"
            rx="6.5"
            fill="#10B981"
          />

          <text
            x="16"
            y="49"
            data-observation-one
            font-size="10.5"
            font-weight="900"
            fill="#065F46"
          ></text>

          <text
            x="375"
            y="49"
            data-observation-two
            font-size="10.5"
            font-weight="900"
            fill="#1D4ED8"
          ></text>

          <text
            x="16"
            y="70"
            data-panel-note
            font-size="10.5"
            font-weight="900"
            fill="#92400E"
          ></text>
        </g>

        <text
          x="22"
          y="423"
          data-footer-note
          font-size="10.5"
          font-weight="900"
          fill="#064E3B"
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

    var populationInput=root.querySelector(
      '[data-population]'
    );
    var quadratInput=root.querySelector(
      '[data-quadrat]'
    );
    var patchInput=root.querySelector(
      '[data-patch]'
    );
    var markedInput=root.querySelector(
      '[data-marked]'
    );
    var captureInput=root.querySelector(
      '[data-capture]'
    );
    var retentionInput=root.querySelector(
      '[data-retention]'
    );
    var biasInput=root.querySelector(
      '[data-bias]'
    );
    var seedInput=root.querySelector(
      '[data-seed]'
    );

    var populationValue=root.querySelector(
      '[data-population-value]'
    );
    var quadratValue=root.querySelector(
      '[data-quadrat-value]'
    );
    var patchValue=root.querySelector(
      '[data-patch-value]'
    );
    var markedValue=root.querySelector(
      '[data-marked-value]'
    );
    var captureValue=root.querySelector(
      '[data-capture-value]'
    );
    var retentionValue=root.querySelector(
      '[data-retention-value]'
    );
    var biasValue=root.querySelector(
      '[data-bias-value]'
    );
    var seedValue=root.querySelector(
      '[data-seed-value]'
    );

    var methodButtons=root.querySelectorAll(
      '[data-method]'
    );
    var stageButtons=root.querySelectorAll(
      '[data-stage]'
    );
    var scenarioButtons=root.querySelectorAll(
      '[data-scenario]'
    );

    var qualityToggle=root.querySelector(
      '[data-quality-toggle]'
    );
    var autoButton=root.querySelector(
      '[data-auto]'
    );
    var labelToggle=root.querySelector(
      '[data-label-toggle]'
    );

    var statusMethod=root.querySelector(
      '[data-status-method]'
    );
    var statusEstimate=root.querySelector(
      '[data-status-estimate]'
    );
    var statusError=root.querySelector(
      '[data-status-error]'
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
    var habitatLayer=root.querySelector(
      '[data-habitat-layer]'
    );
    var populationLayer=root.querySelector(
      '[data-population-layer]'
    );
    var samplingLayer=root.querySelector(
      '[data-sampling-layer]'
    );
    var labelLayer=root.querySelector(
      '[data-label-layer]'
    );
    var calculationLayer=root.querySelector(
      '[data-calculation-layer]'
    );

    var actualBar=root.querySelector(
      '[data-actual-bar]'
    );
    var estimateBar=root.querySelector(
      '[data-estimate-bar]'
    );
    var observationOne=root.querySelector(
      '[data-observation-one]'
    );
    var observationTwo=root.querySelector(
      '[data-observation-two]'
    );
    var panelNote=root.querySelector(
      '[data-panel-note]'
    );
    var footerNote=root.querySelector(
      '[data-footer-note]'
    );

    var methods=[
      'quadrat',
      'markRecapture'
    ];

    var stages=[
      'design',
      'sample',
      'calculate',
      'evaluate'
    ];

    var method='quadrat';
    var stage='design';
    var automatic=true;
    var timer=null;
    var showLabels=${showLabels ? 'true' : 'false'};
    var representativeSampling=true;
    var currentScenario='standardQuadrat';

    var stageInfo={
      design:{
        title:'阶段1：根据生物特征选择调查方法',
        note:'位置相对固定的生物通常适合样方法；活动能力较强的动物可考虑标志重捕法。'
      },
      sample:{
        title:'阶段2：按照设计完成抽样或捕获',
        note:'调查过程应尽量减少选择性取样，并记录样方面积、捕获数量和标记状态。'
      },
      calculate:{
        title:'阶段3：使用调查数据估算种群数量',
        note:'计算式只在相应假设近似满足时具有解释意义，不能脱离调查过程单独使用。'
      },
      evaluate:{
        title:'阶段4：比较估计值、真实值与误差来源',
        note:'估计偏差可能来自随机抽样波动，也可能来自取样偏倚、标记脱落或捕获概率不等。'
      }
    };

    var scenarios={
      standardQuadrat:{
        method:'quadrat',
        representative:true,
        quadrats:8,
        patchiness:28,
        retention:92,
        bias:100
      },
      fewQuadrats:{
        method:'quadrat',
        representative:true,
        quadrats:2,
        patchiness:38,
        retention:92,
        bias:100
      },
      clumped:{
        method:'quadrat',
        representative:true,
        quadrats:5,
        patchiness:94,
        retention:92,
        bias:100
      },
      markLoss:{
        method:'markRecapture',
        representative:true,
        quadrats:6,
        patchiness:35,
        retention:58,
        bias:100
      },
      captureBias:{
        method:'markRecapture',
        representative:true,
        quadrats:6,
        patchiness:35,
        retention:92,
        bias:55
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

    function weightedChoice(
      weights,
      random
    ){
      var total=0;

      for(var i=0;i<weights.length;i++){
        total+=weights[i];
      }

      var value=
        random()*total;

      var running=0;

      for(var j=0;j<weights.length;j++){
        running+=weights[j];

        if(value<running){
          return j;
        }
      }

      return weights.length-1;
    }

    function buildPopulation(
      population,
      patchiness,
      seed
    ){
      var random=createRandom(
        seed,
        17
      );

      var clusterCells=[
        4,
        9,
        19
      ];

      var weights=[];

      for(var cell=0;cell<24;cell++){
        var row=
          Math.floor(
            cell/6
          );

        var column=
          cell%6;

        var clusterEffect=0;

        for(var index=0;
          index<clusterCells.length;
          index++
        ){
          var cluster=
            clusterCells[index];

          var clusterRow=
            Math.floor(
              cluster/6
            );

          var clusterColumn=
            cluster%6;

          var distanceSquared=
            Math.pow(
              row-clusterRow,
              2
            )
            +Math.pow(
              column-clusterColumn,
              2
            );

          clusterEffect+=
            Math.exp(
              -distanceSquared/1.6
            );
        }

        weights.push(
          1
          +patchiness/100
          *8
          *clusterEffect
        );
      }

      var counts=[];

      for(var empty=0;empty<24;empty++){
        counts.push(0);
      }

      var points=[];

      for(var individual=0;
        individual<population;
        individual++
      ){
        var selected=
          weightedChoice(
            weights,
            random
          );

        counts[selected]+=1;

        var selectedRow=
          Math.floor(
            selected/6
          );

        var selectedColumn=
          selected%6;

        points.push({
          cell:selected,
          x:
            45
            +selectedColumn*68
            +9
            +random()*50,
          y:
            119
            +selectedRow*40
            +7
            +random()*27
        });
      }

      return {
        counts:counts,
        points:points
      };
    }

    function chooseSampleCells(
      counts,
      count,
      seed
    ){
      var selected=[];

      if(!representativeSampling){
        var indexes=[];

        for(var i=0;i<counts.length;i++){
          indexes.push(i);
        }

        indexes.sort(function(a,b){
          return counts[b]-counts[a];
        });

        return indexes.slice(
          0,
          count
        );
      }

      var random=createRandom(
        seed,
        53
      );

      while(
        selected.length<count
      ){
        var candidate=
          Math.floor(
            random()*24
          );

        if(
          selected.indexOf(
            candidate
          )<0
        ){
          selected.push(candidate);
        }
      }

      selected.sort(function(a,b){
        return a-b;
      });

      return selected;
    }

    function quadratModel(
      population,
      quadrats,
      patchiness,
      seed
    ){
      var distribution=
        buildPopulation(
          population,
          patchiness,
          seed
        );

      var sampledCells=
        chooseSampleCells(
          distribution.counts,
          quadrats,
          seed
        );

      var sampledCounts=[];

      for(var i=0;
        i<sampledCells.length;
        i++
      ){
        sampledCounts.push(
          distribution.counts[
            sampledCells[i]
          ]
        );
      }

      var average=
        mean(
          sampledCounts
        );

      var deviation=
        standardDeviation(
          sampledCounts,
          average
        );

      var estimate=
        average*24;

      var error=
        Math.abs(
          estimate-population
        )
        /Math.max(
          1,
          population
        )
        *100;

      return {
        distribution:distribution,
        sampledCells:sampledCells,
        sampledCounts:sampledCounts,
        average:average,
        deviation:deviation,
        estimate:estimate,
        error:error
      };
    }

    function simulateBinomial(
      trials,
      probability,
      seed
    ){
      var random=createRandom(
        seed,
        89
      );

      var successes=0;

      for(var i=0;i<trials;i++){
        if(random()<probability){
          successes+=1;
        }
      }

      return successes;
    }

    function markRecaptureModel(
      population,
      firstMarked,
      secondCapture,
      retention,
      bias,
      seed
    ){
      var M=
        clamp(
          Math.round(
            firstMarked
          ),
          1,
          Math.max(
            1,
            population-1
          )
        );

      var C=
        clamp(
          Math.round(
            secondCapture
          ),
          1,
          population
        );

      var recognizable=
        M*retention/100;

      var baseProbability=
        recognizable
        /Math.max(
          1,
          population
        );

      var biasFactor=
        bias/100;

      var adjustedProbability=
        baseProbability*biasFactor
        /Math.max(
          .0001,
          1-baseProbability
          +baseProbability*biasFactor
        );

      adjustedProbability=
        clamp(
          adjustedProbability,
          0,
          1
        );

      var R=
        simulateBinomial(
          C,
          adjustedProbability,
          seed
        );

      R=Math.min(
        R,
        Math.round(
          recognizable
        ),
        C
      );

      var estimate=
        R>0
          ?M*C/R
          :null;

      var error=
        estimate===null
          ?null
          :Math.abs(
            estimate-population
          )
          /Math.max(
            1,
            population
          )
          *100;

      return {
        M:M,
        C:C,
        recognizable:recognizable,
        probability:adjustedProbability,
        R:R,
        estimate:estimate,
        error:error
      };
    }

    function formatEstimate(value){
      if(value===null){
        return '无法估计';
      }

      if(value>=1000){
        return value.toExponential(1);
      }

      return value.toFixed(0);
    }

    function errorLabel(error){
      if(error===null){
        return '无法计算';
      }

      if(error<10){
        return '误差较小';
      }

      if(error<25){
        return '存在偏差';
      }

      return '偏差较大';
    }

    function organism(
      x,
      y,
      marked,
      recaptured
    ){
      var html='';

      html+='<g class="ps-organism"'
        +' transform="translate('+x+' '+y+')">';

      html+='<ellipse cx="0" cy="0"'
        +' rx="6.5" ry="4.5"'
        +' fill="'
        +(marked?'#EC4899':'#10B981')
        +'" stroke="'
        +(marked?'#9D174D':'#047857')
        +'" stroke-width="1.5"/>';

      html+='<circle cx="6" cy="-1" r="3.2"'
        +' fill="'
        +(marked?'#F9A8D4':'#6EE7B7')
        +'" stroke="'
        +(marked?'#9D174D':'#047857')
        +'" stroke-width="1.2"/>';

      html+='<circle cx="7" cy="-2"'
        +' r=".8" fill="#111827"/>';

      if(marked){
        html+='<rect class="ps-mark"'
          +' x="-3" y="-6"'
          +' width="5" height="4"'
          +' rx="1" fill="#FDE68A"'
          +' stroke="#B45309"'
          +' stroke-width=".8"/>';
      }

      if(recaptured){
        html+='<circle class="ps-mark"'
          +' cx="0" cy="0" r="10"'
          +' fill="none" stroke="#DC2626"'
          +' stroke-width="2.5"/>';
      }

      html+='</g>';

      return html;
    }

    function drawHabitatGrid(){
      var html='';

      html+='<rect x="38" y="115"'
        +' width="408" height="160"'
        +' rx="10" fill="#F0FDF4"'
        +' stroke="#059669"'
        +' stroke-width="2.5"/>';

      for(var column=1;column<6;column++){
        var x=
          38+column*68;

        html+='<line x1="'+x
          +'" y1="115"'
          +' x2="'+x
          +'" y2="275"'
          +' stroke="#A7F3D0"'
          +' stroke-width="1.2"/>';
      }

      for(var row=1;row<4;row++){
        var y=
          115+row*40;

        html+='<line x1="38"'
          +' y1="'+y
          +'" x2="446"'
          +' y2="'+y
          +'" stroke="#A7F3D0"'
          +' stroke-width="1.2"/>';
      }

      habitatLayer.innerHTML=html;
    }

    function drawQuadratPopulation(
      model
    ){
      var html='';
      var points=
        model.distribution.points;

      var maxShown=
        Math.min(
          points.length,
          96
        );

      for(var i=0;i<maxShown;i++){
        html+=organism(
          points[i].x,
          points[i].y,
          false,
          false
        );
      }

      if(points.length>maxShown){
        html+='<text x="245" y="292"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#047857">'
          +'图中仅展示部分个体，样方计数使用完整模型种群'
          +'</text>';
      }

      populationLayer.innerHTML=html;
    }

    function drawQuadrats(
      model
    ){
      var html='';

      for(var i=0;
        i<model.sampledCells.length;
        i++
      ){
        var cell=
          model.sampledCells[i];

        var row=
          Math.floor(
            cell/6
          );

        var column=
          cell%6;

        var x=
          38+column*68;

        var y=
          115+row*40;

        html+='<rect class="ps-mark"'
          +' x="'+(x+2)
          +'" y="'+(y+2)
          +'" width="64" height="36"'
          +' rx="6" fill="#FEF3C7"'
          +' fill-opacity=".26"'
          +' stroke="#D97706"'
          +' stroke-width="3"/>';

        html+='<text x="'+(x+34)
          +'" y="'+(y+23)
          +'" text-anchor="middle"'
          +' font-size="11"'
          +' font-weight="900"'
          +' fill="#92400E">'
          +model.sampledCounts[i]
          +'</text>';
      }

      samplingLayer.innerHTML=html;
    }

    function drawMarkPopulation(
      population,
      model,
      seed
    ){
      var distribution=
        buildPopulation(
          population,
          35,
          seed
        );

      var points=
        distribution.points;

      var maxShown=
        Math.min(
          points.length,
          82
        );

      var displayedMarked=
        Math.round(
          maxShown
          *model.M
          /Math.max(
            1,
            population
          )
        );

      var displayedRecaptured=
        Math.min(
          displayedMarked,
          Math.round(
            maxShown
            *model.R
            /Math.max(
              1,
              model.C
            )
            *.7
          )
        );

      var html='';

      for(var i=0;i<maxShown;i++){
        html+=organism(
          points[i].x,
          points[i].y,
          i<displayedMarked,
          i<displayedRecaptured
        );
      }

      populationLayer.innerHTML=html;

      var overlay='';

      if(stage==='sample'){
        overlay+='<path class="ps-flow"'
          +' d="M82 288 C166 309 329 309 414 288"'
          +' fill="none" stroke="#059669"'
          +' stroke-width="4"'
          +' marker-end="url(#${rootId}-arrow)"/>';

        overlay+='<text x="245" y="309"'
          +' text-anchor="middle"'
          +' font-size="10.5"'
          +' font-weight="900"'
          +' fill="#065F46">'
          +'首次标记后放回并充分混合，再进行第二次捕获'
          +'</text>';
      }

      samplingLayer.innerHTML=overlay;
    }

    function drawLabels(
      quadrat,
      mark
    ){
      if(!showLabels){
        labelLayer.innerHTML='';
        return;
      }

      var html='';

      if(method==='quadrat'){
        html+='<text x="42" y="111"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#047857">'
          +'24个等面积样方单元'
          +'</text>';

        html+='<text x="330" y="111"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#92400E">'
          +(representativeSampling
            ?'橙框：随机抽取样方'
            :'橙框：偏向个体密集区域')
          +'</text>';

        if(stage==='evaluate'){
          html+='<text x="245" y="292"'
            +' text-anchor="middle"'
            +' font-size="9.5"'
            +' font-weight="900"'
            +' fill="'
            +(quadrat.error<20
              ?'#047857'
              :'#B91C1C')
            +'">'
            +'样方计数标准差 '
            +quadrat.deviation.toFixed(1)
            +'｜相对误差 '
            +quadrat.error.toFixed(1)
            +'%'
            +'</text>';
        }
      }else{
        html+='<text x="42" y="111"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#047857">'
          +'绿色：未标记个体'
          +'</text>';

        html+='<text x="171" y="111"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#BE185D">'
          +'粉色：首次标记个体'
          +'</text>';

        html+='<text x="324" y="111"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#B91C1C">'
          +'红圈：第二次捕获中的重捕标记个体'
          +'</text>';

        if(stage==='evaluate'){
          html+='<text x="245" y="292"'
            +' text-anchor="middle"'
            +' font-size="9.5"'
            +' font-weight="900"'
            +' fill="'
            +(mark.error!==null
              &&mark.error<20
                ?'#047857'
                :'#B91C1C')
            +'">'
            +(mark.error===null
              ?'R=0，无法进行稳定估算'
              :'相对估计误差 '
                +mark.error.toFixed(1)
                +'%')
            +'</text>';
        }
      }

      labelLayer.innerHTML=html;
    }

    function drawQuadratCalculation(
      model
    ){
      var html='';

      html+='<text x="507" y="132"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'样方调查记录'
        +'</text>';

      html+='<text x="507" y="157"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'抽取样方数'
        +'</text>';

      html+='<text x="714" y="157"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#065F46">'
        +model.sampledCells.length
        +'</text>';

      html+='<text x="507" y="181"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'样方平均个体数'
        +'</text>';

      html+='<text x="714" y="181"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#065F46">'
        +model.average.toFixed(2)
        +'</text>';

      html+='<text x="507" y="205"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'样方计数标准差'
        +'</text>';

      html+='<text x="714" y="205"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#065F46">'
        +model.deviation.toFixed(2)
        +'</text>';

      html+='<rect x="507" y="220"'
        +' width="212" height="53"'
        +' rx="12" fill="#FFFFFF"'
        +' stroke="#93C5FD"'
        +' stroke-width="2"/>';

      html+='<text x="613" y="239"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'估算式'
        +'</text>';

      html+='<text x="613" y="259"'
        +' text-anchor="middle"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'N≈平均数×24＝'
        +model.estimate.toFixed(0)
        +'</text>';

      calculationLayer.innerHTML=html;
    }

    function drawMarkCalculation(
      model
    ){
      var html='';

      html+='<text x="507" y="132"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'标志重捕记录'
        +'</text>';

      html+='<text x="507" y="157"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'首次标记数 M'
        +'</text>';

      html+='<text x="714" y="157"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#BE185D">'
        +model.M
        +'</text>';

      html+='<text x="507" y="181"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'第二次捕获数 C'
        +'</text>';

      html+='<text x="714" y="181"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#065F46">'
        +model.C
        +'</text>';

      html+='<text x="507" y="205"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'重捕标记数 R'
        +'</text>';

      html+='<text x="714" y="205"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="'
        +(model.R>0
          ?'#B45309'
          :'#B91C1C')
        +'">'
        +model.R
        +'</text>';

      html+='<rect x="507" y="220"'
        +' width="212" height="53"'
        +' rx="12" fill="#FFFFFF"'
        +' stroke="#93C5FD"'
        +' stroke-width="2"/>';

      html+='<text x="613" y="239"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'估算式'
        +'</text>';

      html+='<text x="613" y="259"'
        +' text-anchor="middle"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="'
        +(model.R>0
          ?'#1D4ED8'
          :'#B91C1C')
        +'">'
        +(model.R>0
          ?'N≈M×C÷R＝'
            +model.estimate.toFixed(0)
          :'R=0，不能进行除法估算')
        +'</text>';

      calculationLayer.innerHTML=html;
    }

    function qualityNote(
      quadrat,
      mark,
      quadrats,
      patchiness,
      retention,
      bias
    ){
      if(method==='quadrat'){
        if(!representativeSampling){
          return '样方位置偏向个体密集区域，当前估计容易系统性偏高。';
        }

        if(quadrats<=2){
          return '样方数量过少，单个样方的偶然高值或低值会显著影响估计。';
        }

        if(patchiness>=75){
          return '种群呈明显聚集分布，不同样方计数差异较大，应增加样方数量并扩大空间覆盖。';
        }

        if(quadrat.error<10){
          return '当前随机样方覆盖较好，本次估计较接近模型实际数量。';
        }

        return '当前估计仍存在抽样波动，可改变调查编号或增加样方数量比较结果。';
      }

      if(mark.R===0){
        return '第二次捕获中没有识别到标记个体，估算式分母为0，当前无法得到稳定估计。';
      }

      if(mark.R<=2){
        return '重捕标记数很少，R的微小变化会造成估计值大幅波动。';
      }

      if(retention<75){
        return '标记保留率较低，部分首次标记个体无法被识别，R偏小并可能高估种群数量。';
      }

      if(bias<80){
        return '标记个体倾向逃避再次捕获，R偏小并可能高估种群数量。';
      }

      if(bias>120){
        return '标记个体更容易再次被捕获，R偏大并可能低估种群数量。';
      }

      if(
        mark.error!==null
        &&mark.error<12
      ){
        return '当前标记保留和再捕获概率近似合理，本次估计较接近模型实际数量。';
      }

      return '有限捕获样本会出现随机波动，可改变调查编号或增加捕获数量进行比较。';
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

      method=
        data.method;

      representativeSampling=
        data.representative;

      quadratInput.value=
        String(data.quadrats);

      patchInput.value=
        String(data.patchiness);

      retentionInput.value=
        String(data.retention);

      biasInput.value=
        String(data.bias);

      currentScenario=name;

      if(pauseAutomatic){
        automatic=false;
      }

      update();
    }

    function update(){
      var population=
        clamp(
          Math.round(
            Number(populationInput.value)
          ),
          40,
          240
        );

      var quadrats=
        clamp(
          Math.round(
            Number(quadratInput.value)
          ),
          1,
          12
        );

      var patchiness=
        clamp(
          Number(patchInput.value),
          0,
          100
        );

      var firstMarked=
        clamp(
          Number(markedInput.value),
          10,
          100
        );

      var secondCapture=
        clamp(
          Number(captureInput.value),
          10,
          100
        );

      var retention=
        clamp(
          Number(retentionInput.value),
          50,
          100
        );

      var bias=
        clamp(
          Number(biasInput.value),
          40,
          160
        );

      var seed=
        clamp(
          Math.round(
            Number(seedInput.value)
          ),
          1,
          99
        );

      var quadrat=
        quadratModel(
          population,
          quadrats,
          patchiness,
          seed
        );

      var mark=
        markRecaptureModel(
          population,
          firstMarked,
          secondCapture,
          retention,
          bias,
          seed
        );

      var estimate=
        method==='quadrat'
          ?quadrat.estimate
          :mark.estimate;

      var error=
        method==='quadrat'
          ?quadrat.error
          :mark.error;

      var info=
        stageInfo[stage];

      populationValue.textContent=
        population.toFixed(0);

      quadratValue.textContent=
        quadrats+' 个';

      patchValue.textContent=
        patchiness.toFixed(0)+'%';

      markedValue.textContent=
        Math.min(
          firstMarked,
          population-1
        ).toFixed(0);

      captureValue.textContent=
        Math.min(
          secondCapture,
          population
        ).toFixed(0);

      retentionValue.textContent=
        retention.toFixed(0)+'%';

      biasValue.textContent=
        bias.toFixed(0)+'%';

      seedValue.textContent=
        '第 '+seed+' 组';

      setActive(
        methodButtons,
        'data-method',
        method
      );

      setActive(
        stageButtons,
        'data-stage',
        stage
      );

      setScenarioActive(
        currentScenario
      );

      qualityToggle.textContent=
        method==='quadrat'
          ?representativeSampling
            ?'样方布设：随机'
            :'样方布设：偏向密集区'
          :Math.abs(bias-100)<=10
            ?'捕获概率：近似相同'
            :'捕获概率：存在偏差';

      qualityToggle.classList.toggle(
        'off',
        method==='quadrat'
          ?!representativeSampling
          :Math.abs(bias-100)>10
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
          ?'调查标注：显示'
          :'调查标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      root.style.setProperty(
        '--ps-flow-speed',
        clamp(
          2.7
          -(method==='quadrat'
            ?quadrats*6
            :mark.C)/60,
          .62,
          2.5
        ).toFixed(2)+'s'
      );

      title.textContent=
        info.title;

      summary.textContent=
        method==='quadrat'
          ?'样方法：通过部分等面积样方的个体数推断整个调查区域'
          :'标志重捕法：利用第二次捕获样本中的标记个体比例估算种群数量';

      statusMethod.textContent=
        method==='quadrat'
          ?'样方法'
          :'标志重捕';

      statusEstimate.textContent=
        formatEstimate(
          estimate
        );

      statusError.textContent=
        error===null
          ?'—'
          :error.toFixed(1)+'%';

      statusError.style.color=
        error===null
          ?'#B91C1C'
          :error<10
            ?'#047857'
            :error<25
              ?'#B45309'
              :'#B91C1C';

      drawHabitatGrid();

      if(method==='quadrat'){
        drawQuadratPopulation(
          quadrat
        );

        drawQuadrats(
          quadrat
        );

        drawQuadratCalculation(
          quadrat
        );
      }else{
        drawMarkPopulation(
          population,
          mark,
          seed
        );

        drawMarkCalculation(
          mark
        );
      }

      drawLabels(
        quadrat,
        mark
      );

      var maxDisplay=
        Math.max(
          population,
          estimate===null
            ?0
            :estimate,
          100
        )
        *1.12;

      actualBar.setAttribute(
        'width',
        String(
          210
          *population/maxDisplay
        )
      );

      estimateBar.setAttribute(
        'width',
        String(
          estimate===null
            ?0
            :218
              *clamp(
                estimate/maxDisplay,
                0,
                1
              )
        )
      );

      estimateBar.setAttribute(
        'fill',
        error===null
          ?'#94A3B8'
          :error<10
            ?'#10B981'
            :error<25
              ?'#F59E0B'
              :'#EF4444'
      );

      if(method==='quadrat'){
        observationOne.textContent=
          '样方计数：'
          +quadrat.sampledCounts.join('、');

        observationTwo.textContent=
          '平均数 '
          +quadrat.average.toFixed(2)
          +'｜标准差 '
          +quadrat.deviation.toFixed(2);
      }else{
        observationOne.textContent=
          'M='+mark.M
          +'｜C='+mark.C
          +'｜R='+mark.R;

        observationTwo.textContent=
          '可识别标记个体约 '
          +mark.recognizable.toFixed(1)
          +'｜再捕概率 '
          +(mark.probability*100).toFixed(1)
          +'%';
      }

      var quality=
        qualityNote(
          quadrat,
          mark,
          quadrats,
          patchiness,
          retention,
          bias
        );

      panelNote.textContent=
        quality;

      footerNote.textContent=
        method==='quadrat'
          ?'样方法适合位置相对固定的生物；随机或系统布设、足够样方和充分空间覆盖是代表性的关键。'
          :'标志重捕法近似要求种群相对封闭、标记不脱落、充分混合且标记与未标记个体被捕概率相近。';

      var methodBoundary=
        method==='quadrat'
          ?'样方法不适合直接调查移动迅速且会频繁离开样方的动物。'
          :'标志重捕法不应在标记明显影响存活、行为或捕获概率时直接套用估算式。';

      result.innerHTML=
        info.note
        +'<br>'+quality
        +' 当前估计判断：'
        +errorLabel(error)
        +'。 '+methodBoundary
        +' 本模型不提供真实捕捉、标记或野外操作方案，也不用于真实生态管理决策。';
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
      i<methodButtons.length;
      i++
    ){
      methodButtons[i].onclick=function(){
        automatic=false;

        method=this.getAttribute(
          'data-method'
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

    qualityToggle.onclick=function(){
      automatic=false;
      currentScenario='';

      if(method==='quadrat'){
        representativeSampling=
          !representativeSampling;
      }else{
        biasInput.value=
          Math.abs(
            Number(biasInput.value)-100
          )<=10
            ?'55'
            :'100';
      }

      update();
      schedule();
    };

    autoButton.onclick=function(){
      automatic=!automatic;
      update();
      schedule();
    };

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    populationInput.oninput=function(){
      currentScenario='';
      update();
    };

    quadratInput.oninput=function(){
      currentScenario='';
      update();
    };

    patchInput.oninput=function(){
      currentScenario='';
      update();
    };

    markedInput.oninput=function(){
      currentScenario='';
      update();
    };

    captureInput.oninput=function(){
      currentScenario='';
      update();
    };

    retentionInput.oninput=function(){
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
