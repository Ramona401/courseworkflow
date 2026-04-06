/**
 * PublishButton — 发布至课程平台按钮组件
 *
 * 显示条件：Pipeline 状态为 verified（验收通过）
 * 操作：骨干教师手动确认已将课件粘贴发布至课程平台
 * 结果：状态变更为 published（单向不可逆）
 */
import { useState } from 'react'
import { publishPipeline } from '@/api/pipelines'

interface PublishButtonProps {
  pipelineId: string
  onPublished: () => void
}

export function PublishButton({ pipelineId, onPublished }: PublishButtonProps) {
  const [loading, setLoading]       = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  const handleConfirm = async () => {
    setLoading(true)
    try {
      await publishPipeline(pipelineId)
      setShowConfirm(false)
      onPublished()
    } catch (e: any) {
      alert('发布确认失败: ' + (e.message || '请重试'))
    }
    setLoading(false)
  }

  return (
    <>
      {/* 发布提示区块 */}
      <div style={{
        padding: '16px 20px', borderRadius: 14,
        background: 'linear-gradient(135deg, rgba(88,86,214,0.08), rgba(88,86,214,0.03))',
        border: '1px solid rgba(88,86,214,0.2)',
        display: 'flex', alignItems: 'center', gap: 16,
      }}>
        <span style={{ fontSize: 28, flexShrink: 0 }}>🚀</span>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 14, fontWeight: 700, color: '#5856d6', marginBottom: 4 }}>
            待发布至课程平台
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93', lineHeight: 1.6 }}>
            该课件已通过验收，请将课件代码复制并粘贴到课程平台后，点击下方按钮确认发布。
            <br />
            <span style={{ color: '#ff9500', fontWeight: 500 }}>⚠️ 此操作不可撤销，请确认已正确发布后再点击。</span>
          </div>
        </div>
        <button
          onClick={() => setShowConfirm(true)}
          style={{
            padding: '10px 24px', borderRadius: 12, border: 'none',
            background: 'linear-gradient(135deg, #5856d6, #7c3aed)',
            color: '#fff', fontSize: 14, fontWeight: 600,
            cursor: 'pointer', flexShrink: 0,
            boxShadow: '0 4px 12px rgba(88,86,214,0.35)',
            transition: 'all 0.2s ease',
          }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.transform = 'translateY(-1px)'; (e.currentTarget as HTMLElement).style.boxShadow = '0 6px 16px rgba(88,86,214,0.45)' }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'none'; (e.currentTarget as HTMLElement).style.boxShadow = '0 4px 12px rgba(88,86,214,0.35)' }}
        >
          ✓ 确认已发布至课程平台
        </button>
      </div>

      {/* 二次确认弹窗 */}
      {showConfirm && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.45)',
          backdropFilter: 'blur(6px)', zIndex: 9999,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}
          onClick={e => { if (e.target === e.currentTarget && !loading) setShowConfirm(false) }}
        >
          <div style={{
            background: '#fff', borderRadius: 20, padding: '32px 36px',
            maxWidth: 460, width: '90%',
            boxShadow: '0 24px 64px rgba(0,0,0,0.2)',
          }}>
            {/* 标题 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
              <span style={{ fontSize: 32 }}>🚀</span>
              <div>
                <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e' }}>确认发布至课程平台</div>
                <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>此操作不可撤销</div>
              </div>
            </div>

            {/* 说明 */}
            <div style={{
              padding: '14px 16px', borderRadius: 12,
              background: 'rgba(255,149,0,0.08)', border: '1px solid rgba(255,149,0,0.2)',
              marginBottom: 24,
            }}>
              <div style={{ fontSize: 13, color: '#cc6600', lineHeight: 1.7 }}>
                请确认以下事项后再点击「确认发布」：
                <br />
                ① 已将课件代码完整复制到课程平台
                <br />
                ② 已在课程平台验证课件显示正常
                <br />
                ③ 确认无误，准备正式上线
              </div>
            </div>

            {/* 按钮 */}
            <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end' }}>
              <button
                onClick={() => !loading && setShowConfirm(false)}
                disabled={loading}
                style={{
                  padding: '10px 24px', borderRadius: 12,
                  border: '1px solid rgba(0,0,0,0.1)', background: '#f5f5f7',
                  fontSize: 14, fontWeight: 500, cursor: loading ? 'not-allowed' : 'pointer',
                  color: '#3c3c43', opacity: loading ? 0.5 : 1,
                }}
              >
                取消
              </button>
              <button
                onClick={handleConfirm}
                disabled={loading}
                style={{
                  padding: '10px 28px', borderRadius: 12, border: 'none',
                  background: loading ? '#c7c7cc' : 'linear-gradient(135deg, #5856d6, #7c3aed)',
                  color: '#fff', fontSize: 14, fontWeight: 600,
                  cursor: loading ? 'not-allowed' : 'pointer',
                  boxShadow: loading ? 'none' : '0 4px 12px rgba(88,86,214,0.35)',
                }}
              >
                {loading ? '确认中...' : '确认发布 🚀'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
