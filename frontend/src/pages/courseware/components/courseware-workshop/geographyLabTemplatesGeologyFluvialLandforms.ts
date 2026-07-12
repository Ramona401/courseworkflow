/**
 * geographyLabTemplatesGeologyFluvialLandforms.ts
 *
 * 第36批B3：流水侵蚀、搬运、堆积与河流地貌演化。
 *
 * 教学目标：
 * 1. 比较河流上游、中游、下游和河口地区的坡度、流量与地貌差异；
 * 2. 理解流水侵蚀、搬运和堆积作用之间的相互联系；
 * 3. 观察坡度、流量、含沙量、植被和基准面对流水作用的影响；
 * 4. 理解上游下切侵蚀与V形谷、中游侧蚀与曲流、
 *    下游堆积与冲积平原、河口沉积与三角洲之间的关系；
 * 5. 区分曲流凹岸侵蚀和凸岸堆积；
 * 6. 理解河流纵剖面会随侵蚀、搬运和堆积长期调整；
 * 7. 认识地貌是内外力共同作用和长期演化的结果。
 *
 * 教学边界：
 * - 所有坡度、流量、含沙量、侵蚀率和堆积率均为相对教学量；
 * - 河道形态、曲流尺度和三角洲形态不对应任何真实河流；
 * - 不考虑具体岩性、流域面积、洪水过程、河岸工程和潮汐作用；
 * - 不用于洪水预测、航道判断、河道治理、工程选址或灾害决策；
 * - 真实河流地貌还受气候、地质构造、海平面和人类活动影响。
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

function shortNumber(value: number): string {
  return Number(value.toFixed(3)).toString()
}

function buildFluvialLandformsHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const requestedReach = stringValue(
    params,
    'riverReach',
    'middle',
  )

  const riverReach = [
    'upper',
    'middle',
    'lower',
    'estuary',
  ].includes(requestedReach)
    ? requestedReach
    : 'middle'

  const slope = Math.max(
    1,
    Math.min(
      100,
      numberValue(params, 'slope', 52),
    ),
  )

  const discharge = Math.max(
    5,
    Math.min(
      100,
      numberValue(params, 'discharge', 62),
    ),
  )

  const sedimentLoad = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'sedimentLoad', 55),
    ),
  )

  const vegetation = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'vegetation', 48),
    ),
  )

  const baseLevel = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'baseLevel', 50),
    ),
  )

  const requestedMode = stringValue(
    params,
    'observationMode',
    'profile',
  )

  const observationMode = [
    'profile',
    'plan',
    'process',
  ].includes(requestedMode)
    ? requestedMode
    : 'profile'

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  const automatic = booleanValue(
    params,
    'automatic',
    true,
  )

  return `
<div id="${rootId}" class="gl-fluvial-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border-radius:18px;
      border:1px solid #B7D7C4;
      background:#FFFFFF;
      color:#0F172A;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(21,94,89,0.10);
    }

    #${rootId} *{
      box-sizing:border-box;
    }

    #${rootId} .gl-head{
      height:52px;
      padding:0 18px;
      display:flex;
      align-items:center;
      gap:12px;
      background:linear-gradient(
        135deg,
        #DCFCE7,
        #E0F2FE 55%,
        #FEF3C7
      );
      border-bottom:1px solid #B7D7C4;
    }

    #${rootId} .gl-title{
      color:#14532D;
      font-size:16px;
      font-weight:850;
    }

    #${rootId} .gl-note{
      margin-left:auto;
      color:#64748B;
      font-size:11px;
      white-space:nowrap;
    }

    #${rootId} .gl-body{
      height:calc(100% - 52px);
      display:grid;
      grid-template-columns:248px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      padding:13px;
      overflow:auto;
      border-right:1px solid #CDE7D6;
      background:linear-gradient(
        180deg,
        #F0FDF4,
        #EFF6FF
      );
    }

    #${rootId} .gl-stage{
      position:relative;
      min-width:0;
      min-height:0;
      padding:8px;
      background:radial-gradient(
        circle at 48% 18%,
        #FFFFFF 0%,
        #F8FAFC 54%,
        #E0F2FE 100%
      );
    }

    #${rootId} .gl-row{
      margin-bottom:11px;
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
      font-size:11px;
      font-weight:750;
    }

    #${rootId} .gl-value{
      padding:3px 7px;
      border-radius:999px;
      background:#DCFCE7;
      color:#15803D;
      font-size:10.5px;
      font-weight:850;
      white-space:nowrap;
    }

    #${rootId} input[type=range]{
      width:100%;
      height:5px;
      margin:0;
      appearance:none;
      border-radius:999px;
      outline:none;
      background:linear-gradient(
        90deg,
        #BAE6FD,
        #4ADE80
      );
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:16px;
      height:16px;
      appearance:none;
      border-radius:50%;
      background:#15803D;
      border:2px solid #FFFFFF;
      box-shadow:0 1px 5px rgba(21,128,61,0.40);
    }

    #${rootId} select{
      width:100%;
      min-height:34px;
      padding:6px 8px;
      border:1px solid #A7D8BC;
      border-radius:9px;
      background:#FFFFFF;
      color:#166534;
      font-size:11px;
      font-weight:750;
      outline:none;
    }

    #${rootId} .gl-switch-row{
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:8px;
      padding:7px 8px;
      margin-bottom:7px;
      border-radius:10px;
      background:#FFFFFF;
      border:1px solid #D1FAE5;
      color:#334155;
      font-size:10.5px;
      font-weight:750;
    }

    #${rootId} .gl-switch-row input{
      accent-color:#15803D;
    }

    #${rootId} .gl-subtitle{
      margin:10px 0 6px;
      color:#166534;
      font-size:11px;
      font-weight:900;
    }

    #${rootId} .gl-button-grid{
      display:grid;
      grid-template-columns:1fr 1fr;
      gap:6px;
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 5px;
      border:1px solid #A7D8BC;
      border-radius:9px;
      background:#FFFFFF;
      color:#166534;
      font-size:10px;
      font-weight:800;
      cursor:pointer;
      transition:
        transform .14s,
        border-color .14s,
        background .14s;
    }

    #${rootId} button:hover{
      transform:translateY(-1px);
      border-color:#15803D;
    }

    #${rootId} button.active{
      border-color:#15803D;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #22C55E,
        #15803D
      );
    }

    #${rootId} .gl-result{
      margin-top:9px;
      padding:9px;
      border-radius:11px;
      background:#DCFCE7;
      border:1px solid #A7D8BC;
      color:#14532D;
      font-size:10.2px;
      font-weight:650;
      line-height:1.48;
      max-height:80px;
      overflow:auto;
    }

    #${rootId} .gl-fluvial-canvas{
      width:100%;
      height:100%;
      display:block;
    }

    #${rootId} .gl-summary{
      position:absolute;
      left:18px;
      right:18px;
      bottom:15px;
      display:grid;
      grid-template-columns:repeat(4,1fr);
      gap:8px;
      pointer-events:none;
    }

    #${rootId} .gl-summary-card{
      min-width:0;
      padding:6px 8px;
      border-radius:10px;
      background:rgba(255,255,255,0.93);
      border:1px solid #BBE4CB;
      box-shadow:0 5px 15px rgba(21,94,89,0.08);
      text-align:center;
    }

    #${rootId} .gl-summary-card strong{
      display:block;
      color:#0369A1;
      font-size:12px;
      font-weight:900;
      white-space:nowrap;
      overflow:hidden;
      text-overflow:ellipsis;
    }

    #${rootId} .gl-summary-card span{
      display:block;
      margin-top:2px;
      color:#64748B;
      font-size:9px;
      white-space:nowrap;
      overflow:hidden;
      text-overflow:ellipsis;
    }
  </style>

  <div class="gl-head">
    <div style="font-size:23px;">
      🏞️
    </div>

    <div>
      <div class="gl-title">
        流水侵蚀、搬运、堆积与河流地貌演化
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        调节坡度、流量、含沙量、植被和基准面，观察河流地貌变化
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 不用于真实河道决策
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            河段位置
          </span>
        </div>

        <select data-role="river-reach">
          <option
            value="upper"
            ${riverReach === 'upper' ? 'selected' : ''}
          >
            上游山区
          </option>

          <option
            value="middle"
            ${riverReach === 'middle' ? 'selected' : ''}
          >
            中游河段
          </option>

          <option
            value="lower"
            ${riverReach === 'lower' ? 'selected' : ''}
          >
            下游平原
          </option>

          <option
            value="estuary"
            ${riverReach === 'estuary' ? 'selected' : ''}
          >
            河口地区
          </option>
        </select>
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            河床坡度
          </span>

          <span
            class="gl-value"
            data-role="slope-value"
          ></span>
        </div>

        <input
          type="range"
          min="1"
          max="100"
          step="1"
          value="${shortNumber(slope)}"
          data-role="slope"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            河流流量
          </span>

          <span
            class="gl-value"
            data-role="discharge-value"
          ></span>
        </div>

        <input
          type="range"
          min="5"
          max="100"
          step="1"
          value="${shortNumber(discharge)}"
          data-role="discharge"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            河流含沙量
          </span>

          <span
            class="gl-value"
            data-role="sediment-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(sedimentLoad)}"
          data-role="sediment"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            河岸植被覆盖
          </span>

          <span
            class="gl-value"
            data-role="vegetation-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(vegetation)}"
          data-role="vegetation"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            侵蚀基准面
          </span>

          <span
            class="gl-value"
            data-role="base-level-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(baseLevel)}"
          data-role="base-level"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            观察模式
          </span>
        </div>

        <select data-role="observation-mode">
          <option
            value="profile"
            ${observationMode === 'profile' ? 'selected' : ''}
          >
            河流纵剖面
          </option>

          <option
            value="plan"
            ${observationMode === 'plan' ? 'selected' : ''}
          >
            河道平面形态
          </option>

          <option
            value="process"
            ${observationMode === 'process' ? 'selected' : ''}
          >
            侵蚀搬运堆积
          </option>
        </select>
      </div>

      <div class="gl-switch-row">
        <span>显示过程与地貌标注</span>

        <input
          type="checkbox"
          data-role="label-switch"
          ${showLabels ? 'checked' : ''}
        />
      </div>

      <div class="gl-switch-row">
        <span>自动演示典型河段</span>

        <input
          type="checkbox"
          data-role="auto-switch"
          ${automatic ? 'checked' : ''}
        />
      </div>

      <div class="gl-subtitle">
        典型河流地貌
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="valley"
        >
          ⛰️ V形谷
        </button>

        <button
          type="button"
          data-scenario="meander"
        >
          〰️ 曲流河段
        </button>

        <button
          type="button"
          data-scenario="floodplain"
        >
          🌾 冲积平原
        </button>

        <button
          type="button"
          data-scenario="delta"
        >
          🔺 河口三角洲
        </button>

        <button
          type="button"
          data-scenario="aggradation"
        >
          🟤 河床淤积
        </button>

        <button
          type="button"
          data-scenario="incision"
        >
          ⬇️ 河流下切
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-fluvial-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="流水作用与河流地貌演化教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="dominant-value"></strong>
          <span>优势流水作用</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="landform-value"></strong>
          <span>典型河流地貌</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="velocity-value"></strong>
          <span>相对流速</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="capacity-value"></strong>
          <span>搬运能力</span>
        </div>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var rootId='${rootId}';
      var root=document.getElementById(rootId);

      if(!root){
        return;
      }

      function query(selector){
        return root.querySelector(selector);
      }

      function queryAll(selector){
        return root.querySelectorAll(selector);
      }

      function clamp(value,min,max){
        return Math.max(
          min,
          Math.min(max,value)
        );
      }

      function roundedRect(
        context,
        x,
        y,
        width,
        height,
        radius
      ){
        var adjusted=Math.min(
          radius,
          width/2,
          height/2
        );

        context.beginPath();
        context.moveTo(
          x+adjusted,
          y
        );
        context.lineTo(
          x+width-adjusted,
          y
        );
        context.quadraticCurveTo(
          x+width,
          y,
          x+width,
          y+adjusted
        );
        context.lineTo(
          x+width,
          y+height-adjusted
        );
        context.quadraticCurveTo(
          x+width,
          y+height,
          x+width-adjusted,
          y+height
        );
        context.lineTo(
          x+adjusted,
          y+height
        );
        context.quadraticCurveTo(
          x,
          y+height,
          x,
          y+height-adjusted
        );
        context.lineTo(
          x,
          y+adjusted
        );
        context.quadraticCurveTo(
          x,
          y,
          x+adjusted,
          y
        );
        context.closePath();
      }

      function fillRoundedRect(
        context,
        x,
        y,
        width,
        height,
        radius,
        fill,
        stroke
      ){
        roundedRect(
          context,
          x,
          y,
          width,
          height,
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

      function drawText(
        context,
        text,
        x,
        y,
        size,
        color,
        weight,
        align
      ){
        context.save();

        context.font=
          (
            weight ||
            600
          )+
          ' '+
          size+
          'px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif';

        context.fillStyle=
          color ||
          '#334155';

        context.textAlign=
          align ||
          'left';

        context.textBaseline='middle';

        context.fillText(
          text,
          x,
          y
        );

        context.restore();
      }

      function drawArrowHead(
        context,
        x,
        y,
        angle,
        color,
        size
      ){
        context.save();
        context.fillStyle=color;

        context.beginPath();
        context.moveTo(x,y);

        context.lineTo(
          x-
          size*
          Math.cos(
            angle-Math.PI/6
          ),
          y-
          size*
          Math.sin(
            angle-Math.PI/6
          )
        );

        context.lineTo(
          x-
          size*
          Math.cos(
            angle+Math.PI/6
          ),
          y-
          size*
          Math.sin(
            angle+Math.PI/6
          )
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function drawArrow(
        context,
        x1,
        y1,
        x2,
        y2,
        color,
        width
      ){
        var angle=Math.atan2(
          y2-y1,
          x2-x1
        );

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=width || 3;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(x1,y1);
        context.lineTo(x2,y2);
        context.stroke();

        context.restore();

        drawArrowHead(
          context,
          x2,
          y2,
          angle,
          color,
          11
        );
      }

      function reachLabel(value){
        if(value==='upper'){
          return '上游山区';
        }

        if(value==='lower'){
          return '下游平原';
        }

        if(value==='estuary'){
          return '河口地区';
        }

        return '中游河段';
      }

      function readState(){
        return {
          reach:
            reachSelect.value,
          slope:Number(
            slopeInput.value
          ),
          discharge:Number(
            dischargeInput.value
          ),
          sediment:Number(
            sedimentInput.value
          ),
          vegetation:Number(
            vegetationInput.value
          ),
          baseLevel:Number(
            baseLevelInput.value
          ),
          observationMode:
            observationSelect.value
        };
      }

      function calculate(values){
        var slopeRatio=
          values.slope/
          100;

        var dischargeRatio=
          values.discharge/
          100;

        var sedimentRatio=
          values.sediment/
          100;

        var vegetationRatio=
          values.vegetation/
          100;

        var baseRatio=
          values.baseLevel/
          100;

        var reachFactors={
          upper:{
            erosion:25,
            transport:8,
            deposition:-18
          },
          middle:{
            erosion:8,
            transport:18,
            deposition:3
          },
          lower:{
            erosion:-8,
            transport:9,
            deposition:24
          },
          estuary:{
            erosion:-15,
            transport:3,
            deposition:36
          }
        };

        var factor=
          reachFactors[
            values.reach
          ];

        var velocity=clamp(
          12+
          values.slope*
          0.53+
          values.discharge*
          0.35-
          values.vegetation*
          0.10,
          5,
          100
        );

        var erosion=clamp(
          12+
          values.slope*
          0.52+
          values.discharge*
          0.31-
          values.vegetation*
          0.24-
          values.baseLevel*
          0.12+
          factor.erosion,
          0,
          100
        );

        var transport=clamp(
          15+
          values.discharge*
          0.48+
          values.slope*
          0.24-
          values.sediment*
          0.13+
          factor.transport,
          0,
          100
        );

        var deposition=clamp(
          8+
          values.sediment*
          0.48+
          (
            100-values.slope
          )*
          0.22+
          values.baseLevel*
          0.20-
          values.discharge*
          0.16+
          factor.deposition,
          0,
          100
        );

        var bankStability=clamp(
          12+
          values.vegetation*
          0.68-
          values.discharge*
          0.19+
          values.sediment*
          0.08,
          0,
          100
        );

        var meanderPotential=clamp(
          (
            100-values.slope
          )*
          0.42+
          values.discharge*
          0.31+
          values.sediment*
          0.18-
          values.vegetation*
          0.08,
          0,
          100
        );

        var deltaPotential=clamp(
          values.sediment*
          0.55+
          (
            100-values.slope
          )*
          0.25+
          values.baseLevel*
          0.22-
          values.discharge*
          0.10+
          (
            values.reach==='estuary'
              ? 22
              : 0
          ),
          0,
          100
        );

        var dominant='侵蚀';

        if(
          transport>=erosion &&
          transport>=deposition
        ){
          dominant='搬运';
        }else if(
          deposition>=erosion &&
          deposition>=transport
        ){
          dominant='堆积';
        }

        var landform='河谷';

        if(
          values.reach==='upper' &&
          erosion>=55
        ){
          landform='V形谷';
        }else if(
          values.reach==='middle' &&
          meanderPotential>=55
        ){
          landform='曲流河道';
        }else if(
          values.reach==='lower' &&
          deposition>=50
        ){
          landform='冲积平原';
        }else if(
          values.reach==='estuary' &&
          deltaPotential>=55
        ){
          landform='河口三角洲';
        }else if(
          dominant==='堆积'
        ){
          landform='沙洲与河漫滩';
        }else if(
          dominant==='搬运'
        ){
          landform='宽谷河道';
        }

        var capacity=clamp(
          transport+
          velocity*
          0.18,
          0,
          100
        );

        var incision=clamp(
          erosion-
          deposition+
          50,
          0,
          100
        );

        var aggradation=clamp(
          deposition-
          erosion+
          50,
          0,
          100
        );

        return {
          velocity:velocity,
          erosion:erosion,
          transport:transport,
          deposition:deposition,
          bankStability:bankStability,
          meanderPotential:
            meanderPotential,
          deltaPotential:
            deltaPotential,
          dominant:dominant,
          landform:landform,
          capacity:capacity,
          incision:incision,
          aggradation:aggradation,
          slopeRatio:slopeRatio,
          dischargeRatio:
            dischargeRatio,
          sedimentRatio:
            sedimentRatio,
          vegetationRatio:
            vegetationRatio,
          baseRatio:baseRatio
        };
      }

      function describe(values,model){
        if(model.landform==='V形谷'){
          return '上游坡度较大、流速较快，流水以下切侵蚀为主，'+
            '河谷较深而狭窄，常形成V形谷。';
        }

        if(model.landform==='曲流河道'){
          return '中游坡度减小，侧蚀作用增强，河道发生弯曲。'+
            '曲流凹岸水流较急、侵蚀较强，凸岸水流较缓、沉积较明显。';
        }

        if(model.landform==='冲积平原'){
          return '下游坡度较小，河流搬运能力下降，泥沙在河道和河漫滩堆积，'+
            '长期形成较宽广的冲积平原。';
        }

        if(model.landform==='河口三角洲'){
          return '河流进入相对静水环境后流速降低，大量泥沙在河口附近沉积，'+
            '河道可能分汊并逐渐形成三角洲。';
        }

        if(model.dominant==='侵蚀'){
          return '当前侵蚀强于堆积，河床可能发生下切，河谷加深。'+
            '坡度和流量增大通常会增强流水侵蚀能力。';
        }

        if(model.dominant==='堆积'){
          return '当前泥沙供给超过河流搬运能力，河床和河漫滩可能发生淤积。'+
            '坡度减小、流速下降或基准面升高通常有利于沉积。';
        }

        return '当前以物质搬运为主。流水把上游侵蚀产生的泥沙输送到中下游，'+
          '颗粒大小和搬运距离取决于流速与水量。';
      }

      function drawProfile(
        context,
        values,
        model
      ){
        var chart={
          x:58,
          y:87,
          width:680,
          height:248
        };

        fillRoundedRect(
          context,
          chart.x,
          chart.y,
          chart.width,
          chart.height,
          16,
          '#F8FAFC',
          '#BAE6FD'
        );

        var startHeight=
          chart.y+
          40;

        var endHeight=
          chart.y+
          chart.height-
          40-
          values.baseLevel*
          0.30;

        var curveStrength=
          65-
          values.slope*
          0.38;

        context.fillStyle='#D6B377';

        context.beginPath();
        context.moveTo(
          chart.x,
          startHeight
        );

        context.bezierCurveTo(
          chart.x+
          chart.width*
          0.20,
          startHeight+
          curveStrength,
          chart.x+
          chart.width*
          0.62,
          endHeight-
          35,
          chart.x+
          chart.width,
          endHeight
        );

        context.lineTo(
          chart.x+
          chart.width,
          chart.y+
          chart.height
        );

        context.lineTo(
          chart.x,
          chart.y+
          chart.height
        );

        context.closePath();
        context.fill();

        context.strokeStyle='#7C4A16';
        context.lineWidth=3;

        context.beginPath();
        context.moveTo(
          chart.x,
          startHeight
        );

        context.bezierCurveTo(
          chart.x+
          chart.width*
          0.20,
          startHeight+
          curveStrength,
          chart.x+
          chart.width*
          0.62,
          endHeight-
          35,
          chart.x+
          chart.width,
          endHeight
        );

        context.stroke();

        var riverPoints=[];

        for(
          var index=0;
          index<=60;
          index+=1
        ){
          var ratio=
            index/60;

          var x=
            chart.x+
            ratio*
            chart.width;

          var y=
            (
              1-ratio
            )*
            (
              1-ratio
            )*
            startHeight+
            2*
            (
              1-ratio
            )*
            ratio*
            (
              startHeight+
              curveStrength
            )+
            ratio*
            ratio*
            endHeight;

          riverPoints.push({
            x:x,
            y:y-5
          });
        }

        context.strokeStyle='#0EA5E9';
        context.lineWidth=
          5+
          values.discharge*
          0.05;
        context.lineCap='round';

        context.beginPath();

        riverPoints.forEach(
          function(point,index){
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

        context.stroke();

        var particleIndex=
          Math.floor(
            state.phase*
            (
              riverPoints.length-1
            )
          );

        var particle=
          riverPoints[
            particleIndex
          ];

        context.fillStyle='#F97316';
        context.beginPath();
        context.arc(
          particle.x,
          particle.y,
          5,
          0,
          Math.PI*2
        );
        context.fill();

        var reachPositions=[
          {
            x:chart.x+50,
            label:'上游',
            color:'#B91C1C'
          },
          {
            x:chart.x+
              chart.width*
              0.43,
            label:'中游',
            color:'#0F766E'
          },
          {
            x:chart.x+
              chart.width*
              0.76,
            label:'下游',
            color:'#0369A1'
          },
          {
            x:chart.x+
              chart.width-
              42,
            label:'河口',
            color:'#7C3AED'
          }
        ];

        reachPositions.forEach(
          function(item){
            context.strokeStyle=
              'rgba(100,116,139,0.30)';

            context.lineWidth=1;
            context.setLineDash([5,5]);

            context.beginPath();
            context.moveTo(
              item.x,
              chart.y+14
            );
            context.lineTo(
              item.x,
              chart.y+
              chart.height-
              12
            );
            context.stroke();

            context.setLineDash([]);

            drawText(
              context,
              item.label,
              item.x,
              chart.y+21,
              10,
              item.color,
              850,
              'center'
            );
          }
        );

        var selectedX=
          values.reach==='upper'
            ? chart.x+50
            : values.reach==='middle'
              ? chart.x+
                chart.width*
                0.43
              : values.reach==='lower'
                ? chart.x+
                  chart.width*
                  0.76
                : chart.x+
                  chart.width-
                  42;

        context.fillStyle=
          'rgba(250,204,21,0.24)';

        context.beginPath();
        context.arc(
          selectedX,
          chart.y+
          chart.height/
          2,
          34,
          0,
          Math.PI*2
        );
        context.fill();

        if(labelSwitch.checked){
          drawText(
            context,
            '纵剖面坡度逐渐减小',
            chart.x+
            chart.width/
            2,
            chart.y+
            chart.height-
            18,
            10,
            '#475569',
            750,
            'center'
          );

          drawArrow(
            context,
            chart.x+100,
            chart.y+62,
            chart.x+
            chart.width-
            92,
            chart.y+
            chart.height-
            73,
            '#0369A1',
            2.5
          );
        }
      }

      function drawPlan(
        context,
        values,
        model
      ){
        var area={
          x:55,
          y:77,
          width:704,
          height:278
        };

        fillRoundedRect(
          context,
          area.x,
          area.y,
          area.width,
          area.height,
          18,
          '#DCFCE7',
          '#A7D8BC'
        );

        context.save();

        var vegetationOpacity=
          0.15+
          values.vegetation/
          120;

        context.globalAlpha=
          vegetationOpacity;

        context.fillStyle='#16A34A';

        for(
          var treeIndex=0;
          treeIndex<48;
          treeIndex+=1
        ){
          var treeX=
            area.x+
            18+
            (
              treeIndex%12
            )*
            57+
            Math.sin(
              treeIndex*
              1.2
            )*
            9;

          var treeY=
            area.y+
            19+
            Math.floor(
              treeIndex/12
            )*
            78+
            Math.cos(
              treeIndex*
              1.6
            )*
            12;

          context.beginPath();
          context.arc(
            treeX,
            treeY,
            4+
            treeIndex%3,
            0,
            Math.PI*2
          );
          context.fill();
        }

        context.restore();

        var meanderAmount=
          18+
          model.meanderPotential*
          0.75;

        var channelWidth=
          16+
          values.discharge*
          0.16;

        context.strokeStyle='#0284C7';
        context.lineWidth=channelWidth;
        context.lineCap='round';
        context.lineJoin='round';

        context.beginPath();

        for(
          var index=0;
          index<=70;
          index+=1
        ){
          var ratio=
            index/70;

          var x=
            area.x+
            20+
            ratio*
            (
              area.width-40
            );

          var y=
            area.y+
            area.height/
            2+
            Math.sin(
              ratio*
              Math.PI*
              4
            )*
            meanderAmount;

          if(index===0){
            context.moveTo(x,y);
          }else{
            context.lineTo(x,y);
          }
        }

        context.stroke();

        context.strokeStyle=
          'rgba(255,255,255,0.55)';
        context.lineWidth=2;

        context.beginPath();

        for(
          var innerIndex=0;
          innerIndex<=70;
          innerIndex+=1
        ){
          var innerRatio=
            innerIndex/70;

          var innerX=
            area.x+
            20+
            innerRatio*
            (
              area.width-40
            );

          var innerY=
            area.y+
            area.height/
            2+
            Math.sin(
              innerRatio*
              Math.PI*
              4
            )*
            meanderAmount;

          if(innerIndex===0){
            context.moveTo(
              innerX,
              innerY
            );
          }else{
            context.lineTo(
              innerX,
              innerY
            );
          }
        }

        context.stroke();

        var movingRatio=state.phase;

        var particleX=
          area.x+
          20+
          movingRatio*
          (
            area.width-40
          );

        var particleY=
          area.y+
          area.height/
          2+
          Math.sin(
            movingRatio*
            Math.PI*
            4
          )*
          meanderAmount;

        context.fillStyle='#FACC15';
        context.beginPath();
        context.arc(
          particleX,
          particleY,
          6,
          0,
          Math.PI*2
        );
        context.fill();

        var bendRatio=0.375;
        var bendX=
          area.x+
          20+
          bendRatio*
          (
            area.width-40
          );

        var bendY=
          area.y+
          area.height/
          2+
          Math.sin(
            bendRatio*
            Math.PI*
            4
          )*
          meanderAmount;

        context.fillStyle=
          'rgba(220,38,38,0.30)';

        context.beginPath();
        context.arc(
          bendX,
          bendY-
          channelWidth*
          0.55,
          18,
          0,
          Math.PI*2
        );
        context.fill();

        context.fillStyle=
          'rgba(245,158,11,0.48)';

        context.beginPath();
        context.arc(
          bendX,
          bendY+
          channelWidth*
          0.54,
          21,
          0,
          Math.PI*2
        );
        context.fill();

        if(labelSwitch.checked){
          fillRoundedRect(
            context,
            bendX-78,
            bendY-89,
            92,
            25,
            12,
            'rgba(254,226,226,0.92)',
            '#FCA5A5'
          );

          drawText(
            context,
            '凹岸侵蚀',
            bendX-32,
            bendY-76,
            10,
            '#B91C1C',
            850,
            'center'
          );

          fillRoundedRect(
            context,
            bendX+14,
            bendY+61,
            92,
            25,
            12,
            'rgba(254,243,199,0.94)',
            '#FCD34D'
          );

          drawText(
            context,
            '凸岸堆积',
            bendX+60,
            bendY+74,
            10,
            '#B45309',
            850,
            'center'
          );

          drawText(
            context,
            '水流较快',
            bendX-35,
            bendY-44,
            9,
            '#DC2626',
            750,
            'center'
          );

          drawText(
            context,
            '水流较缓',
            bendX+64,
            bendY+43,
            9,
            '#92400E',
            750,
            'center'
          );
        }

        if(
          values.reach==='estuary' ||
          model.deltaPotential>=70
        ){
          var deltaX=
            area.x+
            area.width-
            28;

          context.fillStyle=
            'rgba(217,119,6,0.72)';

          context.beginPath();
          context.moveTo(
            deltaX-92,
            area.y+
            area.height/
            2-
            35
          );
          context.lineTo(
            deltaX,
            area.y+
            area.height/
            2-
            95
          );
          context.lineTo(
            deltaX,
            area.y+
            area.height/
            2+
            96
          );
          context.lineTo(
            deltaX-92,
            area.y+
            area.height/
            2+
            35
          );
          context.closePath();
          context.fill();

          context.strokeStyle='#0284C7';
          context.lineWidth=8;

          [
            -42,
            0,
            42
          ].forEach(
            function(offset){
              context.beginPath();
              context.moveTo(
                deltaX-92,
                area.y+
                area.height/
                2
              );
              context.lineTo(
                deltaX,
                area.y+
                area.height/
                2+
                offset
              );
              context.stroke();
            }
          );

          if(labelSwitch.checked){
            drawText(
              context,
              '分汊河道与三角洲沉积',
              deltaX-80,
              area.y+24,
              10,
              '#92400E',
              850,
              'center'
            );
          }
        }
      }

      function drawProcess(
        context,
        values,
        model
      ){
        var sections=[
          {
            key:'erosion',
            label:'侵蚀',
            color:'#DC2626',
            value:model.erosion,
            x:52
          },
          {
            key:'transport',
            label:'搬运',
            color:'#0369A1',
            value:model.transport,
            x:298
          },
          {
            key:'deposition',
            label:'堆积',
            color:'#D97706',
            value:model.deposition,
            x:544
          }
        ];

        sections.forEach(
          function(section){
            var selected=
              model.dominant===
              section.label;

            fillRoundedRect(
              context,
              section.x,
              83,
              220,
              244,
              16,
              selected
                ? 'rgba(255,247,237,0.98)'
                : '#F8FAFC',
              selected
                ? section.color
                : '#CBD5E1'
            );

            drawText(
              context,
              section.label,
              section.x+110,
              112,
              16,
              section.color,
              900,
              'center'
            );

            drawText(
              context,
              Math.round(
                section.value
              )+
              '%',
              section.x+110,
              147,
              26,
              section.color,
              900,
              'center'
            );

            context.fillStyle='#E2E8F0';

            context.fillRect(
              section.x+32,
              176,
              156,
              14
            );

            context.fillStyle=
              section.color;

            context.fillRect(
              section.x+32,
              176,
              clamp(
                section.value/
                100*
                156,
                3,
                156
              ),
              14
            );
          }
        );

        context.save();

        context.strokeStyle='#DC2626';
        context.lineWidth=
          3+
          model.erosion/
          25;

        for(
          var erosionIndex=0;
          erosionIndex<7;
          erosionIndex+=1
        ){
          var rockX=
            82+
            erosionIndex*
            24;

          var rockY=
            244+
            Math.sin(
              erosionIndex
            )*
            13;

          context.beginPath();
          context.moveTo(
            rockX-7,
            rockY-10
          );
          context.lineTo(
            rockX+8,
            rockY
          );
          context.lineTo(
            rockX-4,
            rockY+11
          );
          context.stroke();
        }

        context.restore();

        var particleCount=
          Math.round(
            4+
            model.transport/
            10
          );

        for(
          var particleIndex=0;
          particleIndex<particleCount;
          particleIndex+=1
        ){
          var transportPhase=
            (
              state.phase+
              particleIndex/
              particleCount
            )%
            1;

          var transportX=
            326+
            transportPhase*
            164;

          var transportY=
            246+
            Math.sin(
              transportPhase*
              Math.PI*
              4
            )*
            19;

          context.fillStyle=
            particleIndex%2===0
              ? '#F59E0B'
              : '#92400E';

          context.beginPath();
          context.arc(
            transportX,
            transportY,
            3+
            particleIndex%3,
            0,
            Math.PI*2
          );
          context.fill();
        }

        var depositLayers=
          Math.round(
            2+
            model.deposition/
            16
          );

        for(
          var layer=0;
          layer<depositLayers;
          layer+=1
        ){
          context.fillStyle=
            layer%2===0
              ? '#D6B377'
              : '#C7924B';

          context.fillRect(
            580,
            289-
            layer*
            11,
            150-
            layer*
            9,
            9
          );
        }

        drawText(
          context,
          '河床下切与河岸冲刷',
          162,
          303,
          9.5,
          '#B91C1C',
          750,
          'center'
        );

        drawText(
          context,
          '推移、跃移与悬移',
          408,
          303,
          9.5,
          '#075985',
          750,
          'center'
        );

        drawText(
          context,
          '沙洲、河漫滩与冲积层',
          654,
          303,
          9.5,
          '#92400E',
          750,
          'center'
        );

        drawArrow(
          context,
          250,
          215,
          292,
          215,
          '#64748B',
          3
        );

        drawArrow(
          context,
          496,
          215,
          538,
          215,
          '#64748B',
          3
        );
      }

      function render(){
        if(!root.isConnected){
          if(state.raf){
            cancelAnimationFrame(
              state.raf
            );

            state.raf=0;
          }

          if(state.timer){
            window.clearTimeout(
              state.timer
            );

            state.timer=null;
          }

          return;
        }

        var values=readState();
        var model=calculate(values);
        var context=
          canvas.getContext('2d');

        if(!context){
          return;
        }

        slopeValue.textContent=
          Math.round(
            values.slope
          )+
          '%';

        dischargeValue.textContent=
          Math.round(
            values.discharge
          )+
          '%';

        sedimentValue.textContent=
          Math.round(
            values.sediment
          )+
          '%';

        vegetationValue.textContent=
          Math.round(
            values.vegetation
          )+
          '%';

        baseLevelValue.textContent=
          Math.round(
            values.baseLevel
          )+
          '%';

        dominantValue.textContent=
          model.dominant;

        landformValue.textContent=
          model.landform;

        velocityValue.textContent=
          Math.round(
            model.velocity
          )+
          '%';

        capacityValue.textContent=
          Math.round(
            model.capacity
          )+
          '%';

        result.textContent=
          reachLabel(
            values.reach
          )+
          '当前以'+
          model.dominant+
          '作用较突出，典型地貌为'+
          model.landform+
          '。'+
          describe(
            values,
            model
          );

        context.clearRect(
          0,
          0,
          canvas.width,
          canvas.height
        );

        var background=
          context.createLinearGradient(
            0,
            0,
            0,
            canvas.height
          );

        background.addColorStop(
          0,
          '#E0F2FE'
        );

        background.addColorStop(
          0.52,
          '#F8FAFC'
        );

        background.addColorStop(
          1,
          '#DCFCE7'
        );

        context.fillStyle=background;

        context.fillRect(
          0,
          0,
          canvas.width,
          canvas.height
        );

        fillRoundedRect(
          context,
          18,
          17,
          784,
          376,
          17,
          'rgba(255,255,255,0.92)',
          '#A7D8BC'
        );

        drawText(
          context,
          reachLabel(
            values.reach
          )+
          ' · '+
          model.landform,
          40,
          43,
          14,
          '#14532D',
          850,
          'left'
        );

        drawText(
          context,
          '地貌比例、速率和演化时间均为课堂示意',
          780,
          43,
          9.5,
          '#64748B',
          650,
          'right'
        );

        if(
          values.observationMode===
          'plan'
        ){
          drawPlan(
            context,
            values,
            model
          );
        }else if(
          values.observationMode===
          'process'
        ){
          drawProcess(
            context,
            values,
            model
          );
        }else{
          drawProfile(
            context,
            values,
            model
          );
        }

        drawText(
          context,
          '河流地貌由侵蚀、搬运和堆积长期共同塑造，本图不用于真实河道与工程判断。',
          410,
          414,
          9.5,
          '#64748B',
          650,
          'center'
        );
      }

      function clearScenarioSelection(){
        Array.prototype.forEach.call(
          scenarioButtons,
          function(button){
            button.classList.remove(
              'active'
            );
          }
        );

        render();
      }

      function applyScenario(name){
        var scenarios={
          valley:{
            reach:'upper',
            slope:92,
            discharge:48,
            sediment:42,
            vegetation:68,
            baseLevel:30,
            observationMode:'profile'
          },
          meander:{
            reach:'middle',
            slope:28,
            discharge:76,
            sediment:52,
            vegetation:38,
            baseLevel:48,
            observationMode:'plan'
          },
          floodplain:{
            reach:'lower',
            slope:11,
            discharge:82,
            sediment:74,
            vegetation:48,
            baseLevel:66,
            observationMode:'profile'
          },
          delta:{
            reach:'estuary',
            slope:4,
            discharge:68,
            sediment:92,
            vegetation:35,
            baseLevel:82,
            observationMode:'plan'
          },
          aggradation:{
            reach:'lower',
            slope:8,
            discharge:38,
            sediment:95,
            vegetation:32,
            baseLevel:88,
            observationMode:'process'
          },
          incision:{
            reach:'upper',
            slope:86,
            discharge:88,
            sediment:24,
            vegetation:18,
            baseLevel:18,
            observationMode:'process'
          }
        };

        var scenario=scenarios[name];

        if(!scenario){
          return;
        }

        reachSelect.value=
          scenario.reach;

        slopeInput.value=String(
          scenario.slope
        );

        dischargeInput.value=String(
          scenario.discharge
        );

        sedimentInput.value=String(
          scenario.sediment
        );

        vegetationInput.value=String(
          scenario.vegetation
        );

        baseLevelInput.value=String(
          scenario.baseLevel
        );

        observationSelect.value=
          scenario.observationMode;

        Array.prototype.forEach.call(
          scenarioButtons,
          function(button){
            button.classList.toggle(
              'active',
              button.getAttribute(
                'data-scenario'
              )===name
            );
          }
        );

        render();
      }

      function schedule(){
        if(state.timer){
          window.clearTimeout(
            state.timer
          );

          state.timer=null;
        }

        if(
          !autoSwitch.checked ||
          !root.isConnected
        ){
          return;
        }

        state.timer=window.setTimeout(
          function(){
            if(!root.isConnected){
              return;
            }

            state.scenarioIndex=
              (
                state.scenarioIndex+
                1
              )%
              scenarioOrder.length;

            applyScenario(
              scenarioOrder[
                state.scenarioIndex
              ]
            );

            schedule();
          },
          3100
        );
      }

      function animate(timestamp){
        if(!root.isConnected){
          state.raf=0;
          return;
        }

        if(!state.startedAt){
          state.startedAt=timestamp;
        }

        state.phase=
          (
            timestamp-
            state.startedAt
          )/
          3800%
          1;

        render();

        state.raf=
          requestAnimationFrame(
            animate
          );
      }

      var reachSelect=query(
        '[data-role="river-reach"]'
      );

      var slopeInput=query(
        '[data-role="slope"]'
      );

      var dischargeInput=query(
        '[data-role="discharge"]'
      );

      var sedimentInput=query(
        '[data-role="sediment"]'
      );

      var vegetationInput=query(
        '[data-role="vegetation"]'
      );

      var baseLevelInput=query(
        '[data-role="base-level"]'
      );

      var observationSelect=query(
        '[data-role="observation-mode"]'
      );

      var labelSwitch=query(
        '[data-role="label-switch"]'
      );

      var autoSwitch=query(
        '[data-role="auto-switch"]'
      );

      var scenarioButtons=queryAll(
        '[data-scenario]'
      );

      var result=query(
        '[data-role="result"]'
      );

      var canvas=query(
        '[data-role="canvas"]'
      );

      var slopeValue=query(
        '[data-role="slope-value"]'
      );

      var dischargeValue=query(
        '[data-role="discharge-value"]'
      );

      var sedimentValue=query(
        '[data-role="sediment-value"]'
      );

      var vegetationValue=query(
        '[data-role="vegetation-value"]'
      );

      var baseLevelValue=query(
        '[data-role="base-level-value"]'
      );

      var dominantValue=query(
        '[data-role="dominant-value"]'
      );

      var landformValue=query(
        '[data-role="landform-value"]'
      );

      var velocityValue=query(
        '[data-role="velocity-value"]'
      );

      var capacityValue=query(
        '[data-role="capacity-value"]'
      );

      if(
        !reachSelect ||
        !slopeInput ||
        !dischargeInput ||
        !sedimentInput ||
        !vegetationInput ||
        !baseLevelInput ||
        !observationSelect ||
        !labelSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !result ||
        !canvas ||
        !slopeValue ||
        !dischargeValue ||
        !sedimentValue ||
        !vegetationValue ||
        !baseLevelValue ||
        !dominantValue ||
        !landformValue ||
        !velocityValue ||
        !capacityValue
      ){
        return;
      }

      var scenarioOrder=[
        'valley',
        'meander',
        'floodplain',
        'delta',
        'aggradation',
        'incision'
      ];

      var state={
        phase:0,
        startedAt:0,
        raf:0,
        timer:null,
        scenarioIndex:-1
      };

      [
        slopeInput,
        dischargeInput,
        sedimentInput,
        vegetationInput,
        baseLevelInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            clearScenarioSelection
          );
        }
      );

      reachSelect.addEventListener(
        'change',
        clearScenarioSelection
      );

      observationSelect.addEventListener(
        'change',
        clearScenarioSelection
      );

      labelSwitch.addEventListener(
        'change',
        render
      );

      autoSwitch.addEventListener(
        'change',
        function(){
          schedule();
          render();
        }
      );

      Array.prototype.forEach.call(
        scenarioButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              var name=
                button.getAttribute(
                  'data-scenario'
                ) ||
                'valley';

              state.scenarioIndex=
                scenarioOrder.indexOf(
                  name
                );

              applyScenario(name);
              schedule();
            }
          );
        }
      );

      render();
      schedule();

      state.raf=
        requestAnimationFrame(
          animate
        );
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_FLUVIAL_LANDFORMS:
GeographyLabTemplate[] = [
  {
    id: 'geography-fluvial-erosion-transport-deposition-landforms',
    group: '⛰️ 地质作用与地貌演化',
    name: '流水侵蚀、搬运、堆积与河流地貌演化',
    emoji: '🏞️',
    desc: '调节坡度、流量、含沙量、植被和基准面，观察V形谷、曲流、冲积平原和河口三角洲的形成。',
    params: [
      {
        key: 'riverReach',
        label: '初始河段位置',
        type: 'select',
        options: [
          {
            label: '上游山区',
            value: 'upper',
          },
          {
            label: '中游河段',
            value: 'middle',
          },
          {
            label: '下游平原',
            value: 'lower',
          },
          {
            label: '河口地区',
            value: 'estuary',
          },
        ],
        defaultValue: 'middle',
      },
      {
        key: 'slope',
        label: '河床坡度',
        type: 'number',
        min: 1,
        max: 100,
        step: 1,
        defaultValue: 52,
        hint: '坡度越大，河流重力势能和下切侵蚀条件通常越强。',
      },
      {
        key: 'discharge',
        label: '河流流量',
        type: 'number',
        min: 5,
        max: 100,
        step: 1,
        defaultValue: 62,
        hint: '流量增加通常会增强流速、侵蚀和搬运能力。',
      },
      {
        key: 'sedimentLoad',
        label: '河流含沙量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 55,
        hint: '泥沙供给较多而搬运能力不足时更容易发生堆积。',
      },
      {
        key: 'vegetation',
        label: '河岸植被覆盖',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 48,
        hint: '植被根系通常有助于增强河岸稳定性、减弱坡面侵蚀。',
      },
      {
        key: 'baseLevel',
        label: '侵蚀基准面',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 50,
        hint: '基准面降低有利于河流下切，升高可能促进河床堆积。',
      },
      {
        key: 'observationMode',
        label: '初始观察模式',
        type: 'select',
        options: [
          {
            label: '河流纵剖面',
            value: 'profile',
          },
          {
            label: '河道平面形态',
            value: 'plan',
          },
          {
            label: '侵蚀搬运堆积',
            value: 'process',
          },
        ],
        defaultValue: 'profile',
      },
      {
        key: 'showLabels',
        label: '显示过程与地貌标注',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'automatic',
        label: '自动演示典型河段',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildFluvialLandformsHTML,
  },
]
