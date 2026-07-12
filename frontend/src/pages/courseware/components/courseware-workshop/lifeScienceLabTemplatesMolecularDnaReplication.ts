/**
 * lifeScienceLabTemplatesMolecularDnaReplication.ts
 *
 * 平面生命科学实验室：DNA复制。
 *
 * 教学边界：
 * 1. DNA复制通常发生在细胞分裂前的间期；
 * 2. 解旋后，两条母链分别作为模板合成互补的新链；
 * 3. DNA聚合酶沿模板合成新链，新链延伸方向为5′→3′；
 * 4. 复制结果符合半保留复制：每个子代DNA含一条母链和一条新链；
 * 5. 复制起点、速率和核苷酸供应均为教学示意。
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

function replicationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .dr-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#EEF2FF);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .dr-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .dr-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .dr-body{height:calc(100% - 46px);display:grid;grid-template-columns:240px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .dr-controls{padding:13px;overflow:auto;background:#FBFAFF;border-right:1px solid #C4B5FD}'
    + '#' + rootId + ' .dr-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .dr-row{margin-bottom:11px}'
    + '#' + rootId + ' .dr-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .dr-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#7C3AED}'
    + '#' + rootId + ' .dr-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .dr-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .dr-button{height:31px;padding:0 4px;border:1px solid #A78BFA;border-radius:8px;background:#fff;color:#5B21B6;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .dr-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.13)}'
    + '#' + rootId + ' .dr-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .dr-auto.paused{background:#64748B}'
    + '#' + rootId + ' .dr-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .dr-card{padding:7px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .dr-card b{display:block;font-size:16px;color:#6D28D9}'
    + '#' + rootId + ' .dr-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .dr-result{padding:9px 10px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .dr-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--dr-speed,1.5s) linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_DNA_REPLICATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-dna-replication',
    group: '🧬 遗传信息表达',
    name: 'DNA复制',
    emoji: '🧬',
    desc: '观察DNA解旋、碱基互补配对、新链延伸和半保留复制结果',
    params: [
      {
        key: 'originCount',
        label: '复制起点示意数',
        type: 'number',
        min: 1,
        max: 3,
        step: 1,
        defaultValue: 1,
      },
      {
        key: 'forkSpeed',
        label: '复制叉推进速度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'nucleotideSupply',
        label: '游离核苷酸供应',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
    ],

    buildHTML: (params, rootId) => {
      const originCount = num(params, 'originCount', 1)
      const forkSpeed = num(params, 'forkSpeed', 62)
      const nucleotideSupply = num(params, 'nucleotideSupply', 82)

      return `
<div id="${rootId}">
${replicationStyle(rootId)}
  <div class="dr-head">
    <div class="dr-title">🧬 DNA半保留复制</div>
    <div class="dr-note">结构、数量和速率均为教学示意</div>
  </div>

  <div class="dr-body">
    <div class="dr-controls">
      <div class="dr-row">
        <div class="dr-label">
          <span>复制起点示意数</span>
          <span class="dr-value" data-origin-value></span>
        </div>
        <input data-origin type="range" min="1" max="3" step="1" value="${n(originCount)}">
      </div>

      <div class="dr-row">
        <div class="dr-label">
          <span>复制叉推进速度</span>
          <span class="dr-value" data-speed-value></span>
        </div>
        <input data-speed type="range" min="20" max="100" step="1" value="${n(forkSpeed)}">
      </div>

      <div class="dr-row">
        <div class="dr-label">
          <span>游离核苷酸供应</span>
          <span class="dr-value" data-supply-value></span>
        </div>
        <input data-supply type="range" min="20" max="100" step="1" value="${n(nucleotideSupply)}">
      </div>

      <div class="dr-subtitle">选择复制阶段</div>

      <div class="dr-buttons">
        <button type="button" class="dr-button active" data-stage="unwind">1. 解旋</button>
        <button type="button" class="dr-button" data-stage="pairing">2. 互补配对</button>
        <button type="button" class="dr-button" data-stage="extension">3. 新链延伸</button>
        <button type="button" class="dr-button" data-stage="completion">4. 复制完成</button>
      </div>

      <button type="button" class="dr-auto" data-auto>自动演示：运行中</button>

      <div class="dr-status">
        <div class="dr-card">
          <b data-completion></b>
          <span>相对完成度</span>
        </div>

        <div class="dr-card">
          <b data-dna-count></b>
          <span>DNA分子示意数</span>
        </div>
      </div>

      <div class="dr-result" data-result></div>
    </div>

    <div class="dr-stage">
      <svg viewBox="0 0 680 414" aria-label="DNA复制互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#4C1D95" flood-opacity=".14"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#5B21B6"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <rect x="28" y="88" width="624" height="10" rx="5" fill="#E2E8F0"/>
        <rect data-progress x="28" y="88" width="0" height="10" rx="5" fill="#8B5CF6"/>

        <g data-origins></g>
        <g data-graphic filter="url(#${rootId}-shadow)"></g>

        <g transform="translate(28 369)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">母链</text>
        </g>

        <g transform="translate(132 369)">
          <circle cx="7" cy="7" r="7" fill="#EC4899"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">新合成链</text>
        </g>

        <g transform="translate(276 369)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">游离核苷酸</text>
        </g>

        <text x="448" y="381" data-stage-note font-size="14" font-weight="900" fill="#6D28D9"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var origin=root.querySelector('[data-origin]');
    var speed=root.querySelector('[data-speed]');
    var supply=root.querySelector('[data-supply]');

    var originValue=root.querySelector('[data-origin-value]');
    var speedValue=root.querySelector('[data-speed-value]');
    var supplyValue=root.querySelector('[data-supply-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var autoButton=root.querySelector('[data-auto]');
    var completion=root.querySelector('[data-completion]');
    var dnaCount=root.querySelector('[data-dna-count]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var progress=root.querySelector('[data-progress]');
    var origins=root.querySelector('[data-origins]');
    var graphic=root.querySelector('[data-graphic]');
    var stageNote=root.querySelector('[data-stage-note]');

    var stages=[
      'unwind',
      'pairing',
      'extension',
      'completion'
    ];

    var information={
      unwind:{
        title:'阶段1：DNA解旋',
        summary:'解旋酶等参与破坏碱基之间的氢键，使两条母链局部分开',
        note:'DNA双链解开后，每条母链都可以作为合成新链的模板。'
      },
      pairing:{
        title:'阶段2：碱基互补配对',
        summary:'游离脱氧核苷酸按照碱基互补配对原则排列到模板链旁',
        note:'腺嘌呤与胸腺嘧啶配对，鸟嘌呤与胞嘧啶配对。'
      },
      extension:{
        title:'阶段3：新链延伸',
        summary:'DNA聚合酶催化相邻核苷酸连接，新链沿5′→3′方向延伸',
        note:'两条新链的合成方式存在差异，但都遵循碱基互补配对原则。'
      },
      completion:{
        title:'阶段4：形成两个子代DNA',
        summary:'复制产生两个结构相同的DNA分子，每个都含一条母链和一条新链',
        note:'这种复制方式称为半保留复制。'
      }
    };

    var stage='unwind';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function pairLine(x,y,index,opacity){
      var colors=['#EF4444','#F59E0B','#10B981','#8B5CF6'];

      return '<line x1="'+(x-34)+'" y1="'+y+'" x2="'+(x+34)+'" y2="'+y
        +'" stroke="'+colors[index%colors.length]
        +'" stroke-width="5" stroke-linecap="round" opacity="'+opacity+'"/>';
    }

    function doubleStrand(cx,startY,height,oldLeft,oldRight){
      var left=[];
      var right=[];
      var pairs='';
      var count=18;

      for(var i=0;i<count;i++){
        var t=i/(count-1);
        var y=startY+t*height;
        var wave=Math.sin(t*Math.PI*4);
        var lx=cx-38*wave;
        var rx=cx+38*wave;

        left.push(lx+','+y);
        right.push(rx+','+y);

        if(i%2===0){
          pairs+=pairLine(cx,y,i,.78);
        }
      }

      return pairs
        +'<polyline points="'+left.join(' ')
        +'" fill="none" stroke="'+oldLeft
        +'" stroke-width="7" stroke-linecap="round"/>'
        +'<polyline points="'+right.join(' ')
        +'" fill="none" stroke="'+oldRight
        +'" stroke-width="7" stroke-linecap="round"/>';
    }

    function templateStrands(openAmount){
      var left=[];
      var right=[];
      var pairs='';
      var count=20;

      for(var i=0;i<count;i++){
        var t=i/(count-1);
        var y=118+t*205;
        var separation=24+openAmount*Math.sin(t*Math.PI);
        var lx=340-separation;
        var rx=340+separation;

        left.push(lx+','+y);
        right.push(rx+','+y);

        if(i<4 || i>15){
          pairs+=pairLine(340,y,i,.7);
        }
      }

      return pairs
        +'<polyline points="'+left.join(' ')
        +'" fill="none" stroke="#2563EB" stroke-width="7" stroke-linecap="round"/>'
        +'<polyline points="'+right.join(' ')
        +'" fill="none" stroke="#2563EB" stroke-width="7" stroke-linecap="round"/>';
    }

    function freeNucleotides(count,opacity){
      var html='';
      var labels=['A','T','G','C'];
      var colors=['#EF4444','#F59E0B','#10B981','#8B5CF6'];

      for(var i=0;i<count;i++){
        var x=100+(i%6)*88;
        var y=136+Math.floor(i/6)*64;

        html+='<circle cx="'+x+'" cy="'+y+'" r="14'
          +'" fill="'+colors[i%4]+'" opacity="'+opacity+'"/>';

        html+='<text x="'+x+'" y="'+(y+5)
          +'" text-anchor="middle" font-size="12" font-weight="900" fill="#FFFFFF">'
          +labels[i%4]+'</text>';
      }

      return html;
    }

    function renderUnwind(originCount){
      var html=templateStrands(52);

      html+='<circle cx="340" cy="219" r="34'
        +'" fill="#FDE68A" stroke="#D97706" stroke-width="5"/>';

      html+='<text x="340" y="225" text-anchor="middle" font-size="13" font-weight="900" fill="#92400E">'
        +'解旋酶</text>';

      html+='<path class="dr-flow" d="M340 185 V126'
        +'" fill="none" stroke="#7C3AED" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<path class="dr-flow" d="M340 253 V312'
        +'" fill="none" stroke="#7C3AED" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      return html;
    }

    function renderPairing(supplyLevel){
      var html=templateStrands(76);
      var count=Math.floor(5+supplyLevel/8);

      html+=freeNucleotides(count,.35+.65*supplyLevel/100);

      for(var i=0;i<7;i++){
        var y=154+i*25;

        html+='<circle cx="'+(285-i%2*7)+'" cy="'+y
          +'" r="8" fill="'+(i%2===0?'#F59E0B':'#10B981')
          +'" opacity=".86"/>';

        html+='<circle cx="'+(395+i%2*7)+'" cy="'+y
          +'" r="8" fill="'+(i%2===0?'#EF4444':'#8B5CF6')
          +'" opacity=".86"/>';
      }

      return html;
    }

    function renderExtension(supplyLevel){
      var html=templateStrands(66);
      var length=70+135*supplyLevel/100;

      html+='<path d="M298 126 C276 176 278 228 298 '+(126+length)
        +'" fill="none" stroke="#EC4899" stroke-width="7" stroke-linecap="round"/>';

      html+='<path d="M382 126 C404 176 402 228 382 '+(126+length)
        +'" fill="none" stroke="#EC4899" stroke-width="7" stroke-linecap="round"/>';

      html+='<circle cx="298" cy="'+(126+length)+'" r="22'
        +'" fill="#DDD6FE" stroke="#7C3AED" stroke-width="4"/>';

      html+='<circle cx="382" cy="'+(126+length)+'" r="22'
        +'" fill="#DDD6FE" stroke="#7C3AED" stroke-width="4"/>';

      html+='<text x="340" y="342" text-anchor="middle" font-size="15" font-weight="900" fill="#6D28D9">'
        +'DNA聚合酶催化新链沿5′→3′方向延伸</text>';

      return html;
    }

    function renderCompletion(){
      var html='';

      html+=doubleStrand(
        215,
        118,
        205,
        '#2563EB',
        '#EC4899'
      );

      html+=doubleStrand(
        465,
        118,
        205,
        '#2563EB',
        '#EC4899'
      );

      html+='<text x="215" y="346" text-anchor="middle" font-size="14" font-weight="900" fill="#1D4ED8">'
        +'母链 + 新链</text>';

      html+='<text x="465" y="346" text-anchor="middle" font-size="14" font-weight="900" fill="#1D4ED8">'
        +'母链 + 新链</text>';

      return html;
    }

    function drawOrigins(count){
      var html='';

      for(var i=0;i<count;i++){
        var x=340+(i-(count-1)/2)*120;

        html+='<circle cx="'+x+'" cy="108" r="8'
          +'" fill="#F59E0B" stroke="#B45309" stroke-width="2"/>';

        html+='<text x="'+x+'" y="126" text-anchor="middle" font-size="10" font-weight="900" fill="#92400E">'
          +'起点'+(i+1)+'</text>';
      }

      origins.innerHTML=html;
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(!automatic || !document.body.contains(root)){
        return;
      }

      var forkSpeed=Number(speed.value);
      var interval=clamp(3500-forkSpeed*20,1000,3100);

      timer=window.setTimeout(function(){
        var index=stages.indexOf(stage);
        stage=stages[(index+1)%stages.length];
        update();
        schedule();
      },interval);
    }

    function update(){
      var originCount=Math.round(Number(origin.value));
      var forkSpeed=Number(speed.value);
      var supplyLevel=Number(supply.value);

      originValue.textContent=originCount.toFixed(0)+' 个';
      speedValue.textContent=forkSpeed.toFixed(0)+'%';
      supplyValue.textContent=supplyLevel.toFixed(0)+'%';

      root.style.setProperty(
        '--dr-speed',
        clamp(2.6-forkSpeed/55,.55,2.4).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var index=stages.indexOf(stage);
      var info=information[stage];
      var baseCompletion=[20,45,75,100][index];
      var adjustedCompletion=stage==='completion'
        ?100
        :baseCompletion*(.55+.45*supplyLevel/100);

      title.textContent=info.title;
      summary.textContent=info.summary;

      completion.textContent=adjustedCompletion.toFixed(0)+'%';
      dnaCount.textContent=stage==='completion'?'2':'1';

      progress.setAttribute(
        'width',
        String(624*adjustedCompletion/100)
      );

      drawOrigins(originCount);

      if(stage==='unwind'){
        graphic.innerHTML=renderUnwind(originCount);
        stageNote.textContent='打开复制叉';
      }else if(stage==='pairing'){
        graphic.innerHTML=renderPairing(supplyLevel);
        stageNote.textContent='A-T，G-C';
      }else if(stage==='extension'){
        graphic.innerHTML=renderExtension(supplyLevel);
        stageNote.textContent='新链5′→3′延伸';
      }else{
        graphic.innerHTML=renderCompletion();
        stageNote.textContent='半保留复制';
      }

      var condition='当前复制起点、复制叉速度和核苷酸供应相对协调。';

      if(supplyLevel<35){
        condition='游离核苷酸供应较低，新链延伸受到限制。';
      }else if(forkSpeed>88 && supplyLevel<55){
        condition='复制叉推进较快而核苷酸供应相对不足，复制过程可能受限。';
      }else if(originCount>1){
        condition='设置了多个复制起点，可同时形成多个复制区域，提高整体复制效率。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' DNA复制通常发生在细胞分裂前的间期。';
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

    origin.oninput=update;
    supply.oninput=update;

    speed.oninput=function(){
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
