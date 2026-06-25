/**
 * ModelAliasCard — 模型别名映射管理卡片（批三-2新增）
 *
 * 挂载位置：AI管理中心 → 连接配置Tab，网关命名卡片之后。
 *
 * 功能：
 *   1. 兜底别名设置（无规则命中时老师侧显示，默认「智学大模型」）
 *   2. 规则列表 + 增删改：match_type(精确/前缀) + pattern + alias + priority + enabled
 *   3. 预览框：输入真实模型名 → 显示当前会被替换成的别名（自测匹配逻辑）
 *
 * 后端：/api/v1/admin/model-alias/*（admin专属）
 * 说明：本卡片仅 admin 侧管理，老师侧据规则实际渲染替换在批三-3接。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getModelAliasRules, createModelAliasRule, updateModelAliasRule, deleteModelAliasRule,
  getModelAliasFallback, setModelAliasFallback, previewModelAlias,
} from '@/api/ai-config'
import type { ModelAliasRule, ModelAliasRuleRequest } from '@/api/ai-config'
import { C } from './AICenterConstants'

interface ModelAliasCardProps {
  showToast: (message: string, type: 'success' | 'error') => void
}

// 空规则模板
const emptyRule: ModelAliasRuleRequest = {
  match_type: 'prefix', pattern: '', alias: '', priority: 0, enabled: true, note: '',
}

export default function ModelAliasCard({ showToast }: ModelAliasCardProps) {
  const [rules, setRules] = useState<ModelAliasRule[]>([])
  const [fallback, setFallback] = useState('')
  const [fallbackInput, setFallbackInput] = useState('')
  const [loading, setLoading] = useState(true)
  const [savingFallback, setSavingFallback] = useState(false)

  // 新增表单
  const [form, setForm] = useState<ModelAliasRuleRequest>({ ...emptyRule })
  const [creating, setCreating] = useState(false)

  // 编辑中的规则
  const [editId, setEditId] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<ModelAliasRuleRequest>({ ...emptyRule })

  // 预览
  const [previewModel, setPreviewModel] = useState('')
  const [previewResult, setPreviewResult] = useState<string | null>(null)
  const [previewing, setPreviewing] = useState(false)

  // ==================== 加载 ====================
  const load = useCallback(async () => {
    try {
      setLoading(true)
      const [rs, fb] = await Promise.all([getModelAliasRules(), getModelAliasFallback()])
      setRules(rs)
      setFallback(fb)
      setFallbackInput(fb)
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '加载别名规则失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast])

  useEffect(() => { load() }, [load])

  // ==================== 兜底名 ====================
  const handleSaveFallback = async () => {
    if (!fallbackInput.trim()) { showToast('兜底别名不能为空', 'error'); return }
    try {
      setSavingFallback(true)
      const fb = await setModelAliasFallback(fallbackInput.trim())
      setFallback(fb)
      showToast('兜底别名已保存', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '保存兜底别名失败', 'error')
    } finally {
      setSavingFallback(false)
    }
  }

  // ==================== 新增规则 ====================
  const handleCreate = async () => {
    if (!form.pattern.trim() || !form.alias.trim()) { showToast('匹配内容与别名不能为空', 'error'); return }
    try {
      setCreating(true)
      await createModelAliasRule({
        ...form, pattern: form.pattern.trim(), alias: form.alias.trim(), note: (form.note || '').trim(),
      })
      setForm({ ...emptyRule })
      await load()
      showToast('规则已添加', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '添加规则失败', 'error')
    } finally {
      setCreating(false)
    }
  }

  // ==================== 编辑规则 ====================
  const startEdit = (r: ModelAliasRule) => {
    setEditId(r.id)
    setEditForm({ match_type: r.match_type, pattern: r.pattern, alias: r.alias, priority: r.priority, enabled: r.enabled, note: r.note })
  }
  const handleUpdate = async (id: string) => {
    if (!editForm.pattern.trim() || !editForm.alias.trim()) { showToast('匹配内容与别名不能为空', 'error'); return }
    try {
      await updateModelAliasRule(id, {
        ...editForm, pattern: editForm.pattern.trim(), alias: editForm.alias.trim(), note: (editForm.note || '').trim(),
      })
      setEditId(null)
      await load()
      showToast('规则已更新', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '更新规则失败', 'error')
    }
  }
  const handleDelete = async (id: string) => {
    try {
      await deleteModelAliasRule(id)
      await load()
      showToast('规则已删除', 'success')
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '删除规则失败', 'error')
    }
  }
  // 行内切换启用
  const handleToggleEnabled = async (r: ModelAliasRule) => {
    try {
      await updateModelAliasRule(r.id, {
        match_type: r.match_type, pattern: r.pattern, alias: r.alias,
        priority: r.priority, enabled: !r.enabled, note: r.note,
      })
      await load()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '切换失败', 'error')
    }
  }

  // ==================== 预览 ====================
  const handlePreview = async () => {
    if (!previewModel.trim()) return
    try {
      setPreviewing(true)
      setPreviewResult(null)
      const r = await previewModelAlias(previewModel.trim())
      setPreviewResult(r.alias)
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '预览失败', 'error')
    } finally {
      setPreviewing(false)
    }
  }

  // ==================== 样式 ====================
  const inp: React.CSSProperties = {
    padding: '7px 10px', borderRadius: '8px', border: `1px solid ${C.border}`,
    fontSize: '13px', outline: 'none', background: C.white, boxSizing: 'border-box',
  }
  const th: React.CSSProperties = { fontSize: '12px', fontWeight: 600, color: C.textSec, textAlign: 'left', padding: '8px 10px' }
  const td: React.CSSProperties = { fontSize: '13px', color: C.text, padding: '8px 10px', verticalAlign: 'middle' }

  return (
    <div style={{
      background: C.card, borderRadius: '16px', border: `1px solid ${C.border}`,
      boxShadow: '0 2px 8px rgba(0,0,0,0.04)', marginBottom: '20px',
    }}>
      {/* 头部 */}
      <div style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
        <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>🎭 模型别名映射</div>
        <div style={{ fontSize: '13px', color: C.textSec, marginTop: '3px' }}>
          真实模型名→业务别名（精确优先于前缀）。无命中显示兜底名。将来老师侧据此替换，不暴露真实模型。
        </div>
      </div>

      <div style={{ padding: '24px' }}>
        {/* 兜底名 */}
        <div style={{
          padding: '14px', borderRadius: '10px', marginBottom: '18px',
          background: C.warningLight, border: `1px solid ${C.warning}33`,
        }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
            兜底别名（无规则命中时显示）
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            <input value={fallbackInput} onChange={e => setFallbackInput(e.target.value)}
              placeholder="智学大模型" style={{ ...inp, flex: '1 1 200px' }} />
            <button onClick={handleSaveFallback} disabled={savingFallback}
              style={{ padding: '7px 16px', borderRadius: '8px', border: 'none', background: savingFallback ? C.textMuted : C.warning, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: savingFallback ? 'not-allowed' : 'pointer' }}>
              {savingFallback ? '保存中...' : '保存兜底名'}
            </button>
          </div>
          <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>当前生效：{fallback || '（未设置）'}</div>
        </div>

        {/* 预览框 */}
        <div style={{
          padding: '14px', borderRadius: '10px', marginBottom: '18px',
          background: C.primaryLight, border: `1px solid ${C.primaryBorder}`,
        }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
            🔍 预览（输入真实模型名，看会显示成什么）
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
            <input value={previewModel} onChange={e => setPreviewModel(e.target.value)}
              placeholder="例如 anthropic/claude-sonnet-4-5 或 qwen-max"
              style={{ ...inp, flex: '1 1 280px', fontFamily: 'monospace' }} />
            <button onClick={handlePreview} disabled={previewing || !previewModel.trim()}
              style={{ padding: '7px 16px', borderRadius: '8px', border: 'none', background: (previewing || !previewModel.trim()) ? C.textMuted : C.primary, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: (previewing || !previewModel.trim()) ? 'not-allowed' : 'pointer' }}>
              {previewing ? '...' : '预览'}
            </button>
            {previewResult !== null && (
              <span style={{ fontSize: '13px', color: C.text }}>
                → 显示为 <b style={{ color: C.primary }}>{previewResult}</b>
              </span>
            )}
          </div>
        </div>

        {/* 新增规则 */}
        <div style={{ padding: '14px', borderRadius: '10px', marginBottom: '16px', border: `1px dashed ${C.border}`, background: C.bg }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '10px' }}>新增规则</div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
            <select value={form.match_type} onChange={e => setForm(p => ({ ...p, match_type: e.target.value as 'exact' | 'prefix' }))}
              style={{ ...inp, cursor: 'pointer' }}>
              <option value="prefix">前缀</option>
              <option value="exact">精确</option>
            </select>
            <input value={form.pattern} onChange={e => setForm(p => ({ ...p, pattern: e.target.value }))}
              placeholder="匹配内容 如 anthropic/" style={{ ...inp, flex: '1 1 200px', fontFamily: 'monospace' }} />
            <input value={form.alias} onChange={e => setForm(p => ({ ...p, alias: e.target.value }))}
              placeholder="别名 如 智学国际大模型" style={{ ...inp, flex: '1 1 160px' }} />
            <input type="number" value={form.priority} onChange={e => setForm(p => ({ ...p, priority: parseInt(e.target.value) || 0 }))}
              placeholder="优先级" title="优先级（大者优先）" style={{ ...inp, width: '80px' }} />
            <button onClick={handleCreate} disabled={creating}
              style={{ padding: '7px 16px', borderRadius: '8px', border: 'none', background: creating ? C.textMuted : C.success, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: creating ? 'not-allowed' : 'pointer', whiteSpace: 'nowrap' }}>
              {creating ? '...' : '+ 添加'}
            </button>
          </div>
        </div>

        {/* 规则列表 */}
        {loading ? (
          <div style={{ fontSize: '13px', color: C.textMuted, padding: '8px 0' }}>加载中...</div>
        ) : rules.length === 0 ? (
          <div style={{ fontSize: '13px', color: C.textMuted, padding: '8px 0' }}>暂无规则，全部模型显示兜底名「{fallback}」。</div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                  <th style={th}>类型</th>
                  <th style={th}>匹配内容</th>
                  <th style={th}>别名</th>
                  <th style={{ ...th, width: '70px' }}>优先级</th>
                  <th style={{ ...th, width: '70px' }}>启用</th>
                  <th style={{ ...th, width: '120px' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {rules.map(r => editId === r.id ? (
                  // 编辑行
                  <tr key={r.id} style={{ borderBottom: `1px solid ${C.border}`, background: C.primaryLight }}>
                    <td style={td}>
                      <select value={editForm.match_type} onChange={e => setEditForm(p => ({ ...p, match_type: e.target.value as 'exact' | 'prefix' }))} style={{ ...inp, cursor: 'pointer' }}>
                        <option value="prefix">前缀</option>
                        <option value="exact">精确</option>
                      </select>
                    </td>
                    <td style={td}><input value={editForm.pattern} onChange={e => setEditForm(p => ({ ...p, pattern: e.target.value }))} style={{ ...inp, width: '100%', fontFamily: 'monospace' }} /></td>
                    <td style={td}><input value={editForm.alias} onChange={e => setEditForm(p => ({ ...p, alias: e.target.value }))} style={{ ...inp, width: '100%' }} /></td>
                    <td style={td}><input type="number" value={editForm.priority} onChange={e => setEditForm(p => ({ ...p, priority: parseInt(e.target.value) || 0 }))} style={{ ...inp, width: '60px' }} /></td>
                    <td style={td}>
                      <input type="checkbox" checked={editForm.enabled} onChange={e => setEditForm(p => ({ ...p, enabled: e.target.checked }))} />
                    </td>
                    <td style={td}>
                      <button onClick={() => handleUpdate(r.id)} style={{ padding: '4px 10px', borderRadius: '6px', border: 'none', background: C.success, color: '#fff', fontSize: '12px', cursor: 'pointer', marginRight: '4px' }}>保存</button>
                      <button onClick={() => setEditId(null)} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '12px', cursor: 'pointer' }}>取消</button>
                    </td>
                  </tr>
                ) : (
                  // 展示行
                  <tr key={r.id} style={{ borderBottom: `1px solid ${C.border}`, opacity: r.enabled ? 1 : 0.5 }}>
                    <td style={td}>
                      <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '8px', fontWeight: 600, background: r.match_type === 'exact' ? C.successLight : C.primaryLight, color: r.match_type === 'exact' ? C.success : C.primary }}>
                        {r.match_type === 'exact' ? '精确' : '前缀'}
                      </span>
                    </td>
                    <td style={{ ...td, fontFamily: 'monospace', fontSize: '12px' }}>{r.pattern}</td>
                    <td style={td}><b>{r.alias}</b></td>
                    <td style={td}>{r.priority}</td>
                    <td style={td}>
                      <button onClick={() => handleToggleEnabled(r)} style={{ padding: '2px 8px', borderRadius: '8px', border: 'none', cursor: 'pointer', fontSize: '11px', fontWeight: 600, background: r.enabled ? C.successLight : C.bg, color: r.enabled ? C.success : C.textMuted }}>
                        {r.enabled ? '✓ 启用' : '○ 停用'}
                      </button>
                    </td>
                    <td style={td}>
                      <button onClick={() => startEdit(r)} style={{ padding: '4px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: C.white, color: C.textSec, fontSize: '12px', cursor: 'pointer', marginRight: '4px' }}>编辑</button>
                      <button onClick={() => handleDelete(r.id)} style={{ padding: '4px 10px', borderRadius: '6px', border: '1px solid #FEE2E2', background: '#FEF2F2', color: '#EF4444', fontSize: '12px', cursor: 'pointer' }}>删除</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
