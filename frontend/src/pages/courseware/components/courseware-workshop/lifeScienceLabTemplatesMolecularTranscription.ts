/**
 * lifeScienceLabTemplatesMolecularTranscription.ts
 *
 * 平面生命科学实验室：DNA转录为mRNA。
 *
 * 教学边界：
 * 1. RNA聚合酶识别启动子并使DNA局部解旋；
 * 2. 以DNA的一条链作为模板链合成RNA；
 * 3. RNA新链沿5′→3′方向延伸；
 * 4. 配对关系为模板链A-U、T-A、G-C、C-G；
 * 5. mRNA序列与编码链基本相同，但用U代替T；
 * 6. 本模型为通用简化示意，不展开不同生物的转录后加工差异。
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

function transcriptionStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #A7F3D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .tr-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#D1FAE5,#ECFDF5);border-bottom:1px solid #A7F3D0}'
    + '#' + rootId + ' .tr-title{font-size:15px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .tr-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .tr-body{height:calc(100% - 46px);display:grid;grid-template-columns:240px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .tr-controls{padding:13px;overflow:auto;background:#F8FFFC;border-right:1px solid #A7F3D0}'
    + '#' + rootId + ' .tr-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .tr-row{margin-bottom:11px}'
    + '#' + rootId + ' .tr-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .tr-value{font-weight:800;color:#059669;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#10B981}'
    + '#' + rootId + ' .tr-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#065F46}'
    + '#' + rootId + ' .tr-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .tr-button{height:31px;padding:0 4px;border:1px solid #6EE7B7;border-radius:8px;background:#fff;color:#065F46;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .tr-button.active{border-color:#10B981;background:#D1FAE5;box-shadow:0 3px 9px rgba(16,185,129,.13)}'
    + '#' + rootId + ' .tr-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .tr-auto.paused{background:#64748B}'
    + '#' + rootId + ' .tr-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .tr-card{padding:7px;border:1px solid #A7F3D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .tr-card b{display:block;font-size:16px;color:#047857}'
    + '#' + rootId + ' .tr-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .tr-result{padding:9px 10px;border-radius:10px;background:#D1FAE5;color:#064E3B;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .tr-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--tr-speed,1.5s) linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_TRANSCRIPTION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-dna-transcription',
    group: '🧬 遗传信息表达',
    name: '转录：DNA到mRNA',
    emoji: '📝',
    desc: '观察启动、局部解旋、互补配对、新链延伸和mRNA释放',
    params: [
      {
        key: 'promoterAffinity',
        label: '启动子识别程度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'polymeraseActivity',
        label: 'RNA聚合酶活性',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'nucleotideSupply',
        label: 'RNA核苷酸供应',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
    ],

    buildHTML: (params, rootId) => {
      const promoterAffinity = num(params, 'promoterAffinity', 78)
      const polymeraseActivity = num(params, 'polymeraseActivity', 72)
      const nucleotideSupply = num(params, 'nucleotideSupply', 82)

      return `
<div id="${rootId}">
${transcriptionStyle(rootId)}
  <div class="tr-head">
    <div class="tr-title">📝 转录：从DNA到mRNA</div>
    <div class="tr-note">以DNA模板链为模板，RNA沿5′→3′方向延伸</div>
  </div>

  <div class="tr-body">
    <div class="tr-controls">
      <div class="tr-row">
        <div class="tr-label">
          <span>启动子识别程度</span>
          <span class="tr-value" data-promoter-value></span>
        </div>
        <input data-promoter type="range" min="20" max="100" step="1" value="${n(promoterAffinity)}">
      </div>

      <div class="tr-row">
        <div class="tr-label">
          <span>RNA聚合酶活性</span>
          <span class="tr-value" data-polymerase-value></span>
        </div>
        <input data-polymerase type="range" min="20" max="100" step="1" value="${n(polymeraseActivity)}">
      </div>

      <div class="tr-row">
        <div class="tr-label">
          <span>RNA核苷酸供应</span>
          <span class="tr-value" data-supply-value></span>
        </div>
        <input data-supply type="range" min="20" max="100" step="1" value="${n(nucleotideSupply)}">
      </div>

      <div class="tr-subtitle">选择转录阶段</div>

      <div class="tr-buttons">
        <button type="button" class="tr-button active" data-stage="initiation">1. 转录起始</button>
        <button type="button" class="tr-button" data-stage="unwind">2. 局部解旋</button>
        <button type="button" class="tr-button" data-stage="elongation">3. RNA延伸</button>
        <button type="button" class="tr-button" data-stage="termination">4. 转录终止</button>
      </div>

      <button type="button" class="tr-auto" data-auto>自动演示：运行中</button>

      <div class="tr-status">
        <div class="tr-card">
          <b data-completion></b>
          <span>转录完成度</span>
        </div>

        <div class="tr-card">
          <b data-rna-length></b>
          <span>mRNA示意长度</span>
        </div>
      </div>

      <div class="tr-result" data-result></div>
    </div>

    <div class="tr-stage">
      <svg viewBox="0 0 680 414" aria-label="DNA转录为mRNA互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#059669"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#064E3B" flood-opacity=".14"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="39" data-title font-size="27" font-weight="900" fill="#065F46"></text>
        <text x="28" y="69" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <rect x="28" y="88" width="624" height="10" rx="5" fill="#E2E8F0"/>
        <rect data-progress x="28" y="88" width="0" height="10" rx="5" fill="#10B981"/>

        <g data-graphic filter="url(#${rootId}-shadow)"></g>

        <g transform="translate(28 370)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">DNA模板链</text>
        </g>

        <g transform="translate(176 370)">
          <circle cx="7" cy="7" r="7" fill="#94A3B8"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">DNA编码链</text>
        </g>

        <g transform="translate(328 370)">
          <circle cx="7" cy="7" r="7" fill="#EC4899"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">mRNA</text>
        </g>

        <text x="472" y="382" data-stage-note font-size="14" font-weight="900" fill="#047857"></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var promoter=root.querySelector('[data-promoter]');
    var polymerase=root.querySelector('[data-polymerase]');
    var supply=root.querySelector('[data-supply]');

    var promoterValue=root.querySelector('[data-promoter-value]');
    var polymeraseValue=root.querySelector('[data-polymerase-value]');
    var supplyValue=root.querySelector('[data-supply-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var autoButton=root.querySelector('[data-auto]');
    var completion=root.querySelector('[data-completion]');
    var rnaLength=root.querySelector('[data-rna-length]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var progress=root.querySelector('[data-progress]');
    var graphic=root.querySelector('[data-graphic]');
    var stageNote=root.querySelector('[data-stage-note]');

    var stages=[
      'initiation',
      'unwind',
      'elongation',
      'termination'
    ];

    var information={
      initiation:{
        title:'阶段1：RNA聚合酶识别启动子',
        summary:'RNA聚合酶结合到基因起始区域附近，准备开始转录',
        note:'启动子是RNA聚合酶等识别和结合的重要DNA区域。'
      },
      unwind:{
        title:'阶段2：DNA局部解旋',
        summary:'RNA聚合酶使基因局部DNA双链分开，暴露模板链',
        note:'转录只需使正在转录的局部区域解开，不需要整条DNA完全解旋。'
      },
      elongation:{
        title:'阶段3：mRNA链延伸',
        summary:'RNA核苷酸按照模板链碱基序列互补配对并连接',
        note:'RNA聚合酶读取DNA模板链，新合成RNA沿5′→3′方向延伸。'
      },
      termination:{
        title:'阶段4：转录终止并释放RNA',
        summary:'RNA聚合酶到达终止区域，新合成的RNA从DNA模板上释放',
        note:'形成的RNA可继续参与后续加工或翻译等过程，具体情况因生物而异。'
      }
    };

    var templateSequence='TACGGTACCTAG';
    var codingSequence='ATGCCATGGATC';
    var rnaSequence='AUGCCAUGGAUC';

    var stage='initiation';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function baseColor(base){
      var colors={
        A:'#EF4444',
        T:'#F59E0B',
        G:'#10B981',
        C:'#8B5CF6',
        U:'#EC4899'
      };

      return colors[base] || '#64748B';
    }

    function nucleotide(x,y,base,color,opacity){
      return '<g opacity="'+opacity+'">'
        +'<circle cx="'+x+'" cy="'+y+'" r="13" fill="'+color+'" stroke="#FFFFFF" stroke-width="2"/>'
        +'<text x="'+x+'" y="'+(y+5)
        +'" text-anchor="middle" font-size="12" font-weight="900" fill="#FFFFFF">'
        +base+'</text>'
        +'</g>';
    }

    function drawDNA(openStart,openEnd){
      var html='';
      var startX=82;
      var gap=43;
      var topY=154;
      var bottomY=242;

      html+='<text x="39" y="'+(topY+5)
        +'" font-size="12" font-weight="900" fill="#64748B">编码链</text>';

      html+='<text x="39" y="'+(bottomY+5)
        +'" font-size="12" font-weight="900" fill="#1D4ED8">模板链</text>';

      for(var i=0;i<codingSequence.length;i++){
        var x=startX+i*gap;
        var open=i>=openStart && i<=openEnd;
        var topOffset=open?-20:0;
        var bottomOffset=open?20:0;

        html+=nucleotide(
          x,
          topY+topOffset,
          codingSequence[i],
          '#94A3B8',
          1
        );

        html+=nucleotide(
          x,
          bottomY+bottomOffset,
          templateSequence[i],
          '#2563EB',
          1
        );

        if(!open){
          html+='<line x1="'+x+'" y1="'+(topY+14)
            +'" x2="'+x+'" y2="'+(bottomY-14)
            +'" stroke="'+baseColor(codingSequence[i])
            +'" stroke-width="4" opacity=".66"/>';
        }
      }

      html+='<path d="M68 '+topY+' H598" stroke="#94A3B8" stroke-width="5" opacity=".5"/>';
      html+='<path d="M68 '+bottomY+' H598" stroke="#2563EB" stroke-width="5" opacity=".5"/>';

      html+='<text x="70" y="129" font-size="11" font-weight="900" fill="#64748B">5′</text>';
      html+='<text x="596" y="129" font-size="11" font-weight="900" fill="#64748B">3′</text>';

      html+='<text x="70" y="278" font-size="11" font-weight="900" fill="#1D4ED8">3′</text>';
      html+='<text x="596" y="278" font-size="11" font-weight="900" fill="#1D4ED8">5′</text>';

      return html;
    }

    function polymeraseAt(index,opacity){
      var x=82+index*43;

      return '<g opacity="'+opacity+'">'
        +'<ellipse cx="'+x+'" cy="198" rx="45" ry="37'
        +'" fill="#D1FAE5" stroke="#059669" stroke-width="5"/>'
        +'<text x="'+x+'" y="194" text-anchor="middle" font-size="11" font-weight="900" fill="#047857">RNA</text>'
        +'<text x="'+x+'" y="210" text-anchor="middle" font-size="11" font-weight="900" fill="#047857">聚合酶</text>'
        +'</g>';
    }

    function rnaChain(length,startY){
      var html='';
      var startX=86;
      var gap=43;

      for(var i=0;i<length;i++){
        var x=startX+i*gap;
        var y=startY+Math.sin(i*.8)*10;

        html+=nucleotide(
          x,
          y,
          rnaSequence[i],
          baseColor(rnaSequence[i]),
          .92
        );

        if(i>0){
          var previousX=startX+(i-1)*gap;
          var previousY=startY+Math.sin((i-1)*.8)*10;

          html+='<line x1="'+(previousX+13)+'" y1="'+previousY
            +'" x2="'+(x-13)+'" y2="'+y
            +'" stroke="#EC4899" stroke-width="5"/>';
        }
      }

      if(length>0){
        html+='<text x="'+(startX-18)+'" y="'+(startY+34)
          +'" font-size="11" font-weight="900" fill="#BE185D">5′</text>';

        html+='<text x="'+(startX+(length-1)*gap+16)
          +'" y="'+(startY+34)
          +'" font-size="11" font-weight="900" fill="#BE185D">3′</text>';
      }

      return html;
    }

    function freeRnaNucleotides(count){
      var html='';

      for(var i=0;i<count;i++){
        var x=102+(i%7)*76;
        var y=315+Math.floor(i/7)*32;
        var bases=['A','U','G','C'];
        var base=bases[i%4];

        html+=nucleotide(
          x,
          y,
          base,
          baseColor(base),
          .45
        );
      }

      return html;
    }

    function renderInitiation(match){
      var html=drawDNA(-1,-1);
      var opacity=.28+.72*match/100;

      html+='<rect x="66" y="112" width="112" height="184" rx="18'
        +'" fill="#FEF3C7" stroke="#F59E0B" stroke-width="4" opacity=".55"/>';

      html+='<text x="122" y="106" text-anchor="middle" font-size="13" font-weight="900" fill="#B45309">'
        +'启动子区域</text>';

      html+=polymeraseAt(1,opacity);

      html+='<path class="tr-flow" d="M235 109 H173'
        +'" fill="none" stroke="#059669" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)" opacity="'+opacity+'"/>';

      return html;
    }

    function renderUnwind(){
      var html=drawDNA(1,5);

      html+=polymeraseAt(3,1);

      html+='<path class="tr-flow" d="M167 198 H235'
        +'" fill="none" stroke="#059669" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      return html;
    }

    function renderElongation(activity,supplyLevel){
      var completed=Math.floor(
        clamp(
          3+(activity+supplyLevel)/22,
          3,
          rnaSequence.length
        )
      );

      var openStart=Math.max(0,completed-4);
      var openEnd=Math.min(
        codingSequence.length-1,
        completed+1
      );

      var html=drawDNA(openStart,openEnd);

      html+=polymeraseAt(
        Math.min(completed,codingSequence.length-1),
        1
      );

      html+=rnaChain(completed,315);

      html+=freeRnaNucleotides(
        Math.floor(3+supplyLevel/14)
      );

      html+='<path class="tr-flow" d="M'
        +(82+completed*43)+' 283 V304'
        +'" fill="none" stroke="#EC4899" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      return html;
    }

    function renderTermination(){
      var html=drawDNA(-1,-1);

      html+=rnaChain(rnaSequence.length,318);

      html+=polymeraseAt(11,.28);

      html+='<path class="tr-flow" d="M565 197 C620 206 628 264 590 298'
        +'" fill="none" stroke="#059669" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="505" y="111" font-size="13" font-weight="900" fill="#B45309">'
        +'终止区域</text>';

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

      var activity=Number(polymerase.value);
      var interval=clamp(
        3500-activity*20,
        1000,
        3100
      );

      timer=window.setTimeout(function(){
        var index=stages.indexOf(stage);
        stage=stages[(index+1)%stages.length];
        update();
        schedule();
      },interval);
    }

    function update(){
      var promoterLevel=Number(promoter.value);
      var polymeraseLevel=Number(polymerase.value);
      var supplyLevel=Number(supply.value);

      promoterValue.textContent=promoterLevel.toFixed(0)+'%';
      polymeraseValue.textContent=polymeraseLevel.toFixed(0)+'%';
      supplyValue.textContent=supplyLevel.toFixed(0)+'%';

      root.style.setProperty(
        '--tr-speed',
        clamp(
          2.6-polymeraseLevel/55,
          .55,
          2.4
        ).toFixed(2)+'s'
      );

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var index=stages.indexOf(stage);
      var info=information[stage];
      var baseCompletion=[18,38,78,100][index];
      var conditionFactor=
        promoterLevel/100
        *polymeraseLevel/100
        *supplyLevel/100;

      var adjustedCompletion=stage==='termination'
        ?100
        :baseCompletion*(.58+.42*conditionFactor);

      var length=stage==='initiation'
        ?0
        :stage==='unwind'
          ?1
          :stage==='elongation'
            ?Math.floor(
              clamp(
                3+(polymeraseLevel+supplyLevel)/22,
                3,
                rnaSequence.length
              )
            )
            :rnaSequence.length;

      completion.textContent=adjustedCompletion.toFixed(0)+'%';
      rnaLength.textContent=length.toFixed(0)+' nt';

      progress.setAttribute(
        'width',
        String(624*adjustedCompletion/100)
      );

      title.textContent=info.title;
      summary.textContent=info.summary;

      if(stage==='initiation'){
        graphic.innerHTML=renderInitiation(promoterLevel);
        stageNote.textContent='识别启动子';
      }else if(stage==='unwind'){
        graphic.innerHTML=renderUnwind();
        stageNote.textContent='暴露模板链';
      }else if(stage==='elongation'){
        graphic.innerHTML=renderElongation(
          polymeraseLevel,
          supplyLevel
        );
        stageNote.textContent='RNA 5′→3′延伸';
      }else{
        graphic.innerHTML=renderTermination();
        stageNote.textContent='释放mRNA';
      }

      var condition='当前启动子识别、聚合酶活性和核苷酸供应相对协调。';

      if(promoterLevel<35){
        condition='启动子识别程度较低，RNA聚合酶结合和转录起始受到限制。';
      }else if(polymeraseLevel<35){
        condition='RNA聚合酶活性较低，mRNA链延伸速度受到限制。';
      }else if(supplyLevel<35){
        condition='RNA核苷酸供应较少，mRNA链合成受到限制。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' mRNA序列与DNA编码链基本相同，但RNA中用U代替T。';
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

    promoter.oninput=update;
    supply.oninput=update;

    polymerase.oninput=function(){
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
