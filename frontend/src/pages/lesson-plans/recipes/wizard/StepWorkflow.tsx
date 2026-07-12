/**
 * StepWorkflow — 配方向导步骤4：备课流程配置
 *
 * v79 新增：分步向导式配方创建
 * 动作2（批次A.1）：移除「备课模式」选择器，promptMode 恒 guided。
 *
 * 【配方搭建一页化 · 批次2B-1】自定义阶段并入本步骤（从旧编辑器迁来）：
 *   - 新增 recipeId prop（主页面透传）：编辑态（有 id）可增删改自定义阶段；
 *     新建态（无 id）「＋自定义阶段」按钮置灰 + 提示「保存配方后可添加」（决策1 甲）。
 *   - 自定义阶段增删改即时调 API 落库（决策2 甲，复用编辑器 handleStageModalConfirm 口径）：
 *     编辑态加载时 getCustomStages 拉取；创建 createCustomStage、编辑 updateCustomStage、
 *     删除 deleteCustomStage；落库同时把阶段并入 formData.stageFlow（带 is_custom 标记，插在
 *     revise 之前），其位置/启停仍随向导「更新配方」最后提交（双轨：阶段定义即时落库，
 *     流程编排最后提交）。
 *   - 自定义阶段渲染：🔧 图标 + 「自定义」标签 + ✏️编辑/🗑️删除按钮；可启停、可排序
 *     （不受 STAGE_REMOVABLE 限制）；复用 CustomStageModal 弹窗。
 *
 * 【配方搭建一页化 · 批次3】顶部提示补区分文案：
 *   - 说明本步骤（备课流程=AI 陪你把教案做出来的工作阶段）与上一步
 *     （教案结构=成品教案长什么样的板块格式）的区别，二者老师易混。仅文案，逻辑零改动。
 *
 * BugFix（自定义阶段提示词无法保存）：
 *   openEditStageModal 原先把 system_prompt / prompt_variants / output_format 写死为空串，
 *   因 CustomStageResponse 当时只回 has_prompt，拿不到提示词原文；编辑弹窗显示空，
 *   老师若未重填就保存 → 空串覆盖数据库已存提示词，表现为"提示词无法保存"。
 *   后端已补全文字段返回，此处改用 cs 的真实值回填（缺字段兜底空/{}）。
 *
 * 设计目标：
 *   - 默认已是5阶段全开，大多数老师直接跳过即可
 *   - 高级用户可微调阶段顺序、启用状态、添加自定义阶段
 */
import { useState, useEffect, useCallback, useMemo } from 'react'
import type {
  StageFlowItem, FlowPreset, FlowValidationMessage,
  CustomStageResponse, CreateCustomStageRequest, UpdateCustomStageRequest,
} from '@/api/recipes'
import {
  getFlowPresets, validateFlow,
  getCustomStages, createCustomStage, updateCustomStage, deleteCustomStage,
} from '@/api/recipes'
import {
  STAGE_CODE_EMOJI, STAGE_CODE_NAME, STAGE_CODE_ROLE, STAGE_CODE_DESC,
  STAGE_REMOVABLE, FLOW_MSG_COLORS,
} from '../../workshop/components/workshopConstants'
import {
  C, stepCardStyle,
  type WizardFormData,
} from './wizardConstants'
import CustomStageModal from '../components/CustomStageModal'

/* ==================== Props 类型 ==================== */
interface StepWorkflowProps {
  formData: WizardFormData
  updateForm: (updates: Partial<WizardFormData>) => void
  recipeId?: string   // 批次2B-1：编辑态有值→可增删改自定义阶段；新建态 undefined→按钮置灰
}

/* ==================== 组件 ==================== */
export default function StepWorkflow({ formData, updateForm, recipeId }: StepWorkflowProps) {
  const isEdit = !!recipeId

  const [flowPresets, setFlowPresets] = useState<FlowPreset[]>([])
  const [flowMessages, setFlowMessages] = useState<FlowValidationMessage[]>([])

  // ---- 批次2B-1：自定义阶段状态 ----
  const [customStages, setCustomStages] = useState<CustomStageResponse[]>([])
  const [stageModalMode, setStageModalMode] = useState<'create' | 'edit' | null>(null)
  const [editingStageCode, setEditingStageCode] = useState<string>('')
  const [editingStageData, setEditingStageData] = useState<Record<string, unknown> | null>(null)
  const [stageSaving, setStageSaving] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  // 加载预设模板
  useEffect(() => {
    getFlowPresets().then(resp => setFlowPresets(resp.presets || [])).catch(() => {})
  }, [])

  // ---- 批次2B-1：编辑态加载已有自定义阶段 ----
  const loadCustomStages = useCallback(async (id: string) => {
    try {
      const resp = await getCustomStages(id)
      setCustomStages(resp.stages || [])
    } catch { setCustomStages([]) }
  }, [])

  useEffect(() => {
    if (recipeId) loadCustomStages(recipeId)
  }, [recipeId, loadCustomStages])

  // 流程校验（仅系统阶段参与规则校验）
  const triggerValidation = useCallback(async (flow: StageFlowItem[]) => {
    try {
      const systemFlow = flow.filter(s => !s.is_custom)
      const result = await validateFlow(systemFlow)
      setFlowMessages(result.messages || [])
    } catch {
      setFlowMessages([])
    }
  }, [])

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { triggerValidation(formData.stageFlow) }, [formData.stageFlow, triggerValidation])

  // 启用阶段数
  const enabledCount = useMemo(() =>
    formData.stageFlow.filter(s => s.enabled).length,
    [formData.stageFlow]
  )

  // ---- 阶段显示信息辅助（兼容系统/自定义）----
  const getStageName = (stage: StageFlowItem) =>
    stage.is_custom ? (stage.stage_name || stage.stage_code) : (STAGE_CODE_NAME[stage.stage_code] || stage.stage_code)
  const getStageEmoji = (stage: StageFlowItem) =>
    stage.is_custom ? '🔧' : (STAGE_CODE_EMOJI[stage.stage_code] || '📋')
  const getStageRole = (stage: StageFlowItem) => {
    if (stage.is_custom) {
      const cs = customStages.find(c => c.stage_code === stage.stage_code)
      return cs?.ai_role || '自定义角色'
    }
    return STAGE_CODE_ROLE[stage.stage_code] || ''
  }
  const getStageDesc = (stage: StageFlowItem) =>
    stage.is_custom ? '自定义阶段' : (STAGE_CODE_DESC[stage.stage_code] || '')
  const isStageRemovable = (stage: StageFlowItem) =>
    stage.is_custom ? true : (STAGE_REMOVABLE[stage.stage_code] !== false)

  // ---- 阶段操作 ----
  const toggleStage = (stage: StageFlowItem) => {
    if (!isStageRemovable(stage)) return
    updateForm({
      stageFlow: formData.stageFlow.map(s =>
        s.stage_code === stage.stage_code ? { ...s, enabled: !s.enabled } : s
      ),
    })
  }

  const moveStage = (idx: number, dir: -1 | 1) => {
    const target = idx + dir
    if (target < 0 || target >= formData.stageFlow.length) return
    if (formData.stageFlow[idx].stage_code === 'revise' || formData.stageFlow[target].stage_code === 'revise') return
    const n = [...formData.stageFlow]; [n[idx], n[target]] = [n[target], n[idx]]
    updateForm({ stageFlow: n.map((s, i) => ({ ...s, order: i + 1 })) })
  }

  const applyPreset = (preset: FlowPreset) => {
    // 保留已有自定义阶段，插在 revise 之前（口径抄编辑器）
    const customItems = formData.stageFlow.filter(s => s.is_custom)
    let newFlow = preset.stages.map(s => ({ ...s }))
    if (customItems.length > 0) {
      const reviseIdx = newFlow.findIndex(s => s.stage_code === 'revise')
      if (reviseIdx >= 0) {
        const before = newFlow.slice(0, reviseIdx)
        const after = newFlow.slice(reviseIdx)
        newFlow = [...before, ...customItems.map(s => ({ ...s, enabled: true })), ...after]
      } else {
        newFlow = [...newFlow, ...customItems.map(s => ({ ...s, enabled: true }))]
      }
    }
    updateForm({ stageFlow: newFlow.map((s, i) => ({ ...s, order: i + 1 })) })
  }

  // ==================== 批次2B-1：自定义阶段增删改（即时落库，决策2 甲）====================
  const openCreateStageModal = () => {
    if (!isEdit || !recipeId) { showToast('请先保存配方后再添加自定义阶段', 'error'); return }
    setStageModalMode('create')
    setEditingStageCode('')
    setEditingStageData(null)
  }

  const openEditStageModal = (stageCode: string) => {
    if (!recipeId) return
    const cs = customStages.find(s => s.stage_code === stageCode)
    if (!cs) return
    setStageModalMode('edit')
    setEditingStageCode(stageCode)
    // BugFix：用后端返回的真实提示词全文回填，缺字段兜底空/{}，
    //   不再写死空串（此前空串会在保存时覆盖数据库已存提示词）
    setEditingStageData({
      stage_code: cs.stage_code,
      stage_name: cs.stage_name,
      ai_role: cs.ai_role,
      system_prompt: cs.system_prompt || '',
      prompt_variants: cs.prompt_variants || '{}',
      output_format: cs.output_format || '{}',
      gate_mode: cs.gate_mode,
      skippable: cs.skippable,
    })
  }

  const handleStageModalConfirm = async (data: CreateCustomStageRequest | UpdateCustomStageRequest) => {
    if (!recipeId) return
    setStageSaving(true)
    try {
      if (stageModalMode === 'create') {
        const created = await createCustomStage(recipeId, data as CreateCustomStageRequest)
        // 并入 stageFlow（带 is_custom，插在 revise 之前）
        const reviseIdx = formData.stageFlow.findIndex(s => s.stage_code === 'revise')
        const newItem: StageFlowItem = {
          stage_code: created.stage_code, enabled: true, order: 0,
          is_custom: true, stage_name: created.stage_name,
        }
        let updated: StageFlowItem[]
        if (reviseIdx >= 0) {
          updated = [...formData.stageFlow.slice(0, reviseIdx), newItem, ...formData.stageFlow.slice(reviseIdx)]
        } else {
          updated = [...formData.stageFlow, newItem]
        }
        updateForm({ stageFlow: updated.map((s, i) => ({ ...s, order: i + 1 })) })
        await loadCustomStages(recipeId)
        showToast(`自定义阶段「${created.stage_name}」已添加`)
      } else if (stageModalMode === 'edit') {
        await updateCustomStage(recipeId, editingStageCode, data as UpdateCustomStageRequest)
        const upd = data as UpdateCustomStageRequest
        updateForm({
          stageFlow: formData.stageFlow.map(s =>
            s.stage_code === editingStageCode ? { ...s, stage_name: upd.stage_name } : s
          ),
        })
        await loadCustomStages(recipeId)
        showToast('自定义阶段已更新')
      }
      setStageModalMode(null)
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '操作失败', 'error')
    } finally { setStageSaving(false) }
  }

  const handleDeleteCustomStage = async (stageCode: string) => {
    if (!recipeId) return
    const cs = customStages.find(s => s.stage_code === stageCode)
    if (!confirm(`确认删除自定义阶段「${cs?.stage_name || stageCode}」？`)) return
    try {
      await deleteCustomStage(recipeId, stageCode)
      updateForm({
        stageFlow: formData.stageFlow.filter(s => s.stage_code !== stageCode).map((s, i) => ({ ...s, order: i + 1 })),
      })
      await loadCustomStages(recipeId)
      showToast('自定义阶段已删除')
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '删除失败', 'error')
    }
  }

  return (
    <div style={stepCardStyle}>
      {/* 顶部提示：说明本步骤是"AI 陪你备课的工作阶段"，并与上一步"教案结构"区分 */}
      <div style={{
        padding: '12px 16px', borderRadius: '8px', marginBottom: '20px',
        background: 'rgba(79,123,232,0.06)', border: '1px solid rgba(79,123,232,0.12)',
      }}>
        <div style={{ fontSize: '13px', color: C.primary, lineHeight: 1.7 }}>
          🔧 这里定义<strong>AI 陪你备课的工作阶段</strong>（分析→设计→撰写→评审→修订）。
          <br />
          它不决定成品教案的板块格式——那是上一步「📋 教案结构」。
          <strong> 默认已是完整5步流程，大多数情况直接跳过即可。</strong>
          高级用户可微调阶段启用、顺序，或添加自定义阶段。
        </div>
      </div>

      {/* ======== 流程搭建器 ======== */}
      <div>
        <div style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          marginBottom: '12px',
        }}>
          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
            🔧 备课阶段
            <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>
              （{enabledCount}个启用）
            </span>
          </div>
          {/* 批次2B-1：＋自定义阶段（新建态置灰 + 提示，决策1 甲）*/}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            {!isEdit && (
              <span style={{ fontSize: '11px', color: C.textMuted }}>保存配方后可添加自定义阶段</span>
            )}
            <button
              onClick={openCreateStageModal}
              disabled={!isEdit}
              style={{
                fontSize: '12px', fontWeight: 600,
                color: isEdit ? C.primary : C.textMuted,
                background: isEdit ? C.primaryLight : '#F3F4F6',
                border: 'none', padding: '6px 14px', borderRadius: '6px',
                cursor: isEdit ? 'pointer' : 'not-allowed',
                opacity: isEdit ? 1 : 0.6,
              }}
            >＋ 自定义阶段</button>
          </div>
        </div>

        {/* 预设模板 */}
        {flowPresets.length > 0 && (
          <div style={{ display: 'flex', gap: '8px', marginBottom: '14px', flexWrap: 'wrap' }}>
            {flowPresets.map(preset => (
              <button
                key={preset.key}
                onClick={() => applyPreset(preset)}
                style={{
                  padding: '6px 12px', borderRadius: '8px',
                  border: `1px solid ${C.border}`, background: 'transparent',
                  fontSize: '12px', color: C.textSec, cursor: 'pointer',
                  display: 'flex', alignItems: 'center', gap: '4px',
                  transition: 'all 150ms ease',
                }}
                onMouseEnter={e => {
                  (e.currentTarget as HTMLElement).style.borderColor = C.primary;
                  (e.currentTarget as HTMLElement).style.color = C.primary
                }}
                onMouseLeave={e => {
                  (e.currentTarget as HTMLElement).style.borderColor = C.border;
                  (e.currentTarget as HTMLElement).style.color = C.textSec
                }}
              >
                <span>{preset.icon}</span>
                <span style={{ fontWeight: 600 }}>{preset.name}</span>
                <span style={{ color: C.textMuted }}>({preset.duration})</span>
              </button>
            ))}
          </div>
        )}

        {/* 阶段列表（系统 + 自定义混排）*/}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {formData.stageFlow.map((stage, idx) => {
            const removable = isStageRemovable(stage)
            const isRevise = stage.stage_code === 'revise'
            return (
              <div key={stage.stage_code} style={{
                display: 'flex', alignItems: 'center', gap: '10px',
                padding: '10px 12px', borderRadius: '10px',
                border: `1px solid ${stage.enabled ? (stage.is_custom ? 'rgba(79,123,232,0.3)' : C.border) : 'rgba(156,163,175,0.2)'}`,
                background: stage.enabled ? (stage.is_custom ? 'rgba(79,123,232,0.03)' : '#FAFBFC') : 'rgba(156,163,175,0.04)',
                opacity: stage.enabled ? 1 : 0.6, transition: 'all 150ms ease',
              }}>
                {/* 排序 */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                  <button
                    onClick={() => moveStage(idx, -1)}
                    disabled={idx === 0 || isRevise}
                    style={{
                      border: 'none', background: 'none', padding: '0',
                      cursor: idx === 0 || isRevise ? 'default' : 'pointer',
                      fontSize: '10px', color: idx === 0 || isRevise ? C.border : C.textMuted,
                    }}
                  >▲</button>
                  <button
                    onClick={() => moveStage(idx, 1)}
                    disabled={idx === formData.stageFlow.length - 1 || isRevise}
                    style={{
                      border: 'none', background: 'none', padding: '0',
                      cursor: idx === formData.stageFlow.length - 1 || isRevise ? 'default' : 'pointer',
                      fontSize: '10px', color: idx === formData.stageFlow.length - 1 || isRevise ? C.border : C.textMuted,
                    }}
                  >▼</button>
                </div>

                {/* 开关 */}
                <div
                  onClick={() => toggleStage(stage)}
                  style={{
                    width: '36px', height: '20px', borderRadius: '10px',
                    cursor: removable ? 'pointer' : 'not-allowed',
                    background: stage.enabled ? C.success : '#D1D5DB',
                    position: 'relative', transition: 'background 200ms ease', flexShrink: 0,
                  }}
                >
                  <div style={{
                    width: '16px', height: '16px', borderRadius: '50%', background: '#fff',
                    position: 'absolute', top: '2px',
                    left: stage.enabled ? '18px' : '2px',
                    transition: 'left 200ms ease',
                    boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
                  }} />
                </div>

                {/* 图标 */}
                <span style={{ fontSize: '18px', flexShrink: 0 }}>
                  {getStageEmoji(stage)}
                </span>

                {/* 信息 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <span style={{
                      fontSize: '14px', fontWeight: 600,
                      color: stage.enabled ? C.text : C.textMuted,
                    }}>
                      {getStageName(stage)}
                    </span>
                    {!removable && (
                      <span style={{
                        fontSize: '10px', padding: '1px 6px', borderRadius: '4px',
                        background: 'rgba(239,68,68,0.08)', color: C.danger,
                      }}>必须</span>
                    )}
                    {stage.is_custom && (
                      <span style={{
                        fontSize: '10px', padding: '1px 6px', borderRadius: '4px',
                        background: C.primaryLight, color: C.primary,
                      }}>自定义</span>
                    )}
                  </div>
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>
                    {getStageRole(stage)} · {getStageDesc(stage)}
                  </div>
                </div>

                {/* 自定义阶段：编辑/删除 */}
                {stage.is_custom && (
                  <div style={{ display: 'flex', gap: '4px', flexShrink: 0 }}>
                    <button onClick={() => openEditStageModal(stage.stage_code)} title="编辑"
                      style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '13px', color: C.primary, padding: '2px 4px' }}>✏️</button>
                    <button onClick={() => handleDeleteCustomStage(stage.stage_code)} title="删除"
                      style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '13px', color: C.danger, padding: '2px 4px' }}>🗑️</button>
                  </div>
                )}

                {/* 序号 */}
                <span style={{ fontSize: '12px', color: C.textMuted, flexShrink: 0 }}>
                  {stage.enabled
                    ? `第${formData.stageFlow.filter((s, j) => j <= idx && s.enabled).length}步`
                    : '已禁用'}
                </span>
              </div>
            )
          })}
        </div>

        {/* 校验消息 */}
        {flowMessages.length > 0 && (
          <div style={{ marginTop: '12px', display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {flowMessages.map((msg, i) => {
              const style = FLOW_MSG_COLORS[msg.level] || FLOW_MSG_COLORS.info
              return (
                <div key={i} style={{
                  padding: '8px 12px', borderRadius: '8px', fontSize: '12px', lineHeight: 1.5,
                  background: style.bg, border: `1px solid ${style.border}`, color: style.text,
                }}>
                  {style.icon} {msg.message}
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* 自定义阶段弹窗 */}
      {stageModalMode && (
        <CustomStageModal
          mode={stageModalMode}
          initial={editingStageData as Parameters<typeof CustomStageModal>[0]['initial']}
          onConfirm={handleStageModalConfirm}
          onCancel={() => setStageModalMode(null)}
          saving={stageSaving}
        />
      )}

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
          zIndex: 9999, whiteSpace: 'nowrap',
          border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
          animation: 'toast-in 200ms ease',
        }}>
          <style>{`@keyframes toast-in { from{opacity:0;transform:translateX(-50%) translateY(8px)} to{opacity:1;transform:translateX(-50%) translateY(0)} }`}</style>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}
