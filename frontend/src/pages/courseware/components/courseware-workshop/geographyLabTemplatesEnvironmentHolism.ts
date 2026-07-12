/**
 * geographyLabTemplatesEnvironmentHolism.ts
 *
 * 第37批B1：自然环境要素相互作用与整体性。
 *
 * 教学目标：
 * 1. 识别气候、水文、地貌、土壤和植被等自然环境要素；
 * 2. 理解各要素之间通过物质迁移和能量交换相互联系；
 * 3. 观察气温、降水、坡度、植被和人类扰动变化引起的连锁反应；
 * 4. 理解一个自然环境要素发生变化，可能导致其他要素共同变化；
 * 5. 比较森林恢复、持续干旱、气候变暖和植被破坏等典型情境；
 * 6. 理解自然环境整体性不等于各要素变化幅度完全相同；
 * 7. 认识自然环境具有一定调节能力，但调节能力存在限度。
 *
 * 教学边界：
 * - 所有气温、降水、径流、土壤水分、侵蚀和生态稳定性均为相对教学量；
 * - 本模板不对应任何真实流域、生态系统、保护区或工程项目；
 * - 不考虑具体岩性、土壤类型、物种组成、极端天气和时间滞后；
 * - 不用于生态评价、环境影响评价、灾害预测、工程决策或土地规划；
 * - 真实自然环境响应具有空间差异、时间滞后、阈值和不确定性。
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

function buildEnvironmentHolismHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const temperature = Math.max(
    -5,
    Math.min(
      35,
      numberValue(params, 'temperature', 18),
    ),
  )

  const precipitation = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'precipitation', 62),
    ),
  )

  const slope = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'slope', 42),
    ),
  )

  const vegetation = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'vegetation', 68),
    ),
  )

  const humanDisturbance = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'humanDisturbance', 22),
    ),
  )

  const requestedMode = stringValue(
    params,
    'observationMode',
    'network',
  )

  const observationMode = [
    'network',
    'landscape',
    'response',
  ].includes(requestedMode)
    ? requestedMode
    : 'network'

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
<div id="${rootId}" class="gl-environment-holism-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border-radius:18px;
      border:1px solid #A7D8BC;
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
        #E0F2FE 52%,
        #FEF3C7
      );
      border-bottom:1px solid #A7D8BC;
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

    #${rootId} .gl-environment-canvas{
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
      🌍
    </div>

    <div>
      <div class="gl-title">
        自然环境要素相互作用与整体性
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        调节气温、降水、坡度、植被和人类扰动，观察自然环境连锁响应
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 不用于真实环境评价
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            年平均气温
          </span>

          <span
            class="gl-value"
            data-role="temperature-value"
          ></span>
        </div>

        <input
          type="range"
          min="-5"
          max="35"
          step="1"
          value="${shortNumber(temperature)}"
          data-role="temperature"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            降水条件
          </span>

          <span
            class="gl-value"
            data-role="precipitation-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(precipitation)}"
          data-role="precipitation"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            地形坡度
          </span>

          <span
            class="gl-value"
            data-role="slope-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(slope)}"
          data-role="slope"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            植被覆盖
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
            人类扰动强度
          </span>

          <span
            class="gl-value"
            data-role="disturbance-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(humanDisturbance)}"
          data-role="disturbance"
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
            value="network"
            ${observationMode === 'network' ? 'selected' : ''}
          >
            要素关系网络
          </option>

          <option
            value="landscape"
            ${observationMode === 'landscape' ? 'selected' : ''}
          >
            景观剖面响应
          </option>

          <option
            value="response"
            ${observationMode === 'response' ? 'selected' : ''}
          >
            连锁变化比较
          </option>
        </select>
      </div>

      <div class="gl-switch-row">
        <span>显示要素与过程标注</span>

        <input
          type="checkbox"
          data-role="label-switch"
          ${showLabels ? 'checked' : ''}
        />
      </div>

      <div class="gl-switch-row">
        <span>自动演示典型情境</span>

        <input
          type="checkbox"
          data-role="auto-switch"
          ${automatic ? 'checked' : ''}
        />
      </div>

      <div class="gl-subtitle">
        典型整体性情境
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="forest"
        >
          🌳 森林恢复
        </button>

        <button
          type="button"
          data-scenario="drought"
        >
          ☀️ 持续干旱
        </button>

        <button
          type="button"
          data-scenario="warming"
        >
          🌡️ 气候变暖
        </button>

        <button
          type="button"
          data-scenario="deforestation"
        >
          🪓 植被破坏
        </button>

        <button
          type="button"
          data-scenario="humid"
        >
          🌧️ 湿润增强
        </button>

        <button
          type="button"
          data-scenario="recovery"
        >
          ♻️ 生态恢复
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-environment-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="自然环境要素相互作用与整体性教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="water-value"></strong>
          <span>土壤水分</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="runoff-value"></strong>
          <span>地表径流</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="erosion-value"></strong>
          <span>侵蚀压力</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="stability-value"></strong>
          <span>环境稳定性</span>
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

      function drawRelation(
        context,
        startX,
        startY,
        endX,
        endY,
        color,
        strength,
        phase
      ){
        var angle=Math.atan2(
          endY-startY,
          endX-startX
        );

        var shortenedStartX=
          startX+
          Math.cos(angle)*
          43;

        var shortenedStartY=
          startY+
          Math.sin(angle)*
          43;

        var shortenedEndX=
          endX-
          Math.cos(angle)*
          45;

        var shortenedEndY=
          endY-
          Math.sin(angle)*
          45;

        context.save();
        context.strokeStyle=color;
        context.lineWidth=
          1.5+
          strength*
          3.5;
        context.globalAlpha=
          0.30+
          strength*
          0.70;
        context.lineCap='round';
        context.setLineDash([8,6]);
        context.lineDashOffset=
          -phase*
          (
            20+
            strength*
            38
          );

        context.beginPath();
        context.moveTo(
          shortenedStartX,
          shortenedStartY
        );
        context.lineTo(
          shortenedEndX,
          shortenedEndY
        );
        context.stroke();
        context.restore();

        drawArrowHead(
          context,
          shortenedEndX,
          shortenedEndY,
          angle,
          color,
          8+
          strength*
          5
        );
      }

      function drawNode(
        context,
        x,
        y,
        radius,
        title,
        subtitle,
        color,
        value,
        selected
      ){
        context.save();

        if(selected){
          context.shadowColor=color;
          context.shadowBlur=19;
        }

        context.globalAlpha=0.96;
        context.fillStyle='#FFFFFF';
        context.strokeStyle=color;
        context.lineWidth=
          selected
            ? 4
            : 2.3;

        context.beginPath();
        context.arc(
          x,
          y,
          radius,
          0,
          Math.PI*2
        );
        context.fill();
        context.stroke();

        context.restore();

        context.fillStyle=
          color.replace(
            ')',
            ',0.12)'
          );

        context.beginPath();
        context.arc(
          x,
          y,
          radius-8,
          0,
          Math.PI*2
        );
        context.fill();

        drawText(
          context,
          title,
          x,
          y-10,
          12,
          color,
          900,
          'center'
        );

        drawText(
          context,
          subtitle,
          x,
          y+10,
          8.5,
          '#64748B',
          700,
          'center'
        );

        drawText(
          context,
          Math.round(value)+'%',
          x,
          y+29,
          10,
          color,
          850,
          'center'
        );
      }

      function readState(){
        return {
          temperature:Number(
            temperatureInput.value
          ),
          precipitation:Number(
            precipitationInput.value
          ),
          slope:Number(
            slopeInput.value
          ),
          vegetation:Number(
            vegetationInput.value
          ),
          disturbance:Number(
            disturbanceInput.value
          ),
          observationMode:
            observationSelect.value
        };
      }

      function calculate(values){
        var heatIndex=clamp(
          (
            values.temperature+
            5
          )/
          40*
          100,
          0,
          100
        );

        var evaporation=clamp(
          heatIndex*
          0.63+
          (
            100-values.precipitation
          )*
          0.18+
          (
            100-values.vegetation
          )*
          0.11,
          0,
          100
        );

        var soilWater=clamp(
          values.precipitation*
          0.72+
          values.vegetation*
          0.24-
          evaporation*
          0.34-
          values.slope*
          0.13-
          values.disturbance*
          0.17+
          20,
          0,
          100
        );

        var infiltration=clamp(
          18+
          values.vegetation*
          0.52+
          soilWater*
          0.24-
          values.slope*
          0.15-
          values.disturbance*
          0.31,
          0,
          100
        );

        var runoff=clamp(
          values.precipitation*
          0.48+
          values.slope*
          0.31+
          values.disturbance*
          0.24-
          values.vegetation*
          0.34-
          infiltration*
          0.12+
          18,
          0,
          100
        );

        var erosion=clamp(
          runoff*
          0.46+
          values.slope*
          0.39+
          values.disturbance*
          0.25-
          values.vegetation*
          0.40,
          0,
          100
        );

        var soilQuality=clamp(
          24+
          soilWater*
          0.38+
          values.vegetation*
          0.34-
          erosion*
          0.31-
          values.disturbance*
          0.20,
          0,
          100
        );

        var vegetationPotential=clamp(
          values.precipitation*
          0.42+
          soilWater*
          0.31+
          soilQuality*
          0.20-
          Math.abs(
            values.temperature-
            20
          )*
          1.25-
          values.disturbance*
          0.28+
          18,
          0,
          100
        );

        var hydrology=clamp(
          soilWater*
          0.42+
          infiltration*
          0.25+
          (
            100-runoff
          )*
          0.18+
          values.precipitation*
          0.15,
          0,
          100
        );

        var landformStability=clamp(
          36+
          values.vegetation*
          0.36-
          values.slope*
          0.26-
          erosion*
          0.31-
          values.disturbance*
          0.19+
          soilQuality*
          0.18,
          0,
          100
        );

        var stability=clamp(
          vegetationPotential*
          0.25+
          soilQuality*
          0.23+
          hydrology*
          0.20+
          landformStability*
          0.20+
          (
            100-values.disturbance
          )*
          0.12,
          0,
          100
        );

        var resilience=clamp(
          stability*
          0.58+
          values.vegetation*
          0.27+
          soilWater*
          0.15,
          0,
          100
        );

        var pressure=clamp(
          (
            100-stability
          )*
          0.62+
          values.disturbance*
          0.38,
          0,
          100
        );

        var dominantChange='环境相对协调';

        if(erosion>=68){
          dominantChange='侵蚀压力突出';
        }else if(soilWater<=28){
          dominantChange='水分限制明显';
        }else if(vegetationPotential<=32){
          dominantChange='植被退化风险';
        }else if(values.disturbance>=68){
          dominantChange='人类扰动突出';
        }else if(stability>=76){
          dominantChange='整体稳定性较高';
        }

        return {
          heatIndex:heatIndex,
          evaporation:evaporation,
          soilWater:soilWater,
          infiltration:infiltration,
          runoff:runoff,
          erosion:erosion,
          soilQuality:soilQuality,
          vegetationPotential:
            vegetationPotential,
          hydrology:hydrology,
          landformStability:
            landformStability,
          stability:stability,
          resilience:resilience,
          pressure:pressure,
          dominantChange:
            dominantChange
        };
      }

      function describe(values,model){
        if(
          values.disturbance>=65 &&
          values.vegetation<=40
        ){
          return '植被减少和地表扰动会降低截留与下渗，增加地表径流和土壤侵蚀；'+
            '土壤质量下降又会进一步限制植被恢复，形成连锁变化。';
        }

        if(
          values.precipitation<=28
        ){
          return '降水减少使土壤水分下降，植被生长受到限制，'+
            '地表覆盖减弱后可能增加风化、侵蚀和生态系统波动。';
        }

        if(
          values.temperature>=29
        ){
          return '气温升高会增强蒸发需求。若降水没有同步增加，'+
            '土壤水分和植被潜力可能下降，并进一步影响径流与土壤形成。';
        }

        if(
          values.vegetation>=78 &&
          values.disturbance<=25
        ){
          return '较高植被覆盖有助于截留降水、增加下渗、稳定土壤和减弱侵蚀；'+
            '土壤与水分条件改善又会支持植被生长，表现出要素间相互促进。';
        }

        if(model.erosion>=65){
          return '当前坡度、径流和地表扰动共同增强侵蚀。'+
            '侵蚀会改变地貌和土壤厚度，并可能降低植被恢复能力。';
        }

        return '气候、水文、地貌、土壤和植被通过水分、热量和物质迁移相互联系。'+
          '任一要素变化都可能引起其他要素调整，但不同要素响应速度并不完全相同。';
      }

      function drawNetwork(
        context,
        values,
        model
      ){
        var nodes={
          climate:{
            x:410,
            y:104,
            title:'气候',
            subtitle:'热量与水分',
            color:'rgba(234,88,12,1)',
            value:
              (
                model.heatIndex+
                values.precipitation
              )/
              2
          },
          water:{
            x:620,
            y:205,
            title:'水文',
            subtitle:'径流与下渗',
            color:'rgba(2,132,199,1)',
            value:model.hydrology
          },
          landform:{
            x:545,
            y:337,
            title:'地貌',
            subtitle:'坡度与侵蚀',
            color:'rgba(146,64,14,1)',
            value:model.landformStability
          },
          soil:{
            x:275,
            y:337,
            title:'土壤',
            subtitle:'水分与肥力',
            color:'rgba(161,98,7,1)',
            value:model.soilQuality
          },
          vegetation:{
            x:200,
            y:205,
            title:'植被',
            subtitle:'覆盖与生态',
            color:'rgba(22,163,74,1)',
            value:model.vegetationPotential
          }
        };

        var relationStrengths={
          climateWater:
            clamp(
              values.precipitation/
              100,
              0.15,
              1
            ),
          climateVegetation:
            clamp(
              model.vegetationPotential/
              100,
              0.15,
              1
            ),
          waterSoil:
            clamp(
              model.soilWater/
              100,
              0.15,
              1
            ),
          soilVegetation:
            clamp(
              model.soilQuality/
              100,
              0.15,
              1
            ),
          vegetationLandform:
            clamp(
              values.vegetation/
              100,
              0.15,
              1
            ),
          landformWater:
            clamp(
              (
                model.runoff+
                values.slope
              )/
              200,
              0.15,
              1
            ),
          climateSoil:
            clamp(
              (
                model.evaporation+
                values.precipitation
              )/
              200,
              0.15,
              1
            ),
          waterVegetation:
            clamp(
              model.soilWater/
              100,
              0.15,
              1
            )
        };

        drawRelation(
          context,
          nodes.climate.x,
          nodes.climate.y,
          nodes.water.x,
          nodes.water.y,
          '#0EA5E9',
          relationStrengths.climateWater,
          state.phase
        );

        drawRelation(
          context,
          nodes.water.x,
          nodes.water.y,
          nodes.landform.x,
          nodes.landform.y,
          '#0369A1',
          relationStrengths.landformWater,
          state.phase
        );

        drawRelation(
          context,
          nodes.landform.x,
          nodes.landform.y,
          nodes.soil.x,
          nodes.soil.y,
          '#A16207',
          relationStrengths.vegetationLandform,
          state.phase
        );

        drawRelation(
          context,
          nodes.soil.x,
          nodes.soil.y,
          nodes.vegetation.x,
          nodes.vegetation.y,
          '#16A34A',
          relationStrengths.soilVegetation,
          state.phase
        );

        drawRelation(
          context,
          nodes.vegetation.x,
          nodes.vegetation.y,
          nodes.climate.x,
          nodes.climate.y,
          '#EA580C',
          relationStrengths.climateVegetation,
          state.phase
        );

        drawRelation(
          context,
          nodes.water.x,
          nodes.water.y,
          nodes.soil.x,
          nodes.soil.y,
          '#0284C7',
          relationStrengths.waterSoil,
          state.phase
        );

        drawRelation(
          context,
          nodes.soil.x,
          nodes.soil.y,
          nodes.climate.x,
          nodes.climate.y,
          '#D97706',
          relationStrengths.climateSoil,
          state.phase
        );

        drawRelation(
          context,
          nodes.vegetation.x,
          nodes.vegetation.y,
          nodes.water.x,
          nodes.water.y,
          '#059669',
          relationStrengths.waterVegetation,
          state.phase
        );

        Object.keys(nodes).forEach(
          function(key){
            var node=nodes[key];

            drawNode(
              context,
              node.x,
              node.y,
              52,
              node.title,
              node.subtitle,
              node.color,
              node.value,
              false
            );
          }
        );

        fillRoundedRect(
          context,
          333,
          184,
          154,
          62,
          15,
          'rgba(255,255,255,0.95)',
          '#A7D8BC'
        );

        drawText(
          context,
          '自然环境整体',
          410,
          204,
          13,
          '#14532D',
          900,
          'center'
        );

        drawText(
          context,
          model.dominantChange,
          410,
          226,
          9.5,
          '#64748B',
          750,
          'center'
        );

        if(labelSwitch.checked){
          drawText(
            context,
            '能量交换',
            518,
            127,
            9,
            '#C2410C',
            750,
            'center'
          );

          drawText(
            context,
            '水分循环',
            583,
            280,
            9,
            '#0369A1',
            750,
            'center'
          );

          drawText(
            context,
            '物质迁移',
            410,
            353,
            9,
            '#92400E',
            750,
            'center'
          );

          drawText(
            context,
            '生态反馈',
            237,
            273,
            9,
            '#15803D',
            750,
            'center'
          );
        }
      }

      function drawTree(
        context,
        x,
        y,
        scale,
        health
      ){
        context.save();

        context.fillStyle='#854D0E';

        context.fillRect(
          x-3*scale,
          y-24*scale,
          6*scale,
          28*scale
        );

        context.globalAlpha=
          0.25+
          health*
          0.75;

        context.fillStyle=
          health>=0.55
            ? '#16A34A'
            : health>=0.30
              ? '#84CC16'
              : '#A16207';

        context.beginPath();
        context.arc(
          x,
          y-35*scale,
          16*scale,
          0,
          Math.PI*2
        );

        context.arc(
          x-10*scale,
          y-27*scale,
          11*scale,
          0,
          Math.PI*2
        );

        context.arc(
          x+10*scale,
          y-27*scale,
          11*scale,
          0,
          Math.PI*2
        );

        context.fill();
        context.restore();
      }

      function drawLandscape(
        context,
        values,
        model
      ){
        var sky=
          context.createLinearGradient(
            0,
            70,
            0,
            236
          );

        sky.addColorStop(
          0,
          values.precipitation>=55
            ? '#BAE6FD'
            : '#FDE68A'
        );

        sky.addColorStop(
          1,
          '#F8FAFC'
        );

        fillRoundedRect(
          context,
          50,
          76,
          720,
          280,
          17,
          sky,
          '#A7D8BC'
        );

        var mountainTop=
          118+
          (
            100-values.slope
          )*
          0.60;

        context.fillStyle='#9A7147';

        context.beginPath();
        context.moveTo(50,285);
        context.lineTo(170,mountainTop);
        context.lineTo(278,286);
        context.lineTo(382,177);
        context.lineTo(495,286);
        context.lineTo(770,286);
        context.lineTo(770,356);
        context.lineTo(50,356);
        context.closePath();
        context.fill();

        context.fillStyle='#65A30D';
        context.globalAlpha=
          0.20+
          values.vegetation/
          125;

        context.beginPath();
        context.moveTo(50,280);
        context.lineTo(170,mountainTop-2);
        context.lineTo(278,280);
        context.lineTo(382,174);
        context.lineTo(495,280);
        context.lineTo(770,280);
        context.lineTo(770,305);
        context.lineTo(50,305);
        context.closePath();
        context.fill();

        context.globalAlpha=1;

        var riverWidth=
          9+
          model.runoff*
          0.10;

        context.strokeStyle='#0284C7';
        context.lineWidth=riverWidth;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(193,190);
        context.bezierCurveTo(
          255,
          244,
          346,
          247,
          424,
          299
        );
        context.bezierCurveTo(
          510,
          353,
          623,
          314,
          748,
          340
        );
        context.stroke();

        context.strokeStyle=
          'rgba(255,255,255,0.55)';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(193,190);
        context.bezierCurveTo(
          255,
          244,
          346,
          247,
          424,
          299
        );
        context.bezierCurveTo(
          510,
          353,
          623,
          314,
          748,
          340
        );
        context.stroke();

        var treeCount=Math.round(
          values.vegetation/
          4
        );

        var health=
          model.vegetationPotential/
          100;

        for(
          var treeIndex=0;
          treeIndex<treeCount;
          treeIndex+=1
        ){
          var treeX=
            78+
            (
              treeIndex*
              53
            )%
            640;

          var treeY=
            281+
            Math.sin(
              treeIndex*
              1.7
            )*
            22;

          drawTree(
            context,
            treeX,
            treeY,
            0.62+
            treeIndex%3*
            0.10,
            health
          );
        }

        var rainCount=Math.round(
          values.precipitation/
          12
        );

        context.strokeStyle='#38BDF8';
        context.lineWidth=2;
        context.globalAlpha=
          0.25+
          values.precipitation/
          145;

        for(
          var rainIndex=0;
          rainIndex<rainCount;
          rainIndex+=1
        ){
          var rainX=
            90+
            rainIndex*
            78;

          context.beginPath();
          context.moveTo(
            rainX,
            105
          );
          context.lineTo(
            rainX-10,
            139
          );
          context.stroke();
        }

        context.globalAlpha=1;

        if(model.erosion>=55){
          var erosionCount=Math.round(
            model.erosion/
            15
          );

          context.strokeStyle='#DC2626';
          context.lineWidth=2.4;

          for(
            var erosionIndex=0;
            erosionIndex<erosionCount;
            erosionIndex+=1
          ){
            var erosionX=
              222+
              erosionIndex*
              29;

            var erosionY=
              242+
              erosionIndex*
              5;

            context.beginPath();
            context.moveTo(
              erosionX,
              erosionY
            );
            context.lineTo(
              erosionX+17,
              erosionY+22
            );
            context.stroke();
          }
        }

        if(labelSwitch.checked){
          fillRoundedRect(
            context,
            78,
            91,
            105,
            27,
            13,
            'rgba(255,255,255,0.90)',
            '#BAE6FD'
          );

          drawText(
            context,
            '气候：热量与降水',
            130,
            105,
            9,
            '#0369A1',
            800,
            'center'
          );

          fillRoundedRect(
            context,
            595,
            147,
            108,
            27,
            13,
            'rgba(255,255,255,0.90)',
            '#A7D8BC'
          );

          drawText(
            context,
            '植被截留与蒸腾',
            649,
            161,
            9,
            '#15803D',
            800,
            'center'
          );

          fillRoundedRect(
            context,
            510,
            294,
            112,
            27,
            13,
            'rgba(255,255,255,0.90)',
            '#BAE6FD'
          );

          drawText(
            context,
            '径流与物质迁移',
            566,
            308,
            9,
            '#0369A1',
            800,
            'center'
          );

          fillRoundedRect(
            context,
            246,
            304,
            102,
            27,
            13,
            'rgba(255,255,255,0.90)',
            '#FDE68A'
          );

          drawText(
            context,
            '土壤形成与侵蚀',
            297,
            318,
            9,
            '#92400E',
            800,
            'center'
          );
        }
      }

      function drawResponse(
        context,
        values,
        model
      ){
        var metrics=[
          {
            label:'蒸发需求',
            value:model.evaporation,
            color:'#EA580C',
            direction:
              values.temperature>=20
                ? '增强'
                : '较弱'
          },
          {
            label:'土壤水分',
            value:model.soilWater,
            color:'#0284C7',
            direction:
              model.soilWater>=55
                ? '充足'
                : model.soilWater>=32
                  ? '中等'
                  : '不足'
          },
          {
            label:'植被潜力',
            value:
              model.vegetationPotential,
            color:'#16A34A',
            direction:
              model.vegetationPotential>=65
                ? '较高'
                : model.vegetationPotential>=38
                  ? '中等'
                  : '较低'
          },
          {
            label:'地表径流',
            value:model.runoff,
            color:'#0369A1',
            direction:
              model.runoff>=65
                ? '较强'
                : model.runoff>=35
                  ? '中等'
                  : '较弱'
          },
          {
            label:'侵蚀压力',
            value:model.erosion,
            color:'#DC2626',
            direction:
              model.erosion>=65
                ? '较高'
                : model.erosion>=35
                  ? '中等'
                  : '较低'
          },
          {
            label:'整体稳定性',
            value:model.stability,
            color:'#7C3AED',
            direction:
              model.stability>=68
                ? '较高'
                : model.stability>=40
                  ? '中等'
                  : '较低'
          }
        ];

        fillRoundedRect(
          context,
          51,
          78,
          717,
          277,
          17,
          '#F8FAFC',
          '#A7D8BC'
        );

        metrics.forEach(
          function(item,index){
            var column=index%3;
            var row=Math.floor(
              index/3
            );

            var x=
              72+
              column*
              232;

            var y=
              103+
              row*
              121;

            fillRoundedRect(
              context,
              x,
              y,
              205,
              95,
              13,
              '#FFFFFF',
              item.color
            );

            drawText(
              context,
              item.label,
              x+17,
              y+20,
              11,
              item.color,
              850,
              'left'
            );

            drawText(
              context,
              item.direction,
              x+185,
              y+20,
              9.5,
              item.color,
              800,
              'right'
            );

            drawText(
              context,
              Math.round(
                item.value
              )+
              '%',
              x+103,
              y+49,
              22,
              item.color,
              900,
              'center'
            );

            context.fillStyle='#E2E8F0';

            context.fillRect(
              x+18,
              y+72,
              169,
              9
            );

            context.fillStyle=item.color;

            context.fillRect(
              x+18,
              y+72,
              clamp(
                item.value/
                100*
                169,
                3,
                169
              ),
              9
            );
          }
        );

        if(labelSwitch.checked){
          drawText(
            context,
            '输入变化',
            86,
            374,
            10,
            '#475569',
            800,
            'center'
          );

          drawText(
            context,
            '→',
            138,
            374,
            16,
            '#64748B',
            900,
            'center'
          );

          drawText(
            context,
            '水热条件调整',
            224,
            374,
            10,
            '#0369A1',
            800,
            'center'
          );

          drawText(
            context,
            '→',
            310,
            374,
            16,
            '#64748B',
            900,
            'center'
          );

          drawText(
            context,
            '土壤与植被响应',
            413,
            374,
            10,
            '#15803D',
            800,
            'center'
          );

          drawText(
            context,
            '→',
            520,
            374,
            16,
            '#64748B',
            900,
            'center'
          );

          drawText(
            context,
            '径流与地貌变化',
            638,
            374,
            10,
            '#B45309',
            800,
            'center'
          );
        }
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

        temperatureValue.textContent=
          Math.round(
            values.temperature
          )+
          '℃';

        precipitationValue.textContent=
          Math.round(
            values.precipitation
          )+
          '%';

        slopeValue.textContent=
          Math.round(
            values.slope
          )+
          '%';

        vegetationValue.textContent=
          Math.round(
            values.vegetation
          )+
          '%';

        disturbanceValue.textContent=
          Math.round(
            values.disturbance
          )+
          '%';

        waterValue.textContent=
          Math.round(
            model.soilWater
          )+
          '%';

        runoffValue.textContent=
          Math.round(
            model.runoff
          )+
          '%';

        erosionValue.textContent=
          Math.round(
            model.erosion
          )+
          '%';

        stabilityValue.textContent=
          Math.round(
            model.stability
          )+
          '%';

        result.textContent=
          model.dominantChange+
          '。'+
          describe(
            values,
            model
          )+
          ' 当前环境韧性关系指标为'+
          Math.round(
            model.resilience
          )+
          '%；该指标仅用于课堂比较。';

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
          '自然环境整体性 · '+
          model.dominantChange,
          40,
          43,
          14,
          '#14532D',
          850,
          'left'
        );

        drawText(
          context,
          '各要素响应存在强弱差异和时间滞后',
          780,
          43,
          9.5,
          '#64748B',
          650,
          'right'
        );

        if(
          values.observationMode===
          'landscape'
        ){
          drawLandscape(
            context,
            values,
            model
          );
        }else if(
          values.observationMode===
          'response'
        ){
          drawResponse(
            context,
            values,
            model
          );
        }else{
          drawNetwork(
            context,
            values,
            model
          );
        }

        drawText(
          context,
          '自然环境要素通过能量交换和物质迁移相互联系，本图不用于真实生态评价或规划决策。',
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
          forest:{
            temperature:18,
            precipitation:72,
            slope:38,
            vegetation:92,
            disturbance:8,
            observationMode:'network'
          },
          drought:{
            temperature:29,
            precipitation:18,
            slope:42,
            vegetation:38,
            disturbance:20,
            observationMode:'response'
          },
          warming:{
            temperature:33,
            precipitation:52,
            slope:42,
            vegetation:58,
            disturbance:24,
            observationMode:'response'
          },
          deforestation:{
            temperature:23,
            precipitation:68,
            slope:65,
            vegetation:12,
            disturbance:86,
            observationMode:'landscape'
          },
          humid:{
            temperature:21,
            precipitation:94,
            slope:36,
            vegetation:78,
            disturbance:15,
            observationMode:'landscape'
          },
          recovery:{
            temperature:19,
            precipitation:66,
            slope:48,
            vegetation:82,
            disturbance:12,
            observationMode:'network'
          }
        };

        var scenario=scenarios[name];

        if(!scenario){
          return;
        }

        temperatureInput.value=String(
          scenario.temperature
        );

        precipitationInput.value=String(
          scenario.precipitation
        );

        slopeInput.value=String(
          scenario.slope
        );

        vegetationInput.value=String(
          scenario.vegetation
        );

        disturbanceInput.value=String(
          scenario.disturbance
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
          3600%
          1;

        render();

        state.raf=
          requestAnimationFrame(
            animate
          );
      }

      var temperatureInput=query(
        '[data-role="temperature"]'
      );

      var precipitationInput=query(
        '[data-role="precipitation"]'
      );

      var slopeInput=query(
        '[data-role="slope"]'
      );

      var vegetationInput=query(
        '[data-role="vegetation"]'
      );

      var disturbanceInput=query(
        '[data-role="disturbance"]'
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

      var temperatureValue=query(
        '[data-role="temperature-value"]'
      );

      var precipitationValue=query(
        '[data-role="precipitation-value"]'
      );

      var slopeValue=query(
        '[data-role="slope-value"]'
      );

      var vegetationValue=query(
        '[data-role="vegetation-value"]'
      );

      var disturbanceValue=query(
        '[data-role="disturbance-value"]'
      );

      var waterValue=query(
        '[data-role="water-value"]'
      );

      var runoffValue=query(
        '[data-role="runoff-value"]'
      );

      var erosionValue=query(
        '[data-role="erosion-value"]'
      );

      var stabilityValue=query(
        '[data-role="stability-value"]'
      );

      if(
        !temperatureInput ||
        !precipitationInput ||
        !slopeInput ||
        !vegetationInput ||
        !disturbanceInput ||
        !observationSelect ||
        !labelSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !result ||
        !canvas ||
        !temperatureValue ||
        !precipitationValue ||
        !slopeValue ||
        !vegetationValue ||
        !disturbanceValue ||
        !waterValue ||
        !runoffValue ||
        !erosionValue ||
        !stabilityValue
      ){
        return;
      }

      var scenarioOrder=[
        'forest',
        'drought',
        'warming',
        'deforestation',
        'humid',
        'recovery'
      ];

      var state={
        phase:0,
        startedAt:0,
        raf:0,
        timer:null,
        scenarioIndex:-1
      };

      [
        temperatureInput,
        precipitationInput,
        slopeInput,
        vegetationInput,
        disturbanceInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            clearScenarioSelection
          );
        }
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
                'forest';

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

export const GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_HOLISM:
GeographyLabTemplate[] = [
  {
    id: 'geography-natural-environment-elements-holism',
    group: '🌍 自然环境整体性与地域分异',
    name: '自然环境要素相互作用与整体性',
    emoji: '🌍',
    desc: '调节气温、降水、坡度、植被和人类扰动，观察气候、水文、地貌、土壤与植被之间的连锁响应。',
    params: [
      {
        key: 'temperature',
        label: '年平均气温',
        type: 'number',
        min: -5,
        max: 35,
        step: 1,
        defaultValue: 18,
        hint: '气温影响蒸发、土壤水分和植被生长条件。',
      },
      {
        key: 'precipitation',
        label: '降水条件',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
        hint: '相对教学量，降水影响径流、下渗、土壤和植被。',
      },
      {
        key: 'slope',
        label: '地形坡度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
        hint: '坡度增大通常会加快汇流并增强侵蚀条件。',
      },
      {
        key: 'vegetation',
        label: '植被覆盖',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
        hint: '植被可截留降水、稳定土壤并影响蒸腾和下渗。',
      },
      {
        key: 'humanDisturbance',
        label: '人类扰动强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 22,
        hint: '表示地表破坏、硬化或资源利用的综合相对强度。',
      },
      {
        key: 'observationMode',
        label: '初始观察模式',
        type: 'select',
        options: [
          {
            label: '要素关系网络',
            value: 'network',
          },
          {
            label: '景观剖面响应',
            value: 'landscape',
          },
          {
            label: '连锁变化比较',
            value: 'response',
          },
        ],
        defaultValue: 'network',
      },
      {
        key: 'showLabels',
        label: '显示要素与过程标注',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'automatic',
        label: '自动演示典型情境',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildEnvironmentHolismHTML,
  },
]
