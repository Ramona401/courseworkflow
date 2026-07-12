#!/usr/bin/env node

/**
 * TE-DNA前端体积与动态加载边界防回归检查。
 *
 * 使用方式：
 *   npm run check:bundle
 *
 * 执行流程：
 *   1. package.json先执行tsc -b；
 *   2. 本脚本调用Vite进行内存生产构建；
 *   3. 不写入、清空或覆盖frontend/dist；
 *   4. 从Rollup模块与chunk关系中检查体积及懒加载边界；
 *   5. 硬性检查失败时以非0状态退出。
 *
 * 当前保护的正式基线：
 *   CoursewareWorkshopPage
 *     - 只负责课件工坊主流程；
 *     - 动态加载SubjectToolsPanel；
 *     - 不包含任何大型学科模板实现。
 *
 *   SubjectToolsPanel
 *     - 只负责轻量工具宫格；
 *     - 动态加载11个工具弹窗；
 *     - 不包含地理、生命科学、物理和化学模板实现。
 */

import path from 'node:path'
import process from 'node:process'
import zlib from 'node:zlib'
import { build } from 'vite'

// ==================== 体积预算 ====================

const BUDGETS = {
  /**
   * 当前正式基线约411 kB / gzip 105 kB。
   * 预算保留合理增长空间，但不允许重新膨胀到MB级。
   */
  coursewareWorkshopRaw: 550_000,
  coursewareWorkshopGzip: 150_000,

  /**
   * 当前正式基线约13 kB / gzip 4.5 kB。
   * 目录组件只应包含工具卡片和动态导入声明。
   */
  subjectToolsRaw: 50_000,
  subjectToolsGzip: 20_000,

  /**
   * 以下为提示线，不阻断构建。
   * 地理和生命科学仅在点击具体实验室时加载。
   */
  geographyModalGzipAdvisory: 350_000,
  lifeScienceModalGzipAdvisory: 550_000,
}

// ==================== 正式弹窗基线 ====================

const MODAL_SPECS = [
  {
    name: 'StrokeOrderModal',
    suffix: '/StrokeOrderModal.tsx',
  },
  {
    name: 'FormulaEditorModal',
    suffix: '/FormulaEditorModal.tsx',
  },
  {
    name: 'MusicScoreModal',
    suffix: '/MusicScoreModal.tsx',
  },
  {
    name: 'MathGraphModal',
    suffix: '/MathGraphModal.tsx',
  },
  {
    name: 'GeographyLabModal',
    suffix: '/GeographyLabModal.tsx',
  },
  {
    name: 'MoleculeLabModal',
    suffix: '/MoleculeLabModal.tsx',
  },
  {
    name: 'LifeScienceLabModal',
    suffix: '/LifeScienceLabModal.tsx',
  },
  {
    name: 'ChemExperimentModal',
    suffix: '/ChemExperimentModal.tsx',
  },
  {
    name: 'PhysicsLabModal',
    suffix: '/PhysicsLabModal.tsx',
  },
  {
    name: 'PhysicsSceneModal',
    suffix: '/PhysicsSceneModal.tsx',
  },
  {
    name: 'ImmersiveLifeScienceModal',
    suffix: '/ImmersiveLifeScienceModal.tsx',
  },
]

// ==================== 数据采集 ====================

const projectRoot = process.cwd()
const chunks = []
const modules = []

function normalizePath(value) {
  return String(value || '').replaceAll('\\', '/')
}

function gzipBytes(code) {
  return zlib.gzipSync(code).length
}

function formatKB(bytes) {
  return `${(Number(bytes || 0) / 1000).toFixed(2)} kB`
}

function classifyModule(rawId) {
  const id = normalizePath(rawId)

  if (
    id.includes('/geographyLabTemplates') &&
    !id.endsWith('/geographyLabTemplates.ts')
  ) {
    return 'geography-template'
  }

  if (
    id.includes('/lifeScienceLabTemplates') &&
    !id.endsWith('/lifeScienceLabTemplates.ts')
  ) {
    return 'life-science-template'
  }

  if (
    id.includes('/physicsLabTemplates') &&
    !id.endsWith('/physicsLabTemplates.ts')
  ) {
    return 'physics-template'
  }

  if (
    id.includes('/chemExperimentTemplates') &&
    !id.endsWith('/chemExperimentTemplates.ts')
  ) {
    return 'chemistry-template'
  }

  return 'other'
}

const auditPlugin = {
  name: 'tedna-bundle-budget-audit',

  generateBundle(_options, bundle) {
    for (const [fileName, item] of Object.entries(bundle)) {
      if (item.type !== 'chunk') continue

      chunks.push({
        fileName,
        name: item.name,
        facadeModuleId: normalizePath(
          item.facadeModuleId,
        ),
        isEntry: item.isEntry,
        isDynamicEntry: item.isDynamicEntry,
        imports: [...item.imports],
        dynamicImports: [...item.dynamicImports],
        codeBytes: Buffer.byteLength(item.code),
        gzipBytes: gzipBytes(item.code),
        moduleCount: Object.keys(item.modules).length,
      })

      for (
        const [moduleId, meta]
        of Object.entries(item.modules)
      ) {
        modules.push({
          chunk: fileName,
          id: normalizePath(moduleId),
          category: classifyModule(moduleId),
          renderedLength: Number(
            meta.renderedLength || 0,
          ),
        })
      }
    }
  },
}

console.log(
  '========= TE-DNA前端体积防回归检查 =========',
)
console.log('')
console.log('正在执行Vite内存生产构建...')
console.log('本次不会写入或覆盖frontend/dist。')
console.log('')

await build({
  root: projectRoot,
  configFile: path.join(
    projectRoot,
    'vite.config.ts',
  ),
  logLevel: 'warn',
  build: {
    write: false,
    rollupOptions: {
      plugins: [auditPlugin],
    },
  },
})

// ==================== 查询工具 ====================

function findModuleBySuffix(suffix) {
  return modules.find(
    item => item.id.endsWith(suffix),
  )
}

function findChunk(fileName) {
  return chunks.find(
    item => item.fileName === fileName,
  )
}

function chunkForModule(suffix) {
  const moduleItem =
    findModuleBySuffix(suffix)

  if (!moduleItem) return undefined

  return findChunk(moduleItem.chunk)
}

function modulesByCategory(category) {
  return modules.filter(
    item => item.category === category,
  )
}

function countCategoryInChunk(
  category,
  chunk,
) {
  if (!chunk) return 0

  return modules.filter(
    item =>
      item.category === category &&
      item.chunk === chunk.fileName,
  ).length
}

function allCategoryInChunk(
  category,
  chunk,
) {
  const categoryModules =
    modulesByCategory(category)

  if (!chunk || categoryModules.length === 0) {
    return false
  }

  return categoryModules.every(
    item => item.chunk === chunk.fileName,
  )
}

// ==================== 关键资源 ====================

const mainChunk = chunkForModule(
  '/CoursewareWorkshopPage.tsx',
)

const subjectChunk = chunkForModule(
  '/SubjectToolsPanel.tsx',
)

const geographyChunk = chunkForModule(
  '/GeographyLabModal.tsx',
)

const lifeScienceChunk = chunkForModule(
  '/LifeScienceLabModal.tsx',
)

const physicsChunk = chunkForModule(
  '/PhysicsLabModal.tsx',
)

const chemistryChunk = chunkForModule(
  '/ChemExperimentModal.tsx',
)

const modalRows = MODAL_SPECS.map(spec => ({
  ...spec,
  chunk: chunkForModule(spec.suffix),
}))

// ==================== 验收工具 ====================

const checks = []
const advisories = []

function addCheck(
  name,
  passed,
  detail = '',
) {
  checks.push({
    name,
    passed,
    detail,
  })

  console.log(
    `${passed ? '✅' : '❌'} ${name}` +
    `${detail ? `：${detail}` : ''}`,
  )
}

function addAdvisory(
  name,
  passed,
  detail = '',
) {
  advisories.push({
    name,
    passed,
    detail,
  })

  console.log(
    `${passed ? '✅' : '⚠'} ${name}` +
    `${detail ? `：${detail}` : ''}`,
  )
}

// ==================== 核心体积 ====================

console.log('一、核心资源体积')
console.log('')

if (mainChunk) {
  console.log(
    `CoursewareWorkshopPage：${mainChunk.fileName}`,
  )
  console.log(
    `  未压缩：${formatKB(mainChunk.codeBytes)}`,
  )
  console.log(
    `  gzip：${formatKB(mainChunk.gzipBytes)}`,
  )
} else {
  console.log(
    '❌ 未找到CoursewareWorkshopPage资源',
  )
}

console.log('')

if (subjectChunk) {
  console.log(
    `SubjectToolsPanel：${subjectChunk.fileName}`,
  )
  console.log(
    `  未压缩：${formatKB(subjectChunk.codeBytes)}`,
  )
  console.log(
    `  gzip：${formatKB(subjectChunk.gzipBytes)}`,
  )
  console.log(
    `  动态依赖数：${subjectChunk.dynamicImports.length}`,
  )
} else {
  console.log(
    '❌ 未找到SubjectToolsPanel资源',
  )
}

console.log('')
console.log('二、硬性体积预算')
console.log('')

addCheck(
  '找到CoursewareWorkshopPage资源',
  Boolean(mainChunk),
)

addCheck(
  '找到SubjectToolsPanel资源',
  Boolean(subjectChunk),
)

addCheck(
  'CoursewareWorkshopPage未压缩体积不超过550 kB',
  Boolean(
    mainChunk &&
    mainChunk.codeBytes <=
      BUDGETS.coursewareWorkshopRaw
  ),
  mainChunk
    ? formatKB(mainChunk.codeBytes)
    : '未找到',
)

addCheck(
  'CoursewareWorkshopPage gzip体积不超过150 kB',
  Boolean(
    mainChunk &&
    mainChunk.gzipBytes <=
      BUDGETS.coursewareWorkshopGzip
  ),
  mainChunk
    ? formatKB(mainChunk.gzipBytes)
    : '未找到',
)

addCheck(
  'SubjectToolsPanel未压缩体积不超过50 kB',
  Boolean(
    subjectChunk &&
    subjectChunk.codeBytes <=
      BUDGETS.subjectToolsRaw
  ),
  subjectChunk
    ? formatKB(subjectChunk.codeBytes)
    : '未找到',
)

addCheck(
  'SubjectToolsPanel gzip体积不超过20 kB',
  Boolean(
    subjectChunk &&
    subjectChunk.gzipBytes <=
      BUDGETS.subjectToolsGzip
  ),
  subjectChunk
    ? formatKB(subjectChunk.gzipBytes)
    : '未找到',
)

// ==================== 动态加载边界 ====================

console.log('')
console.log('三、一级与二级懒加载边界')
console.log('')

addCheck(
  '主页面与学科工具目录位于不同资源',
  Boolean(
    mainChunk &&
    subjectChunk &&
    mainChunk.fileName !==
      subjectChunk.fileName
  ),
)

addCheck(
  'CoursewareWorkshopPage动态引用SubjectToolsPanel',
  Boolean(
    mainChunk &&
    subjectChunk &&
    mainChunk.dynamicImports.includes(
      subjectChunk.fileName,
    )
  ),
)

addCheck(
  'SubjectToolsPanel是动态入口',
  Boolean(
    subjectChunk?.isDynamicEntry
  ),
)

addCheck(
  '找到全部11个工具弹窗资源',
  modalRows.every(
    row => Boolean(row.chunk),
  ),
  `${modalRows.filter(row => row.chunk).length}/11`,
)

addCheck(
  '11个工具弹窗均为动态入口',
  modalRows.every(
    row => Boolean(
      row.chunk?.isDynamicEntry,
    ),
  ),
)

addCheck(
  '11个工具弹窗均已移出CoursewareWorkshopPage',
  modalRows.every(
    row =>
      Boolean(
        row.chunk &&
        mainChunk &&
        row.chunk.fileName !==
          mainChunk.fileName,
      ),
  ),
)

addCheck(
  '11个工具弹窗均已移出SubjectToolsPanel',
  modalRows.every(
    row =>
      Boolean(
        row.chunk &&
        subjectChunk &&
        row.chunk.fileName !==
          subjectChunk.fileName,
      ),
  ),
)

addCheck(
  'SubjectToolsPanel动态引用全部11个工具弹窗',
  Boolean(
    subjectChunk &&
    modalRows.every(
      row =>
        row.chunk &&
        subjectChunk.dynamicImports.includes(
          row.chunk.fileName,
        ),
    )
  ),
)

for (const row of modalRows) {
  console.log(
    (
      row.chunk
        ? '  ✅ '
        : '  ❌ '
    ) +
    row.name +
    (
      row.chunk
        ? ` → ${row.chunk.fileName}` +
          ` | ${formatKB(row.chunk.codeBytes)}` +
          ` | gzip ${formatKB(row.chunk.gzipBytes)}`
        : ' → 未找到'
    ),
  )
}

// ==================== 模板隔离 ====================

console.log('')
console.log('四、学科模板资源隔离')
console.log('')

const templateCategories = [
  'geography-template',
  'life-science-template',
  'physics-template',
  'chemistry-template',
]

for (const category of templateCategories) {
  addCheck(
    `CoursewareWorkshopPage不含${category}`,
    Boolean(
      mainChunk &&
      countCategoryInChunk(
        category,
        mainChunk,
      ) === 0
    ),
  )

  addCheck(
    `SubjectToolsPanel不含${category}`,
    Boolean(
      subjectChunk &&
      countCategoryInChunk(
        category,
        subjectChunk,
      ) === 0
    ),
  )
}

const geographyModules =
  modulesByCategory(
    'geography-template',
  )

const lifeScienceModules =
  modulesByCategory(
    'life-science-template',
  )

const physicsModules =
  modulesByCategory(
    'physics-template',
  )

const chemistryModules =
  modulesByCategory(
    'chemistry-template',
  )

addCheck(
  '地理模板模块数量不少于30',
  geographyModules.length >= 30,
  String(geographyModules.length),
)

addCheck(
  '全部地理模板位于GeographyLabModal资源',
  allCategoryInChunk(
    'geography-template',
    geographyChunk,
  ),
)

addCheck(
  '生命科学模板模块数量不少于68',
  lifeScienceModules.length >= 68,
  String(lifeScienceModules.length),
)

addCheck(
  '全部生命科学模板位于LifeScienceLabModal资源',
  allCategoryInChunk(
    'life-science-template',
    lifeScienceChunk,
  ),
)

addCheck(
  '物理模板模块数量不少于7',
  physicsModules.length >= 7,
  String(physicsModules.length),
)

addCheck(
  '全部物理模板位于PhysicsLabModal资源',
  allCategoryInChunk(
    'physics-template',
    physicsChunk,
  ),
)

addCheck(
  '化学模板模块数量不少于7',
  chemistryModules.length >= 7,
  String(chemistryModules.length),
)

addCheck(
  '全部化学模板位于ChemExperimentModal资源',
  allCategoryInChunk(
    'chemistry-template',
    chemistryChunk,
  ),
)

// ==================== 非阻断提示 ====================

console.log('')
console.log('五、单学科大型动态资源提示')
console.log('')

addAdvisory(
  'GeographyLabModal gzip建议不超过350 kB',
  Boolean(
    geographyChunk &&
    geographyChunk.gzipBytes <=
      BUDGETS.geographyModalGzipAdvisory
  ),
  geographyChunk
    ? formatKB(
        geographyChunk.gzipBytes,
      )
    : '未找到',
)

addAdvisory(
  'LifeScienceLabModal gzip建议不超过550 kB',
  Boolean(
    lifeScienceChunk &&
    lifeScienceChunk.gzipBytes <=
      BUDGETS.lifeScienceModalGzipAdvisory
  ),
  lifeScienceChunk
    ? formatKB(
        lifeScienceChunk.gzipBytes,
      )
    : '未找到',
)

// ==================== 最终结果 ====================

const failedChecks = checks.filter(
  item => !item.passed,
)

console.log('')
console.log('========= 检查摘要 =========')
console.log(
  `HARD_CHECK_TOTAL=${checks.length}`,
)
console.log(
  `HARD_CHECK_FAILED=${failedChecks.length}`,
)
console.log(
  `DYNAMIC_MODAL_COUNT=${modalRows.filter(row => row.chunk).length}`,
)
console.log(
  `GEOGRAPHY_TEMPLATE_MODULES=${geographyModules.length}`,
)
console.log(
  `LIFE_SCIENCE_TEMPLATE_MODULES=${lifeScienceModules.length}`,
)
console.log(
  `PHYSICS_TEMPLATE_MODULES=${physicsModules.length}`,
)
console.log(
  `CHEMISTRY_TEMPLATE_MODULES=${chemistryModules.length}`,
)
console.log(
  `BUNDLE_BUDGET_STATUS=${
    failedChecks.length === 0
      ? 'PASS'
      : 'FAIL'
  }`,
)

if (failedChecks.length > 0) {
  console.log('')
  console.log('❌ 体积或懒加载防回归检查失败：')

  for (const item of failedChecks) {
    console.log(
      `  - ${item.name}` +
      `${item.detail ? `：${item.detail}` : ''}`,
    )
  }

  process.exit(1)
}

console.log('')
console.log(
  '✅ 前端体积与懒加载防回归检查全部通过',
)
