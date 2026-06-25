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
 *   - 课程大纲（已上线）：全册课时地图，组长建给全组
 *   - 单元方案（已上线）：大单元逐步设计，学科负责人产出给全组/全局
 * 未来 Tab（加一个就是多一个 tab 项 + 多渲染一个组件，外壳不用动）：
 *   - 班级学情（老师私有）
 *
 * 每个 Tab 内容组件以 embedded 模式渲染（隐藏自带大标题/外层 padding，
 * 标题与留白由本外壳统一提供，避免双标题双留白）。
 */
import { useState } from 'react'
import CourseOutlinesPage from '@/pages/lesson-plans/course-outlines/CourseOutlinesPage'
import UnitPlansPanel from '@/pages/lesson-plans/resources/unit-plans/UnitPlansPanel'

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

const TABS: ResourceTab[] = [
  { key: 'course-outlines', label: '课程大纲', icon: '📖' },
  { key: 'unit-plans',      label: '单元方案', icon: '🗂️' },
  { key: 'class-situation', label: '班级学情', icon: '🧑\u200d🎓', future: true },
]

export default function MyTeachingResourcesPage() {
  const [active, setActive] = useState('course-outlines')

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
              onClick={() => setActive(t.key)}
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
        {active === 'course-outlines' && <CourseOutlinesPage embedded />}
        {active === 'unit-plans' && <UnitPlansPanel />}
        {active === 'class-situation' && <ComingSoon title="班级学情" desc="记录每个班的基础、特点、需要照顾的学生……备课时作为你私有的背景资料喂给 AI。" />}
      </div>
    </div>
  )
}

function ComingSoon({ title, desc }: { title: string; desc: string }) {
  return (
    <div style={{
      padding: 48, textAlign: 'center',
      background: C.white, borderRadius: 12, border: `1px dashed ${C.border}`,
    }}>
      <div style={{ fontSize: 32, marginBottom: 12 }}>🚧</div>
      <div style={{ fontSize: 16, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>{title} · 即将上线</div>
      <div style={{ fontSize: 13, color: C.textSecondary, maxWidth: 420, margin: '0 auto', lineHeight: 1.6 }}>{desc}</div>
    </div>
  )
}
