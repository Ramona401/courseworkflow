/**
 * lifeScienceLabTemplatesPlantTropism.ts
 *
 * 平面生命科学实验室：植物向性运动。
 *
 * 教学目标：
 * 1. 观察茎的向光性，以及根和茎对重力刺激的不同反应；
 * 2. 理解单侧刺激造成器官两侧生长速度差异，最终表现为弯曲生长；
 * 3. 区分“生长方向发生改变”和“植物主动移动”；
 * 4. 用教学示意模型比较刺激方向、刺激强度和生长时间的影响。
 *
 * 教学边界：
 * 1. 向性运动是生长性运动，需要经过一段生长时间；
 * 2. 茎通常表现为向光性和背地性，根通常表现为向地性；
 * 3. 生长素作用具有器官差异：同一浓度范围对茎和根的影响可能不同；
 * 4. 本模板中的弯曲角度和两侧伸长量均为相对教学指标，不是真实实验测量值。
 *
 * 工程约束：
 * 1. 纯 HTML + SVG + 原生 JavaScript，不依赖外部图片、脚本、样式或 CDN；
 * 2. 所有 DOM 查询均限定在 rootId 内，支持同页放置多个独立实例；
 * 3. 使用生命科学统一 .bl-* 布局协议，嵌入课件后自动转为底部课堂控制条；
 * 4. 模板只导出独立数组，聚合接入由后续批次完成。
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

function bool(
  params: Record<string, LifeScienceLabParamValue>,
  key: string,
  fallback: boolean,
): boolean {
  const value = params[key]

  return typeof value === 'boolean'
    ? value
    : fallback
}

function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

function tropismStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BBF7D0;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#DCFCE7,#FEFCE8);border-bottom:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:242px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:13px;overflow:auto;background:#FAFFF9;border-right:1px solid #BBF7D0}'
    + '#' + rootId + ' .bl-stage{min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:10px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:5px;font-size:12px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#16A34A;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#16A34A}'
    + '#' + rootId + ' .pt-subtitle{margin:7px 0;font-size:12px;font-weight:800;color:#166534}'
    + '#' + rootId + ' .pt-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .pt-buttons.two{grid-template-columns:repeat(2,1fr)}'
    + '#' + rootId + ' .pt-button{height:31px;padding:0 4px;border:1px solid #86EFAC;border-radius:8px;background:#fff;color:#166534;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pt-button.active{border-color:#16A34A;background:#DCFCE7;box-shadow:0 3px 9px rgba(22,163,74,.13)}'
    + '#' + rootId + ' .pt-toggle{width:100%;height:32px;margin-bottom:8px;border:0;border-radius:8px;background:linear-gradient(135deg,#4ADE80,#16A34A);color:#fff;font-size:11px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pt-toggle.off{background:#64748B}'
    + '#' + rootId + ' .pt-status{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin-bottom:8px}'
    + '#' + rootId + ' .pt-card{padding:7px;border:1px solid #BBF7D0;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .pt-card b{display:block;font-size:16px;color:#15803D}'
    + '#' + rootId + ' .pt-card span{font-size:10px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:9px 10px;border-radius:10px;background:#DCFCE7;color:#14532D;font-size:11.5px;line-height:1.5;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .pt-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow 1.4s linear infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_PLANT_TROPISM:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-plant-tropism',
    group: '🌱 植物生长与发育',
    name: '植物向性运动',
    emoji: '🌱',
    desc: '调节光源方向、重力方向、刺激强度和生长时间，观察茎的向光性及根、茎的向地性差异',
    params: [
      {
        key: 'lightDirection',
        label: '光源方向',
        type: 'number',
        min: 0,
        max: 3,
        step: 1,
        defaultValue: 0,
        hint: '0=左侧，1=右侧，2=上方，3=下方',
      },
      {
        key: 'gravityDirection',
        label: '重力方向',
        type: 'number',
        min: 0,
        max: 3,
        step: 1,
        defaultValue: 0,
        hint: '0=向下，1=向左，2=向上，3=向右',
      },
      {
        key: 'stimulusStrength',
        label: '刺激强度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
      },
      {
        key: 'growthTime',
        label: '生长时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 68,
      },
      {
        key: 'showAuxin',
        label: '显示生长素分布',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const lightDirection = num(params, 'lightDirection', 0)
      const gravityDirection = num(params, 'gravityDirection', 0)
      const stimulusStrength = num(params, 'stimulusStrength', 72)
      const growthTime = num(params, 'growthTime', 68)
      const showAuxin = bool(params, 'showAuxin', true)

      return `
<div id="${rootId}">
${tropismStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🌱 植物向性运动</div>
    <div class="bl-note">弯曲角度与伸长量为相对教学指标</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>光源方向</span>
          <span class="bl-value" data-light-value></span>
        </div>
        <input data-light type="range" min="0" max="3" step="1" value="${n(lightDirection)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>重力方向</span>
          <span class="bl-value" data-gravity-value></span>
        </div>
        <input data-gravity type="range" min="0" max="3" step="1" value="${n(gravityDirection)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>刺激强度</span>
          <span class="bl-value" data-strength-value></span>
        </div>
        <input data-strength type="range" min="0" max="100" step="1" value="${n(stimulusStrength)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>生长时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(growthTime)}">
      </div>

      <div class="pt-subtitle">刺激类型</div>

      <div class="pt-buttons">
        <button type="button" class="pt-button active" data-mode="phototropism">向光性</button>
        <button type="button" class="pt-button" data-mode="gravitropism">向地性</button>
        <button type="button" class="pt-button" data-mode="combined">综合刺激</button>
      </div>

      <div class="pt-subtitle">重点观察器官</div>

      <div class="pt-buttons two">
        <button type="button" class="pt-button active" data-organ="stem">观察茎</button>
        <button type="button" class="pt-button" data-organ="root">观察根</button>
      </div>

      <button type="button" class="pt-toggle${showAuxin ? '' : ' off'}" data-auxin>
        ${showAuxin ? '生长素分布：显示' : '生长素分布：隐藏'}
      </button>

      <div class="pt-status">
        <div class="pt-card">
          <b data-angle></b>
          <span>相对弯曲角度</span>
        </div>

        <div class="pt-card">
          <b data-response></b>
          <span>当前主要反应</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg viewBox="0 0 680 414" aria-label="植物向性运动互动示意图">
        <defs>
          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>

          <marker id="${rootId}-arrow-gray" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#475569"/>
          </marker>

          <linearGradient id="${rootId}-soil" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#FDE68A"/>
            <stop offset="100%" stop-color="#D97706"/>
          </linearGradient>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#14532D" flood-opacity=".12"/>
          </filter>
        </defs>

        <rect width="680" height="414" fill="#FFFFFF"/>

        <text x="24" y="35" data-title font-size="26" font-weight="900" fill="#166534"></text>
        <text x="24" y="64" data-summary font-size="14" font-weight="800" fill="#475569"></text>

        <g data-light-source></g>
        <g data-gravity-arrow></g>

        <g filter="url(#${rootId}-shadow)">
          <path d="M275 302 H405 L390 382 H290Z" fill="#FDBA74" stroke="#C2410C" stroke-width="5"/>
          <ellipse cx="340" cy="302" rx="66" ry="18" fill="url(#${rootId}-soil)" stroke="#B45309" stroke-width="4"/>
        </g>

        <path data-stem d="" fill="none" stroke="#16A34A" stroke-width="18" stroke-linecap="round"/>
        <path data-stem-core d="" fill="none" stroke="#86EFAC" stroke-width="6" stroke-linecap="round"/>

        <path data-root d="" fill="none" stroke="#92400E" stroke-width="14" stroke-linecap="round"/>
        <path data-root-core d="" fill="none" stroke="#FCD34D" stroke-width="4" stroke-linecap="round"/>

        <g data-leaves></g>
        <g data-growth-zone></g>
        <g data-auxin-dots></g>
        <g data-response-arrow></g>

        <g transform="translate(478 82)">
          <rect x="0" y="0" width="174" height="154" rx="18" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>
          <text x="87" y="24" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">器官两侧相对伸长</text>

          <text x="32" y="52" data-side-a-label text-anchor="middle" font-size="11" font-weight="800" fill="#64748B"></text>
          <text x="89" y="52" data-side-b-label text-anchor="middle" font-size="11" font-weight="800" fill="#64748B"></text>

          <rect x="18" y="62" width="28" height="70" rx="7" fill="#E2E8F0"/>
          <rect x="75" y="62" width="28" height="70" rx="7" fill="#E2E8F0"/>

          <rect data-bar-a x="18" y="97" width="28" height="35" rx="7" fill="#4ADE80"/>
          <rect data-bar-b x="75" y="97" width="28" height="35" rx="7" fill="#22C55E"/>

          <text x="32" y="147" data-side-a-value text-anchor="middle" font-size="11" font-weight="900" fill="#166534"></text>
          <text x="89" y="147" data-side-b-value text-anchor="middle" font-size="11" font-weight="900" fill="#166534"></text>

          <path data-bend-arrow d="M118 116 Q145 88 155 62" fill="none" stroke="#16A34A" stroke-width="4" marker-end="url(#${rootId}-arrow-green)"/>
          <text x="137" y="143" text-anchor="middle" font-size="10.5" font-weight="800" fill="#166534">伸长差导致弯曲</text>
        </g>

        <text x="478" y="267" data-stage-note font-size="13" font-weight="900" fill="#166534"></text>
        <text x="478" y="290" data-auxin-note font-size="12" font-weight="800" fill="#64748B"></text>

        <g transform="translate(478 316)">
          <rect x="0" y="0" width="174" height="63" rx="14" fill="#FEFCE8" stroke="#FDE68A" stroke-width="2"/>
          <text x="87" y="21" text-anchor="middle" font-size="12" font-weight="900" fill="#854D0E">科学边界</text>
          <text x="87" y="40" text-anchor="middle" font-size="10.5" font-weight="800" fill="#713F12">向性是生长方向改变</text>
          <text x="87" y="55" text-anchor="middle" font-size="10.5" font-weight="800" fill="#713F12">不是植物主动移动</text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var light=root.querySelector('[data-light]');
    var gravity=root.querySelector('[data-gravity]');
    var strength=root.querySelector('[data-strength]');
    var time=root.querySelector('[data-time]');

    var lightValue=root.querySelector('[data-light-value]');
    var gravityValue=root.querySelector('[data-gravity-value]');
    var strengthValue=root.querySelector('[data-strength-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var organButtons=root.querySelectorAll('[data-organ]');
    var auxinButton=root.querySelector('[data-auxin]');

    var angleText=root.querySelector('[data-angle]');
    var responseText=root.querySelector('[data-response]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');
    var auxinNote=root.querySelector('[data-auxin-note]');

    var lightSource=root.querySelector('[data-light-source]');
    var gravityArrow=root.querySelector('[data-gravity-arrow]');
    var stem=root.querySelector('[data-stem]');
    var stemCore=root.querySelector('[data-stem-core]');
    var rootPath=root.querySelector('[data-root]');
    var rootCore=root.querySelector('[data-root-core]');
    var leaves=root.querySelector('[data-leaves]');
    var growthZone=root.querySelector('[data-growth-zone]');
    var auxinDots=root.querySelector('[data-auxin-dots]');
    var responseArrow=root.querySelector('[data-response-arrow]');

    var sideALabel=root.querySelector('[data-side-a-label]');
    var sideBLabel=root.querySelector('[data-side-b-label]');
    var sideAValue=root.querySelector('[data-side-a-value]');
    var sideBValue=root.querySelector('[data-side-b-value]');
    var barA=root.querySelector('[data-bar-a]');
    var barB=root.querySelector('[data-bar-b]');
    var bendArrow=root.querySelector('[data-bend-arrow]');

    var mode='phototropism';
    var organ='stem';
    var showAuxin=${showAuxin ? 'true' : 'false'};

    var lightDirections=[
      {name:'左侧',x:-1,y:0,sourceX:40,sourceY:170,targetX:250,targetY:170},
      {name:'右侧',x:1,y:0,sourceX:440,sourceY:170,targetX:370,targetY:170},
      {name:'上方',x:0,y:-1,sourceX:340,sourceY:78,targetX:340,targetY:138},
      {name:'下方',x:0,y:1,sourceX:340,sourceY:388,targetX:340,targetY:312}
    ];

    var gravityDirections=[
      {name:'向下',x:0,y:1,startX:448,startY:82,endX:448,endY:138},
      {name:'向左',x:-1,y:0,startX:458,startY:110,endX:406,endY:110},
      {name:'向上',x:0,y:-1,startX:448,startY:138,endX:448,endY:82},
      {name:'向右',x:1,y:0,startX:406,startY:110,endX:458,endY:110}
    ];

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function normalize(vector){
      var length=Math.sqrt(vector.x*vector.x+vector.y*vector.y);

      if(length<.001){
        return {x:0,y:0};
      }

      return {
        x:vector.x/length,
        y:vector.y/length
      };
    }

    function vectorForStem(modeName,lightVector,gravityVector){
      if(modeName==='phototropism'){
        return normalize(lightVector);
      }

      if(modeName==='gravitropism'){
        return normalize({
          x:-gravityVector.x,
          y:-gravityVector.y
        });
      }

      return normalize({
        x:lightVector.x*.58-gravityVector.x*.42,
        y:lightVector.y*.58-gravityVector.y*.42
      });
    }

    function vectorForRoot(modeName,gravityVector){
      if(modeName==='phototropism'){
        return {x:0,y:0};
      }

      return normalize(gravityVector);
    }

    function lightHTML(direction){
      var beamWidth=36+Number(strength.value)*.45;

      return ''
        +'<circle cx="'+direction.sourceX+'" cy="'+direction.sourceY+'" r="25" fill="#FACC15" stroke="#F59E0B" stroke-width="4"/>'
        +'<g stroke="#F59E0B" stroke-linecap="round">'
        +'<line x1="'+(direction.sourceX-38)+'" y1="'+direction.sourceY+'" x2="'+(direction.sourceX-54)+'" y2="'+direction.sourceY+'" stroke-width="4"/>'
        +'<line x1="'+(direction.sourceX+38)+'" y1="'+direction.sourceY+'" x2="'+(direction.sourceX+54)+'" y2="'+direction.sourceY+'" stroke-width="4"/>'
        +'<line x1="'+direction.sourceX+'" y1="'+(direction.sourceY-38)+'" x2="'+direction.sourceX+'" y2="'+(direction.sourceY-54)+'" stroke-width="4"/>'
        +'<line x1="'+direction.sourceX+'" y1="'+(direction.sourceY+38)+'" x2="'+direction.sourceX+'" y2="'+(direction.sourceY+54)+'" stroke-width="4"/>'
        +'</g>'
        +'<path class="pt-flow" d="M'+direction.sourceX+' '+direction.sourceY
        +' L'+direction.targetX+' '+direction.targetY
        +'" fill="none" stroke="#FACC15" stroke-width="'+beamWidth.toFixed(1)
        +'" opacity=".18" marker-end="url(#${rootId}-arrow-gray)"/>'
        +'<text x="'+direction.sourceX+'" y="'+(direction.sourceY+45)
        +'" text-anchor="middle" font-size="12" font-weight="900" fill="#A16207">单侧光</text>';
    }

    function gravityHTML(direction){
      return ''
        +'<path d="M'+direction.startX+' '+direction.startY
        +' L'+direction.endX+' '+direction.endY
        +'" fill="none" stroke="#475569" stroke-width="5" marker-end="url(#${rootId}-arrow-gray)"/>'
        +'<text x="448" y="62" text-anchor="middle" font-size="12" font-weight="900" fill="#475569">重力方向</text>';
    }

    function update(){
      var lightIndex=clamp(Math.round(Number(light.value)),0,3);
      var gravityIndex=clamp(Math.round(Number(gravity.value)),0,3);
      var strengthLevel=Number(strength.value);
      var timeLevel=Number(time.value);

      var lightDirection=lightDirections[lightIndex];
      var gravityDirection=gravityDirections[gravityIndex];
      var growthFactor=clamp(strengthLevel/100*timeLevel/100,0,1);

      lightValue.textContent=lightDirection.name;
      gravityValue.textContent=gravityDirection.name;
      strengthValue.textContent=strengthLevel.toFixed(0)+'%';
      timeValue.textContent=timeLevel.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      for(var j=0;j<organButtons.length;j++){
        organButtons[j].classList.toggle(
          'active',
          organButtons[j].getAttribute('data-organ')===organ
        );
      }

      auxinButton.textContent=showAuxin
        ?'生长素分布：显示'
        :'生长素分布：隐藏';

      auxinButton.classList.toggle('off',!showAuxin);

      var stemVector=vectorForStem(
        mode,
        lightDirection,
        gravityDirection
      );

      var rootVector=vectorForRoot(
        mode,
        gravityDirection
      );

      var stemDisplacement={
        x:stemVector.x*92*growthFactor,
        y:stemVector.y*58*growthFactor
      };

      var rootDisplacement={
        x:rootVector.x*74*growthFactor,
        y:rootVector.y*58*growthFactor
      };

      var stemEnd={
        x:clamp(340+stemDisplacement.x,112,450),
        y:clamp(124+stemDisplacement.y,88,236)
      };

      var stemControl1={
        x:340+stemDisplacement.x*.12,
        y:254+stemDisplacement.y*.08
      };

      var stemControl2={
        x:340+stemDisplacement.x*.52,
        y:184+stemDisplacement.y*.42
      };

      var stemD='M340 302 C'
        +stemControl1.x.toFixed(1)+' '+stemControl1.y.toFixed(1)+' '
        +stemControl2.x.toFixed(1)+' '+stemControl2.y.toFixed(1)+' '
        +stemEnd.x.toFixed(1)+' '+stemEnd.y.toFixed(1);

      stem.setAttribute('d',stemD);
      stemCore.setAttribute('d',stemD);

      var rootEnd={
        x:clamp(340+rootDisplacement.x,238,442),
        y:clamp(372+rootDisplacement.y*.62,312,398)
      };

      var rootControl1={
        x:340+rootDisplacement.x*.12,
        y:326+rootDisplacement.y*.08
      };

      var rootControl2={
        x:340+rootDisplacement.x*.58,
        y:350+rootDisplacement.y*.35
      };

      var rootD='M340 302 C'
        +rootControl1.x.toFixed(1)+' '+rootControl1.y.toFixed(1)+' '
        +rootControl2.x.toFixed(1)+' '+rootControl2.y.toFixed(1)+' '
        +rootEnd.x.toFixed(1)+' '+rootEnd.y.toFixed(1);

      rootPath.setAttribute('d',rootD);
      rootCore.setAttribute('d',rootD);

      var leafAngle=Math.atan2(
        stemEnd.y-stemControl2.y,
        stemEnd.x-stemControl2.x
      )*180/Math.PI;

      leaves.innerHTML=''
        +'<ellipse cx="'+(stemEnd.x-18)+'" cy="'+(stemEnd.y+8)
        +'" rx="29" ry="15" fill="#4ADE80" stroke="#15803D" stroke-width="4" transform="rotate('
        +(leafAngle-28).toFixed(1)+' '+(stemEnd.x-18)+' '+(stemEnd.y+8)+')"/>'
        +'<ellipse cx="'+(stemEnd.x+18)+'" cy="'+(stemEnd.y+8)
        +'" rx="29" ry="15" fill="#22C55E" stroke="#15803D" stroke-width="4" transform="rotate('
        +(leafAngle+28).toFixed(1)+' '+(stemEnd.x+18)+' '+(stemEnd.y+8)+')"/>'
        +'<circle cx="'+stemEnd.x+'" cy="'+stemEnd.y+'" r="11" fill="#84CC16" stroke="#3F6212" stroke-width="3"/>';

      lightSource.innerHTML=lightHTML(lightDirection);
      gravityArrow.innerHTML=gravityHTML(gravityDirection);

      var selectedVector=organ==='stem'
        ?stemVector
        :rootVector;

      var selectedEnd=organ==='stem'
        ?stemEnd
        :rootEnd;

      var responseLength=Math.sqrt(
        selectedVector.x*selectedVector.x
        +selectedVector.y*selectedVector.y
      );

      var angle=Math.round(
        Math.atan2(selectedVector.y,selectedVector.x)*180/Math.PI
      );

      if(responseLength<.01 || growthFactor<.03){
        angle=0;
      }

      angleText.textContent=Math.abs(angle)+'°';

      var growthZoneHTML='';

      if(organ==='stem'){
        growthZoneHTML='<circle cx="'+stemEnd.x+'" cy="'+stemEnd.y
          +'" r="24" fill="none" stroke="#16A34A" stroke-width="5" stroke-dasharray="5 5"/>';
      }else{
        growthZoneHTML='<circle cx="'+rootEnd.x+'" cy="'+rootEnd.y
          +'" r="22" fill="none" stroke="#B45309" stroke-width="5" stroke-dasharray="5 5"/>';
      }

      growthZone.innerHTML=growthZoneHTML;

      var arrowStartX=organ==='stem'?340:340;
      var arrowStartY=organ==='stem'?224:330;
      var arrowEndX=arrowStartX+selectedVector.x*70*growthFactor;
      var arrowEndY=arrowStartY+selectedVector.y*70*growthFactor;

      responseArrow.innerHTML=responseLength<.01 || growthFactor<.03
        ?''
        :'<path d="M'+arrowStartX+' '+arrowStartY
          +' L'+arrowEndX.toFixed(1)+' '+arrowEndY.toFixed(1)
          +'" fill="none" stroke="'+(organ==='stem'?'#16A34A':'#B45309')
          +'" stroke-width="5" marker-end="url(#${rootId}-arrow-green)"/>';

      var difference=clamp(growthFactor*42,0,42);
      var sideA=50+difference;
      var sideB=50-difference;

      var sideAName='背刺激侧';
      var sideBName='受刺激侧';
      var responseName='';
      var note='';
      var explanation='';

      if(mode==='phototropism'){
        if(organ==='stem'){
          responseName='茎向光弯曲';
          sideAName='背光侧';
          sideBName='向光侧';
          note='背光侧伸长较快';
          explanation='单侧光照下，茎背光侧生长素相对较多，促进该侧细胞伸长，使茎向光弯曲。';
        }else{
          responseName='根反应较弱';
          sideAName='根一侧';
          sideBName='根另一侧';
          sideA=50;
          sideB=50;
          note='本模型不强调根的向光反应';
          explanation='本模板主要用茎演示向光性。根的生长方向主要结合重力刺激观察。';
        }
      }else if(mode==='gravitropism'){
        if(organ==='stem'){
          responseName='茎背地生长';
          sideAName='下侧';
          sideBName='上侧';
          note='茎下侧伸长较快';
          explanation='重力刺激下，茎下侧生长素相对较多并促进伸长，使茎表现为背地性。';
        }else{
          responseName='根向地生长';
          sideAName='上侧';
          sideBName='下侧';
          note='根上侧伸长较快';
          explanation='重力刺激下，根下侧生长素相对较多，但根对生长素更敏感，下侧伸长受抑，根因而向地弯曲。';
        }
      }else{
        if(organ==='stem'){
          responseName='茎综合响应';
          sideAName='较快侧';
          sideBName='较慢侧';
          note='光和重力共同影响茎';
          explanation='茎同时受到单侧光和重力刺激，最终方向由两类刺激的共同作用决定。';
        }else{
          responseName='根向地生长';
          sideAName='上侧';
          sideBName='下侧';
          note='根主要响应重力';
          explanation='综合情境中，本模型把根的主要方向反应简化为向地性。';
        }
      }

      responseText.textContent=responseName;
      stageNote.textContent=note;
      auxinNote.textContent=showAuxin
        ?'紫色小点表示相对生长素分布'
        :'已隐藏生长素分布';

      sideALabel.textContent=sideAName;
      sideBLabel.textContent=sideBName;
      sideAValue.textContent=sideA.toFixed(0);
      sideBValue.textContent=sideB.toFixed(0);

      var barAHeight=clamp(sideA*.64,12,70);
      var barBHeight=clamp(sideB*.64,12,70);

      barA.setAttribute('y',String(132-barAHeight));
      barA.setAttribute('height',String(barAHeight));
      barB.setAttribute('y',String(132-barBHeight));
      barB.setAttribute('height',String(barBHeight));

      bendArrow.setAttribute(
        'stroke',
        organ==='stem'?'#16A34A':'#B45309'
      );

      var auxinHTML='';

      if(showAuxin){
        var dotCount=Math.floor(4+growthFactor*8);

        for(var q=0;q<dotCount;q++){
          var progress=(q+1)/(dotCount+1);
          var px;
          var py;
          var sideOffset=(q%2===0?1:-1)*(6+growthFactor*8);

          if(organ==='stem'){
            px=340+(stemEnd.x-340)*progress+sideOffset;
            py=302+(stemEnd.y-302)*progress;
          }else{
            px=340+(rootEnd.x-340)*progress+sideOffset;
            py=302+(rootEnd.y-302)*progress;
          }

          auxinHTML+='<circle cx="'+px.toFixed(1)+'" cy="'+py.toFixed(1)
            +'" r="'+(4+q%2)+'" fill="#8B5CF6" opacity=".78"/>';
        }
      }

      auxinDots.innerHTML=auxinHTML;

      var modeTitle=mode==='phototropism'
        ?'单侧光引起的向光性'
        :mode==='gravitropism'
          ?'重力引起的向地性或背地性'
          :'光和重力共同作用';

      title.textContent=modeTitle;
      summary.textContent='当前观察：'+(organ==='stem'?'茎':'根')
        +'；刺激强度 '+strengthLevel.toFixed(0)
        +'%，生长时间 '+timeLevel.toFixed(0)+'%';

      var timeCondition=timeLevel<18
        ?'生长时间很短，尚未形成明显弯曲。'
        :strengthLevel<18
          ?'刺激较弱，两侧生长差异不明显。'
          :'刺激经过一段生长时间后，两侧伸长差异逐渐表现为弯曲生长。';

      result.innerHTML=explanation
        +'<br>'+timeCondition
        +' 向性是植物器官生长方向发生变化，不是植物主动移动。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<organButtons.length;j++){
      organButtons[j].onclick=function(){
        organ=this.getAttribute('data-organ');
        update();
      };
    }

    auxinButton.onclick=function(){
      showAuxin=!showAuxin;
      update();
    };

    light.oninput=update;
    gravity.oninput=update;
    strength.oninput=update;
    time.oninput=update;

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
