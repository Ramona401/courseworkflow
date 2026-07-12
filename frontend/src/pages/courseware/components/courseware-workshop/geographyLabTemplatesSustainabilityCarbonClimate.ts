/**
 * geographyLabTemplatesSustainabilityCarbonClimate.ts
 *
 * 地理第42批B1：
 *   碳循环、温室效应与全球气候变化。
 *
 * 教学目标：
 * - 认识大气、植被、土壤、海洋和化石燃料等主要碳储库；
 * - 理解光合作用、呼吸作用、分解、海气交换和燃烧等主要碳通量；
 * - 比较化石燃料使用、森林破坏和生态恢复对碳循环平衡的影响；
 * - 理解自然温室效应对地球温度的重要作用；
 * - 理解温室气体增加会改变地表长波辐射的吸收与返回过程；
 * - 比较高排放、生态恢复和低碳转型等不同课堂情境；
 * - 建立减排、增汇和适应需要协同推进的可持续发展意识。
 *
 * 教学边界：
 * - 所有碳储量、通量、温室效应、增温和风险数值均为相对教学指数；
 * - 本模型不对应真实二氧化碳浓度、全球平均温度或具体排放情景；
 * - 不考虑完整气候系统反馈、区域差异和长期自然波动；
 * - 不用于真实气候预测、碳核算、政策评估或投资决策。
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

function buildCarbonClimateHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'balanced-cycle',
    'fossil-growth',
    'deforestation',
    'ecosystem-restoration',
    'low-carbon-transition',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'fossil-growth',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'fossil-growth'

  const fossilFuelEmissions = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'fossilFuelEmissions', 8),
    ),
  )

  const deforestation = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'deforestation', 5),
    ),
  )

  const vegetationUptake = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'vegetationUptake', 6),
    ),
  )

  const oceanAbsorption = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'oceanAbsorption', 6),
    ),
  )

  const climateSensitivity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'climateSensitivity', 6),
    ),
  )

  const mitigation = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'mitigation', 4),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-carbon-climate-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #86EFAC;
      border-radius:18px;
      background:#FFFFFF;
      color:#052E16;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(22,163,74,.12);
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
        #ECFDF5 48%,
        #EFF6FF
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
      border-right:1px solid #BBF7D0;
      background:linear-gradient(
        180deg,
        #F0FDF4,
        #ECFDF5 58%,
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
      font-size:11.3px;
      font-weight:730;
    }

    #${rootId} .gl-value{
      min-width:44px;
      padding:3px 7px;
      border-radius:999px;
      background:#DCFCE7;
      color:#15803D;
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
        #86EFAC,
        #38BDF8,
        #FBBF24
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
        #0284C7
      );
      box-shadow:0 1px 5px rgba(22,163,74,.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #86EFAC;
      border-radius:9px;
      background:#FFFFFF;
      color:#15803D;
      font-size:10.5px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#15803D;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #16A34A,
        #0F766E 56%,
        #0284C7
      );
      box-shadow:0 5px 13px rgba(22,163,74,.22);
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
      border:1px solid #BBF7D0;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-carbon-climate-canvas{
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
    <div style="font-size:24px;">🌍</div>

    <div>
      <div class="gl-title">
        碳循环、温室效应与全球气候变化
      </div>

      <div class="gl-subtitle">
        观察碳储库与通量，比较人类排放、陆海碳汇、温室效应和低碳转型路径
      </div>
    </div>

    <div class="gl-note">
      相对教学指数 · 不用于真实气候预测
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        碳循环与发展情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="balanced-cycle">
          自然相对平衡
        </button>

        <button type="button" data-scenario="fossil-growth">
          化石能源增长
        </button>

        <button type="button" data-scenario="deforestation">
          森林破坏
        </button>

        <button type="button" data-scenario="ecosystem-restoration">
          生态系统恢复
        </button>

        <button type="button" data-scenario="low-carbon-transition">
          低碳转型
        </button>
      </div>

      <div class="gl-section-title">
        碳源、碳汇与气候参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">化石燃料排放</span>
          <span class="gl-value" data-role="fossil-value">8</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${fossilFuelEmissions}"
          data-role="fossil"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">森林破坏程度</span>
          <span class="gl-value" data-role="deforestation-value">5</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${deforestation}"
          data-role="deforestation"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">植被吸收能力</span>
          <span class="gl-value" data-role="vegetation-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${vegetationUptake}"
          data-role="vegetation"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">海洋吸收能力</span>
          <span class="gl-value" data-role="ocean-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${oceanAbsorption}"
          data-role="ocean"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">气候敏感程度</span>
          <span class="gl-value" data-role="sensitivity-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${climateSensitivity}"
          data-role="sensitivity"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">减排与转型力度</span>
          <span class="gl-value" data-role="mitigation-value">4</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${mitigation}"
          data-role="mitigation"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          储库与通量标注
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
        自然碳循环包含大量双向交换；当人类新增碳源长期超过陆地和海洋吸收能力时，大气碳负荷会增加。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="cycle">
          碳循环
        </button>

        <button type="button" data-view="greenhouse">
          温室效应
        </button>

        <button type="button" data-view="trend">
          情境变化
        </button>

        <button type="button" data-view="pathway">
          减排与增汇
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-carbon-climate-canvas"
          width="1000"
          height="570"
          data-role="canvas"
          aria-label="碳循环温室效应与全球气候变化教学示意图"
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

      var deforestationInput =
        root.querySelector('[data-role="deforestation"]');

      var vegetationInput =
        root.querySelector('[data-role="vegetation"]');

      var oceanInput =
        root.querySelector('[data-role="ocean"]');

      var sensitivityInput =
        root.querySelector('[data-role="sensitivity"]');

      var mitigationInput =
        root.querySelector('[data-role="mitigation"]');

      var fossilValue =
        root.querySelector('[data-role="fossil-value"]');

      var deforestationValue =
        root.querySelector('[data-role="deforestation-value"]');

      var vegetationValue =
        root.querySelector('[data-role="vegetation-value"]');

      var oceanValue =
        root.querySelector('[data-role="ocean-value"]');

      var sensitivityValue =
        root.querySelector('[data-role="sensitivity-value"]');

      var mitigationValue =
        root.querySelector('[data-role="mitigation-value"]');

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
        !deforestationInput ||
        !vegetationInput ||
        !oceanInput ||
        !sensitivityInput ||
        !mitigationInput ||
        !fossilValue ||
        !deforestationValue ||
        !vegetationValue ||
        !oceanValue ||
        !sensitivityValue ||
        !mitigationValue ||
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
          key:'balanced-cycle',
          name:'自然相对平衡',
          fossil:1,
          deforestation:1,
          vegetation:8,
          ocean:7,
          sensitivity:5,
          mitigation:3,
          view:'cycle',
          color:'#16A34A'
        },
        {
          key:'fossil-growth',
          name:'化石能源增长',
          fossil:9,
          deforestation:4,
          vegetation:5,
          ocean:6,
          sensitivity:7,
          mitigation:2,
          view:'trend',
          color:'#DC2626'
        },
        {
          key:'deforestation',
          name:'森林破坏',
          fossil:6,
          deforestation:9,
          vegetation:2,
          ocean:6,
          sensitivity:7,
          mitigation:2,
          view:'cycle',
          color:'#D97706'
        },
        {
          key:'ecosystem-restoration',
          name:'生态系统恢复',
          fossil:5,
          deforestation:1,
          vegetation:10,
          ocean:7,
          sensitivity:6,
          mitigation:6,
          view:'pathway',
          color:'#0F766E'
        },
        {
          key:'low-carbon-transition',
          name:'低碳转型',
          fossil:2,
          deforestation:1,
          vegetation:9,
          ocean:7,
          sensitivity:6,
          mitigation:10,
          view:'pathway',
          color:'#2563EB'
        }
      ];

      var initial={
        scenario:'${scenario}',
        fossil:${fossilFuelEmissions},
        deforestation:${deforestation},
        vegetation:${vegetationUptake},
        ocean:${oceanAbsorption},
        sensitivity:${climateSensitivity},
        mitigation:${mitigation},
        showLabels:${showLabels}
      };

      var state={
        scenario:initial.scenario,
        view:'cycle',
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
          fossil:clamp(
            Number(fossilInput.value) || 0,
            0,
            10
          ),
          deforestation:clamp(
            Number(deforestationInput.value) || 0,
            0,
            10
          ),
          vegetation:clamp(
            Number(vegetationInput.value) || 0,
            0,
            10
          ),
          ocean:clamp(
            Number(oceanInput.value) || 0,
            0,
            10
          ),
          sensitivity:clamp(
            Number(sensitivityInput.value) || 0,
            0,
            10
          ),
          mitigation:clamp(
            Number(mitigationInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        fossilInput.value=
          String(Math.round(value.fossil));

        deforestationInput.value=
          String(Math.round(value.deforestation));

        vegetationInput.value=
          String(Math.round(value.vegetation));

        oceanInput.value=
          String(Math.round(value.ocean));

        sensitivityInput.value=
          String(Math.round(value.sensitivity));

        mitigationInput.value=
          String(Math.round(value.mitigation));
      }

      function derive(value){
        var humanSource=
          clamp(
            value.fossil*.72+
            value.deforestation*.42,
            0,
            12
          );

        var landSink=
          clamp(
            value.vegetation*.46+
            value.mitigation*.08-
            value.deforestation*.18,
            0,
            10
          );

        var oceanSink=
          clamp(
            value.ocean*.40+
            value.mitigation*.04,
            0,
            10
          );

        var netAtmosphere=
          clamp(
            humanSource-
            landSink*.48-
            oceanSink*.40-
            value.mitigation*.24,
            -5,
            10
          );

        var atmosphericLoad=
          clamp(
            Math.round(
              4+
              value.fossil*.28+
              value.deforestation*.22+
              Math.max(
                0,
                netAtmosphere
              )*.38-
              value.mitigation*.20
            ),
            0,
            10
          );

        var greenhouseStrength=
          clamp(
            Math.round(
              atmosphericLoad*.64+
              value.sensitivity*.36
            ),
            0,
            10
          );

        var warmingPressure=
          clamp(
            Math.round(
              greenhouseStrength*.58+
              value.sensitivity*.25+
              value.deforestation*.12-
              value.mitigation*.18
            ),
            0,
            10
          );

        var oceanPressure=
          clamp(
            Math.round(
              atmosphericLoad*.35+
              value.fossil*.28+
              value.ocean*.12-
              value.mitigation*.18
            ),
            0,
            10
          );

        var ecosystemStress=
          clamp(
            Math.round(
              warmingPressure*.36+
              value.deforestation*.34+
              (
                10-value.vegetation
              )*.22+
              value.fossil*.08
            ),
            0,
            10
          );

        var transitionScore=
          clamp(
            Math.round(
              value.mitigation*.44+
              value.vegetation*.24+
              value.ocean*.10+
              (
                10-value.fossil
              )*.22
            ),
            0,
            10
          );

        var residualPressure=
          clamp(
            Math.round(
              warmingPressure-
              value.mitigation*.34-
              value.vegetation*.12
            ),
            0,
            10
          );

        return {
          humanSource:humanSource,
          landSink:landSink,
          oceanSink:oceanSink,
          netAtmosphere:netAtmosphere,
          atmosphericLoad:atmosphericLoad,
          greenhouseStrength:greenhouseStrength,
          warmingPressure:warmingPressure,
          oceanPressure:oceanPressure,
          ecosystemStress:ecosystemStress,
          transitionScore:transitionScore,
          residualPressure:residualPressure
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
        gradient.addColorStop(.55,'#F0FDF4');
        gradient.addColorStop(1,'#DBEAFE');

        context.fillStyle=gradient;
        context.fillRect(0,0,width,height);

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
          'rgba(255,255,255,.95)',
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
          9.4,
          '#64748B',
          600,
          'left'
        );
      }

      function reservoir(
        x,
        y,
        w,
        h,
        titleValue,
        amount,
        color,
        subtitle
      ){
        box(
          x,
          y,
          w,
          h,
          16,
          '#FFFFFF',
          color
        );

        context.save();
        context.globalAlpha=.16+amount*.045;
        context.fillStyle=color;
        context.fillRect(
          x+1,
          y+1,
          w-2,
          h-2
        );
        context.restore();

        text(
          titleValue,
          x+w/2,
          y+25,
          13,
          color,
          880,
          'center'
        );

        text(
          amount.toFixed(1),
          x+w/2,
          y+53,
          20,
          color,
          900,
          'center'
        );

        text(
          subtitle,
          x+w/2,
          y+h-18,
          9.2,
          '#64748B',
          650,
          'center'
        );
      }

      function flowLabel(
        label,
        x,
        y,
        color,
        value
      ){
        if(!state.showLabels)return;

        box(
          x-46,
          y-12,
          92,
          24,
          8,
          'rgba(255,255,255,.92)',
          color
        );

        text(
          label+' '+value.toFixed(1),
          x,
          y,
          9.2,
          color,
          820,
          'center'
        );
      }

      function cycleView(
        item,
        value,
        derived
      ){
        background(
          '碳储库、自然通量与人类新增碳源',
          '自然碳循环包含持续双向交换；人类活动会增加新的碳源或削弱陆地碳汇。'
        );

        card(
          28,
          78,
          210,
          '人类新增碳源',
          derived.humanSource.toFixed(1),
          '#DC2626',
          '化石燃料与土地利用'
        );

        card(
          250,
          78,
          210,
          '陆地碳汇',
          derived.landSink.toFixed(1),
          '#16A34A',
          '植被吸收与生态恢复'
        );

        card(
          472,
          78,
          210,
          '海洋碳汇',
          derived.oceanSink.toFixed(1),
          '#0284C7',
          '海气交换与海洋吸收'
        );

        card(
          694,
          78,
          258,
          '大气净变化',
          derived.netAtmosphere.toFixed(1),
          item.color,
          '碳源减去陆海吸收'
        );

        reservoir(
          397,
          177,
          206,
          100,
          '大气碳储库',
          derived.atmosphericLoad,
          '#7C3AED',
          '温室气体所在的重要储库'
        );

        reservoir(
          70,
          329,
          206,
          112,
          '植被碳储库',
          clamp(
            value.vegetation-
            value.deforestation*.42+
            value.mitigation*.18,
            0,
            10
          ),
          '#16A34A',
          '森林、草地和农作物'
        );

        reservoir(
          397,
          393,
          206,
          112,
          '土壤碳储库',
          clamp(
            6+
            value.vegetation*.22-
            value.deforestation*.35,
            0,
            10
          ),
          '#A16207',
          '枯落物、腐殖质与土壤有机碳'
        );

        reservoir(
          724,
          329,
          206,
          112,
          '海洋碳储库',
          clamp(
            5+
            value.ocean*.38+
            derived.oceanSink*.18,
            0,
            10
          ),
          '#0284C7',
          '表层与深层海洋'
        );

        reservoir(
          70,
          177,
          206,
          100,
          '化石燃料储库',
          clamp(
            10-value.fossil*.68,
            0,
            10
          ),
          '#475569',
          '煤、石油和天然气'
        );

        curvedArrow(
          275,
          213,
          330,
          156,
          397,
          213,
          '#DC2626',
          2.5+
          value.fossil*.34
        );

        flowLabel(
          '燃烧排放',
          334,
          168,
          '#DC2626',
          value.fossil
        );

        curvedArrow(
          438,
          277,
          320,
          294,
          243,
          329,
          '#16A34A',
          2.5+
          value.vegetation*.26
        );

        flowLabel(
          '光合作用',
          330,
          298,
          '#16A34A',
          derived.landSink
        );

        curvedArrow(
          250,
          329,
          327,
          276,
          457,
          277,
          '#B45309',
          2.5+
          value.deforestation*.22
        );

        flowLabel(
          '呼吸与分解',
          340,
          264,
          '#B45309',
          4+
          value.deforestation*.25
        );

        arrow(
          173,
          441,
          397,
          448,
          '#A16207',
          3
        );

        flowLabel(
          '枯落与入土',
          333,
          452,
          '#A16207',
          value.vegetation*.62
        );

        curvedArrow(
          603,
          220,
          665,
          245,
          724,
          329,
          '#0284C7',
          2.5+
          derived.oceanSink*.28
        );

        flowLabel(
          '海洋吸收',
          676,
          272,
          '#0284C7',
          derived.oceanSink
        );

        curvedArrow(
          757,
          329,
          681,
          273,
          592,
          245,
          '#0E7490',
          2.3
        );

        flowLabel(
          '海洋释放',
          674,
          247,
          '#0E7490',
          3.2
        );

        curvedArrow(
          122,
          329,
          99,
          301,
          163,
          277,
          '#D97706',
          2.5+
          value.deforestation*.32
        );

        flowLabel(
          '森林破坏',
          105,
          300,
          '#D97706',
          value.deforestation
        );

        for(
          var particle=0;
          particle<8;
          particle+=1
        ){
          var angle=
            state.phase*
            Math.PI*
            2+
            particle*
            Math.PI/
            4;

          circle(
            500+
            Math.cos(angle)*
            125,
            298+
            Math.sin(angle)*
            48,
            4,
            item.color,
            '#FFFFFF',
            1
          );
        }

        box(
          70,
          526,
          860,
          28,
          10,
          '#FFFFFF',
          '#BBF7D0'
        );

        text(
          '自然循环中的碳源与碳汇大体相互联系；长期新增碳源会使大气碳储库增加。',
          500,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function greenhouseView(
        item,
        value,
        derived
      ){
        background(
          '自然温室效应与增强的温室效应',
          '大气吸收部分地表长波辐射并向各方向重新辐射；温室气体增加会增强这一过程。'
        );

        card(
          28,
          78,
          210,
          '大气碳负荷',
          derived.atmosphericLoad,
          '#7C3AED',
          '相对教学指数'
        );

        card(
          250,
          78,
          210,
          '温室效应强度',
          derived.greenhouseStrength,
          '#DC2626',
          '长波吸收与返回过程'
        );

        card(
          472,
          78,
          210,
          '增温压力',
          derived.warmingPressure,
          '#EA580C',
          '气候敏感度共同作用'
        );

        card(
          694,
          78,
          258,
          '减排后剩余压力',
          derived.residualPressure,
          item.color,
          '减排增汇后的课堂结果'
        );

        var earthX=500;
        var earthY=456;
        var earthR=164;

        var space=
          context.createLinearGradient(
            0,
            160,
            0,
            520
          );

        space.addColorStop(0,'#0F172A');
        space.addColorStop(1,'#1E3A8A');

        context.fillStyle=space;
        context.fillRect(
          46,
          169,
          908,
          348
        );

        var earth=
          context.createRadialGradient(
            earthX-55,
            earthY-70,
            12,
            earthX,
            earthY,
            earthR
          );

        earth.addColorStop(0,'#86EFAC');
        earth.addColorStop(.48,'#22C55E');
        earth.addColorStop(.49,'#38BDF8');
        earth.addColorStop(1,'#075985');

        circle(
          earthX,
          earthY,
          earthR,
          earth,
          '#E0F2FE',
          3
        );

        var atmosphereAlpha=
          .12+
          derived.atmosphericLoad*.055;

        context.save();
        context.globalAlpha=atmosphereAlpha;
        context.strokeStyle='#A78BFA';
        context.lineWidth=
          20+
          derived.atmosphericLoad*2;

        context.beginPath();
        context.arc(
          earthX,
          earthY,
          earthR+30,
          Math.PI,
          Math.PI*2
        );
        context.stroke();
        context.restore();

        circle(
          140,
          235,
          43,
          '#FACC15',
          '#FEF3C7',
          4
        );

        for(
          var ray=0;
          ray<6;
          ray+=1
        ){
          var startY=195+ray*39;

          arrow(
            188,
            startY,
            390,
            305+ray*23,
            '#FDE047',
            3.2
          );
        }

        for(
          var longwave=0;
          longwave<5;
          longwave+=1
        ){
          var offset=
            (
              longwave-2
            )*
            58;

          curvedArrow(
            earthX+offset,
            326+
            Math.abs(offset)*.08,
            earthX+offset*.9,
            245,
            earthX+offset*.78,
            191,
            '#FB923C',
            2.6
          );
        }

        var returnCount=
          2+
          Math.round(
            derived.greenhouseStrength/2.4
          );

        for(
          var back=0;
          back<returnCount;
          back+=1
        ){
          var ratio=
            returnCount<=1
              ? .5
              : back/
                (
                  returnCount-1
                );

          var startX=
            350+
            ratio*
            300;

          curvedArrow(
            startX,
            226,
            startX+
            (
              ratio-.5
            )*
            42,
            286,
            420+
            ratio*
            160,
            353,
            '#F43F5E',
            2+
            derived.greenhouseStrength*.18
          );
        }

        var moleculeCount=
          10+
          derived.atmosphericLoad*2;

        for(
          var molecule=0;
          molecule<moleculeCount;
          molecule+=1
        ){
          var ratio=
            molecule/
            Math.max(
              1,
              moleculeCount-1
            );

          var angle=
            Math.PI+
            ratio*
            Math.PI;

          var radius=
            earthR+
            30+
            (
              molecule%3
            )*
            13;

          circle(
            earthX+
            Math.cos(angle)*
            radius,
            earthY+
            Math.sin(angle)*
            radius,
            4,
            molecule%2===0
              ? '#C4B5FD'
              : '#FCA5A5',
            '#FFFFFF',
            1
          );
        }

        if(state.showLabels){
          text(
            '太阳短波辐射',
            274,
            230,
            10,
            '#FDE047',
            850,
            'center'
          );

          text(
            '地表吸收并升温',
            500,
            405,
            10,
            '#FFFFFF',
            850,
            'center'
          );

          text(
            '地表长波辐射',
            500,
            285,
            10,
            '#FB923C',
            850,
            'center'
          );

          text(
            '大气吸收并向下返回部分长波辐射',
            500,
            207,
            10,
            '#FCA5A5',
            850,
            'center'
          );

          text(
            '其余能量继续向外辐射',
            780,
            205,
            9.5,
            '#E0E7FF',
            760,
            'center'
          );
        }

        box(
          66,
          457,
          205,
          48,
          11,
          'rgba(255,255,255,.92)',
          '#C4B5FD'
        );

        text(
          '自然温室效应',
          168,
          474,
          10.5,
          '#6D28D9',
          850,
          'center'
        );

        text(
          '使地球保持适宜温度',
          168,
          493,
          9.3,
          '#475569',
          680,
          'center'
        );

        box(
          729,
          457,
          205,
          48,
          11,
          'rgba(255,255,255,.92)',
          '#FCA5A5'
        );

        text(
          '增强的温室效应',
          831,
          474,
          10.5,
          '#DC2626',
          850,
          'center'
        );

        text(
          '改变地球能量收支',
          831,
          493,
          9.3,
          '#475569',
          680,
          'center'
        );

        box(
          66,
          526,
          868,
          28,
          10,
          '#FFFFFF',
          '#BBF7D0'
        );

        text(
          '温室效应本身是自然过程；人类活动增加温室气体后，温室效应可能被增强。',
          500,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function drawAxes(
        x,
        y,
        w,
        h,
        yLabel
      ){
        line(
          x,
          y,
          x,
          y+h,
          '#64748B',
          2
        );

        line(
          x,
          y+h,
          x+w,
          y+h,
          '#64748B',
          2
        );

        for(
          var step=0;
          step<=5;
          step+=1
        ){
          var py=
            y+
            h-
            step*
            h/5;

          line(
            x,
            py,
            x+w,
            py,
            '#E2E8F0',
            1,
            [4,5]
          );

          text(
            String(step*2),
            x-12,
            py,
            9,
            '#64748B',
            650,
            'right'
          );
        }

        text(
          yLabel,
          x-36,
          y+h/2,
          9.5,
          '#475569',
          760,
          'center'
        );

        text(
          '教学时间步',
          x+w/2,
          y+h+25,
          9.5,
          '#475569',
          760,
          'center'
        );
      }

      function drawTrendLine(
        x,
        y,
        w,
        h,
        color,
        lineWidth,
        valueAt
      ){
        context.save();
        context.strokeStyle=color;
        context.lineWidth=lineWidth;
        context.lineCap='round';
        context.lineJoin='round';
        context.beginPath();

        for(
          var step=0;
          step<=40;
          step+=1
        ){
          var ratio=step/40;
          var value=
            clamp(
              valueAt(ratio),
              0,
              10
            );

          var px=x+ratio*w;
          var py=y+h-value/10*h;

          if(step===0){
            context.moveTo(px,py);
          }else{
            context.lineTo(px,py);
          }
        }

        context.stroke();
        context.restore();
      }

      function trendView(
        item,
        value,
        derived
      ){
        background(
          '不同排放与碳汇情境下的变化趋势',
          '趋势图使用相对教学指数，只用于比较方向、速度和转型时点。'
        );

        card(
          28,
          78,
          210,
          '大气碳负荷',
          derived.atmosphericLoad,
          '#7C3AED',
          '碳源与碳汇综合结果'
        );

        card(
          250,
          78,
          210,
          '增温压力',
          derived.warmingPressure,
          '#DC2626',
          '温室效应与敏感度'
        );

        card(
          472,
          78,
          210,
          '生态系统压力',
          derived.ecosystemStress,
          '#D97706',
          '增温和森林破坏综合'
        );

        card(
          694,
          78,
          258,
          '转型指数',
          derived.transitionScore,
          item.color,
          '减排、增汇和能源替代'
        );

        var chartX=86;
        var chartY=190;
        var chartW=650;
        var chartH=300;

        box(
          chartX-28,
          chartY-17,
          chartW+56,
          chartH+52,
          14,
          '#FFFFFF',
          '#BBF7D0'
        );

        drawAxes(
          chartX,
          chartY,
          chartW,
          chartH,
          '压力指数'
        );

        var currentSlope=
          clamp(
            derived.netAtmosphere*.58+
            value.sensitivity*.16,
            -3,
            7
          );

        drawTrendLine(
          chartX,
          chartY,
          chartW,
          chartH,
          '#DC2626',
          3.5,
          function(ratio){
            return clamp(
              2.2+
              ratio*
              (
                3+
                value.fossil*.48+
                value.deforestation*.24
              ),
              0,
              10
            );
          }
        );

        drawTrendLine(
          chartX,
          chartY,
          chartW,
          chartH,
          '#D97706',
          3,
          function(ratio){
            return clamp(
              2.2+
              ratio*
              (
                1.8+
                value.fossil*.25+
                value.deforestation*.17-
                value.vegetation*.11
              ),
              0,
              10
            );
          }
        );

        drawTrendLine(
          chartX,
          chartY,
          chartW,
          chartH,
          '#16A34A',
          3.5,
          function(ratio){
            var early=
              2.2+
              ratio*
              (
                1.3+
                value.fossil*.18
              );

            var transitionEffect=
              Math.max(
                0,
                ratio-.34
              )*
              (
                value.mitigation*.68+
                value.vegetation*.24
              );

            return clamp(
              early-transitionEffect,
              0,
              10
            );
          }
        );

        drawTrendLine(
          chartX,
          chartY,
          chartW,
          chartH,
          item.color,
          4,
          function(ratio){
            var curve=
              2.2+
              ratio*
              (
                1.8+
                currentSlope
              );

            var lateMitigation=
              Math.max(
                0,
                ratio-.48
              )*
              value.mitigation*.58;

            return clamp(
              curve-lateMitigation,
              0,
              10
            );
          }
        );

        line(
          chartX+
          chartW*.48,
          chartY,
          chartX+
          chartW*.48,
          chartY+
          chartH,
          '#2563EB',
          1.5,
          [7,6]
        );

        if(state.showLabels){
          text(
            '转型措施逐步增强',
            chartX+
            chartW*.48,
            chartY+18,
            9.5,
            '#2563EB',
            820,
            'center'
          );
        }

        var legendX=780;

        box(
          legendX,
          190,
          170,
          300,
          14,
          '#FFFFFF',
          '#BBF7D0'
        );

        text(
          '情境比较',
          legendX+18,
          216,
          12,
          '#166534',
          860,
          'left'
        );

        var legends=[
          {
            label:'高排放增长',
            color:'#DC2626'
          },
          {
            label:'有限控制',
            color:'#D97706'
          },
          {
            label:'低碳转型',
            color:'#16A34A'
          },
          {
            label:'当前自定义',
            color:item.color
          }
        ];

        legends.forEach(
          function(legend,index){
            var y=
              255+
              index*
              48;

            line(
              legendX+20,
              y,
              legendX+66,
              y,
              legend.color,
              4
            );

            text(
              legend.label,
              legendX+78,
              y,
              10,
              '#475569',
              760,
              'left'
            );
          }
        );

        line(
          legendX+20,
          445,
          legendX+150,
          445,
          '#E2E8F0',
          1
        );

        text(
          '重要认识',
          legendX+18,
          463,
          10.5,
          '#166534',
          850,
          'left'
        );

        text(
          '越早转型',
          legendX+18,
          485,
          9.3,
          '#475569',
          680,
          'left'
        );

        text(
          '累积压力越低',
          legendX+18,
          503,
          9.3,
          '#475569',
          680,
          'left'
        );

        box(
          58,
          526,
          892,
          28,
          10,
          '#FFFFFF',
          '#BBF7D0'
        );

        text(
          '气候变化与累积排放有关，因此转型时间、减排速度和碳汇保护都很重要。',
          504,
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

      function pathwayView(
        item,
        value,
        derived
      ){
        background(
          '减排、增汇、适应与可持续发展协同',
          '降低新增碳源是核心，保护和恢复生态系统可增强碳汇，同时还需要适应已经出现的风险。'
        );

        card(
          28,
          78,
          210,
          '人类新增碳源',
          derived.humanSource.toFixed(1),
          '#DC2626',
          '能源和土地利用变化'
        );

        card(
          250,
          78,
          210,
          '陆海总碳汇',
          (
            derived.landSink+
            derived.oceanSink
          ).toFixed(1),
          '#16A34A',
          '生态系统和海洋吸收'
        );

        card(
          472,
          78,
          210,
          '剩余气候压力',
          derived.residualPressure,
          '#D97706',
          '采取措施后仍可能存在'
        );

        card(
          694,
          78,
          258,
          '低碳转型指数',
          derived.transitionScore,
          item.color,
          '能源、生态与治理综合'
        );

        pathwayCard(
          58,
          188,
          410,
          '能源结构转型',
          '提高低碳能源比例，减少对高碳化石能源的依赖。',
          '#2563EB',
          (
            10-value.fossil+
            value.mitigation
          )/2
        );

        pathwayCard(
          514,
          188,
          410,
          '提高能源与资源效率',
          '用更少能源和资源提供相同或更高水平的服务。',
          '#7C3AED',
          value.mitigation*.88
        );

        pathwayCard(
          58,
          310,
          410,
          '保护和恢复生态系统',
          '减少森林破坏并恢复森林、草地、湿地和土壤。',
          '#16A34A',
          (
            value.vegetation+
            (
              10-value.deforestation
            )
          )/2
        );

        pathwayCard(
          514,
          310,
          410,
          '城市与交通低碳转型',
          '优化空间结构、公共交通、建筑能效和循环利用。',
          '#0F766E',
          value.mitigation*.82
        );

        box(
          58,
          432,
          866,
          82,
          14,
          '#FFFFFF',
          '#BBF7D0'
        );

        text(
          '协同关系',
          80,
          452,
          12,
          '#166534',
          860,
          'left'
        );

        box(
          105,
          469,
          190,
          28,
          9,
          '#FEE2E2',
          '#FCA5A5'
        );

        text(
          '减少碳源',
          200,
          483,
          10,
          '#B91C1C',
          850,
          'center'
        );

        arrow(
          318,
          483,
          401,
          483,
          '#64748B',
          3
        );

        box(
          424,
          469,
          190,
          28,
          9,
          '#DCFCE7',
          '#86EFAC'
        );

        text(
          '保护和增强碳汇',
          519,
          483,
          10,
          '#15803D',
          850,
          'center'
        );

        arrow(
          637,
          483,
          720,
          483,
          '#64748B',
          3
        );

        box(
          743,
          469,
          140,
          28,
          9,
          '#DBEAFE',
          '#93C5FD'
        );

        text(
          '降低累积压力',
          813,
          483,
          10,
          '#1D4ED8',
          850,
          'center'
        );

        if(state.showLabels){
          text(
            '减排是降低新增碳源的核心路径',
            500,
            535,
            9.4,
            '#166534',
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
          '#BBF7D0'
        );

        text(
          '碳汇保护不能替代深度减排；减缓与适应也需要在发展过程中统筹推进。',
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

        deforestationValue.textContent=
          String(Math.round(value.deforestation));

        vegetationValue.textContent=
          String(Math.round(value.vegetation));

        oceanValue.textContent=
          String(Math.round(value.ocean));

        sensitivityValue.textContent=
          String(Math.round(value.sensitivity));

        mitigationValue.textContent=
          String(Math.round(value.mitigation));

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
            ? '自定义碳循环条件'
            : item.name;

        var direction=
          derived.netAtmosphere>1
            ? '大气碳储库呈增加趋势'
            : derived.netAtmosphere<-1
              ? '陆海碳汇大于新增碳源'
              : '碳源与碳汇接近相对平衡';

        result.textContent=
          scenarioName+
          '下，人类新增碳源为'+
          derived.humanSource.toFixed(1)+
          '，陆地碳汇为'+
          derived.landSink.toFixed(1)+
          '，海洋碳汇为'+
          derived.oceanSink.toFixed(1)+
          '，'+
          direction+
          '；温室效应强度为'+
          derived.greenhouseStrength+
          '，增温压力为'+
          derived.warmingPressure+
          '，低碳转型指数为'+
          derived.transitionScore+
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

        if(state.view==='greenhouse'){
          greenhouseView(
            item,
            value,
            derived
          );
        }else if(state.view==='trend'){
          trendView(
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
          cycleView(
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
            'cycle',
            'greenhouse',
            'trend',
            'pathway'
          ][
            Math.floor(
              elapsed/6800
            )%4
          ];

        state.phase=
          (
            elapsed/3800
          )%1;

        setInputs({
          fossil:lerp(
            from.fossil,
            to.fossil,
            progress
          ),
          deforestation:lerp(
            from.deforestation,
            to.deforestation,
            progress
          ),
          vegetation:lerp(
            from.vegetation,
            to.vegetation,
            progress
          ),
          ocean:lerp(
            from.ocean,
            to.ocean,
            progress
          ),
          sensitivity:lerp(
            from.sensitivity,
            to.sensitivity,
            progress
          ),
          mitigation:lerp(
            from.mitigation,
            to.mitigation,
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
        deforestationInput,
        vegetationInput,
        oceanInput,
        sensitivityInput,
        mitigationInput
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
                'fossil-growth',
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
                'cycle';

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

          state.view='cycle';

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

export const GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_CARBON_CLIMATE:
GeographyLabTemplate[] = [
  {
    id: 'geography-carbon-cycle-greenhouse-effect-global-climate-change',
    group: '🌱 全球变化与可持续发展',
    name: '碳循环、温室效应与全球气候变化',
    emoji: '🌍',
    desc: '调节化石燃料排放、森林破坏、植被和海洋吸收、气候敏感度及减排力度，观察碳储库与通量、温室效应、变化趋势和低碳转型路径。',
    params: [
      {
        key: 'scenario',
        label: '初始碳循环与发展情境',
        type: 'select',
        options: [
          {
            label: '自然相对平衡',
            value: 'balanced-cycle',
          },
          {
            label: '化石能源增长',
            value: 'fossil-growth',
          },
          {
            label: '森林破坏',
            value: 'deforestation',
          },
          {
            label: '生态系统恢复',
            value: 'ecosystem-restoration',
          },
          {
            label: '低碳转型',
            value: 'low-carbon-transition',
          },
        ],
        defaultValue: 'fossil-growth',
      },
      {
        key: 'fossilFuelEmissions',
        label: '化石燃料排放',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 8,
        hint: '表示煤、石油和天然气使用形成的人为新增碳源强度。',
      },
      {
        key: 'deforestation',
        label: '森林破坏程度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '森林破坏既会释放碳，也会削弱后续植被吸收能力。',
      },
      {
        key: 'vegetationUptake',
        label: '植被吸收能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '综合表示森林、草地、农田和生态恢复形成的陆地碳汇。',
      },
      {
        key: 'oceanAbsorption',
        label: '海洋吸收能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示海气交换和海洋过程吸收部分大气碳的相对能力。',
      },
      {
        key: 'climateSensitivity',
        label: '气候敏感程度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示气候系统对温室效应增强作出响应的相对敏感程度。',
      },
      {
        key: 'mitigation',
        label: '减排与转型力度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 4,
        hint: '综合表示能源转型、效率提升、低碳交通和碳汇保护等行动。',
      },
      {
        key: 'showLabels',
        label: '显示碳储库、通量与辐射标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildCarbonClimateHTML,
  },
]
