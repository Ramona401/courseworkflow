/**
 * geographyLabTemplatesAtmosphereThermalCirculation.ts
 *
 * 地理第34批：
 *   热力环流与海陆风。
 *
 * 教学边界：
 *   - 温度、气压和气流速度均为课堂示意；
 *   - 忽略真实地形、摩擦、湿度和天气尺度差异；
 *   - 不用于天气预报、航空判断或灾害决策。
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

function buildThermalCirculationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const temperatureContrast = Math.max(
    2,
    Math.min(
      12,
      numberValue(params, 'temperatureContrast', 6),
    ),
  )

  const requestedPeriod = stringValue(
    params,
    'timePeriod',
    'day',
  )

  const timePeriod = [
    'day',
    'night',
  ].includes(requestedPeriod)
    ? requestedPeriod
    : 'day'

  const showPressure = booleanValue(
    params,
    'showPressure',
    true,
  )

  const showArrows = booleanValue(
    params,
    'showArrows',
    true,
  )

  return `
<div id="${rootId}" class="gl-thermal-circulation-root">
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
      background:linear-gradient(135deg,#E0F2FE,#CCFBF1);
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
        circle at 48% 24%,
        #FFFFFF 0%,
        #F8FAFC 58%,
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
    <div style="font-size:23px;">🌬️</div>

    <div>
      <div class="gl-title">
        热力环流与海陆风
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        比较海陆受热差异，观察气压与空气运动方向
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 非实时天气预报
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            海陆温差
          </span>

          <span
            class="gl-value"
            data-role="contrast-value"
          >
            6℃
          </span>
        </div>

        <input
          type="range"
          min="2"
          max="12"
          step="1"
          value="${temperatureContrast}"
          data-role="contrast"
        />
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-period="day"
        >
          白天海风
        </button>

        <button
          type="button"
          data-period="night"
        >
          夜间陆风
        </button>

        <button
          type="button"
          data-role="pressure-toggle"
          data-active="${showPressure}"
        >
          气压标注
        </button>

        <button
          type="button"
          data-role="arrow-toggle"
          data-active="${showArrows}"
        >
          环流箭头
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
      </div>

      <div
        class="gl-result"
        data-role="result"
      >
        海陆受热差异会造成近地面气压差，并形成局地热力环流。
      </div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-atmosphere-canvas"
        width="780"
        height="440"
        data-role="canvas"
        aria-label="热力环流与海陆风教学示意图"
      ></canvas>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var contrastInput =
        root.querySelector('[data-role="contrast"]');

      var contrastValue =
        root.querySelector('[data-role="contrast-value"]');

      var periodButtons =
        root.querySelectorAll('[data-period]');

      var pressureToggle =
        root.querySelector('[data-role="pressure-toggle"]');

      var arrowToggle =
        root.querySelector('[data-role="arrow-toggle"]');

      var autoToggle =
        root.querySelector('[data-role="auto-toggle"]');

      var resetButton =
        root.querySelector('[data-role="reset"]');

      var result =
        root.querySelector('[data-role="result"]');

      var canvas =
        root.querySelector('[data-role="canvas"]');

      if(
        !contrastInput ||
        !contrastValue ||
        !periodButtons.length ||
        !pressureToggle ||
        !arrowToggle ||
        !autoToggle ||
        !resetButton ||
        !result ||
        !canvas
      ){
        return;
      }

      var context =
        canvas.getContext('2d');

      if(!context)return;

      var initialContrast =
        ${temperatureContrast};

      var state = {
        period:'${timePeriod}',
        showPressure:${showPressure},
        showArrows:${showArrows},
        auto:false,
        startedAt:0,
        animationPhase:0,
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

      function roundedRect(
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

        context.textBaseline =
          'middle';

        context.fillText(
          text,
          x,
          y
        );

        context.restore();
      }

      function drawArrow(
        x1,
        y1,
        x2,
        y2,
        color,
        widthValue
      ){
        var angle =
          Math.atan2(
            y2-y1,
            x2-x1
          );

        var headLength =
          12;

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=
          widthValue || 3;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(x1,y1);
        context.lineTo(x2,y2);
        context.stroke();

        context.beginPath();
        context.moveTo(x2,y2);

        context.lineTo(
          x2-
          headLength*
          Math.cos(
            angle-Math.PI/6
          ),
          y2-
          headLength*
          Math.sin(
            angle-Math.PI/6
          )
        );

        context.lineTo(
          x2-
          headLength*
          Math.cos(
            angle+Math.PI/6
          ),
          y2-
          headLength*
          Math.sin(
            angle+Math.PI/6
          )
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
        context.lineWidth=3;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(
          startX,
          startY
        );

        context.quadraticCurveTo(
          controlX,
          controlY,
          endX,
          endY
        );

        context.stroke();

        var tangentAngle =
          Math.atan2(
            endY-controlY,
            endX-controlX
          );

        var headLength=12;

        context.beginPath();
        context.moveTo(
          endX,
          endY
        );

        context.lineTo(
          endX-
          headLength*
          Math.cos(
            tangentAngle-Math.PI/6
          ),
          endY-
          headLength*
          Math.sin(
            tangentAngle-Math.PI/6
          )
        );

        context.lineTo(
          endX-
          headLength*
          Math.cos(
            tangentAngle+Math.PI/6
          ),
          endY-
          headLength*
          Math.sin(
            tangentAngle+Math.PI/6
          )
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function drawSun(){
        context.save();

        var glow =
          context.createRadialGradient(
            88,
            68,
            8,
            88,
            68,
            52
          );

        glow.addColorStop(
          0,
          'rgba(253,224,71,0.95)'
        );

        glow.addColorStop(
          0.48,
          'rgba(251,191,36,0.54)'
        );

        glow.addColorStop(
          1,
          'rgba(251,191,36,0)'
        );

        context.fillStyle=glow;
        context.beginPath();
        context.arc(
          88,
          68,
          52,
          0,
          Math.PI*2
        );
        context.fill();

        context.fillStyle='#FBBF24';
        context.strokeStyle='#F59E0B';
        context.lineWidth=2.5;

        context.beginPath();
        context.arc(
          88,
          68,
          23,
          0,
          Math.PI*2
        );
        context.fill();
        context.stroke();

        context.restore();
      }

      function drawMoon(){
        context.save();

        context.fillStyle='#E2E8F0';
        context.beginPath();
        context.arc(
          88,
          68,
          23,
          0,
          Math.PI*2
        );
        context.fill();

        context.fillStyle='#0F2747';
        context.beginPath();
        context.arc(
          99,
          59,
          23,
          0,
          Math.PI*2
        );
        context.fill();

        context.restore();
      }

      function drawCloud(
        x,
        y,
        scale,
        opacity
      ){
        context.save();
        context.globalAlpha=opacity;
        context.fillStyle='#FFFFFF';

        context.beginPath();
        context.arc(
          x,
          y,
          16*scale,
          0,
          Math.PI*2
        );

        context.arc(
          x+22*scale,
          y-8*scale,
          20*scale,
          0,
          Math.PI*2
        );

        context.arc(
          x+48*scale,
          y,
          16*scale,
          0,
          Math.PI*2
        );

        context.fillRect(
          x,
          y,
          48*scale,
          18*scale
        );

        context.fill();
        context.restore();
      }

      function drawParticle(
        progress,
        period,
        color
      ){
        var points;

        if(period==='day'){
          points = [
            {x:220,y:332},
            {x:520,y:332},
            {x:548,y:145},
            {x:236,y:145},
            {x:220,y:332}
          ];
        }else{
          points = [
            {x:548,y:332},
            {x:236,y:332},
            {x:220,y:145},
            {x:532,y:145},
            {x:548,y:332}
          ];
        }

        var segmentCount =
          points.length-1;

        var scaled =
          (
            progress%1
          )*
          segmentCount;

        var segment =
          Math.min(
            segmentCount-1,
            Math.floor(scaled)
          );

        var local =
          scaled-segment;

        var start =
          points[segment];

        var end =
          points[segment+1];

        var x =
          start.x+
          (
            end.x-start.x
          )*
          local;

        var y =
          start.y+
          (
            end.y-start.y
          )*
          local;

        context.save();
        context.fillStyle=color;
        context.shadowColor=color;
        context.shadowBlur=8;

        context.beginPath();
        context.arc(
          x,
          y,
          5,
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

        var contrast =
          clamp(
            parseFloat(
              contrastInput.value
            ) || 6,
            2,
            12
          );

        var isDay =
          state.period==='day';

        var seaTemperature =
          isDay
            ? 24
            : 21;

        var landTemperature =
          isDay
            ? seaTemperature+contrast
            : seaTemperature-contrast;

        var warmSide =
          isDay
            ? '陆地'
            : '海洋';

        var coolSide =
          isDay
            ? '海洋'
            : '陆地';

        contrastValue.textContent =
          Math.round(contrast)+'℃';

        Array.prototype.forEach.call(
          periodButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-period'
              )===state.period
                ? 'true'
                : 'false'
            );
          }
        );

        pressureToggle.setAttribute(
          'data-active',
          state.showPressure
            ? 'true'
            : 'false'
        );

        arrowToggle.setAttribute(
          'data-active',
          state.showArrows
            ? 'true'
            : 'false'
        );

        autoToggle.setAttribute(
          'data-active',
          state.auto
            ? 'true'
            : 'false'
        );

        var skyGradient =
          context.createLinearGradient(
            0,
            0,
            0,
            height
          );

        if(isDay){
          skyGradient.addColorStop(
            0,
            '#BAE6FD'
          );

          skyGradient.addColorStop(
            0.68,
            '#E0F2FE'
          );

          skyGradient.addColorStop(
            1,
            '#F8FAFC'
          );
        }else{
          skyGradient.addColorStop(
            0,
            '#0F2747'
          );

          skyGradient.addColorStop(
            0.68,
            '#1E3A5F'
          );

          skyGradient.addColorStop(
            1,
            '#CBD5E1'
          );
        }

        context.fillStyle=skyGradient;
        context.fillRect(
          0,
          0,
          width,
          height
        );

        if(isDay){
          drawSun();
          drawCloud(
            190,
            65,
            0.8,
            0.78
          );

          drawCloud(
            600,
            88,
            0.65,
            0.72
          );
        }else{
          drawMoon();

          context.fillStyle=
            'rgba(255,255,255,0.78)';

          [
            [170,42],
            [245,82],
            [650,48],
            [714,100],
            [545,55]
          ].forEach(
            function(point){
              context.beginPath();
              context.arc(
                point[0],
                point[1],
                1.8,
                0,
                Math.PI*2
              );
              context.fill();
            }
          );
        }

        fillRoundedRect(
          18,
          118,
          744,
          274,
          18,
          isDay
            ? 'rgba(255,255,255,0.72)'
            : 'rgba(15,23,42,0.52)',
          isDay
            ? '#BAE6FD'
            : '#64748B'
        );

        var seaGradient =
          context.createLinearGradient(
            0,
            286,
            0,
            392
          );

        seaGradient.addColorStop(
          0,
          '#38BDF8'
        );

        seaGradient.addColorStop(
          1,
          '#0369A1'
        );

        context.fillStyle=seaGradient;
        context.fillRect(
          18,
          286,
          360,
          106
        );

        var landGradient =
          context.createLinearGradient(
            0,
            286,
            0,
            392
          );

        landGradient.addColorStop(
          0,
          isDay
            ? '#FCD34D'
            : '#94A3B8'
        );

        landGradient.addColorStop(
          1,
          isDay
            ? '#B45309'
            : '#475569'
        );

        context.fillStyle=landGradient;
        context.fillRect(
          378,
          286,
          384,
          106
        );

        context.strokeStyle=
          'rgba(255,255,255,0.65)';

        context.lineWidth=1.5;

        for(
          var wave=0;
          wave<6;
          wave+=1
        ){
          var waveY =
            310+wave*13;

          context.beginPath();

          context.moveTo(
            45,
            waveY
          );

          context.bezierCurveTo(
            110,
            waveY-5,
            180,
            waveY+5,
            245,
            waveY
          );

          context.bezierCurveTo(
            285,
            waveY-4,
            325,
            waveY+4,
            360,
            waveY
          );

          context.stroke();
        }

        context.fillStyle=
          isDay
            ? '#365314'
            : '#334155';

        context.beginPath();
        context.moveTo(
          378,
          286
        );

        context.lineTo(
          425,
          255
        );

        context.lineTo(
          470,
          286
        );

        context.lineTo(
          520,
          245
        );

        context.lineTo(
          578,
          286
        );

        context.closePath();
        context.fill();

        drawText(
          '海洋',
          196,
          355,
          16,
          '#FFFFFF',
          850,
          'center'
        );

        drawText(
          '陆地',
          570,
          355,
          16,
          '#FFFFFF',
          850,
          'center'
        );

        var seaHeat =
          context.createLinearGradient(
            180,
            286,
            180,
            142
          );

        seaHeat.addColorStop(
          0,
          isDay
            ? 'rgba(56,189,248,0.28)'
            : 'rgba(249,115,22,0.46)'
        );

        seaHeat.addColorStop(
          1,
          'rgba(255,255,255,0)'
        );

        context.fillStyle=seaHeat;
        context.fillRect(
          125,
          135,
          190,
          151
        );

        var landHeat =
          context.createLinearGradient(
            570,
            286,
            570,
            142
          );

        landHeat.addColorStop(
          0,
          isDay
            ? 'rgba(249,115,22,0.52)'
            : 'rgba(56,189,248,0.28)'
        );

        landHeat.addColorStop(
          1,
          'rgba(255,255,255,0)'
        );

        context.fillStyle=landHeat;
        context.fillRect(
          472,
          135,
          190,
          151
        );

        drawText(
          seaTemperature.toFixed(0)+'℃',
          198,
          260,
          17,
          isDay
            ? '#075985'
            : '#FDBA74',
          850,
          'center'
        );

        drawText(
          landTemperature.toFixed(0)+'℃',
          570,
          260,
          17,
          isDay
            ? '#C2410C'
            : '#BAE6FD',
          850,
          'center'
        );

        if(state.showPressure){
          var seaSurfacePressure =
            isDay
              ? 'H 高压'
              : 'L 低压';

          var landSurfacePressure =
            isDay
              ? 'L 低压'
              : 'H 高压';

          var seaUpperPressure =
            isDay
              ? 'L'
              : 'H';

          var landUpperPressure =
            isDay
              ? 'H'
              : 'L';

          fillRoundedRect(
            145,
            297,
            105,
            30,
            15,
            isDay
              ? '#DBEAFE'
              : '#FEE2E2',
            null
          );

          drawText(
            seaSurfacePressure,
            197,
            312,
            11.5,
            isDay
              ? '#1D4ED8'
              : '#B91C1C',
            850,
            'center'
          );

          fillRoundedRect(
            518,
            297,
            105,
            30,
            15,
            isDay
              ? '#FEE2E2'
              : '#DBEAFE',
            null
          );

          drawText(
            landSurfacePressure,
            570,
            312,
            11.5,
            isDay
              ? '#B91C1C'
              : '#1D4ED8',
            850,
            'center'
          );

          drawText(
            seaUpperPressure,
            220,
            145,
            18,
            seaUpperPressure==='H'
              ? '#1D4ED8'
              : '#B91C1C',
            900,
            'center'
          );

          drawText(
            landUpperPressure,
            548,
            145,
            18,
            landUpperPressure==='H'
              ? '#1D4ED8'
              : '#B91C1C',
            900,
            'center'
          );
        }

        if(state.showArrows){
          var arrowColor =
            isDay
              ? '#0F766E'
              : '#7C3AED';

          if(isDay){
            drawArrow(
              250,
              338,
              500,
              338,
              arrowColor,
              4
            );

            drawCurvedArrow(
              548,
              286,
              572,
              212,
              548,
              160,
              arrowColor
            );

            drawArrow(
              520,
              151,
              255,
              151,
              arrowColor,
              4
            );

            drawCurvedArrow(
              220,
              162,
              196,
              222,
              220,
              286,
              arrowColor
            );
          }else{
            drawArrow(
              518,
              338,
              270,
              338,
              arrowColor,
              4
            );

            drawCurvedArrow(
              220,
              286,
              196,
              220,
              220,
              160,
              arrowColor
            );

            drawArrow(
              250,
              151,
              520,
              151,
              arrowColor,
              4
            );

            drawCurvedArrow(
              548,
              160,
              572,
              220,
              548,
              286,
              arrowColor
            );
          }

          drawParticle(
            state.animationPhase,
            state.period,
            '#F97316'
          );

          drawParticle(
            state.animationPhase+0.33,
            state.period,
            '#0EA5E9'
          );

          drawParticle(
            state.animationPhase+0.66,
            state.period,
            '#A855F7'
          );
        }

        var modeName =
          isDay
            ? '白天海风'
            : '夜间陆风';

        var surfaceDirection =
          isDay
            ? '海洋吹向陆地'
            : '陆地吹向海洋';

        fillRoundedRect(
          35,
          400,
          710,
          28,
          14,
          isDay
            ? 'rgba(255,255,255,0.84)'
            : 'rgba(15,23,42,0.74)',
          null
        );

        drawText(
          modeName+
          '：近地面空气由'+
          coolSide+
          '高压区流向'+
          warmSide+
          '低压区，即'+
          surfaceDirection+'。',
          390,
          414,
          11,
          isDay
            ? '#334155'
            : '#E2E8F0',
          750,
          'center'
        );

        result.textContent =
          modeName+
          '条件下，'+
          warmSide+
          '升温较快，近地面空气受热上升并形成低压；'+
          coolSide+
          '相对较冷，空气下沉并形成高压。近地面风由'+
          coolSide+
          '吹向'+
          warmSide+
          '，高空形成反向补偿气流。';
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

        state.animationPhase =
          (
            elapsed/4400
          )%1;

        var periodIndex =
          Math.floor(
            elapsed/5200
          )%2;

        state.period =
          periodIndex===0
            ? 'day'
            : 'night';

        render();

        state.raf =
          requestAnimationFrame(
            animate
          );
      }

      contrastInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      Array.prototype.forEach.call(
        periodButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto)stopAuto();

              state.period =
                button.getAttribute(
                  'data-period'
                ) || 'day';

              state.animationPhase=0;
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

      arrowToggle.addEventListener(
        'click',
        function(){
          state.showArrows =
            !state.showArrows;

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
          state.animationPhase=0;

          render();

          state.raf =
            requestAnimationFrame(
              animate
            );
        }
      );

      resetButton.addEventListener(
        'click',
        function(){
          stopAuto();

          contrastInput.value =
            String(initialContrast);

          state.period =
            '${timePeriod}';

          state.showPressure =
            ${showPressure};

          state.showArrows =
            ${showArrows};

          state.animationPhase=0;

          render();
        }
      );

      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_THERMAL:
GeographyLabTemplate[] = [
  {
    id: 'geography-thermal-circulation-sea-land-breeze',
    group: '🌦️ 大气运动与天气系统',
    name: '热力环流与海陆风',
    emoji: '🌬️',
    desc: '调节海陆温差，切换白天和夜间，观察气压差、垂直运动与海陆风环流。',
    params: [
      {
        key: 'temperatureContrast',
        label: '初始海陆温差',
        type: 'number',
        min: 2,
        max: 12,
        step: 1,
        defaultValue: 6,
        hint: '温差越大，教学模型中的热力环流越明显。',
      },
      {
        key: 'timePeriod',
        label: '初始时段',
        type: 'select',
        options: [
          {
            label: '白天海风',
            value: 'day',
          },
          {
            label: '夜间陆风',
            value: 'night',
          },
        ],
        defaultValue: 'day',
      },
      {
        key: 'showPressure',
        label: '显示气压标注',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showArrows',
        label: '显示环流箭头',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildThermalCirculationHTML,
  },
]
