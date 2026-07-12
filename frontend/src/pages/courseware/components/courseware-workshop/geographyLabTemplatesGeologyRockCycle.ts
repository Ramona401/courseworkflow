/**
 * geographyLabTemplatesGeologyRockCycle.ts
 *
 * 第36批B2：岩石圈物质循环与三大岩石转化。
 *
 * 教学目标：
 * 1. 识别岩浆岩、沉积岩和变质岩三大岩石类型；
 * 2. 理解冷却凝固、风化侵蚀、搬运沉积、压实胶结、
 *    变质作用、熔融和地壳抬升等主要过程；
 * 3. 理解任一种岩石都可能经过不同路径转化为其他岩石；
 * 4. 比较温度、压力、水分、抬升和冷却速度对转化过程的影响；
 * 5. 区分快速冷却形成的细粒结构和缓慢冷却形成的粗粒结构；
 * 6. 理解沉积物经过压实和胶结形成沉积岩，
 *    已有岩石在不完全熔融条件下发生变质；
 * 7. 认识岩石圈物质循环没有固定起点和唯一方向。
 *
 * 教学边界：
 * - 所有温度、压力、深度、时间和转化速率均为相对教学量；
 * - 图中过程不代表具体岩层、矿区、火山或地质年代；
 * - 岩石分类与转化关系为中学课堂简化模型；
 * - 不用于矿产勘探、工程地质、边坡判断或实际岩石鉴定；
 * - 真实岩石形成还受矿物成分、流体、构造环境和时间尺度影响。
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

function buildRockCycleHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const requestedMaterial = stringValue(
    params,
    'initialMaterial',
    'magma',
  )

  const initialMaterial = [
    'magma',
    'igneous',
    'sediment',
    'sedimentary',
    'metamorphic',
  ].includes(requestedMaterial)
    ? requestedMaterial
    : 'magma'

  const temperature = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'temperature', 68),
    ),
  )

  const pressure = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'pressure', 52),
    ),
  )

  const surfaceWater = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'surfaceWater', 58),
    ),
  )

  const uplift = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'uplift', 46),
    ),
  )

  const coolingRate = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'coolingRate', 42),
    ),
  )

  const requestedMode = stringValue(
    params,
    'observationMode',
    'cycle',
  )

  const observationMode = [
    'cycle',
    'conditions',
    'structure',
  ].includes(requestedMode)
    ? requestedMode
    : 'cycle'

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
<div id="${rootId}" class="gl-rock-cycle-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border-radius:18px;
      border:1px solid #D6C7A5;
      background:#FFFFFF;
      color:#0F172A;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(91,61,22,0.10);
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
        #FEF3C7,
        #FDE7D5 48%,
        #DCFCE7
      );
      border-bottom:1px solid #D6C7A5;
    }

    #${rootId} .gl-title{
      color:#78350F;
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
      border-right:1px solid #E7D8B7;
      background:linear-gradient(
        180deg,
        #FFFBEB,
        #FFF7ED
      );
    }

    #${rootId} .gl-stage{
      position:relative;
      min-width:0;
      min-height:0;
      padding:8px;
      background:radial-gradient(
        circle at 50% 16%,
        #FFFFFF 0%,
        #F8FAFC 54%,
        #FDE7D5 100%
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
      color:#4B5563;
      font-size:11px;
      font-weight:750;
    }

    #${rootId} .gl-value{
      padding:3px 7px;
      border-radius:999px;
      background:#FEF3C7;
      color:#92400E;
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
        #FDE68A,
        #FB923C
      );
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:16px;
      height:16px;
      appearance:none;
      border-radius:50%;
      background:#B45309;
      border:2px solid #FFFFFF;
      box-shadow:0 1px 5px rgba(146,64,14,0.42);
    }

    #${rootId} select{
      width:100%;
      min-height:34px;
      padding:6px 8px;
      border:1px solid #E7C98D;
      border-radius:9px;
      background:#FFFFFF;
      color:#92400E;
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
      border:1px solid #F1DEC0;
      color:#4B5563;
      font-size:10.5px;
      font-weight:750;
    }

    #${rootId} .gl-switch-row input{
      accent-color:#B45309;
    }

    #${rootId} .gl-subtitle{
      margin:10px 0 6px;
      color:#92400E;
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
      border:1px solid #E7C98D;
      border-radius:9px;
      background:#FFFFFF;
      color:#92400E;
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
      border-color:#B45309;
    }

    #${rootId} button.active{
      border-color:#B45309;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #F97316,
        #B45309
      );
    }

    #${rootId} .gl-result{
      margin-top:9px;
      padding:9px;
      border-radius:11px;
      background:#FEF3C7;
      border:1px solid #E7C98D;
      color:#78350F;
      font-size:10.2px;
      font-weight:650;
      line-height:1.48;
      max-height:80px;
      overflow:auto;
    }

    #${rootId} .gl-rock-canvas{
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
      border:1px solid #F1D6A5;
      box-shadow:0 5px 15px rgba(91,61,22,0.08);
      text-align:center;
    }

    #${rootId} .gl-summary-card strong{
      display:block;
      color:#9A3412;
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
      🪨
    </div>

    <div>
      <div class="gl-title">
        岩石圈物质循环与三大岩石转化
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        调节温度、压力、水分、抬升和冷却速度，观察岩石转化路径
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 不用于实际岩石鉴定
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            初始物质
          </span>
        </div>

        <select data-role="initial-material">
          <option
            value="magma"
            ${initialMaterial === 'magma' ? 'selected' : ''}
          >
            岩浆
          </option>

          <option
            value="igneous"
            ${initialMaterial === 'igneous' ? 'selected' : ''}
          >
            岩浆岩
          </option>

          <option
            value="sediment"
            ${initialMaterial === 'sediment' ? 'selected' : ''}
          >
            沉积物
          </option>

          <option
            value="sedimentary"
            ${initialMaterial === 'sedimentary' ? 'selected' : ''}
          >
            沉积岩
          </option>

          <option
            value="metamorphic"
            ${initialMaterial === 'metamorphic' ? 'selected' : ''}
          >
            变质岩
          </option>
        </select>
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            地下温度
          </span>

          <span
            class="gl-value"
            data-role="temperature-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(temperature)}"
          data-role="temperature"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            构造压力
          </span>

          <span
            class="gl-value"
            data-role="pressure-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(pressure)}"
          data-role="pressure"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            地表水分
          </span>

          <span
            class="gl-value"
            data-role="water-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(surfaceWater)}"
          data-role="water"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            地壳抬升强度
          </span>

          <span
            class="gl-value"
            data-role="uplift-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(uplift)}"
          data-role="uplift"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            岩浆冷却速度
          </span>

          <span
            class="gl-value"
            data-role="cooling-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(coolingRate)}"
          data-role="cooling"
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
            value="cycle"
            ${observationMode === 'cycle' ? 'selected' : ''}
          >
            物质循环路径
          </option>

          <option
            value="conditions"
            ${observationMode === 'conditions' ? 'selected' : ''}
          >
            转化条件
          </option>

          <option
            value="structure"
            ${observationMode === 'structure' ? 'selected' : ''}
          >
            岩石结构
          </option>
        </select>
      </div>

      <div class="gl-switch-row">
        <span>显示过程标注</span>

        <input
          type="checkbox"
          data-role="label-switch"
          ${showLabels ? 'checked' : ''}
        />
      </div>

      <div class="gl-switch-row">
        <span>自动演示典型路径</span>

        <input
          type="checkbox"
          data-role="auto-switch"
          ${automatic ? 'checked' : ''}
        />
      </div>

      <div class="gl-subtitle">
        典型转化路径
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="cooling"
        >
          🌋 岩浆冷却
        </button>

        <button
          type="button"
          data-scenario="sedimentation"
        >
          🌧️ 风化沉积
        </button>

        <button
          type="button"
          data-scenario="lithification"
        >
          🧱 压实胶结
        </button>

        <button
          type="button"
          data-scenario="metamorphism"
        >
          🔥 变质作用
        </button>

        <button
          type="button"
          data-scenario="melting"
        >
          🌡️ 高温熔融
        </button>

        <button
          type="button"
          data-scenario="uplift"
        >
          ⛰️ 抬升出露
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-rock-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="岩石圈物质循环与三大岩石转化教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="material-value"></strong>
          <span>当前物质</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="process-value"></strong>
          <span>优势过程</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="product-value"></strong>
          <span>可能产物</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="structure-value"></strong>
          <span>结构特征</span>
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

      function drawCurveArrow(
        context,
        startX,
        startY,
        controlX,
        controlY,
        endX,
        endY,
        color,
        width,
        phase,
        active
      ){
        context.save();
        context.strokeStyle=color;
        context.lineWidth=
          active
            ? width+1.8
            : width;
        context.globalAlpha=
          active
            ? 1
            : 0.28;
        context.lineCap='round';
        context.setLineDash([10,7]);
        context.lineDashOffset=
          -phase*
          (
            active
              ? 48
              : 22
          );

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
        context.restore();

        var angle=Math.atan2(
          endY-controlY,
          endX-controlX
        );

        context.save();
        context.globalAlpha=
          active
            ? 1
            : 0.28;

        drawArrowHead(
          context,
          endX,
          endY,
          angle,
          color,
          active
            ? 12
            : 9
        );

        context.restore();
      }

      function drawRockCard(
        context,
        x,
        y,
        width,
        height,
        title,
        subtitle,
        fill,
        stroke,
        selected
      ){
        context.save();

        if(selected){
          context.shadowColor=
            'rgba(180,83,9,0.32)';
          context.shadowBlur=16;
        }

        fillRoundedRect(
          context,
          x,
          y,
          width,
          height,
          16,
          fill,
          selected
            ? '#B45309'
            : stroke
        );

        context.restore();

        drawText(
          context,
          title,
          x+width/2,
          y+25,
          13,
          '#3F2B1D',
          900,
          'center'
        );

        drawText(
          context,
          subtitle,
          x+width/2,
          y+48,
          9,
          '#64748B',
          700,
          'center'
        );
      }

      function drawIgneousTexture(
        context,
        x,
        y,
        width,
        height,
        cooling
      ){
        var grainCount=
          cooling>=65
            ? 42
            : cooling>=35
              ? 24
              : 12;

        var grainSize=
          cooling>=65
            ? 2
            : cooling>=35
              ? 4
              : 7;

        context.save();

        for(
          var index=0;
          index<grainCount;
          index+=1
        ){
          var column=
            index%7;

          var row=
            Math.floor(
              index/7
            );

          var grainX=
            x+
            14+
            column*
            (
              width-28
            )/
            6+
            Math.sin(
              index*1.7
            )*
            3;

          var grainY=
            y+
            13+
            row*
            13+
            Math.cos(
              index*1.3
            )*
            2;

          context.fillStyle=
            index%3===0
              ? '#E5E7EB'
              : index%3===1
                ? '#111827'
                : '#9CA3AF';

          context.beginPath();
          context.arc(
            grainX,
            grainY,
            grainSize,
            0,
            Math.PI*2
          );
          context.fill();
        }

        context.restore();
      }

      function drawSedimentaryTexture(
        context,
        x,
        y,
        width,
        height
      ){
        var layerColors=[
          '#D6B377',
          '#C7924B',
          '#E3C999',
          '#A96D32',
          '#DDBB7A'
        ];

        var layerHeight=
          (
            height-20
          )/
          layerColors.length;

        context.save();

        layerColors.forEach(
          function(color,index){
            context.fillStyle=color;

            context.fillRect(
              x+12,
              y+10+
              index*
              layerHeight,
              width-24,
              layerHeight-2
            );

            context.fillStyle=
              'rgba(255,255,255,0.34)';

            for(
              var dot=0;
              dot<5;
              dot+=1
            ){
              context.beginPath();
              context.arc(
                x+
                20+
                dot*
                (
                  width-40
                )/
                4+
                Math.sin(
                  index+
                  dot
                )*
                4,
                y+
                16+
                index*
                layerHeight+
                Math.cos(
                  dot*1.4
                )*
                3,
                1.8,
                0,
                Math.PI*2
              );
              context.fill();
            }
          }
        );

        context.restore();
      }

      function drawMetamorphicTexture(
        context,
        x,
        y,
        width,
        height,
        pressure
      ){
        var bandCount=
          5+
          Math.round(
            pressure/
            25
          );

        context.save();

        for(
          var band=0;
          band<bandCount;
          band+=1
        ){
          var bandY=
            y+
            12+
            band*
            (
              height-24
            )/
            Math.max(
              1,
              bandCount-1
            );

          context.strokeStyle=
            band%2===0
              ? '#5B3C22'
              : '#D6B377';

          context.lineWidth=
            3+
            band%2;

          context.beginPath();
          context.moveTo(
            x+12,
            bandY
          );

          context.bezierCurveTo(
            x+
            width*
            0.32,
            bandY-9,
            x+
            width*
            0.65,
            bandY+9,
            x+
            width-
            12,
            bandY
          );

          context.stroke();
        }

        context.restore();
      }

      function materialLabel(value){
        if(value==='igneous'){
          return '岩浆岩';
        }

        if(value==='sediment'){
          return '沉积物';
        }

        if(value==='sedimentary'){
          return '沉积岩';
        }

        if(value==='metamorphic'){
          return '变质岩';
        }

        return '岩浆';
      }

      function readState(){
        return {
          material:
            materialSelect.value,
          temperature:Number(
            temperatureInput.value
          ),
          pressure:Number(
            pressureInput.value
          ),
          water:Number(
            waterInput.value
          ),
          uplift:Number(
            upliftInput.value
          ),
          cooling:Number(
            coolingInput.value
          ),
          observationMode:
            observationSelect.value
        };
      }

      function calculate(values){
        var coolingScore=clamp(
          values.cooling*
          0.72+
          (
            100-values.temperature
          )*
          0.28,
          0,
          100
        );

        var weatheringScore=clamp(
          values.water*
          0.56+
          values.uplift*
          0.30+
          (
            100-values.temperature
          )*
          0.14,
          0,
          100
        );

        var lithificationScore=clamp(
          values.pressure*
          0.42+
          values.water*
          0.18+
          35,
          0,
          100
        );

        var metamorphismScore=clamp(
          values.temperature*
          0.48+
          values.pressure*
          0.52-
          20,
          0,
          100
        );

        var meltingScore=clamp(
          values.temperature*
          1.18+
          values.pressure*
          0.12-
          52,
          0,
          100
        );

        var upliftScore=clamp(
          values.uplift*
          0.78+
          values.pressure*
          0.16,
          0,
          100
        );

        var possibleProcesses=[
          {
            key:'cooling',
            label:'冷却凝固',
            score:
              values.material==='magma'
                ? coolingScore+30
                : coolingScore*0.26
          },
          {
            key:'weathering',
            label:'风化侵蚀',
            score:
              values.material==='magma'
                ? weatheringScore*0.18
                : weatheringScore
          },
          {
            key:'lithification',
            label:'压实胶结',
            score:
              values.material==='sediment'
                ? lithificationScore+28
                : lithificationScore*0.22
          },
          {
            key:'metamorphism',
            label:'变质作用',
            score:
              (
                values.material==='igneous' ||
                values.material==='sedimentary'
              )
                ? metamorphismScore+22
                : values.material==='metamorphic'
                  ? metamorphismScore*0.35
                  : metamorphismScore*0.12
          },
          {
            key:'melting',
            label:'熔融',
            score:
              values.material==='magma'
                ? meltingScore*0.05
                : meltingScore
          },
          {
            key:'uplift',
            label:'抬升出露',
            score:
              values.material==='magma'
                ? upliftScore*0.20
                : upliftScore
          }
        ];

        possibleProcesses.sort(
          function(left,right){
            return right.score-left.score;
          }
        );

        var dominant=
          possibleProcesses[0];

        var product=
          values.material;

        if(dominant.key==='cooling'){
          product='igneous';
        }else if(
          dominant.key==='weathering'
        ){
          product='sediment';
        }else if(
          dominant.key==='lithification'
        ){
          product='sedimentary';
        }else if(
          dominant.key==='metamorphism'
        ){
          product='metamorphic';
        }else if(
          dominant.key==='melting'
        ){
          product='magma';
        }

        var structure='无固定结构';

        if(product==='igneous'){
          structure=
            values.cooling>=65
              ? '细粒或玻璃质'
              : values.cooling>=35
                ? '中等晶粒'
                : '粗粒晶体';
        }else if(
          product==='sedimentary'
        ){
          structure='层理与碎屑';
        }else if(
          product==='metamorphic'
        ){
          structure=
            values.pressure>=62
              ? '明显片理或条带'
              : '重结晶结构';
        }else if(
          product==='sediment'
        ){
          structure='松散颗粒';
        }else{
          structure='熔融状态';
        }

        return {
          coolingScore:coolingScore,
          weatheringScore:weatheringScore,
          lithificationScore:
            lithificationScore,
          metamorphismScore:
            metamorphismScore,
          meltingScore:meltingScore,
          upliftScore:upliftScore,
          dominant:dominant,
          product:product,
          structure:structure,
          allProcesses:
            possibleProcesses
        };
      }

      function describe(values,model){
        if(model.dominant.key==='cooling'){
          return values.cooling>=65
            ? '岩浆快速冷却，晶体来不及充分生长，形成细粒或玻璃质结构。'
            : '岩浆缓慢冷却，矿物晶体有较长时间生长，形成较粗晶粒结构。';
        }

        if(model.dominant.key==='weathering'){
          return '岩石抬升出露后，在水、温度变化和重力等作用下发生风化侵蚀，'+
            '形成碎屑并经过搬运、沉积成为沉积物。';
        }

        if(model.dominant.key==='lithification'){
          return '沉积物不断堆积，孔隙被压缩，矿物胶结物把颗粒连接起来，'+
            '经过压实和胶结形成沉积岩。';
        }

        if(model.dominant.key==='metamorphism'){
          return '已有岩石在较高温度和压力下发生矿物重结晶和结构调整，'+
            '但没有完全熔融，因此形成变质岩。';
        }

        if(model.dominant.key==='melting'){
          return '岩石温度继续升高并达到熔融条件后，固态岩石转化为岩浆；'+
            '岩浆再次冷却又可形成新的岩浆岩。';
        }

        return '地壳抬升使深部岩石接近或出露地表，'+
          '为后续风化侵蚀和沉积循环创造条件。';
      }

      function drawCycle(
        context,
        values,
        model
      ){
        var cards={
          magma:{
            x:335,
            y:72,
            width:150,
            height:70,
            title:'岩浆',
            subtitle:'熔融物质',
            fill:'#FECACA',
            stroke:'#EF4444'
          },
          igneous:{
            x:567,
            y:169,
            width:150,
            height:78,
            title:'岩浆岩',
            subtitle:'冷却凝固',
            fill:'#E5E7EB',
            stroke:'#6B7280'
          },
          metamorphic:{
            x:462,
            y:296,
            width:150,
            height:78,
            title:'变质岩',
            subtitle:'重结晶与变质',
            fill:'#E9D5FF',
            stroke:'#7E22CE'
          },
          sedimentary:{
            x:207,
            y:296,
            width:150,
            height:78,
            title:'沉积岩',
            subtitle:'压实与胶结',
            fill:'#FDE68A',
            stroke:'#B45309'
          },
          sediment:{
            x:103,
            y:169,
            width:150,
            height:78,
            title:'沉积物',
            subtitle:'碎屑与沉积',
            fill:'#FED7AA',
            stroke:'#EA580C'
          }
        };

        var selectedMaterial=
          values.material;

        Object.keys(cards).forEach(
          function(key){
            var card=cards[key];

            drawRockCard(
              context,
              card.x,
              card.y,
              card.width,
              card.height,
              card.title,
              card.subtitle,
              card.fill,
              card.stroke,
              selectedMaterial===key
            );
          }
        );

        drawCurveArrow(
          context,
          476,
          125,
          587,
          118,
          603,
          169,
          '#EF4444',
          3,
          state.phase,
          model.dominant.key==='cooling'
        );

        drawCurveArrow(
          context,
          567,
          232,
          470,
          287,
          449,
          329,
          '#7E22CE',
          3,
          state.phase,
          model.dominant.key==='metamorphism' &&
          values.material==='igneous'
        );

        drawCurveArrow(
          context,
          357,
          329,
          414,
          253,
          425,
          145,
          '#EF4444',
          3,
          state.phase,
          model.dominant.key==='melting'
        );

        drawCurveArrow(
          context,
          208,
          330,
          151,
          276,
          175,
          247,
          '#B45309',
          3,
          state.phase,
          model.dominant.key==='weathering' &&
          values.material==='sedimentary'
        );

        drawCurveArrow(
          context,
          253,
          208,
          278,
          279,
          284,
          296,
          '#EA580C',
          3,
          state.phase,
          model.dominant.key==='lithification'
        );

        drawCurveArrow(
          context,
          357,
          333,
          405,
          292,
          462,
          333,
          '#7E22CE',
          3,
          state.phase,
          model.dominant.key==='metamorphism' &&
          values.material==='sedimentary'
        );

        drawCurveArrow(
          context,
          495,
          298,
          454,
          210,
          432,
          145,
          '#EF4444',
          3,
          state.phase,
          model.dominant.key==='melting' &&
          values.material==='metamorphic'
        );

        drawCurveArrow(
          context,
          582,
          199,
          400,
          151,
          247,
          193,
          '#EA580C',
          2.5,
          state.phase,
          model.dominant.key==='weathering' &&
          values.material==='igneous'
        );

        drawCurveArrow(
          context,
          462,
          348,
          300,
          410,
          182,
          247,
          '#EA580C',
          2.5,
          state.phase,
          model.dominant.key==='weathering' &&
          values.material==='metamorphic'
        );

        if(labelSwitch.checked){
          drawText(
            context,
            '冷却凝固',
            543,
            127,
            9,
            '#B91C1C',
            800,
            'center'
          );

          drawText(
            context,
            '风化·侵蚀·搬运·沉积',
            357,
            184,
            9,
            '#C2410C',
            800,
            'center'
          );

          drawText(
            context,
            '压实胶结',
            264,
            270,
            9,
            '#92400E',
            800,
            'center'
          );

          drawText(
            context,
            '变质作用',
            416,
            313,
            9,
            '#6B21A8',
            800,
            'center'
          );

          drawText(
            context,
            '熔融',
            432,
            219,
            9,
            '#B91C1C',
            800,
            'center'
          );
        }
      }

      function drawConditions(
        context,
        values,
        model
      ){
        fillRoundedRect(
          context,
          53,
          76,
          504,
          286,
          17,
          '#FFFDF7',
          '#E7C98D'
        );

        var processRows=[
          {
            key:'cooling',
            label:'冷却凝固',
            condition:'低温度或冷却增强',
            value:model.coolingScore,
            color:'#2563EB'
          },
          {
            key:'weathering',
            label:'风化侵蚀',
            condition:'水分、抬升与地表暴露',
            value:model.weatheringScore,
            color:'#EA580C'
          },
          {
            key:'lithification',
            label:'压实胶结',
            condition:'沉积物堆积与压力',
            value:model.lithificationScore,
            color:'#B45309'
          },
          {
            key:'metamorphism',
            label:'变质作用',
            condition:'较高温度与压力',
            value:model.metamorphismScore,
            color:'#7E22CE'
          },
          {
            key:'melting',
            label:'熔融',
            condition:'更高温度',
            value:model.meltingScore,
            color:'#DC2626'
          },
          {
            key:'uplift',
            label:'抬升出露',
            condition:'构造抬升',
            value:model.upliftScore,
            color:'#047857'
          }
        ];

        processRows.forEach(
          function(item,index){
            var y=
              104+
              index*
              41;

            var active=
              model.dominant.key===
              item.key;

            if(active){
              fillRoundedRect(
                context,
                68,
                y-14,
                470,
                34,
                9,
                'rgba(254,243,199,0.92)',
                '#F59E0B'
              );
            }

            drawText(
              context,
              item.label,
              82,
              y,
              10.5,
              item.color,
              850,
              'left'
            );

            drawText(
              context,
              item.condition,
              185,
              y,
              9,
              '#64748B',
              700,
              'left'
            );

            context.fillStyle='#E5E7EB';

            context.fillRect(
              372,
              y-5,
              126,
              10
            );

            context.fillStyle=item.color;

            context.fillRect(
              372,
              y-5,
              clamp(
                item.value/
                100*
                126,
                2,
                126
              ),
              10
            );

            drawText(
              context,
              Math.round(
                item.value
              )+
              '%',
              521,
              y,
              9,
              item.color,
              800,
              'right'
            );
          }
        );

        fillRoundedRect(
          context,
          581,
          76,
          178,
          286,
          17,
          '#F8FAFC',
          '#D6C7A5'
        );

        drawText(
          context,
          '当前优势过程',
          602,
          103,
          12,
          '#78350F',
          850,
          'left'
        );

        drawText(
          context,
          model.dominant.label,
          670,
          141,
          21,
          '#B45309',
          900,
          'center'
        );

        drawText(
          context,
          '可能形成',
          602,
          183,
          10,
          '#64748B',
          700,
          'left'
        );

        drawText(
          context,
          materialLabel(
            model.product
          ),
          670,
          213,
          18,
          '#7E22CE',
          900,
          'center'
        );

        drawText(
          context,
          '注意：优势过程不等于',
          670,
          267,
          9,
          '#64748B',
          700,
          'center'
        );

        drawText(
          context,
          '唯一过程或固定路径',
          670,
          286,
          9,
          '#64748B',
          700,
          'center'
        );

        drawText(
          context,
          '岩石循环可沿多条路径进行',
          670,
          323,
          9,
          '#92400E',
          800,
          'center'
        );
      }

      function drawStructure(
        context,
        values,
        model
      ){
        var cardY=100;
        var cardWidth=201;
        var cardHeight=194;
        var positions=[
          55,
          310,
          565
        ];

        drawRockCard(
          context,
          positions[0],
          cardY,
          cardWidth,
          cardHeight,
          '岩浆岩',
          values.cooling>=65
            ? '快速冷却：细粒'
            : values.cooling>=35
              ? '中速冷却：中粒'
              : '缓慢冷却：粗粒',
          '#E5E7EB',
          '#6B7280',
          model.product==='igneous'
        );

        drawRockCard(
          context,
          positions[1],
          cardY,
          cardWidth,
          cardHeight,
          '沉积岩',
          '层理、碎屑与胶结',
          '#FDE68A',
          '#B45309',
          model.product==='sedimentary'
        );

        drawRockCard(
          context,
          positions[2],
          cardY,
          cardWidth,
          cardHeight,
          '变质岩',
          values.pressure>=62
            ? '片理或条带明显'
            : '矿物重结晶',
          '#E9D5FF',
          '#7E22CE',
          model.product==='metamorphic'
        );

        drawIgneousTexture(
          context,
          positions[0]+12,
          cardY+66,
          cardWidth-24,
          cardHeight-82,
          values.cooling
        );

        drawSedimentaryTexture(
          context,
          positions[1]+12,
          cardY+66,
          cardWidth-24,
          cardHeight-82
        );

        drawMetamorphicTexture(
          context,
          positions[2]+12,
          cardY+66,
          cardWidth-24,
          cardHeight-82,
          values.pressure
        );

        fillRoundedRect(
          context,
          157,
          321,
          506,
          54,
          13,
          '#FFF7ED',
          '#F1D6A5'
        );

        drawText(
          context,
          '岩浆岩结构主要受冷却速度影响；沉积岩常保留层理；',
          410,
          338,
          10,
          '#78350F',
          750,
          'center'
        );

        drawText(
          context,
          '变质岩在温压作用下发生重结晶，可能形成片理或条带。',
          410,
          358,
          10,
          '#78350F',
          750,
          'center'
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

        temperatureValue.textContent=
          Math.round(
            values.temperature
          )+
          '%';

        pressureValue.textContent=
          Math.round(
            values.pressure
          )+
          '%';

        waterValue.textContent=
          Math.round(
            values.water
          )+
          '%';

        upliftValue.textContent=
          Math.round(
            values.uplift
          )+
          '%';

        coolingValue.textContent=
          Math.round(
            values.cooling
          )+
          '%';

        materialValue.textContent=
          materialLabel(
            values.material
          );

        processValue.textContent=
          model.dominant.label;

        productValue.textContent=
          materialLabel(
            model.product
          );

        structureValue.textContent=
          model.structure;

        result.textContent=
          '当前物质为'+
          materialLabel(
            values.material
          )+
          '，优势过程是'+
          model.dominant.label+
          '，可能形成'+
          materialLabel(
            model.product
          )+
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
          '#FFF7ED'
        );

        background.addColorStop(
          0.52,
          '#F8FAFC'
        );

        background.addColorStop(
          1,
          '#FDE7D5'
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
          'rgba(255,255,255,0.91)',
          '#E7C98D'
        );

        drawText(
          context,
          '三大岩石与岩石圈物质循环',
          40,
          43,
          14,
          '#78350F',
          850,
          'left'
        );

        drawText(
          context,
          '循环没有固定起点，也不存在唯一转化方向',
          780,
          43,
          9.5,
          '#64748B',
          650,
          'right'
        );

        if(
          values.observationMode===
          'conditions'
        ){
          drawConditions(
            context,
            values,
            model
          );
        }else if(
          values.observationMode===
          'structure'
        ){
          drawStructure(
            context,
            values,
            model
          );
        }else{
          drawCycle(
            context,
            values,
            model
          );
        }

        drawText(
          context,
          '所有温度、压力、时间和速率均为相对教学量，不用于实际岩石鉴定或工程判断。',
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
          cooling:{
            material:'magma',
            temperature:40,
            pressure:28,
            water:18,
            uplift:22,
            cooling:82,
            observationMode:'structure'
          },
          sedimentation:{
            material:'igneous',
            temperature:30,
            pressure:22,
            water:92,
            uplift:86,
            cooling:45,
            observationMode:'cycle'
          },
          lithification:{
            material:'sediment',
            temperature:36,
            pressure:74,
            water:55,
            uplift:18,
            cooling:42,
            observationMode:'conditions'
          },
          metamorphism:{
            material:'sedimentary',
            temperature:76,
            pressure:88,
            water:24,
            uplift:22,
            cooling:35,
            observationMode:'structure'
          },
          melting:{
            material:'metamorphic',
            temperature:96,
            pressure:70,
            water:14,
            uplift:10,
            cooling:12,
            observationMode:'cycle'
          },
          uplift:{
            material:'metamorphic',
            temperature:34,
            pressure:42,
            water:70,
            uplift:95,
            cooling:50,
            observationMode:'conditions'
          }
        };

        var scenario=scenarios[name];

        if(!scenario){
          return;
        }

        materialSelect.value=
          scenario.material;

        temperatureInput.value=String(
          scenario.temperature
        );

        pressureInput.value=String(
          scenario.pressure
        );

        waterInput.value=String(
          scenario.water
        );

        upliftInput.value=String(
          scenario.uplift
        );

        coolingInput.value=String(
          scenario.cooling
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

      var materialSelect=query(
        '[data-role="initial-material"]'
      );

      var temperatureInput=query(
        '[data-role="temperature"]'
      );

      var pressureInput=query(
        '[data-role="pressure"]'
      );

      var waterInput=query(
        '[data-role="water"]'
      );

      var upliftInput=query(
        '[data-role="uplift"]'
      );

      var coolingInput=query(
        '[data-role="cooling"]'
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

      var pressureValue=query(
        '[data-role="pressure-value"]'
      );

      var waterValue=query(
        '[data-role="water-value"]'
      );

      var upliftValue=query(
        '[data-role="uplift-value"]'
      );

      var coolingValue=query(
        '[data-role="cooling-value"]'
      );

      var materialValue=query(
        '[data-role="material-value"]'
      );

      var processValue=query(
        '[data-role="process-value"]'
      );

      var productValue=query(
        '[data-role="product-value"]'
      );

      var structureValue=query(
        '[data-role="structure-value"]'
      );

      if(
        !materialSelect ||
        !temperatureInput ||
        !pressureInput ||
        !waterInput ||
        !upliftInput ||
        !coolingInput ||
        !observationSelect ||
        !labelSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !result ||
        !canvas ||
        !temperatureValue ||
        !pressureValue ||
        !waterValue ||
        !upliftValue ||
        !coolingValue ||
        !materialValue ||
        !processValue ||
        !productValue ||
        !structureValue
      ){
        return;
      }

      var scenarioOrder=[
        'cooling',
        'sedimentation',
        'lithification',
        'metamorphism',
        'melting',
        'uplift'
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
        pressureInput,
        waterInput,
        upliftInput,
        coolingInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            clearScenarioSelection
          );
        }
      );

      materialSelect.addEventListener(
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
                'cooling';

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

export const GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_ROCK_CYCLE:
GeographyLabTemplate[] = [
  {
    id: 'geography-rock-cycle-three-rock-types',
    group: '⛰️ 地质作用与地貌演化',
    name: '岩石圈物质循环与三大岩石转化',
    emoji: '🪨',
    desc: '调节温度、压力、水分、抬升和冷却速度，观察岩浆岩、沉积岩、变质岩及岩浆之间的多路径转化。',
    params: [
      {
        key: 'initialMaterial',
        label: '初始物质',
        type: 'select',
        options: [
          {
            label: '岩浆',
            value: 'magma',
          },
          {
            label: '岩浆岩',
            value: 'igneous',
          },
          {
            label: '沉积物',
            value: 'sediment',
          },
          {
            label: '沉积岩',
            value: 'sedimentary',
          },
          {
            label: '变质岩',
            value: 'metamorphic',
          },
        ],
        defaultValue: 'magma',
      },
      {
        key: 'temperature',
        label: '地下温度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
        hint: '相对教学量，较高温度有利于变质或熔融。',
      },
      {
        key: 'pressure',
        label: '构造压力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 52,
        hint: '相对教学量，压力可促进压实胶结和变质作用。',
      },
      {
        key: 'surfaceWater',
        label: '地表水分',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
        hint: '水分增加通常会增强风化、侵蚀和物质搬运。',
      },
      {
        key: 'uplift',
        label: '地壳抬升强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 46,
        hint: '抬升可使深部岩石出露并进入地表风化循环。',
      },
      {
        key: 'coolingRate',
        label: '岩浆冷却速度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 42,
        hint: '快速冷却偏向细粒结构，缓慢冷却偏向粗粒晶体。',
      },
      {
        key: 'observationMode',
        label: '初始观察模式',
        type: 'select',
        options: [
          {
            label: '物质循环路径',
            value: 'cycle',
          },
          {
            label: '转化条件',
            value: 'conditions',
          },
          {
            label: '岩石结构',
            value: 'structure',
          },
        ],
        defaultValue: 'cycle',
      },
      {
        key: 'showLabels',
        label: '显示过程标注',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'automatic',
        label: '自动演示典型路径',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildRockCycleHTML,
  },
]
