/**
 * MyTeachingResourcesPage.tsx — 我的备课资料（统一资料中心外壳）
 *
 * 设计（对齐 v2 文档「容器B：我的备课资料」）：
 *   这是一个 Tab 外壳，统一收纳老师备课时要带的各类「料」。
 *   - 助手装「人」（人设），在「我的 AI 助手」；
 *   - 这里装「料」（数据），按类型分 Tab。
 *   备课时老师「选一个助手 + 勾选要带的资料」，两个动作语义清晰。
 *
 * 当前 Tab：
 *   - 单元方案（已上线）：大单元逐步设计，学科负责人产出给全组/全局
 *   - 班级学情（已上线·批次1）：老师私有，每班一张学情卡，备课时挂载注入 AI 做分层教学
 *
 * 【超管收口配套·课程大纲入口收回】变更：
 *   原「课程大纲」Tab 已从老师侧撤除。课程大纲版本多、需保持全局一致，不宜由一线老师
 *   在此增删改；其管理入口统一收敛到后台「基础数据管理」(AdminPage 基础数据 Tab)，
 *   仅系统管理员/二线管理员维护。老师侧无需管理入口——备课时后端按「学科+年级+册次」
 *   自动把对应全局大纲注入 AI（对话式/专家式备课皆然），调用能力完全不受本次撤除影响。
 *   即：老师"用得到大纲"，但"不在这里管大纲"。
 *
 * 每个 Tab 内容组件以 embedded 模式渲染（隐藏自带大标题/外层 padding，
 * 标题与留白由本外壳统一提供，避免双标题双留白）。
 *
 * ⚠ Tab 状态记忆（批次2a 修复）：
 *   当前激活的 Tab 写入 URL query（?tab=xxx），而非纯组件内部 state。
 *   这样从学生档案子页「← 返回备课资料」回来时，可携带 ?tab=class-situation
 *   精确落回班级学情 Tab（此前外壳重新挂载会复位到第一个 Tab）；
 *   同时刷新页面、加书签、外部跳转也都能停在正确的 Tab。
 */
import { useSearchParams } from 'react-router-dom'
import UnitPlansPanel from '@/pages/lesson-plans/resources/unit-plans/UnitPlansPanel'
import ClassProfilesPanel from '@/pages/lesson-plans/resources/class-profiles/ClassProfilesPanel'

const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB',
  borderLight: '#F3F4F6',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  white: '#FFFFFF',
}

/** Tab 定义。future=true 的是占位（还没做），点了给「敬请期待」 */
interface ResourceTab {
  key: string
  label: string
  icon: string
  future?: boolean
}

// 课程大纲 Tab 已撤除（管理入口收敛到后台基础数据管理，见文件头说明）。
const TABS: ResourceTab[] = [
  { key: 'unit-plans',      label: '单元方案', icon: '🗂️' },
  { key: 'class-situation', label: '班级学情', icon: '🧑\u200d🎓' },
]

// 默认 Tab（URL 无 ?tab= 或值非法时回落到此）
// 原默认为 course-outlines，随大纲 Tab 撤除改为 unit-plans。
const DEFAULT_TAB = 'unit-plans'
const VALID_TAB_KEYS = TABS.map((t) => t.key)

export default function MyTeachingResourcesPage() {
  // Tab 状态以 URL query 为唯一真相源（?tab=xxx），便于返回/刷新/书签精确落回
  const [searchParams, setSearchParams] = useSearchParams()
  const rawTab = searchParams.get('tab') || ''
  // 非法值（外部乱传，含已撤除的 course-outlines 旧书签）回落默认 Tab，绝不渲染空白
  const active = VALID_TAB_KEYS.includes(rawTab) ? rawTab : DEFAULT_TAB

  // 切 Tab = 改写 URL query（replace 不往历史栈塞记录，连续点 Tab 不污染后退）
  const selectTab = (key: string) => {
    setSearchParams({ tab: key }, { replace: true })
  }

  return (
    <div style={{ padding: '24px 28px', maxWidth: 1100, margin: '0 auto' }}>
      {/* 统一大标题 */}
      <div style={{ marginBottom: 18 }}>
        <h1 style={{ fontSize: 22, fontWeight: 700, color: C.textPrimary, margin: 0 }}>📂 我的备课资料</h1>
        <div style={{ fontSize: 13, color: C.textSecondary, marginTop: 6 }}>
          备课时要带的各类资料都收在这里。备课时选一个 AI 助手，再勾选要参考的资料，AI 就能既懂“跟谁聊”，又懂“参考什么”。
        </div>
      </div>

      {/* Tab 栏 */}
      <div style={{
        display: 'flex', gap: 4, borderBottom: `1px solid ${C.border}`, marginBottom: 20,
      }}>
        {TABS.map((t) => {
          const isActive = active === t.key
          return (
            <button
              key={t.key}
              onClick={() => selectTab(t.key)}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                padding: '10px 18px', border: 'none', background: 'transparent',
                cursor: 'pointer', fontSize: 14,
                fontWeight: isActive ? 600 : 400,
                color: isActive ? C.primary : C.textSecondary,
                borderBottom: isActive ? `2px solid ${C.primary}` : '2px solid transparent',
                marginBottom: -1,
                transition: 'all 160ms ease',
              }}
            >
              <span>{t.icon}</span>
              <span>{t.label}</span>
              {t.future && (
                <span style={{
                  fontSize: 10, color: C.textMuted, background: C.borderLight,
                  padding: '1px 6px', borderRadius: 6, marginLeft: 2,
                }}>即将上线</span>
              )}
            </button>
          )
        })}
      </div>

      {/* Tab 内容 */}
      <div>
        {active === 'unit-plans' && <UnitPlansPanel />}
        {active === 'class-situation' && <ClassProfilesPanel />}
      </div>
    </div>
  )
}
