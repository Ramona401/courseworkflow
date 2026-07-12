/**
 * lifeScienceLabTemplatesMolecularTranslation.ts
 *
 * 平面生命科学实验室：mRNA翻译为蛋白质。
 *
 * 教学边界：
 * 1. 核糖体沿mRNA的5′→3′方向读取密码子；
 * 2. 起始密码子AUG通常决定翻译起始位置；
 * 3. tRNA反密码子与mRNA密码子互补配对；
 * 4. 相邻氨基酸之间形成肽键，多肽链由N端向C端延伸；
 * 5. 遇到终止密码子后，释放因子参与终止，多肽链释放；
 * 6. 密码子、氨基酸数量和翻译速率均为教学示意。
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

function translationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FBCFE8;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .tl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#FFF1F2);border-bottom:1px solid #FBCFE8}'
    + '#' + rootId + ' .tl-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .tl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .tl-body{height:calc(100% - 46px);display:grid;grid-template-columns:240px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .tl-controls{padding:13px;overflow:auto;background:#FFF9FC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .tl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .tl-row{margin-bottom:11px}'
    + '#' + rootId + ' .tl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .tl-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#DB2777}'
    + '#' + rootId + ' .tl-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .tl-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .tl-button{height:31px;padding:0 4px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .tl-button.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.13)}'
    + '#' + rootId + ' .tl-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .tl-auto.paused{background:#64748B}'
    + '#' + rootId + ' .tl-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .tl-card{padding:7px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .tl-card b{display:block;font-size:16px;color:#BE185D}'
    + '#' + rootId + ' .tl-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .tl-result{padding:9px 10px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .tl-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--tl-speed,1.5s) linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_MOLECULAR_TRANSLATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-mrna-translation',
    group: '🧬 遗传信息表达',
    name: '翻译：mRNA到蛋白质',
    emoji: '🧪',
    desc: '观察核糖体装配、密码子识别、肽键形成、核糖体移位和翻译终止',
    params: [
      {
        key: 'codonCount',
        label: '密码子示意数量',
        type: 'number',
        min: 4,
        max: 8,
        step: 1,
        defaultValue: 7,
      },
      {
        key: 'ribosomeActivity',
        label: '核糖体活跃度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 74,
      },
      {
        key: 'trnaSupply',
        label: 'tRNA供应水平',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
    ],

    buildHTML: (params, rootId) => {
      const codonCount = num(params, 'codonCount', 7)
      const ribosomeActivity = num(params, 'ribosomeActivity', 74)
      const trnaSupply = num(params, 'trnaSupply', 82)

      return `
<div id="${rootId}">
${translationStyle(rootId)}
  <div class="tl-head">
    <div class="tl-title">🧪 翻译：从mRNA到蛋白质</div>
    <div class="tl-note">核糖体沿mRNA的5′→3′方向读取密码子</div>
  </div>

  <div class="tl-body">
    <div class="tl-controls">
      <div class="tl-row">
        <div class="tl-label">
          <span>密码子示意数量</span>
          <span class="tl-value" data-codon-value></span>
        </div>
        <input
          data-codon
          type="range"
          min="4"
          max="8"
          step="1"
          value="${n(codonCount)}"
        >
      </div>

      <div class="tl-row">
        <div class="tl-label">
          <span>核糖体活跃度</span>
          <span class="tl-value" data-ribosome-value></span>
        </div>
        <input
          data-ribosome
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(ribosomeActivity)}"
        >
      </div>

      <div class="tl-row">
        <div class="tl-label">
          <span>tRNA供应水平</span>
          <span class="tl-value" data-trna-value></span>
        </div>
        <input
          data-trna
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(trnaSupply)}"
        >
      </div>

      <div class="tl-subtitle">选择翻译阶段</div>

      <div class="tl-buttons">
        <button type="button" class="tl-button active" data-stage="initiation">1. 翻译起始</button>
        <button type="button" class="tl-button" data-stage="recognition">2. 密码子识别</button>
        <button type="button" class="tl-button" data-stage="bond">3. 肽键形成</button>
        <button type="button" class="tl-button" data-stage="translocation">4. 核糖体移位</button>
        <button type="button" class="tl-button" data-stage="termination">5. 翻译终止</button>
      </div>

      <button
        type="button"
        class="tl-auto"
        data-auto
      >自动演示：运行中</button>

      <div class="tl-status">
        <div class="tl-card">
          <b data-completion></b>
          <span>翻译完成度</span>
        </div>

        <div class="tl-card">
          <b data-peptide-length></b>
          <span>多肽链示意长度</span>
        </div>
      </div>

      <div class="tl-result" data-result></div>
    </div>

    <div class="tl-stage">
      <svg viewBox="0 0 680 414" aria-label="mRNA翻译为蛋白质互动示意图">
        <defs>
          <marker
            id="${rootId}-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#DB2777"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#831843"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="28"
          y="39"
          data-title
          font-size="27"
          font-weight="900"
          fill="#9D174D"
        ></text>

        <text
          x="28"
          y="69"
          data-summary
          font-size="15"
          font-weight="800"
          fill="#475569"
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
          x="28"
          y="88"
          width="0"
          height="10"
          rx="5"
          fill="#EC4899"
        />

        <g
          data-graphic
          filter="url(#${rootId}-shadow)"
        ></g>

        <g transform="translate(28 370)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            mRNA密码子
          </text>
        </g>

        <g transform="translate(180 370)">
          <circle cx="7" cy="7" r="7" fill="#10B981"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            tRNA
          </text>
        </g>

        <g transform="translate(290 370)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            氨基酸
          </text>
        </g>

        <g transform="translate(424 370)">
          <circle cx="7" cy="7" r="7" fill="#8B5CF6"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            核糖体
          </text>
        </g>

        <text
          x="536"
          y="382"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#BE185D"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var codonInput=root.querySelector('[data-codon]');
    var ribosomeInput=root.querySelector('[data-ribosome]');
    var trnaInput=root.querySelector('[data-trna]');

    var codonValue=root.querySelector('[data-codon-value]');
    var ribosomeValue=root.querySelector('[data-ribosome-value]');
    var trnaValue=root.querySelector('[data-trna-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var autoButton=root.querySelector('[data-auto]');
    var completion=root.querySelector('[data-completion]');
    var peptideLength=root.querySelector('[data-peptide-length]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var progress=root.querySelector('[data-progress]');
    var graphic=root.querySelector('[data-graphic]');
    var stageNote=root.querySelector('[data-stage-note]');

    var stages=[
      'initiation',
      'recognition',
      'bond',
      'translocation',
      'termination'
    ];

    var information={
      initiation:{
        title:'阶段1：翻译起始',
        summary:'核糖体亚基在mRNA起始密码子附近装配，起始tRNA进入',
        note:'起始密码子AUG通常编码甲硫氨酸，并确定翻译的起始位置。'
      },
      recognition:{
        title:'阶段2：密码子识别',
        summary:'tRNA反密码子与mRNA密码子进行互补配对',
        note:'携带相应氨基酸的tRNA通过反密码子识别mRNA上的密码子。'
      },
      bond:{
        title:'阶段3：形成肽键',
        summary:'核糖体催化相邻氨基酸之间形成肽键',
        note:'新氨基酸连接到正在延伸的多肽链上，多肽链由N端向C端延伸。'
      },
      translocation:{
        title:'阶段4：核糖体移位',
        summary:'核糖体沿mRNA向5′→3′方向移动一个密码子',
        note:'核糖体移位后，空载tRNA离开，新的密码子进入识别位置。'
      },
      termination:{
        title:'阶段5：翻译终止',
        summary:'核糖体遇到终止密码子，释放因子促使多肽链释放',
        note:'终止密码子不编码氨基酸，由释放因子参与识别并终止翻译。'
      }
    };

    var senseCodons=[
      'AUG',
      'GCU',
      'UUU',
      'GGA',
      'AAA',
      'UAC',
      'CCU'
    ];

    var stopCodon='UGA';

    var aminoAcidMap={
      AUG:'甲硫氨酸',
      GCU:'丙氨酸',
      UUU:'苯丙氨酸',
      GGA:'甘氨酸',
      AAA:'赖氨酸',
      UAC:'酪氨酸',
      CCU:'脯氨酸',
      UGA:'终止'
    };

    var shortAminoAcidMap={
      AUG:'Met',
      GCU:'Ala',
      UUU:'Phe',
      GGA:'Gly',
      AAA:'Lys',
      UAC:'Tyr',
      CCU:'Pro',
      UGA:'Stop'
    };

    var anticodonMap={
      AUG:'UAC',
      GCU:'CGA',
      UUU:'AAA',
      GGA:'CCU',
      AAA:'UUU',
      UAC:'AUG',
      CCU:'GGA'
    };

    var stage='initiation';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function sequenceFor(count){
      var sequence=senseCodons.slice(
        0,
        Math.max(1,count-1)
      );

      sequence.push(stopCodon);

      return sequence;
    }

    function codonColor(codon){
      var colors={
        AUG:'#EF4444',
        GCU:'#10B981',
        UUU:'#2563EB',
        GGA:'#8B5CF6',
        AAA:'#F59E0B',
        UAC:'#EC4899',
        CCU:'#06B6D4',
        UGA:'#64748B'
      };

      return colors[codon] || '#64748B';
    }

    function drawMrna(sequence,activeIndex){
      var html='';
      var startX=74;
      var gap=72;
      var y=252;

      html+='<path d="M52 '+y+' H632'
        +'" stroke="#2563EB" stroke-width="8" stroke-linecap="round"/>';

      html+='<text x="38" y="'+(y+5)
        +'" font-size="13" font-weight="900" fill="#1D4ED8">5′</text>';

      html+='<text x="638" y="'+(y+5)
        +'" font-size="13" font-weight="900" fill="#1D4ED8">3′</text>';

      for(var i=0;i<sequence.length;i++){
        var x=startX+i*gap;
        var codon=sequence[i];
        var active=i===activeIndex;

        html+='<rect x="'+(x-27)+'" y="'+(y-24)
          +'" width="58" height="48" rx="10" fill="'
          +(active?'#DBEAFE':'#FFFFFF')
          +'" stroke="'+codonColor(codon)
          +'" stroke-width="'+(active?5:3)+'"/>';

        html+='<text x="'+(x+2)+'" y="'+(y+6)
          +'" text-anchor="middle" font-size="15" font-weight="900" fill="'
          +codonColor(codon)+'">'+codon+'</text>';
      }

      return html;
    }

    function drawRibosome(index,opacity){
      var x=76+index*72;

      return '<g opacity="'+opacity+'">'
        +'<ellipse cx="'+x+'" cy="211" rx="58" ry="40'
        +'" fill="#EDE9FE" stroke="#7C3AED" stroke-width="5"/>'
        +'<ellipse cx="'+x+'" cy="273" rx="46" ry="27'
        +'" fill="#DDD6FE" stroke="#8B5CF6" stroke-width="4"/>'
        +'<text x="'+x+'" y="205" text-anchor="middle" font-size="12" font-weight="900" fill="#6D28D9">核糖体</text>'
        +'<text x="'+x+'" y="222" text-anchor="middle" font-size="10" font-weight="800" fill="#7C3AED">大小亚基</text>'
        +'</g>';
    }

    function drawTrna(x,codon,opacity){
      var anticodon=anticodonMap[codon] || '—';
      var amino=shortAminoAcidMap[codon] || '';

      return '<g opacity="'+opacity+'">'
        +'<path d="M'+x+' 206 V166'
        +' M'+x+' 184 L'+(x-17)+' 170'
        +' M'+x+' 184 L'+(x+17)+' 170'
        +'" fill="none" stroke="#10B981" stroke-width="7" stroke-linecap="round"/>'
        +'<circle cx="'+x+'" cy="148" r="18'
        +'" fill="#FDE68A" stroke="#D97706" stroke-width="4"/>'
        +'<text x="'+x+'" y="153" text-anchor="middle" font-size="10" font-weight="900" fill="#92400E">'
        +amino+'</text>'
        +'<rect x="'+(x-24)+'" y="205" width="48" height="25" rx="9'
        +'" fill="#D1FAE5" stroke="#059669" stroke-width="3"/>'
        +'<text x="'+x+'" y="222" text-anchor="middle" font-size="11" font-weight="900" fill="#047857">'
        +anticodon+'</text>'
        +'</g>';
    }

    function drawPeptide(sequence,length,released){
      var html='';
      var startX=112;
      var y=118;
      var gap=53;

      for(var i=0;i<length;i++){
        var codon=sequence[i];

        if(codon===stopCodon){
          break;
        }

        var x=startX+i*gap;
        var waveY=y+Math.sin(i*.8)*14;

        if(i>0){
          var previousX=startX+(i-1)*gap;
          var previousY=y+Math.sin((i-1)*.8)*14;

          html+='<line x1="'+(previousX+17)+'" y1="'+previousY
            +'" x2="'+(x-17)+'" y2="'+waveY
            +'" stroke="#DB2777" stroke-width="6"/>';
        }

        html+='<circle cx="'+x+'" cy="'+waveY+'" r="18'
          +'" fill="'+codonColor(codon)
          +'" stroke="#FFFFFF" stroke-width="3"/>';

        html+='<text x="'+x+'" y="'+(waveY+4)
          +'" text-anchor="middle" font-size="9" font-weight="900" fill="#FFFFFF">'
          +shortAminoAcidMap[codon]+'</text>';
      }

      if(length>0){
        html+='<text x="82" y="91" font-size="12" font-weight="900" fill="#BE185D">N端</text>';

        html+='<text x="'+(startX+Math.max(0,length-1)*gap+25)
          +'" y="91" font-size="12" font-weight="900" fill="#BE185D">C端</text>';
      }

      if(released){
        html+='<path class="tl-flow" d="M360 151 C410 105 470 108 515 133'
          +'" fill="none" stroke="#DB2777" stroke-width="4'
          +'" marker-end="url(#${rootId}-arrow)"/>';
      }

      return html;
    }

    function drawReleaseFactor(x){
      return '<g>'
        +'<path d="M'+x+' 205 V166'
        +' M'+x+' 183 L'+(x-20)+' 166'
        +' M'+x+' 183 L'+(x+20)+' 166'
        +'" fill="none" stroke="#64748B" stroke-width="8" stroke-linecap="round"/>'
        +'<circle cx="'+x+'" cy="145" r="21'
        +'" fill="#E2E8F0" stroke="#64748B" stroke-width="4"/>'
        +'<text x="'+x+'" y="150" text-anchor="middle" font-size="10" font-weight="900" fill="#334155">释放</text>'
        +'</g>';
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(!automatic || !document.body.contains(root)){
        return;
      }

      var activity=Number(ribosomeInput.value);
      var interval=clamp(
        3600-activity*21,
        1000,
        3200
      );

      timer=window.setTimeout(function(){
        var index=stages.indexOf(stage);
        stage=stages[(index+1)%stages.length];
        update();
        schedule();
      },interval);
    }

    function update(){
      var count=Math.round(Number(codonInput.value));
      var ribosomeActivity=Number(ribosomeInput.value);
      var trnaSupply=Number(trnaInput.value);

      var sequence=sequenceFor(count);
      var stopIndex=sequence.length-1;

      codonValue.textContent=count.toFixed(0)+' 个';
      ribosomeValue.textContent=ribosomeActivity.toFixed(0)+'%';
      trnaValue.textContent=trnaSupply.toFixed(0)+'%';

      root.style.setProperty(
        '--tl-speed',
        clamp(
          2.6-ribosomeActivity/55,
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

      var stageIndex=stages.indexOf(stage);
      var info=information[stage];

      var activeIndex=0;
      var peptideCount=1;

      if(stage==='recognition'){
        activeIndex=Math.min(1,stopIndex-1);
        peptideCount=1;
      }else if(stage==='bond'){
        activeIndex=Math.min(2,stopIndex-1);
        peptideCount=Math.min(2,stopIndex);
      }else if(stage==='translocation'){
        activeIndex=Math.min(
          Math.max(2,Math.floor(stopIndex*.65)),
          stopIndex-1
        );
        peptideCount=Math.min(activeIndex,stopIndex);
      }else if(stage==='termination'){
        activeIndex=stopIndex;
        peptideCount=stopIndex;
      }

      var conditionFactor=
        ribosomeActivity/100
        *trnaSupply/100;

      var completionBase=[
        15,
        35,
        56,
        78,
        100
      ][stageIndex];

      var adjustedCompletion=stage==='termination'
        ?100
        :completionBase*(.62+.38*conditionFactor);

      completion.textContent=adjustedCompletion.toFixed(0)+'%';
      peptideLength.textContent=peptideCount.toFixed(0)+' aa';

      progress.setAttribute(
        'width',
        String(624*adjustedCompletion/100)
      );

      title.textContent=info.title;
      summary.textContent=info.summary;

      var html=drawMrna(sequence,activeIndex);

      if(stage==='initiation'){
        html+=drawRibosome(0,1);
        html+=drawTrna(76,sequence[0],.95);
        html+=drawPeptide(sequence,1,false);
        stageNote.textContent='AUG决定起始位置';
      }else if(stage==='recognition'){
        html+=drawRibosome(activeIndex,1);
        html+=drawTrna(
          76+activeIndex*72,
          sequence[activeIndex],
          .35+.65*trnaSupply/100
        );
        html+=drawPeptide(sequence,peptideCount,false);
        stageNote.textContent='密码子—反密码子配对';
      }else if(stage==='bond'){
        html+=drawRibosome(activeIndex,1);
        html+=drawTrna(
          76+(activeIndex-1)*72,
          sequence[activeIndex-1],
          .8
        );
        html+=drawTrna(
          76+activeIndex*72,
          sequence[activeIndex],
          .8
        );
        html+=drawPeptide(sequence,peptideCount,false);
        stageNote.textContent='氨基酸之间形成肽键';
      }else if(stage==='translocation'){
        html+=drawRibosome(activeIndex,1);
        html+=drawTrna(
          76+activeIndex*72,
          sequence[activeIndex],
          .82
        );
        html+=drawPeptide(sequence,peptideCount,false);

        html+='<path class="tl-flow" d="M'
          +(76+(activeIndex-1)*72)+' 310 H'
          +(76+activeIndex*72)
          +'" fill="none" stroke="#DB2777" stroke-width="4'
          +'" marker-end="url(#${rootId}-arrow)"/>';

        stageNote.textContent='沿mRNA向3′端移动';
      }else{
        html+=drawRibosome(stopIndex,.55);
        html+=drawReleaseFactor(76+stopIndex*72);
        html+=drawPeptide(sequence,peptideCount,true);
        stageNote.textContent='终止密码子不编码氨基酸';
      }

      graphic.innerHTML=html;

      var condition='当前核糖体活跃度和tRNA供应水平相对协调。';

      if(ribosomeActivity<35){
        condition='核糖体活跃度较低，密码子读取和多肽链延伸速度受到限制。';
      }else if(trnaSupply<35){
        condition='tRNA供应水平较低，携带相应氨基酸的tRNA进入核糖体受到限制。';
      }else if(ribosomeActivity>88 && trnaSupply<55){
        condition='核糖体活跃度较高而tRNA供应相对不足，翻译延伸可能受限。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' 遗传密码具有一定通用性，但少数细胞器和生物中存在例外。';
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

    codonInput.oninput=update;
    trnaInput.oninput=update;

    ribosomeInput.oninput=function(){
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
