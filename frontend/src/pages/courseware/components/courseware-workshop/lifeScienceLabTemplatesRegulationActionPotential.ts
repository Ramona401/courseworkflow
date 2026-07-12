/**
 * lifeScienceLabTemplatesRegulationActionPotential.ts
 *
 * 平面生命科学实验室：神经冲动的产生与传导。
 *
 * 教学目标：
 * 1. 认识神经元静息状态下膜内外钠离子、钾离子的相对分布；
 * 2. 理解静息电位与离子浓度梯度、膜选择透过性有关；
 * 3. 理解钠钾泵长期维持钠钾离子浓度梯度，
 *    但动作电位快速变化主要由电压门控离子通道开闭引起；
 * 4. 观察阈刺激、去极化、反极化、复极化和超极化过程；
 * 5. 理解单根神经纤维动作电位具有全或无特征；
 * 6. 理解刺激强度超过阈值后，
 *    单个动作电位幅度不会随刺激强度继续增大；
 * 7. 观察钠离子内流和钾离子外流与膜电位变化的对应关系；
 * 8. 理解绝对不应期和相对不应期对冲动单向传播及放电频率的意义；
 * 9. 比较无髓神经纤维的连续传导与有髓神经纤维的跳跃传导；
 * 10. 理解髓鞘完整程度和神经纤维直径会影响相对传导速度。
 *
 * 科学边界：
 * 1. 静息状态下膜外钠离子相对较多，膜内钾离子相对较多；
 * 2. 静息电位主要与钾离子漏通道、膜选择透过性和离子浓度梯度有关；
 * 3. 钠钾泵通过消耗ATP维持离子浓度梯度，
 *    不应把动作电位上升支直接解释为钠钾泵瞬时加速；
 * 4. 达到阈值后，电压门控钠通道大量开放，
 *    钠离子顺电化学梯度内流，引起快速去极化；
 * 5. 随后钠通道失活，电压门控钾通道开放，
 *    钾离子外流使膜电位复极化；
 * 6. 钾通道关闭相对滞后可形成短暂超极化；
 * 7. 单根神经纤维的单个动作电位具有全或无特征，
 *    刺激强度常通过冲动频率和募集神经纤维数量编码，
 *    而不是通过增大单个动作电位幅度编码；
 * 8. 动作电位沿轴突传播时，相邻膜段依次去极化；
 * 9. 已兴奋膜段进入不应期，有助于冲动通常向前传播；
 * 10. 有髓神经纤维主要在郎飞结处产生动作电位，
 *     冲动表现为郎飞结之间的跳跃传导；
 * 11. 髓鞘受损通常会降低传导效率，
 *     但本模型不模拟具体神经系统疾病；
 * 12. 真实动作电位受温度、离子浓度、通道类型、
 *     轴突结构和细胞状态等多种因素影响；
 * 13. 图中的膜电位、通道活性、传导速度和不应期均为教学示意；
 * 14. 本模板只用于生物学教学，
 *     不用于神经功能、电生理结果或疾病判断。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript，
 *    不依赖外部图片、脚本、样式、字体或CDN；
 * 2. 所有DOM查询均限定在rootId内部，
 *    支持同页多个独立实例；
 * 3. 使用生命科学统一.bl-*布局协议；
 * 4. 支持参数滑杆、四种观察模式、自动推进、
 *    结构标注开关、动态图示和即时教学结论；
 * 5. 本文件只导出独立模板数组，
 *    聚合入口由后续C1批次统一接入。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

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

function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

function actionPotentialStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C7D2FE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0E7FF,#EDE9FE);border-bottom:1px solid #C7D2FE}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FAFAFF;border-right:1px solid #C7D2FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#4F46E5;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#4F46E5}'
    + '#' + rootId + ' .ap-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .ap-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ap-button{min-height:30px;padding:3px;border:1px solid #A5B4FC;border-radius:8px;background:#fff;color:#3730A3;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .ap-button.active{border-color:#4F46E5;background:#E0E7FF;box-shadow:0 3px 9px rgba(79,70,229,.14)}'
    + '#' + rootId + ' .ap-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#818CF8,#4F46E5);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ap-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ap-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .ap-card{padding:6px 3px;border:1px solid #C7D2FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ap-card b{display:block;font-size:13px;color:#4338CA}'
    + '#' + rootId + ' .ap-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#E0E7FF;color:#312E81;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ap-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--ap-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ap-pulse{animation:' + rootId + '-pulse 1.5s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.35}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_ACTION_POTENTIAL:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-action-potential-conduction',
    group: '🧠 稳态与调节',
    name: '神经冲动的产生与传导',
    emoji: '⚡',
    desc: '调节刺激、阈值、钠钾通道活性、髓鞘完整度、纤维直径和过程时间，观察动作电位与神经冲动传导',
    params: [
      {
        key: 'stimulusIntensity',
        label: '刺激强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'thresholdLevel',
        label: '兴奋阈值',
        type: 'number',
        min: 20,
        max: 80,
        step: 1,
        defaultValue: 48,
      },
      {
        key: 'sodiumChannelActivity',
        label: '钠通道相对活性',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 88,
      },
      {
        key: 'potassiumChannelActivity',
        label: '钾通道相对活性',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'myelinIntegrity',
        label: '髓鞘完整程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 84,
      },
      {
        key: 'fiberDiameter',
        label: '神经纤维相对直径',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'processTime',
        label: '过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 38,
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
        48,
      )
      const sodiumChannelActivity = num(
        params,
        'sodiumChannelActivity',
        88,
      )
      const potassiumChannelActivity = num(
        params,
        'potassiumChannelActivity',
        82,
      )
      const myelinIntegrity = num(
        params,
        'myelinIntegrity',
        84,
      )
      const fiberDiameter = num(
        params,
        'fiberDiameter',
        62,
      )
      const processTime = num(
        params,
        'processTime',
        38,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${actionPotentialStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">⚡ 神经冲动的产生与传导</div>
    <div class="bl-note">膜电位、通道活性和传导速度均为教学示意</div>
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
          <span>兴奋阈值</span>
          <span class="bl-value" data-threshold-value></span>
        </div>
        <input
          data-threshold
          type="range"
          min="20"
          max="80"
          step="1"
          value="${n(thresholdLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>钠通道相对活性</span>
          <span class="bl-value" data-sodium-value></span>
        </div>
        <input
          data-sodium
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(sodiumChannelActivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>钾通道相对活性</span>
          <span class="bl-value" data-potassium-value></span>
        </div>
        <input
          data-potassium
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(potassiumChannelActivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>髓鞘完整程度</span>
          <span class="bl-value" data-myelin-value></span>
        </div>
        <input
          data-myelin
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(myelinIntegrity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>神经纤维相对直径</span>
          <span class="bl-value" data-diameter-value></span>
        </div>
        <input
          data-diameter
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(fiberDiameter)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>过程时间</span>
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

      <div class="ap-subtitle">观察方式</div>

      <div class="ap-buttons">
        <button
          type="button"
          class="ap-button active"
          data-mode="resting"
        >静息电位与离子分布</button>

        <button
          type="button"
          class="ap-button"
          data-mode="action"
        >动作电位各阶段</button>

        <button
          type="button"
          class="ap-button"
          data-mode="conduction"
        >轴突上的冲动传播</button>

        <button
          type="button"
          class="ap-button"
          data-mode="comparison"
        >连续与跳跃传导</button>
      </div>

      <button
        type="button"
        class="ap-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="ap-toggle"
        data-auto
      >过程推进：运行中</button>

      <div class="ap-status">
        <div class="ap-card">
          <b data-potential></b>
          <span>膜电位示意</span>
        </div>

        <div class="ap-card">
          <b data-phase></b>
          <span>当前阶段</span>
        </div>

        <div class="ap-card">
          <b data-speed></b>
          <span>传导指数</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="神经冲动的产生与传导互动示意图"
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
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>

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
              flood-color="#312E81"
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
          fill="#3730A3"
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
            fill="#EEF2FF"
            stroke="#C7D2FE"
            stroke-width="2"
          />

          <text
            x="110"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#3730A3"
          >关键边界</text>

          <text
            x="110"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#312E81"
          >达到阈值后产生全或无反应</text>

          <text
            x="110"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#312E81"
          >刺激增强不放大单个动作电位</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#3730A3"
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
    var sodiumInput=root.querySelector(
      '[data-sodium]'
    );
    var potassiumInput=root.querySelector(
      '[data-potassium]'
    );
    var myelinInput=root.querySelector(
      '[data-myelin]'
    );
    var diameterInput=root.querySelector(
      '[data-diameter]'
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
    var sodiumValue=root.querySelector(
      '[data-sodium-value]'
    );
    var potassiumValue=root.querySelector(
      '[data-potassium-value]'
    );
    var myelinValue=root.querySelector(
      '[data-myelin-value]'
    );
    var diameterValue=root.querySelector(
      '[data-diameter-value]'
    );
    var timeValue=root.querySelector(
      '[data-time-value]'
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
    var phaseText=root.querySelector(
      '[data-phase]'
    );
    var speedText=root.querySelector(
      '[data-speed]'
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

    var mode='resting';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

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
      },760);
    }

    function ion(
      x,
      y,
      label,
      color,
      opacity,
      radius
    ){
      return ''
        +'<g transform="translate('+x+' '+y+')" opacity="'+opacity+'">'
        +'<circle r="'+radius+'" fill="'+color+'" stroke="#FFFFFF" stroke-width="2"/>'
        +'<text x="0" y="3" text-anchor="middle" font-size="7" font-weight="900" fill="#FFFFFF">'+label+'</text>'
        +'</g>';
    }

    function potentialAt(
      progress,
      excited,
      sodium,
      potassium
    ){
      if(!excited){
        var local=Math.sin(
          progress*Math.PI
        );

        return -70+local*12;
      }

      var peak=22+sodium*.12;
      var under=-76-potassium*.07;

      if(progress<.14){
        return -70;
      }

      if(progress<.34){
        var up=(progress-.14)/.20;

        return -70
          +(peak+70)
          *Math.pow(up,1.55);
      }

      if(progress<.59){
        var down=(progress-.34)/.25;

        return peak
          +(under-peak)
          *Math.pow(down,.78);
      }

      if(progress<.79){
        var recover=(progress-.59)/.20;

        return under
          +(-70-under)
          *recover;
      }

      return -70;
    }

    function phaseAt(
      progress,
      excited
    ){
      if(!excited){
        return '未达阈值';
      }

      if(progress<.14){
        return '静息期';
      }

      if(progress<.34){
        return '去极化';
      }

      if(progress<.59){
        return '复极化';
      }

      if(progress<.79){
        return '超极化';
      }

      return '恢复静息';
    }

    function renderResting(
      sodium,
      potassium
    ){
      var outside='';
      var inside='';

      var sodiumOutside=Math.floor(
        7+sodium/12
      );
      var sodiumInside=3;
      var potassiumOutside=3;
      var potassiumInside=Math.floor(
        7+potassium/12
      );

      for(var i=0;i<sodiumOutside;i++){
        outside+=ion(
          77+(i%8)*42,
          128+Math.floor(i/8)*35,
          'Na',
          '#2563EB',
          .82,
          10
        );
      }

      for(var j=0;j<potassiumOutside;j++){
        outside+=ion(
          102+j*83,
          190,
          'K',
          '#F59E0B',
          .62,
          10
        );
      }

      for(var k=0;k<sodiumInside;k++){
        inside+=ion(
          110+k*91,
          305,
          'Na',
          '#2563EB',
          .55,
          10
        );
      }

      for(var q=0;q<potassiumInside;q++){
        inside+=ion(
          72+(q%8)*43,
          247+Math.floor(q/8)*35,
          'K',
          '#F59E0B',
          .84,
          10
        );
      }

      return ''
        +'<rect x="28" y="91" width="450" height="278" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="253" y="119" text-anchor="middle" font-size="15" font-weight="900" fill="#334155">轴突膜静息状态</text>'
        +'<rect x="49" y="132" width="407" height="77" rx="18" fill="#EFF6FF" stroke="#93C5FD" stroke-width="3"/>'
        +'<rect x="49" y="229" width="407" height="112" rx="18" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<rect x="49" y="209" width="407" height="20" rx="10" fill="#A78BFA" stroke="#6D28D9" stroke-width="3"/>'
        +outside
        +inside
        +'<text x="68" y="153" font-size="12" font-weight="900" fill="#1D4ED8">膜外：Na⁺相对较多</text>'
        +'<text x="68" y="329" font-size="12" font-weight="900" fill="#92400E">膜内：K⁺相对较多，膜内相对负</text>'
        +'<g transform="translate(365 184)">'
        +'<rect width="70" height="70" rx="16" fill="#ECFDF5" stroke="#10B981" stroke-width="4"/>'
        +'<text x="35" y="24" text-anchor="middle" font-size="10" font-weight="900" fill="#047857">Na⁺/K⁺泵</text>'
        +'<path class="ap-flow" d="M19 47 H49" fill="none" stroke="#10B981" stroke-width="4" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<text x="35" y="65" text-anchor="middle" font-size="9" font-weight="800" fill="#166534">长期维持梯度</text>'
        +'</g>'
        +'<g transform="translate(505 98)">'
        +'<rect width="221" height="226" rx="22" fill="#EEF2FF" stroke="#C7D2FE" stroke-width="3"/>'
        +'<text x="110" y="29" text-anchor="middle" font-size="14" font-weight="900" fill="#3730A3">静息电位形成要点</text>'
        +'<text x="18" y="63" font-size="11.5" font-weight="900" fill="#4338CA">1. 离子分布不均</text>'
        +'<text x="18" y="87" font-size="10.5" font-weight="800" fill="#475569">膜外Na⁺较多，膜内K⁺较多</text>'
        +'<text x="18" y="121" font-size="11.5" font-weight="900" fill="#4338CA">2. 选择透过性</text>'
        +'<text x="18" y="145" font-size="10.5" font-weight="800" fill="#475569">静息时膜对K⁺通透性相对较高</text>'
        +'<text x="18" y="179" font-size="11.5" font-weight="900" fill="#4338CA">3. 钠钾泵维持梯度</text>'
        +'<text x="18" y="203" font-size="10.5" font-weight="800" fill="#475569">消耗ATP，不是上升支直接原因</text>'
        +'</g>';
    }

    function buildPotentialPath(
      excited,
      sodium,
      potassium
    ){
      var path='';

      for(var i=0;i<=100;i+=2){
        var progress=i/100;
        var potential=potentialAt(
          progress,
          excited,
          sodium,
          potassium
        );
        var x=75+i*4.05;
        var y=285-(potential+90)*1.65;

        path+=(i===0?'M':' L')
          +x.toFixed(1)
          +' '
          +y.toFixed(1);
      }

      return path;
    }

    function renderAction(
      progress,
      excited,
      sodium,
      potassium,
      potential,
      phase
    ){
      var markerX=75+progress*405;
      var markerY=285-(potential+90)*1.65;
      var sodiumOpen=phase==='去极化';
      var potassiumOpen=
        phase==='复极化'
        ||phase==='超极化';

      var thresholdY=
        285-(-55+90)*1.65;

      return ''
        +'<g transform="translate(26 89)">'
        +'<rect width="480" height="284" rx="23" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="240" y="29" text-anchor="middle" font-size="15" font-weight="900" fill="#334155">膜电位随时间变化</text>'
        +'<line x1="49" y1="196" x2="455" y2="196" stroke="#CBD5E1" stroke-width="2" stroke-dasharray="6 6"/>'
        +'<line x1="49" y1="'+thresholdY.toFixed(1)+'" x2="455" y2="'+thresholdY.toFixed(1)+'" stroke="#F59E0B" stroke-width="2.5" stroke-dasharray="8 6"/>'
        +'<line x1="49" y1="46" x2="49" y2="248" stroke="#64748B" stroke-width="3"/>'
        +'<line x1="49" y1="248" x2="455" y2="248" stroke="#64748B" stroke-width="3"/>'
        +'<text x="9" y="52" font-size="10" font-weight="900" fill="#475569">+30</text>'
        +'<text x="13" y="201" font-size="10" font-weight="900" fill="#475569">-70</text>'
        +'<text x="6" y="'+(thresholdY-5).toFixed(1)+'" font-size="10" font-weight="900" fill="#B45309">阈值</text>'
        +'<text x="428" y="270" font-size="10" font-weight="900" fill="#475569">时间</text>'
        +'<path d="'+buildPotentialPath(excited,sodium,potassium)+'" transform="translate(-26 -89)" fill="none" stroke="'+(excited?'#7C3AED':'#94A3B8')+'" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/>'
        +'<line x1="'+(markerX-26).toFixed(1)+'" y1="45" x2="'+(markerX-26).toFixed(1)+'" y2="248" stroke="#4F46E5" stroke-width="2" stroke-dasharray="5 5"/>'
        +'<circle cx="'+(markerX-26).toFixed(1)+'" cy="'+(markerY-89).toFixed(1)+'" r="9" fill="#4F46E5" stroke="#FFFFFF" stroke-width="3"/>'
        +'</g>'
        +'<g transform="translate(530 96)">'
        +'<rect width="202" height="259" rx="22" fill="#EEF2FF" stroke="#C7D2FE" stroke-width="3"/>'
        +'<text x="101" y="29" text-anchor="middle" font-size="14" font-weight="900" fill="#3730A3">当前通道与离子流</text>'
        +'<g transform="translate(21 51)">'
        +'<rect width="160" height="66" rx="14" fill="'+(sodiumOpen?'#DBEAFE':'#F8FAFC')+'" stroke="#2563EB" stroke-width="'+(sodiumOpen?5:3)+'"/>'
        +'<text x="80" y="24" text-anchor="middle" font-size="12" font-weight="900" fill="#1D4ED8">电压门控Na⁺通道</text>'
        +'<text x="80" y="47" text-anchor="middle" font-size="11" font-weight="800" fill="#475569">'+(sodiumOpen?'大量开放，Na⁺内流':'关闭或失活')+'</text>'
        +'</g>'
        +'<g transform="translate(21 132)">'
        +'<rect width="160" height="66" rx="14" fill="'+(potassiumOpen?'#FEF3C7':'#F8FAFC')+'" stroke="#D97706" stroke-width="'+(potassiumOpen?5:3)+'"/>'
        +'<text x="80" y="24" text-anchor="middle" font-size="12" font-weight="900" fill="#92400E">电压门控K⁺通道</text>'
        +'<text x="80" y="47" text-anchor="middle" font-size="11" font-weight="800" fill="#475569">'+(potassiumOpen?'开放，K⁺外流':'多数关闭')+'</text>'
        +'</g>'
        +'<text x="101" y="225" text-anchor="middle" font-size="12" font-weight="900" fill="#4338CA">'+phase+'</text>'
        +'<text x="101" y="246" text-anchor="middle" font-size="11" font-weight="800" fill="#475569">'+potential.toFixed(0)+' mV教学示意</text>'
        +'</g>';
    }

    function renderConduction(
      progress,
      excited,
      myelin,
      diameter,
      speed
    ){
      var startX=73;
      var travel=570;
      var signalX=excited
        ?startX+travel*progress
        :startX+travel*.08;

      var axonWidth=18+diameter*.12;
      var myelinated=myelin>=35;
      var myelinHTML='';
      var nodesHTML='';

      if(myelinated){
        for(var i=0;i<6;i++){
          var x=90+i*101;

          myelinHTML+=''
            +'<rect x="'+x+'" y="'+(203-axonWidth/2-13)+'" width="77" height="'+(axonWidth+26)+'" rx="22" fill="#FDE68A" stroke="#D97706" stroke-width="'+(2+myelin/35)+'" opacity="'+(.35+.65*myelin/100)+'"/>';

          if(i<5){
            nodesHTML+=''
              +'<rect x="'+(x+80)+'" y="'+(203-axonWidth/2-6)+'" width="17" height="'+(axonWidth+12)+'" rx="7" fill="#E0E7FF" stroke="#4F46E5" stroke-width="3"/>';
          }
        }
      }

      var refractoryStart=clamp(
        signalX-145,
        startX,
        startX+travel
      );

      var activeWidth=myelinated?28:55;

      return ''
        +'<rect x="27" y="92" width="706" height="268" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="380" y="122" text-anchor="middle" font-size="15" font-weight="900" fill="#334155">动作电位沿轴突膜向前传播</text>'
        +'<line x1="'+startX+'" y1="203" x2="'+(startX+travel)+'" y2="203" stroke="#A78BFA" stroke-width="'+axonWidth+'" stroke-linecap="round"/>'
        +myelinHTML
        +nodesHTML
        +'<rect x="'+refractoryStart.toFixed(1)+'" y="'+(176-axonWidth/2)+'" width="'+Math.max(0,signalX-refractoryStart-20).toFixed(1)+'" height="'+(axonWidth+54)+'" rx="18" fill="#94A3B8" opacity=".22"/>'
        +'<circle class="ap-pulse" cx="'+signalX.toFixed(1)+'" cy="203" r="'+activeWidth+'" fill="#EF4444" opacity="'+(excited?.76:.28)+'"/>'
        +'<circle cx="'+signalX.toFixed(1)+'" cy="203" r="'+(activeWidth*.46)+'" fill="#F97316" stroke="#FFFFFF" stroke-width="4"/>'
        +'<path class="ap-flow" d="M'+(signalX+activeWidth).toFixed(1)+' 203 H'+clamp(signalX+124,startX,startX+travel).toFixed(1)+'" fill="none" stroke="#2563EB" stroke-width="6" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="131" y="285" text-anchor="middle" font-size="11" font-weight="900" fill="#64748B">已恢复或处于不应期的膜段</text>'
        +'<text x="380" y="321" text-anchor="middle" font-size="12" font-weight="900" fill="#4338CA">'+(myelinated?'主要在郎飞结处依次兴奋':'相邻膜段连续依次兴奋')+'</text>'
        +'<g transform="translate(50 135)">'
        +'<rect width="150" height="41" rx="12" fill="#EEF2FF" stroke="#C7D2FE" stroke-width="2"/>'
        +'<text x="75" y="17" text-anchor="middle" font-size="10.5" font-weight="900" fill="#3730A3">局部电流刺激前方膜段</text>'
        +'<text x="75" y="33" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">后方膜段进入不应期</text>'
        +'</g>'
        +'<g transform="translate(552 135)">'
        +'<rect width="153" height="41" rx="12" fill="#ECFDF5" stroke="#A7F3D0" stroke-width="2"/>'
        +'<text x="76" y="17" text-anchor="middle" font-size="10.5" font-weight="900" fill="#047857">相对传导指数 '+speed.toFixed(0)+'</text>'
        +'<text x="76" y="33" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">受直径和髓鞘影响</text>'
        +'</g>';
    }

    function renderComparison(
      progress,
      myelin,
      diameter,
      speed
    ){
      var unmyelinatedSpeed=clamp(
        18+diameter*.36,
        10,
        60
      );

      var myelinatedSpeed=clamp(
        24+diameter*.36+myelin*.52,
        12,
        100
      );

      var topX=90+520*clamp(
        progress*unmyelinatedSpeed/60,
        0,
        1
      );

      var bottomX=90+520*clamp(
        progress*myelinatedSpeed/60,
        0,
        1
      );

      var myelinHTML='';

      for(var i=0;i<6;i++){
        var x=93+i*93;

        myelinHTML+=''
          +'<rect x="'+x+'" y="266" width="69" height="46" rx="18" fill="#FDE68A" stroke="#D97706" stroke-width="'+(2+myelin/35)+'" opacity="'+(.30+.70*myelin/100)+'"/>';

        if(i<5){
          myelinHTML+=''
            +'<rect x="'+(x+72)+'" y="271" width="17" height="36" rx="7" fill="#E0E7FF" stroke="#4F46E5" stroke-width="3"/>';
        }
      }

      return ''
        +'<rect x="28" y="91" width="704" height="276" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="380" y="121" text-anchor="middle" font-size="15" font-weight="900" fill="#334155">相同时间内两种传导方式的距离比较</text>'
        +'<text x="61" y="169" font-size="13" font-weight="900" fill="#0369A1">无髓神经纤维</text>'
        +'<line x1="90" y1="206" x2="650" y2="206" stroke="#60A5FA" stroke-width="'+(15+diameter*.08)+'" stroke-linecap="round"/>'
        +'<circle cx="'+topX.toFixed(1)+'" cy="206" r="21" fill="#EF4444" stroke="#FFFFFF" stroke-width="4"/>'
        +'<path class="ap-flow" d="M'+Math.max(90,topX-65).toFixed(1)+' 206 H'+Math.min(650,topX+65).toFixed(1)+'" fill="none" stroke="#DC2626" stroke-width="4" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="674" y="211" font-size="11" font-weight="900" fill="#0369A1">连续传导</text>'
        +'<text x="61" y="259" font-size="13" font-weight="900" fill="#92400E">有髓神经纤维</text>'
        +'<line x1="90" y1="289" x2="650" y2="289" stroke="#A78BFA" stroke-width="'+(15+diameter*.08)+'" stroke-linecap="round"/>'
        +myelinHTML
        +'<circle cx="'+bottomX.toFixed(1)+'" cy="289" r="21" fill="#EF4444" stroke="#FFFFFF" stroke-width="4"/>'
        +'<path class="ap-flow" d="M'+Math.max(90,bottomX-75).toFixed(1)+' 289 H'+Math.min(650,bottomX+75).toFixed(1)+'" fill="none" stroke="#7C3AED" stroke-width="4" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<text x="674" y="294" font-size="11" font-weight="900" fill="#92400E">跳跃传导</text>'
        +'<g transform="translate(72 326)">'
        +'<rect width="616" height="27" rx="13" fill="#EEF2FF" stroke="#C7D2FE" stroke-width="2"/>'
        +'<text x="308" y="18" text-anchor="middle" font-size="11" font-weight="900" fill="#3730A3">髓鞘越完整、纤维直径越大，通常越有利于提高相对传导速度；当前综合指数 '+speed.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='resting'){
        labels.innerHTML=''
          +'<path d="M235 209 L235 179" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="176" y="172" font-size="13" font-weight="900" fill="#5B21B6">神经元细胞膜</text>'
          +'<path d="M404 219 L469 186" stroke="#10B981" stroke-width="2.5"/>'
          +'<text x="477" y="185" font-size="13" font-weight="900" fill="#047857">钠钾泵</text>';
        return;
      }

      if(modeName==='action'){
        labels.innerHTML=''
          +'<path d="M183 131 L208 98" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="215" y="95" font-size="13" font-weight="900" fill="#1D4ED8">去极化上升支</text>'
          +'<path d="M355 166 L390 118" stroke="#D97706" stroke-width="2.5"/>'
          +'<text x="397" y="115" font-size="13" font-weight="900" fill="#92400E">复极化下降支</text>'
          +'<path d="M438 286 L470 315" stroke="#64748B" stroke-width="2.5"/>'
          +'<text x="477" y="322" font-size="13" font-weight="900" fill="#475569">超极化</text>';
        return;
      }

      if(modeName==='conduction'){
        labels.innerHTML=''
          +'<path d="M373 203 L373 155" stroke="#EF4444" stroke-width="2.5"/>'
          +'<text x="316" y="149" font-size="13" font-weight="900" fill="#B91C1C">兴奋膜段</text>'
          +'<path d="M264 235 L229 270" stroke="#64748B" stroke-width="2.5"/>'
          +'<text x="132" y="286" font-size="13" font-weight="900" fill="#475569">不应期膜段</text>'
          +'<path d="M505 203 L534 164" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="541" y="161" font-size="13" font-weight="900" fill="#1D4ED8">传播方向</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M245 206 L245 177" stroke="#2563EB" stroke-width="2.5"/>'
        +'<text x="191" y="171" font-size="13" font-weight="900" fill="#1D4ED8">相邻膜段连续兴奋</text>'
        +'<path d="M430 267 L430 235" stroke="#D97706" stroke-width="2.5"/>'
        +'<text x="376" y="229" font-size="13" font-weight="900" fill="#92400E">髓鞘</text>'
        +'<path d="M526 288 L526 253" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="485" y="247" font-size="13" font-weight="900" fill="#5B21B6">郎飞结</text>';
    }

    function update(){
      var stimulus=Number(
        stimulusInput.value
      );
      var threshold=Number(
        thresholdInput.value
      );
      var sodium=Number(
        sodiumInput.value
      );
      var potassium=Number(
        potassiumInput.value
      );
      var myelin=Number(
        myelinInput.value
      );
      var diameter=Number(
        diameterInput.value
      );
      var processTime=Number(
        timeInput.value
      );

      stimulusValue.textContent=
        stimulus.toFixed(0)+'%';
      thresholdValue.textContent=
        threshold.toFixed(0)+'%';
      sodiumValue.textContent=
        sodium.toFixed(0)+'%';
      potassiumValue.textContent=
        potassium.toFixed(0)+'%';
      myelinValue.textContent=
        myelin.toFixed(0)+'%';
      diameterValue.textContent=
        diameter.toFixed(0)+'%';
      timeValue.textContent=
        processTime.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute(
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

      var progress=processTime/100;
      var excited=stimulus>=threshold;
      var potential=potentialAt(
        progress,
        excited,
        sodium,
        potassium
      );
      var phase=phaseAt(
        progress,
        excited
      );

      var conductionSpeed=clamp(
        12
        +diameter*.36
        +myelin*.50,
        8,
        100
      );

      if(!excited){
        conductionSpeed=0;
      }

      if(myelin<18){
        conductionSpeed*=.58;
      }

      potentialText.textContent=
        potential.toFixed(0)+'mV';
      phaseText.textContent=phase;
      speedText.textContent=
        conductionSpeed.toFixed(0);

      root.style.setProperty(
        '--ap-speed',
        clamp(
          2.5-conductionSpeed/68,
          .55,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='resting'){
        title.textContent=
          '静息电位、离子分布与钠钾泵';

        summary.textContent=
          '观察神经元静息状态下膜内外钠钾离子的相对分布与膜选择透过性。';

        dynamic.innerHTML=renderResting(
          sodium,
          potassium
        );

        stageNote.textContent=
          '钠钾泵长期维持离子浓度梯度，静息电位还与膜对离子的选择透过性有关。';

        renderLabels(mode);
      }else if(mode==='action'){
        title.textContent=
          '阈刺激与动作电位各阶段';

        summary.textContent=
          '观察去极化、复极化、超极化及钠钾通道状态的对应关系。';

        dynamic.innerHTML=renderAction(
          progress,
          excited,
          sodium,
          potassium,
          potential,
          phase
        );

        stageNote.textContent=excited
          ?'刺激达到阈值后产生全或无动作电位，单个动作电位幅度不由刺激强度继续放大。'
          :'当前刺激没有达到兴奋阈值，只形成局部电位变化。';

        renderLabels(mode);
      }else if(mode==='conduction'){
        title.textContent=
          '动作电位沿轴突膜向前传播';

        summary.textContent=
          '观察局部电流、相邻膜段兴奋和后方不应期之间的关系。';

        dynamic.innerHTML=renderConduction(
          progress,
          excited,
          myelin,
          diameter,
          conductionSpeed
        );

        stageNote.textContent=excited
          ?'已兴奋膜段进入不应期，有助于神经冲动通常向前传播。'
          :'刺激未达到阈值，轴突上没有形成可传播的动作电位。';

        renderLabels(mode);
      }else{
        title.textContent=
          '无髓连续传导与有髓跳跃传导';

        summary.textContent=
          '比较相同时间内连续传导和郎飞结间跳跃传导的相对距离。';

        dynamic.innerHTML=renderComparison(
          progress,
          myelin,
          diameter,
          conductionSpeed
        );

        stageNote.textContent=
          '有髓神经纤维主要在郎飞结处产生动作电位，完整髓鞘通常有利于提高传导效率。';

        renderLabels(mode);
      }

      var condition=
        '当前刺激达到阈值，钠钾通道活性和轴突结构能够支持动作电位产生与传播。';

      if(!excited){
        condition=
          '当前刺激低于兴奋阈值，只能引起局部膜电位变化，不能形成全或无动作电位。';
      }else if(sodium<35){
        condition=
          '钠通道相对活性较低，快速去极化和动作电位峰值受到限制。';
      }else if(potassium<35){
        condition=
          '钾通道相对活性较低，复极化和恢复静息状态的过程相对减慢。';
      }else if(myelin<18){
        condition=
          '髓鞘完整程度较低，膜电流泄漏增加，跳跃传导效率明显下降。';
      }else if(diameter<28){
        condition=
          '神经纤维相对直径较小，轴向电流传播阻力相对较大，传导指数较低。';
      }else if(processTime<14){
        condition=
          '过程时间较短，膜仍处于刺激开始或静息准备阶段。';
      }

      var principle=mode==='resting'
        ?'静息电位来自离子浓度梯度、膜选择透过性和离子主动运输的共同作用。'
        :mode==='action'
          ?'达到阈值后钠通道快速开放引起去极化，随后钠通道失活、钾通道开放引起复极化和短暂超极化。'
          :mode==='conduction'
            ?'局部电流使前方膜段达到阈值，后方膜段进入不应期，使冲动通常沿轴突向前传播。'
            :'无髓纤维依靠相邻膜段连续兴奋，有髓纤维主要依靠郎飞结之间的跳跃传导。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前膜电位示意 '
        +potential.toFixed(0)
        +' mV，相对传导指数 '
        +conductionSpeed.toFixed(0)
        +'；所有数值只用于课堂比较，不用于神经功能、电生理或疾病判断。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
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
    sodiumInput.oninput=update;
    potassiumInput.oninput=update;
    myelinInput.oninput=update;
    diameterInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
