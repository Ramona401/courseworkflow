/**
 * AssistantSwitcher.tsx — 对话式备课·助手轻量选择面板（Phase 1，PRD §3.3 / §3.4）
 *
 * 渐进式披露的浮层面板：由顶栏「当前助手」指示器点击唤起，列出可选助手 + 系统默认，
 * 点选即时生效并记住（无确认按钮），切换下一轮对话生效（不重写历史）。
 *
 * 受控组件：父组件（ConversationModePage）控制 open 与当前 pref，
 * 本组件负责 open 后拉取候选、处理选择、PUT 偏好、回调通知父组件。
 *
 * 三态高亮：
 *   - pref.has_record && pref.is_system_default → 高亮「系统默认」项
 *   - pref.has_record && assistant_id 命中某助手 → 高亮该助手
 *   - !pref.has_record（从没选过）→ 不强高亮任何项，仅在「系统默认」项标注「当前(自动)」
 *
 * 强默认 + 逃生口：第一项永远是「系统默认(跟随各阶段标准流程)」(= 写空串)，
 * 老师任何时候可一键退回纯骨架。
 */
import { useState, useEffect, useCallback } from 'react'
import { C } from '../components/workshopConstants'
import {
  getAssistantOptions, putAssistantPref,
  type AssistantOption, type AssistantPref,
} from '@/api/assistant-prefs'

export interface AssistantSwitcherProps {
  /** 是否展开面板（受控） */
  open: boolean
  /** 当前学科（必须精确匹配） */
  subject: string
  /** 当前具体年级（必须严格匹配） */
  grade: string
  /** 当前阶段代码（用于场景过滤） */
  stage?: string
  /** 当前偏好三态（父组件持有；null 表示尚未加载） */
  pref: AssistantPref | null
  /** 关闭面板回调 */
  onClose: () => void
  /**
   * 选择已写入成功后的回调，把最新三态交回父组件刷新指示器。
   * 父组件据此更新顶栏「当前助手」文案与本组件高亮。
   */
  onChanged: (next: AssistantPref) => void
}

/** 来源标签底色（个人/学校/系统 视觉区分，低饱和不喧宾夺主） */
function sourceBadgeStyle(source: string): { bg: string; color: string } {
  switch (source) {
    case 'personal':
      return { bg: 'rgba(59,130,246,0.10)', color: '#2563EB' } // 蓝=个人
    case 'group':
      return { bg: 'rgba(16,185,129,0.10)', color: '#059669' } // 绿=学校
    default:
      return { bg: 'rgba(107,114,128,0.10)', color: '#4B5563' } // 灰=系统
  }
}

export default function AssistantSwitcher({
  open, subject, grade, stage, pref, onClose, onChanged,
}: AssistantSwitcherProps) {
  const [options, setOptions] = useState<AssistantOption[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  /** 正在写入的目标ID（''=系统默认项写入中），用于行内 loading 与防重复点击 */
  const [savingId, setSavingId] = useState<string | null>(null)

  // 面板打开时拉取候选助手；关闭时不清空(避免下次打开闪烁)，但出错信息每次打开重置。
  useEffect(() => {
    if (!open) return
    let cancelled = false
    setLoading(true)
    setError('')
    getAssistantOptions(subject, grade, stage)
      .then((resp) => {
        if (cancelled) return
        setOptions(resp.assistants || [])
      })
      .catch(() => {
        if (cancelled) return
        setError('助手列表加载失败，请稍后重试')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, subject, grade, stage])

  // 选择某项（assistantId='' 表示系统默认）→ PUT 偏好 → 回调 → 关闭。
  const handlePick = useCallback(
    async (assistantId: string) => {
      if (savingId !== null) return // 写入中，忽略重复点击
      setSavingId(assistantId)
      setError('')
      try {
        const next = await putAssistantPref(subject, grade, stage, assistantId)
        onChanged(next)
        onClose()
      } catch {
        setError('保存失败，请重试')
      } finally {
        setSavingId(null)
      }
    },
    [subject, grade, stage, savingId, onChanged, onClose]
  )

  if (!open) return null

  // 判定某项是否为「当前生效」：用于打勾。
  const isSystemDefaultActive = !!pref?.has_record && !!pref?.is_system_default
  const activeAssistantId =
    pref?.has_record && !pref?.is_system_default ? pref?.assistant_id : ''
  // 从没选过时，系统默认项标注「当前(自动)」而非强打勾。
  const neverChosen = !pref?.has_record

  return (
    <>
      {/* 点击遮罩关闭（透明，仅作命中层） */}
      <div
        onClick={onClose}
        style={{ position: 'fixed', inset: 0, zIndex: 40 }}
      />
      {/* 浮层面板：锚定在触发点下方，由父容器 position:relative 定位 */}
      <div
        style={{
          position: 'absolute',
          top: '38px',
          left: 0,
          zIndex: 41,
          width: '320px',
          maxHeight: '420px',
          overflowY: 'auto',
          background: C.card,
          border: `1px solid ${C.border}`,
          borderRadius: '12px',
          boxShadow: '0 8px 28px rgba(0,0,0,0.14)',
          padding: '8px',
        }}
      >
        {/* 面板标题 */}
        <div style={{ padding: '6px 8px 8px', fontSize: '12px', color: C.textSec }}>
          为「{subject} · {grade}」选择备课助手
          <span style={{ display: 'block', marginTop: '2px', fontSize: '11px', color: C.textSec, opacity: 0.8 }}>
            切换下一轮起生效，已有对话不变
          </span>
        </div>

        {/* —— 第一项：系统默认（永远在最上，= 写空串） —— */}
        <button
          onClick={() => handlePick('')}
          disabled={savingId !== null}
          style={{
            width: '100%',
            textAlign: 'left',
            padding: '10px',
            marginBottom: '4px',
            borderRadius: '9px',
            border: `1px solid ${isSystemDefaultActive ? C.primary : C.border}`,
            background: isSystemDefaultActive ? 'rgba(99,102,241,0.06)' : 'transparent',
            cursor: savingId !== null ? 'default' : 'pointer',
            display: 'flex',
            alignItems: 'flex-start',
            gap: '8px',
          }}
        >
          <span style={{ fontSize: '16px', lineHeight: '20px' }}>⚙️</span>
          <span style={{ flex: 1, minWidth: 0 }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>系统默认</span>
              {isSystemDefaultActive && <span style={{ fontSize: '12px', color: C.primary }}>✓</span>}
              {neverChosen && (
                <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '6px', background: 'rgba(107,114,128,0.1)', color: '#6B7280' }}>
                  当前·自动
                </span>
              )}
            </span>
            <span style={{ display: 'block', marginTop: '2px', fontSize: '11px', color: C.textSec }}>
              跟随各阶段标准流程，不挂特定助手
            </span>
          </span>
          {savingId === '' && <span style={{ fontSize: '11px', color: C.textSec }}>…</span>}
        </button>

        {/* —— 分隔 —— */}
        <div style={{ height: '1px', background: C.border, margin: '6px 4px' }} />

        {/* —— 候选助手列表 —— */}
        {loading && (
          <div style={{ padding: '16px', textAlign: 'center', fontSize: '12px', color: C.textSec }}>
            加载助手中…
          </div>
        )}
        {error && !loading && (
          <div style={{ padding: '10px', fontSize: '12px', color: '#DC2626' }}>{error}</div>
        )}
        {!loading && !error && options.length === 0 && (
          <div style={{ padding: '16px', textAlign: 'center', fontSize: '12px', color: C.textSec }}>
            当前学科和具体年级暂无可选助手，将使用系统默认
          </div>
        )}
        {!loading &&
          options.map((a) => {
            const active = activeAssistantId === a.id
            const badge = sourceBadgeStyle(a.source)
            return (
              <button
                key={a.id}
                onClick={() => handlePick(a.id)}
                disabled={savingId !== null}
                style={{
                  width: '100%',
                  textAlign: 'left',
                  padding: '10px',
                  marginBottom: '4px',
                  borderRadius: '9px',
                  border: `1px solid ${active ? C.primary : C.border}`,
                  background: active ? 'rgba(99,102,241,0.06)' : 'transparent',
                  cursor: savingId !== null ? 'default' : 'pointer',
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: '8px',
                }}
              >
                <span style={{ flex: 1, minWidth: 0 }}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
                    <span style={{ fontSize: '13px', fontWeight: 600, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {a.name}
                    </span>
                    {active && <span style={{ fontSize: '12px', color: C.primary }}>✓</span>}
                    <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '6px', background: badge.bg, color: badge.color }}>
                      {a.source_label}
                    </span>
                    {a.grade_range && (
                      <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '6px', background: 'rgba(107,114,128,0.08)', color: '#6B7280' }}>
                        {a.grade_range}
                      </span>
                    )}
                    {a.is_default_here && !active && (
                      <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '6px', background: 'rgba(245,158,11,0.12)', color: '#B45309' }}>
                        推荐
                      </span>
                    )}
                  </span>
                  {a.description && (
                    <span style={{ display: 'block', marginTop: '3px', fontSize: '11px', color: C.textSec, lineHeight: '15px' }}>
                      {a.description}
                    </span>
                  )}
                </span>
                {savingId === a.id && <span style={{ fontSize: '11px', color: C.textSec }}>…</span>}
              </button>
            )
          })}
      </div>
    </>
  )
}
