/**
 * lifeScienceLabTemplatesBiotechnologyPcrAmplification.ts
 *
 * 平面生命科学实验室：PCR扩增与循环过程。
 *
 * 教学目标：
 * 1. 理解PCR一个循环由变性、退火和延伸三个主要阶段组成；
 * 2. 理解两条引物分别与两条模板链互补结合，且引物3′端朝向目标区域；
 * 3. 理解DNA聚合酶从引物3′端开始，沿5′→3′方向延伸新链；
 * 4. 观察循环次数、退火温度、扩增效率和初始模板量对理论产物量的影响；
 * 5. 比较样品、阳性对照、阴性对照和空白对照的预期结果；
 * 6. 区分理想倍增模型与真实扩增过程中的效率损失、非特异扩增和平台期。
 *
 * 教学边界：
 * 1. 本模型只用于PCR原理教学，不提供真实实验操作参数；
 * 2. 理想状态下每循环可近似倍增，但实际扩增效率通常低于100%；
 * 3. 理论产物量按N=N0×(1+E)^n进行教学估算，不代表真实定量结果；
 * 4. 退火温度过低可能增加非特异结合，过高可能降低引物结合效率；
 * 5. 阳性对照预期出现扩增，阴性和空白对照预期不出现扩增；
 * 6. 阴性或空白对照出现扩增通常提示污染或非特异扩增，需要进一步排查；
 * 7. 样品出现或未出现扩增都不能单独作为病原体、遗传病或临床诊断结论；
 * 8. 本模型不用于亲子鉴定、疾病筛查、病原体检测或个人健康判断。
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
function pcrStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #93C5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DBEAFE,#ECFDF5);border-bottom:1px solid #93C5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#1D4ED8}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:256px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FBFF;border-right:1px solid #BFDBFE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#2563EB;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#2563EB}'
    + '#' + rootId + ' .pcr-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#1D4ED8}'
    + '#' + rootId + ' .pcr-stages{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .pcr-controls{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .pcr-scenarios{display:grid;grid-template-columns:repeat(4,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .pcr-button{min-height:31px;padding:3px;border:1px solid #93C5FD;border-radius:8px;background:#fff;color:#1D4ED8;font-size:8.9px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .pcr-button.active{border-color:#2563EB;background:#DBEAFE;box-shadow:0 3px 9px rgba(37,99,235,.14)}'
    + '#' + rootId + ' .pcr-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#60A5FA,#2563EB);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pcr-auto.paused{background:#64748B}'
    + '#' + rootId + ' .pcr-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pcr-toggle.off{background:#64748B}'
    + '#' + rootId + ' .pcr-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .pcr-card{padding:6px 3px;border:1px solid #BFDBFE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .pcr-card b{display:block;min-height:18px;font-size:12.5px;color:#1D4ED8}'
    + '#' + rootId + ' .pcr-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#DBEAFE;color:#1E3A8A;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .pcr-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--pcr-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .pcr-pulse{animation:' + rootId + '-pulse 1.05s ease-in-out infinite alternate}'
    + '#' + rootId + ' .pcr-rise{animation:' + rootId + '-rise 1.45s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.38}to{opacity:1}}'
    + '@keyframes ' + rootId + '-rise{from{transform:translateY(4px);opacity:.55}to{transform:translateY(-4px);opacity:1}}'
    + '</style>'
}

/** 避免在外层模板字符串中直接写出脚本结束标签。 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_BIOTECHNOLOGY_PCR_AMPLIFICATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-pcr-amplification-cycle',
    group: '🧬 现代生物技术',
    name: 'PCR扩增与循环过程',
    emoji: '🧬',
    desc: '观察变性、退火、延伸、引物方向、循环倍增、效率损失及阳性阴性空白对照',
    params: [
      {
        key: 'cycleCount',
        label: '循环次数',
        type: 'number',
        min: 1,
        max: 35,
        step: 1,
        defaultValue: 24,
      },
      {
        key: 'annealingTemperature',
        label: '退火温度/℃',
        type: 'number',
        min: 40,
        max: 72,
        step: 1,
        defaultValue: 58,
        hint: '本模型以58℃附近作为教学示意适宜值',
      },
      {
        key: 'extensionEfficiency',
        label: '单循环扩增效率',
        type: 'number',
        min: 40,
        max: 100,
        step: 1,
        defaultValue: 88,
      },
      {
        key: 'initialTemplate',
        label: '初始模板相对量',
        type: 'number',
        min: 1,
        max: 100,
        step: 1,
        defaultValue: 12,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const cycleCount = num(params, 'cycleCount', 24)
      const annealingTemperature = num(
        params,
        'annealingTemperature',
        58,
      )
      const extensionEfficiency = num(
        params,
        'extensionEfficiency',
        88,
      )
      const initialTemplate = num(
        params,
        'initialTemplate',
        12,
      )
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${pcrStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧬 PCR扩增与循环过程</div>
    <div class="bl-note">理论倍增不等于实际必然倍增；对照异常需要排查污染与非特异扩增</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>循环次数</span>
          <span class="bl-value" data-cycle-value></span>
        </div>
        <input
          data-cycle
          type="range"
          min="1"
          max="35"
          step="1"
          value="${n(cycleCount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>退火温度</span>
          <span class="bl-value" data-anneal-value></span>
        </div>
        <input
          data-anneal
          type="range"
          min="40"
          max="72"
          step="1"
          value="${n(annealingTemperature)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>单循环扩增效率</span>
          <span class="bl-value" data-efficiency-value></span>
        </div>
        <input
          data-efficiency
          type="range"
          min="40"
          max="100"
          step="1"
          value="${n(extensionEfficiency)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>初始模板相对量</span>
          <span class="bl-value" data-template-value></span>
        </div>
        <input
          data-template
          type="range"
          min="1"
          max="100"
          step="1"
          value="${n(initialTemplate)}"
        >
      </div>

      <div class="pcr-subtitle">循环阶段</div>

      <div class="pcr-stages">
        <button
          type="button"
          class="pcr-button active"
          data-stage="denaturation"
        >1. 变性</button>

        <button
          type="button"
          class="pcr-button"
          data-stage="annealing"
        >2. 退火</button>

        <button
          type="button"
          class="pcr-button"
          data-stage="extension"
        >3. 延伸</button>

        <button
          type="button"
          class="pcr-button"
          data-stage="result"
        >4. 循环结果</button>
      </div>

      <div class="pcr-subtitle">对照类型</div>

      <div class="pcr-controls">
        <button
          type="button"
          class="pcr-button active"
          data-control="sample"
        >待测样品</button>

        <button
          type="button"
          class="pcr-button"
          data-control="positive"
        >阳性对照</button>

        <button
          type="button"
          class="pcr-button"
          data-control="negative"
        >阴性对照</button>

        <button
          type="button"
          class="pcr-button"
          data-control="blank"
        >空白对照</button>
      </div>

      <div class="pcr-subtitle">快速情境</div>

      <div class="pcr-scenarios">
        <button
          type="button"
          class="pcr-button active"
          data-scenario="standard"
        >标准条件</button>

        <button
          type="button"
          class="pcr-button"
          data-scenario="lowAnneal"
        >退火偏低</button>

        <button
          type="button"
          class="pcr-button"
          data-scenario="highAnneal"
        >退火偏高</button>

        <button
          type="button"
          class="pcr-button"
          data-scenario="lowEfficiency"
        >效率不足</button>
      </div>

      <button
        type="button"
        class="pcr-auto"
        data-auto
      >自动演示：运行中</button>

      <button
        type="button"
        class="pcr-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <div class="pcr-status">
        <div class="pcr-card">
          <b data-status-cycle></b>
          <span>循环设置</span>
        </div>

        <div class="pcr-card">
          <b data-status-copies></b>
          <span>理论相对产物</span>
        </div>

        <div class="pcr-card">
          <b data-status-control></b>
          <span>对照预期</span>
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
        aria-label="PCR扩增循环互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-left"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#EFF6FF"/>
            <stop offset="100%" stop-color="#ECFDF5"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-right"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#F8FAFC"/>
            <stop offset="100%" stop-color="#EEF2FF"/>
          </linearGradient>

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

          <marker
            id="${rootId}-arrow-pink"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path
              d="M0,0 L0,6 L8,3 z"
              fill="#EC4899"
            />
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#1E3A8A"
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
          fill="#1D4ED8"
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
            width="390"
            height="223"
            rx="20"
            fill="url(#${rootId}-left)"
            stroke="#93C5FD"
            stroke-width="3"
          />

          <rect
            x="432"
            y="79"
            width="306"
            height="223"
            rx="20"
            fill="url(#${rootId}-right)"
            stroke="#C7D2FE"
            stroke-width="3"
          />
        </g>

        <text
          x="217"
          y="104"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#1D4ED8"
        >分子过程与引物方向</text>

        <text
          x="585"
          y="104"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#4338CA"
        >循环数与相对产物量</text>

        <g data-molecule-layer></g>
        <g data-label-layer></g>
        <g data-curve-layer></g>

        <g transform="translate(22 322)">
          <rect
            width="716"
            height="78"
            rx="17"
            fill="#F8FAFC"
            stroke="#CBD5E1"
            stroke-width="2"
          />

          <text
            x="16"
            y="23"
            data-temp-one-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="126"
            y="14"
            width="170"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-temp-one-bar
            x="126"
            y="14"
            width="0"
            height="13"
            rx="6.5"
            fill="#EF4444"
          />

          <text
            x="350"
            y="23"
            data-temp-two-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="462"
            y="14"
            width="226"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-temp-two-bar
            x="462"
            y="14"
            width="0"
            height="13"
            rx="6.5"
            fill="#2563EB"
          />

          <text
            x="16"
            y="55"
            data-temp-three-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect
            x="126"
            y="46"
            width="170"
            height="13"
            rx="6.5"
            fill="#E2E8F0"
          />

          <rect
            data-temp-three-bar
            x="126"
            y="46"
            width="0"
            height="13"
            rx="6.5"
            fill="#10B981"
          />

          <text
            x="350"
            y="56"
            data-panel-note
            font-size="10.5"
            font-weight="900"
            fill="#1D4ED8"
          ></text>
        </g>

        <text
          x="22"
          y="421"
          data-footer-note
          font-size="10.5"
          font-weight="900"
          fill="#1E3A8A"
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

    var cycleInput=root.querySelector('[data-cycle]');
    var annealInput=root.querySelector('[data-anneal]');
    var efficiencyInput=root.querySelector('[data-efficiency]');
    var templateInput=root.querySelector('[data-template]');

    var cycleValue=root.querySelector('[data-cycle-value]');
    var annealValue=root.querySelector('[data-anneal-value]');
    var efficiencyValue=root.querySelector('[data-efficiency-value]');
    var templateValue=root.querySelector('[data-template-value]');

    var stageButtons=root.querySelectorAll('[data-stage]');
    var controlButtons=root.querySelectorAll('[data-control]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var autoButton=root.querySelector('[data-auto]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var statusCycle=root.querySelector('[data-status-cycle]');
    var statusCopies=root.querySelector('[data-status-copies]');
    var statusControl=root.querySelector('[data-status-control]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var moleculeLayer=root.querySelector('[data-molecule-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');
    var curveLayer=root.querySelector('[data-curve-layer]');

    var tempOneLabel=root.querySelector('[data-temp-one-label]');
    var tempTwoLabel=root.querySelector('[data-temp-two-label]');
    var tempThreeLabel=root.querySelector('[data-temp-three-label]');
    var tempOneBar=root.querySelector('[data-temp-one-bar]');
    var tempTwoBar=root.querySelector('[data-temp-two-bar]');
    var tempThreeBar=root.querySelector('[data-temp-three-bar]');
    var panelNote=root.querySelector('[data-panel-note]');
    var footerNote=root.querySelector('[data-footer-note]');

    var stages=[
      'denaturation',
      'annealing',
      'extension',
      'result'
    ];

    var stage='denaturation';
    var control='sample';
    var automatic=true;
    var showLabels=${showLabels ? 'true' : 'false'};
    var timer=null;
    var currentScenario='standard';

    var scenarios={
      standard:{
        anneal:58,
        efficiency:88,
        cycle:24
      },
      lowAnneal:{
        anneal:46,
        efficiency:82,
        cycle:24
      },
      highAnneal:{
        anneal:68,
        efficiency:72,
        cycle:24
      },
      lowEfficiency:{
        anneal:58,
        efficiency:48,
        cycle:24
      }
    };

    var stageInfo={
      denaturation:{
        title:'阶段1：高温变性',
        summary:'双链DNA在高温条件下解开，形成两条单链模板',
        note:'变性阶段通常采用较高温度，使双链DNA分开；本图固定为约95℃教学示意。'
      },
      annealing:{
        title:'阶段2：引物退火',
        summary:'两条引物分别与两条模板链互补结合，3′端朝向目标区域',
        note:'退火温度影响引物结合效率与特异性，过低可能增加非特异结合，过高可能使结合不足。'
      },
      extension:{
        title:'阶段3：新链延伸',
        summary:'DNA聚合酶从引物3′端开始，沿5′→3′方向延伸新链',
        note:'聚合酶不能从头开始合成DNA，需要已有引物提供可延伸的3′端。'
      },
      result:{
        title:'阶段4：循环累积',
        summary:'新形成的DNA分子进入下一循环，目标片段数量逐步增加',
        note:'理想模型接近每循环倍增，真实实验会因效率下降、试剂限制和平台期偏离理想曲线。'
      }
    };

    function clamp(value,min,max){
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

    function annealYieldFactor(temperature){
      var distance=
        (temperature-58)/9;

      return Math.exp(
        -distance*distance
      );
    }

    function specificityState(temperature){
      if(temperature<50){
        return {
          label:'特异性偏低',
          text:'退火温度偏低，引物更容易与非完全互补序列结合，非特异扩增风险增加。'
        };
      }

      if(temperature>65){
        return {
          label:'结合偏弱',
          text:'退火温度偏高，引物与模板结合不足，目标扩增效率可能下降。'
        };
      }

      return {
        label:'相对适宜',
        text:'当前退火温度处于本教学模型的相对适宜范围。'
      };
    }

    function controlProfile(
      name,
      initialTemplate
    ){
      if(name==='positive'){
        return {
          template:Math.max(
            20,
            initialTemplate
          ),
          label:'应有扩增',
          title:'阳性对照',
          explanation:'阳性对照含已知目标模板，预期出现扩增，用于确认体系和试剂能够工作。'
        };
      }

      if(name==='negative'){
        return {
          template:0,
          label:'应无扩增',
          title:'阴性对照',
          explanation:'阴性对照不含目标模板，预期不出现目标扩增；若出现扩增，需要排查污染或非特异反应。'
        };
      }

      if(name==='blank'){
        return {
          template:0,
          label:'应无扩增',
          title:'空白对照',
          explanation:'空白对照用水或空白基质替代样品，用于监测试剂、耗材和操作过程中的污染。'
        };
      }

      return {
        template:initialTemplate,
        label:'待判读',
        title:'待测样品',
        explanation:'待测样品是否出现扩增只说明教学模型中的目标序列是否被放大，不能单独形成临床诊断结论。'
      };
    }

    function amplificationModel(
      cycles,
      efficiency,
      temperature,
      initialTemplate,
      controlName
    ){
      var profile=controlProfile(
        controlName,
        initialTemplate
      );

      var annealFactor=
        annealYieldFactor(
          temperature
        );

      var effectiveEfficiency=
        clamp(
          efficiency/100
          *annealFactor,
          0,
          1
        );

      var theoretical=
        profile.template>0
          ?profile.template
            *Math.pow(
              1+effectiveEfficiency,
              cycles
            )
          :0;

      var ideal=
        profile.template>0
          ?profile.template
            *Math.pow(
              2,
              cycles
            )
          :0;

      return {
        profile:profile,
        annealFactor:annealFactor,
        effectiveEfficiency:effectiveEfficiency,
        theoretical:theoretical,
        ideal:ideal
      };
    }

    function formatCopies(value){
      if(value<=0){
        return '0';
      }

      if(value<1000){
        return value.toFixed(0);
      }

      var exponent=Math.floor(
        Math.log(value)/Math.LN10
      );

      var coefficient=
        value/Math.pow(
          10,
          exponent
        );

      return coefficient.toFixed(1)
        +'×10^'
        +exponent;
    }

    function line(
      x1,
      y1,
      x2,
      y2,
      color,
      width,
      extra
    ){
      return '<line x1="'+x1
        +'" y1="'+y1
        +'" x2="'+x2
        +'" y2="'+y2
        +'" stroke="'+color
        +'" stroke-width="'+width
        +'" stroke-linecap="round"'
        +(extra||'')
        +'/>';
    }

    function strandLabels(){
      if(!showLabels){
        return '';
      }

      return ''
        +'<text x="56" y="141" font-size="10" font-weight="900" fill="#1D4ED8">5′</text>'
        +'<text x="366" y="141" font-size="10" font-weight="900" fill="#1D4ED8">3′</text>'
        +'<text x="56" y="247" font-size="10" font-weight="900" fill="#BE185D">3′</text>'
        +'<text x="366" y="247" font-size="10" font-weight="900" fill="#BE185D">5′</text>';
    }

    function drawDenaturation(){
      var html='';

      html+=line(
        72,
        154,
        360,
        154,
        '#2563EB',
        7,
        ' class="pcr-rise"'
      );

      html+=line(
        72,
        232,
        360,
        232,
        '#EC4899',
        7,
        ' class="pcr-rise"'
      );

      for(var i=0;i<9;i++){
        var x=90+i*31;

        html+='<path class="pcr-pulse" d="M'
          +x+' 170 C'
          +(x+8)+' 182 '
          +(x-8)+' 194 '
          +x+' 206" fill="none"'
          +' stroke="#F59E0B"'
          +' stroke-width="3"/>';
      }

      html+='<path class="pcr-flow"'
        +' d="M217 215 V174"'
        +' fill="none"'
        +' stroke="#EF4444"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>';

      html+='<text x="217" y="123"'
        +' text-anchor="middle"'
        +' font-size="15"'
        +' font-weight="900"'
        +' fill="#B91C1C">'
        +'约95℃'
        +'</text>';

      return html;
    }

    function drawAnnealing(){
      var html='';

      html+=line(
        72,
        145,
        360,
        145,
        '#2563EB',
        7,
        ''
      );

      html+=line(
        72,
        241,
        360,
        241,
        '#EC4899',
        7,
        ''
      );

      html+=line(
        86,
        222,
        164,
        222,
        '#10B981',
        8,
        ' class="pcr-pulse" marker-end="url(#${rootId}-arrow-blue)"'
      );

      html+=line(
        346,
        164,
        268,
        164,
        '#8B5CF6',
        8,
        ' class="pcr-pulse" marker-end="url(#${rootId}-arrow-pink)"'
      );

      html+='<text x="124" y="211"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#047857">'
        +'正向引物 5′→3′'
        +'</text>';

      html+='<text x="307" y="184"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#6D28D9">'
        +'反向引物 5′→3′'
        +'</text>';

      return html;
    }

    function drawExtension(){
      var html=
        drawAnnealing();

      html+=line(
        164,
        222,
        335,
        222,
        '#10B981',
        7,
        ' class="pcr-rise" marker-end="url(#${rootId}-arrow-blue)"'
      );

      html+=line(
        268,
        164,
        97,
        164,
        '#8B5CF6',
        7,
        ' class="pcr-rise" marker-end="url(#${rootId}-arrow-pink)"'
      );

      html+='<ellipse cx="218" cy="208"'
        +' rx="34" ry="25"'
        +' fill="#FEF3C7"'
        +' stroke="#D97706"'
        +' stroke-width="4"/>';

      html+='<text x="218" y="205"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#92400E">'
        +'DNA'
        +'</text>';

      html+='<text x="218" y="219"'
        +' text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#92400E">'
        +'聚合酶'
        +'</text>';

      return html;
    }

    function drawResult(){
      var html='';
      var positions=[
        128,
        224
      ];

      for(var i=0;i<positions.length;i++){
        var y=positions[i];

        html+=line(
          84,
          y,
          352,
          y,
          '#2563EB',
          6,
          ''
        );

        html+=line(
          84,
          y+22,
          352,
          y+22,
          '#EC4899',
          6,
          ''
        );

        html+='<text x="218" y="'
          +(y-11)
          +'" text-anchor="middle"'
          +' font-size="10"'
          +' font-weight="900"'
          +' fill="#475569">'
          +'子代DNA '+(i+1)
          +'</text>';
      }

      html+='<path class="pcr-flow"'
        +' d="M218 180 V207"'
        +' fill="none"'
        +' stroke="#2563EB"'
        +' stroke-width="4"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>';

      html+='<text x="218" y="288"'
        +' text-anchor="middle"'
        +' font-size="12"'
        +' font-weight="900"'
        +' fill="#1D4ED8">'
        +'进入下一循环继续作为模板'
        +'</text>';

      return html;
    }

    function renderMolecule(){
      var html='';

      if(stage==='denaturation'){
        html=drawDenaturation();
      }else if(stage==='annealing'){
        html=drawAnnealing();
      }else if(stage==='extension'){
        html=drawExtension();
      }else{
        html=drawResult();
      }

      moleculeLayer.innerHTML=html;
      labelLayer.innerHTML=strandLabels();

      labelLayer.style.display=
        showLabels?'':'none';
    }

    function renderCurve(
      model,
      cycles
    ){
      var left=458;
      var right=716;
      var top=122;
      var bottom=270;
      var width=right-left;
      var height=bottom-top;
      var maxCycle=35;

      var idealMax=
        model.profile.template>0
          ?model.profile.template
            *Math.pow(
              2,
              maxCycle
            )
          :10;

      var maxLog=Math.max(
        1,
        Math.log(
          Math.max(
            10,
            idealMax+1
          )
        )/Math.LN10
      );

      var idealPath='';
      var actualPath='';
      var html='';

      html+=line(
        left,
        bottom,
        right,
        bottom,
        '#64748B',
        2,
        ''
      );

      html+=line(
        left,
        bottom,
        left,
        top,
        '#64748B',
        2,
        ''
      );

      for(var yIndex=0;yIndex<=4;yIndex++){
        var gy=
          bottom-height*yIndex/4;

        html+=line(
          left,
          gy,
          right,
          gy,
          '#E2E8F0',
          1,
          ''
        );
      }

      for(var i=0;i<=maxCycle;i++){
        var x=
          left+width*i/maxCycle;

        var idealValue=
          model.profile.template>0
            ?model.profile.template
              *Math.pow(
                2,
                i
              )
            :0;

        var actualValue=
          model.profile.template>0
            ?model.profile.template
              *Math.pow(
                1+model.effectiveEfficiency,
                i
              )
            :0;

        var idealLog=
          idealValue>0
            ?Math.log(
              idealValue+1
            )/Math.LN10
            :0;

        var actualLog=
          actualValue>0
            ?Math.log(
              actualValue+1
            )/Math.LN10
            :0;

        var idealY=
          bottom-height
          *clamp(
            idealLog/maxLog,
            0,
            1
          );

        var actualY=
          bottom-height
          *clamp(
            actualLog/maxLog,
            0,
            1
          );

        idealPath+=(i===0?'M':' L')
          +x+' '+idealY;

        actualPath+=(i===0?'M':' L')
          +x+' '+actualY;
      }

      html+='<path d="'
        +idealPath
        +'" fill="none"'
        +' stroke="#94A3B8"'
        +' stroke-width="3"'
        +' stroke-dasharray="7 6"/>';

      html+='<path d="'
        +actualPath
        +'" fill="none"'
        +' stroke="#2563EB"'
        +' stroke-width="5"'
        +' stroke-linecap="round"/>';

      var currentX=
        left+width*cycles/maxCycle;

      var currentLog=
        model.theoretical>0
          ?Math.log(
            model.theoretical+1
          )/Math.LN10
          :0;

      var currentY=
        bottom-height
        *clamp(
          currentLog/maxLog,
          0,
          1
        );

      html+='<circle class="pcr-pulse"'
        +' cx="'+currentX
        +'" cy="'+currentY
        +'" r="8"'
        +' fill="#FFFFFF"'
        +' stroke="#EF4444"'
        +' stroke-width="4"/>';

      html+='<text x="'+currentX
        +'" y="'+Math.max(top+10,currentY-14)
        +'" text-anchor="middle"'
        +' font-size="10"'
        +' font-weight="900"'
        +' fill="#B91C1C">'
        +'第'+cycles+'循环'
        +'</text>';

      html+='<text x="'+left
        +'" y="'+(bottom+18)
        +'" font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">0</text>';

      html+='<text x="'+right
        +'" y="'+(bottom+18)
        +'" text-anchor="end"'
        +' font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'35循环'
        +'</text>';

      html+='<text x="'+(left+12)
        +'" y="'+(top-6)
        +'" font-size="10"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'相对产物量（对数示意）'
        +'</text>';

      html+='<line x1="470" y1="291"'
        +' x2="500" y2="291"'
        +' stroke="#94A3B8"'
        +' stroke-width="3"'
        +' stroke-dasharray="7 6"/>';

      html+='<text x="508" y="295"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#64748B">'
        +'理想100%倍增'
        +'</text>';

      html+='<line x1="606" y1="291"'
        +' x2="636" y2="291"'
        +' stroke="#2563EB"'
        +' stroke-width="5"/>';

      html+='<text x="644" y="295"'
        +' font-size="9.5"'
        +' font-weight="800"'
        +' fill="#1D4ED8">'
        +'当前条件'
        +'</text>';

      curveLayer.innerHTML=html;
    }

    function controlInterpretation(model){
      if(control==='positive'){
        return model.theoretical>0
          ?'阳性对照出现扩增，说明本教学模型中的反应体系能够工作。'
          :'阳性对照没有扩增，真实实验中应检查试剂、程序、模板和操作。';
      }

      if(
        control==='negative'
        ||control==='blank'
      ){
        return '当前模型中该对照不含目标模板，因此理论产物为0；若真实实验出现扩增，应优先排查污染或非特异扩增。';
      }

      return model.theoretical>0
        ?'待测样品在当前教学条件下形成扩增曲线，但不能据此直接得出临床或个体身份结论。'
        :'待测样品未形成扩增，也不能单独证明目标序列绝对不存在。';
    }

    function update(){
      var cycles=clamp(
        Math.round(
          Number(cycleInput.value)
        ),
        1,
        35
      );

      var annealTemperature=
        clamp(
          Number(annealInput.value),
          40,
          72
        );

      var efficiency=
        clamp(
          Number(efficiencyInput.value),
          40,
          100
        );

      var initialTemplate=
        clamp(
          Number(templateInput.value),
          1,
          100
        );

      var info=
        stageInfo[stage];

      var model=
        amplificationModel(
          cycles,
          efficiency,
          annealTemperature,
          initialTemplate,
          control
        );

      var specificity=
        specificityState(
          annealTemperature
        );

      cycleValue.textContent=
        cycles+' 次';

      annealValue.textContent=
        annealTemperature.toFixed(0)+'℃';

      efficiencyValue.textContent=
        efficiency.toFixed(0)+'%';

      templateValue.textContent=
        initialTemplate.toFixed(0)
        +' 相对单位';

      setActive(
        stageButtons,
        'data-stage',
        stage
      );

      setActive(
        controlButtons,
        'data-control',
        control
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
        '--pcr-flow-speed',
        clamp(
          2.6-efficiency/58,
          .6,
          2.4
        ).toFixed(2)+'s'
      );

      title.textContent=
        info.title;

      summary.textContent=
        info.summary;

      statusCycle.textContent=
        cycles+'×';

      statusCopies.textContent=
        formatCopies(
          model.theoretical
        );

      statusControl.textContent=
        model.profile.label;

      renderMolecule();

      renderCurve(
        model,
        cycles
      );

      tempOneLabel.textContent=
        '变性约95℃';

      tempTwoLabel.textContent=
        '退火 '
        +annealTemperature.toFixed(0)
        +'℃';

      tempThreeLabel.textContent=
        '延伸约72℃';

      tempOneBar.setAttribute(
        'width',
        '162'
      );

      tempTwoBar.setAttribute(
        'width',
        String(
          226*model.annealFactor
        )
      );

      tempThreeBar.setAttribute(
        'width',
        String(
          170*efficiency/100
        )
      );

      panelNote.textContent=
        specificity.label
        +'｜有效扩增效率约 '
        +(model.effectiveEfficiency*100)
          .toFixed(0)
        +'%';

      footerNote.textContent=
        '计算示意：N=N0×(1+E)^n；实际PCR还会受引物设计、模板质量、试剂限制和平台期影响。';

      result.innerHTML=
        info.note
        +'<br>'+specificity.text
        +' '+model.profile.explanation
        +' '+controlInterpretation(model)
        +' 本模型只用于原理教学，不提供实验处方或诊断结论。';
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

      annealInput.value=
        String(data.anneal);

      efficiencyInput.value=
        String(data.efficiency);

      cycleInput.value=
        String(data.cycle);

      currentScenario=name;

      if(pauseAutomatic){
        automatic=false;
      }

      update();
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
        2100
      );
    }

    for(var i=0;i<stageButtons.length;i++){
      stageButtons[i].onclick=function(){
        automatic=false;

        stage=this.getAttribute(
          'data-stage'
        );

        update();
        schedule();
      };
    }

    for(var j=0;j<controlButtons.length;j++){
      controlButtons[j].onclick=function(){
        control=this.getAttribute(
          'data-control'
        );

        update();
      };
    }

    for(var k=0;k<scenarioButtons.length;k++){
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

    cycleInput.oninput=function(){
      currentScenario='';
      update();
    };

    annealInput.oninput=function(){
      currentScenario='';
      update();
    };

    efficiencyInput.oninput=function(){
      currentScenario='';
      update();
    };

    templateInput.oninput=function(){
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
