/**
 * TemplateRefineModal - AI 微调 + 历史回退 + 发布 弹窗 (v139 / v139.1)
 *
 * Tab 结构:
 *   1. 👁️ 预览     - 展示样例页 + 配色方案
 *   2. ✨ AI 微调  - SSE 流式微调,带进度提示
 *   3. ⏱️ 历史回退 - LIFO 快照列表,可回退任意版本
 *   4. 🚀 发布模板 - v139.1 改造:智能下拉选择 scope,不再让用户填 UUID
 *
 * v139.1 核心改造(PublishTab):
 *   - 加载时调 getPublishTargets() 获取当前用户可发布的所有目标
 *   - personal: 总是可选
 *   - school: 只在用户是某学校 admin 时显示(自动带入 school_id,无需用户输入)
 *   - group: 列出用户担任 lead/backbone 的所有组,用下拉选择
 *   - system: 只对 admin 角色显示
 *   - 不可用选项灰显,附带原因 tooltip
 *
 * v142 P2-1:
 *   - HistoryTab 回退失败由 alert 改为 Toast 轻提示,与 PublishTab 风格统一
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import {
  refineTemplate, subscribeTemplateRefineSSE, getTemplateHistory,
  rollbackTemplate, publishDraft, deleteDraft, getPublishTargets,
  CW_STYLE_CONFIG,
} from '@/api/coursewares'
import type {
  CoursewareTemplate, RefineHistoryEntry, PublishTargetsResponse,
} from '@/api/coursewares'
import { TemplateThumbAuto } from './TemplateThumb'

// ==================== 颜色常量 ====================
const C = {
  primary: '#F59E0B', textPrimary: '#1F2937', textSecondary: '#6B7280',
  textMuted: '#9CA3AF', border: '#E5E7EB', bgCard: '#FFFFFF', danger: '#EF4444',
  success: '#10B981', refine: '#7C3AED',
}

// ==================== Toast 轻提示组件 ====================
function Toast({ message, type = 'success', onClose }: {
  message: string
  type?: 'success' | 'error'
  onClose: () => void
}) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000)
    return () => clearTimeout(timer)
  }, [onClose])

  const bg = type === 'success'
    ? 'linear-gradient(135deg, #10B981, #059669)'
    : 'linear-gradient(135deg, #EF4444, #DC2626)'

  return (
    <div style={{
      position: 'fixed', top: '24px', left: '50%', transform: 'translateX(-50%)',
      padding: '12px 24px', borderRadius: '12px', background: bg,
      color: '#fff', fontSize: '14px', fontWeight: 600,
      boxShadow: '0 8px 24px rgba(0,0,0,0.15)', zIndex: 10001,
      animation: 'toastIn 0.3s ease-out',
      display: 'flex', alignItems: 'center', gap: '8px',
    }}>
      <span>{type === 'success' ? '✨' : '⚠️'}</span>
      <span>{message}</span>
      <style>{`@keyframes toastIn { from { opacity:0; transform:translateX(-50%) translateY(-20px); } to { opacity:1; transform:translateX(-50%) translateY(0); } }`}</style>
    </div>
  )
}

// ==================== 辅助函数 ====================
const safeParse = (s: string): Record<string, string> => {
  try { return JSON.parse(s) || {} } catch { return {} }
}
const safeParseArray = (s: string): string[] => {
  try { const a = JSON.parse(s); return Array.isArray(a) ? a : [] } catch { return [] }
}

interface Props {
  template: CoursewareTemplate
  onClose: () => void
  onPublished: () => void
  onDeleted: () => void
}

type Tab = 'preview' | 'refine' | 'history' | 'publish'

// ==================== 主组件 ====================
export default function TemplateRefineModal({ template, onClose, onPublished, onDeleted }: Props) {
  const [tab, setTab] = useState<Tab>('preview')
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)
  // 本地状态:用最新的 template 内容(微调/回退后会更新)
  const [current, setCurrent] = useState<CoursewareTemplate>(template)
  const colors = safeParse(current.color_scheme)
  const samplePages = safeParseArray(current.sample_pages)
  const [currentPageIdx, setCurrentPageIdx] = useState(0)
  const sc = CW_STYLE_CONFIG[current.style_category] || { label: current.style_category, color: '#6B7280', bg: '#F3F4F6', emoji: '🎨' }

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh',
      background: 'rgba(0,0,0,0.65)', zIndex: 9999,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      backdropFilter: 'blur(4px)',
    }} onClick={onClose}>
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
      <div style={{
        background: '#fff', borderRadius: '20px',
        width: '94%', maxWidth: '1200px', height: '92vh', overflow: 'hidden',
        display: 'flex', flexDirection: 'column',
      }} onClick={e => e.stopPropagation()}>
        {/* 头部 */}
        <div style={{
          padding: '18px 26px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '20px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
            <span style={{ fontSize: '24px' }}>{sc.emoji}</span>
            <div>
              <div style={{ fontSize: '18px', fontWeight: 700, color: C.textPrimary, display: 'flex', alignItems: 'center', gap: '8px' }}>
                {current.name}
                {current.is_draft && (
                  <span style={{ padding: '2px 10px', borderRadius: '10px', fontSize: '11px', fontWeight: 700, color: '#fff', background: C.refine }}>草稿</span>
                )}
              </div>
              {current.description && <div style={{ fontSize: '12px', color: C.textSecondary, marginTop: '2px' }}>{current.description}</div>}
            </div>
            <span style={{ padding: '2px 10px', borderRadius: '10px', fontSize: '11px', fontWeight: 600, color: sc.color, background: sc.bg }}>{sc.label}</span>
          </div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', fontSize: '26px', cursor: 'pointer', color: C.textMuted, lineHeight: 1 }}>×</button>
        </div>

        {/* Tab 切换 */}
        <div style={{ padding: '0 26px', borderBottom: `1px solid ${C.border}`, display: 'flex', gap: '4px' }}>
          {([
            { key: 'preview', label: '👁️ 预览' },
            { key: 'refine', label: '✨ AI 微调' },
            { key: 'history', label: '⏱️ 历史回退' },
            { key: 'publish', label: '🚀 发布模板' },
          ] as { key: Tab; label: string }[]).map(t => (
            <button key={t.key} onClick={() => setTab(t.key)} style={{
              padding: '10px 18px', border: 'none', background: 'transparent',
              borderBottom: `2px solid ${tab === t.key ? C.primary : 'transparent'}`,
              color: tab === t.key ? C.primary : C.textSecondary,
              fontSize: '14px', fontWeight: tab === t.key ? 700 : 500, cursor: 'pointer',
              marginBottom: '-1px',
            }}>{t.label}</button>
          ))}
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex' }}>
          {tab === 'preview' && (
            <PreviewTab
              samplePages={samplePages}
              colors={colors}
              currentPageIdx={currentPageIdx}
              onSelectPage={setCurrentPageIdx}
            />
          )}
          {tab === 'refine' && (
            <RefineTab
              template={current}
              onRefineDone={(updated) => { setCurrent(updated); setCurrentPageIdx(0) }}
            />
          )}
          {tab === 'history' && (
            <HistoryTab
              templateId={current.id}
              onRolledBack={(updated) => { setCurrent(updated); setCurrentPageIdx(0); setTab('preview') }}
              showToast={setToast}
            />
          )}
          {tab === 'publish' && (
            <PublishTab
              template={current}
              onPublished={onPublished}
              onDeleted={onDeleted}
              showToast={setToast}
            />
          )}
        </div>
      </div>
    </div>
  )
}

/* ==================== Tab1: 预览 ==================== */
function PreviewTab({ samplePages, colors, currentPageIdx, onSelectPage }: {
  samplePages: string[]
  colors: Record<string, string>
  currentPageIdx: number
  onSelectPage: (i: number) => void
}) {
  return (
    <div style={{ flex: 1, padding: '20px 26px', overflow: 'auto', display: 'flex', flexDirection: 'column', gap: '14px' }}>
      {samplePages.length > 1 && (
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          {samplePages.map((_, i) => (
            <button key={i} onClick={() => onSelectPage(i)} style={{
              padding: '6px 14px', borderRadius: '14px',
              border: `1.5px solid ${currentPageIdx === i ? '#F59E0B' : '#E5E7EB'}`,
              background: currentPageIdx === i ? 'rgba(245,158,11,0.08)' : 'transparent',
              color: currentPageIdx === i ? '#F59E0B' : '#6B7280',
              fontSize: '12px', fontWeight: 600, cursor: 'pointer',
            }}>第 {i + 1} 页</button>
          ))}
        </div>
      )}
      <div style={{ flex: 1, minHeight: 0 }}>
        {samplePages[currentPageIdx]
          ? <TemplateThumbAuto sampleHTML={samplePages[currentPageIdx]} title={`第 ${currentPageIdx + 1} 页预览`} />
          : <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#9CA3AF' }}>暂无样例页</div>
        }
      </div>
      {Object.keys(colors).length > 0 && (
        <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', alignItems: 'center' }}>
          <span style={{ fontSize: '12px', color: '#6B7280', fontWeight: 600 }}>配色:</span>
          {Object.entries(colors).map(([k, v]) => (
            <div key={k} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <div style={{ width: '20px', height: '20px', borderRadius: '6px', background: v, border: '1px solid rgba(0,0,0,0.08)' }} />
              <span style={{ fontSize: '11px', color: '#6B7280' }}>{k}</span>
              <span style={{ fontSize: '11px', color: '#9CA3AF', fontFamily: 'monospace' }}>{v}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

/* ==================== Tab2: AI 微调 ==================== */
function RefineTab({ template, onRefineDone }: {
  template: CoursewareTemplate
  onRefineDone: (updated: CoursewareTemplate) => void
}) {
  const [instruction, setInstruction] = useState('')
  const [refining, setRefining] = useState(false)
  const [progressMsg, setProgressMsg] = useState('')
  const [chunkCount, setChunkCount] = useState(0)
  const [error, setError] = useState('')
  const sseRef = useRef<{ close: () => void } | null>(null)

  useEffect(() => {
    return () => { sseRef.current?.close() }
  }, [])

  const examples = [
    '把主色改成更柔和的天蓝色',
    '圆角再大一些,字体改成思源宋体',
    '降低背景饱和度,整体更素雅',
    '主标题加上文字阴影效果',
    '把所有按钮的圆角统一为 12px',
  ]

  const handleRefine = async () => {
    setError('')
    const inst = instruction.trim()
    if (!inst) {
      setError('请输入修改指令')
      return
    }
    setRefining(true)
    setProgressMsg('正在连接...')
    setChunkCount(0)

    sseRef.current = subscribeTemplateRefineSSE(template.id, {
      onStart: d => setProgressMsg(d.message),
      onChunk: d => { setChunkCount(d.chunk_no); setProgressMsg(d.message) },
      onProgress: d => setProgressMsg(d.message),
      onDone: d => {
        const updated: CoursewareTemplate = {
          ...template,
          color_scheme: JSON.stringify(d.color_scheme),
          css_variables: JSON.stringify(d.css_variables),
          sample_pages: JSON.stringify(d.sample_pages),
          style_category: d.style_category || template.style_category,
        }
        setRefining(false)
        setProgressMsg('')
        setInstruction('')
        onRefineDone(updated)
      },
      onError: d => {
        setError(d.message)
        setRefining(false)
        setProgressMsg('')
      },
    })

    try {
      await refineTemplate(template.id, inst)
    } catch (e) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
      setError(msg || '触发微调失败')
      setRefining(false)
      sseRef.current?.close()
    }
  }

  return (
    <div style={{ flex: 1, padding: '20px 26px', overflow: 'auto', display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <div>
        <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, display: 'block', marginBottom: '8px' }}>
          修改指令
        </label>
        <textarea
          value={instruction}
          onChange={e => setInstruction(e.target.value)}
          disabled={refining}
          placeholder="用自然语言描述你想要的修改,例如:把主色改成更柔和的天蓝色,圆角加大..."
          style={{
            width: '100%', minHeight: '90px', padding: '12px 14px',
            borderRadius: '10px', border: `1px solid ${C.border}`,
            fontSize: '14px', lineHeight: 1.5, outline: 'none', resize: 'vertical',
            background: refining ? '#F9FAFB' : '#fff',
          }}
        />
      </div>

      <div>
        <div style={{ fontSize: '12px', color: C.textMuted, marginBottom: '8px' }}>💡 试试这些常见指令:</div>
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
          {examples.map(ex => (
            <button key={ex} onClick={() => !refining && setInstruction(ex)} disabled={refining} style={{
              padding: '5px 12px', borderRadius: '14px', border: `1px solid ${C.border}`,
              background: refining ? '#F9FAFB' : '#FFF8F0', color: C.textSecondary,
              fontSize: '12px', cursor: refining ? 'not-allowed' : 'pointer',
            }}>{ex}</button>
          ))}
        </div>
      </div>

      {refining && (
        <div style={{
          padding: '14px 18px', borderRadius: '12px',
          background: 'linear-gradient(135deg, rgba(124,58,237,0.08), rgba(245,158,11,0.08))',
          border: `1px solid ${C.refine}`,
          display: 'flex', alignItems: 'center', gap: '14px',
        }}>
          <span style={{ display: 'inline-block', width: '20px', height: '20px', border: '2.5px solid rgba(124,58,237,0.3)', borderTopColor: C.refine, borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: '13px', fontWeight: 600, color: C.refine }}>{progressMsg}</div>
            {chunkCount > 0 && <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>已接收 {chunkCount} 个数据块</div>}
          </div>
        </div>
      )}

      {error && (
        <div style={{ padding: '10px 14px', borderRadius: '8px', background: '#FEE2E2', color: C.danger, fontSize: '13px' }}>
          ⚠️ {error}
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <button onClick={handleRefine} disabled={refining || !instruction.trim()} style={{
          padding: '10px 28px', borderRadius: '10px', border: 'none',
          background: refining || !instruction.trim() ? '#D1D5DB' : 'linear-gradient(135deg, #7C3AED, #F59E0B)',
          color: '#fff', fontSize: '14px', fontWeight: 700,
          cursor: refining || !instruction.trim() ? 'not-allowed' : 'pointer',
        }}>
          {refining ? 'AI 微调中...' : '✨ 开始微调'}
        </button>
      </div>

      <div style={{ fontSize: '12px', color: C.textMuted, lineHeight: 1.6, padding: '10px 14px', background: '#F9FAFB', borderRadius: '8px' }}>
        💡 微调会自动保存修改前的版本到历史快照(最多保留 20 条),你可以随时回退到任意历史版本。
      </div>
      <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
    </div>
  )
}

/* ==================== Tab3: 历史回退 (v142: alert→Toast) ==================== */
function HistoryTab({ templateId, onRolledBack, showToast }: {
  templateId: string
  onRolledBack: (updated: CoursewareTemplate) => void
  showToast: (t: { message: string; type: 'success' | 'error' }) => void
}) {
  const [history, setHistory] = useState<RefineHistoryEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [rollingIdx, setRollingIdx] = useState<number | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTemplateHistory(templateId)
      setHistory(resp.history || [])
    } catch { /* */ } finally { setLoading(false) }
  }, [templateId])

  useEffect(() => { load() }, [load])

  const handleRollback = async (idx: number) => {
    if (!window.confirm(`确定回退到 ${idx + 1} 步前的版本?当前内容会作为新的历史快照保留。`)) return
    setRollingIdx(idx)
    try {
      const resp = await rollbackTemplate(templateId, idx)
      const updated = {
        id: resp.template_id,
        color_scheme: resp.color_scheme,
        css_variables: resp.css_variables,
        sample_pages: resp.sample_pages,
        style_category: resp.style_category,
      } as CoursewareTemplate
      showToast({ message: '已成功回退到历史版本', type: 'success' })
      onRolledBack(updated)
    } catch (e) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
      showToast({ message: msg || '回退失败', type: 'error' })
    } finally {
      setRollingIdx(null)
    }
  }

  if (loading) {
    return <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C.textMuted }}>加载历史中...</div>
  }
  if (history.length === 0) {
    return (
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C.textMuted, gap: '10px' }}>
        <div style={{ fontSize: '48px' }}>📋</div>
        <div>暂无微调历史</div>
        <div style={{ fontSize: '12px' }}>每次 AI 微调成功后,修改前的版本会自动存入此处</div>
      </div>
    )
  }

  return (
    <div style={{ flex: 1, padding: '20px 26px', overflow: 'auto' }}>
      <div style={{ fontSize: '13px', color: C.textSecondary, marginBottom: '14px' }}>
        共 {history.length} 条历史快照(最近的在最前,最多保留 20 条)
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {history.map((h, i) => (
          <div key={i} style={{
            padding: '14px 18px', borderRadius: '12px', border: `1px solid ${C.border}`,
            background: '#FFF', display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '14px',
          }}>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '6px' }}>
                {i + 1} 步前 · {new Date(h.timestamp).toLocaleString('zh-CN')}
              </div>
              <div style={{ fontSize: '13px', color: C.textPrimary, fontWeight: 600, marginBottom: '4px' }}>
                🗨️ {h.user_instruction || '(无指令)'}
              </div>
              {h.change_summary && (
                <div style={{ fontSize: '12px', color: C.textSecondary, lineHeight: 1.5 }}>
                  💬 {h.change_summary}
                </div>
              )}
            </div>
            <button onClick={() => handleRollback(i)} disabled={rollingIdx !== null} style={{
              padding: '6px 14px', borderRadius: '8px', border: `1px solid ${C.refine}`,
              background: rollingIdx === i ? C.refine : 'transparent',
              color: rollingIdx === i ? '#fff' : C.refine,
              fontSize: '12px', fontWeight: 600,
              cursor: rollingIdx !== null ? 'not-allowed' : 'pointer',
              whiteSpace: 'nowrap',
            }}>{rollingIdx === i ? '回退中...' : '↶ 回退到这里'}</button>
          </div>
        ))}
      </div>
    </div>
  )
}

/* ==================== Tab4: 发布草稿为正式模板 (v139.1 改造) ==================== */
function PublishTab({ template, onPublished, onDeleted, showToast }: {
  template: CoursewareTemplate
  onPublished: () => void
  onDeleted: () => void
  showToast: (t: { message: string; type: 'success' | 'error' }) => void
}) {
  // 表单字段
  const [name, setName] = useState(template.name)
  const [desc, setDesc] = useState(template.description || '')
  const [category, setCategory] = useState(template.style_category)
  const [scope, setScope] = useState<'personal' | 'school' | 'group' | 'system'>('personal')
  const [selectedGroupID, setSelectedGroupID] = useState('') // group scope 时选中的教研组 ID

  // 发布目标(后端返回)
  const [targets, setTargets] = useState<PublishTargetsResponse | null>(null)
  const [loadingTargets, setLoadingTargets] = useState(true)

  // 提交状态
  const [publishing, setPublishing] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState('')

  // 加载发布目标
  useEffect(() => {
    setLoadingTargets(true)
    getPublishTargets()
      .then(setTargets)
      .catch(() => setTargets(null))
      .finally(() => setLoadingTargets(false))
  }, [])

  // 切换到 group scope 时,自动选中第一个组
  useEffect(() => {
    if (scope === 'group' && targets && targets.groups.length > 0) {
      setSelectedGroupID(targets.groups[0].id)
    }
  }, [scope, targets, selectedGroupID])

  const handlePublish = async () => {
    setError('')
    if (!name.trim()) {
      setError('请填写模板名称')
      return
    }

    // 计算 scope_target_id
    let scopeTargetID: string | undefined = undefined
    if (scope === 'school') {
      if (!targets?.school.available) {
        setError('您不是任何学校的管理员,无法发布学校模板')
        return
      }
      scopeTargetID = targets.school.school_id
    } else if (scope === 'group') {
      if (!selectedGroupID) {
        setError('请选择一个教研组')
        return
      }
      scopeTargetID = selectedGroupID
    }

    setPublishing(true)
    try {
      await publishDraft(template.id, {
        name: name.trim(),
        description: desc.trim(),
        style_category: category,
        scope,
        scope_target_id: scopeTargetID,
      })
      showToast({ message: `模板「${name}」已成功发布`, type: 'success' })
      onPublished()
    } catch (e) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
      setError(msg || '发布失败')
    } finally {
      setPublishing(false)
    }
  }

  const handleDeleteDraft = async () => {
    if (!window.confirm(`确定删除草稿「${template.name}」?此操作不可恢复。`)) return
    setDeleting(true)
    try {
      await deleteDraft(template.id)
      onDeleted()
    } catch (e) {
      const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
      showToast({ message: msg || '删除失败', type: 'error' })
      setDeleting(false)
    }
  }

  // 加载中显示
  if (loadingTargets) {
    return <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C.textMuted }}>加载发布选项中...</div>
  }

  // 构建 4 个 scope 选项的元数据(含是否可用 + 原因)
  const scopeOptions: {
    value: 'personal' | 'school' | 'group' | 'system'
    label: string
    emoji: string
    desc: string
    available: boolean
    reason?: string
  }[] = [
    {
      value: 'personal',
      label: '我的模板',
      emoji: '👤',
      desc: '保存到个人模板库,只有自己可见',
      available: targets?.personal.available ?? true,
    },
    {
      value: 'school',
      label: targets?.school.available ? `本校模板 - ${targets.school.name}` : '本校模板',
      emoji: '🏫',
      desc: targets?.school.available
        ? `将自动发布到「${targets.school.name}」,本校所有老师可用`
        : '需要您是某学校的管理员',
      available: targets?.school.available ?? false,
      reason: targets?.school.available ? undefined : '您不是任何学校的管理员',
    },
    {
      value: 'group',
      label: '本组模板',
      emoji: '👥',
      desc: targets && targets.groups.length > 0
        ? `您可发布到 ${targets.groups.length} 个教研组,本组成员可用`
        : '需要您在某教研组担任组长或骨干',
      available: targets ? targets.groups.length > 0 : false,
      reason: targets && targets.groups.length === 0 ? '您不是任何教研组的组长或骨干' : undefined,
    },
    {
      value: 'system',
      label: '系统模板',
      emoji: '⭐',
      desc: '发布到全平台,所有用户可见',
      available: targets?.system.available ?? false,
      reason: targets?.system.reason,
    },
  ]

  return (
    <div style={{ flex: 1, padding: '20px 26px', overflow: 'auto', display: 'flex', flexDirection: 'column', gap: '18px' }}>
      {/* 模板名称 */}
      <div>
        <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, display: 'block', marginBottom: '6px' }}>模板名称 *</label>
        <input value={name} onChange={e => setName(e.target.value)} disabled={publishing} style={{
          width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`,
          fontSize: '14px', outline: 'none', background: publishing ? '#F9FAFB' : '#fff',
        }} />
      </div>

      {/* 描述 */}
      <div>
        <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, display: 'block', marginBottom: '6px' }}>描述</label>
        <input value={desc} onChange={e => setDesc(e.target.value)} disabled={publishing} placeholder="简短描述模板特点(可选)" style={{
          width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`,
          fontSize: '14px', outline: 'none', background: publishing ? '#F9FAFB' : '#fff',
        }} />
      </div>

      {/* 风格类别 */}
      <div>
        <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, display: 'block', marginBottom: '8px' }}>风格类别</label>
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
          {Object.entries(CW_STYLE_CONFIG).map(([k, v]) => (
            <button key={k} onClick={() => setCategory(k)} disabled={publishing} style={{
              padding: '6px 14px', borderRadius: '14px', fontSize: '12px',
              border: `1.5px solid ${category === k ? C.primary : C.border}`,
              background: category === k ? 'rgba(245,158,11,0.08)' : 'transparent',
              color: category === k ? C.primary : C.textSecondary,
              fontWeight: category === k ? 600 : 400,
              cursor: publishing ? 'not-allowed' : 'pointer',
            }}>{v.emoji} {v.label}</button>
          ))}
        </div>
      </div>

      {/* 发布到(v139.1 改造:智能选项 + 不可用灰显) */}
      <div>
        <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, display: 'block', marginBottom: '8px' }}>发布到 *</label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {scopeOptions.map(o => {
            const isSelected = scope === o.value
            const canClick = o.available && !publishing
            return (
              <label key={o.value}
                title={o.reason || ''}
                style={{
                  display: 'flex', alignItems: 'flex-start', gap: '10px', padding: '12px 14px',
                  borderRadius: '10px',
                  border: `1.5px solid ${isSelected && o.available ? C.primary : C.border}`,
                  background: isSelected && o.available ? 'rgba(245,158,11,0.04)' : '#fff',
                  cursor: canClick ? 'pointer' : 'not-allowed',
                  opacity: o.available ? 1 : 0.5,
                }}
                onClick={() => { if (canClick) setScope(o.value) }}
              >
                <input type="radio" name="scope" checked={isSelected}
                  onChange={() => { if (canClick) setScope(o.value) }}
                  disabled={!canClick}
                  style={{ marginTop: '3px' }} />
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, display: 'flex', alignItems: 'center', gap: '8px' }}>
                    {o.emoji} {o.label}
                    {!o.available && (
                      <span style={{ padding: '1px 8px', borderRadius: '8px', fontSize: '10px', fontWeight: 600, color: '#9CA3AF', background: '#F3F4F6' }}>
                        不可用
                      </span>
                    )}
                  </div>
                  <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>{o.desc}</div>
                </div>
              </label>
            )
          })}
        </div>
      </div>

      {/* group scope 时显示教研组下拉选择 */}
      {scope === 'group' && targets && targets.groups.length > 0 && (
        <div>
          <label style={{ fontSize: '13px', fontWeight: 600, color: C.textPrimary, display: 'block', marginBottom: '6px' }}>
            选择教研组 *
          </label>
          <select
            value={selectedGroupID}
            onChange={e => setSelectedGroupID(e.target.value)}
            disabled={publishing}
            style={{
              width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`,
              fontSize: '14px', outline: 'none', background: publishing ? '#F9FAFB' : '#fff',
              cursor: publishing ? 'not-allowed' : 'pointer',
            }}
          >
            {targets.groups.map(g => (
              <option key={g.id} value={g.id}>
                {g.name} ({g.school_name}) · {g.role === 'lead' ? '组长' : '骨干'}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* 错误提示 */}
      {error && (
        <div style={{ padding: '10px 14px', borderRadius: '8px', background: '#FEE2E2', color: C.danger, fontSize: '13px' }}>
          ⚠️ {error}
        </div>
      )}

      {/* 底部按钮:删除草稿 + 立即发布 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: '10px', marginTop: '10px' }}>
        <button onClick={handleDeleteDraft} disabled={publishing || deleting} style={{
          padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.danger}`,
          background: 'transparent', color: C.danger, fontSize: '13px', fontWeight: 600,
          cursor: publishing || deleting ? 'not-allowed' : 'pointer',
        }}>{deleting ? '删除中...' : '🗑️ 删除草稿'}</button>

        <button onClick={handlePublish} disabled={publishing || !name.trim()} style={{
          padding: '10px 28px', borderRadius: '10px', border: 'none',
          background: publishing || !name.trim() ? '#D1D5DB' : 'linear-gradient(135deg, #10B981, #059669)',
          color: '#fff', fontSize: '14px', fontWeight: 700,
          cursor: publishing || !name.trim() ? 'not-allowed' : 'pointer',
        }}>{publishing ? '发布中...' : '🚀 立即发布'}</button>
      </div>
    </div>
  )
}
