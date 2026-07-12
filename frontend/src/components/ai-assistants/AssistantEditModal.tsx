/**
 * AssistantEditModal.tsx — AI 助手编辑/新建弹窗
 *
 * 三种模式(mode 决定标题/source/默认字段):
 *   create-personal 新建我的助手 / create-group 新建本校助手 / edit 编辑现有助手
 * edit 模式 mount 时 getAssistant(id) 拉完整 prompt;提交后 onSaved 回调透传 ID 给父组件。
 *
 * Prompt 编辑区双 Tab(v114):[📝 手动编辑] textarea + [💬 AI 帮我写] DesignerPanel;
 *   切 Tab 不清 state;DesignerPanel onApplyDraft 回写 fullPrompt 并自动切回 manual。
 *
 * share_policy(本次新增):仅【共享助手】(source=group/system)显示 SharePolicyPicker——
 *   edit 时 loadedSource 为 group/system 显示并回填;create-group 显示默认 use_only;
 *   create-personal 不显示(个人助手别人看不到,策略无意义)。能编辑者(属主/组长/校管/admin)
 *   均可改策略(Yuhan 拍板:组长既能维护内容,顺手改策略也无妨)。提交时共享助手带该字段。
 *
 * 重构:样式/数据常量外移至 editModalStyles.ts,三档选择器抽至 SharePolicyPicker.tsx,
 *   以控制本文件行数在 600 红线内。
 *
 * Props:open / mode / assistantId?(edit必填) / defaultScene? / defaultSubject? /
 *   defaultGrade? / onClose / onSaved?(id, source)
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import {
  getAssistant,
  createAssistant,
  updateAssistant,
  parseAssistantScenes,
  ASSISTANT_SCENE_LABELS,
  DEFAULT_SHARE_POLICY,
  type AssistantScene,
  type AssistantSource,
  type AssistantSharePolicy,
  type CreateAIAssistantRequest,
  type UpdateAIAssistantRequest,
} from '@/api/ai-assistants'
import AssistantDesignerPanel from './AssistantDesignerPanel'
import SharePolicyPicker from './SharePolicyPicker'
import {
  C, QUICK_EMOJIS, SUBJECTS, MAX_PROMPT_LEN, labelStyle, inputStyle,
} from './editModalStyles'

/** Tab 类型(v114 新增) */
type EditTab = 'manual' | 'designer'

/* ==================== Props 类型 ==================== */

export type AssistantEditMode = 'create-personal' | 'create-group' | 'edit'

export interface AssistantEditModalProps {
  open: boolean
  mode: AssistantEditMode
  /** edit 模式下必填:要编辑的助手 ID */
  assistantId?: string
  /** create 模式下的默认场景(从 Selector 当前 scene 透传) */
  defaultScene?: AssistantScene
  /** create 模式下的默认学科 */
  defaultSubject?: string
  /** create 模式下的默认年级 */
  defaultGrade?: string
  /** 关闭回调 */
  onClose: () => void
  /** 保存成功回调(可选):参数为新/更新的助手 ID 和来源,便于父组件刷新列表或切换选中 */
  onSaved?: (id: string, source: AssistantSource) => void
}

/* ==================== 主组件 ==================== */

export default function AssistantEditModal(props: AssistantEditModalProps) {
  const { open, mode, assistantId, defaultScene, defaultSubject, defaultGrade, onClose, onSaved } = props

  // ==================== 表单状态 ====================
  const [name, setName]               = useState('')
  const [avatar, setAvatar]           = useState('🤖')
  const [description, setDescription] = useState('')
  const [subject, setSubject]         = useState('')
  const [gradeRange, setGradeRange]   = useState('')
  const [scenes, setScenes]           = useState<AssistantScene[]>([])
  const [fullPrompt, setFullPrompt]   = useState('')
  // share_policy:仅共享助手有意义;create-personal 不展示也不提交
  const [sharePolicy, setSharePolicy] = useState<AssistantSharePolicy>(DEFAULT_SHARE_POLICY)
  // edit 模式下拉到的原始 source(用于 onSaved 回调传回 + 判断是否显示策略选择器)
  const [loadedSource, setLoadedSource] = useState<AssistantSource | null>(null)

  // ==================== UI 状态 ====================
  const [loading, setLoading]   = useState(false)  // 加载详情中
  const [saving, setSaving]     = useState(false)  // 提交保存中
  const [loadErr, setLoadErr]   = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<EditTab>('manual')  // v114:Tab 切换

  // ==================== ref ====================
  const promptRef = useRef<HTMLTextAreaElement>(null)

  // ==================== 重置表单(关闭或模式切换时调用) ====================
  const resetForm = useCallback(() => {
    setName('')
    setAvatar('🤖')
    setDescription('')
    setSubject(defaultSubject || '')
    setGradeRange(defaultGrade || '')
    setScenes(defaultScene ? [defaultScene] : [])
    setFullPrompt('')
    setSharePolicy(DEFAULT_SHARE_POLICY)
    setLoadedSource(null)
    setLoadErr(null)
    setActiveTab('manual')  // v114:每次打开默认手动 Tab
  }, [defaultScene, defaultSubject, defaultGrade])

  // ==================== open 时初始化 ====================
  // create 模式:重置为默认值
  // edit 模式:拉详情填表单
  useEffect(() => {
    if (!open) return

    if (mode === 'edit' && assistantId) {
      setLoading(true)
      setLoadErr(null)
      setActiveTab('manual')  // edit 打开默认在手动 Tab
      getAssistant(assistantId)
        .then(data => {
          setName(data.name || '')
          setAvatar(data.avatar_emoji || '🤖')
          setDescription(data.description || '')
          setSubject(data.subject || '')
          setGradeRange(data.grade_range || '')
          // scenes 详情接口返回的是 JSONB 字符串,需要 parse
          setScenes(parseAssistantScenes(data.scenes))
          setFullPrompt(data.full_prompt || '')
          // 回填 share_policy(后端保证有值,兜底 use_only 防脏)
          setSharePolicy(data.share_policy || DEFAULT_SHARE_POLICY)
          setLoadedSource(data.source as AssistantSource)
          setLoading(false)
        })
        .catch(e => {
          const msg = e instanceof Error ? e.message : '加载助手详情失败'
          setLoadErr(msg)
          setLoading(false)
        })
    } else {
      // create 模式
      resetForm()
    }
  // assistantId/mode 变化时重新初始化,resetForm 已通过依赖收集
  }, [open, mode, assistantId, resetForm])

  // ==================== ESC 关闭 ====================
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  // ==================== 场景 checkbox 切换 ====================
  const toggleScene = (scene: AssistantScene) => {
    setScenes(prev => {
      if (prev.includes(scene)) return prev.filter(s => s !== scene)
      return [...prev, scene]
    })
  }

  // ==================== DesignerPanel 回调:把 AI 草稿应用到 fullPrompt ====================
  // v114:用户点"✓ 应用到编辑"→ 写入 fullPrompt state + 自动切回 manual Tab
  const handleApplyDraft = useCallback((draft: string) => {
    setFullPrompt(draft)
    setActiveTab('manual')
  }, [])

  // ==================== 是否显示分享策略选择器 ====================
  // 共享助手才显示:edit 时 loadedSource 为 group/system;create-group 模式本就是共享助手。
  // create-personal 不显示(个人助手别人看不到,策略无意义)。
  const showPolicyPicker =
    mode === 'create-group' ||
    (mode === 'edit' && (loadedSource === 'group' || loadedSource === 'system'))

  // ==================== 表单校验 ====================
  /** 返回错误提示字符串,null 表示校验通过 */
  const validate = (): string | null => {
    if (!name.trim()) return '请填写助手名称'
    if (!fullPrompt.trim()) return '请填写系统提示词(Prompt)\n如在 AI Tab 生成了草稿,请先点"✓ 应用到编辑"把草稿写入表单'
    if (scenes.length === 0) return '请至少勾选一个适用场景'
    if (fullPrompt.length > MAX_PROMPT_LEN) {
      return `系统提示词过长(${fullPrompt.length} 字符),上限 ${MAX_PROMPT_LEN} 字符`
    }
    return null
  }

  // ==================== 提交 ====================
  const handleSubmit = async () => {
    const err = validate()
    if (err) { alert(err); return }
    if (saving) return

    setSaving(true)
    try {
      if (mode === 'edit' && assistantId) {
        // 编辑模式:全量更新
        const req: UpdateAIAssistantRequest = {
          name: name.trim(),
          avatar_emoji: avatar || '🤖',
          description: description.trim(),
          full_prompt: fullPrompt,
          subject: subject.trim(),
          grade_range: gradeRange.trim(),
          scenes: scenes,
          // 仅共享助手带 share_policy;个人助手不带(后端会忽略/兜底)
          ...(showPolicyPicker ? { share_policy: sharePolicy } : {}),
        }
        await updateAssistant(assistantId, req)
        onSaved?.(assistantId, loadedSource || 'personal')
        onClose()
      } else {
        // 新建模式
        const source: AssistantSource = mode === 'create-group' ? 'group' : 'personal'
        const req: CreateAIAssistantRequest = {
          name: name.trim(),
          avatar_emoji: avatar || '🤖',
          description: description.trim(),
          source,
          full_prompt: fullPrompt,
          subject: subject.trim(),
          grade_range: gradeRange.trim(),
          scenes: scenes,
          // create-group 是共享助手,带 share_policy;create-personal 不带
          ...(showPolicyPicker ? { share_policy: sharePolicy } : {}),
        }
        const created = await createAssistant(req)
        onSaved?.(created.id, created.source as AssistantSource)
        onClose()
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存失败,请重试'
      alert(msg)
    } finally {
      setSaving(false)
    }
  }

  // ==================== 未 open 不渲染 ====================
  if (!open) return null

  // ==================== 标题推断 ====================
  const modalTitle =
    mode === 'edit'            ? (name ? `✏️ 编辑 — ${name}` : '✏️ 编辑助手') :
    mode === 'create-group'    ? '🏫 新建本校助手' :
                                 '➕ 新建我的助手'

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: 'rgba(17,24,39,0.5)',
        zIndex: 10000,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '24px',
      }}
    >
      {/* 弹窗本体(阻止冒泡让点击内部不关闭) */}
      <div
        onClick={e => e.stopPropagation()}
        style={{
          // v114:AI Tab 左右分栏宽度需求,整体宽度从 720px 放到 960px
          width: '960px', maxWidth: '100%', maxHeight: '92vh',
          background: C.card, borderRadius: '12px',
          boxShadow: '0 24px 64px rgba(0,0,0,0.18)',
          display: 'flex', flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        {/* ========== 顶部标题栏 ========== */}
        <div style={{
          padding: '16px 20px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          background: 'linear-gradient(135deg,rgba(79,123,232,0.06),rgba(129,140,248,0.04))',
          flexShrink: 0,
        }}>
          <span style={{ fontSize: '15px', fontWeight: 700, color: C.text }}>
            {modalTitle}
          </span>
          <button
            onClick={onClose}
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              fontSize: '20px', color: C.textMuted, padding: '0 4px',
              lineHeight: 1,
            }}
          >×</button>
        </div>

        {/* ========== 主体表单区(可滚动) ========== */}
        <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px' }}>
          {/* 加载中 */}
          {loading && (
            <div style={{ padding: '40px 0', textAlign: 'center', color: C.textMuted }}>
              加载助手详情中...
            </div>
          )}

          {/* 加载失败 */}
          {loadErr && !loading && (
            <div style={{
              padding: '12px', borderRadius: '8px',
              background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.2)',
              color: C.danger, fontSize: '13px', marginBottom: '12px',
            }}>
              ⚠️ {loadErr}
            </div>
          )}

          {/* 表单(加载成功或 create 模式下才显示) */}
          {!loading && !loadErr && (
            <>
              {/* ---- 通用字段:名称 + emoji ---- */}
              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>
                  名称 <span style={{ color: C.danger }}>*</span>
                </label>
                <div style={{ display: 'flex', gap: '8px' }}>
                  <input
                    type="text"
                    value={avatar}
                    onChange={e => setAvatar(e.target.value)}
                    placeholder="🤖"
                    maxLength={4}
                    style={{ ...inputStyle, width: '50px', textAlign: 'center', fontSize: '18px' }}
                  />
                  <input
                    type="text"
                    value={name}
                    onChange={e => setName(e.target.value)}
                    placeholder="例如:小学AI严苛审核员"
                    maxLength={100}
                    style={{ ...inputStyle, flex: 1 }}
                  />
                </div>
                {/* 快捷 emoji */}
                <div style={{ marginTop: '6px', display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
                  {QUICK_EMOJIS.map(e => (
                    <button
                      key={e}
                      type="button"
                      onClick={() => setAvatar(e)}
                      style={{
                        width: '28px', height: '28px', borderRadius: '6px',
                        border: `1px solid ${avatar === e ? C.primary : C.border}`,
                        background: avatar === e ? C.primaryLight : '#fff',
                        cursor: 'pointer', fontSize: '14px',
                      }}
                    >{e}</button>
                  ))}
                </div>
              </div>

              {/* ---- 通用字段:描述 ---- */}
              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>描述 <span style={{ color: C.textMuted, fontWeight: 400 }}>(选填,用户选择时可见)</span></label>
                <input
                  type="text"
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                  placeholder="一句话说清这个助手的风格定位"
                  maxLength={500}
                  style={{ ...inputStyle, width: '100%' }}
                />
              </div>

              {/* ---- 通用字段:学科 + 年级 ---- */}
              <div style={{ display: 'flex', gap: '12px', marginBottom: '16px' }}>
                <div style={{ flex: 1 }}>
                  <label style={labelStyle}>适用学科 <span style={{ color: C.textMuted, fontWeight: 400 }}>(空=不限)</span></label>
                  <select
                    value={subject}
                    onChange={e => setSubject(e.target.value)}
                    style={{ ...inputStyle, width: '100%', cursor: 'pointer' }}
                  >
                    {SUBJECTS.map(s => (
                      <option key={s} value={s}>{s || '(不限)'}</option>
                    ))}
                  </select>
                </div>
                <div style={{ flex: 1 }}>
                  <label style={labelStyle}>适用年级 <span style={{ color: C.textMuted, fontWeight: 400 }}>(空=不限)</span></label>
                  <input
                    type="text"
                    value={gradeRange}
                    onChange={e => setGradeRange(e.target.value)}
                    placeholder="例:1-6 或 7-9 或 初一"
                    maxLength={20}
                    style={{ ...inputStyle, width: '100%' }}
                  />
                </div>
              </div>

              {/* ---- 通用字段:适用场景(多选) ---- */}
              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>
                  适用场景 <span style={{ color: C.danger }}>*</span>
                  <span style={{ color: C.textMuted, fontWeight: 400 }}> (至少选 1 个)</span>
                </label>
                <div style={{
                  display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)',
                  gap: '6px', marginTop: '4px',
                }}>
                  {(Object.entries(ASSISTANT_SCENE_LABELS) as [AssistantScene, string][]).map(([scene, label]) => {
                    const checked = scenes.includes(scene)
                    return (
                      <label
                        key={scene}
                        style={{
                          display: 'flex', alignItems: 'center', gap: '6px',
                          padding: '7px 10px', borderRadius: '6px',
                          border: `1px solid ${checked ? C.primary : C.border}`,
                          background: checked ? C.primaryLight : '#fff',
                          cursor: 'pointer', fontSize: '13px', color: C.text,
                        }}
                      >
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleScene(scene)}
                          style={{ cursor: 'pointer', accentColor: C.primary }}
                        />
                        {label}
                      </label>
                    )
                  })}
                </div>
              </div>

              {/* ---- share_policy 分享策略选择器(仅共享助手显示) ---- */}
              {showPolicyPicker && (
                <SharePolicyPicker
                  value={sharePolicy}
                  onChange={setSharePolicy}
                />
              )}

              {/* ---- v114:Prompt 编辑区(手动/AI 双 Tab) ---- */}
              <div style={{ marginBottom: '8px' }}>
                {/* Tab 标题行 + 字符数统计 */}
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '6px',
                  }}
                >
                  <label style={{ ...labelStyle, marginBottom: 0 }}>
                    系统提示词 Prompt <span style={{ color: C.danger }}>*</span>
                  </label>
                  <span
                    style={{
                      fontSize: '11px',
                      fontWeight: 400,
                      color: fullPrompt.length > MAX_PROMPT_LEN ? C.danger : C.textMuted,
                    }}
                  >
                    {fullPrompt.length.toLocaleString()} / {MAX_PROMPT_LEN.toLocaleString()} 字符
                  </span>
                </div>

                {/* Tab 切换按钮条 */}
                <div
                  style={{
                    display: 'flex',
                    gap: '4px',
                    marginBottom: '8px',
                    padding: '4px',
                    background: C.bg,
                    borderRadius: '8px',
                    border: `1px solid ${C.border}`,
                    width: 'fit-content',
                  }}
                >
                  <button
                    type="button"
                    onClick={() => setActiveTab('manual')}
                    style={{
                      padding: '6px 14px',
                      borderRadius: '6px',
                      border: 'none',
                      background: activeTab === 'manual' ? C.card : 'transparent',
                      color: activeTab === 'manual' ? C.primary : C.textSec,
                      fontSize: '12px',
                      fontWeight: activeTab === 'manual' ? 700 : 500,
                      cursor: 'pointer',
                      boxShadow: activeTab === 'manual' ? '0 1px 3px rgba(0,0,0,0.06)' : 'none',
                    }}
                  >
                    📝 手动编辑
                  </button>
                  <button
                    type="button"
                    onClick={() => setActiveTab('designer')}
                    style={{
                      padding: '6px 14px',
                      borderRadius: '6px',
                      border: 'none',
                      background: activeTab === 'designer' ? C.card : 'transparent',
                      color: activeTab === 'designer' ? C.primary : C.textSec,
                      fontSize: '12px',
                      fontWeight: activeTab === 'designer' ? 700 : 500,
                      cursor: 'pointer',
                      boxShadow: activeTab === 'designer' ? '0 1px 3px rgba(0,0,0,0.06)' : 'none',
                    }}
                  >
                    💬 AI 帮我写
                  </button>
                </div>

                {/* Tab 内容:手动编辑(原有 textarea) */}
                {activeTab === 'manual' && (
                  <>
                    <textarea
                      ref={promptRef}
                      value={fullPrompt}
                      onChange={e => setFullPrompt(e.target.value)}
                      placeholder="在此粘贴/编写完整的系统提示词...&#10;示例:&#10;你是一位严苛的小学AI课程审核员,以五位专家视角对教案进行分析。...&#10;&#10;💡 也可以切到「💬 AI 帮我写」让 AI 帮您起草"
                      rows={16}
                      style={{
                        width: '100%', padding: '10px 12px',
                        borderRadius: '8px',
                        border: `1px solid ${fullPrompt.length > MAX_PROMPT_LEN ? C.danger : C.border}`,
                        fontSize: '12px', lineHeight: 1.6,
                        fontFamily: 'Menlo, Monaco, Consolas, monospace',
                        color: C.text, outline: 'none', boxSizing: 'border-box',
                        resize: 'vertical', minHeight: '280px', maxHeight: '500px',
                        background: C.bg,
                      }}
                    />
                    <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px', lineHeight: 1.6 }}>
                      💡 这是 AI 的 system prompt,决定助手的角色定位、评审风格、输出格式。
                      支持长至 128KB(约 4 万中文字),可容纳 v3.0 这种完整方法论文档。
                    </div>
                  </>
                )}

                {/* Tab 内容:AI 帮我写(DesignerPanel) */}
                {activeTab === 'designer' && (
                  <AssistantDesignerPanel
                    subject={subject}
                    grade={gradeRange}
                    scenes={scenes}
                    initialDraft={fullPrompt}
                    onApplyDraft={handleApplyDraft}
                  />
                )}
              </div>
            </>
          )}
        </div>

        {/* ========== 底部操作栏 ========== */}
        <div style={{
          padding: '12px 20px', borderTop: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'flex-end', gap: '8px',
          background: C.bg, flexShrink: 0,
        }}>
          <button
            onClick={onClose}
            disabled={saving}
            style={{
              padding: '8px 16px', borderRadius: '7px',
              border: `1px solid ${C.borderMid}`, background: '#fff',
              color: C.textSec, fontSize: '13px', cursor: saving ? 'not-allowed' : 'pointer',
              opacity: saving ? 0.5 : 1,
            }}
          >取消</button>
          <button
            onClick={handleSubmit}
            disabled={saving || loading || !!loadErr}
            style={{
              padding: '8px 20px', borderRadius: '7px',
              border: 'none',
              background: saving || loading || loadErr ? C.borderMid : C.primary,
              color: saving || loading || loadErr ? C.textMuted : '#fff',
              fontSize: '13px', fontWeight: 600,
              cursor: saving || loading || loadErr ? 'not-allowed' : 'pointer',
            }}
          >
            {saving ? '保存中...' : '💾 保存'}
          </button>
        </div>
      </div>
    </div>
  )
}
