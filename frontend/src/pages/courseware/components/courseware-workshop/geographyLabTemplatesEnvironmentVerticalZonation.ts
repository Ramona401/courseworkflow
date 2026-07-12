/**
 * geographyLabTemplatesEnvironmentVerticalZonation.ts
 *
 * 第37批B3：山地垂直地域分异、林线与雪线。
 *
 * 教学目标：
 * 1. 理解气温随海拔升高总体降低，是垂直地域分异的重要基础；
 * 2. 观察山麓到山顶热量、水分、植被和土壤条件的垂直变化；
 * 3. 比较低纬、高纬，湿润、干旱山地的垂直带谱差异；
 * 4. 理解纬度越低、山体越高，可能形成的垂直自然带通常越丰富；
 * 5. 理解迎风坡与背风坡水分条件不同，垂直带谱和林线可能不同；
 * 6. 理解阳坡、阴坡接受太阳辐射不同，热量条件和带谱高度可能不同；
 * 7. 认识林线和雪线受温度、降水、坡向、风力等多因素共同影响；
 * 8. 理解山地垂直分异与从赤道向两极的水平地域分异具有相似性，
 *    但两者并不完全等同。
 *
 * 教学边界：
 * - 海拔、温度递减率、林线、雪线和带谱高度均为课堂示意；
 * - 图中山体和自然带不对应任何真实山脉、自然保护区或登山线路；
 * - 不考虑坡度变化、局地逆温、风速、积雪再分配和岩性等全部因素；
 * - 不用于登山路线、雪崩判断、生态调查、农业区划或工程决策；
 * - 真实林线和雪线具有区域差异、坡向差异、季节变化和长期波动。
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

function buildVerticalZonationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const latitude = Math.max(
    -60,
    Math.min(
      60,
      numberValue(params, 'latitude', 28),
    ),
  )

  const mountainHeight = Math.max(
    1000,
    Math.min(
      7000,
      numberValue(params, 'mountainHeight', 5200),
    ),
  )

  const baseTemperature = Math.max(
    -5,
    Math.min(
      32,
      numberValue(params, 'baseTemperature', 22),
    ),
  )

  const precipitation = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'precipitation', 68),
    ),
  )

  const requestedMoistureSide = stringValue(
    params,
    'moistureSide',
    'windward',
  )

  const moistureSide = [
    'windward',
    'leeward',
  ].includes(requestedMoistureSide)
    ? requestedMoistureSide
    : 'windward'

  const requestedAspect = stringValue(
    params,
    'slopeAspect',
    'shady',
  )

  const slopeAspect = [
    'sunny',
    'shady',
  ].includes(requestedAspect)
    ? requestedAspect
    : 'shady'

  const requestedMode = stringValue(
    params,
    'observationMode',
    'belts',
  )

  const observationMode = [
    'belts',
    'profile',
    'comparison',
  ].includes(requestedMode)
    ? requestedMode
    : 'belts'

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
<div id="${rootId}" class="gl-vertical-zonation-root">
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
        #F8FAFC
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

    #${rootId} .gl-vertical-canvas{
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
      🏔️
    </div>

    <div>
      <div class="gl-title">
        山地垂直地域分异、林线与雪线
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        调节纬度、山高、山麓气温、水分、坡向，观察垂直自然带谱变化
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 不用于真实登山与生态判断
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            山地所在纬度
          </span>

          <span
            class="gl-value"
            data-role="latitude-value"
          ></span>
        </div>

        <input
          type="range"
          min="-60"
          max="60"
          step="1"
          value="${shortNumber(latitude)}"
          data-role="latitude"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            山体海拔高度
          </span>

          <span
            class="gl-value"
            data-role="height-value"
          ></span>
        </div>

        <input
          type="range"
          min="1000"
          max="7000"
          step="100"
          value="${shortNumber(mountainHeight)}"
          data-role="height"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            山麓气温
          </span>

          <span
            class="gl-value"
            data-role="temperature-value"
          ></span>
        </div>

        <input
          type="range"
          min="-5"
          max="32"
          step="1"
          value="${shortNumber(baseTemperature)}"
          data-role="temperature"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            区域水分条件
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
            水汽坡位
          </span>
        </div>

        <select data-role="moisture-side">
          <option
            value="windward"
            ${moistureSide === 'windward' ? 'selected' : ''}
          >
            迎风坡
          </option>

          <option
            value="leeward"
            ${moistureSide === 'leeward' ? 'selected' : ''}
          >
            背风坡
          </option>
        </select>
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            坡向热量条件
          </span>
        </div>

        <select data-role="slope-aspect">
          <option
            value="sunny"
            ${slopeAspect === 'sunny' ? 'selected' : ''}
          >
            阳坡
          </option>

          <option
            value="shady"
            ${slopeAspect === 'shady' ? 'selected' : ''}
          >
            阴坡
          </option>
        </select>
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            观察模式
          </span>
        </div>

        <select data-role="observation-mode">
          <option
            value="belts"
            ${observationMode === 'belts' ? 'selected' : ''}
          >
            垂直自然带谱
          </option>

          <option
            value="profile"
            ${observationMode === 'profile' ? 'selected' : ''}
          >
            温度降水剖面
          </option>

          <option
            value="comparison"
            ${observationMode === 'comparison' ? 'selected' : ''}
          >
            林线雪线比较
          </option>
        </select>
      </div>

      <div class="gl-switch-row">
        <span>显示自然带、林线和雪线</span>

        <input
          type="checkbox"
          data-role="label-switch"
          ${showLabels ? 'checked' : ''}
        />
      </div>

      <div class="gl-switch-row">
        <span>自动演示典型山地</span>

        <input
          type="checkbox"
          data-role="auto-switch"
          ${automatic ? 'checked' : ''}
        />
      </div>

      <div class="gl-subtitle">
        典型垂直分异情境
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="tropical"
        >
          🌴 低纬高山
        </button>

        <button
          type="button"
          data-scenario="temperate"
        >
          🌲 温带湿润山地
        </button>

        <button
          type="button"
          data-scenario="continental"
        >
          🏜️ 内陆干旱山地
        </button>

        <button
          type="button"
          data-scenario="high-latitude"
        >
          ❄️ 高纬山地
        </button>

        <button
          type="button"
          data-scenario="windward"
        >
          🌧️ 迎风坡
        </button>

        <button
          type="button"
          data-scenario="leeward"
        >
          ☀️ 背风坡
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-vertical-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="山地垂直地域分异林线与雪线教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="belt-count-value"></strong>
          <span>垂直自然带数量</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="treeline-value"></strong>
          <span>林线高度示意</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="snowline-value"></strong>
          <span>雪线高度示意</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="summit-value"></strong>
          <span>山顶环境</span>
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

      function latitudeText(value){
        if(Math.abs(value)<0.5){
          return '0° 赤道附近';
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

      function readState(){
        return {
          latitude:Number(
            latitudeInput.value
          ),
          height:Number(
            heightInput.value
          ),
          baseTemperature:Number(
            temperatureInput.value
          ),
          precipitation:Number(
            precipitationInput.value
          ),
          moistureSide:
            moistureSideSelect.value,
          aspect:
            aspectSelect.value,
          observationMode:
            observationSelect.value
        };
      }

      function zoneAt(
        altitude,
        climate
      ){
        var temperature=
          climate.surfaceTemperature-
          altitude/
          100*
          climate.lapseRate;

        var moisture=
          climate.effectiveMoisture+
          22*
          Math.exp(
            -0.5*
            Math.pow(
              (
                altitude-
                climate.height*
                0.38
              )/
              Math.max(
                600,
                climate.height*
                0.22
              ),
              2
            )
          )-
          altitude/
          climate.height*
          17;

        moisture=clamp(
          moisture,
          0,
          100
        );

        if(
          altitude>=
          climate.snowline
        ){
          return {
            key:'snow',
            label:'高山冰雪带',
            color:'#E0F2FE',
            temperature:temperature,
            moisture:moisture
          };
        }

        if(
          altitude>=
          climate.treeline
        ){
          if(
            temperature<=2
          ){
            return {
              key:'tundra',
              label:'高山苔原带',
              color:'#94A3B8',
              temperature:temperature,
              moisture:moisture
            };
          }

          return {
            key:'meadow',
            label:'高山草甸带',
            color:'#A3E635',
            temperature:temperature,
            moisture:moisture
          };
        }

        if(temperature<7){
          return moisture>=38
            ? {
                key:'conifer',
                label:'山地针叶林带',
                color:'#166534',
                temperature:temperature,
                moisture:moisture
              }
            : {
                key:'alpine-steppe',
                label:'高寒草原带',
                color:'#84CC16',
                temperature:temperature,
                moisture:moisture
              };
        }

        if(temperature<15){
          return moisture>=55
            ? {
                key:'mixed',
                label:'山地针阔混交林带',
                color:'#15803D',
                temperature:temperature,
                moisture:moisture
              }
            : moisture>=28
              ? {
                  key:'steppe',
                  label:'山地草原带',
                  color:'#65A30D',
                  temperature:temperature,
                  moisture:moisture
                }
              : {
                  key:'dry-steppe',
                  label:'山地荒漠草原带',
                  color:'#CA8A04',
                  temperature:temperature,
                  moisture:moisture
                };
        }

        if(temperature<23){
          return moisture>=60
            ? {
                key:'broadleaf',
                label:'山地落叶阔叶林带',
                color:'#22C55E',
                temperature:temperature,
                moisture:moisture
              }
            : moisture>=32
              ? {
                  key:'warm-steppe',
                  label:'山地草原灌丛带',
                  color:'#84CC16',
                  temperature:temperature,
                  moisture:moisture
                }
              : {
                  key:'desert',
                  label:'山地荒漠带',
                  color:'#D97706',
                  temperature:temperature,
                  moisture:moisture
                };
        }

        return moisture>=70
          ? {
              key:'evergreen',
              label:'常绿阔叶林或雨林带',
              color:'#16A34A',
              temperature:temperature,
              moisture:moisture
            }
          : moisture>=38
            ? {
                key:'savanna',
                label:'山地稀树草原带',
                color:'#A3E635',
                temperature:temperature,
                moisture:moisture
              }
            : {
                key:'hot-desert',
                label:'山麓荒漠带',
                color:'#F59E0B',
                temperature:temperature,
                moisture:moisture
              };
      }

      function calculate(values){
        var absoluteLatitude=
          Math.abs(
            values.latitude
          );

        var latitudeCooling=
          absoluteLatitude*
          0.065;

        var aspectAdjustment=
          values.aspect==='sunny'
            ? 1.8
            : -1.1;

        var surfaceTemperature=
          values.baseTemperature-
          latitudeCooling+
          aspectAdjustment;

        var lapseRate=
          values.aspect==='sunny'
            ? 0.68
            : 0.59;

        var moistureAdjustment=
          values.moistureSide===
          'windward'
            ? 18
            : -22;

        var effectiveMoisture=clamp(
          values.precipitation+
          moistureAdjustment,
          0,
          100
        );

        var rawTreeline=
          (
            surfaceTemperature-
            6
          )/
          lapseRate*
          100;

        var treeline=clamp(
          rawTreeline+
          (
            effectiveMoisture-
            50
          )*
          4-
          absoluteLatitude*
          7,
          0,
          values.height
        );

        var rawSnowline=
          surfaceTemperature/
          lapseRate*
          100;

        var snowline=clamp(
          rawSnowline-
          (
            effectiveMoisture-
            50
          )*
          5-
          absoluteLatitude*
          9,
          0,
          values.height+
          900
        );

        if(snowline<treeline+350){
          snowline=clamp(
            treeline+350,
            0,
            values.height+900
          );
        }

        var climate={
          height:values.height,
          surfaceTemperature:
            surfaceTemperature,
          lapseRate:lapseRate,
          effectiveMoisture:
            effectiveMoisture,
          treeline:treeline,
          snowline:snowline
        };

        var samples=[];
        var sampleStep=100;

        for(
          var altitude=0;
          altitude<=values.height;
          altitude+=sampleStep
        ){
          var zone=zoneAt(
            altitude,
            climate
          );

          samples.push({
            altitude:altitude,
            key:zone.key,
            label:zone.label,
            color:zone.color,
            temperature:
              zone.temperature,
            moisture:
              zone.moisture
          });
        }

        if(
          samples[
            samples.length-1
          ].altitude<
          values.height
        ){
          var summitZone=zoneAt(
            values.height,
            climate
          );

          samples.push({
            altitude:values.height,
            key:summitZone.key,
            label:summitZone.label,
            color:summitZone.color,
            temperature:
              summitZone.temperature,
            moisture:
              summitZone.moisture
          });
        }

        var bands=[];

        samples.forEach(
          function(sample){
            var last=
              bands[
                bands.length-1
              ];

            if(
              !last ||
              last.key!==sample.key
            ){
              bands.push({
                key:sample.key,
                label:sample.label,
                color:sample.color,
                minAltitude:
                  sample.altitude,
                maxAltitude:
                  sample.altitude
              });
            }else{
              last.maxAltitude=
                sample.altitude;
            }
          }
        );

        bands.forEach(
          function(band,index){
            if(index<
              bands.length-1
            ){
              band.maxAltitude=
                bands[
                  index+1
                ].minAltitude;
            }else{
              band.maxAltitude=
                values.height;
            }
          }
        );

        var summitTemperature=
          surfaceTemperature-
          values.height/
          100*
          lapseRate;

        var summitZone=zoneAt(
          values.height,
          climate
        );

        var beltRichness=clamp(
          bands.length/
          7*
          100,
          0,
          100
        );

        var orographicEffect=
          values.moistureSide===
          'windward'
            ? clamp(
                35+
                values.precipitation*
                0.58,
                0,
                100
              )
            : clamp(
                70-
                values.precipitation*
                0.35,
                0,
                100
              );

        var lineDifference=
          clamp(
            snowline-
            treeline,
            0,
            3000
          );

        return {
          absoluteLatitude:
            absoluteLatitude,
          surfaceTemperature:
            surfaceTemperature,
          lapseRate:lapseRate,
          effectiveMoisture:
            effectiveMoisture,
          treeline:treeline,
          snowline:snowline,
          summitTemperature:
            summitTemperature,
          summitZone:
            summitZone,
          bands:bands,
          samples:samples,
          beltRichness:
            beltRichness,
          orographicEffect:
            orographicEffect,
          lineDifference:
            lineDifference
        };
      }

      function describe(values,model){
        if(
          model.absoluteLatitude<=18 &&
          values.height>=5000
        ){
          return '低纬高山山麓热量充足，山体又具有较大的海拔高差，'+
            '从山麓到山顶可能依次出现森林、草甸、苔原和冰雪等多个垂直自然带。';
        }

        if(
          model.absoluteLatitude>=48
        ){
          return '高纬地区山麓本身热量较少，林线和雪线通常相对较低，'+
            '山地垂直带谱可能较简单，高山冰雪或苔原更容易接近山麓。';
        }

        if(
          values.moistureSide===
          'windward'
        ){
          return '迎风坡空气受地形抬升，水汽较易凝结，水分条件通常优于背风坡；'+
            '森林带可能更完整，雪线也可能因积雪较多而相对降低。';
        }

        if(
          values.moistureSide===
          'leeward'
        ){
          return '背风坡气流下沉增温，水分条件相对较差，可能出现雨影效应；'+
            '森林带缩窄，草原或荒漠带扩大，雪线可能相对升高。';
        }

        if(
          values.aspect==='sunny'
        ){
          return '阳坡接受太阳辐射相对较多，坡面温度较高，'+
            '同一山地的自然带界线、林线和雪线可能高于阴坡。';
        }

        return '阴坡接受太阳辐射相对较少，坡面较凉、蒸发较弱；'+
          '在水分条件相近时，林线和雪线可能低于阳坡。';
      }

      function altitudeToY(
        altitude,
        height,
        chart
      ){
        return chart.y+
          chart.height-
          altitude/
          height*
          chart.height;
      }

      function drawTree(
        context,
        x,
        y,
        scale,
        conifer
      ){
        context.save();

        context.fillStyle='#854D0E';

        context.fillRect(
          x-2.5*scale,
          y-19*scale,
          5*scale,
          22*scale
        );

        if(conifer){
          context.fillStyle='#166534';

          context.beginPath();
          context.moveTo(
            x,
            y-48*scale
          );
          context.lineTo(
            x-15*scale,
            y-17*scale
          );
          context.lineTo(
            x+15*scale,
            y-17*scale
          );
          context.closePath();
          context.fill();

          context.beginPath();
          context.moveTo(
            x,
            y-38*scale
          );
          context.lineTo(
            x-18*scale,
            y-8*scale
          );
          context.lineTo(
            x+18*scale,
            y-8*scale
          );
          context.closePath();
          context.fill();
        }else{
          context.fillStyle='#22C55E';

          context.beginPath();
          context.arc(
            x,
            y-30*scale,
            15*scale,
            0,
            Math.PI*2
          );

          context.arc(
            x-10*scale,
            y-22*scale,
            10*scale,
            0,
            Math.PI*2
          );

          context.arc(
            x+10*scale,
            y-22*scale,
            10*scale,
            0,
            Math.PI*2
          );

          context.fill();
        }

        context.restore();
      }

      function drawMountainBelts(
        context,
        values,
        model
      ){
        var area={
          x:50,
          y:72,
          width:720,
          height:292
        };

        fillRoundedRect(
          context,
          area.x,
          area.y,
          area.width,
          area.height,
          17,
          '#E0F2FE',
          '#A7D8BC'
        );

        var baseY=
          area.y+
          area.height-
          24;

        var summitY=
          area.y+
          28;

        var centerX=
          area.x+
          area.width*
          0.51;

        var halfWidth=
          280;

        model.bands.forEach(
          function(band){
            var lowerY=
              altitudeToY(
                band.minAltitude,
                values.height,
                {
                  y:summitY,
                  height:
                    baseY-
                    summitY
                }
              );

            var upperY=
              altitudeToY(
                band.maxAltitude,
                values.height,
                {
                  y:summitY,
                  height:
                    baseY-
                    summitY
                }
              );

            var lowerRatio=
              band.minAltitude/
              values.height;

            var upperRatio=
              band.maxAltitude/
              values.height;

            var lowerHalfWidth=
              halfWidth*
              (
                1-
                lowerRatio*
                0.87
              );

            var upperHalfWidth=
              halfWidth*
              (
                1-
                upperRatio*
                0.87
              );

            context.fillStyle=
              band.color;

            context.beginPath();
            context.moveTo(
              centerX-
              lowerHalfWidth,
              lowerY
            );
            context.lineTo(
              centerX+
              lowerHalfWidth,
              lowerY
            );
            context.lineTo(
              centerX+
              upperHalfWidth,
              upperY
            );
            context.lineTo(
              centerX-
              upperHalfWidth,
              upperY
            );
            context.closePath();
            context.fill();

            context.strokeStyle=
              'rgba(255,255,255,0.72)';
            context.lineWidth=1.3;
            context.stroke();

            var bandHeight=
              lowerY-
              upperY;

            if(
              labelSwitch.checked &&
              bandHeight>=22
            ){
              drawText(
                context,
                band.label,
                centerX,
                (
                  lowerY+
                  upperY
                )/
                2,
                bandHeight>=38
                  ? 10
                  : 8.5,
                band.key==='snow'
                  ? '#334155'
                  : '#0F172A',
                850,
                'center'
              );
            }
          }
        );

        context.strokeStyle='#475569';
        context.lineWidth=2.2;

        context.beginPath();
        context.moveTo(
          centerX-halfWidth,
          baseY
        );
        context.lineTo(
          centerX,
          summitY
        );
        context.lineTo(
          centerX+halfWidth,
          baseY
        );
        context.stroke();

        var treelineY=
          altitudeToY(
            clamp(
              model.treeline,
              0,
              values.height
            ),
            values.height,
            {
              y:summitY,
              height:
                baseY-
                summitY
            }
          );

        var snowlineY=
          altitudeToY(
            clamp(
              model.snowline,
              0,
              values.height
            ),
            values.height,
            {
              y:summitY,
              height:
                baseY-
                summitY
            }
          );

        context.save();
        context.setLineDash([8,5]);

        context.strokeStyle='#166534';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(
          centerX-
          halfWidth-
          20,
          treelineY
        );
        context.lineTo(
          centerX+
          halfWidth+
          20,
          treelineY
        );
        context.stroke();

        context.strokeStyle='#2563EB';

        context.beginPath();
        context.moveTo(
          centerX-
          halfWidth-
          20,
          snowlineY
        );
        context.lineTo(
          centerX+
          halfWidth+
          20,
          snowlineY
        );
        context.stroke();

        context.restore();

        if(labelSwitch.checked){
          fillRoundedRect(
            context,
            area.x+12,
            treelineY-13,
            92,
            25,
            12,
            'rgba(220,252,231,0.94)',
            '#86EFAC'
          );

          drawText(
            context,
            '林线 '+Math.round(
              model.treeline
            )+'m',
            area.x+58,
            treelineY,
            9,
            '#166534',
            850,
            'center'
          );

          fillRoundedRect(
            context,
            area.x+
            area.width-
            104,
            snowlineY-13,
            92,
            25,
            12,
            'rgba(219,234,254,0.94)',
            '#93C5FD'
          );

          drawText(
            context,
            '雪线 '+Math.round(
              model.snowline
            )+'m',
            area.x+
            area.width-
            58,
            snowlineY,
            9,
            '#1D4ED8',
            850,
            'center'
          );
        }

        var treeCount=clamp(
          Math.round(
            model.effectiveMoisture/
            8
          ),
          3,
          14
        );

        for(
          var treeIndex=0;
          treeIndex<treeCount;
          treeIndex+=1
        ){
          var treeAltitude=
            clamp(
              model.treeline*
              (
                0.25+
                (
                  treeIndex%
                  5
                )*
                0.13
              ),
              120,
              model.treeline-
              80
            );

          var treeY=
            altitudeToY(
              treeAltitude,
              values.height,
              {
                y:summitY,
                height:
                  baseY-
                  summitY
              }
            );

          var treeRatio=
            treeAltitude/
            values.height;

          var slopeHalf=
            halfWidth*
            (
              1-
              treeRatio*
              0.87
            );

          var side=
            treeIndex%2===0
              ? -1
              : 1;

          var treeX=
            centerX+
            side*
            (
              slopeHalf*
              (
                0.42+
                (
                  treeIndex%
                  3
                )*
                0.17
              )
            );

          drawTree(
            context,
            treeX,
            treeY,
            0.44+
            treeIndex%3*
            0.08,
            treeAltitude>
            model.treeline*
            0.62
          );
        }

        var cloudCount=
          values.moistureSide===
          'windward'
            ? 5
            : 2;

        context.globalAlpha=
          0.45+
          model.effectiveMoisture/
          190;

        context.fillStyle='#FFFFFF';

        for(
          var cloudIndex=0;
          cloudIndex<cloudCount;
          cloudIndex+=1
        ){
          var cloudX=
            values.moistureSide===
            'windward'
              ? area.x+
                55+
                cloudIndex*
                58
              : area.x+
                area.width-
                85-
                cloudIndex*
                58;

          var cloudY=
            area.y+
            67+
            cloudIndex%
            2*
            21;

          context.beginPath();
          context.arc(
            cloudX,
            cloudY,
            13,
            0,
            Math.PI*2
          );

          context.arc(
            cloudX+16,
            cloudY-6,
            17,
            0,
            Math.PI*2
          );

          context.arc(
            cloudX+33,
            cloudY+1,
            12,
            0,
            Math.PI*2
          );

          context.fill();
        }

        context.globalAlpha=1;

        context.strokeStyle='#0284C7';
        context.lineWidth=2.2;
        context.setLineDash([9,6]);
        context.lineDashOffset=
          -state.phase*
          40;

        var arrowStartX=
          values.moistureSide===
          'windward'
            ? area.x+48
            : area.x+
              area.width-
              48;

        var arrowEndX=
          values.moistureSide===
          'windward'
            ? centerX-80
            : centerX+80;

        context.beginPath();
        context.moveTo(
          arrowStartX,
          area.y+121
        );
        context.lineTo(
          arrowEndX,
          area.y+121
        );
        context.stroke();

        context.setLineDash([]);

        drawArrowHead(
          context,
          arrowEndX,
          area.y+121,
          values.moistureSide===
          'windward'
            ? 0
            : Math.PI,
          '#0284C7',
          10
        );

        if(labelSwitch.checked){
          drawText(
            context,
            values.moistureSide===
            'windward'
              ? '迎风坡水汽抬升'
              : '背风坡下沉增温',
            values.moistureSide===
            'windward'
              ? area.x+119
              : area.x+
                area.width-
                119,
            area.y+148,
            9,
            values.moistureSide===
            'windward'
              ? '#0369A1'
              : '#C2410C',
            800,
            'center'
          );

          drawText(
            context,
            values.aspect==='sunny'
              ? '阳坡热量较多'
              : '阴坡热量较少',
            centerX,
            baseY+17,
            9.5,
            values.aspect==='sunny'
              ? '#EA580C'
              : '#475569',
            800,
            'center'
          );
        }
      }

      function drawProfile(
        context,
        values,
        model
      ){
        var chart={
          x:61,
          y:81,
          width:509,
          height:278
        };

        fillRoundedRect(
          context,
          chart.x,
          chart.y,
          chart.width,
          chart.height,
          16,
          '#F8FAFC',
          '#A7D8BC'
        );

        context.strokeStyle='#CBD5E1';
        context.lineWidth=1;

        for(
          var gridIndex=0;
          gridIndex<=5;
          gridIndex+=1
        ){
          var gridY=
            chart.y+
            gridIndex/
            5*
            chart.height;

          context.beginPath();
          context.moveTo(
            chart.x,
            gridY
          );
          context.lineTo(
            chart.x+
            chart.width,
            gridY
          );
          context.stroke();

          var altitudeLabel=
            values.height*
            (
              1-
              gridIndex/
              5
            );

          drawText(
            context,
            Math.round(
              altitudeLabel
            )+
            'm',
            chart.x-8,
            gridY,
            8.5,
            '#64748B',
            700,
            'right'
          );
        }

        context.strokeStyle='#475569';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(
          chart.x,
          chart.y
        );
        context.lineTo(
          chart.x,
          chart.y+
          chart.height
        );
        context.lineTo(
          chart.x+
          chart.width,
          chart.y+
          chart.height
        );
        context.stroke();

        var temperatureMin=
          Math.min(
            -30,
            model.summitTemperature-
            5
          );

        var temperatureMax=
          Math.max(
            32,
            model.surfaceTemperature+
            4
          );

        context.strokeStyle='#EA580C';
        context.lineWidth=3;

        context.beginPath();

        model.samples.forEach(
          function(sample,index){
            var x=
              chart.x+
              (
                sample.temperature-
                temperatureMin
              )/
              (
                temperatureMax-
                temperatureMin
              )*
              chart.width;

            var y=
              altitudeToY(
                sample.altitude,
                values.height,
                chart
              );

            if(index===0){
              context.moveTo(x,y);
            }else{
              context.lineTo(x,y);
            }
          }
        );

        context.stroke();

        context.strokeStyle='#0284C7';
        context.lineWidth=3;

        context.beginPath();

        model.samples.forEach(
          function(sample,index){
            var x=
              chart.x+
              sample.moisture/
              100*
              chart.width;

            var y=
              altitudeToY(
                sample.altitude,
                values.height,
                chart
              );

            if(index===0){
              context.moveTo(x,y);
            }else{
              context.lineTo(x,y);
            }
          }
        );

        context.stroke();

        var treelineY=
          altitudeToY(
            clamp(
              model.treeline,
              0,
              values.height
            ),
            values.height,
            chart
          );

        var snowlineY=
          altitudeToY(
            clamp(
              model.snowline,
              0,
              values.height
            ),
            values.height,
            chart
          );

        context.save();
        context.setLineDash([7,5]);

        context.strokeStyle='#166534';

        context.beginPath();
        context.moveTo(
          chart.x,
          treelineY
        );
        context.lineTo(
          chart.x+
          chart.width,
          treelineY
        );
        context.stroke();

        context.strokeStyle='#2563EB';

        context.beginPath();
        context.moveTo(
          chart.x,
          snowlineY
        );
        context.lineTo(
          chart.x+
          chart.width,
          snowlineY
        );
        context.stroke();

        context.restore();

        fillRoundedRect(
          context,
          598,
          81,
          163,
          278,
          16,
          '#FFFFFF',
          '#A7D8BC'
        );

        drawText(
          context,
          '垂直变化',
          618,
          105,
          12,
          '#14532D',
          850,
          'left'
        );

        var rows=[
          {
            label:'山麓气温',
            value:
              model.surfaceTemperature.toFixed(
                1
              )+
              '℃',
            color:'#EA580C'
          },
          {
            label:'山顶气温',
            value:
              model.summitTemperature.toFixed(
                1
              )+
              '℃',
            color:'#C2410C'
          },
          {
            label:'有效水分',
            value:
              Math.round(
                model.effectiveMoisture
              )+
              '%',
            color:'#0284C7'
          },
          {
            label:'气温递减率',
            value:
              model.lapseRate.toFixed(
                2
              )+
              '℃/100m',
            color:'#7C3AED'
          },
          {
            label:'林线—雪线',
            value:
              Math.round(
                model.lineDifference
              )+
              'm',
            color:'#166534'
          }
        ];

        rows.forEach(
          function(row,index){
            var y=
              139+
              index*
              42;

            drawText(
              context,
              row.label,
              618,
              y,
              9.5,
              '#64748B',
              700,
              'left'
            );

            drawText(
              context,
              row.value,
              742,
              y,
              11,
              row.color,
              850,
              'right'
            );
          }
        );

        if(labelSwitch.checked){
          drawText(
            context,
            '橙线：气温',
            150,
            374,
            9,
            '#EA580C',
            800,
            'center'
          );

          drawText(
            context,
            '蓝线：水分条件',
            314,
            374,
            9,
            '#0284C7',
            800,
            'center'
          );

          drawText(
            context,
            '随海拔升高，气温总体降低',
            480,
            374,
            9,
            '#475569',
            800,
            'center'
          );
        }
      }

      function drawComparison(
        context,
        values,
        model
      ){
        var cards=[
          {
            title:'纬度影响',
            main:
              latitudeText(
                values.latitude
              ),
            detail:
              model.absoluteLatitude<=20
                ? '低纬：热量基础较高'
                : model.absoluteLatitude<=40
                  ? '中纬：季节差异明显'
                  : '高纬：热量基础较低',
            value:
              100-
              model.absoluteLatitude/
              60*
              100,
            color:'#EA580C'
          },
          {
            title:'山体高度',
            main:
              Math.round(
                values.height
              )+
              'm',
            detail:
              values.height>=5000
                ? '高差大，带谱较丰富'
                : values.height>=3000
                  ? '高差中等'
                  : '高差较小，带谱较少',
            value:
              (
                values.height-
                1000
              )/
              6000*
              100,
            color:'#7C3AED'
          },
          {
            title:'坡面水分',
            main:
              Math.round(
                model.effectiveMoisture
              )+
              '%',
            detail:
              values.moistureSide===
              'windward'
                ? '迎风坡：水分较好'
                : '背风坡：雨影较明显',
            value:
              model.effectiveMoisture,
            color:'#0284C7'
          },
          {
            title:'带谱丰富度',
            main:
              model.bands.length+
              '个带',
            detail:
              model.bands
                .map(
                  function(band){
                    return band.label;
                  }
                )
                .slice(
                  0,
                  3
                )
                .join('、'),
            value:
              model.beltRichness,
            color:'#16A34A'
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
              card.main.length>=10
                ? 17
                : 24,
              card.color,
              900,
              'center'
            );

            drawText(
              context,
              card.detail,
              x+165,
              y+81,
              card.detail.length>=16
                ? 8.5
                : 9.5,
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
            '低纬',
            119,
            371,
            10,
            '#EA580C',
            800,
            'center'
          );

          drawText(
            context,
            '+ 高山',
            225,
            371,
            10,
            '#7C3AED',
            800,
            'center'
          );

          drawText(
            context,
            '+ 湿润',
            347,
            371,
            10,
            '#0284C7',
            800,
            'center'
          );

          drawText(
            context,
            '→',
            444,
            371,
            16,
            '#64748B',
            900,
            'center'
          );

          drawText(
            context,
            '垂直带谱通常更丰富',
            600,
            371,
            10,
            '#15803D',
            850,
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

        heightValue.textContent=
          Math.round(
            values.height
          )+
          'm';

        temperatureValue.textContent=
          Math.round(
            values.baseTemperature
          )+
          '℃';

        precipitationValue.textContent=
          Math.round(
            values.precipitation
          )+
          '%';

        beltCountValue.textContent=
          model.bands.length+
          '个';

        treelineValue.textContent=
          Math.round(
            model.treeline
          )+
          'm';

        snowlineValue.textContent=
          model.snowline>
          values.height
            ? '高于山顶'
            : Math.round(
                model.snowline
              )+
              'm';

        summitValue.textContent=
          model.summitZone.label;

        result.textContent=
          latitudeText(
            values.latitude
          )+
          '、山高'+
          Math.round(
            values.height
          )+
          'm、'+
          (
            values.moistureSide===
            'windward'
              ? '迎风坡'
              : '背风坡'
          )+
          '、'+
          (
            values.aspect==='sunny'
              ? '阳坡'
              : '阴坡'
          )+
          '条件下，共形成约'+
          model.bands.length+
          '个垂直自然带。'+
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
          latitudeText(
            values.latitude
          )+
          ' · '+
          (
            values.moistureSide===
            'windward'
              ? '迎风坡'
              : '背风坡'
          )+
          ' · '+
          (
            values.aspect==='sunny'
              ? '阳坡'
              : '阴坡'
          ),
          40,
          43,
          14,
          '#14532D',
          850,
          'left'
        );

        drawText(
          context,
          '垂直带界、林线和雪线均为课堂关系示意',
          780,
          43,
          9.5,
          '#64748B',
          650,
          'right'
        );

        if(
          values.observationMode===
          'profile'
        ){
          drawProfile(
            context,
            values,
            model
          );
        }else if(
          values.observationMode===
          'comparison'
        ){
          drawComparison(
            context,
            values,
            model
          );
        }else{
          drawMountainBelts(
            context,
            values,
            model
          );
        }

        drawText(
          context,
          '山地垂直分异受纬度、山高、水分和坡向共同影响，本图不用于真实登山或生态调查。',
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
          tropical:{
            latitude:8,
            height:6500,
            baseTemperature:29,
            precipitation:88,
            moistureSide:'windward',
            aspect:'shady',
            observationMode:'belts'
          },
          temperate:{
            latitude:38,
            height:4600,
            baseTemperature:18,
            precipitation:82,
            moistureSide:'windward',
            aspect:'shady',
            observationMode:'belts'
          },
          continental:{
            latitude:42,
            height:5200,
            baseTemperature:17,
            precipitation:24,
            moistureSide:'leeward',
            aspect:'sunny',
            observationMode:'comparison'
          },
          'high-latitude':{
            latitude:58,
            height:3200,
            baseTemperature:8,
            precipitation:58,
            moistureSide:'windward',
            aspect:'shady',
            observationMode:'profile'
          },
          windward:{
            latitude:30,
            height:5000,
            baseTemperature:22,
            precipitation:76,
            moistureSide:'windward',
            aspect:'shady',
            observationMode:'belts'
          },
          leeward:{
            latitude:30,
            height:5000,
            baseTemperature:22,
            precipitation:76,
            moistureSide:'leeward',
            aspect:'sunny',
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

        heightInput.value=String(
          scenario.height
        );

        temperatureInput.value=String(
          scenario.baseTemperature
        );

        precipitationInput.value=String(
          scenario.precipitation
        );

        moistureSideSelect.value=
          scenario.moistureSide;

        aspectSelect.value=
          scenario.aspect;

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
          3200
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

      var latitudeInput=query(
        '[data-role="latitude"]'
      );

      var heightInput=query(
        '[data-role="height"]'
      );

      var temperatureInput=query(
        '[data-role="temperature"]'
      );

      var precipitationInput=query(
        '[data-role="precipitation"]'
      );

      var moistureSideSelect=query(
        '[data-role="moisture-side"]'
      );

      var aspectSelect=query(
        '[data-role="slope-aspect"]'
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

      var heightValue=query(
        '[data-role="height-value"]'
      );

      var temperatureValue=query(
        '[data-role="temperature-value"]'
      );

      var precipitationValue=query(
        '[data-role="precipitation-value"]'
      );

      var beltCountValue=query(
        '[data-role="belt-count-value"]'
      );

      var treelineValue=query(
        '[data-role="treeline-value"]'
      );

      var snowlineValue=query(
        '[data-role="snowline-value"]'
      );

      var summitValue=query(
        '[data-role="summit-value"]'
      );

      if(
        !latitudeInput ||
        !heightInput ||
        !temperatureInput ||
        !precipitationInput ||
        !moistureSideSelect ||
        !aspectSelect ||
        !observationSelect ||
        !labelSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !result ||
        !canvas ||
        !latitudeValue ||
        !heightValue ||
        !temperatureValue ||
        !precipitationValue ||
        !beltCountValue ||
        !treelineValue ||
        !snowlineValue ||
        !summitValue
      ){
        return;
      }

      var scenarioOrder=[
        'tropical',
        'temperate',
        'continental',
        'high-latitude',
        'windward',
        'leeward'
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
        heightInput,
        temperatureInput,
        precipitationInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            clearScenarioSelection
          );
        }
      );

      moistureSideSelect.addEventListener(
        'change',
        clearScenarioSelection
      );

      aspectSelect.addEventListener(
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
                'tropical';

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

export const GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_VERTICAL_ZONATION:
GeographyLabTemplate[] = [
  {
    id: 'geography-mountain-vertical-zonation-treeline-snowline',
    group: '🌍 自然环境整体性与地域分异',
    name: '山地垂直地域分异、林线与雪线',
    emoji: '🏔️',
    desc: '调节纬度、山高、山麓气温、水分、迎背风坡和坡向，观察垂直自然带谱、林线与雪线变化。',
    params: [
      {
        key: 'latitude',
        label: '山地所在纬度',
        type: 'number',
        min: -60,
        max: 60,
        step: 1,
        defaultValue: 28,
        hint: '纬度越高，山麓热量基础通常越低，林线和雪线可能越低。',
      },
      {
        key: 'mountainHeight',
        label: '山体海拔高度',
        type: 'number',
        min: 1000,
        max: 7000,
        step: 100,
        defaultValue: 5200,
        hint: '山体越高，能够容纳的垂直自然带通常越丰富。',
      },
      {
        key: 'baseTemperature',
        label: '山麓气温',
        type: 'number',
        min: -5,
        max: 32,
        step: 1,
        defaultValue: 22,
        hint: '山麓气温是垂直气温递减和带谱起点的教学基础。',
      },
      {
        key: 'precipitation',
        label: '区域水分条件',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
        hint: '水分影响森林发育、垂直带谱完整性和雪线位置。',
      },
      {
        key: 'moistureSide',
        label: '水汽坡位',
        type: 'select',
        options: [
          {
            label: '迎风坡',
            value: 'windward',
          },
          {
            label: '背风坡',
            value: 'leeward',
          },
        ],
        defaultValue: 'windward',
      },
      {
        key: 'slopeAspect',
        label: '坡向热量条件',
        type: 'select',
        options: [
          {
            label: '阳坡',
            value: 'sunny',
          },
          {
            label: '阴坡',
            value: 'shady',
          },
        ],
        defaultValue: 'shady',
      },
      {
        key: 'observationMode',
        label: '初始观察模式',
        type: 'select',
        options: [
          {
            label: '垂直自然带谱',
            value: 'belts',
          },
          {
            label: '温度降水剖面',
            value: 'profile',
          },
          {
            label: '林线雪线比较',
            value: 'comparison',
          },
        ],
        defaultValue: 'belts',
      },
      {
        key: 'showLabels',
        label: '显示自然带、林线和雪线',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'automatic',
        label: '自动演示典型山地',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildVerticalZonationHTML,
  },
]
