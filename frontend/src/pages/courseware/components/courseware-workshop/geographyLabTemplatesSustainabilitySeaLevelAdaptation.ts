/**
 * geographyLabTemplatesSustainabilitySeaLevelAdaptation.ts
 *
 * 地理第42批B2：
 *   海平面上升、沿海风险与适应。
 *
 * 教学目标：
 * - 理解海水热膨胀和陆地冰川、冰盖融水是全球平均海平面变化的重要因素；
 * - 理解地面沉降或抬升会改变当地相对海平面变化；
 * - 区分长期海平面变化与风暴潮等短时间极端增水；
 * - 综合分析沿海高程、人口和资产暴露、生态缓冲对风险的影响；
 * - 比较工程防护、适应性建设、生态系统保护和有序退让等适应路径；
 * - 认识沿海适应需要因地制宜，并统筹安全、生态、公平和发展成本。
 *
 * 教学边界：
 * - 海平面、高程、风暴潮、淹没范围和风险数值均为相对教学指数；
 * - 海岸线、城市、三角洲、湿地、堤防和人口分布不对应任何真实地区；
 * - 不考虑真实潮汐、波浪、海岸侵蚀、地下水和复杂地形过程；
 * - 不用于真实海岸工程设计、城市规划、保险评估或应急决策。
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

function buildSeaLevelAdaptationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'stable-coast',
    'high-warming',
    'subsiding-delta',
    'storm-surge',
    'adaptation-portfolio',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'subsiding-delta',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'subsiding-delta'

  const thermalExpansion = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'thermalExpansion', 6),
    ),
  )

  const landIceMelt = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'landIceMelt', 6),
    ),
  )

  const localSubsidence = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'localSubsidence', 7),
    ),
  )

  const stormSurge = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'stormSurge', 6),
    ),
  )

  const coastalElevation = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'coastalElevation', 4),
    ),
  )

  const populationExposure = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'populationExposure', 8),
    ),
  )

  const ecosystemBuffer = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'ecosystemBuffer', 4),
    ),
  )

  const adaptationCapacity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'adaptationCapacity', 5),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-sea-level-root">
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
      box-shadow:0 12px 34px rgba(2,132,199,.12);
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
        #E0F2FE 50%,
        #F0FDF4
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
      grid-template-columns:292px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      min-height:0;
      padding:13px;
      overflow:auto;
      border-right:1px solid #BAE6FD;
      background:linear-gradient(
        180deg,
        #ECFEFF,
        #F0F9FF 55%,
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
        circle at 48% 20%,
        #FFFFFF 0%,
        #F8FAFC 60%,
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
      background:#E0F2FE;
      color:#0369A1;
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
        #818CF8
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
        #0284C7,
        #4F46E5
      );
      box-shadow:0 1px 5px rgba(2,132,199,.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #7DD3FC;
      border-radius:9px;
      background:#FFFFFF;
      color:#0369A1;
      font-size:10.5px;
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
        #0F766E
      );
      box-shadow:0 5px 13px rgba(2,132,199,.22);
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
      background:linear-gradient(
        135deg,
        #ECFEFF,
        #F0FDF4
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
      border:1px solid #BAE6FD;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-sea-level-canvas{
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
    <div style="font-size:24px;">🌊</div>

    <div>
      <div class="gl-title">
        海平面上升、沿海风险与适应
      </div>

      <div class="gl-subtitle">
        比较海水热膨胀、陆地冰融化、地面沉降、风暴潮和不同沿海适应路径
      </div>
    </div>

    <div class="gl-note">
      相对教学指数 · 不用于真实海岸规划
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        沿海变化与适应情境
      </div>

      <div class="gl-scenario-grid">
        <button type="button" data-scenario="stable-coast">
          相对稳定海岸
        </button>

        <button type="button" data-scenario="high-warming">
          高增温情境
        </button>

        <button type="button" data-scenario="subsiding-delta">
          沉降三角洲
        </button>

        <button type="button" data-scenario="storm-surge">
          风暴潮叠加
        </button>

        <button type="button" data-scenario="adaptation-portfolio">
          综合适应方案
        </button>
      </div>

      <div class="gl-section-title">
        海平面、暴露与适应参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">海水热膨胀</span>
          <span class="gl-value" data-role="thermal-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${thermalExpansion}"
          data-role="thermal"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">陆地冰融水贡献</span>
          <span class="gl-value" data-role="ice-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${landIceMelt}"
          data-role="ice"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">当地地面沉降</span>
          <span class="gl-value" data-role="subsidence-value">7</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${localSubsidence}"
          data-role="subsidence"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">风暴潮增水</span>
          <span class="gl-value" data-role="surge-value">6</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${stormSurge}"
          data-role="surge"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">沿海地形高程</span>
          <span class="gl-value" data-role="elevation-value">4</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${coastalElevation}"
          data-role="elevation"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">人口与资产暴露</span>
          <span class="gl-value" data-role="exposure-value">8</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${populationExposure}"
          data-role="exposure"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">湿地与生态缓冲</span>
          <span class="gl-value" data-role="ecosystem-value">4</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${ecosystemBuffer}"
          data-role="ecosystem"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">适应能力</span>
          <span class="gl-value" data-role="adaptation-value">5</span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${adaptationCapacity}"
          data-role="adaptation"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          机制与风险标注
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
        全球平均海平面变化与海水热膨胀和陆地冰融水有关，当地相对海平面还会受到地面沉降或抬升影响。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="mechanism">
          上升机制
        </button>

        <button type="button" data-view="exposure">
          沿海风险
        </button>

        <button type="button" data-view="adaptation">
          适应路径
        </button>

        <button type="button" data-view="comparison">
          情境比较
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-sea-level-canvas"
          width="1000"
          height="570"
          data-role="canvas"
          aria-label="海平面上升沿海风险与适应教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var thermalInput =
        root.querySelector('[data-role="thermal"]');

      var iceInput =
        root.querySelector('[data-role="ice"]');

      var subsidenceInput =
        root.querySelector('[data-role="subsidence"]');

      var surgeInput =
        root.querySelector('[data-role="surge"]');

      var elevationInput =
        root.querySelector('[data-role="elevation"]');

      var exposureInput =
        root.querySelector('[data-role="exposure"]');

      var ecosystemInput =
        root.querySelector('[data-role="ecosystem"]');

      var adaptationInput =
        root.querySelector('[data-role="adaptation"]');

      var thermalValue =
        root.querySelector('[data-role="thermal-value"]');

      var iceValue =
        root.querySelector('[data-role="ice-value"]');

      var subsidenceValue =
        root.querySelector('[data-role="subsidence-value"]');

      var surgeValue =
        root.querySelector('[data-role="surge-value"]');

      var elevationValue =
        root.querySelector('[data-role="elevation-value"]');

      var exposureValue =
        root.querySelector('[data-role="exposure-value"]');

      var ecosystemValue =
        root.querySelector('[data-role="ecosystem-value"]');

      var adaptationValue =
        root.querySelector('[data-role="adaptation-value"]');

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
        !thermalInput ||
        !iceInput ||
        !subsidenceInput ||
        !surgeInput ||
        !elevationInput ||
        !exposureInput ||
        !ecosystemInput ||
        !adaptationInput ||
        !thermalValue ||
        !iceValue ||
        !subsidenceValue ||
        !surgeValue ||
        !elevationValue ||
        !exposureValue ||
        !ecosystemValue ||
        !adaptationValue ||
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
          key:'stable-coast',
          name:'相对稳定海岸',
          thermal:2,
          ice:2,
          subsidence:1,
          surge:3,
          elevation:7,
          exposure:4,
          ecosystem:8,
          adaptation:6,
          view:'mechanism',
          color:'#16A34A'
        },
        {
          key:'high-warming',
          name:'高增温情境',
          thermal:9,
          ice:9,
          subsidence:2,
          surge:6,
          elevation:5,
          exposure:7,
          ecosystem:5,
          adaptation:4,
          view:'comparison',
          color:'#DC2626'
        },
        {
          key:'subsiding-delta',
          name:'沉降三角洲',
          thermal:6,
          ice:6,
          subsidence:10,
          surge:7,
          elevation:2,
          exposure:9,
          ecosystem:3,
          adaptation:4,
          view:'exposure',
          color:'#D97706'
        },
        {
          key:'storm-surge',
          name:'风暴潮叠加',
          thermal:5,
          ice:5,
          subsidence:5,
          surge:10,
          elevation:3,
          exposure:9,
          ecosystem:3,
          adaptation:5,
          view:'exposure',
          color:'#7C3AED'
        },
        {
          key:'adaptation-portfolio',
          name:'综合适应方案',
          thermal:7,
          ice:7,
          subsidence:5,
          surge:8,
          elevation:4,
          exposure:7,
          ecosystem:9,
          adaptation:10,
          view:'adaptation',
          color:'#0284C7'
        }
      ];

      var initial={
        scenario:'${scenario}',
        thermal:${thermalExpansion},
        ice:${landIceMelt},
        subsidence:${localSubsidence},
        surge:${stormSurge},
        elevation:${coastalElevation},
        exposure:${populationExposure},
        ecosystem:${ecosystemBuffer},
        adaptation:${adaptationCapacity},
        showLabels:${showLabels}
      };

      var state={
        scenario:initial.scenario,
        view:'mechanism',
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
        var found=scenarios[2];

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
          thermal:clamp(
            Number(thermalInput.value) || 0,
            0,
            10
          ),
          ice:clamp(
            Number(iceInput.value) || 0,
            0,
            10
          ),
          subsidence:clamp(
            Number(subsidenceInput.value) || 0,
            0,
            10
          ),
          surge:clamp(
            Number(surgeInput.value) || 0,
            0,
            10
          ),
          elevation:clamp(
            Number(elevationInput.value) || 0,
            0,
            10
          ),
          exposure:clamp(
            Number(exposureInput.value) || 0,
            0,
            10
          ),
          ecosystem:clamp(
            Number(ecosystemInput.value) || 0,
            0,
            10
          ),
          adaptation:clamp(
            Number(adaptationInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        thermalInput.value=
          String(Math.round(value.thermal));

        iceInput.value=
          String(Math.round(value.ice));

        subsidenceInput.value=
          String(Math.round(value.subsidence));

        surgeInput.value=
          String(Math.round(value.surge));

        elevationInput.value=
          String(Math.round(value.elevation));

        exposureInput.value=
          String(Math.round(value.exposure));

        ecosystemInput.value=
          String(Math.round(value.ecosystem));

        adaptationInput.value=
          String(Math.round(value.adaptation));
      }

      function derive(value){
        var globalMeanRise=
          clamp(
            value.thermal*.42+
            value.ice*.58,
            0,
            10
          );

        var relativeSeaLevel=
          clamp(
            globalMeanRise*.72+
            value.subsidence*.42,
            0,
            10
          );

        var extremeWaterLevel=
          clamp(
            relativeSeaLevel*.58+
            value.surge*.66,
            0,
            10
          );

        var topographicSensitivity=
          clamp(
            10-value.elevation,
            0,
            10
          );

        var inundationPotential=
          clamp(
            extremeWaterLevel*.48+
            topographicSensitivity*.38+
            value.subsidence*.14-
            value.ecosystem*.12,
            0,
            10
          );

        var rawRisk=
          clamp(
            Math.round(
              inundationPotential*.45+
              value.exposure*.32+
              value.surge*.13+
              topographicSensitivity*.10
            ),
            0,
            10
          );

        var ecosystemProtection=
          clamp(
            Math.round(
              value.ecosystem*.68+
              value.adaptation*.12
            ),
            0,
            10
          );

        var residualRisk=
          clamp(
            Math.round(
              rawRisk-
              value.adaptation*.42-
              value.ecosystem*.18
            ),
            0,
            10
          );

        var protectionScore=
          clamp(
            Math.round(
              value.adaptation*.56+
              value.elevation*.18+
              (
                10-value.surge
              )*.10+
              value.ecosystem*.16
            ),
            0,
            10
          );

        var accommodationScore=
          clamp(
            Math.round(
              value.adaptation*.50+
              value.elevation*.14+
              value.ecosystem*.16+
              (
                10-value.exposure
              )*.20
            ),
            0,
            10
          );

        var retreatNeed=
          clamp(
            Math.round(
              rawRisk*.46+
              value.subsidence*.20+
              topographicSensitivity*.20+
              value.exposure*.14-
              value.adaptation*.18
            ),
            0,
            10
          );

        var resilience=
          clamp(
            Math.round(
              value.adaptation*.38+
              ecosystemProtection*.24+
              (
                10-residualRisk
              )*.24+
              value.elevation*.14
            ),
            0,
            10
          );

        return {
          globalMeanRise:globalMeanRise,
          relativeSeaLevel:relativeSeaLevel,
          extremeWaterLevel:extremeWaterLevel,
          topographicSensitivity:topographicSensitivity,
          inundationPotential:inundationPotential,
          rawRisk:rawRisk,
          ecosystemProtection:ecosystemProtection,
          residualRisk:residualRisk,
          protectionScore:protectionScore,
          accommodationScore:accommodationScore,
          retreatNeed:retreatNeed,
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
        gradient.addColorStop(1,'#DCFCE7');

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

      function mechanismView(
        item,
        value,
        derived
      ){
        background(
          '全球平均海平面与当地相对海平面',
          '全球平均变化主要来自海水热膨胀和陆地冰融水，当地还受到地面升降影响。'
        );

        card(
          28,
          78,
          210,
          '全球平均变化',
          derived.globalMeanRise.toFixed(1),
          '#2563EB',
          '热膨胀与陆地冰融水'
        );

        card(
          250,
          78,
          210,
          '当地相对变化',
          derived.relativeSeaLevel.toFixed(1),
          '#7C3AED',
          '叠加当地地面沉降'
        );

        card(
          472,
          78,
          210,
          '极端水位',
          derived.extremeWaterLevel.toFixed(1),
          '#DC2626',
          '长期变化与风暴潮叠加'
        );

        card(
          694,
          78,
          258,
          '沿海韧性',
          derived.resilience,
          item.color,
          '适应、生态与风险综合'
        );

        var seaTop=
          397-
          derived.globalMeanRise*10;

        context.fillStyle='#E0F2FE';
        context.fillRect(
          42,
          176,
          916,
          342
        );

        var ocean=
          context.createLinearGradient(
            0,
            seaTop,
            0,
            518
          );

        ocean.addColorStop(0,'#38BDF8');
        ocean.addColorStop(1,'#075985');

        context.fillStyle=ocean;
        context.fillRect(
          42,
          seaTop,
          916,
          518-seaTop
        );

        line(
          42,
          397,
          958,
          397,
          '#FFFFFF',
          2,
          [9,7]
        );

        line(
          42,
          seaTop,
          958,
          seaTop,
          '#FDE047',
          3
        );

        context.fillStyle='#CBD5E1';
        context.beginPath();
        context.moveTo(62,396);
        context.lineTo(180,229);
        context.lineTo(286,396);
        context.closePath();
        context.fill();

        context.fillStyle='#FFFFFF';
        context.beginPath();
        context.moveTo(118,317);
        context.lineTo(180,229);
        context.lineTo(232,317);
        context.closePath();
        context.fill();

        context.fillStyle='#E0F2FE';
        context.beginPath();
        context.moveTo(130,395);
        context.lineTo(205,286);
        context.lineTo(270,395);
        context.closePath();
        context.fill();

        var meltDropCount=
          2+
          Math.round(
            value.ice/2
          );

        for(
          var drop=0;
          drop<meltDropCount;
          drop+=1
        ){
          var dropX=
            155+
            drop*
            18;

          arrow(
            dropX,
            319+
            drop%2*
            13,
            dropX+12,
            370,
            '#0284C7',
            2
          );
        }

        box(
          375,
          208,
          206,
          188,
          15,
          '#FFFFFF',
          '#7DD3FC'
        );

        text(
          '海水热膨胀',
          478,
          231,
          13,
          '#0369A1',
          880,
          'center'
        );

        var coldColumnHeight=105;
        var warmColumnHeight=
          coldColumnHeight+
          value.thermal*5.6;

        context.fillStyle='#BAE6FD';
        context.fillRect(
          405,
          354-coldColumnHeight,
          55,
          coldColumnHeight
        );

        context.fillStyle='#38BDF8';
        context.fillRect(
          496,
          354-warmColumnHeight,
          55,
          warmColumnHeight
        );

        line(
          405,
          354,
          551,
          354,
          '#475569',
          2
        );

        text(
          '较冷',
          432,
          372,
          9.5,
          '#475569',
          760,
          'center'
        );

        text(
          '较暖',
          523,
          372,
          9.5,
          '#DC2626',
          760,
          'center'
        );

        arrow(
          462,
          305,
          491,
          305,
          '#DC2626',
          3
        );

        context.fillStyle='#D6D3D1';
        context.beginPath();
        context.moveTo(674,518);
        context.lineTo(748,330);
        context.lineTo(958,316);
        context.lineTo(958,518);
        context.closePath();
        context.fill();

        var groundDrop=
          value.subsidence*7;

        line(
          748,
          330,
          958,
          316,
          '#475569',
          3
        );

        line(
          748,
          330+groundDrop,
          958,
          316+groundDrop,
          '#DC2626',
          3,
          [8,6]
        );

        for(
          var building=0;
          building<4;
          building+=1
        ){
          var bx=775+building*43;
          var baseY=
            322+
            groundDrop-
            building%2*
            6;

          context.fillStyle=
            building%2===0
              ? '#64748B'
              : '#475569';

          context.fillRect(
            bx,
            baseY-67,
            31,
            67
          );

          context.fillStyle='#E0F2FE';
          context.fillRect(
            bx+8,
            baseY-53,
            7,
            10
          );

          context.fillRect(
            bx+18,
            baseY-53,
            7,
            10
          );
        }

        arrow(
          920,
          265,
          920,
          315+
          groundDrop,
          '#DC2626',
          3
        );

        if(state.showLabels){
          text(
            '陆地冰融水进入海洋',
            175,
            210,
            10,
            '#0284C7',
            850,
            'center'
          );

          text(
            '水温升高，体积增大',
            478,
            191,
            10,
            '#DC2626',
            850,
            'center'
          );

          text(
            '地面沉降',
            860,
            233,
            10,
            '#DC2626',
            850,
            'center'
          );

          text(
            '基准海平面',
            110,
            387,
            9.5,
            '#FFFFFF',
            800,
            'center'
          );

          text(
            '变化后海平面',
            692,
            seaTop-13,
            9.5,
            '#B45309',
            850,
            'center'
          );
        }

        box(
          57,
          526,
          886,
          28,
          10,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '海上漂浮冰融化与陆地冰融化的作用不同，本模型重点表示陆地冰融水进入海洋。',
          500,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function drawBuilding(
        x,
        baseY,
        w,
        h,
        color
      ){
        context.fillStyle=color;
        context.fillRect(
          x,
          baseY-h,
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
              x+7+col*16,
              baseY-h+12+row*20,
              8,
              10
            );
          }
        }
      }

      function exposureView(
        item,
        value,
        derived
      ){
        background(
          '沿海低地、极端水位与复合风险',
          '长期海平面变化会抬高风暴潮发生时的水位基线，低地、高暴露地区风险更高。'
        );

        card(
          28,
          78,
          210,
          '相对海平面',
          derived.relativeSeaLevel.toFixed(1),
          '#2563EB',
          '全球变化与沉降叠加'
        );

        card(
          250,
          78,
          210,
          '极端水位',
          derived.extremeWaterLevel.toFixed(1),
          '#7C3AED',
          '长期变化与风暴潮'
        );

        card(
          472,
          78,
          210,
          '潜在淹没范围',
          derived.inundationPotential.toFixed(1),
          '#DC2626',
          '高程、增水和生态缓冲'
        );

        card(
          694,
          78,
          258,
          '治理后剩余风险',
          derived.residualRisk,
          item.color,
          '适应措施不能完全消除风险'
        );

        var baseY=456;
        var seaLevel=
          416-
          derived.extremeWaterLevel*12;

        var terrain=
          context.createLinearGradient(
            0,
            260,
            0,
            510
          );

        terrain.addColorStop(0,'#BBF7D0');
        terrain.addColorStop(1,'#A16207');

        context.fillStyle=terrain;
        context.beginPath();
        context.moveTo(42,baseY);
        context.lineTo(420,baseY);
        context.bezierCurveTo(
          500,
          437-
          value.elevation*6,
          595,
          414-
          value.elevation*10,
          690,
          385-
          value.elevation*13
        );
        context.bezierCurveTo(
          790,
          354-
          value.elevation*10,
          858,
          338-
          value.elevation*7,
          958,
          325-
          value.elevation*5
        );
        context.lineTo(958,518);
        context.lineTo(42,518);
        context.closePath();
        context.fill();

        context.fillStyle='#0284C7';
        context.fillRect(
          42,
          seaLevel,
          916,
          518-seaLevel
        );

        context.fillStyle='rgba(125,211,252,.50)';
        context.beginPath();
        context.moveTo(42,seaLevel);
        context.lineTo(
          470+
          derived.inundationPotential*29,
          seaLevel
        );
        context.lineTo(
          470+
          derived.inundationPotential*21,
          baseY
        );
        context.lineTo(42,baseY);
        context.closePath();
        context.fill();

        var surgeTop=
          seaLevel-
          value.surge*5;

        context.fillStyle='rgba(99,102,241,.30)';
        context.fillRect(
          42,
          surgeTop,
          330+
          value.surge*29,
          seaLevel-surgeTop
        );

        for(
          var wave=0;
          wave<6;
          wave+=1
        ){
          var wx=
            65+
            wave*
            65;

          context.strokeStyle='#FFFFFF';
          context.lineWidth=2;
          context.beginPath();
          context.arc(
            wx,
            surgeTop+
            10+
            wave%2*
            7,
            24,
            Math.PI,
            Math.PI*2
          );
          context.stroke();
        }

        var wetlandStart=420;
        var wetlandWidth=
          25+
          value.ecosystem*19;

        context.fillStyle='#22C55E';
        context.fillRect(
          wetlandStart,
          baseY-22,
          wetlandWidth,
          22
        );

        for(
          var plant=0;
          plant<Math.round(value.ecosystem)+2;
          plant+=1
        ){
          var px=
            wetlandStart+
            12+
            plant*
            Math.max(
              12,
              wetlandWidth/
              (
                value.ecosystem+2
              )
            );

          line(
            px,
            baseY-7,
            px-3,
            baseY-34,
            '#15803D',
            2
          );

          line(
            px,
            baseY-13,
            px+8,
            baseY-28,
            '#16A34A',
            2
          );
        }

        var cityStart=625;
        var buildingCount=
          3+
          Math.round(
            value.exposure/2
          );

        for(
          var building=0;
          building<buildingCount;
          building+=1
        ){
          var bx=
            cityStart+
            building*
            43;

          var localBase=
            baseY-
            55-
            value.elevation*7-
            building%3*
            8;

          drawBuilding(
            bx,
            localBase,
            32,
            62+
            building%3*
            23,
            building%2===0
              ? '#475569'
              : '#64748B'
          );
        }

        var defenseHeight=
          18+
          value.adaptation*6;

        context.fillStyle='#94A3B8';
        context.beginPath();
        context.moveTo(
          590,
          baseY-35
        );
        context.lineTo(
          615,
          baseY-
          defenseHeight
        );
        context.lineTo(
          640,
          baseY-35
        );
        context.closePath();
        context.fill();

        line(
          42,
          seaLevel,
          958,
          seaLevel,
          '#FDE047',
          2
        );

        line(
          42,
          surgeTop,
          958,
          surgeTop,
          '#C4B5FD',
          2,
          [8,6]
        );

        if(state.showLabels){
          text(
            '长期相对海平面',
            185,
            seaLevel-13,
            10,
            '#B45309',
            850,
            'center'
          );

          text(
            '风暴潮极端水位',
            260,
            surgeTop-13,
            10,
            '#6D28D9',
            850,
            'center'
          );

          text(
            '滨海湿地缓冲',
            wetlandStart+
            wetlandWidth/2,
            baseY-49,
            10,
            '#15803D',
            850,
            'center'
          );

          text(
            '防潮工程',
            615,
            baseY-
            defenseHeight-
            15,
            10,
            '#475569',
            850,
            'center'
          );

          text(
            '沿海人口与资产',
            800,
            219,
            10,
            '#334155',
            850,
            'center'
          );

          arrow(
            480,
            346,
            538,
            390,
            '#0284C7',
            3
          );

          text(
            '海水向低地扩展',
            492,
            324,
            9.5,
            '#0284C7',
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
          '#BAE6FD'
        );

        text(
          '长期海平面上升会抬高极端增水的起点，使原本较少受淹的低地更容易暴露。',
          491,
          540,
          10,
          '#475569',
          680,
          'center'
        );
      }

      function adaptationCard(
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

      function adaptationView(
        item,
        value,
        derived
      ){
        background(
          '保护、适应、生态缓冲与有序退让',
          '沿海地区需要根据地形、人口、价值、生态和长期风险组合不同适应路径。'
        );

        card(
          28,
          78,
          210,
          '治理前风险',
          derived.rawRisk,
          '#DC2626',
          '危险性、暴露和地形综合'
        );

        card(
          250,
          78,
          210,
          '生态保护能力',
          derived.ecosystemProtection,
          '#16A34A',
          '湿地和适应投入综合'
        );

        card(
          472,
          78,
          210,
          '治理后风险',
          derived.residualRisk,
          '#D97706',
          '组合措施后的剩余风险'
        );

        card(
          694,
          78,
          258,
          '沿海韧性',
          derived.resilience,
          item.color,
          '安全、生态与适应综合'
        );

        adaptationCard(
          58,
          188,
          410,
          '工程保护',
          '建设或提升堤防、防潮闸、泵站和重要设施防护标准。',
          '#2563EB',
          derived.protectionScore
        );

        adaptationCard(
          514,
          188,
          410,
          '适应性建设',
          '提高建筑标高、改善排水、设置避难空间和预警系统。',
          '#7C3AED',
          derived.accommodationScore
        );

        adaptationCard(
          58,
          310,
          410,
          '生态系统适应',
          '保护和恢复湿地、盐沼、红树林、沙丘等自然缓冲空间。',
          '#16A34A',
          derived.ecosystemProtection
        );

        adaptationCard(
          514,
          310,
          410,
          '有序退让与空间管控',
          '限制新增高风险开发，并为持续高风险地区规划渐进式搬迁。',
          '#D97706',
          derived.retreatNeed
        );

        box(
          58,
          430,
          866,
          84,
          14,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '适应组合逻辑',
          80,
          451,
          12,
          '#075985',
          860,
          'left'
        );

        box(
          104,
          468,
          164,
          30,
          9,
          '#DBEAFE',
          '#93C5FD'
        );

        text(
          '保护重要区域',
          186,
          483,
          10,
          '#1D4ED8',
          850,
          'center'
        );

        arrow(
          290,
          483,
          356,
          483,
          '#64748B',
          3
        );

        box(
          378,
          468,
          164,
          30,
          9,
          '#DCFCE7',
          '#86EFAC'
        );

        text(
          '保留生态空间',
          460,
          483,
          10,
          '#15803D',
          850,
          'center'
        );

        arrow(
          564,
          483,
          630,
          483,
          '#64748B',
          3
        );

        box(
          652,
          468,
          164,
          30,
          9,
          '#FEF3C7',
          '#FBBF24'
        );

        text(
          '调整高风险布局',
          734,
          483,
          10,
          '#B45309',
          850,
          'center'
        );

        arrow(
          838,
          483,
          888,
          483,
          item.color,
          3
        );

        text(
          '韧性 '+derived.resilience,
          900,
          483,
          9.5,
          item.color,
          880,
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
          '适应没有单一方案：人口密集区、生态敏感区和持续沉降区需要不同组合。',
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

      function comparisonView(
        item,
        value,
        derived
      ){
        background(
          '不同沿海情境的风险与适应效果比较',
          '相同全球海平面变化在不同高程、沉降、暴露和适应能力下会产生不同风险。'
        );

        card(
          28,
          78,
          210,
          '全球平均变化',
          derived.globalMeanRise.toFixed(1),
          '#2563EB',
          '热膨胀与陆地冰融水'
        );

        card(
          250,
          78,
          210,
          '当地相对变化',
          derived.relativeSeaLevel.toFixed(1),
          '#7C3AED',
          '叠加地面沉降'
        );

        card(
          472,
          78,
          210,
          '治理前风险',
          derived.rawRisk,
          '#DC2626',
          '危险性与暴露综合'
        );

        card(
          694,
          78,
          258,
          '治理后风险',
          derived.residualRisk,
          item.color,
          '生态与适应措施作用'
        );

        riskBar(
          58,
          188,
          410,
          '全球海平面变化压力',
          Math.round(
            derived.globalMeanRise
          ),
          '#2563EB',
          '由海水热膨胀和陆地冰融水共同构成'
        );

        riskBar(
          514,
          188,
          410,
          '当地相对海平面压力',
          Math.round(
            derived.relativeSeaLevel
          ),
          '#7C3AED',
          '地面沉降可能进一步放大当地变化'
        );

        riskBar(
          58,
          306,
          410,
          '极端增水与淹没压力',
          Math.round(
            derived.inundationPotential
          ),
          '#DC2626',
          '长期变化、风暴潮和低地地形共同作用'
        );

        riskBar(
          514,
          306,
          410,
          '适应后剩余风险',
          derived.residualRisk,
          item.color,
          '工程、生态和空间适应后的课堂结果'
        );

        box(
          58,
          426,
          866,
          88,
          14,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '风险形成链',
          80,
          447,
          12,
          '#075985',
          860,
          'left'
        );

        box(
          90,
          469,
          158,
          28,
          9,
          '#DBEAFE',
          '#93C5FD'
        );

        text(
          '海平面与风暴潮',
          169,
          483,
          9.8,
          '#1D4ED8',
          850,
          'center'
        );

        arrow(
          267,
          483,
          330,
          483,
          '#64748B',
          3
        );

        box(
          350,
          469,
          158,
          28,
          9,
          '#FEF3C7',
          '#FBBF24'
        );

        text(
          '低地与沉降',
          429,
          483,
          9.8,
          '#B45309',
          850,
          'center'
        );

        arrow(
          527,
          483,
          590,
          483,
          '#64748B',
          3
        );

        box(
          610,
          469,
          158,
          28,
          9,
          '#FEE2E2',
          '#FCA5A5'
        );

        text(
          '人口资产暴露',
          689,
          483,
          9.8,
          '#B91C1C',
          850,
          'center'
        );

        arrow(
          787,
          483,
          835,
          483,
          item.color,
          3
        );

        text(
          '风险 '+derived.rawRisk,
          875,
          483,
          10,
          item.color,
          880,
          'center'
        );

        if(state.showLabels){
          text(
            '适应和生态缓冲可降低风险，但不能使长期风险自动归零。',
            500,
            535,
            9.4,
            '#075985',
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
          '#BAE6FD'
        );

        text(
          '全球变化决定背景压力，当地地形、沉降、暴露和治理能力决定实际风险差异。',
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
        thermalValue.textContent=
          String(Math.round(value.thermal));

        iceValue.textContent=
          String(Math.round(value.ice));

        subsidenceValue.textContent=
          String(Math.round(value.subsidence));

        surgeValue.textContent=
          String(Math.round(value.surge));

        elevationValue.textContent=
          String(Math.round(value.elevation));

        exposureValue.textContent=
          String(Math.round(value.exposure));

        ecosystemValue.textContent=
          String(Math.round(value.ecosystem));

        adaptationValue.textContent=
          String(Math.round(value.adaptation));

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
            ? '自定义沿海条件'
            : item.name;

        result.textContent=
          scenarioName+
          '下，全球平均海平面变化指数为'+
          derived.globalMeanRise.toFixed(1)+
          '，叠加地面沉降后的当地相对变化为'+
          derived.relativeSeaLevel.toFixed(1)+
          '，风暴潮叠加后的极端水位为'+
          derived.extremeWaterLevel.toFixed(1)+
          '；治理前风险为'+
          derived.rawRisk+
          '，采取工程、生态和空间适应后剩余风险为'+
          derived.residualRisk+
          '，沿海韧性为'+
          derived.resilience+
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

        if(state.view==='exposure'){
          exposureView(
            item,
            value,
            derived
          );
        }else if(state.view==='adaptation'){
          adaptationView(
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
        }else{
          mechanismView(
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
            'mechanism',
            'exposure',
            'adaptation',
            'comparison'
          ][
            Math.floor(
              elapsed/6900
            )%4
          ];

        state.phase=
          (
            elapsed/3900
          )%1;

        setInputs({
          thermal:lerp(
            from.thermal,
            to.thermal,
            progress
          ),
          ice:lerp(
            from.ice,
            to.ice,
            progress
          ),
          subsidence:lerp(
            from.subsidence,
            to.subsidence,
            progress
          ),
          surge:lerp(
            from.surge,
            to.surge,
            progress
          ),
          elevation:lerp(
            from.elevation,
            to.elevation,
            progress
          ),
          exposure:lerp(
            from.exposure,
            to.exposure,
            progress
          ),
          ecosystem:lerp(
            from.ecosystem,
            to.ecosystem,
            progress
          ),
          adaptation:lerp(
            from.adaptation,
            to.adaptation,
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
        thermalInput,
        iceInput,
        subsidenceInput,
        surgeInput,
        elevationInput,
        exposureInput,
        ecosystemInput,
        adaptationInput
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
                'subsiding-delta',
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
                'mechanism';

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

          state.view='mechanism';

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

export const GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_SEA_LEVEL_ADAPTATION:
GeographyLabTemplate[] = [
  {
    id: 'geography-sea-level-rise-coastal-risk-adaptation',
    group: '🌱 全球变化与可持续发展',
    name: '海平面上升、沿海风险与适应',
    emoji: '🌊',
    desc: '调节海水热膨胀、陆地冰融水、地面沉降、风暴潮、沿海高程、人口资产暴露、生态缓冲和适应能力，比较海平面变化机制、沿海风险及保护、适应和退让路径。',
    params: [
      {
        key: 'scenario',
        label: '初始沿海变化情境',
        type: 'select',
        options: [
          {
            label: '相对稳定海岸',
            value: 'stable-coast',
          },
          {
            label: '高增温情境',
            value: 'high-warming',
          },
          {
            label: '沉降三角洲',
            value: 'subsiding-delta',
          },
          {
            label: '风暴潮叠加',
            value: 'storm-surge',
          },
          {
            label: '综合适应方案',
            value: 'adaptation-portfolio',
          },
        ],
        defaultValue: 'subsiding-delta',
      },
      {
        key: 'thermalExpansion',
        label: '海水热膨胀',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '海水增温后体积膨胀，是全球平均海平面变化的重要因素。',
      },
      {
        key: 'landIceMelt',
        label: '陆地冰融水贡献',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示山地冰川和陆地冰盖融水进入海洋的相对贡献。',
      },
      {
        key: 'localSubsidence',
        label: '当地地面沉降',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '地面沉降会放大当地相对海平面上升，实际原因可能包括自然和人类因素。',
      },
      {
        key: 'stormSurge',
        label: '风暴潮增水',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '表示强风和低气压等因素形成的短时间极端增水。',
      },
      {
        key: 'coastalElevation',
        label: '沿海地形高程',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 4,
        hint: '高程较低且地形平缓的地区通常更容易受到海水扩展影响。',
      },
      {
        key: 'populationExposure',
        label: '人口与资产暴露度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 8,
        hint: '表示风险区内人口、建筑、道路和重要设施的集中程度。',
      },
      {
        key: 'ecosystemBuffer',
        label: '湿地与生态缓冲',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 4,
        hint: '湿地、盐沼、红树林和沙丘等生态空间可提供一定缓冲作用。',
      },
      {
        key: 'adaptationCapacity',
        label: '沿海适应能力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '综合表示工程防护、预警排涝、适应性建设和空间管控能力。',
      },
      {
        key: 'showLabels',
        label: '显示机制、风险与适应标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildSeaLevelAdaptationHTML,
  },
]
