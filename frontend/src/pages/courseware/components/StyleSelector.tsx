/**
 * 课件风格选择器 — StyleSelector.tsx v2.0 (2026-05-19)
 *
 * v2.0 改造：
 *   - 引入共享 TemplateThumb 组件：
 *     - 自动检测内容尺寸（系统 960×540 vs 个人 1920×1080）
 *     - ResizeObserver 动态算 scale 消除右下留白
 *     - wrapHTML 强制 overflow:hidden 杜绝滚动条
 *   - 弹窗预览改用 TemplateThumbAuto 自适应宽度
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import {
  getCWTemplatesWithUser, deleteMyTemplate, uploadCWLogo, saveStyleFull, confirmCWStyle,
  CW_STYLE_CONFIG,
} from '@/api/coursewares'
import type { CoursewareTemplate, CoursewareDetail } from '@/api/coursewares'
import TemplateThumb, { TemplateThumbAuto } from './TemplateThumb'

// ==================== 颜色常量 ====================
const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB', success: '#059669', white: '#fff', danger: '#EF4444',
}

// ==================== 辅助函数 ====================
const safeParse = (s: string): Record<string, string> => {
  try { return JSON.parse(s) || {} } catch { return {} }
}
const safeParseArray = (s: string): string[] => {
  try { const a = JSON.parse(s); return Array.isArray(a) ? a : [] } catch { return [] }
}

// ==================== Props ====================
interface StyleSelectorProps {
  courseware: CoursewareDetail
  coursewareId: string
  onStyleConfirmed: () => void
}

export default function StyleSelector({ courseware, coursewareId, onStyleConfirmed }: StyleSelectorProps) {
  const [templates, setTemplates] = useState<CoursewareTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedTplId, setSelectedTplId] = useState('')
  const [styleFilter, setStyleFilter] = useState('')
  const [logoURL, setLogoURL] = useState(courseware.logo_url || '')
  const [orgName, setOrgName] = useState(courseware.org_name || '')
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [saving, setSaving] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [saved, setSaved] = useState(false)
  const [previewTpl, setPreviewTpl] = useState<CoursewareTemplate | null>(null)

  // ==================== 加载模板 ====================
  const loadTemplates = useCallback(async () => {
    setLoading(true)
    try {
      const list = await getCWTemplatesWithUser()
      setTemplates(list || [])
      if (courseware.style_config) {
        try {
          const cfg = JSON.parse(courseware.style_config)
          if (cfg.template_id) setSelectedTplId(cfg.template_id)
        } catch { /* 忽略 */ }
      }
    } catch { /* 忽略 */ } finally { setLoading(false) }
  }, [courseware.style_config])

  useEffect(() => { loadTemplates() }, [loadTemplates])

  // ==================== Logo上传 ====================
  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (file.size > 2 * 1024 * 1024) { alert('Logo文件不能超过2MB'); return }
    setUploading(true)
    try {
      const result = await uploadCWLogo(coursewareId, file)
      setLogoURL(result.url)
      setSaved(false)
    } catch (err) {
      alert('Logo上传失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  // ==================== 保存风格 ====================
  const handleSave = async () => {
    if (!selectedTplId) { alert('请先选择一个风格模板'); return }
    setSaving(true)
    try {
      await saveStyleFull(coursewareId, { template_id: selectedTplId, logo_url: logoURL, org_name: orgName })
      setSaved(true)
    } catch { alert('保存失败') } finally { setSaving(false) }
  }

  // ==================== 确认风格 ====================
  const handleConfirm = async () => {
    if (!saved) {
      if (!selectedTplId) { alert('请先选择一个风格模板'); return }
      setSaving(true)
      try {
        await saveStyleFull(coursewareId, { template_id: selectedTplId, logo_url: logoURL, org_name: orgName })
        setSaved(true)
      } catch { alert('保存失败'); setSaving(false); return } finally { setSaving(false) }
    }
    setConfirming(true)
    try {
      await confirmCWStyle(coursewareId)
      onStyleConfirmed()
    } catch (err) {
      alert('确认失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally { setConfirming(false) }
  }

  // ==================== 筛选+分组 ====================
  const filtered = styleFilter ? templates.filter(t => t.style_category === styleFilter) : templates
  const systemTemplates = filtered.filter(t => !t.scope || t.scope === 'system')
  const personalTemplates = filtered.filter(t => t.scope === 'personal')
  const styleFilters = [
    { value: '', label: '全部风格' },
    ...Object.entries(CW_STYLE_CONFIG).map(([k, v]) => ({ value: k, label: v.emoji + ' ' + v.label })),
  ]
  const selectedTpl = templates.find(t => t.id === selectedTplId)

  // ==================== 卡片缩略图 fallback（兜底渲染） ====================
  const buildFallback = (t: CoursewareTemplate) => {
    const colors = safeParse(t.color_scheme)
    const sc = CW_STYLE_CONFIG[t.style_category] || { emoji: '🎨' }
    return (
      <div style={{
        height: '100%',
        background: colors.primary ? `linear-gradient(135deg, ${colors.primary}30, ${colors.secondary || colors.primary}60)` : '#E5E7EB',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
        <span style={{ fontSize: '40px' }}>{sc.emoji}</span>
      </div>
    )
  }

  return (
    <div>
      {/* ====== 区块1：机构品牌配置 ====== */}
      <div style={{ background: 'linear-gradient(135deg, rgba(245,158,11,0.04), rgba(139,92,246,0.04))', borderRadius: '14px', border: `1px solid ${C.border}`, padding: '20px 24px', marginBottom: '24px' }}>
        <div style={{ fontSize: '15px', fontWeight: 700, color: C.textPrimary, marginBottom: '16px' }}>🏛️ 机构品牌（可选）</div>
        <div style={{ display: 'flex', gap: '24px', alignItems: 'flex-start', flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' }}>
            <div onClick={() => fileInputRef.current?.click()} style={{ width: '80px', height: '80px', borderRadius: '14px', border: `2px dashed ${logoURL ? 'transparent' : '#D1D5DB'}`, background: logoURL ? 'transparent' : '#F9FAFB', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', overflow: 'hidden', transition: 'all 200ms' }}>
              {logoURL ? (
                <img src={logoURL} alt="Logo" style={{ width: '100%', height: '100%', objectFit: 'contain' }} />
              ) : (
                <span style={{ fontSize: '28px', color: '#D1D5DB' }}>{uploading ? '⏳' : '➕'}</span>
              )}
            </div>
            <input ref={fileInputRef} type="file" accept="image/*" onChange={handleLogoUpload} style={{ display: 'none' }} />
            <span style={{ fontSize: '11px', color: C.textMuted }}>{uploading ? '上传中...' : logoURL ? '点击更换' : '上传Logo'}</span>
          </div>
          <div style={{ flex: 1, minWidth: '200px' }}>
            <label style={{ fontSize: '13px', fontWeight: 600, color: C.textSecondary, display: 'block', marginBottom: '6px' }}>机构名称</label>
            <input value={orgName} onChange={e => { setOrgName(e.target.value); setSaved(false) }} placeholder="如：北京大学教育学院" style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none' }} />
            <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>将显示在课件封面和页脚</div>
          </div>
        </div>
      </div>

      {/* ====== 区块2：风格方案选择 ====== */}
      <div style={{ marginBottom: '20px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <div style={{ fontSize: '15px', fontWeight: 700, color: C.textPrimary }}>🎨 选择风格方案</div>
          <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
            {styleFilters.map(f => (
              <button key={f.value} onClick={() => setStyleFilter(f.value)} style={{ padding: '4px 12px', borderRadius: '16px', fontSize: '12px', cursor: 'pointer', border: `1px solid ${styleFilter === f.value ? C.primary : C.border}`, background: styleFilter === f.value ? C.primaryBg : 'transparent', color: styleFilter === f.value ? C.primary : C.textSecondary, fontWeight: styleFilter === f.value ? 600 : 400 }}>{f.label}</button>
            ))}
          </div>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted }}>加载模板中...</div>
        ) : filtered.length === 0 && personalTemplates.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted }}>
            <div style={{ fontSize: '36px', marginBottom: '8px' }}>🎨</div>暂无此类风格模板
          </div>
        ) : (
          <>
          {/* 个人模板分组（v137） */}
          {personalTemplates.length > 0 && (
            <div style={{ marginBottom: '20px' }}>
              <div style={{ fontSize: '14px', fontWeight: 700, color: '#059669', marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span>💾 我的模板</span>
                <span style={{ fontSize: '11px', fontWeight: 400, color: '#9CA3AF' }}>从课件保存的个人模板，可直接复用</span>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '16px' }}>
                {personalTemplates.map(t => {
                  const isSelected = t.id === selectedTplId
                  const sc = CW_STYLE_CONFIG[t.style_category] || { label: t.style_category, color: '#6B7280', bg: '#F3F4F6', emoji: '🎨' }
                  const colors = safeParse(t.color_scheme)
                  const previewUrls = safeParseArray(t.preview_urls)
                  const pages = safeParseArray(t.sample_pages)
                  return (
                    <div key={t.id} onClick={() => { setSelectedTplId(t.id); setSaved(false) }} style={{ borderRadius: '14px', overflow: 'hidden', cursor: 'pointer', border: `2px solid ${isSelected ? '#059669' : C.border}`, boxShadow: isSelected ? '0 4px 20px rgba(5,150,105,0.2)' : '0 1px 4px rgba(0,0,0,0.04)', transition: 'all 200ms', transform: isSelected ? 'translateY(-2px)' : 'none', background: C.white }}>
                      <div style={{ position: 'relative' }}>
                        <TemplateThumb
                          previewUrl={previewUrls[0]}
                          sampleHTML={pages[0]}
                          fallback={buildFallback(t)}
                          height={160}
                          title={t.name}
                        />
                        {isSelected && <div style={{ position: 'absolute', top: '8px', right: '8px', width: '28px', height: '28px', borderRadius: '50%', background: '#059669', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '16px', fontWeight: 700, boxShadow: '0 2px 8px rgba(5,150,105,0.3)', zIndex: 2 }}>✓</div>}
                        <div style={{ position: 'absolute', top: '8px', left: '8px', padding: '2px 8px', borderRadius: '6px', background: 'rgba(5,150,105,0.9)', color: '#fff', fontSize: '10px', fontWeight: 700, zIndex: 2 }}>我的</div>
                        {colors.primary && (
                          <div style={{ position: 'absolute', bottom: '8px', right: '8px', display: 'flex', gap: '3px', background: 'rgba(255,255,255,0.85)', borderRadius: '10px', padding: '3px 5px', zIndex: 2 }}>
                            {['primary', 'secondary', 'accent'].map(k => colors[k] ? <div key={k} style={{ width: '12px', height: '12px', borderRadius: '50%', background: colors[k], border: '1.5px solid rgba(255,255,255,0.8)' }} /> : null)}
                          </div>
                        )}
                      </div>
                      <div style={{ padding: '12px 16px' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                          <span style={{ fontSize: '14px', fontWeight: 700, color: C.textPrimary }}>{t.name}</span>
                          <span style={{ padding: '1px 8px', borderRadius: '10px', fontSize: '10px', fontWeight: 600, color: sc.color, background: sc.bg }}>{sc.label}</span>
                        </div>
                        {t.description && <div style={{ fontSize: '12px', color: C.textSecondary, lineHeight: 1.4 }}>{t.description.length > 40 ? t.description.slice(0, 40) + '...' : t.description}</div>}
                        <div style={{ marginTop: '8px', display: 'flex', gap: '6px' }}>
                          <button onClick={(e) => { e.stopPropagation(); setPreviewTpl(t) }} style={{ padding: '3px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: '11px', cursor: 'pointer' }}>🔍 预览</button>
                          <button onClick={async (e) => { e.stopPropagation(); if (!window.confirm('删除个人模板「' + t.name + '」？')) return; try { await deleteMyTemplate(t.id); const list = await getCWTemplatesWithUser(); setTemplates(list) } catch { alert('删除失败') } }} style={{ padding: '3px 10px', borderRadius: '6px', border: '1px solid #FCA5A5', background: 'transparent', color: '#EF4444', fontSize: '11px', cursor: 'pointer' }}>🗑️ 删除</button>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
          {/* 系统模板分组 */}
          {personalTemplates.length > 0 && systemTemplates.length > 0 && (
            <div style={{ fontSize: '14px', fontWeight: 700, color: '#1F2937', marginBottom: '12px' }}>🎨 系统模板</div>
          )}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '16px' }}>
            {systemTemplates.map(t => {
              const isSelected = t.id === selectedTplId
              const sc = CW_STYLE_CONFIG[t.style_category] || { label: t.style_category, color: '#6B7280', bg: '#F3F4F6', emoji: '🎨' }
              const colors = safeParse(t.color_scheme)
              const previewUrls = safeParseArray(t.preview_urls)
              const pages = safeParseArray(t.sample_pages)
              return (
                <div key={t.id} onClick={() => { setSelectedTplId(t.id); setSaved(false) }} style={{ borderRadius: '14px', overflow: 'hidden', cursor: 'pointer', border: `2px solid ${isSelected ? C.primary : C.border}`, boxShadow: isSelected ? '0 4px 20px rgba(245,158,11,0.2)' : '0 1px 4px rgba(0,0,0,0.04)', transition: 'all 200ms', transform: isSelected ? 'translateY(-2px)' : 'none', background: C.white }}>
                  <div style={{ position: 'relative' }}>
                    <TemplateThumb
                      previewUrl={previewUrls[0]}
                      sampleHTML={pages[0]}
                      fallback={buildFallback(t)}
                      height={160}
                      title={t.name}
                    />
                    {isSelected && <div style={{ position: 'absolute', top: '8px', right: '8px', width: '28px', height: '28px', borderRadius: '50%', background: C.primary, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '16px', fontWeight: 700, boxShadow: '0 2px 8px rgba(245,158,11,0.3)', zIndex: 2 }}>✓</div>}
                    {t.style_category === 'immersive' && <div style={{ position: 'absolute', top: '8px', left: '8px', padding: '2px 8px', borderRadius: '6px', background: 'rgba(220,38,38,0.9)', color: '#fff', fontSize: '10px', fontWeight: 700, zIndex: 2 }}>3D</div>}
                    {colors.primary && (
                      <div style={{ position: 'absolute', bottom: '8px', right: '8px', display: 'flex', gap: '3px', background: 'rgba(255,255,255,0.85)', borderRadius: '10px', padding: '3px 5px', zIndex: 2 }}>
                        {['primary', 'secondary', 'accent'].map(k => colors[k] ? <div key={k} style={{ width: '12px', height: '12px', borderRadius: '50%', background: colors[k], border: '1.5px solid rgba(255,255,255,0.8)' }} /> : null)}
                      </div>
                    )}
                  </div>
                  <div style={{ padding: '12px 16px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                      <span style={{ fontSize: '14px', fontWeight: 700, color: C.textPrimary }}>{t.name}</span>
                      <span style={{ padding: '1px 8px', borderRadius: '10px', fontSize: '10px', fontWeight: 600, color: sc.color, background: sc.bg }}>{sc.label}</span>
                    </div>
                    {t.description && <div style={{ fontSize: '12px', color: C.textSecondary, lineHeight: 1.4 }}>{t.description.length > 40 ? t.description.slice(0, 40) + '...' : t.description}</div>}
                    <button onClick={(e) => { e.stopPropagation(); setPreviewTpl(t) }} style={{ marginTop: '8px', padding: '3px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: '11px', cursor: 'pointer' }}>🔍 预览详情</button>
                  </div>
                </div>
              )
            })}
          </div>
          </>
        )}
      </div>

      {/* ====== 区块3：底部操作栏 ====== */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px 20px', borderRadius: '12px', background: selectedTplId ? 'linear-gradient(135deg, rgba(245,158,11,0.06), rgba(139,92,246,0.04))' : '#F9FAFB', border: `1px solid ${selectedTplId ? 'rgba(245,158,11,0.2)' : C.border}` }}>
        <div>
          {selectedTpl ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <span style={{ fontSize: '13px', color: C.textSecondary }}>已选择：</span>
              <span style={{ fontSize: '14px', fontWeight: 700, color: C.primary }}>{CW_STYLE_CONFIG[selectedTpl.style_category]?.emoji} {selectedTpl.name}</span>
              {saved && <span style={{ fontSize: '11px', color: C.success }}>✅ 已保存</span>}
            </div>
          ) : (
            <span style={{ fontSize: '13px', color: C.textMuted }}>请从上方选择一个风格方案</span>
          )}
        </div>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button onClick={handleSave} disabled={!selectedTplId || saving} style={{ padding: '8px 20px', borderRadius: '8px', border: `1px solid ${selectedTplId ? C.primary : C.border}`, background: 'transparent', color: selectedTplId ? C.primary : C.textMuted, fontSize: '14px', fontWeight: 600, cursor: selectedTplId && !saving ? 'pointer' : 'default', opacity: selectedTplId ? 1 : 0.5 }}>{saving ? '保存中...' : '💾 保存风格'}</button>
          <button onClick={handleConfirm} disabled={!selectedTplId || confirming} style={{ padding: '8px 24px', borderRadius: '8px', border: 'none', background: selectedTplId ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB', color: selectedTplId ? '#fff' : '#9CA3AF', fontSize: '14px', fontWeight: 600, cursor: selectedTplId && !confirming ? 'pointer' : 'default', boxShadow: selectedTplId ? '0 2px 8px rgba(245,158,11,0.3)' : 'none' }}>{confirming ? '确认中...' : '确认风格，下一步 →'}</button>
        </div>
      </div>

      {/* ====== 预览弹窗（v2.0: 用 TemplateThumbAuto） ====== */}
      {previewTpl && (
        <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.6)', zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center', backdropFilter: 'blur(4px)' }} onClick={() => setPreviewTpl(null)}>
          <div style={{ background: '#fff', borderRadius: '20px', width: '95%', maxWidth: '1080px', maxHeight: '92vh', overflow: 'auto' }} onClick={e => e.stopPropagation()}>
            <div style={{ padding: '20px 28px 12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <div style={{ fontSize: '20px', fontWeight: 800, color: C.textPrimary }}>{CW_STYLE_CONFIG[previewTpl.style_category]?.emoji} {previewTpl.name}</div>
                {previewTpl.description && <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '4px' }}>{previewTpl.description}</div>}
              </div>
              <button onClick={() => setPreviewTpl(null)} style={{ background: 'none', border: 'none', fontSize: '28px', cursor: 'pointer', color: C.textMuted, lineHeight: 1 }}>×</button>
            </div>
            {/* 预览区域 — 用共享组件 */}
            <div style={{ padding: '0 28px 20px' }}>
              <TemplateThumbAuto
                previewUrl={safeParseArray(previewTpl.preview_urls)[0]}
                sampleHTML={safeParseArray(previewTpl.sample_pages)[0]}
                title="模板预览"
              />
            </div>
            {/* 配色+选择按钮 */}
            <div style={{ padding: '0 28px 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: '24px' }}>
              <div>
                <div style={{ fontSize: '14px', fontWeight: 700, color: C.textPrimary, marginBottom: '10px' }}>🎨 配色方案</div>
                <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
                  {Object.entries(safeParse(previewTpl.color_scheme)).map(([k, v]) => (
                    <div key={k} style={{ textAlign: 'center' }}>
                      <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: v, border: '2px solid rgba(0,0,0,0.06)', marginBottom: '3px', boxShadow: '0 2px 6px rgba(0,0,0,0.08)' }} />
                      <div style={{ fontSize: '9px', color: C.textMuted }}>{k}</div>
                    </div>
                  ))}
                </div>
              </div>
              <button onClick={() => { setSelectedTplId(previewTpl.id); setSaved(false); setPreviewTpl(null) }} style={{ padding: '12px 36px', borderRadius: '12px', border: 'none', background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff', fontSize: '15px', fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(245,158,11,0.3)', flexShrink: 0 }}>选择此风格</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
