/**
 * 提示词管理页面（P2-3）
 * - 8个槽位的提示词列表（卡片式展示）
 * - 展开编辑：全屏文本编辑区 + 保存（创建新版本）
 * - 版本历史：查看/回滚到历史版本
 * - Apple 风格：毛玻璃卡片 + 渐变 + 圆角 + 微动效
 * - 仅 admin 可访问
 */
import { useState, useEffect, useCallback } from 'react'
import {
  Save, RotateCcw, ChevronDown, ChevronUp,
  Clock, Hash, CheckCircle, AlertCircle, History,
} from 'lucide-react'
import {
  getPrompts, updatePrompt, getPromptVersions, rollbackPromptVersion,
} from '@/api/prompts'
import type {
  PromptInfo, PromptVersion,
} from '@/api/prompts'

// ==================== 提示词用途说明映射 ====================
const PROMPT_DESCRIPTIONS: Record<string, string> = {
  prompt_a: 'K12课程定位：156门课程体系 + 能力定位表 + 学段标准（约15K字）',
  prompt_b: '4维度评估：E1难度适配 + E2时间节奏 + E3互动评估 + E4课程设计（约12K字）',
  prompt_c: '索引差异→逐页修改指令，零编码泄露（约8K字）',
  prompt_d: '一致性 + 质量双层检查（约8K字）',
  prompt_e: 'N轮交叉比对 + 修改方案 + 优化索引（约12K字）',
  prompt_f: 'HTML最小侵入修改 + 5种op分流（约2K字）',
  dict: 'TE-DNA编码格式解压缩速查表（约3K字）',
  ability_table: '156门课×能力等级对照表，Tab分隔（约20K字）',
  prompt_g: '验收用索引压缩：HTML→课程页面索引+模块索引（约17K字）',
}

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
      animation: 'slideIn 0.3s ease',
      display: 'flex', alignItems: 'center', gap: '8px',
    }}>
      {type === 'success' ? <CheckCircle size={16} /> : <AlertCircle size={16} />}
      {message}
    </div>
  )
}

// ==================== 主页面组件 ====================
export default function PromptsPage() {
  // 状态管理
  const [prompts, setPrompts] = useState<PromptInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedKey, setExpandedKey] = useState<string | null>(null)
  const [editContent, setEditContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null)

  // 版本历史相关
  const [showVersions, setShowVersions] = useState<string | null>(null)
  const [versions, setVersions] = useState<PromptVersion[]>([])
  const [versionsLoading, setVersionsLoading] = useState(false)
  const [rollingBack, setRollingBack] = useState(false)

  // 版本内容预览
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

  // 保存提示词（创建新版本）
  const handleSave = async (key: string) => {
    if (!editContent.trim()) {
      setToast({ message: '提示词内容不能为空', type: 'error' })
      return
    }
    try {
      setSaving(true)
      await updatePrompt(key, { content: editContent })
      setToast({ message: '提示词已保存（新版本已创建）', type: 'success' })
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
    backdropFilter: 'blur(20px)',
    WebkitBackdropFilter: 'blur(20px)',
    borderRadius: '16px',
    border: '1px solid rgba(0,0,0,0.06)',
    boxShadow: '0 2px 12px rgba(0,0,0,0.04)',
    marginBottom: '12px',
    overflow: 'hidden',
    transition: 'all 0.2s ease',
  }

  const headerStyle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
    padding: '18px 24px', cursor: 'pointer',
    transition: 'background 0.15s ease',
  }

  const btnPrimary: React.CSSProperties = {
    padding: '10px 24px', borderRadius: '10px', border: 'none', cursor: 'pointer',
    fontSize: '14px', fontWeight: 600, color: '#fff',
    background: 'linear-gradient(135deg, #007aff, #5856d6)',
    boxShadow: '0 2px 8px rgba(0,122,255,0.3)',
    transition: 'all 0.2s ease',
  }

  const btnSecondary: React.CSSProperties = {
    padding: '10px 24px', borderRadius: '10px', border: '1px solid #d1d1d6',
    cursor: 'pointer', fontSize: '14px', fontWeight: 500,
    color: '#1d1d1f', background: '#fff',
    transition: 'all 0.2s ease',
  }

  const btnWarning: React.CSSProperties = {
    padding: '6px 14px', borderRadius: '8px', border: 'none', cursor: 'pointer',
    fontSize: '12px', fontWeight: 600, color: '#fff',
    background: 'linear-gradient(135deg, #ff9500, #ff6b00)',
    boxShadow: '0 2px 6px rgba(255,149,0,0.3)',
    transition: 'all 0.2s ease',
    display: 'inline-flex', alignItems: 'center', gap: '4px',
  }

  // ==================== 渲染 ====================

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '400px' }}>
        <div style={{
          width: '32px', height: '32px',
          border: '2px solid #007aff', borderTopColor: 'transparent',
          borderRadius: '50%', animation: 'spin 0.8s linear infinite',
        }} />
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 'none' }}>
      {/* Toast */}
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}

      {/* 页面描述（标题由MainLayout header提供） */}
      <p style={{ fontSize: '14px', color: '#8e8e93', margin: '0 0 20px 0' }}>
        管理 Pipeline 各槽位的提示词内容，支持版本历史和回滚
      </p>

      {/* 统计卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px', marginBottom: '24px' }}>
        <div style={{ ...cardStyle, padding: '18px 20px', marginBottom: 0 }}>
          <div style={{ fontSize: '12px', color: '#8e8e93', marginBottom: '6px' }}>提示词槽位</div>
          <div style={{ fontSize: '28px', fontWeight: 700, color: '#1d1d1f' }}>{prompts.length}</div>
        </div>
        <div style={{ ...cardStyle, padding: '18px 20px', marginBottom: 0 }}>
          <div style={{ fontSize: '12px', color: '#8e8e93', marginBottom: '6px' }}>已填入内容</div>
          <div style={{ fontSize: '28px', fontWeight: 700, color: '#34c759' }}>
            {prompts.filter(p => p.content_len > 50).length}
          </div>
        </div>
        <div style={{ ...cardStyle, padding: '18px 20px', marginBottom: 0 }}>
          <div style={{ fontSize: '12px', color: '#8e8e93', marginBottom: '6px' }}>待填入</div>
          <div style={{ fontSize: '28px', fontWeight: 700, color: '#ff9500' }}>
            {prompts.filter(p => p.content_len <= 50).length}
          </div>
        </div>
      </div>

      {/* 提示词卡片列表 */}
      {prompts.map((prompt) => {
        const isExpanded = expandedKey === prompt.prompt_key
        const isVersionsOpen = showVersions === prompt.prompt_key
        const isFilled = prompt.content_len > 50

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
                {/* 状态指示灯 */}
                <div style={{
                  width: '10px', height: '10px', borderRadius: '50%',
                  background: isFilled ? '#34c759' : '#ff9500',
                  boxShadow: isFilled ? '0 0 6px rgba(52,199,89,0.4)' : '0 0 6px rgba(255,149,0,0.4)',
                  flexShrink: 0,
                }} />
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <span style={{ fontSize: '15px', fontWeight: 600, color: '#1d1d1f' }}>
                      {prompt.prompt_name}
                    </span>
                    <span style={{
                      fontSize: '11px', padding: '2px 8px', borderRadius: '6px',
                      background: isFilled ? '#e8f5e9' : '#fff3e0',
                      color: isFilled ? '#2e7d32' : '#e65100',
                      fontWeight: 500,
                    }}>
                      {isFilled ? `v${prompt.version} · ${formatLen(prompt.content_len)}字` : '待填入'}
                    </span>
                  </div>
                  <div style={{ fontSize: '12px', color: '#8e8e93', marginTop: '4px' }}>
                    {PROMPT_DESCRIPTIONS[prompt.prompt_key] || ''}
                  </div>
                </div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                {/* 版本历史按钮 */}
                <button
                  onClick={(e) => { e.stopPropagation(); handleShowVersions(prompt.prompt_key) }}
                  style={{
                    padding: '6px 12px', borderRadius: '8px', border: '1px solid #d1d1d6',
                    cursor: 'pointer', fontSize: '12px', fontWeight: 500,
                    color: isVersionsOpen ? '#5856d6' : '#8e8e93',
                    background: isVersionsOpen ? '#f0efff' : '#fff',
                    display: 'flex', alignItems: 'center', gap: '4px',
                    transition: 'all 0.15s ease',
                  }}
                >
                  <History size={13} />
                  版本
                </button>
                {isExpanded ? <ChevronUp size={18} color="#8e8e93" /> : <ChevronDown size={18} color="#8e8e93" />}
              </div>
            </div>

            {/* 展开的编辑区域 */}
            {isExpanded && (
              <div style={{ padding: '0 24px 20px', borderTop: '1px solid rgba(0,0,0,0.04)' }}>
                <div style={{ paddingTop: '16px' }}>
                  {/* 编辑提示 */}
                  <div style={{ fontSize: '12px', color: '#8e8e93', marginBottom: '10px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <Hash size={12} />
                    编辑后保存将创建新版本（v{prompt.version + 1}），旧版本自动归档
                  </div>
                  {/* 文本编辑区 */}
                  <textarea
                    value={editContent}
                    onChange={(e) => setEditContent(e.target.value)}
                    placeholder="请输入提示词完整内容..."
                    style={{
                      width: '100%', minHeight: '320px', padding: '16px',
                      borderRadius: '12px', border: '1px solid #d1d1d6',
                      fontSize: '13px', lineHeight: '1.7', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                      resize: 'vertical', outline: 'none',
                      background: '#fafafa', color: '#1d1d1f',
                      transition: 'border-color 0.2s ease, box-shadow 0.2s ease',
                      boxSizing: 'border-box',
                    }}
                    onFocus={(e) => {
                      e.target.style.borderColor = '#007aff'
                      e.target.style.boxShadow = '0 0 0 3px rgba(0,122,255,0.1)'
                    }}
                    onBlur={(e) => {
                      e.target.style.borderColor = '#d1d1d6'
                      e.target.style.boxShadow = 'none'
                    }}
                  />
                  {/* 底部操作栏 */}
                  <div style={{
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                    marginTop: '14px',
                  }}>
                    <span style={{ fontSize: '12px', color: '#8e8e93' }}>
                      当前字数：{editContent.length.toLocaleString()} 字
                    </span>
                    <div style={{ display: 'flex', gap: '10px' }}>
                      <button
                        onClick={() => { setExpandedKey(null); setEditContent('') }}
                        style={btnSecondary}
                      >
                        取消
                      </button>
                      <button
                        onClick={() => handleSave(prompt.prompt_key)}
                        disabled={saving}
                        style={{
                          ...btnPrimary,
                          opacity: saving ? 0.6 : 1,
                          cursor: saving ? 'not-allowed' : 'pointer',
                          display: 'flex', alignItems: 'center', gap: '6px',
                        }}
                      >
                        <Save size={14} />
                        {saving ? '保存中...' : '保存新版本'}
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
                    <Clock size={14} />
                    版本历史 — {prompt.prompt_name}
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
                          transition: 'all 0.15s ease',
                        }}>
                          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                              <span style={{
                                fontSize: '13px', fontWeight: 600,
                                color: v.is_current ? '#007aff' : '#1d1d1f',
                              }}>
                                v{v.version}
                              </span>
                              {v.is_current && (
                                <span style={{
                                  fontSize: '11px', padding: '1px 8px', borderRadius: '4px',
                                  background: '#007aff', color: '#fff', fontWeight: 500,
                                }}>
                                  当前
                                </span>
                              )}
                              <span style={{ fontSize: '12px', color: '#8e8e93' }}>
                                {formatLen(v.content_len)}字 · {formatTime(v.created_at)}
                              </span>
                            </div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                              {/* 预览按钮 */}
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
                              {/* 回滚按钮（非当前版本才显示） */}
                              {!v.is_current && (
                                <button
                                  onClick={() => handleRollback(prompt.prompt_key, v.id, v.version)}
                                  disabled={rollingBack}
                                  style={{
                                    ...btnWarning,
                                    opacity: rollingBack ? 0.6 : 1,
                                    cursor: rollingBack ? 'not-allowed' : 'pointer',
                                  }}
                                >
                                  <RotateCcw size={11} />
                                  回滚
                                </button>
                              )}
                            </div>
                          </div>
                          {/* 版本内容预览 */}
                          {previewVersion?.id === v.id && (
                            <div style={{
                              marginTop: '10px', padding: '12px',
                              background: '#fff', borderRadius: '8px', border: '1px solid #eee',
                              maxHeight: '200px', overflowY: 'auto',
                              fontSize: '12px', lineHeight: '1.6',
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
      })}

      {/* 底部说明 */}
      <div style={{
        marginTop: '24px', padding: '16px 20px', borderRadius: '12px',
        background: 'rgba(88,86,214,0.04)', border: '1px solid rgba(88,86,214,0.08)',
        fontSize: '12px', lineHeight: '1.8', color: '#8e8e93',
      }}>
        <strong style={{ color: '#5856d6' }}>使用说明：</strong>
        点击提示词卡片展开编辑区域，修改后保存会自动创建新版本。
        点击"版本"按钮查看历史版本，支持预览和回滚到任意历史版本。
        提示词内容将在 Pipeline 执行时被对应步骤调用。
      </div>
    </div>
  )
}
