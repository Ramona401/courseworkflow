/**
 * geographyLabTemplatesHumanUrbanization.ts
 *
 * 地理第38批B3：城市化、城市功能区与城市空间结构。
 *
 * 教学目标：
 * - 理解城市化率、交通、地租和环境质量之间的联系；
 * - 比较商业、居住、工业和公共服务功能区的区位条件；
 * - 比较同心圆、扇形和多核心三种城市空间结构简化模型；
 * - 观察郊区化、逆城市化、热岛、通勤和公共服务压力。
 *
 * 教学边界：
 * - 所有地图、地租、热岛、通勤时间和服务压力均为课堂简化示意；
 * - 功能区边界不代表真实城市规划，也不对应任何具体城市或群体；
 * - 不用于真实城市规划、房地产投资、交通导航或公共政策决策。
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

function buildHumanUrbanizationHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const scenarios = [
    'early-urbanization',
    'rapid-urbanization',
    'mature-urbanization',
    'suburbanization',
    'reverse-urbanization',
  ]

  const requestedScenario = stringValue(
    params,
    'scenario',
    'rapid-urbanization',
  )

  const scenario = scenarios.includes(requestedScenario)
    ? requestedScenario
    : 'rapid-urbanization'

  const urbanizationRate = Math.max(
    15,
    Math.min(95, numberValue(params, 'urbanizationRate', 62)),
  )

  const transportAccessibility = Math.max(
    0,
    Math.min(10, numberValue(params, 'transportAccessibility', 7)),
  )

  const environmentalQuality = Math.max(
    0,
    Math.min(10, numberValue(params, 'environmentalQuality', 5)),
  )

  const landRentGradient = Math.max(
    0,
    Math.min(10, numberValue(params, 'landRentGradient', 7)),
  )

  const showLabels = booleanValue(
    params,
    'showLabels',
    true,
  )

  return `
<div id="${rootId}" class="gl-human-urbanization-root">
  <style>
    #${rootId}{width:100%;height:100%;overflow:hidden;box-sizing:border-box;border:1px solid #FCD34D;border-radius:18px;background:#fff;color:#172554;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;box-shadow:0 12px 34px rgba(146,64,14,.11)}
    #${rootId} *{box-sizing:border-box}
    #${rootId} .gl-head{height:56px;padding:0 18px;display:flex;align-items:center;gap:12px;border-bottom:1px solid #FDE68A;background:linear-gradient(135deg,#FFFBEB,#FFF7ED 56%,#F0FDF4)}
    #${rootId} .gl-title{color:#92400E;font-size:16px;font-weight:880}
    #${rootId} .gl-subtitle{margin-top:2px;color:#64748B;font-size:11px}
    #${rootId} .gl-note{margin-left:auto;padding:5px 10px;border:1px solid #FCD34D;border-radius:999px;background:#fff;color:#B45309;font-size:11px;font-weight:750;white-space:nowrap}
    #${rootId} .gl-body{height:calc(100% - 56px);display:grid;grid-template-columns:280px minmax(0,1fr)}
    #${rootId} .gl-controls{min-height:0;padding:13px;overflow:auto;border-right:1px solid #FDE68A;background:linear-gradient(180deg,#FFFBEB,#FFF7ED 62%,#F0FDF4)}
    #${rootId} .gl-stage{min-width:0;min-height:0;display:grid;grid-template-rows:46px minmax(0,1fr);padding:8px;background:radial-gradient(circle at 48% 22%,#fff 0%,#F8FAFC 64%,#FEF3C7 100%)}
    #${rootId} .gl-section-title{margin:1px 0 8px;color:#92400E;font-size:11.5px;font-weight:850}
    #${rootId} .gl-scenario-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:12px}
    #${rootId} .gl-scenario-grid button:last-child{grid-column:1/-1}
    #${rootId} .gl-row{margin-bottom:10px}
    #${rootId} .gl-label-line{display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:5px}
    #${rootId} .gl-label{color:#334155;font-size:11.5px;font-weight:730}
    #${rootId} .gl-value{min-width:54px;padding:3px 7px;border-radius:999px;background:#FEF3C7;color:#B45309;font-size:11px;font-weight:850;text-align:center}
    #${rootId} input[type=range]{width:100%;height:6px;margin:0;appearance:none;border-radius:999px;outline:none;background:linear-gradient(90deg,#FCD34D,#86EFAC);cursor:pointer}
    #${rootId} input[type=range]::-webkit-slider-thumb{width:16px;height:16px;appearance:none;border:2px solid #fff;border-radius:50%;background:linear-gradient(135deg,#D97706,#16A34A);box-shadow:0 1px 5px rgba(146,64,14,.42)}
    #${rootId} button{min-height:32px;padding:6px 7px;border:1px solid #FCD34D;border-radius:9px;background:#fff;color:#B45309;font-size:10.7px;font-weight:790;cursor:pointer}
    #${rootId} button[data-active="true"]{border-color:#D97706;color:#fff;background:linear-gradient(135deg,#D97706,#EA580C 55%,#16A34A);box-shadow:0 5px 13px rgba(217,119,6,.22)}
    #${rootId} .gl-action-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin:10px 0}
    #${rootId} .gl-result{margin-top:8px;padding:10px;border:1px solid #FCD34D;border-radius:12px;background:linear-gradient(135deg,#FFFBEB,#F0FDF4);color:#334155;font-size:11.2px;font-weight:620;line-height:1.52}
    #${rootId} .gl-view-toolbar{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:7px;align-items:center;padding:0 3px 7px;border-bottom:1px solid #E2E8F0}
    #${rootId} .gl-view-toolbar button{min-height:32px;font-size:11px}
    #${rootId} .gl-canvas-wrap{min-width:0;min-height:0;overflow:hidden;border:1px solid #FDE68A;border-radius:14px;background:#fff}
    #${rootId} .gl-urbanization-canvas{width:100%;height:100%;display:block}
    @media(max-width:900px){#${rootId} .gl-body{grid-template-columns:240px minmax(0,1fr)}#${rootId} .gl-note{display:none}}
  </style>

  <div class="gl-head">
    <div style="font-size:24px;">🏙️</div>
    <div>
      <div class="gl-title">城市化、城市功能区与城市空间结构</div>
      <div class="gl-subtitle">比较城市化阶段、地租梯度、交通可达性和环境质量</div>
    </div>
    <div class="gl-note">课堂简化模型 · 不用于真实城市规划</div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">城市发展情境</div>
      <div class="gl-scenario-grid">
        <button type="button" data-scenario="early-urbanization">城市化初期</button>
        <button type="button" data-scenario="rapid-urbanization">快速城市化</button>
        <button type="button" data-scenario="mature-urbanization">成熟城市化</button>
        <button type="button" data-scenario="suburbanization">郊区化</button>
        <button type="button" data-scenario="reverse-urbanization">逆城市化</button>
      </div>

      <div class="gl-section-title">城市空间参数</div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">城市化率</span><span class="gl-value" data-role="urbanization-value">62%</span></div>
        <input type="range" min="15" max="95" step="1" value="${urbanizationRate}" data-role="urbanization" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">交通可达性</span><span class="gl-value" data-role="transport-value">7</span></div>
        <input type="range" min="0" max="10" step="1" value="${transportAccessibility}" data-role="transport" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">环境质量</span><span class="gl-value" data-role="environment-value">5</span></div>
        <input type="range" min="0" max="10" step="1" value="${environmentalQuality}" data-role="environment" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">地租梯度</span><span class="gl-value" data-role="rent-value">7</span></div>
        <input type="range" min="0" max="10" step="1" value="${landRentGradient}" data-role="rent" />
      </div>

      <div class="gl-action-grid">
        <button type="button" data-role="label-toggle" data-active="${showLabels}">功能区标注</button>
        <button type="button" data-role="auto-toggle" data-active="false">自动演示</button>
        <button type="button" data-role="reset">恢复初始</button>
        <button type="button" data-role="compare">切换下一情境</button>
      </div>

      <div class="gl-result" data-role="result">
        城市功能区会在地租、交通、环境和历史条件共同作用下形成。
      </div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="curve">城市化进程</button>
        <button type="button" data-view="zones">功能区布局</button>
        <button type="button" data-view="models">空间结构模型</button>
        <button type="button" data-view="issues">城市问题</button>
      </div>
      <div class="gl-canvas-wrap">
        <canvas class="gl-urbanization-canvas" width="980" height="570" data-role="canvas" aria-label="城市化、城市功能区与城市空间结构教学示意图"></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root=document.getElementById('${rootId}');
      if(!root)return;

      var urbanizationInput=root.querySelector('[data-role="urbanization"]');
      var transportInput=root.querySelector('[data-role="transport"]');
      var environmentInput=root.querySelector('[data-role="environment"]');
      var rentInput=root.querySelector('[data-role="rent"]');
      var urbanizationValue=root.querySelector('[data-role="urbanization-value"]');
      var transportValue=root.querySelector('[data-role="transport-value"]');
      var environmentValue=root.querySelector('[data-role="environment-value"]');
      var rentValue=root.querySelector('[data-role="rent-value"]');
      var scenarioButtons=root.querySelectorAll('[data-scenario]');
      var viewButtons=root.querySelectorAll('[data-view]');
      var labelToggle=root.querySelector('[data-role="label-toggle"]');
      var autoToggle=root.querySelector('[data-role="auto-toggle"]');
      var resetButton=root.querySelector('[data-role="reset"]');
      var compareButton=root.querySelector('[data-role="compare"]');
      var result=root.querySelector('[data-role="result"]');
      var canvas=root.querySelector('[data-role="canvas"]');

      if(!urbanizationInput||!transportInput||!environmentInput||!rentInput||!urbanizationValue||!transportValue||!environmentValue||!rentValue||!scenarioButtons.length||!viewButtons.length||!labelToggle||!autoToggle||!resetButton||!compareButton||!result||!canvas)return;

      var context=canvas.getContext('2d');
      if(!context)return;

      var width=canvas.width;
      var height=canvas.height;
      var scenarios=[
        {key:'early-urbanization',name:'城市化初期',urbanization:28,transport:3,environment:7,rent:4,view:'curve'},
        {key:'rapid-urbanization',name:'快速城市化',urbanization:62,transport:7,environment:5,rent:7,view:'zones'},
        {key:'mature-urbanization',name:'成熟城市化',urbanization:82,transport:9,environment:6,rent:8,view:'models'},
        {key:'suburbanization',name:'郊区化',urbanization:86,transport:8,environment:7,rent:6,view:'issues'},
        {key:'reverse-urbanization',name:'逆城市化',urbanization:90,transport:8,environment:9,rent:5,view:'curve'}
      ];
      var initial={scenario:'${scenario}',urbanization:${urbanizationRate},transport:${transportAccessibility},environment:${environmentalQuality},rent:${landRentGradient},showLabels:${showLabels}};
      var state={scenario:initial.scenario,view:'curve',showLabels:initial.showLabels,auto:false,startedAt:0,raf:0,phase:0,compareIndex:0};

      function clamp(value,min,max){return Math.max(min,Math.min(max,value));}
      function lerp(a,b,t){return a+(b-a)*t;}
      function ease(t){var p=clamp(t,0,1);return p<.5?2*p*p:1-Math.pow(-2*p+2,2)/2;}
      function roundRect(x,y,w,h,r){var q=Math.min(r,w/2,h/2);context.beginPath();context.moveTo(x+q,y);context.lineTo(x+w-q,y);context.quadraticCurveTo(x+w,y,x+w,y+q);context.lineTo(x+w,y+h-q);context.quadraticCurveTo(x+w,y+h,x+w-q,y+h);context.lineTo(x+q,y+h);context.quadraticCurveTo(x,y+h,x,y+h-q);context.lineTo(x,y+q);context.quadraticCurveTo(x,y,x+q,y);context.closePath();}
      function box(x,y,w,h,r,fill,stroke){roundRect(x,y,w,h,r);if(fill){context.fillStyle=fill;context.fill();}if(stroke){context.strokeStyle=stroke;context.lineWidth=1.2;context.stroke();}}
      function text(value,x,y,size,color,weight,align){context.save();context.font=(weight||600)+' '+size+'px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif';context.fillStyle=color||'#334155';context.textAlign=align||'left';context.textBaseline='middle';context.fillText(String(value),x,y);context.restore();}
      function line(x1,y1,x2,y2,color,lineWidth,dash){context.save();context.strokeStyle=color;context.lineWidth=lineWidth||1.5;context.setLineDash(dash||[]);context.beginPath();context.moveTo(x1,y1);context.lineTo(x2,y2);context.stroke();context.restore();}
      function circle(x,y,r,fill,stroke){context.beginPath();context.arc(x,y,r,0,Math.PI*2);if(fill){context.fillStyle=fill;context.fill();}if(stroke){context.strokeStyle=stroke;context.lineWidth=2;context.stroke();}}
      function scenarioByKey(key){var found=scenarios[1];scenarios.forEach(function(item){if(item.key===key)found=item;});return found;}
      function values(){return {urbanization:clamp(Number(urbanizationInput.value)||62,15,95),transport:clamp(Number(transportInput.value)||0,0,10),environment:clamp(Number(environmentInput.value)||0,0,10),rent:clamp(Number(rentInput.value)||0,0,10)};}
      function setInputs(value){urbanizationInput.value=String(Math.round(value.urbanization));transportInput.value=String(Math.round(value.transport));environmentInput.value=String(Math.round(value.environment));rentInput.value=String(Math.round(value.rent));}

      function derive(value){
        var stage='城市化加速期';
        var stageDesc='人口和产业加快向城市集中，基础设施需求上升。';
        if(value.urbanization<35){stage='城市化初期';stageDesc='城市化水平较低，城市数量和规模开始增长。';}
        else if(value.urbanization>=70&&value.urbanization<85){stage='成熟城市化阶段';stageDesc='城市化速度放缓，城市内部功能和发展质量更受关注。';}
        else if(value.urbanization>=85&&value.environment>=8){stage='郊区化或逆城市化';stageDesc='部分人口和功能向郊区、小城镇或环境较好地区移动。';}
        else if(value.urbanization>=85){stage='高城市化阶段';stageDesc='中心区与外围地区重新分工，城市网络联系增强。';}
        return {
          stage:stage,
          stageDesc:stageDesc,
          heatIsland:clamp(Math.round(value.urbanization*.055+value.rent*.55-value.environment*.75),0,10),
          commute:Math.round(18+value.urbanization*.23+value.transport*1.4+value.rent*1.8-value.environment*.35),
          servicePressure:clamp(Math.round(value.urbanization*.075+value.rent*.55-value.transport*.35),0,10),
          centerShare:clamp(24+value.rent*4-value.transport*1.2,20,64)
        };
      }

      function background(titleValue,subtitle){
        var gradient=context.createLinearGradient(0,0,width,height);
        gradient.addColorStop(0,'#FFFFFF');gradient.addColorStop(.6,'#F8FAFC');gradient.addColorStop(1,'#FEF3C7');
        context.fillStyle=gradient;context.fillRect(0,0,width,height);
        text(titleValue,28,31,18,'#92400E',880,'left');text(subtitle,28,55,11.5,'#64748B',620,'left');
      }

      function card(x,y,w,label,value,color,desc){
        box(x,y,w,72,12,'rgba(255,255,255,.94)','#FDE68A');
        text(label,x+14,y+17,10.5,'#64748B',720,'left');
        text(value,x+14,y+40,20,color,880,'left');
        text(desc,x+14,y+59,9.5,'#64748B',600,'left');
      }

      function curveView(item,value,derived){
        background('城市化进程与阶段变化','用S形曲线理解城市化初期、加速期、成熟期以及郊区化和逆城市化。');
        card(28,78,210,'城市化率',Math.round(value.urbanization)+'%','#D97706','城市人口占总人口比重');
        card(250,78,210,'阶段判断',derived.stage,'#7C3AED',item.name);
        card(472,78,210,'交通可达性',Math.round(value.transport),'#0284C7','影响外围扩展和通勤');
        card(694,78,258,'环境质量',Math.round(value.environment),'#16A34A','影响居住区和人口外迁');

        var left=88,top=190,chartW=814,chartH=274,bottom=top+chartH;
        for(var gx=0;gx<=8;gx+=1){var x=left+chartW*gx/8;line(x,top,x,bottom,'#E2E8F0',1,[]);text(gx*10,x,bottom+22,10,'#64748B',650,'center');}
        for(var gy=0;gy<=5;gy+=1){var y=top+chartH*gy/5;line(left,y,left+chartW,y,'#E2E8F0',1,[]);text(100-gy*20+'%',left-14,y,10,'#64748B',650,'right');}

        context.save();context.strokeStyle='#D97706';context.lineWidth=4;context.lineJoin='round';context.beginPath();
        for(var index=0;index<=80;index+=1){
          var px=left+chartW*index/80;
          var rate=12+78/(1+Math.exp(-(index-37)/9));
          var py=bottom-rate/100*chartH;
          if(index===0)context.moveTo(px,py);else context.lineTo(px,py);
        }
        context.stroke();context.restore();

        var markerX=left+chartW*value.urbanization/100;
        var markerY=bottom-value.urbanization/100*chartH;
        line(markerX,top,markerX,bottom,'#7C3AED',2,[6,5]);
        circle(markerX,markerY,8,'#FFFFFF','#7C3AED');
        if(state.showLabels)text(Math.round(value.urbanization)+'%',markerX,markerY-20,10,'#6D28D9',850,'center');

        [['初期',.13],['加速期',.43],['成熟期',.72],['郊区化／逆城市化',.91]].forEach(function(stage){
          var sx=left+chartW*stage[1];
          box(sx-55,top+20,110,28,14,'#FFFFFF','#FCD34D');
          text(stage[0],sx,top+34,9.5,'#92400E',760,'center');
        });

        box(88,498,814,46,12,'#FFFFFF','#FDE68A');
        text(derived.stageDesc,490,521,10.5,'#475569',690,'center');
      }

      function road(x1,y1,x2,y2,w){
        line(x1,y1,x2,y2,'#94A3B8',w+4,[]);
        line(x1,y1,x2,y2,'#FFFFFF',1.5,[8,7]);
      }

      function zoneLabel(x,y,label,color){
        if(!state.showLabels)return;
        box(x-48,y-14,96,28,14,'#FFFFFF',color);
        text(label,x,y,9.5,color,820,'center');
      }

      function zonesView(item,value,derived){
        background('城市功能区布局','商业、居住、工业和公共服务功能区在地租、交通、环境和历史条件共同作用下形成。');
        card(28,78,210,'中心商业强度',derived.centerShare.toFixed(0),'#DC2626','中心地租与可达性综合结果');
        card(250,78,210,'地租梯度',Math.round(value.rent),'#D97706','中心向外围递减程度');
        card(472,78,210,'交通可达性',Math.round(value.transport),'#0284C7','交通廊道和节点影响');
        card(694,78,258,'环境质量',Math.round(value.environment),'#16A34A','影响居住和公共空间布局');

        var mapX=45,mapY=180,mapW=890,mapH=334;
        box(mapX,mapY,mapW,mapH,18,'#F8FAFC','#CBD5E1');
        context.save();roundRect(mapX,mapY,mapW,mapH,18);context.clip();
        context.fillStyle='#DCFCE7';context.fillRect(mapX,mapY,mapW,mapH);

        var centerX=490+Math.sin(state.phase*Math.PI*2)*8;
        var centerY=340;
        var residentialRadius=128+value.environment*5;
        var industrialWidth=118+value.transport*7;

        context.fillStyle='#BFDBFE';context.beginPath();context.arc(centerX,centerY,residentialRadius,0,Math.PI*2);context.fill();
        context.fillStyle='#FCA5A5';context.beginPath();context.arc(centerX,centerY,56+value.rent*2.4,0,Math.PI*2);context.fill();
        context.fillStyle='#CBD5E1';context.fillRect(mapX+mapW-industrialWidth-36,mapY+72,industrialWidth,mapH-128);
        context.fillStyle='#A7F3D0';context.beginPath();context.arc(mapX+164,mapY+90,54+value.environment*3,0,Math.PI*2);context.fill();
        context.fillStyle='#FDE68A';context.beginPath();context.arc(mapX+210,mapY+mapH-82,48,0,Math.PI*2);context.fill();

        road(mapX+20,centerY,mapX+mapW-20,centerY,8+value.transport);
        road(centerX,mapY+15,centerX,mapY+mapH-15,7+value.transport*.8);
        road(mapX+50,mapY+mapH-30,mapX+mapW-42,mapY+35,5+value.transport*.65);
        context.restore();

        zoneLabel(centerX,centerY,'商业中心','#DC2626');
        zoneLabel(centerX-112,centerY+86,'居住区','#2563EB');
        zoneLabel(mapX+mapW-110,centerY-36,'工业区','#475569');
        zoneLabel(mapX+164,mapY+90,'生态公共空间','#16A34A');
        zoneLabel(mapX+210,mapY+mapH-82,'公共服务区','#D97706');

        box(45,526,890,28,10,'#FFFFFF','#FDE68A');
        text('地租较高且交通便利的中心区更易集聚商业；工业和大面积居住功能通常向外围或交通走廊扩展。',490,540,10,'#475569',670,'center');
      }

      function modelPanel(x,titleValue,color){
        box(x,180,292,320,16,'#FFFFFF',color);
        text(titleValue,x+146,206,13,color,860,'center');
      }

      function modelsView(item,value,derived){
        background('城市空间结构简化模型','同心圆、扇形和多核心模型用于解释总体形态，不是固定规划模板。');
        card(28,78,210,'当前阶段',derived.stage,'#7C3AED',item.name);
        card(250,78,210,'交通可达性',Math.round(value.transport),'#0284C7','越高越易形成走廊和多节点');
        card(472,78,210,'地租梯度',Math.round(value.rent),'#D97706','越高越突出中心层级');
        card(694,78,258,'外围环境',Math.round(value.environment),'#16A34A','影响居住和人口外移');

        modelPanel(24,'同心圆模型','#D97706');
        modelPanel(344,'扇形模型','#0284C7');
        modelPanel(664,'多核心模型','#7C3AED');

        [112,86,60,34].forEach(function(r,index){
          circle(170,346,r,['#DCFCE7','#BFDBFE','#FDE68A','#FCA5A5'][index],'#FFFFFF');
        });
        if(state.showLabels){text('中心商业区',170,346,8.5,'#991B1B',820,'center');text('由中心向外围分层',170,480,9.5,'#64748B',650,'center');}

        circle(490,350,112,'#DCFCE7','#FFFFFF');
        context.save();context.translate(490,350);
        ['#FCA5A5','#FDE68A','#BFDBFE','#CBD5E1'].forEach(function(color,index){
          context.fillStyle=color;context.beginPath();context.moveTo(0,0);context.arc(0,0,108,-.42+index*1.42,.55+index*1.42);context.closePath();context.fill();
        });
        circle(0,0,30,'#DC2626','#FFFFFF');context.restore();
        road(372,408,608,288,8);
        if(state.showLabels)text('交通廊道引导功能区呈扇形扩展',490,480,9.5,'#64748B',650,'center');

        [
          {x:735,y:306,r:40,color:'#FCA5A5',label:'商业'},
          {x:844,y:345,r:50,color:'#BFDBFE',label:'居住'},
          {x:756,y:418,r:44,color:'#FDE68A',label:'服务'},
          {x:890,y:430,r:36,color:'#CBD5E1',label:'工业'}
        ].forEach(function(center){
          circle(center.x,center.y,center.r,center.color,'#FFFFFF');
          if(state.showLabels)text(center.label,center.x,center.y,8.5,'#334155',820,'center');
        });
        road(710,458,918,282,7+value.transport*.5);
        if(state.showLabels)text('多个功能核心通过交通网络联系',810,480,9.5,'#64748B',650,'center');

        box(24,522,932,30,10,'#FFFBEB','#FDE68A');
        text('真实城市往往同时包含历史路径、地形约束和多种模型特征，不能机械套用单一模型。',490,537,10,'#475569',690,'center');
      }

      function gauge(cx,cy,ratio,color,label,valueText){
        context.save();context.lineWidth=14;context.strokeStyle='#E2E8F0';context.beginPath();context.arc(cx,cy,82,Math.PI,Math.PI*2);context.stroke();
        context.strokeStyle=color;context.lineCap='round';context.beginPath();context.arc(cx,cy,82,Math.PI,Math.PI+Math.PI*clamp(ratio,0,1));context.stroke();context.restore();
        text(valueText,cx,cy-4,18,color,880,'center');text(label,cx,cy+24,9.5,'#64748B',720,'center');
      }

      function issuesView(item,value,derived){
        background('城市化问题与治理选择','观察热岛、通勤、公共服务压力和外围扩展之间的联系。');
        card(28,78,210,'热岛强度',derived.heatIsland,'#DC2626','建设密度与绿地共同影响');
        card(250,78,210,'平均通勤时间',derived.commute+'分钟','#0284C7','仅为课堂相对值');
        card(472,78,210,'公共服务压力',derived.servicePressure,'#7C3AED','人口集中与设施供给共同影响');
        card(694,78,258,'发展情境',item.name,'#16A34A',derived.stage);

        gauge(190,312,derived.heatIsland/10,'#DC2626','城市热岛',derived.heatIsland);
        gauge(490,312,derived.commute/70,'#0284C7','通勤压力',derived.commute+'分');
        gauge(790,312,derived.servicePressure/10,'#7C3AED','服务压力',derived.servicePressure);

        [
          {title:'绿色交通',desc:'公共交通、步行和骑行网络',color:'#0284C7',score:value.transport},
          {title:'紧凑混合用地',desc:'缩短居住、就业和服务距离',color:'#D97706',score:10-value.rent*.35},
          {title:'蓝绿空间',desc:'绿地、水体和通风廊道',color:'#16A34A',score:value.environment},
          {title:'公共服务均衡',desc:'教育、医疗和生活服务下沉',color:'#7C3AED',score:10-derived.servicePressure*.45}
        ].forEach(function(policy,index){
          var x=44+index*228;
          box(x,430,204,82,14,'#FFFFFF',policy.color);
          text(policy.title,x+15,451,11,policy.color,850,'left');
          text(policy.desc,x+15,473,9.2,'#64748B',620,'left');
          box(x+15,490,174,9,5,'#E2E8F0',null);
          box(x+15,490,174*clamp(policy.score/10,0,1),9,5,policy.color,null);
        });

        box(44,526,888,28,10,'#FFFFFF','#FDE68A');
        text('城市问题通常需要土地、交通、住房、生态和公共服务协同治理，单一措施难以独立解决。',488,540,10,'#475569',680,'center');
      }

      function update(item,value,derived){
        urbanizationValue.textContent=Math.round(value.urbanization)+'%';
        transportValue.textContent=String(Math.round(value.transport));
        environmentValue.textContent=String(Math.round(value.environment));
        rentValue.textContent=String(Math.round(value.rent));

        Array.prototype.forEach.call(scenarioButtons,function(button){
          button.setAttribute('data-active',button.getAttribute('data-scenario')===state.scenario?'true':'false');
        });
        Array.prototype.forEach.call(viewButtons,function(button){
          button.setAttribute('data-active',button.getAttribute('data-view')===state.view?'true':'false');
        });
        labelToggle.setAttribute('data-active',state.showLabels?'true':'false');
        autoToggle.setAttribute('data-active',state.auto?'true':'false');

        var scenarioName=state.scenario==='custom'?'自定义参数':item.name;
        result.textContent=scenarioName+'下，城市化率约为'+Math.round(value.urbanization)+'%，接近'+derived.stage+'。地租梯度为'+Math.round(value.rent)+'，交通可达性为'+Math.round(value.transport)+'。模型中的热岛强度为'+derived.heatIsland+'，平均通勤时间约为'+derived.commute+'分钟，公共服务压力为'+derived.servicePressure+'。';
      }

      function render(){
        if(!root.isConnected){state.auto=false;return;}
        var item=scenarioByKey(state.scenario);
        var value=values();
        var derived=derive(value);
        update(item,value,derived);
        context.clearRect(0,0,width,height);
        if(state.view==='curve')curveView(item,value,derived);
        else if(state.view==='zones')zonesView(item,value,derived);
        else if(state.view==='models')modelsView(item,value,derived);
        else issuesView(item,value,derived);
      }

      function applyScenario(key,changeView){
        var item=scenarioByKey(key);
        state.scenario=item.key;
        setInputs(item);
        if(changeView)state.view=item.view;
        state.compareIndex=scenarios.indexOf(item);
        render();
      }

      function stopAuto(){
        state.auto=false;
        state.startedAt=0;
        if(state.raf){cancelAnimationFrame(state.raf);state.raf=0;}
        render();
      }

      function animate(timestamp){
        if(!root.isConnected){state.auto=false;return;}
        if(!state.auto)return;
        if(!state.startedAt)state.startedAt=timestamp;

        var elapsed=timestamp-state.startedAt;
        var duration=5200;
        var segment=Math.floor(elapsed/duration);
        var local=(elapsed%duration)/duration;
        var from=scenarios[segment%scenarios.length];
        var to=scenarios[(segment+1)%scenarios.length];
        var progress=ease(clamp(local/.82,0,1));

        state.scenario=local<.5?from.key:to.key;
        state.view=['curve','zones','models','issues'][Math.floor(elapsed/6500)%4];
        state.phase=(elapsed/3200)%1;
        setInputs({
          urbanization:lerp(from.urbanization,to.urbanization,progress),
          transport:lerp(from.transport,to.transport,progress),
          environment:lerp(from.environment,to.environment,progress),
          rent:lerp(from.rent,to.rent,progress)
        });
        render();
        state.raf=requestAnimationFrame(animate);
      }

      function manual(){
        if(state.auto)stopAuto();
        state.scenario='custom';
        render();
      }

      [urbanizationInput,transportInput,environmentInput,rentInput].forEach(function(input){
        input.addEventListener('input',manual);
      });

      Array.prototype.forEach.call(scenarioButtons,function(button){
        button.addEventListener('click',function(){
          if(state.auto)stopAuto();
          applyScenario(button.getAttribute('data-scenario')||'rapid-urbanization',true);
        });
      });

      Array.prototype.forEach.call(viewButtons,function(button){
        button.addEventListener('click',function(){
          if(state.auto)stopAuto();
          state.view=button.getAttribute('data-view')||'curve';
          render();
        });
      });

      labelToggle.addEventListener('click',function(){
        state.showLabels=!state.showLabels;
        render();
      });

      autoToggle.addEventListener('click',function(){
        if(state.auto){stopAuto();return;}
        state.auto=true;
        state.startedAt=0;
        state.raf=requestAnimationFrame(animate);
        render();
      });

      resetButton.addEventListener('click',function(){
        if(state.auto)stopAuto();
        state.scenario=initial.scenario;
        state.view='curve';
        state.showLabels=initial.showLabels;
        setInputs(initial);
        state.compareIndex=scenarios.indexOf(scenarioByKey(initial.scenario));
        render();
      });

      compareButton.addEventListener('click',function(){
        if(state.auto)stopAuto();
        state.compareIndex=(state.compareIndex+1)%scenarios.length;
        var next=scenarios[state.compareIndex];
        state.scenario=next.key;
        state.view=next.view;
        setInputs(next);
        render();
      });

      state.compareIndex=scenarios.indexOf(scenarioByKey(initial.scenario));
      setInputs(initial);
      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_HUMAN_URBANIZATION:
GeographyLabTemplate[] = [
  {
    id: 'geography-urbanization-functional-zones-spatial-structure',
    group: '👥 人口、聚落与城市发展',
    name: '城市化、城市功能区与城市空间结构',
    emoji: '🏙️',
    desc: '调节城市化率、交通可达性、环境质量和地租梯度，观察功能区布局、城市空间模型、郊区化与城市问题。',
    params: [
      {
        key: 'scenario',
        label: '初始城市发展情境',
        type: 'select',
        options: [
          { label: '城市化初期', value: 'early-urbanization' },
          { label: '快速城市化', value: 'rapid-urbanization' },
          { label: '成熟城市化', value: 'mature-urbanization' },
          { label: '郊区化', value: 'suburbanization' },
          { label: '逆城市化', value: 'reverse-urbanization' },
        ],
        defaultValue: 'rapid-urbanization',
      },
      {
        key: 'urbanizationRate',
        label: '城市化率（%）',
        type: 'number',
        min: 15,
        max: 95,
        step: 1,
        defaultValue: 62,
        hint: '城市人口占总人口比重，仅用于课堂阶段比较。',
      },
      {
        key: 'transportAccessibility',
        label: '交通可达性',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '交通可达性影响商业集聚、通勤和城市外围扩展。',
      },
      {
        key: 'environmentalQuality',
        label: '环境质量',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 5,
        hint: '环境质量影响居住区、公共空间和人口向外围移动。',
      },
      {
        key: 'landRentGradient',
        label: '地租梯度',
        type: 'number',
        min: 0,
        max: 10,
        step: 1,
        defaultValue: 7,
        hint: '地租梯度表示中心区与外围地区土地成本差异的课堂强度。',
      },
      {
        key: 'showLabels',
        label: '显示功能区标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildHumanUrbanizationHTML,
  },
]
