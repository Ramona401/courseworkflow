/**
 * geographyLabTemplatesAtmospherePressureWindBelts.ts
 *
 * 地理第34批：
 *   气压带、风带与季节移动。
 *
 * 教学边界：
 *   - 展示赤道低压带、副热带高压带、副极地低压带、
 *     北极高压带和南极高压带；
 *   - 气压带宽度、风向和季节移动幅度均为课堂示意；
 *   - 三圈环流忽略海陆分布、地形和实际天气系统扰动；
 *   - 不用于天气预报、航空导航或真实风场判断。
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

function buildPressureWindBeltsHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const dayOfYear = Math.max(
    1,
    Math.min(
      365,
      numberValue(params, 'dayOfYear', 172),
    ),
  )

  const shiftScale = Math.max(
    0.5,
    Math.min(
      1.5,
      numberValue(params, 'shiftScale', 1),
    ),
  )

  const showPressure = booleanValue(
    params,
    'showPressure',
    true,
  )

  const showWinds = booleanValue(
    params,
    'showWinds',
    true,
  )

  const showCells = booleanValue(
    params,
    'showCells',
    true,
  )

  return `
<div id="${rootId}" class="gl-pressure-wind-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border-radius:18px;
      border:1px solid #99F6E4;
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
      background:linear-gradient(135deg,#DBEAFE,#CCFBF1);
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
      font-size:11.5px;
      white-space:nowrap;
    }

    #${rootId} .gl-body{
      height:calc(100% - 52px);
      display:grid;
      grid-template-columns:245px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      padding:14px;
      overflow:auto;
      border-right:1px solid #CCFBF1;
      background:linear-gradient(180deg,#F0FDFA,#EFF6FF);
    }

    #${rootId} .gl-stage{
      min-width:0;
      min-height:0;
      padding:8px;
      background:radial-gradient(
        circle at 48% 22%,
        #FFFFFF 0%,
        #F8FAFC 60%,
        #E0F2FE 100%
      );
    }

    #${rootId} .gl-row{
      margin-bottom:13px;
    }

    #${rootId} .gl-label-line{
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:8px;
      margin-bottom:6px;
    }

    #${rootId} .gl-label{
      color:#334155;
      font-size:12px;
      font-weight:700;
    }

    #${rootId} .gl-value{
      padding:3px 8px;
      border-radius:999px;
      background:#CCFBF1;
      color:#0F766E;
      font-size:11.5px;
      font-weight:850;
    }

    #${rootId} input[type=range]{
      width:100%;
      height:6px;
      margin:0;
      appearance:none;
      border-radius:999px;
      outline:none;
      background:#BAE6FD;
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:17px;
      height:17px;
      appearance:none;
      border-radius:50%;
      background:linear-gradient(135deg,#38BDF8,#0F766E);
      border:2px solid #FFFFFF;
      box-shadow:0 1px 5px rgba(14,116,144,0.42);
    }

    #${rootId} .gl-season-grid,
    #${rootId} .gl-button-grid{
      display:grid;
      grid-template-columns:1fr 1fr;
      gap:7px;
      margin-bottom:10px;
    }

    #${rootId} button{
      min-height:34px;
      padding:7px 8px;
      border:1px solid #99F6E4;
      border-radius:10px;
      background:#FFFFFF;
      color:#0F766E;
      font-size:11.5px;
      font-weight:800;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#0E7490;
      color:#FFFFFF;
      background:linear-gradient(135deg,#38BDF8,#0F766E);
    }

    #${rootId} .gl-result{
      padding:10px;
      border-radius:12px;
      background:#CCFBF1;
      color:#115E59;
      font-size:11.5px;
      font-weight:650;
      line-height:1.55;
    }

    #${rootId} .gl-atmosphere-canvas{
      width:100%;
      height:100%;
      display:block;
    }
  </style>

  <div class="gl-head">
    <div style="font-size:23px;">🌐</div>

    <div>
      <div class="gl-title">
        气压带、风带与季节移动
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        观察太阳直射点变化与全球气压带、风带的南北移动
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 非真实风场
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">日期</span>

          <span
            class="gl-value"
            data-role="date-value"
          >
            6月21日
          </span>
        </div>

        <input
          type="range"
          min="1"
          max="365"
          step="1"
          value="${dayOfYear}"
          data-role="day"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            季节移动幅度
          </span>

          <span
            class="gl-value"
            data-role="shift-value"
          >
            1.0倍
          </span>
        </div>

        <input
          type="range"
          min="0.5"
          max="1.5"
          step="0.1"
          value="${shiftScale}"
          data-role="shift-scale"
        />
      </div>

      <div class="gl-season-grid">
        <button type="button" data-season-day="80">
          春分
        </button>

        <button type="button" data-season-day="172">
          夏至
        </button>

        <button type="button" data-season-day="266">
          秋分
        </button>

        <button type="button" data-season-day="355">
          冬至
        </button>
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-role="pressure-toggle"
          data-active="${showPressure}"
        >
          气压带
        </button>

        <button
          type="button"
          data-role="wind-toggle"
          data-active="${showWinds}"
        >
          风带
        </button>

        <button
          type="button"
          data-role="cell-toggle"
          data-active="${showCells}"
        >
          三圈环流
        </button>

        <button
          type="button"
          data-role="auto-toggle"
          data-active="false"
        >
          全年演示
        </button>

        <button
          type="button"
          data-role="reset"
        >
          恢复初始
        </button>

        <button
          type="button"
          data-role="pause-marker"
          data-active="true"
        >
          观察当前
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      >
        气压带和风带会随太阳直射点的季节变化而南北移动。
      </div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-atmosphere-canvas"
        width="780"
        height="440"
        data-role="canvas"
        aria-label="气压带风带与季节移动教学示意图"
      ></canvas>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var dayInput =
        root.querySelector('[data-role="day"]');

      var shiftInput =
        root.querySelector('[data-role="shift-scale"]');

      var dateValue =
        root.querySelector('[data-role="date-value"]');

      var shiftValue =
        root.querySelector('[data-role="shift-value"]');

      var seasonButtons =
        root.querySelectorAll('[data-season-day]');

      var pressureToggle =
        root.querySelector('[data-role="pressure-toggle"]');

      var windToggle =
        root.querySelector('[data-role="wind-toggle"]');

      var cellToggle =
        root.querySelector('[data-role="cell-toggle"]');

      var autoToggle =
        root.querySelector('[data-role="auto-toggle"]');

      var resetButton =
        root.querySelector('[data-role="reset"]');

      var pauseMarker =
        root.querySelector('[data-role="pause-marker"]');

      var result =
        root.querySelector('[data-role="result"]');

      var canvas =
        root.querySelector('[data-role="canvas"]');

      if(
        !dayInput ||
        !shiftInput ||
        !dateValue ||
        !shiftValue ||
        !seasonButtons.length ||
        !pressureToggle ||
        !windToggle ||
        !cellToggle ||
        !autoToggle ||
        !resetButton ||
        !pauseMarker ||
        !result ||
        !canvas
      ){
        return;
      }

      var context =
        canvas.getContext('2d');

      if(!context)return;

      var initialDay = ${dayOfYear};
      var initialShiftScale = ${shiftScale};

      var state = {
        showPressure:${showPressure},
        showWinds:${showWinds},
        showCells:${showCells},
        auto:false,
        startedAt:0,
        particlePhase:0,
        raf:0
      };

      var width = canvas.width;
      var height = canvas.height;

      function clamp(value,min,max){
        return Math.max(
          min,
          Math.min(max,value)
        );
      }

      function solarDeclination(day){
        return 23.44*Math.sin(
          2*Math.PI*(284+day)/365
        );
      }

      function monthDay(day){
        var monthLengths = [
          31,28,31,30,31,30,
          31,31,30,31,30,31
        ];

        var month = 0;
        var remaining = Math.round(day);

        while(
          month<monthLengths.length-1 &&
          remaining>monthLengths[month]
        ){
          remaining -= monthLengths[month];
          month += 1;
        }

        return (
          month+1
        )+'月'+remaining+'日';
      }

      function seasonName(day){
        if(day>=80 && day<172){
          return '北半球春季';
        }

        if(day>=172 && day<266){
          return '北半球夏季';
        }

        if(day>=266 && day<355){
          return '北半球秋季';
        }

        return '北半球冬季';
      }

      function latitudeText(latitude){
        var absolute = Math.abs(latitude);

        if(absolute<0.3){
          return '赤道附近';
        }

        return absolute.toFixed(1)+
          '°'+
          (
            latitude>0
              ? 'N'
              : 'S'
          );
      }

      function roundedRect(
        x,
        y,
        w,
        h,
        radius
      ){
        var r = Math.min(
          radius,
          w/2,
          h/2
        );

        context.beginPath();
        context.moveTo(x+r,y);
        context.lineTo(x+w-r,y);
        context.quadraticCurveTo(
          x+w,y,x+w,y+r
        );
        context.lineTo(x+w,y+h-r);
        context.quadraticCurveTo(
          x+w,y+h,x+w-r,y+h
        );
        context.lineTo(x+r,y+h);
        context.quadraticCurveTo(
          x,y+h,x,y+h-r
        );
        context.lineTo(x,y+r);
        context.quadraticCurveTo(
          x,y,x+r,y
        );
        context.closePath();
      }

      function fillRoundedRect(
        x,
        y,
        w,
        h,
        radius,
        fill,
        stroke
      ){
        roundedRect(
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
          context.lineWidth=1.5;
          context.stroke();
        }
      }

      function drawText(
        text,
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

        context.textBaseline='middle';

        context.fillText(
          text,
          x,
          y
        );

        context.restore();
      }

      function latitudeToY(latitude){
        var top = 30;
        var bottom = 400;

        return top+
          (
            90-latitude
          )/180*
          (
            bottom-top
          );
      }

      function drawArrow(
        x1,
        y1,
        x2,
        y2,
        color,
        lineWidth
      ){
        var angle = Math.atan2(
          y2-y1,
          x2-x1
        );

        var headLength=10;

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=lineWidth || 2.6;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(x1,y1);
        context.lineTo(x2,y2);
        context.stroke();

        context.beginPath();
        context.moveTo(x2,y2);

        context.lineTo(
          x2-headLength*Math.cos(angle-Math.PI/6),
          y2-headLength*Math.sin(angle-Math.PI/6)
        );

        context.lineTo(
          x2-headLength*Math.cos(angle+Math.PI/6),
          y2-headLength*Math.sin(angle+Math.PI/6)
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function drawCurvedArrow(
        startX,
        startY,
        controlX,
        controlY,
        endX,
        endY,
        color
      ){
        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=2.4;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(startX,startY);

        context.quadraticCurveTo(
          controlX,
          controlY,
          endX,
          endY
        );

        context.stroke();

        var angle = Math.atan2(
          endY-controlY,
          endX-controlX
        );

        var headLength=9;

        context.beginPath();
        context.moveTo(endX,endY);

        context.lineTo(
          endX-headLength*Math.cos(angle-Math.PI/6),
          endY-headLength*Math.sin(angle-Math.PI/6)
        );

        context.lineTo(
          endX-headLength*Math.cos(angle+Math.PI/6),
          endY-headLength*Math.sin(angle+Math.PI/6)
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function pressureColor(type){
        return type==='H'
          ? '#2563EB'
          : '#DC2626';
      }

      function pressureFill(type){
        return type==='H'
          ? 'rgba(219,234,254,0.84)'
          : 'rgba(254,226,226,0.84)';
      }

      function drawPressureBand(
        latitude,
        type,
        label,
        shift
      ){
        var adjustedLatitude = clamp(
          latitude+shift,
          -88,
          88
        );

        var y = latitudeToY(
          adjustedLatitude
        );

        context.fillStyle=pressureFill(type);

        context.fillRect(
          140,
          y-8,
          398,
          16
        );

        context.strokeStyle=pressureColor(type);
        context.lineWidth=1.5;

        context.beginPath();
        context.moveTo(140,y);
        context.lineTo(538,y);
        context.stroke();

        drawText(
          type,
          153,
          y,
          12,
          pressureColor(type),
          900,
          'center'
        );

        drawText(
          label,
          550,
          y,
          10.2,
          pressureColor(type),
          800,
          'left'
        );

        return y;
      }

      function drawWindZone(
        northLatitude,
        southLatitude,
        name,
        direction,
        color,
        shift
      ){
        var northY = latitudeToY(
          clamp(
            northLatitude+shift,
            -88,
            88
          )
        );

        var southY = latitudeToY(
          clamp(
            southLatitude+shift,
            -88,
            88
          )
        );

        var centerY = (
          northY+southY
        )/2;

        var startX =
          direction==='east'
            ? 205
            : 475;

        var endX =
          direction==='east'
            ? 475
            : 205;

        var vertical =
          northLatitude>0
            ? -12
            : 12;

        for(
          var index=0;
          index<3;
          index+=1
        ){
          var offset = (
            index-1
          )*17;

          drawArrow(
            startX,
            centerY+offset-vertical,
            endX,
            centerY+offset+vertical,
            color,
            2.2
          );
        }

        drawText(
          name,
          340,
          centerY,
          10.5,
          color,
          850,
          'center'
        );
      }

      function drawCell(
        upperLatitude,
        lowerLatitude,
        label,
        side,
        clockwise,
        shift
      ){
        var topY = latitudeToY(
          clamp(
            upperLatitude+shift,
            -88,
            88
          )
        );

        var bottomY = latitudeToY(
          clamp(
            lowerLatitude+shift,
            -88,
            88
          )
        );

        var centerY = (
          topY+bottomY
        )/2;

        var outerX =
          side==='left'
            ? 88
            : 590;

        var innerX =
          side==='left'
            ? 128
            : 548;

        var color =
          label==='Hadley'
            ? '#0F766E'
            : label==='Ferrel'
              ? '#7C3AED'
              : '#0369A1';

        if(clockwise){
          drawCurvedArrow(
            innerX,
            bottomY,
            outerX,
            centerY,
            innerX,
            topY,
            color
          );

          drawCurvedArrow(
            innerX,
            topY,
            outerX+6,
            centerY,
            innerX,
            bottomY,
            color
          );
        }else{
          drawCurvedArrow(
            innerX,
            topY,
            outerX,
            centerY,
            innerX,
            bottomY,
            color
          );

          drawCurvedArrow(
            innerX,
            bottomY,
            outerX+6,
            centerY,
            innerX,
            topY,
            color
          );
        }

        drawText(
          label,
          side==='left'
            ? 42
            : 642,
          centerY,
          9.5,
          color,
          800,
          'center'
        );
      }

      function drawParticle(
        progress,
        latitude,
        shift,
        color
      ){
        var y = latitudeToY(
          clamp(
            latitude+shift,
            -88,
            88
          )
        );

        var x =
          190+
          (
            progress%1
          )*305;

        context.save();
        context.fillStyle=color;
        context.shadowColor=color;
        context.shadowBlur=7;

        context.beginPath();
        context.arc(
          x,
          y,
          4.5,
          0,
          Math.PI*2
        );

        context.fill();
        context.restore();
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var day = clamp(
          parseFloat(dayInput.value) || 1,
          1,
          365
        );

        var scale = clamp(
          parseFloat(shiftInput.value) || 1,
          0.5,
          1.5
        );

        var declination =
          solarDeclination(day);

        var equatorialShift =
          declination*
          0.55*
          scale;

        var tropicalShift =
          equatorialShift*
          0.70;

        var subpolarShift =
          equatorialShift*
          0.35;

        var polarShift =
          equatorialShift*
          0.12;

        dateValue.textContent =
          monthDay(day);

        shiftValue.textContent =
          scale.toFixed(1)+'倍';

        pressureToggle.setAttribute(
          'data-active',
          state.showPressure
            ? 'true'
            : 'false'
        );

        windToggle.setAttribute(
          'data-active',
          state.showWinds
            ? 'true'
            : 'false'
        );

        cellToggle.setAttribute(
          'data-active',
          state.showCells
            ? 'true'
            : 'false'
        );

        autoToggle.setAttribute(
          'data-active',
          state.auto
            ? 'true'
            : 'false'
        );

        pauseMarker.setAttribute(
          'data-active',
          state.auto
            ? 'false'
            : 'true'
        );

        var background =
          context.createLinearGradient(
            0,
            0,
            0,
            height
          );

        background.addColorStop(
          0,
          '#DBEAFE'
        );

        background.addColorStop(
          0.5,
          '#F8FAFC'
        );

        background.addColorStop(
          1,
          '#E0F2FE'
        );

        context.fillStyle=background;
        context.fillRect(
          0,
          0,
          width,
          height
        );

        fillRoundedRect(
          18,
          18,
          744,
          398,
          18,
          'rgba(255,255,255,0.80)',
          '#BAE6FD'
        );

        var earthGradient =
          context.createLinearGradient(
            0,
            30,
            0,
            400
          );

        earthGradient.addColorStop(
          0,
          '#E0F2FE'
        );

        earthGradient.addColorStop(
          0.48,
          '#ECFDF5'
        );

        earthGradient.addColorStop(
          0.52,
          '#FEF3C7'
        );

        earthGradient.addColorStop(
          1,
          '#E0F2FE'
        );

        context.fillStyle=earthGradient;

        context.fillRect(
          140,
          30,
          398,
          370
        );

        context.strokeStyle='#94A3B8';
        context.lineWidth=1;

        [
          90,60,30,0,-30,-60,-90
        ].forEach(
          function(latitude){
            var y =
              latitudeToY(latitude);

            context.beginPath();
            context.moveTo(140,y);
            context.lineTo(538,y);
            context.stroke();

            drawText(
              latitude===0
                ? '0°'
                : Math.abs(latitude)+
                  '°'+
                  (
                    latitude>0
                      ? 'N'
                      : 'S'
                  ),
              127,
              y,
              9.5,
              '#64748B',
              700,
              'right'
            );
          }
        );

        context.strokeStyle='#0284C7';
        context.lineWidth=1.5;

        context.strokeRect(
          140,
          30,
          398,
          370
        );

        if(state.showPressure){
          drawPressureBand(
            90,
            'H',
            '北极高压带',
            polarShift
          );

          drawPressureBand(
            60,
            'L',
            '北副极地低压带',
            subpolarShift
          );

          drawPressureBand(
            30,
            'H',
            '北副热带高压带',
            tropicalShift
          );

          drawPressureBand(
            0,
            'L',
            '赤道低压带',
            equatorialShift
          );

          drawPressureBand(
            -30,
            'H',
            '南副热带高压带',
            tropicalShift
          );

          drawPressureBand(
            -60,
            'L',
            '南副极地低压带',
            subpolarShift
          );

          drawPressureBand(
            -90,
            'H',
            '南极高压带',
            polarShift
          );
        }

        if(state.showWinds){
          drawWindZone(
            88,
            62,
            '北极地东风',
            'west',
            '#0369A1',
            subpolarShift
          );

          drawWindZone(
            58,
            32,
            '北半球西风',
            'east',
            '#7C3AED',
            tropicalShift
          );

          drawWindZone(
            28,
            2,
            '东北信风',
            'west',
            '#0F766E',
            equatorialShift
          );

          drawWindZone(
            -2,
            -28,
            '东南信风',
            'west',
            '#0F766E',
            equatorialShift
          );

          drawWindZone(
            -32,
            -58,
            '南半球西风',
            'east',
            '#7C3AED',
            tropicalShift
          );

          drawWindZone(
            -62,
            -88,
            '南极地东风',
            'west',
            '#0369A1',
            subpolarShift
          );

          drawParticle(
            state.particlePhase,
            15,
            equatorialShift,
            '#0F766E'
          );

          drawParticle(
            state.particlePhase+0.33,
            45,
            tropicalShift,
            '#7C3AED'
          );

          drawParticle(
            state.particlePhase+0.66,
            -45,
            tropicalShift,
            '#7C3AED'
          );
        }

        if(state.showCells){
          drawCell(
            88,
            62,
            'Polar',
            'left',
            true,
            subpolarShift
          );

          drawCell(
            58,
            32,
            'Ferrel',
            'left',
            false,
            tropicalShift
          );

          drawCell(
            28,
            2,
            'Hadley',
            'left',
            true,
            equatorialShift
          );

          drawCell(
            -2,
            -28,
            'Hadley',
            'right',
            false,
            equatorialShift
          );

          drawCell(
            -32,
            -58,
            'Ferrel',
            'right',
            true,
            tropicalShift
          );

          drawCell(
            -62,
            -88,
            'Polar',
            'right',
            false,
            subpolarShift
          );
        }

        var directY =
          latitudeToY(declination);

        context.strokeStyle='#F97316';
        context.lineWidth=2.5;
        context.setLineDash([8,5]);

        context.beginPath();
        context.moveTo(140,directY);
        context.lineTo(538,directY);
        context.stroke();

        context.setLineDash([]);

        fillRoundedRect(
          585,
          40,
          158,
          196,
          15,
          '#FFFFFF',
          '#99F6E4'
        );

        drawText(
          '当前状态',
          603,
          64,
          14,
          '#115E59',
          850,
          'left'
        );

        drawText(
          '日期',
          603,
          94,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          monthDay(day),
          603,
          115,
          14,
          '#0F766E',
          850,
          'left'
        );

        drawText(
          '太阳直射',
          603,
          145,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          latitudeText(declination),
          603,
          166,
          14,
          '#F97316',
          850,
          'left'
        );

        drawText(
          '季节',
          603,
          196,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          seasonName(day),
          603,
          217,
          13,
          '#0F766E',
          850,
          'left'
        );

        fillRoundedRect(
          585,
          251,
          158,
          137,
          15,
          '#FFFFFF',
          '#CBD5E1'
        );

        drawText(
          '移动规律',
          603,
          274,
          13,
          '#334155',
          850,
          'left'
        );

        var directionText =
          Math.abs(equatorialShift)<1
            ? '接近平均位置'
            : equatorialShift>0
              ? '整体偏向北半球'
              : '整体偏向南半球';

        drawText(
          directionText,
          603,
          302,
          11.5,
          '#0F766E',
          800,
          'left'
        );

        drawText(
          '赤道低压带偏移',
          603,
          330,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          Math.abs(
            equatorialShift
          ).toFixed(1)+'°',
          603,
          354,
          18,
          '#0E7490',
          900,
          'left'
        );

        drawText(
          '气压带和风带位置、宽度及移动幅度均为教学示意，不代表真实风场。',
          390,
          426,
          10,
          '#64748B',
          650,
          'center'
        );

        result.textContent =
          monthDay(day)+
          '前后，太阳直射点位于'+
          latitudeText(declination)+
          '。气压带和风带'+
          directionText+
          '；赤道低压带移动较明显，副热带高压带和副极地低压带随之发生较小幅度移动。';
      }

      function stopAuto(){
        state.auto=false;

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

        dayInput.value =
          String(
            1+
            Math.floor(
              (
                elapsed/35
              )%365
            )
          );

        state.particlePhase =
          (
            elapsed/3600
          )%1;

        render();

        state.raf =
          requestAnimationFrame(
            animate
          );
      }

      dayInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      shiftInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      Array.prototype.forEach.call(
        seasonButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto)stopAuto();

              dayInput.value =
                button.getAttribute(
                  'data-season-day'
                ) || '80';

              state.particlePhase=0;
              render();
            }
          );
        }
      );

      pressureToggle.addEventListener(
        'click',
        function(){
          state.showPressure =
            !state.showPressure;

          render();
        }
      );

      windToggle.addEventListener(
        'click',
        function(){
          state.showWinds =
            !state.showWinds;

          render();
        }
      );

      cellToggle.addEventListener(
        'click',
        function(){
          state.showCells =
            !state.showCells;

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
          state.particlePhase=0;

          render();

          state.raf =
            requestAnimationFrame(
              animate
            );
        }
      );

      pauseMarker.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
          }else{
            render();
          }
        }
      );

      resetButton.addEventListener(
        'click',
        function(){
          stopAuto();

          dayInput.value =
            String(initialDay);

          shiftInput.value =
            String(initialShiftScale);

          state.showPressure =
            ${showPressure};

          state.showWinds =
            ${showWinds};

          state.showCells =
            ${showCells};

          state.particlePhase=0;

          render();
        }
      );

      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_PRESSURE_WIND:
GeographyLabTemplate[] = [
  {
    id: 'geography-pressure-wind-belts-seasonal-shift',
    group: '🌦️ 大气运动与天气系统',
    name: '气压带、风带与季节移动',
    emoji: '🌐',
    desc: '调节日期和移动幅度，观察全球气压带、风带及三圈环流的季节变化。',
    params: [
      {
        key: 'dayOfYear',
        label: '初始日期序号',
        type: 'number',
        min: 1,
        max: 365,
        step: 1,
        defaultValue: 172,
        hint: '第172天约对应6月21日前后。',
      },
      {
        key: 'shiftScale',
        label: '季节移动幅度',
        type: 'number',
        min: 0.5,
        max: 1.5,
        step: 0.1,
        defaultValue: 1,
        hint: '只改变教学图中南北移动的表现幅度。',
      },
      {
        key: 'showPressure',
        label: '显示气压带',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showWinds',
        label: '显示风带',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showCells',
        label: '显示三圈环流',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildPressureWindBeltsHTML,
  },
]
