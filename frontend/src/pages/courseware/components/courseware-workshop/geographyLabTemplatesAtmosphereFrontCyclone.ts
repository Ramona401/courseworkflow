/**
 * geographyLabTemplatesAtmosphereFrontCyclone.ts
 *
 * 地理第34批：
 *   冷锋、暖锋和气旋天气过程。
 *
 * 教学边界：
 *   - 锋面坡度、云系、降水区和气压变化均为课堂示意；
 *   - 气旋按北半球近地面逆时针辐合模型展示；
 *   - 忽略真实地形、湿度、摩擦和复杂天气系统相互作用；
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

function buildFrontCycloneHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const progress = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'progress', 42),
    ),
  )

  const temperatureContrast = Math.max(
    4,
    Math.min(
      14,
      numberValue(params, 'temperatureContrast', 8),
    ),
  )

  const requestedMode = stringValue(
    params,
    'systemMode',
    'cold-front',
  )

  const systemMode = [
    'cold-front',
    'warm-front',
    'cyclone',
  ].includes(requestedMode)
    ? requestedMode
    : 'cold-front'

  const showPrecipitation = booleanValue(
    params,
    'showPrecipitation',
    true,
  )

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
<div id="${rootId}" class="gl-front-cyclone-root">
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

    #${rootId} .gl-mode-grid{
      display:grid;
      grid-template-columns:1fr;
      gap:7px;
      margin-bottom:10px;
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

    #${rootId} .gl-weather-canvas{
      width:100%;
      height:100%;
      display:block;
    }
  </style>

  <div class="gl-head">
    <div style="font-size:23px;">🌧️</div>

    <div>
      <div class="gl-title">
        冷锋、暖锋和气旋天气过程
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        切换天气系统，观察气团运动、云雨位置和过境前后变化
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
            天气过程进度
          </span>

          <span
            class="gl-value"
            data-role="progress-value"
          >
            42%
          </span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${progress}"
          data-role="progress"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            冷暖气团温差
          </span>

          <span
            class="gl-value"
            data-role="contrast-value"
          >
            8℃
          </span>
        </div>

        <input
          type="range"
          min="4"
          max="14"
          step="1"
          value="${temperatureContrast}"
          data-role="contrast"
        />
      </div>

      <div class="gl-mode-grid">
        <button
          type="button"
          data-system-mode="cold-front"
        >
          🔵 冷锋天气过程
        </button>

        <button
          type="button"
          data-system-mode="warm-front"
        >
          🔴 暖锋天气过程
        </button>

        <button
          type="button"
          data-system-mode="cyclone"
        >
          🌀 气旋天气过程
        </button>
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-role="precipitation-toggle"
          data-active="${showPrecipitation}"
        >
          云系降水
        </button>

        <button
          type="button"
          data-role="pressure-toggle"
          data-active="${showPressure}"
        >
          气压变化
        </button>

        <button
          type="button"
          data-role="arrow-toggle"
          data-active="${showArrows}"
        >
          气流箭头
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
          data-role="next-stage"
        >
          下一阶段
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      >
        选择天气系统并调节进度，观察过境前后天气变化。
      </div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-weather-canvas"
        width="780"
        height="440"
        data-role="canvas"
        aria-label="冷锋暖锋与气旋天气过程教学示意图"
      ></canvas>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var progressInput =
        root.querySelector('[data-role="progress"]');

      var contrastInput =
        root.querySelector('[data-role="contrast"]');

      var progressValue =
        root.querySelector('[data-role="progress-value"]');

      var contrastValue =
        root.querySelector('[data-role="contrast-value"]');

      var modeButtons =
        root.querySelectorAll('[data-system-mode]');

      var precipitationToggle =
        root.querySelector('[data-role="precipitation-toggle"]');

      var pressureToggle =
        root.querySelector('[data-role="pressure-toggle"]');

      var arrowToggle =
        root.querySelector('[data-role="arrow-toggle"]');

      var autoToggle =
        root.querySelector('[data-role="auto-toggle"]');

      var resetButton =
        root.querySelector('[data-role="reset"]');

      var nextStageButton =
        root.querySelector('[data-role="next-stage"]');

      var result =
        root.querySelector('[data-role="result"]');

      var canvas =
        root.querySelector('[data-role="canvas"]');

      if(
        !progressInput ||
        !contrastInput ||
        !progressValue ||
        !contrastValue ||
        !modeButtons.length ||
        !precipitationToggle ||
        !pressureToggle ||
        !arrowToggle ||
        !autoToggle ||
        !resetButton ||
        !nextStageButton ||
        !result ||
        !canvas
      ){
        return;
      }

      var context =
        canvas.getContext('2d');

      if(!context)return;

      var initialProgress =
        ${progress};

      var initialContrast =
        ${temperatureContrast};

      var state = {
        mode:'${systemMode}',
        showPrecipitation:${showPrecipitation},
        showPressure:${showPressure},
        showArrows:${showArrows},
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
        lineWidth
      ){
        var angle =
          Math.atan2(
            y2-y1,
            x2-x1
          );

        var headLength=11;

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=
          lineWidth || 2.8;
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
        context.lineWidth=2.8;
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

        var angle =
          Math.atan2(
            endY-controlY,
            endX-controlX
          );

        var headLength=10;

        context.beginPath();
        context.moveTo(endX,endY);

        context.lineTo(
          endX-
          headLength*
          Math.cos(
            angle-Math.PI/6
          ),
          endY-
          headLength*
          Math.sin(
            angle-Math.PI/6
          )
        );

        context.lineTo(
          endX-
          headLength*
          Math.cos(
            angle+Math.PI/6
          ),
          endY-
          headLength*
          Math.sin(
            angle+Math.PI/6
          )
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function drawCloud(
        x,
        y,
        scale,
        color,
        opacity
      ){
        context.save();
        context.globalAlpha=
          opacity===undefined
            ? 1
            : opacity;

        context.fillStyle=
          color || '#FFFFFF';

        context.beginPath();

        context.arc(
          x,
          y,
          15*scale,
          0,
          Math.PI*2
        );

        context.arc(
          x+21*scale,
          y-8*scale,
          20*scale,
          0,
          Math.PI*2
        );

        context.arc(
          x+47*scale,
          y,
          16*scale,
          0,
          Math.PI*2
        );

        context.fillRect(
          x,
          y,
          47*scale,
          17*scale
        );

        context.fill();
        context.restore();
      }

      function drawRain(
        x,
        y,
        widthValue,
        intensity
      ){
        if(!state.showPrecipitation){
          return;
        }

        var drops =
          Math.max(
            4,
            Math.round(
              7*intensity
            )
          );

        context.save();
        context.strokeStyle=
          'rgba(14,116,144,0.78)';

        context.lineWidth=2;

        for(
          var index=0;
          index<drops;
          index+=1
        ){
          var ratio =
            drops<=1
              ? 0.5
              : index/(drops-1);

          var dropX =
            x+
            ratio*
            widthValue;

          var offset =
            (
              index%2
            )*5;

          context.beginPath();

          context.moveTo(
            dropX,
            y+offset
          );

          context.lineTo(
            dropX-4,
            y+16+offset
          );

          context.stroke();
        }

        context.restore();
      }

      function drawSnowOrHail(
        x,
        y,
        count
      ){
        if(!state.showPrecipitation){
          return;
        }

        context.save();
        context.fillStyle='#DBEAFE';
        context.strokeStyle='#2563EB';
        context.lineWidth=1;

        for(
          var index=0;
          index<count;
          index+=1
        ){
          var px =
            x+
            (
              index%5
            )*15;

          var py =
            y+
            Math.floor(index/5)*18+
            (
              index%2
            )*4;

          context.beginPath();

          context.arc(
            px,
            py,
            3,
            0,
            Math.PI*2
          );

          context.fill();
          context.stroke();
        }

        context.restore();
      }

      function drawGround(){
        var groundGradient =
          context.createLinearGradient(
            0,
            305,
            0,
            385
          );

        groundGradient.addColorStop(
          0,
          '#D9F99D'
        );

        groundGradient.addColorStop(
          1,
          '#65A30D'
        );

        context.fillStyle=
          groundGradient;

        context.fillRect(
          28,
          305,
          514,
          80
        );

        context.strokeStyle='#4D7C0F';
        context.lineWidth=2;

        context.beginPath();
        context.moveTo(
          28,
          305
        );

        context.lineTo(
          542,
          305
        );

        context.stroke();
      }

      function processStage(progress){
        if(progress<33){
          return {
            key:'before',
            label:'过境前'
          };
        }

        if(progress<67){
          return {
            key:'passing',
            label:'过境时'
          };
        }

        return {
          key:'after',
          label:'过境后'
        };
      }

      function drawStageTimeline(
        progress,
        labels
      ){
        fillRoundedRect(
          28,
          397,
          724,
          30,
          15,
          '#FFFFFF',
          '#CBD5E1'
        );

        var startX=61;
        var endX=719;
        var lineY=412;

        context.strokeStyle='#CBD5E1';
        context.lineWidth=4;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(startX,lineY);
        context.lineTo(endX,lineY);
        context.stroke();

        var progressX =
          startX+
          (
            progress/100
          )*
          (
            endX-startX
          );

        context.strokeStyle='#0F766E';

        context.beginPath();
        context.moveTo(startX,lineY);
        context.lineTo(progressX,lineY);
        context.stroke();

        context.fillStyle='#0F766E';

        context.beginPath();
        context.arc(
          progressX,
          lineY,
          6,
          0,
          Math.PI*2
        );

        context.fill();

        var positions=[
          startX,
          (
            startX+endX
          )/2,
          endX
        ];

        labels.forEach(
          function(label,index){
            drawText(
              label,
              positions[index],
              412,
              10,
              index===1
                ? '#7C3AED'
                : '#475569',
              800,
              'center'
            );
          }
        );
      }

      function drawPressureCard(
        pressureText,
        trendText,
        color
      ){
        fillRoundedRect(
          566,
          36,
          186,
          106,
          15,
          '#FFFFFF',
          '#99F6E4'
        );

        drawText(
          '近地面气压',
          584,
          60,
          11,
          '#64748B',
          700,
          'left'
        );

        drawText(
          pressureText,
          584,
          87,
          20,
          color,
          900,
          'left'
        );

        drawText(
          trendText,
          584,
          119,
          11.5,
          color,
          800,
          'left'
        );
      }

      function drawWeatherCard(
        title,
        temperature,
        wind,
        weather
      ){
        fillRoundedRect(
          566,
          157,
          186,
          218,
          15,
          '#FFFFFF',
          '#CBD5E1'
        );

        drawText(
          title,
          584,
          181,
          14,
          '#115E59',
          850,
          'left'
        );

        drawText(
          '气温',
          584,
          214,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          temperature,
          584,
          238,
          16,
          '#F97316',
          850,
          'left'
        );

        drawText(
          '风向风力',
          584,
          270,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          wind,
          584,
          294,
          14,
          '#0E7490',
          850,
          'left'
        );

        drawText(
          '典型天气',
          584,
          326,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          weather,
          584,
          350,
          13,
          '#334155',
          850,
          'left'
        );
      }

      function drawColdFrontScene(
        progress,
        contrast
      ){
        drawGround();

        var stage =
          processStage(progress);

        var frontX =
          148+
          progress*3.25;

        var coldTopX =
          frontX-120;

        var coldGradient =
          context.createLinearGradient(
            30,
            155,
            420,
            305
          );

        coldGradient.addColorStop(
          0,
          'rgba(37,99,235,0.92)'
        );

        coldGradient.addColorStop(
          1,
          'rgba(186,230,253,0.90)'
        );

        context.fillStyle=
          coldGradient;

        context.beginPath();

        context.moveTo(
          28,
          305
        );

        context.lineTo(
          frontX,
          305
        );

        context.lineTo(
          coldTopX,
          150
        );

        context.lineTo(
          28,
          150
        );

        context.closePath();
        context.fill();

        var warmGradient =
          context.createLinearGradient(
            frontX,
            160,
            542,
            305
          );

        warmGradient.addColorStop(
          0,
          'rgba(254,215,170,0.88)'
        );

        warmGradient.addColorStop(
          1,
          'rgba(249,115,22,0.62)'
        );

        context.fillStyle=
          warmGradient;

        context.beginPath();

        context.moveTo(
          coldTopX,
          150
        );

        context.lineTo(
          frontX,
          305
        );

        context.lineTo(
          542,
          305
        );

        context.lineTo(
          542,
          150
        );

        context.closePath();
        context.fill();

        context.strokeStyle='#2563EB';
        context.lineWidth=4;

        context.beginPath();
        context.moveTo(
          coldTopX,
          150
        );

        context.lineTo(
          frontX,
          305
        );

        context.stroke();

        drawText(
          '冷气团',
          Math.max(
            88,
            frontX-150
          ),
          262,
          16,
          '#FFFFFF',
          900,
          'center'
        );

        drawText(
          '暖气团',
          Math.min(
            485,
            frontX+120
          ),
          262,
          16,
          '#9A3412',
          900,
          'center'
        );

        var cloudX =
          coldTopX-18;

        drawCloud(
          cloudX,
          102,
          0.90,
          '#64748B',
          0.96
        );

        drawCloud(
          cloudX+44,
          121,
          0.68,
          '#94A3B8',
          0.94
        );

        drawRain(
          cloudX+4,
          142,
          76,
          1.25
        );

        if(
          state.showPrecipitation &&
          contrast>=10
        ){
          drawSnowOrHail(
            cloudX+18,
            190,
            10
          );
        }

        if(state.showArrows){
          drawArrow(
            Math.max(
              50,
              frontX-190
            ),
            282,
            frontX-18,
            282,
            '#1D4ED8',
            4
          );

          drawArrow(
            frontX+150,
            270,
            frontX+42,
            270,
            '#EA580C',
            3
          );

          drawCurvedArrow(
            frontX+24,
            250,
            frontX-18,
            195,
            coldTopX+4,
            137,
            '#7C3AED'
          );
        }

        var pressureText;
        var pressureTrend;
        var temperature;
        var wind;
        var weather;

        if(stage.key==='before'){
          pressureText='较低';
          pressureTrend='气压逐渐下降';
          temperature='偏高';
          wind='偏南风增强';
          weather='云量逐渐增多';
        }else if(stage.key==='passing'){
          pressureText='转折';
          pressureTrend='先降后快速升高';
          temperature='明显下降';
          wind='风向突变、风力较大';
          weather='短时强降水';
        }else{
          pressureText='较高';
          pressureTrend='气压继续升高';
          temperature='较低';
          wind='偏北风';
          weather='天气转晴';
        }

        if(state.showPressure){
          drawPressureCard(
            pressureText,
            pressureTrend,
            stage.key==='passing'
              ? '#7C3AED'
              : '#2563EB'
          );
        }else{
          fillRoundedRect(
            566,
            36,
            186,
            106,
            15,
            '#F8FAFC',
            '#E2E8F0'
          );

          drawText(
            '气压标注已隐藏',
            659,
            89,
            12,
            '#64748B',
            750,
            'center'
          );
        }

        drawWeatherCard(
          '冷锋'+stage.label,
          temperature,
          wind,
          weather
        );

        drawStageTimeline(
          progress,
          [
            '暖区控制',
            '冷锋过境',
            '冷区控制'
          ]
        );

        result.textContent =
          '冷锋'+
          stage.label+
          '：冷气团主动推进并楔入暖气团下方，锋面坡度较陡，降水多出现在锋后附近。'+
          (
            stage.key==='passing'
              ? '过境时常出现大风、短时强降水和明显降温。'
              : stage.key==='after'
                ? '过境后气温下降、气压升高，天气逐渐转晴。'
                : '过境前暖气团控制，气温较高，气压逐渐下降。'
          );
      }

      function drawWarmFrontScene(
        progress,
        contrast
      ){
        drawGround();

        var stage =
          processStage(progress);

        var frontX =
          132+
          progress*3.10;

        var slopeTopX =
          Math.min(
            520,
            frontX+245
          );

        var coldGradient =
          context.createLinearGradient(
            frontX,
            175,
            542,
            305
          );

        coldGradient.addColorStop(
          0,
          'rgba(147,197,253,0.92)'
        );

        coldGradient.addColorStop(
          1,
          'rgba(37,99,235,0.74)'
        );

        context.fillStyle=
          coldGradient;

        context.beginPath();

        context.moveTo(
          frontX,
          305
        );

        context.lineTo(
          slopeTopX,
          164
        );

        context.lineTo(
          542,
          164
        );

        context.lineTo(
          542,
          305
        );

        context.closePath();
        context.fill();

        var warmGradient =
          context.createLinearGradient(
            28,
            150,
            slopeTopX,
            305
          );

        warmGradient.addColorStop(
          0,
          'rgba(249,115,22,0.70)'
        );

        warmGradient.addColorStop(
          1,
          'rgba(254,215,170,0.88)'
        );

        context.fillStyle=
          warmGradient;

        context.beginPath();

        context.moveTo(
          28,
          150
        );

        context.lineTo(
          slopeTopX,
          150
        );

        context.lineTo(
          frontX,
          305
        );

        context.lineTo(
          28,
          305
        );

        context.closePath();
        context.fill();

        context.strokeStyle='#DC2626';
        context.lineWidth=4;

        context.beginPath();
        context.moveTo(
          frontX,
          305
        );

        context.lineTo(
          slopeTopX,
          150
        );

        context.stroke();

        drawText(
          '暖气团',
          Math.max(
            100,
            frontX-75
          ),
          258,
          16,
          '#9A3412',
          900,
          'center'
        );

        drawText(
          '冷气团',
          Math.min(
            492,
            frontX+165
          ),
          267,
          16,
          '#FFFFFF',
          900,
          'center'
        );

        var cloudStart =
          Math.min(
            355,
            frontX+70
          );

        drawCloud(
          cloudStart,
          107,
          0.58,
          '#CBD5E1',
          0.92
        );

        drawCloud(
          cloudStart+72,
          129,
          0.72,
          '#94A3B8',
          0.94
        );

        drawCloud(
          cloudStart+134,
          159,
          0.62,
          '#64748B',
          0.92
        );

        drawRain(
          cloudStart+70,
          170,
          126,
          0.95+
          contrast/35
        );

        if(state.showArrows){
          drawArrow(
            Math.max(
              45,
              frontX-135
            ),
            275,
            frontX-15,
            275,
            '#EA580C',
            4
          );

          drawArrow(
            frontX+185,
            285,
            frontX+62,
            285,
            '#2563EB',
            3
          );

          drawCurvedArrow(
            frontX-8,
            260,
            frontX+75,
            202,
            slopeTopX-8,
            140,
            '#7C3AED'
          );
        }

        var pressureText;
        var pressureTrend;
        var temperature;
        var wind;
        var weather;

        if(stage.key==='before'){
          pressureText='下降';
          pressureTrend='气压持续降低';
          temperature='偏低';
          wind='偏东或东南风';
          weather='层状云逐渐增厚';
        }else if(stage.key==='passing'){
          pressureText='较低';
          pressureTrend='气压变化缓慢';
          temperature='逐渐升高';
          wind='风向缓慢转变';
          weather='连续性降水';
        }else{
          pressureText='稳定';
          pressureTrend='气压趋于稳定';
          temperature='升高';
          wind='偏南风';
          weather='降水减弱';
        }

        if(state.showPressure){
          drawPressureCard(
            pressureText,
            pressureTrend,
            stage.key==='passing'
              ? '#7C3AED'
              : '#DC2626'
          );
        }else{
          fillRoundedRect(
            566,
            36,
            186,
            106,
            15,
            '#F8FAFC',
            '#E2E8F0'
          );

          drawText(
            '气压标注已隐藏',
            659,
            89,
            12,
            '#64748B',
            750,
            'center'
          );
        }

        drawWeatherCard(
          '暖锋'+stage.label,
          temperature,
          wind,
          weather
        );

        drawStageTimeline(
          progress,
          [
            '冷区控制',
            '暖锋过境',
            '暖区控制'
          ]
        );

        result.textContent =
          '暖锋'+
          stage.label+
          '：暖气团沿冷气团上方缓慢爬升，锋面坡度较缓，云系和降水多位于锋前较大范围。'+
          (
            stage.key==='passing'
              ? '过境时多为持续时间较长、强度较均匀的连续性降水。'
              : stage.key==='after'
                ? '过境后气温升高，暖气团控制，降水逐渐减弱。'
                : '过境前冷气团控制，气温较低，层状云逐渐增厚。'
          );
      }

      function drawFrontSymbol(
        type,
        startX,
        startY,
        endX,
        endY
      ){
        context.save();

        var color =
          type==='cold'
            ? '#2563EB'
            : '#DC2626';

        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=3;

        context.beginPath();
        context.moveTo(
          startX,
          startY
        );

        context.lineTo(
          endX,
          endY
        );

        context.stroke();

        var length =
          Math.hypot(
            endX-startX,
            endY-startY
          );

        var angle =
          Math.atan2(
            endY-startY,
            endX-startX
          );

        var normalAngle =
          angle+
          Math.PI/2;

        for(
          var distance=30;
          distance<length-10;
          distance+=34
        ){
          var ratio =
            distance/length;

          var x =
            startX+
            (
              endX-startX
            )*
            ratio;

          var y =
            startY+
            (
              endY-startY
            )*
            ratio;

          if(type==='cold'){
            context.beginPath();

            context.moveTo(
              x,
              y
            );

            context.lineTo(
              x+
              Math.cos(
                normalAngle
              )*11+
              Math.cos(angle)*8,
              y+
              Math.sin(
                normalAngle
              )*11+
              Math.sin(angle)*8
            );

            context.lineTo(
              x+
              Math.cos(
                normalAngle
              )*11-
              Math.cos(angle)*8,
              y+
              Math.sin(
                normalAngle
              )*11-
              Math.sin(angle)*8
            );

            context.closePath();
            context.fill();
          }else{
            context.beginPath();

            context.arc(
              x+
              Math.cos(
                normalAngle
              )*7,
              y+
              Math.sin(
                normalAngle
              )*7,
              7,
              angle+Math.PI,
              angle,
              false
            );

            context.stroke();
          }
        }

        context.restore();
      }

      function drawCycloneScene(
        progress,
        contrast
      ){
        var stage;

        if(progress<33){
          stage={
            key:'developing',
            label:'发展阶段'
          };
        }else if(progress<72){
          stage={
            key:'mature',
            label:'成熟阶段'
          };
        }else{
          stage={
            key:'weakening',
            label:'减弱阶段'
          };
        }

        var centerX=287;
        var centerY=208;

        var intensity =
          stage.key==='developing'
            ? 0.55+
              progress/33*0.35
            : stage.key==='mature'
              ? 1
              : 1-
                (
                  progress-72
                )/28*0.42;

        var background =
          context.createLinearGradient(
            28,
            30,
            542,
            385
          );

        background.addColorStop(
          0,
          '#DBEAFE'
        );

        background.addColorStop(
          0.5,
          '#ECFDF5'
        );

        background.addColorStop(
          1,
          '#FEF3C7'
        );

        context.fillStyle=
          background;

        context.fillRect(
          28,
          30,
          514,
          355
        );

        context.strokeStyle='#0284C7';
        context.lineWidth=1.5;

        context.strokeRect(
          28,
          30,
          514,
          355
        );

        var radii=[
          48,
          80,
          116,
          154
        ];

        radii.forEach(
          function(radius,index){
            context.strokeStyle=
              index<2
                ? '#0E7490'
                : '#64748B';

            context.lineWidth=
              index===0
                ? 2.8
                : 1.6;

            context.setLineDash(
              index===3
                ? [7,5]
                : []
            );

            context.beginPath();

            context.ellipse(
              centerX,
              centerY,
              radius,
              radius*0.72,
              -0.15,
              0,
              Math.PI*2
            );

            context.stroke();
          }
        );

        context.setLineDash([]);

        context.fillStyle='#DC2626';

        context.beginPath();

        context.arc(
          centerX,
          centerY,
          22,
          0,
          Math.PI*2
        );

        context.fill();

        drawText(
          'L',
          centerX,
          centerY,
          22,
          '#FFFFFF',
          900,
          'center'
        );

        if(state.showPressure){
          var centerPressure =
            Math.round(
              1010-
              intensity*
              (
                14+
                contrast*0.8
              )
            );

          drawText(
            centerPressure+' hPa',
            centerX,
            centerY+35,
            11,
            '#B91C1C',
            850,
            'center'
          );

          drawText(
            '1000',
            centerX+55,
            centerY-31,
            9.5,
            '#475569',
            700,
            'center'
          );

          drawText(
            '1004',
            centerX+90,
            centerY-54,
            9.5,
            '#475569',
            700,
            'center'
          );

          drawText(
            '1008',
            centerX+128,
            centerY-78,
            9.5,
            '#475569',
            700,
            'center'
          );
        }

        if(state.showArrows){
          var arrowColor='#0F766E';

          [
            {
              sx:centerX-142,
              sy:centerY+20,
              cx:centerX-92,
              cy:centerY+105,
              ex:centerX-35,
              ey:centerY+65
            },
            {
              sx:centerX-30,
              sy:centerY-112,
              cx:centerX-118,
              cy:centerY-90,
              ex:centerX-74,
              ey:centerY-30
            },
            {
              sx:centerX+142,
              sy:centerY-10,
              cx:centerX+98,
              cy:centerY-104,
              ex:centerX+39,
              ey:centerY-65
            },
            {
              sx:centerX+24,
              sy:centerY+112,
              cx:centerX+118,
              cy:centerY+85,
              ex:centerX+72,
              ey:centerY+28
            }
          ].forEach(
            function(arrow){
              drawCurvedArrow(
                arrow.sx,
                arrow.sy,
                arrow.cx,
                arrow.cy,
                arrow.ex,
                arrow.ey,
                arrowColor
              );
            }
          );

          drawText(
            '北半球近地面逆时针辐合',
            centerX,
            360,
            11,
            '#0F766E',
            850,
            'center'
          );
        }

        drawFrontSymbol(
          'cold',
          centerX-5,
          centerY+20,
          centerX-176,
          centerY+127
        );

        drawFrontSymbol(
          'warm',
          centerX+16,
          centerY-8,
          centerX+194,
          centerY-80
        );

        drawText(
          '冷锋',
          centerX-150,
          centerY+105,
          11,
          '#1D4ED8',
          850,
          'center'
        );

        drawText(
          '暖锋',
          centerX+166,
          centerY-93,
          11,
          '#B91C1C',
          850,
          'center'
        );

        if(state.showPrecipitation){
          context.save();

          context.strokeStyle=
            'rgba(14,116,144,0.58)';

          context.fillStyle=
            'rgba(125,211,252,0.28)';

          context.lineWidth=2;

          context.beginPath();

          context.arc(
            centerX,
            centerY,
            105,
            Math.PI*0.18,
            Math.PI*1.74
          );

          context.lineTo(
            centerX,
            centerY
          );

          context.closePath();
          context.fill();
          context.stroke();

          context.restore();

          drawCloud(
            centerX-85,
            centerY-70,
            0.56,
            '#64748B',
            0.88
          );

          drawCloud(
            centerX+62,
            centerY+43,
            0.62,
            '#94A3B8',
            0.88
          );

          drawRain(
            centerX-76,
            centerY-43,
            45,
            intensity
          );

          drawRain(
            centerX+68,
            centerY+71,
            48,
            intensity
          );
        }

        var pressureText;
        var pressureTrend;
        var temperature;
        var wind;
        var weather;

        if(stage.key==='developing'){
          pressureText='下降';
          pressureTrend='中心气压逐渐降低';
          temperature='冷暖差异明显';
          wind='气流开始辐合';
          weather='云系发展';
        }else if(stage.key==='mature'){
          pressureText='最低';
          pressureTrend='等压线密集';
          temperature='冷暖空气交汇';
          wind='风力较强';
          weather='大范围云雨';
        }else{
          pressureText='回升';
          pressureTrend='中心气压升高';
          temperature='温差减小';
          wind='风力减弱';
          weather='云雨逐渐减弱';
        }

        if(state.showPressure){
          drawPressureCard(
            pressureText,
            pressureTrend,
            stage.key==='mature'
              ? '#DC2626'
              : '#7C3AED'
          );
        }else{
          fillRoundedRect(
            566,
            36,
            186,
            106,
            15,
            '#F8FAFC',
            '#E2E8F0'
          );

          drawText(
            '气压标注已隐藏',
            659,
            89,
            12,
            '#64748B',
            750,
            'center'
          );
        }

        drawWeatherCard(
          '气旋'+stage.label,
          temperature,
          wind,
          weather
        );

        drawStageTimeline(
          progress,
          [
            '气旋发展',
            '气旋成熟',
            '气旋减弱'
          ]
        );

        result.textContent =
          '气旋'+
          stage.label+
          '：北半球近地面空气围绕低压中心呈逆时针辐合，上升气流有利于云雨形成。'+
          (
            stage.key==='mature'
              ? '成熟阶段中心气压较低、等压线较密、风力较强，冷锋和暖锋附近常有明显云雨。'
              : stage.key==='weakening'
                ? '减弱阶段中心气压回升、风力减弱，锋面和云雨范围逐渐缩小。'
                : '发展阶段中心气压下降，气流辐合增强，云系逐渐组织发展。'
          );
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var progress =
          clamp(
            parseFloat(
              progressInput.value
            ) || 0,
            0,
            100
          );

        var contrast =
          clamp(
            parseFloat(
              contrastInput.value
            ) || 8,
            4,
            14
          );

        progressValue.textContent =
          Math.round(progress)+'%';

        contrastValue.textContent =
          Math.round(contrast)+'℃';

        Array.prototype.forEach.call(
          modeButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-system-mode'
              )===state.mode
                ? 'true'
                : 'false'
            );
          }
        );

        precipitationToggle.setAttribute(
          'data-active',
          state.showPrecipitation
            ? 'true'
            : 'false'
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
          0.55,
          '#F8FAFC'
        );

        background.addColorStop(
          1,
          '#E0F2FE'
        );

        context.fillStyle=
          background;

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
          367,
          18,
          'rgba(255,255,255,0.82)',
          '#BAE6FD'
        );

        if(state.mode==='warm-front'){
          drawWarmFrontScene(
            progress,
            contrast
          );
        }else if(state.mode==='cyclone'){
          drawCycloneScene(
            progress,
            contrast
          );
        }else{
          drawColdFrontScene(
            progress,
            contrast
          );
        }

        drawText(
          '锋面坡度、降水位置、气压和风场均为教学示意，不代表真实天气预报。',
          390,
          437,
          10,
          '#64748B',
          650,
          'center'
        );
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

        progressInput.value =
          String(
            Math.floor(
              (
                elapsed/70
              )%101
            )
          );

        state.particlePhase =
          (
            elapsed/3200
          )%1;

        render();

        state.raf =
          requestAnimationFrame(
            animate
          );
      }

      progressInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      contrastInput.addEventListener(
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
                  'data-system-mode'
                ) || 'cold-front';

              progressInput.value='42';
              state.particlePhase=0;

              render();
            }
          );
        }
      );

      precipitationToggle.addEventListener(
        'click',
        function(){
          state.showPrecipitation =
            !state.showPrecipitation;

          render();
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
          state.particlePhase=0;

          render();

          state.raf =
            requestAnimationFrame(
              animate
            );
        }
      );

      nextStageButton.addEventListener(
        'click',
        function(){
          if(state.auto)stopAuto();

          var current =
            parseFloat(
              progressInput.value
            ) || 0;

          if(current<33){
            progressInput.value='50';
          }else if(current<67){
            progressInput.value='84';
          }else{
            progressInput.value='16';
          }

          render();
        }
      );

      resetButton.addEventListener(
        'click',
        function(){
          stopAuto();

          progressInput.value =
            String(initialProgress);

          contrastInput.value =
            String(initialContrast);

          state.mode =
            '${systemMode}';

          state.showPrecipitation =
            ${showPrecipitation};

          state.showPressure =
            ${showPressure};

          state.showArrows =
            ${showArrows};

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

export const GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_FRONT_CYCLONE:
GeographyLabTemplate[] = [
  {
    id: 'geography-cold-warm-front-cyclone-weather',
    group: '🌦️ 大气运动与天气系统',
    name: '冷锋、暖锋和气旋天气过程',
    emoji: '🌧️',
    desc: '切换冷锋、暖锋和气旋，调节过程进度，观察云雨位置、气压、气温与风向变化。',
    params: [
      {
        key: 'systemMode',
        label: '初始天气系统',
        type: 'select',
        options: [
          {
            label: '冷锋天气过程',
            value: 'cold-front',
          },
          {
            label: '暖锋天气过程',
            value: 'warm-front',
          },
          {
            label: '气旋天气过程',
            value: 'cyclone',
          },
        ],
        defaultValue: 'cold-front',
      },
      {
        key: 'progress',
        label: '初始过程进度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
        hint: '0—32为前期，33—66为过境或成熟期，67—100为后期。',
      },
      {
        key: 'temperatureContrast',
        label: '冷暖气团温差',
        type: 'number',
        min: 4,
        max: 14,
        step: 1,
        defaultValue: 8,
        hint: '只改变教学模型中的冷暖差异和天气表现强度。',
      },
      {
        key: 'showPrecipitation',
        label: '显示云系与降水',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showPressure',
        label: '显示气压变化',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showArrows',
        label: '显示气流箭头',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildFrontCycloneHTML,
  },
]
