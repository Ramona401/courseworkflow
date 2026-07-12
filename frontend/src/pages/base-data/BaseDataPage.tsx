/**
 * BaseDataPage — 基础数据管理（独立全屏页）
 *
 * 路由：/base-data（独立页面，不在任何 Layout 内，与 /admin「用户管理中心」同级）
 * 权限：admin（路由层 RoleGuard 保护）。admin 与二线管理员（admin2, is_super=false）
 *       role 均为 'admin'，二者都可进入并管理——基础数据是"业务基础字典"，
 *       不属于超管专属敏感入口，故【不】收 is_super。
 *
 * 定位（与"用户管理"分维度）：
 *   - 用户管理(/admin)       ：管"人与组织"（用户、区域、学校、教研组、角色、日志）
 *   - 基础数据管理(/base-data)：管"业务基础字典"（学科、课程大纲）
 *   两者是不同维度的后台管理，在门户首页作为并列入口卡片呈现，互不从属。
 *
 * 页面内两个并列子 Tab：
 *   - 📚 学科     → <SubjectsPanel />（全平台学科下拉的统一数据源）
 *   - 📖 课程大纲 → <CourseOutlinesPage embedded />（全局大纲，备课时按学科+年级+册次自动注入）
 *
 * 课程大纲说明：老师侧已无大纲管理入口（版本多、需全局一致，不宜一线老师增删改），
 *   管理统一收敛到此页；老师仍能"用到"大纲（备课时后端自动注入），只是"不在老师侧管"。
 *   CourseOutlinesPage 内部 admin 建大纲默认落"全局(system)"，符合"大纲全局注入"的定位。
 *
 * 页面外壳仿 /admin：顶部 sticky 顶栏（← 返回 + 居中标题），下方内容区。
 *   子 Tab 状态为组件内部 state（基础数据是低频后台维护，无需写入 URL 做返回记忆）。
 */
import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { C } from '@/pages/admin/components/adminConstants'
import { SubjectsPanel } from '@/pages/admin/components/SubjectsPanel'
// 课程大纲子 Tab：复用备课侧课程大纲页（embedded 模式，隐藏其自带大标题/外层 padding）。
// 该组件保留原文件不动，此处仅作为"基础数据管理"页的课程大纲面板复用。
import CourseOutlinesPage from '@/pages/lesson-plans/course-outlines/CourseOutlinesPage'

export default function BaseDataPage() {
  const navigate = useNavigate()
  const location = useLocation()
  // 返回目标：优先用进入时携带的 from（门户卡片会传 '/'），缺省回首页
  const fromPath: string = (location.state as { from?: string })?.from || '/'

  // 子 Tab：默认停在"学科"
  const [sub, setSub] = useState<'subjects' | 'outlines'>('subjects')
  const subTabs: { key: 'subjects' | 'outlines'; label: string }[] = [
    { key: 'subjects', label: '📚 学科' },
    { key: 'outlines', label: '📖 课程大纲' },
  ]

  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(135deg,#EEF2FF 0%,#FAFBFC 50%,#F0FDF4 100%)' }}>

      {/* ---- 顶部导航（仿 /admin）---- */}
      <header style={{ height: '64px', position: 'sticky', top: 0, zIndex: 100, background: 'rgba(255,255,255,0.88)', backdropFilter: 'blur(20px)', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 32px', gap: '16px' }}>
        <button onClick={() => navigate(fromPath)}
          style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.white, fontSize: '14px', color: C.textSec, cursor: 'pointer' }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.white }}>
          {'<- 返回'}
        </button>
        <div style={{ flex: 1, textAlign: 'center' }}>
          <h1 style={{ fontSize: '18px', fontWeight: 700, color: C.text, margin: 0 }}>📚 基础数据管理</h1>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
            管理全平台业务基础字典：学科、课程大纲
          </div>
        </div>
        {/* 右侧占位，保持标题居中（与 /admin 顶栏结构一致） */}
        <div style={{ width: '92px' }} />
      </header>

      <div style={{ maxWidth: '1400px', margin: '0 auto', padding: '24px' }}>

        {/* 子 Tab 栏（下划线式） */}
        <div style={{ display: 'flex', gap: '4px', borderBottom: `1px solid ${C.border}`, marginBottom: '20px' }}>
          {subTabs.map(t => {
            const active = sub === t.key
            return (
              <button key={t.key} onClick={() => setSub(t.key)}
                style={{
                  display: 'flex', alignItems: 'center', gap: '6px',
                  padding: '10px 18px', border: 'none', background: 'transparent',
                  cursor: 'pointer', fontSize: '14px',
                  fontWeight: active ? 600 : 400,
                  color: active ? C.primary : C.textSec,
                  borderBottom: active ? `2px solid ${C.primary}` : '2px solid transparent',
                  marginBottom: '-1px', transition: 'all 160ms ease',
                }}>
                {t.label}
              </button>
            )
          })}
        </div>

        {/* 子 Tab 内容 */}
        {sub === 'subjects' && <SubjectsPanel />}
        {sub === 'outlines' && (
          <div>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, display: 'flex', alignItems: 'center', gap: '8px' }}>
              📖 课程大纲
            </div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px', marginBottom: '16px', lineHeight: 1.6, maxWidth: '760px' }}>
              全局课程大纲的统一维护入口。一份大纲是一册书的完整课时地图；备课时系统按「学科 + 年级 + 册次」
              自动把对应大纲喂给 AI，让它备某一课时也知道整册全貌。此处由管理员维护、全局生效，
              一线老师在备课中自动调用，无需（也不可）在老师侧增删改，避免版本混乱。
            </div>
            <CourseOutlinesPage embedded />
          </div>
        )}

      </div>
    </div>
  )
}
