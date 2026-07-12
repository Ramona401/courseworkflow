/**
 * lifeScienceLabTemplatesExcretionWaterSaltAdh.ts
 *
 * 平面生命科学实验室：水盐平衡与抗利尿激素调节。
 *
 * 教学目标：
 * 1. 观察饮水、食盐摄入和出汗失水对细胞外液渗透状态的影响；
 * 2. 理解下丘脑渗透压感受器感受细胞外液渗透状态变化；
 * 3. 理解抗利尿激素由下丘脑有关神经分泌细胞合成，
 *    经神经垂体释放进入血液；
 * 4. 观察抗利尿激素促进集合管水通透性增加和水重吸收增强；
 * 5. 理解抗利尿激素水平升高时，尿量通常减少、尿液相对浓缩；
 * 6. 区分抗利尿激素对水重吸收的调节与醛固酮对钠离子重吸收的调节；
 * 7. 理解饮水行为、肾脏调节和激素调节共同维持水盐平衡；
 * 8. 理解水盐平衡调节体现负反馈调节。
 *
 * 科学边界：
 * 1. 细胞外液渗透压升高可刺激下丘脑渗透压感受器；
 * 2. 下丘脑有关神经分泌细胞合成抗利尿激素，
 *    抗利尿激素经神经垂体释放进入血液；
 * 3. 抗利尿激素主要作用于肾脏远端肾小管和集合管，
 *    促进水通道蛋白插入细胞膜，提高水通透性；
 * 4. 抗利尿激素促进水重吸收，通常使尿量减少、尿液相对浓缩；
 * 5. 细胞外液渗透压降低时，抗利尿激素释放通常减少，
 *    集合管水重吸收减弱，尿量相对增加；
 * 6. 醛固酮主要促进远端肾小管和集合管对钠离子的重吸收，
 *    并促进钾离子分泌；
 * 7. 水可随钠离子重吸收发生被动移动，
 *    但醛固酮和抗利尿激素的直接作用对象与调节重点不同；
 * 8. 水盐平衡还受到血容量、血压、肾素—血管紧张素系统、
 *    心房钠尿肽、交感神经和饮水行为等多种因素影响；
 * 9. 本模型不模拟所有激素之间的完整相互作用；
 * 10. 图中的渗透状态、激素水平、尿量和尿液浓缩程度均为相对教学指标；
 * 11. 本模板只用于生物学教学，
 *     不用于脱水判断、电解质判断、尿量评价或医学诊断。
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

function waterSaltAdhStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #BAE6FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0F2FE,#ECFDF5);border-bottom:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#F8FDFF;border-right:1px solid #BAE6FD}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:8px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#0284C7;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#0EA5E9}'
    + '#' + rootId + ' .ws-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#075985}'
    + '#' + rootId + ' .ws-buttons{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .ws-button{min-height:30px;padding:3px;border:1px solid #7DD3FC;border-radius:8px;background:#fff;color:#075985;font-size:10px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .ws-button.active{border-color:#0284C7;background:#E0F2FE;box-shadow:0 3px 9px rgba(2,132,199,.13)}'
    + '#' + rootId + ' .ws-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#38BDF8,#0284C7);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .ws-toggle.off{background:#64748B}'
    + '#' + rootId + ' .ws-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin:7px 0}'
    + '#' + rootId + ' .ws-card{padding:6px 3px;border:1px solid #BAE6FD;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .ws-card b{display:block;font-size:13px;color:#0369A1}'
    + '#' + rootId + ' .ws-card span{font-size:9px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#E0F2FE;color:#0C4A6E;font-size:10.8px;line-height:1.46;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .ws-flow{stroke-dasharray:8 7;animation:' + rootId + '-flow var(--ws-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .ws-pulse{animation:' + rootId + '-pulse 1.6s ease-in-out infinite}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-30}}'
    + '@keyframes ' + rootId + '-pulse{0%,100%{opacity:.42}50%{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_EXCRETION_WATER_SALT_ADH:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-water-salt-adh-regulation',
    group: '💧 排泄与内环境稳态',
    name: '水盐平衡与抗利尿激素调节',
    emoji: '💧',
    desc: '调节饮水、食盐摄入、出汗失水、抗利尿激素反应、醛固酮活性和过程时间，观察水盐平衡负反馈调节',
    params: [
      {
        key: 'waterIntake',
        label: '饮水相对水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 52,
      },
      {
        key: 'saltLoad',
        label: '食盐摄入水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 44,
      },
      {
        key: 'sweatLoss',
        label: '出汗失水水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 38,
      },
      {
        key: 'adhResponse',
        label: '抗利尿激素反应能力',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 82,
      },
      {
        key: 'aldosteroneActivity',
        label: '醛固酮调节活性',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
      {
        key: 'processTime',
        label: '调节过程时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 54,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const waterIntake = num(params, 'waterIntake', 52)
      const saltLoad = num(params, 'saltLoad', 44)
      const sweatLoss = num(params, 'sweatLoss', 38)
      const adhResponse = num(params, 'adhResponse', 82)
      const aldosteroneActivity = num(
        params,
        'aldosteroneActivity',
        58,
      )
      const processTime = num(params, 'processTime', 54)
      const showLabels = bool(params, 'showLabels', true)

      return `
<div id="${rootId}">
${waterSaltAdhStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">💧 水盐平衡与抗利尿激素调节</div>
    <div class="bl-note">渗透状态、激素和尿液指标均为相对教学值</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>饮水相对水平</span>
          <span class="bl-value" data-water-intake-value></span>
        </div>
        <input
          data-water-intake
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(waterIntake)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>食盐摄入水平</span>
          <span class="bl-value" data-salt-value></span>
        </div>
        <input
          data-salt
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(saltLoad)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>出汗失水水平</span>
          <span class="bl-value" data-sweat-value></span>
        </div>
        <input
          data-sweat
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(sweatLoss)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>抗利尿激素反应能力</span>
          <span class="bl-value" data-adh-value></span>
        </div>
        <input
          data-adh
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(adhResponse)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>醛固酮调节活性</span>
          <span class="bl-value" data-aldosterone-value></span>
        </div>
        <input
          data-aldosterone
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(aldosteroneActivity)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>调节过程时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input
          data-time
          type="range"
          min="0"
          max="100"
          step="1"
          value="${n(processTime)}"
        >
      </div>

      <div class="ws-subtitle">观察方式</div>

      <div class="ws-buttons">
        <button
          type="button"
          class="ws-button active"
          data-mode="sensing"
        >渗透状态感受</button>

        <button
          type="button"
          class="ws-button"
          data-mode="adh"
        >抗利尿激素作用</button>

        <button
          type="button"
          class="ws-button"
          data-mode="aldosterone"
        >醛固酮与钠钾</button>

        <button
          type="button"
          class="ws-button"
          data-mode="feedback"
        >负反馈与情境</button>
      </div>

      <button
        type="button"
        class="ws-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <button
        type="button"
        class="ws-toggle"
        data-auto
      >调节推进：运行中</button>

      <div class="ws-status">
        <div class="ws-card">
          <b data-osmotic-index></b>
          <span>渗透状态</span>
        </div>

        <div class="ws-card">
          <b data-adh-index></b>
          <span>ADH信号</span>
        </div>

        <div class="ws-card">
          <b data-urine-volume></b>
          <span>尿量指数</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="水盐平衡与抗利尿激素调节互动示意图"
      >
        <defs>
          <marker
            id="${rootId}-arrow-blue"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker
            id="${rootId}-arrow-purple"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
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

          <marker
            id="${rootId}-arrow-orange"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#075985"
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
          fill="#075985"
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
            fill="#F0F9FF"
            stroke="#BAE6FD"
            stroke-width="2"
          />

          <text
            x="108"
            y="21"
            text-anchor="middle"
            font-size="12"
            font-weight="900"
            fill="#075985"
          >关键边界</text>

          <text
            x="108"
            y="40"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#0C4A6E"
          >ADH主要调节水通透性</text>

          <text
            x="108"
            y="56"
            text-anchor="middle"
            font-size="10.5"
            font-weight="800"
            fill="#0C4A6E"
          >醛固酮主要调节钠钾转运</text>
        </g>

        <text
          x="24"
          y="407"
          data-stage-note
          font-size="14"
          font-weight="900"
          fill="#075985"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var waterIntakeInput=root.querySelector(
      '[data-water-intake]'
    );
    var saltInput=root.querySelector('[data-salt]');
    var sweatInput=root.querySelector('[data-sweat]');
    var adhInput=root.querySelector('[data-adh]');
    var aldosteroneInput=root.querySelector(
      '[data-aldosterone]'
    );
    var timeInput=root.querySelector('[data-time]');

    var waterIntakeValue=root.querySelector(
      '[data-water-intake-value]'
    );
    var saltValue=root.querySelector(
      '[data-salt-value]'
    );
    var sweatValue=root.querySelector(
      '[data-sweat-value]'
    );
    var adhValue=root.querySelector(
      '[data-adh-value]'
    );
    var aldosteroneValue=root.querySelector(
      '[data-aldosterone-value]'
    );
    var timeValue=root.querySelector(
      '[data-time-value]'
    );

    var modeButtons=root.querySelectorAll(
      '[data-mode]'
    );
    var labelToggle=root.querySelector(
      '[data-label-toggle]'
    );
    var autoButton=root.querySelector(
      '[data-auto]'
    );

    var osmoticText=root.querySelector(
      '[data-osmotic-index]'
    );
    var adhText=root.querySelector(
      '[data-adh-index]'
    );
    var urineVolumeText=root.querySelector(
      '[data-urine-volume]'
    );
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var stageNote=root.querySelector(
      '[data-stage-note]'
    );
    var dynamic=root.querySelector('[data-dynamic]');
    var labels=root.querySelector('[data-labels]');

    var mode='sensing';
    var showLabels=${showLabels ? 'true' : 'false'};
    var automatic=true;
    var timer=null;

    function clamp(value,min,max){
      return Math.max(
        min,
        Math.min(max,value)
      );
    }

    function schedule(){
      if(timer){
        window.clearTimeout(timer);
        timer=null;
      }

      if(
        !automatic
        ||!document.body.contains(root)
      ){
        return;
      }

      timer=window.setTimeout(function(){
        var next=Number(timeInput.value)+2;

        timeInput.value=String(
          next>100?0:next
        );

        update();
        schedule();
      },800);
    }

    function waterDrop(
      x,
      y,
      scale,
      opacity,
      color
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<path d="M0 -17 C-12 -2 -15 7 -15 15 C-15 31 15 31 15 15 C15 7 12 -2 0 -17Z" fill="'+color+'" stroke="#0369A1" stroke-width="2.5"/>'
        +'</g>';
    }

    function saltParticle(
      x,
      y,
      scale,
      opacity
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<rect x="-8" y="-8" width="16" height="16" rx="3" fill="#C4B5FD" stroke="#6D28D9" stroke-width="2"/>'
        +'<text x="0" y="3" text-anchor="middle" font-size="7" font-weight="900" fill="#4C1D95">Na</text>'
        +'</g>';
    }

    function hormoneDrop(
      x,
      y,
      scale,
      opacity,
      label
    ){
      return ''
        +'<g transform="translate('+x+' '+y+') scale('+scale+')" opacity="'+opacity+'">'
        +'<path d="M0 -21 C-16 -1 -18 11 -18 20 C-18 39 18 39 18 20 C18 11 16 -1 0 -21Z" fill="#EDE9FE" stroke="#7C3AED" stroke-width="3"/>'
        +'<text x="0" y="21" text-anchor="middle" font-size="8" font-weight="900" fill="#5B21B6">'+label+'</text>'
        +'</g>';
    }

    function hypothalamusShape(
      x,
      y,
      intensity
    ){
      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<ellipse cx="'+x+'" cy="'+y+'" rx="77" ry="57" fill="#EDE9FE" stroke="#7C3AED" stroke-width="5"/>'
        +'<path d="M'+(x-48)+' '+y
        +' Q'+x+' '+(y-48)+' '+(x+48)+' '+y
        +' Q'+x+' '+(y+42)+' '+(x-48)+' '+y
        +'Z" fill="#C4B5FD"/>'
        +'<circle cx="'+x+'" cy="'+y+'" r="20" fill="#8B5CF6" opacity=".72"/>'
        +'<circle class="ws-pulse" cx="'+x+'" cy="'+y+'" r="'+(66+intensity*.15)
        +'" fill="none" stroke="#A855F7" stroke-width="4" stroke-dasharray="8 7" opacity="'+(.22+.70*intensity/100)+'"/>'
        +'</g>';
    }

    function collectingDuctShape(){
      var channels='';

      for(var i=0;i<8;i++){
        var y=126+i*29;

        channels+='<rect x="367" y="'+y
          +'" width="22" height="12" rx="5" fill="#7DD3FC" stroke="#0369A1" stroke-width="2"/>';

        channels+='<rect x="457" y="'+y
          +'" width="22" height="12" rx="5" fill="#7DD3FC" stroke="#0369A1" stroke-width="2"/>';
      }

      return ''
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="360" y="98" width="126" height="252" rx="35" fill="#F0F9FF" stroke="#0284C7" stroke-width="6"/>'
        +'<rect x="397" y="105" width="52" height="237" rx="23" fill="#DBEAFE" stroke="#38BDF8" stroke-width="3"/>'
        +channels
        +'</g>';
    }

    function renderSensing(
      water,
      salt,
      sweat,
      osmotic,
      adhSignal,
      progress
    ){
      var waterCount=Math.floor(
        3+water/13
      );
      var saltCount=Math.floor(
        2+salt/13
      );
      var sweatCount=Math.floor(
        2+sweat/15
      );

      var fluid='';
      var sweatDrops='';
      var hormone='';

      for(var i=0;i<waterCount;i++){
        var wx=67+(i%6)*42;
        var wy=149+Math.floor(i/6)*45;

        fluid+=waterDrop(
          wx,
          wy,
          .55,
          .72,
          '#38BDF8'
        );
      }

      for(var j=0;j<saltCount;j++){
        var sx=70+(j%6)*42;
        var sy=238+Math.floor(j/6)*39;

        fluid+=saltParticle(
          sx,
          sy,
          .75,
          .78
        );
      }

      for(var k=0;k<sweatCount;k++){
        var tx=85+k*32;
        var ty=102-(k%2)*14;

        sweatDrops+=waterDrop(
          tx,
          ty,
          .42,
          .45+.45*sweat/100,
          '#7DD3FC'
        );
      }

      var hormoneCount=Math.floor(
        2+adhSignal/17
      );

      for(var q=0;q<hormoneCount;q++){
        hormone+=hormoneDrop(
          520+(q%4)*42,
          222+Math.floor(q/4)*48,
          .52,
          .40+.55*progress,
          'ADH'
        );
      }

      return ''
        +'<rect x="27" y="87" width="295" height="272" rx="25" fill="#F0F9FF" stroke="#BAE6FD" stroke-width="3"/>'
        +'<text x="174" y="118" text-anchor="middle" font-size="15" font-weight="900" fill="#075985">细胞外液水盐状态</text>'
        +'<rect x="50" y="132" width="248" height="184" rx="20" fill="#DBEAFE" stroke="#60A5FA" stroke-width="4" opacity=".78"/>'
        +fluid
        +sweatDrops
        +'<text x="174" y="343" text-anchor="middle" font-size="12" font-weight="900" fill="#0369A1">相对渗透状态 '+osmotic.toFixed(0)+'</text>'
        +'<path class="ws-flow" d="M318 212 H365" fill="none" stroke="#7C3AED" stroke-width="5" marker-end="url(#${rootId}-arrow-purple)"/>'
        +hypothalamusShape(
          445,
          176,
          osmotic
        )
        +'<text x="445" y="251" text-anchor="middle" font-size="13" font-weight="900" fill="#5B21B6">下丘脑渗透压感受器</text>'
        +'<path class="ws-flow" d="M489 221 C514 233 526 245 539 266" fill="none" stroke="#7C3AED" stroke-width="'+(3+adhSignal/22)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<g transform="translate(520 91)">'
        +'<rect width="206" height="238" rx="22" fill="#FAF5FF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="103" y="27" text-anchor="middle" font-size="14" font-weight="900" fill="#5B21B6">ADH合成与释放</text>'
        +'<ellipse cx="103" cy="86" rx="57" ry="35" fill="#EDE9FE" stroke="#7C3AED" stroke-width="4"/>'
        +'<text x="103" y="82" text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6">下丘脑神经</text>'
        +'<text x="103" y="99" text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6">分泌细胞合成</text>'
        +'<path class="ws-flow" d="M103 123 V153" fill="none" stroke="#7C3AED" stroke-width="4" marker-end="url(#${rootId}-arrow-purple)"/>'
        +'<ellipse cx="103" cy="176" rx="53" ry="28" fill="#DBEAFE" stroke="#2563EB" stroke-width="4"/>'
        +'<text x="103" y="181" text-anchor="middle" font-size="11" font-weight="900" fill="#1D4ED8">神经垂体释放</text>'
        +hormone
        +'</g>';
    }

    function renderAdh(
      adhSignal,
      waterReabsorption,
      urineVolume,
      concentration,
      progress
    ){
      var waterDrops='';
      var urineDrops='';
      var hormones='';

      var channelActivity=clamp(
        adhSignal/100
        *(.20+.80*progress),
        0,
        1
      );

      var waterCount=Math.floor(
        3+waterReabsorption/10
      );

      for(var i=0;i<waterCount;i++){
        var y=126+(i%8)*28;
        var left=i%2===0;

        waterDrops+=waterDrop(
          left?337:510,
          y,
          .43,
          .42+.55*channelActivity,
          '#38BDF8'
        );

        waterDrops+='<path class="ws-flow" d="M'
          +(left?397:449)
          +' '+(y+5)
          +' H'
          +(left?343:503)
          +'" fill="none" stroke="#0284C7" stroke-width="'
          +(2.5+channelActivity*4)
          +'" marker-end="url(#${rootId}-arrow-blue)" opacity="'
          +(.30+.65*channelActivity)
          +'"/>';
      }

      var hormoneCount=Math.floor(
        2+adhSignal/14
      );

      for(var j=0;j<hormoneCount;j++){
        hormones+=hormoneDrop(
          130+(j%4)*48,
          126+Math.floor(j/4)*55,
          .53,
          .46+.48*adhSignal/100,
          'ADH'
        );
      }

      var urineCount=Math.floor(
        2+urineVolume/13
      );

      for(var k=0;k<urineCount;k++){
        urineDrops+=waterDrop(
          423+(k%3)*19,
          360+Math.floor(k/3)*15,
          .30,
          .40+.50*urineVolume/100,
          concentration>65?'#F59E0B':'#7DD3FC'
        );
      }

      return ''
        +'<rect x="27" y="87" width="281" height="272" rx="25" fill="#FAF5FF" stroke="#DDD6FE" stroke-width="3"/>'
        +'<text x="167" y="118" text-anchor="middle" font-size="15" font-weight="900" fill="#5B21B6">血液中的抗利尿激素</text>'
        +hormones
        +'<text x="167" y="330" text-anchor="middle" font-size="12" font-weight="900" fill="#5B21B6">相对ADH信号 '+adhSignal.toFixed(0)+'</text>'
        +'<path class="ws-flow" d="M305 217 H342" fill="none" stroke="#7C3AED" stroke-width="'+(3+adhSignal/21)+'" marker-end="url(#${rootId}-arrow-purple)"/>'
        +collectingDuctShape()
        +waterDrops
        +urineDrops
        +'<text x="423" y="86" text-anchor="middle" font-size="15" font-weight="900" fill="#075985">集合管</text>'
        +'<text x="423" y="382" text-anchor="middle" font-size="11" font-weight="900" fill="#0369A1">终尿流出</text>'
        +'<g transform="translate(535 94)">'
        +'<rect width="194" height="244" rx="21" fill="#F0FDF4" stroke="#A7F3D0" stroke-width="3"/>'
        +'<text x="97" y="28" text-anchor="middle" font-size="14" font-weight="900" fill="#047857">ADH作用结果</text>'
        +'<text x="17" y="62" font-size="11.5" font-weight="900" fill="#0369A1">水通道蛋白</text>'
        +'<text x="17" y="83" font-size="10.5" font-weight="800" fill="#475569">集合管膜水通透性增加</text>'
        +'<text x="17" y="116" font-size="11.5" font-weight="900" fill="#0369A1">水重吸收</text>'
        +'<text x="17" y="137" font-size="10.5" font-weight="800" fill="#475569">水由管腔进入组织液和血液</text>'
        +'<text x="17" y="170" font-size="11.5" font-weight="900" fill="#0369A1">尿液变化</text>'
        +'<text x="17" y="191" font-size="10.5" font-weight="800" fill="#475569">尿量通常减少</text>'
        +'<text x="17" y="211" font-size="10.5" font-weight="800" fill="#475569">尿液相对浓缩</text>'
        +'<text x="17" y="234" font-size="10.5" font-weight="900" fill="#166534">浓缩指数 '+concentration.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderAldosterone(
      aldosterone,
      sodiumRetention,
      potassiumSecretion,
      progress
    ){
      var hormones='';
      var sodium='';
      var potassium='';

      var hormoneCount=Math.floor(
        2+aldosterone/15
      );

      for(var i=0;i<hormoneCount;i++){
        hormones+=hormoneDrop(
          103+(i%4)*45,
          134+Math.floor(i/4)*53,
          .50,
          .43+.50*progress,
          'ALD'
        );
      }

      var sodiumCount=Math.floor(
        3+sodiumRetention/12
      );

      for(var j=0;j<sodiumCount;j++){
        var sy=125+(j%8)*28;

        sodium+=saltParticle(
          407,
          sy,
          .60,
          .55+.40*sodiumRetention/100
        );

        sodium+='<path class="ws-flow" d="M425 '
          +(sy+2)
          +' H520" fill="none" stroke="#16A34A" stroke-width="'
          +(2.5+sodiumRetention/30)
          +'" marker-end="url(#${rootId}-arrow-green)"/>';
      }

      var potassiumCount=Math.floor(
        2+potassiumSecretion/15
      );

      for(var k=0;k<potassiumCount;k++){
        var ky=144+(k%7)*31;

        potassium+='<g transform="translate(574 '+ky+')">'
          +'<circle r="8" fill="#FDE68A" stroke="#B45309" stroke-width="2"/>'
          +'<text x="0" y="3" text-anchor="middle" font-size="7" font-weight="900" fill="#92400E">K</text>'
          +'</g>';

        potassium+='<path class="ws-flow" d="M548 '
          +(ky+2)
          +' H447" fill="none" stroke="#F59E0B" stroke-width="'
          +(2.5+potassiumSecretion/32)
          +'" marker-end="url(#${rootId}-arrow-orange)"/>';
      }

      return ''
        +'<rect x="27" y="87" width="274" height="272" rx="25" fill="#FFF7ED" stroke="#FED7AA" stroke-width="3"/>'
        +'<text x="164" y="118" text-anchor="middle" font-size="15" font-weight="900" fill="#9A3412">血液中的醛固酮</text>'
        +hormones
        +'<text x="164" y="331" text-anchor="middle" font-size="12" font-weight="900" fill="#92400E">相对醛固酮活性 '+aldosterone.toFixed(0)+'</text>'
        +'<path class="ws-flow" d="M299 217 H342" fill="none" stroke="#F59E0B" stroke-width="'+(3+aldosterone/22)+'" marker-end="url(#${rootId}-arrow-orange)"/>'
        +'<g filter="url(#${rootId}-shadow)">'
        +'<rect x="348" y="99" width="105" height="250" rx="34" fill="#EFF6FF" stroke="#2563EB" stroke-width="6"/>'
        +'<rect x="374" y="108" width="54" height="232" rx="22" fill="#DBEAFE" stroke="#60A5FA" stroke-width="3"/>'
        +'<rect x="522" y="99" width="96" height="250" rx="30" fill="#FEE2E2" stroke="#DC2626" stroke-width="6"/>'
        +'</g>'
        +'<text x="401" y="88" text-anchor="middle" font-size="14" font-weight="900" fill="#1D4ED8">远端肾单位管腔</text>'
        +'<text x="570" y="88" text-anchor="middle" font-size="14" font-weight="900" fill="#991B1B">管周血液</text>'
        +sodium
        +potassium
        +'<g transform="translate(634 96)">'
        +'<rect width="102" height="247" rx="19" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="51" y="27" text-anchor="middle" font-size="12" font-weight="900" fill="#334155">作用重点</text>'
        +'<text x="13" y="61" font-size="10.5" font-weight="900" fill="#047857">Na⁺重吸收</text>'
        +'<text x="13" y="83" font-size="9.5" font-weight="800" fill="#475569">管腔 → 血液</text>'
        +'<text x="13" y="119" font-size="10.5" font-weight="900" fill="#92400E">K⁺分泌</text>'
        +'<text x="13" y="141" font-size="9.5" font-weight="800" fill="#475569">血液 → 管腔</text>'
        +'<text x="13" y="177" font-size="10.5" font-weight="900" fill="#0369A1">水的变化</text>'
        +'<text x="13" y="199" font-size="9.5" font-weight="800" fill="#475569">可随钠被动移动</text>'
        +'<text x="13" y="228" font-size="9.5" font-weight="900" fill="#64748B">不等同于ADH</text>'
        +'</g>';
    }

    function feedbackCard(
      x,
      titleText,
      line1,
      line2,
      active,
      color,
      fill
    ){
      return ''
        +'<g transform="translate('+x+' 99)">'
        +'<rect width="164" height="181" rx="20" fill="'+fill+'" stroke="'+color+'" stroke-width="'+(active?5:3)+'"/>'
        +'<circle cx="82" cy="48" r="28" fill="#FFFFFF" stroke="'+color+'" stroke-width="4"/>'
        +'<text x="82" y="55" text-anchor="middle" font-size="20" font-weight="900" fill="'+color+'">'
        +(active?'●':'○')
        +'</text>'
        +'<text x="82" y="99" text-anchor="middle" font-size="14" font-weight="900" fill="'+color+'">'+titleText+'</text>'
        +'<text x="82" y="128" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'+line1+'</text>'
        +'<text x="82" y="150" text-anchor="middle" font-size="10.5" font-weight="800" fill="#475569">'+line2+'</text>'
        +'</g>';
    }

    function renderFeedback(
      osmotic,
      adhSignal,
      urineVolume,
      concentration,
      water,
      salt,
      sweat
    ){
      var state='balanced';

      if(
        osmotic>66
        ||sweat>72
      ){
        state='dehydration';
      }else if(
        osmotic<35
        ||water>82
      ){
        state='overhydration';
      }else if(
        salt>72
        &&water<55
      ){
        state='highSalt';
      }

      return ''
        +feedbackCard(
          24,
          '相对脱水',
          '渗透状态升高',
          'ADH↑ 尿量↓',
          state==='dehydration',
          '#DC2626',
          '#FFF1F2'
        )
        +feedbackCard(
          205,
          '饮水较多',
          '渗透状态降低',
          'ADH↓ 尿量↑',
          state==='overhydration',
          '#0284C7',
          '#F0F9FF'
        )
        +feedbackCard(
          386,
          '盐负荷较高',
          '口渴和ADH增强',
          '促进保水',
          state==='highSalt',
          '#7C3AED',
          '#FAF5FF'
        )
        +feedbackCard(
          567,
          '接近稳态',
          '摄入与排出协调',
          '负反馈减小偏差',
          state==='balanced',
          '#16A34A',
          '#ECFDF5'
        )
        +'<g transform="translate(52 309)">'
        +'<rect width="645" height="66" rx="18" fill="#F8FAFC" stroke="#CBD5E1" stroke-width="3"/>'
        +'<text x="322" y="23" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">当前相对结果</text>'
        +'<text x="107" y="49" text-anchor="middle" font-size="11" font-weight="900" fill="#5B21B6">ADH '+adhSignal.toFixed(0)+'</text>'
        +'<text x="264" y="49" text-anchor="middle" font-size="11" font-weight="900" fill="#0369A1">尿量 '+urineVolume.toFixed(0)+'</text>'
        +'<text x="421" y="49" text-anchor="middle" font-size="11" font-weight="900" fill="#B45309">浓缩 '+concentration.toFixed(0)+'</text>'
        +'<text x="565" y="49" text-anchor="middle" font-size="11" font-weight="900" fill="#047857">渗透状态 '+osmotic.toFixed(0)+'</text>'
        +'</g>';
    }

    function renderLabels(modeName){
      if(!showLabels){
        labels.innerHTML='';
        return;
      }

      if(modeName==='sensing'){
        labels.innerHTML=''
          +'<path d="M445 119 L445 82" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="372" y="77" font-size="13" font-weight="900" fill="#5B21B6">渗透压感受器</text>'
          +'<path d="M623 268 L696 286" stroke="#7C3AED" stroke-width="2.5"/>'
          +'<text x="702" y="292" font-size="13" font-weight="900" fill="#5B21B6">进入血液的ADH</text>'
          +'<path d="M126 134 L81 99" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="26" y="94" font-size="13" font-weight="900" fill="#0369A1">体液水分</text>';
        return;
      }

      if(modeName==='adh'){
        labels.innerHTML=''
          +'<path d="M378 126 L321 95" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="221" y="91" font-size="13" font-weight="900" fill="#0369A1">水通道蛋白</text>'
          +'<path d="M514 215 L588 185" stroke="#0284C7" stroke-width="2.5"/>'
          +'<text x="596" y="187" font-size="13" font-weight="900" fill="#0369A1">水重吸收</text>'
          +'<path d="M423 347 L493 373" stroke="#F59E0B" stroke-width="2.5"/>'
          +'<text x="501" y="379" font-size="13" font-weight="900" fill="#92400E">终尿</text>';
        return;
      }

      if(modeName==='aldosterone'){
        labels.innerHTML=''
          +'<path d="M474 147 L520 116" stroke="#16A34A" stroke-width="2.5"/>'
          +'<text x="527" y="115" font-size="13" font-weight="900" fill="#047857">Na⁺重吸收</text>'
          +'<path d="M536 251 L482 283" stroke="#F59E0B" stroke-width="2.5"/>'
          +'<text x="381" y="290" font-size="13" font-weight="900" fill="#92400E">K⁺分泌</text>'
          +'<path d="M398 348 L350 379" stroke="#2563EB" stroke-width="2.5"/>'
          +'<text x="244" y="385" font-size="13" font-weight="900" fill="#1D4ED8">远端肾小管和集合管</text>';
        return;
      }

      labels.innerHTML=''
        +'<path d="M102 99 L102 76" stroke="#DC2626" stroke-width="2.5"/>'
        +'<text x="47" y="71" font-size="13" font-weight="900" fill="#991B1B">水分减少情境</text>'
        +'<path d="M649 99 L649 76" stroke="#16A34A" stroke-width="2.5"/>'
        +'<text x="586" y="71" font-size="13" font-weight="900" fill="#166534">负反馈稳态</text>';
    }

    function update(){
      var water=Number(
        waterIntakeInput.value
      );
      var salt=Number(saltInput.value);
      var sweat=Number(sweatInput.value);
      var adhCapacity=Number(adhInput.value);
      var aldosterone=Number(
        aldosteroneInput.value
      );
      var processTime=Number(timeInput.value);

      waterIntakeValue.textContent=
        water.toFixed(0)+'%';
      saltValue.textContent=
        salt.toFixed(0)+'%';
      sweatValue.textContent=
        sweat.toFixed(0)+'%';
      adhValue.textContent=
        adhCapacity.toFixed(0)+'%';
      aldosteroneValue.textContent=
        aldosterone.toFixed(0)+'%';
      timeValue.textContent=
        processTime.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute(
            'data-mode'
          )===mode
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
        ?'调节推进：运行中'
        :'调节推进：已暂停';

      autoButton.classList.toggle(
        'off',
        !automatic
      );

      var progress=processTime/100;

      var osmoticIndex=clamp(
        50
        +salt*.42
        +sweat*.33
        -water*.55,
        0,
        100
      );

      var osmoticStimulus=clamp(
        (osmoticIndex-32)/68,
        0,
        1
      );

      var adhSignal=100*clamp(
        osmoticStimulus
        *adhCapacity/100
        *(.18+.82*progress),
        0,
        1
      );

      var waterReabsorption=clamp(
        18
        +adhSignal*.72,
        0,
        100
      );

      var sodiumRetention=clamp(
        18
        +aldosterone
        *(.20+.80*progress)
        *.78,
        0,
        100
      );

      var potassiumSecretion=clamp(
        aldosterone
        *(.18+.82*progress)
        *.82,
        0,
        100
      );

      var urineVolume=clamp(
        92
        -waterReabsorption*.72
        +water*.18
        -sweat*.12,
        5,
        100
      );

      var concentration=clamp(
        18
        +waterReabsorption*.67
        +salt*.12
        -water*.08,
        8,
        100
      );

      var correctedOsmotic=clamp(
        osmoticIndex
        -waterReabsorption*.18
        -water*.10
        +sodiumRetention*.04,
        0,
        100
      );

      osmoticText.textContent=
        osmoticIndex.toFixed(0);
      adhText.textContent=
        adhSignal.toFixed(0);
      urineVolumeText.textContent=
        urineVolume.toFixed(0);

      root.style.setProperty(
        '--ws-speed',
        clamp(
          2.45-Math.max(
            adhSignal,
            sodiumRetention
          )/75,
          .65,
          2.35
        ).toFixed(2)+'s'
      );

      dynamic.innerHTML='';
      labels.innerHTML='';

      if(mode==='sensing'){
        title.textContent=
          '细胞外液渗透状态感受与ADH释放';

        summary.textContent=
          '观察饮水、盐摄入和失水如何影响下丘脑感受及抗利尿激素释放。';

        dynamic.innerHTML=renderSensing(
          water,
          salt,
          sweat,
          osmoticIndex,
          adhSignal,
          progress
        );

        stageNote.textContent=
          '细胞外液渗透压升高时，下丘脑渗透压感受器受到更强刺激。';

        renderLabels(mode);
      }else if(mode==='adh'){
        title.textContent=
          '抗利尿激素促进集合管水重吸收';

        summary.textContent=
          '观察ADH信号、水通道蛋白、水重吸收、尿量和尿液浓缩之间的关系。';

        dynamic.innerHTML=renderAdh(
          adhSignal,
          waterReabsorption,
          urineVolume,
          concentration,
          progress
        );

        stageNote.textContent=
          'ADH提高远端肾小管和集合管的水通透性，使更多水返回血液。';

        renderLabels(mode);
      }else if(mode==='aldosterone'){
        title.textContent=
          '醛固酮调节钠离子重吸收和钾离子分泌';

        summary.textContent=
          '观察醛固酮与远端肾单位钠钾转运的关系，并与ADH作用区分。';

        dynamic.innerHTML=renderAldosterone(
          aldosterone,
          sodiumRetention,
          potassiumSecretion,
          progress
        );

        stageNote.textContent=
          '醛固酮主要调节钠钾转运，ADH主要调节集合管水通透性。';

        renderLabels(mode);
      }else{
        title.textContent=
          '水盐平衡负反馈与典型情境';

        summary.textContent=
          '比较失水、饮水较多、盐负荷较高和接近稳态时的调节方向。';

        dynamic.innerHTML=renderFeedback(
          correctedOsmotic,
          adhSignal,
          urineVolume,
          concentration,
          water,
          salt,
          sweat
        );

        stageNote.textContent=
          '饮水行为、激素调节和肾脏重吸收共同减小内环境水盐偏差。';

        renderLabels(mode);
      }

      var condition=
        '当前饮水、食盐摄入、出汗失水和肾脏调节处于相对协调状态。';

      if(
        sweat>75
        &&water<35
      ){
        condition=
          '出汗失水较多且饮水较少，细胞外液渗透状态升高，ADH信号增强。';
      }else if(
        salt>75
        &&water<50
      ){
        condition=
          '食盐摄入较高且饮水不足，渗透状态升高，口渴和ADH调节倾向增强。';
      }else if(
        water>82
        &&salt<45
      ){
        condition=
          '饮水较多且盐负荷较低，ADH信号受到抑制，尿量相对增加。';
      }else if(
        adhCapacity<20
        &&osmoticIndex>60
      ){
        condition=
          '渗透状态较高，但ADH反应能力较低，集合管水重吸收增强幅度受到限制。';
      }else if(
        aldosterone<18
      ){
        condition=
          '醛固酮调节活性较低，远端肾单位钠离子重吸收和钾离子分泌相对减弱。';
      }else if(
        processTime<15
      ){
        condition=
          '调节过程时间较短，感受和激素信号已经启动，但肾脏效应尚未充分形成。';
      }

      var principle=mode==='sensing'
        ?'细胞外液渗透压升高可刺激下丘脑渗透压感受器；ADH由下丘脑有关神经分泌细胞合成，经神经垂体释放进入血液。'
        :mode==='adh'
          ?'ADH促进远端肾小管和集合管水通道蛋白增加，提高水通透性，使尿量通常减少、尿液相对浓缩。'
          :mode==='aldosterone'
            ?'醛固酮主要促进钠离子重吸收和钾离子分泌；水可随钠被动移动，但其直接作用重点不同于ADH。'
            :'水盐平衡调节通过感受变化、激素和神经信号、肾脏效应以及饮水行为共同减小原有偏差，体现负反馈。';

      result.innerHTML=principle
        +'<br>'+condition
        +' 当前调节后的相对渗透状态 '
        +correctedOsmotic.toFixed(0)
        +'，水重吸收指数 '
        +waterReabsorption.toFixed(0)
        +'；所有数值仅用于教学比较，不用于脱水、电解质或尿量评价。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute(
          'data-mode'
        );
        update();
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

    waterIntakeInput.oninput=update;
    saltInput.oninput=update;
    sweatInput.oninput=update;
    adhInput.oninput=update;
    aldosteroneInput.oninput=update;
    timeInput.oninput=update;

    update();
    schedule();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
