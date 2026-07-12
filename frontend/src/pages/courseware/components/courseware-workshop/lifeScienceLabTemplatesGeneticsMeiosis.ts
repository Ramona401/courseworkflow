/**
 * lifeScienceLabTemplatesGeneticsMeiosis.ts
 *
 * 平面生命科学实验室：减数分裂互动模型。
 *
 * 教学边界：
 * 1. 展示减数第一次分裂和减数第二次分裂的主要阶段；
 * 2. DNA只在减数第一次分裂前复制一次；
 * 3. 第一次分裂分离同源染色体，第二次分裂分离姐妹染色单体；
 * 4. 最终形成四个染色体数目减半的子细胞；
 * 5. 染色体数量和交换位置均为教学示意，不对应特定物种。
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

function meiosisStyle(rootId: string): string {
  return ''
    + '<style>\n'
    + '#' + rootId + '{width:100%;height:100%;box-sizing:border-box;border:1px solid #C7D2FE;border-radius:16px;background:#FFFFFF;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937;}\n'
    + '#' + rootId + ' *{box-sizing:border-box;}\n'
    + '#' + rootId + ' .ms-head{height:46px;display:flex;align-items:center;justify-content:space-between;padding:0 16px;background:linear-gradient(135deg,#E0E7FF,#F5F3FF);border-bottom:1px solid #C7D2FE;}\n'
    + '#' + rootId + ' .ms-title{font-size:15px;font-weight:800;color:#4338CA;}\n'
    + '#' + rootId + ' .ms-note{font-size:12px;color:#64748B;}\n'
    + '#' + rootId + ' .ms-body{height:calc(100% - 46px);display:grid;grid-template-columns:248px minmax(0,1fr);min-height:0;}\n'
    + '#' + rootId + ' .ms-controls{padding:13px;border-right:1px solid #C7D2FE;background:#FAFAFF;overflow:auto;}\n'
    + '#' + rootId + ' .ms-stage{position:relative;min-width:0;min-height:0;background:#FFFFFF;}\n'
    + '#' + rootId + ' .ms-row{margin-bottom:11px;}\n'
    + '#' + rootId + ' .ms-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155;}\n'
    + '#' + rootId + ' .ms-value{font-weight:800;color:#4F46E5;white-space:nowrap;}\n'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#4F46E5;}\n'
    + '#' + rootId + ' .ms-stage-title{margin:8px 0 7px;font-size:12px;font-weight:800;color:#4338CA;}\n'
    + '#' + rootId + ' .ms-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:9px;}\n'
    + '#' + rootId + ' .ms-button{height:30px;padding:0 3px;border:1px solid #A5B4FC;border-radius:8px;background:#FFFFFF;color:#4338CA;font-size:10.5px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .ms-button.active{border-color:#4F46E5;background:#E0E7FF;box-shadow:0 3px 9px rgba(79,70,229,.13);}\n'
    + '#' + rootId + ' .ms-auto{width:100%;height:32px;margin-bottom:9px;border:0;border-radius:8px;background:linear-gradient(135deg,#818CF8,#4F46E5);color:#FFFFFF;font-size:11px;font-weight:800;cursor:pointer;}\n'
    + '#' + rootId + ' .ms-auto.paused{background:#64748B;}\n'
    + '#' + rootId + ' .ms-result{padding:9px 10px;border-radius:10px;background:#E0E7FF;color:#312E81;font-size:11.5px;line-height:1.52;font-weight:600;}\n'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%;}\n'
    + '#' + rootId + ' .ms-progress{transition:width .25s ease;}\n'
    + '#' + rootId + ' .ms-dynamic *{transition:opacity .22s ease,transform .22s ease;}\n'
    + '</style>\n'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MEIOSIS:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-meiosis',
    group: '🧬 遗传与细胞分裂',
    name: '减数分裂',
    emoji: '🔬',
    desc: '观察同源染色体联会、交换和两次连续分裂，理解四个单倍体子细胞的形成',
    params: [
      {
        key: 'homologousPairs',
        label: '同源染色体对数',
        type: 'number',
        min: 1,
        max: 4,
        step: 1,
        defaultValue: 2,
      },
      {
        key: 'crossingOver',
        label: '交换显示强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'animationSpeed',
        label: '自动演示速度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
    ],

    buildHTML: (params, rootId) => {
      const homologousPairs = num(params, 'homologousPairs', 2)
      const crossingOver = num(params, 'crossingOver', 72)
      const animationSpeed = num(params, 'animationSpeed', 58)

      return `
<div id="${rootId}">
${meiosisStyle(rootId)}
  <div class="ms-head">
    <div class="ms-title">🔬 减数分裂与遗传多样性</div>
    <div class="ms-note">染色体数量、形状和交换位置均为教学示意</div>
  </div>

  <div class="ms-body">
    <div class="ms-controls">
      <div class="ms-row">
        <div class="ms-label">
          <span>同源染色体对数</span>
          <span class="ms-value" data-pairs-value></span>
        </div>
        <input
          data-pairs
          type="range"
          min="1"
          max="4"
          step="1"
          value="${n(homologousPairs)}"
        >
      </div>

      <div class="ms-row">
        <div class="ms-label">
          <span>交换显示强度</span>
          <span class="ms-value" data-crossing-value></span>
        </div>
        <input
          data-crossing
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(crossingOver)}"
        >
      </div>

      <div class="ms-row">
        <div class="ms-label">
          <span>自动演示速度</span>
          <span class="ms-value" data-speed-value></span>
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

      <div class="ms-stage-title">选择减数分裂阶段</div>

      <div class="ms-buttons">
        <button type="button" class="ms-button active" data-stage="interphase">间期</button>
        <button type="button" class="ms-button" data-stage="prophase1">前期Ⅰ</button>
        <button type="button" class="ms-button" data-stage="metaphase1">中期Ⅰ</button>
        <button type="button" class="ms-button" data-stage="anaphase1">后期Ⅰ</button>
        <button type="button" class="ms-button" data-stage="telophase1">末期Ⅰ</button>
        <button type="button" class="ms-button" data-stage="prophase2">前期Ⅱ</button>
        <button type="button" class="ms-button" data-stage="metaphase2">中期Ⅱ</button>
        <button type="button" class="ms-button" data-stage="anaphase2">后期Ⅱ</button>
        <button type="button" class="ms-button" data-stage="telophase2">末期Ⅱ</button>
      </div>

      <button
        type="button"
        class="ms-auto"
        data-auto
      >自动演示：运行中</button>

      <div class="ms-result" data-result></div>
    </div>

    <div class="ms-stage">
      <svg viewBox="0 0 680 414" aria-label="减数分裂过程互动示意图">
        <defs>
          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="6"
              stdDeviation="8"
              flood-color="#4338CA"
              flood-opacity=".13"
            />
          </filter>

          <radialGradient id="${rootId}-cell" cx="38%" cy="30%" r="72%">
            <stop offset="0%" stop-color="#F5F3FF"/>
            <stop offset="100%" stop-color="#E0E7FF"/>
          </radialGradient>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="28"
          y="38"
          data-stage-name
          font-size="27"
          font-weight="900"
          fill="#4338CA"
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
          x="480"
          y="38"
          data-ploidy-note
          font-size="15"
          font-weight="900"
          fill="#4F46E5"
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
          class="ms-progress"
          x="28"
          y="88"
          width="0"
          height="10"
          rx="5"
          fill="#6366F1"
        />

        <g
          class="ms-dynamic"
          data-dynamic
          filter="url(#${rootId}-shadow)"
        ></g>

        <g transform="translate(28 370)">
          <rect x="0" y="-10" width="18" height="8" rx="4" fill="#2563EB"/>
          <text x="26" y="0" font-size="13" font-weight="800" fill="#475569">
            父方同源染色体
          </text>
        </g>

        <g transform="translate(218 370)">
          <rect x="0" y="-10" width="18" height="8" rx="4" fill="#EC4899"/>
          <text x="26" y="0" font-size="13" font-weight="800" fill="#475569">
            母方同源染色体
          </text>
        </g>

        <g transform="translate(408 370)">
          <rect x="0" y="-10" width="18" height="8" rx="4" fill="#F59E0B"/>
          <text x="26" y="0" font-size="13" font-weight="800" fill="#475569">
            交换片段
          </text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var pairsInput=root.querySelector('[data-pairs]');
    var crossingInput=root.querySelector('[data-crossing]');
    var speedInput=root.querySelector('[data-speed]');

    var pairsValue=root.querySelector('[data-pairs-value]');
    var crossingValue=root.querySelector('[data-crossing-value]');
    var speedValue=root.querySelector('[data-speed-value]');

    var buttons=root.querySelectorAll('[data-stage]');
    var autoButton=root.querySelector('[data-auto]');
    var result=root.querySelector('[data-result]');

    var stageName=root.querySelector('[data-stage-name]');
    var stageSummary=root.querySelector('[data-stage-summary]');
    var ploidyNote=root.querySelector('[data-ploidy-note]');
    var progress=root.querySelector('[data-progress]');
    var dynamic=root.querySelector('[data-dynamic]');

    var order=[
      'interphase',
      'prophase1',
      'metaphase1',
      'anaphase1',
      'telophase1',
      'prophase2',
      'metaphase2',
      'anaphase2',
      'telophase2'
    ];

    var information={
      interphase:{
        name:'分裂前间期',
        summary:'DNA复制一次，为两次连续分裂准备遗传物质',
        ploidy:'二倍体细胞 · DNA已复制',
        note:'减数分裂开始前，DNA复制一次；减数第一次分裂和第二次分裂之间通常不再复制DNA。'
      },
      prophase1:{
        name:'减数第一次分裂前期',
        summary:'同源染色体联会，非姐妹染色单体之间可发生交换',
        ploidy:'二倍体 · 同源染色体配对',
        note:'同源染色体联会形成四分体，非姐妹染色单体之间可能交换对应片段。'
      },
      metaphase1:{
        name:'减数第一次分裂中期',
        summary:'同源染色体对排列在赤道板两侧',
        ploidy:'二倍体 · 同源染色体成对排列',
        note:'每对同源染色体的排列方向具有一定随机性，有助于形成不同染色体组合。'
      },
      anaphase1:{
        name:'减数第一次分裂后期',
        summary:'同源染色体分离，姐妹染色单体仍连接在一起',
        ploidy:'同源染色体正在分离',
        note:'第一次分裂分离的是同源染色体，而不是姐妹染色单体。'
      },
      telophase1:{
        name:'减数第一次分裂末期',
        summary:'形成两个染色体数目减半的细胞',
        ploidy:'两个单倍体细胞 · 染色体仍为复制状态',
        note:'第一次分裂完成后，染色体数目减半，但每条染色体通常仍含两条姐妹染色单体。'
      },
      prophase2:{
        name:'减数第二次分裂前期',
        summary:'两个细胞分别建立新的纺锤体，不再进行DNA复制',
        ploidy:'两个单倍体细胞 · 无DNA复制',
        note:'第二次分裂开始前通常没有新的DNA复制，这是与第一次分裂前间期的重要区别。'
      },
      metaphase2:{
        name:'减数第二次分裂中期',
        summary:'染色体分别排列在两个细胞的赤道板附近',
        ploidy:'两个单倍体细胞 · 染色体单列排列',
        note:'染色体在各自细胞中单独排列，纺锤丝连接姐妹染色单体。'
      },
      anaphase2:{
        name:'减数第二次分裂后期',
        summary:'姐妹染色单体分离，并移向每个细胞的两极',
        ploidy:'姐妹染色单体正在分离',
        note:'第二次分裂分离的是姐妹染色单体，其过程与有丝分裂后期具有相似之处。'
      },
      telophase2:{
        name:'减数第二次分裂末期',
        summary:'形成四个染色体数目减半的子细胞',
        ploidy:'四个单倍体子细胞',
        note:'一次减数分裂通常形成四个单倍体子细胞，它们的遗传组成可能彼此不同。'
      }
    };

    var stage='interphase';
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function cell(cx,cy,rx,ry){
      return '<ellipse cx="'+cx+'" cy="'+cy+'" rx="'+rx+'" ry="'+ry
        +'" fill="url(#${rootId}-cell)" stroke="#6366F1" stroke-width="5"/>';
    }

    function nucleus(cx,cy,rx,ry,opacity){
      return '<ellipse cx="'+cx+'" cy="'+cy+'" rx="'+rx+'" ry="'+ry
        +'" fill="#F5D0FE" stroke="#A855F7" stroke-width="4" opacity="'+opacity+'"/>';
    }

    function spindle(x1,y1,x2,y2,opacity){
      return '<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2
        +'" stroke="#0EA5E9" stroke-width="2.4" opacity="'+opacity+'"/>';
    }

    function chromosomeX(x,y,size,color,rotation,exchangeSide,exchangeOpacity){
      var segment='';

      if(exchangeSide!==0 && exchangeOpacity>0){
        var segmentX=exchangeSide<0?-size:size;

        segment='<path d="M'+segmentX+' '+(-size*1.42)
          +' L'+(-segmentX*.12)+' '+(-size*.18)
          +'" stroke="#F59E0B" stroke-width="'+(size*.5)
          +'" stroke-linecap="round" opacity="'+exchangeOpacity+'"/>';
      }

      return '<g transform="translate('+x+' '+y+') rotate('+rotation+')">'
        +'<path d="M'+(-size)+' '+(-size*1.4)+' L'+size+' '+(size*1.4)
        +'" stroke="'+color+'" stroke-width="'+(size*.48)
        +'" stroke-linecap="round"/>'
        +'<path d="M'+size+' '+(-size*1.4)+' L'+(-size)+' '+(size*1.4)
        +'" stroke="'+color+'" stroke-width="'+(size*.48)
        +'" stroke-linecap="round"/>'
        +segment
        +'<circle cx="0" cy="0" r="'+(size*.31)
        +'" fill="#FDE68A" stroke="#B45309" stroke-width="1.4"/>'
        +'</g>';
    }

    function chromatid(x,y,size,color,direction,exchangeOpacity){
      var flip=direction<0?-1:1;
      var segment=exchangeOpacity>0
        ?'<path d="M'+(size*.5)+' '+(-size*.42)
          +' L'+(size*1.2)+' '+(-size)
          +'" stroke="#F59E0B" stroke-width="'+(size*.4)
          +'" stroke-linecap="round" opacity="'+exchangeOpacity+'"/>'
        :'';

      return '<g transform="translate('+x+' '+y+') scale('+flip+' 1)">'
        +'<path d="M0 0 L'+(size*1.22)+' '+(-size)
        +' M0 0 L'+(size*1.22)+' '+size
        +'" stroke="'+color+'" stroke-width="'+(size*.42)
        +'" stroke-linecap="round"/>'
        +segment
        +'<circle cx="0" cy="0" r="'+(size*.27)
        +'" fill="#FDE68A" stroke="#B45309" stroke-width="1.3"/>'
        +'</g>';
    }

    function chromatinCurve(x,y,color,index){
      return '<path d="M'+(x-13)+' '+(y-4)
        +' Q'+(x-4)+' '+(y-17)+' '+(x+9)+' '+(y-5)
        +' Q'+(x+17)+' '+(y+8)+' '+(x-9)+' '+(y+10)
        +'" fill="none" stroke="'+color+'" stroke-width="4'
        +'" stroke-linecap="round" opacity="'+(.62+(index%3)*.1)+'"/>';
    }

    function renderInterphase(pairs){
      var html=cell(340,226,202,120);
      html+=nucleus(340,226,104,79,1);

      for(var i=0;i<pairs*4;i++){
        var angle=i*2.399;
        var radius=20+(i%5)*13;
        var x=340+Math.cos(angle)*radius;
        var y=226+Math.sin(angle)*radius*.68;
        var color=i%2===0?'#2563EB':'#EC4899';

        html+=chromatinCurve(x,y,color,i);
      }

      html+='<circle cx="340" cy="226" r="15" fill="#A855F7" opacity=".62"/>';

      return html;
    }

    function renderProphase1(pairs,crossing){
      var html=cell(340,226,202,120);
      html+=nucleus(340,226,108,80,.28);

      var spacing=Math.min(65,180/Math.max(1,pairs-1));
      var exchangeOpacity=.15+.85*crossing/100;

      for(var i=0;i<pairs;i++){
        var y=226-(pairs-1)*spacing/2+i*spacing;

        html+=chromosomeX(
          319,
          y,
          13,
          '#2563EB',
          -10,
          -1,
          exchangeOpacity
        );

        html+=chromosomeX(
          361,
          y,
          13,
          '#EC4899',
          10,
          1,
          exchangeOpacity
        );

        html+='<ellipse cx="340" cy="'+y+'" rx="48" ry="27'
          +'" fill="none" stroke="#A78BFA" stroke-width="3'
          +'" stroke-dasharray="6 5" opacity=".8"/>';
      }

      html+='<text x="270" y="338" font-size="15" font-weight="900" fill="#4338CA">'
        +'同源染色体联会形成四分体</text>';

      return html;
    }

    function renderMetaphase1(pairs,crossing){
      var html=cell(340,226,202,120);
      var spacing=Math.min(56,165/Math.max(1,pairs-1));
      var exchangeOpacity=.12+.78*crossing/100;

      html+='<circle cx="151" cy="226" r="12" fill="#0EA5E9"/>'
        +'<circle cx="529" cy="226" r="12" fill="#0EA5E9"/>'
        +'<line x1="340" y1="113" x2="340" y2="339'
        +'" stroke="#CBD5E1" stroke-width="3" stroke-dasharray="7 7"/>';

      for(var i=0;i<pairs;i++){
        var y=226-(pairs-1)*spacing/2+i*spacing;
        var orientation=i%2===0?1:-1;
        var blueX=orientation>0?319:361;
        var pinkX=orientation>0?361:319;

        html+=spindle(163,226,blueX-12,y,.72);
        html+=spindle(517,226,pinkX+12,y,.72);

        html+=chromosomeX(
          blueX,
          y,
          12,
          '#2563EB',
          -8,
          -1,
          exchangeOpacity
        );

        html+=chromosomeX(
          pinkX,
          y,
          12,
          '#EC4899',
          8,
          1,
          exchangeOpacity
        );
      }

      return html;
    }

    function renderAnaphase1(pairs,crossing){
      var html=cell(340,226,202,120);
      var spacing=Math.min(53,155/Math.max(1,pairs-1));
      var exchangeOpacity=.12+.78*crossing/100;

      html+='<circle cx="151" cy="226" r="12" fill="#0EA5E9"/>'
        +'<circle cx="529" cy="226" r="12" fill="#0EA5E9"/>';

      for(var i=0;i<pairs;i++){
        var y=226-(pairs-1)*spacing/2+i*spacing;
        var blueLeft=i%2===0;
        var leftColor=blueLeft?'#2563EB':'#EC4899';
        var rightColor=blueLeft?'#EC4899':'#2563EB';

        html+=spindle(163,226,247,y,.75);
        html+=spindle(517,226,433,y,.75);

        html+=chromosomeX(
          247,
          y,
          12,
          leftColor,
          -8,
          blueLeft?-1:1,
          exchangeOpacity
        );

        html+=chromosomeX(
          433,
          y,
          12,
          rightColor,
          8,
          blueLeft?1:-1,
          exchangeOpacity
        );
      }

      return html;
    }

    function renderTelophase1(pairs,crossing){
      var html=cell(232,226,110,103);
      html+=cell(448,226,110,103);
      html+=nucleus(232,226,66,67,1);
      html+=nucleus(448,226,66,67,1);

      var spacing=Math.min(40,112/Math.max(1,pairs-1));
      var exchangeOpacity=.1+.72*crossing/100;

      for(var i=0;i<pairs;i++){
        var y=226-(pairs-1)*spacing/2+i*spacing;
        var blueLeft=i%2===0;

        html+=chromosomeX(
          232,
          y,
          10,
          blueLeft?'#2563EB':'#EC4899',
          i%2===0?-10:10,
          blueLeft?-1:1,
          exchangeOpacity
        );

        html+=chromosomeX(
          448,
          y,
          10,
          blueLeft?'#EC4899':'#2563EB',
          i%2===0?10:-10,
          blueLeft?1:-1,
          exchangeOpacity
        );
      }

      html+='<text x="160" y="350" font-size="14" font-weight="900" fill="#4338CA">'
        +'染色体数目减半</text>'
        +'<text x="376" y="350" font-size="14" font-weight="900" fill="#4338CA">'
        +'染色体数目减半</text>';

      return html;
    }

    function renderProphase2(pairs,crossing){
      var html=cell(232,226,110,103);
      html+=cell(448,226,110,103);

      var spacing=Math.min(40,112/Math.max(1,pairs-1));
      var exchangeOpacity=.1+.72*crossing/100;

      for(var i=0;i<pairs;i++){
        var y=226-(pairs-1)*spacing/2+i*spacing;
        var blueLeft=i%2===0;

        html+=chromosomeX(
          232,
          y,
          10,
          blueLeft?'#2563EB':'#EC4899',
          i*12,
          blueLeft?-1:1,
          exchangeOpacity
        );

        html+=chromosomeX(
          448,
          y,
          10,
          blueLeft?'#EC4899':'#2563EB',
          -i*12,
          blueLeft?1:-1,
          exchangeOpacity
        );
      }

      html+='<circle cx="143" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="321" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="359" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="537" cy="226" r="9" fill="#0EA5E9"/>';

      return html;
    }

    function renderMetaphase2(pairs,crossing){
      var html=cell(232,226,110,103);
      html+=cell(448,226,110,103);

      var spacing=Math.min(38,104/Math.max(1,pairs-1));
      var exchangeOpacity=.1+.72*crossing/100;

      html+='<circle cx="143" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="321" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="359" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="537" cy="226" r="9" fill="#0EA5E9"/>'
        +'<line x1="232" y1="136" x2="232" y2="316'
        +'" stroke="#CBD5E1" stroke-width="2.5" stroke-dasharray="6 6"/>'
        +'<line x1="448" y1="136" x2="448" y2="316'
        +'" stroke="#CBD5E1" stroke-width="2.5" stroke-dasharray="6 6"/>';

      for(var i=0;i<pairs;i++){
        var y=226-(pairs-1)*spacing/2+i*spacing;
        var blueLeft=i%2===0;

        html+=spindle(151,226,221,y,.7);
        html+=spindle(313,226,243,y,.7);
        html+=spindle(367,226,437,y,.7);
        html+=spindle(529,226,459,y,.7);

        html+=chromosomeX(
          232,
          y,
          9,
          blueLeft?'#2563EB':'#EC4899',
          0,
          blueLeft?-1:1,
          exchangeOpacity
        );

        html+=chromosomeX(
          448,
          y,
          9,
          blueLeft?'#EC4899':'#2563EB',
          0,
          blueLeft?1:-1,
          exchangeOpacity
        );
      }

      return html;
    }

    function renderAnaphase2(pairs,crossing){
      var html=cell(232,226,110,103);
      html+=cell(448,226,110,103);

      var spacing=Math.min(36,98/Math.max(1,pairs-1));
      var exchangeOpacity=.08+.62*crossing/100;

      html+='<circle cx="143" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="321" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="359" cy="226" r="9" fill="#0EA5E9"/>'
        +'<circle cx="537" cy="226" r="9" fill="#0EA5E9"/>';

      for(var i=0;i<pairs;i++){
        var y=226-(pairs-1)*spacing/2+i*spacing;
        var blueLeft=i%2===0;
        var leftColor=blueLeft?'#2563EB':'#EC4899';
        var rightColor=blueLeft?'#EC4899':'#2563EB';

        html+=spindle(151,226,193,y,.72);
        html+=spindle(313,226,271,y,.72);
        html+=spindle(367,226,409,y,.72);
        html+=spindle(529,226,487,y,.72);

        html+=chromatid(193,y,9,leftColor,-1,exchangeOpacity);
        html+=chromatid(271,y,9,leftColor,1,exchangeOpacity);
        html+=chromatid(409,y,9,rightColor,-1,exchangeOpacity);
        html+=chromatid(487,y,9,rightColor,1,exchangeOpacity);
      }

      return html;
    }

    function renderTelophase2(pairs,crossing){
      var centers=[
        [160,190],
        [300,190],
        [440,190],
        [580,190]
      ];

      var html='';
      var exchangeOpacity=.08+.62*crossing/100;

      for(var c=0;c<centers.length;c++){
        var cx=centers[c][0];
        var cy=centers[c][1];

        html+=cell(cx,cy,69,72);
        html+=nucleus(cx,cy,43,45,1);

        for(var i=0;i<pairs;i++){
          var angle=Math.PI*2*i/Math.max(1,pairs);
          var x=cx+Math.cos(angle)*22;
          var y=cy+Math.sin(angle)*20;
          var color=(c+i)%2===0?'#2563EB':'#EC4899';

          html+='<path d="M'+(x-7)+' '+(y-8)
            +' Q'+x+' '+(y-14)+' '+(x+7)+' '+(y-2)
            +'" fill="none" stroke="'+color+'" stroke-width="4'
            +'" stroke-linecap="round"/>';

          if(exchangeOpacity>.3 && (c+i)%2===0){
            html+='<circle cx="'+(x+5)+'" cy="'+(y-5)
              +'" r="3" fill="#F59E0B" opacity="'+exchangeOpacity+'"/>';
          }
        }

        html+='<text x="'+(cx-34)+'" y="292" font-size="13'
          +'" font-weight="900" fill="#4338CA">单倍体细胞'
          +(c+1)+'</text>';
      }

      html+='<text x="215" y="326" font-size="16" font-weight="900" fill="#312E81">'
        +'四个子细胞的染色体数目均为母细胞的一半</text>';

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
      var interval=4700-speed*30;

      timer=window.setTimeout(function(){
        var index=order.indexOf(stage);
        stage=order[(index+1)%order.length];
        update();
        schedule();
      },clamp(interval,1300,4100));
    }

    function update(){
      var pairs=Math.round(Number(pairsInput.value));
      var crossing=Number(crossingInput.value);
      var speed=Number(speedInput.value);

      pairsValue.textContent=pairs.toFixed(0)+' 对';
      crossingValue.textContent=crossing.toFixed(0)+'%';
      speedValue.textContent=speed.toFixed(0)+'%';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      var index=order.indexOf(stage);
      var info=information[stage];

      stageName.textContent=info.name;
      stageSummary.textContent=info.summary;
      ploidyNote.textContent=info.ploidy;

      progress.setAttribute(
        'width',
        String(624*(index+1)/order.length)
      );

      if(stage==='interphase'){
        dynamic.innerHTML=renderInterphase(pairs);
      }else if(stage==='prophase1'){
        dynamic.innerHTML=renderProphase1(pairs,crossing);
      }else if(stage==='metaphase1'){
        dynamic.innerHTML=renderMetaphase1(pairs,crossing);
      }else if(stage==='anaphase1'){
        dynamic.innerHTML=renderAnaphase1(pairs,crossing);
      }else if(stage==='telophase1'){
        dynamic.innerHTML=renderTelophase1(pairs,crossing);
      }else if(stage==='prophase2'){
        dynamic.innerHTML=renderProphase2(pairs,crossing);
      }else if(stage==='metaphase2'){
        dynamic.innerHTML=renderMetaphase2(pairs,crossing);
      }else if(stage==='anaphase2'){
        dynamic.innerHTML=renderAnaphase2(pairs,crossing);
      }else{
        dynamic.innerHTML=renderTelophase2(pairs,crossing);
      }

      var crossingNote='';

      if(crossing<15){
        crossingNote='当前交换显示较弱，主要观察染色体分离过程。';
      }else if(stage==='prophase1'){
        crossingNote='橙色片段表示非姐妹染色单体之间发生交换的示意区域。';
      }else if(stage==='telophase2'){
        crossingNote='交换和染色体独立组合共同增加了子细胞遗传组成的多样性。';
      }else{
        crossingNote='交换产生的片段会随染色体进入后续子细胞。';
      }

      result.innerHTML=info.note
        +'<br>'+crossingNote
        +' 本模型不表示真实交换频率，也不对应特定生物的染色体数量。';
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

    pairsInput.oninput=update;
    crossingInput.oninput=update;

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
