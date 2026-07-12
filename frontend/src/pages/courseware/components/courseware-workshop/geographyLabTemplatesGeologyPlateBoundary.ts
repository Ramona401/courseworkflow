/**
 * geographyLabTemplatesGeologyPlateBoundary.ts
 *
 * 第36批B1：板块边界、地震与火山。
 *
 * 教学目标：
 * 1. 比较汇聚边界、张裂边界和转换边界的运动方向；
 * 2. 理解俯冲、碰撞、张裂和水平错动的基本过程；
 * 3. 观察板块运动速度、应力积累和岩浆活动之间的关系；
 * 4. 比较不同板块边界的地震深度、火山活动和典型地貌；
 * 5. 理解地震波由震源向外传播，震中位于震源正上方地表；
 * 6. 认识板块边界与地震、火山分布之间存在统计联系，
 *    但不能据简化模型预测具体灾害。
 *
 * 教学边界：
 * - 所有速度、压力、应力、震级和深度均为课堂相对教学量；
 * - 不表示任何真实断层、板块边界、火山或地震事件；
 * - 不用于地震预测、火山预警、建筑抗震、工程选址或应急决策；
 * - 真实地质过程还受岩性、温压、流体、构造历史和三维结构影响；
 * - 地震潜势与火山活跃度只是关系比较指标，不是发生概率。
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

function buildPlateBoundaryHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const requestedBoundary = stringValue(
    params,
    'boundaryType',
    'convergent',
  )

  const boundaryType = [
    'convergent',
    'divergent',
    'transform',
  ].includes(requestedBoundary)
    ? requestedBoundary
    : 'convergent'

  const requestedCrustContext = stringValue(
    params,
    'crustContext',
    'oceanic-continental',
  )

  const crustContext = [
    'oceanic-continental',
    'continental-continental',
    'oceanic-oceanic',
  ].includes(requestedCrustContext)
    ? requestedCrustContext
    : 'oceanic-continental'

  const plateSpeed = Math.max(
    1,
    Math.min(
      10,
      numberValue(params, 'plateSpeed', 6),
    ),
  )

  const stress = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'stress', 68),
    ),
  )

  const magmaPressure = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'magmaPressure', 62),
    ),
  )

  const crustThickness = Math.max(
    20,
    Math.min(
      70,
      numberValue(params, 'crustThickness', 42),
    ),
  )

  const showSeismicWaves = booleanValue(
    params,
    'showSeismicWaves',
    true,
  )

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
<div id="${rootId}" class="gl-plate-boundary-root">
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
        #FDE7D5 52%,
        #E0F2FE
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
        circle at 50% 18%,
        #FFFFFF 0%,
        #F8FAFC 53%,
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

    #${rootId} .gl-geology-canvas{
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
      font-size:12.5px;
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
      🌋
    </div>

    <div>
      <div class="gl-title">
        板块边界、地震与火山
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        比较汇聚、张裂和转换边界，观察震源、地震波与岩浆活动
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 不作灾害预测
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            板块边界类型
          </span>
        </div>

        <select data-role="boundary-type">
          <option
            value="convergent"
            ${boundaryType === 'convergent' ? 'selected' : ''}
          >
            汇聚边界
          </option>

          <option
            value="divergent"
            ${boundaryType === 'divergent' ? 'selected' : ''}
          >
            张裂边界
          </option>

          <option
            value="transform"
            ${boundaryType === 'transform' ? 'selected' : ''}
          >
            转换边界
          </option>
        </select>
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            地壳组合
          </span>
        </div>

        <select data-role="crust-context">
          <option
            value="oceanic-continental"
            ${crustContext === 'oceanic-continental' ? 'selected' : ''}
          >
            大洋板块—大陆板块
          </option>

          <option
            value="continental-continental"
            ${crustContext === 'continental-continental' ? 'selected' : ''}
          >
            大陆板块—大陆板块
          </option>

          <option
            value="oceanic-oceanic"
            ${crustContext === 'oceanic-oceanic' ? 'selected' : ''}
          >
            大洋板块—大洋板块
          </option>
        </select>
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            板块运动速度
          </span>

          <span
            class="gl-value"
            data-role="speed-value"
          ></span>
        </div>

        <input
          type="range"
          min="1"
          max="10"
          step="1"
          value="${shortNumber(plateSpeed)}"
          data-role="speed"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            构造应力积累
          </span>

          <span
            class="gl-value"
            data-role="stress-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(stress)}"
          data-role="stress"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            岩浆活动强度
          </span>

          <span
            class="gl-value"
            data-role="magma-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(magmaPressure)}"
          data-role="magma"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            地壳厚度
          </span>

          <span
            class="gl-value"
            data-role="thickness-value"
          ></span>
        </div>

        <input
          type="range"
          min="20"
          max="70"
          step="1"
          value="${shortNumber(crustThickness)}"
          data-role="thickness"
        />
      </div>

      <div class="gl-switch-row">
        <span>显示地震波</span>

        <input
          type="checkbox"
          data-role="wave-switch"
          ${showSeismicWaves ? 'checked' : ''}
        />
      </div>

      <div class="gl-switch-row">
        <span>显示构造标注</span>

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
        典型板块情境
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="subduction"
        >
          🌋 俯冲火山带
        </button>

        <button
          type="button"
          data-scenario="collision"
        >
          ⛰️ 大陆碰撞
        </button>

        <button
          type="button"
          data-scenario="ridge"
        >
          🌊 洋中脊张裂
        </button>

        <button
          type="button"
          data-scenario="rift"
        >
          🏜️ 大陆裂谷
        </button>

        <button
          type="button"
          data-scenario="transform"
        >
          ↔️ 转换断层
        </button>

        <button
          type="button"
          data-role="release"
        >
          ⚡ 模拟应力释放
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-geology-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="板块边界地震与火山教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="motion-value"></strong>
          <span>相对运动</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="earthquake-value"></strong>
          <span>地震活动特征</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="volcano-value"></strong>
          <span>火山活动程度</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="landform-value"></strong>
          <span>典型构造地貌</span>
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

        var headLength=12;

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=width || 3;
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

      function drawWave(
        context,
        x,
        y,
        radius,
        opacity
      ){
        context.save();
        context.globalAlpha=opacity;
        context.strokeStyle='#F97316';
        context.lineWidth=2;

        context.beginPath();
        context.arc(
          x,
          y,
          radius,
          0,
          Math.PI*2
        );
        context.stroke();

        context.restore();
      }

      function drawVolcano(
        context,
        x,
        baseY,
        scale,
        activity,
        phase
      ){
        var mountainHeight=
          86*
          scale;

        var halfWidth=
          70*
          scale;

        context.save();

        var mountainGradient=
          context.createLinearGradient(
            x,
            baseY-mountainHeight,
            x,
            baseY
          );

        mountainGradient.addColorStop(
          0,
          '#6B4423'
        );

        mountainGradient.addColorStop(
          1,
          '#3F2B1D'
        );

        context.fillStyle=mountainGradient;

        context.beginPath();
        context.moveTo(
          x-halfWidth,
          baseY
        );
        context.lineTo(
          x-18*scale,
          baseY-mountainHeight+14*scale
        );
        context.lineTo(
          x,
          baseY-mountainHeight
        );
        context.lineTo(
          x+17*scale,
          baseY-mountainHeight+14*scale
        );
        context.lineTo(
          x+halfWidth,
          baseY
        );
        context.closePath();
        context.fill();

        context.fillStyle='#111827';
        context.beginPath();
        context.ellipse(
          x,
          baseY-mountainHeight+4*scale,
          17*scale,
          6*scale,
          0,
          0,
          Math.PI*2
        );
        context.fill();

        if(activity>12){
          var lavaHeight=
            22+
            activity*
            0.32;

          context.strokeStyle='#F97316';
          context.lineWidth=
            3+
            activity/35;
          context.lineCap='round';

          context.beginPath();
          context.moveTo(
            x,
            baseY-mountainHeight
          );
          context.lineTo(
            x+
            Math.sin(
              phase*Math.PI*2
            )*
            5,
            baseY-
            mountainHeight-
            lavaHeight
          );
          context.stroke();

          var particleCount=Math.round(
            activity/13
          );

          for(
            var index=0;
            index<particleCount;
            index+=1
          ){
            var angle=
              phase*
              Math.PI*
              2+
              index*
              1.37;

            var distance=
              15+
              index*
              5;

            var particleX=
              x+
              Math.cos(angle)*
              distance;

            var particleY=
              baseY-
              mountainHeight-
              17-
              Math.abs(
                Math.sin(angle)
              )*
              (
                20+
                activity*
                0.3
              );

            context.fillStyle=
              index%2===0
                ? '#F97316'
                : '#FACC15';

            context.beginPath();
            context.arc(
              particleX,
              particleY,
              2.5+
              index%3,
              0,
              Math.PI*2
            );
            context.fill();
          }

          context.globalAlpha=
            0.18+
            activity/180;

          context.fillStyle='#64748B';

          context.beginPath();
          context.arc(
            x-8,
            baseY-mountainHeight-45,
            15+activity*0.08,
            0,
            Math.PI*2
          );

          context.arc(
            x+13,
            baseY-mountainHeight-53,
            18+activity*0.09,
            0,
            Math.PI*2
          );

          context.arc(
            x+31,
            baseY-mountainHeight-42,
            14+activity*0.07,
            0,
            Math.PI*2
          );

          context.fill();
        }

        context.restore();
      }

      function drawMountainRange(
        context,
        centerX,
        baseY,
        compression,
        thickness
      ){
        var mountainCount=7;
        var heightFactor=
          0.55+
          compression/120+
          (
            thickness-20
          )/
          100;

        context.save();

        for(
          var index=0;
          index<mountainCount;
          index+=1
        ){
          var x=
            centerX-
            150+
            index*
            50;

          var height=
            (
              44+
              (
                index%3
              )*
              19
            )*
            heightFactor;

          var gradient=
            context.createLinearGradient(
              x,
              baseY-height,
              x,
              baseY
            );

          gradient.addColorStop(
            0,
            '#D1D5DB'
          );

          gradient.addColorStop(
            0.38,
            '#8B6A45'
          );

          gradient.addColorStop(
            1,
            '#5B3C22'
          );

          context.fillStyle=gradient;

          context.beginPath();
          context.moveTo(
            x-42,
            baseY
          );
          context.lineTo(
            x,
            baseY-height
          );
          context.lineTo(
            x+43,
            baseY
          );
          context.closePath();
          context.fill();

          if(height>72){
            context.fillStyle='#F8FAFC';

            context.beginPath();
            context.moveTo(
              x-10,
              baseY-height+18
            );
            context.lineTo(
              x,
              baseY-height
            );
            context.lineTo(
              x+12,
              baseY-height+21
            );
            context.closePath();
            context.fill();
          }
        }

        context.restore();
      }

      function readState(){
        return {
          boundaryType:
            boundarySelect.value,
          crustContext:
            crustSelect.value,
          speed:Number(
            speedInput.value
          ),
          stress:Number(
            stressInput.value
          ),
          magma:Number(
            magmaInput.value
          ),
          thickness:Number(
            thicknessInput.value
          )
        };
      }

      function calculate(values){
        var speedRatio=
          values.speed/
          10;

        var stressRatio=
          values.stress/
          100;

        var magmaRatio=
          values.magma/
          100;

        var thicknessRatio=
          (
            values.thickness-20
          )/
          50;

        var motionLabel='';
        var earthquakeDepth='';
        var landform='';
        var process='';
        var earthquakePotential=0;
        var volcanoPotential=0;
        var focusDepth=45;
        var magnitude=0;

        if(values.boundaryType==='convergent'){
          motionLabel='相向运动';

          earthquakePotential=clamp(
            30+
            stressRatio*48+
            speedRatio*24,
            0,
            100
          );

          if(
            values.crustContext===
            'continental-continental'
          ){
            earthquakeDepth='浅—中源地震';
            landform='褶皱山系';
            process='大陆碰撞与地壳增厚';

            volcanoPotential=clamp(
              6+
              magmaRatio*20+
              speedRatio*8,
              0,
              42
            );

            focusDepth=
              34+
              stressRatio*
              55;

            magnitude=clamp(
              3.7+
              stressRatio*
              3.2+
              speedRatio*
              1.0,
              3.5,
              8.3
            );
          }else{
            earthquakeDepth='浅—深源地震';
            landform=
              values.crustContext===
              'oceanic-oceanic'
                ? '海沟与岛弧'
                : '海沟与火山弧';

            process='俯冲、脱水与岩浆上升';

            volcanoPotential=clamp(
              25+
              magmaRatio*52+
              speedRatio*19,
              0,
              100
            );

            focusDepth=
              55+
              stressRatio*
              175+
              speedRatio*
              50;

            magnitude=clamp(
              3.8+
              stressRatio*
              3.0+
              speedRatio*
              1.1,
              3.5,
              8.5
            );
          }
        }else if(
          values.boundaryType==='divergent'
        ){
          motionLabel='背向张裂';
          earthquakeDepth='浅源地震';

          earthquakePotential=clamp(
            18+
            stressRatio*30+
            speedRatio*21,
            0,
            78
          );

          volcanoPotential=clamp(
            22+
            magmaRatio*50+
            speedRatio*20,
            0,
            100
          );

          focusDepth=
            12+
            stressRatio*
            28;

          magnitude=clamp(
            2.8+
            stressRatio*
            2.0+
            speedRatio*
            0.8,
            2.5,
            6.2
          );

          if(
            values.crustContext===
            'continental-continental'
          ){
            landform='大陆裂谷';
            process='地壳拉张、断陷与岩浆上涌';
          }else{
            landform='洋中脊';
            process='海底扩张与新洋壳形成';
          }
        }else{
          motionLabel='水平错动';
          earthquakeDepth='浅源地震';
          landform='转换断层';
          process='板块沿断层水平剪切';

          earthquakePotential=clamp(
            32+
            stressRatio*52+
            speedRatio*20,
            0,
            100
          );

          volcanoPotential=clamp(
            2+
            magmaRatio*9,
            0,
            18
          );

          focusDepth=
            8+
            stressRatio*
            25;

          magnitude=clamp(
            3.2+
            stressRatio*
            3.1+
            speedRatio*
            0.9,
            3,
            8.1
          );
        }

        var strainState=
          values.stress>=78
            ? '高应力积累'
            : values.stress>=42
              ? '中等应力积累'
              : '低应力积累';

        var volcanoState=
          volcanoPotential>=72
            ? '较强'
            : volcanoPotential>=38
              ? '中等'
              : '较弱';

        var earthquakeState=
          earthquakePotential>=76
            ? '较强'
            : earthquakePotential>=42
              ? '中等'
              : '较弱';

        return {
          motionLabel:motionLabel,
          earthquakeDepth:earthquakeDepth,
          landform:landform,
          process:process,
          earthquakePotential:
            earthquakePotential,
          volcanoPotential:
            volcanoPotential,
          focusDepth:focusDepth,
          magnitude:magnitude,
          strainState:strainState,
          volcanoState:volcanoState,
          earthquakeState:earthquakeState,
          thicknessRatio:thicknessRatio
        };
      }

      function describe(values,model){
        if(
          values.boundaryType==='convergent' &&
          values.crustContext===
          'continental-continental'
        ){
          return '两个大陆板块汇聚时，密度相近的大陆地壳不易整体俯冲，'+
            '地壳受到挤压、缩短和增厚，形成褶皱山系。'+
            '地震活动可以较强，但典型火山弧通常不如大洋板块俯冲明显。';
        }

        if(values.boundaryType==='convergent'){
          return '较致密的大洋板块向另一板块下方俯冲，形成海沟；'+
            '俯冲带可出现由浅到深的震源分布，板片脱水促进上覆地幔熔融，'+
            '岩浆上升形成火山弧。模型不代表具体火山或地震预测。';
        }

        if(values.boundaryType==='divergent'){
          return values.crustContext===
            'continental-continental'
            ? '大陆地壳受到拉张后发生断裂和断陷，可形成裂谷。'+
              '软流圈物质上涌减压熔融，常伴随浅源地震和岩浆活动。'
            : '板块在洋中脊两侧背向运动，地幔物质上涌并形成新洋壳。'+
              '地震通常较浅，火山活动沿张裂轴分布。';
        }

        return '转换边界两侧板块沿断层水平错动。断层可能因摩擦暂时锁定，'+
          '应力持续积累后突然释放形成浅源地震；'+
          '由于没有典型俯冲或大规模减压熔融，火山活动通常较弱。';
      }

      function drawConvergent(
        context,
        values,
        model,
        area
      ){
        var oceanicContinental=
          values.crustContext===
          'oceanic-continental';

        var continentalCollision=
          values.crustContext===
          'continental-continental';

        var oceanicOceanic=
          values.crustContext===
          'oceanic-oceanic';

        var centerX=
          area.x+
          area.width*
          0.5;

        var surfaceY=
          area.y+
          122;

        var crustHeight=
          34+
          model.thicknessRatio*
          34;

        if(continentalCollision){
          context.fillStyle='#B08958';

          context.beginPath();
          context.moveTo(
            area.x,
            surfaceY
          );
          context.lineTo(
            centerX-28,
            surfaceY
          );
          context.lineTo(
            centerX+15,
            surfaceY+
            crustHeight+
            32
          );
          context.lineTo(
            area.x,
            surfaceY+
            crustHeight+
            7
          );
          context.closePath();
          context.fill();

          context.fillStyle='#9A7147';

          context.beginPath();
          context.moveTo(
            area.x+
            area.width,
            surfaceY
          );
          context.lineTo(
            centerX+28,
            surfaceY
          );
          context.lineTo(
            centerX-15,
            surfaceY+
            crustHeight+
            32
          );
          context.lineTo(
            area.x+
            area.width,
            surfaceY+
            crustHeight+
            7
          );
          context.closePath();
          context.fill();

          drawMountainRange(
            context,
            centerX,
            surfaceY+
            3,
            values.stress,
            values.thickness
          );

          drawArrow(
            context,
            area.x+115,
            surfaceY-40,
            centerX-78,
            surfaceY-40,
            '#B45309',
            4
          );

          drawArrow(
            context,
            area.x+
            area.width-
            115,
            surfaceY-40,
            centerX+78,
            surfaceY-40,
            '#B45309',
            4
          );

          state.focusX=centerX;
          state.focusY=
            surfaceY+
            crustHeight+
            23;
          state.epicenterX=centerX;
          state.epicenterY=
            surfaceY-5;

          return;
        }

        var oceanOnLeft=
          oceanicContinental ||
          oceanicOceanic;

        context.fillStyle='#0EA5E9';
        context.fillRect(
          area.x,
          surfaceY-32,
          area.width*
          0.48,
          32
        );

        context.fillStyle='#DBEAFE';
        context.fillRect(
          centerX,
          surfaceY-30,
          area.width*
          0.5,
          30
        );

        context.fillStyle='#4B6476';

        context.beginPath();
        context.moveTo(
          area.x,
          surfaceY
        );
        context.lineTo(
          centerX-8,
          surfaceY
        );
        context.lineTo(
          centerX+
          190,
          surfaceY+
          165
        );
        context.lineTo(
          centerX+
          160,
          surfaceY+
          185
        );
        context.lineTo(
          centerX-22,
          surfaceY+
          crustHeight
        );
        context.lineTo(
          area.x,
          surfaceY+
          crustHeight
        );
        context.closePath();
        context.fill();

        context.fillStyle=
          oceanicOceanic
            ? '#526D7A'
            : '#A97845';

        context.beginPath();
        context.moveTo(
          centerX+6,
          surfaceY
        );
        context.lineTo(
          area.x+
          area.width,
          surfaceY
        );
        context.lineTo(
          area.x+
          area.width,
          surfaceY+
          crustHeight+
          12
        );
        context.lineTo(
          centerX+40,
          surfaceY+
          crustHeight+
          14
        );
        context.closePath();
        context.fill();

        context.fillStyle='#0F172A';

        context.beginPath();
        context.moveTo(
          centerX-34,
          surfaceY-2
        );
        context.lineTo(
          centerX-3,
          surfaceY+20
        );
        context.lineTo(
          centerX+17,
          surfaceY
        );
        context.closePath();
        context.fill();

        var volcanoX=
          centerX+
          155;

        drawVolcano(
          context,
          volcanoX,
          surfaceY+2,
          oceanicOceanic
            ? 0.78
            : 0.96,
          model.volcanoPotential,
          state.phase
        );

        context.fillStyle=
          'rgba(249,115,22,0.78)';

        context.beginPath();
        context.ellipse(
          volcanoX,
          surfaceY+
          crustHeight+
          78,
          34+
          model.volcanoPotential*
          0.15,
          18+
          model.volcanoPotential*
          0.09,
          0,
          0,
          Math.PI*2
        );
        context.fill();

        context.strokeStyle=
          'rgba(249,115,22,0.76)';

        context.lineWidth=4;

        context.beginPath();
        context.moveTo(
          volcanoX,
          surfaceY+
          crustHeight+
          68
        );
        context.lineTo(
          volcanoX,
          surfaceY-
          38
        );
        context.stroke();

        drawArrow(
          context,
          area.x+100,
          surfaceY-42,
          centerX-64,
          surfaceY-42,
          '#0369A1',
          4
        );

        drawArrow(
          context,
          area.x+
          area.width-
          100,
          surfaceY-42,
          centerX+65,
          surfaceY-42,
          '#B45309',
          4
        );

        state.focusX=
          centerX+
          88;

        state.focusY=
          surfaceY+
          102;

        state.epicenterX=
          centerX+
          12;

        state.epicenterY=
          surfaceY-4;
      }

      function drawDivergent(
        context,
        values,
        model,
        area
      ){
        var continental=
          values.crustContext===
          'continental-continental';

        var centerX=
          area.x+
          area.width*
          0.5;

        var surfaceY=
          area.y+
          130;

        var crustHeight=
          34+
          model.thicknessRatio*
          31;

        if(continental){
          context.fillStyle='#A97845';

          context.beginPath();
          context.moveTo(
            area.x,
            surfaceY-24
          );
          context.lineTo(
            centerX-55,
            surfaceY
          );
          context.lineTo(
            centerX-10,
            surfaceY+
            49
          );
          context.lineTo(
            area.x,
            surfaceY+
            crustHeight
          );
          context.closePath();
          context.fill();

          context.fillStyle='#95653E';

          context.beginPath();
          context.moveTo(
            area.x+
            area.width,
            surfaceY-24
          );
          context.lineTo(
            centerX+55,
            surfaceY
          );
          context.lineTo(
            centerX+10,
            surfaceY+
            49
          );
          context.lineTo(
            area.x+
            area.width,
            surfaceY+
            crustHeight
          );
          context.closePath();
          context.fill();

          context.fillStyle='#7C2D12';

          context.beginPath();
          context.moveTo(
            centerX-55,
            surfaceY
          );
          context.lineTo(
            centerX-10,
            surfaceY+49
          );
          context.lineTo(
            centerX+10,
            surfaceY+49
          );
          context.lineTo(
            centerX+55,
            surfaceY
          );
          context.closePath();
          context.fill();
        }else{
          context.fillStyle='#0EA5E9';

          context.fillRect(
            area.x,
            surfaceY-42,
            area.width,
            42
          );

          context.fillStyle='#4B6476';

          context.beginPath();
          context.moveTo(
            area.x,
            surfaceY
          );
          context.lineTo(
            centerX-42,
            surfaceY
          );
          context.lineTo(
            centerX-10,
            surfaceY+50
          );
          context.lineTo(
            area.x,
            surfaceY+
            crustHeight
          );
          context.closePath();
          context.fill();

          context.beginPath();
          context.moveTo(
            area.x+
            area.width,
            surfaceY
          );
          context.lineTo(
            centerX+42,
            surfaceY
          );
          context.lineTo(
            centerX+10,
            surfaceY+50
          );
          context.lineTo(
            area.x+
            area.width,
            surfaceY+
            crustHeight
          );
          context.closePath();
          context.fill();

          context.fillStyle='#7C2D12';

          context.beginPath();
          context.moveTo(
            centerX-50,
            surfaceY
          );
          context.lineTo(
            centerX,
            surfaceY-46
          );
          context.lineTo(
            centerX+50,
            surfaceY
          );
          context.lineTo(
            centerX+11,
            surfaceY+48
          );
          context.lineTo(
            centerX-11,
            surfaceY+48
          );
          context.closePath();
          context.fill();
        }

        var magmaGradient=
          context.createLinearGradient(
            centerX,
            surfaceY+130,
            centerX,
            surfaceY-60
          );

        magmaGradient.addColorStop(
          0,
          'rgba(239,68,68,0.18)'
        );

        magmaGradient.addColorStop(
          1,
          'rgba(249,115,22,0.94)'
        );

        context.fillStyle=magmaGradient;

        context.beginPath();
        context.moveTo(
          centerX-58,
          surfaceY+160
        );
        context.quadraticCurveTo(
          centerX-18,
          surfaceY+65,
          centerX-7,
          surfaceY-30
        );
        context.lineTo(
          centerX+7,
          surfaceY-30
        );
        context.quadraticCurveTo(
          centerX+18,
          surfaceY+65,
          centerX+58,
          surfaceY+160
        );
        context.closePath();
        context.fill();

        drawArrow(
          context,
          centerX-62,
          surfaceY-50,
          area.x+122,
          surfaceY-50,
          '#0369A1',
          4
        );

        drawArrow(
          context,
          centerX+62,
          surfaceY-50,
          area.x+
          area.width-
          122,
          surfaceY-50,
          '#0369A1',
          4
        );

        drawArrow(
          context,
          centerX,
          surfaceY+120,
          centerX,
          surfaceY-30,
          '#F97316',
          4
        );

        state.focusX=
          centerX+
          17;

        state.focusY=
          surfaceY+
          26;

        state.epicenterX=
          centerX+
          17;

        state.epicenterY=
          surfaceY-10;
      }

      function drawTransform(
        context,
        values,
        model,
        area
      ){
        var centerX=
          area.x+
          area.width*
          0.5;

        var surfaceY=
          area.y+
          133;

        var crustHeight=
          52+
          model.thicknessRatio*
          34;

        context.fillStyle='#A97845';

        context.fillRect(
          area.x,
          surfaceY,
          area.width*
          0.5-
          8,
          crustHeight
        );

        context.fillStyle='#8C603B';

        context.fillRect(
          centerX+8,
          surfaceY,
          area.width*
          0.5-
          8,
          crustHeight
        );

        context.strokeStyle='#1F2937';
        context.lineWidth=5;
        context.setLineDash([12,7]);

        context.beginPath();
        context.moveTo(
          centerX,
          surfaceY-52
        );
        context.lineTo(
          centerX,
          surfaceY+
          crustHeight+
          100
        );
        context.stroke();

        context.setLineDash([]);

        var offset=
          24+
          values.stress*
          0.22;

        context.strokeStyle='#6B4423';
        context.lineWidth=7;

        context.beginPath();
        context.moveTo(
          area.x+80,
          surfaceY-4
        );
        context.lineTo(
          centerX-offset,
          surfaceY-4
        );
        context.stroke();

        context.beginPath();
        context.moveTo(
          centerX+offset,
          surfaceY-4
        );
        context.lineTo(
          area.x+
          area.width-
          80,
          surfaceY-4
        );
        context.stroke();

        drawArrow(
          context,
          centerX-58,
          surfaceY-54,
          area.x+120,
          surfaceY-54,
          '#7C3AED',
          4
        );

        drawArrow(
          context,
          centerX+58,
          surfaceY+50,
          area.x+
          area.width-
          120,
          surfaceY+50,
          '#7C3AED',
          4
        );

        context.save();
        context.globalAlpha=
          0.3+
          values.stress/
          145;

        context.strokeStyle='#F97316';
        context.lineWidth=3;

        for(
          var index=0;
          index<7;
          index+=1
        ){
          var sparkY=
            surfaceY-
            25+
            index*
            17;

          context.beginPath();
          context.moveTo(
            centerX-11,
            sparkY
          );
          context.lineTo(
            centerX+11,
            sparkY+8
          );
          context.stroke();
        }

        context.restore();

        state.focusX=centerX;
        state.focusY=
          surfaceY+
          crustHeight*
          0.48;

        state.epicenterX=centerX;
        state.epicenterY=
          surfaceY-6;
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

        speedValue.textContent=
          values.speed+
          ' 级';

        stressValue.textContent=
          Math.round(
            values.stress
          )+
          '%';

        magmaValue.textContent=
          Math.round(
            values.magma
          )+
          '%';

        thicknessValue.textContent=
          Math.round(
            values.thickness
          )+
          ' km';

        motionValue.textContent=
          model.motionLabel;

        earthquakeValue.textContent=
          model.earthquakeDepth;

        volcanoValue.textContent=
          model.volcanoState;

        landformValue.textContent=
          model.landform;

        result.textContent=
          model.process+
          '。'+
          describe(
            values,
            model
          )+
          ' 当前地震活动关系指标为'+
          Math.round(
            model.earthquakePotential
          )+
          '%，火山活动关系指标为'+
          Math.round(
            model.volcanoPotential
          )+
          '%；这些指标不是灾害发生概率。';

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
          '#DBEAFE'
        );

        background.addColorStop(
          0.46,
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
          boundaryName(
            values.boundaryType
          )+
          ' · '+
          crustName(
            values.crustContext
          ),
          40,
          43,
          14,
          '#78350F',
          850,
          'left'
        );

        drawText(
          context,
          '剖面方向与比例均为课堂示意',
          780,
          43,
          9.5,
          '#64748B',
          650,
          'right'
        );

        var area={
          x:61,
          y:68,
          width:572,
          height:270
        };

        var mantleGradient=
          context.createLinearGradient(
            0,
            area.y+120,
            0,
            area.y+
            area.height
          );

        mantleGradient.addColorStop(
          0,
          '#F6B26B'
        );

        mantleGradient.addColorStop(
          0.55,
          '#D97706'
        );

        mantleGradient.addColorStop(
          1,
          '#9A3412'
        );

        fillRoundedRect(
          context,
          area.x,
          area.y,
          area.width,
          area.height,
          18,
          mantleGradient,
          '#B45309'
        );

        context.fillStyle=
          'rgba(127,29,29,0.18)';

        for(
          var flowIndex=0;
          flowIndex<6;
          flowIndex+=1
        ){
          context.beginPath();
          context.ellipse(
            area.x+
            80+
            flowIndex*
            90,
            area.y+
            area.height-
            43-
            (
              flowIndex%2
            )*
            18,
            49,
            17,
            0,
            0,
            Math.PI*2
          );
          context.fill();
        }

        state.focusX=
          area.x+
          area.width*
          0.5;

        state.focusY=
          area.y+
          190;

        state.epicenterX=
          state.focusX;

        state.epicenterY=
          area.y+
          78;

        if(values.boundaryType==='convergent'){
          drawConvergent(
            context,
            values,
            model,
            area
          );
        }else if(
          values.boundaryType==='divergent'
        ){
          drawDivergent(
            context,
            values,
            model,
            area
          );
        }else{
          drawTransform(
            context,
            values,
            model,
            area
          );
        }

        context.fillStyle='#FACC15';
        context.strokeStyle='#DC2626';
        context.lineWidth=2.5;

        context.beginPath();

        for(
          var ray=0;
          ray<16;
          ray+=1
        ){
          var angle=
            ray/
            16*
            Math.PI*
            2;

          var inner=
            ray%2===0
              ? 6
              : 12;

          var outer=
            ray%2===0
              ? 17
              : 9;

          var radius=
            ray%2===0
              ? outer
              : inner;

          var x=
            state.focusX+
            Math.cos(angle)*
            radius;

          var y=
            state.focusY+
            Math.sin(angle)*
            radius;

          if(ray===0){
            context.moveTo(x,y);
          }else{
            context.lineTo(x,y);
          }
        }

        context.closePath();
        context.fill();
        context.stroke();

        if(waveSwitch.checked){
          var waveBase=
            18+
            state.phase*
            112;

          for(
            var waveIndex=0;
            waveIndex<4;
            waveIndex+=1
          ){
            var radius=
              (
                waveBase+
                waveIndex*
                28
              )%
              135;

            drawWave(
              context,
              state.focusX,
              state.focusY,
              radius,
              clamp(
                1-
                radius/
                150,
                0.08,
                0.72
              )
            );
          }
        }

        context.save();
        context.strokeStyle='#DC2626';
        context.lineWidth=1.5;
        context.setLineDash([6,5]);

        context.beginPath();
        context.moveTo(
          state.focusX,
          state.focusY
        );
        context.lineTo(
          state.epicenterX,
          state.epicenterY
        );
        context.stroke();

        context.restore();

        if(labelSwitch.checked){
          fillRoundedRect(
            context,
            state.focusX+13,
            state.focusY-12,
            83,
            25,
            12,
            'rgba(254,226,226,0.92)',
            '#FCA5A5'
          );

          drawText(
            context,
            '震源',
            state.focusX+54,
            state.focusY+1,
            10,
            '#B91C1C',
            850,
            'center'
          );

          fillRoundedRect(
            context,
            state.epicenterX-42,
            state.epicenterY-37,
            84,
            25,
            12,
            'rgba(255,255,255,0.92)',
            '#FCA5A5'
          );

          drawText(
            context,
            '震中',
            state.epicenterX,
            state.epicenterY-24,
            10,
            '#B91C1C',
            850,
            'center'
          );

          drawText(
            context,
            '地幔与软流圈',
            area.x+
            area.width-
            20,
            area.y+
            area.height-
            24,
            10,
            '#7C2D12',
            800,
            'right'
          );
        }

        fillRoundedRect(
          context,
          655,
          70,
          128,
          264,
          14,
          '#F8FAFC',
          '#F1D6A5'
        );

        drawText(
          context,
          '构造关系',
          674,
          94,
          12,
          '#78350F',
          850,
          'left'
        );

        var rows=[
          {
            label:'应力积累',
            value:values.stress,
            color:'#7C3AED'
          },
          {
            label:'地震活动',
            value:
              model.earthquakePotential,
            color:'#DC2626'
          },
          {
            label:'岩浆活动',
            value:
              model.volcanoPotential,
            color:'#F97316'
          },
          {
            label:'运动速度',
            value:
              values.speed*
              10,
            color:'#0369A1'
          }
        ];

        rows.forEach(
          function(item,index){
            var rowY=
              127+
              index*
              46;

            drawText(
              context,
              item.label,
              674,
              rowY,
              9.5,
              '#475569',
              750,
              'left'
            );

            context.fillStyle='#E2E8F0';

            context.fillRect(
              674,
              rowY+12,
              86,
              8
            );

            context.fillStyle=item.color;

            context.fillRect(
              674,
              rowY+12,
              clamp(
                item.value/
                100*
                86,
                3,
                86
              ),
              8
            );

            drawText(
              context,
              Math.round(
                item.value
              )+
              '%',
              766,
              rowY+16,
              8.5,
              item.color,
              800,
              'right'
            );
          }
        );

        drawText(
          context,
          '示意震级',
          674,
          307,
          9.5,
          '#64748B',
          700,
          'left'
        );

        drawText(
          context,
          model.magnitude.toFixed(1),
          765,
          307,
          18,
          '#B91C1C',
          900,
          'right'
        );

        drawText(
          context,
          '震源深度示意：'+
          Math.round(
            model.focusDepth
          )+
          ' km',
          410,
          414,
          9.5,
          '#64748B',
          650,
          'center'
        );
      }

      function boundaryName(value){
        if(value==='divergent'){
          return '张裂边界';
        }

        if(value==='transform'){
          return '转换边界';
        }

        return '汇聚边界';
      }

      function crustName(value){
        if(value==='continental-continental'){
          return '大陆—大陆';
        }

        if(value==='oceanic-oceanic'){
          return '大洋—大洋';
        }

        return '大洋—大陆';
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
          subduction:{
            boundaryType:'convergent',
            crustContext:'oceanic-continental',
            speed:7,
            stress:76,
            magma:82,
            thickness:43
          },
          collision:{
            boundaryType:'convergent',
            crustContext:'continental-continental',
            speed:5,
            stress:88,
            magma:22,
            thickness:66
          },
          ridge:{
            boundaryType:'divergent',
            crustContext:'oceanic-oceanic',
            speed:6,
            stress:45,
            magma:78,
            thickness:24
          },
          rift:{
            boundaryType:'divergent',
            crustContext:'continental-continental',
            speed:4,
            stress:58,
            magma:64,
            thickness:48
          },
          transform:{
            boundaryType:'transform',
            crustContext:'continental-continental',
            speed:7,
            stress:91,
            magma:10,
            thickness:39
          }
        };

        var scenario=scenarios[name];

        if(!scenario){
          return;
        }

        boundarySelect.value=
          scenario.boundaryType;

        crustSelect.value=
          scenario.crustContext;

        speedInput.value=String(
          scenario.speed
        );

        stressInput.value=String(
          scenario.stress
        );

        magmaInput.value=String(
          scenario.magma
        );

        thicknessInput.value=String(
          scenario.thickness
        );

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
          3100%
          1;

        render();

        state.raf=
          requestAnimationFrame(
            animate
          );
      }

      var boundarySelect=query(
        '[data-role="boundary-type"]'
      );

      var crustSelect=query(
        '[data-role="crust-context"]'
      );

      var speedInput=query(
        '[data-role="speed"]'
      );

      var stressInput=query(
        '[data-role="stress"]'
      );

      var magmaInput=query(
        '[data-role="magma"]'
      );

      var thicknessInput=query(
        '[data-role="thickness"]'
      );

      var waveSwitch=query(
        '[data-role="wave-switch"]'
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

      var releaseButton=query(
        '[data-role="release"]'
      );

      var result=query(
        '[data-role="result"]'
      );

      var canvas=query(
        '[data-role="canvas"]'
      );

      var speedValue=query(
        '[data-role="speed-value"]'
      );

      var stressValue=query(
        '[data-role="stress-value"]'
      );

      var magmaValue=query(
        '[data-role="magma-value"]'
      );

      var thicknessValue=query(
        '[data-role="thickness-value"]'
      );

      var motionValue=query(
        '[data-role="motion-value"]'
      );

      var earthquakeValue=query(
        '[data-role="earthquake-value"]'
      );

      var volcanoValue=query(
        '[data-role="volcano-value"]'
      );

      var landformValue=query(
        '[data-role="landform-value"]'
      );

      if(
        !boundarySelect ||
        !crustSelect ||
        !speedInput ||
        !stressInput ||
        !magmaInput ||
        !thicknessInput ||
        !waveSwitch ||
        !labelSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !releaseButton ||
        !result ||
        !canvas ||
        !speedValue ||
        !stressValue ||
        !magmaValue ||
        !thicknessValue ||
        !motionValue ||
        !earthquakeValue ||
        !volcanoValue ||
        !landformValue
      ){
        return;
      }

      var scenarioOrder=[
        'subduction',
        'collision',
        'ridge',
        'rift',
        'transform'
      ];

      var state={
        phase:0,
        startedAt:0,
        raf:0,
        timer:null,
        scenarioIndex:-1,
        focusX:400,
        focusY:235,
        epicenterX:400,
        epicenterY:150
      };

      [
        speedInput,
        stressInput,
        magmaInput,
        thicknessInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            clearScenarioSelection
          );
        }
      );

      boundarySelect.addEventListener(
        'change',
        clearScenarioSelection
      );

      crustSelect.addEventListener(
        'change',
        clearScenarioSelection
      );

      waveSwitch.addEventListener(
        'change',
        render
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
                'subduction';

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

      releaseButton.addEventListener(
        'click',
        function(){
          var currentStress=Number(
            stressInput.value
          );

          stressInput.value=String(
            Math.max(
              12,
              Math.round(
                currentStress*
                0.28
              )
            )
          );

          state.phase=0;

          Array.prototype.forEach.call(
            scenarioButtons,
            function(button){
              button.classList.remove(
                'active'
              );
            }
          );

          render();

          result.textContent=
            '模拟应力释放后，构造应力指标暂时下降。'+
            '真实地震后的应力变化十分复杂，'+
            '不能据此推断下一次地震时间、位置或规模。';
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

export const GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_PLATE_BOUNDARY:
GeographyLabTemplate[] = [
  {
    id: 'geography-plate-boundary-earthquake-volcano',
    group: '⛰️ 地质作用与地貌演化',
    name: '板块边界、地震与火山',
    emoji: '🌋',
    desc: '切换汇聚、张裂和转换边界，比较板块运动、震源深度、地震活动、火山活动与构造地貌。',
    params: [
      {
        key: 'boundaryType',
        label: '初始板块边界',
        type: 'select',
        options: [
          {
            label: '汇聚边界',
            value: 'convergent',
          },
          {
            label: '张裂边界',
            value: 'divergent',
          },
          {
            label: '转换边界',
            value: 'transform',
          },
        ],
        defaultValue: 'convergent',
      },
      {
        key: 'crustContext',
        label: '初始地壳组合',
        type: 'select',
        options: [
          {
            label: '大洋板块—大陆板块',
            value: 'oceanic-continental',
          },
          {
            label: '大陆板块—大陆板块',
            value: 'continental-continental',
          },
          {
            label: '大洋板块—大洋板块',
            value: 'oceanic-oceanic',
          },
        ],
        defaultValue: 'oceanic-continental',
      },
      {
        key: 'plateSpeed',
        label: '板块运动速度',
        type: 'number',
        min: 1,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '教学等级，数值越大表示相对运动越快。',
      },
      {
        key: 'stress',
        label: '构造应力积累',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
        hint: '用于比较断层锁定和应力积累，不是地震发生概率。',
      },
      {
        key: 'magmaPressure',
        label: '岩浆活动强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
        hint: '表示岩浆生成、上升和喷发活动的相对强弱。',
      },
      {
        key: 'crustThickness',
        label: '地壳厚度',
        type: 'number',
        min: 20,
        max: 70,
        step: 1,
        defaultValue: 42,
        hint: '只改变教学剖面中的地壳厚度与地形表现。',
      },
      {
        key: 'showSeismicWaves',
        label: '显示地震波',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showLabels',
        label: '显示构造标注',
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
    buildHTML: buildPlateBoundaryHTML,
  },
]
