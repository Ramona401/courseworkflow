/**
 * lifeScienceLabTemplatesGeneticsMutationVariation.ts
 *
 * 平面生命科学实验室：
 * 基因突变与染色体变异。
 *
 * 教学目标：
 * 1. 区分基因突变、染色体结构变异和染色体数目变异；
 * 2. 观察碱基替换、插入和缺失对DNA序列的影响；
 * 3. 理解碱基替换可能产生同义、错义或终止密码子变化；
 * 4. 理解编码区插入或缺失的碱基数不是3的倍数时，
 *    可能造成阅读框改变；
 * 5. 理解并非所有DNA序列变化都会改变蛋白质或表现型；
 * 6. 观察染色体片段缺失、重复、倒位和易位；
 * 7. 区分染色体结构变异与染色体数目变异；
 * 8. 观察单体、三体、三倍体和四倍体的染色体数目变化；
 * 9. 理解突变和染色体变异为遗传变异提供来源，
 *    但变异本身不一定有利，也不是按环境需要定向产生。
 *
 * 教学边界：
 * 1. 基因突变模式只展示一段简化的蛋白质编码区；
 * 2. 碱基替换不改变DNA片段长度，
 *    但可能改变密码子，也可能不改变氨基酸；
 * 3. 插入或缺失是否造成移码，需要结合发生位置、
 *    变化碱基数和真实基因结构判断；
 * 4. 本模型没有完整表示启动子、内含子、外显子、
 *    RNA剪接、调控序列和蛋白质折叠；
 * 5. 染色体缺失、重复、倒位和易位均为结构示意，
 *    不对应具体物种或真实患者核型；
 * 6. 易位模式只展示两条非同源染色体交换片段的简化情形；
 * 7. 单体和三体属于非整倍性示意，
 *    三倍体和四倍体属于整倍性变化示意；
 * 8. 染色体数目变异的形成机制可能涉及染色体不分离、
 *    配子异常或整个染色体组变化，本模型不作临床解释；
 * 9. 体细胞变异不一定遗传给后代；
 *    进入生殖细胞或其祖细胞的变异才可能传给后代；
 * 10. 突变具有不定向性，环境因素可能影响发生概率，
 *     但不会按照生物需要定向产生特定有利变异；
 * 11. 本模型不用于疾病诊断、遗传咨询、
 *     胚胎筛查或个人健康判断。
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
function mutationVariationStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#FCE7F3);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:252px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #DDD6FE}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#8B5CF6}'
    + '#' + rootId + ' .mv-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .mv-modes{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .mv-types{display:grid;grid-template-columns:repeat(4,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .mv-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .mv-button{min-height:31px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:9px;font-weight:800;line-height:1.12;cursor:pointer}'
    + '#' + rootId + ' .mv-button.active{border-color:#7C3AED;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .mv-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .mv-toggle.off{background:#64748B}'
    + '#' + rootId + ' .mv-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .mv-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .mv-card b{display:block;min-height:18px;font-size:12.5px;color:#6D28D9}'
    + '#' + rootId + ' .mv-card span{font-size:8.6px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.7px;line-height:1.43;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .mv-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--mv-flow-speed,1.4s) linear infinite}'
    + '#' + rootId + ' .mv-change{animation:' + rootId + '-change 1.05s ease-in-out infinite alternate}'
    + '#' + rootId + ' .mv-chromosome{animation:' + rootId + '-chromosome var(--mv-chromosome-speed,1.8s) ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-change{from{opacity:.38}to{opacity:1}}'
    + '@keyframes ' + rootId + '-chromosome{from{transform:translateY(2px);opacity:.58}to{transform:translateY(-3px);opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_GENETICS_MUTATION_VARIATION:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-gene-chromosome-variation',
    group: '🧬 遗传与细胞分裂',
    name: '基因突变与染色体变异',
    emoji: '🧬',
    desc: '比较碱基替换、插入、缺失，染色体片段缺失、重复、倒位、易位及染色体数目变化',
    params: [
      {
        key: 'sequenceCodons',
        label: '编码序列密码子数',
        type: 'number',
        min: 3,
        max: 8,
        step: 1,
        defaultValue: 6,
      },
      {
        key: 'changePosition',
        label: '变化位置',
        type: 'number',
        min: 1,
        max: 24,
        step: 1,
        defaultValue: 6,
      },
      {
        key: 'fragmentSize',
        label: '变化片段大小',
        type: 'number',
        min: 1,
        max: 4,
        step: 1,
        defaultValue: 1,
      },
      {
        key: 'chromosomePairs',
        label: '同源染色体对数',
        type: 'number',
        min: 2,
        max: 6,
        step: 1,
        defaultValue: 4,
      },
      {
        key: 'showLabels',
        label: '显示结构标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const sequenceCodons = num(
        params,
        'sequenceCodons',
        6,
      )
      const changePosition = num(
        params,
        'changePosition',
        6,
      )
      const fragmentSize = num(
        params,
        'fragmentSize',
        1,
      )
      const chromosomePairs = num(
        params,
        'chromosomePairs',
        4,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${mutationVariationStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧬 基因突变与染色体变异</div>
    <div class="bl-note">变异不一定改变表现型，也不是按照环境需要定向产生</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>编码序列密码子数</span>
          <span class="bl-value" data-codons-value></span>
        </div>
        <input
          data-codons
          type="range"
          min="3"
          max="8"
          step="1"
          value="${n(sequenceCodons)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>变化位置</span>
          <span class="bl-value" data-position-value></span>
        </div>
        <input
          data-position
          type="range"
          min="1"
          max="24"
          step="1"
          value="${n(changePosition)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>变化片段大小</span>
          <span class="bl-value" data-size-value></span>
        </div>
        <input
          data-size
          type="range"
          min="1"
          max="4"
          step="1"
          value="${n(fragmentSize)}"
        >
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>同源染色体对数</span>
          <span class="bl-value" data-pairs-value></span>
        </div>
        <input
          data-pairs
          type="range"
          min="2"
          max="6"
          step="1"
          value="${n(chromosomePairs)}"
        >
      </div>

      <div class="mv-subtitle">变异层级</div>

      <div class="mv-modes">
        <button
          type="button"
          class="mv-button active"
          data-mode="gene"
        >基因突变</button>

        <button
          type="button"
          class="mv-button"
          data-mode="structure"
        >结构变异</button>

        <button
          type="button"
          class="mv-button"
          data-mode="number"
        >数目变异</button>
      </div>

      <div class="mv-subtitle" data-type-title>选择基因突变类型</div>

      <div class="mv-types">
        <button type="button" class="mv-button active" data-kind="type1">碱基替换</button>
        <button type="button" class="mv-button" data-kind="type2">碱基插入</button>
        <button type="button" class="mv-button" data-kind="type3">碱基缺失</button>
        <button type="button" class="mv-button" data-kind="type4">三者比较</button>
      </div>

      <div class="mv-subtitle">快速比较情境</div>

      <div class="mv-scenarios">
        <button type="button" class="mv-button active" data-scenario="silent">同义替换</button>
        <button type="button" class="mv-button" data-scenario="frameshift">移码插入</button>
        <button type="button" class="mv-button" data-scenario="inversion">片段倒位</button>
        <button type="button" class="mv-button" data-scenario="trisomy">三体示意</button>
        <button type="button" class="mv-button" data-scenario="polyploid">多倍体示意</button>
      </div>

      <button
        type="button"
        class="mv-toggle${showLabels ? '' : ' off'}"
        data-label-toggle
      >${showLabels ? '结构标注：显示' : '结构标注：隐藏'}</button>

      <div class="mv-status">
        <div class="mv-card">
          <b data-status-type></b>
          <span>当前变异</span>
        </div>

        <div class="mv-card">
          <b data-status-amount></b>
          <span data-amount-label>长度变化</span>
        </div>

        <div class="mv-card">
          <b data-status-effect></b>
          <span>主要示意结果</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="基因突变与染色体变异互动模型"
      >
        <defs>
          <linearGradient id="${rootId}-original" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#EFF6FF"/>
            <stop offset="100%" stop-color="#EEF2FF"/>
          </linearGradient>

          <linearGradient id="${rootId}-changed" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#FDF2F8"/>
            <stop offset="100%" stop-color="#F5F3FF"/>
          </linearGradient>

          <marker
            id="${rootId}-arrow"
            markerWidth="9"
            markerHeight="9"
            refX="7"
            refY="3"
            orient="auto"
          >
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="6"
              flood-color="#4C1D95"
              flood-opacity=".13"
            />
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>

        <text
          x="22"
          y="34"
          data-title
          font-size="25"
          font-weight="900"
          fill="#5B21B6"
        ></text>

        <text
          x="22"
          y="62"
          data-summary
          font-size="13"
          font-weight="800"
          fill="#475569"
        ></text>

        <g filter="url(#${rootId}-shadow)">
          <rect
            x="22"
            y="80"
            width="310"
            height="195"
            rx="20"
            fill="url(#${rootId}-original)"
            stroke="#93C5FD"
            stroke-width="3"
          />

          <rect
            x="428"
            y="80"
            width="310"
            height="195"
            rx="20"
            fill="url(#${rootId}-changed)"
            stroke="#C4B5FD"
            stroke-width="3"
          />
        </g>

        <text
          x="177"
          y="105"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#1D4ED8"
        >原始状态</text>

        <text
          x="583"
          y="105"
          text-anchor="middle"
          font-size="13"
          font-weight="900"
          fill="#7C3AED"
        >变化后状态</text>

        <path
          class="mv-flow"
          d="M347 177 H411"
          fill="none"
          stroke="#7C3AED"
          stroke-width="5"
          marker-end="url(#${rootId}-arrow)"
        />

        <text
          x="379"
          y="160"
          text-anchor="middle"
          data-arrow-label
          font-size="11"
          font-weight="900"
          fill="#6D28D9"
        ></text>

        <g data-original-layer></g>
        <g data-changed-layer></g>
        <g data-label-layer></g>

        <text
          x="22"
          y="306"
          data-panel-title
          font-size="13"
          font-weight="900"
          fill="#334155"
        ></text>

        <g transform="translate(22 320)">
          <rect
            width="716"
            height="75"
            rx="16"
            fill="#F8FAFC"
            stroke="#CBD5E1"
            stroke-width="2"
          />

          <text
            x="15"
            y="25"
            data-metric-one-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect x="126" y="16" width="190" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-metric-one-bar x="126" y="16" width="0" height="12" rx="6" fill="#2563EB"/>

          <text
            x="354"
            y="25"
            data-metric-two-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect x="465" y="16" width="225" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-metric-two-bar x="465" y="16" width="0" height="12" rx="6" fill="#EC4899"/>

          <text
            x="15"
            y="55"
            data-metric-three-label
            font-size="10.5"
            font-weight="800"
            fill="#64748B"
          ></text>

          <rect x="126" y="46" width="190" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-metric-three-bar x="126" y="46" width="0" height="12" rx="6" fill="#F59E0B"/>

          <text
            x="354"
            y="55"
            data-panel-note
            font-size="10.5"
            font-weight="900"
            fill="#5B21B6"
          ></text>
        </g>

        <g data-footer-labels>
          <text
            x="24"
            y="419"
            data-footer-note
            font-size="11"
            font-weight="900"
            fill="#5B21B6"
          ></text>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var codonsInput=root.querySelector('[data-codons]');
    var positionInput=root.querySelector('[data-position]');
    var sizeInput=root.querySelector('[data-size]');
    var pairsInput=root.querySelector('[data-pairs]');

    var codonsValue=root.querySelector('[data-codons-value]');
    var positionValue=root.querySelector('[data-position-value]');
    var sizeValue=root.querySelector('[data-size-value]');
    var pairsValue=root.querySelector('[data-pairs-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var kindButtons=root.querySelectorAll('[data-kind]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');
    var typeTitle=root.querySelector('[data-type-title]');

    var statusType=root.querySelector('[data-status-type]');
    var statusAmount=root.querySelector('[data-status-amount]');
    var statusEffect=root.querySelector('[data-status-effect]');
    var amountLabel=root.querySelector('[data-amount-label]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var arrowLabel=root.querySelector('[data-arrow-label]');
    var originalLayer=root.querySelector('[data-original-layer]');
    var changedLayer=root.querySelector('[data-changed-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');

    var panelTitle=root.querySelector('[data-panel-title]');
    var metricOneLabel=root.querySelector('[data-metric-one-label]');
    var metricTwoLabel=root.querySelector('[data-metric-two-label]');
    var metricThreeLabel=root.querySelector('[data-metric-three-label]');
    var metricOneBar=root.querySelector('[data-metric-one-bar]');
    var metricTwoBar=root.querySelector('[data-metric-two-bar]');
    var metricThreeBar=root.querySelector('[data-metric-three-bar]');
    var panelNote=root.querySelector('[data-panel-note]');
    var footerNote=root.querySelector('[data-footer-note]');

    var mode='gene';
    var kind='type1';
    var showLabels=${showLabels ? 'true' : 'false'};

    var scenarios={
      silent:{
        mode:'gene',
        kind:'type1',
        codons:6,
        position:6,
        size:1,
        pairs:4
      },
      frameshift:{
        mode:'gene',
        kind:'type2',
        codons:6,
        position:8,
        size:1,
        pairs:4
      },
      inversion:{
        mode:'structure',
        kind:'type3',
        codons:6,
        position:3,
        size:3,
        pairs:4
      },
      trisomy:{
        mode:'number',
        kind:'type2',
        codons:6,
        position:3,
        size:1,
        pairs:4
      },
      polyploid:{
        mode:'number',
        kind:'type3',
        codons:6,
        position:3,
        size:1,
        pairs:4
      }
    };

    var typeLabels={
      gene:[
        '碱基替换',
        '碱基插入',
        '碱基缺失',
        '三者比较'
      ],
      structure:[
        '片段缺失',
        '片段重复',
        '片段倒位',
        '染色体易位'
      ],
      number:[
        '单体 2n−1',
        '三体 2n+1',
        '三倍体 3n',
        '四倍体 4n'
      ]
    };

    var codonTable={
      TTT:'Phe',TTC:'Phe',TTA:'Leu',TTG:'Leu',
      TCT:'Ser',TCC:'Ser',TCA:'Ser',TCG:'Ser',
      TAT:'Tyr',TAC:'Tyr',TAA:'Stop',TAG:'Stop',
      TGT:'Cys',TGC:'Cys',TGA:'Stop',TGG:'Trp',

      CTT:'Leu',CTC:'Leu',CTA:'Leu',CTG:'Leu',
      CCT:'Pro',CCC:'Pro',CCA:'Pro',CCG:'Pro',
      CAT:'His',CAC:'His',CAA:'Gln',CAG:'Gln',
      CGT:'Arg',CGC:'Arg',CGA:'Arg',CGG:'Arg',

      ATT:'Ile',ATC:'Ile',ATA:'Ile',ATG:'Met',
      ACT:'Thr',ACC:'Thr',ACA:'Thr',ACG:'Thr',
      AAT:'Asn',AAC:'Asn',AAA:'Lys',AAG:'Lys',
      AGT:'Ser',AGC:'Ser',AGA:'Arg',AGG:'Arg',

      GTT:'Val',GTC:'Val',GTA:'Val',GTG:'Val',
      GCT:'Ala',GCC:'Ala',GCA:'Ala',GCG:'Ala',
      GAT:'Asp',GAC:'Asp',GAA:'Glu',GAG:'Glu',
      GGT:'Gly',GGC:'Gly',GGA:'Gly',GGG:'Gly'
    };

    function clamp(value,min,max){
      return Math.max(min,Math.min(max,value));
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

    function setKindActive(){
      for(var i=0;i<kindButtons.length;i++){
        var label=typeLabels[mode][i];

        kindButtons[i].textContent=label;

        kindButtons[i].classList.toggle(
          'active',
          kindButtons[i].getAttribute('data-kind')===kind
        );
      }

      typeTitle.textContent=
        mode==='gene'
          ?'选择基因突变类型'
          :mode==='structure'
            ?'选择染色体结构变异'
            :'选择染色体数目变异';
    }

    function baseSequence(codonCount){
      var source='ATGGCTGAATTCGGACCTAAACGTTACGGA';

      return source.slice(
        0,
        codonCount*3
      );
    }

    function translate(sequence){
      var aminoAcids=[];

      for(var i=0;i+2<sequence.length;i+=3){
        var codon=sequence.slice(i,i+3);
        var aminoAcid=codonTable[codon]||'?';

        aminoAcids.push(aminoAcid);

        if(aminoAcid==='Stop'){
          break;
        }
      }

      return aminoAcids;
    }

    function nextBase(base){
      var map={
        A:'G',
        G:'A',
        T:'C',
        C:'T'
      };

      return map[base]||'A';
    }

    function mutateGene(
      sequence,
      mutationKind,
      position,
      fragmentSize
    ){
      var index=clamp(
        position-1,
        0,
        Math.max(0,sequence.length-1)
      );

      var insertFragment='GCTA'.slice(
        0,
        fragmentSize
      );

      var mutated=sequence;
      var changedIndices=[];
      var lengthChange=0;
      var name='碱基替换';

      if(mutationKind==='type1'){
        mutated=
          sequence.slice(0,index)
          +nextBase(sequence.charAt(index))
          +sequence.slice(index+1);

        changedIndices=[index];
        name='碱基替换';
      }else if(mutationKind==='type2'){
        mutated=
          sequence.slice(0,index)
          +insertFragment
          +sequence.slice(index);

        lengthChange=fragmentSize;
        name='碱基插入';

        for(var i=0;i<fragmentSize;i++){
          changedIndices.push(index+i);
        }
      }else if(mutationKind==='type3'){
        var actualDelete=Math.min(
          fragmentSize,
          sequence.length-index
        );

        mutated=
          sequence.slice(0,index)
          +sequence.slice(index+actualDelete);

        lengthChange=-actualDelete;
        name='碱基缺失';
        changedIndices=[index];
      }else{
        mutated=sequence;
        name='三类突变比较';
      }

      var originalProtein=translate(sequence);
      var changedProtein=translate(mutated);
      var frameshift=
        lengthChange!==0
        &&Math.abs(lengthChange)%3!==0;

      var effect='序列未改变';

      if(mutationKind==='type1'){
        var originalCodonIndex=
          Math.floor(index/3);

        var originalAmino=
          originalProtein[originalCodonIndex]||'?';

        var changedAmino=
          changedProtein[originalCodonIndex]||'?';

        if(originalAmino===changedAmino){
          effect='同义变化';
        }else if(changedAmino==='Stop'){
          effect='终止密码子';
        }else{
          effect='氨基酸改变';
        }
      }else if(
        mutationKind==='type2'
        ||mutationKind==='type3'
      ){
        effect=frameshift
          ?'阅读框改变'
          :'阅读框保持';
      }else{
        effect='类型比较';
      }

      return {
        name:name,
        original:sequence,
        mutated:mutated,
        changedIndices:changedIndices,
        lengthChange:lengthChange,
        frameshift:frameshift,
        originalProtein:originalProtein,
        changedProtein:changedProtein,
        effect:effect,
        position:index
      };
    }

    function sequenceGraphic(
      sequence,
      protein,
      x,
      y,
      changedIndices,
      changed
    ){
      var html='';
      var maxBases=28;
      var shown=sequence.slice(0,maxBases);
      var cellWidth=9.3;
      var startX=x-(shown.length*cellWidth)/2;

      for(var i=0;i<shown.length;i++){
        var base=shown.charAt(i);
        var color=
          base==='A'
            ?'#2563EB'
            :base==='T'
              ?'#F59E0B'
              :base==='G'
                ?'#10B981'
                :'#EC4899';

        var highlighted=
          changedIndices.indexOf(i)>=0;

        html+='<rect'
          +(highlighted?' class="mv-change"':'')
          +' x="'+(startX+i*cellWidth)
          +'" y="'+y
          +'" width="'+(cellWidth-1)
          +'" height="22" rx="3" fill="'
          +color+'" opacity="'
          +(highlighted?'1':'.84')
          +'" stroke="'
          +(highlighted?'#DC2626':'#FFFFFF')
          +'" stroke-width="'
          +(highlighted?'2.5':'1')
          +'"/>';

        html+='<text x="'
          +(startX+i*cellWidth+(cellWidth-1)/2)
          +'" y="'+(y+15)
          +'" text-anchor="middle" font-size="8"'
          +' font-weight="900" fill="#FFFFFF">'
          +base+'</text>';
      }

      html+='<text x="'+x+'" y="'+(y+39)
        +'" text-anchor="middle" font-size="9.5"'
        +' font-weight="800" fill="#64748B">'
        +'DNA长度 '+sequence.length+' bp'
        +'</text>';

      var proteinText=protein.slice(0,8).join('–');

      html+='<rect x="'+(x-132)+'" y="'+(y+50)
        +'" width="264" height="40" rx="10"'
        +' fill="'+(changed?'#FCE7F3':'#DBEAFE')
        +'" stroke="'+(changed?'#F9A8D4':'#93C5FD')
        +'" stroke-width="2"/>';

      html+='<text x="'+x+'" y="'+(y+67)
        +'" text-anchor="middle" font-size="9.5"'
        +' font-weight="900" fill="'
        +(changed?'#9D174D':'#1E40AF')
        +'">编码区翻译示意</text>';

      html+='<text x="'+x+'" y="'+(y+83)
        +'" text-anchor="middle" font-size="9.2"'
        +' font-weight="800" fill="#475569">'
        +(proteinText||'无完整密码子')
        +'</text>';

      return html;
    }

    function segmentColor(segment){
      var colors={
        A:'#2563EB',
        B:'#0EA5E9',
        C:'#10B981',
        D:'#F59E0B',
        E:'#EC4899',
        F:'#8B5CF6',
        M:'#1D4ED8',
        N:'#06B6D4',
        O:'#059669',
        P:'#D97706',
        Q:'#DB2777',
        R:'#6D28D9'
      };

      return colors[segment]||'#64748B';
    }

    function chromosomeGraphic(
      segments,
      x,
      y,
      label,
      compact
    ){
      var html='';
      var segmentWidth=compact
        ?Math.min(34,225/Math.max(1,segments.length))
        :Math.min(40,244/Math.max(1,segments.length));

      var totalWidth=
        segmentWidth*segments.length;

      var startX=
        x-totalWidth/2;

      html+='<text x="'+x+'" y="'+(y-15)
        +'" text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#475569">'
        +label+'</text>';

      for(var i=0;i<segments.length;i++){
        var segment=segments[i];

        html+='<rect'
          +' class="mv-chromosome"'
          +' x="'+(startX+i*segmentWidth)
          +'" y="'+y
          +'" width="'+(segmentWidth-2)
          +'" height="38" rx="6" fill="'
          +segmentColor(segment)
          +'" stroke="#FFFFFF" stroke-width="2"/>';

        html+='<text x="'
          +(startX+i*segmentWidth+(segmentWidth-2)/2)
          +'" y="'+(y+24)
          +'" text-anchor="middle" font-size="12"'
          +' font-weight="900" fill="#FFFFFF">'
          +segment+'</text>';
      }

      html+='<circle cx="'+x+'" cy="'+(y+19)
        +'" r="7" fill="#FDE68A"'
        +' stroke="#B45309" stroke-width="2"/>';

      return html;
    }

    function structureVariation(
      mutationKind,
      position,
      fragmentSize
    ){
      var original=[
        'A','B','C','D','E','F'
      ];

      var second=[
        'M','N','O','P','Q','R'
      ];

      var start=clamp(
        position-1,
        0,
        original.length-1
      );

      var actualSize=Math.min(
        fragmentSize,
        original.length-start
      );

      var changed=original.slice();
      var changedSecond=second.slice();
      var affected=original.slice(
        start,
        start+actualSize
      );
      var name='';
      var effect='结构顺序改变';

      if(mutationKind==='type1'){
        changed=
          original.slice(0,start)
          .concat(
            original.slice(start+actualSize)
          );

        name='片段缺失';
        effect='遗传物质减少';
      }else if(mutationKind==='type2'){
        changed=
          original.slice(0,start+actualSize)
          .concat(
            affected,
            original.slice(start+actualSize)
          );

        name='片段重复';
        effect='局部片段增加';
      }else if(mutationKind==='type3'){
        changed=
          original.slice(0,start)
          .concat(
            affected.slice().reverse(),
            original.slice(start+actualSize)
          );

        name='片段倒位';
        effect='片段方向改变';
      }else{
        var cutOne=clamp(
          start,
          1,
          original.length-1
        );

        var cutTwo=clamp(
          fragmentSize,
          1,
          second.length-1
        );

        var tailOne=original.slice(cutOne);
        var tailTwo=second.slice(cutTwo);

        changed=
          original.slice(0,cutOne)
          .concat(tailTwo);

        changedSecond=
          second.slice(0,cutTwo)
          .concat(tailOne);

        affected=tailOne;
        name='染色体易位';
        effect='非同源染色体片段重排';
      }

      return {
        original:original,
        second:second,
        changed:changed,
        changedSecond:changedSecond,
        affected:affected,
        name:name,
        effect:effect,
        start:start,
        size:actualSize
      };
    }

    function chromosomeSymbol(
      x,
      y,
      color,
      index,
      changed
    ){
      var height=30+(index%3)*4;

      return '<g'
        +(changed?' class="mv-change"':'')
        +' transform="translate('+x+' '+y+')">'
        +'<path d="M-7 '+(-height/2)
        +' L7 '+(height/2)
        +'" stroke="'+color+'" stroke-width="7"'
        +' stroke-linecap="round"/>'
        +'<path d="M7 '+(-height/2)
        +' L-7 '+(height/2)
        +'" stroke="'+color+'" stroke-width="7"'
        +' stroke-linecap="round"/>'
        +'<circle cx="0" cy="0" r="4.5"'
        +' fill="#FDE68A" stroke="#B45309"'
        +' stroke-width="1.5"/>'
        +'</g>';
    }

    function chromosomeSetGraphic(
      count,
      x,
      y,
      label,
      highlightIndex
    ){
      var html='';
      var columns=6;

      html+='<text x="'+x+'" y="'+(y-18)
        +'" text-anchor="middle" font-size="10.5"'
        +' font-weight="900" fill="#475569">'
        +label+'</text>';

      for(var i=0;i<count;i++){
        var px=
          x-105+(i%columns)*42;

        var py=
          y+Math.floor(i/columns)*48;

        var color=
          i%2===0
            ?'#2563EB'
            :'#EC4899';

        html+=chromosomeSymbol(
          px,
          py,
          color,
          i,
          i===highlightIndex
        );
      }

      return html;
    }

    function numberVariation(
      mutationKind,
      pairs
    ){
      var baseline=pairs*2;
      var changed=baseline;
      var name='';
      var ploidy='2n';
      var effect='染色体数目改变';

      if(mutationKind==='type1'){
        changed=baseline-1;
        name='单体';
        ploidy='2n−1';
        effect='一条染色体缺少';
      }else if(mutationKind==='type2'){
        changed=baseline+1;
        name='三体';
        ploidy='2n+1';
        effect='一条染色体增加';
      }else if(mutationKind==='type3'){
        changed=pairs*3;
        name='三倍体';
        ploidy='3n';
        effect='三个染色体组';
      }else{
        changed=pairs*4;
        name='四倍体';
        ploidy='4n';
        effect='四个染色体组';
      }

      return {
        baseline:baseline,
        changed:changed,
        name:name,
        ploidy:ploidy,
        effect:effect
      };
    }

    function percentWidth(
      value,
      max,
      width
    ){
      return width*clamp(
        value/Math.max(1,max),
        0,
        1
      );
    }

    function renderGeneMode(
      codonCount,
      position,
      fragmentSize
    ){
      var original=baseSequence(codonCount);
      var data=mutateGene(
        original,
        kind,
        position,
        fragmentSize
      );

      title.textContent=
        '基因突变：DNA序列发生碱基变化';

      summary.textContent=
        '比较替换、插入和缺失对编码序列、密码子和阅读框的影响';

      arrowLabel.textContent=
        data.name;

      originalLayer.innerHTML=
        sequenceGraphic(
          data.original,
          data.originalProtein,
          177,
          132,
          [],
          false
        );

      changedLayer.innerHTML=
        sequenceGraphic(
          data.mutated,
          data.changedProtein,
          583,
          132,
          data.changedIndices,
          true
        );

      labelLayer.innerHTML=
        showLabels
          ?'<text x="177" y="259" text-anchor="middle"'
            +' font-size="10.5" font-weight="900" fill="#1D4ED8">'
            +'原始编码序列'
            +'</text>'
            +'<text x="583" y="259" text-anchor="middle"'
            +' font-size="10.5" font-weight="900" fill="#9D174D">'
            +'红色边框表示变化位置'
            +'</text>'
          :'';

      statusType.textContent=
        data.name;

      statusAmount.textContent=
        data.lengthChange>0
          ?'+'+data.lengthChange+' bp'
          :data.lengthChange<0
            ?data.lengthChange+' bp'
            :'0 bp';

      statusEffect.textContent=
        data.effect;

      amountLabel.textContent=
        'DNA长度变化';

      panelTitle.textContent=
        '编码区变化结果比较';

      metricOneLabel.textContent=
        'DNA长度改变';

      metricTwoLabel.textContent=
        '翻译结果差异';

      metricThreeLabel.textContent=
        '阅读框影响';

      metricOneBar.setAttribute(
        'width',
        String(
          percentWidth(
            Math.abs(data.lengthChange),
            4,
            190
          )
        )
      );

      var originalProteinText=
        data.originalProtein.join('|');

      var changedProteinText=
        data.changedProtein.join('|');

      var proteinDifference=
        originalProteinText===changedProteinText
          ?0
          :data.frameshift
            ?100
            :55;

      metricTwoBar.setAttribute(
        'width',
        String(
          percentWidth(
            proteinDifference,
            100,
            225
          )
        )
      );

      metricThreeBar.setAttribute(
        'width',
        String(
          data.frameshift
            ?190
            :data.lengthChange!==0
              ?45
              :20
        )
      );

      panelNote.textContent=
        data.frameshift
          ?'变化碱基数不是3的倍数，后续密码子分组可能改变'
          :data.effect==='同义变化'
            ?'密码子改变，但示意氨基酸未改变'
            :'并非所有序列变化都会产生相同后果';

      footerNote.textContent=
        '本模式只展示蛋白质编码区；真实基因还包含调控区、内含子和其他结构。';

      root.style.setProperty(
        '--mv-flow-speed',
        clamp(
          2.2-fragmentSize*.22,
          .7,
          2.2
        ).toFixed(2)+'s'
      );

      var explanation='';

      if(kind==='type1'){
        if(data.effect==='同义变化'){
          explanation=
            '本次碱基替换改变了密码子，但由于遗传密码具有简并性，示意氨基酸没有改变。';
        }else if(data.effect==='终止密码子'){
          explanation=
            '本次碱基替换形成终止密码子，可能使翻译提前结束。';
        }else{
          explanation=
            '本次碱基替换改变了对应密码子的氨基酸含义。';
        }
      }else if(kind==='type2'){
        explanation=data.frameshift
          ?'插入的碱基数不是3的倍数，编码区后续密码子分组发生改变，形成移码示意。'
          :'插入的碱基数是3的倍数，增加了完整密码子，后续阅读框保持。';
      }else if(kind==='type3'){
        explanation=data.frameshift
          ?'缺失的碱基数不是3的倍数，编码区后续密码子分组发生改变，形成移码示意。'
          :'缺失的碱基数是3的倍数，减少了完整密码子，后续阅读框保持。';
      }else{
        explanation=
          '替换通常不改变序列长度；插入和缺失可能改变长度，其中非3倍数的编码区变化可能造成移码。';
      }

      result.innerHTML=
        explanation
        +'<br>基因突变不等于一定改变蛋白质或表现型；结果还取决于变化位置、基因功能、调控过程和环境。'
        +' 突变不是按照生物需要定向产生的。';
    }

    function renderStructureMode(
      position,
      fragmentSize
    ){
      var data=structureVariation(
        kind,
        position,
        fragmentSize
      );

      title.textContent=
        '染色体结构变异：片段发生重新排列';

      summary.textContent=
        '比较染色体片段缺失、重复、倒位和非同源染色体易位';

      arrowLabel.textContent=
        data.name;

      var originalHTML=
        chromosomeGraphic(
          data.original,
          177,
          140,
          '染色体1：A—F',
          false
        );

      if(kind==='type4'){
        originalHTML+=
          chromosomeGraphic(
            data.second,
            177,
            210,
            '染色体2：M—R',
            true
          );
      }

      originalLayer.innerHTML=
        originalHTML;

      var changedHTML=
        chromosomeGraphic(
          data.changed,
          583,
          140,
          '变化后染色体1',
          false
        );

      if(kind==='type4'){
        changedHTML+=
          chromosomeGraphic(
            data.changedSecond,
            583,
            210,
            '变化后染色体2',
            true
          );
      }

      changedLayer.innerHTML=
        changedHTML;

      labelLayer.innerHTML=
        showLabels
          ?'<text x="177" y="260" text-anchor="middle"'
            +' font-size="10.5" font-weight="900" fill="#1D4ED8">'
            +'不同颜色代表不同染色体片段'
            +'</text>'
            +'<text x="583" y="260" text-anchor="middle"'
            +' font-size="10.5" font-weight="900" fill="#9D174D">'
            +data.effect
            +'</text>'
          :'';

      statusType.textContent=
        data.name;

      var segmentDifference=
        data.changed.length
        -data.original.length;

      statusAmount.textContent=
        segmentDifference>0
          ?'+'+segmentDifference+' 段'
          :segmentDifference<0
            ?segmentDifference+' 段'
            :'0 段';

      statusEffect.textContent=
        data.effect;

      amountLabel.textContent=
        '片段数量变化';

      panelTitle.textContent=
        '染色体结构与遗传物质变化';

      metricOneLabel.textContent=
        '片段数量变化';

      metricTwoLabel.textContent=
        '片段顺序改变';

      metricThreeLabel.textContent=
        '涉及染色体数';

      metricOneBar.setAttribute(
        'width',
        String(
          percentWidth(
            Math.abs(segmentDifference),
            4,
            190
          )
        )
      );

      var orderChanged=
        data.original.join('')
        !==data.changed.join('');

      metricTwoBar.setAttribute(
        'width',
        String(
          orderChanged
            ?225
            :0
        )
      );

      metricThreeBar.setAttribute(
        'width',
        String(
          kind==='type4'
            ?190
            :95
        )
      );

      panelNote.textContent=
        kind==='type1'
          ?'缺失使部分遗传物质减少'
          :kind==='type2'
            ?'重复使局部片段拷贝增加'
            :kind==='type3'
              ?'倒位改变片段方向和排列顺序'
              :'易位涉及两条非同源染色体';

      footerNote.textContent=
        '染色体结构变异不等于染色体数目一定改变；本图不对应具体物种核型。';

      root.style.setProperty(
        '--mv-chromosome-speed',
        clamp(
          2.3-fragmentSize*.18,
          .8,
          2.3
        ).toFixed(2)+'s'
      );

      var explanation=
        data.name+'改变了染色体片段的数量、方向或位置。';

      if(kind==='type1'){
        explanation+=
          '片段缺失使相应区域的遗传物质减少。';
      }else if(kind==='type2'){
        explanation+=
          '片段重复使相应区域出现额外拷贝。';
      }else if(kind==='type3'){
        explanation+=
          '片段倒位通常不直接改变片段总量，但会改变基因排列方向和位置关系。';
      }else{
        explanation+=
          '本模型展示两条非同源染色体交换末端片段的简化易位。';
      }

      result.innerHTML=
        explanation
        +'<br>结构变异的影响取决于断点位置、涉及基因、基因剂量和调控环境。'
        +' 本模型只作课堂结构比较，不用于真实核型判断。';
    }

    function renderNumberMode(
      pairs
    ){
      var data=numberVariation(
        kind,
        pairs
      );

      title.textContent=
        '染色体数目变异：个别染色体或整个染色体组变化';

      summary.textContent=
        '比较单体、三体、三倍体和四倍体的染色体数目示意';

      arrowLabel.textContent=
        data.name;

      originalLayer.innerHTML=
        chromosomeSetGraphic(
          data.baseline,
          177,
          145,
          '正常二倍体：2n = '+data.baseline,
          -1
        );

      var highlightIndex=
        kind==='type2'
          ?data.changed-1
          :kind==='type1'
            ?Math.max(0,data.changed-1)
            :-1;

      changedLayer.innerHTML=
        chromosomeSetGraphic(
          data.changed,
          583,
          145,
          data.ploidy+' = '+data.changed,
          highlightIndex
        );

      labelLayer.innerHTML=
        showLabels
          ?'<text x="177" y="260" text-anchor="middle"'
            +' font-size="10.5" font-weight="900" fill="#1D4ED8">'
            +'每对同源染色体用两种颜色交替示意'
            +'</text>'
            +'<text x="583" y="260" text-anchor="middle"'
            +' font-size="10.5" font-weight="900" fill="#9D174D">'
            +data.effect
            +'</text>'
          :'';

      statusType.textContent=
        data.name;

      var countDifference=
        data.changed-data.baseline;

      statusAmount.textContent=
        countDifference>0
          ?'+'+countDifference+' 条'
          :countDifference+' 条';

      statusEffect.textContent=
        data.ploidy;

      amountLabel.textContent=
        '染色体数变化';

      panelTitle.textContent=
        '非整倍性与整倍性变化比较';

      metricOneLabel.textContent=
        '染色体数变化';

      metricTwoLabel.textContent=
        '染色体组变化';

      metricThreeLabel.textContent=
        '相对基因剂量';

      metricOneBar.setAttribute(
        'width',
        String(
          percentWidth(
            Math.abs(countDifference),
            Math.max(1,pairs*2),
            190
          )
        )
      );

      var groupChange=
        kind==='type3'
          ?50
          :kind==='type4'
            ?100
            :10;

      metricTwoBar.setAttribute(
        'width',
        String(
          percentWidth(
            groupChange,
            100,
            225
          )
        )
      );

      metricThreeBar.setAttribute(
        'width',
        String(
          percentWidth(
            data.changed,
            pairs*4,
            190
          )
        )
      );

      panelNote.textContent=
        kind==='type1'
          ?'单体属于非整倍性：少一条染色体'
          :kind==='type2'
            ?'三体属于非整倍性：多一条染色体'
            :kind==='type3'
              ?'三倍体含三个完整染色体组'
              :'四倍体含四个完整染色体组';

      footerNote.textContent=
        '非整倍性影响个别染色体数目；多倍体改变完整染色体组数。';

      root.style.setProperty(
        '--mv-chromosome-speed',
        clamp(
          2.5-data.changed/18,
          .75,
          2.4
        ).toFixed(2)+'s'
      );

      var explanation='';

      if(kind==='type1'){
        explanation=
          '单体示意为2n−1，即某一对同源染色体中缺少一条染色体。';
      }else if(kind==='type2'){
        explanation=
          '三体示意为2n+1，即某一种染色体出现三条。';
      }else if(kind==='type3'){
        explanation=
          '三倍体示意为3n，即细胞中存在三个完整的染色体组。';
      }else{
        explanation=
          '四倍体示意为4n，即细胞中存在四个完整的染色体组。';
      }

      result.innerHTML=
        explanation
        +'<br>单体和三体属于非整倍性变化；三倍体和四倍体属于整倍性变化。'
        +' 本模型不表示具体物种，也不用于医学核型分析。';
    }

    function update(){
      var codonCount=clamp(
        Math.round(
          Number(codonsInput.value)
        ),
        3,
        8
      );

      var sequenceLength=
        codonCount*3;

      var position=clamp(
        Math.round(
          Number(positionInput.value)
        ),
        1,
        sequenceLength
      );

      var fragmentSize=clamp(
        Math.round(
          Number(sizeInput.value)
        ),
        1,
        4
      );

      var pairs=clamp(
        Math.round(
          Number(pairsInput.value)
        ),
        2,
        6
      );

      positionInput.max=String(
        sequenceLength
      );

      if(Number(positionInput.value)>sequenceLength){
        positionInput.value=String(
          sequenceLength
        );
        position=sequenceLength;
      }

      codonsValue.textContent=
        codonCount+' 个';

      positionValue.textContent=
        '第 '+position+' 位';

      sizeValue.textContent=
        fragmentSize+' 个单位';

      pairsValue.textContent=
        pairs+' 对';

      labelToggle.textContent=
        showLabels
          ?'结构标注：显示'
          :'结构标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      setModeActive();
      setKindActive();

      if(mode==='gene'){
        renderGeneMode(
          codonCount,
          position,
          fragmentSize
        );
      }else if(mode==='structure'){
        renderStructureMode(
          position,
          fragmentSize
        );
      }else{
        renderNumberMode(
          pairs
        );
      }
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute(
          'data-mode'
        );
        kind='type1';
        setScenarioActive('');
        update();
      };
    }

    for(var j=0;j<kindButtons.length;j++){
      kindButtons[j].onclick=function(){
        kind=this.getAttribute(
          'data-kind'
        );
        setScenarioActive('');
        update();
      };
    }

    for(var k=0;k<scenarioButtons.length;k++){
      scenarioButtons[k].onclick=function(){
        var name=this.getAttribute(
          'data-scenario'
        );
        var data=scenarios[name];

        if(!data){
          return;
        }

        mode=data.mode;
        kind=data.kind;
        codonsInput.value=String(data.codons);
        positionInput.value=String(data.position);
        sizeInput.value=String(data.size);
        pairsInput.value=String(data.pairs);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    codonsInput.oninput=function(){
      setScenarioActive('');
      update();
    };

    positionInput.oninput=function(){
      setScenarioActive('');
      update();
    };

    sizeInput.oninput=function(){
      setScenarioActive('');
      update();
    };

    pairsInput.oninput=function(){
      setScenarioActive('');
      update();
    };

    update();
  })();
  ${SCRIPT_END}
</div>`
    },
  },
]
