/**
 * DeliveryModeSelect.tsx — Step4「交付模式」三档选择器
 *
 * 老师进入批量生成步骤后先选交付模式，三档：
 *   1. manual   纯手动     —— 只逐页生成 HTML，配图/视频后续在工作台手动做（走主页面老逻辑 generateCWPages）
 *   2. no_video HTML+配图  —— 全自动生成 HTML 并自动配图，不做视频占位（走 autoAssemble skip_video=true）
 *   3. full     全自动装配 —— HTML + 配图 + 视频首帧占位一次到位（走 autoAssemble skip_video=false）
 *
 * 后两档（自动配图）依赖风格锚点：后端 prepareAssembly 强制要求已设锚点，否则拒绝。
 *   故本选择器在【未设锚点】时把后两档卡片禁用 + 显示"需先设风格锚点"提示，把约束拦在点击前，
 *   避免老师选了才被后端 error 弹回，体验更顺。锚点在工作台「图片」Tab 设置。
 *
 * 本组件是纯选择：不发请求、不订阅 SSE。选定某档即回调 onSelect(mode)，由父组件决定后续渲染
 *   （manual→主页面批量生成；no_video/full→渲染 AutoAssemblyPanel）。
 */
import { C } from './workshopConstants'

export type DeliveryMode = 'manual' | 'no_video' | 'full'

interface Props {
  /** 是否已设风格锚点（未设时后两档禁用） */
  hasAnchor: boolean
  /** 剩余待生成页数（展示用，让老师对工作量有预期） */
  remainingCount: number
  /** 选定交付模式回调 */
  onSelect: (mode: DeliveryMode) => void
}

interface ModeCard {
  mode: DeliveryMode
  emoji: string
  title: string
  desc: string
  bullets: string[]
  accent: string       // 主色
  accentBg: string     // 浅底
  needsAnchor: boolean // 是否依赖风格锚点
}

const MODE_CARDS: ModeCard[] = [
  {
    mode: 'manual',
    emoji: '✋',
    title: '纯手动',
    desc: '只生成页面，配图和视频你来把控',
    bullets: ['逐页生成 HTML 排版', '配图 / 视频后续在工作台手动做', '最省积分，最灵活'],
    accent: '#059669',
    accentBg: '#ECFDF5',
    needsAnchor: false,
  },
  {
    mode: 'no_video',
    emoji: '🖼',
    title: 'HTML + 配图',
    desc: '自动生成并配图，不做视频',
    bullets: ['逐页生成 HTML + 自动配图', '配图套用风格锚点，全课件统一', '不生成视频占位，比全自动更快省'],
    accent: '#0891B2',
    accentBg: '#F0FDFF',
    needsAnchor: true,
  },
  {
    mode: 'full',
    emoji: '⚡',
    title: '全自动装配',
    desc: '一键交付，图文视频齐备',
    bullets: ['HTML + 配图 + 视频首帧占位', '含视频/动画的页自动备好首帧', '一步到位，耗时与积分最高'],
    accent: '#7C3AED',
    accentBg: '#FaF5FF',
    needsAnchor: true,
  },
]

export default function DeliveryModeSelect({ hasAnchor, remainingCount, onSelect }: Props) {
  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <div style={{ fontSize: 13, color: C.textSecondary, lineHeight: 1.7 }}>
          选择本次交付方式{remainingCount > 0 ? <>，剩余 <b style={{ color: C.textPrimary }}>{remainingCount}</b> 页待生成</> : ''}。
          自动配图会套用你设置的风格锚点，保持全课件视觉统一。
        </div>
      </div>

      {/* 未设锚点提示条（仅当未设时显示，解释为何后两档不可选） */}
      {!hasAnchor && (
        <div style={{
          display: 'flex', alignItems: 'flex-start', gap: 8, padding: '10px 14px', borderRadius: 10,
          marginBottom: 16, background: '#FFFBEB', border: '1px solid #FDE68A',
          fontSize: 12.5, color: '#92400E', lineHeight: 1.6,
        }}>
          <span style={{ fontSize: 15 }}>💡</span>
          <span>
            自动配图的两档需要先设置<b>风格锚点</b>（一张定调全课件配图风格的图）。
            请返回上一步「🎨 风格」，在页面底部的「⭐ 风格锚点」处上传或 AI 生成一张定调图并设为锚点，即可解锁自动配图。
          </span>
        </div>
      )}

      {/* 三档卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 14 }}>
        {MODE_CARDS.map(card => {
          const locked = false // 新方案：三档全部可点；选自动档后由画风弹窗当场设锚点，不再前置禁用
          return (
            <div
              key={card.mode}
              onClick={() => { if (!locked) onSelect(card.mode) }}
              style={{
                position: 'relative', borderRadius: 14, padding: '18px 16px',
                border: `2px solid ${locked ? C.border : card.accent + '55'}`,
                background: locked ? '#F9FAFB' : card.accentBg,
                cursor: locked ? 'not-allowed' : 'pointer',
                opacity: locked ? 0.6 : 1,
                transition: 'all 200ms',
                display: 'flex', flexDirection: 'column',
              }}
              onMouseEnter={e => { if (!locked) e.currentTarget.style.boxShadow = `0 6px 20px ${card.accent}22` }}
              onMouseLeave={e => { e.currentTarget.style.boxShadow = 'none' }}
            >
              {/* 锁标（未设锚点的档） */}
              {locked && (
                <span style={{
                  position: 'absolute', top: 12, right: 12, fontSize: 11, fontWeight: 600,
                  color: '#92400E', background: '#FEF3C7', padding: '2px 8px', borderRadius: 10,
                }}>🔒 需锚点</span>
              )}

              <div style={{ fontSize: 26, marginBottom: 8 }}>{card.emoji}</div>
              <div style={{ fontSize: 16, fontWeight: 700, color: locked ? C.textMuted : card.accent, marginBottom: 4 }}>
                {card.title}
              </div>
              <div style={{ fontSize: 12.5, color: C.textSecondary, lineHeight: 1.5, marginBottom: 12, minHeight: 34 }}>
                {card.desc}
              </div>

              <ul style={{ margin: 0, padding: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 6, flex: 1 }}>
                {card.bullets.map((b, i) => (
                  <li key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: 6, fontSize: 12, color: C.textSecondary, lineHeight: 1.5 }}>
                    <span style={{ color: locked ? C.textMuted : card.accent, fontWeight: 700, flexShrink: 0 }}>·</span>
                    <span>{b}</span>
                  </li>
                ))}
              </ul>

              {/* 选择态提示（可点时显示"点击选择"） */}
              <div style={{
                marginTop: 14, textAlign: 'center', fontSize: 13, fontWeight: 600,
                padding: '8px 0', borderRadius: 8,
                color: locked ? C.textMuted : '#fff',
                background: locked ? '#F3F4F6' : card.accent,
              }}>
                {locked ? '设锚点后解锁' : '选择此方式 →'}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
