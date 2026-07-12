/**
 * lifeScienceLabTemplatesRegulationSynapticTransmission.ts
 *
 * 平面生命科学实验室：突触传递与兴奋、抑制。
 *
 * 教学目标：
 * 1. 认识化学突触由突触前膜、突触间隙和突触后膜组成；
 * 2. 理解动作电位到达突触前末梢后，
 *    电压门控钙通道开放并引起钙离子内流；
 * 3. 理解钙离子促进突触小泡与突触前膜融合，
 *    神经递质通过胞吐释放到突触间隙；
 * 4. 观察神经递质扩散、受体结合和离子通道开放过程；
 * 5. 理解兴奋性突触后电位是局部去极化，
 *    抑制性突触后电位是局部超极化或分流性抑制；
 * 6. 理解“兴奋性”不等于一定产生动作电位，
 *    “抑制性”也不等于神经元完全停止活动；
 * 7. 理解突触后效应取决于受体类型、离子通道和离子梯度，
 *    不能只根据神经递质名称判断兴奋或抑制；
 * 8. 观察时间总和和空间总和；
 * 9. 理解神经元可同时接受多个兴奋性和抑制性突触输入；
 * 10. 理解轴丘附近对多个突触后电位进行整合，
 *     净效应达到阈值后才可能产生动作电位；
 * 11. 理解神经递质的酶解、重摄取和扩散清除有助于终止信号；
 * 12. 理解化学突触通常具有单向传递和突触延搁特点。
 *
 * 科学边界：
 * 1. 动作电位到达突触前末梢后，
 *    电压门控钙通道开放，钙离子内流；
 * 2. 钙离子促进突触小泡融合和神经递质胞吐释放；
 * 3. 神经递质通过突触间隙扩散并与突触后受体结合；
 * 4. 受体激活后可直接或间接改变离子通道开放状态；
 * 5. 兴奋性突触后电位通常使膜电位接近动作电位阈值；
 * 6. 抑制性突触后电位通常使膜电位远离阈值，
 *    或通过提高膜电导削弱兴奋性输入；
 * 7. 兴奋和抑制是对突触后膜效应的描述，
 *    不是对神经递质名称作绝对分类；
 * 8. 同一种神经递质作用于不同受体时，
 *    可能产生不同的突触后效应；
 * 9. 突触后电位是可发生总和的局部渐变电位，
 *    不具有单个动作电位那样的全或无特征；
 * 10. 时间总和是同一突触在较短时间内连续活动产生的叠加；
 * 11. 空间总和是多个突触在相近时间活动产生的叠加；
 * 12. 兴奋性与抑制性输入的整合通常发生在胞体和轴丘附近；
 * 13. 化学突触通常由突触前结构释放递质、
 *     突触后结构表达受体，因此通常表现为单向传递；
 * 14. 递质清除可通过酶解、突触前膜重摄取、
 *     胶质细胞摄取和扩散等方式完成；
 * 15. 本模型不模拟所有神经递质、受体亚型、
 *     第二信使和神经网络连接；
 * 16. 图中的钙信号、递质浓度、突触后电位、
 *     总和强度和触发概率均为相对教学指标；
 * 17. 本模板只用于生物学教学，
 *     不用于神经功能、药物作用或疾病判断。
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

function synapticTransmissionStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #DDD6FE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#E0E7FF);border-bottom:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#7C3AED}'
    + '#' + rootId + ' .st-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .st-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .st-button{min-height:30px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .st-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .st-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .st-toggle.off{background:#64748B}'
    + '#' + rootId + ' .st-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .st-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .st-card b{display:block;font-size:13px;color:#6D28D9}'
    + '#' + rootId + ' .st-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .st-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--st-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .st-pulse{animation:' + rootId + '-pulse 1.5s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.34}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_SYNAPTIC_TRANSMISSION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-synaptic-excitation-inhibition',
    group: '🧠 稳态与调节',
    name: '突触传递与兴奋、抑制',
    emoji: '🔗',
    desc: '调节冲动频率、钙通道、小泡释放、兴奋和抑制输入、受体敏感性及递质清除，观察突触传递与神经元整合',
    params: [
      {
        key: 'presynapticFrequency',
        label: '突触前冲动频率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'calciumChannelActivity',
        label: '钙通道相对活性',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 84,
      },
      {
        key: 'vesicleReleaseCapacity',
        label: '突触小泡释放能力',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 80,
      },
      {
        key: 'excitatoryInput',
        label: '兴奋性输入强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 66,
      },
      {
        key: 'inhibitoryInput',
        label: '抑制性输入强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 34,
      },
      {
        key: 'receptorSensitivity',
        label: '突触后受体敏感性',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'transmitterClearance',
        label: '递质清除效率',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'processTime',
        label: '突触过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const presynapticFrequency = num(
        params,
        'presynapticFrequency',
        62,
      )
      const calciumChannelActivity = num(
        params,
        'calciumChannelActivity',
        84,
      )
      const vesicleReleaseCapacity = num(
        params,
        'vesicleReleaseCapacity',
        80,
      )
      const excitatoryInput = num(
        params,
        'excitatoryInput',
        66,
      )
      const inhibitoryInput = num(
        params,
        'inhibitoryInput',
        34,
      )
      const receptorSensitivity = num(
        params,
        'receptorSensitivity',
        82,
      )
      const transmitterClearance = num(
        params,
        'transmitterClearance',
        72,
      )
      const processTime = num(
        params,
        'processTime',
        42,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${synapticTransmissionStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🔗 突触传递与兴奋、抑制</div>
    <div class="bl-note">递质、膜电位和总和强度均为相对教学值</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>突触前冲动频率</span>
          <span class="bl-value" data-frequency-value></span>
        </div>
        <input
          data-frequency
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(presynapticFrequency)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>钙通道相对活性</span>
          <span class="bl-value" data-calcium-value></span>
        </div>
        <input
          data-calcium
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(calciumChannelActivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>突触小泡释放能力</span>
          <span class="bl-value" data-vesicle-value></span>
        </div>
        <input
          data-vesicle
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(vesicleReleaseCapacity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>兴奋性输入强度</span>
          <span class="bl-value" data-excitatory-value></span>
        </div>
        <input
          data-excitatory
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(excitatoryInput)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>抑制性输入强度</span>
          <span class="bl-value" data-inhibitory-value></span>
        </div>
        <input
          data-inhibitory
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(inhibitoryInput)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>突触后受体敏感性</span>
          <span class="bl-value" data-receptor-value></span>
        </div>
        <input
          data-receptor
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(receptorSensitivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>递质清除效率</span>
          <span class="bl-value" data-clearance-value></span>
        </div>
        <input
          data-clearance
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(transmitterClearance)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>突触过程时间</span>
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

      <div class="st-subtitle">观察方式</div>

      <div class="st-buttons">
        <button
          type="button"
          class="st-button active"
          data-mode="structure"
        >突触结构与单向传递</button>

        <button
          type="button"
          class="st-button"
          data-mode="release"
        >递质释放与受体结合</button>

        <button
          type="button"
          class="st-button"
          data-mode="potentials"
        >兴奋与抑制性电位</button>

        <button
          type="button"
          class="st-button"
          data-mode="summation"
        >时间空间总和与整合</button>
      </div>

      <button
        type="button"
        class="st-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="st-toggle"
        data-auto
      >突触推进：运行中</button>

      <div class="st-status">
        <div class="st-card">
          <b data-release-index></b>
          <span>递质释放</span>
        </div>

        <div class="st-card">
          <b data-potential></b>
          <span>膜电位示意</span>
        </div>

        <div class="st-card">
          <b data-integration></b>
          <span>整合结果</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="突触传递与兴奋抑制互动示意图"
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
            id="${rootId}-arrow-red"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
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
          >兴奋不等于一定产生动作电位</text>

          <text
            x="110"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#4C1D95"
          >突触后效应取决于受体和离子通道</text>
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

    var frequencyInput=root.querySelector(
      '[data-frequency]'
    );
    var calciumInput=root.querySelector(
      '[data-calcium]'
    );
    var vesicleInput=root.querySelector(
      '[data-vesicle]'
    );
    var excitatoryInputElement=root.querySelector(
      '[data-excitatory]'
    );
    var inhibitoryInputElement=root.querySelector(
      '[data-inhibitory]'
    );
    var receptorInput=root.querySelector(
      '[data-receptor]'
    );
    var clearanceInput=root.querySelector(
      '[data-clearance]'
    );
    var timeInput=root.querySelector(
      '[data-time]'
    );

    var frequencyValue=root.querySelector(
      '[data-frequency-value]'
    );
    var calciumValue=root.querySelector(
      '[data-calcium-value]'
    );
    var vesicleValue=root.querySelector(
      '[data-vesicle-value]'
    );
    var excitatoryValue=root.querySelector(
      '[data-excitatory-value]'
    );
    var inhibitoryValue=root.querySelector(
      '[data-inhibitory-value]'
    );
    var receptorValue=root.querySelector(
      '[data-receptor-value]'
    );
    var clearanceValue=root.querySelector(
      '[data-clearance-value]'
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

    var releaseText=root.querySelector(
      '[data-release-index]'
    );
    var potentialText=root.querySelector(
      '[data-potential]'
    );
    var integrationText=root.querySelector(
      '[data-integration]'
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

    var mode='structure';
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
      },780);
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

    function transmitter(
      x,
      y,
      color,
      opacity,
      scale
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle cx="-5" cy="0" r="6" fill="'+color+'" stroke="#FFFFFF" stroke-width="1.5"/>'
        +'<circle cx="5" cy="0" r="6" fill="'+color+'" stroke="#FFFFFF" stroke-width="1.5"/>'
        +'</g>';
    }

    function phaseAt(progress){
      if(progress<.18){
        return '冲动到达';
      }

      if(progress<.38){
        return 'Ca²⁺内流';
      }

      if(progress<.60){
        return '小泡融合';
      }

      if(progress<.80){
        return '递质作用';
      }

      return '递质清除';
    }

    function renderStructure(){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M65 178 C132 116 222 120 278 176 C305 203 305 245 276 267 C224 307 126 299 70 244 C45 219 44 198 65 178Z" fill="#EDE9FE" stroke="#7C3AED" stroke-width="6"/>'
        +'<path d="M278 176 C306 193 315 213 315 224 C315 238 306 254 276 267" fill="#DDD6FE" stroke="#6D28D9" stroke-width="5"/>'
        +'<rect x="315" y="169" width="68" height="111" rx="18" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<path d="M383 174 C421 145 486 145 521 176 C552 204 552 244 522 272 C487 304 423 304 383 275Z" fill="#DBEAFE" stroke="#2563EB" stroke-width="6"/>'
        +'<path d="M383 174 C365 192 357 208 357 224 C357 241 365 258 383 275" fill="#BFDBFE" stroke="#1D4ED8" stroke-width="5"/>'
        +'</g>'
        +'<text x="166" y="102" text-anchor="middle" font-size="15" font-weight="900" fill="#5B21B6">突触前神经元轴突末梢</text>'
        +'<text x="349" y="314" text-anchor="middle" font-size="14" font-weight="900" fill="#475569">突触间隙</text>'
        +'<text x="472" y="102" text-anchor="middle" font-size="15" font-weight="900" fill="#1D4ED8">突触后膜</text>'
        +'<circle cx="124" cy="189" r="19" fill="#C4B5FD" stroke="#6D28D9" stroke-width="3"/>'
        +'<circle cx="176" cy="164" r="19" fill="#C4B5FD" stroke="#6D28D9" stroke-width="3"/>'
        +'<circle cx="216" cy="214" r="19" fill="#C4B5FD" stroke="#6D28D9" stroke-width="3"/>'
        +'<circle cx="157" cy="243" r="19" fill="#C4B5FD" stroke="#6D28D9" stroke-width="3"/>'
        +'<text x="124" y="193" text-anchor="middle" font-size="8" font-weight="900" fill="#4C1D95">NT</text>'
        +'<text x="176" y="168" text-anchor="middle" font-size="8" font-weight="900" fill="#4C1D95">NT</text>'
        +'<text x="216" y="218" text-anchor="middle" font-size="8" font-weight="900" fill="#4C1D95">NT</text>'
        +'<text x="157" y="247" text-anchor="middle" font-size="8" font-weight="900" fill="#4C1D95">NT</text>'
        +'<rect x="294" y="189" width="18" height="39" rx="7" fill="#60A5FA" stroke="#1D4ED8" stroke-width="3"/>'
        +'<rect x="294" y="238" width="18" height="30" rx="7" fill="#60A5FA" stroke="#1D4ED8" stroke-width="3"/>'
        +'<path d="M403 192 V253" stroke="#2563EB" stroke-width="7" stroke-linecap="round"/>'
        +'<path d="M432 192 V253" stroke="#2563EB" stroke-width="7" stroke-linecap="round"/>'
        +'<path d="M461 192 V253" stroke="#2563EB" stroke-width="7" stroke-linecap="round"/>'
        +'<path class="st-flow" d="M83 224 H270" fill="none" stroke="#7C3AED" stroke-width="6" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<path class="st-flow" d="M292 224 H369" fill="none" stroke="#7C3AED" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<g transform="translate(557 121)">'
        +'<rect width="176" height="185" rx="20" fill="#F5F3FF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="88" y="28" text-anchor="middle" font-size="13" font-weight="900" fill="#5B21B6">化学突触特点</text>'
        +'<text x="17" y="61" font-size="11.5" font-weight="900" fill="#6D28D9">结构不对称</text>'
        +'<text x="17" y="83" font-size="10.5" font-weight="800" fill="#475569">前膜释放，后膜有受体</text>'
        +'<text x="17" y="114" font-size="11.5" font-weight="900" fill="#6D28D9">通常单向传递</text>'
        +'<text x="17" y="136" font-size="10.5" font-weight="800" fill="#475569">前膜 → 间隙 → 后膜</text>'
        +'<text x="17" y="167" font-size="11.5" font-weight="900" fill="#6D28D9">存在突触延搁</text>'
        +'</g>';
    }

    function renderRelease(
      progress,
      calciumSignal,
      releaseSignal,
      clearance,
      frequency
    ){
      var phase=phaseAt(progress);
      var calciumHTML='';
      var vesicleHTML='';
      var transmitterHTML='';
      var receptorHTML='';

      var calciumCount=Math.floor(
        2+calciumSignal/13
      );

      for(var i=0;i<calciumCount;i++){
        calciumHTML+=ion(
          176+(i%4)*24,
          128+Math.floor(i/4)*25,
          'Ca',
          '#2563EB',
          .35+.60*calciumSignal/100,
          8
        );
      }

      var vesicleCount=Math.floor(
        3+releaseSignal/17
      );

      for(var j=0;j<vesicleCount;j++){
        var vx=104+(j%5)*43;
        var vy=198+Math.floor(j/5)*43;
        var active=j<Math.floor(
          releaseSignal/20
        );

        vesicleHTML+=''
          +'<circle cx="'+vx+'" cy="'+vy+'" r="17" fill="'+(active?'#C4B5FD':'#EDE9FE')+'" stroke="#7C3AED" stroke-width="'+(active?4:2.5)+'"/>'
          +'<circle cx="'+(vx-5)+'" cy="'+vy+'" r="3.7" fill="#8B5CF6"/>'
          +'<circle cx="'+(vx+5)+'" cy="'+vy+'" r="3.7" fill="#8B5CF6"/>';
      }

      var transmitterCount=Math.floor(
        2+releaseSignal/9
      );

      var persistence=clamp(
        1-clearance/135
        +frequency/300,
        .22,
        .90
      );

      for(var k=0;k<transmitterCount;k++){
        transmitterHTML+=transmitter(
          323+(k%4)*20,
          185+Math.floor(k/4)*23,
          '#8B5CF6',
          (.30+.60*releaseSignal/100)
          *persistence,
          .72
        );
      }

      var receptorCount=6;

      for(var q=0;q<receptorCount;q++){
        var rx=436+q*35;
        var activeReceptor=q<
          Math.floor(
            receptorCount
            *releaseSignal/100
          );

        receptorHTML+=''
          +'<path d="M'+rx+' 221 V256 M'+(rx+15)+' 221 V256" stroke="'+(activeReceptor?'#16A34A':'#94A3B8')+'" stroke-width="'+(activeReceptor?6:4)+'" stroke-linecap="round"/>'
          +'<path d="M'+rx+' 221 Q'+(rx+7.5)+' 207 '+(rx+15)+' 221" fill="none" stroke="'+(activeReceptor?'#16A34A':'#94A3B8')+'" stroke-width="'+(activeReceptor?5:3)+'"/>';
      }

      return ''
        +'<rect x="27" y="91" width="706" height="273" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<path d="M60 168 C111 102 247 104 292 171 C310 197 310 234 290 260 C246 318 112 318 61 261Z" fill="#EDE9FE" stroke="#7C3AED" stroke-width="6"/>'
        +'<path d="M290 171 C318 192 326 210 326 224 C326 241 316 258 289 261" fill="#DDD6FE" stroke="#6D28D9" stroke-width="5"/>'
        +'<rect x="326" y="168" width="72" height="113" rx="18" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="3"/>'
        +'<path d="M398 171 C445 130 646 130 699 174 L699 279 L398 279Z" fill="#DBEAFE" stroke="#2563EB" stroke-width="6"/>'
        +calciumHTML
        +vesicleHTML
        +transmitterHTML
        +receptorHTML
        +'<path class="st-flow" d="M69 143 C111 109 151 105 175 118" fill="none" stroke="#7C3AED" stroke-width="'+(3+frequency/24)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<path class="st-flow" d="M214 145 V185" fill="none" stroke="#2563EB" stroke-width="'+(3+calciumSignal/25)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="st-flow" d="M282 220 H373" fill="none" stroke="#7C3AED" stroke-width="'+(3+releaseSignal/23)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<text x="174" y="341" text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6">动作电位 → Ca²⁺内流 → 小泡融合</text>'
        +'<text x="527" y="341" text-anchor="middle" font-size="11" font-weight="900" fill="#1D4ED8">递质扩散 → 受体结合 → 离子通道改变</text>'
        +'<g transform="translate(536 104)">'
        +'<rect width="171" height="50" rx="14" fill="#F5F3FF" stroke="#DDD6FE" stroke-width="2"/>'
        +'<text x="85" y="20" text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6">当前阶段：'+phase+'</text>'
        +'<text x="85" y="38" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">清除效率 '+clearance.toFixed(0)+'%</text>'
        +'</g>';
    }

    function potentialCurve(
      type,
      strength,
      receptor,
      clearance
    ){
      var path='';

      for(var i=0;i<=100;i+=2){
        var x=52+i*2.55;
        var t=i/100;
        var rise=1-Math.exp(-t*9);
        var decay=Math.exp(
          -Math.max(0,t-.22)
          *(3+clearance/28)
        );

        var amplitude=strength
          *receptor/100
          *rise
          *decay;

        var value=type==='exc'
          ?-70+amplitude*.19
          :-70-amplitude*.13;

        var y=178-(value+70)*4.1;

        path+=(i===0?'M':' L')
          +x.toFixed(1)
          +' '
          +y.toFixed(1);
      }

      return path;
    }

    function renderPotentials(
      excitatory,
      inhibitory,
      receptor,
      clearance,
      epsp,
      ipsp
    ){
      var excWidth=clamp(
        epsp*4.2,
        0,
        160
      );
      var inhWidth=clamp(
        ipsp*4.2,
        0,
        160
      );

      return ''
        +'<g transform="translate(28 91)">'
        +'<rect width="339" height="271" rx="23" fill="#ECFDF5" stroke="#A7F3D0" stroke-width="3"/>'
        +'<text x="169" y="29" text-anchor="middle" font-size="15" font-weight="900" fill="#047857">兴奋性突触后电位 EPSP</text>'
        +'<line x1="52" y1="178" x2="307" y2="178" stroke="#94A3B8" stroke-width="2"/>'
        +'<line x1="52" y1="62" x2="52" y2="222" stroke="#94A3B8" stroke-width="2"/>'
        +'<line x1="52" y1="116" x2="307" y2="116" stroke="#F59E0B" stroke-width="2" stroke-dasharray="7 6"/>'
        +'<text x="8" y="182" font-size="10" font-weight="900" fill="#475569">-70</text>'
        +'<text x="8" y="120" font-size="10" font-weight="900" fill="#B45309">阈值</text>'
        +'<path d="'+potentialCurve('exc',excitatory,receptor,clearance)+'" fill="none" stroke="#16A34A" stroke-width="5" stroke-linecap="round"/>'
        +'<text x="169" y="245" text-anchor="middle" font-size="11" font-weight="900" fill="#166534">局部去极化，使膜电位接近阈值</text>'
        +'</g>'
        +'<g transform="translate(393 91)">'
        +'<rect width="339" height="271" rx="23" fill="#EFF6FF" stroke="#BFDBFE" stroke-width="3"/>'
        +'<text x="169" y="29" text-anchor="middle" font-size="15" font-weight="900" fill="#1D4ED8">抑制性突触后电位 IPSP</text>'
        +'<line x1="52" y1="178" x2="307" y2="178" stroke="#94A3B8" stroke-width="2"/>'
        +'<line x1="52" y1="62" x2="52" y2="222" stroke="#94A3B8" stroke-width="2"/>'
        +'<line x1="52" y1="116" x2="307" y2="116" stroke="#F59E0B" stroke-width="2" stroke-dasharray="7 6"/>'
        +'<text x="8" y="182" font-size="10" font-weight="900" fill="#475569">-70</text>'
        +'<text x="8" y="120" font-size="10" font-weight="900" fill="#B45309">阈值</text>'
        +'<path d="'+potentialCurve('inh',inhibitory,receptor,clearance)+'" fill="none" stroke="#2563EB" stroke-width="5" stroke-linecap="round"/>'
        +'<text x="169" y="245" text-anchor="middle" font-size="11" font-weight="900" fill="#1E40AF">局部超极化或分流，使兴奋作用减弱</text>'
        +'</g>'
        +'<g transform="translate(111 330)">'
        +'<rect width="538" height="34" rx="16" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<rect x="18" y="10" width="'+excWidth.toFixed(1)+'" height="14" rx="7" fill="#16A34A"/>'
        +'<rect x="'+(520-inhWidth).toFixed(1)+'" y="10" width="'+inhWidth.toFixed(1)+'" height="14" rx="7" fill="#2563EB"/>'
        +'<text x="269" y="22" text-anchor="middle" font-size="10.5" font-weight="900" fill="#475569">EPSP和IPSP都是可总和的局部渐变电位</text>'
        +'</g>';
    }

    function inputBranch(
      x,
      y,
      color,
      label,
      strength,
      direction
    ){
      var width=3+strength/24;
      var endX=direction==='left'
        ?x-98
        :x+98;

      return ''
        +'<path class="st-flow" d="M'+endX+' '+(y-44)+' Q'+x+' '+(y-55)+' '+x+' '+y+'" fill="none" stroke="'+color+'" stroke-width="'+width+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<circle cx="'+endX+'" cy="'+(y-44)+'" r="'+(16+strength/18)+'" fill="'+color+'" opacity="'+(.35+.55*strength/100)+'"/>'
        +'<text x="'+endX+'" y="'+(y-40)+'" text-anchor="middle" font-size="10" font-weight="900" fill="#FFFFFF">'+label+'</text>';
    }

    function renderSummation(
      excitatory,
      inhibitory,
      frequency,
      netPotential,
      thresholdReached,
      temporal,
      spatial
    ){
      var somaColor=thresholdReached
        ?'#FEE2E2'
        :'#EDE9FE';

      var somaStroke=thresholdReached
        ?'#DC2626'
        :'#7C3AED';

      var barX=clamp(
        75+(netPotential+85)/40*520,
        75,
        595
      );

      return ''
        +'<rect x="27" y="91" width="706" height="274" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<path d="M350 252 C316 229 301 193 315 160 C332 119 381 101 423 120 C459 136 480 178 466 214 C453 250 394 276 350 252Z" fill="'+somaColor+'" stroke="'+somaStroke+'" stroke-width="6"/>'
        +'<path d="M389 250 C392 290 432 310 489 318" fill="none" stroke="#7C3AED" stroke-width="14" stroke-linecap="round"/>'
        +'<path d="M489 318 H631" fill="none" stroke="#A78BFA" stroke-width="14" stroke-linecap="round"/>'
        +inputBranch(333,155,'#16A34A','E1',excitatory,'left')
        +inputBranch(428,155,'#16A34A','E2',spatial,'right')
        +inputBranch(321,230,'#2563EB','I1',inhibitory,'left')
        +inputBranch(465,228,'#2563EB','I2',inhibitory*.75,'right')
        +'<circle cx="478" cy="312" r="25" fill="'+(thresholdReached?'#EF4444':'#C4B5FD')+'" stroke="'+(thresholdReached?'#B91C1C':'#6D28D9')+'" stroke-width="5"/>'
        +'<text x="478" y="317" text-anchor="middle" font-size="10" font-weight="900" fill="#FFFFFF">轴丘</text>'
        +'<path class="st-flow" d="M505 318 H627" fill="none" stroke="'+(thresholdReached?'#DC2626':'#94A3B8')+'" stroke-width="'+(thresholdReached?7:4)+'" marker-end="url(#${rootId}-arrow-red)"/>'
        +'<g transform="translate(51 112)">'
        +'<rect width="193" height="180" rx="20" fill="#FFFFFF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="96" y="27" text-anchor="middle" font-size="13" font-weight="900" fill="#5B21B6">突触后电位总和</text>'
        +'<text x="17" y="61" font-size="11" font-weight="900" fill="#047857">时间总和</text>'
        +'<text x="17" y="82" font-size="10" font-weight="800" fill="#475569">同一突触连续活动</text>'
        +'<rect x="17" y="92" width="158" height="13" rx="6" fill="#D1FAE5"/>'
        +'<rect x="17" y="92" width="'+(158*temporal/100).toFixed(1)+'" height="13" rx="6" fill="#16A34A"/>'
        +'<text x="17" y="132" font-size="11" font-weight="900" fill="#6D28D9">空间总和</text>'
        +'<text x="17" y="153" font-size="10" font-weight="800" fill="#475569">多个突触同时或近同时活动</text>'
        +'<rect x="17" y="162" width="158" height="13" rx="6" fill="#EDE9FE"/>'
        +'<rect x="17" y="162" width="'+(158*spatial/100).toFixed(1)+'" height="13" rx="6" fill="#7C3AED"/>'
        +'</g>'
        +'<g transform="translate(76 322)">'
        +'<rect width="520" height="20" rx="10" fill="#E2E8F0"/>'
        +'<rect x="0" y="0" width="'+Math.max(0,barX-75).toFixed(1)+'" height="20" rx="10" fill="'+(thresholdReached?'#EF4444':'#8B5CF6')+'"/>'
        +'<line x1="390" y1="-7" x2="390" y2="27" stroke="#F59E0B" stroke-width="4"/>'
        +'<text x="390" y="-12" text-anchor="middle" font-size="10" font-weight="900" fill="#B45309">动作电位阈值</text>'
        +'<text x="532" y="15" font-size="11" font-weight="900" fill="#334155">'+netPotential.toFixed(0)+'mV</text>'
        +'</g>'
        +'<text x="592" y="286" text-anchor="middle" font-size="11" font-weight="900" fill="'+(thresholdReached?'#B91C1C':'#475569')+'">'+(thresholdReached?'净效应达到阈值，产生动作电位':'净效应未达阈值，不产生动作电位')+'</text>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='structure'){
        labels.innerHTML=''
          +'<path d="M191 150 L191 104" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="139" y="98" font-size="13" font-weight="900" fill="#5B21B6">突触小泡</text>'
          +'<path d="M348 170 L348 127" stroke="#64748B" stroke-width="2.5"/>'
          +'<text x="302" y="120" font-size="13" font-weight="900" fill="#475569">突触间隙</text>'
          +'<path d="M431 192 L461 145" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="468" y="141" font-size="13" font-weight="900" fill="#1D4ED8">突触后受体</text>';
        return;
      }

      if(modeName==='release'){
        labels.innerHTML=''
          +'<path d="M206 147 L205 104" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="145" y="98" font-size="13" font-weight="900" fill="#1D4ED8">Ca²⁺内流</text>'
          +'<path d="M279 220 L303 172" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="310" y="168" font-size="13" font-weight="900" fill="#5B21B6">小泡胞吐</text>'
          +'<path d="M356 216 L399 171" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="406" y="167" font-size="13" font-weight="900" fill="#5B21B6">神经递质</text>';
        return;
      }

      if(modeName==='potentials'){
        labels.innerHTML=''
          +'<path d="M191 124 L191 88" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="132" y="82" font-size="13" font-weight="900" fill="#047857">局部去极化</text>'
          +'<path d="M557 222 L557 185" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="498" y="179" font-size="13" font-weight="900" fill="#1D4ED8">局部超极化</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M374 123 L374 83" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="321" y="77" font-size="13" font-weight="900" fill="#5B21B6">胞体整合区</text>'
        +'<path d="M478 286 L527 251" stroke="#DC2626" stroke-width="2.5"/>'
        +'<text x="534" y="247" font-size="13" font-weight="900" fill="#B91C1C">轴丘阈值判断</text>'
        +'<path d="M608 318 L648 285" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="655" y="281" font-size="13" font-weight="900" fill="#5B21B6">轴突</text>';
    }

    function update(){
      var frequency=Number(
        frequencyInput.value
      );
      var calcium=Number(
        calciumInput.value
      );
      var vesicle=Number(
        vesicleInput.value
      );
      var excitatory=Number(
        excitatoryInputElement.value
      );
      var inhibitory=Number(
        inhibitoryInputElement.value
      );
      var receptor=Number(
        receptorInput.value
      );
      var clearance=Number(
        clearanceInput.value
      );
      var processTime=Number(
        timeInput.value
      );

      frequencyValue.textContent=
        frequency.toFixed(0)+'%';
      calciumValue.textContent=
        calcium.toFixed(0)+'%';
      vesicleValue.textContent=
        vesicle.toFixed(0)+'%';
      excitatoryValue.textContent=
        excitatory.toFixed(0)+'%';
      inhibitoryValue.textContent=
        inhibitory.toFixed(0)+'%';
      receptorValue.textContent=
        receptor.toFixed(0)+'%';
      clearanceValue.textContent=
        clearance.toFixed(0)+'%';
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
        ?'突触推进：运行中'
        :'突触推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var progress=processTime/100;

      var arrival=clamp(
        (progress-.05)/.22,
        0,
        1
      );

      var calciumSignal=100*Math.sqrt(
        calcium/100
        *(.18+.82*arrival)
      );

      calciumSignal=clamp(
        calciumSignal,
        0,
        100
      );

      var releaseSignal=100*Math.sqrt(
        calciumSignal/100
        *vesicle/100
      );

      releaseSignal*=(
        .42+.58*frequency/100
      );

      releaseSignal=clamp(
        releaseSignal,
        0,
        100
      );

      var activeTransmitter=releaseSignal
        *(
          1-clearance/180
        );

      activeTransmitter=clamp(
        activeTransmitter,
        0,
        100
      );

      var epsp=activeTransmitter
        *excitatory/100
        *receptor/100;

      var ipsp=activeTransmitter
        *inhibitory/100
        *receptor/100;

      var temporalSummation=clamp(
        epsp
        *(.34+.66*frequency/100),
        0,
        100
      );

      var spatialSummation=clamp(
        excitatory*.62
        +inhibitory*.22,
        0,
        100
      );

      var excitatoryDrive=epsp
        *(.48+.52*frequency/100);

      var inhibitoryDrive=ipsp
        *(.64+.36*frequency/100);

      var netPotential=clamp(
        -70
        +excitatoryDrive*.34
        -inhibitoryDrive*.30,
        -85,
        -42
      );

      var thresholdReached=
        netPotential>=-55;

      releaseText.textContent=
        releaseSignal.toFixed(0);
      potentialText.textContent=
        netPotential.toFixed(0)+'mV';
      integrationText.textContent=
        thresholdReached?'已达阈值':'未达阈值';

      root.style.setProperty(
        '--st-speed',
        clamp(
          2.45-Math.max(
            frequency,
            releaseSignal
          )/72,
          .58,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='structure'){
        title.textContent=
          '化学突触的结构与单向传递';

        summary.textContent=
          '观察突触前末梢、突触间隙和突触后膜的结构不对称性。';

        dynamic.innerHTML=renderStructure();

        stageNote.textContent=
          '突触前结构负责释放神经递质，突触后膜表达相应受体，因此化学突触通常单向传递。';

        renderLabels(mode);
      }else if(mode==='release'){
        title.textContent=
          '动作电位触发神经递质释放';

        summary.textContent=
          '观察Ca²⁺内流、小泡融合、递质扩散、受体结合和递质清除。';

        dynamic.innerHTML=renderRelease(
          progress,
          calciumSignal,
          releaseSignal,
          clearance,
          frequency
        );

        stageNote.textContent=
          '动作电位到达后，电压门控钙通道开放；Ca²⁺内流促进突触小泡胞吐。';

        renderLabels(mode);
      }else if(mode==='potentials'){
        title.textContent=
          '兴奋性与抑制性突触后电位';

        summary.textContent=
          '比较EPSP局部去极化与IPSP局部超极化或分流性抑制。';

        dynamic.innerHTML=renderPotentials(
          excitatory,
          inhibitory,
          receptor,
          clearance,
          epsp,
          ipsp
        );

        stageNote.textContent=
          'EPSP和IPSP都是可以发生时间总和、空间总和的局部渐变电位。';

        renderLabels(mode);
      }else{
        title.textContent=
          '时间总和、空间总和与轴丘整合';

        summary.textContent=
          '观察多个兴奋性和抑制性输入如何共同决定是否达到动作电位阈值。';

        dynamic.innerHTML=renderSummation(
          excitatory,
          inhibitory,
          frequency,
          netPotential,
          thresholdReached,
          temporalSummation,
          spatialSummation
        );

        stageNote.textContent=thresholdReached
          ?'兴奋性和抑制性输入整合后的净效应达到阈值，轴丘可产生动作电位。'
          :'虽然存在兴奋性输入，但净效应没有达到阈值，轴丘不产生动作电位。';

        renderLabels(mode);
      }

      var condition=
        '当前冲动频率、钙通道、小泡释放、受体和递质清除相对协调。';

      if(frequency<12){
        condition=
          '突触前冲动频率较低，单位时间内释放事件较少，时间总和较弱。';
      }else if(calcium<35){
        condition=
          '钙通道相对活性较低，动作电位到达后Ca²⁺内流和小泡融合受到限制。';
      }else if(vesicle<35){
        condition=
          '突触小泡释放能力较低，即使Ca²⁺信号存在，递质释放量仍受到限制。';
      }else if(receptor<35){
        condition=
          '突触后受体敏感性较低，相同递质释放产生的突触后效应较弱。';
      }else if(clearance<30){
        condition=
          '递质清除效率较低，递质在突触间隙中的作用时间相对延长。';
      }else if(inhibitory>excitatory+25){
        condition=
          '抑制性输入明显强于兴奋性输入，突触后膜净效应远离动作电位阈值。';
      }else if(excitatory>inhibitory+35){
        condition=
          '兴奋性输入较强，但是否产生动作电位仍取决于总和后能否达到轴丘阈值。';
      }else if(processTime<14){
        condition=
          '突触过程时间较短，动作电位刚到达或钙通道尚未充分开放。';
      }

      var principle=mode==='structure'
        ?'化学突触由突触前膜、突触间隙和突触后膜构成，结构不对称使信息通常单向传递。'
        :mode==='release'
          ?'动作电位引起Ca²⁺内流，Ca²⁺促进小泡胞吐；递质与受体结合后改变突触后膜离子通道状态。'
          :mode==='potentials'
            ?'EPSP使膜电位接近阈值，IPSP使膜电位远离阈值或削弱兴奋性输入；两者均为局部渐变电位。'
            :'神经元对多个兴奋性和抑制性输入进行时间和空间总和，净效应达到阈值后才产生动作电位。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前递质释放指数 '
        +releaseSignal.toFixed(0)
        +'，膜电位示意 '
        +netPotential.toFixed(0)
        +' mV；突触后效应取决于受体、离子通道和离子梯度，不能只根据递质名称判断。';
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

    frequencyInput.oninput=update;
    calciumInput.oninput=update;
    vesicleInput.oninput=update;
    excitatoryInputElement.oninput=update;
    inhibitoryInputElement.oninput=update;
    receptorInput.oninput=update;
    clearanceInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
