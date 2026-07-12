/**
 * geographyLabTemplatesHumanPopulationTransition.ts
 *
 * 地理第38批B1：人口增长、人口结构与人口转变。
 *
 * 教学目标：联动理解出生率、死亡率、自然增长率、净迁移率、年龄结构、
 * 人口转变阶段和抚养负担之间的关系。
 *
 * 教学边界：
 * - 所有参数、曲线、年龄占比和阶段判断均为课堂简化示意；
 * - 忽略政策、战争、疾病、教育、就业和区域差异等复杂变量；
 * - 自动演示不代表任何国家或地区的真实历史；
 * - 不用于真实人口预测、财政测算或公共政策决策。
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

function buildPopulationTransitionHTML(
  params: Record<string, GeographyLabParamValue>,
  rootId: string,
): string {
  const allowedScenarios = [
    'high-stationary',
    'early-transition',
    'late-transition',
    'low-stationary',
    'aging-decline',
  ]
  const requestedScenario = stringValue(params, 'scenario', 'late-transition')
  const scenario = allowedScenarios.includes(requestedScenario)
    ? requestedScenario
    : 'late-transition'
  const birthRate = Math.max(5, Math.min(45, numberValue(params, 'birthRate', 20)))
  const deathRate = Math.max(4, Math.min(40, numberValue(params, 'deathRate', 8)))
  const lifeExpectancy = Math.max(38, Math.min(88, numberValue(params, 'lifeExpectancy', 72)))
  const netMigration = Math.max(-8, Math.min(8, numberValue(params, 'netMigration', 1)))
  const showLabels = booleanValue(params, 'showLabels', true)

  return `
<div id="${rootId}" class="gl-population-transition-root">
  <style>
    #${rootId}{width:100%;height:100%;overflow:hidden;box-sizing:border-box;border:1px solid #C7D2FE;border-radius:18px;background:#fff;color:#172554;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;box-shadow:0 12px 34px rgba(49,46,129,.11)}
    #${rootId} *{box-sizing:border-box}
    #${rootId} .gl-head{height:56px;padding:0 18px;display:flex;align-items:center;gap:12px;border-bottom:1px solid #C7D2FE;background:linear-gradient(135deg,#EEF2FF,#F5F3FF 54%,#ECFEFF)}
    #${rootId} .gl-title{color:#312E81;font-size:16px;font-weight:880}
    #${rootId} .gl-subtitle{margin-top:2px;color:#64748B;font-size:11px}
    #${rootId} .gl-note{margin-left:auto;padding:5px 10px;border:1px solid #DDD6FE;border-radius:999px;background:#fff;color:#6D28D9;font-size:11px;font-weight:750;white-space:nowrap}
    #${rootId} .gl-body{height:calc(100% - 56px);display:grid;grid-template-columns:276px minmax(0,1fr)}
    #${rootId} .gl-controls{min-height:0;padding:13px;overflow:auto;border-right:1px solid #E0E7FF;background:linear-gradient(180deg,#F8FAFF,#F5F3FF 64%,#ECFEFF)}
    #${rootId} .gl-stage{min-width:0;min-height:0;display:grid;grid-template-rows:46px minmax(0,1fr);padding:8px;background:radial-gradient(circle at 48% 24%,#fff 0%,#F8FAFC 64%,#EEF2FF 100%)}
    #${rootId} .gl-section-title{margin:1px 0 8px;color:#3730A3;font-size:11.5px;font-weight:850}
    #${rootId} .gl-scenario-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:12px}
    #${rootId} .gl-scenario-grid button:last-child{grid-column:1/-1}
    #${rootId} .gl-row{margin-bottom:10px}
    #${rootId} .gl-label-line{display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:5px}
    #${rootId} .gl-label{color:#334155;font-size:11.5px;font-weight:730}
    #${rootId} .gl-value{min-width:58px;padding:3px 7px;border-radius:999px;background:#E0E7FF;color:#4338CA;font-size:11px;font-weight:850;text-align:center}
    #${rootId} input[type=range]{width:100%;height:6px;margin:0;appearance:none;border-radius:999px;outline:none;background:linear-gradient(90deg,#C4B5FD,#A5F3FC);cursor:pointer}
    #${rootId} input[type=range]::-webkit-slider-thumb{width:16px;height:16px;appearance:none;border:2px solid #fff;border-radius:50%;background:linear-gradient(135deg,#4F46E5,#0891B2);box-shadow:0 1px 5px rgba(49,46,129,.42)}
    #${rootId} button{min-height:32px;padding:6px 7px;border:1px solid #C7D2FE;border-radius:9px;background:#fff;color:#4338CA;font-size:10.8px;font-weight:790;cursor:pointer}
    #${rootId} button[data-active="true"]{border-color:#4338CA;color:#fff;background:linear-gradient(135deg,#4F46E5,#7C3AED 58%,#0891B2);box-shadow:0 5px 13px rgba(79,70,229,.23)}
    #${rootId} .gl-action-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin:10px 0}
    #${rootId} .gl-result{margin-top:8px;padding:10px;border:1px solid #C7D2FE;border-radius:12px;background:linear-gradient(135deg,#EEF2FF,#ECFEFF);color:#334155;font-size:11.2px;font-weight:620;line-height:1.52}
    #${rootId} .gl-view-toolbar{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:7px;align-items:center;padding:0 3px 7px;border-bottom:1px solid #E2E8F0}
    #${rootId} .gl-view-toolbar button{min-height:32px;font-size:11px}
    #${rootId} .gl-canvas-wrap{min-width:0;min-height:0;position:relative;overflow:hidden;border:1px solid #E0E7FF;border-radius:14px;background:#fff}
    #${rootId} .gl-population-canvas{width:100%;height:100%;display:block}
    @media(max-width:900px){#${rootId} .gl-body{grid-template-columns:238px minmax(0,1fr)}#${rootId} .gl-note{display:none}}
  </style>

  <div class="gl-head">
    <div style="font-size:24px">👥</div>
    <div>
      <div class="gl-title">人口增长、人口结构与人口转变</div>
      <div class="gl-subtitle">联动观察增长率、年龄金字塔、人口转变阶段和抚养负担</div>
    </div>
    <div class="gl-note">课堂简化模型 · 不用于真实人口预测</div>
  </div>

  <div class="gl-body">
    <div class="gl-controls">
      <div class="gl-section-title">典型人口情境</div>
      <div class="gl-scenario-grid">
        <button type="button" data-scenario="high-stationary">高位稳定</button>
        <button type="button" data-scenario="early-transition">加速增长</button>
        <button type="button" data-scenario="late-transition">增长放缓</button>
        <button type="button" data-scenario="low-stationary">低位稳定</button>
        <button type="button" data-scenario="aging-decline">老龄化与负增长</button>
      </div>

      <div class="gl-section-title">人口过程参数</div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">出生率</span><span class="gl-value" data-role="birth-value">20‰</span></div>
        <input type="range" min="5" max="45" step="1" value="${birthRate}" data-role="birth-rate" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">死亡率</span><span class="gl-value" data-role="death-value">8‰</span></div>
        <input type="range" min="4" max="40" step="1" value="${deathRate}" data-role="death-rate" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">平均预期寿命</span><span class="gl-value" data-role="life-value">72岁</span></div>
        <input type="range" min="38" max="88" step="1" value="${lifeExpectancy}" data-role="life-expectancy" />
      </div>
      <div class="gl-row">
        <div class="gl-label-line"><span class="gl-label">净迁移率</span><span class="gl-value" data-role="migration-value">+1‰</span></div>
        <input type="range" min="-8" max="8" step="1" value="${netMigration}" data-role="net-migration" />
      </div>

      <div class="gl-action-grid">
        <button type="button" data-role="label-toggle" data-active="${showLabels}">结构标注</button>
        <button type="button" data-role="auto-toggle" data-active="false">自动演示</button>
        <button type="button" data-role="reset">恢复初始</button>
        <button type="button" data-role="compare">对比下一阶段</button>
      </div>
      <div class="gl-result" data-role="result">调节参数，观察人口过程与年龄结构之间的联动变化。</div>
    </div>

    <div class="gl-stage">
      <div class="gl-view-toolbar">
        <button type="button" data-view="growth">增长曲线</button>
        <button type="button" data-view="pyramid">年龄金字塔</button>
        <button type="button" data-view="transition">转变阶段</button>
        <button type="button" data-view="dependency">抚养负担</button>
      </div>
      <div class="gl-canvas-wrap">
        <canvas class="gl-population-canvas" width="980" height="570" data-role="canvas" aria-label="人口增长、年龄结构和人口转变教学示意图"></canvas>
      </div>
    </div>
  </div>

  <script>
    (function(){
      var root=document.getElementById('${rootId}');
      if(!root)return;

      var birthInput=root.querySelector('[data-role="birth-rate"]');
      var deathInput=root.querySelector('[data-role="death-rate"]');
      var lifeInput=root.querySelector('[data-role="life-expectancy"]');
      var migrationInput=root.querySelector('[data-role="net-migration"]');
      var birthValue=root.querySelector('[data-role="birth-value"]');
      var deathValue=root.querySelector('[data-role="death-value"]');
      var lifeValue=root.querySelector('[data-role="life-value"]');
      var migrationValue=root.querySelector('[data-role="migration-value"]');
      var scenarioButtons=root.querySelectorAll('[data-scenario]');
      var viewButtons=root.querySelectorAll('[data-view]');
      var labelToggle=root.querySelector('[data-role="label-toggle"]');
      var autoToggle=root.querySelector('[data-role="auto-toggle"]');
      var resetButton=root.querySelector('[data-role="reset"]');
      var compareButton=root.querySelector('[data-role="compare"]');
      var result=root.querySelector('[data-role="result"]');
      var canvas=root.querySelector('[data-role="canvas"]');

      if(!birthInput||!deathInput||!lifeInput||!migrationInput||!birthValue||!deathValue||!lifeValue||!migrationValue||!scenarioButtons.length||!viewButtons.length||!labelToggle||!autoToggle||!resetButton||!compareButton||!result||!canvas)return;
      var context=canvas.getContext('2d');
      if(!context)return;

      var width=canvas.width;
      var height=canvas.height;
      var scenarios=[
        {key:'high-stationary',name:'高位稳定',birth:40,death:36,life:42,migration:0,view:'transition'},
        {key:'early-transition',name:'加速增长',birth:35,death:15,life:56,migration:1,view:'growth'},
        {key:'late-transition',name:'增长放缓',birth:20,death:8,life:72,migration:1,view:'pyramid'},
        {key:'low-stationary',name:'低位稳定',birth:10,death:9,life:80,migration:0,view:'dependency'},
        {key:'aging-decline',name:'老龄化与负增长',birth:7,death:11,life:85,migration:-1,view:'dependency'}
      ];
      var views=['growth','pyramid','transition','dependency'];
      var initial={scenario:'${scenario}',birth:${birthRate},death:${deathRate},life:${lifeExpectancy},migration:${netMigration},showLabels:${showLabels}};
      var state={scenario:initial.scenario,view:'growth',showLabels:initial.showLabels,auto:false,startedAt:0,raf:0,compareIndex:0};

      function clamp(value,min,max){return Math.max(min,Math.min(max,value));}
      function lerp(a,b,t){return a+(b-a)*t;}
      function ease(t){var p=clamp(t,0,1);return p<.5?2*p*p:1-Math.pow(-2*p+2,2)/2;}
      function roundRect(x,y,w,h,r){var q=Math.min(r,w/2,h/2);context.beginPath();context.moveTo(x+q,y);context.lineTo(x+w-q,y);context.quadraticCurveTo(x+w,y,x+w,y+q);context.lineTo(x+w,y+h-q);context.quadraticCurveTo(x+w,y+h,x+w-q,y+h);context.lineTo(x+q,y+h);context.quadraticCurveTo(x,y+h,x,y+h-q);context.lineTo(x,y+q);context.quadraticCurveTo(x,y,x+q,y);context.closePath();}
      function box(x,y,w,h,r,fill,stroke){roundRect(x,y,w,h,r);if(fill){context.fillStyle=fill;context.fill();}if(stroke){context.strokeStyle=stroke;context.lineWidth=1.2;context.stroke();}}
      function text(value,x,y,size,color,weight,align){context.save();context.font=(weight||600)+' '+size+'px -apple-system,BlinkMacSystemFont,Segoe UI,sans-serif';context.fillStyle=color||'#334155';context.textAlign=align||'left';context.textBaseline='middle';context.fillText(String(value),x,y);context.restore();}
      function line(x1,y1,x2,y2,color,lineWidth,dash){context.save();context.strokeStyle=color;context.lineWidth=lineWidth||1.5;context.setLineDash(dash||[]);context.beginPath();context.moveTo(x1,y1);context.lineTo(x2,y2);context.stroke();context.restore();}
      function circle(x,y,r,fill,stroke){context.beginPath();context.arc(x,y,r,0,Math.PI*2);if(fill){context.fillStyle=fill;context.fill();}if(stroke){context.strokeStyle=stroke;context.lineWidth=2;context.stroke();}}
      function signed(value){var rounded=Math.round(value);return rounded>0?'+'+rounded:String(rounded);}
      function scenarioByKey(key){var found=scenarios[2];scenarios.forEach(function(item){if(item.key===key)found=item;});return found;}
      function setInputs(values){birthInput.value=String(Math.round(values.birth));deathInput.value=String(Math.round(values.death));lifeInput.value=String(Math.round(values.life));migrationInput.value=String(Math.round(values.migration));}
      function values(){return {birth:clamp(Number(birthInput.value)||20,5,45),death:clamp(Number(deathInput.value)||8,4,40),life:clamp(Number(lifeInput.value)||72,38,88),migration:clamp(Number(migrationInput.value)||0,-8,8)};}

      function derive(v){
        var natural=v.birth-v.death;
        var total=natural+v.migration;
        var youth=clamp(17+(v.birth-10)*.72-(v.life-70)*.05,8,46);
        var elderly=clamp(6+(v.life-58)*.43+(11-v.birth)*.36,3,35);
        if(youth+elderly>66){var scale=66/(youth+elderly);youth*=scale;elderly*=scale;}
        var working=100-youth-elderly;
        var child=youth/working*100;
        var old=elderly/working*100;
        var stage=4;
        var stageName='低出生率、低死亡率';
        var hint='人口增长趋缓，年龄结构逐步老化。';
        if(v.death>=25&&v.birth>=30){stage=1;stageName='高出生率、高死亡率';hint='人口规模波动较大，自然增长率通常不高。';}
        else if(v.birth>=28&&v.death<25){stage=2;stageName='死亡率快速下降';hint='出生率仍高，自然增长率扩大，人口加速增长。';}
        else if(v.birth>=14&&v.death<=15){stage=3;stageName='出生率持续下降';hint='自然增长率回落，劳动年龄人口比重往往较高。';}
        else if(total<0){stage=5;stageName='低出生率与负增长';hint='人口可能减少，老年抚养负担上升。';}
        var structure=youth>=32?'少年型':(elderly>=18||youth<18?'老年型':'成年型');
        return {natural:natural,total:total,youth:youth,elderly:elderly,working:working,child:child,old:old,dependency:child+old,stage:stage,stageName:stageName,hint:hint,structure:structure};
      }

      function background(titleValue,subtitle){
        var gradient=context.createLinearGradient(0,0,width,height);gradient.addColorStop(0,'#fff');gradient.addColorStop(.6,'#F8FAFC');gradient.addColorStop(1,'#EEF2FF');context.fillStyle=gradient;context.fillRect(0,0,width,height);
        text(titleValue,28,31,18,'#312E81',880,'left');text(subtitle,28,55,11.5,'#64748B',620,'left');
      }

      function card(x,y,w,label,value,color,desc){
        box(x,y,w,72,12,'rgba(255,255,255,.94)','#E0E7FF');text(label,x+14,y+17,10.5,'#64748B',720,'left');text(value,x+14,y+40,20,color,880,'left');text(desc,x+14,y+59,9.5,'#64748B',600,'left');
      }

      function growthView(v,d){
        background('人口增长曲线','以100为起点，演示出生、死亡和迁移共同作用下的相对人口规模变化。');
        card(28,78,204,'自然增长率',signed(d.natural)+'‰',d.natural>=0?'#16A34A':'#DC2626','出生率－死亡率');
        card(244,78,204,'总增长率',signed(d.total)+'‰',d.total>=0?'#4F46E5':'#DC2626','自然增长率＋净迁移率');
        card(460,78,204,'人口转变判断','第'+d.stage+'阶段','#7C3AED',d.stageName);
        card(676,78,276,'年龄结构',d.structure,d.structure==='老年型'?'#EA580C':'#0891B2','由出生率与寿命共同塑造');
        var left=82,top=186,chartW=838,chartH=310,bottom=top+chartH,population=100,series=[],minPop=100,maxPop=100;
        for(var year=0;year<=80;year+=5){if(year>0){var p=year/80;var rate=lerp(v.birth,10,p*.76)-lerp(v.death,9,p*.58)+lerp(v.migration,0,p*.55);population*=Math.exp(rate/1000*5);}series.push({year:year,population:population});minPop=Math.min(minPop,population);maxPop=Math.max(maxPop,population);}
        var yMin=Math.floor((minPop-8)/10)*10,yMax=Math.ceil((maxPop+8)/10)*10;if(yMax-yMin<40){yMin-=20;yMax+=20;}
        for(var gx=0;gx<=8;gx+=1){var x=left+chartW*gx/8;line(x,top,x,bottom,'#E2E8F0',1,[]);text(gx*10,x,bottom+22,10,'#64748B',650,'center');}
        for(var gy=0;gy<=5;gy+=1){var y=top+chartH*gy/5;line(left,y,left+chartW,y,'#E2E8F0',1,[]);text(Math.round(yMax-(yMax-yMin)*gy/5),left-12,y,10,'#64748B',650,'right');}
        context.save();context.strokeStyle=d.total>=0?'#4F46E5':'#DC2626';context.lineWidth=4;context.lineJoin='round';context.beginPath();series.forEach(function(point,index){var px=left+chartW*point.year/80;var py=bottom-(point.population-yMin)/(yMax-yMin)*chartH;if(index===0)context.moveTo(px,py);else context.lineTo(px,py);});context.stroke();context.restore();
        series.forEach(function(point,index){if(index%4!==0&&index!==series.length-1)return;var px=left+chartW*point.year/80;var py=bottom-(point.population-yMin)/(yMax-yMin)*chartH;circle(px,py,5,'#fff',d.total>=0?'#4F46E5':'#DC2626');if(state.showLabels)text(point.population.toFixed(0),px,py-17,9.5,'#312E81',780,'center');});
        text('未来课堂时间（年）',left+chartW/2,bottom+47,10.5,'#475569',720,'center');
      }

      function pyramidView(v,d){
        background('年龄结构金字塔','比较年龄组和性别的课堂示意占比，判断少年型、成年型或老年型结构。');
        card(28,78,210,'结构类型',d.structure,d.structure==='老年型'?'#EA580C':'#4F46E5','按少儿与老年占比判断');
        card(250,78,210,'少儿人口占比',d.youth.toFixed(1)+'%','#0891B2','0—14岁课堂估算');
        card(472,78,210,'劳动年龄人口',d.working.toFixed(1)+'%','#4F46E5','15—59岁课堂估算');
        card(694,78,258,'老年人口占比',d.elderly.toFixed(1)+'%','#EA580C','60岁及以上课堂估算');
        var bands=[['0—14岁',d.youth],['15—29岁',d.working*.31],['30—44岁',d.working*.35],['45—59岁',d.working*.34],['60—74岁',d.elderly*.68],['75岁以上',d.elderly*.32]];
        var center=490,startY=222,maxShare=20;bands.forEach(function(item){maxShare=Math.max(maxShare,item[1]);});
        text('男性',295,190,12,'#2563EB',850,'center');text('年龄组',center,190,12,'#475569',850,'center');text('女性',685,190,12,'#DB2777',850,'center');line(center,204,center,510,'#94A3B8',1.3,[]);
        bands.forEach(function(item,index){var y=startY+index*50;var male=item[1]*(index>=4?.46:.51);var female=item[1]-male;var maleW=male/maxShare*290;var femaleW=female/maxShare*290;box(center-maleW-55,y,maleW,38,8,index===0?'#67E8F9':'#93C5FD','#2563EB');box(center+55,y,femaleW,38,8,index>=4?'#FDBA74':'#F9A8D4',index>=4?'#EA580C':'#DB2777');box(center-48,y+3,96,32,7,'#fff','#E2E8F0');text(item[0],center,y+19,10,'#334155',760,'center');if(state.showLabels){text(male.toFixed(1)+'%',center-maleW-62,y+19,9,'#1D4ED8',730,'right');text(female.toFixed(1)+'%',center+femaleW+62,y+19,9,index>=4?'#C2410C':'#BE185D',730,'left');}});
        box(28,526,924,30,10,'#F8FAFC','#E2E8F0');text(d.structure==='少年型'?'底部较宽：教育和新增就业需求可能较大。':(d.structure==='老年型'?'顶部相对变宽：养老、医疗和劳动力供给议题更突出。':'中部较宽：劳动年龄人口比重较高。'),490,541,10.5,'#475569',720,'center');
      }

      function transitionView(v,d){
        background('人口转变阶段','通过出生率、死亡率和自然增长率的相对变化理解人口转变的一般过程。');
        var left=104,top=132,chartW=780,chartH=286,bottom=top+chartH,names=['高位稳定','早期转变','后期转变','低位稳定'];
        for(var i=0;i<4;i+=1){var sectionX=left+i*chartW/4;context.fillStyle=i%2===0?'rgba(238,242,255,.82)':'rgba(236,254,255,.70)';context.fillRect(sectionX,top,chartW/4,chartH);text('第'+(i+1)+'阶段',sectionX+chartW/8,top+19,10.5,'#4338CA',820,'center');text(names[i],sectionX+chartW/8,top+38,9.5,'#64748B',650,'center');}
        for(var gy=0;gy<=4;gy+=1){var y=top+chartH*gy/4;line(left,y,left+chartW,y,'#CBD5E1',1,[]);text(40-gy*10+'‰',left-16,y,10,'#64748B',650,'right');}
        var birth=[39,38,34,25,17,11,9],death=[37,34,18,11,9,9,10];
        function curve(valuesList,color){context.save();context.strokeStyle=color;context.lineWidth=4;context.beginPath();valuesList.forEach(function(value,index){var x=left+chartW*index/(valuesList.length-1);var y=bottom-value/40*chartH;if(index===0)context.moveTo(x,y);else context.lineTo(x,y);});context.stroke();context.restore();}
        curve(birth,'#DB2777');curve(death,'#2563EB');line(660,96,698,96,'#DB2777',4,[]);text('出生率',708,96,10.5,'#BE185D',780,'left');line(792,96,830,96,'#2563EB',4,[]);text('死亡率',840,96,10.5,'#1D4ED8',780,'left');
        var markerX=d.stage===5?left+chartW-8:left+(d.stage-1)/3*chartW;line(markerX,top-12,markerX,bottom+12,'#7C3AED',2,[6,5]);circle(markerX,top-12,7,'#7C3AED','#fff');
        box(28,454,924,78,12,'#fff','#C4B5FD');text('当前判断：'+d.stageName,48,476,12,'#6D28D9',850,'left');text('出生率 '+Math.round(v.birth)+'‰  ·  死亡率 '+Math.round(v.death)+'‰  ·  自然增长率 '+signed(d.natural)+'‰',48,500,10.5,'#475569',680,'left');text(d.hint,48,520,10.5,'#64748B',650,'left');
      }

      function donut(cx,cy,r,ratio,color,label,valueText){context.save();context.lineWidth=14;context.strokeStyle='#E2E8F0';context.beginPath();context.arc(cx,cy,r,0,Math.PI*2);context.stroke();context.strokeStyle=color;context.lineCap='round';context.beginPath();context.arc(cx,cy,r,-Math.PI/2,-Math.PI/2+Math.PI*2*clamp(ratio,0,1));context.stroke();context.restore();text(valueText,cx,cy-4,18,color,880,'center');text(label,cx,cy+19,9.5,'#64748B',720,'center');}

      function dependencyView(v,d){
        background('抚养负担与人口老龄化','比较每100名劳动年龄人口对应的少儿和老年人口数量。');
        card(28,78,210,'少儿抚养比',d.child.toFixed(1),'#0891B2','少儿数／劳动年龄人口×100');
        card(250,78,210,'老年抚养比',d.old.toFixed(1),'#EA580C','老年数／劳动年龄人口×100');
        card(472,78,210,'总抚养比',d.dependency.toFixed(1),'#7C3AED','少儿抚养比＋老年抚养比');
        card(694,78,258,'预期寿命',Math.round(v.life)+'岁','#4F46E5','仅作影响年龄结构的课堂参数');
        var barX=72,barY=205,barW=836,barH=64,youthW=barW*d.youth/100,workingW=barW*d.working/100,elderlyW=barW*d.elderly/100;box(barX,barY,barW,barH,14,'#F8FAFC','#CBD5E1');context.save();roundRect(barX,barY,barW,barH,14);context.clip();context.fillStyle='#67E8F9';context.fillRect(barX,barY,youthW,barH);context.fillStyle='#818CF8';context.fillRect(barX+youthW,barY,workingW,barH);context.fillStyle='#FDBA74';context.fillRect(barX+youthW+workingW,barY,elderlyW,barH);context.restore();
        if(state.showLabels){text('少儿 '+d.youth.toFixed(1)+'%',barX+youthW/2,barY+32,10,'#155E75',850,'center');text('劳动年龄 '+d.working.toFixed(1)+'%',barX+youthW+workingW/2,barY+32,10,'#fff',850,'center');text('老年 '+d.elderly.toFixed(1)+'%',barX+youthW+workingW+elderlyW/2,barY+32,10,'#9A3412',850,'center');}
        donut(225,371,72,d.child/100,'#0891B2','少儿抚养比',d.child.toFixed(1));donut(490,371,72,d.old/100,'#EA580C','老年抚养比',d.old.toFixed(1));donut(755,371,72,d.dependency/140,'#7C3AED','总抚养比',d.dependency.toFixed(1));
        var message=d.old>=35?'老年抚养比明显上升，应联系养老、医疗和劳动力供给等议题。':(d.child>=55?'少儿抚养比偏高，教育和新增就业岗位可能面临较大需求。':'抚养负担相对较轻，但人口红利不会自动转化为发展优势。');box(72,492,836,50,12,'#fff','#E0E7FF');text(message,490,517,10.5,'#475569',680,'center');
      }

      function update(v,d){
        birthValue.textContent=Math.round(v.birth)+'‰';deathValue.textContent=Math.round(v.death)+'‰';lifeValue.textContent=Math.round(v.life)+'岁';migrationValue.textContent=signed(v.migration)+'‰';
        Array.prototype.forEach.call(scenarioButtons,function(button){button.setAttribute('data-active',button.getAttribute('data-scenario')===state.scenario?'true':'false');});
        Array.prototype.forEach.call(viewButtons,function(button){button.setAttribute('data-active',button.getAttribute('data-view')===state.view?'true':'false');});
        labelToggle.setAttribute('data-active',state.showLabels?'true':'false');autoToggle.setAttribute('data-active',state.auto?'true':'false');
        result.textContent='当前接近'+d.stageName+'，自然增长率为'+signed(d.natural)+'‰，总增长率为'+signed(d.total)+'‰；年龄结构呈'+d.structure+'，总抚养比约为'+d.dependency.toFixed(1)+'。'+d.hint;
      }

      function render(){
        if(!root.isConnected){state.auto=false;return;}
        var v=values();var d=derive(v);update(v,d);context.clearRect(0,0,width,height);
        if(state.view==='growth')growthView(v,d);else if(state.view==='pyramid')pyramidView(v,d);else if(state.view==='transition')transitionView(v,d);else dependencyView(v,d);
      }

      function applyScenario(key,changeView){var item=scenarioByKey(key);state.scenario=item.key;setInputs(item);if(changeView)state.view=item.view;state.compareIndex=scenarios.indexOf(item);render();}
      function stopAuto(){state.auto=false;state.startedAt=0;if(state.raf){cancelAnimationFrame(state.raf);state.raf=0;}state.scenario='custom';render();}
      function animate(timestamp){
        if(!root.isConnected){state.auto=false;return;}if(!state.auto)return;if(!state.startedAt)state.startedAt=timestamp;
        var elapsed=timestamp-state.startedAt;var duration=5200;var segment=Math.floor(elapsed/duration);var local=(elapsed%duration)/duration;var from=scenarios[segment%scenarios.length];var to=scenarios[(segment+1)%scenarios.length];var t=ease(clamp(local/.82,0,1));
        setInputs({birth:lerp(from.birth,to.birth,t),death:lerp(from.death,to.death,t),life:lerp(from.life,to.life,t),migration:lerp(from.migration,to.migration,t)});state.scenario=local<.5?from.key:to.key;state.view=views[Math.floor(elapsed/6500)%views.length];render();state.raf=requestAnimationFrame(animate);
      }
      function manual(){if(state.auto)stopAuto();state.scenario='custom';render();}

      [birthInput,deathInput,lifeInput,migrationInput].forEach(function(input){input.addEventListener('input',manual);});
      Array.prototype.forEach.call(scenarioButtons,function(button){button.addEventListener('click',function(){if(state.auto)stopAuto();applyScenario(button.getAttribute('data-scenario')||'late-transition',true);});});
      Array.prototype.forEach.call(viewButtons,function(button){button.addEventListener('click',function(){if(state.auto)stopAuto();state.view=button.getAttribute('data-view')||'growth';render();});});
      labelToggle.addEventListener('click',function(){state.showLabels=!state.showLabels;render();});
      autoToggle.addEventListener('click',function(){if(state.auto){stopAuto();return;}state.auto=true;state.startedAt=0;state.raf=requestAnimationFrame(animate);render();});
      resetButton.addEventListener('click',function(){if(state.auto)stopAuto();setInputs(initial);state.scenario=initial.scenario;state.view='growth';state.showLabels=initial.showLabels;state.compareIndex=scenarios.indexOf(scenarioByKey(initial.scenario));render();});
      compareButton.addEventListener('click',function(){if(state.auto)stopAuto();state.compareIndex=(state.compareIndex+1)%scenarios.length;var next=scenarios[state.compareIndex];state.scenario=next.key;state.view=next.view;setInputs(next);render();});

      applyScenario(initial.scenario,false);
      setInputs(initial);
      render();
    })();
  ${SCRIPT_END}
</div>
`
}

export const GEOGRAPHY_LAB_TEMPLATES_HUMAN_POPULATION_TRANSITION:
GeographyLabTemplate[] = [
  {
    id: 'geography-population-growth-demographic-transition',
    group: '👥 人口、聚落与城市发展',
    name: '人口增长、人口结构与人口转变',
    emoji: '👥',
    desc: '调节出生率、死亡率、预期寿命和净迁移率，联动观察人口增长曲线、年龄金字塔、人口转变阶段与抚养负担。',
    params: [
      {
        key: 'scenario',
        label: '初始人口情境',
        type: 'select',
        options: [
          { label: '高位稳定', value: 'high-stationary' },
          { label: '早期转变·加速增长', value: 'early-transition' },
          { label: '后期转变·增长放缓', value: 'late-transition' },
          { label: '低位稳定', value: 'low-stationary' },
          { label: '老龄化与负增长', value: 'aging-decline' },
        ],
        defaultValue: 'late-transition',
      },
      {
        key: 'birthRate',
        label: '初始出生率（‰）',
        type: 'number',
        min: 5,
        max: 45,
        step: 1,
        defaultValue: 20,
        hint: '出生率是一定时期内出生人口与平均人口的比率，模板中仅作课堂示意。',
      },
      {
        key: 'deathRate',
        label: '初始死亡率（‰）',
        type: 'number',
        min: 4,
        max: 40,
        step: 1,
        defaultValue: 8,
        hint: '死亡率下降通常与医疗卫生、营养和生活条件改善有关。',
      },
      {
        key: 'lifeExpectancy',
        label: '平均预期寿命（岁）',
        type: 'number',
        min: 38,
        max: 88,
        step: 1,
        defaultValue: 72,
        hint: '预期寿命越高，模型中的老年人口比重通常越高。',
      },
      {
        key: 'netMigration',
        label: '净迁移率（‰）',
        type: 'number',
        min: -8,
        max: 8,
        step: 1,
        defaultValue: 1,
        hint: '迁入人口减去迁出人口后形成净迁移，对总人口增长产生补充影响。',
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],
    buildHTML: buildPopulationTransitionHTML,
  },
]
