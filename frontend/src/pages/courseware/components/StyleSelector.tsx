/**
 * 课件风格选择器 — StyleSelector.tsx v2.3 (2026-07-02)
 *
 * v2.3 改动（BugFix：学校/教研组模板在"选风格"步骤不显示）:
 *   - 后端 GET /courseware-templates/with-user 早已返回 4 级 scope
 *     (system/school/group/personal) 的联合结果，但本组件原先只按
 *     personal / system 两组过滤渲染，scope=school / scope=group 的模板
 *     被整体过滤掉——校管/组长发布的"学校模板/教研组模板"在 Step2 选风格时
 *     永远找不到（风格模板管理页能看到、这里看不到，正是"发布成功却选不到"的根因）。
 *   - 现按 4 组分区渲染：💾我的模板 → 👥教研组模板 → 🏫学校模板 → 🎨系统模板；
 *     school/group 卡片带对应归属徽章；删除按钮仍仅个人模板显示
 *     （学校/教研组模板的撤回与管理走"风格模板"页，不在选择器里做）。
 *   - 四组卡片 JSX 抽出为内部 TplCard 组件消除重复，行为与原先逐行一致
 *     （TemplateThumb 缩略图 / 配色圆点 / 3D 角标 / 预览弹窗全部保留）。
 *   - 预览弹窗：本组件自建遮罩+面板外壳，内部复用 TemplatePagesPreview
 *     （其 props 为 previewUrls/samplePages/accentColor，非整模板对象）。
 *
 * v2.2 改动（配合 TemplateThumb 恢复纯 v2.0)：
 *   - 移除 v2.1 给 TemplateThumb 传的 index prop（渐进挂载方案已整体放弃，回归一次性渲染）。
 *
 * v2.0 改造：
 *   - 引入共享 TemplateThumb 组件：自动检测内容尺寸、ResizeObserver 动态算 scale、wrapHTML 消滚动条
 *   - 弹窗预览改用 TemplateThumbAuto 自适应宽度
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import {
  getCWTemplatesWithUser, deleteMyTemplate, uploadCWLogo, saveStyleFull, confirmCWStyle, getLogoHistory, deleteLogoHistory,
  CW_STYLE_CONFIG,
} from '@/api/coursewares'
import type { CoursewareTemplate, CoursewareDetail } from '@/api/coursewares'
import TemplateThumb from './TemplateThumb'
import TemplatePagesPreview from './TemplatePagesPreview'

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

// 卡片缩略图 fallback（无预览URL、无样例HTML时的兜底渲染；纯静态不依赖组件状态，提到模块级）
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

// ==================== 模板卡片（v2.3 抽出，四个分组共用） ====================
interface TplCardProps {
  t: CoursewareTemplate
  isSelected: boolean
  /** 该分组的强调色（选中边框/对勾底色） */
  accent: string
  /** 左上角归属徽章文字（空串 = 不显示；系统模板不带归属徽章） */
  badgeText: string
  /** 归属徽章底色 */
  badgeBg: string
  /** 是否显示删除按钮（仅个人模板；学校/教研组模板的管理走风格模板页） */
  canDelete: boolean
  onSelect: () => void
  onPreview: () => void
  onDelete: () => void
}

function TplCard({ t, isSelected, accent, badgeText, badgeBg, canDelete, onSelect, onPreview, onDelete }: TplCardProps) {
  const sc = CW_STYLE_CONFIG[t.style_category] || { label: t.style_category, color: '#6B7280', bg: '#F3F4F6', emoji: '🎨' }
  const colors = safeParse(t.color_scheme)
  const previewUrls = safeParseArray(t.preview_urls)
  const pages = safeParseArray(t.sample_pages)
  return (
    <div onClick={onSelect} style={{ borderRadius: '14px', overflow: 'hidden', cursor: 'pointer', border: `2px solid ${isSelected ? accent : C.border}`, boxShadow: isSelected ? `0 4px 20px ${accent}33` : '0 1px 4px rgba(0,0,0,0.04)', transition: 'all 200ms', transform: isSelected ? 'translateY(-2px)' : 'none', background: C.white }}>
      <div style={{ position: 'relative' }}>
        <TemplateThumb
          previewUrl={previewUrls[0]}
          sampleHTML={pages[0]}
          fallback={buildFallback(t)}
          height={160}
          title={t.name}
        />
        {isSelected && <div style={{ position: 'absolute', top: '8px', right: '8px', width: '28px', height: '28px', borderRadius: '50%', background: accent, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '16px', fontWeight: 700, boxShadow: `0 2px 8px ${accent}4D`, zIndex: 2 }}>✓</div>}
        {/* 左上角徽章区：归属徽章(我的/教研组/学校) 与 3D角标(immersive) 可并存 */}
        {(badgeText || t.style_category === 'immersive') && (
          <div style={{ position: 'absolute', top: '8px', left: '8px', display: 'flex', gap: '4px', zIndex: 2 }}>
            {badgeText && <div style={{ padding: '2px 8px', borderRadius: '6px', background: badgeBg, color: '#fff', fontSize: '10px', fontWeight: 700 }}>{badgeText}</div>}
            {t.style_category === 'immersive' && <div style={{ padding: '2px 8px', borderRadius: '6px', background: 'rgba(220,38,38,0.9)', color: '#fff', fontSize: '10px', fontWeight: 700 }}>3D</div>}
          </div>
        )}
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
          <button onClick={(e) => { e.stopPropagation(); onPreview() }} style={{ padding: '3px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: 'transparent', color: C.textSecondary, fontSize: '11px', cursor: 'pointer' }}>🔍 预览</button>
          {canDelete && (
            <button onClick={(e) => { e.stopPropagation(); onDelete() }} style={{ padding: '3px 10px', borderRadius: '6px', border: '1px solid #FCA5A5', background: 'transparent', color: '#EF4444', fontSize: '11px', cursor: 'pointer' }}>🗑️ 删除</button>
          )}
        </div>
      </div>
    </div>
  )
}

// ==================== Props ====================
interface StyleSelectorProps {
  courseware: CoursewareDetail
  coursewareId: string
  onStyleConfirmed: () => void
  /** B方案：Step2内联设锚点成功后通知父级刷新课件（父级传 loadCourseware） */
  onAnchorChanged?: () => void
}

export default function StyleSelector({ courseware, coursewareId, onStyleConfirmed, onAnchorChanged }: StyleSelectorProps) {
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
  const [logoHistory, setLogoHistory] = useState<string[]>([])  // 需求2: 历史 Logo 列表（点击复用免重传）

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

  // 需求2: 加载当前用户历史用过的 Logo（去重、最近优先），供下方历史Logo快速复用
  useEffect(() => {
    getLogoHistory().then(setLogoHistory).catch(() => { /* 历史Logo加载失败不影响主流程 */ })
  }, [])

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

  // 需求2: 删除一条历史 Logo（清空使用它的课件的 logo_url；由用户自行判断）
  const handleDeleteLogo = async (url: string) => {
    if (!window.confirm('确定删除这个历史 Logo 吗？\n使用该 Logo 的课件会一并清空其 Logo（请自行判断）。')) return
    try {
      await deleteLogoHistory(url)
      setLogoHistory(prev => prev.filter(u => u !== url))
      if (logoURL === url) { setLogoURL(''); setSaved(false) }
    } catch (e) {
      alert('删除失败: ' + (e instanceof Error ? e.message : '未知错误'))
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

  // 删除个人模板（仅个人分组卡片可触发）
  const handleDeletePersonal = async (t: CoursewareTemplate) => {
    if (!window.confirm('删除个人模板「' + t.name + '」？')) return
    try {
      await deleteMyTemplate(t.id)
      const list = await getCWTemplatesWithUser()
      setTemplates(list || [])
    } catch { alert('删除失败') }
  }

  // ==================== 筛选 + 4级scope分组（v2.3） ====================
  const filtered = styleFilter ? templates.filter(t => t.style_category === styleFilter) : templates
  const personalTemplates = filtered.filter(t => t.scope === 'personal')
  const groupTemplates = filtered.filter(t => t.scope === 'group')
  const schoolTemplates = filtered.filter(t => t.scope === 'school')
  const systemTemplates = filtered.filter(t => !t.scope || t.scope === 'system')
  const hasNonSystem = personalTemplates.length + groupTemplates.length + schoolTemplates.length > 0

  const styleFilters = [
    { value: '', label: '全部风格' },
    ...Object.entries(CW_STYLE_CONFIG).map(([k, v]) => ({ value: k, label: v.emoji + ' ' + v.label })),
  ]
  const selectedTpl = templates.find(t => t.id === selectedTplId)

  // 分组渲染配置：顺序 我的 → 教研组 → 学校 → 系统（与老师"离自己越近越靠前"的直觉一致）
  const tplGroups = [
    { key: 'personal', list: personalTemplates, title: '💾 我的模板', hint: '从课件保存或自己发布的个人模板，可直接复用', titleColor: '#059669', accent: '#059669', badgeText: '我的', badgeBg: 'rgba(5,150,105,0.9)', canDelete: true },
    { key: 'group', list: groupTemplates, title: '👥 教研组模板', hint: '教研组共享模板，组内老师均可选用', titleColor: '#7C3AED', accent: '#7C3AED', badgeText: '教研组', badgeBg: 'rgba(124,58,237,0.9)', canDelete: false },
    { key: 'school', list: schoolTemplates, title: '🏫 学校模板', hint: '本校共享模板，全校老师均可选用', titleColor: '#2563EB', accent: '#2563EB', badgeText: '学校', badgeBg: 'rgba(37,99,235,0.9)', canDelete: false },
    { key: 'system', list: systemTemplates, title: '🎨 系统模板', hint: '平台内置风格模板', titleColor: '#1F2937', accent: C.primary, badgeText: '', badgeBg: '', canDelete: false },
  ]

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
        {/* 需求2: 历史 Logo 复用 — 点击即选用，免重新上传 */}
        {logoHistory.length > 0 && (
          <div style={{ marginTop: '16px' }}>
            <div style={{ fontSize: '12px', fontWeight: 600, color: C.textSecondary, marginBottom: '8px' }}>🕑 历史 Logo（点击复用，免重新上传）</div>
            <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
              {logoHistory.map(url => {
                const active = url === logoURL
                return (
                  <div key={url} style={{ position: 'relative', width: '56px', height: '56px' }}>
                    <div onClick={() => { setLogoURL(url); setSaved(false) }} title="点击复用此 Logo" style={{ width: '100%', height: '100%', borderRadius: '10px', border: `2px solid ${active ? C.primary : C.border}`, background: '#F9FAFB', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', overflow: 'hidden', boxShadow: active ? '0 2px 8px rgba(245,158,11,0.25)' : 'none' }}>
                      <img src={url} alt="历史Logo" style={{ width: '100%', height: '100%', objectFit: 'contain' }} />
                    </div>
                    {active && <div style={{ position: 'absolute', top: '-4px', left: '-4px', width: '16px', height: '16px', borderRadius: '50%', background: C.primary, color: '#fff', fontSize: '10px', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 2 }}>✓</div>}
                    <button onClick={async (e) => { e.stopPropagation(); await handleDeleteLogo(url) }} title="删除此历史 Logo" style={{ position: 'absolute', top: '-6px', right: '-6px', width: '18px', height: '18px', borderRadius: '50%', border: 'none', background: '#EF4444', color: '#fff', fontSize: '12px', lineHeight: '16px', textAlign: 'center', cursor: 'pointer', padding: 0, zIndex: 3, boxShadow: '0 1px 4px rgba(0,0,0,0.3)' }}>×</button>
                  </div>
                )
              })}
            </div>
          </div>
        )}
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
        ) : filtered.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted }}>
            <div style={{ fontSize: '36px', marginBottom: '8px' }}>🎨</div>暂无此类风格模板
          </div>
        ) : (
          <>
          {/* v2.3: 四级scope分组依次渲染（空分组自动隐藏；系统组仅在有非系统组时显示分组标题） */}
          {tplGroups.map(g => {
            if (g.list.length === 0) return null
            const showHeader = g.key !== 'system' || hasNonSystem
            return (
              <div key={g.key} style={{ marginBottom: '20px' }}>
                {showHeader && (
                  <div style={{ fontSize: '14px', fontWeight: 700, color: g.titleColor, marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span>{g.title}</span>
                    <span style={{ fontSize: '11px', fontWeight: 400, color: '#9CA3AF' }}>{g.hint}</span>
                  </div>
                )}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '16px' }}>
                  {g.list.map(t => (
                    <TplCard
                      key={t.id}
                      t={t}
                      isSelected={t.id === selectedTplId}
                      accent={g.accent}
                      badgeText={g.badgeText}
                      badgeBg={g.badgeBg}
                      canDelete={g.canDelete}
                      onSelect={() => { setSelectedTplId(t.id); setSaved(false) }}
                      onPreview={() => setPreviewTpl(t)}
                      onDelete={() => handleDeletePersonal(t)}
                    />
                  ))}
                </div>
              </div>
            )
          })}
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
          <button onClick={handleSave} disabled={!selectedTplId || saving} style={{ padding: '9px 20px', borderRadius: '9px', border: `1.5px solid ${C.primary}`, background: 'transparent', color: C.primary, fontSize: '13px', fontWeight: 600, cursor: (!selectedTplId || saving) ? 'not-allowed' : 'pointer', opacity: (!selectedTplId || saving) ? 0.5 : 1 }}>
            {saving ? '保存中...' : '💾 保存风格'}
          </button>
          <button onClick={handleConfirm} disabled={!selectedTplId || confirming || saving} style={{ padding: '9px 24px', borderRadius: '9px', border: 'none', background: (!selectedTplId || confirming) ? '#D1D5DB' : `linear-gradient(135deg, ${C.primary}, #F97316)`, color: C.white, fontSize: '13px', fontWeight: 700, cursor: (!selectedTplId || confirming) ? 'not-allowed' : 'pointer', boxShadow: (!selectedTplId || confirming) ? 'none' : '0 2px 10px rgba(245,158,11,0.35)' }}>
            {confirming ? '确认中...' : '✅ 确认风格'}
          </button>
        </div>
      </div>

      {/* ====== 模板预览弹窗（自建遮罩外壳 + TemplatePagesPreview 逐页内容） ====== */}
      {previewTpl && (
        <div onClick={() => setPreviewTpl(null)} style={{ position: 'fixed', inset: 0, background: 'rgba(17,24,39,0.55)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '24px' }}>
          <div onClick={e => e.stopPropagation()} style={{ background: C.white, borderRadius: '16px', width: 'min(920px, 100%)', maxHeight: '88vh', display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.3)' }}>
            {/* 弹窗头：模板名 + 风格标签 + 关闭 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px 20px', borderBottom: `1px solid ${C.border}`, flexShrink: 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <span style={{ fontSize: '15px', fontWeight: 700, color: C.textPrimary }}>
                  {(CW_STYLE_CONFIG[previewTpl.style_category]?.emoji || '🎨')} {previewTpl.name}
                </span>
                <span style={{ padding: '1px 8px', borderRadius: '10px', fontSize: '10px', fontWeight: 600, color: (CW_STYLE_CONFIG[previewTpl.style_category]?.color || '#6B7280'), background: (CW_STYLE_CONFIG[previewTpl.style_category]?.bg || '#F3F4F6') }}>
                  {CW_STYLE_CONFIG[previewTpl.style_category]?.label || previewTpl.style_category}
                </span>
              </div>
              <button onClick={() => setPreviewTpl(null)} style={{ width: '30px', height: '30px', borderRadius: '8px', border: 'none', background: '#F3F4F6', color: C.textSecondary, fontSize: '16px', cursor: 'pointer', lineHeight: 1 }}>×</button>
            </div>
            {/* 弹窗体：逐页预览（组件按 previewUrls 有无自动走 3D 单预览 / 逐样例页渲染） */}
            <div style={{ padding: '20px', overflowY: 'auto' }}>
              <TemplatePagesPreview
                previewUrls={safeParseArray(previewTpl.preview_urls)}
                samplePages={safeParseArray(previewTpl.sample_pages)}
                accentColor={safeParse(previewTpl.color_scheme).primary || C.primary}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
