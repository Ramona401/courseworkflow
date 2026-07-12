/**
 * lifeScienceLabTemplatesHumanPlacentaFetalDevelopment.ts
 *
 * 平面生命科学实验室：胎盘与胎儿发育。
 *
 * 教学目标：
 * 1. 认识胎盘、脐带、子宫内膜和胎儿之间的结构关系；
 * 2. 理解母体血液和胎儿血液通常不直接混合；
 * 3. 观察氧气、营养物质、二氧化碳和代谢废物的交换方向；
 * 4. 理解脐静脉和脐动脉运输方向及其所含物质的相对差异；
 * 5. 观察受精后不同周数的胚胎和胎儿发育阶段；
 * 6. 区分胚胎期和胎儿期，避免把所有早期发育阶段都称为胎儿。
 *
 * 科学边界：
 * 1. 胎盘由胎儿来源的绒毛膜等结构与母体来源的子宫内膜组织共同构成；
 * 2. 母体血液和胎儿血液通常由胎盘屏障分隔，并不直接混合；
 * 3. 氧气和多种营养物质可由母体侧进入胎儿侧，
 *    二氧化碳和部分代谢废物可由胎儿侧进入母体侧；
 * 4. 胎盘具有选择性交换和屏障作用，但不是能够阻挡所有物质的绝对屏障；
 * 5. 一条脐静脉通常将含氧和营养相对较多的血液由胎盘输向胎儿；
 * 6. 两条脐动脉通常将含氧相对较少、代谢废物相对较多的血液由胎儿输向胎盘；
 * 7. 本模板使用受精后的发育周数，不等同于临床通常从末次月经开始计算的孕周；
 * 8. 受精后前八周通常称为胚胎期，受精后第九周起进入胎儿期；
 * 9. 图中胎盘交换效率、血流和发育指标均为相对教学模型；
 * 10. 本模板只用于生物学教学，不用于医学诊断、孕期评估或个体健康判断。
 *
 * 工程约束：
 * 1. 纯HTML、SVG和原生JavaScript，不依赖外部图片、脚本、样式或CDN；
 * 2. 所有DOM查询均限定在rootId内部，支持同页多个独立实例；
 * 3. 使用生命科学统一.bl-*布局协议；
 * 4. 本文件只导出独立模板数组，聚合入口由后续批次统一接入。
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

function placentaStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #FBCFE8;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#FCE7F3,#E0F2FE);border-bottom:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FFF9FC;border-right:1px solid #FBCFE8}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#DB2777;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#EC4899}'
    + '#' + rootId + ' .pf-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#9D174D}'
    + '#' + rootId + ' .pf-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .pf-stages{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .pf-button{min-height:29px;padding:3px;border:1px solid #F9A8D4;border-radius:8px;background:#fff;color:#9D174D;font-size:9.2px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .pf-button.active{border-color:#DB2777;background:#FCE7F3;box-shadow:0 3px 9px rgba(219,39,119,.13)}'
    + '#' + rootId + ' .pf-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#F472B6,#DB2777);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .pf-toggle.off{background:#64748B}'
    + '#' + rootId + ' .pf-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .pf-card{padding:6px 3px;border:1px solid #FBCFE8;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .pf-card b{display:block;font-size:13px;color:#BE185D}'
    + '#' + rootId + ' .pf-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#FCE7F3;color:#831843;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .pf-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--pf-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .pf-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_HUMAN_PLACENTA_FETAL_DEVELOPMENT:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-placenta-fetal-development',
    group: '🧑 人体生殖与发育',
    name: '胎盘与胎儿发育',
    emoji: '🤰',
    desc: '调节发育周数、母体氧气和营养供应、胎盘交换及脐带血流，观察物质交换和胚胎胎儿发育',
    params: [
      {
        key: 'developmentWeek',
        label: '受精后发育周数',
        type: 'number',
        min: 3,
        max: 38,
        step: 1,
        defaultValue: 24,
      },
      {
        key: 'maternalOxygen',
        label: '母体侧氧气供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 86,
      },
      {
        key: 'maternalNutrition',
        label: '母体侧营养供应',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 80,
      },
      {
        key: 'placentalExchange',
        label: '胎盘相对交换状态',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 84,
      },
      {
        key: 'umbilicalFlow',
        label: '脐带相对血流',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const developmentWeek = num(params, 'developmentWeek', 24)
      const maternalOxygen = num(params, 'maternalOxygen', 86)
      const maternalNutrition = num(params, 'maternalNutrition', 80)
      const placentalExchange = num(params, 'placentalExchange', 84)
      const umbilicalFlow = num(params, 'umbilicalFlow', 82)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${placentaStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🤰 胎盘、脐带与胚胎胎儿发育</div>
    <div class="bl-note">本模板使用受精后的发育周数</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>受精后发育周数</span>
          <span class="bl-value" data-week-value></span>
        </div>
        <input
          data-week
          type="range"
          min="3"
          max="38"
          step="1"
          value="${n(developmentWeek)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>母体侧氧气供应</span>
          <span class="bl-value" data-oxygen-value></span>
        </div>
        <input
          data-oxygen
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(maternalOxygen)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>母体侧营养供应</span>
          <span class="bl-value" data-nutrition-value></span>
        </div>
        <input
          data-nutrition
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(maternalNutrition)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>胎盘相对交换状态</span>
          <span class="bl-value" data-exchange-value></span>
        </div>
        <input
          data-exchange
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(placentalExchange)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>脐带相对血流</span>
          <span class="bl-value" data-flow-value></span>
        </div>
        <input
          data-flow
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(umbilicalFlow)}"
        >
      </div>

      <div class="pf-subtitle">观察方式</div>

      <div class="pf-buttons">
        <button
          type="button"
          class="pf-button active"
          data-mode="exchange"
        >胎盘交换</button>

        <button
          type="button"
          class="pf-button"
          data-mode="circulation"
        >脐带运输</button>

        <button
          type="button"
          class="pf-button"
          data-mode="timeline"
        >发育阶段</button>
      </div>

      <div class="pf-subtitle">快速查看阶段</div>

      <div class="pf-stages">
        <button type="button" class="pf-button" data-stage-week="4">第4周</button>
        <button type="button" class="pf-button" data-stage-week="8">第8周</button>
        <button type="button" class="pf-button" data-stage-week="12">第12周</button>
        <button type="button" class="pf-button active" data-stage-week="24">第24周</button>
        <button type="button" class="pf-button" data-stage-week="38">第38周</button>
      </div>

      <button
        type="button"
        class="pf-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="pf-toggle"
        data-auto
      >周数推进：运行中</button>

      <div class="pf-status">
        <div class="pf-card">
          <b data-period></b>
          <span>发育时期</span>
        </div>

        <div class="pf-card">
          <b data-exchange-index></b>
          <span>交换指数</span>
        </div>

        <div class="pf-card">
          <b data-growth-state></b>
          <span>当前状态</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="胎盘、脐带与胚胎胎儿发育互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-red"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <marker
            id="${rootId}-arrow-blue"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#2563EB"/>
          </marker>

          <marker
            id="${rootId}-arrow-green"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#831843"
              flood-opacity=".13"
            />
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text
          x="24"
          y="36"
          data-title
          font-size="26"
          font-weight="900"
          fill="#9D174D"
        ></text>

        <text
          x="24"
          y="65"
          data-summary
          font-size="14"
          font-weight="800"
          fill="#475569"
        ></text>

        <g data-dynamic></g>
        <g data-labels></g>

        <g transform="translate(518 337)">
          <rect
            width="216"
            height="66"
            rx="15"
            fill="#FFF1F2"
            stroke="#FBCFE8"
            stroke-width="2"
          />

          <text
            x="108"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#9D174D"
          >关键边界</text>

          <text
            x="108"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#831843"
          >母体血与胎儿血通常不直接混合</text>

          <text
            x="108"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#831843"
          >胎盘不是绝对屏障</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#9D174D"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var weekInput=root.querySelector('[data-week]');
    var oxygenInput=root.querySelector('[data-oxygen]');
    var nutritionInput=root.querySelector('[data-nutrition]');
    var exchangeInput=root.querySelector('[data-exchange]');
    var flowInput=root.querySelector('[data-flow]');

    var weekValue=root.querySelector('[data-week-value]');
    var oxygenValue=root.querySelector('[data-oxygen-value]');
    var nutritionValue=root.querySelector('[data-nutrition-value]');
    var exchangeValue=root.querySelector('[data-exchange-value]');
    var flowValue=root.querySelector('[data-flow-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var stageButtons=root.querySelectorAll('[data-stage-week]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var autoButton=root.querySelector('[data-auto]');

    var periodText=root.querySelector('[data-period]');
    var exchangeIndexText=root.querySelector('[data-exchange-index]');
    var growthStateText=root.querySelector('[data-growth-state]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector('[data-stage-note]');

    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='exchange';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(!automatic || !document.body.contains(root)){
        return;
      }

      timer=window.setTimeout(function(){
        var next=Number(weekInput.value)+1;

        weekInput.value=String(
          next>38?3:next
        );

        update();
        schedule();
      },900);
    }

    function resolvePeriod(week){
      if(week<=4){
        return {
          short:'胚胎早期',
          name:'胚胎早期与胎盘建立',
          note:'胚胎完成着床后，绒毛膜等结构与子宫内膜逐步建立更密切的联系。'
        };
      }

      if(week<=8){
        return {
          short:'胚胎期',
          name:'胚胎器官形成期',
          note:'胚胎主要器官系统的基本结构逐步形成，此时仍属于胚胎期。'
        };
      }

      if(week<=16){
        return {
          short:'胎儿早期',
          name:'胎儿早期发育',
          note:'受精后第九周起进入胎儿期，已有结构继续生长和分化。'
        };
      }

      if(week<=28){
        return {
          short:'胎儿中期',
          name:'胎儿快速生长阶段',
          note:'胎儿身体比例继续变化，多种器官和生理功能逐渐发育。'
        };
      }

      return {
        short:'胎儿晚期',
        name:'胎儿晚期生长与成熟',
        note:'胎儿体重和器官成熟度继续增加，为出生后的独立生活作准备。'
      };
    }

    function fetusShape(
      cx,
      cy,
      scale,
      color
    ){
      return ''
        +'<g transform="translate('+cx+' '+cy+') scale('+scale+')">'
        +'<circle cx="0" cy="-55" r="28" fill="'+color+'" stroke="#BE185D" stroke-width="5"/>'
        +'<path d="M-10 -27 C-45 -8 -51 39 -20 64 C10 87 50 67 47 32 C44 1 22 -22 -10 -27Z" fill="'+color+'" stroke="#BE185D" stroke-width="5"/>'
        +'<path d="M-26 5 Q-61 28 -43 49" fill="none" stroke="#BE185D" stroke-width="8" stroke-linecap="round"/>'
        +'<path d="M20 18 Q55 35 43 57" fill="none" stroke="#BE185D" stroke-width="8" stroke-linecap="round"/>'
        +'<path d="M-9 62 Q-22 92 -5 108 M25 58 Q45 88 30 108" fill="none" stroke="#BE185D" stroke-width="9" stroke-linecap="round"/>'
        +'</g>';
    }

    function renderExchange(
      oxygen,
      nutrition,
      exchangeIndex
    ){
      var oxygenCount=Math.floor(
        3+oxygen/10
      );

      var nutritionCount=Math.floor(
        3+nutrition/11
      );

      var wasteCount=Math.floor(
        4+exchangeIndex/13
      );

      var particles='';

      for(var i=0;i<oxygenCount;i++){
        var y=124+(i%6)*31;
        var x=128+(i%3)*18;

        particles+='<circle cx="'+x+'" cy="'+y+'" r="6" fill="#EF4444" opacity=".82"/>'
          +'<text x="'+(x+9)+'" y="'+(y+4)+'" font-size="9" font-weight="900" fill="#991B1B">O₂</text>';
      }

      for(var j=0;j<nutritionCount;j++){
        var ny=118+(j%6)*32;
        var nx=195+(j%3)*18;

        particles+='<circle cx="'+nx+'" cy="'+ny+'" r="6" fill="#F59E0B" opacity=".82"/>'
          +'<text x="'+(nx+9)+'" y="'+(ny+4)+'" font-size="9" font-weight="900" fill="#92400E">N</text>';
      }

      var waste='';

      for(var k=0;k<wasteCount;k++){
        var wy=128+(k%6)*29;
        var wx=433+(k%3)*17;

        waste+='<circle cx="'+wx+'" cy="'+wy+'" r="6" fill="#60A5FA" opacity=".82"/>'
          +'<text x="'+(wx+9)+'" y="'+(wy+4)+'" font-size="9" font-weight="900" fill="#1D4ED8">CO₂</text>';
      }

      var arrowWidth=3+exchangeIndex/20;

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="54" y="92" width="226" height="248" rx="28" fill="#FFF1F2" stroke="#FDA4AF" stroke-width="4"/>'
        +'<text x="167" y="121" text-anchor="middle" font-size="16" font-weight="900" fill="#9F1239">母体血液空间</text>'
        +'<path d="M84 150 C126 128 162 175 205 151 C240 132 258 166 250 206 C242 250 209 288 167 303 C125 287 92 251 83 210 C75 183 73 164 84 150Z" fill="#FECACA" stroke="#DC2626" stroke-width="5"/>'
        +particles
        +'</g>'
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M319 106 C344 128 330 166 359 184 C385 201 371 239 397 258 C421 276 410 314 437 332" fill="none" stroke="#F9A8D4" stroke-width="54" stroke-linecap="round"/>'
        +'<path d="M319 106 C344 128 330 166 359 184 C385 201 371 239 397 258 C421 276 410 314 437 332" fill="none" stroke="#BE185D" stroke-width="5" stroke-linecap="round"/>'
        +'<path d="M322 111 C347 135 334 167 361 185 C386 204 375 239 399 258 C422 277 414 309 437 327" fill="none" stroke="#2563EB" stroke-width="13" stroke-linecap="round"/>'
        +'</g>'
        +'<text x="370" y="87" text-anchor="middle" font-size="15" font-weight="900" fill="#BE185D">胎盘绒毛及胎儿毛细血管</text>'
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="475" y="92" width="220" height="248" rx="28" fill="#EFF6FF" stroke="#93C5FD" stroke-width="4"/>'
        +'<text x="585" y="121" text-anchor="middle" font-size="16" font-weight="900" fill="#1D4ED8">胎儿血液侧</text>'
        +'<path d="M516 165 C563 129 641 143 658 202 C673 258 623 304 566 293 C511 283 483 208 516 165Z" fill="#DBEAFE" stroke="#2563EB" stroke-width="5"/>'
        +waste
        +'</g>'
        +'<path class="pf-flow" d="M238 172 C289 151 312 154 339 176" fill="none" stroke="#DC2626" stroke-width="'+arrowWidth+'" marker-end="url(#${rootId}-arrow-red)"/>'
        +'<text x="288" y="140" text-anchor="middle" font-size="11" font-weight="900" fill="#991B1B">氧气</text>'
        +'<path class="pf-flow" d="M246 230 C293 215 321 213 360 233" fill="none" stroke="#F59E0B" stroke-width="'+arrowWidth+'" marker-end="url(#${rootId}-arrow-red)"/>'
        +'<text x="300" y="203" text-anchor="middle" font-size="11" font-weight="900" fill="#92400E">营养物质</text>'
        +'<path class="pf-flow" d="M479 193 C435 185 414 184 384 199" fill="none" stroke="#2563EB" stroke-width="'+arrowWidth+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="441" y="164" text-anchor="middle" font-size="11" font-weight="900" fill="#1D4ED8">二氧化碳</text>'
        +'<path class="pf-flow" d="M484 264 C445 254 423 252 399 264" fill="none" stroke="#64748B" stroke-width="'+arrowWidth+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="450" y="237" text-anchor="middle" font-size="11" font-weight="900" fill="#475569">代谢废物</text>'
        +'<g transform="translate(70 354)">'
        +'<rect width="520" height="35" rx="12" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<text x="260" y="23" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">母体血和胎儿血通常由胎盘屏障分隔，不直接混合</text>'
        +'</g>';
    }

    function renderCirculation(
      week,
      flow,
      exchangeIndex
    ){
      var scale=.42+.48*(week-3)/35;
      var cordWidth=8+flow/12;

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<path d="M92 100 C46 149 51 273 116 327 C180 381 283 352 316 277 C349 201 313 111 233 83 C177 63 124 70 92 100Z" fill="#FCE7F3" stroke="#DB2777" stroke-width="6"/>'
        +'<path d="M74 119 C111 94 162 96 194 128 C145 150 118 193 118 242 C89 220 72 181 74 119Z" fill="#FDA4AF" stroke="#BE185D" stroke-width="5"/>'
        +'<text x="135" y="86" text-anchor="middle" font-size="15" font-weight="900" fill="#9D174D">胎盘</text>'
        +'</g>'
        +'<path class="pf-flow" d="M178 198 C257 159 319 172 378 222 C416 255 444 259 472 247" fill="none" stroke="#DC2626" stroke-width="'+cordWidth+'" marker-end="url(#${rootId}-arrow-red)"/>'
        +'<path class="pf-flow" d="M474 267 C434 285 394 275 361 247 C303 197 254 194 181 224" fill="none" stroke="#2563EB" stroke-width="'+Math.max(6,cordWidth-3)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<path class="pf-flow" d="M474 281 C429 302 387 293 349 261 C295 215 251 213 180 239" fill="none" stroke="#2563EB" stroke-width="'+Math.max(6,cordWidth-3)+'" marker-end="url(#${rootId}-arrow-blue)"/>'
        +'<text x="318" y="154" text-anchor="middle" font-size="13" font-weight="900" fill="#991B1B">脐静脉：胎盘→胎儿</text>'
        +'<text x="337" y="320" text-anchor="middle" font-size="13" font-weight="900" fill="#1D4ED8">两条脐动脉：胎儿→胎盘</text>'
        +'<g transform="translate(546 223)" filter="url(#${rootId}-shadow)">'
        +'<circle r="'+(112*scale)+'" fill="#FFF7ED" stroke="#F59E0B" stroke-width="6"/>'
        +fetusShape(
          0,
          5,
          scale,
          '#FBCFE8'
        )
        +'</g>'
        +'<g transform="translate(496 87)">'
        +'<rect width="220" height="88" rx="17" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="2"/>'
        +'<text x="110" y="25" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">脐带运输方向</text>'
        +'<circle cx="24" cy="47" r="7" fill="#DC2626"/>'
        +'<text x="40" y="52" font-size="11" font-weight="900" fill="#991B1B">一条脐静脉：氧气和营养较多</text>'
        +'<circle cx="24" cy="70" r="7" fill="#2563EB"/>'
        +'<text x="40" y="75" font-size="11" font-weight="900" fill="#1D4ED8">两条脐动脉：废物相对较多</text>'
        +'</g>'
        +'<g transform="translate(57 352)">'
        +'<rect width="440" height="37" rx="12" fill="#EFF6FF" stroke="#BFDBFE" stroke-width="2"/>'
        +'<text x="220" y="24" text-anchor="middle" font-size="12" font-weight="900" fill="#1E3A8A">血管名称由血液离开心脏或流向心脏的方向命名，不按含氧量命名</text>'
        +'</g>'
        +'<text x="555" y="386" text-anchor="middle" font-size="12" font-weight="900" fill="#475569">相对交换指数 '+exchangeIndex.toFixed(0)+'</text>';
    }

    function renderTimeline(
      week,
      period
    ){
      var stages=[
        {
          week:4,
          label:'第4周',
          title:'胚胎早期',
          note:'着床后胎盘结构逐步建立',
          color:'#F9A8D4'
        },
        {
          week:8,
          label:'第8周',
          title:'胚胎期末',
          note:'主要器官基本结构逐步形成',
          color:'#F472B6'
        },
        {
          week:12,
          label:'第12周',
          title:'胎儿早期',
          note:'已有结构继续生长和分化',
          color:'#C084FC'
        },
        {
          week:24,
          label:'第24周',
          title:'胎儿中期',
          note:'身体和多种器官继续发育',
          color:'#60A5FA'
        },
        {
          week:38,
          label:'第38周',
          title:'胎儿晚期',
          note:'生长和器官成熟继续进行',
          color:'#34D399'
        }
      ];

      var html='';

      for(var i=0;i<stages.length;i++){
        var item=stages[i];
        var x=82+i*143;
        var active=Math.abs(week-item.week)<=4
          ||(
            i===0
            &&week<8
          )
          ||(
            i===4
            &&week>31
          );

        var complete=week>=item.week;

        html+='<g>'
          +'<circle cx="'+x+'" cy="214" r="'+(active?53:43)
          +'" fill="'+(complete?item.color:'#F8FAFC')
          +'" stroke="'+(active?'#BE185D':'#CBD5E1')
          +'" stroke-width="'+(active?6:3)+'"/>'
          +'<text x="'+x+'" y="207" text-anchor="middle" font-size="13" font-weight="900" fill="'
          +(complete?'#FFFFFF':'#475569')
          +'">'+item.label+'</text>'
          +'<text x="'+x+'" y="228" text-anchor="middle" font-size="11" font-weight="900" fill="'
          +(complete?'#FFFFFF':'#475569')
          +'">'+item.title+'</text>'
          +'<text x="'+x+'" y="290" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'
          +item.note
          +'</text>'
          +'</g>';

        if(i<stages.length-1){
          html+='<path class="pf-flow" d="M'
            +(x+56)+' 214 H'
            +(x+88)
            +'" fill="none" stroke="#DB2777" stroke-width="4" marker-end="url(#${rootId}-arrow-red)"/>';
        }
      }

      var progressX=82
        +(week-3)/35*572;

      return ''
        +'<line x1="82" y1="112" x2="654" y2="112" stroke="#E2E8F0" stroke-width="12" stroke-linecap="round"/>'
        +'<line x1="82" y1="112" x2="'+progressX.toFixed(1)+'" y2="112" stroke="#EC4899" stroke-width="12" stroke-linecap="round"/>'
        +'<circle cx="'+progressX.toFixed(1)+'" cy="112" r="13" fill="#BE185D" stroke="#FFFFFF" stroke-width="4"/>'
        +'<text x="82" y="88" font-size="12" font-weight="900" fill="#64748B">受精后第3周</text>'
        +'<text x="654" y="88" text-anchor="end" font-size="12" font-weight="900" fill="#64748B">受精后第38周</text>'
        +'<text x="'+progressX.toFixed(1)+'" y="145" text-anchor="middle" font-size="15" font-weight="900" fill="#9D174D">当前第 '+week+' 周</text>'
        +html
        +'<g transform="translate(135 332)">'
        +'<rect width="470" height="55" rx="16" fill="#FAF5FF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="235" y="23" text-anchor="middle" font-size="14" font-weight="900" fill="#6D28D9">'
        +period.name
        +'</text>'
        +'<text x="235" y="43" text-anchor="middle" font-size="11.5" font-weight="800" fill="#475569">'
        +period.note
        +'</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='exchange'){
        labels.innerHTML=''
          +'<path d="M187 302 L102 330" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="31" y="337" font-size="13" font-weight="900" fill="#991B1B">母体血液空间</text>'
          +'<path d="M372 171 L461 118" stroke="#BE185D" stroke-width="2.5"/>'
          +'<text x="469" y="116" font-size="13" font-weight="900" fill="#9D174D">胎盘绒毛</text>'
          +'<path d="M385 230 L487 238" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="495" y="243" font-size="13" font-weight="900" fill="#1D4ED8">胎儿毛细血管</text>';
        return;
      }

      if(modeName==='circulation'){
        labels.innerHTML=''
          +'<path d="M136 160 L72 127" stroke="#BE185D" stroke-width="2.5"/>'
          +'<text x="28" y="123" font-size="13" font-weight="900" fill="#9D174D">胎盘</text>'
          +'<path d="M324 201 L369 162" stroke="#DC2626" stroke-width="2.5"/>'
          +'<text x="378" y="160" font-size="13" font-weight="900" fill="#991B1B">脐静脉</text>'
          +'<path d="M357 275 L414 310" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="423" y="316" font-size="13" font-weight="900" fill="#1D4ED8">脐动脉</text>'
          +'<path d="M565 248 L661 270" stroke="#BE185D" stroke-width="2.5"/>'
          +'<text x="669" y="276" font-size="13" font-weight="900" fill="#9D174D">胚胎或胎儿</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M238 214 L238 166" stroke="#BE185D" stroke-width="2.5"/>'
        +'<text x="178" y="159" font-size="13" font-weight="900" fill="#9D174D">前八周：胚胎期</text>'
        +'<path d="M443 214 L443 166" stroke="#2563EB" stroke-width="2.5"/>'
        +'<text x="394" y="159" font-size="13" font-weight="900" fill="#1D4ED8">第九周起：胎儿期</text>';
    }

    function update(){
      var week=clamp(
        Math.round(Number(weekInput.value)),
        3,
        38
      );

      var oxygen=Number(oxygenInput.value);
      var nutrition=Number(nutritionInput.value);
      var exchange=Number(exchangeInput.value);
      var flow=Number(flowInput.value);

      weekValue.textContent='第 '+week+' 周';
      oxygenValue.textContent=oxygen.toFixed(0)+'%';
      nutritionValue.textContent=nutrition.toFixed(0)+'%';
      exchangeValue.textContent=exchange.toFixed(0)+'%';
      flowValue.textContent=flow.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      for(var j=0;j<stageButtons.length;j++){
        stageButtons[j].classList.toggle(
          'active',
          Number(stageButtons[j].getAttribute('data-stage-week'))===week
        );
      }

      labelToggle.textContent=showLabels
        ?'结构标注：显示'
        :'结构标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      autoButton.textContent=automatic
        ?'周数推进：运行中'
        :'周数推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var period=resolvePeriod(week);

      var exchangeIndex=100
        *Math.pow(
          oxygen/100
          *nutrition/100
          *(.20+.80*exchange/100)
          *(.25+.75*flow/100),
          .25
        );

      exchangeIndex=clamp(
        exchangeIndex,
        0,
        100
      );

      var growthSupport=100
        *Math.sqrt(
          exchangeIndex/100
          *(.35+.65*nutrition/100)
        );

      growthSupport=clamp(
        growthSupport,
        0,
        100
      );

      var growthState=growthSupport>72
        ?'支持较好'
        :growthSupport>42
          ?'支持一般'
          :'支持较低';

      periodText.textContent=period.short;
      exchangeIndexText.textContent=exchangeIndex.toFixed(0);
      growthStateText.textContent=growthState;

      root.style.setProperty(
        '--pf-speed',
        clamp(
          2.5-exchangeIndex/70,
          .65,
          2.4
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='exchange'){
        title.textContent='胎盘中的母胎物质交换';
        summary.textContent='观察氧气、营养物质、二氧化碳和代谢废物的交换方向。';

        dynamic.innerHTML=renderExchange(
          oxygen,
          nutrition,
          exchangeIndex
        );

        stageNote.textContent=
          '母体血液和胎儿血液通常由胎盘屏障分隔，通过胎盘结构进行物质交换。';

        renderLabels(mode);
      }else if(mode==='circulation'){
        title.textContent='胎盘—脐带—胎儿运输通路';
        summary.textContent='观察一条脐静脉和两条脐动脉的血流方向。';

        dynamic.innerHTML=renderCirculation(
          week,
          flow,
          exchangeIndex
        );

        stageNote.textContent=
          '脐静脉由胎盘流向胎儿；脐动脉由胎儿流向胎盘。';

        renderLabels(mode);
      }else{
        title.textContent='胚胎期与胎儿期发育时间线';
        summary.textContent='当前为受精后第 '
          +week
          +' 周：'
          +period.name
          +'。';

        dynamic.innerHTML=renderTimeline(
          week,
          period
        );

        stageNote.textContent=
          '本模板使用受精后的发育周数，临床孕周的计算起点通常不同。';

        renderLabels(mode);
      }

      var condition=
        '当前母体侧氧气、营养、胎盘交换和脐带血流处于相对协调状态。';

      if(oxygen<20){
        condition=
          '母体侧氧气供应较低，进入胎儿侧的相对氧气量明显下降。';
      }else if(nutrition<20){
        condition=
          '母体侧营养供应较低，胎儿侧获得的相对营养物质减少。';
      }else if(exchange<20){
        condition=
          '胎盘相对交换状态较低，双向物质交换受到明显限制。';
      }else if(flow<20){
        condition=
          '脐带相对血流较低，胎盘与胎儿之间的运输效率下降。';
      }

      var principle=mode==='exchange'
        ?'氧气和多种营养物质可由母体侧进入胎儿侧；二氧化碳和部分代谢废物可由胎儿侧进入母体侧。'
        :mode==='circulation'
          ?'一条脐静脉通常将含氧和营养相对较多的血液输向胎儿，两条脐动脉通常将代谢废物相对较多的血液输向胎盘。'
          :'受精后前八周通常称为胚胎期，受精后第九周起进入胎儿期。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 胎盘具有选择性交换和屏障作用，但不是能够阻挡所有物质的绝对屏障。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<stageButtons.length;j++){
      stageButtons[j].onclick=function(){
        weekInput.value=this.getAttribute('data-stage-week');
        mode='timeline';
        automatic=false;
        update();
        schedule();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    autoButton.onclick=function(){
      automatic=!automatic;
      update();
      schedule();
    };

    weekInput.oninput=update;
    oxygenInput.oninput=update;
    nutritionInput.oninput=update;
    exchangeInput.oninput=update;
    flowInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
