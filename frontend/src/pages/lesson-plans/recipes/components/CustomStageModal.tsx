/**
 * CustomStageModal — 自定义阶段创建/编辑弹窗
 *
 * 迭代5新增：老师在配方编辑器中添加/编辑自定义备课阶段
 * 包含：阶段代码、名称、AI角色、系统提示词、对话策略、产出物格式、门控模式、可跳过
 *
 * 【配方搭建一页化 · 批次2C】清理「引导版/高效版」双策略残留（保守口径）：
 *   背景：主流程恒走 guided（动作2 批次A.1/B.1 已下线备课模式选择器），efficient 维度填了
 *   也永不被后端读取，与「配方一律 guided」口径不一致，故清理。
 *   做法（保守，不改写存量数据，不碰后端字段/API）：
 *     - UI 上「🧭 引导版策略 / ⚡ 高效版策略」两栏并为单个「💬 对话策略」全宽框；
 *       底层 prompt_variants 的 guided 键保持不变（后端读取键名不可改），仅去掉 UI 的「版本」语义。
 *     - 编辑老阶段时，prompt_variants JSON 里 guided 之外的键（主要是 efficient，也含任何其它键）
 *       由 preservedExtraVariants 原样存住、提交时合并写回——隐藏不可编辑，但数据零丢失。
 *     - 新建阶段 preservedExtraVariants 为空，JSON 仅 { guided }，天然干净。
 *   注：efficient 的「深清理」（连后端字段/存量数据一起退役）属 prompt_mode 全局清理范畴，另案处理。
 */
import { useState, useEffect } from 'react'
import type { CreateCustomStageRequest, UpdateCustomStageRequest } from '@/api/recipes'

/* ==================== 颜色常量 ==================== */
const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  card: '#FFFFFF', border: '#F3F4F6', bg: '#FAFBFC',
}

/* ==================== Props ==================== */
interface Props {
  /** 弹窗模式：create=创建 / edit=编辑 */
  mode: 'create' | 'edit'
  /** 编辑模式时的初始数据 */
  initial?: {
    stage_code: string
    stage_name: string
    ai_role: string
    system_prompt: string
    prompt_variants: string
    output_format: string
    gate_mode: string
    skippable: boolean
  }
  /** 确认回调 */
  onConfirm: (data: CreateCustomStageRequest | UpdateCustomStageRequest) => void
  /** 取消回调 */
  onCancel: () => void
  /** 是否正在保存 */
  saving?: boolean
}

/* ==================== 组件 ==================== */
export default function CustomStageModal({ mode, initial, onConfirm, onCancel, saving }: Props) {
  const isEdit = mode === 'edit'

  // 表单状态
  const [stageCode, setStageCode] = useState(initial?.stage_code || '')
  const [stageName, setStageName] = useState(initial?.stage_name || '')
  const [aiRole, setAIRole] = useState(initial?.ai_role || '')
  const [systemPrompt, setSystemPrompt] = useState(initial?.system_prompt || '')
  // 批次2C：单一「对话策略」，底层仍对应 prompt_variants.guided 键
  const [dialogStrategy, setDialogStrategy] = useState('')
  // 批次2C（保守）：编辑老阶段时，guided 之外的键（如 efficient）原样存住，提交时合并写回，不丢数据
  const [preservedExtraVariants, setPreservedExtraVariants] = useState<Record<string, string>>({})
  const [outputFormat, setOutputFormat] = useState(initial?.output_format || '')
  const [gateMode, setGateMode] = useState(initial?.gate_mode || 'suggest')
  const [skippable, setSkippable] = useState(initial?.skippable ?? true)

  // 解析 prompt_variants JSON：guided 进对话策略输入框，其余键（efficient 等）进 preserved 隐藏保留
  useEffect(() => {
    if (initial?.prompt_variants) {
      try {
        const pv = JSON.parse(initial.prompt_variants) as Record<string, string>
        const { guided, ...rest } = pv
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setDialogStrategy(guided || '')
        setPreservedExtraVariants(rest || {})
      } catch { /* 忽略解析错误 */ }
    }
  }, [initial?.prompt_variants])

  // 提交
  const handleSubmit = () => {
    if (!stageName.trim() || !aiRole.trim()) return
    if (!isEdit && !stageCode.trim()) return

    // 构建 prompt_variants JSON：先铺隐藏保留键（老 efficient 等），再写当前对话策略到 guided
    const variants: Record<string, string> = { ...preservedExtraVariants }
    if (dialogStrategy.trim()) {
      variants.guided = dialogStrategy.trim()
    } else {
      delete variants.guided
    }
    const promptVariantsJSON = Object.keys(variants).length > 0 ? JSON.stringify(variants) : '{}'

    if (isEdit) {
      const payload: UpdateCustomStageRequest = {
        stage_name: stageName.trim(),
        ai_role: aiRole.trim(),
        system_prompt: systemPrompt.trim(),
        prompt_variants: promptVariantsJSON,
        output_format: outputFormat.trim() || '{}',
        gate_mode: gateMode,
        skippable,
      }
      onConfirm(payload)
    } else {
      const payload: CreateCustomStageRequest = {
        stage_code: stageCode.trim(),
        stage_name: stageName.trim(),
        ai_role: aiRole.trim(),
        system_prompt: systemPrompt.trim(),
        prompt_variants: promptVariantsJSON,
        output_format: outputFormat.trim() || '{}',
        gate_mode: gateMode,
        skippable,
      }
      onConfirm(payload)
    }
  }

  const canSubmit = stageName.trim() && aiRole.trim() && (isEdit || stageCode.trim())

  // 样式
  const labelSt: React.CSSProperties = { display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }
  const inputSt: React.CSSProperties = {
    width: '100%', padding: '8px 12px', borderRadius: '6px', border: `1px solid ${C.border}`,
    fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit',
  }
  const textareaSt: React.CSSProperties = { ...inputSt, resize: 'vertical' as const, lineHeight: 1.6 }

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex',
      alignItems: 'center', justifyContent: 'center', zIndex: 10000,
    }} onClick={e => { if (e.target === e.currentTarget) onCancel() }}>
      <div style={{
        background: C.card, borderRadius: '16px', width: '560px', maxHeight: '85vh',
        overflow: 'auto', boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }}>
        {/* 标题栏 */}
        <div style={{
          padding: '20px 24px 16px', borderBottom: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        }}>
          <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>
            {isEdit ? '✏️ 编辑自定义阶段' : '➕ 添加自定义阶段'}
          </div>
          <button onClick={onCancel} style={{
            border: 'none', background: 'none', cursor: 'pointer', fontSize: '18px', color: C.textMuted, padding: '4px',
          }}>✕</button>
        </div>

        {/* 表单内容 */}
        <div style={{ padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: '14px' }}>
          {/* 阶段代码（创建时可编辑，编辑时只读） */}
          <div>
            <label style={labelSt}>阶段代码 <span style={{ color: C.danger }}>*</span>
              <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '11px' }}>小写英文+数字+下划线</span>
            </label>
            <input
              type="text" value={stageCode}
              onChange={e => setStageCode(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))}
              placeholder="例如：warm_up、group_discuss"
              style={{ ...inputSt, background: isEdit ? '#F9FAFB' : '#fff' }}
              disabled={isEdit} maxLength={30}
            />
          </div>

          {/* 阶段名称 */}
          <div>
            <label style={labelSt}>阶段名称 <span style={{ color: C.danger }}>*</span></label>
            <input type="text" value={stageName} onChange={e => setStageName(e.target.value)}
              placeholder="例如：课前热身、小组讨论" style={inputSt} maxLength={50} />
          </div>

          {/* AI角色 */}
          <div>
            <label style={labelSt}>AI角色 <span style={{ color: C.danger }}>*</span></label>
            <input type="text" value={aiRole} onChange={e => setAIRole(e.target.value)}
              placeholder="例如：热身活动设计师、讨论引导师" style={inputSt} maxLength={50} />
          </div>

          {/* 系统提示词 */}
          <div>
            <label style={labelSt}>系统提示词（共享段）
              <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '11px' }}>AI的角色定义和工作目标</span>
            </label>
            <textarea value={systemPrompt} onChange={e => setSystemPrompt(e.target.value)}
              placeholder="你是一位{AI角色}，负责帮助老师..." rows={4} style={textareaSt} />
          </div>

          {/* 批次2C：单一「对话策略」（原引导版/高效版两栏合并；底层仍写 prompt_variants.guided） */}
          <div>
            <label style={labelSt}>💬 对话策略
              <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '11px' }}>该阶段 AI 与老师对话的引导方式（可选）</span>
            </label>
            <textarea value={dialogStrategy} onChange={e => setDialogStrategy(e.target.value)}
              placeholder="例如：先抛出 2-3 个引导问题帮老师澄清目标，再逐步给出方案..." rows={3} style={textareaSt} />
          </div>

          {/* 产出物格式 */}
          <div>
            <label style={labelSt}>产出物格式（JSON）
              <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px', fontSize: '11px' }}>可选，定义AI输出的结构</span>
            </label>
            <textarea value={outputFormat} onChange={e => setOutputFormat(e.target.value)}
              placeholder='例如：{"type":"json","fields":["summary","suggestions"]}' rows={2} style={{ ...textareaSt, fontFamily: 'monospace', fontSize: '12px' }} />
          </div>

          {/* 门控模式 + 可跳过 */}
          <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-end' }}>
            <div style={{ flex: 1 }}>
              <label style={labelSt}>门控模式</label>
              <select value={gateMode} onChange={e => setGateMode(e.target.value)} style={{ ...inputSt, cursor: 'pointer' }}>
                <option value="suggest">建议确认（suggest）</option>
                <option value="force">强制确认（force）</option>
                <option value="auto">自动进入（auto）</option>
              </select>
            </div>
            <div style={{ flex: 1 }}>
              <label style={{ ...labelSt, display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
                <input type="checkbox" checked={skippable} onChange={e => setSkippable(e.target.checked)}
                  style={{ accentColor: C.primary }} />
                允许跳过此阶段
              </label>
            </div>
          </div>
        </div>

        {/* 底部按钮 */}
        <div style={{
          padding: '16px 24px 20px', borderTop: `1px solid ${C.border}`,
          display: 'flex', justifyContent: 'flex-end', gap: '10px',
        }}>
          <button onClick={onCancel} style={{
            padding: '8px 20px', borderRadius: '8px', border: `1px solid ${C.border}`,
            background: 'transparent', color: C.textSec, fontSize: '13px', cursor: 'pointer',
          }}>取消</button>
          <button onClick={handleSubmit} disabled={!canSubmit || saving} style={{
            padding: '8px 24px', borderRadius: '8px', border: 'none',
            background: canSubmit && !saving ? C.primary : C.border,
            color: canSubmit && !saving ? '#fff' : C.textMuted,
            fontSize: '13px', fontWeight: 600, cursor: canSubmit && !saving ? 'pointer' : 'not-allowed',
          }}>{saving ? '保存中...' : isEdit ? '保存修改' : '添加阶段'}</button>
        </div>
      </div>
    </div>
  )
}
