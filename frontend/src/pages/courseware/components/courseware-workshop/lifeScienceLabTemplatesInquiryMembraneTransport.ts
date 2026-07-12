/**
 * lifeScienceLabTemplatesInquiryMembraneTransport.ts
 *
 * 平面生命科学实验室：物质跨膜运输。
 *
 * 教学目标：
 * 1. 比较自由扩散、协助扩散和主动运输的方向、条件及能量需求；
 * 2. 理解自由扩散和协助扩散通常顺浓度梯度进行；
 * 3. 理解协助扩散需要通道蛋白或载体蛋白，但不直接消耗ATP；
 * 4. 理解主动运输需要载体或泵蛋白，并消耗细胞代谢提供的能量；
 * 5. 观察浓度差、运输蛋白数量和ATP供应对相对运输速率的影响；
 * 6. 区分物质运动、净运输方向和动态平衡。
 *
 * 教学边界：
 * 1. 本模型中的浓度和运输速率均为相对教学单位；
 * 2. 自由扩散适用于部分小分子、脂溶性分子等，不代表所有物质均可直接穿膜；
 * 3. 协助扩散具有选择性，且运输速率会受到运输蛋白数量限制；
 * 4. 主动运输模式自动选择从相对低浓度一侧向高浓度一侧运输，
 *    用于突出“逆浓度梯度和消耗能量”，不对应某一种具体离子泵；
 * 5. 当膜两侧浓度相等时，分子仍可双向运动，但被动运输的净运输接近零；
 * 6. 本模型不展开胞吞、胞吐、继发性主动运输、电化学梯度和膜电位。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式和图片；
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
 * 构建完全限定在当前rootId内的样式。
 *
 * 独立预览保持左侧控制栏；
 * 应用到课件后由lifeScienceLabUtils.ts中的公共覆盖层
 * 调整为“上方实验主体 + 底部课堂控制条”。
 */
function membraneTransportStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #A7F3D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#D1FAE5,#E0F2FE);border-bottom:1px solid #A7F3D0}'
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
    + '#' + rootId + ' .mt-particle{animation:' + rootId + '-particle var(--mt-speed,1.6s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .mt-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--mt-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .mt-atp{animation:' + rootId + '-atp 1.2s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-particle{from{opacity:.48}to{opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-atp{from{transform:scale(.88);opacity:.55}to{transform:scale(1.08);opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_MEMBRANE_TRANSPORT:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-membrane-transport',
    group: '🧪 实验探究',
    name: '物质跨膜运输',
    emoji: '🚪',
    desc: '调节膜两侧浓度、运输蛋白和ATP供应，比较自由扩散、协助扩散与主动运输',
    params: [
      {
        key: 'externalConcentration',
        label: '细胞外物质浓度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
        hint: '相对教学单位',
      },
      {
        key: 'internalConcentration',
        label: '细胞内物质浓度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 28,
        hint: '相对教学单位',
      },
      {
        key: 'transportProteinLevel',
        label: '运输蛋白数量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'atpSupply',
        label: 'ATP供应水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
    ],

    buildHTML: (params, rootId) => {
      const externalConcentration = num(
        params,
        'externalConcentration',
        72,
      )
      const internalConcentration = num(
        params,
        'internalConcentration',
        28,
      )
      const transportProteinLevel = num(
        params,
        'transportProteinLevel',
        68,
      )
      const atpSupply = num(
        params,
        'atpSupply',
        76,
      )

      return `
<div id="${rootId}">
${membraneTransportStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🚪 物质跨膜运输方式比较</div>
    <div class="bl-note">浓度和速率均为相对教学单位</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>细胞外物质浓度</span>
          <span class="bl-value" data-external-value></span>
        </div>
        <input
          data-external
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(externalConcentration)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>细胞内物质浓度</span>
          <span class="bl-value" data-internal-value></span>
        </div>
        <input
          data-internal
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(internalConcentration)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>运输蛋白数量</span>
          <span class="bl-value" data-protein-value></span>
        </div>
        <input
          data-protein
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(transportProteinLevel)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>ATP供应水平</span>
          <span class="bl-value" data-atp-value></span>
        </div>
        <input
          data-atp
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(atpSupply)}"
        >
      </div>

      <div class="bl-subtitle">选择运输方式</div>

      <div class="bl-buttons">
        <button
          type="button"
          class="bl-button active"
          data-mode="simple"
        >自由扩散</button>

        <button
          type="button"
          class="bl-button"
          data-mode="facilitated"
        >协助扩散</button>

        <button
          type="button"
          class="bl-button"
          data-mode="active"
        >主动运输</button>
      </div>

      <div class="bl-status">
        <div class="bl-card">
          <b data-direction></b>
          <span>净运输方向</span>
        </div>

        <div class="bl-card">
          <b data-rate></b>
          <span>相对运输速率</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="自由扩散协助扩散与主动运输互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-green"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#059669"/>
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
            id="${rootId}-outside"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#EFF6FF"/>
            <stop offset="100%" stop-color="#DBEAFE"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-inside"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#ECFDF5"/>
            <stop offset="100%" stop-color="#D1FAE5"/>
          </linearGradient>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#065F46"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <rect
          x="0"
          y="78"
          width="680"
          height="122"
          fill="url(#${rootId}-outside)"
        />

        <rect
          x="0"
          y="252"
          width="680"
          height="122"
          fill="url(#${rootId}-inside)"
        />

        <text
          x="26"
          y="35"
          data-title
          font-size="25"
          font-weight="900"
          fill="#065F46"
        ></text>

        <text
          x="26"
          y="64"
          data-summary
          font-size="14"
          font-weight="800"
          fill="#475569"
        ></text>

        <text
          x="30"
          y="104"
          font-size="14"
          font-weight="900"
          fill="#1D4ED8"
        >细胞外</text>

        <text
          x="30"
          y="354"
          font-size="14"
          font-weight="900"
          fill="#047857"
        >细胞内</text>

        <!-- 磷脂双分子层 -->
        <g data-membrane></g>

        <!-- 物质颗粒 -->
        <g data-external-particles></g>
        <g data-internal-particles></g>

        <!-- 通道蛋白、载体蛋白和泵蛋白 -->
        <g
          data-protein-layer
          filter="url(#${rootId}-shadow)"
        ></g>

        <!-- 净运输路径与ATP -->
        <g data-flow-layer></g>
        <g data-atp-layer></g>

        <!-- 右侧参数比较 -->
        <g transform="translate(500 104)">
          <text
            x="0"
            y="0"
            font-size="14"
            font-weight="900"
            fill="#334155"
          >条件比较</text>

          <text
            x="0"
            y="34"
            font-size="11"
            font-weight="800"
            fill="#64748B"
          >细胞外浓度</text>

          <rect
            x="0"
            y="43"
            width="145"
            height="16"
            rx="8"
            fill="#E2E8F0"
          />

          <rect
            data-external-bar
            x="0"
            y="43"
            width="0"
            height="16"
            rx="8"
            fill="#3B82F6"
          />

          <text
            x="0"
            y="86"
            font-size="11"
            font-weight="800"
            fill="#64748B"
          >细胞内浓度</text>

          <rect
            x="0"
            y="95"
            width="145"
            height="16"
            rx="8"
            fill="#E2E8F0"
          />

          <rect
            data-internal-bar
            x="0"
            y="95"
            width="0"
            height="16"
            rx="8"
            fill="#10B981"
          />

          <text
            x="0"
            y="138"
            font-size="11"
            font-weight="800"
            fill="#64748B"
          >运输蛋白</text>

          <rect
            x="0"
            y="147"
            width="145"
            height="16"
            rx="8"
            fill="#E2E8F0"
          />

          <rect
            data-protein-bar
            x="0"
            y="147"
            width="0"
            height="16"
            rx="8"
            fill="#8B5CF6"
          />

          <text
            x="0"
            y="190"
            font-size="11"
            font-weight="800"
            fill="#64748B"
          >ATP供应</text>

          <rect
            x="0"
            y="199"
            width="145"
            height="16"
            rx="8"
            fill="#E2E8F0"
          />

          <rect
            data-atp-bar
            x="0"
            y="199"
            width="0"
            height="16"
            rx="8"
            fill="#F59E0B"
          />

          <text
            data-energy-label
            x="0"
            y="251"
            font-size="13"
            font-weight="900"
            fill="#B45309"
          ></text>
        </g>

        <g transform="translate(26 391)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text
            x="23"
            y="12"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >跨膜物质</text>
        </g>

        <g transform="translate(154 391)">
          <circle cx="7" cy="7" r="7" fill="#8B5CF6"/>
          <text
            x="23"
            y="12"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >运输蛋白</text>
        </g>

        <g transform="translate(292 391)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text
            x="23"
            y="12"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >ATP能量</text>
        </g>

        <text
          x="425"
          y="403"
          data-stage-note
          font-size="13"
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

    var externalInput=root.querySelector(
      '[data-external]'
    );
    var internalInput=root.querySelector(
      '[data-internal]'
    );
    var proteinInput=root.querySelector(
      '[data-protein]'
    );
    var atpInput=root.querySelector('[data-atp]');

    var externalValue=root.querySelector(
      '[data-external-value]'
    );
    var internalValue=root.querySelector(
      '[data-internal-value]'
    );
    var proteinValue=root.querySelector(
      '[data-protein-value]'
    );
    var atpValue=root.querySelector(
      '[data-atp-value]'
    );

    var buttons=root.querySelectorAll('[data-mode]');
    var directionValue=root.querySelector(
      '[data-direction]'
    );
    var rateValue=root.querySelector('[data-rate]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var membrane=root.querySelector('[data-membrane]');
    var externalParticles=root.querySelector(
      '[data-external-particles]'
    );
    var internalParticles=root.querySelector(
      '[data-internal-particles]'
    );
    var proteinLayer=root.querySelector(
      '[data-protein-layer]'
    );
    var flowLayer=root.querySelector(
      '[data-flow-layer]'
    );
    var atpLayer=root.querySelector(
      '[data-atp-layer]'
    );

    var externalBar=root.querySelector(
      '[data-external-bar]'
    );
    var internalBar=root.querySelector(
      '[data-internal-bar]'
    );
    var proteinBar=root.querySelector(
      '[data-protein-bar]'
    );
    var atpBar=root.querySelector('[data-atp-bar]');
    var energyLabel=root.querySelector(
      '[data-energy-label]'
    );
    var stageNote=root.querySelector(
      '[data-stage-note]'
    );

    var mode='simple';

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    /**
     * 判断被动运输的净方向。
     *
     * 浓度差小于2个相对单位时视为近似平衡。
     */
    function calculatePassiveDirection(
      external,
      internal
    ){
      var difference=external-internal;

      if(Math.abs(difference)<2){
        return 'balanced';
      }

      return difference>0
        ?'outsideToInside'
        :'insideToOutside';
    }

    /**
     * 主动运输方向自动选择为低浓度一侧指向高浓度一侧，
     * 用于突出逆浓度梯度。
     *
     * 浓度相等时，默认从细胞内向细胞外运输，
     * 表达主动运输可以建立新的浓度梯度。
     */
    function calculateActiveDirection(
      external,
      internal
    ){
      if(external<internal){
        return 'outsideToInside';
      }

      return 'insideToOutside';
    }

    /**
     * 计算三种运输方式的相对速率。
     *
     * 自由扩散：
     *   主要受浓度差影响，不使用运输蛋白和ATP参数。
     *
     * 协助扩散：
     *   顺浓度梯度，受运输蛋白数量限制；
     *   使用饱和型函数表示蛋白数量增加后的边际效应下降。
     *
     * 主动运输：
     *   受运输蛋白和ATP供应共同限制；
     *   逆浓度梯度越陡，维持运输的相对难度越大。
     */
    function calculateTransportRate(
      transportMode,
      external,
      internal,
      proteinLevel,
      atpLevel
    ){
      var gradient=Math.abs(external-internal);
      var gradientFactor=gradient/100;
      var proteinFactor=
        proteinLevel/(proteinLevel+28);
      var energyFactor=
        atpLevel/(atpLevel+25);

      if(transportMode==='simple'){
        return 100
          *gradientFactor
          *.72;
      }

      if(transportMode==='facilitated'){
        return 100
          *gradientFactor
          *proteinFactor
          *1.18;
      }

      var resistanceFactor=
        1-.38*gradientFactor;

      return 100
        *proteinFactor
        *energyFactor
        *resistanceFactor;
    }

    function buildMembrane(){
      var html='';
      var startX=42;
      var count=21;
      var gap=21;

      for(var i=0;i<count;i++){
        var x=startX+i*gap;

        html+='<circle cx="'+x+'" cy="207" r="8'
          +'" fill="#38BDF8" stroke="#0369A1"'
          +' stroke-width="2"/>';

        html+='<line x1="'+(x-3)+'" y1="215"'
          +' x2="'+(x-4)+'" y2="233"'
          +' stroke="#F59E0B" stroke-width="3.5"'
          +' stroke-linecap="round"/>';

        html+='<line x1="'+(x+3)+'" y1="215"'
          +' x2="'+(x+4)+'" y2="233"'
          +' stroke="#F59E0B" stroke-width="3.5"'
          +' stroke-linecap="round"/>';

        html+='<circle cx="'+x+'" cy="245" r="8'
          +'" fill="#38BDF8" stroke="#0369A1"'
          +' stroke-width="2"/>';

        html+='<line x1="'+(x-3)+'" y1="237"'
          +' x2="'+(x-4)+'" y2="219"'
          +' stroke="#F59E0B" stroke-width="3.5"'
          +' stroke-linecap="round"/>';

        html+='<line x1="'+(x+3)+'" y1="237"'
          +' x2="'+(x+4)+'" y2="219"'
          +' stroke="#F59E0B" stroke-width="3.5"'
          +' stroke-linecap="round"/>';
      }

      html+='<rect x="30" y="198" width="454"'
        +' height="56" rx="12" fill="none"'
        +' stroke="#0F766E" stroke-width="2.5"'
        +' opacity=".42"/>';

      return html;
    }

    function buildParticles(
      count,
      area
    ){
      var html='';
      var topArea=area==='outside';

      for(var i=0;i<count;i++){
        var col=i%10;
        var row=Math.floor(i/10);
        var x=76+col*39+(row%2)*8;
        var y=topArea
          ?116+row*28+(col%2)*5
          :278+row*28+(col%2)*5;

        if(x>468){
          continue;
        }

        if(topArea && y>184){
          continue;
        }

        if(!topArea && y>360){
          continue;
        }

        html+='<circle class="mt-particle"'
          +' cx="'+x+'" cy="'+y+'"'
          +' r="'+(5+i%3)+'"'
          +' fill="#2563EB" stroke="#1E40AF"'
          +' stroke-width="1.5"/>';

        if(i%5===0){
          html+='<text x="'+(x+8)+'" y="'+(y+4)
            +'" font-size="8" font-weight="900"'
            +' fill="#1E3A8A">S</text>';
        }
      }

      return html;
    }

    function buildSimpleProtein(){
      return ''
        +'<text x="172" y="190"'
        +' font-size="12" font-weight="900"'
        +' fill="#64748B">'
        +'直接穿过磷脂双分子层'
        +'</text>';
    }

    function buildFacilitatedProtein(
      proteinLevel
    ){
      var html='';
      var count=Math.max(
        1,
        Math.floor(proteinLevel/25)
      );

      for(var i=0;i<count;i++){
        var x=160+i*72;

        html+='<g>'
          +'<rect x="'+(x-21)+'" y="192"'
          +' width="42" height="68" rx="18"'
          +' fill="#DDD6FE" stroke="#7C3AED"'
          +' stroke-width="5"/>'
          +'<rect x="'+(x-7)+'" y="195"'
          +' width="14" height="62" rx="7"'
          +' fill="#FFFFFF" stroke="#A78BFA"'
          +' stroke-width="2"/>'
          +'</g>';
      }

      html+='<text x="146" y="184"'
        +' font-size="12" font-weight="900"'
        +' fill="#6D28D9">'
        +'通道或载体具有选择性'
        +'</text>';

      return html;
    }

    function buildActiveProtein(
      proteinLevel
    ){
      var width=54+proteinLevel*.18;

      return ''
        +'<path d="M220 190'
        +' C180 202 184 250 220 262'
        +' L'+(220+width)+' 262'
        +' C'+(260+width)+' 248 '
        +(258+width)+' 204 '
        +(220+width)+' 190 Z"'
        +' fill="#EDE9FE" stroke="#7C3AED"'
        +' stroke-width="6"/>'
        +'<path d="M242 207'
        +' C225 219 227 236 244 246'
        +' C260 236 262 218 246 207 Z"'
        +' fill="#FFFFFF" stroke="#A78BFA"'
        +' stroke-width="3"/>'
        +'<text x="'+(247+width/2)+'" y="181"'
        +' text-anchor="middle" font-size="12"'
        +' font-weight="900" fill="#6D28D9">'
        +'载体或泵蛋白'
        +'</text>';
    }

    function buildFlow(
      direction,
      rate,
      transportMode
    ){
      if(
        transportMode!=='active'
        && direction==='balanced'
      ){
        return ''
          +'<path class="mt-flow"'
          +' d="M160 193 V258"'
          +' fill="none" stroke="#059669"'
          +' stroke-width="4"'
          +' marker-end="url(#${rootId}-arrow-green)"/>'
          +'<path class="mt-flow"'
          +' d="M340 258 V193"'
          +' fill="none" stroke="#059669"'
          +' stroke-width="4"'
          +' marker-end="url(#${rootId}-arrow-green)"/>'
          +'<text x="250" y="278"'
          +' text-anchor="middle" font-size="12"'
          +' font-weight="900" fill="#047857">'
          +'双向运动，净运输接近零'
          +'</text>';
      }

      var outsideToInside=
        direction==='outsideToInside';
      var x=transportMode==='simple'
        ?250
        :transportMode==='facilitated'
          ?232
          :258;

      var startY=outsideToInside?179:275;
      var endY=outsideToInside?273:181;
      var color=transportMode==='active'
        ?'#F59E0B'
        :'#059669';
      var marker=transportMode==='active'
        ?'url(#${rootId}-arrow-orange)'
        :'url(#${rootId}-arrow-green)';
      var thickness=3.5+rate/28;

      return ''
        +'<path class="mt-flow"'
        +' d="M'+x+' '+startY+' V'+endY+'"'
        +' fill="none" stroke="'+color+'"'
        +' stroke-width="'+thickness+'"'
        +' marker-end="'+marker+'"/>'
        +'<text x="'+(x+20)+'" y="226"'
        +' font-size="11" font-weight="900"'
        +' fill="'+color+'">'
        +rate.toFixed(0)
        +'</text>';
    }

    function buildATP(
      atpLevel,
      transportMode
    ){
      if(transportMode!=='active'){
        return '';
      }

      var html='';
      var count=Math.max(
        1,
        Math.floor(atpLevel/16)
      );

      for(var i=0;i<count;i++){
        var x=122+(i%5)*50;
        var y=320+Math.floor(i/5)*25;

        html+='<g class="mt-atp"'
          +' transform="translate('+x+' '+y+')">'
          +'<polygon points="0,-10 9,-4 6,7'
          +' -6,7 -9,-4"'
          +' fill="#FACC15" stroke="#CA8A04"'
          +' stroke-width="2"/>'
          +'<text x="0" y="4" text-anchor="middle"'
          +' font-size="8" font-weight="900"'
          +' fill="#854D0E">ATP</text>'
          +'</g>';
      }

      html+='<path d="M334 325 C350 292 343 268 316 248"'
        +' fill="none" stroke="#F59E0B"'
        +' stroke-width="4" stroke-dasharray="7 6"'
        +' marker-end="url(#${rootId}-arrow-orange)"/>';

      return html;
    }

    function directionLabel(direction){
      if(direction==='outsideToInside'){
        return '外 → 内';
      }

      if(direction==='insideToOutside'){
        return '内 → 外';
      }

      return '近似平衡';
    }

    function update(){
      var external=Number(externalInput.value);
      var internal=Number(internalInput.value);
      var proteinLevel=Number(proteinInput.value);
      var atpLevel=Number(atpInput.value);

      var passiveDirection=
        calculatePassiveDirection(
          external,
          internal
        );

      var direction=mode==='active'
        ?calculateActiveDirection(
          external,
          internal
        )
        :passiveDirection;

      var rate=calculateTransportRate(
        mode,
        external,
        internal,
        proteinLevel,
        atpLevel
      );

      rate=clamp(rate,0,100);

      externalValue.textContent=
        external.toFixed(0);
      internalValue.textContent=
        internal.toFixed(0);
      proteinValue.textContent=
        proteinLevel.toFixed(0)+'%';
      atpValue.textContent=
        atpLevel.toFixed(0)+'%';

      directionValue.textContent=
        directionLabel(direction);
      rateValue.textContent=
        rate.toFixed(0);

      externalBar.setAttribute(
        'width',
        String(145*external/100)
      );
      internalBar.setAttribute(
        'width',
        String(145*internal/100)
      );
      proteinBar.setAttribute(
        'width',
        String(145*proteinLevel/100)
      );
      atpBar.setAttribute(
        'width',
        String(145*atpLevel/100)
      );

      root.style.setProperty(
        '--mt-speed',
        clamp(
          2.4-rate/70,
          .55,
          2.4
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--mt-flow-speed',
        clamp(
          2.4-rate/65,
          .5,
          2.4
        ).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-mode')===mode
        );
      }

      membrane.innerHTML=buildMembrane();

      externalParticles.innerHTML=
        buildParticles(
          Math.floor(2+external/7),
          'outside'
        );

      internalParticles.innerHTML=
        buildParticles(
          Math.floor(2+internal/7),
          'inside'
        );

      if(mode==='simple'){
        proteinLayer.innerHTML=
          buildSimpleProtein();
      }else if(mode==='facilitated'){
        proteinLayer.innerHTML=
          buildFacilitatedProtein(
            proteinLevel
          );
      }else{
        proteinLayer.innerHTML=
          buildActiveProtein(
            proteinLevel
          );
      }

      flowLayer.innerHTML=buildFlow(
        direction,
        rate,
        mode
      );

      atpLayer.innerHTML=buildATP(
        atpLevel,
        mode
      );

      var gradient=Math.abs(
        external-internal
      );
      var explanation='';
      var limiting='';

      if(mode==='simple'){
        title.textContent=
          '自由扩散：直接穿过磷脂双分子层';
        summary.textContent=
          '物质顺浓度梯度进行净运输，不需要运输蛋白，也不直接消耗ATP';
        energyLabel.textContent='直接耗能：否';
        stageNote.textContent='顺浓度梯度';

        if(passiveDirection==='balanced'){
          explanation=
            '膜两侧浓度接近，分子仍可双向运动，但净运输接近零。';
        }else{
          explanation=
            '物质从相对高浓度一侧向低浓度一侧净移动。';
        }

        if(gradient<10){
          limiting=
            '当前浓度差较小，自由扩散速率较低。';
        }else{
          limiting=
            '浓度差越大，其他条件相同时自由扩散的净速率通常越高。';
        }
      }else if(mode==='facilitated'){
        title.textContent=
          '协助扩散：通过通道或载体蛋白';
        summary.textContent=
          '物质顺浓度梯度运输，需要具有选择性的运输蛋白，但不直接消耗ATP';
        energyLabel.textContent='直接耗能：否';
        stageNote.textContent='顺梯度 + 运输蛋白';

        if(passiveDirection==='balanced'){
          explanation=
            '膜两侧浓度接近，即使存在运输蛋白，净运输仍接近零。';
        }else{
          explanation=
            '物质通过通道或载体蛋白，从相对高浓度一侧向低浓度一侧运输。';
        }

        if(proteinLevel<15){
          limiting=
            '运输蛋白数量很少，是当前协助扩散的主要限制因素。';
        }else if(gradient<10){
          limiting=
            '运输蛋白较充足，但膜两侧浓度差较小，净运输速率仍然较低。';
        }else if(proteinLevel>75){
          limiting=
            '运输蛋白较多，继续增加蛋白数量产生的速率提升逐渐减小。';
        }else{
          limiting=
            '浓度差和运输蛋白数量共同影响协助扩散速率。';
        }
      }else{
        title.textContent=
          '主动运输：逆浓度梯度运输';
        summary.textContent=
          '依赖载体或泵蛋白，并消耗细胞代谢提供的能量';
        energyLabel.textContent='直接耗能：需要ATP';
        stageNote.textContent='逆梯度 + 蛋白 + 能量';

        if(external===internal){
          explanation=
            '膜两侧浓度相等时，主动运输仍可消耗能量建立新的浓度梯度。';
        }else{
          explanation=
            '本模式把物质从相对低浓度一侧运向高浓度一侧，用于突出逆浓度梯度。';
        }

        if(proteinLevel<15){
          limiting=
            '泵或载体蛋白数量很少，是当前主动运输的主要限制因素。';
        }else if(atpLevel<15){
          limiting=
            'ATP供应不足，主动运输速率显著降低。';
        }else if(gradient>75){
          limiting=
            '逆浓度梯度较陡，维持运输所需条件更高。';
        }else{
          limiting=
            '运输蛋白数量和ATP供应共同影响主动运输速率。';
        }
      }

      result.innerHTML=explanation
        +'<br>'+limiting
        +' 所有浓度和速率均为教学示意值，不代表真实细胞测量结果。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    externalInput.oninput=update;
    internalInput.oninput=update;
    proteinInput.oninput=update;
    atpInput.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
