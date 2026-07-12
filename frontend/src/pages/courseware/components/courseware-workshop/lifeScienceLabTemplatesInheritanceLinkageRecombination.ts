/**
 * lifeScienceLabTemplatesInheritanceLinkageRecombination.ts
 *
 * 平面生命科学实验室：
 * 基因连锁、交换与重组率。
 *
 * 教学目标：
 * 1. 理解位于同一条染色体上的基因具有连锁关系，
 *    形成配子时亲本型组合通常多于重组型组合；
 * 2. 区分顺式连锁AB/ab和反式连锁Ab/aB；
 * 3. 理解减数第一次分裂前期，
 *    同源染色体的非姐妹染色单体之间可能发生交换；
 * 4. 理解两个基因之间发生交换后，
 *    可以形成不同于亲本组合的重组型染色单体；
 * 5. 理解连锁基因并非永不分离；
 * 6. 根据四类配子数量计算重组率；
 * 7. 理解重组率通常不超过50%；
 * 8. 比较完全连锁、较紧密连锁、较远连锁、
 *    反式连锁和独立分配；
 * 9. 理解位于非同源染色体上的两对基因
 *    可通过独立分配形成四类等机会配子；
 * 10. 比较理论比例和有限样本模拟结果，
 *     认识抽样造成的随机波动。
 *
 * 教学边界：
 * 1. 本模型使用A/a和B/b表示两对等位基因；
 * 2. 顺式连锁写作AB/ab，亲本型配子为AB和ab，
 *    重组型配子为Ab和aB；
 * 3. 反式连锁写作Ab/aB，亲本型配子为Ab和aB，
 *    重组型配子为AB和ab；
 * 4. 图中的一次交换只表示
 *    同源染色体非姐妹染色单体交换的结构示意，
 *    不表示每一次减数分裂都必然发生交换；
 * 5. 两基因之间发生一次交换的一个四分体，
 *    通常可形成两条亲本型和两条重组型染色单体；
 * 6. 群体重组率取决于大量减数分裂和配子的统计结果，
 *    不等于单个四分体中重组染色单体所占比例；
 * 7. 重组率通常不超过50%；
 * 8. 观察到接近50%的重组率，
 *    不能证明两个基因一定不连锁；
 * 9. 相距较远的连锁基因可能因多次交换，
 *    使可观察重组率接近50%；
 * 10. 多次交换可能恢复亲本型排列，
 *     因而不能只凭重组配子数识别全部交换事件；
 * 11. 1%的重组率约对应1 cM
 *     主要适用于较短区间的近似图距估算；
 * 12. 重组率不能直接等同物理距离，
 *     不同染色体区域的交换概率可能不同；
 * 13. 独立分配模型假设两对基因位于非同源染色体，
 *     四类配子理论比例均为25%；
 * 14. 理论比例假设各类配子的存活和结合机会相等；
 * 15. 有限样本模拟可能偏离理论比例，
 *     样本量增加时通常更接近理论概率；
 * 16. 本模型不用于真实育种预测、遗传咨询、
 *     个体基因定位或临床判断。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式、字体、图片或CDN；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用统一.bl-*公共布局协议；
 * 5. 支持同一课件页放置多个独立实例；
 * 6. 不使用document.querySelector或document.querySelectorAll；
 * 7. 本文件只导出独立模板数组，聚合接入由第29批C1完成。
 */

import type {
  LifeScienceLabParamValue,
  LifeScienceLabTemplate,
} from './lifeScienceLabUtils'

/**
 * 安全读取数值参数。
 */
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

/**
 * 安全读取布尔参数。
 */
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

/**
 * 把数值转换为适合写入HTML属性的短字符串。
 */
function n(value: number): string {
  return parseFloat(value.toFixed(3)).toString()
}

/**
 * 构建完全限定在当前rootId内部的样式。
 */
function linkageRecombinationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #A5B4FC;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#E0E7FF,#ECFDF5);border-bottom:1px solid #A5B4FC}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FAFAFF;border-right:1px solid #C7D2FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#4F46E5;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#4F46E5}'
    + '#' + rootId + ' input[type=range]:disabled{opacity:.42;cursor:not-allowed}'
    + '#' + rootId + ' .lr-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#3730A3}'
    + '#' + rootId + ' .lr-modes{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .lr-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .lr-button{min-height:31px;padding:3px;border:1px solid #A5B4FC;border-radius:8px;background:#fff;color:#3730A3;font-size:9px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .lr-button.active{border-color:#4F46E5;background:#E0E7FF;box-shadow:0 3px 9px rgba(79,70,229,.14)}'
    + '#' + rootId + ' .lr-auto{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#818CF8,#4F46E5);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .lr-auto.paused{background:#64748B}'
    + '#' + rootId + ' .lr-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#34D399,#059669);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .lr-toggle.off{background:#64748B}'
    + '#' + rootId + ' .lr-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .lr-card{padding:6px 3px;border:1px solid #C7D2FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .lr-card b{display:block;min-height:18px;font-size:12.5px;color:#4338CA}'
    + '#' + rootId + ' .lr-card span{font-size:8.5px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#E0E7FF;color:#312E81;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .lr-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--lr-flow-speed,1.5s) linear infinite}'
    + '#' + rootId + ' .lr-chromatid{animation:' + rootId + '-chromatid var(--lr-chromatid-speed,1.8s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .lr-exchange{animation:' + rootId + '-exchange 1.05s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-chromatid{from{transform:translateY(2px);opacity:.58}to{transform:translateY(-3px);opacity:1}}'
    + '@keyframes ' + rootId + '-exchange{from{opacity:.3}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_INHERITANCE_LINKAGE_RECOMBINATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-linkage-recombination',
    group: '🧬 遗传规律',
    name: '基因连锁、交换与重组率',
    emoji: '🔗',
    desc: '比较顺式连锁、反式连锁与独立分配，观察非姐妹染色单体交换、四类配子和重组率',
    params: [
      {
        key: 'recombinationRate',
        label: '理论重组率',
        type: 'number',
        min: 0,
        max: 50,
        step: 1,
        defaultValue: 18,
        hint: '独立分配模式固定为四类配子各25%',
      },
      {
        key: 'gameteCount',
        label: '模拟配子数量',
        type: 'number',
        min: 40,
        max: 1000,
        step: 20,
        defaultValue: 400,
      },
      {
        key: 'randomSeed',
        label: '随机实验编号',
        type: 'number',
        min: 1,
        max: 99,
        step: 1,
        defaultValue: 31,
      },
      {
        key: 'animationSpeed',
        label: '自动演示速度',
        type: 'number',
        min: 20,
        max: 100,
        step: 1,
        defaultValue: 56,
      },
      {
        key: 'showLabels',
        label: '显示交换标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const recombinationRate = num(
        params,
        'recombinationRate',
        18,
      )
      const gameteCount = num(
        params,
        'gameteCount',
        400,
      )
      const randomSeed = num(
        params,
        'randomSeed',
        31,
      )
      const animationSpeed = num(
        params,
        'animationSpeed',
        56,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${linkageRecombinationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🔗 基因连锁、交换与重组率</div>
    <div class="bl-note">连锁基因并非永不分离；重组率接近50%也不能证明一定不连锁</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>理论重组率</span>
          <span class="bl-value" data-rate-value></span>
        </div>
        <input
          data-rate
          type="range"
          min="0"
          max="50"
          step="1"
          value="${n(recombinationRate)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>模拟配子数量</span>
          <span class="bl-value" data-count-value></span>
        </div>
        <input
          data-count
          type="range"
          min="40"
          max="1000"
          step="20"
          value="${n(gameteCount)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>随机实验编号</span>
          <span class="bl-value" data-seed-value></span>
        </div>
        <input
          data-seed
          type="range"
          min="1"
          max="99"
          step="1"
          value="${n(randomSeed)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>自动演示速度</span>
          <span class="bl-value" data-speed-value></span>
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

      <div class="lr-subtitle">基因排列与分配模式</div>

      <div class="lr-modes">
        <button type="button" class="lr-button active" data-mode="coupling">顺式连锁</button>
        <button type="button" class="lr-button" data-mode="repulsion">反式连锁</button>
        <button type="button" class="lr-button" data-mode="independent">独立分配</button>
      </div>

      <div class="lr-subtitle">快速比较情境</div>

      <div class="lr-scenarios">
        <button type="button" class="lr-button" data-scenario="complete">完全连锁</button>
        <button type="button" class="lr-button active" data-scenario="close">紧密连锁</button>
        <button type="button" class="lr-button" data-scenario="distant">较远连锁</button>
        <button type="button" class="lr-button" data-scenario="repulsion">反式排列</button>
        <button type="button" class="lr-button" data-scenario="independent">独立分配</button>
      </div>

      <button type="button" class="lr-auto" data-auto>
        自动演示：运行中
      </button>

      <button
        type="button"
        class="lr-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '交换标注：显示' : '交换标注：隐藏'}</button>

      <div class="lr-status">
        <div class="lr-card">
          <b data-status-one></b>
          <span data-status-one-label></span>
        </div>

        <div class="lr-card">
          <b data-status-two></b>
          <span data-status-two-label></span>
        </div>

        <div class="lr-card">
          <b data-status-three></b>
          <span data-status-three-label></span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="基因连锁、交换与重组率互动模型"
      >
        <defs>
          <marker
            id="${rootId}-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#4F46E5"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="4"
              stdDeviation="5"
              flood-color="#312E81"
              flood-opacity=".14"
            />
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text
          x="22"
          y="34"
          data-title
          font-size="24"
          font-weight="900"
          fill="#3730A3"
        ></text>

        <text
          x="22"
          y="61"
          data-summary
          font-size="12.5"
          font-weight="800"
          fill="#475569"
        ></text>

        <g filter="url(#${rootId}-shadow)">
          <rect
            x="22"
            y="79"
            width="348"
            height="195"
            rx="19"
            fill="#F8FAFF"
            stroke="#A5B4FC"
            stroke-width="3"
          />

          <rect
            x="390"
            y="79"
            width="348"
            height="195"
            rx="19"
            fill="#FAFFFC"
            stroke="#A7F3D0"
            stroke-width="3"
          />
        </g>

        <text
          x="196"
          y="104"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#3730A3"
        >染色体与交换示意</text>

        <text
          x="564"
          y="104"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#047857"
        >四类配子理论比例</text>

        <g data-scene-layer></g>
        <g data-frequency-layer></g>
        <g data-label-layer></g>

        <text
          x="22"
          y="301"
          font-size="13"
          font-weight="900"
          fill="#334155"
        >理论比例与有限样本模拟比较</text>

        <g data-chart-layer></g>

        <g transform="translate(22 416)">
          <rect x="0" y="-7" width="15" height="13" rx="3" fill="#C7D2FE"/>
          <text x="22" y="4" font-size="10.5" font-weight="800" fill="#475569">
            浅色：理论比例
          </text>
        </g>

        <g transform="translate(164 416)">
          <rect x="0" y="-7" width="15" height="13" rx="3" fill="#4F46E5"/>
          <text x="22" y="4" font-size="10.5" font-weight="800" fill="#475569">
            深色：模拟比例
          </text>
        </g>

        <text
          x="330"
          y="420"
          data-footer-note
          font-size="10.5"
          font-weight="900"
          fill="#3730A3"
        ></text>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var rateInput=root.querySelector('[data-rate]');
    var countInput=root.querySelector('[data-count]');
    var seedInput=root.querySelector('[data-seed]');
    var speedInput=root.querySelector('[data-speed]');

    var rateValue=root.querySelector('[data-rate-value]');
    var countValue=root.querySelector('[data-count-value]');
    var seedValue=root.querySelector('[data-seed-value]');
    var speedValue=root.querySelector('[data-speed-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var autoButton=root.querySelector('[data-auto]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var statusOne=root.querySelector('[data-status-one]');
    var statusTwo=root.querySelector('[data-status-two]');
    var statusThree=root.querySelector('[data-status-three]');
    var statusOneLabel=root.querySelector('[data-status-one-label]');
    var statusTwoLabel=root.querySelector('[data-status-two-label]');
    var statusThreeLabel=root.querySelector('[data-status-three-label]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var sceneLayer=root.querySelector('[data-scene-layer]');
    var frequencyLayer=root.querySelector('[data-frequency-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');
    var chartLayer=root.querySelector('[data-chart-layer]');
    var footerNote=root.querySelector('[data-footer-note]');

    var mode='coupling';
    var automatic=true;
    var timer=null;
    var showLabels=${showLabels ? 'true' : 'false'};
    var currentScenario='close';

    var gameteKeys=[
      'AB',
      'Ab',
      'aB',
      'ab'
    ];

    var modeOrder=[
      'coupling',
      'repulsion',
      'independent'
    ];

    var scenarioOrder=[
      'complete',
      'close',
      'distant',
      'repulsion',
      'independent'
    ];

    var scenarios={
      complete:{
        mode:'coupling',
        rate:0
      },
      close:{
        mode:'coupling',
        rate:18
      },
      distant:{
        mode:'coupling',
        rate:42
      },
      repulsion:{
        mode:'repulsion',
        rate:18
      },
      independent:{
        mode:'independent',
        rate:50
      }
    };

    function clamp(value,min,max){
      return Math.max(
        min,
        Math.min(
          max,
          value
        )
      );
    }

    function setScenarioActive(name){
      for(var i=0;i<scenarioButtons.length;i++){
        scenarioButtons[i].classList.toggle(
          'active',
          scenarioButtons[i].getAttribute('data-scenario')===name
        );
      }
    }

    function setModeActive(){
      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }
    }

    /**
     * 建立四类配子的理论比例。
     *
     * 顺式连锁AB/ab：
     * - 亲本型AB、ab；
     * - 重组型Ab、aB。
     *
     * 反式连锁Ab/aB：
     * - 亲本型Ab、aB；
     * - 重组型AB、ab。
     *
     * 独立分配：
     * - AB、Ab、aB、ab各占25%。
     */
    function createModel(
      inheritanceMode,
      ratePercent
    ){
      var frequencies={
        AB:0,
        Ab:0,
        aB:0,
        ab:0
      };

      var parental=[];
      var recombinant=[];
      var phase='';
      var effectiveRate=
        inheritanceMode==='independent'
          ?50
          :clamp(
            ratePercent,
            0,
            50
          );

      var r=
        effectiveRate/100;

      if(inheritanceMode==='coupling'){
        frequencies.AB=(1-r)/2;
        frequencies.ab=(1-r)/2;
        frequencies.Ab=r/2;
        frequencies.aB=r/2;

        parental=['AB','ab'];
        recombinant=['Ab','aB'];
        phase='顺式 AB/ab';
      }else if(inheritanceMode==='repulsion'){
        frequencies.Ab=(1-r)/2;
        frequencies.aB=(1-r)/2;
        frequencies.AB=r/2;
        frequencies.ab=r/2;

        parental=['Ab','aB'];
        recombinant=['AB','ab'];
        phase='反式 Ab/aB';
      }else{
        frequencies.AB=.25;
        frequencies.Ab=.25;
        frequencies.aB=.25;
        frequencies.ab=.25;

        parental=[];
        recombinant=[];
        phase='独立分配';
      }

      return {
        mode:inheritanceMode,
        frequencies:frequencies,
        parental:parental,
        recombinant:recombinant,
        phase:phase,
        rate:effectiveRate
      };
    }

    function createRandom(
      seed,
      count,
      rate
    ){
      var state=(
        Math.floor(seed)*2654435761
        +Math.floor(count)*131
        +Math.floor(rate*100)*7919
        +modeOrder.indexOf(mode)*104729
      )>>>0;

      return function(){
        state=(
          Math.imul(
            state,
            1664525
          )
          +1013904223
        )>>>0;

        return state/4294967296;
      };
    }

    function emptyGameteCounts(){
      return {
        AB:0,
        Ab:0,
        aB:0,
        ab:0
      };
    }

    /**
     * 按理论配子概率进行可重复的有限样本模拟。
     */
    function simulateGametes(
      model,
      count,
      seed
    ){
      var random=createRandom(
        seed,
        count,
        model.rate
      );

      var counts=
        emptyGameteCounts();

      var cumulative=[];
      var running=0;

      for(var i=0;i<gameteKeys.length;i++){
        var key=gameteKeys[i];

        running+=
          model.frequencies[key];

        cumulative.push({
          key:key,
          limit:running
        });
      }

      for(var sample=0;sample<count;sample++){
        var value=random();
        var selected=
          cumulative[
            cumulative.length-1
          ].key;

        for(var index=0;
          index<cumulative.length;
          index++
        ){
          if(value<cumulative[index].limit){
            selected=
              cumulative[index].key;
            break;
          }
        }

        counts[selected]+=1;
      }

      return counts;
    }

    function percent(
      value,
      total
    ){
      return total>0
        ?value/total*100
        :0;
    }

    function observedRecombination(
      model,
      counts,
      total
    ){
      if(model.mode==='independent'){
        return 50;
      }

      var recombinantCount=0;

      for(var i=0;
        i<model.recombinant.length;
        i++
      ){
        recombinantCount+=
          counts[
            model.recombinant[i]
          ];
      }

      return percent(
        recombinantCount,
        total
      );
    }

    function gameteColor(gamete){
      var colors={
        AB:'#4F46E5',
        Ab:'#2563EB',
        aB:'#10B981',
        ab:'#EC4899'
      };

      return colors[gamete]
        ||'#64748B';
    }

    function geneMarker(
      x,
      y,
      allele,
      color
    ){
      return ''
        +'<circle cx="'+x+'" cy="'+y
        +'" r="10" fill="'+color
        +'" stroke="#FFFFFF" stroke-width="2.5"/>'
        +'<text x="'+x+'" y="'+(y+4)
        +'" text-anchor="middle" font-size="10"'
        +' font-weight="900" fill="#FFFFFF">'
        +allele+'</text>';
    }

    function chromatid(
      y,
      firstAllele,
      secondAllele,
      firstX,
      secondX,
      color,
      opacity
    ){
      var html='';

      html+='<g class="lr-chromatid"'
        +' opacity="'+opacity+'">';

      html+='<line x1="57" y1="'+y
        +'" x2="319" y2="'+y
        +'" stroke="'+color+'"'
        +' stroke-width="8"'
        +' stroke-linecap="round"/>';

      html+=geneMarker(
        firstX,
        y,
        firstAllele,
        firstAllele===firstAllele.toUpperCase()
          ?'#4F46E5'
          :'#EC4899'
      );

      html+=geneMarker(
        secondX,
        y,
        secondAllele,
        secondAllele===secondAllele.toUpperCase()
          ?'#059669'
          :'#F59E0B'
      );

      html+='</g>';

      return html;
    }

    function renderLinkedScene(
      model
    ){
      var distance=
        54+model.rate*3.1;

      var firstX=105;
      var secondX=clamp(
        firstX+distance,
        165,
        285
      );

      var firstTop=
        model.mode==='coupling'
          ?'A'
          :'A';

      var secondTop=
        model.mode==='coupling'
          ?'B'
          :'b';

      var firstBottom='a';

      var secondBottom=
        model.mode==='coupling'
          ?'b'
          :'B';

      var exchangeOpacity=
        model.rate/50;

      var crossX=
        (firstX+secondX)/2;

      var html='';

      html+=chromatid(
        128,
        firstTop,
        secondTop,
        firstX,
        secondX,
        '#6366F1',
        1
      );

      html+=chromatid(
        149,
        firstTop,
        secondTop,
        firstX,
        secondX,
        '#818CF8',
        1
      );

      html+=chromatid(
        197,
        firstBottom,
        secondBottom,
        firstX,
        secondX,
        '#F472B6',
        1
      );

      html+=chromatid(
        218,
        firstBottom,
        secondBottom,
        firstX,
        secondX,
        '#EC4899',
        1
      );

      if(model.rate>0){
        html+='<g class="lr-exchange"'
          +' opacity="'+(.2+.8*exchangeOpacity)+'">';

        html+='<path d="M'+(firstX+15)+' 149'
          +' C'+(crossX-18)+' 149 '
          +(crossX+18)+' 197 '
          +(secondX-15)+' 197"'
          +' fill="none" stroke="#F59E0B"'
          +' stroke-width="5"'
          +' stroke-linecap="round"/>';

        html+='<path d="M'+(firstX+15)+' 197'
          +' C'+(crossX-18)+' 197 '
          +(crossX+18)+' 149 '
          +(secondX-15)+' 149"'
          +' fill="none" stroke="#10B981"'
          +' stroke-width="5"'
          +' stroke-linecap="round"/>';

        html+='<circle cx="'+crossX+'" cy="173"'
          +' r="8" fill="#FDE68A"'
          +' stroke="#B45309" stroke-width="2.5"/>';

        html+='</g>';
      }

      html+='<text x="188" y="246"'
        +' text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#3730A3">'
        +model.phase
        +'</text>';

      return html;
    }

    function renderIndependentScene(){
      var html='';

      html+='<text x="188" y="124"'
        +' text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#3730A3">'
        +'基因A/a所在染色体对'
        +'</text>';

      html+='<line x1="65" y1="145" x2="172" y2="145"'
        +' stroke="#6366F1" stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+='<line x1="65" y1="169" x2="172" y2="169"'
        +' stroke="#EC4899" stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+=geneMarker(
        117,
        145,
        'A',
        '#4F46E5'
      );

      html+=geneMarker(
        117,
        169,
        'a',
        '#EC4899'
      );

      html+='<text x="188" y="196"'
        +' text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#3730A3">'
        +'基因B/b所在另一对非同源染色体'
        +'</text>';

      html+='<line x1="204" y1="217" x2="311" y2="217"'
        +' stroke="#10B981" stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+='<line x1="204" y1="241" x2="311" y2="241"'
        +' stroke="#F59E0B" stroke-width="9"'
        +' stroke-linecap="round"/>';

      html+=geneMarker(
        257,
        217,
        'B',
        '#059669'
      );

      html+=geneMarker(
        257,
        241,
        'b',
        '#F59E0B'
      );

      return html;
    }

    function renderScene(
      model
    ){
      if(model.mode==='independent'){
        return renderIndependentScene();
      }

      return renderLinkedScene(model);
    }

    function renderFrequencyPanel(
      model
    ){
      var html='';
      var x=407;
      var startY=126;
      var barX=463;
      var barWidth=244;

      for(var i=0;i<gameteKeys.length;i++){
        var key=gameteKeys[i];
        var y=startY+i*34;
        var value=
          model.frequencies[key]*100;

        html+='<text x="'+x+'" y="'+(y+4)
          +'" font-size="11" font-weight="900"'
          +' fill="'+gameteColor(key)+'">'
          +key+'</text>';

        html+='<rect x="'+barX+'" y="'+(y-7)
          +'" width="'+barWidth+'" height="13"'
          +' rx="6.5" fill="#E2E8F0"/>';

        html+='<rect x="'+barX+'" y="'+(y-7)
          +'" width="'+(barWidth*value/100)
          +'" height="13" rx="6.5"'
          +' fill="'+gameteColor(key)+'"/>';

        html+='<text x="718" y="'+(y+4)
          +'" text-anchor="end" font-size="10"'
          +' font-weight="900" fill="#475569">'
          +value.toFixed(1)+'%'
          +'</text>';
      }

      html+='<text x="564" y="261"'
        +' text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#047857">'
        +(model.mode==='independent'
          ?'四类配子等机会形成'
          :'亲本型总计 '
            +(100-model.rate).toFixed(0)
            +'%｜重组型总计 '
            +model.rate.toFixed(0)
            +'%')
        +'</text>';

      return html;
    }

    function renderLabels(
      model
    ){
      if(!showLabels){
        return '';
      }

      var html='';

      if(model.mode==='independent'){
        html+='<text x="196" y="264"'
          +' text-anchor="middle" font-size="10.5"'
          +' font-weight="900" fill="#3730A3">'
          +'非同源染色体独立排列与分离'
          +'</text>';
      }else if(model.rate<=0){
        html+='<text x="196" y="264"'
          +' text-anchor="middle" font-size="10.5"'
          +' font-weight="900" fill="#3730A3">'
          +'当前设定未产生可观察重组配子'
          +'</text>';
      }else{
        html+='<text x="196" y="264"'
          +' text-anchor="middle" font-size="10.5"'
          +' font-weight="900" fill="#B45309">'
          +'非姐妹染色单体交换示意'
          +'</text>';

        html+='<path class="lr-flow"'
          +' d="M303 178 C330 178 343 178 365 178"'
          +' fill="none" stroke="#4F46E5"'
          +' stroke-width="3"'
          +' marker-end="url(#${rootId}-arrow)"/>';
      }

      return html;
    }

    function renderChart(
      model,
      simulated,
      count
    ){
      var html='';
      var baseline=394;
      var top=326;
      var chartHeight=64;

      html+='<line x1="55" y1="'+baseline
        +'" x2="720" y2="'+baseline
        +'" stroke="#64748B" stroke-width="2"/>';

      for(var i=0;i<gameteKeys.length;i++){
        var key=gameteKeys[i];
        var x=112+i*164;

        var theoryPercent=
          model.frequencies[key]*100;

        var simulatedPercent=
          percent(
            simulated[key],
            count
          );

        var theoryHeight=
          chartHeight*theoryPercent/100;

        var simulatedHeight=
          chartHeight*simulatedPercent/100;

        html+='<rect x="'+(x-30)
          +'" y="'+(baseline-theoryHeight)
          +'" width="25" height="'+theoryHeight
          +'" rx="4" fill="'+gameteColor(key)
          +'" opacity=".32"/>';

        html+='<rect x="'+(x+5)
          +'" y="'+(baseline-simulatedHeight)
          +'" width="25" height="'+simulatedHeight
          +'" rx="4" fill="'+gameteColor(key)+'"/>';

        html+='<text x="'+x+'" y="'+(baseline+16)
          +'" text-anchor="middle" font-size="10"'
          +' font-weight="900" fill="#475569">'
          +key+'</text>';

        html+='<text x="'+x+'" y="'+(top-3)
          +'" text-anchor="middle" font-size="8.8"'
          +' font-weight="800" fill="#64748B">'
          +'理 '+theoryPercent.toFixed(1)
          +'% / 模 '+simulatedPercent.toFixed(1)
          +'%'
          +'</text>';
      }

      return html;
    }

    function maximumDeviation(
      model,
      simulated,
      count
    ){
      var maxDeviation=0;

      for(var i=0;i<gameteKeys.length;i++){
        var key=gameteKeys[i];

        var theory=
          model.frequencies[key];

        var observed=
          simulated[key]/count;

        maxDeviation=Math.max(
          maxDeviation,
          Math.abs(
            theory-observed
          )
        );
      }

      return maxDeviation*100;
    }

    function updateStatus(
      model,
      simulated,
      count
    ){
      statusOne.textContent=
        model.phase;

      statusOneLabel.textContent=
        model.mode==='independent'
          ?'分配方式'
          :'连锁相位';

      if(model.mode==='independent'){
        statusTwo.textContent='25%';
        statusTwoLabel.textContent=
          '单类配子理论';

        statusThree.textContent=
          maximumDeviation(
            model,
            simulated,
            count
          ).toFixed(1)+'%';

        statusThreeLabel.textContent=
          '最大模拟偏差';
      }else{
        statusTwo.textContent=
          model.rate.toFixed(0)+'%';

        statusTwoLabel.textContent=
          '理论重组率';

        statusThree.textContent=
          observedRecombination(
            model,
            simulated,
            count
          ).toFixed(1)+'%';

        statusThreeLabel.textContent=
          '模拟重组率';
      }
    }

    function update(){
      var selectedRate=clamp(
        Number(rateInput.value),
        0,
        50
      );

      var count=Math.round(
        Number(countInput.value)
      );

      var seed=Math.round(
        Number(seedInput.value)
      );

      var speed=Number(
        speedInput.value
      );

      var model=createModel(
        mode,
        selectedRate
      );

      var simulated=
        simulateGametes(
          model,
          count,
          seed
        );

      rateInput.disabled=
        mode==='independent';

      rateValue.textContent=
        mode==='independent'
          ?'固定四类各25%'
          :model.rate.toFixed(0)+'%';

      countValue.textContent=
        count+' 个';

      seedValue.textContent=
        '第 '+seed+' 组';

      speedValue.textContent=
        speed.toFixed(0)+'%';

      autoButton.textContent=
        automatic
          ?'自动演示：运行中'
          :'自动演示：已暂停';

      autoButton.classList.toggle(
        'paused',
        !automatic
      );

      labelToggle.textContent=
        showLabels
          ?'交换标注：显示'
          :'交换标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      root.style.setProperty(
        '--lr-flow-speed',
        clamp(
          2.6-speed/58,
          .58,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--lr-chromatid-speed',
        clamp(
          2.5-model.rate/42,
          .75,
          2.4
        ).toFixed(2)+'s'
      );

      setModeActive();
      setScenarioActive(
        currentScenario
      );

      if(mode==='coupling'){
        title.textContent=
          '顺式连锁：AB/ab';

        summary.textContent=
          'AB和ab为亲本型配子，Ab和aB为交换形成的重组型配子';
      }else if(mode==='repulsion'){
        title.textContent=
          '反式连锁：Ab/aB';

        summary.textContent=
          'Ab和aB为亲本型配子，AB和ab为交换形成的重组型配子';
      }else{
        title.textContent=
          '非同源染色体上的两对基因独立分配';

        summary.textContent=
          'A/a与B/b分别排列和分离，形成AB、Ab、aB、ab四类等机会配子';
      }

      sceneLayer.innerHTML=
        renderScene(model);

      frequencyLayer.innerHTML=
        renderFrequencyPanel(model);

      labelLayer.innerHTML=
        renderLabels(model);

      chartLayer.innerHTML=
        renderChart(
          model,
          simulated,
          count
        );

      updateStatus(
        model,
        simulated,
        count
      );

      var sampleDeviation=
        maximumDeviation(
          model,
          simulated,
          count
        );

      var sampleNote='';

      if(count<=100){
        sampleNote=
          '当前样本量较小，四类配子比例出现明显随机波动是正常现象。';
      }else if(sampleDeviation>7){
        sampleNote=
          '本组模拟与理论比例仍有一定偏差，可改变实验编号或增大样本量。';
      }else{
        sampleNote=
          '当前模拟结果已较接近理论概率，但有限样本不会保证严格相等。';
      }

      var teachingNote='';
      var distanceNote='';

      if(mode==='independent'){
        teachingNote=
          '两对基因位于非同源染色体时，可通过独立分配形成四类等机会配子。此时不宜把四类配子简单称为亲本型和重组型。';

        distanceNote=
          '独立分配产生的四类配子各占25%，与重组率达到统计上限时的结果可能相似。';
      }else if(model.rate===0){
        teachingNote=
          '当前为完全连锁教学情境，只形成两类亲本型配子；这不代表真实生物中这些基因永远不会发生交换。';

        distanceNote=
          '未观察到重组配子时，只能说明当前样本和条件下没有检测到重组。';
      }else{
        teachingNote=
          '减数第一次分裂前期，同源染色体的非姐妹染色单体之间发生交换，可以形成重组型配子。图中只展示一次交换的结构示意。';

        if(model.rate<=20){
          distanceNote=
            '在较短区间、没有明显多次交换影响时，'
            +model.rate.toFixed(0)
            +'%的重组率可近似理解为'
            +model.rate.toFixed(0)
            +' cM的遗传图距。';
        }else if(model.rate<50){
          distanceNote=
            '当前重组率较高，不宜再把百分数直接等同遗传图距；多次交换可能使部分事件不能由重组配子统计识别。';
        }else{
          distanceNote=
            '重组率接近50%不能证明两个基因一定不连锁；相距较远的连锁基因也可能表现出接近独立分配的结果。';
        }
      }

      footerNote.textContent=
        mode==='independent'
          ?'独立分配：四类配子理论上各25%'
          :model.phase
            +'｜亲本型 '
            +(100-model.rate).toFixed(0)
            +'%｜重组型 '
            +model.rate.toFixed(0)
            +'%';

      result.innerHTML=
        teachingNote
        +'<br>'+distanceNote
        +' '+sampleNote
        +' 重组率通常不超过50%，且不能直接等同物理距离。'
        +' 本模型不用于真实育种预测或个体基因定位。';
    }

    function applyScenario(
      name,
      pauseAutomatic
    ){
      var data=scenarios[name];

      if(!data){
        return;
      }

      mode=data.mode;
      rateInput.value=String(
        data.rate
      );
      currentScenario=name;

      if(pauseAutomatic){
        automatic=false;
      }

      update();
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

      var speed=Number(
        speedInput.value
      );

      var interval=clamp(
        4500-speed*29,
        1250,
        4000
      );

      timer=window.setTimeout(function(){
        var index=
          scenarioOrder.indexOf(
            currentScenario
          );

        var nextIndex=
          index<0
            ?0
            :(index+1)%scenarioOrder.length;

        seedInput.value=String(
          Number(seedInput.value)>=99
            ?1
            :Number(seedInput.value)+1
        );

        applyScenario(
          scenarioOrder[nextIndex],
          false
        );

        schedule();
      },interval);
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        automatic=false;
        mode=this.getAttribute(
          'data-mode'
        );

        if(mode==='independent'){
          rateInput.value='50';
        }

        currentScenario='';
        update();
        schedule();
      };
    }

    for(var j=0;j<scenarioButtons.length;j++){
      scenarioButtons[j].onclick=function(){
        var name=this.getAttribute(
          'data-scenario'
        );

        applyScenario(
          name,
          true
        );

        schedule();
      };
    }

    autoButton.onclick=function(){
      automatic=!automatic;
      update();
      schedule();
    };

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    rateInput.oninput=function(){
      automatic=false;
      currentScenario='';
      update();
      schedule();
    };

    countInput.oninput=function(){
      currentScenario='';
      update();
    };

    seedInput.oninput=function(){
      currentScenario='';
      update();
    };

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
