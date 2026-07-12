/**
 * lifeScienceLabTemplatesHumanCirculation.ts
 *
 * 平面生命科学实验室：血液循环互动模型。
 *
 * 教学边界：
 * 1. 用于理解体循环、肺循环和心脏泵血之间的关系；
 * 2. 心输出量、血流速度和压力均为相对教学指标；
 * 3. 不用于医学诊断，不把示意数值解释为人体真实测量结果。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

function num(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: number,
): number {
  const value = params[key]
  return typeof value === 'number' && Number.isFinite(value)
    ? value
    : fallback
}

function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

function circulationStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #FBCFE8;border-radius:16px;background:#FFFFFF;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' *{box-sizing:border-box;}\n'
    + '#' + rootId + ' .hc-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#FCE7F3,#FFF1F2);border-bottom:1px solid #FBCFE8;}\n'
    + '#' + rootId + ' .hc-title{font-size:15px;font-weight:800;color:#9F1239;}\n'
    + '#' + rootId + ' .hc-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .hc-body{height:calc(100% - 46px);display:grid;grid-template-columns:220px minmax(0,1fr);min-height:0;}\n'
    + '#' + rootId + ' .hc-controls{padding:14px;border-right:1px solid #FBCFE8;background:#FFF8FA;overflow:auto;}\n'
    + '#' + rootId + ' .hc-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .hc-row{margin-bottom:13px;}\n'
    + '#' + rootId + ' .hc-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:6px;font-size:12px;font-weight:700;color:#334155;}\n'
    + '#' + rootId + ' .hc-value{font-weight:800;color:#BE123C;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#E11D48;}\n'
    + '#' + rootId + ' .hc-result{padding:9px 11px;border-radius:10px;background:#FFE4E6;color:#881337;font-size:12px;line-height:1.55;font-weight:600;}\n'
    + '#' + rootId + ' .hc-legend{display:grid;grid-template-columns:1fr 1fr;gap:7px;margin:11px 0;}\n'
    + '#' + rootId + ' .hc-key{display:flex;align-items:center;gap:6px;font-size:11px;color:#475569;}\n'
    + '#' + rootId + ' .hc-dot{width:10px;height:10px;border-radius:50%;flex:none;}\n'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%;}\n'
    + '#' + rootId + ' .hc-flow{stroke-dasharray:9 9;animation:' + rootId + '-flow var(--hc-flow-duration,1.8s) linear infinite;}\n'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-36;}}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_CIRCULATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-blood-circulation',
    group: '🫀 人体生命活动',
    name: '血液循环',
    emoji: '🫀',
    desc: '调节心率、每搏输出和血管阻力，观察体循环、肺循环及相对血流变化',
    params: [
      {
        key: 'heartRate',
        label: '心率/次·分⁻¹',
        type: 'number',
        min: 40,
        max: 160,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'strokeVolume',
        label: '每搏输出相对值',
        type: 'number',
        min: 30,
        max: 100,
        step: 1,
        defaultValue: 70,
      },
      {
        key: 'resistance',
        label: '血管阻力相对值',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 48,
      },
    ],
    buildHTML: (params, rootId) => {
      const heartRate = num(params, 'heartRate', 72)
      const strokeVolume = num(params, 'strokeVolume', 70)
      const resistance = num(params, 'resistance', 48)

      return `
<div id="${rootId}">
${circulationStyle(rootId)}
  <div class="hc-head">
    <div class="hc-title">🫀 血液循环与心脏泵血</div>
    <div class="hc-note">相对教学模型，不代表临床测量值</div>
  </div>

  <div class="hc-body">
    <div class="hc-controls">
      <div class="hc-row">
        <div class="hc-label">
          <span>心率</span>
          <span class="hc-value" data-heart-rate-value></span>
        </div>
        <input
          data-heart-rate
          type="range"
          min="40"
          max="160"
          step="1"
          value="${n(heartRate)}"
        />
      </div>

      <div class="hc-row">
        <div class="hc-label">
          <span>每搏输出</span>
          <span class="hc-value" data-stroke-value></span>
        </div>
        <input
          data-stroke
          type="range"
          min="30"
          max="100"
          step="1"
          value="${n(strokeVolume)}"
        />
      </div>

      <div class="hc-row">
        <div class="hc-label">
          <span>血管阻力</span>
          <span class="hc-value" data-resistance-value></span>
        </div>
        <input
          data-resistance
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(resistance)}"
        />
      </div>

      <div class="hc-legend">
        <div class="hc-key">
          <span class="hc-dot" style="background:#DC2626"></span>
          <span>含氧较多</span>
        </div>
        <div class="hc-key">
          <span class="hc-dot" style="background:#2563EB"></span>
          <span>含氧较少</span>
        </div>
      </div>

      <div class="hc-result" data-result></div>
    </div>

    <div class="hc-stage">
      <svg viewBox="0 0 680 414" aria-label="血液循环互动示意图">
        <defs>
          <marker
            id="${rootId}-red-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <marker
            id="${rootId}-blue-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>

          <filter id="${rootId}-heart-shadow">
            <feDropShadow
              dx="0"
              dy="6"
              stdDeviation="7"
              flood-color="#881337"
              flood-opacity="0.18"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="30"
          y="40"
          data-output-text
          font-size="26"
          font-weight="900"
          fill="#9F1239"
        ></text>

        <text
          x="30"
          y="70"
          data-pressure-text
          font-size="16"
          font-weight="800"
          fill="#475569"
        ></text>

        <!-- 人体组织 -->
        <g>
          <rect
            x="42"
            y="126"
            width="128"
            height="162"
            rx="28"
            fill="#F8FAFC"
            stroke="#CBD5E1"
            stroke-width="3"
          />
          <circle cx="106" cy="156" r="22" fill="#FDE68A"/>
          <path
            d="M76 208 C88 184 124 184 136 208 L148 266 H64Z"
            fill="#FCA5A5"
          />
          <text
            x="76"
            y="316"
            font-size="17"
            font-weight="900"
            fill="#334155"
          >全身组织</text>
          <text
            x="60"
            y="340"
            font-size="13"
            font-weight="700"
            fill="#64748B"
          >交换氧气和营养物质</text>
        </g>

        <!-- 肺 -->
        <g>
          <rect
            x="510"
            y="126"
            width="128"
            height="162"
            rx="28"
            fill="#F0F9FF"
            stroke="#BAE6FD"
            stroke-width="3"
          />
          <path
            d="M566 160 C536 164 528 198 536 236 C542 264 562 270 579 246 L579 170Z"
            fill="#FDA4AF"
            stroke="#BE123C"
            stroke-width="3"
          />
          <path
            d="M582 170 L582 246 C598 270 620 264 626 236 C634 198 624 164 594 160Z"
            fill="#FDA4AF"
            stroke="#BE123C"
            stroke-width="3"
          />
          <path
            d="M580 142 V218"
            stroke="#94A3B8"
            stroke-width="8"
            stroke-linecap="round"
          />
          <text
            x="552"
            y="316"
            font-size="17"
            font-weight="900"
            fill="#334155"
          >肺</text>
          <text
            x="526"
            y="340"
            font-size="13"
            font-weight="700"
            fill="#64748B"
          >进行气体交换</text>
        </g>

        <!-- 心脏 -->
        <g
          data-heart
          filter="url(#${rootId}-heart-shadow)"
          transform="translate(0 0)"
        >
          <path
            d="M340 118
               C294 72 230 104 242 166
               C250 214 292 250 340 292
               C388 250 430 214 438 166
               C450 104 386 72 340 118Z"
            fill="#FFE4E6"
            stroke="#BE123C"
            stroke-width="6"
          />

          <path
            d="M338 118 V276"
            stroke="#FFFFFF"
            stroke-width="7"
          />

          <path
            d="M264 158 Q300 140 336 160 V198 H268Z"
            fill="#60A5FA"
            stroke="#1D4ED8"
            stroke-width="3"
          />
          <path
            d="M268 202 H336 V270 Q294 238 268 202Z"
            fill="#2563EB"
            stroke="#1D4ED8"
            stroke-width="3"
          />

          <path
            d="M344 160 Q382 140 416 158 L410 198 H344Z"
            fill="#F87171"
            stroke="#B91C1C"
            stroke-width="3"
          />
          <path
            d="M344 202 H410 Q386 240 344 272Z"
            fill="#DC2626"
            stroke="#B91C1C"
            stroke-width="3"
          />

          <text x="276" y="181" font-size="13" font-weight="900" fill="#EFF6FF">右心房</text>
          <text x="276" y="230" font-size="13" font-weight="900" fill="#EFF6FF">右心室</text>
          <text x="356" y="181" font-size="13" font-weight="900" fill="#FFF1F2">左心房</text>
          <text x="356" y="230" font-size="13" font-weight="900" fill="#FFF1F2">左心室</text>
        </g>

        <!-- 体循环 -->
        <path
          data-systemic-red
          class="hc-flow"
          d="M392 238
             C456 286 458 382 340 382
             C214 382 150 346 124 286"
          fill="none"
          stroke="#DC2626"
          stroke-width="10"
          stroke-linecap="round"
          marker-end="url(#${rootId}-red-arrow)"
        />

        <path
          data-systemic-blue
          class="hc-flow"
          d="M92 286
             C82 362 198 394 308 350
             C332 340 320 286 292 238"
          fill="none"
          stroke="#2563EB"
          stroke-width="10"
          stroke-linecap="round"
          marker-end="url(#${rootId}-blue-arrow)"
        />

        <!-- 肺循环 -->
        <path
          data-pulmonary-blue
          class="hc-flow"
          d="M292 206
             C266 102 388 54 520 148"
          fill="none"
          stroke="#2563EB"
          stroke-width="10"
          stroke-linecap="round"
          marker-end="url(#${rootId}-blue-arrow)"
        />

        <path
          data-pulmonary-red
          class="hc-flow"
          d="M520 242
             C456 318 392 306 374 206"
          fill="none"
          stroke="#DC2626"
          stroke-width="10"
          stroke-linecap="round"
          marker-end="url(#${rootId}-red-arrow)"
        />

        <text
          x="194"
          y="365"
          font-size="15"
          font-weight="900"
          fill="#9F1239"
        >体循环</text>

        <text
          x="390"
          y="88"
          font-size="15"
          font-weight="900"
          fill="#0369A1"
        >肺循环</text>

        <g data-flow-particles></g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var heartRate=root.querySelector('[data-heart-rate]');
    var stroke=root.querySelector('[data-stroke]');
    var resistance=root.querySelector('[data-resistance]');

    var heartRateValue=root.querySelector('[data-heart-rate-value]');
    var strokeValue=root.querySelector('[data-stroke-value]');
    var resistanceValue=root.querySelector('[data-resistance-value]');

    var outputText=root.querySelector('[data-output-text]');
    var pressureText=root.querySelector('[data-pressure-text]');
    var result=root.querySelector('[data-result]');
    var heart=root.querySelector('[data-heart]');

    var paths=root.querySelectorAll('.hc-flow');
    var particles=root.querySelector('[data-flow-particles]');

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function update(){
      var rate=Number(heartRate.value);
      var strokeAmount=Number(stroke.value);
      var vesselResistance=Number(resistance.value);

      var outputIndex=(rate/72)*(strokeAmount/70)*100;
      var pressureIndex=outputIndex*(0.55+vesselResistance/105);
      var speed=clamp(2.8-outputIndex/75,0.65,2.4);
      var width=clamp(7+outputIndex/32,8,14);
      var beatScale=clamp(1+outputIndex/1800,1.02,1.09);

      heartRateValue.textContent=rate.toFixed(0)+' 次/分';
      strokeValue.textContent=strokeAmount.toFixed(0)+'%';
      resistanceValue.textContent=vesselResistance.toFixed(0)+'%';

      outputText.textContent='相对心输出量 '+outputIndex.toFixed(0);
      pressureText.textContent='相对循环压力 '+pressureIndex.toFixed(0);

      root.style.setProperty('--hc-flow-duration',speed.toFixed(2)+'s');

      for(var i=0;i<paths.length;i++){
        paths[i].setAttribute('stroke-width',String(width));
      }

      heart.style.transformOrigin='340px 205px';
      heart.style.transition='transform 180ms ease';
      heart.style.transform='scale('+beatScale.toFixed(3)+')';

      window.setTimeout(function(){
        heart.style.transform='scale(1)';
      },180);

      var state='';
      if(rate>130){
        state='心率较快，舒张和心脏充盈时间可能缩短。';
      }else if(rate<50){
        state='心率较慢，单位时间泵血次数减少。';
      }else if(vesselResistance>82){
        state='血管阻力较大，心脏推动血液流动需要更高的压力。';
      }else if(outputIndex>135){
        state='心输出量较高，可对应运动时循环需求增加的情境。';
      }else{
        state='心率、每搏输出和血管阻力处于相对适中的组合。';
      }

      var particleHTML='';
      var count=Math.floor(clamp(outputIndex/12,4,14));

      for(var p=0;p<count;p++){
        var x=190+(p%5)*72;
        var y=104+(p%3)*18;

        particleHTML+='<circle cx="'+x+'" cy="'+y+'" r="4" fill="'
          +(p%2===0?'#F87171':'#60A5FA')
          +'" opacity="0.72"/>';
      }

      particles.innerHTML=particleHTML;

      result.innerHTML='左心室将含氧较多的血液泵入体循环，右心室将含氧较少的血液泵入肺循环。'
        +'<br>'+state
        +' 当前指标用于比较变量变化，不代表真实血压或心输出量。';
    }

    heartRate.oninput=update;
    stroke.oninput=update;
    resistance.oninput=update;

    update();

    window.setInterval(function(){
      if(document.body.contains(root))update();
    },Math.max(500,60000/Number(heartRate.value)));
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
