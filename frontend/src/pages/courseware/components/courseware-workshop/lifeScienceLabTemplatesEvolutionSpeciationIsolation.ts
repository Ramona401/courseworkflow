/**
 * lifeScienceLabTemplatesEvolutionSpeciationIsolation.ts
 *
 * 平面生命科学实验室：物种形成与生殖隔离。
 *
 * 教学目标：
 * 1. 理解同一种群在连通状态下可以通过基因交流保持相对一致；
 * 2. 理解地理屏障可以减少两个种群之间的基因交流；
 * 3. 理解不同环境选择和遗传漂变可使隔离种群逐渐积累遗传差异；
 * 4. 理解地理隔离本身不等于已经形成新物种；
 * 5. 通过再次接触观察交配兼容度和生殖隔离程度；
 * 6. 理解生殖隔离是物种形成的重要标志之一；
 * 7. 理解物种形成通常是多代积累的种群过程。
 *
 * 教学边界：
 * 1. 本模型只演示异域物种形成的一种简化路径；
 * 2. 地理隔离会减少基因交流，但不会自动、立即形成新物种；
 * 3. 环境差异不会按需要定向产生有利变异；
 * 4. 遗传漂变在小种群中可能更明显，但方向具有随机性；
 * 5. 模型中的遗传差异指数和繁殖兼容度是相对教学指标；
 * 6. 不存在适用于所有生物的统一物种形成阈值；
 * 7. 真实物种形成还可能涉及突变、性选择、染色体变化和生态位分化；
 * 8. 生殖隔离可以发生在受精前，也可以表现为杂交后代存活或繁殖能力下降；
 * 9. 本模型不用于真实物种鉴定或进化时间预测。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式或图片；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用.bl-*公共类名，兼容生命科学实验室底部课堂控制条；
 * 5. 支持同一课件页放置多个实例。
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
 * 把数值转换为适合写入HTML属性的短字符串。
 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/**
 * 构建完全限定到当前rootId的样式。
 */
function speciationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#ECFDF5);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:244px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#F8FDFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-button{height:34px;padding:0 3px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:10px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#0EA5E9;background:#E0F2FE;box-shadow:0 3px 9px rgba(14,165,233,.14)}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:14px;color:#0369A1;min-height:20px}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .si-organism{animation:' + rootId + '-organism var(--si-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .si-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--si-flow-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .si-pulse{animation:' + rootId + '-pulse 1.1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-organism{from{transform:translateY(3px);opacity:.58}to{transform:translateY(-4px);opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.28}to{opacity:.9}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_SPECIATION_ISOLATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-speciation-isolation',
    group: '🦋 进化与生物多样性',
    name: '物种形成与生殖隔离',
    emoji: '🏝️',
    desc: '调节地理隔离、环境差异、遗传漂变和世代数，观察基因交流减少及生殖隔离形成',
    params: [
      {
        key: 'isolationStrength',
        label: '地理隔离强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 82,
        hint: '隔离越强，两个种群之间的实际基因交流通常越少',
      },
      {
        key: 'environmentDifference',
        label: '两地环境差异',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'geneticDrift',
        label: '遗传漂变影响',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 48,
        hint: '表示随机因素造成种群遗传组成变化的相对程度',
      },
      {
        key: 'generations',
        label: '隔离世代数',
        type: 'number',
        min: 1,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
    ],

    buildHTML: (params, rootId) => {
      const isolationStrength = num(
        params,
        'isolationStrength',
        82,
      )
      const environmentDifference = num(
        params,
        'environmentDifference',
        78,
      )
      const geneticDrift = num(
        params,
        'geneticDrift',
        48,
      )
      const generations = num(
        params,
        'generations',
        72,
      )

      return `
<div id="${rootId}">
${speciationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🏝️ 物种形成与生殖隔离</div>
    <div class="bl-note">地理隔离不等于立即形成新物种</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>地理隔离强度</span>
          <span class="bl-value" data-isolation-value></span>
        </div>
        <input
          data-isolation
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(isolationStrength)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>两地环境差异</span>
          <span class="bl-value" data-environment-value></span>
        </div>
        <input
          data-environment
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(environmentDifference)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>遗传漂变影响</span>
          <span class="bl-value" data-drift-value></span>
        </div>
        <input
          data-drift
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(geneticDrift)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>隔离世代数</span>
          <span class="bl-value" data-generations-value></span>
        </div>
        <input
          data-generations
          type="range"
          min="1"
          max="100"
          step="1"
          value="${n(generations)}"
        >
      </div>

      <div class="bl-subtitle">观察阶段</div>

      <div class="bl-buttons">
        <button
          type="button"
          class="bl-button active"
          data-stage="connected"
        >连通种群</button>

        <button
          type="button"
          class="bl-button"
          data-stage="isolated"
        >地理隔离</button>

        <button
          type="button"
          class="bl-button"
          data-stage="contact"
        >再次接触</button>
      </div>

      <div class="bl-status">
        <div class="bl-card">
          <b data-divergence></b>
          <span>遗传差异指数</span>
        </div>

        <div class="bl-card">
          <b data-compatibility></b>
          <span>繁殖兼容度</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="物种形成与生殖隔离互动模型"
      >
        <defs>
          <linearGradient
            id="${rootId}-left-habitat"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#FEF3C7"/>
            <stop offset="100%" stop-color="#FDE68A"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-right-habitat"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#DBEAFE"/>
            <stop offset="100%" stop-color="#BFDBFE"/>
          </linearGradient>

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
              dy="4"
              stdDeviation="5"
              flood-color="#075985"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="22"
          y="33"
          data-title
          font-size="23"
          font-weight="900"
          fill="#075985"
        ></text>

        <text
          x="22"
          y="60"
          data-summary
          font-size="13"
          font-weight="800"
          fill="#475569"
        ></text>

        <!-- 两个种群及其生境 -->
        <g filter="url(#${rootId}-shadow)">
          <rect
            x="22"
            y="78"
            width="270"
            height="146"
            rx="18"
            fill="url(#${rootId}-left-habitat)"
            stroke="#F59E0B"
            stroke-width="3"
          />

          <rect
            x="388"
            y="78"
            width="270"
            height="146"
            rx="18"
            fill="url(#${rootId}-right-habitat)"
            stroke="#3B82F6"
            stroke-width="3"
          />
        </g>

        <text
          x="38"
          y="102"
          data-left-label
          font-size="13"
          font-weight="900"
          fill="#92400E"
        ></text>

        <text
          x="404"
          y="102"
          data-right-label
          font-size="13"
          font-weight="900"
          fill="#1D4ED8"
        ></text>

        <g data-left-population></g>
        <g data-right-population></g>
        <g data-barrier-layer></g>
        <g data-gene-flow-layer></g>
        <g data-contact-layer></g>

        <!-- 差异随世代变化曲线 -->
        <text
          x="22"
          y="258"
          font-size="13"
          font-weight="900"
          fill="#334155"
        >遗传差异随隔离世代积累</text>

        <line
          x1="62"
          y1="356"
          x2="636"
          y2="356"
          stroke="#64748B"
          stroke-width="2.5"
        />

        <line
          x1="62"
          y1="356"
          x2="62"
          y2="274"
          stroke="#64748B"
          stroke-width="2.5"
        />

        <g data-graph-grid></g>

        <path
          data-divergence-area
          fill="#BAE6FD"
          opacity=".45"
        ></path>

        <path
          data-divergence-curve
          fill="none"
          stroke="#0284C7"
          stroke-width="5"
          stroke-linecap="round"
          stroke-linejoin="round"
        ></path>

        <g data-divergence-points></g>

        <line
          data-reference-line
          x1="62"
          y1="307"
          x2="636"
          y2="307"
          stroke="#F59E0B"
          stroke-width="2.5"
          stroke-dasharray="8 7"
        />

        <text
          data-reference-label
          x="474"
          y="298"
          font-size="10.5"
          font-weight="900"
          fill="#B45309"
        >生殖隔离增强区（教学参考）</text>

        <text
          x="604"
          y="377"
          font-size="10.5"
          font-weight="800"
          fill="#64748B"
        >世代</text>

        <text
          x="18"
          y="282"
          font-size="10.5"
          font-weight="800"
          fill="#64748B"
        >差异</text>

        <text
          x="420"
          y="398"
          data-stage-note
          font-size="11"
          font-weight="900"
          fill="#0369A1"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var isolationInput=root.querySelector(
      '[data-isolation]'
    );
    var environmentInput=root.querySelector(
      '[data-environment]'
    );
    var driftInput=root.querySelector(
      '[data-drift]'
    );
    var generationsInput=root.querySelector(
      '[data-generations]'
    );

    var isolationValue=root.querySelector(
      '[data-isolation-value]'
    );
    var environmentValue=root.querySelector(
      '[data-environment-value]'
    );
    var driftValue=root.querySelector(
      '[data-drift-value]'
    );
    var generationsValue=root.querySelector(
      '[data-generations-value]'
    );

    var buttons=root.querySelectorAll('[data-stage]');
    var divergenceValue=root.querySelector(
      '[data-divergence]'
    );
    var compatibilityValue=root.querySelector(
      '[data-compatibility]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var leftLabel=root.querySelector(
      '[data-left-label]'
    );
    var rightLabel=root.querySelector(
      '[data-right-label]'
    );

    var leftPopulation=root.querySelector(
      '[data-left-population]'
    );
    var rightPopulation=root.querySelector(
      '[data-right-population]'
    );
    var barrierLayer=root.querySelector(
      '[data-barrier-layer]'
    );
    var geneFlowLayer=root.querySelector(
      '[data-gene-flow-layer]'
    );
    var contactLayer=root.querySelector(
      '[data-contact-layer]'
    );

    var graphGrid=root.querySelector(
      '[data-graph-grid]'
    );
    var divergenceArea=root.querySelector(
      '[data-divergence-area]'
    );
    var divergenceCurve=root.querySelector(
      '[data-divergence-curve]'
    );
    var divergencePoints=root.querySelector(
      '[data-divergence-points]'
    );
    var referenceLine=root.querySelector(
      '[data-reference-line]'
    );
    var referenceLabel=root.querySelector(
      '[data-reference-label]'
    );
    var stageNote=root.querySelector(
      '[data-stage-note]'
    );

    var stage='connected';

    var stageInfo={
      connected:{
        title:'连通种群：基因交流维持相对一致',
        summary:'个体可以在两个区域之间迁移和繁殖，等位基因持续交流',
        note:'基因交流较充分，两个区域仍属于同一繁殖种群'
      },
      isolated:{
        title:'地理隔离：基因交流减少',
        summary:'屏障降低迁移和交配机会，不同环境和随机因素使差异逐代积累',
        note:'地理隔离只是物种形成的可能起点'
      },
      contact:{
        title:'再次接触：检验是否形成生殖隔离',
        summary:'屏障消失后，观察两个种群能否正常交配并产生可育后代',
        note:'生殖隔离增强时，基因交流仍可能保持较低水平'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    /**
     * 估算地理屏障作用后的相对基因交流。
     *
     * 地理隔离越强，跨区域迁移和繁殖机会通常越少；
     * 但该函数只是教学示意，不代表真实迁移率。
     */
    function calculateGeneFlow(
      isolationStrength
    ){
      var isolation=
        isolationStrength/100;

      return clamp(
        100*(1-.95*isolation),
        5,
        100
      );
    }

    /**
     * 计算隔离种群遗传差异的最终教学指数。
     *
     * 差异积累需要时间，并同时受到：
     * - 地理隔离造成的基因交流减少；
     * - 两地环境选择差异；
     * - 遗传漂变等随机因素。
     */
    function calculateFinalDivergence(
      isolationStrength,
      environmentDifference,
      driftStrength,
      generationCount
    ){
      var isolation=
        isolationStrength/100;
      var environment=
        environmentDifference/100;
      var drift=
        driftStrength/100;
      var timeFactor=
        1-Math.exp(-generationCount/28);

      var evolutionaryForces=
        .2+.8*(.7*environment+.3*drift);

      var isolationEffect=
        isolation*(.5+.5*isolation);

      return clamp(
        100
        *timeFactor
        *evolutionaryForces
        *isolationEffect,
        0,
        100
      );
    }

    /**
     * 生殖兼容度与遗传差异负相关，
     * 但本模型不设定真实物种形成的统一阈值。
     */
    function calculateCompatibility(
      divergence
    ){
      return clamp(
        100-divergence*1.08,
        0,
        100
      );
    }

    /**
     * 生成从第0代到设定世代的差异积累记录。
     */
    function calculateRecords(
      isolationStrength,
      environmentDifference,
      driftStrength,
      generationCount
    ){
      var records=[];

      for(var generation=0;
        generation<=generationCount;
        generation++
      ){
        records.push({
          generation:generation,
          divergence:
            calculateFinalDivergence(
              isolationStrength,
              environmentDifference,
              driftStrength,
              generation
            )
        });
      }

      return records;
    }

    function traitColor(trait){
      var hue=205+trait*.8;
      var lightness=68-trait*.28;

      return 'hsl('
        +clamp(hue,205,285)
        +',64%,'
        +clamp(lightness,34,68)
        +'%)';
    }

    function organism(
      x,
      y,
      trait,
      index,
      populationName
    ){
      var color=traitColor(trait);
      var wingColor=populationName==='left'
        ?'#F59E0B'
        :'#3B82F6';

      return ''
        +'<g class="si-organism"'
        +' transform="translate('+x+' '+y+')">'
        +'<ellipse cx="0" cy="0" rx="11" ry="7"'
        +' fill="'+color+'" stroke="#0F172A"'
        +' stroke-width="1.5"/>'
        +'<ellipse cx="-3" cy="-8" rx="8" ry="6"'
        +' fill="'+wingColor+'" opacity=".62"'
        +' stroke="#334155" stroke-width="1"/>'
        +'<ellipse cx="5" cy="-8" rx="8" ry="6"'
        +' fill="'+wingColor+'" opacity=".62"'
        +' stroke="#334155" stroke-width="1"/>'
        +'<circle cx="-11" cy="0" r="4.5"'
        +' fill="'+color+'" stroke="#0F172A"'
        +' stroke-width="1.5"/>'
        +'<line x1="-14" y1="-3" x2="-18" y2="-8"'
        +' stroke="#0F172A" stroke-width="1.1"/>'
        +'<line x1="-14" y1="3" x2="-18" y2="8"'
        +' stroke="#0F172A" stroke-width="1.1"/>'
        +'<text x="0" y="22" text-anchor="middle"'
        +' font-size="8" font-weight="900"'
        +' fill="#334155">'
        +Math.round(trait)
        +'</text>'
        +'</g>';
    }

    function populationHTML(
      side,
      mean,
      spread
    ){
      var html='';
      var baseX=side==='left'
        ?62
        :428;

      for(var i=0;i<18;i++){
        var row=Math.floor(i/6);
        var col=i%6;
        var offset=
          (((i*7)%17)-8)/8;
        var trait=clamp(
          mean+offset*spread,
          2,
          98
        );
        var x=baseX+col*38+(row%2)*8;
        var y=132+row*39;

        html+=organism(
          x,
          y,
          trait,
          i,
          side
        );
      }

      return html;
    }

    function renderBarrier(
      isolationStrength,
      stageName
    ){
      if(stageName==='connected'){
        return ''
          +'<rect x="318" y="82" width="44" height="138"'
          +' rx="18" fill="#D1FAE5" opacity=".55"/>'
          +'<text x="340" y="151" text-anchor="middle"'
          +' font-size="11" font-weight="900"'
          +' fill="#047857" transform="rotate(-90 340 151)">'
          +'连通通道'
          +'</text>';
      }

      var opacity=.25+.75*isolationStrength/100;

      return ''
        +'<path d="M340 78'
        +' L319 111 L350 138 L320 169'
        +' L352 198 L340 224"'
        +' fill="none" stroke="#64748B"'
        +' stroke-width="'+(8+isolationStrength/8)
        +'" stroke-linecap="round"'
        +' opacity="'+opacity+'"/>'
        +'<text x="340" y="237" text-anchor="middle"'
        +' font-size="10.5" font-weight="900"'
        +' fill="#475569">'
        +(stageName==='contact'
          ?'屏障已经减弱或消失'
          :'地理屏障')
        +'</text>';
    }

    function renderGeneFlow(
      geneFlow,
      stageName
    ){
      var html='';
      var opacity=.15+.85*geneFlow/100;
      var width=2+geneFlow/20;

      if(stageName==='contact'){
        return html;
      }

      html+='<path class="si-flow"'
        +' d="M276 125 C307 105 373 105 404 125"'
        +' fill="none" stroke="#0284C7"'
        +' stroke-width="'+width+'"'
        +' opacity="'+opacity+'"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>';

      html+='<path class="si-flow"'
        +' d="M404 194 C373 214 307 214 276 194"'
        +' fill="none" stroke="#0284C7"'
        +' stroke-width="'+width+'"'
        +' opacity="'+opacity+'"'
        +' marker-end="url(#${rootId}-arrow-blue)"/>';

      html+='<text x="340" y="105" text-anchor="middle"'
        +' font-size="10.5" font-weight="900"'
        +' fill="#0369A1">'
        +'相对基因交流 '
        +geneFlow.toFixed(0)
        +'%</text>';

      return html;
    }

    function renderContact(
      compatibility
    ){
      if(stage!=='contact'){
        return '';
      }

      var html='';
      var successful=
        Math.max(
          0,
          Math.floor(compatibility/20)
        );

      html+='<path class="si-flow"'
        +' d="M287 148 C314 132 366 132 393 148"'
        +' fill="none" stroke="'
        +(compatibility>=50?'#10B981':'#F59E0B')
        +'" stroke-width="5"'
        +' marker-end="url(#${rootId}-arrow-orange)"/>';

      html+='<text x="340" y="125" text-anchor="middle"'
        +' font-size="10.5" font-weight="900"'
        +' fill="'
        +(compatibility>=50?'#047857':'#B45309')
        +'">再次接触与交配检验</text>';

      for(var i=0;i<5;i++){
        var x=300+i*20;
        var y=193+(i%2)*8;
        var active=i<successful;

        html+='<g class="si-pulse">'
          +'<circle cx="'+x+'" cy="'+y+'" r="8"'
          +' fill="'
          +(active?'#34D399':'#CBD5E1')
          +'" stroke="'
          +(active?'#047857':'#64748B')
          +'" stroke-width="2"/>'
          +'<text x="'+x+'" y="'+(y+3)
          +'" text-anchor="middle" font-size="7.5"'
          +' font-weight="900" fill="#FFFFFF">'
          +(active?'F':'×')
          +'</text>'
          +'</g>';
      }

      return html;
    }

    function renderGraphGrid(
      generationCount
    ){
      var html='';
      var left=62;
      var right=636;
      var top=274;
      var bottom=356;
      var width=right-left;
      var height=bottom-top;

      for(var yIndex=0;yIndex<=4;yIndex++){
        var y=bottom-height*yIndex/4;
        var value=yIndex*25;

        html+='<line x1="'+left+'" y1="'+y
          +'" x2="'+right+'" y2="'+y
          +'" stroke="#E2E8F0" stroke-width="1.2"/>';

        html+='<text x="'+(left-8)+'" y="'+(y+4)
          +'" text-anchor="end" font-size="9.5"'
          +' font-weight="700" fill="#64748B">'
          +value+'</text>';
      }

      var ticks=Math.min(
        5,
        generationCount
      );

      for(var xIndex=0;xIndex<=ticks;xIndex++){
        var x=left+width*xIndex/ticks;
        var generation=Math.round(
          generationCount*xIndex/ticks
        );

        html+='<line x1="'+x+'" y1="'+bottom
          +'" x2="'+x+'" y2="'+top
          +'" stroke="#F1F5F9" stroke-width="1"/>';

        html+='<text x="'+x+'" y="'+(bottom+17)
          +'" text-anchor="middle" font-size="9.5"'
          +' font-weight="700" fill="#64748B">'
          +generation+'</text>';
      }

      return html;
    }

    function renderDivergenceCurve(
      records
    ){
      var left=62;
      var right=636;
      var top=274;
      var bottom=356;
      var width=right-left;
      var height=bottom-top;
      var path='';
      var points='';

      function x(index){
        return left
          +width*index/(records.length-1);
      }

      function y(value){
        return bottom
          -height*clamp(value/100,0,1);
      }

      for(var i=0;i<records.length;i++){
        var px=x(i);
        var py=y(records[i].divergence);

        path+=(i===0?'M':' L')
          +px+' '+py;

        if(
          i===0
          ||i===records.length-1
          ||i%Math.max(
            1,
            Math.floor(records.length/6)
          )===0
        ){
          points+='<circle cx="'+px+'" cy="'+py
            +'" r="3.8" fill="#FFFFFF"'
            +' stroke="#0284C7" stroke-width="2.4"/>';
        }
      }

      divergenceCurve.setAttribute(
        'd',
        path
      );

      divergenceArea.setAttribute(
        'd',
        path
        +' L'+right+' '+bottom
        +' L'+left+' '+bottom
        +' Z'
      );

      divergencePoints.innerHTML=points;
    }

    function update(){
      var isolationStrength=Number(
        isolationInput.value
      );
      var environmentDifference=Number(
        environmentInput.value
      );
      var driftStrength=Number(
        driftInput.value
      );
      var generationCount=Math.round(
        Number(generationsInput.value)
      );

      var info=stageInfo[stage];
      var finalDivergence=
        calculateFinalDivergence(
          isolationStrength,
          environmentDifference,
          driftStrength,
          generationCount
        );

      var effectiveDivergence=
        stage==='connected'
          ?finalDivergence*.12
          :finalDivergence;

      var geneFlow=
        stage==='connected'
          ?100
          :calculateGeneFlow(
            isolationStrength
          );

      var compatibility=
        calculateCompatibility(
          effectiveDivergence
        );

      var records=calculateRecords(
        stage==='connected'
          ?isolationStrength*.12
          :isolationStrength,
        environmentDifference,
        driftStrength,
        generationCount
      );

      isolationValue.textContent=
        isolationStrength.toFixed(0)+'%';
      environmentValue.textContent=
        environmentDifference.toFixed(0)+'%';
      driftValue.textContent=
        driftStrength.toFixed(0)+'%';
      generationsValue.textContent=
        generationCount.toFixed(0)+' 代';

      divergenceValue.textContent=
        effectiveDivergence.toFixed(0);
      compatibilityValue.textContent=
        compatibility.toFixed(0)+'%';

      root.style.setProperty(
        '--si-speed',
        clamp(
          2.5-environmentDifference/65,
          .55,
          2.4
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--si-flow-speed',
        clamp(
          2.5-geneFlow/60,
          .5,
          2.4
        ).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      title.textContent=info.title;
      summary.textContent=info.summary;
      stageNote.textContent=info.note;

      var driftBias=
        (driftStrength-50)*.035;
      var meanShift=
        effectiveDivergence*.34;

      var leftMean=stage==='connected'
        ?50
        :clamp(
          50-meanShift-driftBias,
          5,
          95
        );

      var rightMean=stage==='connected'
        ?50
        :clamp(
          50+meanShift+driftBias,
          5,
          95
        );

      var spread=
        clamp(
          14-environmentDifference*.06,
          6,
          14
        );

      leftLabel.textContent=
        '种群甲 · 平均性状 '
        +leftMean.toFixed(0);

      rightLabel.textContent=
        '种群乙 · 平均性状 '
        +rightMean.toFixed(0);

      leftPopulation.innerHTML=
        populationHTML(
          'left',
          leftMean,
          spread
        );

      rightPopulation.innerHTML=
        populationHTML(
          'right',
          rightMean,
          spread
        );

      barrierLayer.innerHTML=
        renderBarrier(
          isolationStrength,
          stage
        );

      geneFlowLayer.innerHTML=
        renderGeneFlow(
          geneFlow,
          stage
        );

      contactLayer.innerHTML=
        renderContact(
          compatibility
        );

      graphGrid.innerHTML=
        renderGraphGrid(
          generationCount
        );

      renderDivergenceCurve(records);

      var referenceY=
        356-(356-274)*.6;

      referenceLine.setAttribute(
        'y1',
        String(referenceY)
      );
      referenceLine.setAttribute(
        'y2',
        String(referenceY)
      );

      referenceLabel.setAttribute(
        'y',
        String(referenceY-7)
      );

      var state='';
      var explanation='';

      if(stage==='connected'){
        state='同一繁殖种群';

        explanation=
          '两个区域之间保持较充分的基因交流，局部差异不易长期独立积累。';
      }else if(effectiveDivergence<25){
        state='差异较小';

        explanation=
          '虽然存在一定地理隔离，但隔离时间、环境差异或随机变化尚不足以形成明显分化。';
      }else if(effectiveDivergence<55){
        state='分化积累';

        explanation=
          '两个种群已经积累一定遗传差异，但再次接触后仍可能保持较高繁殖兼容度。';
      }else if(effectiveDivergence<75){
        state='部分隔离';

        explanation=
          '种群差异较明显，再次接触时可能出现交配机会减少或杂交后代繁殖能力下降。';
      }else{
        state='强生殖隔离';

        explanation=
          '两个种群之间的繁殖兼容度已经很低，模型显示出较强生殖隔离。';
      }

      var forceNote='';

      if(isolationStrength<20){
        forceNote=
          '地理隔离较弱，持续基因交流会抵消部分分化趋势。';
      }else if(generationCount<10){
        forceNote=
          '隔离世代较少，物种形成通常不会在短时间内自动完成。';
      }else if(environmentDifference<15){
        forceNote=
          '两地环境差异较小，差异积累主要来自随机变化等其他因素。';
      }else if(driftStrength>75){
        forceNote=
          '遗传漂变影响较强，但漂变方向本身并不是由环境需要决定的。';
      }else{
        forceNote=
          '基因交流减少、不同环境选择和随机遗传变化共同影响两个种群的分化。';
      }

      result.innerHTML=
        explanation
        +'<br>'+forceNote
        +' 当前阶段判断：'
        +state
        +'。地理隔离本身不等于新物种形成；形成稳定的生殖隔离才意味着物种形成过程发生了关键变化。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        stage=this.getAttribute('data-stage');
        update();
      };
    }

    isolationInput.oninput=update;
    environmentInput.oninput=update;
    driftInput.oninput=update;
    generationsInput.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
