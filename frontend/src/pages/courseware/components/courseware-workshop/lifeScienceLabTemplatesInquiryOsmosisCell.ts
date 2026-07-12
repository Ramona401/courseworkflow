/**
 * lifeScienceLabTemplatesInquiryOsmosisCell.ts
 *
 * 平面生命科学实验室：渗透作用与细胞吸水失水。
 *
 * 教学目标：
 * 1. 理解水分子通过选择透过性膜发生净移动的基本方向；
 * 2. 比较细胞内外溶质相对浓度，判断低渗、等渗和高渗环境；
 * 3. 观察植物细胞吸水膨胀、质壁分离以及动物细胞体积变化；
 * 4. 理解细胞壁对植物细胞吸水膨胀具有机械限制作用；
 * 5. 训练控制变量、比较实验和结果解释能力。
 *
 * 教学边界：
 * 1. 本模型假设溶质不能自由通过细胞膜，而水可以通过；
 * 2. 本模型用“相对溶质浓度”描述细胞内外环境，不等同于真实摩尔浓度；
 * 3. 水的净移动方向由细胞内外相对水势差简化表示；
 * 4. 真实细胞还受到溶质种类、膜蛋白、温度、细胞状态等多种因素影响；
 * 5. 植物细胞吸水后受到细胞壁限制，动物细胞在极端低渗环境中可能破裂；
 * 6. 本模型不模拟主动运输、离子泵和可自由跨膜溶质。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式或图片；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用.bl-*公共类名，兼容生命科学实验室底部课堂控制条布局；
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
 * 构建限定到当前rootId的样式。
 *
 * 独立预览时保持左侧参数区；
 * 嵌入课件后由公共.bl-*覆盖层切换为底部课堂控制条。
 */
function osmosisCellStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#ECFDF5);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:242px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#F8FDFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-button{height:32px;padding:0 5px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#0EA5E9;background:#E0F2FE;box-shadow:0 3px 9px rgba(14,165,233,.14)}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:16px;color:#0369A1}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .oc-water{animation:' + rootId + '-water var(--oc-speed,1.5s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .oc-solute{animation:' + rootId + '-solute 2.2s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-water{from{opacity:.38}to{opacity:1}}'
    + '@keyframes ' + rootId + '-solute{from{opacity:.62}to{opacity:.95}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_OSMOSIS_CELL:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-osmosis-cell',
    group: '🧪 实验探究',
    name: '渗透作用与细胞吸水失水',
    emoji: '💧',
    desc: '调节细胞内外溶质浓度、膜透水性和作用时间，比较植物细胞与动物细胞的吸水失水',
    params: [
      {
        key: 'externalSolute',
        label: '细胞外溶质浓度',
        type: 'number',
        min: 0,
        max: 20,
        step: 0.5,
        defaultValue: 6,
        hint: '相对教学单位，不代表真实摩尔浓度',
      },
      {
        key: 'internalSolute',
        label: '细胞内溶质浓度',
        type: 'number',
        min: 0,
        max: 20,
        step: 0.5,
        defaultValue: 10,
        hint: '相对教学单位，不代表真实摩尔浓度',
      },
      {
        key: 'membranePermeability',
        label: '细胞膜透水性',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'elapsedTime',
        label: '作用时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 65,
      },
    ],

    buildHTML: (params, rootId) => {
      const externalSolute = num(
        params,
        'externalSolute',
        6,
      )
      const internalSolute = num(
        params,
        'internalSolute',
        10,
      )
      const membranePermeability = num(
        params,
        'membranePermeability',
        78,
      )
      const elapsedTime = num(
        params,
        'elapsedTime',
        65,
      )

      return `
<div id="${rootId}">
${osmosisCellStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">💧 渗透作用与细胞吸水失水</div>
    <div class="bl-note">相对浓度教学模型：假设溶质不能自由通过细胞膜</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>细胞外溶质浓度</span>
          <span class="bl-value" data-external-value></span>
        </div>
        <input
          data-external
          type="range"
          min="0"
          max="20"
          step="0.5"
          value="${n(externalSolute)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>细胞内溶质浓度</span>
          <span class="bl-value" data-internal-value></span>
        </div>
        <input
          data-internal
          type="range"
          min="0"
          max="20"
          step="0.5"
          value="${n(internalSolute)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>细胞膜透水性</span>
          <span class="bl-value" data-permeability-value></span>
        </div>
        <input
          data-permeability
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(membranePermeability)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>作用时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input
          data-time
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(elapsedTime)}"
        >
      </div>

      <div class="bl-subtitle">观察对象</div>

      <div class="bl-buttons">
        <button
          type="button"
          class="bl-button active"
          data-mode="plant"
        >植物细胞</button>

        <button
          type="button"
          class="bl-button"
          data-mode="animal"
        >动物细胞</button>
      </div>

      <div class="bl-status">
        <div class="bl-card">
          <b data-solution-state></b>
          <span>外界溶液状态</span>
        </div>

        <div class="bl-card">
          <b data-cell-state></b>
          <span>细胞状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="渗透作用与细胞吸水失水互动实验"
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
            id="${rootId}-arrow-orange"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <linearGradient
            id="${rootId}-solution"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#BAE6FD"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-vacuole"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop offset="0%" stop-color="#DBEAFE"/>
            <stop offset="100%" stop-color="#93C5FD"/>
          </linearGradient>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#075985"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="26"
          y="37"
          data-title
          font-size="25"
          font-weight="900"
          fill="#075985"
        ></text>

        <text
          x="26"
          y="66"
          data-summary
          font-size="14"
          font-weight="800"
          fill="#475569"
        ></text>

        <!-- 烧杯与外界溶液 -->
        <g filter="url(#${rootId}-shadow)">
          <path
            d="M91 100 H455 L438 334
               C436 356 420 369 398 369
               H148
               C126 369 110 356 108 334 Z"
            fill="#FFFFFF"
            stroke="#0284C7"
            stroke-width="5"
          />

          <path
            d="M108 160 H438 L427 329
               C425 345 414 352 397 352
               H149
               C132 352 121 345 119 329 Z"
            fill="url(#${rootId}-solution)"
            opacity=".82"
          />

          <line
            x1="108"
            y1="160"
            x2="438"
            y2="160"
            stroke="#38BDF8"
            stroke-width="4"
          />
        </g>

        <text
          x="114"
          y="132"
          data-external-label
          font-size="13"
          font-weight="900"
          fill="#0369A1"
        ></text>

        <!-- 植物细胞图层 -->
        <g data-plant-layer>
          <rect
            x="182"
            y="188"
            width="188"
            height="128"
            rx="20"
            fill="#DCFCE7"
            stroke="#166534"
            stroke-width="9"
          />

          <rect
            data-plant-membrane
            x="194"
            y="200"
            width="164"
            height="104"
            rx="17"
            fill="#F0FDF4"
            stroke="#22C55E"
            stroke-width="5"
          />

          <rect
            data-vacuole
            x="213"
            y="218"
            width="126"
            height="68"
            rx="20"
            fill="url(#${rootId}-vacuole)"
            stroke="#2563EB"
            stroke-width="4"
          />

          <circle
            data-plant-nucleus
            cx="223"
            cy="241"
            r="14"
            fill="#C4B5FD"
            stroke="#7C3AED"
            stroke-width="4"
          />

          <text
            x="276"
            y="339"
            text-anchor="middle"
            font-size="14"
            font-weight="900"
            fill="#166534"
          >细胞壁限制过度膨胀</text>
        </g>

        <!-- 动物细胞图层 -->
        <g data-animal-layer style="display:none">
          <ellipse
            data-animal-cell
            cx="276"
            cy="252"
            rx="88"
            ry="68"
            fill="#FCE7F3"
            stroke="#DB2777"
            stroke-width="6"
          />

          <ellipse
            data-animal-inner
            cx="276"
            cy="252"
            rx="73"
            ry="53"
            fill="#FDF2F8"
            stroke="#F9A8D4"
            stroke-width="3"
          />

          <circle
            cx="246"
            cy="244"
            r="19"
            fill="#DDD6FE"
            stroke="#7C3AED"
            stroke-width="4"
          />

          <text
            x="276"
            y="339"
            text-anchor="middle"
            font-size="14"
            font-weight="900"
            fill="#BE185D"
          >动物细胞没有细胞壁保护</text>
        </g>

        <!-- 水分子、溶质颗粒和净移动箭头 -->
        <g data-external-particles></g>
        <g data-internal-particles></g>
        <g data-water-particles></g>
        <g data-net-flow></g>

        <!-- 右侧浓度比较与体积变化 -->
        <g transform="translate(484 102)">
          <text
            x="0"
            y="0"
            font-size="14"
            font-weight="900"
            fill="#334155"
          >细胞内外比较</text>

          <text
            x="0"
            y="36"
            font-size="12"
            font-weight="800"
            fill="#64748B"
          >细胞外溶质</text>

          <rect
            x="0"
            y="47"
            width="150"
            height="17"
            rx="8.5"
            fill="#E2E8F0"
          />

          <rect
            data-external-bar
            x="0"
            y="47"
            width="0"
            height="17"
            rx="8.5"
            fill="#F59E0B"
          />

          <text
            x="0"
            y="92"
            font-size="12"
            font-weight="800"
            fill="#64748B"
          >细胞内溶质</text>

          <rect
            x="0"
            y="103"
            width="150"
            height="17"
            rx="8.5"
            fill="#E2E8F0"
          />

          <rect
            data-internal-bar
            x="0"
            y="103"
            width="0"
            height="17"
            rx="8.5"
            fill="#8B5CF6"
          />

          <text
            x="0"
            y="151"
            font-size="12"
            font-weight="800"
            fill="#64748B"
          >相对细胞体积</text>

          <rect
            x="0"
            y="162"
            width="150"
            height="17"
            rx="8.5"
            fill="#E2E8F0"
          />

          <rect
            data-volume-bar
            x="0"
            y="162"
            width="0"
            height="17"
            rx="8.5"
            fill="#0EA5E9"
          />

          <line
            x1="75"
            y1="154"
            x2="75"
            y2="187"
            stroke="#10B981"
            stroke-width="3"
          />

          <text
            x="75"
            y="205"
            text-anchor="middle"
            font-size="10"
            font-weight="900"
            fill="#047857"
          >初始体积</text>

          <text
            data-volume-value
            x="0"
            y="238"
            font-size="19"
            font-weight="900"
            fill="#075985"
          ></text>

          <text
            data-flow-label
            x="0"
            y="270"
            font-size="13"
            font-weight="900"
            fill="#0369A1"
          ></text>
        </g>

        <g transform="translate(28 386)">
          <circle cx="7" cy="7" r="7" fill="#38BDF8"/>
          <text
            x="23"
            y="12"
            font-size="13"
            font-weight="800"
            fill="#475569"
          >水分子</text>
        </g>

        <g transform="translate(138 386)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text
            x="23"
            y="12"
            font-size="13"
            font-weight="800"
            fill="#475569"
          >外界溶质</text>
        </g>

        <g transform="translate(278 386)">
          <circle cx="7" cy="7" r="7" fill="#8B5CF6"/>
          <text
            x="23"
            y="12"
            font-size="13"
            font-weight="800"
            fill="#475569"
          >细胞内溶质</text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var externalInput=root.querySelector(
      '[data-external]'
    );
    var internalInput=root.querySelector(
      '[data-internal]'
    );
    var permeabilityInput=root.querySelector(
      '[data-permeability]'
    );
    var timeInput=root.querySelector('[data-time]');

    var externalValue=root.querySelector(
      '[data-external-value]'
    );
    var internalValue=root.querySelector(
      '[data-internal-value]'
    );
    var permeabilityValue=root.querySelector(
      '[data-permeability-value]'
    );
    var timeValue=root.querySelector(
      '[data-time-value]'
    );

    var buttons=root.querySelectorAll('[data-mode]');
    var solutionState=root.querySelector(
      '[data-solution-state]'
    );
    var cellState=root.querySelector(
      '[data-cell-state]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var externalLabel=root.querySelector(
      '[data-external-label]'
    );

    var plantLayer=root.querySelector(
      '[data-plant-layer]'
    );
    var animalLayer=root.querySelector(
      '[data-animal-layer]'
    );

    var plantMembrane=root.querySelector(
      '[data-plant-membrane]'
    );
    var vacuole=root.querySelector('[data-vacuole]');
    var plantNucleus=root.querySelector(
      '[data-plant-nucleus]'
    );

    var animalCell=root.querySelector(
      '[data-animal-cell]'
    );
    var animalInner=root.querySelector(
      '[data-animal-inner]'
    );

    var externalParticles=root.querySelector(
      '[data-external-particles]'
    );
    var internalParticles=root.querySelector(
      '[data-internal-particles]'
    );
    var waterParticles=root.querySelector(
      '[data-water-particles]'
    );
    var netFlow=root.querySelector('[data-net-flow]');

    var externalBar=root.querySelector(
      '[data-external-bar]'
    );
    var internalBar=root.querySelector(
      '[data-internal-bar]'
    );
    var volumeBar=root.querySelector(
      '[data-volume-bar]'
    );
    var volumeValue=root.querySelector(
      '[data-volume-value]'
    );
    var flowLabel=root.querySelector(
      '[data-flow-label]'
    );

    var mode='plant';

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    /**
     * 根据细胞内外溶质相对浓度判断外界环境。
     *
     * 差值绝对值小于0.5时视为近似等渗，
     * 只用于课堂演示，不是严格实验判据。
     */
    function classifySolution(external,internal){
      var difference=external-internal;

      if(Math.abs(difference)<.5){
        return 'isotonic';
      }

      return difference>0
        ?'hypertonic'
        :'hypotonic';
    }

    /**
     * 计算净水流方向和强度。
     *
     * 溶质相对浓度较高的一侧，水的相对含量较低；
     * 水分子总体从低溶质一侧向高溶质一侧净移动。
     */
    function calculateFlux(
      external,
      internal,
      permeability,
      elapsed
    ){
      var gradient=internal-external;
      var magnitude=
        Math.abs(gradient)/20
        *permeability/100
        *elapsed/100;

      return {
        gradient:gradient,
        magnitude:clamp(magnitude,0,1)
      };
    }

    /**
     * 计算相对细胞体积。
     *
     * 植物细胞因细胞壁限制，体积变化范围较小；
     * 动物细胞没有细胞壁，变化范围更大。
     */
    function calculateVolume(flux,cellType){
      var direction=flux.gradient===0
        ?0
        :flux.gradient>0
          ?1
          :-1;

      var range=cellType==='plant'?.28:.52;

      return clamp(
        1+direction*flux.magnitude*range,
        cellType==='plant'?.68:.46,
        cellType==='plant'?1.25:1.48
      );
    }

    function buildExternalParticles(count){
      var html='';

      for(var i=0;i<count;i++){
        var col=i%8;
        var row=Math.floor(i/8);
        var x=129+col*39+(row%2)*8;
        var y=182+row*31;

        if(x>425 || y>337){
          continue;
        }

        html+='<circle class="oc-solute" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%3)
          +'" fill="#F59E0B" stroke="#B45309"'
          +' stroke-width="1.5"/>';
      }

      return html;
    }

    function buildInternalParticles(
      count,
      centerX,
      centerY,
      radiusX,
      radiusY
    ){
      var html='';

      for(var i=0;i<count;i++){
        var angle=i*2.399;
        var radial=.22+(i%5)*.14;
        var x=centerX+Math.cos(angle)*radiusX*radial;
        var y=centerY+Math.sin(angle)*radiusY*radial;

        html+='<circle class="oc-solute" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%2)
          +'" fill="#8B5CF6" stroke="#5B21B6"'
          +' stroke-width="1.5"/>';
      }

      return html;
    }

    function buildWaterParticles(
      count,
      fluxDirection,
      fluxMagnitude
    ){
      var html='';

      for(var i=0;i<count;i++){
        var side=i%2===0?'outside':'inside';
        var x;
        var y;

        if(side==='outside'){
          x=132+(i*43)%310;
          y=170+(i*29)%168;
        }else{
          var angle=i*1.73;
          x=276+Math.cos(angle)*(48+(i%3)*13);
          y=252+Math.sin(angle)*(34+(i%3)*9);
        }

        var opacity=.28+.62*fluxMagnitude;

        html+='<circle class="oc-water" cx="'+x
          +'" cy="'+y+'" r="'+(4+i%3)
          +'" fill="#38BDF8" stroke="#0284C7"'
          +' stroke-width="1.2" opacity="'+opacity+'"/>';
      }

      if(fluxDirection==='in'){
        for(var j=0;j<5;j++){
          var inY=202+j*22;

          html+='<path d="M138 '+inY
            +' C172 '+(inY-9)+' 190 '+(inY+4)
            +' 207 '+(inY+10)
            +'" fill="none" stroke="#0284C7"'
            +' stroke-width="'+(2.5+fluxMagnitude*3)
            +'" marker-end="url(#${rootId}-arrow-blue)"'
            +' opacity="'+(.35+.65*fluxMagnitude)+'"/>';
        }
      }else if(fluxDirection==='out'){
        for(var k=0;k<5;k++){
          var outY=202+k*22;

          html+='<path d="M345 '+(outY+8)
            +' C374 '+(outY-5)+' 398 '+outY
            +' 424 '+(outY-4)
            +'" fill="none" stroke="#0284C7"'
            +' stroke-width="'+(2.5+fluxMagnitude*3)
            +'" marker-end="url(#${rootId}-arrow-blue)"'
            +' opacity="'+(.35+.65*fluxMagnitude)+'"/>';
        }
      }

      return html;
    }

    function buildNetFlow(
      direction,
      magnitude
    ){
      if(direction==='none'){
        return ''
          +'<path d="M228 145 H326"'
          +' fill="none" stroke="#10B981"'
          +' stroke-width="5" stroke-dasharray="8 7"/>'
          +'<text x="277" y="135" text-anchor="middle"'
          +' font-size="13" font-weight="900" fill="#047857">'
          +'双向运动近似平衡</text>';
      }

      var fromX=direction==='in'?134:352;
      var toX=direction==='in'?205:424;
      var marker='url(#${rootId}-arrow-orange)';

      return ''
        +'<path d="M'+fromX+' 148 H'+toX+'"'
        +' fill="none" stroke="#F59E0B"'
        +' stroke-width="'+(4+magnitude*4)
        +'" marker-end="'+marker+'"'
        +' stroke-dasharray="9 7"/>'
        +'<text x="278" y="135" text-anchor="middle"'
        +' font-size="13" font-weight="900" fill="#B45309">'
        +(direction==='in'
          ?'水分子净进入细胞'
          :'水分子净离开细胞')
        +'</text>';
    }

    function updatePlantCell(volume,state){
      var shrink=clamp(
        (1-volume)/.32,
        0,
        1
      );
      var swell=clamp(
        (volume-1)/.25,
        0,
        1
      );

      var membraneX=194+shrink*21-swell*3;
      var membraneY=200+shrink*15-swell*2;
      var membraneWidth=164-shrink*42+swell*6;
      var membraneHeight=104-shrink*30+swell*4;

      plantMembrane.setAttribute(
        'x',
        String(membraneX)
      );
      plantMembrane.setAttribute(
        'y',
        String(membraneY)
      );
      plantMembrane.setAttribute(
        'width',
        String(membraneWidth)
      );
      plantMembrane.setAttribute(
        'height',
        String(membraneHeight)
      );

      var vacuoleX=213+shrink*24-swell*8;
      var vacuoleY=218+shrink*14-swell*6;
      var vacuoleWidth=126-shrink*48+swell*16;
      var vacuoleHeight=68-shrink*28+swell*12;

      vacuole.setAttribute('x',String(vacuoleX));
      vacuole.setAttribute('y',String(vacuoleY));
      vacuole.setAttribute(
        'width',
        String(vacuoleWidth)
      );
      vacuole.setAttribute(
        'height',
        String(vacuoleHeight)
      );

      plantNucleus.setAttribute(
        'cx',
        String(223+shrink*22-swell*5)
      );
      plantNucleus.setAttribute(
        'cy',
        String(241+shrink*5+swell*5)
      );

      plantMembrane.setAttribute(
        'stroke',
        state==='hypertonic'
          ?'#EF4444'
          :state==='hypotonic'
            ?'#10B981'
            :'#22C55E'
      );
    }

    function updateAnimalCell(volume,state){
      var rx=88*volume;
      var ry=68*volume;

      animalCell.setAttribute(
        'rx',
        String(clamp(rx,43,128))
      );
      animalCell.setAttribute(
        'ry',
        String(clamp(ry,34,101))
      );

      animalInner.setAttribute(
        'rx',
        String(clamp(rx-15,28,113))
      );
      animalInner.setAttribute(
        'ry',
        String(clamp(ry-15,20,86))
      );

      animalCell.setAttribute(
        'stroke',
        state==='hypertonic'
          ?'#7C3AED'
          :state==='hypotonic'
            ?'#EF4444'
            :'#DB2777'
      );

      animalCell.setAttribute(
        'stroke-dasharray',
        state==='hypertonic'?'8 5':''
      );
    }

    function update(){
      var external=Number(externalInput.value);
      var internal=Number(internalInput.value);
      var permeability=Number(
        permeabilityInput.value
      );
      var elapsed=Number(timeInput.value);

      var solution=classifySolution(
        external,
        internal
      );
      var flux=calculateFlux(
        external,
        internal,
        permeability,
        elapsed
      );

      var direction=solution==='isotonic'
        ?'none'
        :solution==='hypotonic'
          ?'in'
          :'out';

      var volume=calculateVolume(flux,mode);

      externalValue.textContent=
        external.toFixed(1);
      internalValue.textContent=
        internal.toFixed(1);
      permeabilityValue.textContent=
        permeability.toFixed(0)+'%';
      timeValue.textContent=
        elapsed.toFixed(0)+'%';

      root.style.setProperty(
        '--oc-speed',
        clamp(
          2.4-flux.magnitude*1.7,
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

      plantLayer.style.display=
        mode==='plant'?'':'none';
      animalLayer.style.display=
        mode==='animal'?'':'none';

      externalLabel.textContent=
        '外界溶液：相对溶质浓度 '
        +external.toFixed(1);

      externalBar.setAttribute(
        'width',
        String(150*external/20)
      );
      internalBar.setAttribute(
        'width',
        String(150*internal/20)
      );

      var normalizedVolume=clamp(
        (volume-.5)/1,
        0,
        1
      );

      volumeBar.setAttribute(
        'width',
        String(150*normalizedVolume)
      );

      volumeBar.setAttribute(
        'fill',
        volume>1.18
          ?'#10B981'
          :volume<.82
            ?'#EF4444'
            :'#0EA5E9'
      );

      volumeValue.textContent=
        '相对体积 '
        +(volume*100).toFixed(0)+'%';

      var solutionText=
        solution==='hypertonic'
          ?'高渗'
          :solution==='hypotonic'
            ?'低渗'
            :'等渗';

      solutionState.textContent=solutionText;

      var stateText='';
      var explanation='';

      if(mode==='plant'){
        if(solution==='hypertonic'){
          stateText=volume<.78
            ?'明显质壁分离'
            :'失水';

          explanation=
            '外界溶质浓度高于细胞内，水分子净离开细胞，原生质层收缩并可能与细胞壁分离。';
        }else if(solution==='hypotonic'){
          stateText=volume>1.16
            ?'充分膨胀'
            :'吸水';

          explanation=
            '外界溶质浓度低于细胞内，水分子净进入细胞，液泡增大并产生膨压；细胞壁限制细胞继续膨胀。';
        }else{
          stateText='近似平衡';
          explanation=
            '细胞内外相对溶质浓度接近，水分子仍双向运动，但净移动接近零。';
        }
      }else{
        if(solution==='hypertonic'){
          stateText=volume<.7
            ?'严重皱缩'
            :'皱缩';

          explanation=
            '外界溶质浓度高于细胞内，水分子净离开动物细胞，细胞体积减小并发生皱缩。';
        }else if(solution==='hypotonic'){
          stateText=volume>1.35
            ?'破裂风险'
            :'膨胀';

          explanation=
            '外界溶质浓度低于细胞内，水分子净进入动物细胞；动物细胞没有细胞壁，过度吸水时可能破裂。';
        }else{
          stateText='近似正常';
          explanation=
            '细胞内外相对溶质浓度接近，水分子的净移动接近零，细胞体积相对稳定。';
        }
      }

      cellState.textContent=stateText;

      var externalCount=Math.floor(
        2+external*1.15
      );
      var internalCount=Math.floor(
        2+internal*.82
      );

      externalParticles.innerHTML=
        buildExternalParticles(externalCount);

      internalParticles.innerHTML=
        buildInternalParticles(
          internalCount,
          276,
          252,
          mode==='plant'?72:68*volume,
          mode==='plant'?44:49*volume
        );

      waterParticles.innerHTML=
        buildWaterParticles(
          Math.floor(8+permeability/8),
          direction,
          flux.magnitude
        );

      netFlow.innerHTML=buildNetFlow(
        direction,
        flux.magnitude
      );

      if(mode==='plant'){
        updatePlantCell(volume,solution);

        title.textContent=
          '植物细胞的吸水与失水';
        summary.textContent=
          '细胞膜具有选择透过性，细胞壁限制植物细胞过度膨胀';
      }else{
        updateAnimalCell(volume,solution);

        title.textContent=
          '动物细胞的吸水与失水';
        summary.textContent=
          '动物细胞没有细胞壁，极端渗透环境下体积变化更明显';
      }

      flowLabel.textContent=
        direction==='in'
          ?'净水流方向：细胞外 → 细胞内'
          :direction==='out'
            ?'净水流方向：细胞内 → 细胞外'
            :'净水流方向：近似平衡';

      var rateNote='';

      if(elapsed<15){
        rateNote=
          '作用时间较短，细胞体积变化尚不明显。';
      }else if(permeability<25){
        rateNote=
          '细胞膜透水性较低，水分子净移动速度受到限制。';
      }else if(flux.magnitude>.7){
        rateNote=
          '浓度差、膜透水性和作用时间共同使渗透效应较明显。';
      }else{
        rateNote=
          '当前渗透效应处于较低或中等水平。';
      }

      result.innerHTML=explanation
        +'<br>'+rateNote
        +' 本模型假设溶质不能自由通过细胞膜，数值仅用于课堂比较。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    externalInput.oninput=update;
    internalInput.oninput=update;
    permeabilityInput.oninput=update;
    timeInput.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
