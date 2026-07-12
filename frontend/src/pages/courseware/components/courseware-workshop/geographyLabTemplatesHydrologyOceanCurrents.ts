/**
 * geographyLabTemplatesHydrologyOceanCurrents.ts
 *
 * 第35批B3：洋流分布及其对气候、渔场、航行和污染扩散的影响。
 *
 * 教学目标：
 * 1. 区分暖流和寒流，理解洋流相对于流经海区水温的性质；
 * 2. 观察北半球和南半球副热带环流方向差异；
 * 3. 理解暖流增温增湿、寒流降温减湿的相对影响；
 * 4. 理解寒暖流交汇和上升流对渔场形成的作用；
 * 5. 比较顺流、逆流对教学模型中航行时间和能耗的影响；
 * 6. 观察污染物可随洋流扩散，认识跨区域海洋环境联系。
 *
 * 教学边界：
 * - 本模板为高度简化的海盆环流课堂模型；
 * - 洋流位置、流速、温差、航程和扩散范围均为相对教学量；
 * - 不表示真实海区、港口、航线、渔场或污染事故；
 * - 不用于航海导航、捕捞决策、海上搜救、污染处置或天气预报；
 * - 真实洋流还受盛行风、地转偏向力、海陆轮廓、密度差异等影响。
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

function buildOceanCurrentsHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const requestedHemisphere = stringValue(
    params,
    'hemisphere',
    'north',
  )

  const hemisphere = [
    'north',
    'south',
  ].includes(requestedHemisphere)
    ? requestedHemisphere
    : 'north'

  const currentStrength = Math.max(
    20,
    Math.min(
      100,
      numberValue(params, 'currentStrength', 68),
    ),
  )

  const temperatureContrast = Math.max(
    2,
    Math.min(
      12,
      numberValue(params, 'temperatureContrast', 7),
    ),
  )

  const upwelling = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'upwelling', 55),
    ),
  )

  const nutrientLevel = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'nutrientLevel', 60),
    ),
  )

  const requestedObservation = stringValue(
    params,
    'observationMode',
    'climate',
  )

  const observationMode = [
    'climate',
    'fishery',
    'navigation',
    'pollution',
  ].includes(requestedObservation)
    ? requestedObservation
    : 'climate'

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
<div id="${rootId}" class="gl-ocean-current-root">
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
        circle at 48% 20%,
        #FFFFFF 0%,
        #F8FAFC 55%,
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

    #${rootId} .gl-ocean-canvas{
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
      border:1px solid #BAE6FD;
      box-shadow:0 5px 15px rgba(15,73,71,0.08);
      text-align:center;
    }

    #${rootId} .gl-summary-card strong{
      display:block;
      color:#0369A1;
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
      🌐
    </div>

    <div>
      <div class="gl-title">
        洋流分布及其影响
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        观察暖流、寒流与环流方向，比较气候、渔场、航行和污染扩散
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 不作真实航海依据
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            所在半球
          </span>
        </div>

        <select data-role="hemisphere">
          <option
            value="north"
            ${hemisphere === 'north' ? 'selected' : ''}
          >
            北半球副热带海盆
          </option>

          <option
            value="south"
            ${hemisphere === 'south' ? 'selected' : ''}
          >
            南半球副热带海盆
          </option>
        </select>
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            洋流强度
          </span>

          <span
            class="gl-value"
            data-role="strength-value"
          ></span>
        </div>

        <input
          type="range"
          min="20"
          max="100"
          step="1"
          value="${shortNumber(currentStrength)}"
          data-role="strength"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            寒暖流温差
          </span>

          <span
            class="gl-value"
            data-role="contrast-value"
          ></span>
        </div>

        <input
          type="range"
          min="2"
          max="12"
          step="1"
          value="${shortNumber(temperatureContrast)}"
          data-role="contrast"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            沿岸上升流
          </span>

          <span
            class="gl-value"
            data-role="upwelling-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(upwelling)}"
          data-role="upwelling"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            海水营养盐
          </span>

          <span
            class="gl-value"
            data-role="nutrient-value"
          ></span>
        </div>

        <input
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(nutrientLevel)}"
          data-role="nutrient"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            重点观察
          </span>
        </div>

        <select data-role="observation-mode">
          <option
            value="climate"
            ${observationMode === 'climate' ? 'selected' : ''}
          >
            沿岸气候
          </option>

          <option
            value="fishery"
            ${observationMode === 'fishery' ? 'selected' : ''}
          >
            渔场形成
          </option>

          <option
            value="navigation"
            ${observationMode === 'navigation' ? 'selected' : ''}
          >
            海洋航行
          </option>

          <option
            value="pollution"
            ${observationMode === 'pollution' ? 'selected' : ''}
          >
            污染扩散
          </option>
        </select>
      </div>

      <div class="gl-switch-row">
        <span>显示洋流和影响标注</span>

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
        典型海洋情境
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="warm"
        >
          🔴 暖流沿岸
        </button>

        <button
          type="button"
          data-scenario="cold"
        >
          🔵 寒流沿岸
        </button>

        <button
          type="button"
          data-scenario="fishery"
        >
          🐟 渔场条件
        </button>

        <button
          type="button"
          data-scenario="shipping"
        >
          🚢 航行比较
        </button>

        <button
          type="button"
          data-scenario="pollution"
        >
          🟣 污染扩散
        </button>

        <button
          type="button"
          data-role="reset"
        >
          ↩️ 恢复初始
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      ></div>
    </div>

    <div class="gl-stage">
      <canvas
        class="gl-ocean-canvas"
        width="820"
        height="470"
        data-role="canvas"
        aria-label="洋流分布与影响教学示意图"
      ></canvas>

      <div class="gl-summary">
        <div class="gl-summary-card">
          <strong data-role="rotation-value"></strong>
          <span>海盆环流方向</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="climate-value"></strong>
          <span>沿岸气候效应</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="fishery-value"></strong>
          <span>渔场潜力</span>
        </div>

        <div class="gl-summary-card">
          <strong data-role="navigation-value"></strong>
          <span>顺流航行优势</span>
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

      function drawCurvedCurrent(
        context,
        startX,
        startY,
        controlX,
        controlY,
        endX,
        endY,
        color,
        width,
        phase
      ){
        context.save();
        context.strokeStyle=color;
        context.lineWidth=width;
        context.lineCap='round';
        context.setLineDash([13,8]);
        context.lineDashOffset=
          -phase*42;

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

        drawArrowHead(
          context,
          endX,
          endY,
          angle,
          color,
          13
        );
      }

      function drawFish(
        context,
        x,
        y,
        scale,
        color,
        direction
      ){
        var sign=
          direction==='left'
            ? -1
            : 1;

        context.save();
        context.translate(x,y);
        context.scale(
          sign*scale,
          scale
        );

        context.fillStyle=color;

        context.beginPath();
        context.ellipse(
          0,
          0,
          12,
          6,
          0,
          0,
          Math.PI*2
        );
        context.fill();

        context.beginPath();
        context.moveTo(-10,0);
        context.lineTo(-20,-8);
        context.lineTo(-20,8);
        context.closePath();
        context.fill();

        context.fillStyle='#FFFFFF';
        context.beginPath();
        context.arc(
          5,
          -1,
          1.7,
          0,
          Math.PI*2
        );
        context.fill();

        context.restore();
      }

      function drawShip(
        context,
        x,
        y,
        direction,
        color
      ){
        var sign=
          direction==='left'
            ? -1
            : 1;

        context.save();
        context.translate(x,y);
        context.scale(sign,1);

        context.fillStyle=color;
        context.beginPath();
        context.moveTo(-24,0);
        context.lineTo(25,0);
        context.lineTo(15,13);
        context.lineTo(-16,13);
        context.closePath();
        context.fill();

        context.fillStyle='#F8FAFC';
        context.fillRect(
          -8,
          -16,
          22,
          16
        );

        context.fillStyle='#475569';
        context.fillRect(
          -2,
          -12,
          7,
          6
        );

        context.restore();
      }

      function readState(){
        return {
          hemisphere:
            hemisphereSelect.value,
          strength:Number(
            strengthInput.value
          ),
          contrast:Number(
            contrastInput.value
          ),
          upwelling:Number(
            upwellingInput.value
          ),
          nutrient:Number(
            nutrientInput.value
          ),
          observationMode:
            observationSelect.value
        };
      }

      function calculate(state){
        var strengthRatio=
          state.strength/
          100;

        var contrastRatio=
          state.contrast/
          12;

        var upwellingRatio=
          state.upwelling/
          100;

        var nutrientRatio=
          state.nutrient/
          100;

        var warmClimate=
          clamp(
            28+
            strengthRatio*42+
            contrastRatio*25,
            0,
            100
          );

        var coldClimate=
          clamp(
            24+
            strengthRatio*38+
            contrastRatio*28,
            0,
            100
          );

        var fisheryPotential=
          clamp(
            12+
            upwellingRatio*48+
            nutrientRatio*38+
            contrastRatio*14,
            0,
            100
          );

        var navigationGain=
          clamp(
            8+
            strengthRatio*36,
            5,
            48
          );

        var reversePenalty=
          clamp(
            10+
            strengthRatio*44,
            8,
            60
          );

        var pollutionSpread=
          clamp(
            18+
            strengthRatio*52+
            upwellingRatio*10,
            0,
            100
          );

        var convergence=
          clamp(
            20+
            contrastRatio*34+
            nutrientRatio*26,
            0,
            100
          );

        return {
          rotation:
            state.hemisphere==='north'
              ? '顺时针'
              : '逆时针',
          warmClimate:warmClimate,
          coldClimate:coldClimate,
          fisheryPotential:fisheryPotential,
          navigationGain:navigationGain,
          reversePenalty:reversePenalty,
          pollutionSpread:pollutionSpread,
          convergence:convergence,
          warmLabel:
            warmClimate>=72
              ? '显著增温增湿'
              : warmClimate>=48
                ? '中等增温增湿'
                : '轻微增温增湿',
          coldLabel:
            coldClimate>=72
              ? '显著降温减湿'
              : coldClimate>=48
                ? '中等降温减湿'
                : '轻微降温减湿',
          fisheryLabel:
            fisheryPotential>=75
              ? '较高'
              : fisheryPotential>=48
                ? '中等'
                : '较低'
        };
      }

      function describe(
        state,
        model
      ){
        if(state.observationMode==='fishery'){
          return '当前渔场潜力为'+
            model.fisheryLabel+
            '。沿岸上升流可把深层营养盐带到表层，'+
            '寒暖流交汇也有利于形成水团边界和饵料聚集。'+
            '真实渔场还受水深、海岸、季节和捕捞强度影响。';
        }

        if(state.observationMode==='navigation'){
          return '在本教学模型中，顺流航行速度可提高约'+
            Math.round(
              model.navigationGain
            )+
            '%，逆流航行阻力和能耗相对增加。'+
            '真实船舶航线必须依据专业海图、天气和海况，'+
            '本图不能用于导航。';
        }

        if(state.observationMode==='pollution'){
          return '洋流可使污染物沿环流方向扩散，'+
            '当前扩散强度约为'+
            Math.round(
              model.pollutionSpread
            )+
            '%。污染影响可能跨越行政边界，'+
            '实际处置必须依靠监测、数值模型和专业部门。';
        }

        return '海盆西侧暖流表现为'+
          model.warmLabel+
          '，东侧寒流表现为'+
          model.coldLabel+
          '。暖流通常使沿岸较温暖湿润，'+
          '寒流通常使沿岸较凉、降水条件相对减弱。';
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

        strengthValue.textContent=
          Math.round(
            values.strength
          )+
          '%';

        contrastValue.textContent=
          Math.round(
            values.contrast
          )+
          '℃';

        upwellingValue.textContent=
          Math.round(
            values.upwelling
          )+
          '%';

        nutrientValue.textContent=
          Math.round(
            values.nutrient
          )+
          '%';

        rotationValue.textContent=
          model.rotation;

        climateValue.textContent=
          values.observationMode==='climate'
            ? model.warmLabel
            : model.coldLabel;

        fisheryValue.textContent=
          model.fisheryLabel;

        navigationValue.textContent=
          '+'+
          Math.round(
            model.navigationGain
          )+
          '%';

        result.textContent=describe(
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
          'rgba(255,255,255,0.92)',
          '#BAE6FD'
        );

        drawText(
          context,
          values.hemisphere==='north'
            ? '北半球副热带海盆环流'
            : '南半球副热带海盆环流',
          40,
          43,
          14,
          '#164E63',
          850,
          'left'
        );

        drawText(
          context,
          '暖流以红色表示 · 寒流以蓝色表示 · 位置与尺度均为示意',
          782,
          43,
          9.5,
          '#64748B',
          650,
          'right'
        );

        var basin={
          x:94,
          y:74,
          width:560,
          height:260
        };

        var oceanGradient=
          context.createLinearGradient(
            basin.x,
            basin.y,
            basin.x+
            basin.width,
            basin.y+
            basin.height
          );

        oceanGradient.addColorStop(
          0,
          '#BAE6FD'
        );

        oceanGradient.addColorStop(
          0.5,
          '#38BDF8'
        );

        oceanGradient.addColorStop(
          1,
          '#075985'
        );

        fillRoundedRect(
          context,
          basin.x,
          basin.y,
          basin.width,
          basin.height,
          25,
          oceanGradient,
          '#0369A1'
        );

        context.fillStyle='#D9B77C';

        context.beginPath();
        context.moveTo(70,72);
        context.lineTo(130,72);
        context.bezierCurveTo(
          113,
          118,
          126,
          171,
          110,
          214
        );
        context.bezierCurveTo(
          96,
          257,
          121,
          303,
          89,
          339
        );
        context.lineTo(42,339);
        context.lineTo(42,72);
        context.closePath();
        context.fill();

        context.beginPath();
        context.moveTo(640,72);
        context.lineTo(720,72);
        context.lineTo(720,339);
        context.lineTo(670,339);
        context.bezierCurveTo(
          642,
          304,
          668,
          260,
          650,
          216
        );
        context.bezierCurveTo(
          632,
          170,
          660,
          120,
          640,
          72
        );
        context.closePath();
        context.fill();

        context.strokeStyle='#A16207';
        context.lineWidth=2;
        context.stroke();

        var warmColor='#EF4444';
        var coldColor='#2563EB';
        var horizontalColor='#0F766E';

        var currentWidth=
          3+
          values.strength/
          100*
          5;

        var north=
          values.hemisphere==='north';

        if(north){
          drawCurvedCurrent(
            context,
            145,
            286,
            115,
            180,
            160,
            103,
            warmColor,
            currentWidth,
            state.phase
          );

          drawCurvedCurrent(
            context,
            160,
            103,
            365,
            65,
            590,
            111,
            horizontalColor,
            currentWidth*0.78,
            state.phase
          );

          drawCurvedCurrent(
            context,
            590,
            111,
            665,
            195,
            602,
            285,
            coldColor,
            currentWidth*0.74,
            state.phase
          );

          drawCurvedCurrent(
            context,
            602,
            285,
            355,
            340,
            145,
            286,
            horizontalColor,
            currentWidth*0.68,
            state.phase
          );
        }else{
          drawCurvedCurrent(
            context,
            160,
            103,
            112,
            190,
            145,
            286,
            coldColor,
            currentWidth*0.74,
            state.phase
          );

          drawCurvedCurrent(
            context,
            145,
            286,
            355,
            340,
            602,
            285,
            horizontalColor,
            currentWidth*0.68,
            state.phase
          );

          drawCurvedCurrent(
            context,
            602,
            285,
            665,
            190,
            590,
            111,
            warmColor,
            currentWidth,
            state.phase
          );

          drawCurvedCurrent(
            context,
            590,
            111,
            365,
            65,
            160,
            103,
            horizontalColor,
            currentWidth*0.78,
            state.phase
          );
        }

        context.save();
        context.globalAlpha=
          0.28+
          values.upwelling/
          100*
          0.65;

        for(
          var upIndex=0;
          upIndex<6;
          upIndex+=1
        ){
          var upX=
            north
              ? 606+
                upIndex*6
              : 112+
                upIndex*6;

          var upY=
            275-
            upIndex*22;

          context.strokeStyle='#A7F3D0';
          context.lineWidth=3;

          context.beginPath();
          context.moveTo(
            upX,
            upY+23
          );
          context.lineTo(
            upX,
            upY
          );
          context.stroke();

          drawArrowHead(
            context,
            upX,
            upY,
            -Math.PI/2,
            '#10B981',
            8
          );
        }

        context.restore();

        var labelVisible=
          labelSwitch.checked;

        if(labelVisible){
          fillRoundedRect(
            context,
            north
              ? 116
              : 554,
            143,
            95,
            27,
            13,
            'rgba(254,226,226,0.90)',
            '#FCA5A5'
          );

          drawText(
            context,
            '暖流',
            north
              ? 163
              : 601,
            157,
            11,
            '#B91C1C',
            850,
            'center'
          );

          fillRoundedRect(
            context,
            north
              ? 552
              : 111,
            220,
            95,
            27,
            13,
            'rgba(219,234,254,0.90)',
            '#93C5FD'
          );

          drawText(
            context,
            '寒流',
            north
              ? 599
              : 158,
            234,
            11,
            '#1D4ED8',
            850,
            'center'
          );

          drawText(
            context,
            north
              ? '西岸增温增湿'
              : '东岸增温增湿',
            north
              ? 52
              : 716,
            188,
            9.5,
            '#B91C1C',
            800,
            north
              ? 'left'
              : 'right'
          );

          drawText(
            context,
            north
              ? '东岸降温减湿'
              : '西岸降温减湿',
            north
              ? 716
              : 52,
            188,
            9.5,
            '#1D4ED8',
            800,
            north
              ? 'right'
              : 'left'
          );

          drawText(
            context,
            '上升流',
            north
              ? 660
              : 82,
            280,
            9.5,
            '#047857',
            800,
            north
              ? 'left'
              : 'right'
          );
        }

        if(
          values.observationMode==='climate'
        ){
          context.save();

          var warmSideX=
            north
              ? 72
              : 708;

          var coldSideX=
            north
              ? 708
              : 72;

          context.fillStyle=
            'rgba(239,68,68,0.12)';

          context.fillRect(
            warmSideX-
            33,
            96,
            28,
            205
          );

          context.fillStyle=
            'rgba(37,99,235,0.14)';

          context.fillRect(
            coldSideX+
            (
              north
                ? 5
                : -33
            ),
            96,
            28,
            205
          );

          context.restore();

          drawText(
            context,
            '暖流沿岸：较温暖湿润',
            687,
            363,
            10,
            '#B91C1C',
            800,
            'right'
          );

          drawText(
            context,
            '寒流沿岸：较凉、空气较稳定',
            687,
            381,
            10,
            '#1D4ED8',
            800,
            'right'
          );
        }

        if(
          values.observationMode==='fishery'
        ){
          var fishCount=
            Math.round(
              3+
              model.fisheryPotential/
              10
            );

          for(
            var fishIndex=0;
            fishIndex<fishCount;
            fishIndex+=1
          ){
            var fishX=
              north
                ? 540+
                  (
                    fishIndex%4
                  )*
                  24
                : 150+
                  (
                    fishIndex%4
                  )*
                  24;

            var fishY=
              245+
              Math.floor(
                fishIndex/4
              )*
              19;

            drawFish(
              context,
              fishX,
              fishY,
              0.72,
              fishIndex%2===0
                ? '#F59E0B'
                : '#F8FAFC',
              fishIndex%2===0
                ? 'right'
                : 'left'
            );
          }

          drawText(
            context,
            '营养盐上涌与水团交汇',
            686,
            372,
            10,
            '#047857',
            800,
            'right'
          );
        }

        if(
          values.observationMode==='navigation'
        ){
          var shipDirection=
            north
              ? 'right'
              : 'left';

          var shipX=
            north
              ? 312+
                state.phase*
                170
              : 485-
                state.phase*
                170;

          drawShip(
            context,
            shipX,
            113,
            shipDirection,
            '#334155'
          );

          drawText(
            context,
            '顺流：相对省时省能',
            686,
            363,
            10,
            '#0F766E',
            800,
            'right'
          );

          drawText(
            context,
            '逆流：相对增时增耗',
            686,
            381,
            10,
            '#B45309',
            800,
            'right'
          );
        }

        if(
          values.observationMode==='pollution'
        ){
          var particleCount=
            Math.round(
              6+
              model.pollutionSpread/
              7
            );

          for(
            var particleIndex=0;
            particleIndex<particleCount;
            particleIndex+=1
          ){
            var offset=
              (
                particleIndex/
                particleCount+
                state.phase
              )%
              1;

            var angle=
              offset*
              Math.PI*
              2;

            var centerX=375;
            var centerY=205;
            var radiusX=205;
            var radiusY=92;

            var directionSign=
              north
                ? 1
                : -1;

            var particleAngle=
              angle*
              directionSign;

            var particleX=
              centerX+
              Math.cos(
                particleAngle
              )*
              radiusX;

            var particleY=
              centerY+
              Math.sin(
                particleAngle
              )*
              radiusY;

            context.fillStyle=
              'rgba(147,51,234,'+
              (
                0.35+
                particleIndex/
                particleCount*
                0.45
              )+
              ')';

            context.beginPath();
            context.arc(
              particleX,
              particleY,
              3+
              particleIndex%3,
              0,
              Math.PI*2
            );
            context.fill();
          }

          drawText(
            context,
            '污染物可沿洋流跨区域扩散',
            686,
            372,
            10,
            '#7E22CE',
            800,
            'right'
          );
        }

        fillRoundedRect(
          context,
          678,
          75,
          108,
          240,
          14,
          '#F8FAFC',
          '#CCFBF1'
        );

        drawText(
          context,
          '关系比较',
          696,
          97,
          12,
          '#115E59',
          850,
          'left'
        );

        var relationRows=[
          {
            label:'暖流效应',
            value:model.warmClimate,
            color:'#EF4444'
          },
          {
            label:'寒流效应',
            value:model.coldClimate,
            color:'#2563EB'
          },
          {
            label:'渔场潜力',
            value:model.fisheryPotential,
            color:'#10B981'
          },
          {
            label:'扩散能力',
            value:model.pollutionSpread,
            color:'#9333EA'
          }
        ];

        relationRows.forEach(
          function(item,index){
            var rowY=
              128+
              index*
              48;

            drawText(
              context,
              item.label,
              696,
              rowY,
              9.5,
              '#475569',
              750,
              'left'
            );

            context.fillStyle='#E2E8F0';
            context.fillRect(
              696,
              rowY+12,
              72,
              8
            );

            context.fillStyle=item.color;
            context.fillRect(
              696,
              rowY+12,
              clamp(
                item.value/
                100*
                72,
                3,
                72
              ),
              8
            );

            drawText(
              context,
              Math.round(
                item.value
              )+
              '%',
              774,
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
          '洋流路径、速度及影响范围均为课堂示意，不代表任何真实航线或海区。',
          410,
          414,
          9.5,
          '#64748B',
          650,
          'center'
        );
      }

      function applyScenario(name){
        var scenarios={
          warm:{
            hemisphere:'north',
            strength:82,
            contrast:9,
            upwelling:22,
            nutrient:38,
            observationMode:'climate'
          },
          cold:{
            hemisphere:'north',
            strength:78,
            contrast:10,
            upwelling:72,
            nutrient:68,
            observationMode:'climate'
          },
          fishery:{
            hemisphere:'north',
            strength:68,
            contrast:8,
            upwelling:94,
            nutrient:92,
            observationMode:'fishery'
          },
          shipping:{
            hemisphere:'north',
            strength:92,
            contrast:7,
            upwelling:35,
            nutrient:45,
            observationMode:'navigation'
          },
          pollution:{
            hemisphere:'south',
            strength:88,
            contrast:6,
            upwelling:62,
            nutrient:55,
            observationMode:'pollution'
          }
        };

        var scenario=scenarios[name];

        if(!scenario){
          return;
        }

        hemisphereSelect.value=
          scenario.hemisphere;

        strengthInput.value=
          String(
            scenario.strength
          );

        contrastInput.value=
          String(
            scenario.contrast
          );

        upwellingInput.value=
          String(
            scenario.upwelling
          );

        nutrientInput.value=
          String(
            scenario.nutrient
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
          3000
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
          4200%
          1;

        render();

        state.raf=
          requestAnimationFrame(
            animate
          );
      }

      var hemisphereSelect=query(
        '[data-role="hemisphere"]'
      );

      var strengthInput=query(
        '[data-role="strength"]'
      );

      var contrastInput=query(
        '[data-role="contrast"]'
      );

      var upwellingInput=query(
        '[data-role="upwelling"]'
      );

      var nutrientInput=query(
        '[data-role="nutrient"]'
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

      var resetButton=query(
        '[data-role="reset"]'
      );

      var result=query(
        '[data-role="result"]'
      );

      var canvas=query(
        '[data-role="canvas"]'
      );

      var strengthValue=query(
        '[data-role="strength-value"]'
      );

      var contrastValue=query(
        '[data-role="contrast-value"]'
      );

      var upwellingValue=query(
        '[data-role="upwelling-value"]'
      );

      var nutrientValue=query(
        '[data-role="nutrient-value"]'
      );

      var rotationValue=query(
        '[data-role="rotation-value"]'
      );

      var climateValue=query(
        '[data-role="climate-value"]'
      );

      var fisheryValue=query(
        '[data-role="fishery-value"]'
      );

      var navigationValue=query(
        '[data-role="navigation-value"]'
      );

      if(
        !hemisphereSelect ||
        !strengthInput ||
        !contrastInput ||
        !upwellingInput ||
        !nutrientInput ||
        !observationSelect ||
        !labelSwitch ||
        !autoSwitch ||
        !scenarioButtons.length ||
        !resetButton ||
        !result ||
        !canvas ||
        !strengthValue ||
        !contrastValue ||
        !upwellingValue ||
        !nutrientValue ||
        !rotationValue ||
        !climateValue ||
        !fisheryValue ||
        !navigationValue
      ){
        return;
      }

      var initialState={
        hemisphere:'${hemisphere}',
        strength:${currentStrength},
        contrast:${temperatureContrast},
        upwelling:${upwelling},
        nutrient:${nutrientLevel},
        observationMode:'${observationMode}',
        automatic:${automatic}
      };

      var scenarioOrder=[
        'warm',
        'cold',
        'fishery',
        'shipping',
        'pollution'
      ];

      var state={
        phase:0,
        startedAt:0,
        raf:0,
        timer:null,
        scenarioIndex:-1
      };

      [
        strengthInput,
        contrastInput,
        upwellingInput,
        nutrientInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            clearScenarioSelection
          );
        }
      );

      hemisphereSelect.addEventListener(
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
                'warm';

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

      resetButton.addEventListener(
        'click',
        function(){
          hemisphereSelect.value=
            initialState.hemisphere;

          strengthInput.value=
            String(
              initialState.strength
            );

          contrastInput.value=
            String(
              initialState.contrast
            );

          upwellingInput.value=
            String(
              initialState.upwelling
            );

          nutrientInput.value=
            String(
              initialState.nutrient
            );

          observationSelect.value=
            initialState.observationMode;

          autoSwitch.checked=
            initialState.automatic;

          state.scenarioIndex=-1;

          Array.prototype.forEach.call(
            scenarioButtons,
            function(button){
              button.classList.remove(
                'active'
              );
            }
          );

          schedule();
          render();
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

export const GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_OCEAN_CURRENTS:
GeographyLabTemplate[] = [
  {
    id: 'geography-ocean-current-climate-fishery-navigation',
    group: '🌊 水循环、河流与海洋系统',
    name: '洋流分布及其影响',
    emoji: '🌐',
    desc: '切换南北半球，调节洋流强度、温差、上升流和营养盐，观察气候、渔场、航行与污染扩散。',
    params: [
      {
        key: 'hemisphere',
        label: '初始所在半球',
        type: 'select',
        options: [
          {
            label: '北半球副热带海盆',
            value: 'north',
          },
          {
            label: '南半球副热带海盆',
            value: 'south',
          },
        ],
        defaultValue: 'north',
      },
      {
        key: 'currentStrength',
        label: '洋流强度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 68,
        hint: '只改变教学图中的流速表现和相对影响强度。',
      },
      {
        key: 'temperatureContrast',
        label: '寒暖流温差',
        type: 'number',
        min: 2,
        max: 12,
        step: 1,
        defaultValue: 7,
        hint: '温差越大，沿岸气候影响和水团边界越明显。',
      },
      {
        key: 'upwelling',
        label: '沿岸上升流强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 55,
        hint: '上升流可把较深层海水和营养盐输送到表层。',
      },
      {
        key: 'nutrientLevel',
        label: '海水营养盐水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 60,
        hint: '营养盐较丰富通常有利于浮游生物繁殖和渔场形成。',
      },
      {
        key: 'observationMode',
        label: '初始观察主题',
        type: 'select',
        options: [
          {
            label: '沿岸气候',
            value: 'climate',
          },
          {
            label: '渔场形成',
            value: 'fishery',
          },
          {
            label: '海洋航行',
            value: 'navigation',
          },
          {
            label: '污染扩散',
            value: 'pollution',
          },
        ],
        defaultValue: 'climate',
      },
      {
        key: 'showLabels',
        label: '显示洋流和影响标注',
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
    buildHTML: buildOceanCurrentsHTML,
  },
]
