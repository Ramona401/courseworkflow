/**
 * geographyLabTemplatesHydrologyWaterCycle.ts
 *
 * 第35批B1：水循环过程与人类活动影响。
 *
 * 教学目标：
 * 1. 展示蒸发、蒸腾、水汽输送、降水、下渗、
 *    地表径流和地下径流等主要水循环环节；
 * 2. 比较城市硬化、植被恢复和水库调蓄对流域水循环的影响；
 * 3. 通过水量分配、过程强度和自然流域对照理解各环节联系。
 *
 * 教学边界：
 * - 所有数值均为课堂关系比较使用的相对教学单位；
 * - 不考虑真实流域面积、土壤类型、坡度、降雨历时和蒸散模型；
 * - 不用于洪水预测、防灾决策、水库调度或工程选址；
 * - 不用于真实水文、水资源或环境影响评价。
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

function shortNumber(value: number): string {
  return Number(value.toFixed(3)).toString()
}

function buildWaterCycleHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const precipitation = Math.max(
    20,
    Math.min(
      100,
      numberValue(params, 'precipitation', 65),
    ),
  )

  const temperature = Math.max(
    0,
    Math.min(
      35,
      numberValue(params, 'temperature', 18),
    ),
  )

  const vegetation = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'vegetation', 55),
    ),
  )

  const urbanization = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'urbanization', 25),
    ),
  )

  const reservoir = Math.max(
    0,
    Math.min(
      100,
      numberValue(params, 'reservoir', 20),
    ),
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
<div id="${rootId}" class="gl-water-cycle-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      box-sizing:border-box;
      overflow:hidden;
      border:1px solid #A7D8D2;
      border-radius:18px;
      background:#F8FFFE;
      color:#123B3A;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      display:flex;
      flex-direction:column;
      --gl-water:#0EA5E9;
      --gl-deep:#0369A1;
      --gl-green:#16A34A;
      --gl-soil:#A16207;
      --gl-orange:#F97316;
      --gl-muted:#64748B;
    }

    #${rootId} *{
      box-sizing:border-box;
    }

    #${rootId} .gl-head{
      height:50px;
      display:flex;
      align-items:center;
      gap:12px;
      padding:0 18px;
      background:linear-gradient(
        135deg,
        #CCFBF1,
        #E0F2FE 58%,
        #F0FDFA
      );
      border-bottom:1px solid #A7D8D2;
      flex-shrink:0;
    }

    #${rootId} .gl-title{
      font-size:16px;
      font-weight:900;
      color:#0F5A57;
    }

    #${rootId} .gl-note{
      margin-left:auto;
      font-size:11px;
      color:#477776;
      font-weight:650;
    }

    #${rootId} .gl-body{
      flex:1;
      min-height:0;
      display:grid;
      grid-template-columns:238px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      min-width:0;
      overflow:auto;
      padding:12px 12px 14px;
      background:linear-gradient(
        180deg,
        #F0FDFA,
        #ECFEFF
      );
      border-right:1px solid #BFE8E3;
    }

    #${rootId} .gl-row{
      margin-bottom:10px;
    }

    #${rootId} .gl-label{
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:8px;
      margin-bottom:5px;
      font-size:11px;
      font-weight:750;
      color:#315E5C;
    }

    #${rootId} .gl-value{
      color:#0F766E;
      font-weight:900;
      white-space:nowrap;
    }

    #${rootId} input[type=range]{
      width:100%;
      height:5px;
      border-radius:999px;
      appearance:none;
      outline:none;
      background:linear-gradient(
        90deg,
        #BAE6FD,
        #5EEAD4
      );
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      appearance:none;
      width:16px;
      height:16px;
      border-radius:50%;
      background:#0F766E;
      border:2px solid #FFFFFF;
      box-shadow:0 1px 5px rgba(15,118,110,.38);
    }

    #${rootId} .gl-switch-row{
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:10px;
      padding:7px 8px;
      margin-bottom:7px;
      border-radius:10px;
      background:rgba(255,255,255,.76);
      border:1px solid #CFF1EC;
      font-size:11px;
      font-weight:750;
      color:#315E5C;
    }

    #${rootId} .gl-switch-row input{
      accent-color:#0F766E;
    }

    #${rootId} .gl-divider{
      height:1px;
      background:#BFE8E3;
      margin:10px 0;
    }

    #${rootId} .gl-subtitle{
      font-size:11px;
      font-weight:900;
      color:#0F766E;
      margin:4px 0 7px;
    }

    #${rootId} .gl-button-grid{
      display:grid;
      grid-template-columns:1fr 1fr;
      gap:6px;
    }

    #${rootId} button{
      border:1px solid #A7D8D2;
      border-radius:9px;
      padding:6px 7px;
      background:#FFFFFF;
      color:#315E5C;
      font-size:10.5px;
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
      background:linear-gradient(
        135deg,
        #14B8A6,
        #0F766E
      );
      color:#FFFFFF;
      border-color:#0F766E;
      box-shadow:0 4px 10px rgba(15,118,110,.20);
    }

    #${rootId} .gl-mode-grid{
      display:grid;
      grid-template-columns:repeat(3,1fr);
      gap:5px;
    }

    #${rootId} .gl-mode-grid button{
      padding:7px 3px;
      font-size:10px;
    }

    #${rootId} .gl-result{
      margin-top:10px;
      padding:9px 10px;
      border-radius:11px;
      background:#DFF7F3;
      border:1px solid #A7D8D2;
      color:#155E59;
      font-size:10.5px;
      line-height:1.5;
      font-weight:650;
      max-height:82px;
      overflow:auto;
    }

    #${rootId} .gl-stage{
      position:relative;
      min-width:0;
      min-height:0;
      overflow:hidden;
      background:linear-gradient(
        180deg,
        #E8F7FF 0%,
        #F8FFFF 56%,
        #F5F0E8 56%,
        #EAD9BD 100%
      );
    }

    #${rootId} .gl-stage svg{
      width:100%;
      height:100%;
      display:block;
    }

    #${rootId} .gl-panel{
      position:absolute;
      left:12px;
      right:12px;
      bottom:10px;
      min-height:74px;
      border-radius:13px;
      background:rgba(255,255,255,.93);
      border:1px solid #BFE8E3;
      box-shadow:0 8px 24px rgba(15,73,71,.10);
      padding:9px 11px;
      pointer-events:none;
    }

    #${rootId} .gl-panel-title{
      font-size:11px;
      font-weight:900;
      color:#0F766E;
      margin-bottom:6px;
    }

    #${rootId} .gl-metric-grid{
      display:grid;
      grid-template-columns:repeat(5,1fr);
      gap:7px;
    }

    #${rootId} .gl-metric{
      padding:5px 6px;
      border-radius:9px;
      background:#F0FDFA;
      border:1px solid #D1FAE5;
      text-align:center;
      min-width:0;
    }

    #${rootId} .gl-metric strong{
      display:block;
      font-size:13px;
      color:#0369A1;
    }

    #${rootId} .gl-metric span{
      display:block;
      font-size:9px;
      color:#64748B;
      margin-top:2px;
    }

    #${rootId} .gl-impact-grid{
      display:grid;
      grid-template-columns:repeat(4,1fr);
      gap:7px;
    }

    #${rootId} .gl-impact-item{
      font-size:9px;
      color:#64748B;
    }

    #${rootId} .gl-impact-item b{
      display:block;
      font-size:11px;
      color:#0F766E;
      margin-bottom:3px;
    }

    #${rootId} .gl-track{
      height:6px;
      border-radius:999px;
      background:#E2E8F0;
      overflow:hidden;
    }

    #${rootId} .gl-track i{
      display:block;
      height:100%;
      border-radius:999px;
      background:linear-gradient(
        90deg,
        #38BDF8,
        #14B8A6
      );
    }

    #${rootId} .gl-flow{
      fill:none;
      stroke-linecap:round;
      stroke-linejoin:round;
      stroke-dasharray:10 8;
      animation:${rootId}-flow 1.5s linear infinite;
    }

    #${rootId} .gl-pulse{
      animation:${rootId}-pulse 1.8s ease-in-out infinite;
      transform-box:fill-box;
      transform-origin:center;
    }

    #${rootId} .gl-label-group text{
      paint-order:stroke;
      stroke:#FFFFFF;
      stroke-width:3px;
      stroke-linejoin:round;
    }

    @keyframes ${rootId}-flow{
      to{
        stroke-dashoffset:-36;
      }
    }

    @keyframes ${rootId}-pulse{
      0%,100%{
        opacity:.46;
      }

      50%{
        opacity:1;
      }
    }
  </style>

  <div class="gl-head">
    <div class="gl-title">
      💧 水循环过程与人类活动影响
    </div>

    <div class="gl-note">
      课堂简化模型 · 比较过程关系，不作真实水文预测
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label">
          <span>降水强度</span>

          <span
            class="gl-value"
            data-p-value
          ></span>
        </div>

        <input
          data-p
          type="range"
          min="20"
          max="100"
          step="1"
          value="${shortNumber(precipitation)}"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label">
          <span>近地面气温</span>

          <span
            class="gl-value"
            data-t-value
          ></span>
        </div>

        <input
          data-t
          type="range"
          min="0"
          max="35"
          step="1"
          value="${shortNumber(temperature)}"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label">
          <span>植被覆盖率</span>

          <span
            class="gl-value"
            data-v-value
          ></span>
        </div>

        <input
          data-v
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(vegetation)}"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label">
          <span>地表硬化率</span>

          <span
            class="gl-value"
            data-u-value
          ></span>
        </div>

        <input
          data-u
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(urbanization)}"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label">
          <span>水库调蓄强度</span>

          <span
            class="gl-value"
            data-r-value
          ></span>
        </div>

        <input
          data-r
          type="range"
          min="0"
          max="100"
          step="1"
          value="${shortNumber(reservoir)}"
        />
      </div>

      <div class="gl-switch-row">
        <span>显示过程标注</span>

        <input
          data-label-switch
          type="checkbox"
          ${showLabels ? 'checked' : ''}
        />
      </div>

      <div class="gl-switch-row">
        <span>自动演示典型情境</span>

        <input
          data-auto-switch
          type="checkbox"
          ${automatic ? 'checked' : ''}
        />
      </div>

      <div class="gl-divider"></div>

      <div class="gl-subtitle">
        观察模式
      </div>

      <div class="gl-mode-grid">
        <button
          type="button"
          data-mode="process"
          class="active"
        >
          过程
        </button>

        <button
          type="button"
          data-mode="budget"
        >
          分配
        </button>

        <button
          type="button"
          data-mode="impact"
        >
          影响
        </button>
      </div>

      <div
        class="gl-subtitle"
        style="margin-top:10px;"
      >
        典型情境
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-scenario="natural"
        >
          🌲 自然流域
        </button>

        <button
          type="button"
          data-scenario="urban"
        >
          🏙️ 城市硬化
        </button>

        <button
          type="button"
          data-scenario="restoration"
        >
          🌿 植被恢复
        </button>

        <button
          type="button"
          data-scenario="reservoir"
        >
          🏞️ 水库调蓄
        </button>
      </div>

      <div
        class="gl-result"
        data-result
      ></div>
    </div>

    <div class="gl-stage">
      <svg
        viewBox="0 0 760 460"
        role="img"
        aria-label="流域水循环与人类活动影响示意图"
      >
        <defs>
          <linearGradient
            id="${rootId}-sky"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop
              offset="0"
              stop-color="#DFF4FF"
            />

            <stop
              offset="1"
              stop-color="#F8FFFF"
            />
          </linearGradient>

          <linearGradient
            id="${rootId}-soil"
            x1="0"
            y1="0"
            x2="0"
            y2="1"
          >
            <stop
              offset="0"
              stop-color="#CBA66B"
            />

            <stop
              offset="1"
              stop-color="#7C4A16"
            />
          </linearGradient>

          <linearGradient
            id="${rootId}-water"
            x1="0"
            y1="0"
            x2="1"
            y2="1"
          >
            <stop
              offset="0"
              stop-color="#7DD3FC"
            />

            <stop
              offset="1"
              stop-color="#0284C7"
            />
          </linearGradient>

          <marker
            id="${rootId}-arrow-blue"
            viewBox="0 0 10 10"
            refX="8.5"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path
              d="M0 0 L10 5 L0 10 Z"
              fill="#0284C7"
            />
          </marker>

          <marker
            id="${rootId}-arrow-green"
            viewBox="0 0 10 10"
            refX="8.5"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path
              d="M0 0 L10 5 L0 10 Z"
              fill="#16A34A"
            />
          </marker>

          <marker
            id="${rootId}-arrow-brown"
            viewBox="0 0 10 10"
            refX="8.5"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path
              d="M0 0 L10 5 L0 10 Z"
              fill="#A16207"
            />
          </marker>
        </defs>

        <rect
          width="760"
          height="460"
          fill="url(#${rootId}-sky)"
        />

        <circle
          cx="78"
          cy="66"
          r="28"
          fill="#FACC15"
          opacity=".88"
        />

        <g
          data-cloud-group
          class="gl-pulse"
        >
          <ellipse
            cx="362"
            cy="77"
            rx="72"
            ry="26"
            fill="#FFFFFF"
            stroke="#BAE6FD"
            stroke-width="2"
          />

          <circle
            cx="322"
            cy="69"
            r="27"
            fill="#FFFFFF"
          />

          <circle
            cx="368"
            cy="55"
            r="34"
            fill="#FFFFFF"
          />

          <circle
            cx="414"
            cy="72"
            r="25"
            fill="#FFFFFF"
          />
        </g>

        <path
          d="M0 268 C80 236 126 205 184 184 C233 166 272 180 309 205 C348 232 392 239 437 218 C491 193 520 165 567 153 C622 139 682 167 760 202 L760 460 L0 460 Z"
          fill="url(#${rootId}-soil)"
        />

        <path
          d="M0 267 C86 232 130 200 187 181 C238 164 272 179 312 208 C356 240 397 242 444 216 C495 188 530 160 574 150 C630 137 689 168 760 198"
          fill="none"
          stroke="#65A30D"
          stroke-width="14"
          stroke-linecap="round"
          opacity=".82"
        />

        <g data-vegetation-group>
          <g transform="translate(120 181)">
            <rect
              x="-4"
              y="0"
              width="8"
              height="35"
              fill="#854D0E"
            />

            <circle
              cx="0"
              cy="-8"
              r="24"
              fill="#22C55E"
            />

            <circle
              cx="-15"
              cy="0"
              r="15"
              fill="#16A34A"
            />

            <circle
              cx="16"
              cy="2"
              r="16"
              fill="#15803D"
            />
          </g>

          <g transform="translate(198 169)">
            <rect
              x="-4"
              y="0"
              width="8"
              height="37"
              fill="#854D0E"
            />

            <circle
              cx="0"
              cy="-10"
              r="25"
              fill="#4ADE80"
            />

            <circle
              cx="-17"
              cy="1"
              r="15"
              fill="#16A34A"
            />

            <circle
              cx="17"
              cy="2"
              r="16"
              fill="#15803D"
            />
          </g>

          <g transform="translate(282 201)">
            <rect
              x="-4"
              y="0"
              width="8"
              height="32"
              fill="#854D0E"
            />

            <circle
              cx="0"
              cy="-8"
              r="22"
              fill="#22C55E"
            />

            <circle
              cx="-14"
              cy="1"
              r="14"
              fill="#16A34A"
            />

            <circle
              cx="15"
              cy="2"
              r="14"
              fill="#15803D"
            />
          </g>
        </g>

        <g data-city-group>
          <rect
            x="493"
            y="167"
            width="47"
            height="78"
            rx="3"
            fill="#94A3B8"
            stroke="#475569"
            stroke-width="2"
          />

          <rect
            x="545"
            y="145"
            width="58"
            height="101"
            rx="3"
            fill="#64748B"
            stroke="#334155"
            stroke-width="2"
          />

          <rect
            x="609"
            y="180"
            width="43"
            height="67"
            rx="3"
            fill="#CBD5E1"
            stroke="#64748B"
            stroke-width="2"
          />

          <path
            d="M472 247 H681"
            stroke="#475569"
            stroke-width="13"
          />

          <path
            d="M472 247 H681"
            stroke="#F8FAFC"
            stroke-width="2"
            stroke-dasharray="16 12"
          />

          <g fill="#E0F2FE">
            <rect
              x="502"
              y="178"
              width="9"
              height="10"
            />

            <rect
              x="519"
              y="178"
              width="9"
              height="10"
            />

            <rect
              x="502"
              y="197"
              width="9"
              height="10"
            />

            <rect
              x="519"
              y="197"
              width="9"
              height="10"
            />

            <rect
              x="556"
              y="158"
              width="10"
              height="11"
            />

            <rect
              x="575"
              y="158"
              width="10"
              height="11"
            />

            <rect
              x="556"
              y="180"
              width="10"
              height="11"
            />

            <rect
              x="575"
              y="180"
              width="10"
              height="11"
            />
          </g>
        </g>

        <path
          d="M304 211 C358 246 390 265 438 274 C500 286 574 277 760 306 L760 337 C590 312 508 310 432 304 C374 299 333 277 286 244 Z"
          fill="url(#${rootId}-water)"
          opacity=".92"
        />

        <path
          d="M0 318 C89 307 158 322 223 343 C282 362 350 359 412 343 C486 325 575 335 760 365 L760 460 L0 460 Z"
          fill="#38BDF8"
          opacity=".18"
        />

        <path
          d="M0 357 C125 342 218 372 327 372 C438 372 554 343 760 390"
          fill="none"
          stroke="#0369A1"
          stroke-width="3"
          stroke-dasharray="10 8"
          opacity=".62"
        />

        <g data-reservoir-group>
          <path
            d="M414 249 L432 326"
            stroke="#334155"
            stroke-width="10"
            stroke-linecap="round"
          />

          <path
            data-reservoir-water
            d="M358 260 C382 258 400 259 418 264 L429 312 C395 307 371 304 337 303 Z"
            fill="#38BDF8"
            opacity=".68"
          />

          <text
            x="385"
            y="293"
            text-anchor="middle"
            font-size="10"
            font-weight="900"
            fill="#075985"
          >
            调蓄
          </text>
        </g>

        <g
          data-rain-group
          stroke="#0EA5E9"
          stroke-width="3"
          stroke-linecap="round"
        >
          <line data-rain-line x1="319" y1="101" x2="300" y2="146" />
          <line data-rain-line x1="342" y1="105" x2="326" y2="155" />
          <line data-rain-line x1="367" y1="105" x2="352" y2="153" />
          <line data-rain-line x1="393" y1="104" x2="378" y2="151" />
          <line data-rain-line x1="416" y1="101" x2="402" y2="143" />
          <line data-rain-line x1="441" y1="102" x2="428" y2="137" />
        </g>

        <path
          data-flow="evap"
          class="gl-flow"
          d="M694 300 C714 252 700 206 665 168"
          stroke="#0284C7"
          stroke-width="4"
          marker-end="url(#${rootId}-arrow-blue)"
        />

        <path
          data-flow="transp"
          class="gl-flow"
          d="M201 166 C212 130 235 113 264 99"
          stroke="#16A34A"
          stroke-width="4"
          marker-end="url(#${rootId}-arrow-green)"
        />

        <path
          data-flow="transport"
          class="gl-flow"
          d="M650 136 C572 98 503 79 447 76"
          stroke="#0284C7"
          stroke-width="4"
          marker-end="url(#${rootId}-arrow-blue)"
        />

        <path
          data-flow="surface"
          class="gl-flow"
          d="M181 217 C252 225 307 245 353 272"
          stroke="#F97316"
          stroke-width="5"
          marker-end="url(#${rootId}-arrow-brown)"
        />

        <path
          data-flow="infiltration"
          class="gl-flow"
          d="M251 244 C246 286 255 315 278 341"
          stroke="#A16207"
          stroke-width="4"
          marker-end="url(#${rootId}-arrow-brown)"
        />

        <path
          data-flow="groundwater"
          class="gl-flow"
          d="M282 357 C378 370 487 354 604 365"
          stroke="#0369A1"
          stroke-width="4"
          marker-end="url(#${rootId}-arrow-blue)"
        />

        <path
          data-flow="release"
          class="gl-flow"
          d="M434 309 C469 312 500 315 532 318"
          stroke="#0EA5E9"
          stroke-width="4"
          marker-end="url(#${rootId}-arrow-blue)"
        />

        <g
          data-label-group
          class="gl-label-group"
          font-size="11"
          font-weight="900"
        >
          <text x="686" y="177" fill="#0369A1">
            蒸发
          </text>

          <text x="210" y="113" fill="#15803D">
            蒸腾
          </text>

          <text x="535" y="85" fill="#0369A1">
            水汽输送
          </text>

          <text x="368" y="126" fill="#0369A1">
            降水
          </text>

          <text x="215" y="274" fill="#92400E">
            下渗
          </text>

          <text x="234" y="223" fill="#C2410C">
            地表径流
          </text>

          <text x="430" y="354" fill="#075985">
            地下径流
          </text>
        </g>

        <g
          data-budget-overlay
          style="display:none"
        >
          <rect
            x="55"
            y="73"
            width="245"
            height="150"
            rx="14"
            fill="rgba(255,255,255,.92)"
            stroke="#A7D8D2"
          />

          <text
            x="72"
            y="95"
            font-size="11"
            font-weight="900"
            fill="#0F766E"
          >
            本时段降水水量分配
          </text>

          <g
            font-size="9"
            font-weight="800"
            fill="#475569"
          >
            <text x="72" y="123">蒸发</text>

            <rect
              data-bar="evap"
              x="112"
              y="114"
              width="0"
              height="11"
              rx="5"
              fill="#38BDF8"
            />

            <text x="72" y="147">蒸腾</text>

            <rect
              data-bar="transp"
              x="112"
              y="138"
              width="0"
              height="11"
              rx="5"
              fill="#22C55E"
            />

            <text x="72" y="171">下渗</text>

            <rect
              data-bar="infiltration"
              x="112"
              y="162"
              width="0"
              height="11"
              rx="5"
              fill="#A16207"
            />

            <text x="72" y="195">径流</text>

            <rect
              data-bar="surface"
              x="112"
              y="186"
              width="0"
              height="11"
              rx="5"
              fill="#F97316"
            />

            <text x="72" y="219">调蓄</text>

            <rect
              data-bar="storage"
              x="112"
              y="210"
              width="0"
              height="11"
              rx="5"
              fill="#8B5CF6"
            />
          </g>
        </g>
      </svg>

      <div
        class="gl-panel"
        data-process-panel
      >
        <div class="gl-panel-title">
          当前过程强度（相对教学单位）
        </div>

        <div class="gl-metric-grid">
          <div class="gl-metric">
            <strong data-metric="evap"></strong>
            <span>蒸发</span>
          </div>

          <div class="gl-metric">
            <strong data-metric="transp"></strong>
            <span>蒸腾</span>
          </div>

          <div class="gl-metric">
            <strong data-metric="infiltration"></strong>
            <span>下渗</span>
          </div>

          <div class="gl-metric">
            <strong data-metric="surface"></strong>
            <span>地表径流</span>
          </div>

          <div class="gl-metric">
            <strong data-metric="groundwater"></strong>
            <span>地下水补给</span>
          </div>
        </div>
      </div>

      <div
        class="gl-panel"
        data-impact-panel
        style="display:none"
      >
        <div class="gl-panel-title">
          相对自然流域的变化
        </div>

        <div class="gl-impact-grid">
          <div class="gl-impact-item">
            <b data-impact="infiltration"></b>

            <div class="gl-track">
              <i data-track="infiltration"></i>
            </div>

            <span>下渗变化</span>
          </div>

          <div class="gl-impact-item">
            <b data-impact="peak"></b>

            <div class="gl-track">
              <i data-track="peak"></i>
            </div>

            <span>洪峰压力</span>
          </div>

          <div class="gl-impact-item">
            <b data-impact="lag"></b>

            <div class="gl-track">
              <i data-track="lag"></i>
            </div>

            <span>汇流滞后</span>
          </div>

          <div class="gl-impact-item">
            <b data-impact="quality"></b>

            <div class="gl-track">
              <i data-track="quality"></i>
            </div>

            <span>水环境压力</span>
          </div>
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

      function rounded(value){
        return Math.round(value);
      }

      function signed(value){
        return (
          value>0
            ? '+'
            : ''
        )+
        rounded(value)+
        '%';
      }

      var precipitationInput=query('[data-p]');
      var temperatureInput=query('[data-t]');
      var vegetationInput=query('[data-v]');
      var urbanizationInput=query('[data-u]');
      var reservoirInput=query('[data-r]');
      var labelSwitch=query('[data-label-switch]');
      var autoSwitch=query('[data-auto-switch]');
      var result=query('[data-result]');
      var modeButtons=queryAll('[data-mode]');
      var scenarioButtons=queryAll('[data-scenario]');
      var rainLines=queryAll('[data-rain-line]');
      var labelGroup=query('[data-label-group]');
      var budgetOverlay=query('[data-budget-overlay]');
      var processPanel=query('[data-process-panel]');
      var impactPanel=query('[data-impact-panel]');
      var vegetationGroup=query('[data-vegetation-group]');
      var cityGroup=query('[data-city-group]');
      var reservoirGroup=query('[data-reservoir-group]');
      var reservoirWater=query('[data-reservoir-water]');
      var cloudGroup=query('[data-cloud-group]');

      if(
        !precipitationInput ||
        !temperatureInput ||
        !vegetationInput ||
        !urbanizationInput ||
        !reservoirInput ||
        !labelSwitch ||
        !autoSwitch ||
        !result ||
        !modeButtons.length ||
        !scenarioButtons.length ||
        !rainLines.length ||
        !labelGroup ||
        !budgetOverlay ||
        !processPanel ||
        !impactPanel ||
        !vegetationGroup ||
        !cityGroup ||
        !reservoirGroup ||
        !reservoirWater ||
        !cloudGroup
      ){
        return;
      }

      var currentMode='process';
      var scenarioOrder=[
        'natural',
        'urban',
        'restoration',
        'reservoir'
      ];

      var scenarioIndex=-1;
      var timer=null;

      function readState(){
        return {
          precipitation:Number(
            precipitationInput.value
          ),
          temperature:Number(
            temperatureInput.value
          ),
          vegetation:Number(
            vegetationInput.value
          ),
          urbanization:Number(
            urbanizationInput.value
          ),
          reservoir:Number(
            reservoirInput.value
          )
        };
      }

      function calculate(state){
        var input=Math.max(
          1,
          state.precipitation
        );

        var rawEvap=
          0.15+
          state.temperature/250+
          (
            100-state.vegetation
          )/1200;

        var rawTransp=
          0.03+
          state.vegetation/260;

        var rawInfiltration=Math.max(
          0.05,
          0.12+
          state.vegetation/240-
          state.urbanization/420
        );

        var rawSurface=Math.max(
          0.05,
          0.10+
          state.urbanization/170-
          state.vegetation/650
        );

        var rawStorage=
          0.02+
          state.reservoir/500;

        var rawTotal=
          rawEvap+
          rawTransp+
          rawInfiltration+
          rawSurface+
          rawStorage;

        var evap=
          input*
          rawEvap/
          rawTotal;

        var transp=
          input*
          rawTransp/
          rawTotal;

        var infiltration=
          input*
          rawInfiltration/
          rawTotal;

        var runoffBefore=
          input*
          rawSurface/
          rawTotal;

        var storageBase=
          input*
          rawStorage/
          rawTotal;

        var regulated=
          runoffBefore*
          (
            state.reservoir/100
          )*
          0.34;

        var surface=Math.max(
          0,
          runoffBefore-regulated
        );

        var storage=
          storageBase+
          regulated;

        var groundwater=
          infiltration*
          (
            0.46+
            state.vegetation/500
          );

        var peak=clamp(
          surface/input*145-
          state.reservoir*0.23,
          3,
          100
        );

        var lag=clamp(
          24+
          state.vegetation*0.38-
          state.urbanization*0.24+
          state.reservoir*0.31,
          5,
          96
        );

        var quality=clamp(
          88-
          state.urbanization*0.58+
          state.vegetation*0.22-
          state.precipitation*0.08,
          5,
          98
        );

        return {
          input:input,
          evap:evap,
          transp:transp,
          infiltration:infiltration,
          surface:surface,
          storage:storage,
          groundwater:groundwater,
          moisture:evap+transp,
          peak:peak,
          lag:lag,
          quality:quality
        };
      }

      function naturalBaseline(state){
        return calculate({
          precipitation:state.precipitation,
          temperature:state.temperature,
          vegetation:72,
          urbanization:8,
          reservoir:0
        });
      }

      function setFlow(
        name,
        value,
        maxValue
      ){
        var element=query(
          '[data-flow="'+
          name+
          '"]'
        );

        if(!element){
          return;
        }

        var ratio=clamp(
          value/
          Math.max(
            1,
            maxValue
          ),
          0,
          1
        );

        element.style.opacity=String(
          0.18+
          0.82*ratio
        );

        element.style.strokeWidth=String(
          2.2+
          4.3*ratio
        );

        element.style.animationDuration=
          String(
            clamp(
              2.4-ratio*1.45,
              0.75,
              2.4
            )
          )+
          's';
      }

      function setMetric(
        name,
        value
      ){
        var element=query(
          '[data-metric="'+
          name+
          '"]'
        );

        if(element){
          element.textContent=
            rounded(value).toString();
        }
      }

      function setBar(
        name,
        value,
        input
      ){
        var element=query(
          '[data-bar="'+
          name+
          '"]'
        );

        if(element){
          element.setAttribute(
            'width',
            String(
              clamp(
                value/input*165,
                2,
                165
              )
            )
          );
        }
      }

      function setImpact(
        name,
        text,
        value
      ){
        var label=query(
          '[data-impact="'+
          name+
          '"]'
        );

        var track=query(
          '[data-track="'+
          name+
          '"]'
        );

        if(label){
          label.textContent=text;
        }

        if(track){
          track.style.width=
            clamp(
              value,
              3,
              100
            )+
            '%';
        }
      }

      function describe(
        state,
        model,
        base
      ){
        var infiltrationDelta=
          (
            model.infiltration-
            base.infiltration
          )/
          base.infiltration*
          100;

        var surfaceDelta=
          (
            model.surface-
            base.surface
          )/
          Math.max(
            1,
            base.surface
          )*
          100;

        if(state.urbanization>=65){
          return '城市硬化使雨水更快汇入河道：'+
            '下渗通常减少，地表径流与洪峰压力上升。'+
            '透水铺装、下凹绿地等措施可以缓解，'+
            '但本模型不用于真实防洪决策。';
        }

        if(
          state.vegetation>=80 &&
          state.urbanization<=30
        ){
          return '植被恢复增强截留、蒸腾和土壤孔隙作用，'+
            '下渗增加、汇流变慢。'+
            '它通常有利于地下水补给和削减径流峰值，'+
            '实际效果还受土壤、坡度和降雨历时影响。';
        }

        if(state.reservoir>=65){
          return '水库把部分径流暂时储存并错峰释放，'+
            '可降低教学模型中的洪峰压力、延长汇流过程；'+
            '同时也可能改变下游径流、泥沙和生态节律，'+
            '不能据此进行真实工程调度。';
        }

        if(
          surfaceDelta>20 ||
          infiltrationDelta<-20
        ){
          return '当前流域比自然参照更偏向快速地表汇流，'+
            '地下水补给能力下降。'+
            '应同时观察不透水面比例、植被覆盖和调蓄条件，'+
            '不能只依据降水量判断。';
        }

        return '当前降水被分配到蒸发蒸腾、下渗、'+
          '地表径流和暂时储存等通道。'+
          '水循环各环节相互联系，'+
          '改变一个下垫面条件会引起多项过程联动。';
      }

      function renderMode(){
        Array.prototype.forEach.call(
          modeButtons,
          function(button){
            button.classList.toggle(
              'active',
              button.getAttribute(
                'data-mode'
              )===currentMode
            );
          }
        );

        budgetOverlay.style.display=
          currentMode==='budget'
            ? 'block'
            : 'none';

        impactPanel.style.display=
          currentMode==='impact'
            ? 'block'
            : 'none';

        processPanel.style.display=
          currentMode==='impact'
            ? 'none'
            : 'block';
      }

      function update(){
        if(!root.isConnected){
          if(timer){
            window.clearTimeout(timer);
            timer=null;
          }

          return;
        }

        var state=readState();
        var model=calculate(state);
        var base=naturalBaseline(state);

        query('[data-p-value]').textContent=
          rounded(state.precipitation)+
          ' 单位';

        query('[data-t-value]').textContent=
          rounded(state.temperature)+
          '℃';

        query('[data-v-value]').textContent=
          rounded(state.vegetation)+
          '%';

        query('[data-u-value]').textContent=
          rounded(state.urbanization)+
          '%';

        query('[data-r-value]').textContent=
          rounded(state.reservoir)+
          '%';

        var rainVisible=clamp(
          Math.round(
            state.precipitation/16
          ),
          1,
          rainLines.length
        );

        Array.prototype.forEach.call(
          rainLines,
          function(line,index){
            line.style.opacity=
              index<rainVisible
                ? String(
                    0.45+
                    0.5*
                    state.precipitation/
                    100
                  )
                : '0.08';
          }
        );

        cloudGroup.style.opacity=String(
          0.45+
          0.55*
          state.precipitation/
          100
        );

        vegetationGroup.style.opacity=String(
          0.16+
          0.84*
          state.vegetation/
          100
        );

        cityGroup.style.opacity=String(
          0.12+
          0.88*
          state.urbanization/
          100
        );

        reservoirGroup.style.opacity=String(
          0.18+
          0.82*
          state.reservoir/
          100
        );

        reservoirWater.style.transform=
          'translateY('+
          (
            18-
            state.reservoir*
            0.18
          )+
          'px)';

        labelGroup.style.display=
          labelSwitch.checked
            ? 'block'
            : 'none';

        setFlow(
          'evap',
          model.evap,
          model.input*0.42
        );

        setFlow(
          'transp',
          model.transp,
          model.input*0.36
        );

        setFlow(
          'transport',
          model.moisture,
          model.input*0.62
        );

        setFlow(
          'surface',
          model.surface,
          model.input*0.55
        );

        setFlow(
          'infiltration',
          model.infiltration,
          model.input*0.48
        );

        setFlow(
          'groundwater',
          model.groundwater,
          model.input*0.35
        );

        setFlow(
          'release',
          model.storage,
          model.input*0.35
        );

        setMetric(
          'evap',
          model.evap
        );

        setMetric(
          'transp',
          model.transp
        );

        setMetric(
          'infiltration',
          model.infiltration
        );

        setMetric(
          'surface',
          model.surface
        );

        setMetric(
          'groundwater',
          model.groundwater
        );

        setBar(
          'evap',
          model.evap,
          model.input
        );

        setBar(
          'transp',
          model.transp,
          model.input
        );

        setBar(
          'infiltration',
          model.infiltration,
          model.input
        );

        setBar(
          'surface',
          model.surface,
          model.input
        );

        setBar(
          'storage',
          model.storage,
          model.input
        );

        var infiltrationDelta=
          (
            model.infiltration-
            base.infiltration
          )/
          base.infiltration*
          100;

        var peakDelta=
          (
            model.peak-
            base.peak
          )/
          Math.max(
            1,
            base.peak
          )*
          100;

        var lagDelta=
          (
            model.lag-
            base.lag
          )/
          Math.max(
            1,
            base.lag
          )*
          100;

        var qualityPressure=
          100-
          model.quality;

        setImpact(
          'infiltration',
          signed(infiltrationDelta),
          Math.abs(infiltrationDelta)
        );

        setImpact(
          'peak',
          signed(peakDelta),
          model.peak
        );

        setImpact(
          'lag',
          signed(lagDelta),
          model.lag
        );

        setImpact(
          'quality',
          rounded(qualityPressure)+'%',
          qualityPressure
        );

        result.textContent=describe(
          state,
          model,
          base
        );

        renderMode();
      }

      function applyScenario(name){
        var scenarios={
          natural:[
            65,
            18,
            75,
            8,
            5
          ],
          urban:[
            65,
            22,
            20,
            85,
            5
          ],
          restoration:[
            65,
            18,
            90,
            18,
            15
          ],
          reservoir:[
            65,
            18,
            50,
            25,
            82
          ]
        };

        var data=scenarios[name];

        if(!data){
          return;
        }

        precipitationInput.value=
          String(data[0]);

        temperatureInput.value=
          String(data[1]);

        vegetationInput.value=
          String(data[2]);

        urbanizationInput.value=
          String(data[3]);

        reservoirInput.value=
          String(data[4]);

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

        update();
      }

      function markCustom(){
        Array.prototype.forEach.call(
          scenarioButtons,
          function(button){
            button.classList.remove(
              'active'
            );
          }
        );

        update();
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
          2700
        );
      }

      [
        precipitationInput,
        temperatureInput,
        vegetationInput,
        urbanizationInput,
        reservoirInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            markCustom
          );
        }
      );

      labelSwitch.addEventListener(
        'change',
        update
      );

      autoSwitch.addEventListener(
        'change',
        function(){
          schedule();
          update();
        }
      );

      Array.prototype.forEach.call(
        modeButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              currentMode=
                button.getAttribute(
                  'data-mode'
                ) ||
                'process';

              update();
            }
          );
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
                'natural';

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

      update();
      schedule();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_WATER_CYCLE:
GeographyLabTemplate[] = [
  {
    id: 'geography-water-cycle-human-impact',
    group: '🌊 水循环、河流与海洋系统',
    name: '水循环过程与人类活动影响',
    emoji: '💧',
    desc: '调节降水、气温、植被、城市硬化和水库调蓄，观察蒸发、蒸腾、下渗、径流与地下水补给的联动变化。',
    params: [
      {
        key: 'precipitation',
        label: '降水强度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 65,
        hint: '相对教学单位，表示一次过程或一个时段的水量输入。',
      },
      {
        key: 'temperature',
        label: '近地面气温',
        type: 'number',
        min: 0,
        max: 35,
        step: 1,
        defaultValue: 18,
        hint: '气温升高会增强蒸发需求，实际蒸发仍受可用水量影响。',
      },
      {
        key: 'vegetation',
        label: '植被覆盖率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 55,
        hint: '植被通常增强截留、蒸腾和下渗，并减缓地表汇流。',
      },
      {
        key: 'urbanization',
        label: '地表硬化率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 25,
        hint: '不透水面增多会使下渗减少、地表径流汇集加快。',
      },
      {
        key: 'reservoir',
        label: '水库调蓄强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 20,
        hint: '表示对部分地表径流进行拦蓄和错峰释放的相对能力。',
      },
      {
        key: 'showLabels',
        label: '显示过程标注',
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
    buildHTML: buildWaterCycleHTML,
  },
]
