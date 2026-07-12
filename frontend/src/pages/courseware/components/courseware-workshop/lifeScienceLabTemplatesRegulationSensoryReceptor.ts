/**
 * lifeScienceLabTemplatesRegulationSensoryReceptor.ts
 *
 * 平面生命科学实验室：
 * 感受器、适宜刺激与刺激强度编码。
 *
 * 教学目标：
 * 1. 认识机械感受器、光感受器、温度感受器和化学感受器；
 * 2. 理解不同感受器对特定类型刺激最敏感，
 *    这种刺激称为该感受器的适宜刺激；
 * 3. 理解感受器接受刺激后，
 *    首先通过刺激转换形成局部、渐变的感受器电位；
 * 4. 理解感受器电位达到阈值后，
 *    可在传入神经纤维上引发动作电位；
 * 5. 理解单根神经纤维动作电位幅度具有全或无特征；
 * 6. 理解刺激强度通常通过动作电位频率、
 *    参与活动的感受器和传入神经纤维数量等方式编码；
 * 7. 理解刺激增强不能简单解释为单个动作电位幅度持续增大；
 * 8. 观察慢适应感受器和快适应感受器对持续刺激的不同反应；
 * 9. 理解感觉最终在相应中枢形成，
 *    不是在感受器或传入神经纤维中形成。
 *
 * 科学边界：
 * 1. 适宜刺激是某类感受器最容易响应的刺激类型，
 *    并不意味着其他刺激绝对不能引起反应；
 * 2. 足够强的非适宜刺激有时也可能激活感受器，
 *    但阈值通常更高、灵敏度通常更低；
 * 3. 感受器电位属于局部渐变电位，
 *    其大小可随有效刺激强度改变；
 * 4. 动作电位达到阈值后表现为全或无，
 *    单个动作电位幅度不承担连续表示刺激强弱的主要任务；
 * 5. 刺激强度编码可涉及冲动频率、感受器募集、
 *    神经纤维募集和神经群体活动模式；
 * 6. 不同感觉系统的真实编码方式存在差异，
 *    本模板只展示通用教学模型；
 * 7. 快适应感受器更突出刺激开始、结束或快速变化，
 *    慢适应感受器可在持续刺激期间保持相对稳定的放电；
 * 8. “适应”不等同于感受器完全停止工作，
 *    也不等同于个体主观感觉必然消失；
 * 9. 感受器负责接受刺激和转换信息，
 *    感觉最终在相应中枢神经系统区域形成；
 * 10. 图中的阈值、频率、募集比例、感受器电位和适应速度，
 *     均为相对教学指标；
 * 11. 本模板只用于生物学课堂教学，
 *     不用于感觉功能、神经功能或疾病判断。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部图片、脚本、字体、地图或CDN；
 * 3. 所有DOM查询均限定在rootId内部；
 * 4. 支持同一课件页面插入多个独立实例；
 * 5. 使用生命科学统一.bl-*布局协议；
 * 6. 支持参数滑杆、四种观察方式、四类感受器切换、
 *    自动推进、结构标注开关、动态图示和即时教学结论；
 * 7. 本文件只导出独立模板数组；
 * 8. 聚合入口将在后续C1批次统一接入。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

/**
 * 从模板参数中安全读取数值。
 *
 * 外部调用如果缺少参数、参数类型错误或出现非有限数值，
 * 则使用模板内定义的教学默认值，避免生成无效HTML。
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
 * 从模板参数中安全读取布尔值。
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
 * 把模板参数整理为适合写入HTML属性的简洁数字。
 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/**
 * 模板独立样式。
 *
 * 所有选择器都带rootId前缀，
 * 避免同一课件中多个生命科学模板互相污染。
 */
function sensoryReceptorStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#ECFEFF);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FDFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0284C7}'
    + '#' + rootId + ' .sr-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .sr-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .sr-button{min-height:30px;padding:3px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .sr-button.active{border-color:#0284C7;background:#E0F2FE;box-shadow:0 3px 9px rgba(2,132,199,.14)}'
    + '#' + rootId + ' .sr-receptor-button{min-height:29px;padding:3px;border:1px solid #A5F3FC;border-radius:8px;background:#fff;color:#155E75;font-size:9.8px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .sr-receptor-button.active{border-color:#0891B2;background:#CFFAFE;box-shadow:0 3px 9px rgba(8,145,178,.14)}'
    + '#' + rootId + ' .sr-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .sr-toggle.off{background:#64748B}'
    + '#' + rootId + ' .sr-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .sr-card{padding:6px 3px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .sr-card b{display:block;font-size:13px;color:#0369A1}'
    + '#' + rootId + ' .sr-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .sr-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--sr-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .sr-pulse{animation:' + rootId + '-pulse 1.45s ease-in-out infinite}'
    + '#' + rootId + ' .sr-blink{animation:' + rootId + '-blink 1.1s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.32}50%{opacity:1}}'
    + '@keyframes ' + rootId + '-blink{0%,100%{opacity:.28}50%{opacity:.94}}'
    + '</style>'
}

/**
 * 避免源码中直接书写闭合script标签，
 * 防止模板字符串在HTML解析阶段提前结束。
 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SENSORY_RECEPTOR:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-sensory-receptor-coding',
    group: '🧠 稳态与调节',
    name: '感受器、适宜刺激与刺激强度编码',
    emoji: '👁️',
    desc: '切换机械、光、温度和化学感受器，调节刺激、阈值、灵敏度、感受器密度、适应速度和过程时间，观察刺激转换与强度编码',
    params: [
      {
        key: 'stimulusIntensity',
        label: '刺激强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
        hint: '表示对当前感受器产生作用的相对刺激强度',
      },
      {
        key: 'thresholdLevel',
        label: '感受器兴奋阈值',
        type: 'number',
        min: 15,
        max: 80,
        step: 1,
        defaultValue: 46,
        hint: '有效刺激达到该相对阈值后可引发传入神经冲动',
      },
      {
        key: 'receptorSensitivity',
        label: '感受器相对灵敏度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 84,
        hint: '适宜刺激与当前感受器匹配越好，有效灵敏度通常越高',
      },
      {
        key: 'receptorDensity',
        label: '感受器相对密度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 66,
        hint: '用于观察刺激增强时感受器和神经纤维募集的教学模型',
      },
      {
        key: 'adaptationRate',
        label: '适应速度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
        hint: '数值越高，持续刺激期间的反应衰减越快',
      },
      {
        key: 'processTime',
        label: '观察过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 36,
        hint: '控制刺激转换和适应过程的课堂演示进度',
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const stimulusIntensity = num(
        params,
        'stimulusIntensity',
        68,
      )
      const thresholdLevel = num(
        params,
        'thresholdLevel',
        46,
      )
      const receptorSensitivity = num(
        params,
        'receptorSensitivity',
        84,
      )
      const receptorDensity = num(
        params,
        'receptorDensity',
        66,
      )
      const adaptationRate = num(
        params,
        'adaptationRate',
        42,
      )
      const processTime = num(
        params,
        'processTime',
        36,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${sensoryReceptorStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">👁️ 感受器、适宜刺激与刺激强度编码</div>
    <div class="bl-note">感受器负责刺激转换，感觉最终在相应中枢形成</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>刺激强度</span>
          <span class="bl-value" data-stimulus-value></span>
        </div>
        <input
          data-stimulus
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(stimulusIntensity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>感受器兴奋阈值</span>
          <span class="bl-value" data-threshold-value></span>
        </div>
        <input
          data-threshold
          type="range"
          min="15"
          max="80"
          step="1"
          value="${n(thresholdLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>感受器相对灵敏度</span>
          <span class="bl-value" data-sensitivity-value></span>
        </div>
        <input
          data-sensitivity
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(receptorSensitivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>感受器相对密度</span>
          <span class="bl-value" data-density-value></span>
        </div>
        <input
          data-density
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(receptorDensity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>适应速度</span>
          <span class="bl-value" data-adaptation-value></span>
        </div>
        <input
          data-adaptation
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(adaptationRate)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>观察过程时间</span>
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

      <div class="sr-subtitle">选择感受器类型</div>

      <div class="sr-buttons">
        <button
          type="button"
          class="sr-receptor-button active"
          data-receptor="mechanical"
        >机械感受器</button>

        <button
          type="button"
          class="sr-receptor-button"
          data-receptor="light"
        >光感受器</button>

        <button
          type="button"
          class="sr-receptor-button"
          data-receptor="temperature"
        >温度感受器</button>

        <button
          type="button"
          class="sr-receptor-button"
          data-receptor="chemical"
        >化学感受器</button>
      </div>

      <div class="sr-subtitle">观察方式</div>

      <div class="sr-buttons">
        <button
          type="button"
          class="sr-button active"
          data-mode="receptors"
        >感受器与适宜刺激</button>

        <button
          type="button"
          class="sr-button"
          data-mode="transduction"
        >刺激转换与阈值</button>

        <button
          type="button"
          class="sr-button"
          data-mode="coding"
        >频率与募集编码</button>

        <button
          type="button"
          class="sr-button"
          data-mode="adaptation"
        >快慢适应比较</button>
      </div>

      <button
        type="button"
        class="sr-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="sr-toggle"
        data-auto
      >过程推进：运行中</button>

      <div class="sr-status">
        <div class="sr-card">
          <b data-potential></b>
          <span>感受器电位</span>
        </div>

        <div class="sr-card">
          <b data-frequency></b>
          <span>冲动频率</span>
        </div>

        <div class="sr-card">
          <b data-recruitment></b>
          <span>募集比例</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="感受器、适宜刺激和刺激强度编码互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-blue"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker
            id="${rootId}-arrow-cyan"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#0891B2"/>
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

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#0C4A6E"
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
          fill="#075985"
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
            fill="#ECFEFF"
            stroke="#A5F3FC"
            stroke-width="2"
          />

          <text
            x="110"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#155E75"
          >关键边界</text>

          <text
            x="110"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#164E63"
          >动作电位幅度不连续表示刺激强弱</text>

          <text
            x="110"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#164E63"
          >感觉最终在相应中枢形成</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#075985"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var stimulusInput=root.querySelector(
      '[data-stimulus]'
    );
    var thresholdInput=root.querySelector(
      '[data-threshold]'
    );
    var sensitivityInput=root.querySelector(
      '[data-sensitivity]'
    );
    var densityInput=root.querySelector(
      '[data-density]'
    );
    var adaptationInput=root.querySelector(
      '[data-adaptation]'
    );
    var timeInput=root.querySelector(
      '[data-time]'
    );

    var stimulusValue=root.querySelector(
      '[data-stimulus-value]'
    );
    var thresholdValue=root.querySelector(
      '[data-threshold-value]'
    );
    var sensitivityValue=root.querySelector(
      '[data-sensitivity-value]'
    );
    var densityValue=root.querySelector(
      '[data-density-value]'
    );
    var adaptationValue=root.querySelector(
      '[data-adaptation-value]'
    );
    var timeValue=root.querySelector(
      '[data-time-value]'
    );

    var receptorButtons=root.querySelectorAll(
      '[data-receptor]'
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

    var potentialText=root.querySelector(
      '[data-potential]'
    );
    var frequencyText=root.querySelector(
      '[data-frequency]'
    );
    var recruitmentText=root.querySelector(
      '[data-recruitment]'
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

    var mode='receptors';
    var receptorType='mechanical';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    var receptorInformation={
      mechanical:{
        name:'机械感受器',
        shortName:'机械',
        stimulus:'压力、牵拉、振动或位移',
        example:'皮肤触压和振动',
        color:'#0284C7',
        pale:'#E0F2FE',
        stroke:'#0369A1',
        icon:'pressure',
        match:1.00,
        description:'机械形变可改变感受器膜上的机械敏感通道状态。'
      },
      light:{
        name:'光感受器',
        shortName:'光',
        stimulus:'一定波长和强度的光',
        example:'视网膜接受光刺激',
        color:'#7C3AED',
        pale:'#EDE9FE',
        stroke:'#5B21B6',
        icon:'light',
        match:.96,
        description:'光感受器通过光敏分子和细胞内信号过程完成光信号转换。'
      },
      temperature:{
        name:'温度感受器',
        shortName:'温度',
        stimulus:'温度升高、降低或温度变化',
        example:'皮肤温度变化',
        color:'#EA580C',
        pale:'#FFEDD5',
        stroke:'#C2410C',
        icon:'temperature',
        match:.92,
        description:'温度变化可影响温度敏感离子通道的开放状态。'
      },
      chemical:{
        name:'化学感受器',
        shortName:'化学',
        stimulus:'特定化学物质及其浓度变化',
        example:'味觉、嗅觉和血液化学变化',
        color:'#16A34A',
        pale:'#DCFCE7',
        stroke:'#15803D',
        icon:'chemical',
        match:.94,
        description:'化学物质与受体结合后可直接或间接改变膜电活动。'
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

    function stimulusSymbol(
      type,
      x,
      y,
      scale,
      opacity
    ){
      if(type==='mechanical'){
        return ''
          +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
          +'<path d="M-34 -31 V-6 M-48 -17 L-34 -3 L-20 -17" fill="none" stroke="#0284C7" stroke-width="7" stroke-linecap="round" stroke-linejoin="round"/>'
          +'<rect x="-55" y="5" width="42" height="22" rx="7" fill="#BAE6FD" stroke="#0284C7" stroke-width="4"/>'
          +'<path d="M6 -31 V-6 M-8 -17 L6 -3 L20 -17" fill="none" stroke="#0284C7" stroke-width="7" stroke-linecap="round" stroke-linejoin="round"/>'
          +'<rect x="-13" y="5" width="42" height="22" rx="7" fill="#BAE6FD" stroke="#0284C7" stroke-width="4"/>'
          +'</g>';
      }

      if(type==='light'){
        return ''
          +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
          +'<circle r="21" fill="#FDE68A" stroke="#D97706" stroke-width="5"/>'
          +'<path d="M0 -44 V-30 M0 30 V44 M-44 0 H-30 M30 0 H44 M-31 -31 L-21 -21 M21 21 L31 31 M31 -31 L21 -21 M-21 21 L-31 31" stroke="#F59E0B" stroke-width="6" stroke-linecap="round"/>'
          +'</g>';
      }

      if(type==='temperature'){
        return ''
          +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
          +'<rect x="-9" y="-41" width="18" height="61" rx="9" fill="#FFFFFF" stroke="#EA580C" stroke-width="5"/>'
          +'<circle cx="0" cy="28" r="20" fill="#FB923C" stroke="#C2410C" stroke-width="5"/>'
          +'<rect x="-5" y="-24" width="10" height="52" rx="5" fill="#F97316"/>'
          +'</g>';
      }

      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle cx="-23" cy="-9" r="14" fill="#86EFAC" stroke="#16A34A" stroke-width="4"/>'
        +'<circle cx="12" cy="-23" r="12" fill="#BBF7D0" stroke="#16A34A" stroke-width="4"/>'
        +'<circle cx="25" cy="12" r="15" fill="#4ADE80" stroke="#15803D" stroke-width="4"/>'
        +'<line x1="-10" y1="-14" x2="2" y2="-20" stroke="#15803D" stroke-width="5"/>'
        +'<line x1="18" y1="-11" x2="22" y2="-1" stroke="#15803D" stroke-width="5"/>'
        +'</g>';
    }

    function receptorShape(
      type,
      x,
      y,
      scale,
      active
    ){
      var info=receptorInformation[type];
      var glow=active
        ?'<circle class="sr-pulse" cx="0" cy="0" r="54" fill="'+info.color+'" opacity=".28"/>'
        :'';

      var body='';

      if(type==='mechanical'){
        body=''
          +'<ellipse cx="0" cy="0" rx="34" ry="47" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="6"/>'
          +'<path d="M-18 -28 Q0 -8 18 -28 M-21 -5 Q0 14 21 -5 M-17 19 Q0 37 17 19" fill="none" stroke="'+info.color+'" stroke-width="6" stroke-linecap="round"/>';
      }else if(type==='light'){
        body=''
          +'<path d="M-38 -15 Q0 -49 38 -15 Q0 21 -38 -15Z" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="6"/>'
          +'<circle cx="0" cy="-15" r="16" fill="'+info.color+'" stroke="#FFFFFF" stroke-width="4"/>'
          +'<path d="M0 5 V48" stroke="'+info.stroke+'" stroke-width="11" stroke-linecap="round"/>';
      }else if(type==='temperature'){
        body=''
          +'<circle cx="0" cy="-8" r="35" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="6"/>'
          +'<path d="M-18 -5 Q0 -27 18 -5 Q0 19 -18 -5Z" fill="'+info.color+'" opacity=".78"/>'
          +'<path d="M0 27 V52" stroke="'+info.stroke+'" stroke-width="11" stroke-linecap="round"/>';
      }else{
        body=''
          +'<path d="M-38 -26 Q0 -48 38 -26 V20 Q0 45 -38 20Z" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="6"/>'
          +'<path d="M-24 -14 L-8 2 L7 -16 L24 4" fill="none" stroke="'+info.color+'" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/>'
          +'<path d="M0 31 V55" stroke="'+info.stroke+'" stroke-width="11" stroke-linecap="round"/>';
      }

      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')">'
        +glow
        +body
        +'</g>';
    }

    function buildReceptorPotentialPath(
      amplitude,
      adaptation,
      startX,
      baseY,
      width
    ){
      var path='';

      for(var i=0;i<=100;i+=2){
        var t=i/100;
        var rise=1-Math.exp(-t*13);
        var decay=Math.exp(
          -Math.max(0,t-.18)
          *(adaptation/34)
        );
        var response=amplitude
          *rise
          *(
            .28+.72*decay
          );
        var x=startX+width*t;
        var y=baseY-response*.95;

        path+=(i===0?'M':' L')
          +x.toFixed(1)
          +' '
          +y.toFixed(1);
      }

      return path;
    }

    function spikeTrain(
      startX,
      baseY,
      width,
      frequency,
      color,
      amplitude,
      opacity
    ){
      if(frequency<=0){
        return ''
          +'<line x1="'+startX+'" y1="'+baseY+'" x2="'+(startX+width)+'" y2="'+baseY+'" stroke="#94A3B8" stroke-width="3" opacity="'+opacity+'"/>';
      }

      var count=Math.max(
        1,
        Math.floor(
          2+frequency/10
        )
      );
      var spacing=width/count;
      var path='M'+startX+' '+baseY;

      for(var i=0;i<count;i++){
        var x=startX
          +i*spacing
          +spacing*.38;

        path+=' L'
          +(x-spacing*.16).toFixed(1)
          +' '
          +baseY;

        path+=' L'
          +x.toFixed(1)
          +' '
          +(baseY-amplitude).toFixed(1);

        path+=' L'
          +(x+spacing*.15).toFixed(1)
          +' '
          +baseY;
      }

      path+=' L'
        +(startX+width).toFixed(1)
        +' '
        +baseY;

      return ''
        +'<path d="'+path+'" fill="none" stroke="'+color+'" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" opacity="'+opacity+'"/>';
    }

    function responseCurve(
      type,
      adaptation,
      startX,
      baseY,
      width,
      height
    ){
      var path='';

      for(var i=0;i<=100;i+=2){
        var t=i/100;
        var onset=1-Math.exp(-t*18);
        var value=0;

        if(type==='slow'){
          value=onset
            *(
              .72
              +.28*Math.exp(
                -t*(1+adaptation/130)
              )
            );
        }else{
          value=onset
            *Math.exp(
              -t*(3.2+adaptation/22)
            );
        }

        var x=startX+width*t;
        var y=baseY-height*value;

        path+=(i===0?'M':' L')
          +x.toFixed(1)
          +' '
          +y.toFixed(1);
      }

      return path;
    }

    function selectedResponseCurve(
      adaptation,
      startX,
      baseY,
      width,
      height
    ){
      var path='';

      for(var i=0;i<=100;i+=2){
        var t=i/100;
        var onset=1-Math.exp(-t*18);
        var decay=Math.exp(
          -t*(.6+adaptation/23)
        );
        var minimum=clamp(
          .76-adaptation/145,
          .08,
          .76
        );
        var value=onset
          *(
            minimum
            +(1-minimum)*decay
          );
        var x=startX+width*t;
        var y=baseY-height*value;

        path+=(i===0?'M':' L')
          +x.toFixed(1)
          +' '
          +y.toFixed(1);
      }

      return path;
    }

    function renderReceptors(
      selectedType,
      effectiveStimulus
    ){
      var types=[
        'mechanical',
        'light',
        'temperature',
        'chemical'
      ];
      var positions=[
        [120,199],
        [288,199],
        [456,199],
        [624,199]
      ];
      var html='';

      for(var i=0;i<types.length;i++){
        var type=types[i];
        var info=receptorInformation[type];
        var active=type===selectedType;
        var x=positions[i][0];
        var y=positions[i][1];

        html+=''
          +'<g filter="url(#${rootId}-shadow)">'
          +'<rect x="'+(x-73)+'" y="105" width="146" height="211" rx="22" fill="'+(active?info.pale:'#F8FAFC')+'" stroke="'+(active?info.stroke:'#CBD5E1')+'" stroke-width="'+(active?5:3)+'"/>'
          +'</g>'
          +stimulusSymbol(
            type,
            x,
            143,
            .56,
            active?.95:.52
          )
          +receptorShape(
            type,
            x,
            y+27,
            .66,
            active
          )
          +'<text x="'+x+'" y="281" text-anchor="middle" font-size="13" font-weight="900" fill="'+(active?info.stroke:'#475569')+'">'+info.name+'</text>'
          +'<text x="'+x+'" y="301" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">'+info.example+'</text>';
      }

      var selected=receptorInformation[selectedType];

      return html
        +'<g transform="translate(85 329)">'
        +'<rect width="590" height="45" rx="16" fill="#ECFEFF" stroke="#A5F3FC" stroke-width="2"/>'
        +'<text x="295" y="18" text-anchor="middle" font-size="11.5" font-weight="900" fill="#155E75">当前选择：'+selected.name+'　适宜刺激：'+selected.stimulus+'</text>'
        +'<text x="295" y="35" text-anchor="middle" font-size="10" font-weight="800" fill="#475569">有效刺激指数 '+effectiveStimulus.toFixed(0)+'；适宜刺激表示最容易引起反应的刺激类型，不是绝对排他关系。</text>'
        +'</g>';
    }

    function renderTransduction(
      info,
      progress,
      receptorPotential,
      excited,
      frequency,
      adaptation
    ){
      var processX=321+progress*223;
      var thresholdY=246-55*.95;
      var potentialY=246-receptorPotential*.95;
      var stage='刺激到达';

      if(progress>=.20&&progress<.44){
        stage='感受器膜发生刺激转换';
      }else if(progress>=.44&&progress<.67){
        stage='形成感受器电位';
      }else if(progress>=.67&&progress<.84){
        stage=excited
          ?'达到阈值'
          :'未达到阈值';
      }else if(progress>=.84){
        stage=excited
          ?'传入神经产生动作电位'
          :'只保留局部渐变电位';
      }

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<g transform="translate(67 111)">'
        +'<rect width="188" height="226" rx="21" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="4"/>'
        +'<text x="94" y="28" text-anchor="middle" font-size="13" font-weight="900" fill="'+info.stroke+'">'+info.name+'</text>'
        +stimulusSymbol(receptorType,94,79,.72,.92)
        +receptorShape(receptorType,94,151,.75,excited)
        +'<path class="sr-flow" d="M94 184 V207" fill="none" stroke="'+info.color+'" stroke-width="5" marker-end="url(#${rootId}-arrow-cyan)"/>'
        +'<text x="94" y="219" text-anchor="middle" font-size="10" font-weight="900" fill="#475569">把刺激转换为膜电变化</text>'
        +'</g>'
        +'<g transform="translate(282 111)">'
        +'<rect width="291" height="226" rx="21" fill="#FFFFFF" stroke="#BAE6FD" stroke-width="4"/>'
        +'<text x="145" y="28" text-anchor="middle" font-size="13" font-weight="900" fill="#075985">感受器电位：局部渐变电位</text>'
        +'<line x1="39" y1="165" x2="262" y2="165" stroke="#94A3B8" stroke-width="2"/>'
        +'<line x1="39" y1="54" x2="39" y2="185" stroke="#94A3B8" stroke-width="2"/>'
        +'<line x1="39" y1="'+(thresholdY-111).toFixed(1)+'" x2="262" y2="'+(thresholdY-111).toFixed(1)+'" stroke="#F59E0B" stroke-width="2.5" stroke-dasharray="7 6"/>'
        +'<text x="5" y="'+(thresholdY-116).toFixed(1)+'" font-size="9.5" font-weight="900" fill="#B45309">阈值</text>'
        +'<path d="'+buildReceptorPotentialPath(receptorPotential,adaptation,321,246,223)+'" transform="translate(-282 -111)" fill="none" stroke="'+info.color+'" stroke-width="5" stroke-linecap="round"/>'
        +'<line x1="'+(processX-282).toFixed(1)+'" y1="51" x2="'+(processX-282).toFixed(1)+'" y2="186" stroke="#0284C7" stroke-width="2" stroke-dasharray="5 5"/>'
        +'<circle cx="'+(processX-282).toFixed(1)+'" cy="'+(potentialY-111).toFixed(1)+'" r="8" fill="#0284C7" stroke="#FFFFFF" stroke-width="3"/>'
        +'<text x="145" y="205" text-anchor="middle" font-size="10.5" font-weight="900" fill="#475569">当前感受器电位指数 '+receptorPotential.toFixed(0)+'</text>'
        +'</g>'
        +'<g transform="translate(598 111)">'
        +'<rect width="107" height="226" rx="21" fill="#ECFEFF" stroke="#A5F3FC" stroke-width="4"/>'
        +'<text x="53" y="27" text-anchor="middle" font-size="12" font-weight="900" fill="#155E75">传入神经</text>'
        +'<path d="M53 47 V184" stroke="#67E8F9" stroke-width="15" stroke-linecap="round"/>'
        +spikeTrain(16,184,74,frequency,'#DC2626',31,excited?1:.32)
        +'<text x="53" y="210" text-anchor="middle" font-size="9.5" font-weight="900" fill="'+(excited?'#B91C1C':'#64748B')+'">'+(excited?'全或无动作电位':'未产生动作电位')+'</text>'
        +'</g>'
        +'<g transform="translate(151 340)">'
        +'<rect width="458" height="24" rx="12" fill="#E0F2FE" stroke="#BAE6FD" stroke-width="2"/>'
        +'<text x="229" y="17" text-anchor="middle" font-size="10.5" font-weight="900" fill="#075985">当前阶段：'+stage+'</text>'
        +'</g>';
    }

    function renderCoding(
      info,
      frequency,
      recruitment,
      excited,
      receptorPotential,
      stimulus
    ){
      var lowFrequency=excited
        ?clamp(frequency*.38,4,36)
        :0;
      var mediumFrequency=excited
        ?clamp(frequency*.68,6,68)
        :0;
      var recruitedCount=excited
        ?Math.max(
          1,
          Math.ceil(recruitment/25)
        )
        :0;
      var amplitude=44;

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<g transform="translate(48 109)">'
        +'<rect width="318" height="231" rx="21" fill="#FFFFFF" stroke="#BAE6FD" stroke-width="4"/>'
        +'<text x="159" y="28" text-anchor="middle" font-size="13" font-weight="900" fill="#075985">刺激增强：频率增加，单个峰幅度基本不变</text>'
        +'<text x="18" y="69" font-size="10.5" font-weight="900" fill="#64748B">较弱有效刺激</text>'
        +spikeTrain(102,85,191,lowFrequency,'#0284C7',amplitude,1)
        +'<text x="18" y="132" font-size="10.5" font-weight="900" fill="#64748B">中等有效刺激</text>'
        +spikeTrain(102,148,191,mediumFrequency,'#0284C7',amplitude,1)
        +'<text x="18" y="195" font-size="10.5" font-weight="900" fill="#64748B">当前有效刺激</text>'
        +spikeTrain(102,211,191,frequency,'#DC2626',amplitude,excited?1:.35)
        +'<line x1="91" y1="45" x2="91" y2="218" stroke="#CBD5E1" stroke-width="2"/>'
        +'</g>'
        +'<g transform="translate(391 109)">'
        +'<rect width="314" height="231" rx="21" fill="#ECFEFF" stroke="#A5F3FC" stroke-width="4"/>'
        +'<text x="157" y="28" text-anchor="middle" font-size="13" font-weight="900" fill="#155E75">刺激增强：更多感受器和神经纤维参与</text>'
        +'<g transform="translate(28 47)">'
        +receptorShape(receptorType,40,40,.46,recruitedCount>=1)
        +receptorShape(receptorType,105,40,.46,recruitedCount>=2)
        +receptorShape(receptorType,170,40,.46,recruitedCount>=3)
        +receptorShape(receptorType,235,40,.46,recruitedCount>=4)
        +'</g>'
        +'<path d="M67 138 C107 155 118 171 151 183" fill="none" stroke="'+(recruitedCount>=1?info.color:'#CBD5E1')+'" stroke-width="6" stroke-linecap="round"/>'
        +'<path d="M132 138 C152 155 163 171 169 183" fill="none" stroke="'+(recruitedCount>=2?info.color:'#CBD5E1')+'" stroke-width="6" stroke-linecap="round"/>'
        +'<path d="M197 138 C187 157 185 170 187 183" fill="none" stroke="'+(recruitedCount>=3?info.color:'#CBD5E1')+'" stroke-width="6" stroke-linecap="round"/>'
        +'<path d="M262 138 C224 158 211 171 205 183" fill="none" stroke="'+(recruitedCount>=4?info.color:'#CBD5E1')+'" stroke-width="6" stroke-linecap="round"/>'
        +'<path class="sr-flow" d="M178 183 H278" fill="none" stroke="'+(excited?'#DC2626':'#94A3B8')+'" stroke-width="7" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="157" y="216" text-anchor="middle" font-size="10.5" font-weight="900" fill="#475569">当前募集约 '+recruitment.toFixed(0)+'%，示意参与单位 '+recruitedCount+' / 4</text>'
        +'</g>'
        +'<g transform="translate(83 344)">'
        +'<rect width="594" height="22" rx="11" fill="#E0F2FE" stroke="#BAE6FD" stroke-width="2"/>'
        +'<text x="297" y="15" text-anchor="middle" font-size="10" font-weight="900" fill="#075985">刺激 '+stimulus.toFixed(0)+'%，感受器电位 '+receptorPotential.toFixed(0)+'，动作电位峰幅度示意保持一致，频率和募集水平发生变化。</text>'
        +'</g>';
    }

    function renderAdaptation(
      info,
      progress,
      adaptation,
      frequency,
      initialFrequency,
      remainingResponse
    ){
      var markerX=87+progress*538;
      var selectedPath=selectedResponseCurve(
        adaptation,
        87,
        281,
        538,
        122
      );

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="380" y="122" text-anchor="middle" font-size="14" font-weight="900" fill="#075985">持续刺激期间，快适应与慢适应感受器的放电变化</text>'
        +'<rect x="87" y="140" width="538" height="20" rx="10" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="2"/>'
        +'<text x="356" y="154" text-anchor="middle" font-size="10" font-weight="900" fill="'+info.stroke+'">持续施加当前适宜刺激</text>'
        +'<line x1="87" y1="281" x2="625" y2="281" stroke="#64748B" stroke-width="3"/>'
        +'<line x1="87" y1="165" x2="87" y2="281" stroke="#64748B" stroke-width="3"/>'
        +'<path d="'+responseCurve('slow',adaptation,87,281,538,112)+'" fill="none" stroke="#16A34A" stroke-width="4" stroke-linecap="round"/>'
        +'<path d="'+responseCurve('fast',adaptation,87,281,538,112)+'" fill="none" stroke="#EA580C" stroke-width="4" stroke-linecap="round"/>'
        +'<path d="'+selectedPath+'" fill="none" stroke="'+info.color+'" stroke-width="6" stroke-linecap="round" opacity=".88"/>'
        +'<line x1="'+markerX.toFixed(1)+'" y1="163" x2="'+markerX.toFixed(1)+'" y2="281" stroke="#0284C7" stroke-width="2" stroke-dasharray="5 5"/>'
        +'<circle cx="'+markerX.toFixed(1)+'" cy="'+(281-122*remainingResponse/100).toFixed(1)+'" r="8" fill="#0284C7" stroke="#FFFFFF" stroke-width="3"/>'
        +'<g transform="translate(103 302)">'
        +'<line x1="0" y1="0" x2="33" y2="0" stroke="#16A34A" stroke-width="5"/>'
        +'<text x="42" y="4" font-size="10" font-weight="900" fill="#166534">慢适应：持续报告刺激</text>'
        +'<line x1="200" y1="0" x2="233" y2="0" stroke="#EA580C" stroke-width="5"/>'
        +'<text x="242" y="4" font-size="10" font-weight="900" fill="#9A3412">快适应：突出开始、结束和变化</text>'
        +'</g>'
        +'<g transform="translate(130 330)">'
        +'<rect width="500" height="35" rx="16" fill="#ECFEFF" stroke="#A5F3FC" stroke-width="2"/>'
        +'<text x="250" y="15" text-anchor="middle" font-size="10.5" font-weight="900" fill="#155E75">初始频率 '+initialFrequency.toFixed(0)+'，当前频率 '+frequency.toFixed(0)+'，持续反应保留 '+remainingResponse.toFixed(0)+'%</text>'
        +'<text x="250" y="29" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">适应表示持续刺激期间反应模式改变，不等于感受器绝对失去功能。</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='receptors'){
        labels.innerHTML=''
          +'<path d="M120 154 L120 82" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="54" y="77" font-size="12.5" font-weight="900" fill="#0369A1">适宜刺激输入</text>'
          +'<path d="M288 229 L288 330" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="231" y="346" font-size="12.5" font-weight="900" fill="#5B21B6">感受器结构示意</text>';
        return;
      }

      if(modeName==='transduction'){
        labels.innerHTML=''
          +'<path d="M161 244 L118 281" stroke="#0891B2" stroke-width="2.5"/>'
          +'<text x="43" y="296" font-size="12.5" font-weight="900" fill="#155E75">刺激转换</text>'
          +'<path d="M432 177 L467 91" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="474" y="86" font-size="12.5" font-weight="900" fill="#0369A1">渐变的感受器电位</text>'
          +'<path d="M651 169 L699 103" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="625" y="96" font-size="12.5" font-weight="900" fill="#B91C1C">全或无动作电位</text>';
        return;
      }

      if(modeName==='coding'){
        labels.innerHTML=''
          +'<path d="M248 197 L290 154" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="297" y="149" font-size="12.5" font-weight="900" fill="#B91C1C">峰幅度保持一致</text>'
          +'<path d="M510 198 L544 151" stroke="#0891B2" stroke-width="2.5"/>'
          +'<text x="551" y="146" font-size="12.5" font-weight="900" fill="#155E75">参与单位募集</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M214 209 L243 172" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="250" y="167" font-size="12.5" font-weight="900" fill="#166534">慢适应反应</text>'
        +'<path d="M388 247 L425 199" stroke="#EA580C" stroke-width="2.5"/>'
        +'<text x="432" y="194" font-size="12.5" font-weight="900" fill="#9A3412">快适应反应</text>'
        +'<path d="M563 221 L610 178" stroke="#0284C7" stroke-width="2.5"/>'
        +'<text x="617" y="173" font-size="12.5" font-weight="900" fill="#0369A1">当前过程时刻</text>';
    }

    function update(){
      var stimulus=Number(
        stimulusInput.value
      );
      var threshold=Number(
        thresholdInput.value
      );
      var sensitivity=Number(
        sensitivityInput.value
      );
      var density=Number(
        densityInput.value
      );
      var adaptation=Number(
        adaptationInput.value
      );
      var processTime=Number(
        timeInput.value
      );

      stimulusValue.textContent=
        stimulus.toFixed(0)+'%';
      thresholdValue.textContent=
        threshold.toFixed(0)+'%';
      sensitivityValue.textContent=
        sensitivity.toFixed(0)+'%';
      densityValue.textContent=
        density.toFixed(0)+'%';
      adaptationValue.textContent=
        adaptation.toFixed(0)+'%';
      timeValue.textContent=
        processTime.toFixed(0)+'%';

      for(var i=0;i<receptorButtons.length;i++){
        receptorButtons[i].classList.toggle(
          'active',
          receptorButtons[i].getAttribute(
            'data-receptor'
          )===receptorType
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
        ?'过程推进：运行中'
        :'过程推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var info=receptorInformation[
        receptorType
      ];
      var progress=processTime/100;

      var effectiveStimulus=stimulus
        *sensitivity/100
        *info.match;

      effectiveStimulus=clamp(
        effectiveStimulus,
        0,
        100
      );

      var localBelowThreshold=
        effectiveStimulus
        /Math.max(1,threshold)
        *48;

      var localAboveThreshold=
        Math.max(
          0,
          effectiveStimulus-threshold
        )
        *1.15;

      var receptorPotential=clamp(
        localBelowThreshold
        +localAboveThreshold,
        0,
        100
      );

      var excited=
        effectiveStimulus>=threshold;

      var initialFrequency=excited
        ?clamp(
          10
          +(effectiveStimulus-threshold)*1.30,
          8,
          100
        )
        :0;

      var remainingResponse=clamp(
        100
        -adaptation
        *progress
        *.82,
        8,
        100
      );

      var frequency=excited
        ?initialFrequency
          *remainingResponse/100
        :0;

      frequency=clamp(
        frequency,
        0,
        100
      );

      var recruitment=excited
        ?clamp(
          12
          +(effectiveStimulus-threshold)*1.18
          +density*.38,
          0,
          100
        )
        :0;

      potentialText.textContent=
        receptorPotential.toFixed(0);
      frequencyText.textContent=
        frequency.toFixed(0);
      recruitmentText.textContent=
        recruitment.toFixed(0)+'%';

      root.style.setProperty(
        '--sr-speed',
        clamp(
          2.45-frequency/72,
          .58,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='receptors'){
        title.textContent=
          '不同感受器及其适宜刺激';

        summary.textContent=
          '比较机械、光、温度和化学感受器最容易响应的刺激类型。';

        dynamic.innerHTML=renderReceptors(
          receptorType,
          effectiveStimulus
        );

        stageNote.textContent=
          info.name
          +'对“'
          +info.stimulus
          +'”最敏感；适宜刺激表示最容易引起反应，不表示绝对只能响应这一类刺激。';

        renderLabels(mode);
      }else if(mode==='transduction'){
        title.textContent=
          '刺激转换、感受器电位与动作电位阈值';

        summary.textContent=
          '观察刺激如何形成渐变的感受器电位，并在达到阈值后触发传入神经动作电位。';

        dynamic.innerHTML=renderTransduction(
          info,
          progress,
          receptorPotential,
          excited,
          frequency,
          adaptation
        );

        stageNote.textContent=excited
          ?'当前有效刺激达到阈值：感受器电位可触发传入神经纤维产生全或无动作电位。'
          :'当前有效刺激未达到阈值：仍可存在局部感受器电位，但未触发可传播的动作电位。';

        renderLabels(mode);
      }else if(mode==='coding'){
        title.textContent=
          '刺激强度的频率编码与募集编码';

        summary.textContent=
          '比较刺激增强后动作电位频率和参与活动单位数量的变化。';

        dynamic.innerHTML=renderCoding(
          info,
          frequency,
          recruitment,
          excited,
          receptorPotential,
          stimulus
        );

        stageNote.textContent=excited
          ?'刺激增强通常表现为冲动频率提高、更多感受器或传入纤维参与，而不是单个动作电位峰幅度持续增大。'
          :'当前有效刺激未达到阈值，因此没有形成动作电位频率编码和明显募集。';

        renderLabels(mode);
      }else{
        title.textContent=
          '持续刺激下的快适应与慢适应';

        summary.textContent=
          '比较持续刺激期间不同感受器反应随时间衰减的速度和保留程度。';

        dynamic.innerHTML=renderAdaptation(
          info,
          progress,
          adaptation,
          frequency,
          initialFrequency,
          remainingResponse
        );

        stageNote.textContent=adaptation>=65
          ?'当前适应速度较快，持续刺激期间放电明显下降，更突出刺激开始、结束或快速变化。'
          :'当前适应速度较慢，持续刺激期间仍保留较多反应，可继续报告刺激状态。';

        renderLabels(mode);
      }

      var condition=
        '当前刺激类型与所选感受器相匹配，灵敏度、阈值和感受器密度能够形成较清晰的刺激编码。';

      if(stimulus<10){
        condition=
          '刺激强度很低，感受器膜只出现很小的局部变化，通常难以达到兴奋阈值。';
      }else if(sensitivity<30){
        condition=
          '感受器相对灵敏度较低，相同外界刺激形成的有效刺激和感受器电位较小。';
      }else if(!excited){
        condition=
          '有效刺激尚未达到当前阈值，因此只形成局部渐变电位，没有触发全或无动作电位。';
      }else if(density<30){
        condition=
          '感受器相对密度较低，刺激增强时可募集的感受器和传入纤维数量有限。';
      }else if(adaptation>78&&processTime>60){
        condition=
          '持续刺激时间较长且适应速度较快，当前冲动频率已明显低于刺激开始阶段。';
      }else if(threshold<25){
        condition=
          '当前阈值较低，较弱的有效刺激即可触发传入神经动作电位。';
      }else if(processTime<16){
        condition=
          '过程时间较短，刺激刚到达感受器，刺激转换和放电过程尚处于开始阶段。';
      }

      var principle=mode==='receptors'
        ?'不同感受器对不同类型的适宜刺激最敏感，感受器把外界或体内刺激转换为神经系统可以传递的信息。'
        :mode==='transduction'
          ?'刺激转换首先形成大小可变的感受器电位；达到阈值后，传入神经纤维产生全或无动作电位。'
          :mode==='coding'
            ?'刺激强弱通常通过动作电位频率、感受器募集和神经纤维募集等方式编码，而不是连续改变单个动作电位幅度。'
            :'快适应感受器突出刺激变化，慢适应感受器可在持续刺激期间维持相对稳定的信号。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前有效刺激指数 '
        +effectiveStimulus.toFixed(0)
        +'，感受器电位指数 '
        +receptorPotential.toFixed(0)
        +'，冲动频率指数 '
        +frequency.toFixed(0)
        +'。感受器负责接受和转换刺激，感觉最终在相应中枢形成；所有数值仅供课堂比较。';
    }

    for(var i=0;i<receptorButtons.length;i++){
      receptorButtons[i].onclick=function(){
        receptorType=this.getAttribute(
          'data-receptor'
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

    stimulusInput.oninput=update;
    thresholdInput.oninput=update;
    sensitivityInput.oninput=update;
    densityInput.oninput=update;
    adaptationInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
