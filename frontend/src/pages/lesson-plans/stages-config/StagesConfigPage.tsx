/**
 * StagesConfigPage — 阶段管理页面（Admin专用）
 *
 * 路由：/lesson-plans/stages-config
 * 权限：仅admin可见
 *
 * 迭代1 改造：
 *   - 编辑区域从单一textarea改为三区编辑
 *     共享段（system_prompt）：角色+工作目标，两版本共用
 *     引导版变体（prompt_variants.guided）：多轮对话策略
 *     高效版变体（prompt_variants.efficient）：快速出方案策略
 *   - 技术段说明（只读提示，硬编码在后端不可编辑）
 */
import { useState, useEffect, useCallback } from 'react'
import { getAdminStages, updateAdminStage } from '@/api/lesson-plans'

// ==================== 颜色常量 ====================
const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444', accent: '#F59E0B',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  bg: '#FAFBFC', card: '#FFFFFF', border: '#F3F4F6',
}

// ==================== 阶段数据类型（迭代1：增加prompt_variants）====================
interface StageItem {
  id: string
  stage_code: string
  stage_name: string
  stage_order: number
  ai_role: string
  system_prompt: string
  prompt_variants: string  // 迭代1新增：JSON字符串 {"guided":"...","efficient":"..."}
  output_format: string
  component_types: string
  gate_mode: string
  skippable: boolean
  status: string
}

// 解析prompt_variants JSON为对象
function parseVariants(json: string): { guided: string; efficient: string } {
  try {
    const parsed = JSON.parse(json || '{}')
    return { guided: parsed.guided || '', efficient: parsed.efficient || '' }
  } catch { return { guided: '', efficient: '' } }
}

// ==================== 主组件 ====================
export default function StagesConfigPage() {
  const [stages, setStages] = useState<StageItem[]>([])
  const [loading, setLoading] = useState(true)
  const [editingCode, setEditingCode] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<StageItem | null>(null)
  // 迭代1：变体段独立编辑状态
  const [editGuided, setEditGuided] = useState('')
  const [editEfficient, setEditEfficient] = useState('')
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const loadStages = useCallback(async () => {
    try {
      setLoading(true)
      const resp = await getAdminStages()
      setStages(resp.stages || [])
    } catch (e) {
      console.error('加载阶段失败:', e)
      setToast({ msg: '加载失败', type: 'error' })
    } finally { setLoading(false) }
  }, [])

  useEffect(() => { loadStages() }, [loadStages])
  useEffect(() => { if (!toast) return; const t = setTimeout(() => setToast(null), 3000); return () => clearTimeout(t) }, [toast])

  // 进入编辑模式：解析变体段
  const startEdit = (stage: StageItem) => {
    setEditingCode(stage.stage_code)
    setEditForm({ ...stage })
    const v = parseVariants(stage.prompt_variants)
    setEditGuided(v.guided)
    setEditEfficient(v.efficient)
  }

  const cancelEdit = () => { setEditingCode(null); setEditForm(null); setEditGuided(''); setEditEfficient('') }

  // 保存：将三区内容序列化回去
  const handleSave = async () => {
    if (!editForm || !editingCode) return
    setSaving(true)
    try {
      // 迭代1：将引导版/高效版变体序列化为JSON
      const variantsJSON = JSON.stringify({ guided: editGuided, efficient: editEfficient })
      await updateAdminStage(editingCode, {
        stage_name: editForm.stage_name,
        ai_role: editForm.ai_role,
        system_prompt: editForm.system_prompt,
        prompt_variants: variantsJSON,
        output_format: editForm.output_format,
        component_types: editForm.component_types,
        gate_mode: editForm.gate_mode,
        skippable: editForm.skippable,
        status: editForm.status,
      })
      setToast({ msg: `「${editForm.stage_name}」保存成功`, type: 'success' })
      cancelEdit()
      await loadStages()
    } catch (e) {
      console.error('保存失败:', e)
      setToast({ msg: '保存失败，请重试', type: 'error' })
    } finally { setSaving(false) }
  }

  const codeEmoji: Record<string, string> = { analyze: '🔍', design: '🎯', write: '✏️', review: '🤖', revise: '📝' }

  // textarea 通用样式
  const promptTextarea: React.CSSProperties = {
    width: '100%', padding: '12px 14px', borderRadius: '10px',
    border: `1px solid ${C.border}`, fontSize: '13px', color: C.text,
    lineHeight: 1.7, fontFamily: '"Noto Sans SC", sans-serif',
    outline: 'none', resize: 'vertical', boxSizing: 'border-box', background: '#FAFBFC',
  }

  if (loading) return <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '40vh', color: C.textMuted }}>加载中...</div>

  return (
    <div style={{ maxWidth: '1000px', margin: '0 auto', padding: '8px 0' }}>
      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', top: '80px', right: '24px', zIndex: 9999,
          padding: '12px 20px', borderRadius: '10px',
          background: toast.type === 'success' ? C.success : C.danger, color: '#fff',
          fontSize: '14px', fontWeight: 600, boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
          animation: 'toastIn 200ms ease',
        }}>
          {toast.type === 'success' ? '✅' : '❌'} {toast.msg}
          <style>{`@keyframes toastIn { from { opacity: 0; transform: translateY(-10px); } to { opacity: 1; transform: translateY(0); } }`}</style>
        </div>
      )}

      {/* 说明卡 */}
      <div style={{
        padding: '16px 20px', marginBottom: '20px', borderRadius: '12px',
        background: C.primaryLight, border: `1px solid rgba(79,123,232,0.15)`,
        fontSize: '13px', color: C.primary, lineHeight: 1.7,
      }}>
        💡 每个阶段的提示词分为三段：<strong>共享段</strong>（角色+目标，两版本共用）+ <strong>引导版变体</strong>（多轮对话策略）+ <strong>高效版变体</strong>（快速出方案策略）。
        技术段（产出物标签规则）由系统自动追加，不在此编辑。修改后<strong>下次新建教案</strong>生效。
      </div>

      {/* 阶段卡片列表 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
        {stages.map(stage => {
          const isEditing = editingCode === stage.stage_code
          const form = isEditing ? editForm! : stage
          const isDisabled = stage.status === 'disabled'
          const variants = parseVariants(stage.prompt_variants)

          return (
            <div key={stage.stage_code} style={{
              background: C.card, borderRadius: '14px',
              border: `1px solid ${isEditing ? C.primary : C.border}`,
              boxShadow: isEditing ? '0 4px 20px rgba(79,123,232,0.12)' : '0 1px 4px rgba(0,0,0,0.04)',
              opacity: isDisabled && !isEditing ? 0.6 : 1, transition: 'all 200ms ease',
            }}>
              {/* 卡片头部 */}
              <div style={{ padding: '16px 20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: isEditing ? `1px solid ${C.border}` : 'none' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span style={{ fontSize: '22px' }}>{codeEmoji[stage.stage_code] || '📋'}</span>
                  <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{stage.stage_name}</span>
                      <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '10px', background: isDisabled ? '#FEE2E2' : 'rgba(16,185,129,0.1)', color: isDisabled ? C.danger : C.success }}>
                        {isDisabled ? '已禁用' : '启用中'}
                      </span>
                    </div>
                    <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px' }}>
                      阶段 {stage.stage_order} · {stage.ai_role} · {stage.gate_mode === 'suggest' ? '建议确认' : stage.gate_mode === 'force' ? '强制确认' : '自动'}
                      {' · '}共享段{stage.system_prompt.length}字
                      {' · '}引导版{variants.guided.length}字
                      {' · '}高效版{variants.efficient.length}字
                    </div>
                  </div>
                </div>
                <div>
                  {!isEditing ? (
                    <button onClick={() => startEdit(stage)} style={{ padding: '8px 18px', borderRadius: '8px', border: `1px solid ${C.border}`, background: C.bg, fontSize: '13px', color: C.primary, fontWeight: 600, cursor: 'pointer' }}>✏️ 编辑</button>
                  ) : (
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <button onClick={cancelEdit} disabled={saving} style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.textSec, cursor: 'pointer' }}>取消</button>
                      <button onClick={handleSave} disabled={saving} style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: saving ? '#E5E7EB' : C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer' }}>
                        {saving ? '保存中...' : '💾 保存'}
                      </button>
                    </div>
                  )}
                </div>
              </div>

              {/* 编辑区域（迭代1：三区编辑）*/}
              {isEditing && editForm && (
                <div style={{ padding: '20px' }}>
                  {/* 第一行：基本信息 */}
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: '14px', marginBottom: '16px' }}>
                    <div>
                      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>阶段名称</label>
                      <input value={form.stage_name} onChange={e => setEditForm({ ...form, stage_name: e.target.value })}
                        style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                    </div>
                    <div>
                      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>AI角色</label>
                      <input value={form.ai_role} onChange={e => setEditForm({ ...form, ai_role: e.target.value })}
                        style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
                    </div>
                    <div>
                      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>门控模式</label>
                      <select value={form.gate_mode} onChange={e => setEditForm({ ...form, gate_mode: e.target.value })}
                        style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', background: C.card, boxSizing: 'border-box' }}>
                        <option value="suggest">建议确认</option>
                        <option value="force">强制确认</option>
                        <option value="auto">自动进入</option>
                      </select>
                    </div>
                    <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-end', paddingBottom: '4px' }}>
                      <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: C.text, cursor: 'pointer' }}>
                        <input type="checkbox" checked={form.skippable} onChange={e => setEditForm({ ...form, skippable: e.target.checked })} /> 可跳过
                      </label>
                      <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: form.status === 'active' ? C.success : C.danger, cursor: 'pointer' }}>
                        <input type="checkbox" checked={form.status === 'active'} onChange={e => setEditForm({ ...form, status: e.target.checked ? 'active' : 'disabled' })} /> 启用
                      </label>
                    </div>
                  </div>

                  {/* 共享段（角色+工作目标）*/}
                  <div style={{ marginBottom: '16px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                      <label style={{ fontSize: '12px', fontWeight: 600, color: C.textSec }}>📝 共享段（角色设定+工作目标，引导版和高效版共用）</label>
                      <span style={{ fontSize: '11px', color: C.textMuted }}>{form.system_prompt.length} 字</span>
                    </div>
                    <textarea value={form.system_prompt} onChange={e => setEditForm({ ...form, system_prompt: e.target.value })} rows={8} style={promptTextarea} />
                  </div>

                  {/* 引导版变体 + 高效版变体（并排）*/}
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px', marginBottom: '16px' }}>
                    <div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                        <label style={{ fontSize: '12px', fontWeight: 600, color: '#3B82F6' }}>🧭 引导版对话策略</label>
                        <span style={{ fontSize: '11px', color: C.textMuted }}>{editGuided.length} 字</span>
                      </div>
                      <textarea value={editGuided} onChange={e => setEditGuided(e.target.value)} rows={8}
                        style={{ ...promptTextarea, borderColor: 'rgba(59,130,246,0.2)', background: 'rgba(59,130,246,0.02)' }} />
                      <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>多轮对话、逐步引导、适合新手（15-25分钟）</div>
                    </div>
                    <div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                        <label style={{ fontSize: '12px', fontWeight: 600, color: '#F59E0B' }}>⚡ 高效版对话策略</label>
                        <span style={{ fontSize: '11px', color: C.textMuted }}>{editEfficient.length} 字</span>
                      </div>
                      <textarea value={editEfficient} onChange={e => setEditEfficient(e.target.value)} rows={8}
                        style={{ ...promptTextarea, borderColor: 'rgba(245,158,11,0.2)', background: 'rgba(245,158,11,0.02)' }} />
                      <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>直接出方案、快速确认、适合经验丰富（5-10分钟）</div>
                    </div>
                  </div>

                  {/* 技术段说明（只读）*/}
                  <div style={{ padding: '10px 14px', borderRadius: '8px', background: 'rgba(107,114,128,0.04)', border: `1px solid rgba(107,114,128,0.1)`, marginBottom: '16px' }}>
                    <div style={{ fontSize: '12px', fontWeight: 600, color: C.textMuted, marginBottom: '4px' }}>🔧 技术段（系统自动追加，不可编辑）</div>
                    <div style={{ fontSize: '11px', color: C.textMuted, lineHeight: 1.6 }}>
                      系统会在上述提示词之后自动追加产出物标签规则（&lt;stage_output&gt;格式定义+&lt;stage_complete/&gt;检测规则），确保AI输出可被系统解析。此部分硬编码在后端，管理员无需关心。
                    </div>
                  </div>

                  {/* 产出物格式 + 组件类型 */}
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                    <div>
                      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>产出物格式（JSON）</label>
                      <textarea value={form.output_format} onChange={e => setEditForm({ ...form, output_format: e.target.value })}
                        rows={3} style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '12px', fontFamily: 'monospace', outline: 'none', resize: 'vertical', boxSizing: 'border-box' }} />
                    </div>
                    <div>
                      <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '6px' }}>注入组件类型（JSON数组）</label>
                      <textarea value={form.component_types} onChange={e => setEditForm({ ...form, component_types: e.target.value })}
                        rows={3} style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '12px', fontFamily: 'monospace', outline: 'none', resize: 'vertical', boxSizing: 'border-box' }} />
                    </div>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
