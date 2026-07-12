/**
 * geographyLabTemplatesDisasterRemoteSensingGIS.ts
 *
 * 地理第41批B3：
 *   遥感、GIS与灾害监测分析。
 *
 * 教学目标：
 * - 理解遥感利用不同波段和地物反射差异识别水体、植被、裸地和建设用地；
 * - 比较灾前、灾中和灾后影像，识别淹没、烧毁、滑坡和道路中断等变化；
 * - 理解GIS通过图层叠加综合分析危险性、暴露度和脆弱性；
 * - 认识灾害风险并不等同于危险性，还与人口、资产和承灾能力有关；
 * - 利用风险分区、应急路线和资源配置开展课堂情境决策；
 * - 认识遥感快速发现、GIS综合分析和地面核查之间的互补关系。
 *
 * 教学边界：
 * - 所有影像、波段、地物、道路、人口和风险数据均为课堂简化示意；
 * - 地图、城市、河流、山地和应急设施不对应任何真实地区；
 * - 风险分区和路线规划不用于真实应急指挥、导航或资源配置；
 * - 真实灾害监测与应急决策应使用权威数据并结合现场核查。
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

function buildRemoteSensingGISHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'flood-monitoring',
    'wildfire-monitoring',
    'landslide-monitoring',
    'urban-emergency',
    'integrated-risk',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'flood-monitoring',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'flood-monitoring'

  const spectralContrast = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'spectralContrast', 7),
    ),
  )

  const cloudCover = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'cloudCover', 3),
    ),
  )

  const hazardIntensity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'hazardIntensity', 7),
    ),
  )

  const populationExposure = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'populationExposure', 7),
    ),
  )

  const assetExposure = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'assetExposure', 6),
    ),
  )

  const vulnerability = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'vulnerability', 6),
    ),
  )

  const responseCapacity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'responseCapacity', 6),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-rs-gis-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #C4B5FD;
      border-radius:18px;
      background:#FFFFFF;
      color:#1E1B4B;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(91,33,182,.12);
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
      border-bottom:1px solid #DDD6FE;
      background:linear-gradient(
        135deg,
        #F5F3FF,
        #EFF6FF 52%,
        #ECFEFF
      );
    }

    #${rootId} .gl-title{
      color:#5B21B6;
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
      border:1px solid #C4B5FD;
      border-radius:999px;
      background:#FFFFFF;
      color:#6D28D9;
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
      border-right:1px solid #DDD6FE;
      background:linear-gradient(
        180deg,
        #F5F3FF,
        #EFF6FF 56%,
        #ECFEFF
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
        #EDE9FE 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#5B21B6;
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
      background:#EDE9FE;
      color:#6D28D9;
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
        #93C5FD,
        #A78BFA,
        #67E8F9
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
        #7C3AED,
        #0284C7
      );
      box-shadow:0 1px 5px rgba(91,33,182,.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #C4B5FD;
      border-radius:9px;
      background:#FFFFFF;
      color:#6D28D9;
      font-size:10.5px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#6D28D9;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #7C3AED,
        #2563EB 56%,
        #0891B2
      );
      box-shadow:0 5px 13px rgba(91,33,182,.22);
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
      border:1px solid #C4B5FD;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #F5F3FF,
        #ECFEFF
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
      border:1px solid #DDD6FE;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-rs-gis-canvas{
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
    <div style="font-size:24px;">🛰️</div>

    <div>
      <div class="gl-title">
        遥感、GIS与灾害监测分析
      </div>

      <div class="gl-subtitle">
        比较灾前灾后影像，叠加危险性、暴露度与脆弱性图层，开展风险分区和应急配置
      </div>
    </div>

    <div class="gl-note">
      教学示意数据 · 不用于真实应急指挥
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        灾害监测情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="flood-monitoring">
          洪水监测
        </button>

        <button type="button" data-scenario="wildfire-monitoring">
          森林火灾
        </button>

        <button type="button" data-scenario="landslide-monitoring">
          滑坡监测
        </button>

        <button type="button" data-scenario="urban-emergency">
          城市应急
        </button>

        <button type="button" data-scenario="integrated-risk">
          综合风险分析
        </button>
      </div>

      <div class="gl-section-title">
        观测与风险参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">地物光谱差异</span>
          <span class="gl-value" data-role="spectral-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${spectralContrast}"
          data-role="spectral"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">云层遮挡程度</span>
          <span class="gl-value" data-role="cloud-value">3</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${cloudCover}"
          data-role="cloud"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">灾害危险强度</span>
          <span class="gl-value" data-role="hazard-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${hazardIntensity}"
          data-role="hazard"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">人口暴露度</span>
          <span class="gl-value" data-role="population-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${populationExposure}"
          data-role="population"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">资产暴露度</span>
          <span class="gl-value" data-role="asset-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${assetExposure}"
          data-role="asset"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">承灾体脆弱性</span>
          <span class="gl-value" data-role="vulnerability-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${vulnerability}"
          data-role="vulnerability"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">应急响应能力</span>
          <span class="gl-value" data-role="response-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${responseCapacity}"
          data-role="response"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          图层与地物标注
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
        遥感可快速识别灾害变化，GIS可叠加危险性、暴露度与脆弱性，形成风险分区并辅助应急分析。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="imagery">
          遥感识别
        </button>

        <button type="button" data-view="change">
          变化监测
        </button>

        <button type="button" data-view="overlay">
          GIS图层叠加
        </button>

        <button type="button" data-view="emergency">
          应急路线与资源
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-rs-gis-canvas"
          width="1000"
          height="570"
          data-role="canvas"
          aria-label="遥感GIS灾害监测风险分区与应急分析教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var spectralInput =
        root.querySelector('[data-role="spectral"]');

      var cloudInput =
        root.querySelector('[data-role="cloud"]');

      var hazardInput =
        root.querySelector('[data-role="hazard"]');

      var populationInput =
        root.querySelector('[data-role="population"]');

      var assetInput =
        root.querySelector('[data-role="asset"]');

      var vulnerabilityInput =
        root.querySelector('[data-role="vulnerability"]');

      var responseInput =
        root.querySelector('[data-role="response"]');

      var spectralValue =
        root.querySelector('[data-role="spectral-value"]');

      var cloudValue =
        root.querySelector('[data-role="cloud-value"]');

      var hazardValue =
        root.querySelector('[data-role="hazard-value"]');

      var populationValue =
        root.querySelector('[data-role="population-value"]');

      var assetValue =
        root.querySelector('[data-role="asset-value"]');

      var vulnerabilityValue =
        root.querySelector('[data-role="vulnerability-value"]');

      var responseValue =
        root.querySelector('[data-role="response-value"]');

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
        !spectralInput ||
        !cloudInput ||
        !hazardInput ||
        !populationInput ||
        !assetInput ||
        !vulnerabilityInput ||
        !responseInput ||
        !spectralValue ||
        !cloudValue ||
        !hazardValue ||
        !populationValue ||
        !assetValue ||
        !vulnerabilityValue ||
        !responseValue ||
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
          key:'flood-monitoring',
          name:'洪水监测',
          spectral:8,
          cloud:5,
          hazard:8,
          population:7,
          asset:7,
          vulnerability:6,
          response:6,
          view:'imagery',
          color:'#2563EB'
        },
        {
          key:'wildfire-monitoring',
          name:'森林火灾监测',
          spectral:9,
          cloud:3,
          hazard:8,
          population:4,
          asset:4,
          vulnerability:7,
          response:5,
          view:'change',
          color:'#DC2626'
        },
        {
          key:'landslide-monitoring',
          name:'滑坡监测',
          spectral:7,
          cloud:6,
          hazard:7,
          population:5,
          asset:5,
          vulnerability:8,
          response:5,
          view:'change',
          color:'#D97706'
        },
        {
          key:'urban-emergency',
          name:'城市应急',
          spectral:6,
          cloud:4,
          hazard:7,
          population:10,
          asset:9,
          vulnerability:6,
          response:8,
          view:'emergency',
          color:'#7C3AED'
        },
        {
          key:'integrated-risk',
          name:'综合风险分析',
          spectral:8,
          cloud:3,
          hazard:8,
          population:8,
          asset:8,
          vulnerability:7,
          response:8,
          view:'overlay',
          color:'#16A34A'
        }
      ];

      var initial={
        scenario:'${scenario}',
        spectral:${spectralContrast},
        cloud:${cloudCover},
        hazard:${hazardIntensity},
        population:${populationExposure},
        asset:${assetExposure},
        vulnerability:${vulnerability},
        response:${responseCapacity},
        showLabels:${showLabels}
      };

      var state={
        scenario:initial.scenario,
        view:'imagery',
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
          spectral:clamp(
            Number(spectralInput.value) || 0,
            0,
            10
          ),
          cloud:clamp(
            Number(cloudInput.value) || 0,
            0,
            10
          ),
          hazard:clamp(
            Number(hazardInput.value) || 0,
            0,
            10
          ),
          population:clamp(
            Number(populationInput.value) || 0,
            0,
            10
          ),
          asset:clamp(
            Number(assetInput.value) || 0,
            0,
            10
          ),
          vulnerability:clamp(
            Number(vulnerabilityInput.value) || 0,
            0,
            10
          ),
          response:clamp(
            Number(responseInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        spectralInput.value=
          String(Math.round(value.spectral));

        cloudInput.value=
          String(Math.round(value.cloud));

        hazardInput.value=
          String(Math.round(value.hazard));

        populationInput.value=
          String(Math.round(value.population));

        assetInput.value=
          String(Math.round(value.asset));

        vulnerabilityInput.value=
          String(Math.round(value.vulnerability));

        responseInput.value=
          String(Math.round(value.response));
      }

      function derive(value){
        var observationQuality=
          clamp(
            Math.round(
              value.spectral*.72+
              (
                10-value.cloud
              )*.28
            ),
            0,
            10
          );

        var changeDetection=
          clamp(
            Math.round(
              value.spectral*.50+
              value.hazard*.32+
              (
                10-value.cloud
              )*.18
            ),
            0,
            10
          );

        var exposure=
          clamp(
            Math.round(
              value.population*.55+
              value.asset*.45
            ),
            0,
            10
          );

        var rawRisk=
          clamp(
            Math.round(
              value.hazard*.42+
              exposure*.31+
              value.vulnerability*.27
            ),
            0,
            10
          );

        var residualRisk=
          clamp(
            Math.round(
              rawRisk-
              value.response*.44
            ),
            0,
            10
          );

        var routeAvailability=
          clamp(
            Math.round(
              value.response*.46+
              (
                10-value.hazard
              )*.22+
              (
                10-value.vulnerability
              )*.16+
              observationQuality*.16
            ),
            0,
            10
          );

        var resourceEfficiency=
          clamp(
            Math.round(
              value.response*.42+
              observationQuality*.24+
              changeDetection*.20+
              (
                10-residualRisk
              )*.14
            ),
            0,
            10
          );

        var analysisConfidence=
          clamp(
            Math.round(
              observationQuality*.50+
              changeDetection*.30+
              value.response*.20
            ),
            0,
            10
          );

        return {
          observationQuality:observationQuality,
          changeDetection:changeDetection,
          exposure:exposure,
          rawRisk:rawRisk,
          residualRisk:residualRisk,
          routeAvailability:routeAvailability,
          resourceEfficiency:resourceEfficiency,
          analysisConfidence:analysisConfidence
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
        gradient.addColorStop(.56,'#F5F3FF');
        gradient.addColorStop(1,'#CFFAFE');

        context.fillStyle=gradient;
        context.fillRect(0,0,width,height);

        text(
          titleValue,
          28,
          31,
          18,
          '#5B21B6',
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
          '#DDD6FE'
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

      function drawBaseScene(
        x,
        y,
        w,
        h,
        mode,
        stage,
        value
      ){
        context.save();

        context.beginPath();
        context.rect(x,y,w,h);
        context.clip();

        var base=
          context.createLinearGradient(
            x,
            y,
            x+w,
            y+h
          );

        if(mode==='wildfire-monitoring'){
          base.addColorStop(0,'#166534');
          base.addColorStop(1,'#84CC16');
        }else if(mode==='landslide-monitoring'){
          base.addColorStop(0,'#A16207');
          base.addColorStop(1,'#65A30D');
        }else if(mode==='urban-emergency'){
          base.addColorStop(0,'#94A3B8');
          base.addColorStop(1,'#475569');
        }else{
          base.addColorStop(0,'#86EFAC');
          base.addColorStop(1,'#15803D');
        }

        context.fillStyle=base;
        context.fillRect(x,y,w,h);

        context.fillStyle='#2563EB';
        context.beginPath();
        context.moveTo(x-20,y+h*.72);
        context.bezierCurveTo(
          x+w*.20,
          y+h*.48,
          x+w*.39,
          y+h*.86,
          x+w*.58,
          y+h*.60
        );
        context.bezierCurveTo(
          x+w*.73,
          y+h*.41,
          x+w*.85,
          y+h*.66,
          x+w+30,
          y+h*.45
        );
        context.lineTo(x+w+30,y+h);
        context.lineTo(x-20,y+h);
        context.closePath();
        context.fill();

        context.strokeStyle='#E2E8F0';
        context.lineWidth=8;

        context.beginPath();
        context.moveTo(x+w*.08,y+h*.16);
        context.lineTo(x+w*.88,y+h*.84);
        context.stroke();

        context.strokeStyle='#64748B';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(x+w*.08,y+h*.16);
        context.lineTo(x+w*.88,y+h*.84);
        context.stroke();

        for(
          var block=0;
          block<8;
          block+=1
        ){
          var bx=
            x+w*.56+
            (
              block%4
            )*
            w*.085;

          var by=
            y+h*.13+
            Math.floor(block/4)*
            h*.16;

          context.fillStyle=
            block%2===0
              ? '#CBD5E1'
              : '#E2E8F0';

          context.fillRect(
            bx,
            by,
            w*.06,
            h*.095
          );
        }

        if(stage==='after'){
          context.globalAlpha=
            .30+
            value.hazard*.045;

          if(mode==='wildfire-monitoring'){
            context.fillStyle='#7F1D1D';
            context.beginPath();
            context.ellipse(
              x+w*.35,
              y+h*.38,
              w*.22,
              h*.19,
              -.2,
              0,
              Math.PI*2
            );
            context.fill();
          }else if(mode==='landslide-monitoring'){
            context.fillStyle='#92400E';
            context.beginPath();
            context.moveTo(x+w*.15,y+h*.12);
            context.lineTo(x+w*.50,y+h*.54);
            context.lineTo(x+w*.23,y+h*.74);
            context.closePath();
            context.fill();
          }else{
            context.fillStyle='#1D4ED8';
            context.beginPath();
            context.ellipse(
              x+w*.42,
              y+h*.65,
              w*.30,
              h*.23,
              .1,
              0,
              Math.PI*2
            );
            context.fill();
          }

          context.globalAlpha=1;
        }

        var cloudCount=
          Math.round(value.cloud/2);

        for(
          var cloud=0;
          cloud<cloudCount;
          cloud+=1
        ){
          var cloudX=
            x+
            40+
            (
              cloud*91
            )%
            Math.max(
              80,
              w-90
            );

          var cloudY=
            y+
            32+
            (
              cloud%3
            )*
            44;

          context.fillStyle='rgba(255,255,255,.72)';
          circle(
            cloudX,
            cloudY,
            28,
            'rgba(255,255,255,.72)',
            null,
            0
          );

          circle(
            cloudX+23,
            cloudY-9,
            24,
            'rgba(255,255,255,.72)',
            null,
            0
          );

          circle(
            cloudX+48,
            cloudY,
            27,
            'rgba(255,255,255,.72)',
            null,
            0
          );
        }

        context.restore();

        context.strokeStyle='#FFFFFF';
        context.lineWidth=3;
        context.strokeRect(x,y,w,h);
      }

      function legendItem(
        x,
        y,
        color,
        label
      ){
        context.fillStyle=color;
        context.fillRect(x,y-7,17,14);

        text(
          label,
          x+25,
          y,
          9.5,
          '#475569',
          700,
          'left'
        );
      }

      function imageryView(
        item,
        value,
        derived
      ){
        background(
          '遥感波段组合与地物识别',
          '不同地物在不同波段中的反射特征不同，合理的波段组合有助于突出水体、植被和灾害区域。'
        );

        card(
          28,
          78,
          210,
          '地物光谱差异',
          Math.round(value.spectral),
          '#7C3AED',
          '差异越大越易分类'
        );

        card(
          250,
          78,
          210,
          '云层遮挡',
          Math.round(value.cloud),
          '#64748B',
          '遮挡会降低有效观测'
        );

        card(
          472,
          78,
          210,
          '观测质量',
          derived.observationQuality,
          '#0284C7',
          '光谱差异和云量综合'
        );

        card(
          694,
          78,
          258,
          '变化识别能力',
          derived.changeDetection,
          item.color,
          '观测质量与灾害变化综合'
        );

        var panelY=180;
        var panelW=282;
        var panelH=286;

        drawBaseScene(
          45,
          panelY,
          panelW,
          panelH,
          state.scenario,
          'before',
          value
        );

        drawBaseScene(
          359,
          panelY,
          panelW,
          panelH,
          state.scenario,
          'after',
          value
        );

        drawBaseScene(
          673,
          panelY,
          panelW,
          panelH,
          state.scenario,
          'after',
          {
            cloud:value.cloud*.35,
            hazard:value.hazard
          }
        );

        text(
          '自然色合成',
          186,
          panelY-17,
          12,
          '#334155',
          850,
          'center'
        );

        text(
          '近红外假彩色',
          500,
          panelY-17,
          12,
          '#7C3AED',
          850,
          'center'
        );

        text(
          '水体或灾害增强',
          814,
          panelY-17,
          12,
          '#0284C7',
          850,
          'center'
        );

        context.save();
        context.globalAlpha=
          .18+
          value.spectral*.055;

        context.fillStyle='#DC2626';
        context.fillRect(
          359,
          panelY,
          panelW,
          panelH
        );

        context.restore();

        context.save();
        context.globalAlpha=
          .12+
          value.spectral*.060;

        context.fillStyle=
          state.scenario==='wildfire-monitoring'
            ? '#F97316'
            : state.scenario==='landslide-monitoring'
              ? '#A16207'
              : '#0EA5E9';

        context.fillRect(
          673,
          panelY,
          panelW,
          panelH
        );

        context.restore();

        box(
          45,
          481,
          910,
          37,
          11,
          '#FFFFFF',
          '#DDD6FE'
        );

        legendItem(
          70,
          499,
          '#2563EB',
          '水体'
        );

        legendItem(
          215,
          499,
          '#16A34A',
          '健康植被'
        );

        legendItem(
          381,
          499,
          '#A16207',
          '裸地或滑坡'
        );

        legendItem(
          567,
          499,
          '#DC2626',
          '火烧或受损区'
        );

        legendItem(
          761,
          499,
          '#94A3B8',
          '建设用地'
        );

        if(state.showLabels){
          text(
            '遥感影像',
            188,
            326,
            11,
            '#FFFFFF',
            880,
            'center'
          );

          text(
            '波段组合',
            500,
            326,
            11,
            '#FFFFFF',
            880,
            'center'
          );

          text(
            '专题信息',
            814,
            326,
            11,
            '#FFFFFF',
            880,
            'center'
          );

          arrow(
            330,
            326,
            352,
            326,
            '#7C3AED',
            3
          );

          arrow(
            644,
            326,
            666,
            326,
            '#0284C7',
            3
          );
        }

        box(
          45,
          526,
          910,
          28,
          10,
          '#FFFFFF',
          '#DDD6FE'
        );

        text(
          '遥感识别依赖地物光谱差异，但云层遮挡、分辨率和地物混合会造成不确定性。',
          500,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function changeView(
        item,
        value,
        derived
      ){
        background(
          '灾前、灾中与灾后变化监测',
          '将不同时相影像配准后比较，可识别受灾范围、道路中断和恢复进度。'
        );

        card(
          28,
          78,
          210,
          '观测质量',
          derived.observationQuality,
          '#0284C7',
          '云量和光谱差异综合'
        );

        card(
          250,
          78,
          210,
          '灾害变化强度',
          Math.round(value.hazard),
          '#DC2626',
          '受灾范围与程度'
        );

        card(
          472,
          78,
          210,
          '变化检测能力',
          derived.changeDetection,
          '#7C3AED',
          '多时相比较有效性'
        );

        card(
          694,
          78,
          258,
          '分析可信度',
          derived.analysisConfidence,
          item.color,
          '遥感、GIS与响应能力综合'
        );

        var panelY=182;
        var panelW=272;
        var panelH=270;

        drawBaseScene(
          55,
          panelY,
          panelW,
          panelH,
          state.scenario,
          'before',
          value
        );

        drawBaseScene(
          364,
          panelY,
          panelW,
          panelH,
          state.scenario,
          'after',
          value
        );

        context.fillStyle='#0F172A';
        context.fillRect(
          673,
          panelY,
          panelW,
          panelH
        );

        var grid=9;
        var cellW=panelW/grid;
        var cellH=panelH/grid;

        for(
          var row=0;
          row<grid;
          row+=1
        ){
          for(
            var col=0;
            col<grid;
            col+=1
          ){
            var distance=
              Math.hypot(
                col-4.3,
                row-4.9
              );

            var affected=
              distance<
              1.4+
              value.hazard*.23+
              Math.sin(
                row*1.4+
                col*.8
              )*.45;

            context.fillStyle=
              affected
                ? item.color
                : '#1E293B';

            context.globalAlpha=
              affected
                ? .42+
                  value.hazard*.045
                : .72;

            context.fillRect(
              673+col*cellW+1,
              panelY+row*cellH+1,
              cellW-2,
              cellH-2
            );
          }
        }

        context.globalAlpha=1;

        text(
          '灾前影像',
          191,
          panelY-17,
          12,
          '#334155',
          850,
          'center'
        );

        text(
          '灾中或灾后影像',
          500,
          panelY-17,
          12,
          '#DC2626',
          850,
          'center'
        );

        text(
          '变化检测结果',
          809,
          panelY-17,
          12,
          '#7C3AED',
          850,
          'center'
        );

        arrow(
          331,
          316,
          356,
          316,
          '#64748B',
          3
        );

        arrow(
          640,
          316,
          665,
          316,
          '#64748B',
          3
        );

        box(
          55,
          470,
          890,
          48,
          11,
          '#FFFFFF',
          '#DDD6FE'
        );

        var changeType=
          state.scenario==='wildfire-monitoring'
            ? '植被减少与烧毁斑块'
            : state.scenario==='landslide-monitoring'
              ? '坡面裸露与道路中断'
              : state.scenario==='urban-emergency'
                ? '积水区与交通受阻'
                : '水体扩张与淹没范围';

        text(
          '主要变化：'+changeType,
          78,
          487,
          10.5,
          item.color,
          850,
          'left'
        );

        text(
          '建议核查：云影、季节变化、影像配准误差和现场实际情况',
          78,
          505,
          9.5,
          '#64748B',
          680,
          'left'
        );

        if(state.showLabels){
          text(
            '同一区域',
            344,
            290,
            9.5,
            '#64748B',
            780,
            'center'
          );

          text(
            '相减或分类比较',
            653,
            290,
            9.5,
            '#64748B',
            780,
            'center'
          );
        }

        box(
          55,
          526,
          890,
          28,
          10,
          '#FFFFFF',
          '#DDD6FE'
        );

        text(
          '变化检测能快速圈定疑似受灾区，但仍需要结合地面核查排除误判。',
          500,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function mapCell(
        x,
        y,
        size,
        risk,
        color
      ){
        context.save();

        context.globalAlpha=
          .20+
          risk*.075;

        context.fillStyle=color;
        context.fillRect(
          x+1,
          y+1,
          size-2,
          size-2
        );

        context.restore();

        context.strokeStyle='rgba(255,255,255,.65)';
        context.lineWidth=1;
        context.strokeRect(x,y,size,size);
      }

      function layerRow(
        x,
        y,
        label,
        color,
        value
      ){
        box(
          x,
          y,
          245,
          43,
          10,
          '#FFFFFF',
          '#DDD6FE'
        );

        context.fillStyle=color;
        context.fillRect(
          x+12,
          y+12,
          19,
          19
        );

        text(
          label,
          x+42,
          y+17,
          10,
          '#334155',
          790,
          'left'
        );

        box(
          x+42,
          y+28,
          177,
          7,
          4,
          '#E2E8F0',
          null
        );

        box(
          x+42,
          y+28,
          177*
          value/10,
          7,
          4,
          color,
          null
        );
      }

      function overlayView(
        item,
        value,
        derived
      ){
        background(
          'GIS图层叠加与灾害风险分区',
          '危险性、暴露度和脆弱性图层叠加后形成综合风险，响应能力可降低剩余风险。'
        );

        card(
          28,
          78,
          210,
          '危险性',
          Math.round(value.hazard),
          '#DC2626',
          '灾害发生强度和范围'
        );

        card(
          250,
          78,
          210,
          '综合暴露度',
          derived.exposure,
          '#2563EB',
          '人口与资产暴露'
        );

        card(
          472,
          78,
          210,
          '脆弱性',
          Math.round(value.vulnerability),
          '#D97706',
          '承灾体易损程度'
        );

        card(
          694,
          78,
          258,
          '剩余风险',
          derived.residualRisk,
          item.color,
          '考虑应急响应后的风险'
        );

        layerRow(
          55,
          184,
          '危险性图层',
          '#DC2626',
          value.hazard
        );

        layerRow(
          55,
          236,
          '人口暴露图层',
          '#2563EB',
          value.population
        );

        layerRow(
          55,
          288,
          '资产暴露图层',
          '#7C3AED',
          value.asset
        );

        layerRow(
          55,
          340,
          '脆弱性图层',
          '#D97706',
          value.vulnerability
        );

        layerRow(
          55,
          392,
          '应急能力图层',
          '#16A34A',
          value.response
        );

        var mapX=348;
        var mapY=181;
        var mapSize=337;
        var grid=9;
        var cell=mapSize/grid;

        box(
          mapX-6,
          mapY-6,
          mapSize+12,
          mapSize+12,
          13,
          '#FFFFFF',
          '#DDD6FE'
        );

        for(
          var row=0;
          row<grid;
          row+=1
        ){
          for(
            var col=0;
            col<grid;
            col+=1
          ){
            var hazardFactor=
              clamp(
                value.hazard+
                Math.sin(
                  row*.8+
                  col*.55
                )*
                2.2+
                (
                  4.5-col
                )*
                .18,
                0,
                10
              );

            var exposureFactor=
              clamp(
                derived.exposure+
                (
                  col>4 &&
                  row<6
                    ? 2.2
                    : -1.2
                ),
                0,
                10
              );

            var vulnerabilityFactor=
              clamp(
                value.vulnerability+
                (
                  row>5
                    ? 1.5
                    : -.8
                ),
                0,
                10
              );

            var risk=
              clamp(
                hazardFactor*.42+
                exposureFactor*.31+
                vulnerabilityFactor*.27-
                value.response*.25,
                0,
                10
              );

            var color=
              risk>=7
                ? '#DC2626'
                : risk>=4
                  ? '#F59E0B'
                  : '#16A34A';

            mapCell(
              mapX+col*cell,
              mapY+row*cell,
              cell,
              risk,
              color
            );
          }
        }

        context.strokeStyle='#2563EB';
        context.lineWidth=5;
        context.beginPath();
        context.moveTo(
          mapX,
          mapY+mapSize*.72
        );

        context.bezierCurveTo(
          mapX+mapSize*.24,
          mapY+mapSize*.50,
          mapX+mapSize*.48,
          mapY+mapSize*.86,
          mapX+mapSize,
          mapY+mapSize*.43
        );

        context.stroke();

        context.strokeStyle='#475569';
        context.lineWidth=4;
        context.beginPath();
        context.moveTo(
          mapX+mapSize*.08,
          mapY+mapSize*.14
        );
        context.lineTo(
          mapX+mapSize*.88,
          mapY+mapSize*.86
        );
        context.stroke();

        var panelX=725;

        box(
          panelX,
          181,
          220,
          337,
          13,
          '#FFFFFF',
          '#DDD6FE'
        );

        text(
          '综合风险分区',
          panelX+18,
          205,
          12,
          '#5B21B6',
          860,
          'left'
        );

        legendItem(
          panelX+20,
          239,
          '#DC2626',
          '高风险区'
        );

        legendItem(
          panelX+20,
          272,
          '#F59E0B',
          '中风险区'
        );

        legendItem(
          panelX+20,
          305,
          '#16A34A',
          '低风险区'
        );

        line(
          panelX+20,
          344,
          panelX+200,
          344,
          '#E2E8F0',
          1
        );

        text(
          '风险计算逻辑',
          panelX+20,
          369,
          10.5,
          '#334155',
          820,
          'left'
        );

        text(
          '危险性',
          panelX+20,
          400,
          9.5,
          '#DC2626',
          760,
          'left'
        );

        text(
          '＋ 暴露度',
          panelX+20,
          426,
          9.5,
          '#2563EB',
          760,
          'left'
        );

        text(
          '＋ 脆弱性',
          panelX+20,
          452,
          9.5,
          '#D97706',
          760,
          'left'
        );

        text(
          '－ 响应能力',
          panelX+20,
          478,
          9.5,
          '#16A34A',
          760,
          'left'
        );

        text(
          '＝ 剩余风险 '+derived.residualRisk,
          panelX+20,
          504,
          10.5,
          item.color,
          880,
          'left'
        );

        if(state.showLabels){
          text(
            '图层叠加',
            324,
            360,
            10,
            '#7C3AED',
            850,
            'center'
          );

          arrow(
            304,
            360,
            340,
            360,
            '#7C3AED',
            3
          );

          text(
            '风险栅格',
            516,
            532,
            9.5,
            '#475569',
            760,
            'center'
          );
        }

        box(
          55,
          526,
          890,
          28,
          10,
          '#FFFFFF',
          '#DDD6FE'
        );

        text(
          '危险性相同的地区，因人口、资产和脆弱性不同，最终风险等级也可能不同。',
          500,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function emergencyNode(
        x,
        y,
        label,
        color,
        icon
      ){
        circle(
          x,
          y,
          24,
          '#FFFFFF',
          color,
          4
        );

        text(
          icon,
          x,
          y-1,
          15,
          color,
          850,
          'center'
        );

        text(
          label,
          x,
          y+36,
          9.5,
          color,
          820,
          'center'
        );
      }

      function emergencyView(
        item,
        value,
        derived
      ){
        background(
          '应急路线选择与救援资源配置',
          '路线规划需要避开高风险区和中断路段，资源配置需要优先覆盖高风险、高暴露和高脆弱地区。'
        );

        card(
          28,
          78,
          210,
          '剩余风险',
          derived.residualRisk,
          '#DC2626',
          '考虑应急能力后的风险'
        );

        card(
          250,
          78,
          210,
          '可用路线指数',
          derived.routeAvailability,
          '#2563EB',
          '道路安全与通达性'
        );

        card(
          472,
          78,
          210,
          '资源配置效率',
          derived.resourceEfficiency,
          '#16A34A',
          '监测、分析和响应综合'
        );

        card(
          694,
          78,
          258,
          '分析可信度',
          derived.analysisConfidence,
          item.color,
          '数据质量和响应能力综合'
        );

        var mapX=52;
        var mapY=177;
        var mapW=620;
        var mapH=341;

        box(
          mapX,
          mapY,
          mapW,
          mapH,
          14,
          '#F8FAFC',
          '#DDD6FE'
        );

        context.fillStyle='#DCFCE7';
        context.fillRect(
          mapX+12,
          mapY+12,
          mapW-24,
          mapH-24
        );

        for(
          var road=0;
          road<5;
          road+=1
        ){
          var roadY=
            mapY+
            52+
            road*
            57;

          line(
            mapX+22,
            roadY,
            mapX+mapW-22,
            roadY,
            '#94A3B8',
            7
          );

          line(
            mapX+22,
            roadY,
            mapX+mapW-22,
            roadY,
            '#FFFFFF',
            2,
            [15,10]
          );
        }

        for(
          var cross=0;
          cross<5;
          cross+=1
        ){
          var roadX=
            mapX+
            60+
            cross*
            115;

          line(
            roadX,
            mapY+22,
            roadX,
            mapY+mapH-22,
            '#94A3B8',
            7
          );

          line(
            roadX,
            mapY+22,
            roadX,
            mapY+mapH-22,
            '#FFFFFF',
            2,
            [15,10]
          );
        }

        var hazardRadius=
          48+
          value.hazard*7;

        context.save();
        context.globalAlpha=
          .22+
          value.hazard*.045;

        circle(
          mapX+330,
          mapY+176,
          hazardRadius,
          '#DC2626',
          null,
          0
        );

        context.restore();

        context.save();
        context.globalAlpha=
          .18+
          value.vulnerability*.035;

        circle(
          mapX+184,
          mapY+242,
          45+
          value.vulnerability*4,
          '#F59E0B',
          null,
          0
        );

        context.restore();

        emergencyNode(
          mapX+72,
          mapY+62,
          '避难场所',
          '#16A34A',
          '安'
        );

        emergencyNode(
          mapX+534,
          mapY+72,
          '医疗点',
          '#DC2626',
          '医'
        );

        emergencyNode(
          mapX+533,
          mapY+286,
          '物资点',
          '#7C3AED',
          '物'
        );

        emergencyNode(
          mapX+105,
          mapY+284,
          '受灾社区',
          '#D97706',
          '人'
        );

        var safeColor=
          derived.routeAvailability>=6
            ? '#16A34A'
            : '#F59E0B';

        context.save();
        context.strokeStyle=safeColor;
        context.lineWidth=7;
        context.lineCap='round';
        context.lineJoin='round';

        context.beginPath();
        context.moveTo(
          mapX+105,
          mapY+260
        );

        context.lineTo(
          mapX+175,
          mapY+222
        );

        context.lineTo(
          mapX+175,
          mapY+119
        );

        context.lineTo(
          mapX+72,
          mapY+86
        );

        context.stroke();
        context.restore();

        context.save();
        context.strokeStyle='#DC2626';
        context.lineWidth=4;
        context.setLineDash([10,7]);

        context.beginPath();
        context.moveTo(
          mapX+105,
          mapY+260
        );

        context.lineTo(
          mapX+330,
          mapY+176
        );

        context.lineTo(
          mapX+534,
          mapY+96
        );

        context.stroke();
        context.restore();

        text(
          '推荐避险路线',
          mapX+144,
          mapY+154,
          10,
          safeColor,
          850,
          'center'
        );

        text(
          '穿越高风险区的不可取路线',
          mapX+398,
          mapY+145,
          9.5,
          '#DC2626',
          820,
          'center'
        );

        var panelX=703;

        box(
          panelX,
          mapY,
          244,
          mapH,
          14,
          '#FFFFFF',
          '#DDD6FE'
        );

        text(
          '资源优先级',
          panelX+18,
          mapY+26,
          12,
          '#5B21B6',
          860,
          'left'
        );

        var resources=[
          {
            name:'人员搜救',
            score:clamp(
              value.population*.50+
              value.vulnerability*.30+
              value.hazard*.20,
              0,
              10
            ),
            color:'#DC2626'
          },
          {
            name:'医疗救护',
            score:clamp(
              value.population*.38+
              value.vulnerability*.42+
              value.hazard*.20,
              0,
              10
            ),
            color:'#7C3AED'
          },
          {
            name:'食品与饮水',
            score:clamp(
              value.population*.48+
              value.asset*.12+
              value.vulnerability*.24+
              value.hazard*.16,
              0,
              10
            ),
            color:'#0284C7'
          },
          {
            name:'道路抢通',
            score:clamp(
              value.asset*.36+
              value.hazard*.34+
              value.response*.30,
              0,
              10
            ),
            color:'#D97706'
          },
          {
            name:'通信保障',
            score:clamp(
              value.asset*.28+
              value.population*.22+
              value.response*.50,
              0,
              10
            ),
            color:'#16A34A'
          }
        ];

        resources.forEach(
          function(resource,index){
            var y=
              mapY+
              63+
              index*
              53;

            text(
              resource.name,
              panelX+18,
              y,
              9.8,
              '#334155',
              760,
              'left'
            );

            text(
              resource.score.toFixed(1),
              panelX+222,
              y,
              9.8,
              resource.color,
              850,
              'right'
            );

            box(
              panelX+18,
              y+14,
              204,
              8,
              4,
              '#E2E8F0',
              null
            );

            box(
              panelX+18,
              y+14,
              204*
              resource.score/10,
              8,
              4,
              resource.color,
              null
            );
          }
        );

        if(state.showLabels){
          text(
            '高危险区',
            mapX+330,
            mapY+176,
            10,
            '#B91C1C',
            880,
            'center'
          );

          text(
            '高脆弱区',
            mapX+184,
            mapY+242,
            9.5,
            '#B45309',
            850,
            'center'
          );
        }

        box(
          52,
          526,
          895,
          28,
          10,
          '#FFFFFF',
          '#DDD6FE'
        );

        text(
          '应急路线应避开高风险区，资源配置应优先服务高暴露和高脆弱群体。',
          500,
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
        spectralValue.textContent=
          String(Math.round(value.spectral));

        cloudValue.textContent=
          String(Math.round(value.cloud));

        hazardValue.textContent=
          String(Math.round(value.hazard));

        populationValue.textContent=
          String(Math.round(value.population));

        assetValue.textContent=
          String(Math.round(value.asset));

        vulnerabilityValue.textContent=
          String(Math.round(value.vulnerability));

        responseValue.textContent=
          String(Math.round(value.response));

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
            ? '自定义灾害监测条件'
            : item.name;

        result.textContent=
          scenarioName+
          '下，遥感观测质量为'+
          derived.observationQuality+
          '，变化检测能力为'+
          derived.changeDetection+
          '，综合暴露度为'+
          derived.exposure+
          '，原始风险为'+
          derived.rawRisk+
          '；考虑应急响应后剩余风险为'+
          derived.residualRisk+
          '，可用路线指数为'+
          derived.routeAvailability+
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

        if(state.view==='change'){
          changeView(
            item,
            value,
            derived
          );
        }else if(state.view==='overlay'){
          overlayView(
            item,
            value,
            derived
          );
        }else if(state.view==='emergency'){
          emergencyView(
            item,
            value,
            derived
          );
        }else{
          imageryView(
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
            'imagery',
            'change',
            'overlay',
            'emergency'
          ][
            Math.floor(
              elapsed/6700
            )%4
          ];

        state.phase=
          (
            elapsed/3600
          )%1;

        setInputs({
          spectral:lerp(
            from.spectral,
            to.spectral,
            progress
          ),
          cloud:lerp(
            from.cloud,
            to.cloud,
            progress
          ),
          hazard:lerp(
            from.hazard,
            to.hazard,
            progress
          ),
          population:lerp(
            from.population,
            to.population,
            progress
          ),
          asset:lerp(
            from.asset,
            to.asset,
            progress
          ),
          vulnerability:lerp(
            from.vulnerability,
            to.vulnerability,
            progress
          ),
          response:lerp(
            from.response,
            to.response,
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
        spectralInput,
        cloudInput,
        hazardInput,
        populationInput,
        assetInput,
        vulnerabilityInput,
        responseInput
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
                'flood-monitoring',
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
                'imagery';

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

          state.view='imagery';

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

export const GEOGRAPHY_LAB_TEMPLATES_DISASTER_REMOTE_SENSING_GIS:
GeographyLabTemplate[] = [
  {
    id: 'geography-remote-sensing-gis-disaster-monitoring-risk-zoning',
    group: '🌪️ 自然灾害与地理信息技术',
    name: '遥感、GIS与灾害监测分析',
    emoji: '🛰️',
    desc: '调节光谱差异、云层遮挡、灾害危险性、人口和资产暴露、脆弱性及响应能力，比较遥感地物识别、多时相变化监测、GIS图层叠加、风险分区和应急资源配置。',
    params: [
      {
        key: 'scenario',
        label: '初始灾害监测情境',
        type: 'select',
        options: [
          {
            label: '洪水监测',
            value: 'flood-monitoring',
          },
          {
            label: '森林火灾监测',
            value: 'wildfire-monitoring',
          },
          {
            label: '滑坡监测',
            value: 'landslide-monitoring',
          },
          {
            label: '城市应急',
            value: 'urban-emergency',
          },
          {
            label: '综合风险分析',
            value: 'integrated-risk',
          },
        ],
        defaultValue: 'flood-monitoring',
      },
      {
        key: 'spectralContrast',
        label: '地物光谱差异',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '地物在不同波段中的反射差异越明显，分类和变化识别通常越容易。',
      },
      {
        key: 'cloudCover',
        label: '云层遮挡程度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 3,
        hint: '云层和云影会遮挡地表，降低光学遥感有效观测范围。',
      },
      {
        key: 'hazardIntensity',
        label: '灾害危险强度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '综合表示灾害发生强度、范围和可能造成破坏的程度。',
      },
      {
        key: 'populationExposure',
        label: '人口暴露度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '表示位于危险区域内的人口规模和空间集中程度。',
      },
      {
        key: 'assetExposure',
        label: '资产暴露度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示危险区域内建筑、道路、设施和经济资产的集中程度。',
      },
      {
        key: 'vulnerability',
        label: '承灾体脆弱性',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示人口、建筑和基础设施遭受损失的敏感程度。',
      },
      {
        key: 'responseCapacity',
        label: '应急响应能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '综合表示预警、道路通达、救援组织、物资和医疗保障能力。',
      },
      {
        key: 'showLabels',
        label: '显示图层、地物和路线标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildRemoteSensingGISHTML,
  },
]
