/**
 * geographyLabTemplatesProductionIndustry.ts
 *
 * 地理第39批B2：工业区位因素、工业集聚与产业转移。
 *
 * 教学目标：
 * - 理解原料、市场、交通、劳动力、能源、技术和环境容量对工业区位的影响；
 * - 比较原料导向型、市场导向型、劳动力导向型、技术导向型和综合型工业；
 * - 理解工业集聚的共享设施、专业化协作、信息交流与拥挤成本；
 * - 分析产业转移的推力、拉力及其对转出地和承接地的双向影响。
 *
 * 教学边界：
 * - 所有得分、成本、集聚效应和产业转移流均为课堂简化示意；
 * - 不对应真实企业、园区、城市、工资、能源价格或环境许可；
 * - 不用于真实工业选址、投资决策、产业规划或环境影响评价。
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

function buildIndustryLocationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'raw-material',
    'market',
    'labor',
    'technology',
    'integrated',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'integrated',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'integrated'

  const rawMaterial = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'rawMaterial', 6),
    ),
  )

  const market = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'market', 7),
    ),
  )

  const transport = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'transport', 7),
    ),
  )

  const labor = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'labor', 6),
    ),
  )

  const energy = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'energy', 6),
    ),
  )

  const technology = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'technology', 7),
    ),
  )

  const environment = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'environment', 6),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-industry-location-root">
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
      box-shadow:0 12px 34px rgba(91,33,182,0.11);
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
        #EFF6FF 56%,
        #F0FDF4
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
      border-right:1px solid #EDE9FE;
      background:linear-gradient(
        180deg,
        #F5F3FF,
        #EFF6FF 62%,
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
      background:#EDE9FE;
      color:#6D28D9;
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
        #C4B5FD,
        #93C5FD,
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
        #7C3AED,
        #2563EB
      );
      box-shadow:0 1px 5px rgba(91,33,182,0.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #C4B5FD;
      border-radius:9px;
      background:#FFFFFF;
      color:#6D28D9;
      font-size:10.6px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#6D28D9;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #7C3AED,
        #2563EB 55%,
        #16A34A
      );
      box-shadow:0 5px 13px rgba(91,33,182,0.22);
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
        #EFF6FF
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
      border:1px solid #DDD6FE;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-industry-canvas{
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
      🏭
    </div>

    <div>
      <div class="gl-title">
        工业区位因素、工业集聚与产业转移
      </div>

      <div class="gl-subtitle">
        综合比较成本、市场、技术、环境与区域联系
      </div>
    </div>

    <div class="gl-note">
      课堂简化模型 · 不用于真实工业选址
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        工业区位类型
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="raw-material">
          原料导向型
        </button>

        <button type="button" data-scenario="market">
          市场导向型
        </button>

        <button type="button" data-scenario="labor">
          劳动力导向型
        </button>

        <button type="button" data-scenario="technology">
          技术导向型
        </button>

        <button type="button" data-scenario="integrated">
          综合型工业
        </button>
      </div>

      <div class="gl-section-title">
        工业区位条件
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">原料接近度</span>
          <span class="gl-value" data-role="raw-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${rawMaterial}"
          data-role="raw"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">市场接近度</span>
          <span class="gl-value" data-role="market-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${market}"
          data-role="market"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">交通条件</span>
          <span class="gl-value" data-role="transport-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${transport}"
          data-role="transport"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">劳动力条件</span>
          <span class="gl-value" data-role="labor-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${labor}"
          data-role="labor"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">能源保障</span>
          <span class="gl-value" data-role="energy-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${energy}"
          data-role="energy"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">技术与创新</span>
          <span class="gl-value" data-role="technology-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${technology}"
          data-role="technology"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">环境容量</span>
          <span class="gl-value" data-role="environment-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${environment}"
          data-role="environment"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          因素标注
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
          切换下一类型
        </button>
      </div>

      <div class="gl-result" data-role="result">
        工业区位由成本、市场、技术、环境和区域联系共同决定。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="suitability">
          区位适宜性
        </button>

        <button type="button" data-view="agglomeration">
          工业集聚
        </button>

        <button type="button" data-view="transfer">
          产业转移
        </button>

        <button type="button" data-view="impact">
          区域影响
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-industry-canvas"
          width="980"
          height="570"
          data-role="canvas"
          aria-label="工业区位、工业集聚与产业转移教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var rawInput =
        root.querySelector('[data-role="raw"]');

      var marketInput =
        root.querySelector('[data-role="market"]');

      var transportInput =
        root.querySelector('[data-role="transport"]');

      var laborInput =
        root.querySelector('[data-role="labor"]');

      var energyInput =
        root.querySelector('[data-role="energy"]');

      var technologyInput =
        root.querySelector('[data-role="technology"]');

      var environmentInput =
        root.querySelector('[data-role="environment"]');

      var rawValue =
        root.querySelector('[data-role="raw-value"]');

      var marketValue =
        root.querySelector('[data-role="market-value"]');

      var transportValue =
        root.querySelector('[data-role="transport-value"]');

      var laborValue =
        root.querySelector('[data-role="labor-value"]');

      var energyValue =
        root.querySelector('[data-role="energy-value"]');

      var technologyValue =
        root.querySelector('[data-role="technology-value"]');

      var environmentValue =
        root.querySelector('[data-role="environment-value"]');

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
        !rawInput ||
        !marketInput ||
        !transportInput ||
        !laborInput ||
        !energyInput ||
        !technologyInput ||
        !environmentInput ||
        !rawValue ||
        !marketValue ||
        !transportValue ||
        !laborValue ||
        !energyValue ||
        !technologyValue ||
        !environmentValue ||
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
          key:'raw-material',
          name:'原料导向型',
          icon:'⛏️',
          color:'#B45309',
          raw:10,
          market:4,
          transport:7,
          labor:5,
          energy:8,
          technology:4,
          environment:5,
          view:'suitability',
          ideals:{
            raw:10,
            market:4,
            transport:7,
            labor:5,
            energy:8,
            technology:4,
            environment:5
          },
          weights:{
            raw:1.7,
            market:0.5,
            transport:1,
            labor:0.5,
            energy:1.2,
            technology:0.4,
            environment:0.7
          }
        },
        {
          key:'market',
          name:'市场导向型',
          icon:'🛒',
          color:'#DC2626',
          raw:4,
          market:10,
          transport:8,
          labor:6,
          energy:5,
          technology:6,
          environment:6,
          view:'agglomeration',
          ideals:{
            raw:4,
            market:10,
            transport:8,
            labor:6,
            energy:5,
            technology:6,
            environment:6
          },
          weights:{
            raw:0.4,
            market:1.8,
            transport:1.2,
            labor:0.6,
            energy:0.5,
            technology:0.7,
            environment:0.7
          }
        },
        {
          key:'labor',
          name:'劳动力导向型',
          icon:'👷',
          color:'#0284C7',
          raw:5,
          market:6,
          transport:7,
          labor:10,
          energy:5,
          technology:5,
          environment:6,
          view:'impact',
          ideals:{
            raw:5,
            market:6,
            transport:7,
            labor:10,
            energy:5,
            technology:5,
            environment:6
          },
          weights:{
            raw:0.4,
            market:0.7,
            transport:0.9,
            labor:1.8,
            energy:0.5,
            technology:0.6,
            environment:0.7
          }
        },
        {
          key:'technology',
          name:'技术导向型',
          icon:'🧪',
          color:'#7C3AED',
          raw:3,
          market:7,
          transport:8,
          labor:8,
          energy:6,
          technology:10,
          environment:8,
          view:'transfer',
          ideals:{
            raw:3,
            market:7,
            transport:8,
            labor:8,
            energy:6,
            technology:10,
            environment:8
          },
          weights:{
            raw:0.3,
            market:0.8,
            transport:0.9,
            labor:1,
            energy:0.5,
            technology:1.9,
            environment:1
          }
        },
        {
          key:'integrated',
          name:'综合型工业',
          icon:'🏭',
          color:'#16A34A',
          raw:6,
          market:7,
          transport:7,
          labor:6,
          energy:6,
          technology:7,
          environment:6,
          view:'suitability',
          ideals:{
            raw:6,
            market:7,
            transport:8,
            labor:7,
            energy:7,
            technology:7,
            environment:7
          },
          weights:{
            raw:0.8,
            market:1,
            transport:1.2,
            labor:0.8,
            energy:0.9,
            technology:1,
            environment:1
          }
        }
      ];

      var initial = {
        scenario:'${scenario}',
        raw:${rawMaterial},
        market:${market},
        transport:${transport},
        labor:${labor},
        energy:${energy},
        technology:${technology},
        environment:${environment},
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

        var head=13;

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
          x2-head*
          Math.cos(
            angle-Math.PI/6
          ),
          y2-head*
          Math.sin(
            angle-Math.PI/6
          )
        );

        context.lineTo(
          x2-head*
          Math.cos(
            angle+Math.PI/6
          ),
          y2-head*
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
          raw:clamp(
            Number(rawInput.value) || 0,
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
          labor:clamp(
            Number(laborInput.value) || 0,
            0,
            10
          ),
          energy:clamp(
            Number(energyInput.value) || 0,
            0,
            10
          ),
          technology:clamp(
            Number(technologyInput.value) || 0,
            0,
            10
          ),
          environment:clamp(
            Number(environmentInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        rawInput.value =
          String(Math.round(value.raw));

        marketInput.value =
          String(Math.round(value.market));

        transportInput.value =
          String(Math.round(value.transport));

        laborInput.value =
          String(Math.round(value.labor));

        energyInput.value =
          String(Math.round(value.energy));

        technologyInput.value =
          String(Math.round(value.technology));

        environmentInput.value =
          String(Math.round(value.environment));
      }

      function scoreScenario(
        item,
        value
      ){
        var keys = [
          'raw',
          'market',
          'transport',
          'labor',
          'energy',
          'technology',
          'environment'
        ];

        var score=0;
        var weights=0;

        keys.forEach(
          function(key){
            var difference =
              Math.abs(
                value[key]-
                item.ideals[key]
              );

            var part =
              clamp(
                100-
                difference*14,
                0,
                100
              );

            score +=
              part*
              item.weights[key];

            weights +=
              item.weights[key];
          }
        );

        return Math.round(
          score/
          Math.max(
            0.1,
            weights
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

        var logistics =
          (
            value.raw+
            value.market+
            value.transport
          )/
          3;

        var production =
          (
            value.labor+
            value.energy+
            value.technology
          )/
          3;

        var clusterBenefit =
          clamp(
            Math.round(
              value.transport*0.24+
              value.market*0.18+
              value.labor*0.14+
              value.technology*0.28+
              value.energy*0.08+
              value.raw*0.08
            ),
            0,
            10
          );

        var congestion =
          clamp(
            Math.round(
              clusterBenefit*0.45+
              Math.max(
                0,
                7-value.environment
              )*0.55+
              Math.max(
                0,
                value.market-7
              )*0.25
            ),
            0,
            10
          );

        var transferPush =
          clamp(
            Math.round(
              (
                10-value.environment
              )*0.20+
              (
                10-value.labor
              )*0.12+
              (
                10-value.energy
              )*0.16+
              (
                10-value.market
              )*0.08+
              (
                10-value.transport
              )*0.12+
              value.technology*0.10
            ),
            0,
            10
          );

        var transferPull =
          clamp(
            Math.round(
              value.market*0.20+
              value.transport*0.20+
              value.labor*0.14+
              value.energy*0.12+
              value.technology*0.22+
              value.environment*0.12
            ),
            0,
            10
          );

        var limiting =
          '条件较均衡';

        var limitingValue=11;

        [
          {
            name:'原料',
            value:value.raw
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
            name:'劳动力',
            value:value.labor
          },
          {
            name:'能源',
            value:value.energy
          },
          {
            name:'技术',
            value:value.technology
          },
          {
            name:'环境容量',
            value:value.environment
          }
        ].forEach(
          function(factor){
            if(
              factor.value<
              limitingValue
            ){
              limitingValue =
                factor.value;

              limiting =
                factor.name;
            }
          }
        );

        return {
          suitability:suitability,
          logistics:logistics,
          production:production,
          clusterBenefit:clusterBenefit,
          congestion:congestion,
          transferPush:transferPush,
          transferPull:transferPull,
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
          '#EDE9FE'
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
          'rgba(255,255,255,0.94)',
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
          '工业区位适宜性比较',
          '在同一组区位条件下，比较五类工业区位导向的相对适宜程度。'
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
          '物流条件',
          derived.logistics.toFixed(1),
          '#0284C7',
          '原料、市场和交通'
        );

        card(
          694,
          78,
          258,
          '生产支撑',
          derived.production.toFixed(1),
          '#16A34A',
          '劳动力、能源和技术'
        );

        var ranking =
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

        ranking.sort(
          function(a,b){
            return b.score-a.score;
          }
        );

        ranking.forEach(
          function(candidate,index){
            var y =
              192+
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
              index+1,
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
          '#DDD6FE'
        );

        text(
          '工业区位导向会随技术、交通、全球分工和环境约束变化，不能机械套用单一类型。',
          490,
          536,
          10,
          '#475569',
          670,
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

      function agglomerationView(
        item,
        value,
        derived
      ){
        background(
          '工业集聚与工业联系',
          '观察共享设施、专业化协作、信息交流和拥挤成本的共同作用。'
        );

        card(
          28,
          78,
          210,
          '集聚收益',
          derived.clusterBenefit,
          '#16A34A',
          '共享设施与专业化协作'
        );

        card(
          250,
          78,
          210,
          '拥挤成本',
          derived.congestion,
          '#DC2626',
          '地价、交通与环境压力'
        );

        card(
          472,
          78,
          210,
          '技术联系',
          Math.round(value.technology),
          '#7C3AED',
          '知识与信息交流'
        );

        card(
          694,
          78,
          258,
          '交通条件',
          Math.round(value.transport),
          '#0284C7',
          '园区内外联系效率'
        );

        var centerX=490;
        var centerY=350;

        circle(
          centerX,
          centerY,
          68,
          '#FFFFFF',
          item.color
        );

        text(
          item.icon,
          centerX,
          centerY-17,
          30,
          item.color,
          800,
          'center'
        );

        text(
          '核心企业',
          centerX,
          centerY+18,
          12,
          item.color,
          860,
          'center'
        );

        var nodes = [
          {
            x:250,
            y:245,
            label:'零部件企业',
            sub:'专业化协作',
            color:'#0284C7'
          },
          {
            x:730,
            y:245,
            label:'研发机构',
            sub:'技术与人才',
            color:'#7C3AED'
          },
          {
            x:250,
            y:455,
            label:'物流服务',
            sub:'运输与仓储',
            color:'#16A34A'
          },
          {
            x:730,
            y:455,
            label:'公共设施',
            sub:'能源与处理',
            color:'#D97706'
          }
        ];

        nodes.forEach(
          function(current,index){
            node(
              current.x,
              current.y,
              62,
              current.label,
              current.color,
              current.sub
            );

            arrow(
              current.x+
              (
                current.x<centerX
                  ? 66
                  : -66
              ),
              current.y,
              centerX+
              (
                current.x<centerX
                  ? -74
                  : 74
              ),
              centerY+
              (
                index<2
                  ? -24
                  : 24
              ),
              current.color,
              2+
              derived.clusterBenefit*
              0.25
            );
          }
        );

        for(
          var particle=0;
          particle<6;
          particle+=1
        ){
          var angle =
            state.phase*
            Math.PI*
            2+
            particle*
            Math.PI/
            3;

          var orbit=112;

          circle(
            centerX+
            Math.cos(angle)*
            orbit,
            centerY+
            Math.sin(angle)*
            orbit,
            4,
            item.color,
            null
          );
        }

        if(state.showLabels){
          text(
            '共享信息、劳动力、设施和市场',
            490,
            182,
            11,
            '#5B21B6',
            820,
            'center'
          );

          text(
            '集聚收益过高并不代表无限扩张，拥挤和环境成本会同步增加。',
            490,
            520,
            10,
            '#475569',
            680,
            'center'
          );
        }
      }

      function place(
        x,
        y,
        w,
        titleValue,
        color,
        items
      ){
        box(
          x,
          y,
          w,
          196,
          16,
          '#FFFFFF',
          color
        );

        text(
          titleValue,
          x+w/2,
          y+27,
          14,
          color,
          860,
          'center'
        );

        items.forEach(
          function(item,index){
            var currentY =
              y+
              62+
              index*35;

            circle(
              x+25,
              currentY,
              9,
              color,
              null
            );

            text(
              index+1,
              x+25,
              currentY,
              8,
              '#FFFFFF',
              850,
              'center'
            );

            text(
              item,
              x+45,
              currentY,
              10,
              '#475569',
              650,
              'left'
            );
          }
        );
      }

      function transferView(
        item,
        value,
        derived
      ){
        background(
          '产业转移的推力与拉力',
          '产业转移常由成本、市场、技术、环境约束和区域政策等因素共同驱动。'
        );

        card(
          28,
          78,
          210,
          '转出推力',
          derived.transferPush,
          '#DC2626',
          '成本和环境约束示意'
        );

        card(
          250,
          78,
          210,
          '承接拉力',
          derived.transferPull,
          '#16A34A',
          '市场、交通和要素吸引'
        );

        card(
          472,
          78,
          210,
          '技术可转移性',
          Math.round(value.technology),
          '#7C3AED',
          '技术越复杂越依赖配套'
        );

        card(
          694,
          78,
          258,
          '环境容量',
          Math.round(value.environment),
          '#0284C7',
          '承接地仍需环境约束'
        );

        place(
          58,
          205,
          322,
          '产业转出地',
          '#DC2626',
          [
            '成本与地价可能上升',
            '环境约束趋严',
            '产业结构升级需求',
            '部分环节外迁'
          ]
        );

        place(
          600,
          205,
          322,
          '产业承接地',
          '#16A34A',
          [
            '劳动力与土地条件',
            '市场与交通改善',
            '基础设施与政策支持',
            '承接产业链环节'
          ]
        );

        var strength =
          clamp(
            (
              derived.transferPush+
              derived.transferPull
            )/
            20,
            0,
            1
          );

        arrow(
          398,
          303,
          582,
          303,
          item.color,
          3+
          strength*
          7
        );

        text(
          '产业转移流',
          490,
          278,
          12,
          item.color,
          850,
          'center'
        );

        for(
          var index=0;
          index<5;
          index+=1
        ){
          var progress =
            (
              state.phase+
              index/5
            )%1;

          var x =
            lerp(
              410,
              570,
              progress
            );

          var y =
            303-
            Math.sin(
              progress*
              Math.PI
            )*
            38;

          circle(
            x,
            y,
            5,
            '#FFFFFF',
            item.color
          );
        }

        box(
          58,
          430,
          864,
          82,
          14,
          '#FFFFFF',
          '#DDD6FE'
        );

        text(
          '产业转移不是简单搬迁',
          78,
          452,
          12,
          '#5B21B6',
          850,
          'left'
        );

        text(
          '研发、总部、生产、组装和服务等环节可以在不同区域重新组合，形成新的产业链空间分工。',
          78,
          478,
          10.5,
          '#475569',
          650,
          'left'
        );

        text(
          '承接地不能只追求数量，还需关注技术吸收、就业质量、资源消耗与环境风险。',
          78,
          499,
          10.5,
          '#475569',
          650,
          'left'
        );
      }

      function impactColumn(
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

      function impactView(
        item,
        value,
        derived
      ){
        background(
          '工业发展与区域影响',
          '同时观察企业、转出地和承接地，理解工业区位变化的双向影响。'
        );

        card(
          28,
          78,
          210,
          '综合适宜性',
          derived.suitability,
          item.color,
          '当前区位条件'
        );

        card(
          250,
          78,
          210,
          '集聚收益',
          derived.clusterBenefit,
          '#16A34A',
          '协作与共享'
        );

        card(
          472,
          78,
          210,
          '拥挤成本',
          derived.congestion,
          '#DC2626',
          '地价交通环境压力'
        );

        card(
          694,
          78,
          258,
          '限制因素',
          derived.limiting,
          '#7C3AED',
          '当前最低条件'
        );

        impactColumn(
          42,
          '企业与产业链',
          item.color,
          [
            '降低部分运输与协作成本',
            '共享人才、信息和基础设施',
            '可能形成创新网络',
            '也可能面临路径依赖'
          ]
        );

        impactColumn(
          353,
          '产业转出地',
          '#DC2626',
          [
            '释放土地与环境容量',
            '推动产业结构升级',
            '可能出现就业岗位流失',
            '需发展新的增长动力'
          ]
        );

        impactColumn(
          664,
          '产业承接地',
          '#16A34A',
          [
            '增加就业与财政来源',
            '带动基础设施和配套企业',
            '可能提高资源环境压力',
            '需要提升技术吸收能力'
          ]
        );
      }

      function update(
        item,
        value,
        derived
      ){
        rawValue.textContent =
          String(Math.round(value.raw));

        marketValue.textContent =
          String(Math.round(value.market));

        transportValue.textContent =
          String(Math.round(value.transport));

        laborValue.textContent =
          String(Math.round(value.labor));

        energyValue.textContent =
          String(Math.round(value.energy));

        technologyValue.textContent =
          String(Math.round(value.technology));

        environmentValue.textContent =
          String(Math.round(value.environment));

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
          '。物流条件均值为'+
          derived.logistics.toFixed(1)+
          '，生产支撑均值为'+
          derived.production.toFixed(1)+
          '；当前限制因素是'+
          derived.limiting+
          '。集聚收益为'+
          derived.clusterBenefit+
          '，拥挤成本为'+
          derived.congestion+
          '，产业转移拉力为'+
          derived.transferPull+
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
        }else if(state.view==='agglomeration'){
          agglomerationView(
            item,
            value,
            derived
          );
        }else if(state.view==='transfer'){
          transferView(
            item,
            value,
            derived
          );
        }else{
          impactView(
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
            'agglomeration',
            'transfer',
            'impact'
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
          raw:lerp(
            from.raw,
            to.raw,
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
          labor:lerp(
            from.labor,
            to.labor,
            progress
          ),
          energy:lerp(
            from.energy,
            to.energy,
            progress
          ),
          technology:lerp(
            from.technology,
            to.technology,
            progress
          ),
          environment:lerp(
            from.environment,
            to.environment,
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
        rawInput,
        marketInput,
        transportInput,
        laborInput,
        energyInput,
        technologyInput,
        environmentInput
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
                ) || 'integrated',
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

export const GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_INDUSTRY:
GeographyLabTemplate[] = [
  {
    id: 'geography-industry-location-agglomeration-transfer',
    group: '🏭 生产活动与地域联系',
    name: '工业区位因素、工业集聚与产业转移',
    emoji: '🏭',
    desc: '调节原料、市场、交通、劳动力、能源、技术和环境条件，比较工业区位导向、工业集聚效应与产业转移。',
    params: [
      {
        key: 'scenario',
        label: '初始工业区位类型',
        type: 'select',
        options: [
          {
            label: '原料导向型',
            value: 'raw-material',
          },
          {
            label: '市场导向型',
            value: 'market',
          },
          {
            label: '劳动力导向型',
            value: 'labor',
          },
          {
            label: '技术导向型',
            value: 'technology',
          },
          {
            label: '综合型工业',
            value: 'integrated',
          },
        ],
        defaultValue: 'integrated',
      },
      {
        key: 'rawMaterial',
        label: '原料接近度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示原料产地、原料体积重量和运输损耗对区位的课堂影响。',
      },
      {
        key: 'market',
        label: '市场接近度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '表示消费市场规模、产品时效性和售后服务需求。',
      },
      {
        key: 'transport',
        label: '交通条件',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '交通条件影响原料、产品、人员和信息流动。',
      },
      {
        key: 'labor',
        label: '劳动力条件',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '综合表示劳动力数量、技能、工资和稳定性。',
      },
      {
        key: 'energy',
        label: '能源保障',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '能源密集型工业通常更重视稳定、低成本能源供应。',
      },
      {
        key: 'technology',
        label: '技术与创新',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '表示科研、人才、信息网络和创新环境。',
      },
      {
        key: 'environment',
        label: '环境容量',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示环境承载、治理能力和环境准入条件。',
      },
      {
        key: 'showLabels',
        label: '显示区位因素标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildIndustryLocationHTML,
  },
]
