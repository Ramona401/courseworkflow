/**
 * lifeScienceLabTemplatesRegulationSensoryProcessing.ts
 *
 * 平面生命科学实验室：
 * 感觉形成与中枢信息加工。
 *
 * 教学目标：
 * 1. 理解感受器只负责接受刺激并完成刺激转换，
 *    感觉不是在感受器中直接形成；
 * 2. 理解传入神经把动作电位编码的信息传向中枢神经系统；
 * 3. 认识感觉通路中可存在脊髓、脑干、丘脑等不同层级的
 *    中继、筛选和初步整合；
 * 4. 理解不同感觉通路的具体中继结构并不完全相同；
 * 5. 理解视觉、听觉和躯体感觉等信息最终需要在
 *    相应皮层区域及协同神经网络中进一步加工；
 * 6. 理解感觉信息包含刺激类型、位置、强度和时间变化等特征；
 * 7. 理解中枢神经系统可对这些特征进行并行处理和综合；
 * 8. 理解注意水平可影响中枢对某一路感觉信息的选择和加工；
 * 9. 理解背景噪声会增加有效信息辨认的难度；
 * 10. 理解已有经验和情境线索可影响识别与解释，
 *     但不能改变外界物理刺激本身；
 * 11. 理解感觉是中枢神经系统对传入信息加工后形成的体验，
 *     不是外界刺激的机械复制；
 * 12. 理解快速反射活动可以在形成清晰感觉之前启动，
 *     但感觉仍需要相应中枢加工。
 *
 * 科学边界：
 * 1. 感受器接受适宜刺激后形成感受器电位，
 *    达到阈值后可引起传入神经纤维动作电位；
 * 2. 单根神经纤维动作电位幅度具有全或无特征，
 *    刺激强度通常由频率、募集和群体活动等方式编码；
 * 3. 不同感觉通路具有不同的解剖结构，
 *    本模型用“传入—中继—皮层与协同网络”表示通用过程；
 * 4. 多数感觉通路可经过丘脑等中继结构，
 *    但不能把所有感觉通路都简化为完全相同的路线；
 * 5. 感觉信息加工通常涉及多个脑区和神经网络协同，
 *    不应把感觉形成理解为某一个孤立点的单独功能；
 * 6. 注意可以增强对目标信息的选择和加工，
 *    但不等于注意本身产生了外界刺激；
 * 7. 情境和经验可帮助识别含义，
 *    也可能造成误判，因此识别结果不是对刺激的绝对复制；
 * 8. 背景噪声既可指外界竞争刺激，
 *    也可代表中枢处理中的无关活动，本模型只作相对比较；
 * 9. 反射活动和感觉形成可以并行发生，
 *    反射较快不代表大脑或高级中枢完全不参与后续调节；
 * 10. 图中的传导完整度、中继效率、中枢激活、
 *     感觉清晰度和识别结果均为教学相对值；
 * 11. 本模板不模拟具体脑损伤、感觉障碍、意识状态或疾病；
 * 12. 本模板只用于生物学教学，
 *     不用于神经功能、心理状态或医学诊断。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部图片、脚本、字体或CDN；
 * 3. 所有DOM查询均限定在rootId内部；
 * 4. 支持同一课件页面插入多个独立实例；
 * 5. 使用生命科学统一.bl-*布局协议；
 * 6. 支持视觉、听觉和躯体感觉三类通路切换；
 * 7. 支持参数滑杆、四种观察方式、自动推进、
 *    结构标注开关、动态图示和即时教学结论；
 * 8. 本文件只导出独立模板数组；
 * 9. 聚合入口将在后续C1批次统一接入。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

/**
 * 安全读取数值参数。
 *
 * 参数不存在、类型错误或者不是有限数值时，
 * 使用模板内定义的默认值。
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
 * 将参数转成适合写入HTML属性的简洁数字文本。
 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/**
 * 模板独立样式。
 *
 * 所有选择器均带rootId前缀，
 * 避免同一页面中的多个模板实例互相影响。
 */
function sensoryProcessingStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#EEF2FF);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#7C3AED}'
    + '#' + rootId + ' .sp-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .sp-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .sp-button{min-height:30px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .sp-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .sp-channel-button{min-height:29px;padding:3px;border:1px solid #A5B4FC;border-radius:8px;background:#fff;color:#3730A3;font-size:9.8px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .sp-channel-button.active{border-color:#4F46E5;background:#E0E7FF;box-shadow:0 3px 9px rgba(79,70,229,.14)}'
    + '#' + rootId + ' .sp-channel-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .sp-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .sp-toggle.off{background:#64748B}'
    + '#' + rootId + ' .sp-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .sp-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .sp-card b{display:block;font-size:13px;color:#6D28D9}'
    + '#' + rootId + ' .sp-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .sp-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--sp-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .sp-pulse{animation:' + rootId + '-pulse 1.45s ease-in-out infinite}'
    + '#' + rootId + ' .sp-breathe{animation:' + rootId + '-breathe 1.8s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.30}50%{opacity:1}}'
    + '@keyframes ' + rootId + '-breathe{0%,100%{transform:scale(.94);opacity:.58}50%{transform:scale(1.04);opacity:1}}'
    + '</style>'
}

/**
 * 避免在源码中直接写死闭合script标签，
 * 防止模板字符串在HTML解析时提前结束。
 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SENSORY_PROCESSING:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-sensory-central-processing',
    group: '🧠 稳态与调节',
    name: '感觉形成与中枢信息加工',
    emoji: '🧩',
    desc: '切换视觉、听觉和躯体感觉通路，调节信号、通路完整度、中继效率、注意、背景噪声和情境匹配，观察感觉形成过程',
    params: [
      {
        key: 'signalStrength',
        label: '传入信号强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
        hint: '表示感受器和传入神经产生的相对信号强度',
      },
      {
        key: 'pathwayIntegrity',
        label: '感觉通路完整度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 88,
        hint: '只用于比较传入信息在通路中的相对保留程度',
      },
      {
        key: 'relayEfficiency',
        label: '中继整合效率',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
        hint: '表示中继、筛选和初步整合的相对效率',
      },
      {
        key: 'attentionLevel',
        label: '注意聚焦水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 66,
        hint: '注意影响目标信息选择，不改变外界刺激本身',
      },
      {
        key: 'backgroundNoise',
        label: '背景干扰强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 28,
        hint: '表示竞争刺激或无关活动带来的相对干扰',
      },
      {
        key: 'contextMatch',
        label: '情境线索匹配度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 74,
        hint: '表示已有经验和当前情境对识别的辅助程度',
      },
      {
        key: 'processTime',
        label: '加工过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 40,
        hint: '控制传入、中继、整合和感觉形成过程的演示进度',
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const signalStrength = num(
        params,
        'signalStrength',
        72,
      )
      const pathwayIntegrity = num(
        params,
        'pathwayIntegrity',
        88,
      )
      const relayEfficiency = num(
        params,
        'relayEfficiency',
        82,
      )
      const attentionLevel = num(
        params,
        'attentionLevel',
        66,
      )
      const backgroundNoise = num(
        params,
        'backgroundNoise',
        28,
      )
      const contextMatch = num(
        params,
        'contextMatch',
        74,
      )
      const processTime = num(
        params,
        'processTime',
        40,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${sensoryProcessingStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧩 感觉形成与中枢信息加工</div>
    <div class="bl-note">感觉在相应中枢及协同神经网络加工后形成</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>传入信号强度</span>
          <span class="bl-value" data-signal-value></span>
        </div>
        <input
          data-signal
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(signalStrength)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>感觉通路完整度</span>
          <span class="bl-value" data-pathway-value></span>
        </div>
        <input
          data-pathway
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(pathwayIntegrity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>中继整合效率</span>
          <span class="bl-value" data-relay-value></span>
        </div>
        <input
          data-relay
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(relayEfficiency)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>注意聚焦水平</span>
          <span class="bl-value" data-attention-value></span>
        </div>
        <input
          data-attention
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(attentionLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>背景干扰强度</span>
          <span class="bl-value" data-noise-value></span>
        </div>
        <input
          data-noise
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(backgroundNoise)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>情境线索匹配度</span>
          <span class="bl-value" data-context-value></span>
        </div>
        <input
          data-context
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(contextMatch)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>加工过程时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input
          data-time
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(processTime)}"
        >
      </div>

      <div class="sp-subtitle">选择感觉通路</div>

      <div class="sp-channel-buttons">
        <button
          type="button"
          class="sp-channel-button active"
          data-channel="visual"
        >视觉</button>

        <button
          type="button"
          class="sp-channel-button"
          data-channel="auditory"
        >听觉</button>

        <button
          type="button"
          class="sp-channel-button"
          data-channel="somatic"
        >躯体感觉</button>
      </div>

      <div class="sp-subtitle">观察方式</div>

      <div class="sp-buttons">
        <button
          type="button"
          class="sp-button active"
          data-mode="pathway"
        >感觉通路与中继</button>

        <button
          type="button"
          class="sp-button"
          data-mode="features"
        >特征并行加工</button>

        <button
          type="button"
          class="sp-button"
          data-mode="attention"
        >注意选择与干扰</button>

        <button
          type="button"
          class="sp-button"
          data-mode="perception"
        >感觉形成与识别</button>
      </div>

      <button
        type="button"
        class="sp-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="sp-toggle"
        data-auto
      >加工推进：运行中</button>

      <div class="sp-status">
        <div class="sp-card">
          <b data-activation></b>
          <span>中枢激活</span>
        </div>

        <div class="sp-card">
          <b data-clarity></b>
          <span>感觉清晰度</span>
        </div>

        <div class="sp-card">
          <b data-recognition></b>
          <span>识别结果</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="感觉形成与中枢信息加工互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-purple"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
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

          <marker
            id="${rootId}-arrow-green"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker
            id="${rootId}-arrow-orange"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#EA580C"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#4C1D95"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text
          x="24"
          y="36"
          data-title
          font-size="26"
          font-weight="900"
          fill="#5B21B6"
        ></text>

        <text
          x="24"
          y="65"
          data-summary
          font-size="14"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-dynamic></g>
        <g data-labels></g>

        <g transform="translate(514 337)">
          <rect
            width="220"
            height="66"
            rx="15"
            fill="#F5F3FF"
            stroke="#DDD6FE"
            stroke-width="2"
          />

          <text
            x="110"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#5B21B6"
          >关键边界</text>

          <text
            x="110"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#4C1D95"
          >感觉不是在感受器或传入神经中形成</text>

          <text
            x="110"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#4C1D95"
          >感觉加工涉及相关中枢与协同网络</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#5B21B6"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var signalInput=root.querySelector(
      '[data-signal]'
    );
    var pathwayInput=root.querySelector(
      '[data-pathway]'
    );
    var relayInput=root.querySelector(
      '[data-relay]'
    );
    var attentionInput=root.querySelector(
      '[data-attention]'
    );
    var noiseInput=root.querySelector(
      '[data-noise]'
    );
    var contextInput=root.querySelector(
      '[data-context]'
    );
    var timeInput=root.querySelector(
      '[data-time]'
    );

    var signalValue=root.querySelector(
      '[data-signal-value]'
    );
    var pathwayValue=root.querySelector(
      '[data-pathway-value]'
    );
    var relayValue=root.querySelector(
      '[data-relay-value]'
    );
    var attentionValue=root.querySelector(
      '[data-attention-value]'
    );
    var noiseValue=root.querySelector(
      '[data-noise-value]'
    );
    var contextValue=root.querySelector(
      '[data-context-value]'
    );
    var timeValue=root.querySelector(
      '[data-time-value]'
    );

    var channelButtons=root.querySelectorAll(
      '[data-channel]'
    );
    var modeButtons=root.querySelectorAll(
      '[data-mode]'
    );

    var labelToggle=root.querySelector(
      '[data-label-toggle]'
    );
    var autoButton=root.querySelector(
      '[data-auto]'
    );

    var activationText=root.querySelector(
      '[data-activation]'
    );
    var clarityText=root.querySelector(
      '[data-clarity]'
    );
    var recognitionText=root.querySelector(
      '[data-recognition]'
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
    var stageNote=root.querySelector(
      '[data-stage-note]'
    );
    var dynamic=root.querySelector(
      '[data-dynamic]'
    );
    var labels=root.querySelector(
      '[data-labels]'
    );

    var mode='pathway';
    var channel='visual';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    var channelInformation={
      visual:{
        name:'视觉',
        receptor:'视网膜感受细胞',
        nerve:'视觉传入通路',
        relay:'中继与初步整合',
        cortex:'视觉皮层及协同网络',
        modality:'光的明暗、颜色和形状等',
        location:'视野中的空间位置',
        color:'#7C3AED',
        pale:'#EDE9FE',
        stroke:'#5B21B6',
        icon:'eye'
      },
      auditory:{
        name:'听觉',
        receptor:'耳蜗毛细胞',
        nerve:'听觉传入通路',
        relay:'脑干及其他中继结构',
        cortex:'听觉皮层及协同网络',
        modality:'声音频率、强度和音色等',
        location:'声音来源方向和距离线索',
        color:'#0284C7',
        pale:'#E0F2FE',
        stroke:'#0369A1',
        icon:'ear'
      },
      somatic:{
        name:'躯体感觉',
        receptor:'皮肤或本体感受器',
        nerve:'躯体感觉传入通路',
        relay:'脊髓、脑干及其他中继',
        cortex:'躯体感觉皮层及协同网络',
        modality:'触压、温度、疼痛或位置等',
        location:'身体部位和空间定位',
        color:'#16A34A',
        pale:'#DCFCE7',
        stroke:'#15803D',
        icon:'skin'
      }
    };

    function clamp(value,min,max){
      return Math.max(
        min,
        Math.min(max,value)
      );
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

      timer=window.setTimeout(function(){
        var next=Number(timeInput.value)+2;

        timeInput.value=String(
          next>100?0:next
        );

        update();
        schedule();
      },780);
    }

    function stageProgress(
      progress,
      start,
      end
    ){
      return clamp(
        (progress-start)/(end-start),
        0,
        1
      );
    }

    function channelIcon(
      type,
      x,
      y,
      scale,
      active
    ){
      var info=channelInformation[type];
      var glow=active
        ?'<circle class="sp-pulse" cx="0" cy="0" r="52" fill="'+info.color+'" opacity=".25"/>'
        :'';
      var body='';

      if(type==='visual'){
        body=''
          +'<path d="M-46 0 Q0 -42 46 0 Q0 42 -46 0Z" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="6"/>'
          +'<circle cx="0" cy="0" r="20" fill="'+info.color+'" stroke="#FFFFFF" stroke-width="5"/>'
          +'<circle cx="0" cy="0" r="7" fill="#1E1B4B"/>';
      }else if(type==='auditory'){
        body=''
          +'<path d="M7 -42 C-26 -45 -44 -20 -41 8 C-38 34 -18 42 -9 23 C-2 7 -11 -4 -21 4 C-30 11 -23 25 -12 22 C4 18 12 7 13 -8 C14 -19 21 -26 32 -23 C42 -20 45 -7 41 5" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="6" stroke-linecap="round"/>'
          +'<path d="M8 24 C8 36 17 42 28 39" fill="none" stroke="'+info.stroke+'" stroke-width="6" stroke-linecap="round"/>';
      }else{
        body=''
          +'<rect x="-43" y="-34" width="86" height="68" rx="22" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="6"/>'
          +'<path d="M-31 -8 Q-18 -24 -5 -8 T21 -8 T39 -8" fill="none" stroke="'+info.color+'" stroke-width="6" stroke-linecap="round"/>'
          +'<circle cx="-20" cy="14" r="8" fill="'+info.color+'"/>'
          +'<circle cx="5" cy="16" r="8" fill="'+info.color+'"/>'
          +'<circle cx="29" cy="13" r="8" fill="'+info.color+'"/>';
      }

      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')">'
        +glow
        +body
        +'</g>';
    }

    function brainNetwork(
      x,
      y,
      scale,
      color,
      activity
    ){
      var opacity=.26+.72*activity/100;
      var width=2.5+activity/28;

      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<path d="M-59 -4 C-67 -34 -43 -57 -15 -53 C3 -70 36 -63 43 -42 C66 -36 70 -5 56 10 C64 35 41 57 17 52 C-1 69 -32 60 -38 39 C-62 35 -70 12 -59 -4Z" fill="#F5F3FF" stroke="'+color+'" stroke-width="5"/>'
        +'<circle cx="-31" cy="-22" r="8" fill="'+color+'"/>'
        +'<circle cx="3" cy="-35" r="8" fill="'+color+'"/>'
        +'<circle cx="32" cy="-14" r="8" fill="'+color+'"/>'
        +'<circle cx="-19" cy="17" r="8" fill="'+color+'"/>'
        +'<circle cx="20" cy="25" r="8" fill="'+color+'"/>'
        +'<path d="M-31 -22 L3 -35 L32 -14 L20 25 L-19 17 Z M3 -35 L-19 17 M-31 -22 L20 25" fill="none" stroke="'+color+'" stroke-width="'+width+'" stroke-linecap="round"/>'
        +'</g>';
    }

    function signalDots(
      startX,
      endX,
      y,
      strength,
      progress,
      color
    ){
      var html='';
      var count=Math.max(
        2,
        Math.floor(
          2+strength/14
        )
      );

      for(var i=0;i<count;i++){
        var offset=i/count;
        var position=(
          progress
          +offset
        )%1;
        var x=startX
          +(endX-startX)*position;

        html+=''
          +'<circle cx="'+x.toFixed(1)+'" cy="'+y+'" r="'+(4+strength/35).toFixed(1)+'" fill="'+color+'" opacity="'+(.30+.60*strength/100).toFixed(2)+'"/>';
      }

      return html;
    }

    function renderPathway(
      info,
      progress,
      transmitted,
      relayed,
      activation
    ){
      var p1=stageProgress(
        progress,
        0,
        .27
      );
      var p2=stageProgress(
        progress,
        .20,
        .53
      );
      var p3=stageProgress(
        progress,
        .45,
        .78
      );
      var p4=stageProgress(
        progress,
        .70,
        1
      );

      var x1=122;
      var x2=302;
      var x3=482;
      var x4=654;

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="55" y="126" width="134" height="172" rx="23" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="4"/>'
        +'<rect x="235" y="126" width="134" height="172" rx="23" fill="#EEF2FF" stroke="#818CF8" stroke-width="4"/>'
        +'<rect x="415" y="126" width="134" height="172" rx="23" fill="#F5F3FF" stroke="#A78BFA" stroke-width="4"/>'
        +'<rect x="587" y="126" width="134" height="172" rx="23" fill="#FAF5FF" stroke="#C084FC" stroke-width="4"/>'
        +'</g>'
        +channelIcon(channel,x1,185,.67,p1>.25)
        +'<text x="'+x1+'" y="259" text-anchor="middle" font-size="12" font-weight="900" fill="'+info.stroke+'">'+info.receptor+'</text>'
        +'<text x="'+x1+'" y="280" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">刺激转换和初始编码</text>'
        +'<path d="M189 210 H235" fill="none" stroke="#A5B4FC" stroke-width="9" stroke-linecap="round"/>'
        +'<path class="sp-flow" d="M189 210 H235" fill="none" stroke="'+info.color+'" stroke-width="'+(3+transmitted/24)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +signalDots(191,230,210,transmitted,p1,info.color)
        +'<g transform="translate('+x2+' 190)">'
        +'<path d="M0 -56 V45" stroke="#6366F1" stroke-width="16" stroke-linecap="round"/>'
        +'<path d="M-30 -25 C-8 -9 -6 2 0 17 C8 3 14 -12 34 -26" fill="none" stroke="#A5B4FC" stroke-width="10" stroke-linecap="round"/>'
        +'<circle cx="0" cy="-48" r="15" fill="#4F46E5" stroke="#FFFFFF" stroke-width="4"/>'
        +'<circle cx="0" cy="40" r="15" fill="#4F46E5" stroke="#FFFFFF" stroke-width="4"/>'
        +'</g>'
        +'<text x="'+x2+'" y="259" text-anchor="middle" font-size="12" font-weight="900" fill="#3730A3">'+info.nerve+'</text>'
        +'<text x="'+x2+'" y="280" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">动作电位沿传入通路传播</text>'
        +'<path d="M369 210 H415" fill="none" stroke="#DDD6FE" stroke-width="9" stroke-linecap="round"/>'
        +'<path class="sp-flow" d="M369 210 H415" fill="none" stroke="#7C3AED" stroke-width="'+(3+relayed/24)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +signalDots(371,410,210,relayed,p2,'#7C3AED')
        +'<g transform="translate('+x3+' 195)">'
        +'<ellipse cx="0" cy="-25" rx="39" ry="28" fill="#EDE9FE" stroke="#7C3AED" stroke-width="5"/>'
        +'<ellipse cx="0" cy="29" rx="45" ry="31" fill="#F5F3FF" stroke="#A78BFA" stroke-width="5"/>'
        +'<path class="sp-flow" d="M0 -2 V10" fill="none" stroke="#7C3AED" stroke-width="6" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'</g>'
        +'<text x="'+x3+'" y="259" text-anchor="middle" font-size="12" font-weight="900" fill="#5B21B6">'+info.relay+'</text>'
        +'<text x="'+x3+'" y="280" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">中继、筛选和初步整合</text>'
        +'<path d="M549 210 H587" fill="none" stroke="#E9D5FF" stroke-width="9" stroke-linecap="round"/>'
        +'<path class="sp-flow" d="M549 210 H587" fill="none" stroke="#9333EA" stroke-width="'+(3+activation/24)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +signalDots(551,582,210,activation,p3,'#9333EA')
        +brainNetwork(
          x4,
          195,
          .72,
          info.color,
          activation*p4
        )
        +'<text x="'+x4+'" y="259" text-anchor="middle" font-size="12" font-weight="900" fill="#6B21A8">'+info.cortex+'</text>'
        +'<text x="'+x4+'" y="280" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">进一步加工并形成感觉体验</text>'
        +'<path d="M290 298 C285 326 239 334 189 331" fill="none" stroke="#94A3B8" stroke-width="3" stroke-dasharray="7 6" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<text x="299" y="326" text-anchor="middle" font-size="9.5" font-weight="900" fill="#64748B">部分信息可进入较快反射通路</text>'
        +'<g transform="translate(77 335)">'
        +'<rect width="578" height="30" rx="14" fill="#F5F3FF" stroke="#DDD6FE" stroke-width="2"/>'
        +'<text x="289" y="20" text-anchor="middle" font-size="10.5" font-weight="900" fill="#5B21B6">不同感觉通路的具体结构并不完全相同，本图展示“感受—传入—中继—中枢加工”的通用关系。</text>'
        +'</g>';
    }

    function featureCard(
      x,
      titleText,
      valueText,
      value,
      color,
      pale
    ){
      var barWidth=clamp(
        value,
        0,
        100
      )*1.55;

      return ''
        +'<g transform="translate('+x+' 127)">'
        +'<rect width="190" height="173" rx="21" fill="'+pale+'" stroke="'+color+'" stroke-width="4"/>'
        +'<text x="95" y="29" text-anchor="middle" font-size="13" font-weight="900" fill="'+color+'">'+titleText+'</text>'
        +'<circle cx="95" cy="80" r="'+(20+value/8)+'" fill="'+color+'" opacity="'+(.18+.55*value/100)+'"/>'
        +'<circle cx="95" cy="80" r="12" fill="'+color+'" stroke="#FFFFFF" stroke-width="4"/>'
        +'<rect x="18" y="122" width="155" height="15" rx="7" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>'
        +'<rect x="18" y="122" width="'+barWidth.toFixed(1)+'" height="15" rx="7" fill="'+color+'"/>'
        +'<text x="95" y="158" text-anchor="middle" font-size="10" font-weight="900" fill="#475569">'+valueText+'</text>'
        +'</g>';
    }

    function renderFeatures(
      info,
      modalityValue,
      locationValue,
      intensityValue,
      integration
    ){
      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +featureCard(
          47,
          '刺激类型特征',
          info.modality,
          modalityValue,
          info.color,
          info.pale
        )
        +featureCard(
          255,
          '空间位置特征',
          info.location,
          locationValue,
          '#2563EB',
          '#EFF6FF'
        )
        +featureCard(
          463,
          '强度与时间特征',
          '相对强度和变化节律',
          intensityValue,
          '#EA580C',
          '#FFF7ED'
        )
        +'<path class="sp-flow" d="M142 303 C182 325 244 330 332 335" fill="none" stroke="'+info.color+'" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<path class="sp-flow" d="M350 303 V334" fill="none" stroke="#2563EB" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<path class="sp-flow" d="M558 303 C514 325 450 330 376 335" fill="none" stroke="#EA580C" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<g transform="translate(285 330)">'
        +'<rect width="190" height="37" rx="17" fill="#EDE9FE" stroke="#7C3AED" stroke-width="3"/>'
        +'<text x="95" y="16" text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6">多种特征综合表征</text>'
        +'<text x="95" y="31" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">整合指数 '+integration.toFixed(0)+'</text>'
        +'</g>';
    }

    function noiseWave(
      x,
      y,
      width,
      strength,
      color
    ){
      var path='';

      for(var i=0;i<=20;i++){
        var px=x+width*i/20;
        var py=y
          +Math.sin(i*1.9)*strength*.14
          +Math.cos(i*.73)*strength*.09;

        path+=(i===0?'M':' L')
          +px.toFixed(1)
          +' '
          +py.toFixed(1);
      }

      return path;
    }

    function renderAttention(
      info,
      attention,
      noise,
      gating,
      clarity
    ){
      var gateWidth=clamp(
        28+attention*.92-noise*.36,
        24,
        108
      );

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<g transform="translate(51 116)">'
        +'<rect width="209" height="221" rx="21" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="104" y="27" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">竞争感觉输入</text>'
        +'<path d="'+noiseWave(23,72,162,noise,'#94A3B8')+'" fill="none" stroke="#94A3B8" stroke-width="'+(2+noise/24)+'" opacity=".76"/>'
        +'<path d="'+noiseWave(23,116,162,noise*.86,'#F59E0B')+'" fill="none" stroke="#F59E0B" stroke-width="'+(2+noise/28)+'" opacity=".68"/>'
        +'<path d="'+noiseWave(23,160,162,noise*.72,'#EC4899')+'" fill="none" stroke="#EC4899" stroke-width="'+(2+noise/31)+'" opacity=".58"/>'
        +'<path class="sp-flow" d="M23 199 H184" fill="none" stroke="'+info.color+'" stroke-width="'+(4+attention/20)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<text x="104" y="213" text-anchor="middle" font-size="10" font-weight="900" fill="'+info.stroke+'">目标：'+info.name+'信息</text>'
        +'</g>'
        +'<g transform="translate(286 116)">'
        +'<rect width="188" height="221" rx="21" fill="#F5F3FF" stroke="#A78BFA" stroke-width="4"/>'
        +'<text x="94" y="27" text-anchor="middle" font-size="13" font-weight="900" fill="#5B21B6">注意选择与中枢筛选</text>'
        +'<path d="M32 77 H156" stroke="#DDD6FE" stroke-width="48" stroke-linecap="round"/>'
        +'<path d="M'+(94-gateWidth/2).toFixed(1)+' 77 H'+(94+gateWidth/2).toFixed(1)+'" stroke="#7C3AED" stroke-width="48" stroke-linecap="round" opacity=".82"/>'
        +'<circle cx="94" cy="77" r="18" fill="#FFFFFF" stroke="#5B21B6" stroke-width="5"/>'
        +'<text x="94" y="82" text-anchor="middle" font-size="10" font-weight="900" fill="#5B21B6">选择</text>'
        +'<rect x="26" y="132" width="136" height="16" rx="8" fill="#E2E8F0"/>'
        +'<rect x="26" y="132" width="'+(136*gating/100).toFixed(1)+'" height="16" rx="8" fill="#7C3AED"/>'
        +'<text x="94" y="169" text-anchor="middle" font-size="10" font-weight="900" fill="#475569">目标信息通过指数 '+gating.toFixed(0)+'</text>'
        +'<text x="94" y="194" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">注意影响加工优先级</text>'
        +'<text x="94" y="210" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">不改变物理刺激本身</text>'
        +'</g>'
        +'<g transform="translate(500 116)">'
        +'<rect width="205" height="221" rx="21" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="4"/>'
        +brainNetwork(
          102,
          82,
          .70,
          info.color,
          clarity
        )
        +'<text x="102" y="158" text-anchor="middle" font-size="13" font-weight="900" fill="'+info.stroke+'">目标感觉表征</text>'
        +'<rect x="24" y="177" width="157" height="16" rx="8" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>'
        +'<rect x="24" y="177" width="'+(157*clarity/100).toFixed(1)+'" height="16" rx="8" fill="'+info.color+'"/>'
        +'<text x="102" y="211" text-anchor="middle" font-size="10" font-weight="900" fill="#475569">感觉清晰度 '+clarity.toFixed(0)+'</text>'
        +'</g>'
        +'<path class="sp-flow" d="M260 227 H286" fill="none" stroke="#7C3AED" stroke-width="6" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<path class="sp-flow" d="M474 227 H500" fill="none" stroke="#7C3AED" stroke-width="6" marker-end="url(#${rootId}-arrow-purple)"/>';
    }

    function processNode(
      x,
      titleText,
      subtitleText,
      active,
      color,
      pale
    ){
      return ''
        +'<g transform="translate('+x+' 178)">'
        +'<circle class="'+(active?'sp-pulse':'')+'" cx="0" cy="0" r="48" fill="'+pale+'" stroke="'+color+'" stroke-width="'+(active?6:3)+'" opacity="'+(active?.96:.48)+'"/>'
        +'<circle cx="0" cy="0" r="17" fill="'+color+'" opacity="'+(active?.92:.35)+'"/>'
        +'<text x="0" y="72" text-anchor="middle" font-size="11.5" font-weight="900" fill="'+color+'">'+titleText+'</text>'
        +'<text x="0" y="89" text-anchor="middle" font-size="9.2" font-weight="800" fill="#64748B">'+subtitleText+'</text>'
        +'</g>';
    }

    function renderPerception(
      info,
      progress,
      activation,
      clarity,
      recognition,
      context
    ){
      var p1=progress>=.12;
      var p2=progress>=.33;
      var p3=progress>=.57;
      var p4=progress>=.78;

      var recognitionTextValue=recognition
        ?'形成较清晰识别'
        :'信息仍较模糊';

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +processNode(
          116,
          '感觉信号',
          '感受器与传入通路',
          p1,
          info.color,
          info.pale
        )
        +processNode(
          294,
          '中继与特征加工',
          '筛选、定位和特征提取',
          p2,
          '#4F46E5',
          '#EEF2FF'
        )
        +processNode(
          472,
          '多网络综合',
          '注意、记忆和情境参与',
          p3,
          '#7C3AED',
          '#F5F3FF'
        )
        +processNode(
          650,
          '感觉与识别',
          recognitionTextValue,
          p4&&recognition,
          recognition?'#16A34A':'#64748B',
          recognition?'#DCFCE7':'#F1F5F9'
        )
        +'<path class="sp-flow" d="M164 178 H246" fill="none" stroke="'+info.color+'" stroke-width="'+(p1?7:3)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="sp-flow" d="M342 178 H424" fill="none" stroke="#4F46E5" stroke-width="'+(p2?7:3)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<path class="sp-flow" d="M520 178 H602" fill="none" stroke="#7C3AED" stroke-width="'+(p3?7:3)+'" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<g transform="translate(73 302)">'
        +'<rect width="614" height="63" rx="18" fill="#FFFFFF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="307" y="20" text-anchor="middle" font-size="11.5" font-weight="900" fill="#5B21B6">感觉是中枢加工形成的体验，不是对外界刺激的机械复制</text>'
        +'<text x="307" y="39" text-anchor="middle" font-size="10" font-weight="800" fill="#475569">中枢激活 '+activation.toFixed(0)+'　感觉清晰度 '+clarity.toFixed(0)+'　情境匹配 '+context.toFixed(0)+'%</text>'
        +'<text x="307" y="55" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">快速反射活动可较早启动，但清晰感觉仍需要相应中枢及协同网络加工。</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='pathway'){
        labels.innerHTML=''
          +'<path d="M122 146 L122 85" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="61" y="80" font-size="12.5" font-weight="900" fill="#5B21B6">感受器与刺激转换</text>'
          +'<path d="M302 141 L302 88" stroke="#4F46E5" stroke-width="2.5"/>'
          +'<text x="246" y="82" font-size="12.5" font-weight="900" fill="#3730A3">传入神经通路</text>'
          +'<path d="M482 156 L521 96" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="528" y="91" font-size="12.5" font-weight="900" fill="#5B21B6">中继与初步整合</text>'
          +'<path d="M654 148 L685 91" stroke="#9333EA" stroke-width="2.5"/>'
          +'<text x="584" y="85" font-size="12.5" font-weight="900" fill="#6B21A8">相关皮层和协同网络</text>';
        return;
      }

      if(modeName==='features'){
        labels.innerHTML=''
          +'<path d="M142 165 L101 96" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="41" y="90" font-size="12.5" font-weight="900" fill="#5B21B6">刺激类型通道</text>'
          +'<path d="M350 165 L350 94" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="296" y="88" font-size="12.5" font-weight="900" fill="#1D4ED8">空间定位通道</text>'
          +'<path d="M558 165 L612 96" stroke="#EA580C" stroke-width="2.5"/>'
          +'<text x="619" y="90" font-size="12.5" font-weight="900" fill="#C2410C">强度时间通道</text>';
        return;
      }

      if(modeName==='attention'){
        labels.innerHTML=''
          +'<path d="M153 194 L113 92" stroke="#94A3B8" stroke-width="2.5"/>'
          +'<text x="49" y="87" font-size="12.5" font-weight="900" fill="#475569">竞争和无关输入</text>'
          +'<path d="M380 158 L380 91" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="313" y="85" font-size="12.5" font-weight="900" fill="#5B21B6">注意选择与筛选</text>'
          +'<path d="M602 166 L652 95" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="659" y="89" font-size="12.5" font-weight="900" fill="#5B21B6">目标感觉表征</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M116 137 L82 91" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="30" y="86" font-size="12.5" font-weight="900" fill="#5B21B6">传入信息</text>'
        +'<path d="M294 137 L294 90" stroke="#4F46E5" stroke-width="2.5"/>'
        +'<text x="235" y="84" font-size="12.5" font-weight="900" fill="#3730A3">特征加工</text>'
        +'<path d="M472 137 L515 91" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="522" y="85" font-size="12.5" font-weight="900" fill="#5B21B6">多网络综合</text>'
        +'<path d="M650 137 L698 91" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="632" y="85" font-size="12.5" font-weight="900" fill="#15803D">感觉与识别</text>';
    }

    function update(){
      var signal=Number(
        signalInput.value
      );
      var pathway=Number(
        pathwayInput.value
      );
      var relay=Number(
        relayInput.value
      );
      var attention=Number(
        attentionInput.value
      );
      var noise=Number(
        noiseInput.value
      );
      var context=Number(
        contextInput.value
      );
      var processTime=Number(
        timeInput.value
      );

      signalValue.textContent=
        signal.toFixed(0)+'%';
      pathwayValue.textContent=
        pathway.toFixed(0)+'%';
      relayValue.textContent=
        relay.toFixed(0)+'%';
      attentionValue.textContent=
        attention.toFixed(0)+'%';
      noiseValue.textContent=
        noise.toFixed(0)+'%';
      contextValue.textContent=
        context.toFixed(0)+'%';
      timeValue.textContent=
        processTime.toFixed(0)+'%';

      for(var i=0;i<channelButtons.length;i++){
        channelButtons[i].classList.toggle(
          'active',
          channelButtons[i].getAttribute(
            'data-channel'
          )===channel
        );
      }

      for(var j=0;j<modeButtons.length;j++){
        modeButtons[j].classList.toggle(
          'active',
          modeButtons[j].getAttribute(
            'data-mode'
          )===mode
        );
      }

      labelToggle.textContent=showLabels
        ?'结构标注：显示'
        :'结构标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      autoButton.textContent=automatic
        ?'加工推进：运行中'
        :'加工推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var info=channelInformation[channel];
      var progress=processTime/100;

      var transmitted=signal
        *pathway/100;

      var relayed=transmitted
        *relay/100;

      var attentionGain=
        .54+.46*attention/100;

      var noisePenalty=
        noise*.42;

      var contextContribution=
        context*.18;

      var centralActivation=clamp(
        relayed*attentionGain
        -noisePenalty
        +contextContribution,
        0,
        100
      );

      var gating=clamp(
        38
        +attention*.66
        -noise*.43,
        0,
        100
      );

      var clarity=clamp(
        centralActivation*.67
        +context*.27
        +gating*.18
        -noise*.24,
        0,
        100
      );

      var recognition=
        clarity>=56
        &&centralActivation>=42;

      var modalityValue=clamp(
        relayed*.78
        +attention*.16
        -noise*.22,
        0,
        100
      );

      var locationValue=clamp(
        relayed*.66
        +relay*.22
        -noise*.18,
        0,
        100
      );

      var intensityValue=clamp(
        signal*.62
        +pathway*.20
        +relay*.12
        -noise*.17,
        0,
        100
      );

      var integration=clamp(
        (
          modalityValue
          +locationValue
          +intensityValue
        )/3
        *.72
        +attention*.14
        +context*.14,
        0,
        100
      );

      activationText.textContent=
        centralActivation.toFixed(0);
      clarityText.textContent=
        clarity.toFixed(0);
      recognitionText.textContent=
        recognition?'较清晰':'较模糊';

      root.style.setProperty(
        '--sp-speed',
        clamp(
          2.45-centralActivation/72,
          .58,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='pathway'){
        title.textContent=
          info.name+'通路中的传入、中继与中枢加工';

        summary.textContent=
          '观察感受器产生的信号如何经过传入通路、中继结构和相关中枢网络。';

        dynamic.innerHTML=renderPathway(
          info,
          progress,
          transmitted,
          relayed,
          centralActivation
        );

        stageNote.textContent=
          '感觉通路负责传递和加工信息；感觉最终在相关中枢及协同神经网络中形成，而不是在感受器中形成。';

        renderLabels(mode);
      }else if(mode==='features'){
        title.textContent=
          info.name+'信息的多特征并行加工';

        summary.textContent=
          '把刺激类型、空间位置、强度和时间变化等特征分开处理后再进行综合。';

        dynamic.innerHTML=renderFeatures(
          info,
          modalityValue,
          locationValue,
          intensityValue,
          integration
        );

        stageNote.textContent=
          '中枢不是简单复制传入信号，而是对刺激类型、位置、强度和时间等特征进行并行加工和综合表征。';

        renderLabels(mode);
      }else if(mode==='attention'){
        title.textContent=
          '注意选择、背景干扰与目标信息加工';

        summary.textContent=
          '比较注意聚焦和背景噪声如何改变目标感觉信息的加工优先级和清晰度。';

        dynamic.innerHTML=renderAttention(
          info,
          attention,
          noise,
          gating,
          clarity
        );

        stageNote.textContent=
          '注意可以提高目标信息的加工优先级并抑制部分无关干扰，但注意不会创造或改变外界物理刺激。';

        renderLabels(mode);
      }else{
        title.textContent=
          info.name+'感觉的形成、解释与识别';

        summary.textContent=
          '观察传入信号经过特征加工、网络综合和情境解释后形成感觉与识别结果。';

        dynamic.innerHTML=renderPerception(
          info,
          progress,
          centralActivation,
          clarity,
          recognition,
          context
        );

        stageNote.textContent=recognition
          ?'当前传入、中继、注意和情境条件能够形成较清晰的感觉表征和识别结果。'
          :'当前感觉表征仍较模糊；中枢可能已经接收到信号，但信息尚不足以形成清晰识别。';

        renderLabels(mode);
      }

      var condition=
        '当前传入信号、感觉通路、中继效率、注意和情境线索能够支持较稳定的中枢加工。';

      if(signal<12){
        condition=
          '传入信号很弱，即使通路和中继较完整，中枢获得的有效信息仍然有限。';
      }else if(pathway<35){
        condition=
          '感觉通路完整度较低，传入信息在到达中枢加工网络前已经明显衰减。';
      }else if(relay<35){
        condition=
          '中继整合效率较低，信息筛选、特征保持和向更高层级传递受到限制。';
      }else if(noise>78){
        condition=
          '背景干扰很强，目标感觉信息与竞争输入难以分离，感觉清晰度明显下降。';
      }else if(attention<20){
        condition=
          '注意聚焦水平较低，目标感觉信息获得的加工优先级较低。';
      }else if(context<20){
        condition=
          '情境线索与当前输入匹配度较低，即使形成感觉，也较难快速解释其具体含义。';
      }else if(context>88&&signal<35){
        condition=
          '情境线索很强而传入信号较弱，已有经验可能帮助推测，但也可能增加误判风险。';
      }else if(processTime<18){
        condition=
          '加工过程时间较短，信息仍主要处于传入或早期中继阶段，清晰感觉尚未充分形成。';
      }

      var principle=mode==='pathway'
        ?'感觉信息通常经历感受器转换、传入通路传播、中继与初步整合，再进入相关皮层及协同网络进一步加工。'
        :mode==='features'
          ?'中枢可并行加工刺激类型、位置、强度和时间变化等特征，并把这些特征综合为较完整的感觉表征。'
          :mode==='attention'
            ?'注意提高目标信息的加工优先级，背景干扰降低信噪比；两者影响感觉清晰度，但不改变外界刺激本身。'
            :'感觉是中枢神经系统在传入信息基础上结合注意、经验和情境加工形成的体验，并不是外界刺激的机械复制。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前传入保留指数 '
        +transmitted.toFixed(0)
        +'，中枢激活指数 '
        +centralActivation.toFixed(0)
        +'，感觉清晰度 '
        +clarity.toFixed(0)
        +'。所有数值仅用于课堂比较，不用于神经功能、意识状态或医学判断。';
    }

    for(var i=0;i<channelButtons.length;i++){
      channelButtons[i].onclick=function(){
        channel=this.getAttribute(
          'data-channel'
        );
        update();
      };
    }

    for(var j=0;j<modeButtons.length;j++){
      modeButtons[j].onclick=function(){
        mode=this.getAttribute(
          'data-mode'
        );
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    autoButton.onclick=function(){
      automatic=!automatic;
      update();
      schedule();
    };

    signalInput.oninput=update;
    pathwayInput.oninput=update;
    relayInput.oninput=update;
    attentionInput.oninput=update;
    noiseInput.oninput=update;
    contextInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
