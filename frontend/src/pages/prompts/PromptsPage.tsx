/**
 * 提示词管理页面（v2 治理改造）
 *
 * 本次改造要点：
 *   1. 独立全屏页：脱离 MainLayout，自带顶栏（← 返回首页 + 标题），入口改由门户卡片进入。
 *   2. 纳管全部 key：后端已放开动态校验，列表现展示 prompts 表全部 28 个 key
 *      （含课件系列/知识库系列），不再受旧 9 白名单限制。
 *   3. 危险分档：每个 key 带 category（high/mid/kb），按档分组展示 + 红/橙/绿色标。
 *      分档元数据集中在 promptCategoryMeta.ts。
 *   4. 二次确认（生产环境强确认）：点「保存新版本」先弹 PromptConfirmModal，
 *      high 档需手动键入「确认」二字方可提交（防误改课件重型提示词导致批量生成崩溃）。
 *   5. 用途说明改用后端下发的 description 字段，删除前端硬编码的旧 PROMPT_DESCRIPTIONS
 *      （旧文案含已过时的「156门课程」叙事）。
 *   - 仅 admin 可访问（路由层 RoleGuard 限制）。
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Save, RotateCcw, ChevronDown, ChevronUp,
  Clock, Hash, CheckCircle, AlertCircle, History,
} from 'lucide-react'
import {
  getPrompts, updatePrompt, getPromptVersions, rollbackPromptVersion,
} from '@/api/prompts'
import type {
  PromptInfo, PromptVersion, PromptCategory,
} from '@/api/prompts'
import { getCategoryMeta, CATEGORY_ORDER, CATEGORY_META } from './promptCategoryMeta'
import PromptConfirmModal from './PromptConfirmModal'

// ==================== Toast 组件 ====================
function Toast({ message, type, onClose }: { message: string; type: 'success' | 'error'; onClose: () => void }) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000)
    return () => clearTimeout(timer)
  }, [onClose])

  return (
    <div style={{
      position: 'fixed', top: '24px', right: '24px', zIndex: 10000,
      padding: '14px 24px', borderRadius: '12px',
      background: type === 'success' ? '#e8f5e9' : '#fce4ec',
      color: type === 'success' ? '#2e7d32' : '#c62828',
      border: `1px solid ${type === 'success' ? '#a5d6a7' : '#ef9a9a'}`,
      boxShadow: '0 8px 32px rgba(0,0,0,0.12)',
      fontSize: '14px', fontWeight: 500,
      display: 'flex', alignItems: 'center', gap: '8px',
    }}>
      {type === 'success' ? <CheckCircle size={16} /> : <AlertCircle size={16} />}
      {message}
    </div>
  )
}

// ==================== 主页面组件 ====================
export default function PromptsPage() {
  const navigate = useNavigate()

  // 状态管理
  const [prompts, setPrompts] = useState<PromptInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedKey, setExpandedKey] = useState<string | null>(null)
  const [editContent, setEditContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)

  // 二次确认弹窗：待确认保存的目标（null 表示未弹出）
  const [confirmTarget, setConfirmTarget] = useState<PromptInfo | null>(null)

  // 版本历史相关
  const [showVersions, setShowVersions] = useState<string | null>(null)
  const [versions, setVersions] = useState<PromptVersion[]>([])
  const [versionsLoading, setVersionsLoading] = useState(false)
  const [rollingBack, setRollingBack] = useState(false)
  const [previewVersion, setPreviewVersion] = useState<PromptVersion | null>(null)

  // 加载提示词列表
  const loadPrompts = useCallback(async () => {
    try {
      setLoading(true)
      const data = await getPrompts()
      setPrompts(data.prompts || [])
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (err: any) {
      setToast({ message: '加载提示词失败: ' + (err?.message || '未知错误'), type: 'error' })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadPrompts() }, [loadPrompts])

  // 展开编辑某个提示词
  const handleExpand = (key: string, content: string) => {
    if (expandedKey === key) {
      setExpandedKey(null)
      setEditContent('')
    } else {
      setExpandedKey(key)
      setEditContent(content)
      setShowVersions(null)
      setPreviewVersion(null)
    }
  }

  // 点击「保存新版本」→ 先校验非空，再弹二次确认弹窗（不直接保存）
  const handleSaveClick = (prompt: PromptInfo) => {
    if (!editContent.trim()) {
      setToast({ message: '提示词内容不能为空', type: 'error' })
      return
    }
    setConfirmTarget(prompt)
  }

  // 二次确认弹窗「确认保存」→ 真正执行保存
  const handleConfirmSave = async () => {
    if (!confirmTarget) return
    const key = confirmTarget.prompt_key
    try {
      setSaving(true)
      await updatePrompt(key, { content: editContent })
      setToast({ message: '提示词已保存（新版本已创建）', type: 'success' })
      setConfirmTarget(null)
      setExpandedKey(null)
      setEditContent('')
      await loadPrompts()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (err: any) {
      setToast({ message: '保存失败: ' + (err?.message || '未知错误'), type: 'error' })
    } finally {
      setSaving(false)
    }
  }

  // 加载版本历史
  const handleShowVersions = async (key: string) => {
    if (showVersions === key) {
      setShowVersions(null)
      setVersions([])
      setPreviewVersion(null)
      return
    }
    try {
      setVersionsLoading(true)
      setShowVersions(key)
      setPreviewVersion(null)
      const data = await getPromptVersions(key)
      setVersions(data.versions || [])
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (err: any) {
      setToast({ message: '加载版本历史失败: ' + (err?.message || '未知错误'), type: 'error' })
    } finally {
      setVersionsLoading(false)
    }
  }

  // 回滚到指定版本
  const handleRollback = async (key: string, versionId: string, versionNum: number) => {
    if (!confirm(`确认回滚到版本 v${versionNum}？当前版本将不再是生效版本。`)) return
    try {
      setRollingBack(true)
      await rollbackPromptVersion(key, { version_id: versionId })
      setToast({ message: `已回滚到版本 v${versionNum}`, type: 'success' })
      setShowVersions(null)
      setVersions([])
      setPreviewVersion(null)
      setExpandedKey(null)
      await loadPrompts()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (err: any) {
      setToast({ message: '回滚失败: ' + (err?.message || '未知错误'), type: 'error' })
    } finally {
      setRollingBack(false)
    }
  }

  // 格式化时间
  const formatTime = (t: string) => {
    if (!t) return '—'
    const d = new Date(t)
    return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  }

  // 格式化字符数
  const formatLen = (len: number) => {
    if (len >= 1000) return `${(len / 1000).toFixed(1)}K`
    return `${len}`
  }

  // ==================== 样式定义 ====================
  const cardStyle: React.CSSProperties = {
    background: 'rgba(255,255,255,0.72)',
    backdropFilter: 'blur(20px)', WebkitBackdropFilter: 'blur(20px)',
    borderRadius: '16px', border: '1px solid rgba(0,0,0,0.06)',
    boxShadow: '0 2px 12px rgba(0,0,0,0.04)',
    marginBottom: '12px', overflow: 'hidden', transition: 'all 0.2s ease',
  }
  const headerStyle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '18px 24px', cursor: 'pointer', transition: 'background 0.15s ease',
  }
  const btnPrimary: React.CSSProperties = {
    padding: '10px 24px', borderRadius: '10px', border: 'none', cursor: 'pointer',
    fontSize: '14px', fontWeight: 600, color: '#fff',
    background: 'linear-gradient(135deg, #007aff, #5856d6)',
    boxShadow: '0 2px 8px rgba(0,122,255,0.3)', transition: 'all 0.2s ease',
  }
  const btnSecondary: React.CSSProperties = {
    padding: '10px 24px', borderRadius: '10px', border: '1px solid #d1d1d6',
    cursor: 'pointer', fontSize: '14px', fontWeight: 500,
    color: '#1d1d1f', background: '#fff', transition: 'all 0.2s ease',
  }
  const btnWarning: React.CSSProperties = {
    padding: '6px 14px', borderRadius: '8px', border: 'none', cursor: 'pointer',
    fontSize: '12px', fontWeight: 600, color: '#fff',
    background: 'linear-gradient(135deg, #ff9500, #ff6b00)',
    boxShadow: '0 2px 6px rgba(255,149,0,0.3)', transition: 'all 0.2s ease',
    display: 'inline-flex', alignItems: 'center', gap: '4px',
  }

  // ==================== 单张提示词卡片渲染 ====================
  const renderPromptCard = (prompt: PromptInfo) => {
    const isExpanded = expandedKey === prompt.prompt_key
    const isVersionsOpen = showVersions === prompt.prompt_key
    const isFilled = prompt.content_len > 50
    const meta = getCategoryMeta(prompt.category)

    return (
      <div key={prompt.prompt_key} style={cardStyle}>
        {/* 卡片头部 */}
        <div
          style={headerStyle}
          onClick={() => handleExpand(prompt.prompt_key, prompt.content)}
          onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.02)' }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '14px', flex: 1 }}>
            {/* 危险分档色标 */}
            <div style={{
              width: '10px', height: '10px', borderRadius: '50%',
              background: meta.color, boxShadow: `0 0 6px ${meta.color}66`, flexShrink: 0,
            }} />
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                <span style={{ fontSize: '15px', fontWeight: 600, color: '#1d1d1f' }}>
                  {prompt.prompt_name}
                </span>
                {/* 分档 chip */}
                <span style={{
                  fontSize: '11px', padding: '2px 8px', borderRadius: '6px',
                  background: meta.bg, color: meta.color, fontWeight: 600,
                  border: `1px solid ${meta.border}`,
                }}>
                  {meta.emoji} {meta.label}
                </span>
                {/* 版本/字数 chip */}
                <span style={{
                  fontSize: '11px', padding: '2px 8px', borderRadius: '6px',
                  background: isFilled ? '#e8f5e9' : '#fff3e0',
                  color: isFilled ? '#2e7d32' : '#e65100', fontWeight: 500,
                }}>
                  {isFilled ? `v${prompt.version} · ${formatLen(prompt.content_len)}字` : '待填入'}
                </span>
                {/* key 原文（便于运维核对） */}
                <span style={{ fontSize: '11px', color: '#c7c7cc', fontFamily: 'ui-monospace, Menlo, monospace' }}>
                  {prompt.prompt_key}
                </span>
              </div>
              <div style={{ fontSize: '12px', color: '#8e8e93', marginTop: '4px' }}>
                {prompt.description || ''}
              </div>
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <button
              onClick={(e) => { e.stopPropagation(); handleShowVersions(prompt.prompt_key) }}
              style={{
                padding: '6px 12px', borderRadius: '8px', border: '1px solid #d1d1d6',
                cursor: 'pointer', fontSize: '12px', fontWeight: 500,
                color: isVersionsOpen ? '#5856d6' : '#8e8e93',
                background: isVersionsOpen ? '#f0efff' : '#fff',
                display: 'flex', alignItems: 'center', gap: '4px', transition: 'all 0.15s ease',
              }}
            >
              <History size={13} /> 版本
            </button>
            {isExpanded ? <ChevronUp size={18} color="#8e8e93" /> : <ChevronDown size={18} color="#8e8e93" />}
          </div>
        </div>

        {/* 展开的编辑区域 */}
        {isExpanded && (
          <div style={{ padding: '0 24px 20px', borderTop: '1px solid rgba(0,0,0,0.04)' }}>
            <div style={{ paddingTop: '16px' }}>
              <div style={{ fontSize: '12px', color: '#8e8e93', marginBottom: '10px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Hash size={12} />
                编辑后保存将创建新版本（v{prompt.version + 1}），旧版本自动归档
              </div>
              <textarea
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                placeholder="请输入提示词完整内容..."
                style={{
                  width: '100%', minHeight: '320px', padding: '16px',
                  borderRadius: '12px', border: '1px solid #d1d1d6',
                  fontSize: '13px', lineHeight: '1.7', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                  resize: 'vertical', outline: 'none', background: '#fafafa', color: '#1d1d1f',
                  transition: 'border-color 0.2s ease, box-shadow 0.2s ease', boxSizing: 'border-box',
                }}
                onFocus={(e) => { e.target.style.borderColor = '#007aff'; e.target.style.boxShadow = '0 0 0 3px rgba(0,122,255,0.1)' }}
                onBlur={(e) => { e.target.style.borderColor = '#d1d1d6'; e.target.style.boxShadow = 'none' }}
              />
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '14px' }}>
                <span style={{ fontSize: '12px', color: '#8e8e93' }}>
                  当前字数：{editContent.length.toLocaleString()} 字
                </span>
                <div style={{ display: 'flex', gap: '10px' }}>
                  <button onClick={() => { setExpandedKey(null); setEditContent('') }} style={btnSecondary}>取消</button>
                  <button
                    onClick={() => handleSaveClick(prompt)}
                    disabled={saving}
                    style={{ ...btnPrimary, opacity: saving ? 0.6 : 1, cursor: saving ? 'not-allowed' : 'pointer', display: 'flex', alignItems: 'center', gap: '6px' }}
                  >
                    <Save size={14} /> {saving ? '保存中...' : '保存新版本'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 版本历史面板 */}
        {isVersionsOpen && (
          <div style={{ padding: '0 24px 20px', borderTop: '1px solid rgba(0,0,0,0.04)' }}>
            <div style={{ paddingTop: '16px' }}>
              <div style={{ fontSize: '14px', fontWeight: 600, color: '#1d1d1f', marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Clock size={14} /> 版本历史 — {prompt.prompt_name}
              </div>
              {versionsLoading ? (
                <div style={{ textAlign: 'center', padding: '20px', color: '#8e8e93', fontSize: '13px' }}>加载中...</div>
              ) : versions.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '20px', color: '#8e8e93', fontSize: '13px' }}>暂无版本记录</div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  {versions.map((v) => (
                    <div key={v.id} style={{
                      padding: '12px 16px', borderRadius: '10px',
                      background: v.is_current ? '#f0f7ff' : '#f9f9f9',
                      border: v.is_current ? '1px solid #a8d4ff' : '1px solid #eee',
                    }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                          <span style={{ fontSize: '13px', fontWeight: 600, color: v.is_current ? '#007aff' : '#1d1d1f' }}>v{v.version}</span>
                          {v.is_current && (
                            <span style={{ fontSize: '11px', padding: '1px 8px', borderRadius: '4px', background: '#007aff', color: '#fff', fontWeight: 500 }}>当前</span>
                          )}
                          <span style={{ fontSize: '12px', color: '#8e8e93' }}>
                            {formatLen(v.content_len)}字 · {formatTime(v.created_at)}
                          </span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <button
                            onClick={() => setPreviewVersion(previewVersion?.id === v.id ? null : v)}
                            style={{
                              padding: '4px 10px', borderRadius: '6px', border: '1px solid #d1d1d6',
                              cursor: 'pointer', fontSize: '11px', fontWeight: 500,
                              color: previewVersion?.id === v.id ? '#5856d6' : '#8e8e93',
                              background: previewVersion?.id === v.id ? '#f0efff' : '#fff',
                            }}
                          >
                            {previewVersion?.id === v.id ? '收起' : '预览'}
                          </button>
                          {!v.is_current && (
                            <button
                              onClick={() => handleRollback(prompt.prompt_key, v.id, v.version)}
                              disabled={rollingBack}
                              style={{ ...btnWarning, opacity: rollingBack ? 0.6 : 1, cursor: rollingBack ? 'not-allowed' : 'pointer' }}
                            >
                              <RotateCcw size={11} /> 回滚
                            </button>
                          )}
                        </div>
                      </div>
                      {previewVersion?.id === v.id && (
                        <div style={{
                          marginTop: '10px', padding: '12px', background: '#fff', borderRadius: '8px', border: '1px solid #eee',
                          maxHeight: '200px', overflowY: 'auto', fontSize: '12px', lineHeight: '1.6',
                          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                          color: '#333', whiteSpace: 'pre-wrap', wordBreak: 'break-all',
                        }}>
                          {v.content}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    )
  }

  // ==================== 渲染 ====================
  // 按分档分组（high/mid/kb），组内保持后端返回顺序（prompt_key 升序）
  const grouped: Record<PromptCategory, PromptInfo[]> = { high: [], mid: [], kb: [] }
  for (const p of prompts) {
    const cat: PromptCategory = (p.category === 'high' || p.category === 'kb') ? p.category : 'mid'
    grouped[cat].push(p)
  }

  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(180deg, #f5f5f7 0%, #eef0f3 100%)' }}>
      {/* Toast */}
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 二次确认弹窗 */}
      {confirmTarget && (
        <PromptConfirmModal
          category={confirmTarget.category}
          promptName={confirmTarget.prompt_name}
          promptKey={confirmTarget.prompt_key}
          nextVersion={confirmTarget.version + 1}
          saving={saving}
          onConfirm={handleConfirmSave}
          onCancel={() => setConfirmTarget(null)}
        />
      )}

      {/* 独立页顶栏 */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: '14px',
        padding: '20px 32px', borderBottom: '1px solid rgba(0,0,0,0.06)',
        background: 'rgba(255,255,255,0.8)', backdropFilter: 'blur(20px)',
        position: 'sticky', top: 0, zIndex: 100,
      }}>
        <button
          onClick={() => navigate('/')}
          style={{
            display: 'flex', alignItems: 'center', gap: '6px', padding: '8px 14px',
            borderRadius: '10px', border: '1px solid #d1d1d6', background: '#fff',
            cursor: 'pointer', fontSize: '13px', color: '#8e8e93', fontWeight: 500,
          }}
        >
          <span style={{ fontSize: '15px' }}>←</span> 返回首页
        </button>
        <div>
          <div style={{ fontSize: '20px', fontWeight: 700, color: '#1d1d1f' }}>📝 提示词管理</div>
          <div style={{ fontSize: '12px', color: '#8e8e93', marginTop: '2px' }}>
            管理各业务链路的提示词内容，支持版本历史与回滚 · 仅管理员可见
          </div>
        </div>
      </div>

      {/* 内容区 */}
      <div style={{ maxWidth: 1080, margin: '0 auto', padding: '28px 32px' }}>
        {loading ? (
          <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '400px' }}>
            <div style={{ width: '32px', height: '32px', border: '2px solid #007aff', borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          </div>
        ) : (
          <>
            {/* 统计卡片（按分档） */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px', marginBottom: '24px' }}>
              {CATEGORY_ORDER.map((cat) => {
                const m = CATEGORY_META[cat]
                return (
                  <div key={cat} style={{ ...cardStyle, padding: '18px 20px', marginBottom: 0, borderLeft: `3px solid ${m.color}` }}>
                    <div style={{ fontSize: '12px', color: '#8e8e93', marginBottom: '6px' }}>{m.emoji} {m.label}</div>
                    <div style={{ fontSize: '28px', fontWeight: 700, color: m.color }}>{grouped[cat].length}</div>
                  </div>
                )
              })}
            </div>

            {/* 按分档分组渲染 */}
            {CATEGORY_ORDER.map((cat) => {
              const list = grouped[cat]
              if (list.length === 0) return null
              const m = CATEGORY_META[cat]
              return (
                <div key={cat} style={{ marginBottom: '28px' }}>
                  <div style={{
                    fontSize: '13px', fontWeight: 700, color: m.color,
                    padding: '8px 14px', marginBottom: '12px', borderRadius: '10px',
                    background: m.bg, border: `1px solid ${m.border}`, display: 'inline-block',
                  }}>
                    {m.groupTitle}（{list.length}）
                  </div>
                  {list.map(renderPromptCard)}
                </div>
              )
            })}

            {/* 底部说明 */}
            <div style={{
              marginTop: '24px', padding: '16px 20px', borderRadius: '12px',
              background: 'rgba(88,86,214,0.04)', border: '1px solid rgba(88,86,214,0.08)',
              fontSize: '12px', lineHeight: '1.8', color: '#8e8e93',
            }}>
              <strong style={{ color: '#5856d6' }}>使用说明：</strong>
              点击卡片展开编辑，保存会创建新版本（旧版本自动归档，可随时回滚）。
              <span style={{ color: '#c62828', fontWeight: 600 }}>🔴 高危</span>提示词修改需二次键入「确认」，请谨慎操作。
              提示词按业务危险度分档展示，内容在对应业务链路执行时被调用。
            </div>
          </>
        )}
      </div>
    </div>
  )
}
