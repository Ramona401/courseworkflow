/**
 * lifeScienceLabTemplatesRegulationMotorFeedback.ts
 *
 * 平面生命科学实验室：
 * 随意运动、反射运动与反馈校正。
 *
 * 教学目标：
 * 1. 区分随意运动与反射运动的基本控制特点；
 * 2. 理解随意运动通常由运动意图、运动计划和下行运动指令启动；
 * 3. 理解随意运动并不是由一个孤立脑区独立完成，
 *    而是涉及大脑皮层、基底神经节、小脑、脑干、
 *    脊髓运动回路和肌肉等多个层级协同；
 * 4. 理解反射运动由感受器、传入神经、反射中枢、
 *    传出神经和效应器构成基本反射通路；
 * 5. 理解反射活动通常不需要意识先作出决定，
 *    但反射强度和表现可受到高级中枢调节；
 * 6. 理解随意运动和反射运动并不是完全割裂的两套系统，
 *    实际运动中两者可共同参与；
 * 7. 理解骨骼肌运动常需要主动肌、拮抗肌和稳定肌协调；
 * 8. 理解主动肌兴奋时拮抗肌活动通常需要协调性降低，
 *    但真实运动中也可能出现共同收缩以增强关节稳定；
 * 9. 理解运动指令发出后，
 *    肌梭、腱器官、关节感受器、视觉和前庭等反馈
 *    可报告身体与环境的实际状态；
 * 10. 理解中枢可比较目标状态和实际状态，
 *     根据运动误差形成校正指令；
 * 11. 理解前馈控制可根据已有运动计划提前发出指令，
 *     反馈控制则根据运动后果持续修正误差；
 * 12. 理解外界扰动、肌肉疲劳、通路状态和反馈质量
 *     都会影响运动准确性与稳定性。
 *
 * 科学边界：
 * 1. 随意运动的产生涉及多个神经环路协同，
 *    不能简单等同于“运动皮层单独控制肌肉”；
 * 2. 反射运动通常较快，
 *    但“反射”不等于所有情况下都绝对不受意识和高级中枢影响；
 * 3. 脊髓和脑干可完成多种反射整合，
 *    大脑和其他高级中枢仍可接收相关信息并调节后续行为；
 * 4. 运动神经元是中枢神经系统控制骨骼肌的重要最终输出通路，
 *    但肌肉力量还受运动单位募集、放电频率和肌肉状态影响；
 * 5. 主动肌与拮抗肌之间的交互在本模板中采用简化模型；
 * 6. 交互抑制有助于完成方向明确的运动，
 *    但精细动作和维持姿势时也可能出现适度共同收缩；
 * 7. 本体感觉反馈包括肌肉长度、张力和关节状态等信息，
 *    视觉、前庭和皮肤感觉也可参与运动校正；
 * 8. 小脑参与运动协调、时序和误差校正，
 *    但反馈校正不应被理解为小脑单独完成；
 * 9. 快速反馈校正可包括脊髓反射、脑干回路和较长反馈环路；
 * 10. 前馈与反馈控制通常共同作用，
 *     不能把真实运动严格划分为只有前馈或只有反馈；
 * 11. 图中的反应延迟、肌肉激活、运动角度、
 *     误差、准确度和校正强度均为相对教学指标；
 * 12. 本模板不模拟具体神经损伤、肌肉疾病、
 *     运动障碍、康复训练或临床检查；
 * 13. 本模板只用于生物学课堂教学，
 *     不用于神经功能、运动能力或医学诊断。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部图片、脚本、字体或CDN；
 * 3. 所有DOM查询均限定在rootId内部；
 * 4. 支持同一课件页面插入多个独立实例；
 * 5. 使用生命科学统一.bl-*布局协议；
 * 6. 支持随意运动、反射运动和协同控制三种控制类型；
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
 * 将数字整理为适合写入HTML属性的简洁文本。
 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/**
 * 模板独立样式。
 *
 * 所有选择器均以rootId作为前缀，
 * 防止同一课件中的多个模板实例互相影响。
 */
function motorFeedbackStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FED7AA;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FFEDD5,#FFF7ED);border-bottom:1px solid #FED7AA}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9A3412}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFFCF8;border-right:1px solid #FED7AA}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#EA580C;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#EA580C}'
    + '#' + rootId + ' .mf-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#9A3412}'
    + '#' + rootId + ' .mf-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .mf-control-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .mf-button{min-height:30px;padding:3px;border:1px solid #FDBA74;border-radius:8px;background:#fff;color:#9A3412;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .mf-button.active{border-color:#EA580C;background:#FFEDD5;box-shadow:0 3px 9px rgba(234,88,12,.14)}'
    + '#' + rootId + ' .mf-control-button{min-height:29px;padding:3px;border:1px solid #FCD34D;border-radius:8px;background:#fff;color:#92400E;font-size:9.6px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .mf-control-button.active{border-color:#D97706;background:#FEF3C7;box-shadow:0 3px 9px rgba(217,119,6,.14)}'
    + '#' + rootId + ' .mf-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#FB923C,#EA580C);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .mf-toggle.off{background:#64748B}'
    + '#' + rootId + ' .mf-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .mf-card{padding:6px 3px;border:1px solid #FED7AA;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .mf-card b{display:block;font-size:13px;color:#C2410C}'
    + '#' + rootId + ' .mf-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FFEDD5;color:#7C2D12;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .mf-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--mf-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .mf-pulse{animation:' + rootId + '-pulse 1.45s ease-in-out infinite}'
    + '#' + rootId + ' .mf-error{animation:' + rootId + '-error 1.15s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.30}50%{opacity:1}}'
    + '@keyframes ' + rootId + '-error{0%,100%{opacity:.35}50%{opacity:.96}}'
    + '</style>'
}

/**
 * 避免模板源码中直接出现闭合script标签。
 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_MOTOR_FEEDBACK:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-voluntary-reflex-feedback-control',
    group: '🧠 稳态与调节',
    name: '随意运动、反射运动与反馈校正',
    emoji: '🎯',
    desc: '切换随意运动、反射运动和协同控制，调节运动指令、感觉输入、通路、肌肉、扰动、疲劳和反馈增益，观察运动控制与误差校正',
    params: [
      {
        key: 'motorCommand',
        label: '运动指令强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
        hint: '表示随意运动计划形成的相对下行运动指令',
      },
      {
        key: 'sensoryInput',
        label: '感觉反馈强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 70,
        hint: '表示本体感觉、视觉等传回中枢的相对信息强度',
      },
      {
        key: 'pathwayIntegrity',
        label: '运动通路完整度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 88,
        hint: '用于比较运动指令和反射信号在通路中的相对保留程度',
      },
      {
        key: 'muscleCapacity',
        label: '肌肉响应能力',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 84,
        hint: '表示运动单位募集和肌肉产生力量的相对能力',
      },
      {
        key: 'externalDisturbance',
        label: '外界扰动强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 26,
        hint: '表示负重、推力或环境变化对目标运动的干扰',
      },
      {
        key: 'feedbackGain',
        label: '反馈校正增益',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 74,
        hint: '表示检测到误差后形成校正指令的相对强度',
      },
      {
        key: 'fatigueLevel',
        label: '肌肉疲劳程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 18,
        hint: '疲劳升高会降低肌肉响应和动作稳定性',
      },
      {
        key: 'processTime',
        label: '运动过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 38,
        hint: '控制运动指令、肌肉收缩和反馈校正过程的演示进度',
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const motorCommand = num(
        params,
        'motorCommand',
        72,
      )
      const sensoryInput = num(
        params,
        'sensoryInput',
        70,
      )
      const pathwayIntegrity = num(
        params,
        'pathwayIntegrity',
        88,
      )
      const muscleCapacity = num(
        params,
        'muscleCapacity',
        84,
      )
      const externalDisturbance = num(
        params,
        'externalDisturbance',
        26,
      )
      const feedbackGain = num(
        params,
        'feedbackGain',
        74,
      )
      const fatigueLevel = num(
        params,
        'fatigueLevel',
        18,
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
${motorFeedbackStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🎯 随意运动、反射运动与反馈校正</div>
    <div class="bl-note">前馈计划、反射调节和感觉反馈共同支持准确运动</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>运动指令强度</span>
          <span class="bl-value" data-command-value></span>
        </div>
        <input
          data-command
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(motorCommand)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>感觉反馈强度</span>
          <span class="bl-value" data-sensory-value></span>
        </div>
        <input
          data-sensory
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(sensoryInput)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>运动通路完整度</span>
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
          <span>肌肉响应能力</span>
          <span class="bl-value" data-muscle-value></span>
        </div>
        <input
          data-muscle
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(muscleCapacity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>外界扰动强度</span>
          <span class="bl-value" data-disturbance-value></span>
        </div>
        <input
          data-disturbance
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(externalDisturbance)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>反馈校正增益</span>
          <span class="bl-value" data-feedback-value></span>
        </div>
        <input
          data-feedback
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(feedbackGain)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>肌肉疲劳程度</span>
          <span class="bl-value" data-fatigue-value></span>
        </div>
        <input
          data-fatigue
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(fatigueLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>运动过程时间</span>
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

      <div class="mf-subtitle">选择控制类型</div>

      <div class="mf-control-buttons">
        <button
          type="button"
          class="mf-control-button active"
          data-movement="voluntary"
        >随意运动</button>

        <button
          type="button"
          class="mf-control-button"
          data-movement="reflex"
        >反射运动</button>

        <button
          type="button"
          class="mf-control-button"
          data-movement="combined"
        >协同控制</button>
      </div>

      <div class="mf-subtitle">观察方式</div>

      <div class="mf-buttons">
        <button
          type="button"
          class="mf-button active"
          data-mode="routes"
        >随意与反射通路</button>

        <button
          type="button"
          class="mf-button"
          data-mode="muscles"
        >主动肌与拮抗肌</button>

        <button
          type="button"
          class="mf-button"
          data-mode="feedback"
        >反馈环路与误差校正</button>

        <button
          type="button"
          class="mf-button"
          data-mode="comparison"
        >有无反馈轨迹比较</button>
      </div>

      <button
        type="button"
        class="mf-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="mf-toggle"
        data-auto
      >运动推进：运行中</button>

      <div class="mf-status">
        <div class="mf-card">
          <b data-delay></b>
          <span>相对反应延迟</span>
        </div>

        <div class="mf-card">
          <b data-accuracy></b>
          <span>运动准确度</span>
        </div>

        <div class="mf-card">
          <b data-error></b>
          <span>剩余误差</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="随意运动、反射运动与反馈校正互动示意图"
      >
        <defs>
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
              flood-color="#7C2D12"
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
          fill="#9A3412"
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
            fill="#FFF7ED"
            stroke="#FED7AA"
            stroke-width="2"
          />

          <text
            x="110"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#9A3412"
          >关键边界</text>

          <text
            x="110"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#7C2D12"
          >随意运动与反射运动可共同参与</text>

          <text
            x="110"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#7C2D12"
          >反馈校正不是由单一脑区独立完成</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#9A3412"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var commandInput=root.querySelector(
      '[data-command]'
    );
    var sensoryInputElement=root.querySelector(
      '[data-sensory]'
    );
    var pathwayInput=root.querySelector(
      '[data-pathway]'
    );
    var muscleInput=root.querySelector(
      '[data-muscle]'
    );
    var disturbanceInput=root.querySelector(
      '[data-disturbance]'
    );
    var feedbackInput=root.querySelector(
      '[data-feedback]'
    );
    var fatigueInput=root.querySelector(
      '[data-fatigue]'
    );
    var timeInput=root.querySelector(
      '[data-time]'
    );

    var commandValue=root.querySelector(
      '[data-command-value]'
    );
    var sensoryValue=root.querySelector(
      '[data-sensory-value]'
    );
    var pathwayValue=root.querySelector(
      '[data-pathway-value]'
    );
    var muscleValue=root.querySelector(
      '[data-muscle-value]'
    );
    var disturbanceValue=root.querySelector(
      '[data-disturbance-value]'
    );
    var feedbackValue=root.querySelector(
      '[data-feedback-value]'
    );
    var fatigueValue=root.querySelector(
      '[data-fatigue-value]'
    );
    var timeValue=root.querySelector(
      '[data-time-value]'
    );

    var movementButtons=root.querySelectorAll(
      '[data-movement]'
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

    var delayText=root.querySelector(
      '[data-delay]'
    );
    var accuracyText=root.querySelector(
      '[data-accuracy]'
    );
    var errorText=root.querySelector(
      '[data-error]'
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

    var mode='routes';
    var movementType='voluntary';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    var movementInformation={
      voluntary:{
        name:'随意运动',
        color:'#EA580C',
        pale:'#FFEDD5',
        stroke:'#C2410C',
        source:'运动意图和运动计划',
        route:'高级中枢下行指令',
        feature:'由目标和计划启动，可持续接受感觉反馈校正。'
      },
      reflex:{
        name:'反射运动',
        color:'#2563EB',
        pale:'#DBEAFE',
        stroke:'#1D4ED8',
        source:'感受器接受刺激',
        route:'传入—反射中枢—传出',
        feature:'不需要意识先作决定，通常较快，也可受高级中枢调节。'
      },
      combined:{
        name:'协同控制',
        color:'#16A34A',
        pale:'#DCFCE7',
        stroke:'#15803D',
        source:'运动计划与感觉输入',
        route:'下行指令、反射回路与反馈环',
        feature:'随意计划、反射调节和反馈校正常在同一运动中共同参与。'
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
          2+strength/15
        )
      );

      for(var i=0;i<count;i++){
        var offset=i/count;
        var position=(
          progress+offset
        )%1;
        var x=startX
          +(endX-startX)*position;

        html+=''
          +'<circle cx="'+x.toFixed(1)+'" cy="'+y+'" r="'+(3.5+strength/40).toFixed(1)+'" fill="'+color+'" opacity="'+(.30+.62*strength/100).toFixed(2)+'"/>';
      }

      return html;
    }

    function brainIcon(
      x,
      y,
      scale,
      color,
      active
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+(active?.98:.48)+'">'
        +'<path d="M-55 -4 C-64 -33 -41 -56 -14 -52 C3 -68 34 -61 42 -41 C63 -35 68 -7 55 9 C62 33 40 54 17 50 C0 65 -29 57 -36 38 C-58 34 -66 12 -55 -4Z" fill="#FFF7ED" stroke="'+color+'" stroke-width="'+(active?6:3)+'"/>'
        +'<path d="M-31 -19 C-9 -34 17 -33 36 -13 M-36 7 C-9 -6 17 -3 39 14 M-17 -41 C-6 -23 -4 2 -11 30 M17 -43 C8 -22 8 2 18 31" fill="none" stroke="'+color+'" stroke-width="4" stroke-linecap="round"/>'
        +'</g>';
    }

    function spinalIcon(
      x,
      y,
      scale,
      color,
      active
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+(active?.98:.50)+'">'
        +'<rect x="-30" y="-58" width="60" height="116" rx="26" fill="#F5F3FF" stroke="'+color+'" stroke-width="'+(active?6:3)+'"/>'
        +'<path d="M0 -43 V43" stroke="'+color+'" stroke-width="12" stroke-linecap="round"/>'
        +'<path d="M-20 -27 Q0 -8 20 -27 M-20 2 Q0 21 20 2 M-20 31 Q0 50 20 31" fill="none" stroke="#C4B5FD" stroke-width="7" stroke-linecap="round"/>'
        +'</g>';
    }

    function muscleIcon(
      x,
      y,
      scale,
      color,
      active
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+(active?.98:.48)+'">'
        +'<ellipse cx="0" cy="0" rx="53" ry="30" fill="#FECACA" stroke="'+color+'" stroke-width="'+(active?6:3)+'"/>'
        +'<path d="M-42 0 Q0 -21 42 0 Q0 21 -42 0Z" fill="'+color+'" opacity=".52"/>'
        +'<path d="M-57 0 H-78 M57 0 H78" stroke="#94A3B8" stroke-width="8" stroke-linecap="round"/>'
        +'</g>';
    }

    function routeCard(
      x,
      y,
      width,
      height,
      titleText,
      color,
      pale,
      active
    ){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="'+x+'" y="'+y+'" width="'+width+'" height="'+height+'" rx="22" fill="'+pale+'" stroke="'+color+'" stroke-width="'+(active?5:3)+'" opacity="'+(active?1:.62)+'"/>'
        +'</g>'
        +'<text x="'+(x+width/2)+'" y="'+(y+26)+'" text-anchor="middle" font-size="13" font-weight="900" fill="'+color+'">'+titleText+'</text>';
    }

    function renderRoutes(
      info,
      progress,
      commandDrive,
      reflexDrive,
      feedbackDrive
    ){
      var voluntaryActive=
        movementType==='voluntary'
        ||movementType==='combined';

      var reflexActive=
        movementType==='reflex'
        ||movementType==='combined';

      var p1=stageProgress(
        progress,
        0,
        .28
      );
      var p2=stageProgress(
        progress,
        .20,
        .58
      );
      var p3=stageProgress(
        progress,
        .50,
        .84
      );
      var p4=stageProgress(
        progress,
        .72,
        1
      );

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +routeCard(
          48,
          112,
          314,
          200,
          '随意运动：目标与运动计划启动',
          '#EA580C',
          '#FFF7ED',
          voluntaryActive
        )
        +routeCard(
          398,
          112,
          314,
          200,
          '反射运动：感觉刺激启动',
          '#2563EB',
          '#EFF6FF',
          reflexActive
        )
        +brainIcon(
          103,
          193,
          .63,
          '#EA580C',
          voluntaryActive&&p1>.18
        )
        +'<text x="103" y="267" text-anchor="middle" font-size="10.5" font-weight="900" fill="#9A3412">运动计划与下行指令</text>'
        +'<path d="M163 193 H237" fill="none" stroke="#FDBA74" stroke-width="9" stroke-linecap="round"/>'
        +'<path class="mf-flow" d="M163 193 H237" fill="none" stroke="#EA580C" stroke-width="'+(3+commandDrive/22)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +signalDots(166,231,193,commandDrive,p2,'#EA580C')
        +spinalIcon(
          271,
          193,
          .55,
          '#7C3AED',
          voluntaryActive&&p2>.22
        )
        +'<path class="mf-flow" d="M307 193 H340" fill="none" stroke="#EA580C" stroke-width="'+(3+commandDrive/24)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +muscleIcon(
          332,
          257,
          .43,
          '#DC2626',
          voluntaryActive&&p3>.24
        )
        +'<text x="205" y="292" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">高级中枢—下行通路—脊髓运动神经元—肌肉</text>'
        +'<g transform="translate(447 191)">'
        +'<circle class="'+(reflexActive?'mf-pulse':'')+'" cx="0" cy="0" r="39" fill="#DBEAFE" stroke="#2563EB" stroke-width="'+(reflexActive?6:3)+'"/>'
        +'<path d="M-20 -5 Q0 -27 20 -5 Q0 19 -20 -5Z" fill="#60A5FA"/>'
        +'<path d="M0 22 V48" stroke="#2563EB" stroke-width="9" stroke-linecap="round"/>'
        +'</g>'
        +'<text x="447" y="267" text-anchor="middle" font-size="10.5" font-weight="900" fill="#1D4ED8">感受器与传入神经</text>'
        +'<path d="M492 193 H548" fill="none" stroke="#BFDBFE" stroke-width="9" stroke-linecap="round"/>'
        +'<path class="mf-flow" d="M492 193 H548" fill="none" stroke="#2563EB" stroke-width="'+(3+reflexDrive/22)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +signalDots(495,543,193,reflexDrive,p1,'#2563EB')
        +spinalIcon(
          579,
          193,
          .55,
          '#4F46E5',
          reflexActive&&p2>.18
        )
        +'<path class="mf-flow" d="M612 193 H680" fill="none" stroke="#2563EB" stroke-width="'+(3+reflexDrive/24)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +muscleIcon(
          662,
          257,
          .43,
          '#DC2626',
          reflexActive&&p3>.20
        )
        +'<text x="555" y="292" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">感受器—传入神经—反射中枢—传出神经—效应器</text>'
        +'<path class="mf-flow" d="M658 289 C623 329 509 332 455 299" fill="none" stroke="#16A34A" stroke-width="'+(2+feedbackDrive/28)+'" stroke-dasharray="7 6" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<text x="554" y="334" text-anchor="middle" font-size="9.5" font-weight="900" fill="#15803D">运动结果和本体感觉可继续反馈至中枢</text>'
        +'<g transform="translate(58 329)">'
        +'<rect width="420" height="37" rx="16" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="2"/>'
        +'<text x="210" y="16" text-anchor="middle" font-size="10.5" font-weight="900" fill="'+info.stroke+'">当前控制：'+info.name+'</text>'
        +'<text x="210" y="31" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">'+info.feature+'</text>'
        +'</g>';
    }

    function muscleBar(
      x,
      y,
      width,
      value,
      color,
      labelText
    ){
      return ''
        +'<text x="'+x+'" y="'+(y-8)+'" font-size="10.5" font-weight="900" fill="#475569">'+labelText+'</text>'
        +'<rect x="'+x+'" y="'+y+'" width="'+width+'" height="17" rx="8" fill="#E2E8F0"/>'
        +'<rect x="'+x+'" y="'+y+'" width="'+(width*value/100).toFixed(1)+'" height="17" rx="8" fill="'+color+'"/>'
        +'<text x="'+(x+width+10)+'" y="'+(y+13)+'" font-size="10" font-weight="900" fill="'+color+'">'+value.toFixed(0)+'</text>';
    }

    function armPose(
      shoulderX,
      shoulderY,
      upperLength,
      foreLength,
      angle,
      color
    ){
      var radians=angle*Math.PI/180;
      var elbowX=shoulderX+upperLength;
      var elbowY=shoulderY;
      var handX=elbowX
        +foreLength*Math.cos(-radians);
      var handY=elbowY
        +foreLength*Math.sin(-radians);

      return ''
        +'<line x1="'+shoulderX+'" y1="'+shoulderY+'" x2="'+elbowX+'" y2="'+elbowY+'" stroke="#F1C27D" stroke-width="28" stroke-linecap="round"/>'
        +'<line x1="'+elbowX+'" y1="'+elbowY+'" x2="'+handX.toFixed(1)+'" y2="'+handY.toFixed(1)+'" stroke="#F1C27D" stroke-width="25" stroke-linecap="round"/>'
        +'<circle cx="'+shoulderX+'" cy="'+shoulderY+'" r="21" fill="#FED7AA" stroke="#C2410C" stroke-width="4"/>'
        +'<circle cx="'+elbowX+'" cy="'+elbowY+'" r="18" fill="#FFEDD5" stroke="#EA580C" stroke-width="4"/>'
        +'<circle cx="'+handX.toFixed(1)+'" cy="'+handY.toFixed(1)+'" r="16" fill="#FED7AA" stroke="#C2410C" stroke-width="4"/>'
        +'<path d="M'+(shoulderX+15)+' '+(shoulderY-18)+' Q'+((shoulderX+elbowX)/2)+' '+(shoulderY-48)+' '+(elbowX-13)+' '+(elbowY-15)+'" fill="none" stroke="'+color+'" stroke-width="14" stroke-linecap="round"/>'
        +'<path d="M'+(shoulderX+15)+' '+(shoulderY+18)+' Q'+((shoulderX+elbowX)/2)+' '+(shoulderY+46)+' '+(elbowX-13)+' '+(elbowY+15)+'" fill="none" stroke="#2563EB" stroke-width="12" stroke-linecap="round"/>'
        +'<path d="M'+elbowX+' '+elbowY+' A53 53 0 0 0 '+(elbowX+53*Math.cos(-radians)).toFixed(1)+' '+(elbowY+53*Math.sin(-radians)).toFixed(1)+'" fill="none" stroke="#EA580C" stroke-width="4" stroke-dasharray="6 5"/>'
        +'<text x="'+(elbowX+57)+'" y="'+(elbowY-31)+'" font-size="11" font-weight="900" fill="#9A3412">'+angle.toFixed(0)+'°</text>';
    }

    function renderMuscles(
      info,
      targetAngle,
      actualAngle,
      flexorActivation,
      extensorActivation,
      effectiveMuscle,
      error
    ){
      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<g transform="translate(49 112)">'
        +'<rect width="430" height="227" rx="22" fill="#FFF7ED" stroke="#FED7AA" stroke-width="4"/>'
        +'<text x="215" y="27" text-anchor="middle" font-size="13" font-weight="900" fill="#9A3412">肘关节屈曲示意：主动肌与拮抗肌协调</text>'
        +armPose(
          78,
          142,
          138,
          145,
          actualAngle,
          info.color
        )
        +'<path d="M216 142 A73 73 0 0 0 '+(216+73*Math.cos(-targetAngle*Math.PI/180)).toFixed(1)+' '+(142+73*Math.sin(-targetAngle*Math.PI/180)).toFixed(1)+'" fill="none" stroke="#16A34A" stroke-width="5" stroke-dasharray="8 6"/>'
        +'<text x="282" y="57" font-size="10.5" font-weight="900" fill="#15803D">目标角度 '+targetAngle.toFixed(0)+'°</text>'
        +'<text x="282" y="76" font-size="10.5" font-weight="900" fill="#C2410C">实际角度 '+actualAngle.toFixed(0)+'°</text>'
        +'<text x="282" y="95" font-size="10.5" font-weight="900" fill="#B91C1C">剩余误差 '+error.toFixed(1)+'°</text>'
        +'</g>'
        +'<g transform="translate(505 112)">'
        +'<rect width="198" height="227" rx="22" fill="#FFFFFF" stroke="#FED7AA" stroke-width="4"/>'
        +'<text x="99" y="27" text-anchor="middle" font-size="13" font-weight="900" fill="#9A3412">肌肉激活与关节稳定</text>'
        +muscleBar(
          18,
          65,
          136,
          flexorActivation,
          '#DC2626',
          '屈肌/主动肌'
        )
        +muscleBar(
          18,
          113,
          136,
          extensorActivation,
          '#2563EB',
          '伸肌/拮抗肌'
        )
        +muscleBar(
          18,
          161,
          136,
          effectiveMuscle,
          '#16A34A',
          '有效肌肉能力'
        )
        +'<text x="99" y="211" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">方向性运动常伴随交互协调；维持稳定时可有适度共同收缩。</text>'
        +'</g>'
        +'<g transform="translate(69 346)">'
        +'<rect width="410" height="20" rx="10" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="2"/>'
        +'<text x="205" y="14" text-anchor="middle" font-size="10" font-weight="900" fill="'+info.stroke+'">'+info.name+'下的肌肉激活为教学简化模型。</text>'
        +'</g>';
    }

    function feedbackNode(
      x,
      y,
      width,
      height,
      titleText,
      subtitleText,
      color,
      pale,
      active
    ){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="'+x+'" y="'+y+'" width="'+width+'" height="'+height+'" rx="20" fill="'+pale+'" stroke="'+color+'" stroke-width="'+(active?5:3)+'" opacity="'+(active?1:.58)+'"/>'
        +'</g>'
        +'<text x="'+(x+width/2)+'" y="'+(y+28)+'" text-anchor="middle" font-size="12.5" font-weight="900" fill="'+color+'">'+titleText+'</text>'
        +'<text x="'+(x+width/2)+'" y="'+(y+49)+'" text-anchor="middle" font-size="9.5" font-weight="800" fill="#64748B">'+subtitleText+'</text>';
    }

    function renderFeedback(
      info,
      progress,
      targetAngle,
      openLoopAngle,
      actualAngle,
      errorBefore,
      errorAfter,
      feedbackDrive,
      disturbance
    ){
      var p1=progress>=.12;
      var p2=progress>=.30;
      var p3=progress>=.50;
      var p4=progress>=.68;
      var p5=progress>=.84;

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +feedbackNode(
          52,
          127,
          130,
          78,
          '目标状态',
          '目标角度 '+targetAngle.toFixed(0)+'°',
          '#EA580C',
          '#FFF7ED',
          p1
        )
        +feedbackNode(
          218,
          127,
          130,
          78,
          '运动指令',
          '前馈计划与下行输出',
          '#7C3AED',
          '#F5F3FF',
          p2
        )
        +feedbackNode(
          384,
          127,
          130,
          78,
          '肌肉和关节',
          '执行后 '+openLoopAngle.toFixed(0)+'°',
          '#DC2626',
          '#FEF2F2',
          p3
        )
        +feedbackNode(
          550,
          127,
          130,
          78,
          '实际状态',
          '校正后 '+actualAngle.toFixed(0)+'°',
          '#16A34A',
          '#F0FDF4',
          p5
        )
        +'<path class="mf-flow" d="M182 166 H218" fill="none" stroke="#EA580C" stroke-width="6" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path class="mf-flow" d="M348 166 H384" fill="none" stroke="#7C3AED" stroke-width="6" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<path class="mf-flow" d="M514 166 H550" fill="none" stroke="#DC2626" stroke-width="6" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<path class="mf-error" d="M449 108 V82" fill="none" stroke="#DC2626" stroke-width="'+(3+disturbance/20)+'" marker-end="url(#${rootId}-arrow-red)"/>'
        +'<text x="449" y="75" text-anchor="middle" font-size="10.5" font-weight="900" fill="#B91C1C">外界扰动 '+disturbance.toFixed(0)+'</text>'
        +feedbackNode(
          419,
          250,
          176,
          76,
          '感觉反馈',
          '本体感觉、视觉等',
          '#2563EB',
          '#EFF6FF',
          p4
        )
        +feedbackNode(
          165,
          250,
          176,
          76,
          '比较与误差校正',
          '误差 '+errorBefore.toFixed(1)+'° → '+errorAfter.toFixed(1)+'°',
          '#D97706',
          '#FFFBEB',
          p5
        )
        +'<path class="mf-flow" d="M611 205 C614 233 594 258 595 275" fill="none" stroke="#2563EB" stroke-width="'+(3+feedbackDrive/24)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="mf-flow" d="M419 288 H341" fill="none" stroke="#2563EB" stroke-width="'+(3+feedbackDrive/24)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="mf-flow" d="M253 250 C253 222 280 208 283 205" fill="none" stroke="#D97706" stroke-width="'+(3+feedbackDrive/24)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<g transform="translate(47 337)">'
        +'<rect width="445" height="29" rx="14" fill="'+info.pale+'" stroke="'+info.stroke+'" stroke-width="2"/>'
        +'<text x="222" y="19" text-anchor="middle" font-size="10.5" font-weight="900" fill="'+info.stroke+'">前馈指令先启动运动，感觉反馈再根据目标与实际状态之间的误差持续校正。</text>'
        +'</g>';
    }

    function buildTrajectoryPath(
      startX,
      baseY,
      width,
      height,
      finalRatio,
      overshoot,
      damping
    ){
      var path='';

      for(var i=0;i<=100;i+=2){
        var t=i/100;
        var rise=1-Math.exp(
          -t*(3.2+damping*2.4)
        );
        var oscillation=
          overshoot
          *Math.sin(t*Math.PI*3.2)
          *Math.exp(
            -t*(1.6+damping*3.8)
          );
        var value=clamp(
          finalRatio*rise+oscillation,
          0,
          1.16
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

    function trajectoryValue(
      progress,
      finalRatio,
      overshoot,
      damping
    ){
      var rise=1-Math.exp(
        -progress*(3.2+damping*2.4)
      );
      var oscillation=
        overshoot
        *Math.sin(progress*Math.PI*3.2)
        *Math.exp(
          -progress*(1.6+damping*3.8)
        );

      return clamp(
        finalRatio*rise+oscillation,
        0,
        1.16
      );
    }

    function renderComparison(
      progress,
      openLoopAccuracy,
      feedbackAccuracy,
      combinedAccuracy,
      disturbance,
      feedbackDrive
    ){
      var startX=92;
      var baseY=307;
      var width=554;
      var height=148;

      var openRatio=clamp(
        openLoopAccuracy/100,
        .12,
        1
      );
      var feedbackRatio=clamp(
        feedbackAccuracy/100,
        .12,
        1
      );
      var combinedRatio=clamp(
        combinedAccuracy/100,
        .12,
        1
      );

      var openOvershoot=
        .05+disturbance/520;
      var feedbackOvershoot=
        .04+disturbance/850;
      var combinedOvershoot=
        .025+disturbance/1100;

      var openValue=trajectoryValue(
        progress,
        openRatio,
        openOvershoot,
        .06
      );
      var feedbackValue=trajectoryValue(
        progress,
        feedbackRatio,
        feedbackOvershoot,
        feedbackDrive/170
      );
      var combinedValue=trajectoryValue(
        progress,
        combinedRatio,
        combinedOvershoot,
        feedbackDrive/125
      );

      var markerX=startX+width*progress;

      return ''
        +'<rect x="27" y="91" width="706" height="275" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="380" y="121" text-anchor="middle" font-size="14" font-weight="900" fill="#9A3412">目标运动轨迹：前馈、反馈与反射协同的比较</text>'
        +'<line x1="'+startX+'" y1="'+baseY+'" x2="'+(startX+width)+'" y2="'+baseY+'" stroke="#64748B" stroke-width="3"/>'
        +'<line x1="'+startX+'" y1="'+(baseY-height)+'" x2="'+(startX+width)+'" y2="'+(baseY-height)+'" stroke="#16A34A" stroke-width="3" stroke-dasharray="8 6"/>'
        +'<line x1="'+startX+'" y1="'+(baseY-height-18)+'" x2="'+startX+'" y2="'+baseY+'" stroke="#64748B" stroke-width="3"/>'
        +'<text x="'+(startX-5)+'" y="'+(baseY-height-8)+'" text-anchor="end" font-size="10" font-weight="900" fill="#15803D">目标</text>'
        +'<text x="'+(startX+width)+'" y="'+(baseY+23)+'" text-anchor="end" font-size="10" font-weight="900" fill="#475569">时间</text>'
        +'<path d="'+buildTrajectoryPath(startX,baseY,width,height,openRatio,openOvershoot,.06)+'" fill="none" stroke="#94A3B8" stroke-width="4" stroke-linecap="round"/>'
        +'<path d="'+buildTrajectoryPath(startX,baseY,width,height,feedbackRatio,feedbackOvershoot,feedbackDrive/170)+'" fill="none" stroke="#EA580C" stroke-width="5" stroke-linecap="round"/>'
        +'<path d="'+buildTrajectoryPath(startX,baseY,width,height,combinedRatio,combinedOvershoot,feedbackDrive/125)+'" fill="none" stroke="#16A34A" stroke-width="5" stroke-linecap="round"/>'
        +'<line x1="'+markerX.toFixed(1)+'" y1="'+(baseY-height-12)+'" x2="'+markerX.toFixed(1)+'" y2="'+baseY+'" stroke="#7C3AED" stroke-width="2" stroke-dasharray="5 5"/>'
        +'<circle cx="'+markerX.toFixed(1)+'" cy="'+(baseY-height*openValue).toFixed(1)+'" r="7" fill="#94A3B8" stroke="#FFFFFF" stroke-width="2"/>'
        +'<circle cx="'+markerX.toFixed(1)+'" cy="'+(baseY-height*feedbackValue).toFixed(1)+'" r="7" fill="#EA580C" stroke="#FFFFFF" stroke-width="2"/>'
        +'<circle cx="'+markerX.toFixed(1)+'" cy="'+(baseY-height*combinedValue).toFixed(1)+'" r="7" fill="#16A34A" stroke="#FFFFFF" stroke-width="2"/>'
        +'<g transform="translate(105 333)">'
        +'<line x1="0" y1="0" x2="28" y2="0" stroke="#94A3B8" stroke-width="5"/>'
        +'<text x="37" y="4" font-size="9.5" font-weight="900" fill="#475569">仅前馈</text>'
        +'<line x1="122" y1="0" x2="150" y2="0" stroke="#EA580C" stroke-width="5"/>'
        +'<text x="159" y="4" font-size="9.5" font-weight="900" fill="#9A3412">加入反馈</text>'
        +'<line x1="264" y1="0" x2="292" y2="0" stroke="#16A34A" stroke-width="5"/>'
        +'<text x="301" y="4" font-size="9.5" font-weight="900" fill="#15803D">反馈与反射协同</text>'
        +'</g>'
        +'<g transform="translate(63 348)">'
        +'<rect width="420" height="18" rx="9" fill="#FFF7ED" stroke="#FED7AA" stroke-width="2"/>'
        +'<text x="210" y="13" text-anchor="middle" font-size="9.5" font-weight="900" fill="#9A3412">反馈有助于减小扰动和执行偏差，但过强或延迟的校正也可能造成振荡。</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='routes'){
        labels.innerHTML=''
          +'<path d="M103 147 L76 91" stroke="#EA580C" stroke-width="2.5"/>'
          +'<text x="24" y="86" font-size="12.5" font-weight="900" fill="#9A3412">运动计划网络</text>'
          +'<path d="M271 139 L271 88" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="217" y="82" font-size="12.5" font-weight="900" fill="#5B21B6">脊髓运动回路</text>'
          +'<path d="M447 151 L479 92" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="486" y="87" font-size="12.5" font-weight="900" fill="#1D4ED8">刺激和感觉输入</text>'
          +'<path d="M579 139 L620 91" stroke="#4F46E5" stroke-width="2.5"/>'
          +'<text x="627" y="86" font-size="12.5" font-weight="900" fill="#3730A3">反射中枢</text>';
        return;
      }

      if(modeName==='muscles'){
        labels.innerHTML=''
          +'<path d="M150 197 L121 91" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="53" y="86" font-size="12.5" font-weight="900" fill="#B91C1C">屈肌主动收缩</text>'
          +'<path d="M154 241 L204 91" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="211" y="86" font-size="12.5" font-weight="900" fill="#1D4ED8">伸肌协调性活动</text>'
          +'<path d="M266 252 L334 91" stroke="#EA580C" stroke-width="2.5"/>'
          +'<text x="341" y="86" font-size="12.5" font-weight="900" fill="#9A3412">关节角度与误差</text>';
        return;
      }

      if(modeName==='feedback'){
        labels.innerHTML=''
          +'<path d="M117 127 L87 90" stroke="#EA580C" stroke-width="2.5"/>'
          +'<text x="28" y="85" font-size="12.5" font-weight="900" fill="#9A3412">目标和运动计划</text>'
          +'<path d="M449 127 L482 88" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="489" y="83" font-size="12.5" font-weight="900" fill="#B91C1C">执行器与扰动</text>'
          +'<path d="M507 250 L551 213" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="558" y="208" font-size="12.5" font-weight="900" fill="#1D4ED8">感觉反馈通路</text>'
          +'<path d="M253 250 L214 214" stroke="#D97706" stroke-width="2.5"/>'
          +'<text x="145" y="209" font-size="12.5" font-weight="900" fill="#92400E">误差比较与校正</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M192 264 L153 91" stroke="#94A3B8" stroke-width="2.5"/>'
        +'<text x="92" y="86" font-size="12.5" font-weight="900" fill="#475569">无反馈偏差</text>'
        +'<path d="M358 225 L358 90" stroke="#EA580C" stroke-width="2.5"/>'
        +'<text x="304" y="85" font-size="12.5" font-weight="900" fill="#9A3412">反馈校正轨迹</text>'
        +'<path d="M539 185 L602 91" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="609" y="86" font-size="12.5" font-weight="900" fill="#15803D">反射和反馈协同</text>';
    }

    function update(){
      var command=Number(
        commandInput.value
      );
      var sensory=Number(
        sensoryInputElement.value
      );
      var pathway=Number(
        pathwayInput.value
      );
      var muscle=Number(
        muscleInput.value
      );
      var disturbance=Number(
        disturbanceInput.value
      );
      var feedback=Number(
        feedbackInput.value
      );
      var fatigue=Number(
        fatigueInput.value
      );
      var processTime=Number(
        timeInput.value
      );

      commandValue.textContent=
        command.toFixed(0)+'%';
      sensoryValue.textContent=
        sensory.toFixed(0)+'%';
      pathwayValue.textContent=
        pathway.toFixed(0)+'%';
      muscleValue.textContent=
        muscle.toFixed(0)+'%';
      disturbanceValue.textContent=
        disturbance.toFixed(0)+'%';
      feedbackValue.textContent=
        feedback.toFixed(0)+'%';
      fatigueValue.textContent=
        fatigue.toFixed(0)+'%';
      timeValue.textContent=
        processTime.toFixed(0)+'%';

      for(var i=0;i<movementButtons.length;i++){
        movementButtons[i].classList.toggle(
          'active',
          movementButtons[i].getAttribute(
            'data-movement'
          )===movementType
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
        ?'运动推进：运行中'
        :'运动推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var info=movementInformation[
        movementType
      ];
      var progress=processTime/100;

      var commandDrive=command
        *pathway/100;

      var reflexDrive=sensory
        *pathway/100;

      var feedbackDrive=sensory
        *feedback/100
        *pathway/100;

      var effectiveMuscle=clamp(
        muscle
        *(
          1-fatigue*.0065
        ),
        8,
        100
      );

      var drive=commandDrive;

      if(movementType==='reflex'){
        drive=reflexDrive;
      }else if(movementType==='combined'){
        drive=
          commandDrive*.64
          +reflexDrive*.36;
      }

      drive=clamp(
        drive,
        0,
        100
      );

      var targetSource=movementType==='reflex'
        ?sensory
        :command;

      var targetAngle=clamp(
        15+targetSource*.55,
        15,
        70
      );

      var openLoopAngle=targetAngle
        *effectiveMuscle/100
        *(
          .38+.62*drive/100
        )
        -disturbance*.20;

      openLoopAngle=clamp(
        openLoopAngle,
        0,
        80
      );

      var errorBefore=
        targetAngle-openLoopAngle;

      var correctionFraction=
        feedbackDrive/100
        *(
          movementType==='combined'
            ?1.08
            :movementType==='reflex'
              ?.82
              :.94
        );

      correctionFraction=clamp(
        correctionFraction,
        0,
        .92
      );

      var correction=
        errorBefore
        *correctionFraction;

      var finalAngle=clamp(
        openLoopAngle+correction,
        0,
        82
      );

      var errorAfter=Math.abs(
        targetAngle-finalAngle
      );

      var accuracy=clamp(
        100-errorAfter*1.65,
        0,
        100
      );

      var responseDelay=0;

      if(movementType==='reflex'){
        responseDelay=clamp(
          72
          -sensory*.29
          -pathway*.30
          +fatigue*.08,
          10,
          86
        );
      }else if(movementType==='combined'){
        responseDelay=clamp(
          80
          -Math.max(
            command,
            sensory
          )*.22
          -pathway*.30
          +fatigue*.10,
          12,
          90
        );
      }else{
        responseDelay=clamp(
          94
          -command*.18
          -pathway*.34
          +fatigue*.12,
          18,
          96
        );
      }

      var flexorActivation=clamp(
        drive
        *(
          1-fatigue*.0045
        ),
        0,
        100
      );

      var extensorActivation=clamp(
        30
        -drive*.16
        +disturbance*.20
        +feedbackDrive*.08,
        4,
        72
      );

      if(movementType==='combined'){
        extensorActivation=clamp(
          extensorActivation+8,
          4,
          78
        );
      }

      var progressFactor=
        .22+.78*(
          1-Math.exp(
            -progress*4.2
          )
        );

      var displayedAngle=clamp(
        finalAngle*progressFactor,
        0,
        finalAngle
      );

      delayText.textContent=
        responseDelay.toFixed(0);
      accuracyText.textContent=
        accuracy.toFixed(0);
      errorText.textContent=
        errorAfter.toFixed(1)+'°';

      root.style.setProperty(
        '--mf-speed',
        clamp(
          2.45-Math.max(
            drive,
            feedbackDrive
          )/72,
          .58,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='routes'){
        title.textContent=
          '随意运动、反射运动及其协同通路';

        summary.textContent=
          '比较运动计划启动的下行控制与感觉刺激启动的反射通路。';

        dynamic.innerHTML=renderRoutes(
          info,
          progress,
          commandDrive,
          reflexDrive,
          feedbackDrive
        );

        stageNote.textContent=
          movementType==='voluntary'
            ?'随意运动由目标和运动计划启动，但执行过程中仍需要脊髓回路、肌肉和感觉反馈共同参与。'
            :movementType==='reflex'
              ?'反射运动通常不需要意识先作决定，但反射强度和后续行为仍可受到高级中枢调节。'
              :'真实运动中，随意计划、脊髓反射和感觉反馈可同时参与并相互协调。';

        renderLabels(mode);
      }else if(mode==='muscles'){
        title.textContent=
          '主动肌、拮抗肌与关节运动';

        summary.textContent=
          '观察肌肉响应、疲劳、扰动和反馈如何共同影响关节实际角度。';

        dynamic.innerHTML=renderMuscles(
          info,
          targetAngle,
          displayedAngle,
          flexorActivation,
          extensorActivation,
          effectiveMuscle,
          errorAfter
        );

        stageNote.textContent=
          '主动肌和拮抗肌需要协调激活；方向性运动可伴随交互抑制，维持稳定时也可能出现适度共同收缩。';

        renderLabels(mode);
      }else if(mode==='feedback'){
        title.textContent=
          '目标、运动指令、感觉反馈与误差校正';

        summary.textContent=
          '观察前馈运动指令如何启动动作，以及反馈信息如何根据误差修正运动结果。';

        dynamic.innerHTML=renderFeedback(
          info,
          progress,
          targetAngle,
          openLoopAngle,
          finalAngle,
          Math.abs(errorBefore),
          errorAfter,
          feedbackDrive,
          disturbance
        );

        stageNote.textContent=
          feedbackDrive>55
            ?'感觉反馈和校正增益较高，目标状态与实际状态之间的误差得到较明显修正。'
            :'感觉反馈或校正增益较低，运动误差不能被充分检测和修正。';

        renderLabels(mode);
      }else{
        var openLoopAccuracy=clamp(
          100-Math.abs(
            targetAngle-openLoopAngle
          )*1.65,
          0,
          100
        );

        var feedbackAccuracy=accuracy;

        var combinedImprovement=
          sensory
          *pathway/100
          *feedback/100
          *.16;

        var combinedAccuracy=clamp(
          Math.max(
            feedbackAccuracy,
            openLoopAccuracy
          )
          +combinedImprovement,
          0,
          100
        );

        title.textContent=
          '前馈、反馈和反射协同的运动轨迹';

        summary.textContent=
          '比较只有前馈指令、加入感觉反馈以及反射和反馈协同时的目标逼近过程。';

        dynamic.innerHTML=renderComparison(
          progress,
          openLoopAccuracy,
          feedbackAccuracy,
          combinedAccuracy,
          disturbance,
          feedbackDrive
        );

        stageNote.textContent=
          '前馈控制可快速启动动作，反馈和反射调节有助于抵消扰动、减小误差并提高运动稳定性。';

        renderLabels(mode);
      }

      var condition=
        '当前运动指令、感觉反馈、运动通路和肌肉响应能够形成较稳定的目标运动。';

      if(
        movementType==='voluntary'
        &&command<15
      ){
        condition=
          '随意运动指令很弱，目标运动计划难以形成足够的下行驱动。';
      }else if(
        movementType==='reflex'
        &&sensory<15
      ){
        condition=
          '感觉输入很弱，感受器和传入神经不足以形成明显的反射驱动。';
      }else if(pathway<35){
        condition=
          '运动通路完整度较低，下行指令、传入信号和运动输出均明显衰减。';
      }else if(muscle<35){
        condition=
          '肌肉响应能力较低，即使神经指令到达，关节运动幅度仍受到限制。';
      }else if(fatigue>78){
        condition=
          '肌肉疲劳程度很高，有效肌肉能力下降，动作误差和稳定性问题增加。';
      }else if(disturbance>78){
        condition=
          '外界扰动很强，只有较高质量的感觉反馈和适当校正才能减小运动偏差。';
      }else if(
        sensory<20
        ||feedback<20
      ){
        condition=
          '感觉反馈或校正增益较低，目标与实际状态之间的误差不能被充分修正。';
      }else if(
        feedback>90
        &&sensory>85
      ){
        condition=
          '反馈校正增益很高；真实系统若同时存在反馈延迟，过强校正可能导致振荡，本模型只作边界提示。';
      }else if(processTime<16){
        condition=
          '运动过程时间较短，运动指令刚开始传递，肌肉收缩和反馈校正尚未充分展开。';
      }

      var principle=mode==='routes'
        ?'随意运动主要由目标和计划启动，反射运动主要由感觉刺激启动；两类通路可在真实运动中协同工作。'
        :mode==='muscles'
          ?'骨骼肌运动依赖主动肌、拮抗肌和稳定肌协调，肌肉能力、疲劳和扰动共同决定实际运动结果。'
          :mode==='feedback'
            ?'中枢可比较目标状态和感觉反馈报告的实际状态，根据运动误差形成新的校正指令。'
            :'前馈控制负责快速启动和预测性控制，反馈与反射调节负责根据实际运动后果减小误差。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前目标角度 '
        +targetAngle.toFixed(0)
        +'°，校正后角度 '
        +finalAngle.toFixed(0)
        +'°，运动准确度 '
        +accuracy.toFixed(0)
        +'，相对反应延迟 '
        +responseDelay.toFixed(0)
        +'。所有数值仅用于课堂比较，不用于运动能力或医学判断。';
    }

    for(var i=0;i<movementButtons.length;i++){
      movementButtons[i].onclick=function(){
        movementType=this.getAttribute(
          'data-movement'
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

    commandInput.oninput=update;
    sensoryInputElement.oninput=update;
    pathwayInput.oninput=update;
    muscleInput.oninput=update;
    disturbanceInput.oninput=update;
    feedbackInput.oninput=update;
    fatigueInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
