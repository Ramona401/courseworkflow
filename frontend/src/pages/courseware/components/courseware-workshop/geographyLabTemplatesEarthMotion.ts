/**
 * geographyLabTemplatesEarthMotion.ts
 *
 * 地理第33批：
 *   地球自转、公转与昼夜长短变化。
 *
 * 教学边界：
 *   - 地球、太阳和轨道比例均为教学示意；
 *   - 日期、太阳直射点和昼长使用简化天文模型；
 *   - 不用于精密天文计算、导航或真实日出日落预报。
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

function buildEarthMotionHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const latitude = Math.max(
    -80,
    Math.min(
      80,
      numberValue(params, 'latitude', 40),
    ),
  )

  const dayOfYear = Math.max(
    1,
    Math.min(
      365,
      numberValue(params, 'dayOfYear', 172),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  const requestedMode = stringValue(
    params,
    'motionMode',
    'revolution',
  )

  const motionMode = [
    'rotation',
    'revolution',
  ].includes(requestedMode)
    ? requestedMode
    : 'revolution'

  return `
<div id="${rootId}" class="gl-earth-motion-root">
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
        circle at 48% 32%,
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
      background:#BFDBFE;
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

    #${rootId} .gl-motion-canvas{
      width:100%;
      height:100%;
      display:block;
    }
  </style>

  <div class="gl-head">
    <div style="font-size:23px;">🌍</div>

    <div>
      <div class="gl-title">
        地球自转、公转与昼夜长短变化
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        调节纬度和日期，观察太阳直射点、昼长与地球运动
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 非精密天文计算
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">观察纬度</span>

          <span
            class="gl-value"
            data-role="latitude-value"
          >
            40°N
          </span>
        </div>

        <input
          type="range"
          min="-80"
          max="80"
          step="1"
          value="${latitude}"
          data-role="latitude"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">日期</span>

          <span
            class="gl-value"
            data-role="date-value"
          >
            第172天
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

      <div class="gl-button-grid">
        <button
          type="button"
          data-mode="rotation"
        >
          自转模式
        </button>

        <button
          type="button"
          data-mode="revolution"
        >
          公转模式
        </button>

        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          显示标注
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

        <button
          type="button"
          data-role="season-next"
        >
          下个节气点
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      >
        调节纬度和日期，观察昼夜长短变化。
      </div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-motion-canvas"
        width="780"
        height="440"
        data-role="canvas"
        aria-label="地球自转、公转与昼夜长短教学示意图"
      ></canvas>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var latitudeInput =
        root.querySelector('[data-role="latitude"]');

      var dayInput =
        root.querySelector('[data-role="day"]');

      var latitudeValue =
        root.querySelector('[data-role="latitude-value"]');

      var dateValue =
        root.querySelector('[data-role="date-value"]');

      var modeButtons =
        root.querySelectorAll('[data-mode]');

      var labelToggle =
        root.querySelector('[data-role="label-toggle"]');

      var autoToggle =
        root.querySelector('[data-role="auto-toggle"]');

      var resetButton =
        root.querySelector('[data-role="reset"]');

      var seasonNextButton =
        root.querySelector('[data-role="season-next"]');

      var result =
        root.querySelector('[data-role="result"]');

      var canvas =
        root.querySelector('[data-role="canvas"]');

      if(
        !latitudeInput ||
        !dayInput ||
        !latitudeValue ||
        !dateValue ||
        !modeButtons.length ||
        !labelToggle ||
        !autoToggle ||
        !resetButton ||
        !seasonNextButton ||
        !result ||
        !canvas
      ){
        return;
      }

      var context =
        canvas.getContext('2d');

      if(!context)return;

      var initialLatitude = ${latitude};
      var initialDay = ${dayOfYear};

      var state = {
        mode: '${motionMode}',
        showLabels: ${showLabels},
        auto: false,
        startedAt: 0,
        raf: 0,
        rotationPhase: 0
      };

      var width = canvas.width;
      var height = canvas.height;

      function clamp(value,min,max){
        return Math.max(
          min,
          Math.min(max,value)
        );
      }

      function toRadians(degrees){
        return degrees*Math.PI/180;
      }

      function solarDeclination(day){
        return 23.44*Math.sin(
          2*Math.PI*(284+day)/365
        );
      }

      function dayLengthHours(latitude,declination){
        var latitudeRad =
          toRadians(latitude);

        var declinationRad =
          toRadians(declination);

        var cosineHourAngle =
          -Math.tan(latitudeRad)*
          Math.tan(declinationRad);

        if(cosineHourAngle<=-1){
          return 24;
        }

        if(cosineHourAngle>=1){
          return 0;
        }

        var hourAngle =
          Math.acos(cosineHourAngle);

        return 24*hourAngle/Math.PI;
      }

      function formatLatitude(value){
        var rounded =
          Math.round(Math.abs(value));

        if(Math.abs(value)<0.5){
          return '0°';
        }

        return rounded+
          '°'+
          (
            value>0
              ? 'N'
              : 'S'
          );
      }

      function monthDay(day){
        var monthLengths = [
          31,28,31,30,31,30,
          31,31,30,31,30,31
        ];

        var month = 0;
        var remaining =
          Math.round(day);

        while(
          month<monthLengths.length-1 &&
          remaining>monthLengths[month]
        ){
          remaining -=
            monthLengths[month];

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

      function directPointText(declination){
        var absolute =
          Math.abs(declination);

        if(absolute<0.3){
          return '赤道附近';
        }

        return absolute.toFixed(1)+
          '°'+
          (
            declination>0
              ? 'N'
              : 'S'
          );
      }

      function drawRoundedRect(
        x,
        y,
        w,
        h,
        radius,
        fill,
        stroke
      ){
        context.beginPath();
        context.roundRect(
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

      function drawSun(x,y,radius){
        context.save();

        var glow =
          context.createRadialGradient(
            x,
            y,
            radius*0.2,
            x,
            y,
            radius*1.8
          );

        glow.addColorStop(
          0,
          'rgba(253,224,71,0.95)'
        );

        glow.addColorStop(
          0.45,
          'rgba(251,191,36,0.65)'
        );

        glow.addColorStop(
          1,
          'rgba(251,191,36,0)'
        );

        context.fillStyle=glow;
        context.beginPath();
        context.arc(
          x,
          y,
          radius*1.8,
          0,
          Math.PI*2
        );
        context.fill();

        context.fillStyle='#FBBF24';
        context.strokeStyle='#F59E0B';
        context.lineWidth=3;
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
      }

      function drawEarth(
        x,
        y,
        radius,
        declination,
        phase,
        detailed
      ){
        context.save();

        context.beginPath();
        context.arc(
          x,
          y,
          radius,
          0,
          Math.PI*2
        );
        context.clip();

        context.fillStyle='#0F172A';
        context.fillRect(
          x-radius,
          y-radius,
          radius*2,
          radius*2
        );

        var lightShift =
          Math.cos(phase)*radius*0.32;

        var daylight =
          context.createLinearGradient(
            x-radius+lightShift,
            y,
            x+radius+lightShift,
            y
          );

        daylight.addColorStop(
          0,
          '#0F172A'
        );

        daylight.addColorStop(
          0.42,
          '#1E3A8A'
        );

        daylight.addColorStop(
          0.50,
          '#38BDF8'
        );

        daylight.addColorStop(
          1,
          '#DBEAFE'
        );

        context.fillStyle=daylight;
        context.fillRect(
          x-radius,
          y-radius,
          radius*2,
          radius*2
        );

        context.fillStyle='rgba(34,197,94,0.78)';

        context.beginPath();
        context.ellipse(
          x-radius*0.18,
          y-radius*0.22,
          radius*0.38,
          radius*0.20,
          -0.35,
          0,
          Math.PI*2
        );
        context.fill();

        context.beginPath();
        context.ellipse(
          x+radius*0.24,
          y+radius*0.25,
          radius*0.29,
          radius*0.17,
          0.42,
          0,
          Math.PI*2
        );
        context.fill();

        context.restore();

        context.strokeStyle='#0369A1';
        context.lineWidth=2.5;
        context.beginPath();
        context.arc(
          x,
          y,
          radius,
          0,
          Math.PI*2
        );
        context.stroke();

        var tilt =
          toRadians(-23.44);

        var axisLength =
          radius*1.26;

        context.strokeStyle='#DC2626';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(
          x-Math.sin(tilt)*axisLength,
          y-Math.cos(tilt)*axisLength
        );

        context.lineTo(
          x+Math.sin(tilt)*axisLength,
          y+Math.cos(tilt)*axisLength
        );

        context.stroke();

        if(detailed){
          var directY =
            y-
            (
              declination/23.44
            )*
            radius*0.68;

          context.strokeStyle='#F97316';
          context.lineWidth=2;
          context.setLineDash([6,4]);

          context.beginPath();
          context.moveTo(
            x-radius*0.82,
            directY
          );

          context.lineTo(
            x+radius*0.82,
            directY
          );

          context.stroke();
          context.setLineDash([]);

          if(state.showLabels){
            drawText(
              '太阳直射纬线',
              x+radius+10,
              directY,
              10.5,
              '#C2410C',
              800,
              'left'
            );
          }
        }
      }

      function drawOrbitScene(
        day,
        declination
      ){
        var sunX = 226;
        var sunY = 204;

        drawRoundedRect(
          18,
          20,
          500,
          286,
          18,
          '#F8FAFC',
          '#BAE6FD'
        );

        context.strokeStyle='#94A3B8';
        context.lineWidth=1.5;
        context.setLineDash([7,6]);

        context.beginPath();
        context.ellipse(
          sunX,
          sunY,
          190,
          112,
          0,
          0,
          Math.PI*2
        );
        context.stroke();
        context.setLineDash([]);

        drawSun(
          sunX,
          sunY,
          37
        );

        var orbitAngle =
          (
            day-80
          )/365*
          Math.PI*2;

        var earthX =
          sunX+
          Math.cos(orbitAngle)*190;

        var earthY =
          sunY+
          Math.sin(orbitAngle)*112;

        drawEarth(
          earthX,
          earthY,
          25,
          declination,
          state.rotationPhase,
          false
        );

        context.strokeStyle=
          'rgba(249,115,22,0.55)';

        context.lineWidth=1.5;

        context.beginPath();
        context.moveTo(
          sunX,
          sunY
        );

        context.lineTo(
          earthX,
          earthY
        );

        context.stroke();

        if(state.showLabels){
          drawText(
            '太阳',
            sunX,
            sunY+57,
            11,
            '#B45309',
            850,
            'center'
          );

          drawText(
            monthDay(day),
            earthX,
            earthY-42,
            11,
            '#0F766E',
            850,
            'center'
          );

          drawText(
            '公转方向',
            sunX,
            72,
            11,
            '#64748B',
            750,
            'center'
          );

          drawText(
            '轨道与天体大小均为教学示意',
            sunX,
            288,
            10,
            '#64748B',
            650,
            'center'
          );
        }

        context.strokeStyle='#0EA5E9';
        context.lineWidth=2;

        context.beginPath();
        context.arc(
          sunX,
          sunY,
          142,
          Math.PI*1.16,
          Math.PI*1.52
        );
        context.stroke();
      }

      function drawRotationScene(
        latitude,
        declination
      ){
        drawRoundedRect(
          18,
          20,
          500,
          286,
          18,
          '#F8FAFC',
          '#BAE6FD'
        );

        drawSun(
          87,
          96,
          31
        );

        for(
          var ray=0;
          ray<5;
          ray+=1
        ){
          context.strokeStyle=
            'rgba(245,158,11,0.52)';

          context.lineWidth=2;

          context.beginPath();

          context.moveTo(
            124,
            70+ray*30
          );

          context.lineTo(
            268,
            70+ray*30
          );

          context.stroke();
        }

        drawEarth(
          360,
          160,
          101,
          declination,
          state.rotationPhase,
          true
        );

        var latitudeY =
          160-
          (
            latitude/90
          )*
          81;

        context.strokeStyle='#A855F7';
        context.lineWidth=2;
        context.setLineDash([5,4]);

        context.beginPath();
        context.moveTo(
          280,
          latitudeY
        );

        context.lineTo(
          440,
          latitudeY
        );

        context.stroke();
        context.setLineDash([]);

        context.fillStyle='#A855F7';
        context.beginPath();
        context.arc(
          438,
          latitudeY,
          5,
          0,
          Math.PI*2
        );
        context.fill();

        if(state.showLabels){
          drawText(
            '太阳光线',
            176,
            55,
            11,
            '#B45309',
            800,
            'center'
          );

          drawText(
            formatLatitude(latitude),
            451,
            latitudeY,
            11,
            '#7E22CE',
            850,
            'left'
          );

          drawText(
            '自转方向：自西向东',
            360,
            284,
            11,
            '#0369A1',
            800,
            'center'
          );
        }

        context.strokeStyle='#0284C7';
        context.lineWidth=2.4;

        context.beginPath();
        context.arc(
          360,
          160,
          122,
          Math.PI*0.12,
          Math.PI*0.62
        );
        context.stroke();
      }

      function drawDayLengthChart(
        latitude,
        currentDay
      ){
        drawRoundedRect(
          536,
          20,
          226,
          286,
          18,
          '#FFFFFF',
          '#99F6E4'
        );

        drawText(
          '全年昼长变化',
          556,
          45,
          14,
          '#115E59',
          850,
          'left'
        );

        var left = 558;
        var top = 76;
        var chartWidth = 180;
        var chartHeight = 145;

        context.strokeStyle='#CBD5E1';
        context.lineWidth=1;

        for(
          var line=0;
          line<=4;
          line+=1
        ){
          var y =
            top+
            chartHeight-
            (
              line/4
            )*
            chartHeight;

          context.beginPath();
          context.moveTo(
            left,
            y
          );

          context.lineTo(
            left+chartWidth,
            y
          );

          context.stroke();

          if(state.showLabels){
            drawText(
              String(line*6)+'h',
              left-7,
              y,
              9.5,
              '#64748B',
              650,
              'right'
            );
          }
        }

        context.strokeStyle='#0F766E';
        context.lineWidth=2.5;
        context.beginPath();

        for(
          var day=1;
          day<=365;
          day+=4
        ){
          var declination =
            solarDeclination(day);

          var length =
            dayLengthHours(
              latitude,
              declination
            );

          var x =
            left+
            (
              day-1
            )/364*
            chartWidth;

          var y =
            top+
            chartHeight-
            length/24*
            chartHeight;

          if(day===1){
            context.moveTo(
              x,
              y
            );
          }else{
            context.lineTo(
              x,
              y
            );
          }
        }

        context.stroke();

        var currentDeclination =
          solarDeclination(currentDay);

        var currentLength =
          dayLengthHours(
            latitude,
            currentDeclination
          );

        var currentX =
          left+
          (
            currentDay-1
          )/364*
          chartWidth;

        var currentY =
          top+
          chartHeight-
          currentLength/24*
          chartHeight;

        context.fillStyle='#F97316';
        context.beginPath();
        context.arc(
          currentX,
          currentY,
          5,
          0,
          Math.PI*2
        );
        context.fill();

        if(state.showLabels){
          drawText(
            '1月',
            left,
            top+chartHeight+17,
            9.5,
            '#64748B',
            650,
            'center'
          );

          drawText(
            '7月',
            left+chartWidth*0.5,
            top+chartHeight+17,
            9.5,
            '#64748B',
            650,
            'center'
          );

          drawText(
            '12月',
            left+chartWidth,
            top+chartHeight+17,
            9.5,
            '#64748B',
            650,
            'center'
          );
        }

        drawText(
          '当前昼长',
          556,
          252,
          10.5,
          '#64748B',
          700,
          'left'
        );

        drawText(
          currentLength.toFixed(1)+'小时',
          556,
          277,
          18,
          '#0F766E',
          850,
          'left'
        );
      }

      function drawInformationCards(
        declination,
        dayLength,
        day
      ){
        drawRoundedRect(
          18,
          324,
          744,
          91,
          16,
          '#FFFFFF',
          '#CBD5E1'
        );

        var cards = [
          {
            title:'太阳直射点',
            value:directPointText(declination)
          },
          {
            title:'观察地昼长',
            value:dayLength.toFixed(1)+'小时'
          },
          {
            title:'季节判断',
            value:seasonName(day)
          },
          {
            title:'日期',
            value:monthDay(day)
          }
        ];

        cards.forEach(
          function(card,index){
            var x =
              38+
              index*181;

            if(index>0){
              context.strokeStyle='#E2E8F0';
              context.lineWidth=1;

              context.beginPath();
              context.moveTo(
                x-13,
                340
              );

              context.lineTo(
                x-13,
                400
              );

              context.stroke();
            }

            drawText(
              card.title,
              x,
              350,
              10.5,
              '#64748B',
              700,
              'left'
            );

            drawText(
              card.value,
              x,
              381,
              15,
              '#0F766E',
              850,
              'left'
            );
          }
        );

        drawText(
          '模型中的轨道、天体比例和日期关系均为教学示意，不用于导航或真实日出日落预报。',
          390,
          430,
          10,
          '#64748B',
          650,
          'center'
        );
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var latitude =
          clamp(
            parseFloat(
              latitudeInput.value
            ) || 0,
            -80,
            80
          );

        var day =
          clamp(
            parseFloat(
              dayInput.value
            ) || 1,
            1,
            365
          );

        var declination =
          solarDeclination(day);

        var dayLength =
          dayLengthHours(
            latitude,
            declination
          );

        latitudeValue.textContent =
          formatLatitude(latitude);

        dateValue.textContent =
          monthDay(day);

        Array.prototype.forEach.call(
          modeButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-mode'
              )===state.mode
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

        context.clearRect(
          0,
          0,
          width,
          height
        );

        context.fillStyle='#F8FAFC';
        context.fillRect(
          0,
          0,
          width,
          height
        );

        if(state.mode==='rotation'){
          drawRotationScene(
            latitude,
            declination
          );
        }else{
          drawOrbitScene(
            day,
            declination
          );
        }

        drawDayLengthChart(
          latitude,
          day
        );

        drawInformationCards(
          declination,
          dayLength,
          day
        );

        var hemisphere =
          latitude>=0
            ? '北半球'
            : '南半球';

        var comparison =
          dayLength>12.2
            ? '昼长夜短'
            : dayLength<11.8
              ? '昼短夜长'
              : '昼夜接近等长';

        result.textContent =
          monthDay(day)+
          '，太阳直射约在'+
          directPointText(declination)+
          '。'+
          formatLatitude(latitude)+
          '位于'+
          hemisphere+
          '，昼长约'+
          dayLength.toFixed(1)+
          '小时，表现为'+
          comparison+
          '。';
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

        if(state.mode==='revolution'){
          var day =
            1+
            Math.floor(
              (
                elapsed/38
              )%365
            );

          dayInput.value =
            String(day);
        }else{
          state.rotationPhase =
            elapsed/900;
        }

        render();

        state.raf =
          requestAnimationFrame(
            animate
          );
      }

      latitudeInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      dayInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      Array.prototype.forEach.call(
        modeButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto)stopAuto();

              state.mode =
                button.getAttribute(
                  'data-mode'
                ) ||
                'revolution';

              render();
            }
          );
        }
      );

      labelToggle.addEventListener(
        'click',
        function(){
          state.showLabels =
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

          state.raf =
            requestAnimationFrame(
              animate
            );
        }
      );

      seasonNextButton.addEventListener(
        'click',
        function(){
          if(state.auto)stopAuto();

          var currentDay =
            parseFloat(
              dayInput.value
            ) || 1;

          var seasonalDays = [
            80,
            172,
            266,
            355
          ];

          var nextDay =
            seasonalDays.find(
              function(value){
                return value>
                  currentDay+1;
              }
            ) ||
            seasonalDays[0];

          dayInput.value =
            String(nextDay);

          render();
        }
      );

      resetButton.addEventListener(
        'click',
        function(){
          stopAuto();

          latitudeInput.value =
            String(initialLatitude);

          dayInput.value =
            String(initialDay);

          state.mode =
            '${motionMode}';

          state.showLabels =
            ${showLabels};

          state.rotationPhase=0;

          render();
        }
      );

      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_EARTH_MOTION:
GeographyLabTemplate[] = [
  {
    id: 'geography-earth-rotation-revolution-daylength',
    group: '🧭 基础定位与地球运动',
    name: '地球自转、公转与昼夜长短变化',
    emoji: '🌍',
    desc: '调节纬度和日期，切换自转、公转模式，观察太阳直射点和昼夜长短变化。',
    params: [
      {
        key: 'latitude',
        label: '初始观察纬度',
        type: 'number',
        min: -80,
        max: 80,
        step: 1,
        defaultValue: 40,
        hint: '正值表示北纬，负值表示南纬。',
      },
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
        key: 'motionMode',
        label: '初始观察模式',
        type: 'select',
        options: [
          {
            label: '公转与季节',
            value: 'revolution',
          },
          {
            label: '自转与昼夜',
            value: 'rotation',
          },
        ],
        defaultValue: 'revolution',
      },
      {
        key: 'showLabels',
        label: '显示教学标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildEarthMotionHTML,
  },
]
