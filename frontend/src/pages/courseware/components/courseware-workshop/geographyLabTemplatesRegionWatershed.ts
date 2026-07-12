/**
 * geographyLabTemplatesRegionWatershed.ts
 *
 * 地理第40批B1：流域综合开发、梯级开发与生态治理。
 *
 * 教学目标：
 * - 理解流域上、中、下游自然条件和开发任务的差异；
 * - 观察水库梯级开发对发电、防洪、航运、泥沙和生态连通性的影响；
 * - 比较森林覆盖、污染负荷、开发强度和跨区域协同之间的权衡；
 * - 建立全流域统筹、上下游协同和生态补偿的区域治理意识。
 *
 * 教学边界：
 * - 所有流量、发电、防洪、泥沙和生态指标均为课堂简化示意；
 * - 不对应任何真实河流、水库、行政区、工程参数或环境监测数据；
 * - 不用于真实水利工程设计、防洪调度、环境评价或区域规划决策。
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

function buildRegionWatershedHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const scenarios = [
    'upper-development',
    'middle-cascade',
    'lower-urban',
    'integrated-basin',
    'ecological-restoration',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'integrated-basin',
  )

  const scenario = scenarios.includes(requestedScenario)
    ? requestedScenario
    : 'integrated-basin'

  const runoff = Math.max(
    2,
    Math.min(
      10,
      numberValue(params, 'runoff', 7),
    ),
  )

  const forestCover = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'forestCover', 6),
    ),
  )

  const cascadeIntensity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'cascadeIntensity', 6),
    ),
  )

  const pollutionLoad = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'pollutionLoad', 5),
    ),
  )

  const waterUsePressure = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'waterUsePressure', 6),
    ),
  )

  const coordination = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'coordination', 6),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-region-watershed-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #99F6E4;
      border-radius:18px;
      background:#FFFFFF;
      color:#134E4A;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(15,118,110,0.11);
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
      border-bottom:1px solid #99F6E4;
      background:linear-gradient(
        135deg,
        #ECFEFF,
        #F0FDF4 56%,
        #EFF6FF
      );
    }

    #${rootId} .gl-title{
      color:#115E59;
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
      border:1px solid #5EEAD4;
      border-radius:999px;
      background:#FFFFFF;
      color:#0F766E;
      font-size:11px;
      font-weight:750;
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
      border-right:1px solid #CCFBF1;
      background:linear-gradient(
        180deg,
        #ECFEFF,
        #F0FDF4 62%,
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
        circle at 48% 22%,
        #FFFFFF 0%,
        #F8FAFC 62%,
        #CCFBF1 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#115E59;
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
      min-width:44px;
      padding:3px 7px;
      border-radius:999px;
      background:#CCFBF1;
      color:#0F766E;
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
        #5EEAD4,
        #86EFAC,
        #93C5FD
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
        #0F766E,
        #2563EB
      );
      box-shadow:0 1px 5px rgba(15,118,110,0.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #5EEAD4;
      border-radius:9px;
      background:#FFFFFF;
      color:#0F766E;
      font-size:10.6px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#0F766E;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #0F766E,
        #0891B2 55%,
        #2563EB
      );
      box-shadow:0 5px 13px rgba(15,118,110,0.22);
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
      border:1px solid #5EEAD4;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #ECFEFF,
        #F0FDF4
      );
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
      border:1px solid #99F6E4;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-watershed-canvas{
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
    <div style="font-size:24px;">
      🏞️
    </div>

    <div>
      <div class="gl-title">
        流域综合开发、梯级开发与生态治理
      </div>

      <div class="gl-subtitle">
        统筹上中下游发电、防洪、航运、用水与生态安全
      </div>
    </div>

    <div class="gl-note">
      课堂简化模型 · 不用于真实水利调度
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        流域发展情境
      </div>

      <div class="gl-scenario-grid">
        <button
          type="button"
          data-scenario="upper-development"
        >
          上游开发
        </button>

        <button
          type="button"
          data-scenario="middle-cascade"
        >
          中游梯级开发
        </button>

        <button
          type="button"
          data-scenario="lower-urban"
        >
          下游城镇用水
        </button>

        <button
          type="button"
          data-scenario="integrated-basin"
        >
          全流域统筹
        </button>

        <button
          type="button"
          data-scenario="ecological-restoration"
        >
          生态修复优先
        </button>
      </div>

      <div class="gl-section-title">
        流域条件与治理参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">径流丰度</span>
          <span class="gl-value" data-role="runoff-value">7</span>
        </div>

        <input
          type="range"
          min="2"
          max="10"
          step="1"
          value="${runoff}"
          data-role="runoff"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">森林覆盖与水土保持</span>
          <span class="gl-value" data-role="forest-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${forestCover}"
          data-role="forest"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">梯级开发强度</span>
          <span class="gl-value" data-role="cascade-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${cascadeIntensity}"
          data-role="cascade"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">污染负荷</span>
          <span class="gl-value" data-role="pollution-value">5</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${pollutionLoad}"
          data-role="pollution"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">生产生活用水压力</span>
          <span class="gl-value" data-role="water-use-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${waterUsePressure}"
          data-role="water-use"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">跨区域协同水平</span>
          <span class="gl-value" data-role="coordination-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${coordination}"
          data-role="coordination"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          流域标注
        </button>

        <button
          type="button"
          data-role="auto-toggle"
          data-active="false"
        >
          自动演示
        </button>

        <button
          type="button"
          data-role="reset"
        >
          恢复初始
        </button>

        <button
          type="button"
          data-role="compare"
        >
          切换下一情境
        </button>
      </div>

      <div class="gl-result" data-role="result">
        流域开发需要统筹上游生态、中游工程和下游生产生活用水。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="basin">
          流域系统
        </button>

        <button type="button" data-view="cascade">
          梯级开发
        </button>

        <button type="button" data-view="tradeoff">
          开发权衡
        </button>

        <button type="button" data-view="governance">
          协同治理
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-watershed-canvas"
          width="980"
          height="570"
          data-role="canvas"
          aria-label="流域综合开发与生态治理教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var runoffInput =
        root.querySelector('[data-role="runoff"]');

      var forestInput =
        root.querySelector('[data-role="forest"]');

      var cascadeInput =
        root.querySelector('[data-role="cascade"]');

      var pollutionInput =
        root.querySelector('[data-role="pollution"]');

      var waterUseInput =
        root.querySelector('[data-role="water-use"]');

      var coordinationInput =
        root.querySelector('[data-role="coordination"]');

      var runoffValue =
        root.querySelector('[data-role="runoff-value"]');

      var forestValue =
        root.querySelector('[data-role="forest-value"]');

      var cascadeValue =
        root.querySelector('[data-role="cascade-value"]');

      var pollutionValue =
        root.querySelector('[data-role="pollution-value"]');

      var waterUseValue =
        root.querySelector('[data-role="water-use-value"]');

      var coordinationValue =
        root.querySelector('[data-role="coordination-value"]');

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

      var compareButton =
        root.querySelector('[data-role="compare"]');

      var result =
        root.querySelector('[data-role="result"]');

      var canvas =
        root.querySelector('[data-role="canvas"]');

      if(
        !runoffInput ||
        !forestInput ||
        !cascadeInput ||
        !pollutionInput ||
        !waterUseInput ||
        !coordinationInput ||
        !runoffValue ||
        !forestValue ||
        !cascadeValue ||
        !pollutionValue ||
        !waterUseValue ||
        !coordinationValue ||
        !scenarioButtons.length ||
        !viewButtons.length ||
        !labelToggle ||
        !autoToggle ||
        !resetButton ||
        !compareButton ||
        !result ||
        !canvas
      ){
        return;
      }

      var context =
        canvas.getContext('2d');

      if(!context)return;

      var width = canvas.width;
      var height = canvas.height;

      var scenarios = [
        {
          key:'upper-development',
          name:'上游开发',
          runoff:8,
          forest:4,
          cascade:7,
          pollution:3,
          waterUse:4,
          coordination:4,
          view:'basin',
          color:'#2563EB'
        },
        {
          key:'middle-cascade',
          name:'中游梯级开发',
          runoff:8,
          forest:5,
          cascade:9,
          pollution:4,
          waterUse:5,
          coordination:5,
          view:'cascade',
          color:'#7C3AED'
        },
        {
          key:'lower-urban',
          name:'下游城镇用水',
          runoff:6,
          forest:5,
          cascade:5,
          pollution:8,
          waterUse:9,
          coordination:4,
          view:'tradeoff',
          color:'#EA580C'
        },
        {
          key:'integrated-basin',
          name:'全流域统筹',
          runoff:7,
          forest:7,
          cascade:6,
          pollution:4,
          waterUse:6,
          coordination:8,
          view:'governance',
          color:'#0F766E'
        },
        {
          key:'ecological-restoration',
          name:'生态修复优先',
          runoff:6,
          forest:9,
          cascade:3,
          pollution:2,
          waterUse:4,
          coordination:9,
          view:'basin',
          color:'#16A34A'
        }
      ];

      var initial = {
        scenario:'${scenario}',
        runoff:${runoff},
        forest:${forestCover},
        cascade:${cascadeIntensity},
        pollution:${pollutionLoad},
        waterUse:${waterUsePressure},
        coordination:${coordination},
        showLabels:${showLabels}
      };

      var state = {
        scenario:initial.scenario,
        view:'basin',
        showLabels:initial.showLabels,
        auto:false,
        startedAt:0,
        raf:0,
        phase:0,
        compareIndex:0
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
        var progress =
          clamp(t,0,1);

        return progress<0.5
          ? 2*progress*progress
          : 1-
            Math.pow(
              -2*progress+2,
              2
            )/2;
      }

      function roundRect(
        x,
        y,
        w,
        h,
        radius
      ){
        var r =
          Math.min(
            radius,
            w/2,
            h/2
          );

        context.beginPath();
        context.moveTo(x+r,y);
        context.lineTo(x+w-r,y);

        context.quadraticCurveTo(
          x+w,
          y,
          x+w,
          y+r
        );

        context.lineTo(x+w,y+h-r);

        context.quadraticCurveTo(
          x+w,
          y+h,
          x+w-r,
          y+h
        );

        context.lineTo(x+r,y+h);

        context.quadraticCurveTo(
          x,
          y+h,
          x,
          y+h-r
        );

        context.lineTo(x,y+r);

        context.quadraticCurveTo(
          x,
          y,
          x+r,
          y
        );

        context.closePath();
      }

      function box(
        x,
        y,
        w,
        h,
        radius,
        fill,
        stroke
      ){
        roundRect(
          x,
          y,
          w,
          h,
          radius
        );

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

        context.fillStyle =
          color || '#334155';

        context.textAlign =
          align || 'left';

        context.textBaseline =
          'middle';

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
        radius,
        fill,
        stroke
      ){
        context.beginPath();

        context.arc(
          x,
          y,
          radius,
          0,
          Math.PI*2
        );

        if(fill){
          context.fillStyle=fill;
          context.fill();
        }

        if(stroke){
          context.strokeStyle=stroke;
          context.lineWidth=2;
          context.stroke();
        }
      }

      function scenarioByKey(key){
        var found =
          scenarios[3];

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
          runoff:clamp(
            Number(runoffInput.value) || 2,
            2,
            10
          ),
          forest:clamp(
            Number(forestInput.value) || 0,
            0,
            10
          ),
          cascade:clamp(
            Number(cascadeInput.value) || 0,
            0,
            10
          ),
          pollution:clamp(
            Number(pollutionInput.value) || 0,
            0,
            10
          ),
          waterUse:clamp(
            Number(waterUseInput.value) || 0,
            0,
            10
          ),
          coordination:clamp(
            Number(coordinationInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        runoffInput.value =
          String(Math.round(value.runoff));

        forestInput.value =
          String(Math.round(value.forest));

        cascadeInput.value =
          String(Math.round(value.cascade));

        pollutionInput.value =
          String(Math.round(value.pollution));

        waterUseInput.value =
          String(Math.round(value.waterUse));

        coordinationInput.value =
          String(Math.round(value.coordination));
      }

      function derive(value){
        var floodRisk =
          clamp(
            Math.round(
              value.runoff*0.72+
              (
                10-value.forest
              )*0.48-
              value.cascade*0.28-
              value.coordination*0.18
            ),
            0,
            10
          );

        var powerBenefit =
          clamp(
            Math.round(
              value.runoff*0.42+
              value.cascade*0.58
            ),
            0,
            10
          );

        var navigation =
          clamp(
            Math.round(
              value.runoff*0.36+
              value.cascade*0.30+
              value.coordination*0.24-
              value.pollution*0.08
            ),
            0,
            10
          );

        var sediment =
          clamp(
            Math.round(
              (
                10-value.forest
              )*0.62+
              value.runoff*0.25-
              value.cascade*0.18
            ),
            0,
            10
          );

        var ecology =
          clamp(
            Math.round(
              value.forest*0.38+
              (
                10-value.pollution
              )*0.28+
              (
                10-value.cascade
              )*0.16+
              value.coordination*0.18-
              value.waterUse*0.08
            ),
            0,
            10
          );

        var waterSecurity =
          clamp(
            Math.round(
              value.runoff*0.28+
              value.cascade*0.22+
              value.coordination*0.34+
              (
                10-value.pollution
              )*0.18-
              value.waterUse*0.22
            ),
            0,
            10
          );

        var development =
          clamp(
            Math.round(
              powerBenefit*0.34+
              navigation*0.18+
              waterSecurity*0.26+
              value.waterUse*0.22
            ),
            0,
            10
          );

        var coordinationScore =
          clamp(
            Math.round(
              value.coordination*0.55+
              ecology*0.22+
              waterSecurity*0.23
            ),
            0,
            10
          );

        return {
          floodRisk:floodRisk,
          powerBenefit:powerBenefit,
          navigation:navigation,
          sediment:sediment,
          ecology:ecology,
          waterSecurity:waterSecurity,
          development:development,
          coordinationScore:coordinationScore
        };
      }

      function background(
        titleValue,
        subtitle
      ){
        var gradient =
          context.createLinearGradient(
            0,
            0,
            width,
            height
          );

        gradient.addColorStop(
          0,
          '#FFFFFF'
        );

        gradient.addColorStop(
          0.58,
          '#F8FAFC'
        );

        gradient.addColorStop(
          1,
          '#CCFBF1'
        );

        context.fillStyle=gradient;

        context.fillRect(
          0,
          0,
          width,
          height
        );

        text(
          titleValue,
          28,
          31,
          18,
          '#115E59',
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
          'rgba(255,255,255,0.94)',
          '#99F6E4'
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
          9.5,
          '#64748B',
          600,
          'left'
        );
      }

      function arrow(
        x1,
        y1,
        x2,
        y2,
        color,
        lineWidth
      ){
        var angle =
          Math.atan2(
            y2-y1,
            x2-x1
          );

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
          x2-
          head*
          Math.cos(
            angle-Math.PI/6
          ),
          y2-
          head*
          Math.sin(
            angle-Math.PI/6
          )
        );

        context.lineTo(
          x2-
          head*
          Math.cos(
            angle+Math.PI/6
          ),
          y2-
          head*
          Math.sin(
            angle+Math.PI/6
          )
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function basinView(
        item,
        value,
        derived
      ){
        background(
          '流域上中下游系统',
          '河流把上游生态、中游工程和下游生产生活联系成完整区域系统。'
        );

        card(
          28,
          78,
          210,
          '洪水风险',
          derived.floodRisk,
          '#DC2626',
          '径流、植被与调蓄共同影响'
        );

        card(
          250,
          78,
          210,
          '水土流失与泥沙',
          derived.sediment,
          '#D97706',
          '森林覆盖影响显著'
        );

        card(
          472,
          78,
          210,
          '供水安全',
          derived.waterSecurity,
          '#2563EB',
          '水量、水质与协同调度'
        );

        card(
          694,
          78,
          258,
          '生态健康',
          derived.ecology,
          '#16A34A',
          '连通性、水质与栖息地'
        );

        context.fillStyle='#DCFCE7';

        context.beginPath();
        context.moveTo(34,520);
        context.lineTo(34,230);
        context.lineTo(170,185);
        context.lineTo(310,245);
        context.lineTo(500,285);
        context.lineTo(720,360);
        context.lineTo(946,400);
        context.lineTo(946,520);
        context.closePath();
        context.fill();

        context.strokeStyle='#0284C7';
        context.lineWidth=18;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(130,220);

        context.bezierCurveTo(
          260,
          250,
          330,
          320,
          470,
          334
        );

        context.bezierCurveTo(
          620,
          350,
          710,
          430,
          902,
          470
        );

        context.stroke();

        context.strokeStyle='#67E8F9';
        context.lineWidth=5;

        context.beginPath();
        context.moveTo(130,220);

        context.bezierCurveTo(
          260,
          250,
          330,
          320,
          470,
          334
        );

        context.bezierCurveTo(
          620,
          350,
          710,
          430,
          902,
          470
        );

        context.stroke();

        if(state.showLabels){
          box(
            76,
            175,
            165,
            44,
            12,
            '#FFFFFF',
            '#16A34A'
          );

          text(
            '上游：水源涵养',
            158,
            192,
            11,
            '#15803D',
            850,
            'center'
          );

          text(
            '水土保持与生态保护',
            158,
            208,
            9,
            '#64748B',
            650,
            'center'
          );

          box(
            390,
            285,
            185,
            44,
            12,
            '#FFFFFF',
            '#7C3AED'
          );

          text(
            '中游：梯级开发',
            482,
            302,
            11,
            '#6D28D9',
            850,
            'center'
          );

          text(
            '发电、防洪与航运',
            482,
            318,
            9,
            '#64748B',
            650,
            'center'
          );

          box(
            724,
            412,
            178,
            44,
            12,
            '#FFFFFF',
            '#EA580C'
          );

          text(
            '下游：城镇与农业',
            813,
            429,
            11,
            '#C2410C',
            850,
            'center'
          );

          text(
            '供水、防洪与水质',
            813,
            445,
            9,
            '#64748B',
            650,
            'center'
          );
        }

        var forestCount =
          Math.round(
            value.forest*1.6
          );

        for(
          var tree=0;
          tree<forestCount;
          tree+=1
        ){
          var treeX =
            72+
            (
              tree%8
            )*
            24;

          var treeY =
            238+
            Math.floor(
              tree/8
            )*
            28;

          context.fillStyle='#166534';

          context.beginPath();
          context.moveTo(
            treeX,
            treeY-13
          );

          context.lineTo(
            treeX-9,
            treeY+4
          );

          context.lineTo(
            treeX+9,
            treeY+4
          );

          context.closePath();
          context.fill();

          line(
            treeX,
            treeY+4,
            treeX,
            treeY+13,
            '#92400E',
            2,
            []
          );
        }

        var pollutionDots =
          Math.round(
            value.pollution
          );

        for(
          var dot=0;
          dot<pollutionDots;
          dot+=1
        ){
          circle(
            730+
            dot*14,
            482-
            (
              dot%2
            )*
            9,
            4,
            'rgba(220,38,38,0.75)',
            null
          );
        }

        box(
          44,
          528,
          892,
          28,
          10,
          '#FFFFFF',
          '#99F6E4'
        );

        text(
          '上游保护不足会增加泥沙和洪水风险；下游用水与污染压力会反向要求全流域协同治理。',
          490,
          542,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function dam(
        x,
        y,
        heightValue,
        color,
        label
      ){
        context.fillStyle=color;

        context.beginPath();

        context.moveTo(
          x-18,
          y+
          heightValue/2
        );

        context.lineTo(
          x+18,
          y+
          heightValue/2
        );

        context.lineTo(
          x+10,
          y-
          heightValue/2
        );

        context.lineTo(
          x-10,
          y-
          heightValue/2
        );

        context.closePath();
        context.fill();

        if(state.showLabels){
          text(
            label,
            x,
            y-
            heightValue/2-
            13,
            9,
            color,
            820,
            'center'
          );
        }
      }

      function cascadeView(
        item,
        value,
        derived
      ){
        background(
          '梯级开发与河流连续性',
          '梯级水库可以提高发电和调蓄能力，也会改变泥沙输移和生态连通。'
        );

        card(
          28,
          78,
          210,
          '发电收益',
          derived.powerBenefit,
          '#7C3AED',
          '径流与梯级强度共同影响'
        );

        card(
          250,
          78,
          210,
          '防洪调蓄',
          10-
          derived.floodRisk,
          '#2563EB',
          '水库调蓄的课堂示意'
        );

        card(
          472,
          78,
          210,
          '航运条件',
          derived.navigation,
          '#0891B2',
          '水位、流量与协同调度'
        );

        card(
          694,
          78,
          258,
          '生态连通性',
          derived.ecology,
          '#16A34A',
          '过坝、泥沙与栖息地'
        );

        context.fillStyle='#E0F2FE';

        context.beginPath();
        context.moveTo(58,220);
        context.lineTo(218,250);
        context.lineTo(382,305);
        context.lineTo(546,360);
        context.lineTo(720,418);
        context.lineTo(922,465);
        context.lineTo(922,520);
        context.lineTo(58,520);
        context.closePath();
        context.fill();

        line(
          70,
          244,
          912,
          486,
          '#0284C7',
          18,
          []
        );

        line(
          70,
          244,
          912,
          486,
          '#67E8F9',
          5,
          []
        );

        var damCount =
          Math.max(
            1,
            Math.round(
              value.cascade/2
            )
          );

        for(
          var index=0;
          index<damCount;
          index+=1
        ){
          var ratio =
            (
              index+1
            )/
            (
              damCount+1
            );

          var x =
            90+
            ratio*780;

          var y =
            250+
            ratio*220;

          dam(
            x,
            y,
            36+
            value.cascade*2,
            item.color,
            '第'+
            (
              index+1
            )+
            '级'
          );
        }

        for(
          var particle=0;
          particle<9;
          particle+=1
        ){
          var progress =
            (
              state.phase+
              particle/9
            )%1;

          var particleX =
            80+
            progress*820;

          var particleY =
            248+
            progress*235+
            Math.sin(
              progress*
              Math.PI*
              6
            )*
            5;

          circle(
            particleX,
            particleY,
            4,
            '#FFFFFF',
            '#0284C7'
          );
        }

        box(
          60,
          190,
          250,
          56,
          14,
          '#FFFFFF',
          '#7C3AED'
        );

        text(
          '梯级收益',
          78,
          209,
          11,
          '#6D28D9',
          850,
          'left'
        );

        text(
          '发电、调峰、防洪、供水、改善部分航段',
          78,
          231,
          9.5,
          '#475569',
          650,
          'left'
        );

        box(
          645,
          205,
          275,
          56,
          14,
          '#FFFFFF',
          '#DC2626'
        );

        text(
          '梯级约束',
          663,
          224,
          11,
          '#B91C1C',
          850,
          'left'
        );

        text(
          '阻隔鱼类洄游、拦沙、改变水温和径流节律',
          663,
          246,
          9.5,
          '#475569',
          650,
          'left'
        );

        box(
          60,
          526,
          860,
          28,
          10,
          '#FFFFFF',
          '#99F6E4'
        );

        text(
          '梯级开发强度越高，综合效益可能增加，但必须同步设置生态流量、过鱼和泥沙管理措施。',
          490,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function gauge(
        centerX,
        centerY,
        ratio,
        color,
        label,
        valueText
      ){
        context.save();
        context.lineWidth=14;
        context.strokeStyle='#E2E8F0';

        context.beginPath();

        context.arc(
          centerX,
          centerY,
          78,
          Math.PI,
          Math.PI*2
        );

        context.stroke();

        context.strokeStyle=color;
        context.lineCap='round';

        context.beginPath();

        context.arc(
          centerX,
          centerY,
          78,
          Math.PI,
          Math.PI+
          Math.PI*
          clamp(
            ratio,
            0,
            1
          )
        );

        context.stroke();
        context.restore();

        text(
          valueText,
          centerX,
          centerY-4,
          18,
          color,
          880,
          'center'
        );

        text(
          label,
          centerX,
          centerY+24,
          9.5,
          '#64748B',
          720,
          'center'
        );
      }

      function tradeoffView(
        item,
        value,
        derived
      ){
        background(
          '流域开发效益与生态风险权衡',
          '比较经济开发、防洪供水和生态安全，寻找更均衡的流域方案。'
        );

        card(
          28,
          78,
          210,
          '开发收益',
          derived.development,
          item.color,
          '发电、航运与用水综合'
        );

        card(
          250,
          78,
          210,
          '供水安全',
          derived.waterSecurity,
          '#2563EB',
          '水量水质与调度'
        );

        card(
          472,
          78,
          210,
          '生态健康',
          derived.ecology,
          '#16A34A',
          '森林、水质与连通性'
        );

        card(
          694,
          78,
          258,
          '洪水风险',
          derived.floodRisk,
          '#DC2626',
          '高值表示风险较高'
        );

        gauge(
          190,
          322,
          derived.development/10,
          item.color,
          '开发收益',
          derived.development
        );

        gauge(
          490,
          322,
          derived.waterSecurity/10,
          '#2563EB',
          '供水安全',
          derived.waterSecurity
        );

        gauge(
          790,
          322,
          derived.ecology/10,
          '#16A34A',
          '生态健康',
          derived.ecology
        );

        var measures = [
          {
            title:'水土保持',
            desc:'提升森林和坡面治理',
            color:'#16A34A',
            score:value.forest
          },
          {
            title:'生态调度',
            desc:'控制梯级开发与生态流量',
            color:'#7C3AED',
            score:10-
              Math.abs(
                value.cascade-5
              )
          },
          {
            title:'污染治理',
            desc:'削减工业农业生活污染',
            color:'#DC2626',
            score:10-
              value.pollution
          },
          {
            title:'节水与协同',
            desc:'上下游共同分配和补偿',
            color:'#2563EB',
            score:(
              10-
              value.waterUse+
              value.coordination
            )/
            2
          }
        ];

        measures.forEach(
          function(measure,index){
            var x =
              44+
              index*228;

            box(
              x,
              430,
              204,
              82,
              14,
              '#FFFFFF',
              measure.color
            );

            text(
              measure.title,
              x+15,
              451,
              11,
              measure.color,
              850,
              'left'
            );

            text(
              measure.desc,
              x+15,
              473,
              9.2,
              '#64748B',
              620,
              'left'
            );

            box(
              x+15,
              490,
              174,
              9,
              5,
              '#E2E8F0',
              null
            );

            box(
              x+15,
              490,
              174*
              clamp(
                measure.score/10,
                0,
                1
              ),
              9,
              5,
              measure.color,
              null
            );
          }
        );

        box(
          44,
          526,
          888,
          28,
          10,
          '#FFFFFF',
          '#99F6E4'
        );

        text(
          '流域治理的目标不是让单一指标最大化，而是在安全、发展和生态之间建立可持续平衡。',
          488,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function governanceColumn(
        x,
        titleValue,
        color,
        items
      ){
        box(
          x,
          176,
          274,
          322,
          16,
          '#FFFFFF',
          color
        );

        text(
          titleValue,
          x+137,
          203,
          14,
          color,
          860,
          'center'
        );

        items.forEach(
          function(item,index){
            var y =
              240+
              index*58;

            box(
              x+18,
              y,
              238,
              43,
              10,
              index%2===0
                ? '#F8FAFC'
                : '#FFFFFF',
              '#E2E8F0'
            );

            circle(
              x+39,
              y+21,
              10,
              color,
              null
            );

            text(
              index+1,
              x+39,
              y+21,
              9,
              '#FFFFFF',
              850,
              'center'
            );

            text(
              item,
              x+59,
              y+21,
              10,
              '#475569',
              650,
              'left'
            );
          }
        );
      }

      function governanceView(
        item,
        value,
        derived
      ){
        background(
          '全流域协同治理',
          '以全流域为单元协调行政区、行业和上下游利益。'
        );

        card(
          28,
          78,
          210,
          '协同治理指数',
          derived.coordinationScore,
          '#0F766E',
          '协同、生态与供水综合'
        );

        card(
          250,
          78,
          210,
          '跨区域协同',
          Math.round(value.coordination),
          '#2563EB',
          '信息共享与联合调度'
        );

        card(
          472,
          78,
          210,
          '污染负荷',
          Math.round(value.pollution),
          '#DC2626',
          '需要总量控制和溯源'
        );

        card(
          694,
          78,
          258,
          '用水压力',
          Math.round(value.waterUse),
          '#EA580C',
          '生产生活生态用水竞争'
        );

        governanceColumn(
          42,
          '上游地区',
          '#16A34A',
          [
            '保护水源与森林',
            '控制水土流失',
            '承担生态保护成本',
            '获得生态补偿'
          ]
        );

        governanceColumn(
          353,
          '流域统筹机构',
          '#0F766E',
          [
            '统一监测与信息共享',
            '联合防洪和水库调度',
            '水量水质目标协同',
            '建立补偿和责任机制'
          ]
        );

        governanceColumn(
          664,
          '中下游地区',
          '#2563EB',
          [
            '节水和污染治理',
            '保障生态流量',
            '合理承担补偿责任',
            '共享流域发展收益'
          ]
        );

        box(
          42,
          520,
          896,
          32,
          10,
          '#ECFEFF',
          '#5EEAD4'
        );

        text(
          '流域边界常与行政区边界不一致，因此需要跨区域协调、统一监测和利益补偿。',
          490,
          536,
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
        runoffValue.textContent =
          String(Math.round(value.runoff));

        forestValue.textContent =
          String(Math.round(value.forest));

        cascadeValue.textContent =
          String(Math.round(value.cascade));

        pollutionValue.textContent =
          String(Math.round(value.pollution));

        waterUseValue.textContent =
          String(Math.round(value.waterUse));

        coordinationValue.textContent =
          String(Math.round(value.coordination));

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

        var scenarioName =
          state.scenario==='custom'
            ? '自定义流域条件'
            : item.name;

        result.textContent =
          scenarioName+
          '下，开发收益为'+
          derived.development+
          '，供水安全为'+
          derived.waterSecurity+
          '，生态健康为'+
          derived.ecology+
          '，洪水风险为'+
          derived.floodRisk+
          '，泥沙压力为'+
          derived.sediment+
          '，协同治理指数为'+
          derived.coordinationScore+
          '。';
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var item =
          scenarioByKey(
            state.scenario
          );

        var value =
          values();

        var derived =
          derive(value);

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

        if(state.view==='basin'){
          basinView(
            item,
            value,
            derived
          );
        }else if(state.view==='cascade'){
          cascadeView(
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
        }else{
          governanceView(
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
        var item =
          scenarioByKey(key);

        state.scenario =
          item.key;

        setInputs(item);

        if(changeView){
          state.view =
            item.view;
        }

        state.compareIndex =
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

        var elapsed =
          timestamp-
          state.startedAt;

        var duration =
          5200;

        var segment =
          Math.floor(
            elapsed/duration
          );

        var local =
          (
            elapsed%duration
          )/
          duration;

        var from =
          scenarios[
            segment%
            scenarios.length
          ];

        var to =
          scenarios[
            (
              segment+1
            )%
            scenarios.length
          ];

        var progress =
          ease(
            clamp(
              local/0.82,
              0,
              1
            )
          );

        state.scenario =
          local<0.5
            ? from.key
            : to.key;

        state.view =
          [
            'basin',
            'cascade',
            'tradeoff',
            'governance'
          ][
            Math.floor(
              elapsed/6500
            )%4
          ];

        state.phase =
          (
            elapsed/3300
          )%1;

        setInputs({
          runoff:lerp(
            from.runoff,
            to.runoff,
            progress
          ),
          forest:lerp(
            from.forest,
            to.forest,
            progress
          ),
          cascade:lerp(
            from.cascade,
            to.cascade,
            progress
          ),
          pollution:lerp(
            from.pollution,
            to.pollution,
            progress
          ),
          waterUse:lerp(
            from.waterUse,
            to.waterUse,
            progress
          ),
          coordination:lerp(
            from.coordination,
            to.coordination,
            progress
          )
        });

        render();

        state.raf =
          requestAnimationFrame(
            animate
          );
      }

      function manual(){
        if(state.auto){
          stopAuto();
        }

        state.scenario =
          'custom';

        render();
      }

      [
        runoffInput,
        forestInput,
        cascadeInput,
        pollutionInput,
        waterUseInput,
        coordinationInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            manual
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
                ) || 'integrated-basin',
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

              state.view =
                button.getAttribute(
                  'data-view'
                ) || 'basin';

              render();
            }
          );
        }
      );

      labelToggle.addEventListener(
        'click',
        function(){
          state.showLabels =
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

          state.raf =
            requestAnimationFrame(
              animate
            );

          render();
        }
      );

      resetButton.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
          }

          state.scenario =
            initial.scenario;

          state.view =
            'basin';

          state.showLabels =
            initial.showLabels;

          setInputs(initial);

          state.compareIndex =
            scenarios.indexOf(
              scenarioByKey(
                initial.scenario
              )
            );

          render();
        }
      );

      compareButton.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
          }

          state.compareIndex =
            (
              state.compareIndex+1
            )%
            scenarios.length;

          var next =
            scenarios[
              state.compareIndex
            ];

          state.scenario =
            next.key;

          state.view =
            next.view;

          setInputs(next);
          render();
        }
      );

      state.compareIndex =
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

export const GEOGRAPHY_LAB_TEMPLATES_REGION_WATERSHED:
GeographyLabTemplate[] = [
  {
    id: 'geography-watershed-integrated-development-ecological-governance',
    group: '🗺️ 区域发展与资源环境',
    name: '流域综合开发、梯级开发与生态治理',
    emoji: '🏞️',
    desc: '调节径流、森林覆盖、梯级开发、污染、用水压力和区域协同，观察流域开发、防洪供水、泥沙与生态治理的综合权衡。',
    params: [
      {
        key: 'scenario',
        label: '初始流域情境',
        type: 'select',
        options: [
          {
            label: '上游开发',
            value: 'upper-development',
          },
          {
            label: '中游梯级开发',
            value: 'middle-cascade',
          },
          {
            label: '下游城镇用水',
            value: 'lower-urban',
          },
          {
            label: '全流域统筹',
            value: 'integrated-basin',
          },
          {
            label: '生态修复优先',
            value: 'ecological-restoration',
          },
        ],
        defaultValue: 'integrated-basin',
      },
      {
        key: 'runoff',
        label: '径流丰度',
        type: 'number',
        min: 2,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '径流较丰沛有利于供水和发电，但也可能提高洪水压力。',
      },
      {
        key: 'forestCover',
        label: '森林覆盖与水土保持',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '森林覆盖和坡面治理有助于涵养水源、减缓径流和控制泥沙。',
      },
      {
        key: 'cascadeIntensity',
        label: '梯级开发强度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '梯级开发可增加发电和调蓄，也会改变河流连续性和泥沙输移。',
      },
      {
        key: 'pollutionLoad',
        label: '污染负荷',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '综合表示工业、农业和生活污染对流域水质的课堂压力。',
      },
      {
        key: 'waterUsePressure',
        label: '生产生活用水压力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '用水压力提高会增加生产、生活和生态用水之间的竞争。',
      },
      {
        key: 'coordination',
        label: '跨区域协同水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '协同包括统一监测、联合调度、污染联防和生态补偿。',
      },
      {
        key: 'showLabels',
        label: '显示流域结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildRegionWatershedHTML,
  },
]
