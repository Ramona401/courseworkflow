/**
 * 提示词保存二次确认弹窗（做法 A · 生产环境强确认）
 *
 * 交互按危险分档差异化：
 *   - high（高危课件类）：显示红色警示框，且必须手动键入确认词「确认」二字后
 *     「确认保存」按钮才可点击（防手滑误改导致课件批量生成崩溃）。
 *   - mid / kb：普通确认，展示对应文案，直接点「确认保存」即可。
 *
 * 由 PromptsPage 在点击「保存新版本」时弹出；确认后回调 onConfirm 执行真正的保存。
 * 纯受控组件，不含业务请求，便于复用与测试。
 */
import { useState } from 'react'
import { AlertTriangle, X } from 'lucide-react'
import type { PromptCategory } from '@/api/prompts'
import { getCategoryMeta, CONFIRM_KEYWORD } from './promptCategoryMeta'

interface PromptConfirmModalProps {
  category: PromptCategory   // 危险分档（决定文案与是否需键入）
  promptName: string         // 提示词中文名（弹窗中展示）
  promptKey: string          // 提示词标识（弹窗中展示）
  nextVersion: number        // 保存后的新版本号（提示用）
  saving: boolean            // 是否保存中（禁用按钮 + 文案）
  onConfirm: () => void      // 确认保存回调
  onCancel: () => void       // 取消回调
}

export default function PromptConfirmModal({
  category, promptName, promptKey, nextVersion, saving, onConfirm, onCancel,
}: PromptConfirmModalProps) {
  const meta = getCategoryMeta(category)
  const [typed, setTyped] = useState('')

  // high 档需键入确认词才放行；其余档直接可确认
  const canConfirm = meta.requireTyping ? typed.trim() === CONFIRM_KEYWORD : true

  return (
    <div
      onClick={onCancel}
      style={{
        position: 'fixed', inset: 0, zIndex: 10001,
        background: 'rgba(0,0,0,0.45)', backdropFilter: 'blur(2px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '20px',
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          width: 'min(480px, 94vw)', background: '#fff', borderRadius: '18px',
          boxShadow: '0 24px 64px rgba(0,0,0,0.28)', overflow: 'hidden',
        }}
      >
        {/* 顶部标题条（按档着色） */}
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '18px 22px', background: meta.bg, borderBottom: `1px solid ${meta.border}`,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            {meta.requireTyping
              ? <AlertTriangle size={20} color={meta.color} />
              : <span style={{ fontSize: '18px' }}>{meta.emoji}</span>}
            <span style={{ fontSize: '16px', fontWeight: 700, color: meta.color }}>
              {meta.confirmTitle}
            </span>
          </div>
          <button
            onClick={onCancel}
            style={{ border: 'none', background: 'transparent', cursor: 'pointer', color: meta.color, display: 'flex' }}
          >
            <X size={18} />
          </button>
        </div>

        {/* 正文 */}
        <div style={{ padding: '20px 22px' }}>
          {/* 目标提示词信息 */}
          <div style={{
            fontSize: '13px', color: '#1d1d1f', marginBottom: '14px',
            padding: '10px 14px', background: '#f7f7f8', borderRadius: '10px',
          }}>
            <div style={{ fontWeight: 600 }}>{promptName}</div>
            <div style={{ fontSize: '12px', color: '#8e8e93', marginTop: '2px' }}>
              {promptKey} · 保存后创建新版本 v{nextVersion}
            </div>
          </div>

          {/* 分档警示文案 */}
          <div style={{
            fontSize: '13px', lineHeight: '1.7', color: meta.requireTyping ? meta.color : '#4b4b4f',
            padding: meta.requireTyping ? '12px 14px' : '0',
            background: meta.requireTyping ? meta.bg : 'transparent',
            border: meta.requireTyping ? `1px solid ${meta.border}` : 'none',
            borderRadius: '10px', marginBottom: '16px',
          }}>
            {meta.confirmDesc}
          </div>

          {/* high 档：手动键入确认词 */}
          {meta.requireTyping && (
            <div style={{ marginBottom: '4px' }}>
              <label style={{ fontSize: '12px', color: '#8e8e93', display: 'block', marginBottom: '6px' }}>
                请输入「{CONFIRM_KEYWORD}」二字以启用保存：
              </label>
              <input
                value={typed}
                onChange={(e) => setTyped(e.target.value)}
                placeholder={CONFIRM_KEYWORD}
                autoFocus
                style={{
                  width: '100%', padding: '10px 14px', borderRadius: '10px',
                  border: `1px solid ${canConfirm ? meta.border : '#d1d1d6'}`,
                  fontSize: '14px', outline: 'none', boxSizing: 'border-box',
                  background: '#fff', color: '#1d1d1f',
                }}
              />
            </div>
          )}
        </div>

        {/* 底部按钮 */}
        <div style={{
          display: 'flex', justifyContent: 'flex-end', gap: '10px',
          padding: '0 22px 20px',
        }}>
          <button
            onClick={onCancel}
            disabled={saving}
            style={{
              padding: '10px 22px', borderRadius: '10px', border: '1px solid #d1d1d6',
              background: '#fff', color: '#1d1d1f', fontSize: '14px', fontWeight: 500,
              cursor: saving ? 'not-allowed' : 'pointer', opacity: saving ? 0.6 : 1,
            }}
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={!canConfirm || saving}
            style={{
              padding: '10px 22px', borderRadius: '10px', border: 'none',
              background: meta.requireTyping
                ? `linear-gradient(135deg, ${meta.color}, #a01818)`
                : 'linear-gradient(135deg, #007aff, #5856d6)',
              color: '#fff', fontSize: '14px', fontWeight: 600,
              cursor: (!canConfirm || saving) ? 'not-allowed' : 'pointer',
              opacity: (!canConfirm || saving) ? 0.5 : 1,
              boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
            }}
          >
            {saving ? '保存中...' : '确认保存'}
          </button>
        </div>
      </div>
    </div>
  )
}
