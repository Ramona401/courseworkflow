/**
 * lifeScienceLabTemplatesEcologyNitrogenCycle.ts
 *
 * 平面生命科学实验室：生态系统中的氮循环。
 *
 * 教学目标：
 * 1. 识别大气氮、土壤铵态氮、硝态氮、生物有机氮
 *    和水体氮等主要氮库；
 * 2. 理解多数生物不能直接利用大气中的氮气；
 * 3. 理解固氮作用可把大气氮转化为生态系统可进一步利用的含氮物质；
 * 4. 理解分解者通过氨化作用把遗体、排遗物和枯落物中的
 *    有机氮转化为铵态氮；
 * 5. 理解硝化细菌在有氧条件下可把铵态氮逐步转化为硝态氮；
 * 6. 理解植物可吸收铵态氮和硝态氮合成含氮有机物，
 *    动物通过取食获得含氮物质；
 * 7. 理解反硝化细菌在低氧条件下可把部分硝态氮转化为氮气，
 *    使氮重新进入大气；
 * 8. 比较豆科共生、施肥过量、土壤水淹和干旱等条件
 *    对氮循环过程的影响；
 * 9. 区分氮元素循环与能量流动：
 *    氮元素可在生物群落和非生物环境之间循环，
 *    能量通常单向流动并逐级散失。
 *
 * 教学边界：
 * 1. 所有氮库大小、氮通量和可利用氮水平均为相对教学指标；
 * 2. 本模型统一使用“铵态氮”和“硝态氮”表达主要土壤无机氮，
 *    不展开亚硝态氮等短暂中间产物；
 * 3. 固氮包括生物固氮、闪电固氮和工业固氮等来源，
 *    本模型重点表现微生物固氮及豆科共生固氮；
 * 4. 硝化作用通常需要氧气，反硝化作用常在低氧环境中增强，
 *    但真实过程还受温度、水分、pH和碳源等多种因素影响；
 * 5. 植物对不同形态氮的吸收能力因物种、器官和环境而异，
 *    本模型用综合吸收通量简化表示；
 * 6. 肥料输入可以提高土壤可利用氮，但过量时会增加淋失、
 *    径流、水体富营养化和气态氮损失风险；
 * 7. 干旱和水淹都可能抑制植物生长，但对硝化、反硝化和淋失
 *    的影响方向并不相同；
 * 8. 大气氮库远大于短期生态系统中的其他氮库，
 *    本模型的图形比例不代表真实库容量比例；
 * 9. 本模型不用于真实农业施肥、环境监测或水体污染预测。
 *
 * 工程约束：
 * 1. 使用纯HTML、SVG和原生JavaScript；
 * 2. 不依赖外部脚本、样式、字体、图片或CDN；
 * 3. 所有CSS、DOM查询和事件均限定在rootId内部；
 * 4. 使用统一.bl-*公共布局协议；
 * 5. 支持同一课件页放置多个独立实例；
 * 6. 不使用document.querySelector或document.querySelectorAll；
 * 7. 本文件只导出独立模板数组，聚合接入由第27批C1完成。
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
 * 构建完全限定在当前rootId内的样式。
 *
 * 独立预览时保留左侧控制区；
 * 嵌入课件后由lifeScienceLabUtils.ts中的公共覆盖层
 * 转换为“上方实验主体 + 底部课堂控制条”。
 */
function nitrogenCycleStyle(rootId: string): string {
  return ''
    + '<style>'
    + '#' + rootId + '{width:100%;height:100%;overflow:hidden;border:1px solid #C4B5FD;border-radius:16px;background:#fff;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;color:#1F2937}'
    + '#' + rootId + ' *{box-sizing:border-box}'
    + '#' + rootId + ' .bl-head{height:46px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#EDE9FE,#DCFCE7);border-bottom:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-title{font-size:15px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .bl-note{font-size:12px;color:#64748B}'
    + '#' + rootId + ' .bl-body{height:calc(100% - 46px);display:grid;grid-template-columns:250px minmax(0,1fr);min-height:0}'
    + '#' + rootId + ' .bl-controls{padding:12px;overflow:auto;background:#FCFAFF;border-right:1px solid #C4B5FD}'
    + '#' + rootId + ' .bl-stage{position:relative;min-width:0;min-height:0;background:#fff}'
    + '#' + rootId + ' .bl-row{margin-bottom:9px}'
    + '#' + rootId + ' .bl-label{display:flex;justify-content:space-between;gap:8px;margin-bottom:4px;font-size:11.5px;font-weight:700;color:#334155}'
    + '#' + rootId + ' .bl-value{font-weight:800;color:#7C3AED;white-space:nowrap}'
    + '#' + rootId + ' input[type=range]{width:100%;accent-color:#8B5CF6}'
    + '#' + rootId + ' .nc-subtitle{margin:6px 0;font-size:11.5px;font-weight:800;color:#5B21B6}'
    + '#' + rootId + ' .nc-buttons{display:grid;grid-template-columns:repeat(3,1fr);gap:4px;margin-bottom:7px}'
    + '#' + rootId + ' .nc-scenarios{display:grid;grid-template-columns:repeat(5,1fr);gap:3px;margin-bottom:7px}'
    + '#' + rootId + ' .nc-button{min-height:30px;padding:3px;border:1px solid #C4B5FD;border-radius:8px;background:#fff;color:#5B21B6;font-size:9.5px;font-weight:800;line-height:1.15;cursor:pointer}'
    + '#' + rootId + ' .nc-button.active{border-color:#8B5CF6;background:#EDE9FE;box-shadow:0 3px 9px rgba(124,58,237,.14)}'
    + '#' + rootId + ' .nc-toggle{width:100%;height:31px;margin-bottom:7px;border:0;border-radius:8px;background:linear-gradient(135deg,#A78BFA,#7C3AED);color:#fff;font-size:10.5px;font-weight:800;cursor:pointer}'
    + '#' + rootId + ' .nc-toggle.off{background:#64748B}'
    + '#' + rootId + ' .nc-status{display:grid;grid-template-columns:repeat(3,1fr);gap:5px;margin-bottom:8px}'
    + '#' + rootId + ' .nc-card{padding:6px 3px;border:1px solid #DDD6FE;border-radius:8px;background:#fff;text-align:center}'
    + '#' + rootId + ' .nc-card b{display:block;min-height:18px;font-size:13px;color:#6D28D9}'
    + '#' + rootId + ' .nc-card span{font-size:8.8px;color:#64748B}'
    + '#' + rootId + ' .bl-result{padding:8px 9px;border-radius:10px;background:#EDE9FE;color:#4C1D95;font-size:10.8px;line-height:1.45;font-weight:600}'
    + '#' + rootId + ' svg{display:block;width:100%;height:100%}'
    + '#' + rootId + ' .nc-nitrogen{animation:' + rootId + '-nitrogen var(--nc-nitrogen-speed,1.6s) ease-in-out infinite alternate}'
    + '#' + rootId + ' .nc-flow{stroke-dasharray:9 7;animation:' + rootId + '-flow var(--nc-flow-speed,1.3s) linear infinite}'
    + '#' + rootId + ' .nc-cloud{animation:' + rootId + '-cloud 2.4s ease-in-out infinite alternate}'
    + '#' + rootId + ' .nc-microbe{animation:' + rootId + '-microbe 1.5s ease-in-out infinite alternate}'
    + '#' + rootId + ' .nc-pulse{animation:' + rootId + '-pulse 1.1s ease-in-out infinite alternate}'
    + '@keyframes ' + rootId + '-nitrogen{from{opacity:.42}to{opacity:1}}'
    + '@keyframes ' + rootId + '-flow{to{stroke-dashoffset:-32}}'
    + '@keyframes ' + rootId + '-cloud{from{transform:translateX(-4px);opacity:.72}to{transform:translateX(5px);opacity:1}}'
    + '@keyframes ' + rootId + '-microbe{from{transform:translateY(2px);opacity:.5}to{transform:translateY(-3px);opacity:1}}'
    + '@keyframes ' + rootId + '-pulse{from{opacity:.4}to{opacity:1}}'
    + '</style>'
}

const SCRIPT_END = '</' + 'script>'

export const LIFE_SCIENCE_LAB_TEMPLATES_ECOLOGY_NITROGEN_CYCLE:
LifeScienceLabTemplate[] = [
  {
    id: 'biology-ecosystem-nitrogen-cycle',
    group: '🌎 生态系统',
    name: '生态系统中的氮循环',
    emoji: '🧫',
    desc: '调节固氮、分解、土壤氧气、植物需求、肥料输入和作用时间，观察氨化、硝化、同化、反硝化及氮损失',
    params: [
      {
        key: 'nitrogenFixation',
        label: '固氮作用活跃度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 62,
      },
      {
        key: 'decompositionActivity',
        label: '分解与氨化活跃度',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 58,
      },
      {
        key: 'soilOxygen',
        label: '土壤氧气水平',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 72,
        hint: '有氧促进硝化，低氧有利于反硝化',
      },
      {
        key: 'plantDemand',
        label: '植物吸氮需求',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 76,
      },
      {
        key: 'fertilizerInput',
        label: '含氮肥料输入',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 18,
      },
      {
        key: 'observationTime',
        label: '作用时间',
        type: 'number',
        min: 0,
        max: 100,
        step: 1,
        defaultValue: 64,
      },
      {
        key: 'showLabels',
        label: '显示过程标注',
        type: 'boolean',
        defaultValue: true,
      },
    ],

    buildHTML: (params, rootId) => {
      const nitrogenFixation = num(
        params,
        'nitrogenFixation',
        62,
      )
      const decompositionActivity = num(
        params,
        'decompositionActivity',
        58,
      )
      const soilOxygen = num(
        params,
        'soilOxygen',
        72,
      )
      const plantDemand = num(
        params,
        'plantDemand',
        76,
      )
      const fertilizerInput = num(
        params,
        'fertilizerInput',
        18,
      )
      const observationTime = num(
        params,
        'observationTime',
        64,
      )
      const showLabels = bool(
        params,
        'showLabels',
        true,
      )

      return `
<div id="${rootId}">
${nitrogenCycleStyle(rootId)}
  <div class="bl-head">
    <div class="bl-title">🧫 生态系统中的氮循环</div>
    <div class="bl-note">微生物转化连接大气氮、土壤无机氮与生物有机氮</div>
  </div>

  <div class="bl-body">
    <div class="bl-controls">
      <div class="bl-row">
        <div class="bl-label">
          <span>固氮作用活跃度</span>
          <span class="bl-value" data-fixation-value></span>
        </div>
        <input data-fixation type="range" min="0" max="100" step="1" value="${n(nitrogenFixation)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>分解与氨化活跃度</span>
          <span class="bl-value" data-decomposition-value></span>
        </div>
        <input data-decomposition type="range" min="0" max="100" step="1" value="${n(decompositionActivity)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>土壤氧气水平</span>
          <span class="bl-value" data-oxygen-value></span>
        </div>
        <input data-oxygen type="range" min="0" max="100" step="1" value="${n(soilOxygen)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>植物吸氮需求</span>
          <span class="bl-value" data-demand-value></span>
        </div>
        <input data-demand type="range" min="0" max="100" step="1" value="${n(plantDemand)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>含氮肥料输入</span>
          <span class="bl-value" data-fertilizer-value></span>
        </div>
        <input data-fertilizer type="range" min="0" max="100" step="1" value="${n(fertilizerInput)}">
      </div>

      <div class="bl-row">
        <div class="bl-label">
          <span>作用时间</span>
          <span class="bl-value" data-time-value></span>
        </div>
        <input data-time type="range" min="0" max="100" step="1" value="${n(observationTime)}">
      </div>

      <div class="nc-subtitle">重点观察</div>

      <div class="nc-buttons">
        <button type="button" class="nc-button active" data-mode="system">完整循环</button>
        <button type="button" class="nc-button" data-mode="microbes">微生物转化</button>
        <button type="button" class="nc-button" data-mode="agriculture">农业影响</button>
      </div>

      <div class="nc-subtitle">快速比较情境</div>

      <div class="nc-scenarios">
        <button type="button" class="nc-button active" data-scenario="balanced">相对平衡</button>
        <button type="button" class="nc-button" data-scenario="legume">豆科共生</button>
        <button type="button" class="nc-button" data-scenario="fertilized">过量施肥</button>
        <button type="button" class="nc-button" data-scenario="waterlogged">土壤水淹</button>
        <button type="button" class="nc-button" data-scenario="drought">干旱</button>
      </div>

      <button type="button" class="nc-toggle${showLabels ? '' : ' off'}" data-label-toggle>
        ${showLabels ? '过程标注：显示' : '过程标注：隐藏'}
      </button>

      <div class="nc-status">
        <div class="nc-card">
          <b data-available-nitrogen></b>
          <span>土壤可利用氮</span>
        </div>

        <div class="nc-card">
          <b data-system-state></b>
          <span>当前氮状态</span>
        </div>

        <div class="nc-card">
          <b data-main-process></b>
          <span>主要影响过程</span>
        </div>
      </div>

      <div class="bl-result" data-result></div>
    </div>

    <div class="bl-stage">
      <svg
        viewBox="0 0 760 430"
        aria-label="生态系统氮循环互动示意图"
      >
        <defs>
          <linearGradient id="${rootId}-sky" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E0F2FE"/>
            <stop offset="100%" stop-color="#F8FAFC"/>
          </linearGradient>

          <linearGradient id="${rootId}-soil" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#E7D3A8"/>
            <stop offset="100%" stop-color="#9A642E"/>
          </linearGradient>

          <linearGradient id="${rootId}-water" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#BAE6FD"/>
            <stop offset="100%" stop-color="#0284C7"/>
          </linearGradient>

          <marker id="${rootId}-arrow-purple" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#7C3AED"/>
          </marker>

          <marker id="${rootId}-arrow-green" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#16A34A"/>
          </marker>

          <marker id="${rootId}-arrow-orange" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#F59E0B"/>
          </marker>

          <marker id="${rootId}-arrow-blue" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#0284C7"/>
          </marker>

          <marker id="${rootId}-arrow-red" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#DC2626"/>
          </marker>

          <marker id="${rootId}-arrow-brown" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto">
            <path d="M0,0 L0,6 L8,3 z" fill="#92400E"/>
          </marker>

          <filter id="${rootId}-shadow">
            <feDropShadow dx="0" dy="5" stdDeviation="6" flood-color="#4C1D95" flood-opacity=".13"/>
          </filter>
        </defs>

        <rect width="760" height="430" fill="#FFFFFF"/>
        <rect x="0" y="0" width="760" height="216" fill="url(#${rootId}-sky)" opacity=".72"/>
        <rect x="0" y="216" width="610" height="214" fill="url(#${rootId}-soil)" opacity=".72"/>
        <path d="M610 250 C660 230 708 236 760 220 V430 H610Z" fill="url(#${rootId}-water)" opacity=".82"/>

        <text x="22" y="34" data-title font-size="25" font-weight="900" fill="#5B21B6"></text>
        <text x="22" y="62" data-summary font-size="13" font-weight="800" fill="#475569"></text>

        <!-- 大气氮库 -->
        <g class="nc-cloud" filter="url(#${rootId}-shadow)">
          <path
            d="M261 78
               C268 49 296 42 318 56
               C334 31 374 33 388 60
               C417 53 442 74 436 100
               H275
               C254 100 244 87 261 78Z"
            fill="#FFFFFF"
            stroke="#64748B"
            stroke-width="4"
          />

          <text x="347" y="75" text-anchor="middle" font-size="14" font-weight="900" fill="#334155">大气氮库</text>
          <text x="347" y="94" text-anchor="middle" font-size="12" font-weight="900" fill="#475569">N₂</text>
        </g>

        <!-- 植物有机氮 -->
        <g filter="url(#${rootId}-shadow)">
          <path d="M144 290 V174" stroke="#15803D" stroke-width="17" stroke-linecap="round"/>
          <path d="M144 202 C101 169 64 177 37 208 C78 237 116 231 144 202Z" fill="#4ADE80" stroke="#15803D" stroke-width="4"/>
          <path d="M144 190 C186 160 227 168 252 199 C215 227 176 221 144 190Z" fill="#22C55E" stroke="#15803D" stroke-width="4"/>
          <path d="M144 288 C118 306 100 323 84 343 M144 288 C171 307 193 324 217 344" fill="none" stroke="#92400E" stroke-width="8" stroke-linecap="round"/>

          <g data-root-nodules></g>

          <rect x="55" y="294" width="183" height="47" rx="18" fill="#FFFFFF" stroke="#22C55E" stroke-width="3"/>
          <text x="146" y="313" text-anchor="middle" font-size="13" font-weight="900" fill="#166534">植物有机氮库</text>
          <rect x="76" y="323" width="140" height="10" rx="5" fill="#E2E8F0"/>
          <rect data-plant-bar x="76" y="323" width="0" height="10" rx="5" fill="#22C55E"/>
        </g>

        <!-- 动物有机氮 -->
        <g filter="url(#${rootId}-shadow)">
          <ellipse cx="320" cy="221" rx="54" ry="33" fill="#FEF3C7" stroke="#D97706" stroke-width="4"/>
          <circle cx="365" cy="210" r="22" fill="#FDE68A" stroke="#D97706" stroke-width="4"/>
          <ellipse cx="374" cy="193" rx="8" ry="16" fill="#FDE68A" stroke="#D97706" stroke-width="3" transform="rotate(20 374 193)"/>
          <circle cx="372" cy="206" r="3" fill="#111827"/>
          <path d="M286 245 L276 262 M318 251 L311 269 M342 249 L352 267" stroke="#92400E" stroke-width="5" stroke-linecap="round"/>

          <rect x="252" y="275" width="153" height="46" rx="18" fill="#FFFFFF" stroke="#F59E0B" stroke-width="3"/>
          <text x="328" y="294" text-anchor="middle" font-size="13" font-weight="900" fill="#92400E">动物有机氮库</text>
          <rect x="268" y="304" width="121" height="10" rx="5" fill="#E2E8F0"/>
          <rect data-animal-bar x="268" y="304" width="0" height="10" rx="5" fill="#F59E0B"/>
        </g>

        <!-- 有机氮与分解者 -->
        <g filter="url(#${rootId}-shadow)">
          <rect x="54" y="356" width="338" height="58" rx="23" fill="#D6B47C" stroke="#92400E" stroke-width="4"/>
          <g data-decomposer-microbes></g>
          <text x="229" y="375" font-size="13" font-weight="900" fill="#78350F">遗体、排遗物与土壤有机氮</text>
          <rect x="219" y="385" width="148" height="10" rx="5" fill="#E2E8F0"/>
          <rect data-organic-bar x="219" y="385" width="0" height="10" rx="5" fill="#92400E"/>
          <text x="229" y="407" font-size="10.5" font-weight="800" fill="#78350F">分解者通过氨化作用释放铵态氮</text>
        </g>

        <!-- 铵态氮库 -->
        <g filter="url(#${rootId}-shadow)">
          <rect x="419" y="288" width="139" height="57" rx="18" fill="#F3E8FF" stroke="#7C3AED" stroke-width="4"/>
          <text x="488" y="309" text-anchor="middle" font-size="13" font-weight="900" fill="#5B21B6">铵态氮库</text>
          <text x="488" y="327" text-anchor="middle" font-size="11" font-weight="900" fill="#6D28D9">NH₄⁺</text>
          <rect x="440" y="334" width="97" height="8" rx="4" fill="#E2E8F0"/>
          <rect data-ammonium-bar x="440" y="334" width="0" height="8" rx="4" fill="#8B5CF6"/>
        </g>

        <!-- 硝态氮库 -->
        <g filter="url(#${rootId}-shadow)">
          <rect x="419" y="359" width="139" height="56" rx="18" fill="#E0F2FE" stroke="#0284C7" stroke-width="4"/>
          <text x="488" y="380" text-anchor="middle" font-size="13" font-weight="900" fill="#075985">硝态氮库</text>
          <text x="488" y="398" text-anchor="middle" font-size="11" font-weight="900" fill="#0369A1">NO₃⁻</text>
          <rect x="440" y="404" width="97" height="8" rx="4" fill="#E2E8F0"/>
          <rect data-nitrate-bar x="440" y="404" width="0" height="8" rx="4" fill="#0EA5E9"/>
        </g>

        <!-- 水体氮与淋失 -->
        <g filter="url(#${rootId}-shadow)">
          <ellipse cx="681" cy="327" rx="60" ry="34" fill="#E0F2FE" stroke="#0369A1" stroke-width="4"/>
          <path d="M632 322 Q648 309 664 322 T696 322 T728 322" fill="none" stroke="#38BDF8" stroke-width="4"/>
          <text x="681" y="344" text-anchor="middle" font-size="12" font-weight="900" fill="#075985">水体无机氮</text>
          <rect x="643" y="354" width="76" height="8" rx="4" fill="#DBEAFE"/>
          <rect data-water-bar x="643" y="354" width="0" height="8" rx="4" fill="#0284C7"/>
        </g>

        <!-- 动态过程 -->
        <g data-flow-layer></g>
        <g data-particle-layer></g>
        <g data-microbe-layer></g>
        <g data-fertilizer-layer></g>

        <!-- 过程标注 -->
        <g data-label-layer>
          <text x="89" y="135" font-size="11.5" font-weight="900" fill="#6D28D9">固氮：N₂进入土壤氮库</text>
          <text x="185" y="348" font-size="11.5" font-weight="900" fill="#78350F">氨化：有机氮 → NH₄⁺</text>
          <text x="425" y="264" font-size="11.5" font-weight="900" fill="#0369A1">硝化：NH₄⁺ → NO₃⁻</text>
          <text x="82" y="274" font-size="11.5" font-weight="900" fill="#166534">同化：植物吸收无机氮</text>
          <text x="486" y="145" font-size="11.5" font-weight="900" fill="#B91C1C">反硝化：NO₃⁻ → N₂</text>
          <text x="608" y="288" font-size="11.5" font-weight="900" fill="#075985">淋失与径流</text>
        </g>

        <!-- 右上角过程状态 -->
        <g transform="translate(535 39)">
          <rect width="202" height="115" rx="18" fill="#FFFFFF" stroke="#CBD5E1" stroke-width="2"/>

          <text x="101" y="25" text-anchor="middle" font-size="13" font-weight="900" fill="#334155">微生物转化状态</text>

          <text x="13" y="51" font-size="10.5" font-weight="800" fill="#64748B">硝化作用</text>
          <rect x="72" y="42" width="112" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-nitrification-bar x="72" y="42" width="0" height="12" rx="6" fill="#0EA5E9"/>

          <text x="13" y="77" font-size="10.5" font-weight="800" fill="#64748B">反硝化</text>
          <rect x="72" y="68" width="112" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-denitrification-bar x="72" y="68" width="0" height="12" rx="6" fill="#EF4444"/>

          <text x="13" y="103" font-size="10.5" font-weight="800" fill="#64748B">植物同化</text>
          <rect x="72" y="94" width="112" height="12" rx="6" fill="#E2E8F0"/>
          <rect data-assimilation-bar x="72" y="94" width="0" height="12" rx="6" fill="#22C55E"/>
        </g>
      </svg>
    </div>
  </div>

  <script>
  (function(){
    var root=document.getElementById('${rootId}');
    if(!root)return;

    var fixation=root.querySelector('[data-fixation]');
    var decomposition=root.querySelector('[data-decomposition]');
    var oxygen=root.querySelector('[data-oxygen]');
    var demand=root.querySelector('[data-demand]');
    var fertilizer=root.querySelector('[data-fertilizer]');
    var time=root.querySelector('[data-time]');

    var fixationValue=root.querySelector('[data-fixation-value]');
    var decompositionValue=root.querySelector('[data-decomposition-value]');
    var oxygenValue=root.querySelector('[data-oxygen-value]');
    var demandValue=root.querySelector('[data-demand-value]');
    var fertilizerValue=root.querySelector('[data-fertilizer-value]');
    var timeValue=root.querySelector('[data-time-value]');

    var modeButtons=root.querySelectorAll('[data-mode]');
    var scenarioButtons=root.querySelectorAll('[data-scenario]');
    var labelToggle=root.querySelector('[data-label-toggle]');

    var availableNitrogenText=root.querySelector('[data-available-nitrogen]');
    var systemStateText=root.querySelector('[data-system-state]');
    var mainProcessText=root.querySelector('[data-main-process]');
    var result=root.querySelector('[data-result]');

    var title=root.querySelector('[data-title]');
    var summary=root.querySelector('[data-summary]');
    var flowLayer=root.querySelector('[data-flow-layer]');
    var particleLayer=root.querySelector('[data-particle-layer]');
    var microbeLayer=root.querySelector('[data-microbe-layer]');
    var fertilizerLayer=root.querySelector('[data-fertilizer-layer]');
    var labelLayer=root.querySelector('[data-label-layer]');

    var rootNodules=root.querySelector('[data-root-nodules]');
    var decomposerMicrobes=root.querySelector('[data-decomposer-microbes]');

    var plantBar=root.querySelector('[data-plant-bar]');
    var animalBar=root.querySelector('[data-animal-bar]');
    var organicBar=root.querySelector('[data-organic-bar]');
    var ammoniumBar=root.querySelector('[data-ammonium-bar]');
    var nitrateBar=root.querySelector('[data-nitrate-bar]');
    var waterBar=root.querySelector('[data-water-bar]');

    var nitrificationBar=root.querySelector('[data-nitrification-bar]');
    var denitrificationBar=root.querySelector('[data-denitrification-bar]');
    var assimilationBar=root.querySelector('[data-assimilation-bar]');

    var mode='system';
    var showLabels=${showLabels ? 'true' : 'false'};

    var scenarios={
      balanced:{
        fixation:62,
        decomposition:58,
        oxygen:72,
        demand:76,
        fertilizer:18,
        time:64
      },
      legume:{
        fixation:96,
        decomposition:58,
        oxygen:72,
        demand:84,
        fertilizer:5,
        time:72
      },
      fertilized:{
        fixation:44,
        decomposition:54,
        oxygen:68,
        demand:74,
        fertilizer:96,
        time:72
      },
      waterlogged:{
        fixation:46,
        decomposition:64,
        oxygen:8,
        demand:38,
        fertilizer:45,
        time:72
      },
      drought:{
        fixation:34,
        decomposition:22,
        oxygen:88,
        demand:30,
        fertilizer:35,
        time:72
      }
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

    /**
     * 生成带方向箭头的氮通量路径。
     */
    function flow(
      path,
      color,
      marker,
      width,
      opacity
    ){
      return '<path class="nc-flow" d="'+path
        +'" fill="none" stroke="'+color
        +'" stroke-width="'+width
        +'" marker-end="url(#${rootId}-'+marker+')'
        +'" opacity="'+opacity+'"/>';
    }

    /**
     * 沿两点之间生成氮颗粒。
     */
    function particles(
      x1,
      y1,
      x2,
      y2,
      count,
      color,
      symbol
    ){
      var html='';

      for(var i=0;i<count;i++){
        var ratio=(i+1)/(count+1);
        var x=x1+(x2-x1)*ratio;
        var y=y1+(y2-y1)*ratio;
        var offset=(i%2===0?-1:1)*(4+i%3);

        html+='<circle class="nc-nitrogen" cx="'
          +(x+offset).toFixed(1)
          +'" cy="'+(y-offset*.3).toFixed(1)
          +'" r="'+(3.5+i%2)
          +'" fill="'+color+'" opacity=".86"/>';

        if(symbol && i%3===0){
          html+='<text x="'+(x+offset+6).toFixed(1)
            +'" y="'+(y-offset*.3+3).toFixed(1)
            +'" font-size="7.5" font-weight="900"'
            +' fill="'+color+'">'+symbol+'</text>';
        }
      }

      return html;
    }

    /**
     * 计算主要氮循环过程的相对通量。
     *
     * 公式只用于课堂比较，不代表真实氮循环定量模型。
     */
    function calculateFluxes(
      fixationLevel,
      decompositionLevel,
      oxygenLevel,
      demandLevel,
      fertilizerLevel,
      timeLevel
    ){
      var timeFactor=clamp(
        .3+.7*(1-Math.exp(-timeLevel/28)),
        .3,
        1
      );

      var fixationFactor=
        fixationLevel/(fixationLevel+26);

      var decompositionFactor=
        decompositionLevel/(decompositionLevel+28);

      var oxygenFactor=
        oxygenLevel/(oxygenLevel+24);

      var lowOxygenFactor=
        (100-oxygenLevel)
        /((100-oxygenLevel)+24);

      var demandFactor=
        demandLevel/(demandLevel+24);

      var fixationFlux=68
        *fixationFactor
        *(.45+.55*timeFactor);

      var ammonificationFlux=58
        *decompositionFactor
        *(.42+.58*timeFactor);

      var nitrificationFlux=67
        *oxygenFactor
        *(.42+.58*timeFactor);

      var denitrificationFlux=59
        *lowOxygenFactor
        *(.38+.62*timeFactor);

      var fertilizerFlux=72
        *fertilizerLevel/100
        *(.4+.6*timeFactor);

      var availableInput=
        fixationFlux
        +ammonificationFlux
        +fertilizerFlux;

      var assimilationFlux=78
        *demandFactor
        *clamp(
          availableInput/(availableInput+42),
          0,
          1
        )
        *(.4+.6*timeFactor);

      var feedingFlux=34
        *assimilationFlux/78
        *(.45+.55*timeFactor);

      var organicReturn=31
        *(
          assimilationFlux/78*.58
          +feedingFlux/34*.42
        )
        *(.5+.5*timeFactor);

      var leachingFlux=47
        *fertilizerLevel/100
        *(
          .18
          +.46*lowOxygenFactor
        )
        *(.45+.55*timeFactor);

      return {
        fixation:fixationFlux,
        ammonification:ammonificationFlux,
        nitrification:nitrificationFlux,
        denitrification:denitrificationFlux,
        fertilizer:fertilizerFlux,
        assimilation:assimilationFlux,
        feeding:feedingFlux,
        organicReturn:organicReturn,
        leaching:leachingFlux
      };
    }

    /**
     * 绘制根瘤和固氮微生物。
     */
    function buildRootNodules(level){
      var count=Math.floor(1+level/18);
      var html='';

      for(var i=0;i<count;i++){
        var x=101+(i%4)*28;
        var y=318+Math.floor(i/4)*16+(i%2)*5;

        html+='<circle class="nc-microbe" cx="'+x
          +'" cy="'+y+'" r="'+(5+i%2)
          +'" fill="#C4B5FD" stroke="#6D28D9"'
          +' stroke-width="2"/>';

        if(i%2===0){
          html+='<circle cx="'+(x+2)+'" cy="'+(y-1)
            +'" r="1.8" fill="#FFFFFF"/>';
        }
      }

      return html;
    }

    /**
     * 绘制有机氮库中的分解者。
     */
    function buildDecomposerMicrobes(level){
      var count=Math.floor(2+level/16);
      var html='';

      for(var i=0;i<count;i++){
        var x=76+(i%6)*23;
        var y=375+Math.floor(i/6)*18+(i%2)*4;

        html+='<ellipse class="nc-microbe" cx="'+x
          +'" cy="'+y+'" rx="'+(6+i%2)
          +'" ry="'+(4+i%3)
          +'" fill="#8B5CF6" stroke="#5B21B6"'
          +' stroke-width="1.5"/>';
      }

      return html;
    }

    /**
     * 绘制硝化和反硝化细菌。
     */
    function buildMicrobes(
      nitrification,
      denitrification,
      visible
    ){
      if(!visible){
        return '';
      }

      var html='';
      var nitrifierCount=Math.floor(
        1+nitrification/18
      );
      var denitrifierCount=Math.floor(
        1+denitrification/18
      );

      for(var i=0;i<nitrifierCount;i++){
        var x=399+(i%4)*22;
        var y=340+Math.floor(i/4)*16;

        html+='<ellipse class="nc-microbe" cx="'+x
          +'" cy="'+y+'" rx="7" ry="4.5"'
          +' fill="#38BDF8" stroke="#0369A1"'
          +' stroke-width="1.5"/>';
      }

      for(var j=0;j<denitrifierCount;j++){
        var dx=514+(j%3)*19;
        var dy=375+Math.floor(j/3)*15;

        html+='<ellipse class="nc-microbe" cx="'+dx
          +'" cy="'+dy+'" rx="7" ry="4.5"'
          +' fill="#FCA5A5" stroke="#B91C1C"'
          +' stroke-width="1.5"/>';
      }

      return html;
    }

    /**
     * 绘制肥料袋和肥料进入土壤的过程。
     */
    function buildFertilizer(
      level,
      visible
    ){
      if(!visible || level<2){
        return '';
      }

      var opacity=.3+.7*level/100;
      var count=Math.floor(1+level/14);
      var html=''
        +'<path d="M572 187 L624 187 L635 242'
        +' Q598 260 561 242 Z"'
        +' fill="#FFFFFF" stroke="#F59E0B"'
        +' stroke-width="4" opacity="'+opacity+'"/>'
        +'<text x="598" y="217" text-anchor="middle"'
        +' font-size="14" font-weight="900" fill="#B45309">氮肥</text>'
        +'<text x="598" y="235" text-anchor="middle"'
        +' font-size="10" font-weight="800" fill="#92400E">N</text>';

      for(var i=0;i<count;i++){
        var x=574+(i%5)*12;
        var y=255+Math.floor(i/5)*13+(i%2)*4;

        html+='<circle class="nc-nitrogen" cx="'+x
          +'" cy="'+y+'" r="'+(3+i%2)
          +'" fill="#F59E0B" opacity=".86"/>';
      }

      return html;
    }

    function update(){
      var F=Number(fixation.value);
      var D=Number(decomposition.value);
      var O=Number(oxygen.value);
      var P=Number(demand.value);
      var N=Number(fertilizer.value);
      var T=Number(time.value);

      fixationValue.textContent=F.toFixed(0)+'%';
      decompositionValue.textContent=D.toFixed(0)+'%';
      oxygenValue.textContent=O.toFixed(0)+'%';
      demandValue.textContent=P.toFixed(0)+'%';
      fertilizerValue.textContent=N.toFixed(0)+'%';
      timeValue.textContent=T.toFixed(0)+'%';

      for(var i=0;i<modeButtons.length;i++){
        modeButtons[i].classList.toggle(
          'active',
          modeButtons[i].getAttribute('data-mode')===mode
        );
      }

      labelToggle.textContent=showLabels
        ?'过程标注：显示'
        :'过程标注：隐藏';

      labelToggle.classList.toggle(
        'off',
        !showLabels
      );

      labelLayer.style.display=
        showLabels?'':'none';

      var flux=calculateFluxes(
        F,
        D,
        O,
        P,
        N,
        T
      );

      var ammoniumInput=
        flux.fixation
        +flux.ammonification
        +flux.fertilizer*.45;

      var ammoniumOutput=
        flux.nitrification
        +flux.assimilation*.36;

      var nitrateInput=
        flux.nitrification
        +flux.fertilizer*.55;

      var nitrateOutput=
        flux.denitrification
        +flux.assimilation*.64
        +flux.leaching;

      var ammoniumStock=clamp(
        42
        +(ammoniumInput-ammoniumOutput)
        *.62,
        6,
        98
      );

      var nitrateStock=clamp(
        45
        +(nitrateInput-nitrateOutput)
        *.58,
        6,
        98
      );

      var plantStock=clamp(
        38
        +(flux.assimilation
          -flux.feeding
          -flux.organicReturn*.32)
        *.58,
        8,
        96
      );

      var animalStock=clamp(
        22
        +(flux.feeding
          -flux.organicReturn*.23)
        *.62,
        6,
        82
      );

      var organicStock=clamp(
        47
        +(flux.organicReturn
          -flux.ammonification*.44)
        *.68,
        8,
        96
      );

      var waterNitrogen=clamp(
        8+flux.leaching*1.22,
        5,
        96
      );

      var availableNitrogen=clamp(
        ammoniumStock*.42
        +nitrateStock*.58,
        0,
        100
      );

      var nitrogenState=
        availableNitrogen<28
          ?'氮素不足'
          :availableNitrogen>78
            ?'氮素过剩'
            :flux.leaching>28
              ?'流失风险'
              :flux.denitrification>35
                ?'气态损失'
                :'相对协调';

      availableNitrogenText.textContent=
        availableNitrogen.toFixed(0);

      systemStateText.textContent=
        nitrogenState;

      systemStateText.style.color=
        nitrogenState==='氮素过剩'
        ||nitrogenState==='流失风险'
        ||nitrogenState==='气态损失'
          ?'#DC2626'
          :nitrogenState==='氮素不足'
            ?'#B45309'
            :'#047857';

      var processValues=[
        flux.fixation,
        flux.ammonification,
        flux.nitrification,
        flux.denitrification,
        flux.assimilation,
        flux.fertilizer,
        flux.leaching
      ];

      var processNames=[
        '固氮作用',
        '氨化作用',
        '硝化作用',
        '反硝化',
        '植物同化',
        '肥料输入',
        '氮素流失'
      ];

      var mainIndex=0;

      for(var j=1;j<processValues.length;j++){
        if(processValues[j]>processValues[mainIndex]){
          mainIndex=j;
        }
      }

      mainProcessText.textContent=
        processNames[mainIndex];

      plantBar.setAttribute(
        'width',
        String(140*plantStock/100)
      );

      animalBar.setAttribute(
        'width',
        String(121*animalStock/100)
      );

      organicBar.setAttribute(
        'width',
        String(148*organicStock/100)
      );

      ammoniumBar.setAttribute(
        'width',
        String(97*ammoniumStock/100)
      );

      nitrateBar.setAttribute(
        'width',
        String(97*nitrateStock/100)
      );

      waterBar.setAttribute(
        'width',
        String(76*waterNitrogen/100)
      );

      nitrificationBar.setAttribute(
        'width',
        String(112*flux.nitrification/67)
      );

      denitrificationBar.setAttribute(
        'width',
        String(112*flux.denitrification/59)
      );

      assimilationBar.setAttribute(
        'width',
        String(112*flux.assimilation/78)
      );

      root.style.setProperty(
        '--nc-nitrogen-speed',
        clamp(
          2.5-Math.max.apply(
            null,
            processValues
          )/68,
          .5,
          2.5
        ).toFixed(2)+'s'
      );

      root.style.setProperty(
        '--nc-flow-speed',
        clamp(
          2.4-Math.max.apply(
            null,
            processValues
          )/72,
          .46,
          2.4
        ).toFixed(2)+'s'
      );

      rootNodules.innerHTML=
        buildRootNodules(F);

      decomposerMicrobes.innerHTML=
        buildDecomposerMicrobes(D);

      var showBiological=
        mode==='system';

      var showMicrobes=
        mode==='system'
        ||mode==='microbes';

      var showAgriculture=
        mode==='system'
        ||mode==='agriculture';

      var flowHTML='';
      var particleHTML='';

      if(showBiological){
        flowHTML+=flow(
          'M201 209 C236 205 269 211 292 218',
          '#F59E0B',
          'arrow-orange',
          3+flux.feeding/15,
          .78
        );

        flowHTML+=flow(
          'M166 289 C175 321 189 341 214 358',
          '#92400E',
          'arrow-brown',
          3+flux.organicReturn/16,
          .72
        );

        flowHTML+=flow(
          'M334 251 C334 314 305 342 276 358',
          '#92400E',
          'arrow-brown',
          3+flux.organicReturn/18,
          .68
        );

        flowHTML+=flow(
          'M432 317 C360 300 270 295 204 284',
          '#16A34A',
          'arrow-green',
          3.5+flux.assimilation/24,
          .84
        );

        flowHTML+=flow(
          'M433 387 C345 365 256 321 186 279',
          '#16A34A',
          'arrow-green',
          3+flux.assimilation/28,
          .72
        );

        particleHTML+=particles(
          202,209,292,218,
          Math.floor(1+flux.feeding/11),
          '#F59E0B',
          'N'
        );

        particleHTML+=particles(
          432,316,205,284,
          Math.floor(2+flux.assimilation/14),
          '#16A34A',
          'N'
        );
      }

      if(showMicrobes){
        flowHTML+=flow(
          'M290 106 C250 150 226 216 205 304',
          '#7C3AED',
          'arrow-purple',
          3.5+flux.fixation/22,
          .84
        );

        flowHTML+=flow(
          'M360 367 C390 349 411 330 432 316',
          '#7C3AED',
          'arrow-purple',
          3.5+flux.ammonification/22,
          .82
        );

        flowHTML+=flow(
          'M488 345 V359',
          '#0284C7',
          'arrow-blue',
          3.5+flux.nitrification/20,
          .86
        );

        flowHTML+=flow(
          'M516 359 C535 273 507 174 411 105',
          '#DC2626',
          'arrow-red',
          3.5+flux.denitrification/20,
          .84
        );

        particleHTML+=particles(
          289,109,210,302,
          Math.floor(1+flux.fixation/14),
          '#7C3AED',
          'N'
        );

        particleHTML+=particles(
          363,367,430,318,
          Math.floor(1+flux.ammonification/14),
          '#8B5CF6',
          'NH₄'
        );

        particleHTML+=particles(
          488,346,488,359,
          Math.floor(1+flux.nitrification/12),
          '#0284C7',
          'NO₃'
        );

        particleHTML+=particles(
          515,356,411,108,
          Math.floor(1+flux.denitrification/13),
          '#DC2626',
          'N₂'
        );
      }

      if(showAgriculture){
        flowHTML+=flow(
          'M597 244 C568 269 537 291 512 307',
          '#F59E0B',
          'arrow-orange',
          3+flux.fertilizer/20,
          .85
        );

        flowHTML+=flow(
          'M548 390 C594 383 628 360 650 339',
          '#0284C7',
          'arrow-blue',
          3+flux.leaching/17,
          .8
        );

        particleHTML+=particles(
          593,247,515,306,
          Math.floor(1+flux.fertilizer/12),
          '#F59E0B',
          'N'
        );

        particleHTML+=particles(
          550,391,648,341,
          Math.floor(1+flux.leaching/10),
          '#0284C7',
          'NO₃'
        );
      }

      flowLayer.innerHTML=flowHTML;
      particleLayer.innerHTML=particleHTML;

      microbeLayer.innerHTML=
        buildMicrobes(
          flux.nitrification,
          flux.denitrification,
          showMicrobes
        );

      fertilizerLayer.innerHTML=
        buildFertilizer(
          N,
          showAgriculture
        );

      var explanation='';
      var conditionNote='';

      if(mode==='microbes'){
        title.textContent=
          '微生物推动无机氮形态转化';

        summary.textContent=
          '固氮、氨化、硝化和反硝化把大气氮、有机氮、铵态氮和硝态氮连接起来';

        explanation=
          '固氮微生物把大气氮转化为生态系统可进一步利用的含氮物质；分解者通过氨化释放铵态氮；硝化细菌把铵态氮转化为硝态氮；反硝化细菌可使部分硝态氮重新形成氮气。';

        if(O<18){
          conditionNote=
            '当前土壤氧气很低，硝化作用受抑而反硝化作用增强，硝态氮更容易以气态形式损失。';
        }else if(O>82){
          conditionNote=
            '当前土壤氧气较充足，有利于硝化作用，但低氧条件下进行的反硝化相对较弱。';
        }else if(F<18){
          conditionNote=
            '固氮作用较弱，大气氮进入土壤氮库的相对通量较低。';
        }else if(D<18){
          conditionNote=
            '分解与氨化作用较弱，有机氮转化为铵态氮的速度受到限制。';
        }else{
          conditionNote=
            '当前固氮、氨化、硝化和反硝化共同维持土壤氮形态转化。';
        }
      }else if(mode==='agriculture'){
        title.textContent=
          '农业管理改变土壤氮收支';

        summary.textContent=
          '豆科共生可增强生物固氮，肥料可补充氮素，但过量输入会提高流失风险';

        explanation=
          '农业生态系统中的氮既来自土壤有机质和生物固氮，也可能来自含氮肥料。植物未及时吸收的无机氮可能发生淋失、径流或气态损失。';

        if(N>78){
          conditionNote=
            '当前肥料输入过高，土壤可利用氮增加，但硝态氮淋失、径流和水体富营养化风险也明显上升。';
        }else if(F>82 && N<20){
          conditionNote=
            '当前固氮作用较强且肥料输入较低，可用于演示豆科植物与固氮微生物共生补充氮素。';
        }else if(O<18){
          conditionNote=
            '水淹导致土壤缺氧，植物吸氮和硝化作用下降，反硝化气态损失增强。';
        }else if(P<24){
          conditionNote=
            '植物吸氮需求较低，新增无机氮难以及时进入生物量，剩余氮更容易积累或流失。';
        }else{
          conditionNote=
            '当前肥料输入、植物需求和土壤微生物转化处于中等水平。';
        }
      }else{
        title.textContent=
          '生态系统主要氮库与氮循环过程';

        summary.textContent=
          '氮在大气、土壤、生物群落和水体之间转移，微生物是多条转化路径的关键参与者';

        explanation=
          '多数生物不能直接利用大气氮。氮气需要经过固氮进入土壤氮库，植物再吸收铵态氮或硝态氮合成有机物，动物通过取食获得含氮物质。';

        if(availableNitrogen<28){
          conditionNote=
            '当前土壤可利用氮偏低，植物合成蛋白质、核酸等含氮物质可能受到限制。';
        }else if(availableNitrogen>78 && flux.leaching>22){
          conditionNote=
            '当前无机氮明显过剩，植物难以及时吸收，多余硝态氮具有较高流失风险。';
        }else if(flux.denitrification>35){
          conditionNote=
            '当前低氧条件使反硝化增强，较多硝态氮转化为气态氮并离开土壤。';
        }else if(flux.nitrification<16 && O<25){
          conditionNote=
            '当前氧气不足限制硝化作用，铵态氮向硝态氮的转化较弱。';
        }else{
          conditionNote=
            '当前固氮、氨化、硝化、同化和反硝化过程总体处于相对协调状态。';
        }
      }

      var timeNote=T<15
        ?'作用时间较短，各氮库的累计变化尚不明显。'
        :'作用时间增加会放大当前条件的累计结果，但不能自动修复氮素不足、过量施肥或土壤缺氧。';

      result.innerHTML=explanation
        +'<br>'+conditionNote
        +' '+timeNote
        +' 氮元素可以循环利用，但能量不能在营养级之间循环返回生产者。所有数值均为相对教学指标。';
    }

    for(var i=0;i<modeButtons.length;i++){
      modeButtons[i].onclick=function(){
        mode=this.getAttribute('data-mode');
        update();
      };
    }

    for(var j=0;j<scenarioButtons.length;j++){
      scenarioButtons[j].onclick=function(){
        var name=this.getAttribute('data-scenario');
        var data=scenarios[name];

        if(!data){
          return;
        }

        fixation.value=String(data.fixation);
        decomposition.value=String(data.decomposition);
        oxygen.value=String(data.oxygen);
        demand.value=String(data.demand);
        fertilizer.value=String(data.fertilizer);
        time.value=String(data.time);

        setScenarioActive(name);
        update();
      };
    }

    labelToggle.onclick=function(){
      showLabels=!showLabels;
      update();
    };

    fixation.oninput=function(){
      setScenarioActive('');
      update();
    };

    decomposition.oninput=function(){
      setScenarioActive('');
      update();
    };

    oxygen.oninput=function(){
      setScenarioActive('');
      update();
    };

    demand.oninput=function(){
      setScenarioActive('');
      update();
    };

    fertilizer.oninput=function(){
      setScenarioActive('');
      update();
    };

    time.oninput=function(){
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
