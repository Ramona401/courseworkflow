/**
 * geographyLabTemplatesHydrologyRiverRegime.ts
 *
 * 第35批B2：河流补给、流量过程线与汛期。
 *
 * 教学目标：
 * 1. 比较雨水、冰雪融水、地下水和湖泊调节等补给方式；
 * 2. 观察降水过程与河流流量过程线之间的滞后关系；
 * 3. 理解快速汇流、缓慢汇流和调蓄作用对洪峰的影响；
 * 4. 判读汛期、枯水期、峰现时间及主要补给来源。
 *
 * 教学边界：
 * - 降水量、流量和时间均为课堂关系比较使用的相对教学量；
 * - 不代表具体河流、流域或水文站的真实监测数据；
 * - 不考虑流域面积、河网密度、土壤含水率和真实水库调度；
 * - 不用于洪水预测、防灾预警、航运调度或水利工程决策。
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

function buildRiverRegimeHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const dayOfYear = Math.max(
    1,
    Math.min(
      365,
      numberValue(params, 'dayOfYear', 195),
    ),
  )

  const precipitation = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'precipitation', 65),
    ),
  )

  const temperature = Math.max(
    -10,
    Math.min(
      30,
      numberValue(params, 'temperature', 18),
    ),
  )

  const snowpack = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'snowpack', 35),
    ),
  )

  const groundwater = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'groundwater', 45),
    ),
  )

  const lakeRegulation = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'lakeRegulation', 35),
    ),
  )

  const requestedResponseMode = stringValue(
    params,
    'responseMode',
    'medium',
  )

  const responseMode = [
    'fast',
    'medium',
    'slow',
  ].includes(requestedResponseMode)
    ? requestedResponseMode
    : 'medium'

  const showComponents = booleanValue(
    params,
    'showComponents',
    true,
  )

  const automatic = booleanValue(
    params,
    'automatic',
    true,
  )

  return `
<div id="${rootId}" class="gl-river-regime-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border-radius:18px;
      border:1px solid #A7D8D2;
      background:#FFFFFF;
      color:#0F172A;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(15,118,110,0.10);
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
        #DBEAFE,
        #CCFBF1
      );
      border-bottom:1px solid #99F6E4;
    }

    #${rootId} .gl-title{
      color:#164E63;
      font-size:16px;
      font-weight:850;
    }

    #${rootId} .gl-note{
      margin-left:auto;
      color:#475569;
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
      border-right:1px solid #CCFBF1;
      background:linear-gradient(
        180deg,
        #F0FDFA,
        #EFF6FF
      );
    }

    #${rootId} .gl-stage{
      position:relative;
      min-width:0;
      min-height:0;
      padding:8px;
      background:radial-gradient(
        circle at 45% 18%,
        #FFFFFF 0%,
        #F8FAFC 56%,
        #E0F2FE 100%
      );
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
      font-size:11px;
      font-weight:750;
    }

    #${rootId} .gl-value{
      padding:3px 7px;
      border-radius:999px;
      background:#CCFBF1;
      color:#0F766E;
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
        #5EEAD4
      );
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:16px;
      height:16px;
      appearance:none;
      border-radius:50%;
      background:#0F766E;
      border:2px solid #FFFFFF;
      box-shadow:0 1px 5px rgba(14,116,144,0.42);
    }

    #${rootId} select{
      width:100%;
      min-height:34px;
      padding:6px 8px;
      border:1px solid #99F6E4;
      border-radius:9px;
      background:#FFFFFF;
      color:#0F766E;
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
      border:1px solid #CCFBF1;
      color:#334155;
      font-size:10.5px;
      font-weight:750;
    }

    #${rootId} .gl-switch-row input{
      accent-color:#0F766E;
    }

    #${rootId} .gl-subtitle{
      margin:10px 0 6px;
      color:#0F766E;
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
      border:1px solid #99F6E4;
      border-radius:9px;
      background:#FFFFFF;
      color:#0F766E;
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
      border-color:#0F766E;
    }

    #${rootId} button.active{
      border-color:#0F766E;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #38BDF8,
        #0F766E
      );
    }

    #${rootId} .gl-result{
      margin-top:9px;
      padding:9px;
      border-radius:11px;
      background:#DFF7F3;
      border:1px solid #A7D8D2;
      color:#155E59;
      font-size:10.2px;
      font-weight:650;
      line-height:1.48;
      max-height:76px;
      overflow:auto;
    }

    #${rootId} .gl-river-canvas{
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
      background:rgba(255,255,255,0.92);
      border:1px solid #BAE6FD;
      box-shadow:0 5px 15px rgba(15,73,71,0.08);
      text-align:center;
    }

    #${rootId} .gl-summary-card strong{
      display:block;
      color:#0369A1;
      font-size:13px;
      font-weight:900;
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
      🌊
    </div>

    <div>
      <div class="gl-title">
        河流补给、流量过程线与汛期
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        比较不同补给来源，观察降水与流量之间的滞后关系
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 非真实水文预报
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            日期
          </span>

          <span
            class="gl-value"
            data-role="day-value"
          ></span>
        </div>

        <input
          type="range"
          min="1"
          max="365"
          step="1"
          value="${shortNumber(dayOfYear)}"
          data-role="day"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            降水强度
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
            流域气温
          </span>

          <span
            class="gl-value"
            data-role="temperature-value"
          ></span>
        </div>

        <input
          type="range"
          min="-10"
          max="30"
          step="1"
          value="${shortNumber(temperature)}"
          data-role="temperature"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            积雪储量
          </span>

          <span
            class="gl-value"
            data-role="snowpack-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(snowpack)}"
          data-role="snowpack"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            地下水补给
          </span>

          <span
            class="gl-value"
            data-role="groundwater-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(groundwater)}"
          data-role="groundwater"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            湖泊调节能力
          </span>

          <span
            class="gl-value"
            data-role="lake-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(lakeRegulation)}"
          data-role="lake"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            流域汇流速度
          </span>
        </div>

        <select data-role="response-mode">
          <option
            value="fast"
            ${responseMode === 'fast' ? 'selected' : ''}
          >
            快速汇流
          </option>

          <option
            value="medium"
            ${responseMode === 'medium' ? 'selected' : ''}
          >
            中等汇流
          </option>

          <option
            value="slow"
            ${responseMode === 'slow' ? 'selected' : ''}
          >
            缓慢汇流
          </option>
        </select>
      </div>

      <div class="gl-switch-row">
        <span>显示分项补给曲线</span>

        <input
          type="checkbox"
          data-role="component-switch"
          ${showComponents ? 'checked' : ''}
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
        典型水文情境
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="spring"
        >
          🏔️ 春季融雪
        </button>

        <button
          type="button"
          data-scenario="summer"
        >
          🌧️ 夏季雨汛
        </button>

        <button
          type="button"
          data-scenario="stable"
        >
          💧 地下水稳定
        </button>

        <button
          type="button"
          data-scenario="lake"
        >
          🏞️ 湖泊调节
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-river-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="河流补给与流量过程线教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="peak-value"></strong>
          <span>洪峰流量</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="lag-value"></strong>
          <span>峰现滞后时间</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="season-value"></strong>
          <span>水文时期</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="source-value"></strong>
          <span>主要补给来源</span>
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
        x,
        center,
        spread
      ){
        var distance=
          (
            x-center
          )/
          spread;

        return Math.exp(
          -0.5*
          distance*
          distance
        );
      }

      function monthDay(day){
        var monthLengths=[
          31,
          28,
          31,
          30,
          31,
          30,
          31,
          31,
          30,
          31,
          30,
          31
        ];

        var month=0;
        var remaining=Math.round(day);

        while(
          month<
          monthLengths.length-1 &&
          remaining>
          monthLengths[month]
        ){
          remaining-=
            monthLengths[month];

          month+=1;
        }

        return (
          month+1
        )+
        '月'+
        remaining+
        '日';
      }

      function seasonName(day){
        if(day>=60 && day<152){
          return '春季';
        }

        if(day>=152 && day<244){
          return '夏季';
        }

        if(day>=244 && day<335){
          return '秋季';
        }

        return '冬季';
      }

      function responseSettings(mode){
        if(mode==='fast'){
          return {
            rainLag:7,
            rainSpread:6,
            meltLag:15,
            meltSpread:12,
            label:'快速汇流'
          };
        }

        if(mode==='slow'){
          return {
            rainLag:22,
            rainSpread:13,
            meltLag:32,
            meltSpread:19,
            label:'缓慢汇流'
          };
        }

        return {
          rainLag:14,
          rainSpread:9,
          meltLag:24,
          meltSpread:15,
          label:'中等汇流'
        };
      }

      function readState(){
        return {
          day:Number(
            dayInput.value
          ),
          precipitation:Number(
            precipitationInput.value
          ),
          temperature:Number(
            temperatureInput.value
          ),
          snowpack:Number(
            snowpackInput.value
          ),
          groundwater:Number(
            groundwaterInput.value
          ),
          lake:Number(
            lakeInput.value
          ),
          responseMode:
            responseSelect.value
        };
      }

      function calculate(state){
        var settings=
          responseSettings(
            state.responseMode
          );

        var springFactor=
          gaussian(
            state.day,
            112,
            48
          );

        var summerFactor=
          gaussian(
            state.day,
            205,
            58
          );

        var winterFactor=
          gaussian(
            state.day,
            15,
            62
          )+
          gaussian(
            state.day,
            365,
            62
          );

        var thawFactor=clamp(
          (
            state.temperature+
            2
          )/
          20,
          0,
          1.45
        );

        var rainSeasonFactor=
          0.56+
          summerFactor*
          0.44;

        var rainAmplitude=
          state.precipitation*
          rainSeasonFactor*
          (
            state.responseMode==='fast'
              ? 1.16
              : state.responseMode==='slow'
                ? 0.82
                : 1
          );

        var meltAmplitude=
          state.snowpack*
          thawFactor*
          (
            0.35+
            springFactor*
            0.85
          );

        var groundwaterBase=
          4+
          state.groundwater*
          0.20;

        var lakeFactor=
          state.lake/
          100;

        var hours=[];
        var rainfall=[];
        var rainFlow=[];
        var meltFlow=[];
        var groundwaterFlow=[];
        var lakeRelease=[];
        var total=[];

        var peak=0;
        var peakHour=0;
        var rainPeak=0;
        var meltPeak=0;

        for(
          var hour=0;
          hour<=72;
          hour+=2
        ){
          var rainPulse=
            state.precipitation*
            gaussian(
              hour,
              9,
              5
            );

          var rainComponent=
            rainAmplitude*
            gaussian(
              hour,
              9+
              settings.rainLag+
              lakeFactor*
              6,
              settings.rainSpread+
              lakeFactor*
              5
            )*
            (
              1-
              lakeFactor*
              0.42
            );

          var meltComponent=
            meltAmplitude*
            gaussian(
              hour,
              settings.meltLag+
              lakeFactor*
              5,
              settings.meltSpread+
              lakeFactor*
              4
            )*
            (
              1-
              lakeFactor*
              0.18
            );

          var groundwaterComponent=
            groundwaterBase*
            (
              0.94+
              0.06*
              Math.sin(
                hour/
                72*
                Math.PI
              )
            );

          var delayedRelease=
            (
              rainAmplitude*
              0.32+
              meltAmplitude*
              0.18
            )*
            lakeFactor*
            gaussian(
              hour,
              41+
              lakeFactor*
              12,
              18+
              lakeFactor*
              9
            );

          var totalValue=
            rainComponent+
            meltComponent+
            groundwaterComponent+
            delayedRelease;

          hours.push(hour);
          rainfall.push(rainPulse);
          rainFlow.push(rainComponent);
          meltFlow.push(meltComponent);
          groundwaterFlow.push(
            groundwaterComponent
          );
          lakeRelease.push(
            delayedRelease
          );
          total.push(totalValue);

          rainPeak=Math.max(
            rainPeak,
            rainComponent
          );

          meltPeak=Math.max(
            meltPeak,
            meltComponent
          );

          if(totalValue>peak){
            peak=totalValue;
            peakHour=hour;
          }
        }

        var floodThreshold=
          68+
          state.groundwater*
          0.08;

        var average=
          total.reduce(
            function(sum,value){
              return sum+value;
            },
            0
          )/
          total.length;

        var sourceScores=[
          {
            key:'rain',
            label:'雨水补给',
            score:rainPeak
          },
          {
            key:'melt',
            label:'冰雪融水',
            score:meltPeak
          },
          {
            key:'groundwater',
            label:'地下水',
            score:groundwaterBase
          },
          {
            key:'lake',
            label:'湖泊调节',
            score:
              state.lake*
              0.22
          }
        ];

        sourceScores.sort(
          function(left,right){
            return right.score-left.score;
          }
        );

        var period;

        if(peak>=floodThreshold){
          period='汛期';
        }else if(
          average<
          floodThreshold*
          0.38
        ){
          period='枯水期';
        }else{
          period='平水期';
        }

        if(
          winterFactor>0.65 &&
          state.temperature<0 &&
          state.precipitation<35
        ){
          period='冬季枯水';
        }

        return {
          settings:settings,
          springFactor:springFactor,
          summerFactor:summerFactor,
          hours:hours,
          rainfall:rainfall,
          rainFlow:rainFlow,
          meltFlow:meltFlow,
          groundwaterFlow:groundwaterFlow,
          lakeRelease:lakeRelease,
          total:total,
          peak:peak,
          peakHour:peakHour,
          average:average,
          floodThreshold:floodThreshold,
          dominantSource:
            sourceScores[0].label,
          sourceScores:sourceScores,
          period:period
        };
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

        context.textBaseline=
          'middle';

        context.fillText(
          text,
          x,
          y
        );

        context.restore();
      }

      function drawPolyline(
        context,
        values,
        maxValue,
        chart,
        color,
        width,
        dash
      ){
        context.save();
        context.strokeStyle=color;
        context.lineWidth=width;
        context.lineJoin='round';
        context.lineCap='round';

        if(dash){
          context.setLineDash(dash);
        }

        context.beginPath();

        values.forEach(
          function(value,index){
            var x=
              chart.x+
              index/
              (
                values.length-1
              )*
              chart.width;

            var y=
              chart.y+
              chart.height-
              value/
              maxValue*
              chart.height;

            if(index===0){
              context.moveTo(x,y);
            }else{
              context.lineTo(x,y);
            }
          }
        );

        context.stroke();
        context.restore();
      }

      function render(){
        if(!root.isConnected){
          if(timer){
            window.clearTimeout(timer);
            timer=null;
          }

          return;
        }

        var state=readState();
        var model=calculate(state);
        var context=
          canvas.getContext('2d');

        if(!context){
          return;
        }

        dayValue.textContent=
          monthDay(state.day);

        precipitationValue.textContent=
          Math.round(
            state.precipitation
          )+
          ' 单位';

        temperatureValue.textContent=
          Math.round(
            state.temperature
          )+
          '℃';

        snowpackValue.textContent=
          Math.round(
            state.snowpack
          )+
          '%';

        groundwaterValue.textContent=
          Math.round(
            state.groundwater
          )+
          '%';

        lakeValue.textContent=
          Math.round(
            state.lake
          )+
          '%';

        peakValue.textContent=
          Math.round(
            model.peak
          )+
          ' 单位';

        lagValue.textContent=
          model.peakHour+
          ' 小时';

        seasonValue.textContent=
          model.period;

        sourceValue.textContent=
          model.dominantSource;

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
          '#ECFDF5'
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
          'rgba(255,255,255,0.90)',
          '#BAE6FD'
        );

        drawText(
          context,
          '72小时降水—流量过程线',
          40,
          43,
          14,
          '#164E63',
          850,
          'left'
        );

        drawText(
          context,
          monthDay(state.day)+
          ' · '+
          seasonName(state.day)+
          ' · '+
          model.settings.label,
          780,
          43,
          10,
          '#64748B',
          700,
          'right'
        );

        var chart={
          x:58,
          y:88,
          width:536,
          height:229
        };

        context.strokeStyle='#CBD5E1';
        context.lineWidth=1;

        for(
          var gridIndex=0;
          gridIndex<=4;
          gridIndex+=1
        ){
          var gridY=
            chart.y+
            gridIndex/
            4*
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
        }

        for(
          var hourLabel=0;
          hourLabel<=72;
          hourLabel+=12
        ){
          var hourX=
            chart.x+
            hourLabel/
            72*
            chart.width;

          context.strokeStyle='#E2E8F0';
          context.beginPath();
          context.moveTo(
            hourX,
            chart.y
          );
          context.lineTo(
            hourX,
            chart.y+
            chart.height
          );
          context.stroke();

          drawText(
            context,
            hourLabel+'h',
            hourX,
            chart.y+
            chart.height+
            17,
            9,
            '#64748B',
            650,
            'center'
          );
        }

        var rainfallMax=Math.max.apply(
          null,
          model.rainfall.concat([1])
        );

        context.fillStyle=
          'rgba(14,165,233,0.34)';

        model.rainfall.forEach(
          function(value,index){
            var barWidth=
              chart.width/
              model.rainfall.length*
              0.74;

            var x=
              chart.x+
              index/
              (
                model.rainfall.length-1
              )*
              chart.width-
              barWidth/2;

            var barHeight=
              value/
              rainfallMax*
              48;

            context.fillRect(
              x,
              chart.y,
              barWidth,
              barHeight
            );
          }
        );

        var maxFlow=Math.max.apply(
          null,
          model.total.concat([
            model.floodThreshold,
            1
          ])
        )*1.15;

        if(componentSwitch.checked){
          drawPolyline(
            context,
            model.rainFlow,
            maxFlow,
            chart,
            '#0EA5E9',
            2,
            [6,4]
          );

          drawPolyline(
            context,
            model.meltFlow,
            maxFlow,
            chart,
            '#8B5CF6',
            2,
            [5,4]
          );

          drawPolyline(
            context,
            model.groundwaterFlow,
            maxFlow,
            chart,
            '#16A34A',
            2,
            [4,4]
          );

          drawPolyline(
            context,
            model.lakeRelease,
            maxFlow,
            chart,
            '#F59E0B',
            2,
            [3,4]
          );
        }

        drawPolyline(
          context,
          model.total,
          maxFlow,
          chart,
          '#0369A1',
          4,
          null
        );

        var thresholdY=
          chart.y+
          chart.height-
          model.floodThreshold/
          maxFlow*
          chart.height;

        context.save();
        context.strokeStyle='#DC2626';
        context.lineWidth=1.5;
        context.setLineDash([8,5]);

        context.beginPath();
        context.moveTo(
          chart.x,
          thresholdY
        );
        context.lineTo(
          chart.x+
          chart.width,
          thresholdY
        );
        context.stroke();

        context.restore();

        drawText(
          context,
          '教学汛期判别线',
          chart.x+
          chart.width-
          3,
          thresholdY-
          9,
          9,
          '#DC2626',
          750,
          'right'
        );

        var peakX=
          chart.x+
          model.peakHour/
          72*
          chart.width;

        var peakY=
          chart.y+
          chart.height-
          model.peak/
          maxFlow*
          chart.height;

        context.fillStyle='#DC2626';
        context.beginPath();
        context.arc(
          peakX,
          peakY,
          5,
          0,
          Math.PI*2
        );
        context.fill();

        drawText(
          context,
          '洪峰 '+Math.round(model.peak),
          peakX+
          9,
          peakY-
          10,
          10,
          '#B91C1C',
          850,
          'left'
        );

        context.strokeStyle='#0F766E';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(
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
        context.stroke();

        drawText(
          context,
          '流量',
          26,
          chart.y+
          chart.height/2,
          10,
          '#475569',
          700,
          'center'
        );

        drawText(
          context,
          '时间',
          chart.x+
          chart.width/2,
          chart.y+
          chart.height+
          34,
          10,
          '#475569',
          700,
          'center'
        );

        fillRoundedRect(
          context,
          620,
          73,
          160,
          244,
          14,
          '#F8FAFC',
          '#CCFBF1'
        );

        drawText(
          context,
          '补给贡献比较',
          638,
          95,
          12,
          '#115E59',
          850,
          'left'
        );

        var maximumScore=Math.max.apply(
          null,
          model.sourceScores.map(
            function(item){
              return item.score;
            }
          ).concat([1])
        );

        model.sourceScores.forEach(
          function(item,index){
            var y=
              125+
              index*
              45;

            drawText(
              context,
              item.label,
              638,
              y,
              10,
              '#475569',
              750,
              'left'
            );

            context.fillStyle='#E2E8F0';
            context.fillRect(
              638,
              y+11,
              119,
              8
            );

            var colors={
              rain:'#0EA5E9',
              melt:'#8B5CF6',
              groundwater:'#16A34A',
              lake:'#F59E0B'
            };

            context.fillStyle=
              colors[item.key] ||
              '#64748B';

            context.fillRect(
              638,
              y+11,
              clamp(
                item.score/
                maximumScore*
                119,
                3,
                119
              ),
              8
            );
          }
        );

        var legendItems=[
          ['总流量','#0369A1'],
          ['雨水补给','#0EA5E9'],
          ['冰雪融水','#8B5CF6'],
          ['地下水','#16A34A'],
          ['湖泊释放','#F59E0B']
        ];

        legendItems.forEach(
          function(item,index){
            var x=
              67+
              index*
              104;

            context.strokeStyle=item[1];
            context.lineWidth=
              index===0
                ? 4
                : 2;

            context.beginPath();
            context.moveTo(
              x,
              365
            );
            context.lineTo(
              x+22,
              365
            );
            context.stroke();

            drawText(
              context,
              item[0],
              x+28,
              365,
              8.5,
              '#475569',
              700,
              'left'
            );
          }
        );

        var explanation;

        if(
          model.dominantSource===
          '冰雪融水'
        ){
          explanation=
            '气温升高使积雪融化，流量上升过程通常较降雨型洪峰平缓，'+
            '春季可能形成融雪汛期。';
        }else if(
          model.dominantSource===
          '雨水补给'
        ){
          explanation=
            '降水是当前主要补给来源。汇流越快，洪峰出现越早、'+
            '峰值越高，降水与流量峰值之间存在滞后。';
        }else if(
          model.dominantSource===
          '湖泊调节'
        ){
          explanation=
            '湖泊先拦蓄部分来水，再缓慢释放，使洪峰降低、'+
            '峰现时间推迟，并可在低水期补充河流流量。';
        }else{
          explanation=
            '地下水补给变化缓慢，可维持较稳定的基流，'+
            '减小河流流量的短期波动，并缓解枯水期断流风险。';
        }

        result.textContent=
          monthDay(state.day)+
          '，当前判定为'+
          model.period+
          '；主要补给为'+
          model.dominantSource+
          '。'+
          explanation+
          ' 本结果仅用于课堂过程比较。';
      }

      function applyScenario(name){
        var scenarios={
          spring:{
            day:112,
            precipitation:30,
            temperature:9,
            snowpack:92,
            groundwater:38,
            lake:20,
            responseMode:'medium'
          },
          summer:{
            day:205,
            precipitation:92,
            temperature:26,
            snowpack:6,
            groundwater:38,
            lake:10,
            responseMode:'fast'
          },
          stable:{
            day:278,
            precipitation:22,
            temperature:13,
            snowpack:4,
            groundwater:92,
            lake:25,
            responseMode:'slow'
          },
          lake:{
            day:198,
            precipitation:78,
            temperature:23,
            snowpack:8,
            groundwater:42,
            lake:92,
            responseMode:'medium'
          }
        };

        var scenario=scenarios[name];

        if(!scenario){
          return;
        }

        dayInput.value=String(
          scenario.day
        );

        precipitationInput.value=String(
          scenario.precipitation
        );

        temperatureInput.value=String(
          scenario.temperature
        );

        snowpackInput.value=String(
          scenario.snowpack
        );

        groundwaterInput.value=String(
          scenario.groundwater
        );

        lakeInput.value=String(
          scenario.lake
        );

        responseSelect.value=
          scenario.responseMode;

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

      function schedule(){
        if(timer){
          window.clearTimeout(timer);
          timer=null;
        }

        if(
          !autoSwitch.checked ||
          !root.isConnected
        ){
          return;
        }

        timer=window.setTimeout(
          function(){
            if(!root.isConnected){
              return;
            }

            scenarioIndex=
              (
                scenarioIndex+
                1
              )%
              scenarioOrder.length;

            applyScenario(
              scenarioOrder[
                scenarioIndex
              ]
            );

            schedule();
          },
          3100
        );
      }

      var dayInput=query(
        '[data-role="day"]'
      );

      var precipitationInput=query(
        '[data-role="precipitation"]'
      );

      var temperatureInput=query(
        '[data-role="temperature"]'
      );

      var snowpackInput=query(
        '[data-role="snowpack"]'
      );

      var groundwaterInput=query(
        '[data-role="groundwater"]'
      );

      var lakeInput=query(
        '[data-role="lake"]'
      );

      var responseSelect=query(
        '[data-role="response-mode"]'
      );

      var componentSwitch=query(
        '[data-role="component-switch"]'
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

      var dayValue=query(
        '[data-role="day-value"]'
      );

      var precipitationValue=query(
        '[data-role="precipitation-value"]'
      );

      var temperatureValue=query(
        '[data-role="temperature-value"]'
      );

      var snowpackValue=query(
        '[data-role="snowpack-value"]'
      );

      var groundwaterValue=query(
        '[data-role="groundwater-value"]'
      );

      var lakeValue=query(
        '[data-role="lake-value"]'
      );

      var peakValue=query(
        '[data-role="peak-value"]'
      );

      var lagValue=query(
        '[data-role="lag-value"]'
      );

      var seasonValue=query(
        '[data-role="season-value"]'
      );

      var sourceValue=query(
        '[data-role="source-value"]'
      );

      if(
        !dayInput ||
        !precipitationInput ||
        !temperatureInput ||
        !snowpackInput ||
        !groundwaterInput ||
        !lakeInput ||
        !responseSelect ||
        !componentSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !result ||
        !canvas ||
        !dayValue ||
        !precipitationValue ||
        !temperatureValue ||
        !snowpackValue ||
        !groundwaterValue ||
        !lakeValue ||
        !peakValue ||
        !lagValue ||
        !seasonValue ||
        !sourceValue
      ){
        return;
      }

      var scenarioOrder=[
        'spring',
        'summer',
        'stable',
        'lake'
      ];

      var scenarioIndex=-1;
      var timer=null;

      [
        dayInput,
        precipitationInput,
        temperatureInput,
        snowpackInput,
        groundwaterInput,
        lakeInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            clearScenarioSelection
          );
        }
      );

      responseSelect.addEventListener(
        'change',
        clearScenarioSelection
      );

      componentSwitch.addEventListener(
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
                'spring';

              scenarioIndex=
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
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_RIVER_REGIME:
GeographyLabTemplate[] = [
  {
    id: 'geography-river-recharge-hydrograph-flood-season',
    group: '🌊 水循环、河流与海洋系统',
    name: '河流补给、流量过程线与汛期',
    emoji: '🌊',
    desc: '比较雨水、冰雪融水、地下水和湖泊调节，观察流量过程线、洪峰、汛期和滞后时间。',
    params: [
      {
        key: 'dayOfYear',
        label: '初始日期序号',
        type: 'number',
        min: 1,
        max: 365,
        step: 1,
        defaultValue: 195,
        hint: '用于表现季节差异，不代表具体年份。',
      },
      {
        key: 'precipitation',
        label: '降水强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 65,
        hint: '降水越强，雨水补给形成的径流过程通常越明显。',
      },
      {
        key: 'temperature',
        label: '流域气温',
        type: 'number',
        min: -10,
        max: 30,
        step: 1,
        defaultValue: 18,
        hint: '气温升高会增强积雪融化，但需同时存在积雪储量。',
      },
      {
        key: 'snowpack',
        label: '积雪储量',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 35,
        hint: '积雪越多，适宜温度下形成的融雪补给越明显。',
      },
      {
        key: 'groundwater',
        label: '地下水补给能力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 45,
        hint: '地下水主要维持相对稳定的河流基流。',
      },
      {
        key: 'lakeRegulation',
        label: '湖泊调节能力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 35,
        hint: '湖泊调节可降低洪峰并延长径流过程。',
      },
      {
        key: 'responseMode',
        label: '流域汇流速度',
        type: 'select',
        options: [
          {
            label: '快速汇流',
            value: 'fast',
          },
          {
            label: '中等汇流',
            value: 'medium',
          },
          {
            label: '缓慢汇流',
            value: 'slow',
          },
        ],
        defaultValue: 'medium',
      },
      {
        key: 'showComponents',
        label: '显示分项补给曲线',
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
    buildHTML: buildRiverRegimeHTML,
  },
]
