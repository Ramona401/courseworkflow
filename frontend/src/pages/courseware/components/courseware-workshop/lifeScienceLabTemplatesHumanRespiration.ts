/**
 * lifeScienceLabTemplatesHumanRespiration.ts
 *
 * 人体呼吸互动模型。
 *
 * 教学边界：
 * 1. 展示肺通气、胸腔容积变化和肺泡气体交换；
 * 2. 通气量和交换水平均为相对教学指标；
 * 3. 不用于肺功能评价、疾病判断或医学诊断。
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

function respirationStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #BAE6FD;border-radius:16px;background:#fff;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' *{box-sizing:border-box;}\n'
    + '#' + rootId + ' .rp-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#E0F2FE,#F0F9FF);border-bottom:1px solid #BAE6FD;}\n'
    + '#' + rootId + ' .rp-title{font-size:15px;font-weight:800;color:#075985;}\n'
    + '#' + rootId + ' .rp-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .rp-body{height:calc(100% - 46px);display:grid;grid-template-columns:225px minmax(0,1fr);min-height:0;}\n'
    + '#' + rootId + ' .rp-controls{padding:13px;border-right:1px solid #BAE6FD;background:#F8FCFF;overflow:auto;}\n'
    + '#' + rootId + ' .rp-stage{position:relative;min-width:0;min-height:0;background:#fff;}\n'
    + '#' + rootId + ' .rp-row{margin-bottom:12px;}\n'
    + '#' + rootId + ' .rp-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155;}\n'
    + '#' + rootId + ' .rp-value{font-weight:800;color:#0284C7;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0284C7;}\n'
    + '#' + rootId + ' .rp-phase-title{margin:9px 0 7px;font-size:12px;font-weight:800;color:#075985;}\n'
    + '#' + rootId + ' .rp-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:8px;}\n'
    + '#' + rootId + ' .rp-button{height:32px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:11px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .rp-button.active{border-color:#0284C7;background:#E0F2FE;box-shadow:0 3px 9px rgba(2,132,199,.12);}\n'
    + '#' + rootId + ' .rp-auto{width:100%;height:32px;margin-bottom:10px;border:0;border-radius:8px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:11px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .rp-auto.paused{background:#64748B;}\n'
    + '#' + rootId + ' .rp-result{padding:9px 10px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:11.5px;line-height:1.52;font-weight:600;}\n'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%;}\n'
    + '#' + rootId + ' .rp-airflow{stroke-dasharray:8 8;animation:rp-flow var(--rp-duration,1.4s) linear infinite;}\n'
    + '@keyframes rp-flow{to{stroke-dashoffset:-32;}}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_RESPIRATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-human-respiration',
    group: '🫀 人体生命活动',
    name: '人体呼吸',
    emoji: '🫁',
    desc: '调节呼吸频率、呼吸深度和气道通畅度，观察肺通气及肺泡气体交换',
    params: [
      {
        key: 'breathingRate',
        label: '呼吸频率/次·分⁻¹',
        type: 'number',
        min: 8,
        max: 40,
        step: 1,
        defaultValue: 16,
      },
      {
        key: 'breathingDepth',
        label: '呼吸深度相对值',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 65,
      },
      {
        key: 'airwayPatency',
        label: '气道通畅度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 88,
      },
    ],

    buildHTML: (params, rootId) => {
      const breathingRate = num(params, 'breathingRate', 16)
      const breathingDepth = num(params, 'breathingDepth', 65)
      const airwayPatency = num(params, 'airwayPatency', 88)

      return `
<div id="${rootId}">
${respirationStyle(rootId)}
  <div class="rp-head">
    <div class="rp-title">🫁 肺通气与肺泡气体交换</div>
    <div class="rp-note">相对教学模型，不代表真实肺功能检测</div>
  </div>

  <div class="rp-body">
    <div class="rp-controls">
      <div class="rp-row">
        <div class="rp-label">
          <span>呼吸频率</span>
          <span class="rp-value" data-rate-value></span>
        </div>
        <input
          data-rate
          type="range"
          min="8"
          max="40"
          step="1"
          value="${n(breathingRate)}"
        >
      </div>

      <div class="rp-row">
        <div class="rp-label">
          <span>呼吸深度</span>
          <span class="rp-value" data-depth-value></span>
        </div>
        <input
          data-depth
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(breathingDepth)}"
        >
      </div>

      <div class="rp-row">
        <div class="rp-label">
          <span>气道通畅度</span>
          <span class="rp-value" data-airway-value></span>
        </div>
        <input
          data-airway
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(airwayPatency)}"
        >
      </div>

      <div class="rp-phase-title">观察呼吸运动</div>

      <div class="rp-buttons">
        <button
          type="button"
          class="rp-button active"
          data-phase="inhale"
        >吸气</button>

        <button
          type="button"
          class="rp-button"
          data-phase="exhale"
        >呼气</button>
      </div>

      <button
        type="button"
        class="rp-auto"
        data-auto
      >自动演示：运行中</button>

      <div class="rp-result" data-result></div>
    </div>

    <div class="rp-stage">
      <svg viewBox="0 0 680 414" aria-label="人体呼吸互动示意图">
        <defs>
          <marker
            id="${rootId}-blue-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

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

          <filter id="${rootId}-lung-shadow">
            <feDropShadow
              dx="0"
              dy="6"
              stdDeviation="7"
              flood-color="#075985"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="28"
          y="38"
          data-phase-name
          font-size="26"
          font-weight="900"
          fill="#075985"
        ></text>

        <text
          x="28"
          y="68"
          data-phase-note
          font-size="15"
          font-weight="800"
          fill="#475569"
        ></text>

        <text
          x="430"
          y="38"
          data-ventilation
          font-size="23"
          font-weight="900"
          fill="#0284C7"
        ></text>

        <!-- 气道 -->
        <path
          d="M275 64 V135"
          fill="none"
          stroke="#94A3B8"
          stroke-width="24"
          stroke-linecap="round"
        />

        <path
          d="M275 126 C247 145 226 157 205 176"
          fill="none"
          stroke="#94A3B8"
          stroke-width="16"
          stroke-linecap="round"
        />

        <path
          d="M275 126 C303 145 324 157 345 176"
          fill="none"
          stroke="#94A3B8"
          stroke-width="16"
          stroke-linecap="round"
        />

        <path
          data-air-path
          class="rp-airflow"
          d="M275 60 V126"
          fill="none"
          stroke="#0284C7"
          stroke-width="7"
          marker-end="url(#${rootId}-blue-arrow)"
        />

        <!-- 胸廓 -->
        <path
          d="M128 106
             C102 176 106 298 164 348
             C222 389 329 389 387 348
             C445 298 449 176 422 106"
          fill="#F8FAFC"
          stroke="#CBD5E1"
          stroke-width="5"
        />

        <g filter="url(#${rootId}-lung-shadow)">
          <!-- 左肺 -->
          <path
            data-left-lung
            d="M254 150
               C211 126 162 150 151 213
               C141 277 169 330 232 324
               C253 296 259 236 254 150Z"
            fill="#FDA4AF"
            stroke="#BE123C"
            stroke-width="5"
          />

          <!-- 右肺 -->
          <path
            data-right-lung
            d="M296 150
               C339 126 388 150 399 213
               C409 277 381 330 318 324
               C297 296 291 236 296 150Z"
            fill="#FDA4AF"
            stroke="#BE123C"
            stroke-width="5"
          />

          <path
            d="M205 176 C220 201 232 226 236 250"
            fill="none"
            stroke="#FEE2E2"
            stroke-width="9"
            stroke-linecap="round"
          />

          <path
            d="M345 176 C330 201 318 226 314 250"
            fill="none"
            stroke="#FEE2E2"
            stroke-width="9"
            stroke-linecap="round"
          />
        </g>

        <!-- 膈肌 -->
        <path
          data-diaphragm
          d="M150 331 Q275 286 400 331"
          fill="none"
          stroke="#A855F7"
          stroke-width="13"
          stroke-linecap="round"
        />

        <text
          x="236"
          y="369"
          font-size="15"
          font-weight="900"
          fill="#7E22CE"
        >膈肌</text>

        <!-- 肺泡放大图 -->
        <g transform="translate(462 106)">
          <rect
            width="190"
            height="246"
            rx="22"
            fill="#F0F9FF"
            stroke="#BAE6FD"
            stroke-width="3"
          />

          <text
            x="48"
            y="30"
            font-size="16"
            font-weight="900"
            fill="#075985"
          >肺泡气体交换</text>

          <path
            d="M28 116 C54 84 88 86 104 116
               C126 78 165 91 166 126
               C166 160 132 176 106 151
               C82 184 40 170 36 138Z"
            fill="#FDE2E8"
            stroke="#BE123C"
            stroke-width="4"
          />

          <path
            d="M20 188 C70 166 119 218 170 188"
            fill="none"
            stroke="#2563EB"
            stroke-width="12"
            stroke-linecap="round"
          />

          <path
            d="M20 208 C70 186 119 238 170 208"
            fill="none"
            stroke="#DC2626"
            stroke-width="12"
            stroke-linecap="round"
          />

          <g data-o2-particles></g>
          <g data-co2-particles></g>

          <text
            x="18"
            y="235"
            font-size="12"
            font-weight="800"
            fill="#475569"
          >氧气进入血液，二氧化碳进入肺泡</text>
        </g>

        <g data-air-particles></g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var rate=root.querySelector('[data-rate]');
    var depth=root.querySelector('[data-depth]');
    var airway=root.querySelector('[data-airway]');

    var rateValue=root.querySelector('[data-rate-value]');
    var depthValue=root.querySelector('[data-depth-value]');
    var airwayValue=root.querySelector('[data-airway-value]');

    var phaseButtons=root.querySelectorAll('[data-phase]');
    var autoButton=root.querySelector('[data-auto]');
    var result=root.querySelector('[data-result]');

    var phaseName=root.querySelector('[data-phase-name]');
    var phaseNote=root.querySelector('[data-phase-note]');
    var ventilationText=root.querySelector('[data-ventilation]');

    var leftLung=root.querySelector('[data-left-lung]');
    var rightLung=root.querySelector('[data-right-lung]');
    var diaphragm=root.querySelector('[data-diaphragm]');
    var airPath=root.querySelector('[data-air-path]');

    var airParticles=root.querySelector('[data-air-particles]');
    var o2Particles=root.querySelector('[data-o2-particles]');
    var co2Particles=root.querySelector('[data-co2-particles]');

    var phase='inhale';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(!automatic || !document.body.contains(root)){
        return;
      }

      var interval=clamp(
        30000/Number(rate.value),
        750,
        2800
      );

      timer=window.setTimeout(function(){
        phase=phase==='inhale'?'exhale':'inhale';
        update();
        schedule();
      },interval);
    }

    function update(){
      var frequency=Number(rate.value);
      var depthAmount=Number(depth.value);
      var airwayAmount=Number(airway.value);

      var depthFactor=depthAmount/100;
      var airwayFactor=airwayAmount/100;
      var ventilation=frequency*depthFactor*airwayFactor*6.25;
      var exchange=clamp(
        ventilation*(0.55+0.45*airwayFactor),
        0,
        100
      );

      rateValue.textContent=frequency.toFixed(0)+' 次/分';
      depthValue.textContent=depthAmount.toFixed(0)+'%';
      airwayValue.textContent=airwayAmount.toFixed(0)+'%';

      ventilationText.textContent=
        '相对肺通气量 '+ventilation.toFixed(0);

      root.style.setProperty(
        '--rp-duration',
        clamp(2.4-frequency/25,.55,2.1).toFixed(2)+'s'
      );

      for(var i=0;i<phaseButtons.length;i++){
        phaseButtons[i].classList.toggle(
          'active',
          phaseButtons[i].getAttribute('data-phase')===phase
        );
      }

      var inhale=phase==='inhale';
      var expansion=depthAmount/100;

      var leftPath=inhale
        ? 'M254 150 C207 119 150 145 137 211 C126 284 159 344 232 334 C258 301 265 231 254 150Z'
        : 'M254 150 C218 135 177 157 168 214 C160 267 181 311 232 307 C249 282 255 226 254 150Z';

      var rightPath=inhale
        ? 'M296 150 C343 119 400 145 413 211 C424 284 391 344 318 334 C292 301 285 231 296 150Z'
        : 'M296 150 C332 135 373 157 382 214 C390 267 369 311 318 307 C301 282 295 226 296 150Z';

      leftLung.setAttribute('d',leftPath);
      rightLung.setAttribute('d',rightPath);

      leftLung.setAttribute(
        'opacity',
        String(.55+.45*airwayFactor)
      );

      rightLung.setAttribute(
        'opacity',
        String(.55+.45*airwayFactor)
      );

      diaphragm.setAttribute(
        'd',
        inhale
          ? 'M150 '+(325+18*expansion)+' Q275 '
            +(350+16*expansion)+' 400 '+(325+18*expansion)
          : 'M150 331 Q275 278 400 331'
      );

      airPath.setAttribute(
        'marker-end',
        inhale
          ? 'url(#${rootId}-blue-arrow)'
          : ''
      );

      airPath.setAttribute(
        'd',
        inhale
          ? 'M275 60 V126'
          : 'M275 126 V60'
      );

      phaseName.textContent=inhale?'吸气阶段':'呼气阶段';

      phaseNote.textContent=inhale
        ? '膈肌收缩下降，胸腔容积增大，空气进入肺。'
        : '膈肌舒张上升，胸腔容积减小，空气排出肺。';

      var particleHTML='';
      var airCount=Math.floor(
        clamp(2+depthAmount/14,3,10)
      );

      for(var p=0;p<airCount;p++){
        var y=82+p*20;
        var direction=inhale?1:-1;

        particleHTML+='<circle cx="'
          +(268+(p%2)*14)
          +'" cy="'+(y*direction+(inhale?0:240))
          +'" r="5" fill="#7DD3FC" opacity="'
          +(.35+airwayFactor*.6)+'"/>';
      }

      airParticles.innerHTML=particleHTML;

      var o2HTML='';
      var co2HTML='';
      var exchangeCount=Math.floor(exchange/10);

      for(var q=0;q<exchangeCount;q++){
        var ox=46+(q%5)*26;
        var oy=62+Math.floor(q/5)*24;

        o2HTML+='<circle cx="'+ox+'" cy="'+oy
          +'" r="5" fill="#38BDF8"/>'
          +'<path d="M'+ox+' '+(oy+7)
          +' L'+(ox-4)+' 172" stroke="#0284C7" stroke-width="2" opacity=".65"/>';

        co2HTML+='<circle cx="'+(38+(q%5)*28)
          +'" cy="'+(196-Math.floor(q/5)*18)
          +'" r="4" fill="#94A3B8"/>';
      }

      o2Particles.innerHTML=o2HTML;
      co2Particles.innerHTML=co2HTML;

      var condition='当前呼吸频率、深度和气道通畅度相对协调。';

      if(airwayAmount<35){
        condition='气道通畅度较低，空气进出受到明显限制。';
      }else if(depthAmount<30){
        condition='呼吸较浅，每次呼吸进入和排出的空气相对较少。';
      }else if(frequency>30){
        condition='呼吸频率较快，但有效通气还受到呼吸深度和气道状态影响。';
      }else if(frequency<10){
        condition='呼吸频率较低，单位时间内通气次数较少。';
      }

      result.innerHTML=phaseNote.textContent
        +'<br>'+condition
        +' 肺泡与毛细血管之间通过扩散完成气体交换。'
        +' 各项数值仅用于教学比较。';
    }

    for(var i=0;i<phaseButtons.length;i++){
      phaseButtons[i].onclick=function(){
        automatic=false;
        autoButton.textContent='自动演示：已暂停';
        autoButton.classList.add('paused');
        phase=this.getAttribute('data-phase');
        update();
        schedule();
      };
    }

    autoButton.onclick=function(){
      automatic=!automatic;
      autoButton.textContent=automatic
        ?'自动演示：运行中'
        :'自动演示：已暂停';
      autoButton.classList.toggle('paused',!automatic);
      update();
      schedule();
    };

    rate.oninput=function(){
      update();
      schedule();
    };

    depth.oninput=update;
    airway.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
