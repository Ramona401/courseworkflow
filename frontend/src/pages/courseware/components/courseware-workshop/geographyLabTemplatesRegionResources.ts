/**
 * geographyLabTemplatesRegionResources.ts
 *
 * 地理第40批B2：资源开发、跨区域调配与环境承载力。
 *
 * 教学目标：
 * - 理解资源禀赋、区域需求与资源开发强度之间的关系；
 * - 比较水资源、能源和矿产资源跨区域调配的供需背景与空间联系；
 * - 观察资源调配对输出地、沿线地区和输入地的双向影响；
 * - 理解资源环境承载力、技术效率、生态敏感性和区域协同的重要性。
 *
 * 教学边界：
 * - 所有资源量、需求、输送能力、承载力和生态风险均为课堂简化示意；
 * - 不对应任何真实矿区、水源地、能源基地、城市、线路或工程参数；
 * - 不用于真实资源评估、调水调能、工程选线、投资或环境决策。
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

function buildRegionResourcesHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'energy-base',
    'water-transfer',
    'mineral-development',
    'receiving-region',
    'balanced-coordination',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'balanced-coordination',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'balanced-coordination'

  const resourceEndowment = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'resourceEndowment', 7),
    ),
  )

  const demandPressure = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'demandPressure', 7),
    ),
  )

  const transferCapacity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'transferCapacity', 6),
    ),
  )

  const technologyEfficiency = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'technologyEfficiency', 6),
    ),
  )

  const ecologicalSensitivity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'ecologicalSensitivity', 6),
    ),
  )

  const regionalCoordination = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'regionalCoordination', 6),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-region-resources-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #A5B4FC;
      border-radius:18px;
      background:#FFFFFF;
      color:#1E1B4B;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(67,56,202,0.11);
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
      border-bottom:1px solid #C7D2FE;
      background:linear-gradient(
        135deg,
        #EEF2FF,
        #ECFEFF 56%,
        #F0FDF4
      );
    }

    #${rootId} .gl-title{
      color:#3730A3;
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
      border:1px solid #A5B4FC;
      border-radius:999px;
      background:#FFFFFF;
      color:#4338CA;
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
      border-right:1px solid #E0E7FF;
      background:linear-gradient(
        180deg,
        #EEF2FF,
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
        #E0E7FF 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#3730A3;
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
      background:#E0E7FF;
      color:#4338CA;
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
        #A5B4FC,
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
        #4F46E5,
        #0891B2
      );
      box-shadow:0 1px 5px rgba(67,56,202,0.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #A5B4FC;
      border-radius:9px;
      background:#FFFFFF;
      color:#4338CA;
      font-size:10.6px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#4338CA;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #4F46E5,
        #0891B2 55%,
        #16A34A
      );
      box-shadow:0 5px 13px rgba(67,56,202,0.22);
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
      border:1px solid #A5B4FC;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #EEF2FF,
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
      border:1px solid #C7D2FE;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-resources-canvas{
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
    <div style="font-size:24px;">⚡</div>

    <div>
      <div class="gl-title">
        资源开发、跨区域调配与环境承载力
      </div>

      <div class="gl-subtitle">
        比较资源禀赋、区域需求、输送能力与生态约束
      </div>
    </div>

    <div class="gl-note">
      课堂简化模型 · 不用于真实资源调配
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        区域资源情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="energy-base">
          能源输出基地
        </button>

        <button type="button" data-scenario="water-transfer">
          跨流域调水
        </button>

        <button type="button" data-scenario="mineral-development">
          矿产资源开发
        </button>

        <button type="button" data-scenario="receiving-region">
          资源输入地区
        </button>

        <button type="button" data-scenario="balanced-coordination">
          区域协调方案
        </button>
      </div>

      <div class="gl-section-title">
        资源供需与承载条件
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">资源禀赋</span>
          <span class="gl-value" data-role="endowment-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${resourceEndowment}"
          data-role="endowment"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">区域需求压力</span>
          <span class="gl-value" data-role="demand-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${demandPressure}"
          data-role="demand"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">跨区域输送能力</span>
          <span class="gl-value" data-role="transfer-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${transferCapacity}"
          data-role="transfer"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">资源利用技术效率</span>
          <span class="gl-value" data-role="technology-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${technologyEfficiency}"
          data-role="technology"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">生态敏感性</span>
          <span class="gl-value" data-role="ecology-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${ecologicalSensitivity}"
          data-role="ecology"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">区域协同水平</span>
          <span class="gl-value" data-role="coordination-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${regionalCoordination}"
          data-role="coordination"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          区域标注
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

        <button type="button" data-role="compare">
          切换下一情境
        </button>
      </div>

      <div class="gl-result" data-role="result">
        资源调配需要统筹输出地保护、沿线安全和输入地节约利用。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="endowment">
          资源供需
        </button>

        <button type="button" data-view="allocation">
          调配网络
        </button>

        <button type="button" data-view="carrying">
          环境承载
        </button>

        <button type="button" data-view="governance">
          协同治理
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-resources-canvas"
          width="980"
          height="570"
          data-role="canvas"
          aria-label="资源开发、跨区域调配与环境承载力教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var endowmentInput =
        root.querySelector('[data-role="endowment"]');

      var demandInput =
        root.querySelector('[data-role="demand"]');

      var transferInput =
        root.querySelector('[data-role="transfer"]');

      var technologyInput =
        root.querySelector('[data-role="technology"]');

      var ecologyInput =
        root.querySelector('[data-role="ecology"]');

      var coordinationInput =
        root.querySelector('[data-role="coordination"]');

      var endowmentValue =
        root.querySelector('[data-role="endowment-value"]');

      var demandValue =
        root.querySelector('[data-role="demand-value"]');

      var transferValue =
        root.querySelector('[data-role="transfer-value"]');

      var technologyValue =
        root.querySelector('[data-role="technology-value"]');

      var ecologyValue =
        root.querySelector('[data-role="ecology-value"]');

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
        !endowmentInput ||
        !demandInput ||
        !transferInput ||
        !technologyInput ||
        !ecologyInput ||
        !coordinationInput ||
        !endowmentValue ||
        !demandValue ||
        !transferValue ||
        !technologyValue ||
        !ecologyValue ||
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

      var width=canvas.width;
      var height=canvas.height;

      var scenarios=[
        {
          key:'energy-base',
          name:'能源输出基地',
          icon:'⚡',
          color:'#D97706',
          endowment:9,
          demand:4,
          transfer:8,
          technology:6,
          ecology:7,
          coordination:5,
          view:'endowment'
        },
        {
          key:'water-transfer',
          name:'跨流域调水',
          icon:'💧',
          color:'#0284C7',
          endowment:8,
          demand:7,
          transfer:7,
          technology:7,
          ecology:8,
          coordination:7,
          view:'allocation'
        },
        {
          key:'mineral-development',
          name:'矿产资源开发',
          icon:'⛏️',
          color:'#7C3AED',
          endowment:9,
          demand:6,
          transfer:6,
          technology:5,
          ecology:8,
          coordination:4,
          view:'carrying'
        },
        {
          key:'receiving-region',
          name:'资源输入地区',
          icon:'🏙️',
          color:'#DC2626',
          endowment:3,
          demand:10,
          transfer:8,
          technology:7,
          ecology:5,
          coordination:6,
          view:'allocation'
        },
        {
          key:'balanced-coordination',
          name:'区域协调方案',
          icon:'🤝',
          color:'#16A34A',
          endowment:7,
          demand:7,
          transfer:6,
          technology:8,
          ecology:6,
          coordination:9,
          view:'governance'
        }
      ];

      var initial={
        scenario:'${scenario}',
        endowment:${resourceEndowment},
        demand:${demandPressure},
        transfer:${transferCapacity},
        technology:${technologyEfficiency},
        ecology:${ecologicalSensitivity},
        coordination:${regionalCoordination},
        showLabels:${showLabels}
      };

      var state={
        scenario:initial.scenario,
        view:'endowment',
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
        var p=clamp(t,0,1);

        return p<0.5
          ? 2*p*p
          : 1-
            Math.pow(
              -2*p+2,
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
        var r=
          Math.min(
            radius,
            w/2,
            h/2
          );

        context.beginPath();
        context.moveTo(x+r,y);
        context.lineTo(x+w-r,y);
        context.quadraticCurveTo(x+w,y,x+w,y+r);
        context.lineTo(x+w,y+h-r);
        context.quadraticCurveTo(x+w,y+h,x+w-r,y+h);
        context.lineTo(x+r,y+h);
        context.quadraticCurveTo(x,y+h,x,y+h-r);
        context.lineTo(x,y+r);
        context.quadraticCurveTo(x,y,x+r,y);
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

        context.font=
          (weight || 600)+
          ' '+
          size+
          'px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif';

        context.fillStyle=
          color || '#334155';

        context.textAlign=
          align || 'left';

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
        var angle=
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
          endowment:clamp(
            Number(endowmentInput.value) || 0,
            0,
            10
          ),
          demand:clamp(
            Number(demandInput.value) || 0,
            0,
            10
          ),
          transfer:clamp(
            Number(transferInput.value) || 0,
            0,
            10
          ),
          technology:clamp(
            Number(technologyInput.value) || 0,
            0,
            10
          ),
          ecology:clamp(
            Number(ecologyInput.value) || 0,
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
        endowmentInput.value=
          String(Math.round(value.endowment));

        demandInput.value=
          String(Math.round(value.demand));

        transferInput.value=
          String(Math.round(value.transfer));

        technologyInput.value=
          String(Math.round(value.technology));

        ecologyInput.value=
          String(Math.round(value.ecology));

        coordinationInput.value=
          String(Math.round(value.coordination));
      }

      function derive(value){
        var localBalance=
          clamp(
            Math.round(
              5+
              value.endowment*0.58+
              value.technology*0.25-
              value.demand*0.62
            ),
            0,
            10
          );

        var exportPotential=
          clamp(
            Math.round(
              value.endowment*0.52+
              value.transfer*0.30+
              value.technology*0.18-
              value.demand*0.22
            ),
            0,
            10
          );

        var importNeed=
          clamp(
            Math.round(
              value.demand*0.62+
              (
                10-value.endowment
              )*0.38-
              value.technology*0.16
            ),
            0,
            10
          );

        var allocationBenefit=
          clamp(
            Math.round(
              value.transfer*0.35+
              value.coordination*0.28+
              Math.min(
                exportPotential,
                importNeed
              )*0.25+
              value.technology*0.12
            ),
            0,
            10
          );

        var carryingPressure=
          clamp(
            Math.round(
              value.demand*0.38+
              value.endowment*0.12+
              (
                10-value.technology
              )*0.20+
              value.ecology*0.30-
              value.coordination*0.16
            ),
            0,
            10
          );

        var ecologicalRisk=
          clamp(
            Math.round(
              value.ecology*0.48+
              value.endowment*0.18+
              value.transfer*0.10+
              (
                10-value.technology
              )*0.16-
              value.coordination*0.14
            ),
            0,
            10
          );

        var resourceSecurity=
          clamp(
            Math.round(
              localBalance*0.30+
              allocationBenefit*0.34+
              value.technology*0.18+
              value.coordination*0.18
            ),
            0,
            10
          );

        var sustainability=
          clamp(
            Math.round(
              resourceSecurity*0.34+
              (
                10-carryingPressure
              )*0.24+
              (
                10-ecologicalRisk
              )*0.24+
              value.coordination*0.18
            ),
            0,
            10
          );

        return {
          localBalance:localBalance,
          exportPotential:exportPotential,
          importNeed:importNeed,
          allocationBenefit:allocationBenefit,
          carryingPressure:carryingPressure,
          ecologicalRisk:ecologicalRisk,
          resourceSecurity:resourceSecurity,
          sustainability:sustainability
        };
      }

      function background(
        titleValue,
        subtitle
      ){
        var gradient=
          context.createLinearGradient(
            0,
            0,
            width,
            height
          );

        gradient.addColorStop(0,'#FFFFFF');
        gradient.addColorStop(0.58,'#F8FAFC');
        gradient.addColorStop(1,'#E0E7FF');

        context.fillStyle=gradient;
        context.fillRect(0,0,width,height);

        text(
          titleValue,
          28,
          31,
          18,
          '#3730A3',
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
          '#C7D2FE'
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

      function endowmentView(
        item,
        value,
        derived
      ){
        background(
          '资源禀赋与区域供需',
          '比较资源富集区、资源短缺区和技术效率对区域资源平衡的影响。'
        );

        card(
          28,
          78,
          210,
          '资源禀赋',
          Math.round(value.endowment),
          item.color,
          '自然资源基础'
        );

        card(
          250,
          78,
          210,
          '需求压力',
          Math.round(value.demand),
          '#DC2626',
          '人口与产业需求'
        );

        card(
          472,
          78,
          210,
          '本地平衡',
          derived.localBalance,
          '#16A34A',
          '供给、需求与效率综合'
        );

        card(
          694,
          78,
          258,
          '资源安全',
          derived.resourceSecurity,
          '#2563EB',
          '本地与外部调配共同保障'
        );

        gauge(
          190,
          326,
          derived.exportPotential/10,
          '#D97706',
          '输出潜力',
          derived.exportPotential
        );

        gauge(
          490,
          326,
          derived.importNeed/10,
          '#DC2626',
          '输入需求',
          derived.importNeed
        );

        gauge(
          790,
          326,
          derived.resourceSecurity/10,
          '#2563EB',
          '资源安全',
          derived.resourceSecurity
        );

        var supplyWidth=
          360*
          value.endowment/
          10;

        var demandWidth=
          360*
          value.demand/
          10;

        box(
          92,
          442,
          360,
          24,
          12,
          '#E2E8F0',
          null
        );

        box(
          92,
          442,
          supplyWidth,
          24,
          12,
          item.color,
          null
        );

        text(
          '资源供给',
          92,
          425,
          10,
          item.color,
          800,
          'left'
        );

        box(
          528,
          442,
          360,
          24,
          12,
          '#E2E8F0',
          null
        );

        box(
          528,
          442,
          demandWidth,
          24,
          12,
          '#DC2626',
          null
        );

        text(
          '区域需求',
          528,
          425,
          10,
          '#DC2626',
          800,
          'left'
        );

        box(
          92,
          500,
          796,
          44,
          12,
          '#FFFFFF',
          '#C7D2FE'
        );

        text(
          derived.exportPotential>
          derived.importNeed
            ? '资源供给相对充足，应控制开发强度、提高附加值并保护输出地生态。'
            : '资源需求相对突出，应推进节约利用、技术替代和跨区域调配。',
          490,
          522,
          10.5,
          '#475569',
          680,
          'center'
        );
      }

      function regionNode(
        x,
        y,
        radius,
        titleValue,
        subtitle,
        color,
        icon
      ){
        circle(
          x,
          y,
          radius,
          '#FFFFFF',
          color
        );

        text(
          icon,
          x,
          y-20,
          26,
          color,
          800,
          'center'
        );

        text(
          titleValue,
          x,
          y+10,
          11,
          color,
          850,
          'center'
        );

        text(
          subtitle,
          x,
          y+29,
          9,
          '#64748B',
          650,
          'center'
        );
      }

      function allocationView(
        item,
        value,
        derived
      ){
        background(
          '资源跨区域调配网络',
          '资源输出地、沿线节点和输入地通过输水、输电、输气与综合运输通道联系。'
        );

        card(
          28,
          78,
          210,
          '输出潜力',
          derived.exportPotential,
          '#D97706',
          '资源富集与本地需求综合'
        );

        card(
          250,
          78,
          210,
          '输入需求',
          derived.importNeed,
          '#DC2626',
          '需求与本地资源缺口'
        );

        card(
          472,
          78,
          210,
          '调配能力',
          Math.round(value.transfer),
          '#2563EB',
          '通道与枢纽能力'
        );

        card(
          694,
          78,
          258,
          '调配收益',
          derived.allocationBenefit,
          '#16A34A',
          '供需匹配与协同'
        );

        var leftX=180;
        var centerX=490;
        var rightX=800;
        var nodeY=342;

        regionNode(
          leftX,
          nodeY,
          72,
          '资源输出地',
          '资源开发与生态保护',
          '#D97706',
          '⛰️'
        );

        regionNode(
          centerX,
          nodeY,
          64,
          '沿线枢纽',
          '输送、换装与安全',
          '#2563EB',
          '🔗'
        );

        regionNode(
          rightX,
          nodeY,
          72,
          '资源输入地',
          '生产生活与节约利用',
          '#DC2626',
          '🏙️'
        );

        var flowWidth=
          2+
          derived.allocationBenefit*
          0.65;

        arrow(
          leftX+80,
          nodeY,
          centerX-72,
          nodeY,
          item.color,
          flowWidth
        );

        arrow(
          centerX+72,
          nodeY,
          rightX-80,
          nodeY,
          item.color,
          flowWidth
        );

        for(
          var index=0;
          index<8;
          index+=1
        ){
          var progress=
            (
              state.phase+
              index/8
            )%1;

          var x=
            progress<0.5
              ? lerp(
                  leftX+82,
                  centerX-72,
                  progress*2
                )
              : lerp(
                  centerX+72,
                  rightX-82,
                  (
                    progress-0.5
                  )*2
                );

          var y=
            nodeY-
            Math.sin(
              progress*
              Math.PI*
              2
            )*
            18;

          circle(
            x,
            y,
            5,
            '#FFFFFF',
            item.color
          );
        }

        if(state.showLabels){
          text(
            '资源流',
            335,
            310,
            10,
            item.color,
            820,
            'center'
          );

          text(
            '资源流',
            645,
            310,
            10,
            item.color,
            820,
            'center'
          );

          text(
            '资金、技术和生态补偿可形成反向联系',
            490,
            457,
            10,
            '#0F766E',
            760,
            'center'
          );

          arrow(
            rightX-75,
            442,
            leftX+75,
            442,
            '#0F766E',
            2.4
          );
        }

        box(
          70,
          495,
          840,
          50,
          12,
          '#FFFFFF',
          '#C7D2FE'
        );

        text(
          '跨区域调配可以缓解资源空间分布与需求不匹配，但不能替代输入地节约利用和输出地生态保护。',
          490,
          520,
          10.3,
          '#475569',
          680,
          'center'
        );
      }

      function carryingView(
        item,
        value,
        derived
      ){
        background(
          '资源环境承载力',
          '资源开发和区域需求必须控制在资源供给、环境容量和生态系统可承受范围内。'
        );

        card(
          28,
          78,
          210,
          '承载压力',
          derived.carryingPressure,
          '#DC2626',
          '需求、效率与敏感性综合'
        );

        card(
          250,
          78,
          210,
          '生态风险',
          derived.ecologicalRisk,
          '#EA580C',
          '开发强度与生态敏感性'
        );

        card(
          472,
          78,
          210,
          '技术效率',
          Math.round(value.technology),
          '#7C3AED',
          '节约、循环和替代能力'
        );

        card(
          694,
          78,
          258,
          '可持续指数',
          derived.sustainability,
          '#16A34A',
          '安全、风险与协同综合'
        );

        gauge(
          190,
          320,
          derived.carryingPressure/10,
          '#DC2626',
          '承载压力',
          derived.carryingPressure
        );

        gauge(
          490,
          320,
          derived.ecologicalRisk/10,
          '#EA580C',
          '生态风险',
          derived.ecologicalRisk
        );

        gauge(
          790,
          320,
          derived.sustainability/10,
          '#16A34A',
          '可持续性',
          derived.sustainability
        );

        var measures=[
          {
            title:'节约优先',
            desc:'降低单位产出资源消耗',
            color:'#2563EB',
            score:value.technology
          },
          {
            title:'循环利用',
            desc:'提高回收与再利用水平',
            color:'#7C3AED',
            score:(
              value.technology+
              value.coordination
            )/
            2
          },
          {
            title:'生态分区管控',
            desc:'敏感区降低开发强度',
            color:'#16A34A',
            score:10-
              derived.ecologicalRisk*
              0.55
          },
          {
            title:'需求管理',
            desc:'控制高耗能高耗水增长',
            color:'#EA580C',
            score:10-
              value.demand*
              0.45
          }
        ];

        measures.forEach(
          function(measure,index){
            var x=
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
          '#C7D2FE'
        );

        text(
          '资源环境承载力不是固定不变的，技术和治理可以改善效率，但生态底线不能无限突破。',
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
            var y=
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
          '资源调配与区域协同治理',
          '通过利益共享、生态补偿、技术协作和需求管理提高资源调配的公平性与可持续性。'
        );

        card(
          28,
          78,
          210,
          '资源安全',
          derived.resourceSecurity,
          '#2563EB',
          '本地供给与外部调配'
        );

        card(
          250,
          78,
          210,
          '调配收益',
          derived.allocationBenefit,
          item.color,
          '供需匹配与通道能力'
        );

        card(
          472,
          78,
          210,
          '区域协同',
          Math.round(value.coordination),
          '#0F766E',
          '信息、标准与补偿机制'
        );

        card(
          694,
          78,
          258,
          '可持续指数',
          derived.sustainability,
          '#16A34A',
          '安全、生态和公平综合'
        );

        governanceColumn(
          42,
          '资源输出地',
          '#D97706',
          [
            '控制资源开发强度',
            '提高加工和附加值',
            '保护生态和居民权益',
            '获得合理利益与补偿'
          ]
        );

        governanceColumn(
          353,
          '跨区域协同',
          '#4338CA',
          [
            '统一规划和安全标准',
            '建立价格与利益机制',
            '共享监测和风险信息',
            '推进技术与产业合作'
          ]
        );

        governanceColumn(
          664,
          '资源输入地',
          '#2563EB',
          [
            '坚持节约和需求管理',
            '提高资源利用效率',
            '发展循环经济与替代',
            '承担生态补偿责任'
          ]
        );

        box(
          42,
          520,
          896,
          32,
          10,
          '#EEF2FF',
          '#A5B4FC'
        );

        text(
          '资源调配不能只看输入地收益，还应关注输出地发展权、沿线安全和生态成本。',
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
        endowmentValue.textContent=
          String(Math.round(value.endowment));

        demandValue.textContent=
          String(Math.round(value.demand));

        transferValue.textContent=
          String(Math.round(value.transfer));

        technologyValue.textContent=
          String(Math.round(value.technology));

        ecologyValue.textContent=
          String(Math.round(value.ecology));

        coordinationValue.textContent=
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

        var scenarioName=
          state.scenario==='custom'
            ? '自定义资源条件'
            : item.name;

        result.textContent=
          scenarioName+
          '下，本地资源平衡为'+
          derived.localBalance+
          '，输出潜力为'+
          derived.exportPotential+
          '，输入需求为'+
          derived.importNeed+
          '，调配收益为'+
          derived.allocationBenefit+
          '，承载压力为'+
          derived.carryingPressure+
          '，可持续指数为'+
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

        var value=
          values();

        var derived=
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

        if(state.view==='endowment'){
          endowmentView(
            item,
            value,
            derived
          );
        }else if(state.view==='allocation'){
          allocationView(
            item,
            value,
            derived
          );
        }else if(state.view==='carrying'){
          carryingView(
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
        var item=
          scenarioByKey(key);

        state.scenario=item.key;
        setInputs(item);

        if(changeView){
          state.view=item.view;
        }

        state.compareIndex=
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

        var duration=5200;

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
              local/0.82,
              0,
              1
            )
          );

        state.scenario=
          local<0.5
            ? from.key
            : to.key;

        state.view=
          [
            'endowment',
            'allocation',
            'carrying',
            'governance'
          ][
            Math.floor(
              elapsed/6500
            )%4
          ];

        state.phase=
          (
            elapsed/3300
          )%1;

        setInputs({
          endowment:lerp(
            from.endowment,
            to.endowment,
            progress
          ),
          demand:lerp(
            from.demand,
            to.demand,
            progress
          ),
          transfer:lerp(
            from.transfer,
            to.transfer,
            progress
          ),
          technology:lerp(
            from.technology,
            to.technology,
            progress
          ),
          ecology:lerp(
            from.ecology,
            to.ecology,
            progress
          ),
          coordination:lerp(
            from.coordination,
            to.coordination,
            progress
          )
        });

        render();

        state.raf=
          requestAnimationFrame(
            animate
          );
      }

      function manual(){
        if(state.auto){
          stopAuto();
        }

        state.scenario='custom';
        render();
      }

      [
        endowmentInput,
        demandInput,
        transferInput,
        technologyInput,
        ecologyInput,
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
                ) ||
                'balanced-coordination',
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
                'endowment';

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

          state.raf=
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

          state.scenario=
            initial.scenario;

          state.view='endowment';

          state.showLabels=
            initial.showLabels;

          setInputs(initial);

          state.compareIndex=
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

          state.compareIndex=
            (
              state.compareIndex+1
            )%
            scenarios.length;

          var next=
            scenarios[
              state.compareIndex
            ];

          state.scenario=next.key;
          state.view=next.view;
          setInputs(next);
          render();
        }
      );

      state.compareIndex=
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

export const GEOGRAPHY_LAB_TEMPLATES_REGION_RESOURCES:
GeographyLabTemplate[] = [
  {
    id: 'geography-resource-development-allocation-carrying-capacity',
    group: '🗺️ 区域发展与资源环境',
    name: '资源开发、跨区域调配与环境承载力',
    emoji: '⚡',
    desc: '调节资源禀赋、需求、输送能力、技术效率、生态敏感性和区域协同，观察资源供需、跨区域调配、承载压力与协同治理。',
    params: [
      {
        key: 'scenario',
        label: '初始资源情境',
        type: 'select',
        options: [
          {
            label: '能源输出基地',
            value: 'energy-base',
          },
          {
            label: '跨流域调水',
            value: 'water-transfer',
          },
          {
            label: '矿产资源开发',
            value: 'mineral-development',
          },
          {
            label: '资源输入地区',
            value: 'receiving-region',
          },
          {
            label: '区域协调方案',
            value: 'balanced-coordination',
          },
        ],
        defaultValue: 'balanced-coordination',
      },
      {
        key: 'resourceEndowment',
        label: '资源禀赋',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '表示水、能源或矿产等资源的区域富集程度。',
      },
      {
        key: 'demandPressure',
        label: '区域需求压力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '综合表示人口、产业和城镇发展形成的资源需求。',
      },
      {
        key: 'transferCapacity',
        label: '跨区域输送能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示输水、输电、输气、运输通道及枢纽能力。',
      },
      {
        key: 'technologyEfficiency',
        label: '资源利用技术效率',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '包括节水节能、循环利用、深加工和替代技术。',
      },
      {
        key: 'ecologicalSensitivity',
        label: '生态敏感性',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '生态越敏感，资源开发和线路建设越需严格控制。',
      },
      {
        key: 'regionalCoordination',
        label: '区域协同水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '包括统一规划、利益共享、生态补偿和风险协作。',
      },
      {
        key: 'showLabels',
        label: '显示区域与资源流标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildRegionResourcesHTML,
  },
]
