/**
 * lifeScienceLabTemplatesMicrobiologyBacteria.ts
 *
 * 平面生命科学实验室：细菌结构与形态。
 *
 * 教学边界：
 * 1. 细菌属于原核生物，没有由核膜包围的细胞核；
 * 2. DNA主要分布在拟核区域，部分细菌含有质粒；
 * 3. 细胞膜、细胞壁和核糖体是常见结构；
 * 4. 荚膜、鞭毛和质粒并非所有细菌都具有；
 * 5. 图中比例、数量和颜色均为教学示意。
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

function bacteriaStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bt-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#F5F3FF);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bt-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bt-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bt-body{height:calc(100% - 46px);display:grid;grid-template-columns:232px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bt-controls{padding:13px;overflow:auto;background:#FBFAFF;border-right:1px solid #C4B5FD}'
    + '#' + rootId + ' .bt-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bt-row{margin-bottom:11px}'
    + '#' + rootId + ' .bt-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bt-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#8B5CF6}'
    + '#' + rootId + ' .bt-subtitle{margin:8px 0 7px;font-size:12px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bt-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:9px}'
    + '#' + rootId + ' .bt-button{height:31px;padding:0 3px;border:1px solid #A78BFA;border-radius:8px;background:#fff;color:#5B21B6;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bt-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.13)}'
    + '#' + rootId + ' .bt-options{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:9px}'
    + '#' + rootId + ' .bt-option{height:31px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;color:#475569;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .bt-option.active{border-color:#8B5CF6;background:#F3E8FF;color:#6B21A8}'
    + '#' + rootId + ' .bt-result{padding:9px 10px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:11.5px;line-height:1.52;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .bt-flagellum{stroke-dasharray:8 7;animation:' + rootId + '-move var(--bt-speed,1.8s) linear infinite}'
    + '@keyframes ' + rootId + '-move{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_MICROBIOLOGY_BACTERIA:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-bacteria-structure',
    group: '🦠 微生物与免疫',
    name: '细菌结构与形态',
    emoji: '🦠',
    desc: '切换球菌、杆菌和螺旋菌，观察拟核、细胞壁、细胞膜、核糖体及可选结构',
    params: [
      {
        key: 'magnification',
        label: '观察放大倍数',
        type: 'number',
        min: 100,
        max: 1000,
        step: 100,
        defaultValue: 600,
      },
      {
        key: 'ribosomeDensity',
        label: '核糖体显示密度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'flagellumActivity',
        label: '鞭毛活动强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
    ],

    buildHTML: (params, rootId) => {
      const magnification = num(params, 'magnification', 600)
      const ribosomeDensity = num(params, 'ribosomeDensity', 68)
      const flagellumActivity = num(params, 'flagellumActivity', 62)

      return `
<div id="${rootId}">
${bacteriaStyle(rootId)}
  <div class="bt-head">
    <div class="bt-title">🦠 细菌结构与形态观察</div>
    <div class="bt-note">荚膜、鞭毛和质粒并非所有细菌都具有</div>
  </div>

  <div class="bt-body">
    <div class="bt-controls">
      <div class="bt-row">
        <div class="bt-label">
          <span>观察放大倍数</span>
          <span class="bt-value" data-mag-value></span>
        </div>
        <input
          data-mag
          type="range"
          min="100"
          max="1000"
          step="100"
          value="${n(magnification)}"
        >
      </div>

      <div class="bt-row">
        <div class="bt-label">
          <span>核糖体显示密度</span>
          <span class="bt-value" data-ribosome-value></span>
        </div>
        <input
          data-ribosome
          type="range"
          min="20"
          max="100"
          step="1"
          value="${n(ribosomeDensity)}"
        >
      </div>

      <div class="bt-row">
        <div class="bt-label">
          <span>鞭毛活动强度</span>
          <span class="bt-value" data-flagellum-value></span>
        </div>
        <input
          data-flagellum
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(flagellumActivity)}"
        >
      </div>

      <div class="bt-subtitle">细菌形态</div>

      <div class="bt-buttons">
        <button type="button" class="bt-button active" data-shape="coccus">球菌</button>
        <button type="button" class="bt-button" data-shape="bacillus">杆菌</button>
        <button type="button" class="bt-button" data-shape="spirillum">螺旋菌</button>
      </div>

      <div class="bt-subtitle">可选结构</div>

      <div class="bt-options">
        <button type="button" class="bt-option active" data-option="capsule">荚膜</button>
        <button type="button" class="bt-option active" data-option="flagellum">鞭毛</button>
        <button type="button" class="bt-option active" data-option="plasmid">质粒</button>
        <button type="button" class="bt-option active" data-option="labels">结构标签</button>
      </div>

      <div class="bt-result" data-result></div>
    </div>

    <div class="bt-stage">
      <svg viewBox="0 0 680 414" aria-label="细菌结构与形态互动示意图">
        <defs>
          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="6"
              stdDeviation="7"
              flood-color="#4C1D95"
              flood-opacity=".15"
            />
          </filter>

          <radialGradient id="${rootId}-cytoplasm" cx="35%" cy="28%" r="75%">
            <stop offset="0%" stop-color="#F5F3FF"/>
            <stop offset="100%" stop-color="#DDD6FE"/>
          </radialGradient>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text
          x="28"
          y="39"
          data-title
          font-size="27"
          font-weight="900"
          fill="#5B21B6"
        ></text>

        <text
          x="28"
          y="69"
          data-summary
          font-size="15"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-bacterium filter="url(#${rootId}-shadow)"></g>
        <g data-labels></g>

        <g transform="translate(28 368)">
          <circle cx="7" cy="7" r="7" fill="#7C3AED"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            拟核DNA
          </text>
        </g>

        <g transform="translate(158 368)">
          <circle cx="7" cy="7" r="7" fill="#F59E0B"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            核糖体
          </text>
        </g>

        <g transform="translate(278 368)">
          <circle cx="7" cy="7" r="7" fill="#10B981"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            细胞膜
          </text>
        </g>

        <g transform="translate(398 368)">
          <circle cx="7" cy="7" r="7" fill="#2563EB"/>
          <text x="23" y="12" font-size="13" font-weight="800" fill="#475569">
            细胞壁
          </text>
        </g>

        <text
          x="528"
          y="380"
          data-magnification-note
          font-size="14"
          font-weight="900"
          fill="#6D28D9"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var mag=root.querySelector('[data-mag]');
    var ribosome=root.querySelector('[data-ribosome]');
    var flagellumActivity=root.querySelector('[data-flagellum]');

    var magValue=root.querySelector('[data-mag-value]');
    var ribosomeValue=root.querySelector('[data-ribosome-value]');
    var flagellumValue=root.querySelector('[data-flagellum-value]');

    var shapeButtons=root.querySelectorAll('[data-shape]');
    var optionButtons=root.querySelectorAll('[data-option]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var bacterium=root.querySelector('[data-bacterium]');
    var labels=root.querySelector('[data-labels]');
    var magnificationNote=root.querySelector('[data-magnification-note]');
    var result=root.querySelector('[data-result]');

    var shape='coccus';

    var options={
      capsule:true,
      flagellum:true,
      plasmid:true,
      labels:true
    };

    var shapeInfo={
      coccus:{
        name:'球菌',
        summary:'细胞外形近似球形，可单独存在或形成成对、链状、簇状排列'
      },
      bacillus:{
        name:'杆菌',
        summary:'细胞外形呈杆状，是常见的细菌形态之一'
      },
      spirillum:{
        name:'螺旋菌',
        summary:'细胞外形弯曲或呈螺旋状，形态有助于分类和观察'
      }
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function baseShape(kind,scale){
      if(kind==='coccus'){
        return '<ellipse cx="340" cy="220" rx="'+(112*scale)
          +'" ry="'+(112*scale)+'"/>';
      }

      if(kind==='bacillus'){
        return '<rect x="'+(210-(scale-1)*70)+'" y="'+(142-(scale-1)*40)
          +'" width="'+(260*scale)+'" height="'+(156*scale)
          +'" rx="'+(78*scale)+'"/>';
      }

      return '<path d="M190 240 C235 130 300 315 350 198 C400 83 464 275 505 169'
        +'" fill="none" stroke-width="'+(105*scale)+'" stroke-linecap="round"/>';
    }

    function insidePoint(kind,index,count){
      var angle=index*2.399;
      var radius=18+(index%7)*14;

      if(kind==='coccus'){
        return [
          340+Math.cos(angle)*Math.min(radius,82),
          220+Math.sin(angle)*Math.min(radius,82)
        ];
      }

      if(kind==='bacillus'){
        return [
          250+(index*47)%180,
          170+(index*31)%105
        ];
      }

      return [
        225+(index*43)%245,
        215+Math.sin(index*.85)*52
      ];
    }

    function labelLine(x1,y1,x2,y2,text,color){
      return '<path d="M'+x1+' '+y1+' L'+x2+' '+y2
        +'" stroke="'+color+'" stroke-width="2.5"/>'
        +'<circle cx="'+x1+'" cy="'+y1+'" r="4" fill="'+color+'"/>'
        +'<rect x="'+(x2-8)+'" y="'+(y2-15)+'" width="92" height="25'
        +'" rx="12" fill="#FFFFFF" stroke="'+color+'" stroke-width="2"/>'
        +'<text x="'+(x2+38)+'" y="'+(y2+2)
        +'" text-anchor="middle" font-size="12" font-weight="900" fill="'+color+'">'
        +text+'</text>';
    }

    function update(){
      var magnification=Number(mag.value);
      var ribosomeLevel=Number(ribosome.value);
      var activity=Number(flagellumActivity.value);
      var scale=.82+magnification/5000;

      magValue.textContent=magnification.toFixed(0)+'×';
      ribosomeValue.textContent=ribosomeLevel.toFixed(0)+'%';
      flagellumValue.textContent=activity.toFixed(0)+'%';

      root.style.setProperty(
        '--bt-speed',
        clamp(2.8-activity/55,.55,2.7).toFixed(2)+'s'
      );

      for(var i=0;i<shapeButtons.length;i++){
        shapeButtons[i].classList.toggle(
          'active',
          shapeButtons[i].getAttribute('data-shape')===shape
        );
      }

      for(var j=0;j<optionButtons.length;j++){
        var key=optionButtons[j].getAttribute('data-option');
        optionButtons[j].classList.toggle('active',options[key]);
      }

      title.textContent=shapeInfo[shape].name+'结构示意';
      summary.textContent=shapeInfo[shape].summary;
      magnificationNote.textContent=magnification.toFixed(0)+'× 观察';

      var html='';

      if(options.capsule){
        if(shape==='spirillum'){
          html+=baseShape(shape,scale)
            .replace('fill="none"', 'fill="none" stroke="#DDD6FE"')
            .replace('stroke-width="', 'opacity=".62" stroke-width="');
        }else{
          html+=baseShape(shape,scale*1.12)
            .replace('/>', ' fill="#EDE9FE" stroke="#C4B5FD" stroke-width="4" opacity=".72"/>');
        }
      }

      if(shape==='spirillum'){
        html+=baseShape(shape,scale)
          .replace(
            'fill="none"',
            'fill="none" stroke="#2563EB"'
          );

        html+=baseShape(shape,scale*.82)
          .replace(
            'fill="none"',
            'fill="none" stroke="#10B981"'
          )
          .replace(
            'stroke-width="',
            'opacity=".7" stroke-width="'
          );
      }else{
        html+=baseShape(shape,scale)
          .replace(
            '/>',
            ' fill="#DBEAFE" stroke="#2563EB" stroke-width="6"/>'
          );

        html+=baseShape(shape,scale*.91)
          .replace(
            '/>',
            ' fill="url(#${rootId}-cytoplasm)" stroke="#10B981" stroke-width="5"/>'
          );
      }

      html+='<path d="M270 213 C292 158 345 275 405 195 C433 158 443 224 410 254'
        +'" fill="none" stroke="#7C3AED" stroke-width="8" stroke-linecap="round"/>';

      html+='<circle cx="338" cy="220" r="7" fill="#FDE68A" stroke="#B45309" stroke-width="2"/>';

      var ribosomeCount=Math.floor(8+ribosomeLevel/5);

      for(var r=0;r<ribosomeCount;r++){
        var point=insidePoint(shape,r,ribosomeCount);

        html+='<circle cx="'+point[0]+'" cy="'+point[1]
          +'" r="'+(3+r%2)+'" fill="#F59E0B" opacity=".86"/>';
      }

      if(options.plasmid){
        html+='<ellipse cx="385" cy="254" rx="22" ry="14'
          +'" fill="none" stroke="#EC4899" stroke-width="5"/>';

        html+='<ellipse cx="300" cy="188" rx="16" ry="10'
          +'" fill="none" stroke="#EC4899" stroke-width="4"/>';
      }

      if(options.flagellum){
        html+='<path class="bt-flagellum" d="M445 226 C505 185 520 280 582 228 C614 201 628 236 648 224'
          +'" fill="none" stroke="#8B5CF6" stroke-width="6" stroke-linecap="round" opacity="'
          +(.25+.75*activity/100)+'"/>';
      }

      bacterium.innerHTML=html;

      var labelHTML='';

      if(options.labels){
        labelHTML+=labelLine(340,196,485,102,'拟核DNA','#7C3AED');
        labelHTML+=labelLine(301,252,96,309,'核糖体','#D97706');
        labelHTML+=labelLine(276,170,90,104,'细胞膜','#059669');
        labelHTML+=labelLine(423,176,505,160,'细胞壁','#2563EB');

        if(options.capsule){
          labelHTML+=labelLine(238,226,80,212,'荚膜','#8B5CF6');
        }

        if(options.plasmid){
          labelHTML+=labelLine(385,254,512,302,'质粒','#DB2777');
        }

        if(options.flagellum){
          labelHTML+=labelLine(567,229,554,88,'鞭毛','#6D28D9');
        }
      }

      labels.innerHTML=labelHTML;

      var optional=[];

      if(options.capsule)optional.push('荚膜');
      if(options.flagellum)optional.push('鞭毛');
      if(options.plasmid)optional.push('质粒');

      result.innerHTML='细菌没有由核膜包围的细胞核，DNA主要位于拟核区域。'
        +'<br>细胞膜、细胞壁和核糖体是常见结构。'
        +(optional.length
          ?' 当前另外显示：'+optional.join('、')+'。这些结构并非所有细菌都具有。'
          :' 当前隐藏了荚膜、鞭毛和质粒等可选结构。');
    }

    for(var i=0;i<shapeButtons.length;i++){
      shapeButtons[i].onclick=function(){
        shape=this.getAttribute('data-shape');
        update();
      };
    }

    for(var j=0;j<optionButtons.length;j++){
      optionButtons[j].onclick=function(){
        var key=this.getAttribute('data-option');
        options[key]=!options[key];
        update();
      };
    }

    mag.oninput=update;
    ribosome.oninput=update;
    flagellumActivity.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
