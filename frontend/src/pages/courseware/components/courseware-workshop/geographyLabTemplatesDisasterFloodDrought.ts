/**
 * geographyLabTemplatesDisasterFloodDrought.ts
 *
 * 地理第41批B2：
 *   洪涝、干旱形成机制与灾害风险。
 *
 * 教学目标：
 * - 理解降水强度、持续时间、前期土壤含水量和地形汇流对洪涝的影响；
 * - 比较河流洪水、山洪和城市内涝的形成机制；
 * - 理解不透水面增加、排水能力不足和河道行洪受限对城市内涝的影响；
 * - 理解持续少雨、土壤水分不足、蒸发消耗和蓄水不足对干旱风险的影响；
 * - 比较水库调蓄、河道整治、城市排水等工程措施；
 * - 比较监测预警、风险区划、人员转移、节水管理等非工程措施。
 *
 * 教学边界：
 * - 降水、流量、土壤含水量、洪水位和风险指数均为课堂简化示意；
 * - 流域、城市、河道、水库和农田不对应任何真实地区；
 * - 洪涝和干旱风险模型不用于真实工程设计、调度或防灾决策；
 * - 真实预警、蓄泄调度和人员转移应以权威部门发布的信息为准。
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

function buildFloodDroughtHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'river-flood',
    'urban-waterlogging',
    'mountain-flash-flood',
    'seasonal-drought',
    'integrated-governance',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'river-flood',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'river-flood'

  const rainfallIntensity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'rainfallIntensity', 8),
    ),
  )

  const rainfallDuration = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'rainfallDuration', 7),
    ),
  )

  const terrainConvergence = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'terrainConvergence', 6),
    ),
  )

  const channelCapacity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'channelCapacity', 5),
    ),
  )

  const imperviousSurface = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'imperviousSurface', 6),
    ),
  )

  const soilMoisture = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'soilMoisture', 7),
    ),
  )

  const reservoirRegulation = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'reservoirRegulation', 5),
    ),
  )

  const governanceCapacity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'governanceCapacity', 6),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-flood-drought-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #67E8F9;
      border-radius:18px;
      background:#FFFFFF;
      color:#083344;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(8,145,178,.12);
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
      border-bottom:1px solid #A5F3FC;
      background:linear-gradient(
        135deg,
        #ECFEFF,
        #E0F2FE 48%,
        #FEF3C7
      );
    }

    #${rootId} .gl-title{
      color:#0E7490;
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
      border:1px solid #67E8F9;
      border-radius:999px;
      background:#FFFFFF;
      color:#0E7490;
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
      border-right:1px solid #A5F3FC;
      background:linear-gradient(
        180deg,
        #ECFEFF,
        #F0F9FF 52%,
        #FFFBEB
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
        #CFFAFE 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#0E7490;
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
      background:#CFFAFE;
      color:#0E7490;
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
        #FDE68A,
        #67E8F9,
        #60A5FA
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
        #0891B2,
        #2563EB
      );
      box-shadow:0 1px 5px rgba(8,145,178,.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #67E8F9;
      border-radius:9px;
      background:#FFFFFF;
      color:#0E7490;
      font-size:10.5px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#0E7490;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #0891B2,
        #2563EB 58%,
        #7C3AED
      );
      box-shadow:0 5px 13px rgba(8,145,178,.22);
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
      border:1px solid #67E8F9;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #ECFEFF,
        #FFFBEB
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
      border:1px solid #A5F3FC;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-flood-drought-canvas{
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
    <div style="font-size:24px;">🌧️</div>

    <div>
      <div class="gl-title">
        洪涝、干旱形成机制与灾害风险
      </div>

      <div class="gl-subtitle">
        比较流域洪水、城市内涝、山洪和季节性干旱，并评估综合治理措施
      </div>
    </div>

    <div class="gl-note">
      课堂简化模型 · 不用于真实防灾调度
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        灾害情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="river-flood">
          流域洪水
        </button>

        <button type="button" data-scenario="urban-waterlogging">
          城市内涝
        </button>

        <button type="button" data-scenario="mountain-flash-flood">
          山洪风险
        </button>

        <button type="button" data-scenario="seasonal-drought">
          季节性干旱
        </button>

        <button type="button" data-scenario="integrated-governance">
          综合治理方案
        </button>
      </div>

      <div class="gl-section-title">
        水文与治理参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">降水强度</span>
          <span class="gl-value" data-role="intensity-value">8</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${rainfallIntensity}"
          data-role="intensity"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">降水持续时间</span>
          <span class="gl-value" data-role="duration-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${rainfallDuration}"
          data-role="duration"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">地形汇流程度</span>
          <span class="gl-value" data-role="terrain-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${terrainConvergence}"
          data-role="terrain"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">河道与排水能力</span>
          <span class="gl-value" data-role="channel-value">5</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${channelCapacity}"
          data-role="channel"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">城市不透水面</span>
          <span class="gl-value" data-role="impervious-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${imperviousSurface}"
          data-role="impervious"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">前期土壤含水量</span>
          <span class="gl-value" data-role="soil-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${soilMoisture}"
          data-role="soil"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">水库调蓄能力</span>
          <span class="gl-value" data-role="reservoir-value">5</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${reservoirRegulation}"
          data-role="reservoir"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">非工程治理能力</span>
          <span class="gl-value" data-role="governance-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${governanceCapacity}"
          data-role="governance"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          过程标注
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
        洪涝风险取决于降水、汇流、下渗、行洪和调蓄条件；干旱则与持续少雨和水分储备不足密切相关。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="formation">
          形成过程
        </button>

        <button type="button" data-view="urban">
          城市内涝
        </button>

        <button type="button" data-view="comparison">
          洪旱对比
        </button>

        <button type="button" data-view="mitigation">
          防灾治理
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-flood-drought-canvas"
          width="1000"
          height="570"
          data-role="canvas"
          aria-label="洪涝干旱形成机制与灾害风险教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var intensityInput =
        root.querySelector('[data-role="intensity"]');

      var durationInput =
        root.querySelector('[data-role="duration"]');

      var terrainInput =
        root.querySelector('[data-role="terrain"]');

      var channelInput =
        root.querySelector('[data-role="channel"]');

      var imperviousInput =
        root.querySelector('[data-role="impervious"]');

      var soilInput =
        root.querySelector('[data-role="soil"]');

      var reservoirInput =
        root.querySelector('[data-role="reservoir"]');

      var governanceInput =
        root.querySelector('[data-role="governance"]');

      var intensityValue =
        root.querySelector('[data-role="intensity-value"]');

      var durationValue =
        root.querySelector('[data-role="duration-value"]');

      var terrainValue =
        root.querySelector('[data-role="terrain-value"]');

      var channelValue =
        root.querySelector('[data-role="channel-value"]');

      var imperviousValue =
        root.querySelector('[data-role="impervious-value"]');

      var soilValue =
        root.querySelector('[data-role="soil-value"]');

      var reservoirValue =
        root.querySelector('[data-role="reservoir-value"]');

      var governanceValue =
        root.querySelector('[data-role="governance-value"]');

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
        !intensityInput ||
        !durationInput ||
        !terrainInput ||
        !channelInput ||
        !imperviousInput ||
        !soilInput ||
        !reservoirInput ||
        !governanceInput ||
        !intensityValue ||
        !durationValue ||
        !terrainValue ||
        !channelValue ||
        !imperviousValue ||
        !soilValue ||
        !reservoirValue ||
        !governanceValue ||
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
          key:'river-flood',
          name:'流域洪水',
          intensity:8,
          duration:8,
          terrain:6,
          channel:4,
          impervious:3,
          soil:8,
          reservoir:5,
          governance:6,
          view:'formation',
          color:'#2563EB'
        },
        {
          key:'urban-waterlogging',
          name:'城市内涝',
          intensity:9,
          duration:6,
          terrain:4,
          channel:3,
          impervious:9,
          soil:7,
          reservoir:3,
          governance:5,
          view:'urban',
          color:'#7C3AED'
        },
        {
          key:'mountain-flash-flood',
          name:'山洪风险',
          intensity:10,
          duration:5,
          terrain:10,
          channel:4,
          impervious:2,
          soil:8,
          reservoir:2,
          governance:5,
          view:'formation',
          color:'#DC2626'
        },
        {
          key:'seasonal-drought',
          name:'季节性干旱',
          intensity:2,
          duration:2,
          terrain:4,
          channel:5,
          impervious:5,
          soil:2,
          reservoir:3,
          governance:4,
          view:'comparison',
          color:'#D97706'
        },
        {
          key:'integrated-governance',
          name:'综合治理方案',
          intensity:8,
          duration:7,
          terrain:6,
          channel:8,
          impervious:4,
          soil:6,
          reservoir:9,
          governance:9,
          view:'mitigation',
          color:'#16A34A'
        }
      ];

      var initial={
        scenario:'${scenario}',
        intensity:${rainfallIntensity},
        duration:${rainfallDuration},
        terrain:${terrainConvergence},
        channel:${channelCapacity},
        impervious:${imperviousSurface},
        soil:${soilMoisture},
        reservoir:${reservoirRegulation},
        governance:${governanceCapacity},
        showLabels:${showLabels}
      };

      var state={
        scenario:initial.scenario,
        view:'formation',
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

      function scenarioByKey(key){
        var found=scenarios[0];

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
          intensity:clamp(
            Number(intensityInput.value) || 0,
            0,
            10
          ),
          duration:clamp(
            Number(durationInput.value) || 0,
            0,
            10
          ),
          terrain:clamp(
            Number(terrainInput.value) || 0,
            0,
            10
          ),
          channel:clamp(
            Number(channelInput.value) || 0,
            0,
            10
          ),
          impervious:clamp(
            Number(imperviousInput.value) || 0,
            0,
            10
          ),
          soil:clamp(
            Number(soilInput.value) || 0,
            0,
            10
          ),
          reservoir:clamp(
            Number(reservoirInput.value) || 0,
            0,
            10
          ),
          governance:clamp(
            Number(governanceInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        intensityInput.value=
          String(Math.round(value.intensity));

        durationInput.value=
          String(Math.round(value.duration));

        terrainInput.value=
          String(Math.round(value.terrain));

        channelInput.value=
          String(Math.round(value.channel));

        imperviousInput.value=
          String(Math.round(value.impervious));

        soilInput.value=
          String(Math.round(value.soil));

        reservoirInput.value=
          String(Math.round(value.reservoir));

        governanceInput.value=
          String(Math.round(value.governance));
      }

      function derive(value){
        var rainfallLoad=
          clamp(
            value.intensity*.58+
            value.duration*.42,
            0,
            10
          );

        var infiltrationCapacity=
          clamp(
            (
              10-value.impervious
            )*.48+
            (
              10-value.soil
            )*.34+
            value.governance*.08,
            0,
            10
          );

        var surfaceRunoff=
          clamp(
            rainfallLoad*.48+
            value.terrain*.20+
            value.impervious*.22+
            value.soil*.16-
            infiltrationCapacity*.18-
            value.reservoir*.12,
            0,
            10
          );

        var riverPressure=
          clamp(
            surfaceRunoff*.48+
            value.duration*.20+
            value.terrain*.14+
            value.soil*.12-
            value.channel*.28-
            value.reservoir*.22,
            0,
            10
          );

        var urbanWaterlogging=
          clamp(
            value.intensity*.32+
            value.duration*.18+
            value.impervious*.34+
            value.soil*.10-
            value.channel*.30-
            value.governance*.12,
            0,
            10
          );

        var flashFlood=
          clamp(
            value.intensity*.38+
            value.terrain*.36+
            value.soil*.18+
            value.duration*.10-
            value.reservoir*.10-
            value.governance*.10,
            0,
            10
          );

        var floodRisk=
          clamp(
            Math.round(
              Math.max(
                riverPressure,
                urbanWaterlogging,
                flashFlood
              )
            ),
            0,
            10
          );

        var droughtRisk=
          clamp(
            Math.round(
              (
                10-value.intensity
              )*.24+
              (
                10-value.duration
              )*.27+
              (
                10-value.soil
              )*.25+
              (
                10-value.reservoir
              )*.16+
              value.impervious*.08-
              value.governance*.10
            ),
            0,
            10
          );

        var storageBuffer=
          clamp(
            Math.round(
              value.reservoir*.50+
              value.channel*.15+
              (
                10-value.impervious
              )*.15+
              value.governance*.20
            ),
            0,
            10
          );

        var floodResidual=
          clamp(
            Math.round(
              floodRisk-
              value.reservoir*.24-
              value.channel*.18-
              value.governance*.24
            ),
            0,
            10
          );

        var droughtResidual=
          clamp(
            Math.round(
              droughtRisk-
              value.reservoir*.28-
              value.governance*.25-
              (
                10-value.impervious
              )*.08
            ),
            0,
            10
          );

        var resilience=
          clamp(
            Math.round(
              storageBuffer*.38+
              value.governance*.34+
              value.channel*.18+
              (
                10-Math.max(
                  floodResidual,
                  droughtResidual
                )
              )*.10
            ),
            0,
            10
          );

        return {
          rainfallLoad:rainfallLoad,
          infiltrationCapacity:infiltrationCapacity,
          surfaceRunoff:surfaceRunoff,
          riverPressure:riverPressure,
          urbanWaterlogging:urbanWaterlogging,
          flashFlood:flashFlood,
          floodRisk:floodRisk,
          droughtRisk:droughtRisk,
          storageBuffer:storageBuffer,
          floodResidual:floodResidual,
          droughtResidual:droughtResidual,
          resilience:resilience
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
        gradient.addColorStop(.56,'#ECFEFF');
        gradient.addColorStop(1,'#FEF3C7');

        context.fillStyle=gradient;
        context.fillRect(0,0,width,height);

        text(
          titleValue,
          28,
          31,
          18,
          '#0E7490',
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
          '#A5F3FC'
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

      function drawRain(
        x,
        y,
        w,
        count,
        color
      ){
        context.save();
        context.strokeStyle=color;
        context.lineWidth=2;
        context.lineCap='round';

        for(
          var index=0;
          index<count;
          index+=1
        ){
          var ratio=
            count<=1
              ? .5
              : index/(count-1);

          var px=x+ratio*w;
          var offset=(index%3)*7;

          context.beginPath();
          context.moveTo(px,y+offset);
          context.lineTo(px-5,y+20+offset);
          context.stroke();
        }

        context.restore();
      }

      function drawMountain(
        x,
        y,
        w,
        h,
        fill
      ){
        context.fillStyle=fill;
        context.beginPath();
        context.moveTo(x,y+h);
        context.lineTo(x+w*.5,y);
        context.lineTo(x+w,y+h);
        context.closePath();
        context.fill();
      }

      function formationView(
        item,
        value,
        derived
      ){
        background(
          '流域汇流、河道行洪与洪水形成',
          '降水转化为下渗、地表径流和河流汇流，超过河道与调蓄能力时洪水风险上升。'
        );

        card(
          28,
          78,
          210,
          '降水负荷',
          derived.rainfallLoad.toFixed(1),
          '#2563EB',
          '强度和持续时间综合'
        );

        card(
          250,
          78,
          210,
          '地表径流',
          derived.surfaceRunoff.toFixed(1),
          '#0284C7',
          '下渗后形成的汇流水量'
        );

        card(
          472,
          78,
          210,
          '河道压力',
          derived.riverPressure.toFixed(1),
          '#7C3AED',
          '来水与行洪能力对比'
        );

        card(
          694,
          78,
          258,
          '洪涝风险',
          derived.floodRisk,
          item.color,
          '河洪、内涝和山洪最大值'
        );

        var groundY=446;
        var riverY=399;

        var sky=
          context.createLinearGradient(
            0,
            160,
            0,
            groundY
          );

        sky.addColorStop(0,'#DBEAFE');
        sky.addColorStop(1,'#F8FAFC');

        context.fillStyle=sky;
        context.fillRect(38,170,924,groundY-170);

        drawMountain(
          50,
          225,
          250,
          221,
          '#65A30D'
        );

        drawMountain(
          225,
          260,
          230,
          186,
          '#84CC16'
        );

        drawMountain(
          710,
          264,
          225,
          182,
          '#A3E635'
        );

        context.fillStyle='#A16207';
        context.fillRect(
          38,
          groundY,
          924,
          72
        );

        context.fillStyle='#0284C7';
        context.beginPath();
        context.moveTo(365,groundY);
        context.bezierCurveTo(
          430,
          riverY-30,
          540,
          riverY+28,
          625,
          riverY-8
        );
        context.bezierCurveTo(
          715,
          riverY-46,
          790,
          riverY+15,
          962,
          riverY-14
        );
        context.lineTo(962,518);
        context.lineTo(365,518);
        context.closePath();
        context.fill();

        var waterRise=
          derived.riverPressure*5.5;

        context.fillStyle='rgba(37,99,235,.34)';
        context.fillRect(
          366,
          groundY-waterRise,
          596,
          waterRise
        );

        drawRain(
          70,
          180,
          815,
          10+
          Math.round(
            value.intensity*3
          ),
          'rgba(37,99,235,.72)'
        );

        var runoffCount=
          3+
          Math.round(
            derived.surfaceRunoff/2
          );

        for(
          var runoff=0;
          runoff<runoffCount;
          runoff+=1
        ){
          arrow(
            125+runoff*48,
            315+runoff%2*24,
            370+runoff*16,
            417-runoff%2*10,
            '#0E7490',
            2.2
          );
        }

        box(
          510,
          216,
          125,
          105,
          12,
          '#FFFFFF',
          '#60A5FA'
        );

        context.fillStyle='#60A5FA';
        context.fillRect(
          526,
          255,
          93,
          50
        );

        context.fillStyle='#F8FAFC';
        context.fillRect(
          536,
          265,
          73,
          40-
          value.reservoir*2.2
        );

        text(
          '水库',
          572,
          238,
          11,
          '#1D4ED8',
          850,
          'center'
        );

        text(
          '调蓄 '+Math.round(value.reservoir),
          572,
          310,
          9.5,
          '#475569',
          700,
          'center'
        );

        box(
          729,
          337,
          116,
          62,
          10,
          '#FFFFFF',
          '#7C3AED'
        );

        text(
          '河道断面',
          787,
          355,
          10,
          '#7C3AED',
          850,
          'center'
        );

        text(
          '能力 '+Math.round(value.channel),
          787,
          381,
          9.5,
          '#475569',
          700,
          'center'
        );

        if(state.showLabels){
          text(
            '地形汇流',
            220,
            291,
            10,
            '#166534',
            850,
            'center'
          );

          text(
            '地表径流',
            344,
            366,
            10,
            '#0E7490',
            850,
            'center'
          );

          text(
            '河流汇流',
            626,
            478,
            10,
            '#FFFFFF',
            850,
            'center'
          );

          text(
            '洪水位上升',
            882,
            groundY-waterRise-12,
            10,
            '#DC2626',
            850,
            'center'
          );

          line(
            840,
            groundY-waterRise,
            924,
            groundY-waterRise,
            '#DC2626',
            2,
            [5,4]
          );
        }

        box(
          58,
          526,
          866,
          28,
          10,
          '#FFFFFF',
          '#A5F3FC'
        );

        text(
          '当流域来水超过河道行洪和水库调蓄能力时，河流水位会快速上升并可能漫溢。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function building(
        x,
        groundY,
        w,
        h,
        fill
      ){
        context.fillStyle=fill;
        context.fillRect(
          x,
          groundY-h,
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
              x+9+col*20,
              groundY-h+14+row*23,
              11,
              14
            );
          }
        }
      }

      function urbanView(
        item,
        value,
        derived
      ){
        background(
          '城市不透水面、排水能力与内涝',
          '道路和建筑减少下渗，短时强降水超过排水能力时，低洼区域容易迅速积水。'
        );

        card(
          28,
          78,
          210,
          '不透水面',
          Math.round(value.impervious),
          '#7C3AED',
          '道路、建筑和硬化地面'
        );

        card(
          250,
          78,
          210,
          '下渗能力',
          derived.infiltrationCapacity.toFixed(1),
          '#16A34A',
          '土壤和绿地吸收能力'
        );

        card(
          472,
          78,
          210,
          '排水能力',
          Math.round(value.channel),
          '#0284C7',
          '管网、泵站和河道综合'
        );

        card(
          694,
          78,
          258,
          '城市内涝风险',
          Math.round(derived.urbanWaterlogging),
          item.color,
          '降水与排水能力对比'
        );

        var groundY=430;

        context.fillStyle='#CBD5E1';
        context.fillRect(42,groundY,916,88);

        context.fillStyle='#475569';
        context.fillRect(42,397,916,48);

        line(
          42,
          421,
          958,
          421,
          '#FDE68A',
          4,
          [24,16]
        );

        building(
          84,
          397,
          54,
          128,
          '#64748B'
        );

        building(
          165,
          397,
          64,
          167,
          '#334155'
        );

        building(
          760,
          397,
          61,
          142,
          '#475569'
        );

        building(
          844,
          397,
          56,
          109,
          '#64748B'
        );

        context.fillStyle='#22C55E';
        context.fillRect(309,382,138,15);

        for(
          var tree=0;
          tree<5;
          tree+=1
        ){
          circle(
            327+tree*25,
            367,
            14,
            '#16A34A',
            null,
            0
          );
        }

        var waterDepth=
          derived.urbanWaterlogging*8;

        context.fillStyle='rgba(37,99,235,.48)';
        context.fillRect(
          42,
          groundY-waterDepth,
          916,
          waterDepth
        );

        drawRain(
          70,
          168,
          850,
          12+
          Math.round(
            value.intensity*3.2
          ),
          'rgba(37,99,235,.74)'
        );

        for(
          var inlet=0;
          inlet<5;
          inlet+=1
        ){
          var inletX=285+inlet*91;

          box(
            inletX,
            406,
            38,
            15,
            3,
            '#0F172A',
            '#94A3B8'
          );

          line(
            inletX+7,
            410,
            inletX+31,
            410,
            '#64748B',
            1
          );

          line(
            inletX+7,
            416,
            inletX+31,
            416,
            '#64748B',
            1
          );
        }

        context.fillStyle='#0F766E';
        context.fillRect(280,460,445,22);

        context.fillStyle='#FFFFFF';
        context.fillRect(
          295,
          465,
          415*
          value.channel/10,
          12
        );

        if(state.showLabels){
          text(
            '硬化地面：下渗减少、径流增多',
            500,
            199,
            10,
            '#7C3AED',
            850,
            'center'
          );

          arrow(
            500,
            216,
            500,
            363,
            '#7C3AED',
            3
          );

          text(
            '绿地与下凹空间',
            378,
            346,
            10,
            '#15803D',
            850,
            'center'
          );

          text(
            '排水管网',
            502,
            496,
            10,
            '#0F766E',
            850,
            'center'
          );

          text(
            '低洼区积水',
            660,
            groundY-waterDepth-13,
            10,
            '#1D4ED8',
            850,
            'center'
          );
        }

        box(
          72,
          224,
          197,
          111,
          13,
          '#FFFFFF',
          '#A5F3FC'
        );

        text(
          '城市内涝条件',
          90,
          246,
          11.5,
          '#0E7490',
          850,
          'left'
        );

        text(
          '短时强降水',
          90,
          273,
          9.8,
          '#475569',
          680,
          'left'
        );

        text(
          '不透水面比例高',
          90,
          297,
          9.8,
          '#475569',
          680,
          'left'
        );

        text(
          '排水能力不足',
          90,
          321,
          9.8,
          '#475569',
          680,
          'left'
        );

        box(
          58,
          526,
          866,
          28,
          10,
          '#FFFFFF',
          '#A5F3FC'
        );

        text(
          '城市内涝不仅由雨量决定，还与不透水面、排水管网和低洼地形共同相关。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function riskBar(
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
          value,
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
            value/10,
            0,
            1
          ),
          10,
          5,
          color,
          null
        );
      }

      function comparisonView(
        item,
        value,
        derived
      ){
        background(
          '洪涝与干旱风险对比',
          '洪涝表现为短时间水量过多，干旱表现为较长时期水分供给不足，两者均受水资源管理影响。'
        );

        card(
          28,
          78,
          210,
          '洪涝风险',
          derived.floodRisk,
          '#2563EB',
          '河洪、内涝和山洪综合'
        );

        card(
          250,
          78,
          210,
          '干旱风险',
          derived.droughtRisk,
          '#D97706',
          '少雨、土壤和蓄水综合'
        );

        card(
          472,
          78,
          210,
          '调蓄缓冲',
          derived.storageBuffer,
          '#16A34A',
          '水库、下渗和治理能力'
        );

        card(
          694,
          78,
          258,
          '综合韧性',
          derived.resilience,
          item.color,
          '应对水量过多和不足'
        );

        riskBar(
          58,
          188,
          410,
          '流域洪水风险',
          Math.round(derived.riverPressure),
          '#2563EB',
          '持续降水与汇流超过河道和水库承载'
        );

        riskBar(
          514,
          188,
          410,
          '城市内涝风险',
          Math.round(derived.urbanWaterlogging),
          '#7C3AED',
          '强降水超过城市下渗和排水能力'
        );

        riskBar(
          58,
          306,
          410,
          '山洪风险',
          Math.round(derived.flashFlood),
          '#DC2626',
          '短时强降水与陡坡快速汇流叠加'
        );

        riskBar(
          514,
          306,
          410,
          '季节性干旱风险',
          derived.droughtRisk,
          '#D97706',
          '持续少雨、土壤缺水和蓄水不足'
        );

        box(
          58,
          426,
          866,
          88,
          14,
          '#FFFFFF',
          '#A5F3FC'
        );

        text(
          '水资源调节关系',
          80,
          448,
          12,
          '#0E7490',
          860,
          'left'
        );

        box(
          112,
          469,
          215,
          28,
          9,
          '#DBEAFE',
          '#60A5FA'
        );

        text(
          '汛期：削峰、滞洪、排涝',
          219,
          483,
          10,
          '#1D4ED8',
          820,
          'center'
        );

        arrow(
          354,
          483,
          465,
          483,
          '#16A34A',
          3
        );

        box(
          492,
          469,
          215,
          28,
          9,
          '#DCFCE7',
          '#86EFAC'
        );

        text(
          '水库、湿地、土壤蓄水',
          599,
          483,
          10,
          '#15803D',
          820,
          'center'
        );

        arrow(
          734,
          483,
          817,
          483,
          '#D97706',
          3
        );

        box(
          839,
          469,
          66,
          28,
          9,
          '#FEF3C7',
          '#FBBF24'
        );

        text(
          '旱期供水',
          872,
          483,
          9.4,
          '#B45309',
          820,
          'center'
        );

        box(
          58,
          526,
          866,
          28,
          10,
          '#FFFFFF',
          '#A5F3FC'
        );

        text(
          '同一地区可能在不同季节经历洪涝和干旱，因此需要统筹蓄、滞、排、用和节水。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function measureCard(
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
          98,
          14,
          '#FFFFFF',
          color
        );

        text(
          titleValue,
          x+16,
          y+21,
          11.5,
          color,
          860,
          'left'
        );

        text(
          desc,
          x+16,
          y+49,
          9.2,
          '#64748B',
          620,
          'left'
        );

        box(
          x+16,
          y+72,
          w-32,
          9,
          5,
          '#E2E8F0',
          null
        );

        box(
          x+16,
          y+72,
          (
            w-32
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

      function mitigationView(
        item,
        value,
        derived
      ){
        background(
          '洪旱灾害的工程与非工程治理',
          '工程措施改善蓄泄和排水条件，非工程措施降低暴露度并提高预警和组织响应能力。'
        );

        card(
          28,
          78,
          210,
          '治理前洪涝风险',
          derived.floodRisk,
          '#2563EB',
          '未扣除治理效果'
        );

        card(
          250,
          78,
          210,
          '治理后洪涝风险',
          derived.floodResidual,
          '#0E7490',
          '工程和非工程措施共同作用'
        );

        card(
          472,
          78,
          210,
          '治理后干旱风险',
          derived.droughtResidual,
          '#D97706',
          '蓄水和节水措施共同作用'
        );

        card(
          694,
          78,
          258,
          '综合韧性',
          derived.resilience,
          item.color,
          '适应洪涝与干旱的能力'
        );

        measureCard(
          58,
          188,
          410,
          '水库、堤防与河道整治',
          '发挥削峰、滞洪、行洪和旱期供水作用。',
          '#2563EB',
          (
            value.reservoir+
            value.channel
          )/2
        );

        measureCard(
          514,
          188,
          410,
          '城市排水与海绵设施',
          '提高排水能力，增加绿地、湿地和雨水调蓄空间。',
          '#7C3AED',
          (
            value.channel+
            (
              10-value.impervious
            )
          )/2
        );

        measureCard(
          58,
          310,
          410,
          '监测预警与人员转移',
          '依据雨情、水情和风险区划提前发布预警并组织避险。',
          '#DC2626',
          value.governance
        );

        measureCard(
          514,
          310,
          410,
          '节水管理与抗旱调度',
          '优化生活、农业和工业用水，提高旱期保障能力。',
          '#D97706',
          (
            value.reservoir+
            value.governance
          )/2
        );

        box(
          58,
          430,
          866,
          84,
          14,
          '#FFFFFF',
          '#A5F3FC'
        );

        text(
          '风险变化',
          80,
          452,
          12,
          '#0E7490',
          860,
          'left'
        );

        box(
          102,
          468,
          245,
          30,
          10,
          '#DBEAFE',
          '#93C5FD'
        );

        text(
          '洪涝风险 '+derived.floodRisk,
          224,
          483,
          10.5,
          '#1D4ED8',
          850,
          'center'
        );

        arrow(
          374,
          483,
          602,
          483,
          '#16A34A',
          4
        );

        box(
          628,
          468,
          245,
          30,
          10,
          '#DCFCE7',
          '#86EFAC'
        );

        text(
          '治理后 '+derived.floodResidual,
          750,
          483,
          10.5,
          '#15803D',
          850,
          'center'
        );

        box(
          58,
          526,
          866,
          28,
          10,
          '#FFFFFF',
          '#A5F3FC'
        );

        text(
          '工程措施不能替代预警和组织管理，非工程措施也不能替代必要的基础设施建设。',
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
        intensityValue.textContent=
          String(Math.round(value.intensity));

        durationValue.textContent=
          String(Math.round(value.duration));

        terrainValue.textContent=
          String(Math.round(value.terrain));

        channelValue.textContent=
          String(Math.round(value.channel));

        imperviousValue.textContent=
          String(Math.round(value.impervious));

        soilValue.textContent=
          String(Math.round(value.soil));

        reservoirValue.textContent=
          String(Math.round(value.reservoir));

        governanceValue.textContent=
          String(Math.round(value.governance));

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
            ? '自定义水文条件'
            : item.name;

        result.textContent=
          scenarioName+
          '下，地表径流为'+
          derived.surfaceRunoff.toFixed(1)+
          '，流域洪水压力为'+
          derived.riverPressure.toFixed(1)+
          '，城市内涝风险为'+
          derived.urbanWaterlogging.toFixed(1)+
          '，山洪风险为'+
          derived.flashFlood.toFixed(1)+
          '，干旱风险为'+
          derived.droughtRisk+
          '；综合治理后的洪涝剩余风险为'+
          derived.floodResidual+
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

        if(state.view==='urban'){
          urbanView(
            item,
            value,
            derived
          );
        }else if(state.view==='comparison'){
          comparisonView(
            item,
            value,
            derived
          );
        }else if(state.view==='mitigation'){
          mitigationView(
            item,
            value,
            derived
          );
        }else{
          formationView(
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

        var duration=5400;
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
            'formation',
            'urban',
            'comparison',
            'mitigation'
          ][
            Math.floor(
              elapsed/6600
            )%4
          ];

        state.phase=
          (
            elapsed/3600
          )%1;

        setInputs({
          intensity:lerp(
            from.intensity,
            to.intensity,
            progress
          ),
          duration:lerp(
            from.duration,
            to.duration,
            progress
          ),
          terrain:lerp(
            from.terrain,
            to.terrain,
            progress
          ),
          channel:lerp(
            from.channel,
            to.channel,
            progress
          ),
          impervious:lerp(
            from.impervious,
            to.impervious,
            progress
          ),
          soil:lerp(
            from.soil,
            to.soil,
            progress
          ),
          reservoir:lerp(
            from.reservoir,
            to.reservoir,
            progress
          ),
          governance:lerp(
            from.governance,
            to.governance,
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
        intensityInput,
        durationInput,
        terrainInput,
        channelInput,
        imperviousInput,
        soilInput,
        reservoirInput,
        governanceInput
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
                'river-flood',
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
                'formation';

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

          state.view='formation';

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

export const GEOGRAPHY_LAB_TEMPLATES_DISASTER_FLOOD_DROUGHT:
GeographyLabTemplate[] = [
  {
    id: 'geography-flood-drought-mechanism-disaster-risk',
    group: '🌪️ 自然灾害与地理信息技术',
    name: '洪涝、干旱形成机制与灾害风险',
    emoji: '🌧️',
    desc: '调节降水强度、持续时间、地形汇流、河道排水、不透水面、土壤含水量、水库调蓄和治理能力，比较流域洪水、城市内涝、山洪与干旱风险。',
    params: [
      {
        key: 'scenario',
        label: '初始洪旱灾害情境',
        type: 'select',
        options: [
          {
            label: '流域洪水',
            value: 'river-flood',
          },
          {
            label: '城市内涝',
            value: 'urban-waterlogging',
          },
          {
            label: '山洪风险',
            value: 'mountain-flash-flood',
          },
          {
            label: '季节性干旱',
            value: 'seasonal-drought',
          },
          {
            label: '综合治理方案',
            value: 'integrated-governance',
          },
        ],
        defaultValue: 'river-flood',
      },
      {
        key: 'rainfallIntensity',
        label: '降水强度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 8,
        hint: '数值越高，单位时间内的降水量越大。',
      },
      {
        key: 'rainfallDuration',
        label: '降水持续时间',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '持续时间越长，土壤和河道承受的累积水量越大。',
      },
      {
        key: 'terrainConvergence',
        label: '地形汇流程度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '陡坡、狭窄谷地和较高汇流效率可加快地表水集中。',
      },
      {
        key: 'channelCapacity',
        label: '河道与排水能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '综合表示河道行洪、城市管网、泵站和排涝能力。',
      },
      {
        key: 'imperviousSurface',
        label: '城市不透水面比例',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '道路和建筑硬化面增加会减少下渗并增加地表径流。',
      },
      {
        key: 'soilMoisture',
        label: '前期土壤含水量',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '土壤已接近饱和时，新增降水更容易形成地表径流。',
      },
      {
        key: 'reservoirRegulation',
        label: '水库调蓄能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '水库和蓄滞洪空间可在汛期削峰，并在旱期提供水源。',
      },
      {
        key: 'governanceCapacity',
        label: '非工程治理能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '包括监测预警、风险区划、人员转移、节水和应急管理。',
      },
      {
        key: 'showLabels',
        label: '显示水文过程与风险标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildFloodDroughtHTML,
  },
]
