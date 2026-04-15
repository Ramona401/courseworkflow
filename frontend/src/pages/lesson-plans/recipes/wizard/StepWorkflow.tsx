/**
 * StepWorkflow — 配方向导步骤5：备课流程配置
 *
 * v79 新增：分步向导式配方创建
 *
 * 内容：
 *   - 备课模式选择（引导版/高效版/逐阶段）
 *   - 5阶段流程搭建器（启用/禁用+排序）
 *   - 预设模板快速选择
 *   - 流程完整性校验提示
 *
 * 设计目标：
 *   - 默认已是5阶段全开，大多数老师直接跳过即可
 *   - 高级用户可微调阶段顺序和启用状态
 *   - 自定义阶段功能在编辑页使用（向导中不开放，保持简洁）
 */
import { useState, useEffect, useCallback, useMemo } from 'react'
import type { PromptMode, StageFlowItem } from '@/api/recipes'
import { getFlowPresets, validateFlow, type FlowPreset, type FlowValidationMessage } from '@/api/recipes'
import {
  STAGE_CODE_EMOJI, STAGE_CODE_NAME, STAGE_CODE_ROLE, STAGE_CODE_DESC,
  STAGE_REMOVABLE, FLOW_MSG_COLORS, PROMPT_MODE_OPTIONS,
} from '../../workshop/components/workshopConstants'
import {
  C, stepCardStyle,
  type WizardFormData,
} from './wizardConstants'

/* ==================== Props 类型 ==================== */
interface StepWorkflowProps {
  formData: WizardFormData
  updateForm: (updates: Partial<WizardFormData>) => void
}

/* ==================== 组件 ==================== */
export default function StepWorkflow({ formData, updateForm }: StepWorkflowProps) {
  const [flowPresets, setFlowPresets] = useState<FlowPreset[]>([])
  const [flowMessages, setFlowMessages] = useState<FlowValidationMessage[]>([])

  // 加载预设模板
  useEffect(() => {
    getFlowPresets().then(resp => setFlowPresets(resp.presets || [])).catch(() => {})
  }, [])

  // 流程校验
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

  // ---- 备课模式切换 ----
  const handlePromptModeChange = (newMode: PromptMode) => {
    updateForm({ promptMode: newMode })
  }

  // ---- 阶段操作 ----
  const toggleStage = (code: string) => {
    if (STAGE_REMOVABLE[code] === false) return
    updateForm({
      stageFlow: formData.stageFlow.map(s =>
        s.stage_code === code ? { ...s, enabled: !s.enabled } : s
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
    updateForm({
      stageFlow: preset.stages.map((s, i) => ({ ...s, order: i + 1 })),
      promptMode: preset.prompt_mode as PromptMode,
    })
  }

  return (
    <div style={stepCardStyle}>
      {/* 顶部提示 */}
      <div style={{
        padding: '12px 16px', borderRadius: '8px', marginBottom: '20px',
        background: 'rgba(79,123,232,0.06)', border: '1px solid rgba(79,123,232,0.12)',
      }}>
        <div style={{ fontSize: '13px', color: C.primary, lineHeight: 1.6 }}>
          🔧 配置备课的阶段流程和AI对话风格。
          <strong> 默认已是完整5步流程，大多数情况直接跳过即可。</strong>
          高级用户可微调阶段启用和顺序。
        </div>
      </div>

      {/* ======== 备课模式 ======== */}
      <div style={{ marginBottom: '24px' }}>
        <div style={{
          fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '10px',
        }}>
          🎛️ 备课模式
        </div>
        <div style={{ display: 'flex', gap: '10px' }}>
          {PROMPT_MODE_OPTIONS.filter(item => item.mode !== 'per_stage').map(item => (
            <div
              key={item.mode}
              onClick={() => handlePromptModeChange(item.mode)}
              style={{
                flex: 1, padding: '14px', borderRadius: '10px', cursor: 'pointer',
                transition: 'all 150ms ease',
                border: `2px solid ${formData.promptMode === item.mode ? C.primary : C.border}`,
                background: formData.promptMode === item.mode ? C.primaryLight : 'transparent',
              }}
            >
              <div style={{
                fontSize: '14px', fontWeight: 600, marginBottom: '4px',
                color: formData.promptMode === item.mode ? C.primary : C.text,
              }}>
                {item.icon} {item.label}
              </div>
              <div style={{ fontSize: '12px', color: C.textMuted, lineHeight: 1.5 }}>
                {item.desc}
              </div>
            </div>
          ))}
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

        {/* 阶段列表 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {formData.stageFlow.map((stage, idx) => {
            const removable = STAGE_REMOVABLE[stage.stage_code] !== false
            const isRevise = stage.stage_code === 'revise'
            return (
              <div key={stage.stage_code} style={{
                display: 'flex', alignItems: 'center', gap: '10px',
                padding: '10px 12px', borderRadius: '10px',
                border: `1px solid ${stage.enabled ? C.border : 'rgba(156,163,175,0.2)'}`,
                background: stage.enabled ? '#FAFBFC' : 'rgba(156,163,175,0.04)',
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
                  onClick={() => toggleStage(stage.stage_code)}
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
                  {STAGE_CODE_EMOJI[stage.stage_code] || '📋'}
                </span>

                {/* 信息 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <span style={{
                      fontSize: '14px', fontWeight: 600,
                      color: stage.enabled ? C.text : C.textMuted,
                    }}>
                      {STAGE_CODE_NAME[stage.stage_code] || stage.stage_code}
                    </span>
                    {!removable && (
                      <span style={{
                        fontSize: '10px', padding: '1px 6px', borderRadius: '4px',
                        background: 'rgba(239,68,68,0.08)', color: C.danger,
                      }}>必须</span>
                    )}
                  </div>
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '2px' }}>
                    {STAGE_CODE_ROLE[stage.stage_code] || ''} · {STAGE_CODE_DESC[stage.stage_code] || ''}
                  </div>
                </div>

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
    </div>
  )
}
