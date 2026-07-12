/**
 * lifeScienceLabTemplatesBiotechnologyGeneticEngineering.ts
 *
 * 平面生命科学实验室：基因工程基本流程。
 *
 * 教学目标：
 * 1. 理解基因工程通常包括目的基因获取、载体选择、
 *    限制酶切割、DNA连接、重组载体形成、
 *    导入受体细胞、筛选与鉴定、表达检测等环节；
 * 2. 理解载体需要具备复制起点、筛选标记和适合插入外源片段的位置；
 * 3. 理解限制酶识别特定序列并切割DNA，
 *    相容末端有利于目的基因与载体连接；
 * 4. 理解DNA连接酶催化DNA片段之间形成磷酸二酯键；
 * 5. 理解连接体系中可能同时存在空载体、
 *    错误方向插入和正确重组载体；
 * 6. 理解导入成功率会影响获得候选受体细胞的数量；
 * 7. 区分筛选与鉴定：
 *    筛选用于富集候选细胞，鉴定用于确认是否获得正确构建；
 * 8. 理解构建成功不等于一定表达，
 *    表达还受启动子、受体细胞、阅读框和培养条件等因素影响；
 * 9. 比较正常构建、末端不相容、导入效率低和构建成功但不表达等情境。
 *
 * 教学边界：
 * 1. 本模型只用于基因工程基本原理教学；
 * 2. 本模型不提供限制酶、连接酶、培养基、
 *    温度、时间、浓度或操作步骤等真实实验参数；
 * 3. 图中的目的基因、质粒载体、受体细胞和蛋白质均为简化示意；
 * 4. 相容末端有利于连接，但不能保证所有载体都形成正确重组构建；
 * 5. 筛选标记只能帮助获得候选细胞，
 *    不能替代PCR、测序或其他鉴定方法；
 * 6. 导入受体细胞成功不等于目的基因已正确整合或稳定存在；
 * 7. 重组载体构建成功不等于目的基因一定能够转录和翻译；
 * 8. 表达结果还受启动子适配、插入方向、阅读框、
 *    受体细胞状态和蛋白质稳定性等因素影响；
 * 9. 本模型不提供微生物改造、病原体改造、
 *    人体基因治疗、胚胎编辑或临床应用方案；
 * 10. 本模型不用于疾病诊断、人体治疗、
 *     生物安全决策或真实实验设计。
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
function geneticEngineeringStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #86EFAC;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#ECFDF5);border-bottom:1px solid #86EFAC}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:258px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FAFFFC;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#16A34A}'
    + '#' + rootId + ' .ge-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .ge-stages{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ge-scenarios{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ge-button{min-height:32px;padding:3px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:8.7px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .ge-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.14)}'
    + '#' + rootId + ' .ge-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#4ADE80,#16A34A);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ge-auto.paused{background:#64748B}'
    + '#' + rootId + ' .ge-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#60A5FA,#2563EB);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ge-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ge-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .ge-card{padding:6px 3px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ge-card b{display:block;min-height:18px;font-size:12.5px;color:#15803D}'
    + '#' + rootId + ' .ge-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ge-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--ge-flow-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ge-pulse{animation:' + rootId + '-pulse 1.1s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ge-cell{animation:' + rootId + '-cell 1.55s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ge-protein{animation:' + rootId + '-protein 1.2s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.38}to{opacity:1}}'
    + '@keyframes ' + rootId + '-cell{from{transform:translateY(3px);opacity:.6}to{transform:translateY(-3px);opacity:1}}'
    + '@keyframes ' + rootId + '-protein{from{transform:scale(.88);opacity:.45}to{transform:scale(1.08);opacity:1}}'
    + '</style>'
}

/** 避免在外层模板字符串中直接写出脚本结束标签。 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_GENETIC_ENGINEERING:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-genetic-engineering-workflow',
    group: '🧬 现代生物技术',
    name: '基因工程基本流程',
    emoji: '🧬',
    desc: '观察目的基因获取、载体选择、酶切连接、重组载体、导入、筛选鉴定和表达结果',
    params: [
      {
        key: 'targetLength',
        label: '目的基因长度/bp',
        type: 'number',
        min: 300,
        max: 2400,
        step: 100,
        defaultValue: 900,
      },
      {
        key: 'vectorSize',
        label: '载体大小/bp',
        type: 'number',
        min: 2000,
        max: 10000,
        step: 500,
        defaultValue: 4500,
      },
      {
        key: 'ligationCompatibility',
        label: '末端相容与连接条件',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 88,
      },
      {
        key: 'transformationEfficiency',
        label: '导入效率示意',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'expressionCompatibility',
        label: '表达系统适配度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const targetLength = num(
        params,
        'targetLength',
        900,
      )
      const vectorSize = num(
        params,
        'vectorSize',
        4500,
      )
      const ligationCompatibility = num(
        params,
        'ligationCompatibility',
        88,
      )
      const transformationEfficiency = num(
        params,
        'transformationEfficiency',
        72,
      )
      const expressionCompatibility = num(
        params,
        'expressionCompatibility',
        78,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${geneticEngineeringStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧬 基因工程基本流程</div>
    <div class="bl-note">筛选不等于鉴定；重组载体构建成功不等于目的基因一定表达</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>目的基因长度</span>
          <span class="bl-value" data-target-value></span>
        </div>
        <input
          data-target
          type="range"
          min="300"
          max="2400"
          step="100"
          value="${n(targetLength)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>载体大小</span>
          <span class="bl-value" data-vector-value></span>
        </div>
        <input
          data-vector
          type="range"
          min="2000"
          max="10000"
          step="500"
          value="${n(vectorSize)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>末端相容与连接条件</span>
          <span class="bl-value" data-ligation-value></span>
        </div>
        <input
          data-ligation
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(ligationCompatibility)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>导入效率示意</span>
          <span class="bl-value" data-transform-value></span>
        </div>
        <input
          data-transform
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(transformationEfficiency)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>表达系统适配度</span>
          <span class="bl-value" data-expression-value></span>
        </div>
        <input
          data-expression
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(expressionCompatibility)}"
        >
      </div>

      <div class="ge-subtitle">基因工程流程</div>

      <div class="ge-stages">
        <button
          type="button"
          class="ge-button active"
          data-stage="obtain"
        >1. 获取基因</button>

        <button
          type="button"
          class="ge-button"
          data-stage="vector"
        >2. 选择载体</button>

        <button
          type="button"
          class="ge-button"
          data-stage="digest"
        >3. 限制酶切</button>

        <button
          type="button"
          class="ge-button"
          data-stage="ligate"
        >4. DNA连接</button>

        <button
          type="button"
          class="ge-button"
          data-stage="recombinant"
        >5. 重组载体</button>

        <button
          type="button"
          class="ge-button"
          data-stage="transform"
        >6. 导入细胞</button>

        <button
          type="button"
          class="ge-button"
          data-stage="screen"
        >7. 筛选鉴定</button>

        <button
          type="button"
          class="ge-button"
          data-stage="express"
        >8. 表达检测</button>
      </div>

      <div class="ge-subtitle">快速情境</div>

      <div class="ge-scenarios">
        <button
          type="button"
          class="ge-button active"
          data-scenario="standard"
        >标准流程</button>

        <button
          type="button"
          class="ge-button"
          data-scenario="incompatible"
        >末端不相容</button>

        <button
          type="button"
          class="ge-button"
          data-scenario="lowTransform"
        >导入效率低</button>

        <button
          type="button"
          class="ge-button"
          data-scenario="noExpression"
        >构建不表达</button>
      </div>

      <button
        type="button"
        class="ge-auto"
        data-auto
      >自动演示：运行中</button>

      <button
        type="button"
        class="ge-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <div class="ge-status">
        <div class="ge-card">
          <b data-status-stage></b>
          <span>当前环节</span>
        </div>

        <div class="ge-card">
          <b data-status-construct></b>
          <span>正确构建潜力</span>
        </div>

        <div class="ge-card">
          <b data-status-expression></b>
          <span>表达潜力</span>
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
        aria-label="基因工程基本流程互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-process"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#F0FDF4"/>
            <stop offset="100%" stop-color="#ECFDF5"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-outcome"
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
              fill="#16A34A"
            />
          </marker>

          <marker
            id="${rootId}-arrow-blue"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path
              d="M0,0 L0,6 L8,3 z"
              fill="#2563EB"
            />
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

        <rect
          width="760"
          height="430"
          fill="#FFFFFF"
        />

        <text
          x="22"
          y="33"
          data-title
          font-size="24"
          font-weight="900"
          fill="#166534"
        ></text>

        <text
          x="22"
          y="59"
          data-summary
          font-size="12.5"
          font-weight="800"
          fill="#475569"
        ></text>

        <g
          data-workflow-layer
        ></g>

        <g filter="url(#${rootId}-shadow)">
          <rect
            x="22"
            y="166"
            width="458"
            height="160"
            rx="20"
            fill="url(#${rootId}-process)"
            stroke="#86EFAC"
            stroke-width="3"
          />

          <rect
            x="500"
            y="166"
            width="238"
            height="160"
            rx="20"
            fill="url(#${rootId}-outcome)"
            stroke="#BFDBFE"
            stroke-width="3"
          />
        </g>

        <text
          x="251"
          y="190"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#166534"
        >当前环节结构示意</text>

        <text
          x="619"
          y="190"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#1D4ED8"
        >流程结果与质量判断</text>

        <g
          data-process-layer
        ></g>

        <g
          data-label-layer
        ></g>

        <g
          data-outcome-layer
        ></g>

        <g transform="translate(22 343)">
          <rect
            width="716"
            height="61"
            rx="16"
            fill="#F8FAFC"
            stroke="#CBD5E1"
            stroke-width="2"
          />

          <text
            x="16"
            y="21"
            data-metric-one-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="119"
            y="12"
            width="162"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-metric-one-bar
            x="119"
            y="12"
            width="0"
            height="13"
            rx="6.5"
            fill="#16A34A"
          />

          <text
            x="338"
            y="21"
            data-metric-two-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="448"
            y="12"
            width="240"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-metric-two-bar
            x="448"
            y="12"
            width="0"
            height="13"
            rx="6.5"
            fill="#2563EB"
          />

          <text
            x="16"
            y="48"
            data-metric-three-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="119"
            y="39"
            width="162"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-metric-three-bar
            x="119"
            y="39"
            width="0"
            height="13"
            rx="6.5"
            fill="#F59E0B"
          />

          <text
            x="338"
            y="48"
            data-panel-note
            font-size="10.5"
            font-weight="900"
            fill="#166534"
          ></text>
        </g>

        <text
          x="22"
          y="423"
          data-footer-note
          font-size="10.5"
          font-weight="900"
          fill="#14532D"
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

    var targetInput=root.querySelector(
      '[data-target]'
    );
    var vectorInput=root.querySelector(
      '[data-vector]'
    );
    var ligationInput=root.querySelector(
      '[data-ligation]'
    );
    var transformInput=root.querySelector(
      '[data-transform]'
    );
    var expressionInput=root.querySelector(
      '[data-expression]'
    );

    var targetValue=root.querySelector(
      '[data-target-value]'
    );
    var vectorValue=root.querySelector(
      '[data-vector-value]'
    );
    var ligationValue=root.querySelector(
      '[data-ligation-value]'
    );
    var transformValue=root.querySelector(
      '[data-transform-value]'
    );
    var expressionValue=root.querySelector(
      '[data-expression-value]'
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

    var statusStage=root.querySelector(
      '[data-status-stage]'
    );
    var statusConstruct=root.querySelector(
      '[data-status-construct]'
    );
    var statusExpression=root.querySelector(
      '[data-status-expression]'
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
    var workflowLayer=root.querySelector(
      '[data-workflow-layer]'
    );
    var processLayer=root.querySelector(
      '[data-process-layer]'
    );
    var labelLayer=root.querySelector(
      '[data-label-layer]'
    );
    var outcomeLayer=root.querySelector(
      '[data-outcome-layer]'
    );

    var metricOneLabel=root.querySelector(
      '[data-metric-one-label]'
    );
    var metricTwoLabel=root.querySelector(
      '[data-metric-two-label]'
    );
    var metricThreeLabel=root.querySelector(
      '[data-metric-three-label]'
    );
    var metricOneBar=root.querySelector(
      '[data-metric-one-bar]'
    );
    var metricTwoBar=root.querySelector(
      '[data-metric-two-bar]'
    );
    var metricThreeBar=root.querySelector(
      '[data-metric-three-bar]'
    );
    var panelNote=root.querySelector(
      '[data-panel-note]'
    );
    var footerNote=root.querySelector(
      '[data-footer-note]'
    );

    var stages=[
      'obtain',
      'vector',
      'digest',
      'ligate',
      'recombinant',
      'transform',
      'screen',
      'express'
    ];

    var stageNames={
      obtain:'目的基因获取',
      vector:'载体选择',
      digest:'限制酶切割',
      ligate:'DNA连接',
      recombinant:'重组载体',
      transform:'导入受体细胞',
      screen:'筛选与鉴定',
      express:'表达检测'
    };

    var shortStageNames={
      obtain:'获取',
      vector:'载体',
      digest:'酶切',
      ligate:'连接',
      recombinant:'重组',
      transform:'导入',
      screen:'鉴定',
      express:'表达'
    };

    var stageInfo={
      obtain:{
        title:'阶段1：获得目的基因',
        summary:'从基因组、已有模板或人工合成序列中获得目标DNA片段',
        note:'目的基因来源应与教学任务相匹配；本模型只展示目标DNA片段，不提供真实获取方法。'
      },
      vector:{
        title:'阶段2：选择合适载体',
        summary:'载体需要具备复制起点、筛选标记和适合插入目的基因的位置',
        note:'载体选择需要考虑受体细胞、插入片段大小、复制方式和后续表达目标。'
      },
      digest:{
        title:'阶段3：限制酶切割',
        summary:'限制酶识别特定序列并切割目的基因与载体，形成可连接末端',
        note:'使用相同或能够产生相容末端的切割方式，有利于目的基因与载体连接。'
      },
      ligate:{
        title:'阶段4：DNA连接',
        summary:'DNA连接酶催化目的基因片段与载体DNA形成稳定连接',
        note:'连接体系中可能同时形成正确插入、错误方向插入、空载体和其他副产物。'
      },
      recombinant:{
        title:'阶段5：形成重组载体',
        summary:'目的基因被插入载体后形成重组DNA分子',
        note:'观察到环状重组载体只是结构示意，正确构建仍需进一步鉴定确认。'
      },
      transform:{
        title:'阶段6：导入受体细胞',
        summary:'把重组载体导入适合的受体细胞，获得候选转化细胞',
        note:'导入效率影响候选细胞数量，但导入成功不等于构建一定正确或稳定存在。'
      },
      screen:{
        title:'阶段7：筛选与鉴定',
        summary:'筛选用于富集候选细胞，鉴定用于确认插入片段、方向和序列是否正确',
        note:'筛选不等于鉴定；筛选阳性的候选细胞中仍可能存在空载体或错误构建。'
      },
      express:{
        title:'阶段8：检测目的基因表达',
        summary:'检测目的基因是否完成转录、翻译并形成预期产物',
        note:'构建成功不等于一定表达，表达还受启动子、阅读框、受体细胞和蛋白质稳定性影响。'
      }
    };

    var scenarios={
      standard:{
        ligation:88,
        transform:72,
        expression:78
      },
      incompatible:{
        ligation:28,
        transform:72,
        expression:78
      },
      lowTransform:{
        ligation:88,
        transform:22,
        expression:78
      },
      noExpression:{
        ligation:88,
        transform:72,
        expression:14
      }
    };

    var stage='obtain';
    var currentScenario='standard';
    var automatic=true;
    var showLabels=${showLabels ? 'true' : 'false'};
    var timer=null;

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

    function constructionModel(
      targetLength,
      vectorSize,
      ligation,
      transformation,
      expression
    ){
      var sizeRatio=
        targetLength
        /Math.max(
          1,
          vectorSize
        );

      var sizeBalance=
        Math.exp(
          -Math.pow(
            (sizeRatio-.2)/.22,
            2
          )
        );

      var endCompatibility=
        ligation/100;

      var constructPotential=
        100
        *endCompatibility
        *(.68+.32*sizeBalance);

      var transformedPotential=
        constructPotential
        *transformation/100;

      var screenedPotential=
        transformedPotential
        *(.55+.35*endCompatibility);

      var expressionPotential=
        screenedPotential
        *expression/100;

      return {
        sizeRatio:sizeRatio,
        sizeBalance:sizeBalance,
        constructPotential:clamp(
          constructPotential,
          0,
          100
        ),
        transformedPotential:clamp(
          transformedPotential,
          0,
          100
        ),
        screenedPotential:clamp(
          screenedPotential,
          0,
          100
        ),
        expressionPotential:clamp(
          expressionPotential,
          0,
          100
        )
      };
    }

    function drawWorkflow(){
      var html='';
      var positions=[
        {x:72,y:91},
        {x:230,y:91},
        {x:388,y:91},
        {x:546,y:91},
        {x:72,y:137},
        {x:230,y:137},
        {x:388,y:137},
        {x:546,y:137}
      ];

      for(var i=0;i<positions.length;i++){
        var currentStage=
          stages[i];

        var position=
          positions[i];

        var active=
          currentStage===stage;

        html+='<rect x="'+position.x
          +'" y="'+position.y
          +'" width="130" height="31"'
          +' rx="10" fill="'
          +(active?'#16A34A':'#F0FDF4')
          +'" stroke="'
          +(active?'#15803D':'#86EFAC')
          +'" stroke-width="'
          +(active?'3':'2')
          +'"/>';

        html+='<text x="'+(position.x+65)
          +'" y="'+(position.y+20)
          +'" text-anchor="middle"'
          +' font-size="10"'
          +' font-weight="900"'
          +' fill="'
          +(active?'#FFFFFF':'#166534')
          +'">'
          +(i+1)+'. '
          +shortStageNames[currentStage]
          +'</text>';

        if(i<3){
          html+='<path class="ge-flow"'
            +' d="M'+(position.x+132)
            +' '+(position.y+15)
            +' H'+(positions[i+1].x-4)
            +'" fill="none"'
            +' stroke="#16A34A"'
            +' stroke-width="2.5"'
            +' marker-end="url(#${rootId}-arrow)"/>';
        }

        if(i>=4 && i<7){
          html+='<path class="ge-flow"'
            +' d="M'+(position.x+132)
            +' '+(position.y+15)
            +' H'+(positions[i+1].x-4)
            +'" fill="none"'
            +' stroke="#16A34A"'
            +' stroke-width="2.5"'
            +' marker-end="url(#${rootId}-arrow)"/>';
        }
      }

      html+='<path class="ge-flow"'
        +' d="M676 107'
        +' C708 107 708 151 676 151'
        +' H205"'
        +' fill="none"'
        +' stroke="#16A34A"'
        +' stroke-width="2.5"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      workflowLayer.innerHTML=html;
    }

    function drawLinearDna(
      x,
      y,
      width,
      colorOne,
      colorTwo,
      withInsert
    ){
      var html='';

      html+='<line x1="'+x
        +'" y1="'+y
        +'" x2="'+(x+width)
        +'" y2="'+y
        +'" stroke="'+colorOne
        +'" stroke-width="7"'
        +' stroke-linecap="round"/>';

      html+='<line x1="'+x
        +'" y1="'+(y+25)
        +'" x2="'+(x+width)
        +'" y2="'+(y+25)
        +'" stroke="'+colorTwo
        +'" stroke-width="7"'
        +' stroke-linecap="round"/>';

      if(withInsert){
        html+='<rect x="'+(x+width*.28)
          +'" y="'+(y-7)
          +'" width="'+(width*.44)
          +'" height="39"'
          +' rx="8" fill="#FDE68A"'
          +' stroke="#D97706"'
          +' stroke-width="3"/>';

        html+='<text x="'+(x+width*.5)
          +'" y="'+(y+17)
          +'" text-anchor="middle"'
          +' font-size="10"'
          +' font-weight="900"'
          +' fill="#92400E">'
          +'目的基因'
          +'</text>';
      }

      return html;
    }

    function plasmid(
      cx,
      cy,
      radius,
      insertFraction,
      highlight
    ){
      var html='';

      html+='<circle cx="'+cx
        +'" cy="'+cy
        +'" r="'+radius
        +'" fill="none"'
        +' stroke="#2563EB"'
        +' stroke-width="10"'
        +' opacity=".9"/>';

      html+='<path d="M'
        +(cx-radius*.75)+' '
        +(cy-radius*.66)
        +' A'+radius+' '+radius
        +' 0 0 1 '
        +(cx+radius*.38)+' '
        +(cy-radius*.93)
        +'" fill="none"'
        +' stroke="#F59E0B"'
        +' stroke-width="12"'
        +' stroke-linecap="round"'
        +(highlight
          ?' class="ge-pulse"'
          :'')
        +'/>';

      html+='<path d="M'
        +(cx+radius*.82)+' '
        +(cy+radius*.4)
        +' A'+radius+' '+radius
        +' 0 0 1 '
        +(cx+radius*.18)+' '
        +(cy+radius*.98)
        +'" fill="none"'
        +' stroke="#10B981"'
        +' stroke-width="12"'
        +' stroke-linecap="round"/>';

      if(insertFraction>0){
        html+='<path d="M'
          +(cx-radius*.12)+' '
          +(cy-radius*.99)
          +' A'+radius+' '+radius
          +' 0 0 1 '
          +(cx+radius*.56)+' '
          +(cy-radius*.83)
          +'" fill="none"'
          +' stroke="#EC4899"'
          +' stroke-width="14"'
          +' stroke-linecap="round"'
          +' class="ge-pulse"/>';
      }

      return html;
    }

    function drawObtain(){
      var html='';

      html+=drawLinearDna(
        75,
        222,
        315,
        '#2563EB',
        '#EC4899',
        true
      );

      html+='<path class="ge-flow"'
        +' d="M232 207 V192"'
        +' fill="none"'
        +' stroke="#16A34A"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      html+='<rect x="164" y="267"'
        +' width="136" height="31"'
        +' rx="10" fill="#FFFFFF"'
        +' stroke="#86EFAC"'
        +' stroke-width="2"/>';

      html+='<text x="232" y="287"'
        +' text-anchor="middle"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#166534">'
        +'获得目标DNA片段'
        +'</text>';

      return html;
    }

    function drawVector(){
      var html='';

      html+=plasmid(
        250,
        247,
        55,
        0,
        true
      );

      html+='<text x="250" y="251"'
        +' text-anchor="middle"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'质粒载体'
        +'</text>';

      html+='<text x="87" y="219"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#D97706">'
        +'复制起点'
        +'</text>';

      html+='<path d="M153 218'
        +' C177 219 186 223 194 228"'
        +' fill="none"'
        +' stroke="#D97706"'
        +' stroke-width="2"/>';

      html+='<text x="331" y="219"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#047857">'
        +'筛选标记'
        +'</text>';

      html+='<path d="M329 223'
        +' C313 232 306 244 305 258"'
        +' fill="none"'
        +' stroke="#047857"'
        +' stroke-width="2"/>';

      html+='<text x="171" y="309"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#BE185D">'
        +'插入位点'
        +'</text>';

      return html;
    }

    function drawDigest(
      compatibility
    ){
      var html='';

      html+=drawLinearDna(
        62,
        208,
        146,
        '#2563EB',
        '#EC4899',
        true
      );

      html+='<path d="M195 202'
        +' L213 216'
        +' M195 241'
        +' L213 227"'
        +' stroke="#DC2626"'
        +' stroke-width="4"/>';

      html+='<path d="M287 204'
        +' C258 204 242 221 242 246'
        +' C242 282 271 300 306 300"'
        +' fill="none"'
        +' stroke="#2563EB"'
        +' stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+='<path d="M306 300'
        +' C345 300 377 274 377 240'
        +' C377 205 347 190 320 191"'
        +' fill="none"'
        +' stroke="#2563EB"'
        +' stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+='<path d="M286 194'
        +' L303 207'
        +' M321 184'
        +' L304 207"'
        +' stroke="#DC2626"'
        +' stroke-width="4"/>';

      html+='<ellipse cx="232" cy="216"'
        +' rx="29" ry="21"'
        +' fill="#FEE2E2"'
        +' stroke="#DC2626"'
        +' stroke-width="3"/>';

      html+='<text x="232" y="220"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#991B1B">'
        +'限制酶'
        +'</text>';

      html+='<text x="280" y="318"'
        +' text-anchor="middle"'
        +' font-size="10.5"'
        +' font-weight="900"'
        +' fill="'
        +(compatibility>=60
          ?'#047857'
          :'#B91C1C')
        +'">'
        +(compatibility>=60
          ?'形成相对相容末端'
          :'末端相容性较低')
        +'</text>';

      return html;
    }

    function drawLigate(
      compatibility
    ){
      var html='';

      html+='<path d="M91 242'
        +' H170"'
        +' stroke="#2563EB"'
        +' stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+='<path d="M295 242'
        +' H374"'
        +' stroke="#2563EB"'
        +' stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+='<rect x="174" y="224"'
        +' width="117" height="36"'
        +' rx="9" fill="#FCE7F3"'
        +' stroke="#EC4899"'
        +' stroke-width="4"'
        +(compatibility>=60
          ?' class="ge-pulse"'
          :' opacity=".42"')
        +'/>';

      html+='<text x="232" y="247"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#9D174D">'
        +'目的基因片段'
        +'</text>';

      html+='<ellipse cx="232" cy="288"'
        +' rx="42" ry="23"'
        +' fill="#FEF3C7"'
        +' stroke="#D97706"'
        +' stroke-width="3"/>';

      html+='<text x="232" y="292"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#92400E">'
        +'DNA连接酶'
        +'</text>';

      html+='<path class="ge-flow"'
        +' d="M232 275 V260"'
        +' fill="none"'
        +' stroke="#16A34A"'
        +' stroke-width="3"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      return html;
    }

    function drawRecombinant(
      constructPotential
    ){
      var html='';

      html+=plasmid(
        232,
        248,
        61,
        1,
        constructPotential>=55
      );

      html+='<text x="232" y="244"'
        +' text-anchor="middle"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'重组载体'
        +'</text>';

      html+='<text x="232" y="260"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'结构示意'
        +'</text>';

      html+='<rect x="330" y="213"'
        +' width="105" height="70"'
        +' rx="12" fill="#FFFFFF"'
        +' stroke="#CBD5E1"'
        +' stroke-width="2"/>';

      html+='<text x="382" y="231"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#475569">'
        +'可能产物'
        +'</text>';

      html+='<text x="382" y="249"'
        +' text-anchor="middle"'
        +' font-size="9"'
        +' font-weight="800"'
        +' fill="#047857">'
        +'正确插入'
        +'</text>';

      html+='<text x="382" y="265"'
        +' text-anchor="middle"'
        +' font-size="9"'
        +' font-weight="800"'
        +' fill="#B45309">'
        +'空载体／错误方向'
        +'</text>';

      return html;
    }

    function cell(
      x,
      y,
      hasPlasmid,
      active
    ){
      var html='';

      html+='<ellipse'
        +(active?' class="ge-cell"':'')
        +' cx="'+x
        +'" cy="'+y
        +'" rx="42" ry="27"'
        +' fill="#D1FAE5"'
        +' stroke="#059669"'
        +' stroke-width="3"/>';

      html+='<path d="M'+(x+39)+' '+(y-6)
        +' C'+(x+62)+' '+(y-22)+' '
        +(x+67)+' '+(y+8)+' '
        +(x+49)+' '+(y+17)
        +'" fill="none"'
        +' stroke="#059669"'
        +' stroke-width="3"/>';

      if(hasPlasmid){
        html+='<circle cx="'+x
          +'" cy="'+y
          +'" r="10" fill="none"'
          +' stroke="#2563EB"'
          +' stroke-width="3"/>';

        html+='<path d="M'+(x-2)+' '+(y-10)
          +' A10 10 0 0 1 '
          +(x+8)+' '+(y-6)
          +'" fill="none"'
          +' stroke="#EC4899"'
          +' stroke-width="4"/>';
      }

      return html;
    }

    function drawTransform(
      transformedPotential
    ){
      var html='';

      html+=plasmid(
        112,
        248,
        30,
        1,
        true
      );

      html+='<path class="ge-flow"'
        +' d="M150 248 H197"'
        +' fill="none"'
        +' stroke="#16A34A"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      var cellCount=
        transformedPotential>=55
          ?4
          :transformedPotential>=25
            ?2
            :1;

      for(var i=0;i<4;i++){
        var x=
          258+(i%2)*105;

        var y=
          220+Math.floor(i/2)*65;

        html+=cell(
          x,
          y,
          i<cellCount,
          i<cellCount
        );
      }

      return html;
    }

    function drawScreen(
      screenedPotential
    ){
      var html='';

      html+='<ellipse cx="232" cy="251"'
        +' rx="172" ry="66"'
        +' fill="#FEF3C7"'
        +' stroke="#D97706"'
        +' stroke-width="4"/>';

      var colonyCount=
        Math.max(
          3,
          Math.round(
            5+screenedPotential/7
          )
        );

      for(var i=0;i<colonyCount;i++){
        var angle=
          i*2.399;

        var radius=
          20+(i%4)*11;

        var x=
          232+Math.cos(angle)*radius*1.55;

        var y=
          251+Math.sin(angle)*radius*.72;

        var candidate=
          i<Math.round(
            colonyCount
            *screenedPotential/100
          );

        html+='<circle'
          +(candidate
            ?' class="ge-pulse"'
            :'')
          +' cx="'+x
          +'" cy="'+y
          +'" r="'+(5+i%3)
          +'" fill="'
          +(candidate?'#16A34A':'#94A3B8')
          +'" opacity=".85"/>';
      }

      html+='<text x="232" y="327"'
        +' text-anchor="middle"'
        +' font-size="10.5"'
        +' font-weight="900"'
        +' fill="#92400E">'
        +'绿色仅表示候选细胞，仍需进一步鉴定'
        +'</text>';

      return html;
    }

    function drawExpress(
      expressionPotential
    ){
      var html='';

      html+=cell(
        135,
        248,
        true,
        true
      );

      html+='<rect x="205" y="211"'
        +' width="95" height="30"'
        +' rx="9" fill="'
        +(expressionPotential>=45
          ?'#DCFCE7'
          :'#FEE2E2')
        +'" stroke="'
        +(expressionPotential>=45
          ?'#16A34A'
          :'#DC2626')
        +'" stroke-width="3"/>';

      html+='<text x="252" y="231"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="'
        +(expressionPotential>=45
          ?'#166534'
          :'#991B1B')
        +'">'
        +(expressionPotential>=45
          ?'转录与翻译'
          :'表达受限')
        +'</text>';

      html+='<path class="ge-flow"'
        +' d="M178 248 H204"'
        +' fill="none"'
        +' stroke="#2563EB"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>';

      var proteinCount=
        Math.round(
          expressionPotential/9
        );

      for(var i=0;i<proteinCount;i++){
        var x=
          330+(i%5)*24;

        var y=
          215+Math.floor(i/5)*34;

        html+='<circle class="ge-protein"'
          +' cx="'+x
          +'" cy="'+y
          +'" r="'+(7+i%3)
          +'" fill="'
          +(i%2===0
            ?'#8B5CF6'
            :'#F59E0B')
          +'" opacity=".9"/>';

        html+='<text x="'+x
          +'" y="'+(y+3)
          +'" text-anchor="middle"'
          +' font-size="7"'
          +' font-weight="900"'
          +' fill="#FFFFFF">'
          +'P'
          +'</text>';
      }

      if(proteinCount===0){
        html+='<text x="365" y="252"'
          +' text-anchor="middle"'
          +' font-size="11"'
          +' font-weight="900"'
          +' fill="#B91C1C">'
          +'未形成可见表达产物'
          +'</text>';
      }

      return html;
    }

    function drawProcess(
      model,
      ligation
    ){
      var html='';

      if(stage==='obtain'){
        html=drawObtain();
      }else if(stage==='vector'){
        html=drawVector();
      }else if(stage==='digest'){
        html=drawDigest(
          ligation
        );
      }else if(stage==='ligate'){
        html=drawLigate(
          ligation
        );
      }else if(stage==='recombinant'){
        html=drawRecombinant(
          model.constructPotential
        );
      }else if(stage==='transform'){
        html=drawTransform(
          model.transformedPotential
        );
      }else if(stage==='screen'){
        html=drawScreen(
          model.screenedPotential
        );
      }else{
        html=drawExpress(
          model.expressionPotential
        );
      }

      processLayer.innerHTML=html;
    }

    function drawLabels(){
      if(!showLabels){
        labelLayer.innerHTML='';
        return;
      }

      var html='';

      if(stage==='obtain'){
        html+='<text x="78" y="211"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#1D4ED8">'
          +'5′'
          +'</text>';

        html+='<text x="383" y="211"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#1D4ED8">'
          +'3′'
          +'</text>';
      }else if(stage==='vector'){
        html+='<text x="247" y="207"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#BE185D">'
          +'多克隆或插入区域'
          +'</text>';
      }else if(stage==='digest'){
        html+='<text x="232" y="199"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#B91C1C">'
          +'识别特定序列并切割'
          +'</text>';
      }else if(stage==='ligate'){
        html+='<text x="232" y="209"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#166534">'
          +'相容末端对接'
          +'</text>';
      }else if(stage==='recombinant'){
        html+='<text x="232" y="322"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#BE185D">'
          +'粉色弧段表示插入的目的基因'
          +'</text>';
      }else if(stage==='transform'){
        html+='<text x="112" y="298"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#1D4ED8">'
          +'重组载体'
          +'</text>';
      }else if(stage==='screen'){
        html+='<text x="232" y="205"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#92400E">'
          +'筛选平板或候选群体示意'
          +'</text>';
      }else{
        html+='<text x="365" y="315"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#6D28D9">'
          +'P表示目的蛋白质产物示意'
          +'</text>';
      }

      labelLayer.innerHTML=html;
    }

    function outcomeJudgement(
      model,
      ligation,
      transformation,
      expression
    ){
      var constructState=
        model.constructPotential>=70
          ?'构建潜力较高'
          :model.constructPotential>=40
            ?'构建潜力一般'
            :'构建潜力较低';

      var transformState=
        model.transformedPotential>=50
          ?'候选细胞较多'
          :model.transformedPotential>=20
            ?'候选细胞较少'
            :'导入明显受限';

      var expressionState=
        model.expressionPotential>=55
          ?'表达潜力较高'
          :model.expressionPotential>=25
            ?'表达潜力有限'
            :'表达可能失败';

      var keyIssue='当前各环节相对协调。';

      if(ligation<45){
        keyIssue=
          '末端相容或连接条件不足，是当前构建的主要限制。';
      }else if(transformation<35){
        keyIssue=
          '导入效率较低，获得候选受体细胞的数量受到限制。';
      }else if(expression<35){
        keyIssue=
          '载体可能已经构建并导入，但表达系统适配度不足。';
      }

      return {
        constructState:constructState,
        transformState:transformState,
        expressionState:expressionState,
        keyIssue:keyIssue
      };
    }

    function drawOutcome(
      model,
      ligation,
      transformation,
      expression
    ){
      var judgement=
        outcomeJudgement(
          model,
          ligation,
          transformation,
          expression
        );

      var html='';

      html+='<text x="519" y="218"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'重组载体'
        +'</text>';

      html+='<text x="715" y="218"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#166534">'
        +judgement.constructState
        +'</text>';

      html+='<rect x="519" y="226"'
        +' width="196" height="12"'
        +' rx="6" fill="#E2E8F0"/>';

      html+='<rect x="519" y="226"'
        +' width="'+(196*model.constructPotential/100)
        +'" height="12" rx="6"'
        +' fill="#16A34A"/>';

      html+='<text x="519" y="258"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'候选细胞'
        +'</text>';

      html+='<text x="715" y="258"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +judgement.transformState
        +'</text>';

      html+='<rect x="519" y="266"'
        +' width="196" height="12"'
        +' rx="6" fill="#E2E8F0"/>';

      html+='<rect x="519" y="266"'
        +' width="'+(196*model.transformedPotential/100)
        +'" height="12" rx="6"'
        +' fill="#2563EB"/>';

      html+='<text x="519" y="298"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'表达产物'
        +'</text>';

      html+='<text x="715" y="298"'
        +' text-anchor="end"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="'
        +(model.expressionPotential>=25
          ?'#7C3AED'
          :'#B91C1C')
        +'">'
        +judgement.expressionState
        +'</text>';

      html+='<rect x="519" y="306"'
        +' width="196" height="12"'
        +' rx="6" fill="#E2E8F0"/>';

      html+='<rect x="519" y="306"'
        +' width="'+(196*model.expressionPotential/100)
        +'" height="12" rx="6"'
        +' fill="#8B5CF6"/>';

      outcomeLayer.innerHTML=html;

      return judgement;
    }

    function stageStatusText(){
      return shortStageNames[stage];
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

      ligationInput.value=
        String(data.ligation);

      transformInput.value=
        String(data.transform);

      expressionInput.value=
        String(data.expression);

      currentScenario=name;

      if(pauseAutomatic){
        automatic=false;
      }

      update();
    }

    function update(){
      var targetLength=
        clamp(
          Math.round(
            Number(targetInput.value)
          ),
          300,
          2400
        );

      var vectorSize=
        clamp(
          Math.round(
            Number(vectorInput.value)
          ),
          2000,
          10000
        );

      var ligation=
        clamp(
          Number(ligationInput.value),
          20,
          100
        );

      var transformation=
        clamp(
          Number(transformInput.value),
          10,
          100
        );

      var expression=
        clamp(
          Number(expressionInput.value),
          10,
          100
        );

      var model=
        constructionModel(
          targetLength,
          vectorSize,
          ligation,
          transformation,
          expression
        );

      var info=
        stageInfo[stage];

      targetValue.textContent=
        targetLength+' bp';

      vectorValue.textContent=
        vectorSize+' bp';

      ligationValue.textContent=
        ligation.toFixed(0)+'%';

      transformValue.textContent=
        transformation.toFixed(0)+'%';

      expressionValue.textContent=
        expression.toFixed(0)+'%';

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
          ?'结构标注：显示'
          :'结构标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      root.style.setProperty(
        '--ge-flow-speed',
        clamp(
          2.7-ligation/60,
          .62,
          2.5
        ).toFixed(2)+'s'
      );

      title.textContent=
        info.title;

      summary.textContent=
        info.summary;

      statusStage.textContent=
        stageStatusText();

      statusConstruct.textContent=
        model.constructPotential.toFixed(0)
        +'%';

      statusExpression.textContent=
        model.expressionPotential.toFixed(0)
        +'%';

      drawWorkflow();

      drawProcess(
        model,
        ligation
      );

      drawLabels();

      var judgement=
        drawOutcome(
          model,
          ligation,
          transformation,
          expression
        );

      metricOneLabel.textContent=
        '连接与构建潜力';

      metricTwoLabel.textContent=
        '导入后候选细胞';

      metricThreeLabel.textContent=
        '表达系统适配度';

      metricOneBar.setAttribute(
        'width',
        String(
          162
          *model.constructPotential/100
        )
      );

      metricTwoBar.setAttribute(
        'width',
        String(
          240
          *model.transformedPotential/100
        )
      );

      metricThreeBar.setAttribute(
        'width',
        String(
          162
          *expression/100
        )
      );

      panelNote.textContent=
        judgement.keyIssue;

      footerNote.textContent=
        '流程逻辑：获得目的基因 → 构建重组载体 → 导入受体细胞 → 筛选鉴定 → 检测表达。';

      result.innerHTML=
        info.note
        +'<br>'+judgement.keyIssue
        +' 筛选只能获得候选细胞，不能替代构建鉴定。'
        +' 构建成功不等于一定表达。'
        +' 本模型不提供真实实验参数，也不用于人体治疗、病原体改造或临床方案。';
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
        1900
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
      j<scenarioButtons.length;
      j++
    ){
      scenarioButtons[j].onclick=function(){
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

    targetInput.oninput=function(){
      currentScenario='';
      update();
    };

    vectorInput.oninput=function(){
      currentScenario='';
      update();
    };

    ligationInput.oninput=function(){
      currentScenario='';
      update();
      schedule();
    };

    transformInput.oninput=function(){
      currentScenario='';
      update();
    };

    expressionInput.oninput=function(){
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
