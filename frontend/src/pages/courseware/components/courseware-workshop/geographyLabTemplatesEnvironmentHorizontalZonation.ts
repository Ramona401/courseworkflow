/**
 * geographyLabTemplatesEnvironmentHorizontalZonation.ts
 *
 * 第37批B2：纬度地带性与从沿海到内陆的水平地域分异。
 *
 * 教学目标：
 * 1. 理解太阳辐射随纬度变化是纬度地带性的重要基础；
 * 2. 观察从赤道向两极热量条件和自然带的有规律变化；
 * 3. 理解海陆位置和水汽输送对沿海—内陆分异的影响；
 * 4. 比较同一纬度沿海、内陆地区的降水、温差和植被差异；
 * 5. 理解自然带由热量和水分条件共同决定；
 * 6. 比较热带雨林、热带草原、荒漠、温带森林、
 *    温带草原、寒带苔原等典型自然带；
 * 7. 认识自然带边界具有过渡性，并非严格平直和固定不变；
 * 8. 理解水平地域分异是地带性规律与非地带性因素共同作用的结果。
 *
 * 教学边界：
 * - 所有温度、降水、季节差异和自然带范围均为相对教学量；
 * - 图中海陆分布和自然带位置不对应真实世界地图；
 * - 不考虑洋流、山脉屏障、地形抬升、季风和局地环流等全部因素；
 * - 不用于气候预测、农业区划、植被恢复、土地规划或旅行决策；
 * - 真实自然带边界具有过渡性、区域差异和历史变化。
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

function buildHorizontalZonationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const latitude = Math.max(
    -80,
    Math.min(
      80,
      numberValue(params, 'latitude', 35),
    ),
  )

  const distanceFromOcean = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'distanceFromOcean', 35),
    ),
  )

  const oceanMoisture = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'oceanMoisture', 72),
    ),
  )

  const temperatureAnomaly = Math.max(
    -5,
    Math.min(
      5,
      numberValue(params, 'temperatureAnomaly', 0),
    ),
  )

  const seasonality = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'seasonality', 48),
    ),
  )

  const requestedMode = stringValue(
    params,
    'observationMode',
    'latitude',
  )

  const observationMode = [
    'latitude',
    'continental',
    'comparison',
  ].includes(requestedMode)
    ? requestedMode
    : 'latitude'

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
<div id="${rootId}" class="gl-horizontal-zonation-root">
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

    #${rootId} .gl-zonation-canvas{
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
      🗺️
    </div>

    <div>
      <div class="gl-title">
        纬度地带性与从沿海到内陆的水平地域分异
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        调节纬度、距海距离、水汽输送和季节差异，观察自然带水平变化
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 不对应真实世界地图
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            所在纬度
          </span>

          <span
            class="gl-value"
            data-role="latitude-value"
          ></span>
        </div>

        <input
          type="range"
          min="-80"
          max="80"
          step="1"
          value="${shortNumber(latitude)}"
          data-role="latitude"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            距海距离
          </span>

          <span
            class="gl-value"
            data-role="distance-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(distanceFromOcean)}"
          data-role="distance"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            海洋水汽输送
          </span>

          <span
            class="gl-value"
            data-role="moisture-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(oceanMoisture)}"
          data-role="moisture"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            温度距平
          </span>

          <span
            class="gl-value"
            data-role="anomaly-value"
          ></span>
        </div>

        <input
          type="range"
          min="-5"
          max="5"
          step="1"
          value="${shortNumber(temperatureAnomaly)}"
          data-role="anomaly"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            季节差异强度
          </span>

          <span
            class="gl-value"
            data-role="seasonality-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(seasonality)}"
          data-role="seasonality"
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
            value="latitude"
            ${observationMode === 'latitude' ? 'selected' : ''}
          >
            从赤道向两极
          </option>

          <option
            value="continental"
            ${observationMode === 'continental' ? 'selected' : ''}
          >
            从沿海向内陆
          </option>

          <option
            value="comparison"
            ${observationMode === 'comparison' ? 'selected' : ''}
          >
            水热条件比较
          </option>
        </select>
      </div>

      <div class="gl-switch-row">
        <span>显示自然带与过程标注</span>

        <input
          type="checkbox"
          data-role="label-switch"
          ${showLabels ? 'checked' : ''}
        />
      </div>

      <div class="gl-switch-row">
        <span>自动演示典型区域</span>

        <input
          type="checkbox"
          data-role="auto-switch"
          ${automatic ? 'checked' : ''}
        />
      </div>

      <div class="gl-subtitle">
        典型水平分异情境
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="rainforest"
        >
          🌴 赤道雨林
        </button>

        <button
          type="button"
          data-scenario="desert"
        >
          🏜️ 副热带荒漠
        </button>

        <button
          type="button"
          data-scenario="coastal-forest"
        >
          🌲 温带沿海林
        </button>

        <button
          type="button"
          data-scenario="inland-grassland"
        >
          🌾 温带内陆草原
        </button>

        <button
          type="button"
          data-scenario="inland-desert"
        >
          🐫 温带内陆荒漠
        </button>

        <button
          type="button"
          data-scenario="tundra"
        >
          ❄️ 高纬苔原
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-zonation-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="水平地域分异规律教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="belt-value"></strong>
          <span>纬度热量带</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="temperature-output"></strong>
          <span>年平均气温示意</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="precipitation-output"></strong>
          <span>降水条件示意</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="zone-value"></strong>
          <span>可能自然带</span>
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

      function gaussian(
        value,
        center,
        spread
      ){
        var distance=
          (
            value-center
          )/
          spread;

        return Math.exp(
          -0.5*
          distance*
          distance
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
          10
        );
      }

      function drawTree(
        context,
        x,
        y,
        scale,
        type
      ){
        context.save();

        context.fillStyle='#854D0E';

        context.fillRect(
          x-3*scale,
          y-24*scale,
          6*scale,
          27*scale
        );

        if(type==='conifer'){
          context.fillStyle='#166534';

          context.beginPath();
          context.moveTo(
            x,
            y-57*scale
          );
          context.lineTo(
            x-17*scale,
            y-22*scale
          );
          context.lineTo(
            x+17*scale,
            y-22*scale
          );
          context.closePath();
          context.fill();

          context.beginPath();
          context.moveTo(
            x,
            y-46*scale
          );
          context.lineTo(
            x-20*scale,
            y-13*scale
          );
          context.lineTo(
            x+20*scale,
            y-13*scale
          );
          context.closePath();
          context.fill();
        }else if(
          type==='rainforest'
        ){
          context.fillStyle='#15803D';

          context.beginPath();
          context.arc(
            x,
            y-39*scale,
            20*scale,
            0,
            Math.PI*2
          );
          context.fill();

          context.fillStyle='#22C55E';

          context.beginPath();
          context.arc(
            x-13*scale,
            y-30*scale,
            14*scale,
            0,
            Math.PI*2
          );

          context.arc(
            x+14*scale,
            y-30*scale,
            14*scale,
            0,
            Math.PI*2
          );
          context.fill();
        }else{
          context.fillStyle='#22C55E';

          context.beginPath();
          context.arc(
            x,
            y-35*scale,
            17*scale,
            0,
            Math.PI*2
          );
          context.fill();

          context.fillStyle='#16A34A';

          context.beginPath();
          context.arc(
            x-11*scale,
            y-28*scale,
            11*scale,
            0,
            Math.PI*2
          );

          context.arc(
            x+11*scale,
            y-28*scale,
            11*scale,
            0,
            Math.PI*2
          );
          context.fill();
        }

        context.restore();
      }

      function readState(){
        return {
          latitude:Number(
            latitudeInput.value
          ),
          distance:Number(
            distanceInput.value
          ),
          moisture:Number(
            moistureInput.value
          ),
          anomaly:Number(
            anomalyInput.value
          ),
          seasonality:Number(
            seasonalityInput.value
          ),
          observationMode:
            observationSelect.value
        };
      }

      function calculate(values){
        var absoluteLatitude=
          Math.abs(
            values.latitude
          );

        var solarRadiation=clamp(
          100-
          absoluteLatitude*
          1.04,
          8,
          100
        );

        var temperature=
          29-
          absoluteLatitude*
          0.47+
          values.anomaly-
          values.distance*
          0.025;

        var equatorialWet=
          gaussian(
            absoluteLatitude,
            2,
            17
          );

        var subtropicalDry=
          gaussian(
            absoluteLatitude,
            28,
            11
          );

        var midLatitudeWet=
          gaussian(
            absoluteLatitude,
            48,
            19
          );

        var polarDry=
          gaussian(
            absoluteLatitude,
            76,
            16
          );

        var latitudeMoisture=
          30+
          equatorialWet*
          64-
          subtropicalDry*
          39+
          midLatitudeWet*
          25-
          polarDry*
          18;

        var marineInfluence=
          (
            100-values.distance
          )/
          100;

        var precipitation=clamp(
          latitudeMoisture*
          0.58+
          values.moisture*
          0.48*
          marineInfluence-
          values.distance*
          0.24,
          2,
          100
        );

        var annualRange=clamp(
          5+
          absoluteLatitude*
          0.15+
          values.distance*
          0.31+
          values.seasonality*
          0.28-
          values.moisture*
          marineInfluence*
          0.08,
          3,
          58
        );

        var continentality=clamp(
          values.distance*
          0.66+
          values.seasonality*
          0.34,
          0,
          100
        );

        var belt;

        if(absoluteLatitude<10){
          belt='赤道带';
        }else if(
          absoluteLatitude<23.5
        ){
          belt='热带';
        }else if(
          absoluteLatitude<35
        ){
          belt='副热带';
        }else if(
          absoluteLatitude<55
        ){
          belt='温带';
        }else if(
          absoluteLatitude<66.5
        ){
          belt='亚寒带';
        }else{
          belt='寒带';
        }

        var naturalZone;

        if(
          absoluteLatitude>=74 ||
          temperature<-8
        ){
          naturalZone='冰原带';
        }else if(
          absoluteLatitude>=64 ||
          temperature<1
        ){
          naturalZone=
            precipitation>=28
              ? '苔原带'
              : '寒带荒漠';
        }else if(
          temperature<7
        ){
          naturalZone=
            precipitation>=45
              ? '亚寒带针叶林'
              : precipitation>=24
                ? '寒温带草原'
                : '寒温带荒漠';
        }else if(
          temperature<16
        ){
          naturalZone=
            precipitation>=62
              ? '温带森林'
              : precipitation>=32
                ? '温带草原'
                : '温带荒漠';
        }else if(
          temperature<24
        ){
          naturalZone=
            precipitation>=68
              ? '亚热带森林'
              : precipitation>=38
                ? '亚热带草原灌丛'
                : '亚热带荒漠';
        }else{
          naturalZone=
            precipitation>=72
              ? '热带雨林'
              : precipitation>=40
                ? '热带草原'
                : '热带荒漠';
        }

        var waterCondition=
          precipitation>=68
            ? '湿润'
            : precipitation>=42
              ? '半湿润'
              : precipitation>=22
                ? '半干旱'
                : '干旱';

        var heatCondition=
          temperature>=24
            ? '高温'
            : temperature>=14
              ? '温暖'
              : temperature>=3
                ? '凉爽'
                : '寒冷';

        var dominantFactor=
          values.distance>=68
            ? '海陆位置影响突出'
            : absoluteLatitude>=58 ||
              absoluteLatitude<=18
              ? '纬度热量差异突出'
              : precipitation<=28
                ? '水分限制突出'
                : '水热共同作用';

        return {
          absoluteLatitude:
            absoluteLatitude,
          solarRadiation:
            solarRadiation,
          temperature:
            temperature,
          precipitation:
            precipitation,
          annualRange:
            annualRange,
          continentality:
            continentality,
          belt:belt,
          naturalZone:
            naturalZone,
          waterCondition:
            waterCondition,
          heatCondition:
            heatCondition,
          dominantFactor:
            dominantFactor,
          marineInfluence:
            marineInfluence
        };
      }

      function latitudeText(value){
        if(Math.abs(value)<0.5){
          return '0° 赤道';
        }

        return Math.abs(
          Math.round(value)
        )+
        '°'+
        (
          value>0
            ? 'N'
            : 'S'
        );
      }

      function describe(values,model){
        if(
          model.absoluteLatitude<=12 &&
          model.precipitation>=68
        ){
          return '低纬地区太阳辐射较强、全年高温；充足水汽有利于形成高温多雨环境，'+
            '典型自然带可能表现为热带雨林。';
        }

        if(
          model.absoluteLatitude>=22 &&
          model.absoluteLatitude<=35 &&
          model.precipitation<=32
        ){
          return '副热带部分地区下沉气流和水分不足较明显，'+
            '在距海较远或水汽输送较弱时，更容易形成荒漠或草原景观。';
        }

        if(
          values.distance>=68 &&
          model.precipitation<=40
        ){
          return '同一纬度由沿海向内陆，海洋水汽影响逐渐减弱，降水减少、年温差增大，'+
            '自然带可能由森林向草原、荒漠过渡。';
        }

        if(
          model.absoluteLatitude>=65
        ){
          return '高纬地区太阳高度较低，获得的热量较少，生长期较短，'+
            '自然带通常表现为针叶林、苔原或冰原。';
        }

        if(
          values.distance<=25 &&
          values.moisture>=65
        ){
          return '沿海地区受海洋水汽和热容量影响较明显，降水条件相对较好、年温差较小，'+
            '同纬度自然带通常比内陆更湿润。';
        }

        return '纬度主要影响热量条件，海陆位置和水汽输送主要影响水分及季节差异。'+
          '自然带是热量与水分共同作用的结果，边界通常具有过渡性。';
      }

      function zoneColor(zone){
        if(
          zone.indexOf('雨林')>=0
        ){
          return '#15803D';
        }

        if(
          zone.indexOf('森林')>=0 ||
          zone.indexOf('针叶林')>=0
        ){
          return '#16A34A';
        }

        if(
          zone.indexOf('草原')>=0 ||
          zone.indexOf('灌丛')>=0
        ){
          return '#84CC16';
        }

        if(
          zone.indexOf('荒漠')>=0
        ){
          return '#D97706';
        }

        if(
          zone.indexOf('苔原')>=0
        ){
          return '#94A3B8';
        }

        if(
          zone.indexOf('冰原')>=0
        ){
          return '#E0F2FE';
        }

        return '#0EA5E9';
      }

      function drawLatitudeMode(
        context,
        values,
        model
      ){
        var chart={
          x:57,
          y:82,
          width:704,
          height:278
        };

        fillRoundedRect(
          context,
          chart.x,
          chart.y,
          chart.width,
          chart.height,
          17,
          '#F8FAFC',
          '#A7D8BC'
        );

        var belts=[
          {
            start:0,
            end:10,
            label:'赤道带',
            zone:'热带雨林',
            color:'#15803D'
          },
          {
            start:10,
            end:23.5,
            label:'热带',
            zone:'雨林—草原',
            color:'#22C55E'
          },
          {
            start:23.5,
            end:35,
            label:'副热带',
            zone:'森林—荒漠',
            color:'#F59E0B'
          },
          {
            start:35,
            end:55,
            label:'温带',
            zone:'森林—草原—荒漠',
            color:'#84CC16'
          },
          {
            start:55,
            end:66.5,
            label:'亚寒带',
            zone:'针叶林',
            color:'#166534'
          },
          {
            start:66.5,
            end:90,
            label:'寒带',
            zone:'苔原—冰原',
            color:'#CBD5E1'
          }
        ];

        var stripY=132;
        var stripHeight=82;

        belts.forEach(
          function(belt){
            var x=
              chart.x+
              22+
              belt.start/
              90*
              (
                chart.width-44
              );

            var width=
              (
                belt.end-
                belt.start
              )/
              90*
              (
                chart.width-44
              );

            context.fillStyle=belt.color;

            context.globalAlpha=0.78;

            context.fillRect(
              x,
              stripY,
              width,
              stripHeight
            );

            context.globalAlpha=1;

            if(
              labelSwitch.checked &&
              width>=58
            ){
              drawText(
                context,
                belt.label,
                x+width/2,
                stripY+24,
                9.5,
                '#0F172A',
                850,
                'center'
              );

              drawText(
                context,
                belt.zone,
                x+width/2,
                stripY+57,
                8,
                '#334155',
                700,
                'center'
              );
            }
          }
        );

        context.strokeStyle='#475569';
        context.lineWidth=1.5;

        context.strokeRect(
          chart.x+22,
          stripY,
          chart.width-44,
          stripHeight
        );

        for(
          var latitudeTick=0;
          latitudeTick<=90;
          latitudeTick+=10
        ){
          var tickX=
            chart.x+
            22+
            latitudeTick/
            90*
            (
              chart.width-44
            );

          context.strokeStyle='#94A3B8';
          context.lineWidth=1;

          context.beginPath();
          context.moveTo(
            tickX,
            stripY+
            stripHeight
          );
          context.lineTo(
            tickX,
            stripY+
            stripHeight+
            8
          );
          context.stroke();

          drawText(
            context,
            latitudeTick+'°',
            tickX,
            stripY+
            stripHeight+
            22,
            8.5,
            '#64748B',
            700,
            'center'
          );
        }

        var markerX=
          chart.x+
          22+
          model.absoluteLatitude/
          90*
          (
            chart.width-44
          );

        context.fillStyle='#DC2626';

        context.beginPath();
        context.moveTo(
          markerX,
          stripY-17
        );
        context.lineTo(
          markerX-8,
          stripY-4
        );
        context.lineTo(
          markerX+8,
          stripY-4
        );
        context.closePath();
        context.fill();

        context.strokeStyle='#DC2626';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(
          markerX,
          stripY-4
        );
        context.lineTo(
          markerX,
          stripY+
          stripHeight
        );
        context.stroke();

        drawText(
          context,
          latitudeText(
            values.latitude
          ),
          markerX,
          stripY-29,
          10,
          '#B91C1C',
          850,
          'center'
        );

        var curveLeft=
          chart.x+50;

        var curveRight=
          chart.x+
          chart.width-
          34;

        var curveTop=266;
        var curveHeight=58;

        context.strokeStyle='#F97316';
        context.lineWidth=3;

        context.beginPath();

        for(
          var heatIndex=0;
          heatIndex<=90;
          heatIndex+=2
        ){
          var heatX=
            curveLeft+
            heatIndex/
            90*
            (
              curveRight-
              curveLeft
            );

          var heatValue=clamp(
            100-
            heatIndex*
            1.04,
            8,
            100
          );

          var heatY=
            curveTop+
            curveHeight-
            heatValue/
            100*
            curveHeight;

          if(heatIndex===0){
            context.moveTo(
              heatX,
              heatY
            );
          }else{
            context.lineTo(
              heatX,
              heatY
            );
          }
        }

        context.stroke();

        context.strokeStyle='#0284C7';
        context.lineWidth=3;

        context.beginPath();

        for(
          var waterIndex=0;
          waterIndex<=90;
          waterIndex+=2
        ){
          var waterX=
            curveLeft+
            waterIndex/
            90*
            (
              curveRight-
              curveLeft
            );

          var waterValue=
            30+
            gaussian(
              waterIndex,
              2,
              17
            )*
            64-
            gaussian(
              waterIndex,
              28,
              11
            )*
            39+
            gaussian(
              waterIndex,
              48,
              19
            )*
            25-
            gaussian(
              waterIndex,
              76,
              16
            )*
            18;

          var waterY=
            curveTop+
            curveHeight-
            clamp(
              waterValue,
              0,
              100
            )/
            100*
            curveHeight;

          if(waterIndex===0){
            context.moveTo(
              waterX,
              waterY
            );
          }else{
            context.lineTo(
              waterX,
              waterY
            );
          }
        }

        context.stroke();

        if(labelSwitch.checked){
          drawText(
            context,
            '热量总体由低纬向高纬减少',
            232,
            341,
            9,
            '#C2410C',
            800,
            'center'
          );

          drawText(
            context,
            '水分随纬度变化并非单调递减',
            576,
            341,
            9,
            '#0369A1',
            800,
            'center'
          );
        }
      }

      function drawContinentalMode(
        context,
        values,
        model
      ){
        var area={
          x:53,
          y:80,
          width:714,
          height:279
        };

        var sky=
          context.createLinearGradient(
            0,
            area.y,
            0,
            area.y+
            area.height
          );

        sky.addColorStop(
          0,
          '#BAE6FD'
        );

        sky.addColorStop(
          0.60,
          '#F8FAFC'
        );

        sky.addColorStop(
          1,
          '#FEF3C7'
        );

        fillRoundedRect(
          context,
          area.x,
          area.y,
          area.width,
          area.height,
          17,
          sky,
          '#A7D8BC'
        );

        context.fillStyle='#0EA5E9';

        context.fillRect(
          area.x,
          area.y+181,
          104,
          98
        );

        context.strokeStyle=
          'rgba(255,255,255,0.70)';
        context.lineWidth=1.5;

        for(
          var waveIndex=0;
          waveIndex<5;
          waveIndex+=1
        ){
          var waveY=
            area.y+
            198+
            waveIndex*
            15;

          context.beginPath();
          context.moveTo(
            area.x+10,
            waveY
          );
          context.bezierCurveTo(
            area.x+35,
            waveY-4,
            area.x+70,
            waveY+4,
            area.x+96,
            waveY
          );
          context.stroke();
        }

        context.fillStyle='#D6B377';

        context.beginPath();
        context.moveTo(
          area.x+103,
          area.y+216
        );

        context.bezierCurveTo(
          area.x+245,
          area.y+186,
          area.x+416,
          area.y+205,
          area.x+
          area.width,
          area.y+177
        );

        context.lineTo(
          area.x+
          area.width,
          area.y+
          area.height
        );

        context.lineTo(
          area.x+103,
          area.y+
          area.height
        );

        context.closePath();
        context.fill();

        var moistureReach=
          110+
          values.moisture*
          4.7;

        var moistureEndX=clamp(
          area.x+
          moistureReach-
          values.distance*
          1.5,
          area.x+125,
          area.x+
          area.width-
          25
        );

        context.strokeStyle='#0284C7';
        context.lineWidth=
          2+
          values.moisture/
          25;
        context.globalAlpha=
          0.35+
          values.moisture/
          155;
        context.setLineDash([11,7]);
        context.lineDashOffset=
          -state.phase*
          40;

        context.beginPath();
        context.moveTo(
          area.x+80,
          area.y+130
        );
        context.lineTo(
          moistureEndX,
          area.y+130
        );
        context.stroke();

        context.setLineDash([]);
        context.globalAlpha=1;

        drawArrowHead(
          context,
          moistureEndX,
          area.y+130,
          0,
          '#0284C7',
          11
        );

        var cloudCount=Math.round(
          values.moisture/
          18
        );

        for(
          var cloudIndex=0;
          cloudIndex<cloudCount;
          cloudIndex+=1
        ){
          var cloudX=
            area.x+
            110+
            cloudIndex*
            83;

          if(cloudX>moistureEndX){
            continue;
          }

          var cloudOpacity=
            clamp(
              1-
              (
                cloudX-
                area.x-
                100
              )/
              (
                area.width*
                0.95
              ),
              0.18,
              0.82
            );

          context.globalAlpha=
            cloudOpacity;

          context.fillStyle='#FFFFFF';

          context.beginPath();
          context.arc(
            cloudX,
            area.y+109,
            14,
            0,
            Math.PI*2
          );

          context.arc(
            cloudX+17,
            area.y+102,
            18,
            0,
            Math.PI*2
          );

          context.arc(
            cloudX+35,
            area.y+111,
            13,
            0,
            Math.PI*2
          );

          context.fill();

          context.strokeStyle='#38BDF8';
          context.lineWidth=1.8;

          var localRain=
            clamp(
              values.moisture-
              cloudIndex*
              14,
              0,
              100
            );

          var rainLines=Math.round(
            localRain/
            28
          );

          for(
            var rainIndex=0;
            rainIndex<rainLines;
            rainIndex+=1
          ){
            context.beginPath();
            context.moveTo(
              cloudX+
              7+
              rainIndex*
              10,
              area.y+127
            );

            context.lineTo(
              cloudX+
              rainIndex*
              10,
              area.y+150
            );
            context.stroke();
          }
        }

        context.globalAlpha=1;

        var zoneSections=[
          {
            start:0,
            end:28,
            label:'沿海森林',
            color:'#15803D',
            type:'forest'
          },
          {
            start:28,
            end:62,
            label:'内陆草原',
            color:'#84CC16',
            type:'grass'
          },
          {
            start:62,
            end:100,
            label:'内陆荒漠',
            color:'#D97706',
            type:'desert'
          }
        ];

        zoneSections.forEach(
          function(section){
            var sectionX=
              area.x+
              106+
              section.start/
              100*
              (
                area.width-
                116
              );

            var sectionWidth=
              (
                section.end-
                section.start
              )/
              100*
              (
                area.width-
                116
              );

            context.globalAlpha=0.35;
            context.fillStyle=
              section.color;

            context.fillRect(
              sectionX,
              area.y+216,
              sectionWidth,
              63
            );

            context.globalAlpha=1;

            if(
              section.type==='forest'
            ){
              for(
                var treeIndex=0;
                treeIndex<6;
                treeIndex+=1
              ){
                drawTree(
                  context,
                  sectionX+
                  23+
                  treeIndex*
                  29,
                  area.y+227,
                  0.55,
                  treeIndex%2===0
                    ? 'broadleaf'
                    : 'conifer'
                );
              }
            }else if(
              section.type==='grass'
            ){
              context.strokeStyle='#65A30D';
              context.lineWidth=2;

              for(
                var grassIndex=0;
                grassIndex<18;
                grassIndex+=1
              ){
                var grassX=
                  sectionX+
                  10+
                  grassIndex*
                  10;

                context.beginPath();
                context.moveTo(
                  grassX,
                  area.y+250
                );

                context.lineTo(
                  grassX+
                  Math.sin(
                    grassIndex
                  )*
                  3,
                  area.y+232-
                  grassIndex%3*
                  3
                );
                context.stroke();
              }
            }else{
              context.fillStyle='#F59E0B';

              for(
                var duneIndex=0;
                duneIndex<4;
                duneIndex+=1
              ){
                context.beginPath();
                context.ellipse(
                  sectionX+
                  35+
                  duneIndex*
                  54,
                  area.y+
                  244+
                  duneIndex%2*
                  8,
                  34,
                  10,
                  0,
                  0,
                  Math.PI*2
                );
                context.fill();
              }
            }

            if(labelSwitch.checked){
              drawText(
                context,
                section.label,
                sectionX+
                sectionWidth/
                2,
                area.y+266,
                9,
                '#334155',
                850,
                'center'
              );
            }
          }
        );

        var markerX=
          area.x+
          106+
          values.distance/
          100*
          (
            area.width-
            116
          );

        context.strokeStyle='#DC2626';
        context.lineWidth=2.2;

        context.beginPath();
        context.moveTo(
          markerX,
          area.y+158
        );
        context.lineTo(
          markerX,
          area.y+279
        );
        context.stroke();

        context.fillStyle='#DC2626';

        context.beginPath();
        context.moveTo(
          markerX,
          area.y+151
        );
        context.lineTo(
          markerX-8,
          area.y+165
        );
        context.lineTo(
          markerX+8,
          area.y+165
        );
        context.closePath();
        context.fill();

        drawText(
          context,
          values.distance<=25
            ? '沿海'
            : values.distance<=65
              ? '内陆过渡区'
              : '大陆内部',
          markerX,
          area.y+143,
          9.5,
          '#B91C1C',
          850,
          'center'
        );

        if(labelSwitch.checked){
          drawText(
            context,
            '海洋水汽',
            area.x+55,
            area.y+109,
            9.5,
            '#0369A1',
            850,
            'center'
          );

          drawText(
            context,
            '由沿海向内陆水分总体减少、年温差增大',
            area.x+
            area.width/
            2+
            45,
            area.y+321,
            10,
            '#475569',
            800,
            'center'
          );
        }
      }

      function drawComparisonMode(
        context,
        values,
        model
      ){
        var cards=[
          {
            title:'热量条件',
            main:
              model.temperature.toFixed(1)+
              '℃',
            detail:
              model.heatCondition+
              ' · 辐射'+
              Math.round(
                model.solarRadiation
              )+
              '%',
            value:
              model.solarRadiation,
            color:'#EA580C'
          },
          {
            title:'水分条件',
            main:
              Math.round(
                model.precipitation
              )+
              '%',
            detail:
              model.waterCondition+
              ' · 海洋影响'+
              Math.round(
                model.marineInfluence*
                100
              )+
              '%',
            value:
              model.precipitation,
            color:'#0284C7'
          },
          {
            title:'大陆性',
            main:
              Math.round(
                model.annualRange
              )+
              '℃',
            detail:
              '年温差 · 大陆性'+
              Math.round(
                model.continentality
              )+
              '%',
            value:
              model.continentality,
            color:'#7C3AED'
          },
          {
            title:'自然带判断',
            main:
              model.naturalZone,
            detail:
              model.belt+
              ' · '+
              model.dominantFactor,
            value:
              (
                model.solarRadiation+
                model.precipitation
              )/
              2,
            color:
              zoneColor(
                model.naturalZone
              )
          }
        ];

        cards.forEach(
          function(card,index){
            var column=index%2;
            var row=Math.floor(
              index/2
            );

            var x=
              61+
              column*
              369;

            var y=
              82+
              row*
              142;

            fillRoundedRect(
              context,
              x,
              y,
              330,
              118,
              16,
              '#FFFFFF',
              card.color
            );

            drawText(
              context,
              card.title,
              x+19,
              y+23,
              11,
              card.color,
              850,
              'left'
            );

            drawText(
              context,
              card.main,
              x+165,
              y+54,
              card.main.length>=8
                ? 17
                : 25,
              card.color,
              900,
              'center'
            );

            drawText(
              context,
              card.detail,
              x+165,
              y+82,
              9.5,
              '#64748B',
              750,
              'center'
            );

            context.fillStyle='#E2E8F0';

            context.fillRect(
              x+25,
              y+98,
              280,
              9
            );

            context.fillStyle=card.color;

            context.fillRect(
              x+25,
              y+98,
              clamp(
                card.value/
                100*
                280,
                3,
                280
              ),
              9
            );
          }
        );

        if(labelSwitch.checked){
          drawText(
            context,
            '纬度主要影响热量',
            182,
            371,
            10,
            '#EA580C',
            800,
            'center'
          );

          drawText(
            context,
            '+',
            303,
            371,
            16,
            '#64748B',
            900,
            'center'
          );

          drawText(
            context,
            '海陆位置影响水分',
            424,
            371,
            10,
            '#0284C7',
            800,
            'center'
          );

          drawText(
            context,
            '=',
            546,
            371,
            16,
            '#64748B',
            900,
            'center'
          );

          drawText(
            context,
            '自然带水平分异',
            664,
            371,
            10,
            '#15803D',
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

        latitudeValue.textContent=
          latitudeText(
            values.latitude
          );

        distanceValue.textContent=
          Math.round(
            values.distance
          )+
          '%';

        moistureValue.textContent=
          Math.round(
            values.moisture
          )+
          '%';

        anomalyValue.textContent=
          (
            values.anomaly>0
              ? '+'
              : ''
          )+
          Math.round(
            values.anomaly
          )+
          '℃';

        seasonalityValue.textContent=
          Math.round(
            values.seasonality
          )+
          '%';

        beltValue.textContent=
          model.belt;

        temperatureOutput.textContent=
          model.temperature.toFixed(1)+
          '℃';

        precipitationOutput.textContent=
          model.waterCondition+
          ' '+
          Math.round(
            model.precipitation
          )+
          '%';

        zoneValue.textContent=
          model.naturalZone;

        result.textContent=
          latitudeText(
            values.latitude
          )+
          '、距海程度'+
          Math.round(
            values.distance
          )+
          '%条件下，可能形成'+
          model.naturalZone+
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
          model.belt+
          ' · '+
          model.naturalZone,
          40,
          43,
          14,
          '#14532D',
          850,
          'left'
        );

        drawText(
          context,
          '自然带边界具有过渡性，不是固定直线',
          780,
          43,
          9.5,
          '#64748B',
          650,
          'right'
        );

        if(
          values.observationMode===
          'continental'
        ){
          drawContinentalMode(
            context,
            values,
            model
          );
        }else if(
          values.observationMode===
          'comparison'
        ){
          drawComparisonMode(
            context,
            values,
            model
          );
        }else{
          drawLatitudeMode(
            context,
            values,
            model
          );
        }

        drawText(
          context,
          '纬度影响热量，海陆位置影响水分和季节差异；本图不用于真实气候或土地规划。',
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
          rainforest:{
            latitude:2,
            distance:12,
            moisture:96,
            anomaly:1,
            seasonality:12,
            observationMode:'latitude'
          },
          desert:{
            latitude:28,
            distance:76,
            moisture:24,
            anomaly:2,
            seasonality:48,
            observationMode:'latitude'
          },
          'coastal-forest':{
            latitude:45,
            distance:10,
            moisture:86,
            anomaly:0,
            seasonality:38,
            observationMode:'continental'
          },
          'inland-grassland':{
            latitude:45,
            distance:62,
            moisture:54,
            anomaly:0,
            seasonality:72,
            observationMode:'continental'
          },
          'inland-desert':{
            latitude:42,
            distance:94,
            moisture:20,
            anomaly:1,
            seasonality:88,
            observationMode:'continental'
          },
          tundra:{
            latitude:72,
            distance:38,
            moisture:48,
            anomaly:-2,
            seasonality:82,
            observationMode:'comparison'
          }
        };

        var scenario=scenarios[name];

        if(!scenario){
          return;
        }

        latitudeInput.value=String(
          scenario.latitude
        );

        distanceInput.value=String(
          scenario.distance
        );

        moistureInput.value=String(
          scenario.moisture
        );

        anomalyInput.value=String(
          scenario.anomaly
        );

        seasonalityInput.value=String(
          scenario.seasonality
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

      var latitudeInput=query(
        '[data-role="latitude"]'
      );

      var distanceInput=query(
        '[data-role="distance"]'
      );

      var moistureInput=query(
        '[data-role="moisture"]'
      );

      var anomalyInput=query(
        '[data-role="anomaly"]'
      );

      var seasonalityInput=query(
        '[data-role="seasonality"]'
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

      var latitudeValue=query(
        '[data-role="latitude-value"]'
      );

      var distanceValue=query(
        '[data-role="distance-value"]'
      );

      var moistureValue=query(
        '[data-role="moisture-value"]'
      );

      var anomalyValue=query(
        '[data-role="anomaly-value"]'
      );

      var seasonalityValue=query(
        '[data-role="seasonality-value"]'
      );

      var beltValue=query(
        '[data-role="belt-value"]'
      );

      var temperatureOutput=query(
        '[data-role="temperature-output"]'
      );

      var precipitationOutput=query(
        '[data-role="precipitation-output"]'
      );

      var zoneValue=query(
        '[data-role="zone-value"]'
      );

      if(
        !latitudeInput ||
        !distanceInput ||
        !moistureInput ||
        !anomalyInput ||
        !seasonalityInput ||
        !observationSelect ||
        !labelSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !result ||
        !canvas ||
        !latitudeValue ||
        !distanceValue ||
        !moistureValue ||
        !anomalyValue ||
        !seasonalityValue ||
        !beltValue ||
        !temperatureOutput ||
        !precipitationOutput ||
        !zoneValue
      ){
        return;
      }

      var scenarioOrder=[
        'rainforest',
        'desert',
        'coastal-forest',
        'inland-grassland',
        'inland-desert',
        'tundra'
      ];

      var state={
        phase:0,
        startedAt:0,
        raf:0,
        timer:null,
        scenarioIndex:-1
      };

      [
        latitudeInput,
        distanceInput,
        moistureInput,
        anomalyInput,
        seasonalityInput
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
                'rainforest';

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

export const GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_HORIZONTAL_ZONATION:
GeographyLabTemplate[] = [
  {
    id: 'geography-horizontal-zonation-latitude-coast-inland',
    group: '🌍 自然环境整体性与地域分异',
    name: '纬度地带性与从沿海到内陆的水平地域分异',
    emoji: '🗺️',
    desc: '调节纬度、距海距离、水汽输送、温度距平和季节差异，观察热量带、干湿变化与自然带水平分异。',
    params: [
      {
        key: 'latitude',
        label: '所在纬度',
        type: 'number',
        min: -80,
        max: 80,
        step: 1,
        defaultValue: 35,
        hint: '纬度主要影响太阳辐射和热量条件。',
      },
      {
        key: 'distanceFromOcean',
        label: '距海距离',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 35,
        hint: '相对教学量，数值越大表示越接近大陆内部。',
      },
      {
        key: 'oceanMoisture',
        label: '海洋水汽输送',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
        hint: '水汽输送增强通常有利于沿海及下风向地区降水。',
      },
      {
        key: 'temperatureAnomaly',
        label: '温度距平',
        type: 'number',
        min: -5,
        max: 5,
        step: 1,
        defaultValue: 0,
        hint: '用于比较偏暖或偏冷情境，不代表真实气候预测。',
      },
      {
        key: 'seasonality',
        label: '季节差异强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 48,
        hint: '季节差异与纬度、海陆位置等因素有关。',
      },
      {
        key: 'observationMode',
        label: '初始观察模式',
        type: 'select',
        options: [
          {
            label: '从赤道向两极',
            value: 'latitude',
          },
          {
            label: '从沿海向内陆',
            value: 'continental',
          },
          {
            label: '水热条件比较',
            value: 'comparison',
          },
        ],
        defaultValue: 'latitude',
      },
      {
        key: 'showLabels',
        label: '显示自然带与过程标注',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'automatic',
        label: '自动演示典型区域',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildHorizontalZonationHTML,
  },
]
