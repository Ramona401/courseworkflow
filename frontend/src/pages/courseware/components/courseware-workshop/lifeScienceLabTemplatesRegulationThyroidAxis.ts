/**
 * lifeScienceLabTemplatesRegulationThyroidAxis.ts
 *
 * 平面生命科学实验室：
 * 甲状腺激素的分级调节与负反馈。
 *
 * 教学目标：
 * 1. 认识下丘脑、垂体前叶和甲状腺组成的调节轴；
 * 2. 理解下丘脑可分泌促甲状腺激素释放激素TRH；
 * 3. 理解TRH作用于垂体前叶，
 *    促进促甲状腺激素TSH的合成和分泌；
 * 4. 理解TSH作用于甲状腺，
 *    促进甲状腺激素的合成和释放；
 * 5. 理解甲状腺激素可作用于多种组织，
 *    参与代谢、生长发育等生理过程；
 * 6. 理解下丘脑、垂体和甲状腺之间体现分级调节；
 * 7. 理解循环中的甲状腺激素水平升高后，
 *    可抑制下丘脑和垂体相关激素的分泌；
 * 8. 理解甲状腺激素水平降低时，
 *    对下丘脑和垂体的负反馈抑制相对减弱；
 * 9. 观察环境变化、代谢需求、各级反应能力、
 *    反馈敏感性和激素清除对调节轴的影响；
 * 10. 区分促进方向的分级调节和反向的负反馈调节；
 * 11. 理解激素调节通常存在一定时间延迟，
 *     不会在刺激出现后瞬间达到新稳态；
 * 12. 理解稳态是激素合成、分泌、运输、
 *     靶组织作用和清除共同形成的动态结果。
 *
 * 科学边界：
 * 1. 下丘脑分泌TRH，
 *    TRH经垂体门脉系统作用于垂体前叶；
 * 2. 垂体前叶分泌TSH，
 *    TSH经血液运输并作用于甲状腺；
 * 3. TSH促进甲状腺合成和释放甲状腺激素，
 *    并对甲状腺具有一定营养性作用；
 * 4. 甲状腺主要释放甲状腺素T4和少量三碘甲状腺原氨酸T3，
 *    外周组织还可发生甲状腺激素转化；
 * 5. 本模板统一使用“甲状腺激素”表示教学中的综合相对水平，
 *    不分别模拟游离T3、游离T4和总激素水平；
 * 6. 甲状腺激素可影响能量代谢、产热、生长发育
 *    和神经系统发育等多个过程；
 * 7. 循环甲状腺激素可对下丘脑和垂体产生负反馈；
 * 8. 甲状腺激素升高时，
 *    TRH和TSH的分泌倾向通常受到抑制；
 * 9. 甲状腺激素降低时，
 *    负反馈抑制减弱，TRH和TSH分泌倾向可增强；
 * 10. 寒冷、昼夜节律、营养状态、年龄、
 *     应激和疾病等均可能影响调节轴；
 * 11. 寒冷刺激对人体甲状腺轴的影响具有年龄、
 *     持续时间和生理状态差异，
 *     本模型仅作教材层面的简化演示；
 * 12. 不能把TSH直接等同于甲状腺激素，
 *     TSH主要是调节甲状腺的促激素；
 * 13. 不能把分级调节理解为单向直线控制，
 *     调节轴还受到负反馈及其他神经体液因素影响；
 * 14. 真实内分泌调节存在脉冲分泌、昼夜节律、
 *     结合蛋白、受体差异和外周转化等复杂过程；
 * 15. 本模型不模拟甲状腺自身调节、
 *     碘代谢和所有相关激素之间的相互作用；
 * 16. 图中的TRH、TSH、甲状腺激素、
 *     代谢效应和反馈强度均为相对教学指标；
 * 17. 本模板只用于生物学教学，
 *     不用于甲状腺功能、激素检测或疾病诊断。
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

function thyroidAxisStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FBCFE8;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#EDE9FE);border-bottom:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFF9FC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#DB2777}'
    + '#' + rootId + ' .ta-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .ta-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ta-button{min-height:30px;padding:3px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .ta-button.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.14)}'
    + '#' + rootId + ' .ta-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ta-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ta-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .ta-card{padding:6px 3px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ta-card b{display:block;font-size:13px;color:#BE185D}'
    + '#' + rootId + ' .ta-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ta-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--ta-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ta-pulse{animation:' + rootId + '-pulse 1.55s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.34}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_REGULATION_THYROID_AXIS:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-thyroid-axis-feedback',
    group: '🧠 稳态与调节',
    name: '甲状腺激素的分级调节与负反馈',
    emoji: '🦋',
    desc: '调节环境寒冷、代谢需求、下丘脑和垂体反应、甲状腺合成能力、反馈敏感性及激素清除，观察甲状腺轴分级调节',
    params: [
      {
        key: 'coldStimulus',
        label: '环境寒冷刺激',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 46,
      },
      {
        key: 'metabolicDemand',
        label: '机体代谢需求',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
      {
        key: 'hypothalamusSensitivity',
        label: '下丘脑反应能力',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 84,
      },
      {
        key: 'pituitarySensitivity',
        label: '垂体前叶反应能力',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'thyroidCapacity',
        label: '甲状腺合成能力',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 86,
      },
      {
        key: 'feedbackSensitivity',
        label: '负反馈敏感性',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'hormoneClearance',
        label: '激素清除效率',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'processTime',
        label: '调节过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 52,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const coldStimulus = num(
        params,
        'coldStimulus',
        46,
      )
      const metabolicDemand = num(
        params,
        'metabolicDemand',
        58,
      )
      const hypothalamusSensitivity = num(
        params,
        'hypothalamusSensitivity',
        84,
      )
      const pituitarySensitivity = num(
        params,
        'pituitarySensitivity',
        82,
      )
      const thyroidCapacity = num(
        params,
        'thyroidCapacity',
        86,
      )
      const feedbackSensitivity = num(
        params,
        'feedbackSensitivity',
        78,
      )
      const hormoneClearance = num(
        params,
        'hormoneClearance',
        62,
      )
      const processTime = num(
        params,
        'processTime',
        52,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${thyroidAxisStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🦋 甲状腺激素的分级调节与负反馈</div>
    <div class="bl-note">激素水平和代谢效应均为相对教学指标</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>环境寒冷刺激</span>
          <span class="bl-value" data-cold-value></span>
        </div>
        <input
          data-cold
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(coldStimulus)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>机体代谢需求</span>
          <span class="bl-value" data-demand-value></span>
        </div>
        <input
          data-demand
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(metabolicDemand)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>下丘脑反应能力</span>
          <span class="bl-value" data-hypothalamus-value></span>
        </div>
        <input
          data-hypothalamus
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(hypothalamusSensitivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>垂体前叶反应能力</span>
          <span class="bl-value" data-pituitary-value></span>
        </div>
        <input
          data-pituitary
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(pituitarySensitivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>甲状腺合成能力</span>
          <span class="bl-value" data-thyroid-value></span>
        </div>
        <input
          data-thyroid
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(thyroidCapacity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>负反馈敏感性</span>
          <span class="bl-value" data-feedback-value></span>
        </div>
        <input
          data-feedback
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(feedbackSensitivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>激素清除效率</span>
          <span class="bl-value" data-clearance-value></span>
        </div>
        <input
          data-clearance
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(hormoneClearance)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>调节过程时间</span>
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

      <div class="ta-subtitle">观察方式</div>

      <div class="ta-buttons">
        <button
          type="button"
          class="ta-button active"
          data-mode="axis"
        >调节轴结构</button>

        <button
          type="button"
          class="ta-button"
          data-mode="forward"
        >分级促进过程</button>

        <button
          type="button"
          class="ta-button"
          data-mode="feedback"
        >负反馈调节</button>

        <button
          type="button"
          class="ta-button"
          data-mode="scenarios"
        >典型情境比较</button>
      </div>

      <button
        type="button"
        class="ta-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="ta-toggle"
        data-auto
      >调节推进：运行中</button>

      <div class="ta-status">
        <div class="ta-card">
          <b data-trh-index></b>
          <span>TRH信号</span>
        </div>

        <div class="ta-card">
          <b data-tsh-index></b>
          <span>TSH信号</span>
        </div>

        <div class="ta-card">
          <b data-th-index></b>
          <span>甲状腺激素</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="甲状腺激素分级调节与负反馈互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-pink"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#DB2777"/>
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
            id="${rootId}-arrow-orange"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#D97706"/>
          </marker>

          <marker
            id="${rootId}-inhibit"
            markerWidth="12"
            markerHeight="12"
            refX="10"
            refY="6"
            orient="auto"
          >
            <path
              d="M10 1 V11"
              stroke="#DC2626"
              stroke-width="3"
            />
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#831843"
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
          fill="#9D174D"
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

        <g transform="translate(512 337)">
          <rect
            width="222"
            height="66"
            rx="15"
            fill="#FDF2F8"
            stroke="#FBCFE8"
            stroke-width="2"
          />

          <text
            x="111"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#9D174D"
          >关键边界</text>

          <text
            x="111"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#831843"
          >TSH不是甲状腺激素</text>

          <text
            x="111"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#831843"
          >分级促进与负反馈共同维持稳态</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#9D174D"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var coldInput=root.querySelector(
      '[data-cold]'
    );
    var demandInput=root.querySelector(
      '[data-demand]'
    );
    var hypothalamusInput=root.querySelector(
      '[data-hypothalamus]'
    );
    var pituitaryInput=root.querySelector(
      '[data-pituitary]'
    );
    var thyroidInput=root.querySelector(
      '[data-thyroid]'
    );
    var feedbackInput=root.querySelector(
      '[data-feedback]'
    );
    var clearanceInput=root.querySelector(
      '[data-clearance]'
    );
    var timeInput=root.querySelector(
      '[data-time]'
    );

    var coldValue=root.querySelector(
      '[data-cold-value]'
    );
    var demandValue=root.querySelector(
      '[data-demand-value]'
    );
    var hypothalamusValue=root.querySelector(
      '[data-hypothalamus-value]'
    );
    var pituitaryValue=root.querySelector(
      '[data-pituitary-value]'
    );
    var thyroidValue=root.querySelector(
      '[data-thyroid-value]'
    );
    var feedbackValue=root.querySelector(
      '[data-feedback-value]'
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

    var trhText=root.querySelector(
      '[data-trh-index]'
    );
    var tshText=root.querySelector(
      '[data-tsh-index]'
    );
    var thyroidHormoneText=root.querySelector(
      '[data-th-index]'
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

    var mode='axis';
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
      },820);
    }

    function hormoneParticle(
      x,
      y,
      label,
      color,
      opacity,
      scale
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle r="13" fill="'+color+'" stroke="#FFFFFF" stroke-width="2.5"/>'
        +'<text x="0" y="4" text-anchor="middle" font-size="7.5" font-weight="900" fill="#FFFFFF">'+label+'</text>'
        +'</g>';
    }

    function glandCard(
      x,
      y,
      width,
      height,
      titleText,
      subtitle,
      color,
      fill,
      value
    ){
      var ring=28+value*.14;

      return ''
        +'<g transform="translate('+x+' '+y+')" filter="url(#${rootId}-shadow)">'
        +'<rect width="'+width+'" height="'+height+'" rx="22" fill="'+fill+'" stroke="'+color+'" stroke-width="4"/>'
        +'<circle cx="'+(width/2)+'" cy="49" r="31" fill="#FFFFFF" stroke="'+color+'" stroke-width="4"/>'
        +'<circle class="ta-pulse" cx="'+(width/2)+'" cy="49" r="'+ring.toFixed(1)+'" fill="none" stroke="'+color+'" stroke-width="3" stroke-dasharray="7 6" opacity="'+(.20+.62*value/100)+'"/>'
        +'<text x="'+(width/2)+'" y="55" text-anchor="middle" font-size="16" font-weight="900" fill="'+color+'">'+titleText.substring(0,1)+'</text>'
        +'<text x="'+(width/2)+'" y="101" text-anchor="middle" font-size="14" font-weight="900" fill="'+color+'">'+titleText+'</text>'
        +'<text x="'+(width/2)+'" y="124" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'+subtitle+'</text>'
        +'<text x="'+(width/2)+'" y="'+(height-14)+'" text-anchor="middle" font-size="11" font-weight="900" fill="'+color+'">相对活动 '+value.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderAxis(
      trh,
      tsh,
      thyroidHormone,
      feedbackPressure
    ){
      return ''
        +glandCard(
          30,
          104,
          190,
          178,
          '下丘脑',
          '分泌TRH',
          '#7C3AED',
          '#F5F3FF',
          trh
        )
        +glandCard(
          285,
          104,
          190,
          178,
          '垂体前叶',
          '分泌TSH',
          '#DB2777',
          '#FDF2F8',
          tsh
        )
        +glandCard(
          540,
          104,
          190,
          178,
          '甲状腺',
          '分泌甲状腺激素',
          '#D97706',
          '#FFFBEB',
          thyroidHormone
        )
        +'<path class="ta-flow" d="M222 193 H278" fill="none" stroke="#7C3AED" stroke-width="'+(3+trh/22)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<text x="250" y="179" text-anchor="middle" font-size="12" font-weight="900" fill="#6D28D9">TRH</text>'
        +'<path class="ta-flow" d="M477 193 H533" fill="none" stroke="#DB2777" stroke-width="'+(3+tsh/22)+'" marker-end="url(#${rootId}-arrow-pink)"/>'
        +'<text x="505" y="179" text-anchor="middle" font-size="12" font-weight="900" fill="#BE185D">TSH</text>'
        +'<path class="ta-flow" d="M635 284 C635 337 112 337 112 286" fill="none" stroke="#DC2626" stroke-width="'+(2.5+feedbackPressure/28)+'" marker-end="url(#${rootId}-inhibit)"/>'
        +'<path class="ta-flow" d="M635 297 C635 365 380 365 380 286" fill="none" stroke="#DC2626" stroke-width="'+(2.5+feedbackPressure/28)+'" marker-end="url(#${rootId}-inhibit)"/>'
        +'<text x="376" y="347" text-anchor="middle" font-size="12" font-weight="900" fill="#B91C1C">循环甲状腺激素对下丘脑和垂体产生负反馈</text>';
    }

    function renderForward(
      progress,
      trh,
      tsh,
      thyroidHormone,
      metabolicEffect
    ){
      var trhParticles='';
      var tshParticles='';
      var thyroidParticles='';

      var trhCount=Math.floor(
        2+trh/15
      );
      var tshCount=Math.floor(
        2+tsh/15
      );
      var thyroidCount=Math.floor(
        2+thyroidHormone/13
      );

      for(var i=0;i<trhCount;i++){
        trhParticles+=hormoneParticle(
          188+(i%4)*34,
          156+Math.floor(i/4)*35,
          'TRH',
          '#7C3AED',
          .32+.62*progress,
          .64
        );
      }

      for(var j=0;j<tshCount;j++){
        tshParticles+=hormoneParticle(
          386+(j%4)*34,
          156+Math.floor(j/4)*35,
          'TSH',
          '#DB2777',
          .32+.62*progress,
          .64
        );
      }

      for(var k=0;k<thyroidCount;k++){
        thyroidParticles+=hormoneParticle(
          579+(k%4)*34,
          156+Math.floor(k/4)*35,
          'TH',
          '#D97706',
          .32+.62*progress,
          .64
        );
      }

      return ''
        +'<rect x="27" y="91" width="706" height="274" rx="24" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<g transform="translate(51 119)">'
        +'<rect width="137" height="157" rx="20" fill="#F5F3FF" stroke="#C4B5FD" stroke-width="4"/>'
        +'<ellipse cx="68" cy="58" rx="44" ry="35" fill="#DDD6FE" stroke="#7C3AED" stroke-width="4"/>'
        +'<text x="68" y="63" text-anchor="middle" font-size="13" font-weight="900" fill="#5B21B6">下丘脑</text>'
        +'<text x="68" y="111" text-anchor="middle" font-size="12" font-weight="900" fill="#6D28D9">释放TRH</text>'
        +'<text x="68" y="137" text-anchor="middle" font-size="11" font-weight="800" fill="#475569">'+trh.toFixed(0)+'</text>'
        +'</g>'
        +'<path class="ta-flow" d="M188 198 H270" fill="none" stroke="#7C3AED" stroke-width="'+(3+trh/23)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +trhParticles
        +'<g transform="translate(272 119)">'
        +'<rect width="137" height="157" rx="20" fill="#FDF2F8" stroke="#F9A8D4" stroke-width="4"/>'
        +'<ellipse cx="68" cy="58" rx="44" ry="35" fill="#FBCFE8" stroke="#DB2777" stroke-width="4"/>'
        +'<text x="68" y="63" text-anchor="middle" font-size="13" font-weight="900" fill="#9D174D">垂体前叶</text>'
        +'<text x="68" y="111" text-anchor="middle" font-size="12" font-weight="900" fill="#BE185D">释放TSH</text>'
        +'<text x="68" y="137" text-anchor="middle" font-size="11" font-weight="800" fill="#475569">'+tsh.toFixed(0)+'</text>'
        +'</g>'
        +'<path class="ta-flow" d="M409 198 H491" fill="none" stroke="#DB2777" stroke-width="'+(3+tsh/23)+'" marker-end="url(#${rootId}-arrow-pink)"/>'
        +tshParticles
        +'<g transform="translate(493 119)">'
        +'<rect width="137" height="157" rx="20" fill="#FFFBEB" stroke="#FCD34D" stroke-width="4"/>'
        +'<path d="M47 42 C22 54 24 91 54 94 C61 95 66 89 68 80 C70 89 75 95 82 94 C112 91 114 54 89 42 C79 37 72 44 68 53 C64 44 57 37 47 42Z" fill="#FDE68A" stroke="#D97706" stroke-width="4"/>'
        +'<text x="68" y="111" text-anchor="middle" font-size="12" font-weight="900" fill="#92400E">甲状腺激素</text>'
        +'<text x="68" y="137" text-anchor="middle" font-size="11" font-weight="800" fill="#475569">'+thyroidHormone.toFixed(0)+'</text>'
        +'</g>'
        +thyroidParticles
        +'<path class="ta-flow" d="M631 198 H686" fill="none" stroke="#D97706" stroke-width="'+(3+thyroidHormone/23)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<g transform="translate(633 120)">'
        +'<rect width="76" height="154" rx="18" fill="#ECFDF5" stroke="#A7F3D0" stroke-width="3"/>'
        +'<text x="38" y="31" text-anchor="middle" font-size="11" font-weight="900" fill="#047857">靶组织</text>'
        +'<text x="38" y="57" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">代谢</text>'
        +'<text x="38" y="77" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">产热</text>'
        +'<text x="38" y="97" text-anchor="middle" font-size="9.5" font-weight="800" fill="#475569">生长发育</text>'
        +'<text x="38" y="128" text-anchor="middle" font-size="15" font-weight="900" fill="#047857">'+metabolicEffect.toFixed(0)+'</text>'
        +'</g>'
        +'<text x="380" y="332" text-anchor="middle" font-size="12" font-weight="900" fill="#9D174D">下丘脑 → TRH → 垂体前叶 → TSH → 甲状腺 → 甲状腺激素</text>';
    }

    function feedbackBar(
      x,
      y,
      width,
      value,
      color,
      label
    ){
      return ''
        +'<text x="'+x+'" y="'+(y-8)+'" font-size="11" font-weight="900" fill="#475569">'+label+'</text>'
        +'<rect x="'+x+'" y="'+y+'" width="'+width+'" height="16" rx="8" fill="#E2E8F0"/>'
        +'<rect x="'+x+'" y="'+y+'" width="'+(width*clamp(value,0,100)/100).toFixed(1)+'" height="16" rx="8" fill="'+color+'"/>'
        +'<text x="'+(x+width)+'" y="'+(y+14)+'" text-anchor="end" font-size="10" font-weight="900" fill="#334155">'+value.toFixed(0)+'</text>';
    }

    function renderFeedback(
      trh,
      tsh,
      thyroidHormone,
      feedbackPressure,
      metabolicEffect
    ){
      return ''
        +'<g transform="translate(31 94)">'
        +'<rect width="431" height="268" rx="23" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="215" y="31" text-anchor="middle" font-size="15" font-weight="900" fill="#334155">激素水平与反馈强度</text>'
        +feedbackBar(34,76,362,trh,'#7C3AED','TRH相对信号')
        +feedbackBar(34,126,362,tsh,'#DB2777','TSH相对信号')
        +feedbackBar(34,176,362,thyroidHormone,'#D97706','甲状腺激素相对水平')
        +feedbackBar(34,226,362,feedbackPressure,'#DC2626','负反馈抑制强度')
        +'</g>'
        +'<g transform="translate(484 94)">'
        +'<rect width="245" height="268" rx="23" fill="#FDF2F8" stroke="#FBCFE8" stroke-width="3"/>'
        +'<text x="122" y="31" text-anchor="middle" font-size="15" font-weight="900" fill="#9D174D">动态负反馈环路</text>'
        +'<circle cx="122" cy="111" r="59" fill="#FFFFFF" stroke="#DB2777" stroke-width="6"/>'
        +'<text x="122" y="101" text-anchor="middle" font-size="12" font-weight="900" fill="#9D174D">甲状腺激素</text>'
        +'<text x="122" y="129" text-anchor="middle" font-size="26" font-weight="900" fill="#BE185D">'+thyroidHormone.toFixed(0)+'</text>'
        +'<path class="ta-flow" d="M59 180 C80 155 164 155 186 180 C206 203 186 228 157 225" fill="none" stroke="#DC2626" stroke-width="'+(3+feedbackPressure/25)+'" marker-end="url(#${rootId}-inhibit)"/>'
        +'<text x="122" y="197" text-anchor="middle" font-size="11" font-weight="900" fill="#B91C1C">水平升高</text>'
        +'<text x="122" y="218" text-anchor="middle" font-size="11" font-weight="900" fill="#B91C1C">抑制TRH和TSH</text>'
        +'<text x="122" y="247" text-anchor="middle" font-size="10.5" font-weight="900" fill="#047857">代谢效应 '+metabolicEffect.toFixed(0)+'</text>'
        +'</g>';
    }

    function scenarioCard(
      x,
      titleText,
      line1,
      line2,
      active,
      color,
      fill
    ){
      return ''
        +'<g transform="translate('+x+' 99)">'
        +'<rect width="164" height="184" rx="20" fill="'+fill+'" stroke="'+color+'" stroke-width="'+(active?5:3)+'"/>'
        +'<circle cx="82" cy="48" r="28" fill="#FFFFFF" stroke="'+color+'" stroke-width="4"/>'
        +'<text x="82" y="55" text-anchor="middle" font-size="20" font-weight="900" fill="'+color+'">'+(active?'●':'○')+'</text>'
        +'<text x="82" y="101" text-anchor="middle" font-size="13.5" font-weight="900" fill="'+color+'">'+titleText+'</text>'
        +'<text x="82" y="131" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'+line1+'</text>'
        +'<text x="82" y="153" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'+line2+'</text>'
        +'</g>';
    }

    function renderScenarios(
      cold,
      demand,
      trh,
      tsh,
      thyroidHormone,
      thyroidCapacity,
      feedbackSensitivity,
      clearance
    ){
      var state='balanced';

      if(
        thyroidCapacity<35
        &&tsh>48
      ){
        state='lowCapacity';
      }else if(
        feedbackSensitivity<35
        &&thyroidHormone>58
      ){
        state='weakFeedback';
      }else if(
        cold>72
        ||demand>78
      ){
        state='highDemand';
      }else if(
        clearance<30
        &&thyroidHormone>58
      ){
        state='slowClearance';
      }

      return ''
        +scenarioCard(
          24,
          '需求增强',
          '上游刺激增强',
          'TRH、TSH倾向升高',
          state==='highDemand',
          '#7C3AED',
          '#F5F3FF'
        )
        +scenarioCard(
          205,
          '合成能力较低',
          'TSH刺激存在',
          '甲状腺激素响应受限',
          state==='lowCapacity',
          '#D97706',
          '#FFFBEB'
        )
        +scenarioCard(
          386,
          '反馈敏感性低',
          '负反馈抑制较弱',
          '上游信号相对偏高',
          state==='weakFeedback',
          '#DC2626',
          '#FFF1F2'
        )
        +scenarioCard(
          567,
          state==='slowClearance'
            ?'清除效率较低'
            :'接近动态稳态',
          state==='slowClearance'
            ?'激素清除减慢'
            :'合成与清除协调',
          state==='slowClearance'
            ?'循环水平相对积累'
            :'负反馈减小偏差',
          state==='balanced'
            ||state==='slowClearance',
          state==='slowClearance'
            ?'#0284C7'
            :'#16A34A',
          state==='slowClearance'
            ?'#F0F9FF'
            :'#ECFDF5'
        )
        +'<g transform="translate(51 310)">'
        +'<rect width="651" height="65" rx="17" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="325" y="22" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">当前调节轴相对结果</text>'
        +'<text x="92" y="47" text-anchor="middle" font-size="11" font-weight="900" fill="#6D28D9">TRH '+trh.toFixed(0)+'</text>'
        +'<text x="240" y="47" text-anchor="middle" font-size="11" font-weight="900" fill="#BE185D">TSH '+tsh.toFixed(0)+'</text>'
        +'<text x="411" y="47" text-anchor="middle" font-size="11" font-weight="900" fill="#92400E">甲状腺激素 '+thyroidHormone.toFixed(0)+'</text>'
        +'<text x="570" y="47" text-anchor="middle" font-size="11" font-weight="900" fill="#475569">清除 '+clearance.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='axis'){
        labels.innerHTML=''
          +'<path d="M125 104 L125 78" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="70" y="72" font-size="13" font-weight="900" fill="#5B21B6">下丘脑</text>'
          +'<path d="M380 104 L380 78" stroke="#DB2777" stroke-width="2.5"/>'
          +'<text x="322" y="72" font-size="13" font-weight="900" fill="#9D174D">垂体前叶</text>'
          +'<path d="M635 104 L635 78" stroke="#D97706" stroke-width="2.5"/>'
          +'<text x="587" y="72" font-size="13" font-weight="900" fill="#92400E">甲状腺</text>';
        return;
      }

      if(modeName==='forward'){
        labels.innerHTML=''
          +'<path d="M229 158 L229 91" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="191" y="85" font-size="13" font-weight="900" fill="#5B21B6">TRH</text>'
          +'<path d="M450 158 L450 91" stroke="#DB2777" stroke-width="2.5"/>'
          +'<text x="414" y="85" font-size="13" font-weight="900" fill="#9D174D">TSH</text>'
          +'<path d="M666 158 L666 91" stroke="#D97706" stroke-width="2.5"/>'
          +'<text x="606" y="85" font-size="13" font-weight="900" fill="#92400E">甲状腺激素</text>';
        return;
      }

      if(modeName==='feedback'){
        labels.innerHTML=''
          +'<path d="M246 94 L246 73" stroke="#64748B" stroke-width="2.5"/>'
          +'<text x="185" y="68" font-size="13" font-weight="900" fill="#475569">各级激素信号</text>'
          +'<path d="M606 94 L606 73" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="548" y="68" font-size="13" font-weight="900" fill="#B91C1C">负反馈环路</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M106 99 L106 76" stroke="#7C3AED" stroke-width="2.5"/>'
        +'<text x="54" y="71" font-size="13" font-weight="900" fill="#5B21B6">上游需求</text>'
        +'<path d="M649 99 L649 76" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="592" y="71" font-size="13" font-weight="900" fill="#166534">动态稳态</text>';
    }

    function update(){
      var cold=Number(
        coldInput.value
      );
      var demand=Number(
        demandInput.value
      );
      var hypothalamus=Number(
        hypothalamusInput.value
      );
      var pituitary=Number(
        pituitaryInput.value
      );
      var thyroidCapacity=Number(
        thyroidInput.value
      );
      var feedbackSensitivity=Number(
        feedbackInput.value
      );
      var clearance=Number(
        clearanceInput.value
      );
      var processTime=Number(
        timeInput.value
      );

      coldValue.textContent=
        cold.toFixed(0)+'%';
      demandValue.textContent=
        demand.toFixed(0)+'%';
      hypothalamusValue.textContent=
        hypothalamus.toFixed(0)+'%';
      pituitaryValue.textContent=
        pituitary.toFixed(0)+'%';
      thyroidValue.textContent=
        thyroidCapacity.toFixed(0)+'%';
      feedbackValue.textContent=
        feedbackSensitivity.toFixed(0)+'%';
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
        ?'调节推进：运行中'
        :'调节推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var progress=processTime/100;

      var demandSignal=clamp(
        cold*.43
        +demand*.57,
        0,
        100
      );

      var baseTrh=clamp(
        8
        +demandSignal
        *hypothalamus/100
        *(.20+.80*progress)
        *.94,
        0,
        100
      );

      var baseTsh=clamp(
        7
        +baseTrh
        *pituitary/100
        *.88,
        0,
        100
      );

      var rawThyroidHormone=clamp(
        8
        +baseTsh
        *thyroidCapacity/100
        *(.25+.75*progress)
        *.91
        -clearance*.08,
        0,
        100
      );

      var feedbackPressure=clamp(
        rawThyroidHormone
        *feedbackSensitivity/100
        *(.22+.78*progress),
        0,
        100
      );

      var trh=clamp(
        baseTrh
        -feedbackPressure*.30,
        0,
        100
      );

      var tsh=clamp(
        7
        +trh
        *pituitary/100
        *.86
        -feedbackPressure*.18,
        0,
        100
      );

      var thyroidHormone=clamp(
        8
        +tsh
        *thyroidCapacity/100
        *(.25+.75*progress)
        *.90
        -clearance*.08,
        0,
        100
      );

      var metabolicEffect=clamp(
        28
        +thyroidHormone*.58
        +demand*.19
        -cold*.07,
        0,
        100
      );

      trhText.textContent=
        trh.toFixed(0);
      tshText.textContent=
        tsh.toFixed(0);
      thyroidHormoneText.textContent=
        thyroidHormone.toFixed(0);

      root.style.setProperty(
        '--ta-speed',
        clamp(
          2.45-Math.max(
            trh,
            tsh,
            thyroidHormone
          )/72,
          .60,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='axis'){
        title.textContent=
          '下丘脑—垂体—甲状腺调节轴';

        summary.textContent=
          '观察TRH、TSH和甲状腺激素在不同内分泌器官之间的调节关系。';

        dynamic.innerHTML=renderAxis(
          trh,
          tsh,
          thyroidHormone,
          feedbackPressure
        );

        stageNote.textContent=
          '下丘脑、垂体前叶和甲状腺形成分级调节，甲状腺激素又对上游产生负反馈。';

        renderLabels(mode);
      }else if(mode==='forward'){
        title.textContent=
          'TRH、TSH与甲状腺激素的分级促进过程';

        summary.textContent=
          '观察上一级激素如何促进下一级内分泌腺活动及靶组织效应。';

        dynamic.innerHTML=renderForward(
          progress,
          trh,
          tsh,
          thyroidHormone,
          metabolicEffect
        );

        stageNote.textContent=
          'TRH促进垂体前叶分泌TSH，TSH促进甲状腺合成和释放甲状腺激素。';

        renderLabels(mode);
      }else if(mode==='feedback'){
        title.textContent=
          '循环甲状腺激素的负反馈调节';

        summary.textContent=
          '观察甲状腺激素水平变化如何反向影响下丘脑和垂体信号。';

        dynamic.innerHTML=renderFeedback(
          trh,
          tsh,
          thyroidHormone,
          feedbackPressure,
          metabolicEffect
        );

        stageNote.textContent=
          '甲状腺激素升高时负反馈增强，TRH和TSH分泌倾向受到抑制。';

        renderLabels(mode);
      }else{
        title.textContent=
          '甲状腺调节轴的典型情境比较';

        summary.textContent=
          '比较需求增强、甲状腺能力较低、反馈敏感性较低和接近稳态时的调节方向。';

        dynamic.innerHTML=renderScenarios(
          cold,
          demand,
          trh,
          tsh,
          thyroidHormone,
          thyroidCapacity,
          feedbackSensitivity,
          clearance
        );

        stageNote.textContent=
          '同一激素水平变化可能来自调节轴不同环节，本模型不能用于医学判断。';

        renderLabels(mode);
      }

      var condition=
        '当前上游需求、各级内分泌反应、负反馈和激素清除处于相对协调状态。';

      if(
        cold>78
        &&processTime<28
      ){
        condition=
          '寒冷刺激较强，但调节时间较短，上游信号已经增强，甲状腺激素效应尚未充分形成。';
      }else if(
        hypothalamus<35
        &&demandSignal>60
      ){
        condition=
          '机体调节需求较高，但下丘脑反应能力较低，TRH信号增强幅度受到限制。';
      }else if(
        pituitary<35
        &&trh>42
      ){
        condition=
          'TRH信号已经较强，但垂体前叶反应能力较低，TSH响应受到限制。';
      }else if(
        thyroidCapacity<35
        &&tsh>42
      ){
        condition=
          'TSH刺激较强，但甲状腺合成能力较低，甲状腺激素升高幅度受到限制。';
      }else if(
        feedbackSensitivity<30
        &&thyroidHormone>55
      ){
        condition=
          '甲状腺激素相对较高而负反馈敏感性较低，上游TRH和TSH抑制不足。';
      }else if(
        clearance<28
        &&thyroidHormone>55
      ){
        condition=
          '激素清除效率较低，循环甲状腺激素相对积累，负反馈压力增强。';
      }else if(
        clearance>88
        &&thyroidCapacity<55
      ){
        condition=
          '激素清除较快而甲状腺合成能力一般，循环甲状腺激素水平相对偏低。';
      }else if(
        processTime<14
      ){
        condition=
          '调节过程刚刚开始，激素分级传递和靶组织效应尚未充分形成。';
      }

      var principle=mode==='axis'
        ?'下丘脑分泌TRH，垂体前叶分泌TSH，甲状腺分泌甲状腺激素，三者构成分级调节轴。'
        :mode==='forward'
          ?'TRH促进TSH分泌，TSH促进甲状腺激素合成和释放；TSH本身不是甲状腺激素。'
          :mode==='feedback'
            ?'循环甲状腺激素可抑制下丘脑和垂体相关激素分泌，使原有变化减小，体现负反馈。'
            :'激素调节轴中任一环节的反应、反馈或清除发生变化，都可能改变最终相对激素水平。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前TRH信号 '
        +trh.toFixed(0)
        +'，TSH信号 '
        +tsh.toFixed(0)
        +'，甲状腺激素相对水平 '
        +thyroidHormone.toFixed(0)
        +'；所有数值只用于教学比较，不用于甲状腺功能、激素检测或疾病诊断。';
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

    coldInput.oninput=update;
    demandInput.oninput=update;
    hypothalamusInput.oninput=update;
    pituitaryInput.oninput=update;
    thyroidInput.oninput=update;
    feedbackInput.oninput=update;
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
