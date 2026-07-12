/**
 * geographyLabTemplatesProductionTransport.ts
 *
 * 地理第39批B3：交通运输方式、交通区位与交通网络。
 *
 * 教学目标：
 * - 比较公路、铁路、水运、航空和多式联运的速度、运量、成本与灵活性；
 * - 理解距离、货运量、价值密度、时效要求、地形阻力和网络密度对运输选择的影响；
 * - 理解交通点、线、网的区位条件和交通枢纽的集散作用；
 * - 观察交通网络改善对可达性、物流成本、区域联系和环境压力的影响。
 *
 * 教学边界：
 * - 所有线路、距离、成本、时间和网络指数均为课堂简化示意；
 * - 不对应真实道路、铁路、港口、机场、企业运价或实时交通状态；
 * - 不用于真实运输方案、物流报价、线路规划、导航或工程决策。
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

function buildProductionTransportHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'road',
    'railway',
    'waterway',
    'aviation',
    'multimodal',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'multimodal',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'multimodal'

  const distance = Math.max(
    50,
    Math.min(
      3000,
      numberValue(params, 'distance', 800),
    ),
  )

  const cargoVolume = Math.max(
    1,
    Math.min(
      10,
      numberValue(params, 'cargoVolume', 7),
    ),
  )

  const valueDensity = Math.max(
    1,
    Math.min(
      10,
      numberValue(params, 'valueDensity', 6),
    ),
  )

  const timeSensitivity = Math.max(
    1,
    Math.min(
      10,
      numberValue(params, 'timeSensitivity', 6),
    ),
  )

  const terrainBarrier = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'terrainBarrier', 4),
    ),
  )

  const networkDensity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'networkDensity', 7),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-production-transport-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #BFDBFE;
      border-radius:18px;
      background:#FFFFFF;
      color:#172554;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(30,64,175,0.11);
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
      border-bottom:1px solid #BFDBFE;
      background:linear-gradient(
        135deg,
        #EFF6FF,
        #ECFEFF 56%,
        #F0FDF4
      );
    }

    #${rootId} .gl-title{
      color:#1D4ED8;
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
      border:1px solid #93C5FD;
      border-radius:999px;
      background:#FFFFFF;
      color:#1D4ED8;
      font-size:11px;
      font-weight:750;
      white-space:nowrap;
    }

    #${rootId} .gl-body{
      height:calc(100% - 56px);
      display:grid;
      grid-template-columns:286px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      min-height:0;
      padding:13px;
      overflow:auto;
      border-right:1px solid #DBEAFE;
      background:linear-gradient(
        180deg,
        #EFF6FF,
        #ECFEFF 62%,
        #F0FDF4
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
        #DBEAFE 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#1D4ED8;
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
      min-width:50px;
      padding:3px 7px;
      border-radius:999px;
      background:#DBEAFE;
      color:#1D4ED8;
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
        #86EFAC
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
        #2563EB,
        #0891B2
      );
      box-shadow:0 1px 5px rgba(30,64,175,0.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #93C5FD;
      border-radius:9px;
      background:#FFFFFF;
      color:#1D4ED8;
      font-size:10.6px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#1D4ED8;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #2563EB,
        #0891B2 55%,
        #16A34A
      );
      box-shadow:0 5px 13px rgba(30,64,175,0.22);
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
      border:1px solid #93C5FD;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #EFF6FF,
        #ECFEFF
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
      border:1px solid #BFDBFE;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-transport-canvas{
      width:100%;
      height:100%;
      display:block;
    }

    @media(max-width:900px){
      #${rootId} .gl-body{
        grid-template-columns:244px minmax(0,1fr);
      }

      #${rootId} .gl-note{
        display:none;
      }
    }
  </style>

  <div class="gl-head">
    <div style="font-size:24px;">
      🚆
    </div>

    <div>
      <div class="gl-title">
        交通运输方式、交通区位与交通网络
      </div>

      <div class="gl-subtitle">
        比较运输方式，观察交通枢纽、网络可达性与物流成本
      </div>
    </div>

    <div class="gl-note">
      课堂简化模型 · 不用于真实运输规划
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        运输方式情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="road">
          公路运输
        </button>

        <button type="button" data-scenario="railway">
          铁路运输
        </button>

        <button type="button" data-scenario="waterway">
          水路运输
        </button>

        <button type="button" data-scenario="aviation">
          航空运输
        </button>

        <button type="button" data-scenario="multimodal">
          多式联运
        </button>
      </div>

      <div class="gl-section-title">
        运输需求与网络条件
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">运输距离</span>
          <span class="gl-value" data-role="distance-value">
            800km
          </span>
        </div>

        <input
          type="range"
          min="50"
          max="3000"
          step="50"
          value="${distance}"
          data-role="distance"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">货运量</span>
          <span class="gl-value" data-role="volume-value">
            7
          </span>
        </div>

        <input
          type="range"
          min="1"
          max="10"
          step="1"
          value="${cargoVolume}"
          data-role="volume"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">货物价值密度</span>
          <span class="gl-value" data-role="value-value">
            6
          </span>
        </div>

        <input
          type="range"
          min="1"
          max="10"
          step="1"
          value="${valueDensity}"
          data-role="value"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">时效要求</span>
          <span class="gl-value" data-role="time-value">
            6
          </span>
        </div>

        <input
          type="range"
          min="1"
          max="10"
          step="1"
          value="${timeSensitivity}"
          data-role="time"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">地形阻力</span>
          <span class="gl-value" data-role="terrain-value">
            4
          </span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${terrainBarrier}"
          data-role="terrain"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">交通网络密度</span>
          <span class="gl-value" data-role="network-value">
            7
          </span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${networkDensity}"
          data-role="network"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          线路标注
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
          切换下一方式
        </button>
      </div>

      <div class="gl-result" data-role="result">
        运输方式选择需要综合考虑距离、运量、时效、成本和网络条件。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="comparison">
          方式比较
        </button>

        <button type="button" data-view="network">
          交通网络
        </button>

        <button type="button" data-view="hub">
          交通枢纽
        </button>

        <button type="button" data-view="logistics">
          物流权衡
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-transport-canvas"
          width="980"
          height="570"
          data-role="canvas"
          aria-label="交通运输方式、交通区位与交通网络教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var distanceInput =
        root.querySelector('[data-role="distance"]');

      var volumeInput =
        root.querySelector('[data-role="volume"]');

      var valueInput =
        root.querySelector('[data-role="value"]');

      var timeInput =
        root.querySelector('[data-role="time"]');

      var terrainInput =
        root.querySelector('[data-role="terrain"]');

      var networkInput =
        root.querySelector('[data-role="network"]');

      var distanceValue =
        root.querySelector('[data-role="distance-value"]');

      var volumeValue =
        root.querySelector('[data-role="volume-value"]');

      var valueValue =
        root.querySelector('[data-role="value-value"]');

      var timeValue =
        root.querySelector('[data-role="time-value"]');

      var terrainValue =
        root.querySelector('[data-role="terrain-value"]');

      var networkValue =
        root.querySelector('[data-role="network-value"]');

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
        !distanceInput ||
        !volumeInput ||
        !valueInput ||
        !timeInput ||
        !terrainInput ||
        !networkInput ||
        !distanceValue ||
        !volumeValue ||
        !valueValue ||
        !timeValue ||
        !terrainValue ||
        !networkValue ||
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
          key:'road',
          name:'公路运输',
          icon:'🚚',
          color:'#EA580C',
          distance:350,
          volume:4,
          value:5,
          time:7,
          terrain:5,
          network:8,
          view:'comparison',
          speed:6,
          capacity:4,
          cost:5,
          flexibility:10,
          environment:5
        },
        {
          key:'railway',
          name:'铁路运输',
          icon:'🚆',
          color:'#2563EB',
          distance:1200,
          volume:9,
          value:5,
          time:5,
          terrain:6,
          network:7,
          view:'network',
          speed:7,
          capacity:9,
          cost:8,
          flexibility:4,
          environment:8
        },
        {
          key:'waterway',
          name:'水路运输',
          icon:'🚢',
          color:'#0891B2',
          distance:2200,
          volume:10,
          value:3,
          time:2,
          terrain:2,
          network:5,
          view:'logistics',
          speed:2,
          capacity:10,
          cost:10,
          flexibility:2,
          environment:7
        },
        {
          key:'aviation',
          name:'航空运输',
          icon:'✈️',
          color:'#7C3AED',
          distance:1800,
          volume:2,
          value:10,
          time:10,
          terrain:2,
          network:6,
          view:'hub',
          speed:10,
          capacity:2,
          cost:2,
          flexibility:5,
          environment:3
        },
        {
          key:'multimodal',
          name:'多式联运',
          icon:'📦',
          color:'#16A34A',
          distance:800,
          volume:7,
          value:6,
          time:6,
          terrain:4,
          network:7,
          view:'network',
          speed:8,
          capacity:8,
          cost:8,
          flexibility:8,
          environment:7
        }
      ];

      var initial = {
        scenario:'${scenario}',
        distance:${distance},
        volume:${cargoVolume},
        value:${valueDensity},
        time:${timeSensitivity},
        terrain:${terrainBarrier},
        network:${networkDensity},
        showLabels:${showLabels}
      };

      var state = {
        scenario:initial.scenario,
        view:'comparison',
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

      function scenarioByKey(key){
        var found =
          scenarios[4];

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
          distance:clamp(
            Number(distanceInput.value) || 800,
            50,
            3000
          ),
          volume:clamp(
            Number(volumeInput.value) || 1,
            1,
            10
          ),
          value:clamp(
            Number(valueInput.value) || 1,
            1,
            10
          ),
          time:clamp(
            Number(timeInput.value) || 1,
            1,
            10
          ),
          terrain:clamp(
            Number(terrainInput.value) || 0,
            0,
            10
          ),
          network:clamp(
            Number(networkInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        distanceInput.value =
          String(
            Math.round(
              value.distance/50
            )*50
          );

        volumeInput.value =
          String(Math.round(value.volume));

        valueInput.value =
          String(Math.round(value.value));

        timeInput.value =
          String(Math.round(value.time));

        terrainInput.value =
          String(Math.round(value.terrain));

        networkInput.value =
          String(Math.round(value.network));
      }

      function scoreScenario(
        item,
        value
      ){
        var distanceFit =
          clamp(
            100-
            Math.abs(
              Math.log(
                (
                  value.distance+100
                )/
                (
                  item.distance+100
                )
              )
            )*
            52,
            0,
            100
          );

        var capacityFit =
          clamp(
            100-
            Math.abs(
              value.volume-
              item.capacity
            )*
            13,
            0,
            100
          );

        var timeFit =
          clamp(
            100-
            Math.abs(
              value.time-
              item.speed
            )*
            12,
            0,
            100
          );

        var preferredValue =
          item.key==='aviation'
            ? 9
            : item.key==='waterway'
              ? 3
              : 6;

        var valueFit =
          clamp(
            100-
            Math.abs(
              value.value-
              preferredValue
            )*
            10,
            0,
            100
          );

        var terrainLimit =
          item.key==='road'
            ? 7
            : item.key==='railway'
              ? 5
              : 8;

        var terrainFit =
          clamp(
            100-
            Math.max(
              0,
              value.terrain-
              terrainLimit
            )*
            12,
            0,
            100
          );

        var networkFit =
          clamp(
            35+
            value.network*
            6.5+
            (
              item.key==='multimodal'
                ? 10
                : 0
            ),
            0,
            100
          );

        return Math.round(
          distanceFit*0.18+
          capacityFit*0.20+
          timeFit*0.19+
          valueFit*0.13+
          terrainFit*0.12+
          networkFit*0.18
        );
      }

      function derive(
        item,
        value
      ){
        var suitability =
          scoreScenario(
            item,
            value
          );

        var speedIndex =
          clamp(
            Math.round(
              item.speed*0.7+
              value.network*0.3-
              value.terrain*0.18
            ),
            1,
            10
          );

        var costIndex =
          clamp(
            Math.round(
              item.cost*0.62+
              value.volume*0.22+
              value.network*0.18-
              value.terrain*0.15
            ),
            1,
            10
          );

        var accessibility =
          clamp(
            Math.round(
              value.network*0.55+
              speedIndex*0.25+
              (
                10-value.terrain
              )*
              0.20
            ),
            0,
            10
          );

        var travelTime =
          Math.max(
            1,
            Math.round(
              value.distance/
              (
                32+
                speedIndex*34
              )
            )
          );

        var logisticsCost =
          Math.max(
            8,
            Math.round(
              value.distance*
              (
                12-
                costIndex*0.72
              )*
              (
                0.7+
                value.volume*0.055
              )/
              10
            )
          );

        var hubStrength =
          clamp(
            Math.round(
              value.network*0.45+
              value.volume*0.22+
              value.value*0.12+
              value.time*0.10+
              (
                item.key==='multimodal'
                  ? 1.4
                  : 0
              )
            ),
            0,
            10
          );

        var environmentPressure =
          clamp(
            Math.round(
              (
                11-item.environment
              )*
              0.52+
              value.volume*0.28+
              value.distance/
              1000*
              0.35-
              value.network*0.12
            ),
            0,
            10
          );

        return {
          suitability:suitability,
          speedIndex:speedIndex,
          costIndex:costIndex,
          accessibility:accessibility,
          travelTime:travelTime,
          logisticsCost:logisticsCost,
          hubStrength:hubStrength,
          environmentPressure:environmentPressure
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
          '#DBEAFE'
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
          '#1D4ED8',
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
          '#BFDBFE'
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

      function comparisonView(
        item,
        value,
        derived
      ){
        background(
          '交通运输方式综合比较',
          '比较速度、运量、成本、灵活性以及当前运输需求下的适宜程度。'
        );

        card(
          28,
          78,
          210,
          '当前方式',
          item.name,
          item.color,
          '课堂比较对象'
        );

        card(
          250,
          78,
          210,
          '综合适宜性',
          derived.suitability,
          item.color,
          '0—100课堂示意值'
        );

        card(
          472,
          78,
          210,
          '估算时间',
          derived.travelTime+'小时',
          '#2563EB',
          '简化速度与距离关系'
        );

        card(
          694,
          78,
          258,
          '相对物流成本',
          derived.logisticsCost,
          '#EA580C',
          '非真实运价'
        );

        var ranking =
          scenarios.map(
            function(candidate){
              return {
                name:candidate.name,
                icon:candidate.icon,
                color:candidate.color,
                key:candidate.key,
                score:scoreScenario(
                  candidate,
                  value
                ),
                speed:candidate.speed,
                capacity:candidate.capacity,
                flexibility:candidate.flexibility
              };
            }
          );

        ranking.sort(
          function(a,b){
            return b.score-a.score;
          }
        );

        ranking.forEach(
          function(candidate,index){
            var y =
              190+
              index*65;

            var active =
              candidate.key===
              item.key;

            box(
              54,
              y,
              872,
              48,
              12,
              active
                ? '#FFFFFF'
                : '#F8FAFC',
              active
                ? candidate.color
                : '#E2E8F0'
            );

            text(
              index+1,
              79,
              y+24,
              12,
              candidate.color,
              880,
              'center'
            );

            text(
              candidate.icon+
              ' '+
              candidate.name,
              110,
              y+24,
              11.5,
              active
                ? candidate.color
                : '#475569',
              800,
              'left'
            );

            box(
              290,
              y+17,
              300,
              14,
              7,
              '#E2E8F0',
              null
            );

            box(
              290,
              y+17,
              300*
              candidate.score/
              100,
              14,
              7,
              candidate.color,
              null
            );

            text(
              '速度 '+
              candidate.speed,
              630,
              y+24,
              9.5,
              '#475569',
              700,
              'left'
            );

            text(
              '运量 '+
              candidate.capacity,
              711,
              y+24,
              9.5,
              '#475569',
              700,
              'left'
            );

            text(
              '灵活 '+
              candidate.flexibility,
              792,
              y+24,
              9.5,
              '#475569',
              700,
              'left'
            );

            text(
              candidate.score,
              894,
              y+24,
              12,
              candidate.color,
              880,
              'center'
            );
          }
        );

        box(
          54,
          523,
          872,
          28,
          10,
          '#FFFFFF',
          '#BFDBFE'
        );

        text(
          '不同运输方式各有优势，多式联运通过衔接不同方式改善全程效率。',
          490,
          537,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function node(
        x,
        y,
        radius,
        label,
        color,
        sub
      ){
        circle(
          x,
          y,
          radius,
          '#FFFFFF',
          color
        );

        text(
          label,
          x,
          y-8,
          11,
          color,
          850,
          'center'
        );

        text(
          sub,
          x,
          y+13,
          9,
          '#64748B',
          650,
          'center'
        );
      }

      function networkView(
        item,
        value,
        derived
      ){
        background(
          '交通网络与区域可达性',
          '观察交通点、交通线和交通网如何改变节点之间的联系强度。'
        );

        card(
          28,
          78,
          210,
          '网络密度',
          Math.round(value.network),
          '#2563EB',
          '节点和线路完善程度'
        );

        card(
          250,
          78,
          210,
          '可达性指数',
          derived.accessibility,
          '#16A34A',
          '网络、速度与地形综合'
        );

        card(
          472,
          78,
          210,
          '地形阻力',
          Math.round(value.terrain),
          '#EA580C',
          '建设和运行阻碍'
        );

        card(
          694,
          78,
          258,
          '交通方式',
          item.name,
          item.color,
          '影响线网组织方式'
        );

        var nodes = [
          {
            x:165,
            y:270,
            label:'城市A',
            sub:'生产节点'
          },
          {
            x:490,
            y:205,
            label:'枢纽B',
            sub:'集散节点'
          },
          {
            x:815,
            y:270,
            label:'城市C',
            sub:'消费节点'
          },
          {
            x:300,
            y:445,
            label:'港口D',
            sub:'门户节点'
          },
          {
            x:680,
            y:445,
            label:'园区E',
            sub:'物流节点'
          }
        ];

        var links = [
          [0,1],
          [1,2],
          [0,3],
          [3,4],
          [4,2],
          [1,3],
          [1,4],
          [0,4]
        ];

        links.forEach(
          function(link,index){
            if(
              index>
              2+
              Math.round(
                value.network*0.5
              )
            ){
              return;
            }

            var from =
              nodes[link[0]];

            var to =
              nodes[link[1]];

            line(
              from.x,
              from.y,
              to.x,
              to.y,
              index<5
                ? item.color
                : '#94A3B8',
              2+
              value.network*
              0.16,
              index>=5
                ? [7,5]
                : []
            );
          }
        );

        var nodeColors = [
          '#2563EB',
          '#7C3AED',
          '#16A34A',
          '#0891B2',
          '#EA580C'
        ];

        nodes.forEach(
          function(current,index){
            node(
              current.x,
              current.y,
              index===1
                ? 58
                : 48,
              index===1
                ? '综合枢纽'
                : current.label,
              index===1
                ? item.color
                : nodeColors[index],
              current.sub
            );
          }
        );

        for(
          var particle=0;
          particle<7;
          particle+=1
        ){
          var progress =
            (
              state.phase+
              particle/7
            )%1;

          var x =
            lerp(
              nodes[0].x,
              nodes[2].x,
              progress
            );

          var y =
            progress<0.5
              ? lerp(
                  nodes[0].y,
                  nodes[1].y,
                  progress*2
                )
              : lerp(
                  nodes[1].y,
                  nodes[2].y,
                  (
                    progress-0.5
                  )*
                  2
                );

          circle(
            x,
            y,
            4,
            '#FFFFFF',
            item.color
          );
        }

        if(state.showLabels){
          text(
            '主通道',
            490,
            260,
            10,
            item.color,
            820,
            'center'
          );

          text(
            '支线和替代通道随网络密度增加',
            490,
            505,
            10,
            '#475569',
            680,
            'center'
          );
        }

        box(
          70,
          523,
          840,
          28,
          10,
          '#FFFFFF',
          '#BFDBFE'
        );

        text(
          '网络密度提高通常增强可达性和抗中断能力，但建设成本与生态影响也会增加。',
          490,
          537,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function hubView(
        item,
        value,
        derived
      ){
        background(
          '交通枢纽的区位与集散作用',
          '枢纽连接不同方向、运输方式和功能节点，促进人流、物流与信息流集散。'
        );

        card(
          28,
          78,
          210,
          '枢纽强度',
          derived.hubStrength,
          item.color,
          '网络、运量和时效综合'
        );

        card(
          250,
          78,
          210,
          '交通网络密度',
          Math.round(value.network),
          '#2563EB',
          '腹地联系基础'
        );

        card(
          472,
          78,
          210,
          '货运量',
          Math.round(value.volume),
          '#16A34A',
          '集散规模'
        );

        card(
          694,
          78,
          258,
          '时效要求',
          Math.round(value.time),
          '#7C3AED',
          '影响衔接效率'
        );

        var centerX=490;
        var centerY=347;

        circle(
          centerX,
          centerY,
          84,
          '#FFFFFF',
          item.color
        );

        text(
          item.icon,
          centerX,
          centerY-23,
          34,
          item.color,
          800,
          'center'
        );

        text(
          '综合交通枢纽',
          centerX,
          centerY+15,
          13,
          item.color,
          860,
          'center'
        );

        text(
          '换装 · 分拨 · 集散',
          centerX,
          centerY+38,
          9.5,
          '#64748B',
          700,
          'center'
        );

        var branches = [
          {
            x:170,
            y:230,
            label:'生产基地',
            color:'#16A34A',
            sub:'货源'
          },
          {
            x:810,
            y:230,
            label:'消费市场',
            color:'#DC2626',
            sub:'需求'
          },
          {
            x:170,
            y:465,
            label:'港口门户',
            color:'#0891B2',
            sub:'水运'
          },
          {
            x:810,
            y:465,
            label:'航空节点',
            color:'#7C3AED',
            sub:'航空'
          },
          {
            x:490,
            y:185,
            label:'城市中心',
            color:'#2563EB',
            sub:'客流'
          }
        ];

        branches.forEach(
          function(branch){
            node(
              branch.x,
              branch.y,
              49,
              branch.label,
              branch.color,
              branch.sub
            );

            arrow(
              branch.x+
              (
                branch.x<centerX
                  ? 53
                  : branch.x>centerX
                    ? -53
                    : 0
              ),
              branch.y+
              (
                branch.y<centerY
                  ? 53
                  : -53
              ),
              centerX+
              (
                branch.x<centerX
                  ? -90
                  : branch.x>centerX
                    ? 90
                    : 0
              ),
              centerY+
              (
                branch.y<centerY
                  ? -45
                  : 45
              ),
              branch.color,
              2+
              derived.hubStrength*
              0.28
            );
          }
        );

        for(
          var index=0;
          index<8;
          index+=1
        ){
          var angle =
            state.phase*
            Math.PI*
            2+
            index*
            Math.PI/
            4;

          circle(
            centerX+
            Math.cos(angle)*
            112,
            centerY+
            Math.sin(angle)*
            112,
            4,
            '#FFFFFF',
            item.color
          );
        }

        box(
          72,
          516,
          836,
          34,
          10,
          '#FFFFFF',
          '#BFDBFE'
        );

        text(
          '枢纽区位通常需要良好腹地、交通衔接、用地条件和稳定客货流支撑。',
          490,
          533,
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

      function logisticsView(
        item,
        value,
        derived
      ){
        background(
          '物流时间、成本与环境权衡',
          '运输组织需要在速度、成本、可靠性、灵活性和环境压力之间权衡。'
        );

        card(
          28,
          78,
          210,
          '运输距离',
          Math.round(value.distance)+'km',
          '#2563EB',
          '课堂线路长度'
        );

        card(
          250,
          78,
          210,
          '估算时间',
          derived.travelTime+'小时',
          '#7C3AED',
          '简化速度关系'
        );

        card(
          472,
          78,
          210,
          '相对物流成本',
          derived.logisticsCost,
          '#EA580C',
          '非真实报价'
        );

        card(
          694,
          78,
          258,
          '环境压力',
          derived.environmentPressure,
          '#DC2626',
          '能耗与运量综合示意'
        );

        gauge(
          190,
          320,
          derived.speedIndex/10,
          '#2563EB',
          '速度指数',
          derived.speedIndex
        );

        gauge(
          490,
          320,
          derived.costIndex/10,
          '#16A34A',
          '成本优势',
          derived.costIndex
        );

        gauge(
          790,
          320,
          derived.environmentPressure/10,
          '#DC2626',
          '环境压力',
          derived.environmentPressure
        );

        var measures = [
          {
            title:'优化装载',
            desc:'提高车辆和舱位利用率',
            color:'#16A34A',
            score:value.volume
          },
          {
            title:'多式联运',
            desc:'发挥不同运输方式优势',
            color:'#0891B2',
            score:item.key==='multimodal'
              ? 10
              : value.network
          },
          {
            title:'枢纽衔接',
            desc:'减少换装等待与空驶',
            color:'#7C3AED',
            score:derived.hubStrength
          },
          {
            title:'绿色运输',
            desc:'降低单位货运能耗排放',
            color:'#EA580C',
            score:10-
              derived.environmentPressure*
              0.55
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
          '#BFDBFE'
        );

        text(
          '速度最快、成本最低和环境影响最小往往不能同时达到，需要按运输任务综合选择。',
          488,
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
        distanceValue.textContent =
          Math.round(value.distance)+'km';

        volumeValue.textContent =
          String(Math.round(value.volume));

        valueValue.textContent =
          String(Math.round(value.value));

        timeValue.textContent =
          String(Math.round(value.time));

        terrainValue.textContent =
          String(Math.round(value.terrain));

        networkValue.textContent =
          String(Math.round(value.network));

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
            ? '自定义任务'
            : item.name;

        result.textContent =
          scenarioName+
          '在当前任务下的综合适宜性约为'+
          derived.suitability+
          '。估算运输时间约'+
          derived.travelTime+
          '小时，相对物流成本为'+
          derived.logisticsCost+
          '，网络可达性为'+
          derived.accessibility+
          '，枢纽强度为'+
          derived.hubStrength+
          '，环境压力为'+
          derived.environmentPressure+
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
          derive(
            item,
            value
          );

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

        if(state.view==='comparison'){
          comparisonView(
            item,
            value,
            derived
          );
        }else if(state.view==='network'){
          networkView(
            item,
            value,
            derived
          );
        }else if(state.view==='hub'){
          hubView(
            item,
            value,
            derived
          );
        }else{
          logisticsView(
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
            'comparison',
            'network',
            'hub',
            'logistics'
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
          distance:lerp(
            from.distance,
            to.distance,
            progress
          ),
          volume:lerp(
            from.volume,
            to.volume,
            progress
          ),
          value:lerp(
            from.value,
            to.value,
            progress
          ),
          time:lerp(
            from.time,
            to.time,
            progress
          ),
          terrain:lerp(
            from.terrain,
            to.terrain,
            progress
          ),
          network:lerp(
            from.network,
            to.network,
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
        distanceInput,
        volumeInput,
        valueInput,
        timeInput,
        terrainInput,
        networkInput
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
                ) || 'multimodal',
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
                ) || 'comparison';

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
            'comparison';

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

export const GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_TRANSPORT:
GeographyLabTemplate[] = [
  {
    id: 'geography-transport-mode-network-location-logistics',
    group: '🏭 生产活动与地域联系',
    name: '交通运输方式、交通区位与交通网络',
    emoji: '🚆',
    desc: '调节运输距离、货运量、价值密度、时效、地形阻力和网络密度，比较运输方式、交通网络、交通枢纽及物流权衡。',
    params: [
      {
        key: 'scenario',
        label: '初始运输方式',
        type: 'select',
        options: [
          {
            label: '公路运输',
            value: 'road',
          },
          {
            label: '铁路运输',
            value: 'railway',
          },
          {
            label: '水路运输',
            value: 'waterway',
          },
          {
            label: '航空运输',
            value: 'aviation',
          },
          {
            label: '多式联运',
            value: 'multimodal',
          },
        ],
        defaultValue: 'multimodal',
      },
      {
        key: 'distance',
        label: '运输距离（千米）',
        type: 'number',
        min: 50,
        max: 3000,
        step: 50,
        defaultValue: 800,
        hint: '短距离更重视灵活性，长距离更重视单位运输成本和干线能力。',
      },
      {
        key: 'cargoVolume',
        label: '货运量',
        type: 'number',
        min: 1,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '大宗、大批量货物通常更适合运量较大的运输方式。',
      },
      {
        key: 'valueDensity',
        label: '货物价值密度',
        type: 'number',
        min: 1,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '价值密度较高的货物通常能够承担更高运输成本。',
      },
      {
        key: 'timeSensitivity',
        label: '时效要求',
        type: 'number',
        min: 1,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '鲜活、急件和高价值货物通常具有较高时效要求。',
      },
      {
        key: 'terrainBarrier',
        label: '地形阻力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 4,
        hint: '山地、河流和复杂地质条件会提高线路建设和维护难度。',
      },
      {
        key: 'networkDensity',
        label: '交通网络密度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '网络密度影响区域可达性、线路选择和多式联运衔接。',
      },
      {
        key: 'showLabels',
        label: '显示线路与节点标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildProductionTransportHTML,
  },
]
