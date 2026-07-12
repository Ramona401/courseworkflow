/**
 * geographyLabTemplatesSustainabilityEnergyLowCarbonCity.ts
 *
 * 地理第42批B3：
 *   能源转型、低碳城市与可持续发展。
 *
 * 教学目标：
 * - 理解能源结构、能源需求和电力系统灵活性之间的关系；
 * - 比较化石能源、可再生能源、储能和电网调节的不同作用；
 * - 理解建筑节能、公共交通、紧凑布局和绿色空间对城市碳排放的影响；
 * - 理解循环经济通过减量、再利用和资源循环降低资源环境压力；
 * - 比较单项措施和综合转型方案的协同效应与实施约束；
 * - 认识低碳转型需要兼顾安全、成本、公平、就业和基本公共服务；
 * - 建立能源、交通、建筑、产业和生态协同推进的可持续发展意识。
 *
 * 教学边界：
 * - 能源占比、碳排放、空气质量、成本、公平和韧性均为相对教学指数；
 * - 城市、能源设施、道路、社区和产业系统不对应任何真实地区；
 * - 不考虑真实电力调度、能源价格、技术寿命和复杂产业链约束；
 * - 不用于真实能源规划、城市设计、碳核算、投资或政策决策。
 */

import type {
  GeographyLabParamValue,
  GeographyLabTemplate,
} from './geographyLabUtils'

const SCRIPT_END = '</' + 'script>'

function numberValue(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: number,
): number {
  const value = Number(params[key])
  return Number.isFinite(value) ? value : fallback
}

function booleanValue(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]
  return typeof value === 'boolean' ? value : fallback
}

function stringValue(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: string,
): string {
  const value = params[key]
  return typeof value === 'string' ? value : fallback
}

function buildEnergyLowCarbonCityHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'fossil-intensive',
    'efficiency-first',
    'renewable-expansion',
    'transit-oriented',
    'integrated-transition',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'integrated-transition',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'integrated-transition'

  const fossilShare = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'fossilShare', 5),
    ),
  )

  const renewableShare = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'renewableShare', 7),
    ),
  )

  const gridFlexibility = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'gridFlexibility', 6),
    ),
  )

  const buildingEfficiency = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'buildingEfficiency', 7),
    ),
  )

  const publicTransit = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'publicTransit', 7),
    ),
  )

  const circularEconomy = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'circularEconomy', 6),
    ),
  )

  const greenSpace = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'greenSpace', 6),
    ),
  )

  const equityGovernance = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'equityGovernance', 7),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-energy-city-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #A7F3D0;
      border-radius:18px;
      background:#FFFFFF;
      color:#022C22;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(5,150,105,.12);
    }

    #${rootId} *{
      box-sizing:border-box;
    }

    #${rootId} .gl-head{
      height:56px;
      padding:0 18px;
      display:flex;
      align-items:center;
      gap:12px;
      border-bottom:1px solid #D1FAE5;
      background:linear-gradient(
        135deg,
        #ECFDF5,
        #F0FDF4 48%,
        #EFF6FF
      );
    }

    #${rootId} .gl-title{
      color:#065F46;
      font-size:16px;
      font-weight:880;
    }

    #${rootId} .gl-subtitle{
      margin-top:2px;
      color:#64748B;
      font-size:11px;
    }

    #${rootId} .gl-note{
      margin-left:auto;
      padding:5px 10px;
      border:1px solid #A7F3D0;
      border-radius:999px;
      background:#FFFFFF;
      color:#047857;
      font-size:11px;
      font-weight:760;
      white-space:nowrap;
    }

    #${rootId} .gl-body{
      height:calc(100% - 56px);
      display:grid;
      grid-template-columns:292px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      min-height:0;
      padding:13px;
      overflow:auto;
      border-right:1px solid #D1FAE5;
      background:linear-gradient(
        180deg,
        #ECFDF5,
        #F0FDF4 55%,
        #EFF6FF
      );
    }

    #${rootId} .gl-stage{
      min-width:0;
      min-height:0;
      display:grid;
      grid-template-rows:46px minmax(0,1fr);
      padding:8px;
      background:radial-gradient(
        circle at 48% 20%,
        #FFFFFF 0%,
        #F8FAFC 60%,
        #D1FAE5 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#065F46;
      font-size:11.5px;
      font-weight:850;
    }

    #${rootId} .gl-scenario-grid{
      display:grid;
      grid-template-columns:1fr 1fr;
      gap:6px;
      margin-bottom:12px;
    }

    #${rootId} .gl-scenario-grid button:last-child{
      grid-column:1/-1;
    }

    #${rootId} .gl-row{
      margin-bottom:9px;
    }

    #${rootId} .gl-label-line{
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:8px;
      margin-bottom:5px;
    }

    #${rootId} .gl-label{
      color:#334155;
      font-size:11.2px;
      font-weight:730;
    }

    #${rootId} .gl-value{
      min-width:44px;
      padding:3px 7px;
      border-radius:999px;
      background:#D1FAE5;
      color:#047857;
      font-size:10.8px;
      font-weight:850;
      text-align:center;
    }

    #${rootId} input[type=range]{
      width:100%;
      height:6px;
      margin:0;
      appearance:none;
      border-radius:999px;
      outline:none;
      background:linear-gradient(
        90deg,
        #FBBF24,
        #34D399,
        #38BDF8
      );
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:16px;
      height:16px;
      appearance:none;
      border:2px solid #FFFFFF;
      border-radius:50%;
      background:linear-gradient(
        135deg,
        #059669,
        #0284C7
      );
      box-shadow:0 1px 5px rgba(5,150,105,.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #A7F3D0;
      border-radius:9px;
      background:#FFFFFF;
      color:#047857;
      font-size:10.5px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#047857;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #059669,
        #0F766E 55%,
        #0284C7
      );
      box-shadow:0 5px 13px rgba(5,150,105,.22);
    }

    #${rootId} .gl-action-grid{
      display:grid;
      grid-template-columns:1fr 1fr;
      gap:6px;
      margin:10px 0;
    }

    #${rootId} .gl-result{
      margin-top:8px;
      padding:10px;
      border:1px solid #A7F3D0;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #ECFDF5,
        #EFF6FF
      );
      color:#334155;
      font-size:11px;
      font-weight:620;
      line-height:1.5;
    }

    #${rootId} .gl-view-toolbar{
      display:grid;
      grid-template-columns:repeat(4,minmax(0,1fr));
      gap:7px;
      align-items:center;
      padding:0 3px 7px;
      border-bottom:1px solid #E2E8F0;
    }

    #${rootId} .gl-view-toolbar button{
      min-height:32px;
      font-size:11px;
    }

    #${rootId} .gl-canvas-wrap{
      min-width:0;
      min-height:0;
      overflow:hidden;
      border:1px solid #D1FAE5;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-energy-city-canvas{
      width:100%;
      height:100%;
      display:block;
    }

    @media(max-width:900px){
      #${rootId} .gl-body{
        grid-template-columns:246px minmax(0,1fr);
      }

      #${rootId} .gl-note{
        display:none;
      }
    }
  </style>

  <div class="gl-head">
    <div style="font-size:24px;">🏙️</div>

    <div>
      <div class="gl-title">
        能源转型、低碳城市与可持续发展
      </div>

      <div class="gl-subtitle">
        比较能源结构、电网调节、建筑效率、公共交通、循环经济和公平治理
      </div>
    </div>

    <div class="gl-note">
      相对教学指数 · 不用于真实能源城市规划
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        能源与城市转型情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="fossil-intensive">
          化石能源依赖
        </button>

        <button type="button" data-scenario="efficiency-first">
          效率优先
        </button>

        <button type="button" data-scenario="renewable-expansion">
          可再生能源扩张
        </button>

        <button type="button" data-scenario="transit-oriented">
          公交导向城市
        </button>

        <button type="button" data-scenario="integrated-transition">
          综合低碳转型
        </button>
      </div>

      <div class="gl-section-title">
        能源、城市与治理参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">化石能源依赖</span>
          <span class="gl-value" data-role="fossil-value">5</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${fossilShare}"
          data-role="fossil"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">可再生能源比例</span>
          <span class="gl-value" data-role="renewable-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${renewableShare}"
          data-role="renewable"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">电网与储能灵活性</span>
          <span class="gl-value" data-role="grid-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${gridFlexibility}"
          data-role="grid"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">建筑节能水平</span>
          <span class="gl-value" data-role="building-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${buildingEfficiency}"
          data-role="building"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">公共交通与慢行</span>
          <span class="gl-value" data-role="transit-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${publicTransit}"
          data-role="transit"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">循环经济水平</span>
          <span class="gl-value" data-role="circular-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${circularEconomy}"
          data-role="circular"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">城市绿色空间</span>
          <span class="gl-value" data-role="green-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${greenSpace}"
          data-role="green"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">公平治理与转型支持</span>
          <span class="gl-value" data-role="equity-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${equityGovernance}"
          data-role="equity"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          系统与措施标注
        </button>

        <button
          type="button"
          data-role="auto-toggle"
          data-active="false"
        >
          自动演示
        </button>

        <button type="button" data-role="reset">
          恢复初始
        </button>

        <button type="button" data-role="next">
          下一情境
        </button>
      </div>

      <div class="gl-result" data-role="result">
        低碳转型不仅是增加可再生能源，还需要降低需求、提高电网灵活性并同步改变建筑、交通和资源利用方式。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="energy">
          能源系统
        </button>

        <button type="button" data-view="city">
          低碳城市
        </button>

        <button type="button" data-view="tradeoff">
          效益与约束
        </button>

        <button type="button" data-view="pathway">
          转型路径
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-energy-city-canvas"
          width="1000"
          height="570"
          data-role="canvas"
          aria-label="能源转型低碳城市与可持续发展教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var fossilInput =
        root.querySelector('[data-role="fossil"]');

      var renewableInput =
        root.querySelector('[data-role="renewable"]');

      var gridInput =
        root.querySelector('[data-role="grid"]');

      var buildingInput =
        root.querySelector('[data-role="building"]');

      var transitInput =
        root.querySelector('[data-role="transit"]');

      var circularInput =
        root.querySelector('[data-role="circular"]');

      var greenInput =
        root.querySelector('[data-role="green"]');

      var equityInput =
        root.querySelector('[data-role="equity"]');

      var fossilValue =
        root.querySelector('[data-role="fossil-value"]');

      var renewableValue =
        root.querySelector('[data-role="renewable-value"]');

      var gridValue =
        root.querySelector('[data-role="grid-value"]');

      var buildingValue =
        root.querySelector('[data-role="building-value"]');

      var transitValue =
        root.querySelector('[data-role="transit-value"]');

      var circularValue =
        root.querySelector('[data-role="circular-value"]');

      var greenValue =
        root.querySelector('[data-role="green-value"]');

      var equityValue =
        root.querySelector('[data-role="equity-value"]');

      var scenarioButtons =
        root.querySelectorAll('[data-scenario]');

      var viewButtons =
        root.querySelectorAll('[data-view]');

      var labelToggle =
        root.querySelector('[data-role="label-toggle"]');

      var autoToggle =
        root.querySelector('[data-role="auto-toggle"]');

      var resetButton =
        root.querySelector('[data-role="reset"]');

      var nextButton =
        root.querySelector('[data-role="next"]');

      var result =
        root.querySelector('[data-role="result"]');

      var canvas =
        root.querySelector('[data-role="canvas"]');

      if(
        !fossilInput ||
        !renewableInput ||
        !gridInput ||
        !buildingInput ||
        !transitInput ||
        !circularInput ||
        !greenInput ||
        !equityInput ||
        !fossilValue ||
        !renewableValue ||
        !gridValue ||
        !buildingValue ||
        !transitValue ||
        !circularValue ||
        !greenValue ||
        !equityValue ||
        !scenarioButtons.length ||
        !viewButtons.length ||
        !labelToggle ||
        !autoToggle ||
        !resetButton ||
        !nextButton ||
        !result ||
        !canvas
      ){
        return;
      }

      var context=canvas.getContext('2d');
      if(!context)return;

      var width=canvas.width;
      var height=canvas.height;

      var scenarios=[
        {
          key:'fossil-intensive',
          name:'化石能源依赖',
          fossil:10,
          renewable:2,
          grid:4,
          building:3,
          transit:3,
          circular:3,
          green:4,
          equity:4,
          view:'energy',
          color:'#DC2626'
        },
        {
          key:'efficiency-first',
          name:'效率优先',
          fossil:7,
          renewable:4,
          grid:5,
          building:10,
          transit:6,
          circular:7,
          green:6,
          equity:7,
          view:'tradeoff',
          color:'#7C3AED'
        },
        {
          key:'renewable-expansion',
          name:'可再生能源扩张',
          fossil:3,
          renewable:10,
          grid:8,
          building:6,
          transit:5,
          circular:5,
          green:6,
          equity:6,
          view:'energy',
          color:'#0284C7'
        },
        {
          key:'transit-oriented',
          name:'公交导向城市',
          fossil:5,
          renewable:6,
          grid:6,
          building:7,
          transit:10,
          circular:7,
          green:8,
          equity:8,
          view:'city',
          color:'#D97706'
        },
        {
          key:'integrated-transition',
          name:'综合低碳转型',
          fossil:2,
          renewable:9,
          grid:9,
          building:9,
          transit:9,
          circular:9,
          green:8,
          equity:9,
          view:'pathway',
          color:'#16A34A'
        }
      ];

      var initial={
        scenario:'${scenario}',
        fossil:${fossilShare},
        renewable:${renewableShare},
        grid:${gridFlexibility},
        building:${buildingEfficiency},
        transit:${publicTransit},
        circular:${circularEconomy},
        green:${greenSpace},
        equity:${equityGovernance},
        showLabels:${showLabels}
      };

      var state={
        scenario:initial.scenario,
        view:'energy',
        showLabels:initial.showLabels,
        auto:false,
        startedAt:0,
        phase:0,
        scenarioIndex:0,
        raf:0
      };

      function clamp(value,min,max){
        return Math.max(
          min,
          Math.min(max,value)
        );
      }

      function lerp(a,b,t){
        return a+(b-a)*t;
      }

      function ease(t){
        var p=clamp(t,0,1);

        return p<.5
          ? 2*p*p
          : 1-Math.pow(-2*p+2,2)/2;
      }

      function roundRect(x,y,w,h,r){
        var q=Math.min(r,w/2,h/2);

        context.beginPath();
        context.moveTo(x+q,y);
        context.lineTo(x+w-q,y);
        context.quadraticCurveTo(x+w,y,x+w,y+q);
        context.lineTo(x+w,y+h-q);
        context.quadraticCurveTo(x+w,y+h,x+w-q,y+h);
        context.lineTo(x+q,y+h);
        context.quadraticCurveTo(x,y+h,x,y+h-q);
        context.lineTo(x,y+q);
        context.quadraticCurveTo(x,y,x+q,y);
        context.closePath();
      }

      function box(x,y,w,h,r,fill,stroke){
        roundRect(x,y,w,h,r);

        if(fill){
          context.fillStyle=fill;
          context.fill();
        }

        if(stroke){
          context.strokeStyle=stroke;
          context.lineWidth=1.2;
          context.stroke();
        }
      }

      function text(
        value,
        x,
        y,
        size,
        color,
        weight,
        align
      ){
        context.save();

        context.font =
          (weight || 600)+
          ' '+
          size+
          'px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif';

        context.fillStyle=color || '#334155';
        context.textAlign=align || 'left';
        context.textBaseline='middle';

        context.fillText(
          String(value),
          x,
          y
        );

        context.restore();
      }

      function line(
        x1,
        y1,
        x2,
        y2,
        color,
        lineWidth,
        dash
      ){
        context.save();
        context.strokeStyle=color;
        context.lineWidth=lineWidth || 1.5;
        context.setLineDash(dash || []);
        context.beginPath();
        context.moveTo(x1,y1);
        context.lineTo(x2,y2);
        context.stroke();
        context.restore();
      }

      function circle(
        x,
        y,
        r,
        fill,
        stroke,
        lineWidth
      ){
        context.beginPath();
        context.arc(x,y,r,0,Math.PI*2);

        if(fill){
          context.fillStyle=fill;
          context.fill();
        }

        if(stroke){
          context.strokeStyle=stroke;
          context.lineWidth=lineWidth || 2;
          context.stroke();
        }
      }

      function arrow(
        x1,
        y1,
        x2,
        y2,
        color,
        lineWidth
      ){
        var angle=Math.atan2(y2-y1,x2-x1);
        var head=11;

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=lineWidth || 3;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(x1,y1);
        context.lineTo(x2,y2);
        context.stroke();

        context.beginPath();
        context.moveTo(x2,y2);

        context.lineTo(
          x2-head*Math.cos(angle-Math.PI/6),
          y2-head*Math.sin(angle-Math.PI/6)
        );

        context.lineTo(
          x2-head*Math.cos(angle+Math.PI/6),
          y2-head*Math.sin(angle+Math.PI/6)
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function curvedArrow(
        sx,
        sy,
        cx,
        cy,
        ex,
        ey,
        color,
        lineWidth
      ){
        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=lineWidth || 3;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(sx,sy);
        context.quadraticCurveTo(cx,cy,ex,ey);
        context.stroke();

        var angle=Math.atan2(ey-cy,ex-cx);
        var head=10;

        context.beginPath();
        context.moveTo(ex,ey);

        context.lineTo(
          ex-head*Math.cos(angle-Math.PI/6),
          ey-head*Math.sin(angle-Math.PI/6)
        );

        context.lineTo(
          ex-head*Math.cos(angle+Math.PI/6),
          ey-head*Math.sin(angle+Math.PI/6)
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function scenarioByKey(key){
        var found=scenarios[4];

        scenarios.forEach(
          function(item){
            if(item.key===key){
              found=item;
            }
          }
        );

        return found;
      }

      function values(){
        return {
          fossil:clamp(
            Number(fossilInput.value) || 0,
            0,
            10
          ),
          renewable:clamp(
            Number(renewableInput.value) || 0,
            0,
            10
          ),
          grid:clamp(
            Number(gridInput.value) || 0,
            0,
            10
          ),
          building:clamp(
            Number(buildingInput.value) || 0,
            0,
            10
          ),
          transit:clamp(
            Number(transitInput.value) || 0,
            0,
            10
          ),
          circular:clamp(
            Number(circularInput.value) || 0,
            0,
            10
          ),
          green:clamp(
            Number(greenInput.value) || 0,
            0,
            10
          ),
          equity:clamp(
            Number(equityInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        fossilInput.value=
          String(Math.round(value.fossil));

        renewableInput.value=
          String(Math.round(value.renewable));

        gridInput.value=
          String(Math.round(value.grid));

        buildingInput.value=
          String(Math.round(value.building));

        transitInput.value=
          String(Math.round(value.transit));

        circularInput.value=
          String(Math.round(value.circular));

        greenInput.value=
          String(Math.round(value.green));

        equityInput.value=
          String(Math.round(value.equity));
      }

      function derive(value){
        var energyDemand=
          clamp(
            10-
            value.building*.34-
            value.circular*.14-
            value.transit*.08+
            value.fossil*.10,
            1,
            10
          );

        var cleanPower=
          clamp(
            value.renewable*.66+
            value.grid*.22+
            (
              10-value.fossil
            )*.12,
            0,
            10
          );

        var gridStability=
          clamp(
            value.grid*.52+
            value.renewable*.18+
            (
              10-energyDemand
            )*.14+
            value.equity*.16,
            0,
            10
          );

        var emissionPressure=
          clamp(
            value.fossil*.48+
            energyDemand*.24+
            (
              10-value.transit
            )*.13+
            (
              10-value.circular
            )*.10-
            value.renewable*.20-
            value.green*.05,
            0,
            10
          );

        var airQuality=
          clamp(
            Math.round(
              10-
              emissionPressure*.72+
              value.green*.20+
              value.transit*.10
            ),
            0,
            10
          );

        var mobilityEfficiency=
          clamp(
            Math.round(
              value.transit*.58+
              value.building*.08+
              value.green*.12+
              value.equity*.22
            ),
            0,
            10
          );

        var resourceEfficiency=
          clamp(
            Math.round(
              value.circular*.52+
              value.building*.24+
              value.equity*.14+
              value.renewable*.10
            ),
            0,
            10
          );

        var transitionCost=
          clamp(
            Math.round(
              value.renewable*.22+
              value.grid*.20+
              value.building*.15+
              value.transit*.17+
              value.circular*.10-
              value.equity*.12+
              value.fossil*.06
            ),
            0,
            10
          );

        var equityScore=
          clamp(
            Math.round(
              value.equity*.52+
              value.transit*.22+
              value.building*.10+
              (
                10-transitionCost
              )*.16
            ),
            0,
            10
          );

        var resilience=
          clamp(
            Math.round(
              value.grid*.30+
              value.renewable*.22+
              value.green*.16+
              value.circular*.12+
              value.equity*.20
            ),
            0,
            10
          );

        var sustainability=
          clamp(
            Math.round(
              cleanPower*.22+
              airQuality*.15+
              mobilityEfficiency*.14+
              resourceEfficiency*.16+
              equityScore*.15+
              resilience*.18
            ),
            0,
            10
          );

        var residualCarbon=
          clamp(
            Math.round(
              emissionPressure-
              value.building*.12-
              value.transit*.10-
              value.circular*.10-
              value.equity*.05
            ),
            0,
            10
          );

        return {
          energyDemand:energyDemand,
          cleanPower:cleanPower,
          gridStability:gridStability,
          emissionPressure:emissionPressure,
          airQuality:airQuality,
          mobilityEfficiency:mobilityEfficiency,
          resourceEfficiency:resourceEfficiency,
          transitionCost:transitionCost,
          equityScore:equityScore,
          resilience:resilience,
          sustainability:sustainability,
          residualCarbon:residualCarbon
        };
      }

      function background(titleValue,subtitle){
        var gradient=
          context.createLinearGradient(
            0,
            0,
            width,
            height
          );

        gradient.addColorStop(0,'#FFFFFF');
        gradient.addColorStop(.56,'#ECFDF5');
        gradient.addColorStop(1,'#DBEAFE');

        context.fillStyle=gradient;
        context.fillRect(0,0,width,height);

        text(
          titleValue,
          28,
          31,
          18,
          '#065F46',
          880,
          'left'
        );

        text(
          subtitle,
          28,
          55,
          11.5,
          '#64748B',
          620,
          'left'
        );
      }

      function card(
        x,
        y,
        w,
        label,
        value,
        color,
        desc
      ){
        box(
          x,
          y,
          w,
          72,
          12,
          'rgba(255,255,255,.95)',
          '#D1FAE5'
        );

        text(
          label,
          x+14,
          y+17,
          10.5,
          '#64748B',
          720,
          'left'
        );

        text(
          value,
          x+14,
          y+40,
          20,
          color,
          880,
          'left'
        );

        text(
          desc,
          x+14,
          y+59,
          9.4,
          '#64748B',
          600,
          'left'
        );
      }

      function sourceNode(
        x,
        y,
        r,
        label,
        icon,
        color,
        value
      ){
        circle(
          x,
          y,
          r,
          '#FFFFFF',
          color,
          4
        );

        context.save();
        context.globalAlpha=.12+value*.06;
        circle(
          x,
          y,
          r-4,
          color,
          null,
          0
        );
        context.restore();

        text(
          icon,
          x,
          y-8,
          17,
          color,
          850,
          'center'
        );

        text(
          label,
          x,
          y+17,
          9.5,
          color,
          850,
          'center'
        );

        text(
          value.toFixed(1),
          x,
          y+r+16,
          10,
          color,
          850,
          'center'
        );
      }

      function energyView(
        item,
        value,
        derived
      ){
        background(
          '能源结构、需求侧效率与电力系统灵活性',
          '可再生能源扩张需要与储能、电网调节和需求侧管理同步推进。'
        );

        card(
          28,
          78,
          210,
          '能源需求压力',
          derived.energyDemand.toFixed(1),
          '#D97706',
          '建筑、交通和资源效率综合'
        );

        card(
          250,
          78,
          210,
          '清洁电力指数',
          derived.cleanPower.toFixed(1),
          '#16A34A',
          '可再生能源与低碳电力'
        );

        card(
          472,
          78,
          210,
          '电力系统稳定',
          derived.gridStability.toFixed(1),
          '#2563EB',
          '电网、储能和需求响应'
        );

        card(
          694,
          78,
          258,
          '剩余碳压力',
          derived.residualCarbon,
          item.color,
          '能源与城市措施共同作用'
        );

        var centerX=500;
        var centerY=354;

        sourceNode(
          128,
          250,
          57,
          '化石能源',
          '煤',
          '#475569',
          value.fossil
        );

        sourceNode(
          128,
          443,
          57,
          '风能',
          '风',
          '#0284C7',
          value.renewable*.55
        );

        sourceNode(
          326,
          443,
          57,
          '太阳能',
          '光',
          '#D97706',
          value.renewable*.70
        );

        sourceNode(
          326,
          250,
          57,
          '其他低碳电源',
          '低',
          '#16A34A',
          value.renewable*.40+
          value.grid*.18
        );

        box(
          421,
          270,
          158,
          168,
          18,
          '#FFFFFF',
          '#60A5FA'
        );

        text(
          '综合电力系统',
          centerX,
          297,
          13,
          '#1D4ED8',
          880,
          'center'
        );

        circle(
          centerX,
          centerY,
          48,
          '#DBEAFE',
          '#2563EB',
          5
        );

        text(
          '电网',
          centerX,
          centerY-8,
          18,
          '#1D4ED8',
          900,
          'center'
        );

        text(
          '灵活 '+value.grid.toFixed(1),
          centerX,
          centerY+19,
          10,
          '#475569',
          760,
          'center'
        );

        box(
          450,
          408,
          100,
          22,
          7,
          '#DCFCE7',
          '#86EFAC'
        );

        text(
          '储能与需求响应',
          centerX,
          419,
          9.2,
          '#15803D',
          820,
          'center'
        );

        var sources=[
          {
            x:185,
            y:250,
            color:'#475569',
            strength:value.fossil
          },
          {
            x:185,
            y:443,
            color:'#0284C7',
            strength:value.renewable*.55
          },
          {
            x:383,
            y:443,
            color:'#D97706',
            strength:value.renewable*.70
          },
          {
            x:383,
            y:250,
            color:'#16A34A',
            strength:value.renewable*.40+
              value.grid*.18
          }
        ];

        sources.forEach(
          function(source){
            curvedArrow(
              source.x,
              source.y,
              (
                source.x+
                centerX
              )/2,
              source.y,
              421,
              centerY,
              source.color,
              2+
              source.strength*.25
            );
          }
        );

        box(
          657,
          205,
          279,
          297,
          16,
          '#FFFFFF',
          '#D1FAE5'
        );

        text(
          '终端用能部门',
          796,
          229,
          13,
          '#065F46',
          880,
          'center'
        );

        var sectors=[
          {
            name:'建筑',
            score:10-value.building*.55,
            color:'#7C3AED'
          },
          {
            name:'交通',
            score:10-value.transit*.56,
            color:'#D97706'
          },
          {
            name:'工业与服务',
            score:derived.energyDemand,
            color:'#2563EB'
          },
          {
            name:'资源循环',
            score:10-value.circular*.58,
            color:'#16A34A'
          }
        ];

        sectors.forEach(
          function(sector,index){
            var y=270+index*52;

            text(
              sector.name,
              680,
              y,
              10,
              '#475569',
              760,
              'left'
            );

            box(
              754,
              y-7,
              150,
              14,
              7,
              '#E2E8F0',
              null
            );

            box(
              754,
              y-7,
              150*
              clamp(
                sector.score/10,
                0,
                1
              ),
              14,
              7,
              sector.color,
              null
            );

            text(
              sector.score.toFixed(1),
              916,
              y,
              9.5,
              sector.color,
              850,
              'right'
            );
          }
        );

        arrow(
          579,
          centerY,
          647,
          centerY,
          '#2563EB',
          4
        );

        if(state.showLabels){
          text(
            '供给侧',
            248,
            188,
            10,
            '#475569',
            850,
            'center'
          );

          text(
            '电网、储能与调节',
            500,
            188,
            10,
            '#2563EB',
            850,
            'center'
          );

          text(
            '需求侧',
            796,
            188,
            10,
            '#065F46',
            850,
            'center'
          );

          text(
            '稳定供能',
            612,
            335,
            9.5,
            '#2563EB',
            820,
            'center'
          );
        }

        box(
          58,
          526,
          878,
          28,
          10,
          '#FFFFFF',
          '#D1FAE5'
        );

        text(
          '能源转型不仅是替换能源来源，还要降低不必要需求并增强电网和储能调节能力。',
          497,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function drawBuilding(
        x,
        y,
        w,
        h,
        color,
        efficiency,
        solar
      ){
        context.fillStyle=color;
        context.fillRect(
          x,
          y-h,
          w,
          h
        );

        context.fillStyle='#DBEAFE';

        for(
          var row=0;
          row<3;
          row+=1
        ){
          for(
            var col=0;
            col<2;
            col+=1
          ){
            context.fillRect(
              x+8+col*17,
              y-h+15+row*22,
              9,
              11
            );
          }
        }

        if(efficiency>=6){
          context.strokeStyle='#22C55E';
          context.lineWidth=4;
          context.strokeRect(
            x+2,
            y-h+2,
            w-4,
            h-4
          );
        }

        if(solar>=5){
          context.fillStyle='#1D4ED8';
          context.beginPath();
          context.moveTo(x+4,y-h);
          context.lineTo(x+w-5,y-h);
          context.lineTo(x+w-11,y-h-13);
          context.lineTo(x+10,y-h-13);
          context.closePath();
          context.fill();
        }
      }

      function drawTree(x,y,size){
        context.fillStyle='#854D0E';
        context.fillRect(
          x-3,
          y,
          6,
          size*.54
        );

        circle(
          x,
          y-4,
          size*.40,
          '#16A34A',
          '#15803D',
          1
        );

        circle(
          x-size*.25,
          y+3,
          size*.29,
          '#22C55E',
          null,
          0
        );

        circle(
          x+size*.25,
          y+3,
          size*.29,
          '#15803D',
          null,
          0
        );
      }

      function cityView(
        item,
        value,
        derived
      ){
        background(
          '低碳城市空间、建筑、交通与绿色基础设施',
          '紧凑混合布局、建筑节能、公共交通和绿色空间可共同降低城市能源与环境压力。'
        );

        card(
          28,
          78,
          210,
          '城市碳压力',
          derived.residualCarbon,
          '#DC2626',
          '能源、建筑和交通综合'
        );

        card(
          250,
          78,
          210,
          '交通效率',
          derived.mobilityEfficiency,
          '#D97706',
          '公交、慢行与空间布局'
        );

        card(
          472,
          78,
          210,
          '空气质量',
          derived.airQuality,
          '#0284C7',
          '排放与绿色空间综合'
        );

        card(
          694,
          78,
          258,
          '可持续发展指数',
          derived.sustainability,
          item.color,
          '能源、资源、公平和韧性'
        );

        var mapX=45;
        var mapY=178;
        var mapW=910;
        var mapH=339;

        box(
          mapX,
          mapY,
          mapW,
          mapH,
          15,
          '#F8FAFC',
          '#D1FAE5'
        );

        context.fillStyle='#DCFCE7';
        context.fillRect(
          mapX+12,
          mapY+12,
          mapW-24,
          mapH-24
        );

        for(
          var horizontal=0;
          horizontal<4;
          horizontal+=1
        ){
          var roadY=
            mapY+
            65+
            horizontal*
            79;

          line(
            mapX+18,
            roadY,
            mapX+mapW-18,
            roadY,
            '#64748B',
            16
          );

          line(
            mapX+18,
            roadY,
            mapX+mapW-18,
            roadY,
            '#FFFFFF',
            2,
            [20,14]
          );
        }

        for(
          var vertical=0;
          vertical<5;
          vertical+=1
        ){
          var roadX=
            mapX+
            75+
            vertical*
            180;

          line(
            roadX,
            mapY+18,
            roadX,
            mapY+mapH-18,
            '#64748B',
            16
          );

          line(
            roadX,
            mapY+18,
            roadX,
            mapY+mapH-18,
            '#FFFFFF',
            2,
            [20,14]
          );
        }

        var buildings=[
          {
            x:138,
            y:275,
            w:44,
            h:92,
            color:'#475569'
          },
          {
            x:225,
            y:275,
            w:52,
            h:126,
            color:'#64748B'
          },
          {
            x:407,
            y:275,
            w:47,
            h:105,
            color:'#334155'
          },
          {
            x:494,
            y:275,
            w:56,
            h:137,
            color:'#475569'
          },
          {
            x:680,
            y:275,
            w:49,
            h:104,
            color:'#64748B'
          },
          {
            x:767,
            y:275,
            w:54,
            h:126,
            color:'#334155'
          },
          {
            x:225,
            y:433,
            w:52,
            h:91,
            color:'#475569'
          },
          {
            x:494,
            y:433,
            w:56,
            h:102,
            color:'#64748B'
          },
          {
            x:767,
            y:433,
            w:54,
            h:87,
            color:'#475569'
          }
        ];

        buildings.forEach(
          function(building){
            drawBuilding(
              mapX+
              building.x-
              45,
              mapY+
              building.y-
              178,
              building.w,
              building.h,
              building.color,
              value.building,
              value.renewable
            );
          }
        );

        var treeCount=
          5+
          Math.round(
            value.green*1.8
          );

        for(
          var tree=0;
          tree<treeCount;
          tree+=1
        ){
          var tx=
            mapX+
            105+
            (
              tree*83
            )%
            780;

          var ty=
            mapY+
            48+
            (
              tree%4
            )*
            79;

          drawTree(
            tx,
            ty,
            19+
            tree%3*
            3
          );
        }

        var transitLineY=
          mapY+
          224;

        line(
          mapX+30,
          transitLineY,
          mapX+mapW-30,
          transitLineY,
          '#F59E0B',
          7
        );

        var stationCount=
          3+
          Math.round(
            value.transit/2
          );

        for(
          var station=0;
          station<stationCount;
          station+=1
        ){
          var stationX=
            mapX+
            70+
            station*
            (
              (
                mapW-140
              )/
              Math.max(
                1,
                stationCount-1
              )
            );

          circle(
            stationX,
            transitLineY,
            8,
            '#FFFFFF',
            '#D97706',
            4
          );
        }

        var busX=
          mapX+
          50+
          state.phase*
          (
            mapW-160
          );

        box(
          busX,
          transitLineY-29,
          74,
          26,
          7,
          '#F59E0B',
          '#B45309'
        );

        text(
          '公交',
          busX+37,
          transitLineY-16,
          9.5,
          '#FFFFFF',
          850,
          'center'
        );

        var bikeY=
          mapY+
          300;

        line(
          mapX+30,
          bikeY,
          mapX+mapW-30,
          bikeY,
          '#16A34A',
          5,
          [10,7]
        );

        var bikeCount=
          1+
          Math.round(
            value.transit/3
          );

        for(
          var bike=0;
          bike<bikeCount;
          bike+=1
        ){
          var bikeX=
            mapX+
            100+
            bike*
            190+
            state.phase*
            55;

          circle(
            bikeX,
            bikeY-7,
            7,
            null,
            '#15803D',
            2
          );

          circle(
            bikeX+19,
            bikeY-7,
            7,
            null,
            '#15803D',
            2
          );

          line(
            bikeX,
            bikeY-7,
            bikeX+10,
            bikeY-22,
            '#15803D',
            2
          );

          line(
            bikeX+10,
            bikeY-22,
            bikeX+19,
            bikeY-7,
            '#15803D',
            2
          );
        }

        box(
          mapX+659,
          mapY+20,
          191,
          59,
          11,
          '#FFFFFF',
          '#A7F3D0'
        );

        text(
          '资源循环中心',
          mapX+755,
          mapY+38,
          10.5,
          '#047857',
          850,
          'center'
        );

        text(
          '回收·再利用·再制造',
          mapX+755,
          mapY+60,
          9.2,
          '#64748B',
          680,
          'center'
        );

        var circularArrowRadius=
          21+
          value.circular*1.4;

        context.strokeStyle='#16A34A';
        context.lineWidth=4;
        context.beginPath();
        context.arc(
          mapX+870,
          mapY+49,
          circularArrowRadius,
          .25,
          Math.PI*1.75
        );
        context.stroke();

        if(state.showLabels){
          text(
            '高效建筑与屋顶光伏',
            mapX+345,
            mapY+32,
            10,
            '#7C3AED',
            850,
            'center'
          );

          text(
            '公共交通主走廊',
            mapX+420,
            transitLineY-15,
            10,
            '#B45309',
            850,
            'center'
          );

          text(
            '连续慢行网络',
            mapX+420,
            bikeY-16,
            10,
            '#15803D',
            850,
            'center'
          );

          text(
            '公园、绿廊与雨洪空间',
            mapX+615,
            mapY+318,
            10,
            '#15803D',
            850,
            'center'
          );
        }

        box(
          58,
          526,
          878,
          28,
          10,
          '#FFFFFF',
          '#D1FAE5'
        );

        text(
          '低碳城市需要能源、建筑、交通、资源循环和公共空间协同，而不是只增加绿化。',
          497,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function metricBar(
        x,
        y,
        w,
        label,
        score,
        color,
        desc
      ){
        box(
          x,
          y,
          w,
          92,
          14,
          '#FFFFFF',
          color
        );

        text(
          label,
          x+16,
          y+21,
          12,
          color,
          860,
          'left'
        );

        text(
          score,
          x+w-16,
          y+21,
          17,
          color,
          900,
          'right'
        );

        text(
          desc,
          x+16,
          y+48,
          9.3,
          '#64748B',
          630,
          'left'
        );

        box(
          x+16,
          y+69,
          w-32,
          10,
          5,
          '#E2E8F0',
          null
        );

        box(
          x+16,
          y+69,
          (
            w-32
          )*
          clamp(
            score/10,
            0,
            1
          ),
          10,
          5,
          color,
          null
        );
      }

      function tradeoffView(
        item,
        value,
        derived
      ){
        background(
          '低碳转型的综合效益、实施约束与公平问题',
          '转型可改善空气质量和能源安全，但也需要投资、基础设施更新和公平治理支持。'
        );

        card(
          28,
          78,
          210,
          '减排后碳压力',
          derived.residualCarbon,
          '#DC2626',
          '多部门措施后的剩余压力'
        );

        card(
          250,
          78,
          210,
          '转型成本压力',
          derived.transitionCost,
          '#D97706',
          '设施更新和转型投入'
        );

        card(
          472,
          78,
          210,
          '公平转型指数',
          derived.equityScore,
          '#7C3AED',
          '公共服务与群体支持'
        );

        card(
          694,
          78,
          258,
          '可持续发展指数',
          derived.sustainability,
          item.color,
          '环境、经济、社会与韧性'
        );

        metricBar(
          58,
          188,
          410,
          '空气质量与健康收益',
          derived.airQuality,
          '#0284C7',
          '降低燃烧排放并增加绿色空间有助于改善空气质量'
        );

        metricBar(
          514,
          188,
          410,
          '能源安全与系统韧性',
          derived.resilience,
          '#2563EB',
          '多元能源、灵活电网和本地资源循环提高韧性'
        );

        metricBar(
          58,
          306,
          410,
          '资源利用效率',
          derived.resourceEfficiency,
          '#16A34A',
          '节能、减量、再利用和材料循环共同降低资源压力'
        );

        metricBar(
          514,
          306,
          410,
          '公平与公共服务可及性',
          derived.equityScore,
          '#7C3AED',
          '价格、交通、就业和社区支持影响转型公平'
        );

        box(
          58,
          426,
          866,
          88,
          14,
          '#FFFFFF',
          '#D1FAE5'
        );

        text(
          '关键权衡',
          80,
          447,
          12,
          '#065F46',
          860,
          'left'
        );

        box(
          92,
          468,
          173,
          30,
          9,
          '#DBEAFE',
          '#93C5FD'
        );

        text(
          '能源安全',
          178,
          483,
          10,
          '#1D4ED8',
          850,
          'center'
        );

        arrow(
          286,
          483,
          348,
          483,
          '#64748B',
          3
        );

        box(
          370,
          468,
          173,
          30,
          9,
          '#FEF3C7',
          '#FBBF24'
        );

        text(
          '转型成本',
          456,
          483,
          10,
          '#B45309',
          850,
          'center'
        );

        arrow(
          564,
          483,
          626,
          483,
          '#64748B',
          3
        );

        box(
          648,
          468,
          173,
          30,
          9,
          '#EDE9FE',
          '#C4B5FD'
        );

        text(
          '社会公平',
          734,
          483,
          10,
          '#6D28D9',
          850,
          'center'
        );

        arrow(
          842,
          483,
          888,
          483,
          item.color,
          3
        );

        text(
          '综合 '+derived.sustainability,
          904,
          483,
          9.5,
          item.color,
          880,
          'center'
        );

        if(state.showLabels){
          text(
            '转型政策需要兼顾低收入群体、传统产业就业和基本能源服务。',
            500,
            535,
            9.4,
            '#065F46',
            760,
            'center'
          );
        }

        box(
          58,
          526,
          866,
          28,
          10,
          '#FFFFFF',
          '#D1FAE5'
        );

        text(
          '低碳不等于简单关闭高碳设施，关键是形成有序替代、能力建设和公平支持。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function pathwayCard(
        x,
        y,
        w,
        titleValue,
        desc,
        color,
        score
      ){
        box(
          x,
          y,
          w,
          91,
          13,
          '#FFFFFF',
          color
        );

        text(
          titleValue,
          x+14,
          y+20,
          11,
          color,
          850,
          'left'
        );

        text(
          desc,
          x+14,
          y+45,
          9.1,
          '#64748B',
          620,
          'left'
        );

        box(
          x+14,
          y+67,
          w-28,
          9,
          5,
          '#E2E8F0',
          null
        );

        box(
          x+14,
          y+67,
          (
            w-28
          )*
          clamp(
            score/10,
            0,
            1
          ),
          9,
          5,
          color,
          null
        );
      }

      function pathwayView(
        item,
        value,
        derived
      ){
        background(
          '从高碳锁定到综合可持续转型',
          '通常应先降低不必要需求，再推进电气化和清洁能源替代，并同步建设电网、储能和公平治理能力。'
        );

        card(
          28,
          78,
          210,
          '能源需求压力',
          derived.energyDemand.toFixed(1),
          '#D97706',
          '效率与行为共同影响'
        );

        card(
          250,
          78,
          210,
          '清洁电力指数',
          derived.cleanPower.toFixed(1),
          '#16A34A',
          '能源替代与电网协同'
        );

        card(
          472,
          78,
          210,
          '系统韧性',
          derived.resilience,
          '#2563EB',
          '多元供能和治理能力'
        );

        card(
          694,
          78,
          258,
          '转型完成度',
          derived.sustainability,
          item.color,
          '多部门综合成效'
        );

        var lineY=261;
        var startX=96;
        var endX=904;

        line(
          startX,
          lineY,
          endX,
          lineY,
          '#CBD5E1',
          7
        );

        var stages=[
          {
            title:'需求管理',
            desc:'节能和减少浪费',
            color:'#7C3AED',
            score:(
              value.building+
              value.circular
            )/2
          },
          {
            title:'终端电气化',
            desc:'建筑、交通和工业替代',
            color:'#2563EB',
            score:(
              value.transit+
              value.building
            )/2
          },
          {
            title:'清洁电力',
            desc:'扩张可再生能源',
            color:'#16A34A',
            score:value.renewable
          },
          {
            title:'灵活电网',
            desc:'储能和需求响应',
            color:'#0284C7',
            score:value.grid
          },
          {
            title:'公平治理',
            desc:'就业、价格和公共服务',
            color:'#D97706',
            score:value.equity
          }
        ];

        stages.forEach(
          function(stage,index){
            var x=
              startX+
              index*
              (
                (
                  endX-startX
                )/
                (
                  stages.length-1
                )
              );

            circle(
              x,
              lineY,
              28,
              '#FFFFFF',
              stage.color,
              5
            );

            circle(
              x,
              lineY,
              18,
              stage.color,
              null,
              0
            );

            text(
              String(index+1),
              x,
              lineY,
              12,
              '#FFFFFF',
              900,
              'center'
            );

            text(
              stage.title,
              x,
              lineY-47,
              10.5,
              stage.color,
              850,
              'center'
            );

            text(
              stage.desc,
              x,
              lineY+48,
              8.8,
              '#64748B',
              650,
              'center'
            );

            box(
              x-61,
              lineY+65,
              122,
              8,
              4,
              '#E2E8F0',
              null
            );

            box(
              x-61,
              lineY+65,
              122*
              stage.score/10,
              8,
              4,
              stage.color,
              null
            );
          }
        );

        pathwayCard(
          58,
          365,
          410,
          '城市部门协同',
          '建筑、交通、产业、能源和空间规划共同降低碳锁定。',
          '#0F766E',
          (
            value.building+
            value.transit+
            value.green
          )/3
        );

        pathwayCard(
          514,
          365,
          410,
          '循环经济与资源效率',
          '从产品设计、生产、消费到回收形成闭环。',
          '#16A34A',
          value.circular
        );

        box(
          58,
          474,
          866,
          40,
          12,
          '#FFFFFF',
          '#D1FAE5'
        );

        text(
          '当前综合结果：碳压力 '+
          derived.residualCarbon+
          ' ｜ 空气质量 '+
          derived.airQuality+
          ' ｜ 公平指数 '+
          derived.equityScore+
          ' ｜ 韧性 '+
          derived.resilience,
          491,
          494,
          10.2,
          item.color,
          850,
          'center'
        );

        if(state.showLabels){
          var progressX=
            startX+
            (
              derived.sustainability/10
            )*
            (
              endX-startX
            );

          line(
            startX,
            lineY,
            progressX,
            lineY,
            item.color,
            7
          );

          circle(
            progressX,
            lineY,
            8,
            '#FFFFFF',
            item.color,
            3
          );

          text(
            '综合转型进度',
            progressX,
            lineY-77,
            9.5,
            item.color,
            820,
            'center'
          );
        }

        box(
          58,
          526,
          866,
          28,
          10,
          '#FFFFFF',
          '#D1FAE5'
        );

        text(
          '综合转型需要按系统顺序推进，避免只扩张供给而忽视需求、网络和社会承受能力。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function update(
        item,
        value,
        derived
      ){
        fossilValue.textContent=
          String(Math.round(value.fossil));

        renewableValue.textContent=
          String(Math.round(value.renewable));

        gridValue.textContent=
          String(Math.round(value.grid));

        buildingValue.textContent=
          String(Math.round(value.building));

        transitValue.textContent=
          String(Math.round(value.transit));

        circularValue.textContent=
          String(Math.round(value.circular));

        greenValue.textContent=
          String(Math.round(value.green));

        equityValue.textContent=
          String(Math.round(value.equity));

        Array.prototype.forEach.call(
          scenarioButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-scenario'
              )===state.scenario
                ? 'true'
                : 'false'
            );
          }
        );

        Array.prototype.forEach.call(
          viewButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-view'
              )===state.view
                ? 'true'
                : 'false'
            );
          }
        );

        labelToggle.setAttribute(
          'data-active',
          state.showLabels
            ? 'true'
            : 'false'
        );

        autoToggle.setAttribute(
          'data-active',
          state.auto
            ? 'true'
            : 'false'
        );

        var scenarioName=
          state.scenario==='custom'
            ? '自定义能源城市条件'
            : item.name;

        result.textContent=
          scenarioName+
          '下，能源需求压力为'+
          derived.energyDemand.toFixed(1)+
          '，清洁电力指数为'+
          derived.cleanPower.toFixed(1)+
          '，电力系统稳定指数为'+
          derived.gridStability.toFixed(1)+
          '，剩余碳压力为'+
          derived.residualCarbon+
          '；空气质量为'+
          derived.airQuality+
          '，公平转型指数为'+
          derived.equityScore+
          '，城市可持续发展指数为'+
          derived.sustainability+
          '。';
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var item=
          scenarioByKey(
            state.scenario
          );

        var value=values();
        var derived=derive(value);

        update(
          item,
          value,
          derived
        );

        context.clearRect(
          0,
          0,
          width,
          height
        );

        if(state.view==='city'){
          cityView(
            item,
            value,
            derived
          );
        }else if(state.view==='tradeoff'){
          tradeoffView(
            item,
            value,
            derived
          );
        }else if(state.view==='pathway'){
          pathwayView(
            item,
            value,
            derived
          );
        }else{
          energyView(
            item,
            value,
            derived
          );
        }
      }

      function applyScenario(
        key,
        changeView
      ){
        var item=
          scenarioByKey(key);

        state.scenario=item.key;
        setInputs(item);

        if(changeView){
          state.view=item.view;
        }

        state.scenarioIndex=
          scenarios.indexOf(item);

        render();
      }

      function stopAuto(){
        state.auto=false;
        state.startedAt=0;

        if(state.raf){
          cancelAnimationFrame(
            state.raf
          );

          state.raf=0;
        }

        render();
      }

      function animate(timestamp){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        if(!state.auto)return;

        if(!state.startedAt){
          state.startedAt=timestamp;
        }

        var elapsed=
          timestamp-
          state.startedAt;

        var duration=5600;

        var segment=
          Math.floor(
            elapsed/duration
          );

        var local=
          (
            elapsed%duration
          )/
          duration;

        var from=
          scenarios[
            segment%
            scenarios.length
          ];

        var to=
          scenarios[
            (
              segment+1
            )%
            scenarios.length
          ];

        var progress=
          ease(
            clamp(
              local/.82,
              0,
              1
            )
          );

        state.scenario=
          local<.5
            ? from.key
            : to.key;

        state.view=
          [
            'energy',
            'city',
            'tradeoff',
            'pathway'
          ][
            Math.floor(
              elapsed/6900
            )%4
          ];

        state.phase=
          (
            elapsed/3700
          )%1;

        setInputs({
          fossil:lerp(
            from.fossil,
            to.fossil,
            progress
          ),
          renewable:lerp(
            from.renewable,
            to.renewable,
            progress
          ),
          grid:lerp(
            from.grid,
            to.grid,
            progress
          ),
          building:lerp(
            from.building,
            to.building,
            progress
          ),
          transit:lerp(
            from.transit,
            to.transit,
            progress
          ),
          circular:lerp(
            from.circular,
            to.circular,
            progress
          ),
          green:lerp(
            from.green,
            to.green,
            progress
          ),
          equity:lerp(
            from.equity,
            to.equity,
            progress
          )
        });

        render();

        state.raf=
          requestAnimationFrame(
            animate
          );
      }

      function manualChange(){
        if(state.auto){
          stopAuto();
        }

        state.scenario='custom';
        render();
      }

      [
        fossilInput,
        renewableInput,
        gridInput,
        buildingInput,
        transitInput,
        circularInput,
        greenInput,
        equityInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            manualChange
          );
        }
      );

      Array.prototype.forEach.call(
        scenarioButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto){
                stopAuto();
              }

              applyScenario(
                button.getAttribute(
                  'data-scenario'
                ) ||
                'integrated-transition',
                true
              );
            }
          );
        }
      );

      Array.prototype.forEach.call(
        viewButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto){
                stopAuto();
              }

              state.view=
                button.getAttribute(
                  'data-view'
                ) ||
                'energy';

              render();
            }
          );
        }
      );

      labelToggle.addEventListener(
        'click',
        function(){
          state.showLabels=
            !state.showLabels;

          render();
        }
      );

      autoToggle.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
            return;
          }

          state.auto=true;
          state.startedAt=0;

          render();

          state.raf=
            requestAnimationFrame(
              animate
            );
        }
      );

      resetButton.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
          }

          state.scenario=
            initial.scenario;

          state.view='energy';

          state.showLabels=
            initial.showLabels;

          setInputs(initial);

          state.scenarioIndex=
            scenarios.indexOf(
              scenarioByKey(
                initial.scenario
              )
            );

          state.phase=0;
          render();
        }
      );

      nextButton.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
          }

          state.scenarioIndex=
            (
              state.scenarioIndex+1
            )%
            scenarios.length;

          var next=
            scenarios[
              state.scenarioIndex
            ];

          state.scenario=next.key;
          state.view=next.view;

          setInputs(next);
          render();
        }
      );

      state.scenarioIndex=
        scenarios.indexOf(
          scenarioByKey(
            initial.scenario
          )
        );

      setInputs(initial);
      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_ENERGY_LOW_CARBON_CITY:
GeographyLabTemplate[] = [
  {
    id: 'geography-energy-transition-low-carbon-city-sustainable-development',
    group: '🌱 全球变化与可持续发展',
    name: '能源转型、低碳城市与可持续发展',
    emoji: '🏙️',
    desc: '调节化石能源、可再生能源、电网灵活性、建筑节能、公共交通、循环经济、绿色空间和公平治理，观察能源系统、低碳城市、综合效益与可持续转型路径。',
    params: [
      {
        key: 'scenario',
        label: '初始能源与城市转型情境',
        type: 'select',
        options: [
          {
            label: '化石能源依赖',
            value: 'fossil-intensive',
          },
          {
            label: '效率优先',
            value: 'efficiency-first',
          },
          {
            label: '可再生能源扩张',
            value: 'renewable-expansion',
          },
          {
            label: '公交导向城市',
            value: 'transit-oriented',
          },
          {
            label: '综合低碳转型',
            value: 'integrated-transition',
          },
        ],
        defaultValue: 'integrated-transition',
      },
      {
        key: 'fossilShare',
        label: '化石能源依赖',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '表示能源系统对煤、石油和天然气等化石能源的相对依赖程度。',
      },
      {
        key: 'renewableShare',
        label: '可再生能源比例',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '综合表示风能、太阳能及其他可再生能源在供能系统中的比例。',
      },
      {
        key: 'gridFlexibility',
        label: '电网与储能灵活性',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '包括跨区域电网、储能、调峰和需求响应等系统调节能力。',
      },
      {
        key: 'buildingEfficiency',
        label: '建筑节能水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '综合表示围护结构、设备效率、能源管理和既有建筑改造水平。',
      },
      {
        key: 'publicTransit',
        label: '公共交通与慢行水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '表示公共交通覆盖、换乘便利、步行和自行车网络的综合水平。',
      },
      {
        key: 'circularEconomy',
        label: '循环经济水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '综合表示减量化、再利用、回收、维修和再制造能力。',
      },
      {
        key: 'greenSpace',
        label: '城市绿色空间',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '包括公园、绿廊、湿地和其他生态基础设施的覆盖与连通程度。',
      },
      {
        key: 'equityGovernance',
        label: '公平治理与转型支持',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '包括就业转型、价格支持、公共服务、社区参与和弱势群体保障。',
      },
      {
        key: 'showLabels',
        label: '显示能源、城市与路径标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildEnergyLowCarbonCityHTML,
  },
]
