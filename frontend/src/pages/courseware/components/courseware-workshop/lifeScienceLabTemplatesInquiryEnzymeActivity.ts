/**
 * lifeScienceLabTemplatesInquiryEnzymeActivity.ts
 *
 * 平面生命科学实验室：酶活性与温度、pH探究。
 *
 * 教学目标：
 * 1. 通过控制变量观察温度和pH对酶活性的影响；
 * 2. 理解酶通常具有适宜温度和适宜pH；
 * 3. 理解酶浓度和底物浓度也会影响反应速率；
 * 4. 区分实验自变量、因变量和控制变量；
 * 5. 通过趋势曲线解释“低温抑制、高温可能导致酶结构改变”等现象。
 *
 * 教学边界：
 * 1. 不同酶的适宜温度和适宜pH不同；
 * 2. 本模型以“适宜温度约37℃、适宜pH约7”的假想酶为例；
 * 3. 相对酶活性是教学示意值，不代表真实实验测量结果；
 * 4. 真实实验还会受到反应时间、缓冲液、离子浓度和检测方法等因素影响；
 * 5. 高温影响在真实情况下可能具有不可逆性，本模型只展示当前条件下的相对趋势。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式或图片；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用.bl-*公共类名，使课件嵌入时能够应用统一的底部课堂控制条布局；
 * 5. 支持同一课件页放置多个实例。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

/**
 * 安全读取数值参数。
 * 参数缺失、类型不符或不是有限数时，回退到模板默认值。
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
 * 这里保留左侧控制区的原始独立预览布局。
 * 当模板应用到课件时，lifeScienceLabUtils.ts中的公共覆盖层
 * 会把.bl-controls调整到底部课堂控制条。
 */
function enzymeActivityStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#FFF7ED);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:242px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:11px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#7C3AED}'
    + '#' + rootId + ' .bl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-button{height:32px;padding:0 5px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bl-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .bl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bl-card{padding:7px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .bl-card b{display:block;font-size:16px;color:#6D28D9}'
    + '#' + rootId + ' .bl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ea-bubble{animation:' + rootId + '-bubble var(--ea-speed,1.6s) ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-bubble{from{transform:translateY(4px);opacity:.48}to{transform:translateY(-7px);opacity:1}}'
    + '</style>'
}

/**
 * 避免在外层模板字符串源码中直接出现字面量</script>，
 * 防止HTML解析器过早结束脚本标签。
 */
const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INQUIRY_ENZYME_ACTIVITY:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-enzyme-activity',
    group: '🧪 实验探究',
    name: '酶活性与温度、pH',
    emoji: '🧪',
    desc: '控制温度、pH、酶浓度和底物浓度，观察相对酶活性及单一变量曲线',
    params: [
      {
        key: 'temperature',
        label: '反应温度/℃',
        type: 'number',
        min: 0,
        max: 80,
        step: 1,
        defaultValue: 37,
        hint: '本模型假想酶的适宜温度约为37℃',
      },
      {
        key: 'ph',
        label: '反应液pH',
        type: 'number',
        min: 1,
        max: 14,
        step: 0.1,
        defaultValue: 7,
        hint: '本模型假想酶的适宜pH约为7',
      },
      {
        key: 'enzymeConcentration',
        label: '酶浓度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 65,
      },
      {
        key: 'substrateConcentration',
        label: '底物浓度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 75,
      },
    ],

    buildHTML: (params, rootId) => {
      const temperature = num(params, 'temperature', 37)
      const ph = num(params, 'ph', 7)
      const enzymeConcentration = num(
        params,
        'enzymeConcentration',
        65,
      )
      const substrateConcentration = num(
        params,
        'substrateConcentration',
        75,
      )

      return `
<div id="${rootId}">
${enzymeActivityStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧪 酶活性条件探究</div>
    <div class="bl-note">教学示意模型：不同酶的适宜条件不同</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>反应温度</span>
          <span class="bl-value" data-temperature-value></span>
        </div>
        <input
          data-temperature
          type="range"
          min="0"
          max="80"
          step="1"
          value="${n(temperature)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>反应液pH</span>
          <span class="bl-value" data-ph-value></span>
        </div>
        <input
          data-ph
          type="range"
          min="1"
          max="14"
          step="0.1"
          value="${n(ph)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>酶浓度</span>
          <span class="bl-value" data-enzyme-value></span>
        </div>
        <input
          data-enzyme
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(enzymeConcentration)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>底物浓度</span>
          <span class="bl-value" data-substrate-value></span>
        </div>
        <input
          data-substrate
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(substrateConcentration)}"
        >
      </div>

      <div class="bl-subtitle">单一变量探究</div>

      <div class="bl-buttons">
        <button
          type="button"
          class="bl-button active"
          data-mode="temperature"
        >温度曲线</button>

        <button
          type="button"
          class="bl-button"
          data-mode="ph"
        >pH曲线</button>
      </div>

      <div class="bl-status">
        <div class="bl-card">
          <b data-activity-value></b>
          <span>相对酶活性</span>
        </div>

        <div class="bl-card">
          <b data-condition-state></b>
          <span>当前条件判断</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 680 414"
        aria-label="酶活性与温度和pH关系互动实验"
      >
        <defs>
          <linearGradient
            id="${rootId}-tube"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#F5F3FF"/>
            <stop offset="100%" stop-color="#DDD6FE"/>
          </linearGradient>

          <linearGradient
            id="${rootId}-curve-area"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop offset="0%" stop-color="#8B5CF6" stop-opacity=".38"/>
            <stop offset="100%" stop-color="#8B5CF6" stop-opacity=".03"/>
          </linearGradient>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#4C1D95"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="26"
          y="36"
          data-title
          font-size="25"
          font-weight="900"
          fill="#5B21B6"
        ></text>

        <text
          x="26"
          y="65"
          data-summary
          font-size="14"
          font-weight="800"
          fill="#475569"
        ></text>

        <!-- 左侧：反应试管和酶促反应微观示意 -->
        <g transform="translate(28 88)" filter="url(#${rootId}-shadow)">
          <path
            d="M40 0 H158 V24 L148 202
               C146 238 126 258 99 258
               C72 258 52 238 50 202
               L40 24 Z"
            fill="#FFFFFF"
            stroke="#7C3AED"
            stroke-width="5"
          />

          <path
            d="M54 98 H144 L136 201
               C134 226 121 240 99 240
               C77 240 64 226 62 201 Z"
            fill="url(#${rootId}-tube)"
            stroke="#A78BFA"
            stroke-width="3"
          />

          <text
            x="99"
            y="286"
            text-anchor="middle"
            font-size="14"
            font-weight="900"
            fill="#5B21B6"
          >反应体系</text>

          <g data-reaction-particles></g>

          <rect
            x="22"
            y="306"
            width="154"
            height="18"
            rx="9"
            fill="#E2E8F0"
          />

          <rect
            data-reaction-bar
            x="22"
            y="306"
            width="0"
            height="18"
            rx="9"
            fill="#8B5CF6"
          />

          <text
            x="99"
            y="347"
            text-anchor="middle"
            data-reaction-label
            font-size="13"
            font-weight="900"
            fill="#6D28D9"
          ></text>
        </g>

        <!-- 右侧：自变量与相对酶活性曲线 -->
        <g transform="translate(232 88)">
          <line
            x1="52"
            y1="242"
            x2="410"
            y2="242"
            stroke="#64748B"
            stroke-width="3"
          />

          <line
            x1="52"
            y1="242"
            x2="52"
            y2="20"
            stroke="#64748B"
            stroke-width="3"
          />

          <text
            x="414"
            y="247"
            data-x-axis-title
            font-size="12"
            font-weight="800"
            fill="#475569"
          ></text>

          <text
            x="7"
            y="31"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >相对活性</text>

          <g data-grid></g>

          <line
            data-optimum-line
            x1="220"
            y1="20"
            x2="220"
            y2="242"
            stroke="#F59E0B"
            stroke-width="3"
            stroke-dasharray="8 7"
          />

          <text
            data-optimum-label
            x="220"
            y="15"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#B45309"
          ></text>

          <path
            data-curve-area
            fill="url(#${rootId}-curve-area)"
          ></path>

          <path
            data-curve
            fill="none"
            stroke="#7C3AED"
            stroke-width="6"
            stroke-linecap="round"
            stroke-linejoin="round"
          ></path>

          <g data-curve-points></g>

          <circle
            data-current-point
            cx="0"
            cy="0"
            r="9"
            fill="#FFFFFF"
            stroke="#EF4444"
            stroke-width="5"
          />

          <text
            data-current-label
            x="0"
            y="0"
            text-anchor="middle"
            font-size="11"
            font-weight="900"
            fill="#B91C1C"
          ></text>
        </g>

        <g transform="translate(244 372)">
          <circle cx="7" cy="7" r="7" fill="#7C3AED"/>
          <text
            x="23"
            y="12"
            font-size="13"
            font-weight="800"
            fill="#475569"
          >单一变量曲线</text>
        </g>

        <g transform="translate(420 372)">
          <circle cx="7" cy="7" r="7" fill="#EF4444"/>
          <text
            x="23"
            y="12"
            font-size="13"
            font-weight="800"
            fill="#475569"
          >当前实验条件</text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var temperatureInput=root.querySelector(
      '[data-temperature]'
    );
    var phInput=root.querySelector('[data-ph]');
    var enzymeInput=root.querySelector('[data-enzyme]');
    var substrateInput=root.querySelector('[data-substrate]');

    var temperatureValue=root.querySelector(
      '[data-temperature-value]'
    );
    var phValue=root.querySelector('[data-ph-value]');
    var enzymeValue=root.querySelector('[data-enzyme-value]');
    var substrateValue=root.querySelector(
      '[data-substrate-value]'
    );

    var buttons=root.querySelectorAll('[data-mode]');
    var activityValue=root.querySelector(
      '[data-activity-value]'
    );
    var conditionState=root.querySelector(
      '[data-condition-state]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var reactionParticles=root.querySelector(
      '[data-reaction-particles]'
    );
    var reactionBar=root.querySelector(
      '[data-reaction-bar]'
    );
    var reactionLabel=root.querySelector(
      '[data-reaction-label]'
    );

    var grid=root.querySelector('[data-grid]');
    var xAxisTitle=root.querySelector(
      '[data-x-axis-title]'
    );
    var optimumLine=root.querySelector(
      '[data-optimum-line]'
    );
    var optimumLabel=root.querySelector(
      '[data-optimum-label]'
    );
    var curveArea=root.querySelector(
      '[data-curve-area]'
    );
    var curve=root.querySelector('[data-curve]');
    var curvePoints=root.querySelector(
      '[data-curve-points]'
    );
    var currentPoint=root.querySelector(
      '[data-current-point]'
    );
    var currentLabel=root.querySelector(
      '[data-current-label]'
    );

    var mode='temperature';

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    /**
     * 温度影响函数。
     *
     * 低于适宜温度时活性逐渐降低；
     * 高于适宜温度后下降更快，用于表达高温可能影响酶结构。
     */
    function temperatureFactor(value){
      var optimum=37;
      var width=value<=optimum?24:15;
      var distance=(value-optimum)/width;

      return Math.exp(-distance*distance);
    }

    /**
     * pH影响函数。
     *
     * 本模型假设适宜pH为7。
     */
    function phFactor(value){
      var distance=(value-7)/2.55;

      return Math.exp(-distance*distance);
    }

    /**
     * 酶浓度影响。
     *
     * 在其他条件不变且底物相对充足时，
     * 酶浓度提高通常会提高反应速率。
     */
    function enzymeFactor(value){
      return value/(value+30);
    }

    /**
     * 底物浓度影响。
     *
     * 使用饱和型关系表示底物增加到一定程度后，
     * 继续增加产生的速率提升逐渐减小。
     */
    function substrateFactor(value){
      return value/(value+32);
    }

    function totalActivity(
      temperature,
      ph,
      enzymeLevel,
      substrateLevel
    ){
      return 100
        *temperatureFactor(temperature)
        *phFactor(ph)
        *enzymeFactor(enzymeLevel)
        *substrateFactor(substrateLevel);
    }

    function buildReactionParticles(activity){
      var html='';
      var enzymeCount=Math.max(
        2,
        Math.floor(Number(enzymeInput.value)/12)
      );
      var substrateCount=Math.max(
        3,
        Math.floor(Number(substrateInput.value)/10)
      );
      var productCount=Math.floor(activity/8);

      for(var i=0;i<enzymeCount;i++){
        var ex=66+(i%4)*23;
        var ey=126+Math.floor(i/4)*39;

        html+='<path d="M'+(ex-9)+' '+ey
          +' C'+(ex-4)+' '+(ey-10)+' '
          +(ex+4)+' '+(ey-10)+' '
          +(ex+9)+' '+ey
          +' C'+(ex+4)+' '+(ey+10)+' '
          +(ex-4)+' '+(ey+10)+' '
          +(ex-9)+' '+ey+'Z"'
          +' fill="#8B5CF6" stroke="#5B21B6"'
          +' stroke-width="2" opacity=".86"/>';
      }

      for(var j=0;j<substrateCount;j++){
        var sx=61+(j%5)*19;
        var sy=168+Math.floor(j/5)*24;

        html+='<rect x="'+(sx-6)+'" y="'+(sy-6)
          +'" width="12" height="12" rx="3"'
          +' fill="#F59E0B" stroke="#B45309"'
          +' stroke-width="1.5" opacity=".82"/>';
      }

      for(var k=0;k<productCount;k++){
        var px=64+(k%5)*18;
        var py=91-Math.floor(k/5)*14-(k%2)*5;

        html+='<circle class="ea-bubble" cx="'+px
          +'" cy="'+py+'" r="'+(4+k%3)
          +'" fill="#34D399" stroke="#047857"'
          +' stroke-width="1.5"/>';

        if(k%3===0){
          html+='<text x="'+(px+7)+'" y="'+(py+4)
            +'" font-size="8" font-weight="900"'
            +' fill="#047857">产物</text>';
        }
      }

      return html;
    }

    function buildGrid(
      left,
      right,
      top,
      bottom,
      xMin,
      xMax,
      modeName
    ){
      var html='';
      var width=right-left;
      var height=bottom-top;

      for(var yIndex=0;yIndex<=4;yIndex++){
        var y=bottom-height*yIndex/4;
        var yLabel=yIndex*25;

        html+='<line x1="'+left+'" y1="'+y
          +'" x2="'+right+'" y2="'+y
          +'" stroke="#E2E8F0" stroke-width="1.3"/>';

        html+='<text x="'+(left-8)+'" y="'+(y+4)
          +'" text-anchor="end" font-size="10"'
          +' font-weight="700" fill="#64748B">'
          +yLabel+'</text>';
      }

      var tickCount=modeName==='temperature'?8:7;

      for(var xIndex=0;xIndex<=tickCount;xIndex++){
        var x=left+width*xIndex/tickCount;
        var raw=xMin+(xMax-xMin)*xIndex/tickCount;
        var label=modeName==='temperature'
          ?raw.toFixed(0)
          :raw.toFixed(1);

        html+='<line x1="'+x+'" y1="'+bottom
          +'" x2="'+x+'" y2="'+top
          +'" stroke="#F1F5F9" stroke-width="1"/>';

        html+='<text x="'+x+'" y="'+(bottom+18)
          +'" text-anchor="middle" font-size="10"'
          +' font-weight="700" fill="#64748B">'
          +label+'</text>';
      }

      return html;
    }

    function drawCurve(
      currentTemperature,
      currentPh,
      enzymeLevel,
      substrateLevel
    ){
      var left=52;
      var right=410;
      var top=20;
      var bottom=242;
      var width=right-left;
      var height=bottom-top;

      var xMin=mode==='temperature'?0:1;
      var xMax=mode==='temperature'?80:14;
      var optimum=mode==='temperature'?37:7;
      var samples=60;
      var path='';
      var pointHTML='';

      function x(value){
        return left+width*(value-xMin)/(xMax-xMin);
      }

      function y(value){
        return bottom-height*clamp(value/100,0,1);
      }

      for(var i=0;i<=samples;i++){
        var variable=xMin+(xMax-xMin)*i/samples;
        var activity;

        if(mode==='temperature'){
          activity=totalActivity(
            variable,
            currentPh,
            enzymeLevel,
            substrateLevel
          );
        }else{
          activity=totalActivity(
            currentTemperature,
            variable,
            enzymeLevel,
            substrateLevel
          );
        }

        var px=x(variable);
        var py=y(activity);

        path+=(i===0?'M':' L')+px+' '+py;

        if(i%10===0 || i===samples){
          pointHTML+='<circle cx="'+px+'" cy="'+py
            +'" r="3.6" fill="#FFFFFF"'
            +' stroke="#7C3AED" stroke-width="2.4"/>';
        }
      }

      curve.setAttribute('d',path);
      curvePoints.innerHTML=pointHTML;

      curveArea.setAttribute(
        'd',
        path+' L'+right+' '+bottom
        +' L'+left+' '+bottom+' Z'
      );

      var currentVariable=mode==='temperature'
        ?currentTemperature
        :currentPh;

      var currentActivity=totalActivity(
        currentTemperature,
        currentPh,
        enzymeLevel,
        substrateLevel
      );

      var currentX=x(currentVariable);
      var currentY=y(currentActivity);

      currentPoint.setAttribute('cx',String(currentX));
      currentPoint.setAttribute('cy',String(currentY));

      currentLabel.setAttribute('x',String(currentX));
      currentLabel.setAttribute(
        'y',
        String(Math.max(top+12,currentY-15))
      );
      currentLabel.textContent=currentActivity.toFixed(0);

      var optimumX=x(optimum);

      optimumLine.setAttribute('x1',String(optimumX));
      optimumLine.setAttribute('x2',String(optimumX));

      optimumLabel.setAttribute('x',String(optimumX));
      optimumLabel.textContent=mode==='temperature'
        ?'适宜温度约37℃'
        :'适宜pH约7';

      grid.innerHTML=buildGrid(
        left,
        right,
        top,
        bottom,
        xMin,
        xMax,
        mode
      );

      if(mode==='temperature'){
        title.textContent='温度对酶活性的影响';
        summary.textContent=
          '保持pH、酶浓度和底物浓度不变，只改变温度';
        xAxisTitle.textContent='温度/℃';
      }else{
        title.textContent='pH对酶活性的影响';
        summary.textContent=
          '保持温度、酶浓度和底物浓度不变，只改变pH';
        xAxisTitle.textContent='pH';
      }
    }

    function update(){
      var temperature=Number(temperatureInput.value);
      var ph=Number(phInput.value);
      var enzymeLevel=Number(enzymeInput.value);
      var substrateLevel=Number(substrateInput.value);

      var tempEffect=temperatureFactor(temperature);
      var currentPhEffect=phFactor(ph);
      var activity=totalActivity(
        temperature,
        ph,
        enzymeLevel,
        substrateLevel
      );

      temperatureValue.textContent=
        temperature.toFixed(0)+'℃';
      phValue.textContent=ph.toFixed(1);
      enzymeValue.textContent=enzymeLevel.toFixed(0)+'%';
      substrateValue.textContent=
        substrateLevel.toFixed(0)+'%';

      activityValue.textContent=activity.toFixed(0);

      root.style.setProperty(
        '--ea-speed',
        clamp(2.5-activity/60,.45,2.4).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-mode')===mode
        );
      }

      var state='条件较适宜';
      var explanation=
        '当前温度和pH接近该假想酶的适宜范围。';

      if(temperature>=55){
        state='高温抑制';
        explanation=
          '温度明显偏高，酶的空间结构可能受到影响，活性显著降低。';
      }else if(temperature<=12){
        state='低温抑制';
        explanation=
          '温度较低，分子运动减慢，酶促反应速率下降。';
      }else if(ph<=4 || ph>=11){
        state='pH偏离';
        explanation=
          'pH明显偏离适宜范围，可能影响酶的结构和活性部位状态。';
      }else if(tempEffect<.55){
        state='温度受限';
        explanation=
          '温度偏离适宜范围，是当前酶活性的主要限制因素。';
      }else if(currentPhEffect<.55){
        state='pH受限';
        explanation=
          'pH偏离适宜范围，是当前酶活性的主要限制因素。';
      }else if(enzymeLevel<30){
        state='酶量较少';
        explanation=
          '酶浓度较低，可参与催化的酶分子相对较少。';
      }else if(substrateLevel<30){
        state='底物较少';
        explanation=
          '底物浓度较低，酶与底物有效结合的机会相对较少。';
      }else if(activity<35){
        state='多因素受限';
        explanation=
          '多个实验条件共同限制了当前反应速率。';
      }

      conditionState.textContent=state;

      reactionBar.setAttribute(
        'width',
        String(154*clamp(activity/100,0,1))
      );

      reactionBar.setAttribute(
        'fill',
        activity>=60
          ?'#10B981'
          :activity>=30
            ?'#F59E0B'
            :'#EF4444'
      );

      reactionLabel.textContent=
        '产物形成速率 '+activity.toFixed(0);

      reactionParticles.innerHTML=
        buildReactionParticles(activity);

      drawCurve(
        temperature,
        ph,
        enzymeLevel,
        substrateLevel
      );

      var variableSentence=mode==='temperature'
        ?'探究温度时，应保持pH、酶浓度、底物浓度和反应时间等条件一致。'
        :'探究pH时，应保持温度、酶浓度、底物浓度和反应时间等条件一致。';

      result.innerHTML=explanation
        +'<br>'+variableSentence
        +' 相对酶活性为教学示意值，不代表真实实验测量结果。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    temperatureInput.oninput=update;
    phInput.oninput=update;
    enzymeInput.oninput=update;
    substrateInput.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
