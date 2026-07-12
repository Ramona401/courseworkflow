/**
 * lifeScienceLabTemplatesGeneticsDnaChromosome.ts
 *
 * DNA与染色体互动模型：
 * DNA双螺旋 → 核小体 → 染色质 → 染色体。
 * 图形比例、基因位置及数量均为教学示意。
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

function style(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BFDBFE;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .dc-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DBEAFE,#EEF2FF);border-bottom:1px solid #BFDBFE}'
    + '#' + rootId + ' .dc-title{font-size:15px;font-weight:800;color:#1E40AF}'
    + '#' + rootId + ' .dc-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .dc-body{height:calc(100% - 46px);display:grid;grid-template-columns:230px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .dc-controls{padding:13px;overflow:auto;background:#F8FAFF;border-right:1px solid #BFDBFE}'
    + '#' + rootId + ' .dc-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .dc-row{margin-bottom:12px}'
    + '#' + rootId + ' .dc-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .dc-value{font-weight:800;color:#2563EB;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#2563EB}'
    + '#' + rootId + ' .dc-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#1E40AF}'
    + '#' + rootId + ' .dc-buttons{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:10px}'
    + '#' + rootId + ' .dc-button{height:32px;border:1px solid #93C5FD;border-radius:8px;background:#fff;color:#1E40AF;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .dc-button.active{border-color:#2563EB;background:#DBEAFE;box-shadow:0 3px 9px rgba(37,99,235,.13)}'
    + '#' + rootId + ' .dc-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:10px}'
    + '#' + rootId + ' .dc-card{padding:7px;border:1px solid #BFDBFE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .dc-card b{display:block;font-size:17px;color:#1D4ED8}'
    + '#' + rootId + ' .dc-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .dc-result{padding:9px 10px;border-radius:10px;background:#DBEAFE;color:#1E3A8A;font-size:11.5px;line-height:1.52;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_DNA_CHROMOSOME:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-dna-chromosome',
    group: '🧬 遗传与细胞分裂',
    name: 'DNA与染色体',
    emoji: '🧬',
    desc: '观察DNA、核小体、染色质和染色体的层级关系，比较DNA复制前后的结构与数量',
    params: [
      {
        key: 'condensation',
        label: '凝缩程度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'geneMarkers',
        label: '基因标记数量',
        type: 'number',
        min: 1,
        max: 6,
        step: 1,
        defaultValue: 4,
      },
      {
        key: 'replicationState',
        label: 'DNA复制状态',
        type: 'number',
        min: 0,
        max: 1,
        step: 1,
        defaultValue: 1,
      },
    ],

    buildHTML: (params, rootId) => {
      const condensation = num(params, 'condensation', 68)
      const geneMarkers = num(params, 'geneMarkers', 4)
      const replicationState = num(params, 'replicationState', 1)

      return `
<div id="${rootId}">
${style(rootId)}
  <div class="dc-head">
    <div class="dc-title">🧬 DNA、基因与染色体</div>
    <div class="dc-note">结构比例和基因位置均为教学示意</div>
  </div>

  <div class="dc-body">
    <div class="dc-controls">
      <div class="dc-row">
        <div class="dc-label">
          <span>染色质凝缩程度</span>
          <span class="dc-value" data-cond-value></span>
        </div>
        <input data-cond type="range" min="0" max="100" step="1" value="${n(condensation)}">
      </div>

      <div class="dc-row">
        <div class="dc-label">
          <span>基因标记数量</span>
          <span class="dc-value" data-gene-value></span>
        </div>
        <input data-gene type="range" min="1" max="6" step="1" value="${n(geneMarkers)}">
      </div>

      <div class="dc-row">
        <div class="dc-label">
          <span>DNA复制状态</span>
          <span class="dc-value" data-rep-value></span>
        </div>
        <input data-rep type="range" min="0" max="1" step="1" value="${n(replicationState)}">
      </div>

      <div class="dc-subtitle">选择结构层级</div>

      <div class="dc-buttons">
        <button type="button" class="dc-button active" data-stage="dna">DNA双螺旋</button>
        <button type="button" class="dc-button" data-stage="nucleosome">核小体</button>
        <button type="button" class="dc-button" data-stage="chromatin">染色质</button>
        <button type="button" class="dc-button" data-stage="chromosome">染色体</button>
      </div>

      <div class="dc-status">
        <div class="dc-card">
          <b data-chromosomes>1</b>
          <span>染色体示意数</span>
        </div>
        <div class="dc-card">
          <b data-dna-count>2</b>
          <span>DNA分子示意数</span>
        </div>
      </div>

      <div class="dc-result" data-result></div>
    </div>

    <div class="dc-stage">
      <svg viewBox="0 0 680 414">
        <defs>
          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="6" stdDeviation="7" flood-color="#1E40AF" flood-opacity=".14"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="28" y="40" data-title font-size="27" font-weight="900" fill="#1E40AF"></text>
        <text x="28" y="70" data-summary font-size="15" font-weight="800" fill="#475569"></text>

        <g data-graphic filter="url(#${rootId}-shadow)"></g>

        <g transform="translate(28 368)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">DNA</text>
        </g>

        <g transform="translate(140 368)">
          <circle cx="7" cy="7" r="7" fill="#8B5CF6"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">组蛋白</text>
        </g>

        <g transform="translate(280 368)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">基因片段</text>
        </g>

        <g transform="translate(438 368)">
          <circle cx="7" cy="7" r="7" fill="#EC4899"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">姐妹染色单体</text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var cond=root.querySelector('[data-cond]');
    var gene=root.querySelector('[data-gene]');
    var rep=root.querySelector('[data-rep]');
    var condValue=root.querySelector('[data-cond-value]');
    var geneValue=root.querySelector('[data-gene-value]');
    var repValue=root.querySelector('[data-rep-value]');
    var chromosomes=root.querySelector('[data-chromosomes]');
    var dnaCount=root.querySelector('[data-dna-count]');
    var buttons=root.querySelectorAll('[data-stage]');
    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var graphic=root.querySelector('[data-graphic]');
    var result=root.querySelector('[data-result]');
    var stage='dna';

    var info={
      dna:[
        'DNA双螺旋',
        '两条脱氧核苷酸链通过碱基互补配对形成双螺旋',
        'DNA是主要遗传物质，基因通常是具有遗传效应的DNA片段。'
      ],
      nucleosome:[
        '核小体',
        'DNA缠绕在组蛋白周围，形成染色质的基本结构单位',
        'DNA与蛋白质结合并逐级包装，不是裸露存在于细胞核中。'
      ],
      chromatin:[
        '染色质',
        '核小体链进一步盘曲折叠，形成不同凝缩程度的染色质',
        '分裂间期染色质通常较为舒展，便于DNA复制和基因表达。'
      ],
      chromosome:[
        '染色体',
        '染色质高度凝缩，形成细胞分裂期清晰可见的染色体',
        '染色体主要由DNA和蛋白质组成，是DNA的高度凝缩状态。'
      ]
    };

    function marker(x,y,index){
      var colors=['#F59E0B','#10B981','#EC4899','#8B5CF6','#EF4444','#06B6D4'];
      var color=colors[index%colors.length];

      return '<g>'
        +'<circle cx="'+x+'" cy="'+y+'" r="7" fill="'+color+'" stroke="#fff" stroke-width="2"/>'
        +'<text x="'+(x+11)+'" y="'+(y+5)+'" font-size="12" font-weight="900" fill="'+color+'">基因'+(index+1)+'</text>'
        +'</g>';
    }

    function renderDNA(count){
      var html='';
      var left=[];
      var right=[];

      for(var i=0;i<27;i++){
        var t=i/26;
        var y=102+t*225;
        var wave=Math.sin(t*Math.PI*4);
        var x1=340-72*wave;
        var x2=340+72*wave;

        left.push(x1+','+y);
        right.push(x2+','+y);

        if(i%2===0){
          html+='<line x1="'+x1+'" y1="'+y+'" x2="'+x2+'" y2="'+y
            +'" stroke="'+(i%4===0?'#10B981':'#F59E0B')
            +'" stroke-width="6" stroke-linecap="round" opacity=".82"/>';
        }
      }

      html+='<polyline points="'+left.join(' ')+'" fill="none" stroke="#2563EB" stroke-width="8" stroke-linecap="round"/>';
      html+='<polyline points="'+right.join(' ')+'" fill="none" stroke="#DB2777" stroke-width="8" stroke-linecap="round"/>';

      for(var g=0;g<count;g++){
        var gy=118+(g+1)*190/(count+1);
        html+=marker(470,gy,g);
        html+='<path d="M410 '+gy+' H458" stroke="#F59E0B" stroke-width="3" stroke-dasharray="5 4"/>';
      }

      return html;
    }

    function renderNucleosome(count){
      var html='<path d="M65 224 C150 115 228 330 315 224 C400 115 478 330 615 224" fill="none" stroke="#2563EB" stroke-width="6"/>';

      for(var i=0;i<9;i++){
        var x=100+i*59;
        var y=224+Math.sin(i*1.4)*48;

        html+='<circle cx="'+x+'" cy="'+y+'" r="25" fill="#C4B5FD" stroke="#7C3AED" stroke-width="4"/>';
        html+='<circle cx="'+(x-7)+'" cy="'+(y-5)+'" r="7" fill="#A855F7"/>';
        html+='<circle cx="'+(x+7)+'" cy="'+(y-5)+'" r="7" fill="#8B5CF6"/>';
        html+='<circle cx="'+x+'" cy="'+(y+8)+'" r="7" fill="#6D28D9"/>';

        if(i<count){
          html+=marker(x,y-45,i);
        }
      }

      return html;
    }

    function renderChromatin(count,amount){
      var html='<ellipse cx="340" cy="220" rx="220" ry="118" fill="#EEF2FF" stroke="#C7D2FE" stroke-width="4"/>';
      var loops=Math.floor(5+amount/15);
      var width=9+amount/9;

      for(var i=0;i<loops;i++){
        var a=Math.PI*2*i/loops;
        var x=340+Math.cos(a)*112;
        var y=220+Math.sin(a)*57;

        html+='<ellipse cx="'+x+'" cy="'+y+'" rx="'+(38+amount*.2)
          +'" ry="'+(20+amount*.09)+'" fill="none" stroke="'
          +(i%2===0?'#2563EB':'#7C3AED')
          +'" stroke-width="'+width+'" opacity=".72"/>';
      }

      html+='<circle cx="340" cy="220" r="'+(28+amount*.14)
        +'" fill="#A78BFA" stroke="#6D28D9" stroke-width="4"/>';

      for(var g=0;g<count;g++){
        var angle=Math.PI*2*g/count;
        html+=marker(
          340+Math.cos(angle)*185,
          220+Math.sin(angle)*96,
          g
        );
      }

      return html;
    }

    function renderChromosome(count,replicated,amount){
      var html='<ellipse cx="340" cy="220" rx="220" ry="120" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="4"/>';
      var h=90+amount*.75;

      if(replicated){
        html+='<path d="M285 '+(220-h)+' L395 '+(220+h)
          +'" stroke="#2563EB" stroke-width="30" stroke-linecap="round"/>';
        html+='<path d="M395 '+(220-h)+' L285 '+(220+h)
          +'" stroke="#EC4899" stroke-width="30" stroke-linecap="round"/>';
        html+='<circle cx="340" cy="220" r="18" fill="#FDE68A" stroke="#B45309" stroke-width="4"/>';
      }else{
        html+='<path d="M340 '+(220-h)+' C300 180 300 260 340 '+(220+h)
          +'" fill="none" stroke="#2563EB" stroke-width="34" stroke-linecap="round"/>';
        html+='<circle cx="322" cy="220" r="15" fill="#FDE68A" stroke="#B45309" stroke-width="4"/>';
      }

      for(var g=0;g<count;g++){
        var y=132+g*174/Math.max(1,count-1);
        var x=replicated?(g%2===0?205:460):220;
        html+=marker(x,y,g);
      }

      return html;
    }

    function update(){
      var amount=Number(cond.value);
      var genes=Math.round(Number(gene.value));
      var replicated=Number(rep.value)>=1;

      condValue.textContent=amount.toFixed(0)+'%';
      geneValue.textContent=genes.toFixed(0)+' 个';
      repValue.textContent=replicated?'已复制':'未复制';

      chromosomes.textContent='1';
      dnaCount.textContent=replicated?'2':'1';

      for(var i=0;i<buttons.length;i++){
        buttons[i].classList.toggle(
          'active',
          buttons[i].getAttribute('data-stage')===stage
        );
      }

      title.textContent=info[stage][0];
      summary.textContent=info[stage][1];

      if(stage==='dna'){
        graphic.innerHTML=renderDNA(genes);
      }else if(stage==='nucleosome'){
        graphic.innerHTML=renderNucleosome(genes);
      }else if(stage==='chromatin'){
        graphic.innerHTML=renderChromatin(genes,amount);
      }else{
        graphic.innerHTML=renderChromosome(genes,replicated,amount);
      }

      var stateNote=replicated
        ?'DNA复制后，一条染色体含两条姐妹染色单体和两个DNA分子；着丝粒分裂前仍计为一条染色体。'
        :'DNA复制前，一条染色体通常含一条染色单体和一个DNA分子。';

      var condNote=amount<30
        ?'当前更接近分裂间期较舒展的染色质状态。'
        :amount>78
          ?'当前更接近细胞分裂期高度凝缩的染色体状态。'
          :'当前展示染色质逐级盘曲和凝缩的中间状态。';

      result.innerHTML=info[stage][2]
        +'<br>'+stateNote
        +'<br>'+condNote;
    }

    for(var i=0;i<buttons.length;i++){
      buttons[i].onclick=function(){
        stage=this.getAttribute('data-stage');
        update();
      };
    }

    cond.oninput=update;
    gene.oninput=update;
    rep.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
