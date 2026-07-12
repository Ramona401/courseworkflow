/**
 * geographyLabTemplatesRegionCoordination.ts
 *
 * 地理第40批B3：区域发展差异、区域联系与协调发展。
 *
 * 教学目标：
 * - 理解区域发展水平差异及其形成条件；
 * - 观察交通、产业互补、要素流动和公共服务对区域联系的影响；
 * - 比较产业协作、基础设施互联、公共服务共享和生态补偿等协调路径；
 * - 建立优势互补、利益共享、生态共保和基本公共服务均等化意识。
 *
 * 教学边界：
 * - 所有区域、差距、要素流、收益和治理指标均为课堂简化示意；
 * - 不对应任何真实行政区、城市群、产业园区、企业或统计数据；
 * - 不用于真实区域规划、产业投资、财政分配或公共政策决策。
 */

import type {
  GeographyLabParamValue,
  GeographyLabTemplate,
} from './geographyLabUtils'

const SCRIPT_END = '</' + 'script>'

function numberValue(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: number,
): number {
  const value = Number(params[key])
  return Number.isFinite(value) ? value : fallback
}

function booleanValue(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]
  return typeof value === 'boolean' ? value : fallback
}

function stringValue(
  params: Record<string, GeographyLabParamValue>,
  key: string,
  fallback: string,
): string {
  const value = params[key]
  return typeof value === 'string' ? value : fallback
}

function buildRegionCoordinationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'core-periphery',
    'east-west',
    'urban-rural',
    'regional-cooperation',
    'balanced-development',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'regional-cooperation',
  )

  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'regional-cooperation'

  const developmentGap = Math.max(
    0,
    Math.min(10, numberValue(params, 'developmentGap', 6)),
  )

  const transportConnectivity = Math.max(
    0,
    Math.min(10, numberValue(params, 'transportConnectivity', 7)),
  )

  const industryComplementarity = Math.max(
    0,
    Math.min(10, numberValue(params, 'industryComplementarity', 7)),
  )

  const publicService = Math.max(
    0,
    Math.min(10, numberValue(params, 'publicService', 6)),
  )

  const ecologicalConstraint = Math.max(
    0,
    Math.min(10, numberValue(params, 'ecologicalConstraint', 6)),
  )

  const coordinationMechanism = Math.max(
    0,
    Math.min(10, numberValue(params, 'coordinationMechanism', 7)),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-region-coordination-root">
  <style>
    #${rootId}{width:100%;height:100%;overflow:hidden;box-sizing:border-box;border:1px solid #FDBA74;border-radius:18px;background:#fff;color:#431407;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;box-shadow:0 12px 34px rgba(194,65,12,.11)}
    #${rootId} *{box-sizing:border-box}
    #${rootId} .gl-head{height:56px;padding:0 18px;display:flex;align-items:center;gap:12px;border-bottom:1px solid #FED7AA;background:linear-gradient(135deg,#FFF7ED,#FFFBEB 56%,#F0FDF4)}
    #${rootId} .gl-title{color:#9A3412;font-size:16px;font-weight:880}
    #${rootId} .gl-subtitle{margin-top:2px;color:#64748B;font-size:11px}
    #${rootId} .gl-note{margin-left:auto;padding:5px 10px;border:1px solid #FDBA74;border-radius:999px;background:#fff;color:#C2410C;font-size:11px;font-weight:750;white-space:nowrap}
    #${rootId} .gl-body{height:calc(100% - 56px);display:grid;grid-template-columns:284px minmax(0,1fr)}
    #${rootId} .gl-controls{min-height:0;padding:13px;overflow:auto;border-right:1px solid #FFEDD5;background:linear-gradient(180deg,#FFF7ED,#FFFBEB 62%,#F0FDF4)}
    #${rootId} .gl-stage{min-width:0;min-height:0;display:grid;grid-template-rows:46px minmax(0,1fr);padding:8px;background:radial-gradient(circle at 48% 22%,#fff 0%,#F8FAFC 62%,#FFEDD5 100%)}
    #${rootId} .gl-section-title{margin:1px 0 8px;color:#9A3412;font-size:11.5px;font-weight:850}
    #${rootId} .gl-scenario-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:12px}
    #${rootId} .gl-scenario-grid button:last-child{grid-column:1/-1}
    #${rootId} .gl-row{margin-bottom:10px}
    #${rootId} .gl-label-line{display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:5px}
    #${rootId} .gl-label{color:#334155;font-size:11.4px;font-weight:730}
    #${rootId} .gl-value{min-width:44px;padding:3px 7px;border-radius:999px;background:#FFEDD5;color:#C2410C;font-size:11px;font-weight:850;text-align:center}
    #${rootId} input[type=range]{width:100%;height:6px;margin:0;appearance:none;border-radius:999px;outline:none;background:linear-gradient(90deg,#FDBA74,#FDE68A,#86EFAC);cursor:pointer}
    #${rootId} input[type=range]::-webkit-slider-thumb{width:16px;height:16px;appearance:none;border:2px solid #fff;border-radius:50%;background:linear-gradient(135deg,#EA580C,#16A34A);box-shadow:0 1px 5px rgba(194,65,12,.42)}
    #${rootId} button{min-height:32px;padding:6px 7px;border:1px solid #FDBA74;border-radius:9px;background:#fff;color:#C2410C;font-size:10.6px;font-weight:790;cursor:pointer}
    #${rootId} button[data-active="true"]{border-color:#C2410C;color:#fff;background:linear-gradient(135deg,#EA580C,#D97706 55%,#16A34A);box-shadow:0 5px 13px rgba(194,65,12,.22)}
    #${rootId} .gl-action-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin:10px 0}
    #${rootId} .gl-result{margin-top:8px;padding:10px;border:1px solid #FDBA74;border-radius:12px;background:linear-gradient(135deg,#FFF7ED,#F0FDF4);color:#334155;font-size:11.1px;font-weight:620;line-height:1.5}
    #${rootId} .gl-view-toolbar{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:7px;align-items:center;padding:0 3px 7px;border-bottom:1px solid #E2E8F0}
    #${rootId} .gl-view-toolbar button{min-height:32px;font-size:11px}
    #${rootId} .gl-canvas-wrap{min-width:0;min-height:0;overflow:hidden;border:1px solid #FED7AA;border-radius:14px;background:#fff}
    #${rootId} .gl-coordination-canvas{width:100%;height:100%;display:block}
    @media(max-width:900px){#${rootId} .gl-body{grid-template-columns:242px minmax(0,1fr)}#${rootId} .gl-note{display:none}}
  </style>

  <div class="gl-head">
    <div style="font-size:24px;">🤝</div>
    <div>
      <div class="gl-title">区域发展差异、区域联系与协调发展</div>
      <div class="gl-subtitle">比较发展差距、要素流动、产业互补、公共服务和生态补偿</div>
    </div>
    <div class="gl-note">课堂简化模型 · 不用于真实区域规划</div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">区域发展情境</div>
      <div class="gl-scenario-grid">
        <button type="button" data-scenario="core-periphery">核心—外围</button>
        <button type="button" data-scenario="east-west">东西部协作</button>
        <button type="button" data-scenario="urban-rural">城乡融合</button>
        <button type="button" data-scenario="regional-cooperation">跨区域合作</button>
        <button type="button" data-scenario="balanced-development">协调发展方案</button>
      </div>

      <div class="gl-section-title">区域联系与协调参数</div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">初始发展差距</span><span class="gl-value" data-role="gap-value">6</span></div>
        <input type="range" min="0" max="10" step="1" value="${developmentGap}" data-role="gap" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">交通互联水平</span><span class="gl-value" data-role="transport-value">7</span></div>
        <input type="range" min="0" max="10" step="1" value="${transportConnectivity}" data-role="transport" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">产业互补程度</span><span class="gl-value" data-role="industry-value">7</span></div>
        <input type="range" min="0" max="10" step="1" value="${industryComplementarity}" data-role="industry" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">公共服务均等化</span><span class="gl-value" data-role="service-value">6</span></div>
        <input type="range" min="0" max="10" step="1" value="${publicService}" data-role="service" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">生态约束强度</span><span class="gl-value" data-role="ecology-value">6</span></div>
        <input type="range" min="0" max="10" step="1" value="${ecologicalConstraint}" data-role="ecology" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">协调机制水平</span><span class="gl-value" data-role="coordination-value">7</span></div>
        <input type="range" min="0" max="10" step="1" value="${coordinationMechanism}" data-role="coordination" />
      </div>

      <div class="gl-action-grid">
        <button type="button" data-role="label-toggle" data-active="${showLabels}">要素标注</button>
        <button type="button" data-role="auto-toggle" data-active="false">自动演示</button>
        <button type="button" data-role="reset">恢复初始</button>
        <button type="button" data-role="compare">切换下一情境</button>
      </div>

      <div class="gl-result" data-role="result">区域协调发展需要在优势互补、利益共享、公共服务和生态保护之间建立机制。</div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="difference">区域差异</button>
        <button type="button" data-view="linkage">区域联系</button>
        <button type="button" data-view="strategy">协调路径</button>
        <button type="button" data-view="governance">治理成效</button>
      </div>
      <div class="gl-canvas-wrap">
        <canvas class="gl-coordination-canvas" width="980" height="570" data-role="canvas" aria-label="区域发展差异、区域联系与协调发展教学示意图"></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root=document.getElementById('${rootId}');
      if(!root)return;

      var gapInput=root.querySelector('[data-role="gap"]');
      var transportInput=root.querySelector('[data-role="transport"]');
      var industryInput=root.querySelector('[data-role="industry"]');
      var serviceInput=root.querySelector('[data-role="service"]');
      var ecologyInput=root.querySelector('[data-role="ecology"]');
      var coordinationInput=root.querySelector('[data-role="coordination"]');
      var gapValue=root.querySelector('[data-role="gap-value"]');
      var transportValue=root.querySelector('[data-role="transport-value"]');
      var industryValue=root.querySelector('[data-role="industry-value"]');
      var serviceValue=root.querySelector('[data-role="service-value"]');
      var ecologyValue=root.querySelector('[data-role="ecology-value"]');
      var coordinationValue=root.querySelector('[data-role="coordination-value"]');
      var scenarioButtons=root.querySelectorAll('[data-scenario]');
      var viewButtons=root.querySelectorAll('[data-view]');
      var labelToggle=root.querySelector('[data-role="label-toggle"]');
      var autoToggle=root.querySelector('[data-role="auto-toggle"]');
      var resetButton=root.querySelector('[data-role="reset"]');
      var compareButton=root.querySelector('[data-role="compare"]');
      var result=root.querySelector('[data-role="result"]');
      var canvas=root.querySelector('[data-role="canvas"]');

      if(!gapInput||!transportInput||!industryInput||!serviceInput||!ecologyInput||!coordinationInput||!gapValue||!transportValue||!industryValue||!serviceValue||!ecologyValue||!coordinationValue||!scenarioButtons.length||!viewButtons.length||!labelToggle||!autoToggle||!resetButton||!compareButton||!result||!canvas)return;

      var context=canvas.getContext('2d');
      if(!context)return;

      var width=canvas.width;
      var height=canvas.height;
      var scenarios=[
        {key:'core-periphery',name:'核心—外围',gap:9,transport:5,industry:4,service:4,ecology:5,coordination:3,view:'difference',color:'#DC2626'},
        {key:'east-west',name:'东西部协作',gap:8,transport:6,industry:8,service:5,ecology:7,coordination:6,view:'linkage',color:'#7C3AED'},
        {key:'urban-rural',name:'城乡融合',gap:7,transport:7,industry:6,service:8,ecology:6,coordination:7,view:'strategy',color:'#2563EB'},
        {key:'regional-cooperation',name:'跨区域合作',gap:6,transport:8,industry:8,service:6,ecology:7,coordination:8,view:'linkage',color:'#EA580C'},
        {key:'balanced-development',name:'协调发展方案',gap:4,transport:8,industry:8,service:8,ecology:8,coordination:9,view:'governance',color:'#16A34A'}
      ];
      var initial={scenario:'${scenario}',gap:${developmentGap},transport:${transportConnectivity},industry:${industryComplementarity},service:${publicService},ecology:${ecologicalConstraint},coordination:${coordinationMechanism},showLabels:${showLabels}};
      var state={scenario:initial.scenario,view:'difference',showLabels:initial.showLabels,auto:false,startedAt:0,raf:0,phase:0,compareIndex:0};

      function clamp(value,min,max){return Math.max(min,Math.min(max,value));}
      function lerp(a,b,t){return a+(b-a)*t;}
      function ease(t){var p=clamp(t,0,1);return p<.5?2*p*p:1-Math.pow(-2*p+2,2)/2;}
      function roundRect(x,y,w,h,r){var q=Math.min(r,w/2,h/2);context.beginPath();context.moveTo(x+q,y);context.lineTo(x+w-q,y);context.quadraticCurveTo(x+w,y,x+w,y+q);context.lineTo(x+w,y+h-q);context.quadraticCurveTo(x+w,y+h,x+w-q,y+h);context.lineTo(x+q,y+h);context.quadraticCurveTo(x,y+h,x,y+h-q);context.lineTo(x,y+q);context.quadraticCurveTo(x,y,x+q,y);context.closePath();}
      function box(x,y,w,h,r,fill,stroke){roundRect(x,y,w,h,r);if(fill){context.fillStyle=fill;context.fill();}if(stroke){context.strokeStyle=stroke;context.lineWidth=1.2;context.stroke();}}
      function text(value,x,y,size,color,weight,align){context.save();context.font=(weight||600)+' '+size+'px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif';context.fillStyle=color||'#334155';context.textAlign=align||'left';context.textBaseline='middle';context.fillText(String(value),x,y);context.restore();}
      function line(x1,y1,x2,y2,color,lineWidth,dash){context.save();context.strokeStyle=color;context.lineWidth=lineWidth||1.5;context.setLineDash(dash||[]);context.beginPath();context.moveTo(x1,y1);context.lineTo(x2,y2);context.stroke();context.restore();}
      function circle(x,y,r,fill,stroke){context.beginPath();context.arc(x,y,r,0,Math.PI*2);if(fill){context.fillStyle=fill;context.fill();}if(stroke){context.strokeStyle=stroke;context.lineWidth=2;context.stroke();}}
      function arrow(x1,y1,x2,y2,color,lineWidth){var angle=Math.atan2(y2-y1,x2-x1);var head=12;context.save();context.strokeStyle=color;context.fillStyle=color;context.lineWidth=lineWidth||4;context.lineCap='round';context.beginPath();context.moveTo(x1,y1);context.lineTo(x2,y2);context.stroke();context.beginPath();context.moveTo(x2,y2);context.lineTo(x2-head*Math.cos(angle-Math.PI/6),y2-head*Math.sin(angle-Math.PI/6));context.lineTo(x2-head*Math.cos(angle+Math.PI/6),y2-head*Math.sin(angle+Math.PI/6));context.closePath();context.fill();context.restore();}
      function scenarioByKey(key){var found=scenarios[3];scenarios.forEach(function(item){if(item.key===key)found=item;});return found;}
      function values(){return {gap:clamp(Number(gapInput.value)||0,0,10),transport:clamp(Number(transportInput.value)||0,0,10),industry:clamp(Number(industryInput.value)||0,0,10),service:clamp(Number(serviceInput.value)||0,0,10),ecology:clamp(Number(ecologyInput.value)||0,0,10),coordination:clamp(Number(coordinationInput.value)||0,0,10)};}
      function setInputs(value){gapInput.value=String(Math.round(value.gap));transportInput.value=String(Math.round(value.transport));industryInput.value=String(Math.round(value.industry));serviceInput.value=String(Math.round(value.service));ecologyInput.value=String(Math.round(value.ecology));coordinationInput.value=String(Math.round(value.coordination));}

      function derive(value){
        var linkage=clamp(Math.round(value.transport*.36+value.industry*.30+value.coordination*.24+value.service*.10),0,10);
        var factorFlow=clamp(Math.round(value.transport*.30+value.industry*.26+value.coordination*.24+(10-value.gap)*.20),0,10);
        var cooperationBenefit=clamp(Math.round(value.industry*.30+value.transport*.20+value.coordination*.28+value.service*.12+value.ecology*.10),0,10);
        var remainingGap=clamp(Math.round(value.gap*.72-(value.transport+value.industry+value.service+value.coordination)*.09),0,10);
        var ecologicalPressure=clamp(Math.round((10-value.ecology)*.38+value.industry*.18+value.transport*.12+value.gap*.10-value.coordination*.12),0,10);
        var inclusion=clamp(Math.round(value.service*.34+value.coordination*.28+(10-remainingGap)*.22+value.transport*.16),0,10);
        var coordinationIndex=clamp(Math.round(cooperationBenefit*.30+inclusion*.25+(10-ecologicalPressure)*.20+(10-remainingGap)*.25),0,10);
        return {linkage:linkage,factorFlow:factorFlow,cooperationBenefit:cooperationBenefit,remainingGap:remainingGap,ecologicalPressure:ecologicalPressure,inclusion:inclusion,coordinationIndex:coordinationIndex};
      }

      function background(titleValue,subtitle){var gradient=context.createLinearGradient(0,0,width,height);gradient.addColorStop(0,'#FFFFFF');gradient.addColorStop(.58,'#F8FAFC');gradient.addColorStop(1,'#FFEDD5');context.fillStyle=gradient;context.fillRect(0,0,width,height);text(titleValue,28,31,18,'#9A3412',880,'left');text(subtitle,28,55,11.5,'#64748B',620,'left');}
      function card(x,y,w,label,value,color,desc){box(x,y,w,72,12,'rgba(255,255,255,.94)','#FED7AA');text(label,x+14,y+17,10.5,'#64748B',720,'left');text(value,x+14,y+40,20,color,880,'left');text(desc,x+14,y+59,9.5,'#64748B',600,'left');}
      function gauge(cx,cy,ratio,color,label,valueText){context.save();context.lineWidth=14;context.strokeStyle='#E2E8F0';context.beginPath();context.arc(cx,cy,78,Math.PI,Math.PI*2);context.stroke();context.strokeStyle=color;context.lineCap='round';context.beginPath();context.arc(cx,cy,78,Math.PI,Math.PI+Math.PI*clamp(ratio,0,1));context.stroke();context.restore();text(valueText,cx,cy-4,18,color,880,'center');text(label,cx,cy+24,9.5,'#64748B',720,'center');}

      function differenceView(item,value,derived){
        background('区域发展差异','比较核心区与相对薄弱地区的发展基础、公共服务和生态条件。');
        card(28,78,210,'初始发展差距',Math.round(value.gap),'#DC2626','产业、收入和设施综合示意');
        card(250,78,210,'剩余发展差距',derived.remainingGap,'#EA580C','协调作用后的课堂结果');
        card(472,78,210,'公共服务水平',Math.round(value.service),'#2563EB','教育医疗等综合值');
        card(694,78,258,'协调发展指数',derived.coordinationIndex,item.color,'综合评价');

        var leftX=95,rightX=535,top=190,panelW=350,panelH=292;
        box(leftX,top,panelW,panelH,18,'#FFFFFF','#DC2626');
        box(rightX,top,panelW,panelH,18,'#FFFFFF','#16A34A');
        text('核心或先发地区',leftX+panelW/2,top+28,14,'#B91C1C',860,'center');
        text('外围或后发地区',rightX+panelW/2,top+28,14,'#15803D',860,'center');

        var indicators=[['产业基础',9-value.gap*.18,5+value.industry*.32],['交通可达性',8+value.transport*.12,3+value.transport*.42],['公共服务',8+value.service*.12,3+value.service*.44],['创新与人才',8+value.industry*.10,3+value.coordination*.34],['生态空间',5+value.ecology*.24,7+value.ecology*.16]];
        indicators.forEach(function(indicator,index){var y=top+72+index*42;var leftScore=clamp(indicator[1],0,10);var rightScore=clamp(indicator[2],0,10);text(indicator[0],leftX+18,y,10,'#475569',700,'left');box(leftX+118,y-7,200,14,7,'#E2E8F0',null);box(leftX+118,y-7,200*leftScore/10,14,7,'#DC2626',null);text(leftScore.toFixed(1),leftX+330,y,9.5,'#B91C1C',800,'right');text(indicator[0],rightX+18,y,10,'#475569',700,'left');box(rightX+118,y-7,200,14,7,'#E2E8F0',null);box(rightX+118,y-7,200*rightScore/10,14,7,'#16A34A',null);text(rightScore.toFixed(1),rightX+330,y,9.5,'#15803D',800,'right');});
        arrow(leftX+panelW+18,top+145,rightX-18,top+145,item.color,2+derived.factorFlow*.35);
        arrow(rightX-18,top+205,leftX+panelW+18,top+205,'#0F766E',2+value.coordination*.28);
        if(state.showLabels){text('资本、技术、产业与信息扩散',490,top+126,10,item.color,800,'center');text('资源、劳动力、生态产品与市场联系',490,top+226,10,'#0F766E',800,'center');}
        box(95,505,790,38,12,'#FFFFFF','#FED7AA');text('区域协调不是消除所有差异，而是提高机会公平、基本公共服务和区域整体效率。',490,524,10.2,'#475569',680,'center');
      }

      function regionNode(x,y,r,titleValue,sub,color){circle(x,y,r,'#FFFFFF',color);text(titleValue,x,y-9,11,color,850,'center');text(sub,x,y+14,9,'#64748B',650,'center');}

      function linkageView(item,value,derived){
        background('区域联系与要素流动','交通网络和产业互补促进劳动力、资本、技术、产品和信息跨区域流动。');
        card(28,78,210,'区域联系强度',derived.linkage,item.color,'交通、产业和机制综合');
        card(250,78,210,'要素流动指数',derived.factorFlow,'#7C3AED','多种要素流的课堂值');
        card(472,78,210,'产业互补程度',Math.round(value.industry),'#16A34A','专业化分工与协作');
        card(694,78,258,'合作收益',derived.cooperationBenefit,'#EA580C','优势互补与规模效应');

        var centerX=490,centerY=350;
        regionNode(centerX,centerY,72,'协同平台','标准·信息·利益协调',item.color);
        var regions=[{x:175,y:235,title:'区域A',sub:'资本与技术',color:'#DC2626'},{x:805,y:235,title:'区域B',sub:'资源与空间',color:'#16A34A'},{x:175,y:465,title:'区域C',sub:'劳动力与市场',color:'#2563EB'},{x:805,y:465,title:'区域D',sub:'生态产品与服务',color:'#0F766E'}];
        regions.forEach(function(region,index){regionNode(region.x,region.y,58,region.title,region.sub,region.color);arrow(region.x+(region.x<centerX?62:-62),region.y,centerX+(region.x<centerX?-78:78),centerY+(index<2?-24:24),region.color,2+derived.linkage*.30);});
        var flowLabels=['技术流','产品流','劳动力流','生态补偿'];
        if(state.showLabels){regions.forEach(function(region,index){text(flowLabels[index],(region.x+centerX)/2,(region.y+centerY)/2-16,9.5,region.color,800,'center');});}
        for(var p=0;p<8;p+=1){var angle=state.phase*Math.PI*2+p*Math.PI/4;circle(centerX+Math.cos(angle)*112,centerY+Math.sin(angle)*112,4,'#FFFFFF',item.color);}
        box(70,522,840,30,10,'#FFFFFF','#FED7AA');text('要素流动需要双向联系和利益共享，单向虹吸可能扩大区域差距。',490,537,10,'#475569',680,'center');
      }

      function strategyCard(x,y,w,titleValue,desc,color,score){box(x,y,w,92,14,'#FFFFFF',color);text(titleValue,x+16,y+23,11.5,color,850,'left');text(desc,x+16,y+49,9.4,'#64748B',620,'left');box(x+16,y+70,w-32,9,5,'#E2E8F0',null);box(x+16,y+70,(w-32)*clamp(score/10,0,1),9,5,color,null);}

      function strategyView(item,value,derived){
        background('区域协调发展的主要路径','基础设施、产业协作、公共服务和生态共保需要组合推进。');
        card(28,78,210,'交通互联',Math.round(value.transport),'#2563EB','缩短时空距离');
        card(250,78,210,'产业互补',Math.round(value.industry),'#7C3AED','专业化分工与协作');
        card(472,78,210,'公共服务',Math.round(value.service),'#EA580C','机会公平和人口发展');
        card(694,78,258,'生态约束',Math.round(value.ecology),'#16A34A','底线与生态产品价值');
        strategyCard(58,188,410,'交通基础设施互联','打通干线、支线和数字信息网络。','#2563EB',value.transport);
        strategyCard(512,188,410,'产业分工与协作','避免简单同质竞争，形成优势互补产业链。','#7C3AED',value.industry);
        strategyCard(58,304,410,'公共服务共建共享','教育、医疗和就业服务向薄弱地区延伸。','#EA580C',value.service);
        strategyCard(512,304,410,'生态共保与补偿','保护重要生态空间并建立跨区域补偿机制。','#16A34A',(value.ecology+value.coordination)/2);
        strategyCard(58,420,410,'统一市场与制度协调','降低行政分割，促进要素合理流动。','#0F766E',value.coordination);
        strategyCard(512,420,410,'防止单向虹吸','支持后发地区形成内生发展能力。','#DC2626',(10-derived.remainingGap+value.service)/2);
        box(58,526,864,28,10,'#FFFFFF','#FED7AA');text('协调发展需要长期机制，而不是短期项目叠加；既要提高整体效率，也要关注公平与生态。',490,540,10,'#475569',680,'center');
      }

      function governanceView(item,value,derived){
        background('区域协调治理成效','综合观察差距缩小、合作收益、公共服务、生态风险和协调机制。');
        card(28,78,210,'剩余发展差距',derived.remainingGap,'#DC2626','低值表示差距较小');
        card(250,78,210,'合作收益',derived.cooperationBenefit,item.color,'产业交通与机制综合');
        card(472,78,210,'包容发展',derived.inclusion,'#2563EB','公共服务和机会公平');
        card(694,78,258,'协调发展指数',derived.coordinationIndex,'#16A34A','综合治理成效');
        gauge(190,322,(10-derived.remainingGap)/10,'#EA580C','差距改善',10-derived.remainingGap);
        gauge(490,322,derived.inclusion/10,'#2563EB','包容发展',derived.inclusion);
        gauge(790,322,derived.coordinationIndex/10,'#16A34A','协调指数',derived.coordinationIndex);
        var measures=[{title:'制度协同',desc:'统一规则和利益协调',color:'#7C3AED',score:value.coordination},{title:'公共服务',desc:'基本服务共建共享',color:'#2563EB',score:value.service},{title:'生态共保',desc:'保护与补偿并重',color:'#16A34A',score:(value.ecology+value.coordination)/2},{title:'内生发展',desc:'增强薄弱地区能力',color:'#EA580C',score:(10-derived.remainingGap+value.industry)/2}];
        measures.forEach(function(measure,index){var x=44+index*228;box(x,430,204,82,14,'#FFFFFF',measure.color);text(measure.title,x+15,451,11,measure.color,850,'left');text(measure.desc,x+15,473,9.2,'#64748B',620,'left');box(x+15,490,174,9,5,'#E2E8F0',null);box(x+15,490,174*clamp(measure.score/10,0,1),9,5,measure.color,null);});
        box(44,526,888,28,10,'#FFFFFF','#FED7AA');text('协调发展的最终目标是形成优势互补、机会公平、生态安全和利益共享的区域共同体。',488,540,10,'#475569',680,'center');
      }

      function update(item,value,derived){
        gapValue.textContent=String(Math.round(value.gap));transportValue.textContent=String(Math.round(value.transport));industryValue.textContent=String(Math.round(value.industry));serviceValue.textContent=String(Math.round(value.service));ecologyValue.textContent=String(Math.round(value.ecology));coordinationValue.textContent=String(Math.round(value.coordination));
        Array.prototype.forEach.call(scenarioButtons,function(button){button.setAttribute('data-active',button.getAttribute('data-scenario')===state.scenario?'true':'false');});
        Array.prototype.forEach.call(viewButtons,function(button){button.setAttribute('data-active',button.getAttribute('data-view')===state.view?'true':'false');});
        labelToggle.setAttribute('data-active',state.showLabels?'true':'false');autoToggle.setAttribute('data-active',state.auto?'true':'false');
        var scenarioName=state.scenario==='custom'?'自定义区域条件':item.name;
        result.textContent=scenarioName+'下，区域联系强度为'+derived.linkage+'，合作收益为'+derived.cooperationBenefit+'，剩余发展差距为'+derived.remainingGap+'，包容发展为'+derived.inclusion+'，生态压力为'+derived.ecologicalPressure+'，协调发展指数为'+derived.coordinationIndex+'。';
      }

      function render(){if(!root.isConnected){state.auto=false;return;}var item=scenarioByKey(state.scenario);var value=values();var derived=derive(value);update(item,value,derived);context.clearRect(0,0,width,height);if(state.view==='difference')differenceView(item,value,derived);else if(state.view==='linkage')linkageView(item,value,derived);else if(state.view==='strategy')strategyView(item,value,derived);else governanceView(item,value,derived);}
      function applyScenario(key,changeView){var item=scenarioByKey(key);state.scenario=item.key;setInputs(item);if(changeView)state.view=item.view;state.compareIndex=scenarios.indexOf(item);render();}
      function stopAuto(){state.auto=false;state.startedAt=0;if(state.raf){cancelAnimationFrame(state.raf);state.raf=0;}render();}
      function animate(timestamp){if(!root.isConnected){state.auto=false;return;}if(!state.auto)return;if(!state.startedAt)state.startedAt=timestamp;var elapsed=timestamp-state.startedAt;var duration=5200;var segment=Math.floor(elapsed/duration);var local=(elapsed%duration)/duration;var from=scenarios[segment%scenarios.length];var to=scenarios[(segment+1)%scenarios.length];var progress=ease(clamp(local/.82,0,1));state.scenario=local<.5?from.key:to.key;state.view=['difference','linkage','strategy','governance'][Math.floor(elapsed/6500)%4];state.phase=(elapsed/3300)%1;setInputs({gap:lerp(from.gap,to.gap,progress),transport:lerp(from.transport,to.transport,progress),industry:lerp(from.industry,to.industry,progress),service:lerp(from.service,to.service,progress),ecology:lerp(from.ecology,to.ecology,progress),coordination:lerp(from.coordination,to.coordination,progress)});render();state.raf=requestAnimationFrame(animate);}
      function manual(){if(state.auto)stopAuto();state.scenario='custom';render();}

      [gapInput,transportInput,industryInput,serviceInput,ecologyInput,coordinationInput].forEach(function(input){input.addEventListener('input',manual);});
      Array.prototype.forEach.call(scenarioButtons,function(button){button.addEventListener('click',function(){if(state.auto)stopAuto();applyScenario(button.getAttribute('data-scenario')||'regional-cooperation',true);});});
      Array.prototype.forEach.call(viewButtons,function(button){button.addEventListener('click',function(){if(state.auto)stopAuto();state.view=button.getAttribute('data-view')||'difference';render();});});
      labelToggle.addEventListener('click',function(){state.showLabels=!state.showLabels;render();});
      autoToggle.addEventListener('click',function(){if(state.auto){stopAuto();return;}state.auto=true;state.startedAt=0;state.raf=requestAnimationFrame(animate);render();});
      resetButton.addEventListener('click',function(){if(state.auto)stopAuto();state.scenario=initial.scenario;state.view='difference';state.showLabels=initial.showLabels;setInputs(initial);state.compareIndex=scenarios.indexOf(scenarioByKey(initial.scenario));render();});
      compareButton.addEventListener('click',function(){if(state.auto)stopAuto();state.compareIndex=(state.compareIndex+1)%scenarios.length;var next=scenarios[state.compareIndex];state.scenario=next.key;state.view=next.view;setInputs(next);render();});

      state.compareIndex=scenarios.indexOf(scenarioByKey(initial.scenario));
      setInputs(initial);
      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_REGION_COORDINATION:
GeographyLabTemplate[] = [
  {
    id: 'geography-regional-difference-linkage-coordinated-development',
    group: '🗺️ 区域发展与资源环境',
    name: '区域发展差异、区域联系与协调发展',
    emoji: '🤝',
    desc: '调节发展差距、交通互联、产业互补、公共服务、生态约束和协调机制，观察区域差异、要素流动、协调路径与治理成效。',
    params: [
      {
        key: 'scenario',
        label: '初始区域发展情境',
        type: 'select',
        options: [
          { label: '核心—外围', value: 'core-periphery' },
          { label: '东西部协作', value: 'east-west' },
          { label: '城乡融合', value: 'urban-rural' },
          { label: '跨区域合作', value: 'regional-cooperation' },
          { label: '协调发展方案', value: 'balanced-development' },
        ],
        defaultValue: 'regional-cooperation',
      },
      {
        key: 'developmentGap',
        label: '初始发展差距',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '综合表示产业、基础设施、收入和发展机会的区域差异。',
      },
      {
        key: 'transportConnectivity',
        label: '交通互联水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '交通与信息网络改善有助于要素流动和市场联系。',
      },
      {
        key: 'industryComplementarity',
        label: '产业互补程度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '优势互补和专业化协作通常比同质竞争更有利于区域合作。',
      },
      {
        key: 'publicService',
        label: '公共服务均等化',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '综合表示教育、医疗、就业和社会保障等基本服务共享水平。',
      },
      {
        key: 'ecologicalConstraint',
        label: '生态约束强度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 6,
        hint: '协调发展需要共同保护重要生态空间并控制环境压力。',
      },
      {
        key: 'coordinationMechanism',
        label: '协调机制水平',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '包括统一规则、信息共享、利益分配、生态补偿和争议协调。',
      },
      {
        key: 'showLabels',
        label: '显示区域与要素流标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildRegionCoordinationHTML,
  },
]
