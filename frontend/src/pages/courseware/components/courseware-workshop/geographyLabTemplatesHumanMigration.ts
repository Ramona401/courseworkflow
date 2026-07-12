/**
 * geographyLabTemplatesHumanMigration.ts
 *
 * 地理第38批B2：人口迁移、推拉因素与迁移流。
 *
 * 教学目标：
 * - 理解迁出地推力、迁入地拉力和迁移阻力共同影响迁移决策；
 * - 比较乡村到城市、区域间、季节性、国际和环境迁移等课堂情境；
 * - 分析人口迁移对迁出地、迁入地和迁移者的双向影响。
 *
 * 教学边界：
 * - 所有地点、流量、得分和影响均为离线课堂示意；
 * - 模型不包含真实国家、城市、边境、签证和劳动力市场数据；
 * - 不用于真实人口迁移预测、移民决策、就业选择或公共政策判断。
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

function buildHumanMigrationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'rural-urban',
    'regional-industry',
    'seasonal',
    'international',
    'environmental',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'rural-urban',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'rural-urban'

  const pushIntensity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'pushIntensity', 6),
    ),
  )

  const pullIntensity = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'pullIntensity', 8),
    ),
  )

  const distanceCost = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'distanceCost', 4),
    ),
  )

  const networkSupport = Math.max(
    0,
    Math.min(
      10,
      numberValue(params, 'networkSupport', 5),
    ),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-human-migration-root">
  <style>
    #${rootId}{
      width:100%;
      height:100%;
      overflow:hidden;
      box-sizing:border-box;
      border:1px solid #BAE6FD;
      border-radius:18px;
      background:#FFFFFF;
      color:#0F172A;
      font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      box-shadow:0 12px 34px rgba(3,105,161,0.11);
    }

    #${rootId} *{
      box-sizing:border-box;
    }

    #${rootId} .gl-head{
      height:56px;
      padding:0 18px;
      display:flex;
      align-items:center;
      gap:12px;
      border-bottom:1px solid #BAE6FD;
      background:linear-gradient(
        135deg,
        #F0F9FF,
        #ECFEFF 55%,
        #F0FDF4
      );
    }

    #${rootId} .gl-title{
      color:#0C4A6E;
      font-size:16px;
      font-weight:880;
    }

    #${rootId} .gl-subtitle{
      margin-top:2px;
      color:#64748B;
      font-size:11px;
    }

    #${rootId} .gl-note{
      margin-left:auto;
      padding:5px 10px;
      border:1px solid #A5F3FC;
      border-radius:999px;
      background:#FFFFFF;
      color:#0E7490;
      font-size:11px;
      font-weight:750;
      white-space:nowrap;
    }

    #${rootId} .gl-body{
      height:calc(100% - 56px);
      display:grid;
      grid-template-columns:278px minmax(0,1fr);
    }

    #${rootId} .gl-controls{
      min-height:0;
      padding:13px;
      overflow:auto;
      border-right:1px solid #CFFAFE;
      background:linear-gradient(
        180deg,
        #F0F9FF,
        #ECFEFF 58%,
        #F0FDF4
      );
    }

    #${rootId} .gl-stage{
      min-width:0;
      min-height:0;
      display:grid;
      grid-template-rows:46px minmax(0,1fr);
      padding:8px;
      background:radial-gradient(
        circle at 48% 22%,
        #FFFFFF 0%,
        #F8FAFC 62%,
        #E0F2FE 100%
      );
    }

    #${rootId} .gl-section-title{
      margin:1px 0 8px;
      color:#075985;
      font-size:11.5px;
      font-weight:850;
    }

    #${rootId} .gl-scenario-grid{
      display:grid;
      grid-template-columns:1fr 1fr;
      gap:6px;
      margin-bottom:12px;
    }

    #${rootId} .gl-scenario-grid button:last-child{
      grid-column:1/-1;
    }

    #${rootId} .gl-row{
      margin-bottom:10px;
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
      font-size:11.5px;
      font-weight:730;
    }

    #${rootId} .gl-value{
      min-width:48px;
      padding:3px 7px;
      border-radius:999px;
      background:#CFFAFE;
      color:#0E7490;
      font-size:11px;
      font-weight:850;
      text-align:center;
    }

    #${rootId} input[type=range]{
      width:100%;
      height:6px;
      margin:0;
      appearance:none;
      border-radius:999px;
      outline:none;
      background:linear-gradient(
        90deg,
        #7DD3FC,
        #86EFAC
      );
      cursor:pointer;
    }

    #${rootId} input[type=range]::-webkit-slider-thumb{
      width:16px;
      height:16px;
      appearance:none;
      border:2px solid #FFFFFF;
      border-radius:50%;
      background:linear-gradient(
        135deg,
        #0284C7,
        #16A34A
      );
      box-shadow:0 1px 5px rgba(3,105,161,0.42);
    }

    #${rootId} button{
      min-height:32px;
      padding:6px 7px;
      border:1px solid #BAE6FD;
      border-radius:9px;
      background:#FFFFFF;
      color:#0369A1;
      font-size:10.7px;
      font-weight:790;
      cursor:pointer;
    }

    #${rootId} button[data-active="true"]{
      border-color:#0284C7;
      color:#FFFFFF;
      background:linear-gradient(
        135deg,
        #0284C7,
        #0891B2 55%,
        #16A34A
      );
      box-shadow:0 5px 13px rgba(2,132,199,0.22);
    }

    #${rootId} .gl-action-grid{
      display:grid;
      grid-template-columns:1fr 1fr;
      gap:6px;
      margin:10px 0;
    }

    #${rootId} .gl-result{
      margin-top:8px;
      padding:10px;
      border:1px solid #BAE6FD;
      border-radius:12px;
      background:linear-gradient(
        135deg,
        #F0F9FF,
        #F0FDF4
      );
      color:#334155;
      font-size:11.2px;
      font-weight:620;
      line-height:1.52;
    }

    #${rootId} .gl-view-toolbar{
      display:grid;
      grid-template-columns:repeat(4,minmax(0,1fr));
      gap:7px;
      align-items:center;
      padding:0 3px 7px;
      border-bottom:1px solid #E2E8F0;
    }

    #${rootId} .gl-view-toolbar button{
      min-height:32px;
      font-size:11px;
    }

    #${rootId} .gl-canvas-wrap{
      min-width:0;
      min-height:0;
      overflow:hidden;
      border:1px solid #BAE6FD;
      border-radius:14px;
      background:#FFFFFF;
    }

    #${rootId} .gl-migration-canvas{
      width:100%;
      height:100%;
      display:block;
    }

    @media(max-width:900px){
      #${rootId} .gl-body{
        grid-template-columns:238px minmax(0,1fr);
      }

      #${rootId} .gl-note{
        display:none;
      }
    }
  </style>

  <div class="gl-head">
    <div style="font-size:24px;">
      🧳
    </div>

    <div>
      <div class="gl-title">
        人口迁移、推拉因素与迁移流
      </div>

      <div class="gl-subtitle">
        比较迁出地推力、迁入地拉力、迁移阻力和社会网络支持
      </div>
    </div>

    <div class="gl-note">
      离线课堂模型 · 不用于真实迁移决策
    </div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">
        典型迁移情境
      </div>

      <div class="gl-scenario-grid">
        <button
          type="button"
          data-scenario="rural-urban"
        >
          乡村到城市
        </button>

        <button
          type="button"
          data-scenario="regional-industry"
        >
          区域产业迁移
        </button>

        <button
          type="button"
          data-scenario="seasonal"
        >
          季节性迁移
        </button>

        <button
          type="button"
          data-scenario="international"
        >
          国际迁移
        </button>

        <button
          type="button"
          data-scenario="environmental"
        >
          环境压力迁移
        </button>
      </div>

      <div class="gl-section-title">
        迁移决策参数
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            迁出地推力
          </span>

          <span
            class="gl-value"
            data-role="push-value"
          >
            6
          </span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${pushIntensity}"
          data-role="push"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            迁入地拉力
          </span>

          <span
            class="gl-value"
            data-role="pull-value"
          >
            8
          </span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${pullIntensity}"
          data-role="pull"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            距离与迁移成本
          </span>

          <span
            class="gl-value"
            data-role="cost-value"
          >
            4
          </span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${distanceCost}"
          data-role="cost"
        />
      </div>

      <div class="gl-row">
        <div class="gl-label-line">
          <span class="gl-label">
            社会网络支持
          </span>

          <span
            class="gl-value"
            data-role="network-value"
          >
            5
          </span>
        </div>

        <input
          type="range"
          min="0"
          max="10"
          step="1"
          value="${networkSupport}"
          data-role="network"
        />
      </div>

      <div class="gl-action-grid">
        <button
          type="button"
          data-role="label-toggle"
          data-active="${showLabels}"
        >
          因素标注
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
          data-role="compare"
        >
          切换下一情境
        </button>
      </div>

      <div
        class="gl-result"
        data-role="result"
      >
        迁移决策通常由推力、拉力、阻力和个人社会联系共同作用。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button
          type="button"
          data-view="balance"
        >
          推拉平衡
        </button>

        <button
          type="button"
          data-view="flow"
        >
          迁移流
        </button>

        <button
          type="button"
          data-view="impact"
        >
          区域影响
        </button>

        <button
          type="button"
          data-view="types"
        >
          迁移类型
        </button>
      </div>

      <div class="gl-canvas-wrap">
        <canvas
          class="gl-migration-canvas"
          width="980"
          height="570"
          data-role="canvas"
          aria-label="人口迁移推拉因素与迁移流教学示意图"
        ></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root =
        document.getElementById('${rootId}');

      if(!root)return;

      var pushInput =
        root.querySelector('[data-role="push"]');

      var pullInput =
        root.querySelector('[data-role="pull"]');

      var costInput =
        root.querySelector('[data-role="cost"]');

      var networkInput =
        root.querySelector('[data-role="network"]');

      var pushValue =
        root.querySelector('[data-role="push-value"]');

      var pullValue =
        root.querySelector('[data-role="pull-value"]');

      var costValue =
        root.querySelector('[data-role="cost-value"]');

      var networkValue =
        root.querySelector('[data-role="network-value"]');

      var scenarioButtons =
        root.querySelectorAll('[data-scenario]');

      var viewButtons =
        root.querySelectorAll('[data-view]');

      var labelToggle =
        root.querySelector('[data-role="label-toggle"]');

      var autoToggle =
        root.querySelector('[data-role="auto-toggle"]');

      var resetButton =
        root.querySelector('[data-role="reset"]');

      var compareButton =
        root.querySelector('[data-role="compare"]');

      var result =
        root.querySelector('[data-role="result"]');

      var canvas =
        root.querySelector('[data-role="canvas"]');

      if(
        !pushInput ||
        !pullInput ||
        !costInput ||
        !networkInput ||
        !pushValue ||
        !pullValue ||
        !costValue ||
        !networkValue ||
        !scenarioButtons.length ||
        !viewButtons.length ||
        !labelToggle ||
        !autoToggle ||
        !resetButton ||
        !compareButton ||
        !result ||
        !canvas
      ){
        return;
      }

      var context =
        canvas.getContext('2d');

      if(!context)return;

      var width = canvas.width;
      var height = canvas.height;

      var scenarios = [
        {
          key:'rural-urban',
          name:'乡村到城市',
          origin:'乡村地区',
          destination:'城市地区',
          push:6,
          pull:8,
          cost:3,
          network:5,
          color:'#0284C7',
          view:'flow'
        },
        {
          key:'regional-industry',
          name:'区域产业迁移',
          origin:'传统产业区',
          destination:'新兴产业区',
          push:5,
          pull:8,
          cost:4,
          network:6,
          color:'#7C3AED',
          view:'impact'
        },
        {
          key:'seasonal',
          name:'季节性迁移',
          origin:'常住地',
          destination:'季节性就业地',
          push:4,
          pull:7,
          cost:3,
          network:8,
          color:'#16A34A',
          view:'types'
        },
        {
          key:'international',
          name:'国际迁移',
          origin:'迁出国家或地区',
          destination:'迁入国家或地区',
          push:6,
          pull:8,
          cost:8,
          network:5,
          color:'#EA580C',
          view:'balance'
        },
        {
          key:'environmental',
          name:'环境压力迁移',
          origin:'环境压力区',
          destination:'相对安全区',
          push:9,
          pull:5,
          cost:6,
          network:3,
          color:'#DC2626',
          view:'flow'
        }
      ];

      var initial = {
        scenario:'${scenario}',
        push:${pushIntensity},
        pull:${pullIntensity},
        cost:${distanceCost},
        network:${networkSupport},
        showLabels:${showLabels}
      };

      var state = {
        scenario:initial.scenario,
        view:'balance',
        showLabels:initial.showLabels,
        auto:false,
        startedAt:0,
        raf:0,
        phase:0,
        compareIndex:0
      };

      function clamp(value,min,max){
        return Math.max(
          min,
          Math.min(max,value)
        );
      }

      function lerp(a,b,t){
        return a+(b-a)*t;
      }

      function ease(t){
        var progress =
          clamp(t,0,1);

        return progress<0.5
          ? 2*progress*progress
          : 1-
            Math.pow(
              -2*progress+2,
              2
            )/2;
      }

      function roundRect(
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

      function box(
        x,
        y,
        w,
        h,
        radius,
        fill,
        stroke
      ){
        roundRect(
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
          context.lineWidth=1.2;
          context.stroke();
        }
      }

      function text(
        value,
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
          String(value),
          x,
          y
        );

        context.restore();
      }

      function line(
        x1,
        y1,
        x2,
        y2,
        color,
        lineWidth,
        dash
      ){
        context.save();
        context.strokeStyle=color;
        context.lineWidth=lineWidth || 1.5;
        context.setLineDash(dash || []);
        context.beginPath();
        context.moveTo(x1,y1);
        context.lineTo(x2,y2);
        context.stroke();
        context.restore();
      }

      function circle(
        x,
        y,
        radius,
        fill,
        stroke
      ){
        context.beginPath();
        context.arc(
          x,
          y,
          radius,
          0,
          Math.PI*2
        );

        if(fill){
          context.fillStyle=fill;
          context.fill();
        }

        if(stroke){
          context.strokeStyle=stroke;
          context.lineWidth=2;
          context.stroke();
        }
      }

      function scenarioByKey(key){
        var found =
          scenarios[0];

        scenarios.forEach(
          function(item){
            if(item.key===key){
              found=item;
            }
          }
        );

        return found;
      }

      function values(){
        return {
          push:clamp(
            Number(pushInput.value) || 0,
            0,
            10
          ),
          pull:clamp(
            Number(pullInput.value) || 0,
            0,
            10
          ),
          cost:clamp(
            Number(costInput.value) || 0,
            0,
            10
          ),
          network:clamp(
            Number(networkInput.value) || 0,
            0,
            10
          )
        };
      }

      function setInputs(value){
        pushInput.value =
          String(Math.round(value.push));

        pullInput.value =
          String(Math.round(value.pull));

        costInput.value =
          String(Math.round(value.cost));

        networkInput.value =
          String(Math.round(value.network));
      }

      function derive(value){
        var drive =
          value.push*0.42+
          value.pull*0.46+
          value.network*0.25-
          value.cost*0.48;

        var score =
          clamp(
            Math.round(
              (drive+4)*8
            ),
            0,
            100
          );

        var volume =
          Math.round(
            20+
            score*2.1
          );

        var direction =
          score>=68
            ? '迁移流较强'
            : (
              score>=42
                ? '存在一定迁移流'
                : '迁移流相对较弱'
            );

        var barrier =
          value.cost>=7
            ? '距离、制度或经济成本构成较强阻力'
            : (
              value.cost>=4
                ? '迁移阻力处于中等水平'
                : '迁移阻力相对较小'
            );

        var network =
          value.network>=7
            ? '社会网络显著降低信息和适应成本'
            : (
              value.network>=4
                ? '社会网络提供一定支持'
                : '社会网络支持较弱'
            );

        return {
          score:score,
          volume:volume,
          direction:direction,
          barrier:barrier,
          network:network
        };
      }

      function background(
        titleValue,
        subtitle
      ){
        var gradient =
          context.createLinearGradient(
            0,
            0,
            width,
            height
          );

        gradient.addColorStop(
          0,
          '#FFFFFF'
        );

        gradient.addColorStop(
          0.58,
          '#F8FAFC'
        );

        gradient.addColorStop(
          1,
          '#E0F2FE'
        );

        context.fillStyle=gradient;
        context.fillRect(
          0,
          0,
          width,
          height
        );

        text(
          titleValue,
          28,
          31,
          18,
          '#0C4A6E',
          880,
          'left'
        );

        text(
          subtitle,
          28,
          55,
          11.5,
          '#64748B',
          620,
          'left'
        );
      }

      function card(
        x,
        y,
        w,
        label,
        value,
        color,
        desc
      ){
        box(
          x,
          y,
          w,
          72,
          12,
          'rgba(255,255,255,0.94)',
          '#BAE6FD'
        );

        text(
          label,
          x+14,
          y+17,
          10.5,
          '#64748B',
          720,
          'left'
        );

        text(
          value,
          x+14,
          y+40,
          20,
          color,
          880,
          'left'
        );

        text(
          desc,
          x+14,
          y+59,
          9.5,
          '#64748B',
          600,
          'left'
        );
      }

      function arrow(
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

        var head=14;

        context.save();
        context.strokeStyle=color;
        context.fillStyle=color;
        context.lineWidth=lineWidth || 4;
        context.lineCap='round';

        context.beginPath();
        context.moveTo(x1,y1);
        context.lineTo(x2,y2);
        context.stroke();

        context.beginPath();
        context.moveTo(x2,y2);

        context.lineTo(
          x2-
          head*
          Math.cos(
            angle-Math.PI/6
          ),
          y2-
          head*
          Math.sin(
            angle-Math.PI/6
          )
        );

        context.lineTo(
          x2-
          head*
          Math.cos(
            angle+Math.PI/6
          ),
          y2-
          head*
          Math.sin(
            angle+Math.PI/6
          )
        );

        context.closePath();
        context.fill();
        context.restore();
      }

      function balanceView(
        item,
        value,
        derived
      ){
        background(
          '推力—拉力—阻力平衡',
          '把迁移决策拆分为迁出地推力、迁入地拉力、迁移阻力和社会网络支持。'
        );

        card(
          28,
          78,
          210,
          '迁移倾向指数',
          derived.score,
          '#0284C7',
          '0—100课堂示意值'
        );

        card(
          250,
          78,
          210,
          '迁出地推力',
          Math.round(value.push),
          '#DC2626',
          '就业、资源、环境与家庭压力'
        );

        card(
          472,
          78,
          210,
          '迁入地拉力',
          Math.round(value.pull),
          '#16A34A',
          '就业、教育、收入与服务机会'
        );

        card(
          694,
          78,
          258,
          '迁移阻力',
          Math.round(value.cost),
          '#EA580C',
          '距离、成本、制度与信息障碍'
        );

        var centerX=490;
        var centerY=338;

        box(
          centerX-72,
          centerY-54,
          144,
          108,
          20,
          '#FFFFFF',
          '#7DD3FC'
        );

        text(
          '迁移决策',
          centerX,
          centerY-13,
          15,
          '#075985',
          880,
          'center'
        );

        text(
          derived.direction,
          centerX,
          centerY+17,
          10.5,
          '#475569',
          720,
          'center'
        );

        var forces = [
          {
            label:'推力',
            value:value.push,
            x:78,
            y:234,
            color:'#DC2626',
            targetX:centerX-78,
            targetY:centerY-24
          },
          {
            label:'拉力',
            value:value.pull,
            x:902,
            y:234,
            color:'#16A34A',
            targetX:centerX+78,
            targetY:centerY-24
          },
          {
            label:'社会网络',
            value:value.network,
            x:122,
            y:478,
            color:'#0891B2',
            targetX:centerX-78,
            targetY:centerY+28
          },
          {
            label:'迁移阻力',
            value:value.cost,
            x:858,
            y:478,
            color:'#EA580C',
            targetX:centerX+78,
            targetY:centerY+28
          }
        ];

        forces.forEach(
          function(force,index){
            var radius =
              32+
              force.value*2.4;

            circle(
              force.x,
              force.y,
              radius,
              '#FFFFFF',
              force.color
            );

            text(
              force.label,
              force.x,
              force.y-8,
              11,
              force.color,
              850,
              'center'
            );

            text(
              Math.round(force.value),
              force.x,
              force.y+14,
              18,
              force.color,
              880,
              'center'
            );

            if(index===3){
              arrow(
                force.targetX,
                force.targetY,
                force.x-radius-8,
                force.y,
                force.color,
                2+force.value*0.35
              );
            }else{
              arrow(
                force.x+
                (
                  force.x<centerX
                    ? radius+8
                    : -radius-8
                ),
                force.y,
                force.targetX,
                force.targetY,
                force.color,
                2+force.value*0.35
              );
            }
          }
        );

        box(
          232,
          510,
          516,
          38,
          10,
          '#F0F9FF',
          '#BAE6FD'
        );

        text(
          '迁移动力由推力、拉力、网络支持和迁移阻力共同决定',
          490,
          529,
          10.5,
          '#475569',
          690,
          'center'
        );
      }

      function drawPlace(
        x,
        y,
        w,
        h,
        titleValue,
        subtitle,
        color,
        icon
      ){
        box(
          x,
          y,
          w,
          h,
          18,
          '#FFFFFF',
          color
        );

        box(
          x+18,
          y+18,
          52,
          52,
          14,
          color,
          null
        );

        text(
          icon,
          x+44,
          y+44,
          24,
          '#FFFFFF',
          800,
          'center'
        );

        text(
          titleValue,
          x+86,
          y+34,
          14,
          color,
          850,
          'left'
        );

        text(
          subtitle,
          x+86,
          y+57,
          10,
          '#64748B',
          650,
          'left'
        );
      }

      function movingPerson(
        progress,
        startX,
        startY,
        endX,
        endY,
        color
      ){
        var x =
          lerp(
            startX,
            endX,
            progress
          );

        var arc =
          Math.sin(
            progress*Math.PI
          )*
          -54;

        var y =
          lerp(
            startY,
            endY,
            progress
          )+
          arc;

        circle(
          x,
          y,
          7,
          '#FFFFFF',
          color
        );

        circle(
          x,
          y-3,
          3,
          color,
          null
        );

        line(
          x,
          y+4,
          x,
          y+12,
          color,
          2,
          []
        );

        line(
          x,
          y+7,
          x-5,
          y+12,
          color,
          2,
          []
        );

        line(
          x,
          y+7,
          x+5,
          y+12,
          color,
          2,
          []
        );
      }

      function flowView(
        item,
        value,
        derived
      ){
        background(
          '迁移流方向与规模',
          '用离线示意节点观察迁出地、迁入地、流向和迁移规模。'
        );

        card(
          28,
          78,
          210,
          '情境',
          item.name,
          item.color,
          '课堂比较场景'
        );

        card(
          250,
          78,
          210,
          '迁移倾向',
          derived.score,
          item.color,
          '综合推拉与阻力'
        );

        card(
          472,
          78,
          210,
          '迁移流规模',
          derived.volume,
          item.color,
          '相对流量单位'
        );

        card(
          694,
          78,
          258,
          '主要障碍',
          value.cost>=7
            ? '较强'
            : '中低',
          '#EA580C',
          derived.barrier
        );

        drawPlace(
          64,
          212,
          286,
          126,
          item.origin,
          '迁出地：推力因素较突出',
          '#DC2626',
          'A'
        );

        drawPlace(
          630,
          212,
          286,
          126,
          item.destination,
          '迁入地：拉力因素较突出',
          '#16A34A',
          'B'
        );

        var lineWidth =
          3+
          derived.score/16;

        arrow(
          366,
          275,
          614,
          275,
          item.color,
          lineWidth
        );

        text(
          '主要迁移方向',
          490,
          244,
          11,
          item.color,
          830,
          'center'
        );

        if(state.showLabels){
          text(
            '推力 '+
            Math.round(value.push),
            210,
            365,
            10,
            '#DC2626',
            760,
            'center'
          );

          text(
            '阻力 '+
            Math.round(value.cost),
            490,
            365,
            10,
            '#EA580C',
            760,
            'center'
          );

          text(
            '拉力 '+
            Math.round(value.pull),
            770,
            365,
            10,
            '#16A34A',
            760,
            'center'
          );
        }

        var people =
          Math.max(
            2,
            Math.round(
              derived.score/14
            )
          );

        for(
          var index=0;
          index<people;
          index+=1
        ){
          movingPerson(
            (
              state.phase+
              index/people
            )%1,
            370,
            275,
            610,
            275,
            item.color
          );
        }

        box(
          64,
          408,
          852,
          102,
          14,
          '#FFFFFF',
          '#BAE6FD'
        );

        text(
          '迁移流的强弱不是由单一因素决定',
          86,
          433,
          12,
          '#075985',
          850,
          'left'
        );

        text(
          '推力和拉力增强通常会扩大迁移流；距离、经济、制度和信息障碍会抑制迁移。',
          86,
          458,
          10.5,
          '#475569',
          650,
          'left'
        );

        text(
          derived.network+'。',
          86,
          482,
          10.5,
          '#475569',
          650,
          'left'
        );
      }

      function impactColumn(
        x,
        titleValue,
        color,
        items
      ){
        box(
          x,
          176,
          274,
          322,
          16,
          '#FFFFFF',
          color
        );

        text(
          titleValue,
          x+137,
          203,
          14,
          color,
          860,
          'center'
        );

        items.forEach(
          function(item,index){
            var y =
              240+
              index*58;

            box(
              x+18,
              y,
              238,
              43,
              10,
              index%2===0
                ? '#F8FAFC'
                : '#FFFFFF',
              '#E2E8F0'
            );

            circle(
              x+39,
              y+21,
              10,
              color,
              null
            );

            text(
              index+1,
              x+39,
              y+21,
              9,
              '#FFFFFF',
              850,
              'center'
            );

            text(
              item,
              x+59,
              y+21,
              10,
              '#475569',
              650,
              'left'
            );
          }
        );
      }

      function impactView(
        item,
        value,
        derived
      ){
        background(
          '人口迁移的区域影响',
          '同时观察迁出地、迁入地和迁移者，避免把人口迁移简单理解为单向利弊。'
        );

        card(
          28,
          78,
          210,
          '迁出地',
          item.origin,
          '#DC2626',
          '人口、产业与家庭结构变化'
        );

        card(
          250,
          78,
          210,
          '迁入地',
          item.destination,
          '#16A34A',
          '劳动力与公共服务压力变化'
        );

        card(
          472,
          78,
          210,
          '迁移流强度',
          derived.score,
          item.color,
          '影响范围随流量变化'
        );

        card(
          694,
          78,
          258,
          '社会网络支持',
          Math.round(value.network),
          '#0891B2',
          '影响信息、适应与再迁移'
        );

        impactColumn(
          42,
          '迁出地',
          '#DC2626',
          [
            '缓解部分就业和资源压力',
            '可能出现青壮年劳动力流失',
            '汇款和信息可能回流',
            '家庭照护结构发生变化'
          ]
        );

        impactColumn(
          353,
          '迁移者',
          '#0284C7',
          [
            '获得新的就业或教育机会',
            '承担迁移成本和适应压力',
            '社会网络影响融入速度',
            '身份与家庭关系可能变化'
          ]
        );

        impactColumn(
          664,
          '迁入地',
          '#16A34A',
          [
            '补充劳动力和消费需求',
            '促进文化与技能交流',
            '住房交通和服务压力上升',
            '需要更包容的公共治理'
          ]
        );
      }

      function typeCard(
        x,
        y,
        w,
        h,
        titleValue,
        subtitle,
        color,
        active
      ){
        box(
          x,
          y,
          w,
          h,
          14,
          active
            ? '#FFFFFF'
            : 'rgba(248,250,252,0.88)',
          active
            ? color
            : '#CBD5E1'
        );

        text(
          titleValue,
          x+16,
          y+23,
          12,
          active
            ? color
            : '#64748B',
          850,
          'left'
        );

        text(
          subtitle,
          x+16,
          y+49,
          9.5,
          '#64748B',
          620,
          'left'
        );

        if(active){
          box(
            x+w-54,
            y+14,
            38,
            22,
            11,
            color,
            null
          );

          text(
            '当前',
            x+w-35,
            y+25,
            9,
            '#FFFFFF',
            820,
            'center'
          );
        }
      }

      function typesView(
        item,
        value,
        derived
      ){
        background(
          '人口迁移类型比较',
          '按空间范围、时间周期和主要动力区分不同迁移类型。'
        );

        card(
          28,
          78,
          210,
          '当前情境',
          item.name,
          item.color,
          '用于课堂对比'
        );

        card(
          250,
          78,
          210,
          '空间跨度',
          item.key==='international'
            ? '跨国'
            : '国内或区域内',
          '#7C3AED',
          '不对应真实边界'
        );

        card(
          472,
          78,
          210,
          '时间特征',
          item.key==='seasonal'
            ? '周期性'
            : '长期或阶段性',
          '#0891B2',
          '迁移可能发生回流'
        );

        card(
          694,
          78,
          258,
          '迁移动力',
          item.key==='environmental'
            ? '环境推力突出'
            : '经济社会因素为主',
          '#EA580C',
          '多种因素常共同作用'
        );

        typeCard(
          50,
          190,
          420,
          90,
          '乡村—城市迁移',
          '城市化过程中常见的就业、教育和服务机会驱动。',
          '#0284C7',
          item.key==='rural-urban'
        );

        typeCard(
          510,
          190,
          420,
          90,
          '区域间产业迁移',
          '产业布局变化带来劳动力和家庭迁移。',
          '#7C3AED',
          item.key==='regional-industry'
        );

        typeCard(
          50,
          304,
          420,
          90,
          '季节性迁移',
          '农业、旅游、施工等季节性就业形成周期流动。',
          '#16A34A',
          item.key==='seasonal'
        );

        typeCard(
          510,
          304,
          420,
          90,
          '国际迁移',
          '跨国迁移通常面临更高距离、制度和文化成本。',
          '#EA580C',
          item.key==='international'
        );

        typeCard(
          280,
          418,
          420,
          90,
          '环境压力迁移',
          '灾害、干旱、海岸侵蚀等压力可能提高迁出推力。',
          '#DC2626',
          item.key==='environmental'
        );
      }

      function update(
        item,
        value,
        derived
      ){
        pushValue.textContent =
          String(
            Math.round(value.push)
          );

        pullValue.textContent =
          String(
            Math.round(value.pull)
          );

        costValue.textContent =
          String(
            Math.round(value.cost)
          );

        networkValue.textContent =
          String(
            Math.round(value.network)
          );

        Array.prototype.forEach.call(
          scenarioButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-scenario'
              )===state.scenario
                ? 'true'
                : 'false'
            );
          }
        );

        Array.prototype.forEach.call(
          viewButtons,
          function(button){
            button.setAttribute(
              'data-active',
              button.getAttribute(
                'data-view'
              )===state.view
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
          item.name+
          '情境下，迁移倾向指数约为'+
          derived.score+
          '，'+
          derived.direction+
          '。'+
          derived.barrier+
          '；'+
          derived.network+
          '。人口迁移对迁出地和迁入地通常同时产生积极与消极影响。';
      }

      function render(){
        if(!root.isConnected){
          state.auto=false;
          return;
        }

        var item =
          scenarioByKey(
            state.scenario
          );

        var value =
          values();

        var derived =
          derive(value);

        update(
          item,
          value,
          derived
        );

        context.clearRect(
          0,
          0,
          width,
          height
        );

        if(state.view==='balance'){
          balanceView(
            item,
            value,
            derived
          );
        }else if(state.view==='flow'){
          flowView(
            item,
            value,
            derived
          );
        }else if(state.view==='impact'){
          impactView(
            item,
            value,
            derived
          );
        }else{
          typesView(
            item,
            value,
            derived
          );
        }
      }

      function applyScenario(
        key,
        changeView
      ){
        var item =
          scenarioByKey(key);

        state.scenario =
          item.key;

        setInputs(item);

        if(changeView){
          state.view =
            item.view;
        }

        state.compareIndex =
          scenarios.indexOf(item);

        render();
      }

      function stopAuto(){
        state.auto=false;
        state.startedAt=0;

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

        var duration =
          5100;

        var segment =
          Math.floor(
            elapsed/duration
          );

        var local =
          (
            elapsed%duration
          )/
          duration;

        var from =
          scenarios[
            segment%
            scenarios.length
          ];

        var to =
          scenarios[
            (
              segment+1
            )%
            scenarios.length
          ];

        var t =
          ease(
            clamp(
              local/0.82,
              0,
              1
            )
          );

        state.scenario =
          local<0.5
            ? from.key
            : to.key;

        state.view =
          [
            'balance',
            'flow',
            'impact',
            'types'
          ][
            Math.floor(
              elapsed/6200
            )%4
          ];

        state.phase =
          (
            elapsed/3200
          )%1;

        setInputs({
          push:lerp(
            from.push,
            to.push,
            t
          ),
          pull:lerp(
            from.pull,
            to.pull,
            t
          ),
          cost:lerp(
            from.cost,
            to.cost,
            t
          ),
          network:lerp(
            from.network,
            to.network,
            t
          )
        });

        render();

        state.raf =
          requestAnimationFrame(
            animate
          );
      }

      function manual(){
        if(state.auto){
          stopAuto();
        }

        state.scenario =
          'custom';

        render();
      }

      [
        pushInput,
        pullInput,
        costInput,
        networkInput
      ].forEach(
        function(input){
          input.addEventListener(
            'input',
            manual
          );
        }
      );

      Array.prototype.forEach.call(
        scenarioButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto){
                stopAuto();
              }

              applyScenario(
                button.getAttribute(
                  'data-scenario'
                ) || 'rural-urban',
                true
              );
            }
          );
        }
      );

      Array.prototype.forEach.call(
        viewButtons,
        function(button){
          button.addEventListener(
            'click',
            function(){
              if(state.auto){
                stopAuto();
              }

              state.view =
                button.getAttribute(
                  'data-view'
                ) || 'balance';

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

          state.raf =
            requestAnimationFrame(
              animate
            );

          render();
        }
      );

      resetButton.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
          }

          state.scenario =
            initial.scenario;

          state.view =
            'balance';

          state.showLabels =
            initial.showLabels;

          setInputs(initial);

          state.compareIndex =
            scenarios.indexOf(
              scenarioByKey(
                initial.scenario
              )
            );

          render();
        }
      );

      compareButton.addEventListener(
        'click',
        function(){
          if(state.auto){
            stopAuto();
          }

          state.compareIndex =
            (
              state.compareIndex+1
            )%
            scenarios.length;

          var next =
            scenarios[
              state.compareIndex
            ];

          state.scenario =
            next.key;

          state.view =
            next.view;

          setInputs(next);
          render();
        }
      );

      state.compareIndex =
        scenarios.indexOf(
          scenarioByKey(
            initial.scenario
          )
        );

      setInputs(initial);
      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_HUMAN_MIGRATION:
GeographyLabTemplate[] = [
  {
    id: 'geography-population-migration-push-pull-flow',
    group: '👥 人口、聚落与城市发展',
    name: '人口迁移、推拉因素与迁移流',
    emoji: '🧳',
    desc: '调节迁出地推力、迁入地拉力、迁移成本和社会网络支持，观察迁移倾向、迁移流及其区域影响。',
    params: [
      {
        key: 'scenario',
        label: '初始迁移情境',
        type: 'select',
        options: [
          {
            label: '乡村到城市',
            value: 'rural-urban',
          },
          {
            label: '区域产业迁移',
            value: 'regional-industry',
          },
          {
            label: '季节性迁移',
            value: 'seasonal',
          },
          {
            label: '国际迁移',
            value: 'international',
          },
          {
            label: '环境压力迁移',
            value: 'environmental',
          },
        ],
        defaultValue: 'rural-urban',
      },
      {
        key: 'pushIntensity',
        label: '迁出地推力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '代表就业不足、资源压力、环境变化或家庭因素形成的迁出倾向。',
      },
      {
        key: 'pullIntensity',
        label: '迁入地拉力',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 8,
        hint: '代表就业、收入、教育、公共服务和生活环境形成的吸引力。',
      },
      {
        key: 'distanceCost',
        label: '距离与迁移成本',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 4,
        hint: '距离、交通、经济、制度和信息障碍通常会抑制迁移。',
      },
      {
        key: 'networkSupport',
        label: '社会网络支持',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '亲友和同乡网络可能降低信息获取、就业寻找和适应成本。',
      },
      {
        key: 'showLabels',
        label: '显示因素标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildHumanMigrationHTML,
  },
]
