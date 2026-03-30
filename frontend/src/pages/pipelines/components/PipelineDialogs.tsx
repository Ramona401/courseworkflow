/**
 * PipelineDialogs.tsx — Pipeline列表页弹窗组件
 *   CreateDialog        — 单个Pipeline创建弹窗
 *   BatchCreateDialog   — 批量创建弹窗
 *   AssignDialog        — 批量分配审核员弹窗
 */
import { useState } from 'react'
import { Layers, UserPlus, X as XIcon, CheckCircle as CheckCircleIcon } from 'lucide-react'
import type { CreatePipelineRequest, OperatorInfo } from '@/api/pipelines'

// ==================== 单个创建弹窗 ====================

interface CreateDialogProps {
  onClose: () => void
  onCreate: (req: CreatePipelineRequest) => void
}

export function CreateDialog({ onClose, onCreate }: CreateDialogProps) {
  const [courseCode, setCourseCode]     = useState('')
  const [threshold, setThreshold]       = useState('9.0')
  const [evalRounds, setEvalRounds]     = useState('3')
  const [maxTRLoop, setMaxTRLoop]       = useState('3')
  const [maxMetaRetry, setMaxMetaRetry] = useState('3')
  const [submitting, setSubmitting]     = useState(false)

  const handleSubmit = () => {
    if (!courseCode.trim()) return
    setSubmitting(true)
    onCreate({
      course_code: courseCode.trim(),
      config: {
        threshold:      parseFloat(threshold)  || 9.0,
        eval_rounds:    parseInt(evalRounds)   || 3,
        max_tr_loop:    parseInt(maxTRLoop)    || 3,
        max_meta_retry: parseInt(maxMetaRetry) || 3,
      },
    })
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: 10,
    border: '1px solid rgba(0,0,0,0.1)', fontSize: 14, outline: 'none',
    boxSizing: 'border-box', background: '#fafafa', transition: 'border-color 0.15s ease',
  }
  const labelStyle: React.CSSProperties = {
    fontSize: 12, fontWeight: 600, color: '#3c3c43', marginBottom: 4, display: 'block',
  }

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: '#fff', borderRadius: 20, width: 440, maxWidth: '94vw', padding: 28, boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e', marginBottom: 20 }}>创建 Pipeline</div>

        {/* 课程编号 */}
        <div style={{ marginBottom: 16 }}>
          <label style={labelStyle}>课程编号 *</label>
          <input
            style={inputStyle} placeholder="如 G1-01" value={courseCode}
            onChange={e => setCourseCode(e.target.value)}
            onFocus={e => (e.target.style.borderColor = '#007aff')}
            onBlur={e => (e.target.style.borderColor = 'rgba(0,0,0,0.1)')}
          />
        </div>

        {/* 配置参数（2列）*/}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 20 }}>
          <div>
            <label style={labelStyle}>达标阈值</label>
            <input style={inputStyle} type="number" step="0.5" value={threshold} onChange={e => setThreshold(e.target.value)} />
          </div>
          <div>
            <label style={labelStyle}>评估轮数</label>
            <input style={inputStyle} type="number" min="1" max="10" value={evalRounds} onChange={e => setEvalRounds(e.target.value)} />
          </div>
          <div>
            <label style={labelStyle}>翻译循环上限</label>
            <input style={inputStyle} type="number" min="1" max="5" value={maxTRLoop} onChange={e => setMaxTRLoop(e.target.value)} />
          </div>
          <div>
            <label style={labelStyle}>Meta重试上限</label>
            <input style={inputStyle} type="number" min="1" max="5" value={maxMetaRetry} onChange={e => setMaxMetaRetry(e.target.value)} />
          </div>
        </div>

        {/* 操作按钮 */}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{ padding: '10px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer', color: '#3c3c43' }}>
            取消
          </button>
          <button
            onClick={handleSubmit} disabled={!courseCode.trim() || submitting}
            style={{ padding: '10px 24px', borderRadius: 10, border: 'none', background: courseCode.trim() && !submitting ? '#007aff' : '#c7c7cc', color: '#fff', fontSize: 14, fontWeight: 600, cursor: courseCode.trim() && !submitting ? 'pointer' : 'not-allowed' }}>
            {submitting ? '创建中...' : '创建'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ==================== 批量创建弹窗 ====================

interface BatchCreateDialogProps {
  onClose: () => void
  onBatchCreate: (codes: string[]) => void
}

export function BatchCreateDialog({ onClose, onBatchCreate }: BatchCreateDialogProps) {
  const [inputText, setInputText] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // 解析输入文本：逗号/换行/空格分隔
  const parseCodes = (): string[] =>
    inputText.split(/[,，\n\r\s]+/).map(s => s.trim()).filter(s => s.length > 0)

  const codes = parseCodes()

  const handleSubmit = () => {
    if (codes.length === 0) return
    setSubmitting(true)
    onBatchCreate(codes)
  }

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div style={{ background: '#fff', borderRadius: 20, width: 500, maxWidth: '94vw', padding: 28, boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 20 }}>
          <Layers size={20} style={{ color: '#007aff' }} />
          <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e' }}>批量创建 Pipeline</div>
        </div>

        {/* 输入区域 */}
        <div style={{ marginBottom: 12 }}>
          <label style={{ fontSize: 12, fontWeight: 600, color: '#3c3c43', marginBottom: 4, display: 'block' }}>
            输入课程编号（逗号、换行或空格分隔）
          </label>
          <textarea
            style={{ width: '100%', height: 140, padding: '12px 14px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.1)', fontSize: 14, outline: 'none', boxSizing: 'border-box', background: '#fafafa', fontFamily: 'monospace', resize: 'vertical', lineHeight: 1.6 }}
            placeholder={'G1-01, G1-02, G1-03\nG2-01\nG3-05'}
            value={inputText}
            onChange={e => setInputText(e.target.value)}
          />
        </div>

        {/* 识别结果提示 */}
        <div style={{ fontSize: 12, color: '#8e8e93', marginBottom: 16 }}>
          已识别 <span style={{ fontWeight: 600, color: codes.length > 0 ? '#007aff' : '#8e8e93' }}>{codes.length}</span> 个课程编号
          {codes.length > 0 && codes.length <= 20 && (
            <span style={{ marginLeft: 8, color: '#aeaeb2' }}>{codes.join(', ')}</span>
          )}
          {codes.length > 100 && (
            <span style={{ marginLeft: 8, color: '#ff3b30' }}>（上限100个）</span>
          )}
        </div>

        {/* 操作按钮 */}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{ padding: '10px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer', color: '#3c3c43' }}>
            取消
          </button>
          <button
            onClick={handleSubmit} disabled={codes.length === 0 || codes.length > 100 || submitting}
            style={{ padding: '10px 24px', borderRadius: 10, border: 'none', background: codes.length > 0 && codes.length <= 100 && !submitting ? '#007aff' : '#c7c7cc', color: '#fff', fontSize: 14, fontWeight: 600, cursor: codes.length > 0 && codes.length <= 100 && !submitting ? 'pointer' : 'not-allowed' }}>
            {submitting ? '创建中...' : `批量创建 (${codes.length})`}
          </button>
        </div>
      </div>
    </div>
  )
}

// ==================== 分配审核员弹窗 ====================

interface AssignDialogProps {
  selectedCount: number
  operators: OperatorInfo[]
  selectedOperator: string
  assigning: boolean
  onSelectOperator: (id: string) => void
  onConfirm: () => void
  onClose: () => void
}

export function AssignDialog({
  selectedCount, operators, selectedOperator, assigning,
  onSelectOperator, onConfirm, onClose,
}: AssignDialogProps) {
  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(4px)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget && !assigning) onClose() }}>
      <div style={{ background: '#fff', borderRadius: 20, width: 440, maxWidth: '94vw', padding: 28, boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }}>
        {/* 标题行 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 20 }}>
          <UserPlus size={20} color="#5856d6" />
          <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e', flex: 1 }}>分配审核员</div>
          <button
            onClick={() => !assigning && onClose()}
            style={{ background: '#f2f2f7', border: 'none', borderRadius: '50%', width: 30, height: 30, display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}>
            <XIcon size={16} color="#8e8e93" />
          </button>
        </div>

        <div style={{ fontSize: 13, color: '#86868b', marginBottom: 16 }}>
          将 <span style={{ fontWeight: 600, color: '#1c1c1e' }}>{selectedCount}</span> 个Pipeline分配给指定审核员
        </div>

        {/* 审核员列表 */}
        <div style={{ marginBottom: 20 }}>
          <label style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e', display: 'block', marginBottom: 8 }}>
            选择审核员
          </label>
          {operators.length === 0 ? (
            <div style={{ fontSize: 13, color: '#aeaeb2', padding: '12px 0' }}>暂无可分配的审核员</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {operators.map(op => (
                <div key={op.id} onClick={() => onSelectOperator(op.id)} style={{
                  display: 'flex', alignItems: 'center', gap: 12,
                  padding: '10px 14px', borderRadius: 10, cursor: 'pointer',
                  border: selectedOperator === op.id ? '2px solid #5856d6' : '1px solid rgba(0,0,0,0.08)',
                  background: selectedOperator === op.id ? 'rgba(88,86,214,0.05)' : '#fff',
                }}>
                  {/* 头像 */}
                  <div style={{ width: 32, height: 32, borderRadius: '50%', background: selectedOperator === op.id ? '#5856d6' : 'rgba(88,86,214,0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: selectedOperator === op.id ? '#fff' : '#5856d6', fontSize: 13, fontWeight: 600 }}>
                    {op.display_name.charAt(0)}
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>{op.display_name}</div>
                    <div style={{ fontSize: 11, color: '#86868b' }}>
                      {op.username} · {op.role === 'admin' ? '管理员' : op.role === 'senior_operator' ? '高级操作员' : '操作员'}
                    </div>
                  </div>
                  {selectedOperator === op.id && <CheckCircleIcon size={18} color="#5856d6" />}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* 操作按钮 */}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button onClick={() => !assigning && onClose()} style={{ padding: '10px 20px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)', background: '#fff', fontSize: 14, fontWeight: 500, cursor: 'pointer', color: '#3c3c43' }}>
            取消
          </button>
          <button
            onClick={onConfirm} disabled={!selectedOperator || assigning}
            style={{ padding: '10px 24px', borderRadius: 10, border: 'none', background: selectedOperator && !assigning ? '#5856d6' : '#c7c7cc', color: '#fff', fontSize: 14, fontWeight: 600, cursor: selectedOperator && !assigning ? 'pointer' : 'not-allowed' }}>
            {assigning ? '分配中...' : '确认分配'}
          </button>
        </div>
      </div>
    </div>
  )
}
