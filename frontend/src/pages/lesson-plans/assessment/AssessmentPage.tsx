/**
 * AssessmentPage — 教学风格前测页面
 *
 * 迭代3新增：
 *   - AI对话式前测（6个问题渐进采集）
 *   - 进度条实时显示
 *   - AI回复中检测 <assessment_result> 自动解析并提交
 *   - 跳过前测按钮（使用默认画像）
 *   - 提交后自动生成配方 → 跳转备课工坊
 *   - 重测入口（个人中心可重新前测）
 */
import { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  startAssessment, chatAssessment, submitAssessment, skipAssessment,
  type AssessmentMessage, type AssessmentStartResponse, type AssessmentChatResponse,
} from '@/api/assessment'

// ==================== 样式常量 ====================
const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981',
  warning: '#F59E0B',
  danger: '#EF4444',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  card: '#FFFFFF',
  bg: '#F9FAFB',
}

// 步骤标签映射
const STEP_LABELS: Record<string, string> = {
  q1: '教龄与学科',
  q2: '备课习惯',
  q3: 'AI协作偏好',
  q4: '教学设计思路',
  q5: '质量关注点',
  q6: '教案结构',
}

const STEP_ICONS: Record<string, string> = {
  q1: '👋', q2: '📝', q3: '🤖', q4: '💡', q5: '🎯', q6: '📋',
}

// ==================== 主组件 ====================
export default function AssessmentPage() {
  const navigate = useNavigate()

  // 状态
  const [phase, setPhase] = useState<'loading' | 'chatting' | 'submitting' | 'done'>('loading')
  const [messages, setMessages] = useState<AssessmentMessage[]>([])
  const [inputText, setInputText] = useState('')
  const [isThinking, setIsThinking] = useState(false)
  const [currentStep, setCurrentStep] = useState('q1')
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [skipLoading, setSkipLoading] = useState(false)
  const [resultRecipeId, setResultRecipeId] = useState<string | null>(null)

  const messagesEndRef = useRef<HTMLDivElement>(null)

  // 自动滚动到底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isThinking])

  // ==================== 初始化：开始前测 ====================
  const initAssessment = useCallback(async () => {
    try {
      const resp: AssessmentStartResponse = await startAssessment()
      setMessages([resp.opening_message])
      setCurrentStep(resp.current_step)
      setProgress(0)
      setPhase('chatting')
    } catch (err) {
      console.error('开始前测失败:', err)
      setError('初始化失败，请刷新重试')
      setPhase('chatting')
    }
  }, [])

  useEffect(() => { initAssessment() }, [initAssessment])

  // ==================== 解析 <assessment_result> ====================
  const parseAssessmentResult = (content: string) => {
    const match = content.match(/<assessment_result>\s*([\s\S]*?)\s*<\/assessment_result>/)
    if (!match) return null
    try {
      return JSON.parse(match[1])
    } catch {
      console.error('解析assessment_result失败')
      return null
    }
  }

  // ==================== 发送消息 ====================
  const handleSend = async () => {
    if (!inputText.trim() || isThinking || phase !== 'chatting') return

    const userMsg: AssessmentMessage = {
      role: 'user',
      content: inputText.trim(),
      timestamp: new Date().toISOString(),
      step_code: currentStep,
    }

    setMessages(prev => [...prev, userMsg])
    setInputText('')
    setIsThinking(true)
    setError(null)

    try {
      const resp: AssessmentChatResponse = await chatAssessment(
        userMsg.content,
        [...messages, userMsg].filter(m => m.role === 'user' || m.role === 'assistant'),
      )

      setMessages(prev => [...prev, resp.ai_message])
      setCurrentStep(resp.current_step)
      setProgress(resp.progress)

      // 检查是否完成：AI回复中包含 <assessment_result>
      if (resp.is_complete) {
        const parsed = parseAssessmentResult(resp.ai_message.content)
        if (parsed) {
          // 自动提交
          setPhase('submitting')
          try {
            const allMessages = [...messages, userMsg, resp.ai_message]
            const submitResp = await submitAssessment({
              experience_years: parsed.experience_years || 0,
              subject_primary: parsed.subject_primary || '',
              grade_primary: parsed.grade_primary || '',
              teaching_style: parsed.teaching_style || 'growing',
              ai_collaboration: parsed.ai_collaboration || 'collaborative',
              priorities: parsed.priorities || [],
              lesson_structure_desc: parsed.lesson_structure_desc || '',
              conversation_log: allMessages,
            })
            setResultRecipeId(submitResp.recipe_id || null)
            setProgress(100)
            setPhase('done')
          } catch (submitErr) {
            console.error('提交前测结果失败:', submitErr)
            setError('保存结果失败，请重试')
            setPhase('chatting')
          }
        }
      }
    } catch (err) {
      console.error('前测对话失败:', err)
      setError('AI暂时无法回复，请稍后重试')
    } finally {
      setIsThinking(false)
    }
  }

  // ==================== 跳过前测 ====================
  const handleSkip = async () => {
    if (skipLoading) return
    setSkipLoading(true)
    try {
      const resp = await skipAssessment()
      setResultRecipeId(resp.recipe_id || null)
      setPhase('done')
    } catch (err) {
      console.error('跳过前测失败:', err)
      setError('操作失败，请重试')
    } finally {
      setSkipLoading(false)
    }
  }

  // ==================== 完成后跳转 ====================
  const handleGoToWorkshop = () => {
    navigate('/lesson-plans', { replace: true })
  }

  // ==================== 加载中 ====================
  if (phase === 'loading') {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: '16px' }}>
        <div style={{ width: '36px', height: '36px', border: `3px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        <div style={{ fontSize: '15px', color: C.textSec }}>正在准备前测对话...</div>
      </div>
    )
  }

  // ==================== 完成页 ====================
  if (phase === 'done') {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: '20px', padding: '0 24px' }}>
        <div style={{ fontSize: '48px' }}>🎉</div>
        <h2 style={{ fontSize: '22px', fontWeight: 700, color: C.text, margin: 0 }}>
          风格画像已生成
        </h2>
        <p style={{ fontSize: '15px', color: C.textSec, textAlign: 'center', lineHeight: 1.6, maxWidth: '400px', margin: 0 }}>
          {resultRecipeId
            ? '我们已根据您的教学风格自动生成了一份个性化备课配方，您可以直接开始备课了！'
            : '您的教学风格画像已保存，可以开始备课了。'}
        </p>
        <div style={{ display: 'flex', gap: '12px', marginTop: '8px' }}>
          {resultRecipeId && (
            <button
              onClick={() => navigate(`/lesson-plans/recipes/${resultRecipeId}/edit`)}
              style={{
                padding: '12px 24px', borderRadius: '10px',
                border: `1px solid ${C.border}`, background: C.card,
                fontSize: '15px', color: C.text, cursor: 'pointer',
              }}
            >
              📦 查看我的配方
            </button>
          )}
          <button
            onClick={handleGoToWorkshop}
            style={{
              padding: '12px 28px', borderRadius: '10px', border: 'none',
              background: C.primary, color: '#fff', fontSize: '15px',
              fontWeight: 600, cursor: 'pointer',
            }}
          >
            开始备课 →
          </button>
        </div>
      </div>
    )
  }

  // ==================== 对话页（chatting / submitting） ====================
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 120px)', margin: '-28px -32px' }}>

      {/* ---- 顶部进度条 ---- */}
      <div style={{ padding: '16px 24px', borderBottom: `1px solid ${C.border}`, background: C.card }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '10px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <span style={{ fontSize: '18px' }}>🎓</span>
            <span style={{ fontSize: '16px', fontWeight: 600, color: C.text }}>教学风格前测</span>
            <span style={{ fontSize: '13px', color: C.textMuted, marginLeft: '4px' }}>
              {STEP_ICONS[currentStep] || '📋'} {STEP_LABELS[currentStep] || currentStep}
            </span>
          </div>
          <button
            onClick={handleSkip}
            disabled={skipLoading || phase === 'submitting'}
            style={{
              padding: '6px 16px', borderRadius: '8px',
              border: `1px solid ${C.border}`, background: 'transparent',
              fontSize: '13px', color: C.textMuted,
              cursor: skipLoading ? 'not-allowed' : 'pointer',
            }}
          >
            {skipLoading ? '处理中...' : '跳过，使用默认配置'}
          </button>
        </div>
        {/* 进度条 */}
        <div style={{ display: 'flex', gap: '4px' }}>
          {['q1','q2','q3','q4','q5','q6'].map((step, i) => {
            const stepNum = i + 1
            const isActive = step === currentStep
            const isDone = progress >= stepNum * 100 / 6
            return (
              <div key={step} style={{
                flex: 1, height: '4px', borderRadius: '2px',
                background: isDone ? C.success : isActive ? C.primary : '#E5E7EB',
                transition: 'background 300ms ease',
              }} />
            )
          })}
        </div>
      </div>

      {/* ---- 对话区 ---- */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
        {messages.map((msg, i) => (
          <div key={i} style={{
            display: 'flex',
            justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
          }}>
            <div style={{
              maxWidth: '75%', padding: '12px 16px', borderRadius: '14px',
              background: msg.role === 'user' ? C.primary : C.card,
              color: msg.role === 'user' ? '#fff' : C.text,
              border: msg.role === 'user' ? 'none' : `1px solid ${C.border}`,
              fontSize: '14px', lineHeight: 1.7,
              whiteSpace: 'pre-wrap', wordBreak: 'break-word',
            }}>
              {/* 隐藏 <assessment_result> 标签内容，只显示对话文本 */}
              {msg.content.replace(/<assessment_result>[\s\S]*?<\/assessment_result>/g, '').trim()}
            </div>
          </div>
        ))}

        {/* AI思考中 */}
        {isThinking && (
          <div style={{ display: 'flex', justifyContent: 'flex-start' }}>
            <div style={{
              padding: '12px 16px', borderRadius: '14px',
              background: C.card, border: `1px solid ${C.border}`,
              fontSize: '14px', color: C.textMuted,
            }}>
              <span style={{ animation: 'pulse 1.5s infinite' }}>正在思考...</span>
              <style>{`@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }`}</style>
            </div>
          </div>
        )}

        {/* 提交中 */}
        {phase === 'submitting' && (
          <div style={{ textAlign: 'center', padding: '20px', color: C.primary, fontSize: '14px' }}>
            🔄 正在保存您的风格画像并生成个性化配方...
          </div>
        )}

        {/* 错误提示 */}
        {error && (
          <div style={{
            textAlign: 'center', padding: '10px 16px',
            background: 'rgba(239,68,68,0.06)', borderRadius: '10px',
            color: C.danger, fontSize: '13px',
          }}>
            {error}
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* ---- 输入区 ---- */}
      <div style={{ padding: '16px 24px', borderTop: `1px solid ${C.border}`, background: C.card }}>
        <div style={{
          display: 'flex', gap: '10px', alignItems: 'flex-end',
          background: '#F9FAFB', borderRadius: '12px',
          border: `1px solid ${C.border}`, padding: '10px 12px',
        }}>
          <textarea
            value={inputText}
            onChange={e => setInputText(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSend()
              }
            }}
            placeholder="请回答上面的问题... (Enter发送)"
            rows={2}
            disabled={phase === 'submitting' || isThinking}
            style={{
              flex: 1, background: 'transparent', border: 'none', outline: 'none',
              fontSize: '15px', color: C.text, resize: 'none',
              fontFamily: 'inherit', lineHeight: 1.6,
              opacity: phase === 'submitting' ? 0.5 : 1,
            }}
          />
          <button
            onClick={handleSend}
            disabled={!inputText.trim() || isThinking || phase === 'submitting'}
            style={{
              width: '36px', height: '36px', flexShrink: 0, borderRadius: '50%',
              border: 'none',
              background: !inputText.trim() || isThinking || phase === 'submitting' ? '#E5E7EB' : C.primary,
              color: '#fff', cursor: 'pointer', fontSize: '16px',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >
            →
          </button>
        </div>
        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '8px', textAlign: 'center' }}>
          💡 这是一次轻松的对话，帮助AI了解您的备课习惯，大约需要2-3分钟
        </div>
      </div>
    </div>
  )
}
