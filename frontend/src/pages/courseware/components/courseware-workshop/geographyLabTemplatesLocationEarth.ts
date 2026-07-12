/**
 * geographyLabTemplatesLocationEarth.ts
 *
 * 地理第33批：
 *   经纬网、经纬度与半球定位。
 *
 * 教学边界：
 *   - 使用等距圆柱展开示意，不是精确地图投影；
 *   - 东西半球按20°W和160°E划分；
 *   - 不表示真实行政边界，不用于导航或测绘。
 */

import type {
  GeographyLabParamValue,
  GeographyLabTemplate,
} from './geographyLabUtils'

const SCRIPT_END = '</' + 'script>'

function num(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: number,
): number {
  const value = Number(params[key])
  return Number.isFinite(value) ? value : fallback
}

function bool(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]
  return typeof value === 'boolean' ? value : fallback
}

function buildLatitudeLongitudeHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const longitude = Math.max(
    -180,
    Math.min(180, num(params, 'longitude', 116)),
  )

  const latitude = Math.max(
    -90,
    Math.min(90, num(params, 'latitude', 40)),
  )

  const showGrid = bool(params, 'showGrid', true)
  const showLabels = bool(params, 'showLabels', true)

  return `
<div id="${rootId}" class="gl-latlon-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      box-sizing:border-box;
      overflow:hidden;
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
      background:linear-gradient(135deg,#CCFBF1,#EFF6FF);
      border-bottom:1px solid #99F6E4;
    }

    #${rootId} .gl-title{
      font-size:16px;
      font-weight:850;
      color:#115E59;
    }

    #${rootId} .gl-note{
      margin-left:auto;
      font-size:11.5px;
      color:#475569;
      white-space:nowrap;
    }

    #${rootId} .gl-body{
      height:calc(100% - 52px);
      display:grid;
      grid-template-columns:238px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      padding:14px;
      overflow:auto;
      border-right:1px solid #CCFBF1;
      background:linear-gradient(180deg,#F0FDFA,#F8FAFC);
    }

    #${rootId} .gl-stage{
      min-width:0;
      min-height:0;
      padding:10px;
      background:radial-gradient(
        circle at 50% 25%,
        #FFFFFF 0%,
        #F8FAFC 64%,
        #ECFEFF 100%
      );
    }

    #${rootId} .gl-row{
      margin-bottom:14px;
    }

    #${rootId} .gl-label-line{
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:8px;
      margin-bottom:6px;
    }

    #${rootId} .gl-label{
      font-size:12px;
      font-weight:700;
      color:#334155;
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
      border:none;
      border-radius:10px;
      background:#FFFFFF;
      color:#0F766E;
      border:1px solid #99F6E4;
      font-size:11.5px;
      font-weight:800;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      color:#FFFFFF;
      border-color:#0F766E;
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

    #${rootId} .gl-grid-line{
      stroke:#CBD5E1;
      stroke-width:1;
      stroke-dasharray:4 4;
      opacity:0.86;
    }

    #${rootId} .gl-grid-line.major{
      stroke-width:1.5;
      stroke-dasharray:none;
    }

    #${rootId} .gl-map-label{
      fill:#64748B;
      font-size:11px;
      font-weight:700;
    }

    #${rootId} .gl-info-label{
      fill:#64748B;
      font-size:11px;
      font-weight:700;
    }

    #${rootId} .gl-info-value{
      fill:#0F766E;
      font-size:14px;
      font-weight:850;
    }

    #${rootId} .gl-teaching-boundary{
      fill:#64748B;
      font-size:10.5px;
    }
  </style>

  <div class="gl-head">
    <div style="font-size:23px;">🧭</div>
    <div>
      <div class="gl-title">经纬网、经纬度与半球定位</div>
      <div style="font-size:11px;color:#64748B;margin-top:2px;">
        拖动经纬度，观察坐标表达与半球判定
      </div>
    </div>
    <div class="gl-note">教学简化模型 · 非真实地图投影</div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">经度</span>
          <span class="gl-value" data-role="longitude-value">0°</span>
        </div>
        <input
          type="range"
          min="-180"
          max="180"
          step="1"
          value="${longitude}"
          data-role="longitude"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">纬度</span>
          <span class="gl-value" data-role="latitude-value">0°</span>
        </div>
        <input
          type="range"
          min="-90"
          max="90"
          step="1"
          value="${latitude}"
          data-role="latitude"
        />
      </div>

      <div class="gl-button-grid">
        <button
          type="button"
          data-role="grid-toggle"
          data-active="${showGrid}"
        >
          经纬网
        </button>

        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          坐标标注
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

      <div class="gl-result" data-role="result">
        调整经纬度后显示定位结论。
      </div>
    </div>

    <div class="gl-stage">
      <svg
        class="gl-map-svg"
        viewBox="0 0 760 420"
        role="img"
        aria-label="经纬网与半球定位教学示意图"
      >
        <defs>
          <linearGradient id="${rootId}-ocean" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#CCFBF1"/>
          </linearGradient>

          <filter id="${rootId}-shadow" x="-40%" y="-40%" width="180%" height="180%">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#0F766E"
              flood-opacity="0.25"
            />
          </filter>
        </defs>

        <rect
          x="34"
          y="40"
          width="500"
          height="280"
          rx="18"
          fill="url(#${rootId}-ocean)"
          stroke="#67E8F9"
          stroke-width="2"
        />

        <g data-role="grid-group">
          <line class="gl-grid-line" x1="117.33" y1="40" x2="117.33" y2="320"/>
          <line class="gl-grid-line" x1="200.67" y1="40" x2="200.67" y2="320"/>
          <line class="gl-grid-line major" x1="284" y1="40" x2="284" y2="320" stroke="#0284C7"/>
          <line class="gl-grid-line" x1="367.33" y1="40" x2="367.33" y2="320"/>
          <line class="gl-grid-line" x1="450.67" y1="40" x2="450.67" y2="320"/>

          <line class="gl-grid-line" x1="34" y1="86.67" x2="534" y2="86.67"/>
          <line class="gl-grid-line" x1="34" y1="133.33" x2="534" y2="133.33"/>
          <line class="gl-grid-line major" x1="34" y1="180" x2="534" y2="180" stroke="#F97316"/>
          <line class="gl-grid-line" x1="34" y1="226.67" x2="534" y2="226.67"/>
          <line class="gl-grid-line" x1="34" y1="273.33" x2="534" y2="273.33"/>
        </g>

        <g data-role="label-group">
          <text class="gl-map-label" x="34" y="338">180°W</text>
          <text class="gl-map-label" x="110" y="338">120°W</text>
          <text class="gl-map-label" x="194" y="338">60°W</text>
          <text class="gl-map-label" x="276" y="338">0°</text>
          <text class="gl-map-label" x="356" y="338">60°E</text>
          <text class="gl-map-label" x="438" y="338">120°E</text>
          <text class="gl-map-label" x="498" y="338">180°E</text>

          <text class="gl-map-label" x="5" y="46">90°N</text>
          <text class="gl-map-label" x="5" y="137">30°N</text>
          <text class="gl-map-label" x="12" y="184">0°</text>
          <text class="gl-map-label" x="5" y="277">60°S</text>
          <text class="gl-map-label" x="5" y="320">90°S</text>
        </g>

        <line
          data-role="longitude-cross"
          x1="284"
          y1="40"
          x2="284"
          y2="320"
          stroke="#0F766E"
          stroke-width="2"
          opacity="0.9"
        />

        <line
          data-role="latitude-cross"
          x1="34"
          y1="180"
          x2="534"
          y2="180"
          stroke="#0F766E"
          stroke-width="2"
          opacity="0.9"
        />

        <circle
          data-role="point-halo"
          cx="284"
          cy="180"
          r="14"
          fill="#FFFFFF"
          opacity="0.84"
        />

        <circle
          data-role="point"
          cx="284"
          cy="180"
          r="8"
          fill="#F97316"
          stroke="#FFFFFF"
          stroke-width="3"
          filter="url(#${rootId}-shadow)"
        />

        <rect
          x="566"
          y="40"
          width="160"
          height="280"
          rx="18"
          fill="#FFFFFF"
          stroke="#99F6E4"
          stroke-width="2"
        />

        <text x="586" y="72" fill="#115E59" font-size="15" font-weight="850">
          定位结果
        </text>

        <text class="gl-info-label" x="586" y="105">坐标</text>
        <text class="gl-info-value" x="586" y="127" data-role="coordinate-text">—</text>

        <text class="gl-info-label" x="586" y="163">南北半球</text>
        <text class="gl-info-value" x="586" y="185" data-role="north-south-text">—</text>

        <text class="gl-info-label" x="586" y="221">东西半球</text>
        <text class="gl-info-value" x="586" y="243" data-role="east-west-text">—</text>

        <text class="gl-info-label" x="586" y="279">纬度带</text>
        <text class="gl-info-value" x="586" y="301" data-role="zone-text">—</text>

        <text class="gl-teaching-boundary" x="35" y="378">
          东西半球按20°W和160°E划分；展开图仅用于经纬度与方位教学。
        </text>

        <text class="gl-teaching-boundary" x="35" y="397">
          本图不表示真实海陆轮廓、行政边界或精确地图投影。
        </text>
      </svg>
    </div>
  </div>

  <script>
    (function(){
      var root = document.getElementById('${rootId}');
      if(!root)return;

      var longitudeInput = root.querySelector('[data-role="longitude"]');
      var latitudeInput = root.querySelector('[data-role="latitude"]');
      var longitudeValue = root.querySelector('[data-role="longitude-value"]');
      var latitudeValue = root.querySelector('[data-role="latitude-value"]');

      var gridToggle = root.querySelector('[data-role="grid-toggle"]');
      var labelToggle = root.querySelector('[data-role="label-toggle"]');
      var autoToggle = root.querySelector('[data-role="auto-toggle"]');
      var resetButton = root.querySelector('[data-role="reset"]');

      var gridGroup = root.querySelector('[data-role="grid-group"]');
      var labelGroup = root.querySelector('[data-role="label-group"]');

      var longitudeCross = root.querySelector('[data-role="longitude-cross"]');
      var latitudeCross = root.querySelector('[data-role="latitude-cross"]');
      var point = root.querySelector('[data-role="point"]');
      var pointHalo = root.querySelector('[data-role="point-halo"]');

      var coordinateText = root.querySelector('[data-role="coordinate-text"]');
      var northSouthText = root.querySelector('[data-role="north-south-text"]');
      var eastWestText = root.querySelector('[data-role="east-west-text"]');
      var zoneText = root.querySelector('[data-role="zone-text"]');
      var result = root.querySelector('[data-role="result"]');

      if(
        !longitudeInput ||
        !latitudeInput ||
        !longitudeValue ||
        !latitudeValue ||
        !gridToggle ||
        !labelToggle ||
        !autoToggle ||
        !resetButton ||
        !gridGroup ||
        !labelGroup ||
        !longitudeCross ||
        !latitudeCross ||
        !point ||
        !pointHalo ||
        !coordinateText ||
        !northSouthText ||
        !eastWestText ||
        !zoneText ||
        !result
      ){
        return;
      }

      var initialLongitude = ${longitude};
      var initialLatitude = ${latitude};

      var state = {
        showGrid: ${showGrid},
        showLabels: ${showLabels},
        auto: false,
        startTime: 0,
        raf: 0
      };

      function clamp(value,min,max){
        return Math.max(min,Math.min(max,value));
      }

      function formatLongitude(value){
        var rounded = Math.round(Math.abs(value));
        if(Math.abs(value) < 0.5)return '0°';
        return rounded + '°' + (value > 0 ? 'E' : 'W');
      }

      function formatLatitude(value){
        var rounded = Math.round(Math.abs(value));
        if(Math.abs(value) < 0.5)return '0°';
        return rounded + '°' + (value > 0 ? 'N' : 'S');
      }

      function northSouthHemisphere(latitude){
        if(Math.abs(latitude) < 0.5)return '赤道';
        return latitude > 0 ? '北半球' : '南半球';
      }

      function eastWestHemisphere(longitude){
        if(
          Math.abs(longitude + 20) < 0.5 ||
          Math.abs(longitude - 160) < 0.5
        ){
          return '半球分界线';
        }

        return longitude > -20 && longitude < 160
          ? '东半球'
          : '西半球';
      }

      function latitudeZone(latitude){
        var absolute = Math.abs(latitude);
        if(absolute < 23.5)return '低纬度';
        if(absolute < 66.5)return '中纬度';
        return '高纬度';
      }

      function render(){
        if(!root.isConnected){
          state.auto = false;
          return;
        }

        var longitude = clamp(
          parseFloat(longitudeInput.value) || 0,
          -180,
          180
        );

        var latitude = clamp(
          parseFloat(latitudeInput.value) || 0,
          -90,
          90
        );

        var x = 34 + ((longitude + 180) / 360) * 500;
        var y = 40 + ((90 - latitude) / 180) * 280;

        longitudeValue.textContent = formatLongitude(longitude);
        latitudeValue.textContent = formatLatitude(latitude);

        longitudeCross.setAttribute('x1',String(x));
        longitudeCross.setAttribute('x2',String(x));

        latitudeCross.setAttribute('y1',String(y));
        latitudeCross.setAttribute('y2',String(y));

        point.setAttribute('cx',String(x));
        point.setAttribute('cy',String(y));
        pointHalo.setAttribute('cx',String(x));
        pointHalo.setAttribute('cy',String(y));

        coordinateText.textContent =
          formatLongitude(longitude) + '  ' + formatLatitude(latitude);

        northSouthText.textContent =
          northSouthHemisphere(latitude);

        eastWestText.textContent =
          eastWestHemisphere(longitude);

        zoneText.textContent =
          latitudeZone(latitude);

        gridGroup.style.display =
          state.showGrid ? '' : 'none';

        labelGroup.style.display =
          state.showLabels ? '' : 'none';

        gridToggle.setAttribute(
          'data-active',
          state.showGrid ? 'true' : 'false'
        );

        labelToggle.setAttribute(
          'data-active',
          state.showLabels ? 'true' : 'false'
        );

        autoToggle.setAttribute(
          'data-active',
          state.auto ? 'true' : 'false'
        );

        result.textContent =
          '该点位于' +
          northSouthHemisphere(latitude) +
          '、' +
          eastWestHemisphere(longitude) +
          '，属于' +
          latitudeZone(latitude) +
          '。经度表示相对本初子午线的东西位置，纬度表示相对赤道的南北位置。';
      }

      function stopAuto(){
        state.auto = false;

        if(state.raf){
          cancelAnimationFrame(state.raf);
          state.raf = 0;
        }

        render();
      }

      function animate(timestamp){
        if(!root.isConnected){
          state.auto = false;
          return;
        }

        if(!state.auto)return;

        if(!state.startTime){
          state.startTime = timestamp;
        }

        var elapsed = timestamp - state.startTime;
        var phase = elapsed / 1000;

        longitudeInput.value = String(
          Math.round(((phase * 24) % 360) - 180)
        );

        latitudeInput.value = String(
          Math.round(Math.sin(phase * 0.72) * 72)
        );

        render();
        state.raf = requestAnimationFrame(animate);
      }

      longitudeInput.addEventListener('input',function(){
        if(state.auto)stopAuto();
        render();
      });

      latitudeInput.addEventListener('input',function(){
        if(state.auto)stopAuto();
        render();
      });

      gridToggle.addEventListener('click',function(){
        state.showGrid = !state.showGrid;
        render();
      });

      labelToggle.addEventListener('click',function(){
        state.showLabels = !state.showLabels;
        render();
      });

      autoToggle.addEventListener('click',function(){
        if(state.auto){
          stopAuto();
          return;
        }

        state.auto = true;
        state.startTime = 0;
        render();
        state.raf = requestAnimationFrame(animate);
      });

      resetButton.addEventListener('click',function(){
        stopAuto();
        longitudeInput.value = String(initialLongitude);
        latitudeInput.value = String(initialLatitude);
        state.showGrid = ${showGrid};
        state.showLabels = ${showLabels};
        render();
      });

      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_LOCATION_EARTH:
GeographyLabTemplate[] = [
  {
    id: 'geography-latitude-longitude-hemisphere',
    group: '🧭 基础定位与地球运动',
    name: '经纬网、经纬度与半球定位',
    emoji: '🧭',
    desc: '拖动经纬度，观察坐标表达、纬度带与南北、东西半球判定。',
    params: [
      {
        key: 'longitude',
        label: '初始经度',
        type: 'number',
        min: -180,
        max: 180,
        step: 1,
        defaultValue: 116,
        hint: '正值表示东经，负值表示西经。',
      },
      {
        key: 'latitude',
        label: '初始纬度',
        type: 'number',
        min: -90,
        max: 90,
        step: 1,
        defaultValue: 40,
        hint: '正值表示北纬，负值表示南纬。',
      },
      {
        key: 'showGrid',
        label: '显示经纬网',
        type: 'boolean',
        defaultValue: true,
      },
      {
        key: 'showLabels',
        label: '显示坐标标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildLatitudeLongitudeHTML,
  },
]
