/**
 * lifeScienceLabTemplatesHomeostasisInternalEnvironment.ts
 *
 * 平面生命科学实验室：内环境物质交换与稳态。
 *
 * 教学目标：
 * 1. 认识血浆、组织液和淋巴液构成机体细胞直接生活的内环境；
 * 2. 理解血浆与组织液之间通过毛细血管壁发生物质交换；
 * 3. 理解部分组织液进入毛细淋巴管形成淋巴液，最终回流血液；
 * 4. 观察氧气、营养物质、二氧化碳和代谢废物在血液、组织液与细胞之间的运输；
 * 5. 理解肺、消化系统、肝脏和肾脏共同参与内环境稳态维持；
 * 6. 理解内环境稳态是相对稳定的动态状态，而不是完全不变；
 * 7. 观察细胞代谢负荷、循环运输、肺交换、肾脏清除和调节能力对稳态的影响；
 * 8. 理解神经调节、体液调节和免疫调节共同参与稳态维持。
 *
 * 科学边界：
 * 1. 人体内环境主要包括血浆、组织液和淋巴液；
 * 2. 细胞内液不属于内环境，外界环境也不属于内环境；
 * 3. 血细胞直接生活在血浆中，大多数组织细胞直接生活在组织液中；
 * 4. 毛细血管动脉端相对有利于液体滤出，静脉端相对有利于液体回流，
 *    但真实交换还受毛细血管压、血浆胶体渗透压、组织液压力和淋巴回流等影响；
 * 5. 氧气和营养物质通常由血浆经组织液到达组织细胞；
 * 6. 二氧化碳和代谢废物通常由组织细胞经组织液进入血液；
 * 7. 部分组织液进入毛细淋巴管形成淋巴液，淋巴液最终回流血液循环；
 * 8. 肺参与氧气和二氧化碳交换，消化系统吸收营养物质，
 *    肝脏参与物质转化和代谢，肾脏参与水盐和代谢废物调节；
 * 9. 内环境的温度、酸碱度、渗透状态和各种化学成分都只在一定范围内相对稳定；
 * 10. 稳态被打破后，机体可通过负反馈等调节机制减小偏差；
 * 11. 稳态调节能力有限，外界变化过强或持续时间过长时可能超出调节范围；
 * 12. 图中的交换效率、物质浓度、稳态指数和偏差均为相对教学指标；
 * 13. 本模板只用于生物学教学，不用于循环、呼吸、肝肾功能或疾病判断。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript，不依赖外部图片、脚本、样式或CDN；
 * 2. 所有DOM查询均限定在rootId内部，支持同页多个独立实例；
 * 3. 使用生命科学统一.bl-*布局协议；
 * 4. 本文件只导出独立模板数组，聚合入口由后续批次统一接入。
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

function internalEnvironmentStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #A7F3D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#D1FAE5,#E0F2FE);border-bottom:1px solid #A7F3D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FFFC;border-right:1px solid #A7F3D0}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#059669;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981}'
    + '#' + rootId + ' .ie-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .ie-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ie-button{min-height:30px;padding:3px;border:1px solid #6EE7B7;border-radius:8px;background:#fff;color:#065F46;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .ie-button.active{border-color:#10B981;background:#D1FAE5;box-shadow:0 3px 9px rgba(16,185,129,.13)}'
    + '#' + rootId + ' .ie-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ie-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ie-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .ie-card{padding:6px 3px;border:1px solid #A7F3D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ie-card b{display:block;font-size:13px;color:#047857}'
    + '#' + rootId + ' .ie-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#D1FAE5;color:#064E3B;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ie-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--ie-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ie-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_HOMEOSTASIS_INTERNAL_ENVIRONMENT:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-internal-environment-homeostasis',
    group: '💧 排泄与内环境稳态',
    name: '内环境物质交换与稳态',
    emoji: '🔄',
    desc: '调节细胞代谢负荷、毛细血管交换、淋巴回流、肺交换、肾脏清除和调节能力，观察内环境动态稳态',
    params: [
      {
        key: 'cellMetabolicLoad',
        label: '细胞代谢负荷',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'capillaryExchange',
        label: '毛细血管交换效率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'lymphReturn',
        label: '淋巴回流效率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 74,
      },
      {
        key: 'lungExchange',
        label: '肺气体交换效率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'kidneyClearance',
        label: '肾脏清除调节能力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 80,
      },
      {
        key: 'regulationCapacity',
        label: '综合稳态调节能力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'processTime',
        label: '稳态调节过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 56,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const cellMetabolicLoad = num(
        params,
        'cellMetabolicLoad',
        62,
      )
      const capillaryExchange = num(
        params,
        'capillaryExchange',
        78,
      )
      const lymphReturn = num(params, 'lymphReturn', 74)
      const lungExchange = num(params, 'lungExchange', 82)
      const kidneyClearance = num(
        params,
        'kidneyClearance',
        80,
      )
      const regulationCapacity = num(
        params,
        'regulationCapacity',
        76,
      )
      const processTime = num(params, 'processTime', 56)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${internalEnvironmentStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🔄 内环境物质交换与稳态</div>
    <div class="bl-note">交换、浓度和稳态指数均为相对教学值</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>细胞代谢负荷</span>
          <span class="bl-value" data-metabolic-value></span>
        </div>
        <input
          data-metabolic
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(cellMetabolicLoad)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>毛细血管交换效率</span>
          <span class="bl-value" data-capillary-value></span>
        </div>
        <input
          data-capillary
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(capillaryExchange)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>淋巴回流效率</span>
          <span class="bl-value" data-lymph-value></span>
        </div>
        <input
          data-lymph
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(lymphReturn)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>肺气体交换效率</span>
          <span class="bl-value" data-lung-value></span>
        </div>
        <input
          data-lung
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(lungExchange)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>肾脏清除调节能力</span>
          <span class="bl-value" data-kidney-value></span>
        </div>
        <input
          data-kidney
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(kidneyClearance)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>综合稳态调节能力</span>
          <span class="bl-value" data-regulation-value></span>
        </div>
        <input
          data-regulation
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(regulationCapacity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>稳态调节过程时间</span>
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

      <div class="ie-subtitle">观察方式</div>

      <div class="ie-buttons">
        <button
          type="button"
          class="ie-button active"
          data-mode="compartments"
        >内环境组成</button>

        <button
          type="button"
          class="ie-button"
          data-mode="exchange"
        >组织物质交换</button>

        <button
          type="button"
          class="ie-button"
          data-mode="organs"
        >器官协同调节</button>

        <button
          type="button"
          class="ie-button"
          data-mode="homeostasis"
        >动态稳态与反馈</button>
      </div>

      <button
        type="button"
        class="ie-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="ie-toggle"
        data-auto
      >稳态推进：运行中</button>

      <div class="ie-status">
        <div class="ie-card">
          <b data-oxygen-index></b>
          <span>氧供指数</span>
        </div>

        <div class="ie-card">
          <b data-waste-index></b>
          <span>废物负荷</span>
        </div>

        <div class="ie-card">
          <b data-homeostasis-index></b>
          <span>稳态指数</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="内环境物质交换与稳态互动示意图"
      >
        <defs>
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
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#065F46"
              flood-opacity=".13"
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
          fill="#065F46"
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

        <g transform="translate(518 337)">
          <rect
            width="216"
            height="66"
            rx="15"
            fill="#ECFDF5"
            stroke="#A7F3D0"
            stroke-width="2"
          />

          <text
            x="108"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#065F46"
          >关键边界</text>

          <text
            x="108"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#064E3B"
          >稳态是动态的相对稳定</text>

          <text
            x="108"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#064E3B"
          >细胞内液不属于内环境</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#065F46"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var metabolicInput=root.querySelector(
      '[data-metabolic]'
    );
    var capillaryInput=root.querySelector(
      '[data-capillary]'
    );
    var lymphInput=root.querySelector(
      '[data-lymph]'
    );
    var lungInput=root.querySelector(
      '[data-lung]'
    );
    var kidneyInput=root.querySelector(
      '[data-kidney]'
    );
    var regulationInput=root.querySelector(
      '[data-regulation]'
    );
    var timeInput=root.querySelector(
      '[data-time]'
    );

    var metabolicValue=root.querySelector(
      '[data-metabolic-value]'
    );
    var capillaryValue=root.querySelector(
      '[data-capillary-value]'
    );
    var lymphValue=root.querySelector(
      '[data-lymph-value]'
    );
    var lungValue=root.querySelector(
      '[data-lung-value]'
    );
    var kidneyValue=root.querySelector(
      '[data-kidney-value]'
    );
    var regulationValue=root.querySelector(
      '[data-regulation-value]'
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

    var oxygenText=root.querySelector(
      '[data-oxygen-index]'
    );
    var wasteText=root.querySelector(
      '[data-waste-index]'
    );
    var homeostasisText=root.querySelector(
      '[data-homeostasis-index]'
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

    var mode='compartments';
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
      },800);
    }

    function particle(
      kind,
      x,
      y,
      scale,
      opacity
    ){
      var color=kind==='oxygen'
        ?'#EF4444'
        :kind==='nutrient'
          ?'#F59E0B'
          :kind==='carbon'
            ?'#64748B'
            :kind==='waste'
              ?'#8B5CF6'
              :'#38BDF8';

      var label=kind==='oxygen'
        ?'O₂'
        :kind==='nutrient'
          ?'N'
          :kind==='carbon'
            ?'CO₂'
            :kind==='waste'
              ?'W'
              :'H₂O';

      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<circle r="9" fill="'+color+'" stroke="#334155" stroke-width="1.5"/>'
        +'<text x="0" y="3" text-anchor="middle" font-size="6.5" font-weight="900" fill="#FFFFFF">'+label+'</text>'
        +'</g>';
    }

    function tissueCell(
      x,
      y,
      scale,
      load
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')">'
        +'<ellipse rx="47" ry="37" fill="#FCE7F3" stroke="#DB2777" stroke-width="5"/>'
        +'<ellipse rx="19" ry="15" fill="#DDD6FE" stroke="#7C3AED" stroke-width="3"/>'
        +'<circle class="ie-pulse" r="'+(51+load*.12)+'" fill="none" stroke="#EC4899" stroke-width="3" stroke-dasharray="7 6" opacity="'+(.22+.60*load/100)+'"/>'
        +'</g>';
    }

    function renderCompartments(
      capillary,
      lymph,
      progress
    ){
      var plasmaParticles='';
      var tissueParticles='';
      var lymphParticles='';

      var plasmaCount=Math.floor(
        5+capillary/10
      );
      var tissueCount=Math.floor(
        4+capillary/13
      );
      var lymphCount=Math.floor(
        2+lymph/16
      );

      for(var i=0;i<plasmaCount;i++){
        var px=83+(i%5)*37;
        var py=139+Math.floor(i/5)*40;
        var pKind=i%3===0
          ?'oxygen'
          :i%3===1
            ?'nutrient'
            :'water';

        plasmaParticles+=particle(
          pKind,
          px,
          py,
          .62,
          .76
        );
      }

      for(var j=0;j<tissueCount;j++){
        var tx=314+(j%5)*40;
        var ty=148+Math.floor(j/5)*44;
        var tKind=j%4===0
          ?'oxygen'
          :j%4===1
            ?'nutrient'
            :j%4===2
              ?'carbon'
              :'water';

        tissueParticles+=particle(
          tKind,
          tx,
          ty,
          .58,
          .70
        );
      }

      for(var k=0;k<lymphCount;k++){
        var lx=590+(k%3)*34;
        var ly=160+Math.floor(k/3)*41;

        lymphParticles+=particle(
          k%2===0?'water':'waste',
          lx,
          ly,
          .55,
          .48+.45*progress
        );
      }

      return ''
        +'<g transform="translate(27 91)">'
        +'<rect width="220" height="243" rx="23" fill="#EFF6FF" stroke="#93C5FD" stroke-width="4"/>'
        +'<text x="110" y="30" text-anchor="middle" font-size="16" font-weight="900" fill="#1D4ED8">血浆</text>'
        +'<path d="M22 86 C70 61 150 113 198 82 M22 164 C74 136 144 193 198 159" fill="none" stroke="#DC2626" stroke-width="16" stroke-linecap="round"/>'
        +plasmaParticles
        +'<text x="110" y="220" text-anchor="middle" font-size="11" font-weight="900" fill="#1E40AF">血细胞直接生活在血浆中</text>'
        +'</g>'
        +'<path class="ie-flow" d="M250 208 H291" fill="none" stroke="#2563EB" stroke-width="'+(3+capillary/22)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<g transform="translate(292 91)">'
        +'<rect width="220" height="243" rx="23" fill="#ECFDF5" stroke="#86EFAC" stroke-width="4"/>'
        +'<text x="110" y="30" text-anchor="middle" font-size="16" font-weight="900" fill="#047857">组织液</text>'
        +tissueCell(
          110,
          142,
          .90,
          capillary
        )
        +tissueParticles
        +'<text x="110" y="220" text-anchor="middle" font-size="11" font-weight="900" fill="#166534">大多数组织细胞直接生活在组织液中</text>'
        +'</g>'
        +'<path class="ie-flow" d="M514 208 H555" fill="none" stroke="#16A34A" stroke-width="'+(3+lymph/22)+'" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<g transform="translate(556 91)">'
        +'<rect width="178" height="243" rx="23" fill="#FAF5FF" stroke="#C4B5FD" stroke-width="4"/>'
        +'<text x="89" y="30" text-anchor="middle" font-size="16" font-weight="900" fill="#6D28D9">淋巴液</text>'
        +'<path d="M35 72 C73 110 70 177 111 210 C135 229 145 205 141 170" fill="none" stroke="#8B5CF6" stroke-width="18" stroke-linecap="round"/>'
        +lymphParticles
        +'<text x="89" y="220" text-anchor="middle" font-size="10.5" font-weight="900" fill="#5B21B6">部分组织液进入淋巴管后形成</text>'
        +'</g>'
        +'<path class="ie-flow" d="M666 336 C623 376 257 380 164 342" fill="none" stroke="#7C3AED" stroke-width="'+(3+lymph/24)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<text x="405" y="374" text-anchor="middle" font-size="11.5" font-weight="900" fill="#5B21B6">淋巴液最终回流血液循环</text>';
    }

    function renderExchange(
      metabolic,
      capillary,
      lymph,
      oxygenSupply,
      wasteLoad,
      progress
    ){
      var arterialParticles='';
      var tissueParticles='';
      var venousParticles='';

      var arterialCount=Math.floor(
        4+oxygenSupply/10
      );

      for(var i=0;i<arterialCount;i++){
        var ax=78+(i%5)*39;
        var ay=141+Math.floor(i/5)*38;

        arterialParticles+=particle(
          i%2===0?'oxygen':'nutrient',
          ax,
          ay,
          .60,
          .80
        );
      }

      var tissueCount=Math.floor(
        4+metabolic/9
      );

      for(var j=0;j<tissueCount;j++){
        var tx=313+(j%5)*42;
        var ty=132+Math.floor(j/5)*42;

        tissueParticles+=particle(
          j%2===0?'carbon':'waste',
          tx,
          ty,
          .58,
          .48+.45*metabolic/100
        );
      }

      var venousCount=Math.floor(
        3+wasteLoad/11
      );

      for(var k=0;k<venousCount;k++){
        var vx=552+(k%5)*38;
        var vy=144+Math.floor(k/5)*39;

        venousParticles+=particle(
          k%2===0?'carbon':'waste',
          vx,
          vy,
          .59,
          .72
        );
      }

      var exchangeStrength=capillary/100
        *(.22+.78*progress);

      var edemaRisk=clamp(
        (100-lymph)
        *capillary/100
        *.82,
        0,
        100
      );

      return ''
        +'<rect x="28" y="91" width="704" height="274" rx="25" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<g transform="translate(43 108)">'
        +'<rect width="211" height="213" rx="20" fill="#FFF1F2" stroke="#FCA5A5" stroke-width="4"/>'
        +'<text x="105" y="28" text-anchor="middle" font-size="15" font-weight="900" fill="#991B1B">毛细血管动脉端</text>'
        +'<path d="M24 90 C71 57 135 119 187 84" fill="none" stroke="#DC2626" stroke-width="20" stroke-linecap="round"/>'
        +arterialParticles
        +'<text x="105" y="190" text-anchor="middle" font-size="10.5" font-weight="900" fill="#991B1B">氧气与营养物质较多</text>'
        +'</g>'
        +'<g transform="translate(274 108)">'
        +'<rect width="211" height="213" rx="20" fill="#ECFDF5" stroke="#86EFAC" stroke-width="4"/>'
        +'<text x="105" y="28" text-anchor="middle" font-size="15" font-weight="900" fill="#047857">组织液与组织细胞</text>'
        +tissueCell(
          105,
          104,
          .86,
          metabolic
        )
        +tissueParticles
        +'<text x="105" y="190" text-anchor="middle" font-size="10.5" font-weight="900" fill="#166534">细胞不断进行物质交换和代谢</text>'
        +'</g>'
        +'<g transform="translate(505 108)">'
        +'<rect width="211" height="213" rx="20" fill="#EFF6FF" stroke="#93C5FD" stroke-width="4"/>'
        +'<text x="105" y="28" text-anchor="middle" font-size="15" font-weight="900" fill="#1D4ED8">毛细血管静脉端</text>'
        +'<path d="M24 90 C71 57 135 119 187 84" fill="none" stroke="#2563EB" stroke-width="20" stroke-linecap="round"/>'
        +venousParticles
        +'<text x="105" y="190" text-anchor="middle" font-size="10.5" font-weight="900" fill="#1E40AF">二氧化碳和代谢废物较多</text>'
        +'</g>'
        +'<path class="ie-flow" d="M245 210 H278" fill="none" stroke="#DC2626" stroke-width="'+(3+exchangeStrength*6)+'" marker-end="url(#${rootId}-arrow-red)"/>'
        +'<path class="ie-flow" d="M486 238 H511" fill="none" stroke="#2563EB" stroke-width="'+(3+exchangeStrength*6)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="ie-flow" d="M380 316 C399 345 468 345 502 313" fill="none" stroke="#7C3AED" stroke-width="'+(3+lymph/25)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<text x="147" y="349" text-anchor="middle" font-size="11" font-weight="900" fill="#991B1B">血浆 → 组织液 → 细胞</text>'
        +'<text x="598" y="349" text-anchor="middle" font-size="11" font-weight="900" fill="#1E40AF">细胞 → 组织液 → 血浆</text>'
        +'<text x="393" y="381" text-anchor="middle" font-size="11" font-weight="900" fill="'+(edemaRisk>45?'#B91C1C':'#5B21B6')+'">组织液积聚风险指数 '+edemaRisk.toFixed(0)+'</text>';
    }

    function organCard(
      x,
      titleText,
      color,
      fill,
      value,
      line1,
      line2
    ){
      return ''
        +'<g transform="translate('+x+' 100)">'
        +'<rect width="164" height="194" rx="20" fill="'+fill+'" stroke="'+color+'" stroke-width="4"/>'
        +'<circle cx="82" cy="48" r="30" fill="#FFFFFF" stroke="'+color+'" stroke-width="4"/>'
        +'<text x="82" y="55" text-anchor="middle" font-size="17" font-weight="900" fill="'+color+'">'+titleText.substring(0,1)+'</text>'
        +'<text x="82" y="99" text-anchor="middle" font-size="14" font-weight="900" fill="'+color+'">'+titleText+'</text>'
        +'<text x="82" y="127" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'+line1+'</text>'
        +'<text x="82" y="148" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'+line2+'</text>'
        +'<text x="82" y="176" text-anchor="middle" font-size="12" font-weight="900" fill="'+color+'">相对功能 '+value.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderOrgans(
      lung,
      kidney,
      metabolic,
      regulation,
      progress
    ){
      var digestive=clamp(
        55+regulation*.35,
        0,
        100
      );

      var liver=clamp(
        48+regulation*.42
        -metabolic*.12,
        0,
        100
      );

      return ''
        +organCard(
          25,
          '肺',
          '#0284C7',
          '#F0F9FF',
          lung,
          '吸收氧气',
          '排出二氧化碳'
        )
        +organCard(
          205,
          '消化系统',
          '#D97706',
          '#FFFBEB',
          digestive,
          '吸收营养物质',
          '进入血液运输'
        )
        +organCard(
          385,
          '肝脏',
          '#7C3AED',
          '#FAF5FF',
          liver,
          '物质转化与储存',
          '参与代谢和解毒'
        )
        +organCard(
          565,
          '肾脏',
          '#16A34A',
          '#ECFDF5',
          kidney,
          '调节水盐',
          '排出代谢废物'
        )
        +'<path class="ie-flow" d="M107 305 C176 352 573 352 647 305" fill="none" stroke="#DC2626" stroke-width="'+(3+regulation/24)+'" marker-end="url(#${rootId}-arrow-red)"/>'
        +'<path class="ie-flow" d="M647 318 C571 382 182 382 107 318" fill="none" stroke="#2563EB" stroke-width="'+(3+progress*5)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="376" y="338" text-anchor="middle" font-size="11.5" font-weight="900" fill="#991B1B">血液把氧气和营养物质运送到组织</text>'
        +'<text x="376" y="377" text-anchor="middle" font-size="11.5" font-weight="900" fill="#1E40AF">血液把二氧化碳和代谢废物运送到肺、肝和肾等器官</text>';
    }

    function homeostasisBar(
      x,
      y,
      width,
      value,
      color,
      label,
      reference
    ){
      var markerX=x+width*reference/100;

      return ''
        +'<text x="'+x+'" y="'+(y-8)+'" font-size="11" font-weight="900" fill="#475569">'+label+'</text>'
        +'<rect x="'+x+'" y="'+y+'" width="'+width+'" height="16" rx="8" fill="#E2E8F0"/>'
        +'<rect x="'+x+'" y="'+y+'" width="'+(width*clamp(value,0,100)/100)+'" height="16" rx="8" fill="'+color+'"/>'
        +'<line x1="'+markerX+'" y1="'+(y-5)+'" x2="'+markerX+'" y2="'+(y+23)+'" stroke="#10B981" stroke-width="3"/>'
        +'<text x="'+(x+width)+'" y="'+(y+14)+'" text-anchor="end" font-size="10" font-weight="900" fill="#334155">'+value.toFixed(0)+'</text>';
    }

    function renderHomeostasis(
      oxygen,
      waste,
      fluidBalance,
      homeostasis,
      regulation,
      progress
    ){
      var correctedOxygen=clamp(
        oxygen
        +(70-oxygen)
        *regulation/100
        *progress
        *.55,
        0,
        100
      );

      var correctedWaste=clamp(
        waste
        -(waste-25)
        *regulation/100
        *progress
        *.58,
        0,
        100
      );

      var correctedFluid=clamp(
        fluidBalance
        +(55-fluidBalance)
        *regulation/100
        *progress
        *.50,
        0,
        100
      );

      return ''
        +'<g transform="translate(34 94)">'
        +'<rect width="455" height="264" rx="23" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="227" y="31" text-anchor="middle" font-size="15" font-weight="900" fill="#334155">内环境主要指标的动态变化</text>'
        +homeostasisBar(34,78,382,oxygen,'#EF4444','调节前氧供',70)
        +homeostasisBar(34,128,382,correctedOxygen,'#F87171','调节后氧供',70)
        +homeostasisBar(34,178,382,waste,'#8B5CF6','调节前废物负荷',25)
        +homeostasisBar(34,228,382,correctedWaste,'#A78BFA','调节后废物负荷',25)
        +'</g>'
        +'<g transform="translate(510 94)">'
        +'<rect width="217" height="264" rx="23" fill="#ECFDF5" stroke="#A7F3D0" stroke-width="3"/>'
        +'<text x="108" y="31" text-anchor="middle" font-size="15" font-weight="900" fill="#065F46">负反馈调节</text>'
        +'<circle cx="108" cy="104" r="54" fill="#FFFFFF" stroke="#10B981" stroke-width="6"/>'
        +'<text x="108" y="96" text-anchor="middle" font-size="15" font-weight="900" fill="#047857">稳态指数</text>'
        +'<text x="108" y="124" text-anchor="middle" font-size="26" font-weight="900" fill="#047857">'+homeostasis.toFixed(0)+'</text>'
        +'<path class="ie-flow" d="M52 181 C83 153 136 153 165 181 C189 205 170 235 140 231" fill="none" stroke="#10B981" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>'
        +'<text x="108" y="202" text-anchor="middle" font-size="11" font-weight="900" fill="#166534">偏差 → 感受 → 调节</text>'
        +'<text x="108" y="223" text-anchor="middle" font-size="11" font-weight="900" fill="#166534">效应 → 偏差减小</text>'
        +'<text x="108" y="248" text-anchor="middle" font-size="10" font-weight="900" fill="#475569">调节后体液平衡 '+correctedFluid.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='compartments'){
        labels.innerHTML=''
          +'<path d="M137 91 L137 72" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="93" y="67" font-size="13" font-weight="900" fill="#1D4ED8">血浆</text>'
          +'<path d="M402 91 L402 72" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="356" y="67" font-size="13" font-weight="900" fill="#166534">组织液</text>'
          +'<path d="M645 91 L645 72" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="599" y="67" font-size="13" font-weight="900" fill="#5B21B6">淋巴液</text>';
        return;
      }

      if(modeName==='exchange'){
        labels.innerHTML=''
          +'<path d="M250 181 L286 151" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="294" y="149" font-size="13" font-weight="900" fill="#991B1B">氧气和营养物质</text>'
          +'<path d="M488 252 L535 284" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="543" y="290" font-size="13" font-weight="900" fill="#1E40AF">二氧化碳和废物</text>'
          +'<path d="M380 318 L378 350" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="301" y="367" font-size="13" font-weight="900" fill="#5B21B6">淋巴回流</text>';
        return;
      }

      if(modeName==='organs'){
        labels.innerHTML=''
          +'<path d="M107 100 L107 76" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="57" y="71" font-size="13" font-weight="900" fill="#0369A1">气体交换</text>'
          +'<path d="M287 100 L287 76" stroke="#D97706" stroke-width="2.5"/>'
          +'<text x="227" y="71" font-size="13" font-weight="900" fill="#92400E">营养物质吸收</text>'
          +'<path d="M647 100 L647 76" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="584" y="71" font-size="13" font-weight="900" fill="#166534">水盐和废物调节</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M261 94 L261 73" stroke="#64748B" stroke-width="2.5"/>'
        +'<text x="185" y="68" font-size="13" font-weight="900" fill="#475569">内环境指标变化</text>'
        +'<path d="M618 94 L618 73" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="556" y="68" font-size="13" font-weight="900" fill="#166534">负反馈环路</text>';
    }

    function update(){
      var metabolic=Number(
        metabolicInput.value
      );
      var capillary=Number(
        capillaryInput.value
      );
      var lymph=Number(
        lymphInput.value
      );
      var lung=Number(
        lungInput.value
      );
      var kidney=Number(
        kidneyInput.value
      );
      var regulation=Number(
        regulationInput.value
      );
      var processTime=Number(
        timeInput.value
      );

      metabolicValue.textContent=
        metabolic.toFixed(0)+'%';
      capillaryValue.textContent=
        capillary.toFixed(0)+'%';
      lymphValue.textContent=
        lymph.toFixed(0)+'%';
      lungValue.textContent=
        lung.toFixed(0)+'%';
      kidneyValue.textContent=
        kidney.toFixed(0)+'%';
      regulationValue.textContent=
        regulation.toFixed(0)+'%';
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
        ?'稳态推进：运行中'
        :'稳态推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var progress=processTime/100;

      var oxygenSupply=100*Math.sqrt(
        lung/100
        *(.20+.80*capillary/100)
        *(.20+.80*progress)
      );

      oxygenSupply=clamp(
        oxygenSupply
        -metabolic*.18,
        0,
        100
      );

      var wasteProduction=metabolic
        *(.35+.65*progress);

      var wasteClearance=100*Math.sqrt(
        kidney/100
        *(.20+.80*capillary/100)
        *(.20+.80*progress)
      );

      var wasteLoad=clamp(
        22
        +wasteProduction*.72
        -wasteClearance*.55,
        0,
        100
      );

      var fluidBalance=clamp(
        55
        +lymph*.36
        +kidney*.18
        -capillary*.12
        -metabolic*.08,
        0,
        100
      );

      var oxygenDeviation=Math.abs(
        70-oxygenSupply
      );
      var wasteDeviation=Math.abs(
        25-wasteLoad
      );
      var fluidDeviation=Math.abs(
        55-fluidBalance
      );

      var rawDeviation=clamp(
        oxygenDeviation*.42
        +wasteDeviation*.34
        +fluidDeviation*.24,
        0,
        100
      );

      var correctedDeviation=rawDeviation
        *(1-regulation/100
        *(.22+.78*progress)
        *.72);

      var homeostasisIndex=clamp(
        100-correctedDeviation,
        0,
        100
      );

      oxygenText.textContent=
        oxygenSupply.toFixed(0);
      wasteText.textContent=
        wasteLoad.toFixed(0);
      homeostasisText.textContent=
        homeostasisIndex.toFixed(0);

      root.style.setProperty(
        '--ie-speed',
        clamp(
          2.45-Math.max(
            capillary,
            regulation
          )/75,
          .65,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='compartments'){
        title.textContent=
          '血浆、组织液和淋巴液构成内环境';

        summary.textContent=
          '观察三种细胞外液之间的转化、交换与回流关系。';

        dynamic.innerHTML=renderCompartments(
          capillary,
          lymph,
          progress
        );

        stageNote.textContent=
          '细胞内液不属于内环境；多数组织细胞直接生活在组织液中。';

        renderLabels(mode);
      }else if(mode==='exchange'){
        title.textContent=
          '毛细血管、组织液与组织细胞之间的物质交换';

        summary.textContent=
          '观察氧气、营养物质、二氧化碳和代谢废物的运输方向。';

        dynamic.innerHTML=renderExchange(
          metabolic,
          capillary,
          lymph,
          oxygenSupply,
          wasteLoad,
          progress
        );

        stageNote.textContent=
          '毛细血管壁、组织液和淋巴回流共同参与局部物质交换。';

        renderLabels(mode);
      }else if(mode==='organs'){
        title.textContent=
          '肺、消化系统、肝脏和肾脏协同维持内环境';

        summary.textContent=
          '观察不同器官如何向内环境补充物质、转化物质和清除废物。';

        dynamic.innerHTML=renderOrgans(
          lung,
          kidney,
          metabolic,
          regulation,
          progress
        );

        stageNote.textContent=
          '内环境稳态依赖循环系统连接多个器官共同调节。';

        renderLabels(mode);
      }else{
        title.textContent=
          '内环境动态稳态与负反馈调节';

        summary.textContent=
          '观察氧供、废物负荷和体液平衡偏离后如何被调节回相对稳定范围。';

        dynamic.innerHTML=renderHomeostasis(
          oxygenSupply,
          wasteLoad,
          fluidBalance,
          homeostasisIndex,
          regulation,
          progress
        );

        stageNote.textContent=
          '稳态是不断发生物质交换和调节条件下的动态相对稳定。';

        renderLabels(mode);
      }

      var condition=
        '当前细胞代谢、循环交换、肺交换、肾脏清除和综合调节相对协调。';

      if(
        metabolic>82
        &&lung<45
      ){
        condition=
          '细胞代谢负荷较高而肺气体交换效率较低，组织氧供相对不足。';
      }else if(
        metabolic>82
        &&kidney<45
      ){
        condition=
          '细胞代谢负荷较高而肾脏清除调节能力较低，代谢废物负荷升高。';
      }else if(
        capillary<25
      ){
        condition=
          '毛细血管交换效率较低，血浆、组织液和组织细胞之间的物质运输受到限制。';
      }else if(
        lymph<25
        &&capillary>55
      ){
        condition=
          '毛细血管交换较活跃但淋巴回流效率较低，组织液积聚风险上升。';
      }else if(
        regulation<20
        &&rawDeviation>25
      ){
        condition=
          '综合稳态调节能力较低，内环境偏差难以被及时减小。';
      }else if(
        processTime<15
      ){
        condition=
          '稳态调节过程时间较短，物质交换变化已经出现，但反馈调节效应尚未充分形成。';
      }else if(
        homeostasisIndex<45
      ){
        condition=
          '多项内环境指标偏离相对稳定范围，当前调节能力不足以完全抵消扰动。';
      }

      var principle=mode==='compartments'
        ?'人体内环境主要包括血浆、组织液和淋巴液；细胞内液不属于内环境。'
        :mode==='exchange'
          ?'氧气和营养物质通常由血浆经组织液到达组织细胞，二氧化碳和代谢废物通常由组织细胞经组织液进入血液。'
          :mode==='organs'
            ?'肺、消化系统、肝脏和肾脏通过循环系统相互连接，共同参与物质补充、转化和清除。'
            :'内环境稳态是温度、酸碱度、渗透状态和各种化学成分在一定范围内保持动态相对稳定。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前氧供指数 '
        +oxygenSupply.toFixed(0)
        +'，废物负荷 '
        +wasteLoad.toFixed(0)
        +'，稳态指数 '
        +homeostasisIndex.toFixed(0)
        +'；所有数值仅用于教学比较，不用于循环、呼吸、肝肾功能或疾病判断。';
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

    metabolicInput.oninput=update;
    capillaryInput.oninput=update;
    lymphInput.oninput=update;
    lungInput.oninput=update;
    kidneyInput.oninput=update;
    regulationInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
