/**
 * geographyLabTemplatesTopography.ts
 *
 * 地理第33批：
 *   等高线地形图、坡度与地形剖面。
 *
 * 教学边界：
 *   - 地形由数学函数生成，仅用于课堂判读；
 *   - 等高线、高程、相对高度和坡度均为教学示意；
 *   - 不用于真实测绘、工程选址或灾害风险判断。
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

function buildTopographyHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const contourInterval = Math.max(
    20,
    Math.min(
      100,
      numberValue(params, 'contourInterval', 40),
    ),
  )

  const relief = Math.max(
    0.6,
    Math.min(
      1.6,
      numberValue(params, 'relief', 1),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  const requestedMode = stringValue(
    params,
    'profileMode',
    'west-east',
  )

  const profileMode = [
    'west-east',
    'north-south',
    'diagonal',
  ].includes(requestedMode)
    ? requestedMode
    : 'west-east'

  return `
<div id="${rootId}" class="gl-topography-root">
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
      background:linear-gradient(135deg,#CCFBF1,#FEF3C7);
      border-bottom:1px solid #99F6E4;
    }

    #${rootId} .gl-title{
      color:#115E59;
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
      background:linear-gradient(180deg,#F0FDFA,#FFFBEB);
    }

    #${rootId} .gl-stage{
      min-width:0;
      min-height:0;
      padding:8px;
      background:radial-gradient(
        circle at 48% 22%,
        #FFFFFF 0%,
        #F8FAFC 62%,
        #ECFEFF 100%
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
      background:#CFFAFE;
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:17px;
      height:17px;
      appearance:none;
      border-radius:50%;
      background:linear-gradient(135deg,#2DD4BF,#0F766E);
      border:2px solid #FFFFFF;
      box-shadow:0 1px 5px rgba(15,118,110,0.45);
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
      border-color:#0F766E;
      color:#FFFFFF;
      background:linear-gradient(135deg,#2DD4BF,#0F766E);
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

    #${rootId} .gl-map-svg{
      width:100%;
      height:100%;
      display:block;
    }

    #${rootId} .gl-contour{
      fill:none;
      stroke:#92400E;
      stroke-width:1.5;
      opacity:0.82;
    }

    #${rootId} .gl-contour-index{
      stroke:#78350F;
      stroke-width:2.4;
    }

    #${rootId} .gl-contour-label{
      fill:#78350F;
      stroke:#FFFBEB;
      stroke-width:3px;
      paint-order:stroke;
      font-size:10px;
      font-weight:850;
    }

    #${rootId} .gl-profile-grid{
      stroke:#CBD5E1;
      stroke-width:1;
      stroke-dasharray:4 4;
    }

    #${rootId} .gl-small-label{
      fill:#64748B;
      font-size:10.5px;
      font-weight:700;
    }

    #${rootId} .gl-profile-label{
      fill:#334155;
      font-size:11px;
      font-weight:850;
    }

    #${rootId} .gl-boundary{
      fill:#64748B;
      font-size:10px;
    }
  </style>

  <div class="gl-head">
    <div style="font-size:23px;">⛰️</div>

    <div>
      <div class="gl-title">
        等高线地形图、坡度与地形剖面
      </div>

      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        比较等高线疏密，观察剖面线经过的地势变化
      </div>
    </div>

    <div class="gl-note">
      教学简化模型 · 非真实测绘结果
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">等高距</span>
          <span
            class="gl-value"
            data-role="interval-value"
          >
            40米
          </span>
        </div>

        <input
          type="range"
          min="20"
          max="100"
          step="10"
          value="${contourInterval}"
          data-role="interval"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">地形起伏</span>
          <span
            class="gl-value"
            data-role="relief-value"
          >
            1.0倍
          </span>
        </div>

        <input
          type="range"
          min="0.6"
          max="1.6"
          step="0.1"
          value="${relief}"
          data-role="relief"
        />
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-mode="west-east"
        >
          西—东剖面
        </button>

        <button
          type="button"
          data-mode="north-south"
        >
          北—南剖面
        </button>

        <button
          type="button"
          data-mode="diagonal"
        >
          西北—东南
        </button>

        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          高程标注
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
        等高线越密集，单位水平距离内高差越大，坡度通常越陡。
      </div>
    </div>

    <div class="gl-stage">
      <svg
        class="gl-map-svg"
        viewBox="0 0 780 440"
        role="img"
        aria-label="等高线地形与地形剖面教学示意图"
      >
        <defs>
          <linearGradient
            id="${rootId}-terrain"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop
              offset="0%"
              stop-color="#ECFCCB"
            />

            <stop
              offset="52%"
              stop-color="#FEF3C7"
            />

            <stop
              offset="100%"
              stop-color="#FED7AA"
            />
          </linearGradient>

          <linearGradient
            id="${rootId}-profile"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop
              offset="0%"
              stop-color="#F59E0B"
              stop-opacity="0.62"
            />

            <stop
              offset="100%"
              stop-color="#FEF3C7"
              stop-opacity="0.20"
            />
          </linearGradient>
        </defs>

        <rect
          x="26"
          y="28"
          width="470"
          height="255"
          rx="18"
          fill="url(#${rootId}-terrain)"
          stroke="#FCD34D"
          stroke-width="2"
        />

        <g data-role="contour-group">
          <path
            data-contour="true"
            data-level="1"
            data-cx="180"
            data-cy="150"
            data-rx="122"
            data-ry="88"
            data-skew="-8"
            class="gl-contour"
          />

          <path
            data-contour="true"
            data-level="2"
            data-cx="184"
            data-cy="148"
            data-rx="98"
            data-ry="68"
            data-skew="-5"
            class="gl-contour"
          />

          <path
            data-contour="true"
            data-level="3"
            data-cx="190"
            data-cy="146"
            data-rx="73"
            data-ry="49"
            data-skew="-2"
            class="gl-contour gl-contour-index"
          />

          <path
            data-contour="true"
            data-level="4"
            data-cx="197"
            data-cy="143"
            data-rx="49"
            data-ry="31"
            data-skew="1"
            class="gl-contour"
          />

          <path
            data-contour="true"
            data-level="5"
            data-cx="203"
            data-cy="140"
            data-rx="27"
            data-ry="17"
            data-skew="2"
            class="gl-contour"
          />

          <path
            data-contour="true"
            data-level="1"
            data-cx="350"
            data-cy="170"
            data-rx="98"
            data-ry="72"
            data-skew="7"
            class="gl-contour"
          />

          <path
            data-contour="true"
            data-level="2"
            data-cx="346"
            data-cy="166"
            data-rx="75"
            data-ry="52"
            data-skew="5"
            class="gl-contour"
          />

          <path
            data-contour="true"
            data-level="3"
            data-cx="340"
            data-cy="160"
            data-rx="52"
            data-ry="34"
            data-skew="3"
            class="gl-contour gl-contour-index"
          />

          <path
            data-contour="true"
            data-level="4"
            data-cx="335"
            data-cy="154"
            data-rx="29"
            data-ry="18"
            data-skew="1"
            class="gl-contour"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <text
            data-contour-label="true"
            class="gl-contour-label"
          />

          <circle
            data-role="peak"
            cx="203"
            cy="140"
            r="4.5"
            fill="#78350F"
            stroke="#FFFFFF"
            stroke-width="2"
          />
        </g>

        <line
          data-role="transect-line"
          x1="50"
          y1="155"
          x2="472"
          y2="155"
          stroke="#DC2626"
          stroke-width="3"
          stroke-dasharray="9 6"
        />

        <circle
          data-role="transect-start"
          cx="50"
          cy="155"
          r="6"
          fill="#DC2626"
          stroke="#FFFFFF"
          stroke-width="2"
        />

        <circle
          data-role="transect-end"
          cx="472"
          cy="155"
          r="6"
          fill="#DC2626"
          stroke="#FFFFFF"
          stroke-width="2"
        />

        <text
          data-role="start-label"
          x="40"
          y="142"
          fill="#B91C1C"
          font-size="12"
          font-weight="850"
        >
          A
        </text>

        <text
          data-role="end-label"
          x="480"
          y="142"
          fill="#B91C1C"
          font-size="12"
          font-weight="850"
        >
          B
        </text>

        <rect
          x="520"
          y="28"
          width="232"
          height="255"
          rx="18"
          fill="#FFFFFF"
          stroke="#99F6E4"
          stroke-width="2"
        />

        <text
          x="540"
          y="58"
          fill="#115E59"
          font-size="15"
          font-weight="850"
        >
          地形判读
        </text>

        <text
          class="gl-small-label"
          x="540"
          y="90"
        >
          最高点
        </text>

        <text
          data-role="highest-value"
          x="540"
          y="112"
          fill="#0F766E"
          font-size="14"
          font-weight="850"
        >
          —
        </text>

        <text
          class="gl-small-label"
          x="540"
          y="144"
        >
          相对高度
        </text>

        <text
          data-role="relative-value"
          x="540"
          y="166"
          fill="#0F766E"
          font-size="14"
          font-weight="850"
        >
          —
        </text>

        <text
          class="gl-small-label"
          x="540"
          y="198"
        >
          主要地形部位
        </text>

        <text
          data-role="landform-value"
          x="540"
          y="220"
          fill="#0F766E"
          font-size="14"
          font-weight="850"
        >
          —
        </text>

        <text
          class="gl-small-label"
          x="540"
          y="252"
        >
          坡度判断
        </text>

        <text
          data-role="slope-value"
          x="540"
          y="274"
          fill="#0F766E"
          font-size="14"
          font-weight="850"
        >
          —
        </text>

        <rect
          x="26"
          y="304"
          width="726"
          height="110"
          rx="16"
          fill="#FFFFFF"
          stroke="#CBD5E1"
          stroke-width="1.5"
        />

        <line
          class="gl-profile-grid"
          x1="54"
          y1="328"
          x2="728"
          y2="328"
        />

        <line
          class="gl-profile-grid"
          x1="54"
          y1="358"
          x2="728"
          y2="358"
        />

        <line
          class="gl-profile-grid"
          x1="54"
          y1="388"
          x2="728"
          y2="388"
        />

        <line
          x1="54"
          y1="398"
          x2="728"
          y2="398"
          stroke="#64748B"
          stroke-width="1.5"
        />

        <line
          x1="54"
          y1="320"
          x2="54"
          y2="398"
          stroke="#64748B"
          stroke-width="1.5"
        />

        <path
          data-role="profile-fill"
          d=""
          fill="url(#${rootId}-profile)"
        />

        <path
          data-role="profile-line"
          d=""
          fill="none"
          stroke="#B45309"
          stroke-width="3"
          stroke-linejoin="round"
          stroke-linecap="round"
        />

        <text
          class="gl-profile-label"
          x="54"
          y="316"
        >
          地形剖面 A—B
        </text>

        <text
          class="gl-small-label"
          x="36"
          y="332"
        >
          高
        </text>

        <text
          class="gl-small-label"
          x="36"
          y="396"
        >
          低
        </text>

        <text
          class="gl-small-label"
          x="54"
          y="412"
        >
          A
        </text>

        <text
          class="gl-small-label"
          x="718"
          y="412"
        >
          B
        </text>

        <text
          class="gl-boundary"
          x="27"
          y="432"
        >
          等高线、高程和坡度均为教学示意，不用于测绘、工程选址或灾害风险判断。
        </text>
      </svg>
    </div>
  </div>

  <script>
    (function(){
      var root = document.getElementById('${rootId}');
      if(!root)return;

      var intervalInput =
        root.querySelector('[data-role="interval"]');

      var reliefInput =
        root.querySelector('[data-role="relief"]');

      var intervalValue =
        root.querySelector('[data-role="interval-value"]');

      var reliefValue =
        root.querySelector('[data-role="relief-value"]');

      var profileButtons =
        root.querySelectorAll('[data-mode]');

      var labelToggle =
        root.querySelector('[data-role="label-toggle"]');

      var autoToggle =
        root.querySelector('[data-role="auto-toggle"]');

      var resetButton =
        root.querySelector('[data-role="reset"]');

      var contours =
        root.querySelectorAll('[data-contour="true"]');

      var contourLabels =
        root.querySelectorAll('[data-contour-label="true"]');

      var peak =
        root.querySelector('[data-role="peak"]');

      var transectLine =
        root.querySelector('[data-role="transect-line"]');

      var transectStart =
        root.querySelector('[data-role="transect-start"]');

      var transectEnd =
        root.querySelector('[data-role="transect-end"]');

      var startLabel =
        root.querySelector('[data-role="start-label"]');

      var endLabel =
        root.querySelector('[data-role="end-label"]');

      var profileLine =
        root.querySelector('[data-role="profile-line"]');

      var profileFill =
        root.querySelector('[data-role="profile-fill"]');

      var highestValue =
        root.querySelector('[data-role="highest-value"]');

      var relativeValue =
        root.querySelector('[data-role="relative-value"]');

      var landformValue =
        root.querySelector('[data-role="landform-value"]');

      var slopeValue =
        root.querySelector('[data-role="slope-value"]');

      var result =
        root.querySelector('[data-role="result"]');

      if(
        !intervalInput ||
        !reliefInput ||
        !intervalValue ||
        !reliefValue ||
        !profileButtons.length ||
        !labelToggle ||
        !autoToggle ||
        !resetButton ||
        !contours.length ||
        !contourLabels.length ||
        !peak ||
        !transectLine ||
        !transectStart ||
        !transectEnd ||
        !startLabel ||
        !endLabel ||
        !profileLine ||
        !profileFill ||
        !highestValue ||
        !relativeValue ||
        !landformValue ||
        !slopeValue ||
        !result
      ){
        return;
      }

      var initialInterval = ${contourInterval};
      var initialRelief = ${relief};

      var state = {
        profileMode: '${profileMode}',
        showLabels: ${showLabels},
        auto: false,
        startedAt: 0,
        raf: 0
      };

      function clamp(value,min,max){
        return Math.max(
          min,
          Math.min(max,value)
        );
      }

      function numericAttribute(
        element,
        name,
        fallback
      ){
        var value = parseFloat(
          element.getAttribute(name) || ''
        );

        return Number.isFinite(value)
          ? value
          : fallback;
      }

      function terrainHeight(
        x,
        y,
        reliefFactor
      ){
        var hill1 =
          520 *
          reliefFactor *
          Math.exp(
            -(
              Math.pow((x-0.34)/0.22,2) +
              Math.pow((y-0.44)/0.25,2)
            )
          );

        var hill2 =
          390 *
          reliefFactor *
          Math.exp(
            -(
              Math.pow((x-0.70)/0.19,2) +
              Math.pow((y-0.53)/0.23,2)
            )
          );

        var saddle =
          120 *
          reliefFactor *
          Math.exp(
            -(
              Math.pow((x-0.52)/0.18,2) +
              Math.pow((y-0.48)/0.16,2)
            )
          );

        var valley =
          90 *
          Math.exp(
            -Math.pow(
              (
                y -
                (0.78-x*0.34)
              ) / 0.09,
              2
            )
          );

        return Math.max(
          0,
          80+hill1+hill2+saddle-valley
        );
      }

      function contourPath(
        cx,
        cy,
        rx,
        ry,
        skew,
        reliefFactor
      ){
        var adjustedRx =
          rx *
          (0.88+reliefFactor*0.12);

        var adjustedRy =
          ry *
          (0.88+reliefFactor*0.12);

        var adjustedSkew =
          skew *
          reliefFactor;

        return [
          'M',
          cx-adjustedRx,
          cy,
          'C',
          cx-adjustedRx*0.62,
          cy-adjustedRy-adjustedSkew,
          cx+adjustedRx*0.55,
          cy-adjustedRy+adjustedSkew,
          cx+adjustedRx,
          cy,
          'C',
          cx+adjustedRx*0.70,
          cy+adjustedRy-adjustedSkew,
          cx-adjustedRx*0.55,
          cy+adjustedRy+adjustedSkew,
          cx-adjustedRx,
          cy,
          'Z'
        ].join(' ');
      }

      function renderContours(
        interval,
        reliefFactor
      ){
        Array.prototype.forEach.call(
          contours,
          function(contour,index){
            var level =
              numericAttribute(
                contour,
                'data-level',
                1
              );

            var cx =
              numericAttribute(
                contour,
                'data-cx',
                0
              );

            var cy =
              numericAttribute(
                contour,
                'data-cy',
                0
              );

            var rx =
              numericAttribute(
                contour,
                'data-rx',
                20
              );

            var ry =
              numericAttribute(
                contour,
                'data-ry',
                20
              );

            var skew =
              numericAttribute(
                contour,
                'data-skew',
                0
              );

            contour.setAttribute(
              'd',
              contourPath(
                cx,
                cy,
                rx,
                ry,
                skew,
                reliefFactor
              )
            );

            var label =
              contourLabels[index];

            if(label){
              label.setAttribute(
                'x',
                String(cx+rx*0.56)
              );

              label.setAttribute(
                'y',
                String(cy-ry*0.45)
              );

              label.textContent =
                String(
                  100+level*interval
                );

              label.style.display =
                state.showLabels &&
                (
                  level%2===1 ||
                  level===4
                )
                  ? ''
                  : 'none';
            }
          }
        );

        var firstPeak =
          contours[4];

        if(firstPeak){
          peak.setAttribute(
            'cx',
            firstPeak.getAttribute('data-cx') || '203'
          );

          peak.setAttribute(
            'cy',
            firstPeak.getAttribute('data-cy') || '140'
          );
        }
      }

      function profileEndpoints(mode){
        if(mode==='north-south'){
          return {
            x1:255,
            y1:48,
            x2:255,
            y2:264
          };
        }

        if(mode==='diagonal'){
          return {
            x1:58,
            y1:58,
            x2:464,
            y2:258
          };
        }

        return {
          x1:50,
          y1:155,
          x2:472,
          y2:155
        };
      }

      function mapToTerrainPoint(x,y){
        return {
          x:clamp((x-26)/470,0,1),
          y:clamp((y-28)/255,0,1)
        };
      }

      function buildProfile(
        mode,
        reliefFactor
      ){
        var endpoints =
          profileEndpoints(mode);

        var samples = [];
        var maximum = 0;
        var minimum =
          Number.POSITIVE_INFINITY;
        var steepest = 0;

        for(
          var index=0;
          index<=72;
          index+=1
        ){
          var ratio =
            index/72;

          var mapX =
            endpoints.x1+
            (
              endpoints.x2-
              endpoints.x1
            )*ratio;

          var mapY =
            endpoints.y1+
            (
              endpoints.y2-
              endpoints.y1
            )*ratio;

          var terrainPoint =
            mapToTerrainPoint(
              mapX,
              mapY
            );

          var elevation =
            terrainHeight(
              terrainPoint.x,
              terrainPoint.y,
              reliefFactor
            );

          maximum =
            Math.max(
              maximum,
              elevation
            );

          minimum =
            Math.min(
              minimum,
              elevation
            );

          if(samples.length){
            steepest =
              Math.max(
                steepest,
                Math.abs(
                  elevation-
                  samples[
                    samples.length-1
                  ].elevation
                )
              );
          }

          samples.push({
            ratio:ratio,
            elevation:elevation
          });
        }

        var elevationRange =
          Math.max(
            1,
            maximum-minimum
          );

        var points =
          samples.map(function(sample){
            var x =
              54+sample.ratio*674;

            var normalized =
              (
                sample.elevation-
                minimum
              ) / elevationRange;

            var y =
              394-normalized*67;

            return {
              x:x,
              y:y
            };
          });

        var linePath =
          points.map(function(point,index){
            return (
              index===0
                ? 'M'
                : 'L'
            )+
            point.x.toFixed(2)+
            ' '+
            point.y.toFixed(2);
          }).join(' ');

        return {
          endpoints:endpoints,
          linePath:linePath,
          fillPath:
            linePath+
            ' L 728 398 L 54 398 Z',
          highest:maximum,
          lowest:minimum,
          steepest:steepest
        };
      }

      function updateTransect(endpoints){
        transectLine.setAttribute(
          'x1',
          String(endpoints.x1)
        );

        transectLine.setAttribute(
          'y1',
          String(endpoints.y1)
        );

        transectLine.setAttribute(
          'x2',
          String(endpoints.x2)
        );

        transectLine.setAttribute(
          'y2',
          String(endpoints.y2)
        );

        transectStart.setAttribute(
          'cx',
          String(endpoints.x1)
        );

        transectStart.setAttribute(
          'cy',
          String(endpoints.y1)
        );

        transectEnd.setAttribute(
          'cx',
          String(endpoints.x2)
        );

        transectEnd.setAttribute(
          'cy',
          String(endpoints.y2)
        );

        startLabel.setAttribute(
          'x',
          String(endpoints.x1-10)
        );

        startLabel.setAttribute(
          'y',
          String(endpoints.y1-11)
        );

        endLabel.setAttribute(
          'x',
          String(endpoints.x2+8)
        );

        endLabel.setAttribute(
          'y',
          String(endpoints.y2-11)
        );
      }

      function modeName(mode){
        if(mode==='north-south'){
          return '北—南剖面';
        }

        if(mode==='diagonal'){
          return '西北—东南剖面';
        }

        return '西—东剖面';
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var interval =
          clamp(
            parseFloat(
              intervalInput.value
            ) || 40,
            20,
            100
          );

        var reliefFactor =
          clamp(
            parseFloat(
              reliefInput.value
            ) || 1,
            0.6,
            1.6
          );

        intervalValue.textContent =
          Math.round(interval)+'米';

        reliefValue.textContent =
          reliefFactor.toFixed(1)+'倍';

        renderContours(
          interval,
          reliefFactor
        );

        var profile =
          buildProfile(
            state.profileMode,
            reliefFactor
          );

        updateTransect(
          profile.endpoints
        );

        profileLine.setAttribute(
          'd',
          profile.linePath
        );

        profileFill.setAttribute(
          'd',
          profile.fillPath
        );

        var relativeHeight =
          profile.highest-
          profile.lowest;

        var highestContour =
          Math.floor(
            profile.highest/interval
          )*interval;

        highestValue.textContent =
          Math.round(
            highestContour
          )+'米附近';

        relativeValue.textContent =
          Math.round(
            relativeHeight
          )+'米';

        landformValue.textContent =
          state.profileMode==='diagonal'
            ? '山峰—鞍部—次峰'
            : state.profileMode==='north-south'
              ? '山脊与谷地'
              : '双峰与鞍部';

        slopeValue.textContent =
          profile.steepest>22
            ? '局部坡陡'
            : profile.steepest>12
              ? '坡度中等'
              : '坡度较缓';

        Array.prototype.forEach.call(
          profileButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-mode'
              )===state.profileMode
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

        result.textContent =
          modeName(state.profileMode)+
          '经过的最高、最低海拔差约为'+
          Math.round(relativeHeight)+
          '米。等高线密集处坡度较陡，稀疏处坡度较缓；闭合等高线中心高通常表示山峰。';
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

        var phase =
          Math.floor(
            elapsed/2200
          )%3;

        state.profileMode = [
          'west-east',
          'north-south',
          'diagonal'
        ][phase];

        reliefInput.value =
          (
            1.05+
            Math.sin(
              elapsed/1150
            )*0.35
          ).toFixed(1);

        render();

        state.raf =
          requestAnimationFrame(
            animate
          );
      }

      intervalInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      reliefInput.addEventListener(
        'input',
        function(){
          if(state.auto)stopAuto();
          render();
        }
      );

      Array.prototype.forEach.call(
        profileButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto)stopAuto();

              state.profileMode =
                button.getAttribute(
                  'data-mode'
                ) ||
                'west-east';

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

      resetButton.addEventListener(
        'click',
        function(){
          stopAuto();

          intervalInput.value =
            String(initialInterval);

          reliefInput.value =
            String(initialRelief);

          state.profileMode =
            '${profileMode}';

          state.showLabels =
            ${showLabels};

          render();
        }
      );

      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_TOPOGRAPHY:
GeographyLabTemplate[] = [
  {
    id: 'geography-contour-slope-profile',
    group: '🧭 基础定位与地球运动',
    name: '等高线地形图、坡度与地形剖面',
    emoji: '⛰️',
    desc: '调节等高距和地形起伏，切换剖面线，比较坡度、相对高度与地形部位。',
    params: [
      {
        key: 'contourInterval',
        label: '初始等高距',
        type: 'number',
        min: 20,
        max: 100,
        step: 10,
        defaultValue: 40,
        hint: '等高距越大，相邻等高线代表的高差越大。',
      },
      {
        key: 'relief',
        label: '初始地形起伏',
        type: 'number',
        min: 0.6,
        max: 1.6,
        step: 0.1,
        defaultValue: 1,
        hint: '只改变教学模型中的起伏强度。',
      },
      {
        key: 'profileMode',
        label: '初始剖面方向',
        type: 'select',
        options: [
          {
            label: '西—东剖面',
            value: 'west-east',
          },
          {
            label: '北—南剖面',
            value: 'north-south',
          },
          {
            label: '西北—东南剖面',
            value: 'diagonal',
          },
        ],
        defaultValue: 'west-east',
      },
      {
        key: 'showLabels',
        label: '显示高程标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildTopographyHTML,
  },
]
