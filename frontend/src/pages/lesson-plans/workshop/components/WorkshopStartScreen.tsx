/**
 * WorkshopStartScreen.tsx — 备课工坊首屏(选择入口+新建表单)和恢复中加载态
 *
 * 从 WorkshopPage.tsx 拆分,包含:
 *   - ResumingView: 恢复教案进度时的加载动画
 *   - StartScreen: 首屏双入口(新建备课/导入已有)+风格前测引导+新建表单
 */

import { useNavigate } from 'react-router-dom'
import type { ConversationMessage } from '@/api/lesson-plans'
import { C } from './workshopConstants'
import { StartForm } from './WorkshopPanels'
import ImportPlanModal from './ImportPlanModal'

// ==================== 恢复中加载态 ====================

interface ResumingViewProps {
  resumeError: string | null
}

export function ResumingView({ resumeError }: ResumingViewProps) {
  const navigate = useNavigate()
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: '16px' }}>
      <div style={{ width: '36px', height: '36px', border: `3px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
      <div style={{ fontSize: '15px', color: C.textSec }}>正在恢复备课进度...</div>
      {resumeError && (
        <div style={{ fontSize: '14px', color: C.danger, marginTop: '8px' }}>
          {resumeError}
          <button onClick={() => navigate('/lesson-plans/my-plans')} style={{ marginLeft: '12px', color: C.primary, background: 'none', border: 'none', cursor: 'pointer', textDecoration: 'underline', fontSize: '14px' }}>
            返回我的教案
          </button>
        </div>
      )}
    </div>
  )
}

// ==================== 首屏 ====================

interface StartScreenProps {
  needsAssessment: boolean | null
  setNeedsAssessment: (v: boolean | null) => void
  startMode: 'choose' | 'new'
  setStartMode: (v: 'choose' | 'new') => void
  startLoading: boolean
  showImportModal: boolean
  setShowImportModal: (v: boolean) => void
  onStart: (subject: string, grade: string, topic: string, duration: number, recipeId?: string, textbookPageIds?: string[]) => void
  onImportSuccess: (planId: string, openingMessage: ConversationMessage) => void
}

export function StartScreen({
  needsAssessment, setNeedsAssessment,
  startMode, setStartMode,
  startLoading,
  showImportModal, setShowImportModal,
  onStart, onImportSuccess,
}: StartScreenProps) {
  const navigate = useNavigate()

  return (
    <div style={{ height: 'calc(100vh - 120px)', overflow: 'hidden', margin: '-28px -32px', display: 'flex', flexDirection: 'column' }}>
      <div style={{ flex: 1, overflowY: 'auto', padding: '0 32px' }}>

        {/* 风格前测引导(首次使用) */}
        {needsAssessment === true && (
          <div style={{ margin: '24px auto 20px', maxWidth: '680px', padding: '24px 28px', background: 'linear-gradient(135deg, rgba(79,123,232,0.06) 0%, rgba(16,185,129,0.06) 100%)', borderRadius: '16px', border: '1px solid rgba(79,123,232,0.15)', textAlign: 'center' }}>
            <div style={{ fontSize: '32px', marginBottom: '12px' }}>🎓</div>
            <h3 style={{ fontSize: '18px', fontWeight: 700, color: '#1F2937', margin: '0 0 8px' }}>欢迎!先来了解一下您的备课风格</h3>
            <p style={{ fontSize: '14px', color: '#6B7280', lineHeight: 1.6, margin: '0 0 20px' }}>只需2-3分钟的轻松对话,AI就能为您量身定制备课配方</p>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'center' }}>
              <button onClick={() => navigate('/lesson-plans/assessment')} style={{ padding: '10px 24px', borderRadius: '10px', border: 'none', background: '#4F7BE8', color: '#fff', fontSize: '15px', fontWeight: 600, cursor: 'pointer' }}>开始风格前测 →</button>
              <button onClick={() => setNeedsAssessment(false)} style={{ padding: '10px 20px', borderRadius: '10px', border: '1px solid #E5E7EB', background: 'transparent', fontSize: '14px', color: '#9CA3AF', cursor: 'pointer' }}>以后再说</button>
            </div>
          </div>
        )}

        {/* 双入口选择卡片 */}
        {startMode === 'choose' && (
          <div style={{ maxWidth: '680px', margin: '32px auto 0' }}>
            <div style={{ textAlign: 'center', marginBottom: '28px' }}>
              <h1 style={{ fontSize: '22px', fontWeight: 700, color: C.text, margin: '0 0 6px' }}>✨ 开始今天的备课</h1>
              <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>选择适合您的方式,AI全程陪伴</p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              {/* 左卡:新建备课 */}
              <div
                onClick={() => setStartMode('new')}
                style={{ padding: '32px 28px', borderRadius: '16px', border: `2px solid ${C.border}`, background: C.card, cursor: 'pointer', transition: 'all 200ms ease', boxShadow: '0 2px 12px rgba(0,0,0,0.04)', display: 'flex', flexDirection: 'column', gap: '12px' }}
                onMouseEnter={e => { const el = e.currentTarget as HTMLDivElement; el.style.borderColor = C.primary; el.style.boxShadow = `0 8px 28px rgba(79,123,232,0.16)` }}
                onMouseLeave={e => { const el = e.currentTarget as HTMLDivElement; el.style.borderColor = C.border; el.style.boxShadow = '0 2px 12px rgba(0,0,0,0.04)' }}
              >
                <div style={{ width: '52px', height: '52px', borderRadius: '14px', background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '24px' }}>✨</div>
                <div>
                  <div style={{ fontSize: '17px', fontWeight: 700, color: C.text, marginBottom: '6px' }}>新建备课</div>
                  <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.7 }}>
                    告诉AI要上什么课<br />
                    AI全程陪你从零设计教案<br />
                    可选配方、关联课本图片
                  </div>
                </div>
                <div style={{ marginTop: 'auto', display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', fontWeight: 600, color: C.primary }}>
                  开始备课 <span style={{ fontSize: '16px' }}>→</span>
                </div>
              </div>

              {/* 右卡:导入已有教案 */}
              <div
                onClick={() => setShowImportModal(true)}
                style={{ padding: '32px 28px', borderRadius: '16px', border: `2px solid ${C.border}`, background: C.card, cursor: 'pointer', transition: 'all 200ms ease', boxShadow: '0 2px 12px rgba(0,0,0,0.04)', display: 'flex', flexDirection: 'column', gap: '12px' }}
                onMouseEnter={e => { const el = e.currentTarget as HTMLDivElement; el.style.borderColor = '#10B981'; el.style.boxShadow = '0 8px 28px rgba(16,185,129,0.16)' }}
                onMouseLeave={e => { const el = e.currentTarget as HTMLDivElement; el.style.borderColor = C.border; el.style.boxShadow = '0 2px 12px rgba(0,0,0,0.04)' }}
              >
                <div style={{ width: '52px', height: '52px', borderRadius: '14px', background: 'linear-gradient(135deg, #10B981, #34D399)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '24px' }}>📂</div>
                <div>
                  <div style={{ fontSize: '17px', fontWeight: 700, color: C.text, marginBottom: '6px' }}>导入已有教案</div>
                  <div style={{ fontSize: '13px', color: C.textSec, lineHeight: 1.7 }}>
                    已有教案直接上传<br />
                    AI自动评审并给出改进建议<br />
                    支持粘贴文本 · Word · PDF
                  </div>
                </div>
                <div style={{ marginTop: 'auto', display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', fontWeight: 600, color: '#10B981' }}>
                  导入并AI评审 <span style={{ fontSize: '16px' }}>→</span>
                </div>
              </div>
            </div>

            {/* 快捷入口 */}
            <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '32px' }}>
              {[
                { icon: '📋', text: '我的教案', path: '/lesson-plans/my-plans' },
                { icon: '📦', text: '配方管理', path: '/lesson-plans/recipes' },
                { icon: '📚', text: '教案库',   path: '/lesson-plans/library' },
                { icon: '📷', text: '课本管理', path: '/lesson-plans/textbooks' },
              ].map(item => (
                <button key={item.path} onClick={() => navigate(item.path)}
                  style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: C.textSec, background: 'transparent', border: 'none', padding: '6px 12px', borderRadius: '8px', cursor: 'pointer', transition: 'all 150ms ease' }}
                  onMouseEnter={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = C.primaryLight; el.style.color = C.primary }}
                  onMouseLeave={e => { const el = e.currentTarget as HTMLButtonElement; el.style.background = 'transparent'; el.style.color = C.textSec }}>
                  <span>{item.icon}</span><span>{item.text}</span>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* 新建备课表单 */}
        {startMode === 'new' && (
          <div>
            <div style={{ maxWidth: '960px', margin: '20px auto 0' }}>
              <button
                onClick={() => setStartMode('choose')}
                style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: C.textSec, background: 'none', border: 'none', cursor: 'pointer', padding: 0, marginBottom: '8px' }}
              >
                ← 返回选择
              </button>
            </div>
            <StartForm onStart={onStart} loading={startLoading} />
          </div>
        )}

      </div>

      {/* 导入弹窗 */}
      {showImportModal && (
        <ImportPlanModal
          onSuccess={onImportSuccess}
          onCancel={() => setShowImportModal(false)}
        />
      )}
    </div>
  )
}
