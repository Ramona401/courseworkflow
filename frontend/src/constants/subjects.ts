/**
 * subjects.ts — 学科内置兜底清单（前端单一真相源的「默认值」）
 *
 * 背景：
 *   学科列表原散落在前端 8+ 处硬编码（workshopConstants / wizardConstants /
 *   listConstants / editModalStyles / myPlansConstants / TextbooksPage /
 *   ComponentEditModal / ClassProfilesPanel / UnitPlansPanel / MyAssistantsPage 等），
 *   各副本不一致，导致备课下拉缺「劳动/道德与法治/美术/音乐/体育」等学科。
 *
 * 治理方案（双保险）：
 *   - 权威数据源在数据库 subjects 表，后台可运营增删改，前端各下拉经
 *     useSubjects() 从 GET /api/v1/subjects 拉取。
 *   - 本文件是「内置兜底清单」：接口未返回前（首屏瞬间）、接口失败、或该功能
 *     出任何问题时，各下拉一律回退到本清单，保证下拉永远有内容、绝不空白。
 *   - 两者一致：本清单与数据库种子（v231 迁移）内容顺序完全对齐，正常情况下
 *     用户看到的就是数据库的实时数据，本清单只在异常/首屏兜底时可见。
 *
 * 维护：
 *   日常增删学科走「后台 → 学科管理」界面改数据库即可，本文件无需改动。
 *   仅当想调整「接口挂了时的兜底默认」才需同步本清单（一般不需要）。
 */

/**
 * 内置默认学科清单（与数据库 subjects 表种子顺序一致）。
 * 纯字符串数组、零依赖——任何组件都可安全 import，不会引起循环依赖。
 */
export const DEFAULT_SUBJECTS: string[] = [
  '语文',
  '数学',
  '英语',
  '道德与法治',
  '政治',
  '历史',
  '地理',
  '物理',
  '化学',
  '生物',
  '科学',
  '信息科技',
  '人工智能',
  '音乐',
  '美术',
  '体育',
  '劳动',
  '通用技术',
]

/** 筛选场景用：在学科前加「全部」选项（我的教案 / 课本 / 教案库等筛选下拉）。 */
export const withAllOption = (subjects: string[]): string[] => ['全部', ...subjects]

/** 「不限」场景用：在学科前加空串（AI 助手学科偏好等，空串=不限学科）。 */
export const withAnyOption = (subjects: string[]): string[] => ['', ...subjects]

/** 学科接口返回的单项结构（与后端 models.Subject 对齐，前端下拉只用到 name）。 */
export interface SubjectItem {
  id: string
  name: string
  code: string
  sort_order: number
  is_active: boolean
  is_system: boolean
  note?: string
  updated_by?: string | null
  created_at?: string
  updated_at?: string
}
