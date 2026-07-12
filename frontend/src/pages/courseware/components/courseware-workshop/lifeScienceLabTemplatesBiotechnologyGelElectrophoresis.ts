/**
 * lifeScienceLabTemplatesBiotechnologyGelElectrophoresis.ts
 *
 * 平面生命科学实验室：凝胶电泳与条带判读。
 *
 * 教学目标：
 * 1. 理解DNA分子整体带负电，在电场中通常由负极一侧向正极方向迁移；
 * 2. 理解凝胶具有分子筛作用，在其他条件相同时，
 *    较小的DNA片段通常迁移得更远；
 * 3. 认识上样孔、泳道、DNA分子量标准和样品条带；
 * 4. 观察电压、运行时间、凝胶浓度和上样量对迁移与分离效果的影响；
 * 5. 使用DNA分子量标准对样品条带大小进行相对估算；
 * 6. 理解条带亮度只能在条件相近时粗略反映相对DNA含量；
 * 7. 识别弱条带、过量上样、拖尾、非特异条带和对照污染等异常示意；
 * 8. 区分正常样品泳道、分子量标准泳道和阴性对照泳道。
 *
 * 教学边界：
 * 1. 本模型只用于凝胶电泳原理和条带判读教学；
 * 2. 所有DNA片段大小、迁移距离和条带亮度均为教学模拟值；
 * 3. 较小片段通常迁移更远，但迁移还受凝胶类型、浓度、
 *    电场强度、DNA构象和缓冲体系等因素影响；
 * 4. 电压过高可能造成发热、条带弯曲或分辨率下降；
 * 5. 运行时间过短可能分离不足，过长可能使小片段跑出凝胶；
 * 6. 凝胶浓度升高通常更利于分辨较小片段，
 *    但会减慢较大片段迁移；
 * 7. 上样过量可能形成过亮、过宽、拖尾或涂抹状信号；
 * 8. 阴性对照理论上不应出现目标条带，
 *    若出现条带应排查污染、非特异扩增或样品混入；
 * 9. 条带位置不能直接证明样品身份，
 *    条带亮度也不能直接等同精确定量结果；
 * 10. 本模型不用于亲子鉴定、疾病诊断、病原体检测、
 *     法医学鉴定或个人身份判断。
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
function gelElectrophoresisStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #67E8F9;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#CFFAFE,#ECFEFF);border-bottom:1px solid #67E8F9}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#0E7490}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:256px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F7FEFF;border-right:1px solid #A5F3FC}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0891B2;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0891B2}'
    + '#' + rootId + ' .ge-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#0E7490}'
    + '#' + rootId + ' .ge-stages{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ge-lanes{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ge-scenarios{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ge-button{min-height:31px;padding:3px;border:1px solid #67E8F9;border-radius:8px;background:#fff;color:#0E7490;font-size:8.9px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .ge-button.active{border-color:#0891B2;background:#CFFAFE;box-shadow:0 3px 9px rgba(8,145,178,.14)}'
    + '#' + rootId + ' .ge-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#22D3EE,#0891B2);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ge-auto.paused{background:#64748B}'
    + '#' + rootId + ' .ge-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ge-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ge-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .ge-card{padding:6px 3px;border:1px solid #A5F3FC;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ge-card b{display:block;min-height:18px;font-size:12.5px;color:#0E7490}'
    + '#' + rootId + ' .ge-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#CFFAFE;color:#164E63;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ge-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--ge-flow-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ge-band{animation:' + rootId + '-band 1.1s ease-in-out infinite alternate}'
    + '#' + rootId + ' .ge-pulse{animation:' + rootId + '-pulse 1.25s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-band{from{opacity:.56}to{opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{transform:translateY(2px);opacity:.48}to{transform:translateY(-3px);opacity:1}}'
    + '</style>'
}

/** 避免在外层模板字符串中直接写出脚本结束标签。 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_GEL_ELECTROPHORESIS:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-gel-electrophoresis-band-reading',
    group: '🧬 现代生物技术',
    name: '凝胶电泳与条带判读',
    emoji: '🧫',
    desc: '观察DNA由负极向正极迁移，比较片段大小、分子量标准、泳道条带、亮度、拖尾和污染',
    params: [
      {
        key: 'voltage',
        label: '电场强度示意/V',
        type: 'number',
        min: 40,
        max: 160,
        step: 5,
        defaultValue: 90,
        hint: '电压过高可能降低分辨率并增加发热',
      },
      {
        key: 'runTime',
        label: '运行时间/min',
        type: 'number',
        min: 5,
        max: 60,
        step: 1,
        defaultValue: 28,
      },
      {
        key: 'gelConcentration',
        label: '凝胶浓度/%',
        type: 'number',
        min: 0.8,
        max: 3,
        step: 0.1,
        defaultValue: 1.2,
      },
      {
        key: 'sampleAmount',
        label: '相对上样量',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 65,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const voltage = num(params, 'voltage', 90)
      const runTime = num(params, 'runTime', 28)
      const gelConcentration = num(
        params,
        'gelConcentration',
        1.2,
      )
      const sampleAmount = num(
        params,
        'sampleAmount',
        65,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${gelElectrophoresisStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧫 凝胶电泳与条带判读</div>
    <div class="bl-note">DNA通常由负极一侧向正极迁移；小片段通常迁移更远</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>电场强度示意</span>
          <span class="bl-value" data-voltage-value></span>
        </div>
        <input
          data-voltage
          type="range"
          min="40"
          max="160"
          step="5"
          value="${n(voltage)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>运行时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input
          data-time
          type="range"
          min="5"
          max="60"
          step="1"
          value="${n(runTime)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>凝胶浓度</span>
          <span class="bl-value" data-gel-value></span>
        </div>
        <input
          data-gel
          type="range"
          min="0.8"
          max="3"
          step="0.1"
          value="${n(gelConcentration)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>相对上样量</span>
          <span class="bl-value" data-amount-value></span>
        </div>
        <input
          data-amount
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(sampleAmount)}"
        >
      </div>

      <div class="ge-subtitle">电泳阶段</div>

      <div class="ge-stages">
        <button
          type="button"
          class="ge-button active"
          data-stage="loading"
        >1. 上样</button>

        <button
          type="button"
          class="ge-button"
          data-stage="migration"
        >2. 迁移</button>

        <button
          type="button"
          class="ge-button"
          data-stage="separation"
        >3. 分离</button>

        <button
          type="button"
          class="ge-button"
          data-stage="interpretation"
        >4. 判读</button>
      </div>

      <div class="ge-subtitle">选择泳道</div>

      <div class="ge-lanes">
        <button
          type="button"
          class="ge-button"
          data-lane="ladder"
        >M 标准</button>

        <button
          type="button"
          class="ge-button active"
          data-lane="sampleA"
        >样品A</button>

        <button
          type="button"
          class="ge-button"
          data-lane="sampleB"
        >样品B</button>

        <button
          type="button"
          class="ge-button"
          data-lane="negative"
        >阴性对照</button>
      </div>

      <div class="ge-subtitle">快速情境</div>

      <div class="ge-scenarios">
        <button
          type="button"
          class="ge-button active"
          data-scenario="standard"
        >标准分离</button>

        <button
          type="button"
          class="ge-button"
          data-scenario="weak"
        >弱条带</button>

        <button
          type="button"
          class="ge-button"
          data-scenario="overload"
        >过量拖尾</button>

        <button
          type="button"
          class="ge-button"
          data-scenario="contamination"
        >对照污染</button>
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
          <span>当前阶段</span>
        </div>

        <div class="ge-card">
          <b data-status-distance></b>
          <span>最远迁移示意</span>
        </div>

        <div class="ge-card">
          <b data-status-result></b>
          <span>当前泳道判断</span>
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
        aria-label="凝胶电泳与条带判读互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-gel"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#ECFEFF"/>
            <stop offset="100%" stop-color="#CFFAFE"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-panel"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#F8FAFC"/>
            <stop offset="100%" stop-color="#F0FDFA"/>
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
              fill="#0891B2"
            />
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#164E63"
              flood-opacity=".13"
            />
          </filter>

          <filter id="${rootId}-glow">
            <feGaussianBlur
              stdDeviation="2.5"
              result="blur"
            />
            <feMerge>
              <feMergeNode in="blur"/>
              <feMergeNode in="SourceGraphic"/>
            </feMerge>
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
          fill="#0E7490"
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
            width="440"
            height="240"
            rx="20"
            fill="url(#${rootId}-gel)"
            stroke="#67E8F9"
            stroke-width="3"
          />

          <rect
            x="482"
            y="79"
            width="256"
            height="240"
            rx="20"
            fill="url(#${rootId}-panel)"
            stroke="#99F6E4"
            stroke-width="3"
          />
        </g>

        <text
          x="242"
          y="103"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#0E7490"
        >凝胶、泳道与DNA条带</text>

        <text
          x="610"
          y="103"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#047857"
        >当前泳道相对判读</text>

        <g data-gel-layer></g>
        <g data-electrode-layer></g>
        <g data-label-layer></g>
        <g data-interpretation-layer></g>

        <g transform="translate(22 337)">
          <rect
            width="716"
            height="65"
            rx="16"
            fill="#F8FAFC"
            stroke="#CBD5E1"
            stroke-width="2"
          />

          <text
            x="16"
            y="22"
            data-metric-one-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="118"
            y="13"
            width="165"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-metric-one-bar
            x="118"
            y="13"
            width="0"
            height="13"
            rx="6.5"
            fill="#0891B2"
          />

          <text
            x="342"
            y="22"
            data-metric-two-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="450"
            y="13"
            width="238"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-metric-two-bar
            x="450"
            y="13"
            width="0"
            height="13"
            rx="6.5"
            fill="#10B981"
          />

          <text
            x="16"
            y="50"
            data-metric-three-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="118"
            y="41"
            width="165"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-metric-three-bar
            x="118"
            y="41"
            width="0"
            height="13"
            rx="6.5"
            fill="#F59E0B"
          />

          <text
            x="342"
            y="50"
            data-panel-note
            font-size="10.5"
            font-weight="900"
            fill="#0E7490"
          ></text>
        </g>

        <text
          x="22"
          y="422"
          data-footer-note
          font-size="10.5"
          font-weight="900"
          fill="#164E63"
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

    var voltageInput=root.querySelector(
      '[data-voltage]'
    );
    var timeInput=root.querySelector(
      '[data-time]'
    );
    var gelInput=root.querySelector(
      '[data-gel]'
    );
    var amountInput=root.querySelector(
      '[data-amount]'
    );

    var voltageValue=root.querySelector(
      '[data-voltage-value]'
    );
    var timeValue=root.querySelector(
      '[data-time-value]'
    );
    var gelValue=root.querySelector(
      '[data-gel-value]'
    );
    var amountValue=root.querySelector(
      '[data-amount-value]'
    );

    var stageButtons=root.querySelectorAll(
      '[data-stage]'
    );
    var laneButtons=root.querySelectorAll(
      '[data-lane]'
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
    var statusDistance=root.querySelector(
      '[data-status-distance]'
    );
    var statusResult=root.querySelector(
      '[data-status-result]'
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
    var gelLayer=root.querySelector(
      '[data-gel-layer]'
    );
    var electrodeLayer=root.querySelector(
      '[data-electrode-layer]'
    );
    var labelLayer=root.querySelector(
      '[data-label-layer]'
    );
    var interpretationLayer=root.querySelector(
      '[data-interpretation-layer]'
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
      'loading',
      'migration',
      'separation',
      'interpretation'
    ];

    var stage='loading';
    var lane='sampleA';
    var currentScenario='standard';
    var automatic=true;
    var showLabels=${showLabels ? 'true' : 'false'};
    var timer=null;

    var stageInfo={
      loading:{
        title:'阶段1：样品加入上样孔',
        summary:'分子量标准、样品和阴性对照分别加入独立泳道',
        note:'上样孔位于负极一侧，样品应分别加入独立泳道，避免混样和孔间溢出。'
      },
      migration:{
        title:'阶段2：DNA在电场中迁移',
        summary:'DNA整体带负电，通常由负极一侧向正极方向移动',
        note:'电压升高可加快迁移，但过高可能增加发热、条带弯曲和分辨率下降。'
      },
      separation:{
        title:'阶段3：不同大小片段逐渐分离',
        summary:'凝胶产生分子筛效应，较小片段通常迁移得更远',
        note:'片段大小不是唯一影响因素，DNA构象、凝胶类型和缓冲体系也会影响迁移。'
      },
      interpretation:{
        title:'阶段4：根据标准条带进行相对判读',
        summary:'以分子量标准为参照，比较样品条带的位置、数量、亮度和形态',
        note:'条带大小和亮度只能作相对判读，不能直接证明样品身份或形成诊断结论。'
      }
    };

    var scenarios={
      standard:{
        voltage:90,
        time:28,
        gel:1.2,
        amount:65
      },
      weak:{
        voltage:90,
        time:28,
        gel:1.2,
        amount:25
      },
      overload:{
        voltage:105,
        time:32,
        gel:1.1,
        amount:98
      },
      contamination:{
        voltage:90,
        time:28,
        gel:1.2,
        amount:68
      }
    };

    var laneOrder=[
      'ladder',
      'sampleA',
      'sampleB',
      'negative'
    ];

    var laneNames={
      ladder:'M 分子量标准',
      sampleA:'样品A',
      sampleB:'样品B',
      negative:'阴性对照'
    };

    var laneShortNames={
      ladder:'M',
      sampleA:'A',
      sampleB:'B',
      negative:'N'
    };

    var baseBands={
      ladder:[
        1500,
        1000,
        700,
        500,
        300,
        200
      ],
      sampleA:[
        1000,
        500,
        300
      ],
      sampleB:[
        700,
        300
      ],
      negative:[]
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

    function laneBands(
      laneName
    ){
      var bands=
        baseBands[laneName].slice();

      if(
        currentScenario==='contamination'
        &&laneName==='negative'
      ){
        bands.push(450);
      }

      if(
        currentScenario==='contamination'
        &&laneName==='sampleB'
      ){
        bands.push(520);
      }

      return bands;
    }

    function stageProgress(){
      if(stage==='loading'){
        return 0;
      }

      if(stage==='migration'){
        return .48;
      }

      return 1;
    }

    function voltageFactor(voltage){
      return clamp(
        voltage/90,
        .4,
        1.65
      );
    }

    function timeFactor(time){
      return clamp(
        time/28,
        .18,
        1.8
      );
    }

    function gelMobilityFactor(
      concentration
    ){
      return clamp(
        1.2/concentration,
        .48,
        1.45
      );
    }

    function baseMigrationFraction(
      basePairs
    ){
      var large=
        Math.log(1800)/Math.LN10;

      var small=
        Math.log(150)/Math.LN10;

      var current=
        Math.log(
          clamp(
            basePairs,
            150,
            1800
          )
        )/Math.LN10;

      return clamp(
        (large-current)
        /(large-small),
        0,
        1
      );
    }

    function migrationDistance(
      basePairs,
      voltage,
      time,
      concentration
    ){
      var raw=
        18
        +baseMigrationFraction(
          basePairs
        )
        *142
        *voltageFactor(voltage)
        *timeFactor(time)
        *gelMobilityFactor(
          concentration
        );

      return clamp(
        raw,
        8,
        184
      );
    }

    function resolutionScore(
      voltage,
      time,
      concentration,
      amount
    ){
      var voltagePenalty=
        Math.exp(
          -Math.pow(
            (voltage-90)/58,
            2
          )
        );

      var timePenalty=
        Math.exp(
          -Math.pow(
            (time-30)/30,
            2
          )
        );

      var gelPenalty=
        Math.exp(
          -Math.pow(
            (concentration-1.35)/1.35,
            2
          )
        );

      var overloadPenalty=
        amount>82
          ?clamp(
            1-(amount-82)/35,
            .35,
            1
          )
          :1;

      return clamp(
        100
        *voltagePenalty
        *timePenalty
        *gelPenalty
        *overloadPenalty,
        0,
        100
      );
    }

    function bandOpacity(
      laneName,
      amount
    ){
      if(laneName==='ladder'){
        return .88;
      }

      if(laneName==='negative'){
        return currentScenario==='contamination'
          ?.5
          :0;
      }

      return clamp(
        .18+amount/100,
        .28,
        1
      );
    }

    function bandWidth(
      laneName,
      amount
    ){
      if(laneName==='ladder'){
        return 42;
      }

      if(
        currentScenario==='overload'
        ||amount>86
      ){
        return 58;
      }

      return 44;
    }

    function bandHeight(
      laneName,
      amount
    ){
      if(laneName==='ladder'){
        return 5;
      }

      if(
        currentScenario==='overload'
        ||amount>86
      ){
        return 10;
      }

      if(amount<32){
        return 4;
      }

      return 6;
    }

    function laneX(index){
      return 102+index*92;
    }

    function drawWell(
      x,
      labelText,
      active
    ){
      return ''
        +'<rect x="'+(x-25)
        +'" y="120" width="50" height="16"'
        +' rx="5" fill="'
        +(active?'#164E63':'#0E7490')
        +'" opacity="'
        +(active?'1':'.76')
        +'"/>'
        +'<text x="'+x+'" y="114"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="'
        +(active?'#B91C1C':'#0E7490')
        +'">'
        +labelText
        +'</text>';
    }

    function drawBand(
      x,
      y,
      width,
      height,
      opacity,
      color,
      extraClass
    ){
      return '<rect'
        +(extraClass
          ?' class="'+extraClass+'"'
          :'')
        +' x="'+(x-width/2)
        +'" y="'+(y-height/2)
        +'" width="'+width
        +'" height="'+height
        +'" rx="'+Math.max(2,height/2)
        +'" fill="'+color
        +'" opacity="'+opacity
        +'" filter="url(#${rootId}-glow)"/>';
    }

    function drawSmear(
      x,
      yStart,
      yEnd,
      opacity
    ){
      return ''
        +'<path d="M'+(x-22)+' '+yStart
        +' C'+(x-30)+' '+(yStart+34)+' '
        +(x-14)+' '+(yEnd-28)+' '
        +(x-18)+' '+yEnd
        +' L'+(x+18)+' '+yEnd
        +' C'+(x+14)+' '+(yEnd-28)+' '
        +(x+30)+' '+(yStart+34)+' '
        +(x+22)+' '+yStart
        +' Z" fill="#F59E0B"'
        +' opacity="'+opacity+'"/>';
    }

    function drawGel(
      voltage,
      time,
      concentration,
      amount
    ){
      var html='';
      var progress=
        stageProgress();

      html+='<rect x="58" y="119"'
        +' width="368" height="181"'
        +' rx="12" fill="#E0F2FE"'
        +' stroke="#22D3EE"'
        +' stroke-width="2.5"'
        +' opacity=".88"/>';

      for(var laneIndex=0;
        laneIndex<laneOrder.length;
        laneIndex++
      ){
        var laneName=
          laneOrder[laneIndex];

        var x=
          laneX(laneIndex);

        html+=drawWell(
          x,
          laneShortNames[laneName],
          laneName===lane
        );

        html+='<line x1="'+x
          +'" y1="138" x2="'+x
          +'" y2="292"'
          +' stroke="#BAE6FD"'
          +' stroke-width="1.5"'
          +' stroke-dasharray="4 5"/>';

        var bands=
          laneBands(laneName);

        if(stage==='loading'){
          if(
            bands.length>0
            ||laneName==='negative'
          ){
            html+=drawBand(
              x,
              138,
              laneName==='ladder'?42:44,
              7,
              laneName==='negative'
                ?.18
                :bandOpacity(
                  laneName,
                  amount
                ),
              laneName==='ladder'
                ?'#8B5CF6'
                :'#0891B2',
              'ge-pulse'
            );
          }

          continue;
        }

        for(var bandIndex=0;
          bandIndex<bands.length;
          bandIndex++
        ){
          var basePairs=
            bands[bandIndex];

          var distance=
            migrationDistance(
              basePairs,
              voltage,
              time,
              concentration
            )
            *progress;

          var y=
            139+distance;

          var width=
            bandWidth(
              laneName,
              amount
            );

          var height=
            bandHeight(
              laneName,
              amount
            );

          var opacity=
            bandOpacity(
              laneName,
              amount
            );

          if(
            currentScenario==='weak'
            &&laneName!=='ladder'
          ){
            opacity*=.42;
          }

          if(
            currentScenario==='contamination'
            &&laneName==='negative'
          ){
            opacity=.55;
          }

          var color=
            laneName==='ladder'
              ?'#8B5CF6'
              :laneName==='negative'
                ?'#EF4444'
                :'#0891B2';

          html+=drawBand(
            x,
            y,
            width,
            height,
            opacity,
            color,
            'ge-band'
          );

          if(
            showLabels
            &&laneName==='ladder'
            &&stage==='interpretation'
          ){
            html+='<text x="'+(x-31)
              +'" y="'+(y+3)
              +'" text-anchor="end"'
              +' font-size="8.5"'
              +' font-weight="800"'
              +' fill="#6D28D9">'
              +basePairs+' bp'
              +'</text>';
          }
        }

        if(
          currentScenario==='overload'
          &&laneName!=='ladder'
          &&laneName!=='negative'
          &&stage!=='loading'
        ){
          html+=drawSmear(
            x,
            151,
            286,
            .16+.2*progress
          );
        }
      }

      gelLayer.innerHTML=html;
    }

    function drawElectrodes(
      voltage
    ){
      var html='';

      html+='<rect x="46" y="83"'
        +' width="392" height="21"'
        +' rx="10" fill="#DBEAFE"'
        +' stroke="#2563EB"'
        +' stroke-width="2"/>';

      html+='<text x="242" y="98"'
        +' text-anchor="middle"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'负极一侧（−）｜上样孔'
        +'</text>';

      html+='<rect x="46" y="305"'
        +' width="392" height="21"'
        +' rx="10" fill="#FEE2E2"'
        +' stroke="#EF4444"'
        +' stroke-width="2"/>';

      html+='<text x="242" y="320"'
        +' text-anchor="middle"'
        +' font-size="11"'
        +' font-weight="900"'
        +' fill="#B91C1C">'
        +'正极方向（+）'
        +'</text>';

      html+='<path class="ge-flow"'
        +' d="M446 128 V279"'
        +' fill="none"'
        +' stroke="#0891B2"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="451" y="207"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#0E7490">'
        +'DNA迁移'
        +'</text>';

      html+='<text x="451" y="222"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#0E7490">'
        +voltage.toFixed(0)+' V'
        +'</text>';

      electrodeLayer.innerHTML=html;
    }

    function drawLabels(){
      if(!showLabels){
        labelLayer.innerHTML='';
        return;
      }

      var html='';

      html+='<text x="61" y="113"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#0E7490">'
        +'上样孔'
        +'</text>';

      html+='<text x="62" y="292"'
        +' font-size="9.5"'
        +' font-weight="900"'
        +' fill="#0E7490">'
        +'迁移越远通常表示片段越小'
        +'</text>';

      if(stage==='interpretation'){
        var activeIndex=
          laneOrder.indexOf(lane);

        var activeX=
          laneX(
            activeIndex<0?1:activeIndex
          );

        html+='<rect x="'+(activeX-34)
          +'" y="108" width="68" height="202"'
          +' rx="11" fill="none"'
          +' stroke="#EF4444"'
          +' stroke-width="3"'
          +' stroke-dasharray="7 5"/>';

        html+='<text x="'+activeX
          +'" y="300"'
          +' text-anchor="middle"'
          +' font-size="9.5"'
          +' font-weight="900"'
          +' fill="#B91C1C">'
          +'当前判读泳道'
          +'</text>';
      }

      labelLayer.innerHTML=html;
    }

    function estimatedBandText(
      laneName
    ){
      var bands=
        laneBands(laneName);

      if(bands.length===0){
        return '未见模拟条带';
      }

      return bands
        .slice()
        .sort(function(a,b){
          return b-a;
        })
        .map(function(value){
          return value+' bp';
        })
        .join('、');
    }

    function laneJudgement(
      laneName,
      amount
    ){
      if(laneName==='ladder'){
        return {
          short:'标准参照',
          color:'#6D28D9',
          text:'分子量标准提供一组已知大小的参考条带，用于对样品条带进行相对估算。'
        };
      }

      if(laneName==='negative'){
        if(currentScenario==='contamination'){
          return {
            short:'异常条带',
            color:'#DC2626',
            text:'阴性对照出现模拟条带，真实实验中应优先排查污染、非特异扩增或样品混入。'
          };
        }

        return {
          short:'应无条带',
          color:'#047857',
          text:'阴性对照未出现模拟条带，符合不含目标DNA的教学预期。'
        };
      }

      if(
        currentScenario==='overload'
        ||amount>86
      ){
        return {
          short:'过量拖尾',
          color:'#B45309',
          text:'上样量过大，条带变宽并出现拖尾或涂抹状信号，可能降低相邻条带的分辨能力。'
        };
      }

      if(
        currentScenario==='weak'
        ||amount<32
      ){
        return {
          short:'弱条带',
          color:'#B45309',
          text:'条带亮度较弱，可能与上样量少、产物量低、染色不足或成像条件有关。'
        };
      }

      return {
        short:'条带清晰',
        color:'#047857',
        text:'当前样品泳道形成相对清晰的模拟条带，可与分子量标准比较其相对大小。'
      };
    }

    function drawInterpretation(
      amount
    ){
      var judgement=
        laneJudgement(
          lane,
          amount
        );

      var bands=
        laneBands(lane);

      var html='';

      html+='<text x="502" y="132"'
        +' font-size="12"'
        +' font-weight="900"'
        +' fill="#0F766E">'
        +laneNames[lane]
        +'</text>';

      html+='<rect x="502" y="145"'
        +' width="216" height="42"'
        +' rx="11" fill="#FFFFFF"'
        +' stroke="#99F6E4"'
        +' stroke-width="2"/>';

      html+='<text x="610" y="162"'
        +' text-anchor="middle"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'模拟条带大小'
        +'</text>';

      html+='<text x="610" y="179"'
        +' text-anchor="middle"'
        +' font-size="10.5"'
        +' font-weight="900"'
        +' fill="#0E7490">'
        +estimatedBandText(lane)
        +'</text>';

      html+='<rect x="502" y="198"'
        +' width="216" height="46"'
        +' rx="11" fill="#FFFFFF"'
        +' stroke="#99F6E4"'
        +' stroke-width="2"/>';

      html+='<text x="610" y="217"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'相对判读'
        +'</text>';

      html+='<text x="610" y="237"'
        +' text-anchor="middle"'
        +' font-size="13"'
        +' font-weight="900"'
        +' fill="'+judgement.color+'">'
        +judgement.short
        +'</text>';

      var maxBand=
        bands.length>0
          ?Math.max.apply(
            Math,
            bands
          )
          :0;

      var minBand=
        bands.length>0
          ?Math.min.apply(
            Math,
            bands
          )
          :0;

      html+='<text x="502" y="266"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'条带数：'+bands.length
        +'</text>';

      html+='<text x="502" y="283"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'最大片段：'
        +(maxBand>0?maxBand+' bp':'—')
        +'</text>';

      html+='<text x="502" y="300"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'最小片段：'
        +(minBand>0?minBand+' bp':'—')
        +'</text>';

      interpretationLayer.innerHTML=html;

      return judgement;
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

      voltageInput.value=
        String(data.voltage);

      timeInput.value=
        String(data.time);

      gelInput.value=
        String(data.gel);

      amountInput.value=
        String(data.amount);

      currentScenario=name;

      if(pauseAutomatic){
        automatic=false;
      }

      update();
    }

    function update(){
      var voltage=
        clamp(
          Number(voltageInput.value),
          40,
          160
        );

      var runTime=
        clamp(
          Number(timeInput.value),
          5,
          60
        );

      var concentration=
        clamp(
          Number(gelInput.value),
          .8,
          3
        );

      var amount=
        clamp(
          Number(amountInput.value),
          20,
          100
        );

      var info=
        stageInfo[stage];

      var judgement=
        laneJudgement(
          lane,
          amount
        );

      var activeBands=
        laneBands(lane);

      var smallestBand=
        activeBands.length>0
          ?Math.min.apply(
            Math,
            activeBands
          )
          :0;

      var farthestDistance=
        smallestBand>0
          ?migrationDistance(
            smallestBand,
            voltage,
            runTime,
            concentration
          )
            *stageProgress()
          :0;

      var resolution=
        resolutionScore(
          voltage,
          runTime,
          concentration,
          amount
        );

      voltageValue.textContent=
        voltage.toFixed(0)+' V';

      timeValue.textContent=
        runTime.toFixed(0)+' min';

      gelValue.textContent=
        concentration.toFixed(1)+'%';

      amountValue.textContent=
        amount.toFixed(0)+'%';

      setActive(
        stageButtons,
        'data-stage',
        stage
      );

      setActive(
        laneButtons,
        'data-lane',
        lane
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
          2.8-voltage/85,
          .55,
          2.5
        ).toFixed(2)+'s'
      );

      title.textContent=
        info.title;

      summary.textContent=
        info.summary;

      statusStage.textContent=
        stage==='loading'
          ?'上样'
          :stage==='migration'
            ?'迁移'
            :stage==='separation'
              ?'分离'
              :'判读';

      statusDistance.textContent=
        farthestDistance>0
          ?farthestDistance.toFixed(0)+' px'
          :'0 px';

      statusResult.textContent=
        judgement.short;

      drawGel(
        voltage,
        runTime,
        concentration,
        amount
      );

      drawElectrodes(
        voltage
      );

      drawLabels();

      judgement=
        drawInterpretation(
          amount
        );

      metricOneLabel.textContent=
        '迁移推动程度';

      metricTwoLabel.textContent=
        '分离分辨率';

      metricThreeLabel.textContent=
        '相对上样量';

      metricOneBar.setAttribute(
        'width',
        String(
          165
          *clamp(
            voltageFactor(voltage)
            *timeFactor(runTime)
            /1.8,
            0,
            1
          )
        )
      );

      metricTwoBar.setAttribute(
        'width',
        String(
          238
          *resolution/100
        )
      );

      metricThreeBar.setAttribute(
        'width',
        String(
          165
          *amount/100
        )
      );

      var conditionNote='当前电压、时间和凝胶浓度相对协调。';

      if(voltage>130){
        conditionNote=
          '电压偏高，迁移可能过快并降低条带分辨率。';
      }else if(runTime<15){
        conditionNote=
          '运行时间较短，不同片段可能尚未充分分离。';
      }else if(runTime>50){
        conditionNote=
          '运行时间较长，小片段可能接近或超过凝胶下缘。';
      }else if(concentration>2.3){
        conditionNote=
          '凝胶浓度较高，更适合较小片段分辨，但较大片段迁移较慢。';
      }else if(concentration<1){
        conditionNote=
          '凝胶浓度较低，较大片段迁移相对容易，但小片段分辨可能不足。';
      }

      panelNote.textContent=
        conditionNote
        +' 分辨率示意 '
        +resolution.toFixed(0)
        +'%';

      footerNote.textContent=
        '条带位置只用于相对大小估算；条带亮度只可在条件相近时粗略比较DNA相对含量。';

      result.innerHTML=
        info.note
        +'<br>'+judgement.text
        +' '+conditionNote
        +' 本模型不用于亲子鉴定、疾病诊断、病原体检测或个人身份判断。';
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

      var voltage=
        Number(
          voltageInput.value
        );

      var interval=
        clamp(
          3500-voltage*14,
          1100,
          3000
        );

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
        interval
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
      j<laneButtons.length;
      j++
    ){
      laneButtons[j].onclick=function(){
        lane=this.getAttribute(
          'data-lane'
        );

        update();
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

    voltageInput.oninput=function(){
      currentScenario='';
      update();
      schedule();
    };

    timeInput.oninput=function(){
      currentScenario='';
      update();
    };

    gelInput.oninput=function(){
      currentScenario='';
      update();
    };

    amountInput.oninput=function(){
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
