/**
 * lifeScienceLabTemplatesMicrobiologyVirus.ts
 *
 * 平面生命科学实验室：病毒侵染与复制。
 *
 * 教学边界：
 * 1. 展示吸附、进入、复制与合成、组装、释放五个通用阶段；
 * 2. 病毒必须依赖活细胞完成复制，不能独立进行完整生命活动；
 * 3. 不同病毒的遗传物质、进入方式和释放方式并不完全相同；
 * 4. 图中病毒数量、速率和结构比例均为教学示意。
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

function virusStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FBCFE8;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .vr-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#FFF1F2);border-bottom:1px solid #FBCFE8}'
    + '#' + rootId + ' .vr-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .vr-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .vr-body{height:calc(100% - 46px);display:grid;grid-template-columns:238px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .vr-controls{padding:13px;overflow:auto;background:#FFF9FC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .vr-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .vr-row{margin-bottom:11px}'
    + '#' + rootId + ' .vr-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .vr-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#DB2777}'
    + '#' + rootId + ' .vr-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .vr-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .vr-button{height:31px;padding:0 4px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .vr-button.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.13)}'
    + '#' + rootId + ' .vr-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .vr-auto.paused{background:#64748B}'
    + '#' + rootId + ' .vr-result{padding:9px 10px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:11.5px;line-height:1.52;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .vr-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.5s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_VIRUS:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-virus-infection-replication',
    group: '🦠 微生物与免疫',
    name: '病毒侵染与复制',
    emoji: '🧫',
    desc: '切换吸附、进入、复制、组装和释放阶段，观察受体匹配与宿主细胞依赖',
    params: [
      {
        key: 'receptorMatch',
        label: '受体匹配程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 78,
      },
      {
        key: 'entryEfficiency',
        label: '进入效率',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'replicationActivity',
        label: '复制活跃度',
        type: 'number',
        min: 10,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
    ],

    buildHTML: (params, rootId) => {
      const receptorMatch = num(params, 'receptorMatch', 78)
      const entryEfficiency = num(params, 'entryEfficiency', 68)
      const replicationActivity = num(params, 'replicationActivity', 72)

      return `
<div id="${rootId}">
${virusStyle(rootId)}
  <div class="vr-head">
    <div class="vr-title">🧫 病毒侵染与复制过程</div>
    <div class="vr-note">不同病毒的进入、复制和释放方式可能不同</div>
  </div>

  <div class="vr-body">
    <div class="vr-controls">
      <div class="vr-row">
        <div class="vr-label">
          <span>受体匹配程度</span>
          <span class="vr-value" data-receptor-value></span>
        </div>
        <input
          data-receptor
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(receptorMatch)}"
        >
      </div>

      <div class="vr-row">
        <div class="vr-label">
          <span>进入效率</span>
          <span class="vr-value" data-entry-value></span>
        </div>
        <input
          data-entry
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(entryEfficiency)}"
        >
      </div>

      <div class="vr-row">
        <div class="vr-label">
          <span>复制活跃度</span>
          <span class="vr-value" data-replication-value></span>
        </div>
        <input
          data-replication
          type="range"
          min="10"
          max="100"
          step="1"
          value="${n(replicationActivity)}"
        >
      </div>

      <div class="vr-subtitle">选择生命周期阶段</div>

      <div class="vr-buttons">
        <button type="button" class="vr-button active" data-stage="attachment">1. 吸附</button>
        <button type="button" class="vr-button" data-stage="entry">2. 进入</button>
        <button type="button" class="vr-button" data-stage="replication">3. 复制与合成</button>
        <button type="button" class="vr-button" data-stage="assembly">4. 组装</button>
        <button type="button" class="vr-button" data-stage="release">5. 释放</button>
      </div>

      <button
        type="button"
        class="vr-auto"
        data-auto
      >自动演示：运行中</button>

      <div class="vr-result" data-result></div>
    </div>

    <div class="vr-stage">
      <svg viewBox="0 0 680 414" aria-label="病毒侵染与复制互动示意图">
        <defs>
          <marker id="${rootId}-arrow" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DB2777"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="6"
              stdDeviation="7"
              flood-color="#831843"
              flood-opacity=".14"
            />
          </filter>

          <radialGradient id="${rootId}-cell" cx="38%" cy="30%" r="72%">
            <stop offset="0%" stop-color="#FFF7ED"/>
            <stop offset="100%" stop-color="#FFEDD5"/>
          </radialGradient>
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
          fill="#F1F5F9"
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

        <g transform="translate(28 369)">
          <circle cx="7" cy="7" r="7" fill="#DB2777"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            病毒颗粒
          </text>
        </g>

        <g transform="translate(166 369)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            病毒遗传物质
          </text>
        </g>

        <g transform="translate(344 369)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            宿主细胞
          </text>
        </g>

        <text
          x="495"
          y="381"
          data-output-note
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

    var receptor=root.querySelector('[data-receptor]');
    var entry=root.querySelector('[data-entry]');
    var replication=root.querySelector('[data-replication]');

    var receptorValue=root.querySelector('[data-receptor-value]');
    var entryValue=root.querySelector('[data-entry-value]');
    var replicationValue=root.querySelector('[data-replication-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var autoButton=root.querySelector('[data-auto]');
    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var progress=root.querySelector('[data-progress]');
    var graphic=root.querySelector('[data-graphic]');
    var outputNote=root.querySelector('[data-output-note]');
    var result=root.querySelector('[data-result]');

    var stages=[
      'attachment',
      'entry',
      'replication',
      'assembly',
      'release'
    ];

    var information={
      attachment:{
        title:'阶段1：吸附',
        summary:'病毒表面结构与宿主细胞受体发生特异性识别',
        note:'受体匹配是病毒能否有效吸附特定宿主细胞的重要条件之一。'
      },
      entry:{
        title:'阶段2：进入',
        summary:'病毒或其遗传物质进入宿主细胞',
        note:'不同病毒可通过膜融合、胞吞或注入遗传物质等不同方式进入细胞。'
      },
      replication:{
        title:'阶段3：复制与合成',
        summary:'利用宿主细胞的物质和结构合成病毒遗传物质与蛋白质',
        note:'病毒不能独立完成完整复制过程，必须依赖活细胞的合成系统。'
      },
      assembly:{
        title:'阶段4：组装',
        summary:'新合成的遗传物质和蛋白质组装成新的病毒颗粒',
        note:'病毒各组成部分按一定方式组装，形成相对完整的新病毒颗粒。'
      },
      release:{
        title:'阶段5：释放',
        summary:'新病毒颗粒离开宿主细胞并可能继续侵染其他细胞',
        note:'释放可通过细胞裂解、出芽等不同方式完成，具体过程因病毒而异。'
      }
    };

    var stage='attachment';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function virus(x,y,size,opacity){
      var html='<g transform="translate('+x+' '+y+')" opacity="'+opacity+'">';

      for(var i=0;i<10;i++){
        var angle=Math.PI*2*i/10;
        var x1=Math.cos(angle)*(size+3);
        var y1=Math.sin(angle)*(size+3);
        var x2=Math.cos(angle)*(size+14);
        var y2=Math.sin(angle)*(size+14);

        html+='<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2
          +'" stroke="#DB2777" stroke-width="3"/>';

        html+='<circle cx="'+x2+'" cy="'+y2+'" r="4" fill="#F472B6"/>';
      }

      html+='<circle cx="0" cy="0" r="'+size
        +'" fill="#FBCFE8" stroke="#BE185D" stroke-width="4"/>';

      html+='<path d="M'+(-size*.45)+' '+(-size*.15)
        +' Q0 '+(-size*.65)+' '+(size*.45)+' '+(-size*.15)
        +' Q0 '+(size*.55)+' '+(-size*.45)+' '+(-size*.15)
        +'" fill="none" stroke="#2563EB" stroke-width="4"/>';

      html+='</g>';

      return html;
    }

    function hostCell(){
      return '<ellipse cx="402" cy="232" rx="190" ry="120'
        +'" fill="url(#${rootId}-cell)" stroke="#F59E0B" stroke-width="6"/>'
        +'<ellipse cx="438" cy="238" rx="55" ry="47'
        +'" fill="#FED7AA" stroke="#EA580C" stroke-width="4"/>'
        +'<path d="M244 176 Q264 160 284 176'
        +' M238 208 Q260 192 282 208'
        +' M238 242 Q260 226 282 242'
        +'" fill="none" stroke="#10B981" stroke-width="6" stroke-linecap="round"/>';
    }

    function geneticMaterial(count){
      var html='';

      for(var i=0;i<count;i++){
        var x=332+(i%5)*47;
        var y=170+Math.floor(i/5)*47;

        html+='<path d="M'+(x-15)+' '+y
          +' Q'+x+' '+(y-18)+' '+(x+15)+' '+y
          +' Q'+x+' '+(y+18)+' '+(x-15)+' '+y
          +'" fill="none" stroke="#2563EB" stroke-width="4"/>';
      }

      return html;
    }

    function proteinParts(count){
      var html='';

      for(var i=0;i<count;i++){
        var x=320+(i%6)*43;
        var y=154+Math.floor(i/6)*48;

        html+='<polygon points="'
          +x+','+(y-9)+' '
          +(x+9)+','+y+' '
          +x+','+(y+9)+' '
          +(x-9)+','+y
          +'" fill="#F472B6" stroke="#BE185D" stroke-width="2"/>';
      }

      return html;
    }

    function renderAttachment(match){
      var html=hostCell();
      var opacity=.25+.75*match/100;
      var distance=110-65*match/100;

      html+=virus(213-distance,176,27,opacity);
      html+=virus(205-distance,248,23,opacity*.8);

      html+='<path class="vr-flow" d="M'
        +(238-distance)+' 176 H241'
        +'" fill="none" stroke="#DB2777" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)" opacity="'+opacity+'"/>';

      html+='<text x="63" y="322" font-size="15" font-weight="900" fill="#9D174D">'
        +'受体匹配程度 '+match.toFixed(0)+'%</text>';

      return html;
    }

    function renderEntry(match,entryLevel){
      var html=hostCell();
      var success=match/100*entryLevel/100;

      html+=virus(268,208,24,.45+.55*success);

      html+='<path class="vr-flow" d="M285 208 C316 208 322 222 348 229'
        +'" fill="none" stroke="#2563EB" stroke-width="'+(3+6*success)
        +'" marker-end="url(#${rootId}-arrow)" opacity="'+(.25+.75*success)+'"/>';

      html+='<path d="M348 229 Q376 192 405 226 Q380 268 348 229'
        +'" fill="none" stroke="#2563EB" stroke-width="6" opacity="'+success+'"/>';

      html+='<text x="75" y="322" font-size="15" font-weight="900" fill="#1D4ED8">'
        +'相对进入水平 '+(success*100).toFixed(0)+'</text>';

      return html;
    }

    function renderReplication(activity){
      var html=hostCell();
      var count=Math.floor(3+activity/11);

      html+=geneticMaterial(count);
      html+=proteinParts(Math.floor(count*.8));

      html+='<path class="vr-flow" d="M438 238 C468 138 560 144 570 220'
        +'" fill="none" stroke="#DB2777" stroke-width="4'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="74" y="322" font-size="15" font-weight="900" fill="#9D174D">'
        +'宿主细胞内合成病毒成分</text>';

      return html;
    }

    function renderAssembly(activity){
      var html=hostCell();
      var count=Math.floor(3+activity/14);

      for(var i=0;i<count;i++){
        var x=340+(i%4)*58;
        var y=174+Math.floor(i/4)*70;

        html+=virus(x,y,16,.55+i/count*.4);
      }

      html+='<text x="75" y="322" font-size="15" font-weight="900" fill="#9D174D">'
        +'遗传物质与蛋白质组装成新病毒颗粒</text>';

      return html;
    }

    function renderRelease(activity){
      var html=hostCell();
      var count=Math.floor(4+activity/12);

      html+='<path d="M558 166 Q602 215 568 273'
        +'" fill="none" stroke="#EF4444" stroke-width="8" stroke-dasharray="10 8"/>';

      for(var i=0;i<count;i++){
        var angle=-1.2+i*.35;
        var radius=170+(i%3)*35;
        var x=492+Math.cos(angle)*radius;
        var y=220+Math.sin(angle)*radius*.62;

        html+=virus(x,y,13,.5+.5*activity/100);
      }

      html+='<path class="vr-flow" d="M558 220 C590 220 604 208 627 196'
        +'" fill="none" stroke="#DB2777" stroke-width="5'
        +'" marker-end="url(#${rootId}-arrow)"/>';

      html+='<text x="74" y="322" font-size="15" font-weight="900" fill="#9D174D">'
        +'新病毒颗粒离开宿主细胞</text>';

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

      var activity=Number(replication.value);
      var interval=clamp(3600-activity*20,1200,3300);

      timer=window.setTimeout(function(){
        var index=stages.indexOf(stage);
        stage=stages[(index+1)%stages.length];
        update();
        schedule();
      },interval);
    }

    function update(){
      var match=Number(receptor.value);
      var entryLevel=Number(entry.value);
      var activity=Number(replication.value);

      receptorValue.textContent=match.toFixed(0)+'%';
      entryValue.textContent=entryLevel.toFixed(0)+'%';
      replicationValue.textContent=activity.toFixed(0)+'%';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var info=information[stage];
      var index=stages.indexOf(stage);

      title.textContent=info.title;
      summary.textContent=info.summary;

      progress.setAttribute(
        'width',
        String(624*(index+1)/stages.length)
      );

      if(stage==='attachment'){
        graphic.innerHTML=renderAttachment(match);
      }else if(stage==='entry'){
        graphic.innerHTML=renderEntry(match,entryLevel);
      }else if(stage==='replication'){
        graphic.innerHTML=renderReplication(activity);
      }else if(stage==='assembly'){
        graphic.innerHTML=renderAssembly(activity);
      }else{
        graphic.innerHTML=renderRelease(activity);
      }

      var success=match/100*entryLevel/100*activity/100;
      var output=Math.floor(2+success*18);

      outputNote.textContent='相对产出 '+output+' 个';

      var condition='当前条件允许病毒侵染过程相对顺利地推进。';

      if(match<25){
        condition='受体匹配程度较低，病毒难以有效吸附目标细胞。';
      }else if(entryLevel<25){
        condition='进入效率较低，吸附后仍有较多病毒不能进入细胞。';
      }else if(activity<25){
        condition='复制活跃度较低，病毒成分合成和新颗粒形成受到限制。';
      }

      result.innerHTML=info.note
        +'<br>'+condition
        +' 病毒必须依赖活细胞完成复制，不能脱离宿主细胞独立完成完整生命活动。';
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

    receptor.oninput=update;
    entry.oninput=update;

    replication.oninput=function(){
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
