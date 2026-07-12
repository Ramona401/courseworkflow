/**
 * lifeScienceLabTemplatesEvolutionNaturalSelection.ts
 *
 * 平面生命科学实验室：自然选择与种群性状变化。
 *
 * 教学目标：
 * 1. 理解种群中首先存在可遗传的个体差异；
 * 2. 理解环境条件会使不同性状个体具有不同的生存和繁殖机会；
 * 3. 理解自然选择作用于个体表现型，但进化结果表现为种群性状频率变化；
 * 4. 比较浅色环境、深色环境和稳定环境中的选择结果；
 * 5. 观察初始变异、选择强度、遗传度和世代数对种群平均性状的影响；
 * 6. 区分定向选择与稳定选择；
 * 7. 理解环境改变不会按需要定向产生有利变异。
 *
 * 教学边界：
 * 1. 本模型用0—100的连续数值代表一种可遗传性状，不对应具体物种；
 * 2. 种群初始平均性状设为50，个体间已经存在一定变异；
 * 3. 浅色环境偏向较低性状值，深色环境偏向较高性状值；
 * 4. 稳定环境偏向中间性状值，主要用于演示稳定选择；
 * 5. 遗传度较低时，亲代中的选择差异不一定充分传递到后代；
 * 6. 模型忽略突变、迁移、遗传漂变、性选择和复杂基因互作；
 * 7. 数值和曲线均为教学示意，不用于真实种群预测。
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
function naturalSelectionStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #A7F3D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#D1FAE5,#FEF3C7);border-bottom:1px solid #A7F3D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:244px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#F8FFFC;border-right:1px solid #A7F3D0}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#059669;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .bl-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-button{height:34px;padding:0 3px;border:1px solid #6EE7B7;border-radius:8px;background:#fff;color:#065F46;font-size:10px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#10B981;background:#D1FAE5;box-shadow:0 3px 9px rgba(16,185,129,.14)}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #A7F3D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:15px;color:#047857;min-height:20px}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#D1FAE5;color:#064E3B;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ns-organism{animation:' + rootId + '-organism var(--ns-speed,1.7s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .ns-selection{stroke-dasharray:8 7;animation:' + rootId + '-selection 1.5s linear infinite}'
    + '@keyframes ' + rootId + '-organism{from{transform:translateY(3px);opacity:.58}to{transform:translateY(-4px);opacity:1}}'
    + '@keyframes ' + rootId + '-selection{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_EVOLUTION_NATURAL_SELECTION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-natural-selection',
    group: '🦋 进化与生物多样性',
    name: '自然选择与种群性状变化',
    emoji: '🦋',
    desc: '调节初始变异、选择强度、遗传度和世代数，比较定向选择与稳定选择',
    params: [
      {
        key: 'variationLevel',
        label: '初始性状变异',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 65,
        hint: '表示种群中原本存在的可遗传性状差异',
      },
      {
        key: 'selectionStrength',
        label: '自然选择强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'heritability',
        label: '性状遗传度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 82,
        hint: '表示亲代性状差异传递给后代的相对程度',
      },
      {
        key: 'generations',
        label: '演化世代数',
        type: 'number',
        min: 1,
        max: 30,
        step: 1,
        defaultValue: 18,
      },
    ],

    buildHTML: (params, rootId) => {
      const variationLevel = num(
        params,
        'variationLevel',
        65,
      )
      const selectionStrength = num(
        params,
        'selectionStrength',
        68,
      )
      const heritability = num(
        params,
        'heritability',
        82,
      )
      const generations = num(
        params,
        'generations',
        18,
      )

      return `
<div id="${rootId}">
${naturalSelectionStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🦋 自然选择与种群性状变化</div>
    <div class="bl-note">选择作用于已有变异，种群在多代中发生变化</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>初始性状变异</span>
          <span class="bl-value" data-variation-value></span>
        </div>
        <input
          data-variation
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(variationLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>自然选择强度</span>
          <span class="bl-value" data-selection-value></span>
        </div>
        <input
          data-selection
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(selectionStrength)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>性状遗传度</span>
          <span class="bl-value" data-heritability-value></span>
        </div>
        <input
          data-heritability
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(heritability)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>演化世代数</span>
          <span class="bl-value" data-generations-value></span>
        </div>
        <input
          data-generations
          type="range"
          min="1"
          max="30"
          step="1"
          value="${n(generations)}"
        >
      </div>

      <div class="bl-subtitle">选择情境</div>

      <div class="bl-buttons">
        <button
          type="button"
          class="bl-button active"
          data-mode="light"
        >浅色环境</button>

        <button
          type="button"
          class="bl-button"
          data-mode="dark"
        >深色环境</button>

        <button
          type="button"
          class="bl-button"
          data-mode="stable"
        >稳定环境</button>
      </div>

      <div class="bl-status">
        <div class="bl-card">
          <b data-final-mean></b>
          <span>末代平均性状</span>
        </div>

        <div class="bl-card">
          <b data-survival></b>
          <span>末代相对适合度</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="自然选择与种群性状变化互动实验"
      >
        <defs>
          <linearGradient
            id="${rootId}-light-habitat"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#FEF3C7"/>
            <stop offset="100%" stop-color="#FDE68A"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-dark-habitat"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#475569"/>
            <stop offset="100%" stop-color="#0F172A"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-stable-habitat"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#D1FAE5"/>
            <stop offset="100%" stop-color="#A7F3D0"/>
          </linearGradient>

          <marker
            id="${rootId}-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#059669"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#065F46"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="22"
          y="34"
          data-title
          font-size="24"
          font-weight="900"
          fill="#065F46"
        ></text>

        <text
          x="22"
          y="62"
          data-summary
          font-size="13"
          font-weight="800"
          fill="#475569"
        ></text>

        <!-- 左上：环境与个体差异 -->
        <g filter="url(#${rootId}-shadow)">
          <rect
            data-habitat
            x="22"
            y="82"
            width="286"
            height="150"
            rx="18"
            fill="url(#${rootId}-light-habitat)"
            stroke="#10B981"
            stroke-width="3"
          />
        </g>

        <text
          x="36"
          y="105"
          data-habitat-label
          font-size="13"
          font-weight="900"
          fill="#065F46"
        ></text>

        <g data-organisms></g>
        <g data-selection-arrows></g>

        <!-- 左下：初始与末代分布 -->
        <text
          x="22"
          y="260"
          font-size="13"
          font-weight="900"
          fill="#334155"
        >性状频率分布</text>

        <line
          x1="42"
          y1="354"
          x2="300"
          y2="354"
          stroke="#64748B"
          stroke-width="2.5"
        />

        <line
          x1="42"
          y1="354"
          x2="42"
          y2="278"
          stroke="#64748B"
          stroke-width="2.5"
        />

        <g data-histogram></g>

        <text
          x="42"
          y="373"
          font-size="10"
          font-weight="800"
          fill="#64748B"
        >浅色性状</text>

        <text
          x="264"
          y="373"
          font-size="10"
          font-weight="800"
          fill="#64748B"
        >深色性状</text>

        <!-- 右侧：平均性状随世代变化 -->
        <text
          x="350"
          y="92"
          font-size="13"
          font-weight="900"
          fill="#334155"
        >种群平均性状随世代变化</text>

        <line
          x1="370"
          y1="340"
          x2="640"
          y2="340"
          stroke="#64748B"
          stroke-width="2.5"
        />

        <line
          x1="370"
          y1="340"
          x2="370"
          y2="112"
          stroke="#64748B"
          stroke-width="2.5"
        />

        <g data-graph-grid></g>

        <line
          data-optimum-line
          x1="370"
          y1="220"
          x2="640"
          y2="220"
          stroke="#F59E0B"
          stroke-width="3"
          stroke-dasharray="8 7"
        />

        <text
          data-optimum-label
          x="550"
          y="210"
          font-size="11"
          font-weight="900"
          fill="#B45309"
        ></text>

        <path
          data-mean-area
          fill="#A7F3D0"
          opacity=".35"
        ></path>

        <path
          data-mean-curve
          fill="none"
          stroke="#059669"
          stroke-width="5"
          stroke-linecap="round"
          stroke-linejoin="round"
        ></path>

        <g data-mean-points></g>

        <text
          x="610"
          y="362"
          font-size="11"
          font-weight="800"
          fill="#64748B"
        >世代</text>

        <text
          x="326"
          y="122"
          font-size="11"
          font-weight="800"
          fill="#64748B"
        >性状值</text>

        <g transform="translate(354 384)">
          <rect x="0" y="-9" width="18" height="10" rx="3" fill="#94A3B8"/>
          <text x="26" y="0" font-size="11" font-weight="800" fill="#475569">
            初始分布
          </text>
        </g>

        <g transform="translate(468 384)">
          <rect x="0" y="-9" width="18" height="10" rx="3" fill="#10B981"/>
          <text x="26" y="0" font-size="11" font-weight="800" fill="#475569">
            末代分布
          </text>
        </g>

        <text
          x="570"
          y="385"
          data-stage-note
          font-size="11"
          font-weight="900"
          fill="#047857"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var variationInput=root.querySelector(
      '[data-variation]'
    );
    var selectionInput=root.querySelector(
      '[data-selection]'
    );
    var heritabilityInput=root.querySelector(
      '[data-heritability]'
    );
    var generationsInput=root.querySelector(
      '[data-generations]'
    );

    var variationValue=root.querySelector(
      '[data-variation-value]'
    );
    var selectionValue=root.querySelector(
      '[data-selection-value]'
    );
    var heritabilityValue=root.querySelector(
      '[data-heritability-value]'
    );
    var generationsValue=root.querySelector(
      '[data-generations-value]'
    );

    var buttons=root.querySelectorAll('[data-mode]');
    var finalMeanValue=root.querySelector(
      '[data-final-mean]'
    );
    var survivalValue=root.querySelector(
      '[data-survival]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var habitat=root.querySelector('[data-habitat]');
    var habitatLabel=root.querySelector(
      '[data-habitat-label]'
    );
    var organisms=root.querySelector(
      '[data-organisms]'
    );
    var selectionArrows=root.querySelector(
      '[data-selection-arrows]'
    );
    var histogram=root.querySelector(
      '[data-histogram]'
    );

    var graphGrid=root.querySelector(
      '[data-graph-grid]'
    );
    var optimumLine=root.querySelector(
      '[data-optimum-line]'
    );
    var optimumLabel=root.querySelector(
      '[data-optimum-label]'
    );
    var meanArea=root.querySelector(
      '[data-mean-area]'
    );
    var meanCurve=root.querySelector(
      '[data-mean-curve]'
    );
    var meanPoints=root.querySelector(
      '[data-mean-points]'
    );
    var stageNote=root.querySelector(
      '[data-stage-note]'
    );

    var mode='light';

    var modeInfo={
      light:{
        optimum:25,
        title:'浅色环境中的定向选择',
        summary:'种群原本存在深浅差异，较浅个体具有更高的相对适合度',
        habitat:'浅色环境',
        gradient:'url(#${rootId}-light-habitat)',
        note:'平均性状向较浅方向移动'
      },
      dark:{
        optimum:75,
        title:'深色环境中的定向选择',
        summary:'种群原本存在深浅差异，较深个体具有更高的相对适合度',
        habitat:'深色环境',
        gradient:'url(#${rootId}-dark-habitat)',
        note:'平均性状向较深方向移动'
      },
      stable:{
        optimum:50,
        title:'稳定环境中的稳定选择',
        summary:'中间性状个体相对适合度较高，两端性状受到较强选择',
        habitat:'稳定环境',
        gradient:'url(#${rootId}-stable-habitat)',
        note:'平均值接近不变，变异范围缩小'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function gaussian(
      value,
      mean,
      standardDeviation
    ){
      var distance=
        (value-mean)/Math.max(1,standardDeviation);

      return Math.exp(-.5*distance*distance);
    }

    /**
     * 计算某一性状在当前环境中的相对适合度。
     *
     * 选择强度越大，偏离环境适宜值的性状
     * 相对生存和繁殖机会下降越明显。
     */
    function relativeFitness(
      trait,
      optimum,
      selectionStrength
    ){
      var strength=selectionStrength/100;
      var width=44-31*strength;

      return gaussian(
        trait,
        optimum,
        width
      );
    }

    /**
     * 估算一个连续性状分布的平均相对适合度。
     */
    function averageFitness(
      mean,
      standardDeviation,
      optimum,
      selectionStrength
    ){
      var weightedFitness=0;
      var totalWeight=0;

      for(var trait=0;trait<=100;trait+=2){
        var frequency=gaussian(
          trait,
          mean,
          standardDeviation
        );
        var fitness=relativeFitness(
          trait,
          optimum,
          selectionStrength
        );

        weightedFitness+=frequency*fitness;
        totalWeight+=frequency;
      }

      return totalWeight>0
        ?weightedFitness/totalWeight
        :0;
    }

    /**
     * 模拟多代种群平均性状和变异范围。
     *
     * 这是确定性的教学模型：
     * - 选择强度决定不同性状的繁殖差异；
     * - 初始变异决定可供选择作用的性状差异；
     * - 遗传度决定亲代差异传递给后代的程度；
     * - 选择不会使单个个体按环境需要主动改变性状。
     */
    function calculateEvolution(
      selectionMode,
      variation,
      selectionStrength,
      heritability,
      generationCount
    ){
      var info=modeInfo[selectionMode];
      var optimum=info.optimum;
      var strength=selectionStrength/100;
      var inherited=heritability/100;
      var availableVariation=variation/100;

      var mean=50;
      var standardDeviation=
        7+variation*.18;

      var records=[{
        generation:0,
        mean:mean,
        standardDeviation:standardDeviation,
        fitness:averageFitness(
          mean,
          standardDeviation,
          optimum,
          selectionStrength
        )
      }];

      for(var generation=1;
        generation<=generationCount;
        generation++
      ){
        if(selectionMode==='stable'){
          var returnResponse=
            (optimum-mean)
            *strength
            *inherited
            *.12;

          mean+=returnResponse;

          standardDeviation*=
            1-strength*inherited*.025;
        }else{
          var directionalResponse=
            (optimum-mean)
            *strength
            *inherited
            *(.035+.085*availableVariation);

          mean+=directionalResponse;

          standardDeviation*=
            1-strength*inherited*.007;
        }

        mean=clamp(mean,2,98);
        standardDeviation=clamp(
          standardDeviation,
          4.5,
          28
        );

        records.push({
          generation:generation,
          mean:mean,
          standardDeviation:standardDeviation,
          fitness:averageFitness(
            mean,
            standardDeviation,
            optimum,
            selectionStrength
          )
        });
      }

      return records;
    }

    function traitColor(trait){
      var lightness=84-trait*.58;

      return 'hsl(215,32%,'
        +clamp(lightness,22,82)
        +'%)';
    }

    function organismSymbol(
      x,
      y,
      trait,
      fitness,
      index
    ){
      var color=traitColor(trait);
      var opacity=.24+.76*fitness;
      var scale=.82+.22*fitness;
      var halo=fitness>.72
        ?'<circle cx="'+x+'" cy="'+y+'" r="16"'
          +' fill="none" stroke="#10B981"'
          +' stroke-width="2.5" opacity=".72"/>'
        :'';

      return ''
        +halo
        +'<g class="ns-organism"'
        +' transform="translate('+x+' '+y+')'
        +' scale('+scale+')">'
        +'<ellipse cx="0" cy="0" rx="10" ry="7"'
        +' fill="'+color+'" stroke="#0F172A"'
        +' stroke-width="1.5" opacity="'+opacity+'"/>'
        +'<circle cx="-10" cy="0" r="4.5"'
        +' fill="'+color+'" stroke="#0F172A"'
        +' stroke-width="1.5" opacity="'+opacity+'"/>'
        +'<line x1="-13" y1="-3" x2="-18" y2="-8"'
        +' stroke="#0F172A" stroke-width="1.2"/>'
        +'<line x1="-13" y1="3" x2="-18" y2="8"'
        +' stroke="#0F172A" stroke-width="1.2"/>'
        +'<line x1="-3" y1="6" x2="-7" y2="12"'
        +' stroke="#0F172A" stroke-width="1.2"/>'
        +'<line x1="4" y1="6" x2="8" y2="12"'
        +' stroke="#0F172A" stroke-width="1.2"/>'
        +'<text x="0" y="24" text-anchor="middle"'
        +' font-size="8.5" font-weight="900"'
        +' fill="#334155">'
        +Math.round(trait)
        +'</text>'
        +'</g>';
    }

    function renderOrganisms(
      variation,
      selectionStrength,
      info
    ){
      var html='';
      var baseSpread=
        7+variation*.18;

      for(var i=0;i<18;i++){
        var row=Math.floor(i/6);
        var col=i%6;
        var standardized=
          ((i*7)%19-9)/4.2;
        var trait=clamp(
          50+standardized*baseSpread,
          2,
          98
        );
        var fitness=relativeFitness(
          trait,
          info.optimum,
          selectionStrength
        );
        var x=55+col*43+(row%2)*9;
        var y=130+row*43;

        html+=organismSymbol(
          x,
          y,
          trait,
          fitness,
          i
        );
      }

      return html;
    }

    function renderSelectionArrows(info){
      var targetX=info.optimum<50
        ?80
        :info.optimum>50
          ?260
          :165;

      return ''
        +'<path class="ns-selection"'
        +' d="M165 112 C150 105 '
        +targetX+' 104 '+targetX+' 122"'
        +' fill="none" stroke="#059669"'
        +' stroke-width="3"'
        +' marker-end="url(#${rootId}-arrow)"/>'
        +'<text x="165" y="113" text-anchor="middle"'
        +' font-size="10.5" font-weight="900"'
        +' fill="#047857">'
        +'较高相对适合度'
        +'</text>';
    }

    function distributionValues(
      mean,
      standardDeviation
    ){
      var values=[];

      for(var trait=0;trait<=100;trait+=5){
        values.push({
          trait:trait,
          frequency:gaussian(
            trait,
            mean,
            standardDeviation
          )
        });
      }

      return values;
    }

    function renderHistogram(
      initialRecord,
      finalRecord
    ){
      var initial=distributionValues(
        initialRecord.mean,
        initialRecord.standardDeviation
      );
      var finalValues=distributionValues(
        finalRecord.mean,
        finalRecord.standardDeviation
      );

      var html='';
      var left=44;
      var bottom=352;
      var width=250;
      var height=68;
      var barWidth=width/initial.length;

      for(var i=0;i<initial.length;i++){
        var initialHeight=
          height*initial[i].frequency;
        var finalHeight=
          height*finalValues[i].frequency;
        var x=left+i*barWidth;

        html+='<rect x="'+x+'" y="'
          +(bottom-initialHeight)
          +'" width="'+Math.max(2,barWidth-1.5)
          +'" height="'+initialHeight
          +'" fill="#94A3B8" opacity=".38"/>';

        html+='<rect x="'+(x+barWidth*.18)
          +'" y="'+(bottom-finalHeight)
          +'" width="'+Math.max(2,barWidth*.64)
          +'" height="'+finalHeight
          +'" fill="#10B981" opacity=".86"/>';
      }

      html+='<line x1="'
        +(left+width*initialRecord.mean/100)
        +'" y1="'+(bottom-height-4)
        +'" x2="'
        +(left+width*initialRecord.mean/100)
        +'" y2="'+bottom
        +'" stroke="#64748B" stroke-width="2"'
        +' stroke-dasharray="4 4"/>';

      html+='<line x1="'
        +(left+width*finalRecord.mean/100)
        +'" y1="'+(bottom-height-4)
        +'" x2="'
        +(left+width*finalRecord.mean/100)
        +'" y2="'+bottom
        +'" stroke="#059669" stroke-width="3"/>';

      return html;
    }

    function renderGraphGrid(
      generationCount
    ){
      var html='';
      var left=370;
      var right=640;
      var top=112;
      var bottom=340;
      var height=bottom-top;
      var width=right-left;

      for(var yIndex=0;yIndex<=4;yIndex++){
        var y=bottom-height*yIndex/4;
        var label=yIndex*25;

        html+='<line x1="'+left+'" y1="'+y
          +'" x2="'+right+'" y2="'+y
          +'" stroke="#E2E8F0" stroke-width="1.2"/>';

        html+='<text x="'+(left-8)+'" y="'+(y+4)
          +'" text-anchor="end" font-size="9.5"'
          +' font-weight="700" fill="#64748B">'
          +label+'</text>';
      }

      var tickCount=Math.min(
        6,
        generationCount
      );

      for(var xIndex=0;xIndex<=tickCount;xIndex++){
        var x=left+width*xIndex/tickCount;
        var generation=Math.round(
          generationCount*xIndex/tickCount
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

    function renderMeanCurve(
      records,
      optimum
    ){
      var left=370;
      var right=640;
      var top=112;
      var bottom=340;
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
        var py=y(records[i].mean);

        path+=(i===0?'M':' L')
          +px+' '+py;

             for(var i=0;i<records.length;i++){
        var px=x(i);
        var py=y(records[i].mean);

        path+=(i=== if(
          i===0
          ||i===records.length-1
          ||i%Math.max(
            1,
            Math.floor(records.length/6)
          )===0
        ){
          points+='<circle cx="'+px+'" cy="'+py
            +'" r="4" fill="#FFFFFF"'
            +' stroke="#059669" stroke-width="2.5"/>';
        }
      }

      meanCurve.setAttribute('d',path);
      meanPoints.innerHTML=points;

      meanArea.setAttribute(
        'd',
        path
        +' L'+right+' '+bottom
        +' L'+left+' '+bottom
        +' Z'
      );

      var optimumY=y(optimum);

      optimumLine.setAttribute(
        'y1',
        String(optimumY)
      );
      optimumLine.setAttribute(
        'y2',
        String(optimumY)
      );

      optimumLabel.setAttribute(
        'y',
        String(Math.max(top+12,optimumY-8))
      );
      optimumLabel.textContent=
        '环境适宜值 '+optimum;
    }

    function update(){
      var variation=Number(
        variationInput.value
      );
      var selectionStrength=Number(
        selectionInput.value
      );
      var inherited=Number(
        heritabilityInput.value
      );
      var generationCount=Math.round(
        Number(generationsInput.value)
      );
      var info=modeInfo[mode];

      var records=calculateEvolution(
        mode,
        variation,
        selectionStrength,
        inherited,
        generationCount
      );

      var initialRecord=records[0];
      var finalRecord=records[
        records.length-1
      ];

      variationValue.textContent=
        variation.toFixed(0)+'%';
      selectionValue.textContent=
        selectionStrength.toFixed(0)+'%';
      heritabilityValue.textContent=
        inherited.toFixed(0)+'%';
      generationsValue.textContent=
        generationCount.toFixed(0)+' 代';

      finalMeanValue.textContent=
        finalRecord.mean.toFixed(1);
      survivalValue.textContent=
        (finalRecord.fitness*100).toFixed(0)+'%';

      root.style.setProperty(
        '--ns-speed',
        clamp(
          2.5-selectionStrength/60,
          .55,
          2.4
        ).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-mode')===mode
        );
      }

      title.textContent=info.title;
      summary.textContent=info.summary;
      habitatLabel.textContent=
        info.habitat
        +' · 适宜性状值约 '
        +info.optimum;

      habitat.setAttribute(
        'fill',
        info.gradient
      );

      habitat.setAttribute(
        'stroke',
        mode==='dark'
          ?'#334155'
          :'#10B981'
      );

      organisms.innerHTML=renderOrganisms(
        variation,
        selectionStrength,
        info
      );

      selectionArrows.innerHTML=
        renderSelectionArrows(info);

      histogram.innerHTML=renderHistogram(
        initialRecord,
        finalRecord
      );

      graphGrid.innerHTML=renderGraphGrid(
        generationCount
      );

      renderMeanCurve(
        records,
        info.optimum
      );

      stageNote.textContent=info.note;

      var change=
        finalRecord.mean-initialRecord.mean;
      var changeText=Math.abs(change)<.5
        ?'种群平均性状变化较小'
        :change<0
          ?'种群平均性状向较低数值移动'
          :'种群平均性状向较高数值移动';

      var condition='';

      if(selectionStrength<10){
        condition=
          '选择强度很低，不同性状个体的相对繁殖差异较小。';
      }else if(variation<20){
        condition=
          '初始可遗传变异较少，可供自然选择作用的性状差异有限。';
      }else if(inherited<15){
        condition=
          '性状遗传度较低，亲代中的选择差异难以充分传递到后代。';
      }else if(generationCount<5){
        condition=
          '演化世代较少，种群层面的累积变化尚不明显。';
      }else if(mode==='stable'){
        condition=
          '中间性状个体具有较高相对适合度，末代分布比初始分布更集中。';
      }else{
        condition=
          '具有较高相对适合度的个体留下更多后代，多代累积后性状频率发生变化。';
      }

      result.innerHTML=
        '自然选择不会使个体为了适应环境而主动改变性状，也不会按需要定向产生有利变异。'
        +'<br>'+condition
        +' '+changeText
        +'，由 '
        +initialRecord.mean.toFixed(1)
        +' 变为 '
        +finalRecord.mean.toFixed(1)
        +'。进化表现为种群在多个世代中的遗传组成和性状频率变化。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    variationInput.oninput=update;
    selectionInput.oninput=update;
    heritabilityInput.oninput=update;
    generationsInput.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
