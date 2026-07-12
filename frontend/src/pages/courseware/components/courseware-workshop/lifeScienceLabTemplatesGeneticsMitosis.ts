/**
 * lifeScienceLabTemplatesGeneticsMitosis.ts
 *
 * 平面生命科学实验室：有丝分裂互动模型。
 *
 * 教学边界：
 * 1. 展示间期、前期、中期、后期、末期和胞质分裂；
 * 2. 说明DNA复制发生在分裂前的间期；
 * 3. 展示姐妹染色单体分离并形成两个子细胞；
 * 4. 染色体数量和结构均为教学示意，不对应特定物种。
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

function mitosisStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #DDD6FE;border-radius:16px;background:#FFFFFF;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' *{box-sizing:border-box;}\n'
    + '#' + rootId + ' .mt-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#EDE9FE,#F5F3FF);border-bottom:1px solid #DDD6FE;}\n'
    + '#' + rootId + ' .mt-title{font-size:15px;font-weight:800;color:#5B21B6;}\n'
    + '#' + rootId + ' .mt-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .mt-body{height:calc(100% - 46px);display:grid;grid-template-columns:238px minmax(0,1fr);min-height:0;}\n'
    + '#' + rootId + ' .mt-controls{padding:13px;border-right:1px solid #DDD6FE;background:#FAF9FF;overflow:auto;}\n'
    + '#' + rootId + ' .mt-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .mt-row{margin-bottom:12px;}\n'
    + '#' + rootId + ' .mt-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155;}\n'
    + '#' + rootId + ' .mt-value{font-weight:800;color:#7C3AED;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#7C3AED;}\n'
    + '#' + rootId + ' .mt-stage-title{margin:8px 0 7px;font-size:12px;font-weight:800;color:#5B21B6;}\n'
    + '#' + rootId + ' .mt-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px;}\n'
    + '#' + rootId + ' .mt-button{height:31px;border:1px solid #C4B5FD;border-radius:8px;background:#FFFFFF;color:#5B21B6;font-size:11px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .mt-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.12);}\n'
    + '#' + rootId + ' .mt-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#FFFFFF;font-size:11px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .mt-auto.paused{background:#64748B;}\n'
    + '#' + rootId + ' .mt-result{padding:9px 10px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:11.5px;line-height:1.52;font-weight:600;}\n'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%;}\n'
    + '#' + rootId + ' .mt-progress{transition:width .25s ease;}\n'
    + '#' + rootId + ' .mt-dynamic *{transition:opacity .22s ease,transform .22s ease;}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MITOSIS:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-mitosis',
    group: '🧬 遗传与细胞分裂',
    name: '有丝分裂',
    emoji: '🧬',
    desc: '切换间期、前期、中期、后期、末期和胞质分裂，观察染色体复制、排列与分离',
    params: [
      {
        key: 'chromosomeCount',
        label: '示意染色体数',
        type: 'number',
        min: 2,
        max: 8,
        step: 1,
        defaultValue: 4,
      },
      {
        key: 'spindleStrength',
        label: '纺锤丝显示强度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'animationSpeed',
        label: '自动演示速度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 60,
      },
    ],

    buildHTML: (params, rootId) => {
      const chromosomeCount = num(params, 'chromosomeCount', 4)
      const spindleStrength = num(params, 'spindleStrength', 78)
      const animationSpeed = num(params, 'animationSpeed', 60)

      return `
<div id="${rootId}">
${mitosisStyle(rootId)}
  <div class="mt-head">
    <div class="mt-title">🧬 有丝分裂过程观察</div>
    <div class="mt-note">染色体数量和结构均为教学示意</div>
  </div>

  <div class="mt-body">
    <div class="mt-controls">
      <div class="mt-row">
        <div class="mt-label">
          <span>示意染色体数</span>
          <span class="mt-value" data-count-value></span>
        </div>
        <input
          data-count
          type="range"
          min="2"
          max="8"
          step="1"
          value="${n(chromosomeCount)}"
        >
      </div>

      <div class="mt-row">
        <div class="mt-label">
          <span>纺锤丝显示强度</span>
          <span class="mt-value" data-spindle-value></span>
        </div>
        <input
          data-spindle
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(spindleStrength)}"
        >
      </div>

      <div class="mt-row">
        <div class="mt-label">
          <span>自动演示速度</span>
          <span class="mt-value" data-speed-value></span>
        </div>
        <input
          data-speed
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(animationSpeed)}"
        >
      </div>

      <div class="mt-stage-title">选择细胞周期阶段</div>

      <div class="mt-buttons">
        <button type="button" class="mt-button active" data-stage="interphase">间期</button>
        <button type="button" class="mt-button" data-stage="prophase">前期</button>
        <button type="button" class="mt-button" data-stage="metaphase">中期</button>
        <button type="button" class="mt-button" data-stage="anaphase">后期</button>
        <button type="button" class="mt-button" data-stage="telophase">末期</button>
        <button type="button" class="mt-button" data-stage="cytokinesis">胞质分裂</button>
      </div>

      <button
        type="button"
        class="mt-auto"
        data-auto
      >自动演示：运行中</button>

      <div class="mt-result" data-result></div>
    </div>

    <div class="mt-stage">
      <svg viewBox="0 0 680 414" aria-label="有丝分裂过程互动示意图">
        <defs>
          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="6"
              stdDeviation="8"
              flood-color="#5B21B6"
              flood-opacity=".13"
            />
          </filter>

          <radialGradient id="${rootId}-cell" cx="38%" cy="30%" r="72%">
            <stop offset="0%" stop-color="#FAF5FF"/>
            <stop offset="100%" stop-color="#EDE9FE"/>
          </radialGradient>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="28"
          y="38"
          data-stage-name
          font-size="27"
          font-weight="900"
          fill="#5B21B6"
        ></text>

        <text
          x="28"
          y="69"
          data-stage-summary
          font-size="15"
          font-weight="800"
          fill="#475569"
        ></text>

        <text
          x="490"
          y="38"
          data-count-note
          font-size="15"
          font-weight="900"
          fill="#7C3AED"
        ></text>

        <rect
          x="28"
          y="88"
          width="624"
          height="10"
          rx="5"
          fill="#E2E8F0"
        />

        <rect
          data-progress
          class="mt-progress"
          x="28"
          y="88"
          width="0"
          height="10"
          rx="5"
          fill="#8B5CF6"
        />

        <g
          class="mt-dynamic"
          data-dynamic
          filter="url(#${rootId}-shadow)"
        ></g>

        <g transform="translate(28 366)">
          <circle cx="7" cy="7" r="7" fill="#7C3AED"/>
          <text x="22" y="12" font-size="13" font-weight="800" fill="#475569">
            复制后的每条染色体由两条姐妹染色单体组成
          </text>
        </g>

        <g transform="translate(380 366)">
          <circle cx="7" cy="7" r="7" fill="#EC4899"/>
          <text x="22" y="12" font-size="13" font-weight="800" fill="#475569">
            分离后进入两个子细胞
          </text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var countInput=root.querySelector('[data-count]');
    var spindleInput=root.querySelector('[data-spindle]');
    var speedInput=root.querySelector('[data-speed]');

    var countValue=root.querySelector('[data-count-value]');
    var spindleValue=root.querySelector('[data-spindle-value]');
    var speedValue=root.querySelector('[data-speed-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var autoButton=root.querySelector('[data-auto]');
    var result=root.querySelector('[data-result]');

    var stageName=root.querySelector('[data-stage-name]');
    var stageSummary=root.querySelector('[data-stage-summary]');
    var countNote=root.querySelector('[data-count-note]');
    var progress=root.querySelector('[data-progress]');
    var dynamic=root.querySelector('[data-dynamic]');

    var order=[
      'interphase',
      'prophase',
      'metaphase',
      'anaphase',
      'telophase',
      'cytokinesis'
    ];

    var information={
      interphase:{
        name:'间期',
        summary:'DNA复制完成，染色质仍较为舒展',
        note:'DNA复制发生在分裂前的间期，为后续平均分配遗传物质作准备。'
      },
      prophase:{
        name:'前期',
        summary:'染色质凝缩成染色体，核膜逐渐解体',
        note:'复制后的染色体逐渐清晰，每条染色体含两条姐妹染色单体。'
      },
      metaphase:{
        name:'中期',
        summary:'染色体排列在细胞中央的赤道板附近',
        note:'纺锤丝连接染色体，使染色体有序排列，为准确分离作准备。'
      },
      anaphase:{
        name:'后期',
        summary:'姐妹染色单体分离并移向细胞两极',
        note:'姐妹染色单体分开后成为子染色体，分别移向相反两极。'
      },
      telophase:{
        name:'末期',
        summary:'两组染色体到达两极，新的细胞核形成',
        note:'染色体逐渐解螺旋，核膜重新形成，细胞内出现两个细胞核。'
      },
      cytokinesis:{
        name:'胞质分裂',
        summary:'细胞质分开，形成两个遗传物质基本相同的子细胞',
        note:'动物细胞通常通过细胞膜内陷完成胞质分裂，本图采用该方式示意。'
      }
    };

    var stage='interphase';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function cellOutline(cx,cy,rx,ry){
      return '<ellipse cx="'+cx+'" cy="'+cy+'" rx="'+rx+'" ry="'+ry
        +'" fill="url(#${rootId}-cell)" stroke="#8B5CF6" stroke-width="5"/>';
    }

    function nucleus(cx,cy,rx,ry,opacity){
      return '<ellipse cx="'+cx+'" cy="'+cy+'" rx="'+rx+'" ry="'+ry
        +'" fill="#F5D0FE" stroke="#A855F7" stroke-width="4" opacity="'+opacity+'"/>';
    }

    function chromosomeX(x,y,size,color,rotation){
      var s=size;
      return '<g transform="translate('+x+' '+y+') rotate('+rotation+')">'
        +'<path d="M'+(-s)+' '+(-s*1.4)+' L'+s+' '+(s*1.4)
        +'" stroke="'+color+'" stroke-width="'+(s*.48)
        +'" stroke-linecap="round"/>'
        +'<path d="M'+s+' '+(-s*1.4)+' L'+(-s)+' '+(s*1.4)
        +'" stroke="'+color+'" stroke-width="'+(s*.48)
        +'" stroke-linecap="round"/>'
        +'<circle cx="0" cy="0" r="'+(s*.32)+'" fill="#FDE68A" stroke="#B45309" stroke-width="1.5"/>'
        +'</g>';
    }

    function chromatidV(x,y,size,color,direction){
      var flip=direction<0?-1:1;

      return '<g transform="translate('+x+' '+y+') scale('+flip+' 1)">'
        +'<path d="M0 0 L'+(size*1.25)+' '+(-size)
        +' M0 0 L'+(size*1.25)+' '+size
        +'" stroke="'+color+'" stroke-width="'+(size*.42)
        +'" stroke-linecap="round"/>'
        +'<circle cx="0" cy="0" r="'+(size*.27)
        +'" fill="#FDE68A" stroke="#B45309" stroke-width="1.5"/>'
        +'</g>';
    }

    function spindleLine(x1,y1,x2,y2,opacity,width){
      return '<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2
        +'" stroke="#0EA5E9" stroke-width="'+width+'" opacity="'+opacity+'"/>';
    }

    function renderInterphase(count){
      var html=cellOutline(340,226,202,120);
      html+=nucleus(340,226,94,76,1);

      for(var i=0;i<count*2;i++){
        var angle=i*2.399;
        var radius=18+(i%4)*13;
        var x=340+Math.cos(angle)*radius;
        var y=226+Math.sin(angle)*radius*.7;

        html+='<path d="M'+(x-12)+' '+(y-5)
          +' Q'+x+' '+(y-18)+' '+(x+12)+' '+(y-2)
          +' Q'+x+' '+(y+14)+' '+(x-10)+' '+(y+8)
          +'" fill="none" stroke="'
          +(i%2===0?'#7C3AED':'#EC4899')
          +'" stroke-width="4" stroke-linecap="round" opacity=".8"/>';
      }

      html+='<circle cx="340" cy="226" r="16" fill="#A855F7" opacity=".65"/>';
      return html;
    }

    function renderProphase(count){
      var html=cellOutline(340,226,202,120);
      html+=nucleus(340,226,96,76,.38);

      for(var i=0;i<count;i++){
        var angle=(Math.PI*2*i/count)+.4;
        var x=340+Math.cos(angle)*58;
        var y=226+Math.sin(angle)*43;

        html+=chromosomeX(
          x,
          y,
          13,
          i%2===0?'#7C3AED':'#EC4899',
          i*29
        );
      }

      html+='<circle cx="161" cy="226" r="11" fill="#0EA5E9"/>'
        +'<circle cx="519" cy="226" r="11" fill="#0EA5E9"/>';

      return html;
    }

    function renderMetaphase(count,opacity,width){
      var html=cellOutline(340,226,202,120);

      html+='<circle cx="151" cy="226" r="12" fill="#0EA5E9"/>'
        +'<circle cx="529" cy="226" r="12" fill="#0EA5E9"/>'
        +'<line x1="340" y1="119" x2="340" y2="333" stroke="#CBD5E1" stroke-width="3" stroke-dasharray="7 7"/>';

      var spacing=Math.min(42,170/Math.max(1,count-1));

      for(var i=0;i<count;i++){
        var y=226-(count-1)*spacing/2+i*spacing;
        var color=i%2===0?'#7C3AED':'#EC4899';

        html+=spindleLine(163,226,326,y,opacity,width);
        html+=spindleLine(517,226,354,y,opacity,width);
        html+=chromosomeX(340,y,12,color,i%2===0?0:18);
      }

      return html;
    }

    function renderAnaphase(count,opacity,width){
      var html=cellOutline(340,226,202,120);

      html+='<circle cx="151" cy="226" r="12" fill="#0EA5E9"/>'
        +'<circle cx="529" cy="226" r="12" fill="#0EA5E9"/>';

      var spacing=Math.min(38,150/Math.max(1,count-1));

      for(var i=0;i<count;i++){
        var y=226-(count-1)*spacing/2+i*spacing;
        var color=i%2===0?'#7C3AED':'#EC4899';
        var leftX=248-(i%2)*10;
        var rightX=432+(i%2)*10;

        html+=spindleLine(163,226,leftX,y,opacity,width);
        html+=spindleLine(517,226,rightX,y,opacity,width);
        html+=chromatidV(leftX,y,11,color,-1);
        html+=chromatidV(rightX,y,11,color,1);
      }

      return html;
    }

    function renderTelophase(count){
      var html=cellOutline(340,226,202,120);
      html+=nucleus(247,226,68,68,1);
      html+=nucleus(433,226,68,68,1);

      for(var i=0;i<count;i++){
        var angle=Math.PI*2*i/count;
        var lx=247+Math.cos(angle)*31;
        var ly=226+Math.sin(angle)*26;
        var rx=433+Math.cos(angle)*31;
        var ry=226+Math.sin(angle)*26;
        var color=i%2===0?'#7C3AED':'#EC4899';

        html+='<path d="M'+(lx-8)+' '+(ly-4)+' Q'+lx+' '+(ly-12)
          +' '+(lx+8)+' '+ly+'" fill="none" stroke="'+color
          +'" stroke-width="3.5" stroke-linecap="round"/>';

        html+='<path d="M'+(rx-8)+' '+(ry-4)+' Q'+rx+' '+(ry-12)
          +' '+(rx+8)+' '+ry+'" fill="none" stroke="'+color
          +'" stroke-width="3.5" stroke-linecap="round"/>';
      }

      html+='<path d="M340 118 Q312 226 340 334" fill="none" stroke="#C4B5FD" stroke-width="5" stroke-dasharray="8 8"/>';
      return html;
    }

    function renderCytokinesis(count){
      var html=cellOutline(232,226,104,101);
      html+=cellOutline(448,226,104,101);
      html+=nucleus(232,226,57,57,1);
      html+=nucleus(448,226,57,57,1);

      for(var i=0;i<count;i++){
        var angle=Math.PI*2*i/count;
        var color=i%2===0?'#7C3AED':'#EC4899';
        var lx=232+Math.cos(angle)*27;
        var ly=226+Math.sin(angle)*23;
        var rx=448+Math.cos(angle)*27;
        var ry=226+Math.sin(angle)*23;

        html+='<circle cx="'+lx+'" cy="'+ly+'" r="5" fill="'+color+'" opacity=".8"/>';
        html+='<circle cx="'+rx+'" cy="'+ry+'" r="5" fill="'+color+'" opacity=".8"/>';
      }

      html+='<text x="194" y="348" font-size="15" font-weight="900" fill="#5B21B6">子细胞1</text>'
        +'<text x="410" y="348" font-size="15" font-weight="900" fill="#5B21B6">子细胞2</text>';

      return html;
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(!automatic || !document.body.contains(root)){
        return;
      }

      var speed=Number(speedInput.value);
      var interval=4200-speed*27;

      timer=window.setTimeout(function(){
        var index=order.indexOf(stage);
        stage=order[(index+1)%order.length];
        update();
        schedule();
      },clamp(interval,1200,3700));
    }

    function update(){
      var count=Math.round(Number(countInput.value));
      var spindle=Number(spindleInput.value);
      var speed=Number(speedInput.value);

      countValue.textContent=count.toFixed(0)+' 条';
      spindleValue.textContent=spindle.toFixed(0)+'%';
      speedValue.textContent=speed.toFixed(0)+'%';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var index=order.indexOf(stage);
      var info=information[stage];
      var opacity=.18+.82*spindle/100;
      var width=1.2+spindle/35;

      stageName.textContent=info.name;
      stageSummary.textContent=info.summary;
      countNote.textContent='示意染色体数：'+count;
      progress.setAttribute(
        'width',
        String(624*(index+1)/order.length)
      );

      if(stage==='interphase'){
        dynamic.innerHTML=renderInterphase(count);
      }else if(stage==='prophase'){
        dynamic.innerHTML=renderProphase(count);
      }else if(stage==='metaphase'){
        dynamic.innerHTML=renderMetaphase(count,opacity,width);
      }else if(stage==='anaphase'){
        dynamic.innerHTML=renderAnaphase(count,opacity,width);
      }else if(stage==='telophase'){
        dynamic.innerHTML=renderTelophase(count);
      }else{
        dynamic.innerHTML=renderCytokinesis(count);
      }

      var chromosomeNote='';

      if(stage==='interphase'){
        chromosomeNote='此时DNA已经完成复制，但染色质尚未高度凝缩。';
      }else if(stage==='anaphase'){
        chromosomeNote='姐妹染色单体分离后分别成为子染色体，并向细胞两极移动。';
      }else if(stage==='cytokinesis'){
        chromosomeNote='两个子细胞通常保持与母细胞相同的染色体数目。';
      }else{
        chromosomeNote='复制后的染色体在纺锤体作用下完成排列和分配。';
      }

      result.innerHTML=info.note
        +'<br>'+chromosomeNote
        +' 本模型忽略基因突变和染色体异常等特殊情况。';
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        automatic=false;
        autoButton.textContent='自动演示：已暂停';
        autoButton.classList.add('paused');
        stage=this.getAttribute('data-stage');
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

    countInput.oninput=update;
    spindleInput.oninput=update;

    speedInput.oninput=function(){
      update();
      schedule();
    };

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
