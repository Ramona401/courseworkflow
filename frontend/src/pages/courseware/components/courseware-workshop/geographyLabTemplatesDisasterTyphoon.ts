/**
 * geographyLabTemplatesDisasterTyphoon.ts
 *
 * 地理第41批B1：
 *   台风结构、移动路径与灾害影响。
 *
 * 教学目标：
 * - 认识台风眼、眼墙、螺旋雨带和外围大风区的基本结构；
 * - 理解暖海面、水汽供应和引导气流对台风发展与路径的影响；
 * - 比较台风在海上发展、靠近海岸、登陆和深入内陆后的变化；
 * - 综合分析大风、暴雨、风暴潮和山地次生灾害风险；
 * - 比较监测预警、人员转移、海岸防护和城市排涝等减灾措施。
 *
 * 教学边界：
 * - 台风路径、气压、风速、降水量和风险指数均为课堂简化示意；
 * - 海岸线、城市、山地和人口暴露不对应任何真实地区；
 * - 本模板不用于真实天气预报、航行判断或防灾决策；
 * - 真实台风信息应以权威气象部门发布内容为准。
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

function buildTyphoonHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'open-ocean',
    'approaching-coast',
    'landfall',
    'inland-weakening',
    'preparedness',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'approaching-coast',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'approaching-coast'

  const progress = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'progress', 48),
    ),
  )

  const seaTemperature = Math.max(
    24,
    Math.min(
      31,
      numberValue(params, 'seaTemperature', 29),
    ),
  )

  const steeringFlow = Math.max(
    -10,
    Math.min(
      10,
      numberValue(params, 'steeringFlow', -2),
    ),
  )

  const moistureSupply = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'moistureSupply', 8),
    ),
  )

  const terrainFriction = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'terrainFriction', 5),
    ),
  )

  const preparedness = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'preparedness', 6),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-typhoon-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #7DD3FC;
      border-radius:18px;
      background:#FFFFFF;
      color:#082F49;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(3,105,161,.12);
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
      border-bottom:1px solid #BAE6FD;
      background:linear-gradient(
        135deg,
        #ECFEFF,
        #E0F2FE 52%,
        #EEF2FF
      );
    }

    #${rootId} .gl-title{
      color:#075985;
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
      border:1px solid #7DD3FC;
      border-radius:999px;
      background:#FFFFFF;
      color:#0369A1;
      font-size:11px;
      font-weight:760;
      white-space:nowrap;
    }

    #${rootId} .gl-body{
      height:calc(100% - 56px);
      display:grid;
      grid-template-columns:284px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      min-height:0;
      padding:13px;
      overflow:auto;
      border-right:1px solid #BAE6FD;
      background:linear-gradient(
        180deg,
        #F0F9FF,
        #ECFEFF 58%,
        #EEF2FF
      );
    }

    #${rootId} .gl-stage{
      min-width:0;
      min-height:0;
      display:grid;
      grid-template-rows:46px minmax(0,1fr);
      padding:8px;
      background:radial-gradient(
        circle at 46% 18%,
        #FFFFFF 0%,
        #F8FAFC 58%,
        #E0F2FE 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#075985;
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
      margin-bottom:10px;
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
      font-size:11.4px;
      font-weight:730;
    }

    #${rootId} .gl-value{
      min-width:48px;
      padding:3px 7px;
      border-radius:999px;
      background:#DBEAFE;
      color:#0369A1;
      font-size:11px;
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
        #93C5FD,
        #67E8F9,
        #A5B4FC
      );
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:16px;
      height:16px;
      appearance:none;
      border:2px solid #FFFFFF;
      border-radius:50%;
      background:linear-gradient(135deg,#0284C7,#4F46E5);
      box-shadow:0 1px 5px rgba(3,105,161,.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #7DD3FC;
      border-radius:9px;
      background:#FFFFFF;
      color:#0369A1;
      font-size:10.6px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#0369A1;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #0284C7,
        #2563EB 55%,
        #4F46E5
      );
      box-shadow:0 5px 13px rgba(3,105,161,.22);
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
      border:1px solid #7DD3FC;
      border-radius:12px;
      background:linear-gradient(135deg,#F0F9FF,#EEF2FF);
      color:#334155;
      font-size:11.1px;
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
      border:1px solid #BAE6FD;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-typhoon-canvas{
      width:100%;
      height:100%;
      display:block;
    }

    @media(max-width:900px){
      #${rootId} .gl-body{
        grid-template-columns:242px minmax(0,1fr);
      }

      #${rootId} .gl-note{
        display:none;
      }
    }
  </style>

  <div class="gl-head">
    <div style="font-size:24px;">🌀</div>

    <div>
      <div class="gl-title">
        台风结构、移动路径与灾害影响
      </div>

      <div class="gl-subtitle">
        观察台风眼、眼墙、螺旋雨带、路径变化和登陆前后灾害风险
      </div>
    </div>

    <div class="gl-note">
      课堂简化模型 · 不用于真实天气预报
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        台风发展情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="open-ocean">
          暖洋面发展
        </button>

        <button type="button" data-scenario="approaching-coast">
          靠近海岸
        </button>

        <button type="button" data-scenario="landfall">
          台风登陆
        </button>

        <button type="button" data-scenario="inland-weakening">
          深入内陆
        </button>

        <button type="button" data-scenario="preparedness">
          防灾避险方案
        </button>
      </div>

      <div class="gl-section-title">
        环境与防灾参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">发展过程进度</span>
          <span class="gl-value" data-role="progress-value">48%</span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${progress}"
          data-role="progress"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">海面温度</span>
          <span class="gl-value" data-role="temperature-value">29℃</span>
        </div>

        <input
          type="range"
          min="24"
          max="31"
          step="1"
          value="${seaTemperature}"
          data-role="temperature"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">引导气流偏转</span>
          <span class="gl-value" data-role="steering-value">-2</span>
        </div>

        <input
          type="range"
          min="-10"
          max="10"
          step="1"
          value="${steeringFlow}"
          data-role="steering"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">水汽供应</span>
          <span class="gl-value" data-role="moisture-value">8</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${moistureSupply}"
          data-role="moisture"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">陆地摩擦与地形作用</span>
          <span class="gl-value" data-role="terrain-value">5</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${terrainFriction}"
          data-role="terrain"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">防灾准备水平</span>
          <span class="gl-value" data-role="preparedness-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${preparedness}"
          data-role="preparedness"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          结构标注
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
        台风在暖洋面上获得水汽和能量，靠近海岸后需要同时关注大风、暴雨和风暴潮。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="structure">
          台风结构
        </button>

        <button type="button" data-view="track">
          移动路径
        </button>

        <button type="button" data-view="hazards">
          灾害影响
        </button>

        <button type="button" data-view="response">
          防灾避险
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-typhoon-canvas"
          width="1000"
          height="570"
          data-role="canvas"
          aria-label="台风结构移动路径与灾害影响教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var progressInput =
        root.querySelector('[data-role="progress"]');

      var temperatureInput =
        root.querySelector('[data-role="temperature"]');

      var steeringInput =
        root.querySelector('[data-role="steering"]');

      var moistureInput =
        root.querySelector('[data-role="moisture"]');

      var terrainInput =
        root.querySelector('[data-role="terrain"]');

      var preparednessInput =
        root.querySelector('[data-role="preparedness"]');

      var progressValue =
        root.querySelector('[data-role="progress-value"]');

      var temperatureValue =
        root.querySelector('[data-role="temperature-value"]');

      var steeringValue =
        root.querySelector('[data-role="steering-value"]');

      var moistureValue =
        root.querySelector('[data-role="moisture-value"]');

      var terrainValue =
        root.querySelector('[data-role="terrain-value"]');

      var preparednessValue =
        root.querySelector('[data-role="preparedness-value"]');

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
        !progressInput ||
        !temperatureInput ||
        !steeringInput ||
        !moistureInput ||
        !terrainInput ||
        !preparednessInput ||
        !progressValue ||
        !temperatureValue ||
        !steeringValue ||
        !moistureValue ||
        !terrainValue ||
        !preparednessValue ||
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
          key:'open-ocean',
          name:'暖洋面发展',
          progress:26,
          temperature:30,
          steering:-5,
          moisture:9,
          terrain:0,
          preparedness:4,
          view:'structure',
          color:'#2563EB'
        },
        {
          key:'approaching-coast',
          name:'靠近海岸',
          progress:52,
          temperature:29,
          steering:-2,
          moisture:9,
          terrain:2,
          preparedness:6,
          view:'track',
          color:'#7C3AED'
        },
        {
          key:'landfall',
          name:'台风登陆',
          progress:70,
          temperature:28,
          steering:1,
          moisture:8,
          terrain:6,
          preparedness:7,
          view:'hazards',
          color:'#DC2626'
        },
        {
          key:'inland-weakening',
          name:'深入内陆',
          progress:88,
          temperature:26,
          steering:4,
          moisture:5,
          terrain:9,
          preparedness:7,
          view:'hazards',
          color:'#EA580C'
        },
        {
          key:'preparedness',
          name:'防灾避险方案',
          progress:62,
          temperature:29,
          steering:0,
          moisture:8,
          terrain:5,
          preparedness:9,
          view:'response',
          color:'#16A34A'
        }
      ];

      var initial={
        scenario:'${scenario}',
        progress:${progress},
        temperature:${seaTemperature},
        steering:${steeringFlow},
        moisture:${moistureSupply},
        terrain:${terrainFriction},
        preparedness:${preparedness},
        showLabels:${showLabels}
      };

      var state={
        scenario:initial.scenario,
        view:'structure',
        showLabels:initial.showLabels,
        auto:false,
        startedAt:0,
        phase:0,
        scenarioIndex:0,
        raf:0
      };

      function clamp(value,min,max){
        return Math.max(min,Math.min(max,value));
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

        context.fillText(String(value),x,y);
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
        var head=12;

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=lineWidth || 4;
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

      function spiral(
        cx,
        cy,
        startRadius,
        endRadius,
        startAngle,
        turns,
        color,
        lineWidth,
        alpha
      ){
        var steps=110;

        context.save();
        context.strokeStyle=color;
        context.lineWidth=lineWidth;
        context.globalAlpha=alpha;
        context.lineCap='round';
        context.beginPath();

        for(
          var index=0;
          index<=steps;
          index+=1
        ){
          var ratio=index/steps;
          var radius=lerp(startRadius,endRadius,ratio);

          var angle=
            startAngle+
            ratio*
            Math.PI*
            2*
            turns;

          var x=
            cx+
            Math.cos(angle)*
            radius;

          var y=
            cy+
            Math.sin(angle)*
            radius*
            .72;

          if(index===0){
            context.moveTo(x,y);
          }else{
            context.lineTo(x,y);
          }
        }

        context.stroke();
        context.restore();
      }

      function scenarioByKey(key){
        var found=scenarios[1];

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
          progress:clamp(
            Number(progressInput.value) || 0,
            0,
            100
          ),
          temperature:clamp(
            Number(temperatureInput.value) || 24,
            24,
            31
          ),
          steering:clamp(
            Number(steeringInput.value) || 0,
            -10,
            10
          ),
          moisture:clamp(
            Number(moistureInput.value) || 0,
            0,
            10
          ),
          terrain:clamp(
            Number(terrainInput.value) || 0,
            0,
            10
          ),
          preparedness:clamp(
            Number(preparednessInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        progressInput.value=String(Math.round(value.progress));
        temperatureInput.value=String(Math.round(value.temperature));
        steeringInput.value=String(Math.round(value.steering));
        moistureInput.value=String(Math.round(value.moisture));
        terrainInput.value=String(Math.round(value.terrain));
        preparednessInput.value=String(Math.round(value.preparedness));
      }

      function derive(value){
        var heatEnergy=
          clamp(
            (
              value.temperature-24
            )*
            1.35,
            0,
            10
          );

        var oceanSupport=
          clamp(
            heatEnergy*.62+
            value.moisture*.38,
            0,
            10
          );

        var lifecycleFactor;

        if(value.progress<30){
          lifecycleFactor=
            .48+
            value.progress/30*.38;
        }else if(value.progress<68){
          lifecycleFactor=
            .86+
            (
              value.progress-30
            )/38*.14;
        }else{
          lifecycleFactor=
            1-
            (
              value.progress-68
            )/32*.35;
        }

        var landWeakening=
          value.terrain*
          (
            value.progress>=60
              ? .48
              : .18
          );

        var intensity=
          clamp(
            Math.round(
              oceanSupport*
              lifecycleFactor-
              landWeakening
            ),
            0,
            10
          );

        var pressure=
          Math.round(
            1010-
            intensity*6.4
          );

        var wind=
          Math.round(
            18+
            intensity*6.6
          );

        var rainfall=
          clamp(
            Math.round(
              value.moisture*.55+
              intensity*.42+
              value.terrain*.22
            ),
            0,
            10
          );

        var surge=
          clamp(
            Math.round(
              intensity*.58+
              Math.max(
                0,
                5-Math.abs(value.steering)
              )*.18+
              (
                value.progress>=45 &&
                value.progress<=76
                  ? 2.2
                  : .4
              )
            ),
            0,
            10
          );

        var rawRisk=
          clamp(
            Math.round(
              intensity*.34+
              rainfall*.28+
              surge*.26+
              value.terrain*.12
            ),
            0,
            10
          );

        var residualRisk=
          clamp(
            Math.round(
              rawRisk-
              value.preparedness*.48
            ),
            0,
            10
          );

        var responseScore=
          clamp(
            Math.round(
              value.preparedness*.72+
              (
                10-residualRisk
              )*.28
            ),
            0,
            10
          );

        return {
          heatEnergy:heatEnergy,
          oceanSupport:oceanSupport,
          intensity:intensity,
          pressure:pressure,
          wind:wind,
          rainfall:rainfall,
          surge:surge,
          rawRisk:rawRisk,
          residualRisk:residualRisk,
          responseScore:responseScore
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
        gradient.addColorStop(.55,'#F0F9FF');
        gradient.addColorStop(1,'#E0E7FF');

        context.fillStyle=gradient;
        context.fillRect(0,0,width,height);

        text(
          titleValue,
          28,
          31,
          18,
          '#075985',
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
          '#BAE6FD'
        );

        text(label,x+14,y+17,10.5,'#64748B',720,'left');
        text(value,x+14,y+40,20,color,880,'left');
        text(desc,x+14,y+59,9.5,'#64748B',600,'left');
      }

      function structureView(item,value,derived){
        background(
          '台风的水平结构',
          '台风眼相对平静，眼墙附近风雨最强，外围分布螺旋雨带。'
        );

        card(
          28,
          78,
          210,
          '中心气压',
          derived.pressure+' hPa',
          '#DC2626',
          '强度增大时气压降低'
        );

        card(
          250,
          78,
          210,
          '近中心风速',
          derived.wind+' m/s',
          '#7C3AED',
          '课堂简化风速指标'
        );

        card(
          472,
          78,
          210,
          '海洋能量支持',
          derived.oceanSupport.toFixed(1),
          '#0284C7',
          '海温与水汽共同作用'
        );

        card(
          694,
          78,
          258,
          '台风强度指数',
          derived.intensity,
          item.color,
          '环境和下垫面综合结果'
        );

        var cx=500;
        var cy=350;
        var outerRadius=154+derived.intensity*5;

        var ocean=
          context.createRadialGradient(
            cx,
            cy,
            20,
            cx,
            cy,
            outerRadius+78
          );

        ocean.addColorStop(0,'#E0F2FE');
        ocean.addColorStop(.62,'#7DD3FC');
        ocean.addColorStop(1,'#0369A1');

        circle(
          cx,
          cy,
          outerRadius+62,
          ocean,
          '#075985',
          2
        );

        for(
          var band=0;
          band<5;
          band+=1
        ){
          spiral(
            cx,
            cy,
            55+band*15,
            outerRadius+band*8,
            state.phase*Math.PI*2+
            band*1.18,
            1.45,
            band%2===0
              ? '#FFFFFF'
              : '#DBEAFE',
            12-band,
            .72
          );
        }

        circle(
          cx,
          cy,
          55,
          'rgba(124,58,237,.28)',
          '#7C3AED',
          4
        );

        circle(
          cx,
          cy,
          27,
          '#E0F2FE',
          '#FFFFFF',
          4
        );

        text(
          '台风眼',
          cx,
          cy,
          11,
          '#075985',
          880,
          'center'
        );

        box(
          72,
          182,
          218,
          120,
          14,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '形成与发展条件',
          90,
          203,
          12,
          '#075985',
          850,
          'left'
        );

        text(
          '较暖海面提供能量',
          90,
          231,
          10,
          '#334155',
          700,
          'left'
        );

        text(
          '充足水汽持续输入',
          90,
          257,
          10,
          '#334155',
          700,
          'left'
        );

        text(
          '水汽凝结释放潜热',
          90,
          283,
          10,
          '#334155',
          700,
          'left'
        );

        box(
          708,
          350,
          226,
          133,
          14,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '登陆后的变化',
          726,
          372,
          12,
          '#B45309',
          850,
          'left'
        );

        text(
          '失去暖海面能量供应',
          726,
          401,
          10,
          '#475569',
          680,
          'left'
        );

        text(
          '陆地摩擦增大',
          726,
          427,
          10,
          '#475569',
          680,
          'left'
        );

        text(
          '山地可能加强局地暴雨',
          726,
          453,
          10,
          '#475569',
          680,
          'left'
        );

        text(
          '整体风速逐渐减弱',
          726,
          479,
          10,
          '#475569',
          680,
          'left'
        );

        if(state.showLabels){
          line(
            cx+24,
            cy-12,
            772,
            215,
            '#0369A1',
            1.5,
            [5,4]
          );

          text(
            '台风眼：相对少云少风',
            782,
            209,
            10,
            '#075985',
            760,
            'left'
          );

          line(
            cx+47,
            cy-40,
            772,
            272,
            '#7C3AED',
            1.5,
            [5,4]
          );

          text(
            '眼墙：上升最强，风雨最剧烈',
            782,
            266,
            10,
            '#7C3AED',
            760,
            'left'
          );

          line(
            cx-138,
            cy+76,
            213,
            494,
            '#0284C7',
            1.5,
            [5,4]
          );

          text(
            '螺旋雨带：阵性强降水和大风',
            70,
            507,
            10,
            '#0284C7',
            760,
            'left'
          );

          arrow(
            230,
            245,
            354,
            302,
            '#0F766E',
            3
          );

          text(
            '近地面空气旋转辐合',
            87,
            236,
            10,
            '#0F766E',
            760,
            'left'
          );
        }

        box(
          72,
          526,
          862,
          28,
          10,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '台风眼不是整个台风的安全区，眼墙和螺旋雨带仍可能带来剧烈风雨。',
          503,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function drawCoast(){
        var sea=
          context.createLinearGradient(
            0,
            130,
            600,
            520
          );

        sea.addColorStop(0,'#BAE6FD');
        sea.addColorStop(1,'#0284C7');

        context.fillStyle=sea;
        context.fillRect(34,112,584,412);

        var land=
          context.createLinearGradient(
            618,
            120,
            966,
            520
          );

        land.addColorStop(0,'#FEF3C7');
        land.addColorStop(.58,'#BBF7D0');
        land.addColorStop(1,'#65A30D');

        context.fillStyle=land;
        context.beginPath();
        context.moveTo(618,112);
        context.lineTo(966,112);
        context.lineTo(966,524);
        context.lineTo(618,524);

        context.bezierCurveTo(
          652,
          465,
          581,
          419,
          626,
          358
        );

        context.bezierCurveTo(
          666,
          299,
          590,
          244,
          628,
          184
        );

        context.bezierCurveTo(
          653,
          146,
          623,
          130,
          618,
          112
        );

        context.closePath();
        context.fill();

        context.strokeStyle='#166534';
        context.lineWidth=3;
        context.beginPath();
        context.moveTo(618,112);

        context.bezierCurveTo(
          653,
          146,
          590,
          244,
          626,
          358
        );

        context.bezierCurveTo(
          665,
          430,
          596,
          480,
          618,
          524
        );

        context.stroke();

        for(
          var mountain=0;
          mountain<4;
          mountain+=1
        ){
          var x=748+mountain*52;
          var y=270+mountain%2*45;

          context.fillStyle='#4D7C0F';
          context.beginPath();
          context.moveTo(x-28,y+40);
          context.lineTo(x,y-16);
          context.lineTo(x+31,y+40);
          context.closePath();
          context.fill();
        }

        box(
          678,
          151,
          126,
          56,
          11,
          '#FFFFFF',
          '#FDBA74'
        );

        text(
          '沿海城市',
          741,
          169,
          11,
          '#C2410C',
          850,
          'center'
        );

        text(
          '人口与资产暴露',
          741,
          192,
          9,
          '#64748B',
          650,
          'center'
        );
      }

      function cycloneSymbol(
        x,
        y,
        radius,
        intensity,
        color
      ){
        circle(
          x,
          y,
          radius,
          'rgba(255,255,255,.20)',
          color,
          2
        );

        for(
          var arm=0;
          arm<4;
          arm+=1
        ){
          spiral(
            x,
            y,
            radius*.18,
            radius*.92,
            arm*Math.PI/2+
            state.phase*Math.PI*2,
            .58,
            '#FFFFFF',
            5+intensity*.25,
            .82
          );
        }

        circle(
          x,
          y,
          radius*.18,
          '#E0F2FE',
          '#FFFFFF',
          2
        );
      }

      function trackView(item,value,derived){
        background(
          '台风移动路径与登陆变化',
          '引导气流影响移动方向，海洋条件和陆地摩擦主要影响强度变化。'
        );

        card(
          28,
          78,
          210,
          '引导气流偏转',
          Math.round(value.steering),
          '#2563EB',
          '负值偏西，正值偏北'
        );

        card(
          250,
          78,
          210,
          '中心气压',
          derived.pressure+' hPa',
          '#DC2626',
          '成熟期附近通常较低'
        );

        card(
          472,
          78,
          210,
          '台风强度指数',
          derived.intensity,
          item.color,
          '海洋支持与陆地削弱'
        );

        card(
          694,
          78,
          258,
          '当前剩余风险',
          derived.residualRisk,
          '#EA580C',
          '风减弱后暴雨仍可持续'
        );

        drawCoast();

        var start={x:126,y:420};
        var control={
          x:420,
          y:308-value.steering*7
        };
        var end={
          x:786,
          y:170-value.steering*5
        };

        context.save();
        context.strokeStyle='#FFFFFF';
        context.lineWidth=8;
        context.lineCap='round';
        context.setLineDash([12,9]);
        context.beginPath();
        context.moveTo(start.x,start.y);

        context.quadraticCurveTo(
          control.x,
          control.y,
          end.x,
          end.y
        );

        context.stroke();

        context.strokeStyle=item.color;
        context.lineWidth=3;
        context.setLineDash([10,8]);
        context.beginPath();
        context.moveTo(start.x,start.y);

        context.quadraticCurveTo(
          control.x,
          control.y,
          end.x,
          end.y
        );

        context.stroke();
        context.restore();

        var ratio=
          clamp(
            value.progress/100,
            0,
            1
          );

        var pathX=
          Math.pow(1-ratio,2)*start.x+
          2*(1-ratio)*ratio*control.x+
          ratio*ratio*end.x;

        var pathY=
          Math.pow(1-ratio,2)*start.y+
          2*(1-ratio)*ratio*control.y+
          ratio*ratio*end.y;

        cycloneSymbol(
          pathX,
          pathY,
          37+derived.intensity*3.2,
          derived.intensity,
          item.color
        );

        var guideY=
          213-value.steering*6;

        arrow(
          142,
          guideY,
          400,
          guideY-34,
          '#1D4ED8',
          4
        );

        arrow(
          400,
          guideY-34,
          602,
          guideY-74,
          '#4F46E5',
          4
        );

        line(
          618,
          146,
          618,
          492,
          '#DC2626',
          2,
          [6,5]
        );

        if(state.showLabels){
          text(
            '大尺度引导气流',
            300,
            guideY-44,
            10,
            '#1D4ED8',
            820,
            'center'
          );

          text(
            '暖海面持续供能',
            252,
            486,
            10,
            '#0369A1',
            820,
            'center'
          );

          text(
            '登陆区',
            630,
            242,
            10,
            '#DC2626',
            850,
            'center'
          );

          text(
            '深入内陆后整体减弱',
            826,
            459,
            10,
            '#B45309',
            820,
            'center'
          );
        }

        box(
          656,
          403,
          264,
          94,
          13,
          '#FFFFFF',
          '#FDBA74'
        );

        text(
          '登陆前后强度变化',
          674,
          424,
          11.5,
          '#B45309',
          850,
          'left'
        );

        text(
          '海上：暖水汽供应充分',
          674,
          451,
          9.7,
          '#475569',
          680,
          'left'
        );

        text(
          '登陆：大风、暴雨和风暴潮叠加',
          674,
          473,
          9.7,
          '#475569',
          680,
          'left'
        );

        text(
          '内陆：风速减弱，暴雨仍可持续',
          674,
          493,
          9.7,
          '#475569',
          680,
          'left'
        );

        box(
          55,
          526,
          890,
          28,
          10,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '台风路径主要受大尺度环境气流引导，不是由台风自身任意选择。',
          500,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function hazardBar(
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
          9.4,
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
          clamp(score/10,0,1),
          10,
          5,
          color,
          null
        );
      }

      function hazardsView(item,value,derived){
        background(
          '台风灾害链与复合风险',
          '大风、暴雨和风暴潮可能同时发生，并受到地形和人口资产暴露影响。'
        );

        card(
          28,
          78,
          210,
          '大风危险',
          derived.intensity,
          '#7C3AED',
          '建筑、交通和电力风险'
        );

        card(
          250,
          78,
          210,
          '暴雨危险',
          derived.rainfall,
          '#2563EB',
          '洪涝和山地灾害风险'
        );

        card(
          472,
          78,
          210,
          '风暴潮危险',
          derived.surge,
          '#DC2626',
          '海水增水和沿海倒灌'
        );

        card(
          694,
          78,
          258,
          '防灾后剩余风险',
          derived.residualRisk,
          item.color,
          '准备不能完全消除风险'
        );

        hazardBar(
          58,
          188,
          410,
          '大风灾害',
          derived.intensity,
          '#7C3AED',
          '吹倒树木、损坏建筑并影响交通和电力'
        );

        hazardBar(
          514,
          188,
          410,
          '暴雨与洪涝',
          derived.rainfall,
          '#2563EB',
          '城市积水、河流洪水、山洪和道路中断'
        );

        hazardBar(
          58,
          306,
          410,
          '风暴潮与海岸风险',
          derived.surge,
          '#DC2626',
          '强风推水、低气压增水并与天文潮叠加'
        );

        hazardBar(
          514,
          306,
          410,
          '山地次生灾害',
          clamp(
            Math.round(
              derived.rainfall*.62+
              value.terrain*.38
            ),
            0,
            10
          ),
          '#EA580C',
          '地形抬升可能加强暴雨并诱发滑坡泥石流'
        );

        box(
          58,
          427,
          866,
          91,
          14,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '复合灾害链',
          80,
          448,
          12,
          '#075985',
          860,
          'left'
        );

        text(
          '台风大风',
          108,
          484,
          10,
          '#7C3AED',
          820,
          'center'
        );

        arrow(165,484,258,484,'#64748B',2);

        text(
          '海水增水',
          314,
          484,
          10,
          '#DC2626',
          820,
          'center'
        );

        arrow(369,484,462,484,'#64748B',2);

        text(
          '沿海倒灌',
          517,
          484,
          10,
          '#C2410C',
          820,
          'center'
        );

        arrow(572,484,665,484,'#64748B',2);

        text(
          '城市积水',
          720,
          484,
          10,
          '#2563EB',
          820,
          'center'
        );

        arrow(775,484,846,484,'#64748B',2);

        text(
          '交通中断',
          884,
          484,
          10,
          '#475569',
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
          '#BAE6FD'
        );

        text(
          '评估台风风险不能只看风力，还要分析降水、风暴潮、地形和人口资产暴露。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function responseCard(
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
          clamp(score/10,0,1),
          9,
          5,
          color,
          null
        );
      }

      function responseView(item,value,derived){
        background(
          '台风防灾避险与风险管理',
          '防灾包括监测预警、人员转移、工程防护、城市排涝和灾后恢复。'
        );

        card(
          28,
          78,
          210,
          '灾前原始风险',
          derived.rawRisk,
          '#DC2626',
          '未考虑防灾准备'
        );

        card(
          250,
          78,
          210,
          '防灾准备水平',
          Math.round(value.preparedness),
          '#16A34A',
          '预警、转移和设施准备'
        );

        card(
          472,
          78,
          210,
          '剩余风险',
          derived.residualRisk,
          '#EA580C',
          '采取措施后仍需警惕'
        );

        card(
          694,
          78,
          258,
          '应对能力指数',
          derived.responseScore,
          item.color,
          '准备水平和风险控制综合'
        );

        responseCard(
          58,
          188,
          410,
          '监测预报与分级预警',
          '利用卫星、雷达、海洋浮标和地面站持续跟踪。',
          '#2563EB',
          value.preparedness
        );

        responseCard(
          514,
          188,
          410,
          '人员转移与避险安置',
          '沿海低洼区、危旧房和山洪风险区优先转移。',
          '#7C3AED',
          value.preparedness*.92
        );

        responseCard(
          58,
          310,
          410,
          '海岸工程与生态缓冲',
          '堤防、防潮闸、红树林和滨海湿地共同减灾。',
          '#0F766E',
          value.preparedness*.78+
          derived.surge*.12
        );

        responseCard(
          514,
          310,
          410,
          '城市排涝与生命线保障',
          '检查排水、电力、通信、交通和应急物资。',
          '#EA580C',
          value.preparedness*.86
        );

        box(
          58,
          432,
          866,
          83,
          14,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '准备水平对比',
          80,
          453,
          12,
          '#075985',
          860,
          'left'
        );

        box(
          103,
          468,
          250,
          30,
          10,
          '#FEE2E2',
          '#FCA5A5'
        );

        text(
          '准备不足：风险 '+derived.rawRisk,
          228,
          483,
          10.5,
          '#B91C1C',
          850,
          'center'
        );

        arrow(
          380,
          483,
          608,
          483,
          '#16A34A',
          4
        );

        box(
          628,
          468,
          250,
          30,
          10,
          '#DCFCE7',
          '#86EFAC'
        );

        text(
          '加强准备：剩余风险 '+derived.residualRisk,
          753,
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
          '#BAE6FD'
        );

        text(
          '预警的价值在于提前行动，风险降低依赖公众响应、工程设施和组织协同。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function update(item,value,derived){
        progressValue.textContent=
          Math.round(value.progress)+'%';

        temperatureValue.textContent=
          Math.round(value.temperature)+'℃';

        steeringValue.textContent=
          String(Math.round(value.steering));

        moistureValue.textContent=
          String(Math.round(value.moisture));

        terrainValue.textContent=
          String(Math.round(value.terrain));

        preparednessValue.textContent=
          String(Math.round(value.preparedness));

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
            ? '自定义台风条件'
            : item.name;

        result.textContent=
          scenarioName+
          '下，台风强度指数为'+
          derived.intensity+
          '，中心气压约'+
          derived.pressure+
          ' hPa，大风危险为'+
          derived.intensity+
          '，暴雨危险为'+
          derived.rainfall+
          '，风暴潮危险为'+
          derived.surge+
          '；当前防灾准备下剩余风险为'+
          derived.residualRisk+
          '。';
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var item=scenarioByKey(state.scenario);
        var value=values();
        var derived=derive(value);

        update(item,value,derived);
        context.clearRect(0,0,width,height);

        if(state.view==='track'){
          trackView(item,value,derived);
        }else if(state.view==='hazards'){
          hazardsView(item,value,derived);
        }else if(state.view==='response'){
          responseView(item,value,derived);
        }else{
          structureView(item,value,derived);
        }
      }

      function applyScenario(key,changeView){
        var item=scenarioByKey(key);

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
          cancelAnimationFrame(state.raf);
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
        var segment=Math.floor(elapsed/duration);
        var local=(elapsed%duration)/duration;

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

        var progressRatio=
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
            'structure',
            'track',
            'hazards',
            'response'
          ][
            Math.floor(
              elapsed/6500
            )%4
          ];

        state.phase=(elapsed/3800)%1;

        setInputs({
          progress:lerp(
            from.progress,
            to.progress,
            progressRatio
          ),
          temperature:lerp(
            from.temperature,
            to.temperature,
            progressRatio
          ),
          steering:lerp(
            from.steering,
            to.steering,
            progressRatio
          ),
          moisture:lerp(
            from.moisture,
            to.moisture,
            progressRatio
          ),
          terrain:lerp(
            from.terrain,
            to.terrain,
            progressRatio
          ),
          preparedness:lerp(
            from.preparedness,
            to.preparedness,
            progressRatio
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
        progressInput,
        temperatureInput,
        steeringInput,
        moistureInput,
        terrainInput,
        preparednessInput
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
                'approaching-coast',
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
                'structure';

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

          state.scenario=initial.scenario;
          state.view='structure';
          state.showLabels=initial.showLabels;

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

export const GEOGRAPHY_LAB_TEMPLATES_DISASTER_TYPHOON:
GeographyLabTemplate[] = [
  {
    id: 'geography-typhoon-structure-track-hazard-impact',
    group: '🌪️ 自然灾害与地理信息技术',
    name: '台风结构、移动路径与灾害影响',
    emoji: '🌀',
    desc: '调节海温、引导气流、水汽、陆地摩擦和防灾准备，观察台风眼、眼墙、螺旋雨带、移动路径、登陆变化以及大风、暴雨和风暴潮风险。',
    params: [
      {
        key: 'scenario',
        label: '初始台风发展情境',
        type: 'select',
        options: [
          {
            label: '暖洋面发展',
            value: 'open-ocean',
          },
          {
            label: '靠近海岸',
            value: 'approaching-coast',
          },
          {
            label: '台风登陆',
            value: 'landfall',
          },
          {
            label: '深入内陆',
            value: 'inland-weakening',
          },
          {
            label: '防灾避险方案',
            value: 'preparedness',
          },
        ],
        defaultValue: 'approaching-coast',
      },
      {
        key: 'progress',
        label: '初始发展过程进度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 48,
        hint: '表示台风从海上发展、靠近海岸、登陆到深入内陆的课堂过程。',
      },
      {
        key: 'seaTemperature',
        label: '海面温度',
        type: 'number',
        min: 24,
        max: 31,
        step: 1,
        defaultValue: 29,
        hint: '较暖海面通常能提供更多水汽和能量，真实强度还受多种条件影响。',
      },
      {
        key: 'steeringFlow',
        label: '引导气流偏转',
        type: 'number',
        min: -10,
        max: 10,
        step: 1,
        defaultValue: -2,
        hint: '负值表示路径更偏西，正值表示路径更偏北，仅用于课堂比较。',
      },
      {
        key: 'moistureSupply',
        label: '水汽供应',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 8,
        hint: '水汽供应影响云系发展和降水强度。',
      },
      {
        key: 'terrainFriction',
        label: '陆地摩擦与地形作用',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '陆地摩擦通常削弱风速，山地抬升可能增强局地降水。',
      },
      {
        key: 'preparedness',
        label: '防灾准备水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '综合表示监测预警、人员转移、工程防护和应急保障能力。',
      },
      {
        key: 'showLabels',
        label: '显示结构与路径标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildTyphoonHTML,
  },
]
