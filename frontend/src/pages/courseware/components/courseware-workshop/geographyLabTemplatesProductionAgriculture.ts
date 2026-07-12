/**
 * geographyLabTemplatesProductionAgriculture.ts
 *
 * 地理第39批B1：农业区位因素与农业地域类型。
 *
 * 教学目标：
 * - 理解热量、降水、土壤、地形和水源等自然区位因素；
 * - 理解市场、交通、技术和投入水平等社会经济区位因素；
 * - 比较水稻种植、小麦种植、乳畜业、园艺业和牧业的区位要求；
 * - 综合观察农业投入、产出、市场响应和生态压力之间的关系。
 *
 * 教学边界：
 * - 所有得分、产量、投入和生态压力均为课堂简化示意；
 * - 不对应任何真实地块、农场、企业、价格或农业政策；
 * - 不用于真实农业选址、经营投资、产量预测或生态评价。
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

function buildAgricultureLocationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'rice',
    'wheat',
    'dairy',
    'horticulture',
    'pastoral',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'rice',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'rice'

  const heat = Math.max(
    0,
    Math.min(10, numberValue(params, 'heat', 8)),
  )

  const water = Math.max(
    0,
    Math.min(10, numberValue(params, 'water', 9)),
  )

  const soil = Math.max(
    0,
    Math.min(10, numberValue(params, 'soil', 7)),
  )

  const terrain = Math.max(
    0,
    Math.min(10, numberValue(params, 'terrain', 8)),
  )

  const market = Math.max(
    0,
    Math.min(10, numberValue(params, 'market', 6)),
  )

  const transport = Math.max(
    0,
    Math.min(10, numberValue(params, 'transport', 6)),
  )

  const technology = Math.max(
    0,
    Math.min(10, numberValue(params, 'technology', 6)),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-agriculture-location-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #BBF7D0;
      border-radius:18px;
      background:#FFFFFF;
      color:#14532D;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(21,128,61,0.11);
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
      border-bottom:1px solid #BBF7D0;
      background:linear-gradient(
        135deg,
        #F0FDF4,
        #FEFCE8 56%,
        #FFF7ED
      );
    }

    #${rootId} .gl-title{
      color:#166534;
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
      border:1px solid #86EFAC;
      border-radius:999px;
      background:#FFFFFF;
      color:#15803D;
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
      border-right:1px solid #DCFCE7;
      background:linear-gradient(
        180deg,
        #F0FDF4,
        #FEFCE8 62%,
        #FFF7ED
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
        #DCFCE7 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#166534;
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
      background:#DCFCE7;
      color:#15803D;
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
        #86EFAC,
        #FDE68A
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
        #16A34A,
        #D97706
      );
      box-shadow:0 1px 5px rgba(21,128,61,0.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #86EFAC;
      border-radius:9px;
      background:#FFFFFF;
      color:#15803D;
      font-size:10.6px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#15803D;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #16A34A,
        #65A30D 55%,
        #D97706
      );
      box-shadow:0 5px 13px rgba(21,128,61,0.22);
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
      border:1px solid #86EFAC;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #F0FDF4,
        #FEFCE8
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
      border:1px solid #BBF7D0;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-agriculture-canvas{
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
      🌾
    </div>

    <div>
      <div class="gl-title">
        农业区位因素与农业地域类型
      </div>

      <div class="gl-subtitle">
        综合比较自然条件、市场交通、技术投入与生态压力
      </div>
    </div>

    <div class="gl-note">
      课堂简化模型 · 不用于真实农业选址
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        农业地域类型
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="rice">
          水稻种植
        </button>

        <button type="button" data-scenario="wheat">
          小麦种植
        </button>

        <button type="button" data-scenario="dairy">
          乳畜业
        </button>

        <button type="button" data-scenario="horticulture">
          园艺业
        </button>

        <button type="button" data-scenario="pastoral">
          牧业
        </button>
      </div>

      <div class="gl-section-title">
        自然与社会经济条件
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">热量条件</span>
          <span class="gl-value" data-role="heat-value">8</span>
        </div>
        <input type="range" min="0" max="10" step="1" value="${heat}" data-role="heat" />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">水分与水源</span>
          <span class="gl-value" data-role="water-value">9</span>
        </div>
        <input type="range" min="0" max="10" step="1" value="${water}" data-role="water" />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">土壤肥力</span>
          <span class="gl-value" data-role="soil-value">7</span>
        </div>
        <input type="range" min="0" max="10" step="1" value="${soil}" data-role="soil" />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">地形平坦度</span>
          <span class="gl-value" data-role="terrain-value">8</span>
        </div>
        <input type="range" min="0" max="10" step="1" value="${terrain}" data-role="terrain" />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">市场接近度</span>
          <span class="gl-value" data-role="market-value">6</span>
        </div>
        <input type="range" min="0" max="10" step="1" value="${market}" data-role="market" />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">交通条件</span>
          <span class="gl-value" data-role="transport-value">6</span>
        </div>
        <input type="range" min="0" max="10" step="1" value="${transport}" data-role="transport" />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">技术投入</span>
          <span class="gl-value" data-role="technology-value">6</span>
        </div>
        <input type="range" min="0" max="10" step="1" value="${technology}" data-role="technology" />
      </div>

      <div class="gl-action-grid">
        <button type="button" data-role="label-toggle" data-active="${showLabels}">
          因素标注
        </button>

        <button type="button" data-role="auto-toggle" data-active="false">
          自动演示
        </button>

        <button type="button" data-role="reset">
          恢复初始
        </button>

        <button type="button" data-role="compare">
          切换下一类型
        </button>
      </div>

      <div class="gl-result" data-role="result">
        农业区位由自然条件和社会经济条件共同决定。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="suitability">
          适宜性比较
        </button>

        <button type="button" data-view="factors">
          区位因子
        </button>

        <button type="button" data-view="layout">
          地域布局
        </button>

        <button type="button" data-view="tradeoff">
          投入与生态
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-agriculture-canvas"
          width="980"
          height="570"
          data-role="canvas"
          aria-label="农业区位因素与农业地域类型教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var heatInput =
        root.querySelector('[data-role="heat"]');

      var waterInput =
        root.querySelector('[data-role="water"]');

      var soilInput =
        root.querySelector('[data-role="soil"]');

      var terrainInput =
        root.querySelector('[data-role="terrain"]');

      var marketInput =
        root.querySelector('[data-role="market"]');

      var transportInput =
        root.querySelector('[data-role="transport"]');

      var technologyInput =
        root.querySelector('[data-role="technology"]');

      var heatValue =
        root.querySelector('[data-role="heat-value"]');

      var waterValue =
        root.querySelector('[data-role="water-value"]');

      var soilValue =
        root.querySelector('[data-role="soil-value"]');

      var terrainValue =
        root.querySelector('[data-role="terrain-value"]');

      var marketValue =
        root.querySelector('[data-role="market-value"]');

      var transportValue =
        root.querySelector('[data-role="transport-value"]');

      var technologyValue =
        root.querySelector('[data-role="technology-value"]');

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
        !heatInput ||
        !waterInput ||
        !soilInput ||
        !terrainInput ||
        !marketInput ||
        !transportInput ||
        !technologyInput ||
        !heatValue ||
        !waterValue ||
        !soilValue ||
        !terrainValue ||
        !marketValue ||
        !transportValue ||
        !technologyValue ||
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
          key:'rice',
          name:'水稻种植',
          icon:'🌾',
          color:'#16A34A',
          heat:8,
          water:9,
          soil:7,
          terrain:8,
          market:6,
          transport:6,
          technology:6,
          view:'suitability',
          ideals:{
            heat:9,
            water:9,
            soil:7,
            terrain:8,
            market:5,
            transport:5,
            technology:6
          },
          weights:{
            heat:1.3,
            water:1.5,
            soil:1.0,
            terrain:1.1,
            market:0.6,
            transport:0.6,
            technology:0.8
          }
        },
        {
          key:'wheat',
          name:'小麦种植',
          icon:'🌿',
          color:'#CA8A04',
          heat:6,
          water:5,
          soil:8,
          terrain:9,
          market:5,
          transport:6,
          technology:6,
          view:'factors',
          ideals:{
            heat:6,
            water:5,
            soil:8,
            terrain:9,
            market:5,
            transport:6,
            technology:6
          },
          weights:{
            heat:1.0,
            water:1.0,
            soil:1.3,
            terrain:1.2,
            market:0.5,
            transport:0.7,
            technology:0.8
          }
        },
        {
          key:'dairy',
          name:'乳畜业',
          icon:'🥛',
          color:'#0284C7',
          heat:5,
          water:7,
          soil:6,
          terrain:6,
          market:9,
          transport:9,
          technology:8,
          view:'layout',
          ideals:{
            heat:5,
            water:7,
            soil:6,
            terrain:6,
            market:10,
            transport:9,
            technology:8
          },
          weights:{
            heat:0.5,
            water:0.7,
            soil:0.5,
            terrain:0.5,
            market:1.5,
            transport:1.4,
            technology:1.2
          }
        },
        {
          key:'horticulture',
          name:'园艺业',
          icon:'🍅',
          color:'#EA580C',
          heat:7,
          water:7,
          soil:8,
          terrain:6,
          market:10,
          transport:9,
          technology:8,
          view:'tradeoff',
          ideals:{
            heat:7,
            water:7,
            soil:8,
            terrain:6,
            market:10,
            transport:9,
            technology:8
          },
          weights:{
            heat:0.8,
            water:0.8,
            soil:0.9,
            terrain:0.4,
            market:1.6,
            transport:1.4,
            technology:1.3
          }
        },
        {
          key:'pastoral',
          name:'牧业',
          icon:'🐑',
          color:'#7C3AED',
          heat:5,
          water:4,
          soil:4,
          terrain:4,
          market:5,
          transport:5,
          technology:5,
          view:'layout',
          ideals:{
            heat:5,
            water:4,
            soil:4,
            terrain:4,
            market:5,
            transport:5,
            technology:5
          },
          weights:{
            heat:0.7,
            water:0.8,
            soil:0.4,
            terrain:0.4,
            market:0.7,
            transport:0.7,
            technology:0.7
          }
        }
      ];

      var initial = {
        scenario:'${scenario}',
        heat:${heat},
        water:${water},
        soil:${soil},
        terrain:${terrain},
        market:${market},
        transport:${transport},
        technology:${technology},
        showLabels:${showLabels}
      };

      var state = {
        scenario:initial.scenario,
        view:'suitability',
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
          scenarios[0];

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
          heat:clamp(
            Number(heatInput.value) || 0,
            0,
            10
          ),
          water:clamp(
            Number(waterInput.value) || 0,
            0,
            10
          ),
          soil:clamp(
            Number(soilInput.value) || 0,
            0,
            10
          ),
          terrain:clamp(
            Number(terrainInput.value) || 0,
            0,
            10
          ),
          market:clamp(
            Number(marketInput.value) || 0,
            0,
            10
          ),
          transport:clamp(
            Number(transportInput.value) || 0,
            0,
            10
          ),
          technology:clamp(
            Number(technologyInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        heatInput.value =
          String(Math.round(value.heat));

        waterInput.value =
          String(Math.round(value.water));

        soilInput.value =
          String(Math.round(value.soil));

        terrainInput.value =
          String(Math.round(value.terrain));

        marketInput.value =
          String(Math.round(value.market));

        transportInput.value =
          String(Math.round(value.transport));

        technologyInput.value =
          String(Math.round(value.technology));
      }

      function scoreScenario(
        item,
        value
      ){
        var keys = [
          'heat',
          'water',
          'soil',
          'terrain',
          'market',
          'transport',
          'technology'
        ];

        var weightedScore=0;
        var weightTotal=0;

        keys.forEach(
          function(key){
            var ideal =
              item.ideals[key];

            var weight =
              item.weights[key];

            var difference =
              Math.abs(
                value[key]-
                ideal
              );

            var score =
              clamp(
                100-
                difference*14,
                0,
                100
              );

            weightedScore +=
              score*weight;

            weightTotal +=
              weight;
          }
        );

        return Math.round(
          weightedScore/
          Math.max(
            0.1,
            weightTotal
          )
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

        var natural =
          (
            value.heat+
            value.water+
            value.soil+
            value.terrain
          )/
          4;

        var social =
          (
            value.market+
            value.transport+
            value.technology
          )/
          3;

        var input =
          clamp(
            Math.round(
              value.technology*0.42+
              value.water*0.20+
              value.transport*0.14+
              value.market*0.12+
              value.soil*0.12
            ),
            0,
            10
          );

        var output =
          clamp(
            Math.round(
              suitability*0.07+
              value.technology*0.22+
              value.market*0.08
            ),
            0,
            10
          );

        var ecologicalPressure =
          clamp(
            Math.round(
              input*0.50+
              Math.max(
                0,
                6-value.water
              )*0.35+
              Math.max(
                0,
                6-value.soil
              )*0.26+
              Math.max(
                0,
                value.technology-7
              )*0.40
            ),
            0,
            10
          );

        var limiting = '条件较均衡';
        var limitingValue=11;

        [
          {
            name:'热量',
            value:value.heat
          },
          {
            name:'水分',
            value:value.water
          },
          {
            name:'土壤',
            value:value.soil
          },
          {
            name:'地形',
            value:value.terrain
          },
          {
            name:'市场',
            value:value.market
          },
          {
            name:'交通',
            value:value.transport
          },
          {
            name:'技术',
            value:value.technology
          }
        ].forEach(
          function(factor){
            if(factor.value<limitingValue){
              limitingValue=factor.value;
              limiting=factor.name;
            }
          }
        );

        return {
          suitability:suitability,
          natural:natural,
          social:social,
          input:input,
          output:output,
          ecologicalPressure:ecologicalPressure,
          limiting:limiting
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
          '#DCFCE7'
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
          '#166534',
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
          '#BBF7D0'
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

      function suitabilityView(
        item,
        value,
        derived
      ){
        background(
          '农业地域类型适宜性比较',
          '在同一组区位条件下，比较五类农业地域类型的相对适宜程度。'
        );

        card(
          28,
          78,
          210,
          '当前类型',
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
          '自然条件',
          derived.natural.toFixed(1),
          '#16A34A',
          '热量、水分、土壤和地形'
        );

        card(
          694,
          78,
          258,
          '社会经济条件',
          derived.social.toFixed(1),
          '#0284C7',
          '市场、交通和技术'
        );

        var rankings =
          scenarios.map(
            function(candidate){
              return {
                key:candidate.key,
                name:candidate.name,
                icon:candidate.icon,
                color:candidate.color,
                score:scoreScenario(
                  candidate,
                  value
                )
              };
            }
          );

        rankings.sort(
          function(a,b){
            return b.score-a.score;
          }
        );

        var startY=192;

        rankings.forEach(
          function(candidate,index){
            var y =
              startY+
              index*64;

            var active =
              candidate.key===
              item.key;

            box(
              76,
              y,
              828,
              46,
              12,
              active
                ? '#FFFFFF'
                : '#F8FAFC',
              active
                ? candidate.color
                : '#E2E8F0'
            );

            text(
              String(index+1),
              100,
              y+23,
              12,
              candidate.color,
              880,
              'center'
            );

            text(
              candidate.icon+
              ' '+
              candidate.name,
              130,
              y+23,
              11.5,
              active
                ? candidate.color
                : '#475569',
              800,
              'left'
            );

            box(
              332,
              y+16,
              480,
              14,
              7,
              '#E2E8F0',
              null
            );

            box(
              332,
              y+16,
              480*
              candidate.score/
              100,
              14,
              7,
              candidate.color,
              null
            );

            text(
              candidate.score,
              858,
              y+23,
              12,
              candidate.color,
              880,
              'center'
            );
          }
        );

        box(
          76,
          521,
          828,
          30,
          10,
          '#FFFFFF',
          '#BBF7D0'
        );

        text(
          '适宜性得分只用于课堂比较；真实农业还受劳动力、政策、价格、文化和风险等因素影响。',
          490,
          536,
          10,
          '#475569',
          670,
          'center'
        );
      }

      function radarPoint(
        centerX,
        centerY,
        radius,
        angle,
        ratio
      ){
        return {
          x:centerX+
            Math.cos(angle)*
            radius*
            ratio,
          y:centerY+
            Math.sin(angle)*
            radius*
            ratio
        };
      }

      function factorsView(
        item,
        value,
        derived
      ){
        background(
          '农业区位因素综合图',
          '雷达图展示七类区位条件，并区分自然条件和社会经济条件。'
        );

        card(
          28,
          78,
          210,
          '当前类型',
          item.name,
          item.color,
          '不同类型权重不同'
        );

        card(
          250,
          78,
          210,
          '限制因素',
          derived.limiting,
          '#DC2626',
          '当前最低条件'
        );

        card(
          472,
          78,
          210,
          '自然条件均值',
          derived.natural.toFixed(1),
          '#16A34A',
          '自然基础'
        );

        card(
          694,
          78,
          258,
          '社会经济均值',
          derived.social.toFixed(1),
          '#0284C7',
          '市场响应与组织能力'
        );

        var labels = [
          '热量',
          '水分',
          '土壤',
          '地形',
          '市场',
          '交通',
          '技术'
        ];

        var keys = [
          'heat',
          'water',
          'soil',
          'terrain',
          'market',
          'transport',
          'technology'
        ];

        var centerX=490;
        var centerY=354;
        var radius=150;
        var startAngle=-Math.PI/2;

        for(
          var level=1;
          level<=5;
          level+=1
        ){
          context.beginPath();

          labels.forEach(
            function(label,index){
              var angle =
                startAngle+
                index*
                Math.PI*2/
                labels.length;

              var point =
                radarPoint(
                  centerX,
                  centerY,
                  radius,
                  angle,
                  level/5
                );

              if(index===0){
                context.moveTo(
                  point.x,
                  point.y
                );
              }else{
                context.lineTo(
                  point.x,
                  point.y
                );
              }
            }
          );

          context.closePath();
          context.strokeStyle='#CBD5E1';
          context.lineWidth=1;
          context.stroke();
        }

        labels.forEach(
          function(label,index){
            var angle =
              startAngle+
              index*
              Math.PI*2/
              labels.length;

            var outer =
              radarPoint(
                centerX,
                centerY,
                radius,
                angle,
                1
              );

            line(
              centerX,
              centerY,
              outer.x,
              outer.y,
              '#CBD5E1',
              1,
              []
            );

            var labelPoint =
              radarPoint(
                centerX,
                centerY,
                radius+28,
                angle,
                1
              );

            text(
              label,
              labelPoint.x,
              labelPoint.y,
              10.5,
              index<=3
                ? '#15803D'
                : '#0369A1',
              800,
              'center'
            );
          }
        );

        context.beginPath();

        keys.forEach(
          function(key,index){
            var angle =
              startAngle+
              index*
              Math.PI*2/
              keys.length;

            var point =
              radarPoint(
                centerX,
                centerY,
                radius,
                angle,
                value[key]/10
              );

            if(index===0){
              context.moveTo(
                point.x,
                point.y
              );
            }else{
              context.lineTo(
                point.x,
                point.y
              );
            }
          }
        );

        context.closePath();
        context.fillStyle='rgba(22,163,74,0.22)';
        context.strokeStyle=item.color;
        context.lineWidth=3;
        context.fill();
        context.stroke();

        keys.forEach(
          function(key,index){
            var angle =
              startAngle+
              index*
              Math.PI*2/
              keys.length;

            var point =
              radarPoint(
                centerX,
                centerY,
                radius,
                angle,
                value[key]/10
              );

            circle(
              point.x,
              point.y,
              5,
              '#FFFFFF',
              item.color
            );

            if(state.showLabels){
              text(
                Math.round(value[key]),
                point.x,
                point.y-14,
                9,
                item.color,
                820,
                'center'
              );
            }
          }
        );

        box(
          80,
          520,
          820,
          30,
          10,
          '#FFFFFF',
          '#BBF7D0'
        );

        text(
          '自然条件提供基础，市场、交通和技术会改变农业生产方式与地域分工。',
          490,
          535,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function road(
        x1,
        y1,
        x2,
        y2,
        widthValue
      ){
        line(
          x1,
          y1,
          x2,
          y2,
          '#94A3B8',
          widthValue+4,
          []
        );

        line(
          x1,
          y1,
          x2,
          y2,
          '#FFFFFF',
          1.3,
          [8,7]
        );
      }

      function zoneLabel(
        x,
        y,
        label,
        color
      ){
        if(!state.showLabels)return;

        box(
          x-52,
          y-14,
          104,
          28,
          14,
          '#FFFFFF',
          color
        );

        text(
          label,
          x,
          y,
          9.5,
          color,
          820,
          'center'
        );
      }

      function layoutView(
        item,
        value,
        derived
      ){
        background(
          '农业地域布局示意',
          '观察市场、交通、水源和自然条件如何影响农业生产空间分布。'
        );

        card(
          28,
          78,
          210,
          '当前类型',
          item.name,
          item.color,
          '课堂布局情境'
        );

        card(
          250,
          78,
          210,
          '市场接近度',
          Math.round(value.market),
          '#EA580C',
          '鲜活产品通常更敏感'
        );

        card(
          472,
          78,
          210,
          '交通条件',
          Math.round(value.transport),
          '#0284C7',
          '影响运输时间和成本'
        );

        card(
          694,
          78,
          258,
          '综合适宜性',
          derived.suitability,
          item.color,
          '布局强度参考'
        );

        var mapX=48;
        var mapY=178;
        var mapW=884;
        var mapH=332;

        box(
          mapX,
          mapY,
          mapW,
          mapH,
          18,
          '#F8FAFC',
          '#CBD5E1'
        );

        context.save();
        roundRect(
          mapX,
          mapY,
          mapW,
          mapH,
          18
        );
        context.clip();

        context.fillStyle='#DCFCE7';
        context.fillRect(
          mapX,
          mapY,
          mapW,
          mapH
        );

        context.fillStyle='#BAE6FD';
        context.beginPath();

        context.moveTo(
          mapX+20,
          mapY+mapH-65
        );

        context.bezierCurveTo(
          mapX+180,
          mapY+mapH-130,
          mapX+420,
          mapY+mapH-30,
          mapX+mapW-20,
          mapY+mapH-95
        );

        context.lineTo(
          mapX+mapW-20,
          mapY+mapH
        );

        context.lineTo(
          mapX+20,
          mapY+mapH
        );

        context.closePath();
        context.fill();

        var marketX =
          mapX+
          mapW*0.74;

        var marketY =
          mapY+
          mapH*0.42;

        var productionX =
          item.key==='dairy' ||
          item.key==='horticulture'
            ? marketX-
              175+
              value.market*5
            : mapX+
              mapW*0.34;

        var productionY =
          item.key==='rice'
            ? mapY+mapH-110
            : item.key==='pastoral'
              ? mapY+100
              : mapY+190;

        road(
          mapX+40,
          mapY+60,
          mapX+mapW-35,
          mapY+mapH-40,
          5+
          value.transport*0.6
        );

        road(
          marketX,
          mapY+20,
          marketX,
          mapY+mapH-20,
          4+
          value.transport*0.5
        );

        context.fillStyle='#FCA5A5';
        context.beginPath();

        context.arc(
          marketX,
          marketY,
          52+
          value.market*2.5,
          0,
          Math.PI*2
        );

        context.fill();

        context.fillStyle=item.color;
        context.globalAlpha=
          0.45+
          derived.suitability/
          220;

        context.beginPath();

        context.ellipse(
          productionX,
          productionY,
          105+
          derived.suitability*0.65,
          58+
          derived.suitability*0.30,
          -0.15,
          0,
          Math.PI*2
        );

        context.fill();
        context.globalAlpha=1;

        context.fillStyle='#FDE68A';

        context.beginPath();

        context.arc(
          mapX+165,
          mapY+88,
          45+
          value.technology*2,
          0,
          Math.PI*2
        );

        context.fill();

        context.restore();

        zoneLabel(
          marketX,
          marketY,
          '消费市场',
          '#DC2626'
        );

        zoneLabel(
          productionX,
          productionY,
          item.name+'区',
          item.color
        );

        zoneLabel(
          mapX+165,
          mapY+88,
          '技术服务节点',
          '#CA8A04'
        );

        zoneLabel(
          mapX+490,
          mapY+mapH-52,
          '水源与灌溉带',
          '#0284C7'
        );

        box(
          48,
          522,
          884,
          30,
          10,
          '#FFFFFF',
          '#BBF7D0'
        );

        var layoutText =
          item.key==='dairy' ||
          item.key==='horticulture'
            ? '乳畜业和园艺业产品较鲜活，通常更接近市场和交通节点。'
            : item.key==='pastoral'
              ? '牧业可利用较广阔草场，但水源、交通和市场仍会影响布局。'
              : '粮食种植更依赖耕地、水热和土壤条件，并通过交通网络联系市场。';

        text(
          layoutText,
          490,
          537,
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
          '农业投入、产出与生态压力',
          '比较技术和资源投入带来的产出提升，同时观察可能出现的生态压力。'
        );

        card(
          28,
          78,
          210,
          '生产投入',
          derived.input,
          '#D97706',
          '技术、水源、交通等综合'
        );

        card(
          250,
          78,
          210,
          '产出水平',
          derived.output,
          '#16A34A',
          '适宜性与技术共同作用'
        );

        card(
          472,
          78,
          210,
          '生态压力',
          derived.ecologicalPressure,
          '#DC2626',
          '资源消耗与环境风险示意'
        );

        card(
          694,
          78,
          258,
          '限制因素',
          derived.limiting,
          item.color,
          '短板可能限制产出'
        );

        gauge(
          190,
          328,
          derived.input/10,
          '#D97706',
          '投入强度',
          derived.input
        );

        gauge(
          490,
          328,
          derived.output/10,
          '#16A34A',
          '产出水平',
          derived.output
        );

        gauge(
          790,
          328,
          derived.ecologicalPressure/10,
          '#DC2626',
          '生态压力',
          derived.ecologicalPressure
        );

        var measures = [
          {
            title:'节水灌溉',
            desc:'提高水资源利用效率',
            color:'#0284C7',
            score:value.technology
          },
          {
            title:'土壤培肥',
            desc:'保持地力和有机质',
            color:'#65A30D',
            score:value.soil
          },
          {
            title:'冷链与加工',
            desc:'降低损耗并连接市场',
            color:'#7C3AED',
            score:(
              value.transport+
              value.technology
            )/2
          },
          {
            title:'生态承载控制',
            desc:'控制过度投入和草场压力',
            color:'#EA580C',
            score:10-
              derived.ecologicalPressure*
              0.45
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
          '#BBF7D0'
        );

        text(
          '技术可以突破部分自然限制，但不能无限替代水土资源和生态承载力。',
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
        heatValue.textContent =
          String(
            Math.round(value.heat)
          );

        waterValue.textContent =
          String(
            Math.round(value.water)
          );

        soilValue.textContent =
          String(
            Math.round(value.soil)
          );

        terrainValue.textContent =
          String(
            Math.round(value.terrain)
          );

        marketValue.textContent =
          String(
            Math.round(value.market)
          );

        transportValue.textContent =
          String(
            Math.round(value.transport)
          );

        technologyValue.textContent =
          String(
            Math.round(value.technology)
          );

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
            ? '自定义条件'
            : item.name;

        result.textContent =
          scenarioName+
          '的综合适宜性约为'+
          derived.suitability+
          '。自然条件均值为'+
          derived.natural.toFixed(1)+
          '，社会经济条件均值为'+
          derived.social.toFixed(1)+
          '。当前相对限制因素是'+
          derived.limiting+
          '，投入强度为'+
          derived.input+
          '，生态压力为'+
          derived.ecologicalPressure+
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

        if(state.view==='suitability'){
          suitabilityView(
            item,
            value,
            derived
          );
        }else if(state.view==='factors'){
          factorsView(
            item,
            value,
            derived
          );
        }else if(state.view==='layout'){
          layoutView(
            item,
            value,
            derived
          );
        }else{
          tradeoffView(
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
            'suitability',
            'factors',
            'layout',
            'tradeoff'
          ][
            Math.floor(
              elapsed/6500
            )%4
          ];

        state.phase =
          (
            elapsed/3400
          )%1;

        setInputs({
          heat:lerp(
            from.heat,
            to.heat,
            progress
          ),
          water:lerp(
            from.water,
            to.water,
            progress
          ),
          soil:lerp(
            from.soil,
            to.soil,
            progress
          ),
          terrain:lerp(
            from.terrain,
            to.terrain,
            progress
          ),
          market:lerp(
            from.market,
            to.market,
            progress
          ),
          transport:lerp(
            from.transport,
            to.transport,
            progress
          ),
          technology:lerp(
            from.technology,
            to.technology,
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
        heatInput,
        waterInput,
        soilInput,
        terrainInput,
        marketInput,
        transportInput,
        technologyInput
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
                ) || 'rice',
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
                ) || 'suitability';

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
            'suitability';

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

export const GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_AGRICULTURE:
GeographyLabTemplate[] = [
  {
    id: 'geography-agriculture-location-regional-types',
    group: '🏭 生产活动与地域联系',
    name: '农业区位因素与农业地域类型',
    emoji: '🌾',
    desc: '调节热量、水分、土壤、地形、市场、交通和技术条件，比较农业地域类型的适宜性、布局、投入产出与生态压力。',
    params: [
      {
        key: 'scenario',
        label: '初始农业地域类型',
        type: 'select',
        options: [
          {
            label: '水稻种植',
            value: 'rice',
          },
          {
            label: '小麦种植',
            value: 'wheat',
          },
          {
            label: '乳畜业',
            value: 'dairy',
          },
          {
            label: '园艺业',
            value: 'horticulture',
          },
          {
            label: '牧业',
            value: 'pastoral',
          },
        ],
        defaultValue: 'rice',
      },
      {
        key: 'heat',
        label: '热量条件',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 8,
        hint: '代表生长期长度和积温条件的课堂强度。',
      },
      {
        key: 'water',
        label: '水分与水源',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 9,
        hint: '综合表示降水、灌溉和稳定水源条件。',
      },
      {
        key: 'soil',
        label: '土壤肥力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '表示土壤肥力、耕层和保水保肥能力。',
      },
      {
        key: 'terrain',
        label: '地形平坦度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 8,
        hint: '平坦地形通常更便于耕作、灌溉和机械化。',
      },
      {
        key: 'market',
        label: '市场接近度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '鲜活产品和高附加值农业通常更重视接近消费市场。',
      },
      {
        key: 'transport',
        label: '交通条件',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '交通影响农产品运输时间、损耗和市场半径。',
      },
      {
        key: 'technology',
        label: '技术投入',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '包括机械、设施、育种、冷链和管理技术等课堂综合值。',
      },
      {
        key: 'showLabels',
        label: '显示区位因素标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildAgricultureLocationHTML,
  },
]
